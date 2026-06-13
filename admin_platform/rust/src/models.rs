//! Data models for Admin Platform

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// Chain information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Chain {
    pub id: String,
    pub name: String,
    pub symbol: String,
    pub chain_id: u64,
    pub rpc_url: String,
    pub explorer_url: String,
    pub status: ChainStatus,
}

/// Chain status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum ChainStatus {
    Active,
    Inactive,
    Deprecated,
}

/// Listing request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListingRequest {
    pub id: String,
    pub token: String,
    pub symbol: String,
    pub requester: String,
    pub status: ListingStatus,
    pub created_at: DateTime<Utc>,
}

/// Listing status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum ListingStatus {
    Pending,
    Approved,
    Rejected,
}

/// Master wallet
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MasterWallet {
    pub id: String,
    pub address: String,
    pub chain: String,
    pub balance: f64,
    pub status: WalletStatus,
}

/// Wallet status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum WalletStatus {
    Active,
    Inactive,
    Frozen,
}

/// Admin user
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminUser {
    pub id: String,
    pub username: String,
    pub email: String,
    pub role: AdminRole,
    pub permissions: Vec<String>,
    pub created_at: DateTime<Utc>,
}

/// Admin role
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum AdminRole {
    SuperAdmin,
    Admin,
    Moderator,
    Viewer,
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

/// Health response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthResponse {
    pub status: String,
    pub timestamp: i64,
}