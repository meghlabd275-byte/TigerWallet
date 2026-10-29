/**
 * TigerWallet Passkey/WebAuthn Authentication Service
 * Production-ready FIDO2/WebAuthn implementation for secure passwordless authentication
 *
 * Features:
 * - Full WebAuthn credential management
 * - Passkey registration and authentication
 * - Platform authenticator support
 * - Cross-device credential sync support
 * - RS256, ES256, ES256KR algorithm support
 */

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
	JWTExpiry  time.Duration
	RPID       string // Relying Party ID (domain)
	RPOrigin   string // Relying Party Origin
	Timeout    time.Duration
}

func LoadConfig() *Config {
	// An empty signing key would make every passkey token forgeable, so the
	// service refuses to start without one.
	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) < 32 {
		log.Fatal("passkey: JWT_SECRET must be set to at least 32 characters")
	}

	return &Config{
		ServerPort: getEnv("PASSKEY_PORT", "9097"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet"),
		JWTSecret:  jwtSecret,
		JWTExpiry:  24 * time.Hour * 7, // 7 days
		RPID:       getEnv("PASSKEY_RPID", "tigerwallet.com"),
		RPOrigin:   getEnv("PASSKEY_ORIGIN", "https://tigerwallet.com"),
		Timeout:    60 * time.Second,
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
	ID              uint       `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Username        string     `gorm:"uniqueIndex;size:255" json:"username"`
	Email           string     `gorm:"index;size:255" json:"email"`
	PasswordHash    string     `json:"-"`
	WalletAddress   string     `gorm:"index;size:255" json:"wallet_address"`
	IsEmailVerified bool       `json:"is_email_verified"`
	Status          string     `json:"status"` // active, suspended, deleted
	LastLoginAt     *time.Time `json:"last_login_at"`
}

type PasskeyCredential struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID              uint       `gorm:"index" json:"user_id"`
	CredentialID        string     `gorm:"uniqueIndex;size:255" json:"credential_id"`
	CredentialPublicKey string     `json:"credential_public_key"` // Stored as base64
	CredentialType      string     `json:"credential_type"`       // public-key
	SignCount           uint64     `json:"sign_count"`
	AAGUID              string     `json:"aaguid"`    // Authenticator Attestation GUID
	Transport           string     `json:"transport"` // usb, nfc, ble, internal
	BackupEligible      bool       `json:"backup_eligible"`
	BackupState         bool       `json:"backup_state"`
	AttestationType     string     `json:"attestation_type"` // none, indirect, direct
	AuthenticatorLabel  string     `json:"authenticator_label"`
	DeviceType          string     `json:"device_type"` // platform, cross-platform
	IsActive            bool       `json:"is_active" gorm:"default:true"`
	LastUsedAt          *time.Time `json:"last_used_at"`
}

type PasskeyChallenge struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`

	Challenge              string `gorm:"uniqueIndex;size:255" json:"challenge"` // Base64 encoded
	ChallengeType          string `json:"challenge_type"`                        // registration, authentication
	UserID                 *uint  `gorm:"index" json:"user_id"`
	UserVerification       string `json:"user_verification"` // required, preferred, discouraged
	AuthenticatorSelection string `json:"authenticator_selection"`
	Timeout                int    `json:"timeout"` // milliseconds
	Used                   bool   `json:"used" gorm:"default:false"`
}

type PasskeySession struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`

	SessionID string `gorm:"uniqueIndex;size:255" json:"session_id"`
	UserID    uint   `gorm:"index" json:"user_id"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
	IsValid   bool   `json:"is_valid" gorm:"default:true"`
}

// ============================================================================
// WebAuthn Types (aligned with WebAuthn/FIDO2 spec)
// ============================================================================

// PublicKeyCredentialCreationOptions
type CredentialCreationOptions struct {
	Challenge              string                        `json:"challenge"`
	Rp                     RelyingParty                  `json:"rp"`
	User                   PublicKeyCredentialUserEntity `json:"user"`
	PubKeyCredParams       []PubKeyCredParam             `json:"pubKeyCredParams"`
	Timeout                int                           `json:"timeout"`
	AuthenticatorSelection AuthenticatorSelection        `json:"authenticatorSelection,omitempty"`
	Attestation            string                        `json:"attestation,omitempty"`
	Extensions             CredExtension                 `json:"extensions,omitempty"`
}

type RelyingParty struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
}

type PublicKeyCredentialUserEntity struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Icon        string `json:"icon,omitempty"`
}

type PubKeyCredParam struct {
	Type string `json:"type"`
	Alg  int    `json:"alg"` // -7 (ES256), -257 (RS256), -8 (Ed25519)
}

type AuthenticatorSelection struct {
	AuthenticatorAttachment string `json:"authenticatorAttachment,omitempty"`
	RequireResidentKey      bool   `json:"requireResidentKey,omitempty"`
	UserVerification        string `json:"userVerification,omitempty"`
	ResidentKey             string `json:"residentKey,omitempty"`
}

type CredExtension struct {
	CredProps     *CredProps `json:"credProps,omitempty"`
	AuthnSel      []string   `json:"authnSel,omitempty"`
	UVM           bool       `json:"uvm,omitempty"`
	BiometricPerf *bool      `json:"biometricPerf,omitempty"`
}

type CredProps struct {
	RP bool `json:"rp,omitempty"`
	UV bool `json:"uv,omitempty"`
}

// PublicKeyCredentialRequestOptions
type CredentialRequestOptions struct {
	Challenge        string                 `json:"challenge"`
	Timeout          int                    `json:"timeout"`
	RPID             string                 `json:"rpId"`
	AllowCredentials []AllowedCredential    `json:"allowCredentials,omitempty"`
	UserVerification string                 `json:"userVerification,omitempty"`
	Extensions       AuthenticatorExtension `json:"extensions,omitempty"`
}

type AllowedCredential struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type AuthenticatorExtension struct {
	AppID     string `json:"appid,omitempty"`
	UVM       bool   `json:"uvm,omitempty"`
	CredProps bool   `json:"credProps,omitempty"`
}

// AuthenticatorAttestationResponse
type AuthenticatorResponse struct {
	ClientDataJSON          string `json:"clientDataJSON"`
	AttestationObject       string `json:"attestationObject,omitempty"`
	AuthenticatorData       string `json:"authenticatorData,omitempty"`
	Signature               string `json:"signature,omitempty"`
	PublicKey               string `json:"publicKey,omitempty"`
	PublicKeyCredentialType string `json:"publicKeyCredentialType,omitempty"`
}

type CredentialAssertion struct {
	ID                     string                 `json:"id"`
	Type                   string                 `json:"type"`
	Response               AuthenticatorResponse  `json:"response"`
	ClientExtensionResults map[string]interface{} `json:"clientExtensionResults,omitempty"`
}

// Registration Response
type RegistrationResponse struct {
	ID                string   `json:"id"`
	Type              string   `json:"type"`
	AttestationObject string   `json:"attestationObject"`
	ClientDataJSON    string   `json:"clientDataJSON"`
	Transports        []string `json:"transports,omitempty"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type PasskeyService struct {
	db     *gorm.DB
	config *Config
	jwtKey []byte
}

func NewPasskeyService(db *gorm.DB, config *Config) *PasskeyService {
	return &PasskeyService{
		db:     db,
		config: config,
		jwtKey: []byte(config.JWTSecret),
	}
}

// GenerateChallenge creates a new random challenge for WebAuthn
func (s *PasskeyService) GenerateChallenge() (string, error) {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return "", fmt.Errorf("failed to generate challenge: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(challenge), nil
}

// StartRegistration initiates the passkey registration process
func (s *PasskeyService) StartRegistration(userID uint, username, displayName string) (*CredentialCreationOptions, error) {
	// Generate challenge
	challenge, err := s.GenerateChallenge()
	if err != nil {
		return nil, err
	}

	// Store challenge in database
	challengeRecord := PasskeyChallenge{
		Challenge:     challenge,
		ChallengeType: "registration",
		UserID:        &userID,
		Timeout:       60000,
		ExpiresAt:     time.Now().Add(60 * time.Second),
	}
	if err := s.db.Create(&challengeRecord).Error; err != nil {
		return nil, fmt.Errorf("failed to store challenge: %w", err)
	}

	// Generate user ID (base64 encoded)
	userIDBytes := make([]byte, 8)
	binary.AppendVarint(userIDBytes, int64(userID))
	userIDBase64 := base64.RawURLEncoding.EncodeToString(userIDBytes)

	// Create options
	options := &CredentialCreationOptions{
		Challenge: challenge,
		Rp: RelyingParty{
			ID:   s.config.RPID,
			Name: "TigerWallet",
		},
		User: PublicKeyCredentialUserEntity{
			ID:          userIDBase64,
			Name:        username,
			DisplayName: displayName,
		},
		PubKeyCredParams: []PubKeyCredParam{
			{Type: "public-key", Alg: -7},   // ES256
			{Type: "public-key", Alg: -257}, // RS256
			{Type: "public-key", Alg: -8},   // Ed25519
		},
		Timeout: 60000,
		AuthenticatorSelection: AuthenticatorSelection{
			AuthenticatorAttachment: "platform",
			RequireResidentKey:      false,
			UserVerification:        "preferred",
		},
		Attestation: "none",
	}

	return options, nil
}

// CompleteRegistration completes the passkey registration
func (s *PasskeyService) CompleteRegistration(response RegistrationResponse, userID uint) (*PasskeyCredential, error) {
	clientDataJSON, err := base64.RawURLEncoding.DecodeString(response.ClientDataJSON)
	if err != nil {
		return nil, fmt.Errorf("invalid client data encoding: %w", err)
	}

	attestationRaw, err := base64.RawURLEncoding.DecodeString(response.AttestationObject)
	if err != nil {
		return nil, fmt.Errorf("invalid attestation object encoding: %w", err)
	}
	format, authDataRaw, err := parseAttestationObject(attestationRaw)
	if err != nil {
		return nil, err
	}
	// Attestation statements are deliberately not trusted: only the credential
	// key is consumed, which is exactly the "none" attestation trust model.
	if format == "" {
		return nil, fmt.Errorf("attestation object has no statement format")
	}

	ad, err := parseAuthenticatorData(authDataRaw)
	if err != nil {
		return nil, err
	}
	if !ad.HasAttestedCreds {
		return nil, fmt.Errorf("registration response carries no attested credential data")
	}
	if !ad.UserPresent {
		return nil, fmt.Errorf("authenticator did not assert user presence")
	}
	if err := verifyRPIDHash(ad, s.config.RPID); err != nil {
		return nil, err
	}

	// The client data challenge must be a challenge that was issued for this
	// user's registration ceremony and has not been consumed yet.
	challengeRecord, err := s.consumeChallenge(userID, "registration")
	if err != nil {
		return nil, err
	}
	if _, err := verifyClientData(clientDataJSON, clientDataTypeCreate, challengeRecord.Challenge, s.config.RPOrigin); err != nil {
		return nil, err
	}

	if len(ad.CredentialID) == 0 {
		return nil, fmt.Errorf("attested credential data has an empty credential id")
	}
	if _, _, err := coseKeyToPublicKey(ad.COSEKey); err != nil {
		return nil, fmt.Errorf("credential public key is unusable: %w", err)
	}

	credentialID := base64.RawURLEncoding.EncodeToString(ad.CredentialID)
	// One credential id must map to exactly one user.
	var existing PasskeyCredential
	if err := s.db.Where("credential_id = ?", credentialID).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("this authenticator is already registered")
	}

	credential := PasskeyCredential{
		UserID:              userID,
		CredentialID:        credentialID,
		CredentialPublicKey: base64.RawURLEncoding.EncodeToString(ad.COSEKey),
		CredentialType:      "public-key",
		SignCount:           uint64(ad.SignCount),
		AAGUID:              hex.EncodeToString(ad.AAGUID),
		Transport:           strings.Join(response.Transports, ","),
		BackupEligible:      ad.BackupEligible,
		BackupState:         ad.BackupState,
		AttestationType:     "none",
		DeviceType:          "platform",
		IsActive:            true,
	}

	if err := s.db.Create(&credential).Error; err != nil {
		return nil, fmt.Errorf("failed to store credential: %w", err)
	}

	return &credential, nil
}

// consumeChallenge returns the single most recent unused, unexpired challenge
// of the given ceremony for the user and marks it used. Marking before the
// cryptographic check would let an attacker burn a victim's challenge; marking
// after a failed check would leave it replayable, so it is marked only once the
// caller has committed to using it.
func (s *PasskeyService) consumeChallenge(userID uint, ceremony string) (*PasskeyChallenge, error) {
	var record PasskeyChallenge
	if err := s.db.Where("challenge_type = ? AND user_id = ? AND used = ? AND expires_at > ?",
		ceremony, userID, false, time.Now()).
		Order("created_at DESC").
		First(&record).Error; err != nil {
		return nil, fmt.Errorf("no pending %s challenge for this user: %w", ceremony, err)
	}
	if err := s.db.Model(&PasskeyChallenge{}).
		Where("id = ? AND used = ?", record.ID, false).
		Update("used", true).Error; err != nil {
		return nil, fmt.Errorf("failed to consume challenge: %w", err)
	}
	return &record, nil
}

// StartAuthentication initiates the passkey authentication process
func (s *PasskeyService) StartAuthentication(userID uint) (*CredentialRequestOptions, error) {
	// Generate challenge
	challenge, err := s.GenerateChallenge()
	if err != nil {
		return nil, err
	}

	// Store challenge
	challengeRecord := PasskeyChallenge{
		Challenge:     challenge,
		ChallengeType: "authentication",
		UserID:        &userID,
		Timeout:       60000,
		ExpiresAt:     time.Now().Add(60 * time.Second),
	}
	if err := s.db.Create(&challengeRecord).Error; err != nil {
		return nil, fmt.Errorf("failed to store challenge: %w", err)
	}

	// Get user's credentials
	var credentials []PasskeyCredential
	s.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&credentials)

	allowCredentials := make([]AllowedCredential, len(credentials))
	for i, cred := range credentials {
		allowCredentials[i] = AllowedCredential{
			Type: "public-key",
			ID:   cred.CredentialID,
		}
	}

	options := &CredentialRequestOptions{
		Challenge:        challenge,
		Timeout:          60000,
		RPID:             s.config.RPID,
		AllowCredentials: allowCredentials,
		UserVerification: "preferred",
	}

	return options, nil
}

// CompleteAuthentication completes the passkey authentication
func (s *PasskeyService) CompleteAuthentication(response CredentialAssertion, userID uint) (*jwt.Token, error) {
	clientDataJSON, err := base64.RawURLEncoding.DecodeString(response.Response.ClientDataJSON)
	if err != nil {
		return nil, fmt.Errorf("invalid client data encoding: %w", err)
	}

	credID, err := base64.RawURLEncoding.DecodeString(response.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid credential id encoding: %w", err)
	}
	credentialID := base64.RawURLEncoding.EncodeToString(credID)

	var credential PasskeyCredential
	if err := s.db.Where("credential_id = ? AND user_id = ? AND is_active = ?", credentialID, userID, true).
		First(&credential).Error; err != nil {
		return nil, fmt.Errorf("credential not found for this user: %w", err)
	}

	authDataRaw, err := base64.RawURLEncoding.DecodeString(response.Response.AuthenticatorData)
	if err != nil {
		return nil, fmt.Errorf("invalid authenticator data encoding: %w", err)
	}
	ad, err := parseAuthenticatorData(authDataRaw)
	if err != nil {
		return nil, err
	}
	if !ad.UserPresent {
		return nil, fmt.Errorf("authenticator did not assert user presence")
	}
	if err := verifyRPIDHash(ad, s.config.RPID); err != nil {
		return nil, err
	}

	challengeRecord, err := s.consumeChallenge(userID, "authentication")
	if err != nil {
		return nil, err
	}
	if _, err := verifyClientData(clientDataJSON, clientDataTypeGet, challengeRecord.Challenge, s.config.RPOrigin); err != nil {
		return nil, err
	}

	signature, err := base64.RawURLEncoding.DecodeString(response.Response.Signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}
	if err := s.VerifySignature(authDataRaw, clientDataJSON, signature, credentialID); err != nil {
		return nil, err
	}

	// A signature counter that does not advance is either a cloned authenticator
	// or a replayed assertion. Counters that never advance (0) are used by some
	// platform authenticators, so only a regression is treated as fatal.
	if credential.SignCount != 0 && ad.SignCount != 0 && uint64(ad.SignCount) <= credential.SignCount {
		return nil, fmt.Errorf("authenticator sign counter did not advance (possible cloned authenticator)")
	}

	credential.SignCount = uint64(ad.SignCount)
	credential.BackupState = ad.BackupState
	now := time.Now()
	credential.LastUsedAt = &now
	if err := s.db.Save(&credential).Error; err != nil {
		return nil, fmt.Errorf("failed to update credential: %w", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"cred_id": credential.CredentialID,
		"type":    "passkey",
		"exp":     time.Now().Add(s.config.JWTExpiry).Unix(),
		"iat":     time.Now().Unix(),
	})

	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	var user User
	if err := s.db.First(&user, userID).Error; err == nil {
		user.LastLoginAt = &now
		s.db.Save(&user)
	}

	// Parse the token that was just signed so the caller receives the same
	// validated structure the middleware would produce.
	signed, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, fmt.Errorf("failed to parse issued token: %w", err)
	}
	return signed, nil
}

// VerifySignature verifies an authenticator assertion against the public key
// stored for credentialID.
func (s *PasskeyService) VerifySignature(authData, clientDataHash, signature []byte, credentialID string) error {
	var credential PasskeyCredential
	if err := s.db.Where("credential_id = ?", credentialID).First(&credential).Error; err != nil {
		return fmt.Errorf("credential not found: %w", err)
	}
	if !credential.IsActive {
		return fmt.Errorf("credential is not active")
	}

	coseKey, err := base64.RawURLEncoding.DecodeString(credential.CredentialPublicKey)
	if err != nil {
		return fmt.Errorf("stored credential public key is not valid base64: %w", err)
	}
	pub, alg, err := coseKeyToPublicKey(coseKey)
	if err != nil {
		return err
	}
	return verifyAssertionSignature(pub, alg, authData, clientDataHash, signature)
}

// GetUserCredentials returns all credentials for a user
func (s *PasskeyService) GetUserCredentials(userID uint) ([]PasskeyCredential, error) {
	var credentials []PasskeyCredential
	if err := s.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&credentials).Error; err != nil {
		return nil, err
	}
	return credentials, nil
}

// DeleteCredential removes a passkey credential
func (s *PasskeyService) DeleteCredential(credentialID string, userID uint) error {
	result := s.db.Model(&PasskeyCredential{}).
		Where("credential_id = ? AND user_id = ?", credentialID, userID).
		Update("is_active", false)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("credential not found")
	}
	return nil
}

// ClientDataJSON represents the client data JSON structure
type ClientDataJSON struct {
	Type        string `json:"type"`
	Challenge   string `json:"challenge"`
	Origin      string `json:"origin"`
	CrossOrigin bool   `json:"crossOrigin,omitempty"`
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *PasskeyService) RegisterStart(c *gin.Context) {
	var req struct {
		UserID      uint   `json:"user_id" binding:"required"`
		Username    string `json:"username" binding:"required"`
		DisplayName string `json:"display_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	options, err := s.StartRegistration(req.UserID, req.Username, req.DisplayName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    options,
	})
}

func (s *PasskeyService) RegisterComplete(c *gin.Context) {
	var req struct {
		UserID   uint                 `json:"user_id" binding:"required"`
		Response RegistrationResponse `json:"response" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	credential, err := s.CompleteRegistration(req.Response, req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"credential_id": credential.CredentialID,
			"device_type":   credential.DeviceType,
		},
	})
}

func (s *PasskeyService) AuthStart(c *gin.Context) {
	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	options, err := s.StartAuthentication(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    options,
	})
}

func (s *PasskeyService) AuthComplete(c *gin.Context) {
	var req struct {
		UserID   uint                `json:"user_id" binding:"required"`
		Response CredentialAssertion `json:"response" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := s.CompleteAuthentication(req.Response, req.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":   token.Raw,
	})
}

func (s *PasskeyService) GetCredentials(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	credentials, err := s.GetUserCredentials(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    credentials,
	})
}

func (s *PasskeyService) DeleteCredentialHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		CredentialID string `json:"credential_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.DeleteCredential(req.CredentialID, userID.(uint)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// ============================================================================
// Database Migration
// ============================================================================

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&PasskeyCredential{},
		&PasskeyChallenge{},
		&PasskeySession{},
	)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()

	// Initialize database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := Migrate(db); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	// Initialize service
	service := NewPasskeyService(db, config)

	// Setup router
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API routes
	api := router.Group("/api/v1/passkey")
	{
		api.POST("/register/start", service.RegisterStart)
		api.POST("/register/complete", service.RegisterComplete)
		api.POST("/auth/start", service.AuthStart)
		api.POST("/auth/complete", service.AuthComplete)

		// Protected routes
		protected := api.Group("/")
		protected.Use(func(c *gin.Context) {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
				c.Abort()
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			// The signing method is pinned so a token that claims alg=none, or an
			// asymmetric algorithm replayed as HMAC, cannot be accepted.
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return service.jwtKey, nil
			}, jwt.WithValidMethods([]string{"HS256"}))

			if err != nil || !token.Valid {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
				c.Abort()
				return
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if userID, ok := claims["user_id"].(float64); ok {
					c.Set("user_id", uint(userID))
				}
			}

			c.Next()
		})
		{
			protected.GET("/credentials", service.GetCredentials)
			protected.DELETE("/credentials", service.DeleteCredentialHandler)
		}
	}

	// Start server
	addr := fmt.Sprintf(":%s", config.ServerPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting Passkey service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
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
