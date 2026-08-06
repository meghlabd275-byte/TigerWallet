// Fetcher Gateway - Rust Implementation
// Ultra-low latency, high-performance fetcher management for TigerWallet

use async_trait::async_trait;
use axum::{
    extract::{Path, State},
    http::StatusCode,
    routing::{get, post},
    Json, Router,
};
use chrono::{DateTime, Utc};
use redis::AsyncCommands;
use serde::{Deserialize, Serialize};
use sqlx::{postgres::PgPoolOptions, Pool, Postgres};
use std::sync::Arc;
use std::time::Duration;
use thiserror::Error;
use tokio::sync::RwLock;
use tower::ServiceBuilder;
use tower_http::cors::{Any, CorsLayer};
use tracing::{error, info, warn};
use uuid::Uuid;

// ============ ERROR TYPES ============

#[derive(Error, Debug)]
pub enum FetcherError {
    #[error("Database error: {0}")]
    Database(#[from] sqlx::Error),
    #[error("Redis error: {0}")]
    Redis(#[from] redis::RedisError),
    #[error("HTTP error: {0}")]
    Http(#[from] reqwest::Error),
    #[error("Permission denied: {0}")]
    PermissionDenied(String),
    #[error("Fetcher not found: {0}")]
    NotFound(String),
    #[error("Rate limit exceeded")]
    RateLimitExceeded,
    #[error("Timeout")]
    Timeout,
}

impl Serialize for FetcherError {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(&self.to_string())
    }
}

// ============ DATA MODELS ============

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum WhiteLevelProduct {
    MasterWallet,
    UserWallet,
    Bots,
    BotsClients,
    ProjectParty,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum FetcherType {
    Prices,
    Balances,
    Transactions,
    UserData,
    MarketData,
    Blockchain,
    TokenInfo,
    KYC,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetcherConfig {
    pub id: Uuid,
    pub product: WhiteLevelProduct,
    pub fetcher: FetcherType,
    pub endpoint: String,
    pub timeout_ms: u32,
    pub retry_count: u32,
    pub cache_ttl_seconds: u32,
    pub is_active: bool,
    pub config_data: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetcherRequest {
    pub product: WhiteLevelProduct,
    pub fetcher: FetcherType,
    pub params: serde_json::Value,
    pub connection_key: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetcherResponse {
    pub data: serde_json::Value,
    pub cached: bool,
    pub latency_ms: u64,
    pub timestamp: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FetcherMetrics {
    pub fetcher_id: Uuid,
    pub requests_total: i64,
    pub requests_success: i64,
    pub requests_failed: i64,
    pub avg_latency_ms: f64,
    pub cache_hit_rate: f64,
    pub last_updated: DateTime<Utc>,
}

// ============ STATE ============

pub struct AppState {
    pub db: Pool<Postgres>,
    pub redis: redis::Client,
    pub rate_limiter: Arc<RwLock<RateLimiter>>,
    pub fetcher_cache: Arc<RwLock<FetcherCache>>,
}

pub struct RateLimiter {
    limits: std::collections::HashMap<String, (i64, std::time::Instant)>,
    max_requests: i64,
    window_seconds: i64,
}

impl RateLimiter {
    pub fn new(max_requests: i64, window_seconds: i64) -> Self {
        Self {
            limits: std::collections::HashMap::new(),
            max_requests,
            window_seconds,
        }
    }

    pub fn allow(&mut self, key: &str) -> bool {
        let now = std::time::Instant::now();
        
        if let Some((count, last_reset)) = self.limits.get(key) {
            if now.duration_since(*last_reset).as_secs() as i64 > self.window_seconds {
                self.limits.insert(key.to_string(), (1, now));
                return true;
            }
            if *count >= self.max_requests {
                return false;
            }
            self.limits.insert(key.to_string(), (count + 1, *last_reset));
            return true;
        }
        
        self.limits.insert(key.to_string(), (1, now));
        true
    }
}

pub struct FetcherCache {
    cache: std::collections::HashMap<String, (serde_json::Value, std::time::Instant)>,
    ttl_seconds: u64,
}

impl FetcherCache {
    pub fn new(ttl_seconds: u64) -> Self {
        Self {
            cache: std::collections::HashMap::new(),
            ttl_seconds,
        }
    }

    pub fn get(&self, key: &str) -> Option<serde_json::Value> {
        if let Some((value, timestamp)) = self.cache.get(key) {
            if std::time::Instant::now().duration_since(*timestamp).as_secs() < self.ttl_seconds {
                return Some(value.clone());
            }
        }
        None
    }

    pub fn set(&mut self, key: String, value: serde_json::Value) {
        self.cache.insert(key, (value, std::time::Instant::now()));
    }
}

// ============ FETCHER IMPLEMENTATIONS ============

#[async_trait]
pub trait Fetcher: Send + Sync {
    async fn fetch(&self, params: serde_json::Value) -> Result<serde_json::Value, FetcherError>;
    fn cache_key(&self, params: &serde_json::Value) -> String;
}

pub struct PricesFetcher {
    endpoint: String,
    client: reqwest::Client,
}

impl PricesFetcher {
    pub fn new(endpoint: String) -> Self {
        Self {
            endpoint,
            client: reqwest::Client::new(),
        }
    }
}

#[async_trait]
impl Fetcher for PricesFetcher {
    async fn fetch(&self, params: serde_json::Value) -> Result<serde_json::Value, FetcherError> {
        let symbols = params.get("symbols")
            .and_then(|v| v.as_array())
            .map(|arr| arr.iter().filter_map(|s| s.as_str()).collect::<Vec<_>>().join(","))
            .unwrap_or_default();

        let url = format!("{}/prices?symbols={}", self.endpoint, symbols);
        
        let response = self.client
            .get(&url)
            .timeout(Duration::from_millis(5000))
            .send()
            .await?;

        let data = response.json::<serde_json::Value>().await?;
        Ok(data)
    }

    fn cache_key(&self, params: &serde_json::Value) -> String {
        format!("prices:{:?}", params)
    }
}

pub struct BalancesFetcher {
    endpoint: String,
    client: reqwest::Client,
}

impl BalancesFetcher {
    pub fn new(endpoint: String) -> Self {
        Self {
            endpoint,
            client: reqwest::Client::new(),
        }
    }
}

#[async_trait]
impl Fetcher for BalancesFetcher {
    async fn fetch(&self, params: serde_json::Value) -> Result<serde_json::Value, FetcherError> {
        let address = params.get("address")
            .and_then(|v| v.as_str())
            .ok_or_else(|| FetcherError::NotFound("address required".to_string()))?;

        let chain = params.get("chain")
            .and_then(|v| v.as_str())
            .unwrap_or("ethereum");

        let url = format!("{}/balance/{}/{}", self.endpoint, chain, address);
        
        let response = self.client
            .get(&url)
            .timeout(Duration::from_millis(5000))
            .send()
            .await?;

        let data = response.json::<serde_json::Value>().await?;
        Ok(data)
    }

    fn cache_key(&self, params: &serde_json::Value) -> String {
        format!("balances:{:?}", params)
    }
}

pub struct MarketDataFetcher {
    endpoint: String,
    client: reqwest::Client,
}

impl MarketDataFetcher {
    pub fn new(endpoint: String) -> Self {
        Self {
            endpoint,
            client: reqwest::Client::new(),
        }
    }
}

#[async_trait]
impl Fetcher for MarketDataFetcher {
    async fn fetch(&self, params: serde_json::Value) -> Result<serde_json::Value, FetcherError> {
        let market = params.get("market")
            .and_then(|v| v.as_str())
            .unwrap_or("BTC-USDT");

        let url = format!("{}/market/{}", self.endpoint, market);
        
        let response = self.client
            .get(&url)
            .timeout(Duration::from_millis(3000))
            .send()
            .await?;

        let data = response.json::<serde_json::Value>().await?;
        Ok(data)
    }

    fn cache_key(&self, params: &serde_json::Value) -> String {
        format!("market:{:?}", params)
    }
}

// ============ HANDLERS ============

async fn health_check() -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "status": "ok",
        "service": "fetcher-gateway",
        "timestamp": Utc::now()
    }))
}

async fn fetch_data(
    State(state): State<Arc<AppState>>,
    Json(req): Json<FetcherRequest>,
) -> Result<Json<FetcherResponse>, (StatusCode, String)> {
    // Check rate limit
    {
        let mut limiter = state.rate_limiter.write().await;
        if !limiter.allow(&req.connection_key) {
            return Err((StatusCode::TOO_MANY_REQUESTS, "rate limit exceeded".to_string()));
        }
    }

    // Check cache
    let cache_key = format!("{:?}:{:?}:{:?}", req.product, req.fetcher, req.params);
    {
        let cache = state.fetcher_cache.read().await;
        if let Some(data) = cache.get(&cache_key) {
            return Ok(Json(FetcherResponse {
                data,
                cached: true,
                latency_ms: 0,
                timestamp: Utc::now(),
            }));
        }
    }

    // Get fetcher config from database
    let config = sqlx::query_as::<_, (Uuid, String, String, String, u32, u32, u32, bool)>(
        "SELECT id, product, fetcher, endpoint, timeout_ms, retry_count, cache_ttl_seconds, is_active 
         FROM fetcher_configs 
         WHERE product = $1 AND fetcher = $2 AND is_active = true"
    )
    .bind(format!("{:?}", req.product))
    .bind(format!("{:?}", req.fetcher))
    .fetch_optional(&state.db)
    .await
    .map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

    let config = match config {
        Some(c) => c,
        None => return Err((StatusCode::NOT_FOUND, "fetcher not found".to_string())),
    };

    // Execute fetcher based on type
    let start = std::time::Instant::now();
    let result = match req.fetcher {
        FetcherType::Prices => {
            let fetcher = PricesFetcher::new(config.3.clone());
            fetcher.fetch(req.params).await
        }
        FetcherType::Balances => {
            let fetcher = BalancesFetcher::new(config.3.clone());
            fetcher.fetch(req.params).await
        }
        FetcherType::MarketData => {
            let fetcher = MarketDataFetcher::new(config.3.clone());
            fetcher.fetch(req.params).await
        }
        _ => Err(FetcherError::NotFound("fetcher type not implemented".to_string())),
    };

    let latency = start.elapsed().as_millis() as u64;

    match result {
        Ok(data) => {
            // Cache result
            {
                let mut cache = state.fetcher_cache.write().await;
                cache.set(cache_key, data.clone());
            }

            // Update metrics in Redis
            let mut conn = state.redis.get_async().await.map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;
            let _: () = conn.incr("fetcher:requests:success", 1).await.map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

            Ok(Json(FetcherResponse {
                data,
                cached: false,
                latency_ms: latency,
                timestamp: Utc::now(),
            }))
        }
        Err(e) => {
            // Update error metrics
            let mut conn = state.redis.get_async().await.map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;
            let _: () = conn.incr("fetcher:requests:failed", 1).await.map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

            Err((StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))
        }
    }
}

async fn get_fetcher_config(
    State(state): State<Arc<AppState>>,
    Path((product, fetcher)): Path<(String, String)>,
) -> Result<Json<Vec<FetcherConfig>>, (StatusCode, String)> {
    let configs = sqlx::query_as::<_, (Uuid, String, String, String, u32, u32, u32, bool, Option<String>)>(
        "SELECT id, product, fetcher, endpoint, timeout_ms, retry_count, cache_ttl_seconds, is_active, config_data 
         FROM fetcher_configs 
         WHERE product = $1 AND fetcher = $2"
    )
    .bind(&product)
    .bind(&fetcher)
    .fetch_all(&state.db)
    .await
    .map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

    let result: Vec<FetcherConfig> = configs.into_iter().map(|c| FetcherConfig {
        id: c.0,
        product: serde_json::from_str(&c.1).unwrap_or(WhiteLevelProduct::MasterWallet),
        fetcher: serde_json::from_str(&c.2).unwrap_or(FetcherType::Prices),
        endpoint: c.3,
        timeout_ms: c.4,
        retry_count: c.5,
        cache_ttl_seconds: c.6,
        is_active: c.7,
        config_data: c.8,
    }).collect();

    Ok(Json(result))
}

async fn get_metrics(State(state): State<Arc<AppState>>) -> Result<Json<serde_json::Value>, (StatusCode, String)> {
    let mut conn = state.redis.get_async().await.map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

    let requests_success: i64 = conn.get("fetcher:requests:success").await.unwrap_or(0);
    let requests_failed: i64 = conn.get("fetcher:requests:failed").await.unwrap_or(0);
    let requests_total = requests_success + requests_failed;

    let cache_hits: i64 = conn.get("fetcher:cache:hits").await.unwrap_or(0);
    let cache_misses: i64 = conn.get("fetcher:cache:misses").await.unwrap_or(0);
    let cache_total = cache_hits + cache_misses;
    let cache_hit_rate = if cache_total > 0 { (cache_hits as f64 / cache_total as f64) * 100.0 } else { 0.0 };

    Ok(Json(serde_json::json!({
        "requests_total": requests_total,
        "requests_success": requests_success,
        "requests_failed": requests_failed,
        "cache_hit_rate": cache_hit_rate,
        "active_connections": 0,
        "timestamp": Utc::now()
    })))
}

async fn check_permission(
    State(state): State<Arc<AppState>>,
    Json(req): Json<serde_json::Value>,
) -> Result<Json<serde_json::Value>, (StatusCode, String)> {
    let client_id = req.get("client_id")
        .and_then(|v| v.as_str())
        .ok_or_else(|| (StatusCode::BAD_REQUEST, "client_id required".to_string()))?;
    
    let product = req.get("product")
        .and_then(|v| v.as_str())
        .ok_or_else(|| (StatusCode::BAD_REQUEST, "product required".to_string()))?;
    
    let fetcher = req.get("fetcher")
        .and_then(|v| v.as_str())
        .ok_or_else(|| (StatusCode::BAD_REQUEST, "fetcher required".to_string()))?;

    // Check Redis cache first
    let perm_key = format!("perm:{}:{}:{}", client_id, product, fetcher);
    let has_permission: bool = conn_get(&state.redis, &perm_key).await.unwrap_or(false);

    if has_permission {
        return Ok(Json(serde_json::json!({ "has_permission": true })));
    }

    // Check database
    let result = sqlx::query_scalar::<_, bool>(
        "SELECT is_enabled FROM product_permissions 
         WHERE client_id = $1 AND product = $2 AND fetcher = $3 AND is_enabled = true"
    )
    .bind(client_id)
    .bind(product)
    .bind(fetcher)
    .fetch_optional(&state.db)
    .await
    .map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

    let has_permission = result.unwrap_or(false);

    // Cache the result
    let _: () = conn_set(&state.redis, &perm_key, has_permission, 300).await.unwrap_or(());

    Ok(Json(serde_json::json!({ "has_permission": has_permission })))
}

async fn conn_get(client: &redis::Client, key: &str) -> Option<bool> {
    let mut conn = client.get_async().await.ok()?;
    let val: String = conn.get(key).await.ok()?;
    val.parse().ok()
}

async fn conn_set(client: &redis::Client, key: &str, value: bool, ttl: i64) -> Result<(), redis::RedisError> {
    let mut conn = client.get_async().await?;
    let _: () = conn.set_ex(key, value, ttl).await
}

// ============ MAIN ============

#[tokio::main]
async fn main() {
    // Initialize tracing
    tracing_subscriber::fmt()
        .with_max_level(tracing::Level::INFO)
        .init();

    info!("Starting Fetcher Gateway...");

    // Database connection
    let database_url = std::env::var("DATABASE_URL")
        .unwrap_or_else(|_| "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin".to_string());
    
    let db = PgPoolOptions::new()
        .max_connections(100)
        .acquire_timeout(Duration::from_secs(5))
        .connect(&database_url)
        .await
        .expect("Failed to connect to database");

    info!("Database connected");

    // Redis connection
    let redis_url = std::env::var("REDIS_URL")
        .unwrap_or_else(|_| "redis://localhost:6379".to_string());
    
    let redis = redis::Client::open(redis_url.as_str())
        .expect("Failed to connect to Redis");

    // Test Redis connection
    redis.get_async().await.expect("Failed to ping Redis");
    info!("Redis connected");

    // Initialize state
    let state = Arc::new(AppState {
        db,
        redis: redis.clone(),
        rate_limiter: Arc::new(RwLock::new(RateLimiter::new(1000, 60))),
        fetcher_cache: Arc::new(RwLock::new(FetcherCache::new(60))),
    });

    // CORS
    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);

    // Build router
    let app = Router::new()
        .route("/health", get(health_check))
        .route("/api/v1/fetch", post(fetch_data))
        .route("/api/v1/config/:product/:fetcher", get(get_fetcher_config))
        .route("/api/v1/metrics", get(get_metrics))
        .route("/api/v1/permission/check", post(check_permission))
        .layer(ServiceBuilder::new().layer(cors))
        .with_state(state);

    let port = std::env::var("FETCHER_PORT").unwrap_or_else(|_| "8093".to_string());
    
    let addr = format!("0.0.0.0:{}", port);
    info!("Starting server on {}", addr);

    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
