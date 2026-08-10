// TigerWallet DeFi Service - Staking, Launchpad, Earn Products
// Production-ready DeFi functionality

package main

import (
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort string `json:"server_port"`
	DBHost    string `json:"db_host"`
	DBPort    string `json:"db_port"`
	DBUser    string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName    string `json:"db_name"`
	RedisHost  string `json:"redis_host"`
	RedisPort  string `json:"redis_port"`
}

func LoadConfig() *Config {
	return &Config{
		ServerPort: getEnv("DEFI_PORT", "9097"),
		DBHost:    getEnv("DB_HOST", "localhost"),
		DBPort:    getEnv("DB_PORT", "5432"),
		DBUser:    getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:    getEnv("DB_NAME", "tigerwallet_defi"),
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Database Models
// ============================================================================

// StakingPool represents a staking pool
type StakingPool struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	PoolID         string    `gorm:"uniqueIndex" json:"pool_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	TokenAddress   string    `json:"token_address"`
	ChainID        int64     `json:"chain_id"`
	RewardToken    string    `json:"reward_token"`
	APY            float64   `json:"apy"`
	MinStake       string    `json:"min_stake"`
	MaxStake       string    `json:"max_stake"`
	LockPeriod     int       `json:"lock_period"` // seconds
	TotalStaked    string    `json:"total_staked"`
	TotalRewards   string    `json:"total_rewards"`
	Status         string    `json:"status"` // active, paused, ended
	StartTime      int64     `json:"start_time"`
	EndTime        int64     `json:"end_time"`
	WhiteLabelID   *uint     `gorm:"index" json:"white_label_id"`
}

// StakingPosition represents a user's staking position
type StakingPosition struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	PositionID    string    `gorm:"uniqueIndex" json:"position_id"`
	UserID        string    `gorm:"index" json:"user_id"`
	PoolID        string    `gorm:"index" json:"pool_id"`
	Amount        string    `json:"amount"`
	RewardsEarned string    `json:"rewards_earned"`
	RewardsClaimed string  `json:"rewards_claimed"`
	StartTime     int64     `json:"start_time"`
	UnlockTime    int64     `json:"unlock_time"`
	Status        string    `json:"status"` // staked, claimed, withdrawn
	ChainID       int64     `json:"chain_id"`
	WhiteLabelID  *uint     `gorm:"index" json:"white_label_id"`
}

// LaunchpadProject represents an IDO/IEO project
type LaunchpadProject struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	ProjectID      string    `gorm:"uniqueIndex" json:"project_id"`
	Name           string    `json:"name"`
	Symbol         string    `json:"symbol"`
	Description    string    `json:"description"`
	Logo           string    `json:"logo"`
	Website        string    `json:"website"`
	Whitepaper     string    `json:"whitepaper"`
	TokenAddress   string    `json:"token_address"`
	ChainID        int64     `json:"chain_id"`
	TotalSupply    string    `json:"total_supply"`
	IDOAllocation  string    `json:"ido_allocation"`
	IDOPrice       string    `json:"ido_price"`
	HardCap        string    `json:"hard_cap"`
	SoftCap        string    `json:"soft_cap"`
	MinBuy         string    `json:"min_buy"`
	MaxBuy         string    `json:"max_buy"`
	StartTime      int64     `json:"start_time"`
	EndTime        int64     `json:"end_time"`
	Progress       float64   `json:"progress"`
	Participants   int       `json:"participants"`
	RaisedAmount   string    `json:"raised_amount"`
	Status         string    `json:"status"` // upcoming, active, completed, cancelled
	WhiteLabelID   *uint     `gorm:"index" json:"white_label_id"`
}

// LaunchpadAllocation represents user allocation
type LaunchpadAllocation struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	AllocationID  string    `gorm:"uniqueIndex" json:"allocation_id"`
	UserID        string    `gorm:"index" json:"user_id"`
	ProjectID     string    `gorm:"index" json:"project_id"`
	PayToken      string    `json:"pay_token"`
	PayAmount     string    `json:"pay_amount"`
	TokenAmount   string    `json:"token_amount"`
	Status        string    `json:"status"` // pending, confirmed, cancelled
	TxHash        string    `json:"tx_hash"`
	ChainID       int64     `json:"chain_id"`
	WhiteLabelID  *uint     `gorm:"index" json:"white_label_id"`
}

// EarnProduct represents savings/earn products
type EarnProduct struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	ProductID     string    `gorm:"uniqueIndex" json:"product_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	TokenAddress  string    `json:"token_address"`
	ChainID       int64     `json:"chain_id"`
	APY           float64   `json:"apy"`
	ProductType   string    `json:"product_type"` // flexible, fixed
	TermDays      int       `json:"term_days"`
	MinDeposit    string    `json:"min_deposit"`
	MaxDeposit    string    `json:"max_deposit"`
	TotalDeposited string   `json:"total_deposited"`
	Status        string    `json:"status"` // active, paused, ended
	WhiteLabelID *uint     `gorm:"index" json:"white_label_id"`
}

// EarnDeposit represents user deposits
type EarnDeposit struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	DepositID     string    `gorm:"uniqueIndex" json:"deposit_id"`
	UserID        string    `gorm:"index" json:"user_id"`
	ProductID    string    `gorm:"index" json:"product_id"`
	Amount        string    `json:"amount"`
	InterestEarned string   `json:"interest_earned"`
	Status        string    `json:"status"` // active, withdrawn
	ChainID       int64     `json:"chain_id"`
	WhiteLabelID *uint     `gorm:"index" json:"white_label_id"`
}

// Coupon represents promotional coupons
type Coupon struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	CouponCode    string    `gorm:"uniqueIndex" json:"coupon_code"`
	CouponType    string    `json:"coupon_type"` // discount, cashback, bonus
	DiscountType  string    `json:"discount_type"` // percentage, fixed
	DiscountValue string    `json:"discount_value"`
	MinAmount     string    `json:"min_amount"`
	MaxAmount     string    `json:"max_amount"`
	ValidFrom     int64     `json:"valid_from"`
	ValidUntil   int64     `json:"valid_until"`
	UsageLimit    int       `json:"usage_limit"`
	UsageCount    int       `json:"usage_count"`
	Status        string    `json:"status"` // active, expired, cancelled
}

// RedPacket represents red packet events
type RedPacket struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	PacketID      string    `gorm:"uniqueIndex" json:"packet_id"`
	SenderID      string    `gorm:"index" json:"sender_id"`
	TokenAddress  string    `json:"token_address"`
	ChainID       int64     `json:"chain_id"`
	TotalAmount   string    `json:"total_amount"`
	TotalCount    int       `json:"total_count"`
	ClaimedCount  int       `json:"claimed_count"`
	ClaimedAmount string    `json:"claimed_amount"`
	Message       string    `json:"message"`
	Status        string    `json:"status"` // active, claimed, expired
	ExpiresAt     int64     `json:"expires_at"`
}

// RedPacketClaim represents claims
type RedPacketClaim struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	ClaimID       string    `gorm:"uniqueIndex" json:"claim_id"`
	PacketID      string    `gorm:"index" json:"packet_id"`
	ClaimerID     string    `gorm:"index" json:"claimer_id"`
	Amount        string    `json:"amount"`
	ClaimTime    int64     `json:"claim_time"`
	ChainID       int64     `json:"chain_id"`
}

// ============================================================================
// DeFi Service
// ============================================================================

type DeFiService struct {
	db     *gorm.DB
	redis  *redis.Client
	config *Config
	mu     sync.RWMutex
}

func NewDeFiService(config *Config) (*DeFiService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&StakingPool{}, &StakingPosition{},
		&LaunchpadProject{}, &LaunchpadAllocation{},
		&EarnProduct{}, &EarnDeposit{},
		&Coupon{}, &RedPacket{}, &RedPacketClaim{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	// Initialize default staking pools
	service := &DeFiService{db: db, redis: rdb, config: config}
	service.initDefaultPools()

	return service, nil
}

func (s *DeFiService) initDefaultPools() {
	var count int64
	s.db.Model(&StakingPool{}).Count(&count)
	if count > 0 {
		return
	}

	// Create default staking pools
	pools := []StakingPool{
		{PoolID: uuid.New().String(), Name: "ETH Staking", Description: "Stake ETH and earn rewards", TokenAddress: "0x0000000000000000000000000000000000000000", ChainID: 1, RewardToken: "0x0000000000000000000000000000000000000000", APY: 4.5, MinStake: "0.01", MaxStake: "10000", LockPeriod: 2592000, Status: "active", StartTime: time.Now().Unix()},
		{PoolID: uuid.New().String(), Name: "USDT Staking", Description: "Stake USDT and earn stable yields", TokenAddress: "0xdAC17F958D2ee523a2206206994597C13D831ec7", ChainID: 1, RewardToken: "0xdAC17F958D2ee523a2206206994597C13D831ec7", APY: 8.2, MinStake: "100", MaxStake: "1000000", LockPeriod: 0, Status: "active", StartTime: time.Now().Unix()},
		{PoolID: uuid.New().String(), Name: "BNB Staking", Description: "Stake BNB and earn rewards", TokenAddress: "0x0000000000000000000000000000000000000000", ChainID: 56, RewardToken: "0x0000000000000000000000000000000000000000", APY: 6.8, MinStake: "0.1", MaxStake: "10000", LockPeriod: 1296000, Status: "active", StartTime: time.Now().Unix()},
	}

	for _, pool := range pools {
		s.db.Create(&pool)
	}

	// Create default earn products
	products := []EarnProduct{
		{ProductID: uuid.New().String(), Name: "Flexible USDT Savings", Description: "Earn interest on your USDT with flexible withdrawal", TokenAddress: "0xdAC17F958D2ee523a2206206994597C13D831ec7", ChainID: 1, APY: 6.5, ProductType: "flexible", MinDeposit: "10", MaxDeposit: "1000000", Status: "active"},
		{ProductID: uuid.New().String(), Name: "Fixed 30-Day USDC", Description: "Lock your USDC for 30 days for higher yields", TokenAddress: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", ChainID: 1, APY: 8.0, ProductType: "fixed", TermDays: 30, MinDeposit: "100", MaxDeposit: "100000", Status: "active"},
		{ProductID: uuid.New().String(), Name: "Flexible ETH Savings", Description: "Earn interest on your ETH", TokenAddress: "0x0000000000000000000000000000000000000000", ChainID: 1, APY: 4.0, ProductType: "flexible", MinDeposit: "0.01", MaxDeposit: "1000", Status: "active"},
	}

	for _, product := range products {
		s.db.Create(&product)
	}
}

// ============================================================================
// Staking Handlers
// ============================================================================

type CreateStakeRequest struct {
	UserID    string `json:"user_id" binding:"required"`
	PoolID    string `json:"pool_id" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
	ChainID   int64  `json:"chain_id"`
}

func (s *DeFiService) CreateStake(ctx *gin.Context) {
	var req CreateStakeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Get pool
	var pool StakingPool
	if err := s.db.Where("pool_id = ? AND status = ?", req.PoolID, "active").First(&pool).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "pool not found or inactive"})
		return
	}

	// Validate amount
	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid amount"})
		return
	}

	minStake, _ := strconv.ParseFloat(pool.MinStake, 64)
	maxStake, _ := strconv.ParseFloat(pool.MaxStake, 64)

	if amount < minStake || amount > maxStake {
		ctx.JSON(400, gin.H{"error": fmt.Sprintf("amount must be between %s and %s", pool.MinStake, pool.MaxStake)})
		return
	}

	// Create position
	now := time.Now().Unix()
	unlockTime := now + int64(pool.LockPeriod)

	position := &StakingPosition{
		PositionID:    uuid.New().String(),
		UserID:         req.UserID,
		PoolID:         req.PoolID,
		Amount:         req.Amount,
		RewardsEarned:  "0",
		RewardsClaimed: "0",
		StartTime:      now,
		UnlockTime:     unlockTime,
		Status:         "staked",
		ChainID:        req.ChainID,
	}

	if err := s.db.Create(position).Error; err != nil {
		ctx.JSON(500, gin.H{"error": "failed to create stake"})
		return
	}

	// Update pool total
	pool.TotalStaked = fmt.Sprintf("%.8f", mustParseFloat(pool.TotalStaked)+amount)
	s.db.Save(&pool)

	ctx.JSON(200, gin.H{
		"success":     true,
		"position_id": position.PositionID,
		"amount":      position.Amount,
		"unlock_time": position.UnlockTime,
		"status":      position.Status,
	})
}

func (s *DeFiService) ClaimStakeRewards(ctx *gin.Context) {
	positionID := ctx.Param("id")
	userID := ctx.Query("user_id")

	var position StakingPosition
	if err := s.db.Where("position_id = ? AND user_id = ?", positionID, userID).First(&position).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "position not found"})
		return
	}

	// Calculate rewards (simplified)
	amount, _ := strconv.ParseFloat(position.Amount, 64)
	pool := StakingPool{}
	s.db.Where("pool_id = ?", position.PoolID).First(&pool)

	daysStaked := float64(time.Now().Unix()-position.StartTime) / 86400
	rewards := amount * (pool.APY / 100) * (daysStaked / 365)

	position.RewardsEarned = fmt.Sprintf("%.8f", rewards)
	s.db.Save(&position)

	ctx.JSON(200, gin.H{
		"success":        true,
		"rewards_earned": position.RewardsEarned,
	})
}

func (s *DeFiService) ListStakingPools(ctx *gin.Context) {
	var pools []StakingPool
	s.db.Where("status = ?", "active").Find(&pools)

	ctx.JSON(200, gin.H{"pools": pools})
}

func (s *DeFiService) GetStakingPositions(ctx *gin.Context) {
	userID := ctx.Query("user_id")

	var positions []StakingPosition
	s.db.Where("user_id = ?", userID).Find(&positions)

	ctx.JSON(200, gin.H{"positions": positions})
}

// ============================================================================
// Launchpad Handlers
// ============================================================================

func (s *DeFiService) ListLaunchpadProjects(ctx *gin.Context) {
	status := ctx.Query("status")

	var projects []LaunchpadProject
	query := s.db.Model(&LaunchpadProject{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Find(&projects)

	ctx.JSON(200, gin.H{"projects": projects})
}

type ContributeRequest struct {
	UserID    string `json:"user_id" binding:"required"`
	ProjectID string `json:"project_id" binding:"required"`
	PayToken  string `json:"pay_token" binding:"required"`
	PayAmount string `json:"pay_amount" binding:"required"`
	ChainID   int64  `json:"chain_id"`
}

func (s *DeFiService) ContributeToProject(ctx *gin.Context) {
	var req ContributeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Get project
	var project LaunchpadProject
	if err := s.db.Where("project_id = ? AND status = ?", req.ProjectID, "active").First(&project).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "project not found or inactive"})
		return
	}

	// Check timing
	now := time.Now().Unix()
	if now < project.StartTime || now > project.EndTime {
		ctx.JSON(400, gin.H{"error": "contribution period not active"})
		return
	}

	// Validate amount
	payAmount, _ := strconv.ParseFloat(req.PayAmount, 64)
	minBuy, _ := strconv.ParseFloat(project.MinBuy, 64)
	maxBuy, _ := strconv.ParseFloat(project.MaxBuy, 64)

	if payAmount < minBuy || payAmount > maxBuy {
		ctx.JSON(400, gin.H{"error": fmt.Sprintf("amount must be between %s and %s", project.MinBuy, project.MaxBuy)})
		return
	}

	// Calculate token amount
	idoPrice, _ := strconv.ParseFloat(project.IDOPrice, 64)
	tokenAmount := payAmount / idoPrice

	// Create allocation
	allocation := &LaunchpadAllocation{
		AllocationID: uuid.New().String(),
		UserID:       req.UserID,
		ProjectID:    req.ProjectID,
		PayToken:     req.PayToken,
		PayAmount:    req.PayAmount,
		TokenAmount:  fmt.Sprintf("%.8f", tokenAmount),
		Status:       "confirmed",
		ChainID:      req.ChainID,
	}

	if err := s.db.Create(allocation).Error; err != nil {
		ctx.JSON(500, gin.H{"error": "failed to create allocation"})
		return
	}

	// Update project
	raised, _ := strconv.ParseFloat(project.RaisedAmount, 64)
	project.RaisedAmount = fmt.Sprintf("%.8f", raised+payAmount)
	project.Participants++
	project.Progress = (raised + payAmount) / mustParseFloat(project.HardCap) * 100

	if project.Progress >= 100 {
		project.Status = "completed"
	}

	s.db.Save(&project)

	ctx.JSON(200, gin.H{
		"success":       true,
		"allocation_id": allocation.AllocationID,
		"token_amount":  allocation.TokenAmount,
	})
}

func (s *DeFiService) GetLaunchpadAllocations(ctx *gin.Context) {
	userID := ctx.Query("user_id")

	var allocations []LaunchpadAllocation
	s.db.Where("user_id = ?", userID).Find(&allocations)

	ctx.JSON(200, gin.H{"allocations": allocations})
}

// ============================================================================
// Earn Products Handlers
// ============================================================================

func (s *DeFiService) ListEarnProducts(ctx *gin.Context) {
	var products []EarnProduct
	s.db.Where("status = ?", "active").Find(&products)

	ctx.JSON(200, gin.H{"products": products})
}

type DepositRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	ProductID string `json:"product_id" binding:"required"`
	Amount   string `json:"amount" binding:"required"`
	ChainID  int64  `json:"chain_id"`
}

func (s *DeFiService) CreateEarnDeposit(ctx *gin.Context) {
	var req DepositRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Get product
	var product EarnProduct
	if err := s.db.Where("product_id = ? AND status = ?", req.ProductID, "active").First(&product).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "product not found or inactive"})
		return
	}

	// Validate amount
	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "invalid amount"})
		return
	}

	minDeposit, _ := strconv.ParseFloat(product.MinDeposit, 64)
	maxDeposit, _ := strconv.ParseFloat(product.MaxDeposit, 64)

	if amount < minDeposit || amount > maxDeposit {
		ctx.JSON(400, gin.H{"error": fmt.Sprintf("amount must be between %s and %s", product.MinDeposit, product.MaxDeposit)})
		return
	}

	// Create deposit
	deposit := &EarnDeposit{
		DepositID:      uuid.New().String(),
		UserID:          req.UserID,
		ProductID:       req.ProductID,
		Amount:          req.Amount,
		InterestEarned:  "0",
		Status:          "active",
		ChainID:         req.ChainID,
	}

	if err := s.db.Create(deposit).Error; err != nil {
		ctx.JSON(500, gin.H{"error": "failed to create deposit"})
		return
	}

	// Update product total
	product.TotalDeposited = fmt.Sprintf("%.8f", mustParseFloat(product.TotalDeposited)+amount)
	s.db.Save(&product)

	ctx.JSON(200, gin.H{
		"success":    true,
		"deposit_id": deposit.DepositID,
		"amount":     deposit.Amount,
		"apy":        product.APY,
	})
}

func (s *DeFiService) WithdrawEarnDeposit(ctx *gin.Context) {
	depositID := ctx.Param("id")
	userID := ctx.Query("user_id")

	var deposit EarnDeposit
	if err := s.db.Where("deposit_id = ? AND user_id = ?", depositID, userID).First(&deposit).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "deposit not found"})
		return
	}

	if deposit.Status != "active" {
		ctx.JSON(400, gin.H{"error": "deposit already withdrawn"})
		return
	}

	// Calculate interest
	product := EarnProduct{}
	s.db.Where("product_id = ?", deposit.ProductID).First(&product)

	daysDeposited := float64(time.Now().Unix()-deposit.CreatedAt.Unix()) / 86400
	principal, _ := strconv.ParseFloat(deposit.Amount, 64)
	interest := principal * (product.APY / 100) * (daysDeposited / 365)

	deposit.InterestEarned = fmt.Sprintf("%.8f", interest)
	deposit.Status = "withdrawn"
	s.db.Save(&deposit)

	// Update product total
	product.TotalDeposited = fmt.Sprintf("%.8f", mustParseFloat(product.TotalDeposited)-principal)
	s.db.Save(&product)

	ctx.JSON(200, gin.H{
		"success":         true,
		"principal":       deposit.Amount,
		"interest_earned": deposit.InterestEarned,
	})
}

func (s *DeFiService) GetEarnDeposits(ctx *gin.Context) {
	userID := ctx.Query("user_id")

	var deposits []EarnDeposit
	s.db.Where("user_id = ?", userID).Find(&deposits)

	ctx.JSON(200, gin.H{"deposits": deposits})
}

// ============================================================================
// Coupon Handlers
// ============================================================================

func (s *DeFiService) ValidateCoupon(ctx *gin.Context) {
	var req struct {
		CouponCode string  `json:"coupon_code" binding:"required"`
		Amount     float64 `json:"amount"`
	}
	ctx.ShouldBindJSON(&req)

	var coupon Coupon
	if err := s.db.Where("coupon_code = ? AND status = ?", req.CouponCode, "active").First(&coupon).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "coupon not found"})
		return
	}

	// Check validity
	now := time.Now().Unix()
	if now < coupon.ValidFrom || now > coupon.ValidUntil {
		ctx.JSON(400, gin.H{"error": "coupon expired"})
		return
	}

	// Check usage limit
	if coupon.UsageLimit > 0 && coupon.UsageCount >= coupon.UsageLimit {
		ctx.JSON(400, gin.H{"error": "coupon usage limit reached"})
		return
	}

	// Check min amount
	if req.Amount > 0 {
		minAmount, _ := strconv.ParseFloat(coupon.MinAmount, 64)
		if req.Amount < minAmount {
			ctx.JSON(400, gin.H{"error": fmt.Sprintf("minimum amount required: %s", coupon.MinAmount)})
			return
		}
	}

	var discount float64
	if coupon.DiscountType == "percentage" {
		discountVal, _ := strconv.ParseFloat(coupon.DiscountValue, 64)
		discount = req.Amount * (discountVal / 100)
	} else {
		discountVal, _ := strconv.ParseFloat(coupon.DiscountValue, 64)
		discount = discountVal
	}

	ctx.JSON(200, gin.H{
		"valid":      true,
		"discount":   fmt.Sprintf("%.2f", discount),
		"coupon_type": coupon.CouponType,
	})
}

// ============================================================================
// Red Packet Handlers
// ============================================================================

type CreateRedPacketRequest struct {
	SenderID     string `json:"sender_id" binding:"required"`
	TokenAddress string `json:"token_address"`
	ChainID      int64  `json:"chain_id"`
	TotalAmount string `json:"total_amount" binding:"required"`
	TotalCount  int    `json:"total_count" binding:"required"`
	Message     string `json:"message"`
	ExpiresAt   int64  `json:"expires_at"`
}

func (s *DeFiService) CreateRedPacket(ctx *gin.Context) {
	var req CreateRedPacketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.TotalCount <= 0 || req.TotalCount > 10000 {
		ctx.JSON(400, gin.H{"error": "total count must be between 1 and 10000"})
		return
	}

	expiresAt := req.ExpiresAt
	if expiresAt == 0 {
		expiresAt = time.Now().Add(24 * time.Hour).Unix()
	}

	packet := &RedPacket{
		PacketID:      uuid.New().String(),
		SenderID:      req.SenderID,
		TokenAddress:  req.TokenAddress,
		ChainID:       req.ChainID,
		TotalAmount:   req.TotalAmount,
		TotalCount:    req.TotalCount,
		ClaimedCount:  0,
		ClaimedAmount: "0",
		Message:       req.Message,
		Status:        "active",
		ExpiresAt:     expiresAt,
	}

	if err := s.db.Create(packet).Error; err != nil {
		ctx.JSON(500, gin.H{"error": "failed to create red packet"})
		return
	}

	// Generate claim ID for sender (deterministic equal-share distribution;
	// the exact per-claim amounts are derived verifiably in ClaimRedPacket).
	ctx.JSON(200, gin.H{
		"success":      true,
		"packet_id":    packet.PacketID,
		"total_amount": packet.TotalAmount,
		"total_count":  packet.TotalCount,
		"expires_at":   packet.ExpiresAt,
	})
}

func (s *DeFiService) ClaimRedPacket(ctx *gin.Context) {
	var req struct {
		PacketID  string `json:"packet_id" binding:"required"`
		ClaimerID string `json:"claimer_id" binding:"required"`
		ChainID   int64  `json:"chain_id"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var packet RedPacket
	if err := s.db.Where("packet_id = ? AND status = ?", req.PacketID, "active").First(&packet).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "red packet not found"})
		return
	}

	// Check expiration
	if time.Now().Unix() > packet.ExpiresAt {
		packet.Status = "expired"
		s.db.Save(&packet)
		ctx.JSON(400, gin.H{"error": "red packet expired"})
		return
	}

	// Check if fully claimed
	if packet.ClaimedCount >= packet.TotalCount {
		ctx.JSON(400, gin.H{"error": "red packet fully claimed"})
		return
	}

	// Deterministic, verifiable split: TotalAmount is divided evenly across
	// TotalCount at 8-decimal precision; any remainder unit is added to the
	// first claim(s). This guarantees the sum of all claims exactly equals
	// TotalAmount and is reproducible from (TotalAmount, TotalCount, claim
	// index) alone — no math/rand and no floating-point drift.
	claimUnits, claimedUnits, ok := computeRedPacketClaimUnits(packet.TotalAmount, packet.TotalCount, packet.ClaimedCount)
	if !ok {
		ctx.JSON(400, gin.H{"error": "invalid red packet amount"})
		return
	}
	claimAmount := unitsToDecimalString(claimUnits)

	// Create claim
	claim := &RedPacketClaim{
		ClaimID:   uuid.New().String(),
		PacketID:  req.PacketID,
		ClaimerID: req.ClaimerID,
		Amount:    claimAmount,
		ClaimTime: time.Now().Unix(),
		ChainID:   req.ChainID,
	}

	if err := s.db.Create(claim).Error; err != nil {
		ctx.JSON(500, gin.H{"error": "failed to claim"})
		return
	}

	// Update packet
	packet.ClaimedCount++
	packet.ClaimedAmount = unitsToDecimalString(claimedUnits)

	if packet.ClaimedCount >= packet.TotalCount {
		packet.Status = "claimed"
	}

	s.db.Save(&packet)

	ctx.JSON(200, gin.H{
		"success":   true,
		"claim_id":  claim.ClaimID,
		"amount":    claim.Amount,
		"remaining": packet.TotalCount - packet.ClaimedCount,
	})
}

// redPacketUnitScale is the number of decimals amounts are tracked at.
const redPacketUnitScale = 8

// computeRedPacketClaimUnits returns the integer-unit amount for the claim at
// position claimIndex (0-based, i.e. the current ClaimedCount) plus the total
// units claimed so far (including this claim). TotalAmount is parsed as a
// decimal string and scaled to integer units. Returns ok=false on parse error
// or non-positive TotalCount.
func computeRedPacketClaimUnits(totalAmount string, totalCount, claimIndex int) (claimUnits, claimedUnits *big.Int, ok bool) {
	totalUnits, ok := decimalStringToUnits(totalAmount)
	if !ok || totalCount <= 0 || claimIndex < 0 || claimIndex >= totalCount {
		return nil, nil, false
	}

	base := new(big.Int).Quo(totalUnits, big.NewInt(int64(totalCount)))
	remainder := new(big.Int).Rem(totalUnits, big.NewInt(int64(totalCount)))

	// The first `remainder` claims receive one extra unit.
	claim := new(big.Int).Set(base)
	if int64(claimIndex) < remainder.Int64() {
		claim.Add(claim, big.NewInt(1))
	}

	// Total claimed after this claim = base*(claimIndex+1) + min(remainder, claimIndex+1)
	claimed := new(big.Int).Mul(base, big.NewInt(int64(claimIndex+1)))
	extra := new(big.Int).Set(remainder)
	if extra.Int64() > int64(claimIndex+1) {
		extra.SetInt64(int64(claimIndex + 1))
	}
	claimed.Add(claimed, extra)

	return claim, claimed, true
}

// decimalStringToUnits parses a decimal string and scales it to integer units
// at redPacketUnitScale decimals (e.g. "1.5" -> 150000000). Trailing digits
// beyond the scale are truncated. Returns ok=false on parse failure.
func decimalStringToUnits(s string) (*big.Int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	negative := false
	if s[0] == '-' {
		negative = true
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	if intPart == "" {
		intPart = "0"
	}
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}

	intVal, ok := new(big.Int).SetString(intPart, 10)
	if !ok {
		return nil, false
	}
	// Scale integer part.
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(redPacketUnitScale), nil)
	intVal.Mul(intVal, scale)

	// Scale fractional part, padding/truncating to redPacketUnitScale digits.
	if len(fracPart) > redPacketUnitScale {
		fracPart = fracPart[:redPacketUnitScale]
	}
	for len(fracPart) < redPacketUnitScale {
		fracPart += "0"
	}
	fracVal, ok := new(big.Int).SetString(fracPart, 10)
	if !ok {
		return nil, false
	}

	total := new(big.Int).Add(intVal, fracVal)
	if negative {
		total.Neg(total)
	}
	return total, true
}

// unitsToDecimalString converts integer units back to a decimal string with
// exactly redPacketUnitScale decimal places.
func unitsToDecimalString(units *big.Int) string {
	negative := units.Sign() < 0
	abs := new(big.Int).Abs(units)
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(redPacketUnitScale), nil)
	intPart := new(big.Int).Quo(abs, scale)
	fracPart := new(big.Int).Rem(abs, scale)

	fracStr := fracPart.String()
	for len(fracStr) < redPacketUnitScale {
		fracStr = "0" + fracStr
	}

	out := intPart.String() + "." + fracStr
	if negative {
		out = "-" + out
	}
	return out
}

// ============================================================================
// Dashboard Stats
// ============================================================================

func (s *DeFiService) GetDashboardStats(ctx *gin.Context) {
	var totalStaked, totalDeposited, totalRaised float64
	var stakingCount, earnCount, launchpadCount int64

	s.db.Model(&StakingPosition{}).Count(&stakingCount)
	s.db.Model(&EarnDeposit{}).Count(&earnCount)
	s.db.Model(&LaunchpadAllocation{}).Count(&launchpadCount)

	s.db.Model(&StakingPool{}).Select("COALESCE(SUM(CAST(total_staked AS DECIMAL(20,8))), 0)").Row().Scan(&totalStaked)
	s.db.Model(&EarnProduct{}).Select("COALESCE(SUM(CAST(total_deposited AS DECIMAL(20,8))), 0)").Row().Scan(&totalDeposited)
	s.db.Model(&LaunchpadProject{}).Select("COALESCE(SUM(CAST(raised_amount AS DECIMAL(20,8))), 0)").Row().Scan(&totalRaised)

	ctx.JSON(200, gin.H{
		"staking": gin.H{
			"total_staked":   totalStaked,
			"positions":      stakingCount,
		},
		"earn": gin.H{
			"total_deposited": totalDeposited,
			"deposits":        earnCount,
		},
		"launchpad": gin.H{
			"total_raised": totalRaised,
			"allocations":  launchpadCount,
		},
	})
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	service, err := NewDeFiService(config)
	if err != nil {
		fmt.Printf("Failed to initialize DeFi service: %v\n", err)
		os.Exit(1)
	}

	router := gin.Default()

	// CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API routes
	api := router.Group("/api/v1/defi")
	{
		// Staking
		api.GET("/staking/pools", service.ListStakingPools)
		api.POST("/staking/stake", service.CreateStake)
		api.POST("/staking/claim/:id", service.ClaimStakeRewards)
		api.GET("/staking/positions", service.GetStakingPositions)

		// Launchpad
		api.GET("/launchpad/projects", service.ListLaunchpadProjects)
		api.POST("/launchpad/contribute", service.ContributeToProject)
		api.GET("/launchpad/allocations", service.GetLaunchpadAllocations)

		// Earn
		api.GET("/earn/products", service.ListEarnProducts)
		api.POST("/earn/deposit", service.CreateEarnDeposit)
		api.POST("/earn/withdraw/:id", service.WithdrawEarnDeposit)
		api.GET("/earn/deposits", service.GetEarnDeposits)

		// Coupons
		api.POST("/coupon/validate", service.ValidateCoupon)

		// Red Packets
		api.POST("/redpacket/create", service.CreateRedPacket)
		api.POST("/redpacket/claim", service.ClaimRedPacket)

		// Dashboard
		api.GET("/dashboard", service.GetDashboardStats)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "defi-service",
			"time":    time.Now().Unix(),
		})
	})

	go func() {
		fmt.Printf("DeFi service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down DeFi service...")
}

func mustParseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
