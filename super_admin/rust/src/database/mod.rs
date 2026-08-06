//! Database module
use sqlx::{postgres::PgPoolOptions, PgPool};
use std::time::Duration;
use anyhow::Result;

pub struct Database { pool: PgPool }

impl Database {
    pub async fn new(database_url: &str) -> Result<Self> {
        let pool = PgPoolOptions::new()
            .max_connections(25)
            .min_connections(5)
            .acquire_timeout(Duration::from_secs(30))
            .idle_timeout(Duration::from_secs(600))
            .connect(database_url).await?;
        Ok(Self { pool })
    }
    pub fn pool(&self) -> &PgPool { &self.pool }
}
