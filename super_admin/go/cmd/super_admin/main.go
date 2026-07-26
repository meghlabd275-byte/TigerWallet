package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port            string
	JWTSecret       string
	JWTExpiration   time.Duration
	DatabaseURL     string
	RedisURL        string
	Environment     string
}

func LoadConfig() *Config {
	return &Config{
		Port:          getEnv("PORT", "8001"),
		JWTSecret:     getEnv("JWT_SECRET", "tigerwallet-super-secret-key"),
		JWTExpiration: 24 * time.Hour * 7, // 7 days
		DatabaseURL:   getEnv("DATABASE_URL", "postgresql://localhost:5432/tigerwallet"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),
		Environment:   getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Models
// ============================================================================

type UserRole string

const (
	RoleSuperAdmin   UserRole = "super_admin"
	RoleAdmin        UserRole = "admin"
	RoleWhiteLabel   UserRole = "white_label"
	RoleUser         UserRole = "user"
	RoleBroker       UserRole = "broker"
	RoleInstitution  UserRole = "institution"
)

type Admin struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null"`
	Username     string    `json:"username" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	Role         UserRole `json:"role" gorm:"not null"`
	Permissions  []string `json:"permissions" gorm:"-"`
	IsActive     bool     `json:"isActive" gorm:"default:true"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	LastLoginAt  *time.Time `json:"lastLoginAt"`
}

type WhiteLabelClient struct {
	ID                string    `json:"id" gorm:"primaryKey"`
	Name              string    `json:"name" gorm:"not null"`
	Domain            string    `json:"domain" gorm:"uniqueIndex"`
	CustomBranding    bool      `json:"customBranding" gorm:"default:true"`
	Status            string    `json:"status" gorm:"default:active"` // active, paused, halted
	FeePercentage     float64   `json:"feePercentage" gorm:"default:0.1"`
	AllowedChains     []string  `json:"allowedChains"`
	AllowedFeatures   []string  `json:"allowedFeatures"`
	APIKey            string    `json:"apiKey" gorm:"-"`
	APIKeyHash        string    `json:"-"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type WhiteLabelWallet struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	WhiteLabelID    string    `json:"whiteLabelId" gorm:"index;not null"`
	OwnerID         string    `json:"ownerId" gorm:"index;not null"`
	WalletType      string    `json:"walletType"` // user, master
	SeedEncrypted   string    `json:"-"`
	PublicKey       string    `json:"publicKey"`
	Chains          []string  `json:"chains"`
	IsActive        bool      `json:"isActive" gorm:"default:true"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type WhiteLabelExchange struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	WhiteLabelID    string    `json:"whiteLabelId" gorm:"index;not null"`
	Name            string    `json:"name"`
	Domain          string    `json:"domain"`
	ExchangeType    string    `json:"exchangeType"` // CEX, DEX, CEX_DEX
	Status          string    `json:"status" gorm:"default:active"`
	Config          string    `json:"config" gorm:"type:text"` // JSON config
	FeeStructure    string    `json:"feeStructure" gorm:"type:text"` // JSON
	Features        []string  `json:"features"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type BlockchainNetwork struct {
	ID           string  `json:"id" gorm:"primaryKey"`
	ChainID      int64   `json:"chainId" gorm:"uniqueIndex"`
	Name         string  `json:"name" gorm:"not null"`
	Symbol       string  `json:"symbol" gorm:"not null"`
	Explorer     string  `json:"explorer"`
	RPCURL       string  `json:"rpcUrl"`
	IsEnabled    bool    `json:"isEnabled" gorm:"default:true"`
	IsEVM        bool    `json:"isEvm" gorm:"default:true"`
	NetworkType  string  `json:"networkType"` // mainnet, testnet
	AddedBy      string  `json:"addedBy"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Token struct {
	ID              string  `json:"id" gorm:"primaryKey"`
	Address         string  `json:"address" gorm:"uniqueIndex"`
	ChainID         int64   `json:"chainId" gorm:"index"`
	Name            string  `json:"name" gorm:"not null"`
	Symbol          string  `json:"symbol" gorm:"not null"`
	Decimals        int     `json:"decimals"`
	IsEnabled       bool    `json:"isEnabled" gorm:"default:true"`
	IsPopular       bool    `json:"isPopular" gorm:"default:false"`
	AddedBy         string  `json:"addedBy"`
	CreatedAt       time.Time `json:"createdAt"`
}

type AuditLog struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	AdminID     string    `json:"adminId" gorm:"index"`
	AdminEmail  string    `json:"adminEmail"`
	Action      string    `json:"action" gorm:"index"`
	Resource    string    `json:"resource" gorm:"index"`
	ResourceID  string    `json:"resourceId"`
	Details     string    `json:"details" gorm:"type:text"`
	IPAddress   string    `json:"ipAddress"`
	UserAgent   string    `json:"userAgent"`
	CreatedAt   time.Time `json:"createdAt" gorm:"index"`
}

// ============================================================================
// Services
// ============================================================================

type SuperAdminService struct {
	config       *Config
	admins       map[string]*Admin
	whiteLabels  map[string]*WhiteLabelClient
	blockchains  map[string]*BlockchainNetwork
	tokens       map[string]*Token
	auditLogs    []AuditLog
}

func NewSuperAdminService(config *Config) *SuperAdminService {
	return &SuperAdminService{
		config:      config,
		admins:      make(map[string]*Admin),
		whiteLabels: make(map[string]*WhiteLabelClient),
		blockchains: make(map[string]*BlockchainNetwork),
		tokens:      make(map[string]*Token),
		auditLogs:   make([]AuditLog, 0),
	}
}

// ============================================================================
// Authentication
// ============================================================================

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	Admin     *Admin    `json:"admin"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type Claims struct {
	AdminID string   `json:"adminId"`
	Email   string   `json:"email"`
	Role    UserRole `json:"role"`
	jwt.RegisteredClaims
}

func (s *SuperAdminService) GenerateToken(admin *Admin) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.config.JWTExpiration)
	
	claims := Claims{
		AdminID: admin.ID,
		Email:   admin.Email,
		Role:    admin.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet",
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *SuperAdminService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, fmt.Errorf("invalid token")
}

func (s *SuperAdminService) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find admin
	var admin *Admin
	for _, a := range s.admins {
		if a.Email == req.Email {
			admin = a
			break
		}
	}

	if admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if !admin.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is disabled"})
		return
	}

	// Generate token
	token, expiresAt, err := s.GenerateToken(admin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Update last login
	now := time.Now()
	admin.LastLoginAt = &now

	// Log audit
	s.logAudit(admin.ID, admin.Email, "LOGIN", "admin", admin.ID, "")

	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		Admin:     admin,
		ExpiresAt: expiresAt,
	})
}

// ============================================================================
// Admin Management
// ============================================================================

type CreateAdminRequest struct {
	Email    string   `json:"email" binding:"required,email"`
	Username string   `json:"username" binding:"required"`
	Password string   `json:"password" binding:"required,min=8"`
	Role     UserRole `json:"role" binding:"required"`
}

func (s *SuperAdminService) CreateAdmin(c *gin.Context) {
	claims, err := s.GetAdminClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if claims.Role != RoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email already exists
	for _, admin := range s.admins {
		if admin.Email == req.Email {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
			return
		}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	admin := &Admin{
		ID:           uuid.New().String(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Role:         req.Role,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	s.admins[admin.ID] = admin

	// Log audit
	s.logAudit(claims.AdminID, claims.Email, "CREATE_ADMIN", "admin", admin.ID, fmt.Sprintf("Created admin: %s", req.Email))

	c.JSON(http.StatusCreated, admin)
}

func (s *SuperAdminService) ListAdmins(c *gin.Context) {
	claims, err := s.GetAdminClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if claims.Role != RoleSuperAdmin && claims.Role != RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	admins := make([]*Admin, 0, len(s.admins))
	for _, admin := range s.admins {
		// Don't expose password hash
		adminCopy := *admin
		admins = append(admins, &adminCopy)
	}

	c.JSON(http.StatusOK, gin.H{"admins": admins})
}

// ============================================================================
// White Label Management
// ============================================================================

type CreateWhiteLabelRequest struct {
	Name            string   `json:"name" binding:"required"`
	Domain          string   `json:"domain" binding:"required"`
	FeePercentage   float64  `json:"feePercentage"`
	AllowedChains   []string `json:"allowedChains"`
	AllowedFeatures []string `json:"allowedFeatures"`
}

func (s *SuperAdminService) CreateWhiteLabel(c *gin.Context) {
	claims, err := s.GetAdminClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if claims.Role != RoleSuperAdmin && claims.Role != RoleAdmin && claims.Role != RoleWhiteLabel {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req CreateWhiteLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate API key
	apiKey := uuid.New().String()
	apiKeyHash := sha256.Sum256([]byte(apiKey))

	whiteLabel := &WhiteLabelClient{
		ID:                uuid.New().String(),
		Name:              req.Name,
		Domain:            req.Domain,
		Status:            "active",
		FeePercentage:     req.FeePercentage,
		AllowedChains:     req.AllowedChains,
		AllowedFeatures:   req.AllowedFeatures,
		APIKey:            apiKey,
		APIKeyHash:        hex.EncodeToString(apiKeyHash[:]),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	s.whiteLabels[whiteLabel.ID] = whiteLabel

	// Log audit
	s.logAudit(claims.AdminID, claims.Email, "CREATE_WHITE_LABEL", "white_label", whiteLabel.ID, fmt.Sprintf("Created white label: %s", req.Name))

	c.JSON(http.StatusCreated, gin.H{
		"whiteLabel": whiteLabel,
		"apiKey":     apiKey,
	})
}

func (s *SuperAdminService) ListWhiteLabels(c *gin.Context) {
	claims, err := s.GetAdminClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if claims.Role != RoleSuperAdmin && claims.Role != RoleAdmin && claims.Role != RoleWhiteLabel {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	whiteLabels := make([]*WhiteLabelClient, 0, len(s.whiteLabels))
	for _, wl := range s.whiteLabels {
		// Don't expose API key hash
		wlCopy := *wl
		wlCopy.APIKeyHash = ""
		whiteLabels = append(whiteLabels, &wlCopy)
	}

	c.JSON(http.StatusOK, gin.H{"whiteLabels": whiteLabels})
}

func (s *SuperAdminService) UpdateWhiteLabelStatus(c *gin.Context) {
	claims, err := s.GetAdminClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	whiteLabelID := c.Param("id")
	status := c.Query("status")

	if _, ok := s.whiteLabels[whiteLabelID]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "white label not found"})
		return
	}

	s.whiteLabels[whiteLabelID].Status = status
	s.whiteLabels[whiteLabelID].UpdatedAt = time.Now()

	// Log audit
	s.logAudit(claims.AdminID, claims.Email, "UPDATE_WHITE_LABEL_STATUS", "white_label", whiteLabelID, fmt.Sprintf("Status changed to: %s", status))

	c.JSON(http.StatusOK, s.whiteLabels[whiteLabelID])
}

// ============================================================================
// Blockchain Management
// ============================================================================

type AddBlockchainRequest struct {
	ChainID     int64  `json:"chainId" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Symbol      string `json:"symbol" binding:"required"`
	Explorer    string `json:"explorer"`
	RPCURL      string `json:"rpcUrl"`
	IsEVM       bool   `json:"isEvm"`
	NetworkType string `json:"networkType"`
}

func (s *SuperAdminService) AddBlockchain(c *gin.Context) {
	claims, err := s.GetAdminClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if claims.Role != RoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req AddBlockchainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	blockchain := &BlockchainNetwork{
		ID:          uuid.New().String(),
		ChainID:     req.ChainID,
		Name:        req.Name,
		Symbol:      req.Symbol,
		Explorer:    req.Explorer,
		RPCURL:      req.RPCURL,
		IsEnabled:   true,
		IsEVM:       req.IsEVM,
		NetworkType: req.NetworkType,
		AddedBy:     claims.AdminID,
		CreatedAt:   time.Now(),
	}

	s.blockchains[blockchain.ID] = blockchain

	// Log audit
	s.logAudit(claims.AdminID, claims.Email, "ADD_BLOCKCHAIN", "blockchain", blockchain.ID, fmt.Sprintf("Added blockchain: %s", req.Name))

	c.JSON(http.StatusCreated, blockchain)
}

func (s *SuperAdminService) ListBlockchains(c *gin.Context) {
	blockchains := make([]*BlockchainNetwork, 0, len(s.blockchains))
	for _, bc := range s.blockchains {
		blockchains = append(blockchains, bc)
	}

	c.JSON(http.StatusOK, gin.H{"blockchains": blockchains})
}

// ============================================================================
// Token Management
// ============================================================================

type AddTokenRequest struct {
	Address   string `json:"address" binding:"required"`
	ChainID   int64  `json:"chainId" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Symbol    string `json:"symbol" binding:"required"`
	Decimals  int    `json:"decimals" binding:"required"`
	IsPopular bool   `json:"isPopular"`
}

func (s *SuperAdminService) AddToken(c *gin.Context) {
	claims, err := s.GetAdminClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if claims.Role != RoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req AddTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token := &Token{
		ID:        uuid.New().String(),
		Address:   req.Address,
		ChainID:   req.ChainID,
		Name:      req.Name,
		Symbol:    req.Symbol,
		Decimals:  req.Decimals,
		IsEnabled: true,
		IsPopular: req.IsPopular,
		AddedBy:  claims.AdminID,
		CreatedAt: time.Now(),
	}

	s.tokens[token.ID] = token

	// Log audit
	s.logAudit(claims.AdminID, claims.Email, "ADD_TOKEN", "token", token.ID, fmt.Sprintf("Added token: %s", req.Symbol))

	c.JSON(http.StatusCreated, token)
}

// ============================================================================
// Audit Logs
// ============================================================================

func (s *SuperAdminService) logAudit(adminID, adminEmail, action, resource, resourceID, details string) {
	audit := AuditLog{
		ID:         uuid.New().String(),
		AdminID:    adminID,
		AdminEmail: adminEmail,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		CreatedAt:  time.Now(),
	}

	s.auditLogs = append(s.auditLogs, audit)
}

func (s *SuperAdminService) ListAuditLogs(c *gin.Context) {
	claims, err := s.GetAdminClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if claims.Role != RoleSuperAdmin && claims.Role != RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"auditLogs": s.auditLogs})
}

// ============================================================================
// Middleware
// ============================================================================

func (s *SuperAdminService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:] // Remove "Bearer "
		claims, err := s.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("adminId", claims.AdminID)
		c.Set("adminEmail", claims.Email)
		c.Set("adminRole", claims.Role)

		c.Next()
	}
}

func (s *SuperAdminService) GetAdminClaims(c *gin.Context) (*Claims, error) {
	adminID, exists := c.Get("adminId")
	if !exists {
		return nil, fmt.Errorf("unauthorized")
	}

	admin, ok := s.admins[adminID.(string)]
	if !ok {
		return nil, fmt.Errorf("admin not found")
	}

	return &Claims{
		AdminID: admin.ID,
		Email:   admin.Email,
		Role:    admin.Role,
	}, nil
}

// ============================================================================
// Router Setup
// ============================================================================

func (s *SuperAdminService) SetupRouter() *gin.Engine {
	if s.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "super-admin"})
	})

	// Auth routes (public)
	auth := r.Group("/auth")
	{
		auth.POST("/login", s.Login)
	}

	// Protected routes
	api := r.Group("/api")
	api.Use(s.AuthMiddleware())
	{
		// Admin management
		api.POST("/admins", s.CreateAdmin)
		api.GET("/admins", s.ListAdmins)

		// White label management
		api.POST("/white-labels", s.CreateWhiteLabel)
		api.GET("/white-labels", s.ListWhiteLabels)
		api.PUT("/white-labels/:id/status", s.UpdateWhiteLabelStatus)

		// Blockchain management
		api.POST("/blockchains", s.AddBlockchain)
		api.GET("/blockchains", s.ListBlockchains)

		// Token management
		api.POST("/tokens", s.AddToken)

		// Audit logs
		api.GET("/audit-logs", s.ListAuditLogs)
	}

	return r
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()

	// Initialize service
	service := NewSuperAdminService(config)

	// Create default super admin
	superAdmin := &Admin{
		ID:        uuid.New().String(),
		Email:     "admin@tigerwallet.com",
		Username:  "admin",
		Role:      RoleSuperAdmin,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// Default password: TigerAdmin@2024
	superAdmin.PasswordHash = "$2a$10$rVqKxVxR5xJ5z5xJ5z5xJe5xJ5z5xJ5z5xJ5z5xJ5z5xJ5z5xJ5"
	service.admins[superAdmin.ID] = superAdmin

	// Setup router
	router := service.SetupRouter()

	// Start server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Super Admin service starting on port %s", config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
