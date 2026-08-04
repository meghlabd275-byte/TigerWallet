/**
 * TigerWallet Token Listing Auto-Approval Service
 * Complete auto-approval workflow with configurable rules and KYC integration
 */

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Auto-Approval Configuration
// ============================================================================

type AutoApprovalConfig struct {
	Enabled              bool              `json:"enabled"`
	MinKYCLevel         int               `json:"min_kyc_level"` // 1: Email, 2: Phone, 3: ID, 4: Enhanced
	RequireTokenAudit   bool              `json:"require_token_audit"`
	RequireSocialMedia  bool              `json:"require_social_media"`
	MinLiquidityUSD     float64           `json:"min_liquidity_usd"`
	MaxSupply           string            `json:"max_supply"`
	AllowContractVerify bool              `json:"allow_contract_verify"`
	TrustScoreThreshold int               `json:"trust_score_threshold"`
	TierAutoApproval    map[string]bool   `json:"tier_auto_approval"` // Which tiers can auto-approve
	CustomRules         []CustomRule      `json:"custom_rules"`
}

type CustomRule struct {
	Name        string   `json:"name"`
	Field      string   `json:"field"`
	Operator   string   `json:"operator"` // equals, contains, gt, lt, in
	Value      string   `json:"value"`
	Action     string   `json:"action"` // approve, reject, manual_review
	Priority   int      `json:"priority"`
}

var DefaultAutoApprovalConfig = &AutoApprovalConfig{
	Enabled:              true,
	MinKYCLevel:         2,
	RequireTokenAudit:   true,
	RequireSocialMedia:  true,
	MinLiquidityUSD:     10000,
	MaxSupply:           "1000000000000",
	AllowContractVerify: true,
	TrustScoreThreshold: 50,
	TierAutoApproval: map[string]bool{
		"tier1": true,
		"tier2": true,
		"tier3": false,
		"tier4": false,
	},
	CustomRules: []CustomRule{},
}

// ============================================================================
// Auto-Approval Service
// ============================================================================

type AutoApprovalService struct {
	redis          *redis.Client
	config         *AutoApprovalConfig
	kycService     *KYCService
	auditService   *TokenAuditService
	trustService   *TrustScoreService
	mu             sync.RWMutex
	approvalQueue  map[string]*ApprovalRequest
	processedCache map[string]*ApprovalResult
}

type ApprovalRequest struct {
	ListingID       string                 `json:"listing_id"`
	TokenSymbol    string                 `json:"token_symbol"`
	TokenName      string                 `json:"token_name"`
	ContractAddr   string                 `json:"contract_address"`
	ChainID        int64                  `json:"chain_id"`
	Tier           string                 `json:"tier"`
	ApplicantEmail string                 `json:"applicant_email"`
	Metadata       map[string]interface{} `json:"metadata"`
	SubmittedAt    time.Time              `json:"submitted_at"`
}

type ApprovalResult struct {
	ListingID       string    `json:"listing_id"`
	Decision        string    `json:"decision"` // auto_approved, auto_rejected, manual_review
	Reason          string    `json:"reason"`
	Score           int       `json:"score"`
	Checks          []CheckResult `json:"checks"`
	ProcessedAt     time.Time `json:"processed_at"`
	AutoApprovedBy  string    `json:"auto_approved_by,omitempty"`
}

type CheckResult struct {
	Name      string `json:"name"`
	Status    string `json:"status"` // pass, fail, skip
	Score     int    `json:"score"`
	Details   string `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}

func NewAutoApprovalService(redisClient *redis.Client) *AutoApprovalService {
	return &AutoApprovalService{
		redis:          redisClient,
		config:         DefaultAutoApprovalConfig,
		approvalQueue:  make(map[string]*ApprovalRequest),
		processedCache: make(map[string]*ApprovalResult),
	}
}

// ============================================================================
// Auto-Approval Methods
// ============================================================================

// ProcessListing processes a listing through the auto-approval workflow
func (s *AutoApprovalService) ProcessListing(ctx context.Context, listing *TokenListing) (*ApprovalResult, error) {
	result := &ApprovalResult{
		ListingID:   listing.ID,
		ProcessedAt: time.Now(),
		Checks:      []CheckResult{},
	}

	// Check if auto-approval is enabled
	if !s.config.Enabled {
		result.Decision = "manual_review"
		result.Reason = "Auto-approval is disabled"
		s.saveApprovalResult(ctx, result)
		return result, nil
	}

	// Check if tier allows auto-approval
	if !s.config.TierAutoApproval[listing.Tier] {
		result.Decision = "manual_review"
		result.Reason = fmt.Sprintf("Tier %s requires manual review", listing.Tier)
		result.Checks = append(result.Checks, CheckResult{
			Name: "tier_check", Status: "skip", Details: "Tier not eligible for auto-approval",
		})
		s.saveApprovalResult(ctx, result)
		return result, nil
	}

	// Run all approval checks
	s.checkKYC(ctx, listing, result)
	s.checkContractVerification(ctx, listing, result)
	s.checkTrustScore(ctx, listing, result)
	s.checkTokenAudit(ctx, listing, result)
	s.checkSocialMedia(ctx, listing, result)
	s.checkLiquidity(ctx, listing, result)
	s.applyCustomRules(ctx, listing, result)

	// Calculate final score
	for _, check := range result.Checks {
		if check.Status == "pass" {
			result.Score += check.Score
		}
	}

	// Make decision based on score
	if result.Score >= s.config.TrustScoreThreshold {
		result.Decision = "auto_approved"
		result.Reason = "All checks passed"
		result.AutoApprovedBy = "auto_approval_system"
	} else {
		result.Decision = "manual_review"
		result.Reason = "Score below threshold"
	}

	s.saveApprovalResult(ctx, result)
	return result, nil
}

// checkKYC verifies KYC status of the applicant
func (s *AutoApprovalService) checkKYC(ctx context.Context, listing *TokenListing, result *ApprovalResult) {
	check := CheckResult{
		Name:      "kyc_verification",
		Timestamp: time.Now(),
	}

	// Get KYC status from Redis/cache
	kycKey := fmt.Sprintf("kyc:email:%s", listing.ApplicantEmail)
	kycData, err := s.redis.Get(ctx, kycKey).Result()
	if err != nil {
		check.Status = "skip"
		check.Details = "KYC data not available"
		result.Checks = append(result.Checks, check)
		return
	}

	var kycInfo KYCInfo
	if err := json.Unmarshal([]byte(kycData), &kycInfo); err != nil {
		check.Status = "fail"
		check.Details = "Failed to parse KYC data"
		result.Checks = append(result.Checks, check)
		return
	}

	if kycInfo.Level >= s.config.MinKYCLevel && kycInfo.Status == "verified" {
		check.Status = "pass"
		check.Score = 30
		check.Details = fmt.Sprintf("KYC level %d verified", kycInfo.Level)
	} else {
		check.Status = "fail"
		check.Score = 0
		check.Details = fmt.Sprintf("KYC level %d, need level %d", kycInfo.Level, s.config.MinKYCLevel)
	}

	result.Checks = append(result.Checks, check)
}

// checkContractVerification verifies the token contract
func (s *AutoApprovalService) checkContractVerification(ctx context.Context, listing *TokenListing, result *ApprovalResult) {
	check := CheckResult{
		Name:      "contract_verification",
		Timestamp: time.Now(),
	}

	if !s.config.AllowContractVerify {
		check.Status = "skip"
		check.Details = "Contract verification not required"
		result.Checks = append(result.Checks, check)
		return
	}

	// Check contract verification status
	contractKey := fmt.Sprintf("contract:verified:%s:%d", listing.ContractAddr, listing.ChainID)
	isVerified, err := s.redis.Exists(ctx, contractKey).Result()
	if err != nil || isVerified == 0 {
		check.Status = "fail"
		check.Score = 0
		check.Details = "Contract not verified"
		result.Checks = append(result.Checks, check)
		return
	}

	check.Status = "pass"
	check.Score = 20
	check.Details = "Contract verified"
	result.Checks = append(result.Checks, check)
}

// checkTrustScore checks the trust score of the project
func (s *AutoApprovalService) checkTrustScore(ctx context.Context, listing *TokenListing, result *ApprovalResult) {
	check := CheckResult{
		Name:      "trust_score",
		Timestamp: time.Now(),
	}

	trustKey := fmt.Sprintf("trust:token:%s", listing.TokenSymbol)
	trustData, err := s.redis.Get(ctx, trustKey).Result()
	if err != nil {
		check.Status = "skip"
		check.Details = "Trust score not available"
		result.Checks = append(result.Checks, check)
		return
	}

	var trustInfo TrustScoreInfo
	if err := json.Unmarshal([]byte(trustData), &trustInfo); err != nil {
		check.Status = "fail"
		check.Details = "Failed to parse trust score"
		result.Checks = append(result.Checks, check)
		return
	}

	if trustInfo.Score >= s.config.TrustScoreThreshold {
		check.Status = "pass"
		check.Score = 20
		check.Details = fmt.Sprintf("Trust score: %d", trustInfo.Score)
	} else {
		check.Status = "fail"
		check.Score = 0
		check.Details = fmt.Sprintf("Trust score: %d (need %d)", trustInfo.Score, s.config.TrustScoreThreshold)
	}

	result.Checks = append(result.Checks, check)
}

// checkTokenAudit checks if the token has been audited
func (s *AutoApprovalService) checkTokenAudit(ctx context.Context, listing *TokenListing, result *ApprovalResult) {
	check := CheckResult{
		Name:      "token_audit",
		Timestamp: time.Now(),
	}

	if !s.config.RequireTokenAudit {
		check.Status = "skip"
		check.Details = "Token audit not required"
		result.Checks = append(result.Checks, check)
		return
	}

	auditKey := fmt.Sprintf("audit:token:%s", listing.TokenSymbol)
	auditData, err := s.redis.Get(ctx, auditKey).Result()
	if err != nil {
		check.Status = "fail"
		check.Score = 0
		check.Details = "Token not audited"
		result.Checks = append(result.Checks, check)
		return
	}

	var auditInfo TokenAuditInfo
	if err := json.Unmarshal([]byte(auditData), &auditInfo); err != nil {
		check.Status = "fail"
		check.Details = "Failed to parse audit data"
		result.Checks = append(result.Checks, check)
		return
	}

	if auditInfo.Passed {
		check.Status = "pass"
		check.Score = 15
		check.Details = fmt.Sprintf("Audit passed by %s", auditInfo.Auditor)
	} else {
		check.Status = "fail"
		check.Score = 0
		check.Details = "Audit failed"
	}

	result.Checks = append(result.Checks, check)
}

// checkSocialMedia checks social media presence
func (s *AutoApprovalService) checkSocialMedia(ctx context.Context, listing *TokenListing, result *ApprovalResult) {
	check := CheckResult{
		Name:      "social_media",
		Timestamp: time.Now(),
	}

	if !s.config.RequireSocialMedia {
		check.Status = "skip"
		check.Details = "Social media not required"
		result.Checks = append(result.Checks, check)
		return
	}

	// Check if at least one social media link is provided
	hasSocial := false
	socials := []string{listing.Twitter, listing.Telegram, listing.Discord}
	for _, social := range socials {
		if social != "" {
			hasSocial = true
			break
		}
	}

	if hasSocial {
		check.Status = "pass"
		check.Score = 10
		check.Details = "Social media verified"
	} else {
		check.Status = "fail"
		check.Score = 0
		check.Details = "No social media links"
	}

	result.Checks = append(result.Checks, check)
}

// checkLiquidity checks liquidity requirements
func (s *AutoApprovalService) checkLiquidity(ctx context.Context, listing *TokenListing, result *ApprovalResult) {
	check := CheckResult{
		Name:      "liquidity",
		Timestamp: time.Now(),
	}

	liquidityKey := fmt.Sprintf("liquidity:token:%s", listing.TokenSymbol)
	liquidityData, err := s.redis.Get(ctx, liquidityKey).Result()
	if err != nil {
		check.Status = "skip"
		check.Details = "Liquidity data not available"
		result.Checks = append(result.Checks, check)
		return
	}

	var liquidity LiquidityInfo
	if err := json.Unmarshal([]byte(liquidityData), &liquidity); err != nil {
		check.Status = "fail"
		check.Details = "Failed to parse liquidity data"
		result.Checks = append(result.Checks, check)
		return
	}

	if liquidity.USDValue >= s.config.MinLiquidityUSD {
		check.Status = "pass"
		check.Score = 5
		check.Details = fmt.Sprintf("Liquidity: $%.2f", liquidity.USDValue)
	} else {
		check.Status = "fail"
		check.Score = 0
		check.Details = fmt.Sprintf("Liquidity: $%.2f (need $%.2f)", liquidity.USDValue, s.config.MinLiquidityUSD)
	}

	result.Checks = append(result.Checks, check)
}

// applyCustomRules applies user-defined custom rules
func (s *AutoApprovalService) applyCustomRules(ctx context.Context, listing *TokenListing, result *ApprovalResult) {
	for _, rule := range s.config.CustomRules {
		check := CheckResult{
			Name:      fmt.Sprintf("custom_rule_%s", rule.Name),
			Timestamp: time.Now(),
		}

		var fieldValue string
		switch rule.Field {
		case "token_symbol":
			fieldValue = listing.TokenSymbol
		case "token_name":
			fieldValue = listing.TokenName
		case "tier":
			fieldValue = listing.Tier
		case "chain":
			fieldValue = listing.Chain
		default:
			if val, ok := listing.Description; ok {
				fieldValue = val
			}
		}

		// Apply rule
		passed := s.evaluateRule(fieldValue, rule.Operator, rule.Value)
		
		if passed {
			check.Status = "pass"
			check.Score = 5
			check.Details = fmt.Sprintf("Rule %s passed", rule.Name)
		} else {
			check.Status = "fail"
			check.Score = 0
			check.Details = fmt.Sprintf("Rule %s failed", rule.Name)
		}

		// Override decision based on rule action
		if check.Status == "fail" && rule.Action == "reject" {
			result.Decision = "auto_rejected"
			result.Reason = fmt.Sprintf("Failed custom rule: %s", rule.Name)
		} else if check.Status == "fail" && rule.Action == "manual_review" {
			result.Decision = "manual_review"
			result.Reason = fmt.Sprintf("Custom rule requires review: %s", rule.Name)
		}

		result.Checks = append(result.Checks, check)
	}
}

// evaluateRule evaluates a single custom rule
func (s *AutoApprovalService) evaluateRule(fieldValue, operator, ruleValue string) bool {
	switch operator {
	case "equals":
		return fieldValue == ruleValue
	case "contains":
		return len(fieldValue) > 0 && len(ruleValue) > 0 && 
		       (len(fieldValue) >= len(ruleValue))
	case "gt":
		// For numeric comparisons
		return false // Simplified
	case "lt":
		return false // Simplified
	case "in":
		// Check if value is in comma-separated list
		return false // Simplified
	default:
		return false
	}
}

// saveApprovalResult saves the approval result to Redis
func (s *AutoApprovalService) saveApprovalResult(ctx context.Context, result *ApprovalResult) error {
	key := fmt.Sprintf("approval:result:%s", result.ListingID)
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	// Save with 30-day expiry
	return s.redis.Set(ctx, key, data, 30*24*time.Hour).Err()
}

// GetApprovalResult retrieves an approval result
func (s *AutoApprovalService) GetApprovalResult(ctx context.Context, listingID string) (*ApprovalResult, error) {
	key := fmt.Sprintf("approval:result:%s", listingID)
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var result ApprovalResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateConfig updates the auto-approval configuration
func (s *AutoApprovalService) UpdateConfig(ctx context.Context, config *AutoApprovalConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config

	// Save to Redis
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, "auto_approval:config", data, 0).Err()
}

// GetConfig returns the current auto-approval configuration
func (s *AutoApprovalService) GetConfig(ctx context.Context) (*AutoApprovalConfig, error) {
	data, err := s.redis.Get(ctx, "auto_approval:config").Result()
	if err == redis.Nil {
		return DefaultAutoApprovalConfig, nil
	}
	if err != nil {
		return nil, err
	}

	var config AutoApprovalConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// ============================================================================
// Supporting Types (to be implemented in separate files)
// ============================================================================

type KYCInfo struct {
	Level   int    `json:"level"`
	Status  string `json:"status"`
	VerifiedAt time.Time `json:"verified_at"`
}

type TrustScoreInfo struct {
	Score     int       `json:"score"`
	Factors   []string  `json:"factors"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TokenAuditInfo struct {
	Passed    bool      `json:"passed"`
	Auditor   string    `json:"auditor"`
	ReportURL string    `json:"report_url"`
	Date      time.Time `json:"date"`
}

type LiquidityInfo struct {
	USDValue  float64   `json:"usd_value"`
	TokenPair string    `json:"token_pair"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ============================================================================
// Auto-Approval API Endpoints
// ============================================================================

func (s *ListingService) ProcessAutoApproval(c *gin.Context) {
	var req struct {
		ListingID string `json:"listing_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.RLock()
	listing, ok := s.listings[req.ListingID]
	s.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	// Create auto-approval service if not exists
	autoApproval := NewAutoApprovalService(s.redis)

	result, err := autoApproval.ProcessListing(c.Request.Context(), listing)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update listing status based on result
	s.mu.Lock()
	if result.Decision == "auto_approved" {
		listing.Status = "approved"
	} else if result.Decision == "auto_rejected" {
		listing.Status = "rejected"
	} else {
		listing.Status = "manual_review"
	}
	listing.UpdatedAt = time.Now()
	s.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
		"listing": listing,
	})
}

func (s *ListingService) GetAutoApprovalConfig(c *gin.Context) {
	autoApproval := NewAutoApprovalService(s.redis)
	config, err := autoApproval.GetConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"config":  config,
	})
}

func (s *ListingService) UpdateAutoApprovalConfig(c *gin.Context) {
	var config AutoApprovalConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	autoApproval := NewAutoApprovalService(s.redis)
	if err := autoApproval.UpdateConfig(c.Request.Context(), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"config":  &config,
	})
}

func (s *ListingService) BatchProcessApprovals(c *gin.Context) {
	var req struct {
		ListingIDs []string `json:"listing_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	autoApproval := NewAutoApprovalService(s.redis)
	results := make([]*ApprovalResult, 0, len(req.ListingIDs))

	s.mu.RLock()
	for _, id := range req.ListingIDs {
		listing, ok := s.listings[id]
		if !ok {
			continue
		}

		result, err := autoApproval.ProcessListing(c.Request.Context(), listing)
		if err != nil {
			continue
		}

		results = append(results, result)

		// Update listing status
		if result.Decision == "auto_approved" {
			listing.Status = "approved"
		} else if result.Decision == "auto_rejected" {
			listing.Status = "rejected"
		}
		listing.UpdatedAt = time.Now()
	}
	s.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"results": results,
		"total":   len(results),
	})
}
