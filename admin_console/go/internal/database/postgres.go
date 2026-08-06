package database

import (
	"context"
	"fmt"
	"time"

	"admin_console/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDB struct {
	pool *pgxpool.Pool
}

func NewPostgres(cfg config.DatabaseConfig) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.MaxLifetime
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("Connected to PostgreSQL successfully")
	return &PostgresDB{pool: pool}, nil
}

func (db *PostgresDB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

func (db *PostgresDB) Pool() *pgxpool.Pool {
	return db.pool
}

func (db *PostgresDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

func (db *PostgresDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

func (db *PostgresDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

func (db *PostgresDB) Begin(ctx context.Context) (pgx.Tx, error) {
	return db.pool.Begin(ctx)
}

func (db *PostgresDB) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return db.pool.SendBatch(ctx, b)
}

func RunMigrations(db *PostgresDB) error {
	ctx := context.Background()

	migrations := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			phone VARCHAR(20),
			role VARCHAR(50) NOT NULL DEFAULT 'user',
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			email_verified BOOLEAN DEFAULT FALSE,
			two_factor_enabled BOOLEAN DEFAULT FALSE,
			two_factor_secret VARCHAR(255),
			last_login_at TIMESTAMP,
			login_attempts INTEGER DEFAULT 0,
			locked_until TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// User profiles
		`CREATE TABLE IF NOT EXISTS user_profiles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			avatar_url TEXT,
			bio TEXT,
			date_of_birth DATE,
			address TEXT,
			city VARCHAR(100),
			country VARCHAR(100),
			postal_code VARCHAR(20),
			timezone VARCHAR(50) DEFAULT 'UTC',
			language VARCHAR(10) DEFAULT 'en',
			preferences JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// KYC records
		`CREATE TABLE IF NOT EXISTS kyc_records (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			document_type VARCHAR(50) NOT NULL,
			document_number VARCHAR(100),
			document_front TEXT,
			document_back TEXT,
			selfie_url TEXT,
			proof_of_address TEXT,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			rejection_reason TEXT,
			reviewed_by UUID REFERENCES users(id),
			reviewed_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Tokens
		`CREATE TABLE IF NOT EXISTS tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			symbol VARCHAR(20) NOT NULL,
			contract_address VARCHAR(100),
			chain VARCHAR(50) NOT NULL,
			decimals INTEGER DEFAULT 18,
			total_supply VARCHAR(100),
			description TEXT,
			logo_url TEXT,
			website_url TEXT,
			whitepaper_url TEXT,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			listing_fee DECIMAL(20, 8),
			approved_by UUID REFERENCES users(id),
			approved_at TIMESTAMP,
			rejected_by UUID REFERENCES users(id),
			rejected_at TIMESTAMP,
			rejection_reason TEXT,
			created_by UUID REFERENCES users(id),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Transactions
		`CREATE TABLE IF NOT EXISTS transactions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tx_hash VARCHAR(100) UNIQUE NOT NULL,
			user_id UUID NOT NULL REFERENCES users(id),
			token_id UUID REFERENCES tokens(id),
			type VARCHAR(50) NOT NULL,
			amount DECIMAL(30, 18) NOT NULL,
			fee DECIMAL(30, 18),
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			flag_reason TEXT,
			flagged_by UUID REFERENCES users(id),
			flagged_at TIMESTAMP,
			approved_by UUID REFERENCES users(id),
			approved_at TIMESTAMP,
			rejected_by UUID REFERENCES users(id),
			rejected_at TIMESTAMP,
			from_address VARCHAR(100),
			to_address VARCHAR(100),
			block_number BIGINT,
			chain VARCHAR(50),
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Audit logs
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id),
			action VARCHAR(100) NOT NULL,
			resource_type VARCHAR(100) NOT NULL,
			resource_id UUID,
			old_values JSONB,
			new_values JSONB,
			ip_address VARCHAR(50),
			user_agent TEXT,
			location VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Notifications
		`CREATE TABLE IF NOT EXISTS notifications (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			type VARCHAR(50) NOT NULL,
			read_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Sessions
		`CREATE TABLE IF NOT EXISTS sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			ip_address VARCHAR(50),
			user_agent TEXT,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// API keys
		`CREATE TABLE IF NOT EXISTS api_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name VARCHAR(100) NOT NULL,
			key_hash VARCHAR(255) NOT NULL,
			permissions JSONB DEFAULT '[]',
			rate_limit INTEGER DEFAULT 1000,
			last_used_at TIMESTAMP,
			expires_at TIMESTAMP,
			revoked_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Compliance reports
		`CREATE TABLE IF NOT EXISTS compliance_reports (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			type VARCHAR(50) NOT NULL,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			generated_by UUID REFERENCES users(id),
			date_from DATE,
			date_to DATE,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			file_url TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP
		)`,

		// System config
		`CREATE TABLE IF NOT EXISTS system_config (
			key VARCHAR(100) PRIMARY KEY,
			value TEXT NOT NULL,
			description TEXT,
			updated_by UUID REFERENCES users(id),
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Refresh tokens
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Activity logs (for user activity tracking)
		`CREATE TABLE IF NOT EXISTS user_activities (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			action VARCHAR(100) NOT NULL,
			details JSONB DEFAULT '{}',
			ip_address VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Add indexes
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)`,
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)`,
		`CREATE INDEX IF NOT EXISTS idx_kyc_status ON kyc_records(status)`,
		`CREATE INDEX IF NOT EXISTS idx_kyc_user ON kyc_records(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_status ON tokens(status)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_user ON transactions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_hash ON transactions(tx_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_activities_user ON user_activities(user_id)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(ctx, migration); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, migration)
		}
	}

	fmt.Println("Database migrations completed successfully")
	return nil
}
