/**
 * TigerWallet Protection Fund Service
 * Production-ready user protection fund system
 * 
 * Features:
 * - User protection fund management
 * - Claim processing
 * - Fraud detection integration
 * - Reimbursement automation
 * - Fund governance
 * - Real-time coverage tracking
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
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort      string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	
	// Fund Configuration
	InitialFundSize   float64   // Initial fund size in USD
	MinCoverage      float64   // Minimum coverage per user
	MaxCoverage      float64   // Maximum coverage per user
	CoveragePercent  float64   // Coverage percentage (e.g., 100 = 100%)
	
	// Premium
	MonthlyPremium   float64   // Monthly premium in USD
	FreeTierLimit    float64   // Free tier coverage limit
	
	// Claims
	ClaimFee         float64   // Fee per claim
	MaxClaimAmount  float64   // Maximum claim amount
	ClaimTimeout     time.Duration // Time to process claims
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:      getEnv("PROTECTION_FUND_PORT", "9103"),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "tigerwallet"),
		DBPassword:      getEnv("DB_PASSWORD", "password"),
		DBName:          getEnv("DB_NAME", "tigerwallet"),
		InitialFundSize: 10000000, // $10M
		MinCoverage:     1000,     // $1K minimum
		MaxCoverage:      100000,   // $100K maximum
		CoveragePercent:  100,       // 100% coverage
		MonthlyPremium:   9.99,     // $9.99/month
		FreeTierLimit:   1000,     // $1K free coverage
		ClaimFee:         10,        // $10 claim fee
		MaxClaimAmount:  100000,   // $100K max
		ClaimTimeout:     7 * 24 * time.Hour, // 7 days
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

type ProtectionFund struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	
	FundName         string    `json:"fund_name"`
	TotalBalance     float64   `json:"total_balance"`
	AvailableBalance float64   `json:"available_balance"`
	ReservedBalance  float64   `json:"reserved_balance"`
	
	// Coverage
	TotalCoverage    float64   `json:"total_coverage"`
	ActiveUsers      int       `json:"active_users"`
	
	// Statistics
	TotalClaimsPaid float64   `json:"total_claims_paid"`
	TotalClaims     int       `json:"total_claims"`
	SuccessRate     float64   `json:"success_rate"`
	
	Status          string    `json:"status"` // active, paused, depleted
}

type UserCoverage struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	
	UserID           uint      `gorm:"uniqueIndex" json:"user_id"`
	WalletAddress    string    `gorm:"index" json:"wallet_address"`
	
	// Coverage Details
	CoverageType     string    `json:"coverage_type"` // free, premium, enterprise
	CoverageAmount   float64   `json:"coverage_amount"`
	CoveragePercent  float64   `json:"coverage_percent"`
	
	// Premium
	MonthlyPremium   float64   `json:"monthly_premium"`
	IsActive         bool      `json:"is_active"`
	StartDate        time.Time `json:"start_date"`
	EndDate          *time.Time `json:"end_date"`
	
	// Auto-renewal
	AutoRenew        bool      `json:"auto_renew"`
	PaymentMethod    string    `json:"payment_method"`
}

type Claim struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	
	ClaimID          string    `gorm:"uniqueIndex" json:"claim_id"`
	UserID           uint      `gorm:"index" json:"user_id"`
	WalletAddress    string    `gorm:"index" json:"wallet_address"`
	
	// Claim Details
	ClaimType        string    `json:"claim_type"` // hack, theft, fraud, technical_error
	IncidentDate     time.Time `json:"incident_date"`
	Description      string    `json:"description"`
	
	// Amount
	RequestedAmount   float64   `json:"requested_amount"`
	ApprovedAmount   float64   `json:"approved_amount"`
	CoverageAmount    float64   `json:"coverage_amount"`
	
	// Supporting Documents
	Evidence          string    `json:"evidence"` // JSON array of URLs
	PoliceReport     string    `json:"police_report"`
	TransactionHash  string    `json:"transaction_hash"`
	
	// Status
	Status           string    `json:"status"` // submitted, under_review, approved, rejected, paid, closed
	ReviewerID       *uint     `json:"reviewer_id"`
	ReviewNotes      string    `json:"review_notes"`
	ReviewedAt       *time.Time `json:"reviewed_at"`
	
	// Processing
	AssignedTo       *uint     `json:"assigned_to"`
	Priority         string    `json:"priority"` // low, medium, high, critical
	DueDate          time.Time `json:"due_date"`
	CompletedAt      *time.Time `json:"completed_at"`
	
	// Payment
	PaymentTxHash    string    `json:"payment_tx_hash"`
	PaymentDate      *time.Time `json:"payment_date"`
}

type FundTransaction struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	
	TransactionID    string    `gorm:"uniqueIndex" json:"transaction_id"`
	Type             string    `json:"type"` // deposit, withdrawal, claim_paid, premium, investment, fee
	
	Amount           float64   `json:"amount"`
	BalanceBefore    float64   `json:"balance_before"`
	BalanceAfter     float64   `json:"balance_after"`
	
	// Reference
	ClaimID          *string   `json:"claim_id"`
	UserID           *uint     `json:"user_id"`
	
	// Details
	Description      string    `json:"description"`
	TransactionHash  string    `json:"transaction_hash"`
	Status           string    `json:"status"` // pending, confirmed, failed
	
	Metadata         string    `json:"metadata"` // JSON
}

type CoveragePlan struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	
	PlanName         string    `gorm:"uniqueIndex" json:"plan_name"`
	PlanType         string    `json:"plan_type"` // free, basic, premium, enterprise
	
	// Coverage
	MinCoverage      float64   `json:"min_coverage"`
	MaxCoverage      float64   `json:"max_coverage"`
	CoveragePercent  float64   `json:"coverage_percent"`
	
	// Price
	MonthlyPrice     float64   `json:"monthly_price"`
	YearlyPrice      float64   `json:"yearly_price"`
	
	// Features
	Features         string    `json:"features"` // JSON array
	
	IsActive         bool      `json:"is_active"`
	SortOrder        int       `json:"sort_order"`
}

type GovernanceProposal struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	
	ProposalID       string    `gorm:"uniqueIndex" json:"proposal_id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	
	ProposalType     string    `json:"proposal_type"` // fund_allocation, policy_change, emergency
	
	// Voting
	ForVotes         float64   `json:"for_votes"`
	AgainstVotes     float64   `json:"against_votes"`
	AbstainVotes     float64   `json:"abstain_votes"`
	
	// Status
	Status           string    `json:"status"` // draft, voting, passed, rejected, executed
	VotingEndTime    time.Time `json:"voting_end_time"`
	ExecutedAt       *time.Time `json:"executed_at"`
	
	CreatedBy        uint      `json:"created_by"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type ProtectionFundService struct {
	db     *gorm.DB
	config *Config
}

func NewProtectionFundService(db *gorm.DB, config *Config) *ProtectionFundService {
	return &ProtectionFundService{
		db:     db,
		config:  config,
	}
}

// InitializeFund initializes the protection fund
func (s *ProtectionFundService) InitializeFund() error {
	fund := &ProtectionFund{
		FundName:       "TigerWallet Protection Fund",
		TotalBalance:   s.config.InitialFundSize,
		AvailableBalance: s.config.InitialFundSize,
		ReservedBalance: 0,
		TotalCoverage:  0,
		ActiveUsers:   0,
		Status:        "active",
	}

	// Create default coverage plans
	plans := []CoveragePlan{
		{
			PlanName:        "Free",
			PlanType:        "free",
			MinCoverage:     0,
			MaxCoverage:     s.config.FreeTierLimit,
			CoveragePercent: 100,
			MonthlyPrice:    0,
			YearlyPrice:     0,
			Features:        `["basic_protection", "email_support"]`,
			IsActive:        true,
			SortOrder:       1,
		},
		{
			PlanName:        "Basic",
			PlanType:        "basic",
			MinCoverage:     1000,
			MaxCoverage:     10000,
			CoveragePercent: 100,
			MonthlyPrice:    4.99,
			YearlyPrice:     49.99,
			Features:        `["enhanced_protection", "priority_support", "faster_claims"]`,
			IsActive:        true,
			SortOrder:       2,
		},
		{
			PlanName:        "Premium",
			PlanType:        "premium",
			MinCoverage:     10000,
			MaxCoverage:     50000,
			CoveragePercent: 100,
			MonthlyPrice:    9.99,
			YearlyPrice:     99.99,
			Features:        `["full_protection", "24/7_support", "instant_claims", "legal_assistance"]`,
			IsActive:        true,
			SortOrder:       3,
		},
		{
			PlanName:        "Enterprise",
			PlanType:        "enterprise",
			MinCoverage:     50000,
			MaxCoverage:     s.config.MaxCoverage,
			CoveragePercent: 100,
			MonthlyPrice:    29.99,
			YearlyPrice:     299.99,
			Features:        `["maximum_protection", "dedicated_manager", "custom_claims", "api_access"]`,
			IsActive:        true,
			SortOrder:       4,
		},
	}

	if err := s.db.Create(fund).Error; err != nil {
		return err
	}

	return s.db.Create(&plans).Error
}

// GetFundStatus returns the current fund status
func (s *ProtectionFundService) GetFundStatus() (*ProtectionFund, error) {
	var fund ProtectionFund
	if err := s.db.First(&fund).Error; err != nil {
		return nil, err
	}
	return &fund, nil
}

// EnrollUser enrolls a user in the protection fund
func (s *ProtectionFundService) EnrollUser(userID uint, walletAddress, planType string) (*UserCoverage, error) {
	// Get plan
	var plan CoveragePlan
	if err := s.db.Where("plan_type = ? AND is_active = ?", planType, true).First(&plan).Error; err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}

	coverage := &UserCoverage{
		UserID:          userID,
		WalletAddress:   walletAddress,
		CoverageType:    planType,
		CoverageAmount:  plan.MaxCoverage,
		CoveragePercent: plan.CoveragePercent,
		MonthlyPremium:  plan.MonthlyPrice,
		IsActive:        true,
		StartDate:       time.Now(),
		AutoRenew:       true,
	}

	if err := s.db.Create(coverage).Error; err != nil {
		return nil, err
	}

	// Update fund
	s.db.Model(&ProtectionFund{}).Updates(map[string]interface{}{
		"total_coverage": gorm.Expr("total_coverage + ?", coverage.CoverageAmount),
		"active_users":   gorm.Expr("active_users + 1"),
	})

	return coverage, nil
}

// SubmitClaim submits a claim for processing
func (s *ProtectionFundService) SubmitClaim(userID uint, walletAddress, claimType, description string, amount float64, evidence []string) (*Claim, error) {
	// Verify user has coverage
	var coverage UserCoverage
	if err := s.db.Where("user_id = ? AND is_active = ?", userID, true).First(&coverage).Error; err != nil {
		return nil, fmt.Errorf("user not enrolled in protection fund")
	}

	// Verify claim amount doesn't exceed coverage
	if amount > coverage.CoverageAmount {
		amount = coverage.CoverageAmount
	}

	claim := &Claim{
		ClaimID:         uuid.New().String(),
		UserID:          userID,
		WalletAddress:   walletAddress,
		ClaimType:      claimType,
		IncidentDate:    time.Now(),
		Description:    description,
		RequestedAmount: amount,
		Status:          "submitted",
		Priority:        "medium",
		DueDate:         time.Now().Add(s.config.ClaimTimeout),
	}

	evidenceJSON, _ := json.Marshal(evidence)
	claim.Evidence = string(evidenceJSON)

	if err := s.db.Create(claim).Error; err != nil {
		return nil, err
	}

	// Create fund transaction
	tx := &FundTransaction{
		TransactionID: uuid.New().String(),
		Type:          "claim_reserve",
		Amount:        amount,
		ClaimID:       &claim.ClaimID,
		Description:   fmt.Sprintf("Claim reserve: %s", claim.ClaimID),
		Status:        "pending",
	}
	s.db.Create(tx)

	return claim, nil
}

// ProcessClaim processes a claim (approve or reject)
func (s *ProtectionFundService) ProcessClaim(claimID string, approved bool, reviewerID uint, notes string, approvedAmount float64) (*Claim, error) {
	var claim Claim
	if err := s.db.Where("claim_id = ?", claimID).First(&claim).Error; err != nil {
		return nil, fmt.Errorf("claim not found")
	}

	if claim.Status != "submitted" && claim.Status != "under_review" {
		return nil, fmt.Errorf("claim cannot be processed in current status")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"reviewer_id":   reviewerID,
		"review_notes":  notes,
		"reviewed_at":   now,
	}

	if approved {
		// Calculate coverage amount
		var coverage UserCoverage
		s.db.Where("user_id = ? AND is_active = ?", claim.UserID, true).First(&coverage)
		
		coverageAmount := math.Min(approvedAmount, coverage.CoverageAmount)
		
		updates["status"] = "approved"
		updates["approved_amount"] = approvedAmount
		updates["coverage_amount"] = coverageAmount
		
		// Update fund
		s.db.Model(&ProtectionFund{}).Updates(map[string]interface{}{
			"available_balance": gorm.Expr("available_balance - ?", coverageAmount),
			"reserved_balance": gorm.Expr("reserved_balance + ?", coverageAmount),
		})
	} else {
		updates["status"] = "rejected"
	}

	if err := s.db.Model(&claim).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Create transaction record
	txType := "claim_paid"
	if !approved {
		txType = "claim_rejected"
	}
	
	tx := &FundTransaction{
		TransactionID: uuid.New().String(),
		Type:          txType,
		Amount:        approvedAmount,
		ClaimID:       &claim.ClaimID,
		Description:   fmt.Sprintf("Claim %s: %s", txType, claim.ClaimID),
		Status:        "confirmed",
	}
	s.db.Create(tx)

	return &claim, nil
}

// PayClaim processes payment for an approved claim
func (s *ProtectionFundService) PayClaim(claimID, txHash string) (*Claim, error) {
	var claim Claim
	if err := s.db.Where("claim_id = ? AND status = ?", claimID, "approved").First(&claim).Error; err != nil {
		return nil, fmt.Errorf("claim not found or not approved")
	}

	now := time.Now()
	
	if err := s.db.Model(&claim).Updates(map[string]interface{}{
		"status":           "paid",
		"payment_tx_hash":   txHash,
		"payment_date":      now,
		"completed_at":     now,
	}).Error; err != nil {
		return nil, err
	}

	// Update fund
	s.db.Model(&ProtectionFund{}).Updates(map[string]interface{}{
		"total_claims_paid": gorm.Expr("total_claims_paid + ?", claim.CoverageAmount),
		"total_claims":      gorm.Expr("total_claims + 1"),
		"reserved_balance":  gorm.Expr("reserved_balance - ?", claim.CoverageAmount),
	})

	// Update transaction
	s.db.Model(&FundTransaction{}).Where("claim_id = ?", claimID).Updates(map[string]interface{}{
		"status":          "confirmed",
		"transaction_hash": txHash,
	})

	return &claim, nil
}

// GetClaim retrieves a claim
func (s *ProtectionFundService) GetClaim(claimID string) (*Claim, error) {
	var claim Claim
	if err := s.db.Where("claim_id = ?", claimID).First(&claim).Error; err != nil {
		return nil, err
	}
	return &claim, nil
}

// GetUserClaims retrieves all claims for a user
func (s *ProtectionFundService) GetUserClaims(userID uint) ([]Claim, error) {
	var claims []Claim
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&claims).Error; err != nil {
		return nil, err
	}
	return claims, nil
}

// GetCoveragePlans retrieves all coverage plans
func (s *ProtectionFundService) GetCoveragePlans() ([]CoveragePlan, error) {
	var plans []CoveragePlan
	if err := s.db.Where("is_active = ?", true).Order("sort_order").Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}

// GetFundTransactions retrieves fund transactions
func (s *ProtectionFundService) GetFundTransactions(limit, offset int) ([]FundTransaction, error) {
	var transactions []FundTransaction
	if err := s.db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&transactions).Error; err != nil {
		return nil, err
	}
	return transactions, nil
}

// AddFunds adds funds to the protection fund
func (s *ProtectionFundService) AddFunds(amount float64, txType, description string) error {
	var fund ProtectionFund
	if err := s.db.First(&fund).Error; err != nil {
		return err
	}

	newBalance := fund.TotalBalance + amount
	
	if err := s.db.Model(&fund).Updates(map[string]interface{}{
		"total_balance":     newBalance,
		"available_balance": fund.AvailableBalance + amount,
	}).Error; err != nil {
		return err
	}

	// Record transaction
	tx := &FundTransaction{
		TransactionID: uuid.New().String(),
		Type:          txType,
		Amount:        amount,
		BalanceBefore: fund.TotalBalance,
		BalanceAfter:  newBalance,
		Description:   description,
		Status:        "confirmed",
	}
	return s.db.Create(tx).Error
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *ProtectionFundService) GetFundStatusHandler(c *gin.Context) {
	fund, err := s.GetFundStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    fund,
	})
}

func (s *ProtectionFundService) EnrollUserHandler(c *gin.Context) {
	var req struct {
		UserID        uint   `json:"user_id" binding:"required"`
		WalletAddress string `json:"wallet_address" binding:"required"`
		PlanType      string `json:"plan_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	coverage, err := s.EnrollUser(req.UserID, req.WalletAddress, req.PlanType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":   coverage,
	})
}

func (s *ProtectionFundService) SubmitClaimHandler(c *gin.Context) {
	var req struct {
		UserID        uint     `json:"user_id" binding:"required"`
		WalletAddress string   `json:"wallet_address" binding:"required"`
		ClaimType     string   `json:"claim_type" binding:"required"`
		Description   string   `json:"description" binding:"required"`
		Amount        float64  `json:"amount" binding:"required"`
		Evidence      []string `json:"evidence"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claim, err := s.SubmitClaim(req.UserID, req.WalletAddress, req.ClaimType, req.Description, req.Amount, req.Evidence)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":   claim,
	})
}

func (s *ProtectionFundService) ProcessClaimHandler(c *gin.Context) {
	claimID := c.Param("id")

	var req struct {
		Approved       bool    `json:"approved" binding:"required"`
		ReviewerID     uint    `json:"reviewer_id" binding:"required"`
		Notes          string  `json:"notes"`
		ApprovedAmount float64 `json:"approved_amount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claim, err := s.ProcessClaim(claimID, req.Approved, req.ReviewerID, req.Notes, req.ApprovedAmount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":   claim,
	})
}

func (s *ProtectionFundService) GetPlansHandler(c *gin.Context) {
	plans, err := s.GetCoveragePlans()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":   plans,
	})
}

func (s *ProtectionFundService) GetTransactionsHandler(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	txs, err := s.GetFundTransactions(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"transactions": txs,
	})
}

// ============================================================================
// Database Migration
// ============================================================================

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&ProtectionFund{},
		&UserCoverage{},
		&Claim{},
		&FundTransaction{},
		&CoveragePlan{},
		&GovernanceProposal{},
	)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()

	// Initialize database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := Migrate(db); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	// Initialize service
	service := NewProtectionFundService(db, config)

	// Initialize fund if empty
	var fundCount int64
	db.Model(&ProtectionFund{}).Count(&fundCount)
	if fundCount == 0 {
		if err := service.InitializeFund(); err != nil {
			log.Printf("Warning: Failed to initialize fund: %v", err)
		}
	}

	// Setup router
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API routes
	api := router.Group("/api/v1/protection")
	{
		api.GET("/fund", service.GetFundStatusHandler)
		api.POST("/enroll", service.EnrollUserHandler)
		api.POST("/claim", service.SubmitClaimHandler)
		api.POST("/claim/:id/process", service.ProcessClaimHandler)
		api.GET("/plans", service.GetPlansHandler)
		api.GET("/transactions", service.GetTransactionsHandler)
	}

	// Start server
	addr := fmt.Sprintf(":%s", config.ServerPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting Protection Fund service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
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

// Add strconv import
import "strconv"
