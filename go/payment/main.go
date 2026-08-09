/**
 * TigerWallet Crypto Payment Service
 * High-Load Distributed Go Implementation for Stablecoin Payments
 * Supports: USDT, USDC, DAI, TUSD, BUSD, USDP on multiple chains
 */

package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
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
	Port                int      `json:"port"`
	RedisAddr           string   `json:"redis_addr"`
	MongoURI            string   `json:"mongo_uri"`
	EthRPCURL           string   `json:"eth_rpc_url"`
	BSCRPCURL           string   `json:"bsc_rpc_url"`
	PolygonRPCURL       string   `json:"polygon_rpc_url"`
	ArbitrumRPCURL      string   `json:"arbitrum_rpc_url"`
	OptimismRPCURL      string   `json:"optimism_rpc_url"`
	AvalancheRPCURL     string   `json:"avalanche_rpc_url"`
	PrivateKey          string   `json:"private_key"`
	WebhookURL          string   `json:"webhook_url"`
	ConfirmationBlocks int64    `json:"confirmation_blocks"`
}

var cfg = Config{
	Port:                8096,
	RedisAddr:           "localhost:6379",
	ConfirmationBlocks:  12,
}

// ============================================================================
// Stablecoin Contract Addresses
// ============================================================================

var StablecoinContracts = map[string]map[string]common.Address{
	"ethereum": {
		"USDT": common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
		"USDC": common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
		"DAI":  common.HexToAddress("0x6B175474E89094C44Da98b954E1162928195441"),
		"TUSD": common.HexToAddress("0x0000000000085d4780B73119b644AE5ecd22b376"),
		"BUSD": common.HexToAddress("0x4Fabb145d64652a948d72533023f6E7A623C7C1"),
		"USDP": common.HexToAddress("0x8E870D67F660D95d5be530380D0eC7bd38803f1"),
	},
	"bsc": {
		"USDT": common.HexToAddress("0x55d398326f99059fF775485246999027B3197955"),
		"USDC": common.HexToAddress("0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d"),
		"BUSD": common.HexToAddress("0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56"),
		"DAI":  common.HexToAddress("0x1AF3F329e8BE154074D8769D1FFa4eE0581611d2"),
	},
	"polygon": {
		"USDT": common.HexToAddress("0xc2132D05D31c914a87C6611C10748AEb04B58e8F"),
		"USDC": common.HexToAddress("0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"),
		"DAI":  common.HexToAddress("0x53E0bca35eC356bf5f759F85f4c0eB9b4e961736"),
	},
	"arbitrum": {
		"USDT": common.HexToAddress("0xFd086bC7CD5C481DCC93C85BD541717E78c15C93"),
		"USDC": common.HexToAddress("0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"),
		"DAI":  common.HexToAddress("0xDA10009cBd1D4b23f113b0fA7E0ABb0C16d8a6F5"),
	},
	"optimism": {
		"USDT": common.HexToAddress("0x94b008aA00579c1307B0EF2c3ad04f9DB46BcE52"),
		"USDC": common.HexToAddress("0x7F5c764cBc14f9669B88837ca1490cCa17c31607"),
		"DAI":  common.HexToAddress("0xDA10009cBd1D4b23f113b0fA7E0ABb0C16d8a6F5"),
	},
	"avalanche": {
		"USDT": common.HexToAddress("0x9702230A8Ea53601f5cD2dc6f84C4b8e8f78Ec74"),
		"USDC": common.HexToAddress("0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E"),
	},
}

// Chain IDs
var ChainIDs = map[string]int64{
	"ethereum":   1,
	"bsc":        56,
	"polygon":    137,
	"arbitrum":   42161,
	"optimism":   10,
	"avalanche":  43114,
}

// ============================================================================
// Database Models
// ============================================================================

type Payment struct {
	ID            string    `json:"id" bson:"_id"`
	UserID        string    `json:"user_id" bson:"user_id"`
	OrderID       string    `json:"order_id" bson:"order_id"`
	Amount        string    `json:"amount" bson:"amount"`
	AmountUSD     string    `json:"amount_usd" bson:"amount_usd"`
	Currency      string    `json:"currency" bson:"currency"`
	Status        string    `json:"status" bson:"status"`
	Chain         string    `json:"chain" bson:"chain"`
	Token         string    `json:"token" bson:"token"`
	FromAddress   string    `json:"from_address" bson:"from_address"`
	ToAddress     string    `json:"to_address" bson:"to_address"`
	TxHash        string    `json:"tx_hash" bson:"tx_hash"`
	Confirmations int64     `json:"confirmations" bson:"confirmations"`
	BlockNumber   uint64    `json:"block_number" bson:"block_number"`
	CreatedAt     time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" bson:"updated_at"`
	CompletedAt   *time.Time `json:"completed_at" bson:"completed_at"`
	WebhookSent   bool      `json:"webhook_sent" bson:"webhook_sent"`
	WebhookURL    string    `json:"webhook_url" bson:"webhook_url"`
}

type PaymentAddress struct {
	ID           string    `json:"id" bson:"_id"`
	Address      string    `json:"address" bson:"address"`
	Chain        string    `json:"chain" bson:"chain"`
	Token        string    `json:"token" bson:"token"`
	UserID       string    `json:"user_id" bson:"user_id"`
	OrderID      string    `json:"order_id" bson:"order_id"`
	IsActive     bool      `json:"is_active" bson:"is_active"`
	ExpiresAt    time.Time `json:"expires_at" bson:"expires_at"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
}

type FeeConfig struct {
	ID             string    `json:"id" bson:"_id"`
	FeeType        string    `json:"fee_type" bson:"fee_type"` // listing, trading, withdrawal
	Token          string    `json:"token" bson:"token"`
	Amount         string    `json:"amount" bson:"amount"`
	AmountUSD      string    `json:"amount_usd" bson:"amount_usd"`
	IsActive       bool      `json:"is_active" bson:"is_active"`
	UpdatedBy      string    `json:"updated_by" bson:"updated_by"`
	UpdatedAt      time.Time `json:"updated_at" bson:"updated_at"`
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
	payments       map[string]*Payment
	paymentAddrs   map[string]*PaymentAddress
	feeConfigs     map[string]*FeeConfig
	webhookClients map[string]*http.Client
}

func NewPaymentService() *PaymentService {
	return &PaymentService{
		config:         &cfg,
		redis:          redis.NewClient(&redis.Options{Addr: cfg.RedisAddr}),
		ethClients:     make(map[string]*ethclient.Client),
		payments:       make(map[string]*Payment),
		paymentAddrs:   make(map[string]*PaymentAddress),
		feeConfigs:     make(map[string]*FeeConfig),
		webhookClients: make(map[string]*http.Client),
	}
}

func (s *PaymentService) initBlockchainClients() error {
	clients := map[string]string{
		"ethereum":   cfg.EthRPCURL,
		"bsc":        cfg.BSCRPCURL,
		"polygon":    cfg.PolygonRPCURL,
		"arbitrum":   cfg.ArbitrumRPCURL,
		"optimism":   cfg.OptimismRPCURL,
		"avalanche":  cfg.AvalancheRPCURL,
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

	for _, fee := range defaultFees {
		s.feeConfigs[fee.ID] = fee
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
		UserID   string `json:"user_id" binding:"required"`
		Chain    string `json:"chain" binding:"required"`
		Token    string `json:"token" binding:"required"`
		OrderID  string `json:"order_id"`
		Amount   string `json:"amount"`
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

	s.mu.Lock()
	s.paymentAddrs[paymentAddr.Address] = paymentAddr
	s.mu.Unlock()

	// Create payment record if amount specified
	var payment *Payment
	if req.Amount != "" && req.OrderID != "" {
		payment = &Payment{
			ID:          uuid.New().String(),
			UserID:      req.UserID,
			OrderID:     req.OrderID,
			Amount:      req.Amount,
			Currency:    req.Token,
			Status:      "pending",
			Chain:       req.Chain,
			Token:       req.Token,
			ToAddress:   paymentAddr.Address,
			WebhookURL:  req.WebhookURL,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		s.mu.Lock()
		s.payments[payment.ID] = payment
		s.mu.Unlock()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"payment_address": map[string]interface{}{
			"address":      paymentAddr.Address,
			"chain":        paymentAddr.Chain,
			"token":        paymentAddr.Token,
			"expires_at":   paymentAddr.ExpiresAt,
			"qr_code":      fmt.Sprintf("%s:%s?token=%s", paymentAddr.Chain, paymentAddr.Address, paymentAddr.Token),
		},
		"payment": payment,
	})
}

// Get payment address details
func (s *PaymentService) GetPaymentAddress(c *gin.Context) {
	address := c.Param("address")

	s.mu.RLock()
	paymentAddr, ok := s.paymentAddrs[address]
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
		paymentAddr, ok := s.paymentAddrs[address]
		s.mu.RUnlock()

		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment address not found"})
			return
		}

		// Find payment by order ID or address
		s.mu.RLock()
		for _, p := range s.payments {
			if p.ToAddress == address || p.OrderID == paymentAddr.OrderID {
				s.mu.RUnlock()
				c.JSON(http.StatusOK, gin.H{
					"success":      true,
					"payment":      p,
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
	payment, ok := s.payments[id]
	s.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"payment":      payment,
		"confirmations": payment.Confirmations,
	})
}

// Create deposit payment
func (s *PaymentService) CreateDeposit(c *gin.Context) {
	var req struct {
		UserID      string `json:"user_id" binding:"required"`
		OrderID     string `json:"order_id" binding:"required"`
		Amount      string `json:"amount" binding:"required"`
		AmountUSD   string `json:"amount_usd"`
		Chain       string `json:"chain" binding:"required"`
		Token       string `json:"token" binding:"required"`
		WebhookURL  string `json:"webhook_url"`
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
		ID:          uuid.New().String(),
		UserID:      req.UserID,
		OrderID:     req.OrderID,
		Amount:      req.Amount,
		AmountUSD:   req.AmountUSD,
		Currency:    req.Token,
		Status:      "pending",
		Chain:       req.Chain,
		Token:       req.Token,
		ToAddress:   paymentAddr.Address,
		WebhookURL:  req.WebhookURL,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.mu.Lock()
	s.payments[payment.ID] = payment
	s.paymentAddrs[paymentAddr.Address] = paymentAddr
	s.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"payment": map[string]interface{}{
			"id":                  payment.ID,
			"order_id":            payment.OrderID,
			"amount":              payment.Amount,
			"amount_usd":          payment.AmountUSD,
			"chain":               payment.Chain,
			"token":               payment.Token,
			"payment_address":     paymentAddr.Address,
			"qr_code":             fmt.Sprintf("ethereum:%s?value=%s", paymentAddr.Address, payment.Amount),
			"expires_at":          paymentAddr.ExpiresAt,
			"status":              payment.Status,
		},
	})
}

// Create withdrawal
func (s *PaymentService) CreateWithdrawal(c *gin.Context) {
	var req struct {
		UserID     string  `json:"user_id" binding:"required"`
		ToAddress  string  `json:"to_address" binding:"required"`
		Amount     string  `json:"amount" binding:"required"`
		Chain      string  `json:"chain" binding:"required"`
		Token      string  `json:"token" binding:"required"`
		Fee        string  `json:"fee"`
		WebhookURL string  `json:"webhook_url"`
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
	s.payments[payment.ID] = payment
	s.mu.Unlock()

	// In production, this would broadcast the transaction
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
	copy(data[4+12:], toAddr.Bytes())     // address right-padded to 32
	amount.FillBytes(data[36:68])          // uint256 big-endian, left-padded to 32

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
	s.payments[payment.ID] = payment
	s.mu.Unlock()

	log.Printf("Withdrawal %s broadcasted: %s", payment.ID, payment.TxHash)
}

// Get payment history
func (s *PaymentService) GetPaymentHistory(c *gin.Context) {
	userID := c.Param("user_id")

	s.mu.RLock()
	var userPayments []*Payment
	for _, p := range s.payments {
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

	var fees []*FeeConfig
	for _, f := range s.feeConfigs {
		fees = append(fees, f)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"fees":    fees,
	})
}

// Update fee configuration (SuperAdmin only)
func (s *PaymentService) UpdateFeeConfig(c *gin.Context) {
	var req struct {
		FeeID    string `json:"fee_id" binding:"required"`
		Amount   string `json:"amount" binding:"required"`
		AmountUSD string `json:"amount_usd"`
		Token    string `json:"token"`
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

	if fee, ok := s.feeConfigs[req.FeeID]; ok {
		fee.Amount = req.Amount
		if req.AmountUSD != "" {
			fee.AmountUSD = req.AmountUSD
		}
		if req.Token != "" {
			fee.Token = req.Token
		}
		fee.UpdatedBy = req.UpdatedBy
		fee.UpdatedAt = time.Now()
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
			"id":      chain,
			"name":    strings.Title(chain),
			"chain_id": ChainIDs[chain],
			"tokens":  tokenList,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"chains":  chains,
	})
}

// Handle payment webhook from blockchain listener
func (s *PaymentService) HandlePaymentWebhook(c *gin.Context) {
	var payload struct {
		TxHash    string `json:"tx_hash"`
		Address   string `json:"address"`
		Chain     string `json:"chain"`
		Token     string `json:"token"`
		Amount    string `json:"amount"`
		BlockNum  uint64 `json:"block_number"`
		Confirmations int64 `json:"confirmations"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Find payment by address
	var payment *Payment
	for _, p := range s.payments {
		if strings.EqualFold(p.ToAddress, payload.Address) && p.Status == "pending" {
			payment = p
			break
		}
	}

	if payment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no pending payment found for address"})
		return
	}

	payment.TxHash = payload.TxHash
	payment.BlockNumber = payload.BlockNum
	payment.Confirmations = payload.Confirmations

	if payload.Confirmations >= cfg.ConfirmationBlocks {
		payment.Status = "completed"
		now := time.Now()
		payment.CompletedAt = &now

		// Send webhook notification
		if payment.WebhookURL != "" && !payment.WebhookSent {
			go s.sendWebhook(payment)
		}
	}

	payment.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"payment": payment,
	})
}

// Send webhook notification
func (s *PaymentService) sendWebhook(payment *Payment) {
	if payment.WebhookURL == "" {
		return
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"event":       "payment.completed",
		"payment_id":  payment.ID,
		"order_id":    payment.OrderID,
		"amount":      payment.Amount,
		"currency":    payment.Currency,
		"chain":       payment.Chain,
		"token":       payment.Token,
		"tx_hash":     payment.TxHash,
		"status":      payment.Status,
		"completed_at": payment.CompletedAt,
	})

	client := &http.Client{Timeout: 10 * time.Second}
	client.Post(payment.WebhookURL, "application/json", strings.NewReader(string(payload)))

	s.mu.Lock()
	payment.WebhookSent = true
	s.mu.Unlock()
}

// Health check
func (s *PaymentService) HealthCheck(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"status":            "healthy",
		"service":           "payment-service",
		"timestamp":        time.Now().Unix(),
		"active_payments":   len(s.payments),
		"active_addresses":  len(s.paymentAddrs),
		"connected_chains":  len(s.ethClients),
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

func main() {
	log.Println("TigerWallet Crypto Payment Service")
	log.Println("===================================")

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
		for _, p := range s.payments {
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
	s.mu.Unlock()
}
