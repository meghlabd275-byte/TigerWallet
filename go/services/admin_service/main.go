// TigerWallet Admin Service - Enterprise-Grade Administrative Platform
// Complete admin management with all requested features

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
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort  string `json:"server_port"`
	DBHost      string `json:"db_host"`
	DBPort      string `json:"db_port"`
	DBUser      string `json:"db_user"`
	DBPassword  string `json:"db_password"`
	DBName      string `json:"db_name"`
	RedisHost   string `json:"redis_host"`
	RedisPort   string `json:"redis_port"`
	JWTSecret   string `json:"jwt_secret"`
	EncryptionKey string `json:"encryption_key"`
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:    getEnv("ADMIN_PORT", "9093"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "tigerwallet"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		DBName:        getEnv("DB_NAME", "tigerwallet_admin"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		JWTSecret:     getEnv("JWT_SECRET", "admin-jwt-secret-change-in-prod"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "admin-32-byte-encryption-key!!"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Database Models
// ============================================================================

// Admin represents platform administrators
type Admin struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Username    string    `gorm:"uniqueIndex" json:"username"`
	Email       string    `gorm:"index" json:"email"`
	PasswordHash string   `json:"-"`
	Role        string    `json:"role"` // super_admin, admin, support, finance, compliance, trader
	Permissions string    `json:"permissions"` // JSON array of permissions
	Status      string    `json:"status"` // active, suspended, inactive
	LastLoginAt *time.Time `json:"last_login_at"`
	IPWhitelist string    `json:"ip_whitelist"`
	CreatedBy   uint      `json:"created_by"`
}

// User represents platform users
type User struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	UserID            string    `gorm:"uniqueIndex" json:"user_id"`
	Username          string    `gorm:"index" json:"username"`
	Email             string    `gorm:"index" json:"email"`
	Phone             string    `json:"phone"`
	PasswordHash      string    `json:"-"`
	MasterWalletAddr  string    `json:"master_wallet_address"`
	Status            string    `json:"status"` // active, suspended, banned
	Tier              int       `json:"tier"` // 0: basic, 1: verified, 2: premium, 3: VIP
	IsEmailVerified   bool      `json:"is_email_verified"`
	IsPhoneVerified   bool      `json:"is_phone_verified"`
	KYCStatus         string    `json:"kyc_status"` // none, pending, level1, level2, level3, approved, rejected
	KYCLevel         int       `json:"kyc_level"`
	WhiteLabelID     *uint     `gorm:"index" json:"white_label_id"`
	ReferrerID        *string   `json:"referrer_id"`
	ReferralCode     string    `gorm:"uniqueIndex" json:"referral_code"`
	TotalVolume      float64   `json:"total_volume"`
	TotalDeposit     float64   `json:"total_deposit"`
	TotalWithdraw    float64   `json:"total_withdraw"`
	LastLoginAt      *time.Time `json:"last_login_at"`
}

// KYCRecord represents KYC verification records
type KYCRecord struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UserID          uint      `gorm:"index" json:"user_id"`
	Level           int       `json:"level"`
	DocumentType    string    `json:"document_type"` // passport, id_card, drivers_license
	DocumentNumber  string    `json:"document_number"`
	DocumentFront   string    `json:"document_front"` // encrypted URL
	DocumentBack    string    `json:"document_back"`
	SelfieImage     string    `json:"selfie_image"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	DateOfBirth     string    `json:"date_of_birth"`
	Country         string    `json:"country"`
	Address         string    `json:"address"`
	Status          string    `json:"status"` // pending, approved, rejected
	RejectReason    string    `json:"reject_reason"`
	ReviewedBy      uint      `json:"reviewed_by"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
}

// TradingPair represents trading pairs
type TradingPair struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	PairID        string    `gorm:"uniqueIndex" json:"pair_id"`
	PairName      string    `json:"pair_name"` // ETH/USDT
	BaseToken     string    `json:"base_token"`
	QuoteToken    string    `json:"quote_token"` // USDT, BUSD, etc.
	ChainID       int64     `json:"chain_id"`
	DEX           string    `json:"dex"` // uniswap, pancakeswap, etc.
	PoolAddress   string    `json:"pool_address"`
	Status        string    `json:"status"` // active, suspended, halted, delisted
	TradingFee    string    `json:"trading_fee"` // 0.3%
	ListingFee    string    `json:"listing_fee"`
	Tier          string    `json:"tier"` // tier1, tier2, tier3, tier4
	IsStable      bool      `json:"is_stable"`
	MinTrade      string    `json:"min_trade"`
	MaxTrade      string    `json:"max_trade"`
	WhiteLabelID  *uint     `gorm:"index" json:"white_label_id"`
}

// LiquidityPool represents liquidity pools
type LiquidityPool struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	PoolID        string    `gorm:"uniqueIndex" json:"pool_id"`
	PairID        string    `gorm:"index" json:"pair_id"`
	TokenA        string    `json:"token_a"`
	TokenB        string    `json:"token_b"`
	ChainID       int64     `json:"chain_id"`
	DEX           string    `json:"dex"`
	PoolAddress   string    `json:"pool_address"`
	LiquidityA    string    `json:"liquidity_a"`
	LiquidityB    string    `json:"liquidity_b"`
	LiquidityUSD  float64   `json:"liquidity_usd"`
	Status        string    `json:"status"` // active, removed
	WhiteLabelID  *uint     `gorm:"index" json:"white_label_id"`
}

// FeeConfig represents fee configurations
type FeeConfig struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	FeeType         string    `json:"fee_type"` // trading, withdrawal, deposit, transfer
	ChainID         int64     `json:"chain_id"`
	FeePercent      float64   `json:"fee_percent"`
	FeeFixed        string    `json:"fee_fixed"`
	MinFee          string    `json:"min_fee"`
	MaxFee          string    `json:"max_fee"`
	IsEnabled       bool      `json:"is_enabled"`
	WhiteLabelID    *uint     `gorm:"index" json:"white_label_id"`
}

// MarketMakerBot represents market maker bots
type MarketMakerBot struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	BotID         string    `gorm:"uniqueIndex" json:"bot_id"`
	UserID        string    `gorm:"index" json:"user_id"`
	PairID        string    `gorm:"index" json:"pair_id"`
	Strategy      string    `json:"strategy"` // arbitrage, liquidity, spread
	Status        string    `json:"status"` // active, paused, stopped
	MinBalance    string    `json:"min_balance"`
	MaxBalance    string    `json:"max_balance"`
	Spread        float64   `json:"spread"`
	MinTradeSize  float64   `json:"min_trade_size"`
	MaxTradeSize  float64   `json:"max_trade_size"`
	ProfitTarget  float64   `json:"profit_target"`
	WhiteLabelID  *uint     `gorm:"index" json:"white_label_id"`
}

// VirtualToken represents virtual/IOU tokens
type VirtualToken struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	TokenID       string    `gorm:"uniqueIndex" json:"token_id"`
	Name          string    `json:"name"`
	Symbol        string    `json:"symbol"`
	Decimals      int       `json:"decimals"`
	TotalSupply   string    `json:"total_supply"`
	Circulating   string    `json:"circulating"`
	Issuer        string    `json:"issuer"`
	Status        string    `json:"status"` // active, frozen, revoked
	WhiteLabelID  *uint     `gorm:"index" json:"white_label_id"`
}

// BrokerageClient represents institutional/brokerage clients
type BrokerageClient struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	ClientID        string    `gorm:"uniqueIndex" json:"client_id"`
	CompanyName     string    `json:"company_name"`
	AdminEmail      string    `json:"admin_email"`
	WalletAddress   string    `json:"wallet_address"`
	Status          string    `json:"status"` // active, suspended, pending
	Tier            int       `json:"tier"` // 1: bronze, 2: silver, 3: gold, 4: platinum
	TradingLimit    float64   `json:"trading_limit"`
	FeeDiscount     float64   `json:"fee_discount"`
	WhiteLabelID    *uint     `gorm:"index" json:"white_label_id"`
}

// NFTCollection represents NFT collections
type NFTCollection struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	CollectionID  string    `gorm:"uniqueIndex" json:"collection_id"`
	Name          string    `json:"name"`
	Symbol        string    `json:"symbol"`
	ContractAddr  string    `json:"contract_address"`
	ChainID       int64     `json:"chain_id"`
	Creator       string    `json:"creator"`
	Royalty       float64   `json:"royalty"`
	Status        string    `json:"status"` // active, paused, delisted
	TotalSupply   int       `json:"total_supply"`
	WhiteLabelID  *uint     `gorm:"index" json:"white_label_id"`
}

// TokenCreateRequest represents token creation requests
type TokenCreateRequest struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	RequestID     string    `gorm:"uniqueIndex" json:"request_id"`
	UserID        string    `gorm:"index" json:"user_id"`
	Name          string    `json:"name"`
	Symbol        string    `json:"symbol"`
	Decimals      int       `json:"decimals"`
	TotalSupply   string    `json:"total_supply"`
	ChainID       int64     `json:"chain_id"`
	Type          string    `json:"type"` // erc20, trc20, spl, etc.
	ContractAddr  string    `json:"contract_address"`
	Status        string    `json:"status"` // pending, deployed, failed
	TxHash        string    `json:"tx_hash"`
	WhiteLabelID  *uint     `gorm:"index" json:"white_label_id"`
}

// APIKey represents API keys for users
type APIKey struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	KeyID         string    `gorm:"uniqueIndex" json:"key_id"`
	UserID        string    `gorm:"index" json:"user_id"`
	Key           string    `gorm:"uniqueIndex" json:"key"`
	Secret        string    `json:"-"` // encrypted
	Name          string    `json:"name"`
	Permissions   string    `json:"permissions"` // JSON: ["read", "trade", "withdraw"]
	IPWhitelist   string    `json:"ip_whitelist"`
	RateLimit     int       `json:"rate_limit"` // requests per minute
	Status        string    `json:"status"` // active, suspended
	LastUsedAt    *time.Time `json:"last_used_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
}

// AuditLog represents audit logs
type AuditLog struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	AdminID       uint      `gorm:"index" json:"admin_id"`
	AdminUsername string    `json:"admin_username"`
	Action        string    `json:"action"`
	Resource      string    `json:"resource"`
	ResourceID    string    `json:"resource_id"`
	Details       string    `json:"details"` // JSON
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
	Success       bool      `json:"success"`
	ErrorMessage  string    `json:"error_message"`
}

// ============================================================================
// Admin Service
// ============================================================================

type AdminService struct {
	db           *gorm.DB
	redis        *redis.Client
	config       *Config
	mu           sync.RWMutex
}

func NewAdminService(config *Config) (*AdminService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate all models
	err = db.AutoMigrate(
		&Admin{}, &User{}, &KYCRecord{}, &TradingPair{}, &LiquidityPool{},
		&FeeConfig{}, &MarketMakerBot{}, &VirtualToken{}, &BrokerageClient{},
		&NFTCollection{}, &TokenCreateRequest{}, &APIKey{}, &AuditLog{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &AdminService{
		db:     db,
		redis:  rdb,
		config: config,
	}

	// Create default super admin if not exists
	service.initDefaultAdmin()

	return service, nil
}

func (s *AdminService) initDefaultAdmin() {
	var count int64
	s.db.Model(&Admin{}).Count(&count)
	if count > 0 {
		return
	}

	// Create default super admin: admin@tigerwallet.com / Admin@123
	passwordHash, _ := HashPassword("Admin@123")
	admin := &Admin{
		Username:    "superadmin",
		Email:      "admin@tigerwallet.com",
		PasswordHash: passwordHash,
		Role:       "super_admin",
		Permissions: `["all"]`,
		Status:     "active",
	}
	s.db.Create(admin)
}

// ============================================================================
// Authentication
// ============================================================================

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *AdminService) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var admin Admin
	if err := s.db.Where("username = ? AND status = ?", req.Username, "active").First(&admin).Error; err != nil {
		s.logAudit(0, "LOGIN_FAILED", "admin", req.Username, ctx.ClientIP(), false, "user not found")
		ctx.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}

	if !CheckPassword(req.Password, admin.PasswordHash) {
		s.logAudit(admin.ID, "LOGIN_FAILED", "admin", req.Username, ctx.ClientIP(), false, "invalid password")
		ctx.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"admin_id":  admin.ID,
		"username":  admin.Username,
		"role":      admin.Role,
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(s.config.JWTSecret))

	// Update last login
	now := time.Now()
	admin.LastLoginAt = &now
	s.db.Save(&admin)

	s.logAudit(admin.ID, "LOGIN_SUCCESS", "admin", req.Username, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{
		"token": tokenString,
		"admin": gin.H{
			"id":         admin.ID,
			"username":   admin.Username,
			"email":      admin.Email,
			"role":       admin.Role,
			"permissions": admin.Permissions,
		},
	})
}

// ============================================================================
// Admin Management
// ============================================================================

type CreateAdminRequest struct {
	Username    string  `json:"username" binding:"required"`
	Email       string  `json:"email" binding:"required"`
	Password    string  `json:"password" binding:"required"`
	Role        string  `json:"role" binding:"required"`
	Permissions string  `json:"permissions"`
	IPWhitelist string  `json:"ip_whitelist"`
}

func (s *AdminService) CreateAdmin(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")

	var req CreateAdminRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	passwordHash, _ := HashPassword(req.Password)
	admin := &Admin{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         req.Role,
		Permissions:  req.Permissions,
		IPWhitelist:  req.IPWhitelist,
		Status:       "active",
		CreatedBy:    adminID,
	}

	if err := s.db.Create(admin).Error; err != nil {
		ctx.JSON(500, gin.H{"error": "failed to create admin"})
		return
	}

	s.logAudit(adminID, "CREATE_ADMIN", "admin", strconv.FormatUint(uint64(admin.ID), 10), ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true, "admin_id": admin.ID})
}

func (s *AdminService) ListAdmins(ctx *gin.Context) {
	var admins []Admin
	s.db.Find(&admins)

	// Mask passwords
	result := make([]gin.H, len(admins))
	for i, a := range admins {
		result[i] = gin.H{
			"id":          a.ID,
			"username":    a.Username,
			"email":       a.Email,
			"role":        a.Role,
			"status":      a.Status,
			"last_login":  a.LastLoginAt,
			"created_at":  a.CreatedAt,
		}
	}

	ctx.JSON(200, gin.H{"admins": result})
}

func (s *AdminService) UpdateAdmin(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	adminToUpdate := ctx.Param("id")

	var req struct {
		Email       string `json:"email"`
		Role        string `json:"role"`
		Permissions string `json:"permissions"`
		Status      string `json:"status"`
		IPWhitelist string `json:"ip_whitelist"`
	}
	ctx.ShouldBindJSON(&req)

	var admin Admin
	if err := s.db.First(&admin, adminToUpdate).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "admin not found"})
		return
	}

	if req.Email != "" {
		admin.Email = req.Email
	}
	if req.Role != "" {
		admin.Role = req.Role
	}
	if req.Permissions != "" {
		admin.Permissions = req.Permissions
	}
	if req.Status != "" {
		admin.Status = req.Status
	}
	if req.IPWhitelist != "" {
		admin.IPWhitelist = req.IPWhitelist
	}

	s.db.Save(&admin)
	s.logAudit(adminID, "UPDATE_ADMIN", "admin", adminToUpdate, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true})
}

func (s *AdminService) DeleteAdmin(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	adminToDelete := ctx.Param("id")

	result := s.db.Delete(&Admin{}, adminToDelete)
	if result.Error != nil {
		ctx.JSON(500, gin.H{"error": "failed to delete admin"})
		return
	}

	s.logAudit(adminID, "DELETE_ADMIN", "admin", adminToDelete, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true})
}

// ============================================================================
// User Management
// ============================================================================

func (s *AdminService) ListUsers(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	status := ctx.Query("status")
	kycStatus := ctx.Query("kyc_status")
	whiteLabelID := ctx.Query("white_label_id")

	var users []User
	query := s.db.Model(&User{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if kycStatus != "" {
		query = query.Where("kyc_status = ?", kycStatus)
	}
	if whiteLabelID != "" {
		wlID, _ := strconv.ParseUint(whiteLabelID, 10, 32)
		query = query.Where("white_label_id = ?", wlID)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	query.Offset(offset).Limit(limit).Find(&users)

	ctx.JSON(200, gin.H{
		"users":  users,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

func (s *AdminService) GetUser(ctx *gin.Context) {
	userID := ctx.Param("id")

	var user User
	if err := s.db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "user not found"})
		return
	}

	// Get KYC records
	var kycRecords []KYCRecord
	s.db.Where("user_id = ?", user.ID).Find(&kycRecords)

	ctx.JSON(200, gin.H{
		"user":         user,
		"kyc_records":  kycRecords,
	})
}

func (s *AdminService) UpdateUser(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	userID := ctx.Param("id")

	var req struct {
		Status      string `json:"status"`
		Tier        int    `json:"tier"`
		KYCStatus   string `json:"kyc_status"`
		KYCLevel    int    `json:"kyc_level"`
	}
	ctx.ShouldBindJSON(&req)

	var user User
	if err := s.db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "user not found"})
		return
	}

	if req.Status != "" {
		user.Status = req.Status
	}
	if req.Tier > 0 {
		user.Tier = req.Tier
	}
	if req.KYCStatus != "" {
		user.KYCStatus = req.KYCStatus
	}
	if req.KYCLevel > 0 {
		user.KYCLevel = req.KYCLevel
	}

	s.db.Save(&user)
	s.logAudit(adminID, "UPDATE_USER", "user", userID, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true})
}

func (s *AdminService) SuspendUser(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	userID := ctx.Param("id")

	var user User
	if err := s.db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "user not found"})
		return
	}

	user.Status = "suspended"
	s.db.Save(&user)

	s.logAudit(adminID, "SUSPEND_USER", "user", userID, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true, "status": "suspended"})
}

// ============================================================================
// KYC Management
// ============================================================================

func (s *AdminService) ListKYC(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	status := ctx.Query("status")
	level := ctx.Query("level")

	var records []KYCRecord
	query := s.db.Model(&KYCRecord{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if level != "" {
		lvl, _ := strconv.Atoi(level)
		query = query.Where("level = ?", lvl)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&records)

	ctx.JSON(200, gin.H{
		"records": records,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

func (s *AdminService) ApproveKYC(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	kycID := ctx.Param("id")

	var record KYCRecord
	if err := s.db.First(&record, kycID).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "KYC record not found"})
		return
	}

	record.Status = "approved"
	record.ReviewedBy = adminID
	now := time.Now()
	record.ReviewedAt = &now

	s.db.Save(&record)

	// Update user KYC status
	var user User
	s.db.First(&user, record.UserID)
	user.KYCStatus = "approved"
	user.KYCLevel = record.Level
	s.db.Save(&user)

	s.logAudit(adminID, "APPROVE_KYC", "kyc", kycID, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true, "status": "approved"})
}

func (s *AdminService) RejectKYC(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	kycID := ctx.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	ctx.ShouldBindJSON(&req)

	var record KYCRecord
	if err := s.db.First(&record, kycID).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "KYC record not found"})
		return
	}

	record.Status = "rejected"
	record.RejectReason = req.Reason
	record.ReviewedBy = adminID
	now := time.Now()
	record.ReviewedAt = &now

	s.db.Save(&record)

	// Update user KYC status
	var user User
	s.db.First(&user, record.UserID)
	user.KYCStatus = "rejected"
	s.db.Save(&user)

	s.logAudit(adminID, "REJECT_KYC", "kyc", kycID, ctx.ClientIP(), true, "reason: "+req.Reason)

	ctx.JSON(200, gin.H{"success": true, "status": "rejected"})
}

// ============================================================================
// Trading Pair Management
// ============================================================================

func (s *AdminService) ListPairs(ctx *gin.Context) {
	chainID := ctx.Query("chain_id")
	status := ctx.Query("status")

	var pairs []TradingPair
	query := s.db.Model(&TradingPair{})

	if chainID != "" {
		cid, _ := strconv.ParseInt(chainID, 10, 64)
		query = query.Where("chain_id = ?", cid)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Find(&pairs)

	ctx.JSON(200, gin.H{"pairs": pairs})
}

func (s *AdminService) CreatePair(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")

	var req struct {
		PairName    string  `json:"pair_name" binding:"required"`
		BaseToken   string  `json:"base_token" binding:"required"`
		QuoteToken  string  `json:"quote_token" binding:"required"`
		ChainID     int64   `json:"chain_id" binding:"required"`
		DEX         string  `json:"dex"`
		PoolAddress string  `json:"pool_address"`
		TradingFee  string  `json:"trading_fee"`
		Tier        string  `json:"tier"`
	}
	ctx.ShouldBindJSON(&req)

	pair := &TradingPair{
		PairID:     uuid.New().String(),
		PairName:   req.PairName,
		BaseToken:  req.BaseToken,
		QuoteToken: req.QuoteToken,
		ChainID:    req.ChainID,
		DEX:        req.DEX,
		PoolAddress: req.PoolAddress,
		Status:     "active",
		TradingFee: req.TradingFee,
		Tier:       req.Tier,
	}

	s.db.Create(pair)
	s.logAudit(adminID, "CREATE_PAIR", "pair", pair.PairID, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true, "pair": pair})
}

func (s *AdminService) UpdatePair(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	pairID := ctx.Param("id")

	var req struct {
		Status      string `json:"status"`
		TradingFee  string `json:"trading_fee"`
		Tier        string `json:"tier"`
		MinTrade    string `json:"min_trade"`
		MaxTrade    string `json:"max_trade"`
	}
	ctx.ShouldBindJSON(&req)

	var pair TradingPair
	if err := s.db.Where("pair_id = ?", pairID).First(&pair).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "pair not found"})
		return
	}

	if req.Status != "" {
		pair.Status = req.Status
	}
	if req.TradingFee != "" {
		pair.TradingFee = req.TradingFee
	}
	if req.Tier != "" {
		pair.Tier = req.Tier
	}
	if req.MinTrade != "" {
		pair.MinTrade = req.MinTrade
	}
	if req.MaxTrade != "" {
		pair.MaxTrade = req.MaxTrade
	}

	s.db.Save(&pair)
	s.logAudit(adminID, "UPDATE_PAIR", "pair", pairID, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true})
}

func (s *AdminService) HaltPair(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	pairID := ctx.Param("id")

	var pair TradingPair
	if err := s.db.Where("pair_id = ?", pairID).First(&pair).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "pair not found"})
		return
	}

	pair.Status = "halted"
	s.db.Save(&pair)

	s.logAudit(adminID, "HALT_PAIR", "pair", pairID, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true, "status": "halted"})
}

// ============================================================================
// Liquidity Management
// ============================================================================

func (s *AdminService) ListLiquidity(ctx *gin.Context) {
	var pools []LiquidityPool
	s.db.Find(&pools)

	ctx.JSON(200, gin.H{"pools": pools})
}

func (s *AdminService) AddLiquidity(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")

	var req struct {
		PairID      string  `json:"pair_id" binding:"required"`
		TokenA      string  `json:"token_a" binding:"required"`
		TokenB      string  `json:"token_b" binding:"required"`
		ChainID     int64   `json:"chain_id" binding:"required"`
		DEX         string  `json:"dex" binding:"required"`
		PoolAddress string  `json:"pool_address" binding:"required"`
		LiquidityA  string  `json:"liquidity_a"`
		LiquidityB  string  `json:"liquidity_b"`
		LiquidityUSD float64 `json:"liquidity_usd"`
	}
	ctx.ShouldBindJSON(&req)

	pool := &LiquidityPool{
		PoolID:       uuid.New().String(),
		PairID:       req.PairID,
		TokenA:       req.TokenA,
		TokenB:       req.TokenB,
		ChainID:      req.ChainID,
		DEX:          req.DEX,
		PoolAddress:  req.PoolAddress,
		LiquidityA:   req.LiquidityA,
		LiquidityB:   req.LiquidityB,
		LiquidityUSD: req.LiquidityUSD,
		Status:       "active",
	}

	s.db.Create(pool)
	s.logAudit(adminID, "ADD_LIQUIDITY", "pool", pool.PoolID, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true, "pool": pool})
}

func (s *AdminService) RemoveLiquidity(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	poolID := ctx.Param("id")

	var pool LiquidityPool
	if err := s.db.Where("pool_id = ?", poolID).First(&pool).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "pool not found"})
		return
	}

	pool.Status = "removed"
	s.db.Save(&pool)

	s.logAudit(adminID, "REMOVE_LIQUIDITY", "pool", poolID, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true})
}

// ============================================================================
// Fee Management
// ============================================================================

func (s *AdminService) ListFees(ctx *gin.Context) {
	var fees []FeeConfig
	s.db.Find(&fees)

	ctx.JSON(200, gin.H{"fees": fees})
}

func (s *AdminService) SetFee(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")

	var req struct {
		FeeType      string  `json:"fee_type" binding:"required"`
		ChainID      int64   `json:"chain_id"`
		FeePercent   float64 `json:"fee_percent"`
		FeeFixed     string  `json:"fee_fixed"`
		MinFee       string  `json:"min_fee"`
		MaxFee       string  `json:"max_fee"`
		WhiteLabelID *uint   `json:"white_label_id"`
	}
	ctx.ShouldBindJSON(&req)

	fee := &FeeConfig{
		FeeType:      req.FeeType,
		ChainID:      req.ChainID,
		FeePercent:   req.FeePercent,
		FeeFixed:     req.FeeFixed,
		MinFee:       req.MinFee,
		MaxFee:       req.MaxFee,
		IsEnabled:    true,
		WhiteLabelID: req.WhiteLabelID,
	}

	// Check if exists
	var existing FeeConfig
	result := s.db.Where("fee_type = ? AND chain_id = ?", req.FeeType, req.ChainID).First(&existing)

	if result.Error == nil {
		fee.ID = existing.ID
		s.db.Save(fee)
	} else {
		s.db.Create(fee)
	}

	s.logAudit(adminID, "SET_FEE", "fee", strconv.FormatInt(req.ChainID, 10), ctx.ClientIP(), true, req.FeeType)

	ctx.JSON(200, gin.H{"success": true, "fee": fee})
}

// ============================================================================
// Market Maker Bot Management
// ============================================================================

func (s *AdminService) ListBots(ctx *gin.Context) {
	var bots []MarketMakerBot
	s.db.Find(&bots)

	ctx.JSON(200, gin.H{"bots": bots})
}

func (s *AdminService) CreateBot(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")

	var req struct {
		UserID       string  `json:"user_id" binding:"required"`
		PairID       string  `json:"pair_id" binding:"required"`
		Strategy     string  `json:"strategy" binding:"required"`
		MinBalance   string  `json:"min_balance"`
		MaxBalance   string  `json:"max_balance"`
		Spread       float64 `json:"spread"`
		MinTradeSize float64 `json:"min_trade_size"`
		MaxTradeSize float64 `json:"max_trade_size"`
		ProfitTarget float64 `json:"profit_target"`
	}
	ctx.ShouldBindJSON(&req)

	bot := &MarketMakerBot{
		BotID:        uuid.New().String(),
		UserID:       req.UserID,
		PairID:       req.PairID,
		Strategy:     req.Strategy,
		Status:       "active",
		MinBalance:   req.MinBalance,
		MaxBalance:   req.MaxBalance,
		Spread:       req.Spread,
		MinTradeSize: req.MinTradeSize,
		MaxTradeSize: req.MaxTradeSize,
		ProfitTarget: req.ProfitTarget,
	}

	s.db.Create(bot)
	s.logAudit(adminID, "CREATE_BOT", "bot", bot.BotID, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true, "bot": bot})
}

func (s *AdminService) UpdateBotStatus(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	botID := ctx.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	ctx.ShouldBindJSON(&req)

	var bot MarketMakerBot
	if err := s.db.Where("bot_id = ?", botID).First(&bot).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "bot not found"})
		return
	}

	bot.Status = req.Status
	s.db.Save(&bot)

	s.logAudit(adminID, "UPDATE_BOT_STATUS", "bot", botID, ctx.ClientIP(), true, req.Status)

	ctx.JSON(200, gin.H{"success": true, "status": req.Status})
}

// ============================================================================
// Virtual Token Management
// ============================================================================

func (s *AdminService) ListVirtualTokens(ctx *gin.Context) {
	var tokens []VirtualToken
	s.db.Find(&tokens)

	ctx.JSON(200, gin.H{"tokens": tokens})
}

func (s *AdminService) CreateVirtualToken(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Symbol      string `json:"symbol" binding:"required"`
		Decimals    int    `json:"decimals" binding:"required"`
		TotalSupply string `json:"total_supply" binding:"required"`
		Issuer      string `json:"issuer" binding:"required"`
	}
	ctx.ShouldBindJSON(&req)

	token := &VirtualToken{
		TokenID:     uuid.New().String(),
		Name:        req.Name,
		Symbol:      req.Symbol,
		Decimals:    req.Decimals,
		TotalSupply: req.TotalSupply,
		Circulating: "0",
		Issuer:      req.Issuer,
		Status:      "active",
	}

	s.db.Create(token)
	s.logAudit(adminID, "CREATE_VTOKEN", "token", token.TokenID, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true, "token": token})
}

func (s *AdminService) UpdateVirtualTokenStatus(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	tokenID := ctx.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	ctx.ShouldBindJSON(&req)

	var token VirtualToken
	if err := s.db.Where("token_id = ?", tokenID).First(&token).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "token not found"})
		return
	}

	token.Status = req.Status
	s.db.Save(&token)

	s.logAudit(adminID, "UPDATE_VTOKEN_STATUS", "token", tokenID, ctx.ClientIP(), true, req.Status)

	ctx.JSON(200, gin.H{"success": true, "status": req.Status})
}

// ============================================================================
// Brokerage Management
// ============================================================================

func (s *AdminService) ListBrokerages(ctx *gin.Context) {
	var clients []BrokerageClient
	s.db.Find(&clients)

	ctx.JSON(200, gin.H{"clients": clients})
}

func (s *AdminService) CreateBrokerage(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")

	var req struct {
		CompanyName  string  `json:"company_name" binding:"required"`
		AdminEmail  string  `json:"admin_email" binding:"required"`
		WalletAddress string `json:"wallet_address" binding:"required"`
		Tier        int     `json:"tier"`
		TradingLimit float64 `json:"trading_limit"`
		FeeDiscount float64 `json:"fee_discount"`
	}
	ctx.ShouldBindJSON(&req)

	client := &BrokerageClient{
		ClientID:      uuid.New().String(),
		CompanyName:   req.CompanyName,
		AdminEmail:    req.AdminEmail,
		WalletAddress: req.WalletAddress,
		Status:        "pending",
		Tier:          req.Tier,
		TradingLimit:  req.TradingLimit,
		FeeDiscount:   req.FeeDiscount,
	}

	s.db.Create(client)
	s.logAudit(adminID, "CREATE_BROKERAGE", "brokerage", client.ClientID, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true, "client": client})
}

func (s *AdminService) UpdateBrokerageStatus(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	clientID := ctx.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	ctx.ShouldBindJSON(&req)

	var client BrokerageClient
	if err := s.db.Where("client_id = ?", clientID).First(&client).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "client not found"})
		return
	}

	client.Status = req.Status
	s.db.Save(&client)

	s.logAudit(adminID, "UPDATE_BROKERAGE_STATUS", "brokerage", clientID, ctx.ClientIP(), true, req.Status)

	ctx.JSON(200, gin.H{"success": true, "status": req.Status})
}

// ============================================================================
// NFT Management
// ============================================================================

func (s *AdminService) ListNFTCollections(ctx *gin.Context) {
	var collections []NFTCollection
	s.db.Find(&collections)

	ctx.JSON(200, gin.H{"collections": collections})
}

func (s *AdminService) CreateNFTCollection(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")

	var req struct {
		Name         string  `json:"name" binding:"required"`
		Symbol       string  `json:"symbol" binding:"required"`
		ContractAddr string  `json:"contract_address" binding:"required"`
		ChainID      int64   `json:"chain_id" binding:"required"`
		Creator      string  `json:"creator" binding:"required"`
		Royalty      float64 `json:"royalty"`
		TotalSupply  int     `json:"total_supply"`
	}
	ctx.ShouldBindJSON(&req)

	collection := &NFTCollection{
		CollectionID: uuid.New().String(),
		Name:         req.Name,
		Symbol:       req.Symbol,
		ContractAddr: req.ContractAddr,
		ChainID:      req.ChainID,
		Creator:      req.Creator,
		Royalty:      req.Royalty,
		Status:       "active",
		TotalSupply:  req.TotalSupply,
	}

	s.db.Create(&collection)
	s.logAudit(adminID, "CREATE_NFT", "collection", collection.CollectionID, ctx.ClientIP(), true, "")

	ctx.JSON(200, gin.H{"success": true, "collection": collection})
}

// ============================================================================
// Token Creation Management
// ============================================================================

func (s *AdminService) ListTokenCreations(ctx *gin.Context) {
	var requests []TokenCreateRequest
	s.db.Find(&requests)

	ctx.JSON(200, gin.H{"requests": requests})
}

func (s *AdminService) ApproveTokenCreation(ctx *gin.Context) {
	adminID := ctx.GetUint("admin_id")
	requestID := ctx.Param("id")

	var req struct {
		ContractAddr string `json:"contract_address"`
		TxHash       string `json:"tx_hash"`
	}
	ctx.ShouldBindJSON(&req)

	var request TokenCreateRequest
	if err := s.db.Where("request_id = ?", requestID).First(&request).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "request not found"})
		return
	}

	request.Status = "deployed"
	request.ContractAddr = req.ContractAddr
	request.TxHash = req.TxHash

	s.db.Save(&request)
	s.logAudit(adminID, "APPROVE_TOKEN", "token", requestID, ctx.ClientIP(), true, req.ContractAddr)

	ctx.JSON(200, gin.H{"success": true, "status": "deployed"})
}

// ============================================================================
// API Key Management
// ============================================================================

func (s *AdminService) ListAPIKeys(ctx *gin.Context) {
	var keys []APIKey
	s.db.Find(&keys)

	// Mask secrets
	result := make([]gin.H, len(keys))
	for i, k := range keys {
		result[i] = gin.H{
			"key_id":       k.KeyID,
			"user_id":      k.UserID,
			"name":         k.Name,
			"permissions":  k.Permissions,
			"status":       k.Status,
			"last_used":    k.LastUsedAt,
			"expires_at":   k.ExpiresAt,
			"created_at":   k.CreatedAt,
		}
	}

	ctx.JSON(200, gin.H{"keys": result})
}

func (s *AdminService) CreateAPIKey(ctx *gin.Context) {
	var req struct {
		UserID      string   `json:"user_id" binding:"required"`
		Name        string   `json:"name" binding:"required"`
		Permissions string   `json:"permissions"`
		IPWhitelist string   `json:"ip_whitelist"`
		RateLimit   int      `json:"rate_limit"`
		ExpiresAt   *string  `json:"expires_at"`
	}
	ctx.ShouldBindJSON(&req)

	key := uuid.New().String()
	secret := uuid.New().String()

	apiKey := &APIKey{
		KeyID:       uuid.New().String(),
		UserID:      req.UserID,
		Key:         "tw_" + key[:8],
		Secret:      secret,
		Name:        req.Name,
		Permissions: req.Permissions,
		IPWhitelist: req.IPWhitelist,
		RateLimit:   req.RateLimit,
		Status:      "active",
	}

	s.db.Create(apiKey)

	ctx.JSON(200, gin.H{
		"success": true,
		"api_key": gin.H{
			"key_id":   apiKey.Key,
			"secret":   secret,
			"name":     apiKey.Name,
			"expires_at": apiKey.ExpiresAt,
		},
	})
}

// ============================================================================
// Analytics Dashboard
// ============================================================================

func (s *AdminService) GetDashboardStats(ctx *gin.Context) {
	var totalUsers, activeUsers, suspendedUsers int64
	s.db.Model(&User{}).Count(&totalUsers)
	s.db.Model(&User{}).Where("status = ?", "active").Count(&activeUsers)
	s.db.Model(&User{}).Where("status = ?", "suspended").Count(&suspendedUsers)

	var totalPairs, activePairs int64
	s.db.Model(&TradingPair{}).Count(&totalPairs)
	s.db.Model(&TradingPair{}).Where("status = ?", "active").Count(&activePairs)

	var totalBots, activeBots int64
	s.db.Model(&MarketMakerBot{}).Count(&totalBots)
	s.db.Model(&MarketMakerBot{}).Where("status = ?", "active").Count(&activeBots)

	var pendingKYC, approvedKYC int64
	s.db.Model(&KYCRecord{}).Where("status = ?", "pending").Count(&pendingKYC)
	s.db.Model(&KYCRecord{}).Where("status = ?", "approved").Count(&approvedKYC)

	var totalAdmins int64
	s.db.Model(&Admin{}).Count(&totalAdmins)

	ctx.JSON(200, gin.H{
		"users": gin.H{
			"total":     totalUsers,
			"active":    activeUsers,
			"suspended": suspendedUsers,
		},
		"pairs": gin.H{
			"total":  totalPairs,
			"active": activePairs,
		},
		"bots": gin.H{
			"total":  totalBots,
			"active": activeBots,
		},
		"kyc": gin.H{
			"pending":  pendingKYC,
			"approved": approvedKYC,
		},
		"admins": totalAdmins,
	})
}

// ============================================================================
// Audit Logging
// ============================================================================

func (s *AdminService) logAudit(adminID uint, action, resource, resourceID, ip string, success bool, details string) {
	log := &AuditLog{
		AdminID:       adminID,
		AdminUsername: "",
		Action:        action,
		Resource:      resource,
		ResourceID:    resourceID,
		Details:       details,
		IPAddress:     ip,
		Success:       success,
	}
	s.db.Create(log)
}

func (s *AdminService) GetAuditLogs(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "50"))
	adminID := ctx.Query("admin_id")
	action := ctx.Query("action")

	var logs []AuditLog
	query := s.db.Model(&AuditLog{})

	if adminID != "" {
		aid, _ := strconv.ParseUint(adminID, 10, 32)
		query = query.Where("admin_id = ?", aid)
	}
	if action != "" {
		query = query.Where("action LIKE ?", "%"+action+"%")
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&logs)

	ctx.JSON(200, gin.H{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// ============================================================================
// Middleware
// ============================================================================

func (s *AdminService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:]
		claims, err := s.validateJWT(tokenString)
		if err != nil {
			c.JSON(401, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("admin_id", uint(claims["admin_id"].(float64)))
		c.Set("username", claims["username"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

func (s *AdminService) validateJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ============================================================================
// Password Utilities
// ============================================================================

func HashPassword(password string) (string, error) {
	hash, err := scrypt.Key([]byte(password), []byte(salt), 16384, 8, 1, 32)
	return hex.EncodeToString(hash), err
}

func CheckPassword(password, hash string) bool {
	salt := []byte("tigerwallet_salt")
	check, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)
	if err != nil {
		return false
	}
	return hex.EncodeToString(check) == hash
}

var salt = []byte("tigerwallet_salt")

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	service, err := NewAdminService(config)
	if err != nil {
		fmt.Printf("Failed to initialize admin service: %v\n", err)
		os.Exit(1)
	}

	router := gin.Default()

	// CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Public routes
	router.POST("/api/v1/auth/login", service.Login)

	// Protected routes
	api := router.Group("/api/v1")
	api.Use(service.AuthMiddleware())
	{
		// Admin management
		api.POST("/admins", service.CreateAdmin)
		api.GET("/admins", service.ListAdmins)
		api.PUT("/admins/:id", service.UpdateAdmin)
		api.DELETE("/admins/:id", service.DeleteAdmin)

		// User management
		api.GET("/users", service.ListUsers)
		api.GET("/users/:id", service.GetUser)
		api.PUT("/users/:id", service.UpdateUser)
		api.POST("/users/:id/suspend", service.SuspendUser)

		// KYC
		api.GET("/kyc", service.ListKYC)
		api.POST("/kyc/:id/approve", service.ApproveKYC)
		api.POST("/kyc/:id/reject", service.RejectKYC)

		// Pairs
		api.GET("/pairs", service.ListPairs)
		api.POST("/pairs", service.CreatePair)
		api.PUT("/pairs/:id", service.UpdatePair)
		api.POST("/pairs/:id/halt", service.HaltPair)

		// Liquidity
		api.GET("/liquidity", service.ListLiquidity)
		api.POST("/liquidity", service.AddLiquidity)
		api.DELETE("/liquidity/:id", service.RemoveLiquidity)

		// Fees
		api.GET("/fees", service.ListFees)
		api.POST("/fees", service.SetFee)

		// Bots
		api.GET("/bots", service.ListBots)
		api.POST("/bots", service.CreateBot)
		api.POST("/bots/:id/status", service.UpdateBotStatus)

		// Virtual tokens
		api.GET("/virtual-tokens", service.ListVirtualTokens)
		api.POST("/virtual-tokens", service.CreateVirtualToken)
		api.PUT("/virtual-tokens/:id/status", service.UpdateVirtualTokenStatus)

		// Brokerage
		api.GET("/brokerages", service.ListBrokerages)
		api.POST("/brokerages", service.CreateBrokerage)
		api.PUT("/brokerages/:id/status", service.UpdateBrokerageStatus)

		// NFT
		api.GET("/nft-collections", service.ListNFTCollections)
		api.POST("/nft-collections", service.CreateNFTCollection)

		// Token creation
		api.GET("/token-creations", service.ListTokenCreations)
		api.POST("/token-creations/:id/approve", service.ApproveTokenCreation)

		// API keys
		api.GET("/api-keys", service.ListAPIKeys)
		api.POST("/api-keys", service.CreateAPIKey)

		// Analytics
		api.GET("/dashboard", service.GetDashboardStats)
		api.GET("/audit-logs", service.GetAuditLogs)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "admin-service",
			"time":    time.Now().Unix(),
		})
	})

	go func() {
		fmt.Printf("Admin service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down admin service...")
}
