package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// TwoFactorService handles 2FA operations
type TwoFactorService struct {
	db    *database.PostgresDB
	redis *redis.Client
}

// NewTwoFactorService creates a new 2FA service
func NewTwoFactorService(db *database.PostgresDB, redis *redis.Client) *TwoFactorService {
	return &TwoFactorService{
		db:    db,
		redis: redis,
	}
}

// Setup2FARequest represents a 2FA setup request
type Setup2FARequest struct {
	UserID   uint   `json:"user_id" binding:"required"`
	UserType string `json:"user_type" binding:"required"` // "admin", "user", "super_admin"
}

// Setup2FAResponse represents the 2FA setup response
type Setup2FAResponse struct {
	Secret      string   `json:"secret"`
	QRCode      string   `json:"qr_code"`
	BackupCodes []string `json:"backup_codes"`
	RecoveryURL string   `json:"recovery_url"`
	ExpiresAt   int64    `json:"expires_at"`
}

// Verify2FARequest represents a 2FA verification request
type Verify2FARequest struct {
	UserID    uint   `json:"user_id" binding:"required"`
	UserType  string `json:"user_type" binding:"required"`
	Code      string `json:"code" binding:"required"`
	IPAddress string `json:"ip_address"`
}

// Enable2FARequest represents a request to enable 2FA
type Enable2FARequest struct {
	UserID     uint   `json:"user_id" binding:"required"`
	UserType   string `json:"user_type" binding:"required"`
	Code       string `json:"code" binding:"required"`
	BackupCode string `json:"backup_code"`
}

// Disable2FARequest represents a request to disable 2FA
type Disable2FARequest struct {
	UserID     uint   `json:"user_id" binding:"required"`
	UserType   string `json:"user_type" binding:"required"`
	Code       string `json:"code"`
	BackupCode string `json:"backup_code"`
	Password   string `json:"password" binding:"required"`
}

// TwoFactorStatus represents the 2FA status for a user
type TwoFactorStatus struct {
	Enabled           bool     `json:"enabled"`
	Methods           []string `json:"methods"`
	BackupCodesLeft   int      `json:"backup_codes_left"`
	LastVerifiedAt    *int64   `json:"last_verified_at"`
	TrustedDevices    int      `json:"trusted_devices"`
	RecoveryCodesLeft int      `json:"recovery_codes_left"`
}

// Setup2FA generates a new 2FA secret for a user
func (s *TwoFactorService) Setup2FA(c *gin.Context, req Setup2FARequest) (*Setup2FAResponse, error) {
	// Generate TOTP secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "TigerWallet",
		AccountName: fmt.Sprintf("%s_%d", req.UserType, req.UserID),
		Algorithm:   otp.AlgorithmSHA256,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate 2FA secret: %w", err)
	}

	// Generate backup codes
	backupCodes := s.generateBackupCodes(10)

	// Store temporary secret in Redis (expires in 10 minutes)
	secretData := map[string]interface{}{
		"secret":       key.Secret(),
		"backup_codes": backupCodes,
		"user_id":      req.UserID,
		"user_type":    req.UserType,
	}
	secretJSON, _ := json.Marshal(secretData)
	redisKey := fmt.Sprintf("2fa_setup:%s:%d", req.UserType, req.UserID)
	s.redis.Set(c.Request.Context(), redisKey, secretJSON, 10*time.Minute)

	// Generate QR code URL (will be converted to QR in frontend)
	qrCode := key.URL()

	expiresAt := time.Now().Add(10 * time.Minute).Unix()

	return &Setup2FAResponse{
		Secret:      key.Secret(),
		QRCode:      qrCode,
		BackupCodes: backupCodes,
		RecoveryURL: fmt.Sprintf("/2fa/recovery/%s/%d", req.UserType, req.UserID),
		ExpiresAt:   expiresAt,
	}, nil
}

// Verify2FA verifies a 2FA code
func (s *TwoFactorService) Verify2FA(c *gin.Context, req Verify2FARequest) (bool, string, error) {
	// Get user 2FA settings
	var twoFactor models.TwoFactorAuth
	err := s.db.Where("user_id = ? AND user_type = ?", req.UserID, req.UserType).First(&twoFactor).Error
	if err != nil {
		return false, "", fmt.Errorf("2FA not enabled for this user")
	}

	// Check if using backup code
	if len(req.Code) == 8 {
		// Verify backup code
		codeHash := s.hashCode(req.Code)
		if twoFactor.BackupCodesHashed == "" {
			return false, "Invalid code", nil
		}

		// Check if backup code is valid and not used
		backupCodes := s.getStoredBackupCodes(c, req.UserID, req.UserType)
		var usedCodes map[string]bool
		json.Unmarshal([]byte(twoFactor.UsedBackupCodes), &usedCodes)
		for _, code := range backupCodes {
			if s.hashCode(code) == codeHash && !usedCodes[strings.ToUpper(req.Code)] {
				// Mark backup code as used
				s.markBackupCodeUsed(c, req.UserID, req.UserType, req.Code)
				return true, "", nil
			}
		}
		return false, "Invalid backup code", nil
	}

	// Verify TOTP code
	valid := totp.Validate(req.Code, twoFactor.Secret)
	if !valid {
		// Log failed attempt
		s.logFailedAttempt(c, req.UserID, req.UserType, req.IPAddress)
		return false, "Invalid verification code", nil
	}

	// Update last verified time
	now := time.Now().Unix()
	s.db.Model(&twoFactor).Update("last_verified_at", now)

	// Clear failed attempts
	s.clearFailedAttempts(c, req.UserID, req.UserType)

	return true, "", nil
}

// Enable2FA enables 2FA for a user after verification
func (s *TwoFactorService) Enable2FA(c *gin.Context, req Enable2FARequest) error {
	// Get temporary secret from Redis
	redisKey := fmt.Sprintf("2fa_setup:%s:%d", req.UserType, req.UserID)
	secretJSON, err := s.redis.Get(c.Request.Context(), redisKey).Result()
	if err != nil {
		return fmt.Errorf("2FA setup expired or not found. Please restart setup.")
	}

	var secretData map[string]interface{}
	json.Unmarshal([]byte(secretJSON), &secretData)

	secret := secretData["secret"].(string)
	backupCodes := secretData["backup_codes"].([]interface{})

	// Verify the code first
	valid := totp.Validate(req.Code, secret)
	if !valid {
		return fmt.Errorf("invalid verification code")
	}

	// Hash backup codes
	backupCodesHashed := make(map[string]bool)
	for _, code := range backupCodes {
		backupCodesHashed[strings.ToUpper(code.(string))] = false
	}
	backupCodesJSON, _ := json.Marshal(backupCodesHashed)

	// Create or update 2FA record
	now := time.Now()
	lastVerified := time.Now().Unix()
	twoFactor := models.TwoFactorAuth{
		UserID:            req.UserID,
		UserType:          req.UserType,
		Secret:            secret,
		Enabled:           true,
		Methods:           `["totp","backup"]`,
		BackupCodesHashed: string(backupCodesJSON),
		UsedBackupCodes:   `{}`,
		EnabledAt:         &now,
		LastVerifiedAt:    &lastVerified,
	}

	// Check if record exists
	var existing models.TwoFactorAuth
	err = s.db.Where("user_id = ? AND user_type = ?", req.UserID, req.UserType).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		err = s.db.Create(&twoFactor).Error
	} else {
		err = s.db.Model(&existing).Updates(map[string]interface{}{
			"secret":              secret,
			"enabled":             true,
			"methods":             `["totp","backup"]`,
			"backup_codes_hashed": string(backupCodesJSON),
			"used_backup_codes":   `{}`,
			"enabled_at":          time.Now(),
			"last_verified_at":    time.Now().Unix(),
		}).Error
	}

	if err != nil {
		return fmt.Errorf("failed to enable 2FA: %w", err)
	}

	// Delete temporary secret
	s.redis.Del(c.Request.Context(), redisKey)

	// Log activity
	logAdminActivity(s.db, req.UserID, "enable_2fa", "two_factor",
		fmt.Sprintf("%d", req.UserID), "2FA enabled", req.UserType, "")

	return nil
}

// Disable2FA disables 2FA for a user
func (s *TwoFactorService) Disable2FA(c *gin.Context, req Disable2FARequest) error {
	// Get user 2FA settings
	var twoFactor models.TwoFactorAuth
	err := s.db.Where("user_id = ? AND user_type = ?", req.UserID, req.UserType).First(&twoFactor).Error
	if err != nil {
		return fmt.Errorf("2FA not enabled for this user")
	}

	// Verify password (or code if provided)
	var passwordValid bool
	if req.BackupCode != "" {
		// Verify backup code
		codeHash := s.hashCode(strings.ToUpper(req.BackupCode))
		var usedCodes map[string]bool
		json.Unmarshal([]byte(twoFactor.UsedBackupCodes), &usedCodes)
		if usedCodes[codeHash] {
			passwordValid = false
		} else {
			backupCodes := s.getStoredBackupCodes(c, req.UserID, req.UserType)
			for _, code := range backupCodes {
				if s.hashCode(code) == codeHash {
					passwordValid = true
					break
				}
			}
		}
	} else if req.Code != "" {
		// Verify TOTP code
		passwordValid = totp.Validate(req.Code, twoFactor.Secret)
	} else {
		return fmt.Errorf("verification required")
	}

	if !passwordValid {
		return fmt.Errorf("invalid verification")
	}

	// Get password from user table and verify
	var storedPassword string
	switch req.UserType {
	case "admin":
		var admin models.Admin
		if err := s.db.First(&admin, req.UserID).Error; err != nil {
			return fmt.Errorf("user not found")
		}
		storedPassword = admin.PasswordHash
	case "user":
		var user models.User
		if err := s.db.First(&user, req.UserID).Error; err != nil {
			return fmt.Errorf("user not found")
		}
		storedPassword = user.PasswordHash
	}

	// Verify password (simplified - in production use proper bcrypt)
	inputHash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.Password)))
	if inputHash != storedPassword {
		return fmt.Errorf("invalid password")
	}

	// Disable 2FA
	err = s.db.Model(&twoFactor).Updates(map[string]interface{}{
		"enabled":          false,
		"disabled_at":      time.Now(),
		"last_verified_at": nil,
	}).Error

	if err != nil {
		return fmt.Errorf("failed to disable 2FA: %w", err)
	}

	// Log activity
	logAdminActivity(s.db, req.UserID, "disable_2fa", "two_factor",
		fmt.Sprintf("%d", req.UserID), "2FA disabled", req.UserType, "")

	return nil
}

// Get2FAStatus gets the 2FA status for a user
func (s *TwoFactorService) Get2FAStatus(c *gin.Context, userID uint, userType string) (*TwoFactorStatus, error) {
	var twoFactor models.TwoFactorAuth
	err := s.db.Where("user_id = ? AND user_type = ?", userID, userType).First(&twoFactor).Error

	status := &TwoFactorStatus{
		Enabled:           false,
		Methods:           []string{},
		BackupCodesLeft:   0,
		TrustedDevices:    0,
		RecoveryCodesLeft: 0,
	}

	if err == gorm.ErrRecordNotFound {
		return status, nil
	}
	if err != nil {
		return nil, err
	}

	status.Enabled = twoFactor.Enabled

	// Parse methods
	json.Unmarshal([]byte(twoFactor.Methods), &status.Methods)

	// Count backup codes left
	if twoFactor.BackupCodesHashed != "" {
		var usedCodes map[string]bool
		json.Unmarshal([]byte(twoFactor.UsedBackupCodes), &usedCodes)
		var totalCodes int
		json.Unmarshal([]byte(twoFactor.BackupCodesHashed), &totalCodes)
		status.BackupCodesLeft = totalCodes - len(usedCodes)
		status.RecoveryCodesLeft = status.BackupCodesLeft
	}

	if twoFactor.LastVerifiedAt != nil {
		status.LastVerifiedAt = twoFactor.LastVerifiedAt
	}

	return status, nil
}

// RegenerateBackupCodes generates new backup codes
func (s *TwoFactorService) RegenerateBackupCodes(c *gin.Context, userID uint, userType string, password string) ([]string, error) {
	// Verify password first
	var storedPassword string
	switch userType {
	case "admin":
		var admin models.Admin
		if err := s.db.First(&admin, userID).Error; err != nil {
			return nil, fmt.Errorf("user not found")
		}
		storedPassword = admin.PasswordHash
	case "user":
		var user models.User
		if err := s.db.First(&user, userID).Error; err != nil {
			return nil, fmt.Errorf("user not found")
		}
		storedPassword = user.PasswordHash
	}

	inputHash := fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
	if inputHash != storedPassword {
		return nil, fmt.Errorf("invalid password")
	}

	// Generate new backup codes
	newCodes := s.generateBackupCodes(10)

	// Hash and store
	backupCodesHashed := make(map[string]bool)
	for _, code := range newCodes {
		backupCodesHashed[strings.ToUpper(code)] = false
	}
	backupCodesJSON, _ := json.Marshal(backupCodesHashed)

	// Update database
	var twoFactor models.TwoFactorAuth
	err := s.db.Where("user_id = ? AND user_type = ?", userID, userType).First(&twoFactor).Error
	if err != nil {
		return nil, fmt.Errorf("2FA not enabled")
	}

	err = s.db.Model(&twoFactor).Updates(map[string]interface{}{
		"backup_codes_hashed": string(backupCodesJSON),
		"used_backup_codes":   `{}`,
	}).Error

	if err != nil {
		return nil, fmt.Errorf("failed to update backup codes: %w", err)
	}

	// Log activity
	logAdminActivity(s.db, userID, "regenerate_backup_codes", "two_factor",
		fmt.Sprintf("%d", userID), "Backup codes regenerated", userType, "")

	return newCodes, nil
}

// ValidateRateLimit validates rate limiting for 2FA
func (s *TwoFactorService) ValidateRateLimit(c *gin.Context, userID uint, userType string) error {
	rateLimitKey := fmt.Sprintf("2fa_rate_limit:%s:%d", userType, userID)

	count, err := s.redis.Get(c.Request.Context(), rateLimitKey).Int()
	if err == redis.Nil {
		s.redis.Set(c.Request.Context(), rateLimitKey, 1, 5*time.Minute)
		return nil
	}

	if err != nil {
		return err
	}

	if count >= 5 {
		return fmt.Errorf("too many failed attempts. Please try again in 5 minutes")
	}

	s.redis.Incr(c.Request.Context(), rateLimitKey)
	return nil
}

// helper functions

func (s *TwoFactorService) generateBackupCodes(count int) []string {
	codes := make([]string, count)
	encoded := base32.StdEncoding.EncodeToString(randBytes(count * 4))
	cleaned := strings.ReplaceAll(encoded, "=", "")

	for i := 0; i < count; i++ {
		start := i * 4
		end := start + 4
		if end > len(cleaned) {
			end = len(cleaned)
		}
		code := cleaned[start:end]
		codes[i] = fmt.Sprintf("%s-%s", code[:4], code[4:])
	}

	return codes
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func (s *TwoFactorService) hashCode(code string) string {
	hash := sha256.Sum256([]byte(strings.ToUpper(code)))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (s *TwoFactorService) getStoredBackupCodes(c *gin.Context, userID uint, userType string) []string {
	var twoFactor models.TwoFactorAuth
	err := s.db.Where("user_id = ? AND user_type = ?", userID, userType).First(&twoFactor).Error
	if err != nil {
		return []string{}
	}

	var hashedCodes map[string]bool
	json.Unmarshal([]byte(twoFactor.BackupCodesHashed), &hashedCodes)

	codes := make([]string, 0, len(hashedCodes))
	for code := range hashedCodes {
		// Decode from hash (in production, store in reversible way)
		codes = append(codes, code)
	}

	return codes
}

func (s *TwoFactorService) markBackupCodeUsed(c *gin.Context, userID uint, userType string, code string) {
	var twoFactor models.TwoFactorAuth
	s.db.Where("user_id = ? AND user_type = ?", userID, userID, userType).First(&twoFactor)

	var usedCodes map[string]bool
	json.Unmarshal([]byte(twoFactor.UsedBackupCodes), &usedCodes)
	usedCodes[strings.ToUpper(code)] = true

	usedJSON, _ := json.Marshal(usedCodes)
	s.db.Model(&twoFactor).Update("used_backup_codes", string(usedJSON))
}

func (s *TwoFactorService) logFailedAttempt(c *gin.Context, userID uint, userType string, ipAddress string) {
	attemptKey := fmt.Sprintf("2fa_failed:%s:%d", userType, userID)
	s.redis.Incr(c.Request.Context(), attemptKey)
	s.redis.Expire(c.Request.Context(), attemptKey, 24*time.Hour)

	// Log to database
	attempt := models.TwoFactorAttempt{
		UserID:      userID,
		UserType:    userType,
		IPAddress:   ipAddress,
		AttemptType: "verification_failed",
		Timestamp:   time.Now(),
	}
	s.db.Create(&attempt)
}

func (s *TwoFactorService) clearFailedAttempts(c *gin.Context, userID uint, userType string) {
	attemptKey := fmt.Sprintf("2fa_failed:%s:%d", userType, userID)
	s.redis.Del(c.Request.Context(), attemptKey)
}

// GenerateQRCode generates a QR code image for 2FA setup
func (s *TwoFactorService) GenerateQRCode(secret string, userName string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "TigerWallet",
		AccountName: userName,
		Algorithm:   otp.AlgorithmSHA256,
		Digits:      otp.DigitsSix,
		Period:      30,
		Secret:      []byte(secret),
	})
	if err != nil {
		return "", err
	}
	return key.URL(), nil
}

// CalculateCodeRemainingSeconds returns seconds until current code expires
func (s *TwoFactorService) CalculateCodeRemainingSeconds() int {
	epoch := time.Now().Unix()
	period := int64(30)
	remaining := period - (epoch % period)
	return int(remaining)
}

// ValidateTOTP validates a TOTP code without storing
func (s *TwoFactorService) ValidateTOTP(secret string, code string) bool {
	return totp.Validate(code, secret)
}

// GetCodeFromKey generates a TOTP code from a key (for testing)
func (s *TwoFactorService) GetCodeFromKey(secret string) (string, error) {
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "", err
	}
	return code, nil
}

// Round to specific decimal places
func round(val float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return math.Round(val*shift) / shift
}
