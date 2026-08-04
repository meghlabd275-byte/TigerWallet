//! Fetchers module - Additional fetcher implementations
//! Provides extended functionality for MasterWallet operations

use crate::MasterFetcher;
use crate::DatabasePool;
use crate::CacheManager;
use std::sync::Arc;
use std::collections::HashMap;
use serde_json::json;
use std::time::{SystemTime, UNIX_EPOCH};

/// Fee management fetcher
pub struct FeeManagementFetcher {
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl FeeManagementFetcher {
    pub fn new(db_pool: Arc<DatabasePool>, cache: Arc<CacheManager>) -> Self {
        Self { db_pool, cache }
    }
}

impl MasterFetcher for FeeManagementFetcher {
    fn name(&self) -> &str { "fee_management" }
    
    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String> {
        let master_wallet_id = params.get("master_wallet_id").cloned().unwrap_or_default();
        
        let query = r#"
            SELECT id, fee_type, percentage, flat_fee, min_amount, max_amount, is_active, created_at
            FROM fee_config
            WHERE master_wallet_id = $1 AND is_active = true
        "#;
        
        let rows = self.db_pool.query(query, &[&master_wallet_id])?;
        
        let fees: Vec<serde_json::Value> = rows.iter().map(|row| {
            json!({
                "id": row.get::<_, String>("id"),
                "feeType": row.get::<_, String>("fee_type"),
                "percentage": row.get::<_, f64>("percentage"),
                "flatFee": row.get::<_, String>("flat_fee"),
                "minAmount": row.get::<_, Option<String>>("min_amount"),
                "maxAmount": row.get::<_, Option<String>>("max_amount"),
                "isActive": row.get::<_, bool>("is_active"),
                "createdAt": row.get::<_, i64>("created_at")
            })
        }).collect();
        
        Ok(json!({
            "fees": fees,
            "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
        }))
    }
    
    fn initialize(&self) -> Result<(), String> {
        let create_table = r#"
            CREATE TABLE IF NOT EXISTS fee_config (
                id VARCHAR(64) PRIMARY KEY,
                master_wallet_id VARCHAR(64) NOT NULL,
                fee_type VARCHAR(32) NOT NULL,
                percentage DECIMAL(10, 4) NOT NULL DEFAULT 0,
                flat_fee VARCHAR(32) NOT NULL DEFAULT '0',
                min_amount VARCHAR(32),
                max_amount VARCHAR(32),
                is_active BOOLEAN DEFAULT true,
                created_at BIGINT NOT NULL,
                updated_at BIGINT NOT NULL,
                FOREIGN KEY (master_wallet_id) REFERENCES master_wallets(id) ON DELETE CASCADE
            );
        "#;
        
        self.db_pool.execute(create_table, &[])?;
        Ok(())
    }
}

/// Audit log fetcher
pub struct AuditLogFetcher {
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl AuditLogFetcher {
    pub fn new(db_pool: Arc<DatabasePool>, cache: Arc<CacheManager>) -> Self {
        Self { db_pool, cache }
    }
}

impl MasterFetcher for AuditLogFetcher {
    fn name(&self) -> &str { "audit_logs" }
    
    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String> {
        let master_wallet_id = params.get("master_wallet_id").cloned().unwrap_or_default();
        let user_id = params.get("user_id");
        let action = params.get("action");
        let limit = params.get("limit").and_then(|l| l.parse().ok()).unwrap_or(100);
        
        let mut query = String::from(
            "SELECT id, master_wallet_id, user_id, action, entity_type, entity_id, details, ip_address, created_at 
             FROM audit_logs 
             WHERE master_wallet_id = $1"
        );
        
        let mut query_params: Vec<Box<dyn tokio_postgres::types::ToSql>> = vec![Box::new(master_wallet_id.clone())];
        
        if let Some(uid) = user_id {
            query.push_str(" AND user_id = $2");
            query_params.push(Box::new(uid.clone()));
        }
        
        if let Some(act) = action {
            let param_num = query_params.len() + 1;
            query.push_str(&format!(" AND action = ${}", param_num));
            query_params.push(Box::new(act.clone()));
        }
        
        query.push_str(&format!(" ORDER BY created_at DESC LIMIT {}", limit));
        
        // Note: In production, use proper parameterized queries
        let rows = self.db_pool.query(&query, &[])?;
        
        let logs: Vec<serde_json::Value> = rows.iter().map(|row| {
            json!({
                "id": row.get::<_, String>("id"),
                "masterWalletId": row.get::<_, Option<String>>("master_wallet_id"),
                "userId": row.get::<_, Option<String>>("user_id"),
                "action": row.get::<_, String>("action"),
                "entityType": row.get::<_, Option<String>>("entity_type"),
                "entityId": row.get::<_, Option<String>>("entity_id"),
                "details": row.get::<_, Option<serde_json::Value>>("details"),
                "ipAddress": row.get::<_, Option<String>>("ip_address"),
                "createdAt": row.get::<_, i64>("created_at")
            })
        }).collect();
        
        Ok(json!({
            "logs": logs,
            "count": logs.len(),
            "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
        }))
    }
    
    fn initialize(&self) -> Result<(), String> {
        // Table is created in migrations
        Ok(())
    }
}

/// Treasury management fetcher
pub struct TreasuryFetcher {
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl TreasuryFetcher {
    pub fn new(db_pool: Arc<DatabasePool>, cache: Arc<CacheManager>) -> Self {
        Self { db_pool, cache }
    }
}

impl MasterFetcher for TreasuryFetcher {
    fn name(&self) -> &str { "treasury" }
    
    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String> {
        let master_wallet_id = params.get("master_wallet_id").cloned().unwrap_or_default();
        
        // Get total balance across all sub-wallets
        let balance_query = r#"
            SELECT COALESCE(SUM(wt.balance::numeric), 0) as total_balance
            FROM wallet_tokens wt
            JOIN sub_wallets sw ON sw.id = wt.wallet_id
            WHERE sw.master_wallet_id = $1 AND sw.is_active = true
        "#;
        
        let total_balance: f64 = self.db_pool.query(balance_query, &[&master_wallet_id])
            .ok()
            .and_then(|rows| rows.first().and_then(|r| r.get::<_, f64>("total_balance").ok()))
            .unwrap_or(0.0);
        
        // Get token breakdown
        let token_query = r#"
            SELECT t.symbol, t.name, SUM(wt.balance::numeric) as balance
            FROM wallet_tokens wt
            JOIN sub_wallets sw ON sw.id = wt.wallet_id
            JOIN tokens t ON t.id = wt.token_id
            WHERE sw.master_wallet_id = $1 AND sw.is_active = true
            GROUP BY t.id, t.symbol, t.name
            ORDER BY balance DESC
            LIMIT 20
        "#;
        
        let token_rows = self.db_pool.query(token_query, &[&master_wallet_id]).unwrap_or_default();
        
        let tokens: Vec<serde_json::Value> = token_rows.iter().map(|row| {
            json!({
                "symbol": row.get::<_, String>("symbol"),
                "name": row.get::<_, String>("name"),
                "balance": row.get::<_, f64>("balance")
            })
        }).collect();
        
        Ok(json!({
            "totalBalance": total_balance.to_string(),
            "tokens": tokens,
            "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
        }))
    }
    
    fn initialize(&self) -> Result<(), String> {
        Ok(())
    }
}

/// Policy enforcement fetcher
pub struct PolicyFetcher {
    db_pool: Arc<DatabasePool>,
    cache: Arc<CacheManager>,
}

impl PolicyFetcher {
    pub fn new(db_pool: Arc<DatabasePool>, cache: Arc<CacheManager>) -> Self {
        Self { db_pool, cache }
    }
}

impl MasterFetcher for PolicyFetcher {
    fn name(&self) -> &str { "policy" }
    
    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String> {
        let master_wallet_id = params.get("master_wallet_id").cloned().unwrap_or_default();
        
        // Get policies from configuration
        let query = r#"
            SELECT id, policy_type, rules, is_active, created_at
            FROM policies
            WHERE master_wallet_id = $1 AND is_active = true
        "#;
        
        let rows = self.db_pool.query(query, &[&master_wallet_id]).unwrap_or_default();
        
        let policies: Vec<serde_json::Value> = rows.iter().map(|row| {
            json!({
                "id": row.get::<_, String>("id"),
                "policyType": row.get::<_, String>("policy_type"),
                "rules": row.get::<_, serde_json::Value>("rules"),
                "isActive": row.get::<_, bool>("is_active"),
                "createdAt": row.get::<_, i64>("created_at")
            })
        }).collect();
        
        Ok(json!({
            "policies": policies,
            "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
        }))
    }
    
    fn initialize(&self) -> Result<(), String> {
        let create_table = r#"
            CREATE TABLE IF NOT EXISTS policies (
                id VARCHAR(64) PRIMARY KEY,
                master_wallet_id VARCHAR(64) NOT NULL,
                policy_type VARCHAR(32) NOT NULL,
                rules JSONB NOT NULL DEFAULT '{}',
                is_active BOOLEAN DEFAULT true,
                created_at BIGINT NOT NULL,
                updated_at BIGINT NOT NULL,
                FOREIGN KEY (master_wallet_id) REFERENCES master_wallets(id) ON DELETE CASCADE
            );
        "#;
        
        self.db_pool.execute(create_table, &[])?;
        Ok(())
    }
}
