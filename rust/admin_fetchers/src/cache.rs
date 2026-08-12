//! Admin fetchers — Redis cache module.
//!
//! Real Redis integration via a single owned `redis::Connection` guarded by a
//! `Mutex`. Exposes SYNCHRONOUS wrappers so fetcher code can be plain blocking
//! call sites. No fake data, no stubs.

use std::sync::Mutex;
use std::time::Duration;

use redis::{Client, Commands, RedisError};

#[derive(Debug, Clone)]
pub struct RedisConfig {
    pub host: String,
    pub port: u16,
    pub password: Option<String>,
    pub database: u8,
}

impl RedisConfig {
    pub fn new(host: &str, port: u16) -> Self {
        Self {
            host: host.to_string(),
            port,
            password: None,
            database: 0,
        }
    }

    pub fn with_password(host: &str, port: u16, password: &str) -> Self {
        Self {
            host: host.to_string(),
            port,
            password: Some(password.to_string()),
            database: 0,
        }
    }

    pub fn connection_string(&self) -> String {
        match &self.password {
            Some(password) => format!("redis://:{}@{}:{}/{}", password, self.host, self.port, self.database),
            None => format!("redis://{}:{}/{}", self.host, self.port, self.database),
        }
    }
}

pub struct CacheManager {
    /// Lazily-opened connection (created on first use, then reused).
    connection: Mutex<Option<redis::Connection>>,
    config: RedisConfig,
}

impl CacheManager {
    pub fn new(config: &RedisConfig) -> Result<Self, String> {
        // Eagerly validate the Redis URL by constructing (not opening) a client.
        let _ = Client::open(config.connection_string())
            .map_err(|e| format!("Failed to create Redis client: {}", e))?;
        Ok(Self {
            connection: Mutex::new(None),
            config: config.clone(),
        })
    }

    fn with_conn<F, T>(&self, f: F) -> Result<T, RedisError>
    where
        F: FnOnce(&mut redis::Connection) -> Result<T, RedisError>,
    {
        let mut guard = self.connection.lock().map_err(|_| {
            redis::RedisError::from(std::io::Error::new(std::io::ErrorKind::Other, "cache lock poisoned"))
        })?;
        if guard.is_none() {
            let client = Client::open(self.config.connection_string())
                .map_err(|e| redis::RedisError::from(std::io::Error::new(std::io::ErrorKind::Other, e.to_string())))?;
            let con = client.get_connection()?;
            *guard = Some(con);
        }
        let con = guard.as_mut().expect("connection initialized above");
        f(con)
    }

    pub fn get(&self, key: &str) -> Result<Option<String>, String> {
        self.with_conn(|c| {
            let val: Option<String> = c.get(key)?;
            Ok(val)
        }).map_err(|e| format!("Redis get failed: {}", e))
    }

    pub fn set(&self, key: &str, value: &str, expiration: Option<Duration>) -> Result<(), String> {
        self.with_conn(|c| {
            match expiration {
                Some(exp) => {
                    let _: () = c.set_ex(key, value, exp.as_secs() as usize)?;
                }
                None => {
                    let _: () = c.set(key, value)?;
                }
            }
            Ok(())
        }).map_err(|e| format!("Redis set failed: {}", e))
    }

    pub fn del(&self, key: &str) -> Result<u64, String> {
        self.with_conn(|c| {
            let n: u64 = c.del(key)?;
            Ok(n)
        }).map_err(|e| format!("Redis del failed: {}", e))
    }

    pub fn exists(&self, key: &str) -> Result<bool, String> {
        self.with_conn(|c| {
            let v: bool = c.exists(key)?;
            Ok(v)
        }).map_err(|e| format!("Redis exists failed: {}", e))
    }

    pub fn ping(&self) -> Result<bool, String> {
        self.with_conn(|c| {
            let _: String = redis::cmd("PING").query(c)?;
            Ok(true)
        }).map_err(|e| format!("Redis ping failed: {}", e))
    }

    pub fn flushall(&self) -> Result<(), String> {
        self.with_conn(|c| {
            let _: () = redis::cmd("FLUSHALL").query(c)?;
            Ok(())
        }).map_err(|e| format!("Redis flushall failed: {}", e))
    }

    pub fn health_check(&self) -> Result<bool, String> {
        self.ping()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config() {
        let config = RedisConfig::new("localhost", 6379);
        assert_eq!(config.host, "localhost");
        assert_eq!(config.port, 6379);
        assert_eq!(config.connection_string(), "redis://localhost:6379/0");
    }

    #[test]
    fn test_with_password_config() {
        let config = RedisConfig::with_password("localhost", 6379, "secret");
        assert_eq!(config.connection_string(), "redis://:secret@localhost:6379/0");
    }
}
