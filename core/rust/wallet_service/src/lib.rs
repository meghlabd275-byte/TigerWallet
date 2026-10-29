//! TigerSwap Wallet Service - Production-Ready Rust Wallet API
//! 
//! Complete wallet implementation in Rust with:
//! - HD Wallet generation (BIP39, BIP32, BIP44)
//! - Multi-chain address derivation
//! - Transaction signing
//! - Multi-signature support
//! - Account abstraction
//! - Key vault with HSM integration

use aes_gcm::aead::{OsRng, rand_core::RngCore};
use thiserror::Error;
use zeroize::{Zeroize, ZeroizeOnDrop};

// ============================================================================
// Error Types
// ============================================================================

#[derive(Error, Debug)]
pub enum WalletError {
    #[error("Invalid mnemonic phrase")]
    InvalidMnemonic,
    #[error("Invalid derivation path")]
    InvalidDerivationPath,
    #[error("Key derivation failed: {0}")]
    DerivationFailed(String),
    #[error("Invalid address format")]
    InvalidAddress,
    #[error("Signing failed: {0}")]
    SigningFailed(String),
    #[error("Invalid signature")]
    InvalidSignature,
    #[error("Encryption failed: {0}")]
    EncryptionFailed(String),
    #[error("Decryption failed: {0}")]
    DecryptionFailed(String),
    #[error("Wallet not found")]
    WalletNotFound,
    #[error("Invalid chain")]
    InvalidChain,
}

// ============================================================================
// Cryptographic Types
// ============================================================================

/// 256-bit key with secure memory handling
#[derive(Clone, Zeroize, ZeroizeOnDrop)]
pub struct Key([u8; 32]);

impl Key {
    pub fn new() -> Self {
        let mut key = [0u8; 32];
        OsRng.fill_bytes(&mut key);
        Self(key)
    }
    
    pub fn from_bytes(bytes: [u8; 32]) -> Self {
        Self(bytes)
    }
    
    pub fn as_bytes(&self) -> &[u8; 32] {
        &self.0
    }
    
    pub fn to_hex(&self) -> String {
        hex::encode(self.0)
    }
}

impl Default for Key {
    fn default() -> Self {
        Self::new()
    }
}
// ============================================================================
// Supported Chains
// ============================================================================


// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_mnemonic_generation() {
        let mnemonic = Mnemonic::generate(256).unwrap();
        assert_eq!(mnemonic.words().len(), 24);
        assert!(!mnemonic.phrase().is_empty());
    }
    
    #[test]
    fn test_mnemonic_parsing() {
        let phrase = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";
        let mnemonic = Mnemonic::from_phrase(phrase).unwrap();
        assert_eq!(mnemonic.words().len(), 12);
    }
    
    #[test]
    fn test_wallet_generation() {
        let wallet = HDWallet::generate(Chain::Ethereum);
        let address = wallet.default_address().unwrap();
        assert!(address.starts_with("0x"));
        assert_eq!(address.len(), 42);
    }
    
    #[test]
    fn test_bip44_derivation() {
        let wallet = HDWallet::generate(Chain::Ethereum);
        
        let addr1 = wallet.derive_address(44, 0, 0).unwrap();
        let addr2 = wallet.derive_address(44, 0, 1).unwrap();
        
        assert_ne!(addr1, addr2);
    }
    
    #[tokio::test]
    async fn test_wallet_manager() {
        let manager = WalletManager::new();
        
        let wallet = manager.create_wallet("Test Wallet".to_string(), Chain::Ethereum).await.unwrap();
        
        assert!(!wallet.wallet_id.is_empty());
        assert!(!wallet.address.is_empty());
    }
    
    #[tokio::test]
    async fn test_sign_transaction() {
        let manager = WalletManager::new();
        
        let wallet = manager.create_wallet("Test Wallet".to_string(), Chain::Ethereum).await.unwrap();
        
        let signature = manager.sign_transaction(&wallet.wallet_id, b"test message").await.unwrap();
        
        assert_eq!(signature.len(), 130);
    }
}

// ============================================================================
// Library Exports
// ============================================================================

pub use self::{
    mnemonic::{
        entropy_to_phrase, get_bip39_wordlist, phrase_to_entropy, phrase_to_seed, Mnemonic,
    },
    wallet::{bitcoin_p2pkh, eip191_digest, evm_address, Chain, HDWallet},
    manager::{Balance, TokenBalance, WalletInfo, WalletManager, WalletRecord},
    api_types::{
        CreateWalletRequest, ErrorResponse, HealthResponse, ImportWalletRequest, SignRequest,
        SignResponse, WalletResponse,
    },
};

mod mnemonic;
mod wallet;
mod manager;
mod api_types;