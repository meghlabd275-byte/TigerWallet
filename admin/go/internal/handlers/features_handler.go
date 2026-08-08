/**
 * TigerWallet Admin - Features Handler
 * Complete backend implementation for feature flag management
 */

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FeaturesHandler struct {
	db *gorm.DB
}

func NewFeaturesHandler(db *gorm.DB) *FeaturesHandler {
	return &FeaturesHandler{db: db}
}

// Feature model
type Feature struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"uniqueIndex;not null" json:"name"`
	Description    string    `json:"description"`
	Category       string    `json:"category"`
	Enabled        bool      `gorm:"default:false" json:"enabled"`
	RolloutPercent int       `gorm:"default:0" json:"rollout_percentage"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (Feature) TableName() string {
	return "features"
}

// GetAll handles GET /features
func (h *FeaturesHandler) GetAll(c *gin.Context) {
	var features []Feature
	if err := h.db.Order("category, name").Find(&features).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, features)
}

// GetByID handles GET /features/:id
func (h *FeaturesHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature id"})
		return
	}

	var feature Feature
	if err := h.db.First(&feature, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, feature)
}

// Create handles POST /features
func (h *FeaturesHandler) Create(c *gin.Context) {
	var feature Feature
	if err := c.ShouldBindJSON(&feature); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Create(&feature).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, feature)
}

// Update handles PUT /features/:id
func (h *FeaturesHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature id"})
		return
	}

	var feature Feature
	if err := h.db.First(&feature, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
		return
	}

	var input Feature
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update only provided fields
	updates := make(map[string]interface{})
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if input.Description != "" {
		updates["description"] = input.Description
	}
	if input.Category != "" {
		updates["category"] = input.Category
	}
	updates["rollout_percentage"] = input.RolloutPercent

	if err := h.db.Model(&feature).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, feature)
}

// Toggle handles POST /features/:id/toggle
func (h *FeaturesHandler) Toggle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature id"})
		return
	}

	result := h.db.Model(&Feature{}).Where("id = ?", id).Update("enabled", gorm.Expr("NOT enabled"))

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
		return
	}

	// Get updated feature
	var feature Feature
	h.db.First(&feature, id)

	c.JSON(http.StatusOK, gin.H{
		"message": "feature toggled successfully",
		"enabled": feature.Enabled,
	})
}

// SetRollout handles PUT /features/:id/rollout
func (h *FeaturesHandler) SetRollout(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature id"})
		return
	}

	var input struct {
		Percentage int `json:"percentage" binding:"required,min=0,max=100"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.db.Model(&Feature{}).Where("id = ?", id).Update("rollout_percentage", input.Percentage)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "rollout percentage updated",
		"rollout_percentage": input.Percentage,
	})
}

// Delete handles DELETE /features/:id
func (h *FeaturesHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature id"})
		return
	}

	result := h.db.Delete(&Feature{}, id)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feature deleted successfully"})
}

// CheckFeature handles GET /features/check/:name
func (h *FeaturesHandler) CheckFeature(c *gin.Context) {
	name := c.Param("name")

	var feature Feature
	if err := h.db.Where("name = ?", name).First(&feature).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
		return
	}

	// If rollout is 100%, feature is enabled for all
	if feature.RolloutPercent >= 100 {
		c.JSON(http.StatusOK, gin.H{"enabled": feature.Enabled})
		return
	}

	// If rollout is 0%, feature is disabled for all
	if feature.RolloutPercent <= 0 {
		c.JSON(http.StatusOK, gin.H{"enabled": false})
		return
	}

	// For partial rollout, check user ID
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusOK, gin.H{"enabled": false})
		return
	}

	userID, _ := strconv.ParseUint(userIDStr, 10, 32)
	// Simple hash-based rollout
	hash := (userID % 100)
	enabled := hash < uint64(feature.RolloutPercent) && feature.Enabled

	c.JSON(http.StatusOK, gin.H{
		"enabled":         enabled,
		"rollout_percent": feature.RolloutPercent,
	})
}
