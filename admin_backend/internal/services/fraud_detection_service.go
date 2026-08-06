package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"admin_backend/internal/models"
	"admin_backend/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// FraudDetectionService handles fraud detection
type FraudDetectionService struct {
	db    *database.PostgresDB
	redis *redis.Client
}

// NewFraudDetectionService creates a new fraud detection service
func NewFraudDetectionService(db *database.PostgresDB, redis *redis.Client) *FraudDetectionService {
	return &FraudDetectionService{
		db:    db,
		redis: redis,
	}
}

// FraudCheckResult represents the result of a fraud check
type FraudCheckResult struct {
	IsSuspicious  bool                   `json:"is_suspicious"`
	RiskScore     int                    `json:"risk_score"` // 0-100
	TriggeredRules []string              `json:"triggered_rules"`
	Recommendations []string             `json:"recommendations"`
	Action        string                 `json:"action"` // allow, block, review, limit
	Metadata      map[string]interface{} `json:"metadata"`
}

// CheckTransactionForFraud checks a transaction for fraud
func (s *FraudDetectionService) CheckTransactionForFraud(userID uint, txAmount float64, txType, chain string, metadata map[string]interface{}) (*FraudCheckResult, error) {
	result := &FraudCheckResult{
		IsSuspicious:   false,
		RiskScore:      0,
		TriggeredRules: []string{},
		Recommendations: []string{},
		Action:         "allow",
		Metadata:       metadata,
	}

	// Get active fraud detection rules
	var rules []models.FraudDetection
	err := s.db.Where("is_active = ?", true).Find(&rules).Error
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	for _, rule := range rules {
		triggered, score := s.evaluateRule(ctx, rule, userID, txAmount, txType, chain, metadata)
		if triggered {
			result.TriggeredRules = append(result.TriggeredRules, rule.Name)
			result.RiskScore += score

			switch rule.Action {
			case "block":
				result.IsSuspicious = true
				result.Action = "block"
			case "alert":
				result.Recommendations = append(result.Recommendations, fmt.Sprintf("Alert: %s", rule.Description))
			case "review":
				if result.Action != "block" {
					result.Action = "review"
				}
			case "limit":
				if result.Action == "allow" {
					result.Action = "limit"
					result.Recommendations = append(result.Recommendations, fmt.Sprintf("Transaction limit applied: %s", rule.Description))
				}
			}

			// Send notification if configured
			if rule.Notification {
				s.sendFraudAlert(ctx, userID, rule, metadata)
			}
		}
	}

	// Determine final action based on risk score
	if result.RiskScore >= 80 {
		result.IsSuspicious = true
		result.Action = "block"
	} else if result.RiskScore >= 50 {
		if result.Action == "allow" {
			result.Action = "review"
		}
	}

	// Create fraud alert if suspicious
	if result.IsSuspicious {
		s.createFraudAlert(userID, result, metadata)
	}

	return result, nil
}

// evaluateRule evaluates a single fraud detection rule
func (s *FraudDetectionService) evaluateRule(ctx context.Context, rule models.FraudDetection, userID uint, amount float64, txType, chain string, metadata map[string]interface{}) (bool, int) {
	var condition map[string]interface{}
	json.Unmarshal([]byte(rule.Condition), &condition)

	switch rule.Type {
	case "velocity":
		return s.checkVelocity(ctx, userID, condition, txType)
	case "amount":
		return s.checkAmount(amount, condition)
	case "geographic":
		return s.checkGeographic(ctx, userID, condition, metadata)
	case "pattern":
		return s.checkPattern(ctx, userID, condition, metadata)
	case "time_based":
		return s.checkTimeBased(ctx, userID, condition)
	}

	return false, 0
}

// checkVelocity checks for unusual transaction velocity
func (s *FraudDetectionService) checkVelocity(ctx context.Context, userID uint, condition map[string]interface{}, txType string) (bool, int) {
	maxTransactions, _ := condition["max_transactions"].(float64)
	windowMinutes, _ := condition["window_minutes"].(float64)

	if maxTransactions == 0 {
		maxTransactions = 10
	}
	if windowMinutes == 0 {
		windowMinutes = 60
	}

	// Count transactions in time window
	key := fmt.Sprintf("fraud:velocity:%d:%s", userID, txType)
	count, _ := s.redis.Get(ctx, key).Int()

	// Increment counter
	s.redis.Incr(ctx, key)
	s.redis.Expire(ctx, key, time.Duration(windowMinutes)*time.Minute)

	if count >= int(maxTransactions) {
		return true, 30
	}

	return false, 0
}

// checkAmount checks for unusual transaction amounts
func (s *FraudDetectionService) checkAmount(amount float64, condition map[string]interface{}) (bool, int) {
	maxAmount, _ := condition["max_amount"].(float64)
	minAmount, _ := condition["min_amount"].(float64)
	multiplier, _ := condition["multiplier"].(float64) // compared to average

	// Get user's average transaction amount
	avgAmount := s.getUserAverageTransaction(amount)

	if maxAmount > 0 && amount > maxAmount {
		return true, 40
	}

	if minAmount > 0 && amount < minAmount {
		return true, 10
	}

	if multiplier > 0 && avgAmount > 0 && amount > avgAmount*multiplier {
		return true, 25
	}

	return false, 0
}

// checkGeographic checks for unusual geographic patterns
func (s *FraudDetectionService) checkGeographic(ctx context.Context, userID uint, condition map[string]interface{}, metadata map[string]interface{}) (bool, int) {
	blockedCountries, _ := condition["blocked_countries"].([]interface{})
	blockedRegions, _ := condition["blocked_regions"].([]interface{})

	ip := metadata["ip"].(string)
	country := metadata["country"].(string)
	region := metadata["region"].(string)

	// Check if from blocked country
	for _, c := range blockedCountries {
		if country == c.(string) {
			return true, 50
		}
	}

	// Check if from blocked region
	for _, r := range blockedRegions {
		if region == r.(string) {
			return true, 40
		}
	}

	// Check for VPN/proxy
	isVPN, _ := metadata["is_vpn"].(bool)
	if isVPN == true {
		return true, 20
	}

	return false, 0
}

// checkPattern checks for suspicious transaction patterns
func (s *FraudDetectionService) checkPattern(ctx context.Context, userID uint, condition map[string]interface{}, metadata map[string]interface{}) (bool, int) {
	// Check for round amount transactions
	amount, _ := metadata["amount"].(float64)
	isRoundAmount := amount == math.Floor(amount)

	// Check for timing patterns
	hour := time.Now().Hour()
	isUnusualHour := hour < 6 || hour > 22

	// Check for new account
	isNewAccount, _ := metadata["is_new_account"].(bool)

	if isRoundAmount && isUnusualHour && isNewAccount {
		return true, 35
	}

	return false, 0
}

// checkTimeBased checks for time-based suspicious activity
func (s *FraudDetectionService) checkTimeBased(ctx context.Context, userID uint, condition map[string]interface{}) (bool, int) {
	// Check if user is active at unusual times
	hour := time.Now().Hour()
	allowedHours, _ := condition["allowed_hours"].(map[string]interface{})
	
	if allowedHours != nil {
		startHour, _ := allowedHours["start"].(float64)
		endHour, _ := allowedHours["end"].(float64)
		
		if startHour > endHour {
			// Overnight allowed (e.g., 22-6)
			if hour < int(startHour) && hour > int(endHour) {
				return true, 15
			}
		} else {
			if hour < int(startHour) || hour > int(endHour) {
				return true, 15
			}
		}
	}

	return false, 0
}

func (s *FraudDetectionService) getUserAverageTransaction(currentAmount float64) float64 {
	// In production, calculate from actual user history
	// For now, return the current amount as a baseline
	return currentAmount * 0.8
}

// createFraudAlert creates a fraud alert in the database
func (s *FraudDetectionService) createFraudAlert(userID uint, result *FraudCheckResult, metadata map[string]interface{}) {
	metadataJSON, _ := json.Marshal(metadata)

	alert := models.FraudAlert{
		UserID:      userID,
		Type:        "transaction",
		Description: fmt.Sprintf("Triggered rules: %v", result.TriggeredRules),
		Status:      "pending",
		Metadata:   metadataJSON,
	}

	s.db.Create(&alert)
}

// sendFraudAlert sends a notification for fraud alerts
func (s *FraudDetectionService) sendFraudAlert(ctx context.Context, userID uint, rule models.FraudDetection, metadata map[string]interface{}) {
	// Send to Slack, PagerDuty, etc.
	alertData := map[string]interface{}{
		"user_id":    userID,
		"rule":       rule.Name,
		"severity":   rule.Severity,
		"description": rule.Description,
		"metadata":   metadata,
		"timestamp":  time.Now().Unix(),
	}

	alertJSON, _ := json.Marshal(alertData)
	s.redis.Publish(ctx, "fraud_alerts", alertJSON)
}

// Manage Fraud Detection Rules

// GetFraudRules gets all fraud detection rules
func (s *FraudDetectionService) GetFraudRules() ([]models.FraudDetection, error) {
	var rules []models.FraudDetection
	err := s.db.Order("created_at DESC").Find(&rules).Error
	return rules, err
}

// GetActiveFraudRules gets only active fraud detection rules
func (s *FraudDetectionService) GetActiveFraudRules() ([]models.FraudDetection, error) {
	var rules []models.FraudDetection
	err := s.db.Where("is_active = ?", true).Find(&rules).Error
	return rules, err
}

// CreateFraudRule creates a new fraud detection rule
func (s *FraudDetectionService) CreateFraudRule(c *gin.Context, rule models.FraudDetection) (*models.FraudDetection, error) {
	adminID := c.GetUint("admin_id")

	err := s.db.Create(&rule).Error
	if err != nil {
		return nil, err
	}

	logAdminActivity(s.db, adminID, "create_fraud_rule", "fraud_detection", 
		fmt.Sprintf("%d", rule.ID), "Created fraud rule: "+rule.Name, c.ClientIP(), c.Request.UserAgent())

	return &rule, nil
}

// UpdateFraudRule updates a fraud detection rule
func (s *FraudDetectionService) UpdateFraudRule(c *gin.Context, id uint, updates map[string]interface{}) (*models.FraudDetection, error) {
	adminID := c.GetUint("admin_id")

	var rule models.FraudDetection
	if err := s.db.First(&rule, id).Error; err != nil {
		return nil, err
	}

	err := s.db.Model(&rule).Updates(updates).Error
	if err != nil {
		return nil, err
	}

	logAdminActivity(s.db, adminID, "update_fraud_rule", "fraud_detection", 
		fmt.Sprintf("%d", id), "Updated fraud rule: "+rule.Name, c.ClientIP(), c.Request.UserAgent())

	return &rule, nil
}

// DeleteFraudRule deletes a fraud detection rule
func (s *FraudDetectionService) DeleteFraudRule(c *gin.Context, id uint) error {
	adminID := c.GetUint("admin_id")

	var rule models.FraudDetection
	if err := s.db.First(&rule, id).Error; err != nil {
		return err
	}

	err := s.db.Delete(&rule).Error
	if err != nil {
		return err
	}

	logAdminActivity(s.db, adminID, "delete_fraud_rule", "fraud_detection", 
		fmt.Sprintf("%d", id), "Deleted fraud rule: "+rule.Name, c.ClientIP(), c.Request.UserAgent())

	return nil
}

// GetFraudAlerts gets fraud alerts with filtering
func (s *FraudDetectionService) GetFraudAlerts(status string, page, pageSize int) ([]models.FraudAlert, int64, error) {
	var alerts []models.FraudAlert
	var total int64

	query := s.db.Model(&models.FraudAlert{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&alerts).Error

	return alerts, total, err
}

// UpdateFraudAlert updates a fraud alert
func (s *FraudDetectionService) UpdateFraudAlert(c *gin.Context, id uint, status, resolution string) error {
	adminID := c.GetUint("admin_id")

	var alert models.FraudAlert
	if err := s.db.First(&alert, id).Error; err != nil {
		return err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":     status,
		"resolved_at": now,
	}

	if status == "reviewed" {
		adminIDUint := uint(adminID)
		updates["reviewed_by"] = &adminIDUint
		updates["reviewed_at"] = now
	}

	if resolution != "" {
		updates["resolution"] = resolution
	}

	err := s.db.Model(&alert).Updates(updates).Error
	if err != nil {
		return err
	}

	logAdminActivity(s.db, adminID, "update_fraud_alert", "fraud_alert", 
		fmt.Sprintf("%d", id), "Updated fraud alert: "+status, c.ClientIP(), c.Request.UserAgent())

	return nil
}

// GetFraudStats gets fraud detection statistics
func (s *FraudDetectionService) GetFraudStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalAlerts int64
	var pendingAlerts int64
	var reviewedAlerts int64
	var resolvedAlerts int64

	s.db.Model(&models.FraudAlert{}).Count(&totalAlerts)
	s.db.Model(&models.FraudAlert{}).Where("status = ?", "pending").Count(&pendingAlerts)
	s.db.Model(&models.FraudAlert{}).Where("status = ?", "reviewed").Count(&reviewedAlerts)
	s.db.Model(&models.FraudAlert{}).Where("status = ?", "resolved").Count(&resolvedAlerts)

	stats["total_alerts"] = totalAlerts
	stats["pending"] = pendingAlerts
	stats["reviewed"] = reviewedAlerts
	stats["resolved"] = resolvedAlerts

	var rules []models.FraudDetection
	s.db.Where("is_active = ?", true).Find(&rules)
	stats["active_rules"] = len(rules)

	return stats, nil
}
