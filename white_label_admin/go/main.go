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
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// WHITE LABEL ADMIN MANAGEMENT
// Create and manage admins within a White Label
// ============================================================================

var (
	logger     zerolog.Logger
	redisClient *redis.Client
	dbPool     *pgxpool.Pool
)

// Configuration
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret  string
}

// White Label Admin
type WhiteLabelAdmin struct {
	ID              string    `json:"id"`
	WhiteLabelID    string    `json:"whiteLabelId"`
	Email           string    `json:"email"`
	Username        string    `json:"username"`
	PasswordHash   string    `json:"-"`
	Role            string    `json:"role"` // admin, manager, support, analyst
	Department      string    `json:"department"`
	Permissions     []string  `json:"permissions"`
	Status         string    `json:"status"` // active, suspended, pending
	TwoFactorEnabled bool    `json:"twoFactorEnabled"`
	LastLogin       *time.Time `json:"lastLogin"`
	FailedAttempts  int       `json:"failedAttempts"`
	LockedUntil     *time.Time `json:"lockedUntil"`
	CreatedBy       string    `json:"createdBy"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Admin Role Definition
type RoleDefinition struct {
	Role          string   `json:"role"`
	Department    string   `json:"department"`
	Permissions   []string `json:"permissions"`
	Description   string   `json:"description"`
}

// Create Admin Request
type CreateAdminRequest struct {
	Email       string   `json:"email" binding:"required,email"`
	Username    string   `json:"username" binding:"required"`
	Password    string   `json:"password" binding:"required,min=8"`
	Role        string   `json:"role" binding:"required"`
	Department  string   `json:"department"`
	Permissions []string `json:"permissions"`
}

// Update Admin Request
type UpdateAdminRequest struct {
	Username    string   `json:"username"`
	Role        string   `json:"role"`
	Department  string   `json:"department"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
}

// Database Schema
func createWhiteLabelAdminSchema() error {
	schema := `
	-- White Label Admins Table
	CREATE TABLE IF NOT EXISTS white_label_admins (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		white_label_id UUID NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		username VARCHAR(100) NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'admin',
		department VARCHAR(100),
		permissions JSONB DEFAULT '[]',
		status VARCHAR(20) DEFAULT 'active',
		two_factor_enabled BOOLEAN DEFAULT false,
		last_login TIMESTAMP,
		failed_attempts INTEGER DEFAULT 0,
		locked_until TIMESTAMP,
		created_by UUID,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- Admin Sessions
	CREATE TABLE IF NOT EXISTS wl_admin_sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		admin_id UUID NOT NULL REFERENCES white_label_admins(id) ON DELETE CASCADE,
		token VARCHAR(500) NOT NULL,
		ip_address VARCHAR(50),
		user_agent VARCHAR(255),
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT NOW()
	);

	-- Admin Audit Log
	CREATE TABLE IF NOT EXISTS wl_admin_audit_logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		admin_id UUID NOT NULL,
		white_label_id UUID NOT NULL,
		action VARCHAR(100) NOT NULL,
		resource VARCHAR(100),
		details JSONB DEFAULT '{}',
		ip_address VARCHAR(50),
		created_at TIMESTAMP DEFAULT NOW()
	);

	-- Admin Permissions Matrix
	CREATE TABLE IF NOT EXISTS admin_role_definitions (
		role VARCHAR(50) PRIMARY KEY,
		department VARCHAR(100),
		permissions JSONB NOT NULL,
		description TEXT
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_wl_admins_white_label ON white_label_admins(white_label_id);
	CREATE INDEX IF NOT EXISTS idx_wl_admins_email ON white_label_admins(email);
	CREATE INDEX IF NOT EXISTS idx_wl_admin_audit ON wl_admin_audit_logs(admin_id, white_label_id);
	`

	_, err := dbPool.Exec(context.Background(), schema)
	return err
}

// Role Definitions
var roleDefinitions = []RoleDefinition{
	{
		Role:        "admin",
		Department:  "Administration",
		Permissions: []string{"users:*", "transactions:*", "kyc:*", "settings:*", "analytics:*", "support:*", "reports:*"},
		Description: "Full access to all features",
	},
	{
		Role:        "manager",
		Department:  "Operations",
		Permissions: []string{"users:read", "users:write", "transactions:*", "kyc:*", "analytics:*", "reports:read"},
		Description: "Manage users and transactions",
	},
	{
		Role:        "support",
		Department:  "Customer Support",
		Permissions: []string{"users:read", "transactions:read", "kyc:read", "support:*"},
		Description: "Customer support access",
	},
	{
		Role:        "analyst",
		Department:  "Analytics",
		Permissions: []string{"analytics:*", "reports:*", "transactions:read"},
		Description: "Analytics and reporting access",
	},
	{
		Role:        "finance",
		Department:  "Finance",
		Permissions: []string{"transactions:*", "reports:*", "analytics:read", "withdrawals:*"},
		Description: "Finance team access",
	},
}

// Handler Functions

// Login White Label Admin
func LoginWhiteLabelAdmin(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required"`
		Password    string `json:"password" binding:"required"`
		WhiteLabelID string `json:"whiteLabelId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find admin
	var admin WhiteLabelAdmin
	var passwordHash string
	err := dbPool.QueryRow(context.Background(), `
		SELECT id, white_label_id, email, username, password_hash, role, permissions, status, 
		       two_factor_enabled, failed_attempts, locked_until
		FROM white_label_admins 
		WHERE email = $1 AND white_label_id = $2
	`, req.Email, req.WhiteLabelID).Scan(
		&admin.ID, &admin.WhiteLabelID, &admin.Email, &admin.Username, &passwordHash,
		&admin.Role, &admin.Permissions, &admin.Status, &admin.TwoFactorEnabled,
		&admin.FailedAttempts, &admin.LockedUntil,
	)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check status
	if admin.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active"})
		return
	}

	// Check if locked
	if admin.LockedUntil != nil && admin.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusLocked, gin.H{"error": "Account is locked"})
		return
	}

	// Verify password
	if !checkPasswordWL(req.Password, passwordHash) {
		newAttempts := admin.FailedAttempts + 1
		var lockTime *time.Time
		if newAttempts >= 5 {
			t := time.Now().Add(15 * time.Minute)
			lockTime = &t
		}

		dbPool.Exec(context.Background(), `
			UPDATE white_label_admins SET failed_attempts = $1, locked_until = $2 WHERE id = $3
		`, newAttempts, lockTime, admin.ID)

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate session token
	token := generateTokenWL()
	expiresAt := time.Now().Add(24 * time.Hour)

	// Save session
	dbPool.Exec(context.Background(), `
		INSERT INTO wl_admin_sessions (id, admin_id, token, ip_address, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, generateUUIDWL(), admin.ID, token, c.ClientIP(), expiresAt)

	// Update login time
	dbPool.Exec(context.Background(), `
		UPDATE white_label_admins SET last_login = NOW(), failed_attempts = 0, locked_until = NULL WHERE id = $1
	`, admin.ID)

	// Log
	logWLAdminAction(admin.ID, admin.WhiteLabelID, "login", "session", map[string]interface{}{"ip": c.ClientIP()})

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"admin": gin.H{
			"id":           admin.ID,
			"email":        admin.Email,
			"username":     admin.Username,
			"role":         admin.Role,
			"department":   admin.Department,
			"permissions":  admin.Permissions,
			"whiteLabelId": admin.WhiteLabelID,
		}
	})
}

// Create White Label Admin
func CreateWhiteLabelAdmin(c *gin.Context) {
	var req CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get creator info
	creatorID := c.GetString("adminID")
	whiteLabelID := c.Param("whiteLabelId")

	// Validate role
	validRole := false
	var defaultPermissions []string
	for _, rd := range roleDefinitions {
		if rd.Role == req.Role {
			validRole = true
			defaultPermissions = rd.Permissions
			break
		}
	}

	if !validRole {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	// Check if email exists
	var existingID string
	err := dbPool.QueryRow(context.Background(), 
		"SELECT id FROM white_label_admins WHERE email = $1", req.Email).Scan(&existingID)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	// Hash password
	passwordHash, err := hashPasswordWL(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	permissions := req.Permissions
	if len(permissions) == 0 {
		permissions = defaultPermissions
	}

	admin := WhiteLabelAdmin{
		ID:           generateUUIDWL(),
		WhiteLabelID: whiteLabelID,
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: passwordHash,
		Role:         req.Role,
		Department:   req.Department,
		Permissions:  permissions,
		Status:       "active",
		CreatedBy:    creatorID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = dbPool.Exec(context.Background(), `
		INSERT INTO white_label_admins (id, white_label_id, email, username, password_hash, role, 
			department, permissions, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, admin.ID, admin.WhiteLabelID, admin.Email, admin.Username, admin.PasswordHash,
		admin.Role, admin.Department, admin.Permissions, admin.Status, admin.CreatedBy, admin.CreatedAt, admin.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log action
	logWLAdminAction(creatorID, whiteLabelID, "create_admin", "white_label_admins", map[string]interface{}{
		"newAdminId": admin.ID,
		"role":       admin.Role,
	})

	c.JSON(http.StatusCreated, gin.H{
		"admin": gin.H{
			"id":           admin.ID,
			"email":        admin.Email,
			"username":     admin.Username,
			"role":         admin.Role,
			"department":   admin.Department,
			"whiteLabelId": admin.WhiteLabelID,
			"status":       admin.Status,
		}
	})
}

// Get Admins for White Label
func GetWhiteLabelAdmins(c *gin.Context) {
	whiteLabelID := c.Param("whiteLabelId")

	rows, err := dbPool.Query(context.Background(), `
		SELECT id, white_label_id, email, username, role, department, permissions, status, 
		       two_factor_enabled, last_login, failed_attempts, created_at
		FROM white_label_admins 
		WHERE white_label_id = $1
		ORDER BY created_at DESC
	`, whiteLabelID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var admins []gin.H
	for rows.Next() {
		var admin WhiteLabelAdmin
		rows.Scan(
			&admin.ID, &admin.WhiteLabelID, &admin.Email, &admin.Username,
			&admin.Role, &admin.Department, &admin.Permissions, &admin.Status,
			&admin.TwoFactorEnabled, &admin.LastLogin, &admin.FailedAttempts, &admin.CreatedAt,
		)

		admins = append(admins, gin.H{
			"id":               admin.ID,
			"email":            admin.Email,
			"username":         admin.Username,
			"role":            admin.Role,
			"department":      admin.Department,
			"permissions":     admin.Permissions,
			"status":          admin.Status,
			"twoFactorEnabled": admin.TwoFactorEnabled,
			"lastLogin":       admin.LastLogin,
			"failedAttempts":  admin.FailedAttempts,
			"createdAt":       admin.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"admins": admins})
}

// Get Roles
func GetRoles(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"roles": roleDefinitions})
}

// Helper Functions
func generateUUIDWL() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func generateTokenWL() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashPasswordWL(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func checkPasswordWL(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func logWLAdminAction(adminID, whiteLabelID, action, resource string, details map[string]interface{}) {
	detailsJSON, _ := json.Marshal(details)
	dbPool.Exec(context.Background(), `
		INSERT INTO wl_admin_audit_logs (id, admin_id, white_label_id, action, resource, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, generateUUIDWL(), adminID, whiteLabelID, action, resource, detailsJSON)
}

// Router Setup
func setupRouterWL() *gin.Engine {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Public
	r.POST("/api/v1/wl-admin/login", LoginWhiteLabelAdmin)
	r.GET("/api/v1/wl-admin/roles", GetRoles)

	// Protected
	admin := r.Group("/api/v1/wl-admin")
	admin.Use(func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}
		c.Set("adminID", "admin-123")
		c.Next()
	})
	{
		admin.POST("/white-label/:whiteLabelId", CreateWhiteLabelAdmin)
		admin.GET("/white-label/:whiteLabelId", GetWhiteLabelAdmins)
	}

	return r
}

func main() {
	logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	logger.Info().Msg("Starting White Label Admin Management Service")

	config := Config{
		Port:        getEnvWL("PORT", "8093"),
		DatabaseURL: getEnvWL("DATABASE_URL", "postgres://tigerwallet:tigerpass@localhost:5432/tigerwallet?sslmode=disable"),
		RedisURL:    getEnvWL("REDIS_URL", "localhost:6379"),
	}

	var err error
	dbPool, err = pgxpool.Connect(context.Background(), config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	if err := createWhiteLabelAdminSchema(); err != nil {
		logger.Warn().Err(err).Msg("Schema creation warning")
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})
	defer redisClient.Close()

	router := setupRouterWL()

	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout:  15 * time.Second,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")
}

func getEnvWL(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
