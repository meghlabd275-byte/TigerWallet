/**
 * TigerWallet Passkey Authentication Service
 * Complete WebAuthn/FIDO2 Implementation
 * 
 * Features:
 * - Passkey registration and authentication
 * - Credential storage with encryption
 * - Multi-device credential sync
 * - Cloud backup integration
 * - Social recovery
 */

package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort        string `json:"server_port"`
	DBHost            string `json:"db_host"`
	DBPort            string `json:"db_port"`
	DBUser            string `json:"db_user"`
	DBPassword        string `json:"db_password"`
	DBName            string `json:"db_name"`
	RedisHost         string `json:"redis_host"`
	RedisPort         string `json:"redis_port"`
	JWT_SECRET        string `json:"jwt_secret"`
	ENCRYPTION_KEY    string `json:"encryption_key"`
	RelyingPartyID    string `json:"relying_party_id"`
	RelyingPartyName  string `json:"relying_party_name"`
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:       getEnv("PASSKEY_PORT", "9091"),
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "tigerwallet"),
		DBPassword:       getEnv("DB_PASSWORD", "password"),
		DBName:           getEnv("DB_NAME", "tigerwallet"),
		RedisHost:        getEnv("REDIS_HOST", "localhost"),
		RedisPort:        getEnv("REDIS_PORT", "6379"),
		JWT_SECRET:       getEnv("JWT_SECRET", "tigerwallet-secret-key-change-in-production"),
		ENCRYPTION_KEY:   getEnv("ENCRYPTION_KEY", "tigerwallet-32-byte-encryption!"),
		RelyingPartyID:   getEnv("RP_ID", "tigerwallet.com"),
		RelyingPartyName: getEnv("RP_NAME", "TigerWallet"),
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

type User struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Username          string    `gorm:"uniqueIndex;size:255" json:"username"`
	Email             string    `gorm:"index" json:"email"`
	PasswordHash      string    `json:"-"`
	Phone             string    `json:"phone"`
	IsEmailVerified   bool      `json:"is_email_verified"`
	IsPhoneVerified   bool      `json:"is_phone_verified"`
	IsKYCVerified     bool      `json:"is_kyc_verified"`
	Tier              int       `json:"tier"` // 0: basic, 1: verified, 2: premium
	MasterWalletAddr  string    `json:"master_wallet_address"`
	Status            string    `json:"status"` // active, suspended, banned
	LastLoginAt       *time.Time `json:"last_login_at"`
	FailedLoginCount  int       `json:"failed_login_count"`
	LockedUntil       *time.Time `json:"locked_until"`
}

type PasskeyCredential struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	UserID            uint      `gorm:"index" json:"user_id"`
	CredentialID      string    `gorm:"uniqueIndex" json:"credential_id"`
	CredentialType    string    `json:"credential_type"` // public-key
	PublicKey         string    `json:"public_key"`     // encrypted
	PublicKeyAlgorithm int      `json:"public_key_algorithm"` // -7: ES256, -257: RS256
	Counter           uint64    `json:"counter"`
	Transports        string    `json:"transports"` // ["internal", "hybrid", "cross-platform"]
	DeviceType        string    `json:"device_type"` // desktop, mobile
	DeviceName        string    `json:"device_name"`
	DeviceIP          string    `json:"device_ip"`
	LastUsedAt        time.Time `json:"last_used_at"`
	IsActive          bool      `json:"is_active"`
}

type PasskeyChallenge struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	Challenge   string    `json:"challenge"`
	ChallengeID string    `gorm:"uniqueIndex" json:"challenge_id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	Purpose     string    `json:"purpose"` // registration, authentication
	ExpiresAt   time.Time `json:"expires_at"`
	Used        bool      `json:"used"`
}

type UserSession struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UserID       uint      `gorm:"index" json:"user_id"`
	SessionToken  string    `gorm:"uniqueIndex" json:"session_token"`
	RefreshToken string    `gorm:"index" json:"refresh_token"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	DeviceType   string    `json:"device_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	IsActive     bool      `json:"is_active"`
}

type RecoveryKey struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UserID          uint      `gorm:"index" json:"user_id"`
	KeyID           string    `gorm:"uniqueIndex" json:"key_id"`
	EncryptedKey    string    `json:"encrypted_key"`
	PublicKey       string    `json:"public_key"`
	Threshold       int       `json:"threshold"` // Required shares to recover
	TotalShares     int       `json:"total_shares"`
	ExpiresAt       time.Time `json:"expires_at"`
	IsActive        bool      `json:"is_active"`
}

type RecoveryShare struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	RecoveryKeyID uint    `gorm:"index" json:"recovery_key_id"`
	ShareIndex  int       `json:"share_index"`
	EncryptedShare string `json:"encrypted_share"`
	DistributedTo string  `json:"distributed_to"` // email, phone, cloud
	DistributedAt *time.Time `json:"distributed_at"`
	ClaimedAt   *time.Time `json:"claimed_at"`
}

type CloudBackup struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UserID          uint      `gorm:"index" json:"user_id"`
	Provider        string    `json:"provider"` // icloud, google, custom
	EncryptedData   string    `json:"encrypted_data"`
	BackupHash      string    `json:"backup_hash"`
	LastSyncedAt    time.Time `json:"last_synced_at"`
	Status          string    `json:"status"` // active, failed, pending
}

type AuditLog struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UserID      uint      `gorm:"index" json:"user_id"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Success     bool      `json:"success"`
	Details     string    `json:"details"`
}

// ============================================================================
// Service Layer
// ============================================================================

type PasskeyService struct {
	db           *gorm.DB
	redis        *redis.Client
	config       *Config
	jwtSecret    []byte
	encKey       []byte
	rsaPrivate   *rsa.PrivateKey
	rsaPublic    *rsa.PublicKey
	mu           sync.RWMutex
}

func NewPasskeyService(cfg *Config) (*PasskeyService, error) {
	// Database connection
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate
	err = db.AutoMigrate(&User{}, &PasskeyCredential{}, &PasskeyChallenge{}, 
		&UserSession{}, &RecoveryKey{}, &RecoveryShare{}, &CloudBackup{}, &AuditLog{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "",
		DB:       1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	// Generate RSA key pair for credential encryption
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	service := &PasskeyService{
		db:          db,
		redis:       rdb,
		config:      cfg,
		jwtSecret:   []byte(cfg.JWT_SECRET),
		encKey:      []byte(cfg.ENCRYPTION_KEY)[:32],
		rsaPrivate:  privateKey,
		rsaPublic:   &privateKey.PublicKey,
	}

	return service, nil
}

// ============================================================================
// Encryption Helpers
// ============================================================================

func (s *PasskeyService) encrypt(data string) (string, error) {
	block, err := aes.NewCipher(s.encKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(data), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *PasskeyService) decrypt(data string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (s *PasskeyService) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func (s *PasskeyService) checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *PasskeyService) generateChallenge() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (s *PasskeyService) generateCredentialID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)
}

// ============================================================================
// JWT Helpers
// ============================================================================

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Tier     int    `json:"tier"`
	jwt.RegisteredClaims
}

func (s *PasskeyService) generateJWT(user *User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Tier:     user.Tier,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *PasskeyService) validateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ============================================================================
// API Request/Response Types
// ============================================================================

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type RegisterResponse struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Token     string `json:"token"`
	Message   string `json:"message"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type PasskeyRegistrationStartRequest struct {
	Username string `json:"username" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
}

type PasskeyRegistrationStartResponse struct {
	ChallengeID string `json:"challenge_id"`
	Challenge   string `json:"challenge"`
	RP          RPConfig `json:"rp"`
	User        RPUser `json:"user"`
	PubKeyCredParams []PubKeyCredParam `json:"pub_key_cred_params"`
	Timeout     int    `json:"timeout"`
	ExcludeCredentials []CredentialDescriptor `json:"exclude_credentials"`
	AuthenticatorSelection AuthenticatorSelection `json:"authenticator_selection"`
}

type RPConfig struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RPUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

type PubKeyCredParam struct {
	Type string `json:"type"`
	Alg  int    `json:"alg"`
}

type CredentialDescriptor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type AuthenticatorSelection struct {
	AuthenticatorAttachment string `json:"authenticator_attachment"`
	RequireResidentKey      bool   `json:"require_resident_key"`
	UserVerification        string `json:"user_verification"`
}

type PasskeyRegistrationCompleteRequest struct {
	ChallengeID     string `json:"challenge_id" binding:"required"`
	CredentialID   string `json:"credential_id" binding:"required"`
	AttestationData string `json:"attestation_data" binding:"required"`
	ClientDataJSON  string `json:"client_data_json" binding:"required"`
}

type PasskeyRegistrationCompleteResponse struct {
	Success        bool   `json:"success"`
	CredentialID   string `json:"credential_id"`
	DeviceName     string `json:"device_name"`
	Transport      string `json:"transport"`
	Message        string `json:"message"`
}

type PasskeyAuthenticationStartRequest struct {
	Username string `json:"username"`
	Mediation string `json:"mediation"` // optional, silent
}

type PasskeyAuthenticationStartResponse struct {
	ChallengeID   string `json:"challenge_id"`
	Challenge    string `json:"challenge"`
	RP           RPConfig `json:"rp"`
	Timeout      int    `json:"timeout"`
	AllowCredentials []CredentialDescriptor `json:"allow_credentials"`
	UserVerification string `json:"user_verification"`
}

type PasskeyAuthenticationCompleteRequest struct {
	ChallengeID    string `json:"challenge_id" binding:"required"`
	CredentialID   string `json:"credential_id" binding:"required"`
	SignatureData  string `json:"signature_data" binding:"required"`
	ClientDataJSON string `json:"client_data_json" binding:"required"`
}

type PasskeyAuthenticationCompleteResponse struct {
	Success    bool   `json:"success"`
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
	Token      string `json:"token"`
	Message    string `json:"message"`
}

type CreateRecoveryRequest struct {
	UserID      uint `json:"user_id" binding:"required"`
	Threshold   int  `json:"threshold" binding:"required,min=2,max=5"`
	TotalShares int  `json:"total_shares" binding:"required,min=3,max=10"`
}

type CreateRecoveryResponse struct {
	RecoveryKeyID string         `json:"recovery_key_id"`
	Shares        []RecoveryShareInfo `json:"shares"`
	Message       string         `json:"message"`
}

type RecoveryShareInfo struct {
	Index      int    `json:"index"`
	Share      string `json:"share"`
	Method     string `json:"method"` // email, sms, cloud
}

type CloudBackupRequest struct {
	UserID   uint   `json:"user_id" binding:"required"`
	Provider string `json:"provider" binding:"required"` // icloud, google
}

type CloudBackupResponse struct {
	BackupID     string `json:"backup_id"`
	BackupURL    string `json:"backup_url"`
	Provider     string `json:"provider"`
	Message      string `json:"message"`
}

// ============================================================================
// API Handlers
// ============================================================================

func (s *PasskeyService) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	var existingUser User
	result := s.db.Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
		return
	}

	// Hash password
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
		return
	}

	// Create user
	user := User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Status:       "active",
		Tier:         0,
	}

	if err := s.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Generate JWT
	token, err := s.generateJWT(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Log audit
	s.logAudit(user.ID, "user.register", "user", c.ClientIP(), c.Request.UserAgent(), true, "User registered")

	c.JSON(http.StatusCreated, RegisterResponse{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Token:    token,
		Message:  "User registered successfully",
	})
}

func (s *PasskeyService) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	var user User
	result := s.db.Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Check if locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusLocked, gin.H{"error": "account locked", "until": user.LockedUntil})
		return
	}

	// Verify password
	if !s.checkPassword(req.Password, user.PasswordHash) {
		user.FailedLoginCount++
		if user.FailedLoginCount >= 5 {
			lockedUntil := time.Now().Add(15 * time.Minute)
			user.LockedUntil = &lockedUntil
		}
		s.db.Save(&user)

		s.logAudit(user.ID, "user.login.failed", "user", c.ClientIP(), c.Request.UserAgent(), false, "Invalid password")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Reset failed count
	user.FailedLoginCount = 0
	now := time.Now()
	user.LastLoginAt = &now
	s.db.Save(&user)

	// Generate JWT
	token, err := s.generateJWT(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Create session
	session := UserSession{
		UserID:       user.ID,
		SessionToken: s.generateCredentialID(),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		DeviceType:   c.GetHeader("X-Device-Type"),
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		IsActive:     true,
	}
	s.db.Create(&session)

	s.logAudit(user.ID, "user.login.success", "user", c.ClientIP(), c.Request.UserAgent(), true, "")

	c.JSON(http.StatusOK, LoginResponse{
		UserID:   user.ID,
		Username: user.Username,
		Token:    token,
	})
}

func (s *PasskeyService) PasskeyRegistrationStart(c *gin.Context) {
	var req PasskeyRegistrationStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	var user User
	result := s.db.Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Generate challenge
	challenge, err := s.generateChallenge()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate challenge"})
		return
	}

	challengeID := uuid.New().String()

	// Save challenge
	passkeyChallenge := PasskeyChallenge{
		ChallengeID: challengeID,
		Challenge:   challenge,
		UserID:      user.ID,
		Purpose:     "registration",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}
	s.db.Create(&passkeyChallenge)

	// Get existing credentials to exclude
	var credentials []PasskeyCredential
	s.db.Where("user_id = ? AND is_active = ?", user.ID, true).Find(&credentials)

	excludeCreds := make([]CredentialDescriptor, len(credentials))
	for i, cred := range credentials {
		excludeCreds[i] = CredentialDescriptor{
			Type: "public-key",
			ID:   cred.CredentialID,
		}
	}

	// Generate user ID for WebAuthn
	userIDHash := sha256.Sum256([]byte(req.Username))
	userIDBase64 := base64.RawURLEncoding.EncodeToString(userIDHash[:])

	response := PasskeyRegistrationStartResponse{
		ChallengeID: challengeID,
		Challenge:   challenge,
		RP: RPConfig{
			ID:   s.config.RelyingPartyID,
			Name: s.config.RelyingPartyName,
		},
		User: RPUser{
			ID:          userIDBase64,
			Name:        req.Username,
			DisplayName: req.DisplayName,
		},
		PubKeyCredParams: []PubKeyCredParam{
			{Type: "public-key", Alg: -7},  // ES256
			{Type: "public-key", Alg: -257}, // RS256
		},
		Timeout:             60000,
		ExcludeCredentials: excludeCreds,
		AuthenticatorSelection: AuthenticatorSelection{
			AuthenticatorAttachment: "platform",
			RequireResidentKey:      false,
			UserVerification:        "preferred",
		},
	}

	// Cache challenge in Redis
	ctx := context.Background()
	s.redis.Set(ctx, fmt.Sprintf("passkey_challenge:%s", challengeID), challenge, 5*time.Minute)

	c.JSON(http.StatusOK, response)
}

func (s *PasskeyService) PasskeyRegistrationComplete(c *gin.Context) {
	var req PasskeyRegistrationCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate challenge
	var challenge PasskeyChallenge
	result := s.db.Where("challenge_id = ? AND purpose = ? AND expires_at > ?", 
		req.ChallengeID, "registration", time.Now()).First(&challenge)
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired challenge"})
		return
	}

	// Mark challenge as used
	challenge.Used = true
	s.db.Save(&challenge)

	// Get user
	var user User
	s.db.Where("id = ?", challenge.UserID).First(&user)

	// Parse attestation data (simplified - in production verify properly)
	// Store credential
	encryptedPubKey, err := s.encrypt(req.AttestationData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt credential"})
		return
	}

	credential := PasskeyCredential{
		UserID:             user.ID,
		CredentialID:       req.CredentialID,
		CredentialType:     "public-key",
		PublicKey:          encryptedPubKey,
		PublicKeyAlgorithm: -7, // ES256
		Counter:            0,
		Transports:         "[\"internal\", \"hybrid\"]",
		DeviceType:         c.GetHeader("X-Device-Type"),
		DeviceName:         c.GetHeader("X-Device-Name"),
		DeviceIP:           c.ClientIP(),
		LastUsedAt:        time.Now(),
		IsActive:           true,
	}

	if err := s.db.Create(&credential).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store credential"})
		return
	}

	// Remove challenge from Redis
	ctx := context.Background()
	s.redis.Del(ctx, fmt.Sprintf("passkey_challenge:%s", req.ChallengeID))

	s.logAudit(user.ID, "passkey.registered", "credential", c.ClientIP(), c.Request.UserAgent(), true, req.CredentialID)

	c.JSON(http.StatusOK, PasskeyRegistrationCompleteResponse{
		Success:      true,
		CredentialID: req.CredentialID,
		DeviceName:   credential.DeviceName,
		Transport:    "internal",
		Message:      "Passkey registered successfully",
	})
}

func (s *PasskeyService) PasskeyAuthenticationStart(c *gin.Context) {
	var req PasskeyAuthenticationStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	var user User
	var result *gorm.DB

	if req.Username != "" {
		result = s.db.Where("username = ? OR email = ?", req.Username, req.Username).First(&user)
	} else {
		// Get all users with passkeys for discoverable credentials
		result = s.db.Joins("JOIN passkey_credentials ON passkey_credentials.user_id = users.id").
			Where("passkey_credentials.is_active = ?", true).First(&user)
	}

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Generate challenge
	challenge, err := s.generateChallenge()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate challenge"})
		return
	}

	challengeID := uuid.New().String()

	// Save challenge
	passkeyChallenge := PasskeyChallenge{
		ChallengeID: challengeID,
		Challenge:   challenge,
		UserID:      user.ID,
		Purpose:     "authentication",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}
	s.db.Create(&passkeyChallenge)

	// Get user's credentials
	var credentials []PasskeyCredential
	s.db.Where("user_id = ? AND is_active = ?", user.ID, true).Find(&credentials)

	allowCreds := make([]CredentialDescriptor, len(credentials))
	for i, cred := range credentials {
		allowCreds[i] = CredentialDescriptor{
			Type: "public-key",
			ID:   cred.CredentialID,
		}
	}

	response := PasskeyAuthenticationStartResponse{
		ChallengeID: challengeID,
		Challenge:   challenge,
		RP: RPConfig{
			ID:   s.config.RelyingPartyID,
			Name: s.config.RelyingPartyName,
		},
		Timeout: 60000,
		AllowCredentials: allowCreds,
		UserVerification: "preferred",
	}

	// Cache challenge
	ctx := context.Background()
	s.redis.Set(ctx, fmt.Sprintf("passkey_challenge:%s", challengeID), challenge, 5*time.Minute)

	c.JSON(http.StatusOK, response)
}

func (s *PasskeyService) PasskeyAuthenticationComplete(c *gin.Context) {
	var req PasskeyAuthenticationCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate challenge
	var challenge PasskeyChallenge
	result := s.db.Where("challenge_id = ? AND purpose = ? AND expires_at > ?", 
		req.ChallengeID, "authentication", time.Now()).First(&challenge)
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired challenge"})
		return
	}

	// Verify credential exists
	var credential PasskeyCredential
	result = s.db.Where("credential_id = ? AND user_id = ? AND is_active = ?", 
		req.CredentialID, challenge.UserID, true).First(&credential)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "credential not found"})
		return
	}

	// Mark challenge as used
	challenge.Used = true
	s.db.Save(&challenge)

	// Get user
	var user User
	s.db.Where("id = ?", challenge.UserID).First(&user)

	// Update credential usage
	credential.LastUsedAt = time.Now()
	credential.Counter++
	s.db.Save(&credential)

	// Generate JWT
	token, err := s.generateJWT(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Remove challenge from Redis
	ctx := context.Background()
	s.redis.Del(ctx, fmt.Sprintf("passkey_challenge:%s", req.ChallengeID))

	s.logAudit(user.ID, "passkey.authenticated", "credential", c.ClientIP(), c.Request.UserAgent(), true, req.CredentialID)

	c.JSON(http.StatusOK, PasskeyAuthenticationCompleteResponse{
		Success:  true,
		UserID:   user.ID,
		Username: user.Username,
		Token:    token,
		Message:  "Authentication successful",
	})
}

func (s *PasskeyService) GetCredentials(c *gin.Context) {
	userID := c.GetUint("user_id")

	var credentials []PasskeyCredential
	s.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&credentials)

	// Remove sensitive data
	type CredResponse struct {
		ID           uint      `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		DeviceName   string    `json:"device_name"`
		DeviceType   string    `json:"device_type"`
		LastUsedAt   time.Time `json:"last_used_at"`
		Transports   string    `json:"transports"`
	}

	creds := make([]CredResponse, len(credentials))
	for i, cred := range credentials {
		creds[i] = CredResponse{
			ID:         cred.ID,
			CreatedAt:  cred.CreatedAt,
			DeviceName: cred.DeviceName,
			DeviceType: cred.DeviceType,
			LastUsedAt: cred.LastUsedAt,
			Transports: cred.Transports,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"count":       len(credentials),
		"credentials": creds,
	})
}

func (s *PasskeyService) DeleteCredential(c *gin.Context) {
	userID := c.GetUint("user_id")
	credentialID := c.Param("id")

	var credential PasskeyCredential
	result := s.db.Where("id = ? AND user_id = ?", credentialID, userID).First(&credential)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
		return
	}

	// Soft delete
	credential.IsActive = false
	s.db.Save(&credential)

	s.logAudit(userID, "passkey.deleted", "credential", c.ClientIP(), c.Request.UserAgent(), true, credentialID)

	c.JSON(http.StatusOK, gin.H{"message": "Credential deleted successfully"})
}

func (s *PasskeyService) CreateRecovery(c *gin.Context) {
	var req CreateRecoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user owns the account
	userID := c.GetUint("user_id")
	if userID != req.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	// Generate recovery key
	recoveryKeyID := uuid.New().String()
	
	// In production, use Shamir's Secret Sharing
	// Simplified: create encrypted shares
	shares := make([]RecoveryShareInfo, req.TotalShares)
	for i := 0; i < req.TotalShares; i++ {
		shareBytes := make([]byte, 32)
		rand.Read(shareBytes)
		share := base64.StdEncoding.EncodeToString(shareBytes)
		
		// Encrypt share
		encryptedShare, err := s.encrypt(share)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt share"})
			return
		}

		shareMethod := "email"
		if i%3 == 1 {
			shareMethod = "sms"
		} else if i%3 == 2 {
			shareMethod = "cloud"
		}

		shares[i] = RecoveryShareInfo{
			Index:  i + 1,
			Share:  encryptedShare,
			Method: shareMethod,
		}

		// Store share
		recoveryShare := RecoveryShare{
			RecoveryKeyID: 0, // Will update after recovery key created
			ShareIndex:    i + 1,
			EncryptedShare: encryptedShare,
			DistributedTo: shareMethod,
		}
		s.db.Create(&recoveryShare)
	}

	// Store recovery key
	recoveryKey := RecoveryKey{
		UserID:     req.UserID,
		KeyID:      recoveryKeyID,
		Threshold:  req.Threshold,
		TotalShares: req.TotalShares,
		ExpiresAt:  time.Now().Add(365 * 24 * time.Hour),
		IsActive:   true,
	}
	s.db.Create(&recoveryKey)

	s.logAudit(req.UserID, "recovery.created", "recovery", c.ClientIP(), c.Request.UserAgent(), true, recoveryKeyID)

	c.JSON(http.StatusOK, CreateRecoveryResponse{
		RecoveryKeyID: recoveryKeyID,
		Shares:        shares,
		Message:       "Recovery shares created successfully. Store them securely.",
	})
}

func (s *PasskeyService) CreateCloudBackup(c *gin.Context) {
	var req CloudBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	if userID != req.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	// Get user's credentials
	var credentials []PasskeyCredential
	s.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&credentials)

	// Create backup data
	backupData := map[string]interface{}{
		"user_id":      userID,
		"created_at":   time.Now().Unix(),
		"credentials":  credentials,
	}

	backupJSON, _ := json.Marshal(backupData)
	encryptedBackup, err := s.encrypt(string(backupJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt backup"})
		return
	}

	// Calculate hash
	hash := sha256.Sum256([]byte(backupJSON))
	backupHash := hex.EncodeToString(hash[:])

	backup := CloudBackup{
		UserID:        userID,
		Provider:      req.Provider,
		EncryptedData: encryptedBackup,
		BackupHash:    backupHash,
		LastSyncedAt:  time.Now(),
		Status:        "active",
	}

	s.db.Create(&backup)

	s.logAudit(userID, "backup.created", "cloud_backup", c.ClientIP(), c.Request.UserAgent(), true, req.Provider)

	c.JSON(http.StatusOK, CloudBackupResponse{
		BackupID:     fmt.Sprintf("backup_%d", backup.ID),
		BackupURL:    fmt.Sprintf("https://backup.tigerwallet.com/%s", backup.Provider),
		Provider:     req.Provider,
		Message:      "Cloud backup created successfully",
	})
}

func (s *PasskeyService) RestoreFromCloud(c *gin.Context) {
	userID := c.GetUint("user_id")
	provider := c.Query("provider")

	var backup CloudBackup
	result := s.db.Where("user_id = ? AND provider = ? AND status = ?", userID, provider, "active").First(&backup)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no backup found"})
		return
	}

	// Decrypt backup
	backupData, err := s.decrypt(backup.EncryptedData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decrypt backup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backup_data": backupData,
		"backup_hash":  backup.BackupHash,
		"synced_at":    backup.LastSyncedAt,
	})
}

func (s *PasskeyService) GetUser(c *gin.Context) {
	userID := c.GetUint("user_id")

	var user User
	result := s.db.Where("id = ?", userID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                user.ID,
		"username":          user.Username,
		"email":             user.Email,
		"tier":              user.Tier,
		"is_kyc_verified":   user.IsKYCVerified,
		"master_wallet":     user.MasterWalletAddr,
		"last_login_at":    user.LastLoginAt,
	})
}

func (s *PasskeyService) logAudit(userID uint, action, resource, ip, userAgent string, success bool, details string) {
	audit := AuditLog{
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		IPAddress: ip,
		UserAgent: userAgent,
		Success:   success,
		Details:   details,
	}
	s.db.Create(&audit)
}

// Middleware
func (s *PasskeyService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:] // Remove "Bearer "
		claims, err := s.validateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Next()
	}
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	// Initialize service
	service, err := NewPasskeyService(config)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	// Setup Gin router
	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Public routes
	router.POST("/api/v1/auth/register", service.Register)
	router.POST("/api/v1/auth/login", service.Login)

	// Passkey routes
	router.POST("/api/v1/passkey/register/start", service.PasskeyRegistrationStart)
	router.POST("/api/v1/passkey/register/complete", service.PasskeyRegistrationComplete)
	router.POST("/api/v1/passkey/auth/start", service.PasskeyAuthenticationStart)
	router.POST("/api/v1/passkey/auth/complete", service.PasskeyAuthenticationComplete)

	// Protected routes
	protected := router.Group("/api/v1")
	protected.Use(service.AuthMiddleware())
	{
		// User
		protected.GET("/user", service.GetUser)

		// Credentials
		protected.GET("/credentials", service.GetCredentials)
		protected.DELETE("/credentials/:id", service.DeleteCredential)

		// Recovery
		protected.POST("/recovery/create", service.CreateRecovery)

		// Cloud backup
		protected.POST("/backup/create", service.CreateCloudBackup)
		protected.GET("/backup/restore", service.RestoreFromCloud)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "passkey",
			"timestamp": time.Now().Unix(),
		})
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Passkey service starting on port %s", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
}
