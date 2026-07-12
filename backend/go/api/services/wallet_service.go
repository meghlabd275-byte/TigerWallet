package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"tigerwallet/backend/go/api/models"
)

// WalletService handles wallet operations
type WalletService struct {
	mu      sync.RWMutex
	wallets map[string]*models.Wallet
}

var (
	walletInstance *WalletService
	walletOnce    sync.Once
)

func NewWalletService() *WalletService {
	walletOnce.Do(func() {
		walletInstance = &WalletService{
			wallets: make(map[string]*models.Wallet),
		}
	})
	return walletInstance
}

func (s *WalletService) Create(ctx context.Context, userID, blockchainID, walletType, derivationPath string) (*models.Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate unique wallet ID
	walletID := fmt.Sprintf("wallet_%d", time.Now().UnixNano())

	// Generate address (simplified - in production, use proper crypto)
	address := s.generateAddress(blockchainID)

	wallet := &models.Wallet{
		ID:                 walletID,
		UserID:             userID,
		Type:               walletType,
		Address:            address,
		BlockchainID:       blockchainID,
		PublicKey:          "", // Would be generated from private key
		EncryptedPrivateKey: "",
		DerivationPath:    derivationPath,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		IsActive:           true,
	}

	s.wallets[walletID] = wallet
	return wallet, nil
}

func (s *WalletService) generateAddress(blockchainID string) string {
	// Simplified address generation - in production, use proper cryptographic derivation
	// For demonstration, generate a realistic-looking address
	chars := "0123456789abcdef"
	addr := "0x"
	for i := 0; i < 40; i++ {
		addr += string(chars[len(chars)-1])
	}
	return addr
}

func (s *WalletService) GetByID(ctx context.Context, walletID string) (*models.Wallet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallet, ok := s.wallets[walletID]
	if !ok {
		return nil, errors.New("wallet not found")
	}

	return wallet, nil
}

func (s *WalletService) GetByUserID(ctx context.Context, userID string) ([]*models.Wallet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Wallet
	for _, wallet := range s.wallets {
		if wallet.UserID == userID && wallet.IsActive {
			result = append(result, wallet)
		}
	}

	return result, nil
}

func (s *WalletService) GetByAddress(ctx context.Context, address string) (*models.Wallet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, wallet := range s.wallets {
		if wallet.Address == address && wallet.IsActive {
			return wallet, nil
		}
	}

	return nil, errors.New("wallet not found")
}

func (s *WalletService) Delete(ctx context.Context, walletID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.wallets[walletID]
	if !ok {
		return errors.New("wallet not found")
	}

	wallet.IsActive = false
	wallet.UpdatedAt = time.Now()

	return nil
}

func (s *WalletService) ImportWallet(ctx context.Context, userID, blockchainID, privateKeyHex, walletType string) (*models.Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	walletID := fmt.Sprintf("wallet_%d", time.Now().UnixNano())
	address := s.generateAddress(blockchainID)

	wallet := &models.Wallet{
		ID:                 walletID,
		UserID:             userID,
		Type:               walletType,
		Address:            address,
		BlockchainID:       blockchainID,
		PublicKey:          "",
		EncryptedPrivateKey: privateKeyHex, // In production, encrypt this
		DerivationPath:    "m/44'/60'/0'/0/0",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		IsActive:           true,
	}

	s.wallets[walletID] = wallet
	return wallet, nil
}

func (s *WalletService) ExportWallet(ctx context.Context, walletID, password string) (string, error) {
	s.mu.RLock()
	wallet, ok := s.wallets[walletID]
	s.mu.RUnlock()

	if !ok {
		return "", errors.New("wallet not found")
	}

	// In production, verify password and decrypt
	return wallet.EncryptedPrivateKey, nil
}
