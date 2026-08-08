//! Error handling module

use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::json;

#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("Unauthorized")]
    Unauthorized,
    
    #[error("Forbidden")]
    Forbidden,
    
    #[error("Not found: {0}")]
    NotFound(String),
    
    #[error("Bad request: {0}")]
    BadRequest(String),
    
    #[error("Internal server error: {0}")]
    InternalServerError(String),
    
    #[error("Database error: {0}")]
    DatabaseError(String),
    
    #[error("Authentication error: {0}")]
    AuthenticationError(String),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, error_message) = match self {
            AppError::Unauthorized => (StatusCode::UNAUTHORIZED, "Unauthorized".to_string()),
            AppError::Forbidden => (StatusCode::FORBIDDEN, "Forbidden".to_string()),
            AppError::NotFound(msg) => (StatusCode::NOT_FOUND, msg),
            AppError::BadRequest(msg) => (StatusCode::BAD_REQUEST, msg),
            AppError::InternalServerError(msg) => (StatusCode::INTERNAL_SERVER_ERROR, msg),
            AppError::DatabaseError(msg) => (StatusCode::INTERNAL_SERVER_ERROR, format!("Database error: {}", msg)),
            AppError::AuthenticationError(msg) => (StatusCode::UNAUTHORIZED, msg),
        };

        (
            status,
            Json(json!({
                "error": error_message,
            })),
        ).into_response()
    }
}

pub type AppResult<T> = Result<T, AppError>;
