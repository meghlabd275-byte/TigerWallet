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
		`CREATE TABLE IF NOT EXISTS users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email VARCHAR(255) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL, created_at TIMESTAMPTZ DEFAULT NOW())`,
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

func (s *Store) GetUserByEmail(ctx context.Context, email string) (uuid.UUID, string, error) {
	var id uuid.UUID
	var hash string
	err := s.db.QueryRow(ctx, `SELECT id, password_hash FROM users WHERE email=$1`, email).Scan(&id, &hash)
	return id, hash, err
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
