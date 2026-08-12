//! TigerWallet MasterWallet Fetchers - Rust High-Speed Implementation
//!
//! This module implements fetchers specifically for MasterWallet apps
//! - Multi-user wallet management
//! - Auto-sign operations
//! - Sub-wallet control
//! - Real PostgreSQL and Redis integration
//! - NO user wallet operations (those are in userwallet_fetchers)
//! - NO admin operations (those are in admin_fetchers)

use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH, Duration};
use serde::{Deserialize, Serialize};
use serde_json::json;
use tokio::runtime::Runtime;
use tokio_postgres::{NoTls, Client, Row};
use redis::{Client as RedisClient, Commands, Connection};

pub mod types;
pub mod fetchers;
pub mod database;
pub mod blockchain;
pub mod cache;

pub use types::*;
pub use fetchers::*;
pub use database::*;
pub use blockchain::*;
pub use cache::*;

/// MasterWallet fetcher manager - only includes master wallet operations
pub struct MasterWalletFetcherManager {
    fetchers: HashMap<String, Arc<dyn MasterFetcher>>,
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl MasterWalletFetcherManager {
    pub fn new(db_config: &DatabaseConfig, redis_config: &RedisConfig) -> Result<Self, String> {
        let db_pool = Arc::new(DatabasePool::new(db_config)?);
        let cache = Arc::new(CacheManager::new(redis_config)?);

        let mut fetchers = HashMap::new();

        // Master wallet operations (different from user wallet)
        fetchers.insert("subwallets".to_string(), Arc::new(SubWalletFetcher::new(db_pool.clone(), cache.clone())) as Arc<dyn MasterFetcher>);
        fetchers.insert("auto_sign".to_string(), Arc::new(AutoSignFetcher::new(db_pool.clone(), cache.clone())) as Arc<dyn MasterFetcher>);
        fetchers.insert("sign_approval".to_string(), Arc::new(SignApprovalFetcher::new(db_pool.clone(), cache.clone())) as Arc<dyn MasterFetcher>);
        fetchers.insert("user_management".to_string(), Arc::new(UserManagementFetcher::new(db_pool.clone(), cache.clone())) as Arc<dyn MasterFetcher>);
        fetchers.insert("volume".to_string(), Arc::new(VolumeFetcher::new(db_pool.clone(), cache.clone())) as Arc<dyn MasterFetcher>);
        fetchers.insert("master_analytics".to_string(), Arc::new(MasterAnalyticsFetcher::new(db_pool.clone(), cache.clone())) as Arc<dyn MasterFetcher>);
        fetchers.insert("permissions".to_string(), Arc::new(PermissionsFetcher::new(db_pool.clone(), cache.clone())) as Arc<dyn MasterFetcher>);
        fetchers.insert("whitelist".to_string(), Arc::new(WhitelistFetcher::new(db_pool.clone(), cache.clone())) as Arc<dyn MasterFetcher>);

        Ok(Self {
            fetchers,
            db_pool,
            cache,
        })
    }

    pub fn get_fetcher(&self, name: &str) -> Option<Arc<dyn MasterFetcher>> {
        self.fetchers.get(name).cloned()
    }

    pub fn get_all_fetchers(&self) -> Vec<String> {
        self.fetchers.keys().cloned().collect()
    }
}

// Master fetcher trait
pub trait MasterFetcher: Send + Sync {
    fn name(&self) -> &str;
    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String>;
    fn initialize(&self) -> Result<(), String>;
}

// Database configuration
#[derive(Debug, Clone)]
pub struct DatabaseConfig {
    pub host: String,
    pub port: u16,
    pub database: String,
    pub username: String,
    pub password: String,
    pub max_connections: u32,
}

impl DatabaseConfig {
    pub fn new(host: &str, port: u16, database: &str, username: &str, password: &str) -> Self {
        Self {
            host: host.to_string(),
            port,
            database: database.to_string(),
            username: username.to_string(),
            password: password.to_string(),
            max_connections: 20,
        }
    }

    pub fn connection_string(&self) -> String {
        format!(
            "host={} port={} dbname={} user={} password={} max_connections={}",
            self.host, self.port, self.database, self.username, self.password, self.max_connections
        )
    }
}

// Redis configuration
#[derive(Debug, Clone)]
pub struct RedisConfig {
    pub host: String,
    pub port: u16,
    pub password: Option<String>,
    pub db: u8,
}

impl RedisConfig {
    pub fn new(host: &str, port: u16) -> Self {
        Self {
            host: host.to_string(),
            port,
            password: None,
            db: 0,
        }
    }

    pub fn with_auth(host: &str, port: u16, password: &str) -> Self {
        Self {
            host: host.to_string(),
            port,
            password: Some(password.to_string()),
            db: 0,
        }
    }
}

// Database pool manager
pub struct DatabasePool {
    config: DatabaseConfig,
    runtime: Runtime,
}

impl DatabasePool {
    pub fn new(config: &DatabaseConfig) -> Result<Self, String> {
        Ok(Self {
            config: config.clone(),
            runtime: Runtime::new().map_err(|e| format!("Failed to create runtime: {}", e))?,
        })
    }

    pub fn get_connection(&self) -> Result<Client, String> {
        self.runtime.block_on(async {
            tokio_postgres::connect(&self.config.connection_string(), NoTls)
                .await
                .map_err(|e| format!("Database connection failed: {}", e))
                .map(|(client, _)| client)
        })
    }

    pub fn execute(&self, query: &str, params: &[&(dyn tokio_postgres::types::ToSql + Sync)]) -> Result<u64, String> {
        let client = self.get_connection()?;
        self.runtime.block_on(async {
            client.execute(query, params)
                .await
                .map_err(|e| format!("Query execution failed: {}", e))
        })
    }

    pub fn query(&self, query: &str, params: &[&(dyn tokio_postgres::types::ToSql + Sync)]) -> Result<Vec<Row>, String> {
        let client = self.get_connection()?;
        self.runtime.block_on(async {
            client.query(query, params)
                .await
                .map_err(|e| format!("Query failed: {}", e))
        })
    }
}

// Cache manager for Redis — Connection commands require &mut self, so we
// guard the connection with a Mutex (CacheManager is shared behind an Arc).
pub struct CacheManager {
    client: std::sync::Mutex<redis::Connection>,
    default_ttl: u64,
}

impl CacheManager {
    pub fn new(config: &RedisConfig) -> Result<Self, String> {
        let redis_url = if let Some(ref password) = config.password {
            format!("redis://:{}@{}:{}/{}", password, config.host, config.port, config.db)
        } else {
            format!("redis://{}:{}/{}", config.host, config.port, config.db)
        };

        let conn = redis::Client::open(redis_url.as_str())
            .map_err(|e| format!("Redis connection failed: {}", e))?
            .get_connection()
            .map_err(|e| format!("Redis get connection failed: {}", e))?;

        Ok(Self {
            client: std::sync::Mutex::new(conn),
            default_ttl: 300, // 5 minutes default
        })
    }

    pub fn get(&self, key: &str) -> Result<Option<String>, String> {
        let mut con = self.client.lock().map_err(|e| format!("Cache lock failed: {}", e))?;
        let val: Option<String> = redis::Commands::get(&mut *con, key)
            .map_err(|e| format!("Cache get failed: {}", e))?;
        Ok(val)
    }

    pub fn set(&self, key: &str, value: &str) -> Result<(), String> {
        let mut con = self.client.lock().map_err(|e| format!("Cache lock failed: {}", e))?;
        redis::Commands::set::<&str, &str, ()>(&mut *con, key, value)
            .map_err(|e| format!("Cache set failed: {}", e))
    }

    pub fn set_with_ttl(&self, key: &str, value: &str, ttl: u64) -> Result<(), String> {
        let mut con = self.client.lock().map_err(|e| format!("Cache lock failed: {}", e))?;
        redis::Commands::set_ex(&mut *con, key, value, ttl)
            .map_err(|e| format!("Cache set_ex failed: {}", e))
    }

    pub fn delete(&self, key: &str) -> Result<(), String> {
        let mut con = self.client.lock().map_err(|e| format!("Cache lock failed: {}", e))?;
        redis::Commands::del::<&str, ()>(&mut *con, key)
            .map_err(|e| format!("Cache delete failed: {}", e))
    }

    pub fn exists(&self, key: &str) -> Result<bool, String> {
        let mut con = self.client.lock().map_err(|e| format!("Cache lock failed: {}", e))?;
        let val: bool = redis::Commands::exists(&mut *con, key)
            .map_err(|e| format!("Cache exists failed: {}", e))?;
        Ok(val)
    }

    pub fn incr(&self, key: &str) -> Result<i64, String> {
        let mut con = self.client.lock().map_err(|e| format!("Cache lock failed: {}", e))?;
        let val: i64 = redis::Commands::incr(&mut *con, key, 1)
            .map_err(|e| format!("Cache incr failed: {}", e))?;
        Ok(val)
    }

    pub fn expire(&self, key: &str, ttl: u64) -> Result<(), String> {
        let mut con = self.client.lock().map_err(|e| format!("Cache lock failed: {}", e))?;
        redis::Commands::expire(&mut *con, key, ttl as i64)
            .map_err(|e| format!("Cache expire failed: {}", e))
    }

    pub fn hset(&self, key: &str, field: &str, value: &str) -> Result<(), String> {
        let mut con = self.client.lock().map_err(|e| format!("Cache lock failed: {}", e))?;
        redis::Commands::hset::<&str, &str, &str, ()>(&mut *con, key, field, value)
            .map_err(|e| format!("Cache hset failed: {}", e))
    }

    pub fn hget(&self, key: &str, field: &str) -> Result<Option<String>, String> {
        let mut con = self.client.lock().map_err(|e| format!("Cache lock failed: {}", e))?;
        let val: Option<String> = redis::Commands::hget(&mut *con, key, field)
            .map_err(|e| format!("Cache hget failed: {}", e))?;
        Ok(val)
    }

    pub fn hgetall(&self, key: &str) -> Result<HashMap<String, String>, String> {
        let mut con = self.client.lock().map_err(|e| format!("Cache lock failed: {}", e))?;
        let val: HashMap<String, String> = redis::Commands::hgetall(&mut *con, key)
            .map_err(|e| format!("Cache hgetall failed: {}", e))?;
        Ok(val)
    }
}

// SubWallet Fetcher - Manages sub-wallets under master
pub struct SubWalletFetcher {
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl SubWalletFetcher {
    pub fn new(db_pool: Arc<DatabasePool>, cache: Arc<CacheManager>) -> Self {
        Self { db_pool, cache }
    }
}

impl MasterFetcher for SubWalletFetcher {
    fn name(&self) -> &str { "subwallets" }

    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String> {
        let master_wallet_id = params.get("master_wallet_id").cloned().unwrap_or_default();
        let cache_key = format!("subwallets:{}", master_wallet_id);

        // Try cache first
        if let Ok(Some(cached)) = self.cache.get(&cache_key) {
            return serde_json::from_str(&cached).map_err(|e| format!("Cache parse failed: {}", e));
        }

        // Query from database
        let query = r#"
            SELECT sw.id, sw.master_wallet_id, sw.name, sw.address, sw.address_type,
                   sw.is_active, sw.created_at, sw.updated_at,
                   COALESCE(SUM(wt.balance), 0) as total_balance,
                   COUNT(DISTINCT wu.id) as user_count
            FROM sub_wallets sw
            LEFT JOIN wallet_tokens wt ON wt.wallet_id = sw.id
            LEFT JOIN wallet_users wu ON wu.wallet_id = sw.id
            WHERE sw.master_wallet_id = $1
            GROUP BY sw.id
            ORDER BY sw.created_at DESC
        "#;

        let rows = self.db_pool.query(query, &[&master_wallet_id])?;

        let subwallets: Vec<serde_json::Value> = rows.iter().map(|row| {
            json!({
                "id": row.get::<_, String>("id"),
                "master_wallet_id": row.get::<_, String>("master_wallet_id"),
                "name": row.get::<_, String>("name"),
                "address": row.get::<_, String>("address"),
                "address_type": row.get::<_, String>("address_type"),
                "is_active": row.get::<_, bool>("is_active"),
                "total_balance": row.get::<_, f64>("total_balance"),
                "user_count": row.get::<_, i64>("user_count"),
                "created_at": row.get::<_, i64>("created_at"),
                "updated_at": row.get::<_, i64>("updated_at")
            })
        }).collect();

        // Calculate total volume
        let total_volume: f64 = subwallets.iter()
            .filter_map(|w| w.get("total_balance").and_then(|v| v.as_f64()))
            .sum();

        let result = json!({
            "subWallets": subwallets,
            "totalCount": rows.len(),
            "totalVolume": total_volume.to_string(),
            "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
        });

        // Cache result
        if let Ok(json_str) = serde_json::to_string(&result) {
            let _ = self.cache.set_with_ttl(&cache_key, &json_str, 60);
        }

        Ok(result)
    }

    fn initialize(&self) -> Result<(), String> {
        // Create tables if not exist
        let create_table = r#"
            CREATE TABLE IF NOT EXISTS sub_wallets (
                id VARCHAR(64) PRIMARY KEY,
                master_wallet_id VARCHAR(64) NOT NULL,
                name VARCHAR(255) NOT NULL,
                address VARCHAR(128) NOT NULL,
                address_type VARCHAR(32) NOT NULL DEFAULT 'EVM',
                public_key VARCHAR(256),
                encrypted_private_key TEXT,
                is_active BOOLEAN DEFAULT true,
                created_at BIGINT NOT NULL,
                updated_at BIGINT NOT NULL,
                FOREIGN KEY (master_wallet_id) REFERENCES master_wallets(id)
            );

            CREATE INDEX IF NOT EXISTS idx_subwallets_master ON sub_wallets(master_wallet_id);
            CREATE INDEX IF NOT EXISTS idx_subwallets_address ON sub_wallets(address);
        "#;

        self.db_pool.execute(create_table, &[])?;
        Ok(())
    }
}

// AutoSign Fetcher - Automatic transaction signing
pub struct AutoSignFetcher {
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl AutoSignFetcher {
    pub fn new(db_pool: Arc<DatabasePool>, cache: Arc<CacheManager>) -> Self {
        Self { db_pool, cache }
    }
}

impl MasterFetcher for AutoSignFetcher {
    fn name(&self) -> &str { "auto_sign" }

    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String> {
        let master_wallet_id = params.get("master_wallet_id").cloned().unwrap_or_default();
        let cache_key = format!("autosign:rules:{}", master_wallet_id);

        // Try cache first
        if let Ok(Some(cached)) = self.cache.get(&cache_key) {
            return serde_json::from_str(&cached).map_err(|e| format!("Cache parse failed: {}", e));
        }

        let query = r#"
            SELECT id, master_wallet_id, name, max_amount, chain_ids,
                   token_ids, enabled, conditions, created_at, updated_at
            FROM auto_sign_rules
            WHERE master_wallet_id = $1
            ORDER BY created_at DESC
        "#;

        let rows = self.db_pool.query(query, &[&master_wallet_id])?;

        let rules: Vec<serde_json::Value> = rows.iter().map(|row| {
            json!({
                "id": row.get::<_, String>("id"),
                "master_wallet_id": row.get::<_, String>("master_wallet_id"),
                "name": row.get::<_, String>("name"),
                "maxAmount": row.get::<_, String>("max_amount"),
                "chainIds": row.get::<_, Vec<String>>("chain_ids"),
                "tokenIds": row.get::<_, Vec<String>>("token_ids"),
                "enabled": row.get::<_, bool>("enabled"),
                "conditions": row.get::<_, Vec<String>>("conditions"),
                "createdAt": row.get::<_, i64>("created_at"),
                "updatedAt": row.get::<_, i64>("updated_at")
            })
        }).collect();

        let result = json!({
            "rules": rules,
            "enabled": !rules.is_empty(),
            "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
        });

        // Cache result
        if let Ok(json_str) = serde_json::to_string(&result) {
            let _ = self.cache.set_with_ttl(&cache_key, &json_str, 120);
        }

        Ok(result)
    }

    fn initialize(&self) -> Result<(), String> {
        let create_table = r#"
            CREATE TABLE IF NOT EXISTS auto_sign_rules (
                id VARCHAR(64) PRIMARY KEY,
                master_wallet_id VARCHAR(64) NOT NULL,
                name VARCHAR(255) NOT NULL,
                max_amount VARCHAR(64) NOT NULL,
                chain_ids TEXT[] DEFAULT '{}',
                token_ids TEXT[] DEFAULT '{}',
                enabled BOOLEAN DEFAULT true,
                conditions TEXT[] DEFAULT '{}',
                created_at BIGINT NOT NULL,
                updated_at BIGINT NOT NULL,
                FOREIGN KEY (master_wallet_id) REFERENCES master_wallets(id)
            );

            CREATE INDEX IF NOT EXISTS idx_autosign_master ON auto_sign_rules(master_wallet_id);
        "#;

        self.db_pool.execute(create_table, &[])?;
        Ok(())
    }
}

// Sign Approval Fetcher - Approve/reject transactions
pub struct SignApprovalFetcher {
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl SignApprovalFetcher {
    pub fn new(db_pool: Arc<DatabasePool>, cache: Arc<CacheManager>) -> Self {
        Self { db_pool, cache }
    }
}

impl MasterFetcher for SignApprovalFetcher {
    fn name(&self) -> &str { "sign_approval" }

    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String> {
        let master_wallet_id = params.get("master_wallet_id").cloned().unwrap_or_default();
        let status = params.get("status").cloned().unwrap_or_else(|| "PENDING".to_string());

        let query = r#"
            SELECT id, master_wallet_id, sub_wallet_id, tx_hash, from_address,
                   to_address, amount, token_id, chain_id, status,
                   approved_by, rejected_by, reject_reason, created_at, updated_at
            FROM transaction_approvals
            WHERE master_wallet_id = $1 AND status = $2
            ORDER BY created_at DESC
            LIMIT 100
        "#;

        let rows = self.db_pool.query(query, &[&master_wallet_id, &status])?;

        let transactions: Vec<serde_json::Value> = rows.iter().map(|row| {
            json!({
                "id": row.get::<_, String>("id"),
                "masterWalletId": row.get::<_, String>("master_wallet_id"),
                "subWalletId": row.get::<_, String>("sub_wallet_id"),
                "txHash": row.get::<_, String>("tx_hash"),
                "from": row.get::<_, String>("from_address"),
                "to": row.get::<_, String>("to_address"),
                "amount": row.get::<_, String>("amount"),
                "tokenId": row.get::<_, String>("token_id"),
                "chainId": row.get::<_, i64>("chain_id"),
                "status": row.get::<_, String>("status"),
                "approvedBy": row.get::<_, Option<String>>("approved_by"),
                "rejectedBy": row.get::<_, Option<String>>("rejected_by"),
                "rejectReason": row.get::<_, Option<String>>("reject_reason"),
                "createdAt": row.get::<_, i64>("created_at"),
                "updatedAt": row.get::<_, i64>("updated_at")
            })
        }).collect();

        // Get counts for all statuses
        let pending_count: i64 = self.db_pool.query(
            "SELECT COUNT(*) as count FROM transaction_approvals WHERE master_wallet_id = $1 AND status = 'PENDING'",
            &[&master_wallet_id]
        )?.first().and_then(|r| Some(r.get::<_, i64>("count"))).unwrap_or(0);

        let approved_count: i64 = self.db_pool.query(
            "SELECT COUNT(*) as count FROM transaction_approvals WHERE master_wallet_id = $1 AND status = 'APPROVED'",
            &[&master_wallet_id]
        )?.first().and_then(|r| Some(r.get::<_, i64>("count"))).unwrap_or(0);

        let rejected_count: i64 = self.db_pool.query(
            "SELECT COUNT(*) as count FROM transaction_approvals WHERE master_wallet_id = $1 AND status = 'REJECTED'",
            &[&master_wallet_id]
        )?.first().and_then(|r| Some(r.get::<_, i64>("count"))).unwrap_or(0);

        Ok(json!({
            "pending": transactions.iter().filter(|t| t.get("status").map_or(false, |s| s == "PENDING")).collect::<Vec<_>>(),
            "approved": transactions.iter().filter(|t| t.get("status").map_or(false, |s| s == "APPROVED")).collect::<Vec<_>>(),
            "rejected": transactions.iter().filter(|t| t.get("status").map_or(false, |s| s == "REJECTED")).collect::<Vec<_>>(),
            "counts": {
                "pending": pending_count,
                "approved": approved_count,
                "rejected": rejected_count
            },
            "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
        }))
    }

    fn initialize(&self) -> Result<(), String> {
        let create_table = r#"
            CREATE TABLE IF NOT EXISTS transaction_approvals (
                id VARCHAR(64) PRIMARY KEY,
                master_wallet_id VARCHAR(64) NOT NULL,
                sub_wallet_id VARCHAR(64) NOT NULL,
                tx_hash VARCHAR(128) NOT NULL,
                from_address VARCHAR(64) NOT NULL,
                to_address VARCHAR(64) NOT NULL,
                amount VARCHAR(64) NOT NULL,
                token_id VARCHAR(64),
                chain_id BIGINT NOT NULL,
                status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
                approved_by VARCHAR(64),
                rejected_by VARCHAR(64),
                reject_reason TEXT,
                created_at BIGINT NOT NULL,
                updated_at BIGINT NOT NULL,
                FOREIGN KEY (master_wallet_id) REFERENCES master_wallets(id),
                FOREIGN KEY (sub_wallet_id) REFERENCES sub_wallets(id)
            );

            CREATE INDEX IF NOT EXISTS idx_txapproval_master ON transaction_approvals(master_wallet_id);
            CREATE INDEX IF NOT EXISTS idx_txapproval_status ON transaction_approvals(status);
        "#;

        self.db_pool.execute(create_table, &[])?;
        Ok(())
    }
}

// User Management Fetcher - Manage sub-wallet users
pub struct UserManagementFetcher {
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl UserManagementFetcher {
    pub fn new(db_pool: Arc<DatabasePool>, cache: Arc<CacheManager>) -> Self {
        Self { db_pool, cache }
    }
}

impl MasterFetcher for UserManagementFetcher {
    fn name(&self) -> &str { "user_management" }

    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String> {
        let master_wallet_id = params.get("master_wallet_id").cloned().unwrap_or_default();
        let wallet_id = params.get("wallet_id");

        let query = if let Some(wid) = wallet_id {
            r#"
                SELECT wu.id, wu.wallet_id, wu.user_id, wu.email, wu.name,
                       wu.role, wu.permissions, wu.is_active, wu.created_at, wu.updated_at
                FROM wallet_users wu
                JOIN sub_wallets sw ON sw.id = wu.wallet_id
                WHERE sw.master_wallet_id = $1 AND wu.wallet_id = $2
                ORDER BY wu.created_at DESC
            "#
        } else {
            r#"
                SELECT wu.id, wu.wallet_id, wu.user_id, wu.email, wu.name,
                       wu.role, wu.permissions, wu.is_active, wu.created_at, wu.updated_at
                FROM wallet_users wu
                JOIN sub_wallets sw ON sw.id = wu.wallet_id
                WHERE sw.master_wallet_id = $1
                ORDER BY wu.created_at DESC
            "#
        };

        let rows = if let Some(wid) = wallet_id {
            self.db_pool.query(query, &[&master_wallet_id, &wid])?
        } else {
            self.db_pool.query(query, &[&master_wallet_id])?
        };

        let users: Vec<serde_json::Value> = rows.iter().map(|row| {
            json!({
                "id": row.get::<_, String>("id"),
                "walletId": row.get::<_, String>("wallet_id"),
                "userId": row.get::<_, String>("user_id"),
                "email": row.get::<_, String>("email"),
                "name": row.get::<_, String>("name"),
                "role": row.get::<_, String>("role"),
                "permissions": row.get::<_, Vec<String>>("permissions"),
                "isActive": row.get::<_, bool>("is_active"),
                "createdAt": row.get::<_, i64>("created_at"),
                "updatedAt": row.get::<_, i64>("updated_at")
            })
        }).collect();

        Ok(json!({
            "users": users,
            "count": users.len(),
            "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
        }))
    }

    fn initialize(&self) -> Result<(), String> {
        let create_table = r#"
            CREATE TABLE IF NOT EXISTS wallet_users (
                id VARCHAR(64) PRIMARY KEY,
                wallet_id VARCHAR(64) NOT NULL,
                user_id VARCHAR(64) NOT NULL,
                email VARCHAR(255) NOT NULL,
                name VARCHAR(255),
                role VARCHAR(32) NOT NULL DEFAULT 'USER',
                permissions TEXT[] DEFAULT '{}',
                is_active BOOLEAN DEFAULT true,
                created_at BIGINT NOT NULL,
                updated_at BIGINT NOT NULL,
                FOREIGN KEY (wallet_id) REFERENCES sub_wallets(id),
                UNIQUE(wallet_id, user_id)
            );

            CREATE INDEX IF NOT EXISTS idx_walletusers_wallet ON wallet_users(wallet_id);
            CREATE INDEX IF NOT EXISTS idx_walletusers_email ON wallet_users(email);
        "#;

        self.db_pool.execute(create_table, &[])?;
        Ok(())
    }
}

// Volume Fetcher - Track wallet volumes
pub struct VolumeFetcher {
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl VolumeFetcher {
    pub fn new(db_pool: Arc<DatabasePool>, cache: Arc<CacheManager>) -> Self {
        Self { db_pool, cache }
    }
}

impl MasterFetcher for VolumeFetcher {
    fn name(&self) -> &str { "volume" }

    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String> {
        let master_wallet_id = params.get("master_wallet_id").cloned().unwrap_or_default();
        let period = params.get("period").cloned().unwrap_or_else(|| "30d".to_string());

        let cache_key = format!("volume:{}:{}", master_wallet_id, period);

        // Try cache first (shorter TTL for volume data)
        if let Ok(Some(cached)) = self.cache.get(&cache_key) {
            return serde_json::from_str(&cached).map_err(|e| format!("Cache parse failed: {}", e));
        }

        // Calculate time range
        let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
        let period_seconds: u64 = match period.as_str() {
            "1d" => 86400,
            "7d" => 604800,
            "30d" => 2592000,
            "90d" => 7776000,
            "1y" => 31536000,
            _ => 2592000,
        };
        let start_time = now - period_seconds;

        // Query total volume
        let total_query = r#"
            SELECT COALESCE(SUM(amount), 0) as total
            FROM transaction_approvals
            WHERE master_wallet_id = $1 AND status = 'APPROVED'
            AND created_at >= $2
        "#;

        let total_volume: f64 = self.db_pool.query(total_query, &[&master_wallet_id, &(start_time as i64)])?
            .first()
            .and_then(|r| Some(r.get::<_, f64>("total")))
            .unwrap_or(0.0);

        // Query daily volume
        let daily_query = r#"
            SELECT DATE_TRUNC('day', to_timestamp(created_at)) as day,
                   SUM(amount)::double precision as daily_total
            FROM transaction_approvals
            WHERE master_wallet_id = $1 AND status = 'APPROVED'
            AND created_at >= $2
            GROUP BY day
            ORDER BY day DESC
            LIMIT 30
        "#;

        let daily_rows = self.db_pool.query(daily_query, &[&master_wallet_id, &(start_time as i64)])?;

        let daily_volumes: Vec<serde_json::Value> = daily_rows.iter().map(|row| {
            json!({
                "date": row.get::<_, String>("day"),
                "volume": row.get::<_, f64>("daily_total")
            })
        }).collect();

        // Calculate monthly volume
        let monthly_volume: f64 = daily_volumes.iter()
            .filter_map(|v| v.get("volume").and_then(|x| x.as_f64()))
            .sum();

        // Get transaction count
        let tx_count_query = r#"
            SELECT COUNT(*) as count
            FROM transaction_approvals
            WHERE master_wallet_id = $1 AND status = 'APPROVED'
            AND created_at >= $2
        "#;

        let tx_count: i64 = self.db_pool.query(tx_count_query, &[&master_wallet_id, &(start_time as i64)])?
            .first()
            .and_then(|r| Some(r.get::<_, i64>("count")))
            .unwrap_or(0);

        let result = json!({
            "totalVolume": total_volume.to_string(),
            "dailyVolume": daily_volumes.first().and_then(|v| v.get("volume").and_then(|x| x.as_f64())).unwrap_or(0.0).to_string(),
            "monthlyVolume": monthly_volume.to_string(),
            "dailyBreakdown": daily_volumes,
            "transactionCount": tx_count,
            "period": period,
            "timestamp": now
        });

        // Cache result
        if let Ok(json_str) = serde_json::to_string(&result) {
            let _ = self.cache.set_with_ttl(&cache_key, &json_str, 60);
        }

        Ok(result)
    }

    fn initialize(&self) -> Result<(), String> {
        // Volume uses transaction_approvals table which is created by SignApprovalFetcher
        Ok(())
    }
}

// Master Analytics Fetcher
pub struct MasterAnalyticsFetcher {
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl MasterAnalyticsFetcher {
    pub fn new(db_pool: Arc<DatabasePool>, cache: Arc<CacheManager>) -> Self {
        Self { db_pool, cache }
    }
}

impl MasterFetcher for MasterAnalyticsFetcher {
    fn name(&self) -> &str { "master_analytics" }

    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String> {
        let master_wallet_id = params.get("master_wallet_id").cloned().unwrap_or_default();
        let cache_key = format!("analytics:{}", master_wallet_id);

        // Try cache first
        if let Ok(Some(cached)) = self.cache.get(&cache_key) {
            return serde_json::from_str(&cached).map_err(|e| format!("Cache parse failed: {}", e));
        }

        let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
        let day_ago = now - 86400;
        let week_ago = now - 604800;
        let month_ago = now - 2592000;

        // Get wallet count
        let wallet_count: i64 = self.db_pool.query(
            "SELECT COUNT(*) as count FROM sub_wallets WHERE master_wallet_id = $1 AND is_active = true",
            &[&master_wallet_id]
        )?.first().and_then(|r| Some(r.get::<_, i64>("count"))).unwrap_or(0);

        // Get user count
        let user_count_query = r#"
            SELECT COUNT(DISTINCT wu.id) as count
            FROM wallet_users wu
            JOIN sub_wallets sw ON sw.id = wu.wallet_id
            WHERE sw.master_wallet_id = $1 AND wu.is_active = true
        "#;
        let user_count: i64 = self.db_pool.query(user_count_query, &[&master_wallet_id])?
            .first()
            .and_then(|r| Some(r.get::<_, i64>("count")))
            .unwrap_or(0);

        // Get active sub-wallets (with activity in last 24h)
        let active_wallets_query = r#"
            SELECT COUNT(DISTINCT sub_wallet_id) as count
            FROM transaction_approvals
            WHERE master_wallet_id = $1 AND created_at >= $2
        "#;
        let active_wallets: i64 = self.db_pool.query(active_wallets_query, &[&master_wallet_id, &(day_ago as i64)])?
            .first()
            .and_then(|r| Some(r.get::<_, i64>("count")))
            .unwrap_or(0);

        // Get new wallets this week
        let new_wallets_query = r#"
            SELECT COUNT(*) as count
            FROM sub_wallets
            WHERE master_wallet_id = $1 AND created_at >= $2
        "#;
        let new_wallets: i64 = self.db_pool.query(new_wallets_query, &[&master_wallet_id, &(week_ago as i64)])?
            .first()
            .and_then(|r| Some(r.get::<_, i64>("count")))
            .unwrap_or(0);

        // Get volume stats
        let volume_stats_query = r#"
            SELECT
                COALESCE(SUM(CASE WHEN created_at >= $2 THEN amount ELSE 0 END), 0) as daily_volume,
                COALESCE(SUM(CASE WHEN created_at >= $3 THEN amount ELSE 0 END), 0) as weekly_volume,
                COALESCE(SUM(amount), 0) as total_volume,
                COUNT(*) as total_txs
            FROM transaction_approvals
            WHERE master_wallet_id = $1 AND status = 'APPROVED'
        "#;

        let volume_rows = self.db_pool.query(volume_stats_query, &[&master_wallet_id, &(day_ago as i64), &(week_ago as i64)])?;
        let volume_row = volume_rows.first();

        let daily_volume: f64 = volume_row.and_then(|r| Some(r.get::<_, f64>("daily_volume"))).unwrap_or(0.0);
        let weekly_volume: f64 = volume_row.and_then(|r| Some(r.get::<_, f64>("weekly_volume"))).unwrap_or(0.0);
        let total_volume: f64 = volume_row.and_then(|r| Some(r.get::<_, f64>("total_volume"))).unwrap_or(0.0);
        let total_txs: i64 = volume_row.and_then(|r| Some(r.get::<_, i64>("total_txs"))).unwrap_or(0);

        // Calculate growth
        let growth = if total_volume > 0.0 {
            ((daily_volume - (total_volume / 30.0)) / (total_volume / 30.0) * 100.0).to_string()
        } else {
            "0".to_string()
        };

        let result = json!({
            "walletCount": wallet_count,
            "userCount": user_count,
            "activeWallets": active_wallets,
            "newWalletsThisWeek": new_wallets,
            "volume": {
                "daily": daily_volume.to_string(),
                "weekly": weekly_volume.to_string(),
                "total": total_volume.to_string()
            },
            "transactions": {
                "total": total_txs
            },
            "growth": format!("{}%", growth),
            "timestamp": now
        });

        // Cache result
        if let Ok(json_str) = serde_json::to_string(&result) {
            let _ = self.cache.set_with_ttl(&cache_key, &json_str, 120);
        }

        Ok(result)
    }

    fn initialize(&self) -> Result<(), String> {
        Ok(())
    }
}

// Permissions Fetcher
pub struct PermissionsFetcher {
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl PermissionsFetcher {
    pub fn new(db_pool: Arc<DatabasePool>, cache: Arc<CacheManager>) -> Self {
        Self { db_pool, cache }
    }
}

impl MasterFetcher for PermissionsFetcher {
    fn name(&self) -> &str { "permissions" }

    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String> {
        let master_wallet_id = params.get("master_wallet_id").cloned().unwrap_or_default();
        let user_id = params.get("user_id");

        let cache_key = if let Some(uid) = user_id {
            format!("permissions:{}:{}", master_wallet_id, uid)
        } else {
            format!("permissions:{}", master_wallet_id)
        };

        // Try cache first
        if let Ok(Some(cached)) = self.cache.get(&cache_key) {
            return serde_json::from_str(&cached).map_err(|e| format!("Cache parse failed: {}", e));
        }

        let query = if let Some(uid) = user_id {
            r#"
                SELECT wu.id, wu.user_id, wu.email, wu.role, wu.permissions
                FROM wallet_users wu
                JOIN sub_wallets sw ON sw.id = wu.wallet_id
                WHERE sw.master_wallet_id = $1 AND wu.user_id = $2
            "#
        } else {
            r#"
                SELECT wu.id, wu.user_id, wu.email, wu.role, wu.permissions
                FROM wallet_users wu
                JOIN sub_wallets sw ON sw.id = wu.wallet_id
                WHERE sw.master_wallet_id = $1
            "#
        };

        let rows = if let Some(uid) = user_id {
            self.db_pool.query(query, &[&master_wallet_id, &uid])?
        } else {
            self.db_pool.query(query, &[&master_wallet_id])?
        };

        let permissions: Vec<serde_json::Value> = rows.iter().map(|row| {
            json!({
                "id": row.get::<_, String>("id"),
                "userId": row.get::<_, String>("user_id"),
                "email": row.get::<_, String>("email"),
                "role": row.get::<_, String>("role"),
                "permissions": row.get::<_, Vec<String>>("permissions")
            })
        }).collect();

        // Get role definitions
        let roles_query = r#"
            SELECT role_name, permissions FROM role_permissions
            WHERE master_wallet_id = $1 OR master_wallet_id IS NULL
        "#;

        let roles_rows = self.db_pool.query(roles_query, &[&master_wallet_id])?;

        let role_definitions: HashMap<String, Vec<String>> = roles_rows.iter()
            .map(|row| {
                let name: String = row.get("role_name");
                let perms: Vec<String> = row.get("permissions");
                (name, perms)
            })
            .collect();

        let result = json!({
            "userPermissions": permissions,
            "roleDefinitions": role_definitions,
            "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
        });

        // Cache result
        if let Ok(json_str) = serde_json::to_string(&result) {
            let _ = self.cache.set_with_ttl(&cache_key, &json_str, 300);
        }

        Ok(result)
    }

    fn initialize(&self) -> Result<(), String> {
        let create_table = r#"
            CREATE TABLE IF NOT EXISTS role_permissions (
                id SERIAL PRIMARY KEY,
                master_wallet_id VARCHAR(64),
                role_name VARCHAR(32) NOT NULL,
                permissions TEXT[] NOT NULL,
                description TEXT,
                created_at BIGINT NOT NULL,
                UNIQUE(master_wallet_id, role_name)
            );

            -- Insert default roles
            INSERT INTO role_permissions (master_wallet_id, role_name, permissions, description, created_at)
            VALUES
                (NULL, 'ADMIN', ARRAY['all'], 'Full admin access', EXTRACT(EPOCH FROM NOW())::bigint),
                (NULL, 'USER', ARRAY['view', 'transact'], 'Regular user', EXTRACT(EPOCH FROM NOW())::bigint),
                (NULL, 'VIEWER', ARRAY['view'], 'Read-only access', EXTRACT(EPOCH FROM NOW())::bigint)
            ON CONFLICT (master_wallet_id, role_name) DO NOTHING;
        "#;

        self.db_pool.execute(create_table, &[])?;
        Ok(())
    }
}

// Whitelist Fetcher
pub struct WhitelistFetcher {
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl WhitelistFetcher {
    pub fn new(db_pool: Arc<DatabasePool>, cache: Arc<CacheManager>) -> Self {
        Self { db_pool, cache }
    }
}

impl MasterFetcher for WhitelistFetcher {
    fn name(&self) -> &str { "whitelist" }

    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String> {
        let master_wallet_id = params.get("master_wallet_id").cloned().unwrap_or_default();
        let whitelist_type = params.get("type").cloned().unwrap_or_else(|| "address".to_string());

        let cache_key = format!("whitelist:{}:{}", master_wallet_id, whitelist_type);

        // Try cache first
        if let Ok(Some(cached)) = self.cache.get(&cache_key) {
            return serde_json::from_str(&cached).map_err(|e| format!("Cache parse failed: {}", e));
        }

        let query = r#"
            SELECT id, address, address_type, label, is_verified, added_by, created_at
            FROM address_whitelist
            WHERE master_wallet_id = $1 AND address_type = $2
            ORDER BY created_at DESC
        "#;

        let rows = self.db_pool.query(query, &[&master_wallet_id, &whitelist_type])?;

        let whitelist: Vec<serde_json::Value> = rows.iter().map(|row| {
            json!({
                "id": row.get::<_, String>("id"),
                "address": row.get::<_, String>("address"),
                "addressType": row.get::<_, String>("address_type"),
                "label": row.get::<_, Option<String>>("label"),
                "isVerified": row.get::<_, bool>("is_verified"),
                "addedBy": row.get::<_, String>("added_by"),
                "createdAt": row.get::<_, i64>("created_at")
            })
        }).collect();

        // Get count by type
        let count_query = r#"
            SELECT address_type, COUNT(*) as count
            FROM address_whitelist
            WHERE master_wallet_id = $1
            GROUP BY address_type
        "#;

        let count_rows = self.db_pool.query(count_query, &[&master_wallet_id])?;
        let mut counts = HashMap::new();
        for row in count_rows {
            let t: String = row.get("address_type");
            let c: i64 = row.get("count");
            counts.insert(t, c);
        }

        let result = json!({
            "whitelist": whitelist,
            "counts": counts,
            "type": whitelist_type,
            "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
        });

        // Cache result
        if let Ok(json_str) = serde_json::to_string(&result) {
            let _ = self.cache.set_with_ttl(&cache_key, &json_str, 300);
        }

        Ok(result)
    }

    fn initialize(&self) -> Result<(), String> {
        let create_table = r#"
            CREATE TABLE IF NOT EXISTS address_whitelist (
                id VARCHAR(64) PRIMARY KEY,
                master_wallet_id VARCHAR(64) NOT NULL,
                address VARCHAR(128) NOT NULL,
                address_type VARCHAR(32) NOT NULL,
                label TEXT,
                is_verified BOOLEAN DEFAULT false,
                added_by VARCHAR(64) NOT NULL,
                created_at BIGINT NOT NULL,
                FOREIGN KEY (master_wallet_id) REFERENCES master_wallets(id),
                UNIQUE(master_wallet_id, address)
            );

            CREATE INDEX IF NOT EXISTS idx_whitelist_master ON address_whitelist(master_wallet_id);
            CREATE INDEX IF NOT EXISTS idx_whitelist_address ON address_whitelist(address);
        "#;

        self.db_pool.execute(create_table, &[])?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_database_config() {
        let config = DatabaseConfig::new("localhost", 5432, "tigerwallet", "postgres", "password");
        assert_eq!(config.host, "localhost");
        assert_eq!(config.port, 5432);
    }

    #[test]
    fn test_redis_config() {
        let config = RedisConfig::new("localhost", 6379);
        assert_eq!(config.host, "localhost");
        assert_eq!(config.port, 6379);
    }
}
