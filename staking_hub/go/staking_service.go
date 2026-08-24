// Staking Dashboard Service - Go Implementation
// Stake, unstake, and rewards tracking across chains

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Configuration
type StakingConfig struct {
	ServerPort  string `json:"server_port"`
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	RedisHost  string `json:"redis_host"`
	RedisPort  string `json:"redis_port"`
}

// Validator represents a staking validator
type Validator struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	ValidatorAddr  string `gorm:"uniqueIndex" json:"validator_addr"`
	Name          string `json:"name"`
	ChainID       int64  `json:"chain_id"`
	Commission    int64  `json:"commission"`
	MinStake      string `json:"min_stake"`
	MaxStake      string `json:"max_stake"`
	APY           float64 `json:"apy"`
	TotalStake    string `json:"total_stake"`
	Delegators    int64   `json:"delegators"`
	IsActive     bool   `json:"is_active"`
	Jailed        bool   `json:"jailed"`
}

// StakePosition represents a user's stake
type StakePosition struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	PositionID     string    `gorm:"uniqueIndex" json:"position_id"`
	UserAddress    string    `gorm:"index" json:"user_address"`
	ValidatorAddr  string    `json:"validator_addr"`
	ChainID        int64     `json:"chain_id"`
	Token         string    `json:"token"`
	StakedAmount  string    `json:"staked_amount"`
	RewardsClaimed string   `json:"rewards_claimed"`
	RewardsPending string   `json:"rewards_pending"`
	LockPeriod     int64    `json:"lock_period"`
	StartTime      time.Time `json:"start_time"`
	UnlockTime     time.Time `json:"unlock_time"`
	Status        string   `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Staking Stats
type StakingStats struct {
	TotalStaked     string  `json:"total_staked"`
	TotalRewards    string  `json:"total_rewards"`
	ActivePositions int64   `json:"active_positions"`
	APY             float64 `json:"apy"`
}

// ClaimRecord represents a claim transaction
type ClaimRecord struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ClaimID      string    `gorm:"uniqueIndex" json:"claim_id"`
	PositionID   string    `json:"position_id"`
	UserAddr    string    `json:"user_address"`
	Amount      string    `json:"amount"`
	ChainID     int64     `json:"chain_id"`
	TxHash      string    `json:"tx_hash"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Staking Service
type StakingService struct {
	db      *gorm.DB
	redis  *redis.Client
	config StakingConfig
}

// NewStakingService creates new service
func NewStakingService(cfg StakingConfig) (*StakingService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Validator{}, &StakePosition{}, &ClaimRecord{})
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

	return &StakingService{
		db:      db,
		redis:  rdb,
		config: cfg,
	}, nil
}

// GeneratePositionID generates a unique position ID
func (s *StakingService) GeneratePositionID() string {
	var b [16]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Stake stakes tokens
func (s *StakingService) Stake(req StakeRequest) (*StakePosition, error) {
	positionID := s.GeneratePositionID()
	rewards := calculateRewards(req.Amount, req.APY)

	position := &StakePosition{
		PositionID:     positionID,
		UserAddress:    req.UserAddress,
		ValidatorAddr: req.ValidatorAddr,
		ChainID:       req.ChainID,
		Token:        req.Token,
		StakedAmount: req.Amount,
		RewardsClaimed: "0",
		RewardsPending: rewards,
		LockPeriod:    req.LockPeriod,
		StartTime:    time.Now(),
		UnlockTime:   time.Now().Add(time.Duration(req.LockPeriod) * 24 * time.Hour),
		Status:       "staking",
		CreatedAt:    time.Now(),
	}

	s.db.Create(position)
	return position, nil
}

// Unstake initiates unstaking
func (s *StakingService) Unstake(positionID string) error {
	result := s.db.Model(&StakePosition{}).Where("position_id = ? AND status = ?", positionID, "staking").
		Updates(map[string]interface{}{
			"status":      "unstaking",
			"unlock_time": time.Now().Add(24 * time.Hour),
			"updated_at": time.Now(),
		})

	if result.RowsAffected == 0 {
		return fmt.Errorf("position not found or cannot unstake")
	}
	return nil
}

// Withdraw withdraws staked tokens
func (s *StakingService) Withdraw(positionID string, txHash string) error {
	result := s.db.Model(&StakePosition{}).Where("position_id = ? AND status = ?", positionID, "unstaking").
		Updates(map[string]interface{}{
			"status":     "withdrawn",
			"updated_at": time.Now(),
		})

	if result.RowsAffected == 0 {
		return fmt.Errorf("position not found or not ready")
	}
	return nil
}

// ClaimRewards claims pending rewards
func (s *StakingService) ClaimRewards(positionID string) (*ClaimRecord, error) {
	position, err := s.GetPosition(positionID)
	if err != nil {
		return nil, err
	}

	if position.RewardsPending == "0" {
		return nil, fmt.Errorf("no rewards to claim")
	}

	claimID := s.GeneratePositionID()
	txHash := "0x" + generateHash()

	claim := &ClaimRecord{
		ClaimID:    claimID,
		PositionID: positionID,
		UserAddr:  position.UserAddress,
		Amount:   position.RewardsPending,
		ChainID:  position.ChainID,
		TxHash:  txHash,
		Status:  "pending",
	}

	s.db.Create(claim)
	position.RewardsClaimed = addStrings(position.RewardsClaimed, position.RewardsPending)
	position.RewardsPending = "0"
	s.db.Save(position)

	return claim, nil
}

// GetPosition gets a stake position
func (s *StakingService) GetPosition(positionID string) (*StakePosition, error) {
	var position StakePosition
	if err := s.db.Where("position_id = ?", positionID).First(&position).Error; err != nil {
		return nil, err
	}
	return &position, nil
}

// GetUserPositions gets all positions for a user
func (s *StakingService) GetUserPositions(userAddress string) ([]StakePosition, error) {
	var positions []StakePosition
	err := s.db.Where("user_address = ?", userAddress).Find(&positions).Error
	return positions, err
}

// GetValidators gets active validators
func (s *StakingService) GetValidators(chainID int64) ([]Validator, error) {
	var validators []Validator
	err := s.db.Where("chain_id = ? AND is_active = ? AND jailed = ?", chainID, true, false).Find(&validators).Error
	return validators, err
}

// GetUserStats gets staking stats for a user
func (s *StakingService) GetUserStats(userAddress string) (*StakingStats, error) {
	var positions []StakePosition
	s.db.Where("user_address = ?", userAddress).Find(&positions)

	totalStaked := "0"
	totalRewards := "0"
	activeCount := int64(0)

	for _, p := range positions {
		if p.Status == "staking" || p.Status == "unstaking" {
			totalStaked = addStrings(totalStaked, p.StakedAmount)
			totalRewards = addStrings(totalRewards, p.RewardsClaimed)
			totalRewards = addStrings(totalRewards, p.RewardsPending)
			activeCount++
		}
	}

	return &StakingStats{
		TotalStaked:     totalStaked,
		TotalRewards:    totalRewards,
		ActivePositions: activeCount,
		APY:             5.0,
	}, nil
}

// Request types

type StakeRequest struct {
	UserAddress    string  `json:"user_address" binding:"required"`
	ValidatorAddr string  `json:"validator_addr" binding:"required"`
	ChainID       int64   `json:"chain_id" binding:"required"`
	Token        string  `json:"token" binding:"required"`
	Amount       string  `json:"amount" binding:"required"`
	LockPeriod    int64   `json:"lock_period"`
	APY          float64 `json:"apy"`
}

// Handlers

func (s *StakingService) StakeHandler(c *gin.Context) {
	var req StakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	position, err := s.Stake(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, position)
}

func (s *StakingService) UnstakeHandler(c *gin.Context) {
	positionID := c.Param("position_id")
	if err := s.Unstake(positionID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "unstaking"})
}

func (s *StakingService) WithdrawHandler(c *gin.Context) {
	positionID := c.Param("position_id")
	txHash := c.PostForm("tx_hash")
	if err := s.Withdraw(positionID, txHash); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "withdrawn"})
}

func (s *StakingService) ClaimRewardsHandler(c *gin.Context) {
	positionID := c.Param("position_id")
	claim, err := s.ClaimRewards(positionID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, claim)
}

func (s *StakingService) GetPositionHandler(c *gin.Context) {
	positionID := c.Param("position_id")
	position, err := s.GetPosition(positionID)
	if err != nil {
		c.JSON(404, gin.H{"error": "position not found"})
		return
	}
	c.JSON(200, position)
}

func (s *StakingService) GetUserPositionsHandler(c *gin.Context) {
	address := c.Param("address")
	positions, err := s.GetUserPositions(address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, positions)
}

func (s *StakingService) GetValidatorsHandler(c *gin.Context) {
	chainID := parseInt64(c.Query("chain_id"))
	validators, err := s.GetValidators(chainID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, validators)
}

func (s *StakingService) GetUserStatsHandler(c *gin.Context) {
	address := c.Param("address")
	stats, err := s.GetUserStats(address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, stats)
}

// Utility functions

func calculateRewards(amount string, apy float64) string {
	amountFloat := parseAmount(amount)
	rewards := amountFloat * (apy / 100) / 365
	return formatAmount(rewards)
}

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

func addStrings(a, b string) string {
	af := parseAmount(a)
	bf := parseAmount(b)
	return formatAmount(af + bf)
}

func generateHash() string {
	var b [32]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Main

func main() {
	cfg := StakingConfig{
		ServerPort: getEnv("STAKING_SERVER_PORT", "8084"),
		DBHost:    getEnv("DB_HOST", "localhost"),
		DBPort:    getEnv("DB_PORT", "5432"),
		DBUser:    getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:    getEnv("DB_NAME", "staking_db"),
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewStakingService(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize service: %v\n", err)
		os.Exit(1)
	}

	r := gin.Default()

	r.POST("/stake", service.StakeHandler)
	r.POST("/stake/:position_id/unstake", service.UnstakeHandler)
	r.POST("/stake/:position_id/withdraw", service.WithdrawHandler)
	r.POST("/stake/:position_id/claim", service.ClaimRewardsHandler)
	r.GET("/stake/:position_id", service.GetPositionHandler)
	r.GET("/stake/user/:address", service.GetUserPositionsHandler)
	r.GET("/validators", service.GetValidatorsHandler)
	r.GET("/stats/:address", service.GetUserStatsHandler)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	go func() {
		fmt.Printf("Staking Service starting on port %s\n", cfg.ServerPort)
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