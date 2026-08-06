//! Connection management for White Level SDK

use crate::types::*;
use crate::errors::{Result, SdkError};
use crate::config::Config;
use async_trait::async_trait;
use parking_lot::RwLock;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::mpsc;
use tokio::time::sleep;

/// Connection state
#[derive(Debug, Clone)]
pub struct ConnectionState {
    pub status: ConnectionStatus,
    pub connection_id: Option<uuid::Uuid>,
    pub connection_key: Option<String>,
    pub session_token: Option<String>,
    pub config: Option<ClientConfig>,
    pub connected_at: Option<Instant>,
    pub last_heartbeat: Option<Instant>,
    pub reconnect_attempts: u32,
}

/// Connection event
#[derive(Debug, Clone)]
pub enum ConnectionEvent {
    Connected(ConnectionResponse),
    Disconnected { reason: String },
    Reconnecting { attempt: u32 },
    Error(String),
    ConfigUpdated(ClientConfig),
    Command(RemoteCommand),
}

/// Connection event handler
#[async_trait]
pub trait ConnectionEventHandler: Send + Sync {
    async fn on_connected(&self, response: ConnectionResponse);
    async fn on_disconnected(&self, reason: &str);
    async fn on_reconnecting(&self, attempt: u32);
    async fn on_error(&self, error: &str);
    async fn on_config_update(&self, config: ClientConfig);
    async fn on_command(&self, command: RemoteCommand);
}

/// Default connection handler that does nothing
#[async_trait]
impl ConnectionEventHandler for () {
    async fn on_connected(&self, _: ConnectionResponse) {}
    async fn on_disconnected(&self, _: &str) {}
    async fn on_reconnecting(&self, _: u32) {}
    async fn on_error(&self, _: &str) {}
    async fn on_config_update(&self, _: ClientConfig) {}
    async fn on_command(&self, _: RemoteCommand) {}
}

/// Connection manager
pub struct ConnectionManager {
    config: Config,
    state: Arc<RwLock<ConnectionState>>,
    event_sender: Option<mpsc::Sender<ConnectionEvent>>,
    http_client: reqwest::Client,
}

impl ConnectionManager {
    /// Create new connection manager
    pub fn new(config: Config) -> Self {
        let http_client = reqwest::Client::builder()
            .timeout(config.request_timeout)
            .connect_timeout(config.connect_timeout)
            .build()
            .expect("Failed to create HTTP client");

        Self {
            config,
            state: Arc::new(RwLock::new(ConnectionState {
                status: ConnectionStatus::Disconnected,
                connection_id: None,
                connection_key: None,
                session_token: None,
                config: None,
                connected_at: None,
                last_heartbeat: None,
                reconnect_attempts: 0,
            })),
            event_sender: None,
            http_client,
        }
    }

    /// Connect to Super Admin
    pub async fn connect(&self, request: ConnectionRequest) -> Result<ConnectionResponse> {
        let url = format!("{}/api/v1/connect", self.config.super_admin_url);
        
        let response = self.http_client
            .post(&url)
            .header("Authorization", format!("Bearer {}", self.config.api_key))
            .header("X-API-Key", &self.config.api_key)
            .json(&request)
            .send()
            .await?;

        if !response.status().is_success() {
            let status = response.status();
            let error_text = response.text().await.unwrap_or_default();
            
            if status.as_u16() == 401 {
                return Err(SdkError::AuthenticationFailed(error_text));
            } else if status.as_u16() == 429 {
                return Err(SdkError::RateLimited(error_text));
            } else {
                return Err(SdkError::ConnectionFailed(error_text));
            }
        }

        let connection_response: ConnectionResponse = response.json().await?;

        // Update state
        {
            let mut state = self.state.write();
            state.status = ConnectionStatus::Connected;
            state.connection_id = Some(connection_response.connection_id);
            state.connection_key = Some(connection_response.connection_key.clone());
            state.session_token = Some(connection_response.session_token.clone());
            state.config = Some(connection_response.config.clone());
            state.connected_at = Some(Instant::now());
            state.reconnect_attempts = 0;
        }

        Ok(connection_response)
    }

    /// Disconnect from Super Admin
    pub async fn disconnect(&self) -> Result<()> {
        let state = self.state.read();
        
        if let Some(connection_key) = &state.connection_key {
            let url = format!("{}/api/v1/disconnect", self.config.super_admin_url);
            
            let _ = self.http_client
                .post(&url)
                .header("Authorization", format!("Bearer {}", state.session_token.as_ref().unwrap()))
                .header("X-API-Key", &self.config.api_key)
                .json(&serde_json::json!({ "connection_key": connection_key }))
                .send()
                .await;
        }

        // Update state
        drop(state);
        {
            let mut state = self.state.write();
            state.status = ConnectionStatus::Disconnected;
            state.connection_id = None;
            state.connection_key = None;
            state.session_token = None;
            state.config = None;
            state.connected_at = None;
        }

        Ok(())
    }

    /// Send heartbeat
    pub async fn heartbeat(&self, heartbeat: Heartbeat) -> Result<()> {
        let state = self.state.read();
        
        let Some(connection_key) = &state.connection_key else {
            return Err(SdkError::NotConnected);
        };

        let url = format!("{}/api/v1/heartbeat", self.config.super_admin_url);
        
        let response = self.http_client
            .post(&url)
            .header("Authorization", format!("Bearer {}", state.session_token.as_ref().unwrap()))
            .header("X-API-Key", &self.config.api_key)
            .json(&heartbeat)
            .send()
            .await?;

        if !response.status().is_success() {
            let error_text = response.text().await.unwrap_or_default();
            return Err(SdkError::HeartbeatFailed(error_text));
        }

        // Update last heartbeat time
        drop(state);
        {
            let mut state = self.state.write();
            state.last_heartbeat = Some(Instant::now());
        }

        Ok(())
    }

    /// Reconnect with exponential backoff
    pub async fn reconnect(&self, request: ConnectionRequest) -> Result<ConnectionResponse> {
        let mut attempt = 0;
        let mut delay = self.config.reconnect_delay;

        loop {
            attempt += 1;
            
            // Send reconnection event
            if let Some(sender) = &self.event_sender {
                let _ = sender.send(ConnectionEvent::Reconnecting { attempt }).await;
            }

            match self.connect(request.clone()).await {
                Ok(response) => return Ok(response),
                Err(e) if e.is_retryable() && attempt < self.config.max_reconnects => {
                    sleep(delay).await;
                    delay = std::cmp::min(delay * 2, self.config.max_reconnect_delay);
                }
                Err(e) => {
                    // Update state
                    {
                        let mut state = self.state.write();
                        state.status = ConnectionStatus::Error;
                        state.reconnect_attempts = attempt;
                    }
                    return Err(SdkError::ReconnectionFailed(attempt));
                }
            }
        }
    }

    /// Get current connection state
    pub fn get_state(&self) -> ConnectionState {
        self.state.read().clone()
    }

    /// Check if connected
    pub fn is_connected(&self) -> bool {
        let state = self.state.read();
        state.status == ConnectionStatus::Connected
    }

    /// Get connection key
    pub fn get_connection_key(&self) -> Option<String> {
        self.state.read().connection_key.clone()
    }

    /// Get session token
    pub fn get_session_token(&self) -> Option<String> {
        self.state.read().session_token.clone()
    }

    /// Get client config
    pub fn get_config(&self) -> Option<ClientConfig> {
        self.state.read().config.clone()
    }

    /// Update connection key
    pub fn set_connection_key(&self, key: String) {
        let mut state = self.state.write();
        state.connection_key = Some(key);
    }

    /// Set event sender
    pub fn set_event_sender(&mut self, sender: mpsc::Sender<ConnectionEvent>) {
        self.event_sender = Some(sender);
    }
}

impl Default for ConnectionState {
    fn default() -> Self {
        Self {
            status: ConnectionStatus::Disconnected,
            connection_id: None,
            connection_key: None,
            session_token: None,
            config: None,
            connected_at: None,
            last_heartbeat: None,
            reconnect_attempts: 0,
        }
    }
}
