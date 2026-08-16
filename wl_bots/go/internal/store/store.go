// Package store provides PostgreSQL persistence for the standalone WL-Bots
// backend. It owns its own database (wl_bots) — independent of TigerWallet
// cloud. Tables: users, bots, bot_executions, subscriptions, fee_configs,
// api_keys, bot_logs.
package store

import (
	"context"
	"encoding/json"
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

func (s *Store) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email         VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role          VARCHAR(32) NOT NULL DEFAULT 'client',
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS bots (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name       VARCHAR(255) NOT NULL,
			bot_type   VARCHAR(64) NOT NULL,
			status     VARCHAR(32) NOT NULL DEFAULT 'stopped',
			config     JSONB NOT NULL DEFAULT '{}'::jsonb,
			exchange   VARCHAR(64),
			pair       VARCHAR(32),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bots_user ON bots(user_id)`,
		`CREATE TABLE IF NOT EXISTS bot_executions (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			bot_id     UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
			status     VARCHAR(32) NOT NULL,
			pnl        NUMERIC(20,8) NOT NULL DEFAULT 0,
			started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			ended_at   TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_exec_bot ON bot_executions(bot_id)`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			tier       VARCHAR(32) NOT NULL,
			started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_subs_user ON subscriptions(user_id)`,
		`CREATE TABLE IF NOT EXISTS fee_configs (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name       VARCHAR(255) NOT NULL,
			percentage NUMERIC(10,4) NOT NULL DEFAULT 0,
			enabled    BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			exchange         VARCHAR(64) NOT NULL,
			api_key_encrypted TEXT NOT NULL,
			enabled          BOOLEAN NOT NULL DEFAULT TRUE,
			created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_apikeys_user ON api_keys(user_id)`,
		`CREATE TABLE IF NOT EXISTS bot_logs (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			bot_id     UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
			level      VARCHAR(16) NOT NULL DEFAULT 'info',
			message    TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_bot ON bot_logs(bot_id)`,
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
	CreatedAt    time.Time
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, role string) (*User, error) {
	if role == "" {
		role = "client"
	}
	var u User
	err := s.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1,$2,$3)
		 RETURNING id, email, password_hash, role, created_at`,
		email, passwordHash, role).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, role, created_at FROM users WHERE email=$1`,
		email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ==================== Bots ====================

type Bot struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	BotType   string
	Status    string
	Config    map[string]any
	Exchange  string
	Pair      string
	CreatedAt time.Time
}

func (s *Store) CreateBot(ctx context.Context, userID uuid.UUID, name, botType, exchange, pair string, config map[string]any) (*Bot, error) {
	cfgJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var b Bot
	err = s.db.QueryRow(ctx,
		`INSERT INTO bots (user_id, name, bot_type, status, config, exchange, pair)
		 VALUES ($1,$2,$3,'stopped',$4,$5,$6)
		 RETURNING id, user_id, name, bot_type, status, config, exchange, pair, created_at`,
		userID, name, botType, cfgJSON, strOrNull(exchange), strOrNull(pair)).
		Scan(&b.ID, &b.UserID, &b.Name, &b.BotType, &b.Status, &b.Config, &b.Exchange, &b.Pair, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) ListBots(ctx context.Context, userID uuid.UUID) ([]Bot, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, name, bot_type, status, config, exchange, pair, created_at
		 FROM bots WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBots(rows)
}

func (s *Store) GetBot(ctx context.Context, id uuid.UUID) (*Bot, error) {
	var b Bot
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, name, bot_type, status, config, exchange, pair, created_at
		 FROM bots WHERE id=$1`, id).
		Scan(&b.ID, &b.UserID, &b.Name, &b.BotType, &b.Status, &b.Config, &b.Exchange, &b.Pair, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) SetBotStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := s.db.Exec(ctx, `UPDATE bots SET status=$1 WHERE id=$2`, status, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteBot(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM bots WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ==================== Bot executions ====================

type BotExecution struct {
	ID        uuid.UUID
	BotID     uuid.UUID
	Status    string
	PNL       string
	StartedAt time.Time
	EndedAt   *time.Time
}

func (s *Store) ListBotExecutions(ctx context.Context, botID uuid.UUID) ([]BotExecution, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, bot_id, status, pnl, started_at, ended_at
		 FROM bot_executions WHERE bot_id=$1 ORDER BY started_at DESC LIMIT 200`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BotExecution{}
	for rows.Next() {
		var e BotExecution
		if err := rows.Scan(&e.ID, &e.BotID, &e.Status, &e.PNL, &e.StartedAt, &e.EndedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ==================== Bot logs ====================

type BotLog struct {
	ID        uuid.UUID
	BotID     uuid.UUID
	Level     string
	Message   string
	CreatedAt time.Time
}

func (s *Store) AppendBotLog(ctx context.Context, botID uuid.UUID, level, message string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO bot_logs (bot_id, level, message) VALUES ($1,$2,$3)`,
		botID, level, message)
	return err
}

func (s *Store) ListBotLogs(ctx context.Context, botID uuid.UUID) ([]BotLog, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, bot_id, level, message, created_at
		 FROM bot_logs WHERE bot_id=$1 ORDER BY created_at DESC LIMIT 200`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BotLog{}
	for rows.Next() {
		var l BotLog
		if err := rows.Scan(&l.ID, &l.BotID, &l.Level, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ==================== Subscriptions ====================

type Subscription struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Tier      string
	StartedAt time.Time
	ExpiresAt *time.Time
}

func (s *Store) CreateSubscription(ctx context.Context, userID uuid.UUID, tier string, expiresAt *time.Time) (*Subscription, error) {
	var sub Subscription
	err := s.db.QueryRow(ctx,
		`INSERT INTO subscriptions (user_id, tier, expires_at) VALUES ($1,$2,$3)
		 RETURNING id, user_id, tier, started_at, expires_at`,
		userID, tier, expiresAt).
		Scan(&sub.ID, &sub.UserID, &sub.Tier, &sub.StartedAt, &sub.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (s *Store) ListSubscriptions(ctx context.Context, userID uuid.UUID) ([]Subscription, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, tier, started_at, expires_at
		 FROM subscriptions WHERE user_id=$1 ORDER BY started_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Subscription{}
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.Tier, &sub.StartedAt, &sub.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

// ==================== Fee configs ====================

type FeeConfig struct {
	ID         uuid.UUID
	Name       string
	Percentage string
	Enabled    bool
	CreatedAt  time.Time
}

func (s *Store) CreateFeeConfig(ctx context.Context, name string, percentage string, enabled bool) (*FeeConfig, error) {
	var f FeeConfig
	err := s.db.QueryRow(ctx,
		`INSERT INTO fee_configs (name, percentage, enabled) VALUES ($1,$2,$3)
		 RETURNING id, name, percentage, enabled, created_at`,
		name, percentage, enabled).
		Scan(&f.ID, &f.Name, &f.Percentage, &f.Enabled, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) ListFeeConfigs(ctx context.Context) ([]FeeConfig, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, percentage, enabled, created_at FROM fee_configs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FeeConfig{}
	for rows.Next() {
		var f FeeConfig
		if err := rows.Scan(&f.ID, &f.Name, &f.Percentage, &f.Enabled, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ==================== API keys (encrypted at rest) ====================

type APIKey struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	Exchange         string
	APIKeyEncrypted  string
	Enabled          bool
	CreatedAt        time.Time
}

func (s *Store) CreateAPIKey(ctx context.Context, userID uuid.UUID, exchange, encrypted string) (*APIKey, error) {
	var k APIKey
	err := s.db.QueryRow(ctx,
		`INSERT INTO api_keys (user_id, exchange, api_key_encrypted) VALUES ($1,$2,$3)
		 RETURNING id, user_id, exchange, api_key_encrypted, enabled, created_at`,
		userID, exchange, encrypted).
		Scan(&k.ID, &k.UserID, &k.Exchange, &k.APIKeyEncrypted, &k.Enabled, &k.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (s *Store) ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]APIKey, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, exchange, api_key_encrypted, enabled, created_at
		 FROM api_keys WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []APIKey{}
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Exchange, &k.APIKeyEncrypted, &k.Enabled, &k.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// ==================== helpers ====================

func scanBots(rows interface {
	Next() bool
	Scan(...any) error
}) ([]Bot, error) {
	out := []Bot{}
	for rows.Next() {
		var b Bot
		if err := rows.Scan(&b.ID, &b.UserID, &b.Name, &b.BotType, &b.Status, &b.Config, &b.Exchange, &b.Pair, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func strOrNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}
