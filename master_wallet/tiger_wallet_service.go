package master_wallet

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// TigerWallet Types
// ============================================================================

// Wallet represents a user's wallet
type Wallet struct {
	ID               string                 `json:"id"`
	MasterWalletID  string                 `json:"master_wallet_id"`
	UserID          string                 `json:"user_id"`
	WalletType      string                 `json:"wallet_type"` // "tiger" or "custom"
	Name            string                 `json:"name"`
	Addresses       map[string]string      `json:"addresses"` // chainID -> address
	PublicKeys      map[string]string      `json:"public_keys"`
	Networks        []string               `json:"networks"`
	Tokens          []string               `json:"tokens"`
	Balance         map[string]float64     `json:"balance"` // tokenID -> balance
	IsActive        bool                   `json:"is_active"`
	IsLocked        bool                   `json:"is_locked"`
	CreatedAt       int64                  `json:"created_at"`
	UpdatedAt       int64                  `json:"updated_at"`
}

// Transaction represents a wallet transaction
type Transaction struct {
	ID             string                 `json:"id"`
	WalletID       string                 `json:"wallet_id"`
	Hash           string                 `json:"hash"`
	ChainID        int64                  `json:"chain_id"`
	From           string                 `json:"from"`
	To             string                 `json:"to"`
	TokenID        string                 `json:"token_id"`
	Amount         float64                `json:"amount"`
	Fee            float64                `json:"fee"`
	Status         string                 `json:"status"` // "pending", "confirmed", "failed"
	Type           string                 `json:"type"` // "send", "receive", "swap", "stake"
	Timestamp      int64                  `json:"timestamp"`
	BlockNumber    int64                  `json:"block_number"`
	Confirmations  int                    `json:"confirmations"`
}

// WalletSettings represents wallet settings
type WalletSettings struct {
	WalletID             string   `json:"wallet_id"`
	EnableDefi           bool     `json:"enable_defi"`
	EnableNFT            bool     `json:"enable_nft"`
	EnableStaking        bool     `json:"enable_staking"`
	EnableBridge         bool     `json:"enable_bridge"`
	AutoLockMinutes      int      `json:"auto_lock_minutes"`
	RequireBiometric     bool     `json:"require_biometric"`
	ShowTestnets        bool     `json:"show_testnets"`
	DefaultChain        string   `json:"default_chain"`
	Currency            string   `json:"currency"` // "USD", "EUR", "GBP", etc.
	Theme               string   `json:"theme"` // "light", "dark", "auto"
}

// ============================================================================
// TigerWallet Service
// ============================================================================

// TigerWalletService provides wallet functionality for users
type TigerWalletService struct {
	mu              sync.RWMutex
	wallets         map[string]*Wallet
	transactions    map[string][]*Transaction
	walletSettings  map[string]*WalletSettings
	masterService   *MasterWalletService
}

var (
	tigerWalletService     *TigerWalletService
	tigerWalletServiceOnce sync.Once
)

// GetTigerWalletService returns the singleton tiger wallet service
func GetTigerWalletService() *TigerWalletService {
	tigerWalletServiceOnce.Do(func() {
		tigerWalletService = &TigerWalletService{
			wallets:        make(map[string]*Wallet),
			transactions:   make(map[string][]*Transaction),
			walletSettings: make(map[string]*WalletSettings),
			masterService:  GetMasterWalletService(),
		}
	})
	return tigerWalletService
}

// ============================================================================
// Wallet Operations
// ============================================================================

// CreateWallet creates a new wallet for a user
func (s *TigerWalletService) CreateWallet(userID, name, walletType string) (*Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get master wallet
	masterWallet := s.masterService.GetMasterWalletByType(walletType)
	if masterWallet == nil {
		return nil, fmt.Errorf("master wallet not found for type: %s", walletType)
	}

	now := time.Now().Unix()
	wallet := &Wallet{
		ID:              "wallet_" + uuid.New().String(),
		MasterWalletID:  masterWallet.ID,
		UserID:          userID,
		WalletType:      walletType,
		Name:            name,
		Addresses:       make(map[string]string),
		PublicKeys:      make(map[string]string),
		Networks:        masterWallet.NetworkIDs,
		Tokens:          masterWallet.TokenIDs,
		Balance:         make(map[string]float64),
		IsActive:        true,
		IsLocked:        false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	s.wallets[wallet.ID] = wallet
	
	// Initialize default settings
	s.walletSettings[wallet.ID] = &WalletSettings{
		WalletID:         wallet.ID,
		EnableDefi:       true,
		EnableNFT:        true,
		EnableStaking:    true,
		EnableBridge:     true,
		AutoLockMinutes:  15,
		RequireBiometric: false,
		ShowTestnets:    false,
		DefaultChain:    "ethereum",
		Currency:        "USD",
		Theme:           "dark",
	}

	return wallet, nil
}

// GetWallet returns a wallet by ID
func (s *TigerWalletService) GetWallet(id string) (*Wallet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wallet, ok := s.wallets[id]
	return wallet, ok
}

// GetWalletByUser returns wallets for a user
func (s *TigerWalletService) GetWalletByUser(userID string) []*Wallet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallets := make([]*Wallet, 0)
	for _, wallet := range s.wallets {
		if wallet.UserID == userID {
			wallets = append(wallets, wallet)
		}
	}
	return wallets
}

// GetWalletsByMasterWallet returns wallets for a master wallet
func (s *TigerWalletService) GetWalletsByMasterWallet(masterWalletID string) []*Wallet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallets := make([]*Wallet, 0)
	for _, wallet := range s.wallets {
		if wallet.MasterWalletID == masterWalletID {
			wallets = append(wallets, wallet)
		}
	}
	return wallets
}

// UpdateWallet updates a wallet
func (s *TigerWalletService) UpdateWallet(id string, updates map[string]interface{}) (*Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.wallets[id]
	if !ok {
		return nil, fmt.Errorf("wallet not found")
	}

	if name, ok := updates["name"].(string); ok {
		wallet.Name = name
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		wallet.IsActive = isActive
	}
	if isLocked, ok := updates["is_locked"].(bool); ok {
		wallet.IsLocked = isLocked
	}
	if networks, ok := updates["networks"].([]string); ok {
		wallet.Networks = networks
	}
	if tokens, ok := updates["tokens"].([]string); ok {
		wallet.Tokens = tokens
	}
	if addresses, ok := updates["addresses"].(map[string]string); ok {
		wallet.Addresses = addresses
	}

	wallet.UpdatedAt = time.Now().Unix()
	s.wallets[id] = wallet

	return wallet, nil
}

// DeleteWallet deletes a wallet
func (s *TigerWalletService) DeleteWallet(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.wallets[id]; !ok {
		return fmt.Errorf("wallet not found")
	}

	delete(s.wallets, id)
	delete(s.transactions, id)
	delete(s.walletSettings, id)

	return nil
}

// AddAddress adds an address for a chain
func (s *TigerWalletService) AddAddress(walletID, chainID, address, publicKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.wallets[walletID]
	if !ok {
		return fmt.Errorf("wallet not found")
	}

	wallet.Addresses[chainID] = address
	wallet.PublicKeys[chainID] = publicKey
	wallet.UpdatedAt = time.Now().Unix()

	return nil
}

// GetAddress returns an address for a chain
func (s *TigerWalletService) GetAddress(walletID, chainID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallet, ok := s.wallets[walletID]
	if !ok {
		return "", false
	}

	address, ok := wallet.Addresses[chainID]
	return address, ok
}

// ============================================================================
// Balance Operations
// ============================================================================

// UpdateBalance updates the balance for a token
func (s *TigerWalletService) UpdateBalance(walletID, tokenID string, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.wallets[walletID]
	if !ok {
		return fmt.Errorf("wallet not found")
	}

	wallet.Balance[tokenID] = amount
	wallet.UpdatedAt = time.Now().Unix()

	return nil
}

// GetBalance returns the balance for a token
func (s *TigerWalletService) GetBalance(walletID, tokenID string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallet, ok := s.wallets[walletID]
	if !ok {
		return 0
	}

	return wallet.Balance[tokenID]
}

// GetAllBalances returns all balances for a wallet
func (s *TigerWalletService) GetAllBalances(walletID string) map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallet, ok := s.wallets[walletID]
	if !ok {
		return make(map[string]float64)
	}

	balances := make(map[string]float64)
	for k, v := range wallet.Balance {
		balances[k] = v
	}
	return balances
}

// ============================================================================
// Transaction Operations
// ============================================================================

// CreateTransaction creates a new transaction
func (s *TigerWalletService) CreateTransaction(walletID, hash string, chainID int64, from, to, tokenID string, amount, fee float64, txType string) (*Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.wallets[walletID]
	if !ok {
		return nil, fmt.Errorf("wallet not found")
	}

	tx := &Transaction{
		ID:            "tx_" + uuid.New().String(),
		WalletID:      walletID,
		Hash:          hash,
		ChainID:       chainID,
		From:          from,
		To:            to,
		TokenID:       tokenID,
		Amount:        amount,
		Fee:           fee,
		Status:        "pending",
		Type:          txType,
		Timestamp:     time.Now().Unix(),
		Confirmations: 0,
	}

	s.transactions[walletID] = append(s.transactions[walletID], tx)

	// Update balance
	wallet.Balance[tokenID] = wallet.Balance[tokenID] - amount
	wallet.UpdatedAt = time.Now().Unix()

	return tx, nil
}

// GetTransaction returns a transaction by ID
func (s *TigerWalletService) GetTransaction(walletID, txID string) (*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	txs, ok := s.transactions[walletID]
	if !ok {
		return nil, fmt.Errorf("transactions not found")
	}

	for _, tx := range txs {
		if tx.ID == txID {
			return tx, nil
		}
	}

	return nil, fmt.Errorf("transaction not found")
}

// GetTransactions returns all transactions for a wallet
func (s *TigerWalletService) GetTransactions(walletID string) ([]*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	txs, ok := s.transactions[walletID]
	if !ok {
		return []*Transaction{}, nil
	}

	return txs, nil
}

// GetTransactionsByChain returns transactions for a specific chain
func (s *TigerWalletService) GetTransactionsByChain(walletID string, chainID int64) ([]*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	txs, ok := s.transactions[walletID]
	if !ok {
		return []*Transaction{}, nil
	}

	result := make([]*Transaction, 0)
	for _, tx := range txs {
		if tx.ChainID == chainID {
			result = append(result, tx)
		}
	}

	return result, nil
}

// UpdateTransactionStatus updates the status of a transaction
func (s *TigerWalletService) UpdateTransactionStatus(walletID, txID, status string, blockNumber int64, confirmations int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	txs, ok := s.transactions[walletID]
	if !ok {
		return fmt.Errorf("transactions not found")
	}

	for _, tx := range txs {
		if tx.ID == txID {
			tx.Status = status
			tx.BlockNumber = blockNumber
			tx.Confirmations = confirmations
			return nil
		}
	}

	return fmt.Errorf("transaction not found")
}

// ============================================================================
// Settings Operations
// ============================================================================

// GetSettings returns wallet settings
func (s *TigerWalletService) GetSettings(walletID string) (*WalletSettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	settings, ok := s.walletSettings[walletID]
	if !ok {
		return nil, fmt.Errorf("settings not found")
	}

	return settings, nil
}

// UpdateSettings updates wallet settings
func (s *TigerWalletService) UpdateSettings(walletID string, updates map[string]interface{}) (*WalletSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, ok := s.walletSettings[walletID]
	if !ok {
		return nil, fmt.Errorf("settings not found")
	}

	if enableDefi, ok := updates["enable_defi"].(bool); ok {
		settings.EnableDefi = enableDefi
	}
	if enableNFT, ok := updates["enable_nft"].(bool); ok {
		settings.EnableNFT = enableNFT
	}
	if enableStaking, ok := updates["enable_staking"].(bool); ok {
		settings.EnableStaking = enableStaking
	}
	if enableBridge, ok := updates["enable_bridge"].(bool); ok {
		settings.EnableBridge = enableBridge
	}
	if autoLockMinutes, ok := updates["auto_lock_minutes"].(int); ok {
		settings.AutoLockMinutes = autoLockMinutes
	}
	if requireBiometric, ok := updates["require_biometric"].(bool); ok {
		settings.RequireBiometric = requireBiometric
	}
	if showTestnets, ok := updates["show_testnets"].(bool); ok {
		settings.ShowTestnets = showTestnets
	}
	if defaultChain, ok := updates["default_chain"].(string); ok {
		settings.DefaultChain = defaultChain
	}
	if currency, ok := updates["currency"].(string); ok {
		settings.Currency = currency
	}
	if theme, ok := updates["theme"].(string); ok {
		settings.Theme = theme
	}

	s.walletSettings[walletID] = settings

	return settings, nil
}

// ============================================================================
// Network and Token Access
// ============================================================================

// GetNetworks returns available networks for a wallet
func (s *TigerWalletService) GetNetworks(walletID string) ([]*Network, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallet, ok := s.wallets[walletID]
	if !ok {
		return nil, fmt.Errorf("wallet not found")
	}

	networks := make([]*Network, 0)
	for _, networkID := range wallet.Networks {
		if network, ok := s.masterService.networkRegistry.GetNetwork(networkID); ok {
			networks = append(networks, network)
		}
	}

	return networks, nil
}

// GetTokens returns available tokens for a wallet
func (s *TigerWalletService) GetTokens(walletID string) ([]*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallet, ok := s.wallets[walletID]
	if !ok {
		return nil, fmt.Errorf("wallet not found")
	}

	tokens := make([]*Token, 0)
	for _, tokenID := range wallet.Tokens {
		if token, ok := s.masterService.tokenRegistry.GetToken(tokenID); ok {
			tokens = append(tokens, token)
		}
	}

	return tokens, nil
}

// GetTokensByChain returns tokens for a specific chain
func (s *TigerWalletService) GetTokensByChain(walletID string, chainID int64) ([]*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallet, ok := s.wallets[walletID]
	if !ok {
		return nil, fmt.Errorf("wallet not found")
	}

	tokens := make([]*Token, 0)
	for _, tokenID := range wallet.Tokens {
		if token, ok := s.masterService.tokenRegistry.GetToken(tokenID); ok {
			if token.ChainID == chainID {
				tokens = append(tokens, token)
			}
		}
	}

	return tokens, nil
}

// ============================================================================
// JSON Export
// ============================================================================

// GetWalletJSON returns a wallet as JSON
func (s *TigerWalletService) GetWalletJSON(id string) (string, error) {
	wallet, ok := s.GetWallet(id)
	if !ok {
		return "", fmt.Errorf("wallet not found")
	}
	data, err := json.Marshal(wallet)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetTransactionsJSON returns transactions as JSON
func (s *TigerWalletService) GetTransactionsJSON(walletID string) (string, error) {
	txs, err := s.GetTransactions(walletID)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(txs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetSettingsJSON returns settings as JSON
func (s *TigerWalletService) GetSettingsJSON(walletID string) (string, error) {
	settings, err := s.GetSettings(walletID)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetNetworksJSON returns networks as JSON
func (s *TigerWalletService) GetNetworksJSON(walletID string) (string, error) {
	networks, err := s.GetNetworks(walletID)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(networks)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetTokensJSON returns tokens as JSON
func (s *TigerWalletService) GetTokensJSON(walletID string) (string, error) {
	tokens, err := s.GetTokens(walletID)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(tokens)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ============================================================================
// Statistics
// ============================================================================

// GetWalletCount returns the total number of wallets
func (s *TigerWalletService) GetWalletCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.wallets)
}

// GetUserCount returns the total number of unique users
func (s *TigerWalletService) GetUserCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make(map[string]bool)
	for _, wallet := range s.wallets {
		users[wallet.UserID] = true
	}
	return len(users)
}

// GetTotalVolume returns the total transaction volume
func (s *TigerWalletService) GetTotalVolume() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var total float64
	for _, txs := range s.transactions {
		for _, tx := range txs {
			if tx.Status == "confirmed" {
				total += tx.Amount
			}
		}
	}
	return total
}
