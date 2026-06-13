//! Error types for Token Scanner

use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("Invalid request: {0}")]
    InvalidRequest(String),
    #[error("Analysis error: {0}")]
    AnalysisError(String),
    #[error("Contract error: {0}")]
    ContractError(String),
}

impl serde::Serialize for Error {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(self.to_string().as_ref())
    }
}