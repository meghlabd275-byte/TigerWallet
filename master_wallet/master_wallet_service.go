package master_wallet

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MasterWallet Types
// ============================================================================

// MasterWallet represents the main master wallet that controls all user wallets
type MasterWallet struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Type              string                 `json:"type"` // "tiger" or "custom"
	Description       string                 `json:"description"`
	AdminIDs          []string               `json:"admin_ids"`
	NetworkIDs        []string               `json:"network_ids"`
	TokenIDs          []string               `json:"token_ids"`
	CustomBranding    *CustomBranding        `json:"custom_branding,omitempty"`
	IsActive          bool                   `json:"is_active"`
	Settings          map[string]interface{} `json:"settings"`
	Statistics        *WalletStatistics      `json:"statistics"`
	CreatedAt         int64                  `json:"created_at"`
	UpdatedAt         int64                  `json:"updated_at"`
}

// CustomBranding holds custom branding configuration
type CustomBranding struct {
	BrandName        string `json:"brand_name"`
	BrandLogo        string `json:"brand_logo"`
	BrandColor       string `json:"brand_color"`
	BrandTagline     string `json:"brand_tagline"`
	SupportEmail     string `json:"support_email"`
	WebsiteURL       string `json:"website_url"`
	TermsOfService   string `json:"terms_of_service"`
	PrivacyPolicy    string `json:"privacy_policy"`
}

// UserWallet represents a user wallet under a master wallet
type UserWallet struct {
	ID                string                 `json:"id"`
	MasterWalletID    string                 `json:"master_wallet_id"`
	OwnerID           string                 `json:"owner_id"`
	WalletType        string                 `json:"wallet_type"` // "tiger" or "custom"
	Name              string                 `json:"name"`
	Addresses         map[string]string      `json:"addresses"` // chainID -> address
	Networks          []string               `json:"networks"`
	Tokens            []string               `json:"tokens"`
	IsActive          bool                   `json:"is_active"`
	CreatedAt         int64                  `json:"created_at"`
	UpdatedAt         int64                  `json:"updated_at"`
}

// WalletStatistics holds wallet statistics
type WalletStatistics struct {
	TotalUsers         int     `json:"total_users"`
	TotalTransactions  int64   `json:"total_transactions"`
	TotalVolume        float64 `json:"total_volume"`
	ActiveNetworks     int     `json:"active_networks"`
	ActiveTokens       int     `json:"active_tokens"`
}

// AdminAction represents an admin action log
type AdminAction struct {
	ID          string    `json:"id"`
	AdminID     string    `json:"admin_id"`
	Action      string    `json:"action"`
	Target      string    `json:"target"`
	Details     string    `json:"details"`
	Timestamp   int64     `json:"timestamp"`
}

// NetworkAddRequest represents a request to add a new network
type NetworkAddRequest struct {
	ID              string         `json:"id" validate:"required"`
	Name            string         `json:"name" validate:"required"`
	Symbol          string         `json:"symbol" validate:"required"`
	Type            BlockchainType `json:"type" validate:"required"`
	ChainID         int64          `json:"chain_id" validate:"required"`
	RPCURL          string         `json:"rpc_url" validate:"required"`
	Explorer        string         `json:"explorer"`
	WSSURL          string         `json:"wss_url"`
	IsTestnet       bool           `json:"is_testnet"`
	Confirmations   int            `json:"confirmations"`
	Decimals        int            `json:"decimals"`
	NativeCurrency  string         `json:"native_currency"`
}

// TokenAddRequest represents a request to add a new token
type TokenAddRequest struct {
	ID           string  `json:"id" validate:"required"`
	Address      string  `json:"address"`
	Name         string  `json:"name" validate:"required"`
	Symbol       string  `json:"symbol" validate:"required"`
	Decimals     int     `json:"decimals" validate:"required"`
	ChainID      int64   `json:"chain_id" validate:"required"`
	Type         string  `json:"type" validate:"required"`
	LogoURL      string  `json:"logo_url"`
	Website      string  `json:"website"`
	IsStableCoin bool    `json:"is_stable_coin"`
	IsWrapped    bool    `json:"is_wrapped"`
}

// MasterWalletService is the main service for managing master wallets
type MasterWalletService struct {
	mu            sync.RWMutex
	masterWallets map[string]*MasterWallet
	userWallets   map[string]*UserWallet
	adminActions  []AdminAction
	networkRegistry *BlockchainRegistry
	tokenRegistry   *TokenRegistry
}

var (
	masterWalletService     *MasterWalletService
	masterWalletServiceOnce sync.Once
)

// GetMasterWalletService returns the singleton master wallet service
func GetMasterWalletService() *MasterWalletService {
	masterWalletServiceOnce.Do(func() {
		masterWalletService = &MasterWalletService{
			masterWallets: make(map[string]*MasterWallet),
			userWallets:   make(map[string]*UserWallet),
			adminActions:  make([]AdminAction, 0),
			networkRegistry: GetRegistry(),
			tokenRegistry:   GetTokenRegistry(),
		}
		masterWalletService.initMasterWallets()
	})
	return masterWalletService
}

// initMasterWallets initializes the default master wallets
func (s *MasterWalletService) initMasterWallets() {
	now := time.Now().Unix()

	// Create TigerWallet MasterWallet
	tigerWallet := &MasterWallet{
		ID:          "tiger_master_" + uuid.New().String(),
		Name:        "TigerWallet",
		Type:        "tiger",
		Description: "Main TigerWallet - the primary master wallet for all TigerWallet users",
		AdminIDs:    []string{"admin_tiger_001"},
		NetworkIDs:  s.getAllNetworkIDs(),
		TokenIDs:    s.getAllTokenIDs(),
		CustomBranding: &CustomBranding{
			BrandName:    "TigerWallet",
			BrandLogo:    "https://tigerwallet.com/logo.png",
			BrandColor:   "#FF6B35",
			BrandTagline: "Your Gateway to DeFi",
			SupportEmail: "support@tigerwallet.com",
			WebsiteURL:   "https://tigerwallet.com",
		},
		IsActive: true,
		Settings: map[string]interface{}{
			"max_networks":          200,
			"max_tokens":            1000,
			"enable_bridge":         true,
			"enable_swap":           true,
			"enable_staking":        true,
			"enable_nft":            true,
			"fee_structure":         "standard",
			"support_email":         "support@tigerwallet.com",
		},
		Statistics: &WalletStatistics{
			TotalUsers:        0,
			TotalTransactions: 0,
			TotalVolume:       0,
			ActiveNetworks:    len(s.getAllNetworkIDs()),
			ActiveTokens:      len(s.getAllTokenIDs()),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.masterWallets[tigerWallet.ID] = tigerWallet
}

// getAllNetworkIDs returns all network IDs from the registry
func (s *MasterWalletService) getAllNetworkIDs() []string {
	networks := s.networkRegistry.GetAllNetworks()
	ids := make([]string, len(networks))
	for i, network := range networks {
		ids[i] = network.ID
	}
	return ids
}

// getAllTokenIDs returns all token IDs from the registry
func (s *MasterWalletService) getAllTokenIDs() []string {
	tokens := s.tokenRegistry.GetAllTokens()
	ids := make([]string, len(tokens))
	for i, token := range tokens {
		ids[i] = token.ID
	}
	return ids
}

// ============================================================================
// MasterWallet CRUD Operations
// ============================================================================

// CreateMasterWallet creates a new master wallet
func (s *MasterWalletService) CreateMasterWallet(name, description string, branding *CustomBranding, adminIDs []string) (*MasterWallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	masterWallet := &MasterWallet{
		ID:             "master_" + uuid.New().String(),
		Name:           name,
		Type:           "custom",
		Description:    description,
		AdminIDs:       adminIDs,
		NetworkIDs:     s.getAllNetworkIDs(),
		TokenIDs:       s.getAllTokenIDs(),
		CustomBranding: branding,
		IsActive:       true,
		Settings:       make(map[string]interface{}),
		Statistics: &WalletStatistics{
			ActiveNetworks: len(s.getAllNetworkIDs()),
			ActiveTokens:   len(s.getAllTokenIDs()),
		},
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	s.masterWallets[masterWallet.ID] = masterWallet
	
	s.logAdminAction("system", "create_master_wallet", masterWallet.ID, fmt.Sprintf("Created master wallet: %s", name))

	return masterWallet, nil
}

// GetMasterWallet returns a master wallet by ID
func (s *MasterWalletService) GetMasterWallet(id string) (*MasterWallet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wallet, ok := s.masterWallets[id]
	return wallet, ok
}

// GetMasterWalletByType returns a master wallet by type
func (s *MasterWalletService) GetMasterWalletByType(walletType string) *MasterWallet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, wallet := range s.masterWallets {
		if wallet.Type == walletType {
			return wallet
		}
	}
	return nil
}

// GetAllMasterWallets returns all master wallets
func (s *MasterWalletService) GetAllMasterWallets() []*MasterWallet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallets := make([]*MasterWallet, 0, len(s.masterWallets))
	for _, wallet := range s.masterWallets {
		wallets = append(wallets, wallet)
	}
	return wallets
}

// UpdateMasterWallet updates a master wallet
func (s *MasterWalletService) UpdateMasterWallet(id string, updates map[string]interface{}) (*MasterWallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.masterWallets[id]
	if !ok {
		return nil, fmt.Errorf("master wallet not found")
	}

	if name, ok := updates["name"].(string); ok {
		wallet.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		wallet.Description = description
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		wallet.IsActive = isActive
	}
	if settings, ok := updates["settings"].(map[string]interface{}); ok {
		for k, v := range settings {
			wallet.Settings[k] = v
		}
	}

	wallet.UpdatedAt = time.Now().Unix()
	s.masterWallets[id] = wallet

	s.logAdminAction("system", "update_master_wallet", id, "Updated master wallet")

	return wallet, nil
}

// DeleteMasterWallet deletes a master wallet
func (s *MasterWalletService) DeleteMasterWallet(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.masterWallets[id]; !ok {
		return fmt.Errorf("master wallet not found")
	}

	delete(s.masterWallets, id)
	s.logAdminAction("system", "delete_master_wallet", id, "Deleted master wallet")

	return nil
}

// ============================================================================
// Network Management
// ============================================================================

// AddNetwork adds a new network to a master wallet
func (s *MasterWalletService) AddNetwork(masterWalletID string, req NetworkAddRequest) (*Network, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.masterWallets[masterWalletID]
	if !ok {
		return nil, fmt.Errorf("master wallet not found")
	}

	// Check if network already exists in registry
	network := &Network{
		ID:             req.ID,
		Name:           req.Name,
		Symbol:         req.Symbol,
		Type:           req.Type,
		ChainID:        req.ChainID,
		RPCURL:         req.RPCURL,
		Explorer:       req.Explorer,
		WSSURL:         req.WSSURL,
		IsTestnet:      req.IsTestnet,
		Confirmations:   req.Confirmations,
		Decimals:       req.Decimals,
		NativeToken:    req.NativeCurrency,
		StableCoins:    []string{},
		AddedAt:        time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
	}

	// Add to global registry
	if err := s.networkRegistry.AddNetwork(network); err != nil {
		return nil, err
	}

	// Add to master wallet's network list
	wallet.NetworkIDs = append(wallet.NetworkIDs, req.ID)
	wallet.Statistics.ActiveNetworks = len(wallet.NetworkIDs)
	wallet.UpdatedAt = time.Now().Unix()

	s.logAdminAction("system", "add_network", masterWalletID, fmt.Sprintf("Added network: %s", req.Name))

	return network, nil
}

// RemoveNetwork removes a network from a master wallet
func (s *MasterWalletService) RemoveNetwork(masterWalletID, networkID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.masterWallets[masterWalletID]
	if !ok {
		return fmt.Errorf("master wallet not found")
	}

	// Remove from master wallet's network list
	newNetworkIDs := []string{}
	for _, id := range wallet.NetworkIDs {
		if id != networkID {
			newNetworkIDs = append(newNetworkIDs, id)
		}
	}
	wallet.NetworkIDs = newNetworkIDs
	wallet.Statistics.ActiveNetworks = len(wallet.NetworkIDs)
	wallet.UpdatedAt = time.Now().Unix()

	s.logAdminAction("system", "remove_network", masterWalletID, fmt.Sprintf("Removed network: %s", networkID))

	return nil
}

// GetNetworks returns networks for a master wallet
func (s *MasterWalletService) GetNetworks(masterWalletID string) ([]*Network, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallet, ok := s.masterWallets[masterWalletID]
	if !ok {
		return nil, fmt.Errorf("master wallet not found")
	}

	networks := make([]*Network, 0)
	for _, networkID := range wallet.NetworkIDs {
		if network, ok := s.networkRegistry.GetNetwork(networkID); ok {
			networks = append(networks, network)
		}
	}

	return networks, nil
}

// ============================================================================
// Token Management
// ============================================================================

// AddToken adds a new token to a master wallet
func (s *MasterWalletService) AddToken(masterWalletID string, req TokenAddRequest) (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.masterWallets[masterWalletID]
	if !ok {
		return nil, fmt.Errorf("master wallet not found")
	}

	// Get network info
	network, ok := s.networkRegistry.GetNetworkByChainID(req.ChainID)
	if !ok {
		return nil, fmt.Errorf("chain not found")
	}

	token := &Token{
		ID:            req.ID,
		Address:       req.Address,
		Name:          req.Name,
		Symbol:        req.Symbol,
		Decimals:      req.Decimals,
		ChainID:       req.ChainID,
		ChainSymbol:   network.Symbol,
		Type:          req.Type,
		IsStableCoin:  req.IsStableCoin,
		IsWrapped:     req.IsWrapped,
		IsVerified:    true,
		LogoURL:       req.LogoURL,
		Website:       req.Website,
		AddedAt:       time.Now().Unix(),
	}

	// Add to global registry
	if err := s.tokenRegistry.AddToken(token); err != nil {
		return nil, err
	}

	// Add to master wallet's token list
	wallet.TokenIDs = append(wallet.TokenIDs, req.ID)
	wallet.Statistics.ActiveTokens = len(wallet.TokenIDs)
	wallet.UpdatedAt = time.Now().Unix()

	s.logAdminAction("system", "add_token", masterWalletID, fmt.Sprintf("Added token: %s", req.Name))

	return token, nil
}

// RemoveToken removes a token from a master wallet
func (s *MasterWalletService) RemoveToken(masterWalletID, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.masterWallets[masterWalletID]
	if !ok {
		return fmt.Errorf("master wallet not found")
	}

	// Remove from master wallet's token list
	newTokenIDs := []string{}
	for _, id := range wallet.TokenIDs {
		if id != tokenID {
			newTokenIDs = append(newTokenIDs, id)
		}
	}
	wallet.TokenIDs = newTokenIDs
	wallet.Statistics.ActiveTokens = len(wallet.TokenIDs)
	wallet.UpdatedAt = time.Now().Unix()

	s.logAdminAction("system", "remove_token", masterWalletID, fmt.Sprintf("Removed token: %s", tokenID))

	return nil
}

// GetTokens returns tokens for a master wallet
func (s *MasterWalletService) GetTokens(masterWalletID string) ([]*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallet, ok := s.masterWallets[masterWalletID]
	if !ok {
		return nil, fmt.Errorf("master wallet not found")
	}

	tokens := make([]*Token, 0)
	for _, tokenID := range wallet.TokenIDs {
		if token, ok := s.tokenRegistry.GetToken(tokenID); ok {
			tokens = append(tokens, token)
		}
	}

	return tokens, nil
}

// ============================================================================
// User Wallet Management
// ============================================================================

// CreateUserWallet creates a new user wallet under a master wallet
func (s *MasterWalletService) CreateUserWallet(masterWalletID, ownerID, walletName string) (*UserWallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	masterWallet, ok := s.masterWallets[masterWalletID]
	if !ok {
		return nil, fmt.Errorf("master wallet not found")
	}

	userWallet := &UserWallet{
		ID:             "user_wallet_" + uuid.New().String(),
		MasterWalletID: masterWalletID,
		OwnerID:        ownerID,
		WalletType:     masterWallet.Type,
		Name:           walletName,
		Addresses:      make(map[string]string),
		Networks:       masterWallet.NetworkIDs,
		Tokens:         masterWallet.TokenIDs,
		IsActive:       true,
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
	}

	s.userWallets[userWallet.ID] = userWallet
	
	// Update master wallet statistics
	masterWallet.Statistics.TotalUsers++
	masterWallet.UpdatedAt = time.Now().Unix()

	s.logAdminAction("system", "create_user_wallet", masterWalletID, fmt.Sprintf("Created user wallet for owner: %s", ownerID))

	return userWallet, nil
}

// GetUserWallet returns a user wallet by ID
func (s *MasterWalletService) GetUserWallet(id string) (*UserWallet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wallet, ok := s.userWallets[id]
	return wallet, ok
}

// GetUserWalletsByOwner returns all user wallets for an owner
func (s *MasterWalletService) GetUserWalletsByOwner(ownerID string) []*UserWallet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallets := make([]*UserWallet, 0)
	for _, wallet := range s.userWallets {
		if wallet.OwnerID == ownerID {
			wallets = append(wallets, wallet)
		}
	}
	return wallets
}

// GetUserWalletsByMasterWallet returns all user wallets for a master wallet
func (s *MasterWalletService) GetUserWalletsByMasterWallet(masterWalletID string) []*UserWallet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallets := make([]*UserWallet, 0)
	for _, wallet := range s.userWallets {
		if wallet.MasterWalletID == masterWalletID {
			wallets = append(wallets, wallet)
		}
	}
	return wallets
}

// UpdateUserWallet updates a user wallet
func (s *MasterWalletService) UpdateUserWallet(id string, updates map[string]interface{}) (*UserWallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.userWallets[id]
	if !ok {
		return nil, fmt.Errorf("user wallet not found")
	}

	if name, ok := updates["name"].(string); ok {
		wallet.Name = name
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		wallet.IsActive = isActive
	}
	if networks, ok := updates["networks"].([]string); ok {
		wallet.Networks = networks
	}
	if tokens, ok := updates["tokens"].([]string); ok {
		wallet.Tokens = tokens
	}

	wallet.UpdatedAt = time.Now().Unix()
	s.userWallets[id] = wallet

	return wallet, nil
}

// DeleteUserWallet deletes a user wallet
func (s *MasterWalletService) DeleteUserWallet(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.userWallets[id]
	if !ok {
		return fmt.Errorf("user wallet not found")
	}

	// Update master wallet statistics
	if masterWallet, ok := s.masterWallets[wallet.MasterWalletID]; ok {
		masterWallet.Statistics.TotalUsers--
	}

	delete(s.userWallets, id)
	return nil
}

// ============================================================================
// Statistics
// ============================================================================

// GetStatistics returns statistics for a master wallet
func (s *MasterWalletService) GetStatistics(masterWalletID string) (*WalletStatistics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallet, ok := s.masterWallets[masterWalletID]
	if !ok {
		return nil, fmt.Errorf("master wallet not found")
	}

	return wallet.Statistics, nil
}

// GetGlobalStatistics returns global statistics
func (s *MasterWalletService) GetGlobalStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_master_wallets": len(s.masterWallets),
		"total_user_wallets":   len(s.userWallets),
		"total_networks":       s.networkRegistry.GetSupportedChains(),
		"total_tokens":         s.tokenRegistry.GetTokenCount(),
		"active_networks":      s.networkRegistry.GetActiveChainCount(),
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// logAdminAction logs an admin action
func (s *MasterWalletService) logAdminAction(adminID, action, target, details string) {
	adminAction := AdminAction{
		ID:        uuid.New().String(),
		AdminID:   adminID,
		Action:    action,
		Target:    target,
		Details:   details,
		Timestamp: time.Now().Unix(),
	}
	s.adminActions = append(s.adminActions, adminAction)
}

// GetAdminActions returns admin action logs
func (s *MasterWalletService) GetAdminActions(limit int) []AdminAction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit > len(s.adminActions) {
		limit = len(s.adminActions)
	}
	return s.adminActions[len(s.adminActions)-limit:]
}

// ============================================================================
// JSON Export
// ============================================================================

// GetMasterWalletJSON returns a master wallet as JSON
func (s *MasterWalletService) GetMasterWalletJSON(id string) (string, error) {
	wallet, ok := s.GetMasterWallet(id)
	if !ok {
		return "", fmt.Errorf("master wallet not found")
	}
	data, err := json.Marshal(wallet)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetUserWalletJSON returns a user wallet as JSON
func (s *MasterWalletService) GetUserWalletJSON(id string) (string, error) {
	wallet, ok := s.GetUserWallet(id)
	if !ok {
		return "", fmt.Errorf("user wallet not found")
	}
	data, err := json.Marshal(wallet)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetNetworkRegistryJSON returns network registry data as JSON
func (s *MasterWalletService) GetNetworkRegistryJSON() (string, error) {
	networks := s.networkRegistry.GetAllNetworks()
	data, err := json.Marshal(networks)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetTokenRegistryJSON returns token registry data as JSON
func (s *MasterWalletService) GetTokenRegistryJSON() (string, error) {
	tokens := s.tokenRegistry.GetAllTokens()
	data, err := json.Marshal(tokens)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetNetworksByChainType returns networks by type as JSON
func (s *MasterWalletService) GetNetworksByChainType(blockchainType BlockchainType) (string, error) {
	networks := s.networkRegistry.GetNetworksByType(blockchainType)
	data, err := json.Marshal(networks)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetTokensByChain returns tokens for a specific chain as JSON
func (s *MasterWalletService) GetTokensByChain(chainID int64) (string, error) {
	tokens := s.tokenRegistry.GetTokensByChain(chainID)
	data, err := json.Marshal(tokens)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
