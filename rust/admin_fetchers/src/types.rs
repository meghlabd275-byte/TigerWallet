/**
 * TigerWallet Admin Fetchers - Types Module
 * Complete type definitions for all admin operations
 */

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// User type for admin operations
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    pub id: String,
    pub email: String,
    pub username: String,
    pub status: UserStatus,
    pub kyc_status: KycStatus,
    pub kyc_level: i32,
    pub balance: HashMap<String, String>,
    pub total_volume: String,
    pub created_at: i64,
    pub updated_at: i64,
    pub last_login: Option<i64>,
    pub ip_address: Option<String>,
    pub country: Option<String>,
    pub verified: bool,
    pub suspended: bool,
    pub suspend_reason: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum UserStatus {
    Active,
    Suspended,
    Locked,
    Deleted,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum KycStatus {
    None,
    Pending,
    Level1,
    Level2,
    Level3,
    Approved,
    Rejected,
}

/// KYC request type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KycRequest {
    pub id: String,
    pub user_id: String,
    pub user_email: String,
    pub doc_type: KycDocumentType,
    pub status: KycStatus,
    pub document_url: String,
    pub front_image: Option<String>,
    pub back_image: Option<String>,
    pub selfie_image: Option<String>,
    pub submitted_at: i64,
    pub reviewed_at: Option<i64>,
    pub reviewed_by: Option<String>,
    pub reject_reason: Option<String>,
    pub notes: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum KycDocumentType {
    Passport,
    IdCard,
    DriverLicense,
    UtilityBill,
    BankStatement,
}

/// System health type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemHealth {
    pub status: SystemStatus,
    pub uptime_seconds: u64,
    pub version: String,
    pub services: Vec<ServiceStatus>,
    pub database: DatabaseStatus,
    pub cache: CacheStatus,
    pub cpu_usage: f64,
    pub memory_usage: MemoryUsage,
    pub disk_usage: DiskUsage,
    pub network: NetworkStatus,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SystemStatus {
    Healthy,
    Degraded,
    Unhealthy,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceStatus {
    pub name: String,
    pub status: String,
    pub latency_ms: u64,
    pub requests_per_second: f64,
    pub error_rate: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseStatus {
    pub connected: bool,
    pub pool_size: u32,
    pub active_connections: u32,
    pub idle_connections: u32,
    pub queries_per_second: f64,
    pub avg_query_time_ms: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CacheStatus {
    pub connected: bool,
    pub used_memory: u64,
    pub max_memory: u64,
    pub hit_rate: f64,
    pub miss_rate: f64,
    pub keys: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryUsage {
    pub used: u64,
    pub total: u64,
    pub percentage: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DiskUsage {
    pub used: u64,
    pub total: u64,
    pub percentage: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkStatus {
    pub bytes_sent: u64,
    pub bytes_received: u64,
    pub packets_sent: u64,
    pub packets_received: u64,
}

/// Fee configuration type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeeConfig {
    pub id: String,
    pub fee_type: FeeType,
    pub chain_id: Option<u64>,
    pub token_symbol: Option<String>,
    pub percentage: f64,
    pub fixed: String,
    pub min_fee: String,
    pub max_fee: Option<String>,
    pub is_active: bool,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum FeeType {
    Trading,
    Withdrawal,
    Deposit,
    Nft,
    Fiat,
    Conversion,
}

/// Token type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub id: String,
    pub symbol: String,
    pub name: String,
    pub contract_address: Option<String>,
    pub chain: String,
    pub decimals: i32,
    pub total_supply: String,
    pub circulating_supply: String,
    pub status: TokenStatus,
    pub token_type: TokenType,
    pub is_verified: bool,
    pub price_usd: String,
    pub market_cap: String,
    pub volume_24h: String,
    pub price_change_24h: f64,
    pub created_at: i64,
    pub updated_at: i64,
    pub listing_fee: Option<String>,
    pub whitepaper_url: Option<String>,
    pub website_url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TokenStatus {
    Active,
    Paused,
    Halted,
    Delisted,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum TokenType {
    Erc20,
    Erc721,
    Erc1155,
    Native,
    Spl,
    Trc20,
}

/// Analytics type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Analytics {
    pub total_users: i64,
    pub active_users_24h: i64,
    pub active_users_7d: i64,
    pub active_users_30d: i64,
    pub new_users_24h: i64,
    pub new_users_7d: i64,
    pub new_users_30d: i64,
    pub total_volume_24h: String,
    pub total_volume_7d: String,
    pub total_volume_30d: String,
    pub total_volume_all: String,
    pub revenue_24h: String,
    pub revenue_7d: String,
    pub revenue_30d: String,
    pub revenue_all: String,
    pub growth_percentage: f64,
    pub transactions_24h: i64,
    pub transactions_7d: i64,
    pub transactions_30d: i64,
    pub avg_transaction_value: String,
    pub top_tokens: Vec<TokenVolume>,
    pub top_pairs: Vec<PairVolume>,
    pub top_countries: Vec<CountryStats>,
    pub user_growth: Vec<GrowthDataPoint>,
    pub volume_by_day: Vec<VolumeDataPoint>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenVolume {
    pub symbol: String,
    pub volume: String,
    pub change_24h: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PairVolume {
    pub pair: String,
    pub volume: String,
    pub change_24h: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CountryStats {
    pub country: String,
    pub user_count: i64,
    pub volume: String,
    pub percentage: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GrowthDataPoint {
    pub date: String,
    pub count: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VolumeDataPoint {
    pub date: String,
    pub volume: String,
    pub transactions: i64,
}

/// Transaction type for admin
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminTransaction {
    pub id: String,
    pub user_id: String,
    pub user_email: String,
    pub tx_type: TransactionType,
    pub status: TransactionStatus,
    pub from_address: String,
    pub to_address: String,
    pub amount: String,
    pub token: String,
    pub chain: String,
    pub fee: String,
    pub hash: String,
    pub block_number: Option<u64>,
    pub confirmed_at: Option<i64>,
    pub created_at: i64,
    pub flagged: bool,
    pub flag_reason: Option<String>,
    pub risk_score: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TransactionType {
    Deposit,
    Withdrawal,
    Transfer,
    Swap,
    Mint,
    Burn,
    Staking,
    Unstaking,
    Reward,
    Fee,
    Other,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TransactionStatus {
    Pending,
    Processing,
    Completed,
    Failed,
    Cancelled,
    Flagged,
}

/// System configuration type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemConfig {
    pub id: String,
    pub key: String,
    pub value: String,
    pub value_type: ConfigValueType,
    pub description: String,
    pub category: ConfigCategory,
    pub is_secret: bool,
    pub is_editable: bool,
    pub created_at: i64,
    pub updated_at: i64,
    pub updated_by: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ConfigValueType {
    String,
    Number,
    Boolean,
    Json,
    Array,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ConfigCategory {
    General,
    Security,
    Fees,
    Blockchain,
    Limits,
    Features,
    Integrations,
    Notifications,
}

/// Pagination type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaginatedResponse<T> {
    pub data: Vec<T>,
    pub total: i64,
    pub page: i32,
    pub page_size: i32,
    pub total_pages: i32,
    pub has_next: bool,
    pub has_previous: bool,
}

/// Filter options
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct FilterOptions {
    pub status: Option<String>,
    pub search: Option<String>,
    pub date_from: Option<i64>,
    pub date_to: Option<i64>,
    pub sort_by: Option<String>,
    pub sort_order: Option<String>,
    pub page: Option<i32>,
    pub page_size: Option<i32>,
}

/// Count response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CountResponse {
    pub count: i64,
    pub pending: i64,
    pub approved: i64,
    pub rejected: i64,
}

/// Webhook event type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebhookEvent {
    pub id: String,
    pub event_type: String,
    pub payload: serde_json::Value,
    pub delivered: bool,
    pub delivered_at: Option<i64>,
    pub created_at: i64,
}

/// Scheduled task type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScheduledTask {
    pub id: String,
    pub name: String,
    pub description: String,
    pub cron_expression: String,
    pub task_type: String,
    pub config: serde_json::Value,
    pub status: TaskStatus,
    pub last_run: Option<i64>,
    pub next_run: Option<i64>,
    pub created_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TaskStatus {
    Active,
    Paused,
    Running,
    Failed,
}
