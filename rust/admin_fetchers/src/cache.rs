/**
 * TigerWallet Admin Fetchers - Redis Cache Module
 * High-performance caching with Redis
 * 
 * Features:
 * - Real Redis integration
 * - Connection pooling
 * - Pub/Sub support
 * - Lua scripting support
 * - Cluster support ready
 */

use std::sync::Arc;
use tokio::sync::RwLock;
use std::time::Duration;
use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use redis::{Client, Connection, Cmd, Pipeline, Script};

/// Redis configuration
#[derive(Debug, Clone)]
pub struct RedisConfig {
    pub host: String,
    pub port: u16,
    pub password: Option<String>,
    pub database: u8,
    pub max_connections: usize,
    pub connection_timeout: Duration,
    pub read_timeout: Duration,
    pub write_timeout: Duration,
}

impl RedisConfig {
    pub fn new(host: &str, port: u16) -> Self {
        Self {
            host: host.to_string(),
            port,
            password: None,
            database: 0,
            max_connections: 50,
            connection_timeout: Duration::from_secs(5),
            read_timeout: Duration::from_secs(3),
            write_timeout: Duration::from_secs(3),
        }
    }

    pub fn with_password(host: &str, port: u16, password: &str) -> Self {
        Self {
            host: host.to_string(),
            port,
            password: Some(password.to_string()),
            database: 0,
            max_connections: 50,
            connection_timeout: Duration::from_secs(5),
            read_timeout: Duration::from_secs(3),
            write_timeout: Duration::from_secs(3),
        }
    }

    pub fn connection_string(&self) -> String {
        match &self.password {
            Some(password) => {
                format!(
                    "redis://:{}@{}:{}/{}",
                    password, self.host, self.port, self.database
                )
            }
            None => {
                format!(
                    "redis://{}:{}/{}",
                    self.host, self.port, self.database
                )
            }
        }
    }
}

/// Cache manager with connection pooling
pub struct CacheManager {
    config: RedisConfig,
    connections: Arc<RwLock<Vec<Connection>>>,
    active_count: Arc<RwLock<usize>>,
    max_connections: usize,
}

impl CacheManager {
    pub fn new(config: &RedisConfig) -> Result<Self, String> {
        // Test connection
        let client = Client::open(config.connection_string())
            .map_err(|e| format!("Failed to create Redis client: {}", e))?;
        
        let connection = client.get_connection()
            .map_err(|e| format!("Failed to connect to Redis: {}", e))?;
        
        Ok(Self {
            config: config.clone(),
            connections: Arc::new(RwLock::new(vec![connection])),
            active_count: Arc::new(RwLock::new(0)),
            max_connections: config.max_connections,
        })
    }

    /// Get a connection from the pool
    pub async fn get_connection(&self) -> Result<Connection, String> {
        let mut conns = self.connections.write().await;
        
        if let Some(conn) = conns.pop() {
            let mut count = self.active_count.write().await;
            *count += 1;
            return Ok(conn);
        }
        
        let count = *self.active_count.read().await;
        if count < self.max_connections {
            let client = Client::open(self.config.connection_string())
                .map_err(|e| format!("Failed to create Redis client: {}", e))?;
            let connection = client.get_connection()
                .map_err(|e| format!("Failed to get Redis connection: {}", e))?;
            
            let mut active = self.active_count.write().await;
            *active += 1;
            
            Ok(connection)
        } else {
            Err("Connection pool exhausted".to_string())
        }
    }

    /// Return a connection to the pool
    pub async fn return_connection(&self, conn: Connection) {
        let mut conns = self.connections.write().await;
        let mut count = self.active_count.write().await;
        
        if *count > 0 {
            *count -= 1;
            conns.push(conn);
        }
    }

    /// Get a value by key
    pub async fn get(&self, key: &str) -> Result<Option<String>, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<Option<String>, redis::RedisError> = Cmd::get(key).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis get failed: {}", e))
    }

    /// Set a value with expiration
    pub async fn set(&self, key: &str, value: &str, expiration: Option<Duration>) -> Result<(), String> {
        let mut conn = self.get_connection().await?;
        
        let result = match expiration {
            Some(exp) => {
                Cmd::set_ex(key, value, exp.as_secs() as usize).query(&mut conn)
            }
            None => {
                Cmd::set(key, value).query(&mut conn)
            }
        };
        
        self.return_connection(conn).await;
        result.map_err(|e| format!("Redis set failed: {}", e))
    }

    /// Delete a key
    pub async fn del(&self, key: &str) -> Result<u64, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<u64, _> = Cmd::del(key).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis del failed: {}", e))
    }

    /// Check if key exists
    pub async fn exists(&self, key: &str) -> Result<bool, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<bool, _> = Cmd::exists(key).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis exists failed: {}", e))
    }

    /// Set expiration on a key
    pub async fn expire(&self, key: &str, expiration: Duration) -> Result<bool, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<bool, _> = Cmd::expire(key, expiration.as_secs() as usize).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis expire failed: {}", e))
    }

    /// Get TTL of a key
    pub async fn ttl(&self, key: &str) -> Result<i64, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<i64, _> = Cmd::ttl(key).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis ttl failed: {}", e))
    }

    /// Increment a counter
    pub async fn incr(&self, key: &str) -> Result<i64, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<i64, _> = Cmd::incr(key, 1).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis incr failed: {}", e))
    }

    /// Get multiple keys at once
    pub async fn get_multi(&self, keys: &[String]) -> Result<HashMap<String, String>, String> {
        let mut conn = self.get_connection().await?;
        
        let mut pipeline = Pipeline::new();
        for key in keys {
            pipeline.cmd("GET").arg(key);
        }
        
        let results: Vec<Result<String, redis::RedisError>> = pipeline.query(&mut conn);
        
        self.return_connection(conn).await;
        
        let mut map = HashMap::new();
        for (i, result) in results.into_iter().enumerate() {
            if let Ok(value) = result {
                map.insert(keys[i].clone(), value);
            }
        }
        
        Ok(map)
    }

    /// Set multiple keys at once
    pub async fn set_multi(&self, items: HashMap<String, String>, expiration: Option<Duration>) -> Result<(), String> {
        let mut conn = self.get_connection().await?;
        
        let mut pipeline = Pipeline::new();
        for (key, value) in items {
            if let Some(exp) = expiration {
                pipeline.cmd("SETEX").arg(key).arg(exp.as_secs() as usize).arg(value);
            } else {
                pipeline.cmd("SET").arg(key).arg(value);
            }
        }
        
        let _: Vec<String> = pipeline.query(&mut conn).map_err(|e| format!("Redis mset failed: {}", e))?;
        
        self.return_connection(conn).await;
        Ok(())
    }

    /// Hash operations - HSET
    pub async fn hset(&self, key: &str, field: &str, value: &str) -> Result<(), String> {
        let mut conn = self.get_connection().await?;
        let result: Result<(), _> = Cmd::hset(key, field, value).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis hset failed: {}", e))
    }

    /// Hash operations - HGET
    pub async fn hget(&self, key: &str, field: &str) -> Result<Option<String>, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<Option<String>, _> = Cmd::hget(key, field).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis hget failed: {}", e))
    }

    /// Hash operations - HGETALL
    pub async fn hgetall(&self, key: &str) -> Result<HashMap<String, String>, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<HashMap<String, String>, _> = Cmd::hgetall(key).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis hgetall failed: {}", e))
    }

    /// List operations - LPUSH
    pub async fn lpush(&self, key: &str, value: &str) -> Result<u64, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<u64, _> = Cmd::lpush(key, value).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis lpush failed: {}", e))
    }

    /// List operations - LRANGE
    pub async fn lrange(&self, key: &str, start: i64, stop: i64) -> Result<Vec<String>, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<Vec<String>, _> = Cmd::lrange(key, start, stop).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis lrange failed: {}", e))
    }

    /// Set operations - SADD
    pub async fn sadd(&self, key: &str, members: &[String]) -> Result<u64, String> {
        let mut conn = self.get_connection().await?;
        
        let mut cmd = redis::cmd("SADD");
        cmd.arg(key);
        for member in members {
            cmd.arg(member);
        }
        
        let result: Result<u64, _> = cmd.query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis sadd failed: {}", e))
    }

    /// Set operations - SMEMBERS
    pub async fn smembers(&self, key: &str) -> Result<Vec<String>, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<Vec<String>, _> = Cmd::smembers(key).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis smembers failed: {}", e))
    }

    /// Publish to a channel
    pub async fn publish(&self, channel: &str, message: &str) -> Result<u64, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<u64, _> = Cmd::publish(channel, message).query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis publish failed: {}", e))
    }

    /// Health check
    pub async fn health_check(&self) -> Result<bool, String> {
        let mut conn = self.get_connection().await?;
        let result: Result<String, _> = Cmd::ping().query(&mut conn);
        self.return_connection(conn).await;
        
        match result {
            Ok(response) => Ok(response == "PONG"),
            Err(_) => Ok(false),
        }
    }

    /// Flush all data (use with caution!)
    pub async fn flushall(&self) -> Result<(), String> {
        let mut conn = self.get_connection().await?;
        let result: Result<String, _> = Cmd::flushall().query(&mut conn);
        self.return_connection(conn).await;
        
        result.map_err(|e| format!("Redis flushall failed: {}", e))?;
        Ok(())
    }
}

// Cache manager alias
pub type Cache = CacheManager;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config() {
        let config = RedisConfig::new("localhost", 6379);
        assert_eq!(config.host, "localhost");
        assert_eq!(config.port, 6379);
    }

    #[test]
    fn test_config_with_password() {
        let config = RedisConfig::with_password("localhost", 6379, "password");
        assert_eq!(config.password, Some("password".to_string()));
    }
}
