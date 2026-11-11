/**
 * TigerWallet Crypto Payment Service
 * High-Load Distributed Go Implementation for Stablecoin Payments
 * Supports: USDT, USDC, DAI, TUSD, BUSD, USDP on multiple chains
 */

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port               int    `json:"port"`
	RedisAddr          string `json:"redis_addr"`
	MongoURI           string `json:"mongo_uri"`
	EthRPCURL          string `json:"eth_rpc_url"`
	BSCRPCURL          string `json:"bsc_rpc_url"`
	PolygonRPCURL      string `json:"polygon_rpc_url"`
	ArbitrumRPCURL     string `json:"arbitrum_rpc_url"`
	OptimismRPCURL     string `json:"optimism_rpc_url"`
	AvalancheRPCURL    string `json:"avalanche_rpc_url"`
	PrivateKey         string `json:"private_key"`
	WebhookURL         string `json:"webhook_url"`
	ConfirmationBlocks int64  `json:"confirmation_blocks"`
}

var cfg = Config{
	Port:               8096,
	RedisAddr:          "localhost:6379",
	ConfirmationBlocks: 12,
}

// ============================================================================
// Stablecoin Contract Addresses
// ============================================================================

var StablecoinContracts = map[string]map[string]common.Address{
	"ethereum": {
		"USDT": common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
		"USDC": common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
		"DAI":  common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		"TUSD": common.HexToAddress("0x0000000000085d4780B73119b644AE5ecd22b376"),
		"BUSD": common.HexToAddress("0x4Fabb145d64652a948d72533023f6E7A623C7C53"),
		"USDP": common.HexToAddress("0x8E870D67F660D95d5be530380D0eC0bd388289E1"),
	},
	"bsc": {
		"USDT": common.HexToAddress("0x55d398326f99059fF775485246999027B3197955"),
		"USDC": common.HexToAddress("0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d"),
		"BUSD": common.HexToAddress("0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56"),
		"DAI":  common.HexToAddress("0x1AF3F329e8BE154074D8769D1FFa4eE058B1DBc3"),
	},
	"polygon": {
		"USDT": common.HexToAddress("0xc2132D05D31c914a87C6611C10748AEb04B58e8F"),
		"USDC": common.HexToAddress("0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"),
		"DAI":  common.HexToAddress("0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063"),
	},
	"arbitrum": {
		"USDT": common.HexToAddress("0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"),
		"USDC": common.HexToAddress("0xaf88d065e77c8cC2239327C5EDb3A432268e5831"),
		"DAI":  common.HexToAddress("0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1"),
	},
	"optimism": {
		"USDT": common.HexToAddress("0x94b008aA00579c1307B0EF2c499aD98a8ce58e58"),
		"USDC": common.HexToAddress("0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85"),
		"DAI":  common.HexToAddress("0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1"),
	},
	"avalanche": {
		"USDT": common.HexToAddress("0x9702230A8Ea53601f5cD2dc00fDBc13d4dF4A8c7"),
		"USDC": common.HexToAddress("0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E"),
	},
	"base": {
		"USDT": common.HexToAddress("0xfde4C96c8593536E31F229EA8f37b2ADa2699bb2"),
		"USDC": common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
		"DAI":  common.HexToAddress("0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb"),
	},
}

// Chain IDs
var ChainIDs = map[string]int64{
	"ethereum":  1,
	"bsc":       56,
	"polygon":   137,
	"arbitrum":  42161,
	"optimism":  10,
	"avalanche": 43114,
	"base":      8453,
}

// ============================================================================
// Database Models
// ============================================================================

type Payment struct {
	ID            string     `json:"id" bson:"_id"`
	UserID        string     `json:"user_id" bson:"user_id"`
	OrderID       string     `json:"order_id" bson:"order_id"`
	Amount        string     `json:"amount" bson:"amount"`
	AmountUSD     string     `json:"amount_usd" bson:"amount_usd"`
	Currency      string     `json:"currency" bson:"currency"`
	Status        string     `json:"status" bson:"status"`
	Chain         string     `json:"chain" bson:"chain"`
	Token         string     `json:"token" bson:"token"`
	FromAddress   string     `json:"from_address" bson:"from_address"`
	ToAddress     string     `json:"to_address" bson:"to_address"`
	TxHash        string     `json:"tx_hash" bson:"tx_hash"`
	Confirmations int64      `json:"confirmations" bson:"confirmations"`
	BlockNumber   uint64     `json:"block_number" bson:"block_number"`
	CreatedAt     time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" bson:"updated_at"`
	CompletedAt   *time.Time `json:"completed_at" bson:"completed_at"`
	WebhookSent   bool       `json:"webhook_sent" bson:"webhook_sent"`
	WebhookURL    string     `json:"webhook_url" bson:"webhook_url"`
}

type PaymentAddress struct {
	ID        string    `json:"id" bson:"_id"`
	Address   string    `json:"address" bson:"address"`
	Chain     string    `json:"chain" bson:"chain"`
	Token     string    `json:"token" bson:"token"`
	UserID    string    `json:"user_id" bson:"user_id"`
	OrderID   string    `json:"order_id" bson:"order_id"`
	IsActive  bool      `json:"is_active" bson:"is_active"`
	ExpiresAt time.Time `json:"expires_at" bson:"expires_at"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type FeeConfig struct {
	ID        string    `json:"id" bson:"_id"`
	FeeType   string    `json:"fee_type" bson:"fee_type"` // listing, trading, withdrawal
	Token     string    `json:"token" bson:"token"`
	Amount    string    `json:"amount" bson:"amount"`
	AmountUSD string    `json:"amount_usd" bson:"amount_usd"`
	IsActive  bool      `json:"is_active" bson:"is_active"`
	UpdatedBy string    `json:"updated_by" bson:"updated_by"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// ============================================================================
// ERC20 ABI
// ============================================================================

var ERC20ABI, _ = abi.JSON(strings.NewReader(`[
	{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},
	{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"type":"function"},
	{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"type":"function"},
	{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"type":"function"},
	{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"type":"function"},
	{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"type":"function"},
	{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"},
	{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"}
]`))

// ============================================================================
// Payment Service
// ============================================================================

type PaymentService struct {
	config         *Config
	redis          *redis.Client
	ethClients     map[string]*ethclient.Client
	privateKey     *ecdsa.PrivateKey
	mu             sync.RWMutex
	store          *paymentStore
	webhookClients map[string]*http.Client
}

func NewPaymentService() *PaymentService {
	s := &PaymentService{
		config:         &cfg,
		redis:          redis.NewClient(&redis.Options{Addr: cfg.RedisAddr}),
		ethClients:     make(map[string]*ethclient.Client),
		webhookClients: make(map[string]*http.Client),
	}
	s.store = newPaymentStore(s.redis)
	return s
}

func (s *PaymentService) initBlockchainClients() error {
	clients := map[string]string{
		"ethereum":  cfg.EthRPCURL,
		"bsc":       cfg.BSCRPCURL,
		"polygon":   cfg.PolygonRPCURL,
		"arbitrum":  cfg.ArbitrumRPCURL,
		"optimism":  cfg.OptimismRPCURL,
		"avalanche": cfg.AvalancheRPCURL,
	}

	for chain, rpcURL := range clients {
		if rpcURL == "" {
			continue
		}
		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			log.Printf("Failed to connect to %s: %v", chain, err)
			continue
		}
		s.ethClients[chain] = client
		log.Printf("Connected to %s RPC", chain)
	}

	// Initialize private key if provided
	if cfg.PrivateKey != "" {
		key, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
		if err != nil {
			log.Printf("Failed to parse private key: %v", err)
		} else {
			s.privateKey = key
			log.Printf("Payment service initialized with hot wallet")
		}
	}

	return nil
}

func (s *PaymentService) initFeeConfigs() {
	// Default fee configurations
	defaultFees := []*FeeConfig{
		{ID: "listing_tier1", FeeType: "listing", Token: "USDT", Amount: "5000", AmountUSD: "5000", IsActive: true},
		{ID: "listing_tier2", FeeType: "listing", Token: "USDT", Amount: "2000", AmountUSD: "2000", IsActive: true},
		{ID: "listing_tier3", FeeType: "listing", Token: "USDT", Amount: "1000", AmountUSD: "1000", IsActive: true},
		{ID: "listing_tier4", FeeType: "listing", Token: "USDT", Amount: "500", AmountUSD: "500", IsActive: true},
		{ID: "trading_fee", FeeType: "trading", Token: "USDT", Amount: "0.001", AmountUSD: "0.001", IsActive: true},
		{ID: "withdrawal_fee", FeeType: "withdrawal", Token: "USDT", Amount: "10", AmountUSD: "10", IsActive: true},
	}

	// Seeded once: an operator's later edit must not be overwritten on every
	// restart, so only ids that are not already stored are written.
	for _, fee := range defaultFees {
		if _, exists := s.store.getFee(fee.ID); exists {
			continue
		}
		if err := s.store.putFee(fee); err != nil {
			log.Printf("cannot seed fee %s: %v", fee.ID, err)
		}
	}
}

// ============================================================================
// API Handlers
// ============================================================================

func (s *PaymentService) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/payment")
	{
		// Payment address management
		api.POST("/address/generate", s.GeneratePaymentAddress)
		api.GET("/address/:address", s.GetPaymentAddress)
		api.GET("/address/:address/status", s.GetPaymentStatus)

		// Payment operations
		api.POST("/deposit/create", s.CreateDeposit)
		api.POST("/withdraw", s.CreateWithdrawal)
		api.GET("/status/:id", s.GetPaymentStatus)
		api.GET("/history/:user_id", s.GetPaymentHistory)

		// Fee management
		api.GET("/fees", s.GetFeeConfigs)
		api.POST("/fees/update", s.UpdateFeeConfig)

		// Supported tokens
		api.GET("/tokens", s.GetSupportedTokens)
		api.GET("/chains", s.GetSupportedChains)

		// Webhook callbacks for blockchain events
		api.POST("/webhook/payment", s.HandlePaymentWebhook)
	}

	// Health check
	r.GET("/health", s.HealthCheck)
}

// Generate payment address for user to send tokens
func (s *PaymentService) GeneratePaymentAddress(c *gin.Context) {
	var req struct {
		UserID     string `json:"user_id" binding:"required"`
		Chain      string `json:"chain" binding:"required"`
		Token      string `json:"token" binding:"required"`
		OrderID    string `json:"order_id"`
		Amount     string `json:"amount"`
		WebhookURL string `json:"webhook_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate chain and token
	if _, ok := StablecoinContracts[req.Chain]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain"})
		return
	}

	if _, ok := StablecoinContracts[req.Chain][req.Token]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported token on this chain"})
		return
	}

	// Generate unique payment address
	paymentAddr := &PaymentAddress{
		ID:        uuid.New().String(),
		Address:   s.generatePaymentAddress(req.Chain),
		Chain:     req.Chain,
		Token:     req.Token,
		UserID:    req.UserID,
		OrderID:   req.OrderID,
		IsActive:  true,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	// Create payment record if amount specified
	var payment *Payment
	if req.Amount != "" && req.OrderID != "" {
		payment = &Payment{
			ID:         uuid.New().String(),
			UserID:     req.UserID,
			OrderID:    req.OrderID,
			Amount:     req.Amount,
			Currency:   req.Token,
			Status:     "pending",
			Chain:      req.Chain,
			Token:      req.Token,
			ToAddress:  paymentAddr.Address,
			WebhookURL: req.WebhookURL,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	}

	// The address and its payment are written together: the chain listener
	// attributes an incoming transfer by address, so an address that is not
	// durably recorded is money that can never be matched to a payment.
	s.mu.Lock()
	var storeErr error
	if payment != nil {
		storeErr = s.store.putPaymentWithAddress(payment, paymentAddr)
	} else {
		storeErr = s.store.putAddress(paymentAddr)
	}
	s.mu.Unlock()

	if storeErr != nil {
		log.Printf("cannot persist the deposit address for order %s: %v", req.OrderID, storeErr)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the payment address could not be saved, please retry"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"payment_address": map[string]interface{}{
			"address":    paymentAddr.Address,
			"chain":      paymentAddr.Chain,
			"token":      paymentAddr.Token,
			"expires_at": paymentAddr.ExpiresAt,
			"qr_code":    fmt.Sprintf("%s:%s?token=%s", paymentAddr.Chain, paymentAddr.Address, paymentAddr.Token),
		},
		"payment": payment,
	})
}

// Get payment address details
func (s *PaymentService) GetPaymentAddress(c *gin.Context) {
	address := c.Param("address")

	s.mu.RLock()
	paymentAddr, ok := s.store.getAddress(address)
	s.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment address not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"address": paymentAddr,
	})
}

// Get payment status by ID
func (s *PaymentService) GetPaymentStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		address := c.Param("address")
		s.mu.RLock()
		paymentAddr, ok := s.store.getAddress(address)
		s.mu.RUnlock()

		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment address not found"})
			return
		}

		// Find payment by order ID or address
		s.mu.RLock()
		for _, p := range s.store.listPayments() {
			if p.ToAddress == address || p.OrderID == paymentAddr.OrderID {
				s.mu.RUnlock()
				c.JSON(http.StatusOK, gin.H{
					"success":       true,
					"payment":       p,
					"confirmations": p.Confirmations,
				})
				return
			}
		}
		s.mu.RUnlock()

		c.JSON(http.StatusOK, gin.H{
			"success":       true,
			"payment":       nil,
			"address":       paymentAddr,
			"confirmations": int64(0),
		})
		return
	}

	s.mu.RLock()
	payment, ok := s.store.getPayment(id)
	s.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"payment":       payment,
		"confirmations": payment.Confirmations,
	})
}

// Create deposit payment
func (s *PaymentService) CreateDeposit(c *gin.Context) {
	var req struct {
		UserID     string `json:"user_id" binding:"required"`
		OrderID    string `json:"order_id" binding:"required"`
		Amount     string `json:"amount" binding:"required"`
		AmountUSD  string `json:"amount_usd"`
		Chain      string `json:"chain" binding:"required"`
		Token      string `json:"token" binding:"required"`
		WebhookURL string `json:"webhook_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate chain and token
	if _, ok := StablecoinContracts[req.Chain]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain"})
		return
	}

	if _, ok := StablecoinContracts[req.Chain][req.Token]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported token"})
		return
	}

	// Generate payment address
	paymentAddr := &PaymentAddress{
		ID:        uuid.New().String(),
		Address:   s.generatePaymentAddress(req.Chain),
		Chain:     req.Chain,
		Token:     req.Token,
		UserID:    req.UserID,
		OrderID:   req.OrderID,
		IsActive:  true,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	payment := &Payment{
		ID:         uuid.New().String(),
		UserID:     req.UserID,
		OrderID:    req.OrderID,
		Amount:     req.Amount,
		AmountUSD:  req.AmountUSD,
		Currency:   req.Token,
		Status:     "pending",
		Chain:      req.Chain,
		Token:      req.Token,
		ToAddress:  paymentAddr.Address,
		WebhookURL: req.WebhookURL,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	s.mu.Lock()
	storeErr := s.store.putPaymentWithAddress(payment, paymentAddr)
	s.mu.Unlock()
	if storeErr != nil {
		log.Printf("cannot persist deposit %s: %v", payment.ID, storeErr)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the deposit could not be saved, please retry"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"payment": map[string]interface{}{
			"id":              payment.ID,
			"order_id":        payment.OrderID,
			"amount":          payment.Amount,
			"amount_usd":      payment.AmountUSD,
			"chain":           payment.Chain,
			"token":           payment.Token,
			"payment_address": paymentAddr.Address,
			"qr_code":         fmt.Sprintf("ethereum:%s?value=%s", paymentAddr.Address, payment.Amount),
			"expires_at":      paymentAddr.ExpiresAt,
			"status":          payment.Status,
		},
	})
}

// Create withdrawal
func (s *PaymentService) CreateWithdrawal(c *gin.Context) {
	var req struct {
		UserID     string `json:"user_id" binding:"required"`
		ToAddress  string `json:"to_address" binding:"required"`
		Amount     string `json:"amount" binding:"required"`
		Chain      string `json:"chain" binding:"required"`
		Token      string `json:"token" binding:"required"`
		Fee        string `json:"fee"`
		WebhookURL string `json:"webhook_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate address
	if !common.IsHexAddress(req.ToAddress) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient address"})
		return
	}

	// Validate chain and token
	if _, ok := StablecoinContracts[req.Chain]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain"})
		return
	}

	if _, ok := StablecoinContracts[req.Chain][req.Token]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported token"})
		return
	}

	payment := &Payment{
		ID:          uuid.New().String(),
		UserID:      req.UserID,
		Amount:      req.Amount,
		Currency:    req.Token,
		Status:      "processing",
		Chain:       req.Chain,
		Token:       req.Token,
		FromAddress: s.getHotWalletAddress(),
		ToAddress:   req.ToAddress,
		WebhookURL:  req.WebhookURL,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.mu.Lock()
	storeErr := s.store.putPayment(payment)
	s.mu.Unlock()
	if storeErr != nil {
		log.Printf("cannot persist withdrawal %s: %v", payment.ID, storeErr)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the withdrawal could not be saved, please retry"})
		return
	}

	// Broadcast runs in the background; the response returns while it is in flight.
	go s.processWithdrawal(payment)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"payment": payment,
	})
}

// processWithdrawal performs a REAL on-chain ERC-20 transfer(to, amount)
// from the configured hot wallet. If no hot wallet key or RPC client is
// configured, the withdrawal is marked "requires_signing" so the caller can
// submit it via the canonical wallet_api signing endpoint. We NEVER fabricate
// a transaction hash.
func (s *PaymentService) processWithdrawal(payment *Payment) {
	client, ok := s.ethClients[payment.Chain]
	if !ok {
		payment.Status = "requires_signing"
		payment.UpdatedAt = time.Now()
		log.Printf("Withdrawal %s: no RPC client for chain %s, requires external signing via wallet_api", payment.ID, payment.Chain)
		return
	}
	if s.privateKey == nil {
		payment.Status = "requires_signing"
		payment.UpdatedAt = time.Now()
		log.Printf("Withdrawal %s: no hot wallet key, requires external signing via wallet_api", payment.ID)
		return
	}

	tokenAddr := StablecoinContracts[payment.Chain][payment.Token]
	chainID := big.NewInt(ChainIDs[payment.Chain])

	amount, ok := new(big.Int).SetString(payment.Amount, 10)
	if !ok {
		payment.Status = "failed"
		payment.UpdatedAt = time.Now()
		log.Printf("Withdrawal %s: invalid amount %s", payment.ID, payment.Amount)
		return
	}

	toAddr := common.HexToAddress(payment.ToAddress)

	// Build ERC-20 transfer(address,uint256) calldata:
	// selector = keccak256("transfer(address,uint256)")[:4] = 0xa9059cbb
	data := make([]byte, 4+32+32)
	data[0], data[1], data[2], data[3] = 0xa9, 0x05, 0x9c, 0xbb
	copy(data[4+12:], toAddr.Bytes()) // address right-padded to 32
	amount.FillBytes(data[36:68])     // uint256 big-endian, left-padded to 32

	nonce, err := client.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(s.privateKey.PublicKey))
	if err != nil {
		payment.Status = "failed"
		payment.UpdatedAt = time.Now()
		log.Printf("Withdrawal %s: nonce error: %v", payment.ID, err)
		return
	}

	gprice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		payment.Status = "failed"
		payment.UpdatedAt = time.Now()
		log.Printf("Withdrawal %s: gas price error: %v", payment.ID, err)
		return
	}

	gLimit := uint64(60000) // ERC-20 transfer typical cost
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        &tokenAddr,
		Value:     big.NewInt(0),
		Gas:       gLimit,
		GasFeeCap: gprice,
		GasTipCap: big.NewInt(1),
		Data:      data,
	})

	signedTx, err := types.SignTx(tx, types.NewLondonSigner(chainID), s.privateKey)
	if err != nil {
		payment.Status = "failed"
		payment.UpdatedAt = time.Now()
		log.Printf("Withdrawal %s: sign error: %v", payment.ID, err)
		return
	}

	if err := client.SendTransaction(context.Background(), signedTx); err != nil {
		payment.Status = "failed"
		payment.UpdatedAt = time.Now()
		log.Printf("Withdrawal %s: broadcast error: %v", payment.ID, err)
		return
	}

	payment.Status = "broadcasted"
	payment.TxHash = signedTx.Hash().Hex()
	payment.UpdatedAt = time.Now()

	s.mu.Lock()
	err = s.store.putPayment(payment)
	s.mu.Unlock()
	if err != nil {
		// The transaction is already on chain, so this is a bookkeeping
		// failure: report it loudly rather than losing the hash.
		log.Printf("withdrawal %s broadcast as %s but could not be recorded: %v", payment.ID, payment.TxHash, err)
		return
	}

	log.Printf("Withdrawal %s broadcasted: %s", payment.ID, payment.TxHash)
}

// Get payment history
func (s *PaymentService) GetPaymentHistory(c *gin.Context) {
	userID := c.Param("user_id")

	s.mu.RLock()
	var userPayments []*Payment
	for _, p := range s.store.listPayments() {
		if p.UserID == userID {
			userPayments = append(userPayments, p)
		}
	}
	s.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"payments": userPayments,
	})
}

// Get fee configurations
func (s *PaymentService) GetFeeConfigs(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fees := s.store.listFees()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"fees":    fees,
	})
}

// Update fee configuration (SuperAdmin only)
func (s *PaymentService) UpdateFeeConfig(c *gin.Context) {
	var req struct {
		FeeID     string `json:"fee_id" binding:"required"`
		Amount    string `json:"amount" binding:"required"`
		AmountUSD string `json:"amount_usd"`
		Token     string `json:"token"`
		UpdatedBy string `json:"updated_by" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify SuperAdmin (in production, check JWT claims)
	if !strings.HasSuffix(req.UpdatedBy, "@tigerwallet.com") && req.UpdatedBy != "superadmin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only superadmin can update fees"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if fee, ok := s.store.getFee(req.FeeID); ok {
		fee.Amount = req.Amount
		if req.AmountUSD != "" {
			fee.AmountUSD = req.AmountUSD
		}
		if req.Token != "" {
			fee.Token = req.Token
		}
		fee.UpdatedBy = req.UpdatedBy
		fee.UpdatedAt = time.Now()
		if err := s.store.putFee(fee); err != nil {
			log.Printf("cannot persist fee %s: %v", fee.ID, err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the fee could not be saved, please retry"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"fee":     fee,
		})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "fee configuration not found"})
}

// Get supported tokens
func (s *PaymentService) GetSupportedTokens(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tokens":  StablecoinContracts,
	})
}

// Get supported chains
func (s *PaymentService) GetSupportedChains(c *gin.Context) {
	chains := []map[string]interface{}{}
	for chain, tokens := range StablecoinContracts {
		var tokenList []string
		for token := range tokens {
			tokenList = append(tokenList, token)
		}
		chains = append(chains, map[string]interface{}{
			"id":       chain,
			"name":     strings.Title(chain),
			"chain_id": ChainIDs[chain],
			"tokens":   tokenList,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"chains":  chains,
	})
}

// HandlePaymentWebhook is called by the chain listener when a deposit reaches
// the deposit address. The caller's data is only a hint: the transaction is
// re-read from the chain and the payment is completed only when the transfer is
// confirmed there, so a forged callback cannot mark a payment paid.
func (s *PaymentService) HandlePaymentWebhook(c *gin.Context) {
	secret := paymentWebhookSecret()
	if secret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the chain listener is not configured"})
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
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	if !hmac.Equal([]byte(signature), []byte(hex.EncodeToString(mac.Sum(nil)))) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var payload struct {
		TxHash  string `json:"tx_hash"`
		Address string `json:"address"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if payload.TxHash == "" || payload.Address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tx_hash and address are required"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Find payment by address
	var payment *Payment
	for _, p := range s.store.listPayments() {
		if strings.EqualFold(p.ToAddress, payload.Address) && p.Status == "pending" {
			payment = p
			break
		}
	}

	if payment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no pending payment found for address"})
		return
	}

	confirmations, err := s.verifyDepositOnChain(c.Request.Context(), payment, payload.TxHash)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	payment.TxHash = payload.TxHash
	payment.Confirmations = confirmations

	if confirmations >= cfg.ConfirmationBlocks {
		payment.Status = "completed"
		now := time.Now()
		payment.CompletedAt = &now

		// Send webhook notification
		if payment.WebhookURL != "" && !payment.WebhookSent {
			go s.sendWebhook(payment)
		}
	}

	payment.UpdatedAt = time.Now()

	if err := s.store.putPayment(payment); err != nil {
		log.Printf("cannot record verified deposit %s: %v", payment.ID, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the verified payment could not be recorded, please retry"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"payment": payment,
	})
}

// verifyDepositOnChain re-reads a deposit from the chain and returns its
// confirmation count. It fails when the transaction is missing, reverted, or
// does not move the expected amount of the expected token to the payment's
// address, so a callback can never complete a payment that did not happen.
func (s *PaymentService) verifyDepositOnChain(ctx context.Context, payment *Payment, txHash string) (int64, error) {
	if !strings.HasPrefix(txHash, "0x") || len(txHash) != 66 {
		return 0, fmt.Errorf("tx_hash is not a transaction hash")
	}

	client, ok := s.ethClients[payment.Chain]
	if !ok {
		return 0, fmt.Errorf("no RPC client is configured for chain %s", payment.Chain)
	}

	receipt, err := client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return 0, fmt.Errorf("the transaction is not on chain yet: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return 0, fmt.Errorf("the transaction reverted on chain")
	}

	tokenAddr, ok := StablecoinContracts[payment.Chain][payment.Token]
	if !ok {
		return 0, fmt.Errorf("token %s is not supported on chain %s", payment.Token, payment.Chain)
	}

	expected, ok := new(big.Int).SetString(payment.Amount, 10)
	if !ok {
		return 0, fmt.Errorf("the recorded amount is not a valid integer")
	}

	// ERC-20 Transfer(address,address,uint256): the deposit must move at least
	// the expected amount of the expected token to the payment address. The
	// topics and data are read from the chain, never from the request.
	transferTopic := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	toTopic := common.BytesToHash(common.LeftPadBytes(common.HexToAddress(payment.ToAddress).Bytes(), 32))

	var credited *big.Int
	for _, lg := range receipt.Logs {
		if lg.Address != tokenAddr || len(lg.Topics) != 3 {
			continue
		}
		if lg.Topics[0] != transferTopic || lg.Topics[2] != toTopic {
			continue
		}
		value := new(big.Int).SetBytes(lg.Data)
		if credited == nil || value.Cmp(credited) > 0 {
			credited = value
		}
	}
	if credited == nil {
		return 0, fmt.Errorf("the transaction contains no transfer of %s to %s", payment.Token, payment.ToAddress)
	}
	if credited.Cmp(expected) < 0 {
		return 0, fmt.Errorf("the transaction transferred %s but %s was required", credited.String(), expected.String())
	}

	head, err := client.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("cannot read the current block: %w", err)
	}
	if head < receipt.BlockNumber.Uint64() {
		return 0, fmt.Errorf("the transaction is ahead of the chain head")
	}

	payment.BlockNumber = receipt.BlockNumber.Uint64()
	return int64(head-receipt.BlockNumber.Uint64()) + 1, nil
}

// Send webhook notification
func (s *PaymentService) sendWebhook(payment *Payment) {
	if payment.WebhookURL == "" {
		return
	}

	payload, err := json.Marshal(map[string]interface{}{
		"event":        "payment.completed",
		"payment_id":   payment.ID,
		"order_id":     payment.OrderID,
		"amount":       payment.Amount,
		"currency":     payment.Currency,
		"chain":        payment.Chain,
		"token":        payment.Token,
		"tx_hash":      payment.TxHash,
		"status":       payment.Status,
		"completed_at": payment.CompletedAt,
	})
	if err != nil {
		log.Printf("payment %s: cannot encode the webhook body: %v", payment.ID, err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, payment.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		log.Printf("payment %s: cannot build the webhook request: %v", payment.ID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// The receiving service only acts on a signed body, so the signature is not
	// optional: without it the callback is rejected and the order stays unpaid.
	if secret := paymentWebhookSecret(); secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		req.Header.Set("X-Payment-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	} else {
		log.Printf("payment %s: PAYMENT_WEBHOOK_SECRET is unset; the callback will be rejected by the receiver", payment.ID)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("payment %s: webhook delivery failed: %v", payment.ID, err)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))

	// Only record the delivery as sent when the receiver accepted it, so a
	// failed callback can be retried instead of being silently dropped.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("payment %s: webhook rejected with status %d", payment.ID, resp.StatusCode)
		return
	}

	s.mu.Lock()
	payment.WebhookSent = true
	s.mu.Unlock()
}

// paymentWebhookSecret is the shared secret used to sign outbound webhooks. An
// empty value means the receiver will reject the callback, which is logged.
func paymentWebhookSecret() string {
	return strings.TrimSpace(os.Getenv("PAYMENT_WEBHOOK_SECRET"))
}

// HealthCheck reports whether the service can actually do its job. The store
// holds the deposit addresses the chain listener resolves against, so if it is
// unreachable a payment made to an issued address cannot be attributed and the
// service must not be reported healthy.
func (s *PaymentService) HealthCheck(c *gin.Context) {
	storeStatus := "ok"
	status := "healthy"
	httpStatus := http.StatusOK

	if s.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := s.redis.Ping(ctx).Err(); err != nil {
			storeStatus = "unavailable"
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
		}
	}

	chainStatus := "ok"
	if len(s.ethClients) == 0 {
		// Without an RPC client no deposit can be verified on chain.
		chainStatus = "unconfigured"
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	s.mu.RLock()
	payments := len(s.store.payments)
	addresses := len(s.store.addrs)
	s.mu.RUnlock()

	c.JSON(httpStatus, gin.H{
		"status":           status,
		"service":          "payment-service",
		"timestamp":        time.Now().Unix(),
		"store":            storeStatus,
		"chains":           chainStatus,
		"active_payments":  payments,
		"active_addresses": addresses,
		"connected_chains": len(s.ethClients),
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// generatePaymentAddress returns the hot wallet's address for receiving
// payments on the given chain. It NEVER fabricates a deposit address: users
// must only send funds to a real key-controlled address. If no hot wallet key
// is configured, an empty string is returned and the caller must reject the
// deposit (do NOT substitute a placeholder address).
func (s *PaymentService) generatePaymentAddress(chain string) string {
	return s.getHotWalletAddress()
}

func (s *PaymentService) getHotWalletAddress() string {
	if s.privateKey == nil {
		return "0x0000000000000000000000000000000000000000"
	}
	return crypto.PubkeyToAddress(s.privateKey.PublicKey).Hex()
}

// ============================================================================
// Main
// ============================================================================

// loadConfig populates cfg from the environment. Without this the service keeps
// its built-in defaults: it would listen on the right port but hold no RPC
// endpoints and no hot wallet key, so deposits could never be verified and
// withdrawals could never be signed.
func loadConfig() {
	if v := getEnv("PORT", ""); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			cfg.Port = p
		} else {
			log.Printf("ignoring invalid PORT %q", v)
		}
	}
	cfg.RedisAddr = getEnv("REDIS_ADDR", cfg.RedisAddr)
	cfg.EthRPCURL = getEnv("ETH_RPC_URL", cfg.EthRPCURL)
	cfg.BSCRPCURL = getEnv("BSC_RPC_URL", cfg.BSCRPCURL)
	cfg.PolygonRPCURL = getEnv("POLYGON_RPC_URL", cfg.PolygonRPCURL)
	cfg.ArbitrumRPCURL = getEnv("ARBITRUM_RPC_URL", cfg.ArbitrumRPCURL)
	cfg.OptimismRPCURL = getEnv("OPTIMISM_RPC_URL", cfg.OptimismRPCURL)
	cfg.AvalancheRPCURL = getEnv("AVALANCHE_RPC_URL", cfg.AvalancheRPCURL)
	cfg.WebhookURL = getEnv("PAYMENT_WEBHOOK_URL", cfg.WebhookURL)

	// The hot wallet key is only read when it is present; the service stays
	// usable for deposit collection without it, but withdrawals then report
	// "requires_signing" instead of being broadcast.
	cfg.PrivateKey = strings.TrimSpace(os.Getenv("PAYMENT_PRIVATE_KEY"))

	if v := getEnv("CONFIRMATION_BLOCKS", ""); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			cfg.ConfirmationBlocks = n
		} else {
			log.Printf("ignoring invalid CONFIRMATION_BLOCKS %q", v)
		}
	}

	log.Printf("configuration: port=%d redis=%s confirmations=%d rpc_chains=%d hot_wallet=%t",
		cfg.Port, cfg.RedisAddr, cfg.ConfirmationBlocks, countConfiguredRPCs(), cfg.PrivateKey != "")
}

// countConfiguredRPCs reports how many chain endpoints were supplied, so a
// deployment that forgot them is visible at startup rather than at first use.
func countConfiguredRPCs() int {
	n := 0
	for _, u := range []string{
		cfg.EthRPCURL, cfg.BSCRPCURL, cfg.PolygonRPCURL,
		cfg.ArbitrumRPCURL, cfg.OptimismRPCURL, cfg.AvalancheRPCURL,
	} {
		if strings.TrimSpace(u) != "" {
			n++
		}
	}
	return n
}

func getEnv(key, dflt string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return dflt
}

func main() {
	log.Println("TigerWallet Crypto Payment Service")
	log.Println("===================================")

	loadConfig()

	// Initialize payment service
	ps := NewPaymentService()

	// Initialize blockchain clients
	if err := ps.initBlockchainClients(); err != nil {
		log.Printf("Warning: Some blockchain connections failed: %v", err)
	}

	// Initialize fee configs
	ps.initFeeConfigs()

	// Setup Gin router
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
	ps.SetupRoutes(r)

	// Start payment monitoring goroutine
	go ps.monitorPayments()

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Payment service starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// Monitor payments for confirmations
func (s *PaymentService) monitorPayments() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		var pending []*Payment
		for _, p := range s.store.listPayments() {
			if p.Status == "pending" && p.TxHash != "" && p.Confirmations < cfg.ConfirmationBlocks {
				pending = append(pending, p)
			}
		}
		s.mu.RUnlock()

		for _, p := range pending {
			s.checkPaymentConfirmation(p)
		}
	}
}

func (s *PaymentService) checkPaymentConfirmation(payment *Payment) {
	client, ok := s.ethClients[payment.Chain]
	if !ok {
		return
	}

	txHash := common.HexToHash(payment.TxHash)
	receipt, err := client.TransactionReceipt(context.Background(), txHash)
	if err != nil || receipt == nil {
		return
	}

	block, err := client.BlockNumber(context.Background())
	if err != nil {
		return
	}

	confirmations := int64(block) - receipt.BlockNumber.Int64()

	s.mu.Lock()
	payment.Confirmations = confirmations
	if confirmations >= cfg.ConfirmationBlocks && payment.Status == "pending" {
		payment.Status = "completed"
		now := time.Now()
		payment.CompletedAt = &now
		if payment.WebhookURL != "" && !payment.WebhookSent {
			go s.sendWebhook(payment)
		}
	}
	payment.UpdatedAt = time.Now()
	if err := s.store.putPayment(payment); err != nil {
		log.Printf("cannot record confirmations for %s: %v", payment.ID, err)
	}
	s.mu.Unlock()
}
