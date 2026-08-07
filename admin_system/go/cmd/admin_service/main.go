/**
 * TigerWallet Admin System Service
 * Comprehensive admin management system with Super Admin, Sub Admin, KYC, and all management features
 */

package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// / Configuration
type Config struct {
	ServerPort    string
	RedisAddr     string
	JWTSecret     string
	EncryptionKey string
}

// Admin Types
type AdminRole string
type AdminStatus string
type Permission string
type AuditAction string

const (
	RoleSuperAdmin AdminRole = "SUPER_ADMIN"
	RoleAdmin      AdminRole = "ADMIN"
	RoleSupport    AdminRole = "SUPPORT"
	RoleAnalyst    AdminRole = "ANALYST"
	RoleViewer     AdminRole = "VIEWER"

	StatusActive    AdminStatus = "ACTIVE"
	StatusInactive  AdminStatus = "INACTIVE"
	StatusSuspended AdminStatus = "SUSPENDED"

	AuditCreate AuditAction = "CREATE"
	AuditUpdate AuditAction = "UPDATE"
	AuditDelete AuditAction = "DELETE"
	AuditLogin  AuditAction = "LOGIN"
	AuditLogout AuditAction = "LOGIN"
	AuditAccess AuditAction = "ACCESS"
)

// Permission constants
const (
	PermUserMgmt       Permission = "USER_MANAGEMENT"
	PermAdminMgmt      Permission = "ADMIN_MANAGEMENT"
	PermKYCMgmt        Permission = "KYC_MANAGEMENT"
	PermPairsMgmt      Permission = "PAIRS_MANAGEMENT"
	PermLiquidityMgmt  Permission = "LIQUIDITY_MANAGEMENT"
	PermFeesMgmt       Permission = "FEES_MANAGEMENT"
	PermWithdrawMgmt   Permission = "WITHDRAWAL_MANAGEMENT"
	PermAPIMgmt        Permission = "API_MANAGEMENT"
	PermBlockchainMgmt Permission = "BLOCKCHAIN_MANAGEMENT"
	PermWalletMgmt     Permission = "WALLET_MANAGEMENT"
	PermWhiteLabelMgmt Permission = "WHITE_LABEL_MANAGEMENT"
	PermBrokerageMgmt  Permission = "BROKERAGE_MANAGEMENT"
	PermSupportMgmt    Permission = "SUPPORT_MANAGEMENT"
	PermAnalytics      Permission = "ANALYTICS"
	PermAudit          Permission = "AUDIT_LOG"
	PermSettings       Permission = "SETTINGS"
	PermTokenMgmt      Permission = "TOKEN_MANAGEMENT"
	PermNFTMgmt        Permission = "NFT_MANAGEMENT"
	PermExchangeMgmt   Permission = "EXCHANGE_MANAGEMENT"
)

// Admin Model
type Admin struct {
	AdminID          string       `json:"admin_id"`
	Email            string       `json:"email"`
	Username         string       `json:"username"`
	PasswordHash     string       `json:"password_hash"`
	Role             AdminRole    `json:"role"`
	Status           AdminStatus  `json:"status"`
	Permissions      []Permission `json:"permissions"`
	FirstName        string       `json:"first_name"`
	LastName         string       `json:"last_name"`
	Phone            string       `json:"phone"`
	Avatar           string       `json:"avatar"`
	LastLogin        *time.Time   `json:"last_login,omitempty"`
	FailedAttempts   int          `json:"failed_attempts"`
	LockedUntil      *time.Time   `json:"locked_until,omitempty"`
	TwoFactorEnabled bool         `json:"two_factor_enabled"`
	TwoFactorSecret  string       `json:"two_factor_secret,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
	CreatedBy        string       `json:"created_by"`
	IPWhitelist      []string     `json:"ip_whitelist"`
	MFAEnabled       bool         `json:"mfa_enabled"`
}

// KYC Model
type KYCRecord struct {
	KYCID          string        `json:"kyc_id"`
	UserID         string        `json:"user_id"`
	Level          int           `json:"level"`
	Status         string        `json:"status"`
	DocumentType   string        `json:"document_type"`
	DocumentID     string        `json:"document_id"`
	FirstName      string        `json:"first_name"`
	LastName       string        `json:"last_name"`
	DateOfBirth    string        `json:"date_of_birth"`
	Nationality    string        `json:"nationality"`
	Address        string        `json:"address"`
	City           string        `json:"city"`
	Country        string        `json:"country"`
	PostalCode     string        `json:"postal_code"`
	Documents      []KYCDocument `json:"documents"`
	VerifiedAt     *time.Time    `json:"verified_at,omitempty"`
	RejectedAt     *time.Time    `json:"rejected_at,omitempty"`
	RejectedReason string        `json:"rejected_reason"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	ReviewedBy     string        `json:"reviewed_by"`
}

type KYCDocument struct {
	Type       string `json:"type"`
	URL        string `json:"url"`
	Status     string `json:"status"`
	UploadedAt string `json:"uploaded_at"`
}

// User Model
type User struct {
	UserID       string       `json:"user_id"`
	Email        string       `json:"email"`
	Username     string       `json:"username"`
	Phone        string       `json:"phone"`
	Status       string       `json:"status"`
	KYCLevel     int          `json:"kyc_level"`
	Wallets      []UserWallet `json:"wallets"`
	ReferralCode string       `json:"referral_code"`
	ReferredBy   string       `json:"referred_by"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	LastLogin    *time.Time   `json:"last_login,omitempty"`
}

type UserWallet struct {
	Address   string `json:"address"`
	Chain     string `json:"chain"`
	CreatedAt string `json:"created_at"`
}

// Audit Log
type AuditLog struct {
	LogID      string      `json:"log_id"`
	AdminID    string      `json:"admin_id"`
	AdminEmail string      `json:"admin_email"`
	Action     AuditAction `json:"action"`
	Resource   string      `json:"resource"`
	ResourceID string      `json:"resource_id"`
	Details    string      `json:"details"`
	IPAddress  string      `json:"ip_address"`
	UserAgent  string      `json:"user_agent"`
	Timestamp  time.Time   `json:"timestamp"`
}

// Token Model
type Token struct {
	TokenID         string    `json:"token_id"`
	Name            string    `json:"name"`
	Symbol          string    `json:"symbol"`
	Decimals        int       `json:"decimals"`
	ContractAddress string    `json:"contract_address"`
	Chain           string    `json:"chain"`
	Type            string    `json:"type"`
	Status          string    `json:"status"`
	TotalSupply     string    `json:"total_supply"`
	CreatedAt       time.Time `json:"created_at"`
}

// Pair Model
type TradingPair struct {
	PairID         string    `json:"pair_id"`
	BaseToken      string    `json:"base_token"`
	QuoteToken     string    `json:"quote_token"`
	Chain          string    `json:"chain"`
	Status         string    `json:"status"`
	FeeMaker       float64   `json:"fee_maker"`
	FeeTaker       float64   `json:"fee_taker"`
	MinTradeAmount float64   `json:"min_trade_amount"`
	MaxTradeAmount float64   `json:"max_trade_amount"`
	CreatedAt      time.Time `json:"created_at"`
}

// Liquidity Pool
type LiquidityPool struct {
	PoolID    string    `json:"pool_id"`
	TokenA    string    `json:"token_a"`
	TokenB    string    `json:"token_b"`
	Chain     string    `json:"chain"`
	ReserveA  string    `json:"reserve_a"`
	ReserveB  string    `json:"reserve_b"`
	Liquidity string    `json:"liquidity"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Withdrawal
type Withdrawal struct {
	WithdrawalID string     `json:"withdrawal_id"`
	UserID       string     `json:"user_id"`
	Token        string     `json:"token"`
	Amount       string     `json:"amount"`
	Fee          string     `json:"fee"`
	ToAddress    string     `json:"to_address"`
	Status       string     `json:"status"`
	TxHash       string     `json:"tx_hash"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// API Key
type APIKey struct {
	KeyID       string     `json:"key_id"`
	UserID      string     `json:"user_id"`
	Key         string     `json:"key"`
	Secret      string     `json:"secret"`
	Name        string     `json:"name"`
	Permissions []string   `json:"permissions"`
	IPWhitelist []string   `json:"ip_whitelist"`
	RateLimit   int        `json:"rate_limit"`
	Status      string     `json:"status"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// White Label Client
type WhiteLabelClient struct {
	ClientID    string    `json:"client_id"`
	Name        string    `json:"name"`
	Domain      string    `json:"domain"`
	Status      string    `json:"status"`
	Branding    Branding  `json:"branding"`
	Config      WLConfig  `json:"config"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

type Branding struct {
	Logo           string `json:"logo"`
	Favicon        string `json:"favicon"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	Name           string `json:"name"`
}

type WLConfig struct {
	EnabledFeatures []string  `json:"enabled_features"`
	FeeStructure    FeeConfig `json:"fee_structure"`
	CustomDomain    string    `json:"custom_domain"`
}

type FeeConfig struct {
	TradingFee    float64 `json:"trading_fee"`
	WithdrawalFee float64 `json:"withdrawal_fee"`
	DepositFee    float64 `json:"deposit_fee"`
}

// Admin Service
type AdminService struct {
	config     Config
	redis      *redis.Client
	admins     map[string]*Admin
	users      map[string]*User
	kycRecords map[string]*KYCRecord
	auditLogs  []*AuditLog
	mu         sync.RWMutex
}

// NewAdminService creates a new admin service
func NewAdminService(cfg Config) *AdminService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		DB:   2,
	})

	return &AdminService{
		config:     cfg,
		redis:      redisClient,
		admins:     make(map[string]*Admin),
		users:      make(map[string]*User),
		kycRecords: make(map[string]*KYCRecord),
		auditLogs:  make([]*AuditLog, 0),
	}
}

// Initialize creates default super admin
func (s *AdminService) Initialize() {
	// Create default super admin
	superAdmin := &Admin{
		AdminID:     "admin_" + uuid.New().String()[:8],
		Email:       "admin@tigerwallet.com",
		Username:    "superadmin",
		Role:        RoleSuperAdmin,
		Status:      StatusActive,
		FirstName:   "Super",
		LastName:    "Admin",
		Permissions: s.getAllPermissions(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Hash password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("TigerAdmin2026!"), bcrypt.DefaultCost)
	superAdmin.PasswordHash = string(hashedPassword)

	s.admins[superAdmin.AdminID] = superAdmin
	log.Printf("Created default super admin: %s", superAdmin.Email)
}

func (s *AdminService) getAllPermissions() []Permission {
	return []Permission{
		PermUserMgmt, PermAdminMgmt, PermKYCMgmt, PermPairsMgmt,
		PermLiquidityMgmt, PermFeesMgmt, PermWithdrawMgmt, PermAPIMgmt,
		PermBlockchainMgmt, PermWalletMgmt, PermWhiteLabelMgmt,
		PermBrokerageMgmt, PermSupportMgmt, PermAnalytics, PermAudit,
		PermSettings, PermTokenMgmt, PermNFTMgmt, PermExchangeMgmt,
	}
}

// Authenticate admin
func (s *AdminService) Authenticate(email, password, ipAddress string) (*Admin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var admin *Admin
	for _, a := range s.admins {
		if a.Email == email {
			admin = a
			break
		}
	}

	if admin == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if locked
	if admin.LockedUntil != nil && time.Now().Before(*admin.LockedUntil) {
		return nil, fmt.Errorf("account locked until %v", admin.LockedUntil)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		admin.FailedAttempts++
		if admin.FailedAttempts >= 5 {
			lockedUntil := time.Now().Add(30 * time.Minute)
			admin.LockedUntil = &lockedUntil
		}
		return nil, fmt.Errorf("invalid credentials")
	}

	// Reset failed attempts
	admin.FailedAttempts = 0
	admin.LockedUntil = nil
	now := time.Now()
	admin.LastLogin = &now

	return admin, nil
}

// Check permission
func (s *AdminService) HasPermission(admin *Admin, permission Permission) bool {
	if admin.Role == RoleSuperAdmin {
		return true
	}

	for _, p := range admin.Permissions {
		if p == permission {
			return true
		}
	}

	return false
}

// Create admin (only Super Admin)
func (s *AdminService) CreateAdmin(creator *Admin, newAdmin *Admin) error {
	if !s.HasPermission(creator, PermAdminMgmt) {
		return fmt.Errorf("permission denied")
	}

	// Validate email
	if !s.isValidEmail(newAdmin.Email) {
		return fmt.Errorf("invalid email format")
	}

	// Check if email exists
	for _, a := range s.admins {
		if a.Email == newAdmin.Email {
			return fmt.Errorf("email already exists")
		}
	}

	// Generate ID
	newAdmin.AdminID = "admin_" + uuid.New().String()[:8]

	// Hash password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(newAdmin.PasswordHash), bcrypt.DefaultCost)
	newAdmin.PasswordHash = string(hashedPassword)

	newAdmin.CreatedAt = time.Now()
	newAdmin.UpdatedAt = time.Now()
	newAdmin.CreatedBy = creator.AdminID

	s.admins[newAdmin.AdminID] = newAdmin

	// Create audit log
	s.createAuditLog(creator.AdminID, creator.Email, AuditCreate, "ADMIN", newAdmin.AdminID,
		fmt.Sprintf("Created admin: %s", newAdmin.Email))

	return nil
}

// Update admin
func (s *AdminService) UpdateAdmin(updater *Admin, targetID string, updates map[string]interface{}) error {
	if !s.HasPermission(updater, PermAdminMgmt) {
		return fmt.Errorf("permission denied")
	}

	target, ok := s.admins[targetID]
	if !ok {
		return fmt.Errorf("admin not found")
	}

	// Prevent modifying super admin unless by super admin
	if target.Role == RoleSuperAdmin && updater.AdminID != targetID {
		return fmt.Errorf("cannot modify super admin")
	}

	// Apply updates
	if firstName, ok := updates["first_name"].(string); ok {
		target.FirstName = firstName
	}
	if lastName, ok := updates["last_name"].(string); ok {
		target.LastName = lastName
	}
	if phone, ok := updates["phone"].(string); ok {
		target.Phone = phone
	}
	if role, ok := updates["role"].(string); ok {
		if role == string(RoleSuperAdmin) && updater.Role != RoleSuperAdmin {
			return fmt.Errorf("cannot assign super admin role")
		}
		target.Role = AdminRole(role)
	}
	if status, ok := updates["status"].(string); ok {
		target.Status = AdminStatus(status)
	}
	if permissions, ok := updates["permissions"].([]interface{}); ok {
		var perms []Permission
		for _, p := range permissions {
			if pStr, ok := p.(string); ok {
				perms = append(perms, Permission(pStr))
			}
		}
		target.Permissions = perms
	}

	target.UpdatedAt = time.Now()

	s.createAuditLog(updater.AdminID, updater.Email, AuditUpdate, "ADMIN", targetID, "Updated admin")

	return nil
}

// Delete admin
func (s *AdminService) DeleteAdmin(deleter *Admin, targetID string) error {
	if !s.HasPermission(deleter, PermAdminMgmt) {
		return fmt.Errorf("permission denied")
	}

	target, ok := s.admins[targetID]
	if !ok {
		return fmt.Errorf("admin not found")
	}

	if target.Role == RoleSuperAdmin {
		return fmt.Errorf("cannot delete super admin")
	}

	if deleter.AdminID == targetID {
		return fmt.Errorf("cannot delete yourself")
	}

	delete(s.admins, targetID)

	s.createAuditLog(deleter.AdminID, deleter.Email, AuditDelete, "ADMIN", targetID,
		fmt.Sprintf("Deleted admin: %s", target.Email))

	return nil
}

// Get all admins
func (s *AdminService) GetAdmins() []*Admin {
	s.mu.RLock()
	defer s.mu.RUnlock()

	admins := make([]*Admin, 0, len(s.admins))
	for _, admin := range s.admins {
		// Don't return password hash
		adminCopy := *admin
		adminCopy.PasswordHash = ""
		adminCopy.TwoFactorSecret = ""
		admins = append(admins, &adminCopy)
	}

	return admins
}

// KYC Management
func (s *AdminService) CreateKYC(creator *Admin, kyc *KYCRecord) error {
	if !s.HasPermission(creator, PermKYCMgmt) {
		return fmt.Errorf("permission denied")
	}

	kyc.KYCID = "kyc_" + uuid.New().String()[:8]
	kyc.Status = "PENDING"
	kyc.CreatedAt = time.Now()
	kyc.UpdatedAt = time.Now()

	s.kycRecords[kyc.KYCID] = kyc

	s.createAuditLog(creator.AdminID, creator.Email, AuditCreate, "KYC", kyc.KYCID,
		fmt.Sprintf("Created KYC for user: %s", kyc.UserID))

	return nil
}

func (s *AdminService) ReviewKYC(reviewer *Admin, kycID, decision, reason string) error {
	if !s.HasPermission(reviewer, PermKYCMgmt) {
		return fmt.Errorf("permission denied")
	}

	kyc, ok := s.kycRecords[kycID]
	if !ok {
		return fmt.Errorf("KYC record not found")
	}

	kyc.Status = decision
	kyc.ReviewedBy = reviewer.AdminID
	kyc.UpdatedAt = time.Now()

	if decision == "APPROVED" {
		now := time.Now()
		kyc.VerifiedAt = &now
	} else if decision == "REJECTED" {
		now := time.Now()
		kyc.RejectedAt = &now
		kyc.RejectedReason = reason
	}

	s.createAuditLog(reviewer.AdminID, reviewer.Email, AuditUpdate, "KYC", kycID,
		fmt.Sprintf("Reviewed KYC: %s - %s", decision, reason))

	return nil
}

func (s *AdminService) GetKYCRecords(filters map[string]string) []*KYCRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*KYCRecord, 0)
	for _, kyc := range s.kycRecords {
		match := true

		if userID, ok := filters["user_id"]; ok && kyc.UserID != userID {
			match = false
		}
		if status, ok := filters["status"]; ok && kyc.Status != status {
			match = false
		}
		if level, ok := filters["level"]; ok {
			var levelInt int
			fmt.Sscanf(level, "%d", &levelInt)
			if kyc.Level != levelInt {
				match = false
			}
		}

		if match {
			records = append(records, kyc)
		}
	}

	return records
}

// User Management
func (s *AdminService) CreateUser(user *User) error {
	user.UserID = "user_" + uuid.New().String()[:8]
	user.Status = "ACTIVE"
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	s.users[user.UserID] = user

	return nil
}

func (s *AdminService) GetUsers(filters map[string]string) []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*User, 0)
	for _, user := range s.users {
		match := true

		if email, ok := filters["email"]; ok && !strings.Contains(user.Email, email) {
			match = false
		}
		if status, ok := filters["status"]; ok && user.Status != status {
			match = false
		}

		if match {
			users = append(users, user)
		}
	}

	return users
}

func (s *AdminService) UpdateUser(updater *Admin, userID string, updates map[string]interface{}) error {
	if !s.HasPermission(updater, PermUserMgmt) {
		return fmt.Errorf("permission denied")
	}

	user, ok := s.users[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}

	if status, ok := updates["status"].(string); ok {
		user.Status = status
	}
	if kycLevel, ok := updates["kyc_level"].(int); ok {
		user.KYCLevel = kycLevel
	}

	user.UpdatedAt = time.Now()

	s.createAuditLog(updater.AdminID, updater.Email, AuditUpdate, "USER", userID, "Updated user")

	return nil
}

// Token Management
func (s *AdminService) CreateToken(creator *Admin, token *Token) error {
	if !s.HasPermission(creator, PermTokenMgmt) {
		return fmt.Errorf("permission denied")
	}

	token.TokenID = "token_" + uuid.New().String()[:8]
	token.Status = "ACTIVE"
	token.CreatedAt = time.Now()

	s.createAuditLog(creator.AdminID, creator.Email, AuditCreate, "TOKEN", token.TokenID,
		fmt.Sprintf("Created token: %s (%s)", token.Name, token.Symbol))

	return nil
}

// Trading Pair Management
func (s *AdminService) CreatePair(creator *Admin, pair *TradingPair) error {
	if !s.HasPermission(creator, PermPairsMgmt) {
		return fmt.Errorf("permission denied")
	}

	pair.PairID = "pair_" + uuid.New().String()[:8]
	pair.Status = "ACTIVE"
	pair.CreatedAt = time.Now()

	s.createAuditLog(creator.AdminID, creator.Email, AuditCreate, "PAIR", pair.PairID,
		fmt.Sprintf("Created pair: %s/%s", pair.BaseToken, pair.QuoteToken))

	return nil
}

func (s *AdminService) UpdatePair(updater *Admin, pairID string, updates map[string]interface{}) error {
	if !s.HasPermission(updater, PermPairsMgmt) {
		return fmt.Errorf("permission denied")
	}

	s.createAuditLog(updater.AdminID, updater.Email, AuditUpdate, "PAIR", pairID, "Updated pair")

	return nil
}

func (s *AdminService) TogglePair(updater *Admin, pairID, action string) error {
	if !s.HasPermission(updater, PermPairsMgmt) {
		return fmt.Errorf("permission denied")
	}

	s.createAuditLog(updater.AdminID, updater.Email, AuditUpdate, "PAIR", pairID,
		fmt.Sprintf("Toggled pair: %s", action))

	return nil
}

// Liquidity Management
func (s *AdminService) CreateLiquidityPool(creator *Admin, pool *LiquidityPool) error {
	if !s.HasPermission(creator, PermLiquidityMgmt) {
		return fmt.Errorf("permission denied")
	}

	pool.PoolID = "pool_" + uuid.New().String()[:8]
	pool.Status = "ACTIVE"
	pool.CreatedAt = time.Now()

	s.createAuditLog(creator.AdminID, creator.Email, AuditCreate, "LIQUIDITY", pool.PoolID,
		fmt.Sprintf("Created liquidity pool: %s/%s", pool.TokenA, pool.TokenB))

	return nil
}

// Withdrawal Management
func (s *AdminService) ProcessWithdrawal(processor *Admin, withdrawalID, decision, txHash string) error {
	if !s.HasPermission(processor, PermWithdrawMgmt) {
		return fmt.Errorf("permission denied")
	}

	s.createAuditLog(processor.AdminID, processor.Email, AuditUpdate, "WITHDRAWAL", withdrawalID,
		fmt.Sprintf("Processed withdrawal: %s - %s", decision, txHash))

	return nil
}

// API Key Management
func (s *AdminService) CreateAPIKey(creator *Admin, apiKey *APIKey) error {
	if !s.HasPermission(creator, PermAPIMgmt) {
		return fmt.Errorf("permission denied")
	}

	apiKey.KeyID = "key_" + uuid.New().String()[:8]
	apiKey.Key = "tk_" + uuid.New().String()
	apiKey.Secret = generateSecureSecret()
	apiKey.Status = "ACTIVE"
	apiKey.CreatedAt = time.Now()

	s.createAuditLog(creator.AdminID, creator.Email, AuditCreate, "API_KEY", apiKey.KeyID,
		fmt.Sprintf("Created API key: %s", apiKey.Name))

	return nil
}

// White Label Management
func (s *AdminService) CreateWhiteLabel(creator *Admin, wl *WhiteLabelClient) error {
	if !s.HasPermission(creator, PermWhiteLabelMgmt) {
		return fmt.Errorf("permission denied")
	}

	wl.ClientID = "wl_" + uuid.New().String()[:8]
	wl.Status = "PENDING"
	wl.CreatedAt = time.Now()

	s.createAuditLog(creator.AdminID, creator.Email, AuditCreate, "WHITE_LABEL", wl.ClientID,
		fmt.Sprintf("Created white label: %s", wl.Name))

	return nil
}

func (s *AdminService) UpdateWhiteLabel(updater *Admin, clientID string, updates map[string]interface{}) error {
	if !s.HasPermission(updater, PermWhiteLabelMgmt) {
		return fmt.Errorf("permission denied")
	}

	s.createAuditLog(updater.AdminID, updater.Email, AuditUpdate, "WHITE_LABEL", clientID, "Updated white label")

	return nil
}

// Audit Log
func (s *AdminService) createAuditLog(adminID, adminEmail string, action AuditAction, resource, resourceID, details string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	logEntry := &AuditLog{
		LogID:      "audit_" + uuid.New().String()[:12],
		AdminID:    adminID,
		AdminEmail: adminEmail,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		Timestamp:  time.Now(),
	}

	s.auditLogs = append(s.auditLogs, logEntry)

	// Keep only last 10000 logs in memory
	if len(s.auditLogs) > 10000 {
		s.auditLogs = s.auditLogs[len(s.auditLogs)-10000:]
	}
}

func (s *AdminService) GetAuditLogs(filters map[string]string, limit, offset int) []*AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs := make([]*AuditLog, 0)
	for _, log := range s.auditLogs {
		match := true

		if adminID, ok := filters["admin_id"]; ok && log.AdminID != adminID {
			match = false
		}
		if action, ok := filters["action"]; ok && string(log.Action) != action {
			match = false
		}
		if resource, ok := filters["resource"]; ok && log.Resource != resource {
			match = false
		}

		if match {
			logs = append(logs, log)
		}
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start > len(logs) {
		return []*AuditLog{}
	}
	if end > len(logs) {
		end = len(logs)
	}

	return logs[start:end]
}

// Analytics
func (s *AdminService) GetAnalytics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_users":       len(s.users),
		"total_admins":      len(s.admins),
		"total_kyc_records": len(s.kycRecords),
		"total_audit_logs":  len(s.auditLogs),
		"active_users":      len(s.users),
		"pending_kyc":       s.countPendingKYC(),
	}
}

func (s *AdminService) countPendingKYC() int {
	count := 0
	for _, kyc := range s.kycRecords {
		if kyc.Status == "PENDING" {
			count++
		}
	}
	return count
}

// Helper functions
func (s *AdminService) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	return emailRegex.MatchString(strings.ToLower(email))
}

func generateSecureSecret() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Handlers
func (s *AdminService) LoginHandler(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ipAddress := c.ClientIP()
	admin, err := s.Authenticate(req.Email, req.Password, ipAddress)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Generate token
	token := generateSecureToken(admin.AdminID, s.config.JWTSecret)

	s.createAuditLog(admin.AdminID, admin.Email, AuditLogin, "SYSTEM", admin.AdminID, "Admin login")

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"admin": admin,
	})
}

func (s *AdminService) GetAdminsHandler(c *gin.Context) {
	admins := s.GetAdmins()
	c.JSON(http.StatusOK, gin.H{"admins": admins})
}

func (s *AdminService) GetAnalyticsHandler(c *gin.Context) {
	analytics := s.GetAnalytics()
	c.JSON(http.StatusOK, analytics)
}

func (s *AdminService) GetAuditLogsHandler(c *gin.Context) {
	filters := map[string]string{
		"admin_id": c.Query("admin_id"),
		"action":   c.Query("action"),
		"resource": c.Query("resource"),
	}

	limit := 50
	offset := 0
	fmt.Sscanf(c.DefaultQuery("limit", "50"), "%d", &limit)
	fmt.Sscanf(c.DefaultQuery("offset", "0"), "%d", &offset)

	logs := s.GetAuditLogs(filters, limit, offset)
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (s *AdminService) GetKYCRecordsHandler(c *gin.Context) {
	filters := map[string]string{
		"user_id": c.Query("user_id"),
		"status":  c.Query("status"),
		"level":   c.Query("level"),
	}

	records := s.GetKYCRecords(filters)
	c.JSON(http.StatusOK, gin.H{"records": records})
}

func (s *AdminService) GetUsersHandler(c *gin.Context) {
	filters := map[string]string{
		"email":  c.Query("email"),
		"status": c.Query("status"),
	}

	users := s.GetUsers(filters)
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (s *AdminService) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/admin")
	{
		api.POST("/login", s.LoginHandler)

		// Protected routes
		protected := api.Group("")
		protected.Use(s.AuthMiddleware())
		{
			// Admins
			protected.GET("/admins", s.GetAdminsHandler)

			// Users
			protected.GET("/users", s.GetUsersHandler)

			// KYC
			protected.GET("/kyc", s.GetKYCRecordsHandler)

			// Analytics
			protected.GET("/analytics", s.GetAnalyticsHandler)

			// Audit logs
			protected.GET("/audit-logs", s.GetAuditLogsHandler)
		}
	}
}

func generateSecureToken(adminID, secret string) string {
	data := adminID + ":" + time.Now().String()
	h := sha256.Sum256([]byte(data + secret))
	return base64.URLEncoding.EncodeToString(h[:])
}

func main() {
	cfg := Config{
		ServerPort:    getEnv("ADMIN_SERVICE_PORT", "8087"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:     getEnv("JWT_SECRET", "tiger-admin-secret-2026"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "tiger-admin-encryption"),
	}

	service := NewAdminService(cfg)
	service.Initialize()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "admin-service",
			"timestamp": time.Now().Unix(),
		})
	})

	service.SetupRoutes(r)

	addr := ":" + cfg.ServerPort
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("Starting Admin Service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
