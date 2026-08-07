// AuditService - Audit log service
package services

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type AuditService struct{}

func NewAuditService() *AuditService {
	return &AuditService{}
}

func (s *AuditService) Log(ctx context.Context, adminID uuid.UUID, action, resourceType, resourceID string, details map[string]interface{}, ipAddress, userAgent string) error {
	detailsJSON, _ := json.Marshal(details)
	_, err := database.Exec(ctx, `
		INSERT INTO audit_logs (admin_id, action, resource_type, resource_id, details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`, adminID, action, resourceType, resourceID, detailsJSON, ipAddress, userAgent)
	return err
}

func (s *AuditService) ListAuditLogs(ctx context.Context, adminID, action, resourceType string, limit, offset int) ([]models.AuditLog, int, error) {
	var total int
	database.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs").Scan(&total)

	rows, err := database.Query(ctx, `
		SELECT id, admin_id, action, resource_type, resource_id, details, ip_address, user_agent, created_at
		FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		var detailsJSON []byte
		err := rows.Scan(&log.ID, &log.AdminID, &log.Action, &log.ResourceType, &log.ResourceID, &detailsJSON, &log.IPAddress, &log.UserAgent, &log.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		json.Unmarshal(detailsJSON, &log.Details)
		logs = append(logs, log)
	}
	return logs, total, nil
}

func (s *AuditService) ExportAuditLogs(ctx context.Context, adminID uuid.UUID, format, startDate, endDate string) (string, error) {
	// Create a report record
	var reportID uuid.UUID
	err := database.QueryRow(ctx, `
		INSERT INTO reports (report_type, title, filters, status, generated_by, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id
	`, "audit_logs_export", "Audit Logs Export", `{"format":"`+format+`","start_date":"`+startDate+`","end_date":"`+endDate+`"}`, "processing", adminID).Scan(&reportID)
	if err != nil {
		return "", err
	}

	// In production, this would generate the actual file
	return "/reports/audit_logs_" + reportID.String() + "." + format, nil
}
