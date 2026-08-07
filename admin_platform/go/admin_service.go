/**
 * TigerWallet Admin Platform Service
 * Complete Admin & White Label Management System
 *
 * Features:
 * - Super Admin & Sub Admin management
 * - White Label client management
 * - User management with KYC
 * - Token & Pair management
 * - Fee management
 * - Blockchain management
 * - Analytics & Monitoring
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
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort    string `json:"server_port"`
	DBHost        string `json:"db_host"`
	DBPort        string `json:"db_port"`
	DBUser        string `json:"db_user"`
	DBPassword    string `json:"db_password"`
	DBName        string `json:"db_name"`
	RedisHost     string `json:"redis_host"`
	RedisPort     string `json:"redis_port"`
	JWTSecret     string `json:"jwt_secret"`
	EncryptionKey string `json:"encryption_key"`
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:    getEnv("ADMIN_PORT", "9093"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "tigerwallet"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		DBName:        getEnv("DB_NAME", "tigerwallet"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		JWTSecret:     getEnv("JWT_SECRET", "admin-secret-key-change"),
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

// Admin User
type Admin struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Username     string     `gorm:"uniqueIndex" json:"username"`
	Email        string     `gorm:"index" json:"email"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"` // super_admin, admin, support, finance, compliance
	Permissions  JSON       `json:"permissions" gorm:"type:jsonb"`
	Status       string     `json:"status"` // active, suspended, inactive
	LastLoginAt  *time.Time `json:"last_login_at"`
	IPWhitelist  string     `json:"ip_whitelist"`
	CreatedBy    uint       `json:"created_by"`
}

type JSON json.RawMessage

func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JSON: %v", value)
	}
	*j = JSON(bytes)
	return nil
}

func (j JSON) Value() (interface{}, error) {
	return json.RawMessage(j).MarshalJSON()
}

// White Label Client
type WhiteLabelClient struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ClientID       string `gorm:"uniqueIndex" json:"client_id"`
	CompanyName    string `json:"company_name"`
	Domain         string `json:"domain"`
	DomainVerified bool   `json:"domain_verified"`

	AdminUserID uint   `json:"admin_user_id"`
	Status      string `json:"status"` // active, suspended, pending

	// Branding
	LogoURL        string `json:"logo_url"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	ThemeMode      string `json:"theme_mode"` // light, dark, both

	// Features
	Features JSON `json:"features" gorm:"type:jsonb"`

	// Limits
	MaxUsers       int     `json:"max_users"`
	MaxDailyVolume float64 `json:"max_daily_volume"`

	// Fees
	PlatformFeePercent float64 `json:"platform_fee_percent"`
	CustomFeePercent   float64 `json:"custom_fee_percent"`

	// Liquidity
	LiquiditySource    string `json:"liquidity_source"`
	TradingPairsImport string `json:"trading_pairs_import"`

	// Contacts
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`

	ActivatedAt *time.Time `json:"activated_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// White Label Sub-admin
type WhiteLabelAdmin struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	WhiteLabelID uint      `gorm:"index" json:"white_label_id"`
	UserID       uint      `gorm:"index" json:"user_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // owner, admin, manager, support
	Permissions  JSON      `json:"permissions" gorm:"type:jsonb"`
	Status       string    `json:"status"`
}

// User Management
type User struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID       string `gorm:"uniqueIndex" json:"user_id"`
	Username     string `gorm:"index" json:"username"`
	Email        string `gorm:"index" json:"email"`
	Phone        string `json:"phone"`
	PasswordHash string `json:"-"`

	MasterWalletAddr string `json:"master_wallet_address"`
	Status           string `json:"status"` // active, suspended, banned
	Tier             int    `json:"tier"`   // 0: basic, 1: verified, 2: premium

	IsEmailVerified bool   `json:"is_email_verified"`
	IsPhoneVerified bool   `json:"is_phone_verified"`
	KYCStatus       string `json:"kyc_status"` // none, pending, approved, rejected
	KYCLevel        int    `json:"kyc_level"`

	WhiteLabelID *uint `gorm:"index" json:"white_label_id"`

	ReferrerID   *string `json:"referrer_id"`
	ReferralCode string  `gorm:"uniqueIndex" json:"referral_code"`

	LastLoginAt      *time.Time `json:"last_login_at"`
	FailedLoginCount int        `json:"failed_login_count"`
}

// KYC Records
type KYCRecord struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Level     int       `json:"level"`

	DocumentType   string `json:"document_type"` // passport, id_card, drivers_license
	DocumentNumber string `json:"document_number"`
	DocumentFront  string `json:"document_front"` // encrypted URL
	DocumentBack   string `json:"document_back"`
	SelfieImage    string `json:"selfie_image"`

	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"`
	Country     string `json:"country"`
	Address     string `json:"address"`

	Status       string     `json:"status"` // pending, approved, rejected
	RejectReason string     `json:"reject_reason"`
	ReviewedBy   uint       `json:"reviewed_by"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
}

// Token Management
type Token struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	TokenID      string `gorm:"uniqueIndex" json:"token_id"`
	Name         string `json:"name"`
	Symbol       string `gorm:"index" json:"symbol"`
	ContractAddr string `gorm:"index" json:"contract_addr"`
	Decimals     int    `json:"decimals"`
	TotalSupply  string `json:"total_supply"`

	ChainID   uint   `gorm:"index" json:"chain_id"`
	ChainName string `json:"chain_name"`

	IsActive      bool `json:"is_active"`
	IsVerified    bool `json:"is_verified"`
	IsNativeToken bool `json:"is_native_token"`

	LogoURL     string `json:"logo_url"`
	Website     string `json:"website"`
	Whitepaper  string `json:"whitepaper"`
	Description string `json:"description"`

	MarketCap float64 `json:"market_cap"`
	Price     float64 `json:"price"`

	CreatedBy uint `json:"created_by"`
}

// Trading Pair
type TradingPair struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	PairID       string `gorm:"uniqueIndex" json:"pair_id"`
	BaseTokenID  uint   `gorm:"index" json:"base_token_id"`
	QuoteTokenID uint   `gorm:"index" json:"quote_token_id"`

	BaseSymbol  string `json:"base_symbol"`
	QuoteSymbol string `json:"quote_symbol"`
	PairName    string `json:"pair_name"`

	ChainID   uint   `json:"chain_id"`
	ChainName string `json:"chain_name"`

	Status         string `json:"status"` // active, halted, suspended, removed
	TradingEnabled bool   `json:"trading_enabled"`

	// Trading parameters
	MinTradeAmount float64 `json:"min_trade_amount"`
	MaxTradeAmount float64 `json:"max_trade_amount"`
	MinTradeValue  float64 `json:"min_trade_value"`

	// Fees
	MakerFee float64 `json:"maker_fee"`
	TakerFee float64 `json:"taker_fee"`

	// Liquidity
	PoolAddress string  `json:"pool_address"`
	Liquidity   float64 `json:"liquidity"`

	// Price
	CurrentPrice   float64 `json:"current_price"`
	PriceChange24h float64 `json:"price_change_24h"`
	Volume24h      float64 `json:"volume_24h"`

	// Source
	Source         string `json:"source"`          // internal, imported
	SourceExchange string `json:"source_exchange"` // binance, coinbase, etc.

	WhiteLabelID *uint `gorm:"index" json:"white_label_id"`
}

// Fee Management
type FeeConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	FeeType string `json:"fee_type"` // trading, withdrawal, deposit, transfer
	ChainID *uint  `json:"chain_id"`
	TokenID *uint  `json:"token_id"`

	FeePercent float64 `json:"fee_percent"`
	FeeFixed   float64 `json:"fee_fixed"`
	MinFee     float64 `json:"min_fee"`
	MaxFee     float64 `json:"max_fee"`

	WhiteLabelID *uint `gorm:"index" json:"white_label_id"`

	IsActive bool `json:"is_active"`
}

// Blockchain Management
type Blockchain struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ChainID uint   `gorm:"uniqueIndex" json:"chain_id"`
	Name    string `json:"name"`
	Symbol  string `gorm:"index" json:"symbol"`
	Type    string `json:"type"` // evm, bitcoin, solana, etc.

	RPCURLs      JSON `json:"rpc_urls" gorm:"type:jsonb"`
	ExplorerURLs JSON `json:"explorer_urls" gorm:"type:jsonb"`

	IsActive  bool `json:"is_active"`
	IsTestnet bool `json:"is_testnet"`

	CoinGeckoID     string `json:"coin_gecko_id"`
	CoingeckoSymbol string `json:"coin_gecko_symbol"`

	Confirmations int `json:"confirmations"`
	BlockTime     int `json:"block_time"` // seconds

	NativeTokenID uint `json:"native_token_id"`
}

// Market Maker Bot
type MarketMakerBot struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	BotID       string `gorm:"uniqueIndex" json:"bot_id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	OwnerID      uint  `gorm:"index" json:"owner_id"`
	WhiteLabelID *uint `gorm:"index" json:"white_label_id"`

	Status string `json:"status"` // active, paused, stopped

	// Strategy
	StrategyType string  `json:"strategy_type"` // arbitrage, market_making, grid
	BaseSpread   float64 `json:"base_spread"`   // percentage
	MaxSpread    float64 `json:"max_spread"`
	OrderSize    float64 `json:"order_size"`

	// Pairs
	TradingPairs JSON `json:"trading_pairs" gorm:"type:jsonb"`

	// Capital
	AllocatedCapital float64 `json:"allocated_capital"`
	UsedCapital      float64 `json:"used_capital"`

	// Performance
	TotalVolume24h float64 `json:"total_volume_24h"`
	ProfitLoss24h  float64 `json:"profit_loss_24h"`

	// Limits
	MaxSlippage   float64 `json:"max_slippage"`
	MaxOpenOrders int     `json:"max_open_orders"`
}

// Analytics
type AnalyticsEvent struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	EventType    string    `json:"event_type"` // trade, deposit, withdraw, transfer
	UserID       uint      `gorm:"index" json:"user_id"`
	WhiteLabelID *uint     `gorm:"index" json:"white_label_id"`

	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Token    string  `json:"token"`

	Fee       float64 `json:"fee"`
	IPAddress string  `json:"ip_address"`
	UserAgent string  `json:"user_agent"`

	Metadata JSON `json:"metadata" gorm:"type:jsonb"`
}

// Audit Log
type AdminAuditLog struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	AdminID      uint      `gorm:"index" json:"admin_id"`
	Action       string    `json:"action"`   // user.create, token.update, etc.
	Resource     string    `json:"resource"` // user:123, token:456
	ResourceType string    `json:"resource_type"`
	Details      JSON      `json:"details" gorm:"type:jsonb"`
	IPAddress    string    `json:"ip_address"`
	Success      bool      `json:"success"`
	Error        string    `json:"error"`
}

// ============================================================================
// Service Layer
// ============================================================================

type AdminService struct {
	db        *gorm.DB
	redis     *redis.Client
	config    *Config
	jwtSecret []byte
	encKey    []byte
}

func NewAdminService(cfg *Config) (*AdminService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&Admin{}, &WhiteLabelClient{}, &WhiteLabelAdmin{},
		&User{}, &KYCRecord{}, &Token{}, &TradingPair{},
		&FeeConfig{}, &Blockchain{}, &MarketMakerBot{},
		&AnalyticsEvent{}, &AdminAuditLog{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "",
		DB:       3,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	// Create encryption key
	encKey := []byte(cfg.EncryptionKey)
	if len(encKey) < 32 {
		padded := make([]byte, 32)
		copy(padded, encKey)
		encKey = padded
	}

	service := &AdminService{
		db:        db,
		redis:     rdb,
		config:    cfg,
		jwtSecret: []byte(cfg.JWTSecret),
		encKey:    encKey,
	}

	// Create default super admin if not exists
	service.createDefaultSuperAdmin()

	return service, nil
}

func (s *AdminService) createDefaultSuperAdmin() {
	var admin Admin
	result := s.db.Where("role = ?", "super_admin").First(&admin)
	if result.Error != nil {
		admin := Admin{
			Username:     "superadmin",
			Email:        "admin@tigerwallet.com",
			PasswordHash: s.hashPassword("TigerWallet2024!"),
			Role:         "super_admin",
			Permissions:  JSON(`{"all":true}`),
			Status:       "active",
		}
		s.db.Create(&admin)
		log.Println("Created default super admin")
	}
}

func (s *AdminService) hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func (s *AdminService) checkPassword(password, hash string) bool {
	return s.hashPassword(password) == hash
}

func (s *AdminService) generateJWT(admin *Admin) (string, error) {
	claims := jwt.MapClaims{
		"admin_id": admin.ID,
		"username": admin.Username,
		"role":     admin.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AdminService) validateJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (s *AdminService) encrypt(data string) string {
	block, _ := aes.NewCipher(s.encKey)
	aesGCM, _ := cipher.NewGCM(block)
	nonce := make([]byte, aesGCM.NonceSize())
	rand.Read(nonce)
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(data), nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func (s *AdminService) decrypt(data string) string {
	ciphertext, _ := base64.StdEncoding.DecodeString(data)
	block, _ := aes.NewCipher(s.encKey)
	aesGCM, _ := cipher.NewGCM(block)
	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, _ := aesGCM.Open(nil, nonce, ciphertext, nil)
	return string(plaintext)
}

// ============================================================================
// Admin Auth Handlers
// ============================================================================

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *AdminService) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var admin Admin
	result := s.db.Where("username = ? OR email = ?", req.Username, req.Username).First(&admin)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if !s.checkPassword(req.Password, admin.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if admin.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is not active"})
		return
	}

	token, err := s.generateJWT(&admin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Update last login
	now := time.Now()
	admin.LastLoginAt = &now
	s.db.Save(&admin)

	// Log audit
	s.logAudit(admin.ID, "admin.login", "admin", admin.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"admin": gin.H{
			"id":       admin.ID,
			"username": admin.Username,
			"email":    admin.Email,
			"role":     admin.Role,
		},
	})
}

func (s *AdminService) GetProfile(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var admin Admin
	s.db.First(&admin, adminID)

	c.JSON(http.StatusOK, gin.H{
		"id":         admin.ID,
		"username":   admin.Username,
		"email":      admin.Email,
		"role":       admin.Role,
		"status":     admin.Status,
		"last_login": admin.LastLoginAt,
	})
}

// ============================================================================
// Admin Management
// ============================================================================

type CreateAdminRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required"`
}

func (s *AdminService) CreateAdmin(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("role")

	if role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if exists
	var existing Admin
	result := s.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existing)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "admin already exists"})
		return
	}

	admin := Admin{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: s.hashPassword(req.Password),
		Role:         req.Role,
		Permissions:  JSON(`{}`),
		Status:       "active",
		CreatedBy:    adminID,
	}

	s.db.Create(&admin)

	s.logAudit(adminID, "admin.create", "admin", admin.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusCreated, gin.H{"id": admin.ID, "username": admin.Username, "role": admin.Role})
}

func (s *AdminService) ListAdmins(c *gin.Context) {
	role := c.Query("role")

	var admins []Admin
	query := s.db
	if role != "" {
		query = query.Where("role = ?", role)
	}
	query.Find(&admins)

	c.JSON(http.StatusOK, gin.H{"admins": admins})
}

func (s *AdminService) UpdateAdmin(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	targetID := c.Param("id")

	var admin Admin
	if err := s.db.First(&admin, targetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}

	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)

	// Don't allow role change for super_admin
	if admin.Role == "super_admin" {
		delete(updates, "role")
	}

	s.db.Model(&admin).Updates(updates)

	s.logAudit(adminID, "admin.update", "admin", admin.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "admin updated"})
}

func (s *AdminService) DeleteAdmin(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("role")

	if role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	targetID := c.Param("id")
	if targetID == fmt.Sprintf("%d", adminID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete yourself"})
		return
	}

	s.db.Delete(&Admin{}, targetID)

	s.logAudit(adminID, "admin.delete", "admin", adminID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "admin deleted"})
}

// ============================================================================
// White Label Management
// ============================================================================

type CreateWhiteLabelRequest struct {
	CompanyName string `json:"company_name" binding:"required"`
	Domain      string `json:"domain" binding:"required"`
	AdminUserID uint   `json:"admin_user_id" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
}

func (s *AdminService) CreateWhiteLabel(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req CreateWhiteLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if domain exists
	var existing WhiteLabelClient
	result := s.db.Where("domain = ?", req.Domain).First(&existing)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "domain already exists"})
		return
	}

	client := WhiteLabelClient{
		ClientID:           "WL-" + uuid.New().String()[:8],
		CompanyName:        req.CompanyName,
		Domain:             req.Domain,
		AdminUserID:        req.AdminUserID,
		Status:             "pending",
		ThemeMode:          "both",
		Features:           JSON(`{"trading":true,"staking":true,"nft":true,"defi":true}`),
		MaxUsers:           10000,
		PlatformFeePercent: 0.1,
		ContactEmail:       req.Email,
	}

	s.db.Create(&client)

	s.logAudit(adminID, "whitelabel.create", "whitelabel", client.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusCreated, client)
}

func (s *AdminService) ListWhiteLabels(c *gin.Context) {
	status := c.Query("status")

	var clients []WhiteLabelClient
	query := s.db
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Find(&clients)

	c.JSON(http.StatusOK, gin.H{"clients": clients})
}

func (s *AdminService) GetWhiteLabel(c *gin.Context) {
	clientID := c.Param("id")

	var client WhiteLabelClient
	if err := s.db.First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}

	c.JSON(http.StatusOK, client)
}

func (s *AdminService) UpdateWhiteLabel(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	clientID := c.Param("id")

	var client WhiteLabelClient
	if err := s.db.First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}

	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)

	s.db.Model(&client).Updates(updates)

	s.logAudit(adminID, "whitelabel.update", "whitelabel", client.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "client updated"})
}

func (s *AdminService) ActivateWhiteLabel(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	clientID := c.Param("id")

	var client WhiteLabelClient
	if err := s.db.First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}

	now := time.Now()
	client.Status = "active"
	client.ActivatedAt = &now
	client.DomainVerified = true

	s.db.Save(&client)

	s.logAudit(adminID, "whitelabel.activate", "whitelabel", client.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "client activated"})
}

func (s *AdminService) SuspendWhiteLabel(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	clientID := c.Param("id")

	var client WhiteLabelClient
	if err := s.db.First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}

	client.Status = "suspended"
	s.db.Save(&client)

	s.logAudit(adminID, "whitelabel.suspend", "whitelabel", client.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "client suspended"})
}

// ============================================================================
// User Management
// ============================================================================

func (s *AdminService) ListUsers(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")
	status := c.Query("status")
	kycStatus := c.Query("kyc_status")

	var users []User
	query := s.db.Model(&User{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if kycStatus != "" {
		query = query.Where("kyc_status = ?", kycStatus)
	}

	var total int64
	query.Count(&total)

	pageNum := 0
	fmt.Sscanf(page, "%d", &pageNum)
	limitNum := 0
	fmt.Sscanf(limit, "%d", &limitNum)

	offset := (pageNum - 1) * limitNum
	query.Offset(offset).Limit(limitNum).Find(&users)

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  pageNum,
		"limit": limitNum,
		"pages": (total + int64(limitNum) - 1) / int64(limitNum),
	})
}

func (s *AdminService) GetUser(c *gin.Context) {
	userID := c.Param("id")

	var user User
	if err := s.db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Get KYC records
	var kycRecords []KYCRecord
	s.db.Where("user_id = ?", user.ID).Find(&kycRecords)

	c.JSON(http.StatusOK, gin.H{
		"user":        user,
		"kyc_records": kycRecords,
	})
}

func (s *AdminService) UpdateUser(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	userID := c.Param("id")

	var user User
	if err := s.db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)

	// Don't allow certain updates
	delete(updates, "user_id")
	delete(updates, "password_hash")

	s.db.Model(&user).Updates(updates)

	s.logAudit(adminID, "user.update", "user", user.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}

func (s *AdminService) SuspendUser(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	userID := c.Param("id")

	var user User
	if err := s.db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	user.Status = "suspended"
	s.db.Save(&user)

	s.logAudit(adminID, "user.suspend", "user", user.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "user suspended"})
}

// ============================================================================
// KYC Management
// ============================================================================

func (s *AdminService) ListKYC(c *gin.Context) {
	status := c.Query("status")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")

	var records []KYCRecord
	query := s.db.Model(&KYCRecord{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	pageNum := 0
	fmt.Sscanf(page, "%d", &pageNum)
	limitNum := 0
	fmt.Sscanf(limit, "%d", &limitNum)

	offset := (pageNum - 1) * limitNum
	query.Offset(offset).Limit(limitNum).Find(&records)

	// Get user info for each record
	type KYCWithUser struct {
		KYCRecord
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	var result []KYCWithUser
	for _, record := range records {
		var user User
		s.db.First(&user, record.UserID)
		result = append(result, KYCWithUser{
			KYCRecord: record,
			Username:  user.Username,
			Email:     user.Email,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"records": result,
		"total":   total,
		"page":    pageNum,
		"limit":   limitNum,
	})
}

func (s *AdminService) ApproveKYC(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	recordID := c.Param("id")

	var record KYCRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
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
	user.Tier = record.Level + 1
	s.db.Save(&user)

	s.logAudit(adminID, "kyc.approve", "kyc", record.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "KYC approved"})
}

func (s *AdminService) RejectKYC(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	recordID := c.Param("id")

	var record KYCRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	record.Status = "rejected"
	record.RejectReason = req.Reason
	record.ReviewedBy = adminID
	now := time.Now()
	record.ReviewedAt = &now

	s.db.Save(&record)

	s.logAudit(adminID, "kyc.reject", "kyc", record.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "KYC rejected"})
}

// ============================================================================
// Token Management
// ============================================================================

func (s *AdminService) ListTokens(c *gin.Context) {
	chainID := c.Query("chain_id")
	active := c.Query("active")

	var tokens []Token
	query := s.db.Model(&Token{})
	if chainID != "" {
		var chainIDUint uint
		fmt.Sscanf(chainID, "%d", &chainIDUint)
		query = query.Where("chain_id = ?", chainIDUint)
	}
	if active != "" {
		query = query.Where("is_active = ?", active == "true")
	}
	query.Find(&tokens)

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

func (s *AdminService) CreateToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var token Token
	if err := c.ShouldBindJSON(&token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token.TokenID = "TKN-" + uuid.New().String()[:8]
	token.IsActive = true
	token.CreatedBy = adminID

	s.db.Create(&token)

	s.logAudit(adminID, "token.create", "token", token.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusCreated, token)
}

func (s *AdminService) UpdateToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	tokenID := c.Param("id")

	var token Token
	if err := s.db.First(&token, tokenID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}

	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)

	s.db.Model(&token).Updates(updates)

	s.logAudit(adminID, "token.update", "token", token.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "token updated"})
}

func (s *AdminService) DeleteToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	tokenID := c.Param("id")

	s.db.Delete(&Token{}, tokenID)

	s.logAudit(adminID, "token.delete", "token", adminID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "token deleted"})
}

// ============================================================================
// Trading Pair Management
// ============================================================================

func (s *AdminService) ListPairs(c *gin.Context) {
	status := c.Query("status")
	chainID := c.Query("chain_id")

	var pairs []TradingPair
	query := s.db.Model(&TradingPair{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if chainID != "" {
		var chainIDUint uint
		fmt.Sscanf(chainID, "%d", &chainIDUint)
		query = query.Where("chain_id = ?", chainIDUint)
	}

	query.Find(&pairs)

	c.JSON(http.StatusOK, gin.H{"pairs": pairs})
}

func (s *AdminService) CreatePair(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var pair TradingPair
	if err := c.ShouldBindJSON(&pair); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair.PairID = "PAIR-" + uuid.New().String()[:8]
	pair.Status = "active"
	pair.TradingEnabled = true
	pair.Source = "internal"

	s.db.Create(&pair)

	s.logAudit(adminID, "pair.create", "pair", pair.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusCreated, pair)
}

func (s *AdminService) UpdatePair(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	pairID := c.Param("id")

	var pair TradingPair
	if err := s.db.First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pair not found"})
		return
	}

	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)

	s.db.Model(&pair).Updates(updates)

	s.logAudit(adminID, "pair.update", "pair", pair.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "pair updated"})
}

func (s *AdminService) HaltPair(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	pairID := c.Param("id")

	var pair TradingPair
	if err := s.db.First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pair not found"})
		return
	}

	pair.Status = "halted"
	pair.TradingEnabled = false

	s.db.Save(&pair)

	s.logAudit(adminID, "pair.halt", "pair", pair.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "pair halted"})
}

// ============================================================================
// Blockchain Management
// ============================================================================

func (s *AdminService) ListBlockchains(c *gin.Context) {
	var blockchains []Blockchain
	s.db.Find(&blockchains)

	c.JSON(http.StatusOK, gin.H{"blockchains": blockchains})
}

func (s *AdminService) CreateBlockchain(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var chain Blockchain
	if err := c.ShouldBindJSON(&chain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chain.IsActive = true

	s.db.Create(&chain)

	s.logAudit(adminID, "blockchain.create", "blockchain", chain.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusCreated, chain)
}

func (s *AdminService) UpdateBlockchain(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	chainID := c.Param("id")

	var chain Blockchain
	if err := s.db.First(&chain, chainID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "blockchain not found"})
		return
	}

	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)

	s.db.Model(&chain).Updates(updates)

	s.logAudit(adminID, "blockchain.update", "blockchain", chain.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "blockchain updated"})
}

// ============================================================================
// Fee Management
// ============================================================================

func (s *AdminService) ListFees(c *gin.Context) {
	var fees []FeeConfig
	s.db.Find(&fees)

	c.JSON(http.StatusOK, gin.H{"fees": fees})
}

func (s *AdminService) UpdateFee(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	feeID := c.Param("id")

	var fee FeeConfig
	if err := s.db.First(&fee, feeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "fee not found"})
		return
	}

	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)

	s.db.Model(&fee).Updates(updates)

	s.logAudit(adminID, "fee.update", "fee", fee.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "fee updated"})
}

// ============================================================================
// Market Maker Bot Management
// ============================================================================

func (s *AdminService) ListBots(c *gin.Context) {
	status := c.Query("status")

	var bots []MarketMakerBot
	query := s.db.Model(&MarketMakerBot{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Find(&bots)

	c.JSON(http.StatusOK, gin.H{"bots": bots})
}

func (s *AdminService) CreateBot(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var bot MarketMakerBot
	if err := c.ShouldBindJSON(&bot); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bot.BotID = "BOT-" + uuid.New().String()[:8]
	bot.Status = "paused"
	bot.AllocatedCapital = 0
	bot.UsedCapital = 0

	s.db.Create(&bot)

	s.logAudit(adminID, "bot.create", "bot", bot.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusCreated, bot)
}

func (s *AdminService) UpdateBotStatus(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	botID := c.Param("id")

	var bot MarketMakerBot
	if err := s.db.First(&bot, botID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	bot.Status = req.Status
	s.db.Save(&bot)

	s.logAudit(adminID, "bot.status.update", "bot", bot.ID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "bot status updated"})
}

// ============================================================================
// Analytics
// ============================================================================

func (s *AdminService) GetDashboardStats(c *gin.Context) {
	// User stats
	var totalUsers, activeUsers, suspendedUsers int64
	s.db.Model(&User{}).Count(&totalUsers)
	s.db.Model(&User{}).Where("status = ?", "active").Count(&activeUsers)
	s.db.Model(&User{}).Where("status = ?", "suspended").Count(&suspendedUsers)

	// White label stats
	var totalWL, activeWL int64
	s.db.Model(&WhiteLabelClient{}).Count(&totalWL)
	s.db.Model(&WhiteLabelClient{}).Where("status = ?", "active").Count(&activeWL)

	// KYC stats
	var pendingKYC, approvedKYC, rejectedKYC int64
	s.db.Model(&KYCRecord{}).Where("status = ?", "pending").Count(&pendingKYC)
	s.db.Model(&KYCRecord{}).Where("status = ?", "approved").Count(&approvedKYC)
	s.db.Model(&KYCRecord{}).Where("status = ?", "rejected").Count(&rejectedKYC)

	// Token stats
	var totalTokens, activeTokens int64
	s.db.Model(&Token{}).Count(&totalTokens)
	s.db.Model(&Token{}).Where("is_active = ?", true).Count(&activeTokens)

	// Pair stats
	var totalPairs, activePairs int64
	s.db.Model(&TradingPair{}).Count(&totalPairs)
	s.db.Model(&TradingPair{}).Where("status = ?", "active").Count(&activePairs)

	// Bot stats
	var totalBots, activeBots int64
	s.db.Model(&MarketMakerBot{}).Count(&totalBots)
	s.db.Model(&MarketMakerBot{}).Where("status = ?", "active").Count(&activeBots)

	c.JSON(http.StatusOK, gin.H{
		"users": gin.H{
			"total":     totalUsers,
			"active":    activeUsers,
			"suspended": suspendedUsers,
		},
		"white_labels": gin.H{
			"total":  totalWL,
			"active": activeWL,
		},
		"kyc": gin.H{
			"pending":  pendingKYC,
			"approved": approvedKYC,
			"rejected": rejectedKYC,
		},
		"tokens": gin.H{
			"total":  totalTokens,
			"active": activeTokens,
		},
		"pairs": gin.H{
			"total":  totalPairs,
			"active": activePairs,
		},
		"bots": gin.H{
			"total":  totalBots,
			"active": activeBots,
		},
	})
}

func (s *AdminService) GetAuditLogs(c *gin.Context) {
	adminID := c.Query("admin_id")
	action := c.Query("action")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "50")

	var logs []AdminAuditLog
	query := s.db.Model(&AdminAuditLog{})

	if adminID != "" {
		var adminIDUint uint
		fmt.Sscanf(adminID, "%d", &adminIDUint)
		query = query.Where("admin_id = ?", adminIDUint)
	}
	if action != "" {
		query = query.Where("action LIKE ?", "%"+action+"%")
	}

	var total int64
	query.Count(&total)

	pageNum := 0
	fmt.Sscanf(page, "%d", &pageNum)
	limitNum := 0
	fmt.Sscanf(limit, "%d", &limitNum)

	offset := (pageNum - 1) * limitNum
	query.Order("created_at DESC").Offset(offset).Limit(limitNum).Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"page":  pageNum,
		"limit": limitNum,
	})
}

// ============================================================================
// Audit Logging
// ============================================================================

func (s *AdminService) logAudit(adminID uint, action, resourceType string, resourceID uint, ip string, success bool, details string) {
	log := AdminAuditLog{
		AdminID:      adminID,
		Action:       action,
		Resource:     fmt.Sprintf("%s:%d", resourceType, resourceID),
		ResourceType: resourceType,
		Details:      JSON(`{}`),
		IPAddress:    ip,
		Success:      success,
		Error:        details,
	}
	s.db.Create(&log)
}

// ============================================================================
// Middleware
// ============================================================================

func (s *AdminService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:]
		claims, err := s.validateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("admin_id", uint(claims["admin_id"].(float64)))
		c.Set("username", claims["username"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

func (s *AdminService) RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")

		for _, allowedRole := range allowedRoles {
			if role == allowedRole || role == "super_admin" {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		c.Abort()
	}
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	service, err := NewAdminService(config)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

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
		// Admin profile
		api.GET("/profile", service.GetProfile)

		// Admin management
		api.POST("/admins", service.RoleMiddleware("super_admin"), service.CreateAdmin)
		api.GET("/admins", service.ListAdmins)
		api.PUT("/admins/:id", service.RoleMiddleware("super_admin"), service.UpdateAdmin)
		api.DELETE("/admins/:id", service.RoleMiddleware("super_admin"), service.DeleteAdmin)

		// White Label
		api.POST("/whitelabels", service.CreateWhiteLabel)
		api.GET("/whitelabels", service.ListWhiteLabels)
		api.GET("/whitelabels/:id", service.GetWhiteLabel)
		api.PUT("/whitelabels/:id", service.UpdateWhiteLabel)
		api.POST("/whitelabels/:id/activate", service.ActivateWhiteLabel)
		api.POST("/whitelabels/:id/suspend", service.SuspendWhiteLabel)

		// Users
		api.GET("/users", service.ListUsers)
		api.GET("/users/:id", service.GetUser)
		api.PUT("/users/:id", service.UpdateUser)
		api.POST("/users/:id/suspend", service.SuspendUser)

		// KYC
		api.GET("/kyc", service.ListKYC)
		api.POST("/kyc/:id/approve", service.ApproveKYC)
		api.POST("/kyc/:id/reject", service.RejectKYC)

		// Tokens
		api.GET("/tokens", service.ListTokens)
		api.POST("/tokens", service.CreateToken)
		api.PUT("/tokens/:id", service.UpdateToken)
		api.DELETE("/tokens/:id", service.DeleteToken)

		// Trading Pairs
		api.GET("/pairs", service.ListPairs)
		api.POST("/pairs", service.CreatePair)
		api.PUT("/pairs/:id", service.UpdatePair)
		api.POST("/pairs/:id/halt", service.HaltPair)

		// Blockchains
		api.GET("/blockchains", service.ListBlockchains)
		api.POST("/blockchains", service.CreateBlockchain)
		api.PUT("/blockchains/:id", service.UpdateBlockchain)

		// Fees
		api.GET("/fees", service.ListFees)
		api.PUT("/fees/:id", service.UpdateFee)

		// Market Maker Bots
		api.GET("/bots", service.ListBots)
		api.POST("/bots", service.CreateBot)
		api.POST("/bots/:id/status", service.UpdateBotStatus)

		// Analytics
		api.GET("/dashboard", service.GetDashboardStats)
		api.GET("/audit-logs", service.GetAuditLogs)
	}

	// Health
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "admin-platform",
			"timestamp": time.Now().Unix(),
		})
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Admin Platform starting on port %s", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
}
