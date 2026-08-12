//! Admin fetchers — PostgreSQL database module.
//!
//! Real tokio-postgres integration. The pool owns its own tokio `Runtime` and
//! exposes SYNCHRONOUS wrappers (query / query_all / execute / health_check) so
//! fetcher code can be plain blocking call sites. No fake data, no stubs.

use std::sync::Arc;
use tokio::runtime::Runtime;
use tokio_postgres::{Client, NoTls, Row};
use std::time::Duration;

#[derive(Debug, Clone)]
pub struct DatabaseConfig {
    pub host: String,
    pub port: u16,
    pub database: String,
    pub username: String,
    pub password: String,
    pub max_connections: u32,
    pub connection_timeout: Duration,
    pub idle_timeout: Duration,
}

impl DatabaseConfig {
    pub fn new(host: &str, port: u16, database: &str, username: &str, password: &str) -> Self {
        Self {
            host: host.to_string(),
            port,
            database: database.to_string(),
            username: username.to_string(),
            password: password.to_string(),
            max_connections: 20,
            connection_timeout: Duration::from_secs(30),
            idle_timeout: Duration::from_secs(600),
        }
    }

    /// libpq-style connection string.
    pub fn connection_string(&self) -> String {
        format!(
            "host={} port={} dbname={} user={} password={} connect_timeout={}",
            self.host,
            self.port,
            self.database,
            self.username,
            self.password,
            self.connection_timeout.as_secs(),
        )
    }
}

pub struct DatabasePool {
    config: DatabaseConfig,
    runtime: Runtime,
}

impl DatabasePool {
    pub fn new(config: &DatabaseConfig) -> Result<Self, String> {
        Ok(Self {
            config: config.clone(),
            runtime: Runtime::new().map_err(|e| format!("Failed to create runtime: {}", e))?,
        })
    }

    async fn connect(&self) -> Result<Client, String> {
        let (client, connection) = tokio_postgres::connect(&self.config.connection_string(), NoTls)
            .await
            .map_err(|e| format!("Failed to connect to database: {}", e))?;
        tokio::spawn(async move {
            if let Err(e) = connection.await {
                eprintln!("Database connection error: {}", e);
            }
        });
        Ok(client)
    }

    /// Get a fresh connection (sync wrapper).
    pub fn get_connection(&self) -> Result<Client, String> {
        self.runtime.block_on(self.connect())
    }

    /// Execute a parameterized query and map each row via `mapper`.
    pub fn query<T, F>(&self, sql: &str, params: &[&(dyn tokio_postgres::types::ToSql + Sync)], mapper: F) -> Result<Vec<T>, String>
    where
        F: Fn(&Row) -> T,
    {
        self.runtime.block_on(async {
            let client = self.connect().await?;
            let rows = client.query(sql, params).await
                .map_err(|e| format!("Query failed: {}", e))?;
            Ok(rows.iter().map(mapper).collect::<Vec<T>>())
        })
    }

    /// Execute a parameterless query and map each row.
    pub fn query_all<T, F>(&self, sql: &str, mapper: F) -> Result<Vec<T>, String>
    where
        F: Fn(&Row) -> T,
    {
        self.runtime.block_on(async {
            let client = self.connect().await?;
            let rows = client.query(sql, &[]).await
                .map_err(|e| format!("Query failed: {}", e))?;
            Ok(rows.iter().map(mapper).collect::<Vec<T>>())
        })
    }

    /// Execute a parameterless query, returning the raw rows.
    pub fn query_rows(&self, sql: &str) -> Result<Vec<Row>, String> {
        self.runtime.block_on(async {
            let client = self.connect().await?;
            client.query(sql, &[]).await.map_err(|e| format!("Query failed: {}", e))
        })
    }

    /// Execute an INSERT/UPDATE/DELETE; returns affected row count.
    pub fn execute(&self, sql: &str, params: &[&(dyn tokio_postgres::types::ToSql + Sync)]) -> Result<u64, String> {
        self.runtime.block_on(async {
            let client = self.connect().await?;
            client.execute(sql, params).await.map_err(|e| format!("Execute failed: {}", e))
        })
    }

    /// Execute a statement with no parameters.
    pub fn execute_simple(&self, sql: &str) -> Result<u64, String> {
        self.runtime.block_on(async {
            let client = self.connect().await?;
            client.execute(sql, &[]).await.map_err(|e| format!("Execute failed: {}", e))
        })
    }

    /// Health check.
    pub fn health_check(&self) -> Result<bool, String> {
        self.runtime.block_on(async {
            let client = self.connect().await?;
            match client.query("SELECT 1", &[]).await {
                Ok(rows) => Ok(!rows.is_empty()),
                Err(e) => Err(format!("Health check failed: {}", e)),
            }
        })
    }
}

pub type DbPool = DatabasePool;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config() {
        let config = DatabaseConfig::new("localhost", 5432, "tigerwallet", "admin", "password");
        assert_eq!(config.host, "localhost");
        assert_eq!(config.port, 5432);
        assert_eq!(config.database, "tigerwallet");
        assert!(config.connection_string().contains("host=localhost"));
    }

    #[test]
    fn test_pool_creates_runtime() {
        let config = DatabaseConfig::new("localhost", 5432, "tigerwallet", "admin", "password");
        let pool = DatabasePool::new(&config).unwrap();
        assert!(pool.health_check().is_err()); // no DB running -> honest error, not a panic
    }
}
