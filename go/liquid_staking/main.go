package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port     string
	RedisURL string
}

func LoadConfig() *Config {
	return &Config{
		Port:     getEnv("PORT", "8448"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Models
// ============================================================================

type Validator struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Commission   float64 `json:"commission"`
	StakedAmount string  `json:"stakedAmount"`
	Delegators   int     `json:"delegators"`
	Uptime       float64 `json:"uptime"`
	APY          float64 `json:"apy"`
	Active       bool    `json:"active"`
	Slashed      bool    `json:"slashed"`
}

type StakePosition struct {
	UserID         string `json:"userId"`
	StakerToken    string `json:"stakerToken"`    // LST user receives
	StakedAmount   string `json:"stakedAmount"`   // Amount staked
	StakerTokenAmt string `json:"stakerTokenAmt"` // LST minted
	StakeTime      int64  `json:"stakeTime"`
	ChainID        uint64 `json:"chainId"`
	ValidatorID    string `json:"validatorId"`
	RewardsEarned  string `json:"rewardsEarned"`
	Status         string `json:"status"` // active, unstaking, claimed
}

type UnstakeRequest struct {
	UserID       string `json:"userId"`
	PositionID   string `json:"positionId"`
	Amount       string `json:"amount"`
	RequestTime  int64  `json:"requestTime"`
	CompleteTime int64  `json:"completeTime"`
	Status       string `json:"status"` // pending, ready, completed
}

type PoolStats struct {
	ChainID         uint64  `json:"chainId"`
	TotalStaked     string  `json:"totalStaked"`
	TotalLSTMinted  string  `json:"totalLstMinted"`
	CurrentAPY      float64 `json:"currentApy"`
	RewardRate      float64 `json:"rewardRate"`
	UnstakingQueue  int     `json:"unstakingQueue"`
	TotalValidators int     `json:"totalValidators"`
	TotalDelegators int     `json:"totalDelegators"`
}

type StakeRequest struct {
	UserID      string `json:"userId" binding:"required"`
	ChainID     uint64 `json:"chainId" binding:"required"`
	Amount      string `json:"amount" binding:"required"`
	ValidatorID string `json:"validatorId"`
}

type UnstakeRequestInput struct {
	UserID     string `json:"userId" binding:"required"`
	PositionID string `json:"positionId" binding:"required"`
	Amount     string `json:"amount" binding:"required"`
}

type LiquidStakingService struct {
	config     *Config
	redis      *redis.Client
	validators map[string][]Validator
	positions  map[string][]StakePosition
	unstakes   map[string][]UnstakeRequest
	poolStats  map[uint64]*PoolStats
	mu         sync.RWMutex
}

func NewLiquidStakingService(config *Config) *LiquidStakingService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	// Initialize validators
	validators := map[string][]Validator{
		"1": {
			{ID: "v1", Name: "Lido", Commission: 10, StakedAmount: "500000000000000000000000", Delegators: 10000, Uptime: 99.9, APY: 4.2, Active: true},
			{ID: "v2", Name: "Rocket Pool", Commission: 15, StakedAmount: "200000000000000000000000", Delegators: 5000, Uptime: 99.8, APY: 4.5, Active: true},
			{ID: "v3", Name: "Frax Ether", Commission: 10, StakedAmount: "150000000000000000000000", Delegators: 3000, Uptime: 99.9, APY: 4.1, Active: true},
		},
		"137": {
			{ID: "m1", Name: "Polygon Stake", Commission: 5, StakedAmount: "1000000000000000000000000", Delegators: 20000, Uptime: 99.95, APY: 5.2, Active: true},
		},
	}

	// Initialize pool stats
	poolStats := map[uint64]*PoolStats{
		1: {
			ChainID:         1,
			TotalStaked:     "850000000000000000000000",
			TotalLSTMinted:  "812500000000000000000000",
			CurrentAPY:      4.2,
			RewardRate:      0.000115,
			UnstakingQueue:  0,
			TotalValidators: 3,
			TotalDelegators: 18000,
		},
		137: {
			ChainID:         137,
			TotalStaked:     "1000000000000000000000000",
			TotalLSTMinted:  "950000000000000000000000",
			CurrentAPY:      5.2,
			RewardRate:      0.000142,
			UnstakingQueue:  0,
			TotalValidators: 1,
			TotalDelegators: 20000,
		},
	}

	return &LiquidStakingService{
		config:     config,
		redis:      redisClient,
		validators: validators,
		positions:  make(map[string][]StakePosition),
		unstakes:   make(map[string][]UnstakeRequest),
		poolStats:  poolStats,
	}
}

// ============================================================================
// Staking Operations
// ============================================================================

func (s *LiquidStakingService) Stake(req StakeRequest) (*StakePosition, error) {
	// Validate amount
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok || amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid stake amount")
	}

	// Get validator
	chainStr := strconv.FormatUint(req.ChainID, 10)
	chainValidators := s.validators[chainStr]

	var validator Validator
	if req.ValidatorID != "" {
		for _, v := range chainValidators {
			if v.ID == req.ValidatorID {
				validator = v
				break
			}
		}
	} else {
		// Use best validator (lowest commission, highest uptime)
		for _, v := range chainValidators {
			if v.Active && !v.Slashed && v.Uptime > 99.5 {
				validator = v
				break
			}
		}
	}

	// Calculate LST amount (1:1 for now, would use exchange rate)
	lstAmount := amount.String()

	// Create stake position
	position := &StakePosition{
		UserID:         req.UserID,
		StakerToken:    fmt.Sprintf("tigerLST-%s", chainStr),
		StakedAmount:   req.Amount,
		StakerTokenAmt: lstAmount,
		StakeTime:      time.Now().Unix(),
		ChainID:        req.ChainID,
		ValidatorID:    validator.ID,
		RewardsEarned:  "0",
		Status:         "active",
	}

	// Store position
	s.mu.Lock()
	s.positions[req.UserID] = append(s.positions[req.UserID], *position)

	// Update pool stats
	if stats, ok := s.poolStats[req.ChainID]; ok {
		currentStaked, _ := new(big.Int).SetString(stats.TotalStaked, 10)
		newStaked := new(big.Int).Add(currentStaked, amount)
		stats.TotalStaked = newStaked.String()

		currentMinted, _ := new(big.Int).SetString(stats.TotalLSTMinted, 10)
		newMinted := new(big.Int).Add(currentMinted, amount)
		stats.TotalLSTMinted = newMinted.String()

		stats.TotalDelegators++
	}
	s.mu.Unlock()

	return position, nil
}

func (s *LiquidStakingService) Unstake(req UnstakeRequestInput) (*UnstakeRequest, error) {
	// Find position
	s.mu.RLock()
	userPositions := s.positions[req.UserID]
	s.mu.RUnlock()

	var position *StakePosition
	for i, p := range userPositions {
		if p.StakerTokenAmt == req.PositionID || fmt.Sprintf("pos-%d", i) == req.PositionID {
			position = &p
			break
		}
	}

	if position == nil {
		return nil, fmt.Errorf("position not found")
	}

	// Validate amount
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok || amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid unstake amount")
	}

	// Create unstake request
	unstake := &UnstakeRequest{
		UserID:       req.UserID,
		PositionID:   req.PositionID,
		Amount:       req.Amount,
		RequestTime:  time.Now().Unix(),
		CompleteTime: time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 day unstaking period
		Status:       "pending",
	}

	// Store unstake
	s.mu.Lock()
	s.unstakes[req.UserID] = append(s.unstakes[req.UserID], *unstake)

	// Update pool stats
	if stats, ok := s.poolStats[position.ChainID]; ok {
		stats.UnstakingQueue++
	}
	s.mu.Unlock()

	return unstake, nil
}

func (s *LiquidStakingService) ClaimUnstake(userID, unstakeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userUnstakes := s.unstakes[userID]

	for i, unstake := range userUnstakes {
		if fmt.Sprintf("unstake-%d", i) == unstakeID || unstake.PositionID == unstakeID {
			if unstake.Status != "ready" {
				return fmt.Errorf("unstake not ready")
			}

			userUnstakes[i].Status = "completed"

			// Update pool stats
			if stats, ok := s.poolStats[1]; ok {
				stats.UnstakingQueue--
				currentStaked, _ := new(big.Int).SetString(stats.TotalStaked, 10)
				amount, _ := new(big.Int).SetString(unstake.Amount, 10)
				newStaked := new(big.Int).Sub(currentStaked, amount)
				stats.TotalStaked = newStaked.String()
			}

			return nil
		}
	}

	return fmt.Errorf("unstake not found")
}

func (s *LiquidStakingService) GetUserPositions(userID string) []StakePosition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.positions[userID]
}

func (s *LiquidStakingService) GetPoolStats(chainID uint64) *PoolStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.poolStats[chainID]
}

func (s *LiquidStakingService) GetValidators(chainID uint64) []Validator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	chainStr := strconv.FormatUint(chainID, 10)
	return s.validators[chainStr]
}

func (s *LiquidStakingService) CalculateRewards(userID string) (string, error) {
	s.mu.RLock()
	positions := s.positions[userID]
	stats := s.poolStats
	s.mu.RUnlock()

	totalRewards := big.NewInt(0)

	for _, pos := range positions {
		if pos.Status != "active" {
			continue
		}

		staked, _ := new(big.Int).SetString(pos.StakedAmount, 10)
		if staked == nil {
			continue
		}

		chainStats := stats[pos.ChainID]
		if chainStats == nil {
			continue
		}

		// Calculate rewards: staked * reward_rate * time
		// Simplified calculation
		rewardRate := big.NewFloat(chainStats.RewardRate)
		stakedFloat := new(big.Float).SetInt(staked)
		rewards := new(big.Float).Mul(rewardRate, stakedFloat)

		rewardsInt, _ := rewards.Int(nil)
		totalRewards.Add(totalRewards, rewardsInt)
	}

	return totalRewards.String(), nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *LiquidStakingService) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "liquid-staking"})
	})

	api := r.Group("/api/v1/staking")
	{
		// Pool stats
		api.GET("/pool/:chainId", s.handleGetPoolStats)

		// Validators
		api.GET("/validators/:chainId", s.handleGetValidators)

		// Stake
		api.POST("/stake", s.handleStake)

		// Unstake
		api.POST("/unstake", s.handleUnstake)

		// Claim
		api.POST("/claim", s.handleClaim)

		// User positions
		api.GET("/positions/:userId", s.handleGetPositions)

		// Rewards
		api.GET("/rewards/:userId", s.handleGetRewards)

		// LST price
		api.GET("/lst/:chainId", s.handleGetLSTPrice)
	}
}

func (s *LiquidStakingService) handleGetPoolStats(c *gin.Context) {
	chainID, err := strconv.ParseUint(c.Param("chainId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain ID"})
		return
	}

	stats := s.GetPoolStats(chainID)
	if stats == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chain not supported"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (s *LiquidStakingService) handleGetValidators(c *gin.Context) {
	chainID, err := strconv.ParseUint(c.Param("chainId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain ID"})
		return
	}

	validators := s.GetValidators(chainID)
	c.JSON(http.StatusOK, gin.H{"validators": validators})
}

func (s *LiquidStakingService) handleStake(c *gin.Context) {
	var req StakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	position, err := s.Stake(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, position)
}

func (s *LiquidStakingService) handleUnstake(c *gin.Context) {
	var req UnstakeRequestInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	unstake, err := s.Unstake(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, unstake)
}

func (s *LiquidStakingService) handleClaim(c *gin.Context) {
	var req struct {
		UserID    string `json:"userId" binding:"required"`
		UnstakeID string `json:"unstakeId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.ClaimUnstake(req.UserID, req.UnstakeID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Claimed successfully"})
}

func (s *LiquidStakingService) handleGetPositions(c *gin.Context) {
	userID := c.Param("userId")
	positions := s.GetUserPositions(userID)

	c.JSON(http.StatusOK, gin.H{"positions": positions})
}

func (s *LiquidStakingService) handleGetRewards(c *gin.Context) {
	userID := c.Param("userId")
	rewards, err := s.CalculateRewards(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rewards": rewards})
}

func (s *LiquidStakingService) handleGetLSTPrice(c *gin.Context) {
	chainID, err := strconv.ParseUint(c.Param("chainId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain ID"})
		return
	}

	stats := s.GetPoolStats(chainID)
	if stats == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chain not supported"})
		return
	}

	// Calculate LST price (Total Staked / Total LST Minted)
	staked, _ := new(big.Int).SetString(stats.TotalStaked, 10)
	minted, _ := new(big.Int).SetString(stats.TotalLSTMinted, 10)

	if minted.Sign() == 0 {
		c.JSON(http.StatusOK, gin.H{"price": "1.0", "symbol": "tigerLST"})
		return
	}

	price := new(big.Float).Quo(new(big.Float).SetInt(staked), new(big.Float).SetInt(minted))
	priceStr, _ := price.Float64()

	c.JSON(http.StatusOK, gin.H{
		"price":  strconv.FormatFloat(priceStr, 'f', 4, 64),
		"symbol": fmt.Sprintf("tigerLST-%d", chainID),
		"apy":    stats.CurrentAPY,
	})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()
	service := NewLiquidStakingService(config)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	service.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Liquid Staking service starting on port %s", config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

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
