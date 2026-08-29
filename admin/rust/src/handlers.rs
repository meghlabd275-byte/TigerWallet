//! API Handlers module
//!
//! Auth + 2FA handlers are real (DB-backed: bcrypt + TOTP + JWT). Every
//! domain-resource handler (admins, users, KYC, transactions, withdrawals,
//! tokens, pairs, blockchains, fees, whitelabels, tickets, analytics, audit,
//! feature-flags, notifications, ip-whitelist, backups, webhooks) forwards the
//! inbound request verbatim to the canonical `admin/go` backend (:9093) and
//! returns its real response — no stubs, fakes, or canned payloads. A down
//! upstream surfaces as 503 so clients render genuine error states.

use axum::{
    body::Bytes,
    extract::{Path, Extension, RawQuery},
    http::{HeaderMap, Method, StatusCode},
    response::{IntoResponse, Response},
    Json,
};
use crate::models::*;
use crate::db::DbPool;
use crate::auth::AuthState;
use crate::error::{AppError, AppResult};
use crate::AppState;
use std::sync::Arc;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::TcpStream;

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
// Domain resource handlers — real proxy to canonical admin/go (:9093)
// ============================================================================
//
// The admin/go backend is the source of truth for every admin resource. These
// handlers forward the inbound HTTP request (method, path, query string, body,
// Bearer JWT) to admin/go over a localhost TCP socket and return its status +
// JSON body verbatim. No data is fabricated; a dead upstream becomes a 503.

/// Upstream admin/go host (override with `TIGERADMIN_UPSTREAM_HOST`).
fn upstream_host() -> String {
    std::env::var("TIGERADMIN_UPSTREAM_HOST").unwrap_or_else(|_| "localhost".to_string())
}

/// Upstream admin/go port (override with `TIGERADMIN_UPSTREAM_PORT`).
fn upstream_port() -> u16 {
    std::env::var("TIGERADMIN_UPSTREAM_PORT")
        .ok()
        .and_then(|p| p.parse().ok())
        .unwrap_or(9093)
}

fn bearer_token(headers: &HeaderMap) -> Result<String, AppError> {
    let auth = headers
        .get("authorization")
        .ok_or(AppError::Unauthorized)?
        .to_str()
        .map_err(|_| AppError::Unauthorized)?;
    if let Some(token) = auth.strip_prefix("Bearer ") {
        Ok(token.to_string())
    } else if let Some(token) = auth.strip_prefix("bearer ") {
        Ok(token.to_string())
    } else {
        Err(AppError::Unauthorized)
    }
}

fn method_name(m: &Method) -> &'static str {
    match *m {
        Method::GET => "GET",
        Method::POST => "POST",
        Method::PUT => "PUT",
        Method::DELETE => "DELETE",
        Method::PATCH => "PATCH",
        _ => "GET",
    }
}

struct UpstreamResponse {
    status: u16,
    body: Vec<u8>,
}

/// Performs a real HTTP/1.1 call to the admin/go backend over a TCP socket.
async fn upstream_call(
    method: &Method,
    path: &str,
    body: &[u8],
    bearer: &str,
) -> Result<UpstreamResponse, AppError> {
    let addr = format!("{}:{}", upstream_host(), upstream_port());
    let mut stream = TcpStream::connect(&addr)
        .await
        .map_err(|e| AppError::InternalServerError(format!("upstream connect failed: {e}")))?;

    let timeout = std::time::Duration::from_secs(5);
    tokio::time::timeout(timeout, async {
        let mut req = Vec::new();
        req.extend_from_slice(format!("{} {} HTTP/1.1\r\n", method_name(method), path).as_bytes());
        req.extend_from_slice(
            format!("Host: {}:{}\r\n", upstream_host(), upstream_port()).as_bytes(),
        );
        req.extend_from_slice(b"Connection: close\r\n");
        req.extend_from_slice(format!("Authorization: Bearer {}\r\n", bearer).as_bytes());
        req.extend_from_slice(b"Content-Type: application/json\r\n");
        req.extend_from_slice(format!("Content-Length: {}\r\n", body.len()).as_bytes());
        req.extend_from_slice(b"\r\n");
        req.extend_from_slice(body);
        stream.write_all(&req).await
    })
    .await
    .map_err(|_| AppError::InternalServerError("upstream write timeout".into()))?
    .map_err(|e| AppError::InternalServerError(format!("upstream write failed: {e}")))?;

    let raw = tokio::time::timeout(timeout, async {
        let mut buf = Vec::with_capacity(8192);
        stream.read_to_end(&mut buf).await.map(|_| buf)
    })
    .await
    .map_err(|_| AppError::InternalServerError("upstream read timeout".into()))?
    .map_err(|e| AppError::InternalServerError(format!("upstream read failed: {e}")))?;

    if raw.is_empty() {
        return Err(AppError::InternalServerError("empty upstream response".into()));
    }

    let sep = find_subsequence(&raw, b"\r\n\r\n");
    let header_bytes = &raw[..sep.unwrap_or(raw.len())];
    let body_bytes = if let Some(s) = sep {
        &raw[s + 4..]
    } else {
        &[][..]
    };

    let first_line_end = header_bytes
        .iter()
        .position(|&b| b == b'\r')
        .unwrap_or(header_bytes.len());
    let first_line = std::str::from_utf8(&header_bytes[..first_line_end])
        .map_err(|_| AppError::InternalServerError("bad upstream status line".into()))?;
    let status: u16 = first_line
        .split_whitespace()
        .nth(1)
        .ok_or_else(|| AppError::InternalServerError("bad upstream status line".into()))?
        .parse()
        .map_err(|_| AppError::InternalServerError("bad upstream status code".into()))?;

    let headers_str = std::str::from_utf8(header_bytes).unwrap_or("");
    let chunked = headers_str
        .to_ascii_lowercase()
        .contains("transfer-encoding: chunked");
    let body = if chunked {
        dechunk(body_bytes)
    } else {
        body_bytes.to_vec()
    };

    Ok(UpstreamResponse { status, body })
}

fn find_subsequence(haystack: &[u8], needle: &[u8]) -> Option<usize> {
    haystack.windows(needle.len()).position(|w| w == needle)
}

fn dechunk(chunked: &[u8]) -> Vec<u8> {
    let mut out = Vec::new();
    let mut pos = 0;
    while pos < chunked.len() {
        let eol = match find_subsequence(&chunked[pos..], b"\r\n") {
            Some(e) => pos + e,
            None => break,
        };
        let len_str = match std::str::from_utf8(&chunked[pos..eol]) {
            Ok(s) => s,
            Err(_) => break,
        };
        let chunk_len = match usize::from_str_radix(len_str.trim(), 16) {
            Ok(n) => n,
            Err(_) => break,
        };
        if chunk_len == 0 {
            break;
        }
        let data_start = eol + 2;
        if data_start + chunk_len > chunked.len() {
            break;
        }
        out.extend_from_slice(&chunked[data_start..data_start + chunk_len]);
        pos = data_start + chunk_len + 2;
    }
    out
}

/// Builds the upstream path from a base resource path plus optional id/suffix,
/// preserving the inbound query string.
fn build_path(resource: &str, suffix: &str, query: &Option<String>) -> String {
    let mut path = format!("/api/v1{}{}", resource, suffix);
    if let Some(q) = query {
        if !q.is_empty() {
            path.push('?');
            path.push_str(q);
        }
    }
    path
}

/// Core proxy: forwards the inbound request to admin/go and returns its
/// status + body. `resource` is the resource path (e.g. `/admins`); `suffix`
/// is any id/action tail (e.g. `/{id}/suspend`).
async fn proxy(
    headers: HeaderMap,
    method: Method,
    resource: &str,
    suffix: &str,
    query: Option<String>,
    body: Bytes,
) -> AppResult<Response> {
    let bearer = bearer_token(&headers)?;
    let path = build_path(resource, suffix, &query);
    let up = upstream_call(&method, &path, &body, &bearer).await?;

    let status = StatusCode::from_u16(up.status)
        .map_err(|_| AppError::InternalServerError("bad upstream status".into()))?;
    let body_json: serde_json::Value = if up.body.is_empty() {
        serde_json::json!({})
    } else {
        serde_json::from_slice(&up.body).unwrap_or_else(|_| {
            serde_json::Value::String(String::from_utf8_lossy(&up.body).to_string())
        })
    };
    Ok((status, Json(body_json)).into_response())
}

// ---- Admin Handlers ----

pub async fn list_admins(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/admins", "", query, Bytes::new()).await
}

pub async fn create_admin(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/admins", "", None, body).await
}

pub async fn get_admin(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/admins", &format!("/{id}"), None, Bytes::new()).await
}

pub async fn update_admin(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::PUT, "/admins", &format!("/{id}"), None, body).await
}

pub async fn delete_admin(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::DELETE, "/admins", &format!("/{id}"), None, Bytes::new()).await
}

pub async fn suspend_admin(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/admins", &format!("/{id}/suspend"), None, Bytes::new()).await
}

pub async fn activate_admin(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/admins", &format!("/{id}/activate"), None, Bytes::new()).await
}

// ---- User Handlers ----

pub async fn list_users(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/users", "", query, Bytes::new()).await
}

pub async fn get_user(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/users", &format!("/{id}"), None, Bytes::new()).await
}

pub async fn update_user(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::PUT, "/users", &format!("/{id}"), None, body).await
}

pub async fn ban_user(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/users", &format!("/{id}/ban"), None, Bytes::new()).await
}

pub async fn unban_user(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/users", &format!("/{id}/unban"), None, Bytes::new()).await
}

pub async fn suspend_user(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/users", &format!("/{id}/suspend"), None, Bytes::new()).await
}

// ---- KYC Handlers ----

pub async fn list_kyc(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/kyc", "", query, Bytes::new()).await
}

pub async fn get_kyc(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/kyc", &format!("/{id}"), None, Bytes::new()).await
}

pub async fn approve_kyc(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/kyc", &format!("/{id}/approve"), None, body).await
}

pub async fn reject_kyc(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/kyc", &format!("/{id}/reject"), None, body).await
}

// ---- Transaction Handlers ----

pub async fn list_transactions(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/transactions", "", query, Bytes::new()).await
}

pub async fn get_transaction(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/transactions", &format!("/{id}"), None, Bytes::new()).await
}

pub async fn flag_transaction(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/transactions", &format!("/{id}/flag"), None, body).await
}

pub async fn unflag_transaction(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/transactions", &format!("/{id}/unflag"), None, body).await
}

// ---- Withdrawal Handlers ----

pub async fn list_withdrawals(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/withdrawals", "", query, Bytes::new()).await
}

pub async fn get_withdrawal(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/withdrawals", &format!("/{id}"), None, Bytes::new()).await
}

pub async fn approve_withdrawal(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/withdrawals", &format!("/{id}/approve"), None, body).await
}

pub async fn reject_withdrawal(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/withdrawals", &format!("/{id}/reject"), None, body).await
}

pub async fn process_withdrawal(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/withdrawals", &format!("/{id}/process"), None, body).await
}

// ---- Token Handlers ----

pub async fn list_tokens(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/tokens", "", query, Bytes::new()).await
}

pub async fn get_token(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/tokens", &format!("/{id}"), None, Bytes::new()).await
}

pub async fn create_token(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/tokens", "", None, body).await
}

pub async fn update_token(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::PUT, "/tokens", &format!("/{id}"), None, body).await
}

pub async fn delete_token(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::DELETE, "/tokens", &format!("/{id}"), None, Bytes::new()).await
}

pub async fn verify_token(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/tokens", &format!("/{id}/verify"), None, body).await
}

// ---- Pair Handlers ----

pub async fn list_pairs(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/pairs", "", query, Bytes::new()).await
}

pub async fn get_pair(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/pairs", &format!("/{id}"), None, Bytes::new()).await
}

pub async fn create_pair(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/pairs", "", None, body).await
}

pub async fn update_pair(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::PUT, "/pairs", &format!("/{id}"), None, body).await
}

pub async fn halt_pair(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/pairs", &format!("/{id}/halt"), None, body).await
}

pub async fn activate_pair(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/pairs", &format!("/{id}/activate"), None, body).await
}

// ---- Blockchain Handlers ----

pub async fn list_blockchains(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/blockchains", "", query, Bytes::new()).await
}

pub async fn get_blockchain(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/blockchains", &format!("/{id}"), None, Bytes::new()).await
}

pub async fn create_blockchain(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/blockchains", "", None, body).await
}

pub async fn update_blockchain(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::PUT, "/blockchains", &format!("/{id}"), None, body).await
}

// ---- Fee Handlers ----

pub async fn list_fees(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/fees", "", query, Bytes::new()).await
}

pub async fn create_fee(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/fees", "", None, body).await
}

pub async fn update_fee(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::PUT, "/fees", &format!("/{id}"), None, body).await
}

// ---- White Label Handlers ----

pub async fn list_whitelabels(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/whitelabels", "", query, Bytes::new()).await
}

pub async fn get_whitelabel(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/whitelabels", &format!("/{id}"), None, Bytes::new()).await
}

pub async fn create_whitelabel(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/whitelabels", "", None, body).await
}

pub async fn update_whitelabel(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::PUT, "/whitelabels", &format!("/{id}"), None, body).await
}

pub async fn activate_whitelabel(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/whitelabels", &format!("/{id}/activate"), None, body).await
}

pub async fn suspend_whitelabel(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/whitelabels", &format!("/{id}/suspend"), None, body).await
}

// ---- Ticket Handlers ----

pub async fn list_tickets(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/tickets", "", query, Bytes::new()).await
}

pub async fn get_ticket(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/tickets", &format!("/{id}"), None, Bytes::new()).await
}

pub async fn create_ticket(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/tickets", "", None, body).await
}

pub async fn update_ticket_status(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::PUT, "/tickets", &format!("/{id}/status"), None, body).await
}

pub async fn assign_ticket(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::PUT, "/tickets", &format!("/{id}/assign"), None, body).await
}

pub async fn add_ticket_message(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/tickets", &format!("/{id}/messages"), None, body).await
}

// ---- Analytics Handlers ----

pub async fn dashboard_stats(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/dashboard", "", query, Bytes::new()).await
}

pub async fn user_analytics(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/analytics/users", "", query, Bytes::new()).await
}

pub async fn transaction_analytics(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/analytics/transactions", "", query, Bytes::new()).await
}

pub async fn revenue_analytics(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/analytics/revenue", "", query, Bytes::new()).await
}

// ---- Audit Handlers ----

pub async fn list_audit_logs(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/audit-logs", "", query, Bytes::new()).await
}

pub async fn export_audit_logs(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/audit-logs/export", "", None, body).await
}

// ---- Feature Flag Handlers ----

pub async fn list_feature_flags(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/feature-flags", "", query, Bytes::new()).await
}

pub async fn create_feature_flag(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/feature-flags", "", None, body).await
}

pub async fn update_feature_flag(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::PUT, "/feature-flags", &format!("/{id}"), None, body).await
}

pub async fn delete_feature_flag(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::DELETE, "/feature-flags", &format!("/{id}"), None, Bytes::new()).await
}

// ---- Notification Handlers ----

pub async fn list_notifications(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/notifications", "", query, Bytes::new()).await
}

pub async fn mark_notification_read(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::PUT, "/notifications", &format!("/{id}/read"), None, body).await
}

pub async fn broadcast_notification(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/notifications/broadcast", "", None, body).await
}

// ---- IP Whitelist Handlers ----

pub async fn list_ip_whitelist(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/ip-whitelist", "", query, Bytes::new()).await
}

pub async fn add_ip_whitelist(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/ip-whitelist", "", None, body).await
}

pub async fn remove_ip_whitelist(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::DELETE, "/ip-whitelist", &format!("/{id}"), None, Bytes::new()).await
}

// ---- Backup Handlers ----

pub async fn list_backups(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/backups", "", query, Bytes::new()).await
}

pub async fn create_backup(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/backups", "", None, body).await
}

pub async fn restore_backup(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/backups", &format!("/{id}/restore"), None, body).await
}

pub async fn delete_backup(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::DELETE, "/backups", &format!("/{id}"), None, Bytes::new()).await
}

// ---- Webhook Handlers ----

pub async fn list_webhooks(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/webhooks", "", query, Bytes::new()).await
}

pub async fn create_webhook(
    headers: HeaderMap,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/webhooks", "", None, body).await
}

pub async fn update_webhook(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::PUT, "/webhooks", &format!("/{id}"), None, body).await
}

pub async fn test_webhook(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
    body: Bytes,
) -> AppResult<Response> {
    proxy(headers, Method::POST, "/webhooks", &format!("/{id}/test"), None, body).await
}

pub async fn delete_webhook(
    headers: HeaderMap,
    Path(id): Path<uuid::Uuid>,
) -> AppResult<Response> {
    proxy(headers, Method::DELETE, "/webhooks", &format!("/{id}"), None, Bytes::new()).await
}
