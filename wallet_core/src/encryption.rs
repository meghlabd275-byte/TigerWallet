// ============================================================================
// TIGERWALLET ENCRYPTION MODULE
// AES-256-GCM encryption for secure key storage
// ============================================================================

use aes_gcm::{
    aead::{Aead, KeyInit, OsRng},
    Aes256Gcm, Nonce,
};
use pbkdf2::pbkdf2_hmac;
use sha2::{Sha256, Sha512, Digest};
use zeroize::Zeroize;
use rand::RngCore;

/// Number of PBKDF2 iterations
const PBKDF2_ITERATIONS: u32 = 100_000;
const SALT_LENGTH: usize = 32;
const NONCE_LENGTH: usize = 12;
const KEY_LENGTH: usize = 32;

/// Encrypted data structure
#[derive(Debug, Clone)]
pub struct EncryptedData {
    pub salt: Vec<u8>,
    pub nonce: Vec<u8>,
    pub ciphertext: Vec<u8>,
}

impl EncryptedData {
    pub fn new(salt: Vec<u8>, nonce: Vec<u8>, ciphertext: Vec<u8>) -> Self {
        Self { salt, nonce, ciphertext }
    }

    pub fn to_bytes(&self) -> Vec<u8> {
        let mut result = Vec::new();
        result.extend_from_slice(&self.salt);
        result.extend_from_slice(&self.nonce);
        result.extend_from_slice(&self.ciphertext);
        result
    }

    pub fn from_bytes(data: &[u8]) -> Result<Self, EncryptionError> {
        if data.len() < SALT_LENGTH + NONCE_LENGTH {
            return Err(EncryptionError::InvalidData);
        }
        
        let salt = data[..SALT_LENGTH].to_vec();
        let nonce = data[SALT_LENGTH..SALT_LENGTH + NONCE_LENGTH].to_vec();
        let ciphertext = data[SALT_LENGTH + NONCE_LENGTH..].to_vec();
        
        Ok(Self { salt, nonce, ciphertext })
    }
}

/// Encrypt data with password
pub fn encrypt_with_password(data: &[u8], password: &str) -> Result<EncryptedData, EncryptionError> {
    // Generate random salt
    let mut salt = [0u8; SALT_LENGTH];
    use std::mem::MaybeUninit;
    let mut rng = [0u8; SALT_LENGTH];
    getrandom(&mut rng)?;
    
    // Derive key using PBKDF2
    let mut key = derive_key(password, &rng)?;
    
    // Generate random nonce
    let mut nonce_bytes = [0u8; NONCE_LENGTH];
    getrandom(&mut nonce_bytes)?;
    
    // Encrypt
    let cipher = Aes256Gcm::new_from_slice(&key)
        .map_err(|_| EncryptionError::KeyDerivationFailed)?;
    
    let nonce = Nonce::from_slice(&nonce_bytes);
    let ciphertext = cipher.encrypt(nonce, data)
        .map_err(|_| EncryptionError::EncryptionFailed)?;
    
    // Zeroize key from memory
    key.zeroize();
    
    Ok(EncryptedData::new(rng.to_vec(), nonce_bytes.to_vec(), ciphertext))
}

/// Decrypt data with password
pub fn decrypt_with_password(encrypted: &EncryptedData, password: &str) -> Result<Vec<u8>, EncryptionError> {
    // Derive key using stored salt
    let key = derive_key(password, &encrypted.salt)?;
    
    // Decrypt
    let cipher = Aes256Gcm::new_from_slice(&key)
        .map_err(|_| EncryptionError::KeyDerivationFailed)?;
    
    let nonce = Nonce::from_slice(&encrypted.nonce);
    let plaintext = cipher.decrypt(nonce, encrypted.ciphertext.as_ref())
        .map_err(|_| EncryptionError::DecryptionFailed)?;
    
    // Zeroize key from memory
    let mut key = key;
    key.zeroize();
    
    Ok(plaintext)
}

/// Derive encryption key from password
fn derive_key(password: &str, salt: &[u8]) -> Result<[u8; KEY_LENGTH], EncryptionError> {
    let mut key = [0u8; KEY_LENGTH];
    pbkdf2_hmac::<Sha256>(
        password.as_bytes(),
        salt,
        PBKDF2_ITERATIONS,
        &mut key,
    );
    Ok(key)
}

/// Generate random bytes
fn getrandom(dest: &mut [u8]) -> Result<(), EncryptionError> {
    OsRng.fill_bytes(dest);
    Ok(())
}

/// Encrypt mnemonic phrase
pub fn encrypt_mnemonic(mnemonic: &str, password: &str) -> Result<EncryptedData, EncryptionError> {
    encrypt_with_password(mnemonic.as_bytes(), password)
}

/// Decrypt mnemonic phrase
pub fn decrypt_mnemonic(encrypted: &EncryptedData, password: &str) -> Result<String, EncryptionError> {
    let plaintext = decrypt_with_password(encrypted, password)?;
    String::from_utf8(plaintext)
        .map_err(|_| EncryptionError::InvalidData)
}

/// Encrypt private key
pub fn encrypt_private_key(private_key: &[u8], password: &str) -> Result<EncryptedData, EncryptionError> {
    encrypt_with_password(private_key, password)
}

/// Decrypt private key
pub fn decrypt_private_key(encrypted: &EncryptedData, password: &str) -> Result<Vec<u8>, EncryptionError> {
    decrypt_with_password(encrypted, password)
}

/// Wallet keystore (encrypted JSON)
#[derive(Debug, Clone)]
pub struct Keystore {
    pub version: u32,
    pub id: String,
    pub address: String,
    pub crypto: KeystoreCrypto,
}

#[derive(Debug, Clone)]
pub struct KeystoreCrypto {
    pub cipher: String,
    pub cipherparams: CipherParams,
    pub ciphertext: String,
    pub kdf: String,
    pub kdfparams: KdfParams,
    pub mac: String,
}

#[derive(Debug, Clone)]
pub struct CipherParams {
    pub iv: String,
}

#[derive(Debug, Clone)]
pub struct KdfParams {
    pub salt: String,
    pub dklen: u32,
    pub c: u32,
    pub prf: String,
}

/// Create keystore from private key
pub fn create_keystore(private_key: &[u8], address: &str, password: &str) -> Result<Keystore, EncryptionError> {
    let encrypted = encrypt_with_password(private_key, password)?;
    let encrypted_hex = hex::encode(encrypted.to_bytes());
    
    // Derive mac for verification
    let mut mac_input = encrypted.ciphertext.clone();
    mac_input.extend_from_slice(private_key);
    let mut hasher = Sha256::new();
    hasher.update(&mac_input);
    let mac = hex::encode(hasher.finalize());
    
    // Generate UUID
    let id = generate_uuid();
    
    Ok(Keystore {
        version: 3,
        id,
        address: address.to_string(),
        crypto: KeystoreCrypto {
            cipher: "aes-256-gcm".to_string(),
            cipherparams: CipherParams {
                iv: hex::encode(&encrypted.nonce),
            },
            ciphertext: encrypted_hex,
            kdf: "pbkdf2".to_string(),
            kdfparams: KdfParams {
                salt: hex::encode(&encrypted.salt),
                dklen: 32,
                c: PBKDF2_ITERATIONS,
                prf: "hmac-sha256".to_string(),
            },
            mac,
        },
    })
}

/// Decrypt keystore
pub fn decrypt_keystore(keystore: &Keystore, password: &str) -> Result<Vec<u8>, EncryptionError> {
    let salt = hex::decode(&keystore.crypto.kdfparams.salt)
        .map_err(|_| EncryptionError::InvalidData)?;
    let nonce = hex::decode(&keystore.crypto.cipherparams.iv)
        .map_err(|_| EncryptionError::InvalidData)?;
    let ciphertext = hex::decode(&keystore.crypto.ciphertext)
        .map_err(|_| EncryptionError::InvalidData)?;
    
    let encrypted = EncryptedData::new(salt, nonce, ciphertext);
    decrypt_with_password(&encrypted, password)
}

/// Verify keystore password
pub fn verify_keystore(keystore: &Keystore, password: &str) -> bool {
    match decrypt_keystore(keystore, password) {
        Ok(_) => {
            // Verify MAC
            let decrypted = decrypt_keystore(keystore, password).unwrap();
            let mut mac_input = hex::decode(&keystore.crypto.ciphertext).unwrap();
            mac_input.extend_from_slice(&decrypted);
            let mut hasher = Sha256::new();
            hasher.update(&mac_input);
            let mac = hex::encode(hasher.finalize());
            mac == keystore.crypto.mac
        }
        Err(_) => false,
    }
}

/// Generate UUID v4
fn generate_uuid() -> String {
    let mut bytes = [0u8; 16];
    getrandom(&mut bytes).unwrap_or_default();
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    
    format!(
        "{:02x}{:02x}{:02x}{:02x}-{:02x}{:02x}-{:02x}{:02x}-{:02x}{:02x}-{:02x}{:02x}{:02x}{:02x}{:02x}{:02x}",
        bytes[0], bytes[1], bytes[2], bytes[3],
        bytes[4], bytes[5],
        bytes[6], bytes[7],
        bytes[8], bytes[9],
        bytes[10], bytes[11], bytes[12], bytes[13], bytes[14], bytes[15]
    )
}

/// Encryption errors
#[derive(Debug, Clone)]
pub enum EncryptionError {
    EncryptionFailed,
    DecryptionFailed,
    KeyDerivationFailed,
    InvalidData,
}

impl std::fmt::Display for EncryptionError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            EncryptionError::EncryptionFailed => write!(f, "Encryption failed"),
            EncryptionError::DecryptionFailed => write!(f, "Decryption failed"),
            EncryptionError::KeyDerivationFailed => write!(f, "Key derivation failed"),
            EncryptionError::InvalidData => write!(f, "Invalid data"),
        }
    }
}

impl std::error::Error for EncryptionError {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_encrypt_decrypt() {
        let data = b"Hello, TigerWallet!";
        let password = "secure_password_123";
        
        let encrypted = encrypt_with_password(data, password).unwrap();
        let decrypted = decrypt_with_password(&encrypted, password).unwrap();
        
        assert_eq!(data.to_vec(), decrypted);
    }

    #[test]
    fn test_mnemonic_encrypt() {
        let mnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";
        let password = "wallet_password";
        
        let encrypted = encrypt_mnemonic(mnemonic, password).unwrap();
        let decrypted = decrypt_mnemonic(&encrypted, password).unwrap();
        
        assert_eq!(mnemonic, decrypted);
    }
}