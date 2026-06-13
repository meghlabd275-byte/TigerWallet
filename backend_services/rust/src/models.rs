//! Data models for Backend Services

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// API Gateway Config
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

/// API Key
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiKey {
    pub id: String,
    pub key: String,
    pub user_id: String,
    pub permissions: Vec<String>,
    pub created_at: DateTime<Utc>,
    pub expires_at: Option<DateTime<Utc>>,
}

/// Auth Token
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthToken {
    pub token: String,
    pub user_id: String,
    pub expires_at: DateTime<Utc>,
}

/// Event
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Event {
    pub id: String,
    pub event_type: String,
    pub payload: String,
    pub created_at: DateTime<Utc>,
}

/// Secrets
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Secret {
    pub id: String,
    pub name: String,
    pub encrypted_value: String,
    pub created_at: DateTime<Utc>,
}