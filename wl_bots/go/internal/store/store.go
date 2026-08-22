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
			scopes        TEXT[]    NOT NULL DEFAULT '{}'::text[],
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS scopes TEXT[] NOT NULL DEFAULT '{}'::text[]`,
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

		// is_active for admin user status management (default TRUE so existing
		// users remain active). Mirrors canonical bot_users.is_active.
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE`,

		// Platform-level CEX connector configs (admin-managed). API key + secret
		// are AES-GCM encrypted at rest via wlcrypto, exactly like api_keys.
		`CREATE TABLE IF NOT EXISTS bot_cex_connections (
			id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			exchange            VARCHAR(64) NOT NULL,
			api_key_encrypted   TEXT NOT NULL,
			api_secret_encrypted TEXT NOT NULL,
			is_active           BOOLEAN NOT NULL DEFAULT TRUE,
			created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		// Platform-level DEX connector configs (admin-managed).
		`CREATE TABLE IF NOT EXISTS bot_dex_connections (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			dex        VARCHAR(64) NOT NULL,
			chain_id   BIGINT NOT NULL,
			rpc_url    TEXT,
			is_active  BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		// Admin fee-collection addresses (admin-managed).
		`CREATE TABLE IF NOT EXISTS admin_fee_addresses (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			label      VARCHAR(255) NOT NULL,
			address    TEXT NOT NULL,
			chain_id   BIGINT NOT NULL DEFAULT 1,
			is_active  BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(address, chain_id)
		)`,
		// Audit trail for stateless auth events (e.g. logout). Logout is
		// audit-only because JWTs are stateless; we record the event here.
		`CREATE TABLE IF NOT EXISTS audit_events (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
			action     VARCHAR(64) NOT NULL,
			detail     TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
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
	Scopes       []string
	IsActive     bool
	CreatedAt    time.Time
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, role string) (*User, error) {
	if role == "" {
		role = "client"
	}
	var u User
	err := s.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1,$2,$3)
		 RETURNING id, email, password_hash, role, scopes, is_active, created_at`,
		email, passwordHash, role).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Scopes, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, role, scopes, is_active, created_at FROM users WHERE email=$1`,
		email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Scopes, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, role, scopes, is_active, created_at FROM users WHERE id=$1`,
		id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Scopes, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ListUsers returns up to limit users (admin surface). Mirrors canonical
// adminListUsers.
func (s *Store) ListUsers(ctx context.Context, limit int) ([]User, error) {
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	rows, err := s.db.Query(ctx,
		`SELECT id, email, password_hash, role, scopes, is_active, created_at
		 FROM users ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Scopes, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// UpdateUserStatus sets the is_active flag and (optionally) the role. Mirrors
// canonical adminUserStatus.
func (s *Store) UpdateUserStatus(ctx context.Context, id uuid.UUID, isActive bool, role string) error {
	if role != "" {
		_, err := s.db.Exec(ctx, `UPDATE users SET is_active=$1, role=$2 WHERE id=$3`, isActive, role, id)
		return err
	}
	_, err := s.db.Exec(ctx, `UPDATE users SET is_active=$1 WHERE id=$2`, isActive, id)
	return err
}

// UpdateUserScopes replaces a user's scoped-admin roles (the canonical
// taxonomy from white_label_admin/go/internal/roles). This is how the WL
// client grants an admin bot_admin / finance_admin etc. — the scopes are
// stored in the users table and issued in the JWT at login, then enforced by
// wlgate.RequireScope on each admin route.
func (s *Store) UpdateUserScopes(ctx context.Context, id uuid.UUID, scopes []string) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET scopes=$1 WHERE id=$2`, scopes, id)
	return err
}

func (s *Store) DeleteUser(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateBotUser creates a user with an email-as-username (no password — set
// later). Mirrors canonical createBotUser (admin/operator surface).
func (s *Store) CreateBotUser(ctx context.Context, email, role string) (*User, error) {
	if role == "" {
		role = "client"
	}
	var u User
	err := s.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1,$2,$3)
		 RETURNING id, email, password_hash, role, is_active, created_at`,
		email, "", role).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)
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

// UpdateFeeConfig updates an existing fee config by id (real PG UPDATE).
// Mirrors canonical updateFeeConfig semantics (mutate fee percent / enabled).
func (s *Store) UpdateFeeConfig(ctx context.Context, id uuid.UUID, name, percentage string, enabled bool) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE fee_configs SET name=$1, percentage=$2, enabled=$3 WHERE id=$4`,
		name, percentage, enabled, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
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

// DeleteAPIKey removes an API key owned by userID. Mirrors canonical deleteAPIKey.
func (s *Store) DeleteAPIKey(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM api_keys WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ==================== CEX connector configs (AES-GCM at rest) ====================

type CEXConnection struct {
	ID                 uuid.UUID
	Exchange           string
	APIKeyEncrypted    string
	APISecretEncrypted string
	IsActive           bool
	CreatedAt          time.Time
}

func (s *Store) CreateCEX(ctx context.Context, exchange, encKey, encSecret string) (*CEXConnection, error) {
	var c CEXConnection
	err := s.db.QueryRow(ctx,
		`INSERT INTO bot_cex_connections (exchange, api_key_encrypted, api_secret_encrypted)
		 VALUES ($1,$2,$3)
		 RETURNING id, exchange, api_key_encrypted, api_secret_encrypted, is_active, created_at`,
		exchange, encKey, encSecret).
		Scan(&c.ID, &c.Exchange, &c.APIKeyEncrypted, &c.APISecretEncrypted, &c.IsActive, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) ListCEX(ctx context.Context) ([]CEXConnection, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, exchange, api_key_encrypted, api_secret_encrypted, is_active, created_at
		 FROM bot_cex_connections ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CEXConnection{}
	for rows.Next() {
		var c CEXConnection
		if err := rows.Scan(&c.ID, &c.Exchange, &c.APIKeyEncrypted, &c.APISecretEncrypted, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) DeleteCEX(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM bot_cex_connections WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ==================== DEX connector configs ====================

type DEXConnection struct {
	ID        uuid.UUID
	DEX       string
	ChainID   int64
	RPCURL    string
	IsActive  bool
	CreatedAt time.Time
}

func (s *Store) CreateDEX(ctx context.Context, dex string, chainID int64, rpcURL string) (*DEXConnection, error) {
	var d DEXConnection
	err := s.db.QueryRow(ctx,
		`INSERT INTO bot_dex_connections (dex, chain_id, rpc_url)
		 VALUES ($1,$2,$3)
		 RETURNING id, dex, chain_id, rpc_url, is_active, created_at`,
		dex, chainID, strOrNull(rpcURL)).
		Scan(&d.ID, &d.DEX, &d.ChainID, &d.RPCURL, &d.IsActive, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) ListDEX(ctx context.Context) ([]DEXConnection, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, dex, chain_id, rpc_url, is_active, created_at
		 FROM bot_dex_connections ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DEXConnection{}
	for rows.Next() {
		var d DEXConnection
		if err := rows.Scan(&d.ID, &d.DEX, &d.ChainID, &d.RPCURL, &d.IsActive, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) DeleteDEX(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM bot_dex_connections WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ==================== Admin fee addresses ====================

type FeeAddress struct {
	ID        uuid.UUID
	Label     string
	Address   string
	ChainID   int64
	IsActive  bool
	CreatedAt time.Time
}

func (s *Store) CreateFeeAddress(ctx context.Context, label, address string, chainID int64) (*FeeAddress, error) {
	var f FeeAddress
	err := s.db.QueryRow(ctx,
		`INSERT INTO admin_fee_addresses (label, address, chain_id)
		 VALUES ($1,$2,$3)
		 ON CONFLICT (address, chain_id) DO UPDATE SET label=excluded.label, is_active=TRUE
		 RETURNING id, label, address, chain_id, is_active, created_at`,
		label, address, chainID).
		Scan(&f.ID, &f.Label, &f.Address, &f.ChainID, &f.IsActive, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) ListFeeAddresses(ctx context.Context) ([]FeeAddress, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, label, address, chain_id, is_active, created_at
		 FROM admin_fee_addresses ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FeeAddress{}
	for rows.Next() {
		var f FeeAddress
		if err := rows.Scan(&f.ID, &f.Label, &f.Address, &f.ChainID, &f.IsActive, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) DeleteFeeAddress(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM admin_fee_addresses WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ==================== Platform stats (real COUNTs) ====================

type Stats struct {
	TotalUsers      int
	TotalBots       int
	RunningBots     int
	TotalExecutions int
}

func (s *Store) Stats(ctx context.Context) (*Stats, error) {
	var st Stats
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&st.TotalUsers); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM bots`).Scan(&st.TotalBots); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM bots WHERE status='running'`).Scan(&st.RunningBots); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM bot_executions`).Scan(&st.TotalExecutions); err != nil {
		return nil, err
	}
	return &st, nil
}

// BotTypeCount is a single bot_type -> count row.
type BotTypeCount struct {
	BotType string
	Count   int
}

// BotTypeDistribution returns the per-bot_type counts (real GROUP BY).
func (s *Store) BotTypeDistribution(ctx context.Context) ([]BotTypeCount, error) {
	rows, err := s.db.Query(ctx,
		`SELECT bot_type, COUNT(*) FROM bots GROUP BY bot_type ORDER BY COUNT(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BotTypeCount{}
	for rows.Next() {
		var b BotTypeCount
		if err := rows.Scan(&b.BotType, &b.Count); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ==================== Bot transactions (executions for caller's bots) ====================

// BotTransaction is a flat transaction view derived from bot_executions joined
// to bots. wl_bots has no fabricated trades table; executions ARE the real
// on-record transaction history.
type BotTransaction struct {
	ID        uuid.UUID
	BotID     uuid.UUID
	Status    string
	PNL       string
	StartedAt time.Time
	EndedAt   *time.Time
}

// ListBotTransactions returns executions across the caller's bots (filtered by
// user_id via JOIN). If userID is uuid.Nil, returns all (admin).
func (s *Store) ListBotTransactions(ctx context.Context, userID uuid.UUID) ([]BotTransaction, error) {
	q := `SELECT e.id, e.bot_id, e.status, e.pnl, e.started_at, e.ended_at
	      FROM bot_executions e JOIN bots b ON b.id=e.bot_id `
	args := []any{}
	if userID != uuid.Nil {
		q += `WHERE b.user_id=$1 `
		args = append(args, userID)
	}
	q += `ORDER BY e.started_at DESC LIMIT 200`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BotTransaction{}
	for rows.Next() {
		var t BotTransaction
		if err := rows.Scan(&t.ID, &t.BotID, &t.Status, &t.PNL, &t.StartedAt, &t.EndedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ==================== Audit events (stateless auth audit) ====================

// RecordAuditEvent persists an audit event (e.g. logout). Real PG INSERT.
func (s *Store) RecordAuditEvent(ctx context.Context, userID uuid.UUID, action, detail string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO audit_events (user_id, action, detail) VALUES ($1,$2,$3)`,
		userID, action, detail)
	return err
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
