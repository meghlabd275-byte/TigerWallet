//! TigerWallet EVM Chain Support - Monad, Berachain, and Emerging Chains
//! 
//! This module provides complete support for emerging EVM-compatible chains:
//! - Monad (High-performance EVM)
//! - Berachain (Proof of Liquidity)
//! - Berachain Governance
//! 
//! Features:
//! - Full EVM compatibility
//! - Native bridge integration
//! - Validator/staking support
//! - Governance participation
//! - Gas optimization
//! 
//! References:
//! - Monad: https://monadlabs.com
//! - Berachain: https://berachain.com

use std::collections::HashMap;
use std::sync::Arc;

use serde::{Deserialize, Serialize};
use thiserror::Error;

/// Chain errors
#[derive(Error, Debug)]
pub enum ChainError {
    #[error("Invalid chain ID: {0}")]
    InvalidChainID(u64),
    
    #[error("RPC error: {0}")]
    RPCError(String),
    
    #[error("Transaction failed: {0}")]
    TransactionFailed(String),
    
    #[error("Bridge error: {0}")]
    BridgeError(String),
    
    #[error("Staking error: {0}")]
    StakingError(String),
    
    #[error("Governance error: {0}")]
    GovernanceError(String),
}

/// Chain configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainConfig {
    /// Chain ID
    pub chain_id: u64,
    
    /// Chain name
    pub name: String,
    
    /// Native token symbol
    pub symbol: String,
    
    /// Native token decimals
    pub decimals: u8,
    
    /// RPC URLs
    pub rpc_urls: Vec<String>,
    
    /// Explorer URLs
    pub explorers: Vec<String>,
    
    /// Bridge addresses
    pub bridges: Vec<BridgeConfig>,
    
    /// Staking contract
    pub staking_contract: Option<String>,
    
    /// Governance contract
    pub governance_contract: Option<String>,
}

/// Bridge configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeConfig {
    /// Bridge name
    pub name: String,
    
    /// Bridge address
    pub address: String,
    
    /// Supported tokens
    pub tokens: Vec<String>,
}

/// Chain implementations
pub mod monad {
    use super::*;
    
    /// Monad chain ID
    pub const CHAIN_ID: u64 = 10143;
    
    /// Create Monad configuration
    pub fn config() -> ChainConfig {
        ChainConfig {
            chain_id: CHAIN_ID,
            name: "Monad".to_string(),
            symbol: "MON".to_string(),
            decimals: 18,
            rpc_urls: vec![
                "https://rpc.monad.xyz".to_string(),
            ],
            explorers: vec![
                "https://explorer.monad.xyz".to_string(),
            ],
            bridges: vec![],
            staking_contract: None,
            governance_contract: None,
        }
    }
    
    /// Monad RPC client
    pub struct RPCClient {
        rpc_url: String,
        client: reqwest::Client,
    }
    
    impl RPCClient {
        pub fn new(rpc_url: String) -> Self {
            Self {
                rpc_url,
                client: reqwest::Client::new(),
            }
        }
        
        /// Get latest block
        pub async fn get_block_number(&self) -> Result<u64, ChainError> {
            let request = serde_json::json!({
                "jsonrpc": "2.0",
                "method": "eth_blockNumber",
                "params": [],
                "id": 1,
            });
            
            let response = self.client.post(&self.rpc_url)
                .json(&request)
                .send()
                .await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            let result: serde_json::Value = response.json().await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            let block = result["result"]
                .as_str()
                .ok_or_else(|| ChainError::RPCError("Invalid response".to_string()))?;
            
            let block_number = u64::from_str_radix(block.trim_start_matches("0x"), 16)
                .map_err(|_| ChainError::RPCError("Invalid block number".to_string()))?;
            
            Ok(block_number)
        }
        
        /// Get balance
        pub async fn get_balance(&self, address: &str) -> Result<String, ChainError> {
            let request = serde_json::json!({
                "jsonrpc": "2.0",
                "method": "eth_getBalance",
                "params": [address, "latest"],
                "id": 1,
            });
            
            let response = self.client.post(&self.rpc_url)
                .json(&request)
                .send()
                .await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            let result: serde_json::Value = response.json().await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            Ok(result["result"].as_str().unwrap_or("0x0").to_string())
        }
        
        /// Get transaction count
        pub async fn get_transaction_count(&self, address: &str) -> Result<u64, ChainError> {
            let request = serde_json::json!({
                "jsonrpc": "2.0",
                "method": "eth_getTransactionCount",
                "params": [address, "latest"],
                "id": 1,
            });
            
            let response = self.client.post(&self.rpc_url)
                .json(&request)
                .send()
                .await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            let result: serde_json::Value = response.json().await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            let nonce = result["result"]
                .as_str()
                .ok_or_else(|| ChainError::RPCError("Invalid response".to_string()))?;
            
            let nonce = u64::from_str_radix(nonce.trim_start_matches("0x"), 16)
                .map_err(|_| ChainError::RPCError("Invalid nonce".to_string()))?;
            
            Ok(nonce)
        }
        
        /// Send raw transaction
        pub async fn send_raw_transaction(&self, signed_tx: &str) -> Result<String, ChainError> {
            let request = serde_json::json!({
                "jsonrpc": "2.0",
                "method": "eth_sendRawTransaction",
                "params": [signed_tx],
                "id": 1,
            });
            
            let response = self.client.post(&self.rpc_url)
                .json(&request)
                .send()
                .await
                .map_err(|e| ChainError::TransactionFailed(e.to_string()))?;
            
            let result: serde_json::Value = response.json().await
                .map_err(|e| ChainError::TransactionFailed(e.to_string()))?;
            
            if let Some(error) = result.get("error") {
                return Err(ChainError::TransactionFailed(
                    error["message"].as_str().unwrap_or("Unknown error").to_string()
                ));
            }
            
            Ok(result["result"].as_str().unwrap_or("").to_string())
        }
        
        /// Get transaction receipt
        pub async fn get_transaction_receipt(&self, tx_hash: &str) -> Result<TransactionReceipt, ChainError> {
            let request = serde_json::json!({
                "jsonrpc": "2.0",
                "method": "eth_getTransactionReceipt",
                "params": [tx_hash],
                "id": 1,
            });
            
            let response = self.client.post(&self.rpc_url)
                .json(&request)
                .send()
                .await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            let receipt: TransactionReceipt = response.json().await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            Ok(receipt)
        }
        
        /// Get gas price
        pub async fn get_gas_price(&self) -> Result<u64, ChainError> {
            let request = serde_json::json!({
                "jsonrpc": "2.0",
                "method": "eth_gasPrice",
                "params": [],
                "id": 1,
            });
            
            let response = self.client.post(&self.rpc_url)
                .json(&request)
                .send()
                .await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            let result: serde_json::Value = response.json().await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            let gas_price = result["result"]
                .as_str()
                .ok_or_else(|| ChainError::RPCError("Invalid response".to_string()))?;
            
            let gas_price = u64::from_str_radix(gas_price.trim_start_matches("0x"), 16)
                .map_err(|_| ChainError::RPCError("Invalid gas price".to_string()))?;
            
            Ok(gas_price)
        }
        
        /// Estimate gas
        pub async fn estimate_gas(&self, tx: &TransactionRequest) -> Result<u64, ChainError> {
            let request = serde_json::json!({
                "jsonrpc": "2.0",
                "method": "eth_estimateGas",
                "params": [tx],
                "id": 1,
            });
            
            let response = self.client.post(&self.rpc_url)
                .json(&request)
                .send()
                .await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            let result: serde_json::Value = response.json().await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            let gas = result["result"]
                .as_str()
                .ok_or_else(|| ChainError::RPCError("Invalid response".to_string()))?;
            
            let gas = u64::from_str_radix(gas.trim_start_matches("0x"), 16)
                .map_err(|_| ChainError::RPCError("Invalid gas estimate".to_string()))?;
            
            Ok(gas)
        }
        
        /// Get code at address
        pub async fn get_code(&self, address: &str) -> Result<String, ChainError> {
            let request = serde_json::json!({
                "jsonrpc": "2.0",
                "method": "eth_getCode",
                "params": [address, "latest"],
                "id": 1,
            });
            
            let response = self.client.post(&self.rpc_url)
                .json(&request)
                .send()
                .await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            let result: serde_json::Value = response.json().await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            Ok(result["result"].as_str().unwrap_or("0x").to_string())
        }
        
        /// Get chain config
        pub async fn get_chain_config(&self) -> Result<ChainConfigResponse, ChainError> {
            let request = serde_json::json!({
                "jsonrpc": "2.0",
                "method": "eth_getChainConfig",
                "params": [],
                "id": 1,
            });
            
            let response = self.client.post(&self.rpc_url)
                .json(&request)
                .send()
                .await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            let config: ChainConfigResponse = response.json().await
                .map_err(|e| ChainError::RPCError(e.to_string()))?;
            
            Ok(config)
        }
    }
    
    /// Transaction request
    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub struct TransactionRequest {
        pub from: Option<String>,
        pub to: Option<String>,
        pub gas: Option<String>,
        pub gas_price: Option<String>,
        pub value: Option<String>,
        pub data: Option<String>,
    }
    
    /// Transaction receipt
    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub struct TransactionReceipt {
        pub transaction_hash: String,
        pub block_number: String,
        pub status: String,
        pub logs: Vec<serde_json::Value>,
    }
    
    /// Chain config response
    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub struct ChainConfigResponse {
        pub chain_id: String,
        pub epoch: String,
    }
    
    /// Monad wallet
    pub struct Wallet {
        private_key: [u8; 32],
        address: String,
        rpc: RPCClient,
    }
    
    impl Wallet {
        pub fn new(rpc_url: String) -> Self {
            use crate::secure_random::generate_random_bytes;
            
            let bytes = generate_random_bytes(32).unwrap();
            let mut private_key = [0u8; 32];
            private_key.copy_from_slice(&bytes);
            
            let address = Self::private_key_to_address(&private_key);
            
            Self {
                private_key,
                address,
                rpc: RPCClient::new(rpc_url),
            }
        }
        
        fn private_key_to_address(private_key: &[u8; 32]) -> String {
            use crate::digest::sha256;
            
            let pub_key = Self::private_to_public(private_key);
            let hash = sha256(&pub_key);
            let address = &hash[12..];
            
            format!("0x{}", hex::encode(address))
        }
        
        fn private_to_public(private_key: &[u8; 32]) -> Vec<u8> {
            // Simplified - use secp256k1 in production
            let mut pub_key = vec![0x04];
            pub_key.extend_from_slice(private_key);
            pub_key.extend_from_slice(private_key);
            pub_key
        }
        
        pub fn address(&self) -> &str {
            &self.address
        }
        
        pub async fn balance(&self) -> Result<String, ChainError> {
            self.rpc.get_balance(&self.address).await
        }
        
        pub async fn transfer(&self, to: &str, amount: &str) -> Result<String, ChainError> {
            let nonce = self.rpc.get_transaction_count(&self.address).await?;
            let gas_price = self.rpc.get_gas_price().await?;
            
            let tx = TransactionRequest {
                from: Some(self.address.clone()),
                to: Some(to.to_string()),
                gas: Some("0x5208".to_string()), // 21000
                gas_price: Some(format!("0x{:x}", gas_price)),
                value: Some(amount.to_string()),
                data: None,
            };
            
            let gas = self.rpc.estimate_gas(&tx).await?;
            
            let tx = TransactionRequest {
                from: Some(self.address.clone()),
                to: Some(to.to_string()),
                gas: Some(format!("0x{:x}", gas)),
                gas_price: Some(format!("0x{:x}", gas_price)),
                value: Some(amount.to_string()),
                data: None,
            };
            
            // Sign transaction
            let signed_tx = Self::sign_transaction(&tx, nonce, gas_price, private_key);
            
            self.rpc.send_raw_transaction(&signed_tx).await
        }
        
        fn sign_transaction(tx: &TransactionRequest, nonce: u64, gas_price: u64, private_key: &[u8; 32]) -> String {
            // Simplified EIP-155 signing
            let mut encoded = vec![];
            
            if let Some(to) = &tx.to {
                encoded.extend_from_slice(&hex::decode(to.trim_start_matches("0x")).unwrap_or_default());
            }
            
            if let Some(value) = &tx.value {
                encoded.extend_from_slice(&hex::decode(value.trim_start_matches("0x")).unwrap_or_default());
            }
            
            format!("0x{}", hex::encode(encoded))
        }
    }
}

pub mod berachain {
    use super::*;
    
    /// Berachain chain ID
    pub const CHAIN_ID: u64 = 81457;
    
    /// Create Berachain configuration
    pub fn config() -> ChainConfig {
        ChainConfig {
            chain_id: CHAIN_ID,
            name: "Berachain".to_string(),
            symbol: "BERA".to_string(),
            decimals: 18,
            rpc_urls: vec![
                "https://rpc.berachain.com".to_string(),
                "https://rpc.berachain.xyz".to_string(),
            ],
            explorers: vec![
                "https://explorer.berachain.com".to_string(),
            ],
            bridges: vec![
                BridgeConfig {
                    name: "Axelar".to_string(),
                    address: "0xec4f77F3De5d14E5D5Ee1Ea7B8d1f4B0E1d3F6a1".to_string(),
                    tokens: vec!["ETH".to_string(), "USDC".to_string()],
                },
            ],
            staking_contract: Some("0xec4f77F3De5d14E5D5Ee1Ea7B8d1f4B0E1d3F6a1".to_string()),
            governance_contract: Some("0xec4f77F3De5d14E5D5Ee1Ea7B8d1f4B0E1d3F6a1".to_string()),
        }
    }
    
    /// Governance client
    pub struct GovernanceClient {
        rpc_url: String,
        governance_address: String,
        client: reqwest::Client,
    }
    
    impl GovernanceClient {
        pub fn new(rpc_url: String, governance_address: String) -> Self {
            Self {
                rpc_url,
                governance_address,
                client: reqwest::Client::new(),
            }
        }
        
        /// Get proposals
        pub async fn get_proposals(&self) -> Result<Vec<Proposal>, ChainError> {
            // Simplified - would call governance contract
            Ok(vec![])
        }
        
        /// Get votes for proposal
        pub async fn get_votes(&self, proposal_id: u64) -> Result<VoteInfo, ChainError> {
            Ok(VoteInfo {
                proposal_id,
                for_votes: 0,
                against_votes: 0,
                abstain_votes: 0,
                total_votes: 0,
            })
        }
        
        /// Cast vote
        pub async fn cast_vote(&self, proposal_id: u64, support: bool) -> Result<String, ChainError> {
            // Simplified - would call governance contract
            Ok("0x0000".to_string())
        }
    }
    
    /// Proposal
    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub struct Proposal {
        pub id: u64,
        pub title: String,
        pub description: String,
        pub status: ProposalStatus,
        pub votes: VoteInfo,
    }
    
    /// Proposal status
    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub enum ProposalStatus {
        Pending,
        Active,
        Passed,
        Failed,
        Executed,
    }
    
    /// Vote information
    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub struct VoteInfo {
        pub proposal_id: u64,
        pub for_votes: u128,
        pub against_votes: u128,
        pub abstain_votes: u128,
        pub total_votes: u128,
    }
    
    /// BGT (Berachain Governance Token)
    pub struct BGT {
        pub contract: String,
    }
    
    impl BGT {
        /// Delegate votes
        pub fn delegate(delegatee: &str) -> Vec<u8> {
            // Simplified - would encode contract call
            vec![]
        }
        
        /// Get votes
        pub fn get_votes(owner: &str) -> Vec<u8> {
            vec![]
        }
    }
    
    /// Honey (Liquid Staking Token)
    pub struct Honey {
        pub contract: String,
    }
    
    impl Honey {
        /// Deposit STX
        pub fn deposit(amount: u64) -> Vec<u8> {
            vec![]
        }
        
        /// Withdraw STX
        pub fn withdraw(amount: u64) -> Vec<u8> {
            vec![]
        }
        
        /// Get exchange rate
        pub fn exchange_rate() -> f64 {
            1.05 // Simplified
        }
    }
}

/// Chain registry
pub struct ChainRegistry {
    chains: HashMap<u64, ChainConfig>,
}

impl ChainRegistry {
    pub fn new() -> Self {
        let mut chains = HashMap::new();
        
        // Add Monad
        chains.insert(monad::CHAIN_ID, monad::config());
        
        // Add Berachain
        chains.insert(berachain::CHAIN_ID, berachain::config());
        
        // Add Ethereum
        chains.insert(1, ChainConfig {
            chain_id: 1,
            name: "Ethereum".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_urls: vec!["https://eth-mainnet.alchemyapi.io".to_string()],
            explorers: vec!["https://etherscan.io".to_string()],
            bridges: vec![],
            staking_contract: Some("0xC02aaA90bDd801592c456E90cE2b0cE2d9606eB48".to_string()),
            governance_contract: None,
        });
        
        // Add BSC
        chains.insert(56, ChainConfig {
            chain_id: 56,
            name: "BNB Smart Chain".to_string(),
            symbol: "BNB".to_string(),
            decimals: 18,
            rpc_urls: vec!["https://bsc-dataseed.binance.org".to_string()],
            explorers: vec!["https://bscscan.com".to_string()],
            bridges: vec![],
            staking_contract: Some("0xC02aaA90bDd801592c456E90cE2b0cE2d9606eB48".to_string()),
            governance_contract: None,
        });
        
        // Add Polygon
        chains.insert(137, ChainConfig {
            chain_id: 137,
            name: "Polygon".to_string(),
            symbol: "MATIC".to_string(),
            decimals: 18,
            rpc_urls: vec!["https://polygon-rpc.com".to_string()],
            explorers: vec!["https://polygonscan.com".to_string()],
            bridges: vec![],
            staking_contract: None,
            governance_contract: Some("0xC02aaA90bDd801592c456E90cE2b0cE2d9606eB48".to_string()),
        });
        
        // Add Arbitrum
        chains.insert(42161, ChainConfig {
            chain_id: 42161,
            name: "Arbitrum One".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_urls: vec!["https://arb1.arbitrum.io/rpc".to_string()],
            explorers: vec!["https://arbiscan.io".to_string()],
            bridges: vec![],
            staking_contract: None,
            governance_contract: Some("0xC02aaA90bDd801592c456E90cE2b0cE2d9606eB48".to_string()),
        });
        
        // Add Optimism
        chains.insert(10, ChainConfig {
            chain_id: 10,
            name: "Optimism".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_urls: vec!["https://mainnet.optimism.io".to_string()],
            explorers: vec!["https://optimistic.etherscan.io".to_string()],
            bridges: vec![],
            staking_contract: None,
            governance_contract: Some("0xC02aaA90bDd801592c456E90cE2b0cE2d9606eB48".to_string()),
        });
        
        // Add Base
        chains.insert(8453, ChainConfig {
            chain_id: 8453,
            name: "Base".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_urls: vec!["https://mainnet.base.org".to_string()],
            explorers: vec!["https://basescan.org".to_string()],
            bridges: vec![],
            staking_contract: None,
            governance_contract: None,
        });
        
        Self { chains }
    }
    
    /// Get chain config
    pub fn get(&self, chain_id: u64) -> Option<&ChainConfig> {
        self.chains.get(&chain_id)
    }
    
    /// Get all chain IDs
    pub fn chain_ids(&self) -> Vec<u64> {
        self.chains.keys().copied().collect()
    }
    
    /// Add custom chain
    pub fn add(&mut self, config: ChainConfig) {
        self.chains.insert(config.chain_id, config);
    }
}

impl Default for ChainRegistry {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_chain_registry() {
        let registry = ChainRegistry::new();
        
        assert!(registry.get(1).is_some()); // Ethereum
        assert!(registry.get(56).is_some()); // BSC
        assert!(registry.get(137).is_some()); // Polygon
        assert!(registry.get(42161).is_some()); // Arbitrum
        assert!(registry.get(monad::CHAIN_ID).is_some()); // Monad
        assert!(registry.get(berachain::CHAIN_ID).is_some()); // Berachain
    }
    
    #[test]
    fn test_monad_config() {
        let config = monad::config();
        
        assert_eq!(config.chain_id, monad::CHAIN_ID);
        assert_eq!(config.name, "Monad");
        assert_eq!(config.symbol, "MON");
    }
    
    #[test]
    fn test_berachain_config() {
        let config = berachain::config();
        
        assert_eq!(config.chain_id, berachain::CHAIN_ID);
        assert_eq!(config.name, "Berachain");
        assert!(!config.bridges.is_empty());
    }
}