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
		`CREATE TABLE IF NOT EXISTS address_book (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID REFERENCES users(id) ON DELETE CASCADE, label VARCHAR(255), address VARCHAR(64) NOT NULL, chain_id BIGINT, created_at TIMESTAMPTZ DEFAULT NOW())`,
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
