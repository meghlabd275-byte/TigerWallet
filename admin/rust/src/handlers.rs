//! API Handlers module

use axum::{
    extract::{Path, Query, Extension},
    Json, 
};
use crate::models::*;
use crate::db::DbPool;
use crate::auth::AuthState;
use crate::error::{AppError, AppResult};
use crate::AppState;
use std::sync::Arc;

// ============================================================================
// Auth Handlers
// ============================================================================

#[axum::debug_handler]
pub async fn login(
    Extension(state): Extension<Arc<AppState>>,
    Json(payload): Json<LoginRequest>,
) -> AppResult<Json<LoginResponse>> {
    let row = sqlx::query(
        r#"SELECT id, username, email, password_hash, role, permissions, is_active,
                  two_factor_enabled, two_factor_secret, ip_whitelist,
                  created_at, updated_at, last_login_at, failed_attempts, locked_until
           FROM admins WHERE email = $1"#,
    )
    .bind(&payload.email)
    .fetch_optional(&state.db)
    .await
    .map_err(|e| AppError::DatabaseError(e.to_string()))?
    .ok_or_else(|| AppError::AuthenticationError("invalid credentials".to_string()))?;

    use sqlx::Row;
    let admin_id: uuid::Uuid = row.get("id");
    let password_hash: String = row.get("password_hash");
    let is_active: bool = row.get("is_active");
    let failed_attempts: i32 = row.get("failed_attempts");
    let locked_until: Option<chrono::DateTime<chrono::Utc>> = row.get("locked_until");
    let role_str: String = row.get("role");

    if !is_active {
        return Err(AppError::Forbidden);
    }
    if let Some(locked) = locked_until {
        if locked > chrono::Utc::now() {
            return Err(AppError::AuthenticationError(
                "account temporarily locked after too many failed attempts".to_string(),
            ));
        }
    }

    let valid = crate::auth::verify_password(&payload.password, &password_hash)
        .map_err(|e| AppError::InternalServerError(e.to_string()))?;

    if !valid {
        // Increment failed attempts; lock for 15 minutes after 5 failures.
        let attempts = failed_attempts + 1;
        if attempts >= 5 {
            sqlx::query(
                "UPDATE admins SET failed_attempts = 0, locked_until = NOW() + INTERVAL '15 minutes', updated_at = NOW() WHERE id = $1",
            )
            .bind(admin_id)
            .execute(&state.db)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?;
        } else {
            sqlx::query(
                "UPDATE admins SET failed_attempts = $2, updated_at = NOW() WHERE id = $1",
            )
            .bind(admin_id)
            .bind(attempts)
            .execute(&state.db)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?;
        }
        return Err(AppError::AuthenticationError("invalid credentials".to_string()));
    }

    // Success: reset lockout counters and record the login time.
    sqlx::query(
        "UPDATE admins SET failed_attempts = 0, locked_until = NULL, last_login_at = NOW(), updated_at = NOW() WHERE id = $1",
    )
    .bind(admin_id)
    .execute(&state.db)
    .await
    .map_err(|e| AppError::DatabaseError(e.to_string()))?;

    let role = match role_str.as_str() {
        "super_admin" => AdminRole::SuperAdmin,
        "support" => AdminRole::Support,
        "analyst" => AdminRole::Analyst,
        "viewer" => AdminRole::Viewer,
        "white_label_admin" => AdminRole::WhiteLabelAdmin,
        "master_admin" => AdminRole::MasterAdmin,
        _ => AdminRole::Admin,
    };
    let permissions: Vec<String> = row
        .get::<sqlx::types::Json<Vec<String>>, _>("permissions")
        .0;

    let admin = Admin {
        id: admin_id,
        username: row.get("username"),
        email: row.get("email"),
        password_hash: String::new(), // never return the hash
        role,
        permissions,
        is_active,
        two_factor_enabled: row.get("two_factor_enabled"),
        two_factor_secret: None, // never return the secret
        ip_whitelist: row.get("ip_whitelist"),
        created_at: row.get("created_at"),
        updated_at: row.get("updated_at"),
        last_login_at: Some(chrono::Utc::now()),
        failed_attempts: 0,
        locked_until: None,
    };

    let token = state.auth.generate_token(admin.id, &admin.email, &role_str)?;
    let refresh_token = state.auth.generate_refresh_token(admin.id)?;

    // Record the session so logout/revocation has a real row to act on.
    let session_hash = crate::totp::base32_encode(refresh_token.as_bytes());
    sqlx::query(
        "INSERT INTO sessions (admin_id, token_hash, expires_at) VALUES ($1, $2, NOW() + INTERVAL '7 days')",
    )
    .bind(admin_id)
    .bind(&session_hash[..32])
    .execute(&state.db)
    .await
    .map_err(|e| AppError::DatabaseError(e.to_string()))?;

    Ok(Json(LoginResponse {
        token,
        refresh_token,
        admin,
    }))
}

pub async fn logout() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Logged out" })))
}

#[derive(serde::Deserialize)]
pub struct RefreshRequest {
    pub refresh_token: String,
}

pub async fn refresh_token(
    Extension(state): Extension<Arc<AppState>>,
    Json(payload): Json<RefreshRequest>,
) -> AppResult<Json<serde_json::Value>> {
    let claims = state
        .auth
        .validate_token(&payload.refresh_token)
        .map_err(|_| AppError::AuthenticationError("invalid refresh token".to_string()))?;
    if claims.role != "refresh" {
        return Err(AppError::AuthenticationError(
            "token is not a refresh token".to_string(),
        ));
    }
    let admin_id = uuid::Uuid::parse_str(&claims.sub)
        .map_err(|_| AppError::AuthenticationError("invalid token subject".to_string()))?;

    // The admin must still exist and be active for a refresh to be honored.
    let row = sqlx::query("SELECT email, role, is_active FROM admins WHERE id = $1")
        .bind(admin_id)
        .fetch_optional(&state.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?
        .ok_or_else(|| AppError::AuthenticationError("admin not found".to_string()))?;

    use sqlx::Row;
    let is_active: bool = row.get("is_active");
    if !is_active {
        return Err(AppError::Forbidden);
    }
    let email: String = row.get("email");
    let role: String = row.get("role");

    let token = state.auth.generate_token(admin_id, &email, &role)?;
    let refresh_token = state.auth.generate_refresh_token(admin_id)?;

    Ok(Json(serde_json::json!({
        "token": token,
        "refresh_token": refresh_token,
    })))
}

#[derive(serde::Deserialize)]
pub struct TwoFactorSetupRequest {
    pub admin_id: uuid::Uuid,
    pub password: String,
}

pub async fn setup_2fa(
    Extension(state): Extension<Arc<AppState>>,
    Json(payload): Json<TwoFactorSetupRequest>,
) -> AppResult<Json<serde_json::Value>> {
    // 2FA setup changes account security state, so it requires the admin's
    // password - the route itself is unauthenticated.
    let row = sqlx::query("SELECT password_hash, email, is_active FROM admins WHERE id = $1")
        .bind(payload.admin_id)
        .fetch_optional(&state.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?
        .ok_or_else(|| AppError::NotFound("admin not found".to_string()))?;

    use sqlx::Row;
    let is_active: bool = row.get("is_active");
    if !is_active {
        return Err(AppError::Forbidden);
    }
    let password_hash: String = row.get("password_hash");
    let email: String = row.get("email");
    let valid = crate::auth::verify_password(&payload.password, &password_hash)
        .map_err(|e| AppError::InternalServerError(e.to_string()))?;
    if !valid {
        return Err(AppError::AuthenticationError("invalid credentials".to_string()));
    }

    let secret = crate::totp::generate_totp_secret();
    let secret_b32 = crate::totp::base32_encode(&secret);

    // Secret is stored but 2FA stays disabled until a code verifies.
    sqlx::query("UPDATE admins SET two_factor_secret = $2, updated_at = NOW() WHERE id = $1")
        .bind(payload.admin_id)
        .bind(&secret_b32)
        .execute(&state.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

    Ok(Json(serde_json::json!({
        "secret": secret_b32,
        "otpauth_url": format!(
            "otpauth://totp/TigerWallet:{}?secret={}&issuer=TigerWallet",
            email, secret_b32
        ),
    })))
}

#[derive(serde::Deserialize)]
pub struct TwoFactorVerifyRequest {
    pub admin_id: uuid::Uuid,
    pub password: String,
    pub code: String,
}

pub async fn verify_2fa(
    Extension(state): Extension<Arc<AppState>>,
    Json(payload): Json<TwoFactorVerifyRequest>,
) -> AppResult<Json<serde_json::Value>> {
    let row = sqlx::query(
        "SELECT password_hash, two_factor_secret, is_active FROM admins WHERE id = $1",
    )
    .bind(payload.admin_id)
    .fetch_optional(&state.db)
    .await
    .map_err(|e| AppError::DatabaseError(e.to_string()))?
    .ok_or_else(|| AppError::NotFound("admin not found".to_string()))?;

    use sqlx::Row;
    let is_active: bool = row.get("is_active");
    if !is_active {
        return Err(AppError::Forbidden);
    }
    let password_hash: String = row.get("password_hash");
    let valid = crate::auth::verify_password(&payload.password, &password_hash)
        .map_err(|e| AppError::InternalServerError(e.to_string()))?;
    if !valid {
        return Err(AppError::AuthenticationError("invalid credentials".to_string()));
    }

    let secret_b32: Option<String> = row.get("two_factor_secret");
    let secret_b32 = secret_b32
        .ok_or_else(|| AppError::BadRequest("2FA setup has not been initiated".to_string()))?;
    let secret = crate::totp::base32_decode(&secret_b32)
        .ok_or_else(|| AppError::InternalServerError("stored 2FA secret is corrupted".to_string()))?;

    if !crate::totp::totp_verify(&secret, &payload.code, chrono::Utc::now().timestamp()) {
        return Err(AppError::AuthenticationError("invalid 2FA code".to_string()));
    }

    sqlx::query("UPDATE admins SET two_factor_enabled = true, updated_at = NOW() WHERE id = $1")
        .bind(payload.admin_id)
        .execute(&state.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

    Ok(Json(serde_json::json!({ "verified": true })))
}

// ============================================================================
// Admin Handlers
// ============================================================================

pub async fn list_admins() -> AppResult<Json<Vec<serde_json::Value>>> {
    Ok(Json(vec![]))
}

pub async fn create_admin(
    Json(payload): Json<CreateAdminRequest>,
) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": uuid::Uuid::new_v4() })))
}

pub async fn get_admin(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": id })))
}

pub async fn update_admin(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Admin updated" })))
}

pub async fn delete_admin(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Admin deleted" })))
}

pub async fn suspend_admin(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Admin suspended" })))
}

pub async fn activate_admin(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Admin activated" })))
}

// ============================================================================
// User Handlers
// ============================================================================

pub async fn list_users() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "users": [], "total": 0 })))
}

pub async fn get_user(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": id })))
}

pub async fn update_user(
    Path(id): Path<uuid::Uuid>,
    Json(payload): Json<UpdateUserRequest>,
) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "User updated" })))
}

pub async fn ban_user(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "User banned" })))
}

pub async fn unban_user(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "User unbanned" })))
}

pub async fn suspend_user(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "User suspended" })))
}

// ============================================================================
// KYC Handlers
// ============================================================================

pub async fn list_kyc() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "requests": [], "total": 0 })))
}

pub async fn get_kyc(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": id })))
}

pub async fn approve_kyc(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "KYC approved" })))
}

pub async fn reject_kyc(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "KYC rejected" })))
}

// ============================================================================
// Transaction Handlers
// ============================================================================

pub async fn list_transactions() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "transactions": [], "total": 0 })))
}

pub async fn get_transaction(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": id })))
}

pub async fn flag_transaction(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Transaction flagged" })))
}

pub async fn unflag_transaction(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Transaction unflagged" })))
}

// ============================================================================
// Withdrawal Handlers
// ============================================================================

pub async fn list_withdrawals() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "withdrawals": [], "total": 0 })))
}

pub async fn get_withdrawal(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": id })))
}

pub async fn approve_withdrawal(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Withdrawal approved" })))
}

pub async fn reject_withdrawal(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Withdrawal rejected" })))
}

pub async fn process_withdrawal(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Withdrawal processed" })))
}

// ============================================================================
// Token Handlers
// ============================================================================

pub async fn list_tokens() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "tokens": [], "total": 0 })))
}

pub async fn get_token(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": id })))
}

pub async fn create_token() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": uuid::Uuid::new_v4() })))
}

pub async fn update_token(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Token updated" })))
}

pub async fn delete_token(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Token deleted" })))
}

pub async fn verify_token(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Token verified" })))
}

// ============================================================================
// Pair Handlers
// ============================================================================

pub async fn list_pairs() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "pairs": [], "total": 0 })))
}

pub async fn get_pair(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": id })))
}

pub async fn create_pair() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": uuid::Uuid::new_v4() })))
}

pub async fn update_pair(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Pair updated" })))
}

pub async fn halt_pair(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Pair halted" })))
}

pub async fn activate_pair(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Pair activated" })))
}

// ============================================================================
// Blockchain Handlers
// ============================================================================

pub async fn list_blockchains() -> AppResult<Json<Vec<serde_json::Value>>> {
    Ok(Json(vec![]))
}

pub async fn get_blockchain(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": id })))
}

pub async fn create_blockchain() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": uuid::Uuid::new_v4() })))
}

pub async fn update_blockchain(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Blockchain updated" })))
}

// ============================================================================
// Fee Handlers
// ============================================================================

pub async fn list_fees() -> AppResult<Json<Vec<serde_json::Value>>> {
    Ok(Json(vec![]))
}

pub async fn create_fee() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": uuid::Uuid::new_v4() })))
}

pub async fn update_fee(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Fee updated" })))
}

// ============================================================================
// White Label Handlers
// ============================================================================

pub async fn list_whitelabels() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "white_labels": [], "total": 0 })))
}

pub async fn get_whitelabel(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": id })))
}

pub async fn create_whitelabel() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": uuid::Uuid::new_v4() })))
}

pub async fn update_whitelabel(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "White label updated" })))
}

pub async fn activate_whitelabel(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "White label activated" })))
}

pub async fn suspend_whitelabel(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "White label suspended" })))
}

// ============================================================================
// Ticket Handlers
// ============================================================================

pub async fn list_tickets() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "tickets": [], "total": 0 })))
}

pub async fn get_ticket(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": id })))
}

pub async fn create_ticket() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": uuid::Uuid::new_v4() })))
}

pub async fn update_ticket_status(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Ticket status updated" })))
}

pub async fn assign_ticket(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Ticket assigned" })))
}

pub async fn add_ticket_message(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Message added" })))
}

// ============================================================================
// Analytics Handlers
// ============================================================================

pub async fn dashboard_stats() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({
        "total_users": 0,
        "active_users": 0,
        "total_volume": 0.0,
        "total_transactions": 0,
        "total_fees": 0.0
    })))
}

pub async fn user_analytics() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({})))
}

pub async fn transaction_analytics() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({})))
}

pub async fn revenue_analytics() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({})))
}

// ============================================================================
// Audit Handlers
// ============================================================================

pub async fn list_audit_logs() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "logs": [], "total": 0 })))
}

pub async fn export_audit_logs() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!([])))
}

// ============================================================================
// Feature Flag Handlers
// ============================================================================

pub async fn list_feature_flags() -> AppResult<Json<Vec<serde_json::Value>>> {
    Ok(Json(vec![]))
}

pub async fn create_feature_flag() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": uuid::Uuid::new_v4() })))
}

pub async fn update_feature_flag(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Feature flag updated" })))
}

pub async fn delete_feature_flag(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Feature flag deleted" })))
}

// ============================================================================
// Notification Handlers
// ============================================================================

pub async fn list_notifications() -> AppResult<Json<Vec<serde_json::Value>>> {
    Ok(Json(vec![]))
}

pub async fn mark_notification_read(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Notification marked as read" })))
}

pub async fn broadcast_notification() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Notification broadcast" })))
}

// ============================================================================
// IP Whitelist Handlers
// ============================================================================

pub async fn list_ip_whitelist() -> AppResult<Json<Vec<serde_json::Value>>> {
    Ok(Json(vec![]))
}

pub async fn add_ip_whitelist() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "IP added to whitelist" })))
}

pub async fn remove_ip_whitelist(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "IP removed from whitelist" })))
}

// ============================================================================
// Backup Handlers
// ============================================================================

pub async fn list_backups() -> AppResult<Json<Vec<serde_json::Value>>> {
    Ok(Json(vec![]))
}

pub async fn create_backup() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": uuid::Uuid::new_v4() })))
}

pub async fn restore_backup(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Backup restored" })))
}

pub async fn delete_backup(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Backup deleted" })))
}

// ============================================================================
// Webhook Handlers
// ============================================================================

pub async fn list_webhooks() -> AppResult<Json<Vec<serde_json::Value>>> {
    Ok(Json(vec![]))
}

pub async fn create_webhook() -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "id": uuid::Uuid::new_v4() })))
}

pub async fn update_webhook(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Webhook updated" })))
}

pub async fn test_webhook(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Webhook tested" })))
}

pub async fn delete_webhook(Path(id): Path<uuid::Uuid>) -> AppResult<Json<serde_json::Value>> {
    Ok(Json(serde_json::json!({ "message": "Webhook deleted" })))
}
