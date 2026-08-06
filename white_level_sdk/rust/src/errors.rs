//! Error types for White Level SDK

use thiserror::Error;

/// Result type alias
pub type Result<T> = std::result::Result<T, SdkError>;

/// SDK Error types
#[derive(Error, Debug)]
pub enum SdkError {
    #[error("Connection failed: {0}")]
    ConnectionFailed(String),

    #[error("Authentication failed: {0}")]
    AuthenticationFailed(String),

    #[error("Permission denied: {0}")]
    PermissionDenied(String),

    #[error("Network error: {0}")]
    NetworkError(#[from] reqwest::Error),

    #[error("Serialization error: {0}")]
    SerializationError(#[from] serde_json::Error),

    #[error("IO error: {0}")]
    IoError(#[from] std::io::Error),

    #[error("Timeout: {0}")]
    Timeout(String),

    #[error("Invalid configuration: {0}")]
    InvalidConfig(String),

    #[error("Rate limited: {0}")]
    RateLimited(String),

    #[error("Server error: {0}")]
    ServerError(String),

    #[error("Fetcher error: {0}")]
    FetcherError(String),

    #[error("Sync error: {0}")]
    SyncError(String),

    #[error("Command failed: {0}")]
    CommandFailed(String),

    #[error("Not connected")]
    NotConnected,

    #[error("Already connected")]
    AlreadyConnected,

    #[error("Invalid response: {0}")]
    InvalidResponse(String),

    #[error("Heartbeat failed: {0}")]
    HeartbeatFailed(String),

    #[error("Reconnection failed after {0} attempts")]
    ReconnectionFailed(u32),

    #[error("SSL error: {0}")]
    SslError(String),
}

impl SdkError {
    /// Check if error is retryable
    pub fn is_retryable(&self) -> bool {
        matches!(
            self,
            SdkError::NetworkError(_) | 
            SdkError::Timeout(_) |
            SdkError::RateLimited(_) |
            SdkError::HeartbeatFailed(_)
        )
    }

    /// Check if error is fatal (should not retry)
    pub fn is_fatal(&self) -> bool {
        matches!(
            self,
            SdkError::AuthenticationFailed(_) |
            SdkError::PermissionDenied(_) |
            SdkError::InvalidConfig(_) |
            SdkError::NotConnected |
            SdkError::SslError(_)
        )
    }
}
