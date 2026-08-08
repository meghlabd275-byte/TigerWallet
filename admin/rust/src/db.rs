//! Database module

use sqlx::{postgres::{PgPool, PgPoolOptions}, Postgres};
use std::time::Duration;

pub type DbPool = PgPool;

pub async fn create_pool() -> Result<DbPool, sqlx::Error> {
    let database_url = std::env::var("DATABASE_URL")
        .unwrap_or_else(|_| "postgres://admin:password@localhost:5432/tigerwallet".to_string());

    PgPoolOptions::new()
        .max_connections(20)
        .acquire_timeout(Duration::from_secs(30))
        .idle_timeout(Duration::from_secs(600))
        .connect(&database_url)
        .await
}

pub async fn run_migrations(pool: &DbPool) -> Result<(), sqlx::Error> {
    sqlx::query(r#"
        CREATE TABLE IF NOT EXISTS admins (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            username VARCHAR(255) NOT NULL UNIQUE,
            email VARCHAR(255) NOT NULL UNIQUE,
            password_hash VARCHAR(255) NOT NULL,
            role VARCHAR(50) NOT NULL DEFAULT 'admin',
            permissions JSONB DEFAULT '[]',
            is_active BOOLEAN DEFAULT true,
            two_factor_enabled BOOLEAN DEFAULT false,
            two_factor_secret VARCHAR(255),
            ip_whitelist TEXT,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            last_login_at TIMESTAMP WITH TIME ZONE,
            failed_attempts INTEGER DEFAULT 0,
            locked_until TIMESTAMP WITH TIME ZONE
        );

        CREATE TABLE IF NOT EXISTS sessions (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            admin_id UUID NOT NULL REFERENCES admins(id),
            token_hash VARCHAR(255) NOT NULL,
            ip_address VARCHAR(50),
            user_agent TEXT,
            expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS users (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id VARCHAR(255) NOT NULL UNIQUE,
            username VARCHAR(255),
            email VARCHAR(255),
            phone VARCHAR(50),
            password_hash VARCHAR(255),
            wallet_address VARCHAR(100),
            status VARCHAR(50) DEFAULT 'active',
            tier INTEGER DEFAULT 0,
            email_verified BOOLEAN DEFAULT false,
            phone_verified BOOLEAN DEFAULT false,
            kyc_status VARCHAR(50) DEFAULT 'none',
            kyc_level INTEGER DEFAULT 0,
            white_label_id UUID,
            referrer_id VARCHAR(255),
            referral_code VARCHAR(255) UNIQUE,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            last_login_at TIMESTAMP WITH TIME ZONE,
            failed_login_count INTEGER DEFAULT 0,
            country VARCHAR(100),
            ip_address VARCHAR(50)
        );

        CREATE TABLE IF NOT EXISTS kyc_requests (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id UUID NOT NULL REFERENCES users(id),
            level INTEGER DEFAULT 1,
            document_type VARCHAR(50),
            document_number VARCHAR(100),
            document_front TEXT,
            document_back TEXT,
            selfie_image TEXT,
            first_name VARCHAR(255),
            last_name VARCHAR(255),
            date_of_birth VARCHAR(20),
            country VARCHAR(100),
            address TEXT,
            status VARCHAR(50) DEFAULT 'pending',
            reject_reason TEXT,
            reviewed_by UUID,
            reviewed_at TIMESTAMP WITH TIME ZONE,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS transactions (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id UUID NOT NULL REFERENCES users(id),
            tx_type VARCHAR(50),
            amount VARCHAR(100),
            currency VARCHAR(50),
            status VARCHAR(50) DEFAULT 'pending',
            from_address VARCHAR(100),
            to_address VARCHAR(100),
            tx_hash VARCHAR(200),
            fee VARCHAR(100),
            chain_id INTEGER,
            is_flagged BOOLEAN DEFAULT false,
            flag_reason TEXT,
            timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS withdrawals (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id UUID NOT NULL REFERENCES users(id),
            amount VARCHAR(100),
            currency VARCHAR(50),
            status VARCHAR(50) DEFAULT 'pending',
            address VARCHAR(100),
            tx_hash VARCHAR(200),
            approved_by UUID,
            processed_at TIMESTAMP WITH TIME ZONE,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS tokens (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            token_id VARCHAR(255) NOT NULL UNIQUE,
            name VARCHAR(255) NOT NULL,
            symbol VARCHAR(50) NOT NULL,
            contract_address VARCHAR(100),
            decimals INTEGER DEFAULT 18,
            is_active BOOLEAN DEFAULT true,
            is_verified BOOLEAN DEFAULT false,
            total_supply VARCHAR(100),
            chain_id INTEGER,
            logo_url TEXT,
            website TEXT,
            description TEXT,
            status VARCHAR(50) DEFAULT 'active',
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS trading_pairs (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            base_token_id UUID NOT NULL REFERENCES tokens(id),
            quote_token_id UUID NOT NULL REFERENCES tokens(id),
            pair_name VARCHAR(100) NOT NULL,
            price VARCHAR(100),
            volume_24h VARCHAR(100),
            liquidity VARCHAR(100),
            status VARCHAR(50) DEFAULT 'active',
            chain_id INTEGER,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS blockchains (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name VARCHAR(100) NOT NULL,
            symbol VARCHAR(50) NOT NULL,
            chain_id INTEGER NOT NULL UNIQUE,
            is_evm BOOLEAN DEFAULT true,
            rpc_url TEXT NOT NULL,
            explorer_url TEXT,
            native_token VARCHAR(50),
            decimals INTEGER DEFAULT 18,
            is_active BOOLEAN DEFAULT true,
            avg_gas_price_gwei VARCHAR(50),
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS fee_structures (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            fee_type VARCHAR(50) NOT NULL,
            asset VARCHAR(50) NOT NULL,
            fee_percent VARCHAR(50),
            fee_fixed VARCHAR(50),
            min_fee VARCHAR(50),
            max_fee VARCHAR(50),
            tier VARCHAR(50),
            is_active BOOLEAN DEFAULT true,
            chain_id INTEGER,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS white_labels (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            client_id VARCHAR(255) NOT NULL UNIQUE,
            company_name VARCHAR(255) NOT NULL,
            domain VARCHAR(255) NOT NULL UNIQUE,
            domain_verified BOOLEAN DEFAULT false,
            admin_user_id UUID NOT NULL,
            status VARCHAR(50) DEFAULT 'pending',
            logo_url TEXT,
            primary_color VARCHAR(20),
            secondary_color VARCHAR(20),
            theme_mode VARCHAR(20),
            features JSONB DEFAULT '{}',
            max_users INTEGER DEFAULT 1000,
            max_daily_volume DECIMAL DEFAULT 1000000,
            platform_fee_percent DECIMAL DEFAULT 20,
            custom_fee_percent DECIMAL DEFAULT 0,
            contact_email VARCHAR(255) NOT NULL,
            contact_phone VARCHAR(50),
            activated_at TIMESTAMP WITH TIME ZONE,
            expires_at TIMESTAMP WITH TIME ZONE,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS tickets (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            title VARCHAR(255) NOT NULL,
            description TEXT,
            ticket_type VARCHAR(50),
            priority VARCHAR(50) DEFAULT 'medium',
            status VARCHAR(50) DEFAULT 'open',
            created_by UUID NOT NULL,
            assigned_to UUID,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            resolved_at TIMESTAMP WITH TIME ZONE
        );

        CREATE TABLE IF NOT EXISTS ticket_messages (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            ticket_id UUID NOT NULL REFERENCES tickets(id),
            message TEXT NOT NULL,
            is_internal BOOLEAN DEFAULT false,
            created_by UUID NOT NULL,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS audit_logs (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            admin_id UUID,
            action VARCHAR(100) NOT NULL,
            resource_type VARCHAR(100),
            resource_id VARCHAR(100),
            details JSONB,
            ip_address VARCHAR(50),
            user_agent TEXT,
            success BOOLEAN DEFAULT true,
            error_message TEXT,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS feature_flags (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name VARCHAR(100) NOT NULL UNIQUE,
            description TEXT,
            is_enabled BOOLEAN DEFAULT false,
            rollout_percentage INTEGER DEFAULT 0,
            updated_by UUID,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS notifications (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            admin_id UUID NOT NULL,
            title VARCHAR(255) NOT NULL,
            message TEXT,
            notification_type VARCHAR(50),
            is_read BOOLEAN DEFAULT false,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS webhooks (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name VARCHAR(255) NOT NULL,
            url TEXT NOT NULL,
            secret VARCHAR(255),
            events JSONB DEFAULT '[]',
            is_active BOOLEAN DEFAULT true,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            created_by UUID NOT NULL
        );

        CREATE TABLE IF NOT EXISTS backups (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            backup_type VARCHAR(50) NOT NULL,
            file_path TEXT,
            file_size BIGINT,
            status VARCHAR(50) DEFAULT 'pending',
            created_by UUID,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            completed_at TIMESTAMP WITH TIME ZONE
        );

        CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
        CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
        CREATE INDEX IF NOT EXISTS idx_users_kyc_status ON users(kyc_status);
        CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
        CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);
        CREATE INDEX IF NOT EXISTS idx_kyc_status ON kyc_requests(status);
        CREATE INDEX IF NOT EXISTS idx_audit_logs_admin_id ON audit_logs(admin_id);
        CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
    "#).execute(pool).await?;

    Ok(())
}
