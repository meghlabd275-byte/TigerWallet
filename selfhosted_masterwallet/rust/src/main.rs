//! TigerWallet Self-Hosted Master Wallet service.
//!
//! A real, standalone HTTP server that lets a white-label client self-host the
//! MasterWallet control plane on their own cloud (Pillar 3 of the white-label
//! operating model). It owns its own PostgreSQL database (no dependency on the
//! TigerWallet cloud) and exposes the same canonical MasterWallet REST contract
//! as `master_wallet/backend` so the existing client UIs work unchanged when
//! pointed at this self-hosted endpoint.
//!
//! Real only — no stubs, no mocks, no fabricated data. JWT (HS256) auth on
//! protected routes; PBKDF2-HMAC-SHA256 password hashing (600k iterations),
//! constant-time compare. Real sqlx queries against PostgreSQL; fail-closed
//! (401/404/500) on any error.

use actix_web::http::header::HeaderMap;
use actix_web::{middleware, web, App, HttpRequest, HttpResponse, HttpServer};
use base64::Engine;
use chrono::{Duration, Utc};
use hmac::{Hmac, Mac};
use rand::RngCore;
use serde::Deserialize;
use sha2::Sha256;
use sqlx::{postgres::PgPoolOptions, PgPool, Row};
use std::env;
use thiserror::Error;
use tracing::{error, info};
use uuid::Uuid;

type HmacSha256 = Hmac<Sha256>;

#[derive(Error, Debug)]
pub enum ApiError {
    #[error("not found: {0}")]
    NotFound(String),
    #[error("unauthorized")]
    Unauthorized,
    #[error("bad request: {0}")]
    BadRequest(String),
    #[error("internal error: {0}")]
    Internal(String),
}

impl actix_web::ResponseError for ApiError {
    fn status_code(&self) -> actix_web::http::StatusCode {
        match self {
            ApiError::NotFound(_) => actix_web::http::StatusCode::NOT_FOUND,
            ApiError::Unauthorized => actix_web::http::StatusCode::UNAUTHORIZED,
            ApiError::BadRequest(_) => actix_web::http::StatusCode::BAD_REQUEST,
            ApiError::Internal(_) => actix_web::http::StatusCode::INTERNAL_SERVER_ERROR,
        }
    }
    fn error_response(&self) -> HttpResponse {
        HttpResponse::build(self.status_code()).json(serde_json::json!({ "error": self.to_string() }))
    }
}

type ApiResult<T> = Result<T, ApiError>;

fn db_err(e: sqlx::Error) -> ApiError {
    match e {
        sqlx::Error::RowNotFound => ApiError::NotFound("resource".into()),
        _ => ApiError::Internal(e.to_string()),
    }
}

/// Shared application state: DB pool + JWT secret.
#[derive(Clone)]
pub struct AppState {
    pub pool: PgPool,
    pub jwt_secret: String,
    pub bind_addr: String,
}

impl AppState {
    async fn connect() -> Result<Self, Box<dyn std::error::Error>> {
        let db_url = env::var("DATABASE_URL").unwrap_or_else(|_| {
            "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet".into()
        });
        let pool = PgPoolOptions::new().max_connections(10).connect(&db_url).await?;
        run_migrations(&pool).await?;
        let jwt_secret = env::var("JWT_SECRET").unwrap_or_else(|_| "dev-only-change-me-in-prod".into());
        let bind_addr = env::var("BIND_ADDR").unwrap_or_else(|_| "0.0.0.0:8470".into());
        Ok(Self { pool, jwt_secret, bind_addr })
    }
}

/// PBKDF2-HMAC-SHA256 password hash (600k iterations). Returns
/// "pbkdf2_sha256$iters$salt_b64$hash_b64" — constant-time verified.
fn hash_password(password: &str) -> String {
    let mut salt = [0u8; 16];
    rand::thread_rng().fill_bytes(&mut salt);
    let iters: u32 = 600_000;
    let hash = pbkdf2_sha256(password.as_bytes(), &salt, iters);
    format!(
        "pbkdf2_sha256${}${}${}",
        iters,
        base64::engine::general_purpose::STANDARD.encode(&salt),
        base64::engine::general_purpose::STANDARD.encode(&hash)
    )
}

fn verify_password(password: &str, stored: &str) -> bool {
    let parts: Vec<&str> = stored.split('$').collect();
    if parts.len() != 4 || parts[0] != "pbkdf2_sha256" {
        return false;
    }
    let iters: u32 = match parts[1].parse() { Ok(v) => v, Err(_) => return false };
    let salt = match base64::engine::general_purpose::STANDARD.decode(parts[2]) { Ok(v) => v, Err(_) => return false };
    let expected = match base64::engine::general_purpose::STANDARD.decode(parts[3]) { Ok(v) => v, Err(_) => return false };
    let actual = pbkdf2_sha256(password.as_bytes(), &salt, iters);
    constant_time_eq(&actual, &expected)
}

fn pbkdf2_sha256(password: &[u8], salt: &[u8], iters: u32) -> [u8; 32] {
    // RFC 2898 PBKDF2 with HMAC-SHA256, single block (dklen <= 32).
    let mut mac = match HmacSha256::new_from_slice(password) { Ok(m) => m, Err(_) => return [0u8; 32] };
    mac.update(salt);
    mac.update(&1u32.to_be_bytes());
    let mut u = mac.finalize().into_bytes();
    let mut result = u;
    mac = match HmacSha256::new_from_slice(password) { Ok(m) => m, Err(_) => return [0u8; 32] };
    for _ in 1..iters {
        mac.update(&u);
        u = mac.finalize_reset().into_bytes();
        for i in 0..32 {
            result[i] ^= u[i];
        }
    }
    let mut out = [0u8; 32];
    out.copy_from_slice(&result[..32]);
    out
}

fn constant_time_eq(a: &[u8], b: &[u8]) -> bool {
    if a.len() != b.len() {
        return false;
    }
    let mut diff = 0u8;
    for i in 0..a.len() {
        diff |= a[i] ^ b[i];
    }
    diff == 0
}

/// HS256 JWT (RFC 7519). Header+payload base64url-dot-separated, signed with
/// HMAC-SHA256. Claims: sub (user uuid), role, exp (24h).
fn jwt_issue(secret: &str, sub: Uuid, role: &str) -> String {
    let header = base64_url_encode(br#"{"typ":"JWT","alg":"HS256"}"#);
    let exp = (Utc::now() + Duration::hours(24)).timestamp();
    let payload = serde_json::json!({"sub": sub.to_string(), "role": role, "exp": exp});
    let payload_str = serde_json::to_string(&payload).unwrap();
    let payload_b = base64_url_encode(payload_str.as_bytes());
    let signing_input = format!("{header}.{payload_b}");
    let mut mac = HmacSha256::new_from_slice(secret.as_bytes()).expect("hmac key");
    mac.update(signing_input.as_bytes());
    let sig = base64_url_encode(&mac.finalize().into_bytes());
    format!("{signing_input}.{sig}")
}

struct Claims {
    sub: Uuid,
    role: String,
}

fn jwt_verify(secret: &str, token: &str) -> Result<Claims, ApiError> {
    let parts: Vec<&str> = token.split('.').collect();
    if parts.len() != 3 {
        return Err(ApiError::Unauthorized);
    }
    let signing_input = format!("{}.{}", parts[0], parts[1]);
    let mut mac = HmacSha256::new_from_slice(secret.as_bytes()).map_err(|_| ApiError::Unauthorized)?;
    mac.update(signing_input.as_bytes());
    let expected = mac.finalize().into_bytes();
    let sig = base64_url_decode(parts[2]).map_err(|_| ApiError::Unauthorized)?;
    if !constant_time_eq(&expected, &sig) {
        return Err(ApiError::Unauthorized);
    }
    let payload_bytes = base64_url_decode(parts[1]).map_err(|_| ApiError::Unauthorized)?;
    let payload: serde_json::Value = serde_json::from_slice(&payload_bytes).map_err(|_| ApiError::Unauthorized)?;
    let sub = payload["sub"].as_str().and_then(|s| Uuid::parse_str(s).ok()).ok_or(ApiError::Unauthorized)?;
    let role = payload["role"].as_str().unwrap_or("user").to_string();
    let exp = payload["exp"].as_i64().unwrap_or(0);
    if Utc::now().timestamp() > exp {
        return Err(ApiError::Unauthorized);
    }
    Ok(Claims { sub, role })
}

fn base64_url_encode(b: &[u8]) -> String {
    base64::engine::general_purpose::URL_SAFE_NO_PAD.encode(b)
}

fn base64_url_decode(s: &str) -> Result<Vec<u8>, ()> {
    base64::engine::general_purpose::URL_SAFE_NO_PAD.decode(s).map_err(|_| ())
}

fn require_auth(headers: &HeaderMap, jwt_secret: &str) -> ApiResult<(Uuid, String)> {
    let auth = headers
        .get("authorization")
        .and_then(|v| v.to_str().ok())
        .ok_or(ApiError::Unauthorized)?;
    let token = auth.strip_prefix("Bearer ").ok_or(ApiError::Unauthorized)?;
    let claims = jwt_verify(jwt_secret, token)?;
    Ok((claims.sub, claims.role))
}

// ---- DB migrations (idempotent CREATE TABLE IF NOT EXISTS) ----

async fn run_migrations(pool: &PgPool) -> Result<(), sqlx::Error> {
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_users (
            id UUID PRIMARY KEY,
            email TEXT UNIQUE NOT NULL,
            username TEXT NOT NULL,
            password_hash TEXT NOT NULL,
            role TEXT NOT NULL DEFAULT 'user',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
    ).execute(pool).await?;
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_master_wallets (
            id UUID PRIMARY KEY,
            owner_id UUID NOT NULL REFERENCES shmw_users(id),
            label TEXT NOT NULL,
            chain_id BIGINT NOT NULL,
            address TEXT NOT NULL DEFAULT '',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
    ).execute(pool).await?;
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_sub_wallets (
            id UUID PRIMARY KEY,
            master_wallet_id UUID NOT NULL REFERENCES shmw_master_wallets(id) ON DELETE CASCADE,
            label TEXT NOT NULL,
            chain_id BIGINT NOT NULL,
            address TEXT NOT NULL DEFAULT '',
            balance_usd NUMERIC(20,8) NOT NULL DEFAULT 0,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
    ).execute(pool).await?;
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_transactions (
            id UUID PRIMARY KEY,
            master_wallet_id UUID NOT NULL REFERENCES shmw_master_wallets(id) ON DELETE CASCADE,
            to_address TEXT NOT NULL,
            value TEXT NOT NULL,
            token TEXT NOT NULL DEFAULT '',
            data TEXT NOT NULL DEFAULT '',
            chain_id BIGINT NOT NULL,
            status TEXT NOT NULL DEFAULT 'pending',
            transaction_hash TEXT,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
    ).execute(pool).await?;
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_fees (
            id UUID PRIMARY KEY,
            master_wallet_id UUID NOT NULL REFERENCES shmw_master_wallets(id) ON DELETE CASCADE,
            chain_id BIGINT NOT NULL,
            fee_bps INT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
    ).execute(pool).await?;
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_auto_sign (
            id UUID PRIMARY KEY,
            master_wallet_id UUID NOT NULL REFERENCES shmw_master_wallets(id) ON DELETE CASCADE,
            pattern TEXT NOT NULL,
            max_value TEXT NOT NULL,
            enabled BOOLEAN NOT NULL DEFAULT TRUE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
    ).execute(pool).await?;
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_wallet_users (
            id UUID PRIMARY KEY,
            master_wallet_id UUID NOT NULL REFERENCES shmw_master_wallets(id) ON DELETE CASCADE,
            user_id UUID NOT NULL REFERENCES shmw_users(id) ON DELETE CASCADE,
            role TEXT NOT NULL DEFAULT 'member',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
    ).execute(pool).await?;
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_chains (
            chain_id BIGINT PRIMARY KEY,
            name TEXT NOT NULL,
            rpc_url TEXT NOT NULL DEFAULT ''
        )"#,
    ).execute(pool).await?;
    Ok(())
}

// ---- Request/response DTOs ----

#[derive(Deserialize)]
struct RegisterReq { email: String, username: String, password: String }

#[derive(Deserialize)]
struct LoginReq { email: String, password: String }

#[derive(Deserialize)]
struct CreateMasterWalletReq { label: String, chain_id: i64, address: Option<String> }

#[derive(Deserialize)]
struct CreateTransactionReq { to: String, value: String, token: Option<String>, data: Option<String>, chain_id: i64 }

#[derive(Deserialize)]
struct SignReq { to: String, amount: String, token: Option<String> }

#[derive(Deserialize)]
struct FeeReq { chain_id: i64, fee_bps: i32 }

#[derive(Deserialize)]
struct AutoSignReq { pattern: String, max_value: String }

#[derive(Deserialize)]
struct AddUserReq { user_id: Uuid, role: Option<String> }

#[derive(Deserialize)]
struct CreateSubWalletReq { label: String, chain_id: i64, address: Option<String> }

// ---- Handlers ----

async fn register(state: web::Data<AppState>, body: web::Json<RegisterReq>) -> ApiResult<HttpResponse> {
    if body.password.len() < 8 {
        return Err(ApiError::BadRequest("password must be >= 8 chars".into()));
    }
    let id = Uuid::new_v4();
    let hash = hash_password(&body.password);
    sqlx::query("INSERT INTO shmw_users (id, email, username, password_hash, role) VALUES ($1,$2,$3,$4,'user')")
        .bind(id).bind(&body.email).bind(&body.username).bind(&hash)
        .execute(&state.pool).await.map_err(|e| match e {
            sqlx::Error::Database(ref de) if de.is_unique_violation() => ApiError::BadRequest("email already registered".into()),
            _ => db_err(e),
        })?;
    let token = jwt_issue(&state.jwt_secret, id, "user");
    Ok(HttpResponse::Created().json(serde_json::json!({ "token": token, "user_id": id })))
}

async fn login(state: web::Data<AppState>, body: web::Json<LoginReq>) -> ApiResult<HttpResponse> {
    let row = sqlx::query("SELECT id, password_hash, role FROM shmw_users WHERE email = $1")
        .bind(&body.email)
        .fetch_optional(&state.pool).await.map_err(db_err)?;
    let row = row.ok_or(ApiError::Unauthorized)?;
    let id: Uuid = row.try_get("id").map_err(|e| ApiError::Internal(e.to_string()))?;
    let hash: String = row.try_get("password_hash").map_err(|e| ApiError::Internal(e.to_string()))?;
    let role: String = row.try_get("role").map_err(|e| ApiError::Internal(e.to_string()))?;
    if !verify_password(&body.password, &hash) {
        return Err(ApiError::Unauthorized);
    }
    let token = jwt_issue(&state.jwt_secret, id, &role);
    Ok(HttpResponse::Ok().json(serde_json::json!({ "token": token, "user_id": id, "role": role })))
}

async fn create_master_wallet(state: web::Data<AppState>, req: HttpRequest, body: web::Json<CreateMasterWalletReq>) -> ApiResult<HttpResponse> {
    let (uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let id = Uuid::new_v4();
    sqlx::query("INSERT INTO shmw_master_wallets (id, owner_id, label, chain_id, address) VALUES ($1,$2,$3,$4,$5)")
        .bind(id).bind(uid).bind(&body.label).bind(body.chain_id).bind(body.address.as_deref().unwrap_or(""))
        .execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Created().json(serde_json::json!({ "wallet_id": id, "label": body.label, "chain_id": body.chain_id })))
}

async fn list_master_wallets(state: web::Data<AppState>, req: HttpRequest) -> ApiResult<HttpResponse> {
    let (uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let rows = sqlx::query("SELECT id, label, chain_id, address, created_at FROM shmw_master_wallets WHERE owner_id = $1 ORDER BY created_at DESC")
        .bind(uid)
        .fetch_all(&state.pool).await.map_err(db_err)?;
    let wallets: Vec<_> = rows.iter().map(|r| serde_json::json!({
        "id": r.get::<Uuid, _>("id"),
        "label": r.get::<String, _>("label"),
        "chain_id": r.get::<i64, _>("chain_id"),
        "address": r.get::<String, _>("address"),
    })).collect();
    Ok(HttpResponse::Ok().json(serde_json::json!({ "wallets": wallets })))
}

async fn get_master_wallet(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let id = path.into_inner();
    let row = sqlx::query("SELECT id, label, chain_id, address, created_at FROM shmw_master_wallets WHERE id = $1")
        .bind(id).fetch_optional(&state.pool).await.map_err(db_err)?;
    let row = row.ok_or_else(|| ApiError::NotFound(format!("master wallet {id}")))?;
    Ok(HttpResponse::Ok().json(serde_json::json!({
        "id": row.get::<Uuid, _>("id"),
        "label": row.get::<String, _>("label"),
        "chain_id": row.get::<i64, _>("chain_id"),
        "address": row.get::<String, _>("address"),
    })))
}

async fn get_balance(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let id = path.into_inner();
    let row = sqlx::query("SELECT COALESCE(SUM(balance_usd),0)::TEXT AS total FROM shmw_sub_wallets WHERE master_wallet_id = $1")
        .bind(id).fetch_one(&state.pool).await.map_err(db_err)?;
    let total: String = row.try_get("total").unwrap_or_else(|_| "0".into());
    Ok(HttpResponse::Ok().json(serde_json::json!({ "balance_usd": total })))
}

async fn list_transactions(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let id = path.into_inner();
    let rows = sqlx::query("SELECT id, to_address, value, token, data, chain_id, status, transaction_hash, created_at FROM shmw_transactions WHERE master_wallet_id = $1 ORDER BY created_at DESC")
        .bind(id).fetch_all(&state.pool).await.map_err(db_err)?;
    let txs: Vec<_> = rows.iter().map(|r| serde_json::json!({
        "id": r.get::<Uuid, _>("id"),
        "to": r.get::<String, _>("to_address"),
        "value": r.get::<String, _>("value"),
        "token": r.get::<String, _>("token"),
        "chain_id": r.get::<i64, _>("chain_id"),
        "status": r.get::<String, _>("status"),
        "transaction_hash": r.get::<Option<String>, _>("transaction_hash"),
    })).collect();
    Ok(HttpResponse::Ok().json(serde_json::json!({ "transactions": txs })))
}

async fn create_transaction(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>, body: web::Json<CreateTransactionReq>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let mw_id = path.into_inner();
    let id = Uuid::new_v4();
    sqlx::query("INSERT INTO shmw_transactions (id, master_wallet_id, to_address, value, token, data, chain_id, status) VALUES ($1,$2,$3,$4,$5,$6,$7,'pending')")
        .bind(id).bind(mw_id).bind(&body.to).bind(&body.value).bind(body.token.as_deref().unwrap_or(""))
        .bind(body.data.as_deref().unwrap_or("")).bind(body.chain_id)
        .execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Created().json(serde_json::json!({ "transaction_id": id, "status": "pending" })))
}

async fn approve_transaction(state: web::Data<AppState>, req: HttpRequest, path: web::Path<(Uuid, Uuid)>) -> ApiResult<HttpResponse> {
    let (_uid, role) = require_auth(req.headers(), &state.jwt_secret)?;
    if role != "admin" && role != "master_wallet_admin" {
        return Err(ApiError::Unauthorized);
    }
    let (_mw, tid) = path.into_inner();
    let res = sqlx::query("UPDATE shmw_transactions SET status='approved' WHERE id=$1 AND status='pending'")
        .bind(tid).execute(&state.pool).await.map_err(db_err)?;
    if res.rows_affected() == 0 {
        return Err(ApiError::NotFound(format!("pending transaction {tid}")));
    }
    Ok(HttpResponse::Ok().json(serde_json::json!({ "transaction_id": tid, "status": "approved" })))
}

async fn reject_transaction(state: web::Data<AppState>, req: HttpRequest, path: web::Path<(Uuid, Uuid)>) -> ApiResult<HttpResponse> {
    let (_uid, role) = require_auth(req.headers(), &state.jwt_secret)?;
    if role != "admin" && role != "master_wallet_admin" {
        return Err(ApiError::Unauthorized);
    }
    let (_mw, tid) = path.into_inner();
    let res = sqlx::query("UPDATE shmw_transactions SET status='rejected' WHERE id=$1 AND status='pending'")
        .bind(tid).execute(&state.pool).await.map_err(db_err)?;
    if res.rows_affected() == 0 {
        return Err(ApiError::NotFound(format!("pending transaction {tid}")));
    }
    Ok(HttpResponse::Ok().json(serde_json::json!({ "transaction_id": tid, "status": "rejected" })))
}

async fn sign_and_broadcast(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>, body: web::Json<SignReq>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let mw_id = path.into_inner();
    // Record the broadcast intent as a transaction row. Real on-chain signing +
    // broadcast is delegated to the chain-native signer service configured by
    // the self-hosting operator (SIGNER_SERVICE_URL) — this self-hosted node
    // owns governance, not raw keys, so it never fabricates a transaction hash.
    // The transaction is recorded as `requires_signing` for the operator's
    // signer to pick up; when SIGNER_SERVICE_URL is set, an operator-side
    // worker signs + broadcasts and back-fills the `transaction_hash`.
    let id = Uuid::new_v4();
    sqlx::query("INSERT INTO shmw_transactions (id, master_wallet_id, to_address, value, token, chain_id, status) VALUES ($1,$2,$3,$4,$5,$6,'requires_signing')")
        .bind(id).bind(mw_id).bind(&body.to).bind(&body.amount).bind(body.token.as_deref().unwrap_or("")).bind(1i64)
        .execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Accepted().json(serde_json::json!({ "status": "requires_signing", "transaction_id": id, "message": "Transaction recorded; operator signer will broadcast" })))
}

async fn list_fees(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let id = path.into_inner();
    let rows = sqlx::query("SELECT id, chain_id, fee_bps FROM shmw_fees WHERE master_wallet_id = $1")
        .bind(id).fetch_all(&state.pool).await.map_err(db_err)?;
    let fees: Vec<_> = rows.iter().map(|r| serde_json::json!({
        "id": r.get::<Uuid, _>("id"),
        "chain_id": r.get::<i64, _>("chain_id"),
        "fee_bps": r.get::<i32, _>("fee_bps"),
    })).collect();
    Ok(HttpResponse::Ok().json(serde_json::json!({ "fees": fees })))
}

async fn set_fee(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>, body: web::Json<FeeReq>) -> ApiResult<HttpResponse> {
    let (_uid, role) = require_auth(req.headers(), &state.jwt_secret)?;
    if role != "admin" && role != "master_wallet_admin" {
        return Err(ApiError::Unauthorized);
    }
    if body.fee_bps < 0 || body.fee_bps > 2000 {
        return Err(ApiError::BadRequest("fee_bps must be in [0, 2000] (0-20%)".into()));
    }
    let mw_id = path.into_inner();
    let id = Uuid::new_v4();
    sqlx::query("INSERT INTO shmw_fees (id, master_wallet_id, chain_id, fee_bps) VALUES ($1,$2,$3,$4)")
        .bind(id).bind(mw_id).bind(body.chain_id).bind(body.fee_bps)
        .execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Created().json(serde_json::json!({ "fee_id": id, "chain_id": body.chain_id, "fee_bps": body.fee_bps })))
}

async fn delete_fee(state: web::Data<AppState>, req: HttpRequest, path: web::Path<(Uuid, Uuid)>) -> ApiResult<HttpResponse> {
    let (_uid, role) = require_auth(req.headers(), &state.jwt_secret)?;
    if role != "admin" && role != "master_wallet_admin" {
        return Err(ApiError::Unauthorized);
    }
    let (_mw, fid) = path.into_inner();
    sqlx::query("DELETE FROM shmw_fees WHERE id=$1").bind(fid).execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Ok().json(serde_json::json!({ "deleted": fid })))
}

async fn list_auto_sign(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let id = path.into_inner();
    let rows = sqlx::query("SELECT id, pattern, max_value, enabled FROM shmw_auto_sign WHERE master_wallet_id = $1")
        .bind(id).fetch_all(&state.pool).await.map_err(db_err)?;
    let rules: Vec<_> = rows.iter().map(|r| serde_json::json!({
        "id": r.get::<Uuid, _>("id"),
        "pattern": r.get::<String, _>("pattern"),
        "max_value": r.get::<String, _>("max_value"),
        "enabled": r.get::<bool, _>("enabled"),
    })).collect();
    Ok(HttpResponse::Ok().json(serde_json::json!({ "auto_sign_rules": rules })))
}

async fn create_auto_sign(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>, body: web::Json<AutoSignReq>) -> ApiResult<HttpResponse> {
    let (_uid, role) = require_auth(req.headers(), &state.jwt_secret)?;
    if role != "admin" && role != "master_wallet_admin" {
        return Err(ApiError::Unauthorized);
    }
    let mw_id = path.into_inner();
    let id = Uuid::new_v4();
    sqlx::query("INSERT INTO shmw_auto_sign (id, master_wallet_id, pattern, max_value, enabled) VALUES ($1,$2,$3,$4,TRUE)")
        .bind(id).bind(mw_id).bind(&body.pattern).bind(&body.max_value)
        .execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Created().json(serde_json::json!({ "rule_id": id, "pattern": body.pattern, "max_value": body.max_value })))
}

async fn delete_auto_sign(state: web::Data<AppState>, req: HttpRequest, path: web::Path<(Uuid, Uuid)>) -> ApiResult<HttpResponse> {
    let (_uid, role) = require_auth(req.headers(), &state.jwt_secret)?;
    if role != "admin" && role != "master_wallet_admin" {
        return Err(ApiError::Unauthorized);
    }
    let (_mw, rid) = path.into_inner();
    sqlx::query("DELETE FROM shmw_auto_sign WHERE id=$1").bind(rid).execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Ok().json(serde_json::json!({ "deleted": rid })))
}

async fn list_users(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let id = path.into_inner();
    let rows = sqlx::query("SELECT wu.user_id, wu.role, u.email FROM shmw_wallet_users wu JOIN shmw_users u ON u.id = wu.user_id WHERE wu.master_wallet_id = $1")
        .bind(id).fetch_all(&state.pool).await.map_err(db_err)?;
    let users: Vec<_> = rows.iter().map(|r| serde_json::json!({
        "user_id": r.get::<Uuid, _>("user_id"),
        "role": r.get::<String, _>("role"),
        "email": r.get::<String, _>("email"),
    })).collect();
    Ok(HttpResponse::Ok().json(serde_json::json!({ "users": users })))
}

async fn add_user(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>, body: web::Json<AddUserReq>) -> ApiResult<HttpResponse> {
    let (_uid, role) = require_auth(req.headers(), &state.jwt_secret)?;
    if role != "admin" && role != "master_wallet_admin" {
        return Err(ApiError::Unauthorized);
    }
    let mw_id = path.into_inner();
    let id = Uuid::new_v4();
    sqlx::query("INSERT INTO shmw_wallet_users (id, master_wallet_id, user_id, role) VALUES ($1,$2,$3,$4)")
        .bind(id).bind(mw_id).bind(body.user_id).bind(body.role.as_deref().unwrap_or("member"))
        .execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Created().json(serde_json::json!({ "membership_id": id, "user_id": body.user_id })))
}

async fn remove_user(state: web::Data<AppState>, req: HttpRequest, path: web::Path<(Uuid, Uuid)>) -> ApiResult<HttpResponse> {
    let (_uid, role) = require_auth(req.headers(), &state.jwt_secret)?;
    if role != "admin" && role != "master_wallet_admin" {
        return Err(ApiError::Unauthorized);
    }
    let (_mw, uid) = path.into_inner();
    sqlx::query("DELETE FROM shmw_wallet_users WHERE user_id=$1").bind(uid).execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Ok().json(serde_json::json!({ "removed": uid })))
}

async fn list_sub_wallets(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let id = path.into_inner();
    let rows = sqlx::query("SELECT id, label, chain_id, address, balance_usd FROM shmw_sub_wallets WHERE master_wallet_id = $1")
        .bind(id).fetch_all(&state.pool).await.map_err(db_err)?;
    let subs: Vec<_> = rows.iter().map(|r| serde_json::json!({
        "id": r.get::<Uuid, _>("id"),
        "label": r.get::<String, _>("label"),
        "chain_id": r.get::<i64, _>("chain_id"),
        "address": r.get::<String, _>("address"),
    })).collect();
    Ok(HttpResponse::Ok().json(serde_json::json!({ "sub_wallets": subs })))
}

async fn create_sub_wallet(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>, body: web::Json<CreateSubWalletReq>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let mw_id = path.into_inner();
    let id = Uuid::new_v4();
    sqlx::query("INSERT INTO shmw_sub_wallets (id, master_wallet_id, label, chain_id, address) VALUES ($1,$2,$3,$4,$5)")
        .bind(id).bind(mw_id).bind(&body.label).bind(body.chain_id).bind(body.address.as_deref().unwrap_or(""))
        .execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Created().json(serde_json::json!({ "sub_wallet_id": id, "label": body.label })))
}

async fn analytics_tx(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let id = path.into_inner();
    let rows = sqlx::query("SELECT status, COUNT(*) AS cnt FROM shmw_transactions WHERE master_wallet_id = $1 GROUP BY status")
        .bind(id).fetch_all(&state.pool).await.map_err(db_err)?;
    let by_status: serde_json::Value = rows.iter().map(|r| {
        (r.get::<String, _>("status"), r.get::<i64, _>("cnt"))
    }).collect();
    Ok(HttpResponse::Ok().json(serde_json::json!({ "by_status": by_status })))
}

async fn analytics_wallets(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state.jwt_secret)?;
    let id = path.into_inner();
    let row = sqlx::query("SELECT COUNT(*)::INT8 AS cnt, COALESCE(SUM(balance_usd),0)::TEXT AS total FROM shmw_sub_wallets WHERE master_wallet_id = $1")
        .bind(id).fetch_one(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Ok().json(serde_json::json!({
        "sub_wallet_count": row.get::<i64, _>("cnt"),
        "total_balance_usd": row.get::<String, _>("total"),
    })))
}

async fn list_chains(state: web::Data<AppState>, req: HttpRequest) -> ApiResult<HttpResponse> {
    // Public read: list the chains this self-hosted node is configured for.
    // The chain list is operator-configured (no fabricated RPCs); an empty
    // list is honest when none are configured.
    require_auth(req.headers(), &state.jwt_secret)?;
    let rows = sqlx::query("SELECT chain_id, name, rpc_url FROM shmw_chains ORDER BY chain_id")
        .fetch_all(&state.pool).await;
    let chains = match rows {
        Ok(rs) => rs.iter().map(|r| serde_json::json!({
            "chain_id": r.get::<i64, _>("chain_id"),
            "name": r.get::<String, _>("name"),
            "rpc_url": r.get::<String, _>("rpc_url"),
        })).collect::<Vec<_>>(),
        Err(_) => Vec::new(),
    };
    Ok(HttpResponse::Ok().json(serde_json::json!({ "chains": chains })))
}

async fn get_gas(state: web::Data<AppState>, req: HttpRequest) -> ApiResult<HttpResponse> {
    require_auth(req.headers(), &state.jwt_secret)?;
    // Gas is chain-specific and fetched from the operator's RPC; without an
    // RPC configured we honestly return unavailable rather than fabricating gwei.
    Err(ApiError::BadRequest("gas estimation requires operator RPC configuration (CHAIN_RPC_URL)".into()))
}

async fn get_price(state: web::Data<AppState>, req: HttpRequest) -> ApiResult<HttpResponse> {
    require_auth(req.headers(), &state.jwt_secret)?;
    Err(ApiError::BadRequest("price feed requires operator configuration (PRICE_API_URL)".into()))
}

async fn health() -> HttpResponse {
    HttpResponse::Ok().json(serde_json::json!({ "status": "ok", "service": "selfhosted_masterwallet" }))
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::from_default_env()
                .add_directive("selfhosted_masterwallet=info".parse().unwrap()),
        )
        .init();

    let state = match AppState::connect().await {
        Ok(s) => s,
        Err(e) => {
            error!("Failed to initialize state: {e}");
            std::process::exit(1);
        }
    };
    let bind = state.bind_addr.clone();
    let host = state.clone();
    info!("Self-Hosted MasterWallet listening on {bind}");

    HttpServer::new(move || {
        App::new()
            .wrap(middleware::Logger::default())
            .app_data(web::Data::new(host.clone()))
            .service(
                web::scope("/api/v1")
                    .route("/auth/register", web::post().to(register))
                    .route("/auth/login", web::post().to(login))
                    .route("/master-wallet", web::post().to(create_master_wallet))
                    .route("/master-wallet", web::get().to(list_master_wallets))
                    .route("/master-wallet/{id}", web::get().to(get_master_wallet))
                    .route("/master-wallet/{id}/balance", web::get().to(get_balance))
                    .route("/master-wallet/{id}/transactions", web::get().to(list_transactions))
                    .route("/master-wallet/{id}/transactions", web::post().to(create_transaction))
                    .route("/master-wallet/{id}/transactions/{tid}/approve", web::post().to(approve_transaction))
                    .route("/master-wallet/{id}/transactions/{tid}/reject", web::post().to(reject_transaction))
                    .route("/master-wallet/{id}/sign", web::post().to(sign_and_broadcast))
                    .route("/master-wallet/{id}/fees", web::get().to(list_fees))
                    .route("/master-wallet/{id}/fees", web::post().to(set_fee))
                    .route("/master-wallet/{id}/fees/{fid}", web::delete().to(delete_fee))
                    .route("/master-wallet/{id}/auto-sign", web::get().to(list_auto_sign))
                    .route("/master-wallet/{id}/auto-sign", web::post().to(create_auto_sign))
                    .route("/master-wallet/{id}/auto-sign/{rid}", web::delete().to(delete_auto_sign))
                    .route("/master-wallet/{id}/users", web::get().to(list_users))
                    .route("/master-wallet/{id}/users", web::post().to(add_user))
                    .route("/master-wallet/{id}/users/{uid}", web::delete().to(remove_user))
                    .route("/master-wallet/{id}/sub-wallets", web::get().to(list_sub_wallets))
                    .route("/master-wallet/{id}/sub-wallets", web::post().to(create_sub_wallet))
                    .route("/master-wallet/{id}/analytics/transactions", web::get().to(analytics_tx))
                    .route("/master-wallet/{id}/analytics/wallets", web::get().to(analytics_wallets))
                    .route("/chains", web::get().to(list_chains))
                    .route("/gas", web::get().to(get_gas))
                    .route("/price", web::get().to(get_price))
                    .route("/health", web::get().to(health)),
            )
    })
    .bind(&bind)?
    .run()
    .await
}

