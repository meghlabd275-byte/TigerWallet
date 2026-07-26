/**
 * TigerWallet White Label Service
 * Comprehensive white label management system for customizable exchange/wallet deployment
 */

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
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// Configuration
type Config struct {
	ServerPort string
	RedisAddr  string
	JWTSecret  string
}

// White Label Types
type WLStatus string

const (
	WLStatusPending   WLStatus = "PENDING"
	WLStatusActive    WLStatus = "ACTIVE"
	WLStatusSuspended WLStatus = "SUSPENDED"
	WLStatusCancelled WLStatus = "CANCELLED"
)

type ProductType string

const (
	ProductWallet    ProductType = "WALLET"
	ProductExchange  ProductType = "EXCHANGE"
	ProductDEX       ProductType = "DEX"
	ProductCEXDEX    ProductType = "CEX_DEX"
)

// Branding
type Branding struct {
	Logo          string `json:"logo"`
	Favicon       string `json:"favicon"`
	PrimaryColor  string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor   string `json:"accent_color"`
	BackgroundColor string `json:"background_color"`
	TextColor     string `json:"text_color"`
	Name          string `json:"name"`
	Slogan        string `json:"slogan"`
	SupportEmail  string `json:"support_email"`
	SupportURL    string `json:"support_url"`
}

// Fee Structure
type FeeStructure struct {
	TradingFee      float64 `json:"trading_fee"`
	WithdrawalFee  float64 `json:"withdrawal_fee"`
	DepositFee     float64 `json:"deposit_fee"`
	MakerFee       float64 `json:"maker_fee"`
	TakerFee       float64 `json:"taker_fee"`
	ConversionFee  float64 `json:"conversion_fee"`
}

// Feature Flags
type FeatureFlags struct {
	SpotTrading       bool `json:"spot_trading"`
	FuturesTrading    bool `json:"futures_trading"`
	OptionsTrading    bool `json:"options_trading"`
	Staking           bool `json:"staking"`
	Lending           bool `json:"lending"`
	NFT               bool `json:"nft"`
	Launchpad         bool `json:"launchpad"`
	CopyTrading       bool `json:"copy_trading"`
	P2PTrading        bool `json:"p2p_trading"`
	FiatOnramp       bool `json:"fiat_onramp"`
	CryptoCard        bool `json:"crypto_card"`
	API               bool `json:"api"`
	MobileApp         bool `json:"mobile_app"`
	WebApp            bool `json:"web_app"`
	BrowserExtension  bool `json:"browser_extension"`
}

// Supported Assets
type SupportedAssets struct {
	Cryptocurrencies []string `json:"cryptocurrencies"`
	Fiats           []string `json:"fiats"`
	Networks        []string `json:"networks"`
}

// White Label Client
type WhiteLabelClient struct {
	ClientID       string          `json:"client_id"`
	Name           string          `json:"name"`
	Domain         string          `json:"domain"`
	CustomDomains  []string        `json:"custom_domains"`
	Status         WLStatus        `json:"status"`
	ProductType    ProductType     `json:"product_type"`
	Branding       Branding        `json:"branding"`
	FeeStructure   FeeStructure    `json:"fee_structure"`
	Features       FeatureFlags    `json:"features"`
	Assets         SupportedAssets `json:"assets"`
	AdminEmail     string          `json:"admin_email"`
	AdminID        string          `json:"admin_id"`
	ResellerID     string          `json:"reseller_id"`
	WhitelabelID   string          `json:"whitelabel_id"`
	Tier           string           `json:"tier"` // basic, professional, enterprise
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	ActivatedAt    *time.Time      `json:"activated_at,omitempty"`
	ExpiresAt      *time.Time      `json:"expires_at,omitempty"`
}

// White Label Admin
type WhiteLabelAdmin struct {
	AdminID        string    `json:"admin_id"`
	ClientID       string    `json:"client_id"`
	Email          string    `json:"email"`
	Username       string    `json:"username"`
	PasswordHash   string    `json:"password_hash"`
	Role           string    `json:"role"` // owner, admin, manager, support
	Permissions    []string  `json:"permissions"`
	Status         string    `json:"status"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Phone          string    `json:"phone"`
	LastLogin      *time.Time `json:"last_login,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Subscription
type Subscription struct {
	SubscriptionID string    `json:"subscription_id"`
	ClientID       string    `json:"client_id"`
	Plan           string    `json:"plan"` // starter, professional, enterprise
	Price          float64   `json:"price"`
	BillingCycle   string    `json:"billing_cycle"` // monthly, yearly
	Status         string    `json:"status"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	AutoRenew      bool      `json:"auto_renew"`
}

// White Label Service
type WhiteLabelService struct {
	config      Config
	redis       *redis.Client
	clients     map[string]*WhiteLabelClient
	admins      map[string]*WhiteLabelAdmin
	subscriptions map[string]*Subscription
	mu          sync.RWMutex
}

// NewWhiteLabelService creates a new white label service
func NewWhiteLabelService(cfg Config) *WhiteLabelService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		DB:   4,
	})

	return &WhiteLabelService{
		config:       cfg,
		redis:        redisClient,
		clients:     make(map[string]*WhiteLabelClient),
		admins:      make(map[string]*WhiteLabelAdmin),
		subscriptions: make(map[string]*Subscription),
	}
}

// Create client
func (s *WhiteLabelService) CreateClient(client *WhiteLabelClient) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate domain
	if client.Domain == "" {
		return fmt.Errorf("domain is required")
	}

	// Check if domain exists
	for _, c := range s.clients {
		if c.Domain == client.Domain {
			return fmt.Errorf("domain already exists")
		}
		for _, d := range c.CustomDomains {
			if d == client.Domain {
				return fmt.Errorf("domain already exists")
			}
		}
	}

	client.ClientID = "wl_" + uuid.New().String()[:8]
	client.Status = WLStatusPending
	client.CreatedAt = time.Now()
	client.UpdatedAt = time.Now()

	// Set default branding if not provided
	if client.Branding.Name == "" {
		client.Branding = Branding{
			Name:          client.Name,
			PrimaryColor:  "#f59e0b",
			SecondaryColor: "#1e293b",
			AccentColor:   "#10b981",
		}
	}

	// Set default features
	if !client.Features.WebApp {
		client.Features = FeatureFlags{
			SpotTrading:    true,
			WebApp:        true,
			MobileApp:     true,
			FiatOnramp:    true,
		}
	}

	// Set default assets
	if len(client.Assets.Cryptocurrencies) == 0 {
		client.Assets = SupportedAssets{
			Cryptocurrencies: []string{"BTC", "ETH", "USDT", "USDC", "BNB"},
			Fiats:           []string{"USD", "EUR", "GBP"},
			Networks:        []string{"Ethereum", "BSC", "Polygon"},
		}
	}

	s.clients[client.ClientID] = client

	// Create subscription
	subscription := &Subscription{
		SubscriptionID: "sub_" + uuid.New().String()[:8],
		ClientID:       client.ClientID,
		Plan:           "professional",
		Price:          999.0,
		BillingCycle:  "monthly",
		Status:         "active",
		StartDate:      time.Now(),
		EndDate:        time.Now().AddDate(0, 1, 0),
		AutoRenew:      true,
	}

	s.subscriptions[client.ClientID] = subscription

	return nil
}

// Get client
func (s *WhiteLabelService) GetClient(clientID string) (*WhiteLabelClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, ok := s.clients[clientID]
	if !ok {
		return nil, fmt.Errorf("client not found")
	}

	return client, nil
}

// Get client by domain
func (s *WhiteLabelService) GetClientByDomain(domain string) (*WhiteLabelClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		if client.Domain == domain || contains(client.CustomDomains, domain) {
			return client, nil
		}
	}

	return nil, fmt.Errorf("client not found for domain")
}

// Update client
func (s *WhiteLabelService) UpdateClient(clientID string, updates map[string]interface{}) (*WhiteLabelClient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, ok := s.clients[clientID]
	if !ok {
		return nil, fmt.Errorf("client not found")
	}

	// Update branding
	if branding, ok := updates["branding"].(map[string]interface{}); ok {
		if name, ok := branding["name"].(string); ok {
			client.Branding.Name = name
		}
		if primary, ok := branding["primary_color"].(string); ok {
			client.Branding.PrimaryColor = primary
		}
		if secondary, ok := branding["secondary_color"].(string); ok {
			client.Branding.SecondaryColor = secondary
		}
		if accent, ok := branding["accent_color"].(string); ok {
			client.Branding.AccentColor = accent
		}
		if logo, ok := branding["logo"].(string); ok {
			client.Branding.Logo = logo
		}
	}

	// Update fee structure
	if fees, ok := updates["fee_structure"].(map[string]interface{}); ok {
		if tradingFee, ok := fees["trading_fee"].(float64); ok {
			client.FeeStructure.TradingFee = tradingFee
		}
		if withdrawalFee, ok := fees["withdrawal_fee"].(float64); ok {
			client.FeeStructure.WithdrawalFee = withdrawalFee
		}
		if depositFee, ok := fees["deposit_fee"].(float64); ok {
			client.FeeStructure.DepositFee = depositFee
		}
	}

	// Update status
	if status, ok := updates["status"].(string); ok {
		client.Status = WLStatus(status)
	}

	// Update features
	if features, ok := updates["features"].(map[string]interface{}); ok {
		if spotTrading, ok := features["spot_trading"].(bool); ok {
			client.Features.SpotTrading = spotTrading
		}
		if futuresTrading, ok := features["futures_trading"].(bool); ok {
			client.Features.FuturesTrading = futuresTrading
		}
		if staking, ok := features["staking"].(bool); ok {
			client.Features.Staking = staking
		}
		if lending, ok := features["lending"].(bool); ok {
			client.Features.Lending = lending
		}
		if nft, ok := features["nft"].(bool); ok {
			client.Features.NFT = nft
		}
	}

	client.UpdatedAt = time.Now()

	return client, nil
}

// Activate client
func (s *WhiteLabelService) ActivateClient(clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, ok := s.clients[clientID]
	if !ok {
		return fmt.Errorf("client not found")
	}

	client.Status = WLStatusActive
	now := time.Now()
	client.ActivatedAt = &now
	client.UpdatedAt = time.Now()

	return nil
}

// Suspend client
func (s *WhiteLabelService) SuspendClient(clientID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, ok := s.clients[clientID]
	if !ok {
		return fmt.Errorf("client not found")
	}

	client.Status = WLStatusSuspended
	client.UpdatedAt = time.Now()

	return nil
}

// Get all clients
func (s *WhiteLabelService) GetClients(status, tier string) []*WhiteLabelClient {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]*WhiteLabelClient, 0)
	for _, client := range s.clients {
		match := true
		if status != "" && string(client.Status) != status {
			match = false
		}
		if tier != "" && client.Tier != tier {
			match = false
		}
		if match {
			clients = append(clients, client)
		}
	}

	return clients
}

// Create admin for client
func (s *WhiteLabelService) CreateClientAdmin(clientID string, admin *WhiteLabelAdmin) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, ok := s.clients[clientID]
	if !ok {
		return fmt.Errorf("client not found")
	}

	admin.AdminID = "admin_" + uuid.New().String()[:8]
	admin.ClientID = clientID
	admin.Status = "active"
	admin.CreatedAt = time.Now()
	admin.UpdatedAt = time.Now()

	// Generate password hash
	h := sha256.Sum256([]byte(admin.PasswordHash))
	admin.PasswordHash = hex.EncodeToString(h[:])

	s.admins[admin.AdminID] = admin

	return nil
}

// Get client admin
func (s *WhiteLabelService) GetClientAdmin(adminID string) (*WhiteLabelAdmin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	admin, ok := s.admins[adminID]
	if !ok {
		return nil, fmt.Errorf("admin not found")
	}

	return admin, nil
}

// Authenticate client admin
func (s *WhiteLabelService) AuthenticateAdmin(email, password string) (*WhiteLabelAdmin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var admin *WhiteLabelAdmin
	for _, a := range s.admins {
		if a.Email == email {
			admin = a
			break
		}
	}

	if admin == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password
	h := sha256.Sum256([]byte(password))
	passwordHash := hex.EncodeToString(h[:])

	if admin.PasswordHash != passwordHash {
		return nil, fmt.Errorf("invalid credentials")
	}

	if admin.Status != "active" {
		return nil, fmt.Errorf("account is not active")
	}

	// Update last login
	now := time.Now()
	admin.LastLogin = &now

	return admin, nil
}

// Get subscription
func (s *WhiteLabelService) GetSubscription(clientID string) (*Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sub, ok := s.subscriptions[clientID]
	if !ok {
		return nil, fmt.Errorf("subscription not found")
	}

	return sub, nil
}

// Update subscription
func (s *WhiteLabelService) UpdateSubscription(clientID string, plan string, price float64, billingCycle string) (*Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subscriptions[clientID]
	if !ok {
		return nil, fmt.Errorf("subscription not found")
	}

	sub.Plan = plan
	sub.Price = price
	sub.BillingCycle = billingCycle
	sub.EndDate = time.Now().AddDate(0, 1, 0)

	return sub, nil
}

// Get client stats
func (s *WhiteLabelService) GetClientStats(clientID string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"client_id": clientID,
	}

	client, ok := s.clients[clientID]
	if !ok {
		return stats
	}

	stats["name"] = client.Name
	stats["domain"] = client.Domain
	stats["status"] = client.Status
	stats["tier"] = client.Tier
	stats["created_at"] = client.CreatedAt

	// Count admins
	adminCount := 0
	for _, admin := range s.admins {
		if admin.ClientID == clientID {
			adminCount++
		}
	}
	stats["admin_count"] = adminCount

	return stats
}

// Handlers
func (s *WhiteLabelService) CreateClientHandler(c *gin.Context) {
	var client WhiteLabelClient
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.CreateClient(&client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, client)
}

func (s *WhiteLabelService) GetClientHandler(c *gin.Context) {
	clientID := c.Param("id")

	client, err := s.GetClient(clientID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, client)
}

func (s *WhiteLabelService) GetClientByDomainHandler(c *gin.Context) {
	domain := c.Param("domain")

	client, err := s.GetClientByDomain(domain)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, client)
}

func (s *WhiteLabelService) UpdateClientHandler(c *gin.Context) {
	clientID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := s.UpdateClient(clientID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, client)
}

func (s *WhiteLabelService) ActivateClientHandler(c *gin.Context) {
	clientID := c.Param("id")

	if err := s.ActivateClient(clientID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "activated"})
}

func (s *WhiteLabelService) SuspendClientHandler(c *gin.Context) {
	clientID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	if err := s.SuspendClient(clientID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "suspended"})
}

func (s *WhiteLabelService) GetClientsHandler(c *gin.Context) {
	status := c.Query("status")
	tier := c.Query("tier")

	clients := s.GetClients(status, tier)
	c.JSON(http.StatusOK, gin.H{"clients": clients})
}

func (s *WhiteLabelService) LoginAdminHandler(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin, err := s.AuthenticateAdmin(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, admin)
}

func (s *WhiteLabelService) GetSubscriptionHandler(c *gin.Context) {
	clientID := c.Param("client_id")

	sub, err := s.GetSubscription(clientID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

func (s *WhiteLabelService) GetStatsHandler(c *gin.Context) {
	clientID := c.Param("client_id")

	stats := s.GetClientStats(clientID)
	c.JSON(http.StatusOK, stats)
}

func (s *WhiteLabelService) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/whitelabel")
	{
		// Public routes
		api.GET("/domain/:domain", s.GetClientByDomainHandler)
		api.POST("/admin/login", s.LoginAdminHandler)

		// Protected routes
		protected := api.Group("")
		protected.Use(s.AuthMiddleware())
		{
			protected.POST("/clients", s.CreateClientHandler)
			protected.GET("/clients", s.GetClientsHandler)
			protected.GET("/clients/:id", s.GetClientHandler)
			protected.PUT("/clients/:id", s.UpdateClientHandler)
			protected.POST("/clients/:id/activate", s.ActivateClientHandler)
			protected.POST("/clients/:id/suspend", s.SuspendClientHandler)
			protected.GET("/clients/:client_id/subscription", s.GetSubscriptionHandler)
			protected.GET("/clients/:client_id/stats", s.GetStatsHandler)
		}
	}
}

func (s *WhiteLabelService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func main() {
	cfg := Config{
		ServerPort: getEnv("WHITELABEL_PORT", "8089"),
		RedisAddr:  getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret: getEnv("JWT_SECRET", "tiger-whitelabel-secret-2026"),
	}

	service := NewWhiteLabelService(cfg)

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
			"service":   "white-label-service",
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
		log.Printf("Starting White Label Service on %s", addr)
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
