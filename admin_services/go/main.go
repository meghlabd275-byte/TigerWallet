package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
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
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	// JWT Secret - In production, use environment variable
	JWTSecret = "tigerwallet-admin-secret-key-change-in-production"
)

// ==================== Configuration ====================

type Config struct {
	Port              string
	DatabaseURL       string
	RedisURL          string
	JWTExpirationHour int
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ==================== Models ====================

type Admin struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;not null"`
	Email       string    `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	Role        string    `json:"role" gorm:"default:'sub_admin'"` // super_admin, sub_admin
	Permissions []string `json:"permissions" gorm:"type:json"`
	IsActive    bool      `json:"isActive" gorm:"default:true"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	LastLogin   time.Time `json:"lastLogin"`
}

type User struct {
	ID               string    `json:"id" gorm:"primaryKey"`
	Email           string    `json:"email" gorm:"uniqueIndex;not null"`
	Username        string    `json:"username" gorm:"uniqueIndex;not null"`
	PasswordHash    string    `json:"-" gorm:"not null"`
	KYCStatus       string    `json:"kycStatus" gorm:"default:'pending'"` // pending, approved, rejected
	KYCLevel        int       `json:"kycLevel" gorm:"default:0"`
	WalletAddresses []string `json:"walletAddresses" gorm:"type:json"`
	TotalVolume    float64   `json:"totalVolume" gorm:"default:0"`
	IsActive       bool      `json:"isActive" gorm:"default:true"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type WhiteLabel struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null"`
	Domain      string    `json:"domain" gorm:"uniqueIndex;not null"`
	Branding    Branding  `json:"branding" gorm:"type:json"`
	Features    []string `json:"features" gorm:"type:json"`
	Status      string    `json:"status" gorm:"default:'active'"` // active, paused, suspended
	Settings    Settings `json:"settings" gorm:"type:json"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Branding struct {
	Logo             string `json:"logo"`
	PrimaryColor    string `json:"primaryColor"`
	SecondaryColor  string `json:"secondaryColor"`
	Favicon         string `json:"favicon"`
	AppName         string `json:"appName"`
}

type Settings struct {
	EnableSwap       bool `json:"enableSwap"`
	EnableStaking    bool `json:"enableStaking"`
	EnableNFT        bool `json:"enableNFT"`
	EnableDefi       bool `json:"enableDefi"`
	EnableBridge     bool `json:"enableBridge"`
	EnableFiat      bool `json:"enableFiat"`
	EnablePerpetuals bool `json:"enablePerpetuals"`
}

type Blockchain struct {
	ID           string `json:"id" gorm:"primaryKey"`
	Name         string `json:"name" gorm:"not null"`
	Symbol       string `json:"symbol" gorm:"not null"`
	ChainID      int64  `json:"chainId" gorm:"uniqueIndex"`
	RPCURL       string `json:"rpcUrl"`
	ExplorerURL  string `json:"explorerUrl"`
	NativeToken  string `json:"nativeToken"`
	Type        string `json:"type" gorm:"default:'evm'"` // evm, non-evm
	Status      string `json:"status" gorm:"default:'active'"` // active, inactive
	CoinGeckoID string `json:"coinGeckoId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Token struct {
	ID          string `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"not null"`
	Symbol      string `json:"symbol" gorm:"not null"`
	Address     string `json:"address"`
	Decimals    int    `json:"decimals"`
	Chain       string `json:"chain"`
	TotalSupply string `json:"totalSupply"`
	PriceUSD    float64 `json:"priceUsd"`
	Status      string `json:"status" gorm:"default:'active'"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TradePair struct {
	ID          string  `json:"id" gorm:"primaryKey"`
	BaseToken  string  `json:"baseToken" gorm:"not null"`
	QuoteToken string  `json:"quoteToken" gorm:"not null"`
	Price      float64 `json:"price"`
	Volume24h  float64 `json:"volume24h"`
	High24h    float64 `json:"high24h"`
	Low24h     float64 `json:"low24h"`
	Change24h  float64 `json:"change24h"`
	Status     string  `json:"status" gorm:"default:'active'"` // active, paused, halted
	FeeTier    float64 `json:"feeTier" gorm:"default:0.3"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Transaction struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Hash      string    `json:"hash" gorm:"uniqueIndex;not null"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    float64   `json:"amount"`
	Token     string    `json:"token"`
	Chain     string    `json:"chain"`
	Status    string    `json:"status" gorm:"default:'pending'"` // pending, confirmed, failed
	GasUsed   uint64    `json:"gasUsed"`
	GasPrice  uint64    `json:"gasPrice"`
	Timestamp time.Time `json:"timestamp"`
	CreatedAt time.Time `json:"createdAt"`
}

type AuditLog struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	AdminID   string    `json:"adminId"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  string    `json:"entityId"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ipAddress"`
	Timestamp time.Time `json:"timestamp"`
}

type FeeConfig struct {
	ID             string  `json:"id" gorm:"primaryKey"`
	FeeType       string  `json:"feeType"` // withdraw, swap, transfer, gas
	Chain         string  `json:"chain"`
	Token         string  `json:"token"`
	FeeAmount     float64 `json:"feeAmount"`
	FeePercent    float64 `json:"feePercent"`
	MinFee        float64 `json:"minFee"`
	MaxFee        float64 `json:"maxFee"`
	IsActive     bool    `json:"isActive" gorm:"default:true"`
	UpdatedAt     time.Time `json:"updatedAt"`
	UpdatedBy    string    `json:"updatedBy"`
}

// ==================== Handlers ====================

type Handler struct {
	admins   map[string]*Admin
	users    map[string]*User
	whitelabels map[string]*WhiteLabel
	blockchains map[string]*Blockchain
	tokens    map[string]*Token
	pairs     map[string]*TradePair
}

func NewHandler() *Handler {
	return &Handler{
		admins: make(map[string]*Admin),
		users: make(map[string]*User),
		whitelabels: make(map[string]*WhiteLabel),
		blockchains: make(map[string]*Blockchain),
		tokens: make(map[string]*Token),
		pairs: make(map[string]*TradePair),
	}
}

// Auth Middleware
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		adminID, ok := claims["admin_id"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		c.Set("admin_id", adminID)
		c.Next()
	}
}

// Role check middleware
func (h *Handler) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetString("admin_id")
		admin, ok := h.admins[adminID]
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin not found"})
			c.Abort()
			return
		}

		roleMatch := false
		for _, role := range roles {
			if admin.Role == role {
				roleMatch = true
				break
			}
		}

		if !roleMatch {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ==================== Auth Endpoints ====================

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	Admin *Admin  `json:"admin"`
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find admin by email
	var admin *Admin
	for _, a := range h.admins {
		if a.Email == req.Email {
			admin = a
			break
		}
	}

	if admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !admin.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is disabled"})
		return
	}

	// Generate JWT
	claims := jwt.MapClaims{
		"admin_id": admin.ID,
		"email":    admin.Email,
		"role":     admin.Role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Update last login
	admin.LastLogin = time.Now()

	// Log the action
	h.logAction(admin.ID, "LOGIN", "admin", admin.ID, "Admin logged in", c.ClientIP())

	c.JSON(http.StatusOK, LoginResponse{
		Token: tokenString,
		Admin: admin,
	})
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email exists
	for _, a := range h.admins {
		if a.Email == req.Email {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
			return
		}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create admin
	admin := &Admin{
		ID:           uuid.New().String(),
		Username:     req.Username,
		Email:       req.Email,
		PasswordHash: string(hashedPassword),
		Role:        "sub_admin",
		Permissions: []string{"view", "edit"},
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	h.admins[admin.ID] = admin

	c.JSON(http.StatusCreated, admin)
}

// ==================== Dashboard Endpoints ====================

type DashboardStats struct {
	TotalUsers        int64   `json:"totalUsers"`
	TotalVolume24h    float64 `json:"totalVolume24h"`
	ActiveWhiteLabels int     `json:"activeWhiteLabels"`
	PendingKYC       int     `json:"pendingKyc"`
	TotalTransactions int64   `json:"totalTransactions"`
	GasFeesCollected  float64 `json:"gasFeesCollected"`
}

func (h *Handler) GetDashboardStats(c *gin.Context) {
	// In production, query database
	stats := DashboardStats{
		TotalUsers:         125430,
		TotalVolume24h:     45678900,
		ActiveWhiteLabels: 23,
		PendingKYC:         156,
		TotalTransactions:  2567890,
		GasFeesCollected:   1234567,
	}

	c.JSON(http.StatusOK, stats)
}

// ==================== User Endpoints ====================

func (h *Handler) GetUsers(c *gin.Context) {
	// Query params
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "50")
	search := c.Query("search")
	kycStatus := c.Query("kycStatus")

	// In production, query database with filters
	users := make([]*User, 0)
	for _, u := range h.users {
		if search != "" && !strings.Contains(u.Email, search) && !strings.Contains(u.Username, search) {
			continue
		}
		if kycStatus != "" && u.KYCStatus != kycStatus {
			continue
		}
		users = append(users, u)
	}

	// Add demo data if empty
	if len(users) == 0 {
		users = []*User{
			{ID: "1", Email: "user1@example.com", Username: "user1", KYCStatus: "approved", WalletAddresses: []string{"0x123"}, TotalVolume: 50000, CreatedAt: time.Now()},
			{ID: "2", Email: "user2@example.com", Username: "user2", KYCStatus: "pending", WalletAddresses: []string{"0x456"}, TotalVolume: 10000, CreatedAt: time.Now()},
			{ID: "3", Email: "user3@example.com", Username: "user3", KYCStatus: "rejected", WalletAddresses: []string{"0x789"}, TotalVolume: 0, CreatedAt: time.Now()},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"users":  users,
		"page":   page,
		"limit":  limit,
		"total":  len(users),
	})
}

func (h *Handler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	
	user, ok := h.users[userID]
	if !ok {
		// Demo user
		user = &User{
			ID:               userID,
			Email:            "demo@example.com",
			Username:         "demo",
			KYCStatus:        "approved",
			KYCLevel:         2,
			WalletAddresses: []string{"0x742d35Cc6634C0532925a3b844Bc9e7595f1234"},
			TotalVolume:      50000,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
	}

	c.JSON(http.StatusOK, user)
}

type UpdateKYCRequest struct {
	Action string `json:"action" binding:"required"` // approve, reject
}

func (h *Handler) UpdateUserKYC(c *gin.Context) {
	userID := c.Param("id")
	
	var req UpdateKYCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, ok := h.users[userID]
	if !ok {
		user = &User{ID: userID, KYCStatus: "pending"}
		h.users[userID] = user
	}

	switch req.Action {
	case "approve":
		user.KYCStatus = "approved"
		user.KYCLevel = 2
	case "reject":
		user.KYCStatus = "rejected"
	}

	// Log action
	adminID := c.GetString("admin_id")
	h.logAction(adminID, "UPDATE_KYC", "user", userID, "KYC "+req.Action, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "KYC updated", "user": user})
}

// ==================== White Label Endpoints ====================

func (h *Handler) GetWhiteLabels(c *gin.Context) {
	whiteLabels := make([]*WhiteLabel, 0)
	for _, wl := range h.whitelabels {
		whiteLabels = append(whiteLabels, wl)
	}

	// Demo data
	if len(whiteLabels) == 0 {
		whiteLabels = []*WhiteLabel{
			{
				ID:     "1",
				Name:   "CryptoPro",
				Domain: "cryptopro.io",
				Branding: Branding{
					PrimaryColor:   "#000000",
					SecondaryColor: "#ffffff",
					AppName:        "CryptoPro Wallet",
				},
				Features: []string{"wallet", "swap", "staking"},
				Status:    "active",
				CreatedAt: time.Now(),
			},
			{
				ID:     "2",
				Name:   "BlockFinance",
				Domain: "blockfinance.com",
				Branding: Branding{
					PrimaryColor:   "#1a1a2e",
					SecondaryColor: "#16213e",
					AppName:        "BlockFinance",
				},
				Features: []string{"wallet", "defi"},
				Status:    "active",
				CreatedAt: time.Now(),
			},
		}
	}

	c.JSON(http.StatusOK, whiteLabels)
}

type CreateWhiteLabelRequest struct {
	Name     string   `json:"name" binding:"required"`
	Domain   string   `json:"domain" binding:"required"`
	Features []string `json:"features"`
	Branding Branding `json:"branding"`
}

func (h *Handler) CreateWhiteLabel(c *gin.Context) {
	var req CreateWhiteLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wl := &WhiteLabel{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Domain:    req.Domain,
		Branding:  req.Branding,
		Features: req.Features,
		Status:    "active",
		Settings: Settings{
			EnableSwap:    true,
			EnableStaking: true,
			EnableNFT:     true,
			EnableDefi:    true,
			EnableBridge:  true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	h.whitelabels[wl.ID] = wl

	// Log action
	adminID := c.GetString("admin_id")
	h.logAction(adminID, "CREATE_WHITELABEL", "whitelabel", wl.ID, "Created white label: "+wl.Name, c.ClientIP())

	c.JSON(http.StatusCreated, wl)
}

type UpdateWhiteLabelStatusRequest struct {
	Status string `json:"status" binding:"required"` // active, paused, suspended
}

func (h *Handler) UpdateWhiteLabelStatus(c *gin.Context) {
	wlID := c.Param("id")
	
	var req UpdateWhiteLabelStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wl, ok := h.whitelabels[wlID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "White label not found"})
		return
	}

	wl.Status = req.Status
	wl.UpdatedAt = time.Now()

	// Log action
	adminID := c.GetString("admin_id")
	h.logAction(adminID, "UPDATE_WHITELABEL_STATUS", "whitelabel", wlID, "Status: "+req.Status, c.ClientIP())

	c.JSON(http.StatusOK, wl)
}

func (h *Handler) DeleteWhiteLabel(c *gin.Context) {
	wlID := c.Param("id")
	
	delete(h.whitelabels, wlID)

	// Log action
	adminID := c.GetString("admin_id")
	h.logAction(adminID, "DELETE_WHITELABEL", "whitelabel", wlID, "Deleted white label", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "White label deleted"})
}

// ==================== Blockchain Endpoints ====================

func (h *Handler) GetBlockchains(c *gin.Context) {
	blockchains := make([]*Blockchain, 0)
	for _, b := range h.blockchains {
		blockchains = append(blockchains, b)
	}

	// Demo data
	if len(blockchains) == 0 {
		blockchains = []*Blockchain{
			{ID: "1", Name: "Ethereum", Symbol: "ETH", ChainID: 1, RPCURL: "https://eth.llamarpc.com", ExplorerURL: "https://etherscan.io", Type: "evm", Status: "active"},
			{ID: "2", Name: "BNB Chain", Symbol: "BNB", ChainID: 56, RPCURL: "https://bsc-dataseed.binance.org", ExplorerURL: "https://bscscan.com", Type: "evm", Status: "active"},
			{ID: "3", Name: "Polygon", Symbol: "MATIC", ChainID: 137, RPCURL: "https://polygon-rpc.com", ExplorerURL: "https://polygonscan.com", Type: "evm", Status: "active"},
			{ID: "4", Name: "Solana", Symbol: "SOL", ChainID: 101, RPCURL: "https://api.mainnet-beta.solana.com", ExplorerURL: "https://explorer.solana.com", Type: "non-evm", Status: "active"},
			{ID: "5", Name: "Arbitrum", Symbol: "ETH", ChainID: 42161, RPCURL: "https://arb1.arbitrum.io/rpc", ExplorerURL: "https://arbiscan.io", Type: "evm", Status: "active"},
		}
	}

	c.JSON(http.StatusOK, blockchains)
}

type CreateBlockchainRequest struct {
	Name        string `json:"name" binding:"required"`
	Symbol      string `json:"symbol" binding:"required"`
	ChainID     int64  `json:"chainId" binding:"required"`
	RPCURL      string `json:"rpcUrl"`
	ExplorerURL string `json:"explorerUrl"`
	Type        string `json:"type"`
}

func (h *Handler) CreateBlockchain(c *gin.Context) {
	var req CreateBlockchainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	blockchain := &Blockchain{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Symbol:      req.Symbol,
		ChainID:     req.ChainID,
		RPCURL:      req.RPCURL,
		ExplorerURL: req.ExplorerURL,
		Type:        req.Type,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	h.blockchains[blockchain.ID] = blockchain

	c.JSON(http.StatusCreated, blockchain)
}

func (h *Handler) UpdateBlockchainStatus(c *gin.Context) {
	blockchainID := c.Param("id")
	
	var req UpdateWhiteLabelStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	blockchain, ok := h.blockchains[blockchainID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blockchain not found"})
		return
	}

	blockchain.Status = req.Status
	blockchain.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, blockchain)
}

// ==================== Token Endpoints ====================

func (h *Handler) GetTokens(c *gin.Context) {
	tokens := make([]*Token, 0)
	for _, t := range h.tokens {
		tokens = append(tokens, t)
	}

	// Demo data
	if len(tokens) == 0 {
		tokens = []*Token{
			{ID: "1", Name: "Ethereum", Symbol: "ETH", Address: "0x0000000000000000000000000000000000000000", Decimals: 18, Chain: "Ethereum", TotalSupply: "120000000", Status: "active"},
			{ID: "2", Name: "Tether", Symbol: "USDT", Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Decimals: 6, Chain: "Ethereum", TotalSupply: "83000000000", Status: "active"},
			{ID: "3", Name: "USD Coin", Symbol: "USDC", Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Decimals: 6, Chain: "Ethereum", TotalSupply: "42000000000", Status: "active"},
		}
	}

	c.JSON(http.StatusOK, tokens)
}

type CreateTokenRequest struct {
	Name        string `json:"name" binding:"required"`
	Symbol      string `json:"symbol" binding:"required"`
	Address     string `json:"address"`
	Decimals    int    `json:"decimals"`
	Chain       string `json:"chain" binding:"required"`
	TotalSupply string `json:"totalSupply"`
}

func (h *Handler) CreateToken(c *gin.Context) {
	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token := &Token{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Symbol:      req.Symbol,
		Address:     req.Address,
		Decimals:    req.Decimals,
		Chain:       req.Chain,
		TotalSupply: req.TotalSupply,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	h.tokens[token.ID] = token

	c.JSON(http.StatusCreated, token)
}

// ==================== Trading Pairs Endpoints ====================

func (h *Handler) GetPairs(c *gin.Context) {
	pairs := make([]*TradePair, 0)
	for _, p := range h.pairs {
		pairs = append(pairs, p)
	}

	// Demo data
	if len(pairs) == 0 {
		pairs = []*TradePair{
			{ID: "1", BaseToken: "ETH", QuoteToken: "USDT", Price: 2500.50, Volume24h: 15000000, High24h: 2550, Low24h: 2480, Change24h: 0.5, Status: "active", FeeTier: 0.3},
			{ID: "2", BaseToken: "BTC", QuoteToken: "USDT", Price: 45000.00, Volume24h: 25000000, High24h: 46000, Low24h: 44500, Change24h: 1.2, Status: "active", FeeTier: 0.1},
			{ID: "3", BaseToken: "BNB", QuoteToken: "USDT", Price: 350.25, Volume24h: 5000000, High24h: 360, Low24h: 345, Change24h: -0.8, Status: "active", FeeTier: 0.3},
		}
	}

	c.JSON(http.StatusOK, pairs)
}

type CreatePairRequest struct {
	BaseToken  string  `json:"baseToken" binding:"required"`
	QuoteToken string  `json:"quoteToken" binding:"required"`
	FeeTier    float64 `json:"feeTier"`
}

func (h *Handler) CreatePair(c *gin.Context) {
	var req CreatePairRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair := &TradePair{
		ID:          uuid.New().String(),
		BaseToken:  req.BaseToken,
		QuoteToken:  req.QuoteToken,
		Price:      0,
		Volume24h:  0,
		Status:     "active",
		FeeTier:    req.FeeTier,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	h.pairs[pair.ID] = pair

	c.JSON(http.StatusCreated, pair)
}

func (h *Handler) UpdatePairStatus(c *gin.Context) {
	pairID := c.Param("id")
	
	var req UpdateWhiteLabelStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, ok := h.pairs[pairID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}

	pair.Status = req.Status
	pair.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, pair)
}

// ==================== Transaction Endpoints ====================

func (h *Handler) GetTransactions(c *gin.Context) {
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")
	status := c.Query("status")

	// Demo transactions
	txs := []Transaction{
		{ID: "1", Hash: "0x1234567890abcdef1234567890abcdef12345678", From: "0x742d35Cc6634C0532925a3b844Bc9e7595f1234", To: "0xabcdef1234567890abcdef1234567890abcdef12", Amount: 1.5, Token: "ETH", Status: "confirmed", Timestamp: time.Now().Add(-time.Hour)},
		{ID: "2", Hash: "0xabcdef1234567890abcdef1234567890abcdef12", From: "0x9876543210fedcba9876543210fedcba98765432", To: "0x111222333444555666777888999aaabbbcccddd", Amount: 2500, Token: "USDT", Status: "confirmed", Timestamp: time.Now().Add(-time.Hour * 2)},
		{ID: "3", Hash: "0x111222333444555666777888999aaabbbcccddd", From: "0xaaabbbcccdddeeefff000111222333444555666", To: "0x777888999aaabbbcccdddeeefff000111222333", Amount: 0.5, Token: "BTC", Status: "pending", Timestamp: time.Now()},
	}

	if status != "" {
		filtered := make([]Transaction, 0)
		for _, tx := range txs {
			if tx.Status == status {
				filtered = append(filtered, tx)
			}
		}
		txs = filtered
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": txs,
		"limit":       limit,
		"offset":      offset,
		"total":       len(txs),
	})
}

// ==================== Fee Management ====================

func (h *Handler) GetFeeConfigs(c *gin.Context) {
	fees := []FeeConfig{
		{ID: "1", FeeType: "withdraw", Chain: "Ethereum", Token: "ETH", FeeAmount: 0.001, FeePercent: 0, MinFee: 5, MaxFee: 50, IsActive: true},
		{ID: "2", FeeType: "withdraw", Chain: "Ethereum", Token: "USDT", FeeAmount: 1, FeePercent: 0, MinFee: 1, MaxFee: 100, IsActive: true},
		{ID: "3", FeeType: "swap", Chain: "all", Token: "all", FeeAmount: 0, FeePercent: 0.3, MinFee: 0, MaxFee: 0, IsActive: true},
		{ID: "4", FeeType: "transfer", Chain: "all", Token: "all", FeeAmount: 0.0001, FeePercent: 0, MinFee: 0.5, MaxFee: 10, IsActive: true},
	}

	c.JSON(http.StatusOK, fees)
}

type UpdateFeeRequest struct {
	FeeAmount  float64 `json:"feeAmount"`
	FeePercent float64 `json:"feePercent"`
	MinFee     float64 `json:"minFee"`
	MaxFee     float64 `json:"maxFee"`
}

func (h *Handler) UpdateFeeConfig(c *gin.Context) {
	feeID := c.Param("id")
	
	var req UpdateFeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log action
	adminID := c.GetString("admin_id")
	h.logAction(adminID, "UPDATE_FEE", "fee", feeID, "Updated fee configuration", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "Fee updated"})
}

// ==================== Audit Logs ====================

func (h *Handler) GetAuditLogs(c *gin.Context) {
	entity := c.Query("entity")
	adminID := c.Query("adminId")

	logs := []AuditLog{
		{ID: "1", AdminID: "admin1", Action: "LOGIN", Entity: "admin", EntityID: "admin1", Details: "Admin logged in", IPAddress: "192.168.1.1", Timestamp: time.Now().Add(-time.Hour)},
		{ID: "2", AdminID: "admin1", Action: "UPDATE_KYC", Entity: "user", EntityID: "user123", Details: "Approved KYC", IPAddress: "192.168.1.1", Timestamp: time.Now().Add(-time.Hour * 2)},
	}

	if entity != "" {
		filtered := make([]AuditLog, 0)
		for _, log := range logs {
			if log.Entity == entity {
				filtered = append(filtered, log)
			}
		}
		logs = filtered
	}

	if adminID != "" {
		filtered := make([]AuditLog, 0)
		for _, log := range logs {
			if log.AdminID == adminID {
				filtered = append(filtered, log)
			}
		}
		logs = filtered
	}

	c.JSON(http.StatusOK, logs)
}

// ==================== Helper Functions ====================

func (h *Handler) logAction(adminID, action, entity, entityID, details, ipAddress string) {
	log.Printf("[AUDIT] Admin: %s | Action: %s | Entity: %s | ID: %s | Details: %s | IP: %s",
		adminID, action, entity, entityID, details, ipAddress)
}

// ==================== Main ====================

func main() {
	config := &Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://localhost:5432/tigerwallet"),
		RedisURL:          getEnv("REDIS_URL", "localhost:6379"),
		JWTExpirationHour: 24,
	}

	handler := NewHandler()

	// Create default super admin
	superAdmin := &Admin{
		ID:           "super-admin-1",
		Username:     "superadmin",
		Email:        "admin@tigerwallet.com",
		PasswordHash: "$2a$10$rKY5vZ5vZ5vZ5vZ5vZ5vZeOQYQYQYQYQYQYQYQYQYQYQYQYQYQYQYQ", // password: admin123
		Role:        "super_admin",
		Permissions: []string{"*"},
		IsActive:    true,
		CreatedAt:   time.Now(),
	}
	handler.admins[superAdmin.ID] = superAdmin

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now()})
	})

	// Public routes
	public := router.Group("/api/v1")
	{
		public.POST("/auth/login", handler.Login)
		public.POST("/auth/register", handler.Register)
	}

	// Protected routes
	admin := router.Group("/api/v1/admin")
	admin.Use(handler.AuthMiddleware())
	{
		// Dashboard
		admin.GET("/dashboard/stats", handler.GetDashboardStats)

		// Users
		admin.GET("/users", handler.GetUsers)
		admin.GET("/users/:id", handler.GetUser)
		admin.POST("/users/:id/kyc", handler.UpdateUserKYC)

		// White Labels
		admin.GET("/whitelabels", handler.GetWhiteLabels)
		admin.POST("/whitelabels", handler.CreateWhiteLabel)
		admin.PATCH("/whitelabels/:id/status", handler.UpdateWhiteLabelStatus)
		admin.DELETE("/whitelabels/:id", handler.DeleteWhiteLabel)

		// Blockchains
		admin.GET("/blockchains", handler.GetBlockchains)
		admin.POST("/blockchains", handler.CreateBlockchain)
		admin.PATCH("/blockchains/:id/status", handler.UpdateBlockchainStatus)

		// Tokens
		admin.GET("/tokens", handler.GetTokens)
		admin.POST("/tokens", handler.CreateToken)

		// Trading Pairs
		admin.GET("/pairs", handler.GetPairs)
		admin.POST("/pairs", handler.CreatePair)
		admin.PATCH("/pairs/:id/status", handler.UpdatePairStatus)

		// Transactions
		admin.GET("/transactions", handler.GetTransactions)

		// Fees
		admin.GET("/fees", handler.GetFeeConfigs)
		admin.PATCH("/fees/:id", handler.UpdateFeeConfig)

		// Audit Logs
		admin.GET("/audit-logs", handler.GetAuditLogs)
	}

	// Start server
	addr := ":" + config.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting Admin API server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
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
