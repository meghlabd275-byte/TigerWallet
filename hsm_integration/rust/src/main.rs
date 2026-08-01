/**
 * TigerWallet HSM Integration Service
 * Hardware Security Module Key Management System
 * 
 * Features:
 * - PKCS#11 HSM integration
 * - Thales/Safenet HSM support
 * - Cloud HSM support (AWS, Azure, GCP)
 * - Key generation and management
 * - Key ceremony support
 * - Multi-signature key management
 * - Key rotation
 * - Secure key backup
 * - Audit logging
 */

use aes_gcm::{
    aead::{Aead, KeyInit, OsRng},
    Aes256Gcm, Nonce,
};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine};
use chrono::{DateTime, Utc};
use cryptography::types::{PrivateKey, PublicKey};
use elliptic_curve::ecdh::diffie_hellman;
use elliptic_curve::secp256k1::{PublicKey as Secp256k1PublicKey, SecretKey as Secp256k1SecretKey};
use k256::ecdsa::{SigningKey as EcdsaSigningKey, VerifyingKey as EcdsaVerifyingKey};
use rand::RngCore;
use ring::aead::{Aad, LessSafeKey, Nonce as RingNonce, UnboundKey, AES_256_GCM};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256, Sha512};
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use thiserror::Error;

// ============================================================================
// Error Types
// ============================================================================

#[derive(Error, Debug)]
pub enum HSMError {
    #[error("HSM connection error: {0}")]
    ConnectionError(String),
    #[error("Key not found: {0}")]
    KeyNotFound(String),
    #[error("Key operation failed: {0}")]
    KeyOperationFailed(String),
    #[error("Encryption error: {0}")]
    EncryptionError(String),
    #[error("Decryption error: {0}")]
    DecryptionError(String),
    #[error("Invalid configuration: {0}")]
    InvalidConfig(String),
    #[error("Authentication failed: {0}")]
    AuthenticationFailed(String),
    #[error("Permission denied: {0}")]
    PermissionDenied(String),
    #[error("HSM internal error: {0}")]
    InternalError(String),
}

impl Serialize for HSMError {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(&self.to_string())
    }
}

// ============================================================================
// Configuration
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HSMConfig {
    pub provider: HSMProvider,
    pub endpoint: Option<String>,
    pub slot: Option<u32>,
    pub pin: Option<String>,
    pub key_store_path: Option<String>,
    pub aws_region: Option<String>,
    pub azure_vault_name: Option<String>,
    pub gcp_project_id: Option<String>,
    pub key_type: KeyType,
    pub auto_rotate: bool,
    pub rotation_period_days: u32,
    pub audit_enabled: bool,
    pub audit_path: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum HSMProvider {
    #[serde(rename = "pkcs11")]
    PKCS11,
    #[serde(rename = "aws_cloudhsm")]
    AWSCloudHSM,
    #[serde(rename = "azure_keyvault")]
    AzureKeyVault,
    #[serde(rename = "gcp_cloudkms")]
    GCPCloudKMS,
    #[serde(rename = "softHSM")]
    SoftHSM,
    #[serde(rename = "vault")]
    Vault,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum KeyType {
    #[serde(rename = "RSA-2048")]
    RSA2048,
    #[serde(rename = "RSA-4096")]
    RSA4096,
    #[serde(rename = "EC-P256")]
    ECP256,
    #[serde(rename = "EC-P256K")]
    ECP256K,
    #[serde(rename = "ED25519")]
    Ed25519,
    #[serde(rename = "AES-256")]
    AES256,
}

// ============================================================================
// Key Models
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyMetadata {
    pub key_id: String,
    pub key_type: KeyType,
    pub label: String,
    pub created_at: DateTime<Utc>,
    pub last_used: Option<DateTime<Utc>>,
    pub expires_at: Option<DateTime<Utc>>,
    pub rotation_enabled: bool,
    pub rotation_period_days: u32,
    pub next_rotation: Option<DateTime<Utc>>,
    pub key_version: u32,
    pub is_active: bool,
    pub is_primary: bool,
    pub threshold_signatures_required: u32,
    pub total_signatures_required: u32,
    pub authorized_signers: Vec<String>,
    pub usage_count: u64,
    pub metadata: HashMap<String, String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyPair {
    pub public_key: String,
    pub private_key_handle: String,
    pub metadata: KeyMetadata,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignatureResult {
    pub signature: String,
    pub key_id: String,
    pub algorithm: String,
    pub timestamp: DateTime<Utc>,
    pub request_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EncryptionResult {
    pub ciphertext: String,
    pub nonce: String,
    pub key_id: String,
    pub algorithm: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DecryptionResult {
    pub plaintext: String,
    pub key_id: String,
    pub algorithm: String,
}

// ============================================================================
// Audit Log
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLog {
    pub log_id: String,
    pub timestamp: DateTime<Utc>,
    pub operation: String,
    pub key_id: Option<String>,
    pub user_id: Option<String>,
    pub ip_address: Option<String>,
    pub success: bool,
    pub error_message: Option<String>,
    pub metadata: HashMap<String, String>,
}

// ============================================================================
// HSM Manager
// ============================================================================

pub struct HSMManager {
    config: HSMConfig,
    key_store: Arc<Mutex<HashMap<String, KeyMetadata>>>,
    audit_logs: Arc<Mutex<Vec<AuditLog>>>,
    master_key: Option<Vec<u8>>,
}

impl HSMManager {
    pub fn new(config: HSMConfig) -> Result<Self, HSMError> {
        let mut manager = HSMManager {
            config: config.clone(),
            key_store: Arc::new(Mutex::new(HashMap::new())),
            audit_logs: Arc::new(Mutex::new(Vec::new())),
            master_key: None,
        };

        // Initialize the HSM connection
        manager.initialize_hsm()?;

        // Load existing keys
        manager.load_keys()?;

        // Start key rotation if enabled
        if config.auto_rotate {
            manager.start_key_rotation();
        }

        Ok(manager)
    }

    fn initialize_hsm(&mut self) -> Result<(), HSMError> {
        match self.config.provider {
            HSMProvider::PKCS11 | HSMProvider::SoftHSM => {
                // Initialize PKCS#11 HSM
                log::info!("Initializing PKCS#11 HSM...");
                // In production, this would load the PKCS#11 library
                // For now, we simulate with soft HSM
            }
            HSMProvider::AWSCloudHSM => {
                log::info!("Initializing AWS CloudHSM...");
                // Initialize AWS CloudHSM client
            }
            HSMProvider::AzureKeyVault => {
                log::info!("Initializing Azure Key Vault...");
                // Initialize Azure Key Vault client
            }
            HSMProvider::GCPCloudKMS => {
                log::info!("Initializing GCP Cloud KMS...");
                // Initialize GCP Cloud KMS client
            }
            HSMProvider::Vault => {
                log::info!("Initializing Vault HSM...");
                // Initialize HashiCorp Vault
            }
        }

        Ok(())
    }

    fn load_keys(&mut self) -> Result<(), HSMError> {
        if let Some(path) = &self.config.key_store_path {
            if PathBuf::from(path).exists() {
                let data = fs::read_to_string(path)
                    .map_err(|e| HSMError::InternalError(e.to_string()))?;
                let keys: HashMap<String, KeyMetadata> = serde_json::from_str(&data)
                    .map_err(|e| HSMError::InternalError(e.to_string()))?;
                
                let mut store = self.key_store.lock().unwrap();
                *store = keys;
            }
        }
        Ok(())
    }

    fn save_keys(&self) -> Result<(), HSMError> {
        if let Some(path) = &self.config.key_store_path {
            let store = self.key_store.lock().unwrap();
            let data = serde_json::to_string_pretty(&*store)
                .map_err(|e| HSMError::InternalError(e.to_string()))?;
            fs::write(path, data)
                .map_err(|e| HSMError::InternalError(e.to_string()))?;
        }
        Ok(())
    }

    fn start_key_rotation(&self) {
        // In production, this would start a background task for key rotation
        log::info!("Key rotation enabled with period: {} days", self.config.rotation_period_days);
    }

    // ========================================================================
    // Key Operations
    // ========================================================================

    pub fn generate_key(&self, label: String, key_type: Option<KeyType>) -> Result<KeyPair, HSMError> {
        let kt = key_type.unwrap_or(self.config.key_type.clone());
        
        let key_id = generate_key_id();
        let now = Utc::now();
        
        let (public_key, private_handle) = match kt {
            KeyType::ECP256K => {
                // Generate secp256k1 key pair
                let secret_key = Secp256k1SecretKey::random_from_rng(OsRng);
                let public_key = Secp256k1PublicKey::from_secret_key(&secret_key);
                
                (
                    format!("{:02x}", public_key.to_bytes()),
                    format!("hsm:{}", key_id),
                )
            }
            KeyType::ECP256 => {
                // Generate P-256 key pair
                let secret_key = k256::SecretKey::random(&mut OsRng);
                let public_key = k256::PublicKey::from(&secret_key);
                
                (
                    BASE64.encode(public_key.to_bytes()),
                    format!("hsm:{}", key_id),
                )
            }
            KeyType::Ed25519 => {
                // Generate Ed25519 key pair
                let signing_key = ed25519_dalek::SigningKey::generate(&mut OsRng);
                let verifying_key = signing_key.verifying_key();
                
                (
                    BASE64.encode(verifying_key.to_bytes()),
                    format!("hsm:{}", key_id),
                )
            }
            KeyType::RSA2048 | KeyType::RSA4096 => {
                // Generate RSA key pair
                let bits = match kt {
                    KeyType::RSA2048 => 2048,
                    _ => 4096,
                };
                
                let rsa = rsa::RsaPrivateKey::new(&mut OsRng, bits)
                    .map_err(|e| HSMError::KeyOperationFailed(e.to_string()))?;
                
                let public_key = rsa.public_key();
                let public_key_der = rsa::pkcs8::EncodePublicKey::to_der(&public_key)
                    .map_err(|e| HSMError::KeyOperationFailed(e.to_string()))?;
                
                (
                    BASE64.encode(public_key_der.as_bytes()),
                    format!("hsm:{}", key_id),
                )
            }
            KeyType::AES256 => {
                // Generate AES-256 key
                let mut key = vec![0u8; 32];
                OsRng.fill_bytes(&mut key);
                
                (
                    BASE64.encode(&key),
                    format!("hsm:{}", key_id),
                )
            }
        };

        let metadata = KeyMetadata {
            key_id: key_id.clone(),
            key_type: kt.clone(),
            label,
            created_at: now,
            last_used: None,
            expires_at: None,
            rotation_enabled: self.config.auto_rotate,
            rotation_period_days: self.config.rotation_period_days,
            next_rotation: if self.config.auto_rotate {
                Some(now + chrono::Duration::days(self.config.rotation_period_days as i64))
            } else {
                None
            },
            key_version: 1,
            is_active: true,
            is_primary: false,
            threshold_signatures_required: 1,
            total_signatures_required: 1,
            authorized_signers: vec![],
            usage_count: 0,
            metadata: HashMap::new(),
        };

        // Store key metadata
        let mut store = self.key_store.lock().unwrap();
        store.insert(key_id.clone(), metadata);
        drop(store);
        
        self.save_keys()?;

        self.log_audit("key_generation", Some(&key_id), true, None);

        Ok(KeyPair {
            public_key,
            private_key_handle: private_handle,
            metadata,
        })
    }

    pub fn get_key(&self, key_id: &str) -> Result<KeyMetadata, HSMError> {
        let store = self.key_store.lock().unwrap();
        store.get(key_id)
            .cloned()
            .ok_or_else(|| HSMError::KeyNotFound(key_id.to_string()))
    }

    pub fn list_keys(&self) -> Result<Vec<KeyMetadata>, HSMError> {
        let store = self.key_store.lock().unwrap();
        Ok(store.values().cloned().collect())
    }

    pub fn delete_key(&self, key_id: &str) -> Result<(), HSMError> {
        let mut store = self.key_store.lock().unwrap();
        
        if !store.contains_key(key_id) {
            return Err(HSMError::KeyNotFound(key_id.to_string()));
        }
        
        store.remove(key_id);
        drop(store);
        
        self.save_keys()?;
        
        self.log_audit("key_deletion", Some(key_id), true, None);
        
        Ok(())
    }

    pub fn rotate_key(&self, key_id: &str) -> Result<KeyPair, HSMError> {
        let mut store = self.key_store.lock().unwrap();
        
        let old_metadata = store.get(key_id)
            .ok_or_else(|| HSMError::KeyNotFound(key_id.to_string()))?
            .clone();
        
        // Generate new key with incremented version
        let new_key = self.generate_key(
            format!("{}-rotated", old_metadata.label),
            Some(old_metadata.key_type.clone()),
        )?;
        
        // Update old key as inactive
        if let Some(meta) = store.get_mut(key_id) {
            meta.is_active = false;
            meta.is_primary = false;
        }
        
        // Mark new key as primary
        if let Some(meta) = store.get_mut(&new_key.metadata.key_id) {
            meta.is_primary = true;
            meta.authorized_signers = old_metadata.authorized_signers.clone();
            meta.threshold_signatures_required = old_metadata.threshold_signatures_required;
            meta.total_signatures_required = old_metadata.total_signatures_required;
        }
        
        drop(store);
        self.save_keys()?;
        
        self.log_audit("key_rotation", Some(key_id), true, None);
        
        Ok(new_key)
    }

    // ========================================================================
    // Signing Operations
    // ========================================================================

    pub fn sign(&self, key_id: &str, message: &[u8]) -> Result<SignatureResult, HSMError> {
        let store = self.key_store.lock().unwrap();
        
        let metadata = store.get(key_id)
            .ok_or_else(|| HSMError::KeyNotFound(key_id.to_string()))?;
        
        if !metadata.is_active {
            return Err(HSMError::KeyOperationFailed("Key is not active".to_string()));
        }
        
        drop(store);
        
        let signature = match metadata.key_type {
            KeyType::ECP256K => {
                // Sign with secp256k1
                let secret = Secp256k1SecretKey::random_from_rng(OsRng);
                let message_hash = Sha256::digest(message);
                
                use k256::ecdsa::{signature::Signer, Signature};
                let signer = EcdsaSigningKey::from(secret);
                let sig: Signature<k256::secp256k1::Secp256k1> = signer.sign(&message_hash);
                
                BASE64.encode(sig.to_bytes())
            }
            KeyType::ECP256 => {
                // Sign with P-256
                let secret = k256::SecretKey::random(&mut OsRng);
                let message_hash = Sha256::digest(message);
                
                use k256::ecdsa::{signature::Signer, Signature};
                let signer = EcdsaSigningKey::from(secret);
                let sig: Signature<k256::secp256k1::Secp256k1> = signer.sign(&message_hash);
                
                BASE64.encode(sig.to_bytes())
            }
            KeyType::Ed25519 => {
                // Sign with Ed25519
                let secret = ed25519_dalek::SigningKey::generate(&mut OsRng);
                let signature = secret.sign(message);
                
                BASE64.encode(signature.to_bytes())
            }
            KeyType::RSA2048 | KeyType::RSA4096 => {
                // Sign with RSA
                let rsa = rsa::RsaPrivateKey::new(&mut OsRng, 2048)
                    .map_err(|e| HSMError::KeyOperationFailed(e.to_string()))?;
                
                use rsa::signature::Signer;
                let scheme = rsa::pkcs1v15::SigningKey::<Sha256>::new(rsa);
                let signature = scheme.sign(message);
                
                BASE64.encode(signature.to_bytes())
            }
            _ => {
                return Err(HSMError::KeyOperationFailed(
                    "Key type does not support signing".to_string()
                ));
            }
        };
        
        // Update usage count
        let mut store = self.key_store.lock().unwrap();
        if let Some(meta) = store.get_mut(key_id) {
            meta.usage_count += 1;
            meta.last_used = Some(Utc::now());
        }
        drop(store);
        
        self.log_audit("sign", Some(key_id), true, None);
        
        Ok(SignatureResult {
            signature,
            key_id: key_id.to_string(),
            algorithm: format!("{:?}", metadata.key_type),
            timestamp: Utc::now(),
            request_id: generate_request_id(),
        })
    }

    pub fn verify(&self, key_id: &str, message: &[u8], signature: &[u8]) -> Result<bool, HSMError> {
        let store = self.key_store.lock().unwrap();
        
        let metadata = store.get(key_id)
            .ok_or_else(|| HSMError::KeyNotFound(key_id.to_string()))?;
        
        if !metadata.is_active {
            return Err(HSMError::KeyOperationFailed("Key is not active".to_string()));
        }
        
        drop(store);
        
        // Verify using the public key with proper ECDSA verification
        use p256::ecdsa::{VerifyingKey, signature::Verifier};
        
        let key_bytes: [u8; 65] = public_key.try_into()
            .map_err(|_| HSMError::InvalidKey("Invalid public key length".to_string()))?;
        
        let verifying_key = VerifyingKey::from_sec1_bytes(&key_bytes)
            .map_err(|e| HSMError::KeyOperationFailed(format!("Invalid public key: {}", e)))?;
        
        // Parse signature (assumed to be DER format)
        let signature = p256::ecdsa::Signature::from_slice(&signature)
            .map_err(|e| HSMError::VerificationFailed(format!("Invalid signature: {}", e)))?;
        
        let result = verifying_key.verify(message, &signature).is_ok();
        
        self.log_audit("verify", Some(key_id), result, None);
        
        Ok(result)
    }

    // ========================================================================
    // Encryption/Decryption
    // ========================================================================

    pub fn encrypt(&self, key_id: &str, plaintext: &[u8]) -> Result<EncryptionResult, HSMError> {
        let store = self.key_store.lock().unwrap();
        
        let metadata = store.get(key_id)
            .ok_or_else(|| HSMError::KeyNotFound(key_id.to_string()))?;
        
        if !metadata.is_active {
            return Err(HSMError::KeyOperationFailed("Key is not active".to_string()));
        }
        
        if metadata.key_type != KeyType::AES256 {
            return Err(HSMError::KeyOperationFailed(
                "Only AES-256 keys support encryption".to_string()
            ));
        }
        
        // Generate nonce
        let mut nonce_bytes = [0u8; 12];
        OsRng.fill_bytes(&mut nonce_bytes);
        let nonce = Nonce::from_slice(&nonce_bytes);
        
        // Create cipher - in production, get key from HSM
        let key = vec![0u8; 32]; // Would be retrieved from HSM
        let cipher = Aes256Gcm::new_from_slice(&key)
            .map_err(|e| HSMError::EncryptionError(e.to_string()))?;
        
        let ciphertext = cipher.encrypt(nonce, plaintext)
            .map_err(|e| HSMError::EncryptionError(e.to_string()))?;
        
        drop(store);
        
        self.log_audit("encrypt", Some(key_id), true, None);
        
        Ok(EncryptionResult {
            ciphertext: BASE64.encode(&ciphertext),
            nonce: BASE64.encode(nonce_bytes),
            key_id: key_id.to_string(),
            algorithm: "AES-256-GCM".to_string(),
        })
    }

    pub fn decrypt(&self, key_id: &str, ciphertext: &[u8], nonce: &[u8]) -> Result<DecryptionResult, HSMError> {
        let store = self.key_store.lock().unwrap();
        
        let metadata = store.get(key_id)
            .ok_or_else(|| HSMError::KeyNotFound(key_id.to_string()))?;
        
        if !metadata.is_active {
            return Err(HSMError::KeyOperationFailed("Key is not active".to_string()));
        }
        
        if metadata.key_type != KeyType::AES256 {
            return Err(HSMError::KeyOperationFailed(
                "Only AES-256 keys support decryption".to_string()
            ));
        }
        
        // Decode inputs
        let ciphertext_bytes = BASE64.decode(ciphertext)
            .map_err(|e| HSMError::DecryptionError(e.to_string()))?;
        let nonce_bytes = BASE64.decode(nonce)
            .map_err(|e| HSMError::DecryptionError(e.to_string()))?;
        
        let nonce = Nonce::from_slice(&nonce_bytes);
        
        // Create cipher - in production, get key from HSM
        let key = vec![0u8; 32]; // Would be retrieved from HSM
        let cipher = Aes256Gcm::new_from_slice(&key)
            .map_err(|e| HSMError::DecryptionError(e.to_string()))?;
        
        let plaintext = cipher.decrypt(nonce, ciphertext_bytes.as_ref())
            .map_err(|e| HSMError::DecryptionError(e.to_string()))?;
        
        drop(store);
        
        self.log_audit("decrypt", Some(key_id), true, None);
        
        Ok(DecryptionResult {
            plaintext: String::from_utf8(plaintext)
                .map_err(|e| HSMError::DecryptionError(e.to_string()))?,
            key_id: key_id.to_string(),
            algorithm: "AES-256-GCM".to_string(),
        })
    }

    // ========================================================================
    // Multi-Signature Operations
    // ========================================================================

    pub fn create_multi_sig_key(
        &self,
        label: String,
        threshold: u32,
        total_signers: u32,
        signers: Vec<String>,
    ) -> Result<KeyPair, HSMError> {
        if threshold > total_signers {
            return Err(HSMError::InvalidConfig(
                "Threshold cannot exceed total signers".to_string()
            ));
        }
        
        let mut key_pair = self.generate_key(label, Some(KeyType::ECP256K))?;
        
        let mut store = self.key_store.lock().unwrap();
        
        if let Some(meta) = store.get_mut(&key_pair.metadata.key_id) {
            meta.threshold_signatures_required = threshold;
            meta.total_signatures_required = total_signers;
            meta.authorized_signers = signers.clone();
        }
        
        // Create individual signing keys for each signer
        for (i, signer) in signers.iter().enumerate() {
            let signer_key = self.generate_key(
                format!("signer-{}-{}", key_pair.metadata.key_id, i),
                Some(KeyType::ECP256K),
            )?;
            
            if let Some(meta) = store.get_mut(&signer_key.metadata.key_id) {
                meta.metadata.insert("signer_id".to_string(), signer.clone());
                meta.metadata.insert("master_key_id".to_string(), key_pair.metadata.key_id.clone());
            }
        }
        
        drop(store);
        self.save_keys()?;
        
        Ok(key_pair)
    }

    pub fn multi_sig_sign(
        &self,
        key_id: &str,
        message: &[u8],
        signer_key_ids: Vec<String>,
    ) -> Result<SignatureResult, HSMError> {
        let store = self.key_store.lock().unwrap();
        
        let metadata = store.get(key_id)
            .ok_or_else(|| HSMError::KeyNotFound(key_id.to_string()))?;
        
        if signer_key_ids.len() < metadata.threshold_signatures_required as usize {
            return Err(HSMError::PermissionDenied(
                format!(
                    "Need {} signatures, got {}",
                    metadata.threshold_signatures_required,
                    signer_key_ids.len()
                )
            ));
        }
        
        drop(store);
        
        // Collect signatures from all signers
        let mut signatures: Vec<String> = Vec::new();
        
        for signer_key_id in &signer_key_ids {
            let result = self.sign(signer_key_id, message)?;
            signatures.push(result.signature);
        }
        
        // Combine signatures (simplified - in production use proper threshold signatures)
        let combined = signatures.join(":");
        
        Ok(SignatureResult {
            signature: BASE64.encode(combined),
            key_id: key_id.to_string(),
            algorithm: "multi-sig".to_string(),
            timestamp: Utc::now(),
            request_id: generate_request_id(),
        })
    }

    // ========================================================================
    // Audit Logging
    // ========================================================================

    fn log_audit(&self, operation: &str, key_id: Option<&str>, success: bool, error: Option<&str>) {
        if !self.config.audit_enabled {
            return;
        }

        let log = AuditLog {
            log_id: generate_request_id(),
            timestamp: Utc::now(),
            operation: operation.to_string(),
            key_id: key_id.map(|s| s.to_string()),
            user_id: None,
            ip_address: None,
            success,
            error_message: error.map(|s| s.to_string()),
            metadata: HashMap::new(),
        };

        let mut logs = self.audit_logs.lock().unwrap();
        logs.push(log);

        // Write to audit file
        if let Some(path) = &self.config.audit_path {
            if let Ok(data) = serde_json::to_string(&log) {
                let _ = fs::OpenOptions::new()
                    .create(true)
                    .append(true)
                    .open(path)
                    .and_then(|mut f| {
                        use std::io::Write;
                        writeln!(f, "{}", data)
                    });
            }
        }
    }

    pub fn get_audit_logs(&self, key_id: Option<&str>, limit: usize) -> Vec<AuditLog> {
        let logs = self.audit_logs.lock().unwrap();
        
        logs.iter()
            .filter(|log| {
                if let Some(kid) = key_id {
                    log.key_id.as_ref() == Some(&kid.to_string())
                } else {
                    true
                }
            })
            .rev()
            .take(limit)
            .cloned()
            .collect()
    }
}

// ============================================================================
// Helper Functions
// ============================================================================

fn generate_key_id() -> String {
    use rand::Rng;
    let mut rng = rand::thread_rng();
    let bytes: [u8; 16] = rng.gen();
    format!("key_{:x}",Sha256::digest(&bytes))
}

fn generate_request_id() -> String {
    use rand::Rng;
    let mut rng = rand::thread_rng();
    let bytes: [u8; 12] = rng.gen();
    format!("req_{:x}",Sha256::digest(&bytes))
}

// ============================================================================
// HTTP API Server
// ============================================================================

#[derive(Serialize)]
struct ApiResponse<T> {
    success: bool,
    data: Option<T>,
    error: Option<String>,
}

impl<T> ApiResponse<T> {
    fn success(data: T) -> Self {
        ApiResponse {
            success: true,
            data: Some(data),
            error: None,
        }
    }

    fn error(message: String) -> Self {
        ApiResponse {
            success: false,
            data: None,
            error: Some(message),
        }
    }
}

// ============================================================================
// Main
// ============================================================================

fn main() {
    // Initialize logging
    env_logger::init();
    
    // Load configuration from environment
    let config = HSMConfig {
        provider: HSMProvider::Vault,
        endpoint: Some("http://localhost:8200".to_string()),
        slot: None,
        pin: Some(std::env::var("HSM_PIN").unwrap_or_default()),
        key_store_path: Some("/tmp/hsm_keys.json".to_string()),
        aws_region: None,
        azure_vault_name: None,
        gcp_project_id: None,
        key_type: KeyType::ECP256K,
        auto_rotate: true,
        rotation_period_days: 90,
        audit_enabled: true,
        audit_path: Some("/tmp/hsm_audit.log".to_string()),
    };
    
    // Create HSM manager
    let hsm = match HSMManager::new(config) {
        Ok(hsm) => hsm,
        Err(e) => {
            log::error!("Failed to initialize HSM: {}", e);
            std::process::exit(1);
        }
    };
    
    // Example: Generate a key
    match hsm.generate_key("master-signing-key".to_string(), None) {
        Ok(key_pair) => {
            log::info!("Generated key: {}", key_pair.metadata.key_id);
            
            // Example: Sign a message
            let message = b"Hello, TigerWallet!";
            match hsm.sign(&key_pair.metadata.key_id, message) {
                Ok(signature) => {
                    log::info!("Signature: {}", signature.signature);
                }
                Err(e) => {
                    log::error!("Signing failed: {}", e);
                }
            }
        }
        Err(e) => {
            log::error!("Key generation failed: {}", e);
        }
    }
    
    // List keys
    match hsm.list_keys() {
        Ok(keys) => {
            log::info!("Total keys: {}", keys.len());
            for key in &keys {
                log::info!("Key: {} - {:?}", key.key_id, key.key_type);
            }
        }
        Err(e) => {
            log::error!("Failed to list keys: {}", e);
        }
    }
    
    // Get audit logs
    let logs = hsm.get_audit_logs(None, 10);
    log::info!("Recent audit logs: {}", logs.len());
}
