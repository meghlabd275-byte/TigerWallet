//! Configuration for White Level SDK

use serde::{Deserialize, Serialize};
use std::time::Duration;

/// SDK Configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    /// Super Admin base URL
    pub super_admin_url: String,
    
    /// API Key for authentication
    pub api_key: String,
    
    /// Client ID
    pub client_id: Option<String>,
    
    /// Connection timeout
    #[serde(with = "serde_humanize")]
    pub connect_timeout: Duration,
    
    /// Request timeout
    #[serde(with = "serde_humanize")]
    pub request_timeout: Duration,
    
    /// Heartbeat interval
    #[serde(with = "serde_humanize")]
    pub heartbeat_interval: Duration,
    
    /// Max reconnection attempts
    pub max_reconnects: u32,
    
    /// Initial reconnection delay
    #[serde(with = "serde_humanize")]
    pub reconnect_delay: Duration,
    
    /// Max reconnection delay
    #[serde(with = "serde_humanize")]
    pub max_reconnect_delay: Duration,
    
    /// Rate limit (requests per second)
    pub rate_limit: u32,
    
    /// Enable logging
    pub logging: bool,
    
    /// Log level
    pub log_level: String,
    
    /// Enable caching
    pub caching: bool,
    
    /// Cache TTL
    #[serde(with = "serde_humanize")]
    pub cache_ttl: Duration,
    
    /// Retry count
    pub retry_count: u32,
    
    /// Enable TLS verification
    pub tls_verify: bool,
}

impl Config {
    /// Create new configuration
    pub fn new(super_admin_url: impl Into<String>, api_key: impl Into<String>) -> Self {
        Self {
            super_admin_url: super_admin_url.into(),
            api_key: api_key.into(),
            client_id: None,
            connect_timeout: Duration::from_secs(30),
            request_timeout: Duration::from_secs(60),
            heartbeat_interval: Duration::from_secs(30),
            max_reconnects: 10,
            reconnect_delay: Duration::from_secs(1),
            max_reconnect_delay: Duration::from_secs(300),
            rate_limit: 1000,
            logging: true,
            log_level: "info".to_string(),
            caching: true,
            cache_ttl: Duration::from_secs(300),
            retry_count: 3,
            tls_verify: true,
        }
    }

    /// Create configuration for development
    pub fn development(api_key: impl Into<String>) -> Self {
        Self::new("http://localhost:8082", api_key)
    }

    /// Create configuration for production
    pub fn production(api_key: impl Into<String>) -> Self {
        Self::new("https://superadmin.tigerwallet.com", api_key)
    }

    /// Set client ID
    pub fn with_client_id(mut self, client_id: impl Into<String>) -> Self {
        self.client_id = Some(client_id.into());
        self
    }

    /// Set connect timeout
    pub fn with_connect_timeout(mut self, timeout: Duration) -> Self {
        self.connect_timeout = timeout;
        self
    }

    /// Set request timeout
    pub fn with_request_timeout(mut self, timeout: Duration) -> Self {
        self.request_timeout = timeout;
        self
    }

    /// Set heartbeat interval
    pub fn with_heartbeat_interval(mut self, interval: Duration) -> Self {
        self.heartbeat_interval = interval;
        self
    }

    /// Set max reconnects
    pub fn with_max_reconnects(mut self, max: u32) -> Self {
        self.max_reconnects = max;
        self
    }

    /// Set rate limit
    pub fn with_rate_limit(mut self, limit: u32) -> Self {
        self.rate_limit = limit;
        self
    }

    /// Enable/disable logging
    pub fn with_logging(mut self, enabled: bool) -> Self {
        self.logging = enabled;
        self
    }

    /// Set log level
    pub fn with_log_level(mut self, level: impl Into<String>) -> Self {
        self.log_level = level.into();
        self
    }

    /// Enable/disable caching
    pub fn with_caching(mut self, enabled: bool) -> Self {
        self.caching = enabled;
        self
    }

    /// Set cache TTL
    pub fn with_cache_ttl(mut self, ttl: Duration) -> Self {
        self.cache_ttl = ttl;
        self
    }

    /// Set TLS verification
    pub fn with_tls_verify(mut self, verify: bool) -> Self {
        self.tls_verify = verify;
        self
    }

    /// Validate configuration
    pub fn validate(&self) -> Result<(), String> {
        if self.super_admin_url.is_empty() {
            return Err("super_admin_url cannot be empty".to_string());
        }
        if self.api_key.is_empty() {
            return Err("api_key cannot be empty".to_string());
        }
        if self.connect_timeout.as_secs() == 0 {
            return Err("connect_timeout must be greater than 0".to_string());
        }
        if self.request_timeout.as_secs() == 0 {
            return Err("request_timeout must be greater than 0".to_string());
        }
        if self.heartbeat_interval.as_secs() == 0 {
            return Err("heartbeat_interval must be greater than 0".to_string());
        }
        Ok(())
    }
}

impl Default for Config {
    fn default() -> Self {
        Self::new("http://localhost:8082", "")
    }
}

// Helper module for humanize duration serialization
mod serde_humanize {
    use serde::{Deserialize, Deserializer, Serializer};
    use std::time::Duration;

    /// Serialize Duration to seconds
    pub fn serialize<S>(duration: &Duration, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_u64(duration.as_secs())
    }

    /// Deserialize seconds to Duration
    pub fn deserialize<'de, D>(deserializer: D) -> Result<Duration, D::Error>
    where
        D: Deserializer<'de>,
    {
        let secs = u64::deserialize(deserializer)?;
        Ok(Duration::from_secs(secs))
    }
}
