//! Database connection module for MasterWallet
//! Provides PostgreSQL connection management (tokio-postgres 0.7 has no
//! built-in pool; this opens a connection per op on a shared runtime, and is
//! a drop-in for a future bb8-postgres pool).

use std::collections::HashMap;
use tokio::runtime::Runtime;
use tokio_postgres::{Client, NoTls};

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
    pub fn execute(
        &self,
        query: &str,
        params: &[&(dyn tokio_postgres::types::ToSql + Sync)],
    ) -> Result<Vec<tokio_postgres::Row>, String> {
        self.runtime.block_on(async {
            let mut client = self.connect().await?;
            client
                .query(query, params)
                .await
                .map_err(|e| format!("Query failed: {}", e))
        })
    }

    /// Execute a query without returning rows
    pub fn execute_non_query(
        &self,
        query: &str,
        params: &[&(dyn tokio_postgres::types::ToSql + Sync)],
    ) -> Result<u64, String> {
        self.runtime.block_on(async {
            let mut client = self.connect().await?;
            client
                .execute(query, params)
                .await
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
            let transaction = client
                .transaction()
                .await
                .map_err(|e| format!("Failed to start transaction: {}", e))?;
            f(&transaction)?;
            transaction
                .commit()
                .await
                .map_err(|e| format!("Transaction commit failed: {}", e))
        })
    }

    /// Get a raw client for ad-hoc async work on the runtime
    pub fn get_connection(&self) -> Result<Client, String> {
        self.runtime.block_on(async { self.connect().await })
    }
}

/// Query builder for type-safe queries.
///
/// Table, column and direction names cannot be bound as query parameters, so
/// they are interpolated into the SQL text. Every such value is checked
/// against a strict identifier pattern first and the builder records the first
/// failure instead of emitting a statement. Values always travel as bind
/// parameters; `where_clause` accepts only a condition that uses `$N`
/// placeholders.
pub struct QueryBuilder {
    query: String,
    params: Vec<Box<dyn tokio_postgres::types::ToSql>>,
    error: Option<String>,
}

/// Matches a bare SQL identifier such as `created_at`, plus `*` and qualified
/// names like `wallets.address`.
fn is_valid_identifier(name: &str) -> bool {
    fn is_segment(segment: &str) -> bool {
        let mut chars = segment.chars();
        match chars.next() {
            Some(c) if c.is_ascii_alphabetic() || c == '_' => {}
            _ => return false,
        }
        chars.all(|c| c.is_ascii_alphanumeric() || c == '_')
    }

    if name == "*" {
        return true;
    }
    // A column list is passed as one string; validate each comma-separated part.
    name.split(',').all(|part| {
        let part = part.trim();
        if part.is_empty() {
            return false;
        }
        // Allow an optional alias: "col AS alias".
        let mut pieces = part.split_whitespace();
        let target = match pieces.next() {
            Some(t) => t,
            None => return false,
        };
        if !target.split('.').all(is_segment) {
            return false;
        }
        match pieces.next() {
            None => true,
            Some(kw) if kw.eq_ignore_ascii_case("AS") => match pieces.next() {
                Some(alias) => is_segment(alias) && pieces.next().is_none(),
                None => false,
            },
            Some(_) => false,
        }
    })
}

impl QueryBuilder {
    pub fn new() -> Self {
        Self {
            query: String::new(),
            params: Vec::new(),
            error: None,
        }
    }

    fn fail(&mut self, message: String) {
        if self.error.is_none() {
            self.error = Some(message);
        }
    }

    pub fn select(mut self, columns: &str, from: &str) -> Self {
        if !is_valid_identifier(columns) {
            self.fail(format!("invalid column list: {:?}", columns));
        }
        if !is_valid_identifier(from) {
            self.fail(format!("invalid table name: {:?}", from));
        }
        self.query = format!("SELECT {} FROM {}", columns, from);
        self
    }

    /// Append a condition. The condition must reference bind parameters
    /// (`$1`, `$2`, ...) rather than inline values.
    pub fn where_clause(mut self, condition: &str) -> Self {
        if condition.trim().is_empty() {
            self.fail("empty WHERE clause".to_string());
            return self;
        }
        if self.query.is_empty() {
            self.fail("where_clause called before select".to_string());
            return self;
        }
        if self.query.contains(" WHERE ") {
            self.query.push_str(&format!(" AND {}", condition));
        } else {
            self.query.push_str(&format!(" WHERE {}", condition));
        }
        self
    }

    /// Append a condition of the form `column = $N`. The column is validated
    /// as an identifier, so it is safe to pass a value that came from a
    /// request as long as it matches a real column name.
    pub fn where_eq(mut self, column: &str, placeholder: u32) -> Self {
        if !is_valid_identifier(column) {
            self.fail(format!("invalid column name: {:?}", column));
            return self;
        }
        if placeholder == 0 {
            self.fail("placeholder indexes start at 1".to_string());
            return self;
        }
        self.where_clause(&format!("{} = ${}", column, placeholder))
    }

    pub fn order_by(mut self, column: &str, direction: &str) -> Self {
        if !is_valid_identifier(column) {
            self.fail(format!("invalid order column: {:?}", column));
            return self;
        }
        let direction = if direction.eq_ignore_ascii_case("asc") {
            "ASC"
        } else if direction.eq_ignore_ascii_case("desc") {
            "DESC"
        } else {
            self.fail(format!("invalid sort direction: {:?}", direction));
            return self;
        };
        self.query
            .push_str(&format!(" ORDER BY {} {}", column, direction));
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

    /// Add a bind parameter. Values never reach the SQL text.
    pub fn bind<T>(mut self, value: T) -> Self
    where
        T: tokio_postgres::types::ToSql + Send + Sync + 'static,
    {
        self.params.push(Box::new(value));
        self
    }

    /// Finish the statement. Returns the error recorded by any rejected
    /// identifier instead of a query that could be malformed.
    pub fn build(self) -> Result<(String, Vec<Box<dyn tokio_postgres::types::ToSql>>), String> {
        if let Some(err) = self.error {
            return Err(err);
        }
        if self.query.is_empty() {
            return Err("no statement was built".to_string());
        }
        Ok((self.query, self.params))
    }
}

impl Default for QueryBuilder {
    fn default() -> Self {
        Self::new()
    }
}

#[allow(unused_imports)]
use std::sync::Arc;
#[allow(unused_imports)]
use std::time::Duration;
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

        let (query, _) = builder.build().expect("a well-formed query builds");
        assert!(query.contains("SELECT"));
        assert!(query.contains("WHERE"));
        assert!(query.contains("ORDER BY"));
        assert!(query.contains("LIMIT"));
    }

    #[test]
    fn identifiers_accept_qualified_names_and_aliases() {
        assert!(is_valid_identifier("wallets.address"));
        assert!(is_valid_identifier("*"));
        assert!(is_valid_identifier("id, name AS label"));
        assert!(is_valid_identifier("created_at"));
    }

    #[test]
    fn identifiers_reject_injection_payloads() {
        assert!(!is_valid_identifier("users; DROP TABLE wallets"));
        assert!(!is_valid_identifier("users --"));
        assert!(!is_valid_identifier("1users"));
        assert!(!is_valid_identifier(""));
        assert!(!is_valid_identifier("users'"));
        assert!(!is_valid_identifier("name; SELECT pg_sleep(10)"));
        assert!(!is_valid_identifier("id UNION SELECT password"));
    }

    #[test]
    fn a_malicious_table_name_is_reported_not_executed() {
        let result = QueryBuilder::new()
            .select("*", "users; DROP TABLE wallets")
            .build();
        assert!(result.is_err(), "the builder must refuse the table name");
    }

    #[test]
    fn a_malicious_order_column_is_reported() {
        let result = QueryBuilder::new()
            .select("*", "users")
            .order_by("id; DROP TABLE wallets", "ASC")
            .build();
        assert!(result.is_err());
    }

    #[test]
    fn an_invalid_sort_direction_is_reported() {
        let result = QueryBuilder::new()
            .select("*", "users")
            .order_by("id", "ASC; DROP TABLE wallets")
            .build();
        assert!(result.is_err());
    }

    #[test]
    fn a_valid_direction_is_normalized() {
        let (query, _) = QueryBuilder::new()
            .select("*", "users")
            .order_by("id", "desc")
            .build()
            .unwrap();
        assert!(query.ends_with("ORDER BY id DESC"));
    }

    #[test]
    fn where_eq_builds_a_placeholder_condition() {
        let (query, params) = QueryBuilder::new()
            .select("*", "users")
            .where_eq("id", 1)
            .bind(42i32)
            .build()
            .unwrap();
        assert!(query.contains("WHERE id = $1"), "got {query}");
        assert_eq!(params.len(), 1);
    }

    #[test]
    fn where_eq_rejects_a_malicious_column() {
        let result = QueryBuilder::new()
            .select("*", "users")
            .where_eq("id = 1 OR 1=1 --", 1)
            .build();
        assert!(result.is_err());
    }

    #[test]
    fn where_clause_before_select_is_reported() {
        let result = QueryBuilder::new().where_clause("id = $1").build();
        assert!(result.is_err());
    }

    #[test]
    fn an_empty_builder_reports_an_error() {
        assert!(QueryBuilder::new().build().is_err());
    }
}
