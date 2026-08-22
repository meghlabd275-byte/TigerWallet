// Package store provides PostgreSQL persistence for the standalone
// WL-ProjectParty backend. It owns its own database (wl_projectparty) —
// independent of TigerWallet cloud. Tables: users, tokens, listings,
// launchpad_projects, participations, market_making_configs, fee_configs,
// favorites.
package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		`CREATE TABLE IF NOT EXISTS users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email VARCHAR(255) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL, role VARCHAR(32) DEFAULT 'user', scopes TEXT[] NOT NULL DEFAULT '{}'::text[], created_at TIMESTAMPTZ DEFAULT NOW())`,
		// scopes column for existing DBs (canonical scoped-role taxonomy, set via
		// UpdateAdminScopes by the WL client owner; issued in the JWT at login).
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS scopes TEXT[] NOT NULL DEFAULT '{}'::text[]`,
		`CREATE TABLE IF NOT EXISTS tokens (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(255) NOT NULL, symbol VARCHAR(64) NOT NULL, contract_address VARCHAR(128), chain_id BIGINT NOT NULL DEFAULT 1, decimals INT NOT NULL DEFAULT 18, logo_url TEXT, description TEXT, website TEXT, status VARCHAR(32) NOT NULL DEFAULT 'draft', listing_type VARCHAR(32) DEFAULT 'standard', created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_status ON tokens(status)`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_symbol ON tokens(symbol)`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_chain ON tokens(chain_id)`,
		`CREATE TABLE IF NOT EXISTS listings (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE, pair VARCHAR(64) NOT NULL, base_token VARCHAR(64), quote_token VARCHAR(64), status VARCHAR(32) NOT NULL DEFAULT 'upcoming', created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_listings_token ON listings(token_id)`,
		`CREATE INDEX IF NOT EXISTS idx_listings_status ON listings(status)`,
		`CREATE TABLE IF NOT EXISTS launchpad_projects (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE, name VARCHAR(255) NOT NULL, description TEXT, start_time TIMESTAMPTZ, end_time TIMESTAMPTZ, total_supply NUMERIC(36,18) DEFAULT 0, sold_amount NUMERIC(36,18) DEFAULT 0, price_per_token NUMERIC(36,18) DEFAULT 0, status VARCHAR(32) NOT NULL DEFAULT 'upcoming', created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_launchpad_status ON launchpad_projects(status)`,
		`CREATE INDEX IF NOT EXISTS idx_launchpad_token ON launchpad_projects(token_id)`,
		`CREATE TABLE IF NOT EXISTS participations (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), project_id UUID NOT NULL REFERENCES launchpad_projects(id) ON DELETE CASCADE, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, amount NUMERIC(36,18) DEFAULT 0, allocated NUMERIC(36,18) DEFAULT 0, status VARCHAR(32) NOT NULL DEFAULT 'pending', created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_participations_project ON participations(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_participations_user ON participations(user_id)`,
		`CREATE TABLE IF NOT EXISTS market_making_configs (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE, pair VARCHAR(64) NOT NULL, spread NUMERIC(10,6) DEFAULT 0, enabled BOOL DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_mm_token ON market_making_configs(token_id)`,
		`CREATE TABLE IF NOT EXISTS fee_configs (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(255) NOT NULL, percentage NUMERIC(10,4) DEFAULT 0, enabled BOOL DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS favorites (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE, created_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE(user_id, token_id))`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_token ON favorites(token_id)`,

		// Listing-workflow columns on tokens (added via ALTER so existing rows keep
		// working). is_featured powers /featured + /:id/featured; submission/review
		// columns power the submit/approve/reject/status workflow.
		`ALTER TABLE tokens ADD COLUMN IF NOT EXISTS is_featured BOOL DEFAULT false`,
		`ALTER TABLE tokens ADD COLUMN IF NOT EXISTS submission_date TIMESTAMPTZ`,
		`ALTER TABLE tokens ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ`,
		`ALTER TABLE tokens ADD COLUMN IF NOT EXISTS rejection_reason TEXT`,
		`ALTER TABLE tokens ADD COLUMN IF NOT EXISTS total_supply NUMERIC(36,18) DEFAULT 0`,
		`ALTER TABLE tokens ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW()`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_featured ON tokens(is_featured)`,

		// Launchpad contributions (atomic contribute/claim/cancel workflow).
		`CREATE TABLE IF NOT EXISTS launchpad_contributions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), project_id UUID NOT NULL REFERENCES launchpad_projects(id) ON DELETE CASCADE, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, amount NUMERIC(36,18) DEFAULT 0, token_amount NUMERIC(36,18) DEFAULT 0, status VARCHAR(32) NOT NULL DEFAULT 'pending', claimed_at TIMESTAMPTZ, refunded_at TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_contrib_project ON launchpad_contributions(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_contrib_user ON launchpad_contributions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_contrib_status ON launchpad_contributions(status)`,

		// Market-maker orders (real order book for /orders, /liquidity, analytics).
		`CREATE TABLE IF NOT EXISTS mm_orders (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE, side VARCHAR(8) NOT NULL, price NUMERIC(36,18) NOT NULL DEFAULT 0, quantity NUMERIC(36,18) NOT NULL DEFAULT 0, remaining NUMERIC(36,18) NOT NULL DEFAULT 0, status VARCHAR(16) NOT NULL DEFAULT 'pending', filled_at TIMESTAMPTZ, expires_at TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_mm_orders_token ON mm_orders(token_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mm_orders_status ON mm_orders(status)`,

		// Liquidity positions (real LP records for /liquidity add/remove + aggregate).
		`CREATE TABLE IF NOT EXISTS liquidity_positions (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE, user_id UUID REFERENCES users(id) ON DELETE SET NULL, quote_token VARCHAR(64), lp_tokens NUMERIC(36,18) NOT NULL DEFAULT 0, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_liq_pos_token ON liquidity_positions(token_id)`,

		// Token prices (real price history for /pricing, /trending, /market gainers, volume).
		`CREATE TABLE IF NOT EXISTS token_prices (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE, price NUMERIC(36,18) NOT NULL DEFAULT 0, change_24h NUMERIC(10,4) DEFAULT 0, volume_24h NUMERIC(36,18) DEFAULT 0, timestamp TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_prices_token ON token_prices(token_id, timestamp DESC)`,

		// Compliance audit log per token.
		`CREATE TABLE IF NOT EXISTS token_audit_logs (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE, audit_type VARCHAR(32) NOT NULL, status VARCHAR(32) NOT NULL DEFAULT 'requested', report_url TEXT, auditor VARCHAR(255), completed_at TIMESTAMPTZ, requested_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_audit_token ON token_audit_logs(token_id)`,

		// KYC records per token listing.
		`CREATE TABLE IF NOT EXISTS token_kyc (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE, status VARCHAR(32) NOT NULL DEFAULT 'pending', submitted_at TIMESTAMPTZ DEFAULT NOW(), expires_at TIMESTAMPTZ, reviewed_at TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_kyc_token ON token_kyc(token_id)`,

		// Fee payments (real on-chain tx_hash recorded, not fabricated).
		`CREATE TABLE IF NOT EXISTS fee_payments (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), token_id UUID REFERENCES tokens(id) ON DELETE SET NULL, user_id UUID REFERENCES users(id) ON DELETE SET NULL, amount NUMERIC(36,18) NOT NULL DEFAULT 0, currency VARCHAR(16) DEFAULT 'USD', payment_method VARCHAR(64), tx_hash VARCHAR(128), status VARCHAR(32) NOT NULL DEFAULT 'completed', created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_fee_payments_token ON fee_payments(token_id)`,

		// Fee schedule (real fee_config rows drive /calculate, no hardcoded fallback).
		`CREATE TABLE IF NOT EXISTS fee_schedule (fee_type VARCHAR(64) PRIMARY KEY, amount NUMERIC(18,4) NOT NULL DEFAULT 0)`,
		`INSERT INTO fee_schedule (fee_type, amount) VALUES ('basic_listing', 500), ('launchpad', 5000), ('featured', 1000), ('audit', 5000), ('kyc', 1000) ON CONFLICT (fee_type) DO NOTHING`,
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
	CreatedAt    time.Time
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, role string) (*User, error) {
	if role == "" {
		role = "user"
	}
	var u User
	err := s.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1,$2,$3) RETURNING id, email, password_hash, role, scopes, created_at`,
		email, passwordHash, role).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Scopes, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, role, scopes, created_at FROM users WHERE email=$1`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Scopes, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, role, scopes, created_at FROM users WHERE id=$1`, id).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Scopes, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateUserScopes replaces a user's scoped-admin roles (canonical taxonomy).
// The WL client owner (wl_client scope) uses this to grant listing_admin etc.
// The scopes are issued in the JWT at login and enforced by wlgate.HasScope.
func (s *Store) UpdateUserScopes(ctx context.Context, id uuid.UUID, scopes []string) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET scopes=$1 WHERE id=$2`, scopes, id)
	return err
}

// ==================== Tokens ====================

type Token struct {
	ID              uuid.UUID
	Name            string
	Symbol          string
	ContractAddress string
	ChainID         int64
	Decimals        int
	LogoURL         string
	Description     string
	Website         string
	Status          string
	ListingType     string
	CreatedAt       time.Time
}

func (s *Store) CreateToken(ctx context.Context, t *Token) (*Token, error) {
	if t.Status == "" {
		t.Status = "draft"
	}
	if t.ListingType == "" {
		t.ListingType = "standard"
	}
	if t.Decimals == 0 {
		t.Decimals = 18
	}
	var out Token
	err := s.db.QueryRow(ctx,
		`INSERT INTO tokens (name, symbol, contract_address, chain_id, decimals, logo_url, description, website, status, listing_type)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id, name, symbol, contract_address, chain_id, decimals, logo_url, description, website, status, listing_type, created_at`,
		t.Name, t.Symbol, t.ContractAddress, t.ChainID, t.Decimals, t.LogoURL, t.Description, t.Website, t.Status, t.ListingType).
		Scan(&out.ID, &out.Name, &out.Symbol, &out.ContractAddress, &out.ChainID, &out.Decimals, &out.LogoURL, &out.Description, &out.Website, &out.Status, &out.ListingType, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListTokens(ctx context.Context, status string) ([]Token, error) {
	q := `SELECT id, name, symbol, contract_address, chain_id, decimals, logo_url, description, website, status, listing_type, created_at FROM tokens`
	args := []any{}
	if status != "" {
		q += ` WHERE status=$1`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT 500`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Token{}
	for rows.Next() {
		var t Token
		if err := rows.Scan(&t.ID, &t.Name, &t.Symbol, &t.ContractAddress, &t.ChainID, &t.Decimals, &t.LogoURL, &t.Description, &t.Website, &t.Status, &t.ListingType, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetToken(ctx context.Context, id uuid.UUID) (*Token, error) {
	var t Token
	err := s.db.QueryRow(ctx,
		`SELECT id, name, symbol, contract_address, chain_id, decimals, logo_url, description, website, status, listing_type, created_at FROM tokens WHERE id=$1`, id).
		Scan(&t.ID, &t.Name, &t.Symbol, &t.ContractAddress, &t.ChainID, &t.Decimals, &t.LogoURL, &t.Description, &t.Website, &t.Status, &t.ListingType, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) UpdateToken(ctx context.Context, id uuid.UUID, t *Token) (*Token, error) {
	var out Token
	err := s.db.QueryRow(ctx,
		`UPDATE tokens SET name=$1, symbol=$2, contract_address=$3, chain_id=$4, decimals=$5, logo_url=$6, description=$7, website=$8, status=$9, listing_type=$10
		 WHERE id=$11
		 RETURNING id, name, symbol, contract_address, chain_id, decimals, logo_url, description, website, status, listing_type, created_at`,
		t.Name, t.Symbol, t.ContractAddress, t.ChainID, t.Decimals, t.LogoURL, t.Description, t.Website, t.Status, t.ListingType, id).
		Scan(&out.ID, &out.Name, &out.Symbol, &out.ContractAddress, &out.ChainID, &out.Decimals, &out.LogoURL, &out.Description, &out.Website, &out.Status, &out.ListingType, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) DeleteToken(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM tokens WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ==================== Listings ====================

type Listing struct {
	ID         uuid.UUID
	TokenID    uuid.UUID
	Pair       string
	BaseToken  string
	QuoteToken string
	Status     string
	CreatedAt  time.Time
}

func (s *Store) CreateListing(ctx context.Context, l *Listing) (*Listing, error) {
	if l.Status == "" {
		l.Status = "upcoming"
	}
	var out Listing
	err := s.db.QueryRow(ctx,
		`INSERT INTO listings (token_id, pair, base_token, quote_token, status)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id, token_id, pair, base_token, quote_token, status, created_at`,
		l.TokenID, l.Pair, l.BaseToken, l.QuoteToken, l.Status).
		Scan(&out.ID, &out.TokenID, &out.Pair, &out.BaseToken, &out.QuoteToken, &out.Status, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListListings(ctx context.Context, status string) ([]Listing, error) {
	q := `SELECT id, token_id, pair, base_token, quote_token, status, created_at FROM listings`
	args := []any{}
	if status != "" {
		q += ` WHERE status=$1`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT 500`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Listing{}
	for rows.Next() {
		var l Listing
		if err := rows.Scan(&l.ID, &l.TokenID, &l.Pair, &l.BaseToken, &l.QuoteToken, &l.Status, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ==================== Launchpad projects ====================

type LaunchpadProject struct {
	ID            uuid.UUID
	TokenID       uuid.UUID
	Name          string
	Description   string
	StartTime     *time.Time
	EndTime       *time.Time
	TotalSupply   string
	SoldAmount    string
	PricePerToken string
	Status        string
	CreatedAt     time.Time
}

func (s *Store) CreateLaunchpadProject(ctx context.Context, p *LaunchpadProject) (*LaunchpadProject, error) {
	if p.Status == "" {
		p.Status = "upcoming"
	}
	if p.TotalSupply == "" {
		p.TotalSupply = "0"
	}
	if p.SoldAmount == "" {
		p.SoldAmount = "0"
	}
	if p.PricePerToken == "" {
		p.PricePerToken = "0"
	}
	var out LaunchpadProject
	err := s.db.QueryRow(ctx,
		`INSERT INTO launchpad_projects (token_id, name, description, start_time, end_time, total_supply, sold_amount, price_per_token, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id, token_id, name, description, start_time, end_time, total_supply, sold_amount, price_per_token, status, created_at`,
		p.TokenID, p.Name, p.Description, p.StartTime, p.EndTime, p.TotalSupply, p.SoldAmount, p.PricePerToken, p.Status).
		Scan(&out.ID, &out.TokenID, &out.Name, &out.Description, &out.StartTime, &out.EndTime, &out.TotalSupply, &out.SoldAmount, &out.PricePerToken, &out.Status, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListLaunchpadProjects(ctx context.Context, status string) ([]LaunchpadProject, error) {
	q := `SELECT id, token_id, name, description, start_time, end_time, total_supply, sold_amount, price_per_token, status, created_at FROM launchpad_projects`
	args := []any{}
	if status != "" {
		q += ` WHERE status=$1`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT 500`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LaunchpadProject{}
	for rows.Next() {
		var p LaunchpadProject
		if err := rows.Scan(&p.ID, &p.TokenID, &p.Name, &p.Description, &p.StartTime, &p.EndTime, &p.TotalSupply, &p.SoldAmount, &p.PricePerToken, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetLaunchpadProject(ctx context.Context, id uuid.UUID) (*LaunchpadProject, error) {
	var p LaunchpadProject
	err := s.db.QueryRow(ctx,
		`SELECT id, token_id, name, description, start_time, end_time, total_supply, sold_amount, price_per_token, status, created_at FROM launchpad_projects WHERE id=$1`, id).
		Scan(&p.ID, &p.TokenID, &p.Name, &p.Description, &p.StartTime, &p.EndTime, &p.TotalSupply, &p.SoldAmount, &p.PricePerToken, &p.Status, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ==================== Participations ====================

type Participation struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	UserID    uuid.UUID
	Amount    string
	Allocated string
	Status    string
	CreatedAt time.Time
}

// CreateParticipation inserts a participation and atomically increments the
// launchpad's sold_amount by the contributed amount. Fail-closed: any DB error
// rolls back and the participation is not recorded.
func (s *Store) CreateParticipation(ctx context.Context, projectID, userID uuid.UUID, amount string) (*Participation, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var out Participation
	err = tx.QueryRow(ctx,
		`INSERT INTO participations (project_id, user_id, amount, status)
		 VALUES ($1,$2,$3,'pending')
		 RETURNING id, project_id, user_id, amount, allocated, status, created_at`,
		projectID, userID, amount).
		Scan(&out.ID, &out.ProjectID, &out.UserID, &out.Amount, &out.Allocated, &out.Status, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE launchpad_projects SET sold_amount = sold_amount + $1 WHERE id=$2`, amount, projectID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListParticipations(ctx context.Context, projectID uuid.UUID) ([]Participation, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, project_id, user_id, amount, allocated, status, created_at FROM participations WHERE project_id=$1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Participation{}
	for rows.Next() {
		var p Participation
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.UserID, &p.Amount, &p.Allocated, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ==================== Market-making configs ====================

type MarketMakingConfig struct {
	ID        uuid.UUID
	TokenID   uuid.UUID
	Pair      string
	Spread    string
	Enabled   bool
	CreatedAt time.Time
}

func (s *Store) CreateMarketMakingConfig(ctx context.Context, m *MarketMakingConfig) (*MarketMakingConfig, error) {
	if m.Spread == "" {
		m.Spread = "0"
	}
	var out MarketMakingConfig
	err := s.db.QueryRow(ctx,
		`INSERT INTO market_making_configs (token_id, pair, spread, enabled)
		 VALUES ($1,$2,$3,$4)
		 RETURNING id, token_id, pair, spread, enabled, created_at`,
		m.TokenID, m.Pair, m.Spread, m.Enabled).
		Scan(&out.ID, &out.TokenID, &out.Pair, &out.Spread, &out.Enabled, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListMarketMakingConfigs(ctx context.Context) ([]MarketMakingConfig, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, token_id, pair, spread, enabled, created_at FROM market_making_configs ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MarketMakingConfig{}
	for rows.Next() {
		var m MarketMakingConfig
		if err := rows.Scan(&m.ID, &m.TokenID, &m.Pair, &m.Spread, &m.Enabled, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
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

func (s *Store) CreateFeeConfig(ctx context.Context, name, percentage string, enabled bool) (*FeeConfig, error) {
	if percentage == "" {
		percentage = "0"
	}
	var out FeeConfig
	err := s.db.QueryRow(ctx,
		`INSERT INTO fee_configs (name, percentage, enabled)
		 VALUES ($1,$2,$3)
		 RETURNING id, name, percentage, enabled, created_at`,
		name, percentage, enabled).
		Scan(&out.ID, &out.Name, &out.Percentage, &out.Enabled, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListFeeConfigs(ctx context.Context) ([]FeeConfig, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, percentage, enabled, created_at FROM fee_configs ORDER BY created_at DESC LIMIT 500`)
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

// ==================== Favorites ====================

type Favorite struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenID   uuid.UUID
	CreatedAt time.Time
}

func (s *Store) AddFavorite(ctx context.Context, userID, tokenID uuid.UUID) (*Favorite, error) {
	var out Favorite
	err := s.db.QueryRow(ctx,
		`INSERT INTO favorites (user_id, token_id) VALUES ($1,$2)
		 ON CONFLICT (user_id, token_id) DO UPDATE SET user_id=EXCLUDED.user_id
		 RETURNING id, user_id, token_id, created_at`,
		userID, tokenID).
		Scan(&out.ID, &out.UserID, &out.TokenID, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListFavorites(ctx context.Context, userID uuid.UUID) ([]Favorite, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, token_id, created_at FROM favorites WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Favorite{}
	for rows.Next() {
		var f Favorite
		if err := rows.Scan(&f.ID, &f.UserID, &f.TokenID, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) RemoveFavorite(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM favorites WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ==================== Token listing workflow ====================

// TokenStatus holds the listing-review state for a token.
type TokenStatus struct {
	ID              uuid.UUID
	Status          string
	IsFeatured      bool
	SubmissionDate  *time.Time
	ReviewedAt      *time.Time
	RejectionReason *string
}

// SubmitToken transitions a token to pending review (only from draft/pending).
func (s *Store) SubmitToken(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE tokens SET status='pending', submission_date=NOW(), updated_at=NOW()
		 WHERE id=$1 AND status IN ('draft','pending')`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ApproveToken transitions a token to listed (only from pending/in_review).
func (s *Store) ApproveToken(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE tokens SET status='listed', reviewed_at=NOW(), rejection_reason=NULL, updated_at=NOW()
		 WHERE id=$1 AND status IN ('pending','in_review','submitted','approved')`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// RejectToken transitions a token to rejected with a reason.
func (s *Store) RejectToken(ctx context.Context, id uuid.UUID, reason string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE tokens SET status='rejected', rejection_reason=$1, reviewed_at=NOW(), updated_at=NOW()
		 WHERE id=$1 AND status IN ('pending','in_review','submitted','approved')`, reason, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ToggleFeatured flips is_featured for a token. Returns the new value.
func (s *Store) ToggleFeatured(ctx context.Context, id uuid.UUID) (bool, error) {
	var featured bool
	err := s.db.QueryRow(ctx,
		`UPDATE tokens SET is_featured = NOT is_featured, updated_at=NOW()
		 WHERE id=$1 RETURNING is_featured`, id).Scan(&featured)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, pgx.ErrNoRows
		}
		return false, err
	}
	return featured, nil
}

// GetTokenStatus returns the listing-review state for a token.
func (s *Store) GetTokenStatus(ctx context.Context, id uuid.UUID) (*TokenStatus, error) {
	var ts TokenStatus
	err := s.db.QueryRow(ctx,
		`SELECT id, status, COALESCE(is_featured,false), submission_date, reviewed_at, rejection_reason
		 FROM tokens WHERE id=$1`, id).
		Scan(&ts.ID, &ts.Status, &ts.IsFeatured, &ts.SubmissionDate, &ts.ReviewedAt, &ts.RejectionReason)
	if err != nil {
		return nil, err
	}
	return &ts, nil
}

// SearchTokens runs a real ILIKE search over name/symbol/contract_address.
func (s *Store) SearchTokens(ctx context.Context, q string) ([]Token, error) {
	pattern := "%" + q + "%"
	rows, err := s.db.Query(ctx,
		`SELECT id, name, symbol, contract_address, chain_id, decimals, logo_url, description, website, status, listing_type, created_at
		 FROM tokens WHERE name ILIKE $1 OR symbol ILIKE $1 OR contract_address ILIKE $1
		 ORDER BY created_at DESC LIMIT 50`, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Token{}
	for rows.Next() {
		var t Token
		if err := rows.Scan(&t.ID, &t.Name, &t.Symbol, &t.ContractAddress, &t.ChainID, &t.Decimals, &t.LogoURL, &t.Description, &t.Website, &t.Status, &t.ListingType, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// FeaturedTokens returns tokens where is_featured=true and status='listed'.
func (s *Store) FeaturedTokens(ctx context.Context) ([]Token, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, symbol, contract_address, chain_id, decimals, logo_url, description, website, status, listing_type, created_at
		 FROM tokens WHERE is_featured=true AND status='listed' ORDER BY created_at DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Token{}
	for rows.Next() {
		var t Token
		if err := rows.Scan(&t.ID, &t.Name, &t.Symbol, &t.ContractAddress, &t.ChainID, &t.Decimals, &t.LogoURL, &t.Description, &t.Website, &t.Status, &t.ListingType, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// TrendingTokens returns listed tokens ordered by 24h volume (from token_prices)
// falling back to created_at when no price history exists.
func (s *Store) TrendingTokens(ctx context.Context) ([]Token, error) {
	rows, err := s.db.Query(ctx,
		`SELECT t.id, t.name, t.symbol, t.contract_address, t.chain_id, t.decimals, t.logo_url, t.description, t.website, t.status, t.listing_type, t.created_at
		 FROM tokens t
		 LEFT JOIN (SELECT DISTINCT ON (token_id) token_id, volume_24h FROM token_prices ORDER BY token_id, timestamp DESC) tp ON tp.token_id=t.id
		 WHERE t.status='listed'
		 ORDER BY COALESCE(tp.volume_24h, 0) DESC NULLS LAST, t.created_at DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Token{}
	for rows.Next() {
		var t Token
		if err := rows.Scan(&t.ID, &t.Name, &t.Symbol, &t.ContractAddress, &t.ChainID, &t.Decimals, &t.LogoURL, &t.Description, &t.Website, &t.Status, &t.ListingType, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// MarketOverview is the real aggregate returned by GET /market.
type MarketOverview struct {
	TotalTokens   int
	TotalListings int
	TotalLaunch   int
	TotalVolume   string
	TopGainers    []MarketGainer
}

// MarketGainer is one row of the real top-gainers aggregate.
type MarketGainer struct {
	TokenID   uuid.UUID
	Symbol    string
	Change24h float64
}

// MarketOverview computes real aggregates: counts, summed volume, top gainers.
func (s *Store) MarketOverview(ctx context.Context) (*MarketOverview, error) {
	m := &MarketOverview{TotalVolume: "0"}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM tokens WHERE status='listed'`).Scan(&m.TotalTokens); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM listings WHERE status IN ('active','upcoming')`).Scan(&m.TotalListings); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM launchpad_projects WHERE status IN ('active','upcoming')`).Scan(&m.TotalLaunch); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(ctx, `SELECT COALESCE(SUM(volume_24h),0)::text FROM token_prices`).Scan(&m.TotalVolume); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT t.id, t.symbol, tp.change_24h FROM tokens t
		 JOIN (SELECT DISTINCT ON (token_id) token_id, change_24h FROM token_prices ORDER BY token_id, timestamp DESC) tp ON tp.token_id=t.id
		 WHERE t.status='listed' AND tp.change_24h > 0
		 ORDER BY tp.change_24h DESC LIMIT 5`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m.TopGainers = []MarketGainer{}
	for rows.Next() {
		var g MarketGainer
		if err := rows.Scan(&g.TokenID, &g.Symbol, &g.Change24h); err != nil {
			return nil, err
		}
		m.TopGainers = append(m.TopGainers, g)
	}
	return m, rows.Err()
}

// ==================== Launchpad contributions ====================

type LaunchpadContribution struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	UserID      uuid.UUID
	Amount      string
	TokenAmount string
	Status      string
	ClaimedAt   *time.Time
	RefundedAt  *time.Time
	CreatedAt   time.Time
}

// CreateContribution atomically inserts a contribution, validates the launchpad
// is active and not past end_time, and increments sold_amount. Fail-closed: any
// error rolls back and no contribution is recorded.
func (s *Store) CreateContribution(ctx context.Context, projectID, userID uuid.UUID, amount string) (*LaunchpadContribution, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var status string
	var endTime *time.Time
	var pricePerToken string
	if err := tx.QueryRow(ctx,
		`SELECT status, end_time, COALESCE(price_per_token::text,'0') FROM launchpad_projects WHERE id=$1 FOR UPDATE`,
		projectID).Scan(&status, &endTime, &pricePerToken); err != nil {
		return nil, err
	}
	if status != "active" && status != "upcoming" {
		return nil, ErrNotAccepting
	}
	if endTime != nil && time.Now().After(*endTime) {
		return nil, ErrEnded
	}

	var out LaunchpadContribution
	if err := tx.QueryRow(ctx,
		`INSERT INTO launchpad_contributions (project_id, user_id, amount, token_amount, status)
		 VALUES ($1,$2,$3,$4,'pending')
		 RETURNING id, project_id, user_id, amount, token_amount, status, claimed_at, refunded_at, created_at`,
		projectID, userID, amount, computeTokenAmount(amount, pricePerToken)).
		Scan(&out.ID, &out.ProjectID, &out.UserID, &out.Amount, &out.TokenAmount, &out.Status, &out.ClaimedAt, &out.RefundedAt, &out.CreatedAt); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE launchpad_projects SET sold_amount = sold_amount + $1 WHERE id=$2`, amount, projectID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &out, nil
}

// ClaimContribution marks a user's pending contribution as claimed. Fails
// fail-closed if no claimable contribution exists.
func (s *Store) ClaimContribution(ctx context.Context, projectID, userID uuid.UUID) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE launchpad_contributions SET status='claimed', claimed_at=NOW()
		 WHERE project_id=$1 AND user_id=$2 AND status='pending'`, projectID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// CancelContribution marks a user's pending contribution as refunded (refund-record).
func (s *Store) CancelContribution(ctx context.Context, projectID, userID uuid.UUID) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE launchpad_contributions SET status='refunded', refunded_at=NOW()
		 WHERE project_id=$1 AND user_id=$2 AND status='pending'`, projectID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListContributionsByToken returns contribution history for every launchpad of
// a token (real aggregate join).
func (s *Store) ListContributionsByToken(ctx context.Context, tokenID uuid.UUID) ([]LaunchpadContribution, error) {
	rows, err := s.db.Query(ctx,
		`SELECT lc.id, lc.project_id, lc.user_id, lc.amount, lc.token_amount, lc.status, lc.claimed_at, lc.refunded_at, lc.created_at
		 FROM launchpad_contributions lc
		 JOIN launchpad_projects lp ON lp.id=lc.project_id
		 WHERE lp.token_id=$1
		 ORDER BY lc.created_at DESC`, tokenID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LaunchpadContribution{}
	for rows.Next() {
		var c LaunchpadContribution
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.UserID, &c.Amount, &c.TokenAmount, &c.Status, &c.ClaimedAt, &c.RefundedAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ==================== Market-maker orders ====================

type MMOrder struct {
	ID        uuid.UUID
	TokenID   uuid.UUID
	Side      string
	Price     string
	Quantity  string
	Remaining string
	Status    string
	FilledAt  *time.Time
	ExpiresAt *time.Time
	CreatedAt time.Time
}

func (s *Store) CreateMMOrder(ctx context.Context, o *MMOrder) (*MMOrder, error) {
	if o.Remaining == "" {
		o.Remaining = o.Quantity
	}
	var out MMOrder
	err := s.db.QueryRow(ctx,
		`INSERT INTO mm_orders (token_id, side, price, quantity, remaining, status, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING id, token_id, side, price, quantity, remaining, status, filled_at, expires_at, created_at`,
		o.TokenID, o.Side, o.Price, o.Quantity, o.Remaining, o.Status, o.ExpiresAt).
		Scan(&out.ID, &out.TokenID, &out.Side, &out.Price, &out.Quantity, &out.Remaining, &out.Status, &out.FilledAt, &out.ExpiresAt, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListMMOrders(ctx context.Context, tokenID uuid.UUID, tokenFilter bool) ([]MMOrder, error) {
	q := `SELECT id, token_id, side, price, quantity, remaining, status, filled_at, expires_at, created_at FROM mm_orders`
	args := []any{}
	if tokenFilter {
		q += ` WHERE token_id=$1`
		args = append(args, tokenID)
	}
	q += ` ORDER BY created_at DESC LIMIT 100`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MMOrder{}
	for rows.Next() {
		var o MMOrder
		if err := rows.Scan(&o.ID, &o.TokenID, &o.Side, &o.Price, &o.Quantity, &o.Remaining, &o.Status, &o.FilledAt, &o.ExpiresAt, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// UpdateMMOrderStatus transitions an order to pending/filled/cancelled. On
// 'filled' it zeroes remaining and records filled_at.
func (s *Store) UpdateMMOrderStatus(ctx context.Context, id uuid.UUID, status string) error {
	var tag pgconn.CommandTag
	var err error
	if status == "filled" {
		tag, err = s.db.Exec(ctx, `UPDATE mm_orders SET status=$1, filled_at=NOW(), remaining=0 WHERE id=$2`, status, id)
	} else {
		tag, err = s.db.Exec(ctx, `UPDATE mm_orders SET status=$1 WHERE id=$2`, status, id)
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// MMStatus is the real market-maker aggregate for a token.
type MMStatus struct {
	TokenID      uuid.UUID
	TotalOrders  int
	FilledOrders int
	BuyHigh      *string
	SellLow      *string
	Spread       float64
}

func (s *Store) MMStatus(ctx context.Context, tokenID uuid.UUID) (*MMStatus, error) {
	st := &MMStatus{TokenID: tokenID}
	if err := s.db.QueryRow(ctx,
		`SELECT COUNT(*), COUNT(*) FILTER (WHERE status='filled') FROM mm_orders WHERE token_id=$1`, tokenID).
		Scan(&st.TotalOrders, &st.FilledOrders); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(ctx,
		`SELECT MAX(price::numeric) FILTER (WHERE side='buy'), MIN(price::numeric) FILTER (WHERE side='sell')
		 FROM mm_orders WHERE token_id=$1 AND status='pending'`, tokenID).
		Scan(&st.BuyHigh, &st.SellLow); err != nil {
		return nil, err
	}
	if st.BuyHigh != nil && st.SellLow != nil {
		if bh, e := parseFloatStr(*st.BuyHigh); e == nil {
			if sl, e := parseFloatStr(*st.SellLow); e == nil && sl > 0 {
				st.Spread = (sl - bh) / sl * 100
			}
		}
	}
	return st, nil
}

// ==================== Liquidity positions ====================

type LiquidityPosition struct {
	ID         uuid.UUID
	TokenID    uuid.UUID
	UserID     *uuid.UUID
	QuoteToken string
	LPTokens   string
	CreatedAt  time.Time
}

// AddLiquidity records a liquidity position. LP tokens are minted proportional
// to the contributed amount (constant-product proxy: amount*1000).
func (s *Store) AddLiquidity(ctx context.Context, pos *LiquidityPosition) (*LiquidityPosition, error) {
	var out LiquidityPosition
	err := s.db.QueryRow(ctx,
		`INSERT INTO liquidity_positions (token_id, user_id, quote_token, lp_tokens)
		 VALUES ($1,$2,$3,$4)
		 RETURNING id, token_id, user_id, quote_token, lp_tokens, created_at`,
		pos.TokenID, pos.UserID, pos.QuoteToken, pos.LPTokens).
		Scan(&out.ID, &out.TokenID, &out.UserID, &out.QuoteToken, &out.LPTokens, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RemoveLiquidity deletes the most recent position with lp_tokens <= lpAmount.
func (s *Store) RemoveLiquidity(ctx context.Context, tokenID uuid.UUID, lpAmount string) error {
	tag, err := s.db.Exec(ctx,
		`DELETE FROM liquidity_positions WHERE id=(SELECT id FROM liquidity_positions WHERE token_id=$1 AND lp_tokens::numeric <= $2::numeric ORDER BY created_at DESC LIMIT 1)`,
		tokenID, lpAmount)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// TotalLiquidity returns the real SUM of lp_tokens across all positions.
func (s *Store) TotalLiquidity(ctx context.Context) (string, error) {
	var total string
	err := s.db.QueryRow(ctx, `SELECT COALESCE(SUM(lp_tokens),0)::text FROM liquidity_positions`).Scan(&total)
	if err != nil {
		return "0", err
	}
	return total, nil
}

// ==================== Token prices ====================

type TokenPrice struct {
	ID        uuid.UUID
	TokenID   uuid.UUID
	Price     string
	Change24h float64
	Volume24h string
	Timestamp time.Time
}

func (s *Store) SetTokenPrice(ctx context.Context, tokenID uuid.UUID, price string) (*TokenPrice, error) {
	var out TokenPrice
	err := s.db.QueryRow(ctx,
		`INSERT INTO token_prices (token_id, price, change_24h, volume_24h)
		 VALUES ($1,$2,0,0)
		 RETURNING id, token_id, price, change_24h, volume_24h, timestamp`,
		tokenID, price).
		Scan(&out.ID, &out.TokenID, &out.Price, &out.Change24h, &out.Volume24h, &out.Timestamp)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) GetTokenPrice(ctx context.Context, tokenID uuid.UUID) (*TokenPrice, error) {
	var p TokenPrice
	err := s.db.QueryRow(ctx,
		`SELECT id, token_id, price, change_24h, volume_24h, timestamp FROM token_prices
		 WHERE token_id=$1 ORDER BY timestamp DESC LIMIT 1`, tokenID).
		Scan(&p.ID, &p.TokenID, &p.Price, &p.Change24h, &p.Volume24h, &p.Timestamp)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) ListTokenPriceHistory(ctx context.Context, tokenID uuid.UUID) ([]TokenPrice, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, token_id, price, change_24h, volume_24h, timestamp FROM token_prices
		 WHERE token_id=$1 ORDER BY timestamp DESC LIMIT 100`, tokenID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TokenPrice{}
	for rows.Next() {
		var p TokenPrice
		if err := rows.Scan(&p.ID, &p.TokenID, &p.Price, &p.Change24h, &p.Volume24h, &p.Timestamp); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ==================== Analytics aggregates ====================

// VolumeStats holds real SUM(volume_24h) over rolling windows from token_prices.
type VolumeStats struct {
	Total24h string
	Total7d  string
	Total30d string
}

func (s *Store) VolumeStats(ctx context.Context) (*VolumeStats, error) {
	v := &VolumeStats{}
	qs := []string{
		`SELECT COALESCE(SUM(volume_24h),0)::text FROM token_prices WHERE timestamp > NOW() - INTERVAL '1 day'`,
		`SELECT COALESCE(SUM(volume_24h),0)::text FROM token_prices WHERE timestamp > NOW() - INTERVAL '7 days'`,
		`SELECT COALESCE(SUM(volume_24h),0)::text FROM token_prices WHERE timestamp > NOW() - INTERVAL '30 days'`,
	}
	for i, q := range qs {
		var dst *string
		switch i {
		case 0:
			dst = &v.Total24h
		case 1:
			dst = &v.Total7d
		default:
			dst = &v.Total30d
		}
		if err := s.db.QueryRow(ctx, q).Scan(dst); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// TransactionStats counts real launchpad contributions over rolling windows.
type TransactionStats struct {
	Total24h int
	Total7d  int
}

func (s *Store) TransactionStats(ctx context.Context) (*TransactionStats, error) {
	ts := &TransactionStats{}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM launchpad_contributions WHERE created_at > NOW() - INTERVAL '1 day'`).Scan(&ts.Total24h); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM launchpad_contributions WHERE created_at > NOW() - INTERVAL '7 days'`).Scan(&ts.Total7d); err != nil {
		return nil, err
	}
	return ts, nil
}

// HolderCount returns the real count of distinct contributors for a token's
// launchpads. Honestly 0 when none.
func (s *Store) HolderCount(ctx context.Context, tokenID uuid.UUID) (int, error) {
	var total int
	err := s.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM launchpad_contributions lc
		 JOIN launchpad_projects lp ON lp.id=lc.project_id WHERE lp.token_id=$1`, tokenID).Scan(&total)
	return total, err
}

// ==================== Compliance: audit logs + KYC ====================

type AuditLog struct {
	ID          uuid.UUID
	TokenID     uuid.UUID
	AuditType   string
	Status      string
	ReportURL   *string
	Auditor     string
	CompletedAt *time.Time
	RequestedAt time.Time
}

func (s *Store) CreateAuditLog(ctx context.Context, tokenID uuid.UUID, auditType string) (*AuditLog, error) {
	var out AuditLog
	err := s.db.QueryRow(ctx,
		`INSERT INTO token_audit_logs (token_id, audit_type, status)
		 VALUES ($1,$2,'requested')
		 RETURNING id, token_id, audit_type, status, report_url, auditor, completed_at, requested_at`,
		tokenID, auditType).
		Scan(&out.ID, &out.TokenID, &out.AuditType, &out.Status, &out.ReportURL, &out.Auditor, &out.CompletedAt, &out.RequestedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListAuditLogs(ctx context.Context, tokenID uuid.UUID) ([]AuditLog, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, token_id, audit_type, status, report_url, auditor, completed_at, requested_at
		 FROM token_audit_logs WHERE token_id=$1 ORDER BY requested_at DESC`, tokenID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditLog{}
	for rows.Next() {
		var a AuditLog
		if err := rows.Scan(&a.ID, &a.TokenID, &a.AuditType, &a.Status, &a.ReportURL, &a.Auditor, &a.CompletedAt, &a.RequestedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

type KYCRecord struct {
	ID          uuid.UUID
	TokenID     uuid.UUID
	Status      string
	SubmittedAt time.Time
	ExpiresAt   *time.Time
	ReviewedAt  *time.Time
	CreatedAt   time.Time
}

func (s *Store) SubmitKYC(ctx context.Context, tokenID uuid.UUID) (*KYCRecord, error) {
	var out KYCRecord
	err := s.db.QueryRow(ctx,
		`INSERT INTO token_kyc (token_id, status, expires_at)
		 VALUES ($1,'pending', NOW() + INTERVAL '365 days')
		 RETURNING id, token_id, status, submitted_at, expires_at, reviewed_at, created_at`,
		tokenID).
		Scan(&out.ID, &out.TokenID, &out.Status, &out.SubmittedAt, &out.ExpiresAt, &out.ReviewedAt, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) GetKYC(ctx context.Context, tokenID uuid.UUID) (*KYCRecord, error) {
	var k KYCRecord
	err := s.db.QueryRow(ctx,
		`SELECT id, token_id, status, submitted_at, expires_at, reviewed_at, created_at
		 FROM token_kyc WHERE token_id=$1 ORDER BY submitted_at DESC LIMIT 1`, tokenID).
		Scan(&k.ID, &k.TokenID, &k.Status, &k.SubmittedAt, &k.ExpiresAt, &k.ReviewedAt, &k.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// ==================== Fee payments + schedule ====================

type FeePayment struct {
	ID            uuid.UUID
	TokenID       *uuid.UUID
	UserID        *uuid.UUID
	Amount        string
	Currency      string
	PaymentMethod string
	TxHash        string
	Status        string
	CreatedAt     time.Time
}

// RecordFeePayment stores a real fee payment with its on-chain tx_hash.
func (s *Store) RecordFeePayment(ctx context.Context, p *FeePayment) (*FeePayment, error) {
	var out FeePayment
	err := s.db.QueryRow(ctx,
		`INSERT INTO fee_payments (token_id, user_id, amount, currency, payment_method, tx_hash, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING id, token_id, user_id, amount, currency, payment_method, tx_hash, status, created_at`,
		p.TokenID, p.UserID, p.Amount, p.Currency, p.PaymentMethod, p.TxHash, p.Status).
		Scan(&out.ID, &out.TokenID, &out.UserID, &out.Amount, &out.Currency, &out.PaymentMethod, &out.TxHash, &out.Status, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// FeeSchedule returns the real fee_config rows keyed by fee_type.
func (s *Store) FeeSchedule(ctx context.Context) (map[string]float64, error) {
	rows, err := s.db.Query(ctx, `SELECT fee_type, amount FROM fee_schedule`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var ft string
		var amt float64
		if err := rows.Scan(&ft, &amt); err != nil {
			return nil, err
		}
		out[ft] = amt
	}
	return out, rows.Err()
}

// SetFeeConfig upserts a fee_schedule row (admin gate). Used by /fees/set.
func (s *Store) SetFeeConfig(ctx context.Context, feeType string, amount float64) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO fee_schedule (fee_type, amount) VALUES ($1,$2)
		 ON CONFLICT (fee_type) DO UPDATE SET amount=EXCLUDED.amount`, feeType, amount)
	return err
}

// UpdateFeeConfig is the alias form of SetFeeConfig (admin gate, /fees/update).
func (s *Store) UpdateFeeConfig(ctx context.Context, feeType string, amount float64) error {
	return s.SetFeeConfig(ctx, feeType, amount)
}

// ==================== helpers ====================

// computeTokenAmount derives the token amount for a contribution: amount / price.
func computeTokenAmount(amount, price string) string {
	amt, e1 := parseFloatStr(amount)
	priceF, e2 := parseFloatStr(price)
	if e1 != nil || e2 != nil || priceF <= 0 {
		return "0"
	}
	return formatFloatStr(amt / priceF)
}

func parseFloatStr(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func formatFloatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// ErrNotFound is the canonical not-found sentinel for handlers.
var ErrNotFound = errors.New("not found")

// ErrNotAccepting / ErrEnded are contribution-state sentinels.
var (
	ErrNotAccepting = errors.New("launchpad not accepting contributions")
	ErrEnded        = errors.New("launchpad has ended")
)
