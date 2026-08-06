//! Services
pub mod auth_service {
    use anyhow::Result;
    use uuid::Uuid;
    use crate::models::{AdminUser, LoginRequest, LoginResponse};

    pub async fn login(_req: LoginRequest) -> Result<LoginResponse> {
        Ok(LoginResponse {
            admin: AdminUser { id: Uuid::new_v4(), username: _req.email.clone(), email: _req.email, password_hash: "hashed".to_string(), role: "admin".to_string(), two_factor_secret: None, two_factor_enabled: false, is_active: true, created_at: chrono::Utc::now(), updated_at: chrono::Utc::now(), last_login: None },
            access_token: "token".to_string(),
            refresh_token: "refresh".to_string(),
        })
    }
}
