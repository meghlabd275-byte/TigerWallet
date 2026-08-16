package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Store is the PostgreSQL-backed persistence layer for the license control
// plane. All state lives in PostgreSQL; Redis is used only for real-time
// command fan-out to WL products (see hub.go). No in-memory maps, no SQLite.
type Store struct {
	db *pgxpool.Pool
}

func New(dbURL string) (*Store, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("connect pg: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping pg: %w", err)
	}
	s := &Store{db: pool}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() { s.db.Close() }

// DB exposes the underlying pool (used by the bootstrap helper).
func (s *Store) DB() *pgxpool.Pool { return s.db }

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,
		// SuperAdmin operator accounts (the TigerWallet-side admins who govern WL).
		`CREATE TABLE IF NOT EXISTS sa_admins (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(32) NOT NULL DEFAULT 'superadmin',
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			last_login_at TIMESTAMPTZ
		)`,
		// White-label clients (tenants). One row per WL customer.
		`CREATE TABLE IF NOT EXISTS wl_clients (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(100) UNIQUE NOT NULL,
			contact_email VARCHAR(255) NOT NULL,
			tier VARCHAR(32) NOT NULL DEFAULT 'basic',
			status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending|approved|suspended|revoked|halted
			branding JSONB DEFAULT '{}',
			allowed_products TEXT[] DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			approved_at TIMESTAMPTZ,
			suspended_at TIMESTAMPTZ,
			revoked_at TIMESTAMPTZ,
			halted_at TIMESTAMPTZ
		)`,
		// Licenses issued to WL clients, one per (wl_client_id, product).
		`CREATE TABLE IF NOT EXISTS licenses (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wl_client_id UUID NOT NULL REFERENCES wl_clients(id) ON DELETE CASCADE,
			product VARCHAR(64) NOT NULL, -- master_wallet|user_wallet|bots|project_party|all
			plan VARCHAR(32) NOT NULL DEFAULT 'basic',
			status VARCHAR(32) NOT NULL DEFAULT 'active', -- active|suspended|revoked|expired|halted
			license_key VARCHAR(128) UNIQUE NOT NULL,
			valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			valid_until TIMESTAMPTZ NOT NULL,
			max_users INT NOT NULL DEFAULT 100,
			max_wallets INT NOT NULL DEFAULT 500,
			max_bots INT NOT NULL DEFAULT 50,
			features TEXT[] DEFAULT '{}',
			issued_by UUID REFERENCES sa_admins(id),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			suspended_at TIMESTAMPTZ,
			revoked_at TIMESTAMPTZ,
			halted_at TIMESTAMPTZ,
			UNIQUE(wl_client_id, product)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_licenses_client ON licenses(wl_client_id)`,
		`CREATE INDEX IF NOT EXISTS idx_licenses_status ON licenses(status)`,
		// Per-fetcher feature flags. This is the unified SuperAdmin governance
		// store: keyed by (wl_client_id, product, fetcher). When a flag is
		// disabled, the WL product's fetcher must refuse to serve. A wildcard
		// fetcher '*' disables the whole product.
		`CREATE TABLE IF NOT EXISTS feature_flags (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wl_client_id UUID NOT NULL REFERENCES wl_clients(id) ON DELETE CASCADE,
			product VARCHAR(64) NOT NULL,
			fetcher VARCHAR(128) NOT NULL DEFAULT '*', -- fetcher name or '*' for whole product
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			updated_by UUID REFERENCES sa_admins(id),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(wl_client_id, product, fetcher)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_flags_client_product ON feature_flags(wl_client_id, product)`,
		// WL product heartbeat / connection state.
		`CREATE TABLE IF NOT EXISTS wl_product_state (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wl_client_id UUID NOT NULL REFERENCES wl_clients(id) ON DELETE CASCADE,
			product VARCHAR(64) NOT NULL,
			instance_id VARCHAR(128) NOT NULL, -- external instance identifier
			is_connected BOOLEAN DEFAULT FALSE,
			last_heartbeat TIMESTAMPTZ,
			heartbeat_latency_ms INT DEFAULT 0,
			version VARCHAR(64),
			hostname VARCHAR(255),
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(wl_client_id, product, instance_id)
		)`,
		// Pending remote commands queued for a WL product (delivered on heartbeat).
		`CREATE TABLE IF NOT EXISTS remote_commands (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wl_client_id UUID NOT NULL REFERENCES wl_clients(id) ON DELETE CASCADE,
			product VARCHAR(64) NOT NULL,
			command VARCHAR(64) NOT NULL, -- disable|enable|halt|resume|clear_cache|sync_flags|reload
			params JSONB DEFAULT '{}',
			status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending|delivered|executed|failed|expired
			issued_by UUID REFERENCES sa_admins(id),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			delivered_at TIMESTAMPTZ,
			executed_at TIMESTAMPTZ,
			expires_at TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_commands_pending ON remote_commands(wl_client_id, product, status)`,
		// SuperAdmin-gated audit trail (immutable).
		`CREATE TABLE IF NOT EXISTS sa_audit_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			admin_id UUID REFERENCES sa_admins(id),
			action VARCHAR(100) NOT NULL,
			resource_type VARCHAR(64) NOT NULL,
			resource_id VARCHAR(255),
			details JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sa_audit_admin ON sa_audit_logs(admin_id)`,
		// Two-party withdrawal approvals: every fund/revenue exit requires both
		// a WL-side approver AND a SuperAdmin co-signer.
		`CREATE TABLE IF NOT EXISTS withdrawal_approvals (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wl_client_id UUID NOT NULL REFERENCES wl_clients(id) ON DELETE CASCADE,
			product VARCHAR(64) NOT NULL,
			resource_type VARCHAR(64) NOT NULL, -- treasury|revenue|fee|master_wallet
			resource_id VARCHAR(255) NOT NULL,
			amount_wei TEXT NOT NULL,
			to_address VARCHAR(255) NOT NULL,
			chain_id BIGINT NOT NULL,
			wl_approver_id UUID,
			wl_approved_at TIMESTAMPTZ,
			superadmin_approver_id UUID REFERENCES sa_admins(id),
			superadmin_approved_at TIMESTAMPTZ,
			status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending|wl_approved|approved|rejected|executed|failed
			created_at TIMESTAMPTZ DEFAULT NOW(),
			executed_at TIMESTAMPTZ,
			tx_hash VARCHAR(255)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_withdrawal_approvals_client ON withdrawal_approvals(wl_client_id)`,
		`CREATE INDEX IF NOT EXISTS idx_withdrawal_approvals_status ON withdrawal_approvals(status)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(ctx, q); err != nil {
			return fmt.Errorf("migration stmt failed: %w\nstmt: %s", err, q)
		}
	}
	return nil
}

// --- SuperAdmin account ops ---

func (s *Store) CreateSuperAdmin(ctx context.Context, email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx,
		`INSERT INTO sa_admins (email, password_hash, role) VALUES ($1,$2,'superadmin')
		 ON CONFLICT (email) DO NOTHING`, email, string(hash))
	return err
}

func (s *Store) VerifySuperAdmin(ctx context.Context, email, password string) (uuid.UUID, string, error) {
	var id uuid.UUID
	var hash, role string
	err := s.db.QueryRow(ctx,
		`SELECT id, password_hash, role FROM sa_admins WHERE email=$1 AND is_active=true`, email).
		Scan(&id, &hash, &role)
	if err != nil {
		return uuid.Nil, "", errors.New("invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return uuid.Nil, "", errors.New("invalid credentials")
	}
	_, _ = s.db.Exec(ctx, `UPDATE sa_admins SET last_login_at=NOW() WHERE id=$1`, id)
	return id, role, nil
}

// --- WL client lifecycle ---

type WLClient struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	ContactEmail    string    `json:"contact_email"`
	Tier            string    `json:"tier"`
	Status          string    `json:"status"`
	Branding        map[string]any `json:"branding"`
	AllowedProducts []string  `json:"allowed_products"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (s *Store) CreateWLClient(ctx context.Context, name, slug, contactEmail, tier string, products []string) (*WLClient, error) {
	id := uuid.New()
	bj, _ := json.Marshal(map[string]any{})
	_, err := s.db.Exec(ctx,
		`INSERT INTO wl_clients (id, name, slug, contact_email, tier, status, branding, allowed_products)
		 VALUES ($1,$2,$3,$4,$5,'pending',$6,$7)`,
		id, name, slug, contactEmail, tier, bj, products)
	if err != nil {
		return nil, err
	}
	return s.GetWLClient(ctx, id)
}

func (s *Store) GetWLClient(ctx context.Context, id uuid.UUID) (*WLClient, error) {
	c := &WLClient{}
	var branding []byte
	err := s.db.QueryRow(ctx,
		`SELECT id, name, slug, contact_email, tier, status, branding, allowed_products, created_at, updated_at
		 FROM wl_clients WHERE id=$1`, id).
		Scan(&c.ID, &c.Name, &c.Slug, &c.ContactEmail, &c.Tier, &c.Status, &branding, &c.AllowedProducts, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(branding, &c.Branding)
	return c, nil
}

// GetWLClientBySlug resolves a WL client by its public slug. Used by the
// public branding endpoint so a WL-branded app can fetch its branding config
// on startup without any credentials. Returns ErrWLClientNotFound when no row
// matches so the handler can return a clean 404 and the app can fall back to
// TigerWallet defaults.
func (s *Store) GetWLClientBySlug(ctx context.Context, slug string) (*WLClient, error) {
	c := &WLClient{}
	var branding []byte
	err := s.db.QueryRow(ctx,
		`SELECT id, name, slug, contact_email, tier, status, branding, allowed_products, created_at, updated_at
		 FROM wl_clients WHERE slug=$1`, slug).
		Scan(&c.ID, &c.Name, &c.Slug, &c.ContactEmail, &c.Tier, &c.Status, &branding, &c.AllowedProducts, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(branding, &c.Branding)
	return c, nil
}

func (s *Store) ListWLClients(ctx context.Context) ([]*WLClient, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, slug, contact_email, tier, status, branding, allowed_products, created_at, updated_at
		 FROM wl_clients ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*WLClient
	for rows.Next() {
		c := &WLClient{}
		var branding []byte
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.ContactEmail, &c.Tier, &c.Status, &branding, &c.AllowedProducts, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(branding, &c.Branding)
		out = append(out, c)
	}
	return out, nil
}

// SetWLClientStatus transitions a WL client's status. CRITICAL: the 'active'
// (resume) transition is ONLY permitted from the SuperAdmin side — a WL client
// can never move itself out of suspended/halted/revoked. This is enforced at
// the handler layer (SuperAdmin-only routes), and the status column itself
// refuses the active transition unless caller sets allowResume=true.
func (s *Store) SetWLClientStatus(ctx context.Context, id uuid.UUID, status, adminID string, allowResume bool) error {
	switch status {
	case "approved":
		_, err := s.db.Exec(ctx, `UPDATE wl_clients SET status='approved', approved_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
		return err
	case "suspended":
		_, err := s.db.Exec(ctx, `UPDATE wl_clients SET status='suspended', suspended_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
		return err
	case "halted":
		_, err := s.db.Exec(ctx, `UPDATE wl_clients SET status='halted', halted_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
		return err
	case "revoked":
		_, err := s.db.Exec(ctx, `UPDATE wl_clients SET status='revoked', revoked_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
		return err
	case "active":
		if !allowResume {
			return errors.New("WL client cannot self-resume: only SuperAdmin may reactivate a product")
		}
		_, err := s.db.Exec(ctx, `UPDATE wl_clients SET status='active', suspended_at=NULL, halted_at=NULL, revoked_at=NULL, updated_at=NOW() WHERE id=$1`, id)
		return err
	default:
		return fmt.Errorf("unknown status %q", status)
	}
}

func (s *Store) UpdateWLClient(ctx context.Context, id uuid.UUID, tier string, products []string) error {
	_, err := s.db.Exec(ctx, `UPDATE wl_clients SET tier=$1, allowed_products=$2, updated_at=NOW() WHERE id=$3`, tier, products, id)
	return err
}

// UpdateWLClientBranding persists the white-label branding JSONB for a WL
// client. SuperAdmin sets this when onboarding/rebranding a WL client; the
// public branding endpoint projects it to WL-branded apps on startup. A nil
// map clears branding (app falls back to TigerWallet defaults).
func (s *Store) UpdateWLClientBranding(ctx context.Context, id uuid.UUID, branding map[string]any) error {
	bj, err := json.Marshal(branding)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `UPDATE wl_clients SET branding=$1, updated_at=NOW() WHERE id=$2`, bj, id)
	return err
}

// --- License ops ---

type License struct {
	ID           uuid.UUID `json:"id"`
	WLClientID   uuid.UUID `json:"wl_client_id"`
	Product      string    `json:"product"`
	Plan         string    `json:"plan"`
	Status       string    `json:"status"`
	LicenseKey   string    `json:"license_key"`
	ValidFrom    time.Time `json:"valid_from"`
	ValidUntil   time.Time `json:"valid_until"`
	MaxUsers     int       `json:"max_users"`
	MaxWallets   int       `json:"max_wallets"`
	MaxBots      int       `json:"max_bots"`
	Features     []string  `json:"features"`
	IssuedBy     *uuid.UUID `json:"issued_by"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s *Store) CreateLicense(ctx context.Context, wlClientID uuid.UUID, product, plan, key string,
	validUntil time.Time, maxU, maxW, maxB int, features []string, issuedBy uuid.UUID) (*License, error) {
	id := uuid.New()
	_, err := s.db.Exec(ctx,
		`INSERT INTO licenses (id, wl_client_id, product, plan, status, license_key, valid_until, max_users, max_wallets, max_bots, features, issued_by)
		 VALUES ($1,$2,$3,$4,'active',$5,$6,$7,$8,$9,$10,$11)`,
		id, wlClientID, product, plan, key, validUntil, maxU, maxW, maxB, features, issuedBy)
	if err != nil {
		return nil, err
	}
	return s.GetLicenseByKey(ctx, key)
}

func (s *Store) GetLicenseByKey(ctx context.Context, key string) (*License, error) {
	l := &License{}
	err := s.db.QueryRow(ctx,
		`SELECT id, wl_client_id, product, plan, status, license_key, valid_from, valid_until, max_users, max_wallets, max_bots, features, issued_by, created_at
		 FROM licenses WHERE license_key=$1`, key).
		Scan(&l.ID, &l.WLClientID, &l.Product, &l.Plan, &l.Status, &l.LicenseKey, &l.ValidFrom, &l.ValidUntil, &l.MaxUsers, &l.MaxWallets, &l.MaxBots, &l.Features, &l.IssuedBy, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (s *Store) GetLicense(ctx context.Context, id uuid.UUID) (*License, error) {
	l := &License{}
	err := s.db.QueryRow(ctx,
		`SELECT id, wl_client_id, product, plan, status, license_key, valid_from, valid_until, max_users, max_wallets, max_bots, features, issued_by, created_at
		 FROM licenses WHERE id=$1`, id).
		Scan(&l.ID, &l.WLClientID, &l.Product, &l.Plan, &l.Status, &l.LicenseKey, &l.ValidFrom, &l.ValidUntil, &l.MaxUsers, &l.MaxWallets, &l.MaxBots, &l.Features, &l.IssuedBy, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (s *Store) ListLicenses(ctx context.Context, wlClientID *uuid.UUID) ([]*License, error) {
	q := `SELECT id, wl_client_id, product, plan, status, license_key, valid_from, valid_until, max_users, max_wallets, max_bots, features, issued_by, created_at FROM licenses`
	args := []any{}
	if wlClientID != nil {
		q += ` WHERE wl_client_id=$1 ORDER BY created_at DESC`
		args = append(args, *wlClientID)
	} else {
		q += ` ORDER BY created_at DESC`
	}
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*License
	for rows.Next() {
		l := &License{}
		if err := rows.Scan(&l.ID, &l.WLClientID, &l.Product, &l.Plan, &l.Status, &l.LicenseKey, &l.ValidFrom, &l.ValidUntil, &l.MaxUsers, &l.MaxWallets, &l.MaxBots, &l.Features, &l.IssuedBy, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

// SetLicenseStatus transitions a license. As with WL clients, the 'active'
// transition requires allowResume=true and is SuperAdmin-only.
func (s *Store) SetLicenseStatus(ctx context.Context, id uuid.UUID, status string, allowResume bool) error {
	switch status {
	case "suspended":
		_, err := s.db.Exec(ctx, `UPDATE licenses SET status='suspended', suspended_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
		return err
	case "halted":
		_, err := s.db.Exec(ctx, `UPDATE licenses SET status='halted', halted_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
		return err
	case "revoked":
		_, err := s.db.Exec(ctx, `UPDATE licenses SET status='revoked', revoked_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
		return err
	case "active":
		if !allowResume {
			return errors.New("license cannot self-resume: only SuperAdmin may reactivate")
		}
		_, err := s.db.Exec(ctx, `UPDATE licenses SET status='active', suspended_at=NULL, halted_at=NULL, revoked_at=NULL, updated_at=NOW() WHERE id=$1`, id)
		return err
	default:
		return fmt.Errorf("unknown status %q", status)
	}
}

// --- Feature flags (per-fetcher granularity) ---

type FeatureFlag struct {
	ID         uuid.UUID `json:"id"`
	WLClientID uuid.UUID `json:"wl_client_id"`
	Product    string    `json:"product"`
	Fetcher    string    `json:"fetcher"`
	Enabled    bool      `json:"enabled"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (s *Store) SetFeatureFlag(ctx context.Context, wlClientID uuid.UUID, product, fetcher string, enabled bool, adminID uuid.UUID) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO feature_flags (id, wl_client_id, product, fetcher, enabled, updated_by, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,NOW())
		 ON CONFLICT (wl_client_id, product, fetcher) DO UPDATE SET enabled=$5, updated_by=$6, updated_at=NOW()`,
		uuid.New(), wlClientID, product, fetcher, enabled, adminID)
	return err
}

func (s *Store) ListFeatureFlags(ctx context.Context, wlClientID uuid.UUID, product string) ([]*FeatureFlag, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, wl_client_id, product, fetcher, enabled, updated_at FROM feature_flags
		 WHERE wl_client_id=$1 AND ($2='' OR product=$2) ORDER BY product, fetcher`, wlClientID, product)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*FeatureFlag
	for rows.Next() {
		f := &FeatureFlag{}
		if err := rows.Scan(&f.ID, &f.WLClientID, &f.Product, &f.Fetcher, &f.Enabled, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// IsFetcherEnabled returns false if either the whole-product flag ('*') or the
// specific fetcher flag is disabled. Absent flags default to enabled=true
// (the WL product works until SuperAdmin explicitly disables something).
func (s *Store) IsFetcherEnabled(ctx context.Context, wlClientID uuid.UUID, product, fetcher string) (bool, error) {
	var productOK, fetcherOK *bool
	err := s.db.QueryRow(ctx,
		`SELECT (SELECT enabled FROM feature_flags WHERE wl_client_id=$1 AND product=$2 AND fetcher='*'),
		        (SELECT enabled FROM feature_flags WHERE wl_client_id=$1 AND product=$2 AND fetcher=$3)`,
		wlClientID, product, fetcher).Scan(&productOK, &fetcherOK)
	if err != nil {
		return false, err
	}
	if productOK != nil && !*productOK {
		return false, nil
	}
	if fetcherOK != nil && !*fetcherOK {
		return false, nil
	}
	return true, nil
}

// --- Heartbeat / product state ---

func (s *Store) RecordHeartbeat(ctx context.Context, wlClientID uuid.UUID, product, instanceID string, latency int, version, hostname string, meta map[string]any) error {
	mj, _ := json.Marshal(meta)
	_, err := s.db.Exec(ctx,
		`INSERT INTO wl_product_state (id, wl_client_id, product, instance_id, is_connected, last_heartbeat, heartbeat_latency_ms, version, hostname, metadata, updated_at)
		 VALUES ($1,$2,$3,$4,TRUE,NOW(),$5,$6,$7,$8,NOW())
		 ON CONFLICT (wl_client_id, product, instance_id) DO UPDATE SET is_connected=TRUE, last_heartbeat=NOW(), heartbeat_latency_ms=$5, version=$6, hostname=$7, metadata=$8, updated_at=NOW()`,
		uuid.New(), wlClientID, product, instanceID, latency, version, hostname, mj)
	return err
}

// IsProductAlive returns whether the WL product may serve traffic: the WL
// client must be approved/active, the license must be active, and the last
// heartbeat must be within the timeout window. This is the authoritative
// fail-closed check.
func (s *Store) IsProductAlive(ctx context.Context, wlClientID uuid.UUID, product string, heartbeatTimeout time.Duration) (bool, string, error) {
	var clientStatus string
	err := s.db.QueryRow(ctx, `SELECT status FROM wl_clients WHERE id=$1`, wlClientID).Scan(&clientStatus)
	if err != nil {
		return false, "wl_client_not_found", err
	}
	if clientStatus != "approved" && clientStatus != "active" {
		return false, "wl_client_" + clientStatus, nil
	}
	var licStatus string
	err = s.db.QueryRow(ctx, `SELECT status FROM licenses WHERE wl_client_id=$1 AND product IN ($2,'all')`, wlClientID, product).Scan(&licStatus)
	if err != nil {
		// no license row -> not authorized
		return false, "no_license", nil
	}
	if licStatus != "active" {
		return false, "license_" + licStatus, nil
	}
	var lastHB *time.Time
	err = s.db.QueryRow(ctx,
		`SELECT MAX(last_heartbeat) FROM wl_product_state WHERE wl_client_id=$1 AND product=$2`, wlClientID, product).Scan(&lastHB)
	if err != nil || lastHB == nil {
		return false, "no_heartbeat", nil
	}
	if time.Since(*lastHB) > heartbeatTimeout {
		return false, "heartbeat_stale", nil
	}
	return true, "alive", nil
}

// --- Pending commands (delivered on heartbeat) ---

type RemoteCommand struct {
	ID         uuid.UUID `json:"id"`
	WLClientID uuid.UUID `json:"wl_client_id"`
	Product    string    `json:"product"`
	Command    string    `json:"command"`
	Params     map[string]any `json:"params"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Store) QueueCommand(ctx context.Context, wlClientID uuid.UUID, product, command string, params map[string]any, adminID uuid.UUID) (*RemoteCommand, error) {
	id := uuid.New()
	pj, _ := json.Marshal(params)
	_, err := s.db.Exec(ctx,
		`INSERT INTO remote_commands (id, wl_client_id, product, command, params, status, issued_by, expires_at)
		 VALUES ($1,$2,$3,$4,$5,'pending',$6, NOW() + INTERVAL '1 hour')`,
		id, wlClientID, product, command, pj, adminID)
	if err != nil {
		return nil, err
	}
	return &RemoteCommand{ID: id, WLClientID: wlClientID, Product: product, Command: command, Params: params, Status: "pending", CreatedAt: time.Now()}, nil
}

// DeliverPendingCommands returns pending commands for a product and marks them delivered.
func (s *Store) DeliverPendingCommands(ctx context.Context, wlClientID uuid.UUID, product string) ([]*RemoteCommand, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, command, params, created_at FROM remote_commands
		 WHERE wl_client_id=$1 AND product=$2 AND status='pending' AND (expires_at IS NULL OR expires_at > NOW())
		 ORDER BY created_at ASC`, wlClientID, product)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RemoteCommand
	var ids []uuid.UUID
	for rows.Next() {
		rc := &RemoteCommand{WLClientID: wlClientID, Product: product, Status: "delivered"}
		var pj []byte
		if err := rows.Scan(&rc.ID, &rc.Command, &pj, &rc.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(pj, &rc.Params)
		out = append(out, rc)
		ids = append(ids, rc.ID)
	}
	for _, id := range ids {
		_, _ = s.db.Exec(ctx, `UPDATE remote_commands SET status='delivered', delivered_at=NOW() WHERE id=$1`, id)
	}
	return out, nil
}

func (s *Store) MarkCommandExecuted(ctx context.Context, id uuid.UUID, result string) error {
	_, err := s.db.Exec(ctx, `UPDATE remote_commands SET status='executed', executed_at=NOW() WHERE id=$1`, id)
	return err
}

// --- Audit ---

func (s *Store) Audit(ctx context.Context, adminID uuid.UUID, action, rtype, rid string, details map[string]any) {
	dj, _ := json.Marshal(details)
	_, _ = s.db.Exec(ctx,
		`INSERT INTO sa_audit_logs (admin_id, action, resource_type, resource_id, details) VALUES ($1,$2,$3,$4,$5)`,
		adminID, action, rtype, rid, dj)
}

// --- Withdrawal approvals (two-party) ---

type WithdrawalApproval struct {
	ID                  uuid.UUID `json:"id"`
	WLClientID          uuid.UUID `json:"wl_client_id"`
	Product             string    `json:"product"`
	ResourceType        string    `json:"resource_type"`
	ResourceID          string    `json:"resource_id"`
	AmountWei           string    `json:"amount_wei"`
	ToAddress           string    `json:"to_address"`
	ChainID             int64     `json:"chain_id"`
	WLApproverID        *uuid.UUID `json:"wl_approver_id"`
	WLApprovedAt        *time.Time `json:"wl_approved_at"`
	SuperadminApproverID *uuid.UUID `json:"superadmin_approver_id"`
	SuperadminApprovedAt *time.Time `json:"superadmin_approved_at"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	TxHash              string    `json:"tx_hash"`
}

func (s *Store) CreateWithdrawalApproval(ctx context.Context, wa *WithdrawalApproval) error {
	id := uuid.New()
	wa.ID = id
	_, err := s.db.Exec(ctx,
		`INSERT INTO withdrawal_approvals (id, wl_client_id, product, resource_type, resource_id, amount_wei, to_address, chain_id, wl_approver_id, wl_approved_at, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'wl_approved')`,
		id, wa.WLClientID, wa.Product, wa.ResourceType, wa.ResourceID, wa.AmountWei, wa.ToAddress, wa.ChainID, wa.WLApproverID, wa.WLApprovedAt)
	wa.Status = "wl_approved"
	return err
}

// SuperAdminApproveWithdrawal records the mandatory second (SuperAdmin) approval.
// Only after this is the withdrawal 'approved' and executable.
func (s *Store) SuperAdminApproveWithdrawal(ctx context.Context, id, adminID uuid.UUID) error {
	ct, err := s.db.Exec(ctx,
		`UPDATE withdrawal_approvals SET superadmin_approver_id=$1, superadmin_approved_at=NOW(), status='approved'
		 WHERE id=$2 AND status='wl_approved'`, adminID, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errors.New("withdrawal not found, not in wl_approved state, or already processed")
	}
	return nil
}

func (s *Store) SuperAdminRejectWithdrawal(ctx context.Context, id, adminID uuid.UUID, reason string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE withdrawal_approvals SET superadmin_approver_id=$1, status='rejected'
		 WHERE id=$2 AND status IN ('pending','wl_approved')`, adminID, id)
	return err
}

func (s *Store) MarkWithdrawalExecuted(ctx context.Context, id uuid.UUID, txHash string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE withdrawal_approvals SET status='executed', executed_at=NOW(), tx_hash=$1 WHERE id=$2 AND status='approved'`, txHash, id)
	return err
}

// IsWithdrawalApproved returns true ONLY when both parties have approved.
func (s *Store) IsWithdrawalApproved(ctx context.Context, id uuid.UUID) (bool, error) {
	var status string
	err := s.db.QueryRow(ctx, `SELECT status FROM withdrawal_approvals WHERE id=$1`, id).Scan(&status)
	if err != nil {
		return false, err
	}
	return status == "approved", nil
}

func (s *Store) ListWithdrawalApprovals(ctx context.Context, wlClientID *uuid.UUID) ([]*WithdrawalApproval, error) {
	q := `SELECT id, wl_client_id, product, resource_type, resource_id, amount_wei, to_address, chain_id, wl_approver_id, wl_approved_at, superadmin_approver_id, superadmin_approved_at, status, created_at, COALESCE(tx_hash,'')
		 FROM withdrawal_approvals`
	args := []any{}
	if wlClientID != nil {
		q += ` WHERE wl_client_id=$1 ORDER BY created_at DESC`
		args = append(args, *wlClientID)
	} else {
		q += ` ORDER BY created_at DESC`
	}
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*WithdrawalApproval
	for rows.Next() {
		wa := &WithdrawalApproval{}
		if err := rows.Scan(&wa.ID, &wa.WLClientID, &wa.Product, &wa.ResourceType, &wa.ResourceID, &wa.AmountWei, &wa.ToAddress, &wa.ChainID, &wa.WLApproverID, &wa.WLApprovedAt, &wa.SuperadminApproverID, &wa.SuperadminApprovedAt, &wa.Status, &wa.CreatedAt, &wa.TxHash); err != nil {
			return nil, err
		}
		out = append(out, wa)
	}
	return out, nil
}
