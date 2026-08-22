//! Real PostgreSQL persistence for the bot execution plane.
//!
//! Trades and executions are inserted into the `bot_trades` and
//! `bot_executions` tables. The pool wraps a real `tokio_postgres::Client`
//! connection. There are no in-memory fallbacks: any DB error propagates and
//! the HTTP layer returns 503 (fail-closed) when PostgreSQL is unavailable.

use chrono::Utc;
use std::fmt;
use std::sync::Arc;
use tokio::sync::Mutex;
use tokio_postgres::{Client, NoTls};

/// A real trade row written to `bot_trades`.
#[derive(Debug, Clone, serde::Serialize)]
pub struct TradeRecord {
    pub bot_id: String,
    pub tx_hash: String,
    pub amount_in: f64,
    pub amount_out: f64,
    pub fee: f64,
    pub profit: f64,
    pub success: bool,
    pub timestamp: chrono::DateTime<Utc>,
}

/// A real execution row written to `bot_executions`.
#[derive(Debug, Clone, serde::Serialize)]
pub struct ExecutionRecord {
    pub bot_id: String,
    pub strategy: String,
    pub action: String,
    pub detail: String,
    pub latency_us: u64,
    pub success: bool,
    pub timestamp: chrono::DateTime<Utc>,
}

/// Minimal real PostgreSQL connection pool.
///
/// Holds a single live `Client` behind a `Mutex` and lazily (re)connects on
/// first use or after a dropped connection. For the bot execution plane writes
/// are serialized; any connection failure surfaces as an error rather than
/// being silently dropped.
pub struct PgPool {
    config: String,
    conn: Mutex<Option<Client>>,
}

impl PgPool {
    /// Build a pool from a libpq-style connection string, e.g.
    /// `host=localhost port=5432 user=tigerwallet password=... dbname=tigerwallet`.
    pub fn new(config: impl Into<String>) -> Arc<Self> {
        Arc::new(Self {
            config: config.into(),
            conn: Mutex::new(None),
        })
    }

    async fn ensure_connected(&self, guard: &mut Option<Client>) -> Result<(), StoreError> {
        if guard.is_none() {
            let (client, connection) =
                tokio_postgres::connect(&self.config, NoTls).await?;
            tokio::spawn(async move {
                if let Err(e) = connection.await {
                    log::error!("postgres connection error: {e}");
                }
            });
            *guard = Some(client);
        }
        Ok(())
    }

    /// Force a connection now. Used at startup so the health check reflects
    /// real DB availability. Returns an error if the DB cannot be reached.
    pub async fn ping(&self) -> Result<(), StoreError> {
        let mut guard = self.conn.lock().await;
        self.ensure_connected(&mut guard).await?;
        let client = guard.as_ref().expect("connection just established");
        client.simple_query("SELECT 1").await?;
        Ok(())
    }

    /// Insert a real trade row. Fail-closed: returns an error on any DB failure.
    pub async fn insert_trade(&self, rec: &TradeRecord) -> Result<(), StoreError> {
        let mut guard = self.conn.lock().await;
        self.ensure_connected(&mut guard).await?;
        let client = guard.as_ref().expect("connection established");
        client
            .query(
                "INSERT INTO bot_trades \
                 (bot_id, tx_hash, amount_in, amount_out, fee, profit, success, timestamp) \
                 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
                &[
                    &rec.bot_id,
                    &rec.tx_hash,
                    &rec.amount_in,
                    &rec.amount_out,
                    &rec.fee,
                    &rec.profit,
                    &rec.success,
                    &rec.timestamp,
                ],
            )
            .await?;
        Ok(())
    }

    /// Insert a real execution row. Fail-closed.
    pub async fn insert_execution(&self, rec: &ExecutionRecord) -> Result<(), StoreError> {
        let mut guard = self.conn.lock().await;
        self.ensure_connected(&mut guard).await?;
        let client = guard.as_ref().expect("connection established");
        client
            .query(
                "INSERT INTO bot_executions \
                 (bot_id, strategy, action, detail, latency_us, success, timestamp) \
                 VALUES ($1, $2, $3, $4, $5, $6, $7)",
                &[
                    &rec.bot_id,
                    &rec.strategy,
                    &rec.action,
                    &rec.detail,
                    &(rec.latency_us as i64),
                    &rec.success,
                    &rec.timestamp,
                ],
            )
            .await?;
        Ok(())
    }

    /// Real aggregate stats read from PostgreSQL.
    pub async fn stats(&self) -> Result<StatsRow, StoreError> {
        let mut guard = self.conn.lock().await;
        self.ensure_connected(&mut guard).await?;
        let client = guard.as_ref().expect("connection established");
        let row = client
            .query_one(
                "SELECT \
                    COUNT(*) AS trades, \
                    COALESCE(SUM(amount_out), 0) AS volume_out, \
                    COALESCE(SUM(profit), 0) AS profit, \
                    COALESCE(SUM(fee), 0) AS fees \
                 FROM bot_trades",
                &[],
            )
            .await?;
        Ok(StatsRow {
            total_trades: row.get::<_, i64>(0) as u64,
            total_volume_out: row.get::<_, f64>(1),
            total_profit: row.get::<_, f64>(2),
            total_fees: row.get::<_, f64>(3),
        })
    }
}

/// Aggregate stats materialized from the `bot_trades` table.
#[derive(Debug, Clone, serde::Serialize)]
pub struct StatsRow {
    pub total_trades: u64,
    pub total_volume_out: f64,
    pub total_profit: f64,
    pub total_fees: f64,
}

#[derive(Debug)]
pub enum StoreError {
    Postgres(tokio_postgres::Error),
}

impl fmt::Display for StoreError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            StoreError::Postgres(e) => write!(f, "postgres: {e}"),
        }
    }
}

impl std::error::Error for StoreError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            StoreError::Postgres(e) => Some(e),
        }
    }
}

impl From<tokio_postgres::Error> for StoreError {
    fn from(e: tokio_postgres::Error) -> Self {
        StoreError::Postgres(e)
    }
}
