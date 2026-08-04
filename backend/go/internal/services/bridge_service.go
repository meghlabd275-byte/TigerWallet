package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/backend/internal/models"
)

type BridgeService struct {
	db *sql.DB
}

func NewBridgeService(db *sql.DB) *BridgeService {
	return &BridgeService{db: db}
}

// Supported chains
var SupportedChains = map[string]struct{}{
	"ETHEREUM":    {},
	"BSC":         {},
	"POLYGON":     {},
	"AVALANCHE":   {},
	"ARBITRUM":    {},
	"OPTIMISM":    {},
	"SOLANA":      {},
	"NEAR":        {},
	"APTOS":       {},
	"SUI":         {},
}

// Initiate bridge transaction
func (s *BridgeService) InitiateBridge(ctx context.Context, userID uuid.UUID, fromChain, toChain string, token string, amount float64) (*models.BridgeTransaction, error) {
	// Validate chains
	if _, ok := SupportedChains[fromChain]; !ok {
		return nil, fmt.Errorf("unsupported source chain: %s", fromChain)
	}
	if _, ok := SupportedChains[toChain]; !ok {
		return nil, fmt.Errorf("unsupported destination chain: %s", toChain)
	}

	if fromChain == toChain {
		return nil, fmt.Errorf("source and destination chains must be different")
	}

	// Get bridge fee
	fee, err := s.getBridgeFee(fromChain, toChain, token)
	if err != nil {
		return nil, err
	}

	// Create bridge transaction
	tx := &models.BridgeTransaction{
		ID:              uuid.New(),
		UserID:          userID,
		FromChain:       fromChain,
		ToChain:         toChain,
		Token:           token,
		Amount:          amount,
		Fee:             fee,
		ReceivedAmount:  amount - fee,
		Status:          "PENDING",
		SourceTxHash:    "",
		DestinationTxHash: "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO bridge_transactions 
		(id, user_id, from_chain, to_chain, token, amount, fee, received_amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, tx.ID, tx.UserID, tx.FromChain, tx.ToChain, tx.Token, tx.Amount, tx.Fee, tx.ReceivedAmount, tx.Status, tx.CreatedAt, tx.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create bridge transaction: %w", err)
	}

	return tx, nil
}

// Confirm source chain deposit
func (s *BridgeService) confirmSourceDeposit(ctx context.Context, txID uuid.UUID, sourceTxHash string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE bridge_transactions 
		SET status = 'CONFIRMED', source_tx_hash = $2, updated_at = NOW()
		WHERE id = $1 AND status = 'PENDING'
	`, txID, sourceTxHash)

	if err != nil {
		return fmt.Errorf("failed to confirm deposit: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("transaction not found or already processed")
	}

	return nil
}

// Complete bridge on destination chain
func (s *BridgeService) CompleteBridge(ctx context.Context, txID uuid.UUID, destTxHash string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE bridge_transactions 
		SET status = 'COMPLETED', destination_tx_hash = $2, updated_at = NOW()
		WHERE id = $1 AND status = 'CONFIRMED'
	`, txID, destTxHash)

	if err != nil {
		return fmt.Errorf("failed to complete bridge: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("transaction not found or not confirmed")
	}

	return nil
}

// Cancel bridge
func (s *BridgeService) CancelBridge(ctx context.Context, txID uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE bridge_transactions 
		SET status = 'CANCELLED', updated_at = NOW()
		WHERE id = $1 AND status = 'PENDING'
	`, txID)

	if err != nil {
		return fmt.Errorf("failed to cancel bridge: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("transaction not found or already processed")
	}

	return nil
}

// Get user bridge transactions
func (s *BridgeService) GetUserTransactions(ctx context.Context, userID uuid.UUID) ([]models.BridgeTransaction, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, from_chain, to_chain, token, amount, fee, received_amount, 
			   status, source_tx_hash, destination_tx_hash, created_at, updated_at
		FROM bridge_transactions 
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []models.BridgeTransaction
	for rows.Next() {
		var tx models.BridgeTransaction
		err := rows.Scan(&tx.ID, &tx.UserID, &tx.FromChain, &tx.ToChain, &tx.Token, 
			&tx.Amount, &tx.Fee, &tx.ReceivedAmount, &tx.Status, 
			&tx.SourceTxHash, &tx.DestinationTxHash, &tx.CreatedAt, &tx.UpdatedAt)
		if err != nil {
			continue
		}
		txs = append(txs, tx)
	}

	return txs, nil
}

// Get bridge status
func (s *BridgeService) GetTransactionStatus(ctx context.Context, txID uuid.UUID) (*models.BridgeTransaction, error) {
	var tx models.BridgeTransaction
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, from_chain, to_chain, token, amount, fee, received_amount, 
			   status, source_tx_hash, destination_tx_hash, created_at, updated_at
		FROM bridge_transactions 
		WHERE id = $1
	`, txID).Scan(&tx.ID, &tx.UserID, &tx.FromChain, &tx.ToChain, &tx.Token, 
		&tx.Amount, &tx.Fee, &tx.ReceivedAmount, &tx.Status, 
		&tx.SourceTxHash, &tx.DestinationTxHash, &tx.CreatedAt, &tx.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}

	return &tx, err
}

// Get supported tokens for bridge
func (s *BridgeService) GetSupportedTokens(ctx context.Context, chain string) ([]models.BridgeToken, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT token, chain, min_amount, max_amount, is_active
		FROM bridge_tokens
		WHERE chain = $1 AND is_active = true
	`, chain)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []models.BridgeToken
	for rows.Next() {
		var t models.BridgeToken
		err := rows.Scan(&t.Token, &t.Chain, &t.MinAmount, &t.MaxAmount, &t.IsActive)
		if err != nil {
			continue
		}
		tokens = append(tokens, t)
	}

	return tokens, nil
}

// Get bridge fee
func (s *BridgeService) getBridgeFee(fromChain, toChain, token string) (float64, error) {
	var fee float64
	err := s.db.QueryRow(`
		SELECT fee_percentage FROM bridge_fees 
		WHERE from_chain = $1 AND to_chain = $2 AND token = $3
	`, fromChain, toChain, token).Scan(&fee)

	if err == sql.ErrNoRows {
		// Default 0.3%
		return 0.003, nil
	}

	return fee, err
}

// Get estimated receive amount
func (s *BridgeService) EstimateReceive(ctx context.Context, fromChain, toChain string, token string, amount float64) (float64, float64, error) {
	fee, err := s.getBridgeFee(fromChain, toChain, token)
	if err != nil {
		return 0, 0, err
	}

	received := amount * (1 - fee)
	return received, fee, nil
}

// Get bridge statistics
func (s *BridgeService) GetBridgeStats(ctx context.Context) (map[string]interface{}, error) {
	var totalVolume, totalTransactions float64

	s.db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0), COALESCE(COUNT(*), 0)
		FROM bridge_transactions
		WHERE status = 'COMPLETED'
	`).Scan(&totalVolume, &totalTransactions)

	stats := map[string]interface{}{
		"totalVolume":       totalVolume,
		"totalTransactions": totalTransactions,
		"supportedChains":   len(SupportedChains),
	}

	return stats, nil
}
