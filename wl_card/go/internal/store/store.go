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
			kyc_level        INT NOT NULL DEFAULT 0,
			daily_limit      NUMERIC NOT NULL DEFAULT 0,
			monthly_limit    NUMERIC NOT NULL DEFAULT 0,
			created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cards_user ON cards(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cards_hash ON cards(card_number_hash)`,
		// Upgrades for databases created before KYC/limits were added.
		`ALTER TABLE cards ADD COLUMN IF NOT EXISTS kyc_level INT NOT NULL DEFAULT 0`,
		`ALTER TABLE cards ADD COLUMN IF NOT EXISTS daily_limit NUMERIC NOT NULL DEFAULT 0`,
		`ALTER TABLE cards ADD COLUMN IF NOT EXISTS monthly_limit NUMERIC NOT NULL DEFAULT 0`,
		`CREATE TABLE IF NOT EXISTS card_transactions (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			card_id    UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
			amount     NUMERIC NOT NULL,
			merchant   TEXT,
			category   VARCHAR(64),
			tx_type    VARCHAR(32) NOT NULL DEFAULT 'PURCHASE',
			status     VARCHAR(32) NOT NULL DEFAULT 'completed',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE card_transactions ADD COLUMN IF NOT EXISTS tx_type VARCHAR(32) NOT NULL DEFAULT 'PURCHASE'`,
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
// a SHA-256 hash used for lookups. Balance and limits are NUMERIC, scanned as
// strings. KYCLevel gates the card's tier (0 = unverified).
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
	KYCLevel       int
	DailyLimit     string
	MonthlyLimit   string
	CreatedAt      time.Time
}

// cardColumns is the canonical column list for card reads. Keep in sync with
// the Card struct scan order.
const cardColumns = `id, user_id, card_number_hash, pan_encrypted, cvv_encrypted, holder_name, status, balance, currency, kyc_level, daily_limit, monthly_limit, created_at`

func (s *Store) CreateCard(ctx context.Context, card *Card) (*Card, error) {
	var out Card
	err := s.db.QueryRow(ctx,
		`INSERT INTO cards (user_id, card_number_hash, pan_encrypted, cvv_encrypted, holder_name, status, balance, currency, kyc_level, daily_limit, monthly_limit)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 RETURNING `+cardColumns,
		card.UserID, card.CardNumberHash, card.PANEncrypted, card.CVVEncrypted, card.HolderName,
		card.Status, card.Balance, card.Currency, card.KYCLevel, numericOrZero(card.DailyLimit), numericOrZero(card.MonthlyLimit)).
		Scan(&out.ID, &out.UserID, &out.CardNumberHash, &out.PANEncrypted, &out.CVVEncrypted, &out.HolderName,
			&out.Status, &out.Balance, &out.Currency, &out.KYCLevel, &out.DailyLimit, &out.MonthlyLimit, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListCards(ctx context.Context, userID uuid.UUID) ([]Card, error) {
	q := `SELECT ` + cardColumns + ` FROM cards`
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
		`SELECT `+cardColumns+` FROM cards WHERE id=$1`, id).
		Scan(&out.ID, &out.UserID, &out.CardNumberHash, &out.PANEncrypted, &out.CVVEncrypted, &out.HolderName,
			&out.Status, &out.Balance, &out.Currency, &out.KYCLevel, &out.DailyLimit, &out.MonthlyLimit, &out.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

// UpdateCardStatus sets a card's status. Valid statuses: 'pending', 'active',
// 'frozen', 'blocked', 'cancelled'. Returns ErrNotFound if the card does not
// exist.
func (s *Store) UpdateCardStatus(ctx context.Context, id uuid.UUID, status string) (*Card, error) {
	var out Card
	err := s.db.QueryRow(ctx,
		`UPDATE cards SET status=$1 WHERE id=$2
		 RETURNING `+cardColumns,
		status, id).
		Scan(&out.ID, &out.UserID, &out.CardNumberHash, &out.PANEncrypted, &out.CVVEncrypted, &out.HolderName,
			&out.Status, &out.Balance, &out.Currency, &out.KYCLevel, &out.DailyLimit, &out.MonthlyLimit, &out.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

// UpdateCardLimits sets a card's daily/monthly spending limits (NUMERIC,
// scanned as strings). Returns ErrNotFound if the card does not exist.
func (s *Store) UpdateCardLimits(ctx context.Context, id uuid.UUID, daily, monthly string) (*Card, error) {
	var out Card
	err := s.db.QueryRow(ctx,
		`UPDATE cards SET daily_limit=($1)::NUMERIC, monthly_limit=($2)::NUMERIC WHERE id=$3
		 RETURNING `+cardColumns,
		daily, monthly, id).
		Scan(&out.ID, &out.UserID, &out.CardNumberHash, &out.PANEncrypted, &out.CVVEncrypted, &out.HolderName,
			&out.Status, &out.Balance, &out.Currency, &out.KYCLevel, &out.DailyLimit, &out.MonthlyLimit, &out.CreatedAt)
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

// txColumns is the canonical column list for card_transactions reads. Keep in
// sync with the CardTransaction struct scan order.
const txColumns = `id, card_id, amount, merchant, category, tx_type, status, created_at`

type CardTransaction struct {
	ID        uuid.UUID
	CardID    uuid.UUID
	Amount    string
	Merchant  string
	Category  string
	TxType    string
	Status    string
	CreatedAt time.Time
}

func (s *Store) ListTransactions(ctx context.Context, cardID uuid.UUID) ([]CardTransaction, error) {
	q := `SELECT ` + txColumns + ` FROM card_transactions`
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
		if err := rows.Scan(&t.ID, &t.CardID, &t.Amount, &t.Merchant, &t.Category, &t.TxType, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTransaction returns a single transaction scoped to its card. Returns
// ErrNotFound if no such transaction exists on that card.
func (s *Store) GetTransaction(ctx context.Context, cardID, txID uuid.UUID) (*CardTransaction, error) {
	var t CardTransaction
	err := s.db.QueryRow(ctx,
		`SELECT `+txColumns+` FROM card_transactions WHERE card_id=$1 AND id=$2`,
		cardID, txID).
		Scan(&t.ID, &t.CardID, &t.Amount, &t.Merchant, &t.Category, &t.TxType, &t.Status, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// RecordTransaction inserts a card transaction AND debits/credits the card
// balance atomically in a single DB transaction. A negative effect (debit
// exceeding balance) aborts the whole operation — the card balance is the
// source of truth and never goes negative. txType classifies the row
// ('PURCHASE', 'TOP_UP', ...); empty defaults to 'PURCHASE'.
func (s *Store) RecordTransaction(ctx context.Context, cardID uuid.UUID, amount, merchant, category, direction, txType string) (*CardTransaction, *Card, error) {
	if txType == "" {
		txType = "PURCHASE"
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Lock the card row for the duration of the transaction.
	var card Card
	err = tx.QueryRow(ctx,
		`SELECT `+cardColumns+` FROM cards WHERE id=$1 FOR UPDATE`, cardID).
		Scan(&card.ID, &card.UserID, &card.CardNumberHash, &card.PANEncrypted, &card.CVVEncrypted, &card.HolderName,
			&card.Status, &card.Balance, &card.Currency, &card.KYCLevel, &card.DailyLimit, &card.MonthlyLimit, &card.CreatedAt)
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
		`INSERT INTO card_transactions (card_id, amount, merchant, category, tx_type, status)
		 VALUES ($1,$2,$3,$4,$5,'completed')
		 RETURNING `+txColumns,
		cardID, amount, strOrNull(merchant), strOrNull(category), txType).
		Scan(&txn.ID, &txn.CardID, &txn.Amount, &txn.Merchant, &txn.Category, &txn.TxType, &txn.Status, &txn.CreatedAt)
	if err != nil {
		return nil, nil, err
	}
	card.Balance = newBalance
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return &txn, &card, nil
}

// TopUp credits a card balance and records a tx_type='TOP_UP' transaction
// atomically (same invariants as RecordTransaction).
func (s *Store) TopUp(ctx context.Context, cardID uuid.UUID, amount string) (*CardTransaction, *Card, error) {
	return s.RecordTransaction(ctx, cardID, amount, "Top-up", "TOP_UP", "credit", "TOP_UP")
}

// ==================== Rates (stub) ====================

// Rates returns static card funding rates (USD per unit). Stub: hardcoded
// until a real price oracle is wired in.
func (s *Store) Rates(ctx context.Context) map[string]float64 {
	return map[string]float64{
		"BTC":  67000,
		"ETH":  3500,
		"BNB":  600,
		"USDT": 1,
		"USDC": 1,
	}
}

// ==================== Admin stats ====================

// Stats holds aggregate counts/volumes for the admin dashboard.
type Stats struct {
	TotalUsers        int    `json:"total_users"`
	TotalCards        int    `json:"total_cards"`
	PendingCards      int    `json:"pending_cards"`
	ActiveCards       int    `json:"active_cards"`
	FrozenCards       int    `json:"frozen_cards"`
	BlockedCards      int    `json:"blocked_cards"`
	CancelledCards    int    `json:"cancelled_cards"`
	TotalTransactions int    `json:"total_transactions"`
	TotalVolume       string `json:"total_volume"`
	TotalBalance      string `json:"total_balance"`
}

// Stats aggregates real counts/volumes across users, cards, and transactions.
func (s *Store) Stats(ctx context.Context) (*Stats, error) {
	var st Stats
	err := s.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM cards),
			(SELECT COUNT(*) FROM cards WHERE status='pending'),
			(SELECT COUNT(*) FROM cards WHERE status='active'),
			(SELECT COUNT(*) FROM cards WHERE status='frozen'),
			(SELECT COUNT(*) FROM cards WHERE status='blocked'),
			(SELECT COUNT(*) FROM cards WHERE status='cancelled'),
			(SELECT COUNT(*) FROM card_transactions),
			(SELECT COALESCE(SUM(amount), 0) FROM card_transactions),
			(SELECT COALESCE(SUM(balance), 0) FROM cards)`).
		Scan(&st.TotalUsers, &st.TotalCards, &st.PendingCards, &st.ActiveCards,
			&st.FrozenCards, &st.BlockedCards, &st.CancelledCards,
			&st.TotalTransactions, &st.TotalVolume, &st.TotalBalance)
	if err != nil {
		return nil, err
	}
	return &st, nil
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
			&c.Status, &c.Balance, &c.Currency, &c.KYCLevel, &c.DailyLimit, &c.MonthlyLimit, &c.CreatedAt); err != nil {
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

// numericOrZero maps an empty NUMERIC string to "0" so inserts never send an
// empty value to a NUMERIC column.
func numericOrZero(s string) string {
	if s == "" {
		return "0"
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
