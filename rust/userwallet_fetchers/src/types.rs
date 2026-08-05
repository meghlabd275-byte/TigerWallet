//! TigerWallet UserWallet Fetchers - Types Module
//! 
//! This module contains all type definitions for the UserWallet fetcher system

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Chain types supported by the wallet
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum Chain {
    Ethereum,
    Bitcoin,
    Solana,
    Polygon,
    Bsc,
    Avalanche,
    Fantom,
    Arbitrum,
    Optimism,
    Avalanche,
    Cosmos,
    Near,
    Aptos,
    Sui,
    Starknet,
    #[serde(other)]
    Unknown,
}

/// Token information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub address: String,
    pub chain: String,
    pub symbol: String,
    pub name: String,
    pub decimals: u8,
    pub logo_url: Option<String>,
    pub price: f64,
    pub price_change_24h: f64,
    pub volume_24h: f64,
    pub market_cap: f64,
    pub supply: f64,
}

/// NFT information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Nft {
    pub contract_address: String,
    pub token_id: String,
    pub owner: String,
    pub name: String,
    pub description: Option<String>,
    pub image_url: Option<String>,
    pub animation_url: Option<String>,
    pub attributes: Vec<NftAttribute>,
    pub collection: NftCollection,
    pub last_sale_price: Option<f64>,
    pub listing_price: Option<f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NftAttribute {
    pub trait_type: String,
    pub value: String,
    pub display_type: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NftCollection {
    pub address: String,
    pub name: String,
    pub description: Option<String>,
    pub image_url: Option<String>,
    pub floor_price: f64,
    pub total_supply: u64,
}

/// Transaction information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub hash: String,
    pub chain: String,
    pub from: String,
    pub to: String,
    pub value: String,
    pub gas_used: u64,
    pub gas_price: String,
    pub timestamp: u64,
    pub status: TransactionStatus,
    pub block_number: u64,
    pub logs: Vec<TransactionLog>,
    pub token_transfers: Vec<TokenTransfer>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TransactionStatus {
    Success,
    Failed,
    Pending,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionLog {
    pub address: String,
    pub topics: Vec<String>,
    pub data: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenTransfer {
    pub from: String,
    pub to: String,
    pub token_address: String,
    pub value: String,
    pub token_id: Option<String>,
}

/// Swap quote
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapQuote {
    pub from_token: String,
    pub to_token: String,
    pub from_amount: String,
    pub to_amount: String,
    pub price_impact: f64,
    pub gas_estimate: u64,
    pub route: Vec<SwapRoute>,
    pub integrator: String,
    pub estimated_gas: u64,
    pub estimated_gas_usd: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapRoute {
    pub pool_address: String,
    pub token_in: String,
    pub token_out: String,
    pub swap_type: String,
}

/// Staking position
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakingPosition {
    pub validator_address: String,
    pub chain: String,
    pub staked_amount: f64,
    pub rewards_accrued: f64,
    pub rewards_claimed: f64,
    pub unbonding_amount: f64,
    pub unbonding_completion_time: Option<u64>,
    pub commission: f64,
    pubapr: f64,
}

/// Gas price information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasPrice {
    pub chain: String,
    pub slow_gas_price: String,
    pub standard_gas_price: String,
    pub fast_gas_price: String,
    pub base_fee: Option<String>,
    pub priority_fee: Option<String>,
}

/// Price information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceInfo {
    pub symbol: String,
    pub price: f64,
    pub price_change_24h: f64,
    pub price_change_percent_24h: f64,
    pub volume_24h: f64,
    pub market_cap: f64,
    pub high_24h: f64,
    pub low_24h: f64,
    pub last_updated: u64,
}

/// Bridge information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeQuote {
    pub from_chain: String,
    pub to_chain: String,
    pub from_token: String,
    pub to_token: String,
    pub from_amount: String,
    pub to_amount: String,
    pub estimated_time: u64,
    pub bridge_fee: String,
    pub route: Vec<BridgeRoute>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeRoute {
    pub bridge_name: String,
    pub from_chain: String,
    pub to_chain: String,
}

/// Lending market
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LendingMarket {
    pub pool_address: String,
    pub token: String,
    pub total_supply: f64,
    pub total_borrow: f64,
    pub supply_apr: f64,
    pub borrow_apr: f64,
    pub utilization_rate: f64,
    pub collateral_factor: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LendingPosition {
    pub pool_address: String,
    pub user: String,
    pub supplied_amount: f64,
    pub borrowed_amount: f64,
    pub collateral_value: f64,
    pub health_factor: f64,
}

/// Options data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OptionData {
    pub contract_address: String,
    pub underlying: String,
    pub strike_price: f64,
    pub expiry: u64,
    pub option_type: OptionType,
    pub bid_price: f64,
    pub ask_price: f64,
    pub last_price: f64,
    pub volume_24h: f64,
    pub open_interest: f64,
    pub implied_volatility: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum OptionType {
    Call,
    Put,
}

/// Futures data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FuturesData {
    pub contract_address: String,
    pub symbol: String,
    pub index_price: f64,
    pub mark_price: f64,
    pub funding_rate: f64,
    pub next_funding_time: u64,
    pub open_interest: f64,
    pub volume_24h: f64,
    pub long_ratio: f64,
    pub predicted_funding: f64,
}

/// Margin position
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarginPosition {
    pub user: String,
    pub pool_address: String,
    pub collateral_amount: f64,
    pub borrowed_amount: f64,
    pub position_size: f64,
    pub leverage: f64,
    pub entry_price: f64,
    pub mark_price: f64,
    pub pnl: f64,
    pub pnl_percent: f64,
    pub liquidation_price: f64,
    pub health_factor: f64,
}

/// P2P Order
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct P2POrder {
    pub order_id: String,
    pub maker: String,
    pub side: P2PSide,
    pub from_token: String,
    pub to_token: String,
    pub from_amount: String,
    pub to_amount: String,
    pub price: f64,
    pub status: P2POrderStatus,
    pub created_at: u64,
    pub expires_at: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum P2PSide {
    Buy,
    Sell,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum P2POrderStatus {
    Open,
    Filled,
    Cancelled,
    Expired,
}

/// Copy trading
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CopyTradingSignal {
    pub signal_id: String,
    pub trader: String,
    pub chain: String,
    pub action: String,
    pub token: String,
    pub amount: f64,
    pub price: f64,
    pub timestamp: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CopyTradingPosition {
    pub position_id: String,
    pub trader: String,
    pub follower: String,
    pub token: String,
    pub amount: f64,
    pub entry_price: f64,
    pub current_price: f64,
    pub pnl: f64,
    pub pnl_percent: f64,
    pub copied_at: u64,
}

/// DAO Governance
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DAOProposal {
    pub proposal_id: String,
    pub dao_address: String,
    pub title: String,
    pub description: String,
    pub proposer: String,
    pub status: DAOProposalStatus,
    pub vote_start: u64,
    pub vote_end: u64,
    pub for_votes: f64,
    pub against_votes: f64,
    pub abstain_votes: f64,
    pub quorum: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DAOProposalStatus {
    Pending,
    Active,
    Cancelled,
    Defeated,
    Succeeded,
    Executed,
}

/// Gift Card
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GiftCard {
    pub card_id: String,
    pub code: String,
    pub balance: f64,
    pub currency: String,
    pub issued_by: String,
    pub redeemed_by: Option<String>,
    pub expires_at: Option<u64>,
    pub created_at: u64,
    pub status: GiftCardStatus,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum GiftCardStatus {
    Active,
    Redeemed,
    Expired,
    Cancelled,
}

/// Fiat Ramp
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FiatQuote {
    pub provider: String,
    pub from_currency: String,
    pub to_crypto: String,
    pub from_amount: f64,
    pub to_amount: f64,
    pub rate: f64,
    pub fee: f64,
    pub payment_methods: Vec<String>,
    pub estimated_time: String,
}

/// DApp Registry
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DApp {
    pub address: String,
    pub name: String,
    pub description: String,
    pub logo_url: String,
    pub website_url: String,
    pub category: String,
    pub chains: Vec<String>,
    pub contracts: Vec<String>,
    pub verified: bool,
    pub trust_score: f64,
    pub volume_24h: f64,
}

/// Price Alert
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceAlert {
    pub alert_id: String,
    pub user: String,
    pub token: String,
    pub condition: AlertCondition,
    pub target_price: f64,
    pub created_at: u64,
    pub triggered_at: Option<u64>,
    pub active: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AlertCondition {
    Above,
    Below,
}

/// Fetcher configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetcherConfig {
    pub rpc_urls: HashMap<String, String>,
    pub api_keys: HashMap<String, String>,
    pub cache_ttl: u64,
    pub rate_limit: u64,
    pub max_retries: u32,
}

impl Default for FetcherConfig {
    fn default() -> Self {
        let mut rpc_urls = HashMap::new();
        rpc_urls.insert("ethereum".to_string(), "https://eth-mainnet.g.alchemy.com/v2/demo".to_string());
        rpc_urls.insert("polygon".to_string(), "https://polygon-mainnet.g.alchemy.com/v2/demo".to_string());
        rpc_urls.insert("bsc".to_string(), "https://bsc-dataseed.binance.org".to_string());
        
        let mut api_keys = HashMap::new();
        api_keys.insert("coingecko".to_string(), "".to_string());
        api_keys.insert("alchemy".to_string(), "demo".to_string());
        
        Self {
            rpc_urls,
            api_keys,
            cache_ttl: 60,
            rate_limit: 100,
            max_retries: 3,
        }
    }
}
