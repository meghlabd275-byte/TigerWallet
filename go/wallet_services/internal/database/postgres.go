/**
 * PostgreSQL Database Layer
 * 
 * Implements database operations for wallet services with connection pooling,
 * migrations, and comprehensive CRUD operations.
 */

package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/tigerwallet/wallet-services/internal/config"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("component", "database")

type PostgresDB struct {
	db *sql.DB
}

func NewPostgresDB(cfg config.DatabaseConfig) (*PostgresDB, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=10",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	// Open connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.MaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Infof("Connected to PostgreSQL database: %s@%s:%d/%s", 
		cfg.User, cfg.Host, cfg.Port, cfg.Database)

	return &PostgresDB{db: db}, nil
}

func (p *PostgresDB) Close() error {
	return p.db.Close()
}

func (p *PostgresDB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return p.db.BeginTx(ctx, nil)
}

func (p *PostgresDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return p.db.Exec(query, args...)
}

func (p *PostgresDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return p.db.Query(query, args...)
}

func (p *PostgresDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return p.db.QueryRow(query, args...)
}

// RunMigrations applies all database migrations
func RunMigrations(db *sql.DB) error {
	migrations := []struct {
		name    string
		up     string
		down   string
	}{
		// Users table
		{
			name: "create_users_table",
			up: `
				CREATE TABLE IF NOT EXISTS users (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					email VARCHAR(255) UNIQUE NOT NULL,
					username VARCHAR(100) UNIQUE,
					password_hash VARCHAR(255) NOT NULL,
					first_name VARCHAR(100),
					last_name VARCHAR(100),
					phone VARCHAR(20),
					email_verified BOOLEAN DEFAULT FALSE,
					kyc_status VARCHAR(50) DEFAULT 'none',
					kyc_level INTEGER DEFAULT 0,
					two_factor_enabled BOOLEAN DEFAULT FALSE,
					two_factor_secret VARCHAR(255),
					risk_score INTEGER DEFAULT 0,
					status VARCHAR(50) DEFAULT 'active',
					referral_code VARCHAR(50) UNIQUE,
					referrer_id UUID REFERENCES users(id),
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					last_login_at TIMESTAMP WITH TIME ZONE,
					deleted_at TIMESTAMP WITH TIME ZONE
				);
				CREATE INDEX idx_users_email ON users(email);
				CREATE INDEX idx_users_username ON users(username);
				CREATE INDEX idx_users_status ON users(status);
				CREATE INDEX idx_users_created_at ON users(created_at);
			`,
			down: "DROP TABLE IF EXISTS users CASCADE;",
		},

		// Sessions table
		{
			name: "create_sessions_table",
			up: `
				CREATE TABLE IF NOT EXISTS sessions (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					token_hash VARCHAR(255) NOT NULL,
					refresh_token_hash VARCHAR(255),
					ip_address INET,
					user_agent TEXT,
					device_id VARCHAR(255),
					expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
					refresh_expires_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX idx_sessions_user_id ON sessions(user_id);
				CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
				CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
			`,
			down: "DROP TABLE IF EXISTS sessions CASCADE;",
		},

		// Wallets table
		{
			name: "create_wallets_table",
			up: `
				CREATE TABLE IF NOT EXISTS wallets (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					name VARCHAR(255) NOT NULL,
					type VARCHAR(50) NOT NULL DEFAULT 'hd',
					derivation_type VARCHAR(50) DEFAULT 'bip44',
					encrypted_seed TEXT,
					public_key TEXT,
					chain_type VARCHAR(50) NOT NULL,
					chain_id BIGINT,
					address VARCHAR(255) NOT NULL,
					derivation_path VARCHAR(100),
					is_imported BOOLEAN DEFAULT FALSE,
					is_watch_only BOOLEAN DEFAULT FALSE,
					status VARCHAR(50) DEFAULT 'active',
					metadata JSONB,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX idx_wallets_user_id ON wallets(user_id);
				CREATE INDEX idx_wallets_address ON wallets(address);
				CREATE INDEX idx_wallets_chain_type ON wallets(chain_type);
				CREATE UNIQUE INDEX idx_wallets_user_chain ON wallets(user_id, chain_type, derivation_path);
			`,
			down: "DROP TABLE IF EXISTS wallets CASCADE;",
		},

		// Wallet balances table
		{
			name: "create_wallet_balances_table",
			up: `
				CREATE TABLE IF NOT EXISTS wallet_balances (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
					token_address VARCHAR(255),
					symbol VARCHAR(20) NOT NULL,
					name VARCHAR(100),
					decimals INTEGER DEFAULT 18,
					balance VARCHAR(255) DEFAULT '0',
					pending_balance VARCHAR(255) DEFAULT '0',
					locked_balance VARCHAR(255) DEFAULT '0',
					is_native BOOLEAN DEFAULT TRUE,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					UNIQUE(wallet_id, token_address)
				);
				CREATE INDEX idx_balances_wallet_id ON wallet_balances(wallet_id);
				CREATE INDEX idx_balances_token ON wallet_balances(wallet_id, token_address);
			`,
			down: "DROP TABLE IF EXISTS wallet_balances CASCADE;",
		},

		// Transactions table
		{
			name: "create_transactions_table",
			up: `
				CREATE TABLE IF NOT EXISTS transactions (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
					tx_hash VARCHAR(255) UNIQUE,
					chain_type VARCHAR(50) NOT NULL,
					chain_id BIGINT,
					type VARCHAR(50) NOT NULL,
					status VARCHAR(50) DEFAULT 'pending',
					from_address VARCHAR(255) NOT NULL,
					to_address VARCHAR(255) NOT NULL,
					amount VARCHAR(255) NOT NULL,
					token_address VARCHAR(255),
					token_symbol VARCHAR(20),
					token_decimals INTEGER,
					fee VARCHAR(255),
					fee_token VARCHAR(20),
					nonce BIGINT,
					block_number BIGINT,
					block_hash VARCHAR(255),
					timestamp TIMESTAMP WITH TIME ZONE,
					confirmations INTEGER DEFAULT 0,
					data TEXT,
					metadata JSONB,
					error_message TEXT,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX idx_transactions_user_id ON transactions(user_id);
				CREATE INDEX idx_transactions_wallet_id ON transactions(wallet_id);
				CREATE INDEX idx_transactions_tx_hash ON transactions(tx_hash);
				CREATE INDEX idx_transactions_status ON transactions(status);
				CREATE INDEX idx_transactions_chain ON transactions(chain_type, chain_id);
				CREATE INDEX idx_transactions_timestamp ON transactions(timestamp DESC);
			`,
			down: "DROP TABLE IF EXISTS transactions CASCADE;",
		},

		// Tokens table
		{
			name: "create_tokens_table",
			up: `
				CREATE TABLE IF NOT EXISTS tokens (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					address VARCHAR(255) NOT NULL,
					chain_type VARCHAR(50) NOT NULL,
					chain_id BIGINT,
					symbol VARCHAR(20) NOT NULL,
					name VARCHAR(100) NOT NULL,
					decimals INTEGER DEFAULT 18,
					total_supply VARCHAR(255),
					is_verified BOOLEAN DEFAULT FALSE,
					is_fake BOOLEAN DEFAULT FALSE,
					logo_url TEXT,
					coingecko_id VARCHAR(100),
					price_usd DECIMAL(30, 10),
					market_cap DECIMAL(30, 2),
					volume_24h DECIMAL(30, 2),
					price_change_24h DECIMAL(10, 4),
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					UNIQUE(address, chain_type, chain_id)
				);
				CREATE INDEX idx_tokens_address ON tokens(address);
				CREATE INDEX idx_tokens_symbol ON tokens(symbol);
				CREATE INDEX idx_tokens_chain ON tokens(chain_type, chain_id);
				CREATE INDEX idx_tokens_coingecko ON tokens(coingecko_id);
			`,
			down: "DROP TABLE IF EXISTS tokens CASCADE;",
		},

		// User tokens (tracked tokens)
		{
			name: "create_user_tokens_table",
			up: `
				CREATE TABLE IF NOT EXISTS user_tokens (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
					added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					UNIQUE(user_id, token_id)
				);
				CREATE INDEX idx_user_tokens_user ON user_tokens(user_id);
			`,
			down: "DROP TABLE IF EXISTS user_tokens CASCADE;",
		},

		// Price history table
		{
			name: "create_price_history_table",
			up: `
				CREATE TABLE IF NOT EXISTS price_history (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
					price_usd DECIMAL(30, 10) NOT NULL,
					volume_24h DECIMAL(30, 2),
					market_cap DECIMAL(30, 2),
					timestamp TIMESTAMP WITH TIME ZONE NOT NULL
				);
				CREATE INDEX idx_price_history_token ON price_history(token_id);
				CREATE INDEX idx_price_history_timestamp ON price_history(timestamp DESC);
			`,
			down: "DROP TABLE IF EXISTS price_history CASCADE;",
		},

		// Notifications table
		{
			name: "create_notifications_table",
			up: `
				CREATE TABLE IF NOT EXISTS notifications (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					type VARCHAR(50) NOT NULL,
					title VARCHAR(255) NOT NULL,
					message TEXT,
					data JSONB,
					is_read BOOLEAN DEFAULT FALSE,
					read_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX idx_notifications_user ON notifications(user_id);
				CREATE INDEX idx_notifications_read ON notifications(user_id, is_read);
				CREATE INDEX idx_notifications_timestamp ON notifications(created_at DESC);
			`,
			down: "DROP TABLE IF EXISTS notifications CASCADE;",
		},

		// API Keys table
		{
			name: "create_api_keys_table",
			up: `
				CREATE TABLE IF NOT EXISTS api_keys (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					name VARCHAR(255) NOT NULL,
					key_hash VARCHAR(255) NOT NULL UNIQUE,
					permissions JSONB DEFAULT '[]',
					rate_limit INTEGER DEFAULT 1000,
					expires_at TIMESTAMP WITH TIME ZONE,
					last_used_at TIMESTAMP WITH TIME ZONE,
					status VARCHAR(50) DEFAULT 'active',
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX idx_api_keys_user ON api_keys(user_id);
				CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
			`,
			down: "DROP TABLE IF EXISTS api_keys CASCADE;",
		},

		// Audit logs table
		{
			name: "create_audit_logs_table",
			up: `
				CREATE TABLE IF NOT EXISTS audit_logs (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID REFERENCES users(id) ON DELETE SET NULL,
					action VARCHAR(100) NOT NULL,
					resource_type VARCHAR(50),
					resource_id VARCHAR(255),
					old_value JSONB,
					new_value JSONB,
					ip_address INET,
					user_agent TEXT,
					success BOOLEAN DEFAULT TRUE,
					error_message TEXT,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
				CREATE INDEX idx_audit_logs_action ON audit_logs(action);
				CREATE INDEX idx_audit_logs_timestamp ON audit_logs(created_at DESC);
			`,
			down: "DROP TABLE IF EXISTS audit_logs CASCADE;",
		},

		// Wallets backup table
		{
			name: "create_wallet_backups_table",
			up: `
				CREATE TABLE IF NOT EXISTS wallet_backups (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
					encrypted_backup TEXT NOT NULL,
					backup_type VARCHAR(50) DEFAULT 'auto',
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX idx_wallet_backups_wallet ON wallet_backups(wallet_id);
			`,
			down: "DROP TABLE IF EXISTS wallet_backups CASCADE;",
		},

		// Webhook configurations
		{
			name: "create_webhooks_table",
			up: `
				CREATE TABLE IF NOT EXISTS webhooks (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					url TEXT NOT NULL,
					events JSONB NOT NULL,
					secret_hash VARCHAR(255),
					is_active BOOLEAN DEFAULT TRUE,
					failure_count INTEGER DEFAULT 0,
					last_failure_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX idx_webhooks_user ON webhooks(user_id);
			`,
			down: "DROP TABLE IF EXISTS webhooks CASCADE;",
		},
	}

	// Run each migration in a transaction
	for _, m := range migrations {
		logger.Infof("Applying migration: %s", m.name)

		// Check if migration already applied
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM information_schema.tables 
			WHERE table_name = $1
		`, "schema_migrations").Scan(&count)

		if err == nil && count == 0 {
			// Create migrations table if not exists
			_, err = db.Exec(`
				CREATE TABLE IF NOT EXISTS schema_migrations (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) UNIQUE NOT NULL,
					applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to create migrations table: %w", err)
			}
		}

		// Check if this migration was applied
		var exists bool
		err = db.QueryRow(`
			SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)
		`, m.name).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check migration: %w", err)
		}

		if exists {
			logger.Debugf("Migration already applied: %s", m.name)
			continue
		}

		// Apply migration
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", m.name, err)
		}

		_, err = tx.Exec(m.up)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %s: %w", m.name, err)
		}

		// Record migration
		_, err = tx.Exec(`INSERT INTO schema_migrations (name) VALUES ($1)`, m.name)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", m.name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", m.name, err)
		}

		logger.Infof("Migration applied: %s", m.name)
	}

	logger.Info("All migrations completed successfully")
	return nil
}
