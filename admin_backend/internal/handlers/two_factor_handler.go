package handlers

import (
	"net/http"
	"strconv"

	"admin_backend/internal/models"
	"admin_backend/internal/services"
	"admin_backend/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// TwoFactorHandler handles 2FA related requests
type TwoFactorHandler struct {
	twoFactorService *services.TwoFactorService
}

// NewTwoFactorHandler creates a new 2FA handler
func NewTwoFactorHandler(db *database.PostgresDB, redis *redis.Client) *TwoFactorHandler {
	return &TwoFactorHandler{
		twoFactorService: services.NewTwoFactorService(db, redis),
	}
}

// Setup2FA initiates 2FA setup for a user
// POST /api/v1/2fa/setup
func (h *TwoFactorHandler) Setup2FA(c *gin.Context) {
	var req services.Setup2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	response, err := h.twoFactorService.Setup2FA(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Verify2FA verifies a 2FA code
// POST /api/v1/2fa/verify
func (h *TwoFactorHandler) Verify2FA(c *gin.Context) {
	var req services.Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.IPAddress = c.ClientIP()

	valid, message, err := h.twoFactorService.Verify2FA(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": message})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": "Verification successful",
	})
}

// Enable2FA enables 2FA for a user
// POST /api/v1/2fa/enable
func (h *TwoFactorHandler) Enable2FA(c *gin.Context) {
	var req services.Enable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := h.twoFactorService.Enable2FA(c, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "2FA enabled successfully",
	})
}

// Disable2FA disables 2FA for a user
// POST /api/v1/2fa/disable
func (h *TwoFactorHandler) Disable2FA(c *gin.Context) {
	var req services.Disable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := h.twoFactorService.Disable2FA(c, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "2FA disabled successfully",
	})
}

// Get2FAStatus gets 2FA status for a user
// GET /api/v1/2fa/status/:user_type/:user_id
func (h *TwoFactorHandler) Get2FAStatus(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userType := c.Param("user_type")

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	status, err := h.twoFactorService.Get2FAStatus(c, uint(userID), userType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// RegenerateBackupCodes generates new backup codes
// POST /api/v1/2fa/regenerate-backup-codes
func (h *TwoFactorHandler) RegenerateBackupCodes(c *gin.Context) {
	var req struct {
		UserID   uint   `json:"user_id" binding:"required"`
		UserType string `json:"user_type" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	codes, err := h.twoFactorService.RegenerateBackupCodes(c, req.UserID, req.UserType, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backup_codes": codes,
		"message":      "Backup codes regenerated successfully. Store these codes securely.",
	})
}

// ValidateRateLimit validates rate limiting for 2FA attempts
// GET /api/v1/2fa/rate-limit/:user_type/:user_id
func (h *TwoFactorHandler) ValidateRateLimit(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userType := c.Param("user_type")

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.twoFactorService.ValidateRateLimit(c, uint(userID), userType)
	if err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"allowed": true})
}

// GetCodeRemainingTime returns seconds until current TOTP code expires
// GET /api/v1/2fa/code-timer
func (h *TwoFactorHandler) GetCodeRemainingTime(c *gin.Context) {
	remaining := h.twoFactorService.CalculateCodeRemainingSeconds()

	c.JSON(http.StatusOK, gin.H{
		"remaining_seconds": remaining,
		"valid_until":     remaining,
	})
}

// List2FAUsers lists all users with 2FA enabled (admin only)
// GET /api/v1/admin/2fa/users
func (h *TwoFactorHandler) List2FAUsers(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	enabled := c.Query("enabled")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var users []models.TwoFactorAuth
	var total int64

	query := h.db.Model(&models.TwoFactorAuth{})

	if enabled == "true" {
		query = query.Where("enabled = ?", true)
	} else if enabled == "false" {
		query = query.Where("enabled = ?", false)
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

// Get2FAStats gets 2FA statistics (admin only)
// GET /api/v1/admin/2fa/stats
func (h *TwoFactorHandler) Get2FAStats(c *gin.Context) {
	var stats struct {
		TotalUsers     int64 `json:"total_users"`
		Enabled2FA     int64 `json:"enabled_2fa"`
		Disabled2FA    int64 `json:"disabled_2fa"`
		Admin2FA       int64 `json:"admin_2fa"`
		User2FA        int64 `json:"user_2fa"`
		SuperAdmin2FA  int64 `json:"super_admin_2fa"`
	}

	h.db.Model(&models.TwoFactorAuth{}).Count(&stats.TotalUsers)
	h.db.Model(&models.TwoFactorAuth{}).Where("enabled = ?", true).Count(&stats.Enabled2FA)
	h.db.Model(&models.TwoFactorAuth{}).Where("enabled = ?", false).Count(&stats.Disabled2FA)
	h.db.Model(&models.TwoFactorAuth{}).Where("user_type = ? AND enabled = ?", "admin", true).Count(&stats.Admin2FA)
	h.db.Model(&models.TwoFactorAuth{}).Where("user_type = ? AND enabled = ?", "user", true).Count(&stats.User2FA)
	h.db.Model(&models.TwoFactorAuth{}).Where("user_type = ? AND enabled = ?", "super_admin", true).Count(&stats.SuperAdmin2FA)

	c.JSON(http.StatusOK, stats)
}
