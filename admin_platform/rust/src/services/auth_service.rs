use crate::database::Database;
use crate::error::Error;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoginRequest {
    pub email: String,
    pub password: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoginResponse {
    pub token: String,
    pub refresh_token: String,
    pub expires_in: u64,
    pub admin: AdminInfo,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminInfo {
    pub id: String,
    pub username: String,
    pub email: String,
    pub role: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenClaims {
    pub sub: String,
    pub email: String,
    pub role: String,
    pub exp: i64,
    pub iat: i64,
}

pub struct AuthService {
    db: Database,
    jwt_secret: String,
    jwt_ttl: u64,
}

impl AuthService {
    pub fn new(db: Database, jwt_secret: String, jwt_ttl: u64) -> Self {
        Self { db, jwt_secret, jwt_ttl }
    }

    pub async fn login(&self, email: &str, password: &str) -> Result<LoginResponse, Error> {
        Ok(LoginResponse {
            token: "jwt_token_placeholder".to_string(),
            refresh_token: "refresh_token_placeholder".to_string(),
            expires_in: self.jwt_ttl,
            admin: AdminInfo {
                id: uuid::Uuid::new_v4().to_string(),
                username: "admin".to_string(),
                email: email.to_string(),
                role: "super_admin".to_string(),
            },
        })
    }

    pub async fn logout(&self, admin_id: &str) -> Result<bool, Error> {
        Ok(true)
    }

    pub async fn refresh_token(&self, refresh_token: &str) -> Result<LoginResponse, Error> {
        Ok(LoginResponse {
            token: "jwt_token_placeholder".to_string(),
            refresh_token: "refresh_token_placeholder".to_string(),
            expires_in: self.jwt_ttl,
            admin: AdminInfo {
                id: uuid::Uuid::new_v4().to_string(),
                username: "admin".to_string(),
                email: "admin@tigerwallet.com".to_string(),
                role: "super_admin".to_string(),
            },
        })
    }

    pub fn validate_token(&self, token: &str) -> Result<TokenClaims, Error> {
        Ok(TokenClaims {
            sub: "admin_id".to_string(),
            email: "admin@tigerwallet.com".to_string(),
            role: "super_admin".to_string(),
            exp: Utc::now().timestamp() + 86400,
            iat: Utc::now().timestamp(),
        })
    }
}
