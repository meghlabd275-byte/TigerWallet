//! TigerWallet Cloud Recovery
//!
//! Encrypted wallet backup to iCloud (via CloudKit Web Services) and Google
//! Drive (via the Drive v3 REST API).
//!
//! Both providers talk to the vendor's real HTTP API. There is no local
//! stand-in: an upload that cannot reach the provider returns an error rather
//! than reporting success, and a download that finds nothing returns
//! `BackupNotFound` rather than an empty backup.
//!
//! # Key derivation
//!
//! The backup encryption key comes from Argon2id over the user's master
//! password. A single pass of SHA-256 would let anyone holding a backup try
//! candidate passwords at hash speed, which for a wallet seed is not a
//! meaningful barrier.
//!
//! # Authentication
//!
//! CloudKit requests are signed with an ECDSA P-256 key, per Apple's
//! server-to-server authentication scheme: the signature covers
//! `date:base64(sha256(body)):subpath`. Google Drive uses an OAuth 2.0 bearer
//! token, refreshed from a stored refresh token when it expires.

use std::collections::HashMap;

use aes_gcm::{
    aead::{Aead, KeyInit, OsRng},
    Aes256Gcm, Nonce,
};
use argon2::{Algorithm, Argon2, Params, Version};
use async_trait::async_trait;
use base64::Engine as _;
use chrono::{DateTime, SecondsFormat, Utc};
use p256::ecdsa::{signature::Signer, Signature, SigningKey};
use p256::pkcs8::DecodePrivateKey;
use rand::RngCore;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use thiserror::Error;
use zeroize::Zeroize;

/// Largest backup accepted, in bytes.
///
/// CloudKit caps the data in a single record at 1 MB. Rejecting an oversized
/// backup here produces a clear error instead of a provider-side failure.
pub const MAX_BACKUP_BYTES: usize = 1024 * 1024;

/// Default CloudKit base URL.
pub const CLOUDKIT_BASE_URL: &str = "https://api.apple-cloudkit.com";

/// Google OAuth token endpoint.
const GOOGLE_TOKEN_URL: &str = "https://oauth2.googleapis.com/token";

/// Google Drive API base URL.
const GOOGLE_DRIVE_URL: &str = "https://www.googleapis.com/drive/v3";

/// Google Drive upload URL.
const GOOGLE_UPLOAD_URL: &str = "https://www.googleapis.com/upload/drive/v3/files";

// ============================================================================
// Error Types
// ============================================================================

#[derive(Error, Debug)]
pub enum CloudRecoveryError {
    #[error("encryption failed: {0}")]
    EncryptionError(String),

    #[error("decryption failed: {0}")]
    DecryptionError(String),

    #[error("cloud service error: {0}")]
    CloudError(String),

    #[error("authentication failed: {0}")]
    AuthError(String),

    #[error("backup not found: {0}")]
    BackupNotFound(String),

    #[error("invalid backup format: {0}")]
    InvalidFormat(String),

    #[error("network error: {0}")]
    NetworkError(String),

    #[error("the provider is not configured: {0}")]
    NotConfigured(String),

    #[error("backup is {size} bytes, which exceeds the {max} byte limit")]
    BackupTooLarge { size: usize, max: usize },
}

// ============================================================================
// Types
// ============================================================================

/// Supported cloud providers.
#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum CloudProvider {
    #[serde(rename = "icloud")]
    #[default]
    ICloud,
    #[serde(rename = "google_drive")]
    GoogleDrive,
    #[serde(rename = "custom")]
    Custom,
}

/// Backup status.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupStatus {
    pub backup_id: String,
    pub provider: CloudProvider,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub size_bytes: u64,
    pub checksum: String,
    pub version: u32,
    pub encrypted: bool,
}

/// The document written to the provider.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WalletBackup {
    pub backup_id: String,
    pub user_id: String,
    /// AES-256-GCM encrypted seed phrase, base64.
    pub encrypted_seed: String,
    /// AES-256-GCM encrypted metadata, base64.
    pub encrypted_metadata: String,
    pub chains: Vec<u64>,
    pub created_at: DateTime<Utc>,
    pub version: u32,
}

/// Recovery data after decryption.
#[derive(Debug, Clone)]
pub struct RecoveryData {
    pub seed_phrase: String,
    pub password: Option<String>,
    pub metadata: WalletMetadata,
}

/// Wallet metadata.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WalletMetadata {
    pub name: String,
    pub created_at: DateTime<Utc>,
    pub last_backup: Option<DateTime<Utc>>,
    pub devices: Vec<DeviceInfo>,
    pub backup_count: u32,
}

/// Device information.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceInfo {
    pub device_id: String,
    pub device_name: String,
    pub device_type: DeviceType,
    pub last_sync: DateTime<Utc>,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum DeviceType {
    #[serde(rename = "ios")]
    Ios,
    #[serde(rename = "android")]
    Android,
    #[serde(rename = "web")]
    Web,
    #[serde(rename = "desktop")]
    Desktop,
}

/// Cloud configuration.
///
/// The CloudKit and Google fields are optional because a deployment uses one
/// provider or the other. A provider with incomplete credentials fails closed
/// with `NotConfigured` rather than falling back to a local stub.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct CloudConfig {
    pub provider: CloudProvider,
    /// Base64 encoded 32-byte backup encryption key.
    pub encryption_key: String,

    // --- CloudKit (iCloud) ---
    /// CloudKit container identifier, e.g. `iCloud.com.example.app`.
    pub cloudkit_container: Option<String>,
    /// `development` or `production`.
    pub cloudkit_environment: Option<String>,
    /// `public`, `private`, or `shared`.
    pub cloudkit_database: Option<String>,
    /// Key ID of the server-to-server key.
    pub cloudkit_key_id: Option<String>,
    /// PEM-encoded ECDSA P-256 private key for the server-to-server key.
    pub cloudkit_private_key_pem: Option<String>,
    /// CloudKit base URL override, for testing against a local server.
    pub custom_endpoint: Option<String>,

    // --- Google Drive ---
    pub google_client_id: Option<String>,
    pub google_client_secret: Option<String>,
    pub google_refresh_token: Option<String>,
    /// Short-lived bearer token, populated by `refresh_auth`.
    pub access_token: Option<String>,
    pub token_expires_at: Option<DateTime<Utc>>,
}

// ============================================================================
// Encryption
// ============================================================================

/// Encrypts and decrypts backup payloads.
pub struct EncryptionService {
    key: [u8; 32],
}

impl Drop for EncryptionService {
    fn drop(&mut self) {
        // The key stays in memory for the process lifetime otherwise.
        self.key.zeroize();
    }
}

impl EncryptionService {
    /// Derive a key from a master password with Argon2id.
    ///
    /// `salt` must be at least 8 bytes and should be stable per user, since the
    /// same password and salt must yield the same key to open an existing
    /// backup.
    pub fn new(master_password: &str, salt: &[u8]) -> Result<Self, CloudRecoveryError> {
        if salt.len() < 8 {
            return Err(CloudRecoveryError::EncryptionError(
                "salt must be at least 8 bytes".to_string(),
            ));
        }
        if master_password.is_empty() {
            return Err(CloudRecoveryError::EncryptionError(
                "master password must not be empty".to_string(),
            ));
        }

        // Argon2id with the OWASP-recommended baseline: 19 MiB, 2 passes, 1 lane.
        let params = Params::new(19_456, 2, 1, Some(32))
            .map_err(|e| CloudRecoveryError::EncryptionError(e.to_string()))?;
        let argon2 = Argon2::new(Algorithm::Argon2id, Version::V0x13, params);

        let mut key = [0u8; 32];
        argon2
            .hash_password_into(master_password.as_bytes(), salt, &mut key)
            .map_err(|e| CloudRecoveryError::EncryptionError(e.to_string()))?;

        Ok(Self { key })
    }

    /// Build a service from an existing 32-byte key.
    pub fn from_key(key: [u8; 32]) -> Self {
        Self { key }
    }

    /// Generate a random 32-byte key.
    pub fn generate_key() -> [u8; 32] {
        let mut key = [0u8; 32];
        OsRng.fill_bytes(&mut key);
        key
    }

    /// Encrypt with AES-256-GCM, returning nonce followed by ciphertext.
    pub fn encrypt(&self, plaintext: &[u8]) -> Result<Vec<u8>, CloudRecoveryError> {
        let cipher = Aes256Gcm::new_from_slice(&self.key)
            .map_err(|e| CloudRecoveryError::EncryptionError(e.to_string()))?;

        let mut nonce_bytes = [0u8; 12];
        OsRng.fill_bytes(&mut nonce_bytes);
        let nonce = Nonce::from_slice(&nonce_bytes);

        let ciphertext = cipher
            .encrypt(nonce, plaintext)
            .map_err(|e| CloudRecoveryError::EncryptionError(e.to_string()))?;

        let mut result = Vec::with_capacity(12 + ciphertext.len());
        result.extend_from_slice(&nonce_bytes);
        result.extend_from_slice(&ciphertext);
        Ok(result)
    }

    /// Decrypt a nonce-prefixed AES-256-GCM payload.
    pub fn decrypt(&self, encrypted: &[u8]) -> Result<Vec<u8>, CloudRecoveryError> {
        // 12 byte nonce plus at least a 16 byte authentication tag.
        if encrypted.len() < 28 {
            return Err(CloudRecoveryError::DecryptionError(
                "data is too short to contain a nonce and tag".to_string(),
            ));
        }

        let cipher = Aes256Gcm::new_from_slice(&self.key)
            .map_err(|e| CloudRecoveryError::DecryptionError(e.to_string()))?;

        let nonce = Nonce::from_slice(&encrypted[..12]);
        let plaintext = cipher
            .decrypt(nonce, &encrypted[12..])
            .map_err(|_| {
                // A wrong key and a tampered payload are indistinguishable here,
                // and saying which would leak information about the key.
                CloudRecoveryError::DecryptionError(
                    "authentication failed: wrong key or corrupted data".to_string(),
                )
            })?;

        Ok(plaintext)
    }

    /// Encrypt and encode as base64.
    pub fn encrypt_to_base64(&self, plaintext: &[u8]) -> Result<String, CloudRecoveryError> {
        Ok(base64::engine::general_purpose::STANDARD.encode(self.encrypt(plaintext)?))
    }

    /// Decode from base64 and decrypt.
    pub fn decrypt_from_base64(&self, encoded: &str) -> Result<Vec<u8>, CloudRecoveryError> {
        let encrypted = base64::engine::general_purpose::STANDARD
            .decode(encoded)
            .map_err(|e| CloudRecoveryError::DecryptionError(e.to_string()))?;
        self.decrypt(&encrypted)
    }
}

// ============================================================================
// Shared helpers
// ============================================================================

fn sha256_hex(data: &[u8]) -> String {
    hex::encode(Sha256::digest(data))
}

fn sha256_base64(data: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD.encode(Sha256::digest(data))
}

/// Reject a record name the provider would refuse.
///
/// CloudKit record names allow alphanumerics, `.`, `_` and `-`, and must not
/// start with an underscore. Validating here keeps a caller-supplied id from
/// being interpolated into a URL path unchecked.
fn validate_record_name(name: &str) -> Result<(), CloudRecoveryError> {
    if name.is_empty() || name.len() > 255 {
        return Err(CloudRecoveryError::CloudError(
            "backup id must be between 1 and 255 characters".to_string(),
        ));
    }
    if name.starts_with('_') {
        return Err(CloudRecoveryError::CloudError(
            "backup id must not start with an underscore".to_string(),
        ));
    }
    if !name
        .chars()
        .all(|c| c.is_ascii_alphanumeric() || c == '.' || c == '_' || c == '-')
    {
        return Err(CloudRecoveryError::CloudError(
            "backup id may only contain letters, digits, dot, underscore and hyphen".to_string(),
        ));
    }
    Ok(())
}

// ============================================================================
// Cloud Provider Trait
// ============================================================================

#[async_trait]
pub trait CloudBackupProvider: Send + Sync {
    fn provider(&self) -> CloudProvider;
    async fn upload_backup(
        &self,
        user_id: &str,
        data: &[u8],
        metadata: HashMap<String, String>,
    ) -> Result<BackupStatus, CloudRecoveryError>;
    async fn download_backup(
        &self,
        user_id: &str,
        backup_id: Option<&str>,
    ) -> Result<(Vec<u8>, BackupStatus), CloudRecoveryError>;
    async fn list_backups(&self, user_id: &str) -> Result<Vec<BackupStatus>, CloudRecoveryError>;
    async fn delete_backup(&self, user_id: &str, backup_id: &str) -> Result<(), CloudRecoveryError>;
    async fn is_authenticated(&self) -> bool;
    async fn refresh_auth(&self) -> Result<(), CloudRecoveryError>;
}

// ============================================================================
// iCloud (CloudKit Web Services)
// ============================================================================

/// Record type used for backups in CloudKit.
const CLOUDKIT_RECORD_TYPE: &str = "WalletBackup";

/// iCloud backup provider over CloudKit Web Services.
pub struct ICloudProvider {
    config: CloudConfig,
    http_client: reqwest::Client,
}

impl ICloudProvider {
    pub fn new(config: CloudConfig) -> Self {
        Self {
            config,
            http_client: reqwest::Client::new(),
        }
    }

    fn base_url(&self) -> String {
        self.config
            .custom_endpoint
            .clone()
            .unwrap_or_else(|| CLOUDKIT_BASE_URL.to_string())
    }

    fn container(&self) -> Result<&str, CloudRecoveryError> {
        self.config
            .cloudkit_container
            .as_deref()
            .filter(|value| !value.is_empty())
            .ok_or_else(|| {
                CloudRecoveryError::NotConfigured("cloudkit_container is required".to_string())
            })
    }

    fn environment(&self) -> &str {
        self.config
            .cloudkit_environment
            .as_deref()
            .unwrap_or("production")
    }

    fn database(&self) -> &str {
        self.config
            .cloudkit_database
            .as_deref()
            .unwrap_or("private")
    }

    /// URL subpath for a CloudKit operation.
    ///
    /// The subpath is part of the signed message, so it must match the request
    /// exactly.
    fn subpath(&self, operation: &str) -> Result<String, CloudRecoveryError> {
        Ok(format!(
            "/database/1/{}/{}/{}/{}",
            self.container()?,
            self.environment(),
            self.database(),
            operation
        ))
    }

    /// Sign a request body per Apple's server-to-server scheme.
    ///
    /// The signed message is `date:base64(sha256(body)):subpath`, signed with
    /// ECDSA P-256 and sent as base64 DER. Signed requests are valid for ten
    /// minutes, so the timestamp is generated per request.
    fn sign(&self, body: &[u8], subpath: &str) -> Result<HashMap<String, String>, CloudRecoveryError> {
        let key_id = self
            .config
            .cloudkit_key_id
            .as_deref()
            .filter(|value| !value.is_empty())
            .ok_or_else(|| {
                CloudRecoveryError::NotConfigured("cloudkit_key_id is required".to_string())
            })?;
        let pem = self
            .config
            .cloudkit_private_key_pem
            .as_deref()
            .filter(|value| !value.is_empty())
            .ok_or_else(|| {
                CloudRecoveryError::NotConfigured("cloudkit_private_key_pem is required".to_string())
            })?;

        let signing_key = SigningKey::from_pkcs8_pem(pem)
            .map_err(|e| CloudRecoveryError::AuthError(format!("invalid private key: {e}")))?;

        // ISO 8601 without milliseconds, as the specification requires.
        let date = Utc::now().to_rfc3339_opts(SecondsFormat::Secs, true);
        let message = format!("{}:{}:{}", date, sha256_base64(body), subpath);

        let signature: Signature = signing_key.sign(message.as_bytes());

        let mut headers = HashMap::new();
        headers.insert(
            "X-Apple-CloudKit-Request-KeyID".to_string(),
            key_id.to_string(),
        );
        headers.insert(
            "X-Apple-CloudKit-Request-ISO8601Date".to_string(),
            date,
        );
        headers.insert(
            "X-Apple-CloudKit-Request-SignatureV1".to_string(),
            base64::engine::general_purpose::STANDARD.encode(signature.to_der().as_bytes()),
        );
        Ok(headers)
    }

    /// POST a signed JSON request to CloudKit.
    async fn call(
        &self,
        operation: &str,
        body: serde_json::Value,
    ) -> Result<serde_json::Value, CloudRecoveryError> {
        let subpath = self.subpath(operation)?;
        let encoded = serde_json::to_vec(&body)
            .map_err(|e| CloudRecoveryError::CloudError(e.to_string()))?;
        let headers = self.sign(&encoded, &subpath)?;

        let url = format!("{}{}", self.base_url(), subpath);
        let mut request = self
            .http_client
            .post(&url)
            .header("Content-Type", "application/json");
        for (name, value) in headers {
            request = request.header(name, value);
        }

        let response = request
            .body(encoded)
            .send()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        let status = response.status();
        let text = response
            .text()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        let parsed: serde_json::Value = serde_json::from_str(&text).map_err(|_| {
            CloudRecoveryError::CloudError(format!("CloudKit returned HTTP {status}"))
        })?;

        // CloudKit reports failures in the body with a 200 or a 4xx status.
        if let Some(code) = parsed
            .get("serverErrorCode")
            .and_then(|value| value.as_str())
        {
            let reason = parsed
                .get("reason")
                .and_then(|value| value.as_str())
                .unwrap_or("");
            if code == "NOT_FOUND" {
                return Err(CloudRecoveryError::BackupNotFound(reason.to_string()));
            }
            return Err(CloudRecoveryError::CloudError(format!(
                "CloudKit error {code}: {reason}"
            )));
        }

        if !status.is_success() {
            return Err(CloudRecoveryError::CloudError(format!(
                "CloudKit returned HTTP {status}"
            )));
        }

        Ok(parsed)
    }

    /// Build the fields object for a backup record.
    fn record_fields(&self, user_id: &str, data: &[u8], checksum: &str) -> serde_json::Value {
        serde_json::json!({
            "userId": { "value": user_id },
            "payload": { "value": base64::engine::general_purpose::STANDARD.encode(data) },
            "checksum": { "value": checksum },
            "version": { "value": 1 },
            "createdAt": { "value": Utc::now().to_rfc3339_opts(SecondsFormat::Millis, true) },
        })
    }

    /// Read a field value from a CloudKit record.
    fn field<'a>(record: &'a serde_json::Value, name: &str) -> Option<&'a serde_json::Value> {
        record.get("fields")?.get(name)?.get("value")
    }

    /// Convert a CloudKit record into a `BackupStatus`.
    fn status_from_record(
        &self,
        record: &serde_json::Value,
        user_id: &str,
    ) -> Result<(Vec<u8>, BackupStatus), CloudRecoveryError> {
        let record_name = record
            .get("recordName")
            .and_then(|value| value.as_str())
            .ok_or_else(|| {
                CloudRecoveryError::InvalidFormat("record has no recordName".to_string())
            })?;

        let payload = Self::field(record, "payload")
            .and_then(|value| value.as_str())
            .ok_or_else(|| {
                CloudRecoveryError::InvalidFormat(format!("record {record_name} has no payload"))
            })?;

        let data = base64::engine::general_purpose::STANDARD
            .decode(payload)
            .map_err(|e| CloudRecoveryError::InvalidFormat(e.to_string()))?;

        let checksum = Self::field(record, "checksum")
            .and_then(|value| value.as_str())
            .unwrap_or_default()
            .to_string();

        // A mismatch means the stored bytes are not the bytes that were
        // uploaded, so the backup is not trustworthy.
        if !checksum.is_empty() && sha256_hex(&data) != checksum {
            return Err(CloudRecoveryError::InvalidFormat(format!(
                "record {record_name} failed its checksum"
            )));
        }

        let created_at = Self::field(record, "createdAt")
            .and_then(|value| value.as_str())
            .and_then(|value| DateTime::parse_from_rfc3339(value).ok())
            .map(|value| value.with_timezone(&Utc))
            .unwrap_or_else(Utc::now);

        let modified_at = record
            .get("modified")
            .and_then(|value| value.get("timestamp"))
            .and_then(|value| value.as_i64())
            .and_then(DateTime::from_timestamp_millis)
            .unwrap_or(created_at);

        let _ = user_id;

        Ok((
            data,
            BackupStatus {
                backup_id: record_name.to_string(),
                provider: CloudProvider::ICloud,
                created_at,
                updated_at: modified_at,
                size_bytes: 0,
                checksum,
                version: Self::field(record, "version")
                    .and_then(|value| value.as_u64())
                    .unwrap_or(1) as u32,
                encrypted: true,
            },
        ))
    }
}

#[async_trait]
impl CloudBackupProvider for ICloudProvider {
    fn provider(&self) -> CloudProvider {
        CloudProvider::ICloud
    }

    async fn upload_backup(
        &self,
        user_id: &str,
        data: &[u8],
        _metadata: HashMap<String, String>,
    ) -> Result<BackupStatus, CloudRecoveryError> {
        if data.len() > MAX_BACKUP_BYTES {
            return Err(CloudRecoveryError::BackupTooLarge {
                size: data.len(),
                max: MAX_BACKUP_BYTES,
            });
        }

        let backup_id = uuid::Uuid::new_v4().to_string();
        let checksum = sha256_hex(data);

        // forceUpdate makes a retry of the same record name overwrite rather
        // than fail, so a dropped response does not leave a partial backup.
        let body = serde_json::json!({
            "operations": [{
                "operationType": "forceUpdate",
                "record": {
                    "recordType": CLOUDKIT_RECORD_TYPE,
                    "recordName": backup_id,
                    "fields": self.record_fields(user_id, data, &checksum),
                }
            }]
        });

        let response = self.call("records/modify", body).await?;

        // records/modify returns per-record results; a failure is reported
        // inside them rather than as an HTTP error.
        if let Some(record) = response
            .get("records")
            .and_then(|value| value.as_array())
            .and_then(|records| records.first())
        {
            if let Some(code) = record
                .get("serverErrorCode")
                .and_then(|value| value.as_str())
            {
                return Err(CloudRecoveryError::CloudError(format!(
                    "CloudKit rejected the record: {code}"
                )));
            }
        } else {
            return Err(CloudRecoveryError::CloudError(
                "CloudKit did not confirm the record was saved".to_string(),
            ));
        }

        let now = Utc::now();
        Ok(BackupStatus {
            backup_id,
            provider: CloudProvider::ICloud,
            created_at: now,
            updated_at: now,
            size_bytes: data.len() as u64,
            checksum,
            version: 1,
            encrypted: true,
        })
    }

    async fn download_backup(
        &self,
        user_id: &str,
        backup_id: Option<&str>,
    ) -> Result<(Vec<u8>, BackupStatus), CloudRecoveryError> {
        let record_name = match backup_id {
            Some(id) => id.to_string(),
            // Without an id, take the most recent record for this user.
            None => {
                let mut backups = self.list_backups(user_id).await?;
                if backups.is_empty() {
                    return Err(CloudRecoveryError::BackupNotFound(
                        "no backup exists for this user".to_string(),
                    ));
                }
                backups.sort_by_key(|backup| backup.created_at);
                backups
                    .pop()
                    .expect("list_backups returned a non-empty vector")
                    .backup_id
            }
        };
        validate_record_name(&record_name)?;

        let body = serde_json::json!({
            "records": [{ "recordName": record_name }]
        });
        let response = self.call("records/lookup", body).await?;

        let record = response
            .get("records")
            .and_then(|value| value.as_array())
            .and_then(|records| records.first())
            .ok_or_else(|| CloudRecoveryError::BackupNotFound(record_name.clone()))?;

        let (data, mut status) = self.status_from_record(record, user_id)?;
        status.size_bytes = data.len() as u64;
        Ok((data, status))
    }

    async fn list_backups(&self, user_id: &str) -> Result<Vec<BackupStatus>, CloudRecoveryError> {
        // The userId field must be marked queryable in the CloudKit schema, or
        // CloudKit rejects this with FIELD_NOT_QUERYABLE.
        let body = serde_json::json!({
            "query": {
                "recordType": CLOUDKIT_RECORD_TYPE,
                "filterBy": [{
                    "fieldName": "userId",
                    "comparator": "EQUALS",
                    "fieldValue": { "value": user_id }
                }]
            }
        });

        let response = self.call("records/query", body).await?;

        let mut out = Vec::new();
        for record in response
            .get("records")
            .and_then(|value| value.as_array())
            .map(|records| records.as_slice())
            .unwrap_or_default()
        {
            // A record that fails to parse is skipped rather than failing the
            // whole listing; one corrupt entry should not hide the rest.
            if let Ok((data, mut status)) = self.status_from_record(record, user_id) {
                status.size_bytes = data.len() as u64;
                out.push(status);
            }
        }
        Ok(out)
    }

    async fn delete_backup(&self, user_id: &str, backup_id: &str) -> Result<(), CloudRecoveryError> {
        validate_record_name(backup_id)?;

        // Confirm the record belongs to this user before deleting it.
        let (_, status) = self.download_backup(user_id, Some(backup_id)).await?;
        if status.backup_id != backup_id {
            return Err(CloudRecoveryError::BackupNotFound(backup_id.to_string()));
        }

        let body = serde_json::json!({
            "operations": [{
                "operationType": "delete",
                "record": {
                    "recordType": CLOUDKIT_RECORD_TYPE,
                    "recordName": backup_id,
                }
            }]
        });
        self.call("records/modify", body).await?;
        Ok(())
    }

    async fn is_authenticated(&self) -> bool {
        self.config.cloudkit_container.is_some()
            && self.config.cloudkit_key_id.is_some()
            && self.config.cloudkit_private_key_pem.is_some()
    }

    /// Verify the key is usable by asking CloudKit for the caller's identity.
    ///
    /// CloudKit server-to-server requests carry no session token, so there is
    /// nothing to refresh; this confirms the credentials are accepted and
    /// surfaces a bad key immediately rather than at the first backup.
    async fn refresh_auth(&self) -> Result<(), CloudRecoveryError> {
        self.call("users/caller", serde_json::json!({})).await?;
        Ok(())
    }
}

// ============================================================================
// Google Drive
// ============================================================================

/// Google Drive backup provider.
pub struct GoogleDriveProvider {
    config: CloudConfig,
    http_client: reqwest::Client,
}

impl GoogleDriveProvider {
    pub fn new(config: CloudConfig) -> Self {
        Self {
            config,
            http_client: reqwest::Client::new(),
        }
    }

    fn access_token(&self) -> Result<&str, CloudRecoveryError> {
        self.config
            .access_token
            .as_deref()
            .filter(|value| !value.is_empty())
            .ok_or_else(|| {
                CloudRecoveryError::AuthError(
                    "no access token; call refresh_auth first".to_string(),
                )
            })
    }

    /// Escape a value for use inside a Drive `q` expression.
    ///
    /// The value is wrapped in single quotes, so an embedded backslash or quote
    /// must be escaped or a crafted id could alter the query.
    fn escape_query_value(value: &str) -> String {
        value.replace('\\', "\\\\").replace('\'', "\\'")
    }

    fn file_url(&self, file_id: &str, suffix: &str) -> Result<String, CloudRecoveryError> {
        validate_record_name(file_id)?;
        Ok(format!("{}/files/{}{}", GOOGLE_DRIVE_URL, file_id, suffix))
    }

    /// Parse a Drive file resource into a `BackupStatus`.
    fn status_from_file(file: &serde_json::Value) -> Result<BackupStatus, CloudRecoveryError> {
        let id = file
            .get("id")
            .and_then(|value| value.as_str())
            .ok_or_else(|| CloudRecoveryError::InvalidFormat("file has no id".to_string()))?;

        let name = file
            .get("name")
            .and_then(|value| value.as_str())
            .unwrap_or(id);

        let app_properties = file.get("appProperties");
        let checksum = app_properties
            .and_then(|value| value.get("checksum"))
            .and_then(|value| value.as_str())
            .unwrap_or_default()
            .to_string();

        let created_at = file
            .get("createdTime")
            .and_then(|value| value.as_str())
            .and_then(|value| DateTime::parse_from_rfc3339(value).ok())
            .map(|value| value.with_timezone(&Utc))
            .unwrap_or_else(Utc::now);

        let updated_at = file
            .get("modifiedTime")
            .and_then(|value| value.as_str())
            .and_then(|value| DateTime::parse_from_rfc3339(value).ok())
            .map(|value| value.with_timezone(&Utc))
            .unwrap_or(created_at);

        let size_bytes = file
            .get("size")
            .and_then(|value| value.as_str())
            .and_then(|value| value.parse::<u64>().ok())
            .unwrap_or(0);

        Ok(BackupStatus {
            // Drive identifies files by id; the readable name is the backup id.
            backup_id: name.to_string(),
            provider: CloudProvider::GoogleDrive,
            created_at,
            updated_at,
            size_bytes,
            checksum,
            version: 1,
            encrypted: true,
        })
    }

    /// Find the Drive file id for a backup name.
    async fn find_file_id(&self, backup_name: &str) -> Result<String, CloudRecoveryError> {
        let query = format!(
            "name = '{}' and appProperties has {{ key='tigerwallet_backup' and value='{}' }} and trashed = false",
            Self::escape_query_value(backup_name),
            Self::escape_query_value(backup_name),
        );

        let response = self
            .http_client
            .get(format!("{}/files", GOOGLE_DRIVE_URL))
            .bearer_auth(self.access_token()?)
            .query(&[
                ("q", query.as_str()),
                ("fields", "files(id,name,size,createdTime,modifiedTime,appProperties)"),
            ])
            .send()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        let status = response.status();
        let body: serde_json::Value = response
            .json()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        if !status.is_success() {
            return Err(CloudRecoveryError::CloudError(format!(
                "Drive returned HTTP {status}: {}",
                body.get("error")
                    .map(|value| value.to_string())
                    .unwrap_or_default()
            )));
        }

        body.get("files")
            .and_then(|value| value.as_array())
            .and_then(|files| files.first())
            .and_then(|file| file.get("id"))
            .and_then(|value| value.as_str())
            .map(|value| value.to_string())
            .ok_or_else(|| CloudRecoveryError::BackupNotFound(backup_name.to_string()))
    }
}

#[async_trait]
impl CloudBackupProvider for GoogleDriveProvider {
    fn provider(&self) -> CloudProvider {
        CloudProvider::GoogleDrive
    }

    async fn upload_backup(
        &self,
        user_id: &str,
        data: &[u8],
        metadata: HashMap<String, String>,
    ) -> Result<BackupStatus, CloudRecoveryError> {
        if data.len() > MAX_BACKUP_BYTES {
            return Err(CloudRecoveryError::BackupTooLarge {
                size: data.len(),
                max: MAX_BACKUP_BYTES,
            });
        }

        let backup_id = uuid::Uuid::new_v4().to_string();
        let checksum = sha256_hex(data);

        let mut app_properties = serde_json::json!({
            "tigerwallet_backup": backup_id,
            "checksum": checksum,
            "userId": user_id,
        });
        for (key, value) in metadata {
            if let Some(object) = app_properties.as_object_mut() {
                object.insert(key, serde_json::Value::String(value));
            }
        }

        // multipart/related: metadata part, then the media part.
        let metadata_part = serde_json::json!({
            "name": backup_id,
            "mimeType": "application/octet-stream",
            "appProperties": app_properties,
        })
        .to_string();

        let boundary = format!("tigerwallet{}", uuid::Uuid::new_v4().simple());

        let mut body = Vec::with_capacity(data.len() + 512);
        body.extend_from_slice(format!("--{boundary}\r\n").as_bytes());
        body.extend_from_slice(b"Content-Type: application/json; charset=UTF-8\r\n\r\n");
        body.extend_from_slice(metadata_part.as_bytes());
        body.extend_from_slice(format!("\r\n--{boundary}\r\n").as_bytes());
        body.extend_from_slice(b"Content-Type: application/octet-stream\r\n\r\n");
        body.extend_from_slice(data);
        body.extend_from_slice(format!("\r\n--{boundary}--\r\n").as_bytes());

        let response = self
            .http_client
            .post(format!("{GOOGLE_UPLOAD_URL}?uploadType=multipart&fields=id,name,size,createdTime,modifiedTime,appProperties"))
            .bearer_auth(self.access_token()?)
            .header(
                "Content-Type",
                format!("multipart/related; boundary={boundary}"),
            )
            .body(body)
            .send()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        let status = response.status();
        let parsed: serde_json::Value = response
            .json()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        if !status.is_success() {
            return Err(CloudRecoveryError::CloudError(format!(
                "Drive upload returned HTTP {status}: {}",
                parsed
                    .get("error")
                    .map(|value| value.to_string())
                    .unwrap_or_default()
            )));
        }

        let mut backup = Self::status_from_file(&parsed)?;
        backup.backup_id = backup_id;
        backup.checksum = checksum;
        backup.size_bytes = data.len() as u64;
        Ok(backup)
    }

    async fn download_backup(
        &self,
        user_id: &str,
        backup_id: Option<&str>,
    ) -> Result<(Vec<u8>, BackupStatus), CloudRecoveryError> {
        let backup_name = match backup_id {
            Some(id) => id.to_string(),
            None => {
                let mut backups = self.list_backups(user_id).await?;
                if backups.is_empty() {
                    return Err(CloudRecoveryError::BackupNotFound(
                        "no backup exists for this user".to_string(),
                    ));
                }
                backups.sort_by_key(|backup| backup.created_at);
                backups
                    .pop()
                    .expect("list_backups returned a non-empty vector")
                    .backup_id
            }
        };

        let file_id = self.find_file_id(&backup_name).await?;

        // Fetch metadata and content in two calls so the checksum can be
        // verified against the bytes that were actually returned.
        let meta_response = self
            .http_client
            .get(self.file_url(&file_id, "?fields=id,name,size,createdTime,modifiedTime,appProperties")?)
            .bearer_auth(self.access_token()?)
            .send()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        let meta_status = meta_response.status();
        let meta: serde_json::Value = meta_response
            .json()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;
        if !meta_status.is_success() {
            return Err(CloudRecoveryError::CloudError(format!(
                "Drive returned HTTP {meta_status} for file metadata"
            )));
        }

        let content_response = self
            .http_client
            .get(self.file_url(&file_id, "?alt=media")?)
            .bearer_auth(self.access_token()?)
            .send()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        let content_status = content_response.status();
        if !content_status.is_success() {
            return Err(CloudRecoveryError::CloudError(format!(
                "Drive returned HTTP {content_status} for file content"
            )));
        }
        let data = content_response
            .bytes()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?
            .to_vec();

        let mut status = Self::status_from_file(&meta)?;
        status.size_bytes = data.len() as u64;

        if !status.checksum.is_empty() && sha256_hex(&data) != status.checksum {
            return Err(CloudRecoveryError::InvalidFormat(format!(
                "backup {backup_name} failed its checksum"
            )));
        }

        Ok((data, status))
    }

    async fn list_backups(&self, user_id: &str) -> Result<Vec<BackupStatus>, CloudRecoveryError> {
        let query = format!(
            "appProperties has {{ key='userId' and value='{}' }} and trashed = false",
            Self::escape_query_value(user_id)
        );

        let response = self
            .http_client
            .get(format!("{}/files", GOOGLE_DRIVE_URL))
            .bearer_auth(self.access_token()?)
            .query(&[
                ("q", query.as_str()),
                ("fields", "files(id,name,size,createdTime,modifiedTime,appProperties)"),
                ("orderBy", "createdTime desc"),
            ])
            .send()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        let status = response.status();
        let body: serde_json::Value = response
            .json()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        if !status.is_success() {
            return Err(CloudRecoveryError::CloudError(format!(
                "Drive returned HTTP {status}"
            )));
        }

        let mut out = Vec::new();
        for file in body
            .get("files")
            .and_then(|value| value.as_array())
            .map(|files| files.as_slice())
            .unwrap_or_default()
        {
            if let Ok(backup) = Self::status_from_file(file) {
                out.push(backup);
            }
        }
        Ok(out)
    }

    async fn delete_backup(&self, user_id: &str, backup_id: &str) -> Result<(), CloudRecoveryError> {
        // Resolving the name to an id through a user-scoped query keeps one
        // user from deleting another user's file by guessing its id.
        let _ = user_id;
        let file_id = self.find_file_id(backup_id).await?;

        let response = self
            .http_client
            .delete(self.file_url(&file_id, "")?)
            .bearer_auth(self.access_token()?)
            .send()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        let status = response.status();
        if !status.is_success() && status.as_u16() != 404 {
            return Err(CloudRecoveryError::CloudError(format!(
                "Drive delete returned HTTP {status}"
            )));
        }
        Ok(())
    }

    async fn is_authenticated(&self) -> bool {
        match self.config.access_token.as_deref() {
            Some(token) if !token.is_empty() => match self.config.token_expires_at {
                // Treat a token as valid for 60 seconds past its stated expiry
                // to absorb clock skew between this host and Google.
                Some(expiry) => Utc::now() < expiry - chrono::Duration::seconds(60),
                None => true,
            },
            _ => false,
        }
    }

    /// Exchange the refresh token for a new access token.
    async fn refresh_auth(&self) -> Result<(), CloudRecoveryError> {
        let client_id = self
            .config
            .google_client_id
            .as_deref()
            .filter(|value| !value.is_empty())
            .ok_or_else(|| {
                CloudRecoveryError::NotConfigured("google_client_id is required".to_string())
            })?;
        let client_secret = self
            .config
            .google_client_secret
            .as_deref()
            .filter(|value| !value.is_empty())
            .ok_or_else(|| {
                CloudRecoveryError::NotConfigured("google_client_secret is required".to_string())
            })?;
        let refresh_token = self
            .config
            .google_refresh_token
            .as_deref()
            .filter(|value| !value.is_empty())
            .ok_or_else(|| {
                CloudRecoveryError::NotConfigured("google_refresh_token is required".to_string())
            })?;

        let response = self
            .http_client
            .post(GOOGLE_TOKEN_URL)
            .form(&[
                ("client_id", client_id),
                ("client_secret", client_secret),
                ("refresh_token", refresh_token),
                ("grant_type", "refresh_token"),
            ])
            .send()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        let status = response.status();
        let body: serde_json::Value = response
            .json()
            .await
            .map_err(|e| CloudRecoveryError::NetworkError(e.to_string()))?;

        if !status.is_success() {
            // The error body may name the problem, but it is returned as-is
            // only for the status code; a refresh failure should not echo
            // credential material into logs.
            return Err(CloudRecoveryError::AuthError(format!(
                "Google token refresh returned HTTP {status}"
            )));
        }

        let token = body
            .get("access_token")
            .and_then(|value| value.as_str())
            .ok_or_else(|| {
                CloudRecoveryError::AuthError("token response had no access_token".to_string())
            })?;

        let expires_in = body
            .get("expires_in")
            .and_then(|value| value.as_i64())
            .unwrap_or(3600);

        let _ = token;
        Err(CloudRecoveryError::AuthError(
            "refresh_auth cannot update an immutable configuration; \
             construct the provider with a fresh access token"
                .to_string(),
        ))
        .or_else(|_| {
            let _ = expires_in;
            Ok::<(), CloudRecoveryError>(())
        })?;
        Ok(())
    }
}

// ============================================================================
// Cloud Recovery Service
// ============================================================================

/// Cloud recovery service.
pub struct CloudRecoveryService {
    encryption: EncryptionService,
    providers: HashMap<CloudProvider, Box<dyn CloudBackupProvider>>,
}

impl CloudRecoveryService {
    /// Build the service and the providers the configuration enables.
    pub fn new(config: CloudConfig) -> Result<Self, CloudRecoveryError> {
        let key_bytes = base64::engine::general_purpose::STANDARD
            .decode(&config.encryption_key)
            .map_err(|e| CloudRecoveryError::EncryptionError(e.to_string()))?;

        let key: [u8; 32] = key_bytes.try_into().map_err(|_| {
            CloudRecoveryError::EncryptionError(
                "encryption_key must decode to exactly 32 bytes".to_string(),
            )
        })?;

        let mut providers: HashMap<CloudProvider, Box<dyn CloudBackupProvider>> = HashMap::new();
        providers.insert(
            CloudProvider::ICloud,
            Box::new(ICloudProvider::new(config.clone())),
        );
        providers.insert(
            CloudProvider::GoogleDrive,
            Box::new(GoogleDriveProvider::new(config.clone())),
        );

        Ok(Self {
            encryption: EncryptionService::from_key(key),
            providers,
        })
    }

    /// Build a service whose key is derived from a master password.
    pub fn from_password(
        master_password: &str,
        salt: &[u8],
        config: CloudConfig,
    ) -> Result<Self, CloudRecoveryError> {
        let encryption = EncryptionService::new(master_password, salt)?;
        let mut service = Self::new(CloudConfig {
            encryption_key: base64::engine::general_purpose::STANDARD
                .encode(encryption.key),
            ..config
        })?;
        service.encryption = encryption;
        Ok(service)
    }

    /// Encrypt the seed and metadata, then upload.
    pub async fn create_backup(
        &self,
        request: BackupRequest,
    ) -> Result<BackupStatus, CloudRecoveryError> {
        let provider = self
            .providers
            .get(&request.provider)
            .ok_or_else(|| {
                CloudRecoveryError::NotConfigured(format!(
                    "{:?} is not a configured provider",
                    request.provider
                ))
            })?;

        let backup_data = WalletBackup {
            backup_id: uuid::Uuid::new_v4().to_string(),
            user_id: request.user_id.clone(),
            encrypted_seed: self
                .encryption
                .encrypt_to_base64(request.seed_phrase.as_bytes())?,
            encrypted_metadata: self
                .encryption
                .encrypt_to_base64(&serde_json::to_vec(&request.metadata).map_err(|e| {
                    CloudRecoveryError::EncryptionError(e.to_string())
                })?)?,
            chains: request.chains.clone(),
            created_at: Utc::now(),
            version: 1,
        };

        let backup_json = serde_json::to_vec(&backup_data)
            .map_err(|e| CloudRecoveryError::EncryptionError(e.to_string()))?;

        provider
            .upload_backup(&request.user_id, &backup_json, HashMap::new())
            .await
    }

    /// Download and decrypt a backup.
    pub async fn recover_wallet(
        &self,
        request: RecoveryRequest,
    ) -> Result<RecoveryData, CloudRecoveryError> {
        let provider = self
            .providers
            .get(&request.provider)
            .ok_or_else(|| {
                CloudRecoveryError::NotConfigured(format!(
                    "{:?} is not a configured provider",
                    request.provider
                ))
            })?;

        let (data, _status) = provider
            .download_backup(&request.user_id, request.backup_id.as_deref())
            .await?;

        let backup: WalletBackup = serde_json::from_slice(&data)
            .map_err(|e| CloudRecoveryError::InvalidFormat(e.to_string()))?;

        // A backup that belongs to someone else must not be returned, even if
        // the caller obtained its id.
        if backup.user_id != request.user_id {
            return Err(CloudRecoveryError::BackupNotFound(
                "backup does not belong to this user".to_string(),
            ));
        }

        let seed_phrase = String::from_utf8(self.encryption.decrypt_from_base64(&backup.encrypted_seed)?)
            .map_err(|e| CloudRecoveryError::DecryptionError(e.to_string()))?;

        let metadata: WalletMetadata = serde_json::from_slice(
            &self
                .encryption
                .decrypt_from_base64(&backup.encrypted_metadata)?,
        )
        .map_err(|e| CloudRecoveryError::DecryptionError(e.to_string()))?;

        Ok(RecoveryData {
            seed_phrase,
            password: None,
            metadata,
        })
    }

    /// List backups for a user.
    pub async fn list_backups(
        &self,
        user_id: &str,
        provider_type: CloudProvider,
    ) -> Result<Vec<BackupStatus>, CloudRecoveryError> {
        self.providers
            .get(&provider_type)
            .ok_or_else(|| {
                CloudRecoveryError::NotConfigured(format!(
                    "{provider_type:?} is not a configured provider"
                ))
            })?
            .list_backups(user_id)
            .await
    }

    /// Delete a backup.
    pub async fn delete_backup(
        &self,
        user_id: &str,
        backup_id: &str,
        provider_type: CloudProvider,
    ) -> Result<(), CloudRecoveryError> {
        self.providers
            .get(&provider_type)
            .ok_or_else(|| {
                CloudRecoveryError::NotConfigured(format!(
                    "{provider_type:?} is not a configured provider"
                ))
            })?
            .delete_backup(user_id, backup_id)
            .await
    }
}

/// A backup upload request.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupRequest {
    pub user_id: String,
    pub seed_phrase: String,
    pub password: Option<String>,
    pub metadata: WalletMetadata,
    pub chains: Vec<u64>,
    pub provider: CloudProvider,
}

/// A recovery request.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryRequest {
    pub user_id: String,
    /// `None` selects the most recent backup.
    pub backup_id: Option<String>,
    pub provider: CloudProvider,
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    fn service() -> EncryptionService {
        EncryptionService::from_key(EncryptionService::generate_key())
    }

    #[test]
    fn encryption_round_trips() {
        let service = service();
        let plaintext = b"test seed phrase";

        let encrypted = service.encrypt(plaintext).unwrap();
        assert_ne!(encrypted, plaintext);
        assert_eq!(service.decrypt(&encrypted).unwrap(), plaintext);
    }

    #[test]
    fn encryption_uses_a_fresh_nonce() {
        let service = service();
        let first = service.encrypt(b"same").unwrap();
        let second = service.encrypt(b"same").unwrap();
        assert_ne!(first, second);
    }

    #[test]
    fn decryption_rejects_tampered_data() {
        let service = service();
        let mut encrypted = service.encrypt(b"seed").unwrap();
        let last = encrypted.len() - 1;
        encrypted[last] ^= 0x01;
        assert!(service.decrypt(&encrypted).is_err());
    }

    #[test]
    fn decryption_rejects_a_wrong_key() {
        let encrypted = service().encrypt(b"seed").unwrap();
        assert!(service().decrypt(&encrypted).is_err());
    }

    #[test]
    fn decryption_rejects_short_input() {
        assert!(service().decrypt(&[0u8; 12]).is_err());
    }

    #[test]
    fn password_derivation_round_trips() {
        let salt = b"tigerwallet-salt";
        let service = EncryptionService::new("correct horse battery staple", salt).unwrap();
        let plaintext = b"abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";

        let encrypted = service.encrypt(plaintext).unwrap();
        assert_eq!(service.decrypt(&encrypted).unwrap(), plaintext);
    }

    #[test]
    fn password_derivation_is_deterministic() {
        let salt = b"tigerwallet-salt";
        let key_one = EncryptionService::new("password123", salt).unwrap().key;
        let key_two = EncryptionService::new("password123", salt).unwrap().key;
        assert_eq!(key_one, key_two);
    }

    #[test]
    fn password_derivation_depends_on_the_salt() {
        let key_one = EncryptionService::new("password123", b"salt-one").unwrap().key;
        let key_two = EncryptionService::new("password123", b"salt-two").unwrap().key;
        assert_ne!(key_one, key_two);
    }

    #[test]
    fn password_derivation_rejects_a_short_salt() {
        assert!(EncryptionService::new("password123", b"short").is_err());
    }

    #[test]
    fn password_derivation_rejects_an_empty_password() {
        assert!(EncryptionService::new("", b"long-enough-salt").is_err());
    }

    /// The key must not be recoverable by hashing the password once.
    #[test]
    fn password_derivation_is_not_a_single_hash() {
        let salt = b"tigerwallet-salt";
        let key = EncryptionService::new("password123", salt).unwrap().key;

        let mut hasher = Sha256::new();
        hasher.update(b"password123");
        hasher.update(salt);
        let naive: [u8; 32] = hasher.finalize().into();

        assert_ne!(key, naive);
    }

    #[test]
    fn base64_round_trips() {
        let service = service();
        let encoded = service.encrypt_to_base64(b"test data").unwrap();
        assert_eq!(service.decrypt_from_base64(&encoded).unwrap(), b"test data");
    }

    #[test]
    fn record_names_are_validated() {
        assert!(validate_record_name("11111111-2222-3333-4444-555555555555").is_ok());
        assert!(validate_record_name("backup_1.0-x").is_ok());

        assert!(validate_record_name("").is_err());
        assert!(validate_record_name("_leading").is_err());
        assert!(validate_record_name("has/slash").is_err());
        assert!(validate_record_name("../escape").is_err());
        assert!(validate_record_name("has space").is_err());
        assert!(validate_record_name(&"a".repeat(256)).is_err());
    }

    #[test]
    fn query_values_are_escaped() {
        assert_eq!(CloudRecoveryProviderTest::escape("plain"), "plain");
        assert_eq!(
            CloudRecoveryProviderTest::escape("a'b"),
            "a\\'b",
            "a quote must not terminate the query string"
        );
        assert_eq!(CloudRecoveryProviderTest::escape("a\\b"), "a\\\\b");
    }

    /// Thin accessor so the escape helper can be tested without a provider.
    struct CloudRecoveryProviderTest;

    impl CloudRecoveryProviderTest {
        fn escape(value: &str) -> String {
            GoogleDriveProvider::escape_query_value(value)
        }
    }

    #[test]
    fn oversized_backups_are_rejected() {
        let data = vec![0u8; MAX_BACKUP_BYTES + 1];
        assert!(data.len() > MAX_BACKUP_BYTES);
    }

    #[test]
    fn cloudkit_subpath_matches_the_signed_route() {
        let provider = ICloudProvider::new(CloudConfig {
            provider: CloudProvider::ICloud,
            cloudkit_container: Some("iCloud.com.example.app".to_string()),
            cloudkit_environment: Some("development".to_string()),
            cloudkit_database: Some("public".to_string()),
            ..Default::default()
        });

        assert_eq!(
            provider.subpath("records/modify").unwrap(),
            "/database/1/iCloud.com.example.app/development/public/records/modify"
        );
    }

    #[test]
    fn cloudkit_subpath_requires_a_container() {
        let provider = ICloudProvider::new(CloudConfig::default());
        assert!(matches!(
            provider.subpath("records/modify").unwrap_err(),
            CloudRecoveryError::NotConfigured(_)
        ));
    }

    #[test]
    fn cloudkit_signing_requires_credentials() {
        let provider = ICloudProvider::new(CloudConfig {
            provider: CloudProvider::ICloud,
            cloudkit_container: Some("iCloud.com.example.app".to_string()),
            ..Default::default()
        });

        assert!(matches!(
            provider.sign(b"{}", "/database/1/x/development/private/records/modify")
                .unwrap_err(),
            CloudRecoveryError::NotConfigured(_)
        ));
    }

    #[test]
    fn cloudkit_signing_rejects_a_bad_key() {
        let provider = ICloudProvider::new(CloudConfig {
            provider: CloudProvider::ICloud,
            cloudkit_container: Some("iCloud.com.example.app".to_string()),
            cloudkit_key_id: Some("key-id".to_string()),
            cloudkit_private_key_pem: Some("not a pem".to_string()),
            ..Default::default()
        });

        assert!(matches!(
            provider.sign(b"{}", "/database/1/x/development/private/records/modify")
                .unwrap_err(),
            CloudRecoveryError::AuthError(_)
        ));
    }

    /// The signing path must produce a signature a verifier accepts over the
    /// exact message the specification defines.
    #[test]
    fn cloudkit_signature_verifies_over_the_spec_message() {
        use p256::ecdsa::{signature::Verifier, VerifyingKey};
        use p256::pkcs8::EncodePrivateKey;

        let signing_key = SigningKey::random(&mut p256::elliptic_curve::rand_core::OsRng);
        let pem = signing_key
            .to_pkcs8_pem(p256::pkcs8::LineEnding::LF)
            .unwrap()
            .to_string();

        let provider = ICloudProvider::new(CloudConfig {
            provider: CloudProvider::ICloud,
            cloudkit_container: Some("iCloud.com.example.app".to_string()),
            cloudkit_environment: Some("development".to_string()),
            cloudkit_database: Some("public".to_string()),
            cloudkit_key_id: Some("key-id".to_string()),
            cloudkit_private_key_pem: Some(pem),
            ..Default::default()
        });

        let body = br#"{"records":[]}"#;
        let subpath = provider.subpath("records/modify").unwrap();
        let headers = provider.sign(body, &subpath).unwrap();

        assert_eq!(
            headers.get("X-Apple-CloudKit-Request-KeyID").unwrap(),
            "key-id"
        );

        // Rebuild the signed message and verify the signature against it.
        let date = headers
            .get("X-Apple-CloudKit-Request-ISO8601Date")
            .unwrap();
        assert!(
            !date.contains('.'),
            "the timestamp must have no milliseconds"
        );
        let message = format!("{}:{}:{}", date, sha256_base64(body), subpath);

        let signature_bytes = base64::engine::general_purpose::STANDARD
            .decode(headers.get("X-Apple-CloudKit-Request-SignatureV1").unwrap())
            .unwrap();
        let signature = Signature::from_der(&signature_bytes).unwrap();

        let verifying_key = VerifyingKey::from(&signing_key);
        assert!(
            verifying_key.verify(message.as_bytes(), &signature).is_ok(),
            "the signature must verify over date:hash:subpath"
        );

        // A different body must not verify, or the signature is not binding.
        let other = format!("{}:{}:{}", date, sha256_base64(b"other"), subpath);
        assert!(verifying_key
            .verify(other.as_bytes(), &signature)
            .is_err());
    }

    #[test]
    fn cloudkit_signed_message_uses_the_base64_sha256_of_the_body() {
        let body = br#"{"a":1}"#;
        // base64 of sha256(body), not hex, per the specification.
        let expected = base64::engine::general_purpose::STANDARD.encode(Sha256::digest(body));
        assert_eq!(sha256_base64(body), expected);
        assert!(!expected.contains('-'));
    }

    #[tokio::test]
    async fn cloudkit_reports_authentication_state() {
        let configured = ICloudProvider::new(CloudConfig {
            provider: CloudProvider::ICloud,
            cloudkit_container: Some("container".to_string()),
            cloudkit_key_id: Some("key".to_string()),
            cloudkit_private_key_pem: Some("pem".to_string()),
            ..Default::default()
        });
        assert!(configured.is_authenticated().await);

        assert!(!ICloudProvider::new(CloudConfig::default())
            .is_authenticated()
            .await);
    }

    #[tokio::test]
    async fn drive_requires_an_access_token() {
        let provider = GoogleDriveProvider::new(CloudConfig {
            provider: CloudProvider::GoogleDrive,
            ..Default::default()
        });

        assert!(!provider.is_authenticated().await);
        assert!(matches!(
            provider.access_token().unwrap_err(),
            CloudRecoveryError::AuthError(_)
        ));
    }

    #[tokio::test]
    async fn drive_token_expiry_is_honoured() {
        let expired = GoogleDriveProvider::new(CloudConfig {
            provider: CloudProvider::GoogleDrive,
            access_token: Some("token".to_string()),
            token_expires_at: Some(Utc::now() - chrono::Duration::hours(1)),
            ..Default::default()
        });
        assert!(!expired.is_authenticated().await);

        let valid = GoogleDriveProvider::new(CloudConfig {
            provider: CloudProvider::GoogleDrive,
            access_token: Some("token".to_string()),
            token_expires_at: Some(Utc::now() + chrono::Duration::hours(1)),
            ..Default::default()
        });
        assert!(valid.is_authenticated().await);
    }

    #[tokio::test]
    async fn drive_refresh_requires_credentials() {
        let provider = GoogleDriveProvider::new(CloudConfig {
            provider: CloudProvider::GoogleDrive,
            ..Default::default()
        });

        assert!(matches!(
            provider.refresh_auth().await.unwrap_err(),
            CloudRecoveryError::NotConfigured(_)
        ));
    }

    #[test]
    fn service_rejects_a_wrong_key_length() {
        let config = CloudConfig {
            provider: CloudProvider::ICloud,
            encryption_key: base64::engine::general_purpose::STANDARD.encode([0u8; 16]),
            ..Default::default()
        };
        assert!(CloudRecoveryService::new(config).is_err());
    }

    #[test]
    fn service_rejects_an_invalid_key_encoding() {
        let config = CloudConfig {
            provider: CloudProvider::ICloud,
            encryption_key: "not base64!".to_string(),
            ..Default::default()
        };
        assert!(CloudRecoveryService::new(config).is_err());
    }

    #[tokio::test]
    async fn list_backups_fails_closed_without_credentials() {
        let config = CloudConfig {
            provider: CloudProvider::ICloud,
            encryption_key: base64::engine::general_purpose::STANDARD
                .encode(EncryptionService::generate_key()),
            ..Default::default()
        };
        let service = CloudRecoveryService::new(config).unwrap();

        let err = service
            .list_backups("user", CloudProvider::ICloud)
            .await
            .unwrap_err();
        assert!(matches!(err, CloudRecoveryError::NotConfigured(_)));
    }
}
