//! TigerWallet MPC (Multi-Party Computation) Wallet
//! 
//! Production-ready implementation of Threshold Signature Scheme (TSS)
//! This provides MPC wallet functionality similar to Bitget Wallet
//!
//! Features:
//! - Key generation with threshold (t-of-n)
//! - Distributed key signing
//! - Key resharing
//! - Key recovery
//! - Support for ECDSA (Bitcoin, Ethereum) and Ed25519 (Solana, etc.)

#![allow(dead_code)]

pub mod key_gen;
pub mod signing;
pub mod key_sharing;
pub mod wallet;

pub use key_gen::*;
pub use signing::*;
pub use key_sharing::*;
pub use wallet::*;

use serde::{Deserialize, Serialize};

/// MPC Wallet configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MpcConfig {
    /// Threshold required for signing (t)
    pub threshold: u32,
    /// Total number of shares (n)
    pub total_shares: u32,
    /// Curve type
    pub curve: CurveType,
    /// Key ID for tracking
    pub key_id: String,
}

impl Default for MpcConfig {
    fn default() -> Self {
        Self {
            threshold: 2,
            total_shares: 3,
            curve: CurveType::Secp256k1,
            key_id: String::new(),
        }
    }
}

/// Supported curves
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum CurveType {
    /// secp256k1 (Bitcoin, Ethereum)
    Secp256k1,
    /// Ed25519 (Solana, Aptos)
    Ed25519,
    /// P-256
    P256,
}

impl CurveType {
    pub fn key_length(&self) -> usize {
        match self {
            CurveType::Secp256k1 => 32,
            CurveType::Ed25519 => 32,
            CurveType::P256 => 32,
        }
    }
    
    pub fn signature_length(&self) -> usize {
        match self {
            CurveType::Secp256k1 => 64,
            CurveType::Ed25519 => 64,
            CurveType::P256 => 64,
        }
    }
}

/// Share information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ShareInfo {
    /// Share index (1-based)
    pub index: u32,
    /// Share data (encrypted)
    pub share_data: Vec<u8>,
    /// Public verification key
    pub verification_key: Vec<u8>,
}

/// Key generation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyGenResult {
    /// Generated key ID
    pub key_id: String,
    /// Public key
    pub public_key: Vec<u8>,
    /// Array of shares (one per participant)
    pub shares: Vec<ShareInfo>,
    /// Backup encrypted key (for recovery)
    pub backup: Vec<u8>,
}

/// Signing result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignResult {
    /// Signature
    pub signature: Vec<u8>,
    /// Public key used
    pub public_key: Vec<u8>,
    /// Key ID
    pub key_id: String,
}

/// MPC errors
#[derive(Debug, thiserror::Error)]
pub enum MpcError {
    #[error("Key generation failed: {0}")]
    KeyGenFailed(String),
    
    #[error("Signing failed: {0}")]
    SigningFailed(String),
    
    #[error("Key sharing failed: {0}")]
    KeySharingFailed(String),
    
    #[error("Invalid share: {0}")]
    InvalidShare(String),
    
    #[error("Threshold not met: {0}")]
    ThresholdNotMet(String),
    
    #[error("Invalid configuration: {0}")]
    InvalidConfig(String),
    
    #[error("Encryption error: {0}")]
    EncryptionError(String),
    
    #[error("Network error: {0}")]
    NetworkError(String),
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_mpc_config_default() {
        let config = MpcConfig::default();
        assert_eq!(config.threshold, 2);
        assert_eq!(config.total_shares, 3);
        assert_eq!(config.curve, CurveType::Secp256k1);
    }
    
    #[test]
    fn test_curve_key_lengths() {
        assert_eq!(CurveType::Secp256k1.key_length(), 32);
        assert_eq!(CurveType::Ed25519.key_length(), 32);
        assert_eq!(CurveType::P256.key_length(), 32);
    }
}
