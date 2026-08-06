//! Core types for White Level SDK

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// White Level Product Types
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum WhiteLevelProduct {
    MasterWallet,
    UserWallet,
    Bots,
    BotsClients,
    ProjectParty,
}

impl std::fmt::Display for WhiteLevelProduct {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            WhiteLevelProduct::MasterWallet => write!(f, "master_wallet"),
            WhiteLevelProduct::UserWallet => write!(f, "user_wallet"),
            WhiteLevelProduct::Bots => write!(f, "bots"),
            WhiteLevelProduct::BotsClients => write!(f, "bots_clients"),
            WhiteLevelProduct::ProjectParty => write!(f, "project_party"),
        }
    }
}

/// Fetcher Types
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum FetcherType {
    Prices,
    Balances,
    Transactions,
    UserData,
    MarketData,
    Blockchain,
    TokenInfo,
    KYC,
    NftData,
    GasPrice,
    NetworkStatus,
}

impl std::fmt::Display for FetcherType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            FetcherType::Prices => write!(f, "prices"),
            FetcherType::Balances => write!(f, "balances"),
            FetcherType::Transactions => write!(f, "transactions"),
            FetcherType::UserData => write!(f, "user_data"),
            FetcherType::MarketData => write!(f, "market_data"),
            FetcherType::Blockchain => write!(f, "blockchain"),
            FetcherType::TokenInfo => write!(f, "token_info"),
            FetcherType::KYC => write!(f, "kyc"),
            FetcherType::NftData => write!(f, "nft_data"),
            FetcherType::GasPrice => write!(f, "gas_price"),
            FetcherType::NetworkStatus => write!(f, "network_status"),
        }
    }
}

/// Permission Level
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum PermissionLevel {
    None,
    Read,
    Write,
    Execute,
    Admin,
    SuperAdmin,
}

/// Connection Status
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ConnectionStatus {
    Connected,
    Disconnected,
    Error,
    Timeout,
    Reconnecting,
}

/// Client Status
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ClientStatus {
    Active,
    Suspended,
    Terminated,
    Pending,
}

/// Connection request from White Level product
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectionRequest {
    pub client_id: Uuid,
    pub product: WhiteLevelProduct,
    pub api_key: String,
    pub client_info: ClientInfo,
    pub ip_address: Option<String>,
    pub region: Option<String>,
}

/// Client information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientInfo {
    pub name: String,
    pub version: String,
    pub platform: String,
    pub hostname: String,
    pub ip_address: Option<String>,
    pub metadata: Option<serde_json::Value>,
}

/// Connection response from Super Admin
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectionResponse {
    pub connection_id: Uuid,
    pub connection_key: String,
    pub session_token: String,
    pub expires_at: DateTime<Utc>,
    pub config: ClientConfig,
}

/// Client configuration from Super Admin
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientConfig {
    pub heartbeat_interval: u64,      // seconds
    pub reconnect_timeout: u64,       // seconds
    pub max_reconnects: u32,
    pub timeout_ms: u32,
    pub rate_limit: u32,             // requests per minute
    pub features: Vec<String>,
    pub fetchers: Vec<FetcherConfig>,
}

/// Fetcher configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetcherConfig {
    pub fetcher: FetcherType,
    pub endpoint: String,
    pub timeout_ms: u32,
    pub retry_count: u32,
    pub cache_ttl_seconds: u32,
    pub is_enabled: bool,
}

/// Heartbeat payload
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Heartbeat {
    pub connection_key: String,
    pub status: ConnectionStatus,
    pub latency_ms: u32,
    pub metrics: ConnectionMetrics,
}

/// Connection metrics
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ConnectionMetrics {
    pub requests_sent: u64,
    pub requests_failed: u64,
    pub bytes_sent: u64,
    pub bytes_received: u64,
    pub uptime_seconds: u64,
    pub memory_usage_mb: u64,
    pub cpu_usage_percent: f32,
}

/// Permission structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Permission {
    pub product: WhiteLevelProduct,
    pub fetcher: FetcherType,
    pub level: PermissionLevel,
    pub is_enabled: bool,
}

/// Fetcher data request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetcherRequest {
    pub fetcher: FetcherType,
    pub params: serde_json::Value,
    pub cache: bool,
}

/// Fetcher data response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetcherResponse {
    pub data: serde_json::Value,
    pub cached: bool,
    pub latency_ms: u64,
    pub timestamp: DateTime<Utc>,
}

/// Fetcher update from Super Admin
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetcherUpdate {
    pub fetcher: FetcherType,
    pub version: String,
    pub endpoint: String,
    pub config: serde_json::Value,
    pub force_update: bool,
}

/// Data sync request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SyncRequest {
    pub sync_type: SyncType,
    pub data: serde_json::Value,
    pub timestamp: DateTime<Utc>,
}

/// Data sync type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum SyncType {
    Full,
    Incremental,
    Delta,
}

/// Data sync response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SyncResponse {
    pub sync_id: Uuid,
    pub status: SyncStatus,
    pub data: Option<serde_json::Value>,
    pub timestamp: DateTime<Utc>,
}

/// Sync status
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum SyncStatus {
    Pending,
    InProgress,
    Completed,
    Failed,
}

/// Remote command from Super Admin (kill switch)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RemoteCommand {
    pub command_id: Uuid,
    pub command: CommandType,
    pub params: Option<serde_json::Value>,
    pub timestamp: DateTime<Utc>,
    pub expires_at: Option<DateTime<Utc>>,
}

/// Command types for remote control
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum CommandType {
    Disable,
    Enable,
    UpdateConfig,
    Restart,
    Shutdown,
    ClearCache,
    ForceSync,
    UpdateFetcher,
}

/// Remote command result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CommandResult {
    pub command_id: Uuid,
    pub status: CommandStatus,
    pub message: String,
    pub timestamp: DateTime<Utc>,
}

/// Command execution status
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum CommandStatus {
    Pending,
    Executing,
    Completed,
    Failed,
    Timeout,
}

/// Event from Super Admin
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", content = "payload")]
pub enum AdminEvent {
    PermissionChanged(Permission),
    FetcherUpdate(FetcherUpdate),
    ConfigUpdate(ClientConfig),
    Command(RemoteCommand),
    Disconnect { reason: String },
    Maintenance { start_at: DateTime<Utc>, duration_minutes: u32 },
}
