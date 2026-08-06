// ReportService - Report generation service
package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type ReportService struct{}

func NewReportService() *ReportService {
	return &ReportService{}
}

func (s *ReportService) GenerateReport(ctx context.Context, report *models.Report, adminID uuid.UUID) (*models.Report, error) {
	err := database.QueryRow(ctx, `
		INSERT INTO reports (report_type, title, filters, status, generated_by, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, created_at
	`, report.ReportType, report.Title, report.Filters, "pending", adminID).Scan(&report.ID, &report.CreatedAt)
	return report, err
}

func (s *ReportService) ListReports(ctx context.Context, limit, offset int) ([]models.Report, int, error) {
	var total int
	database.QueryRow(ctx, "SELECT COUNT(*) FROM reports").Scan(&total)

	rows, err := database.Query(ctx, `
		SELECT id, report_type, title, filters, file_path, status, generated_by, created_at, completed_at
		FROM reports ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []models.Report
	for rows.Next() {
		var report models.Report
		err := rows.Scan(&report.ID, &report.ReportType, &report.Title, &report.Filters, &report.FilePath, &report.Status, &report.GeneratedBy, &report.CreatedAt, &report.CompletedAt)
		if err != nil {
			return nil, 0, err
		}
		reports = append(reports, report)
	}
	return reports, total, nil
}

func (s *ReportService) GetReport(ctx context.Context, id uuid.UUID) (*models.Report, error) {
	var report models.Report
	err := database.QueryRow(ctx, `
		SELECT id, report_type, title, filters, file_path, status, generated_by, created_at, completed_at
		FROM reports WHERE id = $1
	`, id).Scan(&report.ID, &report.ReportType, &report.Title, &report.Filters, &report.FilePath, &report.Status, &report.GeneratedBy, &report.CreatedAt, &report.CompletedAt)
	return &report, err
}
