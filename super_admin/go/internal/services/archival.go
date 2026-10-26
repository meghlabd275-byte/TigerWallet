// ArchivalService - Data archival and retention management.
//
// Archiving exports rows past their retention window to disk and removes them
// from the live table inside one transaction. Restore reads an export back in.
// Table names are validated against the live catalog before they reach a query
// string, because PostgreSQL cannot bind an identifier as a parameter.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/super-admin/internal/database"
)

type ArchivalService struct {
	// ArchiveDir is where export files are written. It defaults to
	// ARCHIVE_DIR or /var/lib/tigerwallet/archives.
	ArchiveDir string
}

func NewArchivalService() *ArchivalService {
	dir := os.Getenv("ARCHIVE_DIR")
	if dir == "" {
		dir = "/var/lib/tigerwallet/archives"
	}
	return &ArchivalService{ArchiveDir: dir}
}

type ArchivePolicy struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	TableName        string    `json:"table_name"`
	RetentionDays    int       `json:"retention_days"`
	ArchiveAfterDays int       `json:"archive_after_days"`
	IsActive         bool      `json:"is_active"`
	CreatedBy        uuid.UUID `json:"created_by"`
	CreatedAt        time.Time `json:"created_at"`
}

type ArchiveRecord struct {
	ID          uuid.UUID  `json:"id"`
	PolicyID    uuid.UUID  `json:"policy_id"`
	TableName   string     `json:"table_name"`
	RecordCount int64      `json:"record_count"`
	ArchivePath string     `json:"archive_path"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedBy   uuid.UUID  `json:"created_by"`
}

// archiveBatchSize bounds how many rows are held in memory per batch.
const archiveBatchSize = 1000

// identifierPattern matches a bare, lowercase SQL identifier. Anything else is
// rejected before it is interpolated into a statement.
var identifierPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

// ErrInvalidTableName is returned when a policy names a table that is not a
// plain identifier or does not exist in the connected database.
var ErrInvalidTableName = errors.New("invalid or unknown table name")

// resolveTable validates name and confirms it exists in the public schema. It
// returns the name quoted for safe interpolation.
func resolveTable(ctx context.Context, name string) (string, error) {
	if !identifierPattern.MatchString(name) {
		return "", fmt.Errorf("%w: %q", ErrInvalidTableName, name)
	}

	var exists bool
	err := database.Pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)`,
		name).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("look up table %q: %w", name, err)
	}
	if !exists {
		return "", fmt.Errorf("%w: %q", ErrInvalidTableName, name)
	}

	// Double quotes plus the identifier check above make the interpolation safe.
	return `"` + name + `"`, nil
}

func (s *ArchivalService) ListPolicies(ctx context.Context) ([]ArchivePolicy, error) {
	rows, err := database.Pool.Query(ctx, "SELECT id, name, table_name, retention_days, archive_after_days, is_active, created_by, created_at FROM archive_policies ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []ArchivePolicy
	for rows.Next() {
		var p ArchivePolicy
		if err := rows.Scan(&p.ID, &p.Name, &p.TableName, &p.RetentionDays, &p.ArchiveAfterDays, &p.IsActive, &p.CreatedBy, &p.CreatedAt); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (s *ArchivalService) CreatePolicy(ctx context.Context, policy *ArchivePolicy, adminID uuid.UUID) (*ArchivePolicy, error) {
	if _, err := resolveTable(ctx, policy.TableName); err != nil {
		return nil, err
	}
	if policy.RetentionDays <= 0 || policy.ArchiveAfterDays < 0 {
		return nil, errors.New("retention_days must be positive and archive_after_days must not be negative")
	}

	err := database.Pool.QueryRow(ctx, `
		INSERT INTO archive_policies (name, table_name, retention_days, archive_after_days, is_active, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, created_at
	`, policy.Name, policy.TableName, policy.RetentionDays, policy.ArchiveAfterDays, policy.IsActive, adminID).Scan(&policy.ID, &policy.CreatedAt)
	return policy, err
}

func (s *ArchivalService) UpdatePolicy(ctx context.Context, id uuid.UUID, policy *ArchivePolicy) error {
	if policy.RetentionDays <= 0 || policy.ArchiveAfterDays < 0 {
		return errors.New("retention_days must be positive and archive_after_days must not be negative")
	}
	_, err := database.Pool.Exec(ctx, `
		UPDATE archive_policies SET name = $1, retention_days = $2, archive_after_days = $3, is_active = $4 WHERE id = $5
	`, policy.Name, policy.RetentionDays, policy.ArchiveAfterDays, policy.IsActive, id)
	return err
}

func (s *ArchivalService) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM archive_policies WHERE id = $1", id)
	return err
}

// RunArchive exports every row older than the policy cutoff and deletes the
// exported rows in the same transaction, so a crash cannot lose data that the
// live table no longer holds.
func (s *ArchivalService) RunArchive(ctx context.Context, policyID uuid.UUID, adminID uuid.UUID) (*ArchiveRecord, error) {
	var policy ArchivePolicy
	err := database.Pool.QueryRow(ctx, "SELECT id, name, table_name, retention_days FROM archive_policies WHERE id = $1", policyID).Scan(&policy.ID, &policy.Name, &policy.TableName, &policy.RetentionDays)
	if err != nil {
		return nil, err
	}

	table, err := resolveTable(ctx, policy.TableName)
	if err != nil {
		return nil, err
	}

	record := &ArchiveRecord{
		ID:        uuid.New(),
		PolicyID:  policyID,
		TableName: policy.TableName,
		Status:    "running",
		StartedAt: time.Now(),
		CreatedBy: adminID,
	}

	_, err = database.Pool.Exec(ctx, `
		INSERT INTO archive_records (id, policy_id, table_name, status, started_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, record.ID, record.PolicyID, record.TableName, record.Status, record.StartedAt, record.CreatedBy)
	if err != nil {
		return nil, err
	}

	record, err = s.exportAndPurge(ctx, record, table, policy.RetentionDays)
	if err != nil {
		record.Status = "failed"
		_, _ = database.Pool.Exec(ctx,
			"UPDATE archive_records SET status = $1, completed_at = NOW() WHERE id = $2",
			record.Status, record.ID)
		return record, err
	}
	return record, nil
}

// exportAndPurge performs the actual export and delete. It runs both steps in a
// single transaction and only commits once the file is durably on disk.
func (s *ArchivalService) exportAndPurge(ctx context.Context, record *ArchiveRecord, table string, retentionDays int) (*ArchiveRecord, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	if err := os.MkdirAll(s.ArchiveDir, 0o750); err != nil {
		return record, fmt.Errorf("create archive directory: %w", err)
	}

	path := filepath.Join(s.ArchiveDir,
		fmt.Sprintf("%s_%s_%s.ndjson", record.TableName, time.Now().UTC().Format("20060102T150405Z"), record.ID.String()[:8]))

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return record, fmt.Errorf("create archive file: %w", err)
	}
	defer file.Close()

	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return record, fmt.Errorf("begin archive transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	encoder := json.NewEncoder(file)
	var total int64

	for {
		// ctid identifies the exact physical row, so the delete cannot touch a
		// row that was not exported even if the table has no primary key.
		rows, err := tx.Query(ctx, fmt.Sprintf(
			`SELECT ctid::text, row_to_json(t) FROM %s t WHERE created_at < $1 LIMIT $2`,
			table), cutoff, archiveBatchSize)
		if err != nil {
			return record, fmt.Errorf("select rows to archive: %w", err)
		}

		type archivedRow struct {
			Ctid string          `json:"ctid"`
			Row  json.RawMessage `json:"row"`
		}
		var batch []archivedRow
		for rows.Next() {
			var item archivedRow
			if err := rows.Scan(&item.Ctid, &item.Row); err != nil {
				rows.Close()
				return record, fmt.Errorf("scan row: %w", err)
			}
			batch = append(batch, item)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return record, fmt.Errorf("iterate rows: %w", err)
		}

		if len(batch) == 0 {
			break
		}

		ctids := make([]string, 0, len(batch))
		for _, item := range batch {
			if err := encoder.Encode(item); err != nil {
				return record, fmt.Errorf("write archive entry: %w", err)
			}
			ctids = append(ctids, item.Ctid)
		}

		if _, err := tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE ctid = ANY($1::tid[])`, table), ctids); err != nil {
			return record, fmt.Errorf("delete archived rows: %w", err)
		}

		total += int64(len(batch))
	}

	// Flush the file before committing so a committed record always has a
	// complete export behind it.
	if err := file.Sync(); err != nil {
		return record, fmt.Errorf("sync archive file: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return record, fmt.Errorf("commit archive transaction: %w", err)
	}

	now := time.Now()
	record.RecordCount = total
	record.CompletedAt = &now
	record.Status = "completed"
	record.ArchivePath = path

	_, err = database.Pool.Exec(ctx, `
		UPDATE archive_records SET record_count = $1, archive_path = $2, status = $3, completed_at = $4 WHERE id = $5
	`, record.RecordCount, record.ArchivePath, record.Status, record.CompletedAt, record.ID)
	if err != nil {
		return record, err
	}

	return record, nil
}

func (s *ArchivalService) ListArchiveRecords(ctx context.Context, policyID *uuid.UUID, limit, offset int) ([]ArchiveRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := "SELECT id, policy_id, table_name, record_count, archive_path, status, started_at, completed_at, created_by FROM archive_records WHERE 1=1"
	args := []interface{}{}
	argNum := 1

	if policyID != nil {
		query += fmt.Sprintf(" AND policy_id = $%d", argNum)
		args = append(args, *policyID)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY started_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := database.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ArchiveRecord
	for rows.Next() {
		var r ArchiveRecord
		if err := rows.Scan(&r.ID, &r.PolicyID, &r.TableName, &r.RecordCount, &r.ArchivePath, &r.Status, &r.StartedAt, &r.CompletedAt, &r.CreatedBy); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// RestoreFromArchive re-inserts the rows held in an archive file. Rows are
// inserted with ON CONFLICT DO NOTHING so a partially restored table can be
// retried safely.
func (s *ArchivalService) RestoreFromArchive(ctx context.Context, recordID uuid.UUID, adminID uuid.UUID) error {
	var record ArchiveRecord
	err := database.Pool.QueryRow(ctx, "SELECT id, table_name, archive_path FROM archive_records WHERE id = $1", recordID).Scan(&record.ID, &record.TableName, &record.ArchivePath)
	if err != nil {
		return err
	}
	if record.ArchivePath == "" {
		return errors.New("archive record has no export file")
	}

	table, err := resolveTable(ctx, record.TableName)
	if err != nil {
		return err
	}

	// Confine the read to the configured archive directory so a tampered
	// archive_path cannot be used to read an arbitrary file.
	cleanBase := filepath.Clean(s.ArchiveDir) + string(os.PathSeparator)
	cleanPath := filepath.Clean(record.ArchivePath)
	if !strings.HasPrefix(cleanPath, cleanBase) {
		return fmt.Errorf("archive path %q is outside the archive directory", record.ArchivePath)
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return fmt.Errorf("open archive file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var restored int64

	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin restore transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	batch := make([]json.RawMessage, 0, archiveBatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		payload, err := json.Marshal(batch)
		if err != nil {
			return fmt.Errorf("encode restore batch: %w", err)
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(
			`INSERT INTO %s SELECT * FROM json_populate_recordset(NULL::%s, $1) ON CONFLICT DO NOTHING`,
			table, table), payload); err != nil {
			return fmt.Errorf("restore rows: %w", err)
		}
		restored += int64(len(batch))
		batch = batch[:0]
		return nil
	}

	for {
		var entry struct {
			Ctid string          `json:"ctid"`
			Row  json.RawMessage `json:"row"`
		}
		if err := decoder.Decode(&entry); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("read archive entry: %w", err)
		}
		batch = append(batch, entry.Row)
		if len(batch) >= archiveBatchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit restore transaction: %w", err)
	}

	_, err = database.Pool.Exec(ctx, `
		UPDATE archive_records SET status = $1 WHERE id = $2
	`, "restored", record.ID)
	return err
}
