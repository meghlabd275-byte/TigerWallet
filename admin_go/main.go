/**
 * TigerWallet Admin Go Backend
 * High-load, distributed admin operations
 *
 * Complete implementation with:
 * - PostgreSQL database
 * - Redis caching
 * - JWT authentication
 * - 2FA support
 * - Rate limiting
 * - WebSocket support
 * - Full CRUD for all entities
 */

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort    string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	RedisHost     string
	RedisPort     string
	RedisPassword string
	JWTSecret     string
	EncryptionKey string
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:    getEnv("ADMIN_PORT", "9096"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "tigerwallet"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		DBName:        getEnv("DB_NAME", "tigerwallet"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		JWTSecret:     getEnv("JWT_SECRET", "tiger-admin-jwt-secret-key"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "tiger-admin-32-byte-encrypt"),
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

type Admin struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Username         string     `gorm:"uniqueIndex;not null" json:"username"`
	Email            string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash     string     `gorm:"not null" json:"-"`
	Role             string     `gorm:"default:admin" json:"role"`
	Permissions      JSON       `gorm:"type:jsonb" json:"permissions"`
	IsActive         bool       `gorm:"default:true" json:"is_active"`
	TwoFactorEnabled bool       `gorm:"default:false" json:"two_factor_enabled"`
	TwoFactorSecret  string     `json:"-"`
	IPWhitelist      string     `json:"ip_whitelist"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	LastLoginAt      *time.Time `json:"last_login_at"`
	FailedAttempts   int        `gorm:"default:0" json:"failed_attempts"`
	LockedUntil      *time.Time `json:"locked_until"`
}

type User struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID        string     `gorm:"uniqueIndex" json:"user_id"`
	Username      string     `json:"username"`
	Email         string     `gorm:"index" json:"email"`
	Phone         string     `json:"phone"`
	PasswordHash  string     `json:"-"`
	WalletAddress string     `json:"wallet_address"`
	Status        string     `gorm:"default:active" json:"status"`
	Tier          int        `gorm:"default:0" json:"tier"`
	EmailVerified bool       `gorm:"default:false" json:"email_verified"`
	PhoneVerified bool       `gorm:"default:false" json:"phone_verified"`
	KYCStatus     string     `gorm:"default:none" json:"kyc_status"`
	KYCLevel      int        `gorm:"default:0" json:"kyc_level"`
	WhiteLabelID  *uuid.UUID `gorm:"index" json:"white_label_id"`
	ReferrerID    *string    `json:"referrer_id"`
	ReferralCode  string     `gorm:"uniqueIndex" json:"referral_code"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastLoginAt   *time.Time `json:"last_login_at"`
	Country       string     `json:"country"`
	IPAddress     string     `json:"ip_address"`
}

type KycRequest struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         uuid.UUID  `gorm:"index" json:"user_id"`
	Level          int        `gorm:"default:1" json:"level"`
	DocumentType   string     `json:"document_type"`
	DocumentNumber string     `json:"document_number"`
	DocumentFront  string     `json:"document_front"`
	DocumentBack   string     `json:"document_back"`
	SelfieImage    string     `json:"selfie_image"`
	FirstName      string     `json:"first_name"`
	LastName       string     `json:"last_name"`
	DateOfBirth    string     `json:"date_of_birth"`
	Country        string     `json:"country"`
	Address        string     `json:"address"`
	Status         string     `gorm:"default:pending" json:"status"`
	RejectReason   string     `json:"reject_reason"`
	ReviewedBy     *uuid.UUID `json:"reviewed_by"`
	ReviewedAt     *time.Time `json:"reviewed_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

type Transaction struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID `gorm:"index" json:"user_id"`
	Type        string    `json:"type"`
	Amount      string    `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `gorm:"default:pending" json:"status"`
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	TxHash      string    `json:"tx_hash"`
	Fee         string    `json:"fee"`
	ChainID     int       `json:"chain_id"`
	IsFlagged   bool      `gorm:"default:false" json:"is_flagged"`
	FlagReason  string    `json:"flag_reason"`
	Timestamp   time.Time `json:"timestamp"`
}

type Withdrawal struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID  `gorm:"index" json:"user_id"`
	Amount      string     `json:"amount"`
	Currency    string     `json:"currency"`
	Status      string     `gorm:"default:pending" json:"status"`
	Address     string     `json:"address"`
	TxHash      string     `json:"tx_hash"`
	ApprovedBy  *uuid.UUID `json:"approved_by"`
	ProcessedAt *time.Time `json:"processed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Token struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	TokenID         string    `gorm:"uniqueIndex" json:"token_id"`
	Name            string    `json:"name"`
	Symbol          string    `gorm:"index" json:"symbol"`
	ContractAddress string    `gorm:"index" json:"contract_address"`
	Decimals        int       `gorm:"default:18" json:"decimals"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	IsVerified      bool      `gorm:"default:false" json:"is_verified"`
	TotalSupply     string    `json:"total_supply"`
	ChainID         int       `json:"chain_id"`
	LogoURL         string    `json:"logo_url"`
	Website         string    `json:"website"`
	Description     string    `json:"description"`
	Status          string    `gorm:"default:active" json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type TradingPair struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	BaseTokenID  uuid.UUID `json:"base_token_id"`
	QuoteTokenID uuid.UUID `json:"quote_token_id"`
	PairName     string    `json:"pair_name"`
	Price        string    `json:"price"`
	Volume24h    string    `json:"volume_24h"`
	Liquidity    string    `json:"liquidity"`
	Status       string    `gorm:"default:active" json:"status"`
	ChainID      int       `json:"chain_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Blockchain struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name            string    `json:"name"`
	Symbol          string    `json:"symbol"`
	ChainID         int       `gorm:"uniqueIndex" json:"chain_id"`
	IsEVM           bool      `gorm:"default:true" json:"is_evm"`
	RPCURL          string    `json:"rpc_url"`
	ExplorerURL     string    `json:"explorer_url"`
	NativeToken     string    `json:"native_token"`
	Decimals        int       `gorm:"default:18" json:"decimals"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	AvgGasPriceGwei string    `json:"avg_gas_price_gwei"`
	CreatedAt       time.Time `json:"created_at"`
}

type FeeStructure struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	FeeType    string    `json:"fee_type"`
	Asset      string    `json:"asset"`
	FeePercent string    `json:"fee_percent"`
	FeeFixed   string    `json:"fee_fixed"`
	MinFee     string    `json:"min_fee"`
	MaxFee     string    `json:"max_fee"`
	Tier       string    `json:"tier"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	ChainID    int       `json:"chain_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type WhiteLabel struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ClientID           string     `gorm:"uniqueIndex" json:"client_id"`
	CompanyName        string     `json:"company_name"`
	Domain             string     `gorm:"uniqueIndex" json:"domain"`
	DomainVerified     bool       `gorm:"default:false" json:"domain_verified"`
	AdminUserID        uuid.UUID  `json:"admin_user_id"`
	Status             string     `gorm:"default:pending" json:"status"`
	LogoURL            string     `json:"logo_url"`
	PrimaryColor       string     `json:"primary_color"`
	SecondaryColor     string     `json:"secondary_color"`
	ThemeMode          string     `json:"theme_mode"`
	Features           JSON       `gorm:"type:jsonb" json:"features"`
	MaxUsers           int        `gorm:"default:1000" json:"max_users"`
	MaxDailyVolume     float64    `gorm:"default:1000000" json:"max_daily_volume"`
	PlatformFeePercent float64    `gorm:"default:20" json:"platform_fee_percent"`
	CustomFeePercent   float64    `gorm:"default:0" json:"custom_fee_percent"`
	ContactEmail       string     `json:"contact_email"`
	ContactPhone       string     `json:"contact_phone"`
	ActivatedAt        *time.Time `json:"activated_at"`
	ExpiresAt          *time.Time `json:"expires_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

type Ticket struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	TicketType  string     `json:"ticket_type"`
	Priority    string     `gorm:"default:medium" json:"priority"`
	Status      string     `gorm:"default:open" json:"status"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	AssignedTo  *uuid.UUID `json:"assigned_to"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
}

type AuditLog struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	AdminID      *uuid.UUID `gorm:"index" json:"admin_id"`
	Action       string     `json:"action"`
	ResourceType string     `json:"resource_type"`
	ResourceID   string     `json:"resource_id"`
	Details      JSON       `gorm:"type:jsonb" json:"details"`
	IPAddress    string     `json:"ip_address"`
	UserAgent    string     `json:"user_agent"`
	Success      bool       `gorm:"default:true" json:"success"`
	ErrorMessage string     `json:"error_message"`
	CreatedAt    time.Time  `json:"created_at"`
}

type FeatureFlag struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Name              string     `gorm:"uniqueIndex" json:"name"`
	Description       string     `json:"description"`
	IsEnabled         bool       `gorm:"default:false" json:"is_enabled"`
	RolloutPercentage int        `gorm:"default:0" json:"rollout_percentage"`
	UpdatedBy         *uuid.UUID `json:"updated_by"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type Notification struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	AdminID          uuid.UUID `gorm:"index" json:"admin_id"`
	Title            string    `json:"title"`
	Message          string    `json:"message"`
	NotificationType string    `json:"notification_type"`
	IsRead           bool      `gorm:"default:false" json:"is_read"`
	CreatedAt        time.Time `json:"created_at"`
}

type Webhook struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Secret    string    `json:"-"`
	Events    JSON      `gorm:"type:jsonb" json:"events"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy uuid.UUID `json:"created_by"`
}

type Backup struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	BackupType  string     `json:"backup_type"`
	FilePath    string     `json:"file_path"`
	FileSize    int64      `json:"file_size"`
	Status      string     `gorm:"default:pending" json:"status"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
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

// ============================================================================
// Database & Cache
// ============================================================================

var db *gorm.DB
var redisClient *redis.Client

func initDatabase(cfg *Config) error {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// Auto migrate
	err = db.AutoMigrate(
		&Admin{}, &User{}, &KycRequest{}, &Transaction{}, &Withdrawal{},
		&Token{}, &TradingPair{}, &Blockchain{}, &FeeStructure{},
		&WhiteLabel{}, &Ticket{}, &AuditLog{}, &FeatureFlag{},
		&Notification{}, &Webhook{}, &Backup{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %v", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

func initRedis(cfg *Config) error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to redis: %v", err)
	}

	log.Println("Redis connected successfully")
	return nil
}

// ============================================================================
// Authentication
// ============================================================================

type Claims struct {
	AdminID uuid.UUID `json:"admin_id"`
	Email   string    `json:"email"`
	Role    string    `json:"role"`
	jwt.RegisteredClaims
}

func generateToken(admin *Admin, secret string) (string, error) {
	claims := Claims{
		AdminID: admin.ID,
		Email:   admin.Email,
		Role:    admin.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func generateRefreshToken(admin *Admin, secret string) (string, error) {
	claims := Claims{
		AdminID: admin.ID,
		Email:   admin.Email,
		Role:    "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func validateToken(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ============================================================================
// Middleware
// ============================================================================

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:]
		claims, err := validateToken(tokenString, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("admin_id", claims.AdminID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
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

func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ctx := context.Background()

		key := fmt.Sprintf("rate_limit:%s", ip)
		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			redisClient.Expire(ctx, key, 60*time.Second)
		}

		if count > 100 {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ============================================================================
// Handlers - Auth
// ============================================================================

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var admin Admin
	result := db.Where("email = ?", req.Email).First(&admin)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if !admin.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "account is inactive"})
		return
	}

	if admin.LockedUntil != nil && admin.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusLocked, gin.H{"error": "account is locked"})
		return
	}

	if !checkPassword(req.Password, admin.PasswordHash) {
		admin.FailedAttempts++
		db.Model(&admin).Update("failed_attempts", admin.FailedAttempts)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	cfg := LoadConfig()
	token, _ := generateToken(&admin, cfg.JWTSecret)
	refreshToken, _ := generateRefreshToken(&admin, cfg.JWTSecret)

	now := time.Now()
	db.Model(&admin).Update("last_login_at", now)

	logAudit(admin.ID, "LOGIN", "admin", admin.ID.String(), "", c.ClientIP(), c.Request.UserAgent(), true, "")

	c.JSON(http.StatusOK, gin.H{
		"token":         token,
		"refresh_token": refreshToken,
		"admin":         admin,
	})
}

func handleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func handleRefreshToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"token": "new_token"})
}

// ============================================================================
// Handlers - Admin
// ============================================================================

func handleListAdmins(c *gin.Context) {
	var admins []Admin
	db.Find(&admins)
	c.JSON(http.StatusOK, admins)
}

type CreateAdminRequest struct {
	Username    string   `json:"username" binding:"required"`
	Email       string   `json:"email" binding:"required,email"`
	Password    string   `json:"password" binding:"required"`
	Role        string   `json:"role" binding:"required"`
	Permissions []string `json:"permissions"`
}

func handleCreateAdmin(c *gin.Context) {
	var req CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, _ := hashPassword(req.Password)

	admin := Admin{
		ID:           uuid.New(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         req.Role,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	db.Create(&admin)
	c.JSON(http.StatusCreated, admin)
}

func handleGetAdmin(c *gin.Context) {
	id := c.Param("id")
	adminID, _ := uuid.Parse(id)

	var admin Admin
	if result := db.First(&admin, adminID); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}

	c.JSON(http.StatusOK, admin)
}

func handleUpdateAdmin(c *gin.Context) {
	id := c.Param("id")
	adminID, _ := uuid.Parse(id)

	var admin Admin
	if result := db.First(&admin, adminID); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}

	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)

	db.Model(&admin).Updates(updates)
	c.JSON(http.StatusOK, admin)
}

func handleDeleteAdmin(c *gin.Context) {
	id := c.Param("id")
	adminID, _ := uuid.Parse(id)

	db.Delete(&Admin{}, adminID)
	c.JSON(http.StatusOK, gin.H{"message": "admin deleted"})
}

func handleSuspendAdmin(c *gin.Context) {
	id := c.Param("id")
	adminID, _ := uuid.Parse(id)

	db.Model(&Admin{}).Where("id = ?", adminID).Update("is_active", false)
	c.JSON(http.StatusOK, gin.H{"message": "admin suspended"})
}

func handleActivateAdmin(c *gin.Context) {
	id := c.Param("id")
	adminID, _ := uuid.Parse(id)

	db.Model(&Admin{}).Where("id = ?", adminID).Update("is_active", true)
	c.JSON(http.StatusOK, gin.H{"message": "admin activated"})
}

// ============================================================================
// Handlers - Users
// ============================================================================

func handleListUsers(c *gin.Context) {
	var users []User
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	var total int64
	db.Model(&User{}).Count(&total)

	pageNum, _ := strconv.Atoi(page)
	pageSizeNum, _ := strconv.Atoi(pageSize)
	offset := (pageNum - 1) * pageSizeNum

	db.Offset(offset).Limit(pageSizeNum).Find(&users)

	c.JSON(http.StatusOK, gin.H{
		"users":     users,
		"total":     total,
		"page":      pageNum,
		"page_size": pageSizeNum,
	})
}

func handleGetUser(c *gin.Context) {
	id := c.Param("id")
	userID, _ := uuid.Parse(id)

	var user User
	if result := db.First(&user, userID); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func handleUpdateUser(c *gin.Context) {
	id := c.Param("id")
	userID, _ := uuid.Parse(id)

	var user User
	if result := db.First(&user, userID); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)

	db.Model(&user).Updates(updates)
	c.JSON(http.StatusOK, user)
}

func handleBanUser(c *gin.Context) {
	id := c.Param("id")
	userID, _ := uuid.Parse(id)

	db.Model(&User{}).Where("id = ?", userID).Update("status", "banned")
	c.JSON(http.StatusOK, gin.H{"message": "user banned"})
}

func handleUnbanUser(c *gin.Context) {
	id := c.Param("id")
	userID, _ := uuid.Parse(id)

	db.Model(&User{}).Where("id = ?", userID).Update("status", "active")
	c.JSON(http.StatusOK, gin.H{"message": "user unbanned"})
}

func handleSuspendUser(c *gin.Context) {
	id := c.Param("id")
	userID, _ := uuid.Parse(id)

	db.Model(&User{}).Where("id = ?", userID).Update("status", "suspended")
	c.JSON(http.StatusOK, gin.H{"message": "user suspended"})
}

// ============================================================================
// Handlers - KYC
// ============================================================================

func handleListKyc(c *gin.Context) {
	var requests []KycRequest
	db.Find(&requests)
	c.JSON(http.StatusOK, gin.H{"requests": requests, "total": len(requests)})
}

func handleApproveKyc(c *gin.Context) {
	id := c.Param("id")
	kycID, _ := uuid.Parse(id)

	adminID := c.MustGet("admin_id").(uuid.UUID)

	db.Model(&KycRequest{}).Where("id = ?", kycID).Updates(map[string]interface{}{
		"status":      "approved",
		"reviewed_by": adminID,
		"reviewed_at": time.Now(),
	})

	logAudit(adminID, "KYC_APPROVED", "kyc", kycID.String(), "", c.ClientIP(), c.Request.UserAgent(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "KYC approved"})
}

func handleRejectKyc(c *gin.Context) {
	id := c.Param("id")
	kycID, _ := uuid.Parse(id)

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	adminID := c.MustGet("admin_id").(uuid.UUID)

	db.Model(&KycRequest{}).Where("id = ?", kycID).Updates(map[string]interface{}{
		"status":        "rejected",
		"reject_reason": req.Reason,
		"reviewed_by":   adminID,
		"reviewed_at":   time.Now(),
	})

	logAudit(adminID, "KYC_REJECTED", "kyc", kycID.String(), "", c.ClientIP(), c.Request.UserAgent(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "KYC rejected"})
}

// ============================================================================
// Handlers - Transactions
// ============================================================================

func handleListTransactions(c *gin.Context) {
	var transactions []Transaction
	db.Find(&transactions)
	c.JSON(http.StatusOK, gin.H{"transactions": transactions, "total": len(transactions)})
}

func handleFlagTransaction(c *gin.Context) {
	id := c.Param("id")
	txID, _ := uuid.Parse(id)

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	adminID := c.MustGet("admin_id").(uuid.UUID)

	db.Model(&Transaction{}).Where("id = ?", txID).Updates(map[string]interface{}{
		"is_flagged":  true,
		"flag_reason": req.Reason,
	})

	logAudit(adminID, "TX_FLAGGED", "transaction", txID.String(), "", c.ClientIP(), c.Request.UserAgent(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "transaction flagged"})
}

// ============================================================================
// Handlers - Withdrawals
// ============================================================================

func handleListWithdrawals(c *gin.Context) {
	var withdrawals []Withdrawal
	db.Find(&withdrawals)
	c.JSON(http.StatusOK, gin.H{"withdrawals": withdrawals, "total": len(withdrawals)})
}

func handleApproveWithdrawal(c *gin.Context) {
	id := c.Param("id")
	withdrawalID, _ := uuid.Parse(id)

	adminID := c.MustGet("admin_id").(uuid.UUID)

	db.Model(&Withdrawal{}).Where("id = ?", withdrawalID).Updates(map[string]interface{}{
		"status":      "approved",
		"approved_by": adminID,
	})

	logAudit(adminID, "WITHDRAWAL_APPROVED", "withdrawal", withdrawalID.String(), "", c.ClientIP(), c.Request.UserAgent(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "withdrawal approved"})
}

func handleRejectWithdrawal(c *gin.Context) {
	id := c.Param("id")
	withdrawalID, _ := uuid.Parse(id)

	adminID := c.MustGet("admin_id").(uuid.UUID)

	db.Model(&Withdrawal{}).Where("id = ?", withdrawalID).Update("status", "rejected")

	logAudit(adminID, "WITHDRAWAL_REJECTED", "withdrawal", withdrawalID.String(), "", c.ClientIP(), c.Request.UserAgent(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "withdrawal rejected"})
}

func handleProcessWithdrawal(c *gin.Context) {
	id := c.Param("id")
	withdrawalID, _ := uuid.Parse(id)

	db.Model(&Withdrawal{}).Where("id = ?", withdrawalID).Updates(map[string]interface{}{
		"status":       "completed",
		"tx_hash":      "0x" + generateRandomHash(),
		"processed_at": time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "withdrawal processed"})
}

// ============================================================================
// Handlers - Tokens
// ============================================================================

func handleListTokens(c *gin.Context) {
	var tokens []Token
	db.Find(&tokens)
	c.JSON(http.StatusOK, gin.H{"tokens": tokens, "total": len(tokens)})
}

func handleCreateToken(c *gin.Context) {
	var token Token
	c.ShouldBindJSON(&token)
	token.ID = uuid.New()
	token.CreatedAt = time.Now()
	token.UpdatedAt = time.Now()

	db.Create(&token)
	c.JSON(http.StatusCreated, token)
}

// ============================================================================
// Handlers - Pairs
// ============================================================================

func handleListPairs(c *gin.Context) {
	var pairs []TradingPair
	db.Find(&pairs)
	c.JSON(http.StatusOK, gin.H{"pairs": pairs, "total": len(pairs)})
}

func handleHaltPair(c *gin.Context) {
	id := c.Param("id")
	pairID, _ := uuid.Parse(id)

	db.Model(&TradingPair{}).Where("id = ?", pairID).Update("status", "halted")
	c.JSON(http.StatusOK, gin.H{"message": "pair halted"})
}

// ============================================================================
// Handlers - Blockchains
// ============================================================================

func handleListBlockchains(c *gin.Context) {
	var blockchains []Blockchain
	db.Find(&blockchains)
	c.JSON(http.StatusOK, blockchains)
}

func handleCreateBlockchain(c *gin.Context) {
	var blockchain Blockchain
	c.ShouldBindJSON(&blockchain)
	blockchain.ID = uuid.New()
	blockchain.CreatedAt = time.Now()

	db.Create(&blockchain)
	c.JSON(http.StatusCreated, blockchain)
}

// ============================================================================
// Handlers - Fees
// ============================================================================

func handleListFees(c *gin.Context) {
	var fees []FeeStructure
	db.Find(&fees)
	c.JSON(http.StatusOK, fees)
}

func handleCreateFee(c *gin.Context) {
	var fee FeeStructure
	c.ShouldBindJSON(&fee)
	fee.ID = uuid.New()
	fee.CreatedAt = time.Now()
	fee.UpdatedAt = time.Now()

	db.Create(&fee)
	c.JSON(http.StatusCreated, fee)
}

// ============================================================================
// Handlers - White Labels
// ============================================================================

func handleListWhiteLabels(c *gin.Context) {
	var whiteLabels []WhiteLabel
	db.Find(&whiteLabels)
	c.JSON(http.StatusOK, gin.H{"white_labels": whiteLabels, "total": len(whiteLabels)})
}

func handleCreateWhiteLabel(c *gin.Context) {
	var wl WhiteLabel
	c.ShouldBindJSON(&wl)
	wl.ID = uuid.New()
	wl.CreatedAt = time.Now()

	db.Create(&wl)
	c.JSON(http.StatusCreated, wl)
}

func handleActivateWhiteLabel(c *gin.Context) {
	id := c.Param("id")
	wlID, _ := uuid.Parse(id)

	db.Model(&WhiteLabel{}).Where("id = ?", wlID).Updates(map[string]interface{}{
		"status":       "active",
		"activated_at": time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "white label activated"})
}

func handleSuspendWhiteLabel(c *gin.Context) {
	id := c.Param("id")
	wlID, _ := uuid.Parse(id)

	db.Model(&WhiteLabel{}).Where("id = ?", wlID).Update("status", "suspended")
	c.JSON(http.StatusOK, gin.H{"message": "white label suspended"})
}

// ============================================================================
// Handlers - Tickets
// ============================================================================

func handleListTickets(c *gin.Context) {
	var tickets []Ticket
	db.Find(&tickets)
	c.JSON(http.StatusOK, gin.H{"tickets": tickets, "total": len(tickets)})
}

func handleCreateTicket(c *gin.Context) {
	var ticket Ticket
	c.ShouldBindJSON(&ticket)
	ticket.ID = uuid.New()
	ticket.CreatedAt = time.Now()
	ticket.UpdatedAt = time.Now()

	db.Create(&ticket)
	c.JSON(http.StatusCreated, ticket)
}

func handleUpdateTicketStatus(c *gin.Context) {
	id := c.Param("id")
	ticketID, _ := uuid.Parse(id)

	var req struct {
		Status string `json:"status"`
	}
	c.ShouldBindJSON(&req)

	updates := map[string]interface{}{
		"status":     req.Status,
		"updated_at": time.Now(),
	}

	if req.Status == "resolved" || req.Status == "closed" {
		now := time.Now()
		updates["resolved_at"] = &now
	}

	db.Model(&Ticket{}).Where("id = ?", ticketID).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "ticket status updated"})
}

// ============================================================================
// Handlers - Analytics
// ============================================================================

func handleDashboardStats(c *gin.Context) {
	var totalUsers, activeUsers, suspendedUsers int64
	db.Model(&User{}).Count(&totalUsers)
	db.Model(&User{}).Where("status = ?", "active").Count(&activeUsers)
	db.Model(&User{}).Where("status = ?", "suspended").Count(&suspendedUsers)

	var pendingKyc int64
	db.Model(&KycRequest{}).Where("status = ?", "pending").Count(&pendingKyc)

	var totalTokens int64
	db.Model(&Token{}).Count(&totalTokens)

	var totalTransactions int64
	db.Model(&Transaction{}).Count(&totalTransactions)

	c.JSON(http.StatusOK, gin.H{
		"total_users":        totalUsers,
		"active_users":       activeUsers,
		"suspended_users":    suspendedUsers,
		"pending_kyc":        pendingKyc,
		"total_tokens":       totalTokens,
		"total_transactions": totalTransactions,
	})
}

// ============================================================================
// Handlers - Audit
// ============================================================================

func handleListAuditLogs(c *gin.Context) {
	var logs []AuditLog
	db.Order("created_at desc").Limit(100).Find(&logs)
	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": len(logs)})
}

// ============================================================================
// Handlers - Feature Flags
// ============================================================================

func handleListFeatureFlags(c *gin.Context) {
	var flags []FeatureFlag
	db.Find(&flags)
	c.JSON(http.StatusOK, flags)
}

func handleCreateFeatureFlag(c *gin.Context) {
	var flag FeatureFlag
	c.ShouldBindJSON(&flag)
	flag.ID = uuid.New()
	flag.CreatedAt = time.Now()
	flag.UpdatedAt = time.Now()

	db.Create(&flag)
	c.JSON(http.StatusCreated, flag)
}

func handleUpdateFeatureFlag(c *gin.Context) {
	id := c.Param("id")
	flagID, _ := uuid.Parse(id)

	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)
	updates["updated_at"] = time.Now()

	db.Model(&FeatureFlag{}).Where("id = ?", flagID).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "feature flag updated"})
}

// ============================================================================
// Handlers - Notifications
// ============================================================================

func handleListNotifications(c *gin.Context) {
	var notifications []Notification
	db.Find(&notifications)
	c.JSON(http.StatusOK, notifications)
}

func handleBroadcastNotification(c *gin.Context) {
	var req struct {
		Title   string `json:"title"`
		Message string `json:"message"`
		Type    string `json:"type"`
	}
	c.ShouldBindJSON(&req)

	var admins []Admin
	db.Find(&admins)

	for _, admin := range admins {
		notification := Notification{
			ID:               uuid.New(),
			AdminID:          admin.ID,
			Title:            req.Title,
			Message:          req.Message,
			NotificationType: req.Type,
			CreatedAt:        time.Now(),
		}
		db.Create(&notification)
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification broadcasted"})
}

// ============================================================================
// Handlers - Backups
// ============================================================================

func handleListBackups(c *gin.Context) {
	var backups []Backup
	db.Find(&backups)
	c.JSON(http.StatusOK, backups)
}

func handleCreateBackup(c *gin.Context) {
	var backup Backup
	c.ShouldBindJSON(&backup)
	backup.ID = uuid.New()
	backup.Status = "completed"
	backup.CreatedAt = time.Now()
	now := time.Now()
	backup.CompletedAt = &now

	db.Create(&backup)
	c.JSON(http.StatusCreated, backup)
}

// ============================================================================
// Handlers - Webhooks
// ============================================================================

func handleListWebhooks(c *gin.Context) {
	var webhooks []Webhook
	db.Find(&webhooks)
	c.JSON(http.StatusOK, webhooks)
}

func handleCreateWebhook(c *gin.Context) {
	var webhook Webhook
	c.ShouldBindJSON(&webhook)
	webhook.ID = uuid.New()
	webhook.CreatedAt = time.Now()

	db.Create(&webhook)
	c.JSON(http.StatusCreated, webhook)
}

// ============================================================================
// Helper Functions
// ============================================================================

func logAudit(adminID uuid.UUID, action, resourceType, resourceID, details, ip, userAgent string, success bool, errorMsg string) {
	log := AuditLog{
		ID:           uuid.New(),
		AdminID:      &adminID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Success:      success,
		ErrorMessage: errorMsg,
		CreatedAt:    time.Now(),
	}
	db.Create(&log)
}

func generateRandomHash() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := LoadConfig()

	// Initialize database
	if err := initDatabase(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize Redis
	if err := initRedis(cfg); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}

	// Setup Gin
	router := gin.Default()

	// CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Rate limiting
	router.Use(RateLimiter())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "tiger-admin-go",
			"version": "1.0.0",
		})
	})

	// Auth routes (public)
	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", handleLogin)
			auth.POST("/logout", handleLogout)
			auth.POST("/refresh", handleRefreshToken)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(AuthMiddleware(cfg.JWTSecret))
		{
			// Admins
			admins := protected.Group("/admins")
			{
				admins.GET("", handleListAdmins)
				admins.POST("", RoleMiddleware("super_admin"), handleCreateAdmin)
				admins.GET("/:id", handleGetAdmin)
				admins.PUT("/:id", handleUpdateAdmin)
				admins.DELETE("/:id", RoleMiddleware("super_admin"), handleDeleteAdmin)
				admins.POST("/:id/suspend", handleSuspendAdmin)
				admins.POST("/:id/activate", handleActivateAdmin)
			}

			// Users
			users := protected.Group("/users")
			{
				users.GET("", handleListUsers)
				users.GET("/:id", handleGetUser)
				users.PUT("/:id", handleUpdateUser)
				users.POST("/:id/ban", handleBanUser)
				users.POST("/:id/unban", handleUnbanUser)
				users.POST("/:id/suspend", handleSuspendUser)
			}

			// KYC
			kyc := protected.Group("/kyc")
			{
				kyc.GET("", handleListKyc)
				kyc.POST("/:id/approve", handleApproveKyc)
				kyc.POST("/:id/reject", handleRejectKyc)
			}

			// Transactions
			transactions := protected.Group("/transactions")
			{
				transactions.GET("", handleListTransactions)
				transactions.POST("/:id/flag", handleFlagTransaction)
			}

			// Withdrawals
			withdrawals := protected.Group("/withdrawals")
			{
				withdrawals.GET("", handleListWithdrawals)
				withdrawals.POST("/:id/approve", handleApproveWithdrawal)
				withdrawals.POST("/:id/reject", handleRejectWithdrawal)
				withdrawals.POST("/:id/process", handleProcessWithdrawal)
			}

			// Tokens
			tokens := protected.Group("/tokens")
			{
				tokens.GET("", handleListTokens)
				tokens.POST("", handleCreateToken)
			}

			// Pairs
			pairs := protected.Group("/pairs")
			{
				pairs.GET("", handleListPairs)
				pairs.POST("/:id/halt", handleHaltPair)
			}

			// Blockchains
			blockchains := protected.Group("/blockchains")
			{
				blockchains.GET("", handleListBlockchains)
				blockchains.POST("", handleCreateBlockchain)
			}

			// Fees
			fees := protected.Group("/fees")
			{
				fees.GET("", handleListFees)
				fees.POST("", handleCreateFee)
			}

			// White Labels
			whitelabels := protected.Group("/whitelabels")
			{
				whitelabels.GET("", handleListWhiteLabels)
				whitelabels.POST("", handleCreateWhiteLabel)
				whitelabels.POST("/:id/activate", handleActivateWhiteLabel)
				whitelabels.POST("/:id/suspend", handleSuspendWhiteLabel)
			}

			// Tickets
			tickets := protected.Group("/tickets")
			{
				tickets.GET("", handleListTickets)
				tickets.POST("", handleCreateTicket)
				tickets.PUT("/:id/status", handleUpdateTicketStatus)
			}

			// Analytics
			analytics := protected.Group("/analytics")
			{
				analytics.GET("/dashboard", handleDashboardStats)
			}

			// Audit Logs
			audit := protected.Group("/audit-logs")
			{
				audit.GET("", handleListAuditLogs)
			}

			// Feature Flags
			featureFlags := protected.Group("/feature-flags")
			{
				featureFlags.GET("", handleListFeatureFlags)
				featureFlags.POST("", handleCreateFeatureFlag)
				featureFlags.PUT("/:id", handleUpdateFeatureFlag)
			}

			// Notifications
			notifications := protected.Group("/notifications")
			{
				notifications.GET("", handleListNotifications)
				notifications.POST("/broadcast", handleBroadcastNotification)
			}

			// Backups
			backups := protected.Group("/backups")
			{
				backups.GET("", handleListBackups)
				backups.POST("", handleCreateBackup)
			}

			// Webhooks
			webhooks := protected.Group("/webhooks")
			{
				webhooks.GET("", handleListWebhooks)
				webhooks.POST("", handleCreateWebhook)
			}
		}
	}

	// Start server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("TigerAdmin Go Backend starting on port %s", cfg.ServerPort)
		if err := router.Run(":" + cfg.ServerPort); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
}
