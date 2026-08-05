// TigerWallet Admin System - Main Entry Point
// High-Loaded Worldwide Distributed System
// Built with Go for maximum performance and scalability

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

// ============================ CONSTANTS ============================

const (
	ServiceName        = "tiger-admin-system"
	ServiceVersion     = "2.0.0"
	DefaultPort        = "8090"
	MaxRequestSize     = 10 * 1024 * 1024
	ReadTimeout        = 30 * time.Second
	WriteTimeout       = 30 * time.Second
	IdleTimeout        = 120 * time.Second
	SessionDuration    = 24 * time.Hour
	PasswordMinLength  = 8
	MaxLoginAttempts   = 5
	RateLimitRequests  = 100
	RateLimitDuration  = time.Minute
)

// ============================ TYPES ============================

type Config struct {
	DatabaseURL      string
	RedisURL         string
	Port             string
	JWTSecret        string
	Environment      string
	SessionDuration  time.Duration
	MaxLoginAttempts int
	RateLimit        int
}

type AdminSystemUser struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	Username        string    `json:"username"`
	PasswordHash    string    `json:"-"`
	Role            string    `json:"role"`
	Permissions     []string  `json:"permissions"`
	IsActive        bool      `json:"is_active"`
	IsSuperAdmin    bool      `json:"is_super_admin"`
	WhiteLabelID    string    `json:"white_label_id,omitempty"`
	TwoFactorEnabled bool     `json:"two_factor_enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LastLogin       time.Time `json:"last_login"`
	FailedAttempts  int       `json:"failed_attempts"`
	LockedUntil     time.Time `json:"locked_until"`
}

type AuditLog struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	UserEmail   string    `json:"user_email"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	ResourceID  string    `json:"resource_id"`
	Details     string    `json:"details"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type SystemConfig struct {
	ID            string    `json:"id"`
	Key           string    `json:"key"`
	Value         string    `json:"value"`
	Description   string    `json:"description"`
	IsEncrypted   bool      `json:"is_encrypted"`
	Category      string    `json:"category"`
	UpdatedAt     time.Time `json:"updated_at"`
	UpdatedBy     string    `json:"updated_by"`
}

type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	IsRead    bool      `json:"is_read"`
	Priority  string    `json:"priority"`
	ActionURL string    `json:"action_url"`
	CreatedAt time.Time `json:"created_at"`
}

type FeatureFlag struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	IsEnabled      bool      `json:"is_enabled"`
	RolloutPercent int       `json:"rollout_percent"`
	TargetRoles    []string  `json:"target_roles"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type APIKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Key         string    `json:"key"`
	Secret      string    `json:"-"`
	Permissions []string  `json:"permissions"`
	IsActive    bool      `json:"is_active"`
	ExpiresAt   time.Time `json:"expires_at"`
	LastUsed    time.Time `json:"last_used"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
}

// ============================ GLOBALS ============================

var (
	logger      zerolog.Logger
	dbPool      *pgxpool.Pool
	redisClient *redis.Client
	config      Config
	ctx         context.Context
)

// ============================ INITIALIZATION ============================

func init() {
	config = Config{
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://tigerwallet:securepassword@postgres:5432/tigerwallet_admin_system"),
		RedisURL:         getEnv("REDIS_URL", "redis://redis:6379"),
		Port:             getEnv("PORT", DefaultPort),
		JWTSecret:        getEnv("JWT_SECRET", "tiger-admin-system-secret-key"),
		Environment:      getEnv("ENVIRONMENT", "production"),
		SessionDuration:  SessionDuration,
		MaxLoginAttempts: MaxLoginAttempts,
		RateLimit:        RateLimitRequests,
	}

	zerolog.TimeFieldFormat = time.RFC3339
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger = zerolog.New(output).With().Str("service", ServiceName).Str("version", ServiceVersion).Timestamp().Logger()
	
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	logger.Info().Msg("Initializing TigerWallet Admin System")
	ctx = context.Background()
}

// ============================ MAIN ============================

func main() {
	if err := initializeDatabase(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer dbPool.Close()

	if err := initializeRedis(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize Redis")
	}
	defer redisClient.Close()

	if err := runMigrations(); err != nil {
		logger.Error().Err(err).Msg("Migration warning - continuing")
	}

	if err := createDefaultSuperAdmin(); err != nil {
		logger.Error().Err(err).Msg("Warning: Failed to create default super admin")
	}

	router := initializeRouter()
	
	startMetricsCollection()

	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Admin System server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}
	
	logger.Info().Msg("Server exited gracefully")
}

// ============================ DATABASE ============================

func initializeDatabase() error {
	var err error
	dbPool, err = pgxpool.Connect(ctx, config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := dbPool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().Msg("Database connection established")
	return nil
}

func initializeRedis() error {
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	
	redisClient = redis.NewClient(opt)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info().Msg("Redis connection established")
	return nil
}

func runMigrations() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS admin_system_users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(50) NOT NULL DEFAULT 'admin',
			permissions JSONB DEFAULT '[]',
			is_active BOOLEAN DEFAULT true,
			is_super_admin BOOLEAN DEFAULT false,
			white_label_id VARCHAR(100),
			two_factor_enabled BOOLEAN DEFAULT false,
			failed_attempts INTEGER DEFAULT 0,
			locked_until TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			last_login TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_configs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			key VARCHAR(100) UNIQUE NOT NULL,
			value TEXT NOT NULL,
			description TEXT,
			is_encrypted BOOLEAN DEFAULT false,
			category VARCHAR(50) DEFAULT 'general',
			updated_at TIMESTAMP DEFAULT NOW(),
			updated_by UUID
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID,
			user_email VARCHAR(255),
			action VARCHAR(100) NOT NULL,
			resource VARCHAR(100) NOT NULL,
			resource_id VARCHAR(100),
			details TEXT,
			ip_address VARCHAR(45),
			user_agent TEXT,
			status VARCHAR(20) DEFAULT 'success',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS system_metrics (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			metric_name VARCHAR(100) NOT NULL,
			value DOUBLE PRECISION NOT NULL,
			unit VARCHAR(20),
			tags JSONB DEFAULT '{}',
			timestamp TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID,
			title VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			type VARCHAR(50) DEFAULT 'info',
			is_read BOOLEAN DEFAULT false,
			priority VARCHAR(20) DEFAULT 'normal',
			action_url VARCHAR(500),
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS feature_flags (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) UNIQUE NOT NULL,
			description TEXT,
			is_enabled BOOLEAN DEFAULT false,
			rollout_percent INTEGER DEFAULT 0,
			target_roles JSONB DEFAULT '[]',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			key VARCHAR(100) UNIQUE NOT NULL,
			secret_hash VARCHAR(255) NOT NULL,
			permissions JSONB DEFAULT '[]',
			is_active BOOLEAN DEFAULT true,
			expires_at TIMESTAMP,
			last_used TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			created_by UUID
		)`,
	}

	for _, migration := range migrations {
		if _, err := dbPool.Exec(ctx, migration); err != nil {
			return err
		}
	}

	logger.Info().Msg("Database migrations completed")
	return nil
}

func createDefaultSuperAdmin() error {
	var count int
	err := dbPool.QueryRow(ctx, "SELECT COUNT(*) FROM admin_system_users WHERE is_super_admin = true").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("SuperAdmin@123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = dbPool.Exec(ctx, `
		INSERT INTO admin_system_users (email, username, password_hash, role, permissions, is_active, is_super_admin)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "super@tigerwallet.com", "superadmin", string(hashedPassword), "super_admin", `["*"]`, true, true)

	if err != nil {
		return err
	}

	logger.Info().Msg("Default super admin created")
	return nil
}

// ============================ ROUTER ============================

func initializeRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	router.Use(corsMiddleware())
	router.Use(rateLimitMiddleware())
	router.Use(requestIDMiddleware())

	router.GET("/health", handleHealthCheck)
	router.GET("/metrics", handleMetrics)

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/login", handleLogin)
			auth.POST("/logout", handleLogout)
			auth.POST("/refresh", handleRefreshToken)
			auth.POST("/forgot-password", handleForgotPassword)
			auth.POST("/reset-password", handleResetPassword)
		}

		admin := v1.Group("")
		admin.Use(authMiddleware())
		{
			admin.GET("/dashboard", handleDashboard)
			admin.GET("/dashboard/stats", handleDashboardStats)

			users := admin.Group("/users")
			{
				users.GET("", handleListUsers)
				users.GET("/:id", handleGetUser)
				users.POST("", handleCreateUser)
				users.PUT("/:id", handleUpdateUser)
				users.DELETE("/:id", handleDeleteUser)
				users.POST("/:id/activate", handleActivateUser)
				users.POST("/:id/deactivate", handleDeactivateUser)
				users.POST("/:id/reset-password", handleResetUserPassword)
				users.PUT("/:id/permissions", handleUpdateUserPermissions)
			}

			config := admin.Group("/config")
			{
				config.GET("", handleListConfig)
				config.GET("/:key", handleGetConfig)
				config.PUT("", handleUpdateConfig)
				config.DELETE("/:key", handleDeleteConfig)
			}

			audit := admin.Group("/audit-logs")
			{
				audit.GET("", handleListAuditLogs)
				audit.GET("/export", handleExportAuditLogs)
			}

			notifications := admin.Group("/notifications")
			{
				notifications.GET("", handleListNotifications)
				notifications.PUT("/:id/read", handleMarkNotificationRead)
				notifications.PUT("/read-all", handleMarkAllNotificationsRead)
				notifications.DELETE("/:id", handleDeleteNotification)
			}

			features := admin.Group("/features")
			{
				features.GET("", handleListFeatureFlags)
				features.POST("", handleCreateFeatureFlag)
				features.PUT("/:id", handleUpdateFeatureFlag)
				features.DELETE("/:id", handleDeleteFeatureFlag)
			}

			apiKeys := admin.Group("/api-keys")
			{
				apiKeys.GET("", handleListAPIKeys)
				apiKeys.POST("", handleCreateAPIKey)
				apiKeys.DELETE("/:id", handleDeleteAPIKey)
				apiKeys.POST("/:id/rotate", handleRotateAPIKey)
			}

			metrics := admin.Group("/system")
			{
				metrics.GET("/status", handleSystemStatus)
				metrics.GET("/metrics", handleSystemMetrics)
				metrics.GET("/health", handleDetailedHealth)
			}
		}
	}

	return router
}

// ============================ MIDDLEWARE ============================

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		token := extractToken(authHeader)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		claims, err := verifyToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("user_email", claims["email"])
		c.Set("user_role", claims["role"])
		c.Set("permissions", claims["permissions"])
		c.Next()
	}
}

// ============================ AUTH HANDLERS ============================

func handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user AdminSystemUser
	err := dbPool.QueryRow(ctx, `
		SELECT id, email, username, password_hash, role, permissions, is_active, is_super_admin, failed_attempts, locked_until
		FROM admin_system_users WHERE email = $1
	`, req.Email).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.Permissions, &user.IsActive, &user.IsSuperAdmin, &user.FailedAttempts, &user.LockedUntil)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	if err != nil {
		logger.Error().Err(err).Msg("Login error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if user.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusLocked, gin.H{"error": "Account is locked"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		dbPool.Exec(ctx, `
			UPDATE admin_system_users 
			SET failed_attempts = failed_attempts + 1,
				locked_until = CASE WHEN failed_attempts + 1 >= $1 THEN NOW() + INTERVAL '30 minutes' ELSE NULL END
			WHERE id = $2
		`, config.MaxLoginAttempts, user.ID)

		logAudit(user.ID, user.Email, "LOGIN_FAILED", "user", user.ID, "Invalid password", c)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is deactivated"})
		return
	}

	dbPool.Exec(ctx, "UPDATE admin_system_users SET failed_attempts = 0, locked_until = NULL, last_login = NOW() WHERE id = $1", user.ID)

	token, err := generateToken(user.ID, user.Email, user.Role, user.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	sessionData, _ := json.Marshal(map[string]interface{}{
		"user_id":    user.ID,
		"email":      user.Email,
		"role":       user.Role,
		"permissions": user.Permissions,
		"expires":    time.Now().Add(config.SessionDuration).Unix(),
	})
	redisClient.Set(ctx, "session:"+token, sessionData, config.SessionDuration)

	logAudit(user.ID, user.Email, "LOGIN_SUCCESS", "user", user.ID, "User logged in", c)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":              user.ID,
			"email":           user.Email,
			"username":        user.Username,
			"role":            user.Role,
			"permissions":     user.Permissions,
			"is_super_admin":  user.IsSuperAdmin,
		},
	})
}

func handleLogout(c *gin.Context) {
	token := extractToken(c.GetHeader("Authorization"))
	if token != "" {
		redisClient.Del(ctx, "session:"+token)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func handleRefreshToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, err := verifyToken(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	newToken, err := generateToken(
		claims["user_id"].(string),
		claims["email"].(string),
		claims["role"].(string),
		claims["permissions"].([]string),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": newToken})
}

func handleForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resetToken := generateID()
	redisClient.Set(ctx, "password_reset:"+resetToken, req.Email, 1*time.Hour)

	c.JSON(http.StatusOK, gin.H{"message": "Password reset link sent", "reset_token": resetToken})
}

func handleResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email, err := redisClient.Get(ctx, "password_reset:"+req.Token).Result()
	if err == redis.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	_, err = dbPool.Exec(ctx, "UPDATE admin_system_users SET password_hash = $1, updated_at = NOW() WHERE email = $2", string(hashedPassword), email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	redisClient.Del(ctx, "password_reset:"+req.Token)

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// ============================ USER HANDLERS ============================

func handleListUsers(c *gin.Context) {
	page := getIntParam(c, "page", 1)
	limit := getIntParam(c, "limit", 20)
	offset := (page - 1) * limit

	rows, err := dbPool.Query(ctx, `
		SELECT id, email, username, role, permissions, is_active, is_super_admin, created_at, last_login 
		FROM admin_system_users ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []gin.H
	for rows.Next() {
		var u AdminSystemUser
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.Role, &u.Permissions, &u.IsActive, &u.IsSuperAdmin, &u.CreatedAt, &u.LastLogin); err != nil {
			continue
		}
		users = append(users, gin.H{
			"id":             u.ID,
			"email":          u.Email,
			"username":       u.Username,
			"role":           u.Role,
			"permissions":    u.Permissions,
			"is_active":      u.IsActive,
			"is_super_admin": u.IsSuperAdmin,
			"created_at":     u.CreatedAt,
			"last_login":     u.LastLogin,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": users, "page": page, "limit": limit})
}

func handleGetUser(c *gin.Context) {
	id := c.Param("id")

	var u AdminSystemUser
	err := dbPool.QueryRow(ctx, `
		SELECT id, email, username, role, permissions, is_active, is_super_admin, white_label_id, two_factor_enabled, created_at, last_login
		FROM admin_system_users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.Username, &u.Role, &u.Permissions, &u.IsActive, &u.IsSuperAdmin, &u.WhiteLabelID, &u.TwoFactorEnabled, &u.CreatedAt, &u.LastLogin)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": u})
}

func handleCreateUser(c *gin.Context) {
	var req struct {
		Email       string   `json:"email" binding:"required,email"`
		Username    string   `json:"username" binding:"required,min=3,max=50"`
		Password    string   `json:"password" binding:"required,min=8"`
		Role        string   `json:"role" binding:"required"`
		Permissions []string `json:"permissions"`
		WhiteLabelID string  `json:"white_label_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var exists bool
	err := dbPool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM admin_system_users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil || exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	permissions := getPermissionsForRole(req.Role)
	if len(req.Permissions) > 0 {
		permissions = req.Permissions
	}

	userID := generateID()
	_, err = dbPool.Exec(ctx, `
		INSERT INTO admin_system_users (id, email, username, password_hash, role, permissions, white_label_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, userID, req.Email, req.Username, string(hashedPassword), req.Role, permissions, req.WhiteLabelID, true)

	if err != nil {
		logger.Error().Err(err).Msg("Failed to create user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	logAudit(c.GetString("user_id"), c.GetString("user_email"), "CREATE_USER", "user", userID, fmt.Sprintf("Created user: %s", req.Email), c)

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully", "user_id": userID})
}

func handleUpdateUser(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Email      string `json:"email"`
		Username   string `json:"username"`
		Role       string `json:"role"`
		WhiteLabelID string `json:"white_label_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := dbPool.Exec(ctx, `
		UPDATE admin_system_users 
		SET email = COALESCE(NULLIF($1, ''), email),
			username = COALESCE(NULLIF($2, ''), username),
			role = COALESCE(NULLIF($3, ''), role),
			white_label_id = COALESCE(NULLIF($4, ''), white_label_id),
			updated_at = NOW()
		WHERE id = $5
	`, req.Email, req.Username, req.Role, req.WhiteLabelID, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	logAudit(c.GetString("user_id"), c.GetString("user_email"), "UPDATE_USER", "user", id, "User updated", c)

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

func handleDeleteUser(c *gin.Context) {
	id := c.Param("id")

	if id == c.GetString("user_id") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete yourself"})
		return
	}

	_, err := dbPool.Exec(ctx, "DELETE FROM admin_system_users WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	logAudit(c.GetString("user_id"), c.GetString("user_email"), "DELETE_USER", "user", id, "User deleted", c)

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func handleActivateUser(c *gin.Context) {
	id := c.Param("id")

	_, err := dbPool.Exec(ctx, "UPDATE admin_system_users SET is_active = true, updated_at = NOW() WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate user"})
		return
	}

	logAudit(c.GetString("user_id"), c.GetString("user_email"), "ACTIVATE_USER", "user", id, "User activated", c)

	c.JSON(http.StatusOK, gin.H{"message": "User activated successfully"})
}

func handleDeactivateUser(c *gin.Context) {
	id := c.Param("id")

	if id == c.GetString("user_id") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot deactivate yourself"})
		return
	}

	_, err := dbPool.Exec(ctx, "UPDATE admin_system_users SET is_active = false, updated_at = NOW() WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate user"})
		return
	}

	logAudit(c.GetString("user_id"), c.GetString("user_email"), "DEACTIVATE_USER", "user", id, "User deactivated", c)

	c.JSON(http.StatusOK, gin.H{"message": "User deactivated successfully"})
}

func handleResetUserPassword(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	_, err = dbPool.Exec(ctx, "UPDATE admin_system_users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(hashedPassword), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	logAudit(c.GetString("user_id"), c.GetString("user_email"), "RESET_USER_PASSWORD", "user", id, "Password reset", c)

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

func handleUpdateUserPermissions(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Permissions []string `json:"permissions" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	permissionsJSON, _ := json.Marshal(req.Permissions)

	_, err := dbPool.Exec(ctx, "UPDATE admin_system_users SET permissions = $1, updated_at = NOW() WHERE id = $2", string(permissionsJSON), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update permissions"})
		return
	}

	logAudit(c.GetString("user_id"), c.GetString("user_email"), "UPDATE_PERMISSIONS", "user", id, "Permissions updated", c)

	c.JSON(http.StatusOK, gin.H{"message": "Permissions updated successfully"})
}

// ============================ CONFIG HANDLERS ============================

func handleListConfig(c *gin.Context) {
	rows, err := dbPool.Query(ctx, "SELECT id, key, value, description, is_encrypted, category, updated_at FROM system_configs ORDER BY category, key")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch config"})
		return
	}
	defer rows.Close()

	var configs []SystemConfig
	for rows.Next() {
		var c SystemConfig
		if err := rows.Scan(&c.ID, &c.Key, &c.Value, &c.Description, &c.IsEncrypted, &c.Category, &c.UpdatedAt); err != nil {
			continue
		}
		configs = append(configs, c)
	}

	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

func handleGetConfig(c *gin.Context) {
	key := c.Param("key")

	var cfg SystemConfig
	err := dbPool.QueryRow(ctx, "SELECT id, key, value, description, is_encrypted, category, updated_at FROM system_configs WHERE key = $1", key).
		Scan(&cfg.ID, &cfg.Key, &cfg.Value, &cfg.Description, &cfg.IsEncrypted, &cfg.Category, &cfg.UpdatedAt)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Config not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"config": cfg})
}

func handleUpdateConfig(c *gin.Context) {
	var req struct {
		Key         string `json:"key" binding:"required"`
		Value       string `json:"value" binding:"required"`
		Description string `json:"description"`
		IsEncrypted bool   `json:"is_encrypted"`
		Category    string `json:"category"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")

	_, err := dbPool.Exec(ctx, `
		INSERT INTO system_configs (key, value, description, is_encrypted, category, updated_at, updated_by)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6)
		ON CONFLICT (key) DO UPDATE SET 
			value = $2, 
			description = COALESCE(NULLIF($3, ''), system_configs.description),
			is_encrypted = $4,
			category = COALESCE(NULLIF($5, ''), system_configs.category),
			updated_at = NOW(),
			updated_by = $6
	`, req.Key, req.Value, req.Description, req.IsEncrypted, req.Category, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
		return
	}

	logAudit(userID, c.GetString("user_email"), "UPDATE_CONFIG", "config", req.Key, "Config updated", c)

	c.JSON(http.StatusOK, gin.H{"message": "Config updated successfully"})
}

func handleDeleteConfig(c *gin.Context) {
	key := c.Param("key")

	_, err := dbPool.Exec(ctx, "DELETE FROM system_configs WHERE key = $1", key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete config"})
		return
	}

	logAudit(c.GetString("user_id"), c.GetString("user_email"), "DELETE_CONFIG", "config", key, "Config deleted", c)

	c.JSON(http.StatusOK, gin.H{"message": "Config deleted successfully"})
}

// ============================ AUDIT HANDLERS ============================

func handleListAuditLogs(c *gin.Context) {
	page := getIntParam(c, "page", 1)
	limit := getIntParam(c, "limit", 50)
	offset := (page - 1) * limit

	rows, err := dbPool.Query(ctx, `
		SELECT id, user_id, user_email, action, resource, resource_id, details, ip_address, status, created_at 
		FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.UserEmail, &l.Action, &l.Resource, &l.ResourceID, &l.Details, &l.IPAddress, &l.Status, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs, "page": page, "limit": limit})
}

func handleExportAuditLogs(c *gin.Context) {
	rows, err := dbPool.Query(ctx, `
		SELECT id, user_id, user_email, action, resource, resource_id, details, ip_address, status, created_at 
		FROM audit_logs ORDER BY created_at DESC LIMIT 10000
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export audit logs"})
		return
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.UserEmail, &l.Action, &l.Resource, &l.ResourceID, &l.Details, &l.IPAddress, &l.Status, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs, "count": len(logs)})
}

// ============================ NOTIFICATION HANDLERS ============================

func handleListNotifications(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := dbPool.Query(ctx, `
		SELECT id, user_id, title, message, type, is_read, priority, action_url, created_at 
		FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.Type, &n.IsRead, &n.Priority, &n.ActionURL, &n.CreatedAt); err != nil {
			continue
		}
		notifications = append(notifications, n)
	}

	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

func handleMarkNotificationRead(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	_, err := dbPool.Exec(ctx, "UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

func handleMarkAllNotificationsRead(c *gin.Context) {
	userID := c.GetString("user_id")

	_, err := dbPool.Exec(ctx, "UPDATE notifications SET is_read = true WHERE user_id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notifications as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

func handleDeleteNotification(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	_, err := dbPool.Exec(ctx, "DELETE FROM notifications WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted"})
}

// ============================ FEATURE FLAG HANDLERS ============================

func handleListFeatureFlags(c *gin.Context) {
	rows, err := dbPool.Query(ctx, "SELECT id, name, description, is_enabled, rollout_percent, target_roles, created_at, updated_at FROM feature_flags")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feature flags"})
		return
	}
	defer rows.Close()

	var flags []FeatureFlag
	for rows.Next() {
		var f FeatureFlag
		if err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.IsEnabled, &f.RolloutPercent, &f.TargetRoles, &f.CreatedAt, &f.UpdatedAt); err != nil {
			continue
		}
		flags = append(flags, f)
	}

	c.JSON(http.StatusOK, gin.H{"features": flags})
}

func handleCreateFeatureFlag(c *gin.Context) {
	var req struct {
		Name           string   `json:"name" binding:"required"`
		Description    string   `json:"description"`
		IsEnabled      bool     `json:"is_enabled"`
		RolloutPercent int      `json:"rollout_percent"`
		TargetRoles    []string `json:"target_roles"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := generateID()
	rolesJSON, _ := json.Marshal(req.TargetRoles)

	_, err := dbPool.Exec(ctx, `
		INSERT INTO feature_flags (id, name, description, is_enabled, rollout_percent, target_roles)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, req.Name, req.Description, req.IsEnabled, req.RolloutPercent, string(rolesJSON))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create feature flag"})
		return
	}

	logAudit(c.GetString("user_id"), c.GetString("user_email"), "CREATE_FEATURE_FLAG", "feature_flag", id, "Feature flag created", c)

	c.JSON(http.StatusCreated, gin.H{"message": "Feature flag created"})
}

func handleUpdateFeatureFlag(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Description    string   `json:"description"`
		IsEnabled      bool     `json:"is_enabled"`
		RolloutPercent int      `json:"rollout_percent"`
		TargetRoles    []string `json:"target_roles"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rolesJSON, _ := json.Marshal(req.TargetRoles)

	_, err := dbPool.Exec(ctx, `
		UPDATE feature_flags SET 
			description = COALESCE(NULLIF($1, ''), description),
			is_enabled = $2,
			rollout_percent = $3,
			target_roles = $4,
			updated_at = NOW()
		WHERE id = $5
	`, req.Description, req.IsEnabled, req.RolloutPercent, string(rolesJSON), id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update feature flag"})
		return
	}

	logAudit(c.GetString("user_id"), c.GetString("user_email"), "UPDATE_FEATURE_FLAG", "feature_flag", id, "Feature flag updated", c)

	c.JSON(http.StatusOK, gin.H{"message": "Feature flag updated"})
}

func handleDeleteFeatureFlag(c *gin.Context) {
	id := c.Param("id")

	_, err := dbPool.Exec(ctx, "DELETE FROM feature_flags WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete feature flag"})
		return
	}

	logAudit(c.GetString("user_id"), c.GetString("user_email"), "DELETE_FEATURE_FLAG", "feature_flag", id, "Feature flag deleted", c)

	c.JSON(http.StatusOK, gin.H{"message": "Feature flag deleted"})
}

// ============================ API KEY HANDLERS ============================

func handleListAPIKeys(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := dbPool.Query(ctx, `
		SELECT id, name, key, permissions, is_active, expires_at, last_used, created_at
		FROM api_keys WHERE created_by = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch API keys"})
		return
	}
	defer rows.Close()

	var keys []gin.H
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.Key, &k.Permissions, &k.IsActive, &k.ExpiresAt, &k.LastUsed, &k.CreatedAt); err != nil {
			continue
		}
		keys = append(keys, gin.H{
			"id":          k.ID,
			"name":        k.Name,
			"key":         k.Key,
			"permissions": k.Permissions,
			"is_active":   k.IsActive,
			"expires_at":  k.ExpiresAt,
			"last_used":   k.LastUsed,
			"created_at":  k.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": keys})
}

func handleCreateAPIKey(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Permissions []string `json:"permissions"`
		ExpiresIn   int      `json:"expires_in"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")

	key := generateAPIKey()
	secret := generateAPISecret()
	secretHash, _ := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)

	expiresAt := time.Now().AddDate(0, 0, req.ExpiresIn)
	if req.ExpiresIn == 0 {
		expiresAt = time.Now().AddDate(1, 0, 0)
	}

	permissions := getPermissionsForRole("api_user")
	if len(req.Permissions) > 0 {
		permissions = req.Permissions
	}

	id := generateID()
	permJSON, _ := json.Marshal(permissions)

	_, err := dbPool.Exec(ctx, `
		INSERT INTO api_keys (id, name, key, secret_hash, permissions, is_active, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, req.Name, key, string(secretHash), string(permJSON), true, expiresAt, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	logAudit(userID, c.GetString("user_email"), "CREATE_API_KEY", "api_key", id, "API key created", c)

	c.JSON(http.StatusCreated, gin.H{
		"message": "API key created",
		"api_key": gin.H{
			"id":          id,
			"name":        req.Name,
			"key":         key,
			"secret":      secret,
			"permissions": permissions,
			"expires_at":  expiresAt,
		},
	})
}

func handleDeleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	_, err := dbPool.Exec(ctx, "DELETE FROM api_keys WHERE id = $1 AND created_by = $2", id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key"})
		return
	}

	logAudit(userID, c.GetString("user_email"), "DELETE_API_KEY", "api_key", id, "API key deleted", c)

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
}

func handleRotateAPIKey(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	key := generateAPIKey()
	secret := generateAPISecret()
	secretHash, _ := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)

	_, err := dbPool.Exec(ctx, `
		UPDATE api_keys SET key = $1, secret_hash = $2, updated_at = NOW()
		WHERE id = $3 AND created_by = $4
	`, key, string(secretHash), id, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rotate API key"})
		return
	}

	logAudit(userID, c.GetString("user_email"), "ROTATE_API_KEY", "api_key", id, "API key rotated", c)

	c.JSON(http.StatusOK, gin.H{
		"message": "API key rotated",
		"api_key": gin.H{
			"key":    key,
			"secret": secret,
		},
	})
}

// ============================ SYSTEM HANDLERS ============================

func handleDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"dashboard": gin.H{
			"total_users":       0,
			"active_admins":     0,
			"total_transactions": 0,
			"system_health":     "healthy",
		},
	})
}

func handleDashboardStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"stats": gin.H{
			"users_today":     0,
			"active_sessions": 0,
			"api_calls_today": 0,
			"error_rate":      0,
		},
	})
}

func handleSystemStatus(c *gin.Context) {
	var dbStatus string
	if err := dbPool.Ping(ctx); err != nil {
		dbStatus = "unhealthy"
	} else {
		dbStatus = "healthy"
	}

	var redisStatus string
	if err := redisClient.Ping(ctx).Err(); err != nil {
		redisStatus = "unhealthy"
	} else {
		redisStatus = "healthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status": gin.H{
			"database": dbStatus,
			"redis":    redisStatus,
			"version":  ServiceVersion,
			"timestamp": time.Now(),
		},
	})
}

func handleSystemMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"metrics": []interface{}{}})
}

func handleDetailedHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"healthy":  true,
		"service": ServiceName,
		"version": ServiceVersion,
	})
}

func handleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   ServiceName,
		"version":   ServiceVersion,
		"timestamp": time.Now(),
	})
}

func handleMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"requests_total":    0,
		"requests_failed":   0,
		"response_time_p50": 0,
	})
}

// ============================ UTILITY FUNCTIONS ============================

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateToken(userID, email, role string, permissions []string) (string, error) {
	data := fmt.Sprintf("%s:%s:%s:%d", userID, email, generateID(), time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	token := hex.EncodeToString(hash[:])

	claims := map[string]interface{}{
		"user_id":     userID,
		"email":       email,
		"role":        role,
		"permissions": permissions,
		"exp":         time.Now().Add(config.SessionDuration).Unix(),
	}

	claimsJSON, _ := json.Marshal(claims)
	redisClient.Set(ctx, "token_claims:"+token, claimsJSON, config.SessionDuration)

	return token, nil
}

func verifyToken(token string) (map[string]interface{}, error) {
	claimsJSON, err := redisClient.Get(ctx, "token_claims:"+token).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("invalid token")
	}
	if err != nil {
		return nil, err
	}

	var claims map[string]interface{}
	if err := json.Unmarshal([]byte(claimsJSON), &claims); err != nil {
		return nil, err
	}

	exp, ok := claims["exp"].(float64)
	if !ok || int64(exp) < time.Now().Unix() {
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}

func extractToken(authHeader string) string {
	if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
		return authHeader[7:]
	}
	return ""
}

func generateAPIKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "tw_" + hex.EncodeToString(b)
}

func generateAPISecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func getPermissionsForRole(role string) []string {
	switch role {
	case "super_admin":
		return []string{"*"}
	case "admin":
		return []string{"users:read", "users:write", "config:read", "config:write", "audit:read"}
	case "support":
		return []string{"users:read", "audit:read"}
	case "api_user":
		return []string{"api:read"}
	default:
		return []string{}
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getIntParam(c *gin.Context, param string, defaultValue int) int {
	value := c.Query(param)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func logAudit(userID, userEmail, action, resource, resourceID, details string, c *gin.Context) {
	dbPool.Exec(ctx, `
		INSERT INTO audit_logs (user_id, user_email, action, resource, resource_id, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, userID, userEmail, action, resource, resourceID, details, c.ClientIP(), c.Request.UserAgent())
}

func storeMetric(name string, value float64, unit string) {
	tagsJSON, _ := json.Marshal(map[string]string{})
	dbPool.Exec(ctx, `
		INSERT INTO system_metrics (metric_name, value, unit, tags, timestamp)
		VALUES ($1, $2, $3, $4, $5)
	`, name, value, unit, string(tagsJSON), time.Now())
}

func startMetricsCollection() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			storeMetric("system_cpu_usage", 0.0, "percent")
			storeMetric("system_memory_usage", 0.0, "percent")
			dbPool.Exec(ctx, "DELETE FROM system_metrics WHERE timestamp < NOW() - INTERVAL '7 days'")
		}
	}()
}
