// TigerWallet Staking Service - Comprehensive Staking Operations
// Supports liquid staking, lock staking, DeFi staking across multiple chains

package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port      int    `json:"port"`
	RedisAddr string `json:"redis_addr"`
	JWTSecret string `json:"jwt_secret"`
}

var cfg = Config{
	Port:      8003,
	RedisAddr: "localhost:6379",
	JWTSecret: getEnvOrDefault("JWT_SECRET", ""),
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ============================================================================
// Data Models
// ============================================================================

type StakingPool struct {
	ID              string         `json:"id" bson:"_id"`
	Name            string         `json:"name" bson:"name"`
	Chain           string         `json:"chain" bson:"chain"`
	Token           string         `json:"token" bson:"token"`
	RewardToken     string         `json:"reward_token" bson:"reward_token"`
	ContractAddress string         `json:"contract_address" bson:"contract_address"`
	TotalStaked    string         `json:"total_staked" bson:"total_staked"`
	RewardRate     string         `json:"reward_rate" bson:"reward_rate"`
	MinStake       string         `json:"min_stake" bson:"min_stake"`
	LockPeriod     int            `json:"lock_period" bson:"lock_period"` // seconds
	Status         string         `json:"status" bson:"status"` // active, paused, halted
	APY            string         `json:"apy" bson:"apy"`
	Delegators     int            `json:"delegators" bson:"delegators"`
	CreatedAt      time.Time      `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" bson:"updated_at"`
}

type StakingPosition struct {
	ID           string    `json:"id" bson:"_id"`
	UserID       string    `json:"user_id" bson:"user_id"`
	PoolID       string    `json:"pool_id" bson:"pool_id"`
	Chain        string    `json:"chain" bson:"chain"`
	Amount       string    `json:"amount" bson:"amount"`
	RewardPending string   `json:"reward_pending" bson:"reward_pending"`
	RewardClaimed string   `json:"reward_claimed" bson:"reward_claimed"`
	StakeTime    time.Time `json:"stake_time" bson:"stake_time"`
	UnlockTime   *time.Time `json:"unlock_time" bson:"unlock_time"`
	Status       string    `json:"status" bson:"status"` // staked, unstaking, claimed
}

type UnstakeRequest struct {
	ID          string    `json:"id" bson:"_id"`
	UserID      string    `json:"user_id" bson:"user_id"`
	PoolID      string    `json:"pool_id" bson:"pool_id"`
	PositionID  string    `json:"position_id" bson:"position_id"`
	Amount      string    `json:"amount" bson:"amount"`
	Status      string    `json:"status" bson:"status"` // pending, processing, completed, failed
	RequestTime time.Time `json:"request_time" bson:"request_time"`
	ProcessTime *time.Time `json:"process_time" bson:"process_time"`
	TxHash      string    `json:"tx_hash" bson:"tx_hash"`
}

type Validator struct {
	ID          string `json:"id" bson:"_id"`
	Name        string `json:"name" bson:"name"`
	Chain       string `json:"chain" bson:"chain"`
	Address     string `json:"address" bson:"address"`
	Commission  string `json:"commission" bson:"commission"`
	Uptime      string `json:"uptime" bson:"uptime"`
	Delegators  int    `json:"delegators" bson:"delegators"`
	TotalStake  string `json:"total_stake" bson:"total_stake"`
	RewardRate  string `json:"reward_rate" bson:"reward_rate"`
	Status      string `json:"status" bson:"status"`   // active, inactive, jailed
	Verified    bool   `json:"verified" bson:"verified"` // true only for on-chain verified validators
}

type LiquidStakingToken struct {
	ID          string `json:"id" bson:"_id"`
	Name        string `json:"name" bson:"name"`
	Symbol      string `json:"symbol" bson:"symbol"`
	Chain       string `json:"chain" bson:"chain"`
	StakedToken string `json:"staked_token" bson:"staked_token"`
	Contract    string `json:"contract" bson:"contract"`
	APY         string `json:"apy" bson:"apy"`
	TotalSupply string `json:"total_supply" bson:"total_supply"`
	PriceUSD    string `json:"price_usd" bson:"price_usd"`
	Status      string `json:"status" bson:"status"`
}

// ============================================================================
// Staking Service
// ============================================================================

type StakingService struct {
	redis          *redis.Client
	mu             sync.RWMutex
	pools          map[string]*StakingPool
	positions      map[string]*StakingPosition
	unstakeRequests map[string]*UnstakeRequest
	validators     map[string]*Validator
	liquidTokens   map[string]*LiquidStakingToken
}

func NewStakingService() *StakingService {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	ss := &StakingService{
		redis:           rdb,
		pools:           make(map[string]*StakingPool),
		positions:       make(map[string]*StakingPosition),
		unstakeRequests: make(map[string]*UnstakeRequest),
		validators:      make(map[string]*Validator),
		liquidTokens:   make(map[string]*LiquidStakingToken),
	}

	ss.initializeDefaultData()

	return ss
}

func (ss *StakingService) initializeDefaultData() {
	// Initialize staking pools
	pools := []StakingPool{
		{
			ID: "eth-staking", Name: "Ethereum Staking", Chain: "ethereum", Token: "ETH",
			RewardToken: "ETH", TotalStaked: "15000000", RewardRate: "0.04",
			MinStake: "0.01", LockPeriod: 1296000, Status: "active", APY: "4.5",
			Delegators: 50000,
		},
		{
			ID: "matic-staking", Name: "Polygon Staking", Chain: "polygon", Token: "MATIC",
			RewardToken: "MATIC", TotalStaked: "500000000", RewardRate: "0.08",
			MinStake: "1", LockPeriod: 604800, Status: "active", APY: "8.2",
			Delegators: 25000,
		},
		{
			ID: "sol-staking", Name: "Solana Staking", Chain: "solana", Token: "SOL",
			RewardToken: "SOL", TotalStaked: "100000000", RewardRate: "0.07",
			MinStake: "0.1", LockPeriod: 259200, Status: "active", APY: "7.1",
			Delegators: 35000,
		},
		{
			ID: "dot-staking", Name: "Polkadot Staking", Chain: "polkadot", Token: "DOT",
			RewardToken: "DOT", TotalStaked: "1000000000", RewardRate: "0.12",
			MinStake: "1", LockPeriod: 518400, Status: "active", APY: "12.5",
			Delegators: 15000,
		},
		{
			ID: "avax-staking", Name: "Avalanche Staking", Chain: "avalanche", Token: "AVAX",
			RewardToken: "AVAX", TotalStaked: "35000000", RewardRate: "0.08",
			MinStake: "1", LockPeriod: 1209600, Status: "active", APY: "8.5",
			Delegators: 12000,
		},
		{
			ID: "bnb-staking", Name: "BNB Staking", Chain: "bsc", Token: "BNB",
			RewardToken: "BNB", TotalStaked: "20000000", RewardRate: "0.10",
			MinStake: "0.1", LockPeriod: 604800, Status: "active", APY: "10.2",
			Delegators: 18000,
		},
		{
			ID: "ada-staking", Name: "Cardano Staking", Chain: "cardano", Token: "ADA",
			RewardToken: "ADA", TotalStaked: "25000000000", RewardRate: "0.05",
			MinStake: "10", LockPeriod: 518400, Status: "active", APY: "5.0",
			Delegators: 45000,
		},
		{
			ID: "atom-staking", Name: "Cosmos Staking", Chain: "cosmos", Token: "ATOM",
			RewardToken: "ATOM", TotalStaked: "150000000", RewardRate: "0.09",
			MinStake: "0.1", LockPeriod: 1814400, Status: "active", APY: "9.5",
			Delegators: 22000,
		},
	}

	for _, pool := range pools {
		pool.CreatedAt = time.Now()
		pool.UpdatedAt = time.Now()
		ss.pools[pool.ID] = &pool
	}

	// Initialize validators. These are SAMPLE entries with empty addresses
	// and Verified=false: they MUST NOT be used for real on-chain delegation.
	// Production deployments must load verified validator addresses from an
	// on-chain validator registry (e.g. the deposit contract / staking manager).
	validators := []Validator{
		{ID: "val-1", Name: "Sample Ethereum Validator", Chain: "ethereum", Address: "", Commission: "5%", Uptime: "99.9%", Delegators: 0, TotalStake: "0", RewardRate: "4.2%", Status: "sample", Verified: false},
		{ID: "val-2", Name: "Sample Ethereum Validator 2", Chain: "ethereum", Address: "", Commission: "4%", Uptime: "99.95%", Delegators: 0, TotalStake: "0", RewardRate: "4.3%", Status: "sample", Verified: false},
		{ID: "val-3", Name: "Sample Solana Validator", Chain: "solana", Address: "", Commission: "6%", Uptime: "99.8%", Delegators: 0, TotalStake: "0", RewardRate: "6.5%", Status: "sample", Verified: false},
	}

	for _, val := range validators {
		ss.validators[val.ID] = &val
	}

	// Initialize liquid staking tokens
	liquidTokens := []LiquidStakingToken{
		{ID: "steth", Name: "Liquid Staked ETH", Symbol: "stETH", Chain: "ethereum", StakedToken: "ETH", APY: "4.2%", TotalSupply: "5000000", PriceUSD: "1.042", Status: "active"},
		{ID: "wmatic", Name: "Liquid Matic", Symbol: "wMATIC", Chain: "polygon", StakedToken: "MATIC", APY: "7.8%", TotalSupply: "100000000", PriceUSD: "1.08", Status: "active"},
		{ID: "msol", Name: "Marinade stSOL", Symbol: "mSOL", Chain: "solana", StakedToken: "SOL", APY: "6.5%", TotalSupply: "25000000", PriceUSD: "1.065", Status: "active"},
		{ID: "dotl", Name: "Liquid DOT", Symbol: "lDOT", Chain: "polkadot", StakedToken: "DOT", APY: "11.5%", TotalSupply: "100000000", PriceUSD: "1.12", Status: "active"},
	}

	for _, token := range liquidTokens {
		ss.liquidTokens[token.ID] = &token
	}
}

// ============================================================================
// API Handlers
// ============================================================================

// Get all staking pools
func (ss *StakingService) GetPools(c *gin.Context) {
	pools := make([]*StakingPool, 0)
	for _, pool := range ss.pools {
		pools = append(pools, pool)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"pools":   pools,
		"total":   len(pools),
	})
}

// Get pool by ID
func (ss *StakingService) GetPool(c *gin.Context) {
	poolID := c.Param("id")

	pool, exists := ss.pools[poolID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "pool not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"pool":    pool,
	})
}

// Stake tokens
type StakeRequest struct {
	UserID string `json:"user_id" binding:"required"`
	PoolID string `json:"pool_id" binding:"required"`
	Amount string `json:"amount" binding:"required"`
}

func (ss *StakingService) Stake(c *gin.Context) {
	var req StakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate pool
	pool, exists := ss.pools[req.PoolID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "pool not found"})
		return
	}

	if pool.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pool is not active"})
		return
	}

	// Validate amount
	minStake := new(big.Float)
	minStake.SetString(pool.MinStake)
	amount := new(big.Float)
	amount.SetString(req.Amount)
	if amount.Cmp(minStake) < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount below minimum stake"})
		return
	}

	// Create position
	positionID := uuid.New().String()
	now := time.Now()
	unlockTime := now.Add(time.Duration(pool.LockPeriod) * time.Second)

	position := &StakingPosition{
		ID:            positionID,
		UserID:        req.UserID,
		PoolID:        req.PoolID,
		Chain:         pool.Chain,
		Amount:        req.Amount,
		RewardPending: "0",
		RewardClaimed: "0",
		StakeTime:     now,
		UnlockTime:    &unlockTime,
		Status:        "staked",
	}

	ss.positions[positionID] = position

	// Update pool stats
	pool.TotalStaked = addStrings(pool.TotalStaked, req.Amount)
	pool.Delegators++

	// Log in Redis
	ctx := context.Background()
	ss.redis.Set(ctx, "staking:position:"+positionID, positionID, 0)

	c.JSON(http.StatusCreated, gin.H{
		"success":     true,
		"position_id": positionID,
		"pool_id":     req.PoolID,
		"amount":      req.Amount,
		"stake_time":  now.Unix(),
		"unlock_time": unlockTime.Unix(),
		"status":      "staked",
	})
}

// Unstake tokens
type UnstakeRequestInput struct {
	UserID     string `json:"user_id" binding:"required"`
	PoolID     string `json:"pool_id" binding:"required"`
	PositionID string `json:"position_id" binding:"required"`
	Amount     string `json:"amount" binding:"required"`
}

func (ss *StakingService) Unstake(c *gin.Context) {
	var req UnstakeRequestInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate position
	position, exists := ss.positions[req.PositionID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}

	if position.UserID != req.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	if position.Status != "staked" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "position not active"})
		return
	}

	// Check lock period
	now := time.Now()
	if now.Before(*position.UnlockTime) {
		// Still locked
		lockRemaining := position.UnlockTime.Sub(now)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "tokens still locked",
			"unlock_time":    position.UnlockTime.Unix(),
			"remaining_secs":  int(lockRemaining.Seconds()),
		})
		return
	}

	// Create unstake request
	unstakeID := uuid.New().String()
	unstake := &UnstakeRequest{
		ID:          unstakeID,
		UserID:      req.UserID,
		PoolID:      req.PoolID,
		PositionID:  req.PositionID,
		Amount:      req.Amount,
		Status:      "pending",
		RequestTime: now,
	}

	ss.unstakeRequests[unstakeID] = unstake

	// Update position
	position.Status = "unstaking"

	// Update pool
	if pool, ok := ss.pools[req.PoolID]; ok {
		pool.TotalStaked = subtractStrings(pool.TotalStaked, req.Amount)
		pool.Delegators--
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success":       true,
		"unstake_id":   unstakeID,
		"position_id":  req.PositionID,
		"amount":       req.Amount,
		"status":       "pending",
		"request_time": now.Unix(),
	})
}

// Claim rewards
type ClaimRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	PositionID string `json:"position_id" binding:"required"`
}

func (ss *StakingService) Claim(c *gin.Context) {
	var req ClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate position
	position, exists := ss.positions[req.PositionID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}

	if position.UserID != req.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	// Calculate rewards (simplified)
	pool := ss.pools[position.PoolID]
	rewardRate := new(big.Float)
	rewardRate.SetString(pool.RewardRate)
	amount := new(big.Float)
	amount.SetString(position.Amount)
	stakingDuration := time.Since(position.StakeTime).Seconds()
	days := stakingDuration / 86400

	reward := new(big.Float)
	reward.Mul(amount, rewardRate)
	reward.Mul(reward, big.NewFloat(days/365))

	rewardStr := fmt.Sprintf("%.6f", reward)

	// Update position
	position.RewardPending = "0"
	position.RewardClaimed = addStrings(position.RewardClaimed, rewardStr)

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"position_id":  req.PositionID,
		"claimed":      rewardStr,
		"reward_token": pool.RewardToken,
		"tx_hash":      "",
		"status":       "pending",
	})
}

// Get user positions
func (ss *StakingService) GetUserPositions(c *gin.Context) {
	userID := c.Param("user_id")

	positions := make([]*StakingPosition, 0)
	for _, pos := range ss.positions {
		if pos.UserID == userID {
			positions = append(positions, pos)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"positions": positions,
		"total":     len(positions),
	})
}

// Get validators
func (ss *StakingService) GetValidators(c *gin.Context) {
	chain := c.Query("chain")

	validators := make([]*Validator, 0)
	for _, val := range ss.validators {
		if chain == "" || val.Chain == chain {
			validators = append(validators, val)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"validators": validators,
		"total":     len(validators),
	})
}

// Delegate to validator
type DelegateRequest struct {
	UserID      string `json:"user_id" binding:"required"`
	ValidatorID string `json:"validator_id" binding:"required"`
	Amount      string `json:"amount" binding:"required"`
	Chain       string `json:"chain" binding:"required"`
}

func (ss *StakingService) Delegate(c *gin.Context) {
	var req DelegateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate validator
	validator, exists := ss.validators[req.ValidatorID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "validator not found"})
		return
	}

	// Create delegation position
	delegationID := uuid.New().String()
	position := &StakingPosition{
		ID:            delegationID,
		UserID:        req.UserID,
		PoolID:        "delegate-" + req.ValidatorID,
		Chain:         req.Chain,
		Amount:        req.Amount,
		RewardPending: "0",
		RewardClaimed: "0",
		StakeTime:     time.Now(),
		Status:        "staked",
	}

	ss.positions[delegationID] = position

	// Update validator
	validator.Delegators++
	validator.TotalStake = addStrings(validator.TotalStake, req.Amount)

	c.JSON(http.StatusCreated, gin.H{
		"success":        true,
		"delegation_id": delegationID,
		"validator":     validator.Name,
		"amount":        req.Amount,
		"chain":         req.Chain,
	})
}

// Get liquid staking tokens
func (ss *StakingService) GetLiquidTokens(c *gin.Context) {
	tokens := make([]*LiquidStakingToken, 0)
	for _, token := range ss.liquidTokens {
		tokens = append(tokens, token)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tokens":  tokens,
		"total":   len(tokens),
	})
}

// Convert (wrap) to liquid staking token
type ConvertRequest struct {
	UserID    string `json:"user_id" binding:"required"`
	TokenID   string `json:"token_id" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
	Direction string `json:"direction" binding:"required"` // stake or unstake
}

func (ss *StakingService) Convert(c *gin.Context) {
	var req ConvertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate token
	token, exists := ss.liquidTokens[req.TokenID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "liquid token not found"})
		return
	}

	convertID := uuid.New().String()

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"convert_id":    convertID,
		"token":        token.Symbol,
		"amount":       req.Amount,
		"output_amount": req.Amount, // 1:1 for simplicity
		"tx_hash":      "",
		"status":       "pending",
	})
}

// Get staking rewards
func (ss *StakingService) GetRewards(c *gin.Context) {
	userID := c.Param("user_id")

	totalPending := "0"
	totalClaimed := "0"

	for _, pos := range ss.positions {
		if pos.UserID == userID {
			totalPending = addStrings(totalPending, pos.RewardPending)
			totalClaimed = addStrings(totalClaimed, pos.RewardClaimed)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"reward_pending": totalPending,
		"reward_claimed": totalClaimed,
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func addStrings(a, b string) string {
	af := new(big.Float)
	af.SetString(a)
	bf := new(big.Float)
	bf.SetString(b)
	cf := new(big.Float)
	cf.Add(af, bf)
	return cf.String()
}

func subtractStrings(a, b string) string {
	af := new(big.Float)
	af.SetString(a)
	bf := new(big.Float)
	bf.SetString(b)
	cf := new(big.Float)
	cf.Sub(af, bf)
	return cf.String()
}

// ============================================================================
// Middleware
// ============================================================================

func (ss *StakingService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		if cfg.JWTSecret == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "JWT_SECRET not configured"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			c.Abort()
			return
		}
		uid, _ := claims["user_id"].(string)
		if uid == "" {
			uid, _ = claims["sub"].(string)
		}
		if uid == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user_id in token"})
			c.Abort()
			return
		}
		c.Set("user_id", uid)
		c.Next()
	}
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.Println("TigerWallet Staking Service")
	log.Println("===========================")
	log.Printf("Starting on port %d", cfg.Port)

	ss := NewStakingService()

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

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "staking-service",
			"timestamp": time.Now().Unix(),
		})
	})

	// Public routes
	r.GET("/api/v1/staking/pools", ss.GetPools)
	r.GET("/api/v1/staking/pools/:id", ss.GetPool)
	r.GET("/api/v1/staking/validators", ss.GetValidators)
	r.GET("/api/v1/staking/liquid-tokens", ss.GetLiquidTokens)

	// Protected routes
	api := r.Group("/api/v1/staking")
	api.Use(ss.AuthMiddleware())
	{
		// Staking
		api.POST("/stake", ss.Stake)
		api.POST("/unstake", ss.Unstake)
		api.POST("/claim", ss.Claim)

		// Positions
		api.GET("/users/:user_id/positions", ss.GetUserPositions)
		api.GET("/users/:user_id/rewards", ss.GetRewards)

		// Delegation
		api.POST("/delegate", ss.Delegate)

		// Liquid staking
		api.POST("/convert", ss.Convert)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
