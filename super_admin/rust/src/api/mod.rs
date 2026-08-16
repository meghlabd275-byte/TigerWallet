//! API routes
use axum::{routing::{get, post, put, delete}, Router, extract::{Path, Query, State}, response::Json};
use serde::Deserialize;
use uuid::Uuid;

use crate::domain::{self, AppState, proxy_domain};
use crate::models::*;

pub async fn health_check() -> Json<serde_json::Value> {
    Json(serde_json::json!({"status": "healthy", "service": "tiger-admin-rust"}))
}

#[derive(Deserialize)]
pub struct Pagination { page: Option<i32>, page_size: Option<i32> }

pub fn router() -> Router {
    let state = AppState::new();

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
        // Domain manifest (lists the 12 governance domains).
        .route("/api/v1/admin/domains", get(domain::list_domains))
        // ---- 12 domain admin screens (real calls to Go backend on :8082) ----
        // Each domain registers explicit CRUD routes plus its governance
        // sub-actions (status / approve / reject / RBAC). All forward the JWT
        // bearer token to the upstream Go super-admin backend.
        .domain_routes("futures", &["status"])
        .domain_routes("options", &["status"])
        .domain_routes("copy-trading", &["status"])
        .domain_routes("convert", &["status"])
        .domain_routes("onramp", &["approve", "reject"])
        .domain_routes("offramp", &["approve", "reject"])
        .domain_routes("p2p-clients", &["status"])
        .domain_routes("partners", &["status", "approve", "reject"])
        .domain_routes("rewards", &["status"])
        .domain_routes("marketing", &["status"])
        // admin-roles RBAC: roles + permissions CRUD + assign + effective.
        .domain_routes("admin-roles", &[])
        .domain_routes("admin-permissions", &[])
        .route("/api/v1/admin/admins/:id/roles", post(proxy_domain))
        .route("/api/v1/admin/admins/:id/roles/:role_id", delete(proxy_domain))
        .route("/api/v1/admin/admins/:id/permissions", get(proxy_domain))
        // wl-control: five white-label sub-resources, each CRUD + status.
        .domain_routes("wl-clients", &["status"])
        .domain_routes("wl-master-wallets", &["status"])
        .domain_routes("wl-user-wallets", &["status"])
        .domain_routes("wl-bots", &["status"])
        .domain_routes("wl-bots-clients", &["status"])
        // Fallback catch-all for any deeper admin path not matched above.
        .route("/api/v1/admin/:domain", get(proxy_domain).post(proxy_domain).put(proxy_domain).delete(proxy_domain))
        .route("/api/v1/admin/:domain/:id", get(proxy_domain).put(proxy_domain).delete(proxy_domain))
        .route("/api/v1/admin/:domain/:id/:action", get(proxy_domain).post(proxy_domain).put(proxy_domain).delete(proxy_domain))
        .with_state(state)
}

/// Extension trait that registers CRUD + optional sub-actions for a domain.
trait DomainRouterExt {
    fn domain_routes(self, domain: &'static str, actions: &'static [&'static str]) -> Self;
}

impl DomainRouterExt for Router<AppState> {
    fn domain_routes(self, domain: &'static str, actions: &'static [&'static str]) -> Self {
        let base = format!("/api/v1/admin/{}", domain);
        let by_id = format!("/api/v1/admin/{}/:id", domain);
        let mut r = self
            .route(&base, get(proxy_domain).post(proxy_domain))
            .route(&by_id, get(proxy_domain).put(proxy_domain).delete(proxy_domain));
        for a in actions {
            let p = format!("/api/v1/admin/{}/:id/{}", domain, a);
            r = r.route(&p, post(proxy_domain).put(proxy_domain));
        }
        r
    }
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
