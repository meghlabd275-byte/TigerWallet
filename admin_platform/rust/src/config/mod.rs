use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub server: ServerConfig,
    pub database: DatabaseConfig,
    pub redis: RedisConfig,
    pub security: SecurityConfig,
    pub cors: CorsConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerConfig {
    pub host: String,
    pub port: u16,
    pub workers: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseConfig {
    pub host: String,
    pub port: u16,
    pub username: String,
    pub password: String,
    pub name: String,
    pub min_connections: u32,
    pub max_connections: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RedisConfig {
    pub host: String,
    pub port: u16,
    pub password: Option<String>,
    pub db: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityConfig {
    pub jwt_secret: String,
    pub jwt_ttl: u64,
    pub jwt_issuer: String,
    pub bcrypt_rounds: u32,
    pub require_2fa: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CorsConfig {
    pub allowed_origins: Vec<String>,
    pub allowed_methods: Vec<String>,
    pub allowed_headers: Vec<String>,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            server: ServerConfig {
                host: "0.0.0.0".to_string(),
                port: 9092,
                workers: 8,
            },
            database: DatabaseConfig {
                host: "localhost".to_string(),
                port: 5432,
                username: "tigeradmin".to_string(),
                password: "password".to_string(),
                name: "tigerwallet_admin".to_string(),
                min_connections: 5,
                max_connections: 50,
            },
            redis: RedisConfig {
                host: "localhost".to_string(),
                port: 6379,
                password: None,
                db: 0,
            },
            security: SecurityConfig {
                jwt_secret: "super-secret-jwt-key-change-in-production".to_string(),
                jwt_ttl: 86400,
                jwt_issuer: "tigerwallet".to_string(),
                bcrypt_rounds: 12,
                require_2fa: true,
            },
            cors: CorsConfig {
                allowed_origins: vec!["*".to_string()],
                allowed_methods: vec!["GET".to_string(), "POST".to_string(), "PUT".to_string(), "PATCH".to_string(), "DELETE".to_string(), "OPTIONS".to_string()],
                allowed_headers: vec!["Content-Type".to_string(), "Authorization".to_string()],
            },
        }
    }
}
