/**
 * BIP-44 Multi-Account Hierarchy Implementation
 * 
 * Implements BIP-44 for deterministic wallet structure across multiple chains.
 * https://github.com/satoshilabs/slips/blob/master/slip-0044.md
 */

use crate::bip32::{HDKey, DerivationPath};
use crate::bip32::DerivationError;
use std::fmt;
use std::str::FromStr;

/// BIP-44 Purpose constant
pub const PURPOSE: u32 = 44;

/// Coin type constants (SLIP-44)
pub mod coin_types {
    pub const BITCOIN: u32 = 0;
    pub const LITECOIN: u32 = 2;
    pub const DOGECOIN: u32 = 3;
    pub const ETHEREUM: u32 = 60;
    pub const POLYGON: u32 = 966;
    pub const BNB_CHAIN: u32 = 714;
    pub const ARBITRUM: u32 = 11021;
    pub const OPTIMISM: u32 = 10;
    pub const AVALANCHE_C: u32 = 9000;
    pub const SOLANA: u32 = 501;
    pub const APTOS: u32 = 637;
    pub const SUI: u32 = 784;
    pub const TON: u32 = 607;
    pub const NEAR: u32 = 397;
    pub const COSMOS: u32 = 118;
    pub const ALGORAND: u32 = 283;
    pub const HEDERA: u32 = 3030;
    pub const STARKNET: u32 = 8864;
    pub const ZKSYNC: u32 = 324;
    pub const TRON: u32 = 195;
}

/// BIP-44 derivation path
#[derive(Clone, Debug)]
pub struct Path {
    pub purpose: u32,
    pub coin_type: u32,
    pub account: u32,
    pub change: u32,
    pub index: u32,
}

impl Path {
    /// Create new BIP-44 path
    pub fn new(coin_type: u32, account: u32, change: u32, index: u32) -> Self {
        Self {
            purpose: PURPOSE,
            coin_type,
            account,
            change,
            index,
        }
    }
    
    /// Create path from string (e.g., "m/44'/60'/0'/0/0")
    pub fn from_string(s: &str) -> Result<Self, DerivationError> {
        let path = DerivationPath::parse(s)?;
        let indices = path.as_indices();
        
        if indices.len() != 5 {
            return Err(DerivationError::InvalidPath(
                "BIP-44 path must have exactly 5 components".to_string()
            ));
        }
        
        Ok(Self {
            purpose: indices[0] & 0x7FFFFFFF,
            coin_type: indices[1] & 0x7FFFFFFF,
            account: indices[2] & 0x7FFFFFFF,
            change: indices[3] & 0x7FFFFFFF,
            index: indices[4] & 0x7FFFFFFF,
        })
    }
    
    /// Convert to derivation path
    pub fn to_derivation_path(&self) -> DerivationPath {
        DerivationPath::new(&[
            0x80000000 | self.purpose,
            0x80000000 | self.coin_type,
            0x80000000 | self.account,
            self.change,
            self.index,
        ])
    }
    
    /// Create with hardened derivation
    pub fn to_hardened(&self) -> DerivationPath {
        DerivationPath::new(&[
            0x80000000 | self.purpose,
            0x80000000 | self.coin_type,
            0x80000000 | self.account,
            self.change,
            self.index,
        ])
    }
    
    /// Get path string
    pub fn to_string(&self) -> String {
        // BIP-44: purpose, coin_type, and account are hardened; change and index are not
        format!(
            "m/{}'/{}'/{}'/{}/{}",
            self.purpose, self.coin_type, self.account, self.change, self.index
        )
    }
    
    /// Get with specific index
    pub fn with_index(&self, index: u32) -> Self {
        Self {
            purpose: self.purpose,
            coin_type: self.coin_type,
            account: self.account,
            change: self.change,
            index,
        }
    }
    
    /// Default path for Ethereum (m/44'/60'/0'/0/0)
    pub fn ethereum() -> Self {
        Self::new(coin_types::ETHEREUM, 0, 0, 0)
    }
    
    /// Default path for Bitcoin (m/44'/0'/0'/0/0)
    pub fn bitcoin() -> Self {
        Self::new(coin_types::BITCOIN, 0, 0, 0)
    }
    
    /// Default path for Solana (m/44'/501'/0'/0')
    pub fn solana() -> Self {
        Self::new(coin_types::SOLANA, 0, 0, 0)
    }
    
    /// Default path for Polygon
    pub fn polygon() -> Self {
        Self::new(coin_types::POLYGON, 0, 0, 0)
    }
    
    /// Default path for BNB Chain
    pub fn bnb_chain() -> Self {
        Self::new(coin_types::BNB_CHAIN, 0, 0, 0)
    }
}

impl fmt::Display for Path {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.to_string())
    }
}

/// Chain identifiers
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Chain {
    Bitcoin,
    Ethereum,
    Polygon,
    BNBChain,
    Arbitrum,
    Optimism,
    Avalanche,
    Solana,
    Aptos,
    Sui,
    Ton,
    Near,
    Cosmos,
    Algorand,
    Hedera,
    Starknet,
    ZkSync,
    Tron,
    Dogecoin,
    Litecoin,
}

impl Chain {
    /// Get coin type for chain
    pub fn coin_type(&self) -> u32 {
        match self {
            Chain::Bitcoin => coin_types::BITCOIN,
            Chain::Ethereum => coin_types::ETHEREUM,
            Chain::Polygon => coin_types::POLYGON,
            Chain::BNBChain => coin_types::BNB_CHAIN,
            Chain::Arbitrum => coin_types::ARBITRUM,
            Chain::Optimism => coin_types::OPTIMISM,
            Chain::Avalanche => coin_types::AVALANCHE_C,
            Chain::Solana => coin_types::SOLANA,
            Chain::Aptos => coin_types::APTOS,
            Chain::Sui => coin_types::SUI,
            Chain::Ton => coin_types::TON,
            Chain::Near => coin_types::NEAR,
            Chain::Cosmos => coin_types::COSMOS,
            Chain::Algorand => coin_types::ALGORAND,
            Chain::Hedera => coin_types::HEDERA,
            Chain::Starknet => coin_types::STARKNET,
            Chain::ZkSync => coin_types::ZKSYNC,
            Chain::Tron => coin_types::TRON,
            Chain::Dogecoin => coin_types::DOGECOIN,
            Chain::Litecoin => coin_types::LITECOIN,
        }
    }
    
    /// Get default derivation path
    pub fn default_path(&self) -> Path {
        Path::new(self.coin_type(), 0, 0, 0)
    }
    
    /// Get chain name
    pub fn name(&self) -> &'static str {
        match self {
            Chain::Bitcoin => "Bitcoin",
            Chain::Ethereum => "Ethereum",
            Chain::Polygon => "Polygon",
            Chain::BNBChain => "BNB Chain",
            Chain::Arbitrum => "Arbitrum",
            Chain::Optimism => "Optimism",
            Chain::Avalanche => "Avalanche",
            Chain::Solana => "Solana",
            Chain::Aptos => "Aptos",
            Chain::Sui => "Sui",
            Chain::Ton => "Toncoin",
            Chain::Near => "NEAR",
            Chain::Cosmos => "Cosmos",
            Chain::Algorand => "Algorand",
            Chain::Hedera => "Hedera",
            Chain::Starknet => "Starknet",
            Chain::ZkSync => "zkSync",
            Chain::Tron => "Tron",
            Chain::Dogecoin => "Dogecoin",
            Chain::Litecoin => "Litecoin",
        }
    }
    
    /// Get chain ID (for EVM chains)
    pub fn chain_id(&self) -> Option<u64> {
        match self {
            Chain::Ethereum => Some(1),
            Chain::Polygon => Some(137),
            Chain::BNBChain => Some(56),
            Chain::Arbitrum => Some(42161),
            Chain::Optimism => Some(10),
            Chain::Avalanche => Some(43114),
            Chain::Starknet => Some(1),
            Chain::ZkSync => Some(1),
            _ => None,
        }
    }
    
    /// Get all supported chains
    pub fn all() -> Vec<Chain> {
        vec![
            Chain::Bitcoin,
            Chain::Ethereum,
            Chain::Polygon,
            Chain::BNBChain,
            Chain::Arbitrum,
            Chain::Optimism,
            Chain::Avalanche,
            Chain::Solana,
            Chain::Aptos,
            Chain::Sui,
            Chain::Ton,
            Chain::Near,
            Chain::Cosmos,
            Chain::Algorand,
            Chain::Hedera,
            Chain::Starknet,
            Chain::ZkSync,
            Chain::Tron,
            Chain::Dogecoin,
            Chain::Litecoin,
        ]
    }
}

impl fmt::Display for Chain {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.name())
    }
}

impl FromStr for Chain {
    type Err = String;
    
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "bitcoin" | "btc" => Ok(Chain::Bitcoin),
            "ethereum" | "eth" => Ok(Chain::Ethereum),
            "polygon" | "matic" => Ok(Chain::Polygon),
            "bnb" | "bsc" | "binance" => Ok(Chain::BNBChain),
            "arbitrum" | "arb" => Ok(Chain::Arbitrum),
            "optimism" | "op" => Ok(Chain::Optimism),
            "avalanche" | "avax" => Ok(Chain::Avalanche),
            "solana" | "sol" => Ok(Chain::Solana),
            "aptos" | "apt" => Ok(Chain::Aptos),
            "sui" => Ok(Chain::Sui),
            "ton" | "toncoin" => Ok(Chain::Ton),
            "near" => Ok(Chain::Near),
            "cosmos" | "atom" => Ok(Chain::Cosmos),
            "algorand" | "algo" => Ok(Chain::Algorand),
            "hedera" | "hbar" => Ok(Chain::Hedera),
            "starknet" => Ok(Chain::Starknet),
            "zksync" | "zk" => Ok(Chain::ZkSync),
            "tron" | "trx" => Ok(Chain::Tron),
            "dogecoin" | "doge" => Ok(Chain::Dogecoin),
            "litecoin" | "ltc" => Ok(Chain::Litecoin),
            _ => Err(format!("Unknown chain: {}", s)),
        }
    }
}

/// Chain configuration
#[derive(Clone, Debug)]
pub struct ChainConfig {
    pub chain: Chain,
    pub name: String,
    pub symbol: String,
    pub decimals: u8,
    pub rpc_url: String,
    pub explorer_url: String,
    pub is_evm: bool,
}

impl ChainConfig {
    /// Get default configurations for major chains
    pub fn default_configs() -> Vec<ChainConfig> {
        vec![
            ChainConfig {
                chain: Chain::Ethereum,
                name: "Ethereum".to_string(),
                symbol: "ETH".to_string(),
                decimals: 18,
                rpc_url: "https://eth.llamarpc.com".to_string(),
                explorer_url: "https://etherscan.io".to_string(),
                is_evm: true,
            },
            ChainConfig {
                chain: Chain::Bitcoin,
                name: "Bitcoin".to_string(),
                symbol: "BTC".to_string(),
                decimals: 8,
                rpc_url: "https://blockstream.info/api".to_string(),
                explorer_url: "https://blockstream.info".to_string(),
                is_evm: false,
            },
            ChainConfig {
                chain: Chain::Polygon,
                name: "Polygon".to_string(),
                symbol: "MATIC".to_string(),
                decimals: 18,
                rpc_url: "https://polygon-rpc.com".to_string(),
                explorer_url: "https://polygonscan.com".to_string(),
                is_evm: true,
            },
            ChainConfig {
                chain: Chain::BNBChain,
                name: "BNB Chain".to_string(),
                symbol: "BNB".to_string(),
                decimals: 18,
                rpc_url: "https://bsc-dataseed.binance.org".to_string(),
                explorer_url: "https://bscscan.com".to_string(),
                is_evm: true,
            },
            ChainConfig {
                chain: Chain::Solana,
                name: "Solana".to_string(),
                symbol: "SOL".to_string(),
                decimals: 9,
                rpc_url: "https://api.mainnet-beta.solana.com".to_string(),
                explorer_url: "https://explorer.solana.com".to_string(),
                is_evm: false,
            },
        ]
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_eth_path() {
        let path = Path::ethereum();
        assert_eq!(path.coin_type, coin_types::ETHEREUM);
    }
    
    #[test]
    fn test_btc_path() {
        let path = Path::bitcoin();
        assert_eq!(path.coin_type, coin_types::BITCOIN);
    }
    
    #[test]
    fn test_path_string() {
        let path = Path::ethereum();
        assert_eq!(path.to_string(), "m/44'/60'/0'/0/0");
    }
    
    #[test]
    fn test_chain_coin_type() {
        assert_eq!(Chain::Ethereum.coin_type(), coin_types::ETHEREUM);
        assert_eq!(Chain::Bitcoin.coin_type(), coin_types::BITCOIN);
        assert_eq!(Chain::Solana.coin_type(), coin_types::SOLANA);
    }
    
    #[test]
    fn test_chain_from_str() {
        assert_eq!("eth".parse::<Chain>().unwrap(), Chain::Ethereum);
        assert_eq!("btc".parse::<Chain>().unwrap(), Chain::Bitcoin);
        assert_eq!("sol".parse::<Chain>().unwrap(), Chain::Solana);
    }

    #[test]
    fn test_bip44_path_derivation() {
        // Parsing a canonical BIP-44 path round-trips through to_string
        let path = Path::from_string("m/44'/60'/0'/0/0").unwrap();
        assert_eq!(path.purpose, 44);
        assert_eq!(path.coin_type, 60);
        assert_eq!(path.account, 0);
        assert_eq!(path.change, 0);
        assert_eq!(path.index, 0);
        assert_eq!(path.to_string(), "m/44'/60'/0'/0/0");

        // to_derivation_path yields hardened purpose/coin_type/account indices
        let dp = path.to_derivation_path();
        let indices = dp.as_indices();
        assert_eq!(indices[0], 0x80000000 | 44);
        assert_eq!(indices[1], 0x80000000 | 60);
        assert_eq!(indices[2], 0x80000000 | 0);
        assert_eq!(indices[3], 0);
        assert_eq!(indices[4], 0);
    }

    #[test]
    fn test_bip44_with_index_and_chains() {
        // with_index updates only the address index
        let base = Path::ethereum();
        let next = base.with_index(5);
        assert_eq!(next.index, 5);
        assert_eq!(next.coin_type, base.coin_type);

        // Different chains map to their SLIP-44 coin types
        assert_eq!(Path::bitcoin().coin_type, coin_types::BITCOIN);
        assert_eq!(Path::solana().coin_type, coin_types::SOLANA);
        assert_eq!(Path::polygon().coin_type, coin_types::POLYGON);

        // default_path per chain matches the chain's coin type
        assert_eq!(Chain::Ethereum.default_path().coin_type, coin_types::ETHEREUM);
        assert_eq!(Chain::Bitcoin.default_path().coin_type, coin_types::BITCOIN);
        assert_eq!(Chain::Solana.default_path().coin_type, coin_types::SOLANA);
    }

    #[test]
    fn test_bip44_invalid_path() {
        // Wrong number of components is rejected
        assert!(Path::from_string("m/44'/60'/0'").is_err());
        // Non-numeric index is rejected
        assert!(Path::from_string("m/44'/60'/0'/0/abc").is_err());
    }
}
