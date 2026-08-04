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
// TIGERWALLET MASTER ADMIN MANAGEMENT
// Create and manage Master Admins within White Labels
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

// Master Admin
type MasterAdmin struct {
	ID              string    `json:"id"`
	WhiteLabelID    string    `json:"whiteLabelId"`
	Email           string    `json:"email"`
	Username        string    `json:"username"`
	PasswordHash   string    `json:"-"`
	Role            string    `json:"role"` // master_admin, admin, manager, support
	Permissions     []string  `json:"permissions"`
	Status         string    `json:"status"` // active, suspended, pending
	TwoFactorEnabled bool    `json:"twoFactorEnabled"`
	TwoFactorSecret  string   `json:"-"`
	IPWhitelist     []string  `json:"ipWhitelist"`
	LastLogin       *time.Time `json:"lastLogin"`
	FailedAttempts  int       `json:"failedAttempts"`
	LockedUntil     *time.Time `json:"lockedUntil"`
	CreatedBy       string    `json:"createdBy"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Admin Permission
type AdminPermission struct {
	ID          string   `json:"id"`
	AdminID     string   `json:"adminId"`
	Resource    string   `json:"resource"`
	Actions     []string `json:"actions"` // read, write, delete
	CreatedAt   time.Time `json:"createdAt"`
}

// Login Request
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	WhiteLabelID string `json:"whiteLabelId" binding:"required"`
}

// Create Admin Request
type CreateAdminRequest struct {
	Email       string   `json:"email" binding:"required,email"`
	Username    string   `json:"username" binding:"required"`
	Password    string   `json:"password" binding:"required,min=8"`
	Role        string   `json:"role" binding:"required"`
	Permissions []string `json:"permissions"`
	WhiteLabelID string `json:"whiteLabelId" binding:"required"`
}

// Update Admin Request
type UpdateAdminRequest struct {
	Username    string   `json:"username"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
}

// Database Schema
func createMasterAdminSchema() error {
	schema := `
	-- Master Admins Table
	CREATE TABLE IF NOT EXISTS master_admins (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		white_label_id UUID NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		username VARCHAR(100) NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'admin',
		permissions JSONB DEFAULT '[]',
		status VARCHAR(20) DEFAULT 'active',
		two_factor_enabled BOOLEAN DEFAULT false,
		two_factor_secret VARCHAR(255),
		ip_whitelist JSONB DEFAULT '[]',
		last_login TIMESTAMP,
		failed_attempts INTEGER DEFAULT 0,
		locked_until TIMESTAMP,
		created_by UUID,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- Admin Permissions
	CREATE TABLE IF NOT EXISTS admin_permissions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		admin_id UUID NOT NULL REFERENCES master_admins(id) ON DELETE CASCADE,
		resource VARCHAR(100) NOT NULL,
		actions JSONB DEFAULT '[]',
		created_at TIMESTAMP DEFAULT NOW()
	);

	-- Admin Sessions
	CREATE TABLE IF NOT EXISTS admin_sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		admin_id UUID NOT NULL REFERENCES master_admins(id) ON DELETE CASCADE,
		token VARCHAR(500) NOT NULL,
		ip_address VARCHAR(50),
		user_agent VARCHAR(255),
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT NOW()
	);

	-- Admin Audit Log
	CREATE TABLE IF NOT EXISTS admin_audit_logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		admin_id UUID NOT NULL,
		action VARCHAR(100) NOT NULL,
		resource VARCHAR(100),
		details JSONB DEFAULT '{}',
		ip_address VARCHAR(50),
		created_at TIMESTAMP DEFAULT NOW()
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_master_admins_white_label ON master_admins(white_label_id);
	CREATE INDEX IF NOT EXISTS idx_master_admins_email ON master_admins(email);
	CREATE INDEX IF NOT EXISTS idx_admin_permissions_admin ON admin_permissions(admin_id);
	CREATE INDEX IF NOT EXISTS idx_admin_sessions_admin ON admin_sessions(admin_id);
	CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_admin ON admin_audit_logs(admin_id);
	`

	_, err := dbPool.Exec(context.Background(), schema)
	return err
}

// Handler Functions

// Login Master Admin
func LoginMasterAdmin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find admin
	var admin MasterAdmin
	var passwordHash string
	err := dbPool.QueryRow(context.Background(), `
		SELECT id, white_label_id, email, username, password_hash, role, permissions, status, 
		       two_factor_enabled, failed_attempts, locked_until
		FROM master_admins 
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

	// Check if locked
	if admin.LockedUntil != nil && admin.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusLocked, gin.H{"error": "Account is locked"})
		return
	}

	// Verify password
	if !checkPassword(req.Password, passwordHash) {
		// Increment failed attempts
		newAttempts := admin.FailedAttempts + 1
		var lockTime *time.Time
		if newAttempts >= 5 {
			t := time.Now().Add(15 * time.Minute)
			lockTime = &t
		}

		dbPool.Exec(context.Background(), `
			UPDATE master_admins SET failed_attempts = $1, locked_until = $2 WHERE id = $3
		`, newAttempts, lockTime, admin.ID)

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check 2FA if enabled
	if admin.TwoFactorEnabled {
		// Return 2FA required response
		c.JSON(http.StatusOK, gin.H{
			"require2FA":     true,
			"adminId":        admin.ID,
			"message":        "2FA required"
		})
		return
	}

	// Generate session token
	token := generateToken()
	expiresAt := time.Now().Add(24 * time.Hour)

	// Save session
	dbPool.Exec(context.Background(), `
		INSERT INTO admin_sessions (id, admin_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, generateUUID(), admin.ID, token, expiresAt)

	// Update login time and reset failed attempts
	dbPool.Exec(context.Background(), `
		UPDATE master_admins SET last_login = NOW(), failed_attempts = 0, locked_until = NULL WHERE id = $1
	`, admin.ID)

	// Log the login
	logAdminAction(admin.ID, "login", "session", map[string]interface{}{"ip": c.ClientIP()})

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"admin": gin.H{
			"id":           admin.ID,
			"email":        admin.Email,
			"username":     admin.Username,
			"role":         admin.Role,
			"permissions":  admin.Permissions,
			"whiteLabelId": admin.WhiteLabelID,
		}
	})
}

// Create Master Admin (only by Super Admin or existing Master Admin)
func CreateMasterAdmin(c *gin.Context) {
	var req CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get creator info from context (set by auth middleware)
	creatorID := c.GetString("adminID")
	creatorRole := c.GetString("adminRole")

	// Verify permissions
	if creatorRole != "super_admin" && creatorRole != "master_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	// If creating another master_admin, only super_admin can do it
	if req.Role == "master_admin" && creatorRole != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only Super Admin can create Master Admin"})
		return
	}

	// Check if email already exists
	var existingID string
	err := dbPool.QueryRow(context.Background(), 
		"SELECT id FROM master_admins WHERE email = $1", req.Email).Scan(&existingID)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	// Hash password
	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	admin := MasterAdmin{
		ID:           generateUUID(),
		WhiteLabelID: req.WhiteLabelID,
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: passwordHash,
		Role:         req.Role,
		Permissions:  req.Permissions,
		Status:       "active",
		CreatedBy:    creatorID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = dbPool.Exec(context.Background(), `
		INSERT INTO master_admins (id, white_label_id, email, username, password_hash, role, 
			permissions, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, admin.ID, admin.WhiteLabelID, admin.Email, admin.Username, admin.PasswordHash,
		admin.Role, admin.Permissions, admin.Status, admin.CreatedBy, admin.CreatedAt, admin.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create default permissions
	for _, resource := range []string{"users", "transactions", "kyc", "settings", "analytics"} {
		actions := []string{"read", "write"}
		if admin.Role == "master_admin" || admin.Role == "admin" {
			actions = []string{"read", "write", "delete"}
		}

		dbPool.Exec(context.Background(), `
			INSERT INTO admin_permissions (id, admin_id, resource, actions, created_at)
			VALUES ($1, $2, $3, $4, NOW())
		`, generateUUID(), admin.ID, resource, actions)
	}

	// Log action
	logAdminAction(creatorID, "create_admin", "master_admins", map[string]interface{}{
		"newAdminId": admin.ID,
		"role":       admin.Role,
	})

	c.JSON(http.StatusCreated, gin.H{
		"admin": gin.H{
			"id":           admin.ID,
			"email":        admin.Email,
			"username":     admin.Username,
			"role":         admin.Role,
			"whiteLabelId": admin.WhiteLabelID,
			"status":       admin.Status,
		}
	})
}

// Get Admins for White Label
func GetWhiteLabelAdmins(c *gin.Context) {
	whiteLabelID := c.Param("whiteLabelId")

	rows, err := dbPool.Query(context.Background(), `
		SELECT id, white_label_id, email, username, role, permissions, status, 
		       two_factor_enabled, last_login, failed_attempts, created_at
		FROM master_admins 
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
		var admin MasterAdmin
		rows.Scan(
			&admin.ID, &admin.WhiteLabelID, &admin.Email, &admin.Username,
			&admin.Role, &admin.Permissions, &admin.Status, &admin.TwoFactorEnabled,
			&admin.LastLogin, &admin.FailedAttempts, &admin.CreatedAt,
		)

		admins = append(admins, gin.H{
			"id":              admin.ID,
			"email":           admin.Email,
			"username":        admin.Username,
			"role":           admin.Role,
			"permissions":     admin.Permissions,
			"status":         admin.Status,
			"twoFactorEnabled": admin.TwoFactorEnabled,
			"lastLogin":      admin.LastLogin,
			"failedAttempts":  admin.FailedAttempts,
			"createdAt":      admin.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"admins": admins})
}

// Get Single Admin
func GetAdmin(c *gin.Context) {
	adminID := c.Param("id")

	var admin MasterAdmin
	err := dbPool.QueryRow(context.Background(), `
		SELECT id, white_label_id, email, username, role, permissions, status, 
		       two_factor_enabled, ip_whitelist, last_login, failed_attempts, created_at
		FROM master_admins WHERE id = $1
	`, adminID).Scan(
		&admin.ID, &admin.WhiteLabelID, &admin.Email, &admin.Username,
		&admin.Role, &admin.Permissions, &admin.Status, &admin.TwoFactorEnabled,
		&admin.IPWhitelist, &admin.LastLogin, &admin.FailedAttempts, &admin.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get permissions
	permRows, _ := dbPool.Query(context.Background(), `
		SELECT resource, actions FROM admin_permissions WHERE admin_id = $1
	`, adminID)

	var permissions []gin.H
	for permRows.Next() {
		var resource string
		var actions []string
		permRows.Scan(&resource, &actions)
		permissions = append(permissions, gin.H{
			"resource": resource,
			"actions":  actions,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"admin": gin.H{
			"id":               admin.ID,
			"whiteLabelId":     admin.WhiteLabelID,
			"email":            admin.Email,
			"username":         admin.Username,
			"role":            admin.Role,
			"permissions":     admin.Permissions,
			"detailedPermissions": permissions,
			"status":          admin.Status,
			"twoFactorEnabled": admin.TwoFactorEnabled,
			"ipWhitelist":     admin.IPWhitelist,
			"lastLogin":       admin.LastLogin,
			"failedAttempts":  admin.FailedAttempts,
			"createdAt":       admin.CreatedAt,
		}
	})
}

// Update Admin
func UpdateAdmin(c *gin.Context) {
	adminID := c.Param("id")

	var req UpdateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if admin exists
	var existing MasterAdmin
	err := dbPool.QueryRow(context.Background(), `
		SELECT id, white_label_id, role FROM master_admins WHERE id = $1
	`, adminID).Scan(&existing.ID, &existing.WhiteLabelID, &existing.Role)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	// Build update query
	query := "UPDATE master_admins SET updated_at = NOW()"
	args := []interface{}{}
	argCount := 0

	if req.Username != "" {
		argCount++
		query += fmt.Sprintf(", username = $%d", argCount)
		args = append(args, req.Username)
	}

	if req.Role != "" {
		argCount++
		query += fmt.Sprintf(", role = $%d", argCount)
		args = append(args, req.Role)
	}

	if req.Permissions != nil {
		argCount++
		permJSON, _ := json.Marshal(req.Permissions)
		query += fmt.Sprintf(", permissions = $%d", argCount)
		args = append(args, string(permJSON))
	}

	if req.Status != "" {
		argCount++
		query += fmt.Sprintf(", status = $%d", argCount)
		args = append(args, req.Status)
	}

	argCount++
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, adminID)

	_, err = dbPool.Exec(context.Background(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log action
	creatorID := c.GetString("adminID")
	logAdminAction(creatorID, "update_admin", "master_admins", map[string]interface{}{
		"updatedAdminId": adminID,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Admin updated successfully"})
}

// Delete Admin
func DeleteAdmin(c *gin.Context) {
	adminID := c.Param("id")

	// Get current admin
	currentAdminID := c.GetString("adminID")
	if currentAdminID == adminID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete yourself"})
		return
	}

	// Check role
	currentRole := c.GetString("adminRole")
	if currentRole != "super_admin" && currentRole != "master_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	// Check if trying to delete master_admin
	var targetRole string
	err := dbPool.QueryRow(context.Background(), 
		"SELECT role FROM master_admins WHERE id = $1", adminID).Scan(&targetRole)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	if targetRole == "master_admin" && currentRole != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only Super Admin can delete Master Admin"})
		return
	}

	// Delete
	_, err = dbPool.Exec(context.Background(), 
		"DELETE FROM master_admins WHERE id = $1", adminID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log action
	logAdminAction(currentAdminID, "delete_admin", "master_admins", map[string]interface{}{
		"deletedAdminId": adminID,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Admin deleted successfully"})
}

// Suspend/Activate Admin
func ToggleAdminStatus(c *gin.Context) {
	adminID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Status != "active" && req.Status != "suspended" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	_, err := dbPool.Exec(context.Background(), `
		UPDATE master_admins SET status = $1, updated_at = NOW() WHERE id = $2
	`, req.Status, adminID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log action
	creatorID := c.GetString("adminID")
	logAdminAction(creatorID, "toggle_status", "master_admins", map[string]interface{}{
		"adminId": adminID,
		"status":   req.Status,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Admin status updated"})
}

// Enable 2FA
func EnableTwoFactor(c *gin.Context) {
	adminID := c.Param("id")

	// In production, generate actual TOTP secret
	secret := generateRandomString(16)

	_, err := dbPool.Exec(context.Background(), `
		UPDATE master_admins SET two_factor_enabled = true, two_factor_secret = $1, updated_at = NOW() 
		WHERE id = $2
	`, secret, adminID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"secret": secret,
		"message": "2FA enabled. Use this secret to set up your authenticator app."
	})
}

// Get Admin Audit Logs
func GetAdminAuditLogs(c *gin.Context) {
	adminID := c.Param("id")

	rows, err := dbPool.Query(context.Background(), `
		SELECT id, action, resource, details, ip_address, created_at
		FROM admin_audit_logs 
		WHERE admin_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, adminID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var logs []gin.H
	for rows.Next() {
		var id, action, resource, ipAddress string
		var details json.RawMessage
		var createdAt time.Time
		rows.Scan(&id, &action, &resource, &details, &ipAddress, &createdAt)

		logs = append(logs, gin.H{
			"id":         id,
			"action":     action,
			"resource":   resource,
			"details":    details,
			"ipAddress":  ipAddress,
			"createdAt":  createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// Helper Functions
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)[:length]
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func logAdminAction(adminID, action, resource string, details map[string]interface{}) {
	detailsJSON, _ := json.Marshal(details)
	dbPool.Exec(context.Background(), `
		INSERT INTO admin_audit_logs (id, admin_id, action, resource, details, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, generateUUID(), adminID, action, resource, detailsJSON)
}

// Router Setup
func setupRouter() *gin.Engine {
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
	r.POST("/api/v1/master-admin/login", LoginMasterAdmin)

	// Protected
	admin := r.Group("/api/v1/master-admin")
	admin.Use(func(c *gin.Context) {
		// Simple auth check - in production use JWT
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}
		// Set mock admin info
		c.Set("adminID", "admin-123")
		c.Set("adminRole", "master_admin")
		c.Next()
	})
	{
		admin.POST("", CreateMasterAdmin)
		admin.GET("/white-label/:whiteLabelId", GetWhiteLabelAdmins)
		admin.GET("/:id", GetAdmin)
		admin.PUT("/:id", UpdateAdmin)
		admin.DELETE("/:id", DeleteAdmin)
		admin.POST("/:id/toggle-status", ToggleAdminStatus)
		admin.POST("/:id/enable-2fa", EnableTwoFactor)
		admin.GET("/:id/audit-logs", GetAdminAuditLogs)
	}

	return r
}

func main() {
	logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	logger.Info().Msg("Starting Master Admin Management Service")

	config := Config{
		Port:        getEnv("PORT", "8091"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://tigerwallet:tigerpass@localhost:5432/tigerwallet?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
	}

	var err error
	dbPool, err = pgxpool.Connect(context.Background(), config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	if err := createMasterAdminSchema(); err != nil {
		logger.Warn().Err(err).Msg("Schema creation warning")
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})
	defer redisClient.Close()

	router := setupRouter()

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

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
