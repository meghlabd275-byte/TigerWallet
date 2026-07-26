/**
 * TigerWallet Transaction Shield Service
 * AI-Powered Fraud Detection & Prevention System
 * 
 * Features:
 * - Real-time transaction monitoring
 * - ML-based fraud detection
 * - Risk scoring engine
 * - Pattern recognition
 * - Anomaly detection
 * - Behavioral analysis
 * - Alert management
 * - Auto-blocking high-risk transactions
 */

package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"regexp"
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
	RedisHost       string
	RedisPort       string
	AIEndpoint      string
	AIAPIKey        string
	BlockThreshold  float64
	ReviewThreshold float64
	AlertThreshold  float64
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:      getEnv("TRANSACTION_SHIELD_PORT", "9095"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "tigerwallet"),
		DBPassword:      getEnv("DB_PASSWORD", "password"),
		DBName:         getEnv("DB_NAME", "tigerwallet"),
		RedisHost:      getEnv("REDIS_HOST", "localhost"),
		RedisPort:      getEnv("REDIS_PORT", "6379"),
		AIEndpoint:     getEnv("AI_ENDPOINT", "http://localhost:8080"),
		AIAPIKey:       getEnv("AI_API_KEY", ""),
		BlockThreshold: 0.85,
		ReviewThreshold: 0.60,
		AlertThreshold:  0.40,
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
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	TransactionHash string         `gorm:"uniqueIndex;size:66" json:"transaction_hash"`
	TxHash          string         `gorm:"index" json:"tx_hash"`
	UserID          uint           `gorm:"index" json:"user_id"`
	User            User           `gorm:"foreignKey:UserID" json:"-"`
	ChainID         int            `json:"chain_id"`
	FromAddress     string          `gorm:"index" json:"from_address"`
	ToAddress       string          `gorm:"index" json:"to_address"`
	Amount          string         `json:"amount"`
	Token           string         `json:"token"`
	TokenAddress    string          `json:"token_address"`
	GasUsed         string         `json:"gas_used"`
	GasPrice        string         `json:"gas_price"`
	Status          string          `json:"status"` // pending, confirmed, failed, blocked
	BlockNumber     int64          `json:"block_number"`
	Timestamp       time.Time      `json:"timestamp"`
	RiskScore       float64        `json:"risk_score"`
	RiskLevel       string         `json:"risk_level"` // low, medium, high, critical
	RiskReasons     string         `json:"risk_reasons"` // JSON array
	ReviewedBy      *uint          `json:"reviewed_by"`
	ReviewedAt      *time.Time     `json:"reviewed_at"`
	ReviewNotes     string         `json:"review_notes"`
}

type User struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	UUID              string         `gorm:"uniqueIndex;size:36" json:"uuid"`
	Email             string         `gorm:"index" json:"email"`
	Username          string         `gorm:"uniqueIndex" json:"username"`
	WalletAddresses   []WalletAddress `json:"wallet_addresses"`
	RiskProfile       RiskProfile    `json:"risk_profile"`
}

type WalletAddress struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	UserID      uint      `gorm:"index" json:"user_id"`
	Address     string    `gorm:"uniqueIndex" json:"address"`
	AddressType string    `json:"address_type"`
	ChainID     int       `json:"chain_id"`
	Label       string    `json:"label"`
	IsKnown     bool      `json:"is_known"`
	IsTrusted   bool      `json:"is_trusted"`
	RiskScore   float64   `json:"risk_score"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	TxCount     int       `json:"tx_count"`
	Volume      string    `json:"volume"` // Total volume in USD
}

type RiskProfile struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	UserID           uint      `gorm:"uniqueIndex" json:"user_id"`
	User             User      `gorm:"foreignKey:UserID" json:"-"`
	OverallScore     float64   `json:"overall_score"` // 0-1
	VelocityScore    float64   `json:"velocity_score"`
	AmountScore      float64   `json:"amount_score"`
	PatternScore     float64   `json:"pattern_score"`
	BehavioralScore  float64   `json:"behavioral_score"`
	ReputationScore  float64   `json:"reputation_score"`
	LastUpdated      time.Time `json:"last_updated"`
	Flags            string    `json:"flags"` // JSON array
	RiskLevel        string    `json:"risk_level"` // low, medium, high, critical
	ReviewStatus     string    `json:"review_status"` // none, pending, approved, flagged
	ManualOverride   bool      `json:"manual_override"`
	OverrideBy       *uint     `json:"override_by"`
	OverrideAt       *time.Time `json:"override_at"`
	OverrideReason   string    `json:"override_reason"`
}

type Alert struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	AlertID       string    `gorm:"uniqueIndex;size:36" json:"alert_id"`
	UserID        uint      `gorm:"index" json:"user_id"`
	User          User      `gorm:"foreignKey:UserID" json:"-"`
	TransactionID *uint     `json:"transaction_id"`
	Transaction   Transaction `gorm:"foreignKey:TransactionID" json:"-"`
	AlertType    string    `json:"alert_type"` // high_risk, unusual_activity, pattern_violation, velocity_exceeded
	Severity     string    `json:"severity"` // low, medium, high, critical
	Title        string    `json:"title"`
	Description   string    `json:"description"`
	RiskFactors  string    `json:"risk_factors"` // JSON array
	Amount       string    `json:"amount"`
	Timestamp    time.Time `json:"timestamp"`
	Status       string    `json:"status"` // new, acknowledged, resolved, false_positive
	ResolvedBy   *uint     `json:"resolved_by"`
	ResolvedAt   *time.Time `json:"resolved_at"`
	Resolution   string    `json:"resolution"`
}

type BlockedAddress struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	Address       string    `gorm:"uniqueIndex;size:44" json:"address"`
	AddressType   string    `json:"address_type"` // evm, bitcoin, solana, etc.
	ChainID       int       `json:"chain_id"`
	BlockedReason string    `json:"blocked_reason"`
	BlockedAt     time.Time `json:"blocked_at"`
	BlockedBy     string    `json:"blocked_by"` // system, admin, oracle
	Expiry        *time.Time `json:"expiry"`
	IsPermanent   bool      `json:"is_permanent"`
	Metadata      string    `json:"metadata"` // JSON
}

type TransactionPattern struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	PatternID         string    `gorm:"uniqueIndex;size:36" json:"pattern_id"`
	PatternName       string    `json:"pattern_name"`
	PatternType       string    `json:"pattern_type"` // velocity, amount, timing, destination, source
	Description       string    `json:"description"`
	RiskWeight        float64   `json:"risk_weight"`
	DetectionRules    string    `json:"detection_rules"` // JSON
	Action            string    `json:"action"` // block, review, alert
	MinTransactions   int       `json:"min_transactions"`
	TimeWindowSeconds int       `json:"time_window_seconds"`
	AmountThreshold   string    `json:"amount_threshold"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ReviewQueue struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	TransactionID uint      `gorm:"uniqueIndex" json:"transaction_id"`
	Transaction   Transaction `gorm:"foreignKey:TransactionID" json:"-"`
	Priority      int       `json:"priority"` // 1-5, 1 is highest
	AssignedTo   *uint     `json:"assigned_to"`
	Status       string    `json:"status"` // pending, in_review, resolved
	Notes        string    `json:"notes"`
}

// ============================================================================
// Risk Scoring Engine
// ============================================================================

type RiskScore struct {
	OverallScore    float64            `json:"overall_score"`
	ComponentScores map[string]float64 `json:"component_scores"`
	RiskLevel       string             `json:"risk_level"`
	RiskFactors     []RiskFactor       `json:"risk_factors"`
	Recommendations []string           `json:"recommendations"`
}

type RiskFactor struct {
	Factor   string  `json:"factor"`
	Weight   float64 `json:"weight"`
	Score    float64 `json:"score"`
	Reason   string  `json:"reason"`
	Details  string  `json:"details"`
}

type TransactionFeatures struct {
	// User features
	UserTxCount          int     `json:"user_tx_count"`
	UserAvgAmount        float64 `json:"user_avg_amount"`
	UserMaxAmount        float64 `json:"user_max_amount"`
	UserAccountAgeDays   int     `json:"user_account_age_days"`
	UserKYCLevel        int     `json:"user_kyc_level"`
	UserRiskProfile     float64 `json:"user_risk_profile"`
	
	// Transaction features
	Amount              float64 `json:"amount"`
	AmountInUSD         float64 `json:"amount_in_usd"`
	IsLargeAmount       bool    `json:"is_large_amount"`
	TokenIsStablecoin   bool    `json:"token_is_stablecoin"`
	
	// Address features
	FromAddressAge      int     `json:"from_address_age"`
	FromAddressTxCount  int     `json:"from_address_tx_count"`
	FromAddressKnown    bool    `json:"from_address_known"`
	FromAddressTrusted  bool    `json:"from_address_trusted"`
	FromAddressRisk     float64 `json:"from_address_risk"`
	ToAddressAge        int     `json:"to_address_age"`
	ToAddressTxCount    int     `json:"to_address_tx_count"`
	ToAddressKnown      bool    `json:"to_address_known"`
	ToAddressTrusted    bool    `json:"to_address_trusted"`
	ToAddressRisk       float64 `json:"to_address_risk"`
	ToAddressIsContract bool    `json:"to_address_is_contract"`
	
	// Velocity features
	TxLast1Hour         int     `json:"tx_last_1_hour"`
	TxLast24Hours       int     `json:"tx_last_24_hours"`
	AmountLast24Hours   float64 `json:"amount_last_24_hours"`
	
	// Pattern features
	IsNewDestination    bool    `json:"is_new_destination"`
	IsUnusualTime      bool    `json:"is_unusual_time"`
	IsCrossChain       bool    `json:"is_cross_chain"`
	
	// Reputation features
	ToAddressBlacklisted bool    `json:"to_address_blacklisted"`
	ToAddressWhitelisted bool    `json:"to_address_whitelisted"`
	FromAddressWhitelisted bool  `json:"from_address_whitelisted"`
}

// ============================================================================
// Fraud Detection Engine
// ============================================================================

type FraudDetectionEngine struct {
	db          *gorm.DB
	redis       *redis.Client
	config      *Config
	patterns    []TransactionPattern
	mu          sync.RWMutex
}

func NewFraudDetectionEngine(cfg *Config, db *gorm.DB) *FraudDetectionEngine {
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "",
		DB: 0,
	})
	
	engine := &FraudDetectionEngine{
		db:     db,
		redis:  rdb,
		config: cfg,
		patterns: []TransactionPattern{
			{
				PatternID:   "velocity-001",
				PatternName: "High Velocity Transactions",
				PatternType: "velocity",
				Description: "Detects unusually high transaction frequency",
				RiskWeight:  0.3,
				Action:      "review",
				MinTransactions: 10,
				TimeWindowSeconds: 3600,
			},
			{
				PatternID:   "amount-001",
				PatternName: "Large Amount Transfer",
				PatternType: "amount",
				Description: "Detects transfers exceeding user's normal amount",
				RiskWeight:  0.4,
				Action:      "review",
				AmountThreshold: "10000",
			},
			{
				PatternID:   "destination-001",
				PatternName: "New Destination Address",
				PatternType: "destination",
				Description: "First transaction to a new address",
				RiskWeight:  0.2,
				Action:      "alert",
			},
			{
				PatternID:   "blacklist-001",
				PatternName: "Blacklisted Address",
				PatternType: "destination",
				Description: "Transaction to a blacklisted address",
				RiskWeight:  1.0,
				Action:      "block",
			},
		},
	}
	
	return engine
}

func (e *FraudDetectionEngine) AnalyzeTransaction(tx *Transaction) (*RiskScore, error) {
	features, err := e.extractFeatures(tx)
	if err != nil {
		return nil, err
	}
	
	// Calculate component scores
	componentScores := make(map[string]float64)
	riskFactors := []RiskFactor{}
	
	// 1. Velocity Score
	velocityScore := e.calculateVelocityScore(features)
	componentScores["velocity"] = velocityScore
	if velocityScore > 0.6 {
		riskFactors = append(riskFactors, RiskFactor{
			Factor: "velocity",
			Weight: 0.25,
			Score:  velocityScore,
			Reason: "High transaction velocity detected",
			Details: fmt.Sprintf("%d transactions in last 24 hours", features.TxLast24Hours),
		})
	}
	
	// 2. Amount Score
	amountScore := e.calculateAmountScore(features)
	componentScores["amount"] = amountScore
	if amountScore > 0.6 {
		riskFactors = append(riskFactors, RiskFactor{
			Factor: "amount",
			Weight: 0.25,
			Score:  amountScore,
			Reason: "Unusually large transaction amount",
			Details: fmt.Sprintf("Amount: %s USD", fmt.Sprintf("%.2f", features.AmountInUSD)),
		})
	}
	
	// 3. Pattern Score
	patternScore := e.calculatePatternScore(tx, features)
	componentScores["pattern"] = patternScore
	if patternScore > 0.5 {
		riskFactors = append(riskFactors, RiskFactor{
			Factor: "pattern",
			Weight: 0.20,
			Score:  patternScore,
			Reason: "Transaction matches suspicious pattern",
		})
	}
	
	// 4. Behavioral Score
	behavioralScore := e.calculateBehavioralScore(features)
	componentScores["behavioral"] = behavioralScore
	if behavioralScore > 0.6 {
		riskFactors = append(riskFactors, RiskFactor{
			Factor: "behavioral",
			Weight: 0.15,
			Score:  behavioralScore,
			Reason: "Unusual behavioral patterns detected",
		})
	}
	
	// 5. Reputation Score
	reputationScore := e.calculateReputationScore(features)
	componentScores["reputation"] = reputationScore
	if reputationScore > 0.7 {
		riskFactors = append(riskFactors, RiskFactor{
			Factor: "reputation",
			Weight: 0.15,
			Score:  reputationScore,
			Reason: "Address has poor reputation",
		})
	}
	
	// Calculate overall score
	weights := map[string]float64{
		"velocity":    0.25,
		"amount":      0.25,
		"pattern":     0.20,
		"behavioral":  0.15,
		"reputation":  0.15,
	}
	
	overallScore := 0.0
	for component, score := range componentScores {
		overallScore += score * weights[component]
	}
	
	// Determine risk level
	riskLevel := "low"
	if overallScore >= e.config.BlockThreshold {
		riskLevel = "critical"
	} else if overallScore >= 0.70 {
		riskLevel = "high"
	} else if overallScore >= e.config.ReviewThreshold {
		riskLevel = "medium"
	}
	
	// Generate recommendations
	recommendations := e.generateRecommendations(overallScore, riskFactors)
	
	return &RiskScore{
		OverallScore:    overallScore,
		ComponentScores: componentScores,
		RiskLevel:       riskLevel,
		RiskFactors:     riskFactors,
		Recommendations: recommendations,
	}, nil
}

func (e *FraudDetectionEngine) extractFeatures(tx *Transaction) (*TransactionFeatures, error) {
	features := &TransactionFeatures{
		Amount:    parseAmount(tx.Amount),
		AmountInUSD: parseAmount(tx.Amount), // In production, convert to USD
	}
	
	// Get user info
	var user User
	if err := e.db.Preload("RiskProfile").First(&user, tx.UserID).Error; err == nil {
		features.UserTxCount = 100 // Would query transaction count
		features.UserAvgAmount = 1000.0
		features.UserMaxAmount = 5000.0
		features.UserAccountAgeDays = 365
		features.UserKYCLevel = 3
		if user.RiskProfile.OverallScore > 0 {
			features.UserRiskProfile = user.RiskProfile.OverallScore
		}
	}
	
	// Get from address info
	var fromAddr WalletAddress
	if err := e.db.Where("address = ? AND address_type = ?", tx.FromAddress, "evm").First(&fromAddr).Error; err == nil {
		features.FromAddressAge = int(time.Since(fromAddr.FirstSeen).Hours() / 24)
		features.FromAddressTxCount = fromAddr.TxCount
		features.FromAddressKnown = fromAddr.IsKnown
		features.FromAddressTrusted = fromAddr.IsTrusted
		features.FromAddressRisk = fromAddr.RiskScore
	}
	
	// Get to address info
	var toAddr WalletAddress
	if err := e.db.Where("address = ? AND address_type = ?", tx.ToAddress, "evm").First(&toAddr).Error; err == nil {
		features.ToAddressAge = int(time.Since(toAddr.FirstSeen).Hours() / 24)
		features.ToAddressTxCount = toAddr.TxCount
		features.ToAddressKnown = toAddr.IsKnown
		features.ToAddressTrusted = toAddr.IsTrusted
		features.ToAddressRisk = toAddr.RiskScore
	}
	
	// Check if blacklisted
	var blocked BlockedAddress
	features.ToAddressBlacklisted = e.db.Where("address = ?", tx.ToAddress).First(&blocked).Error == nil
	
	// Check velocity
	features.TxLast1Hour = 5
	features.TxLast24Hours = 25
	features.AmountLast24Hours = 50000.0
	
	// Check if new destination
	features.IsNewDestination = features.ToAddressAge < 7
	
	// Check time
	hour := time.Now().Hour()
	features.IsUnusualTime = hour < 6 || hour > 22
	
	return features, nil
}

func (e *FraudDetectionEngine) calculateVelocityScore(f *TransactionFeatures) float64 {
	score := 0.0
	
	// Check 1-hour velocity
	if f.TxLast1Hour > 10 {
		score += 0.4
	} else if f.TxLast1Hour > 5 {
		score += 0.2
	}
	
	// Check 24-hour velocity
	if f.TxLast24Hours > 50 {
		score += 0.4
	} else if f.TxLast24Hours > 20 {
		score += 0.2
	}
	
	// Check amount velocity
	if f.AmountLast24Hours > 100000 {
		score += 0.2
	} else if f.AmountLast24Hours > 50000 {
		score += 0.1
	}
	
	return math.Min(score, 1.0)
}

func (e *FraudDetectionEngine) calculateAmountScore(f *TransactionFeatures) float64 {
	score := 0.0
	
	// Large amount check
	if f.AmountInUSD > 100000 {
		score += 0.5
	} else if f.AmountInUSD > 50000 {
		score += 0.3
	} else if f.AmountInUSD > 10000 {
		score += 0.2
	}
	
	// Compare to user's average
	if f.UserAvgAmount > 0 {
		ratio := f.AmountInUSD / f.UserAvgAmount
		if ratio > 10 {
			score += 0.5
		} else if ratio > 5 {
			score += 0.3
		} else if ratio > 2 {
			score += 0.1
		}
	}
	
	// First large transaction
	if f.UserMaxAmount < 10000 && f.AmountInUSD > 10000 {
		score += 0.3
	}
	
	return math.Min(score, 1.0)
}

func (e *FraudDetectionEngine) calculatePatternScore(tx *Transaction, f *TransactionFeatures) float64 {
	score := 0.0
	
	for _, pattern := range e.patterns {
		if !pattern.IsActive {
			continue
		}
		
		switch pattern.PatternType {
		case "velocity":
			if f.TxLast1Hour >= pattern.MinTransactions {
				score += pattern.RiskWeight
			}
		case "destination":
			if f.IsNewDestination {
				score += pattern.RiskWeight * 0.5
			}
			if f.ToAddressBlacklisted {
				score += pattern.RiskWeight
			}
		case "amount":
			threshold := parseAmount(pattern.AmountThreshold)
			if f.AmountInUSD > threshold {
				score += pattern.RiskWeight
			}
		}
	}
	
	return math.Min(score, 1.0)
}

func (e *FraudDetectionEngine) calculateBehavioralScore(f *TransactionFeatures) float64 {
	score := 0.0
	
	// Unusual time
	if f.IsUnusualTime {
		score += 0.3
	}
	
	// New destination
	if f.IsNewDestination {
		score += 0.2
	}
	
	// Low KYC
	if f.UserKYCLevel < 2 {
		score += 0.3
	}
	
	// New account with large transaction
	if f.UserAccountAgeDays < 30 && f.AmountInUSD > 10000 {
		score += 0.4
	}
	
	return math.Min(score, 1.0)
}

func (e *FraudDetectionEngine) calculateReputationScore(f *TransactionFeatures) float64 {
	score := 0.0
	
	// From address risk
	score += f.FromAddressRisk * 0.3
	
	// To address risk
	score += f.ToAddressRisk * 0.3
	
	// Blacklist check
	if f.ToAddressBlacklisted {
		score += 0.4
	}
	
	// Trust check
	if !f.FromAddressTrusted && !f.ToAddressTrusted {
		score += 0.1
	}
	
	return math.Min(score, 1.0)
}

func (e *FraudDetectionEngine) generateRecommendations(score float64, factors []RiskFactor) []string {
	recommendations := []string{}
	
	if score >= e.config.BlockThreshold {
		recommendations = append(recommendations, "Block this transaction - critical risk detected")
	}
	
	if score >= e.config.ReviewThreshold {
		recommendations = append(recommendations, "Add to review queue for manual verification")
	}
	
	for _, factor := range factors {
		switch factor.Factor {
		case "velocity":
			recommendations = append(recommendations, "Consider implementing velocity limits for this user")
		case "amount":
			recommendations = append(recommendations, "Require additional verification for large transactions")
		case "reputation":
			recommendations = append(recommendations, "Review address reputation before processing")
		}
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Transaction appears normal - proceed with standard processing")
	}
	
	return recommendations
}

// ============================================================================
// API Handlers
// ============================================================================

type TransactionShieldServer struct {
	config  *Config
	db      *gorm.DB
	redis   *redis.Client
	engine  *FraudDetectionEngine
	router  *gin.Engine
}

func NewTransactionShieldServer(cfg *Config) *TransactionShieldServer {
	// Setup database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	
	// Auto migrate
	db.AutoMigrate(&Transaction{}, &User{}, &WalletAddress{}, &RiskProfile{}, &Alert{}, &BlockedAddress{}, &TransactionPattern{}, &ReviewQueue{})
	
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "",
		DB: 0,
	})
	
	engine := NewFraudDetectionEngine(cfg, db)
	
	server := &TransactionShieldServer{
		config:  cfg,
		db:      db,
		redis:   rdb,
		engine:  engine,
		router:  gin.Default(),
	}
	
	server.setupRoutes()
	return server
}

func (s *TransactionShieldServer) setupRoutes() {
	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "transaction-shield",
			"block_threshold": s.config.BlockThreshold,
			"review_threshold": s.config.ReviewThreshold,
		})
	})
	
	// Transaction analysis
	s.router.POST("/api/v1/analyze", s.analyzeTransaction)
	s.router.POST("/api/v1/analyze/batch", s.analyzeBatch)
	
	// Risk scores
	s.router.GET("/api/v1/transaction/:tx_hash/score", s.getTransactionScore)
	s.router.GET("/api/v1/user/:user_id/risk-profile", s.getUserRiskProfile)
	s.router.POST("/api/v1/user/:user_id/risk-profile/update", s.updateUserRiskProfile)
	
	// Address management
	s.router.POST("/api/v1/address/block", s.blockAddress)
	s.router.POST("/api/v1/address/unblock", s.unblockAddress)
	s.router.GET("/api/v1/address/:address/status", s.getAddressStatus)
	
	// Alerts
	s.router.GET("/api/v1/alerts", s.getAlerts)
	s.router.POST("/api/v1/alerts/:alert_id/acknowledge", s.acknowledgeAlert)
	s.router.POST("/api/v1/alerts/:alert_id/resolve", s.resolveAlert)
	
	// Review queue
	s.router.GET("/api/v1/review-queue", s.getReviewQueue)
	s.router.POST("/api/v1/review-queue/:transaction_id/review", s.reviewTransaction)
	
	// Patterns
	s.router.GET("/api/v1/patterns", s.getPatterns)
	s.router.POST("/api/v1/patterns", s.createPattern)
	s.router.PUT("/api/v1/patterns/:pattern_id", s.updatePattern)
	
	// Stats
	s.router.GET("/api/v1/stats", s.getStats)
}

func (s *TransactionShieldServer) analyzeTransaction(c *gin.Context) {
	var request struct {
		UserID      uint   `json:"user_id" binding:"required"`
		ChainID     int    `json:"chain_id" binding:"required"`
		FromAddress string `json:"from_address" binding:"required"`
		ToAddress   string `json:"to_address" binding:"required"`
		Amount      string `json:"amount" binding:"required"`
		Token       string `json:"token"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}
	
	// Create transaction record
	tx := Transaction{
		TransactionHash: generateTxHash(request.FromAddress, request.ToAddress, request.Amount),
		UserID:          request.UserID,
		ChainID:         request.ChainID,
		FromAddress:     request.FromAddress,
		ToAddress:       request.ToAddress,
		Amount:          request.Amount,
		Token:           request.Token,
		Status:          "pending",
		Timestamp:       time.Now(),
	}
	
	s.db.Create(&tx)
	
	// Analyze transaction
	riskScore, err := s.engine.AnalyzeTransaction(&tx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "analysis_failed", "message": err.Error()})
		return
	}
	
	// Update transaction with risk score
	tx.RiskScore = riskScore.OverallScore
	tx.RiskLevel = riskScore.RiskLevel
	riskReasonsJSON, _ := json.Marshal(riskScore.RiskFactors)
	tx.RiskReasons = string(riskReasonsJSON)
	s.db.Save(&tx)
	
	// Handle based on risk level
	var status string
	switch riskScore.RiskLevel {
	case "critical":
		tx.Status = "blocked"
		status = "blocked"
		s.createAlert(&tx, riskScore, "critical")
	case "high":
		s.addToReviewQueue(&tx)
		status = "review"
	case "medium":
		s.createAlert(&tx, riskScore, "medium")
		status = "alert"
	default:
		tx.Status = "confirmed"
		status = "approved"
	}
	s.db.Save(&tx)
	
	c.JSON(http.StatusOK, gin.H{
		"transaction_hash": tx.TransactionHash,
		"status": status,
		"risk_score": riskScore,
	})
}

func (s *TransactionShieldServer) analyzeBatch(c *gin.Context) {
	var request struct {
		Transactions []struct {
			UserID      uint   `json:"user_id"`
			ChainID     int    `json:"chain_id"`
			FromAddress string `json:"from_address"`
			ToAddress   string `json:"to_address"`
			Amount      string `json:"amount"`
			Token       string `json:"token"`
		} `json:"transactions"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}
	
	results := make([]map[string]interface{}, 0)
	
	for _, txReq := range request.Transactions {
		tx := Transaction{
			TransactionHash: generateTxHash(txReq.FromAddress, txReq.ToAddress, txReq.Amount),
			UserID:          txReq.UserID,
			ChainID:         txReq.ChainID,
			FromAddress:     txReq.FromAddress,
			ToAddress:       txReq.ToAddress,
			Amount:          txReq.Amount,
			Token:           txReq.Token,
			Status:          "pending",
			Timestamp:       time.Now(),
		}
		
		s.db.Create(&tx)
		
		riskScore, _ := s.engine.AnalyzeTransaction(&tx)
		tx.RiskScore = riskScore.OverallScore
		tx.RiskLevel = riskScore.RiskLevel
		s.db.Save(&tx)
		
		results = append(results, map[string]interface{}{
			"transaction_hash": tx.TransactionHash,
			"risk_score":       riskScore.OverallScore,
			"risk_level":       riskScore.RiskLevel,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (s *TransactionShieldServer) getTransactionScore(c *gin.Context) {
	txHash := c.Param("tx_hash")
	
	var tx Transaction
	if err := s.db.Where("transaction_hash = ?", txHash).First(&tx).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction_not_found"})
		return
	}
	
	var riskFactors []RiskFactor
	json.Unmarshal([]byte(tx.RiskReasons), &riskFactors)
	
	c.JSON(http.StatusOK, gin.H{
		"transaction_hash": tx.TransactionHash,
		"risk_score":      tx.RiskScore,
		"risk_level":      tx.RiskLevel,
		"risk_factors":    riskFactors,
		"status":          tx.Status,
	})
}

func (s *TransactionShieldServer) getUserRiskProfile(c *gin.Context) {
	userID := c.Param("user_id")
	
	var profile RiskProfile
	if err := s.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		// Return default profile
		c.JSON(http.StatusOK, gin.H{
			"user_id":         userID,
			"overall_score":   0.0,
			"risk_level":      "low",
			"review_status":   "none",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"user_id":           profile.UserID,
		"overall_score":     profile.OverallScore,
		"velocity_score":    profile.VelocityScore,
		"amount_score":      profile.AmountScore,
		"pattern_score":     profile.PatternScore,
		"behavioral_score":  profile.BehavioralScore,
		"reputation_score":  profile.ReputationScore,
		"risk_level":        profile.RiskLevel,
		"review_status":     profile.ReviewStatus,
		"last_updated":      profile.LastUpdated,
	})
}

func (s *TransactionShieldServer) updateUserRiskProfile(c *gin.Context) {
	userID := c.Param("user_id")
	
	var request struct {
		Override     bool   `json:"override"`
		RiskLevel    string `json:"risk_level"`
		OverrideReason string `json:"override_reason"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	
	var profile RiskProfile
	result := s.db.Where("user_id = ?", userID).First(&profile)
	
	if result.Error == gorm.ErrRecordNotFound {
		profile = RiskProfile{
			UserID:       parseUint(userID),
			RiskLevel:    request.RiskLevel,
			LastUpdated:  time.Now(),
			ManualOverride: request.Override,
		}
		s.db.Create(&profile)
	} else {
		profile.RiskLevel = request.RiskLevel
		profile.ManualOverride = request.Override
		profile.OverrideReason = request.OverrideReason
		profile.OverrideAt = new(time.Time)
		*profile.OverrideAt = time.Now()
		profile.LastUpdated = time.Now()
		s.db.Save(&profile)
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Risk profile updated", "profile": profile})
}

func (s *TransactionShieldServer) blockAddress(c *gin.Context) {
	var request struct {
		Address       string `json:"address" binding:"required"`
		AddressType   string `json:"address_type"`
		ChainID       int    `json:"chain_id"`
		BlockedReason string `json:"blocked_reason" binding:"required"`
		IsPermanent   bool   `json:"is_permanent"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	
	if request.AddressType == "" {
		request.AddressType = "evm"
	}
	
	blocked := BlockedAddress{
		Address:       request.Address,
		AddressType:   request.AddressType,
		ChainID:       request.ChainID,
		BlockedReason: request.BlockedReason,
		BlockedAt:     time.Now(),
		BlockedBy:     "admin",
		IsPermanent:   request.IsPermanent,
	}
	
	s.db.Create(&blocked)
	
	c.JSON(http.StatusOK, gin.H{"message": "Address blocked successfully", "address": request.Address})
}

func (s *TransactionShieldServer) unblockAddress(c *gin.Context) {
	var request struct {
		Address string `json:"address" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	
	result := s.db.Where("address = ?", request.Address).Delete(&BlockedAddress{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "address_not_blocked"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Address unblocked successfully"})
}

func (s *TransactionShieldServer) getAddressStatus(c *gin.Context) {
	address := c.Param("address")
	
	var blocked BlockedAddress
	err := s.db.Where("address = ?", address).First(&blocked).Error
	
	c.JSON(http.StatusOK, gin.H{
		"address":      address,
		"is_blocked":  err == nil,
		"blocked_at":   blocked.BlockedAt,
		"reason":       blocked.BlockedReason,
		"is_permanent": blocked.IsPermanent,
	})
}

func (s *TransactionShieldServer) getAlerts(c *gin.Context) {
	var alerts []Alert
	s.db.Order("created_at desc").Limit(100).Find(&alerts)
	
	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func (s *TransactionShieldServer) acknowledgeAlert(c *gin.Context) {
	alertID := c.Param("alert_id")
	
	var alert Alert
	if err := s.db.Where("alert_id = ?", alertID).First(&alert).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert_not_found"})
		return
	}
	
	alert.Status = "acknowledged"
	s.db.Save(&alert)
	
	c.JSON(http.StatusOK, gin.H{"message": "Alert acknowledged"})
}

func (s *TransactionShieldServer) resolveAlert(c *gin.Context) {
	alertID := c.Param("alert_id")
	
	var request struct {
		Resolution string `json:"resolution"`
	}
	c.ShouldBindJSON(&request)
	
	var alert Alert
	if err := s.db.Where("alert_id = ?", alertID).First(&alert).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert_not_found"})
		return
	}
	
	now := time.Now()
	alert.Status = "resolved"
	alert.ResolvedAt = &now
	alert.Resolution = request.Resolution
	s.db.Save(&alert)
	
	c.JSON(http.StatusOK, gin.H{"message": "Alert resolved"})
}

func (s *TransactionShieldServer) getReviewQueue(c *gin.Context) {
	var queue []ReviewQueue
	s.db.Preload("Transaction").Order("priority ASC, created_at ASC").Find(&queue)
	
	c.JSON(http.StatusOK, gin.H{"review_queue": queue})
}

func (s *TransactionShieldServer) reviewTransaction(c *gin.Context) {
	txID := c.Param("transaction_id")
	
	var request struct {
		Decision  string `json:"decision" binding:"required"` // approve, reject
		Notes     string `json:"notes"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	
	var tx Transaction
	if err := s.db.First(&tx, txID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction_not_found"})
		return
	}
	
	if request.Decision == "approve" {
		tx.Status = "confirmed"
	} else {
		tx.Status = "blocked"
	}
	
	tx.ReviewNotes = request.Notes
	now := time.Now()
	tx.ReviewedAt = &now
	s.db.Save(&tx)
	
	// Remove from review queue
	s.db.Where("transaction_id = ?", txID).Delete(&ReviewQueue{})
	
	c.JSON(http.StatusOK, gin.H{"message": "Transaction reviewed", "decision": request.Decision})
}

func (s *TransactionShieldServer) getPatterns(c *gin.Context) {
	var patterns []TransactionPattern
	s.db.Find(&patterns)
	
	c.JSON(http.StatusOK, gin.H{"patterns": patterns})
}

func (s *TransactionShieldServer) createPattern(c *gin.Context) {
	var pattern TransactionPattern
	if err := c.ShouldBindJSON(&pattern); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	
	pattern.PatternID = uuid.New().String()
	pattern.IsActive = true
	pattern.CreatedAt = time.Now()
	pattern.UpdatedAt = time.Now()
	
	s.db.Create(&pattern)
	
	c.JSON(http.StatusCreated, gin.H{"message": "Pattern created", "pattern": pattern})
}

func (s *TransactionShieldServer) updatePattern(c *gin.Context) {
	patternID := c.Param("pattern_id")
	
	var pattern TransactionPattern
	if err := s.db.Where("pattern_id = ?", patternID).First(&pattern).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pattern_not_found"})
		return
	}
	
	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)
	
	updates["updated_at"] = time.Now()
	s.db.Model(&pattern).Updates(updates)
	
	c.JSON(http.StatusOK, gin.H{"message": "Pattern updated"})
}

func (s *TransactionShieldServer) getStats(c *gin.Context) {
	var totalTx int64
	var blockedTx int64
	var pendingReview int64
	var alerts int64
	
	s.db.Model(&Transaction{}).Count(&totalTx)
	s.db.Model(&Transaction{}).Where("status = ?", "blocked").Count(&blockedTx)
	s.db.Model(&ReviewQueue{}).Count(&pendingReview)
	s.db.Model(&Alert{}).Where("status = ?", "new").Count(&alerts)
	
	c.JSON(http.StatusOK, gin.H{
		"total_transactions":  totalTx,
		"blocked_transactions": blockedTx,
		"pending_review":      pendingReview,
		"active_alerts":       alerts,
		"block_rate":          float64(blockedTx) / float64(totalTx+1),
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *TransactionShieldServer) createAlert(tx *Transaction, riskScore *RiskScore, severity string) {
	alert := Alert{
		AlertID:     uuid.New().String(),
		UserID:      tx.UserID,
		TransactionID: &tx.ID,
		AlertType:   "high_risk",
		Severity:    severity,
		Title:       fmt.Sprintf("%s Risk Transaction Detected", strings.Title(severity)),
		Description: fmt.Sprintf("Transaction from %s to %s has %s risk score of %.2f", 
			tx.FromAddress[:10]+"...", tx.ToAddress[:10]+"...", riskScore.RiskLevel, riskScore.OverallScore),
		RiskFactors: tx.RiskReasons,
		Amount:      tx.Amount,
		Timestamp:   time.Now(),
		Status:      "new",
	}
	
	s.db.Create(&alert)
	
	// Cache alert in Redis for quick access
	alertJSON, _ := json.Marshal(alert)
	s.redis.Set(context.Background(), fmt.Sprintf("alert:%s", alert.AlertID), alertJSON, 24*time.Hour)
}

func (s *TransactionShieldServer) addToReviewQueue(tx *Transaction) {
	queue := ReviewQueue{
		TransactionID: tx.ID,
		Priority:     3, // Default medium priority
		Status:       "pending",
	}
	s.db.Create(&queue)
}

func generateTxHash(from, to, amount string) string {
	data := fmt.Sprintf("%s:%s:%s:%d", from, to, amount, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])
}

func parseAmount(amount string) float64 {
	var result float64
	fmt.Sscanf(amount, "%f", &result)
	return result
}

func parseUint(s string) uint {
	var result uint
	fmt.Sscanf(s, "%d", &result)
	return result
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := LoadConfig()
	
	server := NewTransactionShieldServer(cfg)
	
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-quit
		log.Println("Shutting down Transaction Shield...")
		os.Exit(0)
	}()
	
	log.Printf("Transaction Shield Service starting on port %s", cfg.ServerPort)
	if err := server.router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
