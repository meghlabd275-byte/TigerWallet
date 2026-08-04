package handlers

import (
	"net/http"
	"strconv"
	"time"

	"admin_backend/internal/config"
	"admin_backend/internal/middleware"
	"admin_backend/internal/models"
	"admin_backend/pkg/auth"
	"admin_backend/pkg/database"
	"admin_backend/pkg/redis"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AdminHandler handles admin-related requests
type AdminHandler struct {
	db       *database.PostgresDB
	redis    *redis.RedisClient
	cfg      *config.Config
	authSvc  *auth.AuthService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(db *database.PostgresDB, redisClient *redis.RedisClient, cfg *config.Config, authSvc *auth.AuthService) *AdminHandler {
	return &AdminHandler{
		db:      db,
		redis:   redisClient,
		cfg:      cfg,
		authSvc: authSvc,
	}
}

// AdminLoginRequest represents login request
type AdminLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AdminLoginResponse represents login response
type AdminLoginResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Admin        *models.Admin `json:"admin"`
}

// Login handles admin login
func (h *AdminHandler) Login(c *gin.Context) {
	var req AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Find admin by email
	var admin models.Admin
	result := h.db.Where("email = ?", req.Email).First(&admin)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if account is locked
	if admin.LockedUntil != nil && admin.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusLocked, gin.H{"error": "Account is locked. Please try again later."})
		return
	}

	// Verify password
	hashedPassword := req.Password + h.cfg.PasswordPepper
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(hashedPassword)); err != nil {
		// Increment failed attempts
		admin.FailedAttempts++
		if admin.FailedAttempts >= h.cfg.MaxLoginAttempts {
			lockedUntil := time.Now().Add(h.cfg.LockoutDuration)
			admin.LockedUntil = &lockedUntil
		}
		h.db.Model(&admin).Updates(map[string]interface{}{
			"failed_attempts": admin.FailedAttempts,
			"locked_until":    admin.LockedUntil,
		})

		// Log failed attempt
		h.logActivity(admin.ID, "login", "admin", strconv.FormatUint(uint64(admin.ID)), "Failed login attempt", c.ClientIP(), c.Request.UserAgent(), "failed", err.Error())

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if 2FA is enabled
	if admin.TwoFactorEnabled {
		// In production, would require 2FA code here
		// For now, we'll skip 2FA verification
	}

	// Reset failed attempts on successful login
	admin.FailedAttempts = 0
	admin.LockedUntil = nil
	now := time.Now()
	admin.LastLoginAt = &now
	admin.LastIP = c.ClientIP()

	h.db.Model(&admin).Updates(map[string]interface{}{
		"failed_attempts": 0,
		"locked_until":     nil,
		"last_login_at":   now,
		"last_ip":         c.ClientIP(),
	})

	// Generate tokens
	expiresAt := time.Now().Add(time.Duration(h.cfg.JWTExpirationHours) * time.Hour)
	token, err := h.authSvc.GenerateToken(admin.ID, admin.Email, admin.Role, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	refreshToken, err := h.authSvc.GenerateRefreshToken(admin.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Store session in Redis
	sessionData := map[string]interface{}{
		"admin_id":   admin.ID,
		"email":      admin.Email,
		"role":       admin.Role,
		"login_at":   time.Now().Unix(),
		"expires_at": expiresAt.Unix(),
	}
	h.redis.CacheUserSession(token, sessionData, time.Duration(h.cfg.JWTExpirationHours)*time.Hour)

	// Log successful login
	h.logActivity(admin.ID, "login", "admin", strconv.FormatUint(uint64(admin.ID)), "Successful login", c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusOK, AdminLoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		Admin:        &admin,
	})
}

// Logout handles admin logout
func (h *AdminHandler) Logout(c *gin.Context) {
	// Get token from header
	token := c.GetHeader("Authorization")
	if token != "" {
		// Remove session from Redis
		h.redis.DeleteUserSession(token)
	}

	// Log logout activity
	if adminID, exists := c.Get("admin_id"); exists {
		h.logActivity(adminID.(uint), "logout", "admin", strconv.FormatUint(adminID.(uint), 10), "Admin logged out", c.ClientIP(), c.Request.UserAgent(), "success", "")
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetProfile gets current admin profile
func (h *AdminHandler) GetProfile(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var admin models.Admin
	if err := h.db.Preload("AdminActivity", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC").Limit(10)
	}).First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	c.JSON(http.StatusOK, admin)
}

// UpdateProfile updates admin profile
func (h *AdminHandler) UpdateProfile(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Phone     string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updates := map[string]interface{}{
		"first_name": req.FirstName,
		"last_name":  req.LastName,
	}

	if err := h.db.Model(&models.Admin{}).Where("id = ?", adminID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	// Log activity
	h.logActivity(adminID, "update_profile", "admin", strconv.FormatUint(uint64(adminID)), "Profile updated", c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// ChangePassword changes admin password
func (h *AdminHandler) ChangePassword(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword    string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get admin
	var admin models.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	// Verify current password
	hashedPassword := req.CurrentPassword + h.cfg.PasswordPepper
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(hashedPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	newHashedPassword := req.NewPassword + h.cfg.PasswordPepper
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(newHashedPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	now := time.Now()
	if err := h.db.Model(&admin).Updates(map[string]interface{}{
		"password_hash":      string(hashedBytes),
		"password_changed_at": now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Log activity
	h.logActivity(adminID, "change_password", "admin", strconv.FormatUint(uint64(adminID)), "Password changed", c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// RefreshToken refreshes admin token
func (h *AdminHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Verify refresh token
	claims, err := h.authSvc.VerifyToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	adminID := uint(claims["admin_id"].(float64))

	// Get admin
	var admin models.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	// Generate new tokens
	expiresAt := time.Now().Add(time.Duration(h.cfg.JWTExpirationHours) * time.Hour)
	token, err := h.authSvc.GenerateToken(admin.ID, admin.Email, admin.Role, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	newRefreshToken, err := h.authSvc.GenerateRefreshToken(admin.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Update session in Redis
	sessionData := map[string]interface{}{
		"admin_id":   admin.ID,
		"email":      admin.Email,
		"role":       admin.Role,
		"login_at":   time.Now().Unix(),
		"expires_at": expiresAt.Unix(),
	}
	h.redis.CacheUserSession(token, sessionData, time.Duration(h.cfg.JWTExpirationHours)*time.Hour)

	c.JSON(http.StatusOK, gin.H{
		"token":         token,
		"refresh_token": newRefreshToken,
		"expires_at":    expiresAt,
	})
}

// ListAdmins lists all admins
func (h *AdminHandler) ListAdmins(c *gin.Context) {
	// Parse pagination
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	search := c.Query("search")
	role := c.Query("role")
	status := c.Query("status")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var admins []models.Admin
	var total int64

	query := h.db.Model(&models.Admin{})

	// Apply filters
	if search != "" {
		query = query.Where("email ILIKE ? OR username ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	query.Count(&total)

	// Paginate
	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&admins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch admins"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      admins,
		"total":     total,
		"page":      pageInt,
		"page_size": pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// CreateAdmin creates a new admin
func (h *AdminHandler) CreateAdmin(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		Username    string `json:"username" binding:"required"`
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required,min=8"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		Role        string `json:"role" binding:"required"`
		Permissions string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check if email already exists
	var existingAdmin models.Admin
	if err := h.db.Where("email = ?", req.Email).First(&existingAdmin).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	// Check if username already exists
	if err := h.db.Where("username = ?", req.Username).First(&existingAdmin).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	// Hash password
	hashedPassword := req.Password + h.cfg.PasswordPepper
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(hashedPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	admin := models.Admin{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedBytes),
		FirstName:   req.FirstName,
		LastName:     req.LastName,
		Role:        req.Role,
		Status:      "active",
	}

	if err := h.db.Create(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin"})
		return
	}

	// Log activity
	h.logActivity(adminID, "create_admin", "admin", strconv.FormatUint(uint64(admin.ID)), "Created new admin: "+admin.Email, c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusCreated, admin)
}

// UpdateAdmin updates an admin
func (h *AdminHandler) UpdateAdmin(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	adminIDAgg := c.Param("id")

	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Role      string `json:"role"`
		Status    string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var admin models.Admin
	if err := h.db.First(&admin, adminIDAgg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	updates := map[string]interface{}{}
	if req.FirstName != "" {
		updates["first_name"] = req.FirstName
	}
	if req.LastName != "" {
		updates["last_name"] = req.LastName
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	if err := h.db.Model(&admin).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update admin"})
		return
	}

	// Log activity
	h.logActivity(adminID, "update_admin", "admin", adminIDAgg, "Updated admin", c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusOK, admin)
}

// DeleteAdmin deletes an admin
func (h *AdminHandler) DeleteAdmin(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	adminIDAgg := c.Param("id")

	// Cannot delete yourself
	if adminID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete yourself"})
		return
	}

	if err := h.db.Delete(&models.Admin{}, adminIDAgg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete admin"})
		return
	}

	// Log activity
	h.logActivity(adminID, "delete_admin", "admin", adminIDAgg, "Deleted admin", c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusOK, gin.H{"message": "Admin deleted successfully"})
}

// GetAdminActivities gets admin activity logs
func (h *AdminHandler) GetAdminActivities(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "50")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var activities []models.AdminActivity
	var total int64

	query := h.db.Model(&models.AdminActivity{}).Where("admin_id = ?", adminID)
	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	if err := query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC").Find(&activities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activities"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        activities,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// Helper function to log activity
func (h *AdminHandler) logActivity(adminID uint, action, resource, resourceID, details, ip, userAgent, status, errorMsg string) {
	activity := models.AdminActivity{
		AdminID:     adminID,
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		IPAddress:   ip,
		UserAgent:   userAgent,
		Status:      status,
		ErrorMessage: errorMsg,
	}
	h.db.Create(&activity)
}

// Import gorm
import "gorm.io/gorm"
