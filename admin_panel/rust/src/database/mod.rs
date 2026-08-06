//! Database module for TigerWallet Admin Panel
use sqlx::{postgres::PgPoolOptions, PgPool, Row};
use std::time::Duration;
use anyhow::Result;

pub struct Database {
    pool: PgPool,
}

impl Database {
    pub async fn new(database_url: &str) -> Result<Self> {
        let pool = PgPoolOptions::new()
            .max_connections(25)
            .min_connections(5)
            .acquire_timeout(Duration::from_secs(30))
            .idle_timeout(Duration::from_secs(600))
            .connect(database_url)
            .await?;

        Ok(Self { pool })
    }

    pub fn pool(&self) -> &PgPool {
        &self.pool
    }

    pub async fn run_migrations(&self) -> Result<()> {
        // Run migrations - in production, use sqlx-cli or a migration tool
        let migrations = vec![
            // Admin users
            r#"CREATE TABLE IF NOT EXISTS admin_users (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                username VARCHAR(255) UNIQUE NOT NULL,
                email VARCHAR(255) UNIQUE NOT NULL,
                password_hash VARCHAR(255) NOT NULL,
                role VARCHAR(50) NOT NULL DEFAULT 'admin',
                two_factor_secret VARCHAR(255),
                two_factor_enabled BOOLEAN DEFAULT FALSE,
                is_active BOOLEAN DEFAULT TRUE,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                last_login TIMESTAMP WITH TIME ZONE
            )"#,
            // Sessions
            r#"CREATE TABLE IF NOT EXISTS admin_sessions (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                admin_id UUID REFERENCES admin_users(id) ON DELETE CASCADE,
                token_hash VARCHAR(255) NOT NULL,
                ip_address VARCHAR(45),
                user_agent TEXT,
                expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
            )"#,
            // Users
            r#"CREATE TABLE IF NOT EXISTS users (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                email VARCHAR(255) UNIQUE NOT NULL,
                username VARCHAR(255) UNIQUE NOT NULL,
                wallet_address VARCHAR(255),
                kyc_status VARCHAR(50) DEFAULT 'none',
                status VARCHAR(50) DEFAULT 'active',
                two_factor_enabled BOOLEAN DEFAULT FALSE,
                ip_address VARCHAR(45),
                country VARCHAR(100),
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                last_login TIMESTAMP WITH TIME ZONE
            )"#,
            // Transactions
            r#"CREATE TABLE IF NOT EXISTS transactions (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                user_id UUID REFERENCES users(id) ON DELETE SET NULL,
                type VARCHAR(50) NOT NULL,
                amount DECIMAL(40, 8) NOT NULL,
                currency VARCHAR(20) NOT NULL,
                status VARCHAR(50) DEFAULT 'pending',
                from_address VARCHAR(255),
                to_address VARCHAR(255),
                tx_hash VARCHAR(255),
                fee DECIMAL(40, 8),
                chain_id INTEGER,
                timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
            )"#,
            // Withdrawals
            r#"CREATE TABLE IF NOT EXISTS withdrawals (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                user_id UUID REFERENCES users(id) ON DELETE CASCADE,
                amount DECIMAL(40, 8) NOT NULL,
                currency VARCHAR(20) NOT NULL,
                status VARCHAR(50) DEFAULT 'pending',
                address VARCHAR(255) NOT NULL,
                tx_hash VARCHAR(255),
                approved_by UUID REFERENCES admin_users(id),
                processed_at TIMESTAMP WITH TIME ZONE,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
            )"#,
            // KYC Requests
            r#"CREATE TABLE IF NOT EXISTS kyc_requests (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                user_id UUID REFERENCES users(id) ON DELETE CASCADE,
                doc_type VARCHAR(50) NOT NULL,
                status VARCHAR(50) DEFAULT 'pending',
                document_url TEXT,
                submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                reviewed_at TIMESTAMP WITH TIME ZONE,
                reviewed_by UUID REFERENCES admin_users(id),
                reject_reason TEXT
            )"#,
            // Tokens
            r#"CREATE TABLE IF NOT EXISTS tokens (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                symbol VARCHAR(20) UNIQUE NOT NULL,
                name VARCHAR(255) NOT NULL,
                contract_address VARCHAR(255),
                decimals INTEGER DEFAULT 18,
                is_active BOOLEAN DEFAULT TRUE,
                is_verified BOOLEAN DEFAULT FALSE,
                total_supply DECIMAL(40, 8),
                chain_id INTEGER,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
            )"#,
            // Trading Pairs
            r#"CREATE TABLE IF NOT EXISTS trading_pairs (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                base_token_id UUID REFERENCES tokens(id) ON DELETE CASCADE,
                quote_token_id UUID REFERENCES tokens(id) ON DELETE CASCADE,
                pair_name VARCHAR(50) UNIQUE NOT NULL,
                price DECIMAL(40, 8),
                volume_24h DECIMAL(40, 8) DEFAULT 0,
                liquidity DECIMAL(40, 8) DEFAULT 0,
                status VARCHAR(50) DEFAULT 'active',
                chain_id INTEGER,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
            )"#,
            // Blockchains
            r#"CREATE TABLE IF NOT EXISTS blockchains (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                name VARCHAR(100) NOT NULL,
                symbol VARCHAR(20) NOT NULL,
                chain_id INTEGER UNIQUE,
                is_evm BOOLEAN DEFAULT FALSE,
                rpc_url TEXT,
                explorer_url TEXT,
                native_token VARCHAR(20),
                decimals INTEGER DEFAULT 18,
                is_active BOOLEAN DEFAULT TRUE,
                avg_gas_price_gwei DECIMAL(20, 2),
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
            )"#,
            // Fee Structures
            r#"CREATE TABLE IF NOT EXISTS fee_structures (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                fee_type VARCHAR(50) NOT NULL,
                asset VARCHAR(20),
                fee_percent DECIMAL(10, 6) DEFAULT 0,
                fee_fixed DECIMAL(40, 8) DEFAULT 0,
                min_fee DECIMAL(40, 8) DEFAULT 0,
                max_fee DECIMAL(40, 8),
                tier VARCHAR(50),
                is_active BOOLEAN DEFAULT TRUE,
                chain_id INTEGER,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
            )"#,
            // Webhooks
            r#"CREATE TABLE IF NOT EXISTS webhooks (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                name VARCHAR(255) NOT NULL,
                url TEXT NOT NULL,
                secret VARCHAR(255),
                events TEXT[] NOT NULL,
                is_active BOOLEAN DEFAULT TRUE,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                created_by UUID REFERENCES admin_users(id)
            )"#,
            // Notifications
            r#"CREATE TABLE IF NOT EXISTS notifications (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                admin_id UUID REFERENCES admin_users(id) ON DELETE CASCADE,
                title VARCHAR(255) NOT NULL,
                message TEXT NOT NULL,
                notification_type VARCHAR(50) NOT NULL,
                is_read BOOLEAN DEFAULT FALSE,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
            )"#,
            // Audit Logs
            r#"CREATE TABLE IF NOT EXISTS audit_logs (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                admin_id UUID REFERENCES admin_users(id),
                action VARCHAR(100) NOT NULL,
                resource_type VARCHAR(100) NOT NULL,
                resource_id VARCHAR(255),
                details JSONB,
                ip_address VARCHAR(45),
                user_agent TEXT,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
            )"#,
            // Feature Flags
            r#"CREATE TABLE IF NOT EXISTS feature_flags (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                name VARCHAR(255) UNIQUE NOT NULL,
                description TEXT,
                is_enabled BOOLEAN DEFAULT FALSE,
                rollout_percentage INTEGER DEFAULT 0,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                updated_by UUID REFERENCES admin_users(id)
            )"#,
            // IP Whitelist
            r#"CREATE TABLE IF NOT EXISTS ip_whitelist (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                ip_address VARCHAR(45) UNIQUE NOT NULL,
                description TEXT,
                is_active BOOLEAN DEFAULT TRUE,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                created_by UUID REFERENCES admin_users(id)
            )"#,
            // Tickets
            r#"CREATE TABLE IF NOT EXISTS tickets (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                title VARCHAR(255) NOT NULL,
                description TEXT,
                ticket_type VARCHAR(50) NOT NULL,
                priority VARCHAR(20) DEFAULT 'medium',
                status VARCHAR(50) DEFAULT 'open',
                created_by UUID REFERENCES admin_users(id),
                assigned_to UUID REFERENCES admin_users(id),
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                resolved_at TIMESTAMP WITH TIME ZONE
            )"#,
            // Ticket Messages
            r#"CREATE TABLE IF NOT EXISTS ticket_messages (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                ticket_id UUID REFERENCES tickets(id) ON DELETE CASCADE,
                message TEXT NOT NULL,
                is_internal BOOLEAN DEFAULT FALSE,
                created_by UUID REFERENCES admin_users(id),
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
            )"#,
            // White Labels
            r#"CREATE TABLE IF NOT EXISTS white_labels (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                name VARCHAR(255) NOT NULL,
                domain VARCHAR(255) UNIQUE NOT NULL,
                logo_url TEXT,
                primary_color VARCHAR(20),
                secondary_color VARCHAR(20),
                is_active BOOLEAN DEFAULT TRUE,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
            )"#,
            // Reports
            r#"CREATE TABLE IF NOT EXISTS reports (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                report_type VARCHAR(100) NOT NULL,
                title VARCHAR(255) NOT NULL,
                filters JSONB,
                file_path TEXT,
                status VARCHAR(50) DEFAULT 'pending',
                generated_by UUID REFERENCES admin_users(id),
                created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                completed_at TIMESTAMP WITH TIME ZONE
            )"#,
        ];

        for migration in migrations {
            sqlx::query(migration).execute(&self.pool).await?;
        }

        Ok(())
    }
}
