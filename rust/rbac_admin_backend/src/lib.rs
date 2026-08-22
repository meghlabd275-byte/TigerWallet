/**
 * TigerWallet RBAC Admin - Rust Implementation
 * High-performance, ultra-low latency backend
 * Production-ready with real PostgreSQL storage (tokio-postgres + deadpool).
 *
 * State is persisted in PostgreSQL and shared across instances. No in-memory
 * HashMaps, no fake/seed data — tables start empty and are owned/seeded by the
 * canonical Go backends.
 */

use serde::{Deserialize, Serialize};
use std::sync::Arc;
use std::collections::HashMap;
use chrono::Utc;
use uuid::Uuid;
use tokio_postgres::types::ToSql;
use tokio_postgres::{Config as PgConfig, NoTls, Row};
use tokio::runtime::Runtime;
use deadpool_postgres::{Manager as DeadpoolManager, Pool as DeadpoolPool};

// ==================== TYPES ====================

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum UserStatus {
    Active = 1,
    Suspended = 2,
    Banned = 3,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum KYCStatus {
    None = 0,
    Pending = 1,
    Approved = 2,
    Rejected = 3,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum TransactionStatus {
    Pending = 1,
    Completed = 2,
    Failed = 3,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum TransactionType {
    Deposit = 1,
    Withdrawal = 2,
    Transfer = 3,
    Swap = 4,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum PairStatus {
    Active = 1,
    Suspended = 2,
    Halted = 3,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum APIKeyTier {
    Free = 1,
    Basic = 2,
    Pro = 3,
    Enterprise = 4,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    pub id: String,
    pub email: String,
    pub username: String,
    pub password_hash: String,
    pub wallet_address: String,
    pub kyc_status: KYCStatus,
    pub status: UserStatus,
    pub created_at: i64,
    pub last_login: i64,
    pub balance: HashMap<String, f64>,
    pub two_factor_enabled: bool,
    pub ip_address: String,
    pub country: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KYCRequest {
    pub id: String,
    pub user_id: String,
    pub doc_type: String,
    pub status: KYCStatus,
    pub document_url: String,
    pub submitted_at: i64,
    pub reviewed_at: Option<i64>,
    pub reviewed_by: Option<String>,
    pub reject_reason: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub id: String,
    pub user_id: String,
    pub tx_type: TransactionType,
    pub amount: f64,
    pub currency: String,
    pub status: TransactionStatus,
    pub from_address: String,
    pub to_address: String,
    pub tx_hash: String,
    pub timestamp: i64,
    pub fee: f64,
    pub chain_id: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradingPair {
    pub id: String,
    pub base: String,
    pub quote: String,
    pub pair_name: String,
    pub price: f64,
    pub volume_24h: f64,
    pub liquidity: f64,
    pub status: PairStatus,
    pub chain_id: i32,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LiquidityPool {
    pub id: String,
    pub pair_id: String,
    pub user_id: String,
    pub base_amount: f64,
    pub quote_amount: f64,
    pub liquidity: f64,
    pub apr: f64,
    pub created_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeeStructure {
    pub id: String,
    pub fee_type: String,
    pub asset: String,
    pub fee_percent: f64,
    pub fee_fixed: f64,
    pub min_fee: f64,
    pub max_fee: Option<f64>,
    pub tier: String,
    pub is_active: bool,
    pub chain_id: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Blockchain {
    pub id: String,
    pub name: String,
    pub symbol: String,
    pub chain_id: i32,
    pub is_evm: bool,
    pub rpc_url: String,
    pub explorer_url: String,
    pub native_token: String,
    pub decimals: i32,
    pub is_active: bool,
    pub avg_gas_price_gwei: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BotInstance {
    pub id: String,
    pub user_id: String,
    pub bot_type: String,
    pub name: String,
    pub status: String,
    pub connected_dexs: i32,
    pub connected_cexs: i32,
    pub total_pnl: f64,
    pub total_volume: f64,
    pub total_orders: i32,
    pub avg_latency_us: i32,
    pub created_at: i64,
    pub last_trade_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BotTier {
    pub id: String,
    pub name: String,
    pub display_name: String,
    pub monthly_fee_usd: f64,
    pub per_dex_fee_usd: f64,
    pub per_cex_fee_usd: f64,
    pub max_bots: i32,
    pub max_dexs: i32,
    pub max_cexs: i32,
    pub max_position_usd: f64,
    pub max_daily_volume: f64,
    pub latency_target_ms: i32,
    pub is_active: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct APIKey {
    pub id: String,
    pub user_id: String,
    pub name: String,
    pub key: String,
    pub tier: APIKeyTier,
    pub permissions: APIKeyPermissions,
    pub rate_limit_per_min: i32,
    pub rate_limit_per_day: i32,
    pub is_active: bool,
    pub last_used_at: Option<i64>,
    pub expires_at: i64,
    pub created_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct APIKeyPermissions {
    pub trading: bool,
    pub reading: bool,
    pub withdrawal: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExternalConnection {
    pub id: String,
    pub user_id: String,
    pub exchange_name: String,
    pub account_id: String,
    pub is_active: bool,
    pub can_trade: bool,
    pub can_withdraw: bool,
    pub can_deposit: bool,
    pub last_sync_at: i64,
    pub sync_status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenListing {
    pub id: String,
    pub token_symbol: String,
    pub token_name: String,
    pub contract_address: String,
    pub chain_id: i32,
    pub tier: String,
    pub status: String,
    pub requester_address: String,
    pub requester_email: String,
    pub one_time_fee: f64,
    pub monthly_fee: f64,
    pub requested_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlatformStats {
    pub total_users: i32,
    pub active_users: i32,
    pub total_volume: f64,
    pub total_transactions: i32,
    pub total_fees: f64,
    pub active_bots: i32,
    pub total_bots: i32,
    pub active_cex_connections: i32,
    pub active_dex_connections: i32,
}

// ==================== DATABASE POOL ====================

/// Default connection string. Overridable via the `DATABASE_URL` env var.
const DEFAULT_DATABASE_URL: &str =
    "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable";

/// Idempotent DDL. Mirrors the canonical Go admin/super_admin schema
/// (admin_users / admin_roles / admin_role_assignments / admin_permissions) plus
/// the platform-admin entity tables this service manages. All `IF NOT EXISTS` —
/// safe to run on every startup.
const MIGRATIONS: &[&str] = &[
    // ---- Admin RBAC (mirrors super_admin/go/internal/database/postgres.go) ----
    "CREATE TABLE IF NOT EXISTS admin_users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), username VARCHAR(255) UNIQUE NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL, role VARCHAR(50) NOT NULL DEFAULT 'admin', white_label_id UUID, two_factor_secret VARCHAR(255), two_factor_enabled BOOLEAN DEFAULT FALSE, is_active BOOLEAN DEFAULT TRUE, status VARCHAR(50) DEFAULT 'active', created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW(), last_login TIMESTAMPTZ)",
    "CREATE TABLE IF NOT EXISTS admin_roles (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name TEXT UNIQUE NOT NULL, description TEXT, scope_set TEXT, permissions TEXT[] NOT NULL DEFAULT '{}', is_system BOOLEAN DEFAULT FALSE, is_active BOOLEAN DEFAULT TRUE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW())",
    "CREATE TABLE IF NOT EXISTS admin_role_assignments (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), admin_id UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE, role_id UUID NOT NULL REFERENCES admin_roles(id) ON DELETE CASCADE, granted_by UUID REFERENCES admin_users(id), granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE (admin_id, role_id))",
    "CREATE TABLE IF NOT EXISTS admin_permissions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name TEXT UNIQUE NOT NULL, description TEXT, category TEXT NOT NULL DEFAULT 'general', is_active BOOLEAN DEFAULT TRUE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())",
    "CREATE INDEX IF NOT EXISTS idx_admin_role_assignments_admin ON admin_role_assignments(admin_id)",
    // ---- Platform entities ----
    "CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, email TEXT NOT NULL, username TEXT NOT NULL, password_hash TEXT NOT NULL, wallet_address TEXT NOT NULL DEFAULT '', kyc_status INT NOT NULL DEFAULT 0, status INT NOT NULL DEFAULT 1, created_at BIGINT NOT NULL DEFAULT 0, last_login BIGINT NOT NULL DEFAULT 0, balance JSONB NOT NULL DEFAULT '{}', two_factor_enabled BOOLEAN DEFAULT FALSE, ip_address TEXT NOT NULL DEFAULT '', country TEXT NOT NULL DEFAULT '')",
    "CREATE TABLE IF NOT EXISTS kyc_requests (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, doc_type TEXT NOT NULL, status INT NOT NULL DEFAULT 0, document_url TEXT NOT NULL DEFAULT '', submitted_at BIGINT NOT NULL DEFAULT 0, reviewed_at BIGINT, reviewed_by TEXT, reject_reason TEXT)",
    "CREATE TABLE IF NOT EXISTS transactions (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, tx_type INT NOT NULL, amount DOUBLE PRECISION NOT NULL, currency TEXT NOT NULL, status INT NOT NULL, from_address TEXT NOT NULL DEFAULT '', to_address TEXT NOT NULL DEFAULT '', tx_hash TEXT NOT NULL DEFAULT '', timestamp BIGINT NOT NULL DEFAULT 0, fee DOUBLE PRECISION NOT NULL DEFAULT 0, chain_id INT NOT NULL DEFAULT 0)",
    "CREATE TABLE IF NOT EXISTS trading_pairs (id TEXT PRIMARY KEY, base TEXT NOT NULL, quote TEXT NOT NULL, pair_name TEXT NOT NULL, price DOUBLE PRECISION NOT NULL DEFAULT 0, volume_24h DOUBLE PRECISION NOT NULL DEFAULT 0, liquidity DOUBLE PRECISION NOT NULL DEFAULT 0, status INT NOT NULL DEFAULT 1, chain_id INT NOT NULL DEFAULT 0, created_at BIGINT NOT NULL DEFAULT 0, updated_at BIGINT NOT NULL DEFAULT 0)",
    "CREATE TABLE IF NOT EXISTS liquidity_pools (id TEXT PRIMARY KEY, pair_id TEXT NOT NULL, user_id TEXT NOT NULL, base_amount DOUBLE PRECISION NOT NULL DEFAULT 0, quote_amount DOUBLE PRECISION NOT NULL DEFAULT 0, liquidity DOUBLE PRECISION NOT NULL DEFAULT 0, apr DOUBLE PRECISION NOT NULL DEFAULT 0, created_at BIGINT NOT NULL DEFAULT 0)",
    "CREATE TABLE IF NOT EXISTS fee_structures (id TEXT PRIMARY KEY, fee_type TEXT NOT NULL, asset TEXT NOT NULL, fee_percent DOUBLE PRECISION NOT NULL DEFAULT 0, fee_fixed DOUBLE PRECISION NOT NULL DEFAULT 0, min_fee DOUBLE PRECISION NOT NULL DEFAULT 0, max_fee DOUBLE PRECISION, tier TEXT NOT NULL DEFAULT 'all', is_active BOOLEAN DEFAULT TRUE, chain_id INT NOT NULL DEFAULT 0)",
    "CREATE TABLE IF NOT EXISTS blockchains (id TEXT PRIMARY KEY, name TEXT NOT NULL, symbol TEXT NOT NULL, chain_id INT NOT NULL, is_evm BOOLEAN DEFAULT FALSE, rpc_url TEXT NOT NULL DEFAULT '', explorer_url TEXT NOT NULL DEFAULT '', native_token TEXT NOT NULL DEFAULT '', decimals INT NOT NULL DEFAULT 18, is_active BOOLEAN DEFAULT TRUE, avg_gas_price_gwei DOUBLE PRECISION NOT NULL DEFAULT 0)",
    "CREATE TABLE IF NOT EXISTS bot_instances (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, bot_type TEXT NOT NULL, name TEXT NOT NULL, status TEXT NOT NULL, connected_dexs INT NOT NULL DEFAULT 0, connected_cexs INT NOT NULL DEFAULT 0, total_pnl DOUBLE PRECISION NOT NULL DEFAULT 0, total_volume DOUBLE PRECISION NOT NULL DEFAULT 0, total_orders INT NOT NULL DEFAULT 0, avg_latency_us INT NOT NULL DEFAULT 0, created_at BIGINT NOT NULL DEFAULT 0, last_trade_at BIGINT NOT NULL DEFAULT 0)",
    "CREATE TABLE IF NOT EXISTS bot_tiers (id TEXT PRIMARY KEY, name TEXT NOT NULL, display_name TEXT NOT NULL, monthly_fee_usd DOUBLE PRECISION NOT NULL DEFAULT 0, per_dex_fee_usd DOUBLE PRECISION NOT NULL DEFAULT 0, per_cex_fee_usd DOUBLE PRECISION NOT NULL DEFAULT 0, max_bots INT NOT NULL DEFAULT 0, max_dexs INT NOT NULL DEFAULT 0, max_cexs INT NOT NULL DEFAULT 0, max_position_usd DOUBLE PRECISION NOT NULL DEFAULT 0, max_daily_volume DOUBLE PRECISION NOT NULL DEFAULT 0, latency_target_ms INT NOT NULL DEFAULT 0, is_active BOOLEAN DEFAULT TRUE)",
    "CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, name TEXT NOT NULL, key TEXT NOT NULL, tier INT NOT NULL DEFAULT 1, permissions JSONB NOT NULL DEFAULT '{}', rate_limit_per_min INT NOT NULL DEFAULT 60, rate_limit_per_day INT NOT NULL DEFAULT 10000, is_active BOOLEAN DEFAULT TRUE, last_used_at BIGINT, expires_at BIGINT NOT NULL DEFAULT 0, created_at BIGINT NOT NULL DEFAULT 0)",
    "CREATE TABLE IF NOT EXISTS external_connections (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, exchange_name TEXT NOT NULL, account_id TEXT NOT NULL, is_active BOOLEAN DEFAULT TRUE, can_trade BOOLEAN DEFAULT FALSE, can_withdraw BOOLEAN DEFAULT FALSE, can_deposit BOOLEAN DEFAULT FALSE, last_sync_at BIGINT NOT NULL DEFAULT 0, sync_status TEXT NOT NULL DEFAULT '')",
    "CREATE TABLE IF NOT EXISTS token_listings (id TEXT PRIMARY KEY, token_symbol TEXT NOT NULL, token_name TEXT NOT NULL, contract_address TEXT NOT NULL, chain_id INT NOT NULL DEFAULT 0, tier TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT '', requester_address TEXT NOT NULL DEFAULT '', requester_email TEXT NOT NULL DEFAULT '', one_time_fee DOUBLE PRECISION NOT NULL DEFAULT 0, monthly_fee DOUBLE PRECISION NOT NULL DEFAULT 0, requested_at BIGINT NOT NULL DEFAULT 0)",
];

/// Owns a deadpool-postgres connection pool plus a dedicated tokio `Runtime`.
///
/// The `Runtime` drives the synchronous `migrate()` / `health_check()` paths
/// via `block_on` (mirroring `rust/admin_fetchers/src/database.rs`). The async
/// query/execute helpers use the deadpool pool directly so they can be `.await`ed
/// from the service's `async` methods without nesting runtimes.
pub struct DatabasePool {
    pool: DeadpoolPool,
    runtime: Runtime,
}

impl DatabasePool {
    /// Construct from a libpq-style connection string, build the pool, and run
    /// migrations. Fail-closed: any connection/migration error is returned and
    /// NO service is constructed (never silently falls back to in-memory).
    pub fn new(database_url: &str) -> Result<Self, String> {
        let pg_config: PgConfig = database_url
            .parse()
            .map_err(|e| format!("invalid database url: {e}"))?;

        let manager = DeadpoolManager::new(pg_config, NoTls);
        let pool = DeadpoolPool::builder(manager)
            .max_size(16)
            .build()
            .map_err(|e| format!("failed to create postgres pool: {e}"))?;

        let runtime = Runtime::new().map_err(|e| format!("failed to create runtime: {e}"))?;
        let db = Self { pool, runtime };
        db.migrate()?;
        Ok(db)
    }

    /// Run all `CREATE TABLE IF NOT EXISTS` migrations (idempotent).
    fn migrate(&self) -> Result<(), String> {
        self.runtime.block_on(async {
            let client = self.pool.get().await.map_err(|e| format!("pool get: {e}"))?;
            for stmt in MIGRATIONS {
                client
                    .execute(*stmt, &[])
                    .await
                    .map_err(|e| format!("migration failed: {e}"))?;
            }
            Ok::<(), String>(())
        })
    }

    /// Synchronous health check (uses the owned runtime).
    pub fn health_check(&self) -> Result<bool, String> {
        self.runtime.block_on(async {
            let client = self.pool.get().await.map_err(|e| format!("pool get: {e}"))?;
            let rows = client
                .query("SELECT 1", &[])
                .await
                .map_err(|e| format!("health check failed: {e}"))?;
            Ok(!rows.is_empty())
        })
    }

    async fn conn(&self) -> Result<deadpool_postgres::Object, String> {
        self.pool.get().await.map_err(|e| format!("pool get: {e}"))
    }

    pub async fn query(
        &self,
        sql: &str,
        params: &[&(dyn ToSql + Sync)],
    ) -> Result<Vec<Row>, String> {
        let client = self.conn().await?;
        client.query(sql, params).await.map_err(|e| format!("query failed: {e}"))
    }

    pub async fn query_opt(
        &self,
        sql: &str,
        params: &[&(dyn ToSql + Sync)],
    ) -> Result<Option<Row>, String> {
        let client = self.conn().await?;
        let rows = client
            .query(sql, params)
            .await
            .map_err(|e| format!("query failed: {e}"))?;
        Ok(rows.into_iter().next())
    }

    pub async fn execute(
        &self,
        sql: &str,
        params: &[&(dyn ToSql + Sync)],
    ) -> Result<u64, String> {
        let client = self.conn().await?;
        client.execute(sql, params).await.map_err(|e| format!("execute failed: {e}"))
    }
}

// ==================== RBAC ADMIN SERVICE ====================

pub struct RBACAdminService {
    db: DatabasePool,
}

impl RBACAdminService {
    /// Construct from the `DATABASE_URL` env var (default
    /// `postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable`).
    /// Fail-closed: returns an error if the pool cannot be built or migrations fail.
    pub fn new() -> Result<Arc<Self>, String> {
        Self::with_url(&std::env::var("DATABASE_URL").unwrap_or_else(|_| DEFAULT_DATABASE_URL.to_string()))
    }

    /// Construct from an explicit connection string.
    pub fn with_url(database_url: &str) -> Result<Arc<Self>, String> {
        let db = DatabasePool::new(database_url)?;
        Ok(Arc::new(Self { db }))
    }

    // ==================== ENUM CONVERSIONS ====================

    fn user_status_from_i32(v: i32) -> UserStatus {
        match v {
            2 => UserStatus::Suspended,
            3 => UserStatus::Banned,
            _ => UserStatus::Active,
        }
    }

    fn kyc_status_from_i32(v: i32) -> KYCStatus {
        match v {
            1 => KYCStatus::Pending,
            2 => KYCStatus::Approved,
            3 => KYCStatus::Rejected,
            _ => KYCStatus::None,
        }
    }

    fn tx_status_from_i32(v: i32) -> TransactionStatus {
        match v {
            2 => TransactionStatus::Completed,
            3 => TransactionStatus::Failed,
            _ => TransactionStatus::Pending,
        }
    }

    fn tx_type_from_i32(v: i32) -> TransactionType {
        match v {
            2 => TransactionType::Withdrawal,
            3 => TransactionType::Transfer,
            4 => TransactionType::Swap,
            _ => TransactionType::Deposit,
        }
    }

    fn pair_status_from_i32(v: i32) -> PairStatus {
        match v {
            2 => PairStatus::Suspended,
            3 => PairStatus::Halted,
            _ => PairStatus::Active,
        }
    }

    fn api_key_tier_from_i32(v: i32) -> APIKeyTier {
        match v {
            2 => APIKeyTier::Basic,
            3 => APIKeyTier::Pro,
            4 => APIKeyTier::Enterprise,
            _ => APIKeyTier::Free,
        }
    }

    // ==================== USER MANAGEMENT ====================

    fn row_to_user(r: &Row) -> User {
        let balance_json: serde_json::Value = r.get("balance");
        let balance: HashMap<String, f64> = serde_json::from_value(balance_json).unwrap_or_default();
        User {
            id: r.get("id"),
            email: r.get("email"),
            username: r.get("username"),
            password_hash: r.get("password_hash"),
            wallet_address: r.get("wallet_address"),
            kyc_status: Self::kyc_status_from_i32(r.get("kyc_status")),
            status: Self::user_status_from_i32(r.get("status")),
            created_at: r.get("created_at"),
            last_login: r.get("last_login"),
            balance,
            two_factor_enabled: r.get("two_factor_enabled"),
            ip_address: r.get("ip_address"),
            country: r.get("country"),
        }
    }

    pub async fn get_all_users(&self) -> Vec<User> {
        self.db
            .query("SELECT id, email, username, password_hash, wallet_address, kyc_status, status, created_at, last_login, balance, two_factor_enabled, ip_address, country FROM users", &[])
            .await
            .map(|rows| rows.iter().map(Self::row_to_user).collect())
            .unwrap_or_default()
    }

    pub async fn get_user(&self, id: &str) -> Option<User> {
        self.db
            .query_opt(
                "SELECT id, email, username, password_hash, wallet_address, kyc_status, status, created_at, last_login, balance, two_factor_enabled, ip_address, country FROM users WHERE id = $1",
                &[&id],
            )
            .await
            .ok()
            .flatten()
            .map(|r| Self::row_to_user(&r))
    }

    pub async fn search_users(&self, query: &str) -> Vec<User> {
        let pattern = format!("%{}%", query.to_lowercase());
        self.db
            .query(
                "SELECT id, email, username, password_hash, wallet_address, kyc_status, status, created_at, last_login, balance, two_factor_enabled, ip_address, country FROM users WHERE LOWER(email) LIKE $1 OR LOWER(username) LIKE $1 OR LOWER(wallet_address) LIKE $1",
                &[&pattern],
            )
            .await
            .map(|rows| rows.iter().map(Self::row_to_user).collect())
            .unwrap_or_default()
    }

    pub async fn get_users_by_status(&self, status: UserStatus) -> Vec<User> {
        let v = status as i32;
        self.db
            .query(
                "SELECT id, email, username, password_hash, wallet_address, kyc_status, status, created_at, last_login, balance, two_factor_enabled, ip_address, country FROM users WHERE status = $1",
                &[&v],
            )
            .await
            .map(|rows| rows.iter().map(Self::row_to_user).collect())
            .unwrap_or_default()
    }

    pub async fn update_user_status(&self, user_id: &str, status: UserStatus) -> Result<(), String> {
        let v = status as i32;
        let n = self
            .db
            .execute("UPDATE users SET status = $1 WHERE id = $2", &[&v, &user_id])
            .await?;
        if n == 0 {
            Err("User not found".to_string())
        } else {
            Ok(())
        }
    }

    pub async fn ban_user(&self, user_id: &str) -> Result<(), String> {
        self.update_user_status(user_id, UserStatus::Banned).await
    }

    pub async fn unban_user(&self, user_id: &str) -> Result<(), String> {
        self.update_user_status(user_id, UserStatus::Active).await
    }

    pub async fn suspend_user(&self, user_id: &str) -> Result<(), String> {
        self.update_user_status(user_id, UserStatus::Suspended).await
    }

    // ==================== KYC MANAGEMENT ====================

    fn row_to_kyc(r: &Row) -> KYCRequest {
        KYCRequest {
            id: r.get("id"),
            user_id: r.get("user_id"),
            doc_type: r.get("doc_type"),
            status: Self::kyc_status_from_i32(r.get("status")),
            document_url: r.get("document_url"),
            submitted_at: r.get("submitted_at"),
            reviewed_at: r.get("reviewed_at"),
            reviewed_by: r.get("reviewed_by"),
            reject_reason: r.get("reject_reason"),
        }
    }

    pub async fn get_all_kyc_requests(&self) -> Vec<KYCRequest> {
        self.db
            .query("SELECT id, user_id, doc_type, status, document_url, submitted_at, reviewed_at, reviewed_by, reject_reason FROM kyc_requests", &[])
            .await
            .map(|rows| rows.iter().map(Self::row_to_kyc).collect())
            .unwrap_or_default()
    }

    pub async fn get_kyc_requests_by_status(&self, status: KYCStatus) -> Vec<KYCRequest> {
        let v = status as i32;
        self.db
            .query(
                "SELECT id, user_id, doc_type, status, document_url, submitted_at, reviewed_at, reviewed_by, reject_reason FROM kyc_requests WHERE status = $1",
                &[&v],
            )
            .await
            .map(|rows| rows.iter().map(Self::row_to_kyc).collect())
            .unwrap_or_default()
    }

    pub async fn approve_kyc(&self, request_id: &str, reviewer_id: &str) -> Result<(), String> {
        let now = Utc::now().timestamp();
        let mut client = self.db.conn().await?;
        let tx = client
            .transaction()
            .await
            .map_err(|e| format!("begin tx: {e}"))?;
        let row = tx
            .query_opt(
                "UPDATE kyc_requests SET status = 2, reviewed_at = $1, reviewed_by = $2 WHERE id = $3 RETURNING user_id",
                &[&now, &reviewer_id, &request_id],
            )
            .await
            .map_err(|e| format!("update kyc: {e}"))?;
        let user_id: String = match row {
            Some(r) => r.get(0),
            None => {
                let _ = tx.rollback().await;
                return Err("KYC request not found".to_string());
            }
        };
        tx.execute("UPDATE users SET kyc_status = 2 WHERE id = $1", &[&user_id])
            .await
            .map_err(|e| format!("update user kyc: {e}"))?;
        tx.commit().await.map_err(|e| format!("commit: {e}"))?;
        Ok(())
    }

    pub async fn reject_kyc(&self, request_id: &str, reviewer_id: &str, reason: &str) -> Result<(), String> {
        let now = Utc::now().timestamp();
        let mut client = self.db.conn().await?;
        let tx = client
            .transaction()
            .await
            .map_err(|e| format!("begin tx: {e}"))?;
        let row = tx
            .query_opt(
                "UPDATE kyc_requests SET status = 3, reviewed_at = $1, reviewed_by = $2, reject_reason = $3 WHERE id = $4 RETURNING user_id",
                &[&now, &reviewer_id, &reason, &request_id],
            )
            .await
            .map_err(|e| format!("update kyc: {e}"))?;
        let user_id: String = match row {
            Some(r) => r.get(0),
            None => {
                let _ = tx.rollback().await;
                return Err("KYC request not found".to_string());
            }
        };
        tx.execute("UPDATE users SET kyc_status = 3 WHERE id = $1", &[&user_id])
            .await
            .map_err(|e| format!("update user kyc: {e}"))?;
        tx.commit().await.map_err(|e| format!("commit: {e}"))?;
        Ok(())
    }

    // ==================== TRANSACTION MANAGEMENT ====================

    fn row_to_transaction(r: &Row) -> Transaction {
        Transaction {
            id: r.get("id"),
            user_id: r.get("user_id"),
            tx_type: Self::tx_type_from_i32(r.get("tx_type")),
            amount: r.get("amount"),
            currency: r.get("currency"),
            status: Self::tx_status_from_i32(r.get("status")),
            from_address: r.get("from_address"),
            to_address: r.get("to_address"),
            tx_hash: r.get("tx_hash"),
            timestamp: r.get("timestamp"),
            fee: r.get("fee"),
            chain_id: r.get("chain_id"),
        }
    }

    pub async fn get_all_transactions(&self) -> Vec<Transaction> {
        self.db
            .query("SELECT id, user_id, tx_type, amount, currency, status, from_address, to_address, tx_hash, timestamp, fee, chain_id FROM transactions", &[])
            .await
            .map(|rows| rows.iter().map(Self::row_to_transaction).collect())
            .unwrap_or_default()
    }

    pub async fn get_transactions_by_user(&self, user_id: &str) -> Vec<Transaction> {
        self.db
            .query(
                "SELECT id, user_id, tx_type, amount, currency, status, from_address, to_address, tx_hash, timestamp, fee, chain_id FROM transactions WHERE user_id = $1",
                &[&user_id],
            )
            .await
            .map(|rows| rows.iter().map(Self::row_to_transaction).collect())
            .unwrap_or_default()
    }

    pub async fn get_transactions_by_status(&self, status: TransactionStatus) -> Vec<Transaction> {
        let v = status as i32;
        self.db
            .query(
                "SELECT id, user_id, tx_type, amount, currency, status, from_address, to_address, tx_hash, timestamp, fee, chain_id FROM transactions WHERE status = $1",
                &[&v],
            )
            .await
            .map(|rows| rows.iter().map(Self::row_to_transaction).collect())
            .unwrap_or_default()
    }

    // ==================== TRADING PAIR MANAGEMENT ====================

    fn row_to_pair(r: &Row) -> TradingPair {
        TradingPair {
            id: r.get("id"),
            base: r.get("base"),
            quote: r.get("quote"),
            pair_name: r.get("pair_name"),
            price: r.get("price"),
            volume_24h: r.get("volume_24h"),
            liquidity: r.get("liquidity"),
            status: Self::pair_status_from_i32(r.get("status")),
            chain_id: r.get("chain_id"),
            created_at: r.get("created_at"),
            updated_at: r.get("updated_at"),
        }
    }

    pub async fn get_all_trading_pairs(&self) -> Vec<TradingPair> {
        self.db
            .query("SELECT id, base, quote, pair_name, price, volume_24h, liquidity, status, chain_id, created_at, updated_at FROM trading_pairs", &[])
            .await
            .map(|rows| rows.iter().map(Self::row_to_pair).collect())
            .unwrap_or_default()
    }

    pub async fn get_trading_pair(&self, id: &str) -> Option<TradingPair> {
        self.db
            .query_opt(
                "SELECT id, base, quote, pair_name, price, volume_24h, liquidity, status, chain_id, created_at, updated_at FROM trading_pairs WHERE id = $1",
                &[&id],
            )
            .await
            .ok()
            .flatten()
            .map(|r| Self::row_to_pair(&r))
    }

    pub async fn create_trading_pair(&self, base: &str, quote: &str, chain_id: i32) -> Result<TradingPair, String> {
        let pair_id = format!("{}_{}", base.to_lowercase(), quote.to_lowercase());
        let pair_name = format!("{}/{}", base, quote);
        let now = Utc::now().timestamp();
        let active = PairStatus::Active as i32;
        self.db
            .execute(
                "INSERT INTO trading_pairs (id, base, quote, pair_name, price, volume_24h, liquidity, status, chain_id, created_at, updated_at) VALUES ($1, $2, $3, $4, 0, 0, 0, $5, $6, $7, $7)",
                &[&pair_id, &base, &quote, &pair_name, &active, &chain_id, &now],
            )
            .await
            .map_err(|e| {
                if e.to_lowercase().contains("duplicate") {
                    "Pair already exists".to_string()
                } else {
                    e
                }
            })?;
        Ok(TradingPair {
            id: pair_id,
            base: base.to_string(),
            quote: quote.to_string(),
            pair_name,
            price: 0.0,
            volume_24h: 0.0,
            liquidity: 0.0,
            status: PairStatus::Active,
            chain_id,
            created_at: now,
            updated_at: now,
        })
    }

    pub async fn update_pair_status(&self, pair_id: &str, status: PairStatus) -> Result<(), String> {
        let v = status as i32;
        let now = Utc::now().timestamp();
        let n = self
            .db
            .execute("UPDATE trading_pairs SET status = $1, updated_at = $2 WHERE id = $3", &[&v, &now, &pair_id])
            .await?;
        if n == 0 {
            Err("Pair not found".to_string())
        } else {
            Ok(())
        }
    }

    pub async fn suspend_pair(&self, pair_id: &str) -> Result<(), String> {
        self.update_pair_status(pair_id, PairStatus::Suspended).await
    }

    pub async fn resume_pair(&self, pair_id: &str) -> Result<(), String> {
        self.update_pair_status(pair_id, PairStatus::Active).await
    }

    pub async fn halt_pair(&self, pair_id: &str) -> Result<(), String> {
        self.update_pair_status(pair_id, PairStatus::Halted).await
    }

    // ==================== FEE MANAGEMENT ====================

    fn row_to_fee(r: &Row) -> FeeStructure {
        FeeStructure {
            id: r.get("id"),
            fee_type: r.get("fee_type"),
            asset: r.get("asset"),
            fee_percent: r.get("fee_percent"),
            fee_fixed: r.get("fee_fixed"),
            min_fee: r.get("min_fee"),
            max_fee: r.get("max_fee"),
            tier: r.get("tier"),
            is_active: r.get("is_active"),
            chain_id: r.get("chain_id"),
        }
    }

    pub async fn get_all_fee_structures(&self) -> Vec<FeeStructure> {
        self.db
            .query("SELECT id, fee_type, asset, fee_percent, fee_fixed, min_fee, max_fee, tier, is_active, chain_id FROM fee_structures", &[])
            .await
            .map(|rows| rows.iter().map(Self::row_to_fee).collect())
            .unwrap_or_default()
    }

    pub async fn create_fee_structure(&self, fee_type: &str, asset: &str, tier: &str, fee_percent: f64, fee_fixed: f64, chain_id: i32) -> Result<FeeStructure, String> {
        let fee_id = Uuid::new_v4().to_string();
        self.db
            .execute(
                "INSERT INTO fee_structures (id, fee_type, asset, fee_percent, fee_fixed, min_fee, max_fee, tier, is_active, chain_id) VALUES ($1, $2, $3, $4, $5, 0, NULL, $6, TRUE, $7)",
                &[&fee_id, &fee_type, &asset, &fee_percent, &fee_fixed, &tier, &chain_id],
            )
            .await?;
        Ok(FeeStructure {
            id: fee_id,
            fee_type: fee_type.to_string(),
            asset: asset.to_string(),
            fee_percent,
            fee_fixed,
            min_fee: 0.0,
            max_fee: None,
            tier: tier.to_string(),
            is_active: true,
            chain_id,
        })
    }

    pub async fn update_fee(&self, fee_id: &str, fee_percent: f64, fee_fixed: f64) -> Result<(), String> {
        let n = self
            .db
            .execute("UPDATE fee_structures SET fee_percent = $1, fee_fixed = $2 WHERE id = $3", &[&fee_percent, &fee_fixed, &fee_id])
            .await?;
        if n == 0 {
            Err("Fee structure not found".to_string())
        } else {
            Ok(())
        }
    }

    // ==================== BLOCKCHAIN MANAGEMENT ====================

    fn row_to_blockchain(r: &Row) -> Blockchain {
        Blockchain {
            id: r.get("id"),
            name: r.get("name"),
            symbol: r.get("symbol"),
            chain_id: r.get("chain_id"),
            is_evm: r.get("is_evm"),
            rpc_url: r.get("rpc_url"),
            explorer_url: r.get("explorer_url"),
            native_token: r.get("native_token"),
            decimals: r.get("decimals"),
            is_active: r.get("is_active"),
            avg_gas_price_gwei: r.get("avg_gas_price_gwei"),
        }
    }

    pub async fn get_all_blockchains(&self) -> Vec<Blockchain> {
        self.db
            .query("SELECT id, name, symbol, chain_id, is_evm, rpc_url, explorer_url, native_token, decimals, is_active, avg_gas_price_gwei FROM blockchains", &[])
            .await
            .map(|rows| rows.iter().map(Self::row_to_blockchain).collect())
            .unwrap_or_default()
    }

    pub async fn get_blockchain(&self, id: &str) -> Option<Blockchain> {
        self.db
            .query_opt(
                "SELECT id, name, symbol, chain_id, is_evm, rpc_url, explorer_url, native_token, decimals, is_active, avg_gas_price_gwei FROM blockchains WHERE id = $1",
                &[&id],
            )
            .await
            .ok()
            .flatten()
            .map(|r| Self::row_to_blockchain(&r))
    }

    pub async fn add_blockchain(&self, name: &str, symbol: &str, chain_id: i32, is_evm: bool, rpc_url: &str, explorer_url: &str, native_token: &str, decimals: i32) -> Result<Blockchain, String> {
        let blockchain_id = symbol.to_lowercase();
        self.db
            .execute(
                "INSERT INTO blockchains (id, name, symbol, chain_id, is_evm, rpc_url, explorer_url, native_token, decimals, is_active, avg_gas_price_gwei) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE, 0)",
                &[&blockchain_id, &name, &symbol, &chain_id, &is_evm, &rpc_url, &explorer_url, &native_token, &decimals],
            )
            .await
            .map_err(|e| {
                if e.to_lowercase().contains("duplicate") {
                    "Blockchain already exists".to_string()
                } else {
                    e
                }
            })?;
        Ok(Blockchain {
            id: blockchain_id,
            name: name.to_string(),
            symbol: symbol.to_string(),
            chain_id,
            is_evm,
            rpc_url: rpc_url.to_string(),
            explorer_url: explorer_url.to_string(),
            native_token: native_token.to_string(),
            decimals,
            is_active: true,
            avg_gas_price_gwei: 0.0,
        })
    }

    pub async fn update_blockchain(&self, id: &str, rpc_url: &str, explorer_url: &str) -> Result<(), String> {
        let n = self
            .db
            .execute("UPDATE blockchains SET rpc_url = $1, explorer_url = $2 WHERE id = $3", &[&rpc_url, &explorer_url, &id])
            .await?;
        if n == 0 {
            Err("Blockchain not found".to_string())
        } else {
            Ok(())
        }
    }

    pub async fn set_blockchain_status(&self, id: &str, is_active: bool) -> Result<(), String> {
        let n = self
            .db
            .execute("UPDATE blockchains SET is_active = $1 WHERE id = $2", &[&is_active, &id])
            .await?;
        if n == 0 {
            Err("Blockchain not found".to_string())
        } else {
            Ok(())
        }
    }

    // ==================== BOT MANAGEMENT ====================

    fn row_to_bot(r: &Row) -> BotInstance {
        BotInstance {
            id: r.get("id"),
            user_id: r.get("user_id"),
            bot_type: r.get("bot_type"),
            name: r.get("name"),
            status: r.get("status"),
            connected_dexs: r.get("connected_dexs"),
            connected_cexs: r.get("connected_cexs"),
            total_pnl: r.get("total_pnl"),
            total_volume: r.get("total_volume"),
            total_orders: r.get("total_orders"),
            avg_latency_us: r.get("avg_latency_us"),
            created_at: r.get("created_at"),
            last_trade_at: r.get("last_trade_at"),
        }
    }

    pub async fn get_all_bot_instances(&self) -> Vec<BotInstance> {
        self.db
            .query("SELECT id, user_id, bot_type, name, status, connected_dexs, connected_cexs, total_pnl, total_volume, total_orders, avg_latency_us, created_at, last_trade_at FROM bot_instances", &[])
            .await
            .map(|rows| rows.iter().map(Self::row_to_bot).collect())
            .unwrap_or_default()
    }

    pub async fn get_bot_instances_by_user(&self, user_id: &str) -> Vec<BotInstance> {
        self.db
            .query(
                "SELECT id, user_id, bot_type, name, status, connected_dexs, connected_cexs, total_pnl, total_volume, total_orders, avg_latency_us, created_at, last_trade_at FROM bot_instances WHERE user_id = $1",
                &[&user_id],
            )
            .await
            .map(|rows| rows.iter().map(Self::row_to_bot).collect())
            .unwrap_or_default()
    }

    fn row_to_bot_tier(r: &Row) -> BotTier {
        BotTier {
            id: r.get("id"),
            name: r.get("name"),
            display_name: r.get("display_name"),
            monthly_fee_usd: r.get("monthly_fee_usd"),
            per_dex_fee_usd: r.get("per_dex_fee_usd"),
            per_cex_fee_usd: r.get("per_cex_fee_usd"),
            max_bots: r.get("max_bots"),
            max_dexs: r.get("max_dexs"),
            max_cexs: r.get("max_cexs"),
            max_position_usd: r.get("max_position_usd"),
            max_daily_volume: r.get("max_daily_volume"),
            latency_target_ms: r.get("latency_target_ms"),
            is_active: r.get("is_active"),
        }
    }

    pub async fn get_all_bot_tiers(&self) -> Vec<BotTier> {
        self.db
            .query("SELECT id, name, display_name, monthly_fee_usd, per_dex_fee_usd, per_cex_fee_usd, max_bots, max_dexs, max_cexs, max_position_usd, max_daily_volume, latency_target_ms, is_active FROM bot_tiers", &[])
            .await
            .map(|rows| rows.iter().map(Self::row_to_bot_tier).collect())
            .unwrap_or_default()
    }

    pub async fn update_bot_status(&self, bot_id: &str, status: &str) -> Result<(), String> {
        let n = self
            .db
            .execute("UPDATE bot_instances SET status = $1 WHERE id = $2", &[&status, &bot_id])
            .await?;
        if n == 0 {
            Err("Bot not found".to_string())
        } else {
            Ok(())
        }
    }

    pub async fn pause_bot(&self, bot_id: &str) -> Result<(), String> {
        self.update_bot_status(bot_id, "paused").await
    }

    pub async fn resume_bot(&self, bot_id: &str) -> Result<(), String> {
        self.update_bot_status(bot_id, "running").await
    }

    pub async fn stop_bot(&self, bot_id: &str) -> Result<(), String> {
        self.update_bot_status(bot_id, "stopped").await
    }

    // ==================== API KEY MANAGEMENT ====================

    fn row_to_api_key(r: &Row) -> APIKey {
        let perms_json: serde_json::Value = r.get("permissions");
        let permissions: APIKeyPermissions =
            serde_json::from_value(perms_json).unwrap_or(APIKeyPermissions { trading: false, reading: false, withdrawal: false });
        APIKey {
            id: r.get("id"),
            user_id: r.get("user_id"),
            name: r.get("name"),
            key: r.get("key"),
            tier: Self::api_key_tier_from_i32(r.get("tier")),
            permissions,
            rate_limit_per_min: r.get("rate_limit_per_min"),
            rate_limit_per_day: r.get("rate_limit_per_day"),
            is_active: r.get("is_active"),
            last_used_at: r.get("last_used_at"),
            expires_at: r.get("expires_at"),
            created_at: r.get("created_at"),
        }
    }

    pub async fn get_all_api_keys(&self) -> Vec<APIKey> {
        self.db
            .query("SELECT id, user_id, name, key, tier, permissions, rate_limit_per_min, rate_limit_per_day, is_active, last_used_at, expires_at, created_at FROM api_keys", &[])
            .await
            .map(|rows| rows.iter().map(Self::row_to_api_key).collect())
            .unwrap_or_default()
    }

    pub async fn get_api_keys_by_user(&self, user_id: &str) -> Vec<APIKey> {
        self.db
            .query(
                "SELECT id, user_id, name, key, tier, permissions, rate_limit_per_min, rate_limit_per_day, is_active, last_used_at, expires_at, created_at FROM api_keys WHERE user_id = $1",
                &[&user_id],
            )
            .await
            .map(|rows| rows.iter().map(Self::row_to_api_key).collect())
            .unwrap_or_default()
    }

    pub async fn create_api_key(&self, user_id: &str, name: &str, tier: APIKeyTier, permissions: APIKeyPermissions) -> Result<APIKey, String> {
        let key_id = Uuid::new_v4().to_string();
        let api_key = format!("tw_{}", Uuid::new_v4().to_string().replace("-", ""));
        let tier_v = tier as i32;
        let perms_json = serde_json::to_value(&permissions).map_err(|e| format!("serialize perms: {e}"))?;
        let now = Utc::now().timestamp();
        let expires_at = now + (365 * 24 * 60 * 60);
        self.db
            .execute(
                "INSERT INTO api_keys (id, user_id, name, key, tier, permissions, rate_limit_per_min, rate_limit_per_day, is_active, last_used_at, expires_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, 60, 10000, TRUE, NULL, $7, $8)",
                &[&key_id, &user_id, &name, &api_key, &tier_v, &perms_json, &expires_at, &now],
            )
            .await?;
        Ok(APIKey {
            id: key_id,
            user_id: user_id.to_string(),
            name: name.to_string(),
            key: api_key,
            tier: Self::api_key_tier_from_i32(tier_v),
            permissions,
            rate_limit_per_min: 60,
            rate_limit_per_day: 10000,
            is_active: true,
            last_used_at: None,
            expires_at,
            created_at: now,
        })
    }

    pub async fn revoke_api_key(&self, key_id: &str) -> Result<(), String> {
        let n = self
            .db
            .execute("UPDATE api_keys SET is_active = FALSE WHERE id = $1", &[&key_id])
            .await?;
        if n == 0 {
            Err("API key not found".to_string())
        } else {
            Ok(())
        }
    }

    // ==================== PLATFORM STATS ====================

    pub async fn get_platform_stats(&self) -> PlatformStats {
        // Aggregate real counts from PostgreSQL. No fabricated numbers — every
        // field is derived from a live table count.
        let total_users = self.count("SELECT COUNT(*) FROM users").await.unwrap_or(0);
        let active_users = self
            .count("SELECT COUNT(*) FROM users WHERE status = 1")
            .await
            .unwrap_or(0);
        let total_transactions = self.count("SELECT COUNT(*) FROM transactions").await.unwrap_or(0);
        let total_volume = self
            .sum("SELECT COALESCE(SUM(amount), 0) FROM transactions")
            .await
            .unwrap_or(0.0);
        let total_fees = self
            .sum("SELECT COALESCE(SUM(fee), 0) FROM transactions")
            .await
            .unwrap_or(0.0);
        let total_bots = self.count("SELECT COUNT(*) FROM bot_instances").await.unwrap_or(0);
        let active_bots = self
            .count("SELECT COUNT(*) FROM bot_instances WHERE status = 'running'")
            .await
            .unwrap_or(0);
        let active_cex_connections = self
            .count("SELECT COUNT(*) FROM external_connections WHERE is_active = TRUE")
            .await
            .unwrap_or(0);
        let active_dex_connections = self
            .count("SELECT COUNT(*) FROM liquidity_pools")
            .await
            .unwrap_or(0);

        PlatformStats {
            total_users,
            active_users,
            total_volume,
            total_transactions,
            total_fees,
            active_bots,
            total_bots,
            active_cex_connections,
            active_dex_connections,
        }
    }

    async fn count(&self, sql: &str) -> Result<i32, String> {
        let rows = self.db.query(sql, &[]).await?;
        Ok(rows.first().and_then(|r| r.try_get::<_, i64>(0).ok()).map(|v| v as i32).unwrap_or(0))
    }

    async fn sum(&self, sql: &str) -> Result<f64, String> {
        let rows = self.db.query(sql, &[]).await?;
        Ok(rows.first().and_then(|r| r.try_get::<_, f64>(0).ok()).unwrap_or(0.0))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_new_fail_closed_without_db() {
        // No live DB reachable on this bogus port → fail-closed (Err, never
        // silently falls back to in-memory storage).
        let svc = RBACAdminService::with_url(
            "postgres://tigerwallet:tigerwallet@127.0.0.1:1/tigerwallet?sslmode=disable&connect_timeout=1",
        );
        assert!(svc.is_err());
    }

    #[tokio::test]
    async fn test_enum_conversions_roundtrip() {
        assert_eq!(RBACAdminService::user_status_from_i32(3), UserStatus::Banned);
        assert_eq!(RBACAdminService::kyc_status_from_i32(2), KYCStatus::Approved);
        assert_eq!(RBACAdminService::tx_type_from_i32(4), TransactionType::Swap);
        assert_eq!(RBACAdminService::pair_status_from_i32(2), PairStatus::Suspended);
        assert_eq!(RBACAdminService::api_key_tier_from_i32(4), APIKeyTier::Enterprise);
    }
}
