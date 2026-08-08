// Database - PostgreSQL database connection and operations
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tigerwallet/white-label-admin/internal/config"
)

var Pool *pgxpool.Pool

func Initialize(cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = cfg.DatabaseMaxConns
	poolConfig.MinConns = cfg.DatabaseMinConns

	Pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := Pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	if err := runMigrations(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}

func runMigrations(ctx context.Context) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS admin_users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), username VARCHAR(255) UNIQUE NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL, role VARCHAR(50) NOT NULL DEFAULT 'admin', two_factor_secret VARCHAR(255), two_factor_enabled BOOLEAN DEFAULT FALSE, is_active BOOLEAN DEFAULT TRUE, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), last_login TIMESTAMP WITH TIME ZONE)`,
		`CREATE TABLE IF NOT EXISTS admin_sessions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), admin_id UUID REFERENCES admin_users(id) ON DELETE CASCADE, token_hash VARCHAR(255) NOT NULL, ip_address VARCHAR(45), user_agent TEXT, expires_at TIMESTAMP WITH TIME ZONE NOT NULL, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS ip_whitelist (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), ip_address VARCHAR(45) UNIQUE NOT NULL, description TEXT, is_active BOOLEAN DEFAULT TRUE, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), created_by UUID REFERENCES admin_users(id))`,
		`CREATE TABLE IF NOT EXISTS feature_flags (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(255) UNIQUE NOT NULL, description TEXT, is_enabled BOOLEAN DEFAULT FALSE, rollout_percentage INTEGER DEFAULT 0, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), updated_by UUID REFERENCES admin_users(id))`,
		`CREATE TABLE IF NOT EXISTS audit_logs (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), admin_id UUID REFERENCES admin_users(id), action VARCHAR(100) NOT NULL, resource_type VARCHAR(100) NOT NULL, resource_id VARCHAR(255), details JSONB, ip_address VARCHAR(45), user_agent TEXT, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS notifications (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), admin_id UUID REFERENCES admin_users(id) ON DELETE CASCADE, title VARCHAR(255) NOT NULL, message TEXT NOT NULL, notification_type VARCHAR(50) NOT NULL, is_read BOOLEAN DEFAULT FALSE, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email VARCHAR(255) UNIQUE NOT NULL, username VARCHAR(255) UNIQUE NOT NULL, wallet_address VARCHAR(255), kyc_status VARCHAR(50) DEFAULT 'none', status VARCHAR(50) DEFAULT 'active', two_factor_enabled BOOLEAN DEFAULT FALSE, ip_address VARCHAR(45), country VARCHAR(100), created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), last_login TIMESTAMP WITH TIME ZONE)`,
		`CREATE TABLE IF NOT EXISTS kyc_requests (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, doc_type VARCHAR(50) NOT NULL, status VARCHAR(50) DEFAULT 'pending', document_url TEXT, submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), reviewed_at TIMESTAMP WITH TIME ZONE, reviewed_by UUID REFERENCES admin_users(id), reject_reason TEXT)`,
		`CREATE TABLE IF NOT EXISTS transactions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE SET NULL, type VARCHAR(50) NOT NULL, amount DECIMAL(40, 8) NOT NULL, currency VARCHAR(20) NOT NULL, status VARCHAR(50) DEFAULT 'pending', from_address VARCHAR(255), to_address VARCHAR(255), tx_hash VARCHAR(255), fee DECIMAL(40, 8), chain_id INTEGER, timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS withdrawals (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, amount DECIMAL(40, 8) NOT NULL, currency VARCHAR(20) NOT NULL, status VARCHAR(50) DEFAULT 'pending', address VARCHAR(255) NOT NULL, tx_hash VARCHAR(255), approved_by UUID REFERENCES admin_users(id), processed_at TIMESTAMP WITH TIME ZONE, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS tokens (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), symbol VARCHAR(20) UNIQUE NOT NULL, name VARCHAR(255) NOT NULL, contract_address VARCHAR(255), decimals INTEGER DEFAULT 18, is_active BOOLEAN DEFAULT TRUE, is_verified BOOLEAN DEFAULT FALSE, total_supply DECIMAL(40, 8), chain_id INTEGER, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS trading_pairs (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), base_token_id UUID REFERENCES tokens(id) ON DELETE CASCADE, quote_token_id UUID REFERENCES tokens(id) ON DELETE CASCADE, pair_name VARCHAR(50) UNIQUE NOT NULL, price DECIMAL(40, 8), volume_24h DECIMAL(40, 8) DEFAULT 0, liquidity DECIMAL(40, 8) DEFAULT 0, status VARCHAR(50) DEFAULT 'active', chain_id INTEGER, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS fee_structures (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), fee_type VARCHAR(50) NOT NULL, asset VARCHAR(20), fee_percent DECIMAL(10, 6) DEFAULT 0, fee_fixed DECIMAL(40, 8) DEFAULT 0, min_fee DECIMAL(40, 8) DEFAULT 0, max_fee DECIMAL(40, 8), tier VARCHAR(50), is_active BOOLEAN DEFAULT TRUE, chain_id INTEGER, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS blockchains (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(100) NOT NULL, symbol VARCHAR(20) NOT NULL, chain_id INTEGER UNIQUE, is_evm BOOLEAN DEFAULT FALSE, rpc_url TEXT, explorer_url TEXT, native_token VARCHAR(20), decimals INTEGER DEFAULT 18, is_active BOOLEAN DEFAULT TRUE, avg_gas_price_gwei DECIMAL(20, 2), created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS webhooks (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(255) NOT NULL, url TEXT NOT NULL, secret VARCHAR(255), events TEXT[] NOT NULL, is_active BOOLEAN DEFAULT TRUE, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), created_by UUID REFERENCES admin_users(id))`,
		`CREATE TABLE IF NOT EXISTS white_labels (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(255) NOT NULL, domain VARCHAR(255) UNIQUE NOT NULL, logo_url TEXT, primary_color VARCHAR(20), secondary_color VARCHAR(20), is_active BOOLEAN DEFAULT TRUE, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS tickets (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), title VARCHAR(255) NOT NULL, description TEXT, ticket_type VARCHAR(50) NOT NULL, priority VARCHAR(20) DEFAULT 'medium', status VARCHAR(50) DEFAULT 'open', created_by UUID REFERENCES admin_users(id), assigned_to UUID REFERENCES admin_users(id), created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), resolved_at TIMESTAMP WITH TIME ZONE)`,
		`CREATE TABLE IF NOT EXISTS ticket_messages (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), ticket_id UUID REFERENCES tickets(id) ON DELETE CASCADE, message TEXT NOT NULL, is_internal BOOLEAN DEFAULT FALSE, created_by UUID REFERENCES admin_users(id), created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS backups (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), backup_type VARCHAR(50) NOT NULL, file_path TEXT NOT NULL, file_size BIGINT, status VARCHAR(50) DEFAULT 'pending', created_by UUID REFERENCES admin_users(id), created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), completed_at TIMESTAMP WITH TIME ZONE)`,
		`CREATE TABLE IF NOT EXISTS approval_workflows (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(255) NOT NULL, description TEXT, workflow_type VARCHAR(100) NOT NULL, threshold_amount DECIMAL(40, 8), required_approvals INTEGER DEFAULT 1, approvers TEXT[], is_active BOOLEAN DEFAULT TRUE, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), created_by UUID REFERENCES admin_users(id))`,
		`CREATE TABLE IF NOT EXISTS approval_requests (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), workflow_id UUID REFERENCES approval_workflows(id), request_type VARCHAR(100) NOT NULL, resource_id VARCHAR(255), requester_id UUID REFERENCES admin_users(id), status VARCHAR(50) DEFAULT 'pending', details JSONB, approved_by UUID REFERENCES admin_users(id), approved_at TIMESTAMP WITH TIME ZONE, reject_reason TEXT, created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS knowledge_articles (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), title VARCHAR(255) NOT NULL, content TEXT NOT NULL, category VARCHAR(100), tags TEXT[], is_published BOOLEAN DEFAULT FALSE, view_count INTEGER DEFAULT 0, created_by UUID REFERENCES admin_users(id), created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS archive_policies (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(255) NOT NULL, table_name VARCHAR(100) NOT NULL, retention_days INTEGER NOT NULL, archive_after_days INTEGER NOT NULL, is_active BOOLEAN DEFAULT TRUE, created_by UUID REFERENCES admin_users(id), created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS archive_records (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), policy_id UUID REFERENCES archive_policies(id), table_name VARCHAR(100) NOT NULL, record_count BIGINT DEFAULT 0, archive_path TEXT, status VARCHAR(50) DEFAULT 'pending', started_at TIMESTAMP WITH TIME ZONE, completed_at TIMESTAMP WITH TIME ZONE, created_by UUID REFERENCES admin_users(id))`,
		`CREATE TABLE IF NOT EXISTS report_configs (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(255) NOT NULL, report_type VARCHAR(50) NOT NULL, parameters JSONB, file_format VARCHAR(20) DEFAULT 'json', is_scheduled BOOLEAN DEFAULT FALSE, schedule VARCHAR(100), created_by UUID REFERENCES admin_users(id), created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS reports (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), config_id UUID REFERENCES report_configs(id), name VARCHAR(255) NOT NULL, file_path TEXT, file_size BIGINT, status VARCHAR(50) DEFAULT 'pending', created_by UUID REFERENCES admin_users(id), created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), completed_at TIMESTAMP WITH TIME ZONE)`,
		`CREATE TABLE IF NOT EXISTS sla_policies (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(255) NOT NULL, description TEXT, priority VARCHAR(20) NOT NULL, response_time_sla INTEGER NOT NULL, resolution_time_sla INTEGER NOT NULL, uptime_sla DECIMAL(5,2), is_active BOOLEAN DEFAULT TRUE, created_by UUID REFERENCES admin_users(id), created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS sla_reports (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), policy_id UUID REFERENCES sla_policies(id), period_start TIMESTAMP WITH TIME ZONE NOT NULL, period_end TIMESTAMP WITH TIME ZONE NOT NULL, total_tickets INTEGER DEFAULT 0, met_sla INTEGER DEFAULT 0, breached_sla INTEGER DEFAULT 0, avg_response_time DECIMAL(10,2), avg_resolution_time DECIMAL(10,2), uptime DECIMAL(5,2), created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS integration_configs (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), integration VARCHAR(50) NOT NULL, name VARCHAR(255) NOT NULL, api_key TEXT, api_secret TEXT, webhook_url TEXT, is_active BOOLEAN DEFAULT TRUE, settings JSONB, created_by UUID REFERENCES admin_users(id), created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(), updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_admin_id ON audit_logs(admin_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_admin_id ON sessions(admin_id)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_admin_id ON notifications(admin_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status)`,
	}

	for _, migration := range migrations {
		if _, err := Pool.Exec(ctx, migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}
