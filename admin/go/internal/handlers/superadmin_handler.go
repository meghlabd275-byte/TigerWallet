package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tigerwallet/admin/internal/config"
	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/auth"
	"github.com/tigerwallet/admin/pkg/database"
	"github.com/tigerwallet/admin/pkg/redis"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SuperAdminHandler handles super admin specific operations
type SuperAdminHandler struct {
	db                 *database.PostgresDB
	redis              *redis.RedisClient
	cfg                *config.Config
	authSvc            *auth.AuthService
	masterWalletClient *http.Client
}

// NewSuperAdminHandler creates a new super admin handler
func NewSuperAdminHandler(db *database.PostgresDB, redisClient *redis.RedisClient, cfg *config.Config, authSvc *auth.AuthService) *SuperAdminHandler {
	return &SuperAdminHandler{
		db:                 db,
		redis:              redisClient,
		cfg:                cfg,
		authSvc:            authSvc,
		masterWalletClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// ==================== ADMIN MANAGEMENT ====================

// CreateAdmin creates a new admin user
// POST /api/v1/superadmin/admins
func (h *SuperAdminHandler) CreateAdmin(c *gin.Context) {
	var req struct {
		Email       string   `json:"email" binding:"required,email"`
		Password    string   `json:"password" binding:"required,min=8"`
		Username    string   `json:"username" binding:"required"`
		Role        string   `json:"role" binding:"required"`
		FirstName   string   `json:"first_name"`
		LastName    string   `json:"last_name"`
		Permissions []string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate role
	validRoles := []string{"superadmin", "admin", "support", "analyst", "viewer"}
	valid := false
	for _, r := range validRoles {
		if req.Role == r {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	// Check if email already exists
	var existingAdmin models.Admin
	if err := h.db.Where("email = ?", req.Email).First(&existingAdmin).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Admin with this email already exists"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password+h.cfg.PasswordPepper), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create admin
	permJSON, _ := json.Marshal(req.Permissions)
	admin := models.Admin{
		Email:          req.Email,
		Username:       req.Username,
		PasswordHash:   string(hashedPassword),
		Role:           req.Role,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Status:         "active",
		FailedAttempts: 0,
		Permissions:    json.RawMessage(permJSON),
	}

	if err := h.db.Create(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin"})
		return
	}

	// Log activity
	h.logActivity(c.GetUint("admin_id"), "create_admin", "admin", strconv.FormatUint(uint64(admin.ID), 10), "Created new admin: "+admin.Email, c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusCreated, gin.H{
		"message": "Admin created successfully",
		"admin":   admin,
	})
}

// ListAdmins lists all admins
// GET /api/v1/superadmin/admins
func (h *SuperAdminHandler) ListAdmins(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	role := c.Query("role")
	search := c.Query("search")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var admins []models.Admin
	var total int64

	query := h.db.Model(&models.Admin{})

	if role != "" {
		query = query.Where("role = ?", role)
	}

	if search != "" {
		query = query.Where("email ILIKE ? OR username ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	if err := query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC").Find(&admins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch admins"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      admins,
		"total":     total,
		"page":      pageInt,
		"page_size": pageSizeInt,
	})
}

// GetAdmin gets admin details
// GET /api/v1/superadmin/admins/:id
func (h *SuperAdminHandler) GetAdmin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid admin ID"})
		return
	}

	var admin models.Admin
	if err := h.db.First(&admin, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	c.JSON(http.StatusOK, admin)
}

// UpdateAdmin updates admin details
// PUT /api/v1/superadmin/admins/:id
func (h *SuperAdminHandler) UpdateAdmin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid admin ID"})
		return
	}

	var req struct {
		Email       string   `json:"email"`
		Username    string   `json:"username"`
		Role        string   `json:"role"`
		FirstName   string   `json:"first_name"`
		LastName    string   `json:"last_name"`
		Permissions []string `json:"permissions"`
		IsActive    *bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var admin models.Admin
	if err := h.db.First(&admin, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	updates := make(map[string]interface{})

	if req.Email != "" && req.Email != admin.Email {
		// Check if email is taken
		var existingAdmin models.Admin
		if err := h.db.Where("email = ? AND id != ?", req.Email, id).First(&existingAdmin).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already in use"})
			return
		}
		updates["email"] = req.Email
	}

	if req.Username != "" {
		updates["username"] = req.Username
	}

	if req.Role != "" {
		updates["role"] = req.Role
	}

	if req.FirstName != "" {
		updates["first_name"] = req.FirstName
	}

	if req.LastName != "" {
		updates["last_name"] = req.LastName
	}

	if req.Permissions != nil {
		updates["permissions"] = req.Permissions
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) > 0 {
		if err := h.db.Model(&admin).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update admin"})
			return
		}
	}

	h.logActivity(c.GetUint("admin_id"), "update_admin", "admin", strconv.FormatUint(id, 10), "Updated admin", c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusOK, gin.H{
		"message": "Admin updated successfully",
		"admin":   admin,
	})
}

// DeleteAdmin deletes an admin
// DELETE /api/v1/superadmin/admins/:id
func (h *SuperAdminHandler) DeleteAdmin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid admin ID"})
		return
	}

	// Prevent self-deletion
	if uint(id) == c.GetUint("admin_id") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
		return
	}

	var admin models.Admin
	if err := h.db.First(&admin, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	if err := h.db.Delete(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete admin"})
		return
	}

	h.logActivity(c.GetUint("admin_id"), "delete_admin", "admin", strconv.FormatUint(id, 10), "Deleted admin: "+admin.Email, c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusOK, gin.H{"message": "Admin deleted successfully"})
}

// SuspendAdmin suspends an admin
// PUT /api/v1/superadmin/admins/:id/suspend
func (h *SuperAdminHandler) SuspendAdmin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid admin ID"})
		return
	}

	// Prevent self-suspension
	if uint(id) == c.GetUint("admin_id") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot suspend your own account"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var admin models.Admin
	if err := h.db.First(&admin, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	lockedUntil := time.Now().Add(24 * time.Hour) // 24 hour suspension

	if err := h.db.Model(&admin).Updates(map[string]interface{}{
		"is_active":    false,
		"locked_until": lockedUntil,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to suspend admin"})
		return
	}

	// Revoke all sessions
	h.redis.DeleteUserSession("admin:" + strconv.FormatUint(id, 10))

	h.logActivity(c.GetUint("admin_id"), "suspend_admin", "admin", strconv.FormatUint(id, 10), "Suspended admin: "+req.Reason, c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusOK, gin.H{"message": "Admin suspended successfully"})
}

// ActivateAdmin activates a suspended admin
// PUT /api/v1/superadmin/admins/:id/activate
func (h *SuperAdminHandler) ActivateAdmin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid admin ID"})
		return
	}

	var admin models.Admin
	if err := h.db.First(&admin, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	if err := h.db.Model(&admin).Updates(map[string]interface{}{
		"is_active":    true,
		"locked_until": nil,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate admin"})
		return
	}

	h.logActivity(c.GetUint("admin_id"), "activate_admin", "admin", strconv.FormatUint(id, 10), "Activated admin", c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusOK, gin.H{"message": "Admin activated successfully"})
}

// ==================== SYSTEM CONFIGURATION ====================

// GetSystemConfig gets system configuration
// GET /api/v1/superadmin/config
func (h *SuperAdminHandler) GetSystemConfig(c *gin.Context) {
	config := map[string]interface{}{
		"site_name":                h.cfg.SiteName,
		"maintenance_mode":         h.cfg.MaintenanceMode,
		"registration_enabled":     h.cfg.RegistrationEnabled,
		"max_login_attempts":       h.cfg.MaxLoginAttempts,
		"lockout_duration":         h.cfg.LockoutDuration.String(),
		"jwt_expiration_hours":     h.cfg.JWTExpirationHours,
		"password_min_length":      h.cfg.PasswordMinLength,
		"password_require_number":  h.cfg.PasswordRequireNumber,
		"password_require_special": h.cfg.PasswordRequireSpecial,
	}

	c.JSON(http.StatusOK, config)
}

// UpdateSystemConfig updates system configuration
// PUT /api/v1/superadmin/config
func (h *SuperAdminHandler) UpdateSystemConfig(c *gin.Context) {
	var req struct {
		MaintenanceMode        *bool `json:"maintenance_mode"`
		RegistrationEnabled    *bool `json:"registration_enabled"`
		MaxLoginAttempts       *int  `json:"max_login_attempts"`
		JWTExpirationHours     *int  `json:"jwt_expiration_hours"`
		PasswordMinLength      *int  `json:"password_min_length"`
		PasswordRequireNumber  *bool `json:"password_require_number"`
		PasswordRequireSpecial *bool `json:"password_require_special"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Persist each provided setting as a SystemConfig row so the value survives
	// restarts. Keys are fixed identifiers, so there is no injection surface.
	type setting struct {
		key   string
		value interface{}
		kind  string
	}
	var settings []setting
	if req.MaintenanceMode != nil {
		settings = append(settings, setting{"maintenance_mode", *req.MaintenanceMode, "bool"})
	}
	if req.RegistrationEnabled != nil {
		settings = append(settings, setting{"registration_enabled", *req.RegistrationEnabled, "bool"})
	}
	if req.MaxLoginAttempts != nil {
		settings = append(settings, setting{"max_login_attempts", *req.MaxLoginAttempts, "int"})
	}
	if req.JWTExpirationHours != nil {
		settings = append(settings, setting{"jwt_expiration_hours", *req.JWTExpirationHours, "int"})
	}
	if req.PasswordMinLength != nil {
		settings = append(settings, setting{"password_min_length", *req.PasswordMinLength, "int"})
	}
	if req.PasswordRequireNumber != nil {
		settings = append(settings, setting{"password_require_number", *req.PasswordRequireNumber, "bool"})
	}
	if req.PasswordRequireSpecial != nil {
		settings = append(settings, setting{"password_require_special", *req.PasswordRequireSpecial, "bool"})
	}

	adminID := c.GetUint("admin_id")
	updates := make(map[string]interface{}, len(settings))

	for _, s := range settings {
		encoded := fmt.Sprintf("%v", s.value)
		var existing models.SystemConfig
		err := h.db.Where("key = ?", s.key).First(&existing).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			row := models.SystemConfig{
				Key:       s.key,
				Value:     encoded,
				Type:      s.kind,
				Category:  "security",
				UpdatedBy: &adminID,
			}
			if err := h.db.Create(&row).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist configuration"})
				return
			}
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load configuration"})
			return
		default:
			if err := h.db.Model(&models.SystemConfig{}).Where("key = ?", s.key).
				Updates(map[string]interface{}{"value": encoded, "type": s.kind, "updated_by": adminID}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist configuration"})
				return
			}
		}
		updates[s.key] = s.value
	}

	h.logActivity(adminID, "update_config", "config", "", "Updated system configuration", c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration updated successfully",
		"config":  updates,
	})
}

// ==================== FEATURE FLAGS ====================

// ListFeatureFlags lists all feature flags
// GET /api/v1/superadmin/features
func (h *SuperAdminHandler) ListFeatureFlags(c *gin.Context) {
	var flags []models.FeatureFlag
	if err := h.db.Find(&flags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feature flags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": flags})
}

// CreateFeatureFlag creates a new feature flag
// POST /api/v1/superadmin/features
func (h *SuperAdminHandler) CreateFeatureFlag(c *gin.Context) {
	var req struct {
		Name              string `json:"name" binding:"required"`
		Description       string `json:"description"`
		IsEnabled         bool   `json:"is_enabled"`
		RolloutPercentage int    `json:"rollout_percentage"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flag := models.FeatureFlag{
		Name:              req.Name,
		Description:       req.Description,
		IsEnabled:         req.IsEnabled,
		RolloutPercentage: req.RolloutPercentage,
		CreatedBy:         c.GetUint("admin_id"),
	}

	if err := h.db.Create(&flag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create feature flag"})
		return
	}

	h.publishFeatureState(flag.Name, redis.FeatureStateFromBool(flag.IsEnabled))

	c.JSON(http.StatusCreated, gin.H{"data": flag})
}

// UpdateFeatureFlag updates a feature flag
// PUT /api/v1/superadmin/features/:name
func (h *SuperAdminHandler) UpdateFeatureFlag(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		IsEnabled         *bool  `json:"is_enabled"`
		RolloutPercentage *int   `json:"rollout_percentage"`
		Description       string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var flag models.FeatureFlag
	if err := h.db.Where("name = ?", name).First(&flag).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feature flag not found"})
		return
	}

	updates := make(map[string]interface{})

	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.RolloutPercentage != nil {
		updates["rollout_percentage"] = *req.RolloutPercentage
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if err := h.db.Model(&flag).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update feature flag"})
		return
	}

	// Reload to capture final state and publish to Redis.
	h.db.Where("name = ?", name).First(&flag)
	h.publishFeatureState(flag.Name, redis.FeatureStateFromBool(flag.IsEnabled))

	h.logActivity(c.GetUint("admin_id"), "update_feature", "feature_flag", name, "Updated feature flag", c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusOK, gin.H{"data": flag})
}

// DeleteFeatureFlag deletes a feature flag
// DELETE /api/v1/superadmin/features/:name
func (h *SuperAdminHandler) DeleteFeatureFlag(c *gin.Context) {
	name := c.Param("name")

	if err := h.db.Where("name = ?", name).Delete(&models.FeatureFlag{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete feature flag"})
		return
	}

	h.deleteFeatureState(name)

	c.JSON(http.StatusOK, gin.H{"message": "Feature flag deleted successfully"})
}

// ==================== RATE LIMITS ====================

// publishFeatureState writes the feature flag's live state to Redis (shared
// store downstream services consult). Non-fatal on failure.
func (h *SuperAdminHandler) publishFeatureState(name, state string) {
	if h.redis == nil || name == "" {
		return
	}
	_ = h.redis.PublishFeatureState(name, state)
}

// deleteFeatureState removes the feature flag's live state from Redis.
func (h *SuperAdminHandler) deleteFeatureState(name string) {
	if h.redis == nil || name == "" {
		return
	}
	_ = h.redis.DeleteFeatureState(name)
}

// GetRateLimits gets current rate limits
// GET /api/v1/superadmin/rate-limits
func (h *SuperAdminHandler) GetRateLimits(c *gin.Context) {
	limits := map[string]interface{}{
		"api_requests_per_minute":         60,
		"withdrawal_requests_per_day":     10,
		"transaction_requests_per_minute": 30,
		"login_attempts_per_hour":         5,
	}

	c.JSON(http.StatusOK, limits)
}

// UpdateRateLimits updates rate limits
// PUT /api/v1/superadmin/rate-limits
func (h *SuperAdminHandler) UpdateRateLimits(c *gin.Context) {
	var req struct {
		APILimit          *int `json:"api_requests_per_minute"`
		WithdrawalLimit   *int `json:"withdrawal_requests_per_day"`
		TransactionLimit  *int `json:"transaction_requests_per_minute"`
		LoginAttemptLimit *int `json:"login_attempts_per_hour"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logActivity(c.GetUint("admin_id"), "update_rate_limits", "config", "", "Updated rate limits", c.ClientIP(), c.Request.UserAgent(), "success", "")

	c.JSON(http.StatusOK, gin.H{"message": "Rate limits updated successfully"})
}

// ==================== MASTER WALLET ACCESS ====================

// ListMasterWallets lists all master wallets.
// GET /api/v1/superadmin/system/master-wallets
//
// The MasterWallet backend (:8450) is the only authority on wallet state, so
// this proxies the request rather than synthesising an inventory. A dead or
// unconfigured upstream returns 502/503 — never an empty success.
func (h *SuperAdminHandler) ListMasterWallets(c *gin.Context) {
	h.proxyMasterWallet(c, http.MethodGet, "/api/v1/master-wallet", nil)
}

// GetMasterWallet gets master wallet details
// GET /api/v1/superadmin/system/master-wallets/:id
func (h *SuperAdminHandler) GetMasterWallet(c *gin.Context) {
	id, ok := masterWalletPathID(c)
	if !ok {
		return
	}
	h.proxyMasterWallet(c, http.MethodGet, "/api/v1/master-wallet/"+id, nil)
}

// GetMasterWalletBalance gets master wallet balance
// GET /api/v1/superadmin/system/master-wallets/:id/balance
func (h *SuperAdminHandler) GetMasterWalletBalance(c *gin.Context) {
	id, ok := masterWalletPathID(c)
	if !ok {
		return
	}
	h.proxyMasterWallet(c, http.MethodGet, "/api/v1/master-wallet/"+id+"/balance", nil)
}

// masterWalletPathID extracts and validates the wallet id path parameter.
func masterWalletPathID(c *gin.Context) (string, bool) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "master wallet id is required"})
		return "", false
	}
	return url.PathEscape(id), true
}

// proxyMasterWallet forwards a request to the canonical MasterWallet backend,
// attaching the service token and passing the upstream status and body through
// unchanged.
func (h *SuperAdminHandler) proxyMasterWallet(c *gin.Context, method, path string, body []byte) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), method, masterWalletBaseURL()+path, reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build upstream request"})
		return
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token := strings.TrimSpace(os.Getenv("SERVICE_AUTH_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := h.masterWalletClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":  "master wallet backend unavailable",
			"detail": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read master wallet response"})
		return
	}

	c.Data(resp.StatusCode, "application/json", respBody)
}

// masterWalletBaseURL resolves the MasterWallet backend base URL.
func masterWalletBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("MASTER_WALLET_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:8450"
}

// ==================== AUDIT LOGS ====================

// GetAuditLogs gets audit logs
// GET /api/v1/superadmin/audit-logs
func (h *SuperAdminHandler) GetAuditLogs(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "50")
	adminID := c.Query("admin_id")
	action := c.Query("action")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var logs []models.AuditLog
	var total int64

	query := h.db.Model(&models.AuditLog{})

	if adminID != "" {
		query = query.Where("admin_id = ?", adminID)
	}

	if action != "" {
		query = query.Where("action = ?", action)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	if err := query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      pageInt,
		"page_size": pageSizeInt,
	})
}

// ==================== HELPER FUNCTIONS ====================

func (h *SuperAdminHandler) logActivity(adminID uint, action, resourceType, resourceID, details, ip, userAgent, status, errorMsg string) {
	activity := models.AdminActivity{
		AdminID:      adminID,
		Action:       action,
		Resource:     resourceType,
		ResourceID:   resourceID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       status,
		ErrorMessage: errorMsg,
	}

	h.db.Create(&activity)
}
