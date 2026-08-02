package master_wallet

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// Admin Types
// ============================================================================

type AdminLevel string

const (
	AdminLevelSuper    AdminLevel = "super_admin"
	AdminLevelMaster   AdminLevel = "master_admin"
	AdminLevelAdmin    AdminLevel = "admin"
	AdminLevelWhite    AdminLevel = "white_level"
	AdminLevelClient   AdminLevel = "client"
)

type Admin struct {
	ID        string     `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Level     AdminLevel `json:"level"`
	WalletID  string    `json:"walletId"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
}

type AdminPermission struct {
	AdminID       string   `json:"adminId"`
	CanCreatePair bool     `json:"canCreatePair"`
	CanEditPair   bool     `json:"canEditPair"`
	CanDeletePair bool     `json:"canDeletePair"`
	CanManageFutures bool `json:"canManageFutures"`
	CanManageOptions bool `json:"canManageOptions"`
	CanManageCopyTrading bool `json:"canManageCopyTrading"`
	CanManageRedPacket bool `json:"canManageRedPacket"`
	CanManageClaim bool `json:"canManageClaim"`
	CanManageConvert bool `json:"canManageConvert"`
	CanManageAdmins bool `json:"canManageAdmins"`
	CanManageWallets bool `json:"canManageWallets"`
	CanViewAnalytics bool `json:"canViewAnalytics"`
}

type TradingPairConfig struct {
	ID               string  `json:"id"`
	Symbol           string  `json:"symbol"`
	Base             string  `json:"base"`
	Quote            string  `json:"quote"`
	Category         string  `json:"category"` // futures, options, spot
	IsPreInstalled   bool    `json:"isPreInstalled"`
	Status           string  `json:"status"` // active, suspended, halted
	MinOrderSize     float64 `json:"minOrderSize"`
	MaxOrderSize     float64 `json:"maxOrderSize"`
	MakerFee         float64 `json:"makerFee"`
	TakerFee         float64 `json:"takerFee"`
	LeverageEnabled  bool    `json:"leverageEnabled"`
	MaxLeverage      int     `json:"maxLeverage"`
	MarginMode       string  `json:"marginMode"` // cross, isolated
	CreatedBy        string  `json:"createdBy"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type FeatureConfig struct {
	FeatureName       string `json:"featureName"`
	Enabled           bool   `json:"enabled"`
	MinAmount         float64 `json:"minAmount"`
	MaxAmount         float64 `json:"maxAmount"`
	MaxPerUser        float64 `json:"maxPerUser"`
	FeePercentage     float64 `json:"feePercentage"`
	AdminOnly         bool   `json:"adminOnly"`
}

type SystemConfig struct {
	FuturesEnabled    bool    `json:"futuresEnabled"`
	OptionsEnabled    bool    `json:"optionsEnabled"`
	CopyTradingEnabled bool  `json:"copyTradingEnabled"`
	RedPacketEnabled  bool   `json:"redPacketEnabled"`
	ClaimEnabled      bool   `json:"claimEnabled"`
	ConvertEnabled    bool   `json:"convertEnabled"`
	MaxPairs          int    `json:"maxPairs"`
	PreInstalledPairs int    `json:"preInstalledPairs"`
}

// ============================================================================
// Master Admin Service
// ============================================================================

type MasterAdminService struct {
	mu           sync.RWMutex
	admins       map[string]*Admin
	permissions  map[string]*AdminPermission
	pairConfigs  map[string]*TradingPairConfig
	featureConfigs map[string]*FeatureConfig
	systemConfig *SystemConfig
}

func NewMasterAdminService() *MasterAdminService {
	mas := &MasterAdminService{
		admins:        make(map[string]*Admin),
		permissions:   make(map[string]*AdminPermission),
		pairConfigs:   make(map[string]*TradingPairConfig),
		featureConfigs: make(map[string]*FeatureConfig),
	}
	mas.initializeDefaultConfig()
	mas.initializeDefaultAdmins()
	return mas
}

func (mas *MasterAdminService) initializeDefaultConfig() {
	mas.systemConfig = &SystemConfig{
		FuturesEnabled:      true,
		OptionsEnabled:      true,
		CopyTradingEnabled:  true,
		RedPacketEnabled:    true,
		ClaimEnabled:       true,
		ConvertEnabled:     true,
		MaxPairs:           50000,
		PreInstalledPairs:   200,
	}

	// Initialize feature configs
	features := []string{"futures", "options", "copy_trading", "red_packet", "claim", "convert"}
	for _, feature := range features {
		mas.featureConfigs[feature] = &FeatureConfig{
			FeatureName:   feature,
			Enabled:       true,
			MinAmount:      1,
			MaxAmount:      1000000,
			MaxPerUser:     100000,
			FeePercentage:   0.1,
			AdminOnly:      false,
		}
	}
}

func (mas *MasterAdminService) initializeDefaultAdmins() {
	// Create master admin
	masterAdmin := &Admin{
		ID:        "admin-001",
		Username:  "MasterAdmin",
		Email:     "master@tigerwallet.com",
		Level:     AdminLevelMaster,
		WalletID:  "master-wallet-001",
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	mas.admins[masterAdmin.ID] = masterAdmin

	// Full permissions for master admin
	mas.permissions[masterAdmin.ID] = &AdminPermission{
		AdminID:             masterAdmin.ID,
		CanCreatePair:       true,
		CanEditPair:         true,
		CanDeletePair:       true,
		CanManageFutures:    true,
		CanManageOptions:    true,
		CanManageCopyTrading: true,
		CanManageRedPacket:  true,
		CanManageClaim:      true,
		CanManageConvert:    true,
		CanManageAdmins:     true,
		CanManageWallets:    true,
		CanViewAnalytics:    true,
	}

	// Create super admin
	superAdmin := &Admin{
		ID:        "admin-002",
		Username:  "SuperAdmin",
		Email:     "super@tigerwallet.com",
		Level:     AdminLevelSuper,
		WalletID:  "master-wallet-002",
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	mas.admins[superAdmin.ID] = superAdmin
	mas.permissions[superAdmin.ID] = &AdminPermission{
		AdminID:             superAdmin.ID,
		CanCreatePair:       true,
		CanEditPair:         true,
		CanDeletePair:       true,
		CanManageFutures:    true,
		CanManageOptions:    true,
		CanManageCopyTrading: true,
		CanManageRedPacket:  true,
		CanManageClaim:      true,
		CanManageConvert:    true,
		CanManageAdmins:     true,
		CanManageWallets:    true,
		CanViewAnalytics:    true,
	}

	// Create regular admin
	admin := &Admin{
		ID:        "admin-003",
		Username:  "Admin",
		Email:     "admin@tigerwallet.com",
		Level:     AdminLevelAdmin,
		WalletID:  "master-wallet-003",
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	mas.admins[admin.ID] = admin
	mas.permissions[admin.ID] = &AdminPermission{
		AdminID:             admin.ID,
		CanCreatePair:       true,
		CanEditPair:         true,
		CanDeletePair:       false,
		CanManageFutures:    true,
		CanManageOptions:    true,
		CanManageCopyTrading: true,
		CanManageRedPacket:  true,
		CanManageClaim:      true,
		CanManageConvert:    true,
		CanManageAdmins:     false,
		CanManageWallets:    true,
		CanViewAnalytics:    true,
	}

	// Create white level admin
	whiteAdmin := &Admin{
		ID:        "admin-004",
		Username:  "WhiteAdmin",
		Email:     "white@tigerwallet.com",
		Level:     AdminLevelWhite,
		WalletID:  "master-wallet-004",
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	mas.admins[whiteAdmin.ID] = whiteAdmin
	mas.permissions[whiteAdmin.ID] = &AdminPermission{
		AdminID:              whiteAdmin.ID,
		CanCreatePair:        true,
		CanEditPair:          true,
		CanDeletePair:        false,
		CanManageFutures:     true,
		CanManageOptions:     true,
		CanManageCopyTrading: true,
		CanManageRedPacket:   true,
		CanManageClaim:       true,
		CanManageConvert:     true,
		CanManageAdmins:      false,
		CanManageWallets:     false,
		CanViewAnalytics:     true,
	}
}

// ============================================================================
// Admin Management
// ============================================================================

func (mas *MasterAdminService) CreateAdmin(admin *Admin) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	if _, exists := mas.admins[admin.ID]; exists {
		return fmt.Errorf("admin already exists: %s", admin.ID)
	}

	admin.ID = fmt.Sprintf("admin-%d", time.Now().Unix())
	admin.CreatedAt = time.Now()
	mas.admins[admin.ID] = admin

	// Create default permissions
	mas.permissions[admin.ID] = &AdminPermission{
		AdminID: admin.ID,
	}

	return nil
}

func (mas *MasterAdminService) GetAdmin(adminID string) (*Admin, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	admin, ok := mas.admins[adminID]
	if !ok {
		return nil, fmt.Errorf("admin not found: %s", adminID)
	}
	return admin, nil
}

func (mas *MasterAdminService) GetAllAdmins() []*Admin {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	admins := make([]*Admin, 0, len(mas.admins))
	for _, admin := range mas.admins {
		admins = append(admins, admin)
	}
	return admins
}

func (mas *MasterAdminService) UpdateAdminPermissions(adminID string, perms *AdminPermission) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	if _, exists := mas.admins[adminID]; !exists {
		return fmt.Errorf("admin not found: %s", adminID)
	}

	mas.permissions[adminID] = perms
	return nil
}

func (mas *MasterAdminService) GetPermissions(adminID string) (*AdminPermission, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	perms, ok := mas.permissions[adminID]
	if !ok {
		return nil, fmt.Errorf("permissions not found for admin: %s", adminID)
	}
	return perms, nil
}

func (mas *MasterAdminService) CheckPermission(adminID, permission string) (bool, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	admin, exists := mas.admins[adminID]
	if !exists {
		return false, fmt.Errorf("admin not found: %s", adminID)
	}

	// Master and super admins have all permissions
	if admin.Level == AdminLevelMaster || admin.Level == AdminLevelSuper {
		return true, nil
	}

	perms, ok := mas.permissions[adminID]
	if !ok {
		return false, nil
	}

	// Check specific permission
	switch permission {
	case "create_pair":
		return perms.CanCreatePair, nil
	case "edit_pair":
		return perms.CanEditPair, nil
	case "delete_pair":
		return perms.CanDeletePair, nil
	case "manage_futures":
		return perms.CanManageFutures, nil
	case "manage_options":
		return perms.CanManageOptions, nil
	case "manage_copy_trading":
		return perms.CanManageCopyTrading, nil
	case "manage_red_packet":
		return perms.CanManageRedPacket, nil
	case "manage_claim":
		return perms.CanManageClaim, nil
	case "manage_convert":
		return perms.CanManageConvert, nil
	case "manage_admins":
		return perms.CanManageAdmins, nil
	case "manage_wallets":
		return perms.CanManageWallets, nil
	case "view_analytics":
		return perms.CanViewAnalytics, nil
	}

	return false, fmt.Errorf("unknown permission: %s", permission)
}

// ============================================================================
// Trading Pair Management
// ============================================================================

func (mas *MasterAdminService) CreateTradingPair(config *TradingPairConfig) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	config.ID = fmt.Sprintf("pair-%d", time.Now().Unix())
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	mas.pairConfigs[config.Symbol] = config
	return nil
}

func (mas *MasterAdminService) UpdateTradingPair(symbol string, updates map[string]interface{}) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	pair, exists := mas.pairConfigs[symbol]
	if !exists {
		return fmt.Errorf("pair not found: %s", symbol)
	}

	// Apply updates
	if status, ok := updates["status"].(string); ok {
		pair.Status = status
	}
	if preInstalled, ok := updates["isPreInstalled"].(bool); ok {
		pair.IsPreInstalled = preInstalled
	}
	if makerFee, ok := updates["makerFee"].(float64); ok {
		pair.MakerFee = makerFee
	}
	if takerFee, ok := updates["takerFee"].(float64); ok {
		pair.TakerFee = takerFee
	}
	if maxLeverage, ok := updates["maxLeverage"].(int); ok {
		pair.MaxLeverage = maxLeverage
	}

	pair.UpdatedAt = time.Now()
	return nil
}

func (mas *MasterAdminService) DeleteTradingPair(symbol string) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	if _, exists := mas.pairConfigs[symbol]; !exists {
		return fmt.Errorf("pair not found: %s", symbol)
	}

	delete(mas.pairConfigs, symbol)
	return nil
}

func (mas *MasterAdminService) GetTradingPair(symbol string) (*TradingPairConfig, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	pair, ok := mas.pairConfigs[symbol]
	if !ok {
		return nil, fmt.Errorf("pair not found: %s", symbol)
	}
	return pair, nil
}

func (mas *MasterAdminService) GetAllTradingPairs() []*TradingPairConfig {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	pairs := make([]*TradingPairConfig, 0, len(mas.pairConfigs))
	for _, pair := range mas.pairConfigs {
		pairs = append(pairs, pair)
	}
	return pairs
}

// ============================================================================
// Feature Configuration
// ============================================================================

func (mas *MasterAdminService) UpdateFeatureConfig(featureName string, config *FeatureConfig) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	mas.featureConfigs[featureName] = config
	return nil
}

func (mas *MasterAdminService) GetFeatureConfig(featureName string) (*FeatureConfig, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	config, ok := mas.featureConfigs[featureName]
	if !ok {
		return nil, fmt.Errorf("feature config not found: %s", featureName)
	}
	return config, nil
}

func (mas *MasterAdminService) EnableFeature(featureName string) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	if config, ok := mas.featureConfigs[featureName]; ok {
		config.Enabled = true
	}
	return nil
}

func (mas *MasterAdminService) DisableFeature(featureName string) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	if config, ok := mas.featureConfigs[featureName]; ok {
		config.Enabled = false
	}
	return nil
}

// ============================================================================
// System Configuration
// ============================================================================

func (mas *MasterAdminService) GetSystemConfig() *SystemConfig {
	mas.mu.RLock()
	defer mas.mu.RUnlock()
	return mas.systemConfig
}

func (mas *MasterAdminService) UpdateSystemConfig(config *SystemConfig) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()
	mas.systemConfig = config
	return nil
}

// ============================================================================
// JSON Serialization
// ============================================================================

func (mas *MasterAdminService) ToJSON() (string, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	data := struct {
		Admins         map[string]*Admin         `json:"admins"`
		Permissions    map[string]*AdminPermission `json:"permissions"`
		PairConfigs   map[string]*TradingPairConfig `json:"pairConfigs"`
		FeatureConfigs map[string]*FeatureConfig   `json:"featureConfigs"`
		SystemConfig  *SystemConfig              `json:"systemConfig"`
	}{
		Admins:         mas.admins,
		Permissions:    mas.permissions,
		PairConfigs:   mas.pairConfigs,
		FeatureConfigs: mas.featureConfigs,
		SystemConfig:  mas.systemConfig,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
