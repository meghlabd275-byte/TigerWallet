package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := loadConfig()

	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-auth"})
	})

	api := router.Group("/api/v1/auth")
	{
		// Public endpoints
		api.POST("/register", registerHandler)
		api.POST("/login", loginHandler)
		api.POST("/forgot-password", forgotPasswordHandler)
		api.POST("/reset-password", resetPasswordHandler)
		api.POST("/verify-email", verifyEmailHandler)
		api.POST("/resend-verification", resendVerificationHandler)

		// MFA endpoints
		api.POST("/mfa/setup", mfaSetupHandler)
		api.POST("/mfa/verify", mfaVerifyHandler)
		api.POST("/mfa/disable", mfaDisableHandler)
		api.POST("/mfa/backup-codes/generate", generateBackupCodesHandler)
		api.POST("/mfa/backup-codes/verify", verifyBackupCodeHandler)

		// OAuth
		api.GET("/oauth/google", oauthGoogleLoginHandler)
		api.GET("/oauth/google/callback", oauthGoogleCallbackHandler)
		api.GET("/oauth/github", oauthGithubLoginHandler)
		api.GET("/oauth/github/callback", oauthGithubCallbackHandler)

		// Password management
		api.POST("/change-password", changePasswordHandler)
		api.POST("/password-policy", getPasswordPolicyHandler)
	}

	// Protected endpoints
	protected := api.Group("")
	protected.Use(jwtAuthMiddleware(cfg))
	{
		protected.POST("/logout", logoutHandler)
		protected.POST("/logout-all", logoutAllDevicesHandler)
		protected.GET("/session", getSessionHandler)
		protected.GET("/sessions", listSessionsHandler)
		protected.DELETE("/sessions/:session_id", revokeSessionHandler)
		protected.GET("/me", getCurrentUserHandler)
		protected.PUT("/profile", updateProfileHandler)
		protected.POST("/refresh-token", refreshTokenHandler)

		// API Key management
		protected.GET("/api-keys", listAPIKeysHandler)
		protected.POST("/api-keys", createAPIKeyHandler)
		protected.DELETE("/api-keys/:key_id", revokeAPIKeyHandler)

		// RBAC - Role management
		protected.GET("/roles", listRolesHandler)
		protected.POST("/roles", createRoleHandler)
		protected.PUT("/roles/:role_id", updateRoleHandler)
		protected.DELETE("/roles/:role_id", deleteRoleHandler)

		// RBAC - Permission management
		protected.GET("/permissions", listPermissionsHandler)
		protected.POST("/roles/:role_id/permissions", assignPermissionsHandler)
		protected.DELETE("/roles/:role_id/permissions/:perm_id", removePermissionHandler)

		// RBAC - User role assignment
		protected.GET("/users/:user_id/roles", getUserRolesHandler)
		protected.POST("/users/:user_id/roles", assignUserRoleHandler)
		protected.DELETE("/users/:user_id/roles/:role_id", removeUserRoleHandler)

		// Tenant management
		protected.GET("/tenants", listTenantsHandler)
		protected.POST("/tenants", createTenantHandler)
		protected.PUT("/tenants/:tenant_id", updateTenantHandler)
		protected.DELETE("/tenants/:tenant_id", deleteTenantHandler)
		protected.GET("/tenants/:tenant_id/users", listTenantUsersHandler)
		protected.POST("/tenants/:tenant_id/invite", inviteUserHandler)
	}

	// Admin endpoints
	admin := api.Group("/admin")
	admin.Use(jwtAuthMiddleware(cfg), adminMiddleware())
	{
		admin.GET("/users", adminListUsersHandler)
		admin.GET("/users/:user_id", adminGetUserHandler)
		admin.PUT("/users/:user_id/status", adminUpdateUserStatusHandler)
		admin.DELETE("/users/:user_id", adminDeleteUserHandler)
		admin.GET("/audit-logs", adminGetAuditLogsHandler)
		admin.GET("/stats", adminGetStatsHandler)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	go func() {
		log.Printf("Auth service starting on port %s", cfg.Port)
		srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

type Config struct {
	Port         string
	Database     DatabaseConfig
	JWT          JWTConfig
	SMTP         SMTPConfig
	OAuth        OAuthConfig
	PasswordPolicy PasswordPolicyConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

type JWTConfig struct {
	Secret          string
	ExpiryHours     int
	RefreshExpiryDays int
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GithubClientID    string
	GithubClientSecret string
}

type PasswordPolicyConfig struct {
	MinLength       int
	RequireUppercase bool
	RequireLowercase bool
	RequireNumber    bool
	RequireSpecial   bool
	ExpiryDays       int
	HistoryCount    int
}

func loadConfig() *Config {
	return &Config{
		Port: getEnv("AUTH_PORT", "9001"),
		Database: DatabaseConfig{
			Host:     getEnv("AUTH_DB_HOST", "localhost"),
			Port:     getEnvInt("AUTH_DB_PORT", 5432),
			User:     getEnv("AUTH_DB_USER", "tigerwallet"),
			Password: getEnv("AUTH_DB_PASSWORD", "password"),
			DBName:   getEnv("AUTH_DB_NAME", "tigerwallet_auth"),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "tigerwallet-secret-key"),
			ExpiryHours:     getEnvInt("JWT_EXPIRY_HOURS", 24),
			RefreshExpiryDays: getEnvInt("JWT_REFRESH_EXPIRY_DAYS", 30),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "smtp.mailgun.org"),
			Port:     getEnvInt("SMTP_PORT", 587),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@tigerwallet.com"),
		},
		OAuth: OAuthConfig{
			GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			GithubClientID:    getEnv("GITHUB_CLIENT_ID", ""),
			GithubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		},
		PasswordPolicy: PasswordPolicyConfig{
			MinLength:       getEnvInt("PASSWORD_MIN_LENGTH", 8),
			RequireUppercase: getEnvBool("PASSWORD_REQUIRE_UPPERCASE", true),
			RequireLowercase: getEnvBool("PASSWORD_REQUIRE_LOWERCASE", true),
			RequireNumber:    getEnvBool("PASSWORD_REQUIRE_NUMBER", true),
			RequireSpecial:   getEnvBool("PASSWORD_REQUIRE_SPECIAL", true),
			ExpiryDays:       getEnvInt("PASSWORD_EXPIRY_DAYS", 90),
			HistoryCount:    getEnvInt("PASSWORD_HISTORY_COUNT", 5),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	var v int
	_, err := fmt.Sscan(os.Getenv(key), &v)
	if err != nil {
		return defaultValue
	}
	return v
}

func getEnvBool(key string, defaultValue bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	return v == "true" || v == "1"
}

// Models
type User struct {
	ID                uuid.UUID `json:"id" db:"id"`
	Email            string    `json:"email" db:"email"`
	PasswordHash     string    `json:"-" db:"password_hash"`
	FirstName       string    `json:"first_name" db:"first_name"`
	LastName        string    `json:"last_name" db:"last_name"`
	Phone           string    `json:"phone" db:"phone"`
	Avatar          string    `json:"avatar" db:"avatar"`
	Role            string    `json:"role" db:"role"` // super_admin, admin, user
	TenantID        *uuid.UUID `json:"tenant_id" db:"tenant_id"`
	EmailVerified   bool      `json:"email_verified" db:"email_verified"`
	PhoneVerified   bool      `json:"phone_verified" db:"phone_verified"`
	Status          string    `json:"status" db:"status"` // active, suspended, deactivated
	MFAEnabled     bool      `json:"mfa_enabled" db:"mfa_enabled"`
	MFASecret      string    `json:"-" db:"mfa_secret"`
	LastLoginAt    *time.Time `json:"last_login_at" db:"last_login_at"`
	FailedAttempts int       `json:"-" db:"failed_attempts"`
	LockedUntil   *time.Time `json:"-" db:"locked_until"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type Session struct {
	ID           uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Token       string    `json:"token" db:"token"`
	RefreshToken string   `json:"refresh_token" db:"refresh_token"`
	IPAddress   string    `json:"ip_address" db:"ip_address"`
	UserAgent   string    `json:"user_agent" db:"user_agent"`
	DeviceType  string    `json:"device_type" db:"device_type"`
	Location    string    `json:"location" db:"location"`
	ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type APIKey struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	TenantID   *uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name       string    `json:"name" db:"name"`
	KeyHash    string    `json:"-" db:"key_hash"`
	KeyPrefix  string    `json:"key_prefix" db:"key_prefix"`
	Scopes     []string  `json:"scopes" db:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at" db:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at" db:"last_used_at"`
	Status     string    `json:"status" db:"status"` // active, revoked
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type Role struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	TenantID   *uuid.UUID `json:"tenant_id" db:"tenant_id"`
	IsSystem   bool      `json:"is_system" db:"is_system"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type Permission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Resource    string    `json:"resource" db:"resource"`
	Action      string    `json:"action" db:"action"`
	Description string    `json:"description" db:"description"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type RolePermission struct {
	RoleID       uuid.UUID `json:"role_id" db:"role_id"`
	PermissionID uuid.UUID `json:"permission_id" db:"permission_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type UserRole struct {
	UserID   uuid.UUID `json:"user_id" db:"user_id"`
	RoleID   uuid.UUID `json:"role_id" db:"role_id"`
	AssignedBy uuid.UUID `json:"assigned_by" db:"assigned_by"`
	AssignedAt time.Time `json:"assigned_at" db:"assigned_at"`
}

type Tenant struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Domain      string    `json:"domain" db:"domain"`
	Plan        string    `json:"plan" db:"plan"` // free, basic, pro, enterprise
	Status      string    `json:"status" db:"status"` // active, suspended, trial
	Settings    string    `json:"settings" db:"settings"` // JSON
	OwnerID     uuid.UUID `json:"owner_id" db:"owner_id"`
	TrialEndsAt *time.Time `json:"trial_ends_at" db:"trial_ends_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type PasswordHistory struct {
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type AuditLog struct {
	ID         uuid.UUID `json:"id" db:"id"`
	UserID    *uuid.UUID `json:"user_id" db:"user_id"`
	TenantID *uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Action   string    `json:"action" db:"action"`
	Resource  string    `json:"resource" db:"resource"`
	Details   string    `json:"details" db:"details"`
	IPAddress string    `json:"ip_address" db:"ip_address"`
	UserAgent string   `json:"user_agent" db:"user_agent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Handlers
func registerHandler(c *gin.Context) {
	var req struct {
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required"`
		FirstName string `json:"first_name" binding:"required"`
		LastName  string `json:"last_name" binding:"required"`
		TenantID  string `json:"tenant_id"`
	}
	c.ShouldBindJSON(&req)

	// Validate password policy
	if err := validatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	if userExists(req.Email) {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Create user
	user := map[string]interface{}{
		"id":              uuid.New().String(),
		"email":          req.Email,
		"password_hash":   string(hashedPassword),
		"first_name":     req.FirstName,
		"last_name":      req.LastName,
		"role":           "user",
		"email_verified": false,
		"status":         "active",
		"created_at":     time.Now().Unix(),
	}

	// Send verification email
	go sendVerificationEmail(req.Email, user["id"].(string))

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful. Please verify your email.",
		"user":    user,
	})
}

func loginHandler(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
		MFACode  string `json:"mfa_code"`
	}
	c.ShouldBindJSON(&req)

	// Get user
	user := getUserByEmail(req.Email)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if account is locked
	if user["locked_until"] != nil && time.Now().Before(time.Unix(user["locked_until"].(int64), 0)) {
		c.JSON(http.StatusLocked, gin.H{"error": "Account is temporarily locked"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user["password_hash"].(string)), []byte(req.Password)); err != nil {
		// Increment failed attempts
		incrementFailedAttempts(user["id"].(string))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if MFA is enabled
	if user["mfa_enabled"].(bool) {
		if req.MFACode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "MFA code required", "mfa_required": true})
			return
		}
		if !verifyMFACode(user["mfa_secret"].(string), req.MFACode) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid MFA code"})
			return
		}
	}

	// Create session
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()
	session := createSession(user["id"].(string), ip, userAgent)

	// Generate tokens
	accessToken := generateAccessToken(user["id"].(string), user["role"].(string), user["tenant_id"])
	refreshToken := generateRefreshToken(user["id"].(string))

	// Update last login
	updateLastLogin(user["id"].(string))

	// Log audit
	createAuditLog(user["id"].(string), "user.login", "user", user["id"].(string), "", ip, userAgent)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    86400,
		"user": map[string]interface{}{
			"id":              user["id"],
			"email":          user["email"],
			"first_name":     user["first_name"],
			"last_name":      user["last_name"],
			"role":           user["role"],
			"tenant_id":      user["tenant_id"],
			"email_verified": user["email_verified"],
			"mfa_enabled":   user["mfa_enabled"],
		},
	})
}

func logoutHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	token := c.GetHeader("Authorization")

	revokeSession(userID, token)
	createAuditLog(userID, "user.logout", "session", "", "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func logoutAllDevicesHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	revokeAllSessions(userID)
	createAuditLog(userID, "user.logout_all", "session", userID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Logged out from all devices"})
}

func getSessionHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	session := getCurrentSession(userID)
	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func listSessionsHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	sessions := getUserSessions(userID)
	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

func revokeSessionHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	sessionID := c.Param("session_id")

	revokeSessionByID(userID, sessionID)
	createAuditLog(userID, "session.revoke", "session", sessionID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Session revoked"})
}

func getCurrentUserHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	user := getUserByID(userID)
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func updateProfileHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Phone     string `json:"phone"`
		Avatar    string `json:"avatar"`
	}
	c.ShouldBindJSON(&req)

	updateUserProfile(userID, req.FirstName, req.LastName, req.Phone, req.Avatar)
	createAuditLog(userID, "user.profile.update", "user", userID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated"})
}

func refreshTokenHandler(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	// Verify refresh token
	claims := verifyRefreshToken(req.RefreshToken)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	userID := claims["user_id"].(string)
	user := getUserByID(userID)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Generate new tokens
	accessToken := generateAccessToken(userID, user["role"].(string), user["tenant_id"])
	refreshToken := generateRefreshToken(userID)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    86400,
	})
}

// MFA Handlers
func mfaSetupHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	secret := generateMFASecret()
	qrCode := generateMFAQRCode(userID, secret)

	// Store secret temporarily (not enabled yet)
	storeMFASecret(userID, secret)

	c.JSON(http.StatusOK, gin.H{
		"secret": secret,
		"qr_code": qrCode,
		"message": "Scan the QR code with your authenticator app",
	})
}

func mfaVerifyHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		Code string `json:"code" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	secret := getMFASecret(userID)
	if secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA not set up"})
		return
	}

	if !verifyMFACode(secret, req.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid MFA code"})
		return
	}

	enableMFA(userID, secret)
	createAuditLog(userID, "user.mfa.enable", "user", userID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "MFA enabled successfully"})
}

func mfaDisableHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		Password string `json:"password" binding:"required"`
		Code     string `json:"code" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	// Verify password
	user := getUserByID(userID)
	if err := bcrypt.CompareHashAndPassword([]byte(user["password_hash"].(string)), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// Verify MFA code
	secret := user["mfa_secret"].(string)
	if !verifyMFACode(secret, req.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid MFA code"})
		return
	}

	disableMFA(userID)
	createAuditLog(userID, "user.mfa.disable", "user", userID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "MFA disabled successfully"})
}

func generateBackupCodesHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	codes := generateBackupCodes(userID)
	createAuditLog(userID, "user.backup_codes.generate", "user", userID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"codes": codes,
		"message": "Save these backup codes in a secure place",
	})
}

func verifyBackupCodeHandler(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	// Verify backup code
	valid := verifyBackupCode(req.Code)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid backup code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Backup code verified"})
}

// Password handlers
func changePasswordHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword    string `json:"new_password" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	// Validate new password
	if err := validatePassword(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify current password
	user := getUserByID(userID)
	if err := bcrypt.CompareHashAndPassword([]byte(user["password_hash"].(string)), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Check password history
	if isPasswordInHistory(userID, req.NewPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password was used recently. Choose a different password."})
		return
	}

	// Hash new password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)

	// Update password and save to history
	updatePassword(userID, string(hashedPassword))
	savePasswordHistory(userID, string(hashedPassword))

	createAuditLog(userID, "user.password.change", "user", userID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

func getPasswordPolicyHandler(c *gin.Context) {
	policy := map[string]interface{}{
		"min_length":        8,
		"require_uppercase": true,
		"require_lowercase": true,
		"require_number":    true,
		"require_special":   true,
		"expiry_days":      90,
	}
	c.JSON(http.StatusOK, policy)
}

func forgotPasswordHandler(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	c.ShouldBindJSON(&req)

	user := getUserByEmail(req.Email)
	if user != nil {
		token := generatePasswordResetToken(user["id"].(string))
		go sendPasswordResetEmail(req.Email, token)
	}

	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a password reset link has been sent"})
}

func resetPasswordHandler(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	// Validate password
	if err := validatePassword(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify token
	userID := verifyPasswordResetToken(req.Token)
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Hash new password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)

	// Update password
	updatePassword(userID, string(hashedPassword))
	savePasswordHistory(userID, string(hashedPassword))

	createAuditLog(userID, "user.password.reset", "user", userID, "", "", "")

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

func verifyEmailHandler(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	userID := verifyEmailToken(req.Token)
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
		return
	}

	verifyUserEmail(userID)
	createAuditLog(userID, "user.email.verify", "user", userID, "", "", "")

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

func resendVerificationHandler(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	c.ShouldBindJSON(&req)

	user := getUserByEmail(req.Email)
	if user == nil {
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a verification link has been sent"})
		return
	}

	if user["email_verified"].(bool) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already verified"})
		return
	}

	token := generateEmailVerificationToken(user["id"].(string))
	go sendVerificationEmail(req.Email, token)

	c.JSON(http.StatusOK, gin.H{"message": "Verification email sent"})
}

// API Key handlers
func listAPIKeysHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	keys := getUserAPIKeys(userID)
	c.JSON(http.StatusOK, gin.H{"api_keys": keys})
}

func createAPIKeyHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		Name      string   `json:"name" binding:"required"`
		Scopes    []string `json:"scopes" binding:"required"`
		ExpiresAt string   `json:"expires_at"`
	}
	c.ShouldBindJSON(&req)

	key, keyID := generateAPIKey()

	// Hash and store key
	keyHash := hashAPIKey(key)
	storeAPIKey(userID, keyID, keyHash, req.Name, req.Scopes, req.ExpiresAt)

	createAuditLog(userID, "api_key.create", "api_key", keyID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, gin.H{
		"api_key": key,
		"key_id":  keyID,
		"name":    req.Name,
		"scopes":  req.Scopes,
	})
}

func revokeAPIKeyHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	keyID := c.Param("key_id")

	revokeAPIKey(userID, keyID)
	createAuditLog(userID, "api_key.revoke", "api_key", keyID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}

// Role handlers
func listRolesHandler(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	roles := getRoles(tenantID)
	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

func createRoleHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	tenantID := c.GetString("tenant_id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	c.ShouldBindJSON(&req)

	roleID := createRole(req.Name, req.Description, tenantID)
	createAuditLog(userID, "role.create", "role", roleID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, gin.H{"role_id": roleID})
}

func updateRoleHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	roleID := c.Param("role_id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	c.ShouldBindJSON(&req)

	updateRole(roleID, req.Name, req.Description)
	createAuditLog(userID, "role.update", "role", roleID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Role updated"})
}

func deleteRoleHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	roleID := c.Param("role_id")

	deleteRole(roleID)
	createAuditLog(userID, "role.delete", "role", roleID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Role deleted"})
}

func listPermissionsHandler(c *gin.Context) {
	permissions := getAllPermissions()
	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

func assignPermissionsHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	roleID := c.Param("role_id")

	var req struct {
		Permissions []string `json:"permissions" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	for _, permID := range req.Permissions {
		assignPermissionToRole(roleID, permID)
	}

	createAuditLog(userID, "role.permission.assign", "role", roleID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Permissions assigned"})
}

func removePermissionHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	roleID := c.Param("role_id")
	permID := c.Param("perm_id")

	removePermissionFromRole(roleID, permID)
	createAuditLog(userID, "role.permission.remove", "role", roleID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Permission removed"})
}

func getUserRolesHandler(c *gin.Context) {
	userID := c.Param("user_id")

	roles := getUserRoles(userID)
	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

func assignUserRoleHandler(c *gin.Context) {
	assignerID := c.GetString("user_id")
	userID := c.Param("user_id")

	var req struct {
		RoleID string `json:"role_id" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	assignRoleToUser(userID, req.RoleID, assignerID)
	createAuditLog(assignerID, "user.role.assign", "user", userID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Role assigned"})
}

func removeUserRoleHandler(c *gin.Context) {
	removerID := c.GetString("user_id")
	userID := c.Param("user_id")
	roleID := c.Param("role_id")

	removeRoleFromUser(userID, roleID)
	createAuditLog(removerID, "user.role.remove", "user", userID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Role removed"})
}

// Tenant handlers
func listTenantsHandler(c *gin.Context) {
	tenants := getAllTenants()
	c.JSON(http.StatusOK, gin.H{"tenants": tenants})
}

func createTenantHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		Name   string `json:"name" binding:"required"`
		Slug   string `json:"slug" binding:"required"`
		Domain string `json:"domain"`
	}
	c.ShouldBindJSON(&req)

	tenantID := createTenant(req.Name, req.Slug, req.Domain, userID)
	createAuditLog(userID, "tenant.create", "tenant", tenantID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, gin.H{"tenant_id": tenantID})
}

func updateTenantHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	tenantID := c.Param("tenant_id")

	var req struct {
		Name   string `json:"name"`
		Domain string `json:"domain"`
		Status string `json:"status"`
	}
	c.ShouldBindJSON(&req)

	updateTenant(tenantID, req.Name, req.Domain, req.Status)
	createAuditLog(userID, "tenant.update", "tenant", tenantID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Tenant updated"})
}

func deleteTenantHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	tenantID := c.Param("tenant_id")

	deleteTenant(tenantID)
	createAuditLog(userID, "tenant.delete", "tenant", tenantID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Tenant deleted"})
}

func listTenantUsersHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	users := getTenantUsers(tenantID)
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func inviteUserHandler(c *gin.Context) {
	inviterID := c.GetString("user_id")
	tenantID := c.Param("tenant_id")

	var req struct {
		Email string `json:"email" binding:"required,email"`
		Role  string `json:"role" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	token := generateInviteToken(req.Email, tenantID, req.Role)
	go sendInviteEmail(req.Email, token, inviterID)

	createAuditLog(inviterID, "tenant.user.invite", "tenant", tenantID, req.Email, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Invitation sent"})
}

// Admin handlers
func adminListUsersHandler(c *gin.Context) {
	users := adminGetAllUsers()
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func adminGetUserHandler(c *gin.Context) {
	userID := c.Param("user_id")

	user := getUserByID(userID)
	c.JSON(http.StatusOK, user)
}

func adminUpdateUserStatusHandler(c *gin.Context) {
	adminID := c.GetString("user_id")
	userID := c.Param("user_id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	updateUserStatus(userID, req.Status)
	createAuditLog(adminID, "admin.user.status", "user", userID, req.Status, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "User status updated"})
}

func adminDeleteUserHandler(c *gin.Context) {
	adminID := c.GetString("user_id")
	userID := c.Param("user_id")

	deleteUser(userID)
	createAuditLog(adminID, "admin.user.delete", "user", userID, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func adminGetAuditLogsHandler(c *gin.Context) {
	logs := adminGetLogs()
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func adminGetStatsHandler(c *gin.Context) {
	stats := map[string]interface{}{
		"total_users":       1000,
		"active_users":      850,
		"total_tenants":    100,
		"active_sessions":   500,
		"api_keys_created":  250,
		"login_today":      150,
		"registrations_today": 10,
	}
	c.JSON(http.StatusOK, stats)
}

// OAuth handlers
func oauthGoogleLoginHandler(c *gin.Context) {
	state := generateOAuthState()
	url := "https://accounts.google.com/o/oauth2/v2/auth?" +
		"client_id=" + getEnv("GOOGLE_CLIENT_ID", "") +
		"&redirect_uri=" + getEnv("GOOGLE_REDIRECT_URI", "") +
		"&response_type=code" +
		"&scope=openid%20email%20profile" +
		"&state=" + state

	c.Redirect(http.StatusTemporaryRedirect, url)
}

func oauthGoogleCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	// Exchange code for token
	tokenURL := "https://oauth2.googleapis.com/token"
	// ... implement token exchange

	c.JSON(http.StatusOK, gin.H{"message": "Google OAuth callback"})
}

func oauthGithubLoginHandler(c *gin.Context) {
	state := generateOAuthState()
	url := "https://github.com/login/oauth/authorize?" +
		"client_id=" + getEnv("GITHUB_CLIENT_ID", "") +
		"&redirect_uri=" + getEnv("GITHUB_REDIRECT_URI", "") +
		"&scope=read:user" +
		"&state=" + state

	c.Redirect(http.StatusTemporaryRedirect, url)
}

func oauthGithubCallbackHandler(c *gin.Context) {
	code := c.Query("code")

	// Exchange code for token
	// ... implement token exchange

	c.JSON(http.StatusOK, gin.H{"message": "GitHub OAuth callback"})
}

// Middleware
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func jwtAuthMiddleware(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		c.Set("user_id", claims["user_id"])
		c.Set("role", claims["role"])
		if claims["tenant_id"] != nil {
			c.Set("tenant_id", claims["tenant_id"])
		}

		c.Next()
	}
}

func adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role != "super_admin" && role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// Helper functions
func generateAccessToken(userID, role string, tenantID interface{}) string {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"role":     role,
		"tenant_id": tenantID,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
		"iat":      time.Now().Unix(),
		"type":     "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(getEnv("JWT_SECRET", "tigerwallet-secret-key")))
	return tokenString
}

func generateRefreshToken(userID string) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 30).Unix(),
		"iat":     time.Now().Unix(),
		"type":    "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(getEnv("JWT_SECRET", "tigerwallet-secret-key")+"_refresh"))
	return tokenString
}

func verifyRefreshToken(tokenString string) jwt.MapClaims {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(getEnv("JWT_SECRET", "tigerwallet-secret-key") + "_refresh"), nil
	})
	if err != nil || !token.Valid {
		return nil
	}
	return token.Claims.(jwt.MapClaims)
}

func validatePassword(password string) error {
	policy := loadConfig().PasswordPolicy

	if len(password) < policy.MinLength {
		return fmt.Errorf("password must be at least %d characters", policy.MinLength)
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		default:
			hasSpecial = true
		}
	}

	if policy.RequireUppercase && !hasUpper {
		return fmt.Errorf("password must contain uppercase letter")
	}
	if policy.RequireLowercase && !hasLower {
		return fmt.Errorf("password must contain lowercase letter")
	}
	if policy.RequireNumber && !hasNumber {
		return fmt.Errorf("password must contain number")
	}
	if policy.RequireSpecial && !hasSpecial {
		return fmt.Errorf("password must contain special character")
	}

	return nil
}

func generateMFASecret() string {
	secret := make([]byte, 20)
	rand.Read(secret)
	return base64.StdEncoding.EncodeToString(secret)
}

func generateMFAQRCode(userID, secret string) string {
	return "otpauth://totp/TigerWallet:" + userID + "?secret=" + secret + "&issuer=TigerWallet"
}

func verifyMFACode(secret, code string) bool {
	// In production, use proper TOTP verification
	return len(code) == 6
}

func generateBackupCodes(userID string) []string {
	codes := make([]string, 10)
	for i := 0; i < 10; i++ {
		code := make([]byte, 4)
		rand.Read(code)
		codes[i] = strings.ToUpper(hex.EncodeToString(code))
	}
	return codes
}

func verifyBackupCode(code string) bool {
	return len(code) == 8
}

func generatePasswordResetToken(userID string) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "password_reset",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(getEnv("JWT_SECRET", "tigerwallet-secret-key") + "_reset"))
	return tokenString
}

func verifyPasswordResetToken(tokenString string) string {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(getEnv("JWT_SECRET", "tigerwallet-secret-key") + "_reset"), nil
	})
	if err != nil || !token.Valid {
		return ""
	}
	claims := token.Claims.(jwt.MapClaims)
	if claims["type"] != "password_reset" {
		return ""
	}
	return claims["user_id"].(string)
}

func generateEmailVerificationToken(userID string) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "email_verify",
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(getEnv("JWT_SECRET", "tigerwallet-secret-key") + "_verify"))
	return tokenString
}

func verifyEmailToken(tokenString string) string {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(getEnv("JWT_SECRET", "tigerwallet-secret-key") + "_verify"), nil
	})
	if err != nil || !token.Valid {
		return ""
	}
	claims := token.Claims.(jwt.MapClaims)
	if claims["type"] != "email_verify" {
		return ""
	}
	return claims["user_id"].(string)
}

func generateInviteToken(email, tenantID, role string) string {
	claims := jwt.MapClaims{
		"email":    email,
		"tenant_id": tenantID,
		"role":     role,
		"type":     "invite",
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(getEnv("JWT_SECRET", "tigerwallet-secret-key") + "_invite"))
	return tokenString
}

func generateOAuthState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateAPIKey() (string, string) {
	key := make([]byte, 32)
	rand.Read(key)
	keyID := uuid.New().String()
	return hex.EncodeToString(key), keyID
}

func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func sendVerificationEmail(email, token string) {
	log.Printf("Sending verification email to %s", email)
}

func sendPasswordResetEmail(email, token string) {
	log.Printf("Sending password reset email to %s", email)
}

func sendInviteEmail(email, token, inviterID string) {
	log.Printf("Sending invite email to %s", email)
}

// Database stubs
type DB struct{}

func initDatabase(cfg *Config) (*DB, error) {
	log.Printf("Connecting to PostgreSQL at %s:%d", cfg.Database.Host, cfg.Database.Port)
	return &DB{}, nil
}

func (d *DB) Close() {}

func userExists(email string) bool { return false }
func getUserByEmail(email string) map[string]interface{} { return nil }
func getUserByID(id string) map[string]interface{} { return nil }
func incrementFailedAttempts(userID string) {}
func createSession(userID, ip, userAgent string) map[string]interface{} { return nil }
func getCurrentSession(userID string) map[string]interface{} { return nil }
func getUserSessions(userID string) []map[string]interface{} { return nil }
func revokeSession(userID, token string) {}
func revokeAllSessions(userID string) {}
func revokeSessionByID(userID, sessionID string) {}
func updateLastLogin(userID string) {}
func updateUserProfile(userID, firstName, lastName, phone, avatar string) {}
func getMFASecret(userID string) string { return "" }
func storeMFASecret(userID, secret string) {}
func enableMFA(userID, secret string) {}
func disableMFA(userID string) {}
func isPasswordInHistory(userID, password string) bool { return false }
func updatePassword(userID, hash string) {}
func savePasswordHistory(userID, hash string) {}
func verifyUserEmail(userID string) {}
func getUserAPIKeys(userID string) []map[string]interface{} { return nil }
func storeAPIKey(userID, keyID, keyHash, name, scopes, expiresAt string) {}
func revokeAPIKey(userID, keyID string) {}
func getRoles(tenantID string) []map[string]interface{} { return nil }
func createRole(name, description, tenantID string) string { return "" }
func updateRole(roleID, name, description string) {}
func deleteRole(roleID string) {}
func getAllPermissions() []map[string]interface{} { return nil }
func assignPermissionToRole(roleID, permID string) {}
func removePermissionFromRole(roleID, permID string) {}
func getUserRoles(userID string) []map[string]interface{} { return nil }
func assignRoleToUser(userID, roleID, assignedBy string) {}
func removeRoleFromUser(userID, roleID string) {}
func getAllTenants() []map[string]interface{} { return nil }
func createTenant(name, slug, domain, ownerID string) string { return "" }
func updateTenant(tenantID, name, domain, status string) {}
func deleteTenant(tenantID string) {}
func getTenantUsers(tenantID string) []map[string]interface{} { return nil }
func adminGetAllUsers() []map[string]interface{} { return nil }
func updateUserStatus(userID, status string) {}
func deleteUser(userID string) {}
func adminGetLogs() []map[string]interface{} { return nil }
func createAuditLog(userID, action, resource, details, ip, userAgent string) {}
