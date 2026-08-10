package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port       string
	IssuerName string
	SMTPServer string
	SMTPPort   int
	SMTPUser   string
	SMTPPass   string
	FromEmail  string
}

func LoadConfig() *Config {
	return &Config{
		Port:       getEnv("PORT", "8446"),
		IssuerName: getEnv("ISSUER_NAME", "TigerWallet"),
		SMTPServer: getEnv("SMTP_SERVER", "smtp.gmail.com"),
		SMTPPort:   587,
		SMTPUser:   getEnv("SMTP_USER", ""),
		SMTPPass:   getEnv("SMTP_PASS", ""),
		FromEmail:  getEnv("FROM_EMAIL", "noreply@tigerwallet.com"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Models
// ============================================================================

type User2FA struct {
	UserID         string    `json:"userId"`
	Email          string    `json:"email"`
	Secret         string    `json:"secret,omitempty"`
	Enabled        bool      `json:"enabled"`
	BackupCodes    []string  `json:"backupCodes,omitempty"`
	TrustedDevices []string  `json:"trustedDevices"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type TOTPRequest struct {
	Code   string `json:"code" binding:"required"`
	UserID string `json:"userId" binding:"required"`
}

type Enable2FARequest struct {
	UserID string `json:"userId" binding:"required"`
	Email  string `json:"email" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

type VerifyRequest struct {
	UserID string `json:"userId" binding:"required"`
	Code   string `json:"code" binding:"required"`
	Type   string `json:"type"` // totp, backup, sms, email
}

type VerificationCode struct {
	Code      string
	ExpiresAt time.Time
	Attempts  int
}

type BackupCode struct {
	Code   string
	Used   bool
	UsedAt *time.Time
}

type WebAuthnCredential struct {
	CredentialID string     `json:"credentialId"`
	PublicKey    string     `json:"publicKey"`
	Counter      int64      `json:"counter"`
	DeviceName   string     `json:"deviceName"`
	Transport    string     `json:"transport"` // usb, nfc, hybrid
	CreatedAt    time.Time  `json:"createdAt"`
	LastUsed     *time.Time `json:"lastUsed"`
}

// ============================================================================
// 2FA Service
// ============================================================================

type TwoFactorService struct {
	config   *Config
	users    map[string]*User2FA
	pending  map[string]*VerificationCode
	attempts map[string]int
	mu       sync.RWMutex
	webAuthn map[string][]WebAuthnCredential
}

func NewTwoFactorService(config *Config) *TwoFactorService {
	return &TwoFactorService{
		config:   config,
		users:    make(map[string]*User2FA),
		pending:  make(map[string]*VerificationCode),
		attempts: make(map[string]int),
		webAuthn: make(map[string][]WebAuthnCredential),
	}
}

// ============================================================================
// TOTP Functions
// ============================================================================

func (s *TwoFactorService) GenerateSecret(userID, email string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.config.IssuerName,
		AccountName: email,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", err
	}

	// Use the secret from the generated key
	secret := key.Secret()

	// Generate backup codes
	backupCodes := s.generateBackupCodes(10)

	user := &User2FA{
		UserID:         userID,
		Email:          email,
		Secret:         secret,
		Enabled:        false,
		BackupCodes:    backupCodes,
		TrustedDevices: []string{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	s.mu.Lock()
	s.users[userID] = user
	s.mu.Unlock()

	return secret, nil
}

func (s *TwoFactorService) GenerateQRCodeURL(secret, userID, email string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		s.config.IssuerName, email, secret, s.config.IssuerName)
}

func (s *TwoFactorService) ValidateCode(userID, code string) bool {
	s.mu.RLock()
	user, ok := s.users[userID]
	s.mu.RUnlock()

	if !ok || !user.Enabled {
		return false
	}

	// Validate TOTP
	valid := totp.Validate(code, user.Secret)
	if valid {
		return true
	}

	// Check backup codes
	for i, bc := range user.BackupCodes {
		if bc == code && !strings.HasPrefix(bc, "USED_") {
			user.BackupCodes[i] = "USED_" + bc
			return true
		}
	}

	return false
}

func (s *TwoFactorService) Enable2FA(userID, email, code string) error {
	s.mu.RLock()
	user, ok := s.users[userID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("user not found")
	}

	// Validate the code
	if !totp.Validate(code, user.Secret) {
		return fmt.Errorf("invalid code")
	}

	user.Enabled = true
	user.UpdatedAt = time.Now()

	return nil
}

func (s *TwoFactorService) Disable2FA(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}

	user.Enabled = false
	user.UpdatedAt = time.Now()

	return nil
}

func (s *TwoFactorService) GetUser2FA(userID string) *User2FA {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users[userID]
}

// ============================================================================
// Backup Codes
// ============================================================================

func (s *TwoFactorService) generateBackupCodes(count int) []string {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		codes[i] = s.generateSecureCode(8)
	}
	return codes
}

func (s *TwoFactorService) generateSecureCode(length int) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// ============================================================================
// SMS/Email Verification
// ============================================================================

func (s *TwoFactorService) SendVerificationCode(userID, method, destination string) error {
	code := s.generateSecureCode(6)

	// Store pending code
	s.mu.Lock()
	s.pending[userID] = &VerificationCode{
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Attempts:  0,
	}
	s.mu.Unlock()

	switch method {
	case "sms":
		return s.sendSMS(destination, code)
	case "email":
		return s.sendEmail(destination, code)
	default:
		return fmt.Errorf("unsupported method: %s", method)
	}
}

func (s *TwoFactorService) sendSMS(phone, code string) error {
	// In production, integrate with Twilio or similar
	log.Printf("SMS would be sent to %s: Your verification code is %s", phone, code)
	return nil
}

func (s *TwoFactorService) sendEmail(email, code string) error {
	// In production, use actual SMTP
	log.Printf("Email would be sent to %s: Your verification code is %s", email, code)
	return nil
}

func (s *TwoFactorService) VerifyPendingCode(userID, code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	pending, ok := s.pending[userID]
	if !ok {
		return false
	}

	if time.Now().After(pending.ExpiresAt) {
		delete(s.pending, userID)
		return false
	}

	if pending.Code != code {
		pending.Attempts++
		if pending.Attempts >= 3 {
			delete(s.pending, userID)
		}
		return false
	}

	delete(s.pending, userID)
	return true
}

// ============================================================================
// WebAuthn (Passkey) Functions
// ============================================================================

func (s *TwoFactorService) RegisterWebAuthn(userID, deviceName string) (string, error) {
	// Generate credential ID
	credentialID := make([]byte, 32)
	rand.Read(credentialID)
	credID := base64.URLEncoding.EncodeToString(credentialID)

	// Generate a random public key (in production, this would come from the browser)
	pubKey := make([]byte, 64)
	rand.Read(pubKey)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	credential := WebAuthnCredential{
		CredentialID: credID,
		PublicKey:    pubKeyB64,
		Counter:      0,
		DeviceName:   deviceName,
		Transport:    "hybrid",
		CreatedAt:    time.Now(),
	}

	s.mu.Lock()
	s.webAuthn[userID] = append(s.webAuthn[userID], credential)
	s.mu.Unlock()

	return credID, nil
}

func (s *TwoFactorService) VerifyWebAuthn(userID, credentialID, clientDataJSON, authenticatorData, signature string) bool {
	s.mu.RLock()
	credentials, ok := s.webAuthn[userID]
	s.mu.RUnlock()

	if !ok {
		return false
	}

	for _, cred := range credentials {
		if cred.CredentialID == credentialID {
			// In production, verify the signature properly
			// For now, just check credential exists
			return true
		}
	}

	return false
}

func (s *TwoFactorService) GetWebAuthnCredentials(userID string) []WebAuthnCredential {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.webAuthn[userID]
}

func (s *TwoFactorService) DeleteWebAuthnCredential(userID, credentialID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	credentials := s.webAuthn[userID]
	for i, cred := range credentials {
		if cred.CredentialID == credentialID {
			s.webAuthn[userID] = append(credentials[:i], credentials[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("credential not found")
}

// ============================================================================
// Rate Limiting
// ============================================================================

func (s *TwoFactorService) CheckRateLimit(identifier string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	attempts := s.attempts[identifier]
	if attempts >= 5 {
		return false
	}

	s.attempts[identifier] = attempts + 1

	// Reset after 15 minutes
	go func() {
		time.Sleep(15 * time.Minute)
		s.mu.Lock()
		delete(s.attempts, identifier)
		s.mu.Unlock()
	}()

	return true
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *TwoFactorService) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "2fa-service"})
	})

	api := r.Group("/api/v1/2fa")
	{
		// TOTP Setup
		api.POST("/setup", s.handleSetup2FA)
		api.POST("/enable", s.handleEnable2FA)
		api.POST("/disable", s.handleDisable2FA)
		api.POST("/verify", s.handleVerify2FA)
		api.GET("/status/:userId", s.handleGet2FAStatus)

		// Backup Codes
		api.POST("/backup-codes/regenerate", s.handleRegenerateBackupCodes)

		// SMS/Email Verification
		api.POST("/send-code", s.handleSendCode)
		api.POST("/verify-code", s.handleVerifyCode)

		// WebAuthn (Passkey)
		api.POST("/webauthn/register", s.handleWebAuthnRegister)
		api.POST("/webauthn/verify", s.handleWebAuthnVerify)
		api.GET("/webauthn/credentials/:userId", s.handleGetWebAuthnCredentials)
		api.DELETE("/webauthn/credentials/:userId/:credentialId", s.handleDeleteWebAuthnCredential)
	}
}

func (s *TwoFactorService) handleSetup2FA(c *gin.Context) {
	var req struct {
		UserID string `json:"userId" binding:"required"`
		Email  string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !s.CheckRateLimit(req.UserID + ":setup") {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many attempts. Please try again later."})
		return
	}

	secret, err := s.GenerateSecret(req.UserID, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	qrURL := s.GenerateQRCodeURL(secret, req.UserID, req.Email)

	c.JSON(http.StatusOK, gin.H{
		"secret": secret,
		"qrUrl":  qrURL,
	})
}

func (s *TwoFactorService) handleEnable2FA(c *gin.Context) {
	var req Enable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !s.CheckRateLimit(req.UserID + ":enable") {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many attempts"})
		return
	}

	if err := s.Enable2FA(req.UserID, req.Email, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA enabled successfully"})
}

func (s *TwoFactorService) handleDisable2FA(c *gin.Context) {
	var req struct {
		UserID string `json:"userId" binding:"required"`
		Code   string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !s.ValidateCode(req.UserID, req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code"})
		return
	}

	if err := s.Disable2FA(req.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA disabled successfully"})
}

func (s *TwoFactorService) handleVerify2FA(c *gin.Context) {
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !s.CheckRateLimit(req.UserID + ":verify") {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many attempts"})
		return
	}

	var valid bool
	switch req.Type {
	case "totp":
		valid = s.ValidateCode(req.UserID, req.Code)
	case "backup":
		valid = s.ValidateCode(req.UserID, req.Code)
	case "sms", "email":
		valid = s.VerifyPendingCode(req.UserID, req.Code)
	default:
		valid = s.ValidateCode(req.UserID, req.Code)
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

func (s *TwoFactorService) handleGet2FAStatus(c *gin.Context) {
	userID := c.Param("userId")
	user := s.GetUser2FA(userID)

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":     user.Enabled,
		"backupCodes": len(user.BackupCodes),
		"webAuthn":    len(s.GetWebAuthnCredentials(userID)),
	})
}

func (s *TwoFactorService) handleRegenerateBackupCodes(c *gin.Context) {
	var req struct {
		UserID string `json:"userId" binding:"required"`
		Code   string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !s.ValidateCode(req.UserID, req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code"})
		return
	}

	s.mu.Lock()
	if user, ok := s.users[req.UserID]; ok {
		user.BackupCodes = s.generateBackupCodes(10)
		user.UpdatedAt = time.Now()
	}
	s.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "Backup codes regenerated"})
}

func (s *TwoFactorService) handleSendCode(c *gin.Context) {
	var req struct {
		UserID      string `json:"userId" binding:"required"`
		Method      string `json:"method" binding:"required"` // sms, email
		Destination string `json:"destination" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !s.CheckRateLimit(req.UserID + ":send") {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many attempts"})
		return
	}

	if err := s.SendVerificationCode(req.UserID, req.Method, req.Destination); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification code sent"})
}

func (s *TwoFactorService) handleVerifyCode(c *gin.Context) {
	var req struct {
		UserID string `json:"userId" binding:"required"`
		Code   string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid := s.VerifyPendingCode(req.UserID, req.Code)

	c.JSON(http.StatusOK, gin.H{"valid": valid})
}

func (s *TwoFactorService) handleWebAuthnRegister(c *gin.Context) {
	var req struct {
		UserID     string `json:"userId" binding:"required"`
		DeviceName string `json:"deviceName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	credID, err := s.RegisterWebAuthn(req.UserID, req.DeviceName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"credentialId": credID})
}

func (s *TwoFactorService) handleWebAuthnVerify(c *gin.Context) {
	var req struct {
		UserID            string `json:"userId" binding:"required"`
		CredentialID      string `json:"credentialId" binding:"required"`
		ClientDataJSON    string `json:"clientDataJSON"`
		AuthenticatorData string `json:"authenticatorData"`
		Signature         string `json:"signature"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid := s.VerifyWebAuthn(req.UserID, req.CredentialID, req.ClientDataJSON, req.AuthenticatorData, req.Signature)

	c.JSON(http.StatusOK, gin.H{"valid": valid})
}

func (s *TwoFactorService) handleGetWebAuthnCredentials(c *gin.Context) {
	userID := c.Param("userId")
	credentials := s.GetWebAuthnCredentials(userID)

	// Don't expose public keys
	sanitized := make([]gin.H, len(credentials))
	for i, cred := range credentials {
		sanitized[i] = gin.H{
			"credentialId": cred.CredentialID,
			"deviceName":   cred.DeviceName,
			"transport":    cred.Transport,
			"createdAt":    cred.CreatedAt,
			"lastUsed":     cred.LastUsed,
		}
	}

	c.JSON(http.StatusOK, gin.H{"credentials": sanitized})
}

func (s *TwoFactorService) handleDeleteWebAuthnCredential(c *gin.Context) {
	userID := c.Param("userId")
	credentialID := c.Param("credentialId")

	if err := s.DeleteWebAuthnCredential(userID, credentialID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credential deleted"})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()
	service := NewTwoFactorService(config)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	service.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}

	go func() {
		log.Printf("2FA service starting on port %s", config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
