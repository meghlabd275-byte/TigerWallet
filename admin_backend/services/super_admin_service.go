/**
 * TigerWallet - Super Admin Authorization Service
 * 
 * COMPLETE SUPER ADMIN SYSTEM:
 * - Only Super Admin can authorize Master Wallet Admin accounts
 * - Master Wallet Admin login requires Super Admin authorization
 * - Master Wallet Admin can change password and set 2FA after login
 * - Super Admin has FULL control over all features and functionalities
 * - White Label Admin has full control in their custom branding
 * 
 * This service MUST be identical across ALL platforms
 */

package services

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"time"
	
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// ENUMS
// ============================================================================

type UserRole string

const (
	RoleSuperAdmin       UserRole = "super_admin"
	RoleMasterAdmin     UserRole = "master_admin"
	RoleWhiteLabelAdmin UserRole = "white_label_admin"
	RoleUser            UserRole = "user"
)

type AdminStatus string

const (
	StatusActive   AdminStatus = "active"
	StatusInactive AdminStatus = "inactive"
	StatusPending  AdminStatus = "pending"
	StatusSuspended AdminStatus = "suspended"
)

type AuthorizationStatus string

const (
	AuthAuthorized   AuthorizationStatus = "authorized"
	AuthPending      AuthorizationStatus = "pending"
	AuthRevoked      AuthorizationStatus = "revoked"
	AuthRejected     AuthorizationStatus = "rejected"
)

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// Super Admin - The highest authority
type SuperAdmin struct {
	ID                string          `json:"id"`
	Email             string          `json:"email"`
	PasswordHash      string          `json:"password_hash"`
	SecretKey         string          `json:"secret_key"` // For emergency access
	TwoFactorEnabled  bool            `json:"two_factor_enabled"`
	TwoFactorSecret   string          `json:"two_factor_secret"`
	Phone             string          `json:"phone"`
	CreatedAt         int64           `json:"created_at"`
	LastLogin         int64           `json:"last_login"`
	IsActive          bool            `json:"is_active"`
	Permissions       []string        `json:"permissions"` // ALL permissions
}

// Master Wallet Admin - Requires Super Admin authorization
type MasterAdmin struct {
	ID                string              `json:"id"`
	Email             string              `json:"email"`
	PasswordHash      string              `json:"password_hash"`
	AuthorizedBy      string              `json:"authorized_by"` // Super Admin ID
	AuthorizationStatus AuthorizationStatus `json:"authorization_status"`
	TwoFactorEnabled  bool                `json:"two_factor_enabled"`
	TwoFactorSecret   string              `json:"two_factor_secret"`
	Phone             string              `json:"phone"`
	CanCreateWhiteLabel bool             `json:"can_create_white_label"`
	CanManageUsers   bool                `json:"can_manage_users"`
	CanManageWallets bool                `json:"can_manage_wallets"`
	CanAccessFinance bool                `json:"can_access_finance"`
	CanModifyFeatures bool               `json:"can_modify_features"`
	CanManageTokens  bool                `json:"can_manage_tokens"`
	CanManageNetworks bool               `json:"can_manage_networks"`
	CanViewAnalytics bool                `json:"can_view_analytics"`
	CanManageAdmins  bool                `json:"can_manage_admins"`
	MaxWhiteLabels  int                 `json:"max_white_labels"`
	WhiteLabelCount int                 `json:"white_label_count"`
	Status          AdminStatus          `json:"status"`
	CreatedAt       int64                `json:"created_at"`
	LastLogin       int64                `json:"last_login"`
	PasswordChangedAt int64             `json:"password_changed_at"`
	FailedAttempts  int                  `json:"failed_attempts"`
	LockedUntil     int64                `json:"locked_until"`
}

// White Label Admin - Has full control in their branding
type WhiteLabelAdmin struct {
	ID                  string          `json:"id"`
	Email               string          `json:"email"`
	PasswordHash        string          `json:"password_hash"`
	MasterAdminID       string          `json:"master_admin_id"` // Who created them
	BrandName           string          `json:"brand_name"`
	BrandLogo           string          `json:"brand_logo"`
	BrandColor          string          `json:"brand_color"`
	CustomDomain        string          `json:"custom_domain"`
	AuthorizationStatus AuthorizationStatus `json:"authorization_status"`
	TwoFactorEnabled    bool            `json:"two_factor_enabled"`
	TwoFactorSecret     string          `json:"two_factor_secret"`
	CanCustomizeUI      bool            `json:"can_customize_ui"`
	CanCustomizeFees    bool            `json:"can_customize_fees"`
	CanManageUsers      bool            `json:"can_manage_users"`
	CanManageWallets   bool            `json:"can_manage_wallets"`
	CanAccessAnalytics bool            `json:"can_access_analytics"`
	CanManageTokens    bool            `json:"can_manage_tokens"`
	FeePercentage      float64          `json:"fee_percentage"`
	Status             AdminStatus      `json:"status"`
	CreatedAt          int64            `json:"created_at"`
	LastLogin          int64            `json:"last_login"`
}

// Authorization Request - For Master Admin approval
type AuthorizationRequest struct {
	ID            string              `json:"id"`
	Email         string              `json:"email"`
	RequestedBy   string              `json:"requested_by"`
	Role          UserRole           `json:"role"`
	Status        AuthorizationStatus `json:"status"`
	RequestedAt   int64               `json:"requested_at"`
	ReviewedBy    string              `json:"reviewed_by"`
	ReviewedAt    int64               `json:"reviewed_at"`
	Notes         string              `json:"notes"`
}

// Audit Log - Track all admin actions
type AuditLog struct {
	ID          string    `json:"id"`
	AdminID     string    `json:"admin_id"`
	AdminRole   UserRole  `json:"admin_role"`
	Action      string    `json:"action"`
	Details     string    `json:"details"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Timestamp   int64     `json:"timestamp"`
}

// Feature Control - Super Admin control over features
type FeatureControl struct {
	FeatureName     string `json:"feature_name"`
	Enabled         bool   `json:"enabled"`
	GlobalEnabled   bool   `json:"global_enabled"`
	MasterAdminID   string `json:"master_admin_id,omitempty"`
	WhiteLabelID    string `json:"white_label_id,omitempty"`
	UpdatedBy       string `json:"updated_by"`
	UpdatedAt       int64  `json:"updated_at"`
}

// ============================================================================
// SUPER ADMIN SERVICE
// ============================================================================

type SuperAdminService struct {
	superAdmins      map[string]*SuperAdmin
	masterAdmins    map[string]*MasterAdmin
	whiteLabelAdmins map[string]*WhiteLabelAdmin
	authRequests    map[string]*AuthorizationRequest
	featureControls map[string]*FeatureControl
	auditLogs      []*AuditLog
}

// Singleton
var superAdminService *SuperAdminService

func GetSuperAdminService() *SuperAdminService {
	if superAdminService == nil {
		superAdminService = &SuperAdminService{
			superAdmins:      make(map[string]*SuperAdmin),
			masterAdmins:    make(map[string]*MasterAdmin),
			whiteLabelAdmins: make(map[string]*WhiteLabelAdmin),
			authRequests:    make(map[string]*AuthorizationRequest),
			featureControls: make(map[string]*FeatureControl),
			auditLogs:      []*AuditLog{},
		}
		// Create default super admin
		superAdminService.CreateDefaultSuperAdmin()
	}
	return superAdminService
}

func (s *SuperAdminService) CreateDefaultSuperAdmin() {
	// Default Super Admin - CHANGE IMMEDIATELY AFTER FIRST LOGIN
	// Email: superadmin@tigerwallet.com
	// Password: SuperAdmin@2024! (MUST CHANGE)
	
	hash, _ := bcrypt.GenerateFromPassword([]byte("SuperAdmin@2024!"), bcrypt.DefaultCost)
	
	superAdmin := &SuperAdmin{
		ID:               "super_admin_001",
		Email:            "superadmin@tigerwallet.com",
		PasswordHash:     string(hash),
		SecretKey:        generateSecretKey(),
		TwoFactorEnabled: false,
		CreatedAt:        time.Now().Unix(),
		IsActive:         true,
		Permissions:      []string{"*"}, // ALL PERMISSIONS
	}
	
	s.superAdmins[superAdmin.ID] = superAdmin
	s.superAdmins[superAdmin.Email] = superAdmin
	
	// Initialize default feature controls
	s.initializeFeatureControls()
}

func (s *SuperAdminService) initializeFeatureControls() {
	features := []string{
		"master_wallet_creation",
		"multi_blockchain",
		"token_management",
		"user_wallet_ownership",
		"hd_wallet",
		"biometric_auth",
		"pin_code_auth",
		"nft_support",
		"defi_integration",
		"staking",
		"bridge_support",
		"mev_protection",
		"swap_trading",
		"hardware_wallet",
		"admin_controls",
		"network_management",
		"gas_optimization",
		"multi_sig",
		"transaction_history",
		"price_alerts",
		"privacy_zk",
		"coinjoin",
		"account_abstraction",
		"session_keys",
		"paymaster",
		"passkeys",
		"tax_integration",
		"analytics",
		"cross_chain_intent",
		"dapp_browser",
	}
	
	for _, feature := range features {
		s.featureControls[feature] = &FeatureControl{
			FeatureName:   feature,
			Enabled:       true,
			GlobalEnabled: true,
			UpdatedAt:     time.Now().Unix(),
		}
	}
}

// ============================================================================
// SUPER ADMIN OPERATIONS
// ============================================================================

// Login as Super Admin
func (s *SuperAdminService) SuperAdminLogin(email, password, twoFactorCode string) (*SuperAdmin, error) {
	superAdmin, ok := s.superAdmins[email]
	if !ok {
		return nil, fmt.Errorf("invalid credentials")
	}
	
	if !superAdmin.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}
	
	if err := bcrypt.CompareHashAndPassword([]byte(superAdmin.PasswordHash), []byte(password)); err != nil {
		s.logAudit(superAdmin.ID, string(RoleSuperAdmin), "LOGIN_FAILED", "Invalid password: "+email, "", "")
		return nil, fmt.Errorf("invalid credentials")
	}
	
	// Check 2FA if enabled
	if superAdmin.TwoFactorEnabled {
		if !s.verifyTwoFactor(superAdmin.TwoFactorSecret, twoFactorCode) {
			return nil, fmt.Errorf("invalid 2FA code")
		}
	}
	
	superAdmin.LastLogin = time.Now().Unix()
	s.logAudit(superAdmin.ID, string(RoleSuperAdmin), "LOGIN_SUCCESS", "Super admin logged in", "", "")
	
	return superAdmin, nil
}

// Authorize Master Admin - ONLY Super Admin can do this
func (s *SuperAdminService) AuthorizeMasterAdmin(superAdminID, masterAdminID string, authorized bool, notes string) error {
	// Verify super admin
	if _, ok := s.superAdmins[superAdminID]; !ok {
		return fmt.Errorf("unauthorized: only super admin can authorize")
	}
	
	masterAdmin, ok := s.masterAdmins[masterAdminID]
	if !ok {
		return fmt.Errorf("master admin not found")
	}
	
	if authorized {
		masterAdmin.AuthorizationStatus = AuthAuthorized
		s.logAudit(superAdminID, string(RoleSuperAdmin), "AUTHORIZE_MASTER_ADMIN", 
			fmt.Sprintf("Authorized master admin: %s", masterAdmin.Email), "", "")
	} else {
		masterAdmin.AuthorizationStatus = AuthRejected
		s.logAudit(superAdminID, string(RoleSuperAdmin), "REJECT_MASTER_ADMIN",
			fmt.Sprintf("Rejected master admin: %s - %s", masterAdmin.Email, notes), "", "")
	}
	
	return nil
}

// Create Master Admin - Requires Super Admin
func (s *SuperAdminService) CreateMasterAdminRequest(email, requestedBy string) (*AuthorizationRequest, error) {
	// Check if already exists
	if _, ok := s.masterAdmins[email]; ok {
		return nil, fmt.Errorf("email already registered")
	}
	
	// Create pending request
	request := &AuthorizationRequest{
		ID:            generateID(),
		Email:         email,
		RequestedBy:   requestedBy,
		Role:          RoleMasterAdmin,
		Status:        AuthPending,
		RequestedAt:   time.Now().Unix(),
	}
	
	s.authRequests[request.ID] = request
	
	// Also create the master admin record (pending)
	hash, _ := bcrypt.GenerateFromPassword([]byte(generateTempPassword()), bcrypt.DefaultCost)
	
	masterAdmin := &MasterAdmin{
		ID:                  generateID(),
		Email:               email,
		PasswordHash:        string(hash),
		AuthorizationStatus:  AuthPending,
		CanCreateWhiteLabel: false,
		CanManageUsers:      false,
		CanManageWallets:    false,
		CanAccessFinance:    false,
		CanModifyFeatures:   false,
		CanManageTokens:     false,
		CanManageNetworks:   false,
		CanViewAnalytics:    false,
		CanManageAdmins:     false,
		MaxWhiteLabels:      0,
		Status:               StatusPending,
		CreatedAt:           time.Now().Unix(),
		FailedAttempts:      0,
	}
	
	s.masterAdmins[masterAdmin.ID] = masterAdmin
	s.masterAdmins[email] = masterAdmin
	
	// Send notification to super admin (in production, send email)
	s.logAudit("SYSTEM", string(RoleSuperAdmin), "MASTER_ADMIN_REQUEST", 
		fmt.Sprintf("New master admin request: %s", email), "", "")
	
	return request, nil
}

// Master Admin Login - Requires authorization
func (s *SuperAdminService) MasterAdminLogin(email, password, twoFactorCode string) (*MasterAdmin, error) {
	masterAdmin, ok := s.masterAdmins[email]
	if !ok {
		return nil, fmt.Errorf("invalid credentials")
	}
	
	// Check if authorized by super admin
	if masterAdmin.AuthorizationStatus != AuthAuthorized {
		return nil, fmt.Errorf("account not authorized by super admin")
	}
	
	if masterAdmin.Status != StatusActive {
		return nil, fmt.Errorf("account is not active")
	}
	
	// Check if locked
	if masterAdmin.LockedUntil > time.Now().Unix() {
		return nil, fmt.Errorf("account is locked until %s", time.Unix(masterAdmin.LockedUntil, 0).Format(time.RFC3339))
	}
	
	if err := bcrypt.CompareHashAndPassword([]byte(masterAdmin.PasswordHash), []byte(password)); err != nil {
		masterAdmin.FailedAttempts++
		if masterAdmin.FailedAttempts >= 5 {
			masterAdmin.LockedUntil = time.Now().Add(15 * time.Minute).Unix()
			masterAdmin.Status = StatusSuspended
		}
		s.logAudit(masterAdmin.ID, string(RoleMasterAdmin), "LOGIN_FAILED", "Invalid password: "+email, "", "")
		return nil, fmt.Errorf("invalid credentials")
	}
	
	// Check 2FA if enabled
	if masterAdmin.TwoFactorEnabled {
		if !s.verifyTwoFactor(masterAdmin.TwoFactorSecret, twoFactorCode) {
			return nil, fmt.Errorf("invalid 2FA code")
		}
	}
	
	masterAdmin.LastLogin = time.Now().Unix()
	masterAdmin.FailedAttempts = 0
	s.logAudit(masterAdmin.ID, string(RoleMasterAdmin), "LOGIN_SUCCESS", "Master admin logged in", "", "")
	
	return masterAdmin, nil
}

// Change Password - Master Admin can change their own password
func (s *SuperAdminService) ChangeMasterAdminPassword(adminID, oldPassword, newPassword string) error {
	masterAdmin, ok := s.masterAdmins[adminID]
	if !ok {
		return fmt.Errorf("admin not found")
	}
	
	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(masterAdmin.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("invalid current password")
	}
	
	// Validate new password
	if len(newPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	
	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password")
	}
	
	masterAdmin.PasswordHash = string(hash)
	masterAdmin.PasswordChangedAt = time.Now().Unix()
	
	s.logAudit(adminID, string(RoleMasterAdmin), "PASSWORD_CHANGED", "Password changed successfully", "", "")
	
	return nil
}

// Enable 2FA - Master Admin can enable 2FA
func (s *SuperAdminService) EnableMasterAdmin2FA(adminID, secret string) error {
	masterAdmin, ok := s.masterAdmins[adminID]
	if !ok {
		return fmt.Errorf("admin not found")
	}
	
	masterAdmin.TwoFactorEnabled = true
	masterAdmin.TwoFactorSecret = secret
	
	s.logAudit(adminID, string(RoleMasterAdmin), "2FA_ENABLED", "Two-factor authentication enabled", "", "")
	
	return nil
}

// Disable 2FA - Master Admin can disable 2FA
func (s *SuperAdminService) DisableMasterAdmin2FA(adminID string) error {
	masterAdmin, ok := s.masterAdmins[adminID]
	if !ok {
		return fmt.Errorf("admin not found")
	}
	
	masterAdmin.TwoFactorEnabled = false
	masterAdmin.TwoFactorSecret = ""
	
	s.logAudit(adminID, string(RoleMasterAdmin), "2FA_DISABLED", "Two-factor authentication disabled", "", "")
	
	return nil
}

// ============================================================================
// WHITE LABEL ADMIN OPERATIONS
// ============================================================================

// Create White Label Admin - By Master Admin
func (s *SuperAdminService) CreateWhiteLabelAdmin(masterAdminID, email, brandName string) (*WhiteLabelAdmin, error) {
	masterAdmin, ok := s.masterAdmins[masterAdminID]
	if !ok {
		return nil, fmt.Errorf("master admin not found")
	}
	
	if !masterAdmin.CanCreateWhiteLabel {
		return nil, fmt.Errorf("not authorized to create white label admins")
	}
	
	if masterAdmin.WhiteLabelCount >= masterAdmin.MaxWhiteLabels {
		return nil, fmt.Errorf("maximum white label limit reached")
	}
	
	// Check if already exists
	if _, ok := s.whiteLabelAdmins[email]; ok {
		return nil, fmt.Errorf("email already registered")
	}
	
	hash, _ := bcrypt.GenerateFromPassword([]byte(generateTempPassword()), bcrypt.DefaultCost)
	
	whiteLabel := &WhiteLabelAdmin{
		ID:                generateID(),
		Email:             email,
		PasswordHash:      string(hash),
		MasterAdminID:     masterAdminID,
		BrandName:         brandName,
		AuthorizationStatus: AuthAuthorized, // Auto-authorized by master admin
		CanCustomizeUI:    true,
		CanCustomizeFees:  true,
		CanManageUsers:    true,
		CanManageWallets:  true,
		CanAccessAnalytics: true,
		CanManageTokens:   true,
		FeePercentage:     0.0,
		Status:            StatusActive,
		CreatedAt:        time.Now().Unix(),
	}
	
	s.whiteLabelAdmins[whiteLabel.ID] = whiteLabel
	s.whiteLabelAdmins[email] = whiteLabel
	
	masterAdmin.WhiteLabelCount++
	
	s.logAudit(masterAdminID, string(RoleMasterAdmin), "WHITE_LABEL_CREATED", 
		fmt.Sprintf("Created white label: %s - %s", email, brandName), "", "")
	
	return whiteLabel, nil
}

// White Label Admin Login
func (s *SuperAdminService) WhiteLabelLogin(email, password, twoFactorCode string) (*WhiteLabelAdmin, error) {
	whiteLabel, ok := s.whiteLabelAdmins[email]
	if !ok {
		return nil, fmt.Errorf("invalid credentials")
	}
	
	if whiteLabel.AuthorizationStatus != AuthAuthorized {
		return nil, fmt.Errorf("account not authorized")
	}
	
	if whiteLabel.Status != StatusActive {
		return nil, fmt.Errorf("account is not active")
	}
	
	if err := bcrypt.CompareHashAndPassword([]byte(whiteLabel.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	
	if whiteLabel.TwoFactorEnabled {
		if !s.verifyTwoFactor(whiteLabel.TwoFactorSecret, twoFactorCode) {
			return nil, fmt.Errorf("invalid 2FA code")
		}
	}
	
	whiteLabel.LastLogin = time.Now().Unix()
	
	return whiteLabel, nil
}

// ============================================================================
// FEATURE CONTROL - Super Admin has FULL control
// ============================================================================

// Enable/Disable feature globally
func (s *SuperAdminService) SetGlobalFeature(superAdminID, featureName string, enabled bool) error {
	// Verify super admin
	if _, ok := s.superAdmins[superAdminID]; !ok {
		return fmt.Errorf("unauthorized: only super admin can modify features")
	}
	
	feature, ok := s.featureControls[featureName]
	if !ok {
		return fmt.Errorf("feature not found")
	}
	
	feature.GlobalEnabled = enabled
	feature.Enabled = enabled
	feature.UpdatedBy = superAdminID
	feature.UpdatedAt = time.Now().Unix()
	
	s.logAudit(superAdminID, string(RoleSuperAdmin), "FEATURE_TOGGLE", 
		fmt.Sprintf("Set global feature %s = %v", featureName, enabled), "", "")
	
	return nil
}

// Enable/Disable feature for specific Master Admin
func (s *SuperAdminService) SetMasterAdminFeature(superAdminID, masterAdminID, featureName string, enabled bool) error {
	if _, ok := s.superAdmins[superAdminID]; !ok {
		return fmt.Errorf("unauthorized: only super admin can modify features")
	}
	
	feature, ok := s.featureControls[featureName]
	if !ok {
		return fmt.Errorf("feature not found")
	}
	
	feature.MasterAdminID = masterAdminID
	feature.Enabled = enabled
	feature.UpdatedBy = superAdminID
	feature.UpdatedAt = time.Now().Unix()
	
	s.logAudit(superAdminID, string(RoleSuperAdmin), "MASTER_ADMIN_FEATURE",
		fmt.Sprintf("Set feature %s for master admin %s = %v", featureName, masterAdminID, enabled), "", "")
	
	return nil
}

// Get all feature controls
func (s *SuperAdminService) GetAllFeatures() []*FeatureControl {
	features := make([]*FeatureControl, 0, len(s.featureControls))
	for _, f := range s.featureControls {
		features = append(features, f)
	}
	return features
}

// Check if feature is enabled for admin
func (s *SuperAdminService) IsFeatureEnabled(featureName, adminID, adminRole string) bool {
	feature, ok := s.featureControls[featureName]
	if !ok {
		return false
	}
	
	// Global check
	if !feature.GlobalEnabled {
		return false
	}
	
	// Role-based check
	switch adminRole {
	case string(RoleSuperAdmin):
		return true // Super admin has access to everything
	case string(RoleMasterAdmin):
		// Check if overridden for this master admin
		if feature.MasterAdminID != "" && feature.MasterAdminID != adminID {
			return false
		}
		return feature.Enabled
	case string(RoleWhiteLabelAdmin):
		if feature.WhiteLabelID != "" && feature.WhiteLabelID != adminID {
			return false
		}
		return feature.Enabled
	}
	
	return false
}

// ============================================================================
// AUDIT LOGGING
// ============================================================================

func (s *SuperAdminService) logAudit(adminID, role, action, details, ipAddress, userAgent string) {
	log := &AuditLog{
		ID:        generateID(),
		AdminID:   adminID,
		AdminRole: UserRole(role),
		Action:    action,
		Details:   details,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Timestamp: time.Now().Unix(),
	}
	
	s.auditLogs = append(s.auditLogs, log)
	
	// In production, save to database
	fmt.Printf("[AUDIT] %s | %s | %s | %s\n", time.Now().Format(time.RFC3339), role, action, details)
}

// Get audit logs
func (s *SuperAdminService) GetAuditLogs(adminID string, limit int) []*AuditLog {
	logs := make([]*AuditLog, 0)
	count := 0
	
	for i := len(s.auditLogs) - 1; i >= 0 && count < limit; i-- {
		if adminID == "" || s.auditLogs[i].AdminID == adminID {
			logs = append(logs, s.auditLogs[i])
			count++
		}
	}
	
	return logs
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func generateID() string {
	return fmt.Sprintf("id_%d_%s", time.Now().Unix(), randomString(8))
}

func generateSecretKey() string {
	return randomString(32)
}

func generateTempPassword() string {
	return randomString(16)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

func (s *SuperAdminService) verifyTwoFactor(secret, code string) bool {
	// In production, use proper TOTP verification
	// This is a simplified version
	return len(code) == 6
}

// Send Email Notification (simplified)
func sendEmail(to, subject, body string) error {
	// In production, configure SMTP
	from := os.Getenv("SMTP_FROM")
	password := os.Getenv("SMTP_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	
	if from == "" || password == "" || smtpHost == "" {
		fmt.Printf("[EMAIL] To: %s, Subject: %s\n", to, subject)
		return nil
	}
	
	auth := smtp.PlainAuth("", from, password, smtpHost)
	
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n", from, to, subject, body)
	
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, []byte(msg))
	
	if err != nil {
		fmt.Printf("Email error: %v\n", err)
	}
	
	return err
}

// Hash data
func hashData(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// Export Admin Data
func (s *SuperAdminService) ExportData() string {
	data := map[string]interface{}{
		"super_admins":     len(s.superAdmins),
		"master_admins":    len(s.masterAdmins),
		"white_labels":     len(s.whiteLabelAdmins),
		"features":        len(s.featureControls),
		"audit_logs":      len(s.auditLogs),
		"exported_at":     time.Now().Unix(),
	}
	
	json, _ := json.MarshalIndent(data, "", "  ")
	return string(json)
}
