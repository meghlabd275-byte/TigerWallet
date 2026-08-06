//! Fetcher management for White Level SDK

use crate::types::*;
use crate::errors::{Result, SdkError};
use crate::config::Config;
use async_trait::async_trait;
use parking_lot::RwLock;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};

/// Fetcher cache entry
#[derive(Clone)]
struct CacheEntry {
    data: serde_json::Value,
    timestamp: Instant,
    ttl: Duration,
}

impl CacheEntry {
    fn is_valid(&self) -> bool {
        self.timestamp.elapsed() < self.ttl
    }
}

/// Fetcher manager
pub struct FetcherManager {
    config: Config,
    fetchers: Arc<RwLock<HashMap<FetcherType, FetcherConfig>>>,
    cache: Arc<RwLock<HashMap<String, CacheEntry>>>,
    http_client: reqwest::Client,
}

impl FetcherManager {
    /// Create new fetcher manager
    pub fn new(config: Config) -> Self {
        Self {
            config,
            fetchers: Arc::new(RwLock::new(HashMap::new())),
            cache: Arc::new(RwLock::new(HashMap::new())),
            http_client: reqwest::Client::new(),
        }
    }

    /// Initialize fetchers from config
    pub fn initialize(&self, fetcher_configs: Vec<FetcherConfig>) {
        let mut fetchers = self.fetchers.write();
        for fc in fetcher_configs {
            fetchers.insert(fc.fetcher, fc);
        }
    }

    /// Update fetcher configuration
    pub fn update_fetcher(&self, config: FetcherConfig) {
        let mut fetchers = self.fetchers.write();
        fetchers.insert(config.fetcher, config);
    }

    /// Fetch data from Super Admin
    pub async fn fetch(&self, request: FetcherRequest) -> Result<FetcherResponse> {
        // Check permission
        // (Handled by the main client)

        // Check cache first
        if request.cache && self.config.caching {
            let cache_key = self.get_cache_key(&request);
            if let Some(response) = self.get_from_cache(&cache_key) {
                return Ok(response);
            }
        }

        // Get fetcher config
        let fetcher_config = {
            let fetchers = self.fetchers.read();
            fetchers.get(&request.fetcher).cloned()
        };

        let config = match fetcher_config {
            Some(c) => c,
            None => {
                // Use default config
                FetcherConfig {
                    fetcher: request.fetcher,
                    endpoint: format!("{}/api/v1/fetch", self.config.super_admin_url),
                    timeout_ms: 5000,
                    retry_count: self.config.retry_count,
                    cache_ttl_seconds: 60,
                    is_enabled: true,
                }
            }
        };

        // Make request with retries
        let mut last_error = None;
        for attempt in 0..config.retry_count {
            match self.do_fetch(&config, &request).await {
                Ok(mut response) => {
                    // Cache if enabled
                    if request.cache && self.config.caching {
                        let cache_key = self.get_cache_key(&request);
                        self.put_to_cache(&cache_key, response.data.clone(), Duration::from_secs(config.cache_ttl_seconds as u64));
                        response.cached = false;
                    }
                    return Ok(response);
                }
                Err(e) if attempt < config.retry_count - 1 => {
                    last_error = Some(e);
                }
                Err(e) => {
                    return Err(e);
                }
            }
        }

        Err(last_error.unwrap_or(SdkError::FetcherError("Unknown error".to_string())))
    }

    /// Do the actual fetch
    async fn do_fetch(&self, config: &FetcherConfig, request: &FetcherRequest) -> Result<FetcherResponse> {
        let url = &config.endpoint;
        
        let start = Instant::now();
        
        let response = self.http_client
            .post(url)
            .header("X-API-Key", &self.config.api_key)
            .timeout(Duration::from_millis(config.timeout_ms as u64))
            .json(&request)
            .send()
            .await?;

        let latency_ms = start.elapsed().as_millis() as u64;

        if !response.status().is_success() {
            let error_text = response.text().await.unwrap_or_default();
            return Err(SdkError::FetcherError(error_text));
        }

        let data: serde_json::Value = response.json().await?;

        Ok(FetcherResponse {
            data,
            cached: false,
            latency_ms,
            timestamp: chrono::Utc::now(),
        })
    }

    /// Get from cache
    fn get_from_cache(&self, key: &str) -> Option<FetcherResponse> {
        let cache = self.cache.read();
        if let Some(entry) = cache.get(key) {
            if entry.is_valid() {
                return Some(FetcherResponse {
                    data: entry.data.clone(),
                    cached: true,
                    latency_ms: 0,
                    timestamp: chrono::Utc::now(),
                });
            }
        }
        None
    }

    /// Put to cache
    fn put_to_cache(&self, key: &str, data: serde_json::Value, ttl: Duration) {
        let mut cache = self.cache.write();
        cache.insert(key.to_string(), CacheEntry {
            data,
            timestamp: Instant::now(),
            ttl,
        });
    }

    /// Generate cache key
    fn get_cache_key(&self, request: &FetcherRequest) -> String {
        format!("{:?}:{}", request.fetcher, request.params.to_string())
    }

    /// Clear cache
    pub fn clear_cache(&self) {
        let mut cache = self.cache.write();
        cache.clear();
    }

    /// Clear specific fetcher cache
    pub fn clear_fetcher_cache(&self, fetcher: FetcherType) {
        let mut cache = self.cache.write();
        cache.retain(|key, _| !key.starts_with(&format!("{:?}:", fetcher)));
    }

    /// Get fetcher status
    pub fn get_fetcher_status(&self) -> Vec<(FetcherType, bool)> {
        let fetchers = self.fetchers.read();
        fetchers.iter()
            .map(|(k, v)| (*k, v.is_enabled))
            .collect()
    }

    /// Enable/disable fetcher
    pub fn set_fetcher_enabled(&self, fetcher: FetcherType, enabled: bool) {
        let mut fetchers = self.fetchers.write();
        if let Some(config) = fetchers.get_mut(&fetcher) {
            config.is_enabled = enabled;
        }
    }

    /// Get cache statistics
    pub fn get_cache_stats(&self) -> CacheStats {
        let cache = self.cache.read();
        let total = cache.len();
        let valid = cache.values().filter(|e| e.is_valid()).count();
        
        CacheStats {
            total_entries: total,
            valid_entries: valid,
            expired_entries: total - valid,
        }
    }
}

/// Cache statistics
#[derive(Debug, Clone)]
pub struct CacheStats {
    pub total_entries: usize,
    pub valid_entries: usize,
    pub expired_entries: usize,
}
