// Package store provides PostgreSQL persistence for the standalone WL-Card
// backend. It owns its own database (wl_card) — independent of TigerWallet
// cloud. Tables: users (with role), cards (PAN/CVV encrypted at rest via
// AES-GCM; only a hash of the card number is queryable), card_transactions
// (linked to cards).
//
// REAL PostgreSQL only. No fake balances/transactions — real DB rows only.
package store

import (
	"context"
	"errors"
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
	if err := s.Migrate(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() { s.db.Close() }

func (s *Store) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

// Migrate creates the schema. users carries a role column (default 'user') so
// the admin gate (RequireRole) can read it fresh on each request. cards stores
// the PAN/CVV only in AES-GCM-encrypted form (pan_encrypted / cvv_encrypted);
// a SHA-256 hash of the card number (card_number_hash) is the queryable
// identifier. Balance is NUMERIC (scanned as string for precision).
func (s *Store) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email         VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role          VARCHAR(32) NOT NULL DEFAULT 'user',
			is_active     BOOLEAN NOT NULL DEFAULT TRUE,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS cards (
			id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			card_number_hash TEXT NOT NULL,
			pan_encrypted    TEXT NOT NULL,
			cvv_encrypted    TEXT NOT NULL,
			holder_name      TEXT NOT NULL,
			status           VARCHAR(32) NOT NULL DEFAULT 'active',
			balance          NUMERIC NOT NULL DEFAULT 0,
			currency         VARCHAR(8) NOT NULL DEFAULT 'USD',
			created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cards_user ON cards(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cards_hash ON cards(card_number_hash)`,
		`CREATE TABLE IF NOT EXISTS card_transactions (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			card_id    UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
			amount     NUMERIC NOT NULL,
			merchant   TEXT,
			category   VARCHAR(64),
			status     VARCHAR(32) NOT NULL DEFAULT 'completed',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_card_tx_card ON card_transactions(card_id)`,
	}
	for _, st := range stmts {
		if _, err := s.db.Exec(ctx, st); err != nil {
			return err
		}
	}
	return nil
}

// ==================== Users ====================

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         string
	IsActive     bool
	CreatedAt    time.Time
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, role string) (*User, error) {
	if role == "" {
		role = "user"
	}
	var u User
	err := s.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1,$2,$3)
		 RETURNING id, email, password_hash, role, is_active, created_at`,
		email, passwordHash, role).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, role, is_active, created_at FROM users WHERE email=$1`,
		email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, role, is_active, created_at FROM users WHERE id=$1`,
		id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ==================== Cards ====================

// Card is a persisted WL-branded crypto card. PAN/CVV are stored ONLY in
// AES-GCM-encrypted form (pan_encrypted / cvv_encrypted); card_number_hash is
// a SHA-256 hash used for lookups. Balance is NUMERIC, scanned as a string.
type Card struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	CardNumberHash string
	PANEncrypted   string
	CVVEncrypted   string
	HolderName     string
	Status         string
	Balance        string
	Currency       string
	CreatedAt      time.Time
}

func (s *Store) CreateCard(ctx context.Context, card *Card) (*Card, error) {
	var out Card
	err := s.db.QueryRow(ctx,
		`INSERT INTO cards (user_id, card_number_hash, pan_encrypted, cvv_encrypted, holder_name, status, balance, currency)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id, user_id, card_number_hash, pan_encrypted, cvv_encrypted, holder_name, status, balance, currency, created_at`,
		card.UserID, card.CardNumberHash, card.PANEncrypted, card.CVVEncrypted, card.HolderName,
		card.Status, card.Balance, card.Currency).
		Scan(&out.ID, &out.UserID, &out.CardNumberHash, &out.PANEncrypted, &out.CVVEncrypted, &out.HolderName,
			&out.Status, &out.Balance, &out.Currency, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListCards(ctx context.Context, userID uuid.UUID) ([]Card, error) {
	q := `SELECT id, user_id, card_number_hash, pan_encrypted, cvv_encrypted, holder_name, status, balance, currency, created_at
	      FROM cards`
	args := []any{}
	if userID != uuid.Nil {
		q += ` WHERE user_id=$1`
		args = append(args, userID)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCards(rows)
}

func (s *Store) GetCard(ctx context.Context, id uuid.UUID) (*Card, error) {
	var out Card
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, card_number_hash, pan_encrypted, cvv_encrypted, holder_name, status, balance, currency, created_at
		 FROM cards WHERE id=$1`, id).
		Scan(&out.ID, &out.UserID, &out.CardNumberHash, &out.PANEncrypted, &out.CVVEncrypted, &out.HolderName,
			&out.Status, &out.Balance, &out.Currency, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateCardStatus sets a card's status (freeze/unfreeze). Returns ErrNotFound
// if the card does not exist.
func (s *Store) UpdateCardStatus(ctx context.Context, id uuid.UUID, status string) (*Card, error) {
	var out Card
	err := s.db.QueryRow(ctx,
		`UPDATE cards SET status=$1 WHERE id=$2
		 RETURNING id, user_id, card_number_hash, pan_encrypted, cvv_encrypted, holder_name, status, balance, currency, created_at`,
		status, id).
		Scan(&out.ID, &out.UserID, &out.CardNumberHash, &out.PANEncrypted, &out.CVVEncrypted, &out.HolderName,
			&out.Status, &out.Balance, &out.Currency, &out.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

// CardCount returns the real total + active card counts (for /health).
func (s *Store) CardCount(ctx context.Context) (total, active int, err error) {
	err = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM cards`).Scan(&total)
	if err != nil {
		return 0, 0, err
	}
	err = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM cards WHERE status='active'`).Scan(&active)
	return total, active, err
}

// ==================== Card transactions ====================

type CardTransaction struct {
	ID         uuid.UUID
	CardID     uuid.UUID
	Amount     string
	Merchant   string
	Category   string
	Status     string
	CreatedAt  time.Time
}

func (s *Store) ListTransactions(ctx context.Context, cardID uuid.UUID) ([]CardTransaction, error) {
	q := `SELECT id, card_id, amount, merchant, category, status, created_at
	      FROM card_transactions`
	args := []any{}
	if cardID != uuid.Nil {
		q += ` WHERE card_id=$1`
		args = append(args, cardID)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CardTransaction{}
	for rows.Next() {
		var t CardTransaction
		if err := rows.Scan(&t.ID, &t.CardID, &t.Amount, &t.Merchant, &t.Category, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// RecordTransaction inserts a card transaction AND debits/credits the card
// balance atomically in a single DB transaction. A negative effect (debit
// exceeding balance) aborts the whole operation — the card balance is the
// source of truth and never goes negative.
func (s *Store) RecordTransaction(ctx context.Context, cardID uuid.UUID, amount, merchant, category, direction string) (*CardTransaction, *Card, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Lock the card row for the duration of the transaction.
	var card Card
	err = tx.QueryRow(ctx,
		`SELECT id, user_id, card_number_hash, pan_encrypted, cvv_encrypted, holder_name, status, balance, currency, created_at
		 FROM cards WHERE id=$1 FOR UPDATE`, cardID).
		Scan(&card.ID, &card.UserID, &card.CardNumberHash, &card.PANEncrypted, &card.CVVEncrypted, &card.HolderName,
			&card.Status, &card.Balance, &card.Currency, &card.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	if card.Status != "active" {
		return nil, &card, ErrCardNotActive
	}

	// amount is always positive; direction decides debit vs credit.
	effect := amount
	if direction == "debit" {
		effect = "-" + amount
	}
	// Apply the effect; reject if it would drive the balance negative.
	var newBalance string
	err = tx.QueryRow(ctx,
		`UPDATE cards SET balance = balance + ($1)::NUMERIC WHERE id=$2
		 RETURNING balance`, effect, cardID).Scan(&newBalance)
	if err != nil {
		return nil, nil, err
	}
	if balanceNegative(newBalance) {
		return nil, &card, ErrInsufficientFunds
	}

	var txn CardTransaction
	err = tx.QueryRow(ctx,
		`INSERT INTO card_transactions (card_id, amount, merchant, category, status)
		 VALUES ($1,$2,$3,$4,'completed')
		 RETURNING id, card_id, amount, merchant, category, status, created_at`,
		cardID, amount, strOrNull(merchant), strOrNull(category)).
		Scan(&txn.ID, &txn.CardID, &txn.Amount, &txn.Merchant, &txn.Category, &txn.Status, &txn.CreatedAt)
	if err != nil {
		return nil, nil, err
	}
	card.Balance = newBalance
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return &txn, &card, nil
}

// ==================== helpers ====================

func scanCards(rows interface {
	Next() bool
	Scan(...any) error
}) ([]Card, error) {
	out := []Card{}
	for rows.Next() {
		var c Card
		if err := rows.Scan(&c.ID, &c.UserID, &c.CardNumberHash, &c.PANEncrypted, &c.CVVEncrypted, &c.HolderName,
			&c.Status, &c.Balance, &c.Currency, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func strOrNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func balanceNegative(s string) bool {
	for _, r := range s {
		if r == '-' {
			return true
		}
		if r >= '0' && r <= '9' {
			return false
		}
	}
	return false
}

// ErrNotFound is returned when a row lookup/update affects no rows.
var ErrNotFound = errors.New("record not found")

// ErrCardNotActive is returned when a transaction targets a frozen/inactive card.
var ErrCardNotActive = errors.New("card not active")

// ErrInsufficientFunds is returned when a debit would drive the balance negative.
var ErrInsufficientFunds = errors.New("insufficient funds")
