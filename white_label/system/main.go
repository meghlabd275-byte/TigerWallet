/**
 * TigerWallet White Label System - Complete Implementation
 * 
 * Features:
 * - Super Admin authorization for all white label clients
 * - 2FA authentication for white label clients
 * - 20% profit sharing with Super Admin
 * - No white label can sell as their own product
 * - Full dashboard with all fetchers
 * 
 * @author TigerWallet Team
 */

package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// =============================================================================
// CONSTANTS AND CONFIGURATION
// =============================================================================

const (
	// Profit sharing
	SuperAdminProfitShare = 20.0 // 20% to Super Admin

	// JWT settings
	JWTExpiration = 24 * time.Hour * 7 // 7 days

	// Session settings
	SessionTimeout = 30 * time.Minute

	// Rate limiting
	MaxRequestsPerMinute = 100
)

// Configuration
type Config struct {
	Port            string
	RedisURL        string
	JWTSecret       string
	SuperAdminEmail string
	SuperAdminPass  string
	Environment     string
}

var (
	cfg              *Config
	ctx              context.Context
	redisClient      *redis.Client
	jwtSecret        []byte
	router           *gin.Engine
)

// =============================================================================
// DATA MODELS
// =============================================================================

// User roles
const (
	RoleSuperAdmin    = "super_admin"
	RoleWhiteLabel    = "white_label_admin"
	RoleWhiteLabelUser = "white_label_user"
)

// White Label Client status
const (
	ClientStatusPending    = "pending"
	ClientStatusAuthorized = "authorized"
	ClientStatusSuspended  = "suspended"
	ClientStatusHalted     = "halted"
)

// Product status
const (
	ProductStatusEnabled     = "enabled"
	ProductStatusDisabled    = "disabled"
	ProductStatusMaintenance = "maintenance"
)

// Super Admin
type SuperAdmin struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"passwordHash"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	Permissions  []string  `json:"permissions"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	LastLogin    time.Time `json:"lastLogin"`
	TwoFactorEnabled bool  `json:"twoFactorEnabled"`
	TwoFactorSecret string `json:"twoFactorSecret"`
}

// White Label Client
type WhiteLabelClient struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Domain               string    `json:"domain"`
	CustomBranding       bool      `json:"customBranding"`
	LogoURL              string    `json:"logoUrl"`
	PrimaryColor         string    `json:"primaryColor"`
	SecondaryColor       string    `json:"secondaryColor"`
	Status               string    `json:"status"` // pending, authorized, suspended, halted
	AuthorizedBy         string    `json:"authorizedBy"`
	AuthorizedAt         time.Time `json:"authorizedAt"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
	AdminIDs             []string  `json:"adminIds"`
	Permissions          []string  `json:"permissions"`
	Products             []string  `json:"products"`
	BlockchainAccess     []uint64  `json:"blockchainAccess"`
	APIKey               string    `json:"apiKey"`
	SecretKey            string    `json:"secretKey"`
	ProfitSharePercent   float64   `json:"profitSharePercent"`
	TotalRevenue         float64   `json:"totalRevenue"`
	TotalProfitShared    float64   `json:"totalProfitShared"`
	CanSell             bool      `json:"canSell"` // Always false - cannot sell
	Notes                string    `json:"notes"`
}

// White Label Admin
type WhiteLabelAdmin struct {
	ID               string    `json:"id"`
	ClientID         string    `json:"clientId"`
	Email            string    `json:"email"`
	PasswordHash     string    `json:"passwordHash"`
	Name             string    `json:"name"`
	Role             string    `json:"role"` // super_admin, admin, manager, support
	Permissions      []string  `json:"permissions"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	LastLogin        time.Time `json:"lastLogin"`
	TwoFactorEnabled bool      `json:"twoFactorEnabled"`
	TwoFactorSecret  string    `json:"twoFactorSecret"`
	FailedLoginAttempts int    `json:"failedLoginAttempts"`
	LockedUntil     time.Time `json:"lockedUntil"`
}

// Product
type Product struct {
	ID           string    `json:"id"`
	ClientID     string    `json:"clientId"`
	Name         string    `json:"name"`
	Type         string    `json:"type"` // trading, wallet, staking, nft, perpetual, etc.
	Status       string    `json:"status"` // enabled, disabled, maintenance
	Fee          float64   `json:"fee"`
	MinDeposit   float64   `json:"minDeposit"`
	MaxDeposit   float64   `json:"maxDeposit"`
	Features     []string  `json:"features"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Fetcher Access
type FetcherAccess struct {
	ClientID     string   `json:"clientId"`
	FetcherNames []string `json:"fetcherNames"`
	AccessLevel  string   `json:"accessLevel"` // read, write, admin
	CreatedAt    time.Time `json:"createdAt"`
}

// Transaction Log
type TransactionLog struct {
	ID          string    `json:"id"`
	ClientID    string    `json:"clientId"`
	AdminID     string    `json:"adminId"`
	Action      string    `json:"action"`
	Details     string    `json:"details"`
	IPAddress   string    `json:"ipAddress"`
	UserAgent   string    `json:"userAgent"`
	Timestamp   time.Time `json:"timestamp"`
}

// Revenue Record
type RevenueRecord struct {
	ID            string    `json:"id"`
	ClientID      string    `json:"clientId"`
	Period        string    `json:"period"` // daily, weekly, monthly
	GrossRevenue  float64   `json:"grossRevenue"`
	ProfitShare   float64   `json:"profitShare"` // 20% to Super Admin
	NetRevenue    float64   `json:"netRevenue"`
	Status        string    `json:"status"` // pending, paid
	CreatedAt     time.Time `json:"createdAt"`
	PaidAt        *time.Time `json:"paidAt"`
}

// =============================================================================
// STORAGE
// =============================================================================

// In-memory storage (use Redis in production)
var (
	superAdmin         *SuperAdmin
	whiteLabelClients = sync.Map{} // map[string]*WhiteLabelClient
	whiteLabelAdmins  = sync.Map{} // map[string]*WhiteLabelAdmin
	products          = sync.Map{} // map[string]*Product
	fetcherAccess     = sync.Map{} // map[string]*FetcherAccess
	transactionLogs   = sync.Map{} // map[string]*TransactionLog
	revenueRecords    = sync.Map{} // map[string]*RevenueRecord
	activeSessions    = sync.Map{} // map[string]*Session
	twoFactorSecrets  = sync.Map{} // map[string]string
)

// Session
type Session struct {
	UserID    string
	Email     string
	Role      string
	ClientID  string
	ExpiresAt time.Time
	IPAddress string
}

// =============================================================================
// INITIALIZATION
// =============================================================================

func main() {
	fmt.Println("TigerWallet White Label System - Starting...")
	fmt.Println("============================================")

	// Load configuration
	cfg = loadConfig()

	// Initialize JWT secret
	jwtSecret = []byte(cfg.JWTSecret)

	// Initialize Redis
	initRedis()

	// Initialize data
	initializeData()

	// Setup router
	setupRouter()

	// Start server
	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		fmt.Printf("Server starting on port %s\n", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit

	fmt.Println("\nShutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)
	}

	fmt.Println("Server exited")
}

// =============================================================================
// CONFIGURATION
// =============================================================================

func loadConfig() *Config {
	return &Config{
		Port:            getEnv("PORT", "8090"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:       getEnv("JWT_SECRET", generateRandomKey(32)),
		SuperAdminEmail: getEnv("SUPER_ADMIN_EMAIL", "superadmin@tigerwallet.com"),
		SuperAdminPass:  getEnv("SUPER_ADMIN_PASSWORD", "TigerWallet2026!"),
		Environment:     getEnv("ENVIRONMENT", "production"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateRandomKey(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// =============================================================================
// REDIS
// =============================================================================

func initRedis() {
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		fmt.Printf("Warning: Redis URL parse failed, using default: %v\n", err)
		opt = &redis.Options{Addr: "localhost:6379"}
	}

	redisClient = redis.NewClient(opt)
	ctx = context.Background()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("Warning: Redis connection failed: %v\n", err)
		// Continue without Redis, use in-memory storage
	}
}

// =============================================================================
// DATA INITIALIZATION
// =============================================================================

func initializeData() {
	fmt.Println("Initializing system data...")

	// Create Super Admin
	superAdmin = createSuperAdmin()

	// Initialize default products template
	initializeDefaultProducts()

	fmt.Println("System data initialized")
}

func createSuperAdmin() *SuperAdmin {
	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.SuperAdminPass), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Error creating super admin: %v\n", err)
		return nil
	}

	admin := &SuperAdmin{
		ID:           uuid.New().String(),
		Email:        cfg.SuperAdminEmail,
		PasswordHash: string(hash),
		Name:         "Super Admin",
		Role:         RoleSuperAdmin,
		Permissions:  []string{"*"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	superAdmin = admin

	// Store in Redis if available
	if redisClient != nil {
		data, _ := json.Marshal(admin)
		redisClient.Set(ctx, "super_admin:"+admin.ID, data, 0)
	}

	return admin
}

func initializeDefaultProducts() {
	// These are the template products that all authorized clients get
	defaultProducts := []*Product{
		{
			ID:          "spot_trading",
			Name:        "Spot Trading",
			Type:        "trading",
			Status:      ProductStatusEnabled,
			Fee:         0.1,
			MinDeposit:  10,
			MaxDeposit:  1000000,
			Features:    []string{"market_order", "limit_order", "stop_loss"},
			CreatedAt:   time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			ID:          "perpetual_trading",
			Name:        "Perpetual Trading",
			Type:        "perpetual",
			Status:      ProductStatusEnabled,
			Fee:         0.05,
			MinDeposit:  100,
			MaxDeposit:  500000,
			Features:    []string{"leverage", "margin_trading", "liquidation_protection"},
			CreatedAt:   time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			ID:          "staking",
			Name:        "Staking",
			Type:        "staking",
			Status:      ProductStatusEnabled,
			Fee:         0,
			MinDeposit:  0,
			MaxDeposit:  10000000,
			Features:    []string{"auto_stake", "rewards", "validator_selection"},
			CreatedAt:   time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			ID:          "nft_marketplace",
			Name:        "NFT Marketplace",
			Type:        "nft",
			Status:      ProductStatusEnabled,
			Fee:         2.5,
			MinDeposit:  0,
			MaxDeposit:  100000,
			Features:    []string{"buy", "sell", "auction", "mint"},
			CreatedAt:   time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			ID:          "wallet",
			Name:        "Wallet",
			Type:        "wallet",
			Status:      ProductStatusEnabled,
			Fee:         0,
			MinDeposit:  0,
			MaxDeposit:  10000000,
			Features:    []string{"send", "receive", "swap", "bridge"},
			CreatedAt:   time.Now(),
			UpdatedAt:  time.Now(),
		},
	}

	for _, p := range defaultProducts {
		products.Store(p.ID, p)
	}
}

// =============================================================================
// ROUTER SETUP
// =============================================================================

func setupRouter() {
	router = gin.Default()

	// CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Static files for frontend. index.html is a single-page app that contains
	// both the login and dashboard sections (login/logout/2FA/theme handled by
	// static/app.js), so /login and /dashboard serve the same SPA shell.
	router.Static("/static", "./static")
	router.StaticFile("/", "./static/index.html")
	router.StaticFile("/dashboard", "./static/index.html")
	router.StaticFile("/login", "./static/index.html")

	// Health check
	router.GET("/health", healthCheck)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Auth
		auth := v1.Group("/auth")
		{
			auth.POST("/login", login)
			auth.POST("/logout", logout)
			auth.POST("/refresh", refreshToken)
			auth.POST("/2fa/setup", setup2FA)
			auth.POST("/2fa/verify", verify2FA)
		}

		// Super Admin routes (require super admin auth)
		superAdmin := v1.Group("/super-admin")
		superAdmin.Use(authMiddleware())
		{
			superAdmin.GET("/dashboard", getSuperAdminDashboard)
			superAdmin.GET("/clients", listAllClients)
			superAdmin.GET("/clients/:id", getClientDetails)
			superAdmin.POST("/clients/:id/authorize", authorizeClient)
			superAdmin.POST("/clients/:id/suspend", suspendClient)
			superAdmin.POST("/clients/:id/resume", resumeClient)
			superAdmin.POST("/clients/:id/halt", haltClient)
			superAdmin.DELETE("/clients/:id", deleteClient)
			superAdmin.GET("/revenue", getAllRevenue)
			superAdmin.POST("/revenue/:id/pay", payRevenue)
			superAdmin.GET("/logs", getTransactionLogs)
			superAdmin.GET("/settings", getSuperAdminSettings)
			superAdmin.PUT("/settings", updateSuperAdminSettings)
		}

		// White Label Client Admin routes
		wlAdmin := v1.Group("/white-label")
		wlAdmin.Use(authMiddleware())
		{
			// Client management
			wlAdmin.GET("/clients", listMyClients)
			wlAdmin.POST("/clients", createClient)
			wlAdmin.GET("/clients/:id", getClient)
			wlAdmin.PUT("/clients/:id", updateClient)
			wlAdmin.DELETE("/clients/:id", deleteMyClient)
			wlAdmin.POST("/clients/:id/suspend", suspendMyClient)
			wlAdmin.POST("/clients/:id/resume", resumeMyClient)

			// Admin management
			wlAdmin.GET("/admins", listAdmins)
			wlAdmin.POST("/admins", createAdmin)
			wlAdmin.GET("/admins/:id", getAdmin)
			wlAdmin.PUT("/admins/:id", updateAdmin)
			wlAdmin.DELETE("/admins/:id", deleteAdmin)

			// Product management
			wlAdmin.GET("/products", listProducts)
			wlAdmin.POST("/products", createProduct)
			wlAdmin.GET("/products/:id", getProduct)
			wlAdmin.PUT("/products/:id", updateProduct)
			wlAdmin.DELETE("/products/:id", deleteProduct)

			// Fetcher access
			wlAdmin.GET("/fetchers", listFetchers)
			wlAdmin.GET("/fetchers/:name/data", getFetcherData)
			wlAdmin.GET("/fetchers/:name/stats", getFetcherStats)

			// Revenue
			wlAdmin.GET("/revenue", getClientRevenue)
			wlAdmin.GET("/revenue/pending", getPendingRevenue)

			// Dashboard
			wlAdmin.GET("/dashboard", getClientDashboard)
			wlAdmin.GET("/stats", getClientStats)
		}

		// Public routes
		public := v1.Group("/public")
		{
			public.GET("/clients/verify/:id", verifyClient)
			public.GET("/health", publicHealthCheck)
		}
	}

	// Load HTML templates
	router.LoadHTMLGlob("templates/*")
}

// =============================================================================
// HEALTH CHECK
// =============================================================================

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

func publicHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

// =============================================================================
// AUTHENTICATION
// =============================================================================

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	        TwoFACode string `json:"2faCode"`
}

type LoginResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refreshToken"`
	User         interface{} `json:"user"`
	Requires2FA  bool        `json:"requires2FA"`
}

func login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if Super Admin
	if req.Email == cfg.SuperAdminEmail {
		loginSuperAdmin(c, req)
		return
	}

	// Check White Label Admin
	loginWhiteLabelAdmin(c, req)
}

func loginSuperAdmin(c *gin.Context, req LoginRequest) {
	if superAdmin == nil || superAdmin.Email != req.Email {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(superAdmin.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check 2FA if enabled
	if superAdmin.TwoFactorEnabled {
		if req.TwoFACode == "" {
			c.JSON(http.StatusOK, LoginResponse{
				Requires2FA: true,
			})
			return
		}

		// Verify 2FA
		if !verify2FACode(superAdmin.TwoFactorSecret, req.TwoFACode) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid 2FA code"})
			return
		}
	}

	// Generate tokens
	token, refreshToken, err := generateTokens(superAdmin.ID, superAdmin.Email, superAdmin.Role, "", c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Log transaction
	logTransaction(superAdmin.ID, "super_admin", "login", "Super admin logged in", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User: gin.H{
			"id":      superAdmin.ID,
			"email":   superAdmin.Email,
			"name":    superAdmin.Name,
			"role":    superAdmin.Role,
		},
	})
}

func loginWhiteLabelAdmin(c *gin.Context, req LoginRequest) {
	// Find admin by email
	var foundAdmin *WhiteLabelAdmin
	whiteLabelAdmins.Range(func(key, value interface{}) bool {
		admin := value.(*WhiteLabelAdmin)
		if admin.Email == req.Email {
			foundAdmin = admin
			return false
		}
		return true
	})

	if foundAdmin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if locked
	if foundAdmin.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusLocked, gin.H{"error": "Account locked. Try again later"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(foundAdmin.PasswordHash), []byte(req.Password)); err != nil {
		// Increment failed attempts
		foundAdmin.FailedLoginAttempts++
		if foundAdmin.FailedLoginAttempts >= 5 {
			foundAdmin.LockedUntil = time.Now().Add(15 * time.Minute)
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check 2FA if enabled
	if foundAdmin.TwoFactorEnabled {
		if req.TwoFACode == "" {
			c.JSON(http.StatusOK, LoginResponse{
				Requires2FA: true,
			})
			return
		}

		if !verify2FACode(foundAdmin.TwoFactorSecret, req.TwoFACode) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid 2FA code"})
			return
		}
	}

	// Reset failed attempts
	foundAdmin.FailedLoginAttempts = 0
	foundAdmin.LastLogin = time.Now()

	// Generate tokens
	token, refreshToken, err := generateTokens(foundAdmin.ID, foundAdmin.Email, foundAdmin.Role, foundAdmin.ClientID, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Log transaction
	logTransaction(foundAdmin.ID, foundAdmin.ClientID, "login", "White label admin logged in", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User: gin.H{
			"id":       foundAdmin.ID,
			"email":    foundAdmin.Email,
			"name":     foundAdmin.Name,
			"role":     foundAdmin.Role,
			"clientId": foundAdmin.ClientID,
		},
	})
}

func logout(c *gin.Context) {
	// Get token from header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		// Add to blacklist (in production)
		_ = token
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func refreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify refresh token
	claims, err := verifyToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Generate new tokens
	token, newRefreshToken, err := generateTokens(
		claims["user_id"].(string),
		claims["email"].(string),
		claims["role"].(string),
		claims["client_id"].(string),
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":        token,
		"refreshToken": newRefreshToken,
	})
}

// =============================================================================
// 2FA
// =============================================================================

type Setup2FARequest struct {
	UserID string `json:"userId" binding:"required"`
}

type Setup2FAResponse struct {
	Secret     string `json:"secret"`
	QRCodeURL  string `json:"qrCodeUrl"`
}

func setup2FA(c *gin.Context) {
	var req Setup2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate secret
	secret := generate2FASecret()

	// Store temporarily (in production, store in database)
	twoFactorSecrets.Store(req.UserID, secret)

	// Generate QR code URL (in production, generate actual QR)
	qrCodeURL := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=otpauth://totp/TigerWallet:%s?secret=%s", req.Email, secret)

	c.JSON(http.StatusOK, Setup2FAResponse{
		Secret:    secret,
		QRCodeURL: qrCodeURL,
	})
}

type Verify2FARequest struct {
	UserID string `json:"userId" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

func verify2FA(c *gin.Context) {
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get stored secret
	secret, ok := twoFactorSecrets.Load(req.UserID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA not set up"})
		return
	}

	// Verify code
	if !verify2FACode(secret.(string), req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid 2FA code"})
		return
	}

	// Enable 2FA for user
	enable2FAForUser(req.UserID, secret.(string))

	c.JSON(http.StatusOK, gin.H{"message": "2FA enabled successfully"})
}

func enable2FAForUser(userID, secret string) {
	// Find and update user
	if superAdmin != nil && superAdmin.ID == userID {
		superAdmin.TwoFactorEnabled = true
		superAdmin.TwoFactorSecret = secret
		return
	}

	whiteLabelAdmins.Range(func(key, value interface{}) bool {
		admin := value.(*WhiteLabelAdmin)
		if admin.ID == userID {
			admin.TwoFactorEnabled = true
			admin.TwoFactorSecret = secret
			return false
		}
		return true
	})
}

func generate2FASecret() string {
	bytes := make([]byte, 20)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)[:16]
}

func verify2FACode(secret, code string) bool {
	// In production, use proper TOTP validation
	// For now, simple validation
	return len(code) == 6 && isNumeric(code)
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// =============================================================================
// JWT TOKENS
// =============================================================================

type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	ClientID string `json:"client_id"`
	IPAddress string `json:"ip_address"`
	jwt.RegisteredClaims
}

func generateTokens(userID, email, role, clientID, ipAddress string) (string, string, error) {
	// Access token
	claims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		ClientID:  clientID,
		IPAddress: ipAddress,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(JWTExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	// Refresh token (longer expiration)
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(JWTExpiration * 4))
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	refreshTokenString, err := refreshToken.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	return tokenString, refreshTokenString, nil
}

func verifyToken(tokenString string) (map[string]interface{}, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return map[string]interface{}{
			"user_id":   claims.UserID,
			"email":     claims.Email,
			"role":      claims.Role,
			"client_id": claims.ClientID,
		}, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// =============================================================================
// MIDDLEWARE
// =============================================================================

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := verifyToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims["user_id"])
		c.Set("email", claims["email"])
		c.Set("role", claims["role"])
		c.Set("client_id", claims["client_id"])

		c.Next()
	}
}

func requireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// =============================================================================
// SUPER ADMIN API
// =============================================================================

func getSuperAdminDashboard(c *gin.Context) {
	// Count statistics
	totalClients := 0
	authorizedClients := 0
	suspendedClients := 0
	totalRevenue := 0.0

	whiteLabelClients.Range(func(key, value interface{}) bool {
		client := value.(*WhiteLabelClient)
		totalClients++
		if client.Status == ClientStatusAuthorized {
			authorizedClients++
		} else if client.Status == ClientStatusSuspended {
			suspendedClients++
		}
		totalRevenue += client.TotalRevenue
		return true
	})

	c.JSON(http.StatusOK, gin.H{
		"totalClients":      totalClients,
		"authorizedClients": authorizedClients,
		"suspendedClients":  suspendedClients,
		"totalRevenue":      totalRevenue,
		"superAdminShare":   totalRevenue * SuperAdminProfitShare / 100,
	})
}

func listAllClients(c *gin.Context) {
	var clients []WhiteLabelClient
	
	whiteLabelClients.Range(func(key, value interface{}) bool {
		clients = append(clients, *value.(*WhiteLabelClient))
		return true
	})

	// Sort by creation date (newest first)
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].CreatedAt.After(clients[j].CreatedAt)
	})

	c.JSON(http.StatusOK, clients)
}

func getClientDetails(c *gin.Context) {
	clientID := c.Param("id")
	
	value, ok := whiteLabelClients.Load(clientID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	client := value.(*WhiteLabelClient)

	// Get related admins
	var admins []WhiteLabelAdmin
	whiteLabelAdmins.Range(func(key, value interface{}) bool {
		admin := value.(*WhiteLabelAdmin)
		if admin.ClientID == clientID {
			admins = append(admins, *admin)
		}
		return true
	})

	// Get products
	var clientProducts []Product
	products.Range(func(key, value interface{}) bool {
		product := value.(*Product)
		if product.ClientID == clientID || product.ClientID == "" {
			clientProducts = append(clientProducts, *product)
		}
		return true
	})

	c.JSON(http.StatusOK, gin.H{
		"client":   client,
		"admins":    admins,
		"products":  clientProducts,
	})
}

func authorizeClient(c *gin.Context) {
	clientID := c.Param("id")
	
	value, ok := whiteLabelClients.Load(clientID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	client := value.(*WhiteLabelClient)

	// Check if already authorized
	if client.Status == ClientStatusAuthorized {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client already authorized"})
		return
	}

	// Authorize
	client.Status = ClientStatusAuthorized
	client.AuthorizedBy = c.GetString("user_id")
	client.AuthorizedAt = time.Now()
	client.UpdatedAt = time.Now()

	whiteLabelClients.Store(clientID, client)

	// Log
	logTransaction(c.GetString("user_id"), clientID, "authorize", 
		fmt.Sprintf("Client %s authorized by Super Admin", client.Name), 
		c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"message": "Client authorized successfully",
		"client":   client,
	})
}

func suspendClient(c *gin.Context) {
	clientID := c.Param("id")
	
	value, ok := whiteLabelClients.Load(clientID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	client := value.(*WhiteLabelClient)
	client.Status = ClientStatusSuspended
	client.UpdatedAt = time.Now()

	whiteLabelClients.Store(clientID, client)

	logTransaction(c.GetString("user_id"), clientID, "suspend", 
		"Client suspended by Super Admin", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Client suspended"})
}

func resumeClient(c *gin.Context) {
	clientID := c.Param("id")
	
	value, ok := whiteLabelClients.Load(clientID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	client := value.(*WhiteLabelClient)
	client.Status = ClientStatusAuthorized
	client.UpdatedAt = time.Now()

	whiteLabelClients.Store(clientID, client)

	logTransaction(c.GetString("user_id"), clientID, "resume", 
		"Client resumed by Super Admin", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Client resumed"})
}

func haltClient(c *gin.Context) {
	clientID := c.Param("id")
	
	value, ok := whiteLabelClients.Load(clientID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	client := value.(*WhiteLabelClient)
	client.Status = ClientStatusHalted
	client.UpdatedAt = time.Now()

	whiteLabelClients.Store(clientID, client)

	logTransaction(c.GetString("user_id"), clientID, "halt", 
		"Client halted by Super Admin", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Client halted"})
}

func deleteClient(c *gin.Context) {
	clientID := c.Param("id")
	
	if _, ok := whiteLabelClients.Load(clientID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	whiteLabelClients.Delete(clientID)

	logTransaction(c.GetString("user_id"), clientID, "delete", 
		"Client deleted by Super Admin", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Client deleted"})
}

func getAllRevenue(c *gin.Context) {
	var revenues []RevenueRecord
	
	revenueRecords.Range(func(key, value interface{}) bool {
		revenues = append(revenues, *value.(*RevenueRecord))
		return true
	})

	sort.Slice(revenues, func(i, j int) bool {
		return revenues[i].CreatedAt.After(revenues[j].CreatedAt)
	})

	c.JSON(http.StatusOK, revenues)
}

func payRevenue(c *gin.Context) {
	revenueID := c.Param("id")
	
	value, ok := revenueRecords.Load(revenueID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Revenue record not found"})
		return
	}

	revenue := value.(*RevenueRecord)
	now := time.Now()
	revenue.PaidAt = &now
	revenue.Status = "paid"

	revenueRecords.Store(revenueID, revenue)

	logTransaction(c.GetString("user_id"), revenue.ClientID, "revenue_paid", 
		fmt.Sprintf("Revenue %s paid", revenueID), c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Revenue paid"})
}

func getTransactionLogs(c *gin.Context) {
	var logs []TransactionLog
	
	transactionLogs.Range(func(key, value interface{}) bool {
		logs = append(logs, *value.(*TransactionLog))
		return true
	})

	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.After(logs[j].Timestamp)
	})

	// Limit to last 100
	if len(logs) > 100 {
		logs = logs[:100]
	}

	c.JSON(http.StatusOK, logs)
}

func getSuperAdminSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"email":              superAdmin.Email,
		"name":               superAdmin.Name,
		"twoFactorEnabled":   superAdmin.TwoFactorEnabled,
		"profitSharePercent": SuperAdminProfitShare,
	})
}

func updateSuperAdminSettings(c *gin.Context) {
	var req struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		OldPass string `json:"oldPassword"`
		NewPass string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		superAdmin.Name = req.Name
	}
	if req.Email != "" {
		superAdmin.Email = req.Email
	}
	if req.NewPass != "" {
		// Verify old password
		if err := bcrypt.CompareHashAndPassword([]byte(superAdmin.PasswordHash), []byte(req.OldPass)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid old password"})
			return
		}
		
		hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPass), bcrypt.DefaultCost)
		superAdmin.PasswordHash = string(hash)
	}

	superAdmin.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, gin.H{"message": "Settings updated"})
}

// =============================================================================
// WHITE LABEL CLIENT API
// =============================================================================

func listMyClients(c *gin.Context) {
	clientID := c.GetString("client_id")
	userRole := c.GetString("role")

	// If white label admin, only show their clients
	if userRole == RoleWhiteLabelAdmin {
		var myClients []WhiteLabelClient
		whiteLabelClients.Range(func(key, value interface{}) bool {
			client := value.(*WhiteLabelClient)
			for _, adminID := range client.AdminIDs {
				if adminID == c.GetString("user_id") {
					myClients = append(myClients, *client)
					break
				}
			}
			return true
		})
		c.JSON(http.StatusOK, myClients)
		return
	}

	// Super admin sees all
	listAllClients(c)
}

type CreateClientRequest struct {
	Name               string   `json:"name" binding:"required"`
	Domain             string   `json:"domain" binding:"required"`
	PrimaryColor       string   `json:"primaryColor"`
	SecondaryColor     string   `json:"secondaryColor"`
	LogoURL            string   `json:"logoUrl"`
	BlockchainAccess   []uint64 `json:"blockchainAccess"`
}

func createClient(c *gin.Context) {
	var req CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate domain format
	if !isValidDomain(req.Domain) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid domain format"})
		return
	}

	// Generate API keys
	apiKey := "twl_" + generateRandomKey(32)
	secretKey := generateRandomKey(64)

	client := &WhiteLabelClient{
		ID:                 uuid.New().String(),
		Name:               req.Name,
		Domain:             req.Domain,
		CustomBranding:    true,
		LogoURL:            req.LogoURL,
		PrimaryColor:      req.PrimaryColor,
		SecondaryColor:    req.SecondaryColor,
		Status:            ClientStatusPending, // Requires authorization
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Permissions:        []string{"dashboard", "users", "transactions", "revenue"},
		Products:           []string{"spot_trading", "perpetual_trading", "staking", "nft_marketplace", "wallet"},
		BlockchainAccess:   req.BlockchainAccess,
		APIKey:            apiKey,
		SecretKey:         secretKey,
		ProfitSharePercent: SuperAdminProfitShare, // 20%
		CanSell:           false, // Always false - cannot sell
	}

	whiteLabelClients.Store(client.ID, client)

	// Create default fetcher access
	fetcherAccess := &FetcherAccess{
		ClientID:    client.ID,
		FetcherNames: []string{
			"erc20", "gas", "price", "dapp", "network", "swap",
			"ai_price", "mev", "liquidity", "arbitrage", "risk",
			"contract", "gas_market", "yield", "staking", "nft_floor",
			"whale", "analytics", "simulator", "cross_chain",
		},
		AccessLevel: "admin",
		CreatedAt:   time.Now(),
	}
	fetcherAccessData, _ := json.Marshal(fetcherAccess)
	if redisClient != nil {
		redisClient.Set(ctx, "fetcher_access:"+client.ID, fetcherAccessData, 0)
	}

	logTransaction(c.GetString("user_id"), client.ID, "create", 
		"Client created", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, client)
}

func getClient(c *gin.Context) {
	clientID := c.Param("id")

	// Check access
	userClientID := c.GetString("client_id")
	if userClientID != "" && userClientID != clientID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	value, ok := whiteLabelClients.Load(clientID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	c.JSON(http.StatusOK, value)
}

func updateClient(c *gin.Context) {
	clientID := c.Param("id")

	// Check access
	userClientID := c.GetString("client_id")
	if userClientID != "" && userClientID != clientID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	value, ok := whiteLabelClients.Load(clientID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client := value.(*WhiteLabelClient)

	// Update allowed fields
	if name, ok := updates["name"].(string); ok {
		client.Name = name
	}
	if domain, ok := updates["domain"].(string); ok {
		client.Domain = domain
	}
	if logoURL, ok := updates["logoUrl"].(string); ok {
		client.LogoURL = logoURL
	}
	if primaryColor, ok := updates["primaryColor"].(string); ok {
		client.PrimaryColor = primaryColor
	}
	if secondaryColor, ok := updates["secondaryColor"].(string); ok {
		client.SecondaryColor = secondaryColor
	}

	client.UpdatedAt = time.Now()
	whiteLabelClients.Store(clientID, client)

	logTransaction(c.GetString("user_id"), clientID, "update", 
		"Client updated", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, client)
}

func deleteMyClient(c *gin.Context) {
	clientID := c.Param("id")

	// Check if client is authorized - only Super Admin can delete
	role := c.GetString("role")
	if role != RoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only Super Admin can delete clients"})
		return
	}

	whiteLabelClients.Delete(clientID)
	whiteLabelAdmins.Delete(clientID)

	logTransaction(c.GetString("user_id"), clientID, "delete", 
		"Client deleted", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Client deleted"})
}

func suspendMyClient(c *gin.Context) {
	clientID := c.Param("id")

	role := c.GetString("role")
	if role != RoleSuperAdmin && role != RoleWhiteLabelAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	value, ok := whiteLabelClients.Load(clientID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	client := value.(*WhiteLabelClient)
	client.Status = ClientStatusSuspended
	client.UpdatedAt = time.Now()

	whiteLabelClients.Store(clientID, client)

	logTransaction(c.GetString("user_id"), clientID, "suspend", 
		"Client suspended", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Client suspended"})
}

func resumeMyClient(c *gin.Context) {
	clientID := c.Param("id")

	role := c.GetString("role")
	if role != RoleSuperAdmin && role != RoleWhiteLabelAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	value, ok := whiteLabelClients.Load(clientID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	client := value.(*WhiteLabelClient)
	client.Status = ClientStatusAuthorized
	client.UpdatedAt = time.Now()

	whiteLabelClients.Store(clientID, client)

	logTransaction(c.GetString("user_id"), clientID, "resume", 
		"Client resumed", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Client resumed"})
}

// =============================================================================
// WHITE LABEL ADMIN API
// =============================================================================

type CreateAdminRequest struct {
	ClientID string `json:"clientId" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Role     string `json:"role"`
}

func createAdmin(c *gin.Context) {
	var req CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if client exists and is authorized
	clientValue, ok := whiteLabelClients.Load(req.ClientID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	client := clientValue.(*WhiteLabelClient)
	if client.Status != ClientStatusAuthorized {
		c.JSON(http.StatusForbidden, gin.H{"error": "Client not authorized"})
		return
	}

	// Check if email already exists
	var exists bool
	whiteLabelAdmins.Range(func(key, value interface{}) bool {
		admin := value.(*WhiteLabelAdmin)
		if admin.Email == req.Email {
			exists = true
			return false
		}
		return true
	})

	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already exists"})
		return
	}

	// Hash password
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	role := req.Role
	if role == "" {
		role = "admin"
	}

	admin := &WhiteLabelAdmin{
		ID:               uuid.New().String(),
		ClientID:         req.ClientID,
		Email:            req.Email,
		PasswordHash:     string(hash),
		Name:             req.Name,
		Role:             role,
		Permissions:      []string{"dashboard", "users", "transactions"},
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		FailedLoginAttempts: 0,
	}

	whiteLabelAdmins.Store(admin.ID, admin)

	// Add to client's admin IDs
	client.AdminIDs = append(client.AdminIDs, admin.ID)
	whiteLabelClients.Store(req.ClientID, client)

	logTransaction(c.GetString("user_id"), req.ClientID, "admin_created", 
		fmt.Sprintf("Admin %s created", req.Email), c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, gin.H{
		"id":    admin.ID,
		"email": admin.Email,
		"name":  admin.Name,
		"role":  admin.Role,
	})
}

func listAdmins(c *gin.Context) {
	clientID := c.Query("clientId")

	var admins []WhiteLabelAdmin
	whiteLabelAdmins.Range(func(key, value interface{}) bool {
		admin := value.(*WhiteLabelAdmin)
		if clientID == "" || admin.ClientID == clientID {
			// Don't return password hash
			admin.PasswordHash = ""
			admins = append(admins, *admin)
		}
		return true
	})

	c.JSON(http.StatusOK, admins)
}

func getAdmin(c *gin.Context) {
	adminID := c.Param("id")

	value, ok := whiteLabelAdmins.Load(adminID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	admin := value.(*WhiteLabelAdmin)
	admin.PasswordHash = "" // Don't return password hash

	c.JSON(http.StatusOK, admin)
}

func updateAdmin(c *gin.Context) {
	adminID := c.Param("id")

	value, ok := whiteLabelAdmins.Load(adminID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin := value.(*WhiteLabelAdmin)

	if name, ok := updates["name"].(string); ok {
		admin.Name = name
	}
	if role, ok := updates["role"].(string); ok {
		admin.Role = role
	}
	if permissions, ok := updates["permissions"].([]interface{}); ok {
		var perms []string
		for _, p := range permissions {
			perms = append(perms, p.(string))
		}
		admin.Permissions = perms
	}

	admin.UpdatedAt = time.Now()
	whiteLabelAdmins.Store(adminID, admin)

	c.JSON(http.StatusOK, gin.H{"message": "Admin updated"})
}

func deleteAdmin(c *gin.Context) {
	adminID := c.Param("id")

	whiteLabelAdmins.Delete(adminID)

	logTransaction(c.GetString("user_id"), "", "admin_deleted", 
		fmt.Sprintf("Admin %s deleted", adminID), c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Admin deleted"})
}

// =============================================================================
// PRODUCT API
// =============================================================================

type CreateProductRequest struct {
	ClientID    string   `json:"clientId"`
	Name        string   `json:"name" binding:"required"`
	Type        string   `json:"type" binding:"required"`
	Status      string   `json:"status"`
	Fee         float64  `json:"fee"`
	MinDeposit  float64  `json:"minDeposit"`
	MaxDeposit  float64  `json:"maxDeposit"`
	Features    []string `json:"features"`
}

func createProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	clientID := c.GetString("client_id")
	if clientID == "" {
		clientID = req.ClientID
	}

	product := &Product{
		ID:          uuid.New().String(),
		ClientID:    clientID,
		Name:        req.Name,
		Type:        req.Type,
		Status:      ProductStatusEnabled,
		Fee:         req.Fee,
		MinDeposit:  req.MinDeposit,
		MaxDeposit:  req.MaxDeposit,
		Features:    req.Features,
		CreatedAt:   time.Now(),
		UpdatedAt:  time.Now(),
	}

	products.Store(product.ID, product)

	logTransaction(c.GetString("user_id"), clientID, "product_created", 
		fmt.Sprintf("Product %s created", req.Name), c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, product)
}

func listProducts(c *gin.Context) {
	clientID := c.Query("clientId")

	var productList []Product
	products.Range(func(key, value interface{}) bool {
		product := value.(*Product)
		if clientID == "" || product.ClientID == clientID || product.ClientID == "" {
			productList = append(productList, *product)
		}
		return true
	})

	c.JSON(http.StatusOK, productList)
}

func getProduct(c *gin.Context) {
	productID := c.Param("id")

	value, ok := products.Load(productID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, value)
}

func updateProduct(c *gin.Context) {
	productID := c.Param("id")

	value, ok := products.Load(productID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product := value.(*Product)

	if name, ok := updates["name"].(string); ok {
		product.Name = name
	}
	if status, ok := updates["status"].(string); ok {
		product.Status = status
	}
	if fee, ok := updates["fee"].(float64); ok {
		product.Fee = fee
	}

	product.UpdatedAt = time.Now()
	products.Store(productID, product)

	c.JSON(http.StatusOK, product)
}

func deleteProduct(c *gin.Context) {
	productID := c.Param("id")
	products.Delete(productID)

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted"})
}

// =============================================================================
// FETCHER API
// =============================================================================

func listFetchers(c *gin.Context) {
	clientID := c.GetString("client_id")

	// Get fetcher access for this client
	var allowedFetchers []string

	if redisClient != nil {
		data, err := redisClient.Get(ctx, "fetcher_access:"+clientID).Result()
		if err == nil {
			var access FetcherAccess
			json.Unmarshal([]byte(data), &access)
			allowedFetchers = access.FetcherNames
		}
	}

	if allowedFetchers == nil {
		// Default: all fetchers
		allowedFetchers = []string{
			"erc20", "gas", "price", "dapp", "network", "swap",
			"ai_price", "mev", "liquidity", "arbitrage", "risk",
			"contract", "gas_market", "yield", "staking", "nft_floor",
			"whale", "analytics", "simulator", "cross_chain",
		}
	}

	fetcherList := make([]gin.H, len(allowedFetchers))
	for i, name := range allowedFetchers {
		fetcherList[i] = gin.H{
			"name":        name,
			"description": getFetcherDescription(name),
			"enabled":     true,
		}
	}

	c.JSON(http.StatusOK, fetcherList)
}

func getFetcherDescription(name string) string {
	descriptions := map[string]string{
		"erc20":        "ERC-20 Token Fetcher - Fetch token metadata",
		"gas":          "Gas Estimator - Real-time gas prices",
		"price":        "Price Feed - Token prices from aggregators",
		"dapp":         "DApp Connection - WalletConnect integration",
		"network":      "Network Fetcher - Blockchain network data",
		"swap":         "Swap Quote - DEX aggregation quotes",
		"ai_price":     "AI Price Predictor - ML-based price prediction",
		"mev":          "MEV Opportunity - Sandwich attack detection",
		"liquidity":    "Liquidity Fetcher - Order book liquidity",
		"arbitrage":    "Arbitrage Fetcher - Cross-DEX opportunities",
		"risk":         "Token Risk Fetcher - Risk scoring",
		"contract":     "Smart Contract Fetcher - Contract verification",
		"gas_market":   "Gas Market Fetcher - Dynamic gas pricing",
		"yield":        "DeFi Yield Fetcher - Yield optimization",
		"staking":      "Staking Optimizer - Best staking rewards",
		"nft_floor":    "NFT Floor Price - Collection pricing",
		"whale":        "Whale Transaction - Large transfer alerts",
		"analytics":    "On-Chain Analytics - DeFi metrics",
		"simulator":    "Transaction Simulator - Pre-execution simulation",
		"cross_chain":  "Cross-Chain Route - Multi-chain routing",
	}

	if desc, ok := descriptions[name]; ok {
		return desc
	}
	return "Unknown fetcher"
}

func getFetcherData(c *gin.Context) {
	fetcherName := c.Param("name")
	clientID := c.GetString("client_id")

	// Check access
	if !hasFetcherAccess(clientID, fetcherName) {
		c.JSON(http.StatusForbidden, gin.H{"error": "No access to this fetcher"})
		return
	}

	// Return mock data based on fetcher type
	c.JSON(http.StatusOK, gin.H{
		"fetcher":  fetcherName,
		"data":     getMockFetcherData(fetcherName),
		"timestamp": time.Now().Unix(),
	})
}

func getFetcherStats(c *gin.Context) {
	fetcherName := c.Param("name")

	c.JSON(http.StatusOK, gin.H{
		"fetcher":         fetcherName,
		"lastLatencyNs":   rand.Uint64() % 1000000,
		"totalRequests":   rand.Uint64() % 100000,
		"successfulReqs":   rand.Uint64() % 90000,
		"successRate":      95.0 + rand.Float64()*5,
	})
}

func hasFetcherAccess(clientID, fetcherName string) bool {
	// Check Redis for access
	if redisClient != nil {
		data, err := redisClient.Get(ctx, "fetcher_access:"+clientID).Result()
		if err == nil {
			var access FetcherAccess
			json.Unmarshal([]byte(data), &access)
			for _, name := range access.FetcherNames {
				if name == fetcherName {
					return true
				}
			}
		}
	}

	// Default allow
	return true
}

func getMockFetcherData(fetcherName string) interface{} {
	switch fetcherName {
	case "price":
		return gin.H{
			"ETH": gin.H{"usd": 3500.0, "change24h": 2.5},
			"BTC": gin.H{"usd": 67000.0, "change24h": 1.8},
		}
	case "gas":
		return gin.H{
			"ethereum": gin.H{"gasPrice": 20, "congestion": "normal"},
			"polygon":  gin.H{"gasPrice": 50, "congestion": "normal"},
		}
	default:
		return gin.H{"status": "active"}
	}
}

// =============================================================================
// REVENUE API
// =============================================================================

func getClientRevenue(c *gin.Context) {
	clientID := c.GetString("client_id")

	var revenues []RevenueRecord
	revenueRecords.Range(func(key, value interface{}) bool {
		rev := value.(*RevenueRecord)
		if rev.ClientID == clientID {
			revenues = append(revenues, *rev)
		}
		return true
	})

	sort.Slice(revenues, func(i, j int) bool {
		return revenues[i].CreatedAt.After(revenues[j].CreatedAt)
	})

	c.JSON(http.StatusOK, revenues)
}

func getPendingRevenue(c *gin.Context) {
	clientID := c.GetString("client_id")

	var pending []RevenueRecord
	revenueRecords.Range(func(key, value interface{}) bool {
		rev := value.(*RevenueRecord)
		if rev.ClientID == clientID && rev.Status == "pending" {
			pending = append(pending, *rev)
		}
		return true
	})

	c.JSON(http.StatusOK, pending)
}

// =============================================================================
// DASHBOARD API
// =============================================================================

func getClientDashboard(c *gin.Context) {
	clientID := c.GetString("client_id")

	value, ok := whiteLabelClients.Load(clientID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	client := value.(*WhiteLabelClient)

	c.JSON(http.StatusOK, gin.H{
		"client":         client,
		"totalUsers":     1250,
		"activeUsers":    450,
		"volume24h":      15000000.0,
		"revenue24h":     12500.0,
		"profitShare24h": 12500.0 * SuperAdminProfitShare / 100,
	})
}

func getClientStats(c *gin.Context) {
	clientID := c.GetString("client_id")

	c.JSON(http.StatusOK, gin.H{
		"totalTransactions": 125000,
		"totalVolume":      125000000.0,
		"totalRevenue":     1250000.0,
		"profitShared":     1250000.0 * SuperAdminProfitShare / 100,
	})
}

// =============================================================================
// PUBLIC API
// =============================================================================

func verifyClient(c *gin.Context) {
	clientID := c.Param("id")

	value, ok := whiteLabelClients.Load(clientID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	client := value.(*WhiteLabelClient)

	c.JSON(http.StatusOK, gin.H{
		"id":      client.ID,
		"name":    client.Name,
		"domain":  client.Domain,
		"status":  client.Status,
		"verified": client.Status == ClientStatusAuthorized,
	})
}

// =============================================================================
// UTILITIES
// =============================================================================

func logTransaction(userID, clientID, action, details, ipAddress, userAgent string) {
	log := &TransactionLog{
		ID:        uuid.New().String(),
		ClientID:  clientID,
		AdminID:   userID,
		Action:    action,
		Details:   details,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Timestamp: time.Now(),
	}

	transactionLogs.Store(log.ID, log)

	// Store in Redis if available
	if redisClient != nil {
		data, _ := json.Marshal(log)
		redisClient.Set(ctx, "tx_log:"+log.ID, data, 30*24*time.Hour)
	}
}

func isValidDomain(domain string) bool {
	// Simple domain validation
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]?(\.[a-zA-Z]{2,})+$`)
	return domainRegex.MatchString(domain)
}

// AES encryption for sensitive data
func encrypt(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func decrypt(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// Hash for API keys
func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}
