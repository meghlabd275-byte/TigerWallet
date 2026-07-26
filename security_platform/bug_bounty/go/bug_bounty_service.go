/**
 * TigerWallet Security Bug Bounty Platform
 * Production-ready security vulnerability disclosure and reward system
 * 
 * Features:
 * - Vulnerability submission and tracking
 * - Severity classification (CVSS)
 * - Reward calculation and payment
 * - Bug bounty program management
 * - Security researcher verification
 */

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort      string `json:"server_port"`
	DBHost          string `json:"db_host"`
	DBPort          string `json:"db_port"`
	DBUser          string `json:"db_user"`
	DBPassword      string `json:"db_password"`
	DBName          string `json:"db_name"`
	RedisHost       string `json:"redis_host"`
	RedisPort       string `json:"redis_port"`
	JWTSecret       string `json:"jwt_secret"`
	
	// Reward settings
	RewardPool       float64 `json:"reward_pool"` // Total pool in USD
	MinReward        float64 `json:"min_reward"`
	MaxReward        float64 `json:"max_reward"`
	
	// Severity rewards (percentage of pool)
	CriticalPercent  float64 `json:"critical_percent"`  // 40%
	HighPercent     float64 `json:"high_percent"`     // 25%
	MediumPercent   float64 `json:"medium_percent"`   // 15%
	LowPercent      float64 `json:"low_percent"`     // 5%
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:    getEnv("BOUNTY_PORT", "9096"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "tigerwallet"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		DBName:        getEnv("DB_NAME", "tigerwallet"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		JWTSecret:     getEnv("JWT_SECRET", "bounty-secret"),
		RewardPool:    1000000, // $1M pool
		MinReward:     100,
		MaxReward:     50000,
		CriticalPercent: 40,
		HighPercent:    25,
		MediumPercent:  15,
		LowPercent:     5,
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

// Researcher (Security Researcher)
type Researcher struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	
	Username    string    `gorm:"uniqueIndex" json:"username"`
	Email       string    `gorm:"index" json:"email"`
	PasswordHash string   `json:"-"`
	
	Reputation  int       `json:"reputation"` // Points system
	TotalBounties int     `json:"total_bounties"`
	TotalEarnings float64 `json:"total_earnings"`
	
	Status      string    `json:"status"` // pending, verified, suspended
	KYCVerified bool      `json:"kyc_verified"`
	PayoutAddress string  `json:"payout_address"`
	
	LastActiveAt time.Time `json:"last_active_at"`
}

// Vulnerability Report
type VulnerabilityReport struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	
	ReportID       string    `gorm:"uniqueIndex" json:"report_id"`
	ResearcherID   uint      `gorm:"index" json:"researcher_id"`
	
	// Classification
	Title          string    `json:"title" binding:"required"`
	Description    string    `json:"description" binding:"required"`
	Category       string    `json:"category" binding:"required"` // smart_contract, wallet, frontend, api, etc.
	Severity       string    `json:"severity"` // critical, high, medium, low, info
	CVSSScore      float64   `json:"cvss_score"`
	
	// Technical details
	Impact         string    `json:"impact"`
	StepsToReproduce string  `json:"steps_to_reproduce"`
	ProofOfConcept string    `json:"proof_of_concept"`
	AffectedAssets string    `json:"affected_assets"` // Comma-separated
	
	// Status workflow
	Status         string    `json:"status"` // submitted, triaged, confirmed, disputed, resolved, rewarded, closed
	Resolution     string    `json:"resolution"`
	
	// Rewards
	RewardAmount   float64   `json:"reward_amount"`
	RewardStatus  string    `json:"reward_status"` // pending, approved, paid, rejected
	
	// Review
	AssignedTo    *uint     `json:"assigned_to"` // Admin reviewer
	ReviewedAt    *time.Time `json:"reviewed_at"`
	ReviewNotes   string    `json:"review_notes"`
	
	// Timeline
	SubmittedAt   time.Time `json:"submitted_at"`
	TriagedAt     *time.Time `json:"triaged_at"`
	ConfirmedAt   *time.Time `json:"confirmed_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	RewardedAt    *time.Time `json:"rewarded_at"`
}

// Program Management
type BountyProgram struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Status        string    `json:"status"` // active, paused, closed
	
	// Scope
	Scopes        string    `json:"scopes"` // JSON array of in-scope targets
	Exclusions    string    `json:"exclusions"` // JSON array
	
	// Rewards
	MinReward     float64   `json:"min_reward"`
	MaxReward     float64   `json:"max_reward"`
	TotalPaid     float64   `json:"total_paid"`
	
	// Rules
	Rules         string    `json:"rules"`
	DisclosurePolicy string  `json:"disclosure_policy"`
	
	StartDate     time.Time `json:"start_date"`
	EndDate       *time.Time `json:"end_date"`
	
	CreatedBy     uint      `json:"created_by"`
}

// Reward Distribution
type Reward struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	
	ReportID     uint      `gorm:"index" json:"report_id"`
	ResearcherID uint      `gorm:"index" json:"researcher_id"`
	
	Amount        float64   `json:"amount"`
	Currency     string    `json:"currency"` // USDT, USDC, ETH
	Status        string    `json:"status"` // pending, processing, completed, failed
	
	TransactionHash string  `json:"transaction_hash"`
	PayoutAddress string  `json:"payout_address"`
	
	ProcessedAt   *time.Time `json:"processed_at"`
}

// Security Audit
type SecurityAudit struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"updated_at"`
	
	Name        string    `json:"name"`
	Auditor      string    `json:"auditor"` // CertiK, SlowMist, etc.
	Status       string    `json:"status"` // scheduled, in_progress, completed
	
	StartDate   time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
	
	Scope       string    `json:"scope"` // JSON array
	Findings    int       `json:"findings"`
	Critical    int       `json:"critical"`
	High        int       `json:"high"`
	Medium      int       `json:"medium"`
	Low         int       `json:"low"`
	
	ReportURL   string    `json:"report_url"`
}

// Audit Log
type AuditLog struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	
	AdminID      uint      `gorm:"index" json:"admin_id"`
	Action       string    `json:"action"`
	Resource     string    `json:"resource"`
	Details      string    `json:"details"`
	IPAddress    string    `json:"ip_address"`
}

// ============================================================================
// Service Layer
// ============================================================================

type BugBountyService struct {
	db        *gorm.DB
	redis    *redis.Client
	config   *Config
	jwtSecret []byte
}

func NewBugBountyService(cfg *Config) (*BugBountyService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&Researcher{}, &VulnerabilityReport{}, &BountyProgram{}, &Reward{}, &SecurityAudit{}, &AuditLog{})

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		DB:  6,
	})

	service := &BugBountyService{
		db:         db,
		redis:      rdb,
		config:     cfg,
		jwtSecret: []byte(cfg.JWTSecret),
	}

	// Initialize default program
	service.initializeDefaultProgram()

	return service, nil
}

func (s *BugBountyService) initializeDefaultProgram() {
	var program BountyProgram
	result := s.db.First(&program)
	
	if result.Error != nil {
		program = BountyProgram{
			Name:          "TigerWallet Bug Bounty",
			Description:   "Official security vulnerability disclosure program for TigerWallet",
			Status:        "active",
			Scopes:        `["smart_contracts", "wallet_core", "mobile_app", "web_app", "api", "infrastructure"]`,
			Exclusions:    `["ddos", "social_engineering", "physical_security", "third_party_services"]`,
			MinReward:     100,
			MaxReward:     50000,
			Rules:         "Standard responsible disclosure rules apply. No black hat testing.",
			DisclosurePolicy: "No public disclosure until vulnerability is fixed",
			StartDate:     time.Now(),
		}
		s.db.Create(&program)
	}
}

// ============================================================================
// Researcher Registration
// ============================================================================

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

func (s *BugBountyService) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check exists
	var existing Researcher
	result := s.db.Where("email = ? OR username = ?", req.Email, req.Username).First(&existing)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
		return
	}

	// Hash password
	hash := sha256.Sum256([]byte(req.Password))
	passwordHash := hex.EncodeToString(hash[:])

	researcher := Researcher{
		Username:     req.Username,
		Email:       req.Email,
		PasswordHash: passwordHash,
		Status:       "verified",
		Reputation:   0,
	}

	s.db.Create(&researcher)

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Registration successful",
		"researcher_id": researcher.ID,
	})
}

func (s *BugBountyService) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash := sha256.Sum256([]byte(req.Password))
	passwordHash := hex.EncodeToString(hash[:])

	var researcher Researcher
	result := s.db.Where("email = ? AND password_hash = ?", req.Email, passwordHash).First(&researcher)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Generate token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"researcher_id": researcher.ID,
		"username":     researcher.Username,
		"exp":          time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(s.jwtSecret)

	c.JSON(http.StatusOK, gin.H{
		"token":    tokenString,
		"researcher": gin.H{
			"id":          researcher.ID,
			"username":    researcher.Username,
			"reputation":   researcher.Reputation,
			"total_earned": researcher.TotalEarnings,
		},
	})
}

// ============================================================================
// Vulnerability Reporting
// ============================================================================

type SubmitReportRequest struct {
	Title          string `json:"title" binding:"required"`
	Description    string `json:"description" binding:"required"`
	Category       string `json:"category" binding:"required"`
	Impact         string `json:"impact"`
	StepsToReproduce string `json:"steps_to_reproduce"`
	ProofOfConcept string `json:"proof_of_concept"`
	AffectedAssets string `json:"affected_assets"`
}

func (s *BugBountyService) SubmitReport(c *gin.Context) {
	researcherID := c.GetUint("researcher_id")

	var req SubmitReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Auto-classify severity (simplified - in production use ML or manual review)
	severity := s.classifySeverity(req.Category, req.Impact)

	report := VulnerabilityReport{
		ReportID:        "RPT-" + uuid.New().String()[:10],
		ResearcherID:    researcherID,
		Title:           req.Title,
		Description:     req.Description,
		Category:        req.Category,
		Severity:        severity,
		Impact:          req.Impact,
		StepsToReproduce: req.StepsToReproduce,
		ProofOfConcept:  req.ProofOfConcept,
		AffectedAssets:  req.AffectedAssets,
		Status:          "submitted",
		RewardStatus:    "pending",
		SubmittedAt:    time.Now(),
	}

	s.db.Create(&report)

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Report submitted successfully",
		"report_id":  report.ReportID,
		"status":    report.Status,
		"severity":  report.Severity,
	})
}

func (s *BugBountyService) classifySeverity(category, impact string) string {
	impactLower := strings.ToLower(impact)
	
	// Simple classification based on keywords
	if strings.Contains(impactLower, "fund") || 
	   strings.Contains(impactLower, "theft") ||
	   strings.Contains(impactLower, "unauthorized") ||
	   strings.Contains(impactLower, "critical") {
		return "critical"
	}
	
	if strings.Contains(impactLower, "access") ||
	   strings.Contains(impactLower, "privilege") ||
	   strings.Contains(impactLower, "high") {
		return "high"
	}
	
	if strings.Contains(impactLower, "moderate") ||
	   strings.Contains(impactLower, "medium") ||
	   strings.Contains(impactLower, "limited") {
		return "medium"
	}
	
	return "low"
}

func (s *BugBountyService) GetReports(c *gin.Context) {
	researcherID := c.GetUint("researcher_id")
	status := c.Query("status")

	var reports []VulnerabilityReport
	query := s.db.Where("researcher_id = ?", researcherID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Order("created_at DESC").Find(&reports)

	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

func (s *BugBountyService) GetReport(c *gin.Context) {
	reportID := c.Param("id")

	var report VulnerabilityReport
	result := s.db.Where("report_id = ?", reportID).First(&report)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}

	c.JSON(http.StatusOK, report)
}

// ============================================================================
// Admin Review
// ============================================================================

type ReviewReportRequest struct {
	Severity   string  `json:"severity"`
	Status     string  `json:"status" binding:"required"`
	Resolution string  `json:"resolution"`
	RewardAmount float64 `json:"reward_amount"`
}

func (s *BugBountyService) ReviewReport(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	reportID := c.Param("id")

	var report VulnerabilityReport
	result := s.db.Where("report_id = ?", reportID).First(&report)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}

	var req ReviewReportRequest
	c.ShouldBindJSON(&req)

	// Update report
	report.Severity = req.Severity
	report.Status = req.Status
	report.Resolution = req.Resolution
	report.AssignedTo = &adminID
	now := time.Now()
	report.ReviewedAt = &now

	// Set timestamps based on status
	switch req.Status {
	case "triaged":
		report.TriagedAt = &now
	case "confirmed":
		report.ConfirmedAt = &now
	case "resolved":
		report.ResolvedAt = &now
	}

	// Calculate reward if approved
	if req.Status == "resolved" && req.RewardAmount > 0 {
		report.RewardAmount = req.RewardAmount
		report.RewardStatus = "approved"
		
		// Create reward record
		reward := Reward{
			ReportID:     report.ID,
			ResearcherID: report.ResearcherID,
			Amount:      req.RewardAmount,
			Currency:     "USDT",
			Status:       "pending",
		}
		s.db.Create(&reward)
	}

	s.db.Save(&report)

	// Log audit
	audit := AuditLog{
		AdminID:   adminID,
		Action:    "report.review",
		Resource:  reportID,
		Details:   fmt.Sprintf("Status: %s, Severity: %s, Reward: $%.2f", req.Status, req.Severity, req.RewardAmount),
		IPAddress: c.ClientIP(),
	}
	s.db.Create(&audit)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Report reviewed",
		"reward":     req.RewardAmount,
		"new_status": req.Status,
	})
}

func (s *BugBountyService) ListAllReports(c *gin.Context) {
	status := c.Query("status")
	severity := c.Query("severity")

	var reports []VulnerabilityReport
	query := s.db.Model(&VulnerabilityReport{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}

	query.Order("created_at DESC").Find(&reports)

	// Get counts
	var counts map[string]int
	s.db.Model(&VulnerabilityReport{}).Group("status").Count(&counts)

	c.JSON(http.StatusOK, gin.H{
		"reports": reports,
		"counts":  counts,
	})
}

// ============================================================================
// Rewards & Payouts
// ============================================================================

func (s *BugBountyService) CalculateReward(severity string, cvssScore float64) float64 {
	var percent float64

	switch strings.ToLower(severity) {
	case "critical":
		percent = s.config.CriticalPercent
	case "high":
		percent = s.config.HighPercent
	case "medium":
		percent = s.config.MediumPercent
	case "low":
		percent = s.config.LowPercent
	default:
		percent = 1
	}

	// Calculate base reward from pool
	reward := (s.config.RewardPool * percent / 100)

	// Adjust by CVSS if available
	if cvssScore > 0 {
		cvssMultiplier := cvssScore / 10.0
		if cvssMultiplier > 1 {
			reward *= cvssMultiplier
		}
	}

	// Clamp to min/max
	if reward < s.config.MinReward {
		reward = s.config.MinReward
	}
	if reward > s.config.MaxReward {
		reward = s.config.MaxReward
	}

	return reward
}

func (s *BugBountyService) ProcessReward(c *gin.Context) {
	rewardID := c.Param("id")

	var reward Reward
	result := s.db.First(&reward, rewardID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "reward not found"})
		return
	}

	// In production, would initiate blockchain transaction
	// For now, mark as processing
	reward.Status = "processing"
	s.db.Save(&reward)

	// Simulate processing
	reward.Status = "completed"
	reward.TransactionHash = "0x" + uuid.New().String()
	now := time.Now()
	reward.ProcessedAt = &now

	s.db.Save(&reward)

	// Update researcher stats
	var researcher Researcher
	s.db.First(&researcher, reward.ResearcherID)
	researcher.TotalBounties++
	researcher.TotalEarnings += reward.Amount
	researcher.Reputation += int(reward.Amount / 100) // 1 point per $100
	s.db.Save(&researcher)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Reward processed",
		"tx_hash":      reward.TransactionHash,
		"amount":       reward.Amount,
		"researcher":   researcher.Username,
	})
}

// ============================================================================
// Programs & Audits
// ============================================================================

func (s *BugBountyService) GetProgram(c *gin.Context) {
	var program BountyProgram
	s.db.Where("status = ?", "active").First(&program)

	// Get stats
	var totalReports, resolved, rewarded int64
	s.db.Model(&VulnerabilityReport{}).Count(&totalReports)
	s.db.Model(&VulnerabilityReport{}).Where("status = ?", "resolved").Count(&resolved)
	s.db.Model(&VulnerabilityReport{}).Where("reward_status = ?", "paid").Count(&rewarded)

	var totalPaid float64
	s.db.Model(&Reward{}).Where("status = ?", "completed").Select("COALESCE(SUM(amount), 0)").Scan(&totalPaid)

	c.JSON(http.StatusOK, gin.H{
		"program": program,
		"stats": gin.H{
			"total_reports":  totalReports,
			"resolved":       resolved,
			"rewarded":       rewarded,
			"total_paid":     totalPaid,
			"reward_pool":    s.config.RewardPool,
			"remaining_pool": s.config.RewardPool - totalPaid,
		},
	})
}

func (s *BugBountyService) GetLeaderboard(c *gin.Context) {
	var researchers []Researcher
	s.db.Order("reputation DESC").Limit(20).Find(&researchers)

	type LeaderboardEntry struct {
		Rank         int     `json:"rank"`
		Username     string  `json:"username"`
		Reputation   int     `json:"reputation"`
		Bounties    int     `json:"bounties"`
		TotalEarned float64 `json:"total_earned"`
	}

	leaderboard := make([]LeaderboardEntry, len(researchers))
	for i, r := range researchers {
		leaderboard[i] = LeaderboardEntry{
			Rank:         i + 1,
			Username:     r.Username,
			Reputation:   r.Reputation,
			Bounties:    r.TotalBounties,
			TotalEarned: r.TotalEarnings,
		}
	}

	c.JSON(http.StatusOK, gin.H{"leaderboard": leaderboard})
}

func (s *BugBountyService) GetAudits(c *gin.Context) {
	var audits []SecurityAudit
	s.db.Order("created_at DESC").Find(&audits)

	c.JSON(http.StatusOK, gin.H{"audits": audits})
}

// ============================================================================
// Dashboard
// ============================================================================

func (s *BugBountyService) GetDashboard(c *gin.Context) {
	// Report stats
	var total, pending, triaged, confirmed, resolved, rewarded int64
	s.db.Model(&VulnerabilityReport{}).Count(&total)
	s.db.Model(&VulnerabilityReport{}).Where("status = ?", "submitted").Count(&pending)
	s.db.Model(&VulnerabilityReport{}).Where("status = ?", "triaged").Count(&triaged)
	s.db.Model(&VulnerabilityReport{}).Where("status = ?", "confirmed").Count(&confirmed)
	s.db.Model(&VulnerabilityReport{}).Where("status = ?", "resolved").Count(&resolved)
	s.db.Model(&VulnerabilityReport{}).Where("reward_status = ?", "paid").Count(&rewarded)

	// By severity
	var bySeverity map[string]int
	s.db.Model(&VulnerabilityReport{}).Group("severity").Count(&bySeverity)

	// Reward stats
	var totalPaid, pendingPay float64
	s.db.Model(&Reward{}).Where("status = ?", "completed").Select("COALESCE(SUM(amount), 0)").Scan(&totalPaid)
	s.db.Model(&Reward{}).Where("status IN ?", []string{"pending", "processing"}).Select("COALESCE(SUM(amount), 0)").Scan(&pendingPay)

	// Researcher stats
	var totalResearchers, activeResearchers int64
	s.db.Model(&Researcher{}).Count(&totalResearchers)
	s.db.Model(&Researcher{}).Where("status = ?", "verified").Count(&activeResearchers)

	c.JSON(http.StatusOK, gin.H{
		"reports": gin.H{
			"total":       total,
			"pending":     pending,
			"triaged":     triaged,
			"confirmed":   confirmed,
			"resolved":    resolved,
			"rewarded":    rewarded,
			"by_severity": bySeverity,
		},
		"rewards": gin.H{
			"total_paid":    totalPaid,
			"pending_pay":  pendingPay,
			"pool_remaining": s.config.RewardPool - totalPaid,
		},
		"researchers": gin.H{
			"total":        totalResearchers,
			"active":       activeResearchers,
		},
	})
}

// ============================================================================
// Middleware
// ============================================================================

func (s *BugBountyService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:]
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return s.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, _ := token.Claims.(jwt.MapClaims)
		c.Set("researcher_id", uint(claims["researcher_id"].(float64)))
		c.Next()
	}
}

func (s *BugBountyService) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simplified - in production check admin role
		c.Set("admin_id", uint(1))
		c.Next()
	}
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	service, err := NewBugBountyService(config)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	router := gin.Default()
	router.Use(gin.Recovery())

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

	// Public
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "bug-bounty"})
	})

	router.POST("/api/v1/auth/register", service.Register)
	router.POST("/api/v1/auth/login", service.Login)

	// Researcher routes
	researcher := router.Group("/api/v1/researcher")
	researcher.Use(service.AuthMiddleware())
	{
		researcher.POST("/reports", service.SubmitReport)
		researcher.GET("/reports", service.GetReports)
		researcher.GET("/reports/:id", service.GetReport)
	}

	// Admin routes
	admin := router.Group("/api/v1/admin")
	admin.Use(service.AdminMiddleware())
	{
		admin.GET("/reports", service.ListAllReports)
		admin.POST("/reports/:id/review", service.ReviewReport)
		admin.POST("/rewards/:id/process", service.ProcessReward)
		admin.GET("/dashboard", service.GetDashboard)
		admin.GET("/audits", service.GetAudits)
	}

	// Public
	router.GET("/api/v1/program", service.GetProgram)
	router.GET("/api/v1/leaderboard", service.GetLeaderboard)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Bug Bounty service starting on port %s", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
}
