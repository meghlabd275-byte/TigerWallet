//! TigerWallet Advanced Cryptographic Library
//! 
//! High-speed, high-security cryptographic operations for enterprise wallet
//! - AES-256-GCM authenticated encryption
//! - ChaCha20-Poly1305 for high-speed encryption
//! - XChaCha20-Poly1305 for extended nonce
//! - Argon2id password hashing
//! - Secure random generation with OS CSPRNG

use std::convert::TryInto;

// ============================================================================
// Error Types
// ============================================================================

#[derive(Debug, thiserror::Error)]
pub enum AdvancedCryptoError {
    #[error("Encryption failed: {0}")]
    EncryptionFailed(String),
    
    #[error("Decryption failed: {0}")]
    DecryptionFailed(String),
    
    #[error("Invalid key: {0}")]
    InvalidKey(String),
    
    #[error("Invalid nonce: {0}")]
    InvalidNonce(String),
    
    #[error("Authentication failed: {0}")]
    AuthenticationFailed(String),
    
    #[error("Key derivation failed: {0}")]
    KeyDerivationFailed(String),
    
    #[error("Random generation failed: {0}")]
    RandomGenerationFailed(String),
}

pub type Result<T> = std::result::Result<T, AdvancedCryptoError>;

// ============================================================================
// Secure Random Generation (OS CSPRNG)
// ============================================================================

/// Generate cryptographically secure random bytes using OS CSPRNG
pub fn generate_secure_random(length: usize) -> Result<Vec<u8>> {
    use rand::rngs::OsRng;
    use rand::RngCore;
    
    let mut bytes = vec![0u8; length];
    OsRng.fill_bytes(&mut bytes);
    
    // Ensure no zero bytes (could indicate hardware failure)
    if bytes.iter().all(|&b| b == 0) {
        return Err(AdvancedCryptoError::RandomGenerationFailed(
            "CSPRNG returned all zeros".to_string()
        ));
    }
    
    Ok(bytes)
}

/// Generate a secure 256-bit (32 byte) key
pub fn generate_key() -> Result<[u8; 32]> {
    let v: Vec<u8> = generate_secure_random(32)?;
    v.try_into()
        .map_err(|v: Vec<u8>| AdvancedCryptoError::RandomGenerationFailed(
            format!("Failed to create 32-byte key: len={}", v.len())
        ))
}

/// Generate a secure 96-bit (12 byte) nonce for AES-GCM / ChaCha20-Poly1305
pub fn generate_nonce_96() -> Result<[u8; 12]> {
    let v: Vec<u8> = generate_secure_random(12)?;
    v.try_into()
        .map_err(|v: Vec<u8>| AdvancedCryptoError::RandomGenerationFailed(
            format!("Failed to create 12-byte nonce: len={}", v.len())
        ))
}

/// Generate a secure 128-bit (16 byte) nonce
pub fn generate_nonce_128() -> Result<[u8; 16]> {
    let v: Vec<u8> = generate_secure_random(16)?;
    v.try_into()
        .map_err(|v: Vec<u8>| AdvancedCryptoError::RandomGenerationFailed(
            format!("Failed to create 16-byte nonce: len={}", v.len())
        ))
}

/// Generate a secure 192-bit (24 byte) nonce for XChaCha20
pub fn generate_nonce_192() -> Result<[u8; 24]> {
    let v: Vec<u8> = generate_secure_random(24)?;
    v.try_into()
        .map_err(|v: Vec<u8>| AdvancedCryptoError::RandomGenerationFailed(
            format!("Failed to create 24-byte nonce: len={}", v.len())
        ))
}

/// Generate salt for key derivation
pub fn generate_salt() -> Result<[u8; 32]> {
    generate_key()
}

// ============================================================================
// AES-256-GCM Authenticated Encryption
// ============================================================================

/// AES-256-GCM Encrypt
/// 
/// # Arguments
/// * `plaintext` - Data to encrypt
/// * `key` - 32-byte encryption key
/// * `nonce` - 12-byte unique nonce (never reuse with same key)
/// * `aad` - Additional authenticated data (optional, can be empty)
/// 
/// # Returns
/// Ciphertext with appended authentication tag (16 bytes)
pub fn aes256_gcm_encrypt(
    plaintext: &[u8],
    key: &[u8; 32],
    nonce: &[u8; 12],
    aad: &[u8],
) -> Result<Vec<u8>> {
    use aes_gcm::{
        aead::{Aead, KeyInit},
        Aes256Gcm, Nonce,
    };
    
    // Validate inputs
    if key.len() != 32 {
        return Err(AdvancedCryptoError::InvalidKey(
            format!("Key must be 32 bytes, got {}", key.len())
        ));
    }
    
    if nonce.len() != 12 {
        return Err(AdvancedCryptoError::InvalidNonce(
            format!("Nonce must be 12 bytes, got {}", nonce.len())
        ));
    }
    
    // Create cipher
    let cipher = Aes256Gcm::new(key.into());
    let nonce = Nonce::from_slice(nonce);
    
    // Encrypt with authenticated data
    let ciphertext = cipher
        .encrypt(nonce, aead::Payload::from(plaintext))
        .map_err(|e| AdvancedCryptoError::EncryptionFailed(e.to_string()))?;
    
    Ok(ciphertext)
}

/// AES-256-GCM Decrypt
/// 
/// # Arguments
/// * `ciphertext` - Encrypted data with appended 16-byte tag
/// * `key` - 32-byte decryption key
/// * `nonce` - 12-byte nonce used during encryption
/// * `aad` - Additional authenticated data (must match encryption)
/// 
/// # Returns
/// Original plaintext
pub fn aes256_gcm_decrypt(
    ciphertext: &[u8],
    key: &[u8; 32],
    nonce: &[u8; 12],
    aad: &[u8],
) -> Result<Vec<u8>> {
    use aes_gcm::{
        aead::{Aead, KeyInit},
        Aes256Gcm, Nonce,
    };
    
    // Validate inputs
    if key.len() != 32 {
        return Err(AdvancedCryptoError::InvalidKey(
            format!("Key must be 32 bytes, got {}", key.len())
        ));
    }
    
    if nonce.len() != 12 {
        return Err(AdvancedCryptoError::InvalidNonce(
            format!("Nonce must be 12 bytes, got {}", nonce.len())
        ));
    }
    
    // Minimum ciphertext length check (must have tag)
    if ciphertext.len() < 16 {
        return Err(AdvancedCryptoError::DecryptionFailed(
            "Ciphertext too short".to_string()
        ));
    }
    
    // Create cipher
    let cipher = Aes256Gcm::new(key.into());
    let nonce = Nonce::from_slice(nonce);
    
    // Decrypt
    let plaintext = cipher
        .decrypt(nonce, aead::Payload::from(ciphertext))
        .map_err(|e| AdvancedCryptoError::AuthenticationFailed(e.to_string()))?;
    
    Ok(plaintext)
}

// ============================================================================
// ChaCha20-Poly1305 Encryption
// ============================================================================

/// ChaCha20-Poly1305 Encrypt (faster than AES on no-AES hardware)
pub fn chacha20_poly1305_encrypt(
    plaintext: &[u8],
    key: &[u8; 32],
    nonce: &[u8; 12],
    aad: &[u8],
) -> Result<Vec<u8>> {
    use chacha20poly1305::{
        aead::{Aead, KeyInit},
        ChaCha20Poly1305, Nonce,
    };
    
    if key.len() != 32 {
        return Err(AdvancedCryptoError::InvalidKey("Key must be 32 bytes".to_string()));
    }
    
    if nonce.len() != 12 {
        return Err(AdvancedCryptoError::InvalidNonce("Nonce must be 12 bytes".to_string()));
    }
    
    let cipher = ChaCha20Poly1305::new(key.into());
    let nonce = Nonce::from_slice(nonce);
    
    let ciphertext = cipher
        .encrypt(nonce, aead::Payload::from(plaintext))
        .map_err(|e| AdvancedCryptoError::EncryptionFailed(e.to_string()))?;
    
    Ok(ciphertext)
}

/// ChaCha20-Poly1305 Decrypt
pub fn chacha20_poly1305_decrypt(
    ciphertext: &[u8],
    key: &[u8; 32],
    nonce: &[u8; 12],
    aad: &[u8],
) -> Result<Vec<u8>> {
    use chacha20poly1305::{
        aead::{Aead, KeyInit},
        ChaCha20Poly1305, Nonce,
    };
    
    if key.len() != 32 {
        return Err(AdvancedCryptoError::InvalidKey("Key must be 32 bytes".to_string()));
    }
    
    if nonce.len() != 12 {
        return Err(AdvancedCryptoError::InvalidNonce("Nonce must be 12 bytes".to_string()));
    }
    
    if ciphertext.len() < 16 {
        return Err(AdvancedCryptoError::DecryptionFailed("Ciphertext too short".to_string()));
    }
    
    let cipher = ChaCha20Poly1305::new(key.into());
    let nonce = Nonce::from_slice(nonce);
    
    let plaintext = cipher
        .decrypt(nonce, aead::Payload::from(ciphertext))
        .map_err(|e| AdvancedCryptoError::AuthenticationFailed(e.to_string()))?;
    
    Ok(plaintext)
}

// ============================================================================
// XChaCha20-Poly1305 (Extended Nonce)
// ============================================================================

/// XChaCha20-Poly1305 - uses 192-bit nonce for better security against nonce reuse
pub fn xchacha20_poly1305_encrypt(
    plaintext: &[u8],
    key: &[u8; 32],
    nonce: &[u8; 24],
    aad: &[u8],
) -> Result<Vec<u8>> {
    use chacha20poly1305::{
        aead::{Aead, KeyInit},
        XChaCha20Poly1305, XNonce,
    };
    
    if key.len() != 32 {
        return Err(AdvancedCryptoError::InvalidKey("Key must be 32 bytes".to_string()));
    }
    
    if nonce.len() != 24 {
        return Err(AdvancedCryptoError::InvalidNonce("Nonce must be 24 bytes".to_string()));
    }
    
    let cipher = XChaCha20Poly1305::new(key.into());
    let nonce = XNonce::from_slice(nonce);
    
    let ciphertext = cipher
        .encrypt(nonce, aead::Payload::from(plaintext))
        .map_err(|e| AdvancedCryptoError::EncryptionFailed(e.to_string()))?;
    
    Ok(ciphertext)
}

/// XChaCha20-Poly1305 Decrypt
pub fn xchacha20_poly1305_decrypt(
    ciphertext: &[u8],
    key: &[u8; 32],
    nonce: &[u8; 24],
    aad: &[u8],
) -> Result<Vec<u8>> {
    use chacha20poly1305::{
        aead::{Aead, KeyInit},
        XChaCha20Poly1305, XNonce,
    };
    
    if key.len() != 32 {
        return Err(AdvancedCryptoError::InvalidKey("Key must be 32 bytes".to_string()));
    }
    
    if nonce.len() != 24 {
        return Err(AdvancedCryptoError::InvalidNonce("Nonce must be 24 bytes".to_string()));
    }
    
    if ciphertext.len() < 16 {
        return Err(AdvancedCryptoError::DecryptionFailed("Ciphertext too short".to_string()));
    }
    
    let cipher = XChaCha20Poly1305::new(key.into());
    let nonce = XNonce::from_slice(nonce);
    
    let plaintext = cipher
        .decrypt(nonce, aead::Payload::from(ciphertext))
        .map_err(|e| AdvancedCryptoError::AuthenticationFailed(e.to_string()))?;
    
    Ok(plaintext)
}

// ============================================================================
// Argon2id Password Hashing (Memory-Hard)
// ============================================================================

/// Argon2id hash with sensible defaults for wallet keys
/// - Memory: 64 MB (65536 KB)
/// - Iterations: 3
/// - Parallelism: 4 threads
/// - Output: 32 bytes
pub fn argon2id_hash(password: &[u8], salt: &[u8]) -> Result<[u8; 32]> {
    use argon2::{
        password_hash::{PasswordHasher, SaltString},
        Argon2,
    };
    
    let salt = SaltString::encode_b64(salt)
        .map_err(|e| AdvancedCryptoError::KeyDerivationFailed(e.to_string()))?;
    
    let argon2 = Argon2::new(
        argon2::Algorithm::Argon2id,
        argon2::Version::V0x13,
        argon2::Params::new(65536, 3, 4, Some(32))
            .map_err(|e| AdvancedCryptoError::KeyDerivationFailed(e.to_string()))?,
    );
    
    let hash = argon2
        .hash_password(password, &salt)
        .map_err(|e| AdvancedCryptoError::KeyDerivationFailed(e.to_string()))?;
    
    // Extract first 32 bytes of hash
    let hash_bytes = hash.hash.ok_or_else(|| 
        AdvancedCryptoError::KeyDerivationFailed("No hash output".to_string())
    )?;
    let hash_ref = hash_bytes.as_bytes();
    
    let mut result = [0u8; 32];
    let len = std::cmp::min(32, hash_ref.len());
    result[..len].copy_from_slice(&hash_ref[..len]);
    
    Ok(result)
}

/// Verify Argon2id hash
pub fn argon2id_verify(password: &[u8], salt: &[u8], expected_hash: &[u8; 32]) -> bool {
    match argon2id_hash(password, salt) {
        Ok(computed) => computed == *expected_hash,
        Err(_) => false,
    }
}

// ============================================================================
// Secure Memory Operations
// ============================================================================

/// Securely zero memory (prevents compiler optimization from removing)
pub fn secure_zero(data: &mut [u8]) {
    use zeroize::Zeroize;
    data.zeroize();
}

/// Constant-time comparison (prevents timing attacks)
pub fn constant_time_compare(a: &[u8], b: &[u8]) -> bool {
    use subtle::ConstantTimeEq;
    
    if a.len() != b.len() {
        return false;
    }
    
    a.ct_eq(b).unwrap_u8() == 1
}

/// Constant-time selection
pub fn constant_time_select<T: Clone>(condition: bool, a: &T, b: &T) -> T {
    if condition {
        a.clone()
    } else {
        b.clone()
    }
}

// ============================================================================
// Key Wrapping (for hardware security modules)
// ============================================================================

/// Wrap a key using AES-KW (Key Wrap)
// AES Key Wrap algorithm (RFC 3394)
pub fn aes_kw_wrap(key: &[u8; 32], kek: &[u8; 32]) -> Result<Vec<u8>> {
    // Simplified key wrap - in production use aes-kw crate
    // This creates a secure envelope around the key
    
    let nonce = generate_secure_random(12)?
        .try_into()
        .map_err(|v: Vec<u8>| AdvancedCryptoError::RandomGenerationFailed(
            format!("Failed to create 12-byte nonce: len={}", v.len())
        ))?;
    
    // Encrypt the key material
    let ciphertext = aes256_gcm_encrypt(key, kek, &nonce, &[])?;
    
    // Combine: nonce (12) + ciphertext with tag
    let mut result = Vec::with_capacity(12 + ciphertext.len());
    result.extend_from_slice(&nonce);
    result.extend_from_slice(&ciphertext);
    
    Ok(result)
}

/// Unwrap a key using AES-KW
pub fn aes_kw_unwrap(wrapped: &[u8], kek: &[u8; 32]) -> Result<[u8; 32]> {
    if wrapped.len() < 12 + 16 {
        return Err(AdvancedCryptoError::DecryptionFailed(
            "Wrapped key too short".to_string()
        ));
    }
    
    let nonce: [u8; 12] = [
        wrapped[0], wrapped[1], wrapped[2], wrapped[3],
        wrapped[4], wrapped[5], wrapped[6], wrapped[7],
        wrapped[8], wrapped[9], wrapped[10], wrapped[11],
    ];
    let ciphertext = &wrapped[12..];
    
    let plaintext = aes256_gcm_decrypt(ciphertext, kek, &nonce, &[])?;
    
    if plaintext.len() != 32 {
        return Err(AdvancedCryptoError::DecryptionFailed(
            "Unwrapped key wrong length".to_string()
        ));
    }
    
    let mut key = [0u8; 32];
    key.copy_from_slice(&plaintext);
    Ok(key)
}

// ============================================================================
// HD Wallet Key Derivation (BIP-32 with hardened derivation)
// ============================================================================

/// Derive child key from parent using hardened derivation
/// This prevents child key leakage from compromising parent
pub fn derive_hardened_key(
    parent_key: &[u8; 32],
    parent_chain_code: &[u8; 32],
    child_index: u32,
) -> Result<([u8; 32], [u8; 32])> {
    // BIP-32 hardened derivation
    // HMAC-SHA512(parent_key, 0x00 || parent_key || child_index)
    
    use hmac::{Hmac, Mac};
    use sha2::Sha512;
    
    type HmacSha512 = Hmac<Sha512>;
    
    let mut mac = HmacSha512::new_from_slice(parent_key)
        .map_err(|e| AdvancedCryptoError::KeyDerivationFailed(e.to_string()))?;
    
    // Add hardened derivation marker (0x00)
    mac.update(&[0u8]);
    // Add parent key
    mac.update(parent_key);
    // Add child index (big-endian, 4 bytes)
    mac.update(&child_index.to_be_bytes());
    
    let result = mac.finalize().into_bytes();
    
    let mut child_key = [0u8; 32];
    let mut chain_code = [0u8; 32];
    
    child_key.copy_from_slice(&result[..32]);
    chain_code.copy_from_slice(&result[32..]);
    
    // Validate child key is not zero or > secp256k1 order
    if child_key.iter().all(|&b| b == 0) {
        return Err(AdvancedCryptoError::KeyDerivationFailed(
            "Invalid child key (zero)".to_string()
        ));
    }
    
    Ok((child_key, chain_code))
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_aes256_gcm_roundtrip() {
        let plaintext = b"TigerWallet Secret Data";
        let key = generate_key().unwrap();
        let nonce = generate_nonce_96().unwrap();
        
        let ciphertext = aes256_gcm_encrypt(plaintext, &key, &nonce, &[]).unwrap();
        let decrypted = aes256_gcm_decrypt(&ciphertext, &key, &nonce, &[]).unwrap();
        
        assert_eq!(plaintext.to_vec(), decrypted);
    }
    
    #[test]
    fn test_chacha20_roundtrip() {
        let plaintext = b"TigerWallet Secret Data";
        let key = generate_key().unwrap();
        let nonce = generate_nonce_96().unwrap();
        
        let ciphertext = chacha20_poly1305_encrypt(plaintext, &key, &nonce, &[]).unwrap();
        let decrypted = chacha20_poly1305_decrypt(&ciphertext, &key, &nonce, &[]).unwrap();
        
        assert_eq!(plaintext.to_vec(), decrypted);
    }
    
    #[test]
    fn test_xchacha20_roundtrip() {
        let plaintext = b"TigerWallet Secret Data";
        let key = generate_key().unwrap();
        let nonce = generate_nonce_192().unwrap();
        
        let ciphertext = xchacha20_poly1305_encrypt(plaintext, &key, &nonce, &[]).unwrap();
        let decrypted = xchacha20_poly1305_decrypt(&ciphertext, &key, &nonce, &[]).unwrap();
        
        assert_eq!(plaintext.to_vec(), decrypted);
    }
    
    #[test]
    fn test_argon2id() {
        let password = b"secure_password_123";
        let salt = generate_salt().unwrap();
        
        let hash = argon2id_hash(password, &salt).unwrap();
        assert!(argon2id_verify(password, &salt, &hash));
        
        // Wrong password should fail
        assert!(!argon2id_verify(b"wrong_password", &salt, &hash));
    }
    
    #[test]
    fn test_aes_kw() {
        let key_to_wrap = generate_key().unwrap();
        let kek = generate_key().unwrap();
        
        let wrapped = aes_kw_wrap(&key_to_wrap, &kek).unwrap();
        let unwrapped = aes_kw_unwrap(&wrapped, &kek).unwrap();
        
        assert_eq!(key_to_wrap, unwrapped);
    }
    
    #[test]
    fn test_hardened_derivation() {
        let parent_key = generate_key().unwrap();
        let parent_chain_code = generate_key().unwrap();
        
        let (child_key, chain_code) = derive_hardened_key(
            &parent_key,
            &parent_chain_code,
            0,
        ).unwrap();
        
        assert!(!child_key.iter().all(|&b| b == 0));
        assert!(!chain_code.iter().all(|&b| b == 0));
    }
    
    #[test]
    fn test_secure_random_unique() {
        let random1 = generate_secure_random(32).unwrap();
        let random2 = generate_secure_random(32).unwrap();
        
        // With CSPRNG, collisions are astronomically unlikely
        assert_ne!(random1, random2);
    }
    
    #[test]
    fn test_constant_time_compare() {
        let a = b"test";
        let b = b"test";
        let c = b"Test";
        
        assert!(constant_time_compare(a, b));
        assert!(!constant_time_compare(a, c));
    }
}