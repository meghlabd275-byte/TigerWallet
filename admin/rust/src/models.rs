//! TigerAdmin Rust - Data Models
//! High-performance admin operations

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

// ============================================================================
// Enums
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum AdminRole {
    SuperAdmin,
    Admin,
    Support,
    Analyst,
    Viewer,
    WhiteLabelAdmin,
    MasterAdmin,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum UserStatus {
    Active,
    Suspended,
    Banned,
    Pending,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum KycStatus {
    None,
    Pending,
    Approved,
    Rejected,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TransactionStatus {
    Pending,
    Confirmed,
    Failed,
    Flagged,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum WithdrawalStatus {
    Pending,
    Approved,
    Rejected,
    Processing,
    Completed,
    Failed,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TokenStatus {
    Active,
    Inactive,
    Suspended,
    Deleted,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum PairStatus {
    Active,
    Halted,
    Suspended,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TicketStatus {
    Open,
    InProgress,
    Resolved,
    Closed,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TicketPriority {
    Low,
    Medium,
    High,
    Urgent,
}

// ============================================================================
// Admin Models
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Admin {
    pub id: Uuid,
    pub username: String,
    pub email: String,
    #[serde(skip_serializing)]
    pub password_hash: String,
    pub role: AdminRole,
    pub permissions: Vec<String>,
    pub is_active: bool,
    pub two_factor_enabled: bool,
    #[serde(skip_serializing)]
    pub two_factor_secret: Option<String>,
    pub ip_whitelist: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub last_login_at: Option<DateTime<Utc>>,
    pub failed_attempts: i32,
    pub locked_until: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Session {
    pub id: Uuid,
    pub admin_id: Uuid,
    #[serde(skip_serializing)]
    pub token_hash: String,
    pub ip_address: String,
    pub user_agent: String,
    pub expires_at: DateTime<Utc>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    pub id: Uuid,
    pub user_id: String,
    pub username: String,
    pub email: String,
    pub phone: Option<String>,
    #[serde(skip_serializing)]
    pub password_hash: Option<String>,
    pub wallet_address: Option<String>,
    pub status: UserStatus,
    pub tier: i32,
    pub email_verified: bool,
    pub phone_verified: bool,
    pub kyc_status: KycStatus,
    pub kyc_level: i32,
    pub white_label_id: Option<Uuid>,
    pub referrer_id: Option<String>,
    pub referral_code: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub last_login_at: Option<DateTime<Utc>>,
    pub failed_login_count: i32,
    pub country: Option<String>,
    pub ip_address: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KycRequest {
    pub id: Uuid,
    pub user_id: Uuid,
    pub level: i32,
    pub document_type: String,
    pub document_number: Option<String>,
    pub document_front: Option<String>,
    pub document_back: Option<String>,
    pub selfie_image: Option<String>,
    pub first_name: String,
    pub last_name: String,
    pub date_of_birth: Option<String>,
    pub country: String,
    pub address: Option<String>,
    pub status: KycStatus,
    pub reject_reason: Option<String>,
    pub reviewed_by: Option<Uuid>,
    pub reviewed_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub id: Uuid,
    pub user_id: Uuid,
    pub tx_type: String,
    pub amount: String,
    pub currency: String,
    pub status: TransactionStatus,
    pub from_address: Option<String>,
    pub to_address: Option<String>,
    pub tx_hash: Option<String>,
    pub fee: Option<String>,
    pub chain_id: i32,
    pub is_flagged: bool,
    pub flag_reason: Option<String>,
    pub timestamp: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Withdrawal {
    pub id: Uuid,
    pub user_id: Uuid,
    pub amount: String,
    pub currency: String,
    pub status: WithdrawalStatus,
    pub address: String,
    pub tx_hash: Option<String>,
    pub approved_by: Option<Uuid>,
    pub processed_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub id: Uuid,
    pub token_id: String,
    pub name: String,
    pub symbol: String,
    pub contract_address: Option<String>,
    pub decimals: i32,
    pub is_active: bool,
    pub is_verified: bool,
    pub total_supply: Option<String>,
    pub chain_id: i32,
    pub logo_url: Option<String>,
    pub website: Option<String>,
    pub description: Option<String>,
    pub status: TokenStatus,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradingPair {
    pub id: Uuid,
    pub base_token_id: Uuid,
    pub quote_token_id: Uuid,
    pub pair_name: String,
    pub price: Option<String>,
    pub volume_24h: Option<String>,
    pub liquidity: Option<String>,
    pub status: PairStatus,
    pub chain_id: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Blockchain {
    pub id: Uuid,
    pub name: String,
    pub symbol: String,
    pub chain_id: i32,
    pub is_evm: bool,
    pub rpc_url: String,
    pub explorer_url: Option<String>,
    pub native_token: String,
    pub decimals: i32,
    pub is_active: bool,
    pub avg_gas_price_gwei: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeeStructure {
    pub id: Uuid,
    pub fee_type: String,
    pub asset: String,
    pub fee_percent: String,
    pub fee_fixed: Option<String>,
    pub min_fee: Option<String>,
    pub max_fee: Option<String>,
    pub tier: Option<String>,
    pub is_active: bool,
    pub chain_id: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WhiteLabel {
    pub id: Uuid,
    pub client_id: String,
    pub company_name: String,
    pub domain: String,
    pub domain_verified: bool,
    pub admin_user_id: Uuid,
    pub status: String,
    pub logo_url: Option<String>,
    pub primary_color: Option<String>,
    pub secondary_color: Option<String>,
    pub theme_mode: Option<String>,
    pub features: Option<serde_json::Value>,
    pub max_users: i32,
    pub max_daily_volume: f64,
    pub platform_fee_percent: f64,
    pub custom_fee_percent: f64,
    pub contact_email: String,
    pub contact_phone: Option<String>,
    pub activated_at: Option<DateTime<Utc>>,
    pub expires_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Ticket {
    pub id: Uuid,
    pub title: String,
    pub description: String,
    pub ticket_type: String,
    pub priority: TicketPriority,
    pub status: TicketStatus,
    pub created_by: Uuid,
    pub assigned_to: Option<Uuid>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub resolved_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TicketMessage {
    pub id: Uuid,
    pub ticket_id: Uuid,
    pub message: String,
    pub is_internal: bool,
    pub created_by: Uuid,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLog {
    pub id: Uuid,
    pub admin_id: Option<Uuid>,
    pub action: String,
    pub resource_type: String,
    pub resource_id: Option<String>,
    pub details: Option<serde_json::Value>,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
    pub success: bool,
    pub error_message: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeatureFlag {
    pub id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub is_enabled: bool,
    pub rollout_percentage: i32,
    pub updated_by: Option<Uuid>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlatformStats {
    pub total_users: i64,
    pub active_users: i64,
    pub suspended_users: i64,
    pub total_volume: f64,
    pub total_transactions: i64,
    pub total_fees: f64,
    pub active_bots: i32,
    pub total_bots: i32,
    pub pending_kyc: i64,
    pub approved_kyc: i64,
    pub rejected_kyc: i64,
}

// ============================================================================
// Request/Response Types
// ============================================================================

#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub email: String,
    pub password: String,
}

#[derive(Debug, Serialize)]
pub struct LoginResponse {
    pub token: String,
    pub refresh_token: String,
    pub admin: Admin,
}

#[derive(Debug, Deserialize)]
pub struct CreateAdminRequest {
    pub username: String,
    pub email: String,
    pub password: String,
    pub role: AdminRole,
    pub permissions: Option<Vec<String>>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateUserRequest {
    pub status: Option<UserStatus>,
    pub kyc_status: Option<KycStatus>,
    pub kyc_level: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct KycApproveRequest {
    pub admin_id: Uuid,
}

#[derive(Debug, Deserialize)]
pub struct KycRejectRequest {
    pub admin_id: Uuid,
    pub reason: String,
}

#[derive(Debug, Deserialize)]
pub struct PaginationParams {
    pub page: Option<i32>,
    pub page_size: Option<i32>,
    pub search: Option<String>,
}

impl Default for PaginationParams {
    fn default() -> Self {
        Self {
            page: Some(1),
            page_size: Some(20),
            search: None,
        }
    }
}
