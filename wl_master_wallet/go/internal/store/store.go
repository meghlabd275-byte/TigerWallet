// Package store provides PostgreSQL persistence for the standalone
// WL-MasterWallet backend. It owns its own database (wl_masterwallet) —
// independent of TigerWallet cloud. Tables: users, master_wallets (encrypted
// seed), sub_wallets, transactions, policies, fee_configs, auto_sign_rules,
// audit_log.
package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	s := &Store{db: pool}
	if err := s.migrate(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() { s.db.Close() }

// DB exposes the underlying pgxpool for handlers that run inline SQL (matching
// the canonical master_wallet/backend pattern of direct pgxpool use).
func (s *Store) DB() *pgxpool.Pool { return s.db }

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email VARCHAR(255) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL, role VARCHAR(32) NOT NULL DEFAULT 'user', created_at TIMESTAMPTZ DEFAULT NOW())`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(32) NOT NULL DEFAULT 'user'`,
		`CREATE TABLE IF NOT EXISTS master_wallets (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID, label VARCHAR(255), address VARCHAR(64) NOT NULL, encrypted_seed TEXT NOT NULL, chain_id BIGINT DEFAULT 1, wl_client_id VARCHAR(64), created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_master_wallets_user ON master_wallets(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_master_wallets_wl_client ON master_wallets(wl_client_id)`,
		`CREATE TABLE IF NOT EXISTS sub_wallets (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, label VARCHAR(255), address VARCHAR(64) NOT NULL, derivation_path VARCHAR(255) NOT NULL, chain_id BIGINT DEFAULT 1, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_sub_wallets_master ON sub_wallets(master_wallet_id)`,
		`CREATE TABLE IF NOT EXISTS transactions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, tx_hash VARCHAR(80), tx_type VARCHAR(32), status VARCHAR(32), from_address VARCHAR(64), to_address VARCHAR(64), amount VARCHAR(64), token VARCHAR(64), chain_id BIGINT, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_tx_master ON transactions(master_wallet_id)`,
		`CREATE TABLE IF NOT EXISTS policies (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, name VARCHAR(255), type VARCHAR(64), config JSONB DEFAULT '{}'::jsonb, enabled BOOL DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_policies_master ON policies(master_wallet_id)`,
		`CREATE TABLE IF NOT EXISTS fee_configs (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, name VARCHAR(255), percentage NUMERIC(10,4) DEFAULT 0, cap NUMERIC(30,8) DEFAULT 0, enabled BOOL DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_fee_configs_master ON fee_configs(master_wallet_id)`,
		`CREATE TABLE IF NOT EXISTS auto_sign_rules (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, trigger VARCHAR(255), action VARCHAR(255), enabled BOOL DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_auto_sign_master ON auto_sign_rules(master_wallet_id)`,
		`CREATE TABLE IF NOT EXISTS audit_log (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID, action VARCHAR(255), entity VARCHAR(255), entity_id VARCHAR(255), severity VARCHAR(32) DEFAULT 'info', details JSONB DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_audit_master ON audit_log(master_wallet_id)`,
		// Per-master-wallet governed users (role-bearing accounts).
		`CREATE TABLE IF NOT EXISTS master_wallet_users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, email VARCHAR(255) NOT NULL, name VARCHAR(255), role VARCHAR(32) NOT NULL DEFAULT 'user', password_hash VARCHAR(255) NOT NULL, is_active BOOL DEFAULT TRUE, last_login_at TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE (master_wallet_id, email))`,
		`CREATE INDEX IF NOT EXISTS idx_mw_users_master ON master_wallet_users(master_wallet_id)`,
		// Notifications.
		`CREATE TABLE IF NOT EXISTS notifications (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, user_id UUID, notification_type VARCHAR(64) NOT NULL, category VARCHAR(64), title VARCHAR(255) NOT NULL, message TEXT NOT NULL, priority VARCHAR(32) DEFAULT 'normal', channel VARCHAR(32) DEFAULT 'in_app', is_read BOOL DEFAULT FALSE, data JSONB DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_master ON notifications(master_wallet_id)`,
		// Webhooks.
		`CREATE TABLE IF NOT EXISTS webhooks (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, name VARCHAR(255) NOT NULL, url TEXT NOT NULL, events TEXT[] DEFAULT '{}', retry_count INT DEFAULT 3, is_active BOOL DEFAULT TRUE, is_verified BOOL DEFAULT FALSE, total_delivered BIGINT DEFAULT 0, total_failed BIGINT DEFAULT 0, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_webhooks_master ON webhooks(master_wallet_id)`,
		// Auto-sign logs (audit trail for auto-sign transactions).
		`CREATE TABLE IF NOT EXISTS auto_sign_logs (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, rule_id UUID, tx_hash VARCHAR(80), chain_id BIGINT, from_address VARCHAR(64), to_address VARCHAR(64), amount VARCHAR(64), token VARCHAR(64), status VARCHAR(32) DEFAULT 'pending', error TEXT, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_auto_sign_logs_master ON auto_sign_logs(master_wallet_id)`,
		// UserWallet-management governance: EVM chains.
		`CREATE TABLE IF NOT EXISTS user_chains_evm (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, chain_id BIGINT NOT NULL, name VARCHAR(255) NOT NULL, symbol VARCHAR(64) NOT NULL, rpc_url TEXT NOT NULL, explorer_url TEXT, decimals INT DEFAULT 18, enabled BOOL DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE (master_wallet_id, chain_id))`,
		`CREATE INDEX IF NOT EXISTS idx_user_chains_evm_master ON user_chains_evm(master_wallet_id)`,
		// Non-EVM chains.
		`CREATE TABLE IF NOT EXISTS user_chains_nonevm (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, chain_id BIGINT NOT NULL, name VARCHAR(255) NOT NULL, symbol VARCHAR(64) NOT NULL, chain_type VARCHAR(32) NOT NULL, rpc_url TEXT, explorer_url TEXT, decimals INT DEFAULT 18, bech32_prefix VARCHAR(32), enabled BOOL DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE (master_wallet_id, chain_id))`,
		`CREATE INDEX IF NOT EXISTS idx_user_chains_nonevm_master ON user_chains_nonevm(master_wallet_id)`,
		// Tokens.
		`CREATE TABLE IF NOT EXISTS user_tokens (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, chain_id BIGINT NOT NULL, contract_address VARCHAR(64) NOT NULL, symbol VARCHAR(64) NOT NULL, name VARCHAR(255), decimals INT DEFAULT 18, enabled BOOL DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE (master_wallet_id, chain_id, contract_address))`,
		`CREATE INDEX IF NOT EXISTS idx_user_tokens_master ON user_tokens(master_wallet_id)`,
		// Derived user wallet addresses (one per master_wallet_id + chain + derivation path).
		`CREATE TABLE IF NOT EXISTS user_wallet_addresses (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, user_id UUID, chain_id BIGINT NOT NULL, chain_type VARCHAR(32) NOT NULL, address VARCHAR(128) NOT NULL, derivation_path VARCHAR(255) NOT NULL, label VARCHAR(255), created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_user_wallet_addresses_master ON user_wallet_addresses(master_wallet_id)`,
		// Feature flags.
		`CREATE TABLE IF NOT EXISTS feature_flags (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, flag_id VARCHAR(128) NOT NULL, description TEXT, enabled BOOL DEFAULT FALSE, config JSONB DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE (master_wallet_id, flag_id))`,
		`CREATE INDEX IF NOT EXISTS idx_feature_flags_master ON feature_flags(master_wallet_id)`,
		// Treasury overview snapshots (upserted on read).
		`CREATE TABLE IF NOT EXISTS treasury_overview (master_wallet_id UUID PRIMARY KEY REFERENCES master_wallets(id) ON DELETE CASCADE, total_value_usd NUMERIC(30,8) DEFAULT 0, total_balance TEXT, allocated NUMERIC(30,8) DEFAULT 0, reserved NUMERIC(30,8) DEFAULT 0, updated_at TIMESTAMPTZ DEFAULT NOW())`,
		// Treasury transactions (transfers/sweeps/allocations).
		`CREATE TABLE IF NOT EXISTS treasury_transactions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, tx_type VARCHAR(32) NOT NULL, amount VARCHAR(64) NOT NULL, token_symbol VARCHAR(64), chain_id BIGINT, tx_hash VARCHAR(80), status VARCHAR(32) DEFAULT 'pending', counterparty VARCHAR(64), notes TEXT, created_at TIMESTAMPTZ DEFAULT NOW(), confirmed_at TIMESTAMPTZ)`,
		`CREATE INDEX IF NOT EXISTS idx_treasury_tx_master ON treasury_transactions(master_wallet_id)`,
		// Multisig wallets (threshold + owners).
		`CREATE TABLE IF NOT EXISTS multisig_wallets (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), master_wallet_id UUID REFERENCES master_wallets(id) ON DELETE CASCADE, name VARCHAR(255) NOT NULL, chain_id BIGINT DEFAULT 1, threshold INT NOT NULL, owners TEXT[] NOT NULL, nonce INT DEFAULT 0, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_multisig_wallets_master ON multisig_wallets(master_wallet_id)`,
		// Multisig transactions (collect signatures, execute at threshold).
		`CREATE TABLE IF NOT EXISTS multisig_transactions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), multisig_wallet_id UUID REFERENCES multisig_wallets(id) ON DELETE CASCADE, to_address VARCHAR(64) NOT NULL, value VARCHAR(64) NOT NULL, data TEXT DEFAULT '0x', nonce INT NOT NULL, status VARCHAR(32) DEFAULT 'pending', signatures JSONB DEFAULT '[]'::jsonb, created_at TIMESTAMPTZ DEFAULT NOW(), executed_at TIMESTAMPTZ)`,
		`CREATE INDEX IF NOT EXISTS idx_multisig_tx_wallet ON multisig_transactions(multisig_wallet_id)`,
	}
	for _, st := range stmts {
		if _, err := s.db.Exec(ctx, st); err != nil {
			return err
		}
	}
	return nil
}

// ==================== Users ====================

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (uuid.UUID, error) {
	id := uuid.New()
	_, err := s.db.Exec(ctx, `INSERT INTO users (id, email, password_hash) VALUES ($1,$2,$3)`, id, email, passwordHash)
	return id, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (uuid.UUID, string, error) {
	var id uuid.UUID
	var hash string
	err := s.db.QueryRow(ctx, `SELECT id, password_hash FROM users WHERE email=$1`, email).Scan(&id, &hash)
	return id, hash, err
}

// ==================== Master wallets ====================

type MasterWallet struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Label         string
	Address       string
	EncryptedSeed string
	ChainID       int64
	WLClientID    string
	CreatedAt     time.Time
}

func (s *Store) CreateMasterWallet(ctx context.Context, userID uuid.UUID, label, address, encSeed string, chainID int64, wlClientID string) (*MasterWallet, error) {
	id := uuid.New()
	var createdAt time.Time
	err := s.db.QueryRow(ctx,
		`INSERT INTO master_wallets (id, user_id, label, address, encrypted_seed, chain_id, wl_client_id) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING created_at`,
		id, userID, label, address, encSeed, chainID, wlClientID).Scan(&createdAt)
	if err != nil {
		return nil, err
	}
	return &MasterWallet{ID: id, UserID: userID, Label: label, Address: address, EncryptedSeed: encSeed, ChainID: chainID, WLClientID: wlClientID, CreatedAt: createdAt}, nil
}

func (s *Store) GetMasterWallet(ctx context.Context, id uuid.UUID) (*MasterWallet, error) {
	var w MasterWallet
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, label, address, encrypted_seed, chain_id, wl_client_id, created_at FROM master_wallets WHERE id=$1`, id).
		Scan(&w.ID, &w.UserID, &w.Label, &w.Address, &w.EncryptedSeed, &w.ChainID, &w.WLClientID, &w.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *Store) ListMasterWallets(ctx context.Context, userID uuid.UUID) ([]MasterWallet, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, label, address, encrypted_seed, chain_id, wl_client_id, created_at FROM master_wallets WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MasterWallet{}
	for rows.Next() {
		var w MasterWallet
		_ = rows.Scan(&w.ID, &w.UserID, &w.Label, &w.Address, &w.EncryptedSeed, &w.ChainID, &w.WLClientID, &w.CreatedAt)
		out = append(out, w)
	}
	return out, nil
}

func (s *Store) DeleteMasterWallet(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM master_wallets WHERE id=$1`, id)
	return err
}

// ==================== Sub wallets ====================

type SubWallet struct {
	ID             uuid.UUID
	MasterWalletID uuid.UUID
	Label          string
	Address        string
	DerivationPath string
	ChainID        int64
	CreatedAt      time.Time
}

func (s *Store) CreateSubWallet(ctx context.Context, masterWalletID uuid.UUID, label, address, derivationPath string, chainID int64) (*SubWallet, error) {
	id := uuid.New()
	var createdAt time.Time
	err := s.db.QueryRow(ctx,
		`INSERT INTO sub_wallets (id, master_wallet_id, label, address, derivation_path, chain_id) VALUES ($1,$2,$3,$4,$5,$6) RETURNING created_at`,
		id, masterWalletID, label, address, derivationPath, chainID).Scan(&createdAt)
	if err != nil {
		return nil, err
	}
	return &SubWallet{ID: id, MasterWalletID: masterWalletID, Label: label, Address: address, DerivationPath: derivationPath, ChainID: chainID, CreatedAt: createdAt}, nil
}

func (s *Store) ListSubWallets(ctx context.Context, masterWalletID uuid.UUID) ([]SubWallet, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, master_wallet_id, label, address, derivation_path, chain_id, created_at FROM sub_wallets WHERE master_wallet_id=$1 ORDER BY created_at DESC`, masterWalletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SubWallet{}
	for rows.Next() {
		var sw SubWallet
		_ = rows.Scan(&sw.ID, &sw.MasterWalletID, &sw.Label, &sw.Address, &sw.DerivationPath, &sw.ChainID, &sw.CreatedAt)
		out = append(out, sw)
	}
	return out, nil
}

// ==================== Transactions ====================

func (s *Store) CreateTransaction(ctx context.Context, masterWalletID uuid.UUID, txHash, txType, status, fromAddr, toAddr, amount, token string, chainID int64) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO transactions (id, master_wallet_id, tx_hash, tx_type, status, from_address, to_address, amount, token, chain_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		uuid.New(), masterWalletID, txHash, txType, status, fromAddr, toAddr, amount, token, chainID)
	return err
}

func (s *Store) ListTransactions(ctx context.Context, masterWalletID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, tx_hash, tx_type, status, from_address, to_address, amount, token, chain_id, created_at FROM transactions WHERE master_wallet_id=$1 ORDER BY created_at DESC LIMIT 100`, masterWalletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var txHash, txType, status, from, to, amount, token string
		var chainID int64
		var created time.Time
		_ = rows.Scan(&id, &txHash, &txType, &status, &from, &to, &amount, &token, &chainID, &created)
		out = append(out, map[string]any{
			"id": id, "tx_hash": txHash, "type": txType, "status": status,
			"from": from, "to": to, "amount": amount, "token": token,
			"chain_id": chainID, "created_at": created,
		})
	}
	return out, nil
}

// ==================== Policies ====================

func (s *Store) CreatePolicy(ctx context.Context, masterWalletID uuid.UUID, name, policyType string, config []byte, enabled bool) (uuid.UUID, error) {
	id := uuid.New()
	_, err := s.db.Exec(ctx,
		`INSERT INTO policies (id, master_wallet_id, name, type, config, enabled) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, masterWalletID, name, policyType, config, enabled)
	return id, err
}

func (s *Store) ListPolicies(ctx context.Context, masterWalletID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, type, config::text, enabled, created_at FROM policies WHERE master_wallet_id=$1 ORDER BY created_at DESC`, masterWalletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var name, ptype, config string
		var enabled bool
		var created time.Time
		_ = rows.Scan(&id, &name, &ptype, &config, &enabled, &created)
		out = append(out, map[string]any{
			"id": id, "name": name, "type": ptype, "config": config,
			"enabled": enabled, "created_at": created,
		})
	}
	return out, nil
}

// ==================== Fee configs ====================

func (s *Store) CreateFeeConfig(ctx context.Context, masterWalletID uuid.UUID, name string, percentage, cap float64, enabled bool) (uuid.UUID, error) {
	id := uuid.New()
	_, err := s.db.Exec(ctx,
		`INSERT INTO fee_configs (id, master_wallet_id, name, percentage, cap, enabled) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, masterWalletID, name, percentage, cap, enabled)
	return id, err
}

func (s *Store) ListFeeConfigs(ctx context.Context, masterWalletID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, percentage, cap, enabled, created_at FROM fee_configs WHERE master_wallet_id=$1 ORDER BY created_at DESC`, masterWalletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var name string
		var percentage, cap float64
		var enabled bool
		var created time.Time
		_ = rows.Scan(&id, &name, &percentage, &cap, &enabled, &created)
		out = append(out, map[string]any{
			"id": id, "name": name, "percentage": percentage, "cap": cap,
			"enabled": enabled, "created_at": created,
		})
	}
	return out, nil
}

// ==================== Auto-sign rules ====================

func (s *Store) CreateAutoSignRule(ctx context.Context, masterWalletID uuid.UUID, trigger, action string, enabled bool) (uuid.UUID, error) {
	id := uuid.New()
	_, err := s.db.Exec(ctx,
		`INSERT INTO auto_sign_rules (id, master_wallet_id, trigger, action, enabled) VALUES ($1,$2,$3,$4,$5)`,
		id, masterWalletID, trigger, action, enabled)
	return id, err
}

func (s *Store) ListAutoSignRules(ctx context.Context, masterWalletID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, trigger, action, enabled, created_at FROM auto_sign_rules WHERE master_wallet_id=$1 ORDER BY created_at DESC`, masterWalletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var trigger, action string
		var enabled bool
		var created time.Time
		_ = rows.Scan(&id, &trigger, &action, &enabled, &created)
		out = append(out, map[string]any{
			"id": id, "trigger": trigger, "action": action,
			"enabled": enabled, "created_at": created,
		})
	}
	return out, nil
}

// ==================== Audit log ====================

func (s *Store) Audit(ctx context.Context, masterWalletID uuid.UUID, action, entity, entityID, severity string, details []byte) error {
	var mwID any = masterWalletID
	if masterWalletID == uuid.Nil {
		mwID = nil
	}
	_, err := s.db.Exec(ctx,
		`INSERT INTO audit_log (id, master_wallet_id, action, entity, entity_id, severity, details) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New(), mwID, action, entity, entityID, severity, details)
	return err
}

// UserRole loads the role for a user id (default "user", fail-closed if absent).
func (s *Store) UserRole(ctx context.Context, userID uuid.UUID) string {
	var role string
	err := s.db.QueryRow(ctx, `SELECT role FROM users WHERE id=$1`, userID).Scan(&role)
	if err != nil || role == "" {
		return "user"
	}
	return role
}

// ListAuditLogs returns recent audit rows for a master wallet.
func (s *Store) ListAuditLogs(ctx context.Context, masterWalletID uuid.UUID, limit int) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, action, entity, entity_id, severity, details::text, created_at FROM audit_log WHERE master_wallet_id=$1 ORDER BY created_at DESC LIMIT $2`,
		masterWalletID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var action, entity, entityID, severity, details string
		var created time.Time
		_ = rows.Scan(&id, &action, &entity, &entityID, &severity, &details, &created)
		out = append(out, map[string]any{
			"id": id, "action": action, "entity": entity, "entity_id": entityID,
			"severity": severity, "details": details, "created_at": created,
		})
	}
	return out, nil
}

// errNoRows wraps pgx detection for callers.
func ErrNoRows(err error) bool { return err == pgx.ErrNoRows }
