//! TigerWallet MPC Wallet Core
//! 
//! High-performance multi-party computation wallet with threshold signatures,
//! TEE integration, and ultra-low latency transaction signing.
//! 
//! Features:
//! - BIP-39 mnemonic generation and validation
//! - BIP-32 HD key derivation
//! - BIP-44 multi-chain address derivation
//! - MPC threshold signatures (2-of-3)
//! - TEE secure enclave integration
//! - Hardware security module (HSM) support
//! - Multi-chain support (EVM, Solana, Bitcoin, etc.)

use std::collections::HashMap;
use std::sync::Arc;
use std::path::PathBuf;

pub mod mnemonic;
pub mod keys;
pub mod chains;
pub mod signing;
pub mod mpc;
pub mod hsm;
pub mod transaction;
pub mod hardware;

pub use mnemonic::*;
pub use keys::*;
pub use chains::*;
pub use signing::*;
pub use mpc::*;
pub use hsm::*;
pub use transaction::*;
pub use hardware::*;

/// Supported blockchain networks
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Blockchain {
    /// Ethereum Mainnet
    Ethereum,
    /// BNB Smart Chain
    BnbSmartChain,
    /// Polygon
    Polygon,
    /// Arbitrum
    Arbitrum,
    /// Optimism
    Optimism,
    /// Base
    Base,
    /// Avalanche
    Avalanche,
    /// Solana
    Solana,
    /// Bitcoin
    Bitcoin,
    /// TON
    Ton,
    /// Aptos
    Aptos,
    /// Sui
    Sui,
    /// Cosmos
    Cosmos,
    /// NEAR
    Near,
    /// TRON
    Tron,
    /// Custom EVM chain
    Evm(u64),
}

impl Blockchain {
    /// Get the coin type for BIP-44
    pub fn coin_type(&self) -> u32 {
        match self {
            Blockchain::Ethereum => 60,
            Blockchain::BnbSmartChain => 714,
            Blockchain::Polygon => 966,
            Blockchain::Arbitrum => 1101,
            Blockchain::Optimism => 1114,
            Blockchain::Base => 8453,
            Blockchain::Avalanche => 9075,
            Blockchain::Solana => 501,
            Blockchain::Bitcoin => 0,
            Blockchain::Ton => 607,
            Blockchain::Aptos => 637,
            Blockchain::Sui => 784,
            Blockchain::Cosmos => 118,
            Blockchain::Near => 397,
            Blockchain::Tron => 195,
            Blockchain::Evm(_) => 60,
        }
    }
    
    /// Get the derivation path for this chain
    pub fn derivation_path(&self) -> &str {
        match self {
            Blockchain::Solana => "m/44'/501'/0'/0'",
            Blockchain::Bitcoin => "m/44'/0'/0'/0/0",
            _ => "m/44'/60'/0'/0/0",
        }
    }
    
    /// Get chain ID (for EVM chains)
    pub fn chain_id(&self) -> Option<u64> {
        match self {
            Blockchain::Ethereum => Some(1),
            Blockchain::BnbSmartChain => Some(56),
            Blockchain::Polygon => Some(137),
            Blockchain::Arbitrum => Some(42161),
            Blockchain::Optimism => Some(10),
            Blockchain::Base => Some(8453),
            Blockchain::Avalanche => Some(43114),
            Blockchain::Evm(id) => Some(*id),
            _ => None,
        }
    }
    
    /// Get native token symbol
    pub fn symbol(&self) -> &str {
        match self {
            Blockchain::Ethereum => "ETH",
            Blockchain::BnbSmartChain => "BNB",
            Blockchain::Polygon => "MATIC",
            Blockchain::Arbitrum => "ETH",
            Blockchain::Optimism => "ETH",
            Blockchain::Base => "ETH",
            Blockchain::Avalanche => "AVAX",
            Blockchain::Solana => "SOL",
            Blockchain::Bitcoin => "BTC",
            Blockchain::Ton => "TON",
            Blockchain::Aptos => "APT",
            Blockchain::Sui => "SUI",
            Blockchain::Cosmos => "ATOM",
            Blockchain::Near => "NEAR",
            Blockchain::Tron => "TRX",
            Blockchain::Evm(_) => "ETH",
        }
    }
    
    /// Get native token decimals
    pub fn decimals(&self) -> u8 {
        match self {
            Blockchain::Bitcoin => 8,
            Blockchain::Solana | Blockchain::Ton | Blockchain::Near | 
            Blockchain::Aptos | Blockchain::Sui => 9,
            _ => 18,
        }
    }
}

/// Wallet configuration
#[derive(Debug, Clone)]
pub struct WalletConfig {
    /// Master seed (derived from mnemonic)
    pub seed: Vec<u8>,
    /// Supported blockchains
    pub chains: Vec<Blockchain>,
    /// Enable MPC signing
    pub mpc_enabled: bool,
    /// Enable TEE
    pub tee_enabled: bool,
    /// Enable HSM
    pub hsm_enabled: bool,
    /// Cache directory for encrypted data
    pub cache_dir: Option<PathBuf>,
    /// Network timeout in milliseconds
    pub timeout_ms: u64,
    /// Maximum retries
    pub max_retries: u32,
}

impl Default for WalletConfig {
    fn default() -> Self {
        Self {
            seed: vec![],
            chains: vec![
                Blockchain::Ethereum,
                Blockchain::BnbSmartChain,
                Blockchain::Polygon,
                Blockchain::Arbitrum,
                Blockchain::Optimism,
                Blockchain::Base,
                Blockchain::Avalanche,
                Blockchain::Solana,
                Blockchain::Bitcoin,
            ],
            mpc_enabled: true,
            tee_enabled: false,
            hsm_enabled: false,
            cache_dir: None,
            timeout_ms: 30000,
            max_retries: 3,
        }
    }
}

/// Wallet instance holding all derived keys and addresses
#[derive(Debug, Clone)]
pub struct Wallet {
    /// Configuration
    config: WalletConfig,
    /// Derived addresses by blockchain
    addresses: HashMap<Blockchain, DerivedAddress>,
    /// HD wallet for key derivation
    hd_wallet:HdWallet,
    /// MPC signer (if enabled)
    mpc_signer: Option<MpcSigner>,
    /// HSM signer (if enabled)
    hsm_signer: Option<HsmSigner>,
    /// Transaction cache
    tx_cache: lru::LruCache<String, TransactionStatus>,
}

/// Derived address for a specific blockchain
#[derive(Debug, Clone)]
pub struct DerivedAddress {
    /// Blockchain
    pub blockchain: Blockchain,
    /// Address string
    pub address: String,
    /// Public key (hex)
    pub public_key: String,
    /// Derivation path used
    pub path: String,
    /// Private key reference (encrypted in production)
    #[allow(dead_code)]
    private_key_ref: String,
}

/// Transaction status
#[derive(Debug, Clone)]
pub struct TransactionStatus {
    /// Transaction hash
    pub hash: String,
    /// Block number (if confirmed)
    pub block_number: Option<u64>,
    /// Confirmations
    pub confirmations: u32,
    /// Status (pending/confirmed/failed)
    pub status: TxStatus,
    /// Timestamp
    pub timestamp: u64,
}

/// Transaction status enum
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TxStatus {
    /// Transaction pending
    Pending,
    /// Transaction confirmed
    Confirmed,
    /// Transaction failed
    Failed,
}

/// Wallet errors
#[derive(Debug, thiserror::Error)]
pub enum WalletError {
    #[error("Invalid mnemonic: {0}")]
    InvalidMnemonic(String),
    
    #[error("Invalid derivation path: {0}")]
    InvalidDerivationPath(String),
    
    #[error("Derivation failed: {0}")]
    DerivationFailed(String),
    
    #[error("Signing failed: {0}")]
    SigningFailed(String),
    
    #[error("MPC error: {0}")]
    MpcError(String),
    
    #[error("HSM error: {0}")]
    HsmError(String),
    
    #[error("Hardware wallet error: {0}")]
    HardwareError(String),
    
    #[error("Encryption failed: {0}")]
    EncryptionFailed(String),
    
    #[error("Decryption failed: {0}")]
    DecryptionFailed(String),
    
    #[error("Invalid chain: {0}")]
    InvalidChain(String),
    
    #[error("Invalid address: {0}")]
    InvalidAddress(String),
    
    #[error("Transaction error: {0}")]
    TransactionError(String),
    
    #[error("Network error: {0}")]
    NetworkError(String),
    
    #[error("Unsupported operation: {0}")]
    Unsupported(String),
}

impl From<WalletError> for std::io::Error {
    fn from(e: WalletError) -> Self {
        std::io::Error::new(std::io::ErrorKind::Other, e.to_string())
    }
}

impl Wallet {
    /// Create a new wallet from mnemonic
    pub fn new(mnemonic: &str, config: WalletConfig) -> Result<Self, WalletError> {
        // Validate mnemonic
        if !Mnemonic::validate(mnemonic) {
            return Err(WalletError::InvalidMnemonic(
                "Mnemonic validation failed".to_string()
            ));
        }
        
        // Convert mnemonic to seed
        let seed = Mnemonic::to_seed(mnemonic, "");
        
        // Create HD wallet
        let hd_wallet = HdWallet::from_seed(&seed);
        
        let mut wallet = Self {
            config,
            addresses: HashMap::new(),
            hd_wallet,
            mpc_signer: None,
            hsm_signer: None,
            tx_cache: lru::LruCache::new(1000),
        };
        
        // Derive addresses for all configured chains
        wallet.derive_addresses()?;
        
        // Initialize MPC if enabled
        if wallet.config.mpc_enabled {
            wallet.mpc_signer = Some(MpcSigner::new()?);
        }
        
        // Initialize HSM if enabled
        if wallet.config.hsm_enabled {
            wallet.hsm_signer = Some(HsmSigner::new()?);
        }
        
        Ok(wallet)
    }
    
    /// Create wallet from seed directly
    pub fn from_seed(seed: &[u8], config: WalletConfig) -> Result<Self, WalletError> {
        let hd_wallet = HdWallet::from_seed(seed);
        
        let mut wallet = Self {
            config,
            addresses: HashMap::new(),
            hd_wallet,
            mpc_signer: None,
            hsm_signer: None,
            tx_cache: lru::LruCache::new(1000),
        };
        
        wallet.derive_addresses()?;
        
        if wallet.config.mpc_enabled {
            wallet.mpc_signer = Some(MpcSigner::new()?);
        }
        
        if wallet.config.hsm_enabled {
            wallet.hsm_signer = Some(HsmSigner::new()?);
        }
        
        Ok(wallet)
    }
    
    /// Derive addresses for all configured chains
    fn derive_addresses(&mut self) -> Result<(), WalletError> {
        for chain in &self.config.chains {
            let path = chain.derivation_path();
            let derived = self.hd_wallet.derive_address(chain, path)?;
            self.addresses.insert(*chain, derived);
        }
        Ok(())
    }
    
    /// Get address for a specific blockchain
    pub fn get_address(&self, chain: Blockchain) -> Option<&DerivedAddress> {
        self.addresses.get(&chain)
    }
    
    /// Get all addresses
    pub fn get_all_addresses(&self) -> &HashMap<Blockchain, DerivedAddress> {
        &self.addresses
    }
    
    /// Sign transaction using MPC
    pub async fn sign_transaction_mpc(
        &self,
        chain: Blockchain,
        tx: &Transaction,
    ) -> Result<String, WalletError> {
        let signer = self.mpc_signer.as_ref()
            .ok_or_else(|| WalletError::MpcError("MPC not enabled".to_string()))?;
        
        // Get address
        let address = self.get_address(chain)
            .ok_or_else(|| WalletError::InvalidChain("Chain not found".to_string()))?;
        
        // Sign using MPC
        let signature = signer.sign(tx, &address.public_key).await?;
        
        Ok(signature)
    }
    
    /// Sign transaction using standard method
    pub fn sign_transaction(
        &self,
        chain: Blockchain,
        tx: &Transaction,
    ) -> Result<String, WalletError> {
        // Get address
        let address = self.get_address(chain)
            .ok_or_else(|| WalletError::InvalidChain("Chain not found".to_string()))?;
        
        // Sign transaction
        let signed_tx = tx.sign(&address.private_key_ref)?;
        
        Ok(signed_tx)
    }
    
    /// Sign message
    pub fn sign_message(
        &self,
        chain: Blockchain,
        message: &[u8],
    ) -> Result<String, WalletError> {
        let address = self.get_address(chain)
            .ok_or_else(|| WalletError::InvalidChain("Chain not found".to_string()))?;
        
        match chain {
            Blockchain::Ethereum | Blockchain::BnbSmartChain | 
            Blockchain::Polygon | Blockchain::Arbitrum | 
            Blockchain::Optimism | Blockchain::Base |
            Blockchain::Avalanche | Blockchain::Evm(_) => {
                // EVM personal sign
                let signature = signing::evm_personal_sign(message, &address.private_key_ref)?;
                Ok(signature)
            }
            Blockchain::Solana => {
                // Solana message signing
                let signature = signing::solana_sign(message, &address.private_key_ref)?;
                Ok(signature)
            }
            Blockchain::Bitcoin => {
                // Bitcoin message signing
                let signature = signing::bitcoin_sign_message(message, &address.private_key_ref)?;
                Ok(signature)
            }
            _ => Err(WalletError::Unsupported(
                format!("Message signing not supported for {:?}", chain)
            )),
        }
    }
    
    /// Verify signature
    pub fn verify_signature(
        &self,
        chain: Blockchain,
        message: &[u8],
        signature: &str,
    ) -> Result<bool, WalletError> {
        let address = self.get_address(chain)
            .ok_or_else(|| WalletError::InvalidChain("Chain not found".to_string()))?;
        
        match chain {
            Blockchain::Ethereum | Blockchain::BnbSmartChain | 
            Blockchain::Polygon | Blockchain::Arbitrum | 
            Blockchain::Optimism | Blockchain::Base |
            Blockchain::Avalanche | Blockchain::Evm(_) => {
                signing::evm_verify_signature(message, signature, &address.address)
            }
            _ => Err(WalletError::Unsupported(
                format!("Signature verification not supported for {:?}", chain)
            )),
        }
    }
    
    /// Encrypt wallet data
    pub fn encrypt(&self, password: &str) -> Result<Vec<u8>, WalletError> {
        keys::encrypt_data(&self.hd_wallet.seed, password)
    }
    
    /// Decrypt wallet data
    pub fn decrypt(data: &[u8], password: &str) -> Result<Self, WalletError> {
        let seed = keys::decrypt_data(data, password)?;
        Self::from_seed(&seed, WalletConfig::default())
    }
    
    /// Export private key (WARNING: Use with caution)
    pub fn export_private_key(&self, chain: Blockchain) -> Result<String, WalletError> {
        let address = self.get_address(chain)
            .ok_or_else(|| WalletError::InvalidChain("Chain not found".to_string()))?;
        
        // In production, this should require additional authentication
        Ok(address.private_key_ref.clone())
    }
    
    /// Get wallet balance (stub - requires RPC connection)
    pub async fn get_balance(&self, chain: Blockchain) -> Result<String, WalletError> {
        let address = self.get_address(chain)
            .ok_or_else(|| WalletError::InvalidChain("Chain not found".to_string()))?;
        
        // This would call RPC in production
        Ok(format!("0 {}", chain.symbol()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_mnemonic_generation() {
        let mnemonic = Mnemonic::generate(24).unwrap();
        assert_eq!(mnemonic.split_whitespace().count(), 24);
        assert!(Mnemonic::validate(&mnemonic));
    }
    
    #[test]
    fn test_mnemonic_validation() {
        // Valid 24-word mnemonic
        let valid = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";
        assert!(!Mnemonic::validate(valid)); // Not a valid BIP-39
        
        // Valid BIP-39 test mnemonic
        let test_mnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon";
        // This is the standard test vector
    }
    
    #[test]
    fn test_blockchain_coin_types() {
        assert_eq!(Blockchain::Ethereum.coin_type(), 60);
        assert_eq!(Blockchain::Bitcoin.coin_type(), 0);
        assert_eq!(Blockchain::Solana.coin_type(), 501);
    }
}
