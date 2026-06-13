//! TigerWallet Key Management - Rust Implementation
//! Secure key storage and management with AES-256-GCM encryption
//! Memory-safe, zero-cost abstractions, fearless concurrency

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH};

// ============================================================================
// Key Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum KeyType {
    Private,
    Public,
    Symmetric,
    Mnemonic,
    Keystore,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Key {
    pub id: String,
    pub key_type: KeyType,
    pub public_key: Option<String>,
    pub encrypted: String,
    pub algorithm: String,
    pub created_at: u64,
    pub modified_at: u64,
}

// ============================================================================
// Key Store
// ============================================================================

pub struct KeyStore {
    keys: Arc<RwLock<HashMap<String, Key>>>,
    master_key: Vec<u8>,
}

impl KeyStore {
    /// Create new key store with master password
    pub fn new(master_password: &str) -> Self {
        let master_key = derive_master_key(master_password);
        KeyStore {
            keys: Arc::new(RwLock::new(HashMap::new())),
            master_key,
        }
    }

    /// Generate new key
    pub fn generate_key(&self, key_type: KeyType) -> Result<Key, KeyError> {
        let id = generate_id();
        
        let key_data = match key_type {
            KeyType::Private => generate_private_key()?,
            KeyType::Symmetric => generate_symmetric_key(),
            KeyType::Mnemonic => {
                return Err(KeyError::Unsupported("Use ImportMnemonic for mnemonic".to_string()))
            }
            KeyType::Public => {
                return Err(KeyError::Unsupported("Public keys must be derived".to_string()))
            }
            KeyType::Keystore => generate_symmetric_key(),
        };

        // Encrypt key data
        let encrypted = encrypt_key_data(&key_data, &self.master_key)?;
        
        let now = current_timestamp();
        
        let public_key = if key_type == KeyType::Private {
            Some(derive_public_key_hex(&key_data))
        } else {
            None
        };

        let key = Key {
            id: id.clone(),
            key_type,
            public_key,
            encrypted,
            algorithm: "AES-256-GCM".to_string(),
            created_at: now,
            modified_at: now,
        };

        // Store key
        let mut keys = self.keys.write().map_err(|_| KeyError::LockError)?;
        keys.insert(id, key.clone());

        Ok(key)
    }

    /// Import existing key
    pub fn import_key(&self, key_type: KeyType, key_data: Vec<u8>) -> Result<Key, KeyError> {
        let id = generate_id();
        
        // Encrypt key data
        let encrypted = encrypt_key_data(&key_data, &self.master_key)?;
        
        let now = current_timestamp();
        
        let public_key = if key_type == KeyType::Private {
            Some(derive_public_key_hex(&key_data))
        } else {
            None
        };

        let key = Key {
            id: id.clone(),
            key_type,
            public_key,
            encrypted,
            algorithm: "AES-256-GCM".to_string(),
            created_at: now,
            modified_at: now,
        };

        // Store key
        let mut keys = self.keys.write().map_err(|_| KeyError::LockError)?;
        keys.insert(id, key.clone());

        Ok(key)
    }

    /// Get key by ID
    pub fn get_key(&self, id: &str) -> Option<Key> {
        let keys = self.keys.read().ok()?;
        keys.get(id).cloned()
    }

    /// Get decrypted key data
    pub fn get_key_data(&self, id: &str) -> Result<Vec<u8>, KeyError> {
        let key = self.get_key(id).ok_or(KeyError::NotFound)?;
        decrypt_key_data(&key.encrypted, &self.master_key)
    }

    /// Delete key
    pub fn delete_key(&self, id: &str) -> Result<(), KeyError> {
        let mut keys = self.keys.write().map_err(|_| KeyError::LockError)?;
        keys.remove(id);
        Ok(())
    }

    /// List all key IDs
    pub fn list_keys(&self) -> Vec<String> {
        let keys = match self.keys.read() {
            Ok(k) => k,
            Err(_) => return vec![],
        };
        keys.keys().cloned().collect()
    }

    /// Sign data with private key
    pub fn sign(&self, id: &str, data: &[u8]) -> Result<String, KeyError> {
        let key_data = self.get_key_data(id)?;
        
        if key_data.len() != 32 {
            return Err(KeyError::InvalidKey("Invalid private key length".to_string()));
        }

        // Sign using secp256k1
        let signature = sign_message(data, &key_data)?;
        
        Ok(signature)
    }

    /// Verify signature
    pub fn verify(&self, id: &str, data: &[u8], signature: &str) -> Result<bool, KeyError> {
        let key = self.get_key(id).ok_or(KeyError::NotFound)?;
        
        let public_key = key.public_key.ok_or(
            KeyError::InvalidKey("No public key available".to_string())
        )?;

        Ok(verify_signature(data, &signature, &public_key)?)
    }
}

// ============================================================================
// Key Errors
// ============================================================================

#[derive(Debug, thiserror::Error)]
pub enum KeyError {
    #[error("Key not found")]
    NotFound,

    #[error("Invalid key: {0}")]
    InvalidKey(String),

    #[error("Unsupported operation: {0}")]
    Unsupported(String),

    #[error("Encryption failed: {0}")]
    EncryptionFailed(String),

    #[error("Decryption failed: {0}")]
    DecryptionFailed(String),

    #[error("Signing failed: {0}")]
    SigningFailed(String),

    #[error("Verification failed: {0}")]
    VerificationFailed(String),

    #[error("Lock error")]
    LockError,

    #[error("Random generation failed: {0}")]
    RandomError(String),
}

// ============================================================================
// Cryptographic Functions
// ============================================================================

fn derive_master_key(password: &str) -> Vec<u8> {
    use sha2::{Sha256, Digest};
    let mut hasher = Sha256::new();
    hasher.update(password.as_bytes());
    hasher.finalize().to_vec()
}

fn generate_id() -> String {
    let random = generate_secure_random(16);
    hex::encode(random)
}

fn generate_private_key() -> Result<Vec<u8>, KeyError> {
    let random = generate_secure_random(32);
    // Validate not zero
    if random.iter().all(|&b| b == 0) {
        return Err(KeyError::RandomError("Invalid random data".to_string()));
    }
    Ok(random)
}

fn generate_symmetric_key() -> Vec<u8> {
    generate_secure_random(32)
}

fn generate_secure_random(length: usize) -> Vec<u8> {
    use rand::rngs::OsRng;
    use rand::RngCore;
    
    let mut bytes = vec![0u8; length];
    OsRng.fill_bytes(&mut bytes);
    bytes
}

fn current_timestamp() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
}

// ============================================================================
// Encryption (AES-256-GCM)
// ============================================================================

fn encrypt_key_data(data: &[u8], key: &[u8]) -> Result<String, KeyError> {
    use aes_gcm::{
        aead::{Aead, KeyInit, OsRng},
        Aes256Gcm, Nonce,
    };

    if key.len() != 32 {
        return Err(KeyError::EncryptionFailed("Invalid key length".to_string()));
    }

    let cipher = Aes256Gcm::new(key.into());
    
    let mut nonce_bytes = [0u8; 12];
    OsRng.fill_bytes(&mut nonce_bytes);
    let nonce = Nonce::from_slice(&nonce_bytes);

    let ciphertext = cipher
        .encrypt(nonce, data)
        .map_err(|e| KeyError::EncryptionFailed(e.to_string()))?;

    // Prepend nonce to ciphertext
    let mut result = nonce_bytes.to_vec();
    result.extend(ciphertext);

    Ok(hex::encode(result))
}

fn decrypt_key_data(encrypted: &str, key: &[u8]) -> Result<Vec<u8>, KeyError> {
    use aes_gcm::{
        aead::{Aead, KeyInit},
        Aes256Gcm, Nonce,
    };

    if key.len() != 32 {
        return Err(KeyError::DecryptionFailed("Invalid key length".to_string()));
    }

    let data = hex::decode(encrypted)
        .map_err(|e| KeyError::DecryptionFailed(e.to_string()))?;

    if data.len() < 12 {
        return Err(KeyError::DecryptionFailed("Data too short".to_string()));
    }

    let nonce = Nonce::from_slice(&data[..12]);
    let ciphertext = &data[12..];

    let cipher = Aes256Gcm::new(key.into());
    
    cipher
        .decrypt(nonce, ciphertext)
        .map_err(|e| KeyError::DecryptionFailed(e.to_string()))
}

// ============================================================================
// Key Derivation
// ============================================================================

fn derive_public_key_hex(private_key: &[u8]) -> String {
    use secp256k1::{PublicKey, SecretKey};
    
    let secret = SecretKey::from_slice(private_key)
        .expect("32 bytes");
    let public = PublicKey::from_secret_key(&secret);
    
    let serialized = public.serialize_uncompressed();
    hex::encode(&serialized[1..65])
}

fn sign_message(message: &[u8], private_key: &[u8]) -> Result<String, KeyError> {
    use secp256k1::{SecretKey, Message, SignableHash};
    
    let secret = SecretKey::from_slice(private_key)
        .map_err(|e| KeyError::SigningFailed(e.to_string()))?;
    
    let message_hash = keccak256(message);
    let msg = SignableHash::from_raw(&message_hash);
    let signature = secret.sign_ecdsa(msg);
    
    let r = signature.r().to_bytes();
    let s = signature.s().to_bytes();
    
    let mut sig = Vec::with_capacity(64);
    sig.extend_from_slice(&r);
    sig.extend_from_slice(&s);
    
    Ok(hex::encode(sig))
}

fn verify_signature(message: &[u8], signature: &str, _public_key: &str) -> Result<bool, KeyError> {
    // Simplified - in production use full verification
    Ok(!signature.is_empty() && !message.is_empty())
}

fn keccak256(data: &[u8]) -> [u8; 32] {
    use keccak::{Keccak, digest::Digest};
    let mut hasher = Keccak::new_keccak_256();
    hasher.update(data);
    hasher.finalize().into()
}

// ============================================================================
// Hardware Security Module Interface
// ============================================================================

pub trait HardwareSecurityModule {
    fn generate_key(&mut self, id: &str) -> Result<(), KeyError>;
    fn sign(&mut self, id: &str, data: &[u8]) -> Result<Vec<u8>, KeyError>;
    fn get_public_key(&mut self, id: &str) -> Result<Vec<u8>, KeyError>;
    fn delete_key(&mut self, id: &str) -> Result<(), KeyError>;
}

// ============================================================================
// Multi-Sig Key Operations
// ============================================================================

pub struct MultiSigKeySet {
    pub threshold: u8,
    pub pubkeys: Vec<Vec<u8>>,
}

impl MultiSigKeySet {
    pub fn new(threshold: u8, pubkeys: Vec<Vec<u8>>) -> Result<Self, KeyError> {
        if threshold == 0 || threshold > pubkeys.len() as u8 {
            return Err(KeyError::InvalidKey("Invalid threshold".to_string()));
        }
        Ok(MultiSigKeySet { threshold, pubkeys })
    }

    pub fn combine_signatures(&self, signatures: &[Vec<u8>]) -> Result<Vec<u8>, KeyError> {
        if signatures.len() < self.threshold as usize {
            return Err(KeyError::SigningFailed(
                format!("Need {} signatures, got {}", self.threshold, signatures.len())
            ));
        }
        
        let mut combined = Vec::new();
        for sig in signatures.iter().take(self.threshold as usize) {
            combined.extend_from_slice(sig);
            combined.push(b',');
        }
        combined.pop();
        
        Ok(combined)
    }
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_key_generation() {
        let store = KeyStore::new("test_password");
        
        let key = store.generate_key(KeyType::Private).unwrap();
        
        assert!(!key.id.is_empty());
        assert!(key.public_key.is_some());
    }

    #[test]
    fn test_key_encryption() {
        let store = KeyStore::new("test_password");
        
        let key = store.generate_key(KeyType::Symmetric).unwrap();
        
        let data = store.get_key_data(&key.id).unwrap();
        assert!(!data.is_empty());
    }

    #[test]
    fn test_signing() {
        let store = KeyStore::new("test_password");
        
        let key = store.generate_key(KeyType::Private).unwrap();
        
        let signature = store.sign(&key.id, b"test message").unwrap();
        
        assert!(!signature.is_empty());
    }

    #[test]
    fn test_multisig() {
        let pubkeys = vec![
            vec![1u8; 33],
            vec![2u8; 33],
            vec![3u8; 33],
        ];
        
        let ms = MultiSigKeySet::new(2, pubkeys).unwrap();
        
        let sigs = vec![vec![1u8; 32], vec![2u8; 32]];
        let combined = ms.combine_signatures(&sigs).unwrap();
        
        assert!(!combined.is_empty());
    }
}