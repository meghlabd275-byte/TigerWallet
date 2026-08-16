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

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email VARCHAR(255) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL, created_at TIMESTAMPTZ DEFAULT NOW())`,
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

// errNoRows wraps pgx detection for callers.
func ErrNoRows(err error) bool { return err == pgx.ErrNoRows }
