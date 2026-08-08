package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
)

// APIKeyHandler handles API key-related requests - COMPLETE IMPLEMENTATION
type APIKeyHandler struct {
	db *database.PostgresDB
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(db *database.PostgresDB) *APIKeyHandler {
	return &APIKeyHandler{db: db}
}

// generateAPIKey generates a random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateSecret generates a random secret
func generateSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ListAPIKeys lists all API keys
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	userID := c.Query("user_id")
	status := c.Query("status")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var apiKeys []models.APIKey
	var total int64

	query := h.db.Model(&models.APIKey{})

	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&apiKeys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch API keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        apiKeys,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetAPIKey gets an API key by ID
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	keyID := c.Param("id")

	var apiKey models.APIKey
	if err := h.db.First(&apiKey, keyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

// CreateAPIKey creates a new API key
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		Name            string   `json:"name" binding:"required"`
		UserID          uint     `json:"user_id" binding:"required"`
		Permissions     []string `json:"permissions"`
		RateLimitMinute int      `json:"rate_limit_minute"`
		RateLimitDay    int      `json:"rate_limit_day"`
		ExpiresAt       *string  `json:"expires_at"`
		IPWhitelist     []string `json:"ip_whitelist"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}

	secret, err := generateSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate secret"})
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		parsed, err := time.Parse("2006-01-02T15:04:05Z", *req.ExpiresAt)
		if err == nil {
			expiresAt = &parsed
		}
	}

	permissionsJSON, _ := json.Marshal(req.Permissions)
	newAPIKey := models.APIKey{
		Name:        req.Name,
		AdminID:     req.UserID,
		Key:         apiKey,
		Permissions: permissionsJSON,
		RateLimit:   req.RateLimitMinute,
		ExpiresAt:   expiresAt,
		Status:      "active",
	}

	if err := h.db.Create(&newAPIKey).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	logAdminActivity(h.db, adminID, "create_api_key", "api_key", strconv.FormatUint(uint64(newAPIKey.ID), 10), "Created API key: "+newAPIKey.Name, c.ClientIP(), c.Request.UserAgent())

	// Return full key only on creation
	c.JSON(http.StatusCreated, gin.H{
		"id":          newAPIKey.ID,
		"name":        newAPIKey.Name,
		"api_key":     apiKey,
		"secret":      secret,
		"permissions": newAPIKey.Permissions,
		"created_at":  newAPIKey.CreatedAt,
		"expires_at":  newAPIKey.ExpiresAt,
	})
}

// UpdateAPIKey updates an API key
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	keyID := c.Param("id")

	var req struct {
		Name            string   `json:"name"`
		Permissions     []string `json:"permissions"`
		RateLimitMinute int      `json:"rate_limit_minute"`
		RateLimitDay    int      `json:"rate_limit_day"`
		ExpiresAt       *string  `json:"expires_at"`
		IPWhitelist     []string `json:"ip_whitelist"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var apiKey models.APIKey
	if err := h.db.First(&apiKey, keyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	updates := map[string]interface{}{}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Permissions != nil {
		updates["permissions"] = req.Permissions
	}
	if req.RateLimitMinute > 0 {
		updates["rate_limit_minute"] = req.RateLimitMinute
	}
	if req.RateLimitDay > 0 {
		updates["rate_limit_day"] = req.RateLimitDay
	}
	if req.ExpiresAt != nil {
		parsed, err := time.Parse("2006-01-02T15:04:05Z", *req.ExpiresAt)
		if err == nil {
			updates["expires_at"] = parsed
		}
	}
	if req.IPWhitelist != nil {
		updates["ip_whitelist"] = req.IPWhitelist
	}

	if err := h.db.Model(&apiKey).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update API key"})
		return
	}

	logAdminActivity(h.db, adminID, "update_api_key", "api_key", keyID, "Updated API key: "+apiKey.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, apiKey)
}

// DeleteAPIKey deletes an API key
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	keyID := c.Param("id")

	var apiKey models.APIKey
	if err := h.db.First(&apiKey, keyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	if err := h.db.Delete(&apiKey).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key"})
		return
	}

	logAdminActivity(h.db, adminID, "delete_api_key", "api_key", keyID, "Deleted API key: "+apiKey.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted successfully"})
}

// RevokeAPIKey revokes an API key
func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	keyID := c.Param("id")

	var apiKey models.APIKey
	if err := h.db.First(&apiKey, keyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	if err := h.db.Model(&apiKey).Update("status", "revoked").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API key"})
		return
	}

	logAdminActivity(h.db, adminID, "revoke_api_key", "api_key", keyID, "Revoked API key: "+apiKey.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked successfully"})
}

// ReactivateAPIKey reactivates an API key
func (h *APIKeyHandler) ReactivateAPIKey(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	keyID := c.Param("id")

	var apiKey models.APIKey
	if err := h.db.First(&apiKey, keyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	if err := h.db.Model(&apiKey).Update("status", "active").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reactivate API key"})
		return
	}

	logAdminActivity(h.db, adminID, "reactivate_api_key", "api_key", keyID, "Reactivated API key: "+apiKey.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "API key reactivated successfully"})
}

// RegenerateAPIKey regenerates an API key
func (h *APIKeyHandler) RegenerateAPIKey(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	keyID := c.Param("id")

	var apiKey models.APIKey
	if err := h.db.First(&apiKey, keyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	newAPIKey, err := generateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}

	newSecret, err := generateSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate secret"})
		return
	}

	updates := map[string]interface{}{
		"api_key":    newAPIKey,
		"secret":     newSecret,
		"updated_at": time.Now(),
	}

	if err := h.db.Model(&apiKey).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to regenerate API key"})
		return
	}

	logAdminActivity(h.db, adminID, "regenerate_api_key", "api_key", keyID, "Regenerated API key: "+apiKey.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"api_key": newAPIKey,
		"secret":  newSecret,
	})
}

// GetAPIKeyStats gets API key statistics
func (h *APIKeyHandler) GetAPIKeyStats(c *gin.Context) {
	var stats struct {
		TotalKeys   int64 `json:"total_keys"`
		ActiveKeys  int64 `json:"active_keys"`
		RevokedKeys int64 `json:"revoked_keys"`
		ExpiredKeys int64 `json:"expired_keys"`
	}

	h.db.Model(&models.APIKey{}).Count(&stats.TotalKeys)
	h.db.Model(&models.APIKey{}).Where("status = ?", "active").Count(&stats.ActiveKeys)
	h.db.Model(&models.APIKey{}).Where("status = ?", "revoked").Count(&stats.RevokedKeys)
	h.db.Model(&models.APIKey{}).Where("expires_at < ?", time.Now()).Count(&stats.ExpiredKeys)

	c.JSON(http.StatusOK, stats)
}

// ValidateAPIKey validates an API key
func (h *APIKeyHandler) ValidateAPIKey(c *gin.Context) {
	var req struct {
		APIKey string `json:"api_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var apiKey models.APIKey
	if err := h.db.Where("api_key = ?", req.APIKey).First(&apiKey).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		return
	}

	// Check if revoked
	if apiKey.Status == "revoked" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key has been revoked"})
		return
	}

	// Check if expired
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key has expired"})
		return
	}

	// Update last used
	h.db.Model(&apiKey).Update("last_used_at", time.Now())

	c.JSON(http.StatusOK, gin.H{
		"valid":       true,
		"user_id":     apiKey.AdminID,
		"permissions": apiKey.Permissions,
	})
}
