//! Error types for Security Platform

use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("Invalid request: {0}")]
    InvalidRequest(String),
    #[error("Audit error: {0}")]
    AuditError(String),
    #[error("Security violation: {0}")]
    SecurityViolation(String),
}

impl serde::Serialize for Error {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(self.to_string().as_ref())
    }
}