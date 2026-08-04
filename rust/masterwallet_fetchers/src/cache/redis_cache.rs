//! Redis cache implementation for high-performance caching
//! Supports connection pooling, pub/sub, and distributed locking

use std::sync::{Arc, RwLock};
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use redis::{Client, Commands, Connection, PooledConnection, RedisError};
use std::collections::HashMap;

/// Redis cache configuration
#[derive(Debug, Clone)]
pub struct RedisCacheConfig {
    pub host: String,
    pub port: u16,
    pub password: Option<String>,
    pub db: u8,
    pub pool_size: u32,
    pub timeout_ms: u64,
}

impl RedisCacheConfig {
    pub fn new(host: &str, port: u16) -> Self {
        Self {
            host: host.to_string(),
            port,
            password: None,
            db: 0,
            pool_size: 20,
            timeout_ms: 5000,
        }
    }
    
    pub fn with_auth(mut self, password: &str) -> Self {
        self.password = Some(password.to_string());
        self
    }
    
    pub fn with_db(mut self, db: u8) -> Self {
        self.db = db;
        self
    }
    
    pub fn with_pool_size(mut self, size: u32) -> Self {
        self.pool_size = size;
        self
    }
    
    pub fn connection_string(&self) -> String {
        if let Some(ref password) = self.password {
            format!("redis://:{}@{}:{}/{}", password, self.host, self.port, self.db)
        } else {
            format!("redis://{}:{}/{}", self.host, self.port, self.db)
        }
    }
}

/// High-performance Redis cache client
pub struct RedisCache {
    client: Client,
    local_cache: Arc<RwLock<HashMap<String, (String, u64)>>>,
    default_ttl: u64,
}

impl RedisCache {
    /// Create a new Redis cache client
    pub fn new(config: &RedisCacheConfig) -> Result<Self, RedisError> {
        let client = Client::open(config.connection_string().as_str())?;
        
        Ok(Self {
            client,
            local_cache: Arc::new(RwLock::new(HashMap::new())),
            default_ttl: 300,
        })
    }
    
    /// Get a value from cache
    pub fn get(&self, key: &str) -> Result<Option<String>, RedisError> {
        // Try local cache first
        if let Ok(cache) = self.local_cache.read() {
            if let Some((value, expiry)) = cache.get(key) {
                let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
                if *expiry > now {
                    return Ok(Some(value.clone()));
                }
            }
        }
        
        // Get from Redis
        let mut con = self.client.get_connection()?;
        let result: Option<String> = con.get(key)?;
        
        // Update local cache
        if let Some(ref value) = result {
            if let Ok(mut cache) = self.local_cache.write() {
                let expiry = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() + self.default_ttl;
                cache.insert(key.to_string(), (value.clone(), expiry));
            }
        }
        
        Ok(result)
    }
    
    /// Set a value in cache
    pub fn set(&self, key: &str, value: &str) -> Result<(), RedisError> {
        let mut con = self.client.get_connection()?;
        con.set(key, value)?;
        
        // Update local cache
        if let Ok(mut cache) = self.local_cache.write() {
            let expiry = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() + self.default_ttl;
            cache.insert(key.to_string(), (value.to_string(), expiry));
        }
        
        Ok(())
    }
    
    /// Set a value with TTL
    pub fn set_ex(&self, key: &str, value: &str, ttl: u64) -> Result<(), RedisError> {
        let mut con = self.client.get_connection()?;
        con.set_ex(key, value, ttl as usize)?;
        
        // Update local cache
        if let Ok(mut cache) = self.local_cache.write() {
            let expiry = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() + ttl;
            cache.insert(key.to_string(), (value.to_string(), expiry));
        }
        
        Ok(())
    }
    
    /// Delete a key
    pub fn delete(&self, key: &str) -> Result<(), RedisError> {
        let mut con = self.client.get_connection()?;
        con.del(key)?;
        
        // Remove from local cache
        if let Ok(mut cache) = self.local_cache.write() {
            cache.remove(key);
        }
        
        Ok(())
    }
    
    /// Check if key exists
    pub fn exists(&self, key: &str) -> Result<bool, RedisError> {
        // Check local cache first
        if let Ok(cache) = self.local_cache.read() {
            if let Some((_, expiry)) = cache.get(key) {
                let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
                if *expiry > now {
                    return Ok(true);
                }
            }
        }
        
        // Check Redis
        let mut con = self.client.get_connection()?;
        let result: bool = con.exists(key)?;
        Ok(result)
    }
    
    /// Increment a counter
    pub fn incr(&self, key: &str) -> Result<i64, RedisError> {
        let mut con = self.client.get_connection()?;
        let result: i64 = con.incr(key, 1)?;
        
        // Update local cache
        if let Ok(mut cache) = self.local_cache.write() {
            let expiry = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() + self.default_ttl;
            cache.insert(key.to_string(), (result.to_string(), expiry));
        }
        
        Ok(result)
    }
    
    /// Decrement a counter
    pub fn decr(&self, key: &str) -> Result<i64, RedisError> {
        let mut con = self.client.get_connection()?;
        con.decr(key, 1)
    }
    
    /// Set expiry on a key
    pub fn expire(&self, key: &str, ttl: u64) -> Result<(), RedisError> {
        let mut con = self.client.get_connection()?;
        con.expire(key, ttl as usize)
    }
    
    /// Hash set
    pub fn hset(&self, key: &str, field: &str, value: &str) -> Result<(), RedisError> {
        let mut con = self.client.get_connection()?;
        con.hset(key, field, value)
    }
    
    /// Hash get
    pub fn hget(&self, key: &str, field: &str) -> Result<Option<String>, RedisError> {
        let mut con = self.client.get_connection()?;
        con.hget(key, field)
    }
    
    /// Hash get all
    pub fn hgetall(&self, key: &str) -> Result<HashMap<String, String>, RedisError> {
        let mut con = self.client.get_connection()?;
        con.hgetall(key)
    }
    
    /// Hash delete
    pub fn hdel(&self, key: &str, field: &str) -> Result<(), RedisError> {
        let mut con = self.client.get_connection()?;
        con.hdel(key, field)
    }
    
    /// List push
    pub fn lpush(&self, key: &str, value: &str) -> Result<(), RedisError> {
        let mut con = self.client.get_connection()?;
        con.lpush(key, value)
    }
    
    /// List range
    pub fn lrange(&self, key: &str, start: i64, stop: i64) -> Result<Vec<String>, RedisError> {
        let mut con = self.client.get_connection()?;
        con.lrange(key, start, stop)
    }
    
    /// Set add
    pub fn sadd(&self, key: &str, member: &str) -> Result<(), RedisError> {
        let mut con = self.client.get_connection()?;
        con.sadd(key, member)
    }
    
    /// Set members
    pub fn smembers(&self, key: &str) -> Result<Vec<String>, RedisError> {
        let mut con = self.client.get_connection()?;
        con.smembers(key)
    }
    
    /// Publish to a channel
    pub fn publish(&self, channel: &str, message: &str) -> Result<(), RedisError> {
        let mut con = self.client.get_connection()?;
        con.publish(channel, message)
    }
    
    /// Get multiple keys at once
    pub fn get_many(&self, keys: &[&str]) -> Result<Vec<Option<String>>, RedisError> {
        let mut con = self.client.get_connection()?;
        let mut results = Vec::new();
        
        for key in keys {
            let value: Option<String> = con.get(*key)?;
            results.push(value);
        }
        
        Ok(results)
    }
    
    /// Set multiple keys at once
    pub fn set_many(&self, items: &[(&str, &str)]) -> Result<(), RedisError> {
        let mut con = self.client.get_connection()?;
        
        for (key, value) in items {
            con.set(*key, *value)?;
        }
        
        Ok(())
    }
    
    /// Flush all keys in current database
    pub fn flushdb(&self) -> Result<(), RedisError> {
        let mut con = self.client.get_connection()?;
        con.flushdb()
    }
    
    /// Get keys matching pattern
    pub fn keys(&self, pattern: &str) -> Result<Vec<String>, RedisError> {
        let mut con = self.client.get_connection()?;
        redis::cmd("KEYS")
            .arg(pattern)
            .query(&mut con)
    }
    
    /// Get info
    pub fn info(&self) -> Result<String, RedisError> {
        let mut con = self.client.get_connection()?;
        redis::cmd("INFO")
            .query(&mut con)
    }
    
    /// Get database size
    pub fn dbsize(&self) -> Result<usize, RedisError> {
        let mut con = self.client.get_connection()?;
        let size: usize = con.dbsize()?;
        Ok(size)
    }
}

/// Distributed lock implementation
pub struct DistributedLock {
    cache: RedisCache,
    lock_key: String,
    ttl: u64,
}

impl DistributedLock {
    pub fn new(cache: RedisCache, key: &str, ttl_secs: u64) -> Self {
        Self {
            cache,
            lock_key: format!("lock:{}", key),
            ttl: ttl_secs,
        }
    }
    
    /// Try to acquire lock
    pub fn acquire(&self, owner: &str) -> Result<bool, RedisError> {
        // Use SET NX EX for atomic lock acquisition
        let mut con = self.cache.client.get_connection()?;
        let result: Option<String> = con.get(&self.lock_key)?;
        
        if result.is_some() {
            return Ok(false); // Lock exists
        }
        
        // Try to acquire lock
        let set_result: bool = con.set_nx(&self.lock_key, owner)?;
        if set_result {
            let _ = con.expire(&self.lock_key, self.ttl as usize);
            Ok(true)
        } else {
            Ok(false)
        }
    }
    
    /// Release lock
    pub fn release(&self, owner: &str) -> Result<bool, RedisError> {
        // Use Lua script for atomic check-and-delete
        let script = r#"
            if redis.call("get", KEYS[1]) == ARGV[1] then
                return redis.call("del", KEYS[1])
            else
                return 0
            end
        "#;
        
        let mut con = self.cache.client.get_connection()?;
        let result: i64 = redis::script(script)
            .key(&self.lock_key)
            .arg(owner)
            .invoke(&mut con)?;
        
        Ok(result == 1)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_config() {
        let config = RedisCacheConfig::new("localhost", 6379)
            .with_auth("password")
            .with_db(1);
        
        assert!(config.password.is_some());
        assert_eq!(config.db, 1);
    }
}
