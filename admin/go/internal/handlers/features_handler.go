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
	"github.com/tigerwallet/admin/pkg/redis"
	"gorm.io/gorm"
)

type FeaturesHandler struct {
	db    *gorm.DB
	redis *redis.RedisClient
}

func NewFeaturesHandler(db *gorm.DB, redisClient *redis.RedisClient) *FeaturesHandler {
	return &FeaturesHandler{db: db, redis: redisClient}
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

	h.publishFeatureState(feature.Name, redis.FeatureStateFromBool(feature.Enabled))

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

	// Reload to reflect the persisted state, then publish to Redis so
	// downstream services see the latest name/enabled mapping.
	h.db.First(&feature, id)
	h.publishFeatureState(feature.Name, redis.FeatureStateFromBool(feature.Enabled))

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

	h.publishFeatureState(feature.Name, redis.FeatureStateFromBool(feature.Enabled))

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

	// Load name before deletion so we can clear the live Redis state.
	var feature Feature
	h.db.First(&feature, id)

	result := h.db.Delete(&Feature{}, id)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
		return
	}

	h.deleteFeatureState(feature.Name)

	c.JSON(http.StatusOK, gin.H{"message": "feature deleted successfully"})
}

// CheckFeature handles GET /features/:id/check
//
// Reads the LIVE feature state from Redis (the same store downstream services
// consult) and augments it with the DB rollout-percentage logic. The Redis
// value is authoritative for behavioral enforcement: a "paused" or "disabled"
// state short-circuits to disabled regardless of the DB row.
func (h *FeaturesHandler) CheckFeature(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		name = c.Param("id")
	}

	var feature Feature
	if err := h.db.Where("name = ?", name).First(&feature).Error; err != nil {
		// Fall back to looking up by id when the route param is an id.
		if idErr := h.db.First(&feature, name).Error; idErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
			return
		}
	}

	// Live state from Redis (authoritative for enforcement).
	liveState, _ := h.getFeatureState(feature.Name)

	switch liveState {
	case redis.FeatureStatePaused:
		c.JSON(http.StatusOK, gin.H{
			"enabled":         false,
			"state":           liveState,
			"rollout_percent": feature.RolloutPercent,
			"message":         "feature is paused",
		})
		return
	case redis.FeatureStateDisabled:
		c.JSON(http.StatusOK, gin.H{
			"enabled":         false,
			"state":           liveState,
			"rollout_percent": feature.RolloutPercent,
		})
		return
	}

	// liveState == enabled (or unknown -> fail-closed disabled).
	if liveState != redis.FeatureStateEnabled {
		c.JSON(http.StatusOK, gin.H{
			"enabled":         false,
			"state":           liveState,
			"rollout_percent": feature.RolloutPercent,
		})
		return
	}

	// Enabled in Redis; apply DB rollout-percentage gating.
	if feature.RolloutPercent >= 100 {
		c.JSON(http.StatusOK, gin.H{"enabled": true, "state": liveState})
		return
	}
	if feature.RolloutPercent <= 0 {
		c.JSON(http.StatusOK, gin.H{"enabled": false, "state": liveState, "rollout_percent": feature.RolloutPercent})
		return
	}

	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusOK, gin.H{"enabled": false, "state": liveState, "rollout_percent": feature.RolloutPercent})
		return
	}

	userID, _ := strconv.ParseUint(userIDStr, 10, 32)
	hash := (userID % 100)
	enabled := hash < uint64(feature.RolloutPercent)

	c.JSON(http.StatusOK, gin.H{
		"enabled":         enabled,
		"state":           liveState,
		"rollout_percent": feature.RolloutPercent,
	})
}

// publishFeatureState writes the feature's live state to Redis. Failures are
// non-fatal: the DB write already succeeded and the next toggle re-publishes.
func (h *FeaturesHandler) publishFeatureState(name, state string) {
	if h.redis == nil || name == "" {
		return
	}
	_ = h.redis.PublishFeatureState(name, state)
}

// getFeatureState reads the feature's live state from Redis. Returns "" when
// the key is missing or Redis is unavailable (fail-closed: unknown = disabled).
func (h *FeaturesHandler) getFeatureState(name string) (string, error) {
	if h.redis == nil || name == "" {
		return "", nil
	}
	return h.redis.GetFeatureState(name)
}

// deleteFeatureState removes the feature's live state from Redis.
func (h *FeaturesHandler) deleteFeatureState(name string) {
	if h.redis == nil || name == "" {
		return
	}
	_ = h.redis.DeleteFeatureState(name)
}

// SetStatus handles PATCH/PUT /features/:id/status
//
// Realizes SuperAdmin halt/pause/start/resume behaviorally: it writes the
// requested state ("enabled" | "disabled" | "paused") to Redis, the store
// downstream services consult, and syncs the DB is_enabled flag so the record
// stays consistent (enabled -> true; disabled -> false; paused -> true so a
// resume returns to enabled).
func (h *FeaturesHandler) SetStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature id"})
		return
	}

	var input struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !redis.ValidFeatureState(input.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be one of: enabled, disabled, paused"})
		return
	}

	var feature Feature
	if err := h.db.First(&feature, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
		return
	}

	dbEnabled := true
	switch input.Status {
	case redis.FeatureStateDisabled:
		dbEnabled = false
	case redis.FeatureStateEnabled, redis.FeatureStatePaused:
		dbEnabled = true
	}
	if err := h.db.Model(&feature).Update("enabled", dbEnabled).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.redis.PublishFeatureState(feature.Name, input.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish feature state: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "feature state updated",
		"name":    feature.Name,
		"state":   input.Status,
		"enabled": dbEnabled,
	})
}

