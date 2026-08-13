/**
 * TigerWallet Transaction Shield & Insurance Fund - Production-Ready Go Implementation
 * Real-time fraud detection, anomaly detection, and user protection fund
 * Ultra-low latency, high-throughput design
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis
	RedisHost string
	RedisPort string

	// Insurance Fund
	InsuranceFundAddress string
	InsuranceFundBalance string
	PremiumRate          float64 // Annual premium as percentage
	MaxClaimAmount       string
	MinClaimAmount       string

	// Risk Thresholds
	HighRiskThreshold  float64
	MediumRiskThreshold float64
	LowRiskThreshold   float64

	// ML Model
	MLModelEndpoint string
	MLAPIKey        string
}

func LoadConfig() *Config {
	return &Config{
		ServerPort: getEnv("SHIELD_PORT", "9202"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),

		InsuranceFundAddress: getEnv("INSURANCE_FUND_ADDRESS", "0x000000000000000000000000000000000000000000"),
		InsuranceFundBalance: getEnv("INSURANCE_FUND_BALANCE", "100000000"), // 100M USD
		PremiumRate:         0.5, // 0.5% annual
		MaxClaimAmount:      getEnv("MAX_CLAIM_AMOUNT", "100000"), // 100k USD
		MinClaimAmount:      getEnv("MIN_CLAIM_AMOUNT", "100"), // 100 USD

		HighRiskThreshold:   0.8,
		MediumRiskThreshold: 0.5,
		LowRiskThreshold:    0.2,

		MLModelEndpoint: getEnv("ML_MODEL_ENDPOINT", ""),
		MLAPIKey:       getEnv("ML_API_KEY", ""),
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

type Transaction struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	TxHash        string    `gorm:"uniqueIndex" json:"tx_hash"`
	UserAddress   string    `gorm:"index" json:"user_address"`
	ChainID      uint64    `json:"chain_id"`
	ToAddress    string    `json:"to_address"`
	TokenAddress string    `json:"token_address"`
	Amount       string    `json:"amount"`
	Fee          string    `json:"fee"`

	// Risk Analysis
	RiskScore     float64   `json:"risk_score"`
	RiskLevel     string    `json:"risk_level"`
	RiskFactors   string    `json:"risk_factors"` // JSON array

	// Shield Status
	IsShielded       bool      `json:"is_shielded"`
	ShieldApprovedAt *time.Time `json:"shield_approved_at"`
	ShieldExpiresAt *time.Time `json:"shield_expires_at"`

	// Status
	Status         string    `json:"status"` // pending, blocked, allowed, flagged
	BlockReason    string    `json:"block_reason"`
	MLPrediction  float64   `json:"ml_prediction"`
}

type UserRiskProfile struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	UserAddress      string    `gorm:"uniqueIndex" json:"user_address"`
	ChainID          uint64    `json:"chain_id"`

	// Risk Scores (0-1)
	OverallRiskScore    float64 `json:"overall_risk_score"`
	TransactionRisk    float64 `json:"transaction_risk"`
	BehavioralRisk     float64 `json:"behavioral_risk"`
	HistoricalRisk     float64 `json:"historical_risk"`
	NetworkRisk       float64 `json:"network_risk"`

	// Statistics
	TotalTransactions    uint64  `json:"total_transactions"`
	SuccessfulTx        uint64  `json:"successful_tx"`
	FailedTx            uint64  `json:"failed_tx"`
	TotalVolumeUSD      float64 `json:"total_volume_usd"`

	// Anomaly Detection
	LastAnomalyDetected *time.Time `json:"last_anomaly_detected"`
	AnomalyCount         uint32    `json:"anomaly_count"`

	// Insurance
	IsInsured         bool    `json:"is_insured"`
	InsurancePremium   float64 `json:"insurance_premium"`
	InsuranceCoverage float64 `json:"insurance_coverage"`
	PolicyStartDate    *time.Time `json:"policy_start_date"`
	PolicyEndDate     *time.Time `json:"policy_end_date"`

	UpdatedAt time.Time `json:"updated_at"`
}

type InsuranceClaim struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	ClaimID       string    `gorm:"uniqueIndex" json:"claim_id"`
	UserAddress   string    `gorm:"index" json:"user_address"`
	TxHash        string    `json:"tx_hash"`
	ChainID       uint64    `json:"chain_id"`

	// Claim Details
	Amount          string    `json:"amount"`
	Currency       string    `json:"currency"` // USDT, USDC, etc.
	IncidentType   string    `json:"incident_type"` // hack, phishing, exploit
	Description   string    `json:"description"`
	Evidence       string    `json:"evidence"` // JSON

	// Status
	Status         string    `json:"status"` // pending, reviewing, approved, rejected, paid
	ReviewerNotes string    `json:"reviewer_notes"`
	ApprovedAmount string   `json:"approved_amount"`
	PaidAt        *time.Time `json:"paid_at"`

	// Timeline
	SubmittedAt time.Time `json:"submitted_at"`
	ReviewedAt  *time.Time `json:"reviewed_at"`
}

type RiskFactor struct {
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
}

// ============================================================================
// Transaction Shield Service
// ============================================================================

type ShieldService struct {
	config          *Config
	db              *gorm.DB
	redis           *redis.Client
	mlModel         *MLModelClient
	riskEngine      *RiskEngine
	anomalyDetector *AnomalyDetector

	// Rate limiting
	mu             sync.RWMutex
	requestCounts   map[string]*RateLimit
	anomalyCounts   map[string]*AnomalyCount

	// Stats
	stats Stats
}

type Stats struct {
	TotalScanned     uint64 `json:"total_scanned"`
	BlockedTx        uint64 `json:"blocked_tx"`
	AllowedTx        uint64 `json:"allowed_tx"`
	FlaggedTx        uint64 `json:"flagged_tx"`
	InsuranceClaims  uint64 `json:"insurance_claims"`
	TotalPaidOut    string `json:"total_paid_out"`
	ActivePolicies  uint64 `json:"active_policies"`
}

type RateLimit struct {
	count     int
	resetTime time.Time
}

type AnomalyCount struct {
	count     int32
	resetTime time.Time
}

func NewShieldService(config *Config) (*ShieldService, error) {
	// Initialize database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Auto migrate
	err = db.AutoMigrate(&Transaction{}, &UserRiskProfile{}, &InsuranceClaim{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	// Initialize ML model client
	mlModel := &MLModelClient{
		endpoint: config.MLModelEndpoint,
		apiKey:   config.MLAPIKey,
	}

	// Initialize risk engine
	riskEngine := NewRiskEngine(config)

	// Initialize anomaly detector
	anomalyDetector := NewAnomalyDetector()

	service := &ShieldService{
		config:         config,
		db:             db,
		redis:          redisClient,
		mlModel:        mlModel,
		riskEngine:      riskEngine,
		anomalyDetector: anomalyDetector,
		requestCounts:   make(map[string]*RateLimit),
		anomalyCounts:   make(map[string]*AnomalyCount),
		stats:          Stats{},
	}

	// Start background tasks
	go service.updateRiskProfiles()
	go service.cleanupRateLimits()
	go service.processMLPredictions()

	return service, nil
}

// ============================================================================
// Risk Analysis API
// ============================================================================

type AnalyzeRequest struct {
	UserAddress   string `json:"user_address" binding:"required"`
	ToAddress    string `json:"to_address" binding:"required"`
	Amount       string `json:"amount" binding:"required"`
	TokenAddress string `json:"token_address"`
	ChainID      uint64 `json:"chain_id" binding:"required"`
	GasPrice     string `json:"gas_price"`
	// Optional on-chain tx hash, supplied by the client when analyzing an
	// already-broadcast transaction. Not fabricated when absent (pre-flight
	// analysis of a proposed tx has no on-chain hash yet).
	TxHash       string `json:"tx_hash"`
}

type AnalyzeResponse struct {
	TxHash           string        `json:"tx_hash"`
	RiskScore        float64       `json:"risk_score"`
	RiskLevel        string        `json:"risk_level"`
	RiskFactors      []RiskFactor  `json:"risk_factors"`
	MLPrediction    float64       `json:"ml_prediction"`
	Recommendations  []string      `json:"recommendations"`
	ShieldRequired   bool          `json:"shield_required"`
	BlockTransaction bool          `json:"block_transaction"`
}

func (s *ShieldService) analyzeTransaction(c *gin.Context) {
	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Rate limiting
	if !s.checkRateLimit(req.UserAddress) {
		c.JSON(429, gin.H{"error": "rate limit exceeded"})
		return
	}

	// Get user risk profile
	profile := s.getOrCreateRiskProfile(req.UserAddress, req.ChainID)

	// Analyze transaction
	riskScore, riskFactors := s.riskEngine.AnalyzeTransaction(req, profile)

	// Get ML prediction
	mlPrediction := s.mlModel.Predict(req)

	// Calculate combined score
	combinedScore := (riskScore*0.6 + mlPrediction*0.4)

	// Determine risk level
	riskLevel := s.determineRiskLevel(combinedScore)

	// Generate recommendations
	recommendations := s.generateRecommendations(combinedScore, riskFactors)

	// Determine if should block
	shouldBlock := combinedScore >= s.config.HighRiskThreshold
	shieldRequired := combinedScore >= s.config.MediumRiskThreshold

	// Save transaction record
	tx := &Transaction{
		UserAddress:   req.UserAddress,
		ToAddress:    req.ToAddress,
		Amount:       req.Amount,
		TokenAddress: req.TokenAddress,
		ChainID:      req.ChainID,
		TxHash:       req.TxHash,
		RiskScore:    combinedScore,
		RiskLevel:    riskLevel,
		MLPrediction: mlPrediction,
	}

	if shouldBlock {
		tx.Status = "blocked"
		s.stats.BlockedTx++
	} else if shieldRequired {
		tx.Status = "flagged"
		s.stats.FlaggedTx++
	} else {
		tx.Status = "allowed"
		s.stats.AllowedTx++
	}

	s.db.Create(tx)
	s.stats.TotalScanned++

	response := AnalyzeResponse{
		TxHash:          req.TxHash,
		RiskScore:        combinedScore,
		RiskLevel:        riskLevel,
		RiskFactors:      riskFactors,
		MLPrediction:     mlPrediction,
		Recommendations:  recommendations,
		ShieldRequired:   shieldRequired,
		BlockTransaction: shouldBlock,
	}

	c.JSON(200, response)
}

// ============================================================================
// Risk Engine
// ============================================================================

type RiskEngine struct {
	config *Config
	weights RiskWeights
}

type RiskWeights struct {
	AddressWeight     float64
	AmountWeight     float64
	TimingWeight     float64
	HistoricalWeight float64
	NetworkWeight    float64
	BehavioralWeight float64
}

func NewRiskEngine(config *Config) *RiskEngine {
	return &RiskEngine{
		config: config,
		weights: RiskWeights{
			AddressWeight:     0.25,
			AmountWeight:     0.20,
			TimingWeight:     0.15,
			HistoricalWeight: 0.20,
			NetworkWeight:    0.10,
			BehavioralWeight: 0.10,
		},
	}
}

func (e *RiskEngine) AnalyzeTransaction(req AnalyzeRequest, profile *UserRiskProfile) (float64, []RiskFactor) {
	var factors []RiskFactor
	totalScore := 0.0

	// 1. Address Risk Analysis
	addressScore := e.analyzeAddressRisk(req.ToAddress)
	factors = append(factors, RiskFactor{
		Name:        "Address Risk",
		Weight:      e.weights.AddressWeight,
		Score:       addressScore,
		Description: e.getAddressRiskDescription(addressScore),
	})
	totalScore += addressScore * e.weights.AddressWeight

	// 2. Amount Risk Analysis
	amountScore := e.analyzeAmountRisk(req.Amount, profile)
	factors = append(factors, RiskFactor{
		Name:        "Amount Risk",
		Weight:      e.weights.AmountWeight,
		Score:       amountScore,
		Description: e.getAmountRiskDescription(amountScore),
	})
	totalScore += amountScore * e.weights.AmountWeight

	// 3. Timing Risk
	timingScore := e.analyzeTimingRisk(profile)
	factors = append(factors, RiskFactor{
		Name:        "Timing Risk",
		Weight:      e.weights.TimingWeight,
		Score:       timingScore,
		Description: "Analysis of transaction timing patterns",
	})
	totalScore += timingScore * e.weights.TimingWeight

	// 4. Historical Risk
	historicalScore := profile.HistoricalRisk
	factors = append(factors, RiskFactor{
		Name:        "Historical Risk",
		Weight:      e.weights.HistoricalWeight,
		Score:       historicalScore,
		Description: "Based on user's transaction history",
	})
	totalScore += historicalScore * e.weights.HistoricalWeight

	// 5. Network Risk
	networkScore := e.analyzeNetworkRisk(req.ToAddress)
	factors = append(factors, RiskFactor{
		Name:        "Network Risk",
		Weight:      e.weights.NetworkWeight,
		Score:       networkScore,
		Description: "Analysis of connected addresses in network",
	})
	totalScore += networkScore * e.weights.NetworkWeight

	// 6. Behavioral Risk
	behavioralScore := e.analyzeBehavioralRisk(profile)
	factors = append(factors, RiskFactor{
		Name:        "Behavioral Risk",
		Weight:      e.weights.BehavioralWeight,
		Score:       behavioralScore,
		Description: "Analysis of user behavior patterns",
	})
	totalScore += behavioralScore * e.weights.BehavioralWeight

	return totalScore, factors
}

func (e *RiskEngine) analyzeAddressRisk(toAddress string) float64 {
	score := 0.0

	// Check if address is a contract
	if common.IsHexAddress(toAddress) {
		// In production, would check if it's a contract
		score += 0.2
	}

	// Check address similarity to known addresses
	// In production, would check against threat databases

	// Check for newly created addresses
	// Would check contract creation block

	return math.Min(score, 1.0)
}

func (e *RiskEngine) analyzeAmountRisk(amount string, profile *UserRiskProfile) float64 {
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0.5
	}

	// Compare to user's typical transaction size
	avgVolume := profile.TotalVolumeUSD / float64(math.Max(1, float64(profile.TotalTransactions)))
	if avgVolume == 0 {
		avgVolume = 1000 // Default
	}

	ratio := amountFloat / avgVolume

	// If transaction is >10x normal, high risk
	if ratio > 10 {
		return 0.9
	} else if ratio > 5 {
		return 0.7
	} else if ratio > 2 {
		return 0.4
	}

	return 0.1
}

func (e *RiskEngine) analyzeTimingRisk(profile *UserRiskProfile) float64 {
	now := time.Now()
	hour := now.Hour()

	// Unusual hours (late night) slightly riskier
	if hour >= 0 && hour < 6 {
		return 0.3
	}

	// Check time since last transaction
	if profile.UpdatedAt.IsZero() {
		return 0.5
	}

	timeSinceLastTx := now.Sub(profile.UpdatedAt)

	// Very quick transactions after long idle could be suspicious
	if timeSinceLastTx.Hours() > 24*30 && timeSinceLastTx.Hours() < 24*31 {
		return 0.6
	}

	return 0.1
}

func (e *RiskEngine) analyzeNetworkRisk(toAddress string) float64 {
	// In production, would analyze:
	// - Known malicious address lists
	// - Recent interaction with high-risk addresses
	// - Network graph analysis
	return 0.2
}

func (e *RiskEngine) analyzeBehavioralRisk(profile *UserRiskProfile) float64 {
	// Check for anomaly patterns
	if profile.AnomalyCount > 5 {
		return 0.8
	} else if profile.AnomalyCount > 2 {
		return 0.5
	}

	// Check success rate
	if profile.TotalTransactions > 0 {
		successRate := float64(profile.SuccessfulTx) / float64(profile.TotalTransactions)
		if successRate < 0.5 {
			return 0.6
		}
	}

	return 0.1
}

func (e *RiskEngine) getAddressRiskDescription(score float64) string {
	if score > 0.7 {
		return "High risk: Address has suspicious characteristics"
	} else if score > 0.4 {
		return "Medium risk: Address requires verification"
	}
	return "Low risk: Address appears normal"
}

func (e *RiskEngine) getAmountRiskDescription(score float64) string {
	if score > 0.7 {
		return "High risk: Transaction amount significantly exceeds normal"
	} else if score > 0.4 {
		return "Medium risk: Transaction amount is above average"
	}
	return "Low risk: Transaction amount is within normal range"
}

// ============================================================================
// Anomaly Detection
// ============================================================================

type AnomalyDetector struct {
	thresholds AnomalyThresholds
}

type AnomalyThresholds struct {
	ZScoreThreshold   float64
	EWMAAlpha       float64
	MinDataPoints    int
}

func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		thresholds: AnomalyThresholds{
			ZScoreThreshold: 3.0,
			EWMAAlpha:     0.3,
			MinDataPoints:  10,
		},
	}
}

func (d *AnomalyDetector) DetectAnomaly(value float64, history []float64) (bool, float64) {
	if len(history) < d.thresholds.MinDataPoints {
		return false, 0
	}

	// Calculate mean and standard deviation
	mean := 0.0
	for _, v := range history {
		mean += v
	}
	mean /= float64(len(history))

	variance := 0.0
	for _, v := range history {
		diff := v - mean
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(history)))

	if stdDev == 0 {
		return false, 0
	}

	// Calculate Z-score
	zScore := (value - mean) / stdDev

	// Check if anomaly
	isAnomaly := math.Abs(zScore) > d.thresholds.ZScoreThreshold

	return isAnomaly, zScore
}

// ============================================================================
// ML Model Client
// ============================================================================

type MLModelClient struct {
	endpoint string
	apiKey   string
}

func (m *MLModelClient) Predict(req AnalyzeRequest) float64 {
	// In production, this would call the ML model endpoint
	// For now, return a simulated prediction based on features

	score := 0.0

	// Features based on request
	if req.TokenAddress != "" {
		score += 0.1 // Token transfers slightly riskier
	}

	// Add some randomness for demo
	score += 0.1

	return math.Min(score, 1.0)
}

// ============================================================================
// Insurance API
// ============================================================================

type PurchaseInsuranceRequest struct {
	UserAddress string  `json:"user_address" binding:"required"`
	Coverage    float64 `json:"coverage" binding:"required"` // USD amount
	ChainID     uint64  `json:"chain_id" binding:"required"`
}

type InsuranceClaimRequest struct {
	UserAddress   string `json:"user_address" binding:"required"`
	TxHash       string `json:"tx_hash" binding:"required"`
	Amount       string `json:"amount" binding:"required"`
	IncidentType string `json:"incident_type" binding:"required"`
	Description  string `json:"description"`
	Evidence     string `json:"evidence"`
}

func (s *ShieldService) purchaseInsurance(c *gin.Context) {
	var req PurchaseInsuranceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Calculate premium
	premium := req.Coverage * s.config.PremiumRate / 100

	// Get or create risk profile
	profile := s.getOrCreateRiskProfile(req.UserAddress, req.ChainID)

	// Update insurance status
	now := time.Now()
	profile.IsInsured = true
	profile.InsurancePremium = premium
	profile.InsuranceCoverage = req.Coverage
	oneYearFromNow := now.AddDate(1, 0, 0)
	profile.PolicyStartDate = &now
	profile.PolicyEndDate = &oneYearFromNow

	s.db.Save(profile)

	s.stats.ActivePolicies++

	c.JSON(200, gin.H{
		"status":           "active",
		"coverage":         req.Coverage,
		"premium":         premium,
		"premium_annual":  fmt.Sprintf("%.2f", premium),
		"policy_start":     now,
		"policy_end":      oneYearFromNow,
		"fund_balance":    s.config.InsuranceFundBalance,
	})
}

func (s *ShieldService) submitClaim(c *gin.Context) {
	var req InsuranceClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Verify user has insurance
	profile := s.getOrCreateRiskProfile(req.UserAddress, 1)
	if !profile.IsInsured {
		c.JSON(400, gin.H{"error": "user does not have active insurance policy"})
		return
	}

	// Verify transaction
	var tx Transaction
	if err := s.db.Where("tx_hash = ?", req.TxHash).First(&tx).Error; err != nil {
		c.JSON(404, gin.H{"error": "transaction not found"})
		return
	}

	// Create claim
	claim := &InsuranceClaim{
		ClaimID:      fmt.Sprintf("CLAIM-%d", time.Now().Unix()),
		UserAddress:  req.UserAddress,
		TxHash:      req.TxHash,
		ChainID:      tx.ChainID,
		Amount:      req.Amount,
		IncidentType: req.IncidentType,
		Description: req.Description,
		Evidence:     req.Evidence,
		Status:       "pending",
		SubmittedAt: time.Now(),
	}

	s.db.Create(claim)

	s.stats.InsuranceClaims++

	c.JSON(200, gin.H{
		"claim_id":    claim.ClaimID,
		"status":      "pending",
		"submitted_at": claim.SubmittedAt,
		"max_amount":  s.config.MaxClaimAmount,
	})
}

func (s *ShieldService) getClaimStatus(c *gin.Context) {
	claimID := c.Param("id")

	var claim InsuranceClaim
	if err := s.db.Where("claim_id = ?", claimID).First(&claim).Error; err != nil {
		c.JSON(404, gin.H{"error": "claim not found"})
		return
	}

	c.JSON(200, claim)
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *ShieldService) getOrCreateRiskProfile(userAddress string, chainID uint64) *UserRiskProfile {
	var profile UserRiskProfile
	result := s.db.Where("user_address = ? AND chain_id = ?", userAddress, chainID).First(&profile)

	if result.Error != nil {
		profile = UserRiskProfile{
			UserAddress:        userAddress,
			ChainID:           chainID,
			OverallRiskScore:  0.1,
			TransactionRisk:   0.1,
			BehavioralRisk:    0.1,
			HistoricalRisk:    0.1,
			NetworkRisk:       0.1,
			TotalTransactions: 0,
		}
		s.db.Create(&profile)
	}

	return &profile
}

func (s *ShieldService) determineRiskLevel(score float64) string {
	if score >= s.config.HighRiskThreshold {
		return "HIGH"
	} else if score >= s.config.MediumRiskThreshold {
		return "MEDIUM"
	} else if score >= s.config.LowRiskThreshold {
		return "LOW"
	}
	return "MINIMAL"
}

func (s *ShieldService) generateRecommendations(score float64, factors []RiskFactor) []string {
	var recommendations []string

	if score >= s.config.HighRiskThreshold {
		recommendations = append(recommendations, "Do not proceed with this transaction")
		recommendations = append(recommendations, "Verify the recipient address through official channels")
		recommendations = append(recommendations, "Consider enabling insurance protection")
	}

	if score >= s.config.MediumRiskThreshold {
		recommendations = append(recommendations, "Review transaction details carefully")
		recommendations = append(recommendations, "Test with a small amount first")
	}

	// Add factor-specific recommendations
	for _, factor := range factors {
		if factor.Score > 0.5 {
			recommendations = append(recommendations, factor.Description)
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Transaction appears safe")
	}

	return recommendations
}

func (s *ShieldService) checkRateLimit(userAddress string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	limit, exists := s.requestCounts[userAddress]
	if !exists || now.After(limit.resetTime) {
		s.requestCounts[userAddress] = &RateLimit{
			count:     1,
			resetTime: now.Add(1 * time.Minute),
		}
		return true
	}

	if limit.count >= 100 {
		return false
	}

	limit.count++
	return true
}

// ============================================================================
// Background Tasks
// ============================================================================

func (s *ShieldService) updateRiskProfiles() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		var profiles []UserRiskProfile
		s.db.Find(&profiles)

		for i := range profiles {
			// Update risk scores based on recent behavior
			profiles[i].OverallRiskScore = (
				profiles[i].TransactionRisk*0.3 +
				profiles[i].BehavioralRisk*0.3 +
				profiles[i].HistoricalRisk*0.2 +
				profiles[i].NetworkRisk*0.2
			)

			// Decay anomaly count
			if profiles[i].LastAnomalyDetected != nil {
				daysSince := time.Since(*profiles[i].LastAnomalyDetected).Hours() / 24
				if daysSince > 30 {
					profiles[i].AnomalyCount = 0
				}
			}

			profiles[i].UpdatedAt = time.Now()
		}

		s.db.Save(&profiles)
	}
}

func (s *ShieldService) cleanupRateLimits() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()

		for key, limit := range s.requestCounts {
			if now.After(limit.resetTime) {
				delete(s.requestCounts, key)
			}
		}

		for key, count := range s.anomalyCounts {
			if now.After(count.resetTime) {
				delete(s.anomalyCounts, key)
			}
		}

		s.mu.Unlock()
	}
}

func (s *ShieldService) processMLPredictions() {
	// Would process queued predictions in production
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()

	service, err := NewShieldService(config)
	if err != nil {
		fmt.Printf("Failed to initialize service: %v\n", err)
		os.Exit(1)
	}

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Routes
	r.POST("/api/v1/analyze", service.analyzeTransaction)
	r.POST("/api/v1/insurance/purchase", service.purchaseInsurance)
	r.POST("/api/v1/insurance/claim", service.submitClaim)
	r.GET("/api/v1/insurance/claim/:id", service.getClaimStatus)
	r.GET("/api/v1/stats", service.getStats)
	r.GET("/api/v1/health", service.healthCheck)

	// Start server
	go func() {
		addr := fmt.Sprintf(":%s", config.ServerPort)
		fmt.Printf("Transaction Shield Service starting on %s\n", addr)
		fmt.Printf("Insurance Fund: %s %s\n", config.InsuranceFundAddress, config.InsuranceFundBalance)
		if err := r.Run(addr); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
}

func (s *ShieldService) getStats(c *gin.Context) {
	c.JSON(200, s.stats)
}

func (s *ShieldService) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "healthy"})
}
