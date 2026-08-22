//! API routes — REAL SQL handlers wired to PostgreSQL via sqlx::PgPool.
//!
//! Every handler runs SELECT/INSERT/UPDATE/DELETE against the admin tables
//! created by `database::run_migrations` (mirroring the Go backend schema).
//! Tenant-scoped queries use `state.white_label_id` (WL_CLIENT_ID). Numeric
//! columns are cast to ::text on SELECT so the `String` model fields decode
//! cleanly (Postgres NUMERIC is not natively String-decodable by sqlx). No
//! handler returns fake data or echoes input — empty tables yield empty
//! arrays, which is real, not stubbed.
use axum::{
    extract::{Path, State},
    response::Json,
    routing::{delete, get, post, put},
    Router,
};
use chrono::{Duration, Utc};
use sqlx::PgPool;
use uuid::Uuid;

use crate::database::AppState;
use crate::middleware::auth::Claims;
use crate::models::*;

pub async fn health_check() -> Json<serde_json::Value> {
    Json(serde_json::json!({"status": "healthy", "service": "tiger-admin-rust"}))
}

/// Helper: issue an HS256 access + refresh JWT for an admin user, using the
/// AppState's JWT secret. Mirrors the Go `issueJWT`.
fn issue_jwt(admin: &AdminUser, secret: &str) -> Result<(String, String), String> {
    use jsonwebtoken::{encode, Algorithm, EncodingKey, Header};
    let now = Utc::now();
    let access_claims = Claims {
        sub: admin.id.to_string(),
        email: admin.email.clone(),
        role: admin.role.clone(),
        iat: now.timestamp(),
        exp: (now + Duration::hours(24)).timestamp(),
    };
    let refresh_claims = Claims {
        sub: admin.id.to_string(),
        email: admin.email.clone(),
        role: admin.role.clone(),
        iat: now.timestamp(),
        exp: (now + Duration::days(7)).timestamp(),
    };
    let header = Header::new(Algorithm::HS256);
    let key = EncodingKey::from_secret(secret.as_bytes());
    let access = encode(&header, &access_claims, &key).map_err(|e| e.to_string())?;
    let refresh = encode(&header, &refresh_claims, &key).map_err(|e| e.to_string())?;
    Ok((access, refresh))
}

/// Write an audit log row (best-effort: failures are logged, not fatal).
async fn audit(
    pool: &PgPool,
    admin_id: Option<Uuid>,
    action: &str,
    resource_type: &str,
    resource_id: Option<&str>,
) {
    if let Err(e) = sqlx::query(
        "INSERT INTO audit_logs (admin_id, action, resource_type, resource_id) VALUES ($1,$2,$3,$4)",
    )
    .bind(admin_id)
    .bind(action)
    .bind(resource_type)
    .bind(resource_id)
    .execute(pool)
    .await
    {
        tracing::warn!("audit log write failed: {e}");
    }
}

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/health", get(health_check))
        .route("/api/v1/auth/login", post(login))
        .route("/api/v1/auth/register", post(register))
        .route("/api/v1/admin/users", get(get_users))
        .route("/api/v1/admin/users/:id", get(get_user))
        .route("/api/v1/admin/users/:id/ban", post(ban_user))
        .route("/api/v1/admin/users/:id/unban", post(unban_user))
        .route("/api/v1/admin/users/:id/status", put(update_user_status))
        .route("/api/v1/admin/kyc", get(get_kyc))
        .route("/api/v1/admin/kyc/:id/approve", post(approve_kyc))
        .route("/api/v1/admin/kyc/:id/reject", post(reject_kyc))
        .route("/api/v1/admin/transactions", get(get_transactions))
        .route("/api/v1/admin/withdrawals", get(get_withdrawals))
        .route("/api/v1/admin/withdrawals/:id/approve", post(approve_withdrawal))
        .route("/api/v1/admin/withdrawals/:id/reject", post(reject_withdrawal))
        .route("/api/v1/admin/withdrawals/:id/process", post(process_withdrawal))
        .route("/api/v1/admin/tokens", get(get_tokens).post(create_token))
        .route("/api/v1/admin/pairs", get(get_pairs))
        .route("/api/v1/admin/blockchains", get(get_blockchains).post(create_blockchain))
        .route("/api/v1/admin/fees", get(get_fees))
        .route("/api/v1/admin/webhooks", get(get_webhooks).post(create_webhook))
        .route("/api/v1/admin/notifications", get(get_notifications))
        .route("/api/v1/admin/audit-logs", get(get_audit_logs))
        .route("/api/v1/admin/sessions", get(get_sessions))
        .route("/api/v1/admin/feature-flags", get(get_feature_flags))
        .route("/api/v1/admin/ip-whitelist", get(get_ip_whitelist).post(add_ip_whitelist))
        .route("/api/v1/admin/ip-whitelist/:id", delete(remove_ip_whitelist))
        .route("/api/v1/admin/tickets", get(get_tickets).post(create_ticket))
        .route("/api/v1/admin/tickets/:id", get(get_ticket))
        .route("/api/v1/admin/tickets/:id/status", put(update_ticket_status))
        .route("/api/v1/admin/tickets/:id/messages", post(add_ticket_message))
        .route("/api/v1/admin/tickets/:id/assign", put(assign_ticket))
        .route("/api/v1/admin/white-labels", get(get_white_labels))
        .route("/api/v1/admin/stats", get(get_stats))
        .route("/api/v1/admin/backups", get(get_backups).post(create_backup))
        .route("/api/v1/admin/workflows", get(get_workflows).post(create_workflow))
        .route("/api/v1/admin/approval-requests", get(get_approval_requests))
        // --- 11 domain backends (governance/config records only, no fund movement) ---
        .route("/api/v1/admin/futures", get(list_futures).post(create_futures))
        .route("/api/v1/admin/futures/:id", get(get_futures).put(update_futures).delete(delete_futures))
        .route("/api/v1/admin/futures/:id/status", put(update_futures_status))
        .route("/api/v1/admin/options", get(list_options).post(create_options))
        .route("/api/v1/admin/options/:id", get(get_options).put(update_options).delete(delete_options))
        .route("/api/v1/admin/options/:id/status", put(update_options_status))
        .route("/api/v1/admin/copy-trading", get(list_copy_trading).post(create_copy_trading))
        .route("/api/v1/admin/copy-trading/:id", get(get_copy_trading).put(update_copy_trading).delete(delete_copy_trading))
        .route("/api/v1/admin/copy-trading/:id/status", put(update_copy_trading_status))
        .route("/api/v1/admin/convert", get(list_convert).post(create_convert))
        .route("/api/v1/admin/convert/:id", get(get_convert).put(update_convert).delete(delete_convert))
        .route("/api/v1/admin/convert/:id/status", put(update_convert_status))
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
        .route("/api/v1/admin/p2p-clients/:id/status", put(update_p2p_client_status))
        .route("/api/v1/admin/partners", get(list_partners).post(create_partner))
        .route("/api/v1/admin/partners/:id", get(get_partner).put(update_partner).delete(delete_partner))
        .route("/api/v1/admin/partners/:id/status", put(update_partner_status))
        .route("/api/v1/admin/partners/:id/approve", post(approve_partner))
        .route("/api/v1/admin/partners/:id/reject", post(reject_partner))
        .route("/api/v1/admin/rewards", get(list_rewards).post(create_reward))
        .route("/api/v1/admin/rewards/:id", get(get_reward).put(update_reward).delete(delete_reward))
        .route("/api/v1/admin/rewards/:id/status", put(update_reward_status))
        .route("/api/v1/admin/marketing", get(list_marketing).post(create_marketing))
        .route("/api/v1/admin/marketing/:id", get(get_marketing).put(update_marketing).delete(delete_marketing))
        .route("/api/v1/admin/marketing/:id/status", put(update_marketing_status))
        // --- RBAC: admin-roles, admin-permissions, role assignment ---
        .route("/api/v1/admin/admin-roles", get(list_admin_roles).post(create_admin_role))
        .route("/api/v1/admin/admin-roles/:id", get(get_admin_role).put(update_admin_role).delete(delete_admin_role))
        .route("/api/v1/admin/admin-permissions", get(list_admin_permissions).post(create_admin_permission))
        .route("/api/v1/admin/admin-permissions/:id", get(get_admin_permission).put(update_admin_permission).delete(delete_admin_permission))
        .route("/api/v1/admin/admins/:id/role", post(assign_admin_role))
        .route("/api/v1/admin/admins/:id/permissions", get(get_admin_permissions))
        // --- 4 WL product governance domains (mirrors Go wl_products.go) ---
        .route("/api/v1/admin/wl-liquidity/sources", get(list_wl_liquidity_sources).post(create_wl_liquidity_source))
        .route("/api/v1/admin/wl-liquidity/sources/:id", get(get_wl_liquidity_source).put(update_wl_liquidity_source).delete(delete_wl_liquidity_source))
        .route("/api/v1/admin/wl-liquidity/allocations", get(list_wl_liquidity_allocations).post(set_wl_liquidity_allocation))
        .route("/api/v1/admin/wl-liquidity/stats", get(wl_liquidity_stats))
        .route("/api/v1/admin/wl-cards", get(list_wl_cards).post(issue_wl_card))
        .route("/api/v1/admin/wl-cards/:id/status", put(update_wl_card_status))
        .route("/api/v1/admin/wl-cards/transactions", get(list_wl_card_transactions))
        .route("/api/v1/admin/wl-cards/stats", get(wl_card_stats))
        .route("/api/v1/admin/wl-bots/operators", get(list_wl_bot_operators).post(register_wl_bot_operator))
        .route("/api/v1/admin/wl-bots/operators/:id/status", put(update_wl_bot_operator_status))
        .route("/api/v1/admin/wl-bots/config", get(get_wl_bot_config))
        .route("/api/v1/admin/wl-bots/stats", get(wl_bot_stats))
        .with_state(state)
}

// ===========================================================================
// Auth (login uses bcrypt::verify + issues a real HS256 JWT; register
// bcrypt-hashes the password and inserts an admin_users row).
// ===========================================================================

async fn login(
    State(state): State<AppState>,
    Json(req): Json<LoginRequest>,
) -> Json<ApiResponse<LoginResponse>> {
    let admin = match sqlx::query_as::<_, AdminUser>(
        "SELECT id, username, email, password_hash, role, two_factor_secret,
                two_factor_enabled, is_active, created_at, updated_at, last_login
         FROM admin_users WHERE email = $1",
    )
    .bind(&req.email)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(a) => a,
        Err(e) => return Json(ApiResponse::error(format!("db error: {e}"))),
    };

    let admin = match admin {
        Some(a) if a.is_active => a,
        _ => return Json(ApiResponse::error("invalid credentials".into())),
    };

    let valid = bcrypt::verify(&req.password, &admin.password_hash).unwrap_or(false);
    if !valid {
        return Json(ApiResponse::error("invalid credentials".into()));
    }

    let (access_token, refresh_token) = match issue_jwt(&admin, &state.jwt_secret) {
        Ok(t) => t,
        Err(e) => return Json(ApiResponse::error(format!("jwt error: {e}"))),
    };

    // Record the login time.
    let _ = sqlx::query("UPDATE admin_users SET last_login = NOW() WHERE id = $1")
        .bind(admin.id)
        .execute(&state.pool)
        .await;

    Json(ApiResponse::success(LoginResponse {
        admin,
        access_token,
        refresh_token,
    }))
}

async fn register(
    State(state): State<AppState>,
    Json(req): Json<RegisterRequest>,
) -> Json<ApiResponse<AdminUser>> {
    if req.username.is_empty() || req.email.is_empty() || req.password.is_empty() {
        return Json(ApiResponse::error("username, email and password required".into()));
    }
    let hash = match bcrypt::hash(&req.password, bcrypt::DEFAULT_COST) {
        Ok(h) => h,
        Err(e) => return Json(ApiResponse::error(format!("hash error: {e}"))),
    };
    let admin = match sqlx::query_as::<_, AdminUser>(
        "INSERT INTO admin_users (username, email, password_hash, role, white_label_id)
         VALUES ($1, $2, $3, 'admin', $4)
         RETURNING id, username, email, password_hash, role, two_factor_secret,
                   two_factor_enabled, is_active, created_at, updated_at, last_login",
    )
    .bind(&req.username)
    .bind(&req.email)
    .bind(&hash)
    .bind(state.white_label_id)
    .fetch_one(&state.pool)
    .await
    {
        Ok(a) => a,
        Err(e) => return Json(ApiResponse::error(format!("db error: {e}"))),
    };
    Json(ApiResponse::success(admin))
}

// ===========================================================================
// Users (tenant-scoped)
// ===========================================================================

async fn get_users(State(state): State<AppState>) -> Json<ApiResponse<Vec<User>>> {
    match sqlx::query_as::<_, User>(
        "SELECT id, email, username, wallet_address, kyc_status, status, created_at
         FROM users WHERE white_label_id = $1 ORDER BY created_at DESC",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_user(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<User>> {
    match sqlx::query_as::<_, User>(
        "SELECT id, email, username, wallet_address, kyc_status, status, created_at
         FROM users WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(u)) => Json(ApiResponse::success(u)),
        Ok(None) => Json(ApiResponse::error("user not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn ban_user(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    let res = sqlx::query("UPDATE users SET status = 'banned' WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await;
    match res {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("user not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "user.ban", "user", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn unban_user(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    let res = sqlx::query("UPDATE users SET status = 'active' WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await;
    match res {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("user not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "user.unban", "user", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

/// PUT /users/:id/status — real status update (active/banned/suspended).
async fn update_user_status(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(req): Json<UserStatusUpdateRequest>,
) -> Json<ApiResponse<()>> {
    if req.status.is_empty() {
        return Json(ApiResponse::error("status required".into()));
    }
    match sqlx::query("UPDATE users SET status = $1 WHERE id = $2 AND white_label_id = $3")
        .bind(&req.status)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("user not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "user.status", "user", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// ===========================================================================
// KYC (tenant-scoped)
// ===========================================================================

async fn get_kyc(State(state): State<AppState>) -> Json<ApiResponse<Vec<KycRequest>>> {
    match sqlx::query_as::<_, KycRequest>(
        "SELECT k.id, k.user_id, k.doc_type, k.status, k.submitted_at
         FROM kyc_requests k
         JOIN users u ON k.user_id = u.id
         WHERE u.white_label_id = $1
         ORDER BY k.submitted_at DESC",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn approve_kyc(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE kyc_requests SET status = 'approved', reviewed_at = NOW(), reject_reason = ''
         WHERE id = $1 AND EXISTS (
             SELECT 1 FROM users u WHERE u.id = kyc_requests.user_id AND u.white_label_id = $2
         )",
    )
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("kyc request not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "kyc.approve", "kyc", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn reject_kyc(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(r): Json<RejectRequest>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE kyc_requests SET status = 'rejected', reviewed_at = NOW(), reject_reason = $1
         WHERE id = $2 AND EXISTS (
             SELECT 1 FROM users u WHERE u.id = kyc_requests.user_id AND u.white_label_id = $3
         )",
    )
    .bind(&r.reason)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("kyc request not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "kyc.reject", "kyc", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// ===========================================================================
// Transactions (tenant-scoped, read-only)
// ===========================================================================

async fn get_transactions(State(state): State<AppState>) -> Json<ApiResponse<Vec<Transaction>>> {
    match sqlx::query_as::<_, Transaction>(
        "SELECT t.id, t.user_id, t.type AS tx_type, t.amount::text AS amount,
                t.currency, t.status, t.timestamp
         FROM transactions t
         JOIN users u ON t.user_id = u.id
         WHERE u.white_label_id = $1
         ORDER BY t.timestamp DESC LIMIT 200",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// ===========================================================================
// Withdrawals (tenant-scoped; approve/reject/process — WL-side only)
// ===========================================================================

async fn get_withdrawals(State(state): State<AppState>) -> Json<ApiResponse<Vec<Withdrawal>>> {
    match sqlx::query_as::<_, Withdrawal>(
        "SELECT w.id, w.user_id, w.amount::text AS amount, w.currency, w.status,
                w.address, w.created_at
         FROM withdrawals w
         JOIN users u ON w.user_id = u.id
         WHERE u.white_label_id = $1
         ORDER BY w.created_at DESC LIMIT 200",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn approve_withdrawal(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE withdrawals SET status = 'approved'
         WHERE id = $1 AND EXISTS (
             SELECT 1 FROM users u, withdrawals w
             WHERE u.id = w.user_id AND u.white_label_id = $2 AND w.id = $1
         )",
    )
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("withdrawal not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "withdrawal.approve_wl_side", "withdrawal", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn reject_withdrawal(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE withdrawals SET status = 'rejected'
         WHERE id = $1 AND EXISTS (
             SELECT 1 FROM users u, withdrawals w
             WHERE u.id = w.user_id AND u.white_label_id = $2 AND w.id = $1
         )",
    )
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("withdrawal not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "withdrawal.reject", "withdrawal", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

/// ProcessWithdrawal — records the tx_hash after the SuperAdmin two-party gate
/// is satisfied. The actual broadcast happens in the master-wallet backend.
async fn process_withdrawal(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(req): Json<TxHashRequest>,
) -> Json<ApiResponse<()>> {
    if req.tx_hash.is_empty() {
        return Json(ApiResponse::error("tx_hash required".into()));
    }
    match sqlx::query(
        "UPDATE withdrawals SET status = 'processed', tx_hash = $1, processed_at = NOW()
         WHERE id = $2 AND EXISTS (
             SELECT 1 FROM users u, withdrawals w
             WHERE u.id = w.user_id AND u.white_label_id = $3 AND w.id = $2
         )",
    )
    .bind(&req.tx_hash)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("withdrawal not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "withdrawal.process", "withdrawal", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// ===========================================================================
// Tokens / pairs / blockchains / fees / webhooks / notifications / audit /
// sessions / feature-flags / ip-whitelist / tickets / white-labels / stats /
// backups / workflows / approval-requests.
// ===========================================================================

async fn get_tokens(State(state): State<AppState>) -> Json<ApiResponse<Vec<Token>>> {
    match sqlx::query_as::<_, Token>(
        "SELECT id, symbol, name, is_active, is_verified, created_at
         FROM tokens WHERE white_label_id = $1 ORDER BY created_at DESC",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_token(
    State(state): State<AppState>,
    Json(t): Json<Token>,
) -> Json<ApiResponse<Token>> {
    match sqlx::query_as::<_, Token>(
        "INSERT INTO tokens (white_label_id, symbol, name, is_active, is_verified)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, symbol, name, is_active, is_verified, created_at",
    )
    .bind(state.white_label_id)
    .bind(&t.symbol)
    .bind(&t.name)
    .bind(t.is_active)
    .bind(t.is_verified)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_pairs(State(state): State<AppState>) -> Json<ApiResponse<Vec<TradingPair>>> {
    match sqlx::query_as::<_, TradingPair>(
        "SELECT id, pair_name, status, created_at
         FROM trading_pairs WHERE white_label_id = $1 ORDER BY created_at DESC",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_blockchains(State(state): State<AppState>) -> Json<ApiResponse<Vec<Blockchain>>> {
    match sqlx::query_as::<_, Blockchain>(
        "SELECT id, name, symbol, chain_id, is_evm, is_active, created_at
         FROM blockchains WHERE white_label_id = $1 OR white_label_id IS NULL
         ORDER BY created_at DESC",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_blockchain(
    State(state): State<AppState>,
    Json(b): Json<Blockchain>,
) -> Json<ApiResponse<Blockchain>> {
    match sqlx::query_as::<_, Blockchain>(
        "INSERT INTO blockchains (white_label_id, name, symbol, chain_id, is_evm, is_active)
         VALUES ($1, $2, $3, $4, $5, $6)
         RETURNING id, name, symbol, chain_id, is_evm, is_active, created_at",
    )
    .bind(state.white_label_id)
    .bind(&b.name)
    .bind(&b.symbol)
    .bind(b.chain_id)
    .bind(b.is_evm)
    .bind(b.is_active)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_fees(State(state): State<AppState>) -> Json<ApiResponse<Vec<FeeStructure>>> {
    match sqlx::query_as::<_, FeeStructure>(
        "SELECT id, fee_type, fee_percent::text AS fee_percent, is_active, created_at
         FROM fee_structures WHERE white_label_id = $1 OR white_label_id IS NULL
         ORDER BY created_at DESC",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_webhooks(State(state): State<AppState>) -> Json<ApiResponse<Vec<Webhook>>> {
    match sqlx::query_as::<_, Webhook>(
        "SELECT id, name, url, events, is_active, created_at
         FROM webhooks WHERE white_label_id = $1 OR white_label_id IS NULL
         ORDER BY created_at DESC",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_webhook(
    State(state): State<AppState>,
    Json(w): Json<Webhook>,
) -> Json<ApiResponse<Webhook>> {
    match sqlx::query_as::<_, Webhook>(
        "INSERT INTO webhooks (white_label_id, name, url, events, is_active)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, name, url, events, is_active, created_at",
    )
    .bind(state.white_label_id)
    .bind(&w.name)
    .bind(&w.url)
    .bind(&w.events)
    .bind(w.is_active)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_notifications(State(state): State<AppState>) -> Json<ApiResponse<Vec<Notification>>> {
    match sqlx::query_as::<_, Notification>(
        "SELECT id, admin_id, title, message, notification_type, is_read, created_at
         FROM notifications ORDER BY created_at DESC LIMIT 200",
    )
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_audit_logs(State(state): State<AppState>) -> Json<ApiResponse<Vec<AuditLog>>> {
    match sqlx::query_as::<_, AuditLog>(
        "SELECT id, admin_id, action, resource_type, resource_id, created_at
         FROM audit_logs ORDER BY created_at DESC LIMIT 200",
    )
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_sessions(State(state): State<AppState>) -> Json<ApiResponse<Vec<Session>>> {
    match sqlx::query_as::<_, Session>(
        "SELECT id, admin_id, ip_address, expires_at, created_at
         FROM admin_sessions ORDER BY created_at DESC LIMIT 200",
    )
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_feature_flags(State(state): State<AppState>) -> Json<ApiResponse<Vec<FeatureFlag>>> {
    match sqlx::query_as::<_, FeatureFlag>(
        "SELECT id, name, description, is_enabled, rollout_percentage, created_at
         FROM feature_flags ORDER BY created_at DESC",
    )
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// --- ip-whitelist (missing POST/DELETE now wired) ---

async fn get_ip_whitelist(State(state): State<AppState>) -> Json<ApiResponse<Vec<IpWhitelist>>> {
    match sqlx::query_as::<_, IpWhitelist>(
        "SELECT id, ip_address, description, is_active, created_at
         FROM ip_whitelist ORDER BY created_at DESC",
    )
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn add_ip_whitelist(
    State(state): State<AppState>,
    Json(req): Json<IpWhitelistCreateRequest>,
) -> Json<ApiResponse<IpWhitelist>> {
    if req.ip_address.is_empty() {
        return Json(ApiResponse::error("ip_address required".into()));
    }
    match sqlx::query_as::<_, IpWhitelist>(
        "INSERT INTO ip_whitelist (ip_address, description, is_active)
         VALUES ($1, $2, TRUE)
         RETURNING id, ip_address, description, is_active, created_at",
    )
    .bind(&req.ip_address)
    .bind(&req.description)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => {
            audit(&state.pool, None, "ip_whitelist.add", "ip_whitelist", Some(&row.id.to_string())).await;
            Json(ApiResponse::success(row))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn remove_ip_whitelist(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM ip_whitelist WHERE id = $1")
        .bind(id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("ip whitelist entry not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "ip_whitelist.remove", "ip_whitelist", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// --- tickets (missing POST/:id/status/messages/assign now wired) ---

async fn get_tickets(State(state): State<AppState>) -> Json<ApiResponse<Vec<Ticket>>> {
    match sqlx::query_as::<_, Ticket>(
        "SELECT id, title, description, ticket_type, priority, status, created_by,
                assigned_to, created_at
         FROM tickets WHERE white_label_id = $1 ORDER BY created_at DESC",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_ticket(
    State(state): State<AppState>,
    Json(req): Json<CreateTicketRequest>,
) -> Json<ApiResponse<Ticket>> {
    if req.title.is_empty() {
        return Json(ApiResponse::error("title required".into()));
    }
    let ttype = req.ticket_type.unwrap_or_else(|| "support".to_string());
    let priority = req.priority.unwrap_or_else(|| "medium".to_string());
    match sqlx::query_as::<_, Ticket>(
        "INSERT INTO tickets (white_label_id, user_id, title, description, ticket_type, priority, status, created_by)
         VALUES ($1, $2, $3, $4, $5, $6, 'open', NULL)
         RETURNING id, title, description, ticket_type, priority, status, created_by, assigned_to, created_at",
    )
    .bind(state.white_label_id)
    .bind(req.user_id)
    .bind(&req.title)
    .bind(&req.description)
    .bind(&ttype)
    .bind(&priority)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => {
            audit(&state.pool, None, "ticket.create", "ticket", Some(&row.id.to_string())).await;
            Json(ApiResponse::success(row))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_ticket(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<Ticket>> {
    match sqlx::query_as::<_, Ticket>(
        "SELECT id, title, description, ticket_type, priority, status, created_by,
                assigned_to, created_at
         FROM tickets WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(t)) => Json(ApiResponse::success(t)),
        Ok(None) => Json(ApiResponse::error("ticket not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_ticket_status(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(req): Json<StatusUpdate>,
) -> Json<ApiResponse<()>> {
    if req.status.is_empty() {
        return Json(ApiResponse::error("status required".into()));
    }
    match sqlx::query(
        "UPDATE tickets SET status = $1, updated_at = NOW()
         WHERE id = $2 AND white_label_id = $3",
    )
    .bind(&req.status)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("ticket not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "ticket.status", "ticket", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn add_ticket_message(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(req): Json<CreateTicketMessageRequest>,
) -> Json<ApiResponse<IdResponse>> {
    if req.message.is_empty() {
        return Json(ApiResponse::error("message required".into()));
    }
    let internal = req.is_internal.unwrap_or(false);
    // Ensure the ticket belongs to this tenant before attaching a message.
    let exists: Option<(Uuid,)> = sqlx::query_as("SELECT id FROM tickets WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .fetch_optional(&state.pool)
        .await
        .unwrap_or(None);
    if exists.is_none() {
        return Json(ApiResponse::error("ticket not found".into()));
    }
    match sqlx::query_as::<_, (Uuid,)>("INSERT INTO ticket_messages (ticket_id, sender_id, message, is_internal) VALUES ($1, NULL, $2, $3) RETURNING id")
        .bind(id)
        .bind(&req.message)
        .bind(internal)
        .fetch_one(&state.pool)
        .await
    {
        Ok((mid,)) => {
            audit(&state.pool, None, "ticket.message", "ticket_message", Some(&mid.to_string())).await;
            Json(ApiResponse::success(IdResponse { id: mid }))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn assign_ticket(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(req): Json<AssignTicketRequest>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE tickets SET assigned_to = $1, updated_at = NOW()
         WHERE id = $2 AND white_label_id = $3",
    )
    .bind(req.assigned_to)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("ticket not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "ticket.assign", "ticket", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// --- white-labels / stats / backups / workflows / approval-requests ---

async fn get_white_labels(State(state): State<AppState>) -> Json<ApiResponse<Vec<WhiteLabel>>> {
    match sqlx::query_as::<_, WhiteLabel>(
        "SELECT id, name, domain, is_active, created_at FROM white_labels ORDER BY created_at DESC",
    )
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_stats(State(state): State<AppState>) -> Json<ApiResponse<PlatformStats>> {
    let total_users: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM users WHERE white_label_id = $1")
        .bind(state.white_label_id)
        .fetch_one(&state.pool)
        .await
        .unwrap_or(0);
    let active_users: i64 =
        sqlx::query_scalar("SELECT COUNT(*) FROM users WHERE white_label_id = $1 AND status = 'active'")
            .bind(state.white_label_id)
            .fetch_one(&state.pool)
            .await
            .unwrap_or(0);
    let total_transactions: i64 = sqlx::query_scalar(
        "SELECT COUNT(*) FROM transactions t JOIN users u ON t.user_id = u.id WHERE u.white_label_id = $1",
    )
    .bind(state.white_label_id)
    .fetch_one(&state.pool)
    .await
    .unwrap_or(0);
    let total_volume: f64 = sqlx::query_scalar(
        "SELECT COALESCE(SUM(amount), 0)::float8 FROM transactions t
         JOIN users u ON t.user_id = u.id WHERE u.white_label_id = $1",
    )
    .bind(state.white_label_id)
    .fetch_one(&state.pool)
    .await
    .unwrap_or(0.0);
    Json(ApiResponse::success(PlatformStats {
        total_users,
        active_users,
        total_transactions,
        total_volume,
    }))
}

async fn get_backups(State(state): State<AppState>) -> Json<ApiResponse<Vec<Backup>>> {
    match sqlx::query_as::<_, Backup>(
        "SELECT id, backup_type, file_path, status, created_at FROM backups ORDER BY created_at DESC LIMIT 100",
    )
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_backup(
    State(state): State<AppState>,
    Json(b): Json<Backup>,
) -> Json<ApiResponse<Backup>> {
    match sqlx::query_as::<_, Backup>(
        "INSERT INTO backups (backup_type, file_path, status)
         VALUES ($1, $2, $3)
         RETURNING id, backup_type, file_path, status, created_at",
    )
    .bind(&b.backup_type)
    .bind(&b.file_path)
    .bind(&b.status)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_workflows(State(state): State<AppState>) -> Json<ApiResponse<Vec<ApprovalWorkflow>>> {
    match sqlx::query_as::<_, ApprovalWorkflow>(
        "SELECT id, name, workflow_type, threshold_amount::text AS threshold_amount,
                required_approvals, is_active, created_at
         FROM approval_workflows ORDER BY created_at DESC",
    )
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_workflow(
    State(state): State<AppState>,
    Json(w): Json<ApprovalWorkflow>,
) -> Json<ApiResponse<ApprovalWorkflow>> {
    match sqlx::query_as::<_, ApprovalWorkflow>(
        "INSERT INTO approval_workflows (name, workflow_type, threshold_amount, required_approvals, is_active)
         VALUES ($1, $2, $3::numeric, $4, $5)
         RETURNING id, name, workflow_type, threshold_amount::text AS threshold_amount,
                   required_approvals, is_active, created_at",
    )
    .bind(&w.name)
    .bind(&w.workflow_type)
    .bind(&w.threshold_amount)
    .bind(w.required_approvals)
    .bind(w.is_active)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_approval_requests(State(state): State<AppState>) -> Json<ApiResponse<Vec<ApprovalRequest>>> {
    match sqlx::query_as::<_, ApprovalRequest>(
        "SELECT id, workflow_id, request_type, resource_id, requester_id, status, created_at
         FROM approval_requests ORDER BY created_at DESC LIMIT 200",
    )
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// ===========================================================================
// 11 domain backends — governance/config records only (no fund movement).
// Each handler runs real SQL against its tenant-scoped table.
// ===========================================================================

// --- futures (table: futures_positions) ---

async fn list_futures(State(state): State<AppState>) -> Json<ApiResponse<Vec<FuturesConfig>>> {
    let res = sqlx::query_as::<_, FuturesConfig>(
        "SELECT id, white_label_id, pair AS symbol, side AS contract_type,
                leverage::int4 AS leverage_max, 'USD'::text AS margin_currency,
                status, created_at
         FROM futures_positions WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await;
    match res {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_futures(
    State(state): State<AppState>,
    Json(c): Json<FuturesConfig>,
) -> Json<ApiResponse<FuturesConfig>> {
    match sqlx::query_as::<_, FuturesConfig>(
        "INSERT INTO futures_positions (white_label_id, pair, side, leverage, status)
         VALUES ($1, $2, $3, $4::numeric, $5)
         RETURNING id, white_label_id, pair AS symbol, side AS contract_type,
                   leverage::int4 AS leverage_max, 'USD'::text AS margin_currency,
                   status, created_at",
    )
    .bind(state.white_label_id)
    .bind(&c.symbol)
    .bind(&c.contract_type)
    .bind(c.leverage_max)
    .bind(&c.status)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_futures(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<FuturesConfig>> {
    match sqlx::query_as::<_, FuturesConfig>(
        "SELECT id, white_label_id, pair AS symbol, side AS contract_type,
                leverage::int4 AS leverage_max, 'USD'::text AS margin_currency,
                status, created_at
         FROM futures_positions WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(c)) => Json(ApiResponse::success(c)),
        Ok(None) => Json(ApiResponse::error("position not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_futures(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(c): Json<FuturesConfig>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE futures_positions SET pair = $1, side = $2, leverage = $3::numeric,
               status = $4, updated_at = NOW()
         WHERE id = $5 AND white_label_id = $6",
    )
    .bind(&c.symbol)
    .bind(&c.contract_type)
    .bind(c.leverage_max)
    .bind(&c.status)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("position not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_futures(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM futures_positions WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("position not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_futures_status(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(s): Json<StatusUpdate>,
) -> Json<ApiResponse<()>> {
    if s.status.is_empty() {
        return Json(ApiResponse::error("status required".into()));
    }
    match sqlx::query("UPDATE futures_positions SET status = $1, updated_at = NOW() WHERE id = $2 AND white_label_id = $3")
        .bind(&s.status)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("position not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// --- options (table: options_contracts) ---

async fn list_options(State(state): State<AppState>) -> Json<ApiResponse<Vec<OptionsConfig>>> {
    match sqlx::query_as::<_, OptionsConfig>(
        "SELECT id, white_label_id, symbol, option_type, strike::text AS strike,
                expiry, status, created_at
         FROM options_contracts WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_options(
    State(state): State<AppState>,
    Json(c): Json<OptionsConfig>,
) -> Json<ApiResponse<OptionsConfig>> {
    match sqlx::query_as::<_, OptionsConfig>(
        "INSERT INTO options_contracts (white_label_id, symbol, option_type, strike, expiry, status)
         VALUES ($1, $2, $3, $4::numeric, $5, $6)
         RETURNING id, white_label_id, symbol, option_type, strike::text AS strike, expiry, status, created_at",
    )
    .bind(state.white_label_id)
    .bind(&c.symbol)
    .bind(&c.option_type)
    .bind(&c.strike)
    .bind(c.expiry)
    .bind(&c.status)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_options(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<OptionsConfig>> {
    match sqlx::query_as::<_, OptionsConfig>(
        "SELECT id, white_label_id, symbol, option_type, strike::text AS strike,
                expiry, status, created_at
         FROM options_contracts WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(c)) => Json(ApiResponse::success(c)),
        Ok(None) => Json(ApiResponse::error("contract not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_options(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(c): Json<OptionsConfig>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE options_contracts SET symbol = $1, option_type = $2, strike = $3::numeric,
               expiry = $4, status = $5
         WHERE id = $6 AND white_label_id = $7",
    )
    .bind(&c.symbol)
    .bind(&c.option_type)
    .bind(&c.strike)
    .bind(c.expiry)
    .bind(&c.status)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("contract not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_options(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM options_contracts WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("contract not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_options_status(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(s): Json<StatusUpdate>,
) -> Json<ApiResponse<()>> {
    if s.status.is_empty() {
        return Json(ApiResponse::error("status required".into()));
    }
    match sqlx::query("UPDATE options_contracts SET status = $1 WHERE id = $2 AND white_label_id = $3")
        .bind(&s.status)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("contract not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// --- copy-trading (table: copy_trading_configs) ---

async fn list_copy_trading(State(state): State<AppState>) -> Json<ApiResponse<Vec<CopyTradingConfig>>> {
    match sqlx::query_as::<_, CopyTradingConfig>(
        "SELECT id, white_label_id, lead_trader_id, max_followers, fee_bps, status, created_at
         FROM copy_trading_configs WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_copy_trading(
    State(state): State<AppState>,
    Json(c): Json<CopyTradingConfig>,
) -> Json<ApiResponse<CopyTradingConfig>> {
    match sqlx::query_as::<_, CopyTradingConfig>(
        "INSERT INTO copy_trading_configs (white_label_id, lead_trader_id, max_followers, fee_bps, status)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, white_label_id, lead_trader_id, max_followers, fee_bps, status, created_at",
    )
    .bind(state.white_label_id)
    .bind(c.lead_trader_id)
    .bind(c.max_followers)
    .bind(c.fee_bps)
    .bind(&c.status)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_copy_trading(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<CopyTradingConfig>> {
    match sqlx::query_as::<_, CopyTradingConfig>(
        "SELECT id, white_label_id, lead_trader_id, max_followers, fee_bps, status, created_at
         FROM copy_trading_configs WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(c)) => Json(ApiResponse::success(c)),
        Ok(None) => Json(ApiResponse::error("config not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_copy_trading(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(c): Json<CopyTradingConfig>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE copy_trading_configs SET lead_trader_id = $1, max_followers = $2, fee_bps = $3, status = $4
         WHERE id = $5 AND white_label_id = $6",
    )
    .bind(c.lead_trader_id)
    .bind(c.max_followers)
    .bind(c.fee_bps)
    .bind(&c.status)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("config not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_copy_trading(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM copy_trading_configs WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("config not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_copy_trading_status(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(s): Json<StatusUpdate>,
) -> Json<ApiResponse<()>> {
    if s.status.is_empty() {
        return Json(ApiResponse::error("status required".into()));
    }
    match sqlx::query("UPDATE copy_trading_configs SET status = $1 WHERE id = $2 AND white_label_id = $3")
        .bind(&s.status)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("config not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// --- convert (table: convert_orders) ---

async fn list_convert(State(state): State<AppState>) -> Json<ApiResponse<Vec<ConvertConfig>>> {
    match sqlx::query_as::<_, ConvertConfig>(
        "SELECT id, white_label_id, from_currency, to_currency, spread_bps, status, created_at
         FROM convert_orders WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_convert(
    State(state): State<AppState>,
    Json(c): Json<ConvertConfig>,
) -> Json<ApiResponse<ConvertConfig>> {
    match sqlx::query_as::<_, ConvertConfig>(
        "INSERT INTO convert_orders (white_label_id, from_currency, to_currency, spread_bps, status)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, white_label_id, from_currency, to_currency, spread_bps, status, created_at",
    )
    .bind(state.white_label_id)
    .bind(&c.from_currency)
    .bind(&c.to_currency)
    .bind(c.spread_bps)
    .bind(&c.status)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_convert(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<ConvertConfig>> {
    match sqlx::query_as::<_, ConvertConfig>(
        "SELECT id, white_label_id, from_currency, to_currency, spread_bps, status, created_at
         FROM convert_orders WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(c)) => Json(ApiResponse::success(c)),
        Ok(None) => Json(ApiResponse::error("order not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_convert(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(c): Json<ConvertConfig>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE convert_orders SET from_currency = $1, to_currency = $2, spread_bps = $3, status = $4
         WHERE id = $5 AND white_label_id = $6",
    )
    .bind(&c.from_currency)
    .bind(&c.to_currency)
    .bind(c.spread_bps)
    .bind(&c.status)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("order not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_convert(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM convert_orders WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("order not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_convert_status(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(s): Json<StatusUpdate>,
) -> Json<ApiResponse<()>> {
    if s.status.is_empty() {
        return Json(ApiResponse::error("status required".into()));
    }
    match sqlx::query("UPDATE convert_orders SET status = $1 WHERE id = $2 AND white_label_id = $3")
        .bind(&s.status)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("order not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// --- onramp (table: onramp_orders) ---

async fn list_onramp(State(state): State<AppState>) -> Json<ApiResponse<Vec<OnrampOrder>>> {
    match sqlx::query_as::<_, OnrampOrder>(
        "SELECT id, white_label_id, user_id, fiat_currency, fiat_amount::text AS fiat_amount,
                crypto_currency, status, reject_reason, created_at
         FROM onramp_orders WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_onramp(
    State(state): State<AppState>,
    Json(o): Json<OnrampOrder>,
) -> Json<ApiResponse<OnrampOrder>> {
    match sqlx::query_as::<_, OnrampOrder>(
        "INSERT INTO onramp_orders (white_label_id, user_id, fiat_currency, fiat_amount, crypto_currency, status)
         VALUES ($1, $2, $3, $4::numeric, $5, $6)
         RETURNING id, white_label_id, user_id, fiat_currency, fiat_amount::text AS fiat_amount,
                   crypto_currency, status, reject_reason, created_at",
    )
    .bind(state.white_label_id)
    .bind(o.user_id)
    .bind(&o.fiat_currency)
    .bind(&o.fiat_amount)
    .bind(&o.crypto_currency)
    .bind(&o.status)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_onramp(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<OnrampOrder>> {
    match sqlx::query_as::<_, OnrampOrder>(
        "SELECT id, white_label_id, user_id, fiat_currency, fiat_amount::text AS fiat_amount,
                crypto_currency, status, reject_reason, created_at
         FROM onramp_orders WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(o)) => Json(ApiResponse::success(o)),
        Ok(None) => Json(ApiResponse::error("order not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_onramp(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(o): Json<OnrampOrder>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE onramp_orders SET fiat_currency = $1, fiat_amount = $2::numeric, crypto_currency = $3, status = $4
         WHERE id = $5 AND white_label_id = $6",
    )
    .bind(&o.fiat_currency)
    .bind(&o.fiat_amount)
    .bind(&o.crypto_currency)
    .bind(&o.status)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("order not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_onramp(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM onramp_orders WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("order not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn approve_onramp(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("UPDATE onramp_orders SET status = 'approved', reject_reason = NULL WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("order not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "onramp.approve", "onramp", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn reject_onramp(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(r): Json<RejectRequest>,
) -> Json<ApiResponse<()>> {
    match sqlx::query("UPDATE onramp_orders SET status = 'rejected', reject_reason = $1 WHERE id = $2 AND white_label_id = $3")
        .bind(&r.reason)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("order not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "onramp.reject", "onramp", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// --- offramp (table: offramp_orders) ---

async fn list_offramp(State(state): State<AppState>) -> Json<ApiResponse<Vec<OfframpOrder>>> {
    match sqlx::query_as::<_, OfframpOrder>(
        "SELECT id, white_label_id, user_id, crypto_currency, crypto_amount::text AS crypto_amount,
                fiat_currency, status, reject_reason, created_at
         FROM offramp_orders WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_offramp(
    State(state): State<AppState>,
    Json(o): Json<OfframpOrder>,
) -> Json<ApiResponse<OfframpOrder>> {
    match sqlx::query_as::<_, OfframpOrder>(
        "INSERT INTO offramp_orders (white_label_id, user_id, crypto_currency, crypto_amount, fiat_currency, status)
         VALUES ($1, $2, $3, $4::numeric, $5, $6)
         RETURNING id, white_label_id, user_id, crypto_currency, crypto_amount::text AS crypto_amount,
                   fiat_currency, status, reject_reason, created_at",
    )
    .bind(state.white_label_id)
    .bind(o.user_id)
    .bind(&o.crypto_currency)
    .bind(&o.crypto_amount)
    .bind(&o.fiat_currency)
    .bind(&o.status)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_offramp(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<OfframpOrder>> {
    match sqlx::query_as::<_, OfframpOrder>(
        "SELECT id, white_label_id, user_id, crypto_currency, crypto_amount::text AS crypto_amount,
                fiat_currency, status, reject_reason, created_at
         FROM offramp_orders WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(o)) => Json(ApiResponse::success(o)),
        Ok(None) => Json(ApiResponse::error("order not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_offramp(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(o): Json<OfframpOrder>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE offramp_orders SET crypto_currency = $1, crypto_amount = $2::numeric, fiat_currency = $3, status = $4
         WHERE id = $5 AND white_label_id = $6",
    )
    .bind(&o.crypto_currency)
    .bind(&o.crypto_amount)
    .bind(&o.fiat_currency)
    .bind(&o.status)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("order not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_offramp(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM offramp_orders WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("order not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn approve_offramp(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("UPDATE offramp_orders SET status = 'approved', reject_reason = NULL WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("order not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "offramp.approve", "offramp", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn reject_offramp(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(r): Json<RejectRequest>,
) -> Json<ApiResponse<()>> {
    match sqlx::query("UPDATE offramp_orders SET status = 'rejected', reject_reason = $1 WHERE id = $2 AND white_label_id = $3")
        .bind(&r.reason)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("order not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "offramp.reject", "offramp", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// --- p2p-clients (table: p2p_clients) ---

async fn list_p2p_clients(State(state): State<AppState>) -> Json<ApiResponse<Vec<P2pClient>>> {
    match sqlx::query_as::<_, P2pClient>(
        "SELECT id, white_label_id, user_id, display_name, rating::float8 AS rating,
                total_trades, status, created_at
         FROM p2p_clients WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_p2p_client(
    State(state): State<AppState>,
    Json(c): Json<P2pClient>,
) -> Json<ApiResponse<P2pClient>> {
    match sqlx::query_as::<_, P2pClient>(
        "INSERT INTO p2p_clients (white_label_id, user_id, display_name, rating, total_trades, status)
         VALUES ($1, $2, $3, $4::numeric, $5, $6)
         RETURNING id, white_label_id, user_id, display_name, rating::float8 AS rating,
                   total_trades, status, created_at",
    )
    .bind(state.white_label_id)
    .bind(c.user_id)
    .bind(&c.display_name)
    .bind(c.rating)
    .bind(c.total_trades)
    .bind(&c.status)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_p2p_client(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<P2pClient>> {
    match sqlx::query_as::<_, P2pClient>(
        "SELECT id, white_label_id, user_id, display_name, rating::float8 AS rating,
                total_trades, status, created_at
         FROM p2p_clients WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(c)) => Json(ApiResponse::success(c)),
        Ok(None) => Json(ApiResponse::error("client not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_p2p_client(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(c): Json<P2pClient>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE p2p_clients SET display_name = $1, rating = $2::numeric, total_trades = $3, status = $4
         WHERE id = $5 AND white_label_id = $6",
    )
    .bind(&c.display_name)
    .bind(c.rating)
    .bind(c.total_trades)
    .bind(&c.status)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("client not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_p2p_client(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM p2p_clients WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("client not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_p2p_client_status(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(s): Json<StatusUpdate>,
) -> Json<ApiResponse<()>> {
    if s.status.is_empty() {
        return Json(ApiResponse::error("status required".into()));
    }
    match sqlx::query("UPDATE p2p_clients SET status = $1 WHERE id = $2 AND white_label_id = $3")
        .bind(&s.status)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("client not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// --- partners (table: partners) ---

async fn list_partners(State(state): State<AppState>) -> Json<ApiResponse<Vec<Partner>>> {
    match sqlx::query_as::<_, Partner>(
        "SELECT id, white_label_id, name, partner_type, api_key_hint, status, created_at
         FROM partners WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_partner(
    State(state): State<AppState>,
    Json(p): Json<Partner>,
) -> Json<ApiResponse<Partner>> {
    match sqlx::query_as::<_, Partner>(
        "INSERT INTO partners (white_label_id, name, partner_type, api_key_hint, status)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, white_label_id, name, partner_type, api_key_hint, status, created_at",
    )
    .bind(state.white_label_id)
    .bind(&p.name)
    .bind(&p.partner_type)
    .bind(&p.api_key_hint)
    .bind(&p.status)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_partner(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<Partner>> {
    match sqlx::query_as::<_, Partner>(
        "SELECT id, white_label_id, name, partner_type, api_key_hint, status, created_at
         FROM partners WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(p)) => Json(ApiResponse::success(p)),
        Ok(None) => Json(ApiResponse::error("partner not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_partner(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(p): Json<Partner>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE partners SET name = $1, partner_type = $2, api_key_hint = $3, status = $4
         WHERE id = $5 AND white_label_id = $6",
    )
    .bind(&p.name)
    .bind(&p.partner_type)
    .bind(&p.api_key_hint)
    .bind(&p.status)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("partner not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_partner(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM partners WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("partner not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_partner_status(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(s): Json<StatusUpdate>,
) -> Json<ApiResponse<()>> {
    if s.status.is_empty() {
        return Json(ApiResponse::error("status required".into()));
    }
    match sqlx::query("UPDATE partners SET status = $1 WHERE id = $2 AND white_label_id = $3")
        .bind(&s.status)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("partner not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn approve_partner(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("UPDATE partners SET status = 'approved' WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("partner not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "partner.approve", "partner", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn reject_partner(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(r): Json<RejectRequest>,
) -> Json<ApiResponse<()>> {
    if r.reason.is_empty() {
        return Json(ApiResponse::error("reason required".into()));
    }
    match sqlx::query("UPDATE partners SET status = 'rejected' WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("partner not found".into())),
        Ok(_) => {
            audit(&state.pool, None, "partner.reject", "partner", Some(&id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// --- rewards (table: reward_campaigns) ---

async fn list_rewards(State(state): State<AppState>) -> Json<ApiResponse<Vec<Reward>>> {
    match sqlx::query_as::<_, Reward>(
        "SELECT id, white_label_id, name, reward_type, amount::text AS amount, status, created_at
         FROM reward_campaigns WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_reward(
    State(state): State<AppState>,
    Json(r): Json<Reward>,
) -> Json<ApiResponse<Reward>> {
    match sqlx::query_as::<_, Reward>(
        "INSERT INTO reward_campaigns (white_label_id, name, reward_type, amount, status)
         VALUES ($1, $2, $3, $4::numeric, $5)
         RETURNING id, white_label_id, name, reward_type, amount::text AS amount, status, created_at",
    )
    .bind(state.white_label_id)
    .bind(&r.name)
    .bind(&r.reward_type)
    .bind(&r.amount)
    .bind(&r.status)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_reward(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<Reward>> {
    match sqlx::query_as::<_, Reward>(
        "SELECT id, white_label_id, name, reward_type, amount::text AS amount, status, created_at
         FROM reward_campaigns WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(r)) => Json(ApiResponse::success(r)),
        Ok(None) => Json(ApiResponse::error("reward not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_reward(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(r): Json<Reward>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE reward_campaigns SET name = $1, reward_type = $2, amount = $3::numeric, status = $4
         WHERE id = $5 AND white_label_id = $6",
    )
    .bind(&r.name)
    .bind(&r.reward_type)
    .bind(&r.amount)
    .bind(&r.status)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("reward not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_reward(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM reward_campaigns WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("reward not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_reward_status(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(s): Json<StatusUpdate>,
) -> Json<ApiResponse<()>> {
    if s.status.is_empty() {
        return Json(ApiResponse::error("status required".into()));
    }
    match sqlx::query("UPDATE reward_campaigns SET status = $1 WHERE id = $2 AND white_label_id = $3")
        .bind(&s.status)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("reward not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// --- marketing (table: marketing_campaigns) ---

async fn list_marketing(State(state): State<AppState>) -> Json<ApiResponse<Vec<MarketingCampaign>>> {
    match sqlx::query_as::<_, MarketingCampaign>(
        "SELECT id, white_label_id, name, channel, budget::text AS budget, status, created_at
         FROM marketing_campaigns WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_marketing(
    State(state): State<AppState>,
    Json(m): Json<MarketingCampaign>,
) -> Json<ApiResponse<MarketingCampaign>> {
    match sqlx::query_as::<_, MarketingCampaign>(
        "INSERT INTO marketing_campaigns (white_label_id, name, channel, budget, status)
         VALUES ($1, $2, $3, $4::numeric, $5)
         RETURNING id, white_label_id, name, channel, budget::text AS budget, status, created_at",
    )
    .bind(state.white_label_id)
    .bind(&m.name)
    .bind(&m.channel)
    .bind(&m.budget)
    .bind(&m.status)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_marketing(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<MarketingCampaign>> {
    match sqlx::query_as::<_, MarketingCampaign>(
        "SELECT id, white_label_id, name, channel, budget::text AS budget, status, created_at
         FROM marketing_campaigns WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(m)) => Json(ApiResponse::success(m)),
        Ok(None) => Json(ApiResponse::error("campaign not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_marketing(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(m): Json<MarketingCampaign>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE marketing_campaigns SET name = $1, channel = $2, budget = $3::numeric, status = $4
         WHERE id = $5 AND white_label_id = $6",
    )
    .bind(&m.name)
    .bind(&m.channel)
    .bind(&m.budget)
    .bind(&m.status)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("campaign not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_marketing(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM marketing_campaigns WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("campaign not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_marketing_status(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(s): Json<StatusUpdate>,
) -> Json<ApiResponse<()>> {
    if s.status.is_empty() {
        return Json(ApiResponse::error("status required".into()));
    }
    match sqlx::query("UPDATE marketing_campaigns SET status = $1 WHERE id = $2 AND white_label_id = $3")
        .bind(&s.status)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("campaign not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// ===========================================================================
// RBAC: admin-roles / admin-permissions / role assignment
// ===========================================================================

async fn list_admin_roles(State(state): State<AppState>) -> Json<ApiResponse<Vec<AdminRole>>> {
    match sqlx::query_as::<_, AdminRole>(
        "SELECT id, white_label_id, name, scopes, is_system, created_at
         FROM admin_roles WHERE white_label_id = $1 ORDER BY created_at DESC",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_admin_role(
    State(state): State<AppState>,
    Json(r): Json<AdminRole>,
) -> Json<ApiResponse<AdminRole>> {
    match sqlx::query_as::<_, AdminRole>(
        "INSERT INTO admin_roles (white_label_id, name, scopes, is_system)
         VALUES ($1, $2, $3, $4)
         RETURNING id, white_label_id, name, scopes, is_system, created_at",
    )
    .bind(state.white_label_id)
    .bind(&r.name)
    .bind(&r.scopes)
    .bind(r.is_system)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_admin_role(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<AdminRole>> {
    match sqlx::query_as::<_, AdminRole>(
        "SELECT id, white_label_id, name, scopes, is_system, created_at
         FROM admin_roles WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(r)) => Json(ApiResponse::success(r)),
        Ok(None) => Json(ApiResponse::error("role not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_admin_role(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(r): Json<AdminRole>,
) -> Json<ApiResponse<()>> {
    match sqlx::query("UPDATE admin_roles SET name = $1, scopes = $2, is_system = $3 WHERE id = $4 AND white_label_id = $5")
        .bind(&r.name)
        .bind(&r.scopes)
        .bind(r.is_system)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("role not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_admin_role(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM admin_roles WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("role not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn list_admin_permissions(State(state): State<AppState>) -> Json<ApiResponse<Vec<AdminPermission>>> {
    let res = sqlx::query_as::<_, AdminPermission>(
        "SELECT id, scope, description, created_at FROM admin_permissions ORDER BY scope",
    )
    .fetch_all(&state.pool)
    .await;
    match res {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_admin_permission(
    State(state): State<AppState>,
    Json(p): Json<AdminPermission>,
) -> Json<ApiResponse<AdminPermission>> {
    match sqlx::query_as::<_, AdminPermission>(
        "INSERT INTO admin_permissions (scope, description) VALUES ($1, $2)
         ON CONFLICT (scope) DO UPDATE SET description = EXCLUDED.description
         RETURNING id, scope, description, created_at",
    )
    .bind(&p.scope)
    .bind(&p.description)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_admin_permission(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<AdminPermission>> {
    match sqlx::query_as::<_, AdminPermission>(
        "SELECT id, scope, description, created_at FROM admin_permissions WHERE id = $1",
    )
    .bind(id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(p)) => Json(ApiResponse::success(p)),
        Ok(None) => Json(ApiResponse::error("permission not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_admin_permission(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(p): Json<AdminPermission>,
) -> Json<ApiResponse<()>> {
    match sqlx::query("UPDATE admin_permissions SET scope = $1, description = $2 WHERE id = $3")
        .bind(&p.scope)
        .bind(&p.description)
        .bind(id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("permission not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_admin_permission(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM admin_permissions WHERE id = $1")
        .bind(id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("permission not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn assign_admin_role(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(req): Json<AssignRoleRequest>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "INSERT INTO admin_role_assignments (admin_id, role_id) VALUES ($1, $2)
         ON CONFLICT (admin_id, role_id) DO NOTHING",
    )
    .bind(id)
    .bind(req.role_id)
    .execute(&state.pool)
    .await
    {
        Ok(_) => {
            audit(&state.pool, Some(id), "admin.role.assign", "admin_role", Some(&req.role_id.to_string())).await;
            Json(ApiResponse::success(()))
        }
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_admin_permissions(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Json<ApiResponse<Vec<AdminPermission>>> {
    match sqlx::query_as::<_, AdminPermission>(
        "SELECT p.id, p.scope, p.description, p.created_at
         FROM admin_permissions p
         JOIN admin_roles r ON r.scopes @> ARRAY[p.scope]::text[]
         JOIN admin_role_assignments a ON a.role_id = r.id
         WHERE a.admin_id = $1",
    )
    .bind(id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

// ===========================================================================
// WL product governance handlers (mirror Go wl_products.go). Governance/config
// records only — no fund movement. Real shapes match the Go backend.
// ===========================================================================

// --- liquidity_admin: /wl-liquidity/sources (+ allocations + stats) ---

async fn list_wl_liquidity_sources(State(state): State<AppState>) -> Json<ApiResponse<Vec<WLLiquiditySource>>> {
    match sqlx::query_as::<_, WLLiquiditySource>(
        "SELECT id, white_label_id, name, chain, dex, pool_address, token_a, token_b,
                reserve_a::text AS reserve_a, reserve_b::text AS reserve_b,
                fee_pct::text AS fee_pct, is_active, created_at
         FROM wl_liquidity_sources WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn create_wl_liquidity_source(
    State(state): State<AppState>,
    Json(s): Json<WLLiquiditySource>,
) -> Json<ApiResponse<WLLiquiditySource>> {
    match sqlx::query_as::<_, WLLiquiditySource>(
        "INSERT INTO wl_liquidity_sources (white_label_id, name, chain, dex, pool_address, token_a, token_b,
                reserve_a, reserve_b, fee_pct, is_active)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8::numeric, $9::numeric, $10::numeric, $11)
         RETURNING id, white_label_id, name, chain, dex, pool_address, token_a, token_b,
                   reserve_a::text AS reserve_a, reserve_b::text AS reserve_b,
                   fee_pct::text AS fee_pct, is_active, created_at",
    )
    .bind(state.white_label_id)
    .bind(&s.name)
    .bind(&s.chain)
    .bind(&s.dex)
    .bind(&s.pool_address)
    .bind(&s.token_a)
    .bind(&s.token_b)
    .bind(&s.reserve_a)
    .bind(&s.reserve_b)
    .bind(&s.fee_pct)
    .bind(s.is_active)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_wl_liquidity_source(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<WLLiquiditySource>> {
    match sqlx::query_as::<_, WLLiquiditySource>(
        "SELECT id, white_label_id, name, chain, dex, pool_address, token_a, token_b,
                reserve_a::text AS reserve_a, reserve_b::text AS reserve_b,
                fee_pct::text AS fee_pct, is_active, created_at
         FROM wl_liquidity_sources WHERE id = $1 AND white_label_id = $2",
    )
    .bind(id)
    .bind(state.white_label_id)
    .fetch_optional(&state.pool)
    .await
    {
        Ok(Some(s)) => Json(ApiResponse::success(s)),
        Ok(None) => Json(ApiResponse::error("source not found".into())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_wl_liquidity_source(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(s): Json<WLLiquiditySource>,
) -> Json<ApiResponse<()>> {
    match sqlx::query(
        "UPDATE wl_liquidity_sources SET name = $1, chain = $2, dex = $3, pool_address = $4,
               token_a = $5, token_b = $6, reserve_a = $7::numeric, reserve_b = $8::numeric,
               fee_pct = $9::numeric, is_active = $10
         WHERE id = $11 AND white_label_id = $12",
    )
    .bind(&s.name)
    .bind(&s.chain)
    .bind(&s.dex)
    .bind(&s.pool_address)
    .bind(&s.token_a)
    .bind(&s.token_b)
    .bind(&s.reserve_a)
    .bind(&s.reserve_b)
    .bind(&s.fee_pct)
    .bind(s.is_active)
    .bind(id)
    .bind(state.white_label_id)
    .execute(&state.pool)
    .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("source not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn delete_wl_liquidity_source(State(state): State<AppState>, Path(id): Path<Uuid>) -> Json<ApiResponse<()>> {
    match sqlx::query("DELETE FROM wl_liquidity_sources WHERE id = $1 AND white_label_id = $2")
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("source not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn list_wl_liquidity_allocations(State(state): State<AppState>) -> Json<ApiResponse<Vec<WLLiquidityAllocation>>> {
    match sqlx::query_as::<_, WLLiquidityAllocation>(
        "SELECT id, white_label_id, name, fee_share_pct::text AS fee_share_pct,
                destination, is_active, created_at, updated_at
         FROM wl_liquidity_allocations WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn set_wl_liquidity_allocation(
    State(state): State<AppState>,
    Json(a): Json<WLLiquidityAllocation>,
) -> Json<ApiResponse<WLLiquidityAllocation>> {
    match sqlx::query_as::<_, WLLiquidityAllocation>(
        "INSERT INTO wl_liquidity_allocations (white_label_id, name, fee_share_pct, destination, is_active)
         VALUES ($1, $2, $3::numeric, $4, $5)
         RETURNING id, white_label_id, name, fee_share_pct::text AS fee_share_pct,
                   destination, is_active, created_at, updated_at",
    )
    .bind(state.white_label_id)
    .bind(&a.name)
    .bind(&a.fee_share_pct)
    .bind(&a.destination)
    .bind(a.is_active)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn wl_liquidity_stats(State(state): State<AppState>) -> Json<serde_json::Value> {
    let total: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM wl_liquidity_sources WHERE white_label_id = $1")
        .bind(state.white_label_id)
        .fetch_one(&state.pool)
        .await
        .unwrap_or(0);
    let active: i64 =
        sqlx::query_scalar("SELECT COUNT(*) FROM wl_liquidity_sources WHERE white_label_id = $1 AND is_active")
            .bind(state.white_label_id)
            .fetch_one(&state.pool)
            .await
            .unwrap_or(0);
    let reserve_a: String = sqlx::query_scalar(
        "SELECT COALESCE(SUM(reserve_a), 0)::text FROM wl_liquidity_sources WHERE white_label_id = $1",
    )
    .bind(state.white_label_id)
    .fetch_one(&state.pool)
    .await
    .unwrap_or_else(|_| "0".to_string());
    let allocations: i64 =
        sqlx::query_scalar("SELECT COUNT(*) FROM wl_liquidity_allocations WHERE white_label_id = $1")
            .bind(state.white_label_id)
            .fetch_one(&state.pool)
            .await
            .unwrap_or(0);
    Json(serde_json::json!({
        "total_sources": total,
        "active_sources": active,
        "total_reserve_a": reserve_a,
        "allocations": allocations
    }))
}

// --- card_admin: /wl-cards (+ transactions + stats) ---

async fn list_wl_cards(State(state): State<AppState>) -> Json<ApiResponse<Vec<WLCard>>> {
    match sqlx::query_as::<_, WLCard>(
        "SELECT id, white_label_id, user_id, holder_name, status, balance::text AS balance,
                currency, created_at, updated_at
         FROM wl_cards WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn issue_wl_card(
    State(state): State<AppState>,
    Json(c): Json<WLCard>,
) -> Json<ApiResponse<WLCard>> {
    match sqlx::query_as::<_, WLCard>(
        "INSERT INTO wl_cards (white_label_id, user_id, holder_name, status, balance, currency)
         VALUES ($1, $2, $3, $4, $5::numeric, $6)
         RETURNING id, white_label_id, user_id, holder_name, status, balance::text AS balance,
                   currency, created_at, updated_at",
    )
    .bind(state.white_label_id)
    .bind(c.user_id)
    .bind(&c.holder_name)
    .bind(&c.status)
    .bind(&c.balance)
    .bind(&c.currency)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_wl_card_status(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(s): Json<StatusUpdate>,
) -> Json<ApiResponse<()>> {
    if s.status.is_empty() {
        return Json(ApiResponse::error("status required".into()));
    }
    match sqlx::query("UPDATE wl_cards SET status = $1, updated_at = NOW() WHERE id = $2 AND white_label_id = $3")
        .bind(&s.status)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("card not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn list_wl_card_transactions(State(state): State<AppState>) -> Json<ApiResponse<Vec<WLCardTransaction>>> {
    match sqlx::query_as::<_, WLCardTransaction>(
        "SELECT id, white_label_id, card_id, amount::text AS amount, merchant, category, status, created_at
         FROM wl_card_transactions WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 200",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn wl_card_stats(State(state): State<AppState>) -> Json<serde_json::Value> {
    let total: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM wl_cards WHERE white_label_id = $1")
        .bind(state.white_label_id)
        .fetch_one(&state.pool)
        .await
        .unwrap_or(0);
    let active: i64 =
        sqlx::query_scalar("SELECT COUNT(*) FROM wl_cards WHERE white_label_id = $1 AND status = 'active'")
            .bind(state.white_label_id)
            .fetch_one(&state.pool)
            .await
            .unwrap_or(0);
    let frozen: i64 =
        sqlx::query_scalar("SELECT COUNT(*) FROM wl_cards WHERE white_label_id = $1 AND status = 'frozen'")
            .bind(state.white_label_id)
            .fetch_one(&state.pool)
            .await
            .unwrap_or(0);
    let txns: i64 =
        sqlx::query_scalar("SELECT COUNT(*) FROM wl_card_transactions WHERE white_label_id = $1")
            .bind(state.white_label_id)
            .fetch_one(&state.pool)
            .await
            .unwrap_or(0);
    Json(serde_json::json!({
        "total_cards": total,
        "active_cards": active,
        "frozen_cards": frozen,
        "transactions": txns
    }))
}

// --- bot_admin: /wl-bots/operators (+ config + stats) ---

async fn list_wl_bot_operators(State(state): State<AppState>) -> Json<ApiResponse<Vec<WLBotOperator>>> {
    match sqlx::query_as::<_, WLBotOperator>(
        "SELECT id, white_label_id, name, strategy, status, config, created_at, updated_at
         FROM wl_bot_operators WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT 100",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await
    {
        Ok(rows) => Json(ApiResponse::success(rows)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn register_wl_bot_operator(
    State(state): State<AppState>,
    Json(o): Json<WLBotOperator>,
) -> Json<ApiResponse<WLBotOperator>> {
    let cfg = if o.config.is_null() {
        serde_json::Value::Object(serde_json::Map::new())
    } else {
        o.config.clone()
    };
    match sqlx::query_as::<_, WLBotOperator>(
        "INSERT INTO wl_bot_operators (white_label_id, name, strategy, status, config)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, white_label_id, name, strategy, status, config, created_at, updated_at",
    )
    .bind(state.white_label_id)
    .bind(&o.name)
    .bind(&o.strategy)
    .bind(&o.status)
    .bind(&cfg)
    .fetch_one(&state.pool)
    .await
    {
        Ok(row) => Json(ApiResponse::success(row)),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn update_wl_bot_operator_status(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(s): Json<StatusUpdate>,
) -> Json<ApiResponse<()>> {
    if s.status.is_empty() {
        return Json(ApiResponse::error("status required".into()));
    }
    // Halt is recorded here; resume requires SuperAdmin two-party collaboration,
    // so 'active' is rejected (fail-closed), mirroring the Go backend.
    if s.status == "active" {
        return Json(ApiResponse::error(
            "SuperAdmin collaboration required to resume operator".into(),
        ));
    }
    match sqlx::query("UPDATE wl_bot_operators SET status = $1, updated_at = NOW() WHERE id = $2 AND white_label_id = $3")
        .bind(&s.status)
        .bind(id)
        .bind(state.white_label_id)
        .execute(&state.pool)
        .await
    {
        Ok(r) if r.rows_affected() == 0 => Json(ApiResponse::error("operator not found".into())),
        Ok(_) => Json(ApiResponse::success(())),
        Err(e) => Json(ApiResponse::error(format!("db error: {e}"))),
    }
}

async fn get_wl_bot_config(State(state): State<AppState>) -> Json<serde_json::Value> {
    let rows = sqlx::query_as::<_, (Uuid, String, String, chrono::DateTime<Utc>)>(
        "SELECT id, key, value, updated_at FROM wl_bot_config WHERE white_label_id = $1 ORDER BY key",
    )
    .bind(state.white_label_id)
    .fetch_all(&state.pool)
    .await;
    match rows {
        Ok(rs) => {
            let config: Vec<serde_json::Value> = rs
                .into_iter()
                .map(|(id, key, value, updated)| {
                    serde_json::json!({"id": id, "key": key, "value": value, "updated_at": updated})
                })
                .collect();
            Json(serde_json::json!({"config": config}))
        }
        Err(e) => Json(serde_json::json!({"config": [], "error": e.to_string()})),
    }
}

async fn wl_bot_stats(State(state): State<AppState>) -> Json<serde_json::Value> {
    let total: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM wl_bot_operators WHERE white_label_id = $1")
        .bind(state.white_label_id)
        .fetch_one(&state.pool)
        .await
        .unwrap_or(0);
    let active: i64 =
        sqlx::query_scalar("SELECT COUNT(*) FROM wl_bot_operators WHERE white_label_id = $1 AND status = 'active'")
            .bind(state.white_label_id)
            .fetch_one(&state.pool)
            .await
            .unwrap_or(0);
    let halted: i64 =
        sqlx::query_scalar("SELECT COUNT(*) FROM wl_bot_operators WHERE white_label_id = $1 AND status = 'halted'")
            .bind(state.white_label_id)
            .fetch_one(&state.pool)
            .await
            .unwrap_or(0);
    Json(serde_json::json!({
        "total_operators": total,
        "active": active,
        "halted": halted
    }))
}
