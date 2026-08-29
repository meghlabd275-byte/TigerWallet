// Package store provides PostgreSQL persistence for the standalone WL-UserWallet
// backend. It owns its own database (wl_userwallet) — independent of TigerWallet
// cloud. Tables: users, wallets (encrypted_seed), transactions, address_book.
package store

import (
	"context"
	"time"

	"github.com/google/uuid"
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
		`CREATE TABLE IF NOT EXISTS users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email VARCHAR(255) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL, scopes TEXT[] NOT NULL DEFAULT '{}'::text[], is_active BOOLEAN NOT NULL DEFAULT true, created_at TIMESTAMPTZ DEFAULT NOW())`,
		// Backfill the scopes + is_active columns on existing DBs (added for the scoped-role taxonomy + admin oversight).
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS scopes TEXT[] NOT NULL DEFAULT '{}'::text[]`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true`,
		`CREATE TABLE IF NOT EXISTS wallets (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, label VARCHAR(255), address VARCHAR(64) NOT NULL, encrypted_seed TEXT NOT NULL, chain_id BIGINT DEFAULT 1, wl_client_id VARCHAR(64), created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_wallets_user ON wallets(user_id)`,
		`CREATE TABLE IF NOT EXISTS transactions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), wallet_id UUID REFERENCES wallets(id) ON DELETE CASCADE, tx_hash VARCHAR(80), tx_type VARCHAR(32), status VARCHAR(32), from_address VARCHAR(64), to_address VARCHAR(64), amount VARCHAR(64), token VARCHAR(64), chain_id BIGINT, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_tx_wallet ON transactions(wallet_id)`,
		`CREATE TABLE IF NOT EXISTS address_book (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, name VARCHAR(255) NOT NULL, address VARCHAR(64) NOT NULL, chain_id BIGINT, note TEXT, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_address_book_user ON address_book(user_id)`,
		`CREATE TABLE IF NOT EXISTS devices (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, name VARCHAR(255) NOT NULL, device_type VARCHAR(64) NOT NULL, status VARCHAR(32) DEFAULT 'offline', last_sync TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_devices_user ON devices(user_id)`,
		`CREATE TABLE IF NOT EXISTS erc20_token_cache (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), chain_id BIGINT NOT NULL, contract VARCHAR(64) NOT NULL, symbol VARCHAR(64), name VARCHAR(255), decimals INT, UNIQUE(chain_id, contract))`,
		`CREATE TABLE IF NOT EXISTS nft_cache (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), chain_id BIGINT NOT NULL, contract VARCHAR(64) NOT NULL, token_id VARCHAR(128) NOT NULL, owner VARCHAR(64), name VARCHAR(255), description TEXT, image_url TEXT, collection VARCHAR(255), standard VARCHAR(16) DEFAULT 'ERC-721', UNIQUE(chain_id, contract, token_id))`,
	}
	        extraStmts := []string{
                `CREATE TABLE IF NOT EXISTS price_alerts (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, symbol VARCHAR(32) NOT NULL, target_price NUMERIC(34,18) NOT NULL, direction VARCHAR(8) NOT NULL DEFAULT 'above', enabled BOOLEAN NOT NULL DEFAULT true, created_at TIMESTAMPTZ DEFAULT NOW())`,
                `CREATE TABLE IF NOT EXISTS p2p_adverts (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, side VARCHAR(8) NOT NULL DEFAULT 'sell', asset VARCHAR(64) NOT NULL, price NUMERIC(34,18) NOT NULL, created_at TIMESTAMPTZ DEFAULT NOW())`,
                `CREATE TABLE IF NOT EXISTS p2p_orders (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), advert_id UUID REFERENCES p2p_adverts(id) ON DELETE CASCADE, buyer_id UUID REFERENCES users(id) ON DELETE CASCADE, amount NUMERIC(34,18) NOT NULL, status VARCHAR(32) NOT NULL DEFAULT 'open', created_at TIMESTAMPTZ DEFAULT NOW())`,
                `CREATE TABLE IF NOT EXISTS dao_proposals (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), title VARCHAR(255) NOT NULL, description TEXT, votes_for BIGINT DEFAULT 0, votes_against BIGINT DEFAULT 0, created_at TIMESTAMPTZ DEFAULT NOW())`,
                `CREATE TABLE IF NOT EXISTS dao_votes (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), proposal_id UUID REFERENCES dao_proposals(id) ON DELETE CASCADE, voter_id UUID REFERENCES users(id) ON DELETE CASCADE, support BOOLEAN NOT NULL, created_at TIMESTAMPTZ DEFAULT NOW())`,
                `CREATE TABLE IF NOT EXISTS launchpool (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, amount NUMERIC(34,18) NOT NULL, created_at TIMESTAMPTZ DEFAULT NOW())`,
                `CREATE TABLE IF NOT EXISTS token_sales (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(255) UNIQUE NOT NULL, symbol VARCHAR(32), supply NUMERIC(34,18) NOT NULL, raised NUMERIC(34,18) DEFAULT 0, created_at TIMESTAMPTZ DEFAULT NOW())`,
                `CREATE TABLE IF NOT EXISTS token_sale_entry (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), sale_id UUID REFERENCES token_sales(id) ON DELETE CASCADE, user_id UUID REFERENCES users(id) ON DELETE CASCADE, amount NUMERIC(34,18) NOT NULL, created_at TIMESTAMPTZ DEFAULT NOW())`,
                `CREATE TABLE IF NOT EXISTS token_approvals (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, token VARCHAR(64) NOT NULL, spender VARCHAR(64) NOT NULL, allowance NUMERIC(78,0) NOT NULL DEFAULT 0, created_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE(user_id, token, spender))`,
                `CREATE TABLE IF NOT EXISTS kyc_records (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, status VARCHAR(32) NOT NULL DEFAULT 'pending', full_name VARCHAR(255), doc_type VARCHAR(64), doc_number VARCHAR(128), session JSONB, created_at TIMESTAMPTZ DEFAULT NOW())`,
                `CREATE TABLE IF NOT EXISTS card_accounts (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, balance NUMERIC(34,18) NOT NULL DEFAULT 0, created_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE(user_id))`,
                `CREATE TABLE IF NOT EXISTS card_transactions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, amount NUMERIC(34,18) NOT NULL, merchant VARCHAR(255), status VARCHAR(32) NOT NULL DEFAULT 'settled', created_at TIMESTAMPTZ DEFAULT NOW())`,
                `CREATE TABLE IF NOT EXISTS fees (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), wallet_id UUID REFERENCES wallets(id) ON DELETE CASCADE, fee_type VARCHAR(32) NOT NULL DEFAULT 'send', amount NUMERIC(34,18) NOT NULL, tx_hash VARCHAR(80), created_at TIMESTAMPTZ DEFAULT NOW())`,
                `CREATE TABLE IF NOT EXISTS margin_positions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, pair VARCHAR(32) NOT NULL, side VARCHAR(8) NOT NULL, size NUMERIC(34,18) NOT NULL, leverage INT NOT NULL DEFAULT 1, pnl NUMERIC(34,18) DEFAULT 0, status VARCHAR(32) NOT NULL DEFAULT 'open', created_at TIMESTAMPTZ DEFAULT NOW())`,
                `CREATE TABLE IF NOT EXISTS perp_positions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, pair VARCHAR(32) NOT NULL, side VARCHAR(8) NOT NULL, size NUMERIC(34,18) NOT NULL, leverage INT NOT NULL DEFAULT 1, entry NUMERIC(34,18) DEFAULT 0, pnl NUMERIC(34,18) DEFAULT 0, status VARCHAR(32) NOT NULL DEFAULT 'open', created_at TIMESTAMPTZ DEFAULT NOW())`,
        }
        stmts = append(stmts, extraStmts...)
for _, st := range stmts {
		if _, err := s.db.Exec(ctx, st); err != nil {
			return err
		}
	}
	return nil
}

type Wallet struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Label         string
	Address       string
	EncryptedSeed string
	ChainID       int64
	WLClientID    string
	CreatedAt     time.Time
}

func (s *Store) CreateWallet(ctx context.Context, userID uuid.UUID, label, address, encSeed string, chainID int64, wlClientID string) (*Wallet, error) {
	id := uuid.New()
	var createdAt time.Time
	err := s.db.QueryRow(ctx,
		`INSERT INTO wallets (id, user_id, label, address, encrypted_seed, chain_id, wl_client_id) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING created_at`,
		id, userID, label, address, encSeed, chainID, wlClientID).Scan(&createdAt)
	if err != nil {
		return nil, err
	}
	return &Wallet{ID: id, UserID: userID, Label: label, Address: address, EncryptedSeed: encSeed, ChainID: chainID, CreatedAt: createdAt}, nil
}

func (s *Store) ListWallets(ctx context.Context, userID uuid.UUID) ([]Wallet, error) {
	rows, err := s.db.Query(ctx, `SELECT id, user_id, label, address, encrypted_seed, chain_id, created_at FROM wallets WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Wallet{}
	for rows.Next() {
		var w Wallet
		_ = rows.Scan(&w.ID, &w.UserID, &w.Label, &w.Address, &w.EncryptedSeed, &w.ChainID, &w.CreatedAt)
		out = append(out, w)
	}
	return out, nil
}

func (s *Store) GetWallet(ctx context.Context, id uuid.UUID) (*Wallet, error) {
	var w Wallet
	err := s.db.QueryRow(ctx, `SELECT id, user_id, label, address, encrypted_seed, chain_id, created_at FROM wallets WHERE id=$1`, id).
		Scan(&w.ID, &w.UserID, &w.Label, &w.Address, &w.EncryptedSeed, &w.ChainID, &w.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *Store) CreateTransaction(ctx context.Context, walletID uuid.UUID, txHash, txType, status, fromAddr, toAddr, amount, token string, chainID int64) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO transactions (id, wallet_id, tx_hash, tx_type, status, from_address, to_address, amount, token, chain_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		uuid.New(), walletID, txHash, txType, status, fromAddr, toAddr, amount, token, chainID)
	return err
}

func (s *Store) ListTransactions(ctx context.Context, walletID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `SELECT id, tx_hash, tx_type, status, from_address, to_address, amount, token, chain_id, created_at FROM transactions WHERE wallet_id=$1 ORDER BY created_at DESC LIMIT 100`, walletID)
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
		out = append(out, map[string]any{"id": id, "tx_hash": txHash, "type": txType, "status": status, "from": from, "to": to, "amount": amount, "token": token, "chain_id": chainID, "created_at": created})
	}
	return out, nil
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (uuid.UUID, error) {
	id := uuid.New()
	_, err := s.db.Exec(ctx, `INSERT INTO users (id, email, password_hash) VALUES ($1,$2,$3)`, id, email, passwordHash)
	return id, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (uuid.UUID, string, []string, error) {
	var id uuid.UUID
	var hash string
	var scopes []string
	err := s.db.QueryRow(ctx, `SELECT id, password_hash, scopes FROM users WHERE email=$1`, email).Scan(&id, &hash, &scopes)
	return id, hash, scopes, err
}

// UpdateUserScopes replaces a user's scoped-admin roles (canonical taxonomy).
// The WL client owner (wl_client scope) uses this to grant wallet_admin etc.
// The scopes are issued in the JWT at login and enforced by HasScope.
func (s *Store) UpdateUserScopes(ctx context.Context, id uuid.UUID, scopes []string) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET scopes=$1 WHERE id=$2`, scopes, id)
	return err
}

// ==================== Admin oversight (wallet_admin / wl_client scope) ====================
// These methods power the WL client's wallet-management admin panel. They are
// read-only or status-only (NO fund movement — withdrawals stay two-party
// gated). A wallet_admin scoped admin can view all wallets/users in the
// tenancy + suspend a user; the WL client owner (wl_client) can do the same.

// AdminUserRow is a user record WITHOUT the password_hash (never exposed).
type AdminUserRow struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Scopes    []string  `json:"scopes"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// ListAllUsers returns all users in the tenancy (admin oversight). Password
// hashes are never selected — only id/email/scopes/is_active/created_at.
func (s *Store) ListAllUsers(ctx context.Context) ([]AdminUserRow, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, email, scopes, is_active, created_at FROM users ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdminUserRow{}
	for rows.Next() {
		var u AdminUserRow
		if err := rows.Scan(&u.ID, &u.Email, &u.Scopes, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// SetUserActive activates/suspends a user. A suspended user's JWT still
// validates (stateless HS256) but every wallet route re-checks is_active via
// RequireActiveUser middleware — so a suspended user is immediately locked out.
func (s *Store) SetUserActive(ctx context.Context, id uuid.UUID, active bool) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET is_active=$1 WHERE id=$2`, active, id)
	return err
}

// GetUserActive returns the user's is_active flag (for the RequireActiveUser
// middleware that reads it on every wallet request).
func (s *Store) GetUserActive(ctx context.Context, id uuid.UUID) (bool, error) {
	var active bool
	err := s.db.QueryRow(ctx, `SELECT is_active FROM users WHERE id=$1`, id).Scan(&active)
	if err != nil {
		return false, err
	}
	return active, nil
}

// ListAllWallets returns all wallets in the tenancy (admin oversight). The
// encrypted_seed is NEVER selected — only id/user_id/label/address/chain_id.
func (s *Store) ListAllWallets(ctx context.Context) ([]Wallet, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, label, address, chain_id, wl_client_id, created_at
		 FROM wallets ORDER BY created_at DESC LIMIT 1000`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Wallet{}
	for rows.Next() {
		var w Wallet
		if err := rows.Scan(&w.ID, &w.UserID, &w.Label, &w.Address, &w.ChainID, &w.WLClientID, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// CountWalletsByUser returns the number of wallets owned by a user (admin
// oversight dashboard stat).
func (s *Store) CountWalletsByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	var n int64
	err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM wallets WHERE user_id=$1`, userID).Scan(&n)
	return n, err
}

// Pool exposes the underlying pgxpool for handlers that issue direct queries
// (address book, devices) mirroring the canonical wallet_api pattern.
func (s *Store) Pool() *pgxpool.Pool { return s.db }

// ==================== Address book ====================

// AddressBookRecord mirrors the address_book table.
type AddressBookRecord struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	ChainID   int64     `json:"chain_id"`
	Note      string    `json:"note"`
	CreatedAt int64     `json:"created_at"`
}

func (s *Store) ListContacts(ctx context.Context, userID uuid.UUID) ([]AddressBookRecord, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, name, address, chain_id, COALESCE(note,''), extract(epoch from created_at)::bigint FROM address_book WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AddressBookRecord{}
	for rows.Next() {
		var r AddressBookRecord
		if err := rows.Scan(&r.ID, &r.UserID, &r.Name, &r.Address, &r.ChainID, &r.Note, &r.CreatedAt); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Store) CreateContact(ctx context.Context, userID uuid.UUID, name, address string, chainID int64, note string) (uuid.UUID, error) {
	id := uuid.New()
	_, err := s.db.Exec(ctx,
		`INSERT INTO address_book (id, user_id, name, address, chain_id, note) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, userID, name, address, chainID, note)
	return id, err
}

func (s *Store) UpdateContact(ctx context.Context, userID, id uuid.UUID, name, address string, chainID int64, note string) (int64, error) {
	tag, err := s.db.Exec(ctx,
		`UPDATE address_book SET name=$1, address=$2, chain_id=$3, note=$4 WHERE id=$5 AND user_id=$6`,
		name, address, chainID, note, id, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *Store) DeleteContact(ctx context.Context, userID, id uuid.UUID) (int64, error) {
	tag, err := s.db.Exec(ctx, `DELETE FROM address_book WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ==================== Devices ====================

// DeviceRecord is a connected device entry for multi-device sync.
type DeviceRecord struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	LastSync  int64  `json:"lastSync"`
	CreatedAt int64  `json:"createdAt"`
}

func (s *Store) ListDevices(ctx context.Context, userID uuid.UUID) ([]DeviceRecord, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, device_type, status, COALESCE(extract(epoch from last_sync)::bigint, 0), extract(epoch from created_at)::bigint FROM devices WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DeviceRecord{}
	for rows.Next() {
		var r DeviceRecord
		if err := rows.Scan(&r.ID, &r.Name, &r.Type, &r.Status, &r.LastSync, &r.CreatedAt); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Store) RegisterDevice(ctx context.Context, userID uuid.UUID, name, deviceType string) (uuid.UUID, error) {
	id := uuid.New()
	_, err := s.db.Exec(ctx,
		`INSERT INTO devices (id, user_id, name, device_type, status) VALUES ($1,$2,$3,$4,'offline')`,
		id, userID, name, deviceType)
	return id, err
}

func (s *Store) SyncDevice(ctx context.Context, userID, id uuid.UUID) (int64, error) {
	tag, err := s.db.Exec(ctx,
		`UPDATE devices SET status='online', last_sync=NOW() WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *Store) DeleteDevice(ctx context.Context, userID, id uuid.UUID) (int64, error) {
	tag, err := s.db.Exec(ctx, `DELETE FROM devices WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
