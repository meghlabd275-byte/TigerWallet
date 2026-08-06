// Fraud Detection Service
// Real-time fraud detection and prevention

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// FraudConfig - Fraud Detection Configuration
type FraudConfig struct {
	// Detection Settings
	MaxTransactionsPerHour int     `json:"max_transactions_per_hour"`
	MaxWithdrawalPerDay  float64 `json:"max_withdrawal_per_day"`
	HighRiskCountries    string   `json:"high_risk_countries"` // JSON array
	EnableVelocityCheck  bool     `json:"enable_velocity_check"`
	EnablePatternMatch  bool     `json:"enable_pattern_match"`
	EnableGeoAnomaly    bool     `json:"enable_geo_anomaly"`
	
	// Scoring Weights
	VelocityWeight float64 `json:"velocity_weight"`
	PatternWeight  float64 `json:"pattern_weight"`
	GeoWeight      float64 `json:"geo_weight"`
	HistoryWeight  float64 `json:"history_weight"`
	
	// Database Settings
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	
	// Redis Settings
	RedisHost string `json:"redis_host"`
	RedisPort string `json:"redis_port"`
	
	// Server
	ServerPort string `json:"server_port"`
}

// FraudRule - Fraud detection rule
type FraudRule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RuleID     string    `gorm:"uniqueIndex" json:"rule_id"`
	Name       string    `json:"name"`
	RuleType   string    `json:"rule_type"` // velocity, pattern, geo, amount, frequency
	Expression string    `gorm:"type:text" json:"expression"`
	Threshold  float64   `json:"threshold"`
	Severity   string    `json:"severity"` // low, medium, high, critical
	Action     string    `json:"action"` // alert, block, review
	IsEnabled  bool      `gorm:"default:true" json:"is_enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// FraudAlert - Fraud alert
type FraudAlert struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AlertID    string    `gorm:"uniqueIndex" json:"alert_id"`
	UserID     string    `gorm:"index" json:"user_id"`
	RuleID     string    `gorm:"index" json:"rule_id"`
	Severity   string    `json:"severity"`
	Status     string    `json:"status"` // new, reviewing, resolved, false_positive
	Title      string    `json:"title"`
	Description string  `json:"description"`
	Evidence   string    `gorm:"type:jsonb" json:"evidence"`
	Score      float64   `json:"score"`
	ResolvedBy string    `json:"resolved_by"`
	ResolvedAt *time.Time `json:"resolved_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// UserBehavior - User behavior tracking
type UserBehavior struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"index" json:"user_id"`
	EventType   string    `json:"event_type"` // login, transaction, withdrawal, kyc
	IPAddress  string    `json:"ip_address"`
	Country    string    `json:"country"`
	Device     string    `json:"device"`
	Amount     float64   `json:"amount"`
	Currency   string    `json:"currency"`
	Metadata   string    `gorm:"type:jsonb" json:"metadata"`
	CreatedAt  time.Time `json:"created_at"`
}

// BlockedUser - Blocked user
type BlockedUser struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID     string    `gorm:"uniqueIndex" json:"user_id"`
	Reason     string    `json:"reason"`
	BlockType  string    `json:"block_type"` // fraud, manual, system
	Severity   string    `json:"severity"` // temp, permanent
	ExpiresAt  *time.Time `json:"expires_at"`
	BlockedBy  string    `json:"blocked_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// FraudDetectionService - Main fraud detection service
type FraudDetectionService struct {
	config FraudConfig
	db     *gorm.DB
	redis *redis.Client
}

// NewFraudDetectionService - Create new fraud detection service
func NewFraudDetectionService(cfg FraudConfig) (*FraudDetectionService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	err = db.AutoMigrate(&FraudRule{}, &FraudAlert{}, &UserBehavior{}, &BlockedUser{})
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
	
	// Set defaults
	if cfg.MaxTransactionsPerHour == 0 {
		cfg.MaxTransactionsPerHour = 50
	}
	if cfg.MaxWithdrawalPerDay == 0 {
		cfg.MaxWithdrawalPerDay = 100000
	}
	
	// Seed default rules
	service := &FraudDetectionService{
		config: cfg,
		db:     db,
		redis: rdb,
	}
	
	service.seedDefaultRules()
	
	return service, nil
}

// seedDefaultRules - Seed default fraud detection rules
func (s *FraudDetectionService) seedDefaultRules() {
	rules := []FraudRule{
		{
			RuleID: "velocity_1", Name: "High Transaction Velocity", RuleType: "velocity",
			Expression: "transactions_per_hour > 50", Threshold: 50, Severity: "high", Action: "alert",
		},
		{
			RuleID: "amount_1", Name: "High Value Withdrawal", RuleType: "amount",
			Expression: "withdrawal_amount > 10000", Threshold: 10000, Severity: "critical", Action: "block",
		},
		{
			RuleID: "geo_1", Name: "High Risk Country", RuleType: "geo",
			Expression: "country in high_risk_countries", Severity: "high", Action: "review",
		},
		{
			RuleID: "pattern_1", Name: "Unusual Pattern", RuleType: "pattern",
			Expression: "unusual_transaction_pattern", Severity: "medium", Action: "alert",
		},
		{
			RuleID: "frequency_1", Name: "Rapid Withdrawals", RuleType: "frequency",
			Expression: "withdrawals_per_hour > 5", Threshold: 5, Severity: "high", Action: "block",
		},
	}
	
	for _, rule := range rules {
		var existing FraudRule
		if s.db.Where("rule_id = ?", rule.RuleID).First(&existing).Error != nil {
			s.db.Create(&rule)
		}
	}
}

// AnalyzeTransaction - Analyze transaction for fraud
func (s *FraudDetectionService) AnalyzeTransaction(userID, txType, ipAddress, country, device string, amount float64, currency string) (*FraudResult, error) {
	result := &FraudResult{
		UserID:   userID,
		Score:    0,
		Risk:     "low",
		Decision: "allow",
		Reasons: []string{},
		Alerts:  []string{},
	}
	
	// Check if user is blocked
	var blocked BlockedUser
	if s.db.Where("user_id = ?", userID).First(&blocked).Error == nil {
		if blocked.ExpiresAt == nil || blocked.ExpiresAt.After(time.Now()) {
			result.Decision = "block"
			result.Risk = "critical"
			result.Reasons = append(result.Reasons, "User is blocked")
			return result, nil
		}
	}
	
	// Velocity check
	if s.config.EnableVelocityCheck {
		velocityScore, velocityReason := s.checkVelocity(userID)
		result.Score += velocityScore * s.config.VelocityWeight
		if velocityReason != "" {
			result.Reasons = append(result.Reasons, velocityReason)
		}
	}
	
	// Amount check
	amountScore, amountReason := s.checkAmount(amount)
	result.Score += amountScore * 1.0 // Amount weight
	if amountReason != "" {
		result.Reasons = append(result.Reasons, amountReason)
	}
	
	// Geo check
	if s.config.EnableGeoAnomaly {
		geoScore, geoReason := s.checkGeo(country)
		result.Score += geoScore * s.config.GeoWeight
		if geoReason != "" {
			result.Reasons = append(result.Reasons, geoReason)
		}
	}
	
	// Pattern check
	if s.config.EnablePatternMatch {
		patternScore, patternReason := s.checkPattern(userID, txType)
		result.Score += patternScore * s.config.PatternWeight
		if patternReason != "" {
			result.Reasons = append(result.Reasons, patternReason)
		}
	}
	
	// History check
	historyScore, historyReason := s.checkHistory(userID)
	result.Score += historyScore * s.config.HistoryWeight
	if historyReason != "" {
		result.Reasons = append(result.Reasons, historyReason)
	}
	
	// Determine risk level
	if result.Score >= 80 {
		result.Risk = "critical"
		result.Decision = "block"
		result.Alerts = append(result.Alerts, "Critical risk - transaction blocked")
	} else if result.Score >= 50 {
		result.Risk = "high"
		result.Decision = "review"
		result.Alerts = append(result.Alerts, "High risk - manual review required")
	} else if result.Score >= 20 {
		result.Risk = "medium"
		result.Decision = "allow"
		result.Alerts = append(result.Alerts, "Medium risk - flagged for monitoring")
	}
	
	// Log behavior
	s.logBehavior(userID, txType, ipAddress, country, device, amount, currency)
	
	// Create alert if high risk
	if result.Risk == "high" || result.Risk == "critical" {
		s.createAlert(userID, result)
	}
	
	return result, nil
}

// FraudResult - Fraud detection result
type FraudResult struct {
	UserID   string   `json:"user_id"`
	Score    float64  `json:"score"`
	Risk     string   `json:"risk"` // low, medium, high, critical
	Decision string   `json:"decision"` // allow, review, block
	Reasons  []string `json:"reasons"`
	Alerts   []string `json:"alerts"`
}

func (s *FraudDetectionService) checkVelocity(userID string) (float64, string) {
	// Check transaction count in Redis
	key := fmt.Sprintf("fraud:velocity:%s", userID)
	count, err := s.redis.Get(context.Background(), key).Int()
	if err != nil {
		count = 0
	}
	
	// Increment counter
	s.redis.Incr(context.Background(), key)
	s.redis.Expire(context.Background(), key, time.Hour)
	
	if count >= s.config.MaxTransactionsPerHour {
		return 80, fmt.Sprintf("High velocity: %d transactions per hour", count)
	}
	
	if count > 30 {
		return 40, fmt.Sprintf("Elevated velocity: %d transactions per hour", count)
	}
	
	return 0, ""
}

func (s *FraudDetectionService) checkAmount(amount float64) (float64, string) {
	if amount > s.config.MaxWithdrawalPerDay {
		return 100, fmt.Sprintf("Exceeds daily withdrawal limit: %.2f", amount)
	}
	
	if amount > 50000 {
		return 60, fmt.Sprintf("Very high amount: %.2f", amount)
	}
	
	if amount > 10000 {
		return 30, fmt.Sprintf("High amount: %.2f", amount)
	}
	
	return 0, ""
}

func (s *FraudDetectionService) checkGeo(country string) (float64, string) {
	highRiskCountries := []string{"NK", "IR", "SY", "CU", "RU", "BY"}
	
	for _, c := range highRiskCountries {
		if country == c {
			return 70, fmt.Sprintf("High risk country: %s", country)
		}
	}
	
	return 0, ""
}

func (s *FraudDetectionService) checkPattern(userID, txType string) (float64, string) {
	// Check for unusual patterns
	key := fmt.Sprintf("fraud:pattern:%s:%s", userID, txType)
	count, err := s.redis.Get(context.Background(), key).Int()
	if err != nil {
		count = 0
	}
	
	if count == 0 {
		// First transaction of this type
		s.redis.Set(context.Background(), key, 1, 24*time.Hour)
		return 10, "First transaction of this type"
	}
	
	return 0, ""
}

func (s *FraudDetectionService) checkHistory(userID string) (float64, string) {
	// Check user's fraud history
	var alertCount int64
	s.db.Model(&FraudAlert{}).Where("user_id = ? AND status != ?", userID, "resolved").Count(&alertCount)
	
	if alertCount >= 3 {
		return 90, fmt.Sprintf("Previous fraud alerts: %d", alertCount)
	}
	
	if alertCount >= 1 {
		return 40, fmt.Sprintf("Previous fraud alert: %d", alertCount)
	}
	
	return 0, ""
}

func (s *FraudDetectionService) logBehavior(userID, eventType, ipAddress, country, device string, amount float64, currency string) {
	behavior := &UserBehavior{
		UserID:    userID,
		EventType: eventType,
		IPAddress: ipAddress,
		Country:   country,
		Device:    device,
		Amount:    amount,
		Currency:  currency,
		CreatedAt: time.Now(),
	}
	s.db.Create(behavior)
}

func (s *FraudDetectionService) createAlert(userID string, result *FraudResult) {
	alert := &FraudAlert{
		AlertID:     fmt.Sprintf("FA-%d-%s", time.Now().Unix(), userID[:8]),
		UserID:      userID,
		Severity:    result.Risk,
		Status:      "new",
		Title:       fmt.Sprintf("Fraud Risk: %s", result.Risk),
		Description: strings.Join(result.Reasons, "; "),
		Evidence:    "",
		Score:       result.Score,
		CreatedAt:   time.Now(),
	}
	
	evidences, _ := json.Marshal(result)
	alert.Evidence = string(evidences)
	
	s.db.Create(alert)
}

// GetAlerts - Get fraud alerts
func (s *FraudDetectionService) GetAlerts(userID string, status string, limit int) ([]FraudAlert, error) {
	query := s.db
	
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	
	if status != "" {
		query = query.Where("status = ?", status)
	}
	
	if limit == 0 {
		limit = 100
	}
	
	var alerts []FraudAlert
	err := query.Order("created_at DESC").Limit(limit).Find(&alerts).Error
	return alerts, err
}

// ResolveAlert - Resolve fraud alert
func (s *FraudAlert) Resolve(resolvedBy, resolution string) error {
	now := time.Now()
	return gorm.Model(FraudAlert{}).Where("id = ?", FraudAlert.ID).Updates(map[string]interface{}{
		"status":      resolution,
		"resolved_by": resolvedBy,
		"resolved_at": now,
	}).Error
}

// BlockUser - Block user
func (s *FraudDetectionService) BlockUser(userID, reason, blockType, severity string, expiresAt *time.Time, blockedBy string) error {
	block := &BlockedUser{
		UserID:    userID,
		Reason:   reason,
		BlockType: blockType,
		Severity:  severity,
		ExpiresAt: expiresAt,
		BlockedBy: blockedBy,
		CreatedAt: time.Now(),
	}
	
	return s.db.Create(block).Error
}

// UnblockUser - Unblock user
func (s *FraudDetectionService) UnblockUser(userID string) error {
	return s.db.Where("user_id = ?", userID).Delete(&BlockedUser{}).Error
}

// HTTP Handlers

type AnalyzeRequest struct {
	UserID   string  `json:"user_id" binding:"required"`
	TxType  string  `json:"tx_type" binding:"required"`
	IPAddress string `json:"ip_address"`
	Country  string  `json:"country"`
	Device   string  `json:"device"`
	Amount   float64 `json:"amount" binding:"required"`
	Currency string  `json:"currency"`
}

func (s *FraudDetectionService) AnalyzeHandler(c *gin.Context) {
	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	result, err := s.AnalyzeTransaction(
		req.UserID, req.TxType, req.IPAddress, 
		req.Country, req.Device, req.Amount, req.Currency,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, result)
}

func (s *FraudDetectionService) GetAlertsHandler(c *gin.Context) {
	userID := c.Query("user_id")
	status := c.Query("status")
	
	alerts, err := s.GetAlerts(userID, status, 100)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"alerts": alerts})
}

type ResolveAlertRequest struct {
	AlertID    string `json:"alert_id" binding:"required"`
	Resolution string `json:"resolution" binding:"required"` // resolved, false_positive
	ResolvedBy string `json:"resolved_by"`
}

func (s *FraudDetectionService) ResolveAlertHandler(c *gin.Context) {
	var req ResolveAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	var alert FraudAlert
	if err := s.db.Where("alert_id = ?", req.AlertID).First(&alert).Error; err != nil {
		c.JSON(404, gin.H{"error": "alert not found"})
		return
	}
	
	alert.Status = req.Resolution
	alert.ResolvedBy = req.ResolvedBy
	now := time.Now()
	alert.ResolvedAt = &now
	
	s.db.Save(&alert)
	
	c.JSON(200, gin.H{"status": "resolved"})
}

type BlockUserRequest struct {
	UserID    string     `json:"user_id" binding:"required"`
	Reason   string     `json:"reason" binding:"required"`
	Severity string     `json:"severity"` // temp, permanent
	ExpiresAt *time.Time `json:"expires_at"`
}

func (s *FraudDetectionService) BlockUserHandler(c *gin.Context) {
	var req BlockUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	severity := req.Severity
	if severity == "" {
		severity = "permanent"
	}
	
	err := s.BlockUser(req.UserID, req.Reason, "fraud", severity, req.ExpiresAt, "system")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "blocked"})
}

func (s *FraudDetectionService) UnblockUserHandler(c *gin.Context) {
	userID := c.Param("user_id")
	
	err := s.UnblockUser(userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "unblocked"})
}

// Main

func main() {
	cfg := FraudConfig{
		MaxTransactionsPerHour: getEnvInt("FRAUD_MAX_TX_HOUR", 50),
		MaxWithdrawalPerDay:   getEnvFloat("FRAUD_MAX_WITHDRAWAL", 100000),
		HighRiskCountries:    getEnv("FRAUD_HIGH_RISK_COUNTRIES", `["NK","IR","SY","CU"]`),
		EnableVelocityCheck:  getEnvBool("FRAUD_ENABLE_VELOCITY", true),
		EnablePatternMatch:   getEnvBool("FRAUD_ENABLE_PATTERN", true),
		EnableGeoAnomaly:    getEnvBool("FRAUD_ENABLE_GEO", true),
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "postgres"),
		DBPassword:          getEnv("DB_PASSWORD", "password"),
		DBName:              getEnv("DB_NAME", "fraud_db"),
		RedisHost:           getEnv("REDIS_HOST", "localhost"),
		RedisPort:           getEnv("REDIS_PORT", "6379"),
		ServerPort:          getEnv("FRAUD_SERVER_PORT", "8098"),
	}
	
	service, err := NewFraudDetectionService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize fraud detection service: %v", err)
	}
	
	r := gin.Default()
	
	r.POST("/analyze", service.AnalyzeHandler)
	r.GET("/alerts", service.GetAlertsHandler)
	r.POST("/alerts/resolve", service.ResolveAlertHandler)
	r.POST("/block", service.BlockUserHandler)
	r.DELETE("/block/:user_id", service.UnblockUserHandler)
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "fraud-detection"})
	})
	
	log.Printf("Fraud Detection Service starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		fmt.Sscanf(value, "%d", &i)
		return i
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		var f float64
		fmt.Sscanf(value, "%f", &f)
		return f
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}
