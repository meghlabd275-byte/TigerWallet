//! Error types for Staking Hub

use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("Invalid request: {0}")]
    InvalidRequest(String),
    #[error("Not found: {0}")]
    NotFound(String),
    #[error("Insufficient balance: {0}")]
    InsufficientBalance(String),
    #[error("Staking error: {0}")]
    StakingError(String),
    #[error("Unstaking error: {0}")]
    UnstakingError(String),
}

impl serde::Serialize for Error {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(self.to_string().as_ref())
    }
}