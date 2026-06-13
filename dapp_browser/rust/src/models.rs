//! Data models for DApp Browser

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// DApp Session
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DappSession {
    pub id: String,
    pub url: String,
    pub origin: String,
    pub wallet_address: String,
    pub created_at: DateTime<Utc>,
}

/// WalletConnect Session
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WalletConnectSession {
    pub topic: String,
    pub peer_meta: PeerMeta,
    pub accounts: Vec<String>,
    pub chain_id: u64,
    pub created_at: DateTime<Utc>,
}

/// Peer Metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PeerMeta {
    pub name: String,
    pub url: String,
    pub icons: Vec<String>,
}

/// JSON-RPC Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JsonRpcRequest {
    pub jsonrpc: String,
    pub method: String,
    pub params: Vec<serde_json::Value>,
    pub id: u64,
}

/// JSON-RPC Response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JsonRpcResponse {
    pub jsonrpc: String,
    pub result: Option<serde_json::Value>,
    pub error: Option<JsonRpcError>,
    pub id: u64,
}

/// JSON-RPC Error
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JsonRpcError {
    pub code: i32,
    pub message: String,
}

/// Config
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub port: String,
    pub relay_url: String,
}

impl Default for Config {
    fn default() -> Self {
        Config {
            port: "8080".to_string(),
            relay_url: "wss://relay.walletconnect.org".to_string(),
        }
    }
}