//! Data models for Staking Hub

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// Staking Position
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakingPosition {
    pub id: String,
    pub user_id: String,
    pub token: String,
    pub amount: f64,
    pub rewards: f64,
    pub lock_period: u64,
    pub started_at: DateTime<Utc>,
    pub unlocked_at: Option<DateTime<Utc>>,
}

/// Validator
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Validator {
    pub id: String,
    pub name: String,
    pub address: String,
    pub commission: f64,
    pub uptime: f64,
    pub total_staked: f64,
}

/// Pool
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakingPool {
    pub id: String,
    pub name: String,
    pub token: String,
    pub apy: f64,
    pub min_stake: f64,
    pub max_stake: f64,
    pub lock_period: u64,
    pub validators: Vec<Validator>,
}

/// Stake Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakeRequest {
    pub user_id: String,
    pub pool_id: String,
    pub amount: f64,
}

/// Unstake Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnstakeRequest {
    pub user_id: String,
    pub position_id: String,
    pub amount: f64,
}

/// Claim Rewards Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClaimRewardsRequest {
    pub user_id: String,
    pub position_id: String,
}

/// Config
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub port: String,
    pub database_url: String,
}

impl Default for Config {
    fn default() -> Self {
        Config {
            port: "8080".to_string(),
            database_url: "postgres://localhost:5432/tigerwallet".to_string(),
        }
    }
}