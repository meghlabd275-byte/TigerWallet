//! Main White Level Client

use crate::config::Config;
use crate::connection::{ConnectionManager, ConnectionState};
use crate::errors::{Result, SdkError};
use crate::fetcher::FetcherManager;
use crate::permissions::{PermissionManager, get_default_permissions};
use crate::types::*;
use parking_lot::RwLock;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::mpsc;
use tokio::time::interval;

/// White Level Client main entry point
pub struct WhiteLevelClient {
    config: Config,
    connection: ConnectionManager,
    permissions: PermissionManager,
    fetchers: FetcherManager,
    product: WhiteLevelProduct,
    client_info: ClientInfo,
    running: Arc<RwLock<bool>>,
    event_receiver: Option<mpsc::Receiver<AdminEvent>>,
}

impl WhiteLevelClient {
    /// Create new White Level client
    pub async fn new(config: Config, product: WhiteLevelProduct) -> Result<Self> {
        config.validate().map_err(|e| SdkError::InvalidConfig(e))?;

        // Initialize default permissions
        let permissions = PermissionManager::new(config.clone());
        let default_perms = get_default_permissions(product.clone());
        for perm in default_perms {
            permissions.update_permission(perm);
        }

        let fetchers = FetcherManager::new(config.clone());
        
        // Get client info
        let client_info = ClientInfo {
            name: hostname::get()
                .map(|h| h.to_string_lossy().to_string())
                .unwrap_or_else(|_| "unknown".to_string()),
            version: env!("CARGO_PKG_VERSION").to_string(),
            platform: std::env::consts::OS.to_string(),
            hostname: hostname::get()
                .map(|h| h.to_string_lossy().to_string())
                .unwrap_or_else(|_| "unknown".to_string()),
            ip_address: None,
            metadata: None,
        };

        Ok(Self {
            config: config.clone(),
            connection: ConnectionManager::new(config),
            permissions,
            fetchers,
            product,
            client_info,
            running: Arc::new(RwLock::new(false)),
            event_receiver: None,
        })
    }

    /// Connect to Super Admin
    pub async fn connect(&mut self, client_id: &str) -> Result<ConnectionResponse> {
        let request = ConnectionRequest {
            client_id: uuid::Uuid::parse_str(client_id)
                .map_err(|_| SdkError::InvalidConfig("Invalid client_id".to_string()))?,
            product: self.product,
            api_key: self.config.api_key.clone(),
            client_info: self.client_info.clone(),
            ip_address: None,
            region: None,
        };

        let response = self.connection.connect(request).await?;

        // Initialize fetchers from config
        self.fetchers.initialize(response.config.fetchers.clone());

        // Load permissions
        self.permissions.load_permissions(client_id).await?;

        Ok(response)
    }

    /// Disconnect from Super Admin
    pub async fn disconnect(&self) -> Result<()> {
        self.connection.disconnect().await
    }

    /// Start heartbeat loop
    pub fn start_heartbeat(&self) {
        let mut running = self.running.write();
        if *running {
            return;
        }
        *running = true;
        drop(running);

        let connection = Arc::new(self.connection.get_state());
        let config = self.config.clone();
        
        tokio::spawn(async move {
            let mut ticker = interval(Duration::from_secs(30));
            
            while {
                ticker.tick().await;
                *self.running.read()
            } {
                let state = connection.clone();
                
                let heartbeat = Heartbeat {
                    connection_key: state.connection_key.clone().unwrap_or_default(),
                    status: ConnectionStatus::Connected,
                    latency_ms: 0,
                    metrics: ConnectionMetrics::default(),
                };

                if let Err(e) = self.connection.heartbeat(heartbeat).await {
                    tracing::error!("Heartbeat failed: {}", e);
                }
            }
        });
    }

    /// Stop heartbeat
    pub fn stop_heartbeat(&self) {
        let mut running = self.running.write();
        *running = false;
    }

    /// Fetch data from Super Admin
    pub async fn fetch(&self, fetcher: FetcherType, params: serde_json::Value) -> Result<FetcherResponse> {
        // Check permission
        if !self.permissions.has_permission(self.product, fetcher, PermissionLevel::Read) {
            return Err(SdkError::PermissionDenied(
                format!("No permission for {:?} fetcher", fetcher)
            ));
        }

        let request = FetcherRequest {
            fetcher,
            params,
            cache: true,
        };

        self.fetchers.fetch(request).await
    }

    /// Check if connected
    pub fn is_connected(&self) -> bool {
        self.connection.is_connected()
    }

    /// Get connection state
    pub fn get_connection_state(&self) -> ConnectionState {
        self.connection.get_state()
    }

    /// Get permissions
    pub fn get_permissions(&self) -> Vec<Permission> {
        self.permissions.get_all()
    }

    /// Check permission for fetcher
    pub fn has_permission(&self, fetcher: FetcherType, level: PermissionLevel) -> bool {
        self.permissions.has_permission(self.product, fetcher, level)
    }

    /// Get enabled fetchers
    pub fn get_enabled_fetchers(&self) -> Vec<FetcherType> {
        self.permissions.get_enabled_fetchers(self.product)
    }

    /// Clear cache
    pub fn clear_cache(&self) {
        self.fetchers.clear_cache();
    }

    /// Get cache statistics
    pub fn get_cache_stats(&self) -> crate::fetcher::CacheStats {
        self.fetchers.get_cache_stats()
    }

    /// Set custom client info
    pub fn set_client_info(&mut self, info: ClientInfo) {
        self.client_info = info;
    }
}

impl Default for ClientInfo {
    fn default() -> Self {
        Self {
            name: "unknown".to_string(),
            version: "1.0.0".to_string(),
            platform: "unknown".to_string(),
            hostname: "unknown".to_string(),
            ip_address: None,
            metadata: None,
        }
    }
}
