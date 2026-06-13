//! Data models for Services

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

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

/// Health response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthResponse {
    pub status: String,
    pub timestamp: i64,
}

/// Admin Service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminUser {
    pub id: String,
    pub username: String,
    pub email: String,
    pub role: String,
    pub permissions: Vec<String>,
    pub created_at: DateTime<Utc>,
}

/// API Key
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiKey {
    pub id: String,
    pub key: String,
    pub user_id: String,
    pub permissions: Vec<String>,
    pub created_at: DateTime<Utc>,
}

/// Auth Service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthToken {
    pub token: String,
    pub user_id: String,
    pub expires_at: DateTime<Utc>,
}

/// Compliance Service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceRecord {
    pub id: String,
    pub user_id: String,
    pub status: String,
    pub kyc_level: u8,
    pub created_at: DateTime<Utc>,
}

/// Event Stream
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Event {
    pub id: String,
    pub event_type: String,
    pub payload: String,
    pub created_at: DateTime<Utc>,
}

/// Exchange Service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExchangeRate {
    pub from: String,
    pub to: String,
    pub rate: f64,
    pub updated_at: DateTime<Utc>,
}

/// Staking Service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakingPosition {
    pub id: String,
    pub user_id: String,
    pub token: String,
    pub amount: f64,
    pub rewards: f64,
    pub created_at: DateTime<Utc>,
}

/// Treasury Service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TreasuryWallet {
    pub id: String,
    pub address: String,
    pub chain: String,
    pub balance: f64,
    pub currency: String,
}

/// ENS Service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnsRecord {
    pub name: String,
    pub owner: String,
    pub resolver: String,
    pub expires_at: DateTime<Utc>,
}

/// Social Service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SocialAccount {
    pub id: String,
    pub user_id: String,
    pub platform: String,
    pub username: String,
}

/// White Label
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WhiteLabelConfig {
    pub id: String,
    pub name: String,
    pub branding: BrandingConfig,
    pub created_at: DateTime<Utc>,
}

/// Branding Config
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BrandingConfig {
    pub logo: String,
    pub primary_color: String,
    pub secondary_color: String,
}

/// Observability
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Metrics {
    pub requests_total: u64,
    pub errors_total: u64,
    pub latency_avg: f64,
}

/// Institutional
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InstitutionalAccount {
    pub id: String,
    pub name: String,
    pub account_type: String,
    pub kyc_level: u8,
    pub created_at: DateTime<Utc>,
}

/// Data Platform
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketData {
    pub symbol: String,
    pub price: f64,
    pub volume_24h: f64,
    pub change_24h: f64,
    pub updated_at: DateTime<Utc>,
}

/// Real-time Service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RealtimeSubscription {
    pub id: String,
    pub channel: String,
    pub user_id: String,
}