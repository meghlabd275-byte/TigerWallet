// TigerWallet Super Admin Service - PostgreSQL Database Module

package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseConfig holds PostgreSQL connection configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	Database        string
	Username        string
	Password        string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// ConnectionString builds PostgreSQL connection string
func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable&pool_max_conns=%d&pool_min_conns=%d",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.MaxConns, c.MinConns,
	)
}

// DB is the database connection pool wrapper
type DB struct {
	pool   *pgxpool.Pool
	config *DatabaseConfig
	mu     sync.RWMutex
}

// New creates a new database connection pool
func New(config *DatabaseConfig) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(config.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	poolConfig.MaxConns = config.MaxConns
	poolConfig.MinConns = config.MinConns
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		pool:   pool,
		config: config,
	}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// Pool returns the underlying connection pool
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// Feature Flags operations
func (db *DB) GetFeatureFlags(ctx context.Context) ([]map[string]interface{}, error) {
	query := "SELECT name, enabled, description, created_at, updated_at FROM feature_flags ORDER BY name"

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// SetFeatureFlag updates or creates a feature flag
func (db *DB) SetFeatureFlag(ctx context.Context, name string, enabled bool, description string) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO feature_flags (name, enabled, description, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (name) DO UPDATE SET enabled = $2, description = $3, updated_at = NOW()
	`, name, enabled, description)
	return err
}

// Profit Share operations
func (db *DB) GetProfitShares(ctx context.Context) ([]map[string]interface{}, error) {
	query := "SELECT id, name, percentage, recipients, created_at FROM profit_shares ORDER BY created_at DESC"

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// SetProfitShare sets profit share configuration
func (db *DB) SetProfitShare(ctx context.Context, name string, percentage float64, recipients string) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO profit_shares (name, percentage, recipients, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (name) DO UPDATE SET percentage = $2, recipients = $3
	`, name, percentage, recipients)
	return err
}

// Admin operations
func (db *DB) GetAllAdmins(ctx context.Context) ([]map[string]interface{}, error) {
	query := "SELECT id, email, username, role, status, two_factor_enabled, created_at, last_login FROM admins ORDER BY created_at DESC"

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// DeleteAdmin deletes an admin
func (db *DB) DeleteAdmin(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, "UPDATE admins SET status = 'deleted' WHERE id = $1", id)
	return err
}

// White label operations
func (db *DB) GetAllWhiteLabels(ctx context.Context) ([]map[string]interface{}, error) {
	query := "SELECT id, name, domain, status, owner_email, owner_name, fee_structure, created_at, approved_at FROM white_labels ORDER BY created_at DESC"

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// DeleteWhiteLabel deletes a white label
func (db *DB) DeleteWhiteLabel(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, "UPDATE white_labels SET status = 'deleted' WHERE id = $1", id)
	return err
}

// ApproveWhiteLabel approves a white label
func (db *DB) ApproveWhiteLabel(ctx context.Context, id string, adminID string) error {
	_, err := db.pool.Exec(ctx, "UPDATE white_labels SET status = 'active', approved_at = NOW() WHERE id = $1", id)
	return err
}

// API Key operations
func (db *DB) GetAPIKeys(ctx context.Context, whiteLabelID string) ([]map[string]interface{}, error) {
	query := "SELECT id, name, key_hash, created_at, expires_at, last_used FROM api_keys WHERE white_label_id = $1 ORDER BY created_at DESC"

	rows, err := db.pool.Query(ctx, query, whiteLabelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// CreateAPIKey creates a new API key
func (db *DB) CreateAPIKey(ctx context.Context, whiteLabelID, name, keyHash string, expiresAt time.Time) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO api_keys (white_label_id, name, key_hash, created_at, expires_at)
		VALUES ($1, $2, $3, NOW(), $4)
	`, whiteLabelID, name, keyHash, expiresAt)
	return err
}

// Audit Log operations
func (db *DB) CreateAuditLog(ctx context.Context, adminID, action, resource, resourceID, details, ipAddress, userAgent string) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO audit_logs (admin_id, action, resource, resource_id, details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`, adminID, action, resource, resourceID, details, ipAddress, userAgent)
	return err
}
