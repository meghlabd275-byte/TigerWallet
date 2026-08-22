//! Data models for TigerWallet Admin
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminUser {
    pub id: Uuid,
    pub username: String,
    pub email: String,
    pub password_hash: String,
    pub role: String,
    pub two_factor_secret: Option<String>,
    pub two_factor_enabled: bool,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub last_login: Option<DateTime<Utc>>,
}

#[derive(Debug, Serialize, Deserialize)]
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    pub id: Uuid,
    pub email: String,
    pub username: String,
    pub wallet_address: Option<String>,
    pub kyc_status: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub id: Uuid,
    pub user_id: Uuid,
    pub tx_type: String,
    pub amount: String,
    pub currency: String,
    pub status: String,
    pub timestamp: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Withdrawal {
    pub id: Uuid,
    pub user_id: Uuid,
    pub amount: String,
    pub currency: String,
    pub status: String,
    pub address: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KycRequest {
    pub id: Uuid,
    pub user_id: Uuid,
    pub doc_type: String,
    pub status: String,
    pub submitted_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub id: Uuid,
    pub symbol: String,
    pub name: String,
    pub is_active: bool,
    pub is_verified: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradingPair {
    pub id: Uuid,
    pub pair_name: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Blockchain {
    pub id: Uuid,
    pub name: String,
    pub symbol: String,
    pub chain_id: i32,
    pub is_evm: bool,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeeStructure {
    pub id: Uuid,
    pub fee_type: String,
    pub fee_percent: Option<String>,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Webhook {
    pub id: Uuid,
    pub name: String,
    pub url: String,
    pub events: Vec<String>,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Notification {
    pub id: Uuid,
    pub admin_id: Uuid,
    pub title: String,
    pub message: String,
    pub notification_type: String,
    pub is_read: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLog {
    pub id: Uuid,
    pub admin_id: Option<Uuid>,
    pub action: String,
    pub resource_type: String,
    pub resource_id: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Session {
    pub id: Uuid,
    pub admin_id: Uuid,
    pub ip_address: Option<String>,
    pub expires_at: DateTime<Utc>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeatureFlag {
    pub id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub is_enabled: bool,
    pub rollout_percentage: i32,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IpWhitelist {
    pub id: Uuid,
    pub ip_address: String,
    pub description: Option<String>,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WhiteLabel {
    pub id: Uuid,
    pub name: String,
    pub domain: String,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Backup {
    pub id: Uuid,
    pub backup_type: String,
    pub file_path: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalWorkflow {
    pub id: Uuid,
    pub name: String,
    pub workflow_type: String,
    pub threshold_amount: Option<String>,
    pub required_approvals: i32,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalRequest {
    pub id: Uuid,
    pub workflow_id: Option<Uuid>,
    pub request_type: String,
    pub resource_id: Option<String>,
    pub requester_id: Uuid,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Serialize)]
pub struct PlatformStats {
    pub total_users: i64,
    pub active_users: i64,
    pub total_transactions: i64,
    pub total_volume: f64,
}

#[derive(Debug, Serialize)]
pub struct ApiResponse<T> {
    pub success: bool,
    pub data: Option<T>,
    pub error: Option<String>,
}

impl<T> ApiResponse<T> {
    pub fn success(data: T) -> Self {
        Self { success: true, data: Some(data), error: None }
    }
    pub fn error(message: String) -> Self {
        Self { success: false, data: None, error: Some(message) }
    }
}

#[derive(Debug, Serialize)]
pub struct PaginatedResponse<T> {
    pub items: Vec<T>,
    pub total: i64,
    pub page: i32,
    pub page_size: i32,
}

// ---------------------------------------------------------------------------
// Domain governance models (white-label admin). These are governance/config
// records only — no fund movement. All carry a `white_label_id` for tenant
// isolation and a `status` field managed via dedicated approve/reject or
// status endpoints.
// ---------------------------------------------------------------------------

/// Generic status-update body for /:id/status endpoints.
#[derive(Debug, Deserialize)]
pub struct StatusUpdate {
    pub status: String,
}

/// Generic rejection body for /:id/reject endpoints (onramp also carries reason).
#[derive(Debug, Deserialize)]
pub struct RejectRequest {
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FuturesConfig {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub symbol: String,
    pub contract_type: String,
    pub leverage_max: i32,
    pub margin_currency: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OptionsConfig {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub symbol: String,
    pub option_type: String,
    pub strike: String,
    pub expiry: DateTime<Utc>,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CopyTradingConfig {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub lead_trader_id: Uuid,
    pub max_followers: i32,
    pub fee_bps: i32,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConvertConfig {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub from_currency: String,
    pub to_currency: String,
    pub spread_bps: i32,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OnrampOrder {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub user_id: Uuid,
    pub fiat_currency: String,
    pub fiat_amount: String,
    pub crypto_currency: String,
    pub status: String,
    pub reject_reason: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OfframpOrder {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub user_id: Uuid,
    pub crypto_currency: String,
    pub crypto_amount: String,
    pub fiat_currency: String,
    pub status: String,
    pub reject_reason: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct P2pClient {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub user_id: Uuid,
    pub display_name: String,
    pub rating: f64,
    pub total_trades: i64,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Partner {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub name: String,
    pub partner_type: String,
    pub api_key_hint: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Reward {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub name: String,
    pub reward_type: String,
    pub amount: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketingCampaign {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub name: String,
    pub channel: String,
    pub budget: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

// --- RBAC ---

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminRole {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub name: String,
    pub scopes: Vec<String>,
    pub is_system: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminPermission {
    pub id: Uuid,
    pub scope: String,
    pub description: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Deserialize)]
pub struct AssignRoleRequest {
    pub role_id: Uuid,
}

// ---------------------------------------------------------------------------
// WL product governance models (mirrors the Go backend wl_products.go shapes:
// /wl-liquidity/sources, /wl-cards, /wl-bots/operators). Governance records
// only — no fund movement. The wallet-management domain reuses Withdrawal +
// FeeStructure (already defined above) plus user-status updates.
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WLLiquiditySource {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub name: String,
    pub chain: String,
    pub dex: String,
    pub pool_address: String,
    pub token_a: String,
    pub token_b: String,
    pub reserve_a: String,
    pub reserve_b: String,
    pub fee_pct: String,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WLLiquidityAllocation {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub name: String,
    pub fee_share_pct: String,
    pub destination: String,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WLCard {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub user_id: Option<Uuid>,
    pub holder_name: String,
    pub status: String,
    pub balance: String,
    pub currency: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WLCardTransaction {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub card_id: Uuid,
    pub amount: String,
    pub merchant: String,
    pub category: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WLBotOperator {
    pub id: Uuid,
    pub white_label_id: Uuid,
    pub name: String,
    pub strategy: String,
    pub status: String,
    pub config: serde_json::Value,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

