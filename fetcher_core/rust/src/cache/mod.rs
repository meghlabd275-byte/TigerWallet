use std::sync::Arc;
use std::time::Duration;

use anyhow::Result;
use async_trait::async_trait;
use redis::{aio::ConnectionManager, AsyncCommands, Client};
use serde::{de::DeserializeOwned, Serialize};
use tokio::sync::RwLock;
use tracing::{debug, error, info};

/// Cache implementation using Redis
pub struct FetcherCache {
    redis: Arc<RwLock<Option<ConnectionManager>>>,
    memory_cache: Arc<RwLock<std::collections::HashMap<String, (serde_json::Value, std::time::Instant)>>>,
    enable_memory_fallback: bool,
}

impl FetcherCache {
    pub fn new(enable_memory_fallback: bool) -> Self {
        Self {
            redis: Arc::new(RwLock::new(None)),
            memory_cache: Arc::new(RwLock::new(std::collections::HashMap::new())),
            enable_memory_fallback,
        }
    }

    /// Connect to Redis
    pub async fn connect(&self, url: &str) -> Result<()> {
        let client = Client::open(url)?;
        let manager = ConnectionManager::new(client).await?;
        
        let mut redis = self.redis.write().await;
        *redis = Some(manager);
        
        info!("Connected to Redis cache");
        Ok(())
    }

    /// Get a value from cache
    pub async fn get<T: DeserializeOwned>(&self, key: &str) -> Result<T> {
        // Try Redis first
        {
            let redis = self.redis.read().await;
            if let Some(ref conn) = *redis {
                if let Ok(value) = conn.get(key).await {
                    if let Ok(data) = value {
                        debug!("Cache hit (Redis): {}", key);
                        return Ok(serde_json::from_slice(&data)?);
                    }
                }
            }
        }

        // Try memory cache
        if self.enable_memory_fallback {
            let memory = self.memory_cache.read().await;
            if let Some((value, _)) = memory.get(key) {
                debug!("Cache hit (Memory): {}", key);
                return Ok(serde_json::from_value(value.clone())?);
            }
        }

        debug!("Cache miss: {}", key);
        Err(anyhow::anyhow!("Cache miss"))
    }

    /// Set a value in cache
    pub async fn set<T: Serialize>(&self, key: &str, value: &T, ttl_seconds: u64) -> Result<()> {
        let data = serde_json::to_vec(value)?;
        
        // Try Redis first
        {
            let redis = self.redis.read().await;
            if let Some(ref conn) = *redis {
                let _: () = conn.set_ex(key, data, ttl_seconds as usize).await?;
                debug!("Cached (Redis): {} ({}s)", key, ttl_seconds);
            }
        }

        // Also store in memory for fallback
        if self.enable_memory_fallback {
            let mut memory = self.memory_cache.write().await;
            memory.insert(
                key.to_string(),
                (serde_json::to_value(value)?, std::time::Instant::now()),
            );
            
            // Clean old entries
            if memory.len() > 10000 {
                let now = std::time::Instant::now();
                memory.retain(|_, (_, created)| {
                    now.duration_since(*created) < Duration::from_secs(300)
                });
            }
        }

        Ok(())
    }

    /// Delete a key from cache
    pub async fn delete(&self, key: &str) -> Result<()> {
        // Delete from Redis
        {
            let redis = self.redis.read().await;
            if let Some(ref conn) = *redis {
                let _: () = conn.del(key).await?;
            }
        }

        // Delete from memory
        {
            let mut memory = self.memory_cache.write().await;
            memory.remove(key);
        }

        Ok(())
    }

    /// Check if a key exists
    pub async fn exists(&self, key: &str) -> bool {
        // Check Redis
        {
            let redis = self.redis.read().await;
            if let Some(ref conn) = *redis {
                if let Ok(exists) = conn.exists(key).await {
                    if exists {
                        return true;
                    }
                }
            }
        }

        // Check memory
        if self.enable_memory_fallback {
            let memory = self.memory_cache.read().await;
            if memory.contains_key(key) {
                return true;
            }
        }

        false
    }

    /// Get multiple keys at once
    pub async fn get_many<T: DeserializeOwned>(&self, keys: &[String]) -> Result<Vec<Option<T>>> {
        let mut results = Vec::with_capacity(keys.len());
        
        for key in keys {
            match self.get(key).await {
                Ok(value) => results.push(Some(value)),
                Err(_) => results.push(None),
            }
        }
        
        Ok(results)
    }

    /// Invalidate cache by pattern
    pub async fn invalidate_pattern(&self, pattern: &str) -> Result<u64> {
        let mut deleted = 0u64;

        // Delete from memory
        {
            let mut memory = self.memory_cache.write().await;
            let keys: Vec<String> = memory.keys()
                .filter(|k| k.contains(pattern))
                .cloned()
                .collect();
            
            for key in keys {
                memory.remove(&key);
                deleted += 1;
            }
        }

        // Note: Redis pattern deletion would require SCAN + DEL in production
        
        info!("Invalidated {} cache entries matching pattern: {}", deleted, pattern);
        Ok(deleted)
    }

    /// Get cache statistics
    pub async fn stats(&self) -> CacheStats {
        let redis_size = {
            let redis = self.redis.read().await;
            if redis.is_some() {
                "connected".to_string()
            } else {
                "not connected".to_string()
            }
        };

        let memory_size = {
            let memory = self.memory_cache.read().await;
            memory.len()
        };

        CacheStats {
            redis_status: redis_size,
            memory_entries: memory_size,
            max_memory_entries: 10000,
        }
    }
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct CacheStats {
    pub redis_status: String,
    pub memory_entries: usize,
    pub max_memory_entries: usize,
}

/// In-memory cache for development/testing
#[derive(Debug, Clone)]
pub struct MemoryCache {
    data: Arc<RwLock<std::collections::HashMap<String, (Vec<u8>, std::time::Instant)>>>,
    ttl: Duration,
}

impl MemoryCache {
    pub fn new(ttl: Duration) -> Self {
        Self {
            data: Arc::new(RwLock::new(std::collections::HashMap::new())),
            ttl,
        }
    }

    pub async fn get(&self, key: &str) -> Option<Vec<u8>> {
        let data = self.data.read().await;
        if let Some((value, created)) = data.get(key) {
            if created.elapsed() < self.ttl {
                return Some(value.clone());
            }
        }
        None
    }

    pub async fn set(&self, key: &str, value: Vec<u8>) {
        let mut data = self.data.write().await;
        data.insert(key.to_string(), (value, std::time::Instant::now()));
    }

    pub async fn delete(&self, key: &str) {
        let mut data = self.data.write().await;
        data.remove(key);
    }

    pub async fn clear(&self) {
        let mut data = self.data.write().await;
        data.clear();
    }

    pub async fn size(&self) -> usize {
        let data = self.data.read().await;
        data.len()
    }
}
