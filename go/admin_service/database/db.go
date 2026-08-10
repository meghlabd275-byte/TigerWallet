// TigerWallet Admin Service - PostgreSQL Database Module
// High-performance database operations with connection pooling

package database

import (
	"context"
	"fmt"
	"log"
	"os"
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

// DefaultConfig returns default PostgreSQL configuration
func DefaultConfig() *DatabaseConfig {
	password := os.Getenv("DATABASE_PASSWORD")
	if password == "" {
		log.Fatal("DATABASE_PASSWORD environment variable must be set")
	}
	host := "localhost"
	if v := os.Getenv("DB_HOST"); v != "" {
		host = v
	}
	return &DatabaseConfig{
		Host:            host,
		Port:            5432,
		Database:        "tigerwallet",
		Username:        "postgres",
		Password:        password,
		MaxConns:        20,
		MinConns:        5,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
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

	// Test connection
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

// InitSchema creates the necessary database tables
func (db *DB) InitSchema(ctx context.Context) error {
	schema := `
	-- Admins table
	CREATE TABLE IF NOT EXISTS admins (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		username VARCHAR(100) NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'admin',
		status VARCHAR(50) NOT NULL DEFAULT 'active',
		two_factor_enabled BOOLEAN DEFAULT FALSE,
		two_factor_secret VARCHAR(255),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		last_login TIMESTAMP WITH TIME ZONE,
		ip_whitelist TEXT[],
		permissions TEXT[]
	);

	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		username VARCHAR(100),
		status VARCHAR(50) NOT NULL DEFAULT 'active',
		kyc_status VARCHAR(50) NOT NULL DEFAULT 'none',
		kyc_level INT DEFAULT 0,
		total_volume DECIMAL(40, 2) DEFAULT 0,
		wallet_count INT DEFAULT 0,
		verified BOOLEAN DEFAULT FALSE,
		suspended BOOLEAN DEFAULT FALSE,
		suspend_reason TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		last_login TIMESTAMP WITH TIME ZONE,
		ip_address VARCHAR(45),
		country VARCHAR(2),
		two_factor_enabled BOOLEAN DEFAULT FALSE,
		referrer_id UUID
	);

	-- KYC records table
	CREATE TABLE IF NOT EXISTS kyc_records (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id),
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		document_type VARCHAR(50) NOT NULL,
		document_front TEXT,
		document_back TEXT,
		selfie_url TEXT,
		risk_score INT,
		submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		reviewed_at TIMESTAMP WITH TIME ZONE,
		reviewer_id UUID REFERENCES admins(id),
		rejection_reason TEXT,
		notes TEXT
	);

	-- Transactions table
	CREATE TABLE IF NOT EXISTS transactions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id),
		hash VARCHAR(100) NOT NULL,
		from_address VARCHAR(100) NOT NULL,
		to_address VARCHAR(100) NOT NULL,
		amount DECIMAL(40, 18) NOT NULL,
		token VARCHAR(50) NOT NULL,
		chain VARCHAR(50) NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		type VARCHAR(50) NOT NULL,
		block_number BIGINT,
		gas_used VARCHAR(50),
		flag_reason TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		confirmed_at TIMESTAMP WITH TIME ZONE
	);

	-- Tokens table
	CREATE TABLE IF NOT EXISTS tokens (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		address VARCHAR(100) NOT NULL,
		name VARCHAR(255) NOT NULL,
		symbol VARCHAR(20) NOT NULL,
		decimals INT NOT NULL,
		total_supply DECIMAL(40, 2),
		is_listed BOOLEAN DEFAULT FALSE,
		is_paused BOOLEAN DEFAULT FALSE,
		chain VARCHAR(50) NOT NULL,
		logo_url TEXT,
		website_url TEXT,
		description TEXT,
		verified BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	-- Trading pairs table
	CREATE TABLE IF NOT EXISTS trading_pairs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		base_token VARCHAR(50) NOT NULL,
		quote_token VARCHAR(50) NOT NULL,
		price DECIMAL(40, 18),
		volume_24h DECIMAL(40, 2) DEFAULT 0,
		liquidity DECIMAL(40, 2) DEFAULT 0,
		is_active BOOLEAN DEFAULT TRUE,
		min_trade_amount DECIMAL(40, 18),
		max_trade_amount DECIMAL(40, 18),
		maker_fee DECIMAL(10, 6),
		taker_fee DECIMAL(10, 6),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	-- Blockchains table
	CREATE TABLE IF NOT EXISTS blockchains (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(100) NOT NULL,
		symbol VARCHAR(20) NOT NULL,
		chain_id INT NOT NULL,
		rpc_url TEXT NOT NULL,
		explorer_url TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		is_testnet BOOLEAN DEFAULT FALSE,
		native_token VARCHAR(20),
		avg_block_time INT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	-- Withdrawals table
	CREATE TABLE IF NOT EXISTS withdrawals (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id),
		amount DECIMAL(40, 18) NOT NULL,
		token VARCHAR(50) NOT NULL,
		chain VARCHAR(50) NOT NULL,
		to_address VARCHAR(100) NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		fee DECIMAL(40, 18),
		tx_hash VARCHAR(100),
		requested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		processed_at TIMESTAMP WITH TIME ZONE,
		processed_by UUID REFERENCES admins(id),
		rejection_reason TEXT
	);

	-- White labels table
	CREATE TABLE IF NOT EXISTS white_labels (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		domain VARCHAR(255) NOT NULL,
		logo_url TEXT,
		primary_color VARCHAR(20),
		secondary_color VARCHAR(20),
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		owner_email VARCHAR(255) NOT NULL,
		owner_name VARCHAR(255),
		api_key VARCHAR(255),
		api_secret VARCHAR(255),
		fee_structure JSONB,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		approved_at TIMESTAMP WITH TIME ZONE
	);

	-- Audit logs table
	CREATE TABLE IF NOT EXISTS audit_logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		admin_id UUID NOT NULL REFERENCES admins(id),
		action VARCHAR(100) NOT NULL,
		resource VARCHAR(100),
		resource_id VARCHAR(100),
		details JSONB,
		ip_address VARCHAR(45),
		user_agent TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	-- Sessions table
	CREATE TABLE IF NOT EXISTS sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		admin_id UUID NOT NULL REFERENCES admins(id),
		token VARCHAR(500) NOT NULL,
		ip_address VARCHAR(45),
		user_agent TEXT,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	-- Fee configuration table
	CREATE TABLE IF NOT EXISTS fee_config (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		trading_fee DECIMAL(10, 6) DEFAULT 0.003,
		withdrawal_fee DECIMAL(40, 18) DEFAULT 0.0001,
		deposit_fee DECIMAL(10, 6) DEFAULT 0,
		maker_fee DECIMAL(10, 6) DEFAULT 0.001,
		taker_fee DECIMAL(10, 6) DEFAULT 0.002,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_by UUID REFERENCES admins(id)
	);

	-- Feature flags table
	CREATE TABLE IF NOT EXISTS feature_flags (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(100) UNIQUE NOT NULL,
		enabled BOOLEAN DEFAULT TRUE,
		description TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
	CREATE INDEX IF NOT EXISTS idx_users_kyc ON users(kyc_status);
	CREATE INDEX IF NOT EXISTS idx_transactions_user ON transactions(user_id);
	CREATE INDEX IF NOT EXISTS idx_transactions_hash ON transactions(hash);
	CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);
	CREATE INDEX IF NOT EXISTS idx_kyc_user ON kyc_records(user_id);
	CREATE INDEX IF NOT EXISTS idx_kyc_status ON kyc_records(status);
	CREATE INDEX IF NOT EXISTS idx_withdrawals_user ON withdrawals(user_id);
	CREATE INDEX IF NOT EXISTS idx_withdrawals_status ON withdrawals(status);
	CREATE INDEX IF NOT EXISTS idx_audit_admin ON audit_logs(admin_id);
	CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at);
	`

	_, err := db.pool.Exec(ctx, schema)
	return err
}

// User operations
func (db *DB) GetUsers(ctx context.Context, params map[string]string) ([]map[string]interface{}, int, error) {
	page := 1
	pageSize := 20
	if p, ok := params["page"]; ok {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps, ok := params["page_size"]; ok {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	query := "SELECT id, email, username, status, kyc_status, kyc_level, total_volume, created_at, last_login, verified, suspended FROM users"
	countQuery := "SELECT COUNT(*) FROM users"
	whereClause := ""

	if status, ok := params["status"]; ok && status != "" {
		whereClause += fmt.Sprintf(" WHERE status = '%s'", status)
	}
	if kyc, ok := params["kyc_status"]; ok && kyc != "" {
		if whereClause == "" {
			whereClause = fmt.Sprintf(" WHERE kyc_status = '%s'", kyc)
		} else {
			whereClause += fmt.Sprintf(" AND kyc_status = '%s'", kyc)
		}
	}
	if search, ok := params["search"]; ok && search != "" {
		if whereClause == "" {
			whereClause = fmt.Sprintf(" WHERE email LIKE '%%%s%%' OR username LIKE '%%%s%%'", search, search)
		} else {
			whereClause += fmt.Sprintf(" AND (email LIKE '%%%s%%' OR username LIKE '%%%s%%')", search, search)
		}
	}

	query += whereClause
	countQuery += whereClause

	// Get total count
	var total int
	err := db.pool.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d OFFSET %d", pageSize, offset)

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// Transaction operations
func (db *DB) GetTransactions(ctx context.Context, params map[string]string) ([]map[string]interface{}, int, error) {
	page := 1
	pageSize := 20
	if p, ok := params["page"]; ok {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps, ok := params["page_size"]; ok {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	query := `SELECT t.id, t.hash, t.from_address, t.to_address, t.amount, t.token, t.chain, 
		t.status, t.type, t.created_at, t.gas_used, t.flag_reason, u.email as user_email
		FROM transactions t LEFT JOIN users u ON t.user_id = u.id`
	countQuery := "SELECT COUNT(*) FROM transactions"
	whereClause := ""

	if status, ok := params["status"]; ok && status != "" {
		whereClause += fmt.Sprintf(" WHERE t.status = '%s'", status)
	}
	if txType, ok := params["type"]; ok && txType != "" {
		if whereClause == "" {
			whereClause = fmt.Sprintf(" WHERE t.type = '%s'", txType)
		} else {
			whereClause += fmt.Sprintf(" AND t.type = '%s'", txType)
		}
	}

	query += whereClause
	countQuery += whereClause

	var total int
	err := db.pool.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY t.created_at DESC LIMIT %d OFFSET %d", pageSize, offset)

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// KYC operations
func (db *DB) GetKycRecords(ctx context.Context, params map[string]string) ([]map[string]interface{}, int, error) {
	page := 1
	pageSize := 20
	if p, ok := params["page"]; ok {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps, ok := params["page_size"]; ok {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	query := `SELECT k.id, k.user_id, k.status, k.document_type, k.risk_score, 
		k.submitted_at, k.reviewed_at, k.rejection_reason, u.email as user_email
		FROM kyc_records k LEFT JOIN users u ON k.user_id = u.id`
	countQuery := "SELECT COUNT(*) FROM kyc_records"
	whereClause := ""

	if status, ok := params["status"]; ok && status != "" {
		whereClause += fmt.Sprintf(" WHERE k.status = '%s'", status)
	}

	query += whereClause
	countQuery += whereClause

	var total int
	err := db.pool.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY k.submitted_at DESC LIMIT %d OFFSET %d", pageSize, offset)

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// Token operations
func (db *DB) GetTokens(ctx context.Context, params map[string]string) ([]map[string]interface{}, int, error) {
	page := 1
	pageSize := 20
	if p, ok := params["page"]; ok {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps, ok := params["page_size"]; ok {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	query := "SELECT id, address, name, symbol, decimals, chain, is_listed, is_paused, logo_url, created_at FROM tokens"
	countQuery := "SELECT COUNT(*) FROM tokens"
	whereClause := ""

	if chain, ok := params["chain"]; ok && chain != "" {
		whereClause += fmt.Sprintf(" WHERE chain = '%s'", chain)
	}
	if listed, ok := params["listed"]; ok && listed != "" {
		if whereClause == "" {
			whereClause = fmt.Sprintf(" WHERE is_listed = %s", listed)
		} else {
			whereClause += fmt.Sprintf(" AND is_listed = %s", listed)
		}
	}

	query += whereClause
	countQuery += whereClause

	var total int
	err := db.pool.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d OFFSET %d", pageSize, offset)

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// Withdrawal operations
func (db *DB) GetWithdrawals(ctx context.Context, params map[string]string) ([]map[string]interface{}, int, error) {
	page := 1
	pageSize := 20
	if p, ok := params["page"]; ok {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps, ok := params["page_size"]; ok {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	query := `SELECT w.id, w.amount, w.token, w.chain, w.to_address, w.status, w.fee, 
		w.tx_hash, w.requested_at, w.processed_at, u.email as user_email
		FROM withdrawals w LEFT JOIN users u ON w.user_id = u.id`
	countQuery := "SELECT COUNT(*) FROM withdrawals"
	whereClause := ""

	if status, ok := params["status"]; ok && status != "" {
		whereClause += fmt.Sprintf(" WHERE w.status = '%s'", status)
	}

	query += whereClause
	countQuery += whereClause

	var total int
	err := db.pool.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY w.requested_at DESC LIMIT %d OFFSET %d", pageSize, offset)

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// Analytics operations
func (db *DB) GetAnalytics(ctx context.Context, period string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Total users
	var totalUsers int64
	err := db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&totalUsers)
	if err != nil {
		return nil, err
	}
	result["totalUsers"] = totalUsers

	// Active users (last 24h)
	var activeUsers int64
	err = db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE last_login > NOW() - INTERVAL '24 hours'").Scan(&activeUsers)
	if err != nil {
		return nil, err
	}
	result["activeUsers"] = activeUsers

	// Total transactions
	var totalTransactions int64
	err = db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions").Scan(&totalTransactions)
	if err != nil {
		return nil, err
	}
	result["totalTransactions"] = totalTransactions

	// Total volume
	var totalVolume string
	err = db.pool.QueryRow(ctx, "SELECT COALESCE(SUM(amount), 0)::text FROM transactions WHERE status = 'confirmed'").Scan(&totalVolume)
	if err != nil {
		return nil, err
	}
	result["totalVolume"] = totalVolume

	// 24h transactions
	var dailyTransactions int64
	err = db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions WHERE created_at > NOW() - INTERVAL '24 hours'").Scan(&dailyTransactions)
	if err != nil {
		return nil, err
	}
	result["dailyTransactions"] = dailyTransactions

	// Pending KYC
	var pendingKyc int64
	err = db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM kyc_records WHERE status = 'pending'").Scan(&pendingKyc)
	if err != nil {
		return nil, err
	}
	result["pendingKyc"] = pendingKyc

	// Pending withdrawals
	var pendingWithdrawals int64
	err = db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM withdrawals WHERE status = 'pending'").Scan(&pendingWithdrawals)
	if err != nil {
		return nil, err
	}
	result["pendingWithdrawals"] = pendingWithdrawals

	return result, nil
}

// Fee config operations
func (db *DB) GetFeeConfig(ctx context.Context) (map[string]interface{}, error) {
	row := db.pool.QueryRow(ctx, "SELECT trading_fee, withdrawal_fee, deposit_fee, maker_fee, taker_fee FROM fee_config ORDER BY created_at DESC LIMIT 1")

	var tradingFee, withdrawalFee, depositFee, makerFee, takerFee string
	err := row.Scan(&tradingFee, &withdrawalFee, &depositFee, &makerFee, &takerFee)
	if err != nil {
		// Return defaults if no config exists
		return map[string]interface{}{
			"tradingFee":    "0.003",
			"withdrawalFee": "0.0001",
			"depositFee":    "0",
			"makerFee":      "0.001",
			"takerFee":      "0.002",
		}, nil
	}

	return map[string]interface{}{
		"tradingFee":    tradingFee,
		"withdrawalFee": withdrawalFee,
		"depositFee":    depositFee,
		"makerFee":      makerFee,
		"takerFee":      takerFee,
	}, nil
}

// Update fee config
func (db *DB) UpdateFeeConfig(ctx context.Context, config map[string]string, adminID string) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO fee_config (trading_fee, withdrawal_fee, deposit_fee, maker_fee, taker_fee, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, config["tradingFee"], config["withdrawalFee"], config["depositFee"], config["makerFee"], config["takerFee"], adminID)
	return err
}

// White label operations
func (db *DB) GetWhiteLabels(ctx context.Context, params map[string]string) ([]map[string]interface{}, int, error) {
	page := 1
	pageSize := 20
	if p, ok := params["page"]; ok {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps, ok := params["page_size"]; ok {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	query := "SELECT id, name, domain, status, owner_email, owner_name, primary_color, created_at FROM white_labels"
	countQuery := "SELECT COUNT(*) FROM white_labels"
	whereClause := ""

	if status, ok := params["status"]; ok && status != "" {
		whereClause += fmt.Sprintf(" WHERE status = '%s'", status)
	}

	query += whereClause
	countQuery += whereClause

	var total int
	err := db.pool.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d OFFSET %d", pageSize, offset)

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// Admin operations
func (db *DB) GetAdmins(ctx context.Context) ([]map[string]interface{}, error) {
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

// Audit log operations
func (db *DB) GetAuditLogs(ctx context.Context, params map[string]string) ([]map[string]interface{}, int, error) {
	page := 1
	pageSize := 20
	if p, ok := params["page"]; ok {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps, ok := params["page_size"]; ok {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	query := `SELECT a.id, a.action, a.resource, a.resource_id, a.details, a.ip_address, a.created_at, 
		ad.email as admin_email FROM audit_logs a LEFT JOIN admins ad ON a.admin_id = ad.id`
	countQuery := "SELECT COUNT(*) FROM audit_logs"

	var total int
	err := db.pool.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY a.created_at DESC LIMIT %d OFFSET %d", pageSize, offset)

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// Create audit log
func (db *DB) CreateAuditLog(ctx context.Context, adminID, action, resource, resourceID, details, ipAddress, userAgent string) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO audit_logs (admin_id, action, resource, resource_id, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, adminID, action, resource, resourceID, details, ipAddress, userAgent)
	return err
}

// Blockchain operations
func (db *DB) GetBlockchains(ctx context.Context) ([]map[string]interface{}, error) {
	query := "SELECT id, name, symbol, chain_id, rpc_url, explorer_url, is_active, is_testnet, native_token FROM blockchains ORDER BY name"

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

// Trading pairs operations
func (db *DB) GetTradingPairs(ctx context.Context) ([]map[string]interface{}, error) {
	query := "SELECT id, base_token, quote_token, price, volume_24h, liquidity, is_active, maker_fee, taker_fee FROM trading_pairs ORDER BY base_token"

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
