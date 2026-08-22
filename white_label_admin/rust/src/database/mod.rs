//! Database module — PgPool creation, shared AppState, and idempotent migrations
//! mirroring the Go backend white_label_admin/go schema (CREATE TABLE IF NOT EXISTS).
use anyhow::Result;
use sqlx::{postgres::PgPoolOptions, PgPool};
use std::time::Duration;
use uuid::Uuid;

/// Shared application state threaded through axum handlers via `State<AppState>`.
/// Holds the Postgres connection pool plus the JWT signing secret and the
/// default tenant id used for WL-scoped governance queries (the Rust backend
/// runs as a single-tenant alternative admin in the absence of the Go TenantID
/// middleware; the tenant id comes from WL_CLIENT_ID so the same DB can be shared).
#[derive(Clone)]
pub struct AppState {
    pub pool: PgPool,
    pub jwt_secret: String,
    pub white_label_id: Uuid,
}

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

/// Read env var or return default.
pub fn env_or(key: &str, default: &str) -> String {
    std::env::var(key).unwrap_or_else(|_| default.to_string())
}

/// Build the AppState: a PgPool from DATABASE_URL, a JWT secret, and the
/// default tenant id. Mirrors the Go config defaults.
pub async fn build_state() -> Result<AppState> {
    let database_url = env_or(
        "DATABASE_URL",
        "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable",
    );
    let pool = PgPoolOptions::new()
        .max_connections(25)
        .min_connections(5)
        .acquire_timeout(Duration::from_secs(30))
        .idle_timeout(Duration::from_secs(600))
        .connect(&database_url)
        .await?;

    let jwt_secret = env_or("JWT_SECRET", "tigerwallet-dev-jwt-secret-change-me");
    let wl = env_or("WL_CLIENT_ID", "00000000-0000-0000-0000-000000000001");
    let white_label_id = Uuid::parse_str(&wl).unwrap_or_else(|_| {
        Uuid::parse_str("00000000-0000-0000-0000-000000000001").expect("valid default uuid")
    });

    Ok(AppState { pool, jwt_secret, white_label_id })
}

/// Run idempotent schema migrations. Every statement uses CREATE TABLE IF NOT
/// EXISTS / ALTER ... ADD COLUMN IF NOT EXISTS / CREATE INDEX IF NOT EXISTS so
/// the Rust backend can share the Go backend's existing database. Column types
/// mirror the Go schema (postgres.go) so the two backends are interchangeable.
pub async fn run_migrations(pool: &PgPool) -> Result<()> {
    // We batch the DDL into one execute per statement for clarity; each is
    // individually idempotent.
    let stmts: &[&str] = &[
        // --- core admin / auth ---
        r#"CREATE TABLE IF NOT EXISTS admin_users (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID,
            username TEXT NOT NULL UNIQUE,
            email TEXT NOT NULL UNIQUE,
            password_hash TEXT NOT NULL,
            role TEXT NOT NULL DEFAULT 'admin',
            two_factor_secret TEXT,
            two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
            is_active BOOLEAN NOT NULL DEFAULT TRUE,
            last_login TIMESTAMPTZ,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS admin_sessions (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            admin_id UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
            ip_address TEXT,
            user_agent TEXT,
            expires_at TIMESTAMPTZ NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS white_labels (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name TEXT NOT NULL,
            domain TEXT NOT NULL,
            is_active BOOLEAN NOT NULL DEFAULT TRUE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        // --- tenant-scoped user/wallet domain ---
        r#"CREATE TABLE IF NOT EXISTS users (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            email TEXT NOT NULL,
            username TEXT NOT NULL,
            wallet_address TEXT,
            kyc_status TEXT NOT NULL DEFAULT 'none',
            status TEXT NOT NULL DEFAULT 'active',
            two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
            ip_address TEXT,
            country TEXT,
            last_login TIMESTAMPTZ,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS transactions (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            type TEXT NOT NULL,
            amount NUMERIC(40,8) NOT NULL DEFAULT 0,
            currency TEXT NOT NULL,
            status TEXT NOT NULL,
            from_address TEXT,
            to_address TEXT,
            tx_hash TEXT,
            fee NUMERIC(40,8) NOT NULL DEFAULT 0,
            chain_id BIGINT NOT NULL DEFAULT 0,
            timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS withdrawals (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            amount NUMERIC(40,8) NOT NULL DEFAULT 0,
            currency TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'pending',
            address TEXT NOT NULL,
            tx_hash TEXT,
            approved_by UUID,
            processed_at TIMESTAMPTZ,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS kyc_requests (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            doc_type TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'pending',
            document_url TEXT,
            reject_reason TEXT,
            submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            reviewed_at TIMESTAMPTZ,
            reviewed_by UUID,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS tokens (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            symbol TEXT NOT NULL,
            name TEXT NOT NULL,
            is_active BOOLEAN NOT NULL DEFAULT TRUE,
            is_verified BOOLEAN NOT NULL DEFAULT FALSE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS trading_pairs (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            pair_name TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'active',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS blockchains (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID,
            name TEXT NOT NULL,
            symbol TEXT NOT NULL,
            chain_id INTEGER NOT NULL DEFAULT 0,
            is_evm BOOLEAN NOT NULL DEFAULT FALSE,
            is_active BOOLEAN NOT NULL DEFAULT TRUE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS fee_structures (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID,
            fee_type TEXT NOT NULL,
            fee_percent NUMERIC(10,4),
            is_active BOOLEAN NOT NULL DEFAULT TRUE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS webhooks (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID,
            name TEXT NOT NULL,
            url TEXT NOT NULL,
            events TEXT[] NOT NULL DEFAULT '{}',
            is_active BOOLEAN NOT NULL DEFAULT TRUE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS notifications (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            admin_id UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
            title TEXT NOT NULL,
            message TEXT NOT NULL,
            notification_type TEXT NOT NULL DEFAULT 'info',
            is_read BOOLEAN NOT NULL DEFAULT FALSE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS audit_logs (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            admin_id UUID,
            action TEXT NOT NULL,
            resource_type TEXT NOT NULL,
            resource_id TEXT,
            details JSONB,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS feature_flags (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name TEXT NOT NULL UNIQUE,
            description TEXT,
            is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
            rollout_percentage INTEGER NOT NULL DEFAULT 0,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS ip_whitelist (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            ip_address TEXT NOT NULL,
            description TEXT,
            is_active BOOLEAN NOT NULL DEFAULT TRUE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS tickets (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            user_id UUID,
            title TEXT NOT NULL,
            description TEXT,
            ticket_type TEXT NOT NULL DEFAULT 'support',
            priority TEXT NOT NULL DEFAULT 'medium',
            status TEXT NOT NULL DEFAULT 'open',
            created_by UUID,
            assigned_to UUID,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS ticket_messages (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
            sender_id UUID,
            message TEXT NOT NULL,
            is_internal BOOLEAN NOT NULL DEFAULT FALSE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        // --- approval workflows / backups ---
        r#"CREATE TABLE IF NOT EXISTS approval_workflows (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name TEXT NOT NULL,
            workflow_type TEXT NOT NULL,
            threshold_amount NUMERIC(40,8),
            required_approvals INTEGER NOT NULL DEFAULT 1,
            is_active BOOLEAN NOT NULL DEFAULT TRUE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS approval_requests (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            workflow_id UUID,
            request_type TEXT NOT NULL,
            resource_id TEXT,
            requester_id UUID NOT NULL,
            status TEXT NOT NULL DEFAULT 'pending',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS backups (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            backup_type TEXT NOT NULL,
            file_path TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'pending',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        // --- RBAC ---
        r#"CREATE TABLE IF NOT EXISTS admin_roles (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            name TEXT NOT NULL,
            scopes TEXT[] NOT NULL DEFAULT '{}',
            is_system BOOLEAN NOT NULL DEFAULT FALSE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS admin_permissions (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            scope TEXT NOT NULL UNIQUE,
            description TEXT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS admin_role_assignments (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            admin_id UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
            role_id UUID NOT NULL REFERENCES admin_roles(id) ON DELETE CASCADE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            UNIQUE(admin_id, role_id)
        )"#,
        // --- trading domain governance ---
        r#"CREATE TABLE IF NOT EXISTS futures_positions (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            user_id UUID,
            pair TEXT NOT NULL,
            side TEXT NOT NULL,
            size NUMERIC(40,8) NOT NULL DEFAULT 0,
            leverage NUMERIC(10,2) NOT NULL DEFAULT 1,
            entry_price NUMERIC(40,8) NOT NULL DEFAULT 0,
            liquidation_price NUMERIC(40,8) NOT NULL DEFAULT 0,
            margin NUMERIC(40,8) NOT NULL DEFAULT 0,
            chain_id BIGINT NOT NULL DEFAULT 0,
            status TEXT NOT NULL DEFAULT 'open',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS options_contracts (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            symbol TEXT NOT NULL,
            option_type TEXT NOT NULL,
            strike NUMERIC(40,8) NOT NULL DEFAULT 0,
            expiry TIMESTAMPTZ NOT NULL,
            status TEXT NOT NULL DEFAULT 'active',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS copy_trading_configs (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            lead_trader_id UUID NOT NULL,
            max_followers INTEGER NOT NULL DEFAULT 0,
            fee_bps INTEGER NOT NULL DEFAULT 0,
            status TEXT NOT NULL DEFAULT 'active',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS convert_orders (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            from_currency TEXT NOT NULL,
            to_currency TEXT NOT NULL,
            spread_bps INTEGER NOT NULL DEFAULT 0,
            status TEXT NOT NULL DEFAULT 'active',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS onramp_orders (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            user_id UUID,
            fiat_currency TEXT NOT NULL,
            fiat_amount NUMERIC(40,8) NOT NULL DEFAULT 0,
            crypto_currency TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'pending',
            reject_reason TEXT,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS offramp_orders (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            user_id UUID,
            crypto_currency TEXT NOT NULL,
            crypto_amount NUMERIC(40,8) NOT NULL DEFAULT 0,
            fiat_currency TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'pending',
            reject_reason TEXT,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS p2p_clients (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            user_id UUID NOT NULL,
            display_name TEXT NOT NULL,
            rating NUMERIC(3,2) NOT NULL DEFAULT 0,
            total_trades BIGINT NOT NULL DEFAULT 0,
            status TEXT NOT NULL DEFAULT 'active',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS partners (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            name TEXT NOT NULL,
            partner_type TEXT NOT NULL,
            api_key_hint TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'pending',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS reward_campaigns (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            name TEXT NOT NULL,
            reward_type TEXT NOT NULL,
            amount NUMERIC(40,8) NOT NULL DEFAULT 0,
            status TEXT NOT NULL DEFAULT 'active',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS marketing_campaigns (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            name TEXT NOT NULL,
            channel TEXT NOT NULL,
            budget NUMERIC(40,8) NOT NULL DEFAULT 0,
            status TEXT NOT NULL DEFAULT 'active',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        // --- WL product governance (wl_* tables) ---
        r#"CREATE TABLE IF NOT EXISTS wl_liquidity_sources (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            name TEXT NOT NULL,
            chain TEXT NOT NULL,
            dex TEXT NOT NULL,
            pool_address TEXT NOT NULL,
            token_a TEXT NOT NULL,
            token_b TEXT NOT NULL,
            reserve_a NUMERIC(40,8) NOT NULL DEFAULT 0,
            reserve_b NUMERIC(40,8) NOT NULL DEFAULT 0,
            fee_pct NUMERIC(10,4) NOT NULL DEFAULT 0,
            is_active BOOLEAN NOT NULL DEFAULT TRUE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS wl_liquidity_allocations (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            name TEXT NOT NULL,
            fee_share_pct NUMERIC(10,4) NOT NULL DEFAULT 0,
            destination TEXT NOT NULL,
            is_active BOOLEAN NOT NULL DEFAULT TRUE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS wl_cards (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            user_id UUID,
            holder_name TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'active',
            balance NUMERIC(40,8) NOT NULL DEFAULT 0,
            currency TEXT NOT NULL DEFAULT 'USD',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS wl_card_transactions (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            card_id UUID NOT NULL REFERENCES wl_cards(id) ON DELETE CASCADE,
            amount NUMERIC(40,8) NOT NULL DEFAULT 0,
            merchant TEXT NOT NULL,
            category TEXT NOT NULL DEFAULT 'general',
            status TEXT NOT NULL DEFAULT 'pending',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS wl_bot_operators (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            name TEXT NOT NULL,
            strategy TEXT NOT NULL DEFAULT 'mm',
            status TEXT NOT NULL DEFAULT 'active',
            config JSONB NOT NULL DEFAULT '{}'::jsonb,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        r#"CREATE TABLE IF NOT EXISTS wl_bot_config (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            white_label_id UUID NOT NULL,
            key TEXT NOT NULL,
            value TEXT NOT NULL,
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"#,
        // --- indexes (idempotent) ---
        r#"CREATE INDEX IF NOT EXISTS idx_users_white_label ON users(white_label_id)"#,
        r#"CREATE INDEX IF NOT EXISTS idx_transactions_user ON transactions(user_id)"#,
        r#"CREATE INDEX IF NOT EXISTS idx_withdrawals_user ON withdrawals(user_id)"#,
        r#"CREATE INDEX IF NOT EXISTS idx_kyc_user ON kyc_requests(user_id)"#,
        r#"CREATE INDEX IF NOT EXISTS idx_tokens_white_label ON tokens(white_label_id)"#,
        r#"CREATE INDEX IF NOT EXISTS idx_pairs_white_label ON trading_pairs(white_label_id)"#,
        r#"CREATE INDEX IF NOT EXISTS idx_tickets_white_label ON tickets(white_label_id)"#,
        r#"CREATE INDEX IF NOT EXISTS idx_ticket_messages_ticket ON ticket_messages(ticket_id)"#,
        r#"CREATE INDEX IF NOT EXISTS idx_audit_logs_admin ON audit_logs(admin_id)"#,
        r#"CREATE INDEX IF NOT EXISTS idx_notifications_admin ON notifications(admin_id)"#,
        r#"CREATE INDEX IF NOT EXISTS idx_admin_sessions_admin ON admin_sessions(admin_id)"#,
        r#"CREATE INDEX IF NOT EXISTS idx_admin_roles_white_label ON admin_roles(white_label_id)"#,
        r#"CREATE INDEX IF NOT EXISTS idx_role_assignments_admin ON admin_role_assignments(admin_id)"#,
    ];

    for stmt in stmts {
        sqlx::query(stmt).execute(pool).await?;
    }
    tracing::info!("migrations applied ({} statements)", stmts.len());
    Ok(())
}
