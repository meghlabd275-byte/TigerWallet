//! Authentication module

use jsonwebtoken::{encode, decode, Header, Validation, EncodingKey, DecodingKey};
use serde::{Deserialize, Serialize};
use chrono::{Utc, Duration};
use uuid::Uuid;

#[derive(Debug, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String,  // admin_id
    pub email: String,
    pub role: String,
    pub exp: i64,
    pub iat: i64,
}

#[derive(Clone)]
pub struct AuthState {
    pub jwt_secret: String,
}

impl AuthState {
    pub fn new() -> Self {
        // Fail closed: a missing JWT_SECRET must never fall back to a
        // hardcoded secret, or every deployment without configuration would
        // accept tokens signed with a publicly known key.
        let jwt_secret = std::env::var("JWT_SECRET")
            .expect("JWT_SECRET environment variable is required");
        
        Self { jwt_secret }
    }

    pub fn generate_token(&self, admin_id: Uuid, email: &str, role: &str) -> Result<String, jsonwebtoken::errors::Error> {
        let now = Utc::now();
        let claims = Claims {
            sub: admin_id.to_string(),
            email: email.to_string(),
            role: role.to_string(),
            exp: (now + Duration::hours(24)).timestamp(),
            iat: now.timestamp(),
        };

        encode(
            &Header::default(),
            &claims,
            &EncodingKey::from_secret(self.jwt_secret.as_bytes()),
        )
    }

    pub fn generate_refresh_token(&self, admin_id: Uuid) -> Result<String, jsonwebtoken::errors::Error> {
        let now = Utc::now();
        let claims = Claims {
            sub: admin_id.to_string(),
            email: "refresh".to_string(),
            role: "refresh".to_string(),
            exp: (now + Duration::days(7)).timestamp(),
            iat: now.timestamp(),
        };

        encode(
            &Header::default(),
            &claims,
            &EncodingKey::from_secret(self.jwt_secret.as_bytes()),
        )
    }

    pub fn validate_token(&self, token: &str) -> Result<Claims, jsonwebtoken::errors::Error> {
        decode::<Claims>(
            token,
            &DecodingKey::from_secret(self.jwt_secret.as_bytes()),
            &Validation::default(),
        ).map(|data| data.claims)
    }
}

impl Default for AuthState {
    fn default() -> Self {
        Self::new()
    }
}

// Password hashing
pub fn hash_password(password: &str) -> Result<String, bcrypt::BcryptError> {
    bcrypt::hash(password, 10)
}

pub fn verify_password(password: &str, hash: &str) -> Result<bool, bcrypt::BcryptError> {
    bcrypt::verify(password, hash)
}
