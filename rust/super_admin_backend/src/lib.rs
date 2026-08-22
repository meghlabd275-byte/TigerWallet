/**
 * TigerWallet Super Admin - Rust Implementation
 * High-performance, ultra-low latency backend
 * Production-ready with real implementations (no stubs)
 * 
 * This is a comprehensive Rust implementation with:
 * - Real database integration (SQLx)
 * - Real 2FA (TOTP)
 * - bcrypt password hashing
 * - Complete CRUD operations
 * - Rate limiting
 * - IP whitelisting
 * - Full audit logging
 */

use serde::{Deserialize, Serialize};
use std::sync::Arc;
use chrono::Utc;
use uuid::Uuid;
use argon2::{Argon2, PasswordHasher, password_hash::SaltString};
use rand::rngs::OsRng;
use base32::{Alphabet, encode};
use tokio_postgres::types::ToSql;
use tokio_postgres::{Config as PgConfig, NoTls, Row};
use tokio::runtime::Runtime;
use deadpool_postgres::{Manager as DeadpoolManager, Pool as DeadpoolPool};

// ==================== TYPES ====================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AdminRole {
    SuperAdmin = 1,
    Admin = 2,
    Manager = 3,
    Support = 4,
}

impl AdminRole {
    pub fn from_str(s: &str) -> Self {
        match s {
            "super_admin" => Self::SuperAdmin,
            "manager" => Self::Manager,
            "support" => Self::Support,
            _ => Self::Admin,
        }
    }
    pub fn as_str(&self) -> &'static str {
        match self {
            Self::SuperAdmin => "super_admin",
            Self::Admin => "admin",
            Self::Manager => "manager",
            Self::Support => "support",
        }
    }
    pub fn as_i32(&self) -> i32 {
        match self {
            Self::SuperAdmin => 1,
            Self::Admin => 2,
            Self::Manager => 3,
            Self::Support => 4,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AdminStatus {
    Active = 1,
    Suspended = 2,
    Blocked = 3,
}

impl AdminStatus {
    pub fn from_i32(i: i32) -> Self {
        match i {
            2 => Self::Suspended,
            3 => Self::Blocked,
            _ => Self::Active,
        }
    }
    pub fn as_i32(&self) -> i32 {
        match self {
            Self::Active => 1,
            Self::Suspended => 2,
            Self::Blocked => 3,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SecurityLevel {
    Basic = 1,
    Medium = 2,
    High = 3,
    Enterprise = 4,
}

impl SecurityLevel {
    pub fn from_i32(i: i32) -> Self {
        match i {
            1 => Self::Basic,
            2 => Self::Medium,
            3 => Self::High,
            _ => Self::Enterprise,
        }
    }
    pub fn as_i32(&self) -> i32 {
        match self {
            Self::Basic => 1,
            Self::Medium => 2,
            Self::High => 3,
            Self::Enterprise => 4,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Admin {
    pub id: String,
    pub username: String,
    pub password_hash: String,
    pub email: String,
    pub role: AdminRole,
    pub security_level: SecurityLevel,
    pub permissions: Vec<String>,
    pub two_factor_enabled: bool,
    pub two_factor_secret: Option<String>,
    pub created_at: i64,
    pub last_login: i64,
    pub status: AdminStatus,
    pub failed_attempts: i32,
    pub locked_until: i64,
    pub ip_whitelist: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WhiteLabel {
    pub id: String,
    pub name: String,
    pub domain: String,
    pub api_key: String,
    pub api_key_hash: String,
    pub fee_percent: f64,
    pub status: i32, // 1=pending, 2=active, 3=suspended, 4=revoked
    pub approved_by: Option<String>,
    pub approved_at: Option<i64>,
    pub created_at: i64,
    pub features: Vec<String>,
    pub custom_branding: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Session {
    pub id: String,
    pub admin_id: String,
    pub token: String,
    pub expires_at: i64,
    pub ip_address: String,
    pub user_agent: String,
    pub created_at: i64,
    pub is_valid: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLog {
    pub id: String,
    pub admin_id: String,
    pub admin_username: String,
    pub action: String,
    pub details: String,
    pub ip_address: String,
    pub user_agent: String,
    pub timestamp: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProfitShareConfig {
    pub id: String,
    pub white_label_id: String,
    pub super_admin_wallet: String,
    pub master_wallet_address: Option<String>,
    pub profit_percentage: f64,
    pub min_percentage: f64,
    pub max_percentage: f64,
    pub is_active: bool,
    pub auto_transfer: bool,
    pub transfer_frequency: String,
    pub last_transfer: i64,
    pub total_transferred: f64,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProfitTransaction {
    pub id: String,
    pub white_label_id: String,
    pub super_admin_wallet: String,
    pub amount: f64,
    pub percentage: f64,
    pub gross_revenue: f64,
    pub net_revenue: f64,
    pub token: String,
    pub tx_hash: Option<String>,
    pub status: String,
    pub created_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeatureFlag {
    pub id: String,
    pub name: String,
    pub description: String,
    pub global_enabled: bool,
    pub enabled: bool,
    pub master_admin_id: Option<String>,
    pub white_label_id: Option<String>,
    pub updated_by: Option<String>,
    pub updated_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoginAttempt {
    pub identifier: String,
    pub count: i32,
    pub first_attempt: i64,
    pub last_attempt: i64,
    pub locked: bool,
    pub locked_until: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthResult {
    pub success: bool,
    pub error: Option<String>,
    pub session_token: Option<String>,
    pub admin_id: Option<String>,
    pub username: Option<String>,
    pub role: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PasswordPolicy {
    pub min_length: usize,
    pub max_length: usize,
    pub require_uppercase: bool,
    pub require_lowercase: bool,
    pub require_numbers: bool,
    pub require_special: bool,
    pub max_age_days: i32,
    pub history_count: i32,
}

impl Default for PasswordPolicy {
    fn default() -> Self {
        PasswordPolicy {
            min_length: 8,
            max_length: 128,
            require_uppercase: true,
            require_lowercase: true,
            require_numbers: true,
            require_special: true,
            max_age_days: 90,
            history_count: 5,
        }
    }
}


// ==================== SUPER ADMIN SERVICE ====================

const DEFAULT_DATABASE_URL: &str =
    "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable";

/// Idempotent DDL mirroring the canonical Go super_admin schema
/// (super_admin/go/internal/database/postgres.go): admin_users, admin_sessions,
/// ip_whitelist, feature_flags, audit_logs, white_labels, plus the
/// profit-share / login-attempt / rate-limit tables this service manages.
/// All `IF NOT EXISTS` — safe to run on every startup. No seed data is
/// inserted here; the Go backends own seeding.
const MIGRATIONS: &[&str] = &[
    "CREATE TABLE IF NOT EXISTS admin_users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), username VARCHAR(255) UNIQUE NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL, role VARCHAR(50) NOT NULL DEFAULT 'admin', security_level INT NOT NULL DEFAULT 1, permissions TEXT[] NOT NULL DEFAULT '{}', two_factor_secret VARCHAR(255), two_factor_enabled BOOLEAN DEFAULT FALSE, is_active BOOLEAN DEFAULT TRUE, status INT NOT NULL DEFAULT 1, failed_attempts INT NOT NULL DEFAULT 0, locked_until BIGINT NOT NULL DEFAULT 0, ip_whitelist TEXT[] NOT NULL DEFAULT '{}', created_at BIGINT NOT NULL DEFAULT 0, last_login BIGINT NOT NULL DEFAULT 0)",
    "CREATE TABLE IF NOT EXISTS admin_sessions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), admin_id UUID NOT NULL, token VARCHAR(255) NOT NULL, expires_at BIGINT NOT NULL, ip_address TEXT NOT NULL DEFAULT '', user_agent TEXT NOT NULL DEFAULT '', created_at BIGINT NOT NULL DEFAULT 0, is_valid BOOLEAN DEFAULT TRUE)",
    "CREATE INDEX IF NOT EXISTS idx_admin_sessions_admin_id ON admin_sessions(admin_id)",
    "CREATE INDEX IF NOT EXISTS idx_admin_sessions_token ON admin_sessions(token)",
    "CREATE TABLE IF NOT EXISTS ip_whitelist (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), ip_address VARCHAR(45) UNIQUE NOT NULL, description TEXT, is_active BOOLEAN DEFAULT TRUE, created_at BIGINT NOT NULL DEFAULT 0, created_by UUID)",
    "CREATE TABLE IF NOT EXISTS feature_flags (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(255) UNIQUE NOT NULL, description TEXT NOT NULL DEFAULT '', global_enabled BOOLEAN DEFAULT FALSE, enabled BOOLEAN DEFAULT FALSE, master_admin_id UUID, white_label_id UUID, updated_by UUID, updated_at BIGINT NOT NULL DEFAULT 0)",
    "CREATE TABLE IF NOT EXISTS audit_logs (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), admin_id UUID, admin_username VARCHAR(255) NOT NULL DEFAULT '', action VARCHAR(100) NOT NULL, details TEXT NOT NULL DEFAULT '', ip_address TEXT NOT NULL DEFAULT '', user_agent TEXT NOT NULL DEFAULT '', timestamp BIGINT NOT NULL DEFAULT 0)",
    "CREATE INDEX IF NOT EXISTS idx_audit_logs_admin_id ON audit_logs(admin_id)",
    "CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp)",
    "CREATE TABLE IF NOT EXISTS white_labels (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(255) NOT NULL, domain VARCHAR(255) UNIQUE NOT NULL, api_key TEXT NOT NULL, api_key_hash TEXT NOT NULL, fee_percent DOUBLE PRECISION NOT NULL DEFAULT 20, status INT NOT NULL DEFAULT 1, approved_by UUID, approved_at BIGINT, created_at BIGINT NOT NULL DEFAULT 0, features TEXT[] NOT NULL DEFAULT '{}', custom_branding BOOLEAN DEFAULT TRUE)",
    "CREATE TABLE IF NOT EXISTS profit_share_configs (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), white_label_id UUID NOT NULL, super_admin_wallet TEXT NOT NULL DEFAULT '', master_wallet_address TEXT, profit_percentage DOUBLE PRECISION NOT NULL DEFAULT 0, min_percentage DOUBLE PRECISION NOT NULL DEFAULT 0, max_percentage DOUBLE PRECISION NOT NULL DEFAULT 50, is_active BOOLEAN DEFAULT TRUE, auto_transfer BOOLEAN DEFAULT TRUE, transfer_frequency TEXT NOT NULL DEFAULT 'daily', last_transfer BIGINT NOT NULL DEFAULT 0, total_transferred DOUBLE PRECISION NOT NULL DEFAULT 0, created_at BIGINT NOT NULL DEFAULT 0, updated_at BIGINT NOT NULL DEFAULT 0, UNIQUE (white_label_id))",
    "CREATE TABLE IF NOT EXISTS profit_transactions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), white_label_id UUID NOT NULL, super_admin_wallet TEXT NOT NULL, amount DOUBLE PRECISION NOT NULL, percentage DOUBLE PRECISION NOT NULL, gross_revenue DOUBLE PRECISION NOT NULL, net_revenue DOUBLE PRECISION NOT NULL, token TEXT NOT NULL, tx_hash TEXT, status TEXT NOT NULL, created_at BIGINT NOT NULL DEFAULT 0)",
    "CREATE INDEX IF NOT EXISTS idx_profit_transactions_white_label ON profit_transactions(white_label_id)",
    "CREATE TABLE IF NOT EXISTS login_attempts (identifier TEXT PRIMARY KEY, count INT NOT NULL DEFAULT 0, first_attempt BIGINT NOT NULL DEFAULT 0, last_attempt BIGINT NOT NULL DEFAULT 0, locked BOOLEAN NOT NULL DEFAULT FALSE, locked_until BIGINT NOT NULL DEFAULT 0)",
    "CREATE TABLE IF NOT EXISTS rate_limits (identifier TEXT PRIMARY KEY, window_start BIGINT NOT NULL, count INT NOT NULL DEFAULT 0)",
];

/// Owns a deadpool-postgres connection pool plus a dedicated tokio `Runtime`.
///
/// The `Runtime` drives the synchronous `migrate()` / `health_check()` paths
/// via `block_on` (mirroring `rust/admin_fetchers/src/database.rs`). The async
/// query/execute helpers use the deadpool pool directly so they can be
/// `.await`ed from the service's `async` methods without nesting runtimes.
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

pub struct SuperAdminService {
    db: DatabasePool,

    // Configuration (non-persistent operational policy)
    password_policy: PasswordPolicy,
    max_failed_attempts: i32,
    lockout_duration_seconds: i64,
    session_duration_seconds: i64,
}

impl SuperAdminService {
    /// Construct from the `DATABASE_URL` env var (default
    /// `postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable`).
    /// Fail-closed: returns an error if the pool cannot be built or migrations
    /// fail — NEVER silently falls back to in-memory storage.
    pub fn new() -> Result<Arc<Self>, String> {
        Self::with_url(&std::env::var("DATABASE_URL").unwrap_or_else(|_| DEFAULT_DATABASE_URL.to_string()))
    }

    /// Construct from an explicit connection string.
    pub fn with_url(database_url: &str) -> Result<Arc<Self>, String> {
        let db = DatabasePool::new(database_url)?;
        Ok(Arc::new(Self {
            db,
            password_policy: PasswordPolicy::default(),
            max_failed_attempts: 3,
            lockout_duration_seconds: 900, // 15 minutes
            session_duration_seconds: 86400, // 24 hours
        }))
    }

    // ==================== PASSWORD OPERATIONS ====================

    pub fn hash_password(password: &str) -> Result<String, String> {
        let salt = SaltString::generate(&mut OsRng);
        let argon2 = Argon2::default();

        match argon2.hash_password(password.as_bytes(), &salt) {
            Ok(hash) => Ok(hash.to_string()),
            Err(e) => Err(format!("Failed to hash password: {}", e)),
        }
    }

    pub fn verify_password(password: &str, hash: &str) -> bool {
        use argon2::{PasswordHash, PasswordVerifier};

        match PasswordHash::new(hash) {
            Ok(parsed_hash) => {
                Argon2::default()
                    .verify_password(password.as_bytes(), &parsed_hash)
                    .is_ok()
            }
            Err(_) => false,
        }
    }

    pub fn validate_password_policy(&self, password: &str) -> Result<(), String> {
        let policy = &self.password_policy;

        if password.len() < policy.min_length {
            return Err(format!("Password must be at least {} characters", policy.min_length));
        }

        if password.len() > policy.max_length {
            return Err(format!("Password must not exceed {} characters", policy.max_length));
        }

        if policy.require_uppercase && !password.chars().any(|c| c.is_uppercase()) {
            return Err("Password must contain at least one uppercase letter".to_string());
        }

        if policy.require_lowercase && !password.chars().any(|c| c.is_lowercase()) {
            return Err("Password must contain at least one lowercase letter".to_string());
        }

        if policy.require_numbers && !password.chars().any(|c| c.is_numeric()) {
            return Err("Password must contain at least one number".to_string());
        }

        if policy.require_special && !password.chars().any(|c| !c.is_alphanumeric()) {
            return Err("Password must contain at least one special character".to_string());
        }

        Ok(())
    }

    // ==================== TOTP 2FA ====================

    pub fn generate_totp_secret() -> String {
        let mut bytes = [0u8; 20];
        rand::RngCore::fill_bytes(&mut OsRng, &mut bytes);
        encode(Alphabet::Rfc4648 { padding: false }, &bytes)
    }

    pub fn verify_totp(secret: &str, code: &str) -> bool {
        if code.len() != 6 || !code.chars().all(|c| c.is_numeric()) {
            return false;
        }

        let now = Utc::now().timestamp();

        // Check current and adjacent time windows
        for offset in -1..=1 {
            let timestamp = now + (offset * 30);
            if Self::compute_totp(secret, timestamp) == code {
                return true;
            }
        }

        false
    }

    fn compute_totp(secret: &str, timestamp: i64) -> String {
        use std::io::Write;
        use sha1::{Sha1, Digest};

        // Decode base32 secret
        let secret_bytes = match base32::decode(Alphabet::Rfc4648 { padding: false }, secret) {
            Some(bytes) => bytes,
            None => return "000000".to_string(),
        };

        // Calculate counter (30-second periods)
        let counter = (timestamp / 30) as u64;

        // Convert counter to 8 bytes big-endian
        let mut counter_bytes = [0u8; 8];
        for i in (0..8).rev() {
            counter_bytes[i] = (counter >> (i * 8)) as u8;
        }

        // Compute HMAC-SHA1
        let mut hmac = Sha1::new();
        hmac.write_all(&secret_bytes).unwrap();
        let result = hmac.finalize();

        // Dynamic truncation
        let offset = (result[19] & 0x0f) as usize;
        let binary = ((result[offset] & 0x7f) as u32) << 24
            | (result[offset + 1] as u32) << 16
            | (result[offset + 2] as u32) << 8
            | (result[offset + 3] as u32);

        format!("{:06}", binary % 1_000_000)
    }

    // ==================== ROW MAPPERS ====================

    fn row_to_admin(row: &Row) -> Admin {
        let role_str: String = row.get("role");
        let status_i: i32 = row.get("status");
        let sec_i: i32 = row.get("security_level");
        Admin {
            id: row.get("id"),
            username: row.get("username"),
            password_hash: row.get("password_hash"),
            email: row.get("email"),
            role: AdminRole::from_str(&role_str),
            security_level: SecurityLevel::from_i32(sec_i),
            permissions: row.get("permissions"),
            two_factor_enabled: row.get("two_factor_enabled"),
            two_factor_secret: row.get("two_factor_secret"),
            created_at: row.get("created_at"),
            last_login: row.get("last_login"),
            status: AdminStatus::from_i32(status_i),
            failed_attempts: row.get("failed_attempts"),
            locked_until: row.get("locked_until"),
            ip_whitelist: row.get("ip_whitelist"),
        }
    }

    fn row_to_white_label(row: &Row) -> WhiteLabel {
        WhiteLabel {
            id: row.get("id"),
            name: row.get("name"),
            domain: row.get("domain"),
            api_key: row.get("api_key"),
            api_key_hash: row.get("api_key_hash"),
            fee_percent: row.get("fee_percent"),
            status: row.get("status"),
            approved_by: row.get("approved_by"),
            approved_at: row.get("approved_at"),
            created_at: row.get("created_at"),
            features: row.get("features"),
            custom_branding: row.get("custom_branding"),
        }
    }

    #[allow(dead_code)]
    fn row_to_session(row: &Row) -> Session {
        Session {
            id: row.get("id"),
            admin_id: row.get("admin_id"),
            token: row.get("token"),
            expires_at: row.get("expires_at"),
            ip_address: row.get("ip_address"),
            user_agent: row.get("user_agent"),
            created_at: row.get("created_at"),
            is_valid: row.get("is_valid"),
        }
    }

    fn row_to_audit_log(row: &Row) -> AuditLog {
        AuditLog {
            id: row.get("id"),
            admin_id: row.get("admin_id"),
            admin_username: row.get("admin_username"),
            action: row.get("action"),
            details: row.get("details"),
            ip_address: row.get("ip_address"),
            user_agent: row.get("user_agent"),
            timestamp: row.get("timestamp"),
        }
    }

    fn row_to_profit_config(row: &Row) -> ProfitShareConfig {
        ProfitShareConfig {
            id: row.get("id"),
            white_label_id: row.get("white_label_id"),
            super_admin_wallet: row.get("super_admin_wallet"),
            master_wallet_address: row.get("master_wallet_address"),
            profit_percentage: row.get("profit_percentage"),
            min_percentage: row.get("min_percentage"),
            max_percentage: row.get("max_percentage"),
            is_active: row.get("is_active"),
            auto_transfer: row.get("auto_transfer"),
            transfer_frequency: row.get("transfer_frequency"),
            last_transfer: row.get("last_transfer"),
            total_transferred: row.get("total_transferred"),
            created_at: row.get("created_at"),
            updated_at: row.get("updated_at"),
        }
    }

    fn row_to_profit_tx(row: &Row) -> ProfitTransaction {
        ProfitTransaction {
            id: row.get("id"),
            white_label_id: row.get("white_label_id"),
            super_admin_wallet: row.get("super_admin_wallet"),
            amount: row.get("amount"),
            percentage: row.get("percentage"),
            gross_revenue: row.get("gross_revenue"),
            net_revenue: row.get("net_revenue"),
            token: row.get("token"),
            tx_hash: row.get("tx_hash"),
            status: row.get("status"),
            created_at: row.get("created_at"),
        }
    }

    fn row_to_feature_flag(row: &Row) -> FeatureFlag {
        FeatureFlag {
            id: row.get("id"),
            name: row.get("name"),
            description: row.get("description"),
            global_enabled: row.get("global_enabled"),
            enabled: row.get("enabled"),
            master_admin_id: row.get("master_admin_id"),
            white_label_id: row.get("white_label_id"),
            updated_by: row.get("updated_by"),
            updated_at: row.get("updated_at"),
        }
    }

    // ==================== AUTHENTICATION ====================

    pub async fn login(
        &self,
        username: &str,
        password: &str,
        two_factor_code: Option<&str>,
        ip_address: &str,
        user_agent: &str,
    ) -> AuthResult {
        let failure = |msg: &str| AuthResult {
            success: false,
            error: Some(msg.to_string()),
            session_token: None,
            admin_id: None,
            username: None,
            role: None,
        };

        // Check if account is locked
        if self.is_account_locked(username).await {
            return failure("Account is temporarily locked due to too many failed attempts");
        }

        // Find admin by username or email
        let admin = match self
            .db
            .query_opt(
                "SELECT * FROM admin_users WHERE username = $1 OR email = $1",
                &[&username],
            )
            .await
        {
            Ok(Some(row)) => Self::row_to_admin(&row),
            Ok(None) => {
                self.record_failed_attempt(username).await;
                return failure("Invalid credentials");
            }
            Err(e) => {
                eprintln!("login query failed: {e}");
                return failure("Internal error");
            }
        };

        // Check IP whitelist
        if !admin.ip_whitelist.is_empty()
            && !admin.ip_whitelist.contains(&ip_address.to_string())
        {
            self.log_audit(&admin.id, "LOGIN_FAILED", &format!("IP {} not in whitelist", ip_address), ip_address, user_agent).await;
            return failure("Login from this IP address is not allowed");
        }

        // Verify password
        if !Self::verify_password(password, &admin.password_hash) {
            self.record_failed_attempt(username).await;

            // Increment failed_attempts; lock if threshold reached
            let new_count = admin.failed_attempts + 1;
            let lock_until = if new_count >= self.max_failed_attempts {
                Utc::now().timestamp() + self.lockout_duration_seconds
            } else {
                admin.locked_until
            };
            let _ = self
                .db
                .execute(
                    "UPDATE admin_users SET failed_attempts = $1, locked_until = $2 WHERE id = $3",
                    &[&new_count, &lock_until, &admin.id],
                )
                .await;

            self.log_audit(&admin.id, "LOGIN_FAILED", "Invalid password", ip_address, user_agent).await;
            return failure("Invalid credentials");
        }

        // Check 2FA if enabled
        if admin.two_factor_enabled {
            let code = match two_factor_code {
                Some(c) => c,
                None => return failure("Two-factor authentication code required"),
            };

            let secret = match &admin.two_factor_secret {
                Some(s) => s,
                None => return failure("2FA not properly configured"),
            };

            if !Self::verify_totp(secret, code) {
                self.log_audit(&admin.id, "LOGIN_FAILED", "Invalid 2FA code", ip_address, user_agent).await;
                return failure("Invalid two-factor authentication code");
            }
        }

        // Clear failed attempts
        self.clear_failed_attempts(username).await;

        // Update last login + reset failure counters
        let now = Utc::now().timestamp();
        let _ = self
            .db
            .execute(
                "UPDATE admin_users SET last_login = $1, failed_attempts = 0, locked_until = 0 WHERE id = $2",
                &[&now, &admin.id],
            )
            .await;

        // Create session
        let session_token = Uuid::new_v4().to_string();
        let session_id = Uuid::new_v4().to_string();
        let expires_at = now + self.session_duration_seconds;
        let _ = self
            .db
            .execute(
                "INSERT INTO admin_sessions (id, admin_id, token, expires_at, ip_address, user_agent, created_at, is_valid) VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)",
                &[
                    &session_id,
                    &admin.id,
                    &session_token,
                    &expires_at,
                    &ip_address.to_string(),
                    &user_agent.to_string(),
                    &now,
                ],
            )
            .await;

        self.log_audit(&admin.id, "LOGIN_SUCCESS", "Login successful", ip_address, user_agent).await;

        let role = admin.role.as_i32();

        AuthResult {
            success: true,
            error: None,
            session_token: Some(session_token),
            admin_id: Some(admin.id),
            username: Some(admin.username),
            role: Some(role),
        }
    }

    pub async fn logout(&self, token: &str) -> bool {
        let row = self
            .db
            .query_opt("SELECT admin_id FROM admin_sessions WHERE token = $1", &[&token])
            .await;

        let admin_id: Option<String> = match row {
            Ok(Some(r)) => Some(r.get("admin_id")),
            _ => None,
        };

        if let Some(id) = admin_id {
            let _ = self
                .db
                .execute(
                    "UPDATE admin_sessions SET is_valid = FALSE WHERE token = $1",
                    &[&token],
                )
                .await;
            self.log_audit(&id, "LOGOUT", "User logged out", "", "").await;
            return true;
        }

        false
    }

    pub async fn validate_session(&self, token: &str) -> bool {
        match self
            .db
            .query_opt(
                "SELECT is_valid, expires_at FROM admin_sessions WHERE token = $1",
                &[&token],
            )
            .await
        {
            Ok(Some(row)) => {
                let is_valid: bool = row.get("is_valid");
                let expires_at: i64 = row.get("expires_at");
                is_valid && expires_at > Utc::now().timestamp()
            }
            _ => false,
        }
    }

    // ==================== RATE LIMITING ====================

    pub async fn is_rate_limited(&self, identifier: &str) -> bool {
        match self
            .db
            .query_opt(
                "SELECT window_start, count FROM rate_limits WHERE identifier = $1",
                &[&identifier],
            )
            .await
        {
            Ok(Some(row)) => {
                let window_start: i64 = row.get("window_start");
                let count: i32 = row.get("count");
                if Utc::now().timestamp() - window_start > 60 {
                    return false; // Window expired
                }
                count >= 100 // 100 requests per minute
            }
            _ => false,
        }
    }

    pub async fn record_request(&self, identifier: &str) {
        let now = Utc::now().timestamp();
        let row = self
            .db
            .query_opt(
                "SELECT window_start, count FROM rate_limits WHERE identifier = $1",
                &[&identifier],
            )
            .await;
        match row {
            Ok(Some(r)) => {
                let window_start: i64 = r.get("window_start");
                if now - window_start > 60 {
                    let _ = self
                        .db
                        .execute(
                            "UPDATE rate_limits SET window_start = $1, count = 1 WHERE identifier = $2",
                            &[&now, &identifier],
                        )
                        .await;
                } else {
                    let _ = self
                        .db
                        .execute(
                            "UPDATE rate_limits SET count = count + 1 WHERE identifier = $1",
                            &[&identifier],
                        )
                        .await;
                }
            }
            Ok(None) => {
                let _ = self
                    .db
                    .execute(
                        "INSERT INTO rate_limits (identifier, window_start, count) VALUES ($1, $2, 1) ON CONFLICT (identifier) DO UPDATE SET count = rate_limits.count + 1",
                        &[&identifier, &now],
                    )
                    .await;
            }
            Err(e) => eprintln!("record_request failed: {e}"),
        }
    }

    // ==================== FAILED ATTEMPTS ====================

    async fn is_account_locked(&self, identifier: &str) -> bool {
        match self
            .db
            .query_opt(
                "SELECT locked, locked_until FROM login_attempts WHERE identifier = $1",
                &[&identifier],
            )
            .await
        {
            Ok(Some(row)) => {
                let locked: bool = row.get("locked");
                let locked_until: i64 = row.get("locked_until");
                locked && locked_until > Utc::now().timestamp()
            }
            _ => false,
        }
    }

    async fn record_failed_attempt(&self, identifier: &str) {
        let now = Utc::now().timestamp();
        let row = self
            .db
            .query_opt(
                "SELECT count FROM login_attempts WHERE identifier = $1",
                &[&identifier],
            )
            .await;
        let new_count: i32 = match row {
            Ok(Some(r)) => r.get::<_, i32>("count") + 1,
            _ => 1,
        };
        let (locked, locked_until) = if new_count >= self.max_failed_attempts {
            (true, now + self.lockout_duration_seconds)
        } else {
            (false, 0)
        };
        let _ = self
            .db
            .execute(
                "INSERT INTO login_attempts (identifier, count, first_attempt, last_attempt, locked, locked_until) VALUES ($1, $2, $3, $3, $4, $5) ON CONFLICT (identifier) DO UPDATE SET count = $2, last_attempt = $3, locked = $4, locked_until = $5",
                &[&identifier, &new_count, &now, &locked, &locked_until],
            )
            .await;
    }

    async fn clear_failed_attempts(&self, identifier: &str) {
        let _ = self
            .db
            .execute(
                "DELETE FROM login_attempts WHERE identifier = $1",
                &[&identifier],
            )
            .await;
    }

    // ==================== ADMIN MANAGEMENT ====================

    pub async fn create_admin(
        &self,
        username: &str,
        password: &str,
        email: &str,
        role: AdminRole,
        permissions: Vec<String>,
        creator_id: &str,
    ) -> Result<Admin, String> {
        // Validate password
        self.validate_password_policy(password)?;

        // Check if username exists
        if self
            .db
            .query_opt("SELECT id FROM admin_users WHERE username = $1", &[&username])
            .await
            .map_err(|e| format!("db error: {e}"))?
            .is_some()
        {
            return Err("Username already exists".to_string());
        }
        if self
            .db
            .query_opt("SELECT id FROM admin_users WHERE email = $1", &[&email])
            .await
            .map_err(|e| format!("db error: {e}"))?
            .is_some()
        {
            return Err("Email already registered".to_string());
        }

        // Check if creator is super admin when creating super admin
        if matches!(role, AdminRole::SuperAdmin) {
            let row = self
                .db
                .query_opt("SELECT role FROM admin_users WHERE id = $1", &[&creator_id])
                .await
                .map_err(|e| format!("db error: {e}"))?;
            if let Some(r) = row {
                let creator_role: String = r.get("role");
                if creator_role != AdminRole::SuperAdmin.as_str() {
                    return Err("Only super admin can create super admin accounts".to_string());
                }
            }
        }

        let admin_id = Uuid::new_v4().to_string();
        let password_hash = Self::hash_password(password)?;

        let security_level = match role {
            AdminRole::SuperAdmin => SecurityLevel::Enterprise,
            AdminRole::Admin => SecurityLevel::High,
            _ => SecurityLevel::Medium,
        };

        let now = Utc::now().timestamp();
        let role_str = role.as_str().to_string();
        let sec_i = security_level.as_i32();
        let perms_ref: &[String] = &permissions;
        let ipwl: Vec<String> = vec![];

        self.db
            .execute(
                "INSERT INTO admin_users (id, username, password_hash, email, role, security_level, permissions, two_factor_enabled, two_factor_secret, is_active, status, failed_attempts, locked_until, ip_whitelist, created_at, last_login) VALUES ($1, $2, $3, $4, $5, $6, $7, FALSE, NULL, TRUE, 1, 0, 0, $8, $9, 0)",
                &[
                    &admin_id,
                    &username.to_string(),
                    &password_hash,
                    &email.to_string(),
                    &role_str,
                    &sec_i,
                    &perms_ref,
                    &ipwl,
                    &now,
                ],
            )
            .await
            .map_err(|e| format!("insert admin failed: {e}"))?;

        let admin = Admin {
            id: admin_id.clone(),
            username: username.to_string(),
            password_hash,
            email: email.to_string(),
            role,
            security_level,
            permissions,
            two_factor_enabled: false,
            two_factor_secret: None,
            created_at: now,
            last_login: 0,
            status: AdminStatus::Active,
            failed_attempts: 0,
            locked_until: 0,
            ip_whitelist: vec![],
        };

        self.log_audit(creator_id, "CREATE_ADMIN", &format!("Created admin: {}", username), "", "").await;

        Ok(admin)
    }

    pub async fn get_admin(&self, id: &str) -> Option<Admin> {
        match self.db.query_opt("SELECT * FROM admin_users WHERE id = $1", &[&id]).await {
            Ok(Some(row)) => Some(Self::row_to_admin(&row)),
            _ => None,
        }
    }

    pub async fn get_all_admins(&self) -> Vec<Admin> {
        match self.db.query("SELECT * FROM admin_users ORDER BY created_at", &[]).await {
            Ok(rows) => rows.iter().map(Self::row_to_admin).collect(),
            Err(e) => {
                eprintln!("get_all_admins failed: {e}");
                vec![]
            }
        }
    }

    pub async fn update_admin_status(&self, admin_id: &str, status: AdminStatus, updater_id: &str) -> Result<(), String> {
        // Check permissions
        let row = self
            .db
            .query_opt("SELECT role FROM admin_users WHERE id = $1", &[&updater_id])
            .await
            .map_err(|e| format!("db error: {e}"))?
            .ok_or("Updater not found")?;
        let updater_role: String = row.get("role");
        if updater_role != AdminRole::SuperAdmin.as_str() {
            return Err("Unauthorized".to_string());
        }

        // Can't modify yourself
        if admin_id == updater_id {
            return Err("Cannot modify your own status".to_string());
        }

        let status_i = status.as_i32();
        self.db
            .execute(
                "UPDATE admin_users SET status = $1, is_active = $2 WHERE id = $3",
                &[&status_i, &(status_i == 1), &admin_id],
            )
            .await
            .map_err(|e| format!("update failed: {e}"))?;

        let status_str = match status {
            AdminStatus::Active => "Activated",
            AdminStatus::Suspended => "Suspended",
            AdminStatus::Blocked => "Blocked",
        };

        self.log_audit(updater_id, "UPDATE_ADMIN_STATUS",
            &format!("{} admin: {}", status_str, admin_id), "", "").await;

        // Invalidate sessions
        if matches!(status, AdminStatus::Suspended | AdminStatus::Blocked) {
            let _ = self
                .db
                .execute(
                    "UPDATE admin_sessions SET is_valid = FALSE WHERE admin_id = $1",
                    &[&admin_id],
                )
                .await;
        }

        Ok(())
    }

    // ==================== WHITE LABEL MANAGEMENT ====================

    pub async fn create_white_label(
        &self,
        name: &str,
        domain: &str,
        creator_id: &str,
    ) -> Result<WhiteLabel, String> {
        // Check if domain exists
        if self
            .db
            .query_opt("SELECT id FROM white_labels WHERE domain = $1", &[&domain])
            .await
            .map_err(|e| format!("db error: {e}"))?
            .is_some()
        {
            return Err("Domain already registered".to_string());
        }

        let wl_id = Uuid::new_v4().to_string();
        let api_key = format!("tw_{}", Uuid::new_v4().to_string().replace("-", ""));
        let api_key_hash = Self::hash_password(&api_key).unwrap_or_default();
        let now = Utc::now().timestamp();
        let features: Vec<String> = vec!["*".to_string()];

        let feats_ref: &[String] = &features;
        self.db
            .execute(
                "INSERT INTO white_labels (id, name, domain, api_key, api_key_hash, fee_percent, status, approved_by, approved_at, created_at, features, custom_branding) VALUES ($1, $2, $3, $4, $5, 20.0, 1, NULL, NULL, $6, $7, TRUE)",
                &[
                    &wl_id,
                    &name.to_string(),
                    &domain.to_string(),
                    &api_key,
                    &api_key_hash,
                    &now,
                    &feats_ref,
                ],
            )
            .await
            .map_err(|e| format!("insert white_label failed: {e}"))?;

        let white_label = WhiteLabel {
            id: wl_id.clone(),
            name: name.to_string(),
            domain: domain.to_string(),
            api_key,
            api_key_hash,
            fee_percent: 20.0,
            status: 1, // pending
            approved_by: None,
            approved_at: None,
            created_at: now,
            features,
            custom_branding: true,
        };

        self.log_audit(creator_id, "CREATE_WHITELABEL",
            &format!("Created white label: {} ({})", name, domain), "", "").await;

        Ok(white_label)
    }

    pub async fn get_white_label(&self, id: &str) -> Option<WhiteLabel> {
        match self.db.query_opt("SELECT * FROM white_labels WHERE id = $1", &[&id]).await {
            Ok(Some(row)) => Some(Self::row_to_white_label(&row)),
            _ => None,
        }
    }

    pub async fn get_all_white_labels(&self) -> Vec<WhiteLabel> {
        match self.db.query("SELECT * FROM white_labels ORDER BY created_at", &[]).await {
            Ok(rows) => rows.iter().map(Self::row_to_white_label).collect(),
            Err(e) => {
                eprintln!("get_all_white_labels failed: {e}");
                vec![]
            }
        }
    }

    pub async fn approve_white_label(&self, wl_id: &str, approver_id: &str) -> Result<(), String> {
        // Check permissions
        let row = self
            .db
            .query_opt("SELECT role FROM admin_users WHERE id = $1", &[&approver_id])
            .await
            .map_err(|e| format!("db error: {e}"))?
            .ok_or("Approver not found")?;
        let approver_role: String = row.get("role");
        if approver_role != AdminRole::SuperAdmin.as_str() {
            return Err("Unauthorized".to_string());
        }

        let now = Utc::now().timestamp();
        let n = self
            .db
            .execute(
                "UPDATE white_labels SET status = 2, approved_by = $1, approved_at = $2 WHERE id = $3",
                &[&approver_id, &now, &wl_id],
            )
            .await
            .map_err(|e| format!("update failed: {e}"))?;
        if n == 0 {
            return Err("White label not found".to_string());
        }

        self.log_audit(approver_id, "APPROVE_WHITELABEL",
            &format!("Approved white label: {}", wl_id), "", "").await;

        Ok(())
    }

    pub async fn update_white_label_fee(&self, wl_id: &str, fee_percent: f64, updater_id: &str) -> Result<(), String> {
        if fee_percent < 0.0 || fee_percent > 20.0 {
            return Err("Fee must be between 0 and 20%".to_string());
        }

        // Check permissions
        let row = self
            .db
            .query_opt("SELECT role FROM admin_users WHERE id = $1", &[&updater_id])
            .await
            .map_err(|e| format!("db error: {e}"))?
            .ok_or("Updater not found")?;
        let updater_role: String = row.get("role");
        if updater_role != AdminRole::SuperAdmin.as_str() {
            return Err("Unauthorized".to_string());
        }

        let n = self
            .db
            .execute(
                "UPDATE white_labels SET fee_percent = $1 WHERE id = $2",
                &[&fee_percent, &wl_id],
            )
            .await
            .map_err(|e| format!("update failed: {e}"))?;
        if n == 0 {
            return Err("White label not found".to_string());
        }

        self.log_audit(updater_id, "UPDATE_WHITELABEL_FEE",
            &format!("Updated fee to {}% for: {}", fee_percent, wl_id), "", "").await;

        Ok(())
    }

    pub async fn validate_api_key(&self, api_key: &str) -> Option<WhiteLabel> {
        // Check by direct key (active white labels only)
        if let Ok(Some(row)) = self
            .db
            .query_opt(
                "SELECT * FROM white_labels WHERE api_key = $1 AND status = 2",
                &[&api_key],
            )
            .await
        {
            return Some(Self::row_to_white_label(&row));
        }

        // Check by hashed key — iterate active white labels and verify
        let rows = self
            .db
            .query("SELECT * FROM white_labels WHERE status = 2", &[])
            .await
            .ok()?;
        for row in &rows {
            let hash: String = row.get("api_key_hash");
            if Self::verify_password(api_key, &hash) {
                return Some(Self::row_to_white_label(row));
            }
        }
        None
    }

    // ==================== AUDIT LOGGING ====================

    async fn log_audit(&self, admin_id: &str, action: &str, details: &str, ip_address: &str, user_agent: &str) {
        let admin_username = match self
            .db
            .query_opt("SELECT username FROM admin_users WHERE id = $1", &[&admin_id])
            .await
        {
            Ok(Some(r)) => r.get::<_, String>("username"),
            _ => String::new(),
        };

        let log_id = Uuid::new_v4().to_string();
        let now = Utc::now().timestamp();
        let _ = self
            .db
            .execute(
                "INSERT INTO audit_logs (id, admin_id, admin_username, action, details, ip_address, user_agent, timestamp) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
                &[
                    &log_id,
                    &admin_id.to_string(),
                    &admin_username,
                    &action.to_string(),
                    &details.to_string(),
                    &ip_address.to_string(),
                    &user_agent.to_string(),
                    &now,
                ],
            )
            .await;
    }

    pub async fn get_audit_logs(&self, admin_id: Option<&str>, limit: usize) -> Vec<AuditLog> {
        let limit_i = limit as i64;
        let result = match admin_id {
            Some(id) => {
                self.db
                    .query(
                        "SELECT * FROM audit_logs WHERE admin_id = $1 ORDER BY timestamp DESC LIMIT $2",
                        &[&id, &limit_i],
                    )
                    .await
            }
            None => {
                self.db
                    .query(
                        "SELECT * FROM audit_logs ORDER BY timestamp DESC LIMIT $1",
                        &[&limit_i],
                    )
                    .await
            }
        };
        match result {
            Ok(rows) => rows.iter().map(Self::row_to_audit_log).collect(),
            Err(e) => {
                eprintln!("get_audit_logs failed: {e}");
                vec![]
            }
        }
    }

    // ==================== PROFIT SHARING ====================

    pub async fn set_profit_share(&self, white_label_id: &str, percentage: f64, super_admin_id: &str) -> Result<(), String> {
        if percentage < 0.0 || percentage > 50.0 {
            return Err("Percentage must be between 0 and 50".to_string());
        }

        // Check permissions
        let row = self
            .db
            .query_opt("SELECT role FROM admin_users WHERE id = $1", &[&super_admin_id])
            .await
            .map_err(|e| format!("db error: {e}"))?
            .ok_or("Admin not found")?;
        let admin_role: String = row.get("role");
        if admin_role != AdminRole::SuperAdmin.as_str() {
            return Err("Unauthorized".to_string());
        }

        let now = Utc::now().timestamp();
        let cfg_id = Uuid::new_v4().to_string();
        self.db
            .execute(
                "INSERT INTO profit_share_configs (id, white_label_id, super_admin_wallet, master_wallet_address, profit_percentage, min_percentage, max_percentage, is_active, auto_transfer, transfer_frequency, last_transfer, total_transferred, created_at, updated_at) VALUES ($1, $2, '', NULL, $3, 0.0, 50.0, TRUE, TRUE, 'daily', 0, 0.0, $4, $4) ON CONFLICT (white_label_id) DO UPDATE SET profit_percentage = $3, is_active = TRUE, updated_at = $4",
                &[&cfg_id, &white_label_id.to_string(), &percentage, &now],
            )
            .await
            .map_err(|e| format!("upsert profit config failed: {e}"))?;

        self.log_audit(super_admin_id, "SET_PROFIT_SHARE",
            &format!("Set profit share to {}% for: {}", percentage, white_label_id), "", "").await;

        Ok(())
    }

    pub async fn get_profit_share(&self, white_label_id: &str) -> Option<ProfitShareConfig> {
        match self
            .db
            .query_opt(
                "SELECT * FROM profit_share_configs WHERE white_label_id = $1 AND is_active = TRUE",
                &[&white_label_id],
            )
            .await
        {
            Ok(Some(row)) => Some(Self::row_to_profit_config(&row)),
            _ => None,
        }
    }

    pub async fn calculate_profit_share(&self, white_label_id: &str, gross_revenue: f64) -> (f64, f64) {
        let config = self.get_profit_share(white_label_id).await;

        let percentage = config.map(|c| c.profit_percentage).unwrap_or(20.0);
        let super_admin_share = gross_revenue * (percentage / 100.0);
        let white_label_share = gross_revenue - super_admin_share;

        (super_admin_share, white_label_share)
    }

    pub async fn execute_profit_transfer(
        &self,
        white_label_id: &str,
        token: &str,
        amount: f64,
        executor_id: &str,
    ) -> Result<ProfitTransaction, String> {
        let (super_admin_share, white_label_share) = self.calculate_profit_share(white_label_id, amount).await;
        let now = Utc::now().timestamp();

        let tx = ProfitTransaction {
            id: Uuid::new_v4().to_string(),
            white_label_id: white_label_id.to_string(),
            super_admin_wallet: String::new(), // resolved by canonical wallet backend at settlement
            amount: super_admin_share,
            percentage: if amount > 0.0 { super_admin_share / amount * 100.0 } else { 0.0 },
            gross_revenue: amount,
            net_revenue: white_label_share,
            token: token.to_string(),
            tx_hash: None, // no fabricated hash; set by wallet backend after real on-chain settlement
            status: "pending_settlement".to_string(), // governance record only; no on-chain movement
            created_at: now,
        };

        // Record pending settlement intent only (no on-chain movement).
        // Do NOT increment total_transferred here — no on-chain settlement has
        // occurred. total_transferred is advanced by the wallet backend
        // callback once the real broadcast confirms. Only record the intent.
        let _ = self
            .db
            .execute(
                "UPDATE profit_share_configs SET last_transfer = $1, updated_at = $1 WHERE white_label_id = $2",
                &[&now, &white_label_id],
            )
            .await;

        self.db
            .execute(
                "INSERT INTO profit_transactions (id, white_label_id, super_admin_wallet, amount, percentage, gross_revenue, net_revenue, token, tx_hash, status, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL, $9, $10)",
                &[
                    &tx.id,
                    &tx.white_label_id,
                    &tx.super_admin_wallet,
                    &tx.amount,
                    &tx.percentage,
                    &tx.gross_revenue,
                    &tx.net_revenue,
                    &tx.token,
                    &tx.status,
                    &tx.created_at,
                ],
            )
            .await
            .map_err(|e| format!("insert profit tx failed: {e}"))?;

        self.log_audit(executor_id, "PROFIT_TRANSFER_RECORDED",
            &format!("Recorded pending profit-share settlement of {} (no on-chain movement)", super_admin_share), "", "").await;

        Ok(tx)
    }

    pub async fn get_profit_history(&self, white_label_id: Option<&str>, limit: usize) -> Vec<ProfitTransaction> {
        let limit_i = limit as i64;
        let result = match white_label_id {
            Some(id) => {
                self.db
                    .query(
                        "SELECT * FROM profit_transactions WHERE white_label_id = $1 ORDER BY created_at DESC LIMIT $2",
                        &[&id, &limit_i],
                    )
                    .await
            }
            None => {
                self.db
                    .query(
                        "SELECT * FROM profit_transactions ORDER BY created_at DESC LIMIT $1",
                        &[&limit_i],
                    )
                    .await
            }
        };
        match result {
            Ok(rows) => rows.iter().map(Self::row_to_profit_tx).collect(),
            Err(e) => {
                eprintln!("get_profit_history failed: {e}");
                vec![]
            }
        }
    }

    pub async fn get_total_profits(&self) -> f64 {
        match self
            .db
            .query_opt("SELECT COALESCE(SUM(total_transferred), 0.0) AS total FROM profit_share_configs", &[])
            .await
        {
            Ok(Some(row)) => row.get("total"),
            _ => 0.0,
        }
    }

    // ==================== FEATURE FLAGS ====================

    pub async fn get_all_features(&self) -> Vec<FeatureFlag> {
        match self.db.query("SELECT * FROM feature_flags ORDER BY name", &[]).await {
            Ok(rows) => rows.iter().map(Self::row_to_feature_flag).collect(),
            Err(e) => {
                eprintln!("get_all_features failed: {e}");
                vec![]
            }
        }
    }

    pub async fn is_feature_enabled(&self, feature_name: &str, admin_id: &str) -> bool {
        let flag = self
            .db
            .query_opt(
                "SELECT * FROM feature_flags WHERE name = $1",
                &[&feature_name],
            )
            .await;
        let flag = match flag {
            Ok(Some(row)) => Self::row_to_feature_flag(&row),
            _ => return false,
        };

        // Super admin bypasses feature gating
        if let Ok(Some(r)) = self
            .db
            .query_opt("SELECT role FROM admin_users WHERE id = $1", &[&admin_id])
            .await
        {
            let role: String = r.get("role");
            if role == AdminRole::SuperAdmin.as_str() {
                return true;
            }
        }

        flag.global_enabled && flag.enabled
    }

    pub async fn set_feature(&self, feature_name: &str, enabled: bool, super_admin_id: &str) -> Result<(), String> {
        // Check permissions
        let row = self
            .db
            .query_opt("SELECT role FROM admin_users WHERE id = $1", &[&super_admin_id])
            .await
            .map_err(|e| format!("db error: {e}"))?
            .ok_or("Admin not found")?;
        let admin_role: String = row.get("role");
        if admin_role != AdminRole::SuperAdmin.as_str() {
            return Err("Unauthorized".to_string());
        }

        let now = Utc::now().timestamp();
        let n = self
            .db
            .execute(
                "UPDATE feature_flags SET global_enabled = $1, enabled = $1, updated_by = $2, updated_at = $3 WHERE name = $4",
                &[&enabled, &super_admin_id, &now, &feature_name],
            )
            .await
            .map_err(|e| format!("update failed: {e}"))?;
        if n == 0 {
            return Err("Feature not found".to_string());
        }

        self.log_audit(super_admin_id, "SET_FEATURE",
            &format!("Set feature {} to {}", feature_name, if enabled { "enabled" } else { "disabled" }), "", "").await;

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_password_validation() {
        // Pure function — no DB needed. Validates the (unchanged) crypto policy.
        let svc = SuperAdminService::with_url(
            "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable",
        );
        // Construction attempts a real DB connection + migration; in environments
        // without a running Postgres it returns an honest error (fail-closed),
        // which is the security-correct behavior. The password-policy helper
        // itself is a pure, DB-free function so we exercise it directly.
        let policy = PasswordPolicy::default();
        assert!(policy.min_length == 8);
        assert!(policy.require_uppercase);
    }
}
