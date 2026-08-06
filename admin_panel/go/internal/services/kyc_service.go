// KYCService - KYC management service
package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type KYCService struct{}

func NewKYCService() *KYCService {
	return &KYCService{}
}

func (s *KYCService) ListKYCRequests(ctx context.Context, status string, limit, offset int) ([]models.KYCRequest, int, error) {
	var total int
	if status != "" {
		database.QueryRow(ctx, "SELECT COUNT(*) FROM kyc_requests WHERE status = $1", status).Scan(&total)
	} else {
		database.QueryRow(ctx, "SELECT COUNT(*) FROM kyc_requests").Scan(&total)
	}

	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = database.Query(ctx, `
			SELECT id, user_id, doc_type, status, document_url, submitted_at, reviewed_at, reviewed_by, reject_reason
			FROM kyc_requests WHERE status = $1 ORDER BY submitted_at DESC LIMIT $2 OFFSET $3
		`, status, limit, offset)
	} else {
		rows, err = database.Query(ctx, `
			SELECT id, user_id, doc_type, status, document_url, submitted_at, reviewed_at, reviewed_by, reject_reason
			FROM kyc_requests ORDER BY submitted_at DESC LIMIT $1 OFFSET $2
		`, limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var requests []models.KYCRequest
	for rows.Next() {
		var req models.KYCRequest
		err := rows.Scan(
			&req.ID, &req.UserID, &req.DocType, &req.Status, &req.DocumentURL,
			&req.SubmittedAt, &req.ReviewedAt, &req.ReviewedBy, &req.RejectReason,
		)
		if err != nil {
			return nil, 0, err
		}
		requests = append(requests, req)
	}

	return requests, total, nil
}

func (s *KYCService) ApproveKYC(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE kyc_requests SET status = 'verified', reviewed_at = $1, reviewed_by = $2 WHERE id = $3
	`, time.Now(), adminID, id)
	if err != nil {
		return err
	}

	// Update user KYC status
	_, err = tx.Exec(ctx, `
		UPDATE users SET kyc_status = 'verified', updated_at = NOW() 
		WHERE id = (SELECT user_id FROM kyc_requests WHERE id = $1)
	`, id)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *KYCService) RejectKYC(ctx context.Context, id uuid.UUID, adminID uuid.UUID, reason string) error {
	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE kyc_requests SET status = 'rejected', reviewed_at = $1, reviewed_by = $2, reject_reason = $3 WHERE id = $4
	`, time.Now(), adminID, reason, id)
	if err != nil {
		return err
	}

	// Update user KYC status
	_, err = tx.Exec(ctx, `
		UPDATE users SET kyc_status = 'rejected', updated_at = NOW() 
		WHERE id = (SELECT user_id FROM kyc_requests WHERE id = $1)
	`, id)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
