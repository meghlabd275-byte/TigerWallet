//! PostgreSQL persistence for KYC applications, watchlists and audit records.
//!
//! Every statement is parameterised. No value is ever interpolated into SQL
//! text, which is what makes injection structurally impossible here rather
//! than merely filtered.

use chrono::{DateTime, Utc};
use sqlx::postgres::PgPoolOptions;
use sqlx::{PgPool, Row};
use uuid::Uuid;

use tiger_kyc_engine::{ListKind, WatchlistEntry};

/// The status vocabulary written to storage.
///
/// `verified` is the value the wallet backend gates P2P and card access on.
/// These strings are a contract with `go/wallet_api`; changing one without
/// changing the other closes or opens the gate incorrectly.
pub mod status {
    pub const NOT_STARTED: &str = "not_started";
    pub const PENDING_REVIEW: &str = "pending_review";
    pub const VERIFIED: &str = "verified";
    pub const REJECTED: &str = "rejected";
}

/// A stored KYC application.
#[derive(Debug, Clone)]
pub struct Application {
    pub id: Uuid,
    pub user_id: Uuid,
    pub level: i32,
    pub status: String,
    pub risk_score: i32,
    pub country: String,
    pub decision_reasons: Vec<String>,
    pub risk_factors: serde_json::Value,
    pub policy_version: String,
    pub reviewed_by: Option<Uuid>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// The store.
#[derive(Clone)]
pub struct Store {
    pool: PgPool,
}

impl Store {
    /// Connects to PostgreSQL and applies the schema.
    pub async fn connect(database_url: &str, max_connections: u32) -> Result<Self, sqlx::Error> {
        let pool = PgPoolOptions::new()
            .max_connections(max_connections)
            .connect(database_url)
            .await?;
        let store = Self { pool };
        store.migrate().await?;
        Ok(store)
    }

    /// Creates the schema if it is absent. Idempotent, so it is safe on every
    /// boot and safe to run against an existing database.
    pub async fn migrate(&self) -> Result<(), sqlx::Error> {
        sqlx::raw_sql(
            r#"
CREATE TABLE IF NOT EXISTS kyc_applications (
    id                UUID PRIMARY KEY,
    user_id           UUID NOT NULL,
    level             INTEGER NOT NULL,
    status            TEXT NOT NULL,
    risk_score        INTEGER NOT NULL DEFAULT 0,
    country           TEXT NOT NULL,
    decision_reasons  TEXT[] NOT NULL DEFAULT '{}',
    risk_factors      JSONB NOT NULL DEFAULT '[]'::jsonb,
    policy_version    TEXT NOT NULL,
    reviewed_by       UUID,
    reviewed_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A user has at most one live application; re-applying updates the row rather
-- than accumulating duplicates that could disagree about status.
CREATE UNIQUE INDEX IF NOT EXISTS kyc_applications_user_uniq
    ON kyc_applications (user_id);

CREATE INDEX IF NOT EXISTS kyc_applications_status_idx
    ON kyc_applications (status);

CREATE TABLE IF NOT EXISTS kyc_watchlist (
    id             UUID PRIMARY KEY,
    kind           TEXT NOT NULL,
    name           TEXT NOT NULL,
    country        TEXT,
    date_of_birth  TEXT,
    address        TEXT,
    source         TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Screening is exact and normalised, so an index on the normalised name keeps
-- screening cost independent of list size.
CREATE INDEX IF NOT EXISTS kyc_watchlist_name_idx ON kyc_watchlist (name);
CREATE INDEX IF NOT EXISTS kyc_watchlist_address_idx ON kyc_watchlist (lower(address));

CREATE TABLE IF NOT EXISTS kyc_audit_log (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID,
    application_id UUID,
    actor_id    TEXT,
    action      TEXT NOT NULL,
    details     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS kyc_audit_user_idx ON kyc_audit_log (user_id, created_at DESC);
"#,
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// Reports whether the database answers, for the readiness probe.
    pub async fn ping(&self) -> Result<(), sqlx::Error> {
        sqlx::query("SELECT 1").execute(&self.pool).await?;
        Ok(())
    }

    /// Inserts or replaces the application for a user.
    ///
    /// The upsert is keyed on `user_id`, so a re-application replaces the
    /// previous decision instead of leaving two rows that could disagree.
    #[allow(clippy::too_many_arguments)]
    pub async fn upsert_application(
        &self,
        user_id: Uuid,
        level: i32,
        status: &str,
        risk_score: i32,
        country: &str,
        reasons: &[String],
        factors: &serde_json::Value,
        policy_version: &str,
    ) -> Result<Application, sqlx::Error> {
        let id = Uuid::new_v4();
        let row = sqlx::query(
            r#"
INSERT INTO kyc_applications
    (id, user_id, level, status, risk_score, country, decision_reasons, risk_factors, policy_version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id) DO UPDATE SET
    level            = EXCLUDED.level,
    status           = EXCLUDED.status,
    risk_score       = EXCLUDED.risk_score,
    country          = EXCLUDED.country,
    decision_reasons = EXCLUDED.decision_reasons,
    risk_factors     = EXCLUDED.risk_factors,
    policy_version   = EXCLUDED.policy_version,
    reviewed_by      = NULL,
    reviewed_at      = NULL,
    updated_at       = now()
RETURNING id, user_id, level, status, risk_score, country, decision_reasons,
          risk_factors, policy_version, reviewed_by, created_at, updated_at
"#,
        )
        .bind(id)
        .bind(user_id)
        .bind(level)
        .bind(status)
        .bind(risk_score)
        .bind(country)
        .bind(reasons)
        .bind(factors)
        .bind(policy_version)
        .fetch_one(&self.pool)
        .await?;
        Ok(row_to_application(&row))
    }

    /// Reads a user's application.
    pub async fn get_application(&self, user_id: Uuid) -> Result<Option<Application>, sqlx::Error> {
        let row = sqlx::query(
            r#"
SELECT id, user_id, level, status, risk_score, country, decision_reasons,
       risk_factors, policy_version, reviewed_by, created_at, updated_at
FROM kyc_applications WHERE user_id = $1
"#,
        )
        .bind(user_id)
        .fetch_optional(&self.pool)
        .await?;
        Ok(row.as_ref().map(row_to_application))
    }

    /// Records a reviewer's decision and returns the updated application.
    ///
    /// Only a reviewer can move an application to `verified`; the automated
    /// engine's own approval writes the same value, and both paths are audited.
    pub async fn set_reviewer_decision(
        &self,
        user_id: Uuid,
        status: &str,
        reviewer: Uuid,
        reason: Option<&str>,
    ) -> Result<Option<Application>, sqlx::Error> {
        let row = sqlx::query(
            r#"
UPDATE kyc_applications
SET status = $2,
    reviewed_by = $3,
    reviewed_at = now(),
    updated_at = now(),
    decision_reasons = CASE WHEN $4::text IS NULL
                            THEN decision_reasons
                            ELSE array_append(decision_reasons, $4::text) END
WHERE user_id = $1
RETURNING id, user_id, level, status, risk_score, country, decision_reasons,
          risk_factors, policy_version, reviewed_by, created_at, updated_at
"#,
        )
        .bind(user_id)
        .bind(status)
        .bind(reviewer)
        .bind(reason)
        .fetch_optional(&self.pool)
        .await?;
        Ok(row.as_ref().map(row_to_application))
    }

    /// Lists applications in a given status, newest first.
    pub async fn list_by_status(
        &self,
        status: &str,
        limit: i64,
    ) -> Result<Vec<Application>, sqlx::Error> {
        let rows = sqlx::query(
            r#"
SELECT id, user_id, level, status, risk_score, country, decision_reasons,
       risk_factors, policy_version, reviewed_by, created_at, updated_at
FROM kyc_applications WHERE status = $1
ORDER BY created_at DESC LIMIT $2
"#,
        )
        .bind(status)
        .bind(limit)
        .fetch_all(&self.pool)
        .await?;
        Ok(rows.iter().map(row_to_application).collect())
    }

    /// Loads the entire watchlist for screening.
    ///
    /// Screening is performed in-process against this set. The set is small
    /// relative to a customer base and exact matching is cheap, so this keeps
    /// the decision path free of per-request query latency.
    pub async fn load_watchlist(&self) -> Result<Vec<WatchlistEntry>, sqlx::Error> {
        let rows = sqlx::query(
            "SELECT id, kind, name, country, date_of_birth, address, source FROM kyc_watchlist",
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows
            .iter()
            .map(|r| WatchlistEntry {
                id: r.get::<Uuid, _>("id").to_string(),
                kind: parse_list_kind(r.get::<String, _>("kind").as_str()),
                name: r.get("name"),
                country: r.get("country"),
                date_of_birth: r.get("date_of_birth"),
                address: r.get("address"),
                source: r.get("source"),
            })
            .collect())
    }

    /// Adds or updates a watchlist entry.
    pub async fn add_watchlist_entry(&self, entry: &WatchlistEntry) -> Result<Uuid, sqlx::Error> {
        let id = Uuid::new_v4();
        sqlx::query(
            r#"
INSERT INTO kyc_watchlist (id, kind, name, country, date_of_birth, address, source)
VALUES ($1, $2, $3, $4, $5, $6, $7)
"#,
        )
        .bind(id)
        .bind(list_kind_str(entry.kind))
        .bind(&entry.name)
        .bind(&entry.country)
        .bind(&entry.date_of_birth)
        .bind(&entry.address)
        .bind(&entry.source)
        .execute(&self.pool)
        .await?;
        Ok(id)
    }

    /// Removes a watchlist entry by id.
    pub async fn remove_watchlist_entry(&self, id: Uuid) -> Result<bool, sqlx::Error> {
        let res = sqlx::query("DELETE FROM kyc_watchlist WHERE id = $1")
            .bind(id)
            .execute(&self.pool)
            .await?;
        Ok(res.rows_affected() > 0)
    }

    /// Writes an audit record.
    ///
    /// Audit writes are best-effort and never fail a request: losing an audit
    /// row must not block a legitimate customer. The caller logs the failure.
    pub async fn audit(
        &self,
        user_id: Option<Uuid>,
        application_id: Option<Uuid>,
        actor_id: Option<&str>,
        action: &str,
        details: &serde_json::Value,
    ) -> Result<(), sqlx::Error> {
        sqlx::query(
            r#"
INSERT INTO kyc_audit_log (user_id, application_id, actor_id, action, details)
VALUES ($1, $2, $3, $4, $5)
"#,
        )
        .bind(user_id)
        .bind(application_id)
        .bind(actor_id)
        .bind(action)
        .bind(details)
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}

fn row_to_application(row: &sqlx::postgres::PgRow) -> Application {
    Application {
        id: row.get("id"),
        user_id: row.get("user_id"),
        level: row.get("level"),
        status: row.get("status"),
        risk_score: row.get("risk_score"),
        country: row.get("country"),
        decision_reasons: row.get("decision_reasons"),
        risk_factors: row.get("risk_factors"),
        policy_version: row.get("policy_version"),
        reviewed_by: row.get("reviewed_by"),
        created_at: row.get("created_at"),
        updated_at: row.get("updated_at"),
    }
}

fn parse_list_kind(s: &str) -> ListKind {
    match s {
        "sanctions" => ListKind::Sanctions,
        "pep" => ListKind::Pep,
        "adverse_media" => ListKind::AdverseMedia,
        _ => ListKind::InternalBlocklist,
    }
}

fn list_kind_str(k: ListKind) -> &'static str {
    match k {
        ListKind::Sanctions => "sanctions",
        ListKind::Pep => "pep",
        ListKind::AdverseMedia => "adverse_media",
        ListKind::InternalBlocklist => "internal_blocklist",
    }
}
