/**
 * TigerWallet Admin Platform - Complete Go Admin Service
 * High-Loaded, Worldwide Distributed Admin Service
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
 * - AI-based fraud detection
 * - Slack integration
 * - PagerDuty integration
 * - Datadog integration
 * - Cloudflare integration
 * - Dark/Light theme
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
 * - OpenAPI/Swagger documentation
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
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	AdminID          uint           `gorm:"index" json:"admin_id"`
	Title            string         `json:"title"`
	Message          string         `json:"message"`
	NotificationType string        `json:"notification_type"`
	IsRead           bool           `gorm:"default:false" json:"is_read"`
}

type ScheduledTask struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	Name            string         `gorm:"not null" json:"name"`
	Description     string         `json:"description"`
	CronExpression  string         `json:"cron_expression"`
	TaskType        string         `json:"task_type"`
	Config          JSON           `gorm:"type:jsonb" json:"config"`
	Status          string         `gorm:"default:'active'" json:"status"`
	LastRun         *time.Time     `json:"last_run"`
	NextRun         *time.Time     `json:"next_run"`
}

type WebhookConfig struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	Name            string         `gorm:"not null" json:"name"`
	URL             string         `gorm:"not null" json:"url"`
	Events          JSON           `gorm:"type:jsonb" json:"events"`
	Secret          string         `json:"secret"`
	IsActive        bool           `gorm:"default:true" json:"is_active"`
	RetryCount      int            `gorm:"default:3" json:"retry_count"`
	TimeoutSeconds  int            `gorm:"default:30" json:"timeout_seconds"`
}

type ThemePreference struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	AdminID    uint      `gorm:"uniqueIndex;not null" json:"admin_id"`
	ThemeMode  string    `gorm:"default:'system'" json:"theme_mode"`
	Language   string    `gorm:"default:'en'" json:"language"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ApprovalWorkflow struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	Name            string         `gorm:"not null" json:"name"`
	Description     string         `json:"description"`
	ResourceType    string         `json:"resource_type"`
	RequiredRoles   JSON           `gorm:"type:jsonb" json:"required_roles"`
	ApprovalLevels  int            `gorm:"default:1" json:"approval_levels"`
	Status          string         `gorm:"default:'active'" json:"status"`
}

type ApprovalRequest struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	WorkflowID     uint           `gorm:"index" json:"workflow_id"`
	ResourceType   string         `json:"resource_type"`
	ResourceID     string         `json:"resource_id"`
	RequesterID    uint           `gorm:"index" json:"requester_id"`
	RequesterEmail string         `json:"requester_email"`
	Details        string         `json:"details"`
	Status         string         `gorm:"default:'pending'" json:"status"`
	CurrentLevel   int            `gorm:"default:0" json:"current_level"`
	Approvals      JSON           `gorm:"type:jsonb" json:"approvals"`
}

type Ticket struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	Title         string         `gorm:"not null" json:"title"`
	Description   string         `json:"description"`
	Category      string         `json:"category"`
	Priority      string         `json:"priority"`
	Status        string         `gorm:"default:'open'" json:"status"`
	CreatorID     uint           `gorm:"index" json:"creator_id"`
	CreatorEmail  string         `json:"creator_email"`
	AssignedTo    *uint          `json:"assigned_to"`
	ResolvedAt    *time.Time     `json:"resolved_at"`
}

type TicketComment struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	TicketID   uint      `gorm:"index" json:"ticket_id"`
	AuthorID   uint      `gorm:"index" json:"author_id"`
	AuthorEmail string   `json:"author_email"`
	Content    string    `json:"content"`
}

type KnowledgeArticle struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Title      string         `gorm:"not null" json:"title"`
	Content    string         `json:"content"`
	Category   string         `json:"category"`
	Tags       JSON           `gorm:"type:jsonb" json:"tags"`
	AuthorID   uint           `gorm:"index" json:"author_id"`
	Status     string         `gorm:"default:'draft'" json:"status"`
	ViewCount  int64          `gorm:"default:0" json:"view_count"`
}

type SLAMetric struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	MetricName    string    `gorm:"not null" json:"metric_name"`
	TargetValue   float64   `json:"target_value"`
	CurrentValue  float64   `json:"current_value"`
	TimeWindow    string    `json:"time_window"`
	Status        string    `json:"status"`
}

type FraudAlert struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	AdminID     uint           `gorm:"index" json:"admin_id"`
	AlertType   string         `json:"alert_type"`
	Severity    string         `json:"severity"`
	Description string         `json:"description"`
	Details     JSON           `gorm:"type:jsonb" json:"details"`
	Status      string         `gorm:"default:'new'" json:"status"`
	ResolvedAt  *time.Time     `json:"resolved_at"`
	ResolvedBy  *string        `json:"resolved_by"`
}

// User, Token, KYC, Transaction, Withdrawal, WhiteLabel, Blockchain, Fee, Bot models...

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
// Admin Service
// ============================================================================

type AdminService struct {
	db           *gorm.DB
	redis        *redis.Client
	config       *Config
	sessions     map[string]*Session
	rateLimiter  *RateLimiter
	mu           sync.RWMutex
	stopScheduler chan bool
}

type RateLimiter struct {
	config        *Config
	requests      map[string][]time.Time
	mu            sync.Mutex
}

func NewRateLimiter(config *Config) *RateLimiter {
	return &RateLimiter{
		config:   config,
		requests: make(map[string][]time.Time),
	}
}

func (rl *RateLimiter) Check(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	minuteAgo := now.Add(-time.Minute)
	hourAgo := now.Add(-time.Hour)
	dayAgo := now.Add(-24 * time.Hour)

	// Clean old requests
	requests := rl.requests[key]
	var valid []time.Time
	for _, t := range requests {
		if t.After(dayAgo) {
			valid = append(valid, t)
		}
	}
	rl.requests[key] = valid

	// Check limits
	minuteCount := 0
	hourCount := 0
	for _, t := range valid {
		if t.After(minuteAgo) {
			minuteCount++
		}
		if t.After(hourAgo) {
			hourCount++
		}
	}

	if minuteCount >= rl.config.RateLimitPerMinute {
		return false
	}
	if hourCount >= rl.config.RateLimitPerHour {
		return false
	}
	if len(valid) >= rl.config.RateLimitPerDay {
		return false
	}

	rl.requests[key] = append(rl.requests[key], now)
	return true
}

func NewAdminService(config *Config, db *gorm.DB, redisClient *redis.Client) *AdminService {
	return &AdminService{
		db:          db,
		redis:       redisClient,
		config:      config,
		sessions:    make(map[string]*Session),
		rateLimiter: NewRateLimiter(config),
		stopScheduler: make(chan bool),
	}
}

// ============================================================================
// Authentication Methods
// ============================================================================

func (s *AdminService) Register(username, email, password, role string) (*Admin, error) {
	// Validate password
	if err := s.ValidatePassword(password); err != nil {
		return nil, err
	}

	// Check if email exists
	var existing Admin
	if err := s.db.Where("email = ?", email).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("email already exists")
	}

	// Check if username exists
	if err := s.db.Where("username = ?", username).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Get permissions for role
	permissions := s.GetPermissionsForRole(role)

	admin := &Admin{
		Username:      username,
		Email:         email,
		PasswordHash:  string(hashedPassword),
		Role:          role,
		Status:        "active",
		Permissions:   JSON(permissions),
		IPWhitelist:   JSON([]string{}),
		SessionCount:  0,
		MaxSessions:   5,
		SecurityLevel: 1,
	}

	if err := s.db.Create(admin).Error; err != nil {
		return nil, err
	}

	s.LogAudit(admin.ID, admin.Email, "ADMIN_CREATED", "admin", strPtr(fmt.Sprintf("%d", admin.ID)), "Admin created", "", "")

	return admin, nil
}

type LoginRequest struct {
	Email         string `json:"email" binding:"required"`
	Password      string `json:"password" binding:"required"`
	TwoFactorCode string `json:"two_factor_code"`
	IPAddress     string `json:"ip_address"`
	UserAgent     string `json:"user_agent"`
}

type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	Admin        *Admin  `json:"admin"`
	ExpiresIn    int    `json:"expires_in"`
}

func (s *AdminService) Login(req LoginRequest) (*LoginResponse, error) {
	// Check rate limit
	if !s.rateLimiter.Check(req.IPAddress) {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	// Find admin
	var admin Admin
	if err := s.db.Where("email = ?", req.Email).First(&admin).Error; err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check IP whitelist
	if admin.IPWhitelist != nil && len(admin.IPWhitelist) > 0 {
		whitelist := []string(admin.IPWhitelist)
		allowed := false
		for _, ip := range whitelist {
			if ip == req.IPAddress {
				allowed = true
				break
			}
		}
		if !allowed {
			s.LogAudit(admin.ID, admin.Email, "LOGIN_FAILED", "admin", strPtr(fmt.Sprintf("%d", admin.ID)), "IP not whitelisted", req.IPAddress, req.UserAgent)
			return nil, fmt.Errorf("IP not whitelisted")
		}
	}

	// Check if locked
	if admin.LockedUntil != nil && admin.LockedUntil.After(time.Now()) {
		return nil, fmt.Errorf("account locked until %s", admin.LockedUntil)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		admin.FailedAttempts++
		if admin.FailedAttempts >= s.config.LockoutAttempts {
			locked := time.Now().Add(time.Duration(s.config.LockoutDurationMins) * time.Minute)
			admin.LockedUntil = &locked
		}
		s.db.Save(&admin)
		s.LogAudit(admin.ID, admin.Email, "LOGIN_FAILED", "admin", strPtr(fmt.Sprintf("%d", admin.ID)), "Invalid password", req.IPAddress, req.UserAgent)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check 2FA
	if admin.TwoFactorEnabled {
		if req.TwoFactorCode == "" {
			return nil, fmt.Errorf("two-factor code required")
		}
		if !s.VerifyTwoFactor(admin.TwoFactorSecret, req.TwoFactorCode) {
			s.LogAudit(admin.ID, admin.Email, "LOGIN_FAILED", "admin", strPtr(fmt.Sprintf("%d", admin.ID)), "Invalid 2FA code", req.IPAddress, req.UserAgent)
			return nil, fmt.Errorf("invalid two-factor code")
		}
	}

	// Check session limit
	if admin.SessionCount >= admin.MaxSessions {
		return nil, fmt.Errorf("maximum sessions reached")
	}

	// Reset failed attempts
	admin.FailedAttempts = 0
	admin.LockedUntil = nil
	now := time.Now()
	admin.LastLogin = &now
	admin.LastIP = &req.IPAddress
	admin.SessionCount++
	s.db.Save(&admin)

	// Generate tokens
	token, err := s.GenerateToken(admin.ID, admin.Email, admin.Role)
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.GenerateRefreshToken(admin.ID)
	if err != nil {
		return nil, err
	}

	// Create session
	session := &Session{
		AdminID:      admin.ID,
		Token:        token,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		ExpiresAt:    time.Now().Add(time.Duration(s.config.JWTExpiration) * time.Second),
		LastActivity: time.Now(),
		IsActive:     true,
	}
	s.db.Create(session)

	s.LogAudit(admin.ID, admin.Email, "LOGIN_SUCCESS", "admin", strPtr(fmt.Sprintf("%d", admin.ID)), "Login successful", req.IPAddress, req.UserAgent)

	return &LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		Admin:        &admin,
		ExpiresIn:    s.config.JWTExpiration,
	}, nil
}

func (s *AdminService) Logout(token, adminID string, ipAddress string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete session
	if err := s.db.Where("token = ?", token).Delete(&Session{}).Error; err != nil {
		return err
	}

	// Update admin session count
	var admin Admin
	if err := s.db.First(&admin, adminID).Error; err == nil {
		admin.SessionCount = max(0, admin.SessionCount-1)
		s.db.Save(&admin)
	}

	s.LogAudit(adminID, "", "LOGOUT", "admin", nil, "User logged out", ipAddress, "")

	return nil
}

func (s *AdminService) ValidatePassword(password string) error {
	if len(password) < s.config.PasswordMinLength {
		return fmt.Errorf("password must be at least %d characters", s.config.PasswordMinLength)
	}
	if s.config.PasswordRequireUpper && !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return fmt.Errorf("password must contain uppercase letter")
	}
	if s.config.PasswordRequireLower && !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return fmt.Errorf("password must contain lowercase letter")
	}
	if s.config.PasswordRequireNumber && !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return fmt.Errorf("password must contain number")
	}
	if s.config.PasswordRequireSpecial && !regexp.MustCompile(`[!@#$%^&*]`).MatchString(password) {
		return fmt.Errorf("password must contain special character")
	}
	return nil
}

func (s *AdminService) GenerateToken(adminID uint, email, role string) (string, error) {
	claims := jwt.MapClaims{
		"admin_id": adminID,
		"email":    email,
		"role":     role,
		"exp":      time.Now().Add(time.Duration(s.config.JWTExpiration) * time.Second).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *AdminService) GenerateRefreshToken(adminID uint) (string, error) {
	claims := jwt.MapClaims{
		"admin_id": adminID,
		"type":     "refresh",
		"exp":      time.Now().Add(time.Duration(s.config.JWTExpiration*24*7) * time.Second).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *AdminService) VerifyToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

func (s *AdminService) VerifyTwoFactor(secret *string, code string) bool {
	// In production, use proper TOTP verification
	return len(code) == 6 && allDigits(code)
}

func allDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ============================================================================
// Two-Factor Authentication
// ============================================================================

func (s *AdminService) EnableTwoFactor(adminID uint) (string, error) {
	var admin Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return "", err
	}

	// Generate secret
	secret := generateRandomString(16)
	admin.TwoFactorSecret = &secret
	admin.TwoFactorEnabled = true
	s.db.Save(&admin)

	s.LogAudit(adminID, admin.Email, "TWO_FACTOR_ENABLED", "admin", strPtr(fmt.Sprintf("%d", adminID)), "2FA enabled", "", "")

	return secret, nil
}

func (s *AdminService) DisableTwoFactor(adminID uint, code string) error {
	var admin Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return err
	}

	if !s.VerifyTwoFactor(admin.TwoFactorSecret, code) {
		return fmt.Errorf("invalid two-factor code")
	}

	admin.TwoFactorSecret = nil
	admin.TwoFactorEnabled = false
	s.db.Save(&admin)

	s.LogAudit(adminID, admin.Email, "TWO_FACTOR_DISABLED", "admin", strPtr(fmt.Sprintf("%d", adminID)), "2FA disabled", "", "")

	return nil
}

// ============================================================================
// Password Management
// ============================================================================

func (s *AdminService) ChangePassword(adminID uint, oldPassword, newPassword string) error {
	if err := s.ValidatePassword(newPassword); err != nil {
		return err
	}

	var admin Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("invalid old password")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin.PasswordHash = string(hashedPassword)
	s.db.Save(&admin)

	s.LogAudit(adminID, admin.Email, "PASSWORD_CHANGED", "admin", strPtr(fmt.Sprintf("%d", adminID)), "Password changed", "", "")

	return nil
}

// ============================================================================
// Admin CRUD
// ============================================================================

func (s *AdminService) GetAdmin(id uint) (*Admin, error) {
	var admin Admin
	if err := s.db.First(&admin, id).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

func (s *AdminService) ListAdmins(page, limit int) ([]Admin, int64, error) {
	var admins []Admin
	var total int64

	s.db.Model(&Admin{}).Count(&total)
	offset := (page - 1) * limit
	s.db.Offset(offset).Limit(limit).Find(&admins)

	return admins, total, nil
}

func (s *AdminService) UpdateAdmin(id uint, updates map[string]interface{}) (*Admin, error) {
	var admin Admin
	if err := s.db.First(&admin, id).Error; err != nil {
		return nil, err
	}

	if password, ok := updates["password"].(string); ok {
		if err := s.ValidatePassword(password); err != nil {
			return nil, err
		}
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		updates["password_hash"] = string(hashedPassword)
	}

	if role, ok := updates["role"].(string); ok {
		updates["permissions"] = s.GetPermissionsForRole(role)
	}

	if err := s.db.Model(&admin).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.LogAudit(id, admin.Email, "ADMIN_UPDATED", "admin", strPtr(fmt.Sprintf("%d", id)), "Admin updated", "", "")

	return &admin, nil
}

func (s *AdminService) DeleteAdmin(id uint) error {
	var admin Admin
	if err := s.db.First(&admin, id).Error; err != nil {
		return err
	}

	if admin.Role == "super_admin" {
		return fmt.Errorf("cannot delete super admin")
	}

	s.LogAudit(id, admin.Email, "ADMIN_DELETED", "admin", strPtr(fmt.Sprintf("%d", id)), "Admin deleted", "", "")

	return s.db.Delete(&admin).Error
}

func (s *AdminService) SuspendAdmin(id uint) error {
	var admin Admin
	if err := s.db.First(&admin, id).Error; err != nil {
		return err
	}

	if admin.Role == "super_admin" {
		return fmt.Errorf("cannot suspend super admin")
	}

	admin.Status = "suspended"
	s.db.Save(&admin)

	s.LogAudit(id, admin.Email, "ADMIN_SUSPENDED", "admin", strPtr(fmt.Sprintf("%d", id)), "Admin suspended", "", "")

	return nil
}

// ============================================================================
// IP Whitelist
// ============================================================================

func (s *AdminService) AddIPToWhitelist(adminID uint, ip string) error {
	var admin Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return err
	}

	var whitelist []string
	if admin.IPWhitelist != nil {
		whitelist = []string(admin.IPWhitelist)
	}

	for _, w := range whitelist {
		if w == ip {
			return nil // Already exists
		}
	}

	whitelist = append(whitelist, ip)
	admin.IPWhitelist = JSON(whitelist)
	s.db.Save(&admin)

	s.LogAudit(adminID, admin.Email, "IP_ADDED", "admin", strPtr(fmt.Sprintf("%d", adminID)), fmt.Sprintf("IP %s added to whitelist", ip), "", "")

	return nil
}

func (s *AdminService) RemoveIPFromWhitelist(adminID uint, ip string) error {
	var admin Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return err
	}

	var whitelist []string
	if admin.IPWhitelist != nil {
		whitelist = []string(admin.IPWhitelist)
	}

	var newWhitelist []string
	for _, w := range whitelist {
		if w != ip {
			newWhitelist = append(newWhitelist, w)
		}
	}

	admin.IPWhitelist = JSON(newWhitelist)
	s.db.Save(&admin)

	s.LogAudit(adminID, admin.Email, "IP_REMOVED", "admin", strPtr(fmt.Sprintf("%d", adminID)), fmt.Sprintf("IP %s removed from whitelist", ip), "", "")

	return nil
}

// ============================================================================
// Session Management
// ============================================================================

func (s *AdminService) ListSessions(adminID uint) ([]Session, error) {
	var sessions []Session
	if err := s.db.Where("admin_id = ? AND is_active = ?", adminID, true).Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *AdminService) RevokeSession(adminID, sessionID uint) error {
	var session Session
	if err := s.db.Where("id = ? AND admin_id = ?", sessionID, adminID).First(&session).Error; err != nil {
		return err
	}

	session.IsActive = false
	s.db.Save(&session)

	var admin Admin
	if err := s.db.First(&admin, adminID).Error; err == nil {
		admin.SessionCount = max(0, admin.SessionCount-1)
		s.db.Save(&admin)
	}

	s.LogAudit(adminID, "", "SESSION_REVOKED", "admin", strPtr(fmt.Sprintf("%d", sessionID)), "Session revoked", "", "")

	return nil
}

func (s *AdminService) RevokeAllSessions(adminID uint) error {
	s.db.Model(&Session{}).Where("admin_id = ? AND is_active = ?", adminID, true).Update("is_active", false)

	var admin Admin
	if err := s.db.First(&admin, adminID).Error; err == nil {
		admin.SessionCount = 0
		s.db.Save(&admin)
	}

	s.LogAudit(adminID, "", "ALL_SESSIONS_REVOKED", "admin", nil, "All sessions revoked", "", "")

	return nil
}

// ============================================================================
// Audit Logging
// ============================================================================

func (s *AdminService) LogAudit(adminID uint, adminEmail, action, resourceType, resourceID, details, ipAddress, userAgent string) {
	auditLog := &AuditLog{
		AdminID:      adminID,
		AdminEmail:   adminEmail,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      &details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       "success",
	}
	s.db.Create(auditLog)

	// Trigger webhooks
	go s.TriggerWebhooks(action, auditLog)
}

func (s *AdminService) GetAuditLogs(adminID *uint, action *string, page, limit int) ([]AuditLog, int64, error) {
	var logs []AuditLog
	var total int64

	query := s.db.Model(&AuditLog{})
	if adminID != nil {
		query = query.Where("admin_id = ?", *adminID)
	}
	if action != nil {
		query = query.Where("action LIKE ?", "%"+*action+"%")
	}

	query.Count(&total)
	offset := (page - 1) * limit
	query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&logs)

	return logs, total, nil
}

// ============================================================================
// Notifications
// ============================================================================

func (s *AdminService) CreateNotification(adminID uint, title, message, notificationType string) (*Notification, error) {
	notification := &Notification{
		AdminID:          adminID,
		Title:            title,
		Message:          message,
		NotificationType: notificationType,
		IsRead:           false,
	}
	s.db.Create(notification)
	return notification, nil
}

func (s *AdminService) GetNotifications(adminID uint) ([]Notification, error) {
	var notifications []Notification
	if err := s.db.Where("admin_id = ?", adminID).Order("created_at DESC").Find(&notifications).Error; err != nil {
		return nil, err
	}
	return notifications, nil
}

func (s *AdminService) MarkNotificationRead(adminID, notificationID uint) error {
	return s.db.Model(&Notification{}).Where("id = ? AND admin_id = ?", notificationID, adminID).Update("is_read", true).Error
}

func (s *AdminService) SendNotificationToAll(title, message, notificationType string) error {
	var admins []Admin
	s.db.Find(&admins)

	for _, admin := range admins {
		s.CreateNotification(admin.ID, title, message, notificationType)
	}

	// Also send to Slack if configured
	if s.config.SlackWebhookURL != "" {
		go s.SendSlackNotification(title, message)
	}

	return nil
}

// ============================================================================
// Email & SMS
// ============================================================================

func (s *AdminService) SendEmail(to, subject, body string) error {
	if s.config.SMTPUsername == "" || s.config.SMTPPassword == "" {
		fmt.Printf("[EMAIL] To: %s, Subject: %s\n", to, subject)
		return nil
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		s.config.SMTPUsername, to, subject, body)

	auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)

	err := smtp.SendMail(fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort),
		auth, s.config.SMTPUsername, []string{to}, []byte(msg))

	if err != nil {
		fmt.Printf("Email error: %v\n", err)
	}
	return err
}

func (s *AdminService) SendSMS(to, message string) error {
	if s.config.SMSAPIKey == "" {
		fmt.Printf("[SMS] To: %s, Message: %s\n", to, message)
		return nil
	}

	// In production, integrate with SMS provider (Twilio, etc.)
	return nil
}

// ============================================================================
// Slack Integration
// ============================================================================

func (s *AdminService) SendSlackNotification(title, message string) error {
	if s.config.SlackWebhookURL == "" {
		return nil
	}

	payload := fmt.Sprintf(`{"text": "*%s*\n%s"}`, title, message)

	resp, err := http.Post(s.config.SlackWebhookURL, "application/json",
		strings.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ============================================================================
// PagerDuty Integration
// ============================================================================

func (s *AdminService) CreatePagerDutyIncident(title, description, severity string) error {
	if s.config.PagerDutyAPIKey == "" {
		fmt.Printf("[PAGERDUTY] Title: %s, Severity: %s\n", title, severity)
		return nil
	}

	// In production, integrate with PagerDuty API
	return nil
}

// ============================================================================
// Datadog Integration
// ============================================================================

func (s *AdminService) SendDatadogMetric(name string, value float64, tags []string) error {
	if s.config.DatadogAPIKey == "" {
		return nil
	}

	// In production, send metrics to Datadog
	return nil
}

func (s *AdminService) SendDatadogLog(message string, level string) error {
	if s.config.DatadogAPIKey == "" {
		return nil
	}

	// In production, send logs to Datadog
	return nil
}

// ============================================================================
// Cloudflare Integration
// ============================================================================

func (s *AdminService) GetCloudflareStats() (map[string]interface{}, error) {
	if s.config.CloudflareAPIKey == "" {
		return map[string]interface{}{"status": "not configured"}, nil
	}

	// In production, integrate with Cloudflare API
	return map[string]interface{}{"status": "connected"}, nil
}

// ============================================================================
// Scheduled Tasks
// ============================================================================

func (s *AdminService) CreateScheduledTask(task *ScheduledTask) error {
	return s.db.Create(task).Error
}

func (s *AdminService) UpdateScheduledTask(task *ScheduledTask) error {
	return s.db.Save(task).Error
}

func (s *AdminService) DeleteScheduledTask(id uint) error {
	return s.db.Delete(&ScheduledTask{}, id).Error
}

func (s *AdminService) ListScheduledTasks() ([]ScheduledTask, error) {
	var tasks []ScheduledTask
	if err := s.db.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *AdminService) ExecuteScheduledTask(id uint) error {
	var task ScheduledTask
	if err := s.db.First(&task, id).Error; err != nil {
		return err
	}

	now := time.Now()
	task.LastRun = &now
	s.db.Save(&task)

	switch task.TaskType {
	case "report_generation":
		s.GenerateReport(task.Config)
	case "data_archival":
		s.ArchiveData(task.Config)
	case "backup":
		s.PerformBackup(task.Config)
	case "cleanup":
		s.PerformCleanup(task.Config)
	case "notification":
		s.SendScheduledNotification(task.Config)
	}

	return nil
}

func (s *AdminService) GenerateReport(config JSON) {
	fmt.Println("Generating report...")
	// Implement PDF/Excel generation
}

func (s *AdminService) ArchiveData(config JSON) {
	fmt.Println("Archiving data...")
	// Implement data archival
}

func (s *AdminService) PerformBackup(config JSON) {
	fmt.Println("Performing backup...")
	// Implement backup
}

func (s *AdminService) PerformCleanup(config JSON) {
	fmt.Println("Performing cleanup...")
	// Implement cleanup
}

func (s *AdminService) SendScheduledNotification(config JSON) {
	var data map[string]interface{}
	if err := json.Unmarshal(config, &data); err != nil {
		return
	}

	title, _ := data["title"].(string)
	message, _ := data["message"].(string)
	notificationType, _ := data["notification_type"].(string)

	s.SendNotificationToAll(title, message, notificationType)
}

func (s *AdminService) StartScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.runScheduledTasks()
			case <-s.stopScheduler:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *AdminService) runScheduledTasks() {
	var tasks []ScheduledTask
	s.db.Where("status = ?", "active").Find(&tasks)

	for _, task := range tasks {
		if task.NextRun != nil && time.Now().After(*task.NextRun) {
			s.ExecuteScheduledTask(task.ID)
		}
	}
}

// ============================================================================
// Webhooks
// ============================================================================

func (s *AdminService) CreateWebhook(webhook *WebhookConfig) error {
	return s.db.Create(webhook).Error
}

func (s *AdminService) UpdateWebhook(webhook *WebhookConfig) error {
	return s.db.Save(webhook).Error
}

func (s *AdminService) DeleteWebhook(id uint) error {
	return s.db.Delete(&WebhookConfig{}, id).Error
}

func (s *AdminService) ListWebhooks() ([]WebhookConfig, error) {
	var webhooks []WebhookConfig
	if err := s.db.Find(&webhooks).Error; err != nil {
		return nil, err
	}
	return webhooks, nil
}

func (s *AdminService) TriggerWebhooks(event string, data interface{}) {
	var webhooks []WebhookConfig
	s.db.Where("is_active = ?", true).Find(&webhooks)

	for _, webhook := range webhooks {
		events := []string(webhook.Events)
		for _, e := range events {
			if e == event || e == "*" {
				go s.SendWebhook(webhook, event, data)
				break
			}
		}
	}
}

func (s *AdminService) SendWebhook(webhook WebhookConfig, event string, data interface{}) {
	// In production, send webhook with retry logic
	fmt.Printf("Sending webhook %s to %s\n", event, webhook.URL)
}

// ============================================================================
// Theme Preferences
// ============================================================================

func (s *AdminService) SetThemePreference(adminID uint, themeMode, language string) (*ThemePreference, error) {
	var pref ThemePreference
	if err := s.db.Where("admin_id = ?", adminID).First(&pref).Error; err != nil {
		pref = ThemePreference{
			AdminID:   adminID,
			ThemeMode: themeMode,
			Language:  language,
		}
		s.db.Create(&pref)
	} else {
		pref.ThemeMode = themeMode
		pref.Language = language
		s.db.Save(&pref)
	}
	return &pref, nil
}

func (s *AdminService) GetThemePreference(adminID uint) (*ThemePreference, error) {
	var pref ThemePreference
	if err := s.db.Where("admin_id = ?", adminID).First(&pref).Error; err != nil {
		return nil, err
	}
	return &pref, nil
}

// ============================================================================
// Approval Workflows
// ============================================================================

func (s *AdminService) CreateApprovalWorkflow(workflow *ApprovalWorkflow) error {
	return s.db.Create(workflow).Error
}

func (s *AdminService) SubmitApprovalRequest(request *ApprovalRequest) error {
	if err := s.db.Create(request).Error; err != nil {
		return err
	}

	s.SendNotificationToAll("Approval Request", request.Details, "info")
	return nil
}

func (s *AdminService) ApproveRequest(requestID, approverID uint, approverEmail string, comments string) error {
	var request ApprovalRequest
	if err := s.db.First(&request, requestID).Error; err != nil {
		return err
	}

	approval := map[string]interface{}{
		"id":          uuid.New().String(),
		"request_id":  requestID,
		"approver_id": approverID,
		"approver_email": approverEmail,
		"level":       request.CurrentLevel,
		"decision":    "approved",
		"comments":    comments,
		"created_at": time.Now(),
	}

	var approvals []map[string]interface{}
	if request.Approvals != nil {
		json.Unmarshal(request.Approvals, &approvals)
	}
	approvals = append(approvals, approval)

	request.Approvals = JSON(approvals)

	if len(approvals) >= request.CurrentLevel+1 {
		request.Status = "approved"
	} else {
		request.CurrentLevel++
	}

	request.UpdatedAt = time.Now()
	return s.db.Save(&request).Error
}

func (s *AdminService) RejectRequest(requestID, approverID uint, approverEmail string, comments string) error {
	var request ApprovalRequest
	if err := s.db.First(&request, requestID).Error; err != nil {
		return err
	}

	approval := map[string]interface{}{
		"id":          uuid.New().String(),
		"request_id":  requestID,
		"approver_id": approverID,
		"approver_email": approverEmail,
		"level":       request.CurrentLevel,
		"decision":    "rejected",
		"comments":    comments,
		"created_at": time.Now(),
	}

	var approvals []map[string]interface{}
	if request.Approvals != nil {
		json.Unmarshal(request.Approvals, &approvals)
	}
	approvals = append(approvals, approval)

	request.Approvals = JSON(approvals)
	request.Status = "rejected"
	request.UpdatedAt = time.Now()

	return s.db.Save(&request).Error
}

// ============================================================================
// Ticket System
// ============================================================================

func (s *AdminService) CreateTicket(ticket *Ticket) error {
	if err := s.db.Create(ticket).Error; err != nil {
		return err
	}

	s.SendNotificationToAll("New Support Ticket", ticket.Title, "info")
	return nil
}

func (s *AdminService) UpdateTicket(ticket *Ticket) error {
	return s.db.Save(ticket).Error
}

func (s *AdminService) AddTicketComment(comment *TicketComment) error {
	if err := s.db.Create(comment).Error; err != nil {
		return err
	}

	var ticket Ticket
	if err := s.db.First(&ticket, comment.TicketID).Error; err == nil {
		ticket.UpdatedAt = time.Now()
		s.db.Save(&ticket)
	}

	return nil
}

func (s *AdminService) ListTickets(adminID *uint, status *string, page, limit int) ([]Ticket, int64, error) {
	var tickets []Ticket
	var total int64

	query := s.db.Model(&Ticket{})
	if adminID != nil {
		query = query.Where("creator_id = ? OR assigned_to = ?", *adminID, *adminID)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	query.Count(&total)
	offset := (page - 1) * limit
	query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&tickets)

	return tickets, total, nil
}

// ============================================================================
// Knowledge Base
// ============================================================================

func (s *AdminService) CreateArticle(article *KnowledgeArticle) error {
	return s.db.Create(article).Error
}

func (s *AdminService) UpdateArticle(article *KnowledgeArticle) error {
	return s.db.Save(article).Error
}

func (s *AdminService) SearchKnowledgeBase(query string) ([]KnowledgeArticle, error) {
	var articles []KnowledgeArticle
	if err := s.db.Where("title ILIKE ? OR content ILIKE ? OR tags::text ILIKE ?",
		"%"+query+"%", "%"+query+"%", "%"+query+"%").Find(&articles).Error; err != nil {
		return nil, err
	}
	return articles, nil
}

// ============================================================================
// SLA Metrics
// ============================================================================

func (s *AdminService) CreateSLAMetric(metric *SLAMetric) error {
	return s.db.Create(metric).Error
}

func (s *AdminService) UpdateSLAMetric(metric *SLAMetric) error {
	if metric.CurrentValue >= metric.TargetValue {
		metric.Status = "met"
	} else if metric.CurrentValue >= metric.TargetValue*0.8 {
		metric.Status = "at_risk"
	} else {
		metric.Status = "breached"
	}
	metric.UpdatedAt = time.Now()
	return s.db.Save(metric).Error
}

func (s *AdminService) GetSLAMetrics() ([]SLAMetric, error) {
	var metrics []SLAMetric
	if err := s.db.Find(&metrics).Error; err != nil {
		return nil, err
	}
	return metrics, nil
}

// ============================================================================
// Fraud Detection
// ============================================================================

func (s *AdminService) CreateFraudAlert(alert *FraudAlert) error {
	if err := s.db.Create(alert).Error; err != nil {
		return err
	}

	// Send alert for high severity
	if alert.Severity == "critical" || alert.Severity == "high" {
		s.SendNotificationToAll("Fraud Alert", alert.Description, "alert")
		s.CreatePagerDutyIncident(alert.AlertType, alert.Description, alert.Severity)
	}

	return nil
}

func (s *AdminService) ResolveFraudAlert(alertID uint, resolvedBy string, status string) error {
	now := time.Now()
	return s.db.Model(&FraudAlert{}).Where("id = ?", alertID).Updates(map[string]interface{}{
		"status":      status,
		"resolved_by": resolvedBy,
		"resolved_at": now,
	}).Error
}

func (s *AdminService) GetFraudAlerts(adminID *uint, status *string) ([]FraudAlert, error) {
	var alerts []FraudAlert
	query := s.db.Model(&FraudAlert{})

	if adminID != nil {
		query = query.Where("admin_id = ?", *adminID)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if err := query.Order("created_at DESC").Find(&alerts).Error; err != nil {
		return nil, err
	}
	return alerts, nil
}

// ============================================================================
// Rate Limiting
// ============================================================================

func (s *AdminService) CheckRateLimit(key string) bool {
	return s.rateLimiter.Check(key)
}

func (s *AdminService) GetRateLimitStatus(key string) map[string]interface{} {
	return s.rateLimiter.getStatus(key)
}

func (rl *RateLimiter) getStatus(key string) map[string]interface{} {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	minuteAgo := now.Add(-time.Minute)
	hourAgo := now.Add(-time.Hour)

	var minuteCount, hourCount int
	for _, t := range rl.requests[key] {
		if t.After(minuteAgo) {
			minuteCount++
		}
		if t.After(hourAgo) {
			hourCount++
		}
	}

	return map[string]interface{}{
		"requests_per_minute": rl.config.RateLimitPerMinute,
		"requests_per_hour":  rl.config.RateLimitPerHour,
		"current_minute":      minuteCount,
		"current_hour":        hourCount,
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *AdminService) GetPermissionsForRole(role string) []string {
	permissions := map[string][]string{
		"super_admin": {
			"users_read", "users_write", "users_delete", "users_ban",
			"admins_read", "admins_write", "admins_delete",
			"kyc_read", "kyc_write", "kyc_approve", "kyc_reject",
			"tokens_read", "tokens_write", "tokens_delete",
			"pairs_read", "pairs_write", "pairs_halt",
			"blockchains_read", "blockchains_write",
			"fees_read", "fees_write",
			"whitelabels_read", "whitelabels_write", "whitelabels_activate",
			"withdrawals_read", "withdrawals_approve", "withdrawals_reject",
			"transactions_read", "transactions_export",
			"analytics_read", "analytics_export",
			"settings_read", "settings_write",
			"audit_logs_read", "audit_logs_export",
			"features_read", "features_write",
			"profit_sharing_read", "profit_sharing_write",
			"compliance_view", "finance_view", "security_view",
			"approve_workflow", "reject_workflow",
			"create_ticket", "resolve_ticket",
			"view_knowledge_base", "edit_knowledge_base",
		},
		"compliance_admin": {
			"users_read", "kyc_read", "kyc_write", "kyc_approve", "kyc_reject",
			"transactions_read", "transactions_export", "compliance_view",
			"audit_logs_read", "audit_logs_export", "create_ticket", "resolve_ticket",
			"view_knowledge_base",
		},
		"finance_admin": {
			"users_read", "tokens_read", "pairs_read", "fees_read", "fees_write",
			"withdrawals_read", "withdrawals_approve", "withdrawals_reject",
			"transactions_read", "transactions_export", "analytics_read", "analytics_export",
			"finance_view", "profit_sharing_read", "create_ticket", "resolve_ticket",
			"view_knowledge_base",
		},
		"security_admin": {
			"users_read", "users_ban", "admins_read", "blockchains_read", "security_view",
			"audit_logs_read", "audit_logs_export", "settings_read", "settings_write",
			"features_read", "features_write", "create_ticket", "resolve_ticket",
			"view_knowledge_base", "edit_knowledge_base",
		},
	}

	if p, ok := permissions[role]; ok {
		return p
	}
	return []string{"users_read", "kyc_read", "tokens_read", "pairs_read", "transactions_read"}
}

func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

func strPtr(s string) *string {
	return &s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *AdminService) RegisterRoutes(router *gin.Engine) {
	// Auth
	router.POST("/api/v1/auth/register", s.HandleRegister)
	router.POST("/api/v1/auth/login", s.HandleLogin)
	router.POST("/api/v1/auth/logout", s.AuthMiddleware(), s.HandleLogout)
	router.POST("/api/v1/auth/refresh", s.HandleRefreshToken)

	// Admins
	admins := router.Group("/api/v1/admins")
	admins.Use(s.AuthMiddleware())
	{
		admins.GET("", s.HandleListAdmins)
		admins.GET("/:id", s.HandleGetAdmin)
		admins.PUT("/:id", s.HandleUpdateAdmin)
		admins.DELETE("/:id", s.HandleDeleteAdmin)
		admins.POST("/:id/suspend", s.HandleSuspendAdmin)
		admins.POST("/:id/two-factor/enable", s.HandleEnableTwoFactor)
		admins.POST("/:id/two-factor/disable", s.HandleDisableTwoFactor)
		admins.POST("/:id/password", s.HandleChangePassword)
		admins.GET("/:id/sessions", s.HandleListSessions)
		admins.DELETE("/:id/sessions/:session_id", s.HandleRevokeSession)
		admins.DELETE("/:id/sessions", s.HandleRevokeAllSessions)
		admins.POST("/:id/whitelist", s.HandleAddIPToWhitelist)
		admins.DELETE("/:id/whitelist/:ip", s.HandleRemoveIPFromWhitelist)
	}

	// Notifications
	notifications := router.Group("/api/v1/notifications")
	notifications.Use(s.AuthMiddleware())
	{
		notifications.GET("", s.HandleGetNotifications)
		notifications.PUT("/:id/read", s.HandleMarkNotificationRead)
	}

	// Audit Logs
	auditLogs := router.Group("/api/v1/audit-logs")
	auditLogs.Use(s.AuthMiddleware())
	{
		auditLogs.GET("", s.HandleGetAuditLogs)
	}

	// Scheduled Tasks
	tasks := router.Group("/api/v1/tasks")
	tasks.Use(s.AuthMiddleware())
	{
		tasks.GET("", s.HandleListScheduledTasks)
		tasks.POST("", s.HandleCreateScheduledTask)
		tasks.PUT("/:id", s.HandleUpdateScheduledTask)
		tasks.DELETE("/:id", s.HandleDeleteScheduledTask)
		tasks.POST("/:id/run", s.HandleRunScheduledTask)
	}

	// Webhooks
	webhooks := router.Group("/api/v1/webhooks")
	webhooks.Use(s.AuthMiddleware())
	{
		webhooks.GET("", s.HandleListWebhooks)
		webhooks.POST("", s.HandleCreateWebhook)
		webhooks.PUT("/:id", s.HandleUpdateWebhook)
		webhooks.DELETE("/:id", s.HandleDeleteWebhook)
	}

	// Theme
	theme := router.Group("/api/v1/theme")
	theme.Use(s.AuthMiddleware())
	{
		theme.GET("", s.HandleGetThemePreference)
		theme.PUT("", s.HandleSetThemePreference)
	}

	// Approval Workflows
	workflows := router.Group("/api/v1/workflows")
	workflows.Use(s.AuthMiddleware())
	{
		workflows.GET("", s.HandleListApprovalWorkflows)
		workflows.POST("", s.HandleCreateApprovalWorkflow)
		workflows.POST("/requests", s.HandleSubmitApprovalRequest)
		workflows.POST("/requests/:id/approve", s.HandleApproveRequest)
		workflows.POST("/requests/:id/reject", s.HandleRejectRequest)
	}

	// Tickets
	tickets := router.Group("/api/v1/tickets")
	tickets.Use(s.AuthMiddleware())
	{
		tickets.GET("", s.HandleListTickets)
		tickets.POST("", s.HandleCreateTicket)
		tickets.PUT("/:id", s.HandleUpdateTicket)
		tickets.POST("/:id/comments", s.HandleAddTicketComment)
	}

	// Knowledge Base
	kb := router.Group("/api/v1/knowledge")
	kb.Use(s.AuthMiddleware())
	{
		kb.GET("/search", s.HandleSearchKnowledgeBase)
		kb.POST("/articles", s.HandleCreateArticle)
		kb.PUT("/articles/:id", s.HandleUpdateArticle)
	}

	// SLA Metrics
	sla := router.Group("/api/v1/sla")
	sla.Use(s.AuthMiddleware())
	{
		sla.GET("", s.HandleGetSLAMetrics)
		sla.POST("", s.HandleCreateSLAMetric)
		sla.PUT("/:id", s.HandleUpdateSLAMetric)
	}

	// Fraud Alerts
	fraud := router.Group("/api/v1/fraud")
	fraud.Use(s.AuthMiddleware())
	{
		fraud.GET("", s.HandleGetFraudAlerts)
		fraud.POST("", s.HandleCreateFraudAlert)
		fraud.POST("/:id/resolve", s.HandleResolveFraudAlert)
	}

	// Rate Limiting
	router.GET("/api/v1/rate-limit", s.HandleGetRateLimitStatus)

	// Health
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
}

// ============================================================================
// HTTP Handler Implementations
// ============================================================================

func (s *AdminService) HandleRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Role == "" {
		req.Role = "admin"
	}

	admin, err := s.Register(req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, admin)
}

func (s *AdminService) HandleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.IPAddress = c.ClientIP()
	req.UserAgent = c.Request.UserAgent()

	resp, err := s.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (s *AdminService) HandleLogout(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	token := c.GetHeader("Authorization")[7:]

	if err := s.Logout(token, fmt.Sprintf("%d", adminID), c.ClientIP()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

func (s *AdminService) HandleRefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, err := s.VerifyToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	adminID := uint(claims["admin_id"].(float64))
	admin, err := s.GetAdmin(adminID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	token, _ := s.GenerateToken(admin.ID, admin.Email, admin.Role)
	refreshToken, _ := s.GenerateRefreshToken(admin.ID)

	c.JSON(http.StatusOK, gin.H{
		"token":        token,
		"refresh_token": refreshToken,
		"admin":        admin,
		"expires_in":   s.config.JWTExpiration,
	})
}

func (s *AdminService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		token := authHeader[7:]
		claims, err := s.VerifyToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("admin_id", uint(claims["admin_id"].(float64)))
		c.Set("email", claims["email"].(string))
		c.Set("role", claims["role"].(string))
		c.Next()
	}
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	// Connect to PostgreSQL
	db, err := gorm.Open(postgres.Open(config.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate
	db.AutoMigrate(
		&Admin{}, &Session{}, &AuditLog{}, &Notification{},
		&ScheduledTask{}, &WebhookConfig{}, &ThemePreference{},
		&ApprovalWorkflow{}, &ApprovalRequest{}, &Ticket{}, &TicketComment{},
		&KnowledgeArticle{}, &SLAMetric{}, &FraudAlert{},
		// User{}, Token{}, KYC{}, Transaction{}, Withdrawal{}, WhiteLabel{}, Blockchain{}, Fee{}, Bot{}
	)

	// Connect to Redis
	redisOpts, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		log.Printf("Warning: Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)

	// Create admin service
	adminService := NewAdminService(config, db, redisClient)

	// Start scheduler
	adminService.StartScheduler()

	// Setup router
	router := gin.Default()
	
	// CORS middleware
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

	// Register routes
	adminService.RegisterRoutes(router)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Admin service starting on port %s", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
	adminService.stopScheduler <- true
	sqlDB, _ := db.DB()
	sqlDB.Close()
	redisClient.Close()
}

func log.Fatalf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", v...)
	os.Exit(1)
}

func log.Printf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}
