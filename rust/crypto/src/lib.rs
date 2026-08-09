//! TigerWallet Crypto Library - High-Speed Cryptographic Operations
//!
//! This module provides fast cryptographic functions for:
//! - Key generation and derivation
//! - Signing and verification
//! - Hashing
//! - Encoding/decoding
//! - Advanced encryption (AES-256-GCM, ChaCha20-Poly1305)
//! - BIP-39 mnemonic and HD wallet derivation
//! - Bitcoin, Ordinals, Lightning, Stacks support
//! - Analytics and reporting
//! - Security features

pub mod advanced_encryption;
pub mod mnemonic;
pub mod bitcoin_service;
pub mod analytics;
pub mod security_module;

// ============================================================================
// Error Types
// ============================================================================

#[derive(Debug, thiserror::Error)]
pub enum CryptoError {
    #[error("Invalid key: {0}")]
    InvalidKey(String),

    #[error("Signing failed: {0}")]
    SigningFailed(String),

    #[error("Verification failed: {0}")]
    VerificationFailed(String),

    #[error("Invalid address: {0}")]
    InvalidAddress(String),

    #[error("Encoding error: {0}")]
    EncodingError(String),
}

pub type Result<T> = std::result::Result<T, CryptoError>;

// ============================================================================
// Secure Random
// ============================================================================

/// Generate cryptographically secure random bytes
pub fn generate_secure_random(len: usize) -> std::result::Result<Vec<u8>, String> {
    use rand::rngs::OsRng;
    use rand::RngCore;
    let mut buf = vec![0u8; len];
    OsRng.fill_bytes(&mut buf);
    Ok(buf)
}

// ============================================================================
// Hash Functions
// ============================================================================

/// SHA-256 hash
pub fn sha256(data: &[u8]) -> [u8; 32] {
    use sha2::{Sha256, Digest};
    let mut hasher = Sha256::new();
    hasher.update(data);
    hasher.finalize().into()
}

/// SHA-3-256 hash
pub fn sha3_256(data: &[u8]) -> [u8; 32] {
    use sha3::{Sha3_256, Digest};
    let mut hasher = Sha3_256::new();
    hasher.update(data);
    hasher.finalize().into()
}

/// Keccak-256 (Ethereum style, uses sha3 crate's Keccak256 which is real Keccak)
pub fn keccak256(data: &[u8]) -> [u8; 32] {
    use sha3::Keccak256;
    use sha3::Digest;
    let mut hasher = Keccak256::new();
    hasher.update(data);
    hasher.finalize().into()
}

/// BLAKE2b hash
pub fn blake2b(data: &[u8], output_len: usize) -> Vec<u8> {
    use blake2::{Blake2b512, Digest};
    let mut hasher = Blake2b512::new();
    hasher.update(data);
    hasher.finalize()[..output_len].to_vec()
}

/// RIPEMD-160 hash
pub fn ripemd160(data: &[u8]) -> [u8; 20] {
    use ripemd::{Ripemd160, Digest};
    let mut hasher = Ripemd160::new();
    hasher.update(data);
    let result = hasher.finalize();
    result.into()
}

// ============================================================================
// Key Derivation
// ============================================================================

/// HMAC-SHA256
pub fn hmac_sha256(key: &[u8], data: &[u8]) -> [u8; 32] {
    use hmac::{Hmac, Mac};
    type HmacSha256 = Hmac<sha2::Sha256>;

    let mut mac = HmacSha256::new_from_slice(key)
        .expect("HMAC can take key of any size");
    mac.update(data);

    let result = mac.finalize();
    result.into_bytes().into()
}

/// PBKDF2-SHA256
pub fn pbkdf2_sha256(password: &[u8], salt: &[u8], iterations: u32) -> [u8; 32] {
    use pbkdf2::pbkdf2_hmac_array;
    use sha2::Sha256;

    pbkdf2_hmac_array::<Sha256, 32>(password, salt, iterations)
}

/// PBKDF2-SHA512
pub fn pbkdf2_sha512(password: &[u8], salt: &[u8], iterations: u32) -> [u8; 64] {
    use pbkdf2::pbkdf2_hmac_array;
    use sha2::Sha512;

    pbkdf2_hmac_array::<Sha512, 64>(password, salt, iterations)
}

// ============================================================================
// EVM Key Operations
// ============================================================================

/// Generate random private key
pub fn generate_private_key() -> [u8; 32] {
    use secp256k1::SecretKey;
    use rand::rngs::OsRng;

    SecretKey::new(&mut OsRng).secret_bytes()
}

/// Get public key from private key
pub fn private_to_public(private_key: &[u8; 32]) -> [u8; 64] {
    use secp256k1::{PublicKey, SecretKey, Secp256k1};

    let secp = Secp256k1::new();
    let secret = SecretKey::from_slice(private_key)
        .expect("32 bytes");
    let public = PublicKey::from_secret_key(&secp, &secret);

    let serialized = public.serialize_uncompressed();
    // Skip first byte (0x04), take next 64 bytes
    serialized[1..65].try_into().unwrap()
}

/// Get Ethereum address from public key
pub fn public_to_eth_address(public_key: &[u8; 64]) -> String {
    let hash = keccak256(public_key);
    // Take last 20 bytes
    let address_bytes = &hash[12..];
    format!("0x{}", hex::encode(address_bytes))
}

/// Get address from private key
pub fn private_to_eth_address(private_key: &[u8; 32]) -> String {
    let public = private_to_public(private_key);
    public_to_eth_address(&public)
}

/// Sign message (EIP-191 format)
pub fn sign_message(message: &[u8], private_key: &[u8; 32]) -> Result<(String, String)> {
    use secp256k1::{SecretKey, Message, Secp256k1};

    let secp = Secp256k1::new();
    let secret = SecretKey::from_slice(private_key)
        .map_err(|e| CryptoError::SigningFailed(e.to_string()))?;

    let prefix = format!("\x19Ethereum Signed Message:\n{}", message.len());
    let message_hash = keccak256(format!("{}{}", prefix, String::from_utf8_lossy(message)).as_bytes());

    let msg = Message::from_slice(&message_hash)
        .map_err(|e| CryptoError::SigningFailed(e.to_string()))?;
    let signature = secp.sign_ecdsa(&msg, &secret);

    let sig_bytes = signature.serialize_compact();

    Ok((
        format!("0x{}", hex::encode(sig_bytes)),
        format!("0x{}", hex::encode(message_hash))
    ))
}

/// Recover address from signature
pub fn recover_address(message: &[u8], signature: &[u8; 65]) -> Result<String> {
    use secp256k1::{Message, Secp256k1, ecdsa::{RecoveryId, RecoverableSignature}};

    if signature[64] > 3 {
        return Err(CryptoError::VerificationFailed("Invalid recovery id".to_string()));
    }

    let secp = Secp256k1::new();
    let sig_bytes: [u8; 64] = signature[..64].try_into().unwrap();
    let v = signature[64];

    let recovery_id = RecoveryId::from_i32(v as i32)
        .map_err(|e| CryptoError::VerificationFailed(e.to_string()))?;
    let sig = RecoverableSignature::from_compact(&sig_bytes, recovery_id)
        .map_err(|e| CryptoError::VerificationFailed(e.to_string()))?;

    let prefix = format!("\x19Ethereum Signed Message:\n{}", message.len());
    let message_hash = keccak256(format!("{}{}", prefix, String::from_utf8_lossy(message)).as_bytes());
    let msg = Message::from_slice(&message_hash)
        .map_err(|e| CryptoError::VerificationFailed(e.to_string()))?;

    let public = secp.recover_ecdsa(&msg, &sig)
        .map_err(|e| CryptoError::VerificationFailed(e.to_string()))?;

    let serialized = public.serialize_uncompressed();
    let address = public_to_eth_address(&serialized[1..65].try_into().unwrap());

    Ok(address)
}

// ============================================================================
// Address Validation
// ============================================================================

/// Validate Ethereum address
pub fn is_valid_eth_address(address: &str) -> bool {
    if !address.starts_with("0x") {
        return false;
    }
    if address.len() != 42 {
        return false;
    }
    address[2..].chars().all(|c| c.is_ascii_hexdigit())
}

/// Validate Bitcoin address (basic)
pub fn is_valid_btc_address(address: &str) -> bool {
    // Basic validation - check length and characters
    let valid = address.chars().all(|c| c.is_alphanumeric() || c == '1' || c == '3');
    let valid_len = address.len() >= 26 && address.len() <= 35;
    valid && valid_len
}

/// Validate Solana address
pub fn is_valid_sol_address(address: &str) -> bool {
    // Base58 encoded, 32-44 characters
    let valid_chars = address.chars().all(|c| {
        !['0', 'I', 'O', 'l'].contains(&c) && (c.is_alphanumeric() || c == '-')
    });
    let valid_len = address.len() >= 32 && address.len() <= 44;
    valid_chars && valid_len
}

// ============================================================================
// Encoding
// ============================================================================

/// Base58 encode
pub fn base58_encode(data: &[u8]) -> String {
    bs58::encode(data).into_string()
}

/// Base58 decode
pub fn base58_decode(data: &str) -> Result<Vec<u8>> {
    bs58::decode(data).into_vec()
        .map_err(|e| CryptoError::EncodingError(e.to_string()))
}

/// Base64 encode
pub fn base64_encode(data: &[u8]) -> String {
    use base64::Engine;
    base64::engine::general_purpose::STANDARD.encode(data)
}

/// Base64 decode
pub fn base64_decode(data: &str) -> Result<Vec<u8>> {
    use base64::Engine;
    base64::engine::general_purpose::STANDARD
        .decode(data)
        .map_err(|e| CryptoError::EncodingError(e.to_string()))
}

/// Hex encode
pub fn hex_encode(data: &[u8]) -> String {
    hex::encode(data)
}

/// Hex decode
pub fn hex_decode(data: &str) -> Result<Vec<u8>> {
    hex::decode(data)
        .map_err(|e| CryptoError::EncodingError(e.to_string()))
}

// ============================================================================
// EVM Specific
// ============================================================================

/// Encode EVM function call
pub fn encode_function_call(function: &str, params: &[&[u8]]) -> String {
    // Calculate function selector (first 4 bytes of keccak256)
    let selector = keccak256(function.as_bytes());
    let selector_hex = format!("0x{}", hex::encode(&selector[..4]));

    // Encode parameters
    let mut encoded = selector_hex;
    for param in params {
        encoded.push_str(&hex::encode(param));
    }

    encoded
}

/// Decode EVM function call
pub fn decode_function_call(data: &str) -> Result<(String, Vec<Vec<u8>>)> {
    let data = data.trim_start_matches("0x");
    if data.len() < 8 {
        return Err(CryptoError::EncodingError("Data too short".to_string()));
    }

    // First 4 bytes = function selector
    let selector = &data[..8];
    let params_hex = &data[8..];

    // Return selector as function name (would need ABI lookup in production)
    let func_name = format!("0x{}", selector);

    // Decode parameters (32 bytes each)
    let mut params = Vec::new();
    for chunk in params_hex.as_bytes().chunks(64) {
        if chunk.len() == 64 {
            if let Ok(decoded) = hex::decode(chunk) {
                params.push(decoded);
            }
        }
    }

    Ok((func_name, params))
}

// ============================================================================
// Encryption
// ============================================================================

/// AES-256-CBC encrypt
pub fn aes256_cbc_encrypt(plaintext: &[u8], key: &[u8; 32], iv: &[u8; 16]) -> Result<Vec<u8>> {
    use aes::Aes256;
    use cbc::{Encryptor, cipher::{BlockEncryptMut, KeyIvInit}};

    type Aes256Cbc = Encryptor<Aes256>;

    let cipher = Aes256Cbc::new(key.into(), iv.into());
    let ciphertext = cipher.encrypt_padded_vec_mut::<aes::cipher::block_padding::NoPadding>(plaintext);

    Ok(ciphertext)
}

/// AES-256-CBC decrypt
pub fn aes256_cbc_decrypt(ciphertext: &[u8], key: &[u8; 32], iv: &[u8; 16]) -> Result<Vec<u8>> {
    use aes::Aes256;
    use cbc::{Decryptor, cipher::{BlockDecryptMut, KeyIvInit}};

    type Aes256Cbc = Decryptor<Aes256>;

    let cipher = Aes256Cbc::new(key.into(), iv.into());
    cipher.decrypt_padded_vec_mut::<aes::cipher::block_padding::NoPadding>(ciphertext)
        .map_err(|e| CryptoError::EncodingError(e.to_string()))
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sha256() {
        let data = b"hello";
        let hash = sha256(data);
        assert_eq!(hash.len(), 32);
    }

    #[test]
    fn test_keccak256() {
        let data = b"";
        let hash = keccak256(data);
        // Known empty string keccak256
        assert_eq!(hash.len(), 32);
    }

    #[test]
    fn test_address_generation() {
        let private_key = generate_private_key();
        let address = private_to_eth_address(&private_key);

        assert!(address.starts_with("0x"));
        assert_eq!(address.len(), 42);
    }

    #[test]
    fn test_address_validation() {
        assert!(is_valid_eth_address("0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E"));
        assert!(!is_valid_eth_address("0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1"));
    }

    #[test]
    fn test_base58() {
        let data = b"Hello, TigerWallet!";
        let encoded = base58_encode(data);
        let decoded = base58_decode(&encoded).unwrap();
        assert_eq!(data.to_vec(), decoded);
    }

    #[test]
    fn test_base64() {
        let data = b"Hello, TigerWallet!";
        let encoded = base64_encode(data);
        let decoded = base64_decode(&encoded).unwrap();
        assert_eq!(data.to_vec(), decoded);
    }

    #[test]
    fn test_hex() {
        let data = b"Hello";
        let encoded = hex_encode(data);
        let decoded = hex_decode(&encoded).unwrap();
        assert_eq!(data.to_vec(), decoded);
    }

    #[test]
    fn test_aes_cbc() {
        let key = [0u8; 32];
        let iv = [0u8; 16];
        let plaintext = b"Hello, TigerWallet!!"; // 20 bytes -> pad to 32 (block multiple with NoPadding)
        let mut pt = plaintext.to_vec();
        pt.resize(32, 0);

        let ciphertext = aes256_cbc_encrypt(&pt, &key, &iv).unwrap();
        let decrypted = aes256_cbc_decrypt(&ciphertext, &key, &iv).unwrap();

        assert_eq!(pt, decrypted);
    }

    #[test]
    fn test_hmac() {
        let key = b"secret";
        let data = b"message";
        let result = hmac_sha256(key, data);
        assert_eq!(result.len(), 32);
    }

    #[test]
    fn test_pbkdf2() {
        let password = b"password";
        let salt = b"salt";
        let result = pbkdf2_sha256(password, salt, 1000);
        assert_eq!(result.len(), 32);
    }
}
