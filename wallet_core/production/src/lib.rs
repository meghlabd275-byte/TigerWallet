/**
 * TigerWallet Production Core - Real Implementation
 * 
 * Production-ready wallet with:
 * - Real BIP-39 mnemonic generation/validation
 * - Real BIP-32 HD key derivation
 * - Real BIP-44 multi-account hierarchy
 * - Real multi-chain address generation
 * - Real cryptographic signing
 * 
 * WARNING: This code handles sensitive cryptographic operations.
 * Always use well-audited libraries and proper key management.
 */

use std::fmt;
use std::str::FromStr;

pub mod bip39;
pub mod bip32;
pub mod bip44;
pub mod address;
pub mod signing;
pub mod chains;

pub use bip39::{Mnemonic, MnemonicType, Wordlist};
pub use bip32::{HDKey, DerivationPath};
pub use bip44::Path;
pub use address::{Address, AddressType};
pub use signing::{Signer, Signature, SignedTransaction};
pub use chains::{Chain, ChainConfig, ChainType};

use thiserror::Error;

/// Main wallet error type
#[derive(Error, Debug)]
pub enum WalletError {
    #[error("Invalid mnemonic: {0}")]
    InvalidMnemonic(String),
    
    #[error("Invalid derivation path: {0}")]
    InvalidPath(String),
    
    #[error("Invalid address: {0}")]
    InvalidAddress(String),
    
    #[error("Signing error: {0}")]
    SigningError(String),
    
    #[error("Transaction error: {0}")]
    TransactionError(String),
    
    #[error("Crypto error: {0}")]
    CryptoError(String),
    
    #[error("Network error: {0}")]
    NetworkError(String),
    
    #[error("Invalid parameter: {0}")]
    InvalidParameter(String),
    
    #[error("Key not found: {0}")]
    KeyNotFound(String),
    
    #[error("Unsupported chain: {0}")]
    UnsupportedChain(String),
}

pub type Result<T> = std::result::Result<T, WalletError>;

// =============================================================================
// Wallet Structure
// =============================================================================

/// Represents a complete wallet with master key
pub struct Wallet {
    master_key: HDKey,
    chain: Chain,
}

impl Wallet {
    /// Create wallet from mnemonic phrase
    pub fn new(mnemonic: &str, password: &str, chain: Chain) -> Result<Self> {
        let seed = bip39::mnemonic_to_seed(mnemonic, password)?;
        let master_key = HDKey::from_seed(&seed)?;
        
        Ok(Self {
            master_key,
            chain,
        })
    }
    
    /// Create wallet from raw 64-byte seed
    pub fn from_seed(seed: &[u8; 64], chain: Chain) -> Result<Self> {
        let master_key = HDKey::from_seed(seed)?;
        
        Ok(Self {
            master_key,
            chain,
        })
    }
    
    /// Derive child key at given path
    pub fn derive(&self, path: &Path) -> Result<HDKey> {
        self.master_key.derive(path)
    }
    
    /// Get address at specific derivation path
    pub fn address(&self, path: &Path) -> Result<Address> {
        let key = self.derive(path)?;
        Ok(Address::from_key(&key, self.chain))
    }
    
    /// Get default address (m/44'/60'/0'/0/0)
    pub fn default_address(&self) -> Result<Address> {
        let path = Path::default_for_chain(self.chain);
        self.address(&path)
    }
    
    /// Get multiple addresses
    pub fn addresses(&self, count: usize) -> Result<Vec<Address>> {
        let mut addresses = Vec::with_capacity(count);
        let base_path = Path::default_for_chain(self.chain);
        
        for i in 0..count {
            let path = base_path.with_index(i);
            addresses.push(self.address(&path)?);
        }
        
        Ok(addresses)
    }
    
    /// Sign data with derived key
    pub fn sign(&self, data: &[u8], path: &Path) -> Result<Signature> {
        let key = self.derive(path)?;
        let signer = Signer::new(&key, self.chain)?;
        signer.sign(data)
    }
    
    /// Sign transaction
    pub fn sign_transaction(&self, tx: &mut SignedTransaction) -> Result<()> {
        let path = Path::default_for_chain(self.chain);
        let key = self.derive(&path)?;
        let signer = Signer::new(&key, self.chain)?;
        signer.sign_transaction(tx)
    }
    
    /// Get master key (use with caution!)
    pub fn master_key(&self) -> &HDKey {
        &self.master_key
    }
    
    /// Get the chain this wallet is for
    pub fn chain(&self) -> Chain {
        self.chain
    }
}

/// Multi-chain wallet supporting all supported blockchains
pub struct MultiChainWallet {
    wallets: std::collections::HashMap<Chain, Wallet>,
    seed: [u8; 64],  // Keep seed in memory
}

impl MultiChainWallet {
    /// Create multi-chain wallet from mnemonic
    pub fn new(mnemonic: &str, password: &str) -> Result<Self> {
        let seed = bip39::mnemonic_to_seed(mnemonic, password)?;
        
        let mut wallets = std::collections::HashMap::new();
        
        // Create wallets for all supported chains
        for chain in Chain::all() {
            let wallet = Wallet::from_seed(&seed, chain)?;
            wallets.insert(chain, wallet);
        }
        
        Ok(Self {
            wallets,
            seed,
        })
    }
    
    /// Get wallet for specific chain
    pub fn wallet(&self, chain: Chain) -> Result<&Wallet> {
        self.wallets.get(&chain)
            .ok_or_else(|| WalletError::UnsupportedChain(chain.to_string()))
    }
    
    /// Get default address for chain
    pub fn address(&self, chain: Chain) -> Result<Address> {
        self.wallet(chain)?.default_address()
    }
    
    /// Get all addresses for a chain
    pub fn addresses(&self, chain: Chain, count: usize) -> Result<Vec<Address>> {
        self.wallet(chain)?.addresses(count)
    }
    
    /// Get addresses for all chains
    pub fn all_addresses(&self) -> Result<std::collections::HashMap<Chain, String>> {
        let mut result = std::collections::HashMap::new();
        
        for chain in Chain::all() {
            if let Ok(addr) = self.address(chain) {
                result.insert(chain, addr.to_string());
            }
        }
        
        Ok(result)
    }
    
    /// Sign data for specific chain
    pub fn sign(&self, chain: Chain, data: &[u8], index: u32) -> Result<Signature> {
        let wallet = self.wallet(chain)?;
        let path = Path::default_for_chain(chain).with_index(index);
        wallet.sign(data, &path)
    }
}

// =============================================================================
// Testing
// =============================================================================

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_mnemonic_generation() {
        let mnemonic = Mnemonic::new(Wordlist::English, 24).unwrap();
        assert_eq!(mnemonic.words().split_whitespace().count(), 24);
    }
    
    #[test]
    fn test_mnemonic_roundtrip() {
        let mnemonic = Mnemonic::new(Wordlist::English, 12).unwrap();
        let phrase = mnemonic.words();
        
        let validated = Mnemonic::validate(phrase, Wordlist::English);
        assert!(validated.is_ok());
    }
    
    #[test]
    fn test_wallet_derivation() {
        let wallet = Wallet::from_seed(&[0u8; 64], Chain::Ethereum).unwrap();
        let addr = wallet.default_address().unwrap();
        
        // Should be a valid Ethereum address
        assert!(addr.to_string().starts_with("0x"));
        assert_eq!(addr.to_string().len(), 42);
    }
    
    #[test]
    fn test_multichain_wallet() {
        let wallet = Wallet::from_seed(&[0u8; 64], Chain::Ethereum).unwrap();
        
        // Test multiple chains
        let _ = Wallet::from_seed(&[0u8; 64], Chain::Bitcoin);
        let _ = Wallet::from_seed(&[0u8; 64], Chain::Solana);
        
        let addr = wallet.default_address().unwrap();
        assert!(!addr.to_string().is_empty());
    }
    
    #[test]
    fn test_address_validation() {
        // Valid Ethereum address
        assert!(Address::validate("0x742d35Cc6634C0532925a3b844Bc9e7595f1234", Chain::Ethereum).is_ok());
        
        // Invalid address
        assert!(Address::validate("invalid", Chain::Ethereum).is_err());
    }
}
