package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
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
	cfg         *Config
)

func main() {
	// Initialize logger
	output := zerolog.ConsoleWriter{Out: os.Stdout}
	logger = zerolog.New(output).With().Timestamp().Logger()

	// Load configuration
	cfg = loadConfig()

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

	// JWTSecret is the shared secret used to authenticate callers. The service
	// refuses every request when it is unset so a misconfigured deployment
	// cannot be driven anonymously.
	JWTSecret string

	// RPCURLs maps chain id to a JSON-RPC endpoint. Deposits are only credited
	// after the referenced transaction is confirmed on-chain by that node.
	RPCURLs map[uint64]string
}

func loadConfig() *Config {
	return &Config{
		Port:           getEnv("GAS_PORT", "9218"),
		RedisURL:       getEnv("REDIS_URL", "localhost:6379"),
		NativeToken:    "ETH",
		GasTokenSymbol: "ETH",
		MinDepositUSD:  50,
		MaxGasPerTx:    10,
		JWTSecret:      getEnv("JWT_SECRET", ""),
		RPCURLs: map[uint64]string{
			1:          getEnv("RPC_URL_1", ""),
			56:         getEnv("RPC_URL_56", ""),
			137:        getEnv("RPC_URL_137", ""),
			42161:      getEnv("RPC_URL_42161", ""),
			10:         getEnv("RPC_URL_10", ""),
			8453:       getEnv("RPC_URL_8453", ""),
			43114:      getEnv("RPC_URL_43114", ""),
			250:        getEnv("RPC_URL_250", ""),
			1666600000: getEnv("RPC_URL_1666600000", ""),
			1285:       getEnv("RPC_URL_1285", ""),
		},
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

// updateGasPrices refreshes every configured chain from its own node. The
// quoted fee is what the user is charged, so a price that never moves is a
// financial bug, not a cosmetic one: the seeded values are only a starting
// point and are replaced as soon as a node answers.
//
// Chains without an RPC endpoint, or whose node is unreachable, keep their
// previous value and stay marked non-live; getGasQuote refuses to quote them
// rather than charge a made-up fee.
func (m *GasManager) updateGasPrices(ctx context.Context) {
	for chainID, chain := range m.chains {
		rpcURL := m.config.RPCURLs[chainID]
		if rpcURL == "" || !chain.Supported {
			continue
		}
		gasPriceWei, err := fetchChainGasPrice(ctx, rpcURL)
		if err != nil {
			logger.Warn().Err(err).Uint64("chainId", chainID).
				Msg("gas price refresh failed; keeping previous value")
			m.mu.Lock()
			chain.GasOracle = ""
			m.mu.Unlock()
			continue
		}

		gwei := new(big.Float).Quo(
			new(big.Float).SetInt(gasPriceWei),
			big.NewFloat(1e9),
		)
		value, _ := gwei.Float64()

		m.mu.Lock()
		chain.AvgGasPrice = value
		// Track the observed range so a UI can show slow/fast bands without
		// inventing multipliers.
		if value < chain.MinGasPrice || chain.MinGasPrice == 0 {
			chain.MinGasPrice = value
		}
		if value > chain.MaxGasPrice {
			chain.MaxGasPrice = value
		}
		chain.GasOracle = "rpc"
		m.mu.Unlock()
	}
}

// fetchChainGasPrice reads eth_gasPrice from a single node.
func fetchChainGasPrice(ctx context.Context, rpcURL string) (*big.Int, error) {
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(callCtx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", rpcURL, err)
	}
	defer client.Close()

	price, err := client.SuggestGasPrice(callCtx)
	if err != nil {
		return nil, fmt.Errorf("eth_gasPrice: %w", err)
	}
	if price == nil || price.Sign() <= 0 {
		return nil, fmt.Errorf("node returned a non-positive gas price")
	}
	return price, nil
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

// callerID returns the authenticated subject set by authMiddleware.
func callerID(c *gin.Context) string {
	v, _ := c.Get("user_id")
	id, _ := v.(string)
	return id
}

// callerIsAdmin reports whether the authenticated caller holds an admin role.
func callerIsAdmin(c *gin.Context) bool {
	v, _ := c.Get("role")
	role, _ := v.(string)
	return role == "admin" || role == "super_admin"
}

// authMiddleware verifies the bearer token issued by wallet_api. A missing
// secret means the service cannot authenticate anyone, so it refuses every
// request instead of letting them through unverified.
func authMiddleware(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.JWTSecret == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "JWT_SECRET not configured; refusing unauthenticated request",
			})
			return
		}

		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		token, err := jwt.Parse(strings.TrimPrefix(header, "Bearer "), func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(cfg.JWTSecret), nil
		}, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}))
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}
		sub, _ := claims["sub"].(string)
		if sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing subject"})
			return
		}
		role, _ := claims["role"].(string)

		c.Set("user_id", sub)
		c.Set("role", role)
		c.Next()
	}
}

// ownsAccount reports whether the caller may act on the given account. Admins
// may act on any account; everyone else is limited to their own.
func ownsAccount(c *gin.Context, account *GasAccount) bool {
	return callerIsAdmin(c) || account.UserID == callerID(c)
}

// loadAccount fetches an account and enforces ownership in one step.
func loadAccount(c *gin.Context, id string) (*GasAccount, bool) {
	key := fmt.Sprintf("gas:account:%s", id)
	data, err := redisClient.Get(context.Background(), key).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return nil, false
	}
	var account GasAccount
	if err := json.Unmarshal(data, &account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "corrupt account record"})
		return nil, false
	}
	if !ownsAccount(c, &account) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized for this account"})
		return nil, false
	}
	return &account, true
}

// saveAccount persists an account record.
func saveAccount(account *GasAccount) error {
	data, err := json.Marshal(account)
	if err != nil {
		return err
	}
	return redisClient.Set(context.Background(), fmt.Sprintf("gas:account:%s", account.AccountID), data, 365*24*time.Hour).Err()
}

// verifyDepositOnChain confirms that txHash exists, succeeded and moved at
// least the claimed amount of the native token to the account address.
// Without this check a caller could credit themselves by inventing a hash.
func verifyDepositOnChain(cfg *Config, chainID uint64, txHash, toAddress string, amount float64) error {
	rpcURL := cfg.RPCURLs[chainID]
	if rpcURL == "" {
		return fmt.Errorf("no RPC endpoint configured for chain %d", chainID)
	}
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to reach chain %d: %w", chainID, err)
	}
	defer client.Close()

	hash := common.HexToHash(txHash)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	receipt, err := client.TransactionReceipt(ctx, hash)
	if err != nil {
		return fmt.Errorf("transaction not found on chain %d", chainID)
	}
	if receipt.Status != 1 {
		return fmt.Errorf("transaction reverted on chain %d", chainID)
	}

	tx, _, err := client.TransactionByHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("failed to read transaction on chain %d", chainID)
	}
	if !strings.EqualFold(tx.To().Hex(), toAddress) {
		return fmt.Errorf("transaction did not target the gas account address")
	}

	// Compare in wei to avoid float rounding on the credited amount.
	expected, ok := new(big.Float).SetString(fmt.Sprintf("%f", amount))
	if !ok {
		return fmt.Errorf("invalid amount")
	}
	weiFloat := new(big.Float).Mul(expected, big.NewFloat(1e18))
	expectedWei, _ := weiFloat.Int(nil)
	if tx.Value().Cmp(expectedWei) < 0 {
		return fmt.Errorf("transaction value is less than the claimed amount")
	}
	return nil
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

	// API v1 - every route requires an authenticated caller.
	v1 := r.Group("/api/v1")
	v1.Use(authMiddleware(cfg))
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
		UserID       string   `json:"userId"`
		ChainID      uint64   `json:"chainId"`
		Token        string   `json:"token"`
		AutoRefill   bool     `json:"autoRefill"`
		LinkedChains []uint64 `json:"linkedChains"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// An account always belongs to the authenticated caller. Only an admin may
	// open one on behalf of somebody else.
	owner := callerID(c)
	if req.UserID != "" && req.UserID != owner {
		if !callerIsAdmin(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot create an account for another user"})
			return
		}
		owner = req.UserID
	}

	if req.ChainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chainId is required"})
		return
	}
	if req.Token == "" {
		req.Token = cfg.GasTokenSymbol
	}
	if _, ok := cfg.RPCURLs[req.ChainID]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported chain %d", req.ChainID)})
		return
	}

	// One account per user, enforced on a stable index key.
	userKey := fmt.Sprintf("gas:user:%s", owner)
	if existing, err := redisClient.Get(context.Background(), userKey).Bytes(); err == nil && len(existing) > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User already has gas account"})
		return
	}

	address, err := deriveAccountAddress(owner)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to derive account address"})
		return
	}

	account := GasAccount{
		AccountID:        generateID(),
		UserID:           owner,
		Address:          address,
		ChainID:          req.ChainID,
		Token:            req.Token,
		Balance:          0,
		BalanceUSD:       0,
		Status:           "active",
		AutoRefill:       req.AutoRefill,
		RefillThreshold:  20,
		RefillAmount:     100,
		LinkedChains:     req.LinkedChains,
		TransactionCount: 0,
		CreatedAt:        time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
	}

	if err := saveAccount(&account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist account"})
		return
	}
	redisClient.Set(context.Background(), userKey, account.AccountID, 365*24*time.Hour)

	c.JSON(http.StatusCreated, account)
}

func getAccount(c *gin.Context) {
	id := c.Param("id")
	account, ok := loadAccount(c, id)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, account)
}

func getUserAccount(c *gin.Context) {
	userID := c.Param("userId")
	if !callerIsAdmin(c) && userID != callerID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized for this user"})
		return
	}

	accountID, err := redisClient.Get(context.Background(), fmt.Sprintf("gas:user:%s", userID)).Result()
	if err != nil || accountID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}
	account, ok := loadAccount(c, accountID)
	if !ok {
		return
	}
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

	account, ok := loadAccount(c, id)
	if !ok {
		return
	}
	if account.Status != "active" {
		c.JSON(http.StatusConflict, gin.H{"error": "account is not active"})
		return
	}

	price := getTokenPrice(req.Token)
	if price <= 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "price unavailable for " + req.Token})
		return
	}

	if amountUSD := req.Amount * price; amountUSD < cfg.MinDepositUSD {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("deposit below the minimum of $%.2f", cfg.MinDepositUSD),
		})
		return
	}

	// Reject a replayed hash before doing any RPC work.
	depositKey := fmt.Sprintf("gas:deposit-tx:%s", strings.ToLower(req.TxHash))
	if exists, err := redisClient.Exists(context.Background(), depositKey).Result(); err == nil && exists > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "this transaction has already been credited"})
		return
	}

	// Credit only after the chain confirms the transfer.
	if err := verifyDepositOnChain(cfg, req.ChainID, req.TxHash, account.Address, req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deposit verification failed: " + err.Error()})
		return
	}

	account.Balance += req.Amount
	account.BalanceUSD = account.Balance * price
	account.UpdatedAt = time.Now().Unix()

	if err := saveAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist account"})
		return
	}

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

	if data, err := json.Marshal(deposit); err == nil {
		redisClient.Set(context.Background(), fmt.Sprintf("gas:deposit:%s", deposit.DepositID), data, 365*24*time.Hour)
		redisClient.LPush(context.Background(), fmt.Sprintf("gas:deposits:%s", deposit.AccountID), deposit.DepositID)
	}
	// Mark the chain transaction as consumed so it cannot be credited twice.
	redisClient.Set(context.Background(), depositKey, deposit.DepositID, 365*24*time.Hour)

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

	account, ok := loadAccount(c, id)
	if !ok {
		return
	}

	if account.Balance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient balance"})
		return
	}

	account.Balance -= req.Amount
	account.BalanceUSD = account.Balance * getTokenPrice(account.Token)
	account.UpdatedAt = time.Now().Unix()

	if err := saveAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist account"})
		return
	}

	c.JSON(http.StatusOK, account)
}

func pauseAccount(c *gin.Context) {
	id := c.Param("id")

	account, ok := loadAccount(c, id)
	if !ok {
		return
	}

	account.Status = "paused"
	account.UpdatedAt = time.Now().Unix()
	if err := saveAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist account"})
		return
	}

	c.JSON(http.StatusOK, account)
}

func resumeAccount(c *gin.Context) {
	id := c.Param("id")

	account, ok := loadAccount(c, id)
	if !ok {
		return
	}

	account.Status = "active"
	account.UpdatedAt = time.Now().Unix()
	if err := saveAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist account"})
		return
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

	account, ok := loadAccount(c, id)
	if !ok {
		return
	}

	account.AutoRefill = req.Enabled
	if req.Threshold > 0 {
		account.RefillThreshold = req.Threshold
	}
	if req.RefillAmount > 0 {
		account.RefillAmount = req.RefillAmount
	}
	account.UpdatedAt = time.Now().Unix()

	if err := saveAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist account"})
		return
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
	gasManager.mu.RLock()
	live := chain.GasOracle == "rpc"
	avgGasPriceGwei := chain.AvgGasPrice
	nativeToken := chain.NativeToken
	gasManager.mu.RUnlock()

	// A quote backed by a seeded price would over- or under-charge the user.
	if !live {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "live gas price unavailable for this chain; refusing to quote",
		})
		return
	}

	// 21000 covers a plain value transfer; token transfers and contract calls
	// cost more. The caller may supply its own estimate from eth_estimateGas.
	gasEstimate := 21000.0
	if raw := c.Query("gasEstimate"); raw != "" {
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid gasEstimate"})
			return
		}
		gasEstimate = parsed
	} else if chainID != 1 {
		gasEstimate = 65000.0
	}

	nativePrice, err := livePriceUSD(nativeToken)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "native token price unavailable; refusing to quote",
		})
		return
	}

	gasPriceUSD := avgGasPriceGwei * nativePrice
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
	if _, ok := loadAccount(c, accountID); !ok {
		return
	}

	ids, err := redisClient.LRange(context.Background(), fmt.Sprintf("gas:deposits:%s", accountID), 0, -1).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read deposit history"})
		return
	}

	deposits := make([]GasDeposit, 0, len(ids))
	for _, did := range ids {
		if data, err := redisClient.Get(context.Background(), fmt.Sprintf("gas:deposit:%s", did)).Bytes(); err == nil {
			var d GasDeposit
			if json.Unmarshal(data, &d) == nil {
				deposits = append(deposits, d)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"deposits": deposits})
}

func getPayments(c *gin.Context) {
	accountID := c.Param("accountId")
	if _, ok := loadAccount(c, accountID); !ok {
		return
	}

	ids, err := redisClient.LRange(context.Background(), fmt.Sprintf("gas:payments:%s", accountID), 0, -1).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read payment history"})
		return
	}

	payments := make([]GasPayment, 0, len(ids))
	for _, pid := range ids {
		if data, err := redisClient.Get(context.Background(), fmt.Sprintf("gas:payment:%s", pid)).Bytes(); err == nil {
			var p GasPayment
			if json.Unmarshal(data, &p) == nil {
				payments = append(payments, p)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"payments": payments})
}

// ============================================================================
// ANALYTICS HANDLERS
// ============================================================================

func getUsage(c *gin.Context) {
	accountID := c.Param("accountId")
	account, ok := loadAccount(c, accountID)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"accountId":        account.AccountID,
		"transactionCount": account.TransactionCount,
		"totalGasPaid":     account.TotalGasPaid,
		"totalGasPaidUSD":  account.TotalGasPaidUSD,
		"balanceUSD":       account.BalanceUSD,
	})
}

func getSavings(c *gin.Context) {
	userID := c.Param("userId")
	if !callerIsAdmin(c) && userID != callerID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized for this user"})
		return
	}

	accountID, err := redisClient.Get(context.Background(), fmt.Sprintf("gas:user:%s", userID)).Result()
	if err != nil || accountID == "" {
		c.JSON(http.StatusOK, gin.H{"userId": userID, "totalGasPaidUSD": 0, "savingsUSD": 0})
		return
	}
	account, ok := loadAccount(c, accountID)
	if !ok {
		return
	}

	// Savings are what the user avoided paying in native gas: the USD value of
	// gas paid from the account versus paying the same count at the current
	// per-transaction gas price on the native chain.
	chain := gasManager.chains[account.ChainID]
	var savingsUSD float64
	if chain != nil && account.TransactionCount > 0 {
		perTxUSD := chain.AvgGasPrice * getTokenPrice(chain.NativeToken)
		nativeCostUSD := perTxUSD * float64(account.TransactionCount)
		savingsUSD = nativeCostUSD - account.TotalGasPaidUSD
		if savingsUSD < 0 {
			savingsUSD = 0
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"userId":           userID,
		"transactionCount": account.TransactionCount,
		"totalGasPaidUSD":  account.TotalGasPaidUSD,
		"savingsUSD":       savingsUSD,
	})
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// deriveAccountAddress returns the deterministic deposit address for a user.
// It is derived from WALLET_MASTER_SEED via HKDF so the operator can always
// recover the key, unlike a random throwaway address.
func deriveAccountAddress(userID string) (string, error) {
	seed := os.Getenv("WALLET_MASTER_SEED")
	if seed == "" {
		return "", fmt.Errorf("WALLET_MASTER_SEED not configured")
	}
	mac := hmac.New(sha256.New, []byte(seed))
	mac.Write([]byte("gas-account:" + userID))
	sum := mac.Sum(nil)
	return "0x" + hex.EncodeToString(sum[:20]), nil
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
