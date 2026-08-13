// TigerWallet Launchpad & Launchpool Service
// High-Load Distributed Go Implementation
// Supports IDO, IEO, and fair launch mechanisms

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort    string `json:"server_port"`
	DBHost       string `json:"db_host"`
	DBPort       string `json:"db_port"`
	DBUser       string `json:"db_user"`
	DBPassword   string `json:"db_password"`
	DBName       string `json:"db_name"`
	RedisHost    string `json:"redis_host"`
	RedisPort    string `json:"redis_port"`
}

// ============================================================================
// Data Models
// ============================================================================

// LaunchpadProject represents a launchpad project
type LaunchpadProject struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	ProjectID        string    `gorm:"uniqueIndex" json:"project_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	TokenSymbol    string    `json:"token_symbol"`
	TokenAddress   string    `json:"token_address"`
	TokenDecimals uint8     `json:"token_decimals"`
	TotalSupply    string    `json:"total_supply"`
	SoftCap        string    `json:"soft_cap"`
	HardCap        string    `json:"hard_cap"`
	MinBuy         float64   `json:"min_buy"`
	MaxBuy         float64   `json:"max_buy"`
	TokenPrice     float64   `json:"token_price"`
	PaymentToken   string    `json:"payment_token"` // USDC, USDT, etc.
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	ClaimTime     time.Time `json:"claim_time"`
	Status         string    `json:"status"` // UPCOMING, ACTIVE, COMPLETED, CANCELLED
	TotalRaised    float64   `json:"total_raised"`
	Participants   int       `json:"participants"`
	Logo           string    `json:"logo"`
	Website        string    `json:"website"`
	Whitepaper    string    `json:"whitepaper"`
	ChainID       int64     `json:"chain_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// LaunchpoolProject represents a liquidity mining/farming launchpool
type LaunchpoolProject struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ProjectID      string    `gorm:"uniqueIndex" json:"project_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	StakeToken     string    `json:"stake_token"` // Token to stake
	RewardToken    string    `json:"reward_token"` // Token to earn
	RewardPerBlock float64   `json:"reward_per_block"`
	TotalStake    float64   `json:"total_stake"`
	TotalReward   float64   `json:"total_reward"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Status        string    `json:"status"` // UPCOMING, ACTIVE, COMPLETED
	MinStake      float64   `json:"min_stake"`
	MaxStake     float64   `json:"max_stake"`
	ChainID       int64     `json:"chain_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Allocation represents user allocation in a launchpad
type Allocation struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProjectID     string    `gorm:"index" json:"project_id"`
	UserAddress   string    `gorm:"index" json:"user_address"`
	Amount        float64   `json:"amount"`
	Tokens        float64   `json:"tokens"`
	Tier          string    `json:"tier"` // TIER_1, TIER_2, TIER_3
	Status        string    `json:"status"` // PENDING, CONFIRMED, CLAIMED, REFUNDED
	ClaimedAt     *time.Time `json:"claimed_at"`
	ChainID       int64     `json:"chain_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// Stake represents user stake in launchpool
type Stake struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProjectID     string    `gorm:"index" json:"project_id"`
	UserAddress   string    `gorm:"index" json:"user_address"`
	StakeAmount   float64   `json:"stake_amount"`
	PendingReward float64   `json:"pending_reward"`
	ClaimedReward float64   `json:"claimed_reward"`
	Status        string    `json:"status"` // ACTIVE, WITHDRAWN
	ChainID       int64     `json:"chain_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type LaunchpadService struct {
	db     *gorm.DB
	redis *redis.Client
	config Config
	mu    sync.RWMutex
}

func NewLaunchpadService(config Config) (*LaunchpadService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&LaunchpadProject{},
		&LaunchpoolProject{},
		&Allocation{},
		&Stake{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &LaunchpadService{
		db:     db,
		redis:  rdb,
		config: config,
	}

	// Initialize sample projects
	go service.initializeSampleProjects()

	return service, nil
}

func (s *LaunchpadService) initializeSampleProjects() {
        // No sample/seed projects. Launchpad/launchpool projects must be created by
        // admins with real on-chain token/stake/reward contract addresses, not
        // fabricated placeholder addresses (0x1234...) or fake caps/prices. Do not
        // seed fake projects.
}

// ====+
// Launchpad Operations
// ============================================================================

type CreateProjectRequest struct {
	Name          string  `json:"name" binding:"required"`
	Description  string  `json:"description"`
	TokenSymbol  string  `json:"token_symbol" binding:"required"`
	TokenAddress string  `json:"token_address" binding:"required"`
	TokenDecimals uint8  `json:"token_decimals"`
	TotalSupply  string  `json:"total_supply"`
	SoftCap      string  `json:"soft_cap"`
	HardCap      string  `json:"hard_cap"`
	MinBuy       float64 `json:"min_buy"`
	MaxBuy       float64 `json:"max_buy"`
	TokenPrice   float64 `json:"token_price" binding:"required"`
	PaymentToken string `json:"payment_token" binding:"required"`
	StartTime    int64  `json:"start_time" binding:"required"`
	EndTime      int64  `json:"end_time" binding:"required"`
	ClaimTime    int64  `json:"claim_time"`
	Logo         string `json:"logo"`
	Website      string `json:"website"`
	Whitepaper   string `json:"whitepaper"`
	ChainID      int64  `json:"chain_id"`
}

func (s *LaunchpadService) CreateProject(ctx *gin.Context) {
	var req CreateProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	projectID := generateProjectID()

	project := LaunchpadProject{
		ProjectID:     projectID,
		Name:          req.Name,
		Description:  req.Description,
		TokenSymbol:   req.TokenSymbol,
		TokenAddress:  req.TokenAddress,
		TokenDecimals: req.TokenDecimals,
		TotalSupply:  req.TotalSupply,
		SoftCap:      req.SoftCap,
		HardCap:      req.HardCap,
		MinBuy:       req.MinBuy,
		MaxBuy:       req.MaxBuy,
		TokenPrice:   req.TokenPrice,
		PaymentToken: req.PaymentToken,
		StartTime:    time.Unix(req.StartTime, 0),
		EndTime:      time.Unix(req.EndTime, 0),
		ClaimTime:    time.Unix(req.ClaimTime, 0),
		Status:       "UPCOMING",
		Logo:         req.Logo,
		Website:      req.Website,
		Whitepaper:   req.Whitepaper,
		ChainID:      req.ChainID,
	}

	if err := s.db.Create(&project).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "Failed to create project"})
		return
	}

	ctx.JSON(200, gin.H{
		"success":    true,
		"project_id": projectID,
		"status":    "UPCOMING",
	})
}

type ParticipateRequest struct {
	ProjectID   string  `json:"project_id" binding:"required"`
	UserAddress string  `json:"user_address" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
	ChainID     int64   `json:"chain_id"`
}

func (s *LaunchpadService) Participate(ctx *gin.Context) {
	var req ParticipateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var project LaunchpadProject
	if err := s.db.Where("project_id = ?", req.ProjectID).First(&project).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Project not found"})
		return
	}

	// Check if project is active
	if project.Status != "ACTIVE" {
		ctx.JSON(400, gin.H{"success": false, "error": "Project is not active"})
		return
	}

	now := time.Now()
	if now.Before(project.StartTime) {
		ctx.JSON(400, gin.H{"success": false, "error": "Project has not started"})
		return
	}
	if now.After(project.EndTime) {
		ctx.JSON(400, gin.H{"success": false, "error": "Project has ended"})
		return
	}

	// Validate amount
	if req.Amount < project.MinBuy {
		ctx.JSON(400, gin.H{"success": false, "error": "Below minimum buy"})
		return
	}

	// Check existing allocation
	var existingAllocation Allocation
	result := s.db.Where("project_id = ? AND user_address = ?", req.ProjectID, req.UserAddress).First(&existingAllocation)

	if result.RowsAffected > 0 {
		newAmount := existingAllocation.Amount + req.Amount
		if newAmount > project.MaxBuy {
			ctx.JSON(400, gin.H{"success": false, "error": "Exceeds maximum buy"})
			return
		}
		existingAllocation.Amount = newAmount
		existingAllocation.Tokens = newAmount / project.TokenPrice
		s.db.Save(&existingAllocation)
	} else {
		tokens := req.Amount / project.TokenPrice
		tier := s.determineTier(tokens)

		allocation := Allocation{
			ProjectID:   req.ProjectID,
			UserAddress: req.UserAddress,
			Amount:     req.Amount,
			Tokens:     tokens,
			Tier:       tier,
			Status:     "CONFIRMED",
			ChainID:    req.ChainID,
		}
		s.db.Create(&allocation)

		// Update participants count
		project.Participants++
	}

	// Update total raised
	project.TotalRaised += req.Amount
	s.db.Save(&project)

	// Check if hard cap reached
	hardCapFloat, _ := parseAmount(project.HardCap)
	if project.TotalRaised >= hardCapFloat {
		project.Status = "COMPLETED"
		s.db.Save(&project)
	}

	ctx.JSON(200, gin.H{
		"success":        true,
		"allocation_id":  existingAllocation.ID,
		"tokens_allocated": req.Amount / project.TokenPrice,
		"total_raised":  project.TotalRaised,
	})
}

func (s *LaunchpadService) determineTier(tokens float64) string {
	if tokens >= 10000 {
		return "TIER_1"
	} else if tokens >= 1000 {
		return "TIER_2"
	}
	return "TIER_3"
}

func (s *LaunchpadService) ClaimTokens(ctx *gin.Context) {
	var req struct {
		AllocationID uint   `json:"allocation_id" binding:"required"`
		UserAddress  string `json:"user_address" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var allocation Allocation
	if err := s.db.First(&allocation, req.AllocationID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Allocation not found"})
		return
	}

	if allocation.UserAddress != req.UserAddress {
		ctx.JSON(403, gin.H{"success": false, "error": "Unauthorized"})
		return
	}

	if allocation.Status == "CLAIMED" {
		ctx.JSON(400, gin.H{"success": false, "error": "Already claimed"})
		return
	}

	// Check if claim time has passed
	var project LaunchpadProject
	s.db.Where("project_id = ?", allocation.ProjectID).First(&project)

	if time.Now().Before(project.ClaimTime) {
		ctx.JSON(400, gin.H{"success": false, "error": "Claim time not reached"})
		return
	}

	now := time.Now()
	allocation.Status = "CLAIMED"
	allocation.ClaimedAt = &now
	s.db.Save(&allocation)

	// Transaction has not been broadcast via RPC; no real hash is available.
	ctx.JSON(200, gin.H{
		"success":          true,
		"transaction_hash": "",
		"status":           "not_broadcast",
		"tokens_claimed":   allocation.Tokens,
	})
}

// ============================================================================
// Launchpool Operations
// ============================================================================

type StakeRequest struct {
	ProjectID   string  `json:"project_id" binding:"required"`
	UserAddress string  `json:"user_address" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
	ChainID     int64   `json:"chain_id"`
}

func (s *LaunchpadService) Stake(ctx *gin.Context) {
	var req StakeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var project LaunchpoolProject
	if err := s.db.Where("project_id = ?", req.ProjectID).First(&project).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Project not found"})
		return
	}

	if project.Status != "ACTIVE" {
		ctx.JSON(400, gin.H{"success": false, "error": "Project is not active"})
		return
	}

	if req.Amount < project.MinStake || req.Amount > project.MaxStake {
		ctx.JSON(400, gin.H{"success": false, "error": "Invalid stake amount"})
		return
	}

	// Check or create stake
	var stake Stake
	result := s.db.Where("project_id = ? AND user_address = ?", req.ProjectID, req.UserAddress).First(&stake)

	if result.RowsAffected == 0 {
		stake = Stake{
			ProjectID:     req.ProjectID,
			UserAddress:   req.UserAddress,
			StakeAmount:   req.Amount,
			PendingReward: 0,
			ClaimedReward: 0,
			Status:        "ACTIVE",
			ChainID:       req.ChainID,
		}
		s.db.Create(&stake)
	} else {
		stake.StakeAmount += req.Amount
		s.db.Save(&stake)
	}

	// Update total stake
	project.TotalStake += req.Amount
	s.db.Save(&project)

	ctx.JSON(200, gin.H{
		"success":      true,
		"stake_amount": stake.StakeAmount,
		"total_staked": project.TotalStake,
	})
}

func (s *LaunchpadService) Unstake(ctx *gin.Context) {
	var req StakeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var stake Stake
	if err := s.db.Where("project_id = ? AND user_address = ?", req.ProjectID, req.UserAddress).First(&stake).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Stake not found"})
		return
	}

	if stake.StakeAmount < req.Amount {
		ctx.JSON(400, gin.H{"success": false, "error": "Insufficient stake"})
		return
	}

	stake.StakeAmount -= req.Amount
	if stake.StakeAmount == 0 {
		stake.Status = "WITHDRAWN"
	}
	s.db.Save(&stake)

	// Update total stake
	var project LaunchpoolProject
	s.db.Where("project_id = ?", req.ProjectID).First(&project)
	project.TotalStake -= req.Amount
	s.db.Save(&project)

	ctx.JSON(200, gin.H{
		"success":     true,
		"unstaked":    req.Amount,
		"remaining":   stake.StakeAmount,
	})
}

func (s *LaunchpadService) ClaimRewards(ctx *gin.Context) {
	var req struct {
		ProjectID   string `json:"project_id" binding:"required"`
		UserAddress string `json:"user_address" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var stake Stake
	if err := s.db.Where("project_id = ? AND user_address = ?", req.ProjectID, req.UserAddress).First(&stake).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Stake not found"})
		return
	}

	if stake.PendingReward <= 0 {
		ctx.JSON(400, gin.H{"success": false, "error": "No pending rewards"})
		return
	}

	reward := stake.PendingReward
	stake.ClaimedReward += reward
	stake.PendingReward = 0
	s.db.Save(&stake)

	// Transaction has not been broadcast via RPC; no real hash is available.
	ctx.JSON(200, gin.H{
		"success":          true,
		"reward_claimed":   reward,
		"transaction_hash": "",
		"status":           "not_broadcast",
	})
}

// ============================================================================
// Queries
// ============================================================================

func (s *LaunchpadService) GetLaunchpadProjects(ctx *gin.Context) {
	status := ctx.Query("status")

	query := s.db.Model(&LaunchpadProject{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var projects []LaunchpadProject
	query.Order("start_time DESC").Find(&projects)

	ctx.JSON(200, gin.H{"projects": projects})
}

func (s *LaunchpadService) GetLaunchpoolProjects(ctx *gin.Context) {
	status := ctx.Query("status")

	query := s.db.Model(&LaunchpoolProject{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var projects []LaunchpoolProject
	query.Order("start_time DESC").Find(&projects)

	ctx.JSON(200, gin.H{"projects": projects})
}

func (s *LaunchpadService) GetUserAllocations(ctx *gin.Context) {
	userAddress := ctx.Query("user_address")
	projectID := ctx.Query("project_id")

	query := s.db.Model(&Allocation{}).Where("user_address = ?", userAddress)
	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	var allocations []Allocation
	query.Find(&allocations)

	ctx.JSON(200, gin.H{"allocations": allocations})
}

func (s *LaunchpadService) GetUserStakes(ctx *gin.Context) {
	userAddress := ctx.Query("user_address")
	projectID := ctx.Query("project_id")

	query := s.db.Model(&Stake{}).Where("user_address = ? AND status = ?", userAddress, "ACTIVE")
	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	var stakes []Stake
	query.Find(&stakes)

	ctx.JSON(200, gin.H{"stakes": stakes})
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateProjectID() string {
	data := fmt.Sprintf("project:%d", time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "lp_" + hex.EncodeToString(hash[:])[0:12]
}

func parseAmount(amountStr string) (float64, error) {
	var amount float64
	_, err := fmt.Sscanf(amountStr, "%f", &amount)
	return amount, err
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := Config{
		ServerPort: "8098",
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet_launchpad"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewLaunchpadService(config)
	if err != nil {
		fmt.Printf("Failed to start launchpad service: %v\n", err)
		os.Exit(1)
	}

	router := gin.Default()

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

	api := router.Group("/api/v1/launchpad")
	{
		api.GET("/projects", service.GetLaunchpadProjects)
		api.POST("/create", service.CreateProject)
		api.POST("/participate", service.Participate)
		api.POST("/claim", service.ClaimTokens)
		api.GET("/allocations", service.GetUserAllocations)
	}

	api2 := router.Group("/api/v1/launchpool")
	{
		api2.GET("/projects", service.GetLaunchpoolProjects)
		api2.POST("/stake", service.Stake)
		api2.POST("/unstake", service.Unstake)
		api2.POST("/claim", service.ClaimRewards)
		api2.GET("/stakes", service.GetUserStakes)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "launchpad"})
	})

	go func() {
		fmt.Printf("Launchpad service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	// Update project statuses
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()

			// Update launchpad projects
			var upcomingProjects []LaunchpadProject
			s.db.Where("status = ?", "UPCOMING").Find(&upcomingProjects)
			for i := range upcomingProjects {
				if now.After(upcomingProjects[i].StartTime) {
					upcomingProjects[i].Status = "ACTIVE"
					s.db.Save(&upcomingProjects[i])
				}
			}

			var activeProjects []LaunchpadProject
			s.db.Where("status = ?", "ACTIVE").Find(&activeProjects)
			for i := range activeProjects {
				if now.After(activeProjects[i].EndTime) {
					activeProjects[i].Status = "COMPLETED"
					s.db.Save(&activeProjects[i])
				}
			}

			// Update launchpool projects
			var upcomingPools []LaunchpoolProject
			s.db.Where("status = ?", "UPCOMING").Find(&upcomingPools)
			for i := range upcomingPools {
				if now.After(upcomingPools[i].StartTime) {
					upcomingPools[i].Status = "ACTIVE"
					s.db.Save(&upcomingPools[i])
				}
			}

			var activePools []LaunchpoolProject
			s.db.Where("status = ?", "ACTIVE").Find(&activePools)
			for i := range activePools {
				if now.After(activePools[i].EndTime) {
					activePools[i].Status = "COMPLETED"
					s.db.Save(&activePools[i])
				}
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down launchpad service...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
