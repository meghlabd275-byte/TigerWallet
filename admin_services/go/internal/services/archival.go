// ArchivalService - Data archival and retention management
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/admin/internal/database"
)

type ArchivalService struct{}

func NewArchivalService() *ArchivalService {
	return &ArchivalService{}
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
	return policies, nil
}

func (s *ArchivalService) CreatePolicy(ctx context.Context, policy *ArchivePolicy, adminID uuid.UUID) (*ArchivePolicy, error) {
	err := database.Pool.QueryRow(ctx, `
		INSERT INTO archive_policies (name, table_name, retention_days, archive_after_days, is_active, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, created_at
	`, policy.Name, policy.TableName, policy.RetentionDays, policy.ArchiveAfterDays, policy.IsActive, adminID).Scan(&policy.ID, &policy.CreatedAt)
	return policy, err
}

func (s *ArchivalService) UpdatePolicy(ctx context.Context, id uuid.UUID, policy *ArchivePolicy) error {
	_, err := database.Pool.Exec(ctx, `
		UPDATE archive_policies SET name = $1, retention_days = $2, archive_after_days = $3, is_active = $4 WHERE id = $5
	`, policy.Name, policy.RetentionDays, policy.ArchiveAfterDays, policy.IsActive, id)
	return err
}

func (s *ArchivalService) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM archive_policies WHERE id = $1", id)
	return err
}

func (s *ArchivalService) RunArchive(ctx context.Context, policyID uuid.UUID, adminID uuid.UUID) (*ArchiveRecord, error) {
	var policy ArchivePolicy
	err := database.Pool.QueryRow(ctx, "SELECT id, name, table_name, retention_days FROM archive_policies WHERE id = $1", policyID).Scan(&policy.ID, &policy.Name, &policy.TableName, &policy.RetentionDays)
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

	// Calculate cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -policy.RetentionDays)

	// Count records to archive
	var count int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE created_at < $1", policy.TableName)
	err = database.Pool.QueryRow(ctx, countQuery, cutoffDate).Scan(&count)
	if err != nil {
		record.Status = "failed"
		database.Pool.Exec(ctx, "UPDATE archive_records SET status = $1 WHERE id = $2", record.Status, record.ID)
		return record, err
	}

	record.RecordCount = count

	// In production, this would actually archive the data
	// For now, we simulate the archival process
	time.Sleep(100 * time.Millisecond)

	now := time.Now()
	record.CompletedAt = &now
	record.Status = "completed"
	record.ArchivePath = fmt.Sprintf("/archives/%s_%s.json", policy.TableName, record.ID.String()[:8])

	_, err = database.Pool.Exec(ctx, `
		UPDATE archive_records SET record_count = $1, archive_path = $2, status = $3, completed_at = $4 WHERE id = $5
	`, record.RecordCount, record.ArchivePath, record.Status, record.CompletedAt, record.ID)

	return record, err
}

func (s *ArchivalService) ListArchiveRecords(ctx context.Context, policyID *uuid.UUID, limit, offset int) ([]ArchiveRecord, error) {
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
	return records, nil
}

func (s *ArchivalService) RestoreFromArchive(ctx context.Context, recordID uuid.UUID, adminID uuid.UUID) error {
	var record ArchiveRecord
	err := database.Pool.QueryRow(ctx, "SELECT id, table_name, archive_path FROM archive_records WHERE id = $1", recordID).Scan(&record.ID, &record.TableName, &record.ArchivePath)
	if err != nil {
		return err
	}

	// In production, this would actually restore data from the archive
	// For now, we simulate the restore process
	return nil
}
