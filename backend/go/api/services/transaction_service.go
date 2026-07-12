package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"tigerwallet/backend/go/api/models"
)

type TransactionService struct {
	mu           sync.RWMutex
	transactions map[string]*models.Transaction
}

var (
	txInstance *TransactionService
	txOnce    sync.Once
)

func NewTransactionService() *TransactionService {
	txOnce.Do(func() {
		txInstance = &TransactionService{
			transactions: make(map[string]*models.Transaction),
		}
	})
	return txInstance
}

func (s *TransactionService) Create(ctx context.Context, req *models.CreateTransactionRequest) (*models.Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	txID := fmt.Sprintf("tx_%d", time.Now().UnixNano())

	tx := &models.Transaction{
		ID:           txID,
		WalletID:    req.WalletID,
		BlockchainID: req.BlockchainID,
		Type:         req.Type,
		Status:       "pending",
		From:         req.From,
		To:           req.To,
		TokenSymbol:  req.TokenSymbol,
		Amount:       req.Amount,
		AmountUSD:    req.AmountUSD,
		Fee:          req.Fee,
		FeeUSD:       req.FeeUSD,
		Hash:         "",
		Timestamp:    time.Now(),
	}

	s.transactions[txID] = tx
	return tx, nil
}

func (s *TransactionService) GetByID(ctx context.Context, txID string) (*models.Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tx, ok := s.transactions[txID]
	if !ok {
		return nil, errors.New("transaction not found")
	}

	return tx, nil
}

func (s *TransactionService) GetByWalletID(ctx context.Context, walletID string) ([]*models.Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Transaction
	for _, tx := range s.transactions {
		if tx.WalletID == walletID {
			result = append(result, tx)
		}
	}

	return result, nil
}

func (s *TransactionService) Sign(ctx context.Context, txID, signedData string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, ok := s.transactions[txID]
	if !ok {
		return errors.New("transaction not found")
	}

	tx.Status = "signed"
	tx.UpdatedAt = time.Now()

	return nil
}

func (s *TransactionService) Broadcast(ctx context.Context, txID, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, ok := s.transactions[txID]
	if !ok {
		return errors.New("transaction not found")
	}

	tx.Hash = hash
	tx.Status = "broadcast"
	tx.UpdatedAt = time.Now()

	return nil
}

func (s *TransactionService) Confirm(ctx context.Context, txID string, blockNumber uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, ok := s.transactions[txID]
	if !ok {
		return errors.New("transaction not found")
	}

	tx.Status = "confirmed"
	tx.BlockNumber = &blockNumber
	tx.UpdatedAt = time.Now()

	return nil
}

func (s *TransactionService) Fail(ctx context.Context, txID, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, ok := s.transactions[txID]
	if !ok {
		return errors.New("transaction not found")
	}

	tx.Status = "failed"
	tx.Error = &errorMsg
	tx.UpdatedAt = time.Now()

	return nil
}

func (s *TransactionService) Cancel(ctx context.Context, txID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, ok := s.transactions[txID]
	if !ok {
		return errors.New("transaction not found")
	}

	tx.Status = "cancelled"
	tx.UpdatedAt = time.Now()

	return nil
}

// EstimateGas estimates gas for a transaction
func (s *TransactionService) EstimateGas(ctx context.Context, to, data string, value string) (string, string, error) {
	// Simplified gas estimation - in production, query RPC
	gasLimit := "21000"
	gasPrice := "50000000000" // 50 Gwei

	return gasLimit, gasPrice, nil
}

// GetNonce gets the next nonce for a wallet
func (s *TransactionService) GetNonce(ctx context.Context, walletID string) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var nonce uint64 = 0
	for _, tx := range s.transactions {
		if tx.WalletID == walletID {
			if tx.Nonce != nil && *tx.Nonce >= nonce {
				nonce = *tx.Nonce + 1
			}
		}
	}

	return nonce, nil
}

// CreateTransactionRequest represents a request to create a transaction
type CreateTransactionRequest struct {
	WalletID    string
	BlockchainID string
	Type        string
	From        string
	To          string
	TokenSymbol string
	Amount      string
	AmountUSD   float64
	Fee         string
	FeeUSD      float64
	Nonce       *uint64
	Data        string
}
