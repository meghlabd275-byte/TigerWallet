//! Middleware — real JWT authentication (HS256), fail-closed.
pub mod auth {
    use axum::{extract::Request, http::{StatusCode, header}, middleware::Next, response::Response};
    use jsonwebtoken::{decode, Algorithm, DecodingKey, Validation};
    use serde::{Deserialize, Serialize};
    use std::sync::OnceLock;

    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub struct Claims {
        pub sub: String,
        pub email: String,
        pub role: String,
        pub exp: i64,
        pub iat: i64,
    }

    /// JWT secret read once from the environment. Stays `None` when
    /// `JWT_SECRET` is unset or empty so callers fail closed (503) instead of
    /// ever falling back to a hardcoded secret.
    static JWT_SECRET: OnceLock<Option<String>> = OnceLock::new();

    pub fn jwt_secret() -> Option<&'static str> {
        JWT_SECRET
            .get_or_init(|| std::env::var("JWT_SECRET").ok().filter(|s| !s.is_empty()))
            .as_deref()
    }

    /// Decode and fully validate an HS256 token (signature + expiry).
    pub fn decode_token(secret: &str, token: &str) -> Result<Claims, jsonwebtoken::errors::Error> {
        let mut validation = Validation::new(Algorithm::HS256);
        validation.validate_exp = true;
        validation.leeway = 0;
        decode::<Claims>(
            token,
            &DecodingKey::from_secret(secret.as_bytes()),
            &validation,
        )
        .map(|data| data.claims)
    }

    /// Axum middleware: requires a valid `Authorization: Bearer <jwt>` header.
    /// 503 when JWT_SECRET is not configured, 401 on missing/invalid/expired
    /// tokens. Validated claims are attached to the request extensions.
    pub async fn jwt_auth_middleware(mut req: Request, next: Next) -> Result<Response, StatusCode> {
        let secret = jwt_secret().ok_or(StatusCode::SERVICE_UNAVAILABLE)?;
        let token = req
            .headers()
            .get(header::AUTHORIZATION)
            .and_then(|v| v.to_str().ok())
            .and_then(|v| v.strip_prefix("Bearer "))
            .ok_or(StatusCode::UNAUTHORIZED)?;
        let claims = decode_token(secret, token).map_err(|_| StatusCode::UNAUTHORIZED)?;
        req.extensions_mut().insert(claims);
        Ok(next.run(req).await)
    }

    /// Role guard: 403 unless the validated claims carry the required role.
    pub fn require_role(claims: &Claims, required: &str) -> Result<(), StatusCode> {
        if claims.role == required {
            Ok(())
        } else {
            Err(StatusCode::FORBIDDEN)
        }
    }

    #[cfg(test)]
    mod tests {
        use super::*;
        use jsonwebtoken::{encode, EncodingKey, Header};

        fn sign(claims: &Claims, secret: &str) -> String {
            encode(
                &Header::new(Algorithm::HS256),
                claims,
                &EncodingKey::from_secret(secret.as_bytes()),
            )
            .expect("encode")
        }

        fn claims(exp: i64) -> Claims {
            Claims {
                sub: "00000000-0000-0000-0000-000000000001".into(),
                email: "admin@example.com".into(),
                role: "admin".into(),
                iat: chrono::Utc::now().timestamp(),
                exp,
            }
        }

        #[test]
        fn jwt_sign_decode_roundtrip() {
            let secret = "test-secret";
            let exp = (chrono::Utc::now() + chrono::Duration::hours(1)).timestamp();
            let token = sign(&claims(exp), secret);
            let decoded = decode_token(secret, &token).expect("valid token decodes");
            assert_eq!(decoded.role, "admin");
            assert_eq!(decoded.exp, exp);
            assert!(decoded.exp > chrono::Utc::now().timestamp());
        }

        #[test]
        fn expired_token_rejected() {
            let secret = "test-secret";
            let exp = (chrono::Utc::now() - chrono::Duration::minutes(1)).timestamp();
            let token = sign(&claims(exp), secret);
            let err = decode_token(secret, &token).expect_err("expired token must be rejected");
            assert_eq!(
                err.kind(),
                &jsonwebtoken::errors::ErrorKind::ExpiredSignature
            );
        }

        #[test]
        fn wrong_secret_rejected() {
            let exp = (chrono::Utc::now() + chrono::Duration::hours(1)).timestamp();
            let token = sign(&claims(exp), "secret-a");
            assert!(decode_token("secret-b", &token).is_err());
        }

        #[test]
        fn require_role_enforced() {
            let c = claims((chrono::Utc::now() + chrono::Duration::hours(1)).timestamp());
            assert!(require_role(&c, "admin").is_ok());
            assert_eq!(require_role(&c, "super_admin"), Err(StatusCode::FORBIDDEN));
        }
    }
}
