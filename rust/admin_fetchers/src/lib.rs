//! TigerWallet Admin Fetchers - Rust High-Speed Implementation
//! 
//! This module implements fetchers specifically for Admin apps
//! - Platform management
//! - User management
//! - System configuration
//! - Analytics
//! - Real PostgreSQL and Redis integration
//! - NO user wallet operations (those are in userwallet_fetchers)
//! - NO master wallet operations (those are in masterwallet_fetchers)

use std::collections::HashMap;
use std::sync::Arc;

pub mod types;
pub mod database;
pub mod cache;
pub mod fetchers;

pub use types::*;
pub use database::{DatabasePool, DatabaseConfig};
pub use cache::{CacheManager, RedisConfig};
pub use fetchers::*;

/// Admin fetcher manager - only includes admin operations
pub struct AdminFetcherManager {
    fetchers: HashMap<String, Arc<dyn AdminFetcher>>,
    db_pool: Option<Arc<DatabasePool>>,
    cache: Option<Arc<CacheManager>>,
}

impl AdminFetcherManager {
    /// Create a new AdminFetcherManager with database and cache connections
    pub fn new(db_config: Option<&DatabaseConfig>, redis_config: Option<&RedisConfig>) -> Result<Self, String> {
        let mut fetchers = HashMap::new();
        
        // Initialize database pool if config provided
        let db_pool = if let Some(config) = db_config {
            Some(Arc::new(DatabasePool::new(config)?))
        } else {
            None
        };
        
        // Initialize cache if config provided
        let cache = if let Some(config) = redis_config {
            Some(Arc::new(CacheManager::new(config)?))
        } else {
            None
        };
        
        // Create fetchers with real implementations
        let db = db_pool.clone();
        let ch = cache.clone();
        
        // Admin operations only - using real implementations
        fetchers.insert(
            "users".to_string(), 
            Arc::new(AdminUserFetcher::new(
                db.clone().ok_or("Database pool required")?,
                ch.clone().ok_or("Cache required")?
            )) as Arc<dyn AdminFetcher>
        );
        
        fetchers.insert(
            "kyc".to_string(), 
            Arc::new(KycFetcher::new(
                db.clone().ok_or("Database pool required")?,
                ch.clone().ok_or("Cache required")?
            )) as Arc<dyn AdminFetcher>
        );
        
        fetchers.insert(
            "system".to_string(), 
            Arc::new(SystemFetcher::new(
                db.clone().ok_or("Database pool required")?,
                ch.clone().ok_or("Cache required")?
            )) as Arc<dyn AdminFetcher>
        );
        
        fetchers.insert(
            "fees".to_string(), 
            Arc::new(FeesFetcher::new(
                db.clone().ok_or("Database pool required")?,
                ch.clone().ok_or("Cache required")?
            )) as Arc<dyn AdminFetcher>
        );
        
        fetchers.insert(
            "tokens".to_string(), 
            Arc::new(TokenManagementFetcher::new(
                db.clone().ok_or("Database pool required")?,
                ch.clone().ok_or("Cache required")?
            )) as Arc<dyn AdminFetcher>
        );
        
        fetchers.insert(
            "analytics".to_string(), 
            Arc::new(AdminAnalyticsFetcher::new(
                db.clone().ok_or("Database pool required")?,
                ch.clone().ok_or("Cache required")?
            )) as Arc<dyn AdminFetcher>
        );
        
        fetchers.insert(
            "transactions_admin".to_string(), 
            Arc::new(TransactionAdminFetcher::new(
                db.clone().ok_or("Database pool required")?,
                ch.clone().ok_or("Cache required")?
            )) as Arc<dyn AdminFetcher>
        );
        
        fetchers.insert(
            "config".to_string(), 
            Arc::new(ConfigFetcher::new(
                db.clone().ok_or("Database pool required")?,
                ch.clone().ok_or("Cache required")?
            )) as Arc<dyn AdminFetcher>
        );
        
        // Initialize all fetchers
        for fetcher in fetchers.values() {
            fetcher.initialize()?;
        }
        
        Ok(Self { fetchers, db_pool, cache })
    }
    
    /// Get a fetcher by name
    pub fn get_fetcher(&self, name: &str) -> Option<Arc<dyn AdminFetcher>> {
        self.fetchers.get(name).cloned()
    }
    
    /// Get all fetcher names
    pub fn get_all_fetchers(&self) -> Vec<String> {
        self.fetchers.keys().cloned().collect()
    }
    
    /// Get database pool
    pub fn db_pool(&self) -> Option<&Arc<DatabasePool>> {
        self.db_pool.as_ref()
    }
    
    /// Get cache manager
    pub fn cache(&self) -> Option<&Arc<CacheManager>> {
        self.cache.as_ref()
    }
}
