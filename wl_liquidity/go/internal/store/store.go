// Package store provides PostgreSQL persistence for the standalone WL-Liquidity
// backend. It owns its own database (wl_liquidity) — independent of
// TigerWallet cloud. Tables: users (with role), liquidity_sources (DEX pools /
// aggregator config), liquidity_routes (per-source routing shares), and the
// P2P trade surface: p2p_orders / p2p_trades / p2p_messages.
//
// REAL PostgreSQL only. No fabricated pool data: the sources table starts
// empty and is populated by admin CRUD.
package store

import (
	"context"
	"errors"
	"strconv"
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
// the admin gate (RequireRole) can read it fresh on each request.
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
		`CREATE TABLE IF NOT EXISTS liquidity_sources (
			id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name         VARCHAR(255) NOT NULL,
			chain        VARCHAR(64) NOT NULL,
			dex          VARCHAR(64) NOT NULL,
			pool_address TEXT,
			token_a      VARCHAR(128) NOT NULL,
			token_b      VARCHAR(128) NOT NULL,
			reserve_a    NUMERIC NOT NULL DEFAULT 0,
			reserve_b    NUMERIC NOT NULL DEFAULT 0,
			fee_pct       NUMERIC NOT NULL DEFAULT 0,
			apy          NUMERIC NOT NULL DEFAULT 0,
			is_active    BOOLEAN NOT NULL DEFAULT TRUE,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_liq_sources_chain ON liquidity_sources(chain)`,
		`CREATE INDEX IF NOT EXISTS idx_liq_sources_tokens ON liquidity_sources(token_a, token_b)`,
		`CREATE TABLE IF NOT EXISTS liquidity_routes (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			source_id  UUID NOT NULL REFERENCES liquidity_sources(id) ON DELETE CASCADE,
			from_token VARCHAR(128) NOT NULL,
			to_token   VARCHAR(128) NOT NULL,
			share_pct  NUMERIC NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_liq_routes_source ON liquidity_routes(source_id)`,
		// P2P trade surface. An order is a buy/sell advertisement: the creator
		// fills one side (buyer_id or seller_id), the taker fills the other when
		// a trade is initiated. Both sides are nullable on the order.
		`CREATE TABLE IF NOT EXISTS p2p_orders (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			buyer_id   UUID REFERENCES users(id),
			seller_id  UUID REFERENCES users(id),
			asset      VARCHAR(128) NOT NULL,
			amount     NUMERIC NOT NULL DEFAULT 0,
			price      NUMERIC NOT NULL DEFAULT 0,
			status     VARCHAR(32) NOT NULL DEFAULT 'open',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_p2p_orders_status ON p2p_orders(status)`,
		`CREATE INDEX IF NOT EXISTS idx_p2p_orders_asset ON p2p_orders(asset)`,
		`CREATE TABLE IF NOT EXISTS p2p_trades (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			order_id   UUID NOT NULL REFERENCES p2p_orders(id) ON DELETE CASCADE,
			buyer_id   UUID NOT NULL REFERENCES users(id),
			seller_id  UUID NOT NULL REFERENCES users(id),
			status     VARCHAR(32) NOT NULL DEFAULT 'open',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_p2p_trades_order ON p2p_trades(order_id)`,
		`CREATE TABLE IF NOT EXISTS p2p_messages (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			trade_id   UUID NOT NULL REFERENCES p2p_trades(id) ON DELETE CASCADE,
			from_user  UUID NOT NULL REFERENCES users(id),
			body       TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_p2p_messages_trade ON p2p_messages(trade_id)`,
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

// ==================== Liquidity sources ====================

// Source is a persisted DEX pool / aggregator entry. Reserves + fee + apy are
// NUMERIC, scanned as strings to preserve full precision (matches wl_bots
// handling of NUMERIC columns like pnl / percentage).
type Source struct {
	ID          uuid.UUID
	Name        string
	Chain       string
	DEX         string
	PoolAddress string
	TokenA      string
	TokenB      string
	ReserveA    string
	ReserveB    string
	FeePct       string
	APY         string
	IsActive    bool
	CreatedAt   time.Time
}

func (s *Store) CreateSource(ctx context.Context, src *Source) (*Source, error) {
	var out Source
	err := s.db.QueryRow(ctx,
		`INSERT INTO liquidity_sources (name, chain, dex, pool_address, token_a, token_b, reserve_a, reserve_b, fee_pct, apy, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 RETURNING id, name, chain, dex, pool_address, token_a, token_b, reserve_a, reserve_b, fee_pct, apy, is_active, created_at`,
		src.Name, src.Chain, src.DEX, strOrNull(src.PoolAddress), src.TokenA, src.TokenB,
		src.ReserveA, src.ReserveB, src.FeePct, src.APY, src.IsActive).
		Scan(&out.ID, &out.Name, &out.Chain, &out.DEX, &out.PoolAddress, &out.TokenA, &out.TokenB,
			&out.ReserveA, &out.ReserveB, &out.FeePct, &out.APY, &out.IsActive, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListSources(ctx context.Context, chain string) ([]Source, error) {
	q := `SELECT id, name, chain, dex, pool_address, token_a, token_b, reserve_a, reserve_b, fee_pct, apy, is_active, created_at
	      FROM liquidity_sources`
	args := []any{}
	if chain != "" {
		q += ` WHERE chain=$1`
		args = append(args, chain)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSources(rows)
}

func (s *Store) GetSource(ctx context.Context, id uuid.UUID) (*Source, error) {
	var out Source
	err := s.db.QueryRow(ctx,
		`SELECT id, name, chain, dex, pool_address, token_a, token_b, reserve_a, reserve_b, fee_pct, apy, is_active, created_at
		 FROM liquidity_sources WHERE id=$1`, id).
		Scan(&out.ID, &out.Name, &out.Chain, &out.DEX, &out.PoolAddress, &out.TokenA, &out.TokenB,
			&out.ReserveA, &out.ReserveB, &out.FeePct, &out.APY, &out.IsActive, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) UpdateSource(ctx context.Context, id uuid.UUID, src *Source) (*Source, error) {
	var out Source
	err := s.db.QueryRow(ctx,
		`UPDATE liquidity_sources SET name=$1, chain=$2, dex=$3, pool_address=$4, token_a=$5, token_b=$6,
		      reserve_a=$7, reserve_b=$8, fee_pct=$9, apy=$10, is_active=$11
		 WHERE id=$12
		 RETURNING id, name, chain, dex, pool_address, token_a, token_b, reserve_a, reserve_b, fee_pct, apy, is_active, created_at`,
		src.Name, src.Chain, src.DEX, strOrNull(src.PoolAddress), src.TokenA, src.TokenB,
		src.ReserveA, src.ReserveB, src.FeePct, src.APY, src.IsActive, id).
		Scan(&out.ID, &out.Name, &out.Chain, &out.DEX, &out.PoolAddress, &out.TokenA, &out.TokenB,
			&out.ReserveA, &out.ReserveB, &out.FeePct, &out.APY, &out.IsActive, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) DeleteSource(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM liquidity_sources WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MatchingSources returns the active sources that can serve a from->to swap on
// the given chain (either direction of the token pair). Used by the real
// x*y=k quote / best_dex math.
func (s *Store) MatchingSources(ctx context.Context, from, to, chain string) ([]Source, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, chain, dex, pool_address, token_a, token_b, reserve_a, reserve_b, fee_pct, apy, is_active, created_at
		 FROM liquidity_sources
		 WHERE is_active=TRUE AND (($1='' AND $2='') OR ((token_a=$1 AND token_b=$2) OR (token_a=$2 AND token_b=$1)))
		   AND ($3='' OR chain=$3)
		 ORDER BY created_at DESC`,
		from, to, chain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSources(rows)
}

// DepthSources returns active sources for a token pair on a chain (either
// direction) for depth aggregation.
func (s *Store) DepthSources(ctx context.Context, tokenA, tokenB, chain string) ([]Source, error) {
	return s.MatchingSources(ctx, tokenA, tokenB, chain)
}

// PoolSources returns active sources for a chain (all pools). Used by /pools.
func (s *Store) PoolSources(ctx context.Context, chain string) ([]Source, error) {
	q := `SELECT id, name, chain, dex, pool_address, token_a, token_b, reserve_a, reserve_b, fee_pct, apy, is_active, created_at
	      FROM liquidity_sources WHERE is_active=TRUE`
	args := []any{}
	if chain != "" {
		q += ` AND chain=$1`
		args = append(args, chain)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSources(rows)
}

// SourceCount returns the real total + active source counts (for /health).
func (s *Store) SourceCount(ctx context.Context) (total, active int, err error) {
	err = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM liquidity_sources`).Scan(&total)
	if err != nil {
		return 0, 0, err
	}
	err = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM liquidity_sources WHERE is_active=TRUE`).Scan(&active)
	return total, active, err
}

// ==================== Liquidity routes ====================

type Route struct {
	ID         uuid.UUID
	SourceID   uuid.UUID
	FromToken  string
	ToToken    string
	SharePct   string
	CreatedAt  time.Time
}

func (s *Store) CreateRoute(ctx context.Context, sourceID uuid.UUID, from, to, sharePct string) (*Route, error) {
	var out Route
	err := s.db.QueryRow(ctx,
		`INSERT INTO liquidity_routes (source_id, from_token, to_token, share_pct)
		 VALUES ($1,$2,$3,$4)
		 RETURNING id, source_id, from_token, to_token, share_pct, created_at`,
		sourceID, from, to, sharePct).
		Scan(&out.ID, &out.SourceID, &out.FromToken, &out.ToToken, &out.SharePct, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListRoutes(ctx context.Context) ([]Route, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, source_id, from_token, to_token, share_pct, created_at
		 FROM liquidity_routes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Route{}
	for rows.Next() {
		var r Route
		if err := rows.Scan(&r.ID, &r.SourceID, &r.FromToken, &r.ToToken, &r.SharePct, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DeleteRoute(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM liquidity_routes WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ==================== P2P orders ====================

// Order is a P2P buy/sell advertisement. Amount/Price are NUMERIC scanned as
// strings (same precision handling as Source reserves). BuyerID/SellerID are
// nullable: the creator fills one side, the taker fills the other at trade
// initiation.
type Order struct {
	ID        uuid.UUID
	BuyerID   *uuid.UUID
	SellerID  *uuid.UUID
	Asset     string
	Amount    string
	Price     string
	Status    string
	CreatedAt time.Time
}

func (s *Store) CreateOrder(ctx context.Context, o *Order) (*Order, error) {
	var out Order
	err := s.db.QueryRow(ctx,
		`INSERT INTO p2p_orders (buyer_id, seller_id, asset, amount, price)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id, buyer_id, seller_id, asset, amount, price, status, created_at`,
		o.BuyerID, o.SellerID, o.Asset, o.Amount, o.Price).
		Scan(&out.ID, &out.BuyerID, &out.SellerID, &out.Asset, &out.Amount, &out.Price, &out.Status, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListOrders returns orders, optionally filtered by asset and/or status.
func (s *Store) ListOrders(ctx context.Context, asset, status string) ([]Order, error) {
	q := `SELECT id, buyer_id, seller_id, asset, amount, price, status, created_at FROM p2p_orders`
	args := []any{}
	where := ""
	if asset != "" {
		args = append(args, asset)
		where += ` AND asset=$` + strconv.Itoa(len(args))
	}
	if status != "" {
		args = append(args, status)
		where += ` AND status=$` + strconv.Itoa(len(args))
	}
	if where != "" {
		q += ` WHERE` + where[4:]
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Order{}
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.BuyerID, &o.SellerID, &o.Asset, &o.Amount, &o.Price, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) GetOrder(ctx context.Context, id uuid.UUID) (*Order, error) {
	var out Order
	err := s.db.QueryRow(ctx,
		`SELECT id, buyer_id, seller_id, asset, amount, price, status, created_at
		 FROM p2p_orders WHERE id=$1`, id).
		Scan(&out.ID, &out.BuyerID, &out.SellerID, &out.Asset, &out.Amount, &out.Price, &out.Status, &out.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (s *Store) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := s.db.Exec(ctx, `UPDATE p2p_orders SET status=$1 WHERE id=$2`, status, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ==================== P2P trades ====================

// Trade is one execution against an order. Status lifecycle:
// open -> confirmed -> released (terminal), or open/confirmed -> disputed.
type Trade struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	BuyerID   uuid.UUID
	SellerID  uuid.UUID
	Status    string
	CreatedAt time.Time
}

// CreateTrade inserts the trade and marks the parent order 'pending' in one
// transaction so an order can never be taken twice without a status change.
func (s *Store) CreateTrade(ctx context.Context, t *Trade) (*Trade, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var out Trade
	err = tx.QueryRow(ctx,
		`INSERT INTO p2p_trades (order_id, buyer_id, seller_id)
		 VALUES ($1,$2,$3)
		 RETURNING id, order_id, buyer_id, seller_id, status, created_at`,
		t.OrderID, t.BuyerID, t.SellerID).
		Scan(&out.ID, &out.OrderID, &out.BuyerID, &out.SellerID, &out.Status, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE p2p_orders SET status='pending' WHERE id=$1`, t.OrderID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) GetTrade(ctx context.Context, id uuid.UUID) (*Trade, error) {
	var out Trade
	err := s.db.QueryRow(ctx,
		`SELECT id, order_id, buyer_id, seller_id, status, created_at
		 FROM p2p_trades WHERE id=$1`, id).
		Scan(&out.ID, &out.OrderID, &out.BuyerID, &out.SellerID, &out.Status, &out.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (s *Store) UpdateTradeStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := s.db.Exec(ctx, `UPDATE p2p_trades SET status=$1 WHERE id=$2`, status, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ==================== P2P trade messages ====================

type Message struct {
	ID        uuid.UUID
	TradeID   uuid.UUID
	FromUser  uuid.UUID
	Body      string
	CreatedAt time.Time
}

func (s *Store) CreateMessage(ctx context.Context, m *Message) (*Message, error) {
	var out Message
	err := s.db.QueryRow(ctx,
		`INSERT INTO p2p_messages (trade_id, from_user, body)
		 VALUES ($1,$2,$3)
		 RETURNING id, trade_id, from_user, body, created_at`,
		m.TradeID, m.FromUser, m.Body).
		Scan(&out.ID, &out.TradeID, &out.FromUser, &out.Body, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListMessages(ctx context.Context, tradeID uuid.UUID) ([]Message, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, trade_id, from_user, body, created_at
		 FROM p2p_messages WHERE trade_id=$1 ORDER BY created_at ASC`, tradeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.TradeID, &m.FromUser, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ==================== P2P user profile ====================

// TradeStats aggregates a user's real P2P trade counters (as buyer or seller).
type TradeStats struct {
	Total     int
	Completed int
	Disputed  int
}

func (s *Store) TradeStats(ctx context.Context, userID uuid.UUID) (*TradeStats, error) {
	var st TradeStats
	err := s.db.QueryRow(ctx,
		`SELECT COUNT(*),
		        COUNT(*) FILTER (WHERE status='released'),
		        COUNT(*) FILTER (WHERE status='disputed')
		 FROM p2p_trades WHERE buyer_id=$1 OR seller_id=$1`, userID).
		Scan(&st.Total, &st.Completed, &st.Disputed)
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// ==================== helpers ====================

func scanSources(rows interface {
	Next() bool
	Scan(...any) error
}) ([]Source, error) {
	out := []Source{}
	for rows.Next() {
		var s Source
		if err := rows.Scan(&s.ID, &s.Name, &s.Chain, &s.DEX, &s.PoolAddress, &s.TokenA, &s.TokenB,
			&s.ReserveA, &s.ReserveB, &s.FeePct, &s.APY, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func strOrNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ErrNotFound is returned when a row lookup/update affects no rows.
var ErrNotFound = errors.New("record not found")
