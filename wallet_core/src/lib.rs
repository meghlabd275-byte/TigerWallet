//! TigerWallet Core - Secure wallet operations in Rust
//! 
//! This crate provides the core security-critical wallet operations including:
//! - Mnemonic generation and validation
//! - Key derivation (BIP-32, BIP-39, BIP-44)
//! - Multi-chain address derivation
//! - Transaction signing
//! - Encryption and secure storage

use std::collections::HashMap;
use std::sync::Arc;

pub mod mnemonic;
pub mod key_derivation;
pub mod address;
pub mod transaction;
pub mod encryption;
pub mod multisig;
pub mod account_abstraction;
pub mod evm;
pub mod bitcoin;

pub use mnemonic::*;
pub use key_derivation::*;
pub use address::*;
pub use transaction::*;
pub use encryption::*;
pub use multisig::*;
pub use account_abstraction::*;
pub use evm::*;
pub use bitcoin::*;

/// Chain types supported by TigerWallet
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum ChainType {
    /// Ethereum Virtual Machine
    EVM,
    /// Bitcoin
    Bitcoin,
    /// Solana
    Solana,
    /// Aptos
    Aptos,
    /// Sui
    Sui,
    /// TRON
    TRON,
    /// Cosmos
    Cosmos,
    /// TON
    TON,
    /// NEAR
    NEAR,
    /// Algorand
    Algorand,
}

impl ChainType {
    /// Get the coin type for BIP-44
    pub fn coin_type(&self) -> u32 {
        match self {
            ChainType::EVM => 60,
            ChainType::Bitcoin => 0,
            ChainType::Solana => 501,
            ChainType::Aptos => 637,
            ChainType::Sui => 784,
            ChainType::TRON => 195,
            ChainType::Cosmos => 118,
            ChainType::TON => 607,
            ChainType::NEAR => 397,
            ChainType::Algorand => 283,
        }
    }
    
    /// Get the derivation path prefix
    pub fn derivation_prefix(&self) -> &str {
        match self {
            ChainType::EVM => "m/44'/60'/0'/0/0",
            ChainType::Bitcoin => "m/44'/0'/0'/0/0",
            ChainType::Solana => "m/44'/501'/0'/0'",
            ChainType::Aptos => "m/44'/637'/0'/0'/0'",
            ChainType::Sui => "m/44'/784'/0'/0'/0'",
            ChainType::TRON => "m/44'/195'/0'/0/0'",
            ChainType::Cosmos => "m/44'/118'/0'/0/0'",
            ChainType::TON => "m/44'/607'/0'/0/0'",
            ChainType::NEAR => "m/44'/397'/0'/0'",
            ChainType::Algorand => "m/44'/283'/0'/0/0'",
        }
    }
}

/// Chain configuration
#[derive(Debug, Clone)]
pub struct ChainConfig {
    /// Chain ID
    pub chain_id: u64,
    /// Chain type
    pub chain_type: ChainType,
    /// Chain name
    pub name: String,
    /// Symbol
    pub symbol: String,
    /// Decimals
    pub decimals: u8,
    /// RPC URL
    pub rpc_url: String,
    /// Explorer URL
    pub explorer_url: String,
}

impl Default for ChainConfig {
    fn default() -> Self {
        Self {
            chain_id: 1,
            chain_type: ChainType::EVM,
            name: "Ethereum".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_url: "https://eth.llamarpc.com".to_string(),
            explorer_url: "https://etherscan.io".to_string(),
        }
    }
}

/// Default chain configurations
pub fn default_chains() -> Vec<ChainConfig> {
    vec![
        // EVM Chains
        ChainConfig { chain_id: 1, chain_type: ChainType::EVM, name: "Ethereum".to_string(), symbol: "ETH".to_string(), decimals: 18, rpc_url: "https://eth.llamarpc.com".to_string(), explorer_url: "https://etherscan.io".to_string() },
        ChainConfig { chain_id: 56, chain_type: ChainType::EVM, name: "BNB Smart Chain".to_string(), symbol: "BNB".to_string(), decimals: 18, rpc_url: "https://bsc-dataseed.binance.org".to_string(), explorer_url: "https://bscscan.com".to_string() },
        ChainConfig { chain_id: 137, chain_type: ChainType::EVM, name: "Polygon".to_string(), symbol: "MATIC".to_string(), decimals: 18, rpc_url: "https://polygon-rpc.com".to_string(), explorer_url: "https://polygonscan.com".to_string() },
        ChainConfig { chain_id: 42161, chain_type: ChainType::EVM, name: "Arbitrum One".to_string(), symbol: "ETH".to_string(), decimals: 18, rpc_url: "https://arb1.arbitrum.io/rpc".to_string(), explorer_url: "https://arbiscan.io".to_string() },
        ChainConfig { chain_id: 10, chain_type: ChainType::EVM, name: "Optimism".to_string(), symbol: "ETH".to_string(), decimals: 18, rpc_url: "https://mainnet.optimism.io".to_string(), explorer_url: "https://optimistic.etherscan.io".to_string() },
        ChainConfig { chain_id: 8453, chain_type: ChainType::EVM, name: "Base".to_string(), symbol: "ETH".to_string(), decimals: 18, rpc_url: "https://mainnet.base.org".to_string(), explorer_url: "https://basescan.org".to_string() },
        ChainConfig { chain_id: 43114, chain_type: ChainType::EVM, name: "Avalanche C-Chain".to_string(), symbol: "AVAX".to_string(), decimals: 18, rpc_url: "https://api.avax.network/ext/bc/C/rpc".to_string(), explorer_url: "https://snowtrace.io".to_string() },
        ChainConfig { chain_id: 25, chain_type: ChainType::EVM, name: "Cronos".to_string(), symbol: "CRO".to_string(), decimals: 18, rpc_url: "https://evm.cronos.org".to_string(), explorer_url: "https://cronoscan.com".to_string() },
        ChainConfig { chain_id: 42220, chain_type: ChainType::EVM, name: "Celo".to_string(), symbol: "CELO".to_string(), decimals: 18, rpc_url: "https://forno.celo.org".to_string(), explorer_url: "https://explorer.celo.org".to_string() },
        ChainConfig { chain_id: 8217, chain_type: ChainType::EVM, name: "Klaytn".to_string(), symbol: "KLAY".to_string(), decimals: 18, rpc_url: "https://klaytn.fandom.finance".to_string(), explorer_url: "https://scope.klaytn.com".to_string() },
        // Non-EVM Chains
        ChainConfig { chain_id: 0, chain_type: ChainType::Bitcoin, name: "Bitcoin".to_string(), symbol: "BTC".to_string(), decimals: 8, rpc_url: "https://blockstream.info/api".to_string(), explorer_url: "https://mempool.space".to_string() },
        ChainConfig { chain_id: 101, chain_type: ChainType::Solana, name: "Solana".to_string(), symbol: "SOL".to_string(), decimals: 9, rpc_url: "https://api.mainnet-beta.solana.com".to_string(), explorer_url: "https://solscan.io".to_string() },
        ChainConfig { chain_id: 1, chain_type: ChainType::Aptos, name: "Aptos".to_string(), symbol: "APT".to_string(), decimals: 8, rpc_url: "https://fullnode.mainnet.aptoslabs.com".to_string(), explorer_url: "https://aptoscan.com".to_string() },
        ChainConfig { chain_id: 1, chain_type: ChainType::Sui, name: "Sui".to_string(), symbol: "SUI".to_string(), decimals: 9, rpc_url: "https://fullnode.mainnet.sui.io".to_string(), explorer_url: "https://suiexplorer.com".to_string() },
        ChainConfig { chain_id: 7281265, chain_type: ChainType::TRON, name: "TRON".to_string(), symbol: "TRX".to_string(), decimals: 6, rpc_url: "https://api.trongrid.io".to_string(), explorer_url: "https://tronscan.org".to_string() },
        ChainConfig { chain_id: 118, chain_type: ChainType::Cosmos, name: "Cosmos Hub".to_string(), symbol: "ATOM".to_string(), decimals: 6, rpc_url: "https://cosmos-rpc.polkachu.com".to_string(), explorer_url: "https://mintscan.io/cosmos".to_string() },
        ChainConfig { chain_id: 607, chain_type: ChainType::TON, name: "TON".to_string(), symbol: "TON".to_string(), decimals: 9, rpc_url: "https://toncenter.com/api/v2/".to_string(), explorer_url: "https://tonscan.org".to_string() },
    ]
}

/// Wallet instance holding derived addresses
#[derive(Debug, Clone)]
pub struct Wallet {
    /// Seed phrase (encrypted)
    pub seed: Vec<u8>,
    /// Derived addresses by chain ID
    pub addresses: HashMap<u64, DerivedAddress>,
}

/// Derived address for a specific chain
#[derive(Debug, Clone)]
pub struct DerivedAddress {
    /// Chain configuration
    pub chain: ChainConfig,
    /// Address
    pub address: String,
    /// Public key (hex)
    pub public_key: String,
    /// Derivation path
    pub path: String,
}

impl Wallet {
    /// Create a new wallet from seed
    pub fn from_seed(seed: &[u8], chains: &[ChainConfig]) -> Result<Self, WalletError> {
        let mut addresses = HashMap::new();
        
        for chain in chains {
            let derived = key_derivation::derive_address(seed, chain)?;
            addresses.insert(chain.chain_id, derived);
        }
        
        Ok(Self {
            seed: seed.to_vec(),
            addresses,
        })
    }
    
    /// Get address for a specific chain
    pub fn get_address(&self, chain_id: u64) -> Option<&DerivedAddress> {
        self.addresses.get(&chain_id)
    }
}

/// Errors that can occur in wallet operations
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
    
    #[error("Encryption failed: {0}")]
    EncryptionFailed(String),
    
    #[error("Decryption failed: {0}")]
    DecryptionFailed(String),
    
    #[error("Invalid chain: {0}")]
    InvalidChain(String),
    
    #[error("Invalid address: {0}")]
    InvalidAddress(String),
}

impl From<WalletError> for std::io::Error {
    fn from(e: WalletError) -> Self {
        std::io::Error::new(std::io::ErrorKind::Other, e.to_string())
    }
}