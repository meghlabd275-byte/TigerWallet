// WithdrawalService - Withdrawal management service
package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type WithdrawalService struct{}

func NewWithdrawalService() *WithdrawalService {
	return &WithdrawalService{}
}

func (s *WithdrawalService) ListWithdrawals(ctx context.Context, status string, limit, offset int) ([]models.Withdrawal, int, error) {
	var total int
	database.QueryRow(ctx, "SELECT COUNT(*) FROM withdrawals").Scan(&total)

	rows, err := database.Query(ctx, `
		SELECT id, user_id, amount, currency, status, address, tx_hash, approved_by, processed_at, created_at
		FROM withdrawals ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var withdrawals []models.Withdrawal
	for rows.Next() {
		var w models.Withdrawal
		err := rows.Scan(
			&w.ID, &w.UserID, &w.Amount, &w.Currency, &w.Status,
			&w.Address, &w.TXHash, &w.ApprovedBy, &w.ProcessedAt, &w.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		withdrawals = append(withdrawals, w)
	}

	return withdrawals, total, nil
}

func (s *WithdrawalService) ApproveWithdrawal(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	_, err := database.Exec(ctx, `
		UPDATE withdrawals SET status = 'approved', approved_by = $1 WHERE id = $2
	`, adminID, id)
	return err
}

func (s *WithdrawalService) RejectWithdrawal(ctx context.Context, id uuid.UUID, reason string) error {
	_, err := database.Exec(ctx, `
		UPDATE withdrawals SET status = 'rejected', reject_reason = $1 WHERE id = $2
	`, reason, id)
	return err
}

func (s *WithdrawalService) ProcessWithdrawal(ctx context.Context, id uuid.UUID, txHash string) error {
	_, err := database.Exec(ctx, `
		UPDATE withdrawals SET status = 'completed', tx_hash = $1, processed_at = $2 WHERE id = $3
	`, txHash, time.Now(), id)
	return err
}
