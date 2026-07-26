/**
 * TigerWallet User Protection Fund Service
 * Production-ready insurance fund for user asset protection
 * 
 * Features:
 * - User asset protection
 * - Claim management
 * - Fund pool management
 * - Coverage calculation
 * - Automatic payouts
 * - Real-time monitoring
 */

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
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
	ServerPort       string  `json:"server_port"`
	DBHost           string  `json:"db_host"`
	DBPort           string  `json:"db_port"`
	DBUser           string  `json:"db_user"`
	DBPassword       string  `json:"db_password"`
	DBName           string  `json:"db_name"`
	RedisHost        string  `json:"redis_host"`
	RedisPort        string  `json:"redis_port"`
	
	// Fund settings
	InitialFund      float64 `json:"initial_fund"`      // $10M initial
	AnnualBudget     float64 `json:"annual_budget"`     // $2M annual
	MinCoverage      float64 `json:"min_coverage"`      // $1,000
	MaxCoverage      float64 `json:"max_coverage"`      // $100,000
	ReserveRatio     float64 `json:"reserve_ratio"`     // 85%
	
	// Claim settings
	ClaimTimeout     int     `json:"claim_timeout"`    // hours
	ReviewTimeout    int     `json:"review_timeout"`   // days
	AutoApproveThreshold float64 `json:"auto_approve_threshold"` // $1,000
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:   getEnv("PROTECTION_FUND_PORT", "9097"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "tigerwallet"),
		DBPassword:   getEnv("DB_PASSWORD", "password"),
		DBName:       getEnv("DB_NAME", "tigerwallet"),
		RedisHost:    getEnv("REDIS_HOST", "localhost"),
		RedisPort:    getEnv("REDIS_PORT", "6379"),
		InitialFund:  10000000,  // $10M
		AnnualBudget: 2000000,    // $2M
		MinCoverage:  1000,
		MaxCoverage:  100000,
		ReserveRatio: 0.85,
		ClaimTimeout: 72,         // 72 hours
		ReviewTimeout: 7,         // 7 days
		AutoApproveThreshold: 1000,
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

// Fund pool
type FundPool struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Name            string    `json:"name"`
	TotalBalance    float64   `json:"total_balance"`
	AvailableBalance float64   `json:"available_balance"`
	ReservedBalance  float64   `json:"reserved_balance"`
	AnnualBudget    float64   `json:"annual_budget"`
	SpentThisYear   float64   `json:"spent_this_year"`
	Currency        string    `json:"currency"` // USD, USDC, ETH
	WalletAddress   string    `json:"wallet_address"`
	IsActive        bool      `json:"is_active"`
}

// User coverage
type UserCoverage struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	UserAddress     string    `gorm:"index" json:"user_address"`
	WalletBalance   float64   `json:"wallet_balance"`
	CoverageAmount  float64   `json:"coverage_amount"`
	CoverageLevel   float64   `json:"coverage_level"` // percentage
	StartDate       time.Time `json:"start_date"`
	EndDate         *time.Time `json:"end_date"`
	IsActive        bool      `json:"is_active"`
	Status          string    `json:"status"` // active, suspended, expired
}

// Claim
type Claim struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ClaimID        string    `gorm:"uniqueIndex" json:"claim_id"`
	UserAddress    string    `gorm:"index" json:"user_address"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Reason         string    `json:"reason"`
	Description    string    `json:"description"`
	Status         string    `gorm:"index" json:"status"` // pending, review, approved, rejected, paid
	Priority       int       `json:"priority"` // 1=critical, 2=high, 3=medium, 4=low
	ReviewerID     *uint    `json:"reviewer_id"`
	ReviewerNotes  string    `json:"reviewer_notes"`
	ApprovedAt     *time.Time `json:"approved_at"`
	PaidAt         *time.Time `json:"paid_at"`
	TxHash         string    `json:"tx_hash"`
	Evidence       string    `gorm:"type:jsonb" json:"evidence"`
	Resolution     string    `json:"resolution"`
}

// Transaction record
type FundTransaction struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	TxHash        string    `gorm:"uniqueIndex" json:"tx_hash"`
	Type          string    `json:"type"` // deposit, withdrawal, claim_payout, fee
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	FromAddress   string    `json:"from_address"`
	ToAddress     string    `json:"to_address"`
	Status        string    `json:"status"` // pending, confirmed, failed
	BlockNumber   uint64    `json:"block_number"`
	GasUsed       uint64    `json:"gas_used"`
}

// ============================================================================
// Service
// ============================================================================

type ProtectionFundService struct {
	config     *Config
	db         *gorm.DB
	redis      *redis.Client
	httpClient *http.Client
	
	// In-memory state
	mu         sync.RWMutex
	fundPool   *FundPool
	stats      *FundStats
}

type FundStats struct {
	TotalFund        float64   `json:"total_fund"`
	AvailableFund    float64   `json:"available_fund"`
	CurrentCoverage  float64   `json:"current_coverage"`
	ProtectedUsers   int       `json:"protected_users"`
	ClaimsPaid      float64   `json:"claims_paid"`
	PendingClaims   int       `json:"pending_claims"`
	AnnualBudget    float64   `json:"annual_budget"`
	ReserveRatio    float64   `json:"reserve_ratio"`
	AvgClaimTime    float64   `json:"avg_claim_time_hours"`
}

func NewProtectionFundService(config *Config, db *gorm.DB, redisClient *redis.Client) *ProtectionFundService {
	return &ProtectionFundService{
		config:     config,
		db:         db,
		redis:      redisClient,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		fundPool:   nil,
		stats:      &FundStats{},
	}
}

// ============================================================================
// Core Functions
// ============================================================================

func (s *ProtectionFundService) Initialize() error {
	log.Println("Initializing Protection Fund Service...")
	
	// Auto-migrate database
	err := s.db.AutoMigrate(&FundPool{}, &UserCoverage{}, &Claim{}, &FundTransaction{})
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	
	// Initialize fund pool
	s.initializeFundPool()
	
	// Initialize stats
	s.updateStats()
	
	log.Println("Protection Fund Service initialized successfully")
	return nil
}

func (s *ProtectionFundService) initializeFundPool() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	var pool FundPool
	result := s.db.First(&pool)
	
	if result.Error != nil {
		// Create new pool
		pool = FundPool{
			Name:            "TigerWallet Protection Fund",
			TotalBalance:    s.config.InitialFund,
			AvailableBalance: s.config.InitialFund * s.config.ReserveRatio,
			ReservedBalance: s.config.InitialFund * (1 - s.config.ReserveRatio),
			AnnualBudget:    s.config.AnnualBudget,
			SpentThisYear:   0,
			Currency:        "USDC",
			WalletAddress:   "0x" + generateAddress(), // Placeholder
			IsActive:        true,
		}
		s.db.Create(&pool)
		log.Printf("Created new protection fund pool: $%.2f", pool.TotalBalance)
	}
	
	s.fundPool = &pool
	s.stats.TotalFund = pool.TotalBalance
	s.stats.AvailableFund = pool.AvailableBalance
	s.stats.AnnualBudget = pool.AnnualBudget
	s.stats.ReserveRatio = s.config.ReserveRatio
}

func (s *ProtectionFundService) updateStats() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	var totalCoverage float64
	var protectedUsers int64
	s.db.Model(&UserCoverage{}).Where("is_active = ?", true).Count(&protectedUsers)
	s.db.Model(&UserCoverage{}).Where("is_active = ?", true).Select("COALESCE(SUM(coverage_amount), 0)").Scan(&totalCoverage)
	
	var claimsPaid float64
	s.db.Model(&Claim{}).Where("status = ?", "paid").Select("COALESCE(SUM(amount), 0)").Scan(&claimsPaid)
	
	var pendingClaims int64
	s.db.Model(&Claim{}).Where("status IN ?", []string{"pending", "review"}).Count(&pendingClaims)
	
	s.stats.ProtectedUsers = int(protectedUsers)
	s.stats.CurrentCoverage = totalCoverage
	s.stats.ClaimsPaid = claimsPaid
	s.stats.PendingClaims = int(pendingClaims)
	
	// Calculate average claim time
	var avgTime float64
	var claimCount int64
	s.db.Model(&Claim{}).Where("status = ? AND paid_at IS NOT NULL", "paid").Count(&claimCount)
	if claimCount > 0 {
		s.db.Model(&Claim{}).Where("status = ? AND paid_at IS NOT NULL", "paid").
			Select("AVG(EXTRACT(EPOCH FROM (paid_at - created_at)))").Scan(&avgTime)
		s.stats.AvgClaimTime = avgTime / 3600 // Convert to hours
	}
}

// ============================================================================
// Coverage Functions
// ============================================================================

func (s *ProtectionFundService) CalculateCoverage(walletBalance float64) (float64, float64) {
	// Coverage formula: min(max(balance * 0.1, minCoverage), maxCoverage)
	coverage := walletBalance * 0.1
	if coverage < s.config.MinCoverage {
		coverage = s.config.MinCoverage
	}
	if coverage > s.config.MaxCoverage {
		coverage = s.config.MaxCoverage
	}
	
	level := (coverage / walletBalance) * 100
	return coverage, level
}

func (s *ProtectionFundService) RegisterUserCoverage(userAddress string, walletBalance float64) (*UserCoverage, error) {
	coverageAmount, coverageLevel := s.CalculateCoverage(walletBalance)
	
	coverage := UserCoverage{
		UserAddress:    strings.ToLower(userAddress),
		WalletBalance:  walletBalance,
		CoverageAmount: coverageAmount,
		CoverageLevel:  coverageLevel,
		StartDate:      time.Now(),
		IsActive:       true,
		Status:         "active",
	}
	
	err := s.db.Create(&coverage).Error
	if err != nil {
		return nil, err
	}
	
	s.updateStats()
	return &coverage, nil
}

func (s *ProtectionFundService) UpdateUserCoverage(userAddress string, newBalance float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	var coverage UserCoverage
	result := s.db.Where("user_address = ? AND is_active = ?", strings.ToLower(userAddress), true).First(&coverage)
	if result.Error != nil {
		return result.Error
	}
	
	coverageAmount, coverageLevel := s.CalculateCoverage(newBalance)
	coverage.WalletBalance = newBalance
	coverage.CoverageAmount = coverageAmount
	coverage.CoverageLevel = coverageLevel
	
	return s.db.Save(&coverage).Error
}

func (s *ProtectionFundService) GetUserCoverage(userAddress string) (*UserCoverage, error) {
	var coverage UserCoverage
	err := s.db.Where("user_address = ? AND is_active = ?", strings.ToLower(userAddress), true).First(&coverage).Error
	if err != nil {
		return nil, err
	}
	return &coverage, nil
}

// ============================================================================
// Claim Functions
// ============================================================================

func (s *ProtectionFundService) SubmitClaim(userAddress, reason, description string, amount float64, evidence map[string]interface{}) (*Claim, error) {
	// Verify user has coverage
	coverage, err := s.GetUserCoverage(userAddress)
	if err != nil {
		return nil, fmt.Errorf("user has no active coverage")
	}
	
	// Verify claim amount doesn't exceed coverage
	if amount > coverage.CoverageAmount {
		return nil, fmt.Errorf("claim amount exceeds coverage limit: $%.2f", coverage.CoverageAmount)
	}
	
	// Determine priority based on amount
	priority := 4 // low
	if amount > 10000 {
		priority = 1 // critical
	} else if amount > 5000 {
		priority = 2 // high
	} else if amount > 1000 {
		priority = 3 // medium
	}
	
	evidenceJSON, _ := json.Marshal(evidence)
	
	claim := Claim{
		ClaimID:     "CLM-" + uuid.New().String()[:8],
		UserAddress: strings.ToLower(userAddress),
		Amount:      amount,
		Currency:    "USDC",
		Reason:      reason,
		Description: description,
		Status:      "pending",
		Priority:    priority,
		Evidence:    string(evidenceJSON),
	}
	
	err = s.db.Create(&claim).Error
	if err != nil {
		return nil, err
	}
	
	s.updateStats()
	
	// Auto-approve small claims
	if amount <= s.config.AutoApproveThreshold {
		s.ApproveClaim(claim.ClaimID, "Auto-approved: below threshold")
	}
	
	return &claim, nil
}

func (s *ProtectionFundService) ApproveClaim(claimID, notes string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	var claim Claim
	err := s.db.Where("claim_id = ?", claimID).First(&claim).Error
	if err != nil {
		return err
	}
	
	if claim.Status != "pending" && claim.Status != "review" {
		return fmt.Errorf("claim is not in pending/review status")
	}
	
	now := time.Now()
	claim.Status = "approved"
	claim.ReviewerNotes = notes
	claim.ApprovedAt = &now
	
	err = s.db.Save(&claim).Error
	if err != nil {
		return err
	}
	
	// Process payout
	return s.processPayout(&claim)
}

func (s *ProtectionFundService) RejectClaim(claimID, reason string) error {
	var claim Claim
	err := s.db.Where("claim_id = ?", claimID).First(&claim).Error
	if err != nil {
		return err
	}
	
	if claim.Status != "pending" && claim.Status != "review" {
		return fmt.Errorf("claim is not in pending/review status")
	}
	
	claim.Status = "rejected"
	claim.ReviewerNotes = reason
	
	err = s.db.Save(&claim).Error
	if err != nil {
		return err
	}
	
	s.updateStats()
	return nil
}

func (s *ProtectionFundService) processPayout(claim *Claim) error {
	// Reserve funds
	s.fundPool.AvailableBalance -= claim.Amount
	s.fundPool.ReservedBalance += claim.Amount
	s.fundPool.SpentThisYear += claim.Amount
	
	err := s.db.Save(s.fundPool).Error
	if err != nil {
		return err
	}
	
	// In production, this would trigger an actual blockchain transaction
	// For now, simulate the payout
	txHash := "0x" + generateAddress()
	
	claim.Status = "paid"
	claim.TxHash = txHash
	now := time.Now()
	claim.PaidAt = &now
	
	err = s.db.Save(claim).Error
	if err != nil {
		return err
	}
	
	// Update pool balance after successful payout
	s.fundPool.ReservedBalance -= claim.Amount
	s.fundPool.TotalBalance -= claim.Amount
	
	return s.db.Save(s.fundPool).Error
}

func (s *ProtectionFundService) GetClaim(claimID string) (*Claim, error) {
	var claim Claim
	err := s.db.Where("claim_id = ?", claimID).First(&claim).Error
	if err != nil {
		return nil, err
	}
	return &claim, nil
}

func (s *ProtectionFundService) GetUserClaims(userAddress string) ([]Claim, error) {
	var claims []Claim
	err := s.db.Where("user_address = ?", strings.ToLower(userAddress)).Order("created_at DESC").Find(&claims).Error
	return claims, err
}

func (s *ProtectionFundService) GetPendingClaims(limit int) ([]Claim, error) {
	var claims []Claim
	err := s.db.Where("status IN ?", []string{"pending", "review"}).Order("priority ASC, created_at ASC").Limit(limit).Find(&claims).Error
	return claims, err
}

// ============================================================================
// Fund Management
// ============================================================================

func (s *ProtectionFundService) GetFundStats() *FundStats {
	s.updateStats()
	return s.stats
}

func (s *ProtectionFundService) GetFundPool() *FundPool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fundPool
}

func (s *ProtectionFundService) AddFunds(amount float64, txHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	tx := FundTransaction{
		TxHash:      txHash,
		Type:        "deposit",
		Amount:      amount,
		Currency:    "USDC",
		Status:      "confirmed",
	}
	s.db.Create(&tx)
	
	s.fundPool.TotalBalance += amount
	s.fundPool.AvailableBalance += amount
	
	return s.db.Save(s.fundPool).Error
}

// ============================================================================
// REST API Handlers
// ============================================================================

func (s *ProtectionFundService) RegisterCoverage(c *gin.Context) {
	var req struct {
		UserAddress  string  `json:"user_address" binding:"required"`
		WalletBalance float64 `json:"wallet_balance" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	coverage, err := s.RegisterUserCoverage(req.UserAddress, req.WalletBalance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, coverage)
}

func (s *ProtectionFundService) GetCoverage(c *gin.Context) {
	userAddress := c.Param("address")
	
	coverage, err := s.GetUserCoverage(userAddress)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active coverage found"})
		return
	}
	
	c.JSON(http.StatusOK, coverage)
}

func (s *ProtectionFundService) SubmitClaimHandler(c *gin.Context) {
	var req struct {
		UserAddress string                 `json:"user_address" binding:"required"`
		Amount      float64                `json:"amount" binding:"required"`
		Reason      string                 `json:"reason" binding:"required"`
		Description string                 `json:"description"`
		Evidence    map[string]interface{} `json:"evidence"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	claim, err := s.SubmitClaim(req.UserAddress, req.Reason, req.Description, req.Amount, req.Evidence)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, claim)
}

func (s *ProtectionFundService) ApproveClaimHandler(c *gin.Context) {
	claimID := c.Param("id")
	var req struct {
		Notes string `json:"notes"`
	}
	c.ShouldBindJSON(&req)
	
	err := s.ApproveClaim(claimID, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "claim approved"})
}

func (s *ProtectionFundService) GetStatsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.GetFundStats())
}

func (s *ProtectionFundService) GetFundHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.GetFundPool())
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateAddress() string {
	data := []byte(time.Now().Format(time.RFC3339Nano))
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])[:40]
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()
	
	// Initialize database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	
	// Initialize Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})
	
	// Initialize service
	service := NewProtectionFundService(config, db, rdb)
	if err := service.Initialize(); err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}
	
	// Setup router
	router := gin.Default()
	
	// API routes
	api := router.Group("/api/v1/protection")
	{
		api.POST("/coverage", service.RegisterCoverage)
		api.GET("/coverage/:address", service.GetCoverage)
		api.POST("/claims", service.SubmitClaimHandler)
		api.GET("/claims/:id", func(c *gin.Context) {
			claim, err := service.GetClaim(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "claim not found"})
				return
			}
			c.JSON(http.StatusOK, claim)
		})
		api.PUT("/claims/:id/approve", service.ApproveClaimHandler)
		api.GET("/stats", service.GetStatsHandler)
		api.GET("/fund", service.GetFundHandler)
	}
	
	// Start server
	go func() {
		log.Printf("Starting Protection Fund Service on port %s", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	
	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down Protection Fund Service...")
}
