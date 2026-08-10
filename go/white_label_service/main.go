/**
 * TigerWallet White Label Management Service
 * Go High-Load Distributed Backend
 * Real PostgreSQL, Redis, Complete REST API
 */

package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// CONFIGURATION
// ============================================================================

type Config struct {
	ServerPort     string
	DatabaseURL    string
	RedisURL       string
	JWTSecret      string
	AllowedOrigins string
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:     getEnv("WL_SERVICE_PORT", "8085"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://tigerwallet:tigerpass@localhost:5432/tigerwallet?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

var config = LoadConfig()

// ============================================================================
// DATABASE
// ============================================================================

var db *sql.DB
var redisClient *redis.Client
var ctx = context.Background()

func InitDatabase(connStr string) error {
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("✅ Database connected successfully")
	return nil
}

func InitRedis(redisURL string) error {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("failed to parse redis URL: %w", err)
	}

	redisClient = redis.NewClient(opt)

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Println("✅ Redis connected successfully")
	return nil
}

// ============================================================================
// MODELS
// ============================================================================

type AdminRole string

const (
	RoleSuperAdmin      AdminRole = "super_admin"
	RoleMasterAdmin     AdminRole = "master_admin"
	RoleWhiteLabelAdmin AdminRole = "white_label_admin"
	RoleSupport         AdminRole = "support"
)

type WhiteLabelStatus string

const (
	WLStatusPending   WhiteLabelStatus = "pending"
	WLStatusActive    WhiteLabelStatus = "active"
	WLStatusSuspended WhiteLabelStatus = "suspended"
	WLStatusRevoked   WhiteLabelStatus = "revoked"
)

type Admin struct {
	ID               string     `json:"id"`
	Username         string     `json:"username"`
	Email            string     `json:"email"`
	PasswordHash     string     `json:"-"`
	Role             AdminRole  `json:"role"`
	SecurityLevel    int        `json:"security_level"`
	Permissions      []string   `json:"permissions"`
	TwoFactorEnabled bool       `json:"two_factor_enabled"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	LastLogin        *time.Time `json:"last_login"`
}

type WhiteLabel struct {
	ID                  string           `json:"id"`
	Name                string           `json:"name"`
	Domain              string           `json:"domain"`
	Subdomain           *string          `json:"subdomain"`
	APIKeyHash          string           `json:"-"`
	Status              WhiteLabelStatus `json:"status"`
	FeePercent          float64          `json:"fee_percent"`
	ProfitSharePercent  float64          `json:"profit_share_percent"`
	PlanTier            string           `json:"plan_tier"`
	MaxUsers            int              `json:"max_users"`
	MaxAPICalls         int              `json:"max_api_calls"`
	MonthlyFee          float64          `json:"monthly_fee"`
	BrandingConfig      json.RawMessage  `json:"branding_config"`
	Features            []string         `json:"features"`
	MasterWalletAddress *string          `json:"master_wallet_address"`
	CustomBranding      bool             `json:"custom_branding"`
	SupportEmail        *string          `json:"support_email"`
	TermsURL            *string          `json:"terms_url"`
	PrivacyURL          *string          `json:"privacy_url"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
	ApprovedAt          *time.Time       `json:"approved_at"`
}

type WhiteLabelAdmin struct {
	ID           string     `json:"id"`
	WhiteLabelID string     `json:"white_label_id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	Permissions  []string   `json:"permissions"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	LastLogin    *time.Time `json:"last_login"`
}

type Product struct {
	ID              string    `json:"id"`
	WhiteLabelID    *string   `json:"white_label_id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Description     *string   `json:"description"`
	Status          string    `json:"status"`
	FeePercent      float64   `json:"fee_percent"`
	MinDeposit      float64   `json:"min_deposit"`
	MaxDeposit      *float64  `json:"max_deposit"`
	Features        []string  `json:"features"`
	SupportedChains []string  `json:"supported_chains"`
	IsGlobal        bool      `json:"is_global"`
	CreatedAt       time.Time `json:"created_at"`
}

type TradingPair struct {
	ID                string    `json:"id"`
	WhiteLabelID      *string   `json:"white_label_id"`
	BaseToken         string    `json:"base_token"`
	QuoteToken        string    `json:"quote_token"`
	PairSymbol        string    `json:"pair_symbol"`
	MinTradeAmount    float64   `json:"min_trade_amount"`
	MaxTradeAmount    *float64  `json:"max_trade_amount"`
	MakerFee          float64   `json:"maker_fee"`
	TakerFee          float64   `json:"taker_fee"`
	Status            string    `json:"status"`
	ChainID           *string   `json:"chain_id"`
	PricePrecision    int       `json:"price_precision"`
	QuantityPrecision int       `json:"quantity_precision"`
	CreatedAt         time.Time `json:"created_at"`
}

type User struct {
	ID            string          `json:"id"`
	WhiteLabelID  *string         `json:"white_label_id"`
	Email         *string         `json:"email"`
	Username      *string         `json:"username"`
	WalletAddress *string         `json:"wallet_address"`
	KYCStatus     string          `json:"kyc_status"`
	Status        string          `json:"status"`
	RiskScore     int             `json:"risk_score"`
	Metadata      json.RawMessage `json:"metadata"`
	Country       *string         `json:"country"`
	CreatedAt     time.Time       `json:"created_at"`
	LastActive    *time.Time      `json:"last_active"`
}

type Transaction struct {
	ID           string    `json:"id"`
	UserID       *string   `json:"user_id"`
	WhiteLabelID *string   `json:"white_label_id"`
	TxHash       *string   `json:"tx_hash"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	FromAddress  *string   `json:"from_address"`
	ToAddress    *string   `json:"to_address"`
	TokenSymbol  *string   `json:"token_symbol"`
	ChainID      *string   `json:"chain_id"`
	Amount       float64   `json:"amount"`
	Fee          float64   `json:"fee"`
	USDValue     *float64  `json:"usd_value"`
	CreatedAt    time.Time `json:"created_at"`
}

type Blockchain struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Symbol       string   `json:"symbol"`
	ChainID      string   `json:"chain_id"`
	ChainType    *string  `json:"chain_type"`
	NativeToken  string   `json:"native_token"`
	Decimals     int      `json:"decimals"`
	IsActive     bool     `json:"is_active"`
	IsTestnet    bool     `json:"is_testnet"`
	RPCURLs      []string `json:"rpc_urls"`
	ExplorerURLs []string `json:"explorer_urls"`
}

type FeeStructure struct {
	ID           string  `json:"id"`
	WhiteLabelID *string `json:"white_label_id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	TokenSymbol  *string `json:"token_symbol"`
	ChainID      *string `json:"chain_id"`
	FeePercent   float64 `json:"fee_percent"`
	FeeFixed     float64 `json:"fee_fixed"`
	IsActive     bool    `json:"is_active"`
}

type APIKey struct {
	ID           string     `json:"id"`
	WhiteLabelID string     `json:"white_label_id"`
	Name         string     `json:"name"`
	KeyHash      string     `json:"-"`
	Permissions  []string   `json:"permissions"`
	RateLimitMin int        `json:"rate_limit_minute"`
	RateLimitDay int        `json:"rate_limit_day"`
	IsActive     bool       `json:"is_active"`
	ExpiresAt    *time.Time `json:"expires_at"`
	LastUsed     *time.Time `json:"last_used"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Webhook struct {
	ID            string     `json:"id"`
	WhiteLabelID  string     `json:"white_label_id"`
	Name          string     `json:"name"`
	URL           string     `json:"url"`
	Events        []string   `json:"events"`
	SecretHash    string     `json:"-"`
	IsActive      bool       `json:"is_active"`
	RetryCount    int        `json:"retry_count"`
	LastTriggered *time.Time `json:"last_triggered"`
	FailureCount  int        `json:"failure_count"`
	CreatedAt     time.Time  `json:"created_at"`
}

type Notification struct {
	ID           string          `json:"id"`
	UserID       *string         `json:"user_id"`
	WhiteLabelID *string         `json:"white_label_id"`
	Type         string          `json:"type"`
	Title        string          `json:"title"`
	Message      *string         `json:"message"`
	Data         json.RawMessage `json:"data"`
	IsRead       bool            `json:"is_read"`
	CreatedAt    time.Time       `json:"created_at"`
}

// ============================================================================
// AUTHENTICATION
// ============================================================================

type JWTClaims struct {
	AdminID      string `json:"admin_id"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	WhiteLabelID string `json:"white_label_id,omitempty"`
	jwt.RegisteredClaims
}

func GenerateToken(adminID, username, role, whiteLabelID string) (string, error) {
	claims := JWTClaims{
		AdminID:      adminID,
		Username:     username,
		Role:         role,
		WhiteLabelID: whiteLabelID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.JWTSecret))
}

func ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// ============================================================================
// MIDDLEWARE
// ============================================================================

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("admin_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("white_label_id", claims.WhiteLabelID)

		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Role not found"})
			c.Abort()
			return
		}

		roleStr := role.(string)
		for _, allowed := range allowedRoles {
			if roleStr == allowed || roleStr == string(RoleSuperAdmin) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("rate_limit:%s:%s", ip, c.FullPath())

		count, err := redisClient.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			log.Printf("Redis error: %v", err)
		}

		if count >= 100 {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		pipe := redisClient.Pipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, 60*time.Second)
		_, err = pipe.Exec(ctx)

		c.Next()
	}
}

// ============================================================================
// ADMIN HANDLERS
// ============================================================================

func LoginHandler(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Rate limiting
	ip := c.ClientIP()
	rateKey := fmt.Sprintf("login_attempt:%s", ip)
	attempts, _ := redisClient.Get(ctx, rateKey).Int()
	if attempts >= 5 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many login attempts"})
		return
	}

	redisClient.Incr(ctx, rateKey)
	redisClient.Expire(ctx, rateKey, 15*time.Minute)

	var admin Admin
	err := db.QueryRow(`
		SELECT id, username, email, password_hash, role, status 
		FROM admin_users 
		WHERE username = $1 OR email = $1
	`, req.Username).Scan(&admin.ID, &admin.Username, &admin.Email, &admin.PasswordHash, &admin.Role, &admin.Status)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if admin.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active"})
		return
	}

	if !VerifyPassword(req.Password, admin.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate token
	token, err := GenerateToken(admin.ID, admin.Username, string(admin.Role), "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Update last login
	db.Exec("UPDATE admin_users SET last_login = NOW() WHERE id = $1", admin.ID)

	// Log audit
	logAudit(admin.ID, "LOGIN", "admin", admin.ID, "Login successful", c.ClientIP())

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

func GetAdminsHandler(c *gin.Context) {
	role := c.Query("role")
	search := c.Query("search")

	query := "SELECT id, username, email, role, status, created_at, last_login FROM admin_users WHERE 1=1"
	args := []interface{}{}
	argNum := 1

	if role != "" {
		query += fmt.Sprintf(" AND role = $%d", argNum)
		args = append(args, role)
		argNum++
	}

	if search != "" {
		query += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d)", argNum, argNum)
		args = append(args, "%"+search+"%")
		argNum++
	}

	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var admins []gin.H
	for rows.Next() {
		var admin Admin
		if err := rows.Scan(&admin.ID, &admin.Username, &admin.Email, &admin.Role, &admin.Status, &admin.CreatedAt, &admin.LastLogin); err != nil {
			continue
		}
		admins = append(admins, gin.H{
			"id":         admin.ID,
			"username":   admin.Username,
			"email":      admin.Email,
			"role":       admin.Role,
			"status":     admin.Status,
			"created_at": admin.CreatedAt,
			"last_login": admin.LastLogin,
		})
	}

	c.JSON(http.StatusOK, gin.H{"admins": admins})
}

func CreateAdminHandler(c *gin.Context) {
	adminID, _ := c.Get("admin_id")

	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Role     string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email format"})
		return
	}

	// Hash password
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Validate role
	validRoles := []string{string(RoleMasterAdmin), string(RoleWhiteLabelAdmin), string(RoleSupport)}
	valid := false
	for _, r := range validRoles {
		if req.Role == r {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	id := uuid.New().String()
	_, err = db.Exec(`
		INSERT INTO admin_users (id, username, email, password_hash, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'active', NOW(), NOW())
	`, id, req.Username, req.Email, hashedPassword, req.Role)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin: " + err.Error()})
		return
	}

	logAudit(adminID.(string), "CREATE_ADMIN", "admin", id, "Created admin: "+req.Username, c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{
		"id":       id,
		"username": req.Username,
		"email":    req.Email,
		"role":     req.Role,
	})
}

func UpdateAdminHandler(c *gin.Context) {
	adminID := c.Param("id")
	requestingAdminID, _ := c.Get("admin_id")
	requestingRole, _ := c.Get("role")

	// Only super admin can update others, or admin can update themselves
	if requestingRole != string(RoleSuperAdmin) && requestingAdminID != adminID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot update other admins"})
		return
	}

	var req struct {
		Username    *string  `json:"username"`
		Email       *string  `json:"email"`
		Role        *string  `json:"role"`
		Permissions []string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update query
	query := "UPDATE admin_users SET updated_at = NOW()"
	args := []interface{}{}
	argNum := 1

	if req.Username != nil {
		query += fmt.Sprintf(", username = $%d", argNum)
		args = append(args, *req.Username)
		argNum++
	}

	if req.Email != nil {
		query += fmt.Sprintf(", email = $%d", argNum)
		args = append(args, *req.Email)
		argNum++
	}

	if req.Role != nil && requestingRole == string(RoleSuperAdmin) {
		query += fmt.Sprintf(", role = $%d", argNum)
		args = append(args, *req.Role)
		argNum++
	}

	if req.Permissions != nil {
		permsJSON, _ := json.Marshal(req.Permissions)
		query += fmt.Sprintf(", permissions = $%d", argNum)
		args = append(args, string(permsJSON))
		argNum++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argNum)
	args = append(args, adminID)

	_, err := db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(requestingAdminID.(string), "UPDATE_ADMIN", "admin", adminID, "Updated admin", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "Admin updated successfully"})
}

func SuspendAdminHandler(c *gin.Context) {
	adminID := c.Param("id")
	requestingAdminID, _ := c.Get("admin_id")

	_, err := db.Exec(`
		UPDATE admin_users SET status = 'suspended', updated_at = NOW() WHERE id = $1
	`, adminID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(requestingAdminID.(string), "SUSPEND_ADMIN", "admin", adminID, "Suspended admin", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "Admin suspended"})
}

// ============================================================================
// WHITE LABEL HANDLERS
// ============================================================================

func GetWhiteLabelsHandler(c *gin.Context) {
	status := c.Query("status")
	search := c.Query("search")

	query := `SELECT id, name, domain, status, fee_percent, profit_share_percent, plan_tier, 
		max_users, monthly_fee, custom_branding, created_at, updated_at 
		FROM white_labels WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, status)
		argNum++
	}

	if search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR domain ILIKE $%d)", argNum, argNum)
		args = append(args, "%"+search+"%")
		argNum++
	}

	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var whiteLabels []gin.H
	for rows.Next() {
		var wl WhiteLabel
		if err := rows.Scan(&wl.ID, &wl.Name, &wl.Domain, &wl.Status, &wl.FeePercent,
			&wl.ProfitSharePercent, &wl.PlanTier, &wl.MaxUsers, &wl.MonthlyFee,
			&wl.CustomBranding, &wl.CreatedAt, &wl.UpdatedAt); err != nil {
			continue
		}
		whiteLabels = append(whiteLabels, gin.H{
			"id":                   wl.ID,
			"name":                 wl.Name,
			"domain":               wl.Domain,
			"status":               wl.Status,
			"fee_percent":          wl.FeePercent,
			"profit_share_percent": wl.ProfitSharePercent,
			"plan_tier":            wl.PlanTier,
			"max_users":            wl.MaxUsers,
			"monthly_fee":          wl.MonthlyFee,
			"custom_branding":      wl.CustomBranding,
			"created_at":           wl.CreatedAt,
			"updated_at":           wl.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"white_labels": whiteLabels})
}

func GetWhiteLabelHandler(c *gin.Context) {
	wlID := c.Param("id")

	var wl WhiteLabel
	err := db.QueryRow(`
		SELECT id, name, domain, subdomain, status, fee_percent, profit_share_percent, 
			plan_tier, max_users, max_api_calls, monthly_fee, branding_config, features,
			master_wallet_address, custom_branding, support_email, terms_url, privacy_url,
			created_at, updated_at, approved_at
		FROM white_labels WHERE id = $1
	`, wlID).Scan(&wl.ID, &wl.Name, &wl.Domain, &wl.Subdomain, &wl.Status,
		&wl.FeePercent, &wl.ProfitSharePercent, &wl.PlanTier, &wl.MaxUsers,
		&wl.MaxAPICalls, &wl.MonthlyFee, &wl.BrandingConfig, &wl.Features,
		&wl.MasterWalletAddress, &wl.CustomBranding, &wl.SupportEmail,
		&wl.TermsURL, &wl.PrivacyURL, &wl.CreatedAt, &wl.UpdatedAt, &wl.ApprovedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "White label not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"white_label": wl})
}

func CreateWhiteLabelHandler(c *gin.Context) {
	adminID, _ := c.Get("admin_id")

	var req struct {
		Name       string  `json:"name" binding:"required"`
		Domain     string  `json:"domain" binding:"required"`
		PlanTier   string  `json:"plan_tier"`
		MaxUsers   int     `json:"max_users"`
		MonthlyFee float64 `json:"monthly_fee"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate API key
	apiKey := uuid.New().String() + "-" + uuid.New().String()
	apiKeyHash := HashAPIKey(apiKey)

	// Set defaults
	if req.PlanTier == "" {
		req.PlanTier = "basic"
	}
	if req.MaxUsers == 0 {
		req.MaxUsers = 1000
	}

	planTiers := map[string]int{"basic": 1000, "professional": 10000, "enterprise": 100000}
	maxUsers := planTiers[req.PlanTier]
	if req.MaxUsers > 0 {
		maxUsers = req.MaxUsers
	}

	id := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO white_labels (id, name, domain, api_key_hash, status, fee_percent, 
			profit_share_percent, plan_tier, max_users, max_api_calls, monthly_fee,
			custom_branding, features, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'pending', 20.0, 20.0, $5, $6, 100000, $7, TRUE, '["*"]', $8, NOW(), NOW())
	`, id, req.Name, req.Domain, apiKeyHash, req.PlanTier, maxUsers, req.MonthlyFee, adminID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create white label: " + err.Error()})
		return
	}

	logAudit(adminID.(string), "CREATE_WHITE_LABEL", "white_label", id,
		"Created white label: "+req.Name, c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{
		"id":        id,
		"name":      req.Name,
		"domain":    req.Domain,
		"api_key":   apiKey,
		"status":    "pending",
		"plan_tier": req.PlanTier,
		"max_users": maxUsers,
	})
}

func UpdateWhiteLabelHandler(c *gin.Context) {
	wlID := c.Param("id")
	adminID, _ := c.Get("admin_id")

	var req struct {
		Name               *string         `json:"name"`
		Domain             *string         `json:"domain"`
		PlanTier           *string         `json:"plan_tier"`
		MaxUsers           *int            `json:"max_users"`
		MonthlyFee         *float64        `json:"monthly_fee"`
		FeePercent         *float64        `json:"fee_percent"`
		ProfitSharePercent *float64        `json:"profit_share_percent"`
		BrandingConfig     json.RawMessage `json:"branding_config"`
		Features           []string        `json:"features"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update query
	query := "UPDATE white_labels SET updated_at = NOW()"
	args := []interface{}{}
	argNum := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argNum)
		args = append(args, *req.Name)
		argNum++
	}

	if req.Domain != nil {
		query += fmt.Sprintf(", domain = $%d", argNum)
		args = append(args, *req.Domain)
		argNum++
	}

	if req.PlanTier != nil {
		query += fmt.Sprintf(", plan_tier = $%d", argNum)
		args = append(args, *req.PlanTier)
		argNum++
	}

	if req.MaxUsers != nil {
		query += fmt.Sprintf(", max_users = $%d", argNum)
		args = append(args, *req.MaxUsers)
		argNum++
	}

	if req.MonthlyFee != nil {
		query += fmt.Sprintf(", monthly_fee = $%d", argNum)
		args = append(args, *req.MonthlyFee)
		argNum++
	}

	if req.FeePercent != nil {
		if *req.FeePercent < 0 || *req.FeePercent > 20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Fee percent must be between 0 and 20"})
			return
		}
		query += fmt.Sprintf(", fee_percent = $%d", argNum)
		args = append(args, *req.FeePercent)
		argNum++
	}

	if req.ProfitSharePercent != nil {
		if *req.ProfitSharePercent < 0 || *req.ProfitSharePercent > 50 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Profit share must be between 0 and 50"})
			return
		}
		query += fmt.Sprintf(", profit_share_percent = $%d", argNum)
		args = append(args, *req.ProfitSharePercent)
		argNum++
	}

	if req.BrandingConfig != nil {
		query += fmt.Sprintf(", branding_config = $%d", argNum)
		args = append(args, req.BrandingConfig)
		argNum++
	}

	if req.Features != nil {
		featuresJSON, _ := json.Marshal(req.Features)
		query += fmt.Sprintf(", features = $%d", argNum)
		args = append(args, string(featuresJSON))
		argNum++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argNum)
	args = append(args, wlID)

	_, err := db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(adminID.(string), "UPDATE_WHITE_LABEL", "white_label", wlID, "Updated white label", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "White label updated"})
}

func ApproveWhiteLabelHandler(c *gin.Context) {
	wlID := c.Param("id")
	adminID, _ := c.Get("admin_id")

	_, err := db.Exec(`
		UPDATE white_labels SET status = 'active', approved_by = $1, approved_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`, adminID, wlID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(adminID.(string), "APPROVE_WHITE_LABEL", "white_label", wlID, "Approved white label", c.ClientIP())

	// Create profit sharing record
	db.Exec(`
		INSERT INTO profit_sharing (white_label_id, super_admin_wallet, profit_percentage, is_active, auto_transfer, transfer_frequency, created_at)
		VALUES ($1, '0xSuperAdminWallet', 20.0, TRUE, TRUE, 'daily', NOW())
		ON CONFLICT (white_label_id) DO NOTHING
	`, wlID)

	c.JSON(http.StatusOK, gin.H{"message": "White label approved"})
}

func SuspendWhiteLabelHandler(c *gin.Context) {
	wlID := c.Param("id")
	adminID, _ := c.Get("admin_id")

	_, err := db.Exec(`
		UPDATE white_labels SET status = 'suspended', updated_at = NOW() WHERE id = $1
	`, wlID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(adminID.(string), "SUSPEND_WHITE_LABEL", "white_label", wlID, "Suspended white label", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "White label suspended"})
}

func RevokeWhiteLabelHandler(c *gin.Context) {
	wlID := c.Param("id")
	adminID, _ := c.Get("admin_id")

	_, err := db.Exec(`
		UPDATE white_labels SET status = 'revoked', updated_at = NOW() WHERE id = $1
	`, wlID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(adminID.(string), "REVOKE_WHITE_LABEL", "white_label", wlID, "Revoked white label", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "White label revoked"})
}

func DestroyWhiteLabelHandler(c *gin.Context) {
	wlID := c.Param("id")
	adminID, _ := c.Get("admin_id")

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Delete related data
	_, err = tx.Exec("DELETE FROM wl_api_keys WHERE white_label_id = $1", wlID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = tx.Exec("DELETE FROM white_label_admins WHERE white_label_id = $1", wlID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = tx.Exec("DELETE FROM webhooks WHERE white_label_id = $1", wlID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = tx.Exec("DELETE FROM profit_sharing WHERE white_label_id = $1", wlID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = tx.Exec("DELETE FROM white_labels WHERE id = $1", wlID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx.Commit()

	logAudit(adminID.(string), "DESTROY_WHITE_LABEL", "white_label", wlID, "Destroyed white label", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "White label destroyed"})
}

// ============================================================================
// PRODUCT HANDLERS
// ============================================================================

func GetProductsHandler(c *gin.Context) {
	wlID := c.Query("white_label_id")

	query := `SELECT id, white_label_id, name, type, description, status, fee_percent, 
		min_deposit, max_deposit, features, supported_chains, is_global, created_at 
		FROM products WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if wlID != "" {
		query += fmt.Sprintf(" AND (white_label_id = $%d OR is_global = TRUE)", argNum)
		args = append(args, wlID)
		argNum++
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var products []gin.H
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.WhiteLabelID, &p.Name, &p.Type, &p.Description,
			&p.Status, &p.FeePercent, &p.MinDeposit, &p.MaxDeposit, &p.Features,
			&p.SupportedChains, &p.IsGlobal, &p.CreatedAt); err != nil {
			continue
		}
		products = append(products, gin.H{
			"id":               p.ID,
			"white_label_id":   p.WhiteLabelID,
			"name":             p.Name,
			"type":             p.Type,
			"description":      p.Description,
			"status":           p.Status,
			"fee_percent":      p.FeePercent,
			"min_deposit":      p.MinDeposit,
			"max_deposit":      p.MaxDeposit,
			"features":         p.Features,
			"supported_chains": p.SupportedChains,
			"is_global":        p.IsGlobal,
			"created_at":       p.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"products": products})
}

func CreateProductHandler(c *gin.Context) {
	adminID, _ := c.Get("admin_id")

	var req struct {
		WhiteLabelID    *string  `json:"white_label_id"`
		Name            string   `json:"name" binding:"required"`
		Type            string   `json:"type" binding:"required"`
		Description     *string  `json:"description"`
		FeePercent      float64  `json:"fee_percent"`
		MinDeposit      float64  `json:"min_deposit"`
		MaxDeposit      *float64 `json:"max_deposit"`
		Features        []string `json:"features"`
		SupportedChains []string `json:"supported_chains"`
		IsGlobal        bool     `json:"is_global"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validTypes := map[string]bool{"trading": true, "wallet": true, "staking": true, "nft": true, "bridge": true, "defi": true}
	if !validTypes[req.Type] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product type"})
		return
	}

	id := uuid.New().String()
	featuresJSON, _ := json.Marshal(req.Features)
	chainsJSON, _ := json.Marshal(req.SupportedChains)

	_, err := db.Exec(`
		INSERT INTO products (id, white_label_id, name, type, description, status, fee_percent, 
			min_deposit, max_deposit, features, supported_chains, is_global, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'enabled', $6, $7, $8, $9, $10, $11, NOW(), NOW())
	`, id, req.WhiteLabelID, req.Name, req.Type, req.Description, req.FeePercent,
		req.MinDeposit, req.MaxDeposit, string(featuresJSON), string(chainsJSON), req.IsGlobal)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(adminID.(string), "CREATE_PRODUCT", "product", id, "Created product: "+req.Name, c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "type": req.Type})
}

func UpdateProductHandler(c *gin.Context) {
	productID := c.Param("id")
	adminID, _ := c.Get("admin_id")

	var req struct {
		Name            *string  `json:"name"`
		Description     *string  `json:"description"`
		Status          *string  `json:"status"`
		FeePercent      *float64 `json:"fee_percent"`
		MinDeposit      *float64 `json:"min_deposit"`
		MaxDeposit      *float64 `json:"max_deposit"`
		Features        []string `json:"features"`
		SupportedChains []string `json:"supported_chains"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := "UPDATE products SET updated_at = NOW()"
	args := []interface{}{}
	argNum := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argNum)
		args = append(args, *req.Name)
		argNum++
	}

	if req.Description != nil {
		query += fmt.Sprintf(", description = $%d", argNum)
		args = append(args, *req.Description)
		argNum++
	}

	if req.Status != nil {
		query += fmt.Sprintf(", status = $%d", argNum)
		args = append(args, *req.Status)
		argNum++
	}

	if req.FeePercent != nil {
		query += fmt.Sprintf(", fee_percent = $%d", argNum)
		args = append(args, *req.FeePercent)
		argNum++
	}

	if req.MinDeposit != nil {
		query += fmt.Sprintf(", min_deposit = $%d", argNum)
		args = append(args, *req.MinDeposit)
		argNum++
	}

	if req.MaxDeposit != nil {
		query += fmt.Sprintf(", max_deposit = $%d", argNum)
		args = append(args, *req.MaxDeposit)
		argNum++
	}

	if req.Features != nil {
		featuresJSON, _ := json.Marshal(req.Features)
		query += fmt.Sprintf(", features = $%d", argNum)
		args = append(args, string(featuresJSON))
		argNum++
	}

	if req.SupportedChains != nil {
		chainsJSON, _ := json.Marshal(req.SupportedChains)
		query += fmt.Sprintf(", supported_chains = $%d", argNum)
		args = append(args, string(chainsJSON))
		argNum++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argNum)
	args = append(args, productID)

	_, err := db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(adminID.(string), "UPDATE_PRODUCT", "product", productID, "Updated product", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "Product updated"})
}

// ============================================================================
// TRADING PAIRS HANDLERS
// ============================================================================

func GetTradingPairsHandler(c *gin.Context) {
	wlID := c.Query("white_label_id")

	query := `SELECT id, white_label_id, base_token, quote_token, pair_symbol, 
		min_trade_amount, max_trade_amount, maker_fee, taker_fee, status, chain_id,
		price_precision, quantity_precision, created_at
		FROM trading_pairs WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if wlID != "" {
		query += fmt.Sprintf(" AND white_label_id = $%d", argNum)
		args = append(args, wlID)
		argNum++
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var pairs []gin.H
	for rows.Next() {
		var p TradingPair
		if err := rows.Scan(&p.ID, &p.WhiteLabelID, &p.BaseToken, &p.QuoteToken,
			&p.PairSymbol, &p.MinTradeAmount, &p.MaxTradeAmount, &p.MakerFee,
			&p.TakerFee, &p.Status, &p.ChainID, &p.PricePrecision,
			&p.QuantityPrecision, &p.CreatedAt); err != nil {
			continue
		}
		pairs = append(pairs, gin.H{
			"id":                 p.ID,
			"white_label_id":     p.WhiteLabelID,
			"base_token":         p.BaseToken,
			"quote_token":        p.QuoteToken,
			"pair_symbol":        p.PairSymbol,
			"min_trade_amount":   p.MinTradeAmount,
			"max_trade_amount":   p.MaxTradeAmount,
			"maker_fee":          p.MakerFee,
			"taker_fee":          p.TakerFee,
			"status":             p.Status,
			"chain_id":           p.ChainID,
			"price_precision":    p.PricePrecision,
			"quantity_precision": p.QuantityPrecision,
			"created_at":         p.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"pairs": pairs})
}

func CreateTradingPairHandler(c *gin.Context) {
	adminID, _ := c.Get("admin_id")

	var req struct {
		WhiteLabelID      *string  `json:"white_label_id"`
		BaseToken         string   `json:"base_token" binding:"required"`
		QuoteToken        string   `json:"quote_token" binding:"required"`
		PairSymbol        string   `json:"pair_symbol" binding:"required"`
		MinTradeAmount    float64  `json:"min_trade_amount"`
		MaxTradeAmount    *float64 `json:"max_trade_amount"`
		MakerFee          float64  `json:"maker_fee"`
		TakerFee          float64  `json:"taker_fee"`
		ChainID           *string  `json:"chain_id"`
		PricePrecision    int      `json:"price_precision"`
		QuantityPrecision int      `json:"quantity_precision"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.MinTradeAmount == 0 {
		req.MinTradeAmount = 1
	}
	if req.MakerFee == 0 {
		req.MakerFee = 0.001
	}
	if req.TakerFee == 0 {
		req.TakerFee = 0.001
	}
	if req.PricePrecision == 0 {
		req.PricePrecision = 8
	}
	if req.QuantityPrecision == 0 {
		req.QuantityPrecision = 8
	}

	id := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO trading_pairs (id, white_label_id, base_token, quote_token, pair_symbol,
			min_trade_amount, max_trade_amount, maker_fee, taker_fee, status, chain_id,
			price_precision, quantity_precision, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active', $10, $11, $12, NOW(), NOW())
	`, id, req.WhiteLabelID, req.BaseToken, req.QuoteToken, req.PairSymbol,
		req.MinTradeAmount, req.MaxTradeAmount, req.MakerFee, req.TakerFee,
		req.ChainID, req.PricePrecision, req.QuantityPrecision)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(adminID.(string), "CREATE_TRADING_PAIR", "trading_pair", id,
		"Created trading pair: "+req.PairSymbol, c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{"id": id, "pair_symbol": req.PairSymbol})
}

// ============================================================================
// USER HANDLERS
// ============================================================================

func GetUsersHandler(c *gin.Context) {
	wlID := c.Query("white_label_id")
	status := c.Query("status")
	search := c.Query("search")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	query := `SELECT id, white_label_id, email, username, wallet_address, kyc_status, 
		status, risk_score, country, created_at, last_active 
		FROM users WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if wlID != "" {
		query += fmt.Sprintf(" AND white_label_id = $%d", argNum)
		args = append(args, wlID)
		argNum++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, status)
		argNum++
	}

	if search != "" {
		query += fmt.Sprintf(" AND (email ILIKE $%d OR username ILIKE $%d OR wallet_address ILIKE $%d)", argNum, argNum, argNum)
		args = append(args, "%"+search+"%")
		argNum++
	}

	// Get total count
	var total int
	countQuery := strings.Replace(query, "SELECT id, white_label_id, email, username, wallet_address, kyc_status, status, risk_score, country, created_at, last_active", "SELECT COUNT(*)", 1)
	db.QueryRow(countQuery, args...).Scan(&total)

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var users []gin.H
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.WhiteLabelID, &u.Email, &u.Username, &u.WalletAddress,
			&u.KYCStatus, &u.Status, &u.RiskScore, &u.Country, &u.CreatedAt, &u.LastActive); err != nil {
			continue
		}
		users = append(users, gin.H{
			"id":             u.ID,
			"white_label_id": u.WhiteLabelID,
			"email":          u.Email,
			"username":       u.Username,
			"wallet_address": u.WalletAddress,
			"kyc_status":     u.KYCStatus,
			"status":         u.Status,
			"risk_score":     u.RiskScore,
			"country":        u.Country,
			"created_at":     u.CreatedAt,
			"last_active":    u.LastActive,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ============================================================================
// STATISTICS HANDLERS
// ============================================================================

func GetDashboardStatsHandler(c *gin.Context) {
	var stats struct {
		ActiveWhiteLabels  int64   `json:"active_white_labels"`
		PendingWhiteLabels int64   `json:"pending_white_labels"`
		TotalUsers         int64   `json:"total_users"`
		ActiveUsers        int64   `json:"active_users"`
		Transactions24h    int64   `json:"transactions_24h"`
		TotalRevenue       float64 `json:"total_revenue"`
		TotalAdmins        int64   `json:"total_admins"`
		TotalProducts      int64   `json:"total_products"`
	}

	db.QueryRow("SELECT COUNT(*) FROM white_labels WHERE status = 'active'").Scan(&stats.ActiveWhiteLabels)
	db.QueryRow("SELECT COUNT(*) FROM white_labels WHERE status = 'pending'").Scan(&stats.PendingWhiteLabels)
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	db.QueryRow("SELECT COUNT(*) FROM users WHERE status = 'active'").Scan(&stats.ActiveUsers)
	db.QueryRow("SELECT COUNT(*) FROM transactions WHERE created_at > NOW() - INTERVAL '24 hours'").Scan(&stats.Transactions24h)
	db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM profit_transactions WHERE status = 'completed'").Scan(&stats.TotalRevenue)
	db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&stats.TotalAdmins)
	db.QueryRow("SELECT COUNT(*) FROM products").Scan(&stats.TotalProducts)

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func GetWhiteLabelStatsHandler(c *gin.Context) {
	wlID := c.Param("id")

	var stats struct {
		TotalUsers        int64   `json:"total_users"`
		ActiveUsers       int64   `json:"active_users"`
		TotalVolume       float64 `json:"total_volume"`
		TotalTransactions int64   `json:"total_transactions"`
		Revenue           float64 `json:"revenue"`
	}

	db.QueryRow("SELECT COUNT(*) FROM users WHERE white_label_id = $1", wlID).Scan(&stats.TotalUsers)
	db.QueryRow("SELECT COUNT(*) FROM users WHERE white_label_id = $1 AND status = 'active'", wlID).Scan(&stats.ActiveUsers)
	db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE white_label_id = $1 AND status = 'completed'", wlID).Scan(&stats.TotalVolume)
	db.QueryRow("SELECT COUNT(*) FROM transactions WHERE white_label_id = $1", wlID).Scan(&stats.TotalTransactions)
	db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM profit_transactions WHERE white_label_id = $1 AND status = 'completed'", wlID).Scan(&stats.Revenue)

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// ============================================================================
// API KEY HANDLERS
// ============================================================================

func GetAPIKeysHandler(c *gin.Context) {
	wlID := c.Param("id")

	rows, err := db.Query(`
		SELECT id, name, permissions, rate_limit_minute, rate_limit_day, 
			is_active, last_used, expires_at, created_at
		FROM wl_api_keys WHERE white_label_id = $1
		ORDER BY created_at DESC
	`, wlID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var keys []gin.H
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.Permissions, &k.RateLimitMin,
			&k.RateLimitDay, &k.IsActive, &k.LastUsed, &k.ExpiresAt, &k.CreatedAt); err != nil {
			continue
		}
		keys = append(keys, gin.H{
			"id":             k.ID,
			"name":           k.Name,
			"permissions":    k.Permissions,
			"rate_limit_min": k.RateLimitMin,
			"rate_limit_day": k.RateLimitDay,
			"is_active":      k.IsActive,
			"last_used":      k.LastUsed,
			"expires_at":     k.ExpiresAt,
			"created_at":     k.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": keys})
}

func CreateAPIKeyHandler(c *gin.Context) {
	wlID := c.Param("id")
	adminID, _ := c.Get("admin_id")

	var req struct {
		Name         string   `json:"name" binding:"required"`
		Permissions  []string `json:"permissions"`
		RateLimitMin int      `json:"rate_limit_minute"`
		RateLimitDay int      `json:"rate_limit_day"`
		ExpiresAt    *string  `json:"expires_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.RateLimitMin == 0 {
		req.RateLimitMin = 60
	}
	if req.RateLimitDay == 0 {
		req.RateLimitDay = 10000
	}

	// Generate API key
	apiKey := uuid.New().String() + "-" + uuid.New().String()
	apiKeyHash := HashAPIKey(apiKey)

	id := uuid.New().String()
	permissionsJSON, _ := json.Marshal(req.Permissions)

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	_, err := db.Exec(`
		INSERT INTO wl_api_keys (id, white_label_id, name, key_hash, permissions, 
			rate_limit_minute, rate_limit_day, is_active, expires_at, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, $8, $9, NOW())
	`, id, wlID, req.Name, apiKeyHash, string(permissionsJSON), req.RateLimitMin, req.RateLimitDay, expiresAt, adminID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(adminID.(string), "CREATE_API_KEY", "api_key", id, "Created API key: "+req.Name, c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{
		"id":         id,
		"api_key":    apiKey,
		"name":       req.Name,
		"expires_at": expiresAt,
	})
}

func RevokeAPIKeyHandler(c *gin.Context) {
	keyID := c.Param("key_id")
	adminID, _ := c.Get("admin_id")

	_, err := db.Exec("UPDATE wl_api_keys SET is_active = FALSE WHERE id = $1", keyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAudit(adminID.(string), "REVOKE_API_KEY", "api_key", keyID, "Revoked API key", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}

// ============================================================================
// BLOCKCHAIN HANDLERS
// ============================================================================

func GetBlockchainsHandler(c *gin.Context) {
	rows, err := db.Query(`
		SELECT id, name, symbol, chain_id, chain_type, native_token, decimals, 
			is_active, is_testnet, rpc_urls, explorer_urls
		FROM blockchains ORDER BY name
	`)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var chains []gin.H
	for rows.Next() {
		var b Blockchain
		var rpcJSON, explorerJSON sql.NullString
		if err := rows.Scan(&b.ID, &b.Name, &b.Symbol, &b.ChainID, &b.ChainType,
			&b.NativeToken, &b.Decimals, &b.IsActive, &b.IsTestnet,
			&rpcJSON, &explorerJSON); err != nil {
			continue
		}

		if rpcJSON.Valid {
			json.Unmarshal([]byte(rpcJSON.String), &b.RPCURLs)
		}
		if explorerJSON.Valid {
			json.Unmarshal([]byte(explorerJSON.String), &b.ExplorerURLs)
		}

		chains = append(chains, gin.H{
			"id":            b.ID,
			"name":          b.Name,
			"symbol":        b.Symbol,
			"chain_id":      b.ChainID,
			"chain_type":    b.ChainType,
			"native_token":  b.NativeToken,
			"decimals":      b.Decimals,
			"is_active":     b.IsActive,
			"is_testnet":    b.IsTestnet,
			"rpc_urls":      b.RPCURLs,
			"explorer_urls": b.ExplorerURLs,
		})
	}

	c.JSON(http.StatusOK, gin.H{"blockchains": chains})
}

// ============================================================================
// FEE STRUCTURE HANDLERS
// ============================================================================

func GetFeeStructuresHandler(c *gin.Context) {
	wlID := c.Query("white_label_id")

	query := `SELECT id, white_label_id, name, type, token_symbol, chain_id, 
		fee_percent, fee_fixed, is_active
		FROM fee_structures WHERE 1=1`
	args := []interface{}{}

	if wlID != "" {
		query += " AND (white_label_id = $1 OR white_label_id IS NULL)"
		args = append(args, wlID)
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var fees []gin.H
	for rows.Next() {
		var f FeeStructure
		if err := rows.Scan(&f.ID, &f.WhiteLabelID, &f.Name, &f.Type,
			&f.TokenSymbol, &f.ChainID, &f.FeePercent, &f.FeeFixed, &f.IsActive); err != nil {
			continue
		}
		fees = append(fees, gin.H{
			"id":             f.ID,
			"white_label_id": f.WhiteLabelID,
			"name":           f.Name,
			"type":           f.Type,
			"token_symbol":   f.TokenSymbol,
			"chain_id":       f.ChainID,
			"fee_percent":    f.FeePercent,
			"fee_fixed":      f.FeeFixed,
			"is_active":      f.IsActive,
		})
	}

	c.JSON(http.StatusOK, gin.H{"fees": fees})
}

// ============================================================================
// AUDIT LOGGING
// ============================================================================

func logAudit(adminID, action, entityType, entityID, details, ipAddress string) {
	db.Exec(`
		INSERT INTO audit_logs (admin_id, action, entity_type, entity_id, details, ip_address, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'success', NOW())
	`, adminID, action, entityType, entityID, details, ipAddress)
}

func GetAuditLogsHandler(c *gin.Context) {
	adminID := c.Query("admin_id")
	action := c.Query("action")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	query := `SELECT id, admin_id, action, entity_type, entity_id, details, ip_address, status, created_at
		FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if adminID != "" {
		query += fmt.Sprintf(" AND admin_id = $%d", argNum)
		args = append(args, adminID)
		argNum++
	}

	if action != "" {
		query += fmt.Sprintf(" AND action = $%d", argNum)
		args = append(args, action)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argNum)
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var logs []gin.H
	for rows.Next() {
		var log struct {
			ID         string    `json:"id"`
			AdminID    string    `json:"admin_id"`
			Action     string    `json:"action"`
			EntityType string    `json:"entity_type"`
			EntityID   string    `json:"entity_id"`
			Details    string    `json:"details"`
			IPAddress  string    `json:"ip_address"`
			Status     string    `json:"status"`
			CreatedAt  time.Time `json:"created_at"`
		}

		if err := rows.Scan(&log.ID, &log.AdminID, &log.Action, &log.EntityType,
			&log.EntityID, &log.Details, &log.IPAddress, &log.Status, &log.CreatedAt); err != nil {
			continue
		}

		logs = append(logs, gin.H{
			"id":          log.ID,
			"admin_id":    log.AdminID,
			"action":      log.Action,
			"entity_type": log.EntityType,
			"entity_id":   log.EntityID,
			"details":     log.Details,
			"ip_address":  log.IPAddress,
			"status":      log.Status,
			"created_at":  log.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"audit_logs": logs})
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

func HealthCheckHandler(c *gin.Context) {
	health := gin.H{
		"status":   "healthy",
		"database": "ok",
		"redis":    "ok",
	}

	if err := db.Ping(); err != nil {
		health["database"] = "error"
		health["status"] = "unhealthy"
	}

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		health["redis"] = "error"
		health["status"] = "unhealthy"
	}

	statusCode := http.StatusOK
	if health["status"] == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}

// ============================================================================
// ROUTES
// ============================================================================

func SetupRoutes(r *gin.Engine) {
	// Health check
	r.GET("/health", HealthCheckHandler)

	// Auth (no auth required)
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/login", LoginHandler)
	}

	// Protected routes
	api := r.Group("/api/v1")
	api.Use(AuthMiddleware())
	api.Use(RateLimiter())
	{
		// Admins
		admins := api.Group("/admins")
		{
			admins.GET("", RoleMiddleware(string(RoleSuperAdmin)), GetAdminsHandler)
			admins.POST("", RoleMiddleware(string(RoleSuperAdmin)), CreateAdminHandler)
			admins.PUT("/:id", UpdateAdminHandler)
			admins.POST("/:id/suspend", RoleMiddleware(string(RoleSuperAdmin)), SuspendAdminHandler)
		}

		// White Labels
		whiteLabels := api.Group("/white-labels")
		{
			whiteLabels.GET("", GetWhiteLabelsHandler)
			whiteLabels.GET("/:id", GetWhiteLabelHandler)
			whiteLabels.POST("", RoleMiddleware(string(RoleSuperAdmin), string(RoleMasterAdmin)), CreateWhiteLabelHandler)
			whiteLabels.PUT("/:id", RoleMiddleware(string(RoleSuperAdmin), string(RoleMasterAdmin)), UpdateWhiteLabelHandler)
			whiteLabels.POST("/:id/approve", RoleMiddleware(string(RoleSuperAdmin)), ApproveWhiteLabelHandler)
			whiteLabels.POST("/:id/suspend", RoleMiddleware(string(RoleSuperAdmin)), SuspendWhiteLabelHandler)
			whiteLabels.POST("/:id/revoke", RoleMiddleware(string(RoleSuperAdmin)), RevokeWhiteLabelHandler)
			whiteLabels.DELETE("/:id", RoleMiddleware(string(RoleSuperAdmin)), DestroyWhiteLabelHandler)
			whiteLabels.GET("/:id/stats", GetWhiteLabelStatsHandler)
			whiteLabels.GET("/:id/api-keys", GetAPIKeysHandler)
			whiteLabels.POST("/:id/api-keys", RoleMiddleware(string(RoleSuperAdmin), string(RoleMasterAdmin)), CreateAPIKeyHandler)
			whiteLabels.DELETE("/api-keys/:key_id", RoleMiddleware(string(RoleSuperAdmin), string(RoleMasterAdmin)), RevokeAPIKeyHandler)
		}

		// Products
		products := api.Group("/products")
		{
			products.GET("", GetProductsHandler)
			products.POST("", RoleMiddleware(string(RoleSuperAdmin), string(RoleMasterAdmin)), CreateProductHandler)
			products.PUT("/:id", RoleMiddleware(string(RoleSuperAdmin), string(RoleMasterAdmin)), UpdateProductHandler)
		}

		// Trading Pairs
		pairs := api.Group("/trading-pairs")
		{
			pairs.GET("", GetTradingPairsHandler)
			pairs.POST("", RoleMiddleware(string(RoleSuperAdmin), string(RoleMasterAdmin)), CreateTradingPairHandler)
		}

		// Users
		users := api.Group("/users")
		{
			users.GET("", RoleMiddleware(string(RoleSuperAdmin), string(RoleMasterAdmin), string(RoleWhiteLabelAdmin)), GetUsersHandler)
		}

		// Blockchains
		blockchains := api.Group("/blockchains")
		{
			blockchains.GET("", GetBlockchainsHandler)
		}

		// Fee Structures
		fees := api.Group("/fee-structures")
		{
			fees.GET("", GetFeeStructuresHandler)
		}

		// Audit Logs
		audit := api.Group("/audit-logs")
		{
			audit.GET("", RoleMiddleware(string(RoleSuperAdmin)), GetAuditLogsHandler)
		}

		// Dashboard Stats
		api.GET("/dashboard/stats", RoleMiddleware(string(RoleSuperAdmin), string(RoleMasterAdmin)), GetDashboardStatsHandler)
	}
}

// ============================================================================
// MAIN
// ============================================================================

func main() {

	// Initialize database
	if err := InitDatabase(config.DatabaseURL); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	if err := InitRedis(config.RedisURL); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redisClient.Close()

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// CORS
	origins := strings.Split(config.AllowedOrigins, ",")
	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Setup routes
	SetupRoutes(r)

	// TLS config
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Server
	srv := &http.Server{
		Addr:         ":" + config.ServerPort,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		TLSConfig:    tlsConfig,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("✅ White Label Service running on port %s", config.ServerPort)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
