/**
 * TigerWallet Token Listing Service
 * Complete backend for token listing with crypto payment integration
 */

package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// jwtSecret is the shared HS256 secret used by wallet_api to sign JWTs. The
// listing service validates tokens issued by wallet_api so admins are
// authenticated through the same realm as regular users. Override via
// JWT_SECRET env (must match wallet_api).
var jwtSecret = getEnv("JWT_SECRET", "")

func getEnvInt(key string, dflt int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return dflt
}

func getEnv(key, dflt string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return dflt
}

// listingWebhookSecret is the shared secret the payment service signs webhook
// bodies with (the same value the payment service reads). It is read per
// request so a rotated value takes effect without a restart. An empty value
// disables the endpoint rather than accepting unsigned callers.
func listingWebhookSecret() string {
	return strings.TrimSpace(os.Getenv("PAYMENT_WEBHOOK_SECRET"))
}

// parseAdminJWT validates an HS256 JWT and returns the subject. The subject
// is expected to be the admin's email (wallet_api issues sub = user id, but
// for admins the listing service maps sub->email via adminUsers).
func parseAdminJWT(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", errors.New("missing subject")
	}
	return sub, nil
}

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port              int    `json:"port"`
	RedisAddr         string `json:"redis_addr"`
	MongoURI          string `json:"mongo_uri"`
	PaymentServiceURL string `json:"payment_service_url"`
	AdminWallet       string `json:"admin_wallet"`
	EthRPCURL         string `json:"eth_rpc_url"`
}

func loadListingConfig() Config {
	return Config{
		Port:              getEnvInt("PORT", 8097),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		PaymentServiceURL: getEnv("PAYMENT_SERVICE_URL", "http://localhost:8096"),
		EthRPCURL:         getEnv("ETH_RPC_URL", ""),
		AdminWallet:       getEnv("ADMIN_WALLET", ""),
	}
}

var cfg = loadListingConfig()

// ============================================================================
// Supported Chains & Tokens
// ============================================================================

var SupportedChains = map[string]int64{
	"ethereum":  1,
	"bsc":       56,
	"polygon":   137,
	"arbitrum":  42161,
	"optimism":  10,
	"avalanche": 43114,
	"base":      8453,
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
	ID                string     `json:"id" bson:"_id"`
	ApplicantEmail    string     `json:"applicant_email" bson:"applicant_email"`
	ApplicantName     string     `json:"applicant_name" bson:"applicant_name"`
	TokenSymbol       string     `json:"token_symbol" bson:"token_symbol"`
	TokenName         string     `json:"token_name" bson:"token_name"`
	ContractAddr      string     `json:"contract_address" bson:"contract_address"`
	ChainID           int64      `json:"chain_id" bson:"chain_id"`
	Chain             string     `json:"chain" bson:"chain"`
	QuoteToken        string     `json:"quote_token" bson:"quote_token"`
	Tier              string     `json:"tier" bson:"tier"`
	FeeAmount         string     `json:"fee_amount" bson:"fee_amount"`
	FeeToken          string     `json:"fee_token" bson:"fee_token"`
	FeeUSD            string     `json:"fee_usd" bson:"fee_usd"`
	PaymentID         string     `json:"payment_id" bson:"payment_id"`
	PaymentStatus     string     `json:"payment_status" bson:"payment_status"`
	PaymentTxHash     string     `json:"payment_tx_hash" bson:"payment_tx_hash"`
	PaymentAddress    string     `json:"payment_address" bson:"payment_address"`
	Status            string     `json:"status" bson:"status"` // pending, paid, reviewing, approved, rejected
	AdminNotes        string     `json:"admin_notes" bson:"admin_notes"`
	ReviewedBy        string     `json:"reviewed_by" bson:"reviewed_by"`
	ReviewedAt        *time.Time `json:"reviewed_at" bson:"reviewed_at"`
	Website           string     `json:"website" bson:"website"`
	Twitter           string     `json:"twitter" bson:"twitter"`
	Telegram          string     `json:"telegram" bson:"telegram"`
	Discord           string     `json:"discord" bson:"discord"`
	Whitepaper        string     `json:"whitepaper" bson:"whitepaper"`
	LogoURL           string     `json:"logo_url" bson:"logo_url"`
	Description       string     `json:"description" bson:"description"`
	TotalSupply       string     `json:"total_supply" bson:"total_supply"`
	CirculatingSupply string     `json:"circulating_supply" bson:"circulating_supply"`
	TokenType         string     `json:"token_type" bson:"token_type"` // ERC20, BEP20, etc
	CreatedAt         time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" bson:"updated_at"`
}

type TierConfig struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	FeeAmount string   `json:"fee_amount"`
	FeeUSD    string   `json:"fee_usd"`
	FeeToken  string   `json:"fee_token"`
	Features  []string `json:"features"`
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
	redis      *redis.Client
	ethClient  *ethclient.Client
	mu         sync.RWMutex
	listings   map[string]*TokenListing
	adminUsers map[string]*AdminUser
	paymentURL string
}

type AdminUser struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Role       string    `json:"role"` // superadmin, admin, reviewer
	CanApprove bool      `json:"can_approve"`
	CanEdit    bool      `json:"can_edit"`
	CanDelete  bool      `json:"can_delete"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewListingService() *ListingService {
	return &ListingService{
		redis:      redis.NewClient(&redis.Options{Addr: cfg.RedisAddr}),
		listings:   make(map[string]*TokenListing),
		adminUsers: make(map[string]*AdminUser),
		paymentURL: cfg.PaymentServiceURL,
	}
}

// listingKey namespaces a single listing inside Redis.
func listingKey(id string) string { return "listing:" + id }

// storeListing writes a listing to Redis and updates the in-process cache. The
// map alone is not enough: this service is deployed as a container that can be
// restarted or rescheduled, and an application whose listing disappears when
// that happens loses the record of a fee the applicant already paid. Redis is
// the durable copy; the map is a read-through cache.
func (s *ListingService) storeListing(ctx context.Context, listing *TokenListing) error {
	payload, err := json.Marshal(listing)
	if err != nil {
		return fmt.Errorf("cannot encode listing %s: %w", listing.ID, err)
	}
	// No TTL: a listing is a financial record, not a cache entry. It is
	// removed only by an explicit delete.
	if err := s.redis.Set(ctx, listingKey(listing.ID), payload, 0).Err(); err != nil {
		return fmt.Errorf("cannot persist listing %s: %w", listing.ID, err)
	}
	s.mu.Lock()
	s.listings[listing.ID] = listing
	s.mu.Unlock()
	return nil
}

// loadListing returns a listing from the cache, falling back to Redis. The
// fallback is what makes a restart survivable: a replica that did not handle
// the original request, or one that just started, still answers correctly.
//
// The returned value is a copy. Callers that need to modify a listing mutate
// their copy and hand it back through storeListing, so a read never observes a
// half-updated record and no lock has to be held across request handling.
func (s *ListingService) loadListing(ctx context.Context, id string) (*TokenListing, bool) {
	s.mu.RLock()
	cached, ok := s.listings[id]
	s.mu.RUnlock()
	if ok {
		snapshot := *cached
		return &snapshot, true
	}

	payload, err := s.redis.Get(ctx, listingKey(id)).Bytes()
	if err != nil {
		return nil, false
	}
	var listing TokenListing
	if err := json.Unmarshal(payload, &listing); err != nil {
		log.Printf("listing %s in the store is not readable: %v", id, err)
		return nil, false
	}

	s.mu.Lock()
	s.listings[id] = &listing
	s.mu.Unlock()

	snapshot := listing
	return &snapshot, true
}

// deleteListing removes a listing from Redis and the cache.
func (s *ListingService) deleteListing(ctx context.Context, id string) error {
	if err := s.redis.Del(ctx, listingKey(id)).Err(); err != nil {
		return fmt.Errorf("cannot delete listing %s: %w", id, err)
	}
	s.mu.Lock()
	delete(s.listings, id)
	s.mu.Unlock()
	return nil
}

// listListings returns every listing matching the filters, reading from Redis so
// the result does not depend on which replica served earlier requests.
func (s *ListingService) listListings(ctx context.Context, status, tier string) ([]*TokenListing, error) {
	keys, err := s.redis.Keys(ctx, "listing:*").Result()
	if err != nil {
		return nil, fmt.Errorf("cannot enumerate listings: %w", err)
	}
	if len(keys) == 0 {
		return nil, nil
	}

	values, err := s.redis.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("cannot read listings: %w", err)
	}

	result := make([]*TokenListing, 0, len(values))
	for _, raw := range values {
		text, ok := raw.(string)
		if !ok {
			continue
		}
		var listing TokenListing
		if err := json.Unmarshal([]byte(text), &listing); err != nil {
			log.Printf("a listing in the store is not readable: %v", err)
			continue
		}
		if status != "" && listing.Status != status {
			continue
		}
		if tier != "" && listing.Tier != tier {
			continue
		}
		result = append(result, &listing)
	}
	return result, nil
}

func (s *ListingService) initAdminUsers() {
	// The superadmin identity is deployment configuration, not a constant.
	//
	// Admin rights here are decided purely by the JWT subject, and the JWT is
	// issued by wallet_api. A hardcoded address would therefore hand superadmin
	// to anyone who manages to register that email on wallet_api. Requiring an
	// operator to name the address keeps the grant out of reach of self-service
	// registration, and with nothing configured no admin exists at all, so every
	// admin route fails closed rather than falling open.
	email := getEnv("LISTING_SUPERADMIN_EMAIL", "")
	if email == "" {
		log.Printf("WARN: LISTING_SUPERADMIN_EMAIL is unset; every admin route will reject callers")
		return
	}

	s.adminUsers[email] = &AdminUser{
		ID:         uuid.New().String(),
		Email:      email,
		Role:       "superadmin",
		CanApprove: true,
		CanEdit:    true,
		CanDelete:  true,
		CreatedAt:  time.Now(),
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
		Telegram:       req.Telegram,
		Discord:        req.Discord,
		Whitepaper:     req.Whitepaper,
		Description:    req.Description,
		LogoURL:        req.LogoURL,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// The listing is persisted before the payment intent is requested. An
	// applicant who is shown a payment address must be findable afterwards,
	// even if the process dies between the two steps.
	if err := s.storeListing(c.Request.Context(), listing); err != nil {
		log.Printf("cannot store the new listing: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the listing could not be saved, please retry"})
		return
	}

	// Create payment request via Payment Service
	paymentReq := map[string]interface{}{
		"user_id":     listing.ID,
		"order_id":    fmt.Sprintf("LISTING-%s", listing.ID[:8]),
		"amount":      tier.FeeAmount,
		"amount_usd":  tier.FeeUSD,
		"chain":       "ethereum",
		"token":       tier.FeeToken,
		"webhook_url": fmt.Sprintf("%s/api/v1/listing/payment/webhook", cfg.PaymentServiceURL),
	}

	// Create a real payment intent via the payment service HTTP API. If the
	// payment service is unreachable, record the listing as "payment_pending"
	// (no fabricated payment id or address).
	intent, payErr := s.createPaymentIntent(paymentReq)
	if payErr != nil {
		log.Printf("listing %s: payment service error: %v", listing.ID, payErr)
		listing.PaymentID = ""
		listing.PaymentAddress = ""
		listing.PaymentStatus = "payment_pending"
	} else {
		listing.PaymentID = intent.PaymentID
		listing.PaymentAddress = intent.PaymentAddress
		listing.PaymentStatus = "pending"
	}

	// Persist the payment outcome so the record the applicant is shown matches
	// what the store holds.
	if err := s.storeListing(c.Request.Context(), listing); err != nil {
		log.Printf("listing %s: cannot record the payment outcome: %v", listing.ID, err)
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"listing": map[string]interface{}{
			"id":              listing.ID,
			"status":          listing.Status,
			"tier":            listing.Tier,
			"fee_amount":      listing.FeeAmount,
			"fee_token":       listing.FeeToken,
			"payment_id":      listing.PaymentID,
			"payment_status":  listing.PaymentStatus,
			"payment_address": s.getPaymentAddress(listing),
			"expires_at":      time.Now().Add(24 * time.Hour),
		},
		"instructions": map[string]string{
			"title":           "Complete Payment",
			"description":     fmt.Sprintf("Send exactly %s %s to the address below", listing.FeeAmount, listing.FeeToken),
			"payment_address": s.getPaymentAddress(listing),
			"network":         "Ethereum (ERC20)",
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

// getPaymentAddress returns the address the applicant must pay to. The address
// issued by the payment service for this listing takes precedence: it is the one
// the chain listener watches, so paying anywhere else is never matched. The
// admin wallet is only a fallback for when no per-listing address was issued.
func (s *ListingService) getPaymentAddress(listing *TokenListing) string {
	if listing != nil && listing.PaymentAddress != "" {
		if common.IsHexAddress(listing.PaymentAddress) {
			return common.HexToAddress(listing.PaymentAddress).Hex()
		}
		log.Printf("listing %s: the payment service issued an unusable address %q", listing.ID, listing.PaymentAddress)
	}
	if cfg.AdminWallet == "" || !common.IsHexAddress(cfg.AdminWallet) {
		if listing != nil {
			log.Printf("listing %s: no valid ADMIN_WALLET configured, payment unavailable", listing.ID)
		}
		return ""
	}
	return common.HexToAddress(cfg.AdminWallet).Hex()
}

// paymentIntent is what the payment service returns for a deposit request.
type paymentIntent struct {
	PaymentID      string `json:"payment_id"`
	PaymentAddress string `json:"payment_address"`
	ExpiresAt      string `json:"expires_at"`
}

// createPaymentIntent asks the payment service for a deposit address and
// returns it. The address is what the applicant pays to, so a failure here
// leaves the listing without a way to pay and is reported as such rather than
// being papered over with a fabricated id.
func (s *ListingService) createPaymentIntent(req map[string]interface{}) (*paymentIntent, error) {
	if s.paymentURL == "" {
		return nil, fmt.Errorf("payment service URL not configured")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(s.paymentURL+"/api/v1/payment/deposit/create", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("payment service unreachable: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("cannot read the payment service response: %v", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("payment service returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	// The deposit endpoint nests the payment under "payment"; accept a flat
	// body too so a version skew does not silently drop the address.
	var decoded struct {
		PaymentID      string `json:"payment_id"`
		PaymentAddress string `json:"payment_address"`
		ExpiresAt      string `json:"expires_at"`
		Payment        struct {
			ID             string `json:"id"`
			PaymentAddress string `json:"payment_address"`
			ExpiresAt      string `json:"expires_at"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, fmt.Errorf("invalid payment service response: %v", err)
	}

	out := &paymentIntent{
		PaymentID:      decoded.PaymentID,
		PaymentAddress: decoded.PaymentAddress,
		ExpiresAt:      decoded.ExpiresAt,
	}
	if decoded.Payment.ID != "" {
		out.PaymentID = decoded.Payment.ID
	}
	if decoded.Payment.PaymentAddress != "" {
		out.PaymentAddress = decoded.Payment.PaymentAddress
	}
	if decoded.Payment.ExpiresAt != "" {
		out.ExpiresAt = decoded.Payment.ExpiresAt
	}

	if out.PaymentID == "" {
		return nil, fmt.Errorf("payment service returned no payment id")
	}
	if out.PaymentAddress == "" {
		// Without an address the applicant cannot pay. Treat this as a failure
		// so the listing is not shown a payment step that cannot be completed.
		return nil, fmt.Errorf("payment service returned no deposit address")
	}
	return out, nil
}

// Get listing status
func (s *ListingService) GetListingStatus(c *gin.Context) {
	id := c.Param("id")

	listing, ok := s.loadListing(c.Request.Context(), id)
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
	// This endpoint is the only path that moves a listing to "paid", and it is
	// reachable without a session, so the caller must prove it is the payment
	// service. Without a signature anyone who learns a listing ID could mark
	// their own application paid and skip the fee.
	secret := listingWebhookSecret()
	if secret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment processor not configured"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil || len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	signature := strings.TrimPrefix(c.GetHeader("X-Payment-Signature"), "sha256=")
	if signature == "" {
		signature = c.GetHeader("X-Signature-256")
	}
	computed := hmac.New(sha256.New, []byte(secret))
	computed.Write(body)
	if !hmac.Equal([]byte(signature), []byte(hex.EncodeToString(computed.Sum(nil)))) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var payload struct {
		Event       string `json:"event"`
		PaymentID   string `json:"payment_id"`
		OrderID     string `json:"order_id"`
		Status      string `json:"status"`
		TxHash      string `json:"tx_hash"`
		Amount      string `json:"amount"`
		Token       string `json:"token"`
		Chain       string `json:"chain"`
		CompletedAt string `json:"completed_at"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	// Extract listing ID from order ID
	orderID := payload.OrderID
	if !strings.HasPrefix(orderID, "LISTING-") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	// Find the listing the order belongs to. The order id carries the first
	// eight characters of the listing id (see SubmitApplication), so match on
	// that. Matching on the payment id would never succeed: the payment service
	// returns plain UUIDs, not "pay_"-prefixed ids.
	suffix := orderID[len("LISTING-"):]
	if suffix == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	all, err := s.listListings(c.Request.Context(), "", "")
	if err != nil {
		log.Printf("payment webhook: cannot read listings: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the listings could not be read, please retry"})
		return
	}
	var listing *TokenListing
	for _, l := range all {
		if strings.HasPrefix(l.ID, suffix) {
			listing = l
			break
		}
	}

	if listing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	// The payment must belong to this listing. The service echoes the id it
	// issued at application time; a callback for a different intent is ignored.
	if payload.PaymentID != "" && listing.PaymentID != "" && payload.PaymentID != listing.PaymentID {
		c.JSON(http.StatusConflict, gin.H{"error": "payment does not belong to this listing"})
		return
	}

	// A callback is only conclusive for a completed payment. Any other status
	// leaves the listing untouched, so a stray event cannot move it.
	if payload.Status != "completed" {
		c.JSON(http.StatusOK, gin.H{"success": true, "listing": listing})
		return
	}

	// A completed payment must name the transaction that settled it; without one
	// there is nothing to reconcile against.
	if strings.TrimSpace(payload.TxHash) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a completed payment must carry a tx_hash"})
		return
	}

	// Idempotent: a repeated delivery of the same payment must not rewrite the
	// record or re-trigger downstream work.
	if listing.PaymentStatus == "paid" && listing.PaymentTxHash == payload.TxHash {
		c.JSON(http.StatusOK, gin.H{"success": true, "listing": listing})
		return
	}

	listing.PaymentStatus = "paid"
	listing.Status = "paid"
	listing.PaymentID = payload.PaymentID
	listing.PaymentTxHash = payload.TxHash
	listing.UpdatedAt = time.Now()

	if err := s.storeListing(c.Request.Context(), listing); err != nil {
		log.Printf("payment webhook: cannot record the payment: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the payment could not be recorded, please retry"})
		return
	}

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

		// Validate the JWT issued by wallet_api; the subject is the admin email.
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		email, err := parseAdminJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

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
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		email, err := parseAdminJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

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

	result, err := s.listListings(c.Request.Context(), status, tier)
	if err != nil {
		log.Printf("cannot list listings: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the listings could not be read, please retry"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"listings": result,
		"total":    len(result),
		"page":     page,
		"limit":    limit,
	})
}

func (s *ListingService) GetListingDetails(c *gin.Context) {
	id := c.Param("id")

	listing, ok := s.loadListing(c.Request.Context(), id)
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

	listing, ok := s.loadListing(c.Request.Context(), id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	validStatuses := map[string]bool{
		"pending":   true,
		"paid":      true,
		"reviewing": true,
		"approved":  true,
		"rejected":  true,
	}

	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	listing.Status = req.Status
	listing.UpdatedAt = time.Now()

	if err := s.storeListing(c.Request.Context(), listing); err != nil {
		log.Printf("listing %s: cannot record the new status: %v", id, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the status could not be saved, please retry"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"listing": listing,
	})
}

func (s *ListingService) ReviewListing(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status     string `json:"status" binding:"required"`
		AdminNotes string `json:"admin_notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	listing, ok := s.loadListing(c.Request.Context(), id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	now := time.Now()
	listing.Status = req.Status
	listing.AdminNotes = req.AdminNotes
	if adminEmail, ok := c.Get("admin_email"); ok {
		if email, isString := adminEmail.(string); isString {
			listing.ReviewedBy = email
		}
	}
	listing.ReviewedAt = &now
	listing.UpdatedAt = now

	if err := s.storeListing(c.Request.Context(), listing); err != nil {
		log.Printf("listing %s: cannot record the review: %v", id, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the review could not be saved, please retry"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"listing": listing,
	})
}

func (s *ListingService) DeleteListing(c *gin.Context) {
	id := c.Param("id")

	if _, ok := s.loadListing(c.Request.Context(), id); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	if err := s.deleteListing(c.Request.Context(), id); err != nil {
		log.Printf("listing %s: cannot delete: %v", id, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the listing could not be deleted, please retry"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "listing deleted",
	})
}

func (s *ListingService) GetListingStats(c *gin.Context) {
	all, err := s.listListings(c.Request.Context(), "", "")
	if err != nil {
		log.Printf("cannot read listings for the stats: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the stats could not be read, please retry"})
		return
	}

	stats := map[string]int{
		"total":     len(all),
		"pending":   0,
		"paid":      0,
		"reviewing": 0,
		"approved":  0,
		"rejected":  0,
	}

	for _, l := range all {
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
		Email      string `json:"email" binding:"required,email"`
		Role       string `json:"role" binding:"required"`
		CanApprove bool   `json:"can_approve"`
		CanEdit    bool   `json:"can_edit"`
		CanDelete  bool   `json:"can_delete"`
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
		ID:         uuid.New().String(),
		Email:      req.Email,
		Role:       req.Role,
		CanApprove: req.CanApprove,
		CanEdit:    req.CanEdit,
		CanDelete:  req.CanDelete,
		CreatedAt:  time.Now(),
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
	// The health endpoint reports the store this service actually depends on.
	// Listings live in Redis, so a Redis outage means the service cannot serve
	// them; reporting "healthy" regardless would keep a broken replica in the
	// load balancer and hide the outage from operators.
	store := "healthy"
	status := http.StatusOK
	code := "healthy"

	if err := s.redis.Ping(c.Request.Context()).Err(); err != nil {
		store = "unavailable"
		status = http.StatusServiceUnavailable
		code = "degraded"
	}

	payload := gin.H{
		"status":    code,
		"service":   "listing-service",
		"timestamp": time.Now().Unix(),
		"store":     store,
	}

	if all, err := s.listListings(c.Request.Context(), "", ""); err == nil {
		payload["listings"] = len(all)
	}

	c.JSON(status, payload)
}

// ============================================================================
// Main
// ============================================================================

// corsOrigins parses the browser-origin allowlist from ALLOWED_ORIGINS (comma
// separated). An empty allowlist emits no CORS headers, which keeps browsers on
// the same origin by default instead of exposing the API to every site.
func corsOrigins() map[string]bool {
	origins := make(map[string]bool)
	for _, origin := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = true
		}
	}
	return origins
}

func main() {
	if jwtSecret == "" {
		log.Fatalf("JWT_SECRET environment variable must be set")
	}
	log.Println("TigerWallet Token Listing Service")
	log.Println("==================================")

	// Initialize service
	ls := NewListingService()

	// Listings are stored in Redis, so a service that cannot reach it would
	// accept applications and then lose them. Refuse to start instead.
	{
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := ls.redis.Ping(ctx).Err()
		cancel()
		if err != nil {
			log.Fatalf("cannot reach Redis at %s: %v", cfg.RedisAddr, err)
		}
		log.Printf("Connected to Redis at %s", cfg.RedisAddr)
	}

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
		// CORS: explicit origin allowlist. A wildcard would let any site
		// read this API with the visitor's credentials attached.
		if origin := c.Request.Header.Get("Origin"); origin != "" && corsOrigins()[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
		}
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
	ls.KYCRouter(r)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Listing service starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
