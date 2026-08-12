//! Database connection module for MasterWallet
//! Provides PostgreSQL connection management (tokio-postgres 0.7 has no
//! built-in pool; this opens a connection per op on a shared runtime, and is
//! a drop-in for a future bb8-postgres pool).

use tokio::runtime::Runtime;
use tokio_postgres::{NoTls, Client};
use std::collections::HashMap;

/// PostgreSQL connection manager for MasterWallet.
pub struct MasterWalletDatabase {
    conn_str: String,
    runtime: Runtime,
}

impl MasterWalletDatabase {
    pub fn new(
        host: &str,
        port: u16,
        database: &str,
        username: &str,
        password: &str,
        _max_connections: u32,
    ) -> Result<Self, String> {
        let conn_str = format!(
            "host={} port={} dbname={} user={} password={} application_name=tigerwallet_master",
            host, port, database, username, password
        );
        let runtime = Runtime::new().map_err(|e| format!("Failed to create runtime: {}", e))?;
        Ok(Self { conn_str, runtime })
    }

    async fn connect(&self) -> Result<Client, String> {
        let (client, connection) = tokio_postgres::connect(&self.conn_str, NoTls)
            .await
            .map_err(|e| format!("Database connection failed: {}", e))?;
        tokio::spawn(async move {
            let _ = connection.await;
        });
        Ok(client)
    }

    /// Execute a query and return rows
    pub fn execute(&self, query: &str, params: &[&(dyn tokio_postgres::types::ToSql + Sync)])
        -> Result<Vec<tokio_postgres::Row>, String> {
        self.runtime.block_on(async {
            let mut client = self.connect().await?;
            client.query(query, params).await
                .map_err(|e| format!("Query failed: {}", e))
        })
    }

    /// Execute a query without returning rows
    pub fn execute_non_query(&self, query: &str, params: &[&(dyn tokio_postgres::types::ToSql + Sync)])
        -> Result<u64, String> {
        self.runtime.block_on(async {
            let mut client = self.connect().await?;
            client.execute(query, params).await
                .map_err(|e| format!("Execute failed: {}", e))
        })
    }

    /// Execute multiple statements in a single transaction. The closure
    /// receives the live `Transaction` so it can run statements atomically.
    pub fn execute_transaction<F>(&self, f: F) -> Result<(), String>
    where
        F: FnOnce(&tokio_postgres::Transaction<'_>) -> Result<(), String> + Send + 'static,
    {
        self.runtime.block_on(async {
            let mut client = self.connect().await?;
            let transaction = client.transaction().await
                .map_err(|e| format!("Failed to start transaction: {}", e))?;
            f(&transaction)?;
            transaction.commit().await
                .map_err(|e| format!("Transaction commit failed: {}", e))
        })
    }

    /// Get a raw client for ad-hoc async work on the runtime
    pub fn get_connection(&self) -> Result<Client, String> {
        self.runtime.block_on(async { self.connect().await })
    }
}

/// Query builder for type-safe queries
pub struct QueryBuilder {
    query: String,
    params: Vec<Box<dyn tokio_postgres::types::ToSql>>,
}

impl QueryBuilder {
    pub fn new() -> Self {
        Self {
            query: String::new(),
            params: Vec::new(),
        }
    }

    pub fn select(mut self, columns: &str, from: &str) -> Self {
        self.query = format!("SELECT {} FROM {}", columns, from);
        self
    }

    pub fn where_clause(mut self, condition: &str) -> Self {
        if self.query.contains("WHERE") {
            self.query.push_str(&format!(" AND {}", condition));
        } else {
            self.query.push_str(&format!(" WHERE {}", condition));
        }
        self
    }

    pub fn order_by(mut self, column: &str, direction: &str) -> Self {
        self.query.push_str(&format!(" ORDER BY {} {}", column, direction));
        self
    }

    pub fn limit(mut self, limit: u32) -> Self {
        self.query.push_str(&format!(" LIMIT {}", limit));
        self
    }

    pub fn offset(mut self, offset: u32) -> Self {
        self.query.push_str(&format!(" OFFSET {}", offset));
        self
    }

    pub fn build(self) -> (String, Vec<Box<dyn tokio_postgres::types::ToSql>>) {
        (self.query, self.params)
    }
}

impl Default for QueryBuilder {
    fn default() -> Self {
        Self::new()
    }
}

#[allow(unused_imports)]
use std::time::Duration;
#[allow(unused_imports)]
use std::sync::Arc;
#[allow(dead_code)]
fn _unused(_: HashMap<String, String>, _: Duration) {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_query_builder() {
        let builder = QueryBuilder::new()
            .select("id, name", "users")
            .where_clause("active = true")
            .order_by("created_at", "DESC")
            .limit(10);

        let (query, _) = builder.build();
        assert!(query.contains("SELECT"));
        assert!(query.contains("WHERE"));
        assert!(query.contains("ORDER BY"));
        assert!(query.contains("LIMIT"));
    }
}
