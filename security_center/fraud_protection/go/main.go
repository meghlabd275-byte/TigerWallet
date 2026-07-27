package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	"github.com/rs/zerolog"
)

// ============================================================================
// TIGERWALLET FRAUD PROTECTION & TRANSACTION SHIELD SYSTEM
// Like MetaMask's $10k fraud protection
// Provides real-time transaction monitoring, fraud detection, and coverage
// ============================================================================

var (
	logger       zerolog.Logger
	redisClient  *redis.Client
	policyEngine *PolicyEngine
)

func main() {
	// Initialize logger
	output := zerolog.ConsoleWriter{Output: os.Stdout}
	logger = zerolog.New(output).With().Timestamp().Logger()

	// Load configuration
	cfg := loadConfig()

	// Initialize Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("Redis connection failed")
	}

	// Initialize policy engine
	policyEngine = NewPolicyEngine(cfg)

	// Setup router
	router := setupRouter(cfg)

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	logger.Info().Str("port", cfg.Port).Msg("Fraud Protection service started")

	// Start background tasks
	go policyEngine.StartBackgroundTasks(ctx)

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	logger.Info().Msg("Server exited")
}

// Configuration
type Config struct {
	Port           string
	RedisURL       string
	MaxCoverageUSD float64
	MonthlyPremium float64
	AutoRenewDays  int
}

func loadConfig() *Config {
	return &Config{
		Port:           getEnv("FRAUD_PORT", "9210"),
		RedisURL:       getEnv("REDIS_URL", "localhost:6379"),
		MaxCoverageUSD: 10000, // $10k coverage like MetaMask
		MonthlyPremium: 4.99,  // $4.99/month
		AutoRenewDays:  30,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ============================================================================
// DATA MODELS
// ============================================================================

// ProtectionPolicy represents a user's fraud protection policy
type ProtectionPolicy struct {
	PolicyID          string    `json:"policyId"`
	UserID            string    `json:"userId"`
	Status            string    `json:"status"` // active, suspended, expired, cancelled
	CoverageLimitUSD float64   `json:"coverageLimitUSD"`
	CoverageUsedUSD   float64   `json:"coverageUsedUSD"`
	PremiumUSD        float64   `json:"premiumUSD"`
	StartDate        int64     `json:"startDate"`
	EndDate          int64     `json:"endDate"`
	AutoRenew        bool      `json:"autoRenew"`
	Features         []string  `json:"features"` // fraud_detection, real_time_alerts, coverage
	CreatedAt        int64     `json:"createdAt"`
	UpdatedAt        int64     `json:"updatedAt"`
}

// Claim represents a fraud claim
type Claim struct {
	ClaimID         string    `json:"claimId"`
	PolicyID        string    `json:"policyId"`
	UserID          string    `json:"userId"`
	IncidentDate    int64     `json:"incidentDate"`
	IncidentType    string    `json:"incidentType"` // unauthorized_access, phishing, rug_pull, exchange_hack
	AmountUSD       float64   `json:"amountUSD"`
	Status          string    `json:"status"` // pending, reviewing, approved, rejected, paid
	Description     string    `json:"description"`
	Evidence        []string  `json:"evidence"` // URLs to evidence
	TxHashes        []string  `json:"txHashes"`
	ApprovedAmount  float64   `json:"approvedAmount,omitempty"`
	ProcessedAt     int64     `json:"processedAt,omitempty"`
	PaidAt          int64     `json:"paidAt,omitempty"`
	CreatedAt       int64     `json:"createdAt"`
}

// TransactionAlert represents a real-time transaction alert
type TransactionAlert struct {
	AlertID       string    `json:"alertId"`
	UserID        string    `json:"userId"`
	TxHash        string    `json:"txHash"`
	ChainID       uint64    `json:"chainId"`
	FromAddress   string    `json:"fromAddress"`
	ToAddress     string    `json:"toAddress"`
	AmountUSD     float64   `json:"amountUSD"`
	Token         string    `json:"token"`
	RiskScore     float64   `json:"riskScore"` // 0-100
	RiskLevel     string    `json:"riskLevel"` // low, medium, high, critical
	Alerts        []string  `json:"alerts"` // suspicious_address, high_value, unusual_pattern
	Recommendation string    `json:"recommendation"` // block, warn, allow
	CreatedAt     int64     `json:"createdAt"`
}

// UserBehavior represents user behavior patterns
type UserBehavior struct {
	UserID             string    `json:"userId"`
	AvgTransactionUSD  float64   `json:"avgTransactionUSD"`
	MaxTransactionUSD  float64   `json:"maxTransactionUSD"`
	TransactionCount   int       `json:"transactionCount"`
	UniqueAddresses   int       `json:"uniqueAddresses"`
	Timezone          string    `json:"timezone"`
	ActiveHours       []int     `json:"activeHours"` // 0-23
	PreferredChains   []uint64  `json:"preferredChains"`
	LastActivity      int64     `json:"lastActivity"`
	FirstActivity     int64     `json:"firstActivity"`
	UpdatedAt         int64     `json:"updatedAt"`
}

// ============================================================================
// POLICY ENGINE
// ============================================================================

type PolicyEngine struct {
	config *Config
	mu     sync.RWMutex
	rules  []RiskRule
}

func NewPolicyEngine(cfg *Config) *PolicyEngine {
	engine := &PolicyEngine{
		config: cfg,
		rules:  []RiskRule{},
	}

	// Load default rules
	engine.loadDefaultRules()

	return engine
}

func (e *PolicyEngine) loadDefaultRules() {
	e.rules = []RiskRule{
		{
			Name:        "high_value_transaction",
			Description: "Flag transactions over $1000",
			Condition: func(ctx *RiskContext) bool {
				return ctx.AmountUSD > 1000
			},
			Severity:   "medium",
			Weight:     20,
		},
		{
			Name:        "very_high_value_transaction",
			Description: "Flag transactions over $5000",
			Condition: func(ctx *RiskContext) bool {
				return ctx.AmountUSD > 5000
			},
			Severity:   "high",
			Weight:     50,
		},
		{
			Name:        "unknown_recipient",
			Description: "Transaction to address not in user's contacts",
			Condition: func(ctx *RiskContext) bool {
				return !ctx.IsKnownRecipient
			},
			Severity:   "medium",
			Weight:     30,
		},
		{
			Name:        "suspicious_address",
			Description: "Recipient address flagged in threat database",
			Condition: func(ctx *RiskContext) bool {
				return ctx.IsSuspiciousAddress
			},
			Severity:   "critical",
			Weight:     100,
		},
		{
			Name:        "unusual_time",
			Description: "Transaction at unusual time for user",
			Condition: func(ctx *RiskContext) bool {
				return ctx.IsUnusualTime
			},
			Severity:   "low",
			Weight:     10,
		},
		{
			Name:        "new_address",
			Description: "First transaction to this address",
			Condition: func(ctx *RiskContext) bool {
				return ctx.IsNewAddress
			},
			Severity:   "medium",
			Weight:     25,
		},
		{
			Name:        "rapid_transactions",
			Description: "Multiple transactions in short period",
			Condition: func(ctx *RiskContext) bool {
				return ctx.RecentTxCount > 5
			},
			Severity:   "high",
			Weight:     40,
		},
		{
			Name:        "cross_chain_unusual",
			Description: "Cross-chain to unusual chain",
			Condition: func(ctx *RiskContext) bool {
				return ctx.IsCrossChain && !ctx.IsPreferredChain
			},
			Severity:   "medium",
			Weight:     25,
		},
	}
}

// RiskContext for rule evaluation
type RiskContext struct {
	UserID              string
	AmountUSD           float64
	ChainID             uint64
	IsKnownRecipient    bool
	IsSuspiciousAddress bool
	IsUnusualTime       bool
	IsNewAddress        bool
	RecentTxCount       int
	IsCrossChain        bool
	IsPreferredChain    bool
	HourOfDay           int
}

// RiskRule definition
type RiskRule struct {
	Name        string
	Description string
	Condition   func(*RiskContext) bool
	Severity    string
	Weight      float64
}

func (e *PolicyEngine) EvaluateTransaction(ctx *RiskContext) *TransactionAlert {
	alert := &TransactionAlert{
		AlertID:     generateID(),
		UserID:      ctx.UserID,
		RiskScore:   0,
		RiskLevel:   "low",
		Alerts:      []string{},
		CreatedAt:    time.Now().Unix(),
	}

	// Evaluate all rules
	for _, rule := range e.rules {
		if rule.Condition(ctx) {
			alert.Alerts = append(alert.Alerts, rule.Name)
			alert.RiskScore += rule.Weight
		}
	}

	// Cap risk score at 100
	if alert.RiskScore > 100 {
		alert.RiskScore = 100
	}

	// Determine risk level
	switch {
	case alert.RiskScore >= 80:
		alert.RiskLevel = "critical"
		alert.Recommendation = "block"
	case alert.RiskScore >= 50:
		alert.RiskLevel = "high"
		alert.Recommendation = "block"
	case alert.RiskScore >= 25:
		alert.RiskLevel = "medium"
		alert.Recommendation = "warn"
	default:
		alert.RiskLevel = "low"
		alert.Recommendation = "allow"
	}

	return alert
}

func (e *PolicyEngine) StartBackgroundTasks(ctx context.Context) {
	// Periodic cleanup and updates
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.cleanupOldData(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (e *PolicyEngine) cleanupOldData(ctx context.Context) {
	// Clean up old alerts and data
	logger.Info().Msg("Running periodic cleanup")
}

// ============================================================================
// API HANDLERS
// ============================================================================

func setupRouter(cfg *Config) *gin.Engine {
	r := gin.Default()

	r.Use(corsMiddleware())

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Protection policies
		policies := v1.Group("/policies")
		{
			policies.POST("", createPolicy)
			policies.GET("/:id", getPolicy)
			policies.GET("/user/:userId", getUserPolicy)
			policies.PUT("/:id", updatePolicy)
			policies.POST("/:id/renew", renewPolicy)
			policies.POST("/:id/cancel", cancelPolicy)
		}

		// Claims
		claims := v1.Group("/claims")
		{
			claims.POST("", createClaim)
			claims.GET("/:id", getClaim)
			claims.GET("/user/:userId", getUserClaims)
			claims.POST("/:id/evidence", addEvidence)
		}

		// Real-time monitoring
		monitor := v1.Group("/monitor")
		{
			monitor.POST("/transaction", analyzeTransaction)
			monitor.POST("/batch", analyzeBatch)
			monitor.GET("/alerts/:userId", getUserAlerts)
			monitor.POST("/alert/:id/resolve", resolveAlert)
		}

		// User behavior
		behavior := v1.Group("/behavior")
		{
			behavior.POST("", updateBehavior)
			behavior.GET("/:userId", getBehavior)
		}

		// Coverage
		coverage := v1.Group("/coverage")
		{
			coverage.GET("/:userId", getCoverage)
			coverage.GET("/usage/:userId", getCoverageUsage)
		}
	}

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ============================================================================
// POLICY HANDLERS
// ============================================================================

func createPolicy(c *gin.Context) {
	var req struct {
		UserID     string  `json:"userId" binding:"required"`
		Coverage  float64 `json:"coverage"`
		AutoRenew bool    `json:"autoRenew"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already has policy
	existingKey := fmt.Sprintf("policy:user:%s", req.UserID)
	if data, err := redisClient.Get(context.Background(), existingKey).Bytes(); err == nil {
		var existing ProtectionPolicy
		if json.Unmarshal(data, &existing) == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "User already has active policy"})
			return
		}
	}

	// Determine coverage (cap at max)
	coverage := req.Coverage
	if coverage > 10000 {
		coverage = 10000
	}
	if coverage <= 0 {
		coverage = 1000 // Default $1k coverage
	}

	now := time.Now()
	policy := ProtectionPolicy{
		PolicyID:          generateID(),
		UserID:            req.UserID,
		Status:            "active",
		CoverageLimitUSD:  coverage,
		CoverageUsedUSD:   0,
		PremiumUSD:        4.99,
		StartDate:         now.Unix(),
		EndDate:           now.AddDate(0, 1, 0).Unix(),
		AutoRenew:         req.AutoRenew,
		Features:          []string{"fraud_detection", "real_time_alerts", "coverage"},
		CreatedAt:         now.Unix(),
		UpdatedAt:         now.Unix(),
	}

	// Save to Redis
	policyKey := fmt.Sprintf("policy:%s", policy.PolicyID)
	if data, err := json.Marshal(policy); err == nil {
		redisClient.Set(context.Background(), policyKey, data, 365*24*time.Hour)
		redisClient.Set(context.Background(), existingKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusCreated, policy)
}

func getPolicy(c *gin.Context) {
	id := c.Param("id")
	policyKey := fmt.Sprintf("policy:%s", id)

	data, err := redisClient.Get(context.Background(), policyKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}

	var policy ProtectionPolicy
	json.Unmarshal(data, &policy)
	c.JSON(http.StatusOK, policy)
}

func getUserPolicy(c *gin.Context) {
	userID := c.Param("userId")
	existingKey := fmt.Sprintf("policy:user:%s", userID)

	data, err := redisClient.Get(context.Background(), existingKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active policy"})
		return
	}

	var policy ProtectionPolicy
	json.Unmarshal(data, &policy)
	c.JSON(http.StatusOK, policy)
}

func updatePolicy(c *gin.Context) {
	id := c.Param("id")
	var updates struct {
		CoverageLimitUSD float64 `json:"coverageLimitUSD"`
		AutoRenew        bool    `json:"autoRenew"`
	}

	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policyKey := fmt.Sprintf("policy:%s", id)
	data, err := redisClient.Get(context.Background(), policyKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}

	var policy ProtectionPolicy
	json.Unmarshal(data, &policy)

	if updates.CoverageLimitUSD > 0 && updates.CoverageLimitUSD <= 10000 {
		policy.CoverageLimitUSD = updates.CoverageLimitUSD
	}
	policy.AutoRenew = updates.AutoRenew
	policy.UpdatedAt = time.Now().Unix()

	// Save updated policy
	if data, err := json.Marshal(policy); err == nil {
		redisClient.Set(context.Background(), policyKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, policy)
}

func renewPolicy(c *gin.Context) {
	id := c.Param("id")
	policyKey := fmt.Sprintf("policy:%s", id)

	data, err := redisClient.Get(context.Background(), policyKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}

	var policy ProtectionPolicy
	json.Unmarshal(data, &policy)

	// Extend policy by 1 month
	policy.EndDate = time.Unix(policy.EndDate, 0).AddDate(0, 1, 0).Unix()
	policy.Status = "active"
	policy.UpdatedAt = time.Now().Unix()

	if data, err := json.Marshal(policy); err == nil {
		redisClient.Set(context.Background(), policyKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, policy)
}

func cancelPolicy(c *gin.Context) {
	id := c.Param("id")
	policyKey := fmt.Sprintf("policy:%s", id)

	data, err := redisClient.Get(context.Background(), policyKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}

	var policy ProtectionPolicy
	json.Unmarshal(data, &policy)

	policy.Status = "cancelled"
	policy.UpdatedAt = time.Now().Unix()

	if data, err := json.Marshal(policy); err == nil {
		redisClient.Set(context.Background(), policyKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, policy)
}

// ============================================================================
// CLAIM HANDLERS
// ============================================================================

func createClaim(c *gin.Context) {
	var req struct {
		PolicyID     string  `json:"policyId" binding:"required"`
		UserID       string  `json:"userId" binding:"required"`
		IncidentType string  `json:"incidentType" binding:"required"`
		AmountUSD    float64 `json:"amountUSD" binding:"required,gt=0"`
		Description  string  `json:"description"`
		TxHashes     []string `json:"txHashes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify policy exists and is active
	policyKey := fmt.Sprintf("policy:%s", req.PolicyID)
	data, err := redisClient.Get(context.Background(), policyKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}

	var policy ProtectionPolicy
	json.Unmarshal(data, &policy)

	if policy.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Policy is not active"})
		return
	}

	// Check coverage limit
	if req.AmountUSD > (policy.CoverageLimitUSD - policy.CoverageUsedUSD) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Claim exceeds available coverage"})
		return
	}

	claim := Claim{
		ClaimID:      generateID(),
		PolicyID:     req.PolicyID,
		UserID:       req.UserID,
		IncidentDate: time.Now().Unix(),
		IncidentType: req.IncidentType,
		AmountUSD:    req.AmountUSD,
		Status:       "pending",
		Description:  req.Description,
		TxHashes:     req.TxHashes,
		CreatedAt:    time.Now().Unix(),
	}

	// Save claim
	claimKey := fmt.Sprintf("claim:%s", claim.ClaimID)
	if data, err := json.Marshal(claim); err == nil {
		redisClient.Set(context.Background(), claimKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusCreated, claim)
}

func getClaim(c *gin.Context) {
	id := c.Param("id")
	claimKey := fmt.Sprintf("claim:%s", id)

	data, err := redisClient.Get(context.Background(), claimKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Claim not found"})
		return
	}

	var claim Claim
	json.Unmarshal(data, &claim)
	c.JSON(http.StatusOK, claim)
}

func getUserClaims(c *gin.Context) {
	userID := c.Param("userId")
	// In production: query from database
	c.JSON(http.StatusOK, []Claim{})
}

func addEvidence(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		EvidenceURL string `json:"evidenceUrl" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claimKey := fmt.Sprintf("claim:%s", id)
	data, err := redisClient.Get(context.Background(), claimKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Claim not found"})
		return
	}

	var claim Claim
	json.Unmarshal(data, &claim)

	claim.Evidence = append(claim.Evidence, req.EvidenceURL)

	if data, err := json.Marshal(claim); err == nil {
		redisClient.Set(context.Background(), claimKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, claim)
}

// ============================================================================
// MONITORING HANDLERS
// ============================================================================

func analyzeTransaction(c *gin.Context) {
	var ctx RiskContext
	if err := c.ShouldBindJSON(&ctx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	alert := policyEngine.EvaluateTransaction(&ctx)

	// Save alert
	if alert.RiskLevel == "high" || alert.RiskLevel == "critical" {
		alertKey := fmt.Sprintf("alert:%s", alert.AlertID)
		if data, err := json.Marshal(alert); err == nil {
			redisClient.Set(context.Background(), alertKey, data, 7*24*time.Hour)
		}
	}

	c.JSON(http.StatusOK, alert)
}

func analyzeBatch(c *gin.Context) {
	var req struct {
		Transactions []RiskContext `json:"transactions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	alerts := make([]*TransactionAlert, len(req.Transactions))
	for i, tx := range req.Transactions {
		alerts[i] = policyEngine.EvaluateTransaction(&tx)
	}

	c.JSON(http.StatusOK, alerts)
}

func getUserAlerts(c *gin.Context) {
	userID := c.Param("userId")
	// In production: query from Redis/DB
	c.JSON(http.StatusOK, []TransactionAlert{})
}

func resolveAlert(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Alert resolved"})
}

// ============================================================================
// BEHAVIOR HANDLERS
// ============================================================================

func updateBehavior(c *gin.Context) {
	var behavior UserBehavior
	if err := c.ShouldBindJSON(&behavior); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	behavior.UpdatedAt = time.Now().Unix()

	behaviorKey := fmt.Sprintf("behavior:%s", behavior.UserID)
	if data, err := json.Marshal(behavior); err == nil {
		redisClient.Set(context.Background(), behaviorKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, behavior)
}

func getBehavior(c *gin.Context) {
	userID := c.Param("userId")
	behaviorKey := fmt.Sprintf("behavior:%s", userID)

	data, err := redisClient.Get(context.Background(), behaviorKey).Bytes()
	if err != nil {
		// Return default behavior
		c.JSON(http.StatusOK, UserBehavior{
			UserID:      userID,
			ActiveHours: []int{9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21},
		})
		return
	}

	var behavior UserBehavior
	json.Unmarshal(data, &behavior)
	c.JSON(http.StatusOK, behavior)
}

// ============================================================================
// COVERAGE HANDLERS
// ============================================================================

func getCoverage(c *gin.Context) {
	userID := c.Param("userId")
	existingKey := fmt.Sprintf("policy:user:%s", userID)

	data, err := redisClient.Get(context.Background(), existingKey).Bytes()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"hasCoverage":   false,
			"coverageUSD":  0,
			"maxCoverage":  10000,
		})
		return
	}

	var policy ProtectionPolicy
	json.Unmarshal(data, &policy)

	c.JSON(http.StatusOK, gin.H{
		"hasCoverage":     policy.Status == "active",
		"coverageUSD":     policy.CoverageLimitUSD - policy.CoverageUsedUSD,
		"maxCoverage":     policy.CoverageLimitUSD,
		"premiumUSD":      policy.PremiumUSD,
		"renewalDate":    policy.EndDate,
		"autoRenew":      policy.AutoRenew,
	})
}

func getCoverageUsage(c *gin.Context) {
	userID := c.Param("userId")
	existingKey := fmt.Sprintf("policy:user:%s", userID)

	data, err := redisClient.Get(context.Background(), existingKey).Bytes()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"totalUsedUSD":   0,
			"claimsCount":   0,
			"claimHistory":  []Claim{},
		})
		return
	}

	var policy ProtectionPolicy
	json.Unmarshal(data, &policy)

	// In production: get actual claims from database
	c.JSON(http.StatusOK, gin.H{
		"totalUsedUSD":  policy.CoverageUsedUSD,
		"remainingUSD":  policy.CoverageLimitUSD - policy.CoverageUsedUSD,
		"claimsCount":  0,
		"claimHistory":  []Claim{},
	})
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
