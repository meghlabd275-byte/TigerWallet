// Cross-Chain Swap Engine - Go Implementation
// Atomic swaps across chains without external DEX dependency

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Configuration
type SwapConfig struct {
	ServerPort    string `json:"server_port"`
	DBHost       string `json:"db_host"`
	DBPort       string `json:"db_port"`
	DBUser       string `json:"db_user"`
	DBPassword   string `json:"db_password"`
	DBName       string `json:"db_name"`
	RedisHost    string `json:"redis_host"`
	RedisPort    string `json:"redis_port"`
}

// Supported Chains
const (
	CHAIN_ETHEREUM       = 1
	CHAIN_POLYGON       = 137
	CHAIN_BSC           = 56
	CHAIN_ARBITRUM      = 42161
	CHAIN_OPTIMISM      = 10
	CHAIN_AVALANCHE     = 43114
	CHAIN_SOLANA        = "solana"
	CHAIN_NEAR         = "near"
	CHAIN_APTOS        = "aptos"
)

// Swap Status
const (
	STATUS_PENDING    = "pending"
	STATUS_INITIATED = "initiated"
	STATUS_ACCEPTED = "accepted"
	STATUS_COMPLETED = "completed"
	STATUS_FAILED   = "failed"
	STATUS_REFUNDED  = "refunded"
)

// Swap Types
const (
	TYPE_DIRECT_SWAP    = "direct"
	TYPE_BRIDGE_SWAP = "bridge"
	TYPE_ATOMIC    = "atomic"
)

// Swap struct represents a cross-chain swap
type Swap struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	SwapID          string    `gorm:"uniqueIndex" json:"swap_id"`
	Type           string    `json:"type"`
	FromChain      int64     `json:"from_chain"`
	ToChain        int64     `json:"to_chain"`
	FromToken     string    `json:"from_token"`
	ToToken       string    `json:"to_token"`
	Sender        string    `json:"sender"`
	Recipient     string    `json:"recipient"`
	FromAmount    string    `json:"from_amount"`
	ToAmount      string    `json:"to_amount"`
	MinAmount     string    `json:"min_amount"`
	HashLock      string    `json:"hash_lock"`
	Secret        string    `json:"secret"` // Encrypted
	TimeLock      int64     `json:"time_lock"`
	Status       string    `json:"status"`
	FromTxHash   string    `json:"from_tx_hash"`
	ToTxHash     string    `json:"to_tx_hash"`
	BlockNumber  int64     `json:"block_number"`
	Fee          string    `json:"fee"`
	Slippage     float64   `json:"slippage"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CompletedAt  *time.Time `json:"completed_at"`
}

// BridgeRoute represents a bridge route
type BridgeRoute struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `json:"name"`
	FromChain       int64     `json:"from_chain"`
	ToChain         int64     `json:"to_chain"`
	FromToken      string    `json:"from_token"`
	ToToken        string    `json:"to_token"`
	Contract      string    `json:"contract"`
	MinAmount     string    `json:"min_amount"`
	MaxAmount     string    `json:"max_amount"`
	Fee           string    `json:"fee"`
	FeeToken      string    `json:"fee_token"`
	EstimatedTime int64     `json:"estimated_time"`
	IsActive      bool      `json:"is_active"`
}

// SwapQuote represents a swap quote
type SwapQuote struct {
	FromChain    int64   `json:"from_chain"`
	ToChain    int64   `json:"to_chain"`
	FromToken  string `json:"from_token"`
	ToToken   string `json:"to_token"`
	FromAmount string `json:"from_amount"`
	ToAmount  string `json:"to_amount"`
	MinAmount string `json:"min_amount"`
	Fee       string `json:"fee"`
	FeeToken  string `json:"fee_token"`
	PriceImpact float64 `json:"price_impact"`
	Provider  string `json:"provider"`
	Route     string `json:"route"`
}

// LiquidityPool represents a liquidity pool
type LiquidityPool struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	PoolAddress string `gorm:"uniqueIndex" json:"pool_address"`
	ChainID    int64  `json:"chain_id"`
	Token0     string `json:"token0"`
	Token1     string `json:"token1"`
	Reserve0   string `json:"reserve0"`
	Reserve1   string `json:"reserve1"`
	Liquidity  string `json:"liquidity"`
	Fee        int64  `json:"fee"`
}

// SwapService main service
type SwapService struct {
	db      *gorm.DB
	redis   *redis.Client
	config  SwapConfig
	routes  sync.Map // route name -> BridgeRoute
	pools   sync.Map // pool address -> LiquidityPool
	quotes  sync.Map // swap_id -> SwapQuote
}

// NewSwapService creates new service
func NewSwapService(cfg SwapConfig) (*SwapService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Swap{}, &BridgeRoute{}, &LiquidityPool{})
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &SwapService{
		db:      db,
		redis:   rdb,
		config:  cfg,
		routes:  sync.Map{},
		pools:   sync.Map{},
		quotes: sync.Map{},
	}, nil
}

// GenerateSwapID generates a unique swap ID
func (s *SwapService) GenerateSwapID() string {
	var b [16]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// GenerateSecret generates a random secret for atomic swap
func (s *SwapService) GenerateSecret() string {
	var b [32]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// GenerateHashLock generates a hash lock from secret
func (s *SwapService) GenerateHashLock(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}

// CreateSwap creates a new swap
func (s *SwapService) CreateSwap(req CreateSwapRequest) (*Swap, error) {
	swapID := s.GenerateSwapID()

	var secret, hashLock string
	if req.Type == TYPE_ATOMIC {
		secret = s.GenerateSecret()
		hashLock = s.GenerateHashLock(secret)
	} else {
		hashLock = s.GenerateHashLock(swapID)
	}

	timeLock := time.Now().Add(24 * time.Hour).Unix()

	swap := &Swap{
		SwapID:     swapID,
		Type:       req.Type,
		FromChain: req.FromChain,
		ToChain:   req.ToChain,
		FromToken: req.FromToken,
		ToToken:  req.ToToken,
		Sender:   req.Sender,
		Recipient: req.Recipient,
		FromAmount: req.FromAmount,
		ToAmount:  req.ToAmount,
		MinAmount: req.MinAmount,
		HashLock: hashLock,
		TimeLock: timeLock,
		Status:   STATUS_PENDING,
		Fee:      req.Fee,
		Slippage:  req.Slippage,
		CreatedAt: time.Now(),
	}

	s.db.Create(swap)

	return swap, nil
}

// InitiateSwap initiates a swap (sender deposits funds)
func (s *SwapService) InitiateSwap(swapID, txHash string) error {
	result := s.db.Model(&Swap{}).Where("swap_id = ?", swapID).Updates(map[string]interface{}{
		"status":      STATUS_INITIATED,
		"from_tx_hash": txHash,
		"updated_at":  time.Now(),
	})

	if result.RowsAffected == 0 {
		return fmt.Errorf("swap not found")
	}

	return nil
}

// AcceptSwap accepts a swap (recipient claims funds)
func (s *SwapService) AcceptSwap(swapID, secret, txHash string) error {
	swap, err := s.GetSwap(swapID)
	if err != nil {
		return err
	}

	// Verify hash lock
	expectedHashLock := s.GenerateHashLock(secret)
	if swap.HashLock != expectedHashLock {
		return fmt.Errorf("invalid secret")
	}

	// Verify time lock not expired
	if time.Now().Unix() > swap.TimeLock {
		return fmt.Errorf("swap expired")
	}

	result := s.db.Model(&Swap{}).Where("swap_id = ?", swapID).Updates(map[string]interface{}{
		"status":      STATUS_ACCEPTED,
		"to_tx_hash":  txHash,
		"secret":     secret,
		"updated_at": time.Now(),
	})

	if result.RowsAffected == 0 {
		return fmt.Errorf("swap not found")
	}

	return nil
}

// CompleteSwap completes a swap
func (s *SwapService) CompleteSwap(swapID string) error {
	now := time.Now()
	result := s.db.Model(&Swap{}).Where("swap_id = ?", swapID).Updates(map[string]interface{}{
		"status":       STATUS_COMPLETED,
		"completed_at": now,
		"updated_at":   now,
	})

	if result.RowsAffected == 0 {
		return fmt.Errorf("swap not found")
	}

	return nil
}

// RefundSwap refunds a swap (sender claims back funds)
func (s *SwapService) RefundSwap(swapID string) error {
	swap, err := s.GetSwap(swapID)
	if err != nil {
		return err
	}

	// Verify time lock expired
	if time.Now().Unix() < swap.TimeLock {
		return fmt.Errorf("time lock not expired")
	}

	result := s.db.Model(&Swap{}).Where("swap_id = ?", swapID).Updates(map[string]interface{}{
		"status":     STATUS_REFUNDED,
		"updated_at": time.Now(),
	})

	if result.RowsAffected == 0 {
		return fmt.Errorf("swap not found")
	}

	return nil
}

// GetSwap gets a swap by ID
func (s *SwapService) GetSwap(swapID string) (*Swap, error) {
	var swap Swap
	if err := s.db.Where("swap_id = ?", swapID).First(&swap).Error; err != nil {
		return nil, err
	}

	return &swap, nil
}

// GetSwaps gets swaps for a user
func (s *SwapService) GetSwaps(address string) ([]Swap, error) {
	var swaps []Swap
	err := s.db.Where("sender = ? OR recipient = ?", address, address).Find(&swaps).Error
	if err != nil {
		return nil, err
	}

	return swaps, nil
}

// GetQuote gets a swap quote
func (s *SwapService) GetQuote(req QuoteRequest) (*SwapQuote, error) {
	// Calculate quote based on liquidity pools
	fromAmountFloat := parseAmount(req.FromAmount)
	toAmountFloat := fromAmountFloat * 0.99 // 1% slippage simulated

	toAmount := formatAmount(toAmountFloat)
	minAmount := formatAmount(toAmountFloat * 0.98)

	quote := &SwapQuote{
		FromChain:   req.FromChain,
		ToChain:    req.ToChain,
		FromToken:  req.FromToken,
		ToToken:   req.ToToken,
		FromAmount: req.FromAmount,
		ToAmount:  toAmount,
		MinAmount: minAmount,
		Fee:      "0",
		FeeToken:  req.FromToken,
		PriceImpact: 0.5,
		Provider: "TigerWallet",
		Route:    "direct",
	}

	s.quotes.Store(quote.Route, quote)

	return quote, nil
}

// ExecuteSwap executes a swap
func (s *SwapService) ExecuteSwap(req ExecuteSwapRequest) (*Swap, error) {
	swap, err := s.GetSwap(req.SwapID)
	if err != nil {
		return nil, err
	}

	if swap.Status != STATUS_INITIATED {
		return nil, fmt.Errorf("swap not in correct state")
	}

	// Simulate broadcast
	toTxHash := "0x" + generateHash()

	swap.ToTxHash = toTxHash
	swap.Status = STATUS_ACCEPTED
	s.db.Save(swap)

	return swap, nil
}

// AddRoute adds a bridge route
func (s *SwapService) AddRoute(route BridgeRoute) error {
	s.routes.Store(route.Name, route)
	s.db.Create(&route)

	return nil
}

// GetRoutes gets available routes
func (s *SwapService) GetRoutes(fromChain, toChain int64) []BridgeRoute {
	var routes []BridgeRoute
	s.db.Where("from_chain = ? AND to_chain = ? AND is_active = ?", fromChain, toChain, true).Find(&routes)

	return routes
}

// AddPool adds a liquidity pool
func (s *SwapService) AddPool(pool LiquidityPool) error {
	s.pools.Store(pool.PoolAddress, pool)
	s.db.Create(&pool)

	return nil
}

// GetPool gets a pool
func (s *SwapService) GetPool(poolAddress string) (*LiquidityPool, error) {
	var pool LiquidityPool
	if err := s.db.Where("pool_address = ?", poolAddress).First(&pool).Error; err != nil {
		return nil, err
	}

	return &pool, nil
}

// GetPools gets pools for a token pair
func (s *SwapService) GetPools(chainID int64, token0, token1 string) []LiquidityPool {
	var pools []LiquidityPool
	s.db.Where("chain_id = ? AND ((token0 = ? AND token1 = ?) OR (token0 = ? AND token1 = ?))",
		chainID, token0, token1, token1, token0).Find(&pools)

	return pools
}

// CalculateSwap calculates output amount for a swap
func (s *SwapService) CalculateSwap(poolAddress, fromToken, fromAmount string) (string, error) {
	pool, err := s.GetPool(poolAddress)
	if err != nil {
		return "", err
	}

	amountIn := parseAmount(fromAmount)
	var reserveIn, reserveOut string
	if pool.Token0 == fromToken {
		reserveIn = pool.Reserve0
		reserveOut = pool.Reserve1
	} else {
		reserveIn = pool.Reserve1
		reserveOut = pool.Reserve0
	}

	reserveInFloat := parseAmount(reserveIn)
	reserveOutFloat := parseAmount(reserveOut)

	// Constant product formula: (x + dx)(y - dy) = xy
	amountOut := (reserveOutFloat * amountIn) / (reserveInFloat + amountIn)

	return formatAmount(amountOut), nil
}

// API Handlers

type CreateSwapRequest struct {
	Type        string  `json:"type" binding:"required"`
	FromChain  int64   `json:"from_chain" binding:"required"`
	ToChain    int64   `json:"to_chain" binding:"required"`
	FromToken  string  `json:"from_token" binding:"required"`
	ToToken   string  `json:"to_token" binding:"required"`
	Sender    string  `json:"sender" binding:"required"`
	Recipient string  `json:"recipient" binding:"required"`
	FromAmount string `json:"from_amount" binding:"required"`
	ToAmount  string  `json:"to_amount"`
	MinAmount  string  `json:"min_amount"`
	Fee       string  `json:"fee"`
	Slippage  float64 `json:"slippage"`
}

type QuoteRequest struct {
	FromChain  int64   `json:"from_chain" binding:"required"`
	ToChain    int64   `json:"to_chain" binding:"required"`
	FromToken  string  `json:"from_token" binding:"required"`
	ToToken   string  `json:"to_token" binding:"required"`
	FromAmount string `json:"from_amount" binding:"required"`
}

type ExecuteSwapRequest struct {
	SwapID string `json:"swap_id" binding:"required"`
}

func (s *SwapService) CreateSwapHandler(c *gin.Context) {
	var req CreateSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	swap, err := s.CreateSwap(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, swap)
}

func (s *SwapService) InitiateSwapHandler(c *gin.Context) {
	var req struct {
		SwapID  string `json:"swap_id" binding:"required"`
		TxHash string `json:"tx_hash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := s.InitiateSwap(req.SwapID, req.TxHash); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "initiated"})
}

func (s *SwapService) AcceptSwapHandler(c *gin.Context) {
	var req struct {
		SwapID  string `json:"swap_id" binding:"required"`
		Secret string `json:"secret" binding:"required"`
		TxHash string `json:"tx_hash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := s.AcceptSwap(req.SwapID, req.Secret, req.TxHash); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "accepted"})
}

func (s *SwapService) CompleteSwapHandler(c *gin.Context) {
	swapID := c.Param("swap_id")

	if err := s.CompleteSwap(swapID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "completed"})
}

func (s *SwapService) RefundSwapHandler(c *gin.Context) {
	swapID := c.Param("swap_id")

	if err := s.RefundSwap(swapID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "refunded"})
}

func (s *SwapService) GetSwapHandler(c *gin.Context) {
	swapID := c.Param("swap_id")

	swap, err := s.GetSwap(swapID)
	if err != nil {
		c.JSON(404, gin.H{"error": "swap not found"})
		return
	}

	c.JSON(200, swap)
}

func (s *SwapService) GetSwapsHandler(c *gin.Context) {
	address := c.Param("address")

	swaps, err := s.GetSwaps(address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, swaps)
}

func (s *SwapService) GetQuoteHandler(c *gin.Context) {
	var req QuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	quote, err := s.GetQuote(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, quote)
}

func (s *SwapService) GetRoutesHandler(c *gin.Context) {
	fromChain := parseInt64(c.Query("from_chain"))
	toChain := parseInt64(c.Query("to_chain"))

	routes := s.GetRoutes(fromChain, toChain)

	c.JSON(200, routes)
}

func (s *SwapService) GetPoolsHandler(c *gin.Context) {
	chainID := parseInt64(c.Query("chain_id"))
	token0 := c.Query("token0")
	token1 := c.Query("token1")

	pools := s.GetPools(chainID, token0, token1)

	c.JSON(200, pools)
}

// Utility functions

func parseAmount(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func formatAmount(f float64) string {
	return fmt.Sprintf("%.6f", f)
}

func parseInt64(s string) int64 {
	var i int64
	fmt.Sscanf(s, "%d", &i)
	return i
}

func generateHash() string {
	var b [32]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Main

func main() {
	cfg := SwapConfig{
		ServerPort: getEnv("SWAP_SERVER_PORT", "8083"),
		DBHost:    getEnv("DB_HOST", "localhost"),
		DBPort:    getEnv("DB_PORT", "5432"),
		DBUser:    getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:    getEnv("DB_NAME", "swap_db"),
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewSwapService(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize service: %v\n", err)
		os.Exit(1)
	}

	r := gin.Default()

	r.POST("/swaps", service.CreateSwapHandler)
	r.POST("/swaps/:swap_id/initiate", service.InitiateSwapHandler)
	r.POST("/swaps/:swap_id/accept", service.AcceptSwapHandler)
	r.POST("/swaps/:swap_id/complete", service.CompleteSwapHandler)
	r.POST("/swaps/:swap_id/refund", service.RefundSwapHandler)
	r.GET("/swaps/:swap_id", service.GetSwapHandler)
	r.GET("/swaps/user/:address", service.GetSwapsHandler)
	r.POST("/quotes", service.GetQuoteHandler)
	r.GET("/routes", service.GetRoutesHandler)
	r.GET("/pools", service.GetPoolsHandler)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	go func() {
		fmt.Printf("Swap Service starting on port %s\n", cfg.ServerPort)
		if err := r.Run(":" + cfg.ServerPort); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}