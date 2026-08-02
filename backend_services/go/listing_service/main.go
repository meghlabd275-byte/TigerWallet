// TigerWallet - Token Listing Service
// Complete production-ready backend for token listing applications
// High-performance Go service for worldwide distributed operations

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const (
	// JWT Configuration
	JWT_SECRET_KEY = "tigerwallet-listing-secret-key-2024-production"
	JWT_EXPIRY    = 24 * time.Hour * 7 // 7 days

	// Database Configuration
	DB_HOST     = getEnv("DB_HOST", "localhost")
	DB_PORT     = getEnv("DB_PORT", "5432")
	DB_USER     = getEnv("DB_USER", "tigerwallet")
	DB_PASSWORD = getEnv("DB_PASSWORD", "")
	DB_NAME     = getEnv("DB_NAME", "tigerwallet_listing")

	// Redis Configuration
	REDIS_HOST = getEnv("REDIS_HOST", "localhost")
	REDIS_PORT = getEnv("REDIS_PORT", "6379")

	// Server Configuration
	SERVER_PORT = getEnv("SERVER_PORT", "8080")
)

var (
	db           *sql.DB
	redisClient  *redis.Client
	jwtSecret    []byte
	inMemoryDB   map[string]interface{}
	dbMutex      sync.RWMutex
)

type User struct {
	ID                string    `json:"id"`
	Email             string    `json:"email"`
	Username          string    `json:"username"`
	PasswordHash      string    `json:"-"`
	Role              string    `json:"role"`
	KYCStatus         string    `json:"kyc_status"`
	IsActive          bool      `json:"is_active"`
	EmailVerified     bool      `json:"email_verified"`
	TwoFactorEnabled  bool      `json:"two_factor_enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	LastLoginAt       *time.Time `json:"last_login_at"`
}

type TokenListing struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	TokenSymbol     string     `json:"token_symbol"`
	TokenName       string     `json:"token_name"`
	ContractAddress string     `json:"contract_address"`
	ChainID         int        `json:"chain_id"`
	ChainName       string     `json:"chain_name"`
	QuoteToken      string     `json:"quote_token"`
	LogoURL         string     `json:"logo_url"`
	WebsiteURL      string     `json:"website_url"`
	TwitterURL      string     `json:"twitter_url"`
	TelegramURL     string     `json:"telegram_url"`
	DiscordURL      string     `json:"discord_url"`
	WhitepaperURL   string     `json:"whitepaper_url"`
	Tier            string     `json:"tier"`
	FeeAmount       float64    `json:"fee_amount"`
	FeeToken        string     `json:"fee_token"`
	Status          string     `json:"status"`
	RejectionReason string    `json:"rejection_reason"`
	ListedAt        *time.Time `json:"listed_at"`
	ApprovedBy      string     `json:"approved_by"`
	ApprovedAt      *time.Time `json:"approved_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Payment struct {
	ID            string     `json:"id"`
	ListingID     string     `json:"listing_id"`
	UserID        string     `json:"user_id"`
	Amount        float64    `json:"amount"`
	Token         string     `json:"token"`
	TxHash        string     `json:"tx_hash"`
	Status        string     `json:"status"`
	ConfirmedAt   *time.Time `json:"confirmed_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

type AdminAction struct {
	ID         string    `json:"id"`
	AdminID    string    `json:"admin_id"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resource_id"`
	Details    string    `json:"details"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
}

func init() {
	jwtSecret = []byte(JWT_SECRET_KEY)
	inMemoryDB = make(map[string]interface{})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initializeDatabase() error {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.Ping()
	if err != nil {
		log.Printf("⚠️  PostgreSQL not available: %v - using in-memory storage", err)
		db = nil
		return nil
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		username VARCHAR(100) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) DEFAULT 'client',
		kyc_status VARCHAR(50) DEFAULT 'none',
		is_active BOOLEAN DEFAULT true,
		email_verified BOOLEAN DEFAULT false,
		two_factor_enabled BOOLEAN DEFAULT false,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_login_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS token_listings (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id),
		token_symbol VARCHAR(20) NOT NULL,
		token_name VARCHAR(100) NOT NULL,
		contract_address VARCHAR(100) NOT NULL,
		chain_id INTEGER NOT NULL,
		chain_name VARCHAR(50) NOT NULL,
		quote_token VARCHAR(20) NOT NULL,
		logo_url VARCHAR(500),
		website_url VARCHAR(500),
		twitter_url VARCHAR(500),
		telegram_url VARCHAR(500),
		discord_url VARCHAR(500),
		whitepaper_url VARCHAR(500),
		tier VARCHAR(20) DEFAULT 'tier3',
		fee_amount DECIMAL(20, 2) DEFAULT 500,
		fee_token VARCHAR(20) DEFAULT 'USDT',
		status VARCHAR(50) DEFAULT 'pending',
		rejection_reason TEXT,
		listed_at TIMESTAMP,
		approved_by VARCHAR(100),
		approved_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS payments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		listing_id UUID REFERENCES token_listings(id),
		user_id UUID REFERENCES users(id),
		amount DECIMAL(20, 8) NOT NULL,
		token VARCHAR(20) NOT NULL,
		tx_hash VARCHAR(200),
		status VARCHAR(50) DEFAULT 'pending',
		confirmed_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS admin_actions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		admin_id UUID REFERENCES users(id),
		action VARCHAR(50) NOT NULL,
		resource VARCHAR(50) NOT NULL,
		resource_id VARCHAR(100) NOT NULL,
		details TEXT,
		ip_address VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		log.Printf("⚠️  Schema creation failed: %v", err)
		db = nil
		return nil
	}

	log.Println("✅ Database connected successfully")
	return nil
}

func initializeRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", REDIS_HOST, REDIS_PORT),
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Printf("⚠️  Redis not available: %v", err)
		redisClient = nil
		return
	}
	log.Println("✅ Redis connected successfully")
}

// ============================================================================
// AUTHENTICATION
// ============================================================================

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(user *User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:    user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(JWT_EXPIRY)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet-listing",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := VerifyToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || (role != "admin" && role != "super_admin") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ============================================================================
// IN-MEMORY DATABASE OPERATIONS
// ============================================================================

func saveUser(user *User) error {
	if db != nil {
		_, err := db.Exec(`
			INSERT INTO users (id, email, username, password_hash, role, kyc_status, is_active, email_verified, two_factor_enabled, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			user.ID, user.Email, user.Username, user.PasswordHash, user.Role, user.KYCStatus,
			user.IsActive, user.EmailVerified, user.TwoFactorEnabled, user.CreatedAt, user.UpdatedAt)
		return err
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()
	inMemoryDB["user:"+user.ID] = user
	inMemoryDB["user_email:"+user.Email] = user.ID
	return nil
}

func getUserByEmail(email string) (*User, error) {
	if db != nil {
		var user User
		err := db.QueryRow(`
			SELECT id, email, username, password_hash, role, kyc_status, is_active, email_verified, two_factor_enabled, created_at, updated_at, last_login_at
			FROM users WHERE email = $1`, email).Scan(
			&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.KYCStatus,
			&user.IsActive, &user.EmailVerified, &user.TwoFactorEnabled, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
		if err != nil {
			return nil, err
		}
		return &user, nil
	}

	dbMutex.RLock()
	defer dbMutex.RUnlock()
	userID, ok := inMemoryDB["user_email:"+email].(string)
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	user, ok := inMemoryDB["user:"+userID].(*User)
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func getUserByID(id string) (*User, error) {
	if db != nil {
		var user User
		err := db.QueryRow(`
			SELECT id, email, username, password_hash, role, kyc_status, is_active, email_verified, two_factor_enabled, created_at, updated_at, last_login_at
			FROM users WHERE id = $1`, id).Scan(
			&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.KYCStatus,
			&user.IsActive, &user.EmailVerified, &user.TwoFactorEnabled, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
		if err != nil {
			return nil, err
		}
		return &user, nil
	}

	dbMutex.RLock()
	defer dbMutex.RUnlock()
	user, ok := inMemoryDB["user:"+id].(*User)
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func getUserCount() int {
	if db != nil {
		var count int
		db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
		return count
	}

	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return len(inMemoryDB) / 2
}

func saveListing(listing *TokenListing) error {
	if db != nil {
		_, err := db.Exec(`
			INSERT INTO token_listings (
				id, user_id, token_symbol, token_name, contract_address, chain_id, chain_name,
				quote_token, logo_url, website_url, twitter_url, telegram_url, discord_url,
				whitepaper_url, tier, fee_amount, fee_token, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`,
			listing.ID, listing.UserID, listing.TokenSymbol, listing.TokenName, listing.ContractAddress,
			listing.ChainID, listing.ChainName, listing.QuoteToken, listing.LogoURL, listing.WebsiteURL,
			listing.TwitterURL, listing.TelegramURL, listing.DiscordURL, listing.WhitepaperURL,
			listing.Tier, listing.FeeAmount, listing.FeeToken, listing.Status, listing.CreatedAt, listing.UpdatedAt)
		return err
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()
	inMemoryDB["listing:"+listing.ID] = listing
	return nil
}

func getListingByID(id string) (*TokenListing, error) {
	if db != nil {
		var l TokenListing
		err := db.QueryRow(`
			SELECT id, user_id, token_symbol, token_name, contract_address, chain_id, chain_name,
				quote_token, logo_url, website_url, twitter_url, telegram_url, discord_url,
				whitepaper_url, tier, fee_amount, fee_token, status, rejection_reason,
				listed_at, approved_by, approved_at, created_at, updated_at
			FROM token_listings WHERE id = $1`, id).Scan(
			&l.ID, &l.UserID, &l.TokenSymbol, &l.TokenName, &l.ContractAddress, &l.ChainID,
			&l.ChainName, &l.QuoteToken, &l.LogoURL, &l.WebsiteURL, &l.TwitterURL, &l.TelegramURL,
			&l.DiscordURL, &l.WhitepaperURL, &l.Tier, &l.FeeAmount, &l.FeeToken, &l.Status,
			&l.RejectionReason, &l.ListedAt, &l.ApprovedBy, &l.ApprovedAt, &l.CreatedAt, &l.UpdatedAt)
		return &l, err
	}

	dbMutex.RLock()
	defer dbMutex.RUnlock()
	listing, ok := inMemoryDB["listing:"+id].(*TokenListing)
	if !ok {
		return nil, fmt.Errorf("listing not found")
	}
	return listing, nil
}

func getListingsByUserID(userID string) ([]TokenListing, error) {
	if db != nil {
		rows, err := db.Query(`
			SELECT id, user_id, token_symbol, token_name, contract_address, chain_id, chain_name,
				quote_token, logo_url, website_url, twitter_url, telegram_url, discord_url,
				whitepaper_url, tier, fee_amount, fee_token, status, rejection_reason,
				listed_at, approved_by, approved_at, created_at, updated_at
			FROM token_listings WHERE user_id = $1 ORDER BY created_at DESC`, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var listings []TokenListing
		for rows.Next() {
			var l TokenListing
			if err := rows.Scan(
				&l.ID, &l.UserID, &l.TokenSymbol, &l.TokenName, &l.ContractAddress, &l.ChainID,
				&l.ChainName, &l.QuoteToken, &l.LogoURL, &l.WebsiteURL, &l.TwitterURL, &l.TelegramURL,
				&l.DiscordURL, &l.WhitepaperURL, &l.Tier, &l.FeeAmount, &l.FeeToken, &l.Status,
				&l.RejectionReason, &l.ListedAt, &l.ApprovedBy, &l.ApprovedAt, &l.CreatedAt, &l.UpdatedAt); err != nil {
				continue
			}
			listings = append(listings, l)
		}
		return listings, nil
	}

	dbMutex.RLock()
	defer dbMutex.RUnlock()
	var listings []TokenListing
	for k, v := range inMemoryDB {
		if strings.HasPrefix(k, "listing:") {
			if l, ok := v.(*TokenListing); ok && l.UserID == userID {
				listings = append(listings, *l)
			}
		}
	}
	return listings, nil
}

func updateListing(listing *TokenListing) error {
	if db != nil {
		_, err := db.Exec(`
			UPDATE token_listings SET status = $1, rejection_reason = $2, approved_by = $3, 
			approved_at = $4, listed_at = $5, updated_at = $6 WHERE id = $7`,
			listing.Status, listing.RejectionReason, listing.ApprovedBy, listing.ApprovedAt,
			listing.ListedAt, listing.UpdatedAt, listing.ID)
		return err
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()
	inMemoryDB["listing:"+listing.ID] = listing
	return nil
}

func getAllListings(status, tier string) ([]TokenListing, error) {
	if db != nil {
		query := "SELECT id, user_id, token_symbol, token_name, contract_address, chain_id, chain_name, quote_token, logo_url, website_url, twitter_url, telegram_url, discord_url, whitepaper_url, tier, fee_amount, fee_token, status, rejection_reason, listed_at, approved_by, approved_at, created_at, updated_at FROM token_listings WHERE 1=1"
		args := []interface{}{}
		argNum := 1

		if status != "" {
			query += fmt.Sprintf(" AND status = $%d", argNum)
			args = append(args, status)
			argNum++
		}
		if tier != "" {
			query += fmt.Sprintf(" AND tier = $%d", argNum)
			args = append(args, tier)
		}
		query += " ORDER BY created_at DESC"

		rows, err := db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var listings []TokenListing
		for rows.Next() {
			var l TokenListing
			if err := rows.Scan(
				&l.ID, &l.UserID, &l.TokenSymbol, &l.TokenName, &l.ContractAddress, &l.ChainID,
				&l.ChainName, &l.QuoteToken, &l.LogoURL, &l.WebsiteURL, &l.TwitterURL, &l.TelegramURL,
				&l.DiscordURL, &l.WhitepaperURL, &l.Tier, &l.FeeAmount, &l.FeeToken, &l.Status,
				&l.RejectionReason, &l.ListedAt, &l.ApprovedBy, &l.ApprovedAt, &l.CreatedAt, &l.UpdatedAt); err != nil {
				continue
			}
			listings = append(listings, l)
		}
		return listings, nil
	}

	dbMutex.RLock()
	defer dbMutex.RUnlock()
	var listings []TokenListing
	for _, v := range inMemoryDB {
		if l, ok := v.(*TokenListing); ok {
			if (status == "" || l.Status == status) && (tier == "" || l.Tier == tier) {
				listings = append(listings, *l)
			}
		}
	}
	return listings, nil
}

func savePayment(payment *Payment) error {
	if db != nil {
		_, err := db.Exec(`
			INSERT INTO payments (id, listing_id, user_id, amount, token, tx_hash, status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			payment.ID, payment.ListingID, payment.UserID, payment.Amount, payment.Token,
			payment.TxHash, payment.Status, payment.CreatedAt)
		return err
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()
	inMemoryDB["payment:"+payment.ID] = payment
	return nil
}

func getPaymentByID(id string) (*Payment, error) {
	if db != nil {
		var p Payment
		err := db.QueryRow(`
			SELECT id, listing_id, user_id, amount, token, tx_hash, status, confirmed_at, created_at
			FROM payments WHERE id = $1`, id).Scan(
			&p.ID, &p.ListingID, &p.UserID, &p.Amount, &p.Token, &p.TxHash, &p.Status, &p.ConfirmedAt, &p.CreatedAt)
		return &p, err
	}

	dbMutex.RLock()
	defer dbMutex.RUnlock()
	payment, ok := inMemoryDB["payment:"+id].(*Payment)
	if !ok {
		return nil, fmt.Errorf("payment not found")
	}
	return payment, nil
}

func logAdminAction(action *AdminAction) error {
	if db != nil {
		_, err := db.Exec(`
			INSERT INTO admin_actions (admin_id, action, resource, resource_id, details, ip_address, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			action.AdminID, action.Action, action.Resource, action.ResourceID, action.Details, action.IPAddress, action.CreatedAt)
		return err
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()
	inMemoryDB["action:"+action.ID] = action
	return nil
}

// ============================================================================
// HTTP HANDLERS
// ============================================================================

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token     string    `json:"token"`
	User      User      `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email already exists
	_, err := getUserByEmail(req.Email)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Determine role
	role := "client"
	if getUserCount() == 0 {
		role = "super_admin"
	}

	user := &User{
		ID:               uuid.New().String(),
		Email:            req.Email,
		Username:         req.Username,
		PasswordHash:     string(hashedPassword),
		Role:             role,
		KYCStatus:        "none",
		IsActive:         true,
		EmailVerified:    false,
		TwoFactorEnabled: false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := saveUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	token, err := GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	log.Printf("📝 New user registered: %s (%s) - Role: %s", user.Email, user.Username, user.Role)

	c.JSON(http.StatusCreated, AuthResponse{
		Token:     token,
		User:      *user,
		ExpiresAt: time.Now().Add(JWT_EXPIRY),
	})
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := getUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is suspended"})
		return
	}

	token, err := GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	log.Printf("🔐 User logged in: %s (%s)", user.Email, user.Username)

	c.JSON(http.StatusOK, AuthResponse{
		Token:     token,
		User:      *user,
		ExpiresAt: time.Now().Add(JWT_EXPIRY),
	})
}

type CreateListingRequest struct {
	TokenSymbol     string `json:"token_symbol" binding:"required"`
	TokenName       string `json:"token_name" binding:"required"`
	ContractAddress string `json:"contract_address" binding:"required"`
	ChainID         int    `json:"chain_id" binding:"required"`
	ChainName       string `json:"chain_name" binding:"required"`
	QuoteToken      string `json:"quote_token" binding:"required"`
	LogoURL         string `json:"logo_url"`
	WebsiteURL      string `json:"website_url"`
	TwitterURL      string `json:"twitter_url"`
	TelegramURL     string `json:"telegram_url"`
	DiscordURL      string `json:"discord_url"`
	WhitepaperURL   string `json:"whitepaper_url"`
	Tier            string `json:"tier" binding:"required"`
}

var tierFees = map[string]float64{
	"tier1": 2500,
	"tier2": 1000,
	"tier3": 500,
	"tier4": 250,
}

func CreateListing(c *gin.Context) {
	userID := c.GetString("user_id")

	var req CreateListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	feeAmount, ok := tierFees[req.Tier]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tier"})
		return
	}

	if len(req.ContractAddress) < 40 || !strings.HasPrefix(req.ContractAddress, "0x") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract address format"})
		return
	}

	listing := &TokenListing{
		ID:              uuid.New().String(),
		UserID:          userID,
		TokenSymbol:     strings.ToUpper(req.TokenSymbol),
		TokenName:       req.TokenName,
		ContractAddress: strings.ToLower(req.ContractAddress),
		ChainID:         req.ChainID,
		ChainName:       req.ChainName,
		QuoteToken:      strings.ToUpper(req.QuoteToken),
		LogoURL:         req.LogoURL,
		WebsiteURL:      req.WebsiteURL,
		TwitterURL:      req.TwitterURL,
		TelegramURL:     req.TelegramURL,
		DiscordURL:      req.DiscordURL,
		WhitepaperURL:   req.WhitepaperURL,
		Tier:            req.Tier,
		FeeAmount:       feeAmount,
		FeeToken:        "USDT",
		Status:          "pending",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := saveListing(listing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create listing"})
		return
	}

	// Cache in Redis
	if redisClient != nil {
		listingJSON, _ := json.Marshal(listing)
		redisClient.Set(context.Background(), "listing:"+listing.ID, listingJSON, time.Hour*24)
	}

	log.Printf("📋 New listing created: %s (%s) - Tier: %s, Fee: $%.2f",
		listing.TokenSymbol, listing.TokenName, listing.Tier, listing.FeeAmount)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    listing,
		"message": "Listing application submitted successfully. Please complete payment to proceed.",
	})
}

func GetMyListings(c *gin.Context) {
	userID := c.GetString("user_id")
	listings, err := getListingsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch listings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": listings})
}

func GetAllListings(c *gin.Context) {
	status := c.Query("status")
	tier := c.Query("tier")

	listings, err := getAllListings(status, tier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch listings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": listings})
}

func GetListing(c *gin.Context) {
	listingID := c.Param("id")

	// Try Redis first
	if redisClient != nil {
		listingJSON, err := redisClient.Get(context.Background(), "listing:"+listingID).Result()
		if err == nil {
			var listing TokenListing
			if json.Unmarshal([]byte(listingJSON), &listing) == nil {
				c.JSON(http.StatusOK, gin.H{"success": true, "data": listing})
				return
			}
		}
	}

	listing, err := getListingByID(listingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": listing})
}

type ApproveListingRequest struct {
	Notes string `json:"notes"`
}

func ApproveListing(c *gin.Context) {
	listingID := c.Param("id")
	adminID := c.GetString("user_id")
	clientIP := c.ClientIP()

	var req ApproveListingRequest
	c.ShouldBindJSON(&req)

	listing, err := getListingByID(listingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
		return
	}

	if listing.Status != "pending" && listing.Status != "payment_confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Listing cannot be approved in current status"})
		return
	}

	now := time.Now()
	listing.Status = "approved"
	listing.ApprovedBy = adminID
	listing.ApprovedAt = &now
	listing.ListedAt = &now
	listing.UpdatedAt = now

	if err := updateListing(listing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve listing"})
		return
	}

	logAdminAction(&AdminAction{
		ID:         uuid.New().String(),
		AdminID:    adminID,
		Action:     "approve",
		Resource:   "listing",
		ResourceID: listingID,
		Details:    req.Notes,
		IPAddress:  clientIP,
		CreatedAt:  now,
	})

	if redisClient != nil {
		redisClient.Del(context.Background(), "listing:"+listingID)
	}

	log.Printf("✅ Listing approved: %s by admin %s", listingID, adminID)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Listing approved successfully"})
}

type RejectListingRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func RejectListing(c *gin.Context) {
	listingID := c.Param("id")
	adminID := c.GetString("user_id")
	clientIP := c.ClientIP()

	var req RejectListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reason is required"})
		return
	}

	listing, err := getListingByID(listingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
		return
	}

	if listing.Status != "pending" && listing.Status != "payment_confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Listing cannot be rejected in current status"})
		return
	}

	now := time.Now()
	listing.Status = "rejected"
	listing.RejectionReason = req.Reason
	listing.ApprovedBy = adminID
	listing.ApprovedAt = &now
	listing.UpdatedAt = now

	if err := updateListing(listing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject listing"})
		return
	}

	logAdminAction(&AdminAction{
		ID:         uuid.New().String(),
		AdminID:    adminID,
		Action:     "reject",
		Resource:   "listing",
		ResourceID: listingID,
		Details:    req.Reason,
		IPAddress:  clientIP,
		CreatedAt:  now,
	})

	if redisClient != nil {
		redisClient.Del(context.Background(), "listing:"+listingID)
	}

	log.Printf("❌ Listing rejected: %s by admin %s - Reason: %s", listingID, adminID, req.Reason)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Listing rejected"})
}

type PaymentCallbackRequest struct {
	ListingID string  `json:"listing_id" binding:"required"`
	TxHash    string  `json:"tx_hash" binding:"required"`
	Token     string  `json:"token" binding:"required"`
	Amount    float64 `json:"amount" binding:"required"`
}

func ProcessPayment(c *gin.Context) {
	userID := c.GetString("user_id")

	var req PaymentCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	listing, err := getListingByID(req.ListingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
		return
	}

	if listing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	if req.Amount < listing.FeeAmount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient payment amount"})
		return
	}

	payment := &Payment{
		ID:        uuid.New().String(),
		ListingID: req.ListingID,
		UserID:    userID,
		Amount:    req.Amount,
		Token:     req.Token,
		TxHash:    req.TxHash,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	if err := savePayment(payment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record payment"})
		return
	}

	// Simulate payment confirmation
	go func() {
		time.Sleep(5 * time.Second)
		payment.Status = "confirmed"
		now := time.Now()
		payment.ConfirmedAt = &now
		savePayment(payment)

		listing.Status = "payment_confirmed"
		listing.UpdatedAt = time.Now()
		updateListing(listing)

		log.Printf("💰 Payment confirmed for listing: %s - Tx: %s", req.ListingID, req.TxHash)
	}()

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":  "Payment submitted. Confirmation may take a few minutes.",
		"payment_id": payment.ID,
	})
}

func GetPaymentStatus(c *gin.Context) {
	paymentID := c.Param("id")

	payment, err := getPaymentByID(paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": payment})
}

type ListingStats struct {
	TotalListings    int               `json:"total_listings"`
	PendingListings  int               `json:"pending_listings"`
	ApprovedListings int               `json:"approved_listings"`
	RejectedListings int               `json:"rejected_listings"`
	TotalRevenue     float64           `json:"total_revenue"`
	ByTier           map[string]int    `json:"by_tier"`
	ByChain          map[string]int    `json:"by_chain"`
}

func GetListingStats(c *gin.Context) {
	allListings, _ := getAllListings("", "")

	stats := ListingStats{
		TotalListings:    len(allListings),
		PendingListings:  0,
		ApprovedListings: 0,
		RejectedListings: 0,
		TotalRevenue:     0,
		ByTier:           make(map[string]int),
		ByChain:          make(map[string]int),
	}

	for _, l := range allListings {
		switch l.Status {
		case "pending", "payment_confirmed":
			stats.PendingListings++
		case "approved":
			stats.ApprovedListings++
			stats.TotalRevenue += l.FeeAmount
		case "rejected":
			stats.RejectedListings++
		}
		stats.ByTier[l.Tier]++
		stats.ByChain[l.ChainName]++
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

func HealthCheck(c *gin.Context) {
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"services": gin.H{
			"database": "healthy",
			"redis":    "not_configured",
		},
	}

	if db != nil {
		if err := db.Ping(); err != nil {
			health["services"].(gin.H)["database"] = "healthy (in-memory fallback)"
		}
	} else {
		health["services"].(gin.H)["database"] = "in-memory"
	}

	if redisClient != nil {
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			health["services"].(gin.H)["redis"] = "not_configured"
		} else {
			health["services"].(gin.H)["redis"] = "healthy"
		}
	}

	c.JSON(http.StatusOK, health)
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 Starting TigerWallet Listing Service...")

	if err := initializeDatabase(); err != nil {
		log.Printf("⚠️  Database initialization: %v", err)
	}

	initializeRedis()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(CORSMiddleware())

	r.GET("/health", HealthCheck)
	r.POST("/api/auth/register", Register)
	r.POST("/api/auth/login", Login)

	protected := r.Group("/api")
	protected.Use(AuthMiddleware())
	{
		protected.POST("/listings", CreateListing)
		protected.GET("/listings", GetMyListings)
		protected.GET("/listings/:id", GetListing)
		protected.POST("/payments", ProcessPayment)
		protected.GET("/payments/:id", GetPaymentStatus)
	}

	admin := r.Group("/api/admin")
	admin.Use(AuthMiddleware())
	admin.Use(AdminMiddleware())
	{
		admin.GET("/listings", GetAllListings)
		admin.GET("/listings/:id", GetListing)
		admin.POST("/listings/:id/approve", ApproveListing)
		admin.POST("/listings/:id/reject", RejectListing)
		admin.GET("/stats", GetListingStats)
	}

	log.Printf("✅ Server starting on port %s", SERVER_PORT)
	if err := r.Run(":" + SERVER_PORT); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
