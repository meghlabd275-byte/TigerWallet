//! Data models for TigerWallet Admin Panel
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

// ==================== Admin User ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct AdminUser {
    pub id: Uuid,
    pub username: String,
    pub email: String,
    #[serde(skip_serializing)]
    pub password_hash: String,
    pub role: String,
    pub two_factor_secret: Option<String>,
    pub two_factor_enabled: bool,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub last_login: Option<DateTime<Utc>>,
}

#[derive(Debug, Deserialize)]
pub struct CreateAdminRequest {
    pub username: String,
    pub email: String,
    pub password: String,
    pub role: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub email: String,
    pub password: String,
}

#[derive(Debug, Serialize)]
pub struct LoginResponse {
    pub admin: AdminUser,
    pub access_token: String,
    pub refresh_token: String,
}

// ==================== User ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct User {
    pub id: Uuid,
    pub email: String,
    pub username: String,
    pub wallet_address: Option<String>,
    pub kyc_status: String,
    pub status: String,
    pub two_factor_enabled: bool,
    pub ip_address: Option<String>,
    pub country: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub last_login: Option<DateTime<Utc>>,
}

// ==================== Transaction ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Transaction {
    pub id: Uuid,
    pub user_id: Uuid,
    #[serde(rename = "type")]
    pub tx_type: String,
    pub amount: String,
    pub currency: String,
    pub status: String,
    pub from_address: Option<String>,
    pub to_address: Option<String>,
    pub tx_hash: Option<String>,
    pub fee: Option<String>,
    pub chain_id: Option<i32>,
    pub timestamp: DateTime<Utc>,
}

// ==================== Withdrawal ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Withdrawal {
    pub id: Uuid,
    pub user_id: Uuid,
    pub amount: String,
    pub currency: String,
    pub status: String,
    pub address: String,
    pub tx_hash: Option<String>,
    pub approved_by: Option<Uuid>,
    pub processed_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

// ==================== KYC Request ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct KycRequest {
    pub id: Uuid,
    pub user_id: Uuid,
    pub doc_type: String,
    pub status: String,
    pub document_url: Option<String>,
    pub submitted_at: DateTime<Utc>,
    pub reviewed_at: Option<DateTime<Utc>>,
    pub reviewed_by: Option<Uuid>,
    pub reject_reason: Option<String>,
}

// ==================== Token ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Token {
    pub id: Uuid,
    pub symbol: String,
    pub name: String,
    pub contract_address: Option<String>,
    pub decimals: i32,
    pub is_active: bool,
    pub is_verified: bool,
    pub total_supply: Option<String>,
    pub chain_id: Option<i32>,
    pub created_at: DateTime<Utc>,
}

// ==================== Trading Pair ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct TradingPair {
    pub id: Uuid,
    pub base_token_id: Uuid,
    pub quote_token_id: Uuid,
    pub pair_name: String,
    pub price: Option<String>,
    pub volume_24h: Option<String>,
    pub liquidity: Option<String>,
    pub status: String,
    pub chain_id: Option<i32>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

// ==================== Blockchain ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Blockchain {
    pub id: Uuid,
    pub name: String,
    pub symbol: String,
    pub chain_id: i32,
    pub is_evm: bool,
    pub rpc_url: Option<String>,
    pub explorer_url: Option<String>,
    pub native_token: Option<String>,
    pub decimals: i32,
    pub is_active: bool,
    pub avg_gas_price_gwei: Option<String>,
    pub created_at: DateTime<Utc>,
}

// ==================== Fee Structure ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct FeeStructure {
    pub id: Uuid,
    pub fee_type: String,
    pub asset: Option<String>,
    pub fee_percent: Option<String>,
    pub fee_fixed: Option<String>,
    pub min_fee: Option<String>,
    pub max_fee: Option<String>,
    pub tier: Option<String>,
    pub is_active: bool,
    pub chain_id: Option<i32>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

// ==================== Webhook ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Webhook {
    pub id: Uuid,
    pub name: String,
    pub url: String,
    #[serde(skip_serializing)]
    pub secret: Option<String>,
    pub events: Vec<String>,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
    pub created_by: Uuid,
}

// ==================== Notification ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Notification {
    pub id: Uuid,
    pub admin_id: Uuid,
    pub title: String,
    pub message: String,
    pub notification_type: String,
    pub is_read: bool,
    pub created_at: DateTime<Utc>,
}

// ==================== Audit Log ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct AuditLog {
    pub id: Uuid,
    pub admin_id: Option<Uuid>,
    pub action: String,
    pub resource_type: String,
    pub resource_id: Option<String>,
    pub details: Option<serde_json::Value>,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
    pub created_at: DateTime<Utc>,
}

// ==================== Session ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Session {
    pub id: Uuid,
    pub admin_id: Uuid,
    pub token_hash: String,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
    pub expires_at: DateTime<Utc>,
    pub created_at: DateTime<Utc>,
}

// ==================== Feature Flag ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct FeatureFlag {
    pub id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub is_enabled: bool,
    pub rollout_percentage: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub updated_by: Option<Uuid>,
}

// ==================== IP Whitelist ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct IpWhitelist {
    pub id: Uuid,
    pub ip_address: String,
    pub description: Option<String>,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
    pub created_by: Uuid,
}

// ==================== Ticket ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Ticket {
    pub id: Uuid,
    pub title: String,
    pub description: Option<String>,
    pub ticket_type: String,
    pub priority: String,
    pub status: String,
    pub created_by: Uuid,
    pub assigned_to: Option<Uuid>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub resolved_at: Option<DateTime<Utc>>,
}

// ==================== Ticket Message ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct TicketMessage {
    pub id: Uuid,
    pub ticket_id: Uuid,
    pub message: String,
    pub is_internal: bool,
    pub created_by: Uuid,
    pub created_at: DateTime<Utc>,
}

// ==================== White Label ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct WhiteLabel {
    pub id: Uuid,
    pub name: String,
    pub domain: String,
    pub logo_url: Option<String>,
    pub primary_color: Option<String>,
    pub secondary_color: Option<String>,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
}

// ==================== Report ====================

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Report {
    pub id: Uuid,
    pub report_type: String,
    pub title: String,
    pub filters: Option<serde_json::Value>,
    pub file_path: Option<String>,
    pub status: String,
    pub generated_by: Uuid,
    pub created_at: DateTime<Utc>,
    pub completed_at: Option<DateTime<Utc>>,
}

// ==================== Platform Stats ====================

#[derive(Debug, Serialize)]
pub struct PlatformStats {
    pub total_users: i64,
    pub active_users: i64,
    pub total_transactions: i64,
    pub total_volume: f64,
    pub total_fees: f64,
    pub active_bots: i32,
    pub total_bots: i32,
}

// ==================== API Response Types ====================

#[derive(Debug, Serialize)]
pub struct ApiResponse<T> {
    pub success: bool,
    pub data: Option<T>,
    pub error: Option<String>,
}

impl<T> ApiResponse<T> {
    pub fn success(data: T) -> Self {
        Self {
            success: true,
            data: Some(data),
            error: None,
        }
    }

    pub fn error(message: String) -> Self {
        Self {
            success: false,
            data: None,
            error: Some(message),
        }
    }
}

#[derive(Debug, Serialize)]
pub struct PaginatedResponse<T> {
    pub items: Vec<T>,
    pub total: i64,
    pub page: i32,
    pub page_size: i32,
    pub total_pages: i32,
}
