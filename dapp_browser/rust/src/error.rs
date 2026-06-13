//! Error types for DApp Browser

use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("Invalid request: {0}")]
    InvalidRequest(String),
    #[error("Not found: {0}")]
    NotFound(String),
    #[error("Internal server error: {0}")]
    Internal(String),
    #[error("WalletConnect error: {0}")]
    WalletConnect(String),
}

impl serde::Serialize for Error {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(self.to_string().as_ref())
    }
}