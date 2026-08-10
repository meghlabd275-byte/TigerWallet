//! Type definitions for TigerWallet Full Fetchers

use std::collections::HashMap;
use std::time::{SystemTime, UNIX_EPOCH};
use serde::{Deserialize, Serialize};

/// Timestamp in milliseconds
pub type Timestamp = u64;
/// Chain ID
pub type ChainId = u64;
/// Gas price in gwei
pub type GasPrice = u64;
/// Token amount as string
pub type TokenAmount = String;
/// Ethereum address
pub type Address = String;

/// Token metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenMetadata {
    pub address: Address,
    pub name: String,
    pub symbol: String,
    pub decimals: u8,
    pub logo_url: String,
    pub total_supply: String,
    pub is_verified: bool,
    pub last_updated: Timestamp,
}

/// Price data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceData {
    pub token_address: Address,
    pub price_usd: f64,
    pub price_eth: f64,
    pub change_24h: f64,
    pub volume_24h: f64,
    pub market_cap: f64,
    pub timestamp: Timestamp,
    pub confidence: u8,
}

/// Gas data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasData {
    pub chain_id: ChainId,
    pub gas_price_gwei: GasPrice,
    pub gas_limit: u64,
    pub estimated_gas: u64,
    pub max_fee_per_gas: u64,
    pub max_priority_fee_per_gas: u64,
    pub network_congestion: String,
    pub timestamp: Timestamp,
}

/// Network data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkData {
    pub chain_id: ChainId,
    pub name: String,
    pub symbol: String,
    pub rpc_url: String,
    pub block_number: u64,
    pub block_time_ms: u64,
    pub gas_limit: u64,
    pub network_status: String,
    pub last_synced: Timestamp,
}

/// Swap quote
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapQuote {
    pub id: String,
    pub from_token: Address,
    pub to_token: Address,
    pub from_amount: TokenAmount,
    pub to_amount: TokenAmount,
    pub price_impact: f64,
    pub gas_limit: u64,
    pub estimated_gas: u64,
    pub route: Vec<SwapRoute>,
    pub expires_at: Timestamp,
}

/// Swap route step
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapRoute {
    pub protocol: String,
    pub from_token: Address,
    pub to_token: Address,
    pub from_amount: TokenAmount,
    pub to_amount: TokenAmount,
    pub fee_percentage: f64,
}

/// MEV opportunity
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MEVOpportunity {
    pub opportunity_type: String, // sandwich, arbitrage, liquidation
    pub front_run_tx: String,
    pub back_run_tx: String,
    pub estimated_profit_eth: f64,
    pub estimated_profit_usd: f64,
    pub affected_addresses: Vec<Address>,
    pub block_number: u64,
    pub detected_at: Timestamp,
}

/// Liquidity data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LiquidityData {
    pub pair_address: Address,
    pub token_a: Address,
    pub token_b: Address,
    pub reserve_a: f64,
    pub reserve_b: f64,
    pub liquidity_usd: f64,
    pub volume_24h: f64,
    pub fees_24h: f64,
    pub last_updated: Timestamp,
}

/// Arbitrage opportunity
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ArbitrageOpportunity {
    pub dex_a: String,
    pub dex_b: String,
    pub token_a: Address,
    pub token_b: Address,
    pub price_diff_percentage: f64,
    pub max_trade_amount: f64,
    pub estimated_profit: f64,
    pub profitable_block: u64,
}

/// Token risk data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenRiskData {
    pub token_address: Address,
    pub risk_score: u8, // 0-100
    pub risk_level: String, // low, medium, high, critical
    pub is_verified: bool,
    pub is_honeypot: bool,
    pub is_pausable: bool,
    pub is_mintable: bool,
    pub has_blacklist: bool,
    pub holder_count: f64,
    pub transfer_count_24h: f64,
    pub flags: Vec<String>,
    pub analyzed_at: Timestamp,
}

/// Smart contract info
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContractInfo {
    pub contract_address: Address,
    pub contract_type: String,
    pub source_code: String,
    pub is_verified: bool,
    pub compiler_version: String,
    pub functions: Vec<String>,
    pub abi: HashMap<String, String>,
    pub last_verified: Timestamp,
}

/// DeFi yield data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct YieldData {
    pub protocol: String,
    pub pool_address: Address,
    pub reward_token: Address,
    pub apy: f64,
    pub tvl: f64,
    pub reward_rate: f64,
    pub lock_period: u64,
    pub risk_level: String,
    pub last_updated: Timestamp,
}

/// Staking data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakingData {
    pub validator: Address,
    pub network: String,
    pub total_staked: f64,
    pub rewards_earned: f64,
    pub commission: f64,
    pub uptime_percentage: f64,
    pub last_reward_block: u64,
}

/// NFT floor price
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NFTFloorPrice {
    pub collection_address: Address,
    pub collection_name: String,
    pub floor_price_eth: f64,
    pub floor_price_usd: f64,
    pub volume_24h: f64,
    pub sales_24h: u64,
    pub average_price: f64,
    pub last_sale: Timestamp,
}

/// Whale transaction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WhaleTransaction {
    pub tx_hash: String,
    pub from: Address,
    pub to: Address,
    pub amount: TokenAmount,
    pub amount_usd: f64,
    pub token_symbol: String,
    pub timestamp: Timestamp,
    pub block_number: u64,
}

/// On-chain analytics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OnChainAnalytics {
    pub chain_id: ChainId,
    pub total_value_locked: f64,
    pub total_volume_24h: f64,
    pub total_transactions_24h: f64,
    pub average_gas_price: f64,
    pub active_addresses: u64,
    pub defi_tvl: f64,
    pub nft_volume: f64,
    pub timestamp: Timestamp,
}

/// Transaction simulation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SimulationResult {
    pub tx_hash: String,
    pub success: bool,
    pub revert_reason: String,
    pub gas_used: u64,
    pub state_changes: String,
    pub estimated_value: f64,
    pub logs: Vec<LogEvent>,
    pub simulated_at: Timestamp,
}

/// Log event
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LogEvent {
    pub address: Address,
    pub topics: Vec<String>,
    pub data: String,
    pub log_index: u64,
}

/// Cross-chain route
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CrossChainRoute {
    pub from_chain: String,
    pub to_chain: String,
    pub from_token: Address,
    pub to_token: Address,
    pub from_amount: TokenAmount,
    pub to_amount: TokenAmount,
    pub price_impact: f64,
    pub estimated_time_minutes: u64,
    pub total_fee_usd: f64,
    pub steps: Vec<BridgeStep>,
}

/// Bridge step
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeStep {
    pub protocol: String,
    pub from_chain: String,
    pub to_chain: String,
    pub from_token: Address,
    pub to_token: Address,
}

/// Fetcher statistics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetcherStats {
    pub name: String,
    pub last_latency_ns: u64,
    pub total_requests: u64,
    pub successful_requests: u64,
    pub success_rate: f64,
}

impl Default for FetcherStats {
    fn default() -> Self {
        Self {
            name: String::new(),
            last_latency_ns: 0,
            total_requests: 0,
            successful_requests: 0,
            success_rate: 0.0,
        }
    }
}

/// Price prediction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PricePrediction {
    pub token: Address,
    pub current_price: f64,
    pub predictions: HashMap<u64, f64>, // horizon -> predicted price
    pub confidence: f64,
    pub predicted_at: Timestamp,
}

/// WalletConnect session
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WCSession {
    pub topic: String,
    pub wallet_address: Address,
    pub peer_metadata: String,
    pub chain_id: String,
    pub created_at: Timestamp,
    pub updated_at: Timestamp,
    pub expires_at: Timestamp,
}

/// Current timestamp in milliseconds
pub fn current_timestamp() -> Timestamp {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_millis() as Timestamp
}
