//! TigerWallet White Label System - High Speed Components
//! 
//! This module provides high-performance, ultra-low latency components
//! for the white label system using Rust for maximum throughput.

use std::sync::Arc;
use std::time::{Duration, Instant};
use std::collections::HashMap;

use parking_lot::RwLock;
use serde::{Deserialize, Serialize};
use uuid::Uuid;
use chrono::{DateTime, Utc};

// ============================================================================
// High-Speed Authentication Module
// ============================================================================

/// Authentication token with ultra-fast validation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthToken {
    pub id: String,
    pub admin_id: String,
    pub role: String,
    pub client_id: Option<String>,
    pub permissions: Vec<String>,
    pub issued_at: i64,
    pub expires_at: i64,
}

impl AuthToken {
    /// Fast token validation
    pub fn is_valid(&self) -> bool {
        let now = Utc::now().timestamp();
        now >= self.issued_at && now < self.expires_at
    }

    /// Check if token has specific permission
    pub fn has_permission(&self, permission: &str) -> bool {
        self.permissions.iter().any(|p| p == permission || p == "*")
    }

    /// Create new token
    pub fn new(admin_id: &str, role: &str, permissions: Vec<String>, ttl_seconds: i64) -> Self {
        let now = Utc::now().timestamp();
        Self {
            id: Uuid::new_v4().to_string(),
            admin_id: admin_id.to_string(),
            role: role.to_string(),
            client_id: None,
            permissions,
            issued_at: now,
            expires_at: now + ttl_seconds,
        }
    }
}

/// Authentication cache with O(1) lookups
pub struct AuthCache {
    tokens: Arc<RwLock<HashMap<String, AuthToken>>>,
    admin_sessions: Arc<RwLock<HashMap<String, Instant>>>,
}

impl AuthCache {
    pub fn new() -> Self {
        Self {
            tokens: Arc::new(RwLock::new(HashMap::new())),
            admin_sessions: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// Store token with O(1) lookup
    pub fn store_token(&self, token: &AuthToken) {
        let mut tokens = self.tokens.write();
        tokens.insert(token.id.clone(), token.clone());
        
        let mut sessions = self.admin_sessions.write();
        sessions.insert(token.admin_id.clone(), Instant::now());
    }

    /// Validate token - O(1) lookup
    pub fn validate_token(&self, token_id: &str) -> Option<AuthToken> {
        let tokens = self.tokens.read();
        let token = tokens.get(token_id)?;
        
        if token.is_valid() {
            Some(token.clone())
        } else {
            None
        }
    }

    /// Invalidate token
    pub fn invalidate_token(&self, token_id: &str) {
        let mut tokens = self.tokens.write();
        tokens.remove(token_id);
    }

    /// Invalidate all sessions for admin
    pub fn invalidate_admin_sessions(&self, admin_id: &str) {
        let mut tokens = self.tokens.write();
        tokens.retain(|_, token| token.admin_id != admin_id);
        
        let mut sessions = self.admin_sessions.write();
        sessions.remove(admin_id);
    }

    /// Get active session count
    pub fn active_session_count(&self) -> usize {
        let sessions = self.admin_sessions.read();
        
        // Clean up old sessions
        let now = Instant::now();
        let valid: Vec<_> = sessions
            .iter()
            .filter(|(_, last_activity)| now.duration_since(**last_activity) < Duration::from_secs(86400))
            .collect();
        
        valid.len()
    }
}

impl Default for AuthCache {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================================================
// High-Speed Rate Limiter
// ============================================================================

/// Token bucket rate limiter for API throttling
pub struct RateLimiter {
    buckets: Arc<RwLock<HashMap<String, TokenBucket>>>,
    config: RateLimiterConfig,
}

#[derive(Debug, Clone)]
pub struct RateLimiterConfig {
    pub max_requests_per_minute: u32,
    pub max_requests_per_hour: u32,
    pub burst_size: u32,
}

impl Default for RateLimiterConfig {
    fn default() -> Self {
        Self {
            max_requests_per_minute: 1000,
            max_requests_per_hour: 50000,
            burst_size: 100,
        }
    }
}

struct TokenBucket {
    tokens: f64,
    last_refill: Instant,
    requests_count: u64,
    requests_allowed: u64,
    requests_rejected: u64,
}

impl RateLimiter {
    pub fn new(config: RateLimiterConfig) -> Self {
        Self {
            buckets: Arc::new(RwLock::new(HashMap::new())),
            config,
        }
    }

    /// Check if request is allowed - O(1) operation
    pub fn is_allowed(&self, key: &str) -> RateLimitResult {
        let mut buckets = self.buckets.write();
        
        let bucket = buckets.entry(key.to_string()).or_insert_with(|| TokenBucket {
            tokens: self.config.burst_size as f64,
            last_refill: Instant::now(),
            requests_count: 0,
            requests_allowed: 0,
            requests_rejected: 0,
        });
        
        // Refill tokens
        let elapsed = bucket.last_refill.elapsed().as_secs_f64();
        let refill_rate = self.config.max_requests_per_minute as f64 / 60.0;
        bucket.tokens = (bucket.tokens + elapsed * refill_rate).min(self.config.burst_size as f64);
        bucket.last_refill = Instant::now();
        
        // Try to consume token
        if bucket.tokens >= 1.0 {
            bucket.tokens -= 1.0;
            bucket.requests_count += 1;
            bucket.requests_allowed += 1;
            
            RateLimitResult {
                allowed: true,
                remaining: bucket.tokens as u32,
                reset_after: Duration::from_secs_f64(1.0 / refill_rate),
            }
        } else {
            bucket.requests_count += 1;
            bucket.requests_rejected += 1;
            
            RateLimitResult {
                allowed: false,
                remaining: 0,
                reset_after: Duration::from_secs_f64(1.0 / bucket.tokens.abs()),
            }
        }
    }

    /// Get stats for a key
    pub fn get_stats(&self, key: &str) -> Option<RateLimitStats> {
        let buckets = self.buckets.read();
        let bucket = buckets.get(key)?;
        
        Some(RateLimitStats {
            total_requests: bucket.requests_count,
            allowed_requests: bucket.requests_allowed,
            rejected_requests: bucket.requests_rejected,
            current_tokens: bucket.tokens,
        })
    }
}

#[derive(Debug, Clone, Serialize)]
pub struct RateLimitResult {
    pub allowed: bool,
    pub remaining: u32,
    pub reset_after: Duration,
}

#[derive(Debug, Clone, Serialize)]
pub struct RateLimitStats {
    pub total_requests: u64,
    pub allowed_requests: u64,
    pub rejected_requests: u64,
    pub current_tokens: f64,
}

// ============================================================================
// High-Speed Cache
// ============================================================================

/// In-memory cache with TTL support
pub struct Cache {
    entries: Arc<RwLock<HashMap<String, CacheEntry>>>,
    max_size: usize,
}

struct CacheEntry {
    value: Vec<u8>,
    expires_at: Option<Instant>,
    access_count: u64,
    created_at: Instant,
}

impl Cache {
    pub fn new(max_size: usize) -> Self {
        Self {
            entries: Arc::new(RwLock::new(HashMap::new())),
            max_size,
        }
    }

    /// Set value with TTL
    pub fn set(&self, key: &str, value: Vec<u8>, ttl_secs: Option<u64>) {
        let mut entries = self.entries.write();
        
        // Evict if needed
        while entries.len() >= self.max_size {
            if let Some(oldest) = entries
                .iter()
                .min_by_key(|(_, e)| e.created_at)
                .map(|(k, _)| k.clone())
            {
                entries.remove(&oldest);
            } else {
                break;
            }
        }
        
        let expires_at = ttl_secs.map(|s| Instant::now() + Duration::from_secs(s));
        
        entries.insert(key.to_string(), CacheEntry {
            value,
            expires_at,
            access_count: 0,
            created_at: Instant::now(),
        });
    }

    /// Get value - O(1) lookup
    pub fn get(&self, key: &str) -> Option<Vec<u8>> {
        let mut entries = self.entries.write();
        let entry = entries.get(key)?;
        
        // Check expiry
        if let Some(expires_at) = entry.expires_at {
            if Instant::now() > expires_at {
                entries.remove(key);
                return None;
            }
        }
        
        // Update access count
        let entry = entries.get_mut(key).unwrap();
        entry.access_count += 1;
        
        Some(entry.value.clone())
    }

    /// Check if key exists and is valid
    pub fn contains(&self, key: &str) -> bool {
        self.get(key).is_some()
    }

    /// Delete key
    pub fn delete(&self, key: &str) {
        let mut entries = self.entries.write();
        entries.remove(key);
    }

    /// Clear all entries
    pub fn clear(&self) {
        let mut entries = self.entries.write();
        entries.clear();
    }

    /// Get cache stats
    pub fn stats(&self) -> CacheStats {
        let entries = self.entries.read();
        
        let total_accesses: u64 = entries.values().map(|e| e.access_count).sum();
        
        CacheStats {
            size: entries.len(),
            max_size: self.max_size,
            total_accesses,
        }
    }
}

#[derive(Debug, Clone, Serialize)]
pub struct CacheStats {
    pub size: usize,
    pub max_size: usize,
    pub total_accesses: u64,
}

// ============================================================================
// Secure Password Handling
// ============================================================================

/// Secure password hasher using bcrypt
pub struct PasswordHasher;

impl PasswordHasher {
    /// Hash password with bcrypt - production ready
    pub fn hash(password: &str) -> Result<String, PasswordHashError> {
        // Use bcrypt for production
        let salt = bcrypt::gen_salt(12);
        bcrypt::hash(password, salt)
            .map_err(|e| PasswordHashError::HashFailed(e.to_string()))
    }

    /// Verify password against hash
    pub fn verify(password: &str, hash: &str) -> bool {
        bcrypt::verify(password, hash).unwrap_or(false)
    }

    /// Hash with argon2 - even more secure
    pub fn hash_argon2(password: &str) -> Result<String, PasswordHashError> {
        use argon2::{
            password_hash::{PasswordHasher, SaltString},
            Argon2,
        };
        
        let salt = SaltString::generate(&mut rand::thread_rng());
        let argon2 = Argon2::default();
        
        argon2
            .hash_password(password.as_bytes(), &salt)
            .map(|h| h.to_string())
            .map_err(|e| PasswordHashError::HashFailed(e.to_string()))
    }
}

#[derive(Debug, thiserror::Error)]
pub enum PasswordHashError {
    #[error("Password hashing failed: {0}")]
    HashFailed(String),
}

// ============================================================================
// API Key Management
// ============================================================================

/// API Key with fast validation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct APIKey {
    pub id: String,
    pub client_id: String,
    pub name: String,
    pub key_hash: String,
    pub permissions: Vec<String>,
    pub rate_limit: u32,
    pub status: APIKeyStatus,
    pub created_at: DateTime<Utc>,
    pub expires_at: Option<DateTime<Utc>>,
    pub last_used: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum APIKeyStatus {
    Active,
    Suspended,
    Revoked,
    Expired,
}

impl APIKey {
    /// Create new API key
    pub fn new(client_id: &str, name: &str, permissions: Vec<String>, rate_limit: u32) -> Self {
        // Generate random key
        let key_bytes: [u8; 32] = rand::random();
        let key = base64::Engine::encode(&base64::engine::general_purpose::STANDARD, &key_bytes);
        let key_hash = format!("{:x}", sha2::Sha256::digest(&key));
        
        Self {
            id: Uuid::new_v4().to_string(),
            client_id: client_id.to_string(),
            name: name.to_string(),
            key_hash,
            permissions,
            rate_limit,
            status: APIKeyStatus::Active,
            created_at: Utc::now(),
            expires_at: None,
            last_used: None,
        }
    }

    /// Validate API key
    pub fn validate(&self, provided_key: &str) -> bool {
        if self.status != APIKeyStatus::Active {
            return false;
        }
        
        if let Some(expires_at) = self.expires_at {
            if Utc::now() > expires_at {
                return false;
            }
        }
        
        let hash = format!("{:x}", sha2::Sha256::digest(provided_key.as_bytes()));
        hash == self.key_hash
    }

    /// Check permission
    pub fn has_permission(&self, permission: &str) -> bool {
        self.permissions.iter().any(|p| p == permission || p == "*")
    }
}

/// API Key storage with fast lookups
pub struct APIKeyStore {
    keys: Arc<RwLock<HashMap<String, APIKey>>>,
    key_hash_index: Arc<RwLock<HashMap<String, String>>>,
}

impl APIKeyStore {
    pub fn new() -> Self {
        Self {
            keys: Arc::new(RwLock::new(HashMap::new())),
            key_hash_index: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// Store API key
    pub fn store(&self, key: &APIKey) {
        let mut keys = self.keys.write();
        keys.insert(key.id.clone(), key.clone());
        
        let mut index = self.key_hash_index.write();
        index.insert(key.key_hash.clone(), key.id.clone());
    }

    /// Get API key by ID
    pub fn get(&self, id: &str) -> Option<APIKey> {
        let keys = self.keys.read();
        keys.get(id).cloned()
    }

    /// Validate API key
    pub fn validate(&self, provided_key: &str) -> Option<APIKey> {
        let hash = format!("{:x}", sha2::Sha256::digest(provided_key.as_bytes()));
        
        let index = self.key_hash_index.read();
        let key_id = index.get(&hash)?;
        
        let keys = self.keys.read();
        let key = keys.get(key_id)?;
        
        if key.validate(provided_key) {
            Some(key.clone())
        } else {
            None
        }
    }

    /// Revoke API key
    pub fn revoke(&self, id: &str) -> bool {
        let mut keys = self.keys.write();
        if let Some(key) = keys.get_mut(id) {
            key.status = APIKeyStatus::Revoked;
            return true;
        }
        false
    }
}

impl Default for APIKeyStore {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================================================
// Feature Flags
// ============================================================================

/// Feature flag system with fast lookups
pub struct FeatureFlags {
    flags: Arc<RwLock<HashMap<String, FeatureFlag>>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeatureFlag {
    pub name: String,
    pub enabled: bool,
    pub rollout_percentage: u8,  // 0-100
    pub client_overrides: HashMap<String, bool>,
}

impl FeatureFlags {
    pub fn new() -> Self {
        Self {
            flags: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// Set feature flag
    pub fn set(&self, name: &str, enabled: bool, rollout_percentage: u8) {
        let mut flags = self.flags.write();
        
        if let Some(flag) = flags.get_mut(name) {
            flag.enabled = enabled;
            flag.rollout_percentage = rollout_percentage;
        } else {
            flags.insert(name.to_string(), FeatureFlag {
                name: name.to_string(),
                enabled,
                rollout_percentage,
                client_overrides: HashMap::new(),
            });
        }
    }

    /// Check if feature is enabled for client
    pub fn is_enabled(&self, name: &str, client_id: Option<&str>) -> bool {
        let flags = self.flags.read();
        
        if let Some(flag) = flags.get(name) {
            // Check client override first
            if let Some(cid) = client_id {
                if let Some(override_value) = flag.client_overrides.get(cid) {
                    return *override_value;
                }
            }
            
            // Check rollout percentage
            if flag.rollout_percentage < 100 {
                // Simple deterministic rollout based on client_id hash
                let hash = if let Some(cid) = client_id {
                    cid.bytes().fold(0u64, |acc, b| acc.wrapping_mul(31).wrapping_add(b as u64))
                } else {
                    rand::random()
                };
                
                return (hash % 100) < flag.rollout_percentage as u64;
            }
            
            flag.enabled
        } else {
            false
        }
    }

    /// Set client-specific override
    pub fn set_override(&self, name: &str, client_id: &str, enabled: bool) {
        let mut flags = self.flags.write();
        if let Some(flag) = flags.get_mut(name) {
            flag.client_overrides.insert(client_id.to_string(), enabled);
        }
    }
}

impl Default for FeatureFlags {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================================================
// Library Exports
// ============================================================================

pub mod prelude {
    pub use super::*;
}
