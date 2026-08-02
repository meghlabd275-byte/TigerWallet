//! TigerWallet White Label System - High Speed Service
//! 
//! This is the main entry point for the high-speed Rust service
//! that handles authentication, rate limiting, and caching.

use std::sync::Arc;
use std::net::SocketAddr;
use std::time::Duration;

use axum::{
    Router,
    routing::{get, post, put, delete},
    Json, Extract,
    http::{StatusCode, HeaderMap, header},
    response::IntoResponse,
};
use tower_http::cors::{CorsLayer, Any};
use serde::{Deserialize, Serialize};
use tracing_subscriber;

use white_label_high_speed::*;

// ============================================================================
// Application State
// ============================================================================

pub struct AppState {
    pub auth_cache: AuthCache,
    pub rate_limiter: RateLimiter,
    pub cache: Cache,
    pub api_keys: APIKeyStore,
    pub features: FeatureFlags,
}

impl AppState {
    pub fn new() -> Self {
        Self {
            auth_cache: AuthCache::new(),
            rate_limiter: RateLimiter::new(RateLimiterConfig {
                max_requests_per_minute: 1000,
                max_requests_per_hour: 50000,
                burst_size: 100,
            }),
            cache: Cache::new(10000),
            api_keys: APIKeyStore::new(),
            features: FeatureFlags::new(),
        }
    }
}

// ============================================================================
// API Request/Response Types
// ============================================================================

#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub email: String,
    pub password: String,
}

#[derive(Debug, Serialize)]
pub struct LoginResponse {
    pub token: String,
    pub admin_id: String,
    pub role: String,
    pub permissions: Vec<String>,
}

#[derive(Debug, Deserialize)]
pub struct CreateAPIKeyRequest {
    pub client_id: String,
    pub name: String,
    pub permissions: Vec<String>,
    pub rate_limit: u32,
}

#[derive(Debug, Serialize)]
pub struct APIKeyResponse {
    pub id: String,
    pub key: String,
    pub name: String,
    pub permissions: Vec<String>,
    pub rate_limit: u32,
    pub status: String,
}

#[derive(Debug, Serialize)]
pub struct RateLimitResponse {
    pub allowed: bool,
    pub remaining: u32,
    pub reset_after_ms: u64,
}

#[derive(Debug, Serialize)]
pub struct CacheStatsResponse {
    pub size: usize,
    pub max_size: usize,
    pub total_accesses: u64,
}

#[derive(Debug, Serialize)]
pub struct FeatureFlagResponse {
    pub name: String,
    pub enabled: bool,
    pub rollout_percentage: u8,
}

// ============================================================================
// API Handlers
// ============================================================================

// Health check
async fn health_check() -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "status": "healthy",
        "service": "white-label-high-speed",
        "timestamp": chrono::Utc::now().to_rfc3339()
    }))
}

// Login handler
async fn login(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<LoginRequest>,
) -> Result<Json<LoginResponse>, StatusCode> {
    // For demo - in production this would validate against database
    // Create demo admin
    let admin_id = Uuid::new_v4().to_string();
    let permissions = vec![
        "clients.read".to_string(),
        "clients.write".to_string(),
        "admins.read".to_string(),
        "products.read".to_string(),
    ];
    
    let token = AuthToken::new(&admin_id, "super_admin", permissions.clone(), 86400);
    let token_id = token.id.clone();
    
    // Store in cache
    state.auth_cache.store_token(&token);
    
    Ok(Json(LoginResponse {
        token: token_id,
        admin_id,
        role: "super_admin".to_string(),
        permissions,
    }))
}

// Logout handler
async fn logout(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let token = headers
        .get(header::AUTHORIZATION)
        .and_then(|v| v.to_str().ok())
        .and_then(|v| v.strip_prefix("Bearer "));
    
    if let Some(token_id) = token {
        state.auth_cache.invalidate_token(token_id);
    }
    
    Ok(Json(serde_json::json!({"message": "Logged out"})))
}

// Rate limit check
async fn check_rate_limit(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> Result<Json<RateLimitResponse>, StatusCode> {
    let key = headers
        .get("x-api-key")
        .or_else(|| headers.get("x-forwarded-for"))
        .and_then(|v| v.to_str().ok())
        .unwrap_or("anonymous");
    
    let result = state.rate_limiter.is_allowed(key);
    
    Ok(Json(RateLimitResponse {
        allowed: result.allowed,
        remaining: result.remaining,
        reset_after_ms: result.reset_after.as_millis() as u64,
    }))
}

// Create API key
async fn create_api_key(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<CreateAPIKeyRequest>,
) -> Result<Json<APIKeyResponse>, StatusCode> {
    let key = APIKey::new(
        &payload.client_id,
        &payload.name,
        payload.permissions.clone(),
        payload.rate_limit,
    );
    
    let key_id = key.id.clone();
    let key_value = format!("wl_{}", base64::Engine::encode(
        &base64::engine::general_purpose::STANDARD,
        &rand::random::<[u8; 32]>()[..]
    ));
    
    // Store key
    state.api_keys.store(&key);
    
    Ok(Json(APIKeyResponse {
        id: key_id,
        key: key_value,
        name: payload.name,
        permissions: payload.permissions,
        rate_limit: payload.rate_limit,
        status: "active".to_string(),
    }))
}

// Validate API key
async fn validate_api_key(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let key = headers
        .get("x-api-key")
        .and_then(|v| v.to_str().ok())
        .ok_or(StatusCode::UNAUTHORIZED)?;
    
    if let Some(api_key) = state.api_keys.validate(key) {
        Ok(Json(serde_json::json!({
            "valid": true,
            "client_id": api_key.client_id,
            "permissions": api_key.permissions,
        })))
    } else {
        Ok(Json(serde_json::json!({
            "valid": false,
        })))
    }
}

// Cache stats
async fn get_cache_stats(
    State(state): State<Arc<AppState>>,
) -> Json<CacheStatsResponse> {
    let stats = state.cache.stats();
    Json(CacheStatsResponse {
        size: stats.size,
        max_size: stats.max_size,
        total_accesses: stats.total_accesses,
    })
}

// Set cache value
async fn set_cache(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<serde_json::Value>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let key = payload["key"].as_str().ok_or(StatusCode::BAD_REQUEST)?;
    let value = payload["value"].as_str().ok_or(StatusCode::BAD_REQUEST)?;
    let ttl = payload["ttl"].as_u64();
    
    state.cache.set(key, value.as_bytes().to_vec(), ttl);
    
    Ok(Json(serde_json::json!({"success": true})))
}

// Get cache value
async fn get_cache(
    State(state): State<Arc<AppState>>,
    axum::extract::Path(key): axum::extract::Path<String>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    if let Some(value) = state.cache.get(&key) {
        let value_str = String::from_utf8_lossy(&value);
        Ok(Json(serde_json::json!({
            "found": true,
            "value": value_str,
        })))
    } else {
        Ok(Json(serde_json::json!({
            "found": false,
        })))
    }
}

// Feature flags - list all
async fn list_features(
    State(state): State<Arc<AppState>>,
) -> Json<Vec<FeatureFlagResponse>> {
    // Return default features
    let features = vec![
        FeatureFlagResponse {
            name: "trading".to_string(),
            enabled: true,
            rollout_percentage: 100,
        },
        FeatureFlagResponse {
            name: "staking".to_string(),
            enabled: true,
            rollout_percentage: 100,
        },
        FeatureFlagResponse {
            name: "nft".to_string(),
            enabled: true,
            rollout_percentage: 100,
        },
        FeatureFlagResponse {
            name: "bridge".to_string(),
            enabled: true,
            rollout_percentage: 100,
        },
    ];
    
    Json(features)
}

// Set feature flag
async fn set_feature(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<serde_json::Value>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let name = payload["name"].as_str().ok_or(StatusCode::BAD_REQUEST)?;
    let enabled = payload["enabled"].as_bool().unwrap_or(true);
    let rollout = payload["rollout_percentage"].as_u64().unwrap_or(100) as u8;
    
    state.features.set(name, enabled, rollout);
    
    Ok(Json(serde_json::json!({"success": true})))
}

// Check feature flag
async fn check_feature(
    State(state): State<Arc<AppState>>,
    axum::extract::Path(name): axum::extract::Path<String>,
    axum::extract::Query(query): axum::extract::Query<serde_json::Value>,
) -> Json<serde_json::Value> {
    let client_id = query.get("client_id").and_then(|v| v.as_str());
    
    let enabled = state.features.is_enabled(&name, client_id);
    
    Json(serde_json::json!({
        "name": name,
        "enabled": enabled,
    }))
}

// ============================================================================
// Main Entry Point
// ============================================================================

#[tokio::main]
async fn main() {
    // Initialize logging
    tracing_subscriber::fmt()
        .with_target(false)
        .with_env_filter("info")
        .init();
    
    tracing::info!("Starting TigerWallet White Label High-Speed Service");
    
    // Create application state
    let state = Arc::new(AppState::new());
    
    // CORS configuration
    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);
    
    // Build router
    let app = Router::new()
        .route("/health", get(health_check))
        .route("/api/v1/auth/login", post(login))
        .route("/api/v1/auth/logout", post(logout))
        .route("/api/v1/rate-limit", get(check_rate_limit))
        .route("/api/v1/api-keys", post(create_api_key))
        .route("/api/v1/api-keys/validate", post(validate_api_key))
        .route("/api/v1/cache/stats", get(get_cache_stats))
        .route("/api/v1/cache", post(set_cache))
        .route("/api/v1/cache/:key", get(get_cache))
        .route("/api/v1/features", get(list_features))
        .route("/api/v1/features", post(set_feature))
        .route("/api/v1/features/:name", get(check_feature))
        .layer(cors)
        .with_state(state);
    
    // Start server
    let addr = SocketAddr::from(([0, 0, 0, 0], 8080));
    tracing::info!("Server listening on {}", addr);
    
    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
