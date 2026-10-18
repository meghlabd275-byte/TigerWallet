/**
 * TigerWallet Social Authentication Service
 * Complete Social Login Implementation (Google, Apple, Twitter)
 *
 * Features:
 * - OAuth2/OAuth1a authentication
 * - JWT token generation
 * - User account linking
 * - Multi-provider support
 * - Rate limiting
 * - Security features
 */

package main

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort         string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	RedisHost          string
	RedisPort          string
	JWTSecret          string
	JWTExpiration      time.Duration
	RefreshTokenExpiry time.Duration

	// OAuth2 Google
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// OAuth2 Apple
	AppleClientID    string
	AppleTeamID      string
	AppleKeyID       string
	ApplePrivateKey  string
	AppleRedirectURL string

	// Twitter OAuth1a
	TwitterConsumerKey    string
	TwitterConsumerSecret string
	TwitterRedirectURL    string

	// Frontend URLs
	FrontendURL string
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:         getEnv("SOCIAL_AUTH_PORT", "9094"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "tigerwallet"),
		DBPassword:         getEnv("DB_PASSWORD", "password"),
		DBName:             getEnv("DB_NAME", "tigerwallet"),
		RedisHost:          getEnv("REDIS_HOST", "localhost"),
		RedisPort:          getEnv("REDIS_PORT", "6379"),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		JWTExpiration:      24 * time.Hour,
		RefreshTokenExpiry: 30 * 24 * time.Hour,

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:3000/auth/google/callback"),

		AppleClientID:    getEnv("APPLE_CLIENT_ID", ""),
		AppleTeamID:      getEnv("APPLE_TEAM_ID", ""),
		AppleKeyID:       getEnv("APPLE_KEY_ID", ""),
		ApplePrivateKey:  getEnv("APPLE_PRIVATE_KEY", ""),
		AppleRedirectURL: getEnv("APPLE_REDIRECT_URL", "http://localhost:3000/auth/apple/callback"),

		TwitterConsumerKey:    getEnv("TWITTER_CONSUMER_KEY", ""),
		TwitterConsumerSecret: getEnv("TWITTER_CONSUMER_SECRET", ""),
		TwitterRedirectURL:    getEnv("TWITTER_REDIRECT_URL", "http://localhost:3000/auth/twitter/callback"),

		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
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
	ID               uint            `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        gorm.DeletedAt  `gorm:"index" json:"-"`
	UUID             string          `gorm:"uniqueIndex;size:36" json:"uuid"`
	Email            string          `gorm:"index" json:"email"`
	EmailVerified    bool            `json:"email_verified"`
	Username         string          `gorm:"uniqueIndex" json:"username"`
	PasswordHash     string          `json:"-"`
	ProfilePicture   string          `json:"profile_picture"`
	FirstName        string          `json:"first_name"`
	LastName         string          `json:"last_name"`
	Phone            string          `json:"phone"`
	Status           string          `json:"status"`     // active, suspended, pending
	KYCStatus        string          `json:"kyc_status"` // none, pending, verified, rejected
	TwoFactorEnabled bool            `json:"two_factor_enabled"`
	LoginOTP         string          `json:"-"`
	LoginOTPExpiry   *time.Time      `json:"-"`
	FailedLoginCount int             `json:"failed_login_count"`
	LockedUntil      *time.Time      `json:"locked_until"`
	LastLoginAt      *time.Time      `json:"last_login_at"`
	LastLoginIP      string          `json:"last_login_ip"`
	LastLoginUA      string          `json:"last_login_ua"`
	ReferralCode     string          `gorm:"uniqueIndex" json:"referral_code"`
	ReferredBy       *uint           `json:"referred_by"`
	Referrer         *User           `gorm:"foreignKey:ReferredBy" json:"-"`
	WalletAddresses  []WalletAddress `json:"wallet_addresses"`
}

type WalletAddress struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	UserID      uint      `gorm:"index" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"-"`
	Address     string    `gorm:"uniqueIndex" json:"address"`
	AddressType string    `json:"address_type"` // evm, solana, cosmos, etc.
	ChainID     int       `json:"chain_id"`
	Label       string    `json:"label"`
	IsHidden    bool      `json:"is_hidden"`
}

type SocialAccount struct {
	ID                 uint       `gorm:"primarykey" json:"id"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	UserID             uint       `gorm:"index" json:"user_id"`
	User               User       `gorm:"foreignKey:UserID" json:"-"`
	Provider           string     `gorm:"index" json:"provider"` // google, apple, twitter
	ProviderUserID     string     `gorm:"index" json:"provider_user_id"`
	ProviderEmail      string     `json:"provider_email"`
	ProviderUsername   string     `json:"provider_username"`
	ProviderProfileURL string     `json:"provider_profile_url"`
	AccessToken        string     `json:"-"` // encrypted
	RefreshToken       string     `json:"-"` // encrypted
	TokenExpiry        *time.Time `json:"token_expiry"`
	Metadata           string     `json:"metadata"` // JSON blob
}

type AuthSession struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	SessionID    string         `gorm:"uniqueIndex;size:64" json:"session_id"`
	UserID       uint           `gorm:"index" json:"user_id"`
	User         User           `gorm:"foreignKey:UserID" json:"-"`
	IPAddress    string         `json:"ip_address"`
	UserAgent    string         `json:"user_agent"`
	DeviceID     string         `json:"device_id"`
	DeviceName   string         `json:"device_name"`
	Location     string         `json:"location"`
	ExpiresAt    time.Time      `json:"expires_at"`
	LastActiveAt time.Time      `json:"last_active_at"`
	IsRevoked    bool           `json:"is_revoked"`
	RevokedAt    *time.Time     `json:"revoked_at"`
	RevokeReason string         `json:"revoke_reason"`
}

type RefreshToken struct {
	ID            uint       `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	TokenHash     string     `gorm:"uniqueIndex;size:64" json:"-"`
	UserID        uint       `gorm:"index" json:"user_id"`
	User          User       `gorm:"foreignKey:UserID" json:"-"`
	Provider      string     `json:"provider"` // google, apple, twitter, email
	ExpiresAt     time.Time  `json:"expires_at"`
	RevokedAt     *time.Time `json:"revoked_at"`
	RevokedReason string     `json:"revoked_reason"`
	Metadata      string     `json:"metadata"` // JSON
}

// ============================================================================
// OAuth2 Configurations
// ============================================================================

var (
	googleOAuth2Config  *oauth2.Config
	twitterOAuth2Config *oauth2.Config
)

func initOAuth2(cfg *Config) {
	// Google OAuth2
	googleOAuth2Config = &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile", "openid"},
		Endpoint:     google.Endpoint,
		RedirectURL:  cfg.GoogleRedirectURL,
	}

	// Twitter OAuth2 (using OAuth2 for Twitter v2)
	twitterOAuth2Config = &oauth2.Config{
		ClientID:     cfg.TwitterConsumerKey,
		ClientSecret: cfg.TwitterConsumerSecret,
		Scopes:       []string{"tweet.read", "users.read", "offline.access"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://twitter.com/i/oauth2/authorize",
			TokenURL: "https://api.twitter.com/2/oauth2/token",
		},
		RedirectURL: cfg.TwitterRedirectURL,
	}
}

// ============================================================================
// JWT Token Management
// ============================================================================

type Claims struct {
	jwt.RegisteredClaims
	UserID            uint   `json:"user_id"`
	UUID              string `json:"uuid"`
	Email             string `json:"email"`
	Role              string `json:"role"`
	Provider          string `json:"provider"`
	TwoFactorVerified bool   `json:"two_factor_verified"`
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	UserID   uint   `json:"user_id"`
	Provider string `json:"provider"`
	Type     string `json:"type"` // access, refresh
}

func generateJWT(cfg *Config, user *User, provider string) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.JWTExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet",
			Subject:   user.UUID,
		},
		UserID:   user.ID,
		UUID:     user.UUID,
		Email:    user.Email,
		Role:     "user",
		Provider: provider,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func generateRefreshToken(cfg *Config, userID uint, provider string) (string, error) {
	claims := RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.RefreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet",
			ID:        uuid.New().String(),
		},
		UserID:   userID,
		Provider: provider,
		Type:     "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func validateJWT(cfg *Config, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.JWTSecret), nil
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
// Rate Limiting
// ============================================================================

type RateLimiter struct {
	redis     *redis.Client
	mu        sync.Mutex
	attempts  map[string]int
	lastReset map[string]time.Time
}

func NewRateLimiter(redisHost, redisPort string) *RateLimiter {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: "",
		DB:       0,
	})

	return &RateLimiter{
		redis:     rdb,
		attempts:  make(map[string]int),
		lastReset: make(map[string]time.Time),
	}
}

func (rl *RateLimiter) isRateLimited(key string, maxAttempts int, window time.Duration) bool {
	ctx := context.Background()

	count, err := rl.redis.Incr(ctx, fmt.Sprintf("rate_limit:%s", key)).Result()
	if err != nil {
		rl.mu.Lock()
		defer rl.mu.Unlock()

		rl.attempts[key]++
		lastReset := rl.lastReset[key]

		if time.Since(lastReset) > window {
			rl.attempts[key] = 1
			rl.lastReset[key] = time.Now()
		}

		return rl.attempts[key] > maxAttempts
	}

	rl.redis.Expire(ctx, fmt.Sprintf("rate_limit:%s", key), window)
	return count > int64(maxAttempts)
}

// ============================================================================
// Encryption
// ============================================================================

func encryptData(data, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key[:32]))
	if err != nil {
		return "", err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	encrypted := make([]byte, len(data))
	cfb.XORKeyStream(encrypted, []byte(data))

	result := base64.StdEncoding.EncodeToString(append(iv, encrypted...))
	return result, nil
}

func decryptData(encryptedData, key string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key[:32]))
	if err != nil {
		return "", err
	}

	iv := data[:aes.BlockSize]
	encrypted := data[aes.BlockSize:]

	cfb := cipher.NewCFBDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	cfb.XORKeyStream(decrypted, encrypted)

	return string(decrypted), nil
}

// ============================================================================
// Database Setup
// ============================================================================

func setupDatabase(cfg *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// Auto migrate
	err = db.AutoMigrate(&User{}, &SocialAccount{}, &AuthSession{}, &RefreshToken{}, &WalletAddress{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// ============================================================================
// Google OAuth
// ============================================================================

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

func handleGoogleLogin(cfg *Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate state for CSRF protection
		state := uuid.New().String()
		stateKey := fmt.Sprintf("oauth_state:%s", state)

		// Store state in Redis with 10 minute expiry
		ctx := context.Background()
		rdb := redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		})
		rdb.Set(ctx, stateKey, "valid", 10*time.Minute)

		// Generate PKCE code verifier and challenge
		codeVerifier := generateCodeVerifier()
		codeChallenge := generateCodeChallenge(codeVerifier)

		// Store code verifier
		rdb.Set(ctx, fmt.Sprintf("pkce:%s", state), codeVerifier, 10*time.Minute)

		// Build authorization URL with PKCE
		authURL := googleOAuth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("code_challenge", codeChallenge), oauth2.SetAuthURLParam("code_challenge_method", "S256"))

		c.Redirect(http.StatusTemporaryRedirect, authURL)
	}
}

func handleGoogleCallback(cfg *Config, db *gorm.DB, rateLimiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		state := c.Query("state")
		errorParam := c.Query("error")

		// Check for OAuth errors
		if errorParam != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": errorParam, "error_description": c.Query("error_description")})
			return
		}

		// Validate state
		ctx := context.Background()
		rdb := redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		})

		stateKey := fmt.Sprintf("oauth_state:%s", state)
		_, err := rdb.Get(ctx, stateKey).Result()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_state", "message": "Invalid or expired state parameter"})
			return
		}
		rdb.Del(ctx, stateKey)

		// Get PKCE code verifier
		pkceKey := fmt.Sprintf("pkce:%s", state)
		codeVerifier, err := rdb.Get(ctx, pkceKey).Result()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_pkce", "message": "Invalid PKCE code verifier"})
			return
		}
		rdb.Del(ctx, pkceKey)

		// Rate limiting
		clientIP := c.ClientIP()
		if rateLimiter.isRateLimited("google_callback:"+clientIP, 10, 5*time.Minute) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited", "message": "Too many requests"})
			return
		}

		// Exchange code for token
		token, err := googleOAuth2Config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
		if err != nil {
			log.Printf("Google token exchange error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token_exchange_failed", "message": "Failed to exchange authorization code"})
			return
		}

		// Get user info
		userInfo, err := getGoogleUserInfo(token.AccessToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_info_failed", "message": "Failed to get user information"})
			return
		}

		// Find or create user
		user, err := findOrCreateSocialUser(db, "google", userInfo.ID, userInfo.Email, userInfo.Name, userInfo.Picture)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_creation_failed", "message": "Failed to create user account"})
			return
		}

		// Generate JWT tokens
		accessToken, err := generateJWT(cfg, user, "google")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed", "message": "Failed to generate access token"})
			return
		}

		refreshToken, err := generateRefreshToken(cfg, user.ID, "google")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed", "message": "Failed to generate refresh token"})
			return
		}

		// Create session
		sessionID := uuid.New().String()
		session := AuthSession{
			SessionID:    sessionID,
			UserID:       user.ID,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			DeviceID:     c.GetHeader("X-Device-ID"),
			DeviceName:   c.GetHeader("X-Device-Name"),
			ExpiresAt:    time.Now().Add(cfg.JWTExpiration),
			LastActiveAt: time.Now(),
		}
		db.Create(&session)

		// Update last login
		now := time.Now()
		user.LastLoginAt = &now
		user.LastLoginIP = c.ClientIP()
		user.LastLoginUA = c.Request.UserAgent()
		db.Save(user)

		// Redirect to frontend with tokens
		redirectURL := fmt.Sprintf("%s/auth/callback?access_token=%s&refresh_token=%s&session_id=%s", cfg.FrontendURL, accessToken, refreshToken, sessionID)
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}

func getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}
	if userInfo.ID == "" {
		return nil, errors.New("google userinfo contained no subject id")
	}

	return &userInfo, nil
}

// ============================================================================
// Apple OAuth
// ============================================================================

func handleAppleLogin(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		state := uuid.New().String()

		// Apple uses a different flow - generate state
		authURL := fmt.Sprintf("https://appleid.apple.com/auth/authorize?client_id=%s&redirect_uri=%s&response_mode=form_post&response_type=code&scope=name%%20email&state=%s",
			cfg.AppleClientID, url.QueryEscape(cfg.AppleRedirectURL), state)

		c.Redirect(http.StatusTemporaryRedirect, authURL)
	}
}

func handleAppleCallback(cfg *Config, db *gorm.DB, rateLimiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		idToken := c.PostForm("id_token")
		errorParam := c.PostForm("error")

		if errorParam != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": errorParam, "message": c.PostForm("error_description")})
			return
		}

		// Rate limiting
		clientIP := c.ClientIP()
		if rateLimiter.isRateLimited("apple_callback:"+clientIP, 10, 5*time.Minute) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited", "message": "Too many requests"})
			return
		}

		// Validate and decode Apple ID token
		claims, err := validateAppleIDToken(cfg, idToken)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_token", "message": "Invalid Apple ID token"})
			return
		}

		// Find or create user
		user, err := findOrCreateSocialUser(db, "apple", claims.Subject, claims.Email, claims.Name, "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_creation_failed", "message": "Failed to create user account"})
			return
		}

		// Generate tokens
		accessToken, _ := generateJWT(cfg, user, "apple")
		refreshToken, _ := generateRefreshToken(cfg, user.ID, "apple")

		// Redirect
		redirectURL := fmt.Sprintf("%s/auth/callback?access_token=%s&refresh_token=%s", cfg.FrontendURL, accessToken, refreshToken)
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}

type AppleIDTokenClaims struct {
	Issuer         string `json:"iss"`
	Subject        string `json:"sub"`
	Audience       string `json:"aud"`
	IssuedAt       int64  `json:"iat"`
	ExpirationTime int64  `json:"exp"`
	Email          string `json:"email"`
	EmailVerified  string `json:"email_verified"`
	Name           string `json:"name"`
}

// appleJWKS is the RSA public key set Apple publishes for ID-token signing.
type appleJWKS struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		Use string `json:"use"`
		Alg string `json:"alg"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

var (
	appleJWKSCache   *appleJWKS
	appleJWKSCacheMu sync.RWMutex
	appleJWKSFetched time.Time
)

const appleJWKSURL = "https://appleid.apple.com/auth/keys"

// fetchAppleJWKS retrieves Apple's signing keys, caching them for one hour.
func fetchAppleJWKS() (*appleJWKS, error) {
	appleJWKSCacheMu.RLock()
	if appleJWKSCache != nil && time.Since(appleJWKSFetched) < time.Hour {
		defer appleJWKSCacheMu.RUnlock()
		return appleJWKSCache, nil
	}
	appleJWKSCacheMu.RUnlock()

	resp, err := http.Get(appleJWKSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Apple JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Apple JWKS returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var jwks appleJWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, err
	}
	if len(jwks.Keys) == 0 {
		return nil, errors.New("Apple JWKS contained no keys")
	}

	appleJWKSCacheMu.Lock()
	appleJWKSCache = &jwks
	appleJWKSFetched = time.Now()
	appleJWKSCacheMu.Unlock()

	return &jwks, nil
}

// applePublicKey converts the JWK entry with the given kid into an *rsa.PublicKey.
func applePublicKey(kid string) (*rsa.PublicKey, error) {
	jwks, err := fetchAppleJWKS()
	if err != nil {
		return nil, err
	}

	for _, k := range jwks.Keys {
		if k.Kid != kid {
			continue
		}
		if k.Kty != "RSA" {
			return nil, fmt.Errorf("unexpected Apple key type %q", k.Kty)
		}

		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, fmt.Errorf("invalid Apple key modulus: %w", err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, fmt.Errorf("invalid Apple key exponent: %w", err)
		}

		e := 0
		for _, b := range eBytes {
			e = e<<8 | int(b)
		}
		if e == 0 {
			return nil, errors.New("invalid Apple key exponent")
		}

		return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
	}

	return nil, fmt.Errorf("no Apple key found for kid %q", kid)
}

// validateAppleIDToken verifies the RS256 signature against Apple's JWKS and
// enforces issuer, audience and expiry. A token whose signature cannot be
// verified is rejected.
func validateAppleIDToken(cfg *Config, idToken string) (*AppleIDTokenClaims, error) {
	if cfg.AppleClientID == "" {
		return nil, errors.New("Apple sign-in is not configured (APPLE_CLIENT_ID unset)")
	}

	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid token header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("invalid token header: %w", err)
	}
	if header.Alg != "RS256" {
		return nil, fmt.Errorf("unexpected token algorithm %q", header.Alg)
	}

	pubKey, err := applePublicKey(header.Kid)
	if err != nil {
		return nil, err
	}

	signingInput := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid token signature encoding: %w", err)
	}

	digest := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, digest[:], signature); err != nil {
		return nil, errors.New("token signature verification failed")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims AppleIDTokenClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, err
	}

	if claims.Issuer != "https://appleid.apple.com" {
		return nil, fmt.Errorf("unexpected token issuer %q", claims.Issuer)
	}
	if claims.Audience != cfg.AppleClientID {
		return nil, fmt.Errorf("token audience %q does not match client id", claims.Audience)
	}
	now := time.Now().Unix()
	if claims.ExpirationTime == 0 || now >= claims.ExpirationTime {
		return nil, errors.New("token expired")
	}
	if claims.IssuedAt != 0 && claims.IssuedAt > now+300 {
		return nil, errors.New("token issued in the future")
	}

	return &claims, nil
}

// ============================================================================
// Twitter OAuth
// ============================================================================

func handleTwitterLogin(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		state := uuid.New().String()

		ctx := context.Background()
		rdb := redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		})
		defer rdb.Close()

		// Twitter uses OAuth2 with PKCE; the verifier must be persisted so the
		// callback can exchange the code. A hardcoded challenge never works.
		codeVerifier := generateCodeVerifier()
		codeChallenge := generateCodeChallenge(codeVerifier)

		if err := rdb.Set(ctx, fmt.Sprintf("oauth_state:%s", state), "valid", 10*time.Minute).Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "state_store_failed", "message": "Failed to start OAuth flow"})
			return
		}
		if err := rdb.Set(ctx, fmt.Sprintf("pkce:%s", state), codeVerifier, 10*time.Minute).Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "state_store_failed", "message": "Failed to start OAuth flow"})
			return
		}

		authURL := fmt.Sprintf("https://twitter.com/i/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=tweet.read%%20users.read%%20offline.access&state=%s&code_challenge=%s&code_challenge_method=S256",
			cfg.TwitterConsumerKey, url.QueryEscape(cfg.TwitterRedirectURL), state, codeChallenge)

		c.Redirect(http.StatusTemporaryRedirect, authURL)
	}
}

func handleTwitterCallback(cfg *Config, db *gorm.DB, rateLimiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		state := c.Query("state")

		if rateLimiter.isRateLimited("twitter_callback:"+c.ClientIP(), 10, 5*time.Minute) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited", "message": "Too many requests"})
			return
		}

		ctx := context.Background()
		rdb := redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		})
		defer rdb.Close()

		stateKey := fmt.Sprintf("oauth_state:%s", state)
		if _, err := rdb.Get(ctx, stateKey).Result(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_state", "message": "Invalid or expired state parameter"})
			return
		}
		rdb.Del(ctx, stateKey)

		codeVerifier, err := rdb.Get(ctx, fmt.Sprintf("pkce:%s", state)).Result()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_pkce", "message": "Invalid PKCE code verifier"})
			return
		}
		rdb.Del(ctx, fmt.Sprintf("pkce:%s", state))

		// Exchange code for token
		token, err := twitterOAuth2Config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token_exchange_failed", "message": "Failed to exchange code"})
			return
		}

		// Get Twitter user info
		userInfo, err := getTwitterUserInfo(token.AccessToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_info_failed", "message": "Failed to get user info"})
			return
		}

		// Find or create user
		user, err := findOrCreateSocialUser(db, "twitter", userInfo.ID, "", userInfo.Name, userInfo.ProfileImageURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_creation_failed", "message": "Failed to create user"})
			return
		}

		accessToken, _ := generateJWT(cfg, user, "twitter")
		refreshToken, _ := generateRefreshToken(cfg, user.ID, "twitter")

		redirectURL := fmt.Sprintf("%s/auth/callback?access_token=%s&refresh_token=%s", cfg.FrontendURL, accessToken, refreshToken)
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}

type TwitterUserInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	ProfileImageURL string `json:"profile_image_url"`
}

func getTwitterUserInfo(accessToken string) (*TwitterUserInfo, error) {
	req, _ := http.NewRequest("GET", "https://api.twitter.com/2/users/me?user.fields=profile_image_url", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data TwitterUserInfo `json:"data"`
	}

	json.NewDecoder(resp.Body).Decode(&result)
	return &result.Data, nil
}

// ============================================================================
// PKCE Helpers
// ============================================================================

func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// ============================================================================
// User Management
// ============================================================================

func findOrCreateSocialUser(db *gorm.DB, provider, providerUserID, email, name, picture string) (*User, error) {
	var socialAccount SocialAccount
	result := db.Where("provider = ? AND provider_user_id = ?", provider, providerUserID).First(&socialAccount)

	if result.Error == nil {
		// User exists, return the user
		var user User
		db.First(&user, socialAccount.UserID)
		return &user, nil
	}

	// Create new user
	names := strings.SplitN(name, " ", 2)
	firstName := ""
	lastName := ""
	if len(names) > 0 {
		firstName = names[0]
	}
	if len(names) > 1 {
		lastName = names[1]
	}

	user := User{
		UUID:           uuid.New().String(),
		Email:          email,
		EmailVerified:  true,
		Username:       generateUsername(db, email, provider),
		ProfilePicture: picture,
		FirstName:      firstName,
		LastName:       lastName,
		Status:         "active",
		KYCStatus:      "none",
		ReferralCode:   generateReferralCode(),
	}

	err := db.Create(&user).Error
	if err != nil {
		return nil, err
	}

	// Create social account
	socialAccount = SocialAccount{
		UserID:           user.ID,
		Provider:         provider,
		ProviderUserID:   providerUserID,
		ProviderEmail:    email,
		ProviderUsername: name,
	}
	db.Create(&socialAccount)

	return &user, nil
}

func generateUsername(db *gorm.DB, email, provider string) string {
	username := strings.Split(email, "@")[0]
	username = strings.ToLower(username)
	username = strings.ReplaceAll(username, "_", "")
	username = strings.ReplaceAll(username, ".", "")

	// Ensure uniqueness
	counter := 0
	baseUsername := username
	for {
		var count int64
		db.Model(&User{}).Where("username = ?", username).Count(&count)
		if count == 0 {
			break
		}
		counter++
		username = fmt.Sprintf("%s%d", baseUsername, counter)
	}

	return username
}

func generateReferralCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("TW%s", base64.URLEncoding.EncodeToString(b))
}

// ============================================================================
// Token Refresh
// ============================================================================

func handleRefreshToken(cfg *Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshToken := c.PostForm("refresh_token")
		if refreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing_refresh_token", "message": "Refresh token is required"})
			return
		}

		// Validate refresh token
		token, err := jwt.ParseWithClaims(refreshToken, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "message": "Invalid or expired refresh token"})
			return
		}

		claims := token.Claims.(*RefreshClaims)

		// Get user
		var user User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found", "message": "User not found"})
			return
		}

		// Generate new access token
		accessToken, err := generateJWT(cfg, &user, claims.Provider)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed", "message": "Failed to generate new token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   int(cfg.JWTExpiration.Seconds()),
		})
	}
}

// ============================================================================
// Logout
// ============================================================================

func handleLogout(cfg *Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing_token", "message": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := validateJWT(cfg, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "message": "Invalid token"})
			return
		}

		// Revoke all sessions for user
		db.Model(&AuthSession{}).Where("user_id = ? AND is_revoked = ?", claims.UserID, false).Updates(map[string]interface{}{
			"is_revoked":    true,
			"revoked_at":    time.Now(),
			"revoke_reason": "user_logout",
		})

		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
	}
}

// ============================================================================
// Get Current User
// ============================================================================

func handleGetCurrentUser(cfg *Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing_token", "message": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := validateJWT(cfg, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "message": "Invalid token"})
			return
		}

		var user User
		if err := db.Preload("WalletAddresses").First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found", "message": "User not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":                 user.ID,
			"uuid":               user.UUID,
			"email":              user.Email,
			"email_verified":     user.EmailVerified,
			"username":           user.Username,
			"profile_picture":    user.ProfilePicture,
			"first_name":         user.FirstName,
			"last_name":          user.LastName,
			"status":             user.Status,
			"kyc_status":         user.KYCStatus,
			"two_factor_enabled": user.TwoFactorEnabled,
			"referral_code":      user.ReferralCode,
			"wallet_addresses":   user.WalletAddresses,
		})
	}
}

// ============================================================================
// Link Social Account
// ============================================================================

func handleLinkSocialAccount(cfg *Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing_token", "message": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := validateJWT(cfg, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "message": "Invalid token"})
			return
		}

		var request struct {
			Provider string `json:"provider" binding:"required"`
			Code     string `json:"code" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
			return
		}

		// Check if account already linked
		var existing SocialAccount
		result := db.Where("user_id = ? AND provider = ?", claims.UserID, request.Provider).First(&existing)
		if result.Error == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "account_already_linked", "message": "Social account already linked"})
			return
		}

		// Resolve the provider identity from the supplied authorization code.
		// A linked account must carry the provider's real subject id; a random
		// placeholder would link an arbitrary identity.
		var (
			providerUserID string
			providerEmail  string
		)
		switch request.Provider {
		case "google":
			tok, err := googleOAuth2Config.Exchange(c.Request.Context(), request.Code)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "token_exchange_failed", "message": "Failed to exchange authorization code"})
				return
			}
			info, err := getGoogleUserInfo(tok.AccessToken)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "user_info_failed", "message": "Failed to resolve Google identity"})
				return
			}
			providerUserID, providerEmail = info.ID, info.Email
		case "twitter":
			tok, err := twitterOAuth2Config.Exchange(c.Request.Context(), request.Code)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "token_exchange_failed", "message": "Failed to exchange authorization code"})
				return
			}
			info, err := getTwitterUserInfo(tok.AccessToken)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "user_info_failed", "message": "Failed to resolve Twitter identity"})
				return
			}
			providerUserID = info.ID
		case "apple":
			info, err := validateAppleIDToken(cfg, request.Code)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_token", "message": "Invalid Apple identity token"})
				return
			}
			providerUserID, providerEmail = info.Subject, info.Email
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_provider", "message": "Unsupported provider"})
			return
		}

		if providerUserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "identity_unresolved", "message": "Could not resolve provider identity"})
			return
		}

		// The provider identity must not already belong to another user.
		var taken SocialAccount
		if err := db.Where("provider = ? AND provider_user_id = ?", request.Provider, providerUserID).First(&taken).Error; err == nil {
			if taken.UserID != claims.UserID {
				c.JSON(http.StatusConflict, gin.H{"error": "identity_in_use", "message": "This social account is linked to another user"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Social account already linked", "provider": request.Provider})
			return
		}

		socialAccount := SocialAccount{
			UserID:         claims.UserID,
			Provider:       request.Provider,
			ProviderUserID: providerUserID,
			ProviderEmail:  providerEmail,
		}
		if err := db.Create(&socialAccount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "link_failed", "message": "Failed to link social account"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Social account linked successfully", "provider": request.Provider})
	}
}

// ============================================================================
// Unlink Social Account
// ============================================================================

func handleUnlinkSocialAccount(cfg *Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := validateJWT(cfg, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "message": "Invalid token"})
			return
		}

		provider := c.Param("provider")

		result := db.Where("user_id = ? AND provider = ?", claims.UserID, provider).Delete(&SocialAccount{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unlink_failed", "message": "Failed to unlink social account"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Social account unlinked successfully"})
	}
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := LoadConfig()

	// Fail closed: an empty JWT secret would let anyone mint valid tokens.
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required and must not be empty")
	}
	if len(cfg.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 bytes")
	}

	// Setup database
	db, err := setupDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize OAuth2
	initOAuth2(cfg)

	// Initialize rate limiter
	rateLimiter := NewRateLimiter(cfg.RedisHost, cfg.RedisPort)

	// Setup Gin
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", cfg.FrontendURL)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "social-auth"})
	})

	// OAuth routes
	oauth := router.Group("/auth")
	{
		// Google
		oauth.GET("/google", handleGoogleLogin(cfg, db))
		oauth.GET("/google/callback", handleGoogleCallback(cfg, db, rateLimiter))

		// Apple
		oauth.GET("/apple", handleAppleLogin(cfg))
		oauth.POST("/apple/callback", handleAppleCallback(cfg, db, rateLimiter))

		// Twitter
		oauth.GET("/twitter", handleTwitterLogin(cfg))
		oauth.GET("/twitter/callback", handleTwitterCallback(cfg, db, rateLimiter))

		// Token management
		oauth.POST("/refresh", handleRefreshToken(cfg, db))
		oauth.POST("/logout", handleLogout(cfg, db))
	}

	// Protected routes
	api := router.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing_token", "message": "Authorization required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := validateJWT(cfg, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "message": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_uuid", claims.UUID)
		c.Next()
	})
	{
		api.GET("/me", handleGetCurrentUser(cfg, db))
		api.POST("/link-account", handleLinkSocialAccount(cfg, db))
		api.DELETE("/unlink-account/:provider", handleUnlinkSocialAccount(cfg, db))
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		os.Exit(0)
	}()

	log.Printf("Social Auth Service starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
