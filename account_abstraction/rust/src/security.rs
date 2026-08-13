//! Security module for Account Abstraction.
//!
//! Uses REAL secp256k1 ECDSA (the Ethereum curve) via `k256` with keccak256
//! message hashing. `generate_key_pair` stores the private key material so
//! `sign` always signs with the SAME key whose public key was registered --
//! the signature is verifiable. `verify_signature` performs a real ECDSA
//! verification against the stored public key (never returns true blindly).
//!
//! AES-256-GCM encrypt/decrypt (ring) is retained -- it is real authenticated
//! encryption and unaffected by the signing-curve fix.

use crate::AAError;
use k256::ecdsa::{Signature, SigningKey, VerifyingKey};
use ring::aead::{Aad, BoundKey, Nonce, NonceSequence, OpeningKey, SealingKey, UnboundKey, AES_256_GCM};
use sha3::{Digest, Keccak256};
use signature::hazmat::{PrehashVerifier, RandomizedPrehashSigner};
use std::collections::HashMap;
use k256::elliptic_curve::rand_core::{OsRng, RngCore};

/// Security level
#[derive(Debug, Clone)]
pub enum SecurityLevel {
    Standard,
    High,
    Institutional,
}

/// Key information (public material only; the private key lives in the
/// module's `secret_keys` map and is never exposed through this struct).
#[derive(Debug, Clone)]
pub struct KeyInfo {
    /// SEC1-encoded (33-byte compressed or 65-byte uncompressed) secp256k1 public key.
    pub public_key: Vec<u8>,
    pub key_id: String,
    pub created_at: u64,
}

/// Security module
pub struct SecurityModule {
    /// Public key info keyed by key_id.
    keys: HashMap<String, KeyInfo>,
    /// Private signing keys keyed by key_id. Held in memory so `sign` signs
    /// with the same key whose public key was registered by `generate_key_pair`.
    secret_keys: HashMap<String, SigningKey>,
    /// Security level
    level: SecurityLevel,
    /// Rate limits
    rate_limits: HashMap<String, RateLimit>,
    /// Failed attempts tracking
    failed_attempts: HashMap<String, u32>,
    /// Locked accounts
    locked: HashMap<String, u64>,
}

impl SecurityModule {
    pub fn new() -> Self {
        Self {
            keys: HashMap::new(),
            secret_keys: HashMap::new(),
            level: SecurityLevel::Standard,
            rate_limits: HashMap::new(),
            failed_attempts: HashMap::new(),
            locked: HashMap::new(),
        }
    }

    /// Set security level
    pub fn set_level(&mut self, level: SecurityLevel) {
        self.level = level;
    }

    /// Generate a real secp256k1 key pair and store it under `key_id`. The
    /// private key is retained so subsequent `sign` calls produce signatures
    /// verifiable against the returned public key.
    pub fn generate_key_pair(&mut self, key_id: &str) -> Result<KeyInfo, AAError> {
        let signing_key = SigningKey::random(&mut OsRng);
        let verifying_key = VerifyingKey::from(&signing_key);
        // SEC1 uncompressed (65 bytes) encoding of the public key.
        let public_key = verifying_key.to_sec1_bytes().to_vec();

        let info = KeyInfo {
            public_key: public_key.clone(),
            key_id: key_id.to_string(),
            created_at: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs(),
        };

        self.keys.insert(key_id.to_string(), info.clone());
        self.secret_keys.insert(key_id.to_string(), signing_key);
        Ok(info)
    }

    /// Sign `data` with the stored secp256k1 private key for `key_id`, after
    /// hashing it with keccak256 (the Ethereum message-digest convention). The
    /// returned bytes are a fixed-size ECDSA signature (r || s, 64 bytes).
    pub async fn sign(&self, data: &[u8], key_id: &str) -> Result<Vec<u8>, AAError> {
        let signing_key = self.secret_keys.get(key_id)
            .ok_or_else(|| AAError::SecurityError(format!("signing key not found for key_id={}", key_id)))?;

        // keccak256 message digest, then sign the prehash with a CSPRNG nonce.
        let digest = Keccak256::digest(data);
        let signature: Signature = signing_key
            .sign_prehash_with_rng(&mut OsRng, &digest)
            .map_err(|e| AAError::SecurityError(format!("sign: {}", e)))?;

        Ok(signature.to_bytes().to_vec())
    }

    /// Verify a secp256k1 ECDSA signature against the stored public key for
    /// `signer` (used as the key_id). Performs a REAL keccak256 + ECDSA
    /// verification; returns `Ok(false)` on a bad signature (never `Ok(true)`
    /// unconditionally).
    pub fn verify_signature(&self, data: &[u8], signature: &[u8], signer: &str) -> Result<bool, String> {
        let info = self.keys.get(signer)
            .ok_or_else(|| format!("no public key registered for signer={}", signer))?;

        let verifying_key = VerifyingKey::from_sec1_bytes(&info.public_key)
            .map_err(|e| format!("invalid stored public key: {}", e))?;

        let sig = Signature::from_slice(signature)
            .map_err(|e| format!("invalid signature encoding: {}", e))?;

        let digest = Keccak256::digest(data);
        Ok(verifying_key.verify_prehash(&digest, &sig).is_ok())
    }

    /// Encrypt data with AES-256-GCM. Output = nonce(12) || ciphertext+tag.
    pub fn encrypt(&self, data: &[u8], key: &[u8]) -> Result<Vec<u8>, AAError> {
        if key.len() != 32 {
            return Err(AAError::SecurityError("Invalid key length".to_string()));
        }

        let mut nonce_bytes = [0u8; 12];
        OsRng.fill_bytes(&mut nonce_bytes);

        let unbound_key = UnboundKey::new(&AES_256_GCM, key)
            .map_err(|e| AAError::SecurityError(e.to_string()))?;

        let nonce_seq = OneNonceSequence::new(Nonce::assume_unique_for_key(nonce_bytes));
        let mut bound_key = SealingKey::new(unbound_key, nonce_seq);

        let mut in_out = data.to_vec();
        bound_key.seal_in_place_append_tag(Aad::empty(), &mut in_out)
            .map_err(|e| AAError::SecurityError(e.to_string()))?;

        let mut result = nonce_bytes.to_vec();
        result.append(&mut in_out);
        Ok(result)
    }

    /// Decrypt data produced by `encrypt` (AES-256-GCM).
    pub fn decrypt(&self, data: &[u8], key: &[u8]) -> Result<Vec<u8>, AAError> {
        if key.len() != 32 {
            return Err(AAError::SecurityError("Invalid key length".to_string()));
        }
        if data.len() < 12 {
            return Err(AAError::SecurityError("Data too short".to_string()));
        }

        let nonce_bytes: [u8; 12] = data[..12]
            .try_into()
            .map_err(|_| AAError::SecurityError("Invalid nonce".to_string()))?;
        let mut ciphertext = data[12..].to_vec();

        let unbound_key = UnboundKey::new(&AES_256_GCM, key)
            .map_err(|e| AAError::SecurityError(e.to_string()))?;

        let nonce_seq = OneNonceSequence::new(Nonce::assume_unique_for_key(nonce_bytes));
        let mut bound_key = OpeningKey::new(unbound_key, nonce_seq);

        // open_in_place expects ciphertext||tag and overwrites with plaintext (sans tag).
        let plaintext = bound_key.open_in_place(Aad::empty(), &mut ciphertext)
            .map_err(|e| AAError::SecurityError(e.to_string()))?;

        Ok(plaintext.to_vec())
    }

    /// Set rate limit
    pub fn set_rate_limit(&mut self, key: &str, max_requests: u32, window_seconds: u64) {
        self.rate_limits.insert(key.to_string(), RateLimit {
            max_requests,
            window_seconds,
            requests: vec![],
        });
    }

    /// Check rate limit
    pub fn check_rate_limit(&self, key: &str) -> Result<bool, AAError> {
        if let Some(limit) = self.rate_limits.get(key) {
            let now = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs();

            let recent_requests = limit.requests.iter()
                .filter(|t| **t > now.saturating_sub(limit.window_seconds))
                .count() as u32;

            if recent_requests >= limit.max_requests {
                return Err(AAError::SecurityError("Rate limit exceeded".to_string()));
            }
        }
        Ok(true)
    }

    /// Record failed attempt
    pub fn record_failed_attempt(&mut self, key: &str) {
        let count = self.failed_attempts.entry(key.to_string()).or_insert(0);
        *count += 1;

        if *count >= 5 {
            let lock_time = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs() + 3600;
            self.locked.insert(key.to_string(), lock_time);
        }
    }

    /// Check if locked
    pub fn is_locked(&self, key: &str) -> bool {
        if let Some(lock_time) = self.locked.get(key) {
            let now = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs();
            return now < *lock_time;
        }
        false
    }

    /// Unlock
    pub fn unlock(&mut self, key: &str) {
        self.locked.remove(key);
        self.failed_attempts.remove(key);
    }
}

impl Default for SecurityModule {
    fn default() -> Self {
        Self::new()
    }
}

/// Rate limit configuration
#[derive(Debug, Clone)]
pub struct RateLimit {
    pub max_requests: u32,
    pub window_seconds: u64,
    pub requests: Vec<u64>,
}

/// One-time nonce sequence for AES-GCM.
pub struct OneNonceSequence {
    nonce: Option<Nonce>,
}

impl OneNonceSequence {
    pub fn new(nonce: Nonce) -> Self {
        Self { nonce: Some(nonce) }
    }
}

impl NonceSequence for OneNonceSequence {
    fn advance(&mut self) -> Result<Nonce, ring::error::Unspecified> {
        self.nonce.take().ok_or(ring::error::Unspecified)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_sign_verify_roundtrip() {
        let mut sec = SecurityModule::new();
        sec.generate_key_pair("owner1").unwrap();
        let msg = b"hello world";
        let sig = sec.sign(msg, "owner1").await.unwrap();
        assert!(sig.len() == 64, "fixed-size signature must be 64 bytes");
        assert!(sec.verify_signature(msg, &sig, "owner1").unwrap(),
            "real secp256k1 signature must verify against stored public key");
    }

    #[tokio::test]
    async fn test_verify_rejects_tampered() {
        let mut sec = SecurityModule::new();
        sec.generate_key_pair("owner2").unwrap();
        let msg = b"hello world";
        let mut sig = sec.sign(msg, "owner2").await.unwrap();
        sig[0] ^= 0xff; // tamper
        assert!(!sec.verify_signature(msg, &sig, "owner2").unwrap(),
            "tampered signature must NOT verify");
    }

    #[tokio::test]
    async fn test_verify_rejects_wrong_message() {
        let mut sec = SecurityModule::new();
        sec.generate_key_pair("owner3").unwrap();
        let sig = sec.sign(b"message A", "owner3").await.unwrap();
        assert!(!sec.verify_signature(b"message B", &sig, "owner3").unwrap(),
            "signature over A must NOT verify for B");
    }

    #[test]
    fn test_encrypt_decrypt_roundtrip() {
        let sec = SecurityModule::new();
        let key = [0x42u8; 32];
        let plaintext = b"secret custody data";
        let ct = sec.encrypt(plaintext, &key).unwrap();
        let pt = sec.decrypt(&ct, &key).unwrap();
        assert_eq!(pt.as_slice(), plaintext);
    }

    #[test]
    fn test_decrypt_wrong_key_fails() {
        let sec = SecurityModule::new();
        let ct = sec.encrypt(b"data", &[0x42u8; 32]).unwrap();
        assert!(sec.decrypt(&ct, &[0x99u8; 32]).is_err(),
            "wrong key must fail GCM auth tag");
    }
}
