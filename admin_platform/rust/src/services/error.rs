/**
 * TigerWallet Admin Platform - Error Types
 */

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceError {
    pub code: String,
    pub message: String,
    pub status: u16,
}

impl ServiceError {
    pub fn new(code: &str, message: &str, status: u16) -> Self {
        Self {
            code: code.to_string(),
            message: message.to_string(),
            status,
        }
    }
    
    pub fn not_found(resource: &str) -> Self {
        Self::new("NOT_FOUND", &format!("{} not found", resource), 404)
    }
    
    pub fn unauthorized() -> Self {
        Self::new("UNAUTHORIZED", "Invalid credentials", 401)
    }
    
    pub fn forbidden() -> Self {
        Self::new("FORBIDDEN", "Access denied", 403)
    }
    
    pub fn bad_request(message: &str) -> Self {
        Self::new("BAD_REQUEST", message, 400)
    }
    
    pub fn internal_error(message: &str) -> Self {
        Self::new("INTERNAL_ERROR", message, 500)
    }
}

impl std::fmt::Display for ServiceError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}: {}", self.code, self.message)
    }
}

impl std::error::Error for ServiceError {}
