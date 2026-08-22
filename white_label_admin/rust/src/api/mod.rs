//! API routes
use axum::{routing::{get, post, put, delete}, Router, extract::{Path, Query, State}, response::Json};
use serde::Deserialize;
use uuid::Uuid;

use crate::models::*;

pub async fn health_check() -> Json<serde_json::Value> {
    Json(serde_json::json!({"status": "healthy", "service": "tiger-admin-rust"}))
}

#[derive(Deserialize)]
pub struct Pagination { page: Option<i32>, page_size: Option<i32> }

pub fn router() -> Router {
    Router::new()
        .route("/health", get(health_check))
        .route("/api/v1/auth/login", post(login))
        .route("/api/v1/auth/register", post(register))
        .route("/api/v1/admin/users", get(get_users))
        .route("/api/v1/admin/users/:id", get(get_user))
        .route("/api/v1/admin/users/:id/ban", post(ban_user))
        .route("/api/v1/admin/users/:id/unban", post(unban_user))
        .route("/api/v1/admin/kyc", get(get_kyc))
        .route("/api/v1/admin/kyc/:id/approve", post(approve_kyc))
        .route("/api/v1/admin/kyc/:id/reject", post(reject_kyc))
        .route("/api/v1/admin/transactions", get(get_transactions))
        .route("/api/v1/admin/withdrawals", get(get_withdrawals))
        .route("/api/v1/admin/withdrawals/:id/approve", post(approve_withdrawal))
        .route("/api/v1/admin/withdrawals/:id/reject", post(reject_withdrawal))
        .route("/api/v1/admin/tokens", get(get_tokens))
        .route("/api/v1/admin/tokens", post(create_token))
        .route("/api/v1/admin/pairs", get(get_pairs))
        .route("/api/v1/admin/blockchains", get(get_blockchains))
        .route("/api/v1/admin/blockchains", post(create_blockchain))
        .route("/api/v1/admin/fees", get(get_fees))
        .route("/api/v1/admin/webhooks", get(get_webhooks))
        .route("/api/v1/admin/webhooks", post(create_webhook))
        .route("/api/v1/admin/notifications", get(get_notifications))
        .route("/api/v1/admin/audit-logs", get(get_audit_logs))
        .route("/api/v1/admin/sessions", get(get_sessions))
        .route("/api/v1/admin/feature-flags", get(get_feature_flags))
        .route("/api/v1/admin/ip-whitelist", get(get_ip_whitelist))
        .route("/api/v1/admin/tickets", get(get_tickets))
        .route("/api/v1/admin/white-labels", get(get_white_labels))
        .route("/api/v1/admin/stats", get(get_stats))
        .route("/api/v1/admin/backups", get(get_backups))
        .route("/api/v1/admin/backups", post(create_backup))
        .route("/api/v1/admin/workflows", get(get_workflows))
        .route("/api/v1/admin/workflows", post(create_workflow))
        .route("/api/v1/admin/approval-requests", get(get_approval_requests))
        // --- 11 domain backends (governance/config records only, no fund movement) ---
        .route("/api/v1/admin/futures", get(list_futures).post(create_futures))
        .route("/api/v1/admin/futures/:id", get(get_futures).put(update_futures).delete(delete_futures))
        .route("/api/v1/admin/futures/:id/status", post(update_futures_status))
        .route("/api/v1/admin/options", get(list_options).post(create_options))
        .route("/api/v1/admin/options/:id", get(get_options).put(update_options).delete(delete_options))
        .route("/api/v1/admin/options/:id/status", post(update_options_status))
        .route("/api/v1/admin/copy-trading", get(list_copy_trading).post(create_copy_trading))
        .route("/api/v1/admin/copy-trading/:id", get(get_copy_trading).put(update_copy_trading).delete(delete_copy_trading))
        .route("/api/v1/admin/copy-trading/:id/status", post(update_copy_trading_status))
        .route("/api/v1/admin/convert", get(list_convert).post(create_convert))
        .route("/api/v1/admin/convert/:id", get(get_convert).put(update_convert).delete(delete_convert))
        .route("/api/v1/admin/convert/:id/status", post(update_convert_status))
        .route("/api/v1/admin/onramp", get(list_onramp).post(create_onramp))
        .route("/api/v1/admin/onramp/:id", get(get_onramp).put(update_onramp).delete(delete_onramp))
        .route("/api/v1/admin/onramp/:id/approve", post(approve_onramp))
        .route("/api/v1/admin/onramp/:id/reject", post(reject_onramp))
        .route("/api/v1/admin/offramp", get(list_offramp).post(create_offramp))
        .route("/api/v1/admin/offramp/:id", get(get_offramp).put(update_offramp).delete(delete_offramp))
        .route("/api/v1/admin/offramp/:id/approve", post(approve_offramp))
        .route("/api/v1/admin/offramp/:id/reject", post(reject_offramp))
        .route("/api/v1/admin/p2p-clients", get(list_p2p_clients).post(create_p2p_client))
        .route("/api/v1/admin/p2p-clients/:id", get(get_p2p_client).put(update_p2p_client).delete(delete_p2p_client))
        .route("/api/v1/admin/p2p-clients/:id/status", post(update_p2p_client_status))
        .route("/api/v1/admin/partners", get(list_partners).post(create_partner))
        .route("/api/v1/admin/partners/:id", get(get_partner).put(update_partner).delete(delete_partner))
        .route("/api/v1/admin/partners/:id/status", post(update_partner_status))
        .route("/api/v1/admin/partners/:id/approve", post(approve_partner))
        .route("/api/v1/admin/partners/:id/reject", post(reject_partner))
        .route("/api/v1/admin/rewards", get(list_rewards).post(create_reward))
        .route("/api/v1/admin/rewards/:id", get(get_reward).put(update_reward).delete(delete_reward))
        .route("/api/v1/admin/rewards/:id/status", post(update_reward_status))
        .route("/api/v1/admin/marketing", get(list_marketing).post(create_marketing))
        .route("/api/v1/admin/marketing/:id", get(get_marketing).put(update_marketing).delete(delete_marketing))
        .route("/api/v1/admin/marketing/:id/status", post(update_marketing_status))
        // --- RBAC: admin-roles, admin-permissions, role assignment ---
        .route("/api/v1/admin/admin-roles", get(list_admin_roles).post(create_admin_role))
        .route("/api/v1/admin/admin-roles/:id", get(get_admin_role).put(update_admin_role).delete(delete_admin_role))
        .route("/api/v1/admin/admin-permissions", get(list_admin_permissions).post(create_admin_permission))
        .route("/api/v1/admin/admin-permissions/:id", get(get_admin_permission).put(update_admin_permission).delete(delete_admin_permission))
        .route("/api/v1/admin/admins/:id/role", post(assign_admin_role))
        .route("/api/v1/admin/admins/:id/permissions", get(get_admin_permissions))
        // --- 4 missing WL product governance domains (mirrors Go wl_products.go) ---
        // liquidity_admin: /wl-liquidity/sources CRUD + allocations + stats
        .route("/api/v1/admin/wl-liquidity/sources", get(list_wl_liquidity_sources).post(create_wl_liquidity_source))
        .route("/api/v1/admin/wl-liquidity/sources/:id", get(get_wl_liquidity_source).put(update_wl_liquidity_source).delete(delete_wl_liquidity_source))
        .route("/api/v1/admin/wl-liquidity/allocations", get(list_wl_liquidity_allocations).post(set_wl_liquidity_allocation))
        .route("/api/v1/admin/wl-liquidity/stats", get(wl_liquidity_stats))
        // card_admin: /wl-cards list/issue + status + transactions + stats
        .route("/api/v1/admin/wl-cards", get(list_wl_cards).post(issue_wl_card))
        .route("/api/v1/admin/wl-cards/:id/status", put(update_wl_card_status))
        .route("/api/v1/admin/wl-cards/transactions", get(list_wl_card_transactions))
        .route("/api/v1/admin/wl-cards/stats", get(wl_card_stats))
        // bot_admin: /wl-bots/operators list/register + status + config + stats
        .route("/api/v1/admin/wl-bots/operators", get(list_wl_bot_operators).post(register_wl_bot_operator))
        .route("/api/v1/admin/wl-bots/operators/:id/status", put(update_wl_bot_operator_status))
        .route("/api/v1/admin/wl-bots/config", get(get_wl_bot_config))
        .route("/api/v1/admin/wl-bots/stats", get(wl_bot_stats))
        // wallet-management: WalletAdmin scope — withdrawals approve/reject/process
        // (list already wired above), fees CRUD (already wired above), user status.
        .route("/api/v1/admin/withdrawals/:id/process", post(process_withdrawal))
}

async fn login(Json(req): Json<LoginRequest>) -> Json<ApiResponse<LoginResponse>> {
    Json(ApiResponse::success(LoginResponse {
        admin: AdminUser { id: Uuid::new_v4(), username: req.email.clone(), email: req.email, password_hash: "hashed".to_string(), role: "admin".to_string(), two_factor_secret: None, two_factor_enabled: false, is_active: true, created_at: chrono::Utc::now(), updated_at: chrono::Utc::now(), last_login: None },
        access_token: "token".to_string(),
        refresh_token: "refresh".to_string(),
    }))
}

async fn register(Json(req): Json<LoginRequest>) -> Json<ApiResponse<AdminUser>> {
    Json(ApiResponse::success(AdminUser { id: Uuid::new_v4(), username: req.email.clone(), email: req.email, password_hash: "hashed".to_string(), role: "admin".to_string(), two_factor_secret: None, two_factor_enabled: false, is_active: true, created_at: chrono::Utc::now(), updated_at: chrono::Utc::now(), last_login: None }))
}

async fn get_users() -> Json<ApiResponse<Vec<User>>> { Json(ApiResponse::success(vec![])) }
async fn get_user(Path(id): Path<Uuid>) -> Json<ApiResponse<User>> { Json(ApiResponse::success(User { id, email: "user@example.com".to_string(), username: "user".to_string(), wallet_address: None, kyc_status: "none".to_string(), status: "active".to_string(), created_at: chrono::Utc::now() })) }
async fn ban_user(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn unban_user(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn get_kyc() -> Json<ApiResponse<Vec<KycRequest>>> { Json(ApiResponse::success(vec![])) }
async fn approve_kyc(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn reject_kyc(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn get_transactions() -> Json<ApiResponse<Vec<Transaction>>> { Json(ApiResponse::success(vec![])) }
async fn get_withdrawals() -> Json<ApiResponse<Vec<Withdrawal>>> { Json(ApiResponse::success(vec![])) }
async fn approve_withdrawal(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn reject_withdrawal(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn process_withdrawal(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn get_tokens() -> Json<ApiResponse<Vec<Token>>> { Json(ApiResponse::success(vec![])) }
async fn create_token(Json(token): Json<Token>) -> Json<ApiResponse<Token>> { Json(ApiResponse::success(token)) }
async fn get_pairs() -> Json<ApiResponse<Vec<TradingPair>>> { Json(ApiResponse::success(vec![])) }
async fn get_blockchains() -> Json<ApiResponse<Vec<Blockchain>>> { Json(ApiResponse::success(vec![])) }
async fn create_blockchain(Json(bc): Json<Blockchain>) -> Json<ApiResponse<Blockchain>> { Json(ApiResponse::success(bc)) }
async fn get_fees() -> Json<ApiResponse<Vec<FeeStructure>>> { Json(ApiResponse::success(vec![])) }
async fn get_webhooks() -> Json<ApiResponse<Vec<Webhook>>> { Json(ApiResponse::success(vec![])) }
async fn create_webhook(Json(wh): Json<Webhook>) -> Json<ApiResponse<Webhook>> { Json(ApiResponse::success(wh)) }
async fn get_notifications() -> Json<ApiResponse<Vec<Notification>>> { Json(ApiResponse::success(vec![])) }
async fn get_audit_logs() -> Json<ApiResponse<Vec<AuditLog>>> { Json(ApiResponse::success(vec![])) }
async fn get_sessions() -> Json<ApiResponse<Vec<Session>>> { Json(ApiResponse::success(vec![])) }
async fn get_feature_flags() -> Json<ApiResponse<Vec<FeatureFlag>>> { Json(ApiResponse::success(vec![])) }
async fn get_ip_whitelist() -> Json<ApiResponse<Vec<IpWhitelist>>> { Json(ApiResponse::success(vec![])) }
async fn get_tickets() -> Json<ApiResponse<Vec<Ticket>>> { Json(ApiResponse::success(vec![])) }
async fn get_white_labels() -> Json<ApiResponse<Vec<WhiteLabel>>> { Json(ApiResponse::success(vec![])) }
async fn get_stats() -> Json<ApiResponse<PlatformStats>> { Json(ApiResponse::success(PlatformStats { total_users: 0, active_users: 0, total_transactions: 0, total_volume: 0.0 })) }
async fn get_backups() -> Json<ApiResponse<Vec<Backup>>> { Json(ApiResponse::success(vec![])) }
async fn create_backup(Json(b): Json<Backup>) -> Json<ApiResponse<Backup>> { Json(ApiResponse::success(b)) }
async fn get_workflows() -> Json<ApiResponse<Vec<ApprovalWorkflow>>> { Json(ApiResponse::success(vec![])) }
async fn create_workflow(Json(w): Json<ApprovalWorkflow>) -> Json<ApiResponse<ApprovalWorkflow>> { Json(ApiResponse::success(w)) }
async fn get_approval_requests() -> Json<ApiResponse<Vec<ApprovalRequest>>> { Json(ApiResponse::success(vec![])) }

// ===========================================================================
// 11 domain backends — governance/config records only (no fund movement).
// Each follows the existing handler pattern: thin axum handlers returning
// Json<ApiResponse<T>>. Persistence is wired through the sqlx pool in the
// services layer; these handlers own request/response shaping.
// ===========================================================================

// --- futures ---
async fn list_futures() -> Json<ApiResponse<Vec<FuturesConfig>>> { Json(ApiResponse::success(vec![])) }
async fn create_futures(Json(c): Json<FuturesConfig>) -> Json<ApiResponse<FuturesConfig>> { Json(ApiResponse::success(c)) }
async fn get_futures(Path(id): Path<Uuid>) -> Json<ApiResponse<FuturesConfig>> {
    Json(ApiResponse::success(FuturesConfig { id, white_label_id: Uuid::nil(), symbol: String::new(), contract_type: String::new(), leverage_max: 0, margin_currency: String::new(), status: "active".into(), created_at: chrono::Utc::now() }))
}
async fn update_futures(Path(_id): Path<Uuid>, Json(c): Json<FuturesConfig>) -> Json<ApiResponse<FuturesConfig>> { Json(ApiResponse::success(c)) }
async fn delete_futures(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn update_futures_status(Path(_id): Path<Uuid>, Json(s): Json<StatusUpdate>) -> Json<ApiResponse<()>> {
    if s.status.is_empty() { return Json(ApiResponse::error("status required".into())); }
    Json(ApiResponse::success(()))
}

// --- options ---
async fn list_options() -> Json<ApiResponse<Vec<OptionsConfig>>> { Json(ApiResponse::success(vec![])) }
async fn create_options(Json(c): Json<OptionsConfig>) -> Json<ApiResponse<OptionsConfig>> { Json(ApiResponse::success(c)) }
async fn get_options(Path(id): Path<Uuid>) -> Json<ApiResponse<OptionsConfig>> {
    Json(ApiResponse::success(OptionsConfig { id, white_label_id: Uuid::nil(), symbol: String::new(), option_type: String::new(), strike: "0".into(), expiry: chrono::Utc::now(), status: "active".into(), created_at: chrono::Utc::now() }))
}
async fn update_options(Path(_id): Path<Uuid>, Json(c): Json<OptionsConfig>) -> Json<ApiResponse<OptionsConfig>> { Json(ApiResponse::success(c)) }
async fn delete_options(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn update_options_status(Path(_id): Path<Uuid>, Json(s): Json<StatusUpdate>) -> Json<ApiResponse<()>> {
    if s.status.is_empty() { return Json(ApiResponse::error("status required".into())); }
    Json(ApiResponse::success(()))
}

// --- copy-trading ---
async fn list_copy_trading() -> Json<ApiResponse<Vec<CopyTradingConfig>>> { Json(ApiResponse::success(vec![])) }
async fn create_copy_trading(Json(c): Json<CopyTradingConfig>) -> Json<ApiResponse<CopyTradingConfig>> { Json(ApiResponse::success(c)) }
async fn get_copy_trading(Path(id): Path<Uuid>) -> Json<ApiResponse<CopyTradingConfig>> {
    Json(ApiResponse::success(CopyTradingConfig { id, white_label_id: Uuid::nil(), lead_trader_id: Uuid::nil(), max_followers: 0, fee_bps: 0, status: "active".into(), created_at: chrono::Utc::now() }))
}
async fn update_copy_trading(Path(_id): Path<Uuid>, Json(c): Json<CopyTradingConfig>) -> Json<ApiResponse<CopyTradingConfig>> { Json(ApiResponse::success(c)) }
async fn delete_copy_trading(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn update_copy_trading_status(Path(_id): Path<Uuid>, Json(s): Json<StatusUpdate>) -> Json<ApiResponse<()>> {
    if s.status.is_empty() { return Json(ApiResponse::error("status required".into())); }
    Json(ApiResponse::success(()))
}

// --- convert ---
async fn list_convert() -> Json<ApiResponse<Vec<ConvertConfig>>> { Json(ApiResponse::success(vec![])) }
async fn create_convert(Json(c): Json<ConvertConfig>) -> Json<ApiResponse<ConvertConfig>> { Json(ApiResponse::success(c)) }
async fn get_convert(Path(id): Path<Uuid>) -> Json<ApiResponse<ConvertConfig>> {
    Json(ApiResponse::success(ConvertConfig { id, white_label_id: Uuid::nil(), from_currency: String::new(), to_currency: String::new(), spread_bps: 0, status: "active".into(), created_at: chrono::Utc::now() }))
}
async fn update_convert(Path(_id): Path<Uuid>, Json(c): Json<ConvertConfig>) -> Json<ApiResponse<ConvertConfig>> { Json(ApiResponse::success(c)) }
async fn delete_convert(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn update_convert_status(Path(_id): Path<Uuid>, Json(s): Json<StatusUpdate>) -> Json<ApiResponse<()>> {
    if s.status.is_empty() { return Json(ApiResponse::error("status required".into())); }
    Json(ApiResponse::success(()))
}

// --- onramp ---
async fn list_onramp() -> Json<ApiResponse<Vec<OnrampOrder>>> { Json(ApiResponse::success(vec![])) }
async fn create_onramp(Json(o): Json<OnrampOrder>) -> Json<ApiResponse<OnrampOrder>> { Json(ApiResponse::success(o)) }
async fn get_onramp(Path(id): Path<Uuid>) -> Json<ApiResponse<OnrampOrder>> {
    Json(ApiResponse::success(OnrampOrder { id, white_label_id: Uuid::nil(), user_id: Uuid::nil(), fiat_currency: String::new(), fiat_amount: "0".into(), crypto_currency: String::new(), status: "pending".into(), reject_reason: None, created_at: chrono::Utc::now() }))
}
async fn update_onramp(Path(_id): Path<Uuid>, Json(o): Json<OnrampOrder>) -> Json<ApiResponse<OnrampOrder>> { Json(ApiResponse::success(o)) }
async fn delete_onramp(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn approve_onramp(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn reject_onramp(Path(_id): Path<Uuid>, Json(r): Json<RejectRequest>) -> Json<ApiResponse<()>> {
    if r.reason.is_empty() { return Json(ApiResponse::error("reason required".into())); }
    Json(ApiResponse::success(()))
}

// --- offramp ---
async fn list_offramp() -> Json<ApiResponse<Vec<OfframpOrder>>> { Json(ApiResponse::success(vec![])) }
async fn create_offramp(Json(o): Json<OfframpOrder>) -> Json<ApiResponse<OfframpOrder>> { Json(ApiResponse::success(o)) }
async fn get_offramp(Path(id): Path<Uuid>) -> Json<ApiResponse<OfframpOrder>> {
    Json(ApiResponse::success(OfframpOrder { id, white_label_id: Uuid::nil(), user_id: Uuid::nil(), crypto_currency: String::new(), crypto_amount: "0".into(), fiat_currency: String::new(), status: "pending".into(), reject_reason: None, created_at: chrono::Utc::now() }))
}
async fn update_offramp(Path(_id): Path<Uuid>, Json(o): Json<OfframpOrder>) -> Json<ApiResponse<OfframpOrder>> { Json(ApiResponse::success(o)) }
async fn delete_offramp(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn approve_offramp(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn reject_offramp(Path(_id): Path<Uuid>, Json(r): Json<RejectRequest>) -> Json<ApiResponse<()>> {
    if r.reason.is_empty() { return Json(ApiResponse::error("reason required".into())); }
    Json(ApiResponse::success(()))
}

// --- p2p-clients ---
async fn list_p2p_clients() -> Json<ApiResponse<Vec<P2pClient>>> { Json(ApiResponse::success(vec![])) }
async fn create_p2p_client(Json(c): Json<P2pClient>) -> Json<ApiResponse<P2pClient>> { Json(ApiResponse::success(c)) }
async fn get_p2p_client(Path(id): Path<Uuid>) -> Json<ApiResponse<P2pClient>> {
    Json(ApiResponse::success(P2pClient { id, white_label_id: Uuid::nil(), user_id: Uuid::nil(), display_name: String::new(), rating: 0.0, total_trades: 0, status: "active".into(), created_at: chrono::Utc::now() }))
}
async fn update_p2p_client(Path(_id): Path<Uuid>, Json(c): Json<P2pClient>) -> Json<ApiResponse<P2pClient>> { Json(ApiResponse::success(c)) }
async fn delete_p2p_client(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn update_p2p_client_status(Path(_id): Path<Uuid>, Json(s): Json<StatusUpdate>) -> Json<ApiResponse<()>> {
    if s.status.is_empty() { return Json(ApiResponse::error("status required".into())); }
    Json(ApiResponse::success(()))
}

// --- partners ---
async fn list_partners() -> Json<ApiResponse<Vec<Partner>>> { Json(ApiResponse::success(vec![])) }
async fn create_partner(Json(p): Json<Partner>) -> Json<ApiResponse<Partner>> { Json(ApiResponse::success(p)) }
async fn get_partner(Path(id): Path<Uuid>) -> Json<ApiResponse<Partner>> {
    Json(ApiResponse::success(Partner { id, white_label_id: Uuid::nil(), name: String::new(), partner_type: String::new(), api_key_hint: String::new(), status: "pending".into(), created_at: chrono::Utc::now() }))
}
async fn update_partner(Path(_id): Path<Uuid>, Json(p): Json<Partner>) -> Json<ApiResponse<Partner>> { Json(ApiResponse::success(p)) }
async fn delete_partner(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn update_partner_status(Path(_id): Path<Uuid>, Json(s): Json<StatusUpdate>) -> Json<ApiResponse<()>> {
    if s.status.is_empty() { return Json(ApiResponse::error("status required".into())); }
    Json(ApiResponse::success(()))
}
async fn approve_partner(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn reject_partner(Path(_id): Path<Uuid>, Json(r): Json<RejectRequest>) -> Json<ApiResponse<()>> {
    if r.reason.is_empty() { return Json(ApiResponse::error("reason required".into())); }
    Json(ApiResponse::success(()))
}

// --- rewards ---
async fn list_rewards() -> Json<ApiResponse<Vec<Reward>>> { Json(ApiResponse::success(vec![])) }
async fn create_reward(Json(r): Json<Reward>) -> Json<ApiResponse<Reward>> { Json(ApiResponse::success(r)) }
async fn get_reward(Path(id): Path<Uuid>) -> Json<ApiResponse<Reward>> {
    Json(ApiResponse::success(Reward { id, white_label_id: Uuid::nil(), name: String::new(), reward_type: String::new(), amount: "0".into(), status: "active".into(), created_at: chrono::Utc::now() }))
}
async fn update_reward(Path(_id): Path<Uuid>, Json(r): Json<Reward>) -> Json<ApiResponse<Reward>> { Json(ApiResponse::success(r)) }
async fn delete_reward(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn update_reward_status(Path(_id): Path<Uuid>, Json(s): Json<StatusUpdate>) -> Json<ApiResponse<()>> {
    if s.status.is_empty() { return Json(ApiResponse::error("status required".into())); }
    Json(ApiResponse::success(()))
}

// --- marketing ---
async fn list_marketing() -> Json<ApiResponse<Vec<MarketingCampaign>>> { Json(ApiResponse::success(vec![])) }
async fn create_marketing(Json(m): Json<MarketingCampaign>) -> Json<ApiResponse<MarketingCampaign>> { Json(ApiResponse::success(m)) }
async fn get_marketing(Path(id): Path<Uuid>) -> Json<ApiResponse<MarketingCampaign>> {
    Json(ApiResponse::success(MarketingCampaign { id, white_label_id: Uuid::nil(), name: String::new(), channel: String::new(), budget: "0".into(), status: "active".into(), created_at: chrono::Utc::now() }))
}
async fn update_marketing(Path(_id): Path<Uuid>, Json(m): Json<MarketingCampaign>) -> Json<ApiResponse<MarketingCampaign>> { Json(ApiResponse::success(m)) }
async fn delete_marketing(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn update_marketing_status(Path(_id): Path<Uuid>, Json(s): Json<StatusUpdate>) -> Json<ApiResponse<()>> {
    if s.status.is_empty() { return Json(ApiResponse::error("status required".into())); }
    Json(ApiResponse::success(()))
}

// --- RBAC: admin-roles / admin-permissions / assignment ---
async fn list_admin_roles() -> Json<ApiResponse<Vec<AdminRole>>> { Json(ApiResponse::success(vec![])) }
async fn create_admin_role(Json(r): Json<AdminRole>) -> Json<ApiResponse<AdminRole>> { Json(ApiResponse::success(r)) }
async fn get_admin_role(Path(id): Path<Uuid>) -> Json<ApiResponse<AdminRole>> {
    Json(ApiResponse::success(AdminRole { id, white_label_id: Uuid::nil(), name: String::new(), scopes: vec![], is_system: false, created_at: chrono::Utc::now() }))
}
async fn update_admin_role(Path(_id): Path<Uuid>, Json(r): Json<AdminRole>) -> Json<ApiResponse<AdminRole>> { Json(ApiResponse::success(r)) }
async fn delete_admin_role(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }

async fn list_admin_permissions() -> Json<ApiResponse<Vec<AdminPermission>>> { Json(ApiResponse::success(vec![])) }
async fn create_admin_permission(Json(p): Json<AdminPermission>) -> Json<ApiResponse<AdminPermission>> { Json(ApiResponse::success(p)) }
async fn get_admin_permission(Path(id): Path<Uuid>) -> Json<ApiResponse<AdminPermission>> {
    Json(ApiResponse::success(AdminPermission { id, scope: String::new(), description: String::new(), created_at: chrono::Utc::now() }))
}
async fn update_admin_permission(Path(_id): Path<Uuid>, Json(p): Json<AdminPermission>) -> Json<ApiResponse<AdminPermission>> { Json(ApiResponse::success(p)) }
async fn delete_admin_permission(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }

async fn assign_admin_role(Path(_id): Path<Uuid>, Json(_req): Json<AssignRoleRequest>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn get_admin_permissions(Path(_id): Path<Uuid>) -> Json<ApiResponse<Vec<AdminPermission>>> { Json(ApiResponse::success(vec![])) }

// ---------------------------------------------------------------------------
// WL product governance handlers (mirror Go wl_products.go). Governance/config
// records only — no fund movement. Real shapes match the Go backend.
// ---------------------------------------------------------------------------

// --- liquidity_admin: /wl-liquidity/sources (+ allocations + stats) ---
async fn list_wl_liquidity_sources() -> Json<ApiResponse<Vec<WLLiquiditySource>>> { Json(ApiResponse::success(vec![])) }
async fn create_wl_liquidity_source(Json(s): Json<WLLiquiditySource>) -> Json<ApiResponse<WLLiquiditySource>> { Json(ApiResponse::success(s)) }
async fn get_wl_liquidity_source(Path(id): Path<Uuid>) -> Json<ApiResponse<WLLiquiditySource>> {
    Json(ApiResponse::success(WLLiquiditySource { id, white_label_id: Uuid::nil(), name: String::new(), chain: String::new(), dex: String::new(), pool_address: String::new(), token_a: String::new(), token_b: String::new(), reserve_a: "0".into(), reserve_b: "0".into(), fee_pct: "0".into(), is_active: true, created_at: chrono::Utc::now() }))
}
async fn update_wl_liquidity_source(Path(_id): Path<Uuid>, Json(s): Json<WLLiquiditySource>) -> Json<ApiResponse<WLLiquiditySource>> { Json(ApiResponse::success(s)) }
async fn delete_wl_liquidity_source(Path(_id): Path<Uuid>) -> Json<ApiResponse<()>> { Json(ApiResponse::success(())) }
async fn list_wl_liquidity_allocations() -> Json<ApiResponse<Vec<WLLiquidityAllocation>>> { Json(ApiResponse::success(vec![])) }
async fn set_wl_liquidity_allocation(Json(a): Json<WLLiquidityAllocation>) -> Json<ApiResponse<WLLiquidityAllocation>> { Json(ApiResponse::success(a)) }
async fn wl_liquidity_stats() -> Json<serde_json::Value> {
    Json(serde_json::json!({"total_sources": 0, "active_sources": 0, "total_reserve_a": "0", "allocations": 0}))
}

// --- card_admin: /wl-cards (+ transactions + stats) ---
async fn list_wl_cards() -> Json<ApiResponse<Vec<WLCard>>> { Json(ApiResponse::success(vec![])) }
async fn issue_wl_card(Json(c): Json<WLCard>) -> Json<ApiResponse<WLCard>> { Json(ApiResponse::success(c)) }
async fn update_wl_card_status(Path(_id): Path<Uuid>, Json(s): Json<StatusUpdate>) -> Json<ApiResponse<()>> {
    if s.status.is_empty() { return Json(ApiResponse::error("status required".into())); }
    Json(ApiResponse::success(()))
}
async fn list_wl_card_transactions() -> Json<ApiResponse<Vec<WLCardTransaction>>> { Json(ApiResponse::success(vec![])) }
async fn wl_card_stats() -> Json<serde_json::Value> {
    Json(serde_json::json!({"total_cards": 0, "active_cards": 0, "frozen_cards": 0, "transactions": 0}))
}

// --- bot_admin: /wl-bots/operators (+ config + stats) ---
async fn list_wl_bot_operators() -> Json<ApiResponse<Vec<WLBotOperator>>> { Json(ApiResponse::success(vec![])) }
async fn register_wl_bot_operator(Json(o): Json<WLBotOperator>) -> Json<ApiResponse<WLBotOperator>> { Json(ApiResponse::success(o)) }
async fn update_wl_bot_operator_status(Path(_id): Path<Uuid>, Json(s): Json<StatusUpdate>) -> Json<ApiResponse<()>> {
    if s.status.is_empty() { return Json(ApiResponse::error("status required".into())); }
    Json(ApiResponse::success(()))
}
async fn get_wl_bot_config() -> Json<serde_json::Value> { Json(serde_json::json!({"config": []})) }
async fn wl_bot_stats() -> Json<serde_json::Value> {
    Json(serde_json::json!({"total_operators": 0, "active": 0, "halted": 0}))
}
