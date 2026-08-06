use std::sync::Arc;
use std::time::Duration;

use anyhow::Result;
use async_trait::async_trait;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use tokio::sync::RwLock;
use uuid::Uuid;

/// Represents the type of data a fetcher can retrieve
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum FetcherType {
    Blockchain,
    Market,
    Wallet,
    User,
    Token,
    Price,
    Transaction,
    Balance,
}

/// Status of a fetcher
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum FetcherStatus {
    Idle,
    Running,
    Paused,
    Error,
    Stopped,
}

/// Configuration for a specific fetcher
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetcherConfig {
    pub fetcher_type: FetcherType,
    pub source: String,
    pub interval_ms: u64,
    pub timeout_ms: u64,
    pub retry_count: u32,
    pub retry_delay_ms: u64,
    pub batch_size: usize,
    pub enabled: bool,
}

impl Default for FetcherConfig {
    fn default() -> Self {
        Self {
            fetcher_type: FetcherType::Blockchain,
            source: String::new(),
            interval_ms: 1000,
            timeout_ms: 5000,
            retry_count: 3,
            retry_delay_ms: 100,
            batch_size: 100,
            enabled: true,
        }
    }
}

/// Generic fetcher trait that all fetchers must implement
#[async_trait]
pub trait Fetcher: Send + Sync {
    /// Returns the type of this fetcher
    fn fetcher_type(&self) -> FetcherType;

    /// Returns the name of this fetcher
    fn name(&self) -> &str;

    /// Fetch data from the source
    async fn fetch(&self, params: FetchParams) -> Result<FetchResult>;

    /// Initialize the fetcher
    async fn initialize(&mut self, config: &FetcherConfig) -> Result<()>;

    /// Shutdown the fetcher
    async fn shutdown(&mut self) -> Result<()>;
}

/// Parameters for fetching data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetchParams {
    pub tenant_id: Uuid,
    pub chain: Option<String>,
    pub address: Option<String>,
    pub symbols: Option<Vec<String>>,
    pub from_block: Option<u64>,
    pub to_block: Option<u64>,
    pub limit: Option<usize>,
    pub force_refresh: bool,
}

impl Default for FetchParams {
    fn default() -> Self {
        Self {
            tenant_id: Uuid::nil(),
            chain: None,
            address: None,
            symbols: None,
            from_block: None,
            to_block: None,
            limit: Some(100),
            force_refresh: false,
        }
    }
}

/// Result from a fetch operation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetchResult {
    pub data: serde_json::Value,
    pub timestamp: DateTime<Utc>,
    pub latency_ms: u64,
    pub source: String,
    pub cached: bool,
    pub error: Option<String>,
}

impl FetchResult {
    pub fn success(data: serde_json::Value, source: String, latency_ms: u64, cached: bool) -> Self {
        Self {
            data,
            timestamp: Utc::now(),
            latency_ms,
            source,
            cached,
            error: None,
        }
    }

    pub fn error(source: String, error: String, latency_ms: u64) -> Self {
        Self {
            data: serde_json::Value::Null,
            timestamp: Utc::now(),
            latency_ms,
            source,
            cached: false,
            error: Some(error),
        }
    }
}

/// Metrics for a fetcher
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct FetcherMetrics {
    pub total_requests: u64,
    pub successful_requests: u64,
    pub failed_requests: u64,
    pub total_latency_ms: u64,
    pub cache_hits: u64,
    pub cache_misses: u64,
    pub rate_limited: u64,
}

impl FetcherMetrics {
    pub fn success(&mut self, latency_ms: u64, cached: bool) {
        self.total_requests += 1;
        self.successful_requests += 1;
        self.total_latency_ms += latency_ms;
        if cached {
            self.cache_hits += 1;
        } else {
            self.cache_misses += 1;
        }
    }

    pub fn failure(&mut self) {
        self.total_requests += 1;
        self.failed_requests += 1;
    }

    pub fn rate_limited(&mut self) {
        self.rate_limited += 1;
    }

    pub fn avg_latency(&self) -> f64 {
        if self.total_requests == 0 {
            return 0.0;
        }
        self.total_latency_ms as f64 / self.total_requests as f64
    }

    pub fn success_rate(&self) -> f64 {
        if self.total_requests == 0 {
            return 0.0;
        }
        (self.successful_requests as f64 / self.total_requests as f64) * 100.0
    }

    pub fn cache_hit_rate(&self) -> f64 {
        let total = self.cache_hits + self.cache_misses;
        if total == 0 {
            return 0.0;
        }
        (self.cache_hits as f64 / total as f64) * 100.0
    }
}

/// State of a fetcher instance
pub struct FetcherState {
    pub status: FetcherStatus,
    pub config: FetcherConfig,
    pub metrics: FetcherMetrics,
    pub last_run: Option<DateTime<Utc>>,
    pub last_error: Option<String>,
    pub running_tasks: usize,
}

impl Default for FetcherState {
    fn default() -> Self {
        Self {
            status: FetcherStatus::Idle,
            config: FetcherConfig::default(),
            metrics: FetcherMetrics::default(),
            last_run: None,
            last_error: None,
            running_tasks: 0,
        }
    }
}

impl FetcherState {
    pub fn new(config: FetcherConfig) -> Self {
        Self {
            status: FetcherStatus::Idle,
            config,
            metrics: FetcherMetrics::default(),
            last_run: None,
            last_error: None,
            running_tasks: 0,
        }
    }
}

/// Trait for managing a collection of fetchers
#[async_trait]
pub trait FetcherManager: Send + Sync {
    /// Register a new fetcher
    async fn register_fetcher(&self, fetcher: Arc<dyn Fetcher>) -> Result<()>;

    /// Unregister a fetcher
    async fn unregister_fetcher(&self, fetcher_type: FetcherType) -> Result<()>;

    /// Get a fetcher by type
    async fn get_fetcher(&self, fetcher_type: FetcherType) -> Option<Arc<dyn Fetcher>>;

    /// Execute a fetch operation
    async fn fetch(&self, fetcher_type: FetcherType, params: FetchParams) -> Result<FetchResult>;

    /// Get metrics for a specific fetcher
    async fn get_metrics(&self, fetcher_type: FetcherType) -> Option<FetcherMetrics>;

    /// Get all metrics
    async fn get_all_metrics(&self) -> Vec<(FetcherType, FetcherMetrics)>;

    /// Start all fetchers
    async fn start_all(&self) -> Result<()>;

    /// Stop all fetchers
    async fn stop_all(&self) -> Result<()>;
}
