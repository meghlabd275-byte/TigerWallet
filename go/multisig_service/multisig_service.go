/**
 * TigerWallet Multisig Service
 * 
 * Multi-signature transaction support.
 * Built with Go for high-load distributed operations.
 */

package multisig

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MultisigWallet represents a multisig wallet
type MultisigWallet struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Owners       []string `json:"owners"`
	RequiredSigs int      `json:"required_sigs"`
	Threshold    int      `json:"threshold"`
	ChainID      uint64   `json:"chain_id"`
	Address      string   `json:"address"`
	Status       string   `json:"status"`
	CreatedAt    int64    `json:"created_at"`
}

// MultisigTransaction represents a multisig transaction
type MultisigTransaction struct {
	ID          string   `json:"id"`
	WalletID    string   `json:"wallet_id"`
	To          string   `json:"to"`
	Value       string   `json:"value"`
	Data        string   `json:"data"`
	Signatures  []string `json:"signatures"`
	SignedBy    []string `json:"signed_by"`
	Status      string   `json:"status"`
	ExecutedAt  int64    `json:"executed_at"`
	CreatedAt   int64    `json:"created_at"`
}

// MultisigService manages multisig operations
type MultisigService struct {
	mu         sync.RWMutex
	wallets    map[string]*MultisigWallet
	transactions map[string]*MultisigTransaction
}

var (
	multisigService     *MultisigService
	multisigServiceOnce sync.Once
)

func GetMultisigService() *MultisigService {
	multisigServiceOnce.Do(func() {
		multisigService = &MultisigService{
			wallets:      make(map[string]*MultisigWallet),
			transactions: make(map[string]*MultisigTransaction),
		}
	})
	return multisigService
}

func (s *MultisigService) CreateWallet(ctx context.Context, wallet *MultisigWallet) (*MultisigWallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(wallet.Owners) < wallet.RequiredSigs {
		return nil, fmt.Errorf("owners must be >= required signatures")
	}

	wallet.ID = "multisig_" + uuid.New().String()
	wallet.Status = "active"
	wallet.CreatedAt = time.Now().Unix()

	s.wallets[wallet.ID] = wallet
	return wallet, nil
}

func (s *MultisigService) CreateTransaction(ctx context.Context, tx *MultisigTransaction) (*MultisigTransaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, exists := s.wallets[tx.WalletID]
	if !exists {
		return nil, fmt.Errorf("wallet not found")
	}

	tx.ID = "tx_" + uuid.New().String()
	tx.Status = "pending"
	tx.CreatedAt = time.Now().Unix()

	s.transactions[tx.ID] = tx
	return tx, nil
}

func (s *MultisigService) SignTransaction(ctx context.Context, txID, signature, signer string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, exists := s.transactions[txID]
	if !exists {
		return fmt.Errorf("transaction not found")
	}

	if tx.Status == "executed" {
		return fmt.Errorf("transaction already executed")
	}

	wallet, _ := s.wallets[tx.WalletID]
	if wallet == nil {
		return fmt.Errorf("wallet not found")
	}

	// Check if signer is an owner
	isOwner := false
	for _, owner := range wallet.Owners {
		if owner == signer {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return fmt.Errorf("signer is not an owner")
	}

	// Check if already signed
	for _, signed := range tx.SignedBy {
		if signed == signer {
			return fmt.Errorf("already signed")
		}
	}

	tx.Signatures = append(tx.Signatures, signature)
	tx.SignedBy = append(tx.SignedBy, signer)

	// Check if threshold reached
	if len(tx.SignedBy) >= wallet.RequiredSigs {
		tx.Status = "ready"
	}

	return nil
}

func (s *MultisigService) ExecuteTransaction(ctx context.Context, txID, txHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, exists := s.transactions[txID]
	if !exists {
		return fmt.Errorf("transaction not found")
	}

	if tx.Status == "executed" {
		return fmt.Errorf("transaction already executed")
	}

	wallet, _ := s.wallets[tx.WalletID]
	if wallet == nil {
		return fmt.Errorf("wallet not found")
	}

	if len(tx.SignedBy) < wallet.RequiredSigs {
		return fmt.Errorf("insufficient signatures")
	}

	tx.Status = "executed"
	tx.ExecutedAt = time.Now().Unix()
	return nil
}

func (s *MultisigService) GetTransaction(ctx context.Context, txID string) (*MultisigTransaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tx, exists := s.transactions[txID]
	if !exists {
		return nil, fmt.Errorf("transaction not found")
	}
	return tx, nil
}

func (s *MultisigService) GetWalletTransactions(ctx context.Context, walletID string) ([]*MultisigTransaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*MultisigTransaction, 0)
	for _, tx := range s.transactions {
		if tx.WalletID == walletID {
			result = append(result, tx)
		}
	}
	return result, nil
}

func (w *MultisigWallet) ToJSON() (string, error) {
	data, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
