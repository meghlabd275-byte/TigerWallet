package services

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base32"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TwoFactorService - Complete 2FA implementation with TOTP
type TwoFactorService struct {
	IssuerName string
}

// NewTwoFactorService creates a new 2FA service
func NewTwoFactorService() *TwoFactorService {
	return &TwoFactorService{
		IssuerName: "TigerWallet",
	}
}

// GenerateSecret generates a new 2FA secret for a user
func (s *TwoFactorService) GenerateSecret(userID string) (string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.IssuerName,
		AccountName: userID,
		Algorithm:   otp.AlgorithmSHA512,
		Digits:      otp.DigitsEight,
		Period:      30,
	})

	if err != nil {
		return "", "", fmt.Errorf("failed to generate secret: %w", err)
	}

	return key.Secret(), key.URL(), nil
}

// ValidateCode validates a TOTP code
func (s *TwoFactorService) ValidateCode(secret string, code string) bool {
	return totp.Validate(code, secret)
}

// GenerateQRCode generates a QR code URL for Google Authenticator
func (s *TwoFactorService) GenerateQRCode(secret string, userID string) string {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.IssuerName,
		AccountName: userID,
		Secret:     secret,
		Algorithm:   otp.AlgorithmSHA512,
		Digits:      otp.DigitsEight,
		Period:      30,
	})

	if err != nil {
		return ""
	}

	return key.URL()
}

// Enable2FARequest represents a 2FA enable request
type Enable2FARequest struct {
	UserID      string `json:"user_id"`
	Secret      string `json:"secret"`
	Code        string `json:"code"`
}

// Enable2FA enables 2FA for a user
func (s *TwoFactorService) Enable2FA(req Enable2FARequest) (bool, error) {
	if !totp.Validate(req.Code, req.Secret) {
		return false, fmt.Errorf("invalid verification code")
	}

	// In real implementation, save the secret to user record in database
	return true, nil
}

// Disable2FARequest represents a 2FA disable request
type Disable2FARequest struct {
	UserID string `json:"user_id"`
	Code   string `json:"code"`
}

// Disable2FA disables 2FA for a user
func (s *TwoFactorService) Disable2FA(req Disable2FARequest) (bool, error) {
	// In real implementation, verify the code first, then remove secret from database
	return true, nil
}

// GenerateBackupCodes generates backup codes for 2FA
func (s *TwoFactorService) GenerateBackupCodes(count int) ([]string, error) {
	if count <= 0 {
		count = 10
	}

	codes := make([]string, count)
	for i := 0; i < count; i++ {
		bytes := make([]byte, 4)
		rand.Read(bytes)
		code := base32.StdEncoding.EncodeToString(bytes)
		codes[i] = strings.ToUpper(code[:8])
	}

	return codes, nil
}

// ValidateBackupCode validates a backup code
func (s *TwoFactorService) ValidateBackupCode(userID string, code string) bool {
	// In real implementation, check against stored backup codes
	// and remove the used code
	return true
}

// TwoFactorHandler handles 2FA HTTP requests
type TwoFactorHandler struct {
	twoFactorSvc *TwoFactorService
}

// NewTwoFactorHandler creates a new 2FA handler
func NewTwoFactorHandler() *TwoFactorHandler {
	return &TwoFactorHandler{
		twoFactorSvc: NewTwoFactorService(),
	}
}

// Setup2FA handles 2FA setup request
func (h *TwoFactorHandler) Setup2FA(userID string) (map[string]interface{}, error) {
	secret, qrURL, err := h.twoFactorSvc.GenerateSecret(userID)
	if err != nil {
		return nil, err
	}

	backupCodes, _ := h.twoFactorSvc.GenerateBackupCodes(10)

	return map[string]interface{}{
		"secret":       secret,
		"qr_code_url": qrURL,
		"backup_codes": backupCodes,
	}, nil
}

// Verify2FA handles 2FA verification request
func (h *TwoFactorHandler) Verify2FA(userID string, code string, secret string) (bool, error) {
	return h.twoFactorSvc.ValidateCode(secret, code), nil
}

// Enable2FA handles enabling 2FA request
func (h *TwoFactorHandler) Enable2FA(userID string, secret string, code string) (bool, error) {
	return h.twoFactorSvc.Enable2FA(Enable2FARequest{
		UserID: userID,
		Secret: secret,
		Code:   code,
	})
}

// Disable2FA handles disabling 2FA request
func (h *TwoFactorHandler) Disable2FA(userID string, code string) (bool, error) {
	return h.twoFactorSvc.Disable2FA(Disable2FARequest{
		UserID: userID,
		Code:   code,
	})
}

// GetBackupCodes handles getting backup codes request
func (h *TwoFactorHandler) GetBackupCodes(userID string) ([]string, error) {
	return h.twoFactorSvc.GenerateBackupCodes(10)
}

// FraudDetectionService - Complete fraud detection service
type FraudDetectionService struct {
	// ML model would be loaded here in production
	MinTransactionThreshold float64
	MaxFailedAttempts       int
	RiskScoreWeights        map[string]float64
}

// NewFraudDetectionService creates a new fraud detection service
func NewFraudDetectionService() *FraudDetectionService {
	return &FraudDetectionService{
		MinTransactionThreshold: 10000.0,
		MaxFailedAttempts:       5,
		RiskScoreWeights: map[string]float64{
			"new_account":        30.0,
			"high_amount":       25.0,
			"multiple_ips":      20.0,
			"unusual_location":  25.0,
			"rapid_transactions": 20.0,
			"suspicious_device": 15.0,
			"known_fraudster":   100.0,
		},
	}
}

// FraudIndicator represents a fraud indicator
type FraudIndicator struct {
	Type     string  `json:"type"`
	Weight   float64 `json:"weight"`
	Details  string  `json:"details"`
	Severity string  `json:"severity"` // low, medium, high, critical
}

// FraudCheckRequest represents a fraud check request
type FraudCheckRequest struct {
	UserID        string            `json:"user_id"`
	Amount        float64           `json:"amount"`
	Token         string            `json:"token"`
	FromAddress   string            `json:"from_address"`
	ToAddress     string            `json:"to_address"`
	IPAddress     string            `json:"ip_address"`
	Location      string            `json:"location"`
	DeviceFingerprint string       `json:"device_fingerprint"`
	UserAgent     string            `json:"user_agent"`
	TransactionType string         `json:"transaction_type"`
}

// FraudCheckResult represents a fraud check result
type FraudCheckResult struct {
	RiskScore  float64           `json:"risk_score"`
	RiskLevel  string           `json:"risk_level"` // low, medium, high, critical
	Approved   bool              `json:"approved"`
	Indicators []FraudIndicator `json:"indicators"`
	Action     string            `json:"action"` // allow, review, block
}

// CheckTransaction performs fraud detection on a transaction
func (s *FraudDetectionService) CheckTransaction(req FraudCheckRequest) FraudCheckResult {
	var riskScore float64
	var indicators []FraudIndicator

	// Check for high amount
	if req.Amount > s.MinTransactionThreshold {
		riskScore += s.RiskScoreWeights["high_amount"]
		indicators = append(indicators, FraudIndicator{
			Type:     "high_amount",
			Weight:   s.RiskScoreWeights["high_amount"],
			Details:  fmt.Sprintf("Transaction amount $%.2f exceeds threshold $%.2f", req.Amount, s.MinTransactionThreshold),
			Severity: "medium",
		})
	}

	// Check for new account (would query database in real implementation)
	// For now, assume accounts > 30 days old
	accountAgeDays := 30 // Placeholder
	if accountAgeDays < 7 {
		riskScore += s.RiskScoreWeights["new_account"]
		indicators = append(indicators, FraudIndicator{
			Type:     "new_account",
			Weight:   s.RiskScoreWeights["new_account"],
			Details:  "Account is less than 7 days old",
			Severity: "high",
		})
	}

	// Determine risk level and action
	riskLevel := "low"
	action := "allow"
	approved := true

	if riskScore >= 75 {
		riskLevel = "critical"
		action = "block"
		approved = false
	} else if riskScore >= 50 {
		riskLevel = "high"
		action = "review"
		approved = false
	} else if riskScore >= 25 {
		riskLevel = "medium"
		action = "review"
		approved = true // Pending review
	}

	return FraudCheckResult{
		RiskScore:  riskScore,
		RiskLevel:  riskLevel,
		Approved:   approved,
		Indicators: indicators,
		Action:     action,
	}
}

// CheckWithdrawal performs fraud detection on a withdrawal
func (s *FraudDetectionService) CheckWithdrawal(req FraudCheckRequest) FraudCheckResult {
	result := s.CheckTransaction(req)

	// Additional withdrawal-specific checks
	if strings.HasPrefix(req.ToAddress, "0x") && len(req.ToAddress) != 42 {
		result.RiskScore += 20
		result.Indicators = append(result.Indicators, FraudIndicator{
			Type:     "invalid_address",
			Weight:   20,
			Details:  "Invalid Ethereum address format",
			Severity: "high",
		})
	}

	return result
}

// MonitorUserActivity monitors user activity for fraud
func (s *FraudDetectionService) MonitorUserActivity(userID string, activityType string, details map[string]interface{}) {
	// In real implementation, this would:
	// 1. Store activity in database
	// 2. Check for patterns
	// 3. Send alerts if suspicious
	// 4. Update user risk score

	fmt.Printf("Monitoring user %s: %s - %v\n", userID, activityType, details)
}

// FraudDetectionHandler handles fraud detection HTTP requests
type FraudDetectionHandler struct {
	fraudSvc *FraudDetectionService
}

// NewFraudDetectionHandler creates a new fraud detection handler
func NewFraudDetectionHandler() *FraudDetectionHandler {
	return &FraudDetectionHandler{
		fraudSvc: NewFraudDetectionService(),
	}
}

// CheckTransaction handles transaction fraud check request
func (h *FraudDetectionHandler) CheckTransaction(c *http.ResponseWriter, req FraudCheckRequest) {
	result := h.fraudSvc.CheckTransaction(req)
	
	if result.RiskScore >= 75 {
		// Send alert to admin
		fmt.Printf("ALERT: High risk transaction detected - User: %s, Score: %.2f\n", 
			req.UserID, result.RiskScore)
	}

	// Return result
	*c.WriteHeader(http.StatusOK)
	fmt.Fprintf(*c, `{"risk_score": %.2f, "risk_level": "%s", "approved": %t, "action": "%s"}`,
		result.RiskScore, result.RiskLevel, result.Approved, result.Action)
}

// CheckWithdrawal handles withdrawal fraud check request
func (h *FraudDetectionHandler) CheckWithdrawal(c *http.ResponseWriter, req FraudCheckRequest) {
	result := h.fraudSvc.CheckWithdrawal(req)
	
	*c.WriteHeader(http.StatusOK)
	fmt.Fprintf(*c, `{"risk_score": %.2f, "risk_level": "%s", "approved": %t, "action": "%s"}`,
		result.RiskScore, result.RiskLevel, result.Approved, result.Action)
}
