// PostgreSQL persistence for KYC applications, documents, AML checks and
// watchlist entries.
//
// Every read and write in this service goes through this store. Nothing is
// answered from a fixture: a missing application returns ErrNotFound, and a
// search that matches nothing returns an empty result.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("record not found")

// Store owns the PostgreSQL connection pool and the schema.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore connects to PostgreSQL and applies the schema.
func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	store := &Store{pool: pool}
	if err := store.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

// Close releases the connection pool.
func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// Ping reports whether the database is reachable.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// migrate creates the tables this service owns. It is additive and safe to run
// on every start.
func (s *Store) migrate(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS kyc_applications (
    id                   UUID PRIMARY KEY,
    user_id              UUID NOT NULL,
    tenant_id            UUID,
    level                TEXT NOT NULL,
    status               TEXT NOT NULL,
    rejection_reason     TEXT,
    reviewer_id          UUID,
    reviewed_at          TIMESTAMPTZ,
    first_name           TEXT NOT NULL DEFAULT '',
    last_name            TEXT NOT NULL DEFAULT '',
    date_of_birth        TEXT NOT NULL DEFAULT '',
    nationality          TEXT NOT NULL DEFAULT '',
    country_of_residence TEXT NOT NULL DEFAULT '',
    address              TEXT NOT NULL DEFAULT '',
    city                 TEXT NOT NULL DEFAULT '',
    state                TEXT NOT NULL DEFAULT '',
    postal_code          TEXT NOT NULL DEFAULT '',
    phone_number         TEXT NOT NULL DEFAULT '',
    sumsub_applicant_id  TEXT,
    aml_check_passed     BOOLEAN,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS kyc_applications_user_idx ON kyc_applications (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS kyc_applications_status_idx ON kyc_applications (status, created_at DESC);

CREATE TABLE IF NOT EXISTS kyc_documents (
    id             UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES kyc_applications (id) ON DELETE CASCADE,
    type           TEXT NOT NULL,
    side           TEXT NOT NULL,
    file_url       TEXT NOT NULL,
    file_hash      TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL,
    verified_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS kyc_documents_application_idx ON kyc_documents (application_id, created_at DESC);

CREATE TABLE IF NOT EXISTS aml_checks (
    id                  UUID PRIMARY KEY,
    user_id             UUID NOT NULL,
    application_id      UUID,
    status              TEXT NOT NULL,
    pep_match           BOOLEAN NOT NULL DEFAULT FALSE,
    sanction_match      BOOLEAN NOT NULL DEFAULT FALSE,
    adverse_media_match BOOLEAN NOT NULL DEFAULT FALSE,
    risk_score          INTEGER NOT NULL DEFAULT 0,
    risk_level          TEXT NOT NULL DEFAULT 'low',
    details             JSONB NOT NULL DEFAULT '{}'::jsonb,
    checked_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS aml_checks_user_idx ON aml_checks (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS watchlist_entries (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    country     TEXT NOT NULL DEFAULT '',
    list_type   TEXT NOT NULL,
    risk_level  TEXT NOT NULL,
    details     JSONB NOT NULL DEFAULT '{}'::jsonb,
    added_by    UUID,
    added_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS watchlist_entries_name_idx ON watchlist_entries (lower(name));

CREATE TABLE IF NOT EXISTS kyc_audit_log (
    id          BIGSERIAL PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id   UUID,
    action      TEXT NOT NULL,
    actor_id    TEXT,
    details     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS kyc_audit_log_created_idx ON kyc_audit_log (created_at DESC);
`
	_, err := s.pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("apply kyc schema: %w", err)
	}
	return nil
}

// CreateApplication inserts a new KYC application.
func (s *Store) CreateApplication(ctx context.Context, app *KYCApplication) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO kyc_applications (
			id, user_id, tenant_id, level, status, first_name, last_name,
			date_of_birth, nationality, country_of_residence, address, city,
			state, postal_code, phone_number, sumsub_applicant_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		app.ID, app.UserID, app.TenantID, app.Level, app.Status, app.FirstName, app.LastName,
		app.DateOfBirth, app.Nationality, app.CountryOfResidence, app.Address, app.City,
		app.State, app.PostalCode, app.PhoneNumber, app.SumSubApplicantID, app.CreatedAt, app.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert kyc application: %w", err)
	}
	return nil
}

// UpdateApplicationStatus records a review decision.
func (s *Store) UpdateApplicationStatus(ctx context.Context, id uuid.UUID, status string, reviewerID *uuid.UUID, reason *string) (*KYCApplication, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE kyc_applications
		SET status = $2, reviewer_id = COALESCE($3, reviewer_id), rejection_reason = $4,
		    reviewed_at = NOW(), updated_at = NOW()
		WHERE id = $1`, id, status, reviewerID, reason)
	if err != nil {
		return nil, fmt.Errorf("update kyc application: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetApplication(ctx, id)
}

// SetAMLCheckResult stores the outcome of an AML screening on the application.
func (s *Store) SetAMLCheckResult(ctx context.Context, id uuid.UUID, passed bool) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE kyc_applications SET aml_check_passed = $2, updated_at = NOW() WHERE id = $1`, id, passed)
	if err != nil {
		return fmt.Errorf("update aml result: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetApplication loads one application by id.
func (s *Store) GetApplication(ctx context.Context, id uuid.UUID) (*KYCApplication, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, user_id, tenant_id, level, status, rejection_reason, reviewer_id, reviewed_at,
		       first_name, last_name, date_of_birth, nationality, country_of_residence, address,
		       city, state, postal_code, phone_number, sumsub_applicant_id, aml_check_passed,
		       created_at, updated_at
		FROM kyc_applications WHERE id = $1`, id)
	return scanApplication(row)
}

// GetLatestApplicationForUser loads the most recent application for a user.
func (s *Store) GetLatestApplicationForUser(ctx context.Context, userID uuid.UUID) (*KYCApplication, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, user_id, tenant_id, level, status, rejection_reason, reviewer_id, reviewed_at,
		       first_name, last_name, date_of_birth, nationality, country_of_residence, address,
		       city, state, postal_code, phone_number, sumsub_applicant_id, aml_check_passed,
		       created_at, updated_at
		FROM kyc_applications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, userID)
	return scanApplication(row)
}

// ListApplications returns applications filtered by status.
func (s *Store) ListApplications(ctx context.Context, status string, limit, offset int) ([]KYCApplication, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	where := ""
	args := []interface{}{}
	if status != "" {
		where = " WHERE status = $1"
		args = append(args, status)
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM kyc_applications"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count applications: %w", err)
	}

	query := `SELECT id, user_id, tenant_id, level, status, rejection_reason, reviewer_id, reviewed_at,
	       first_name, last_name, date_of_birth, nationality, country_of_residence, address,
	       city, state, postal_code, phone_number, sumsub_applicant_id, aml_check_passed,
	       created_at, updated_at
	FROM kyc_applications` + where +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d OFFSET %d", limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	applications := make([]KYCApplication, 0, limit)
	for rows.Next() {
		app, err := scanApplication(rows)
		if err != nil {
			return nil, 0, err
		}
		applications = append(applications, *app)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return applications, total, nil
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanApplication reads one application row.
func scanApplication(row rowScanner) (*KYCApplication, error) {
	var app KYCApplication
	err := row.Scan(&app.ID, &app.UserID, &app.TenantID, &app.Level, &app.Status,
		&app.RejectionReason, &app.ReviewerID, &app.ReviewedAt, &app.FirstName, &app.LastName,
		&app.DateOfBirth, &app.Nationality, &app.CountryOfResidence, &app.Address, &app.City,
		&app.State, &app.PostalCode, &app.PhoneNumber, &app.SumSubApplicantID,
		&app.AMLCheckPassed, &app.CreatedAt, &app.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan kyc application: %w", err)
	}
	return &app, nil
}

// CreateDocument records an uploaded identity document.
func (s *Store) CreateDocument(ctx context.Context, doc *KYCDocument) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO kyc_documents (id, application_id, type, side, file_url, file_hash, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		doc.ID, doc.ApplicationID, doc.Type, doc.Side, doc.FileURL, doc.FileHash, doc.Status, doc.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert document: %w", err)
	}
	return nil
}

// ListDocuments returns the documents attached to an application.
func (s *Store) ListDocuments(ctx context.Context, applicationID uuid.UUID) ([]KYCDocument, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, application_id, type, side, file_url, file_hash, status, verified_at, created_at
		FROM kyc_documents WHERE application_id = $1 ORDER BY created_at DESC`, applicationID)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	documents := []KYCDocument{}
	for rows.Next() {
		var doc KYCDocument
		if err := rows.Scan(&doc.ID, &doc.ApplicationID, &doc.Type, &doc.Side, &doc.FileURL,
			&doc.FileHash, &doc.Status, &doc.VerifiedAt, &doc.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		documents = append(documents, doc)
	}
	return documents, rows.Err()
}

// DeleteDocument removes a document and reports whether it existed.
func (s *Store) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM kyc_documents WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateAMLCheck stores an AML screening result.
func (s *Store) CreateAMLCheck(ctx context.Context, check *AMLCheck) error {
	details, err := json.Marshal(check.Details)
	if err != nil {
		details = []byte("{}")
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO aml_checks (id, user_id, application_id, status, pep_match, sanction_match,
			adverse_media_match, risk_score, risk_level, details, checked_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		check.ID, check.UserID, check.ApplicationID, check.Status, check.PepMatch, check.SanctionMatch,
		check.AdverseMediaMatch, check.RiskScore, check.RiskLevel, details, check.CheckedAt, check.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert aml check: %w", err)
	}
	return nil
}

// ListAMLChecks returns the AML history for a user.
func (s *Store) ListAMLChecks(ctx context.Context, userID uuid.UUID, limit int) ([]AMLCheck, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, application_id, status, pep_match, sanction_match, adverse_media_match,
		       risk_score, risk_level, details, checked_at, created_at
		FROM aml_checks WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list aml checks: %w", err)
	}
	defer rows.Close()

	checks := []AMLCheck{}
	for rows.Next() {
		var check AMLCheck
		var details []byte
		if err := rows.Scan(&check.ID, &check.UserID, &check.ApplicationID, &check.Status,
			&check.PepMatch, &check.SanctionMatch, &check.AdverseMediaMatch, &check.RiskScore,
			&check.RiskLevel, &details, &check.CheckedAt, &check.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan aml check: %w", err)
		}
		if len(details) > 0 {
			_ = json.Unmarshal(details, &check.Details)
		}
		checks = append(checks, check)
	}
	return checks, rows.Err()
}

// AddWatchlistEntry stores a watchlist record.
func (s *Store) AddWatchlistEntry(ctx context.Context, entry *WatchlistEntry) error {
	details, err := json.Marshal(entry.Details)
	if err != nil {
		details = []byte("{}")
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO watchlist_entries (id, name, entity_type, country, list_type, risk_level, details, added_by, added_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		entry.ID, entry.Name, entry.EntityType, entry.Country, entry.ListType,
		entry.RiskLevel, details, entry.AddedBy, entry.AddedAt, entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert watchlist entry: %w", err)
	}
	return nil
}

// RemoveWatchlistEntry deletes a watchlist record.
func (s *Store) RemoveWatchlistEntry(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM watchlist_entries WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete watchlist entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SearchWatchlist finds entries whose name matches the query.
func (s *Store) SearchWatchlist(ctx context.Context, query string, limit int) ([]WatchlistEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, entity_type, country, list_type, risk_level, details, added_by, added_at, created_at
		FROM watchlist_entries WHERE lower(name) LIKE '%' || lower($1) || '%'
		ORDER BY added_at DESC LIMIT $2`, strings.TrimSpace(query), limit)
	if err != nil {
		return nil, fmt.Errorf("search watchlist: %w", err)
	}
	defer rows.Close()

	entries := []WatchlistEntry{}
	for rows.Next() {
		var entry WatchlistEntry
		var details []byte
		if err := rows.Scan(&entry.ID, &entry.Name, &entry.EntityType, &entry.Country,
			&entry.ListType, &entry.RiskLevel, &details, &entry.AddedBy, &entry.AddedAt, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan watchlist entry: %w", err)
		}
		if len(details) > 0 {
			_ = json.Unmarshal(details, &entry.Details)
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// LogAudit records an action for the audit report.
func (s *Store) LogAudit(ctx context.Context, entityType string, entityID *uuid.UUID, action, actorID string, details map[string]interface{}) {
	payload, err := json.Marshal(details)
	if err != nil {
		payload = []byte("{}")
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO kyc_audit_log (entity_type, entity_id, action, actor_id, details)
		VALUES ($1,$2,$3,$4,$5)`, entityType, entityID, action, actorID, payload); err != nil {
		// Audit failures are surfaced in the log; they must not roll back a
		// completed review.
		log.Printf("could not write audit log entry for %s/%s: %v", entityType, action, err)
	}
}

// ComplianceSummary is the aggregated compliance report.
type ComplianceSummary struct {
	TotalApplications int            `json:"total_applications"`
	Approved          int            `json:"approved"`
	Rejected          int            `json:"rejected"`
	Pending           int            `json:"pending"`
	InReview          int            `json:"in_review"`
	NeedsMoreInfo     int            `json:"needs_more_info"`
	AMLChecksPassed   int            `json:"aml_checks_passed"`
	AMLChecksFailed   int            `json:"aml_checks_failed"`
}

// ComplianceReport aggregates application and AML counts for a period.
func (s *Store) ComplianceReport(ctx context.Context, start, end time.Time) (*ComplianceSummary, error) {
	summary := &ComplianceSummary{}

	if err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'approved'),
			COUNT(*) FILTER (WHERE status = 'rejected'),
			COUNT(*) FILTER (WHERE status = 'pending'),
			COUNT(*) FILTER (WHERE status = 'in_review'),
			COUNT(*) FILTER (WHERE status = 'needs_more_info')
		FROM kyc_applications WHERE created_at >= $1 AND created_at < $2`, start, end).
		Scan(&summary.TotalApplications, &summary.Approved, &summary.Rejected,
			&summary.Pending, &summary.InReview, &summary.NeedsMoreInfo); err != nil {
		return nil, fmt.Errorf("aggregate applications: %w", err)
	}

	if err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE aml_check_passed IS TRUE),
			COUNT(*) FILTER (WHERE aml_check_passed IS FALSE)
		FROM kyc_applications WHERE created_at >= $1 AND created_at < $2`, start, end).
		Scan(&summary.AMLChecksPassed, &summary.AMLChecksFailed); err != nil {
		return nil, fmt.Errorf("aggregate aml results: %w", err)
	}

	return summary, nil
}

// AuditSummary is the aggregated audit report.
type AuditSummary struct {
	TotalActions int            `json:"total_actions"`
	ByType       map[string]int `json:"by_type"`
	ByActor      map[string]int `json:"by_actor"`
}

// AuditReport aggregates audit-log actions for a period.
func (s *Store) AuditReport(ctx context.Context, start, end time.Time) (*AuditSummary, error) {
	summary := &AuditSummary{ByType: map[string]int{}, ByActor: map[string]int{}}

	rows, err := s.pool.Query(ctx, `
		SELECT action, COALESCE(actor_id, 'system'), COUNT(*)
		FROM kyc_audit_log WHERE created_at >= $1 AND created_at < $2
		GROUP BY action, COALESCE(actor_id, 'system')`, start, end)
	if err != nil {
		return nil, fmt.Errorf("aggregate audit log: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var action, actor string
		var count int
		if err := rows.Scan(&action, &actor, &count); err != nil {
			return nil, fmt.Errorf("scan audit row: %w", err)
		}
		summary.ByType[action] += count
		summary.ByActor[actor] += count
		summary.TotalActions += count
	}
	return summary, rows.Err()
}
