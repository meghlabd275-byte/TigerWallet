//! Data models for API Gateway

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// Config
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub port: String,
    pub backend_url: String,
    pub rate_limit: u64,
}

impl Default for Config {
    fn default() -> Self {
        Config {
            port: "8080".to_string(),
            backend_url: "http://localhost:8081".to_string(),
            rate_limit: 1000,
        }
    }
}

/// Health response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthResponse {
    pub status: String,
    pub timestamp: i64,
}

/// Auth Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthRequest {
    pub username: String,
    pub password: String,
}

/// Auth Response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthResponse {
    pub token: String,
    pub expires_at: DateTime<Utc>,
}

/// Security Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityRequest {
    pub user_id: String,
    pub action: String,
}

/// License Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LicenseRequest {
    pub user_id: String,
    pub license_type: String,
}

/// White Label Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WhiteLabelRequest {
    pub name: String,
    pub domain: String,
    pub branding: BrandingRequest,
}

/// Branding Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BrandingRequest {
    pub logo: String,
    pub primary_color: String,
}

/// Master Wallet Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MasterWalletRequest {
    pub chain: String,
    pub wallet_type: String,
}

/// Chain Management
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainConfig {
    pub id: String,
    pub name: String,
    pub chain_id: u64,
    pub rpc_url: String,
    pub explorer_url: String,
    pub status: String,
}

/// External Trading
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradingRequest {
    pub from_token: String,
    pub to_token: String,
    pub amount: f64,
    pub slippage: f64,
}

/// Bot Subscription
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BotSubscription {
    pub id: String,
    pub user_id: String,
    pub bot_type: String,
    pub status: String,
}