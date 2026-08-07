package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port          string
	RedisURL      string
	JWTSecret     string
	AdminEmail    string
	AdminPassword string
	MongoDBURL    string
}

func LoadConfig() *Config {
	return &Config{
		Port:          getEnv("PORT", "8448"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:     getEnv("JWT_SECRET", "tigerwallet-secret-key-change-in-production"),
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@tigerwallet.io"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		MongoDBURL:    getEnv("MONGODB_URL", "mongodb://localhost:27017"),
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

type Admin struct {
	ID               string   `json:"id" bson:"_id"`
	Username         string   `json:"username" bson:"username"`
	Email            string   `json:"email" bson:"email"`
	PasswordHash     string   `json:"-" bson:"password_hash"`
	Role             string   `json:"role" bson:"role"`
	Permissions      []string `json:"permissions" bson:"permissions"`
	CreatedAt        int64    `json:"created_at" bson:"created_at"`
	UpdatedAt        int64    `json:"updated_at" bson:"updated_at"`
	LastLoginAt      int64    `json:"last_login_at" bson:"last_login_at"`
	Status           string   `json:"status" bson:"status"`
	TwoFactorEnabled bool     `json:"two_factor_enabled" bson:"two_factor_enabled"`
	IPWhitelist      []string `json:"ip_whitelist" bson:"ip_whitelist"`
}

type WhiteLabelClient struct {
	ID           string             `json:"id" bson:"_id"`
	Name         string             `json:"name" bson:"name"`
	Domain       string             `json:"domain" bson:"domain"`
	Branding     WhiteLabelBranding `json:"branding" bson:"branding"`
	Config       WhiteLabelConfig   `json:"config" bson:"config"`
	Permissions  []string           `json:"permissions" bson:"permissions"`
	Status       string             `json:"status" bson:"status"`
	CreatedAt    int64              `json:"created_at" bson:"created_at"`
	UpdatedAt    int64              `json:"updated_at" bson:"updated_at"`
	MasterWallet string             `json:"master_wallet" bson:"master_wallet"`
}

type WhiteLabelBranding struct {
	Name            string `json:"name" bson:"name"`
	Logo            string `json:"logo" bson:"logo"`
	Favicon         string `json:"favicon" bson:"favicon"`
	PrimaryColor    string `json:"primary_color" bson:"primary_color"`
	SecondaryColor  string `json:"secondary_color" bson:"secondary_color"`
	BackgroundColor string `json:"background_color" bson:"background_color"`
	TextColor       string `json:"text_color" bson:"text_color"`
}

type WhiteLabelConfig struct {
	Blockchains []int     `json:"blockchains" bson:"blockchains"`
	Tokens      []string  `json:"tokens" bson:"tokens"`
	Features    []string  `json:"features" bson:"features"`
	Fees        FeeConfig `json:"fees" bson:"fees"`
}

type FeeConfig struct {
	WithdrawFee        string  `json:"withdraw_fee" bson:"withdraw_fee"`
	WithdrawFeePercent float64 `json:"withdraw_fee_percent" bson:"withdraw_fee_percent"`
	SwapFeePercent     float64 `json:"swap_fee_percent" bson:"swap_fee_percent"`
	TransferFeePercent float64 `json:"transfer_fee_percent" bson:"transfer_fee_percent"`
}

type User struct {
	ID              string   `json:"id" bson:"_id"`
	Email           string   `json:"email" bson:"email"`
	Username        string   `json:"username" bson:"username"`
	WalletAddresses []string `json:"wallet_addresses" bson:"wallet_addresses"`
	KYCStatus       string   `json:"kyc_status" bson:"kyc_status"`
	KYCLevel        int      `json:"kyc_level" bson:"kyc_level"`
	CreatedAt       int64    `json:"created_at" bson:"created_at"`
	UpdatedAt       int64    `json:"updated_at" bson:"updated_at"`
	Status          string   `json:"status" bson:"status"`
}

type Transaction struct {
	ID          string `json:"id" bson:"_id"`
	Hash        string `json:"hash" bson:"hash"`
	From        string `json:"from" bson:"from"`
	To          string `json:"to" bson:"to"`
	Value       string `json:"value" bson:"value"`
	ChainID     int    `json:"chain_id" bson:"chain_id"`
	Status      string `json:"status" bson:"status"`
	Type        string `json:"type" bson:"type"`
	Fee         string `json:"fee" bson:"fee"`
	CreatedAt   int64  `json:"created_at" bson:"created_at"`
	ProcessedAt int64  `json:"processed_at" bson:"processed_at"`
	AdminID     string `json:"admin_id" bson:"admin_id"`
}

type Token struct {
	ID          string `json:"id" bson:"_id"`
	ChainID     int    `json:"chain_id" bson:"chain_id"`
	Address     string `json:"address" bson:"address"`
	Name        string `json:"name" bson:"name"`
	Symbol      string `json:"symbol" bson:"symbol"`
	Decimals    int    `json:"decimals" bson:"decimals"`
	TotalSupply string `json:"total_supply" bson:"total_supply"`
	Status      string `json:"status" bson:"status"`
	CreatedAt   int64  `json:"created_at" bson:"created_at"`
}

type Pair struct {
	ID          string `json:"id" bson:"_id"`
	BaseToken   string `json:"base_token" bson:"base_token"`
	QuoteToken  string `json:"quote_token" bson:"quote_token"`
	ChainID     int    `json:"chain_id" bson:"chain_id"`
	PoolAddress string `json:"pool_address" bson:"pool_address"`
	Status      string `json:"status" bson:"status"`
	CreatedAt   int64  `json:"created_at" bson:"created_at"`
}

type Liquidity struct {
	ID          string `json:"id" bson:"_id"`
	PairID      string `json:"pair_id" bson:"pair_id"`
	Provider    string `json:"provider" bson:"provider"`
	AmountBase  string `json:"amount_base" bson:"amount_base"`
	AmountQuote string `json:"amount_quote" bson:"amount_quote"`
	CreatedAt   int64  `json:"created_at" bson:"created_at"`
}

type APIKey struct {
	ID          string   `json:"id" bson:"_id"`
	UserID      string   `json:"user_id" bson:"user_id"`
	Key         string   `json:"key" bson:"key"`
	Name        string   `json:"name" bson:"name"`
	Permissions []string `json:"permissions" bson:"permissions"`
	Status      string   `json:"status" bson:"status"`
	CreatedAt   int64    `json:"created_at" bson:"created_at"`
	ExpiresAt   int64    `json:"expires_at" bson:"expires_at"`
	LastUsedAt  int64    `json:"last_used_at" bson:"last_used_at"`
}

// ============================================================================
// Admin Service
// ============================================================================

type AdminService struct {
	config  *Config
	redis   *redis.Client
	admins  map[string]*Admin
	clients map[string]*WhiteLabelClient
}

func NewAdminService(config *Config) *AdminService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	service := &AdminService{
		config:  config,
		redis:   redisClient,
		admins:  make(map[string]*Admin),
		clients: make(map[string]*WhiteLabelClient),
	}

	// Initialize default admin
	service.initializeDefaultAdmin()

	return service
}

func (s *AdminService) initializeDefaultAdmin() {
	// Hash the admin password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(s.config.AdminPassword), bcrypt.DefaultCost)

	admin := &Admin{
		ID:               "admin-1",
		Username:         "superadmin",
		Email:            s.config.AdminEmail,
		PasswordHash:     string(hashedPassword),
		Role:             "super_admin",
		Permissions:      s.getAllPermissions(),
		CreatedAt:        time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
		Status:           "active",
		TwoFactorEnabled: false,
	}

	s.admins[admin.ID] = admin
	s.admins[admin.Email] = admin
}

func (s *AdminService) getAllPermissions() []string {
	return []string{
		"users:read", "users:write", "users:delete",
		"wallets:read", "wallets:write",
		"transactions:read", "transactions:approve", "transactions:reject",
		"kyc:read", "kyc:approve", "kyc:reject",
		"fees:manage",
		"pairs:manage",
		"liquidity:manage",
		"whitelabel:manage",
		"analytics:read",
		"system:config",
		"admin:manage",
	}
}

// ============================================================================
// Authentication
// ============================================================================

func (s *AdminService) login(email, password string) (*Admin, string, error) {
	admin, ok := s.admins[email]
	if !ok {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	if admin.Status != "active" {
		return nil, "", fmt.Errorf("account is not active")
	}

	// Generate JWT token
	token, err := s.generateToken(admin)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token")
	}

	admin.LastLoginAt = time.Now().Unix()
	s.admins[admin.ID] = admin

	return admin, token, nil
}

func (s *AdminService) generateToken(admin *Admin) (string, error) {
	claims := jwt.MapClaims{
		"admin_id": admin.ID,
		"email":    admin.Email,
		"role":     admin.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *AdminService) validateToken(tokenString string) (*Admin, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	adminID, ok := claims["admin_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid admin ID in token")
	}

	admin, ok := s.admins[adminID]
	if !ok {
		return nil, fmt.Errorf("admin not found")
	}

	return admin, nil
}

func (s *AdminService) hasPermission(admin *Admin, permission string) bool {
	for _, p := range admin.Permissions {
		if p == permission || p == "admin:manage" {
			return true
		}
	}
	return false
}

// ============================================================================
// Admin Management
// ============================================================================

func (s *AdminService) createAdmin(email, username, password, role string, permissions []string) (*Admin, error) {
	if _, ok := s.admins[email]; ok {
		return nil, fmt.Errorf("admin already exists")
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	admin := &Admin{
		ID:           fmt.Sprintf("admin-%d", len(s.admins)+1),
		Email:        email,
		Username:     username,
		PasswordHash: string(hashedPassword),
		Role:         role,
		Permissions:  permissions,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
		Status:       "active",
	}

	s.admins[admin.ID] = admin
	s.admins[admin.Email] = admin

	return admin, nil
}

func (s *AdminService) updateAdmin(id string, updates map[string]interface{}) (*Admin, error) {
	admin, ok := s.admins[id]
	if !ok {
		return nil, fmt.Errorf("admin not found")
	}

	if username, ok := updates["username"].(string); ok {
		admin.Username = username
	}
	if role, ok := updates["role"].(string); ok {
		admin.Role = role
	}
	if permissions, ok := updates["permissions"].([]string); ok {
		admin.Permissions = permissions
	}
	if status, ok := updates["status"].(string); ok {
		admin.Status = status
	}

	admin.UpdatedAt = time.Now().Unix()
	s.admins[id] = admin

	return admin, nil
}

func (s *AdminService) deleteAdmin(id string) error {
	admin, ok := s.admins[id]
	if !ok {
		return fmt.Errorf("admin not found")
	}

	if admin.Role == "super_admin" {
		return fmt.Errorf("cannot delete super admin")
	}

	delete(s.admins, id)
	delete(s.admins, admin.Email)
	return nil
}

func (s *AdminService) listAdmins() []*Admin {
	admins := make([]*Admin, 0)
	for _, admin := range s.admins {
		if admin.Role != "super_admin" || admin.Email == s.config.AdminEmail {
			admins = append(admins, admin)
		}
	}
	return admins
}

// ============================================================================
// White Label Management
// ============================================================================

func (s *AdminService) createWhiteLabel(name, domain string, branding WhiteLabelBranding, config WhiteLabelConfig) (*WhiteLabelClient, error) {
	if _, ok := s.clients[domain]; ok {
		return nil, fmt.Errorf("domain already registered")
	}

	client := &WhiteLabelClient{
		ID:           fmt.Sprintf("wl-%d", len(s.clients)+1),
		Name:         name,
		Domain:       domain,
		Branding:     branding,
		Config:       config,
		Permissions:  []string{},
		Status:       "pending",
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
		MasterWallet: "",
	}

	s.clients[client.ID] = client
	s.clients[client.Domain] = client

	return client, nil
}

func (s *AdminService) updateWhiteLabel(id string, updates map[string]interface{}) (*WhiteLabelClient, error) {
	client, ok := s.clients[id]
	if !ok {
		return nil, fmt.Errorf("client not found")
	}

	if name, ok := updates["name"].(string); ok {
		client.Name = name
	}
	if status, ok := updates["status"].(string); ok {
		client.Status = status
	}
	if config, ok := updates["config"].(WhiteLabelConfig); ok {
		client.Config = config
	}

	client.UpdatedAt = time.Now().Unix()
	s.clients[id] = client

	return client, nil
}

func (s *AdminService) deleteWhiteLabel(id string) error {
	client, ok := s.clients[id]
	if !ok {
		return fmt.Errorf("client not found")
	}

	delete(s.clients, id)
	delete(s.clients, client.Domain)
	return nil
}

func (s *AdminService) getWhiteLabel(idOrDomain string) (*WhiteLabelClient, error) {
	client, ok := s.clients[idOrDomain]
	if !ok {
		client, ok = s.clients[idOrDomain]
	}
	if !ok {
		return nil, fmt.Errorf("client not found")
	}
	return client, nil
}

func (s *AdminService) listWhiteLabels() []*WhiteLabelClient {
	clients := make([]*WhiteLabelClient, 0)
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	return clients
}

// ============================================================================
// User Management
// ============================================================================

func (s *AdminService) listUsers(page, pageSize int) ([]*User, int64) {
	// Simplified - would use database in production
	return []*User{}, 0
}

func (s *AdminService) getUser(id string) (*User, error) {
	// Would fetch from database
	return nil, nil
}

func (s *AdminService) updateUserStatus(id, status string) error {
	// Would update in database
	return nil
}

func (s *AdminService) updateKYCStatus(id, status string, level int) error {
	// Would update in database
	return nil
}

// ============================================================================
// Transaction Management
// ============================================================================

func (s *AdminService) listTransactions(page, pageSize int, status string) ([]*Transaction, int64) {
	// Would fetch from database
	return []*Transaction{}, 0
}

func (s *AdminService) approveTransaction(id, adminID string) error {
	// Would approve transaction
	return nil
}

func (s *AdminService) rejectTransaction(id, adminID, reason string) error {
	// Would reject transaction
	return nil
}

// ============================================================================
// Token & Pair Management
// ============================================================================

func (s *AdminService) createToken(token *Token) error {
	// Would save to database
	return nil
}

func (s *AdminService) updateTokenStatus(id, status string) error {
	// Would update in database
	return nil
}

func (s *AdminService) createPair(pair *Pair) error {
	// Would save to database
	return nil
}

func (s *AdminService) updatePairStatus(id, status string) error {
	// Would update in database
	return nil
}

// ============================================================================
// Liquidity Management
// ============================================================================

func (s *AdminService) addLiquidity(liquidity *Liquidity) error {
	// Would save to database
	return nil
}

func (s *AdminService) removeLiquidity(id string) error {
	// Would remove from database
	return nil
}

// ============================================================================
// Analytics
// ============================================================================

func (s *AdminService) getDashboardStats() map[string]interface{} {
	return map[string]interface{}{
		"total_users":          0,
		"active_users":         0,
		"total_transactions":   0,
		"volume_24h":           "0",
		"volume_7d":            "0",
		"revenue_24h":          "0",
		"pending_kyc":          0,
		"pending_transactions": 0,
	}
}

func (s *AdminService) getUserAnalytics(startDate, endDate int64) map[string]interface{} {
	return map[string]interface{}{
		"new_users":     0,
		"active_users":  0,
		"kyc_completed": 0,
	}
}

func (s *AdminService) getTransactionAnalytics(startDate, endDate int64) map[string]interface{} {
	return map[string]interface{}{
		"total_transactions": 0,
		"total_volume":       "0",
		"by_status":          map[string]int{},
		"by_type":            map[string]int{},
	}
}

// ============================================================================
// Audit Logging
// ============================================================================

type AuditLog struct {
	ID         string `json:"id"`
	AdminID    string `json:"admin_id"`
	Action     string `json:"action"`
	Resource   string `json:"resource"`
	ResourceID string `json:"resource_id"`
	Details    string `json:"details"`
	IPAddress  string `json:"ip_address"`
	CreatedAt  int64  `json:"created_at"`
}

func (s *AdminService) logAudit(adminID, action, resource, resourceID, details, ipAddress string) {
	logEntry := AuditLog{
		ID:         fmt.Sprintf("audit-%d", time.Now().Unix()),
		AdminID:    adminID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		IPAddress:  ipAddress,
		CreatedAt:  time.Now().Unix(),
	}

	// Would save to database
	log.Printf("AUDIT: %+v", logEntry)
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *AdminService) RegisterRoutes(r *gin.Engine) {
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "admin-service"})
	})

	// Public routes
	api := r.Group("/api/v1/admin")
	{
		api.POST("/login", s.handleLogin)
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(s.authMiddleware())
	{
		// Auth
		protected.POST("/logout", s.handleLogout)
		protected.GET("/me", s.handleGetMe)

		// Admins (super admin only)
		protected.GET("/admins", s.handleListAdmins)
		protected.POST("/admins", s.handleCreateAdmin)
		protected.PUT("/admins/:id", s.handleUpdateAdmin)
		protected.DELETE("/admins/:id", s.handleDeleteAdmin)

		// White Label
		protected.GET("/whitelabels", s.handleListWhiteLabels)
		protected.POST("/whitelabels", s.handleCreateWhiteLabel)
		protected.PUT("/whitelabels/:id", s.handleUpdateWhiteLabel)
		protected.DELETE("/whitelabels/:id", s.handleDeleteWhiteLabel)

		// Users
		protected.GET("/users", s.handleListUsers)
		protected.GET("/users/:id", s.handleGetUser)
		protected.PUT("/users/:id/status", s.handleUpdateUserStatus)
		protected.PUT("/users/:id/kyc", s.handleUpdateKYC)

		// Transactions
		protected.GET("/transactions", s.handleListTransactions)
		protected.POST("/transactions/:id/approve", s.handleApproveTransaction)
		protected.POST("/transactions/:id/reject", s.handleRejectTransaction)

		// Tokens
		protected.GET("/tokens", s.handleListTokens)
		protected.POST("/tokens", s.handleCreateToken)
		protected.PUT("/tokens/:id/status", s.handleUpdateTokenStatus)

		// Pairs
		protected.GET("/pairs", s.handleListPairs)
		protected.POST("/pairs", s.handleCreatePair)
		protected.PUT("/pairs/:id/status", s.handleUpdatePairStatus)

		// Liquidity
		protected.GET("/liquidity", s.handleListLiquidity)
		protected.POST("/liquidity", s.handleAddLiquidity)
		protected.DELETE("/liquidity/:id", s.handleRemoveLiquidity)

		// Analytics
		protected.GET("/analytics/dashboard", s.handleGetDashboard)
		protected.GET("/analytics/users", s.handleGetUserAnalytics)
		protected.GET("/analytics/transactions", s.handleGetTransactionAnalytics)

		// Audit
		protected.GET("/audit", s.handleGetAuditLogs)
	}
}

func (s *AdminService) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		admin, err := s.validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("admin", admin)
		c.Next()
	}
}

func (s *AdminService) handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin, token, err := s.login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"admin": admin,
	})
}

func (s *AdminService) handleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (s *AdminService) handleGetMe(c *gin.Context) {
	admin := c.MustGet("admin").(*Admin)
	c.JSON(http.StatusOK, admin)
}

func (s *AdminService) handleListAdmins(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"admins": s.listAdmins()})
}

func (s *AdminService) handleCreateAdmin(c *gin.Context) {
	var req struct {
		Email       string   `json:"email" binding:"required"`
		Username    string   `json:"username" binding:"required"`
		Password    string   `json:"password" binding:"required"`
		Role        string   `json:"role" binding:"required"`
		Permissions []string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin, err := s.createAdmin(req.Email, req.Username, req.Password, req.Role, req.Permissions)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, admin)
}

func (s *AdminService) handleUpdateAdmin(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin, err := s.updateAdmin(id, updates)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, admin)
}

func (s *AdminService) handleDeleteAdmin(c *gin.Context) {
	id := c.Param("id")

	if err := s.deleteAdmin(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin deleted"})
}

func (s *AdminService) handleListWhiteLabels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"clients": s.listWhiteLabels()})
}

func (s *AdminService) handleCreateWhiteLabel(c *gin.Context) {
	var req struct {
		Name     string             `json:"name" binding:"required"`
		Domain   string             `json:"domain" binding:"required"`
		Branding WhiteLabelBranding `json:"branding"`
		Config   WhiteLabelConfig   `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := s.createWhiteLabel(req.Name, req.Domain, req.Branding, req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, client)
}

func (s *AdminService) handleUpdateWhiteLabel(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := s.updateWhiteLabel(id, updates)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, client)
}

func (s *AdminService) handleDeleteWhiteLabel(c *gin.Context) {
	id := c.Param("id")

	if err := s.deleteWhiteLabel(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "client deleted"})
}

func (s *AdminService) handleListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	users, total := s.listUsers(page, pageSize)
	c.JSON(http.StatusOK, gin.H{
		"users":     users,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func (s *AdminService) handleGetUser(c *gin.Context) {
	id := c.Param("id")

	user, err := s.getUser(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (s *AdminService) handleUpdateUserStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.updateUserStatus(id, req.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

func (s *AdminService) handleUpdateKYC(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
		Level  int    `json:"level"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.updateKYCStatus(id, req.Status, req.Level); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "KYC updated"})
}

func (s *AdminService) handleListTransactions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	txns, total := s.listTransactions(page, pageSize, status)
	c.JSON(http.StatusOK, gin.H{
		"transactions": txns,
		"page":         page,
		"page_size":    pageSize,
		"total":        total,
	})
}

func (s *AdminService) handleApproveTransaction(c *gin.Context) {
	id := c.Param("id")
	admin := c.MustGet("admin").(*Admin)

	if err := s.approveTransaction(id, admin.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.logAudit(admin.ID, "approve", "transaction", id, "", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "transaction approved"})
}

func (s *AdminService) handleRejectTransaction(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin := c.MustGet("admin").(*Admin)

	if err := s.rejectTransaction(id, admin.ID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.logAudit(admin.ID, "reject", "transaction", id, req.Reason, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "transaction rejected"})
}

func (s *AdminService) handleListTokens(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tokens": []*Token{}})
}

func (s *AdminService) handleCreateToken(c *gin.Context) {
	var token Token
	if err := c.ShouldBindJSON(&token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.createToken(&token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, token)
}

func (s *AdminService) handleUpdateTokenStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.updateTokenStatus(id, req.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

func (s *AdminService) handleListPairs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"pairs": []*Pair{}})
}

func (s *AdminService) handleCreatePair(c *gin.Context) {
	var pair Pair
	if err := c.ShouldBindJSON(&pair); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.createPair(&pair); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, pair)
}

func (s *AdminService) handleUpdatePairStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.updatePairStatus(id, req.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

func (s *AdminService) handleListLiquidity(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"liquidity": []*Liquidity{}})
}

func (s *AdminService) handleAddLiquidity(c *gin.Context) {
	var liquidity Liquidity
	if err := c.ShouldBindJSON(&liquidity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.addLiquidity(&liquidity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, liquidity)
}

func (s *AdminService) handleRemoveLiquidity(c *gin.Context) {
	id := c.Param("id")

	if err := s.removeLiquidity(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "liquidity removed"})
}

func (s *AdminService) handleGetDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, s.getDashboardStats())
}

func (s *AdminService) handleGetUserAnalytics(c *gin.Context) {
	startDate, _ := strconv.ParseInt(c.Query("start_date"), 10, 64)
	endDate, _ := strconv.ParseInt(c.Query("end_date"), 10, 64)

	c.JSON(http.StatusOK, s.getUserAnalytics(startDate, endDate))
}

func (s *AdminService) handleGetTransactionAnalytics(c *gin.Context) {
	startDate, _ := strconv.ParseInt(c.Query("start_date"), 10, 64)
	endDate, _ := strconv.ParseInt(c.Query("end_date"), 10, 64)

	c.JSON(http.StatusOK, s.getTransactionAnalytics(startDate, endDate))
}

func (s *AdminService) handleGetAuditLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"logs": []*AuditLog{}})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()
	service := NewAdminService(config)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	service.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Admin Service starting on port %s", config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

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
