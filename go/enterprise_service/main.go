package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	Security   SecurityConfig
	Blockchain BlockchainConfig
	Enterprise EnterpriseConfig
}

type ServerConfig struct {
	Port           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
	EnableTLS      bool
	TLSCertFile    string
	TLSKeyFile     string
	RateLimit      RateLimitConfig
}

type RateLimitConfig struct {
	RequestsPerSecond int
	Burst             int
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

type SecurityConfig struct {
	JWTSecret        string
	JWTExpiration    time.Duration
	BCryptCost       int
	EnableMFA        bool
	SessionTimeout   time.Duration
	MaxLoginAttempts int
	LockoutDuration  time.Duration
}

type BlockchainConfig struct {
	Networks           []NetworkConfig
	GasOracleURL       string
	MaxGasPrice        uint64
	ConfirmationBlocks uint64
}

type NetworkConfig struct {
	ID          uint64
	Name        string
	Symbol      string
	ChainID     int64
	RPCURLs     []string
	ExplorerURL string
	IsEnabled   bool
	IsTestnet   bool
}

type EnterpriseConfig struct {
	EnableWhiteLabel  bool
	EnableBroker      bool
	EnableInstitution bool
	MaxWhiteLabels    int
	MaxAPIKeys        int
	WebhookTimeout    time.Duration
}

func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:           getEnv("PORT", "8443"),
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
			EnableTLS:      getEnv("ENABLE_TLS", "false") == "true",
			TLSCertFile:    getEnv("TLS_CERT_FILE", "server.crt"),
			TLSKeyFile:     getEnv("TLS_KEY_FILE", "server.key"),
			RateLimit: RateLimitConfig{
				RequestsPerSecond: 1000,
				Burst:             2000,
			},
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            5432,
			User:            getEnv("DB_USER", "tigerwallet"),
			Password:        getEnv("DB_PASSWORD", "password"),
			Database:        getEnv("DB_NAME", "tigerwallet"),
			MaxOpenConns:    100,
			MaxIdleConns:    10,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         6379,
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           0,
			PoolSize:     100,
			MinIdleConns: 10,
		},
		Security: SecurityConfig{
			JWTSecret:        getEnv("JWT_SECRET", ""),
			JWTExpiration:    24 * 7 * time.Hour,
			BCryptCost:       12,
			EnableMFA:        true,
			SessionTimeout:   24 * time.Hour,
			MaxLoginAttempts: 5,
			LockoutDuration:  15 * time.Minute,
		},
		Blockchain: BlockchainConfig{
			Networks: []NetworkConfig{
				{ID: 1, Name: "Ethereum", Symbol: "ETH", ChainID: 1, IsEnabled: true},
				{ID: 2, Name: "Polygon", Symbol: "MATIC", ChainID: 137, IsEnabled: true},
				{ID: 3, Name: "Arbitrum", Symbol: "ETH", ChainID: 42161, IsEnabled: true},
				{ID: 4, Name: "Optimism", Symbol: "ETH", ChainID: 10, IsEnabled: true},
				{ID: 5, Name: "Avalanche", Symbol: "AVAX", ChainID: 43114, IsEnabled: true},
				{ID: 6, Name: "BNB Chain", Symbol: "BNB", ChainID: 56, IsEnabled: true},
			},
			GasOracleURL:       "https://api.etherscan.io/api?module=gastracker&action=gasoracle",
			MaxGasPrice:        500000000000, // 500 Gwei
			ConfirmationBlocks: 12,
		},
		Enterprise: EnterpriseConfig{
			EnableWhiteLabel:  true,
			EnableBroker:      true,
			EnableInstitution: true,
			MaxWhiteLabels:    1000,
			MaxAPIKeys:        100,
			WebhookTimeout:    30 * time.Second,
		},
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
	ID               string     `json:"id" db:"id"`
	Email            string     `json:"email" db:"email"`
	Username         string     `json:"username" db:"username"`
	PasswordHash     string     `json:"-" db:"password_hash"`
	KYCStatus        string     `json:"kycStatus" db:"kyc_status"`
	KYCLevel         int        `json:"kycLevel" db:"kyc_level"`
	EmailVerified    bool       `json:"emailVerified" db:"email_verified"`
	PhoneVerified    bool       `json:"phoneVerified" db:"phone_verified"`
	MFAEnabled       bool       `json:"mfaEnabled" db:"mfa_enabled"`
	MFASecret        string     `json:"-" db:"mfa_secret"`
	FailedLoginCount int        `json:"failedLoginCount" db:"failed_login_count"`
	LockedUntil      *time.Time `json:"lockedUntil" db:"locked_until"`
	RiskScore        int        `json:"riskScore" db:"risk_score"`
	CreatedAt        time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time  `json:"updatedAt" db:"updated_at"`
	LastLoginAt      *time.Time `json:"lastLoginAt" db:"last_login_at"`
	IPWhitelist      []string   `json:"ipWhitelist" db:"ip_whitelist"`
}

type WhiteLabelClient struct {
	ID              string    `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	Domain          string    `json:"domain" db:"domain"`
	Email           string    `json:"email" db:"email"`
	Status          string    `json:"status" db:"status"` // active, paused, halted
	FeePercentage   float64   `json:"feePercentage" db:"fee_percentage"`
	APIKey          string    `json:"apiKey" db:"-"`
	APIKeyHash      string    `json:"-" db:"api_key_hash"`
	AllowedChains   []string  `json:"allowedChains" db:"allowed_chains"`
	AllowedFeatures []string  `json:"allowedFeatures" db:"allowed_features"`
	CustomBranding  bool      `json:"customBranding" db:"custom_branding"`
	MaxUsers        int       `json:"maxUsers" db:"max_users"`
	CurrentUsers    int       `json:"currentUsers" db:"current_users"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time `json:"updatedAt" db:"updated_at"`
}

type Broker struct {
	ID             string    `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Email          string    `json:"email" db:"email"`
	WhiteLabelID   string    `json:"whiteLabelId" db:"white_label_id"`
	Status         string    `json:"status" db:"status"`
	CommissionRate float64   `json:"commissionRate" db:"commission_rate"`
	APIKey         string    `json:"apiKey" db:"-"`
	APIKeyHash     string    `json:"-" db:"api_key_hash"`
	AllowedIPs     []string  `json:"allowedIPs" db:"allowed_ips"`
	MaxDailyVolume float64   `json:"maxDailyVolume" db:"max_daily_volume"`
	CurrentVolume  float64   `json:"currentVolume" db:"current_volume"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

type Institution struct {
	ID            string    `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Email         string    `json:"email" db:"email"`
	WhiteLabelID  string    `json:"whiteLabelId" db:"white_label_id"`
	Status        string    `json:"status" db:"status"`
	APIKey        string    `json:"apiKey" db:"-"`
	APIKeyHash    string    `json:"-" db:"api_key_hash"`
	KYCStatus     string    `json:"kycStatus" db:"kyc_status"`
	AccountType   string    `json:"accountType" db:"account_type"` // retail, professional, institutional
	TradingLimits float64   `json:"tradingLimits" db:"trading_limits"`
	FeeTier       int       `json:"feeTier" db:"fee_tier"`
	AllowedChains []string  `json:"allowedChains" db:"allowed_chains"`
	WebhookURL    string    `json:"webhookUrl" db:"webhook_url"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time `json:"updatedAt" db:"updated_at"`
}

type Wallet struct {
	ID           string    `json:"id" db:"id"`
	UserID       string    `json:"userId" db:"user_id"`
	WhiteLabelID string    `json:"whiteLabelId" db:"white_label_id"`
	WalletType   string    `json:"walletType" db:"wallet_type"` // user, master
	Address      string    `json:"address" db:"address"`
	ChainID      uint64    `json:"chainId" db:"chain_id"`
	EncryptedKey string    `json:"-" db:"encrypted_key"`
	IsActive     bool      `json:"isActive" db:"is_active"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

type Transaction struct {
	ID           string    `json:"id" db:"id"`
	Hash         string    `json:"hash" db:"hash"`
	FromAddress  string    `json:"fromAddress" db:"from_address"`
	ToAddress    string    `json:"toAddress" db:"to_address"`
	Value        string    `json:"value" db:"value"`
	GasPrice     string    `json:"gasPrice" db:"gas_price"`
	GasLimit     uint64    `json:"gasLimit" db:"gas_limit"`
	GasUsed      uint64    `json:"gasUsed" db:"gas_used"`
	ChainID      uint64    `json:"chainId" db:"chain_id"`
	Status       string    `json:"status" db:"status"` // pending, confirmed, failed
	ErrorMessage string    `json:"errorMessage" db:"error_message"`
	BlockNumber  uint64    `json:"blockNumber" db:"block_number"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
}

type APIKey struct {
	ID           string     `json:"id" db:"id"`
	UserID       string     `json:"userId" db:"user_id"`
	WhiteLabelID string     `json:"whiteLabelId" db:"white_label_id"`
	KeyHash      string     `json:"-" db:"key_hash"`
	Name         string     `json:"name" db:"name"`
	Permissions  []string   `json:"permissions" db:"permissions"`
	RateLimit    int        `json:"rateLimit" db:"rate_limit"`
	ExpiresAt    *time.Time `json:"expiresAt" db:"expires_at"`
	LastUsedAt   *time.Time `json:"lastUsedAt" db:"last_used_at"`
	IsActive     bool       `json:"isActive" db:"is_active"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
}

type AuditLog struct {
	ID         string    `json:"id" db:"id"`
	UserID     string    `json:"userId" db:"user_id"`
	Action     string    `json:"action" db:"action"`
	Resource   string    `json:"resource" db:"resource"`
	ResourceID string    `json:"resourceId" db:"resource_id"`
	Details    string    `json:"details" db:"details"`
	IPAddress  string    `json:"ipAddress" db:"ip_address"`
	UserAgent  string    `json:"userAgent" db:"user_agent"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
}

type Token struct {
	ID        string    `json:"id" db:"id"`
	Address   string    `json:"address" db:"address"`
	ChainID   uint64    `json:"chainId" db:"chain_id"`
	Name      string    `json:"name" db:"name"`
	Symbol    string    `json:"symbol" db:"symbol"`
	Decimals  int       `json:"decimals" db:"decimals"`
	IsEnabled bool      `json:"isEnabled" db:"is_enabled"`
	IsPopular bool      `json:"isPopular" db:"is_popular"`
	LogoURL   string    `json:"logoUrl" db:"logo_url"`
	PriceUSD  float64   `json:"priceUsd" db:"price_usd"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// ============================================================================
// Services
// ============================================================================

type EnterpriseService struct {
	config      *Config
	db          *sql.DB
	redis       *redis.Client
	authService *AuthService
	walletSvc   *WalletService
	txSvc       *TransactionService
	apiKeySvc   *APIKeyService
	auditSvc    *AuditService
	metrics     *Metrics
	wsHub       *WebSocketHub
}

func NewEnterpriseService(config *Config) (*EnterpriseService, error) {
	// Initialize database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.Database.Host, config.Database.Port, config.Database.User,
		config.Database.Password, config.Database.Database)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(config.Database.MaxOpenConns)
	db.SetMaxIdleConns(config.Database.MaxIdleConns)
	db.SetConnMaxLifetime(config.Database.ConnMaxLifetime)

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
		PoolSize: config.Redis.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	service := &EnterpriseService{
		config:      config,
		db:          db,
		redis:       redisClient,
		authService: NewAuthService(config.Security, db),
		walletSvc:   NewWalletService(db, redisClient),
		txSvc:       NewTransactionService(db, redisClient, config.Blockchain),
		apiKeySvc:   NewAPIKeyService(db, redisClient),
		auditSvc:    NewAuditService(db),
		metrics:     NewMetrics(),
		wsHub:       NewWebSocketHub(),
	}

	// Initialize database tables
	if err := service.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return service, nil
}

func (s *EnterpriseService) initDatabase() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			kyc_status VARCHAR(50) DEFAULT 'pending',
			kyc_level INTEGER DEFAULT 0,
			email_verified BOOLEAN DEFAULT FALSE,
			phone_verified BOOLEAN DEFAULT FALSE,
			mfa_enabled BOOLEAN DEFAULT FALSE,
			mfa_secret VARCHAR(255),
			failed_login_count INTEGER DEFAULT 0,
			locked_until TIMESTAMP,
			risk_score INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			last_login_at TIMESTAMP,
			ip_whitelist TEXT[]
		)`,
		`CREATE TABLE IF NOT EXISTS white_labels (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			domain VARCHAR(255) UNIQUE NOT NULL,
			email VARCHAR(255) NOT NULL,
			status VARCHAR(50) DEFAULT 'active',
			fee_percentage DECIMAL(5,4) DEFAULT 0.001,
			api_key_hash VARCHAR(255),
			allowed_chains TEXT[],
			allowed_features TEXT[],
			custom_branding BOOLEAN DEFAULT TRUE,
			max_users INTEGER DEFAULT 10000,
			current_users INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS brokers (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			white_label_id UUID REFERENCES white_labels(id),
			status VARCHAR(50) DEFAULT 'active',
			commission_rate DECIMAL(5,4) DEFAULT 0.002,
			api_key_hash VARCHAR(255),
			allowed_ips TEXT[],
			max_daily_volume DECIMAL(20,8) DEFAULT 1000000,
			current_volume DECIMAL(20,8) DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS institutions (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			white_label_id UUID REFERENCES white_labels(id),
			status VARCHAR(50) DEFAULT 'active',
			api_key_hash VARCHAR(255),
			kyc_status VARCHAR(50) DEFAULT 'pending',
			account_type VARCHAR(50) DEFAULT 'retail',
			trading_limits DECIMAL(20,8) DEFAULT 100000,
			fee_tier INTEGER DEFAULT 1,
			allowed_chains TEXT[],
			webhook_url VARCHAR(500),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS wallets (
			id UUID PRIMARY KEY,
			user_id UUID REFERENCES users(id),
			white_label_id UUID REFERENCES white_labels(id),
			wallet_type VARCHAR(50) NOT NULL,
			address VARCHAR(100) NOT NULL,
			chain_id BIGINT NOT NULL,
			encrypted_key TEXT,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(address, chain_id)
		)`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id UUID PRIMARY KEY,
			hash VARCHAR(100) UNIQUE NOT NULL,
			from_address VARCHAR(100) NOT NULL,
			to_address VARCHAR(100) NOT NULL,
			value VARCHAR(50) NOT NULL,
			gas_price VARCHAR(50),
			gas_limit BIGINT,
			gas_used BIGINT,
			chain_id BIGINT NOT NULL,
			status VARCHAR(50) DEFAULT 'pending',
			error_message TEXT,
			block_number BIGINT,
			timestamp TIMESTAMP DEFAULT NOW(),
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id UUID PRIMARY KEY,
			user_id UUID REFERENCES users(id),
			white_label_id UUID REFERENCES white_labels(id),
			key_hash VARCHAR(255) NOT NULL,
			name VARCHAR(100) NOT NULL,
			permissions TEXT[],
			rate_limit INTEGER DEFAULT 1000,
			expires_at TIMESTAMP,
			last_used_at TIMESTAMP,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY,
			user_id UUID,
			action VARCHAR(100) NOT NULL,
			resource VARCHAR(100) NOT NULL,
			resource_id VARCHAR(100),
			details TEXT,
			ip_address VARCHAR(50),
			user_agent TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS tokens (
			id UUID PRIMARY KEY,
			address VARCHAR(100),
			chain_id BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			symbol VARCHAR(50) NOT NULL,
			decimals INTEGER NOT NULL,
			is_enabled BOOLEAN DEFAULT TRUE,
			is_popular BOOLEAN DEFAULT FALSE,
			logo_url VARCHAR(500),
			price_usd DECIMAL(20,8),
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_wallets_address ON wallets(address)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_hash ON transactions(hash)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// ============================================================================
// Authentication Service
// ============================================================================

type AuthService struct {
	config *SecurityConfig
	db     *sql.DB
}

func NewAuthService(config SecurityConfig, db *sql.DB) *AuthService {
	return &AuthService{config: &config, db: db}
}

type JWTClaims struct {
	UserID       string   `json:"userId"`
	Email        string   `json:"email"`
	Role         string   `json:"role"`
	WhiteLabelID string   `json:"whiteLabelId,omitempty"`
	Permissions  []string `json:"permissions"`
	jwt.RegisteredClaims
}

func (s *AuthService) GenerateToken(user *User, permissions []string) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.config.JWTExpiration)

	claims := JWTClaims{
		UserID:      user.ID,
		Email:       user.Email,
		Role:        "user",
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.config.JWTSecret))
	return signed, expiresAt, err
}

func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *AuthService) RegisterUser(email, username, password string) (*User, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), s.config.BCryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		ID:            uuid.New().String(),
		Email:         email,
		Username:      username,
		PasswordHash:  string(hashedPassword),
		KYCStatus:     "pending",
		KYCLevel:      0,
		EmailVerified: false,
		RiskScore:     0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err = s.db.Exec(`
		INSERT INTO users (id, email, username, password_hash, kyc_status, kyc_level, 
			email_verified, risk_score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		user.ID, user.Email, user.Username, user.PasswordHash, user.KYCStatus,
		user.KYCLevel, user.EmailVerified, user.RiskScore, user.CreatedAt, user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *AuthService) Login(email, password, ipAddress string) (*User, string, error) {
	var user User
	err := s.db.QueryRow(`
		SELECT id, email, username, password_hash, kyc_status, kyc_level, 
			email_verified, phone_verified, mfa_enabled, failed_login_count, 
			locked_until, risk_score, created_at, updated_at, last_login_at
		FROM users WHERE email = $1`, email).Scan(
		&user.ID, &user.Email, &user.Username, &user.PasswordHash,
		&user.KYCStatus, &user.KYCLevel, &user.EmailVerified,
		&user.PhoneVerified, &user.MFAEnabled, &user.FailedLoginCount,
		&user.LockedUntil, &user.RiskScore, &user.CreatedAt,
		&user.UpdatedAt, &user.LastLoginAt)

	if err == sql.ErrNoRows {
		return nil, "", fmt.Errorf("invalid credentials")
	}
	if err != nil {
		return nil, "", fmt.Errorf("database error: %w", err)
	}

	// Check if locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, "", fmt.Errorf("account is locked until %s", user.LockedUntil)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// Increment failed login count
		s.db.Exec(`UPDATE users SET failed_login_count = failed_login_count + 1 WHERE id = $1`, user.ID)

		if user.FailedLoginCount+1 >= s.config.MaxLoginAttempts {
			lockoutUntil := time.Now().Add(s.config.LockoutDuration)
			s.db.Exec(`UPDATE users SET locked_until = $1 WHERE id = $2`, lockoutUntil, user.ID)
			return nil, "", fmt.Errorf("account locked due to too many failed attempts")
		}

		return nil, "", fmt.Errorf("invalid credentials")
	}

	// Reset failed login count and update last login
	now := time.Now()
	s.db.Exec(`UPDATE users SET failed_login_count = 0, last_login_at = $1 WHERE id = $2`, now, user.ID)
	user.LastLoginAt = &now

	// Generate token
	token, _, err := s.GenerateToken(&user, []string{})
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return &user, token, nil
}

// ============================================================================
// Wallet Service
// ============================================================================

type WalletService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewWalletService(db *sql.DB, redis *redis.Client) *WalletService {
	return &WalletService{db: db, redis: redis}
}

func (s *WalletService) CreateWallet(userID, whiteLabelID, walletType string, chainID uint64) (*Wallet, error) {
	// Generate address (in practice, would derive from HD wallet)
	address := generateAddress()

	wallet := &Wallet{
		ID:           uuid.New().String(),
		UserID:       userID,
		WhiteLabelID: whiteLabelID,
		WalletType:   walletType,
		Address:      address,
		ChainID:      chainID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err := s.db.Exec(`
		INSERT INTO wallets (id, user_id, white_label_id, wallet_type, address, chain_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		wallet.ID, wallet.UserID, wallet.WhiteLabelID, wallet.WalletType,
		wallet.Address, wallet.ChainID, wallet.IsActive, wallet.CreatedAt, wallet.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	return wallet, nil
}

func (s *WalletService) GetUserWallets(userID string) ([]Wallet, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, white_label_id, wallet_type, address, chain_id, is_active, created_at, updated_at
		FROM wallets WHERE user_id = $1 AND is_active = TRUE`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []Wallet
	for rows.Next() {
		var w Wallet
		if err := rows.Scan(&w.ID, &w.UserID, &w.WhiteLabelID, &w.WalletType,
			&w.Address, &w.ChainID, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}

	return wallets, nil
}

func (s *WalletService) GetBalance(address string, chainID uint64) (string, error) {
	// In practice, would query blockchain node or indexer
	// For now, return mock data
	return "0", nil
}

// ============================================================================
// Transaction Service
// ============================================================================

type TransactionService struct {
	db         *sql.DB
	redis      *redis.Client
	blockchain BlockchainConfig
}

func NewTransactionService(db *sql.DB, redis *redis.Client, blockchain BlockchainConfig) *TransactionService {
	return &TransactionService{db: db, redis: redis, blockchain: blockchain}
}

func (s *TransactionService) CreateTransaction(tx *Transaction) (*Transaction, error) {
	tx.ID = uuid.New().String()
	tx.Status = "pending"
	tx.CreatedAt = time.Now()

	_, err := s.db.Exec(`
		INSERT INTO transactions (id, hash, from_address, to_address, value, gas_price, gas_limit, chain_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		tx.ID, tx.Hash, tx.FromAddress, tx.ToAddress, tx.Value,
		tx.GasPrice, tx.GasLimit, tx.ChainID, tx.Status, tx.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return tx, nil
}

func (s *TransactionService) GetTransaction(hash string) (*Transaction, error) {
	var tx Transaction
	err := s.db.QueryRow(`
		SELECT id, hash, from_address, to_address, value, gas_price, gas_limit, 
			gas_used, chain_id, status, error_message, block_number, timestamp, created_at
		FROM transactions WHERE hash = $1`, hash).Scan(
		&tx.ID, &tx.Hash, &tx.FromAddress, &tx.ToAddress, &tx.Value,
		&tx.GasPrice, &tx.GasLimit, &tx.GasUsed, &tx.ChainID, &tx.Status,
		&tx.ErrorMessage, &tx.BlockNumber, &tx.Timestamp, &tx.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &tx, nil
}

func (s *TransactionService) GetUserTransactions(userID string, limit int) ([]Transaction, error) {
	rows, err := s.db.Query(`
		SELECT id, hash, from_address, to_address, value, gas_price, gas_limit,
			gas_used, chain_id, status, error_message, block_number, timestamp, created_at
		FROM transactions 
		WHERE from_address IN (SELECT address FROM wallets WHERE user_id = $1)
		   OR to_address IN (SELECT address FROM wallets WHERE user_id = $1)
		ORDER BY created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.Hash, &tx.FromAddress, &tx.ToAddress, &tx.Value,
			&tx.GasPrice, &tx.GasLimit, &tx.GasUsed, &tx.ChainID, &tx.Status,
			&tx.ErrorMessage, &tx.BlockNumber, &tx.Timestamp, &tx.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, nil
}

func (s *TransactionService) BroadcastTransaction(tx *Transaction) error {
	// In practice, would broadcast to blockchain node
	tx.Status = "broadcast"
	_, err := s.db.Exec(`
		UPDATE transactions SET status = $1 WHERE id = $2`, tx.Status, tx.ID)
	return err
}

// ============================================================================
// API Key Service
// ============================================================================

type APIKeyService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewAPIKeyService(db *sql.DB, redis *redis.Client) *APIKeyService {
	return &APIKeyService{db: db, redis: redis}
}

func (s *APIKeyService) CreateAPIKey(userID, whiteLabelID, name string, permissions []string) (*APIKey, string, error) {
	// Generate API key
	apiKey := generateAPIKey()
	keyHash := sha256.Sum256([]byte(apiKey))
	keyHashStr := hex.EncodeToString(keyHash[:])

	apiKeyRecord := &APIKey{
		ID:           uuid.New().String(),
		UserID:       userID,
		WhiteLabelID: whiteLabelID,
		KeyHash:      keyHashStr,
		Name:         name,
		Permissions:  permissions,
		RateLimit:    1000,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	_, err := s.db.Exec(`
		INSERT INTO api_keys (id, user_id, white_label_id, key_hash, name, permissions, rate_limit, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		apiKeyRecord.ID, apiKeyRecord.UserID, apiKeyRecord.WhiteLabelID,
		apiKeyRecord.KeyHash, apiKeyRecord.Name, pq.Array(apiKeyRecord.Permissions),
		apiKeyRecord.RateLimit, apiKeyRecord.IsActive, apiKeyRecord.CreatedAt)

	if err != nil {
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	return apiKeyRecord, apiKey, nil
}

func (s *APIKeyService) ValidateAPIKey(apiKey string) (*APIKey, error) {
	keyHash := sha256.Sum256([]byte(apiKey))
	keyHashStr := hex.EncodeToString(keyHash[:])

	var key APIKey
	err := s.db.QueryRow(`
		SELECT id, user_id, white_label_id, key_hash, name, permissions, rate_limit, 
			expires_at, last_used_at, is_active, created_at
		FROM api_keys WHERE key_hash = $1 AND is_active = TRUE`, keyHashStr).Scan(
		&key.ID, &key.UserID, &key.WhiteLabelID, &key.KeyHash, &key.Name,
		pq.Array(&key.Permissions), &key.RateLimit, &key.ExpiresAt,
		&key.LastUsedAt, &key.IsActive, &key.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Check expiration
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}

	// Update last used
	s.db.Exec(`UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, key.ID)

	return &key, nil
}

func (s *APIKeyService) RevokeAPIKey(keyID string) error {
	_, err := s.db.Exec(`UPDATE api_keys SET is_active = FALSE WHERE id = $1`, keyID)
	return err
}

// ============================================================================
// Audit Service
// ============================================================================

type AuditService struct {
	db *sql.DB
}

func NewAuditService(db *sql.DB) *AuditService {
	return &AuditService{db: db}
}

func (s *AuditService) LogAction(userID, action, resource, resourceID, details, ipAddress, userAgent string) error {
	_, err := s.db.Exec(`
		INSERT INTO audit_logs (id, user_id, action, resource, resource_id, details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		uuid.New().String(), userID, action, resource, resourceID, details, ipAddress, userAgent, time.Now())
	return err
}

func (s *AuditService) GetAuditLogs(userID string, limit int) ([]AuditLog, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, action, resource, resource_id, details, ip_address, user_agent, created_at
		FROM audit_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		if err := rows.Scan(&log.ID, &log.UserID, &log.Action, &log.Resource,
			&log.ResourceID, &log.Details, &log.IPAddress, &log.UserAgent, &log.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// ============================================================================
// Metrics
// ============================================================================

type Metrics struct {
	totalRequests atomic.Int64
	totalErrors   atomic.Int64
	activeUsers   atomic.Int64
	activeConns   atomic.Int64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) RecordRequest() {
	m.totalRequests.Add(1)
}

func (m *Metrics) RecordError() {
	m.totalErrors.Add(1)
}

func (m *Metrics) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_requests": m.totalRequests.Load(),
		"total_errors":   m.totalErrors.Load(),
		"active_users":   m.activeUsers.Load(),
		"active_conns":   m.activeConns.Load(),
	}
}

// ============================================================================
// WebSocket Hub
// ============================================================================

type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mutex      sync.RWMutex
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func (h *WebSocketHub) Broadcast(message []byte) {
	h.broadcast <- message
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateAddress() string {
	// In practice, would generate from HD wallet
	return "0x" + hex.EncodeToString(generateRandomBytes(20))
}

func generateAPIKey() string {
	return "tk_live_" + hex.EncodeToString(generateRandomBytes(32))
}

func generateRandomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *EnterpriseService) RegisterRoutes(r *gin.Engine) {
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "enterprise-api",
			"metrics": s.metrics.GetStats(),
		})
	})

	// Auth routes
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register", s.handleRegister)
		auth.POST("/login", s.handleLogin)
		auth.POST("/logout", s.handleLogout)
		auth.POST("/refresh", s.handleRefreshToken)
	}

	// Protected routes
	api := r.Group("/api/v1")
	api.Use(s.authMiddleware())
	{
		// User routes
		api.GET("/profile", s.handleGetProfile)
		api.PUT("/profile", s.handleUpdateProfile)

		// Wallet routes
		api.GET("/wallets", s.handleGetWallets)
		api.POST("/wallets", s.handleCreateWallet)
		api.GET("/wallets/:id/balance", s.handleGetBalance)

		// Transaction routes
		api.POST("/transactions", s.handleCreateTransaction)
		api.GET("/transactions", s.handleGetTransactions)
		api.GET("/transactions/:hash", s.handleGetTransaction)

		// API Key routes
		api.GET("/api-keys", s.handleGetAPIKeys)
		api.POST("/api-keys", s.handleCreateAPIKey)
		api.DELETE("/api-keys/:id", s.handleRevokeAPIKey)

		// White Label routes
		api.GET("/white-labels", s.handleGetWhiteLabels)
		api.POST("/white-labels", s.handleCreateWhiteLabel)
		api.PUT("/white-labels/:id", s.handleUpdateWhiteLabel)

		// Broker routes
		api.GET("/brokers", s.handleGetBrokers)
		api.POST("/brokers", s.handleCreateBroker)

		// Institution routes
		api.GET("/institutions", s.handleGetInstitutions)
		api.POST("/institutions", s.handleCreateInstitution)

		// Token routes
		api.GET("/tokens", s.handleGetTokens)
		api.POST("/tokens", s.handleCreateToken)

		// Audit logs
		api.GET("/audit-logs", s.handleGetAuditLogs)
	}

	// WebSocket
	r.GET("/ws", s.handleWebSocket)
}

func (s *EnterpriseService) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := s.authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("permissions", claims.Permissions)
		c.Next()
	}
}

// ============================================================================
// Handler Implementations
// ============================================================================

func (s *EnterpriseService) handleRegister(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := s.authService.RegisterUser(req.Email, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	token, _, _ := s.authService.GenerateToken(user, []string{})

	c.JSON(http.StatusCreated, gin.H{
		"user":  user,
		"token": token,
	})
}

func (s *EnterpriseService) handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := s.authService.Login(req.Email, req.Password, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  user,
		"token": token,
	})
}

func (s *EnterpriseService) handleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (s *EnterpriseService) handleRefreshToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"token": "new_token"})
}

func (s *EnterpriseService) handleGetProfile(c *gin.Context) {
	userID := c.GetString("userID")
	c.JSON(http.StatusOK, gin.H{"userId": userID})
}

func (s *EnterpriseService) handleUpdateProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

func (s *EnterpriseService) handleGetWallets(c *gin.Context) {
	userID := c.GetString("userID")
	wallets, err := s.walletSvc.GetUserWallets(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wallets": wallets})
}

func (s *EnterpriseService) handleCreateWallet(c *gin.Context) {
	userID := c.GetString("userID")

	var req struct {
		WalletType string `json:"walletType" binding:"required"`
		ChainID    uint64 `json:"chainId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := s.walletSvc.CreateWallet(userID, "", req.WalletType, req.ChainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"wallet": wallet})
}

func (s *EnterpriseService) handleGetBalance(c *gin.Context) {
	address := c.Param("address")
	chainID := c.Query("chainId")

	var chainIDUint uint64
	fmt.Sscanf(chainID, "%d", &chainIDUint)

	balance, err := s.walletSvc.GetBalance(address, chainIDUint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

func (s *EnterpriseService) handleCreateTransaction(c *gin.Context) {
	var tx Transaction
	if err := c.ShouldBindJSON(&tx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.txSvc.CreateTransaction(&tx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"transaction": result})
}

func (s *EnterpriseService) handleGetTransactions(c *gin.Context) {
	userID := c.GetString("userID")
	limit := 50

	txs, err := s.txSvc.GetUserTransactions(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

func (s *EnterpriseService) handleGetTransaction(c *gin.Context) {
	hash := c.Param("hash")

	tx, err := s.txSvc.GetTransaction(hash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tx == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction": tx})
}

func (s *EnterpriseService) handleGetAPIKeys(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"apiKeys": []interface{}{}})
}

func (s *EnterpriseService) handleCreateAPIKey(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Permissions []string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	key, apiKey, err := s.apiKeySvc.CreateAPIKey(userID, "", req.Name, req.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"apiKey": key, "key": apiKey})
}

func (s *EnterpriseService) handleRevokeAPIKey(c *gin.Context) {
	keyID := c.Param("id")
	if err := s.apiKeySvc.RevokeAPIKey(keyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}

func (s *EnterpriseService) handleGetWhiteLabels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"whiteLabels": []interface{}{}})
}

func (s *EnterpriseService) handleCreateWhiteLabel(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "white label created"})
}

func (s *EnterpriseService) handleUpdateWhiteLabel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "white label updated"})
}

func (s *EnterpriseService) handleGetBrokers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"brokers": []interface{}{}})
}

func (s *EnterpriseService) handleCreateBroker(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "broker created"})
}

func (s *EnterpriseService) handleGetInstitutions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"institutions": []interface{}{}})
}

func (s *EnterpriseService) handleCreateInstitution(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "institution created"})
}

func (s *EnterpriseService) handleGetTokens(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tokens": []interface{}{}})
}

func (s *EnterpriseService) handleCreateToken(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "token created"})
}

func (s *EnterpriseService) handleGetAuditLogs(c *gin.Context) {
	userID := c.GetString("userID")
	logs, _ := s.auditSvc.GetAuditLogs(userID, 100)
	c.JSON(http.StatusOK, gin.H{"auditLogs": logs})
}

func (s *EnterpriseService) handleWebSocket(c *gin.Context) {
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	s.wsHub.register <- conn

	go func() {
		defer func() {
			s.wsHub.unregister <- conn
			conn.Close()
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			// Handle message
			fmt.Printf("Received: %s\n", message)
		}
	}()
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()

	// Initialize service
	service, err := NewEnterpriseService(config)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Register routes
	service.RegisterRoutes(r)

	// Start WebSocket hub
	go service.wsHub.Run()

	// Create server
	srv := &http.Server{
		Addr:           ":" + config.Server.Port,
		Handler:        r,
		ReadTimeout:    config.Server.ReadTimeout,
		WriteTimeout:   config.Server.WriteTimeout,
		MaxHeaderBytes: config.Server.MaxHeaderBytes,
	}

	// Start server
	go func() {
		log.Printf("Enterprise API starting on port %s", config.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
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
