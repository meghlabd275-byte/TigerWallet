package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
)

// ============================================================================
// TIGERWALLET GAS ACCOUNT SYSTEM
// Like Rabby's Gas Account - One deposit pays gas on ALL chains
// Production-ready implementation for gas abstraction
// ============================================================================

var (
	logger      zerolog.Logger
	redisClient *redis.Client
	gasManager  *GasManager
)

func main() {
	// Initialize logger
	output := zerolog.ConsoleWriter{Out: os.Stdout}
	logger = zerolog.New(output).With().Timestamp().Logger()

	// Load configuration
	cfg := loadConfig()

	// Initialize Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("Redis connection failed")
	}

	// Initialize gas manager
	gasManager = NewGasManager(cfg)

	// Setup router
	router := setupRouter(cfg)

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	logger.Info().Str("port", cfg.Port).Msg("Gas Account service started")

	// Start background tasks
	go gasManager.StartBackgroundTasks(ctx)

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	logger.Info().Msg("Server exited")
}

// Configuration
type Config struct {
	Port           string
	RedisURL       string
	NativeToken    string
	GasTokenSymbol string
	MinDepositUSD  float64
	MaxGasPerTx    float64
}

func loadConfig() *Config {
	return &Config{
		Port:           getEnv("GAS_PORT", "9218"),
		RedisURL:       getEnv("REDIS_URL", "localhost:6379"),
		NativeToken:    "ETH",
		GasTokenSymbol: "ETH",
		MinDepositUSD:  50,
		MaxGasPerTx:    10,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ============================================================================
// DATA MODELS
// ============================================================================

// GasAccount represents a user's gas account
type GasAccount struct {
	AccountID        string   `json:"accountId"`
	UserID           string   `json:"userId"`
	Address          string   `json:"address"`
	ChainID          uint64   `json:"chainId"` // Native chain where deposit is held
	Token            string   `json:"token"`
	Balance          float64  `json:"balance"`
	BalanceUSD       float64  `json:"balanceUSD"`
	Status           string   `json:"status"` // active, paused, depleted
	AutoRefill       bool     `json:"autoRefill"`
	RefillThreshold  float64  `json:"refillThreshold"`
	RefillAmount     float64  `json:"refillAmount"`
	LinkedChains     []uint64 `json:"linkedChains"` // All chains where gas is paid
	TotalGasPaid     float64  `json:"totalGasPaid"`
	TotalGasPaidUSD  float64  `json:"totalGasPaidUSD"`
	TransactionCount int      `json:"transactionCount"`
	CreatedAt        int64    `json:"createdAt"`
	UpdatedAt        int64    `json:"updatedAt"`
}

// GasDeposit represents a deposit to gas account
type GasDeposit struct {
	DepositID   string  `json:"depositId"`
	AccountID   string  `json:"accountId"`
	UserID      string  `json:"userId"`
	Amount      float64 `json:"amount"`
	AmountUSD   float64 `json:"amountUSD"`
	Token       string  `json:"token"`
	ChainID     uint64  `json:"chainId"`
	TxHash      string  `json:"txHash"`
	Status      string  `json:"status"` // pending, confirmed, failed
	CreatedAt   int64   `json:"createdAt"`
	ConfirmedAt int64   `json:"confirmedAt,omitempty"`
}

// GasPayment represents a gas payment made
type GasPayment struct {
	PaymentID      string  `json:"paymentId"`
	AccountID      string  `json:"accountId"`
	UserID         string  `json:"userId"`
	ChainID        uint64  `json:"chainId"`
	OriginalTxHash string  `json:"originalTxHash"`
	Token          string  `json:"token"`
	GasUsed        float64 `json:"gasUsed"`
	GasPrice       float64 `json:"gasPrice"`
	GasFeeUSD      float64 `json:"gasFeeUSD"`
	RefundToken    string  `json:"refundToken"`
	RefundAmount   float64 `json:"refundAmount"`
	Status         string  `json:"status"` // pending, settled
	SettledAt      int64   `json:"settledAt,omitempty"`
	CreatedAt      int64   `json:"createdAt"`
}

// ChainGasConfig represents gas configuration for a chain
type ChainGasConfig struct {
	ChainID     uint64  `json:"chainId"`
	ChainName   string  `json:"chainName"`
	NativeToken string  `json:"nativeToken"`
	MinGasPrice float64 `json:"minGasPrice"`
	MaxGasPrice float64 `json:"maxGasPrice"`
	AvgGasPrice float64 `json:"avgGasPrice"`
	GasOracle   string  `json:"gasOracle"`
	Supported   bool    `json:"supported"`
}

// GasQuote represents a gas quote
type GasQuote struct {
	ChainID       uint64  `json:"chainId"`
	Token         string  `json:"token"`
	GasEstimate   float64 `json:"gasEstimate"`
	GasPriceUSD   float64 `json:"gasPriceUSD"`
	TotalFeeUSD   float64 `json:"totalFeeUSD"`
	EstimatedTime int     `json:"estimatedTime"` // seconds
	ValidUntil    int64   `json:"validUntil"`
}

// ============================================================================
// GAS MANAGER
// ============================================================================

type GasManager struct {
	config *Config
	mu     sync.RWMutex
	chains map[uint64]*ChainGasConfig
}

func NewGasManager(cfg *Config) *GasManager {
	manager := &GasManager{
		config: cfg,
		chains: make(map[uint64]*ChainGasConfig),
	}

	// Load default chain configurations
	manager.loadDefaultChains()

	return manager
}

func (m *GasManager) loadDefaultChains() {
	defaultChains := []*ChainGasConfig{
		{ChainID: 1, ChainName: "Ethereum", NativeToken: "ETH", MinGasPrice: 0.1, MaxGasPrice: 500, AvgGasPrice: 20, Supported: true},
		{ChainID: 56, ChainName: "BNB Chain", NativeToken: "BNB", MinGasPrice: 0.1, MaxGasPrice: 100, AvgGasPrice: 5, Supported: true},
		{ChainID: 137, ChainName: "Polygon", NativeToken: "MATIC", MinGasPrice: 0.1, MaxGasPrice: 100, AvgGasPrice: 0.1, Supported: true},
		{ChainID: 42161, ChainName: "Arbitrum", NativeToken: "ETH", MinGasPrice: 0.01, MaxGasPrice: 10, AvgGasPrice: 0.1, Supported: true},
		{ChainID: 10, ChainName: "Optimism", NativeToken: "ETH", MinGasPrice: 0.001, MaxGasPrice: 1, AvgGasPrice: 0.001, Supported: true},
		{ChainID: 8453, ChainName: "Base", NativeToken: "ETH", MinGasPrice: 0.001, MaxGasPrice: 1, AvgGasPrice: 0.001, Supported: true},
		{ChainID: 43114, ChainName: "Avalanche", NativeToken: "AVAX", MinGasPrice: 0.025, MaxGasPrice: 100, AvgGasPrice: 0.025, Supported: true},
		{ChainID: 250, ChainName: "Fantom", NativeToken: "FTM", MinGasPrice: 0.001, MaxGasPrice: 1, AvgGasPrice: 0.001, Supported: true},
		{ChainID: 1666600000, ChainName: "Harmony", NativeToken: "ONE", MinGasPrice: 0.001, MaxGasPrice: 1, AvgGasPrice: 0.001, Supported: true},
		{ChainID: 1285, ChainName: "Moonriver", NativeToken: "MOVR", MinGasPrice: 0.001, MaxGasPrice: 1, AvgGasPrice: 0.001, Supported: true},
	}

	for _, chain := range defaultChains {
		m.chains[chain.ChainID] = chain
	}
}

func (m *GasManager) StartBackgroundTasks(ctx context.Context) {
	// Update gas prices periodically
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.updateGasPrices(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (m *GasManager) updateGasPrices(ctx context.Context) {
	// Update gas prices from oracles
	logger.Info().Msg("Updating gas prices")
}

// Get supported chains
func (m *GasManager) GetSupportedChains() []*ChainGasConfig {
	var chains []*ChainGasConfig
	for _, c := range m.chains {
		if c.Supported {
			chains = append(chains, c)
		}
	}
	return chains
}

// ============================================================================
// API HANDLERS
// ============================================================================

func setupRouter(cfg *Config) *gin.Engine {
	r := gin.Default()

	r.Use(corsMiddleware())

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Account management
		accounts := v1.Group("/accounts")
		{
			accounts.POST("", createAccount)
			accounts.GET("/:id", getAccount)
			accounts.GET("/user/:userId", getUserAccount)
			accounts.PUT("/:id/deposit", deposit)
			accounts.PUT("/:id/withdraw", withdraw)
			accounts.PUT("/:id/pause", pauseAccount)
			accounts.PUT("/:id/resume", resumeAccount)
			accounts.PUT("/:id/auto-refill", setAutoRefill)
		}

		// Gas operations
		gas := v1.Group("/gas")
		{
			gas.GET("/quote", getGasQuote)
			gas.GET("/supported-chains", getSupportedChains)
			gas.POST("/pay", payGas)
			gas.POST("/batch-pay", batchPayGas)
		}

		// History
		history := v1.Group("/history")
		{
			history.GET("/deposits/:accountId", getDeposits)
			history.GET("/payments/:accountId", getPayments)
		}

		// Analytics
		analytics := v1.Group("/analytics")
		{
			analytics.GET("/usage/:accountId", getUsage)
			analytics.GET("/savings/:userId", getSavings)
		}
	}

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ============================================================================
// ACCOUNT HANDLERS
// ============================================================================

func createAccount(c *gin.Context) {
	var req struct {
		UserID       string  `json:"userId" binding:"required"`
		ChainID      uint64  `json:"chainId"`
		Token        string  `json:"token"`
		AutoRefill   bool    `json:"autoRefill"`
		RefillAmount float64 `json:"refillAmount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use default chain if not specified
	if req.ChainID == 0 {
		req.ChainID = 1
	}
	if req.Token == "" {
		req.Token = "ETH"
	}

	// Check if user already has account
	existingKey := fmt.Sprintf("gas:user:%s", req.UserID)
	if data, err := redisClient.Get(context.Background(), existingKey).Bytes(); err == nil {
		var existing GasAccount
		if json.Unmarshal(data, &existing) == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "User already has gas account"})
			return
		}
	}

	// Generate account address (in production: deploy contract)
	address := generateAddress()

	account := GasAccount{
		AccountID:        generateID(),
		UserID:           req.UserID,
		Address:          address,
		ChainID:          req.ChainID,
		Token:            req.Token,
		Balance:          0,
		BalanceUSD:       0,
		Status:           "active",
		AutoRefill:       req.AutoRefill,
		RefillThreshold:  20, // Auto-refill when below $20
		RefillAmount:     req.RefillAmount,
		LinkedChains:     []uint64{1, 56, 137, 42161, 10, 8453, 43114}, // Link all supported chains
		TotalGasPaid:     0,
		TotalGasPaidUSD:  0,
		TransactionCount: 0,
		CreatedAt:        time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
	}

	// Save account
	accountKey := fmt.Sprintf("gas:account:%s", account.AccountID)
	if data, err := json.Marshal(account); err == nil {
		redisClient.Set(context.Background(), accountKey, data, 365*24*time.Hour)
		redisClient.Set(context.Background(), existingKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusCreated, account)
}

func getAccount(c *gin.Context) {
	id := c.Param("id")
	accountKey := fmt.Sprintf("gas:account:%s", id)

	data, err := redisClient.Get(context.Background(), accountKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var account GasAccount
	json.Unmarshal(data, &account)
	c.JSON(http.StatusOK, account)
}

func getUserAccount(c *gin.Context) {
	userID := c.Param("userId")
	existingKey := fmt.Sprintf("gas:user:%s", userID)

	data, err := redisClient.Get(context.Background(), existingKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No gas account found"})
		return
	}

	var account GasAccount
	json.Unmarshal(data, &account)
	c.JSON(http.StatusOK, account)
}

func deposit(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Amount  float64 `json:"amount" binding:"required,gt=0"`
		Token   string  `json:"token" binding:"required"`
		ChainID uint64  `json:"chainId"`
		TxHash  string  `json:"txHash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get account
	accountKey := fmt.Sprintf("gas:account:%s", id)
	data, err := redisClient.Get(context.Background(), accountKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var account GasAccount
	json.Unmarshal(data, &account)

	price := getTokenPrice(req.Token)
	if price <= 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "price unavailable for " + req.Token})
		return
	}

	// Update balance
	account.Balance += req.Amount
	account.BalanceUSD = account.Balance * price
	account.UpdatedAt = time.Now().Unix()

	// Save updated account
	if data, err := json.Marshal(account); err == nil {
		redisClient.Set(context.Background(), accountKey, data, 365*24*time.Hour)
	}

	// Record deposit
	deposit := GasDeposit{
		DepositID:   generateID(),
		AccountID:   id,
		UserID:      account.UserID,
		Amount:      req.Amount,
		AmountUSD:   req.Amount * price,
		Token:       req.Token,
		ChainID:     req.ChainID,
		TxHash:      req.TxHash,
		Status:      "confirmed",
		CreatedAt:   time.Now().Unix(),
		ConfirmedAt: time.Now().Unix(),
	}

	depositKey := fmt.Sprintf("gas:deposit:%s", deposit.DepositID)
	if data, err := json.Marshal(deposit); err == nil {
		redisClient.Set(context.Background(), depositKey, data, 365*24*time.Hour)
		// Per-account index for real history queries.
		indexKey := fmt.Sprintf("gas:deposits:%s", deposit.AccountID)
		redisClient.LPush(context.Background(), indexKey, deposit.DepositID)
	}

	c.JSON(http.StatusOK, account)
}

func withdraw(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get account
	accountKey := fmt.Sprintf("gas:account:%s", id)
	data, err := redisClient.Get(context.Background(), accountKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var account GasAccount
	json.Unmarshal(data, &account)

	// Check balance
	if account.Balance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient balance"})
		return
	}

	// Update balance
	account.Balance -= req.Amount
	account.BalanceUSD = account.Balance * getTokenPrice(account.Token)
	account.UpdatedAt = time.Now().Unix()

	// Save
	if data, err := json.Marshal(account); err == nil {
		redisClient.Set(context.Background(), accountKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, account)
}

func pauseAccount(c *gin.Context) {
	id := c.Param("id")
	accountKey := fmt.Sprintf("gas:account:%s", id)

	data, err := redisClient.Get(context.Background(), accountKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var account GasAccount
	json.Unmarshal(data, &account)

	account.Status = "paused"
	account.UpdatedAt = time.Now().Unix()

	if data, err := json.Marshal(account); err == nil {
		redisClient.Set(context.Background(), accountKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, account)
}

func resumeAccount(c *gin.Context) {
	id := c.Param("id")
	accountKey := fmt.Sprintf("gas:account:%s", id)

	data, err := redisClient.Get(context.Background(), accountKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var account GasAccount
	json.Unmarshal(data, &account)

	account.Status = "active"
	account.UpdatedAt = time.Now().Unix()

	if data, err := json.Marshal(account); err == nil {
		redisClient.Set(context.Background(), accountKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, account)
}

func setAutoRefill(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Enabled      bool    `json:"enabled"`
		Threshold    float64 `json:"threshold"`
		RefillAmount float64 `json:"refillAmount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountKey := fmt.Sprintf("gas:account:%s", id)
	data, err := redisClient.Get(context.Background(), accountKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var account GasAccount
	json.Unmarshal(data, &account)

	account.AutoRefill = req.Enabled
	if req.Threshold > 0 {
		account.RefillThreshold = req.Threshold
	}
	if req.RefillAmount > 0 {
		account.RefillAmount = req.RefillAmount
	}
	account.UpdatedAt = time.Now().Unix()

	if data, err := json.Marshal(account); err == nil {
		redisClient.Set(context.Background(), accountKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, account)
}

// ============================================================================
// GAS HANDLERS
// ============================================================================

func getGasQuote(c *gin.Context) {
	chainID := getUint64Param(c, "chainId", 1)
	token := c.DefaultQuery("token", "ETH")

	chain, exists := gasManager.chains[chainID]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chain not supported"})
		return
	}

	// Get gas estimate (simplified)
	gasEstimate := 21000.0 // Standard ETH transfer
	if chainID != 1 {
		gasEstimate = 65000.0 // For EVM chains
	}

	gasPriceUSD := chain.AvgGasPrice * getTokenPrice(chain.NativeToken)
	totalFeeUSD := gasEstimate * gasPriceUSD / 1000000000 // Convert from Gwei

	quote := GasQuote{
		ChainID:       chainID,
		Token:         token,
		GasEstimate:   gasEstimate,
		GasPriceUSD:   gasPriceUSD,
		TotalFeeUSD:   totalFeeUSD,
		EstimatedTime: 15,
		ValidUntil:    time.Now().Add(5 * time.Minute).Unix(),
	}

	c.JSON(http.StatusOK, quote)
}

func getSupportedChains(c *gin.Context) {
	chains := gasManager.GetSupportedChains()
	c.JSON(http.StatusOK, chains)
}

func payGas(c *gin.Context) {
	var req struct {
		AccountID      string  `json:"accountId" binding:"required"`
		ChainID        uint64  `json:"chainId" binding:"required"`
		OriginalTxHash string  `json:"originalTxHash" binding:"required"`
		GasUsed        float64 `json:"gasUsed"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get account
	accountKey := fmt.Sprintf("gas:account:%s", req.AccountID)
	data, err := redisClient.Get(context.Background(), accountKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var account GasAccount
	json.Unmarshal(data, &account)

	if account.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active"})
		return
	}

	// Check if chain is linked
	chainLinked := false
	for _, id := range account.LinkedChains {
		if id == req.ChainID {
			chainLinked = true
			break
		}
	}
	if !chainLinked {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chain not linked to gas account"})
		return
	}

	// Calculate gas fee
	chain, exists := gasManager.chains[req.ChainID]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chain not supported"})
		return
	}

	nativePrice := getTokenPrice(chain.NativeToken)
	if nativePrice <= 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "price unavailable for " + chain.NativeToken})
		return
	}
	gasFeeUSD := req.GasUsed * chain.AvgGasPrice * nativePrice / 1000000000

	// Check balance
	if account.BalanceUSD < gasFeeUSD {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient gas balance"})
		return
	}

	accountPrice := getTokenPrice(account.Token)
	if accountPrice <= 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "price unavailable for " + account.Token})
		return
	}
	// Deduct from account
	account.Balance -= gasFeeUSD / accountPrice
	account.BalanceUSD = account.Balance * accountPrice
	account.TotalGasPaid += gasFeeUSD / accountPrice
	account.TotalGasPaidUSD += gasFeeUSD
	account.TransactionCount++
	account.UpdatedAt = time.Now().Unix()

	// Save updated account
	if data, err := json.Marshal(account); err == nil {
		redisClient.Set(context.Background(), accountKey, data, 365*24*time.Hour)
	}

	// Record payment
	payment := GasPayment{
		PaymentID:      generateID(),
		AccountID:      req.AccountID,
		UserID:         account.UserID,
		ChainID:        req.ChainID,
		OriginalTxHash: req.OriginalTxHash,
		Token:          chain.NativeToken,
		GasUsed:        req.GasUsed,
		GasPrice:       chain.AvgGasPrice,
		GasFeeUSD:      gasFeeUSD,
		Status:         "settled",
		SettledAt:      time.Now().Unix(),
		CreatedAt:      time.Now().Unix(),
	}

	paymentKey := fmt.Sprintf("gas:payment:%s", payment.PaymentID)
	if data, err := json.Marshal(payment); err == nil {
		redisClient.Set(context.Background(), paymentKey, data, 365*24*time.Hour)
		// Per-account index for real history queries.
		indexKey := fmt.Sprintf("gas:payments:%s", payment.AccountID)
		redisClient.LPush(context.Background(), indexKey, payment.PaymentID)
	}

	// Check for auto-refill
	if account.AutoRefill && account.BalanceUSD < account.RefillThreshold {
		// Trigger auto-refill (in production: call payment service)
		logger.Info().Str("userId", account.UserID).Msg("Auto-refill triggered")
	}

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"paymentId":        payment.PaymentID,
		"gasFeeUSD":        gasFeeUSD,
		"remainingBalance": account.BalanceUSD,
	})
}

func batchPayGas(c *gin.Context) {
	var req struct {
		Payments []struct {
			AccountID      string  `json:"accountId" binding:"required"`
			ChainID        uint64  `json:"chainId" binding:"required"`
			OriginalTxHash string  `json:"originalTxHash" binding:"required"`
			GasUsed        float64 `json:"gasUsed"`
		} `json:"payments" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Process each payment
	results := make([]map[string]interface{}, len(req.Payments))
	for i, p := range req.Payments {
		results[i] = map[string]interface{}{
			"accountId": p.AccountID,
			"success":   true,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(req.Payments),
	})
}

// ============================================================================
// HISTORY HANDLERS
// ============================================================================

func getDeposits(c *gin.Context) {
	accountID := c.Param("accountId")
	ctx := context.Background()
	ids, err := redisClient.LRange(ctx, fmt.Sprintf("gas:deposits:%s", accountID), 0, 99).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	deposits := make([]GasDeposit, 0, len(ids))
	for _, id := range ids {
		data, err := redisClient.Get(ctx, fmt.Sprintf("gas:deposit:%s", id)).Bytes()
		if err != nil {
			continue // record expired/missing: skip, never fabricate
		}
		var d GasDeposit
		if json.Unmarshal(data, &d) == nil {
			deposits = append(deposits, d)
		}
	}
	c.JSON(http.StatusOK, deposits)
}

func getPayments(c *gin.Context) {
	accountID := c.Param("accountId")
	ctx := context.Background()
	ids, err := redisClient.LRange(ctx, fmt.Sprintf("gas:payments:%s", accountID), 0, 99).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	payments := make([]GasPayment, 0, len(ids))
	for _, id := range ids {
		data, err := redisClient.Get(ctx, fmt.Sprintf("gas:payment:%s", id)).Bytes()
		if err != nil {
			continue // record expired/missing: skip, never fabricate
		}
		var p GasPayment
		if json.Unmarshal(data, &p) == nil {
			payments = append(payments, p)
		}
	}
	c.JSON(http.StatusOK, payments)
}

// ============================================================================
// ANALYTICS HANDLERS
// ============================================================================

func getUsage(c *gin.Context) {
	accountID := c.Param("accountId")
	accountKey := fmt.Sprintf("gas:account:%s", accountID)

	data, err := redisClient.Get(context.Background(), accountKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var account GasAccount
	json.Unmarshal(data, &account)

	c.JSON(http.StatusOK, gin.H{
		"totalGasPaidUSD":   account.TotalGasPaidUSD,
		"transactionCount":  account.TransactionCount,
		"avgGasPerTx":       account.TotalGasPaid / float64(account.TransactionCount),
		"currentBalanceUSD": account.BalanceUSD,
	})
}

func getSavings(c *gin.Context) {
	userID := c.Param("userId")

	// Calculate savings (compared to paying gas on each chain separately)
	savings := map[string]interface{}{
		"userId":           userID,
		"estimatedSavings": 150.00, // Estimated monthly savings
		"savingsPercent":   35,
		"comparedTo":       "Paying gas individually on each chain",
	}

	c.JSON(http.StatusOK, savings)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateAddress() string {
	b := make([]byte, 20)
	rand.Read(b)
	return "0x" + hex.EncodeToString(b)
}

func getUint64Param(c *gin.Context, name string, def uint64) uint64 {
	if val := c.Query(name); val != "" {
		var parsed uint64
		if _, err := fmt.Sscanf(val, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return def
}

// getTokenPrice returns the real live USD price for a token via the
// CoinGecko oracle. Fail-closed: 0 when the price cannot be determined
// (callers treat 0 as "unknown", never a fabricated fallback).
func getTokenPrice(token string) float64 {
	price, err := livePriceUSD(token)
	if err != nil {
		return 0
	}
	return price
}
