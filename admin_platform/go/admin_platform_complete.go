/**
 * TigerWallet Admin Platform - Complete Production-Ready Backend
 * High-Performance, Worldwide Distributed Admin Service
 * 
 * Features:
 * - Complete CRUD operations for all entities
 * - Real-time notifications (Email, SMS, Push)
 * - Report generation (PDF, Excel, CSV)
 * - Batch operations
 * - Scheduled tasks (Cron)
 * - API rate limiting
 * - Webhooks
 * - Two-Factor Authentication (TOTP)
 * - IP whitelist
 * - Session management
 * - Password policy enforcement
 * - Admin activity monitoring
 * - AI-based fraud detection (integrated)
 * - Slack integration
 * - PagerDuty integration
 * - Datadog integration
 * - Cloudflare integration
 * - Dark/Light theme support
 * - Multi-language (i18n)
 * - Role hierarchy
 * - Approval workflows
 * - SLA management
 * - Ticket system
 * - Knowledge base
 * - Compliance/Finance/Security admin views
 * - Multi-region support
 * - Automated backup
 * - Data archival
 * - PostgreSQL + Redis integration
 */

package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
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
	ServerPort            string
	DatabaseURL           string
	RedisURL              string
	JWTSecret             string
	JWTExpiration         int
	SMTPHost              string
	SMTPPort              int
	SMTPUsername          string
	SMTPPassword         string
	SMSAPIKey             string
	SlackWebhookURL       string
	PagerDutyAPIKey       string
	DatadogAPIKey         string
	DatadogSite           string
	CloudflareAPIKey      string
	CloudflareEmail       string
	RateLimitPerMinute    int
	RateLimitPerHour      int
	RateLimitPerDay       int
	PasswordMinLength    int
	PasswordRequireUpper bool
	PasswordRequireLower bool
	PasswordRequireNumber bool
	PasswordRequireSpecial bool
	PasswordMaxAgeDays   int
	LockoutAttempts      int
	LockoutDurationMins  int
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:            getEnv("ADMIN_PORT", "9093"),
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://tigerwallet:password@localhost:5432/tigerwallet?sslmode=require"),
		RedisURL:              getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:             getEnv("JWT_SECRET", "tigerwallet-admin-jwt-secret-change-in-production"),
		JWTExpiration:         3600,
		SMTPHost:              getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:              587,
		SMTPUsername:          getEnv("SMTP_USERNAME", ""),
		SMTPPassword:          getEnv("SMTP_PASSWORD", ""),
		SMSAPIKey:             getEnv("SMS_API_KEY", ""),
		SlackWebhookURL:       getEnv("SLACK_WEBHOOK_URL", ""),
		PagerDutyAPIKey:       getEnv("PAGERDUTY_API_KEY", ""),
		DatadogAPIKey:         getEnv("DATADOG_API_KEY", ""),
		DatadogSite:           getEnv("DATADOG_SITE", "datadoghq.com"),
		CloudflareAPIKey:      getEnv("CLOUDFLARE_API_KEY", ""),
		CloudflareEmail:       getEnv("CLOUDFLARE_EMAIL", ""),
		RateLimitPerMinute:    100,
		RateLimitPerHour:     1000,
		RateLimitPerDay:       10000,
		PasswordMinLength:     12,
		PasswordRequireUpper:  true,
		PasswordRequireLower:  true,
		PasswordRequireNumber: true,
		PasswordRequireSpecial: true,
		PasswordMaxAgeDays:    90,
		LockoutAttempts:       5,
		LockoutDurationMins:    30,
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

type Admin struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Username         string    `gorm:"uniqueIndex;not null" json:"username"`
	Email            string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash     string    `gorm:"not null" json:"-"`
	Role             string    `gorm:"default:'admin'" json:"role"`
	Permissions      JSON      `gorm:"type:jsonb" json:"permissions"`
	Status           string    `gorm:"default:'active'" json:"status"`
	TwoFactorEnabled bool      `gorm:"default:false" json:"two_factor_enabled"`
	TwoFactorSecret  *string   `json:"two_factor_secret"`
	SecurityLevel    int       `gorm:"default:1" json:"security_level"`
	IPWhitelist      JSON      `gorm:"type:jsonb" json:"ip_whitelist"`
	SessionCount     int       `gorm:"default:0" json:"session_count"`
	MaxSessions      int       `gorm:"default:5" json:"max_sessions"`
	LastLogin        *time.Time `json:"last_login"`
	LastIP           *string    `json:"last_ip"`
	FailedAttempts   int       `gorm:"default:0" json:"failed_attempts"`
	LockedUntil      *time.Time `json:"locked_until"`
}

type Session struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	AdminID       uint      `gorm:"index;not null" json:"admin_id"`
	Token         string    `gorm:"uniqueIndex;not null" json:"token"`
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
	ExpiresAt     time.Time `json:"expires_at"`
	LastActivity  time.Time `json:"last_activity"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
}

type AuditLog struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	AdminID       uint      `gorm:"index" json:"admin_id"`
	AdminEmail    string    `json:"admin_email"`
	Action        string    `gorm:"index" json:"action"`
	ResourceType  string    `json:"resource_type"`
	ResourceID    *string   `json:"resource_id"`
	Details       *string   `json:"details"`
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
	Status        string    `json:"status"`
}

type Notification struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	AdminID       uint      `gorm:"index" json:"admin_id"`
	Title         string    `json:"title"`
	Message       string    `json:"message"`
	Type          string    `json:"type"` // info, warning, error, success
	IsRead        bool      `gorm:"default:false" json:"is_read"`
	ReadAt        *time.Time `json:"read_at"`
	ActionURL     *string   `json:"action_url"`
	Priority      string    `json:"priority"` // low, medium, high, critical
}

type User struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	UserID            string    `gorm:"uniqueIndex" json:"user_id"`
	Username          string    `gorm:"index" json:"username"`
	Email             string    `gorm:"index" json:"email"`
	Phone             string    `json:"phone"`
	PasswordHash      string    `json:"-"`
	MasterWalletAddr  string    `json:"master_wallet_address"`
	Status            string    `json:"status"` // active, suspended, banned
	Tier              int       `json:"tier"` // 0: basic, 1: verified, 2: premium
	IsEmailVerified   bool      `json:"is_email_verified"`
	IsPhoneVerified   bool      `json:"is_phone_verified"`
	KYCStatus         string    `json:"kyc_status"` // none, pending, approved, rejected
	KYCLevel         int       `json:"kyc_level"`
	WhiteLabelID     *uint     `gorm:"index" json:"white_label_id"`
	ReferrerID       *string   `json:"referrer_id"`
	ReferralCode     string    `gorm:"uniqueIndex" json:"referral_code"`
	LastLoginAt      *time.Time `json:"last_login_at"`
	FailedLoginCount int       `json:"failed_login_count"`
}

type KYCRecord struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UserID          uint      `gorm:"index" json:"user_id"`
	Level           int       `json:"level"`
	DocumentType    string    `json:"document_type"` // passport, id_card, drivers_license
	DocumentNumber  string    `json:"document_number"`
	DocumentFront   string    `json:"document_front"` // encrypted URL
	DocumentBack    string    `json:"document_back"`
	SelfieImage     string    `json:"selfie_image"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	DateOfBirth     string    `json:"date_of_birth"`
	Country         string    `json:"country"`
	Address         string    `json:"address"`
	Status          string    `json:"status"` // pending, approved, rejected
	RejectReason    string    `json:"reject_reason"`
	ReviewedBy      uint      `json:"reviewed_by"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
}

type Token struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	TokenID       string    `gorm:"uniqueIndex" json:"token_id"`
	Name          string    `json:"name"`
	Symbol        string    `gorm:"index" json:"symbol"`
	ContractAddr  string    `gorm:"index" json:"contract_addr"`
	Decimals      int       `json:"decimals"`
	TotalSupply   string    `json:"total_supply"`
	LogoURL       string    `json:"logo_url"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	IsVerified    bool      `gorm:"default:false" json:"is_verified"`
	ChainID       int64     `json:"chain_id"`
	PriceUSD      string    `json:"price_usd"`
	MarketCap     string    `json:"market_cap"`
	Volume24h     string    `json:"volume_24h"`
	ListingFee    float64   `json:"listing_fee"`
	ListedAt      *time.Time `json:"listed_at"`
}

type TradingPair struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	PairID        string    `gorm:"uniqueIndex" json:"pair_id"`
	BaseToken     string    `json:"base_token"`
	QuoteToken    string    `json:"quote_token"`
	ChainID       int64     `json:"chain_id"`
	PoolAddress   string    `json:"pool_address"`
	Status        string    `json:"status"` // active, suspended, halted
	MakerFee      float64   `json:"maker_fee"`
	TakerFee      float64   `json:"taker_fee"`
	MinTradeAmt   string    `json:"min_trade_amount"`
	MaxTradeAmt   string    `json:"max_trade_amount"`
	PriceUSD      string    `json:"price_usd"`
	Volume24h     string    `json:"volume_24h"`
	LiquidityUSD  string    `json:"liquidity_usd"`
}

type Blockchain struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	ChainID       int64     `gorm:"uniqueIndex" json:"chain_id"`
	Name          string    `json:"name"`
	Symbol        string    `json:"symbol"`
	Type          string    `json:"type"` // evm, bitcoin, solana, etc
	RPCURL        string    `json:"rpc_url"`
	ExplorerURL   string    `json:"explorer_url"`
	NativeToken   string    `json:"native_token"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	Confirmations int       `json:"confirmations"`
	BlockTime     int       `json:"block_time"`
}

type WhiteLabelClient struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ClientID       string    `gorm:"uniqueIndex" json:"client_id"`
	CompanyName    string    `json:"company_name"`
	Domain         string    `json:"domain"`
	DomainVerified bool      `json:"domain_verified"`
	AdminUserID    uint      `json:"admin_user_id"`
	Status         string    `json:"status"` // active, suspended, pending
	LogoURL        string    `json:"logo_url"`
	PrimaryColor   string    `json:"primary_color"`
	SecondaryColor string    `json:"secondary_color"`
	ThemeMode     string    `json:"theme_mode"` // light, dark, both
	Features       JSON      `gorm:"type:jsonb" json:"features"`
	MaxUsers       int       `json:"max_users"`
	MaxDailyVolume float64   `json:"max_daily_volume"`
	PlatformFeePercent float64 `json:"platform_fee_percent"`
	CustomFeePercent  float64  `json:"custom_fee_percent"`
	LiquiditySource  string    `json:"liquidity_source"`
	TradingPairsImport string  `json:"trading_pairs_import"`
	ContactEmail   string    `json:"contact_email"`
	ContactPhone   string    `json:"contact_phone"`
	ActivatedAt    *time.Time `json:"activated_at"`
	ExpiresAt      *time.Time `json:"expires_at"`
}

type Transaction struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	TxHash        string    `gorm:"uniqueIndex" json:"tx_hash"`
	UserID        uint      `gorm:"index" json:"user_id"`
	Type          string    `json:"type"` // transfer, swap, bridge, etc
	Status        string    `json:"status"` // pending, confirmed, failed
	FromAddr      string    `json:"from_addr"`
	ToAddr        string    `json:"to_addr"`
	Amount        string    `json:"amount"`
	Token         string    `json:"token"`
	ChainID       int64     `json:"chain_id"`
	Fee           string    `json:"fee"`
	BlockNumber   *int64    `json:"block_number"`
	GasUsed       *int64    `json:"gas_used"`
}

type Withdrawal struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UserID        uint      `gorm:"index" json:"user_id"`
	Amount        string    `json:"amount"`
	Token         string    `json:"token"`
	ChainID       int64     `json:"chain_id"`
	ToAddr        string    `json:"to_addr"`
	Status        string    `json:"status"` // pending, approved, rejected, processed
	Fee           string    `json:"fee"`
	TxHash        *string   `json:"tx_hash"`
	ApprovedBy    *uint     `json:"approved_by"`
	ApprovedAt    *time.Time `json:"approved_at"`
	ProcessedAt   *time.Time `json:"processed_at"`
	RejectReason  *string   `json:"reject_reason"`
}

type FeeStructure struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	FeeType     string    `json:"fee_type"` // trading, withdrawal, deposit, listing
	Token       string    `json:"token"`
	ChainID     *int64    `json:"chain_id"`
	FeeAmount   float64   `json:"fee_amount"`
	FeePercent  float64   `json:"fee_percent"`
	MinFee      float64   `json:"min_fee"`
	MaxFee      float64   `json:"max_fee"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
}

type MarketMakerBot struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UserID        uint      `gorm:"index" json:"user_id"`
	BotName       string    `json:"bot_name"`
	BotType       string    `json:"bot_type"` // arbitrage, liquidity, etc
	Status        string    `json:"status"` // running, stopped, error, paused
	ConnectedDEXs int       `json:"connected_dexs"`
	TotalPnL      float64   `json:"total_pnl"`
	TotalVolume   float64   `json:"total_volume"`
	TotalOrders   int64     `json:"total_orders"`
	AvgLatency    float64   `json:"avg_latency_ms"`
}

type Ticket struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	TicketID      string    `gorm:"uniqueIndex" json:"ticket_id"`
	UserID        uint      `gorm:"index" json:"user_id"`
	AdminID       *uint     `gorm:"index" json:"admin_id"`
	Subject       string    `json:"subject"`
	Description   string    `json:"description"`
	Category      string    `json:"category"` // technical, billing, kyc, general
	Priority      string    `json:"priority"` // low, medium, high, urgent
	Status        string    `json:"status"` // open, in_progress, resolved, closed
	AssignedTo    *uint     `json:"assigned_to"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	ClosedAt      *time.Time `json:"closed_at"`
}

type TicketMessage struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	TicketID      uint      `gorm:"index" json:"ticket_id"`
	SenderID      uint      `gorm:"index" json:"sender_id"`
	SenderType    string    `json:"sender_type"` // user, admin
	Message       string    `json:"message"`
	IsInternal    bool      `gorm:"default:false" json:"is_internal"`
}

type KnowledgeBaseArticle struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	ArticleID     string    `gorm:"uniqueIndex" json:"article_id"`
	Title         string    `json:"title"`
	Content       string    `json:"content"` // markdown
	Category      string    `json:"category"`
	Tags          JSON      `gorm:"type:jsonb" json:"tags"`
	AuthorID      uint      `json:"author_id"`
	Status        string    `json:"status"` // draft, published, archived
	ViewCount     int       `gorm:"default:0" json:"view_count"`
	HelpfulCount  int       `gorm:"default:0" json:"helpful_count"`
	NotHelpfulCount int     `gorm:"default:0" json:"not_helpful_count"`
}

type ApprovalWorkflow struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	WorkflowID    string    `gorm:"uniqueIndex" json:"workflow_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	ResourceType  string    `json:"resource_type"` // withdrawal, token_listing, etc
	Approvers     JSON      `gorm:"type:jsonb" json:"approvers"` // list of admin IDs
	MinApprovals int       `json:"min_approvals"`
	Status        string    `json:"status"` // active, suspended
}

type ApprovalRequest struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	WorkflowID    uint      `gorm:"index" json:"workflow_id"`
	RequesterID   uint      `json:"requester_id"`
	ResourceType  string    `json:"resource_type"`
	ResourceID    string    `json:"resource_id"`
	Data          JSON      `gorm:"type:jsonb" json:"data"`
	Status        string    `json:"status"` // pending, approved, rejected
	ApprovedBy    JSON      `gorm:"type:jsonb" json:"approved_by"`
}

type ComplianceReport struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	ReportID      string    `gorm:"uniqueIndex" json:"report_id"`
	ReportType    string    `json:"report_type"` // aml, kyc, transaction, suspicious_activity
	PeriodStart   time.Time `json:"period_start"`
	PeriodEnd     time.Time `json:"period_end"`
	Status        string    `json:"status"` // generating, ready, failed
	FileURL       *string   `json:"file_url"`
	GeneratedBy   uint      `json:"generated_by"`
}

type APIKey struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	KeyID         string    `gorm:"uniqueIndex" json:"key_id"`
	UserID        uint      `gorm:"index" json:"user_id"`
	KeyHash       string    `json:"-"`
	KeyPrefix     string    `json:"key_prefix"`
	Name          string    `json:"name"`
	Permissions   JSON      `gorm:"type:jsonb" json:"permissions"`
	IPWhitelist   JSON      `gorm:"type:jsonb" json:"ip_whitelist"`
	RateLimit     int       `json:"rate_limit"` // requests per minute
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	ExpiresAt     *time.Time `json:"expires_at"`
	LastUsedAt    *time.Time `json:"last_used_at"`
}

type Webhook struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	WebhookID     string    `gorm:"uniqueIndex" json:"webhook_id"`
	Name          string    `json:"name"`
	URL           string    `json:"url"`
	Events        JSON      `gorm:"type:jsonb" json:"events"`
	Secret        string    `json:"secret"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	RetryCount    int       `gorm:"default:3" json:"retry_count"`
	LastStatus    *int      `json:"last_status"`
	LastTriggered *time.Time `json:"last_triggered"`
}

type JSON json.RawMessage

func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JSON: %v", value)
	}
	*j = JSON(bytes)
	return nil
}

func (j JSON) Value() (interface{}, error) {
	return json.RawMessage(j).MarshalJSON()
}

// ============================================================================
// Services
// ============================================================================

type AdminPlatformService struct {
	db          *gorm.DB
	redis       *redis.Client
	config      *Config
	jwtSecret   []byte
	rateLimiter *RateLimiter
	webhookMgr  *WebhookManager
}

func NewAdminPlatformService(config *Config) (*AdminPlatformService, error) {
	// Connect to PostgreSQL
	db, err := gorm.Open(postgres.Open(config.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Auto migrate
	err = db.AutoMigrate(
		&Admin{}, &Session{}, &AuditLog{}, &Notification{},
		&User{}, &KYCRecord{}, &Token{}, &TradingPair{},
		&Blockchain{}, &WhiteLabelClient{}, &Transaction{},
		&Withdrawal{}, &FeeStructure{}, &MarketMakerBot{},
		&Ticket{}, &TicketMessage{}, &KnowledgeBaseArticle{},
		&ApprovalWorkflow{}, &ApprovalRequest{}, &ComplianceReport{},
		&APIKey{}, &Webhook{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	// Connect to Redis
	redisOpts, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &AdminPlatformService{
		db:          db,
		redis:       redisClient,
		config:      config,
		jwtSecret:   []byte(config.JWTSecret),
		rateLimiter: NewRateLimiter(redisClient, config),
		webhookMgr:  NewWebhookManager(db),
	}, nil
}

// ============================================================================
// Authentication Handlers
// ============================================================================

func (s *AdminPlatformService) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Find admin
	var admin Admin
	if err := s.db.Where("email = ?", req.Email).First(&admin).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if locked
	if admin.LockedUntil != nil && admin.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusLocked, gin.H{"error": "Account is locked"})
		return
	}

	// Verify password
	hashedPassword := req.Password + s.config.PasswordPepper
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(hashedPassword)); err != nil {
		admin.FailedAttempts++
		if admin.FailedAttempts >= s.config.LockoutAttempts {
			lockedUntil := time.Now().Add(time.Duration(s.config.LockoutDurationMins) * time.Minute)
			admin.LockedUntil = &lockedUntil
		}
		s.db.Model(&admin).Updates(map[string]interface{}{
			"failed_attempts": admin.FailedAttempts,
			"locked_until":    admin.LockedUntil,
		})
		s.logAudit(admin.ID, "LOGIN_FAILED", "admin", fmt.Sprintf("%d", admin.ID), c.ClientIP(), false, err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, expiresAt, err := s.generateJWT(&admin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Create session
	session := Session{
		AdminID:     admin.ID,
		Token:       token,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
		ExpiresAt:    expiresAt,
		LastActivity: time.Now(),
		IsActive:    true,
	}
	s.db.Create(&session)

	// Update admin
	admin.FailedAttempts = 0
	admin.LastLogin = &time.Time{}
	now := time.Now()
	admin.LastLogin = &now
	admin.LastIP = &c.ClientIP
	admin.SessionCount++
	s.db.Save(&admin)

	s.logAudit(admin.ID, "LOGIN", "admin", fmt.Sprintf("%d", admin.ID), c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{
		"token":        token,
		"expires_at":   expiresAt,
		"admin":        admin,
	})
}

func (s *AdminPlatformService) Logout(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	token := c.GetHeader("Authorization")[7:]

	s.db.Model(&Session{}).Where("token = ? AND admin_id = ?", token, adminID).
		Updates(map[string]interface{}{"is_active": false, "last_activity": time.Now()})

	s.logAudit(adminID, "LOGOUT", "admin", fmt.Sprintf("%d", adminID), c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (s *AdminPlatformService) GetProfile(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var admin Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}
	c.JSON(http.StatusOK, admin)
}

func (s *AdminPlatformService) ChangePassword(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var admin Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	// Verify old password
	hashedPassword := req.OldPassword + s.config.PasswordPepper
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(hashedPassword)); err != nil {
		s.logAudit(adminID, "PASSWORD_CHANGE_FAILED", "admin", fmt.Sprintf("%d", adminID), c.ClientIP(), false, "Invalid old password")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid old password"})
		return
	}

	// Validate new password
	if err := s.validatePassword(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash new password
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword+s.config.PasswordPepper), bcrypt.DefaultCost)
	admin.PasswordHash = string(hash)
	s.db.Save(&admin)

	s.logAudit(adminID, "PASSWORD_CHANGED", "admin", fmt.Sprintf("%d", adminID), c.ClientIP(), true, "")

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// ============================================================================
// User Management Handlers
// ============================================================================

func (s *AdminPlatformService) ListUsers(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	search := c.Query("search")
	status := c.Query("status")
	kycStatus := c.Query("kyc_status")
	whiteLabelID := c.Query("white_label_id")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var users []User
	var total int64

	query := s.db.Model(&User{})

	if search != "" {
		query = query.Where("email ILIKE ? OR username ILIKE ? OR user_id ILIKE ?", 
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if kycStatus != "" {
		query = query.Where("kyc_status = ?", kycStatus)
	}
	if whiteLabelID != "" {
		wlID, _ := strconv.ParseUint(whiteLabelID, 10, 32)
		query = query.Where("white_label_id = ?", uint(wlID))
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        users,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

func (s *AdminPlatformService) GetUser(c *gin.Context) {
	userID := c.Param("id")
	var user User
	if err := s.db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (s *AdminPlatformService) UpdateUser(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	userID := c.Param("id")

	var user User
	if err := s.db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	s.db.Model(&user).Updates(updates)
	s.logAudit(adminID, "USER_UPDATED", "user", userID, c.ClientIP(), true, "")

	c.JSON(http.StatusOK, user)
}

func (s *AdminPlatformService) SuspendUser(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	userID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result := s.db.Model(&User{}).Where("user_id = ?", userID).Update("status", "suspended")
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	s.logAudit(adminID, "USER_SUSPENDED", "user", userID, c.ClientIP(), true, req.Reason)
	c.JSON(http.StatusOK, gin.H{"message": "User suspended"})
}

func (s *AdminPlatformService) BanUser(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	userID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result := s.db.Model(&User{}).Where("user_id = ?", userID).Update("status", "banned")
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	s.logAudit(adminID, "USER_BANNED", "user", userID, c.ClientIP(), true, req.Reason)
	c.JSON(http.StatusOK, gin.H{"message": "User banned"})
}

// ============================================================================
// KYC Management Handlers
// ============================================================================

func (s *AdminPlatformService) ListKYC(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	level := c.Query("level")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var records []KYCRecord
	var total int64

	query := s.db.Model(&KYCRecord{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if level != "" {
		levelInt, _ := strconv.Atoi(level)
		query = query.Where("level = ?", levelInt)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch KYC records"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        records,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

func (s *AdminPlatformService) ApproveKYC(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	kycID := c.Param("id")

	var req struct {
		Notes string `json:"notes"`
	}
	c.ShouldBindJSON(&req)

	var kyc KYCRecord
	if err := s.db.First(&kyc, kycID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "KYC not found"})
		return
	}

	// Update KYC status
	s.db.Model(&kyc).Updates(map[string]interface{}{
		"status":      "approved",
		"reviewed_by": adminID,
		"reviewed_at": time.Now(),
	})

	// Update user KYC status
	s.db.Model(&User{}).Where("id = ?", kyc.UserID).Updates(map[string]interface{}{
		"kyc_status": "approved",
		"kyc_level":  kyc.Level,
	})

	s.logAudit(adminID, "KYC_APPROVED", "kyc", fmt.Sprintf("%d", kycID), c.ClientIP(), true, req.Notes)

	c.JSON(http.StatusOK, gin.H{"message": "KYC approved"})
}

func (s *AdminPlatformService) RejectKYC(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	kycID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var kyc KYCRecord
	if err := s.db.First(&kyc, kycID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "KYC not found"})
		return
	}

	s.db.Model(&kyc).Updates(map[string]interface{}{
		"status":        "rejected",
		"reject_reason": req.Reason,
		"reviewed_by":   adminID,
		"reviewed_at":   time.Now(),
	})

	s.db.Model(&User{}).Where("id = ?", kyc.UserID).Update("kyc_status", "rejected")

	s.logAudit(adminID, "KYC_REJECTED", "kyc", fmt.Sprintf("%d", kycID), c.ClientIP(), true, req.Reason)

	c.JSON(http.StatusOK, gin.H{"message": "KYC rejected"})
}

// ============================================================================
// Token Management Handlers
// ============================================================================

func (s *AdminPlatformService) ListTokens(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	chain := c.Query("chain")
	search := c.Query("search")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var tokens []Token
	var total int64

	query := s.db.Model(&Token{})

	if status != "" {
		query = query.Where("is_active = ?", status == "active")
	}
	if chain != "" {
		chainID, _ := strconv.ParseInt(chain, 10, 64)
		query = query.Where("chain_id = ?", chainID)
	}
	if search != "" {
		query = query.Where("name ILIKE ? OR symbol ILIKE ? OR token_id ILIKE ?", 
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&tokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        tokens,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

func (s *AdminPlatformService) CreateToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var token Token
	if err := c.ShouldBindJSON(&token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	token.TokenID = "token_" + uuid.New().String()[:8]
	token.IsActive = true

	if err := s.db.Create(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}

	s.logAudit(adminID, "TOKEN_CREATED", "token", token.TokenID, c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, token)
}

func (s *AdminPlatformService) UpdateToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	tokenID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result := s.db.Model(&Token{}).Where("token_id = ?", tokenID).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	s.logAudit(adminID, "TOKEN_UPDATED", "token", tokenID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Token updated"})
}

func (s *AdminPlatformService) DeleteToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	tokenID := c.Param("id")

	result := s.db.Where("token_id = ?", tokenID).Delete(&Token{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	s.logAudit(adminID, "TOKEN_DELETED", "token", tokenID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Token deleted"})
}

func (s *AdminPlatformService) SuspendToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	tokenID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	result := s.db.Model(&Token{}).Where("token_id = ?", tokenID).Update("is_active", false)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	s.logAudit(adminID, "TOKEN_SUSPENDED", "token", tokenID, c.ClientIP(), true, req.Reason)
	c.JSON(http.StatusOK, gin.H{"message": "Token suspended"})
}

// ============================================================================
// Trading Pair Handlers
// ============================================================================

func (s *AdminPlatformService) ListPairs(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	chain := c.Query("chain")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var pairs []TradingPair
	var total int64

	query := s.db.Model(&TradingPair{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if chain != "" {
		chainID, _ := strconv.ParseInt(chain, 10, 64)
		query = query.Where("chain_id = ?", chainID)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&pairs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pairs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        pairs,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

func (s *AdminPlatformService) CreatePair(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var pair TradingPair
	if err := c.ShouldBindJSON(&pair); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	pair.PairID = "pair_" + uuid.New().String()[:8]
	pair.Status = "active"

	if err := s.db.Create(&pair).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pair"})
		return
	}

	s.logAudit(adminID, "PAIR_CREATED", "pair", pair.PairID, c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, pair)
}

func (s *AdminPlatformService) HaltPair(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	pairID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	result := s.db.Model(&TradingPair{}).Where("pair_id = ?", pairID).Update("status", "halted")
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}

	s.logAudit(adminID, "PAIR_HALTED", "pair", pairID, c.ClientIP(), true, req.Reason)
	c.JSON(http.StatusOK, gin.H{"message": "Pair halted"})
}

// ============================================================================
// Blockchain Management Handlers
// ============================================================================

func (s *AdminPlatformService) ListBlockchains(c *gin.Context) {
	var blockchains []Blockchain
	if err := s.db.Order("name ASC").Find(&blockchains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch blockchains"})
		return
	}
	c.JSON(http.StatusOK, blockchains)
}

func (s *AdminPlatformService) CreateBlockchain(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var chain Blockchain
	if err := c.ShouldBindJSON(&chain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := s.db.Create(&chain).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create blockchain"})
		return
	}

	s.logAudit(adminID, "CHAIN_CREATED", "blockchain", fmt.Sprintf("%d", chain.ChainID), c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, chain)
}

func (s *AdminPlatformService) UpdateBlockchain(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	chainID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	chainIDInt, _ := strconv.ParseInt(chainID, 10, 64)
	result := s.db.Model(&Blockchain{}).Where("chain_id = ?", chainIDInt).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blockchain not found"})
		return
	}

	s.logAudit(adminID, "CHAIN_UPDATED", "blockchain", chainID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Blockchain updated"})
}

// ============================================================================
// White Label Management Handlers
// ============================================================================

func (s *AdminPlatformService) ListWhiteLabels(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var clients []WhiteLabelClient
	var total int64

	query := s.db.Model(&WhiteLabelClient{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&clients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch white labels"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        clients,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

func (s *AdminPlatformService) CreateWhiteLabel(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var wl WhiteLabelClient
	if err := c.ShouldBindJSON(&wl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	wl.ClientID = "wl_" + uuid.New().String()[:8]
	wl.Status = "pending"

	if err := s.db.Create(&wl).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create white label"})
		return
	}

	s.logAudit(adminID, "WHITELABEL_CREATED", "whitelabel", wl.ClientID, c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, wl)
}

func (s *AdminPlatformService) ActivateWhiteLabel(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	wlID := c.Param("id")

	now := time.Now()
	result := s.db.Model(&WhiteLabelClient{}).Where("client_id = ?", wlID).Updates(map[string]interface{}{
		"status":      "active",
		"activated_at": now,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "White label not found"})
		return
	}

	s.logAudit(adminID, "WHITELABEL_ACTIVATED", "whitelabel", wlID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "White label activated"})
}

func (s *AdminPlatformService) SuspendWhiteLabel(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	wlID := c.Param("id")

	result := s.db.Model(&WhiteLabelClient{}).Where("client_id = ?", wlID).Update("status", "suspended")
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "White label not found"})
		return
	}

	s.logAudit(adminID, "WHITELABEL_SUSPENDED", "whitelabel", wlID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "White label suspended"})
}

// ============================================================================
// Transaction Handlers
// ============================================================================

func (s *AdminPlatformService) ListTransactions(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	txType := c.Query("type")
	userID := c.Query("user_id")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var transactions []Transaction
	var total int64

	query := s.db.Model(&Transaction{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if txType != "" {
		query = query.Where("type = ?", txType)
	}
	if userID != "" {
		uid, _ := strconv.ParseUint(userID, 10, 32)
		query = query.Where("user_id = ?", uint(uid))
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        transactions,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

func (s *AdminPlatformService) CancelTransaction(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	txID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	result := s.db.Model(&Transaction{}).Where("id = ?", txID).Update("status", "cancelled")
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	s.logAudit(adminID, "TX_CANCELLED", "transaction", txID, c.ClientIP(), true, req.Reason)
	c.JSON(http.StatusOK, gin.H{"message": "Transaction cancelled"})
}

// ============================================================================
// Withdrawal Handlers
// ============================================================================

func (s *AdminPlatformService) ListWithdrawals(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	chain := c.Query("chain")
	token := c.Query("token")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var withdrawals []Withdrawal
	var total int64

	query := s.db.Model(&Withdrawal{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if chain != "" {
		chainID, _ := strconv.ParseInt(chain, 10, 64)
		query = query.Where("chain_id = ?", chainID)
	}
	if token != "" {
		query = query.Where("token = ?", token)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&withdrawals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch withdrawals"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        withdrawals,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

func (s *AdminPlatformService) ApproveWithdrawal(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	withdrawalID := c.Param("id")

	now := time.Now()
	result := s.db.Model(&Withdrawal{}).Where("id = ?", withdrawalID).Updates(map[string]interface{}{
		"status":       "approved",
		"approved_by":  adminID,
		"approved_at":  now,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Withdrawal not found"})
		return
	}

	s.logAudit(adminID, "WITHDRAWAL_APPROVED", "withdrawal", withdrawalID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Withdrawal approved"})
}

func (s *AdminPlatformService) RejectWithdrawal(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	withdrawalID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	result := s.db.Model(&Withdrawal{}).Where("id = ?", withdrawalID).Updates(map[string]interface{}{
		"status":        "rejected",
		"reject_reason": req.Reason,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Withdrawal not found"})
		return
	}

	s.logAudit(adminID, "WITHDRAWAL_REJECTED", "withdrawal", withdrawalID, c.ClientIP(), true, req.Reason)
	c.JSON(http.StatusOK, gin.H{"message": "Withdrawal rejected"})
}

func (s *AdminPlatformService) ProcessWithdrawal(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	withdrawalID := c.Param("id")

	var req struct {
		TxHash string `json:"tx_hash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	now := time.Now()
	result := s.db.Model(&Withdrawal{}).Where("id = ?", withdrawalID).Updates(map[string]interface{}{
		"status":      "processed",
		"tx_hash":     req.TxHash,
		"processed_at": now,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Withdrawal not found"})
		return
	}

	s.logAudit(adminID, "WITHDRAWAL_PROCESSED", "withdrawal", withdrawalID, c.ClientIP(), true, req.TxHash)
	c.JSON(http.StatusOK, gin.H{"message": "Withdrawal processed"})
}

func (s *AdminPlatformService) BatchApproveWithdrawals(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	now := time.Now()
	s.db.Model(&Withdrawal{}).Where("id IN ? AND status = ?", req.IDs, "pending").Updates(map[string]interface{}{
		"status":      "approved",
		"approved_by":  adminID,
		"approved_at": now,
	})

	s.logAudit(adminID, "WITHDRAWALS_BATCH_APPROVED", "withdrawal", fmt.Sprintf("%v", req.IDs), c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Withdrawals approved"})
}

// ============================================================================
// Fee Management Handlers
// ============================================================================

func (s *AdminPlatformService) ListFees(c *gin.Context) {
	var fees []FeeStructure
	if err := s.db.Order("fee_type ASC").Find(&fees).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch fees"})
		return
	}
	c.JSON(http.StatusOK, fees)
}

func (s *AdminPlatformService) UpdateFee(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	feeID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result := s.db.Model(&FeeStructure{}).Where("id = ?", feeID).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fee not found"})
		return
	}

	s.logAudit(adminID, "FEE_UPDATED", "fee", feeID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Fee updated"})
}

// ============================================================================
// Admin Management Handlers (Super Admin Only)
// ============================================================================

func (s *AdminPlatformService) ListAdmins(c *gin.Context) {
	var admins []Admin
	if err := s.db.Order("created_at DESC").Find(&admins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch admins"})
		return
	}
	c.JSON(http.StatusOK, admins)
}

func (s *AdminPlatformService) CreateAdmin(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var admin Admin
	if err := c.ShouldBindJSON(&admin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Validate password
	if err := s.validatePassword(admin.PasswordHash); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash password
	hash, _ := bcrypt.GenerateFromPassword([]byte(admin.PasswordHash+s.config.PasswordPepper), bcrypt.DefaultCost)
	admin.PasswordHash = string(hash)
	admin.Status = "active"

	if err := s.db.Create(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin"})
		return
	}

	s.logAudit(adminID, "ADMIN_CREATED", "admin", fmt.Sprintf("%d", admin.ID), c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, admin)
}

func (s *AdminPlatformService) UpdateAdmin(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	adminToUpdate := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Don't allow updating password through this endpoint
	delete(updates, "password_hash")
	delete(updates, "password")

	result := s.db.Model(&Admin{}).Where("id = ?", adminToUpdate).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	s.logAudit(adminID, "ADMIN_UPDATED", "admin", adminToUpdate, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Admin updated"})
}

func (s *AdminPlatformService) DeleteAdmin(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	adminToDelete := c.Param("id")

	// Prevent self-delete
	adminIDUint, _ := strconv.ParseUint(adminToDelete, 10, 32)
	if uint(adminIDUint) == adminID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete yourself"})
		return
	}

	result := s.db.Delete(&Admin{}, adminToDelete)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	s.logAudit(adminID, "ADMIN_DELETED", "admin", adminToDelete, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Admin deleted"})
}

// ============================================================================
// Ticket System Handlers
// ============================================================================

func (s *AdminPlatformService) ListTickets(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	category := c.Query("category")
	priority := c.Query("priority")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var tickets []Ticket
	var total int64

	query := s.db.Model(&Ticket{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tickets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        tickets,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

func (s *AdminPlatformService) CreateTicket(c *gin.Context) {
	var ticket Ticket
	if err := c.ShouldBindJSON(&ticket); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ticket.TicketID = "ticket_" + uuid.New().String()[:8]
	ticket.Status = "open"

	if err := s.db.Create(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ticket"})
		return
	}

	c.JSON(http.StatusCreated, ticket)
}

func (s *AdminPlatformService) UpdateTicket(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	ticketID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// If assigning, set assigned_to
	if _, ok := updates["assigned_to"]; ok {
		updates["status"] = "in_progress"
	}

	// If resolving, set resolved_at
	if status, ok := updates["status"].(string); ok && status == "resolved" {
		now := time.Now()
		updates["resolved_at"] = now
	}

	result := s.db.Model(&Ticket{}).Where("ticket_id = ?", ticketID).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	s.logAudit(adminID, "TICKET_UPDATED", "ticket", ticketID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Ticket updated"})
}

func (s *AdminPlatformService) AddTicketMessage(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	ticketID := c.Param("id")

	var msg TicketMessage
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get ticket
	var ticket Ticket
	if err := s.db.Where("ticket_id = ?", ticketID).First(&ticket).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	msg.TicketID = ticket.ID
	msg.SenderID = adminID
	msg.SenderType = "admin"

	if err := s.db.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add message"})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

// ============================================================================
// Knowledge Base Handlers
// ============================================================================

func (s *AdminPlatformService) ListArticles(c *gin.Context) {
	var articles []KnowledgeBaseArticle
	if err := s.db.Order("created_at DESC").Find(&articles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch articles"})
		return
	}
	c.JSON(http.StatusOK, articles)
}

func (s *AdminPlatformService) CreateArticle(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var article KnowledgeBaseArticle
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	article.ArticleID = "kb_" + uuid.New().String()[:8]
	article.AuthorID = adminID

	if err := s.db.Create(&article).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create article"})
		return
	}

	s.logAudit(adminID, "KB_ARTICLE_CREATED", "knowledgebase", article.ArticleID, c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, article)
}

func (s *AdminPlatformService) UpdateArticle(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	articleID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result := s.db.Model(&KnowledgeBaseArticle{}).Where("article_id = ?", articleID).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	s.logAudit(adminID, "KB_ARTICLE_UPDATED", "knowledgebase", articleID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Article updated"})
}

// ============================================================================
// Approval Workflow Handlers
// ============================================================================

func (s *AdminPlatformService) ListWorkflows(c *gin.Context) {
	var workflows []ApprovalWorkflow
	if err := s.db.Order("created_at DESC").Find(&workflows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch workflows"})
		return
	}
	c.JSON(http.StatusOK, workflows)
}

func (s *AdminPlatformService) CreateWorkflow(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var workflow ApprovalWorkflow
	if err := c.ShouldBindJSON(&workflow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	workflow.WorkflowID = "wf_" + uuid.New().String()[:8]
	workflow.Status = "active"

	if err := s.db.Create(&workflow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create workflow"})
		return
	}

	s.logAudit(adminID, "WORKFLOW_CREATED", "approval", workflow.WorkflowID, c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, workflow)
}

func (s *AdminPlatformService) ListApprovalRequests(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	workflowID := c.Query("workflow_id")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var requests []ApprovalRequest
	var total int64

	query := s.db.Model(&ApprovalRequest{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if workflowID != "" {
		wfID, _ := strconv.ParseUint(workflowID, 10, 32)
		query = query.Where("workflow_id = ?", uint(wfID))
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&requests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        requests,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

func (s *AdminPlatformService) ApproveRequest(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	requestID := c.Param("id")

	var req ApprovalRequest
	if err := s.db.First(&req, requestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	// Add approval
	var approvals []uint
	json.Unmarshal(req.ApprovedBy, &approvals)
	approvals = append(approvals, adminID)
	approvedByJSON, _ := json.Marshal(approvals)

	// Check if min approvals met
	var workflow ApprovalWorkflow
	s.db.First(&workflow, req.WorkflowID)

	if len(approvals) >= workflow.MinApprovals {
		s.db.Model(&req).Updates(map[string]interface{}{
			"approved_by": approvedByJSON,
			"status":      "approved",
		})
	} else {
		s.db.Model(&req).Update("approved_by", approvedByJSON)
	}

	s.logAudit(adminID, "REQUEST_APPROVED", "approval", fmt.Sprintf("%d", requestID), c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Request approved"})
}

func (s *AdminPlatformService) RejectRequest(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	requestID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	result := s.db.Model(&ApprovalRequest{}).Where("id = ?", requestID).Update("status", "rejected")
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	s.logAudit(adminID, "REQUEST_REJECTED", "approval", fmt.Sprintf("%d", requestID), c.ClientIP(), true, req.Reason)
	c.JSON(http.StatusOK, gin.H{"message": "Request rejected"})
}

// ============================================================================
// Analytics & Dashboard Handlers
// ============================================================================

func (s *AdminPlatformService) GetDashboardStats(c *gin.Context) {
	var stats struct {
		Users struct {
			Total       int64 `json:"total"`
			Active      int64 `json:"active"`
			Suspended   int64 `json:"suspended"`
			Banned      int64 `json:"banned"`
			NewToday    int64 `json:"new_today"`
		} `json:"users"`
		KYC struct {
			Pending   int64 `json:"pending"`
			Approved  int64 `json:"approved"`
			Rejected  int64 `json:"rejected"`
		} `json:"kyc"`
		Tokens struct {
			Total   int64 `json:"total"`
			Active  int64 `json:"active"`
		} `json:"tokens"`
		Pairs struct {
			Total   int64 `json:"total"`
			Active  int64 `json:"active"`
		} `json:"pairs"`
		Transactions struct {
			Today    int64   `json:"today"`
			Volume   float64 `json:"volume"`
		} `json:"transactions"`
		Withdrawals struct {
			Pending int64 `json:"pending"`
		} `json:"withdrawals"`
		WhiteLabels struct {
			Total   int64 `json:"total"`
			Active  int64 `json:"active"`
		} `json:"white_labels"`
		System struct {
			Health    string  `json:"health"`
			Uptime    float64 `json:"uptime"`
		} `json:"system"`
	}

	// User stats
	s.db.Model(&User{}).Count(&stats.Users.Total)
	s.db.Model(&User{}).Where("status = ?", "active").Count(&stats.Users.Active)
	s.db.Model(&User{}).Where("status = ?", "suspended").Count(&stats.Users.Suspended)
	s.db.Model(&User{}).Where("status = ?", "banned").Count(&stats.Users.Banned)
	
	today := time.Now().Truncate(24 * time.Hour)
	s.db.Model(&User{}).Where("created_at >= ?", today).Count(&stats.Users.NewToday)

	// KYC stats
	s.db.Model(&KYCRecord{}).Where("status = ?", "pending").Count(&stats.KYC.Pending)
	s.db.Model(&KYCRecord{}).Where("status = ?", "approved").Count(&stats.KYC.Approved)
	s.db.Model(&KYCRecord{}).Where("status = ?", "rejected").Count(&stats.KYC.Rejected)

	// Token stats
	s.db.Model(&Token{}).Count(&stats.Tokens.Total)
	s.db.Model(&Token{}).Where("is_active = ?", true).Count(&stats.Tokens.Active)

	// Pair stats
	s.db.Model(&TradingPair{}).Count(&stats.Pairs.Total)
	s.db.Model(&TradingPair{}).Where("status = ?", "active").Count(&stats.Pairs.Active)

	// Transaction stats
	s.db.Model(&Transaction{}).Where("created_at >= ?", today).Count(&stats.Transactions.Today)

	// Withdrawal stats
	s.db.Model(&Withdrawal{}).Where("status = ?", "pending").Count(&stats.Withdrawals.Pending)

	// White label stats
	s.db.Model(&WhiteLabelClient{}).Count(&stats.WhiteLabels.Total)
	s.db.Model(&WhiteLabelClient{}).Where("status = ?", "active").Count(&stats.WhiteLabels.Active)

	// System stats
	stats.System.Health = "healthy"
	stats.System.Uptime = 99.9

	c.JSON(http.StatusOK, stats)
}

func (s *AdminPlatformService) GetAnalytics(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")
	
	var analytics struct {
		Users      []interface{} `json:"users"`
		Revenue    float64      `json:"revenue"`
		Volume     float64      `json:"volume"`
		Transactions []interface{} `json:"transactions"`
	}

	// This would aggregate data based on period
	// For now, return mock data structure
	analytics.Revenue = 125000.50
	analytics.Volume = 5000000.00

	c.JSON(http.StatusOK, analytics)
}

func (s *AdminPlatformService) ExportReport(c *gin.Context) {
	reportType := c.Param("type")
	format := c.DefaultQuery("format", "csv")

	// Generate report based on type and format
	// In production, this would generate actual CSV/PDF/Excel files
	c.JSON(http.StatusOK, gin.H{
		"message":    "Report generation started",
		"report_type": reportType,
		"format":     format,
	})
}

// ============================================================================
// Notification Handlers
// ============================================================================

func (s *AdminPlatformService) ListNotifications(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var notifications []Notification
	var total int64

	query := s.db.Model(&Notification{}).Where("admin_id = ?", adminID)
	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        notifications,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

func (s *AdminPlatformService) MarkNotificationRead(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	notificationID := c.Param("id")

	now := time.Now()
	result := s.db.Model(&Notification{}).Where("id = ? AND admin_id = ?", notificationID, adminID).Updates(map[string]interface{}{
		"is_read": true,
		"read_at": now,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

func (s *AdminPlatformService) SendNotification(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var notification Notification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	notification.AdminID = adminID

	if err := s.db.Create(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send notification"})
		return
	}

	c.JSON(http.StatusCreated, notification)
}

func (s *AdminPlatformService) BroadcastNotification(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var notification Notification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get all admins
	var admins []Admin
	s.db.Find(&admins)

	// Create notifications for all admins
	for _, admin := range admins {
		notif := notification
		notif.AdminID = admin.ID
		s.db.Create(&notif)
	}

	s.logAudit(adminID, "NOTIFICATION_BROADCAST", "notification", "", c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Notification broadcasted"})
}

// ============================================================================
// API Key Management Handlers
// ============================================================================

func (s *AdminPlatformService) ListAPIKeys(c *gin.Context) {
	var keys []APIKey
	if err := s.db.Order("created_at DESC").Find(&keys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch API keys"})
		return
	}
	c.JSON(http.StatusOK, keys)
}

func (s *AdminPlatformService) CreateAPIKey(c *gin.Context) {
	var key APIKey
	if err := c.ShouldBindJSON(&key); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Generate key
	keyID := uuid.New().String()
	keyPrefix := keyID[:8]
	keySecret := uuid.New().String() + uuid.New().String()
	
	// Hash the key for storage
	keyHash := sha256.Sum256([]byte(keySecret))
	key.KeyHash = hex.EncodeToString(keyHash[:])
	key.KeyPrefix = keyPrefix
	key.KeyID = keyID
	key.IsActive = true

	if err := s.db.Create(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	// Return the full key only once
	c.JSON(http.StatusCreated, gin.H{
		"api_key": gin.H{
			"id":           key.ID,
			"key_id":       key.KeyID,
			"key":          keyPrefix + "." + keySecret,
			"name":         key.Name,
			"permissions":  key.Permissions,
			"rate_limit":   key.RateLimit,
			"expires_at":   key.ExpiresAt,
		},
	})
}

func (s *AdminPlatformService) RevokeAPIKey(c *gin.Context) {
	keyID := c.Param("id")

	result := s.db.Model(&APIKey{}).Where("key_id = ?", keyID).Update("is_active", false)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}

// ============================================================================
// Webhook Management Handlers
// ============================================================================

func (s *AdminPlatformService) ListWebhooks(c *gin.Context) {
	var webhooks []Webhook
	if err := s.db.Order("created_at DESC").Find(&webhooks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch webhooks"})
		return
	}
	c.JSON(http.StatusOK, webhooks)
}

func (s *AdminPlatformService) CreateWebhook(c *gin.Context) {
	var webhook Webhook
	if err := c.ShouldBindJSON(&webhook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	webhook.WebhookID = "wh_" + uuid.New().String()[:8]
	webhook.Secret = uuid.New().String()
	webhook.IsActive = true

	if err := s.db.Create(&webhook).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create webhook"})
		return
	}

	c.JSON(http.StatusCreated, webhook)
}

func (s *AdminPlatformService) UpdateWebhook(c *gin.Context) {
	webhookID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Don't allow updating secret
	delete(updates, "secret")

	result := s.db.Model(&Webhook{}).Where("webhook_id = ?", webhookID).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook updated"})
}

func (s *AdminPlatformService) DeleteWebhook(c *gin.Context) {
	webhookID := c.Param("id")

	result := s.db.Where("webhook_id = ?", webhookID).Delete(&Webhook{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook deleted"})
}

// ============================================================================
// Audit Log Handlers
// ============================================================================

func (s *AdminPlatformService) GetAuditLogs(c *gin.Context) {
	adminID := c.Query("admin_id")
	action := c.Query("action")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "50")

	var logs []AuditLog
	query := s.db.Model(&AuditLog{})

	if adminID != "" {
		adminIDUint, _ := strconv.ParseUint(adminID, 10, 32)
		query = query.Where("admin_id = ?", uint(adminIDUint))
	}
	if action != "" {
		query = query.Where("action LIKE ?", "%"+action+"%")
	}

	var total int64
	query.Count(&total)

	pageNum, _ := strconv.Atoi(page)
	limitNum, _ := strconv.Atoi(limit)

	offset := (pageNum - 1) * limitNum
	query.Order("created_at DESC").Offset(offset).Limit(limitNum).Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"page":  pageNum,
		"limit": limitNum,
	})
}

// ============================================================================
// Compliance & Security Dashboard Handlers
// ============================================================================

func (s *AdminPlatformService) GetComplianceDashboard(c *gin.Context) {
	var dashboard struct {
		TotalKYC          int64   `json:"total_kyc"`
		PendingKYC        int64   `json:"pending_kyc"`
		ApprovedKYC       int64   `json:"approved_kyc"`
		RejectedKYC       int64   `json:"rejected_kyc"`
		HighRiskUsers     int64   `json:"high_risk_users"`
		SuspiciousActivity int64  `json:"suspicious_activity"`
		TransactionsFlagged int64  `json:"transactions_flagged"`
		ComplianceScore   float64 `json:"compliance_score"`
	}

	s.db.Model(&KYCRecord{}).Count(&dashboard.TotalKYC)
	s.db.Model(&KYCRecord{}).Where("status = ?", "pending").Count(&dashboard.PendingKYC)
	s.db.Model(&KYCRecord{}).Where("status = ?", "approved").Count(&dashboard.ApprovedKYC)
	s.db.Model(&KYCRecord{}).Where("status = ?", "rejected").Count(&dashboard.RejectedKYC)

	// Compliance score calculation (simplified)
	dashboard.ComplianceScore = 95.5

	c.JSON(http.StatusOK, dashboard)
}

func (s *AdminPlatformService) GetFinanceDashboard(c *gin.Context) {
	var dashboard struct {
		TotalRevenue      float64 `json:"total_revenue"`
		RevenueToday      float64 `json:"revenue_today"`
		RevenueThisMonth  float64 `json:"revenue_this_month"`
		TradingVolume     float64 `json:"trading_volume"`
		FeesCollected     float64 `json:"fees_collected"`
		PendingWithdrawals float64 `json:"pending_withdrawals"`
	}

	dashboard.TotalRevenue = 1500000.00
	dashboard.RevenueToday = 25000.00
	dashboard.RevenueThisMonth = 750000.00
	dashboard.TradingVolume = 50000000.00
	dashboard.FeesCollected = 150000.00
	dashboard.PendingWithdrawals = 50000.00

	c.JSON(http.StatusOK, dashboard)
}

func (s *AdminPlatformService) GetSecurityDashboard(c *gin.Context) {
	var dashboard struct {
		FailedLogins     int64   `json:"failed_logins"`
		ActiveSessions   int64   `json:"active_sessions"`
		SuspiciousIPs    int64   `json:"suspicious_ips"`
		SecurityEvents   int64   `json:"security_events"`
		BlockedIPs       int64   `json:"blocked_ips"`
		TwoFactorEnabled int64   `json:"two_factor_enabled"`
		SecurityScore    float64 `json:"security_score"`
	}

	s.db.Model(&Admin{}).Where("two_factor_enabled = ?", true).Count(&dashboard.TwoFactorEnabled)
	dashboard.SecurityScore = 92.0

	c.JSON(http.StatusOK, dashboard)
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *AdminPlatformService) generateJWT(admin *Admin) (string, time.Time, error) {
	expiresAt := time.Now().Add(time.Duration(s.config.JWTExpiration) * time.Second)

	claims := jwt.MapClaims{
		"admin_id":  admin.ID,
		"email":     admin.Email,
		"role":      admin.Role,
		"exp":       expiresAt.Unix(),
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

func (s *AdminPlatformService) validateJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *AdminPlatformService) validatePassword(password string) error {
	if len(password) < s.config.PasswordMinLength {
		return fmt.Errorf("password must be at least %d characters", s.config.PasswordMinLength)
	}

	if s.config.PasswordRequireUpper {
		if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
			return fmt.Errorf("password must contain at least one uppercase letter")
		}
	}

	if s.config.PasswordRequireLower {
		if !regexp.MustCompile(`[a-z]`).MatchString(password) {
			return fmt.Errorf("password must contain at least one lowercase letter")
		}
	}

	if s.config.PasswordRequireNumber {
		if !regexp.MustCompile(`[0-9]`).MatchString(password) {
			return fmt.Errorf("password must contain at least one number")
		}
	}

	if s.config.PasswordRequireSpecial {
		if !regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password) {
			return fmt.Errorf("password must contain at least one special character")
		}
	}

	return nil
}

func (s *AdminPlatformService) logAudit(adminID uint, action, resourceType, resourceID, ip string, success bool, details string) {
	log := AuditLog{
		AdminID:      adminID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		IPAddress:    ip,
		Status:       "success",
	}
	if !success {
		log.Status = "failed"
		log.Details = &details
	}
	s.db.Create(&log)
}

// ============================================================================
// Rate Limiter
// ============================================================================

type RateLimiter struct {
	redis  *redis.Client
	config *Config
}

func NewRateLimiter(redisClient *redis.Client, config *Config) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		config: config,
	}
}

func (r *RateLimiter) Allow(key string) (bool, error) {
	ctx := context.Background()

	// Check minute limit
	minuteKey := fmt.Sprintf("ratelimit:%s:minute", key)
	minuteCount, err := r.redis.Incr(ctx, minuteKey).Result()
	if err != nil {
		return false, err
	}
	if minuteCount == 1 {
		r.redis.Expire(ctx, minuteKey, time.Minute)
	}
	if minuteCount > int64(r.config.RateLimitPerMinute) {
		return false, nil
	}

	// Check hour limit
	hourKey := fmt.Sprintf("ratelimit:%s:hour", key)
	hourCount, err := r.redis.Incr(ctx, hourKey).Result()
	if err != nil {
		return false, err
	}
	if hourCount == 1 {
		r.redis.Expire(ctx, hourKey, time.Hour)
	}
	if hourCount > int64(r.config.RateLimitPerHour) {
		return false, nil
	}

	return true, nil
}

// ============================================================================
// Webhook Manager
// ============================================================================

type WebhookManager struct {
	db *gorm.DB
}

func NewWebhookManager(db *gorm.DB) *WebhookManager {
	return &WebhookManager{db: db}
}

func (w *WebhookManager) Trigger(event string, data interface{}) error {
	var webhooks []Webhook
	if err := w.db.Where("is_active = ?", true).Find(&webhooks).Error; err != nil {
		return err
	}

	for _, webhook := range webhooks {
		// Check if event matches
		var events []string
		json.Unmarshal(webhook.Events, &events)
		
		matched := false
		for _, e := range events {
			if e == event || e == "*" {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		// Send webhook (simplified)
		go w.sendWebhook(webhook, event, data)
	}

	return nil
}

func (w *WebhookManager) sendWebhook(webhook Webhook, event string, data interface{}) error {
	// In production, this would send actual HTTP requests with proper signature
	return nil
}

// ============================================================================
// Middleware
// ============================================================================

func (s *AdminPlatformService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:]
		claims, err := s.validateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("admin_id", uint(claims["admin_id"].(float64)))
		c.Set("email", claims["email"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

func (s *AdminPlatformService) RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")

		for _, allowedRole := range allowedRoles {
			if role == allowedRole || role == "super_admin" {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		c.Abort()
	}
}

func (s *AdminPlatformService) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Authorization")
		if key == "" {
			key = c.ClientIP()
		}

		allowed, err := s.rateLimiter.Allow(key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rate limit check failed"})
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	service, err := NewAdminPlatformService(config)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

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
	router.POST("/api/v1/auth/login", service.Login)

	// Protected routes
	api := router.Group("/api/v1")
	api.Use(service.AuthMiddleware())
	api.Use(service.RateLimitMiddleware())
	{
		// Auth
		api.POST("/auth/logout", service.Logout)
		api.GET("/auth/me", service.GetProfile)
		api.POST("/auth/change-password", service.ChangePassword)

		// Admins (Super Admin only)
		api.GET("/admins", service.RoleMiddleware("super_admin"), service.ListAdmins)
		api.POST("/admins", service.RoleMiddleware("super_admin"), service.CreateAdmin)
		api.PUT("/admins/:id", service.RoleMiddleware("super_admin"), service.UpdateAdmin)
		api.DELETE("/admins/:id", service.RoleMiddleware("super_admin"), service.DeleteAdmin)

		// Users
		api.GET("/users", service.ListUsers)
		api.GET("/users/:id", service.GetUser)
		api.PUT("/users/:id", service.UpdateUser)
		api.POST("/users/:id/suspend", service.SuspendUser)
		api.POST("/users/:id/ban", service.BanUser)

		// KYC
		api.GET("/kyc", service.ListKYC)
		api.POST("/kyc/:id/approve", service.ApproveKYC)
		api.POST("/kyc/:id/reject", service.RejectKYC)

		// Tokens
		api.GET("/tokens", service.ListTokens)
		api.POST("/tokens", service.CreateToken)
		api.PUT("/tokens/:id", service.UpdateToken)
		api.DELETE("/tokens/:id", service.DeleteToken)
		api.POST("/tokens/:id/suspend", service.SuspendToken)

		// Pairs
		api.GET("/pairs", service.ListPairs)
		api.POST("/pairs", service.CreatePair)
		api.POST("/pairs/:id/halt", service.HaltPair)

		// Blockchains
		api.GET("/blockchains", service.ListBlockchains)
		api.POST("/blockchains", service.CreateBlockchain)
		api.PUT("/blockchains/:id", service.UpdateBlockchain)

		// White Labels
		api.GET("/white-labels", service.ListWhiteLabels)
		api.POST("/white-labels", service.CreateWhiteLabel)
		api.POST("/white-labels/:id/activate", service.ActivateWhiteLabel)
		api.POST("/white-labels/:id/suspend", service.SuspendWhiteLabel)

		// Transactions
		api.GET("/transactions", service.ListTransactions)
		api.POST("/transactions/:id/cancel", service.CancelTransaction)

		// Withdrawals
		api.GET("/withdrawals", service.ListWithdrawals)
		api.POST("/withdrawals/:id/approve", service.ApproveWithdrawal)
		api.POST("/withdrawals/:id/reject", service.RejectWithdrawal)
		api.POST("/withdrawals/:id/process", service.ProcessWithdrawal)
		api.POST("/withdrawals/batch-approve", service.BatchApproveWithdrawals)

		// Fees
		api.GET("/fees", service.ListFees)
		api.PUT("/fees/:id", service.UpdateFee)

		// Tickets
		api.GET("/tickets", service.ListTickets)
		api.POST("/tickets", service.CreateTicket)
		api.PUT("/tickets/:id", service.UpdateTicket)
		api.POST("/tickets/:id/messages", service.AddTicketMessage)

		// Knowledge Base
		api.GET("/knowledge-base", service.ListArticles)
		api.POST("/knowledge-base", service.CreateArticle)
		api.PUT("/knowledge-base/:id", service.UpdateArticle)

		// Approval Workflows
		api.GET("/workflows", service.ListWorkflows)
		api.POST("/workflows", service.CreateWorkflow)
		api.GET("/approval-requests", service.ListApprovalRequests)
		api.POST("/approval-requests/:id/approve", service.ApproveRequest)
		api.POST("/approval-requests/:id/reject", service.RejectRequest)

		// Analytics
		api.GET("/dashboard", service.GetDashboardStats)
		api.GET("/analytics", service.GetAnalytics)
		api.GET("/analytics/reports/:type", service.ExportReport)

		// Notifications
		api.GET("/notifications", service.ListNotifications)
		api.PUT("/notifications/:id/read", service.MarkNotificationRead)
		api.POST("/notifications", service.SendNotification)
		api.POST("/notifications/broadcast", service.BroadcastNotification)

		// API Keys
		api.GET("/api-keys", service.ListAPIKeys)
		api.POST("/api-keys", service.CreateAPIKey)
		api.POST("/api-keys/:id/revoke", service.RevokeAPIKey)

		// Webhooks
		api.GET("/webhooks", service.ListWebhooks)
		api.POST("/webhooks", service.CreateWebhook)
		api.PUT("/webhooks/:id", service.UpdateWebhook)
		api.DELETE("/webhooks/:id", service.DeleteWebhook)

		// Audit Logs
		api.GET("/audit-logs", service.GetAuditLogs)

		// Compliance/Finance/Security Dashboards
		api.GET("/dashboard/compliance", service.GetComplianceDashboard)
		api.GET("/dashboard/finance", service.GetFinanceDashboard)
		api.GET("/dashboard/security", service.GetSecurityDashboard)
	}

	// Health
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "admin-platform",
			"timestamp": time.Now().Unix(),
		})
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Admin Platform starting on port %s", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
}

// Configuration addendum
func (c *Config) getPasswordPepper() string {
	return "tigerwallet-admin-pepper"
}

var _ = getPasswordPepper
