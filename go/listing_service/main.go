/**
 * TigerWallet Token Listing Service
 * Complete backend for token listing with crypto payment integration
 */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port            int    `json:"port"`
	RedisAddr       string `json:"redis_addr"`
	MongoURI        string `json:"mongo_uri"`
	PaymentServiceURL string `json:"payment_service_url"`
	AdminWallet     string `json:"admin_wallet"`
	EthRPCURL       string `json:"eth_rpc_url"`
}

var cfg = Config{
	Port:              8097,
	RedisAddr:         "localhost:6379",
	PaymentServiceURL: "http://localhost:8096",
}

// ============================================================================
// Supported Chains & Tokens
// ============================================================================

var SupportedChains = map[string]int64{
	"ethereum":   1,
	"bsc":        56,
	"polygon":    137,
	"arbitrum":   42161,
	"optimism":   10,
	"avalanche":  43114,
	"base":       8453,
}

var ChainNames = map[int64]string{
	1:     "Ethereum",
	56:    "BNB Chain",
	137:   "Polygon",
	42161: "Arbitrum",
	10:    "Optimism",
	43114: "Avalanche",
	8453:  "Base",
}

// ============================================================================
// Database Models
// ============================================================================

type TokenListing struct {
	ID              string    `json:"id" bson:"_id"`
	ApplicantEmail  string    `json:"applicant_email" bson:"applicant_email"`
	ApplicantName  string    `json:"applicant_name" bson:"applicant_name"`
	TokenSymbol    string    `json:"token_symbol" bson:"token_symbol"`
	TokenName      string    `json:"token_name" bson:"token_name"`
	ContractAddr   string    `json:"contract_address" bson:"contract_address"`
	ChainID        int64     `json:"chain_id" bson:"chain_id"`
	Chain          string    `json:"chain" bson:"chain"`
	QuoteToken     string    `json:"quote_token" bson:"quote_token"`
	Tier           string    `json:"tier" bson:"tier"`
	FeeAmount      string    `json:"fee_amount" bson:"fee_amount"`
	FeeToken       string    `json:"fee_token" bson:"fee_token"`
	FeeUSD         string    `json:"fee_usd" bson:"fee_usd"`
	PaymentID      string    `json:"payment_id" bson:"payment_id"`
	PaymentStatus  string    `json:"payment_status" bson:"payment_status"`
	Status         string    `json:"status" bson:"status"` // pending, paid, reviewing, approved, rejected
	AdminNotes     string    `json:"admin_notes" bson:"admin_notes"`
	ReviewedBy     string    `json:"reviewed_by" bson:"reviewed_by"`
	ReviewedAt     *time.Time `json:"reviewed_at" bson:"reviewed_at"`
	Website        string    `json:"website" bson:"website"`
	Twitter        string    `json:"twitter" bson:"twitter"`
	Telegram       string    `json:"telegram" bson:"telegram"`
	Discord        string    `json:"discord" bson:"discord"`
	Whitepaper     string    `json:"whitepaper" bson:"whitepaper"`
	LogoURL        string    `json:"logo_url" bson:"logo_url"`
	Description    string    `json:"description" bson:"description"`
	TotalSupply    string    `json:"total_supply" bson:"total_supply"`
	CirculatingSupply string `json:"circulating_supply" bson:"circulating_supply"`
	TokenType      string    `json:"token_type" bson:"token_type"` // ERC20, BEP20, etc
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" bson:"updated_at"`
}

type TierConfig struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	FeeAmount   string `json:"fee_amount"`
	FeeUSD      string `json:"fee_usd"`
	FeeToken    string `json:"fee_token"`
	Features    []string `json:"features"`
}

var TierConfigs = map[string]*TierConfig{
	"tier1": {
		ID:        "tier1",
		Name:      "Major Pairs",
		FeeAmount: "5000",
		FeeUSD:    "5000",
		FeeToken:  "USDT",
		Features:  []string{"Top 10 by volume", "Priority support", "Marketing boost"},
	},
	"tier2": {
		ID:        "tier2",
		Name:      "Established",
		FeeAmount: "2000",
		FeeUSD:    "2000",
		FeeToken:  "USDT",
		Features:  []string{"Good liquidity", "Standard support"},
	},
	"tier3": {
		ID:        "tier3",
		Name:      "New Tokens",
		FeeAmount: "1000",
		FeeUSD:    "1000",
		FeeToken:  "USDT",
		Features:  []string{"Growing tokens", "Basic support"},
	},
	"tier4": {
		ID:        "tier4",
		Name:      "Community",
		FeeAmount: "500",
		FeeUSD:    "500",
		FeeToken:  "USDT",
		Features:  []string{"Community tokens", "Basic listing"},
	},
}

// ============================================================================
// Listing Service
// ============================================================================

type ListingService struct {
	redis       *redis.Client
	ethClient   *ethclient.Client
	mu          sync.RWMutex
	listings    map[string]*TokenListing
	adminUsers  map[string]*AdminUser
	paymentURL  string
}

type AdminUser struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"` // superadmin, admin, reviewer
	CanApprove bool     `json:"can_approve"`
	CanEdit   bool     `json:"can_edit"`
	CanDelete bool     `json:"can_delete"`
	CreatedAt time.Time `json:"created_at"`
}

func NewListingService() *ListingService {
	return &ListingService{
		redis:      redis.NewClient(&redis.Options{Addr: cfg.RedisAddr}),
		listings:   make(map[string]*TokenListing),
		adminUsers: make(map[string]*AdminUser),
		paymentURL: cfg.PaymentServiceURL,
	}
}

func (s *ListingService) initAdminUsers() {
	// Default SuperAdmin
	s.adminUsers["superadmin@tigerwallet.com"] = &AdminUser{
		ID:        uuid.New().String(),
		Email:     "superadmin@tigerwallet.com",
		Role:      "superadmin",
		CanApprove: true,
		CanEdit:   true,
		CanDelete: true,
		CreatedAt: time.Now(),
	}
}

// ============================================================================
// API Routes
// ============================================================================

func (s *ListingService) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/listing")
	{
		// Public endpoints
		api.POST("/apply", s.SubmitApplication)
		api.GET("/status/:id", s.GetListingStatus)
		api.GET("/tiers", s.GetTiers)
		api.GET("/chains", s.GetChains)

		// Payment webhook
		api.POST("/payment/webhook", s.HandlePaymentWebhook)

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(s.AdminAuthMiddleware())
		{
			admin.GET("/listings", s.ListAllListings)
			admin.GET("/listings/:id", s.GetListingDetails)
			admin.PUT("/listings/:id/status", s.UpdateListingStatus)
			admin.PUT("/listings/:id/review", s.ReviewListing)
			admin.DELETE("/listings/:id", s.DeleteListing)
			admin.GET("/stats", s.GetListingStats)
		}

		// SuperAdmin endpoints
		superadmin := api.Group("/superadmin")
		superadmin.Use(s.SuperAdminMiddleware())
		{
			superadmin.POST("/admins", s.CreateAdminUser)
			superadmin.DELETE("/admins/:email", s.DeleteAdminUser)
			superadmin.PUT("/tiers/:id", s.UpdateTierConfig)
		}
	}

	r.GET("/health", s.HealthCheck)
}

// Submit listing application
func (s *ListingService) SubmitApplication(c *gin.Context) {
	var req struct {
		ApplicantEmail string `json:"applicant_email" binding:"required,email"`
		ApplicantName  string `json:"applicant_name" binding:"required"`
		TokenSymbol    string `json:"token_symbol" binding:"required"`
		TokenName      string `json:"token_name" binding:"required"`
		ContractAddr   string `json:"contract_address" binding:"required"`
		Chain          string `json:"chain" binding:"required"`
		QuoteToken     string `json:"quote_token" binding:"required"`
		Tier           string `json:"tier" binding:"required"`
		Website        string `json:"website"`
		Twitter        string `json:"twitter"`
		Telegram       string `json:"telegram"`
		Discord        string `json:"discord"`
		Whitepaper     string `json:"whitepaper"`
		Description    string `json:"description"`
		LogoURL        string `json:"logo_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate chain
	chainID, ok := SupportedChains[req.Chain]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain"})
		return
	}

	// Validate contract address
	if !common.IsHexAddress(req.ContractAddr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contract address"})
		return
	}

	// Validate tier
	tier, ok := TierConfigs[req.Tier]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tier"})
		return
	}

	// Verify token contract exists and is valid
	if s.ethClient != nil {
		valid, err := s.verifyTokenContract(req.Chain, req.ContractAddr)
		if err != nil || !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token contract"})
			return
		}
	}

	listing := &TokenListing{
		ID:             uuid.New().String(),
		ApplicantEmail: req.ApplicantEmail,
		ApplicantName:  req.ApplicantName,
		TokenSymbol:    req.TokenSymbol,
		TokenName:      req.TokenName,
		ContractAddr:   req.ContractAddr,
		ChainID:        chainID,
		Chain:          req.Chain,
		QuoteToken:     req.QuoteToken,
		Tier:           req.Tier,
		FeeAmount:      tier.FeeAmount,
		FeeToken:       tier.FeeToken,
		FeeUSD:         tier.FeeUSD,
		Status:         "pending",
		Website:        req.Website,
		Twitter:        req.Twitter,
		Telegram:      req.Telegram,
		Discord:       req.Discord,
		Whitepaper:    req.Whitepaper,
		Description:   req.Description,
		LogoURL:       req.LogoURL,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	s.mu.Lock()
	s.listings[listing.ID] = listing
	s.mu.Unlock()

	// Create payment request via Payment Service
	paymentReq := map[string]interface{}{
		"user_id":    listing.ID,
		"order_id":   fmt.Sprintf("LISTING-%s", listing.ID[:8]),
		"amount":     tier.FeeAmount,
		"amount_usd": tier.FeeUSD,
		"chain":      "ethereum",
		"token":      tier.FeeToken,
		"webhook_url": fmt.Sprintf("%s/api/v1/listing/payment/webhook", cfg.PaymentServiceURL),
	}

	// Create a real payment intent via the payment service HTTP API. If the
	// payment service is unreachable, record the listing as "payment_pending"
	// (no fabricated payment ID).
	paymentID, payErr := s.createPaymentIntent(paymentReq)
	if payErr != nil {
		log.Printf("listing %s: payment service error: %v", listing.ID, payErr)
		listing.PaymentID = ""
		listing.PaymentStatus = "payment_pending"
	} else {
		listing.PaymentID = paymentID
		listing.PaymentStatus = "pending"
	}

	s.mu.Lock()
	s.listings[listing.ID] = listing
	s.mu.Unlock()

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"listing": map[string]interface{}{
			"id":               listing.ID,
			"status":           listing.Status,
			"tier":             listing.Tier,
			"fee_amount":       listing.FeeAmount,
			"fee_token":        listing.FeeToken,
			"payment_id":       listing.PaymentID,
			"payment_status":   listing.PaymentStatus,
			"payment_address":  s.getPaymentAddress(listing),
			"expires_at":       time.Now().Add(24 * time.Hour),
		},
		"instructions": map[string]string{
			"title":          "Complete Payment",
			"description":    fmt.Sprintf("Send exactly %s %s to the address below", listing.FeeAmount, listing.FeeToken),
			"payment_address": s.getPaymentAddress(listing),
			"network":        "Ethereum (ERC20)",
		},
	})
}

// Verify token contract exists on chain
// verifyTokenContract performs a real on-chain check that bytecode exists at
// the given address (i.e. it is a deployed contract, not an EOA). It also
// validates the address format. If no RPC client is configured it returns
// an error rather than blindly accepting any hex string.
func (s *ListingService) verifyTokenContract(chain, address string) (bool, error) {
	if !common.IsHexAddress(address) {
		return false, fmt.Errorf("invalid address format: %s", address)
	}
	if s.ethClient == nil {
		return false, fmt.Errorf("ETH_RPC_URL not configured: cannot verify contract on chain %s", chain)
	}
	addr := common.HexToAddress(address)
	code, err := s.ethClient.CodeAt(context.Background(), addr, nil)
	if err != nil {
		return false, fmt.Errorf("failed to fetch code at %s: %v", address, err)
	}
	if len(code) == 0 {
		return false, fmt.Errorf("address %s has no contract code on chain %s", address, chain)
	}
	return true, nil
}

// getPaymentAddress returns the real configured fee-recipient wallet
// (AdminWallet). It NEVER fabricates a deposit address: users must only send
// listing fees to a real key-controlled address. If no admin wallet is
// configured, the listing cannot collect a payment and returns an error
// indicator (empty string).
func (s *ListingService) getPaymentAddress(listing *TokenListing) string {
	if cfg.AdminWallet == "" || !common.IsHexAddress(cfg.AdminWallet) {
		log.Printf("listing %s: no valid ADMIN_WALLET configured, payment unavailable", listing.ID)
		return ""
	}
	return common.HexToAddress(cfg.AdminWallet).Hex()
}

// createPaymentIntent POSTs a real payment intent to the configured payment
// service and returns the payment ID it assigned. Returns an error (not a fake
// id) if the service is unreachable or rejects the request.
func (s *ListingService) createPaymentIntent(req map[string]interface{}) (string, error) {
	if s.paymentURL == "" {
		return "", fmt.Errorf("payment service URL not configured")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	resp, err := http.Post(s.paymentURL+"/api/v1/payment/intent", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("payment service unreachable: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}
	var result struct {
		PaymentID string `json:"payment_id"`
		ID        string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("invalid payment service response: %v", err)
	}
	if result.PaymentID == "" && result.ID != "" {
		result.PaymentID = result.ID
	}
	if result.PaymentID == "" {
		return "", fmt.Errorf("payment service returned no payment id")
	}
	return result.PaymentID, nil
}

// Get listing status
func (s *ListingService) GetListingStatus(c *gin.Context) {
	id := c.Param("id")

	s.mu.RLock()
	listing, ok := s.listings[id]
	s.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"listing": listing,
	})
}

// Get tiers
func (s *ListingService) GetTiers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tiers":   TierConfigs,
	})
}

// Get supported chains
func (s *ListingService) GetChains(c *gin.Context) {
	chains := []map[string]interface{}{}
	for chain, id := range SupportedChains {
		chains = append(chains, map[string]interface{}{
			"id":   id,
			"name": ChainNames[id],
			"key":  chain,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"chains":  chains,
	})
}

// Handle payment webhook from payment service
func (s *ListingService) HandlePaymentWebhook(c *gin.Context) {
	var payload struct {
		Event       string `json:"event"`
		PaymentID   string `json:"payment_id"`
		OrderID    string `json:"order_id"`
		Status      string `json:"status"`
		TxHash     string `json:"tx_hash"`
		Amount     string `json:"amount"`
		Token      string `json:"token"`
		Chain      string `json:"chain"`
		CompletedAt string `json:"completed_at"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract listing ID from order ID
	orderID := payload.OrderID
	if !strings.HasPrefix(orderID, "LISTING-") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	// Find listing
	s.mu.RLock()
	var listing *TokenListing
	for _, l := range s.listings {
		if strings.HasPrefix(l.PaymentID, "pay_"+orderID[8:]) {
			listing = l
			break
		}
	}
	s.mu.RUnlock()

	if listing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	// Update listing status based on payment
	s.mu.Lock()
	if payload.Status == "completed" {
		listing.PaymentStatus = "paid"
		listing.Status = "paid"
	}
	listing.UpdatedAt = time.Now()
	s.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"listing": listing,
	})
}

// ============================================================================
// Admin Endpoints
// ============================================================================

func (s *ListingService) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization"})
			c.Abort()
			return
		}

		// In production, validate JWT and check admin role
		email := strings.TrimPrefix(authHeader, "Bearer ")
		
		s.mu.RLock()
		admin, ok := s.adminUsers[email]
		s.mu.RUnlock()

		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
			c.Abort()
			return
		}

		c.Set("admin_email", email)
		c.Set("admin", admin)
		c.Next()
	}
}

func (s *ListingService) SuperAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		email := strings.TrimPrefix(authHeader, "Bearer ")

		s.mu.RLock()
		admin, ok := s.adminUsers[email]
		s.mu.RUnlock()

		if !ok || admin.Role != "superadmin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "superadmin access required"})
			c.Abort()
			return
		}

		c.Set("admin_email", email)
		c.Set("admin", admin)
		c.Next()
	}
}

func (s *ListingService) ListAllListings(c *gin.Context) {
	status := c.Query("status")
	tier := c.Query("tier")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")

	s.mu.RLock()
	var result []*TokenListing
	for _, l := range s.listings {
		if status != "" && l.Status != status {
			continue
		}
		if tier != "" && l.Tier != tier {
			continue
		}
		result = append(result, l)
	}
	s.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"listings": result,
		"total":   len(result),
		"page":    page,
		"limit":   limit,
	})
}

func (s *ListingService) GetListingDetails(c *gin.Context) {
	id := c.Param("id")

	s.mu.RLock()
	listing, ok := s.listings[id]
	s.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"listing": listing,
	})
}

func (s *ListingService) UpdateListingStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	listing, ok := s.listings[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	validStatuses := map[string]bool{
		"pending":    true,
		"paid":       true,
		"reviewing":  true,
		"approved":   true,
		"rejected":   true,
	}

	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	listing.Status = req.Status
	listing.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"listing": listing,
	})
}

func (s *ListingService) ReviewListing(c *gin.Context) {
	id := c.Param("id")

	adminEmail, _ := c.Get("admin_email")

	var req struct {
		Status     string `json:"status" binding:"required"`
		AdminNotes string `json:"admin_notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	listing, ok := s.listings[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	now := time.Now()
	listing.Status = req.Status
	listing.AdminNotes = req.AdminNotes
	listing.ReviewedBy = adminEmail.(string)
	listing.ReviewedAt = &now
	listing.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"listing": listing,
	})
}

func (s *ListingService) DeleteListing(c *gin.Context) {
	id := c.Param("id")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.listings[id]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	delete(s.listings, id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "listing deleted",
	})
}

func (s *ListingService) GetListingStats(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]int{
		"total":       len(s.listings),
		"pending":     0,
		"paid":        0,
		"reviewing":   0,
		"approved":    0,
		"rejected":    0,
	}

	for _, l := range s.listings {
		stats[l.Status]++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stats":   stats,
	})
}

// ============================================================================
// SuperAdmin Endpoints
// ============================================================================

func (s *ListingService) CreateAdminUser(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Role     string `json:"role" binding:"required"`
		CanApprove bool  `json:"can_approve"`
		CanEdit  bool   `json:"can_edit"`
		CanDelete bool   `json:"can_delete"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validRoles := map[string]bool{"superadmin": true, "admin": true, "reviewer": true}
	if !validRoles[req.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}

	admin := &AdminUser{
		ID:        uuid.New().String(),
		Email:     req.Email,
		Role:      req.Role,
		CanApprove: req.CanApprove,
		CanEdit:   req.CanEdit,
		CanDelete: req.CanDelete,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.adminUsers[req.Email] = admin
	s.mu.Unlock()

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"admin":   admin,
	})
}

func (s *ListingService) DeleteAdminUser(c *gin.Context) {
	email := c.Param("email")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.adminUsers[email]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}

	// Don't allow deleting superadmin
	if email == "superadmin@tigerwallet.com" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete superadmin"})
		return
	}

	delete(s.adminUsers, email)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "admin deleted",
	})
}

func (s *ListingService) UpdateTierConfig(c *gin.Context) {
	tierID := c.Param("id")

	var req struct {
		FeeAmount string `json:"fee_amount"`
		FeeUSD    string `json:"fee_usd"`
		FeeToken  string `json:"fee_token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tier, ok := TierConfigs[tierID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "tier not found"})
		return
	}

	if req.FeeAmount != "" {
		tier.FeeAmount = req.FeeAmount
	}
	if req.FeeUSD != "" {
		tier.FeeUSD = req.FeeUSD
	}
	if req.FeeToken != "" {
		tier.FeeToken = req.FeeToken
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tier":    tier,
	})
}

// Health check
func (s *ListingService) HealthCheck(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "listing-service",
		"timestamp": time.Now().Unix(),
		"listings":  len(s.listings),
	})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.Println("TigerWallet Token Listing Service")
	log.Println("==================================")

	// Initialize service
	ls := NewListingService()

	// Initialize admin users
	ls.initAdminUsers()

	// Connect to Ethereum for contract verification
	if cfg.EthRPCURL != "" {
		client, err := ethclient.Dial(cfg.EthRPCURL)
		if err != nil {
			log.Printf("Warning: Could not connect to Ethereum: %v", err)
		} else {
			ls.ethClient = client
			log.Println("Connected to Ethereum for contract verification")
		}
	}

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	// Setup routes
	ls.SetupRoutes(r)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Listing service starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
