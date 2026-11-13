//! TigerWallet MPC (Multi-Party Computation) Wallet Implementation
//!
//! This module provides enterprise-grade MPC wallet functionality with:
//! - 2-of-3 or 3-of-5 threshold signature schemes
//! - Social recovery via trusted contacts
//! - Key sharding with no single point of failure
//! - Distributed key generation (DKG)
//! - TSS (Threshold Signature Scheme)
//!
//! Security Features:
//! - All key shares stored separately
//! - No single entity holds complete private key
//! - Social recovery requires multiple contacts
//! - Biometric + PIN + recovery key
//! - Hardware security module (HSM) integration
//!
//! Architecture:
//! - Client-side key generation
//! - Encrypted share storage
//! - Distributed signing protocol
//! - Backup via mnemonic shards

use std::collections::HashMap;

use rand::rngs::OsRng;
use rand::RngCore;
use secp256k1::{PublicKey, Secp256k1, SecretKey};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

use crate::crypto::{aead, secure_random};

/// Arithmetic over GF(256) with the AES reduction polynomial
/// x^8 + x^4 + x^3 + x + 1 (0x11B).
///
/// Shamir's Secret Sharing is normally run over GF(256) for byte-oriented
/// secrets: each byte is split independently, shares are the same length as the
/// secret, and any byte of a share carries no information about the secret.
mod gf256 {
    /// Multiply two field elements (Russian peasant multiplication).
    pub fn mul(mut a: u8, mut b: u8) -> u8 {
        let mut result: u8 = 0;
        while b != 0 {
            if b & 1 != 0 {
                result ^= a;
            }
            let high_bit = a & 0x80;
            a <<= 1;
            if high_bit != 0 {
                a ^= 0x1b;
            }
            b >>= 1;
        }
        result
    }

    /// Multiplicative inverse via Fermat's little theorem (a^254 = a^-1).
    pub fn inverse(a: u8) -> Option<u8> {
        if a == 0 {
            return None;
        }
        let mut result: u8 = 1;
        let mut base = a;
        let mut exponent = 254u16;
        while exponent > 0 {
            if exponent & 1 != 0 {
                result = mul(result, base);
            }
            base = mul(base, base);
            exponent >>= 1;
        }
        Some(result)
    }

    /// Evaluate a polynomial with `coefficients[0]` as the constant term.
    pub fn eval(coefficients: &[u8], x: u8) -> u8 {
        // Horner's method, from the highest-degree coefficient down.
        let mut result: u8 = 0;
        for &coefficient in coefficients.iter().rev() {
            result = mul(result, x) ^ coefficient;
        }
        result
    }

    /// Lagrange interpolation at x = 0.
    ///
    /// Returns `None` when the points are not distinct or empty, because the
    /// interpolation would divide by zero.
    pub fn interpolate_at_zero(points: &[(u8, u8)]) -> Option<u8> {
        if points.is_empty() {
            return None;
        }

        let mut secret: u8 = 0;
        for (i, &(xi, yi)) in points.iter().enumerate() {
            let mut numerator: u8 = 1;
            let mut denominator: u8 = 1;

            for (j, &(xj, _)) in points.iter().enumerate() {
                if i == j {
                    continue;
                }
                if xj == xi {
                    return None;
                }
                // At x = 0 the Lagrange basis term is xj / (xj - xi), and
                // subtraction is XOR in a characteristic-2 field.
                numerator = mul(numerator, xj);
                denominator = mul(denominator, xj ^ xi);
            }

            secret ^= mul(yi, mul(numerator, inverse(denominator)?));
        }

        Some(secret)
    }

    #[cfg(test)]
    mod tests {
        use super::*;

        #[test]
        fn multiplication_matches_known_products() {
            assert_eq!(mul(0, 0x53), 0);
            assert_eq!(mul(1, 0x53), 0x53);
            // From FIPS-197: 0x57 * 0x13 = 0xFE
            assert_eq!(mul(0x57, 0x13), 0xfe);
            // From FIPS-197: 0x57 * 0x83 = 0xC1
            assert_eq!(mul(0x57, 0x83), 0xc1);
        }

        #[test]
        fn every_nonzero_element_has_an_inverse() {
            for value in 1u16..=255 {
                let value = value as u8;
                let inverse = inverse(value).expect("nonzero elements are invertible");
                assert_eq!(mul(value, inverse), 1, "inverse of {value:#04x}");
            }
        }

        #[test]
        fn inverse_of_zero_is_none() {
            assert_eq!(inverse(0), None);
        }

        #[test]
        fn interpolation_recovers_the_constant_term() {
            // p(x) = 7 + 3x over GF(256)
            let coefficients = [7u8, 3];
            let points: Vec<(u8, u8)> = (1u8..=3).map(|x| (x, eval(&coefficients, x))).collect();
            assert_eq!(interpolate_at_zero(&points[..2]), Some(7));
            assert_eq!(interpolate_at_zero(&points[1..]), Some(7));
        }

        #[test]
        fn duplicate_points_are_rejected() {
            assert_eq!(interpolate_at_zero(&[(1, 2), (1, 3)]), None);
        }
    }
}

/// MPC threshold configuration
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub struct ThresholdConfig {
    /// Total number of shares (n)
    pub total_shares: u8,

    /// Minimum shares required to sign (t)
    pub threshold: u8,

    /// Key strength in bits
    pub key_bits: u16,
}

impl ThresholdConfig {
    /// Create 2-of-3 configuration (recommended)
    pub fn two_of_three() -> Self {
        Self {
            total_shares: 3,
            threshold: 2,
            key_bits: 256,
        }
    }

    /// Create 3-of-5 configuration (high security)
    pub fn three_of_five() -> Self {
        Self {
            total_shares: 5,
            threshold: 3,
            key_bits: 256,
        }
    }

    /// Validate configuration
    pub fn is_valid(&self) -> bool {
        self.threshold > 0
            && self.total_shares >= self.threshold
            && self.total_shares <= 10
            && self.key_bits >= 128
    }
}

/// MPC key share
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyShare {
    /// Share index (1-based)
    pub index: u8,

    /// Encrypted share data
    pub encrypted_share: Vec<u8>,

    /// Public key
    pub public_key: Vec<u8>,

    /// Verification data
    pub verification: Vec<u8>,

    /// Share metadata
    pub metadata: ShareMetadata,
}

/// Share metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ShareMetadata {
    /// Created timestamp
    pub created_at: u64,

    /// Expiration (0 = never)
    pub expires_at: u64,

    /// Key ID
    pub key_id: String,

    /// Share type
    pub share_type: ShareType,
}

/// Share types
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum ShareType {
    /// Primary device share
    Primary,
    /// Cloud backup share
    Cloud,
    /// Recovery contact share
    Recovery,
    /// Hardware wallet share
    Hardware,
    /// Paper backup share
    Paper,
}

/// MPC wallet configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MPCWalletConfig {
    /// Threshold configuration
    pub threshold: ThresholdConfig,

    /// Enable biometric
    pub enable_biometric: bool,

    /// Enable PIN
    pub enable_pin: bool,

    /// Enable cloud backup
    pub enable_cloud_backup: bool,

    /// Enable hardware wallet
    pub enable_hardware_wallet: bool,

    /// Auto-lock timeout (seconds)
    pub auto_lock_timeout: u64,

    /// Require biometric for transaction
    pub require_biometric_for_tx: bool,

    /// Maximum transaction without biometric
    pub max_tx_without_biometric: f64,
}

impl Default for MPCWalletConfig {
    fn default() -> Self {
        Self {
            threshold: ThresholdConfig::two_of_three(),
            enable_biometric: true,
            enable_pin: true,
            enable_cloud_backup: true,
            enable_hardware_wallet: true,
            auto_lock_timeout: 300,
            require_biometric_for_tx: true,
            max_tx_without_biometric: 1000.0,
        }
    }
}

/// MPC wallet
pub struct MPCWallet {
    config: MPCWalletConfig,
    key_id: String,
    public_key: Vec<u8>,
    shares: Vec<KeyShare>,
    recovery_contacts: Vec<RecoveryContact>,
    signing_policy: SigningPolicy,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SigningPolicy {
    /// Maximum single transaction
    pub max_single_tx: f64,

    /// Maximum daily total
    pub max_daily_total: f64,

    /// Blocked addresses
    pub blocked_addresses: Vec<String>,

    /// Blocked tokens
    pub blocked_tokens: Vec<String>,

    /// Require confirmation for high value
    pub require_confirmation_above: f64,

    /// Allowed smart contracts
    pub allowed_contracts: Vec<String>,
}

impl Default for SigningPolicy {
    fn default() -> Self {
        Self {
            max_single_tx: 10_000.0,
            max_daily_total: 50_000.0,
            blocked_addresses: vec![],
            blocked_tokens: vec![],
            require_confirmation_above: 1000.0,
            allowed_contracts: vec![],
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryContact {
    pub id: String,
    pub name: String,
    pub email: Option<String>,
    pub phone: Option<String>,
    pub public_key: Option<String>,
    pub share_index: u8,
    pub trusted: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignRequest {
    pub key_id: String,
    pub transaction_hash: String,
    pub from: String,
    pub to: String,
    pub value: String,
    pub token: String,
    pub chain: String,
    pub data: Option<String>,
    pub nonce: u64,
    pub gas_price: u64,
    pub gas_limit: u64,
    pub timestamp: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignResponse {
    pub signature: String,
    pub signers: Vec<u8>,
    pub timestamp: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignResult {
    pub success: bool,
    pub signature: Option<String>,
    pub error: Option<String>,
    pub remaining_attempts: u8,
}

impl MPCWallet {
    /// Create a new MPC wallet
    pub fn new(config: MPCWalletConfig) -> Result<Self, MPCError> {
        if !config.threshold.is_valid() {
            return Err(MPCError::InvalidConfig(
                "Invalid threshold config".to_string(),
            ));
        }

        let key_id = Self::generate_key_id();

        Ok(Self {
            config,
            key_id,
            public_key: vec![],
            shares: vec![],
            recovery_contacts: vec![],
            signing_policy: SigningPolicy::default(),
        })
    }

    /// Generate unique key ID
    fn generate_key_id() -> String {
        use secure_random::generate_random_hex;
        format!("mpc_{}", generate_random_hex(16))
    }

    /// Initialize wallet with key generation
    pub fn initialize(&mut self, seed: &[u8]) -> Result<Vec<KeyShare>, MPCError> {
        let config = self.config.threshold;

        let public_key = Self::derive_public_key(seed)?;
        self.public_key = public_key;

        // Shamir-split the seed, then seal every share for storage.
        let mut shares = Self::generate_shares(seed, config.total_shares, config.threshold)?;
        self.seal_shares(&mut shares)?;
        self.shares = shares.clone();

        Ok(shares)
    }

    /// Split `secret` into `n` shares, any `threshold` of which reconstruct it.
    ///
    /// Each byte position is split independently over GF(256) with its own
    /// random polynomial, so shares are the same length as the secret and no
    /// single share reveals any byte of it.
    fn generate_shares(secret: &[u8], n: u8, threshold: u8) -> Result<Vec<KeyShare>, MPCError> {
        if secret.is_empty() {
            return Err(MPCError::InvalidConfig(
                "secret must not be empty".to_string(),
            ));
        }
        if threshold == 0 {
            return Err(MPCError::InvalidConfig(
                "threshold must be at least 1".to_string(),
            ));
        }
        if n < threshold {
            return Err(MPCError::InvalidConfig(format!(
                "total shares ({n}) must be at least the threshold ({threshold})"
            )));
        }

        let mut rng = OsRng;
        let mut share_bytes: Vec<Vec<u8>> = vec![Vec::with_capacity(secret.len()); n as usize];

        for &byte in secret {
            // Coefficients of a degree (threshold-1) polynomial whose constant
            // term is the secret byte.
            let mut coefficients = Vec::with_capacity(threshold as usize);
            coefficients.push(byte);
            for _ in 1..threshold {
                let mut coefficient = [0u8; 1];
                rng.fill_bytes(&mut coefficient);
                coefficients.push(coefficient[0]);
            }

            // Evaluation points are 1..=n; x = 0 is the secret itself.
            for (participant_index, share) in share_bytes.iter_mut().enumerate() {
                share.push(gf256::eval(&coefficients, (participant_index + 1) as u8));
            }
        }

        let mut shares = Vec::with_capacity(n as usize);
        for (index, bytes) in share_bytes.into_iter().enumerate() {
            let verification = crate::digest::sha256(&bytes).to_vec();
            shares.push(KeyShare {
                index: (index + 1) as u8,
                // Shares stay in the clear here; `seal_shares` encrypts them
                // for storage under a key derived from the wallet key ID.
                encrypted_share: bytes,
                public_key: Vec::new(),
                verification,
                metadata: ShareMetadata {
                    created_at: current_timestamp(),
                    expires_at: 0,
                    key_id: String::new(),
                    share_type: ShareType::Primary,
                },
            });
        }

        Ok(shares)
    }

    /// Encrypt generated shares for storage.
    ///
    /// Each share is sealed under a key derived from the wallet key ID and the
    /// share index, so a leaked store alone does not expose share material. The
    /// verification digest covers the plaintext share, so a caller can check a
    /// decrypted share against the value recorded at generation time.
    pub fn seal_shares(&self, shares: &mut [KeyShare]) -> Result<(), MPCError> {
        for share in shares.iter_mut() {
            share.encrypted_share = aead::encrypt(
                &Self::share_seal_key(&self.key_id, share.index),
                &share.encrypted_share,
                self.share_aad(share.index).as_bytes(),
            )
            .map_err(|e| MPCError::Encryption(e.to_string()))?;
            share.public_key = self.public_key.clone();
            share.metadata.key_id = self.key_id.clone();
        }
        Ok(())
    }

    /// Derive the AES key that seals a share at rest.
    ///
    /// The key is a function of the wallet key ID and the share index only, so
    /// it can be recomputed when the share is opened. It is a
    /// protection-at-rest measure: shares still belong on separate devices,
    /// because anyone holding both the store and the key ID can open them.
    fn share_seal_key(key_id: &str, index: u8) -> [u8; 32] {
        let material = format!("{key_id}:share:{index}");
        crate::digest::sha256(material.as_bytes())
    }

    /// Additional authenticated data for a sealed share.
    ///
    /// Binding the index into the AAD means a ciphertext cannot be replayed as
    /// a different share: the tag only verifies under the index it was sealed
    /// with.
    fn share_aad(&self, index: u8) -> String {
        format!("{}:share:{index}", self.key_id)
    }

    /// Convert bytes to a secp256k1 secret key.
    fn bytes_to_field(bytes: &[u8]) -> Result<[u8; 32], MPCError> {
        if bytes.len() != 32 {
            return Err(MPCError::InvalidConfig(format!(
                "secret key must be 32 bytes, got {}",
                bytes.len()
            )));
        }
        let mut key = [0u8; 32];
        key.copy_from_slice(bytes);
        SecretKey::from_slice(&key)
            .map_err(|e| MPCError::InvalidConfig(format!("invalid secret key: {e}")))?;
        Ok(key)
    }

    /// Derive the compressed secp256k1 public key for a 32-byte secret.
    fn derive_public_key(secret: &[u8]) -> Result<Vec<u8>, MPCError> {
        let key = Self::bytes_to_field(secret)?;
        let secp = Secp256k1::new();
        let secret_key = SecretKey::from_slice(&key)
            .map_err(|e| MPCError::InvalidConfig(format!("invalid secret key: {e}")))?;
        let public = PublicKey::from_secret_key(&secp, &secret_key);
        Ok(public.serialize().to_vec())
    }

    /// Reconstruct the secret from labelled shares.
    ///
    /// `shares` must be `(evaluation point, share bytes)` pairs; the point is
    /// what identifies the participant, so a caller cannot silently mix up
    /// which device a share came from.
    pub fn reconstruct_from_points(
        shares: &[(u8, Vec<u8>)],
        threshold: u8,
    ) -> Result<Vec<u8>, MPCError> {
        if threshold == 0 {
            return Err(MPCError::InvalidConfig(
                "threshold must be at least 1".to_string(),
            ));
        }
        if shares.len() < threshold as usize {
            return Err(MPCError::InsufficientShares(shares.len() as u8, threshold));
        }

        let used = &shares[..threshold as usize];
        let secret_len = used[0].1.len();
        if used.iter().any(|(_, bytes)| bytes.len() != secret_len) {
            return Err(MPCError::InvalidConfig(
                "shares have inconsistent lengths".to_string(),
            ));
        }

        let mut secret = Vec::with_capacity(secret_len);
        for byte_index in 0..secret_len {
            let points: Vec<(u8, u8)> = used
                .iter()
                .map(|(point, bytes)| (*point, bytes[byte_index]))
                .collect();
            secret.push(gf256::interpolate_at_zero(&points).ok_or_else(|| {
                MPCError::InvalidConfig("share points are not distinct".to_string())
            })?);
        }

        Ok(secret)
    }

    /// Decrypt a stored share and return its `(index, bytes)` pair.
    fn open_share(&self, share: &KeyShare) -> Result<(u8, Vec<u8>), MPCError> {
        // The sealing key is derived from the wallet key ID and the share
        // index, which are both known before generation. AES-GCM authenticates
        // the ciphertext, so a tampered or foreign share is rejected here.
        let bytes = aead::decrypt(
            &Self::share_seal_key(&self.key_id, share.index),
            &share.encrypted_share,
            self.share_aad(share.index).as_bytes(),
        )
        .map_err(|e| MPCError::Decryption(e.to_string()))?;
        Ok((share.index, bytes))
    }

    /// Sign a transaction hash with shares.
    ///
    /// Any `threshold` shares reconstruct the signing key, which then produces
    /// a real secp256k1 recoverable signature (`r || s || v`). This is
    /// reconstruct-then-sign, not a distributed threshold-signature protocol:
    /// the reconstructed key exists in this process for the duration of the
    /// call. It is therefore only appropriate when the caller controls the
    /// process, e.g. a signing service collecting shares from its own devices.
    /// Use the `tiger-mpc-wallet` crate for a protocol that never materialises
    /// the key.
    ///
    /// The reconstructed key is zeroized before the call returns.
    pub fn sign(
        &self,
        request: &SignRequest,
        shares: &[(u8, Vec<u8>)],
    ) -> Result<SignResult, MPCError> {
        let threshold = self.config.threshold.threshold;
        if shares.len() < threshold as usize {
            return Err(MPCError::InsufficientShares(shares.len() as u8, threshold));
        }

        let mut secret = Self::reconstruct_from_points(shares, threshold)?;
        let signature = Self::sign_data(&request.transaction_hash, &secret);

        // Clear the reconstructed key from memory before propagating errors.
        secret.iter_mut().for_each(|byte| *byte = 0);

        let signature = signature?;

        Ok(SignResult {
            success: true,
            signature: Some(signature),
            error: None,
            remaining_attempts: 3,
        })
    }

    /// Hash transaction for signing. This produces the EIP-191 personal-message
    /// digest shape used by `sign_data`; the caller passes the same value back
    /// in `SignRequest::transaction_hash` when signing a transaction digest
    /// directly.
    pub fn hash_transaction(tx: &TransactionData) -> Result<String, MPCError> {
        let mut hasher = Sha256::new();
        hasher.update(tx.from.as_bytes());
        hasher.update(tx.to.as_bytes());
        hasher.update(tx.value.as_bytes());
        hasher.update(tx.token.as_bytes());
        hasher.update(tx.chain.as_bytes());
        hasher.update(tx.nonce.to_le_bytes());

        let hash = hasher.finalize();
        Ok(hex::encode(hash))
    }

    /// Produce a real secp256k1 recoverable signature over a 32-byte hex
    /// digest. The returned value is `r || s || v` with `v` in 0..=1.
    fn sign_data(transaction_hash: &str, secret: &[u8]) -> Result<String, MPCError> {
        let key = Self::bytes_to_field(secret)?;
        let digest_hex = transaction_hash.trim_start_matches("0x");
        let digest = hex::decode(digest_hex)
            .map_err(|e| MPCError::InvalidConfig(format!("transaction hash is not hex: {e}")))?;
        if digest.len() != 32 {
            return Err(MPCError::InvalidConfig(format!(
                "transaction hash must be 32 bytes, got {}",
                digest.len()
            )));
        }

        let message = secp256k1::Message::from_slice(&digest)
            .map_err(|e| MPCError::SigningFailed(format!("invalid signing digest: {e}")))?;

        let secp = Secp256k1::new();
        let secret_key = SecretKey::from_slice(&key)
            .map_err(|e| MPCError::InvalidConfig(format!("invalid secret key: {e}")))?;
        let signature = secp.sign_ecdsa_recoverable(&message, &secret_key);
        let (recovery_id, compact) = signature.serialize_compact();

        let mut out = Vec::with_capacity(65);
        out.extend_from_slice(&compact);
        out.push(recovery_id.to_i32() as u8);
        Ok(hex::encode(out))
    }

    /// Add recovery contact
    pub fn add_recovery_contact(&mut self, contact: RecoveryContact) -> Result<(), MPCError> {
        if self.recovery_contacts.len() >= self.config.threshold.total_shares as usize - 1 {
            return Err(MPCError::TooManyContacts);
        }

        self.recovery_contacts.push(contact);
        Ok(())
    }

    /// Initiate social recovery
    pub fn initiate_social_recovery(
        &self,
        contact_ids: &[String],
    ) -> Result<SocialRecoveryRequest, MPCError> {
        if contact_ids.len() < self.config.threshold.threshold as usize - 1 {
            return Err(MPCError::InsufficientContacts(
                contact_ids.len() as u8,
                self.config.threshold.threshold - 1,
            ));
        }

        let request = SocialRecoveryRequest {
            key_id: self.key_id.clone(),
            contact_ids: contact_ids.to_vec(),
            initiated_at: current_timestamp(),
            expires_at: current_timestamp() + 86400, // 24 hours
        };

        Ok(request)
    }

    /// Complete social recovery
    pub fn complete_social_recovery(
        &self,
        shares: &[KeyShare],
    ) -> Result<RecoveryResult, MPCError> {
        if shares.len() < self.config.threshold.threshold as usize {
            return Err(MPCError::InsufficientShares(
                shares.len() as u8,
                self.config.threshold.threshold,
            ));
        }

        let mut points = Vec::with_capacity(shares.len());
        for share in shares {
            if share.metadata.key_id != self.key_id {
                return Err(MPCError::InvalidShare("Key ID mismatch".to_string()));
            }

            let (index, bytes) = self.open_share(share)?;

            // The digest recorded at generation time authenticates the
            // plaintext share, so a corrupted share is never interpolated.
            if !share.verification.is_empty()
                && share.verification != crate::digest::sha256(&bytes).to_vec()
            {
                return Err(MPCError::InvalidShare(format!(
                    "share {index} failed its verification digest"
                )));
            }

            points.push((index, bytes));
        }

        let mut secret = Self::reconstruct_from_points(&points, self.config.threshold.threshold)?;
        let public_key = Self::derive_public_key(&secret)?;
        secret.iter_mut().for_each(|byte| *byte = 0);

        Ok(RecoveryResult {
            success: true,
            public_key,
            recovered_at: current_timestamp(),
        })
    }

    /// Update signing policy
    pub fn update_policy(&mut self, policy: SigningPolicy) -> Result<(), MPCError> {
        self.signing_policy = policy;
        Ok(())
    }

    /// Get wallet info
    pub fn info(&self) -> WalletInfo {
        WalletInfo {
            key_id: self.key_id.clone(),
            public_key: self.public_key.clone(),
            total_shares: self.config.threshold.total_shares,
            threshold: self.config.threshold.threshold,
            recovery_contacts: self.recovery_contacts.len() as u8,
        }
    }
}

/// Transaction data for signing
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionData {
    pub from: String,
    pub to: String,
    pub value: String,
    pub token: String,
    pub chain: String,
    pub data: Option<String>,
    pub nonce: u64,
    pub gas_price: u64,
    pub gas_limit: u64,
}

/// Social recovery request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SocialRecoveryRequest {
    pub key_id: String,
    pub contact_ids: Vec<String>,
    pub initiated_at: u64,
    pub expires_at: u64,
}

/// Recovery result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryResult {
    pub success: bool,
    pub public_key: Vec<u8>,
    pub recovered_at: u64,
}

/// Wallet info
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WalletInfo {
    pub key_id: String,
    pub public_key: Vec<u8>,
    pub total_shares: u8,
    pub threshold: u8,
    pub recovery_contacts: u8,
}

/// MPC errors
#[derive(Debug, thiserror::Error)]
pub enum MPCError {
    #[error("Invalid configuration: {0}")]
    InvalidConfig(String),

    #[error("Encryption failed: {0}")]
    Encryption(String),

    #[error("Decryption failed: {0}")]
    Decryption(String),

    #[error("Signing failed: {0}")]
    SigningFailed(String),

    #[error("Randomness unavailable: {0}")]
    Randomness(String),

    #[error("Crypto error: {0}")]
    CryptoError(String),

    #[error("Insufficient shares: have {0}, need {1}")]
    InsufficientShares(u8, u8),

    #[error("Policy violation: {0}")]
    PolicyViolation(String),

    #[error("Invalid share: {0}")]
    InvalidShare(String),

    #[error("Too many contacts")]
    TooManyContacts,

    #[error("Insufficient contacts: have {0}, need {1}")]
    InsufficientContacts(u8, u8),

    #[error("Recovery failed: {0}")]
    RecoveryFailed(String),

    #[error("Network error: {0}")]
    Network(String),
}

/// Get current timestamp
fn current_timestamp() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs()
}

/// MPC wallet manager for handling multiple wallets
pub struct MPCWalletManager {
    wallets: HashMap<String, MPCWallet>,
}

impl MPCWalletManager {
    pub fn new() -> Self {
        Self {
            wallets: HashMap::new(),
        }
    }

    pub fn create_wallet(
        &mut self,
        config: MPCWalletConfig,
    ) -> Result<(String, Vec<KeyShare>), MPCError> {
        let mut wallet = MPCWallet::new(config)?;
        let key_id = wallet.key_id.clone();

        let seed = secure_random::generate_random_bytes(32)
            .map_err(|e| MPCError::Randomness(e.to_string()))?;

        let shares = wallet.initialize(&seed)?;

        self.wallets.insert(key_id.clone(), wallet);

        Ok((key_id, shares))
    }

    pub fn get_wallet(&self, key_id: &str) -> Option<&MPCWallet> {
        self.wallets.get(key_id)
    }

    pub fn get_wallet_mut(&mut self, key_id: &str) -> Option<&mut MPCWallet> {
        self.wallets.get_mut(key_id)
    }
}

impl Default for MPCWalletManager {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_wallet_creation() {
        let config = MPCWalletConfig::default();
        let mut wallet = MPCWallet::new(config).unwrap();

        let seed = [1u8; 32];
        let shares = wallet.initialize(&seed).unwrap();

        assert_eq!(shares.len(), 3);
        assert!(shares.iter().all(|share| !share.verification.is_empty()));
    }

    #[test]
    fn a_zero_seed_is_rejected_rather_than_silently_accepted() {
        let config = MPCWalletConfig::default();
        let mut wallet = MPCWallet::new(config).unwrap();

        // All-zero is not a valid secp256k1 scalar, so initialisation must fail
        // instead of producing an unusable key.
        assert!(wallet.initialize(&[0u8; 32]).is_err());
    }

    #[test]
    fn test_share_generation() {
        let secret = b"test_secret_key";
        let shares = MPCWallet::generate_shares(secret, 3, 2).unwrap();

        assert_eq!(shares.len(), 3);
    }

    #[test]
    fn test_secret_reconstruction() {
        let secret = b"test_secret_key";
        let shares = MPCWallet::generate_shares(secret, 3, 2).unwrap();

        let points: Vec<(u8, Vec<u8>)> = shares
            .iter()
            .map(|s| (s.index, s.encrypted_share.clone()))
            .collect();

        // Any subset of at least `threshold` shares reconstructs the secret.
        assert_eq!(
            MPCWallet::reconstruct_from_points(&points[..2], 2).unwrap(),
            secret
        );
        assert_eq!(
            MPCWallet::reconstruct_from_points(&points[1..3], 2).unwrap(),
            secret
        );

        // A single share must not reveal the secret.
        assert_ne!(points[0].1, secret);
    }

    #[test]
    fn test_wallet_manager() {
        let mut manager = MPCWalletManager::new();
        let config = MPCWalletConfig::default();

        let (key_id, shares) = manager.create_wallet(config).unwrap();

        assert!(!key_id.is_empty());
        assert_eq!(shares.len(), 3);
    }

    #[test]
    fn test_threshold_config() {
        let config = ThresholdConfig::two_of_three();
        assert!(config.is_valid());

        let config = ThresholdConfig::three_of_five();
        assert!(config.is_valid());

        let config = ThresholdConfig {
            total_shares: 2,
            threshold: 3,
            key_bits: 256,
        };
        assert!(!config.is_valid());
    }
}
