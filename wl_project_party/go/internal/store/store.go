// Package store provides PostgreSQL persistence for the standalone
// WL-ProjectParty backend. It owns its own database (wl_projectparty) —
// independent of TigerWallet cloud. Tables: users, tokens, listings,
// launchpad_projects, participations, market_making_configs, fee_configs,
// favorites.
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
	if err := s.migrate(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() { s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email VARCHAR(255) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL, role VARCHAR(32) DEFAULT 'user', created_at TIMESTAMPTZ DEFAULT NOW())`,
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
		role = "user"
	}
	var u User
	err := s.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1,$2,$3) RETURNING id, email, password_hash, role, created_at`,
		email, passwordHash, role).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, role, created_at FROM users WHERE email=$1`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, role, created_at FROM users WHERE id=$1`, id).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
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

// ErrNotFound is the canonical not-found sentinel for handlers.
var ErrNotFound = errors.New("not found")
