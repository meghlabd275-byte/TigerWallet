/**
 * Wallet Service - Core wallet operations
 */

package services

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/wallet-services/internal/cache"
	"github.com/tigerwallet/wallet-services/internal/models"
	"github.com/sirupsen/logrus"
)

type WalletService struct {
	db    *sql.DB
	cache *cache.RedisClient
}

func NewWalletService(db *sql.DB, cache *cache.RedisClient) *WalletService {
	return &WalletService{db: db, cache: cache}
}

// CreateWallet creates a new wallet for a user
func (s *WalletService) CreateWallet(ctx context.Context, userID, name, chainType string, derivationType string) (*models.Wallet, error) {
	wallet := &models.Wallet{
		ID:             uuid.New().String(),
		UserID:         userID,
		Name:           name,
		Type:           "hd",
		DerivationType: derivationType,
		ChainType:      chainType,
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Generate derivation path
	wallet.DerivationPath = s.getDerivationPath(chainType, derivationType)

	// In production, generate actual keys here using the C++ core or Rust SDK
	wallet.PublicKey = hex.EncodeToString([]byte("public_key_placeholder"))
	wallet.Address = s.generateAddress(chainType, wallet.PublicKey)

	query := `
		INSERT INTO wallets (id, user_id, name, type, derivation_type, public_key, address, derivation_path, chain_type, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	_, err := s.db.ExecContext(ctx, query,
		wallet.ID, wallet.UserID, wallet.Name, wallet.Type, wallet.DerivationType,
		wallet.PublicKey, wallet.Address, wallet.DerivationPath, wallet.ChainType,
		wallet.Status, wallet.CreatedAt, wallet.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	// Cache the wallet
	s.cache.CacheWallet(ctx, wallet.ID, wallet, 15*time.Minute)

	return wallet, nil
}

// GetWallet retrieves a wallet by ID
func (s *WalletService) GetWallet(ctx context.Context, walletID string) (*models.Wallet, error) {
	// Try cache first
	var wallet models.Wallet
	if err := s.cache.GetCachedWallet(ctx, walletID, &wallet); err == nil {
		return &wallet, nil
	}

	// Load from database
	query := `
		SELECT id, user_id, name, type, derivation_type, encrypted_seed, public_key, 
		       chain_type, chain_id, address, derivation_path, is_imported, is_watch_only, 
		       status, metadata, created_at, updated_at
		FROM wallets
		WHERE id = $1 AND deleted_at IS NULL
	`

	wallet = models.Wallet{}
	err := s.db.QueryRowContext(ctx, query, walletID).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.DerivationType,
		&wallet.EncryptedSeed, &wallet.PublicKey, &wallet.ChainType, &wallet.ChainID,
		&wallet.Address, &wallet.DerivationPath, &wallet.IsImported, &wallet.IsWatchOnly,
		&wallet.Status, &wallet.Metadata, &wallet.CreatedAt, &wallet.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("wallet not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	// Cache for future requests
	s.cache.CacheWallet(ctx, wallet.ID, &wallet, 15*time.Minute)

	return &wallet, nil
}

// ListWallets returns all wallets for a user
func (s *WalletService) ListWallets(ctx context.Context, userID string, chainType string, limit, offset int) ([]*models.Wallet, int, error) {
	countQuery := `SELECT COUNT(*) FROM wallets WHERE user_id = $1 AND deleted_at IS NULL`
	countArgs := []interface{}{userID}

	if chainType != "" {
		countQuery += " AND chain_type = $2"
		countArgs = append(countArgs, chainType)
	}

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count wallets: %w", err)
	}

	query := `
		SELECT id, user_id, name, type, derivation_type, encrypted_seed, public_key,
		       chain_type, chain_id, address, derivation_path, is_imported, is_watch_only,
		       status, metadata, created_at, updated_at
		FROM wallets
		WHERE user_id = $1 AND deleted_at IS NULL
	`

	args := []interface{}{userID}
	if chainType != "" {
		query += " AND chain_type = $2"
		args = append(args, chainType)
	}

	query += " ORDER BY created_at DESC LIMIT $" + fmt.Sprint(len(args)+1) + " OFFSET $" + fmt.Sprint(len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list wallets: %w", err)
	}
	defer rows.Close()

	var wallets []*models.Wallet
	for rows.Next() {
		wallet := &models.Wallet{}
		err := rows.Scan(
			&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.DerivationType,
			&wallet.EncryptedSeed, &wallet.PublicKey, &wallet.ChainType, &wallet.ChainID,
			&wallet.Address, &wallet.DerivationPath, &wallet.IsImported, &wallet.IsWatchOnly,
			&wallet.Status, &wallet.Metadata, &wallet.CreatedAt, &wallet.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan wallet: %w", err)
		}
		wallets = append(wallets, wallet)
	}

	return wallets, total, nil
}

// DeleteWallet marks a wallet as deleted
func (s *WalletService) DeleteWallet(ctx context.Context, walletID, userID string) error {
	query := `UPDATE wallets SET deleted_at = $1 WHERE id = $2 AND user_id = $3`

	_, err := s.db.ExecContext(ctx, query, time.Now(), walletID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete wallet: %w", err)
	}

	// Invalidate cache
	s.cache.Delete(ctx, cache.PrefixWallet+walletID)

	return nil
}

// GetBalance returns the balance for a wallet
func (s *WalletService) GetBalance(ctx context.Context, walletID, tokenAddress string) (*models.Balance, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("%s%s:%s", cache.PrefixBalance, walletID, tokenAddress)
	var cachedBalance models.Balance
	if err := s.cache.GetStruct(ctx, cacheKey, &cachedBalance); err == nil {
		return &cachedBalance, nil
	}

	// Load from database
	query := `
		SELECT id, wallet_id, token_address, symbol, name, decimals, balance, pending_balance, locked_balance, is_native, updated_at
		FROM wallet_balances
		WHERE wallet_id = $1 AND (token_address = $2 OR ($2 = '' AND is_native = true))
	`

	balance := models.Balance{}
	err := s.db.QueryRowContext(ctx, query, walletID, tokenAddress).Scan(
		&balance.ID, &balance.WalletID, &balance.TokenAddress, &balance.Symbol,
		&balance.Name, &balance.Decimals, &balance.Balance, &balance.PendingBalance,
		&balance.LockedBalance, &balance.IsNative, &balance.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return zero balance
		return &models.Balance{
			WalletID:   walletID,
			TokenAddress: tokenAddress,
			Balance:    "0",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	// Cache the balance
	s.cache.Set(ctx, cacheKey, &balance, 1*time.Minute)

	return &balance, nil
}

// UpdateBalance updates the balance for a wallet
func (s *WalletService) UpdateBalance(ctx context.Context, walletID, tokenAddress, balance string) error {
	query := `
		INSERT INTO wallet_balances (wallet_id, token_address, symbol, decimals, balance, is_native, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (wallet_id, token_address) 
		DO UPDATE SET balance = $5, updated_at = $7
	`

	symbol := "ETH"
	if tokenAddress != "" {
		symbol = "TOKEN"
	}

	_, err := s.db.ExecContext(ctx, query,
		walletID, tokenAddress, symbol, 18, balance,
		tokenAddress == "", time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("%s%s:%s", cache.PrefixBalance, walletID, tokenAddress)
	s.cache.Delete(ctx, cacheKey)

	return nil
}

// GetAllBalances returns all balances for a wallet
func (s *WalletService) GetAllBalances(ctx context.Context, walletID string) ([]*models.Balance, error) {
	query := `
		SELECT id, wallet_id, token_address, symbol, name, decimals, balance, pending_balance, locked_balance, is_native, updated_at
		FROM wallet_balances
		WHERE wallet_id = $1
		ORDER BY is_native DESC, balance DESC
	`

	rows, err := s.db.QueryContext(ctx, query, walletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balances: %w", err)
	}
	defer rows.Close()

	var balances []*models.Balance
	for rows.Next() {
		balance := &models.Balance{}
		err := rows.Scan(
			&balance.ID, &balance.WalletID, &balance.TokenAddress, &balance.Symbol,
			&balance.Name, &balance.Decimals, &balance.Balance, &balance.PendingBalance,
			&balance.LockedBalance, &balance.IsNative, &balance.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan balance: %w", err)
		}
		balances = append(balances, balance)
	}

	return balances, nil
}

// Helper functions

func (s *WalletService) getDerivationPath(chainType, derivationType string) string {
	switch strings.ToLower(chainType) {
	case "bitcoin":
		return "m/44'/0'/0'/0/0"
	case "ethereum", "evm":
		return "m/44'/60'/0'/0/0"
	case "solana":
		return "m/44'/501'/0'/0'"
	case "cosmos":
		return "m/44'/118'/0'/0/0'"
	case "tron":
		return "m/44'/195'/0'/0/0'"
	case "polkadot":
		return "//0//0"
	default:
		return "m/44'/60'/0'/0/0"
	}
}

func (s *WalletService) generateAddress(chainType, publicKey string) string {
	// Simplified - in production use proper address derivation
	switch strings.ToLower(chainType) {
	case "bitcoin":
		return "bc1q" + strings.Repeat("x", 38)
	case "ethereum", "evm", "polygon", "bsc":
		return "0x" + strings.Repeat("a", 40)
	case "solana":
		return strings.Repeat("1", 44)
	case "cosmos":
		return "cosmos1" + strings.Repeat("p", 38)
	case "tron":
		return "T" + strings.Repeat("K", 33)
	default:
		return "0x" + strings.Repeat("a", 40)
	}
}

// ExportWallet exports wallet data
func (s *WalletService) ExportWallet(ctx context.Context, walletID, userID string, encrypted bool) (map[string]interface{}, error) {
	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		return nil, err
	}

	if wallet.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	export := map[string]interface{}{
		"id":              wallet.ID,
		"name":            wallet.Name,
		"chain_type":      wallet.ChainType,
		"address":         wallet.Address,
		"derivation_path": wallet.DerivationPath,
		"created_at":      wallet.CreatedAt,
	}

	if encrypted && wallet.EncryptedSeed != "" {
		export["encrypted_seed"] = wallet.EncryptedSeed
	}

	return export, nil
}
