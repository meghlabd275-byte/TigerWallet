// ReportService - Report generation service
package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/master-admin-management/internal/database"
)

type ReportService struct{}

func NewReportService() *ReportService {
	return &ReportService{}
}

type ReportConfig struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	ReportType  string    `json:"report_type"` // compliance, finance, security, transaction, user
	Parameters  map[string]interface{} `json:"parameters"`
	FileFormat  string    `json:"file_format"` // csv, json, pdf
	IsScheduled bool      `json:"is_scheduled"`
	Schedule    string    `json:"schedule"` // cron expression
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type Report struct {
	ID          uuid.UUID  `json:"id"`
	ConfigID    uuid.UUID  `json:"config_id"`
	Name        string    `json:"name"`
	FilePath    string    `json:"file_path"`
	FileSize    int64     `json:"file_size"`
	Status      string    `json:"status"` // pending, generating, completed, failed
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

func (s *ReportService) GenerateReport(ctx context.Context, config *ReportConfig, adminID uuid.UUID) (*Report, error) {
	report := &Report{
		ID:        uuid.New(),
		ConfigID:  config.ID,
		Name:      config.Name,
		Status:    "generating",
		CreatedBy: adminID,
		CreatedAt: time.Now(),
	}

	// Insert report record
	_, err := database.Pool.Exec(ctx, `
		INSERT INTO reports (id, config_id, name, status, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, report.ID, report.ConfigID, report.Name, report.Status, report.CreatedBy, report.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Generate report data based on type
	var data interface{}
	var err error

	switch config.ReportType {
	case "compliance":
		data, err = s.generateComplianceReport(ctx, config.Parameters)
	case "finance":
		data, err = s.generateFinanceReport(ctx, config.Parameters)
	case "security":
		data, err = s.generateSecurityReport(ctx, config.Parameters)
	case "transaction":
		data, err = s.generateTransactionReport(ctx, config.Parameters)
	case "user":
		data, err = s.generateUserReport(ctx, config.Parameters)
	default:
		data, err = s.generateGenericReport(ctx, config.Parameters)
	}

	if err != nil {
		report.Status = "failed"
		database.Pool.Exec(ctx, "UPDATE reports SET status = $1 WHERE id = $2", report.Status, report.ID)
		return report, err
	}

	// Write to file
	var filePath string
	switch config.FileFormat {
	case "json":
		filePath, err = s.writeJSONReport(report.ID.String(), data)
	case "csv":
		filePath, err = s.writeCSVReport(report.ID.String(), data)
	default:
		filePath, err = s.writeJSONReport(report.ID.String(), data)
	}

	if err != nil {
		report.Status = "failed"
		database.Pool.Exec(ctx, "UPDATE reports SET status = $1 WHERE id = $2", report.Status, report.ID)
		return report, err
	}

	now := time.Now()
	report.CompletedAt = &now
	report.Status = "completed"
	report.FilePath = filePath

	database.Pool.Exec(ctx, `
		UPDATE reports SET status = $1, file_path = $2, completed_at = $3 WHERE id = $4
	`, report.Status, report.FilePath, report.CompletedAt, report.ID)

	return report, nil
}

func (s *ReportService) generateComplianceReport(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT u.id, u.email, u.username, u.kyc_status, u.status, u.created_at,
			   k.id, k.doc_type, k.status, k.submitted_at, k.reviewed_at
		FROM users u
		LEFT JOIN kyc_requests k ON u.id = k.user_id
		WHERE u.created_at > NOW() - INTERVAL '30 days'
		ORDER BY u.created_at DESC
		LIMIT 1000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []map[string]interface{}
	for rows.Next() {
		var userID, kycID uuid.UUID
		var email, username, kycStatus, userStatus, docType, kycStatus2 string
		var createdAt, submittedAt, reviewedAt *time.Time

		err := rows.Scan(&userID, &email, &username, &kycStatus, &userStatus, &createdAt,
			&kycID, &docType, &kycStatus2, &submittedAt, &reviewedAt)
		if err != nil {
			continue
		}

		reports = append(reports, map[string]interface{}{
			"user_id":      userID,
			"email":        email,
			"username":     username,
			"kyc_status":  kycStatus,
			"user_status": userStatus,
			"doc_type":    docType,
			"kyc_status":  kycStatus2,
			"submitted_at": submittedAt,
			"reviewed_at": reviewedAt,
			"created_at":  createdAt,
		})
	}
	return reports, nil
}

func (s *ReportService) generateFinanceReport(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT DATE(created_at) as date, COUNT(*) as transaction_count, 
			   SUM(amount) as total_volume, SUM(fee) as total_fees
		FROM transactions
		WHERE created_at > NOW() - INTERVAL '30 days'
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []map[string]interface{}
	for rows.Next() {
		var date time.Time
		var count int64
		var volume, fees float64

		err := rows.Scan(&date, &count, &volume, &fees)
		if err != nil {
			continue
		}

		reports = append(reports, map[string]interface{}{
			"date":            date.Format("2006-01-02"),
			"transaction_count": count,
			"total_volume":    volume,
			"total_fees":      fees,
		})
	}
	return reports, nil
}

func (s *ReportService) generateSecurityReport(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT al.id, al.admin_id, al.action, al.resource_type, al.ip_address, al.created_at,
			   au.username
		FROM audit_logs al
		LEFT JOIN admin_users au ON al.admin_id = au.id
		WHERE al.created_at > NOW() - INTERVAL '30 days'
		ORDER BY al.created_at DESC
		LIMIT 1000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []map[string]interface{}
	for rows.Next() {
		var logID, adminID *uuid.UUID
		var action, resourceType, ipAddress, username string
		var createdAt time.Time

		err := rows.Scan(&logID, &adminID, &action, &resourceType, &ipAddress, &createdAt, &username)
		if err != nil {
			continue
		}

		reports = append(reports, map[string]interface{}{
			"log_id":        logID,
			"admin_id":     adminID,
			"admin_name":   username,
			"action":       action,
			"resource_type": resourceType,
			"ip_address":   ipAddress,
			"created_at":   createdAt,
		})
	}
	return reports, nil
}

func (s *ReportService) generateTransactionReport(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT id, user_id, type, amount, currency, status, from_address, to_address, tx_hash, created_at
		FROM transactions
		WHERE created_at > NOW() - INTERVAL '30 days'
		ORDER BY created_at DESC
		LIMIT 1000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []map[string]interface{}
	for rows.Next() {
		var id, userID uuid.UUID
		var txType, currency, status, fromAddr, toAddr, txHash string
		var amount float64
		var createdAt time.Time

		err := rows.Scan(&id, &userID, &txType, &amount, &currency, &status, &fromAddr, &toAddr, &txHash, &createdAt)
		if err != nil {
			continue
		}

		reports = append(reports, map[string]interface{}{
			"id":          id,
			"user_id":     userID,
			"type":        txType,
			"amount":      amount,
			"currency":    currency,
			"status":      status,
			"from_address": fromAddr,
			"to_address":  toAddr,
			"tx_hash":     txHash,
			"created_at":  createdAt,
		})
	}
	return reports, nil
}

func (s *ReportService) generateUserReport(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT id, email, username, kyc_status, status, country, created_at, last_login
		FROM users
		WHERE created_at > NOW() - INTERVAL '30 days'
		ORDER BY created_at DESC
		LIMIT 1000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var email, username, kycStatus, status, country string
		var createdAt, lastLogin *time.Time

		err := rows.Scan(&id, &email, &username, &kycStatus, &status, &country, &createdAt, &lastLogin)
		if err != nil {
			continue
		}

		reports = append(reports, map[string]interface{}{
			"id":          id,
			"email":       email,
			"username":    username,
			"kyc_status":  kycStatus,
			"status":      status,
			"country":     country,
			"created_at":  createdAt,
			"last_login":  lastLogin,
		})
	}
	return reports, nil
}

func (s *ReportService) generateGenericReport(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"message": "Generic report generated"}, nil
}

func (s *ReportService) writeJSONReport(id string, data interface{}) (string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	// In production, write to actual file
	return fmt.Sprintf("/reports/%s.json", id[:8]), nil
}

func (s *ReportService) writeCSVReport(id string, data interface{}) (string, error) {
	// In production, write to actual file
	return fmt.Sprintf("/reports/%s.csv", id[:8]), nil
}

func (s *ReportService) ListReportConfigs(ctx context.Context) ([]ReportConfig, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT id, name, report_type, parameters, file_format, is_scheduled, schedule, created_by, created_at
		FROM report_configs ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []ReportConfig
	for rows.Next() {
		var c ReportConfig
		var paramsJSON []byte
		err := rows.Scan(&c.ID, &c.Name, &c.ReportType, &paramsJSON, &c.FileFormat, &c.IsScheduled, &c.Schedule, &c.CreatedBy, &c.CreatedAt)
		if err != nil {
			continue
		}
		json.Unmarshal(paramsJSON, &c.Parameters)
		configs = append(configs, c)
	}
	return configs, nil
}

func (s *ReportService) CreateReportConfig(ctx context.Context, config *ReportConfig, adminID uuid.UUID) (*ReportConfig, error) {
	paramsJSON, _ := json.Marshal(config.Parameters)
	err := database.Pool.QueryRow(ctx, `
		INSERT INTO report_configs (name, report_type, parameters, file_format, is_scheduled, schedule, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, created_at
	`, config.Name, config.ReportType, paramsJSON, config.FileFormat, config.IsScheduled, config.Schedule, adminID).Scan(&config.ID, &config.CreatedAt)
	return config, err
}

func (s *ReportService) ListReports(ctx context.Context, limit, offset int) ([]Report, int, error) {
	var total int
	database.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM reports").Scan(&total)

	rows, err := database.Pool.Query(ctx, `
		SELECT id, config_id, name, file_path, file_size, status, created_by, created_at, completed_at
		FROM reports ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		var r Report
		err := rows.Scan(&r.ID, &r.ConfigID, &r.Name, &r.FilePath, &r.FileSize, &r.Status, &r.CreatedBy, &r.CreatedAt, &r.CompletedAt)
		if err != nil {
			continue
		}
		reports = append(reports, r)
	}
	return reports, total, nil
}
