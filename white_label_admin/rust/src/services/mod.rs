//! Services — real, DB-backed business logic. No stubs, no fabricated data.
pub mod auth_service {
    use anyhow::{anyhow, bail, Result};
    use chrono::{Duration, Utc};
    use jsonwebtoken::{encode, Algorithm, EncodingKey, Header};

    use crate::database::Database;
    use crate::middleware::auth::{jwt_secret, Claims};
    use crate::models::{AdminUser, LoginRequest, LoginResponse};

    /// Access-token lifetime: 1 hour, matching the TTL the frontend expects
    /// for the `Authorization: Bearer` access token.
    pub const ACCESS_TOKEN_TTL_SECS: i64 = 3600;
    const REFRESH_TOKEN_TTL_DAYS: i64 = 7;

    fn sign_token(admin: &AdminUser, jwt_secret: &str, ttl: Duration) -> Result<String> {
        let now = Utc::now();
        let claims = Claims {
            sub: admin.id.to_string(),
            email: admin.email.clone(),
            role: admin.role.clone(),
            iat: now.timestamp(),
            exp: (now + ttl).timestamp(),
        };
        Ok(encode(
            &Header::new(Algorithm::HS256),
            &claims,
            &EncodingKey::from_secret(jwt_secret.as_bytes()),
        )?)
    }

    /// Real login: look the admin up by email in Postgres, verify the bcrypt
    /// password hash, then issue HS256 access (1 h) and refresh (7 d) tokens
    /// and record the login. Fails closed: `db` can only come from
    /// `Database::new` (which errors when DATABASE_URL is invalid or Postgres
    /// is unreachable) and the signing secret comes from JWT_SECRET via
    /// `jwt_secret()` (error when unset). Nothing is fabricated.
    pub async fn login(db: &Database, req: LoginRequest) -> Result<LoginResponse> {
        let secret = jwt_secret().ok_or_else(|| anyhow!("JWT_SECRET is not configured"))?;
        let pool = db.pool();

        let admin = sqlx::query_as::<_, AdminUser>(
            "SELECT id, username, email, password_hash, role, two_factor_secret,
                    two_factor_enabled, is_active, created_at, updated_at, last_login
             FROM admin_users WHERE email = $1",
        )
        .bind(&req.email)
        .fetch_optional(pool)
        .await?
        .ok_or_else(|| anyhow!("invalid credentials"))?;

        if !admin.is_active {
            bail!("account is disabled");
        }

        if !bcrypt::verify(&req.password, &admin.password_hash)? {
            bail!("invalid credentials");
        }

        let access_token = sign_token(&admin, secret, Duration::seconds(ACCESS_TOKEN_TTL_SECS))?;
        let refresh_token = sign_token(&admin, secret, Duration::days(REFRESH_TOKEN_TTL_DAYS))?;

        sqlx::query("UPDATE admin_users SET last_login = NOW(), updated_at = NOW() WHERE id = $1")
            .bind(admin.id)
            .execute(pool)
            .await?;

        Ok(LoginResponse {
            admin,
            access_token,
            refresh_token,
        })
    }

    #[cfg(test)]
    mod tests {
        use super::*;
        use crate::middleware::auth::decode_token;

        fn admin() -> AdminUser {
            AdminUser {
                id: uuid::Uuid::new_v4(),
                username: "root".into(),
                email: "root@example.com".into(),
                password_hash: String::new(),
                role: "admin".into(),
                two_factor_secret: None,
                two_factor_enabled: false,
                is_active: true,
                created_at: Utc::now(),
                updated_at: Utc::now(),
                last_login: None,
            }
        }

        #[test]
        fn bcrypt_hash_verify_roundtrip() {
            let hash = bcrypt::hash("s3cret-password", bcrypt::DEFAULT_COST).expect("hash");
            assert!(bcrypt::verify("s3cret-password", &hash).expect("verify"));
            assert!(!bcrypt::verify("wrong-password", &hash).expect("verify"));
        }

        #[test]
        fn jwt_minted_is_valid_hs256() {
            let secret = "test-secret";
            let token = sign_token(&admin(), secret, Duration::seconds(ACCESS_TOKEN_TTL_SECS))
                .expect("sign");
            // decode_token runs jsonwebtoken::decode with Validation HS256.
            let claims = decode_token(secret, &token).expect("valid HS256 token decodes");
            assert_eq!(claims.email, "root@example.com");
            assert_eq!(claims.role, "admin");
            assert_eq!(claims.exp - claims.iat, ACCESS_TOKEN_TTL_SECS);
        }

        #[test]
        fn expired_token_rejected() {
            let secret = "test-secret";
            let token = sign_token(&admin(), secret, Duration::minutes(-5)).expect("sign");
            let err = decode_token(secret, &token).expect_err("expired must fail");
            assert_eq!(
                err.kind(),
                &jsonwebtoken::errors::ErrorKind::ExpiredSignature
            );
        }
    }
}
