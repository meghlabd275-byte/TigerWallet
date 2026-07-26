/**
 * TigerWallet DeFi/Earn Module
 * Comprehensive DeFi services including Launchpad, Staking, Earn products, ETF trading
 */

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
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
}

// Launchpad Types
type LaunchpadStatus string

const (
	LaunchpadStatusUpcoming LaunchpadStatus = "UPCOMING"
	LaunchpadStatusActive   LaunchpadStatus = "ACTIVE"
	LaunchpadStatusEnded    LaunchpadStatus = "ENDED"
	LaunchpadStatusCancelled LaunchpadStatus = "CANCELLED"
)

// Launchpad Project
type LaunchpadProject struct {
	ProjectID          string            `json:"project_id"`
	Name               string            `json:"name"`
	Symbol             string            `json:"symbol"`
	Description        string            `json:"description"`
	LogoURL            string            `json:"logo_url"`
	WebsiteURL         string            `json:"website_url"`
	WhitepaperURL     string            `json:"whitepaper_url"`
	Chain              string            `json:"chain"`
	TokenAddress       string            `json:"token_address"`
	TotalSupply       string            `json:"total_supply"`
	InitialLiquidity   string            `json:"initial_liquidity"`
	LaunchPrice        string            `json:"launch_price"`
	HardCap            string            `json:"hard_cap"`
	SoftCap            string            `json:"soft_cap"`
	MinAllocation      string            `json:"min_allocation"`
	MaxAllocation      string            `json:"max_allocation"`
	StartTime          time.Time        `json:"start_time"`
	EndTime            time.Time        `json:"end_time"`
	Status             LaunchpadStatus   `json:"status"`
	Participants       int              `json:"participants"`
	TotalRaised        string            `json:"total_raised"`
	CreatedAt          time.Time        `json:"created_at"`
}

// Staking Pool
type StakingPool struct {
	PoolID          string    `json:"pool_id"`
	Name            string    `json:"name"`
	Token           string    `json:"token"`
	Chain           string    `json:"chain"`
	RewardToken     string    `json:"reward_token"`
	APY             float64   `json:"apy"`
	LockPeriod       int      `json:"lock_period"` // in days
	MinStake         string    `json:"min_stake"`
	MaxStake         string    `json:"max_stake"`
	TotalStaked     string    `json:"total_staked"`
	RewardDistribution string `json:"reward_distribution"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

// Earn Product
type EarnProduct struct {
	ProductID     string    `json:"product_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"` // flexible, locked, structured
	Token         string    `json:"token"`
	Chain        string    `json:"chain"`
	APY          float64   `json:"apy"`
	MinAmount    string    `json:"min_amount"`
	MaxAmount    string    `json:"max_amount"`
	Duration     int       `json:"duration"` // in days
	InterestType string    `json:"interest_type"` // fixed, variable
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// Savings Account
type SavingsAccount struct {
	AccountID    string    `json:"account_id"`
	UserID       string    `json:"user_id"`
	ProductID    string    `json:"product_id"`
	Principal    string    `json:"principal"`
	Interest     string    `json:"interest"`
	APY          float64   `json:"apy"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Status       string    `json:"status"`
}

// ETF Product
type ETFProduct struct {
	ProductID    string    `json:"product_id"`
	Name         string    `json:"name"`
	Symbol       string    `json:"symbol"`
	Description string    `json:"description"`
	Underlying  []string `json:"underlying"` // token addresses
	Weights     []float64 `json:"weights"`
	ManagerFee  float64   `json:"manager_fee"`
	TokenPrice  string    `json:"token_price"`
	TotalSupply string    `json:"total_supply"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Coupon
type Coupon struct {
	CouponID     string    `json:"coupon_id"`
	Code        string    `json:"code"`
	Type        string    `json:"type"` // discount, reward, cashback
	Value       float64   `json:"value"`
	MinPurchase string    `json:"min_purchase"`
	ValidFrom   time.Time `json:"valid_from"`
	ValidUntil  time.Time `json:"valid_until"`
	MaxUses     int       `json:"max_uses"`
	UsedCount   int       `json:"used_count"`
	Status      string    `json:"status"`
}

// Red Packet
type RedPacket struct {
	PacketID    string    `json:"packet_id"`
	CreatorID   string    `json:"creator_id"`
	Token       string    `json:"token"`
	Chain       string    `json:"chain"`
	Amount      string    `json:"amount"`
	Count       int       `json:"count"`
	Claimed     int       `json:"claimed"`
	Message     string    `json:"message"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Conversion
type Conversion struct {
	ConversionID  string    `json:"conversion_id"`
	UserID        string    `json:"user_id"`
	FromToken     string    `json:"from_token"`
	ToToken       string    `json:"to_token"`
	FromAmount    string    `json:"from_amount"`
	ToAmount      string    `json:"to_amount"`
	Rate          string    `json:"rate"`
	Status        string    `json:"status"`
	Chain         string    `json:"chain"`
	Timestamp     time.Time `json:"timestamp"`
}

// DeFi Service
type DeFiService struct {
	config       Config
	redis       *redis.Client
	launchpads  map[string]*LaunchpadProject
	stakingPools map[string]*StakingPool
	earnProducts map[string]*EarnProduct
	etfProducts map[string]*ETFProduct
	coupons     map[string]*Coupon
	accounts    map[string]*SavingsAccount
	conversions map[string]*Conversion
	mu          sync.RWMutex
}

// NewDeFiService creates a new DeFi service
func NewDeFiService(cfg Config) *RedisAddr {
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		DB:   6,
	})

	service := &DeFiService{
		config:       cfg,
		redis:        redisClient,
		launchpads:  make(map[string]*LaunchpadProject),
		stakingPools: make(map[string]*StakingPool),
		earnProducts: make(map[string]*EarnProduct),
		etfProducts: make(map[string]*ETFProduct),
		coupons:     make(map[string]*Coupon),
		accounts:    make(map[string]*SavingsAccount),
		conversions: make(map[string]*Conversion),
	}

	// Initialize with sample data
	service.initializeLaunchpads()
	service.initializeStakingPools()
	service.initializeEarnProducts()
	service.initializeETFs()
	service.initializeCoupons()

	return service
}

func (s *DeFiService) initializeLaunchpads() {
	launchpads := []*LaunchpadProject{
		{
			ProjectID: "lp_001", Name: "Tiger Token", Symbol: "TIGER",
			Description: "The native token of TigerWallet ecosystem", LogoURL: "https://example.com/logo.png",
			Chain: "ETHEREUM", TotalSupply: "1000000000", HardCap: "500000", SoftCap: "100000",
			MinAllocation: "100", MaxAllocation: "5000", APY: 0,
			Status: LaunchpadStatusActive, StartTime: time.Now(), EndTime: time.Now().Add(7 * 24 * time.Hour),
		},
		{
			ProjectID: "lp_002", Name: "DeFi Protocol", Symbol: "DEF",
			Description: "Next generation DeFi protocol", LogoURL: "https://example.com/def.png",
			Chain: "BSC", TotalSupply: "500000000", HardCap: "300000", SoftCap: "50000",
			MinAllocation: "50", MaxAllocation: "2000", APY: 0,
			Status: LaunchpadStatusUpcoming, StartTime: time.Now().Add(14 * 24 * time.Hour), EndTime: time.Now().Add(21 * 24 * time.Hour),
		},
	}

	for _, lp := range launchpads {
		lp.CreatedAt = time.Now()
		s.launchpads[lp.ProjectID] = lp
	}
}

func (s *DeFiService) initializeStakingPools() {
	pools := []*StakingPool{
		{
			PoolID: "stake_eth_01", Name: "Ethereum Staking", Token: "ETH", Chain: "ETHEREUM",
			RewardToken: "ETH", APY: 4.5, LockPeriod: 0, MinStake: "0.01", MaxStake: "10000",
			TotalStaked: "150000", RewardDistribution: "daily", Status: "ACTIVE",
		},
		{
			PoolID: "stake_bnb_01", Name: "BNB Staking", Token: "BNB", Chain: "BSC",
			RewardToken: "BNB", APY: 8.2, LockPeriod: 30, MinStake: "0.1", MaxStake: "1000",
			TotalStaked: "50000", RewardDistribution: "daily", Status: "ACTIVE",
		},
		{
			PoolID: "stake_sol_01", Name: "Solana Staking", Token: "SOL", Chain: "SOLANA",
			RewardToken: "SOL", APY: 6.8, LockPeriod: 0, MinStake: "1", MaxStake: "10000",
			TotalStaked: "200000", RewardDistribution: "daily", Status: "ACTIVE",
		},
		{
			PoolID: "stake_dot_01", Name: "Polkadot Staking", Token: "DOT", Chain: "POLKADOT",
			RewardToken: "DOT", APY: 12.5, LockPeriod: 28, MinStake: "10", MaxStake: "5000",
			TotalStaked: "100000", RewardDistribution: "daily", Status: "ACTIVE",
		},
		{
			PoolID: "stake_ada_01", Name: "Cardano Staking", Token: "ADA", Chain: "CARDANO",
			RewardToken: "ADA", APY: 4.2, LockPeriod: 0, MinStake: "100", MaxStake: "100000",
			TotalStaked: "500000", RewardDistribution: "epoch", Status: "ACTIVE",
		},
		{
			PoolID: "stake_apt_01", Name: "Aptos Staking", Token: "APT", Chain: "APTOS",
			RewardToken: "APT", APY: 7.5, LockPeriod: 0, MinStake: "10", MaxStake: "5000",
			TotalStaked: "75000", RewardDistribution: "daily", Status: "ACTIVE",
		},
		{
			PoolID: "stake_sui_01", Name: "Sui Staking", Token: "SUI", Chain: "SUI",
			RewardToken: "SUI", APY: 6.2, LockPeriod: 0, MinStake: "100", MaxStake: "10000",
			TotalStaked: "60000", RewardDistribution: "daily", Status: "ACTIVE",
		},
		{
			PoolID: "stake_near_01", Name: "NEAR Staking", Token: "NEAR", Chain: "NEAR",
			RewardToken: "NEAR", APY: 10.0, LockPeriod: 0, MinStake: "10", MaxStake: "10000",
			TotalStaked: "80000", RewardDistribution: "daily", Status: "ACTIVE",
		},
		{
			PoolID: "stake_atom_01", Name: "Cosmos Staking", Token: "ATOM", Chain: "COSMOS",
			RewardToken: "ATOM", APY: 9.8, LockPeriod: 21, MinStake: "1", MaxStake: "10000",
			TotalStaked: "120000", RewardDistribution: "daily", Status: "ACTIVE",
		},
	}

	for _, pool := range pools {
		pool.CreatedAt = time.Now()
		s.stakingPools[pool.PoolID] = pool
	}
}

func (s *DeFiService) initializeEarnProducts() {
	products := []*EarnProduct{
		{
			ProductID: "earn_eth_01", Name: "ETH Flexible Savings", Type: "flexible",
			Token: "ETH", Chain: "ETHEREUM", APY: 3.5, MinAmount: "0.01", MaxAmount: "100",
			InterestType: "variable", Status: "ACTIVE",
		},
		{
			ProductID: "earn_usdt_01", Name: "USDT Fixed Savings", Type: "locked",
			Token: "USDT", Chain: "ETHEREUM", APY: 5.2, MinAmount: "100", MaxAmount: "100000",
			Duration: 30, InterestType: "fixed", Status: "ACTIVE",
		},
		{
			ProductID: "earn_usdc_01", Name: "USDC Flexible Savings", Type: "flexible",
			Token: "USDC", Chain: "ETHEREUM", APY: 4.0, MinAmount: "100", MaxAmount: "100000",
			InterestType: "variable", Status: "ACTIVE",
		},
		{
			ProductID: "earn_btc_01", Name: "WBTC Earn", Type: "flexible",
			Token: "WBTC", Chain: "ETHEREUM", APY: 2.5, MinAmount: "0.001", MaxAmount: "10",
			InterestType: "variable", Status: "ACTIVE",
		},
		{
			ProductID: "earn_bnb_01", Name: "BNB Flexible Savings", Type: "flexible",
			Token: "BNB", Chain: "BSC", APY: 4.8, MinAmount: "0.01", MaxAmount: "1000",
			InterestType: "variable", Status: "ACTIVE",
		},
		{
			ProductID: "earn_sol_01", Name: "SOL Flexible Savings", Type: "flexible",
			Token: "SOL", Chain: "SOLANA", APY: 5.0, MinAmount: "1", MaxAmount: "1000",
			InterestType: "variable", Status: "ACTIVE",
		},
		{
			ProductID: "earn_struct_01", Name: "Structured Product Alpha", Type: "structured",
			Token: "USDT", Chain: "ETHEREUM", APY: 12.0, MinAmount: "1000", MaxAmount: "100000",
			Duration: 90, InterestType: "fixed", Status: "ACTIVE",
		},
	}

	for _, p := range products {
		p.CreatedAt = time.Now()
		s.earnProducts[p.ProductID] = p
	}
}

func (s *DeFiService) initializeETFs() {
	etfs := []*ETFProduct{
		{
			ProductID: "etf_crypto_01", Name: "Crypto Blue Chip ETF", Symbol: "CBC",
			Description: "Diversified portfolio of top cryptocurrencies",
			Underlying: []string{"BTC", "ETH", "BNB"},
			Weights: []float64{40, 40, 20},
			ManagerFee: 0.5, Status: "ACTIVE",
		},
		{
			ProductID: "etf_defi_01", Name: "DeFi Index ETF", Symbol: "DEF",
			Description: "Exposure to leading DeFi protocols",
			Underlying: []string{"UNI", "AAVE", "MKR", "COMP"},
			Weights: []float64{30, 30, 25, 15},
			ManagerFee: 0.75, Status: "ACTIVE",
		},
		{
			ProductID: "etf_stable_01", Name: "Stablecoin Yield ETF", Symbol: "SYE",
			Description: "Stable returns from stablecoin lending",
			Underlying: []string{"USDT", "USDC", "DAI"},
			Weights: []float64{40, 40, 20},
			ManagerFee: 0.25, Status: "ACTIVE",
		},
	}

	for _, e := range etfs {
		e.CreatedAt = time.Now()
		s.etfProducts[e.ProductID] = e
	}
}

func (s *DeFiService) initializeCoupons() {
	coupons := []*Coupon{
		{
			CouponID: "coupon_001", Code: "WELCOME10", Type: "discount",
			Value: 10, MinPurchase: "100", ValidFrom: time.Now(),
			ValidUntil: time.Now().Add(30 * 24 * time.Hour), MaxUses: 10000, Status: "ACTIVE",
		},
		{
			CouponID: "coupon_002", Code: "STAKE20", Type: "reward",
			Value: 20, MinPurchase: "0", ValidFrom: time.Now(),
			ValidUntil: time.Now().Add(60 * 24 * time.Hour), MaxUses: 5000, Status: "ACTIVE",
		},
	}

	for _, c := range coupons {
		s.coupons[c.CouponID] = c
	}
}

// Launchpad functions
func (s *DeFiService) GetLaunchpads(status string) []*LaunchpadProject {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*LaunchpadProject, 0)
	for _, lp := range s.launchpads {
		if status == "" || string(lp.Status) == status {
			result = append(result, lp)
		}
	}
	return result
}

func (s *DeFiService) GetLaunchpad(projectID string) (*LaunchpadProject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lp, ok := s.launchpads[projectID]
	if !ok {
		return nil, fmt.Errorf("project not found")
	}
	return lp, nil
}

func (s *DeFiService) CreateLaunchpad(lp *LaunchpadProject) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lp.ProjectID = "lp_" + uuid.New().String()[:8]
	lp.Status = LaunchpadStatusUpcoming
	lp.CreatedAt = time.Now()

	s.launchpads[lp.ProjectID] = lp
	return nil
}

// Staking functions
func (s *DeFiService) GetStakingPools(token, chain string) []*StakingPool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*StakingPool, 0)
	for _, pool := range s.stakingPools {
		match := true
		if token != "" && pool.Token != token {
			match = false
		}
		if chain != "" && pool.Chain != chain {
			match = false
		}
		if match {
			result = append(result, pool)
		}
	}
	return result
}

func (s *DeFiService) CreateStake(userID, poolID, amount string) (*SavingsAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.stakingPools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found")
	}

	account := &SavingsAccount{
		AccountID: "stake_" + uuid.New().String()[:8],
		UserID:   userID,
		ProductID: poolID,
		Principal: amount,
		APY:     pool.APY,
		Status:  "ACTIVE",
		StartTime: time.Now(),
	}

	s.accounts[account.AccountID] = account
	return account, nil
}

// Earn functions
func (s *DeFiService) GetEarnProducts(token, productType string) []*EarnProduct {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*EarnProduct, 0)
	for _, p := range s.earnProducts {
		match := true
		if token != "" && p.Token != token {
			match = false
		}
		if productType != "" && p.Type != productType {
			match = false
		}
		if match {
			result = append(result, p)
		}
	}
	return result
}

func (s *DeFiService) CreateEarnPosition(userID, productID, amount string) (*SavingsAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, ok := s.earnProducts[productID]
	if !ok {
		return nil, fmt.Errorf("product not found")
	}

	account := &SavingsAccount{
		AccountID: "earn_" + uuid.New().String()[:8],
		UserID:   userID,
		ProductID: productID,
		Principal: amount,
		APY:     product.APY,
		Status:  "ACTIVE",
		StartTime: time.Now(),
		EndTime: time.Now().Add(time.Duration(product.Duration) * 24 * time.Hour),
	}

	s.accounts[account.AccountID] = account
	return account, nil
}

// ETF functions
func (s *DeFiService) GetETFProducts() []*ETFProduct {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*ETFProduct, 0)
	for _, e := range s.etfProducts {
		result = append(result, e)
	}
	return result
}

// Coupon functions
func (s *DeFiService) ValidateCoupon(code string) (*Coupon, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.coupons {
		if c.Code == code && c.Status == "ACTIVE" {
			if time.Now().Before(c.ValidUntil) && time.Now().After(c.ValidFrom) {
				if c.MaxUses == 0 || c.UsedCount < c.MaxUses {
					return c, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("invalid or expired coupon")
}

// Conversion functions
func (s *DeFiService) ConvertToken(userID, fromToken, toToken, fromAmount, chain string) (*Conversion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Mock rate calculation (in production, use real oracle)
	rate := "1.0" // Simplified

	toAmount := fromAmount // Would be calculated based on rate

	conversion := &Conversion{
		ConversionID: "conv_" + uuid.New().String()[:8],
		UserID:     userID,
		FromToken: fromToken,
		ToToken:   toToken,
		FromAmount: fromAmount,
		ToAmount:   toAmount,
		Rate:      rate,
		Status:    "COMPLETED",
		Chain:     chain,
		Timestamp: time.Now(),
	}

	s.conversions[conversion.ConversionID] = conversion
	return conversion, nil
}

// Get user accounts
func (s *DeFiService) GetUserAccounts(userID string) []*SavingsAccount {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*SavingsAccount, 0)
	for _, acc := range s.accounts {
		if acc.UserID == userID {
			result = append(result, acc)
		}
	}
	return result
}

// Get stats
func (s *DeFiService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_launchpads":   len(s.launchpads),
		"total_staking_pools": len(s.stakingPools),
		"total_earn_products": len(s.earnProducts),
		"total_etf_products": len(s.etfProducts),
		"total_accounts":    len(s.accounts),
	}
}

// Handlers
func (s *DeFiService) GetLaunchpadsHandler(c *gin.Context) {
	status := c.Query("status")
	launchpads := s.GetLaunchpads(status)
	c.JSON(http.StatusOK, gin.H{"launchpads": launchpads})
}

func (s *DeFiService) GetStakingPoolsHandler(c *gin.Context) {
	token := c.Query("token")
	chain := c.Query("chain")
	pools := s.GetStakingPools(token, chain)
	c.JSON(http.StatusOK, gin.H{"pools": pools})
}

func (s *DeFiService) GetEarnProductsHandler(c *gin.Context) {
	token := c.Query("token")
	productType := c.Query("type")
	products := s.GetEarnProducts(token, productType)
	c.JSON(http.StatusOK, gin.H{"products": products})
}

func (s *DeFiService) GetETFProductsHandler(c *gin.Context) {
	products := s.GetETFProducts()
	c.JSON(http.StatusOK, gin.H{"products": products})
}

func (s *DeFiService) ValidateCouponHandler(c *gin.Context) {
	code := c.Param("code")

	coupon, err := s.ValidateCoupon(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, coupon)
}

func (s *DeFiService) GetStatsHandler(c *gin.Context) {
	stats := s.GetStats()
	c.JSON(http.StatusOK, stats)
}

func (s *DeFiService) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/defi")
	{
		api.GET("/launchpads", s.GetLaunchpadsHandler)
		api.GET("/staking", s.GetStakingPoolsHandler)
		api.GET("/earn", s.GetEarnProductsHandler)
		api.GET("/etf", s.GetETFProductsHandler)
		api.GET("/coupons/:code", s.ValidateCouponHandler)
		api.GET("/stats", s.GetStatsHandler)
	}
}

func main() {
	cfg := Config{
		ServerPort: getEnv("DEFI_PORT", "8091"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
	}

	service := NewDeFiService(cfg)

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
			"service":   "defi-earn-service",
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
		log.Printf("Starting DeFi/Earn Service on %s", addr)
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
