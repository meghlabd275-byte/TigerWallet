// TransactionService - Transaction management service
package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type TransactionService struct{}

func NewTransactionService() *TransactionService {
	return &TransactionService{}
}

func (s *TransactionService) ListTransactions(ctx context.Context, status, userID string, limit, offset int) ([]models.Transaction, int, error) {
	var total int
	database.QueryRow(ctx, "SELECT COUNT(*) FROM transactions").Scan(&total)

	rows, err := database.Query(ctx, `
		SELECT id, user_id, type, amount, currency, status, from_address, to_address, tx_hash, fee, chain_id, timestamp
		FROM transactions ORDER BY timestamp DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txs []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(
			&tx.ID, &tx.UserID, &tx.Type, &tx.Amount, &tx.Currency, &tx.Status,
			&tx.FromAddress, &tx.ToAddress, &tx.TXHash, &tx.Fee, &tx.ChainID, &tx.Timestamp,
		)
		if err != nil {
			return nil, 0, err
		}
		txs = append(txs, tx)
	}

	return txs, total, nil
}

func (s *TransactionService) GetTransactionByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	var tx models.Transaction
	err := database.QueryRow(ctx, `
		SELECT id, user_id, type, amount, currency, status, from_address, to_address, tx_hash, fee, chain_id, timestamp
		FROM transactions WHERE id = $1
	`, id).Scan(
		&tx.ID, &tx.UserID, &tx.Type, &tx.Amount, &tx.Currency, &tx.Status,
		&tx.FromAddress, &tx.ToAddress, &tx.TXHash, &tx.Fee, &tx.ChainID, &tx.Timestamp,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &tx, err
}

func (s *TransactionService) FlagTransaction(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "UPDATE transactions SET status = 'flagged' WHERE id = $1", id)
	return err
}

func (s *TransactionService) UnflagTransaction(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "UPDATE transactions SET status = 'pending' WHERE id = $1", id)
	return err
}
