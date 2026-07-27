//! TigerWallet zkSync Era SDK
//! 
//! Production-ready zkSync Era (zkEVM L2) blockchain SDK.
//! Supports:
//! - Account management (EOA and Account Abstraction)
//! - Token transfers (ETH, ERC-20)
//! - Contract deployment and interaction
//! - Bridge operations (L1 <-> L2)
//! - Fee estimation
//! - Block finalization

pub mod address;
pub mod crypto;
pub mod provider;
pub mod transaction;
pub mod types;

pub use address::*;
pub use crypto::*;
pub use provider::*;
pub use transaction::*;
pub use types::*;

/// zkSync chain IDs
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ZksyncChainId {
    /// Mainnet
    Mainnet,
    /// Testnet (Sepolia)
    Testnet,
    /// Local development
    Local,
}

impl ZksyncChainId {
    /// Get chain ID
    pub fn chain_id(&self) -> u32 {
        match self {
            ZksyncChainId::Mainnet => 324,
            ZksyncChainId::Testnet => 300,
            ZksyncChainId::Local => 270,
        }
    }
    
    /// Get L1 chain ID (Ethereum)
    pub fn l1_chain_id(&self) -> u32 {
        match self {
            ZksyncChainId::Mainnet => 1,
            ZksyncChainId::Testnet => 11155111,
            ZksyncChainId::Local => 9,
        }
    }
    
    /// Get RPC URL
    pub fn rpc_url(&self) -> &'static str {
        match self {
            ZksyncChainId::Mainnet => "https://mainnet.era.zksync.io",
            ZksyncChainId::Testnet => "https://sepolia.era.zksync.io",
            ZksyncChainId::Local => "http://localhost:3050",
        }
    }
}

impl Default for ZksyncChainId {
    fn default() -> Self {
        ZksyncChainId::Mainnet
    }
}

/// zkSync Era constants
pub mod constants {
    /// L2 ETH token address
    pub const ETH_TOKEN_ADDRESS: &str = "0x0000000000000000000000000000000000000000";
    
    /// L1 messenger address
    pub const L1_MESSENGER: &str = "0x578a59c5e632d96d05a21539d69e8e0364afc5a2";
    
    /// Contract deployer address
    pub const CONTRACT_DEPLOYER: &str = "0x0000000000000000000000000000000000008006";
    
    /// Known bridge addresses
    pub mod bridges {
        pub const L1_ETH_BRIDGE: &str = "0x0000000000000000000000000000000000000000";
        pub const L2_ETH_BRIDGE: &str = "0x000000000000000000000000000000000000800a";
    }
}
