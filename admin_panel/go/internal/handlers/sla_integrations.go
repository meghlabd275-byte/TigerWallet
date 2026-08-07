// SLA and Integration handlers
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/services"
)

// GetSLAPolicies returns all SLA policies
func (h *Handler) GetSLAPolicies(c *gin.Context) {
	policies, err := h.slaService.ListPolicies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

// CreateSLAPolicy creates a new SLA policy
func (h *Handler) CreateSLAPolicy(c *gin.Context) {
	var req struct {
		Name              string  `json:"name" binding:"required"`
		Description       string  `json:"description"`
		Priority          string  `json:"priority" binding:"required"`
		ResponseTimeSLA   int     `json:"response_time_sla" binding:"required"`
		ResolutionTimeSLA int     `json:"resolution_time_sla" binding:"required"`
		UptimeSLA         float64 `json:"uptime_sla"`
		IsActive          bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy := &services.SLAPolicy{
		Name:              req.Name,
		Description:       req.Description,
		Priority:          req.Priority,
		ResponseTimeSLA:   req.ResponseTimeSLA,
		ResolutionTimeSLA: req.ResolutionTimeSLA,
		UptimeSLA:         req.UptimeSLA,
		IsActive:          req.IsActive,
	}

	adminID := uuid.MustParse(c.GetString("admin_id"))
	created, err := h.slaService.CreatePolicy(c.Request.Context(), policy, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"policy": created})
}

// UpdateSLAPolicy updates an SLA policy
func (h *Handler) UpdateSLAPolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Name              string  `json:"name"`
		Description       string  `json:"description"`
		Priority          string  `json:"priority"`
		ResponseTimeSLA   int     `json:"response_time_sla"`
		ResolutionTimeSLA int     `json:"resolution_time_sla"`
		UptimeSLA         float64 `json:"uptime_sla"`
		IsActive          bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy := &services.SLAPolicy{
		Name:              req.Name,
		Description:       req.Description,
		Priority:          req.Priority,
		ResponseTimeSLA:   req.ResponseTimeSLA,
		ResolutionTimeSLA: req.ResolutionTimeSLA,
		UptimeSLA:         req.UptimeSLA,
		IsActive:          req.IsActive,
	}

	if err := h.slaService.UpdatePolicy(c.Request.Context(), id, policy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "SLA policy updated"})
}

// DeleteSLAPolicy deletes an SLA policy
func (h *Handler) DeleteSLAPolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.slaService.DeletePolicy(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "SLA policy deleted"})
}

// GetSLAReports returns all SLA reports
func (h *Handler) GetSLAReports(c *gin.Context) {
	policyIDStr := c.Query("policy_id")
	var policyID *uuid.UUID
	if policyIDStr != "" {
		id := uuid.MustParse(policyIDStr)
		policyID = &id
	}

	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		limit = 0
	}
	if o := c.Query("offset"); o != "" {
		offset = 0
	}

	reports, total, err := h.slaService.ListReports(c.Request.Context(), policyID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reports": reports, "total": total})
}

// GenerateSLAReport generates a new SLA report
func (h *Handler) GenerateSLAReport(c *gin.Context) {
	var req struct {
		PolicyID    string `json:"policy_id" binding:"required"`
		PeriodStart string `json:"period_start" binding:"required"`
		PeriodEnd   string `json:"period_end" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policyID := uuid.MustParse(req.PolicyID)
	// Parse dates and generate report
	report, err := h.slaService.GenerateReport(c.Request.Context(), policyID, time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"report": report})
}

// GetIntegrations returns all integrations
func (h *Handler) GetIntegrations(c *gin.Context) {
	integration := c.Query("type")
	configs, err := h.integrationService.GetIntegrationConfigs(c.Request.Context(), integration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"integrations": configs})
}

// CreateIntegration creates a new integration
func (h *Handler) CreateIntegration(c *gin.Context) {
	var req struct {
		Integration string                 `json:"integration" binding:"required"`
		Name        string                 `json:"name" binding:"required"`
		APIKey      string                 `json:"api_key"`
		APISecret   string                 `json:"api_secret"`
		WebhookURL  string                 `json:"webhook_url"`
		IsActive    bool                   `json:"is_active"`
		Settings    map[string]interface{} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := &services.IntegrationConfig{
		Integration: req.Integration,
		Name:        req.Name,
		APIKey:      req.APIKey,
		APISecret:   req.APISecret,
		WebhookURL:  req.WebhookURL,
		IsActive:    req.IsActive,
		Settings:    req.Settings,
	}

	adminID := uuid.MustParse(c.GetString("admin_id"))
	created, err := h.integrationService.SaveIntegrationConfig(c.Request.Context(), config, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"integration": created})
}

// UpdateIntegration updates an integration
func (h *Handler) UpdateIntegration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Name       string                 `json:"name"`
		APIKey     string                 `json:"api_key"`
		APISecret  string                 `json:"api_secret"`
		WebhookURL string                 `json:"webhook_url"`
		IsActive   bool                   `json:"is_active"`
		Settings   map[string]interface{} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := &services.IntegrationConfig{
		ID:         id,
		Name:       req.Name,
		APIKey:     req.APIKey,
		APISecret:  req.APISecret,
		WebhookURL: req.WebhookURL,
		IsActive:   req.IsActive,
		Settings:   req.Settings,
	}

	if _, err := h.integrationService.SaveIntegrationConfig(c.Request.Context(), config, uuid.Nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "integration updated"})
}

// DeleteIntegration deletes an integration
func (h *Handler) DeleteIntegration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.integrationService.DeleteIntegrationConfig(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "integration deleted"})
}

// TestIntegration tests an integration
func (h *Handler) TestIntegration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	success, message, err := h.integrationService.TestIntegration(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": success, "message": message})
}
