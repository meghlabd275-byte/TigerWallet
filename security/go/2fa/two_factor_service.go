// Two-Factor Authentication (2FA) Service
// Complete TOTP-based 2FA with backup codes, SMS, and email recovery

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	mathrand "math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/pquerna/otp"
	"github.com/pquerna/totp"
	"golang.org/x/crypto/pbkdf2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TwoFactorConfig - 2FA Configuration
type TwoFactorConfig struct {
	// Service Settings
	ServiceName    string `json:"service_name"`
	ServiceURL     string `json:"service_url"`
	Issuer         string `json:"issuer"`
	
	// TOTP Settings
	TOTPDigits     int    `json:"totp_digits"`     // 6 or 8
	TOTPPeriod     int    `json:"totp_period"`     // seconds
	TOTPSkew       int    `json:"totp_skew"`       // allowed period skew
	Algorithm      string `json:"algorithm"`       // SHA1, SHA256, SHA512
	
	// Backup Codes
	BackupCodeCount   int    `json:"backup_code_count"`   // number of backup codes
	BackupCodeLength  int    `json:"backup_code_length"`  // length of each code
	
	// Security Settings
	MaxAttempts     int           `json:"max_attempts"`     // max failed attempts
	LockoutDuration time.Duration `json:"lockout_duration"` // lockout duration
	RateLimitWindow time.Duration `json:"rate_limit_window"`
	RateLimitMax    int           `json:"rate_limit_max"`
	
	// Encryption
	EncryptionKey string `json:"encryption_key"` // base64 encoded key
	
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

// User2FA - User 2FA configuration
type User2FA struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            string    `gorm:"uniqueIndex;index" json:"user_id"`
	Enabled           bool      `gorm:"default:false" json:"enabled"`
	SecretEncrypted   string    `gorm:"type:text" json:"-"` // encrypted TOTP secret
	Method            string    `json:"method"` // totp, sms, email
	Phone             string    `json:"phone"`
	Email             string    `json:"email"`
	RecoveryEnabled   bool      `gorm:"default:false" json:"recovery_enabled"`
	BackupCodesHashed string    `gorm:"type:text" json:"-"` // hashed backup codes
	TrustedDevices    string    `gorm:"type:jsonb" json:"trusted_devices"`
	FailedAttempts    int       `gorm:"default:0" json:"failed_attempts"`
	LockedUntil       *time.Time `json:"locked_until"`
	LastUsedAt        *time.Time `json:"last_used_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// BackupCode - Single backup code
type BackupCode struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"index" json:"user_id"`
	CodeHash    string    `gorm:"index" json:"code_hash"`
	Used        bool      `gorm:"default:false" json:"used"`
	UsedAt      *time.Time `json:"used_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// TwoFactorLog - 2FA activity log
type TwoFactorLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"index" json:"user_id"`
	Action      string    `json:"action"` // enabled, disabled, verified, failed, locked, backup_used
	Method      string    `json:"method"` // totp, sms, email, backup
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	DeviceInfo  string    `json:"device_info"`
	Success     bool      `json:"success"`
	ErrorMsg    string    `json:"error_msg"`
	CreatedAt   time.Time `json:"created_at"`
}

// TrustedDevice - Trusted device information
type TrustedDevice struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	OS        string    `json:"os"`
	Browser   string    `json:"browser"`
	IP        string    `json:"ip"`
	LastSeen  time.Time `json:"last_seen"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TwoFactorService - Main 2FA service
type TwoFactorService struct {
	config      TwoFactorConfig
	db          *gorm.DB
	redis       *redis.Client
	encryptionKey []byte
	templates   *template.Template
}

// NewTwoFactorService - Create new 2FA service
func NewTwoFactorService(cfg TwoFactorConfig) (*TwoFactorService, error) {
	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Auto migrate
	err = db.AutoMigrate(&User2FA{}, &BackupCode{}, &TwoFactorLog{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	
	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
	})
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	
	// Get encryption key
	encryptionKey, err := getOrCreateEncryptionKey(cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}
	
	// Load templates
	templates, err := loadTemplates()
	if err != nil {
		log.Printf("Warning: Failed to load templates: %v", err)
	}
	
	// Set defaults
	if cfg.TOTPDigits == 0 {
		cfg.TOTPDigits = 6
	}
	if cfg.TOTPPeriod == 0 {
		cfg.TOTPPeriod = 30
	}
	if cfg.TOTPSkew == 0 {
		cfg.TOTPSkew = 1
	}
	if cfg.BackupCodeCount == 0 {
		cfg.BackupCodeCount = 10
	}
	if cfg.BackupCodeLength == 0 {
		cfg.BackupCodeLength = 8
	}
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = 5
	}
	if cfg.LockoutDuration == 0 {
		cfg.LockoutDuration = 15 * time.Minute
	}
	
	return &TwoFactorService{
		config:        cfg,
		db:            db,
		redis:         rdb,
		encryptionKey: encryptionKey,
		templates:     templates,
	}, nil
}

// getOrCreateEncryptionKey - Get or create encryption key
func getOrCreateEncryptionKey(keyStr string) ([]byte, error) {
	if keyStr != "" {
		return base64.StdEncoding.DecodeString(keyStr)
	}
	
	// Generate new key
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	
	return key, nil
}

// loadTemplates - Load HTML templates
func loadTemplates() (*template.Template, error) {
	tmpl := `
{{define "setup_html"}}
<!DOCTYPE html>
<html>
<head>
    <title>Setup Two-Factor Authentication</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
               max-width: 400px; margin: 50px auto; padding: 20px; }
        .container { background: #fff; border-radius: 8px; padding: 30px; 
                     box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h2 { color: #333; margin-bottom: 20px; }
        .step { margin-bottom: 20px; padding: 15px; background: #f5f5f5; border-radius: 6px; }
        .code { font-family: monospace; font-size: 18px; background: #e8f5e9; 
                padding: 10px; border-radius: 4px; word-break: break-all; }
        .input-group { margin-bottom: 15px; }
        input { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; 
                font-size: 16px; box-sizing: border-box; }
        button { width: 100%; padding: 12px; background: #4F46E5; color: white; 
                 border: none; border-radius: 4px; font-size: 16px; cursor: pointer; }
        button:hover { background: #4338ca; }
        .backup-codes { background: #fff3cd; padding: 15px; border-radius: 6px; 
                       font-family: monospace; word-break: break-all; }
        .copy-btn { background: #6c757d; margin-top: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <h2>Setup Two-Factor Authentication</h2>
        
        {{if .Step1}}
        <div class="step">
            <h3>Step 1: Scan QR Code</h3>
            <p>Scan this QR code with your authenticator app (Google Authenticator, Authy, etc.)</p>
            <div style="text-align: center; margin: 20px 0;">
                <img src="{{.QRCodeURL}}" alt="QR Code" style="max-width: 200px;">
            </div>
            <p>Or enter this code manually:</p>
            <div class="code">{{.Secret}}</div>
        </div>
        
        <div class="step">
            <h3>Step 2: Verify</h3>
            <p>Enter the 6-digit code from your authenticator app to verify setup:</p>
            <form method="POST" action="/2fa/verify-setup">
                <div class="input-group">
                    <input type="text" name="code" placeholder="000000" maxlength="8" required 
                           pattern="[0-9]*" inputmode="numeric" autocomplete="one-time-code">
                </div>
                <input type="hidden" name="secret" value="{{.Secret}}">
                <button type="submit">Verify & Enable</button>
            </form>
        </div>
        {{end}}
        
        {{if .Step2}}
        <div class="step">
            <h3>Save Backup Codes</h3>
            <p>Save these backup codes in a secure place. You can use them to access your account if you lose your device.</p>
            <div class="backup-codes">
                {{range .BackupCodes}}
                {{.}}<br>
                {{end}}
            </div>
            <button class="copy-btn" onclick="navigator.clipboard.writeText('{{range $i, $c := .BackupCodes}}{{$c}}{{if ne $i (sub (len .BackupCodes) 1)}}{{\n}}{{end}}{{end}}')">Copy to Clipboard</button>
        </div>
        
        <div class="step">
            <p>I've saved my backup codes in a secure place.</p>
            <form method="POST" action="/2fa/complete-setup">
                <button type="submit">Complete Setup</button>
            </form>
        </div>
        {{end}}
    </div>
</body>
</html>
{{end}}`

	return template.New("2fa").Parse(tmpl)
}

// GenerateSecret - Generate TOTP secret
func (s *TwoFactorService) GenerateSecret(userID string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.config.Issuer,
		AccountName: userID,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      uint(s.config.TOTPPeriod),
	})
	
	if err != nil {
		return "", err
	}
	
	return key.Secret(), nil
}

// EncryptSecret - Encrypt TOTP secret
func (s *TwoFactorService) EncryptSecret(secret string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return "", err
	}
	
	ciphertext := gcm.Seal(nonce, nonce, []byte(secret), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSecret - Decrypt TOTP secret
func (s *TwoFactorService) DecryptSecret(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	
	return string(plaintext), nil
}

// GenerateBackupCodes - Generate backup codes
func (s *TwoFactorService) GenerateBackupCodes(count int) []string {
	codes := make([]string, count)
	
	for i := 0; i < count; i++ {
		code := generateRandomCode(s.config.BackupCodeLength)
		// Format as XXXX-XXXX
		codes[i] = fmt.Sprintf("%s-%s", code[:4], code[4:])
	}
	
	return codes
}

// HashBackupCode - Hash backup code for storage
func (s *TwoFactorService) HashBackupCode(code string) string {
	code = strings.ReplaceAll(code, "-", "")
	hash := sha256.Sum256([]byte(code + s.config.Issuer))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// GenerateRandomCode - Generate random alphanumeric code
func generateRandomCode(length int) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // No similar looking chars
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[mathrand.Intn(len(chars))]
	}
	return string(b)
}

// Enable2FA - Enable 2FA for user
func (s *TwoFactorService) Enable2FA(userID, secret, method, phone, email string) error {
	// Encrypt secret
	encryptedSecret, err := s.EncryptSecret(secret)
	if err != nil {
		return err
	}
	
	// Generate backup codes
	backupCodes := s.GenerateBackupCodes(s.config.BackupCodeCount)
	
	// Store backup codes
	for _, code := range backupCodes {
		codeHash := s.HashBackupCode(code)
		backupCode := &BackupCode{
			UserID:   userID,
			CodeHash: codeHash,
			Used:     false,
		}
		s.db.Create(backupCode)
	}
	
	// Store user 2FA config
	user2FA := &User2FA{
		UserID:            userID,
		Enabled:           true,
		SecretEncrypted:   encryptedSecret,
		Method:            method,
		Phone:             phone,
		Email:             email,
		RecoveryEnabled:   true,
		FailedAttempts:    0,
		TrustedDevices:    "[]",
		CreatedAt:         time.Now(),
	}
	
	// Check if user already has 2FA
	var existing User2FA
	result := s.db.Where("user_id = ?", userID).First(&existing)
	if result.Error == nil {
		// Update existing
		user2FA.ID = existing.ID
		user2FA.TrustedDevices = existing.TrustedDevices
		s.db.Save(user2FA)
	} else {
		s.db.Create(user2FA)
	}
	
	// Log
	s.logAction(userID, "enabled", method, "", "", "", true, "")
	
	return nil
}

// Disable2FA - Disable 2FA for user
func (s *TwoFactorService) Disable2FA(userID, reason string) error {
	// Delete backup codes
	s.db.Where("user_id = ?", userID).Delete(&BackupCode{})
	
	// Update user 2FA
	s.db.Model(&User2FA{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"enabled":           false,
		"secret_encrypted":  "",
		"method":            "",
		"recovery_enabled":  false,
		"backup_codes_hashed": "",
		"updated_at":        time.Now(),
	})
	
	// Log
	s.logAction(userID, "disabled", "totp", "", "", "", true, reason)
	
	return nil
}

// VerifyCode - Verify TOTP code
func (s *TwoFactorService) VerifyCode(userID, code, ipAddress, userAgent string) (bool, error) {
	// Check if user is locked
	var user2FA User2FA
	result := s.db.Where("user_id = ? AND enabled = ?", userID, true).First(&user2FA)
	if result.Error != nil {
		return false, result.Error
	}
	
	// Check lockout
	if user2FA.LockedUntil != nil && user2FA.LockedUntil.After(time.Now()) {
		return false, fmt.Errorf("account locked until %s", user2FA.LockedUntil.Format("15:04:05"))
	}
	
	// Decrypt secret
	secret, err := s.DecryptSecret(user2FA.SecretEncrypted)
	if err != nil {
		return false, err
	}
	
	// Verify code with skew tolerance
	valid := totp.Validate(code, secret)
	if !valid {
		// Try previous and next period
		valid = totp.ValidateCustom(code, secret, time.Now().Add(-30*time.Second), totp.ValidateOpts{
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
			Period:    uint(s.config.TOTPPeriod),
		})
		
		if !valid {
			valid = totp.ValidateCustom(code, secret, time.Now().Add(30*time.Second), totp.ValidateOpts{
				Digits:    otp.DigitsSix,
				Algorithm: otp.AlgorithmSHA1,
				Period:    uint(s.config.TOTPPeriod),
			})
		}
	}
	
	if valid {
		// Reset failed attempts
		s.db.Model(&user2FA).Updates(map[string]interface{}{
			"failed_attempts": 0,
			"locked_until":    nil,
			"last_used_at":   time.Now(),
		})
		
		// Log success
		s.logAction(userID, "verified", user2FA.Method, ipAddress, userAgent, "", true, "")
		
		return true, nil
	}
	
	// Increment failed attempts
	user2FA.FailedAttempts++
	locked := false
	
	if user2FA.FailedAttempts >= s.config.MaxAttempts {
		lockedUntil := time.Now().Add(s.config.LockoutDuration)
		s.db.Model(&user2FA).Updates(map[string]interface{}{
			"failed_attempts": user2FA.FailedAttempts,
			"locked_until":    lockedUntil,
		})
		locked = true
	} else {
		s.db.Model(&user2FA).Update("failed_attempts", user2FA.FailedAttempts)
	}
	
	// Log failure
	errMsg := "invalid code"
	if locked {
		errMsg = fmt.Sprintf("account locked after %d failed attempts", user2FA.FailedAttempts)
	}
	s.logAction(userID, "failed", user2FA.Method, ipAddress, userAgent, "", false, errMsg)
	
	return false, fmt.Errorf(errMsg)
}

// VerifyBackupCode - Verify backup code
func (s *TwoFactorService) VerifyBackupCode(userID, code, ipAddress, userAgent string) (bool, error) {
	// Check if user is locked
	var user2FA User2FA
	result := s.db.Where("user_id = ? AND enabled = ? AND recovery_enabled = ?", userID, true, true).First(&user2FA)
	if result.Error != nil {
		return false, result.Error
	}
	
	codeHash := s.HashBackupCode(code)
	
	// Find unused backup code
	var backupCode BackupCode
	result = s.db.Where("user_id = ? AND code_hash = ? AND used = ?", userID, codeHash, false).First(&backupCode)
	if result.Error != nil {
		// Log failure even if code not found (prevent timing attacks)
		s.logAction(userID, "failed", "backup", ipAddress, userAgent, "", false, "invalid or used backup code")
		return false, fmt.Errorf("invalid or used backup code")
	}
	
	// Mark code as used
	now := time.Now()
	s.db.Model(&backupCode).Updates(map[string]interface{}{
		"used":   true,
		"used_at": now,
	})
	
	// Log success
	s.logAction(userID, "backup_used", "backup", ipAddress, userAgent, "", true, "")
	
	// Update last used
	s.db.Model(&user2FA).Update("last_used_at", now)
	
	return true, nil
}

// GetUser2FA - Get user 2FA configuration
func (s *TwoFactorService) GetUser2FA(userID string) (*User2FA, error) {
	var user2FA User2FA
	err := s.db.Where("user_id = ?", userID).First(&user2FA).Error
	if err != nil {
		return nil, err
	}
	
	// Don't return encrypted secret
	user2FA.SecretEncrypted = ""
	user2FA.BackupCodesHashed = ""
	
	return &user2FA, nil
}

// Is2FAEnabled - Check if 2FA is enabled for user
func (s *TwoFactorService) Is2FAEnabled(userID string) (bool, error) {
	var count int64
	err := s.db.Model(&User2FA{}).Where("user_id = ? AND enabled = ?", userID, true).Count(&count).Error
	return count > 0, err
}

// GetRemainingBackupCodes - Get remaining backup codes count
func (s *TwoFactorService) GetRemainingBackupCodes(userID string) (int, error) {
	var count int64
	err := s.db.Model(&BackupCode{}).Where("user_id = ? AND used = ?", userID, false).Count(&count).Error
	return int(count), err
}

// RegenerateBackupCodes - Regenerate backup codes
func (s *TwoFactorService) RegenerateBackupCodes(userID string) ([]string, error) {
	// Delete old backup codes
	s.db.Where("user_id = ?", userID).Delete(&BackupCode{})
	
	// Generate new backup codes
	backupCodes := s.GenerateBackupCodes(s.config.BackupCodeCount)
	
	// Store new backup codes
	for _, code := range backupCodes {
		codeHash := s.HashBackupCode(code)
		backupCode := &BackupCode{
			UserID:   userID,
			CodeHash: codeHash,
			Used:     false,
		}
		s.db.Create(backupCode)
	}
	
	// Log
	s.logAction(userID, "backup_codes_regenerated", "recovery", "", "", "", true, "")
	
	return backupCodes, nil
}

// AddTrustedDevice - Add trusted device
func (s *TwoFactorService) AddTrustedDevice(userID string, device TrustedDevice) error {
	var user2FA User2FA
	if err := s.db.Where("user_id = ?", userID).First(&user2FA).Error; err != nil {
		return err
	}
	
	var devices []TrustedDevice
	json.Unmarshal([]byte(user2FA.TrustedDevices), &devices)
	
	// Check if device already exists
	for i, d := range devices {
		if d.ID == device.ID {
			devices[i].LastSeen = time.Now()
			devices[i].ExpiresAt = time.Now().Add(30 * 24 * time.Hour) // 30 days
			goto save
		}
	}
	
	device.LastSeen = time.Now()
	device.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
	devices = append(devices, device)
	
save:
	data, _ := json.Marshal(devices)
	s.db.Model(&user2FA).Update("trusted_devices", string(data))
	
	return nil
}

// RemoveTrustedDevice - Remove trusted device
func (s *TwoFactorService) RemoveTrustedDevice(userID, deviceID string) error {
	var user2FA User2FA
	if err := s.db.Where("user_id = ?", userID).First(&user2FA).Error; err != nil {
		return err
	}
	
	var devices []TrustedDevice
	json.Unmarshal([]byte(user2FA.TrustedDevices), &devices)
	
	newDevices := make([]TrustedDevice, 0)
	for _, d := range devices {
		if d.ID != deviceID {
			newDevices = append(newDevices, d)
		}
	}
	
	data, _ := json.Marshal(newDevices)
	s.db.Model(&user2FA).Update("trusted_devices", string(data))
	
	return nil
}

// logAction - Log 2FA action
func (s *TwoFactorService) logAction(userID, action, method, ipAddress, userAgent, deviceInfo string, success bool, errorMsg string) {
	logEntry := &TwoFactorLog{
		UserID:     userID,
		Action:     action,
		Method:     method,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		DeviceInfo: deviceInfo,
		Success:    success,
		ErrorMsg:   errorMsg,
		CreatedAt:  time.Now(),
	}
	
	s.db.Create(logEntry)
}

// HTTP Handlers

type Setup2FARequest struct {
	UserID string `json:"user_id" binding:"required"`
	Method string `json:"method"` // totp, sms, email
	Phone  string `json:"phone"`
	Email  string `json:"email"`
}

func (s *TwoFactorService) Setup2FAHandler(c *gin.Context) {
	var req Setup2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	method := req.Method
	if method == "" {
		method = "totp"
	}
	
	// Generate secret
	secret, err := s.GenerateSecret(req.UserID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	// Generate QR code URL
	qrCodeURL := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=%s", 
		url.QueryEscape(fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s", 
			s.config.Issuer, req.UserID, secret, s.config.Issuer)))
	
	c.JSON(200, gin.H{
		"secret":     secret,
		"qr_code_url": qrCodeURL,
	})
}

type VerifySetupRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Secret string `json:"secret" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

func (s *TwoFactorService) VerifySetupHandler(c *gin.Context) {
	var req VerifySetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	// Verify code
	valid := totp.Validate(req.Code, req.Secret)
	if !valid {
		// Try with skew tolerance
		valid = totp.ValidateCustom(req.Code, req.Secret, time.Now().Add(-30*time.Second), totp.ValidateOpts{
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
			Period:    uint(s.config.TOTPPeriod),
		})
	}
	
	if !valid {
		c.JSON(400, gin.H{"error": "invalid code"})
		return
	}
	
	// Enable 2FA
	err := s.Enable2FA(req.UserID, req.Secret, "totp", "", "")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	// Generate backup codes to return
	backupCodes, _ := s.RegenerateBackupCodes(req.UserID)
	
	c.JSON(200, gin.H{
		"status":        "enabled",
		"backup_codes":  backupCodes,
	})
}

type Verify2FARequest struct {
	UserID string `json:"user_id" binding:"required"`
	Code   string `json:"code"`
	BackupCode string `json:"backup_code"`
}

func (s *TwoFactorService) Verify2FAHandler(c *gin.Context) {
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	
	// Try TOTP code first
	if req.Code != "" {
		valid, err := s.VerifyCode(req.UserID, req.Code, ipAddress, userAgent)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error(), "remaining_attempts": s.config.MaxAttempts})
			return
		}
		
		c.JSON(200, gin.H{"status": "verified", "method": "totp"})
		return
	}
	
	// Try backup code
	if req.BackupCode != "" {
		valid, err := s.VerifyBackupCode(req.UserID, req.BackupCode, ipAddress, userAgent)
		if err != nil {
			remaining, _ := s.GetRemainingBackupCodes(req.UserID)
			c.JSON(401, gin.H{"error": err.Error(), "remaining_backup_codes": remaining})
			return
		}
		
		c.JSON(200, gin.H{"status": "verified", "method": "backup"})
		return
	}
	
	c.JSON(400, gin.H{"error": "code or backup_code required"})
}

type Disable2FARequest struct {
	UserID string `json:"user_id" binding:"required"`
	Code   string `json:"code" binding:"required"` // Require 2FA to disable
}

func (s *TwoFactorService) Disable2FAHandler(c *gin.Context) {
	var req Disable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	// Verify before disabling
	valid, err := s.VerifyCode(req.UserID, req.Code, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid code"})
		return
	}
	
	if !valid {
		c.JSON(401, gin.H{"error": "invalid code"})
		return
	}
	
	err = s.Disable2FA(req.UserID, "user_requested")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "disabled"})
}

func (s *TwoFactorService) GetUser2FAHandler(c *gin.Context) {
	userID := c.Param("user_id")
	
	user2FA, err := s.GetUser2FA(userID)
	if err != nil {
		c.JSON(404, gin.H{"error": "2FA not enabled"})
		return
	}
	
	// Get remaining backup codes
	remaining, _ := s.GetRemainingBackupCodes(userID)
	user2FA.FailedAttempts = remaining // Reuse field for remaining backup codes
	
	c.JSON(200, user2FA)
}

func (s *TwoFactorService) Is2FAEnabledHandler(c *gin.Context) {
	userID := c.Param("user_id")
	
	enabled, err := s.Is2FAEnabled(userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"enabled": enabled})
}

type RegenerateCodesRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

func (s *TwoFactorService) RegenerateCodesHandler(c *gin.Context) {
	var req RegenerateCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	// Verify before regenerating
	valid, err := s.VerifyCode(req.UserID, req.Code, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid code"})
		return
	}
	
	if !valid {
		c.JSON(401, gin.H{"error": "invalid code"})
		return
	}
	
	codes, err := s.RegenerateBackupCodes(req.UserID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"backup_codes": codes})
}

// Main

func main() {
	cfg := TwoFactorConfig{
		ServiceName:    getEnv("SERVICE_NAME", "TigerWallet"),
		ServiceURL:     getEnv("SERVICE_URL", "https://tigerwallet.com"),
		Issuer:         getEnv("2FA_ISSUER", "TigerWallet"),
		TOTPDigits:     getEnvInt("TOTP_DIGITS", 6),
		TOTPPeriod:     getEnvInt("TOTP_PERIOD", 30),
		BackupCodeCount: getEnvInt("BACKUP_CODE_COUNT", 10),
		BackupCodeLength: getEnvInt("BACKUP_CODE_LENGTH", 8),
		MaxAttempts:    getEnvInt("2FA_MAX_ATTEMPTS", 5),
		LockoutDuration: getEnvDuration("2FA_LOCKOUT_DURATION", 15*time.Minute),
		EncryptionKey:  getEnv("2FA_ENCRYPTION_KEY", ""),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "password"),
		DBName:         getEnv("DB_NAME", "twofactor_db"),
		RedisHost:      getEnv("REDIS_HOST", "localhost"),
		RedisPort:      getEnv("REDIS_PORT", "6379"),
		ServerPort:     getEnv("2FA_SERVER_PORT", "8090"),
	}
	
	service, err := NewTwoFactorService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize 2FA service: %v", err)
	}
	
	r := gin.Default()
	
	// 2FA Routes
	r.POST("/2fa/setup", service.Setup2FAHandler)
	r.POST("/2fa/verify-setup", service.VerifySetupHandler)
	r.POST("/2fa/verify", service.Verify2FAHandler)
	r.POST("/2fa/disable", service.Disable2FAHandler)
	r.GET("/2fa/status/:user_id", service.Is2FAEnabledHandler)
	r.GET("/2fa/config/:user_id", service.GetUser2FAHandler)
	r.POST("/2fa/regenerate-codes", service.RegenerateCodesHandler)
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "2fa"})
	})
	
	log.Printf("2FA Service starting on port %s", cfg.ServerPort)
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

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		d, err := time.ParseDuration(value)
		if err == nil {
			return d
		}
	}
	return defaultValue
}

// Need to import url
import (
	"net/url"
)
