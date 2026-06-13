//! Security module for Account Abstraction

use crate::AAError;
use ring::aead::{Aad, BoundKey, Nonce, NonceSequence, UnboundKey, AES_256_GCM};
use ring::rand::{SecureRandom, SystemRandom};
use ring::signature::{EcdsaKeyPair, ECDSA_P256_SHA256_FIXED, ECDSA_P256_SHA256_FIXED_SIGNING};
use std::collections::HashMap;

/// Security level
#[derive(Debug, Clone)]
pub enum SecurityLevel {
    Standard,
    High,
    Institutional,
}

/// Key information
#[derive(Debug, Clone)]
pub struct KeyInfo {
    pub public_key: Vec<u8>,
    pub key_id: String,
    pub created_at: u64,
}

/// Security module
pub struct SecurityModule {
    /// Keys storage
    keys: HashMap<String, KeyInfo>,
    /// Random generator
    rng: SystemRandom,
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
            rng: SystemRandom::new(),
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

    /// Generate key pair
    pub fn generate_key_pair(&mut self, key_id: &str) -> Result<KeyInfo, AAError> {
        let pkcs8_bytes = EcdsaKeyPair::generate_pkcs8(&ECDSA_P256_SHA256_FIXED_SIGNING, &self.rng)
            .map_err(|e| AAError::SecurityError(e.to_string()))?;
        
        let key_pair = EcdsaKeyPair::from_pkcs8(&ECDSA_P256_SHA256_FIXED_SIGNING, pkcs8_bytes.as_ref(), &self.rng)
            .map_err(|e| AAError::SecurityError(e.to_string()))?;
        
        let public_key = key_pair.public_key().as_ref().to_vec();
        
        let info = KeyInfo {
            public_key: public_key.clone(),
            key_id: key_id.to_string(),
            created_at: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs(),
        };
        
        self.keys.insert(key_id.to_string(), info);
        
        Ok(info)
    }

    /// Sign data
    pub async fn sign(&self, data: &[u8], key_id: &str) -> Result<Vec<u8>, AAError> {
        let key = self.keys.get(key_id)
            .ok_or(AAError::SecurityError("Key not found".to_string()))?;
        
        // Generate signature
        let pkcs8_bytes = EcdsaKeyPair::generate_pkcs8(&ECDSA_P256_SHA256_FIXED_SIGNING, &self.rng)
            .map_err(|e| AAError::SecurityError(e.to_string()))?;
        
        let key_pair = EcdsaKeyPair::from_pkcs8(&ECDSA_P256_SHA256_FIXED_SIGNING, pkcs8_bytes.as_ref(), &self.rng)
            .map_err(|e| AAError::SecurityError(e.to_string()))?;
        
        let signature = key_pair.sign(&self.rng, data)
            .map_err(|e| AAError::SecurityError(e.to_string()))?;
        
        Ok(signature.as_ref().to_vec())
    }

    /// Verify signature
    pub fn verify_signature(&self, data: &[u8], signature: &[u8], signer: &str) -> Result<bool, String> {
        // In production, would verify against stored public key
        // For now, just return true if signature present
        if signature.is_empty() {
            return Ok(false);
        }
        
        Ok(true)
    }

    /// Encrypt data
    pub fn encrypt(&self, data: &[u8], key: &[u8]) -> Result<Vec<u8>, AAError> {
        if key.len() != 32 {
            return Err(AAError::SecurityError("Invalid key length".to_string()));
        }
        
        // Generate nonce
        let mut nonce_bytes = [0u8; 12];
        self.rng.fill(&mut nonce_bytes)
            .map_err(|e| AAError::SecurityError(e.to_string()))?;
        
        // Create unbound key
        let unbound_key = UnboundKey::new(&AES_256_GCM, key)
            .map_err(|e| AAError::SecurityError(e.to_string()))?;
        
        // Create nonce sequence
        let nonce_seq = OneNonceSequence::new(Nonce::assume_unique_for_slice(nonce_bytes));
        
        // Create bound key
        let mut bound_key = bound_key.into_bound_key(nonce_seq);
        
        // Seal
        let mut in_out = data.to_vec();
        bound_key.seal_in_place_separate_tag(Aad::empty())
            .map_err(|e| AAError::SecurityError(e.to_string()))?;
        
        // Prepend nonce
        let mut result = nonce_bytes.to_vec();
        result.append(&mut in_out);
        
        Ok(result)
    }

    /// Decrypt data
    pub fn decrypt(&self, data: &[u8], key: &[u8]) -> Result<Vec<u8>, AAError> {
        if key.len() != 32 {
            return Err(AAError::SecurityError("Invalid key length".to_string()));
        }
        
        if data.len() < 12 {
            return Err(AAError::SecurityError("Data too short".to_string()));
        }
        
        // Extract nonce
        let nonce_bytes: [u8; 12] = data[..12].try_into()
            .map_err(|_| AAError::SecurityError("Invalid nonce".to_string()))?;
        
        let ciphertext = &data[12..];
        
        // Create unbound key
        let unbound_key = UnboundKey::new(&AES_256_GCM, key)
            .map_err(|e| AAError::SecurityError(e.to_string()))?;
        
        // Create nonce sequence
        let nonce_seq = OneNonceSequence::new(Nonce::assume_unique_for_slice(nonce_bytes));
        
        // Create bound key
        let mut bound_key = unbound_key.into_bound_key(nonce_seq);
        
        // Open
        let mut in_out = ciphertext.to_vec();
        bound_key.open_in_place(Aad::empty())
            .map_err(|e| AAError::SecurityError(e.to_string()))?;
        
        Ok(in_out)
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
                .filter(|t| *t > now - limit.window_seconds as u64)
                .count() as u32;
            
            if recent_requests >= limit.max_requests {
                return Err(AAError::SecurityError("Rate limit exceeded".to_string()));
            }
        }
        
        Ok(())
    }

    /// Record failed attempt
    pub fn record_failed_attempt(&mut self, key: &str) {
        let count = self.failed_attempts.entry(key.to_string()).or_insert(0);
        *count += 1;
        
        // Lock after 5 failed attempts
        if *count >= 5 {
            let lock_time = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs() + 3600; // 1 hour
            
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

/// One-time nonce sequence
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
        self.nonce.take()
            .ok_or(ring::error::Unspecified)
    }
}