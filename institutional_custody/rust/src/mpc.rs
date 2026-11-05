//! MPC (Multi-Party Computation) module for Institutional Custody

use crate::{CustodyError, Role};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;

/// MPC key share
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyShare {
    pub share_id: String,
    pub public_key: Vec<u8>,
    pub share: Vec<u8>,
    pub threshold: u32,
    pub participants: u32,
    pub created_at: u64,
}

/// MPC key pair
#[derive(Debug, Clone)]
pub struct KeyPair {
    pub public_key: Vec<u8>,
    pub key_shares: Vec<KeyShare>,
    pub threshold: u32,
}

/// MPC signature
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MpcSignature {
    pub signature: Vec<u8>,
    pub signers: Vec<String>,
    pub message_hash: Vec<u8>,
    pub timestamp: u64,
}

/// MPC service
pub struct MpcService {
    /// Key pairs
    key_pairs: HashMap<String, KeyPair>,
    /// Pending signatures
    pending_signatures: HashMap<String, Vec<SignatureShare>>,
    /// Completed signatures
    completed_signatures: HashMap<String, MpcSignature>,
    /// Security threshold
    security_threshold: u32,
}

/// Smallest plausible partial signature. A raw ECDSA P-256 signature is 64
/// bytes; anything shorter is not one.
const MIN_PARTIAL_SIGNATURE_LEN: usize = 64;

/// Derives an Ethereum address from an uncompressed or compressed public key.
pub fn address_from_public_key(public_key: &[u8]) -> String {
    use sha3::{Digest, Keccak256};

    // Strip the 0x04 uncompressed prefix if present; keccak is taken over the
    // raw x||y coordinates.
    let body = match public_key.first() {
        Some(0x04) => &public_key[1..],
        _ => public_key,
    };
    let hash = Keccak256::digest(body);
    format!("0x{}", hex::encode(&hash[12..]))
}

/// Signature share
#[derive(Debug, Clone)]
pub struct SignatureShare {
    pub signer: String,
    pub share: Vec<u8>,
}

impl MpcService {
    pub fn new() -> Self {
        Self {
            key_pairs: HashMap::new(),
            pending_signatures: HashMap::new(),
            completed_signatures: HashMap::new(),
            security_threshold: 2,
        }
    }

    /// Generates the key material for `wallet` and records it.
    ///
    /// The previous version built a key pair, returned the share, and stored
    /// nothing: `sign` and `verify` both look the wallet up in `key_pairs`, so
    /// every signing request for a freshly created custody wallet failed with
    /// "No key pair". The share is now persisted under the wallet address, and
    /// the address is derived from the public key so the key and the address
    /// cannot drift apart.
    pub async fn generate_key_share(
        &mut self,
        wallet: &str,
    ) -> Result<KeyShare, CustodyError> {
        let rng = ring::rand::SystemRandom::new();

        let pkcs8 = ring::signature::EcdsaKeyPair::generate_pkcs8(
            &ring::signature::ECDSA_P256_SHA256_FIXED_SIGNING,
            &rng,
        )
        .map_err(|e| CustodyError::MpcError(e.to_string()))?;

        let key_pair = ring::signature::EcdsaKeyPair::from_pkcs8(
            &ring::signature::ECDSA_P256_SHA256_FIXED_SIGNING,
            pkcs8.as_ref(),
            &rng,
        )
        .map_err(|e| CustodyError::MpcError(e.to_string()))?;

        let public_key = ring::signature::KeyPair::public_key(&key_pair).as_ref().to_vec();

        let share = KeyShare {
            share_id: wallet.to_string(),
            public_key: public_key.clone(),
            share: pkcs8.as_ref().to_vec(),
            threshold: self.security_threshold,
            participants: self.security_threshold + 1,
            created_at: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs(),
        };

        self.key_pairs.insert(
            wallet.to_string(),
            KeyPair {
                public_key,
                key_shares: vec![share.clone()],
                threshold: self.security_threshold,
            },
        );

        Ok(share)
    }

    /// Re-keys a generated key from a temporary id to the wallet address it
    /// belongs to, so lookups by address find the key material.
    pub fn bind_address(&mut self, from: &str, address: &str) -> Result<(), CustodyError> {
        let key_pair = self
            .key_pairs
            .remove(from)
            .ok_or_else(|| CustodyError::MpcError(format!("no key material for {}", from)))?;
        self.key_pairs.insert(address.to_string(), key_pair);
        Ok(())
    }

    /// The public key generated for `wallet`, if any.
    pub fn public_key_for(&self, wallet: &str) -> Option<Vec<u8>> {
        self.key_pairs.get(wallet).map(|kp| kp.public_key.clone())
    }

    /// The Ethereum address of `wallet`'s public key: the last 20 bytes of the
    /// keccak-256 of the uncompressed key.
    pub fn address_for(&self, wallet: &str) -> Option<String> {
        let public_key = self.public_key_for(wallet)?;
        Some(address_from_public_key(&public_key))
    }

    /// Signs `message` with the wallet's key.
    ///
    /// A single-share signature is produced here; a real deployment would
    /// gather a threshold of partial signatures first. The signature returned
    /// is a genuine ECDSA signature over `message` under the wallet's public
    /// key, so it verifies.
    pub async fn sign(&self, wallet: &str, message: &[u8]) -> Result<Vec<u8>, CustodyError> {
        let key_pair = self
            .key_pairs
            .get(wallet)
            .ok_or_else(|| CustodyError::MpcError(format!("no key material for {}", wallet)))?;

        let rng = ring::rand::SystemRandom::new();

        let share = key_pair
            .key_shares
            .first()
            .ok_or_else(|| CustodyError::MpcError("wallet has no key shares".to_string()))?;

        let signing_key = ring::signature::EcdsaKeyPair::from_pkcs8(
            &ring::signature::ECDSA_P256_SHA256_FIXED_SIGNING,
            &share.share,
            &rng,
        )
        .map_err(|e| CustodyError::MpcError(e.to_string()))?;

        let signature = signing_key
            .sign(&rng, message)
            .map_err(|e| CustodyError::MpcError(e.to_string()))?;

        Ok(signature.as_ref().to_vec())
    }

    /// Verify a signature against the wallet's public key.
    ///
    /// This previously returned Ok(true) for any non-empty byte string, so
    /// every signature verified. That is the worst possible failure mode for
    /// a custody service: a forged signature would have been accepted. It now
    /// performs a real ECDSA P-256 verification with the key the wallet was
    /// created with, and reports an error rather than success when it cannot
    /// check.
    pub async fn verify(
        &self,
        wallet: &str,
        message: &[u8],
        signature: &[u8],
    ) -> Result<bool, CustodyError> {
        let key_pair = self
            .key_pairs
            .get(wallet)
            .ok_or_else(|| CustodyError::MpcError(format!("no key material for {}", wallet)))?;

        let public_key = ring::signature::UnparsedPublicKey::new(
            &ring::signature::ECDSA_P256_SHA256_FIXED,
            &key_pair.public_key,
        );

        Ok(public_key.verify(message, signature).is_ok())
    }

    /// Add signature share
    pub async fn add_signature_share(
        &mut self,
        wallet: &str,
        request_id: &str,
        signer: &str,
        share: Vec<u8>,
    ) -> Result<bool, CustodyError> {
        let shares = self.pending_signatures
            .entry(wallet.to_string())
            .or_insert_with(Vec::new);

        shares.push(SignatureShare {
            signer: signer.to_string(),
            share,
        });

        // Check if threshold reached
        let key_pair = self.key_pairs.get(wallet)
            .ok_or(CustodyError::MpcError("No key pair".to_string()))?;

        Ok(shares.len() >= key_pair.threshold as usize)
    }

    /// Combines signature shares into a threshold signature.
    ///
    /// Aggregating ECDSA shares requires the real multi-round threshold
    /// protocol, which is implemented in `rust/mpc` (Shamir over the curve,
    /// Feldman VSS, Lagrange-weighted partial signatures). This module
    /// previously returned the first party's share as the combined signature
    /// and reported success, which produces a value that verifies against
    /// nothing. It now fails closed and points at the real implementation.
    pub async fn combine_shares(
        &self,
        wallet: &str,
        request_id: &str,
        _message: &[u8],
    ) -> Result<MpcSignature, CustodyError> {
        let shares = self
            .pending_signatures
            .get(wallet)
            .ok_or_else(|| CustodyError::MpcError(format!("no pending shares for {}", wallet)))?;

        if shares.is_empty() {
            return Err(CustodyError::MpcError("no shares".to_string()));
        }

        // Each entry must be a partial signature produced by the threshold
        // protocol, not a raw key share.
        for share in shares {
            if share.share.len() < MIN_PARTIAL_SIGNATURE_LEN {
                return Err(CustodyError::MpcError(format!(
                    "share from {} is {} bytes, too short to be a partial signature",
                    share.signer,
                    share.share.len()
                )));
            }
        }

        Err(CustodyError::MpcError(format!(
            "threshold aggregation for request {} is not performed by this module; \
             use the tiger-mpc-wallet crate (rust/mpc)",
            request_id
        )))
    }

    /// Get public key
    pub async fn get_public_key(&self, wallet: &str) -> Option<Vec<u8>> {
        self.key_pairs.get(wallet)
            .map(|kp| kp.public_key.clone())
    }
}

impl Default for MpcService {
    fn default() -> Self {
        Self::new()
    }
}