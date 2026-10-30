package main

// store.go — PostgreSQL persistence for prediction markets.
//
// Trades and positions are durable and every amount is stored as NUMERIC, so a
// balance never loses precision through a float. The schema is created on
// startup when it is absent; the service refuses to serve trading traffic if
// the database is unavailable, because an unrecorded trade is a lost order.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrDuplicateTrade is returned when a transaction hash has already been
// recorded. The caller reports it as a conflict rather than a server fault.
var ErrDuplicateTrade = errors.New("this transaction hash has already been recorded")

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Store owns the database connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore connects to PostgreSQL and prepares the schema.
func NewStore(ctx context.Context, dsn string) (*Store, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is not configured")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the connection pool.
func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// migrate creates the schema when it is absent. The statements are additive, so
// running them against an existing database is a no-op.
func (s *Store) migrate(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS prediction_markets (
    id               TEXT PRIMARY KEY,
    question         TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    source           TEXT NOT NULL,
    url              TEXT NOT NULL DEFAULT '',
    image_url        TEXT NOT NULL DEFAULT '',
    end_date         BIGINT NOT NULL DEFAULT 0,
    created_at       BIGINT NOT NULL DEFAULT 0,
    updated_at       BIGINT NOT NULL DEFAULT 0,
    volume           NUMERIC(38,18) NOT NULL DEFAULT 0,
    volume_usd       NUMERIC(38,18) NOT NULL DEFAULT 0,
    open_interest    NUMERIC(38,18) NOT NULL DEFAULT 0,
    active           BOOLEAN NOT NULL DEFAULT TRUE,
    closed           BOOLEAN NOT NULL DEFAULT FALSE,
    resolved         BOOLEAN NOT NULL DEFAULT FALSE,
    outcome          TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS prediction_markets_source_idx ON prediction_markets (source);
CREATE INDEX IF NOT EXISTS prediction_markets_active_idx ON prediction_markets (active, closed);

CREATE TABLE IF NOT EXISTS prediction_outcomes (
    id          TEXT PRIMARY KEY,
    market_id   TEXT NOT NULL REFERENCES prediction_markets (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    price       NUMERIC(38,18) NOT NULL DEFAULT 0,
    probability NUMERIC(38,18) NOT NULL DEFAULT 0,
    volume      NUMERIC(38,18) NOT NULL DEFAULT 0,
    yes_volume  NUMERIC(38,18) NOT NULL DEFAULT 0,
    no_volume   NUMERIC(38,18) NOT NULL DEFAULT 0,
    active      BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE INDEX IF NOT EXISTS prediction_outcomes_market_idx ON prediction_outcomes (market_id);

CREATE TABLE IF NOT EXISTS prediction_trades (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    market_id  TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    side       TEXT NOT NULL,
    amount     NUMERIC(38,18) NOT NULL,
    price      NUMERIC(38,18) NOT NULL,
    total      NUMERIC(38,18) NOT NULL,
    chain_id   BIGINT NOT NULL DEFAULT 0,
    tx_hash    TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL,
    created_at BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS prediction_trades_user_idx ON prediction_trades (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS prediction_trades_market_idx ON prediction_trades (market_id);
-- A transaction hash identifies one on-chain fill. Enforcing uniqueness here is
-- what stops a replayed callback from double-counting a trade.
CREATE UNIQUE INDEX IF NOT EXISTS prediction_trades_txhash_idx
    ON prediction_trades (tx_hash) WHERE tx_hash <> '';

CREATE TABLE IF NOT EXISTS prediction_positions (
    user_id       TEXT NOT NULL,
    market_id     TEXT NOT NULL,
    outcome_id    TEXT NOT NULL,
    quantity      NUMERIC(38,18) NOT NULL DEFAULT 0,
    avg_price     NUMERIC(38,18) NOT NULL DEFAULT 0,
    current_value NUMERIC(38,18) NOT NULL DEFAULT 0,
    profit_loss   NUMERIC(38,18) NOT NULL DEFAULT 0,
    updated_at    BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, market_id, outcome_id)
);
`
	if _, err := s.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Markets
// ---------------------------------------------------------------------------

// UpsertMarket inserts or updates a market.
func (s *Store) UpsertMarket(ctx context.Context, m Market) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO prediction_markets
			(id, question, description, source, url, image_url, end_date,
			 created_at, updated_at, volume, volume_usd, open_interest,
			 active, closed, resolved, outcome)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (id) DO UPDATE SET
			question = EXCLUDED.question,
			description = EXCLUDED.description,
			source = EXCLUDED.source,
			url = EXCLUDED.url,
			image_url = EXCLUDED.image_url,
			end_date = EXCLUDED.end_date,
			updated_at = EXCLUDED.updated_at,
			volume = EXCLUDED.volume,
			volume_usd = EXCLUDED.volume_usd,
			open_interest = EXCLUDED.open_interest,
			active = EXCLUDED.active,
			closed = EXCLUDED.closed,
			resolved = EXCLUDED.resolved,
			outcome = EXCLUDED.outcome`,
		m.ID, m.Question, m.Description, m.Source, m.URL, m.ImageURL, m.EndDate,
		m.CreatedAt, m.UpdatedAt, m.Volume, m.VolumeUSD, m.OpenInterest,
		m.Active, m.Closed, m.Resolved, m.Outcome)
	return err
}

// UpsertOutcome inserts or updates an outcome.
func (s *Store) UpsertOutcome(ctx context.Context, o Outcome) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO prediction_outcomes
			(id, market_id, name, price, probability, volume, yes_volume, no_volume, active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			price = EXCLUDED.price,
			probability = EXCLUDED.probability,
			volume = EXCLUDED.volume,
			yes_volume = EXCLUDED.yes_volume,
			no_volume = EXCLUDED.no_volume,
			active = EXCLUDED.active`,
		o.ID, o.MarketID, o.Name, o.Price, o.Probability, o.Volume,
		o.YesVolume, o.NoVolume, o.Active)
	return err
}

// ListMarkets returns markets matching the filter.
func (s *Store) ListMarkets(ctx context.Context, source, status string, limit, offset int) ([]Market, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, question, description, source, url, image_url, end_date,
		       created_at, updated_at, volume::float8, volume_usd::float8,
		       open_interest::float8, active, closed, resolved, outcome
		FROM prediction_markets
		WHERE ($1 = '' OR source = $1)
		  AND ($2 = '' OR ($2 = 'active' AND active AND NOT closed)
		                OR ($2 = 'closed' AND closed)
		            OR ($2 = 'resolved' AND resolved))
		ORDER BY volume_usd DESC, id
		LIMIT $3 OFFSET $4`, source, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list markets: %w", err)
	}
	defer rows.Close()

	return scanMarkets(rows)
}

// ListTrendingMarkets returns the markets with the highest volume.
func (s *Store) ListTrendingMarkets(ctx context.Context, limit int) ([]Market, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, question, description, source, url, image_url, end_date,
		       created_at, updated_at, volume::float8, volume_usd::float8,
		       open_interest::float8, active, closed, resolved, outcome
		FROM prediction_markets
		WHERE active AND NOT closed
		ORDER BY volume_usd DESC, id
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list trending markets: %w", err)
	}
	defer rows.Close()

	return scanMarkets(rows)
}

// ListRecentMarkets returns the most recently created markets.
func (s *Store) ListRecentMarkets(ctx context.Context, limit int) ([]Market, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, question, description, source, url, image_url, end_date,
		       created_at, updated_at, volume::float8, volume_usd::float8,
		       open_interest::float8, active, closed, resolved, outcome
		FROM prediction_markets
		ORDER BY created_at DESC, id
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent markets: %w", err)
	}
	defer rows.Close()

	return scanMarkets(rows)
}

// GetMarket returns a market by id.
func (s *Store) GetMarket(ctx context.Context, id string) (Market, error) {
	var m Market
	err := s.pool.QueryRow(ctx, `
		SELECT id, question, description, source, url, image_url, end_date,
		       created_at, updated_at, volume::float8, volume_usd::float8,
		       open_interest::float8, active, closed, resolved, outcome
		FROM prediction_markets WHERE id = $1`, id).
		Scan(&m.ID, &m.Question, &m.Description, &m.Source, &m.URL, &m.ImageURL,
			&m.EndDate, &m.CreatedAt, &m.UpdatedAt, &m.Volume, &m.VolumeUSD,
			&m.OpenInterest, &m.Active, &m.Closed, &m.Resolved, &m.Outcome)
	if err != nil {
		return Market{}, err
	}
	return m, nil
}

// ListOutcomes returns the outcomes for a market.
func (s *Store) ListOutcomes(ctx context.Context, marketID string) ([]Outcome, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, market_id, name, price::float8, probability::float8,
		       volume::float8, yes_volume::float8, no_volume::float8, active
		FROM prediction_outcomes
		WHERE market_id = $1
		ORDER BY id`, marketID)
	if err != nil {
		return nil, fmt.Errorf("list outcomes: %w", err)
	}
	defer rows.Close()

	outcomes := make([]Outcome, 0)
	for rows.Next() {
		var o Outcome
		if err := rows.Scan(&o.ID, &o.MarketID, &o.Name, &o.Price, &o.Probability,
			&o.Volume, &o.YesVolume, &o.NoVolume, &o.Active); err != nil {
			return nil, err
		}
		outcomes = append(outcomes, o)
	}
	return outcomes, rows.Err()
}

// scanMarkets reads a market result set.
func scanMarkets(rows pgx.Rows) ([]Market, error) {
	markets := make([]Market, 0)
	for rows.Next() {
		var m Market
		if err := rows.Scan(&m.ID, &m.Question, &m.Description, &m.Source, &m.URL,
			&m.ImageURL, &m.EndDate, &m.CreatedAt, &m.UpdatedAt, &m.Volume,
			&m.VolumeUSD, &m.OpenInterest, &m.Active, &m.Closed, &m.Resolved,
			&m.Outcome); err != nil {
			return nil, err
		}
		markets = append(markets, m)
	}
	return markets, rows.Err()
}

// ---------------------------------------------------------------------------
// Trades and positions
// ---------------------------------------------------------------------------

// RecordTrade stores a trade and updates the position it opens, in one
// transaction so a trade cannot be recorded without its position.
func (s *Store) RecordTrade(ctx context.Context, trade Trade) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO prediction_trades
			(id, user_id, market_id, outcome_id, side, amount, price, total,
			 chain_id, tx_hash, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		trade.ID, trade.UserID, trade.MarketID, trade.OutcomeID, trade.Side,
		trade.Amount, trade.Price, trade.Total, trade.ChainID, trade.TxHash,
		trade.Status, trade.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateTrade
		}
		return fmt.Errorf("insert trade: %w", err)
	}

	if trade.Status == "confirmed" {
		// Quantity is the amount divided by the price paid. Every parameter is
		// cast explicitly: without the cast PostgreSQL infers the price from
		// the integer literal in NULLIF, truncates 0.185 to 0, and the
		// division yields NULL instead of a quantity.
		_, err = tx.Exec(ctx, `
			INSERT INTO prediction_positions
				(user_id, market_id, outcome_id, quantity, avg_price, updated_at)
			VALUES ($1,$2,$3,
			        $4::numeric / NULLIF($5::numeric, 0),
			        $5::numeric,
			        $6::bigint)
			ON CONFLICT (user_id, market_id, outcome_id) DO UPDATE SET
				avg_price = ((prediction_positions.quantity * prediction_positions.avg_price)
				             + (EXCLUDED.quantity * EXCLUDED.avg_price))
				            / NULLIF(prediction_positions.quantity + EXCLUDED.quantity, 0),
				quantity = prediction_positions.quantity + EXCLUDED.quantity,
				updated_at = EXCLUDED.updated_at`,
			trade.UserID, trade.MarketID, trade.OutcomeID, trade.Amount,
			trade.Price, trade.CreatedAt)
		if err != nil {
			return fmt.Errorf("update position: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ListTrades returns a user's trades, newest first.
func (s *Store) ListTrades(ctx context.Context, userID string, limit int) ([]Trade, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, market_id, outcome_id, side, amount::float8,
		       price::float8, total::float8, chain_id, tx_hash, status, created_at
		FROM prediction_trades
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list trades: %w", err)
	}
	defer rows.Close()

	trades := make([]Trade, 0)
	for rows.Next() {
		var t Trade
		if err := rows.Scan(&t.ID, &t.UserID, &t.MarketID, &t.OutcomeID, &t.Side,
			&t.Amount, &t.Price, &t.Total, &t.ChainID, &t.TxHash, &t.Status,
			&t.CreatedAt); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}
	return trades, rows.Err()
}

// ListPositions returns a user's open positions, valued at the current outcome
// price so the profit or loss is real rather than stored stale.
func (s *Store) ListPositions(ctx context.Context, userID string) ([]Position, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.user_id, p.market_id, p.outcome_id, p.quantity::float8,
		       p.avg_price::float8,
		       COALESCE(o.price, 0)::float8 AS current_price,
		       p.updated_at
		FROM prediction_positions p
		LEFT JOIN prediction_outcomes o ON o.id = p.outcome_id
		WHERE p.user_id = $1 AND p.quantity <> 0
		ORDER BY p.updated_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list positions: %w", err)
	}
	defer rows.Close()

	positions := make([]Position, 0)
	for rows.Next() {
		var (
			p            Position
			currentPrice float64
		)
		if err := rows.Scan(&p.UserID, &p.MarketID, &p.OutcomeID, &p.Quantity,
			&p.AvgPrice, &currentPrice, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.CurrentValue = p.Quantity * currentPrice
		p.ProfitLoss = (currentPrice - p.AvgPrice) * p.Quantity
		positions = append(positions, p)
	}
	return positions, rows.Err()
}

// ---------------------------------------------------------------------------
// Cache
// ---------------------------------------------------------------------------

// cacheSet stores a JSON value in Redis. A cache failure is not fatal: the
// database remains the source of truth.
func cacheSet(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	redisClient.Set(ctx, key, data, ttl)
}

// cacheGet loads a JSON value from Redis into dest.
func cacheGet(ctx context.Context, key string, dest interface{}) bool {
	data, err := redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}
	return json.Unmarshal(data, dest) == nil
}

// cacheDelete removes a key.
func cacheDelete(ctx context.Context, key string) {
	redisClient.Del(ctx, key)
}
