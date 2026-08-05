/**
 * TigerWallet Admin Fetchers - PostgreSQL Database Module
 * High-performance database operations with connection pooling
 * 
 * Features:
 * - Real PostgreSQL integration
 * - Connection pooling (sqlx)
 * - Async operations (tokio)
 * - Prepared statements for security
 * - Transaction support
 */

use std::sync::Arc;
use tokio::sync::RwLock;
use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use tokio_postgres::{NoTls, Client, Row, types::Type};
use std::time::Duration;

/// Database configuration
#[derive(Debug, Clone)]
pub struct DatabaseConfig {
    pub host: String,
    pub port: u16,
    pub database: String,
    pub username: String,
    pub password: String,
    pub max_connections: u32,
    pub min_connections: u32,
    pub connection_timeout: Duration,
    pub idle_timeout: Duration,
}

impl DatabaseConfig {
    pub fn new(
        host: &str,
        port: u16,
        database: &str,
        username: &str,
        password: &str,
    ) -> Self {
        Self {
            host: host.to_string(),
            port,
            database: database.to_string(),
            username: username.to_string(),
            password: password.to_string(),
            max_connections: 20,
            min_connections: 5,
            connection_timeout: Duration::from_secs(30),
            idle_timeout: Duration::from_secs(600),
        }
    }

    pub fn connection_string(&self) -> String {
        format!(
            "host={} port={} dbname={} user={} password={} \
             pool_max_connections={} pool_min_connections={} \
             connect_timeout={} idle_timeout={}",
            self.host,
            self.port,
            self.database,
            self.username,
            self.password,
            self.max_connections,
            self.min_connections,
            self.connection_timeout.as_secs(),
            self.idle_timeout.as_secs()
        )
    }
}

/// Connection pool manager
pub struct DatabasePool {
    config: DatabaseConfig,
    clients: Arc<RwLock<Vec<Client>>>,
    active_count: Arc<RwLock<usize>>,
    max_connections: usize,
}

impl DatabasePool {
    pub fn new(config: &DatabaseConfig) -> Result<Self, String> {
        Ok(Self {
            config: config.clone(),
            clients: Arc::new(RwLock::new(Vec::new())),
            active_count: Arc::new(RwLock::new(0)),
            max_connections: config.max_connections as usize,
        })
    }

    /// Get a database connection from the pool
    pub async fn get_connection(&self) -> Result<Client, String> {
        let mut clients = self.clients.write().await;
        
        // Try to get an existing connection
        if let Some(client) = clients.pop() {
            let mut count = self.active_count.write().await;
            *count += 1;
            return Ok(client);
        }
        
        // Create new connection if pool not full
        let count = *self.active_count.read().await;
        if count < self.max_connections {
            let connection_string = self.config.connection_string();
            let (client, connection) = tokio_postgres::connect(&connection_string, NoTls)
                .await
                .map_err(|e| format!("Failed to connect to database: {}", e))?;
            
            // Spawn connection handler
            tokio::spawn(async move {
                if let Err(e) = connection.await {
                    eprintln!("Database connection error: {}", e);
                }
            });
            
            let mut active = self.active_count.write().await;
            *active += 1;
            
            Ok(client)
        } else {
            Err("Connection pool exhausted".to_string())
        }
    }

    /// Return a connection to the pool
    pub async fn return_connection(&self, client: Client) {
        let mut clients = self.clients.write().await;
        let mut count = self.active_count.write().await;
        
        if *count > 0 {
            *count -= 1;
            clients.push(client);
        }
    }

    /// Execute a query with parameters
    pub async fn query<T, F>(&self, sql: &str, params: &[&dyn tokio_postgres::types::ToSql], mapper: F) -> Result<Vec<T>, String>
    where
        F: Fn(Row) -> Result<T, tokio_postgres::Error>,
    {
        let client = self.get_connection().await?;
        let rows = client.query(sql, params).await
            .map_err(|e| format!("Query failed: {}", e))?;
        
        let mut results = Vec::new();
        for row in rows {
            results.push(mapper(row).map_err(|e| format!("Row mapping failed: {}", e))?);
        }
        
        self.return_connection(client).await;
        Ok(results)
    }

    /// Execute a query without parameters
    pub async fn query_all<T, F>(&self, sql: &str, mapper: F) -> Result<Vec<T>, String>
    where
        F: Fn(Row) -> Result<T, tokio_postgres::Error>,
    {
        let client = self.get_connection().await?;
        let rows = client.query(sql, &[]).await
            .map_err(|e| format!("Query failed: {}", e))?;
        
        let mut results = Vec::new();
        for row in rows {
            results.push(mapper(row).map_err(|e| format!("Row mapping failed: {}", e))?);
        }
        
        self.return_connection(client).await;
        Ok(results)
    }

    /// Execute a statement (INSERT, UPDATE, DELETE)
    pub async fn execute(&self, sql: &str, params: &[&dyn tokio_postgres::types::ToSql]) -> Result<u64, String> {
        let client = self.get_connection().await?;
        let affected = client.execute(sql, params).await
            .map_err(|e| format!("Execute failed: {}", e))?;
        
        self.return_connection(client).await;
        Ok(affected)
    }

    /// Execute a transaction
    pub async fn transaction<F, R>(&self, f: F) -> Result<R, String>
    where
        F: Fn(&Client) -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<R, String>> + Send>>,
    {
        let client = self.get_connection().await?;
        let transaction = client.transaction().await
            .map_err(|e| format!("Transaction failed: {}", e))?;
        
        let result = f(&transaction).await;
        
        if result.is_ok() {
            transaction.commit().await
                .map_err(|e| format!("Commit failed: {}", e))?;
        }
        
        self.return_connection(client).await;
        result
    }

    /// Health check
    pub async fn health_check(&self) -> Result<bool, String> {
        let client = self.get_connection().await?;
        let result = client.query("SELECT 1", &[]).await;
        self.return_connection(client).await;
        
        match result {
            Ok(rows) => Ok(!rows.is_empty()),
            Err(_) => Ok(false),
        }
    }
}

// Database pool alias for easier use
pub type DbPool = DatabasePool;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config() {
        let config = DatabaseConfig::new(
            "localhost",
            5432,
            "tigerwallet",
            "admin",
            "password",
        );
        
        assert_eq!(config.host, "localhost");
        assert_eq!(config.port, 5432);
        assert_eq!(config.database, "tigerwallet");
    }
}
