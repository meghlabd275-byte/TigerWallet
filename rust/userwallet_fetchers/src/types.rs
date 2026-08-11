//! TigerWallet UserWallet Fetchers — Types
//!
//! Typed response models for UserWallet data. These mirror the JSON shapes
//! returned by the canonical TigerWallet Go wallet-api backend
//! (go/wallet_api, port 8443). Fetchers that delegate to the backend produce
//! these; fetchers for endpoints the backend does not expose are fail-closed
//! (return `Err`) and never fabricate data.

use serde::{Deserialize, Serialize};

/// Chains supported by the canonical wallet-api backend.
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
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
    Cosmos,
    Near,
    Aptos,
    Sui,
    Starknet,
}

impl Chain {
    pub fn id(&self) -> i64 {
        match self {
            Chain::Ethereum => 1,
            Chain::Bsc => 56,
            Chain::Polygon => 137,
            Chain::Avalanche => 43114,
            Chain::Fantom => 250,
            Chain::Arbitrum => 42161,
            Chain::Optimism => 10,
            Chain::Bitcoin => 0,
            Chain::Solana => 0,
            Chain::Cosmos => 0,
            Chain::Near => 0,
            Chain::Aptos => 0,
            Chain::Sui => 0,
            Chain::Starknet => 0,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WalletRecord {
    pub id: String,
    pub label: String,
    pub chain_id: i64,
    pub address: String,
    pub derivation_path: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mnemonic: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BalanceResult {
    pub chain_id: i64,
    pub symbol: String,
    pub address: String,
    pub balance: String,
    pub balance_f: f64,
    pub usd_value: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionRecord {
    pub hash: String,
    pub from: String,
    pub to: String,
    pub value: String,
    #[serde(rename = "timeStamp")]
    pub time_stamp: String,
    #[serde(rename = "isError")]
    pub is_error: String,
}

/// Token asset (ERC-20 metadata + balance).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub address: String,
    pub chain: String,
    pub symbol: String,
    pub name: String,
    pub decimals: u8,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub logo_url: Option<String>,
    #[serde(default)]
    pub price: f64,
    #[serde(default)]
    pub price_change_24h: f64,
    #[serde(default)]
    pub volume_24h: f64,
    #[serde(default)]
    pub market_cap: f64,
    #[serde(default)]
    pub supply: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Nft {
    pub contract_address: String,
    pub token_id: String,
    pub owner: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub image_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub animation_url: Option<String>,
    #[serde(default)]
    pub attributes: Vec<NftAttribute>,
    #[serde(default)]
    pub collection: NftCollection,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_sale_price: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub listing_price: Option<f64>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct NftAttribute {
    pub trait_type: String,
    pub value: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub display_type: Option<String>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct NftCollection {
    pub address: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub image_url: Option<String>,
    #[serde(default)]
    pub floor_price: f64,
    #[serde(default)]
    pub total_supply: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasPrice {
    pub chain: String,
    #[serde(default)]
    pub slow_gas_price: String,
    #[serde(default)]
    pub standard_gas_price: String,
    #[serde(default)]
    pub fast_gas_price: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub base_fee: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub priority_fee: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceInfo {
    pub symbol: String,
    #[serde(default)]
    pub price: f64,
    #[serde(default)]
    pub price_change_24h: f64,
    #[serde(default)]
    pub price_change_percent_24h: f64,
    #[serde(default)]
    pub volume_24h: f64,
    #[serde(default)]
    pub market_cap: f64,
    #[serde(default)]
    pub high_24h: f64,
    #[serde(default)]
    pub low_24h: f64,
    #[serde(default)]
    pub last_updated: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapQuote {
    pub from_token: String,
    pub to_token: String,
    pub from_amount: String,
    #[serde(default)]
    pub to_amount: String,
    #[serde(default)]
    pub price_impact: f64,
    #[serde(default)]
    pub gas_estimate: u64,
    #[serde(default)]
    pub route: Vec<SwapRoute>,
    pub integrator: String,
    #[serde(default)]
    pub estimated_gas: u64,
    #[serde(default)]
    pub estimated_gas_usd: f64,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct SwapRoute {
    pub pool_address: String,
    pub token_in: String,
    pub token_out: String,
    pub swap_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakingQuote {
    pub asset: String,
    pub apr: f64,
    #[serde(default)]
    pub lock_period_days: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DApp {
    pub id: String,
    pub name: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub logo_url: String,
    #[serde(default)]
    pub website_url: String,
    pub category: String,
    #[serde(default)]
    pub chains: Vec<String>,
    #[serde(default)]
    pub contracts: Vec<String>,
    pub verified: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenRegistryEntry {
    pub address: String,
    pub symbol: String,
    pub name: String,
    pub decimals: u8,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub logo_url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainInfo {
    pub chain_id: i64,
    pub name: String,
    pub symbol: String,
    #[serde(default)]
    pub rpc_url: String,
    #[serde(default)]
    pub explorer_url: String,
}
