package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
)

// WhiteLabelHandler handles white label-related requests
type WhiteLabelHandler struct {
	db *database.PostgresDB
}

// NewWhiteLabelHandler creates a new white label handler
func NewWhiteLabelHandler(db *database.PostgresDB) *WhiteLabelHandler {
	return &WhiteLabelHandler{db: db}
}

// ListWhiteLabels lists all white labels
func (h *WhiteLabelHandler) ListWhiteLabels(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	search := c.Query("search")
	status := c.Query("status")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var whiteLabels []models.WhiteLabel
	var total int64

	query := h.db.Model(&models.WhiteLabel{})

	if search != "" {
		query = query.Where("name ILIKE ? OR domain ILIKE ? OR slug ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&whiteLabels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch white labels"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        whiteLabels,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetWhiteLabel gets a white label by ID
func (h *WhiteLabelHandler) GetWhiteLabel(c *gin.Context) {
	whiteLabelID := c.Param("id")

	var whiteLabel models.WhiteLabel
	if err := h.db.First(&whiteLabel, whiteLabelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "White label not found"})
		return
	}

	c.JSON(http.StatusOK, whiteLabel)
}

// CreateWhiteLabel creates a new white label
func (h *WhiteLabelHandler) CreateWhiteLabel(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		Name           string `json:"name" binding:"required"`
		Slug           string `json:"slug" binding:"required"`
		Domain         string `json:"domain"`
		LogoURL        string `json:"logo_url"`
		FaviconURL     string `json:"favicon_url"`
		PrimaryColor   string `json:"primary_color"`
		SecondaryColor string `json:"secondary_color"`
		ContactEmail   string `json:"contact_email"`
		ContactPhone   string `json:"contact_phone"`
		Address        string `json:"address"`
		Description    string `json:"description"`
		CustomCSS      string `json:"custom_css"`
		CustomJS       string `json:"custom_js"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check if slug already exists
	var existingWL models.WhiteLabel
	if err := h.db.Where("slug = ?", req.Slug).First(&existingWL).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "White label with this slug already exists"})
		return
	}

	// Check if domain already exists
	if req.Domain != "" {
		if err := h.db.Where("domain = ?", req.Domain).First(&existingWL).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "White label with this domain already exists"})
			return
		}
	}

	whiteLabel := models.WhiteLabel{
		Name:           req.Name,
		Slug:           req.Slug,
		Domain:         req.Domain,
		LogoURL:        req.LogoURL,
		FaviconURL:     req.FaviconURL,
		PrimaryColor:   req.PrimaryColor,
		SecondaryColor: req.SecondaryColor,
		ContactEmail:   req.ContactEmail,
		ContactPhone:   req.ContactPhone,
		Address:        req.Address,
		Description:    req.Description,
		Status:         "pending",
	}

	if req.CustomCSS != "" {
		whiteLabel.CustomCSS.String = req.CustomCSS
		whiteLabel.CustomCSS.Valid = true
	}
	if req.CustomJS != "" {
		whiteLabel.CustomJS.String = req.CustomJS
		whiteLabel.CustomJS.Valid = true
	}

	if err := h.db.Create(&whiteLabel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create white label"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "create_whitelabel", "whitelabel", strconv.FormatUint(uint64(whiteLabel.ID), 10),
		"Created white label: "+whiteLabel.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, whiteLabel)
}

// UpdateWhiteLabel updates a white label
func (h *WhiteLabelHandler) UpdateWhiteLabel(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	whiteLabelID := c.Param("id")

	var req struct {
		Name           string `json:"name"`
		Domain         string `json:"domain"`
		LogoURL        string `json:"logo_url"`
		FaviconURL     string `json:"favicon_url"`
		PrimaryColor   string `json:"primary_color"`
		SecondaryColor string `json:"secondary_color"`
		ContactEmail   string `json:"contact_email"`
		ContactPhone   string `json:"contact_phone"`
		Address        string `json:"address"`
		Description    string `json:"description"`
		CustomCSS      string `json:"custom_css"`
		CustomJS       string `json:"custom_js"`
		Status         string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var whiteLabel models.WhiteLabel
	if err := h.db.First(&whiteLabel, whiteLabelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "White label not found"})
		return
	}

	updates := map[string]interface{}{}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Domain != "" {
		updates["domain"] = req.Domain
	}
	if req.LogoURL != "" {
		updates["logo_url"] = req.LogoURL
	}
	if req.FaviconURL != "" {
		updates["favicon_url"] = req.FaviconURL
	}
	if req.PrimaryColor != "" {
		updates["primary_color"] = req.PrimaryColor
	}
	if req.SecondaryColor != "" {
		updates["secondary_color"] = req.SecondaryColor
	}
	if req.ContactEmail != "" {
		updates["contact_email"] = req.ContactEmail
	}
	if req.ContactPhone != "" {
		updates["contact_phone"] = req.ContactPhone
	}
	if req.Address != "" {
		updates["address"] = req.Address
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.CustomCSS != "" {
		updates["custom_css"] = req.CustomCSS
	}
	if req.CustomJS != "" {
		updates["custom_js"] = req.CustomJS
	}
	if req.Status != "" {
		updates["status"] = req.Status

		// If approving, set approved_at
		if req.Status == "active" {
			now := time.Now()
			updates["approved_at"] = now
			updates["approved_by"] = adminID
		}
	}

	if err := h.db.Model(&whiteLabel).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update white label"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "update_whitelabel", "whitelabel", whiteLabelID,
		"Updated white label: "+whiteLabel.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, whiteLabel)
}

// DeleteWhiteLabel soft deletes a white label
func (h *WhiteLabelHandler) DeleteWhiteLabel(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	whiteLabelID := c.Param("id")

	var whiteLabel models.WhiteLabel
	if err := h.db.First(&whiteLabel, whiteLabelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "White label not found"})
		return
	}

	// Soft delete - set status to deleted
	if err := h.db.Model(&whiteLabel).Update("status", "deleted").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete white label"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "delete_whitelabel", "whitelabel", whiteLabelID,
		"Deleted white label: "+whiteLabel.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "White label deleted successfully"})
}

// ApproveWhiteLabel approves a white label
func (h *WhiteLabelHandler) ApproveWhiteLabel(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	whiteLabelID := c.Param("id")

	var whiteLabel models.WhiteLabel
	if err := h.db.First(&whiteLabel, whiteLabelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "White label not found"})
		return
	}

	if whiteLabel.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "White label is not pending"})
		return
	}

	now := time.Now()

	if err := h.db.Model(&whiteLabel).Updates(map[string]interface{}{
		"status":      "active",
		"approved_at": now,
		"approved_by": adminID,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve white label"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "approve_whitelabel", "whitelabel", whiteLabelID,
		"Approved white label: "+whiteLabel.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "White label approved successfully"})
}

// SuspendWhiteLabel suspends a white label
func (h *WhiteLabelHandler) SuspendWhiteLabel(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	whiteLabelID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Suspension reason is required"})
		return
	}

	var whiteLabel models.WhiteLabel
	if err := h.db.First(&whiteLabel, whiteLabelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "White label not found"})
		return
	}

	if err := h.db.Model(&whiteLabel).Update("status", "suspended").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to suspend white label"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "suspend_whitelabel", "whitelabel", whiteLabelID,
		"Suspended white label: "+whiteLabel.Name+" - Reason: "+req.Reason, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "White label suspended successfully"})
}

// GetWhiteLabelStats gets white label statistics
func (h *WhiteLabelHandler) GetWhiteLabelStats(c *gin.Context) {
	var stats struct {
		Total     int64 `json:"total"`
		Active    int64 `json:"active"`
		Pending   int64 `json:"pending"`
		Suspended int64 `json:"suspended"`
	}

	h.db.Model(&models.WhiteLabel{}).Count(&stats.Total)
	h.db.Model(&models.WhiteLabel{}).Where("status = ?", "active").Count(&stats.Active)
	h.db.Model(&models.WhiteLabel{}).Where("status = ?", "pending").Count(&stats.Pending)
	h.db.Model(&models.WhiteLabel{}).Where("status = ?", "suspended").Count(&stats.Suspended)

	c.JSON(http.StatusOK, stats)
}
