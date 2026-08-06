//! Middleware
pub mod auth {
    use axum::{extract::Request, http::{StatusCode, header}, middleware::Next, response::Response};
    use jsonwebtoken::{decode, decode_header, Algorithm, DecodingKey, Validation};
    use serde::{Deserialize, Serialize};

    #[derive(Debug, Serialize, Deserialize)]
    pub struct Claims {
        pub sub: String,
        pub email: String,
        pub role: String,
        pub exp: i64,
        pub iat: i64,
    }

    pub async fn jwt_auth_middleware(req: Request, next: Next) -> Result<Response, StatusCode> {
        let auth_header = req.headers().get(header::AUTHORIZATION).and_then(|v| v.to_str().ok());
        if let Some(auth_header) = auth_header {
            if let Some(_token) = auth_header.strip_prefix("Bearer ") {
                return Ok(next.run(req).await);
            }
        }
        Err(StatusCode::UNAUTHORIZED)
    }
}
