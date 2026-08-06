package database

import (
	"context"
	"fmt"
	"time"

	"admin_system/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDB struct {
	pool *pgxpool.Pool
}

func NewPostgres(cfg config.DatabaseConfig) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.MaxLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

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

func RunMigrations(db *PostgresDB) error {
	ctx := context.Background()

	migrations := []string{
		// System users (admins for this system)
		`CREATE TABLE IF NOT EXISTS system_users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			role VARCHAR(50) NOT NULL DEFAULT 'admin',
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			last_login_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// System configuration
		`CREATE TABLE IF NOT EXISTS system_config (
			key VARCHAR(100) PRIMARY KEY,
			value TEXT NOT NULL,
			value_type VARCHAR(20) DEFAULT 'string',
			description TEXT,
			is_secret BOOLEAN DEFAULT FALSE,
			updated_by UUID REFERENCES system_users(id),
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// System backups
		`CREATE TABLE IF NOT EXISTS system_backups (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL,
			file_path TEXT,
			file_size BIGINT,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			created_by UUID REFERENCES system_users(id),
			completed_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// System logs
		`CREATE TABLE IF NOT EXISTS system_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			level VARCHAR(20) NOT NULL,
			message TEXT NOT NULL,
			component VARCHAR(100),
			user_id UUID REFERENCES system_users(id),
			ip_address VARCHAR(50),
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// System metrics
		`CREATE TABLE IF NOT EXISTS system_metrics (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			metric_type VARCHAR(50) NOT NULL,
			metric_name VARCHAR(100) NOT NULL,
			value DECIMAL(20, 6) NOT NULL,
			unit VARCHAR(20),
			tags JSONB DEFAULT '{}',
			recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// System alerts
		`CREATE TABLE IF NOT EXISTS system_alerts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			severity VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			source VARCHAR(100),
			acknowledged_by UUID REFERENCES system_users(id),
			acknowledged_at TIMESTAMP,
			resolved_by UUID REFERENCES system_users(id),
			resolved_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Monitoring data
		`CREATE TABLE IF NOT EXISTS monitoring_data (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			resource_type VARCHAR(50) NOT NULL,
			metric_name VARCHAR(100) NOT NULL,
			value DECIMAL(20, 6) NOT NULL,
			unit VARCHAR(20),
			hostname VARCHAR(100),
			recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Network statistics
		`CREATE TABLE IF NOT EXISTS network_stats (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			interface_name VARCHAR(50),
			bytes_sent BIGINT,
			bytes_received BIGINT,
			packets_sent BIGINT,
			packets_received BIGINT,
			errors_in BIGINT,
			errors_out BIGINT,
			recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Process information
		`CREATE TABLE IF NOT EXISTS process_info (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			pid INTEGER NOT NULL,
			name VARCHAR(255) NOT NULL,
			user VARCHAR(100),
			cpu_percent DECIMAL(5, 2),
			memory_percent DECIMAL(5, 2),
			status VARCHAR(20),
			started_at TIMESTAMP,
			recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Refresh tokens
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES system_users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Sessions
		`CREATE TABLE IF NOT EXISTS system_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES system_users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			ip_address VARCHAR(50),
			user_agent TEXT,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs(level)`,
		`CREATE INDEX IF NOT EXISTS idx_system_logs_created ON system_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_system_metrics_type ON system_metrics(metric_type)`,
		`CREATE INDEX IF NOT EXISTS idx_system_metrics_recorded ON system_metrics(recorded_at)`,
		`CREATE INDEX IF NOT EXISTS idx_system_alerts_status ON system_alerts(status)`,
		`CREATE INDEX IF NOT EXISTS idx_monitoring_resource ON monitoring_data(resource_type)`,
		`CREATE INDEX IF NOT EXISTS idx_network_recorded ON network_stats(recorded_at)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(ctx, migration); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, migration)
		}
	}

	fmt.Println("Database migrations completed successfully")
	return nil
}
