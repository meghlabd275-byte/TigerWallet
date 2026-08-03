use sqlx::{PgPool, Postgres, Pool};
use std::sync::Arc;
use tokio::sync::RwLock;

pub struct Database {
    pool: PgPool,
}

impl Database {
    pub async fn new(connection_string: &str) -> Result<Self, sqlx::Error> {
        let pool = PgPool::connect(connection_string).await?;
        Ok(Self { pool })
    }

    pub fn pool(&self) -> &PgPool {
        &self.pool
    }

    pub async fn health_check(&self) -> Result<(), sqlx::Error> {
        sqlx::query("SELECT 1")
            .fetch_one(self.pool())
            .await?;
        Ok(())
    }
}

impl Clone for Database {
    fn clone(&self) -> Self {
        Self {
            pool: self.pool.clone(),
        }
    }
}

pub type DbPool = Arc<Database>;
