//! Error types for AI Platform

use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("Invalid request: {0}")]
    InvalidRequest(String),
    #[error("Model error: {0}")]
    ModelError(String),
    #[error("Prediction error: {0}")]
    PredictionError(String),
}

impl serde::Serialize for Error {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(self.to_string().as_ref())
    }
}