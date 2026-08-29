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
use k256::elliptic_curve::sec1::ToEncodedPoint;
use rand::RngCore;
use serde::Deserialize;
use sha2::{Digest as _, Sha256};
use sqlx::{postgres::PgPoolOptions, PgPool, Row};
use std::env;
use std::sync::Arc;
use std::time::Duration as StdDuration;
use thiserror::Error;
use tracing::{error, info};
use uuid::Uuid;

mod auto_signer;
mod chains_data;
mod crypto;
mod evm_tx;
mod license_gate;
mod multisig;
mod non_evm;
mod rlp;

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
    #[error("product not authorized: {0}")]
    ServiceUnavailable(String),
}

impl actix_web::ResponseError for ApiError {
    fn status_code(&self) -> actix_web::http::StatusCode {
        match self {
            ApiError::NotFound(_) => actix_web::http::StatusCode::NOT_FOUND,
            ApiError::Unauthorized => actix_web::http::StatusCode::UNAUTHORIZED,
            ApiError::BadRequest(_) => actix_web::http::StatusCode::BAD_REQUEST,
            ApiError::Internal(_) => actix_web::http::StatusCode::INTERNAL_SERVER_ERROR,
            ApiError::ServiceUnavailable(_) => actix_web::http::StatusCode::SERVICE_UNAVAILABLE,
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

/// Shared application state: DB pool + JWT secret + license gate.
#[derive(Clone)]
pub struct AppState {
    pub pool: PgPool,
    pub jwt_secret: String,
    pub bind_addr: String,
    pub gate: Arc<license_gate::LicenseGate>,
}

impl AppState {
    async fn connect() -> Result<Self, Box<dyn std::error::Error>> {
        let db_url = env::var("DATABASE_URL").unwrap_or_else(|_| {
            "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet".into()
        });
        let pool = PgPoolOptions::new().max_connections(10).connect(&db_url).await?;
        run_migrations(&pool).await?;
        seed_chains(&pool).await?;
        // Fail-closed: no dev fallback. A missing JWT secret must prevent boot.
        let jwt_secret = match env::var("JWT_SECRET") {
            Ok(v) if !v.is_empty() => v,
            _ => {
                error!("JWT_SECRET environment variable must be set (fail-closed)");
                std::process::exit(1);
            }
        };
        let bind_addr = env::var("BIND_ADDR").unwrap_or_else(|_| "0.0.0.0:8470".into());
        let gate = Arc::new(license_gate::LicenseGate::new());
        Ok(Self { pool, jwt_secret, bind_addr, gate })
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

pub fn require_auth(headers: &HeaderMap, state: &AppState) -> ApiResult<(Uuid, String)> {
    // Fail-closed license gate: no protected request is served unless the
    // SuperAdmin control plane has validated the product license on a recent
    // heartbeat. This mirrors wl_shared/go/wlgate (503 when dead).
    if !state.gate.is_alive() {
        return Err(ApiError::ServiceUnavailable(format!(
            "product is not authorized to serve (license suspended/revoked or heartbeat stale): {}",
            state.gate.reason()
        )));
    }
    let auth = headers
        .get("authorization")
        .and_then(|v| v.to_str().ok())
        .ok_or(ApiError::Unauthorized)?;
    let token = auth.strip_prefix("Bearer ").ok_or(ApiError::Unauthorized)?;
    let claims = jwt_verify(&state.jwt_secret, token)?;
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

    // Canonical MasterWallet columns (mirrors master_wallets in the Go backend).
    for stmt in [
        "ALTER TABLE shmw_master_wallets ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT ''",
        "ALTER TABLE shmw_master_wallets ADD COLUMN IF NOT EXISTS blockchain TEXT NOT NULL DEFAULT 'ethereum'",
        "ALTER TABLE shmw_master_wallets ADD COLUMN IF NOT EXISTS public_key TEXT NOT NULL DEFAULT ''",
        "ALTER TABLE shmw_master_wallets ADD COLUMN IF NOT EXISTS wallet_type TEXT NOT NULL DEFAULT 'hot'",
        "ALTER TABLE shmw_master_wallets ADD COLUMN IF NOT EXISTS encrypted_seed TEXT NOT NULL DEFAULT ''",
        "ALTER TABLE shmw_master_wallets ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE",
    ] {
        sqlx::query(stmt).execute(pool).await?;
    }

    // Canonical chain registry (seeded from the embedded registry data).
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_user_chains_evm (
            chain_id BIGINT PRIMARY KEY,
            name TEXT NOT NULL,
            symbol TEXT NOT NULL DEFAULT '',
            rpc_url TEXT NOT NULL DEFAULT '',
            explorer_url TEXT NOT NULL DEFAULT '',
            decimals INT NOT NULL DEFAULT 18,
            derivation_path TEXT NOT NULL DEFAULT 'm/44''/60''/0''/0/0'
        )"#,
    ).execute(pool).await?;
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_user_chains_nonevm (
            chain_id BIGINT PRIMARY KEY,
            name TEXT NOT NULL,
            symbol TEXT NOT NULL DEFAULT '',
            chain_type TEXT NOT NULL DEFAULT '',
            rpc_url TEXT NOT NULL DEFAULT '',
            explorer_url TEXT NOT NULL DEFAULT '',
            decimals INT NOT NULL DEFAULT 18,
            derivation_path TEXT NOT NULL DEFAULT '',
            address_prefix TEXT NOT NULL DEFAULT ''
        )"#,
    ).execute(pool).await?;

    // Multisig: threshold wallets + pending transactions with owner signatures.
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_multisig_wallets (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            chain_id BIGINT NOT NULL,
            owners TEXT[] NOT NULL,
            threshold INT NOT NULL,
            nonce BIGINT NOT NULL DEFAULT 0,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
    ).execute(pool).await?;
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_multisig_transactions (
            id TEXT PRIMARY KEY,
            wallet_id TEXT NOT NULL REFERENCES shmw_multisig_wallets(id) ON DELETE CASCADE,
            to_address TEXT NOT NULL,
            value_wei TEXT NOT NULL,
            data TEXT NOT NULL DEFAULT '',
            nonce BIGINT NOT NULL,
            signatures JSONB NOT NULL DEFAULT '[]'::jsonb,
            threshold INT NOT NULL,
            status TEXT NOT NULL DEFAULT 'pending',
            tx_hash TEXT,
            chain_id BIGINT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
    ).execute(pool).await?;

    // Deterministic derived addresses per (wallet, chain, account index).
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS shmw_user_wallet_addresses (
            id UUID PRIMARY KEY,
            master_wallet_id UUID NOT NULL REFERENCES shmw_master_wallets(id) ON DELETE CASCADE,
            seed_hash TEXT NOT NULL,
            chain_id BIGINT NOT NULL,
            chain_type TEXT NOT NULL,
            address TEXT NOT NULL,
            derivation_path TEXT NOT NULL,
            account_index INT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            UNIQUE (master_wallet_id, seed_hash, chain_id, account_index)
        )"#,
    ).execute(pool).await?;
    Ok(())
}

// ---- Chain registry seeding + RPC resolution (canonical, fail-closed) ----

/// SeedChains — idempotently insert the canonical chain registry (120 EVM +
/// 66 non-EVM mainnet chains). Mirrors chain_seeding.go: ON CONFLICT DO
/// NOTHING so user edits to rpc_url/explorer_url are never overwritten.
async fn seed_chains(pool: &PgPool) -> Result<(), sqlx::Error> {
    let mut evm_count = 0u32;
    for c in chains_data::EVM_CHAINS {
        sqlx::query(
            "INSERT INTO shmw_user_chains_evm (chain_id, name, symbol, rpc_url, explorer_url, decimals, derivation_path) \
             VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (chain_id) DO NOTHING",
        )
        .bind(c.chain_id).bind(c.name).bind(c.symbol).bind(c.rpc_url)
        .bind(c.explorer_url).bind(c.decimals as i32).bind(c.derivation_path)
        .execute(pool).await?;
        evm_count += 1;
    }
    let mut non_evm_count = 0u32;
    for c in chains_data::NON_EVM_CHAINS {
        sqlx::query(
            "INSERT INTO shmw_user_chains_nonevm (chain_id, name, symbol, chain_type, rpc_url, explorer_url, decimals, derivation_path, address_prefix) \
             VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (chain_id) DO NOTHING",
        )
        .bind(c.chain_id).bind(c.name).bind(c.symbol).bind(c.chain_type).bind(c.rpc_url)
        .bind(c.explorer_url).bind(c.decimals as i32).bind(c.derivation_path).bind(c.address_prefix)
        .execute(pool).await?;
        non_evm_count += 1;
    }
    info!("chain registry seeded: {evm_count} EVM + {non_evm_count} non-EVM mainnet chains");
    Ok(())
}

/// chainRPCEndpoint — resolve the RPC URL for a chain id from env vars only
/// (no fabricated endpoints). Mirrors the canonical env-var mapping, with a
/// generic CHAIN_RPC_URL_<chain_id> fallback for the rest of the registry.
pub fn chain_rpc_endpoint(chain_id: i64) -> String {
    let var = match chain_id {
        1 => Some("ETH_RPC_URL"),
        56 => Some("BSC_RPC_URL"),
        137 => Some("POLYGON_RPC_URL"),
        42161 => Some("ARBITRUM_RPC_URL"),
        10 => Some("OPTIMISM_RPC_URL"),
        43114 => Some("AVALANCHE_RPC_URL"),
        8453 => Some("BASE_RPC_URL"),
        42220 => Some("CELO_RPC_URL"),
        250 => Some("FANTOM_RPC_URL"),
        25 => Some("CRONOS_RPC_URL"),
        59144 => Some("LINEA_RPC_URL"),
        534352 => Some("SCROLL_RPC_URL"),
        11155111 => Some("ETH_SEPOLIA_RPC_URL"),
        _ => None,
    };
    if let Some(v) = var {
        if let Ok(url) = env::var(v) {
            if !url.trim().is_empty() {
                return url;
            }
        }
    }
    env::var(format!("CHAIN_RPC_URL_{chain_id}")).unwrap_or_default()
}

/// Canonical curated chain metadata (mirrors chains.go supportedChains):
/// chain_id → (blockchain name, decimals). Falls back to the seeded registry.
pub fn canonical_chain(chain_id: i64) -> Option<(String, u32)> {
    let curated: Option<&'static str> = match chain_id {
        1 => Some("ethereum"),
        56 => Some("bsc"),
        137 => Some("polygon"),
        42161 => Some("arbitrum"),
        10 => Some("optimism"),
        43114 => Some("avalanche"),
        8453 => Some("base"),
        42220 => Some("celo"),
        250 => Some("fantom"),
        25 => Some("cronos"),
        59144 => Some("linea"),
        534352 => Some("scroll"),
        11155111 => Some("sepolia"),
        _ => None,
    };
    if let Some(name) = curated {
        return Some((name.to_string(), 18));
    }
    chains_data::EVM_CHAINS
        .iter()
        .find(|c| c.chain_id == chain_id)
        .map(|c| (c.symbol.to_lowercase(), c.decimals as u32))
}

/// bech32PrefixForChainID — per-chain Cosmos-SDK bech32 account prefix
/// (mirrors chain_seeding.go). Falls back to "cosmos".
fn bech32_prefix_for_chain_id(chain_id: i64) -> &'static str {
    match chain_id {
        9000000118 => "cosmos",
        9000026317 => "osmo",
        9000000330 => "terra",
        9000073068 => "inj",
        9000014648 => "celestia",
        9000049823 => "dydx",
        9000073741 => "sei",
        9000041857 => "kujira",
        9000012099 => "stride",
        9000090063 => "neutron",
        9000005267 => "juno",
        9000007183 => "akash",
        9000018759 => "persistence",
        9000034677 => "evmos",
        9000054841 => "canto",
        9000003318 => "kava",
        9000062954 => "cro",
        9000016892 => "stars",
        9000021252 => "saga",
        9000086660 => "noble",
        9000040572 => "axelar",
        9000007153 => "umee",
        9000000529 => "secret",
        _ => "cosmos",
    }
}

// ---- Request/response DTOs ----

#[derive(Deserialize)]
struct RegisterReq { email: String, username: String, password: String }

#[derive(Deserialize)]
struct LoginReq { email: String, password: String }

#[derive(Deserialize)]
struct CreateMasterWalletReq {
    name: Option<String>,
    label: Option<String>,
    password: Option<String>,
    mnemonic: Option<String>,
    chain_id: Option<i64>,
    wallet_type: Option<String>,
    address: Option<String>,
}

#[derive(Deserialize)]
struct UpdateMasterWalletReq { name: Option<String>, wallet_type: Option<String>, is_active: Option<bool> }

#[derive(Deserialize)]
struct CreateTransactionReq { to: String, value: String, token: Option<String>, data: Option<String>, chain_id: i64 }

/// Canonical sendReq (mirrors handlers.go): human-readable amount, optional
/// ERC-20 token contract, password to decrypt the stored seed.
#[derive(Deserialize)]
struct SignReq {
    to: String,
    amount: String,
    token: Option<String>,
    password: String,
    gas_limit: Option<u64>,
    withdrawal_id: Option<String>,
}

#[derive(Deserialize)]
struct SignMessageReq {
    password: String,
    chain_type: Option<String>,
    message: String,
    account_index: Option<u32>,
    derivation_path: Option<String>,
}

#[derive(Deserialize)]
struct DeriveUserAddressReq {
    password: String,
    chain_id: i64,
    chain_type: String,
    account_index: Option<u32>,
}

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

/// CreateMasterWallet — POST /api/v1/master-wallet (canonical contract).
///
/// Generates a real BIP-39 24-word mnemonic (256-bit entropy) unless one is
/// supplied (validated, never logged), derives the EVM address at
/// m/44'/60'/0'/0/0, encrypts the seed with scrypt+AES-256-GCM under the
/// operator's password, and returns the mnemonic ONCE.
async fn create_master_wallet(state: web::Data<AppState>, req: HttpRequest, body: web::Json<CreateMasterWalletReq>) -> ApiResult<HttpResponse> {
    let (uid, _role) = require_auth(req.headers(), &state)?;
    let name = body
        .name
        .clone()
        .or_else(|| body.label.clone())
        .unwrap_or_default();
    if name.trim().is_empty() {
        return Err(ApiError::BadRequest("name required".into()));
    }
    let password = body.password.clone().unwrap_or_default();
    if password.len() < 8 {
        return Err(ApiError::BadRequest("password required (min 8 chars) to encrypt the wallet seed".into()));
    }
    let chain_id = body.chain_id.unwrap_or(1);
    let (blockchain, _decimals) = canonical_chain(chain_id)
        .ok_or_else(|| ApiError::BadRequest(format!("unsupported chain id {chain_id}")))?;
    let wallet_type = body.wallet_type.clone().unwrap_or_else(|| "hot".into());
    if !matches!(wallet_type.as_str(), "hot" | "cold" | "multisig") {
        return Err(ApiError::BadRequest("wallet_type must be hot, cold, or multisig".into()));
    }

    // Mnemonic: operator-supplied (validated) or freshly generated. Real BIP-39.
    let mnemonic = match body.mnemonic.as_deref() {
        Some(m) => {
            let m = m.trim().to_string();
            if !crypto::validate_mnemonic(&m) {
                return Err(ApiError::BadRequest("invalid BIP-39 mnemonic".into()));
            }
            m
        }
        None => crypto::generate_mnemonic(),
    };
    let seed = crypto::mnemonic_to_seed(&mnemonic);
    let priv_key = crypto::derive_evm_private_key(&seed, 0)
        .map_err(|e| ApiError::Internal(format!("key derivation failed: {e}")))?;
    let address = crypto::private_key_to_address(&priv_key)
        .map_err(|e| ApiError::Internal(format!("address derivation failed: {e}")))?;
    // Compressed public key (matches the canonical backend).
    let sk = k256::SecretKey::from_slice(&priv_key).map_err(|e| ApiError::Internal(e.to_string()))?;
    let public_key = hex::encode(sk.public_key().to_encoded_point(true).as_bytes());
    let enc_seed = crypto::encrypt_seed(&seed, &password)
        .map_err(|e| ApiError::Internal(format!("seed encryption failed: {e}")))?;

    let id = Uuid::new_v4();
    sqlx::query(
        "INSERT INTO shmw_master_wallets (id, owner_id, label, name, blockchain, address, public_key, wallet_type, chain_id, encrypted_seed) \
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)",
    )
    .bind(id).bind(uid).bind(name.trim()).bind(name.trim()).bind(&blockchain)
    .bind(&address).bind(&public_key).bind(&wallet_type).bind(chain_id).bind(&enc_seed)
    .execute(&state.pool).await.map_err(db_err)?;

    Ok(HttpResponse::Created().json(serde_json::json!({
        "wallet_id": id,
        "id": id,
        "name": name.trim(),
        "blockchain": blockchain,
        "address": address,
        "public_key": public_key,
        "wallet_type": wallet_type,
        "chain_id": chain_id,
        "mnemonic": mnemonic,
    })))
}

async fn list_master_wallets(state: web::Data<AppState>, req: HttpRequest) -> ApiResult<HttpResponse> {
    let (uid, _role) = require_auth(req.headers(), &state)?;
    let rows = sqlx::query(
        "SELECT id, name, blockchain, address, public_key, wallet_type, chain_id, is_active, created_at \
         FROM shmw_master_wallets WHERE owner_id = $1 ORDER BY created_at DESC",
    )
        .bind(uid)
        .fetch_all(&state.pool).await.map_err(db_err)?;
    let wallets: Vec<_> = rows.iter().map(|r| serde_json::json!({
        "id": r.get::<Uuid, _>("id"),
        "name": r.get::<String, _>("name"),
        "blockchain": r.get::<String, _>("blockchain"),
        "address": r.get::<String, _>("address"),
        "public_key": r.get::<String, _>("public_key"),
        "wallet_type": r.get::<String, _>("wallet_type"),
        "chain_id": r.get::<i64, _>("chain_id"),
        "is_active": r.get::<bool, _>("is_active"),
        "created_at": r.get::<chrono::DateTime<Utc>, _>("created_at"),
    })).collect();
    Ok(HttpResponse::Ok().json(serde_json::json!({ "wallets": wallets })))
}

async fn get_master_wallet(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
    let id = path.into_inner();
    let row = sqlx::query(
        "SELECT id, name, blockchain, address, public_key, wallet_type, chain_id, is_active, created_at \
         FROM shmw_master_wallets WHERE id = $1",
    )
        .bind(id).fetch_optional(&state.pool).await.map_err(db_err)?;
    let row = row.ok_or_else(|| ApiError::NotFound(format!("master wallet {id}")))?;
    Ok(HttpResponse::Ok().json(serde_json::json!({
        "id": row.get::<Uuid, _>("id"),
        "name": row.get::<String, _>("name"),
        "blockchain": row.get::<String, _>("blockchain"),
        "address": row.get::<String, _>("address"),
        "public_key": row.get::<String, _>("public_key"),
        "wallet_type": row.get::<String, _>("wallet_type"),
        "chain_id": row.get::<i64, _>("chain_id"),
        "is_active": row.get::<bool, _>("is_active"),
        "created_at": row.get::<chrono::DateTime<Utc>, _>("created_at"),
    })))
}

/// UpdateMasterWallet — PUT /api/v1/master-wallet/:id. Mutable metadata only
/// (name, wallet_type, is_active); address/seed/public_key are immutable.
async fn update_master_wallet(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>, body: web::Json<UpdateMasterWalletReq>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
    let id = path.into_inner();
    if body.name.is_none() && body.wallet_type.is_none() && body.is_active.is_none() {
        return Err(ApiError::BadRequest("no updatable fields provided".into()));
    }
    if let Some(n) = &body.name {
        if n.trim().is_empty() {
            return Err(ApiError::BadRequest("name cannot be empty".into()));
        }
        sqlx::query("UPDATE shmw_master_wallets SET name=$2, label=$2 WHERE id=$1")
            .bind(id).bind(n.trim()).execute(&state.pool).await.map_err(db_err)?;
    }
    if let Some(wt) = &body.wallet_type {
        if !matches!(wt.as_str(), "hot" | "cold" | "multisig") {
            return Err(ApiError::BadRequest("wallet_type must be hot, cold, or multisig".into()));
        }
        sqlx::query("UPDATE shmw_master_wallets SET wallet_type=$2 WHERE id=$1")
            .bind(id).bind(wt).execute(&state.pool).await.map_err(db_err)?;
    }
    if let Some(active) = body.is_active {
        sqlx::query("UPDATE shmw_master_wallets SET is_active=$2 WHERE id=$1")
            .bind(id).bind(active).execute(&state.pool).await.map_err(db_err)?;
    }
    Ok(HttpResponse::Ok().json(serde_json::json!({ "id": id, "updated": true })))
}

/// DeleteMasterWallet — DELETE /api/v1/master-wallet/:id.
async fn delete_master_wallet(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
    let id = path.into_inner();
    let res = sqlx::query("DELETE FROM shmw_master_wallets WHERE id=$1")
        .bind(id).execute(&state.pool).await.map_err(db_err)?;
    if res.rows_affected() == 0 {
        return Err(ApiError::NotFound(format!("master wallet {id}")));
    }
    Ok(HttpResponse::Ok().json(serde_json::json!({ "deleted": true, "id": id })))
}

/// GetMasterWalletBalance — LIVE native balance from the chain RPC.
/// Fail-closed: without a configured RPC the balance is reported as unknown.
async fn get_balance(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
    let id = path.into_inner();
    let row = sqlx::query("SELECT address, chain_id FROM shmw_master_wallets WHERE id = $1")
        .bind(id).fetch_optional(&state.pool).await.map_err(db_err)?;
    let row = row.ok_or_else(|| ApiError::NotFound(format!("master wallet {id}")))?;
    let address: String = row.get("address");
    let chain_id: i64 = row.get("chain_id");
    let rpc = chain_rpc_endpoint(chain_id);
    if rpc.is_empty() {
        return Err(ApiError::BadRequest(format!("RPC endpoint not configured for chain {chain_id}")));
    }
    let wei = evm_tx::rpc_get_balance(&rpc, &address)
        .await
        .map_err(|e| ApiError::BadRequest(format!("balance fetch failed: {e}")))?;
    let (_chain, decimals) = canonical_chain(chain_id).unwrap_or_else(|| ("ethereum".into(), 18));
    Ok(HttpResponse::Ok().json(serde_json::json!({
        "master_wallet_id": id,
        "address": address,
        "chain_id": chain_id,
        "balance_wei": wei,
        "balance": evm_tx::wei_to_human(&wei, decimals),
    })))
}

async fn list_transactions(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
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
    let (_uid, _role) = require_auth(req.headers(), &state)?;
    let mw_id = path.into_inner();
    let id = Uuid::new_v4();
    sqlx::query("INSERT INTO shmw_transactions (id, master_wallet_id, to_address, value, token, data, chain_id, status) VALUES ($1,$2,$3,$4,$5,$6,$7,'pending')")
        .bind(id).bind(mw_id).bind(&body.to).bind(&body.value).bind(body.token.as_deref().unwrap_or(""))
        .bind(body.data.as_deref().unwrap_or("")).bind(body.chain_id)
        .execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Created().json(serde_json::json!({ "transaction_id": id, "status": "pending" })))
}

async fn approve_transaction(state: web::Data<AppState>, req: HttpRequest, path: web::Path<(Uuid, Uuid)>) -> ApiResult<HttpResponse> {
    let (_uid, role) = require_auth(req.headers(), &state)?;
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
    let (_uid, role) = require_auth(req.headers(), &state)?;
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

/// SignTransaction — POST /api/v1/master-wallet/:id/sign (canonical contract).
///
/// Real in-process signing: decrypt the stored seed with the operator
/// password, derive m/44'/60'/0'/0/0, fetch nonce + EIP-1559 fees from the
/// chain RPC, sign a type-2 transaction, and broadcast it. Fail-closed:
/// no RPC → 503; bad password → 400; RPC failure → 502; never fabricates a
/// transaction hash.
async fn sign_and_broadcast(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>, body: web::Json<SignReq>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
    let mw_id = path.into_inner();

    // Two-party withdrawal gate: the self-hosted node has no license control
    // plane, so gated withdrawals fail closed honestly.
    if let Some(wid) = &body.withdrawal_id {
        if !wid.trim().is_empty() {
            return Ok(HttpResponse::Forbidden().json(serde_json::json!({
                "error": "two-party SuperAdmin collaboration required before withdrawal; self-hosted node has no license gate",
            })));
        }
    }
    if evm_tx::parse_hex_fixed::<20>(&body.to).is_err() {
        return Err(ApiError::BadRequest("invalid to address".into()));
    }

    let row = sqlx::query("SELECT address, encrypted_seed, chain_id FROM shmw_master_wallets WHERE id = $1")
        .bind(mw_id).fetch_optional(&state.pool).await.map_err(db_err)?;
    let row = row.ok_or_else(|| ApiError::NotFound("master wallet not found".into()))?;
    let from_addr: String = row.get("address");
    let enc_seed: String = row.get("encrypted_seed");
    let chain_id: i64 = row.get("chain_id");
    if enc_seed.is_empty() {
        return Err(ApiError::BadRequest("wallet has no managed seed (operator-supplied address only)".into()));
    }
    let seed = crypto::decrypt_seed(&enc_seed, &body.password)
        .map_err(|_| ApiError::BadRequest("invalid password (seed decryption failed)".into()))?;
    let priv_key = crypto::derive_evm_private_key(&seed, 0)
        .map_err(|e| ApiError::Internal(format!("key derivation failed: {e}")))?;

    let rpc = chain_rpc_endpoint(chain_id);
    if rpc.is_empty() {
        return Ok(HttpResponse::ServiceUnavailable().json(serde_json::json!({
            "error": format!("RPC endpoint not configured for chain {chain_id}"),
        })));
    }
    let (_chain, decimals) = canonical_chain(chain_id).unwrap_or_else(|| ("ethereum".into(), 18));

    let nonce = evm_tx::rpc_get_nonce(&rpc, &from_addr)
        .await
        .map_err(|e| ApiError::Internal(format!("fetch nonce: {e}")))?;
    let tip = evm_tx::rpc_max_priority_fee(&rpc)
        .await
        .unwrap_or_else(|_| "1000000000".into()); // 1 Gwei fallback (canonical)
    // Canonical: maxFee = tip + baseFee when the head block has one, else gasPrice.
    let max_fee = match evm_tx::rpc_call(&rpc, "eth_getBlockByNumber", serde_json::json!(["latest", false])).await {
        Ok(block) => block
            .get("baseFeePerGas")
            .and_then(|b| b.as_str())
            .and_then(|b| evm_tx::hex_quantity_to_dec(b).ok())
            .and_then(|base| add_dec(&base, &tip).ok()),
        Err(_) => None,
    };
    let max_fee = match max_fee {
        Some(f) => f,
        None => evm_tx::rpc_gas_price(&rpc)
            .await
            .map_err(|e| ApiError::Internal(format!("fetch gas price: {e}")))?,
    };

    let gas_limit = body.gas_limit.unwrap_or(21_000);
    let (to_addr, value_wei, data) = match body.token.as_deref().filter(|t| !t.trim().is_empty()) {
        None => {
            let wei = evm_tx::human_to_wei(&body.amount, decimals)
                .map_err(|_| ApiError::BadRequest("invalid amount".into()))?;
            (body.to.clone(), wei, Vec::new())
        }
        Some(token) => {
            if evm_tx::parse_hex_fixed::<20>(token).is_err() {
                return Err(ApiError::BadRequest("invalid token contract address".into()));
            }
            // ERC-20 transfer(to, amount) — token decimals default to 18 (canonical).
            let wei = evm_tx::human_to_wei(&body.amount, 18)
                .map_err(|_| ApiError::BadRequest("invalid amount".into()))?;
            let to_bytes = evm_tx::parse_hex_fixed::<20>(&body.to).unwrap();
            let amount_be = evm_tx::dec_to_be(&wei).map_err(|_| ApiError::BadRequest("invalid amount".into()))?;
            (token.to_string(), "0".to_string(), evm_tx::erc20_transfer_calldata(&to_bytes, &amount_be))
        }
    };

    let params = evm_tx::TxParams {
        chain_id: chain_id as u64,
        nonce,
        gas_limit,
        to: to_addr,
        value_wei,
        data,
        gas_price_wei: String::new(),
        max_priority_fee_wei: tip,
        max_fee_wei: max_fee,
        eip1559: true,
    };
    let signed = evm_tx::sign_transaction(&priv_key, &params)
        .map_err(|e| ApiError::Internal(format!("sign: {e}")))?;
    let tx_hash = evm_tx::rpc_send_raw_transaction(&rpc, &signed.raw)
        .await
        .map_err(|e| ApiError::Internal(format!("broadcast: {e}")))?;

    // Persist the transaction record (canonical response shape).
    let tx_rec = Uuid::new_v4();
    let _ = sqlx::query(
        "INSERT INTO shmw_transactions (id, master_wallet_id, to_address, value, token, data, chain_id, status, transaction_hash) \
         VALUES ($1,$2,$3,$4,$5,$6,$7,'pending',$8)",
    )
    .bind(tx_rec).bind(mw_id).bind(&body.to).bind(&body.amount)
    .bind(body.token.as_deref().unwrap_or("")).bind("").bind(chain_id).bind(&tx_hash)
    .execute(&state.pool).await;

    Ok(HttpResponse::Ok().json(serde_json::json!({
        "transaction_hash": tx_hash,
        "status": "broadcast",
        "from": from_addr,
        "chain_id": chain_id,
    })))
}

/// Add two decimal-string non-negative integers.
pub fn add_dec(a: &str, b: &str) -> Result<String, String> {
    let mut x = evm_tx::dec_to_be(a)?;
    let y = evm_tx::dec_to_be(b)?;
    if y.len() > x.len() {
        x = [vec![0u8; y.len() - x.len()], x].concat();
    }
    let mut out = vec![0u8; x.len().max(y.len())];
    let ypad = [vec![0u8; out.len() - y.len()], y].concat();
    let mut carry = 0u16;
    for i in (0..out.len()).rev() {
        let cur = x[i] as u16 + ypad[i] as u16 + carry;
        out[i] = (cur & 0xff) as u8;
        carry = cur >> 8;
    }
    let mut full = if carry > 0 { vec![carry as u8] } else { Vec::new() };
    full.extend_from_slice(&out);
    evm_tx::hex_quantity_to_dec(&format!("0x{}", hex::encode(full)))
}

/// SignMessage — POST /api/v1/master-wallet/:id/sign-message.
///
/// Real message signing with the wallet's decrypted seed: EIP-191
/// personal_sign for EVM, Ed25519 for Solana, Bitcoin signed-message format
/// for BTC, amino-style for Cosmos.
async fn sign_message(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>, body: web::Json<SignMessageReq>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
    let mw_id = path.into_inner();
    if body.message.is_empty() {
        return Err(ApiError::BadRequest("message required".into()));
    }
    let row = sqlx::query("SELECT encrypted_seed FROM shmw_master_wallets WHERE id = $1")
        .bind(mw_id).fetch_optional(&state.pool).await.map_err(db_err)?;
    let row = row.ok_or_else(|| ApiError::NotFound("master wallet not found".into()))?;
    let enc_seed: String = row.get("encrypted_seed");
    if enc_seed.is_empty() {
        return Err(ApiError::BadRequest("wallet has no managed seed (operator-supplied address only)".into()));
    }
    let seed = crypto::decrypt_seed(&enc_seed, &body.password)
        .map_err(|_| ApiError::BadRequest("invalid password (seed decryption failed)".into()))?;

    let chain_type = body.chain_type.as_deref().unwrap_or("evm").to_lowercase();
    let index = body.account_index.unwrap_or(0);
    let msg = body.message.as_bytes();
    let (address, signature, algorithm) = match chain_type.as_str() {
        "evm" => {
            let path = body
                .derivation_path
                .clone()
                .unwrap_or_else(|| crypto::derive_path_for_account(60, index));
            let key = crypto::derive_private_key_from_path(&seed, &path)
                .map_err(|e| ApiError::Internal(format!("key derivation failed: {e}")))?;
            let addr = crypto::private_key_to_address(&key)
                .map_err(|e| ApiError::Internal(format!("address derivation failed: {e}")))?;
            let sig = evm_tx::personal_sign(&key, msg)
                .map_err(|e| ApiError::Internal(format!("signing failed: {e}")))?;
            (addr, format!("0x{}", hex::encode(sig)), "eip191-ecdsa-secp256k1")
        }
        "solana" => {
            let path = body
                .derivation_path
                .clone()
                .unwrap_or_else(|| format!("m/44'/501'/{}'/0'", index));
            let addr = non_evm::solana_address_from_seed(&seed, &path)
                .map_err(|e| ApiError::Internal(format!("address derivation failed: {e}")))?;
            let sig = non_evm::solana_sign(&seed, &path, msg)
                .map_err(|e| ApiError::Internal(format!("signing failed: {e}")))?;
            (addr, non_evm::base58_encode(&sig), "ed25519")
        }
        "bitcoin" => {
            let path = body
                .derivation_path
                .clone()
                .unwrap_or_else(|| crypto::derive_path_for_account(0, index));
            let addr = non_evm::btc_address_from_seed(&seed, &path)
                .map_err(|e| ApiError::Internal(format!("address derivation failed: {e}")))?;
            let sig = non_evm::btc_sign(&seed, &path, msg)
                .map_err(|e| ApiError::Internal(format!("signing failed: {e}")))?;
            (addr, sig, "bitcoin-signed-message")
        }
        "cosmos" => {
            let path = body
                .derivation_path
                .clone()
                .unwrap_or_else(|| crypto::derive_path_for_account(118, index));
            let addr = non_evm::cosmos_address_from_seed(&seed, &path, "cosmos")
                .map_err(|e| ApiError::Internal(format!("address derivation failed: {e}")))?;
            let key = crypto::derive_private_key_from_path(&seed, &path)
                .map_err(|e| ApiError::Internal(format!("key derivation failed: {e}")))?;
            let digest: [u8; 32] = sha2::Sha256::digest(msg).into();
            let sk = k256::ecdsa::SigningKey::from_slice(&key).map_err(|e| ApiError::Internal(e.to_string()))?;
            let (sig, recid) = sk.sign_prehash_recoverable(&digest).map_err(|e| ApiError::Internal(e.to_string()))?;
            let mut sig65 = sig.to_bytes().to_vec();
            sig65.push(recid.is_y_odd() as u8);
            (addr, hex::encode(sig65), "secp256k1-sha256")
        }
        other => {
            return Err(ApiError::BadRequest(format!("unsupported chain_type: {other}")));
        }
    };
    Ok(HttpResponse::Ok().json(serde_json::json!({
        "address": address,
        "chain_type": chain_type,
        "signature": signature,
        "algorithm": algorithm,
    })))
}

/// DeriveUserAddress — POST /api/v1/master-wallet/:id/derive-user-address
/// (canonical contract). Deterministic per (seed, chain, account_index);
/// persisted for idempotent re-derivation.
async fn derive_user_address(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>, body: web::Json<DeriveUserAddressReq>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
    let mw_id = path.into_inner();
    if body.chain_id <= 0 {
        return Err(ApiError::BadRequest("chain_id must be positive".into()));
    }
    let chain_type = body.chain_type.to_lowercase();
    let account_index = body.account_index.unwrap_or(0);
    let row = sqlx::query("SELECT encrypted_seed FROM shmw_master_wallets WHERE id = $1")
        .bind(mw_id).fetch_optional(&state.pool).await.map_err(db_err)?;
    let row = row.ok_or_else(|| ApiError::NotFound("master wallet not found".into()))?;
    let enc_seed: String = row.get("encrypted_seed");
    if enc_seed.is_empty() {
        return Err(ApiError::BadRequest("wallet has no managed seed (operator-supplied address only)".into()));
    }
    let seed = crypto::decrypt_seed(&enc_seed, &body.password)
        .map_err(|_| ApiError::BadRequest("invalid password (seed decryption failed)".into()))?;

    let (address, derivation_path) = match chain_type.as_str() {
        "evm" => {
            let path = crypto::derive_path_for_account(60, account_index);
            let key = crypto::derive_private_key_from_path(&seed, &path)
                .map_err(|e| ApiError::Internal(format!("evm derivation failed: {e}")))?;
            let addr = crypto::private_key_to_address(&key)
                .map_err(|e| ApiError::Internal(format!("evm address failed: {e}")))?;
            (addr, path)
        }
        "solana" => {
            let path = format!("m/44'/501'/{account_index}'/0'");
            let addr = non_evm::solana_address_from_seed(&seed, &path)
                .map_err(|e| ApiError::Internal(format!("solana derivation failed: {e}")))?;
            (addr, path)
        }
        "bitcoin" => {
            let path = crypto::derive_path_for_account(0, account_index);
            let addr = non_evm::btc_address_from_seed(&seed, &path)
                .map_err(|e| ApiError::Internal(format!("bitcoin derivation failed: {e}")))?;
            (addr, path)
        }
        "cosmos" => {
            let prefix = bech32_prefix_for_chain_id(body.chain_id);
            let path = crypto::derive_path_for_account(118, account_index);
            let addr = non_evm::cosmos_address_from_seed(&seed, &path, prefix)
                .map_err(|e| ApiError::Internal(format!("cosmos derivation failed: {e}")))?;
            (addr, path)
        }
        other => return Err(ApiError::BadRequest(format!("unsupported chain_type: {other}"))),
    };

    let seed_hash = hex::encode(sha2::Sha256::digest(&seed));
    sqlx::query(
        "INSERT INTO shmw_user_wallet_addresses (id, master_wallet_id, seed_hash, chain_id, chain_type, address, derivation_path, account_index) \
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8) \
         ON CONFLICT (master_wallet_id, seed_hash, chain_id, account_index) DO NOTHING",
    )
    .bind(Uuid::new_v4()).bind(mw_id).bind(&seed_hash).bind(body.chain_id)
    .bind(&chain_type).bind(&address).bind(&derivation_path).bind(account_index as i32)
    .execute(&state.pool).await.map_err(db_err)?;

    Ok(HttpResponse::Ok().json(serde_json::json!({
        "address": address,
        "chain_id": body.chain_id,
        "chain_type": chain_type,
        "derivation_path": derivation_path,
        "account_index": account_index,
    })))
}

/// ListUserWalletAddresses — GET /api/v1/master-wallet/:id/user-wallet-addresses.
async fn list_user_wallet_addresses(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
    let mw_id = path.into_inner();
    let rows = sqlx::query(
        "SELECT chain_id, chain_type, address, derivation_path, account_index, created_at \
         FROM shmw_user_wallet_addresses WHERE master_wallet_id=$1 ORDER BY chain_id, account_index LIMIT 1000",
    )
    .bind(mw_id).fetch_all(&state.pool).await.map_err(db_err)?;
    let addrs: Vec<_> = rows.iter().map(|r| serde_json::json!({
        "chain_id": r.get::<i64, _>("chain_id"),
        "chain_type": r.get::<String, _>("chain_type"),
        "address": r.get::<String, _>("address"),
        "derivation_path": r.get::<String, _>("derivation_path"),
        "account_index": r.get::<i32, _>("account_index"),
        "created_at": r.get::<chrono::DateTime<Utc>, _>("created_at"),
    })).collect();
    let count = addrs.len();
    Ok(HttpResponse::Ok().json(serde_json::json!({ "addresses": addrs, "count": count })))
}

async fn list_fees(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
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
    let (_uid, role) = require_auth(req.headers(), &state)?;
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
    let (_uid, role) = require_auth(req.headers(), &state)?;
    if role != "admin" && role != "master_wallet_admin" {
        return Err(ApiError::Unauthorized);
    }
    let (_mw, fid) = path.into_inner();
    sqlx::query("DELETE FROM shmw_fees WHERE id=$1").bind(fid).execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Ok().json(serde_json::json!({ "deleted": fid })))
}

async fn list_auto_sign(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
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
    let (_uid, role) = require_auth(req.headers(), &state)?;
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
    let (_uid, role) = require_auth(req.headers(), &state)?;
    if role != "admin" && role != "master_wallet_admin" {
        return Err(ApiError::Unauthorized);
    }
    let (_mw, rid) = path.into_inner();
    sqlx::query("DELETE FROM shmw_auto_sign WHERE id=$1").bind(rid).execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Ok().json(serde_json::json!({ "deleted": rid })))
}

async fn list_users(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
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
    let (_uid, role) = require_auth(req.headers(), &state)?;
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
    let (_uid, role) = require_auth(req.headers(), &state)?;
    if role != "admin" && role != "master_wallet_admin" {
        return Err(ApiError::Unauthorized);
    }
    let (_mw, uid) = path.into_inner();
    sqlx::query("DELETE FROM shmw_wallet_users WHERE user_id=$1").bind(uid).execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Ok().json(serde_json::json!({ "removed": uid })))
}

async fn list_sub_wallets(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
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
    let (_uid, _role) = require_auth(req.headers(), &state)?;
    let mw_id = path.into_inner();
    let id = Uuid::new_v4();
    sqlx::query("INSERT INTO shmw_sub_wallets (id, master_wallet_id, label, chain_id, address) VALUES ($1,$2,$3,$4,$5)")
        .bind(id).bind(mw_id).bind(&body.label).bind(body.chain_id).bind(body.address.as_deref().unwrap_or(""))
        .execute(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Created().json(serde_json::json!({ "sub_wallet_id": id, "label": body.label })))
}

async fn analytics_tx(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
    let id = path.into_inner();
    let rows = sqlx::query("SELECT status, COUNT(*) AS cnt FROM shmw_transactions WHERE master_wallet_id = $1 GROUP BY status")
        .bind(id).fetch_all(&state.pool).await.map_err(db_err)?;
    let by_status: serde_json::Value = rows.iter().map(|r| {
        (r.get::<String, _>("status"), r.get::<i64, _>("cnt"))
    }).collect();
    Ok(HttpResponse::Ok().json(serde_json::json!({ "by_status": by_status })))
}

async fn analytics_wallets(state: web::Data<AppState>, req: HttpRequest, path: web::Path<Uuid>) -> ApiResult<HttpResponse> {
    let (_uid, _role) = require_auth(req.headers(), &state)?;
    let id = path.into_inner();
    let row = sqlx::query("SELECT COUNT(*)::INT8 AS cnt, COALESCE(SUM(balance_usd),0)::TEXT AS total FROM shmw_sub_wallets WHERE master_wallet_id = $1")
        .bind(id).fetch_one(&state.pool).await.map_err(db_err)?;
    Ok(HttpResponse::Ok().json(serde_json::json!({
        "sub_wallet_count": row.get::<i64, _>("cnt"),
        "total_balance_usd": row.get::<String, _>("total"),
    })))
}

/// ListChains — GET /api/v1/chains. The canonical seeded registry: EVM and
/// non-EVM mainnet chains with metadata; RPC URLs resolve from env vars.
async fn list_chains(state: web::Data<AppState>, req: HttpRequest) -> ApiResult<HttpResponse> {
    require_auth(req.headers(), &state)?;
    let evm_rows = sqlx::query(
        "SELECT chain_id, name, symbol, rpc_url, explorer_url, decimals, derivation_path \
         FROM shmw_user_chains_evm ORDER BY chain_id",
    )
        .fetch_all(&state.pool).await.map_err(db_err)?;
    let mut chains: Vec<_> = evm_rows.iter().map(|r| {
        let chain_id = r.get::<i64, _>("chain_id");
        serde_json::json!({
            "chain_id": chain_id,
            "name": r.get::<String, _>("name"),
            "symbol": r.get::<String, _>("symbol"),
            "chain_type": "evm",
            "is_evm": true,
            "rpc_url": chain_rpc_endpoint(chain_id),
            "explorer_url": r.get::<String, _>("explorer_url"),
            "decimals": r.get::<i32, _>("decimals"),
            "derivation_path": r.get::<String, _>("derivation_path"),
        })
    }).collect();
    let non_evm_rows = sqlx::query(
        "SELECT chain_id, name, symbol, chain_type, rpc_url, explorer_url, decimals, derivation_path, address_prefix \
         FROM shmw_user_chains_nonevm ORDER BY chain_id",
    )
        .fetch_all(&state.pool).await.map_err(db_err)?;
    for r in &non_evm_rows {
        let chain_id = r.get::<i64, _>("chain_id");
        chains.push(serde_json::json!({
            "chain_id": chain_id,
            "name": r.get::<String, _>("name"),
            "symbol": r.get::<String, _>("symbol"),
            "chain_type": r.get::<String, _>("chain_type"),
            "is_evm": false,
            "rpc_url": r.get::<String, _>("rpc_url"),
            "explorer_url": r.get::<String, _>("explorer_url"),
            "decimals": r.get::<i32, _>("decimals"),
            "derivation_path": r.get::<String, _>("derivation_path"),
            "address_prefix": r.get::<String, _>("address_prefix"),
        }));
    }
    let count = chains.len();
    Ok(HttpResponse::Ok().json(serde_json::json!({ "chains": chains, "count": count })))
}

/// GetGas — GET /api/v1/gas?chain_id=N. Real fee data from the chain RPC;
/// fail-closed without an operator-configured endpoint.
async fn get_gas(state: web::Data<AppState>, req: HttpRequest) -> ApiResult<HttpResponse> {
    require_auth(req.headers(), &state)?;
    let chain_id: i64 = req
        .uri()
        .query()
        .and_then(|q| {
            q.split('&').find_map(|kv| {
                let (k, v) = kv.split_once('=')?;
                if k == "chain_id" { v.parse().ok() } else { None }
            })
        })
        .unwrap_or(1);
    let rpc = chain_rpc_endpoint(chain_id);
    if rpc.is_empty() {
        return Err(ApiError::BadRequest(format!("RPC endpoint not configured for chain {chain_id}")));
    }
    let gas_price = evm_tx::rpc_gas_price(&rpc)
        .await
        .map_err(|e| ApiError::BadRequest(format!("gas price fetch failed: {e}")))?;
    let tip = evm_tx::rpc_max_priority_fee(&rpc).await.ok();
    Ok(HttpResponse::Ok().json(serde_json::json!({
        "chain_id": chain_id,
        "gas_price_wei": gas_price,
        "max_priority_fee_wei": tip,
    })))
}

async fn get_price(state: web::Data<AppState>, req: HttpRequest) -> ApiResult<HttpResponse> {
    require_auth(req.headers(), &state)?;
    let chain_id: i64 = req
        .uri()
        .query()
        .and_then(|q| {
            q.split('&').find_map(|kv| {
                let (k, v) = kv.split_once('=')?;
                if k == "chain_id" { v.parse().ok() } else { None }
            })
        })
        .unwrap_or(1);

    let coin_id = chain_coin_gecko_id(chain_id);
    if coin_id.is_empty() {
        return Err(ApiError::BadRequest(format!(
            "no CoinGecko coin id configured for chain {chain_id}"
        )));
    }

    let price = fetch_token_price(&coin_id)
        .await
        .map_err(|e| ApiError::BadRequest(format!("price fetch failed: {e}")))?;

    Ok(HttpResponse::Ok().json(serde_json::json!({
        "chain_id": chain_id,
        "coin_id": coin_id,
        "price_usd": price.usd,
        "usd_24h_change": price.usd_24h_change,
        "usd_market_cap": price.usd_market_cap,
    })))
}

/// chainCoinGeckoID maps an EVM chain id to the CoinGecko coin id used for
/// native-asset price lookups (mirrors master_wallet/backend/config.go).
fn chain_coin_gecko_id(chain_id: i64) -> &'static str {
    match chain_id {
        1 => "ethereum",
        56 => "binancecoin",
        137 => "matic-network",
        42161 => "arbitrum",
        10 => "optimism",
        43114 => "avalanche-2",
        8453 => "base",
        _ => "",
    }
}

/// CoinGeckoPrice is a subset of the CoinGecko simple/price response.
#[derive(serde::Deserialize)]
struct CoinGeckoPrice {
    #[serde(default)]
    usd: f64,
    #[serde(default)]
    usd_24h_change: f64,
    #[serde(default)]
    usd_market_cap: f64,
}

/// fetch_token_price queries CoinGecko for the USD price of a coin id.
async fn fetch_token_price(coin_id: &str) -> Result<CoinGeckoPrice, String> {
    let url = format!(
        "https://api.coingecko.com/api/v3/simple/price?ids={coin_id}&vs_currencies=usd&include_24hr_change=true&include_market_cap=true"
    );
    let client = reqwest::Client::builder()
        .timeout(StdDuration::from_secs(10))
        .build()
        .map_err(|e| e.to_string())?;
    let mut req = client.get(&url);
    if let Ok(key) = env::var("COINGECKO_API_KEY") {
        if !key.is_empty() {
            req = req.header("x-cg-demo-api-key", key);
        }
    }
    let resp = req.send().await.map_err(|e| e.to_string())?;
    if resp.status().as_u16() != 200 {
        return Err(format!("CoinGecko returned HTTP {}", resp.status().as_u16()));
    }
    let raw: std::collections::HashMap<String, CoinGeckoPrice> =
        resp.json().await.map_err(|e| e.to_string())?;
    raw.get(coin_id)
        .map(|p| CoinGeckoPrice {
            usd: p.usd,
            usd_24h_change: p.usd_24h_change,
            usd_market_cap: p.usd_market_cap,
        })
        .ok_or_else(|| format!("no price for {coin_id}"))
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

    // Fail-closed license gate: phone home to the SuperAdmin control plane and
    // only serve protected routes while the license is valid. This closes the
    // P0 gap (previously this reference implementation ran unlicensed).
    let control_plane_url = env::var("TWO_PARTY_GATE_URL").unwrap_or_default();
    let control_plane_token = env::var("TWO_PARTY_GATE_TOKEN").unwrap_or_default();
    let license_key = env::var("WL_LICENSE_KEY").unwrap_or_default();
    let product = env::var("WL_PRODUCT").unwrap_or_else(|_| "master_wallet".into());
    let instance_id = env::var("WL_INSTANCE_ID")
        .unwrap_or_else(|_| std::env::var("HOSTNAME").unwrap_or_else(|_| "default".into()));
    let heartbeat_interval = env::var("HEARTBEAT_INTERVAL")
        .ok()
        .and_then(|v| v.parse::<u64>().ok())
        .map(StdDuration::from_secs)
        .unwrap_or_else(|| StdDuration::from_secs(30));

    let gate = state.gate.clone();
    tokio::spawn(license_gate::LicenseGate::heartbeat_loop(
        gate,
        control_plane_url,
        control_plane_token,
        license_key,
        product,
        instance_id,
        heartbeat_interval,
    ));

    // Auto-signer daemon: polls pending txs, auto-approves + signs + broadcasts
    // matching ones (fail-closed if MASTER_AUTO_SIGN_PASSWORD unset; never
    // touches two-party withdrawal-gated funds). Mirrors the Go canonical.
    tokio::spawn(auto_signer::run(state.pool.clone()));

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
                    .route("/master-wallet/{id}", web::put().to(update_master_wallet))
                    .route("/master-wallet/{id}", web::delete().to(delete_master_wallet))
                    .route("/master-wallet/{id}/balance", web::get().to(get_balance))
                    .route("/master-wallet/{id}/transactions", web::get().to(list_transactions))
                    .route("/master-wallet/{id}/transactions", web::post().to(create_transaction))
                    .route("/master-wallet/{id}/transactions/{tid}/approve", web::post().to(approve_transaction))
                    .route("/master-wallet/{id}/transactions/{tid}/reject", web::post().to(reject_transaction))
                    .route("/master-wallet/{id}/sign", web::post().to(sign_and_broadcast))
                    .route("/master-wallet/{id}/sign-message", web::post().to(sign_message))
                    .route("/master-wallet/{id}/derive-user-address", web::post().to(derive_user_address))
                    .route("/master-wallet/{id}/user-wallet-addresses", web::get().to(list_user_wallet_addresses))
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
                    .route("/health", web::get().to(health))
                    .service(
                        web::scope("/multisig")
                            .route("/wallets", web::post().to(multisig::create_multisig_wallet))
                            .route("/wallets", web::get().to(multisig::list_multisig_wallets))
                            .route("/wallets/{id}/transactions", web::get().to(multisig::list_multisig_transactions))
                            .route("/transactions", web::post().to(multisig::create_multisig_transaction))
                            .route("/transactions/{tx_id}/sign", web::post().to(multisig::sign_multisig_transaction))
                            .route("/transactions/{tx_id}/execute", web::post().to(multisig::execute_multisig_transaction)),
                    ),
            )
    })
    .bind(&bind)?
    .run()
    .await
}

