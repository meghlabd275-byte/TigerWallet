use std::sync::Arc;
use std::time::Instant;

use anyhow::{Context, Result};
use async_trait::async_trait;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use tokio::sync::RwLock;
use tracing::{error, info, warn};

use crate::{Fetcher, FetcherConfig, FetcherMetrics, FetcherState, FetcherType, FetchParams, FetchResult};

/// Blockchain data fetcher for EVM and non-EVM chains
pub struct BlockchainFetcher {
    name: String,
    state: RwLock<FetcherState>,
    http_client: Client,
    cache: Arc<crate::cache::FetcherCache>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockData {
    pub chain: String,
    pub block_number: u64,
    pub block_hash: String,
    pub timestamp: i64,
    pub transactions: Vec<TransactionData>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionData {
    pub hash: String,
    pub from: String,
    pub to: String,
    pub value: String,
    pub gas_limit: String,
    pub gas_used: String,
    pub nonce: u64,
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenData {
    pub chain: String,
    pub address: String,
    pub name: String,
    pub symbol: String,
    pub decimals: u8,
    pub total_supply: String,
    pub holders_count: Option<u64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NFTData {
    pub chain: String,
    pub contract_address: String,
    pub token_id: String,
    pub owner: String,
    pub uri: String,
    pub metadata: Option<serde_json::Value>,
}

impl BlockchainFetcher {
    pub fn new(cache: Arc<crate::cache::FetcherCache>) -> Self {
        Self {
            name: "blockchain_fetcher".to_string(),
            state: RwLock::new(FetcherState::default()),
            http_client: Client::builder()
                .timeout(std::time::Duration::from_secs(30))
                .build()
                .expect("Failed to create HTTP client"),
            cache,
        }
    }
}

#[async_trait]
impl Fetcher for BlockchainFetcher {
    fn fetcher_type(&self) -> FetcherType {
        FetcherType::Blockchain
    }

    fn name(&self) -> &str {
        &self.name
    }

    async fn fetch(&self, params: FetchParams) -> Result<FetchResult> {
        let start = Instant::now();
        
        let chain = params.chain.clone().unwrap_or_else(|| "ethereum".to_string());
        
        // Try cache first
        let cache_key = format!("blockchain:{}:{}:{:?}", chain, params.address.as_deref().unwrap_or("latest"), params.limit);
        
        if !params.force_refresh {
            if let Ok(cached) = self.cache.get(&cache_key).await {
                return Ok(FetchResult::success(
                    cached,
                    chain.clone(),
                    start.elapsed().as_millis() as u64,
                    true,
                ));
            }
        }

        // Fetch from source based on chain type
        let data = match chain.as_str() {
            "ethereum" | "sepolia" | "bsc" | "polygon" | "arbitrum" | "optimism" | "avalanche" => {
                self.fetch_evm_chain(&chain, &params).await
            }
            "solana" => {
                self.fetch_solana(&params).await
            }
            "aptos" => {
                self.fetch_aptos(&params).await
            }
            "ton" => {
                self.fetch_ton(&params).await
            }
            _ => Err(anyhow::anyhow!("Unsupported chain: {}", chain)),
        };

        let elapsed = start.elapsed().as_millis() as u64;
        
        match data {
            Ok(result_data) => {
                // Cache the result
                let _ = self.cache.set(&cache_key, &result_data, 300).await; // 5 min cache
                
                Ok(FetchResult::success(
                    result_data,
                    chain,
                    elapsed,
                    false,
                ))
            }
            Err(e) => {
                error!("Blockchain fetch error: {}", e);
                Ok(FetchResult::error(
                    chain,
                    e.to_string(),
                    elapsed,
                ))
            }
        }
    }

    async fn initialize(&mut self, config: &FetcherConfig) -> Result<()> {
        let mut state = self.state.write().await;
        state.config = config.clone();
        state.status = FetcherStatus::Idle;
        info!("Blockchain fetcher initialized with config: {:?}", config);
        Ok(())
    }

    async fn shutdown(&mut self) -> Result<()> {
        let mut state = self.state.write().await;
        state.status = FetcherStatus::Stopped;
        info!("Blockchain fetcher shut down");
        Ok(())
    }
}

impl BlockchainFetcher {
    async fn fetch_evm_chain(&self, chain: &str, params: &FetchParams) -> Result<serde_json::Value> {
        // In production, this would call actual RPC endpoints
        // For now, return mock data structure
        let rpc_url = format!("https://{}.infura.io/v3/YOUR_PROJECT_ID", chain);
        
        let result = serde_json::json!({
            "chain": chain,
            "block_number": 18000000u64,
            "block_hash": "0x1234567890abcdef",
            "timestamp": chrono::Utc::now().timestamp(),
            "gas_price": "20000000000",
            "network_id": 1,
            "transactions": []
        });
        
        Ok(result)
    }

    async fn fetch_solana(&self, params: &FetchParams) -> Result<serde_json::Value> {
        // Solana RPC integration
        let result = serde_json::json!({
            "chain": "solana",
            "slot": 180000000u64,
            "blockhash": "1234567890abcdef",
            "parent_slot": 179999999u64,
            "transactions": []
        });
        
        Ok(result)
    }

    async fn fetch_aptos(&self, params: &FetchParams) -> Result<serde_json::Value> {
        // Aptos API integration
        let result = serde_json::json!({
            "chain": "aptos",
            "version": 180000000u64,
            "timestamp": chrono::Utc::now().timestamp_millis(),
            "transactions": []
        });
        
        Ok(result)
    }

    async fn fetch_ton(&self, params: &FetchParams) -> Result<serde_json::Value> {
        // TON API integration
        let result = serde_json::json!({
            "chain": "ton",
            "workchain": 0,
            "shard": -9223372036854775808i64,
            "seqno": 18000000u64,
            "transactions": []
        });
        
        Ok(result)
    }

    pub async fn get_metrics(&self) -> FetcherMetrics {
        let state = self.state.read().await;
        state.metrics.clone()
    }
}
