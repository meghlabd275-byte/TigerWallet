//! Error types for Fiat Ramp service

use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("Invalid request: {0}")]
    InvalidRequest(String),
    
    #[error("Not found: {0}")]
    NotFound(String),
    
    #[error("Internal server error: {0}")]
    Internal(String),
    
    #[error("Database error: {0}")]
    Database(String),
    
    #[error("Payment failed: {0}")]
    PaymentFailed(String),
    
    #[error("KYC failed: {0}")]
    KycFailed(String),
    
    #[error("Compliance violation: {0}")]
    Compliance(String),
    
    #[error("Rate limit exceeded")]
    RateLimitExceeded,
    
    #[error("Authentication required")]
    Unauthorized,
}

impl serde::Serialize for Error {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(self.to_string().as_ref())
    }
}