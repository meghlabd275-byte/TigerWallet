package master_wallet

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Custom Branding Wallet Types
// ============================================================================

// CustomBrandingWallet represents a white-label/custom branding wallet
type CustomBrandingWallet struct {
	ID                string                 `json:"id"`
	BrandID           string                 `json:"brand_id"`
	MasterWalletID    string                 `json:"master_wallet_id"`
	Name              string                 `json:"name"`
	BrandName         string                 `json:"brand_name"`
	BrandLogo         string                 `json:"brand_logo"`
	BrandColor        string                 `json:"brand_color"`
	BrandTagline      string                 `json:"brand_tagline"`
	SupportEmail      string                 `json:"support_email"`
	WebsiteURL        string                 `json:"website_url"`
	TermsOfService    string                 `json:"terms_of_service"`
	PrivacyPolicy    string                 `json:"privacy_policy"`
	CustomDomain     string                 `json:"custom_domain"`
	CustomCSS        string                 `json:"custom_css"`
	Features         map[string]bool        `json:"features"`
	NetworkIDs        []string               `json:"network_ids"`
	TokenIDs          []string               `json:"token_ids"`
	IsActive          bool                   `json:"is_active"`
	CreatedAt         int64                  `json:"created_at"`
	UpdatedAt         int64                  `json:"updated_at"`
}

// BrandAdmin represents an admin for a custom branding wallet
type BrandAdmin struct {
	ID                string   `json:"id"`
	BrandWalletID     string   `json:"brand_wallet_id"`
	Email             string   `json:"email"`
	Name              string   `json:"name"`
	Role              string   `json:"role"` // "owner", "admin", "support"
	Permissions       []string `json:"permissions"`
	IsActive          bool     `json:"is_active"`
	CreatedAt         int64    `json:"created_at"`
	UpdatedAt         int64    `json:"updated_at"`
}

// BrandUser represents a user in a custom branding wallet
type BrandUser struct {
	ID                string                 `json:"id"`
	BrandWalletID     string                 `json:"brand_wallet_id"`
	Email             string                 `json:"email"`
	Name              string                 `json:"name"`
	WalletAddresses   map[string]string      `json:"wallet_addresses"`
	KYCStatus         string                 `json:"kyc_status"` // "none", "pending", "verified", "rejected"
	KYCLevel          int                    `json:"kyc_level"`
	IsActive          bool                   `json:"is_active"`
	Settings          map[string]interface{} `json:"settings"`
	CreatedAt         int64                  `json:"created_at"`
	UpdatedAt         int64                  `json:"updated_at"`
}

// ============================================================================
// Custom Branding Service
// ============================================================================

// CustomBrandingService manages custom branding wallets
type CustomBrandingService struct {
	mu             sync.RWMutex
	brandWallets  map[string]*CustomBrandingWallet
	brandAdmins   map[string][]*BrandAdmin
	brandUsers    map[string][]*BrandUser
	masterService *MasterWalletService
}

var (
	customBrandingService     *CustomBrandingService
	customBrandingServiceOnce sync.Once
)

// GetCustomBrandingService returns the singleton custom branding service
func GetCustomBrandingService() *CustomBrandingService {
	customBrandingServiceOnce.Do(func() {
		customBrandingService = &CustomBrandingService{
			brandWallets: make(map[string]*CustomBrandingWallet),
			brandAdmins:  make(map[string][]*BrandAdmin),
			brandUsers:   make(map[string][]*BrandUser),
			masterService: GetMasterWalletService(),
		}
	})
	return customBrandingService
}

// ============================================================================
// Brand Wallet Operations
// ============================================================================

// CreateBrandWallet creates a new custom branding wallet
func (s *CustomBrandingService) CreateBrandWallet(brandName, brandLogo, brandColor, brandTagline, supportEmail, websiteURL string, adminEmail, adminName string) (*CustomBrandingWallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()

	// Create master wallet for this brand
	masterWallet, err := s.masterService.CreateMasterWallet(
		brandName,
		fmt.Sprintf("Custom branding wallet for %s", brandName),
		&CustomBranding{
			BrandName:    brandName,
			BrandLogo:    brandLogo,
			BrandColor:   brandColor,
			BrandTagline: brandTagline,
			SupportEmail: supportEmail,
			WebsiteURL:   websiteURL,
		},
		[]string{},
	)
	if err != nil {
		return nil, err
	}

	// Create brand wallet
	brandWallet := &CustomBrandingWallet{
		ID:               "brand_" + uuid.New().String(),
		BrandID:          "BRAND-" + strings.ToUpper(brandName)[:3] + "-" + uuid.New().String()[:8],
		MasterWalletID:   masterWallet.ID,
		Name:             brandName,
		BrandName:        brandName,
		BrandLogo:        brandLogo,
		BrandColor:       brandColor,
		BrandTagline:     brandTagline,
		SupportEmail:     supportEmail,
		WebsiteURL:       websiteURL,
		Features:         s.getDefaultFeatures(),
		NetworkIDs:       masterWallet.NetworkIDs,
		TokenIDs:         masterWallet.TokenIDs,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	s.brandWallets[brandWallet.ID] = brandWallet

	// Create brand admin
	admin := &BrandAdmin{
		ID:              "admin_" + uuid.New().String(),
		BrandWalletID:   brandWallet.ID,
		Email:           adminEmail,
		Name:            adminName,
		Role:            "owner",
		Permissions:     []string{"*"},
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	s.brandAdmins[brandWallet.ID] = append(s.brandAdmins[brandWallet.ID], admin)

	return brandWallet, nil
}

// getDefaultFeatures returns default features for a brand
func (s *CustomBrandingService) getDefaultFeatures() map[string]bool {
	return map[string]bool{
		"defi":           true,
		"nft":            true,
		"staking":        true,
		"bridge":         true,
		"swap":           true,
		"limit_orders":   true,
		"dca_bot":        true,
		"grid_trading":   true,
		"copy_trading":   false,
		"ai_assistant":   true,
		"analytics":       true,
		"portfolio":      true,
		"security":       true,
		"2fa":            true,
		"biometric":      true,
		"multi_sig":      false,
		"hardware_wallet": true,
	}
}

// GetBrandWallet returns a brand wallet by ID
func (s *CustomBrandingService) GetBrandWallet(id string) (*CustomBrandingWallet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wallet, ok := s.brandWallets[id]
	return wallet, ok
}

// GetBrandWalletByMasterID returns a brand wallet by master wallet ID
func (s *CustomBrandingService) GetBrandWalletByMasterID(masterWalletID string) *CustomBrandingWallet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, wallet := range s.brandWallets {
		if wallet.MasterWalletID == masterWalletID {
			return wallet
		}
	}
	return nil
}

// GetAllBrandWallets returns all brand wallets
func (s *CustomBrandingService) GetAllBrandWallets() []*CustomBrandingWallet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallets := make([]*CustomBrandingWallet, 0, len(s.brandWallets))
	for _, wallet := range s.brandWallets {
		wallets = append(wallets, wallet)
	}
	return wallets
}

// GetActiveBrandWallets returns all active brand wallets
func (s *CustomBrandingService) GetActiveBrandWallets() []*CustomBrandingWallet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallets := make([]*CustomBrandingWallet, 0)
	for _, wallet := range s.brandWallets {
		if wallet.IsActive {
			wallets = append(wallets, wallet)
		}
	}
	return wallets
}

// UpdateBrandWallet updates a brand wallet
func (s *CustomBrandingService) UpdateBrandWallet(id string, updates map[string]interface{}) (*CustomBrandingWallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.brandWallets[id]
	if !ok {
		return nil, fmt.Errorf("brand wallet not found")
	}

	if name, ok := updates["name"].(string); ok {
		wallet.Name = name
	}
	if brandName, ok := updates["brand_name"].(string); ok {
		wallet.BrandName = brandName
	}
	if brandLogo, ok := updates["brand_logo"].(string); ok {
		wallet.BrandLogo = brandLogo
	}
	if brandColor, ok := updates["brand_color"].(string); ok {
		wallet.BrandColor = brandColor
	}
	if brandTagline, ok := updates["brand_tagline"].(string); ok {
		wallet.BrandTagline = brandTagline
	}
	if supportEmail, ok := updates["support_email"].(string); ok {
		wallet.SupportEmail = supportEmail
	}
	if websiteURL, ok := updates["website_url"].(string); ok {
		wallet.WebsiteURL = websiteURL
	}
	if customDomain, ok := updates["custom_domain"].(string); ok {
		wallet.CustomDomain = customDomain
	}
	if customCSS, ok := updates["custom_css"].(string); ok {
		wallet.CustomCSS = customCSS
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		wallet.IsActive = isActive
	}
	if features, ok := updates["features"].(map[string]bool); ok {
		for k, v := range features {
			wallet.Features[k] = v
		}
	}

	wallet.UpdatedAt = time.Now().Unix()
	s.brandWallets[id] = wallet

	return wallet, nil
}

// DeleteBrandWallet deletes a brand wallet
func (s *CustomBrandingService) DeleteBrandWallet(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.brandWallets[id]; !ok {
		return fmt.Errorf("brand wallet not found")
	}

	delete(s.brandWallets, id)
	delete(s.brandAdmins, id)
	delete(s.brandUsers, id)

	return nil
}

// ============================================================================
// Brand Admin Operations
// ============================================================================

// AddBrandAdmin adds an admin to a brand wallet
func (s *CustomBrandingService) AddBrandAdmin(brandWalletID, email, name, role string, permissions []string) (*BrandAdmin, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.brandWallets[brandWalletID]; !ok {
		return nil, fmt.Errorf("brand wallet not found")
	}

	now := time.Now().Unix()
	admin := &BrandAdmin{
		ID:            "admin_" + uuid.New().String(),
		BrandWalletID: brandWalletID,
		Email:         email,
		Name:          name,
		Role:          role,
		Permissions:   permissions,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	s.brandAdmins[brandWalletID] = append(s.brandAdmins[brandWalletID], admin)

	return admin, nil
}

// GetBrandAdmins returns all admins for a brand wallet
func (s *CustomBrandingService) GetBrandAdmins(brandWalletID string) ([]*BrandAdmin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.brandWallets[brandWalletID]; !ok {
		return nil, fmt.Errorf("brand wallet not found")
	}

	return s.brandAdmins[brandWalletID], nil
}

// RemoveBrandAdmin removes an admin from a brand wallet
func (s *CustomBrandingService) RemoveBrandAdmin(brandWalletID, adminID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	admins, ok := s.brandAdmins[brandWalletID]
	if !ok {
		return fmt.Errorf("brand wallet not found")
	}

	newAdmins := make([]*BrandAdmin, 0)
	for _, admin := range admins {
		if admin.ID != adminID {
			newAdmins = append(newAdmins, admin)
		}
	}
	s.brandAdmins[brandWalletID] = newAdmins

	return nil
}

// ============================================================================
// Brand User Operations
// ============================================================================

// AddBrandUser adds a user to a brand wallet
func (s *CustomBrandingService) AddBrandUser(brandWalletID, email, name string) (*BrandUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.brandWallets[brandWalletID]; !ok {
		return nil, fmt.Errorf("brand wallet not found")
	}

	now := time.Now().Unix()
	user := &BrandUser{
		ID:                "user_" + uuid.New().String(),
		BrandWalletID:     brandWalletID,
		Email:             email,
		Name:              name,
		WalletAddresses:   make(map[string]string),
		KYCStatus:         "none",
		KYCLevel:          0,
		IsActive:          true,
		Settings:          make(map[string]interface{}),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	s.brandUsers[brandWalletID] = append(s.brandUsers[brandWalletID], user)

	return user, nil
}

// GetBrandUsers returns all users for a brand wallet
func (s *CustomBrandingService) GetBrandUsers(brandWalletID string) ([]*BrandUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.brandWallets[brandWalletID]; !ok {
		return nil, fmt.Errorf("brand wallet not found")
	}

	return s.brandUsers[brandWalletID], nil
}

// GetBrandUser returns a specific user for a brand wallet
func (s *CustomBrandingService) GetBrandUser(brandWalletID, userID string) (*BrandUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users, ok := s.brandUsers[brandWalletID]
	if !ok {
		return nil, fmt.Errorf("brand wallet not found")
	}

	for _, user := range users {
		if user.ID == userID {
			return user, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

// UpdateBrandUser updates a brand user
func (s *CustomBrandingService) UpdateBrandUser(brandWalletID, userID string, updates map[string]interface{}) (*BrandUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	users, ok := s.brandUsers[brandWalletID]
	if !ok {
		return nil, fmt.Errorf("brand wallet not found")
	}

	for i, user := range users {
		if user.ID == userID {
			if name, ok := updates["name"].(string); ok {
				users[i].Name = name
			}
			if email, ok := updates["email"].(string); ok {
				users[i].Email = email
			}
			if kycStatus, ok := updates["kyc_status"].(string); ok {
				users[i].KYCStatus = kycStatus
			}
			if kycLevel, ok := updates["kyc_level"].(int); ok {
				users[i].KYCLevel = kycLevel
			}
			if isActive, ok := updates["is_active"].(bool); ok {
				users[i].IsActive = isActive
			}
			users[i].UpdatedAt = time.Now().Unix()
			return users[i], nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

// RemoveBrandUser removes a user from a brand wallet
func (s *CustomBrandingService) RemoveBrandUser(brandWalletID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	users, ok := s.brandUsers[brandWalletID]
	if !ok {
		return fmt.Errorf("brand wallet not found")
	}

	newUsers := make([]*BrandUser, 0)
	for _, user := range users {
		if user.ID != userID {
			newUsers = append(newUsers, user)
		}
	}
	s.brandUsers[brandWalletID] = newUsers

	return nil
}

// ============================================================================
// Network and Token Management for Brand
// ============================================================================

// AddNetworkToBrand adds a network to a brand wallet
func (s *CustomBrandingService) AddNetworkToBrand(brandWalletID, networkID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.brandWallets[brandWalletID]
	if !ok {
		return fmt.Errorf("brand wallet not found")
	}

	// Check if network exists
	if _, ok := s.masterService.networkRegistry.GetNetwork(networkID); !ok {
		return fmt.Errorf("network not found")
	}

	// Check if already exists
	for _, id := range wallet.NetworkIDs {
		if id == networkID {
			return fmt.Errorf("network already added")
		}
	}

	wallet.NetworkIDs = append(wallet.NetworkIDs, networkID)
	wallet.UpdatedAt = time.Now().Unix()

	return nil
}

// RemoveNetworkFromBrand removes a network from a brand wallet
func (s *CustomBrandingService) RemoveNetworkFromBrand(brandWalletID, networkID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.brandWallets[brandWalletID]
	if !ok {
		return fmt.Errorf("brand wallet not found")
	}

	newNetworkIDs := make([]string, 0)
	for _, id := range wallet.NetworkIDs {
		if id != networkID {
			newNetworkIDs = append(newNetworkIDs, id)
		}
	}
	wallet.NetworkIDs = newNetworkIDs
	wallet.UpdatedAt = time.Now().Unix()

	return nil
}

// AddTokenToBrand adds a token to a brand wallet
func (s *CustomBrandingService) AddTokenToBrand(brandWalletID, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.brandWallets[brandWalletID]
	if !ok {
		return fmt.Errorf("brand wallet not found")
	}

	// Check if token exists
	if _, ok := s.masterService.tokenRegistry.GetToken(tokenID); !ok {
		return fmt.Errorf("token not found")
	}

	// Check if already exists
	for _, id := range wallet.TokenIDs {
		if id == tokenID {
			return fmt.Errorf("token already added")
		}
	}

	wallet.TokenIDs = append(wallet.TokenIDs, tokenID)
	wallet.UpdatedAt = time.Now().Unix()

	return nil
}

// RemoveTokenFromBrand removes a token from a brand wallet
func (s *CustomBrandingService) RemoveTokenFromBrand(brandWalletID, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, ok := s.brandWallets[brandWalletID]
	if !ok {
		return fmt.Errorf("brand wallet not found")
	}

	newTokenIDs := make([]string, 0)
	for _, id := range wallet.TokenIDs {
		if id != tokenID {
			newTokenIDs = append(newTokenIDs, id)
		}
	}
	wallet.TokenIDs = newTokenIDs
	wallet.UpdatedAt = time.Now().Unix()

	return nil
}

// ============================================================================
// JSON Export
// ============================================================================

// GetBrandWalletJSON returns a brand wallet as JSON
func (s *CustomBrandingService) GetBrandWalletJSON(id string) (string, error) {
	wallet, ok := s.GetBrandWallet(id)
	if !ok {
		return "", fmt.Errorf("brand wallet not found")
	}
	data, err := json.Marshal(wallet)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetBrandUsersJSON returns brand users as JSON
func (s *CustomBrandingService) GetBrandUsersJSON(brandWalletID string) (string, error) {
	users, err := s.GetBrandUsers(brandWalletID)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(users)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ============================================================================
// Statistics
// ============================================================================

// GetBrandCount returns the total number of brand wallets
func (s *CustomBrandingService) GetBrandCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.brandWallets)
}

// GetBrandUserCount returns the total number of users across all brands
func (s *CustomBrandingService) GetBrandUserCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, users := range s.brandUsers {
		count += len(users)
	}
	return count
}

// Helper function to import strings package
func init() {
	// This is a placeholder - strings package is used in CreateBrandWallet
}

// strings package workaround - define the function locally
func stringsToUpper(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			result[i] = c - 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
