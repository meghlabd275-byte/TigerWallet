//! Data models for Backend services

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// Wallet
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Wallet {
    pub id: String,
    pub user_id: String,
    pub address: String,
    pub chain: String,
    pub wallet_type: WalletType,
    pub created_at: DateTime<Utc>,
}

/// Wallet type
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum WalletType {
    EOA,
    Contract,
    MPC,
    Hardware,
}

/// Transaction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub id: String,
    pub hash: String,
    pub from: String,
    pub to: String,
    pub amount: f64,
    pub token: String,
    pub status: TxStatus,
    pub created_at: DateTime<Utc>,
}

/// Transaction status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum TxStatus {
    Pending,
    Confirmed,
    Failed,
}

/// Swap request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapRequest {
    pub from_token: String,
    pub to_token: String,
    pub amount: f64,
    pub slippage: f64,
}

/// Swap result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapResult {
    pub tx_hash: String,
    pub from_amount: f64,
    pub to_amount: f64,
    pub price_impact: f64,
}

/// Staking position
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakingPosition {
    pub id: String,
    pub user_id: String,
    pub token: String,
    pub amount: f64,
    pub rewards: f64,
    pub created_at: DateTime<Utc>,
}

/// NFT
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Nft {
    pub id: String,
    pub token_id: String,
    pub contract: String,
    pub owner: String,
    pub metadata: String,
}

/// Gas estimate
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasEstimate {
    pub gas_price: f64,
    pub estimated_gas: u64,
    pub estimated_cost: f64,
}

/// DApp browser session
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DappSession {
    pub id: String,
    pub url: String,
    pub origin: String,
    pub wallet_address: String,
    pub created_at: DateTime<Utc>,
}

/// Config
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub port: String,
    pub database_url: String,
    pub redis_url: String,
}

impl Default for Config {
    fn default() -> Self {
        Config {
            port: "8080".to_string(),
            database_url: "postgres://localhost:5432/tigerwallet".to_string(),
            redis_url: "localhost:6379".to_string(),
        }
    }
}