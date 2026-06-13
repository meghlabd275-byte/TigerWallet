//! Error types for Backend Services

use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("Invalid request: {0}")]
    InvalidRequest(String),
    #[error("Not found: {0}")]
    NotFound(String),
    #[error("Internal server error: {0}")]
    Internal(String),
    #[error("Unauthorized")]
    Unauthorized,
    #[error("Database error: {0}")]
    Database(String),
}

impl serde::Serialize for Error {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(self.to_string().as_ref())
    }
}