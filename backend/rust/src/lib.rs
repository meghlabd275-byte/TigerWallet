//! TigerWallet Security Library
//! 
//! Security-critical operations including:
//! - Key derivation (Argon2)
//! - Encryption (AES-256-GCM)
//! - Digital signatures (Ed25519)
//! - HMAC-based message authentication
//! - Secure random number generation

use aes_gcm::{
    aead::{Aead, KeyInit, OsRng},
    Aes256Gcm, Nonce,
};
use argon2::{
    password_hash::{rand_core::RngCore, PasswordHash, PasswordHasher, PasswordVerifier, SaltString},
    Argon2,
};
use ed25519_dalek::{Signature, Signer, SigningKey, Verifier, VerifyingKey};
use hmac::{Hmac, Mac};
use sha2::Sha256;

type HmacSha256 = Hmac<Sha256>;

/// Error type for security operations
#[derive(Debug)]
pub enum SecurityError {
    EncryptionError(String),
    DecryptionError(String),
    KeyDerivationError(String),
    SignatureError(String),
    InvalidKeyError(String),
}

impl std::fmt::Display for SecurityError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::EncryptionError(s) => write!(f, "Encryption error: {}", s),
            Self::DecryptionError(s) => write!(f, "Decryption error: {}", s),
            Self::KeyDerivationError(s) => write!(f, "Key derivation error: {}", s),
            Self::SignatureError(s) => write!(f, "Signature error: {}", s),
            Self::InvalidKeyError(s) => write!(f, "Invalid key error: {}", s),
        }
    }
}

impl std::error::Error for SecurityError {}

/// Encryption service for sensitive data
pub struct Encryption;

impl Encryption {
    /// Encrypt data using AES-256-GCM
    /// 
    /// # Arguments
    /// * `plaintext` - Data to encrypt
    /// * `key` - 32-byte encryption key
    /// 
    /// # Returns
    /// Base64-encoded ciphertext with nonce prepended
    pub fn encrypt(plaintext: &[u8], key: &[u8; 32]) -> Result<Vec<u8>, SecurityError> {
        if key.len() != 32 {
            return Err(SecurityError::InvalidKeyError("Key must be 32 bytes".into()));
        }

        let cipher = Aes256Gcm::new_from_slice(key)
            .map_err(|e| SecurityError::EncryptionError(e.to_string()))?;

        // Generate random nonce
        let mut nonce_bytes = [0u8; 12];
        OsRng.fill_bytes(&mut nonce_bytes);
        let nonce = Nonce::from_slice(&nonce_bytes);

        // Encrypt
        let ciphertext = cipher
            .encrypt(nonce, plaintext)
            .map_err(|e| SecurityError::EncryptionError(e.to_string()))?;

        // Prepend nonce to ciphertext
        let mut result = Vec::with_capacity(12 + ciphertext.len());
        result.extend_from_slice(&nonce_bytes);
        result.extend(ciphertext);

        Ok(result)
    }

    /// Decrypt data using AES-256-GCM
    /// 
    /// # Arguments
    /// * `ciphertext` - Base64-encoded ciphertext with nonce prepended
    /// * `key` - 32-byte decryption key
    /// 
    /// # Returns
    /// Decrypted plaintext
    pub fn decrypt(ciphertext: &[u8], key: &[u8; 32]) -> Result<Vec<u8>, SecurityError> {
        if key.len() != 32 {
            return Err(SecurityError::InvalidKeyError("Key must be 32 bytes".into()));
        }

        if ciphertext.len() < 12 {
            return Err(SecurityError::DecryptionError("Ciphertext too short".into()));
        }

        let cipher = Aes256Gcm::new_from_slice(key)
            .map_err(|e| SecurityError::DecryptionError(e.to_string()))?;

        let nonce = Nonce::from_slice(&ciphertext[..12]);
        let encrypted = &ciphertext[12..];

        cipher
            .decrypt(nonce, encrypted)
            .map_err(|e| SecurityError::DecryptionError(e.to_string()))
    }
}

/// Key derivation service
pub struct KeyDerivation;

impl KeyDerivation {
    /// Derive a key from a password using Argon2
    /// 
    /// # Arguments
    /// * `password` - User password
    /// * `salt` - Optional salt (generated if None)
    /// 
    /// # Returns
    /// Base64-encoded hashed password
    pub fn derive_key(password: &str, salt: Option<&[u8]>) -> Result<String, SecurityError> {
        let salt_string = match salt {
            Some(s) => SaltString::encode_b64(s)
                .map_err(|e| SecurityError::KeyDerivationError(e.to_string()))?,
            None => SaltString::generate(&mut OsRng),
        };

        let argon2 = Argon2::default();
        let password_hash = argon2
            .hash_password(password.as_bytes(), &salt_string)
            .map_err(|e| SecurityError::KeyDerivationError(e.to_string()))?
            .to_string();

        Ok(password_hash)
    }

    /// Verify a password against a stored hash
    pub fn verify_password(password: &str, hash: &str) -> Result<bool, SecurityError> {
        let parsed_hash = PasswordHash::new(hash)
            .map_err(|e| SecurityError::KeyDerivationError(e.to_string()))?;

        Ok(Argon2::default()
            .verify_password(password.as_bytes(), &parsed_hash)
            .is_ok())
    }

    /// Generate a secure random key
    pub fn generate_key() -> [u8; 32] {
        let mut key = [0u8; 32];
        OsRng.fill_bytes(&mut key);
        key
    }
}

/// Digital signature service
pub struct Signer as Sig;

impl Sig {
    /// Generate a new Ed25519 key pair
    pub fn generate_keypair() -> (SigningKey, VerifyingKey) {
        let signing_key = SigningKey::generate(&mut OsRng);
        let verifying_key = signing_key.verifying_key();
        (signing_key, verifying_key)
    }

    /// Sign a message
    pub fn sign(message: &[u8], signing_key: &SigningKey) -> Signature {
        signing_key.sign(message)
    }

    /// Verify a signature
    pub fn verify(message: &[u8], signature: &Signature, verifying_key: &VerifyingKey) -> bool {
        verifying_key.verify(message, signature).is_ok()
    }
}

/// HMAC service for message authentication
pub struct HmacService;

impl HmacService {
    /// Compute HMAC-SHA256
    pub fn compute(key: &[u8], message: &[u8]) -> Vec<u8> {
        let mut mac = HmacSha256::new_from_slice(key).expect("HMAC can take key of any size");
        mac.update(message);
        mac.finalize().into_bytes().to_vec()
    }

    /// Verify HMAC
    pub fn verify(key: &[u8], message: &[u8], expected: &[u8]) -> bool {
        let computed = Self::compute(key, message);
        constant_time_eq::constant_time_eq(&computed, expected)
    }
}

/// Utility functions
pub mod util {
    use base64::{engine::general_purpose::STANDARD as BASE64, Engine};

    /// Encode bytes to Base64
    pub fn encode_base64(data: &[u8]) -> String {
        BASE64.encode(data)
    }

    /// Decode Base64 to bytes
    pub fn decode_base64(data: &str) -> Result<Vec<u8>, SecurityError> {
        BASE64
            .decode(data)
            .map_err(|e| SecurityError::DecryptionError(e.to_string()))
    }

    /// Constant-time string comparison (for secrets)
    pub fn secure_compare(a: &[u8], b: &[u8]) -> bool {
        constant_time_eq::constant_time_eq(a, b)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_encryption_roundtrip() {
        let key = KeyDerivation::generate_key();
        let plaintext = b"Hello, TigerWallet!";
        
        let ciphertext = Encryption::encrypt(plaintext, &key).unwrap();
        let decrypted = Encryption::decrypt(&ciphertext, &key).unwrap();
        
        assert_eq!(plaintext, decrypted.as_slice());
    }

    #[test]
    fn test_key_derivation() {
        let password = "secure_password_123";
        let hash = KeyDerivation::derive_key(password, None).unwrap();
        
        assert!(KeyDerivation::verify_password(password, &hash).unwrap());
        assert!(!KeyDerivation::verify_password("wrong_password", &hash).unwrap());
    }

    #[test]
    fn test_signing() {
        let (signing_key, verifying_key) = Sig::generate_keypair();
        let message = b"Transaction: 100 USDT to 0x123...";
        
        let signature = Sig::sign(message, &signing_key);
        assert!(Sig::verify(message, &signature, &verifying_key));
    }

    #[test]
    fn test_hmac() {
        let key = b"secret_key";
        let message = b"important message";
        
        let hmac = HmacService::compute(key, message);
        assert!(HmacService::verify(key, message, &hmac));
    }
}
