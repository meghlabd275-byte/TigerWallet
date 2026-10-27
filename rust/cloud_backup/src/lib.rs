//! Encrypted cloud backup for wallet material.
//!
//! # Key handling
//!
//! Backups are encrypted with AES-256-GCM under a key derived from a
//! caller-supplied passphrase and a random 16-byte salt. The passphrase never
//! leaves the caller; the store only ever sees ciphertext.
//!
//! An earlier version derived the key from the user id. A user id is public
//! data, so that key was guessable by anyone who knew which account a backup
//! belonged to, and the same key was reused for every backup a user made.
//! Derivation now requires a passphrase, uses PBKDF2-HMAC-SHA256 with a
//! per-backup random salt, and the salt is stored alongside the ciphertext.
//!
//! # Integrity
//!
//! AES-GCM authenticates the ciphertext, so tampering is detected on decrypt
//! rather than after the data has been trusted. A separate checksum guards
//! against silently truncated records.

use aes_gcm::aead::{Aead, KeyInit, Payload};
use aes_gcm::{Aes256Gcm, Key, Nonce};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::HashMap;
use std::sync::Mutex;
use std::time::{SystemTime, UNIX_EPOCH};

const SALT_LEN: usize = 16;
const NONCE_LEN: usize = 12;
const KEY_LEN: usize = 32;
/// PBKDF2 rounds. Chosen so a backup costs a fraction of a second to open
/// while making an offline guessing attack on the passphrase expensive.
const PBKDF2_ROUNDS: u32 = 210_000;
/// Bound on how much plaintext a single backup may carry (16 MiB).
const MAX_BACKUP_BYTES: usize = 16 * 1024 * 1024;

/// A stored backup. Only the ciphertext and public parameters are persisted.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WalletBackup {
    pub id: String,
    pub user_id: String,
    /// AES-256-GCM ciphertext with the 16-byte tag appended.
    pub encrypted_data: Vec<u8>,
    pub nonce: Vec<u8>,
    /// PBKDF2 salt for this backup.
    pub salt: Vec<u8>,
    pub created_at: u64,
    pub updated_at: u64,
    pub version: u32,
    /// SHA-256 of the plaintext, used to confirm a restore is byte-exact.
    pub checksum: String,
}

/// Non-secret metadata used to list a user's backups.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupMetadata {
    pub id: String,
    pub user_id: String,
    pub chain_type: String,
    pub wallet_count: u32,
    pub size_bytes: u64,
    pub created_at: u64,
    pub is_auto: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupConfig {
    pub auto_backup: bool,
    pub backup_interval_minutes: u32,
    pub max_backups: u32,
    pub encrypt_before_upload: bool,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum RecoveryMethod {
    SeedPhrase,
    GoogleDrive,
    ICloud,
    CustomCloud,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct BackupStats {
    pub total_backups: u64,
    pub total_size_bytes: u64,
    pub successful_restores: u64,
    pub failed_backups: u64,
    pub last_backup: Option<u64>,
}

#[derive(Debug, PartialEq, Eq)]
pub enum BackupError {
    EncryptionFailed,
    DecryptionFailed,
    AuthenticationFailed,
    NotFound,
    StorageError,
    InvalidConfig,
    /// The passphrase was missing, too short, or the input was malformed.
    InvalidPassphrase(String),
    /// The backup exceeds `MAX_BACKUP_BYTES`.
    PayloadTooLarge(usize),
}

impl std::fmt::Display for BackupError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            BackupError::EncryptionFailed => write!(f, "failed to encrypt data"),
            BackupError::DecryptionFailed => write!(f, "failed to decrypt data"),
            BackupError::AuthenticationFailed => write!(f, "authentication failed"),
            BackupError::NotFound => write!(f, "backup not found"),
            BackupError::StorageError => write!(f, "storage error"),
            BackupError::InvalidConfig => write!(f, "invalid configuration"),
            BackupError::InvalidPassphrase(msg) => write!(f, "invalid passphrase: {}", msg),
            BackupError::PayloadTooLarge(n) => {
                write!(f, "backup of {} bytes exceeds the limit", n)
            }
        }
    }
}

impl std::error::Error for BackupError {}

// ============================================================================
// Key derivation and encryption
// ============================================================================

/// Derive the encryption key for one backup.
///
/// The salt must be unique per backup: reusing it with the same passphrase
/// would produce the same key and make the two ciphertexts comparable.
fn derive_key(passphrase: &str, salt: &[u8]) -> Result<[u8; KEY_LEN], BackupError> {
    if passphrase.len() < 8 {
        return Err(BackupError::InvalidPassphrase(
            "must be at least 8 characters".into(),
        ));
    }
    if salt.len() != SALT_LEN {
        return Err(BackupError::InvalidConfig);
    }

    let mut key = [0u8; KEY_LEN];
    pbkdf2::pbkdf2_hmac::<Sha256>(passphrase.as_bytes(), salt, PBKDF2_ROUNDS, &mut key);
    Ok(key)
}

/// Encrypt plaintext under a passphrase.
///
/// The additional authenticated data binds the ciphertext to the owning user
/// and backup id, so a ciphertext cannot be moved to another record.
fn encrypt_data(
    plaintext: &[u8],
    passphrase: &str,
    user_id: &str,
    backup_id: &str,
    salt: &[u8],
) -> Result<(Vec<u8>, [u8; NONCE_LEN]), BackupError> {
    if plaintext.len() > MAX_BACKUP_BYTES {
        return Err(BackupError::PayloadTooLarge(plaintext.len()));
    }

    let key = derive_key(passphrase, salt)?;
    let cipher = Aes256Gcm::new(Key::<Aes256Gcm>::from_slice(&key));

    let mut nonce_bytes = [0u8; NONCE_LEN];
    getrandom::getrandom(&mut nonce_bytes).map_err(|_| BackupError::EncryptionFailed)?;
    let nonce = Nonce::from_slice(&nonce_bytes);

    let aad = format!("tigerwallet:backup:v1:{}:{}", user_id, backup_id);
    let ciphertext = cipher
        .encrypt(
            nonce,
            Payload {
                msg: plaintext,
                aad: aad.as_bytes(),
            },
        )
        .map_err(|_| BackupError::EncryptionFailed)?;

    Ok((ciphertext, nonce_bytes))
}

/// Decrypt and authenticate a backup.
fn decrypt_data(
    ciphertext: &[u8],
    passphrase: &str,
    user_id: &str,
    backup_id: &str,
    salt: &[u8],
    nonce_bytes: &[u8],
) -> Result<Vec<u8>, BackupError> {
    if nonce_bytes.len() != NONCE_LEN {
        return Err(BackupError::DecryptionFailed);
    }
    let key = derive_key(passphrase, salt)?;
    let cipher = Aes256Gcm::new(Key::<Aes256Gcm>::from_slice(&key));
    let nonce = Nonce::from_slice(nonce_bytes);

    let aad = format!("tigerwallet:backup:v1:{}:{}", user_id, backup_id);
    cipher
        .decrypt(
            nonce,
            Payload {
                msg: ciphertext,
                aad: aad.as_bytes(),
            },
        )
        // A wrong passphrase, a tampered ciphertext, or a record moved to a
        // different user all land here. Distinguishing them would leak
        // information about the key.
        .map_err(|_| BackupError::AuthenticationFailed)
}

fn random_salt() -> Result<[u8; SALT_LEN], BackupError> {
    let mut salt = [0u8; SALT_LEN];
    getrandom::getrandom(&mut salt).map_err(|_| BackupError::EncryptionFailed)?;
    Ok(salt)
}

fn random_id() -> Result<String, BackupError> {
    let mut buf = [0u8; 16];
    getrandom::getrandom(&mut buf).map_err(|_| BackupError::EncryptionFailed)?;
    Ok(format!("backup_{}", hex::encode(buf)))
}

fn current_timestamp() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

fn calculate_checksum(data: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(data);
    hex::encode(hasher.finalize())
}

// ============================================================================
// Store
// ============================================================================

pub struct CloudBackupStore {
    backups: Mutex<HashMap<String, Vec<WalletBackup>>>,
    metadata: Mutex<HashMap<String, Vec<BackupMetadata>>>,
    config: Mutex<HashMap<String, BackupConfig>>,
    stats: Mutex<BackupStats>,
}

impl Default for CloudBackupStore {
    fn default() -> Self {
        Self::new()
    }
}

impl CloudBackupStore {
    pub fn new() -> Self {
        Self {
            backups: Mutex::new(HashMap::new()),
            metadata: Mutex::new(HashMap::new()),
            config: Mutex::new(HashMap::new()),
            stats: Mutex::new(BackupStats::default()),
        }
    }

    /// Encrypt `data` and store it as a new backup.
    pub fn create_backup(
        &self,
        user_id: &str,
        data: &[u8],
        chain_type: &str,
        passphrase: &str,
    ) -> Result<WalletBackup, BackupError> {
        if user_id.trim().is_empty() {
            return Err(BackupError::InvalidConfig);
        }

        let id = random_id()?;
        let salt = random_salt()?;
        let (encrypted_data, nonce) = encrypt_data(data, passphrase, user_id, &id, &salt)?;

        let now = current_timestamp();
        let backup = WalletBackup {
            id: id.clone(),
            user_id: user_id.to_string(),
            encrypted_data,
            nonce: nonce.to_vec(),
            salt: salt.to_vec(),
            created_at: now,
            updated_at: now,
            version: 1,
            checksum: calculate_checksum(data),
        };

        let mut backups = self.backups.lock().map_err(|_| BackupError::StorageError)?;
        backups
            .entry(user_id.to_string())
            .or_default()
            .push(backup.clone());

        let metadata = BackupMetadata {
            id,
            user_id: user_id.to_string(),
            chain_type: chain_type.to_string(),
            wallet_count: 1,
            size_bytes: data.len() as u64,
            created_at: now,
            is_auto: true,
        };

        let mut meta_store = self.metadata.lock().map_err(|_| BackupError::StorageError)?;
        meta_store
            .entry(user_id.to_string())
            .or_default()
            .push(metadata);

        let mut stats = self.stats.lock().map_err(|_| BackupError::StorageError)?;
        stats.total_backups += 1;
        stats.total_size_bytes += data.len() as u64;
        stats.last_backup = Some(now);

        Ok(backup)
    }

    /// List the non-secret metadata for a user's backups.
    pub fn list_backups(&self, user_id: &str) -> Vec<BackupMetadata> {
        self.metadata
            .lock()
            .map(|m| m.get(user_id).cloned().unwrap_or_default())
            .unwrap_or_default()
    }

    pub fn get_backup(&self, user_id: &str, backup_id: &str) -> Option<WalletBackup> {
        self.backups.lock().ok().and_then(|backups| {
            backups
                .get(user_id)
                .and_then(|list| list.iter().find(|b| b.id == backup_id).cloned())
        })
    }

    /// Decrypt a backup with the user's passphrase.
    pub fn restore_backup(
        &self,
        user_id: &str,
        backup_id: &str,
        passphrase: &str,
    ) -> Result<Vec<u8>, BackupError> {
        let backup = self
            .get_backup(user_id, backup_id)
            .ok_or(BackupError::NotFound)?;

        // The additional data is bound to the *requesting* account, not the
        // account recorded in the row. Binding the stored value would let a
        // record copied into another user's list decrypt for that user.
        let result = decrypt_data(
            &backup.encrypted_data,
            passphrase,
            user_id,
            &backup.id,
            &backup.salt,
            &backup.nonce,
        );

        let mut stats = self.stats.lock().map_err(|_| BackupError::StorageError)?;
        match &result {
            Ok(_) => stats.successful_restores += 1,
            Err(_) => stats.failed_backups += 1,
        }
        drop(stats);

        result
    }

    /// Confirm a backup decrypts and its plaintext matches the stored checksum.
    pub fn verify_backup(
        &self,
        user_id: &str,
        backup_id: &str,
        passphrase: &str,
    ) -> Result<bool, BackupError> {
        let backup = self
            .get_backup(user_id, backup_id)
            .ok_or(BackupError::NotFound)?;
        let data = self.restore_backup(user_id, backup_id, passphrase)?;
        Ok(calculate_checksum(&data) == backup.checksum)
    }

    pub fn delete_backup(&self, user_id: &str, backup_id: &str) -> Result<(), BackupError> {
        {
            let mut backups = self.backups.lock().map_err(|_| BackupError::StorageError)?;
            if let Some(list) = backups.get_mut(user_id) {
                list.retain(|b| b.id != backup_id);
            }
        }
        {
            let mut metadata = self.metadata.lock().map_err(|_| BackupError::StorageError)?;
            if let Some(list) = metadata.get_mut(user_id) {
                list.retain(|m| m.id != backup_id);
            }
        }
        Ok(())
    }

    pub fn configure(&self, user_id: &str, config: BackupConfig) {
        if let Ok(mut cfg) = self.config.lock() {
            cfg.insert(user_id.to_string(), config);
        }
    }

    pub fn get_config(&self, user_id: &str) -> Option<BackupConfig> {
        self.config.lock().ok().and_then(|c| c.get(user_id).cloned())
    }

    pub fn get_stats(&self) -> BackupStats {
        self.stats
            .lock()
            .map(|s| s.clone())
            .unwrap_or_default()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const PASSPHRASE: &str = "correct horse battery staple";

    #[test]
    fn backup_round_trips() {
        let store = CloudBackupStore::new();
        let secret = b"seed phrase material".to_vec();

        let backup = store
            .create_backup("user-1", &secret, "evm", PASSPHRASE)
            .unwrap();
        let restored = store
            .restore_backup("user-1", &backup.id, PASSPHRASE)
            .unwrap();

        assert_eq!(restored, secret);
    }

    #[test]
    fn stored_bytes_do_not_contain_the_plaintext() {
        let store = CloudBackupStore::new();
        let secret = b"SUPER_SECRET_SEED_PHRASE_VALUE".to_vec();

        let backup = store
            .create_backup("user-1", &secret, "evm", PASSPHRASE)
            .unwrap();

        // The whole point of the feature: the stored blob must not leak the
        // plaintext. The old implementation copied it verbatim.
        assert!(
            !backup
                .encrypted_data
                .windows(secret.len())
                .any(|w| w == secret.as_slice()),
            "ciphertext contains the plaintext"
        );
        assert_ne!(backup.encrypted_data, secret);
    }

    #[test]
    fn wrong_passphrase_is_rejected() {
        let store = CloudBackupStore::new();
        let backup = store
            .create_backup("user-1", b"seed", "evm", PASSPHRASE)
            .unwrap();

        let result = store.restore_backup("user-1", &backup.id, "wrong passphrase");
        assert_eq!(result, Err(BackupError::AuthenticationFailed));
    }

    #[test]
    fn short_passphrase_is_rejected() {
        let store = CloudBackupStore::new();
        let result = store.create_backup("user-1", b"seed", "evm", "short");
        assert!(matches!(result, Err(BackupError::InvalidPassphrase(_))));
    }

    #[test]
    fn tampered_ciphertext_is_detected() {
        let store = CloudBackupStore::new();
        let backup = store
            .create_backup("user-1", b"seed material here", "evm", PASSPHRASE)
            .unwrap();

        // Flip a bit in the ciphertext.
        {
            let mut backups = store.backups.lock().unwrap();
            let list = backups.get_mut("user-1").unwrap();
            list[0].encrypted_data[0] ^= 0x01;
        }

        let result = store.restore_backup("user-1", &backup.id, PASSPHRASE);
        assert_eq!(result, Err(BackupError::AuthenticationFailed));
    }

    #[test]
    fn backup_cannot_be_moved_to_another_user() {
        let store = CloudBackupStore::new();
        let backup = store
            .create_backup("user-1", b"seed", "evm", PASSPHRASE)
            .unwrap();

        // The user id is bound into the AEAD additional data, so a record
        // replayed under a different account fails to authenticate.
        {
            let mut backups = store.backups.lock().unwrap();
            let entry = backups.get_mut("user-1").unwrap().remove(0);
            backups.entry("user-2".to_string()).or_default().push(entry);
        }

        let result = store.restore_backup("user-2", &backup.id, PASSPHRASE);
        assert_eq!(result, Err(BackupError::AuthenticationFailed));
    }

    #[test]
    fn each_backup_uses_a_fresh_salt_and_nonce() {
        let store = CloudBackupStore::new();
        let first = store
            .create_backup("user-1", b"same plaintext", "evm", PASSPHRASE)
            .unwrap();
        let second = store
            .create_backup("user-1", b"same plaintext", "evm", PASSPHRASE)
            .unwrap();

        assert_ne!(first.salt, second.salt, "salt was reused");
        assert_ne!(first.nonce, second.nonce, "nonce was reused");
        // Identical plaintext under the same passphrase must not produce
        // identical ciphertext.
        assert_ne!(first.encrypted_data, second.encrypted_data);
    }

    #[test]
    fn verify_backup_confirms_integrity() {
        let store = CloudBackupStore::new();
        let backup = store
            .create_backup("user-1", b"seed material", "evm", PASSPHRASE)
            .unwrap();

        assert!(store
            .verify_backup("user-1", &backup.id, PASSPHRASE)
            .unwrap());
        assert!(store
            .verify_backup("user-1", &backup.id, "wrong passphrase")
            .is_err());
    }

    #[test]
    fn checksum_is_a_real_sha256() {
        let checksum = calculate_checksum(b"");
        // SHA-256 of the empty string.
        assert_eq!(
            checksum,
            "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
        );
        assert_eq!(checksum.len(), 64);
    }

    #[test]
    fn missing_backup_reports_not_found() {
        let store = CloudBackupStore::new();
        assert_eq!(
            store.restore_backup("user-1", "backup_missing", PASSPHRASE),
            Err(BackupError::NotFound)
        );
    }

    #[test]
    fn oversized_payload_is_rejected() {
        let store = CloudBackupStore::new();
        let huge = vec![0u8; MAX_BACKUP_BYTES + 1];
        let result = store.create_backup("user-1", &huge, "evm", PASSPHRASE);
        assert!(matches!(result, Err(BackupError::PayloadTooLarge(_))));
    }

    #[test]
    fn delete_removes_the_backup_and_its_metadata() {
        let store = CloudBackupStore::new();
        let backup = store
            .create_backup("user-1", b"seed", "evm", PASSPHRASE)
            .unwrap();

        assert_eq!(store.list_backups("user-1").len(), 1);
        store.delete_backup("user-1", &backup.id).unwrap();

        assert!(store.get_backup("user-1", &backup.id).is_none());
        assert!(store.list_backups("user-1").is_empty());
    }

    #[test]
    fn backups_are_isolated_per_user() {
        let store = CloudBackupStore::new();
        let alice = store
            .create_backup("alice", b"alice seed", "evm", PASSPHRASE)
            .unwrap();
        store
            .create_backup("bob", b"bob seed", "evm", PASSPHRASE)
            .unwrap();

        assert_eq!(store.list_backups("alice").len(), 1);
        assert_eq!(store.list_backups("bob").len(), 1);
        // Alice cannot reach Bob's backup by id.
        assert!(store.get_backup("bob", &alice.id).is_none());
    }

    #[test]
    fn empty_user_id_is_rejected() {
        let store = CloudBackupStore::new();
        assert!(matches!(
            store.create_backup("", b"seed", "evm", PASSPHRASE),
            Err(BackupError::InvalidConfig)
        ));
    }

    #[test]
    fn stats_track_backups_and_restores() {
        let store = CloudBackupStore::new();
        let backup = store
            .create_backup("user-1", b"seed", "evm", PASSPHRASE)
            .unwrap();
        store.restore_backup("user-1", &backup.id, PASSPHRASE).unwrap();
        store.restore_backup("user-1", &backup.id, "wrong").unwrap_err();

        let stats = store.get_stats();
        assert_eq!(stats.total_backups, 1);
        assert_eq!(stats.successful_restores, 1);
        assert_eq!(stats.failed_backups, 1);
        assert!(stats.last_backup.is_some());
    }

    #[test]
    fn config_round_trips() {
        let store = CloudBackupStore::new();
        store.configure(
            "user-1",
            BackupConfig {
                auto_backup: true,
                backup_interval_minutes: 60,
                max_backups: 5,
                encrypt_before_upload: true,
            },
        );

        let config = store.get_config("user-1").unwrap();
        assert!(config.auto_backup);
        assert_eq!(config.backup_interval_minutes, 60);
    }
}
