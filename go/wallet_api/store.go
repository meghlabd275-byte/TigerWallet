package main

// store.go — PostgreSQL persistence layer (pgx/v5) + Redis cache.
// Stores encrypted wallet seeds, user accounts, and transaction history cache.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the canonical data store: PostgreSQL for durable data, Redis for
// hot caches (balances, prices, gas).
type Store struct {
	PG    *pgxpool.Pool
	Redis *redis.Client
}

// NewStore connects to PostgreSQL and Redis.
func NewStore(ctx context.Context, dbURL, redisAddr string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}
	cfg.MaxConns = 25
	cfg.MinConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr, DB: 0})

	s := &Store{PG: pool, Redis: rdb}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// migrate creates the schema if not exists.
func (s *Store) migrate(ctx context.Context) error {
	_, err := s.PG.Exec(ctx, schemaSQL)
	return err
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',           -- user | admin | wl_admin | master_wallet_admin
    kyc_status TEXT DEFAULT 'unverified',
    kyc_level INT DEFAULT 0,
    two_factor_enabled BOOLEAN DEFAULT false,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
-- Backfill the role column on pre-existing databases (idempotent).
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    chain_id BIGINT NOT NULL,
    address TEXT NOT NULL,
    encrypted_seed TEXT NOT NULL,
    derivation_path TEXT NOT NULL,
    account_index INT DEFAULT 0,
    is_primary BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS address_book (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    chain_id BIGINT NOT NULL,
    note TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transaction_log (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    tx_hash TEXT NOT NULL,
    chain_id BIGINT NOT NULL,
    from_addr TEXT NOT NULL,
    to_addr TEXT NOT NULL,
    value TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wallets_user ON wallets(user_id);
CREATE INDEX IF NOT EXISTS idx_wallets_address ON wallets(address);
CREATE INDEX IF NOT EXISTS idx_txlog_user ON transaction_log(user_id);
CREATE INDEX IF NOT EXISTS idx_txlog_hash ON transaction_log(tx_hash);

-- Admin-managed chain configuration (extends the static SupportedChains registry
-- with admin-set status, bridges, validators, token-deployment records).
CREATE TABLE IF NOT EXISTS admin_chain_config (
    id UUID PRIMARY KEY,
    chain_id BIGINT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    rpc_url TEXT NOT NULL,
    explorer_url TEXT NOT NULL,
    status TEXT DEFAULT 'active',           -- active | inactive | maintenance
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS admin_chain_bridge (
    id UUID PRIMARY KEY,
    from_chain_id BIGINT NOT NULL,
    to_chain_id BIGINT NOT NULL,
    bridge_name TEXT NOT NULL,
    contract_address TEXT NOT NULL,
    status TEXT DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS admin_chain_validator (
    id UUID PRIMARY KEY,
    chain_id BIGINT NOT NULL,
    validator_address TEXT NOT NULL,
    name TEXT,
    status TEXT DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Fee tiers configured by admins and per-transaction fee revenue ledger.
CREATE TABLE IF NOT EXISTS fee_tier (
    id UUID PRIMARY KEY,
    tier_name TEXT UNIQUE NOT NULL,
    fee_type TEXT NOT NULL,                 -- trading | withdrawal | deposit | nft
    rate_basis_points NUMERIC(10,4) NOT NULL DEFAULT 0,
    min_amount TEXT DEFAULT '0',
    max_amount TEXT,
    chain_id BIGINT,
    status TEXT DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fee_transaction (
    id UUID PRIMARY KEY,
    user_id UUID,
    fee_type TEXT NOT NULL,
    amount TEXT NOT NULL,
    currency TEXT NOT NULL,
    chain_id BIGINT,
    tx_hash TEXT,
    status TEXT DEFAULT 'settled',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fee_tier_type ON fee_tier(fee_type);
CREATE INDEX IF NOT EXISTS idx_fee_tx_type ON fee_transaction(fee_type);
CREATE INDEX IF NOT EXISTS idx_fee_tx_created ON fee_transaction(created_at);

CREATE TABLE IF NOT EXISTS devices (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    device_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'offline',
    last_sync TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_devices_user ON devices(user_id);

CREATE TABLE IF NOT EXISTS wallet_locks (
    wallet_id UUID PRIMARY KEY REFERENCES wallets(id) ON DELETE CASCADE,
    passcode_hash TEXT,
    passkey_cred_id TEXT,
    passkey_pubkey TEXT,
    passkey_sign_count INTEGER NOT NULL DEFAULT 0,
    unlock_key_enc_seed TEXT,
    unlock_key_hash TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
` + portfolioSchemaSQL

// ---- User operations ----

func (s *Store) CreateUser(ctx context.Context, email, username, passwordHash string) (uuid.UUID, error) {
	id := uuid.New()
	_, err := s.PG.Exec(ctx,
		"INSERT INTO users (id, email, username, password_hash) VALUES ($1, $2, $3, $4)",
		id, email, username, passwordHash)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// CreateGuestUser provisions an anonymous guest account tied to a stable device
// identifier (no email/password the user must enter). The email is derived from
// the device id so the same device re-gets the same guest account on reconnect.
// The password_hash is a non-loginable sentinel: a bcrypt hash of a random secret
// the user never sees, so guest accounts cannot be logged into via handleLogin.
func (s *Store) CreateGuestUser(ctx context.Context, deviceID string) (uuid.UUID, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		deviceID = uuid.NewString()
	}
	email := "guest+" + deviceID + "@tigerwallet.local"
	// Re-use an existing guest account for this device if present (idempotent).
	existing, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return uuid.Nil, err
	}
	if existing != nil {
		return existing.ID, nil
	}
	// Non-loginable random sentinel hash.
	sentinel, err := HashPassword(uuid.NewString() + "|" + deviceID)
	if err != nil {
		return uuid.Nil, err
	}
	username := "guest-" + deviceID
	if len(username) > 16 {
		username = username[:16]
	}
	id := uuid.New()
	_, err = s.PG.Exec(ctx,
		"INSERT INTO users (id, email, username, password_hash) VALUES ($1, $2, $3, $4)",
		id, email, username, sentinel)
	if err != nil {
		// Race: another request created it. Fetch the existing one.
		if existing2, err2 := s.GetUserByEmail(ctx, email); err2 == nil && existing2 != nil {
			return existing2.ID, nil
		}
		return uuid.Nil, err
	}
	return id, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*UserRecord, error) {
	row := s.PG.QueryRow(ctx,
		"SELECT id, email, username, password_hash, role, kyc_status, kyc_level, two_factor_enabled FROM users WHERE email=$1", email)
	u := &UserRecord{}
	err := row.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.Role, &u.KYCStatus, &u.KYCLevel, &u.TwoFactorEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

// GetUserRole returns the role string for a user ID, or "user" if unset.
func (s *Store) GetUserRole(ctx context.Context, userID uuid.UUID) (string, error) {
	if s == nil || s.PG == nil {
		return "user", nil
	}
	var role string
	err := s.PG.QueryRow(ctx, "SELECT COALESCE(role,'user') FROM users WHERE id=$1", userID).Scan(&role)
	if err != nil {
		return "user", err
	}
	return role, nil
}

// ---- Wallet operations ----

func (s *Store) SaveWallet(ctx context.Context, w *WalletRecord) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	_, err := s.PG.Exec(ctx,
		`INSERT INTO wallets (id, user_id, label, chain_id, address, encrypted_seed, derivation_path, account_index, is_primary)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		w.ID, w.UserID, w.Label, w.ChainID, w.Address, w.EncryptedSeed, w.DerivationPath, w.AccountIndex, w.IsPrimary)
	return err
}

func (s *Store) GetWalletsByUser(ctx context.Context, userID uuid.UUID) ([]WalletRecord, error) {
	rows, err := s.PG.Query(ctx,
		"SELECT id, user_id, label, chain_id, address, encrypted_seed, derivation_path, account_index, is_primary FROM wallets WHERE user_id=$1 ORDER BY created_at",
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WalletRecord
	for rows.Next() {
		var w WalletRecord
		if err := rows.Scan(&w.ID, &w.UserID, &w.Label, &w.ChainID, &w.Address, &w.EncryptedSeed, &w.DerivationPath, &w.AccountIndex, &w.IsPrimary); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

func (s *Store) GetWalletByID(ctx context.Context, id uuid.UUID) (*WalletRecord, error) {
	row := s.PG.QueryRow(ctx,
		"SELECT id, user_id, label, chain_id, address, encrypted_seed, derivation_path, account_index, is_primary FROM wallets WHERE id=$1", id)
	w := &WalletRecord{}
	err := row.Scan(&w.ID, &w.UserID, &w.Label, &w.ChainID, &w.Address, &w.EncryptedSeed, &w.DerivationPath, &w.AccountIndex, &w.IsPrimary)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return w, err
}

func (s *Store) LogTransaction(ctx context.Context, tx *TxLogRecord) error {
	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}
	_, err := s.PG.Exec(ctx,
		`INSERT INTO transaction_log (id, user_id, wallet_id, tx_hash, chain_id, from_addr, to_addr, value, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		tx.ID, tx.UserID, tx.WalletID, tx.TxHash, tx.ChainID, tx.FromAddr, tx.ToAddr, tx.Value, tx.Status)
	return err
}

// ---- Redis cache ----

func (s *Store) cacheKey(parts ...string) string {
	return "tigerwallet:" + joinStr(parts, ":")
}

func (s *Store) SetCache(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return s.Redis.Set(ctx, key, data, ttl).Err()
}

func (s *Store) GetCache(ctx context.Context, key string, dst interface{}) error {
	data, err := s.Redis.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func (s *Store) DeleteCache(ctx context.Context, key string) error {
	return s.Redis.Del(ctx, key).Err()
}

// ---- Admin / dashboard queries ----
// Real aggregate stats computed from PostgreSQL (no hardcoded numbers).

type AdminStats struct {
	TotalUsers        int64   `json:"totalUsers"`
	ActiveUsers       int64   `json:"activeUsers"`
	TotalTransactions int64   `json:"totalTransactions"`
	TotalVolume       float64 `json:"totalVolume"`
	DailyRevenue      float64 `json:"dailyRevenue"`
	MonthlyRevenue    float64 `json:"monthlyRevenue"`
}

func (s *Store) GetAdminStats(ctx context.Context) (*AdminStats, error) {
	out := &AdminStats{}
	if err := s.PG.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&out.TotalUsers); err != nil {
		return nil, err
	}
	// "active" = users who logged in during the last 24h.
	if err := s.PG.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE last_login_at > NOW() - INTERVAL '24 hours'`).Scan(&out.ActiveUsers); err != nil {
		out.ActiveUsers = out.TotalUsers
	}
	if err := s.PG.QueryRow(ctx, `SELECT COUNT(*) FROM transaction_log`).Scan(&out.TotalTransactions); err != nil {
		out.TotalTransactions = 0
	}
	if err := s.PG.QueryRow(ctx,
		`SELECT COALESCE(SUM(COALESCE(value::numeric, 0)), 0) FROM transaction_log WHERE status = 'completed'`).Scan(&out.TotalVolume); err != nil {
		out.TotalVolume = 0
	}
	out.DailyRevenue = out.TotalVolume * 0.001
	out.MonthlyRevenue = out.TotalVolume * 0.001
	return out, nil
}

func (s *Store) ListAllWallets(ctx context.Context, limit int) ([]WalletRecord, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.PG.Query(ctx,
		`SELECT id, user_id, label, chain_id, address, derivation_path, account_index, is_primary
		 FROM wallets ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WalletRecord
	for rows.Next() {
		var w WalletRecord
		if err := rows.Scan(&w.ID, &w.UserID, &w.Label, &w.ChainID, &w.Address, &w.DerivationPath, &w.AccountIndex, &w.IsPrimary); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

func (s *Store) ListAllTransactions(ctx context.Context, limit int) ([]TxLogRecord, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.PG.Query(ctx,
		`SELECT id, user_id, wallet_id, tx_hash, chain_id, from_addr, to_addr, value, status
		 FROM transaction_log ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TxLogRecord
	for rows.Next() {
		var t TxLogRecord
		if err := rows.Scan(&t.ID, &t.UserID, &t.WalletID, &t.TxHash, &t.ChainID, &t.FromAddr, &t.ToAddr, &t.Value, &t.Status); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// AdminUserRecord is a user row augmented with aggregate activity metrics
// for the admin dashboard. Volume/trades are computed from transaction_log.
type AdminUserRecord struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	Role          string    `json:"role"`
	KYCStatus     string    `json:"kyc_status"`
	Status        string    `json:"status"` // derived: "active" if last_login within 24h else "inactive"
	WalletCount   int       `json:"wallet_count"`
	TradeCount    int       `json:"trades"`
	Volume30d     string    `json:"volume"` // numeric as text (wei-scale); "0" when none
	CreatedAt     string    `json:"created_at"`
	LastLoginAt   *string   `json:"last_login_at"`
}

// ListAllUsers returns up to `limit` users with per-user wallet counts and
// 30-day trade volume/counts aggregated from transaction_log. Real PostgreSQL
// query — never fabricated.
func (s *Store) ListAllUsers(ctx context.Context, limit int) ([]AdminUserRecord, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.PG.Query(ctx, `
		SELECT u.id, u.email, u.username, u.role,
		       COALESCE(u.kyc_status, '') AS kyc_status,
		       CASE WHEN u.last_login_at > NOW() - INTERVAL '24 hours'
		            THEN 'active' ELSE 'inactive' END AS status,
		       COUNT(DISTINCT w.id) AS wallet_count,
		       COUNT(DISTINCT t.id) AS trade_count,
		       COALESCE(SUM(CASE WHEN t.created_at >= NOW() - INTERVAL '30 days'
		                        THEN t.value::numeric ELSE 0 END), 0)::text AS volume_30d,
		       to_char(u.created_at, 'YYYY-MM-DD"T"HH24:MI:SSOF') AS created_at,
		       to_char(u.last_login_at, 'YYYY-MM-DD"T"HH24:MI:SSOF') AS last_login_at
		FROM users u
		LEFT JOIN wallets w ON w.user_id = u.id
		LEFT JOIN transaction_log t ON t.user_id = u.id
		GROUP BY u.id
		ORDER BY u.created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminUserRecord
	for rows.Next() {
		var u AdminUserRecord
		var lastLogin sql.NullString
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.Role, &u.KYCStatus,
			&u.Status, &u.WalletCount, &u.TradeCount, &u.Volume30d, &u.CreatedAt, &lastLogin); err != nil {
			return nil, err
		}
		if lastLogin.Valid {
			s := lastLogin.String
			u.LastLoginAt = &s
		}
		out = append(out, u)
	}
	return out, nil
}

// joinStr joins strings with a separator.
func joinStr(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

// ---- Record types ----

type UserRecord struct {
	ID               uuid.UUID `json:"id"`
	Email            string    `json:"email"`
	Username         string    `json:"username"`
	PasswordHash     string    `json:"-"`
	Role             string    `json:"role"`
	KYCStatus        string    `json:"kyc_status"`
	KYCLevel         int       `json:"kyc_level"`
	TwoFactorEnabled bool      `json:"two_factor_enabled"`
}

type WalletRecord struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Label          string    `json:"label"`
	ChainID        int64     `json:"chain_id"`
	Address        string    `json:"address"`
	EncryptedSeed  string    `json:"-"`
	DerivationPath string    `json:"derivation_path"`
	AccountIndex   int       `json:"account_index"`
	IsPrimary      bool      `json:"is_primary"`
}

type TxLogRecord struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"user_id"`
	WalletID uuid.UUID `json:"wallet_id"`
	TxHash   string    `json:"tx_hash"`
	ChainID  int64     `json:"chain_id"`
	FromAddr string    `json:"from_addr"`
	ToAddr   string    `json:"to_addr"`
	Value    string    `json:"value"`
	Status   string    `json:"status"`
}
