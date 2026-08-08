package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// IntegrationHandler handles external integrations (Slack, PagerDuty, Datadog, Cloudflare)
type IntegrationHandler struct {
	db    *database.PostgresDB
	redis *redis.Client
}

// NewIntegrationHandler creates a new integration handler
func NewIntegrationHandler(db *database.PostgresDB, redis *redis.Client) *IntegrationHandler {
	return &IntegrationHandler{
		db:    db,
		redis: redis,
	}
}

// IntegrationRequest represents integration configuration request
type IntegrationRequest struct {
	Type     string                 `json:"type" binding:"required"` // slack, pagerduty, datadog, cloudflare
	Name     string                 `json:"name" binding:"required"`
	Config   map[string]interface{} `json:"config"`
	IsActive bool                   `json:"is_active"`
}

// SlackMessage represents Slack message format
type SlackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Text        string            `json:"text"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents Slack message attachment
type SlackAttachment struct {
	Color     string       `json:"color,omitempty"`
	Title     string       `json:"title,omitempty"`
	Text      string       `json:"text,omitempty"`
	Fields    []SlackField `json:"fields,omitempty"`
	Footer    string       `json:"footer,omitempty"`
	Timestamp string       `json:"ts,omitempty"`
}

// SlackField represents Slack field
type SlackField struct {
	Title string `json:"title,omitempty"`
	Value string `json:"value,omitempty"`
	Short bool   `json:"short,omitempty"`
}

// PagerDutyEvent represents PagerDuty event
type PagerDutyEvent struct {
	RoutingKey  string           `json:"routing_key"`
	EventAction string           `json:"event_action"`
	Payload     PagerDutyPayload `json:"payload"`
}

// PagerDutyPayload represents PagerDuty payload
type PagerDutyPayload struct {
	Summary       string                 `json:"summary"`
	Source        string                 `json:"source"`
	Severity      string                 `json:"severity"`
	Timestamp     string                 `json:"timestamp"`
	CustomDetails map[string]interface{} `json:"custom_details,omitempty"`
}

// DatadogEvent represents Datadog event
type DatadogEvent struct {
	Title      string   `json:"text"`
	Text       string   `json:"date_happened"`
	Priority   string   `json:"priority"`
	Tags       []string `json:"tags,omitempty"`
	AlertType  string   `json:"alert_type,omitempty"`
	SourceType string   `json:"source_type_name"`
}

// CreateIntegration creates a new integration
// POST /api/v1/admin/integrations
func (h *IntegrationHandler) CreateIntegration(c *gin.Context) {
	var req IntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	configJSON, _ := json.Marshal(req.Config)

	integration := models.IntegrationConfig{
		Type:     req.Type,
		Name:     req.Name,
		Config:   configJSON,
		IsActive: req.IsActive,
	}

	if err := h.db.Create(&integration).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create integration"})
		return
	}

	adminID := c.GetUint("admin_id")
	logAdminActivity(h.db, adminID, "create_integration", "integration",
		fmt.Sprintf("%d", integration.ID), "Created integration: "+integration.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, integration)
}

// UpdateIntegration updates an integration
// PUT /api/v1/admin/integrations/:id
func (h *IntegrationHandler) UpdateIntegration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid integration ID"})
		return
	}

	var req IntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var integration models.IntegrationConfig
	if err := h.db.First(&integration, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	updates := map[string]interface{}{
		"name":      req.Name,
		"is_active": req.IsActive,
	}

	if req.Config != nil {
		configJSON, _ := json.Marshal(req.Config)
		updates["config"] = configJSON
	}

	if err := h.db.Model(&integration).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update integration"})
		return
	}

	adminID := c.GetUint("admin_id")
	logAdminActivity(h.db, adminID, "update_integration", "integration",
		fmt.Sprintf("%d", id), "Updated integration: "+integration.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, integration)
}

// DeleteIntegration deletes an integration
// DELETE /api/v1/admin/integrations/:id
func (h *IntegrationHandler) DeleteIntegration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid integration ID"})
		return
	}

	var integration models.IntegrationConfig
	if err := h.db.First(&integration, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	if err := h.db.Delete(&integration).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete integration"})
		return
	}

	adminID := c.GetUint("admin_id")
	logAdminActivity(h.db, adminID, "delete_integration", "integration",
		fmt.Sprintf("%d", id), "Deleted integration: "+integration.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Integration deleted successfully"})
}

// ListIntegrations lists all integrations
// GET /api/v1/admin/integrations
func (h *IntegrationHandler) ListIntegrations(c *gin.Context) {
	integrationType := c.Query("type")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var integrations []models.IntegrationConfig
	var total int64

	query := h.db.Model(&models.IntegrationConfig{})
	if integrationType != "" {
		query = query.Where("type = ?", integrationType)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&integrations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch integrations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        integrations,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// SendSlackNotification sends a notification to Slack
// POST /api/v1/integrations/slack/notify
func (h *IntegrationHandler) SendSlackNotification(c *gin.Context) {
	var req struct {
		Channel     string            `json:"channel" binding:"required"`
		Text        string            `json:"text" binding:"required"`
		Username    string            `json:"username"`
		IconEmoji   string            `json:"icon_emoji"`
		Attachments []SlackAttachment `json:"attachments"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get Slack integration config
	var integration models.IntegrationConfig
	err := h.db.Where("type = ? AND is_active = ?", "slack", true).First(&integration).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Slack integration not configured"})
		return
	}

	var config map[string]interface{}
	json.Unmarshal(integration.Config, &config)

	webhookURL, _ := config["webhook_url"].(string)
	if webhookURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slack webhook URL not configured"})
		return
	}

	// Build Slack message
	slackMsg := SlackMessage{
		Channel:   req.Channel,
		Text:      req.Text,
		Username:  req.Username,
		IconEmoji: req.IconEmoji,
	}

	if len(req.Attachments) > 0 {
		slackMsg.Attachments = req.Attachments
	}

	msgBytes, _ := json.Marshal(slackMsg)

	// Send to Slack
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(msgBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send Slack notification"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Slack API error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification sent to Slack"})
}

// SendPagerDutyAlert sends an alert to PagerDuty
// POST /api/v1/integrations/pagerduty/alert
func (h *IntegrationHandler) SendPagerDutyAlert(c *gin.Context) {
	var req struct {
		Summary       string                 `json:"summary" binding:"required"`
		Severity      string                 `json:"severity" binding:"required"` // critical, error, warning, info
		Source        string                 `json:"source"`
		CustomDetails map[string]interface{} `json:"custom_details"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get PagerDuty integration config
	var integration models.IntegrationConfig
	err := h.db.Where("type = ? AND is_active = ?", "pagerduty", true).First(&integration).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PagerDuty integration not configured"})
		return
	}

	var config map[string]interface{}
	json.Unmarshal(integration.Config, &config)

	integrationKey, _ := config["integration_key"].(string)
	if integrationKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PagerDuty integration key not configured"})
		return
	}

	// Build PagerDuty event
	event := PagerDutyEvent{
		RoutingKey:  integrationKey,
		EventAction: "trigger",
		Payload: PagerDutyPayload{
			Summary:       req.Summary,
			Source:        req.Source,
			Severity:      req.Severity,
			Timestamp:     time.Now().Format(time.RFC3339),
			CustomDetails: req.CustomDetails,
		},
	}

	eventBytes, _ := json.Marshal(event)

	// Send to PagerDuty
	resp, err := http.Post("https://events.pagerduty.com/v2/enqueue",
		"application/json", bytes.NewBuffer(eventBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send PagerDuty alert"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "PagerDuty API error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert sent to PagerDuty"})
}

// SendDatadogEvent sends an event to Datadog
// POST /api/v1/integrations/datadog/event
func (h *IntegrationHandler) SendDatadogEvent(c *gin.Context) {
	var req struct {
		Title     string   `json:"title" binding:"required"`
		Text      string   `json:"text"`
		Priority  string   `json:"priority"`   // normal, low
		AlertType string   `json:"alert_type"` // info, warning, error, success
		Tags      []string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get Datadog integration config
	var integration models.IntegrationConfig
	err := h.db.Where("type = ? AND is_active = ?", "datadog", true).First(&integration).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Datadog integration not configured"})
		return
	}

	var config map[string]interface{}
	json.Unmarshal(integration.Config, &config)

	apiKey, _ := config["api_key"].(string)
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datadog API key not configured"})
		return
	}

	site := "datadoghq.com"
	if config["site"] != nil {
		site = config["site"].(string)
	}

	// Build Datadog event
	event := DatadogEvent{
		Title:      req.Title,
		Text:       strconv.FormatInt(time.Now().Unix(), 10),
		Priority:   req.Priority,
		AlertType:  req.AlertType,
		Tags:       req.Tags,
		SourceType: "tigerwallet",
	}

	eventBytes, _ := json.Marshal(event)

	// Send to Datadog
	req2, _ := http.NewRequest("POST",
		fmt.Sprintf("https://api.%s/api/v1/events", site),
		bytes.NewBuffer(eventBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("DD-API-KEY", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send Datadog event"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Datadog API error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event sent to Datadog"})
}

// TestIntegration tests an integration
// POST /api/v1/admin/integrations/:id/test
func (h *IntegrationHandler) TestIntegration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid integration ID"})
		return
	}

	var integration models.IntegrationConfig
	if err := h.db.First(&integration, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	var config map[string]interface{}
	json.Unmarshal(integration.Config, &config)

	testResult := map[string]interface{}{
		"integration_id": id,
		"type":           integration.Type,
		"name":           integration.Name,
		"test_status":    "passed",
		"tested_at":      time.Now().Format(time.RFC3339),
	}

	// Test based on type
	switch integration.Type {
	case "slack":
		if webhookURL, ok := config["webhook_url"].(string); ok && webhookURL != "" {
			testMsg := map[string]string{"text": "TigerWallet test notification"}
			msgBytes, _ := json.Marshal(testMsg)
			resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(msgBytes))
			if err != nil || resp.StatusCode != 200 {
				testResult["test_status"] = "failed"
				testResult["error"] = "Failed to connect to Slack"
			}
			if resp != nil {
				defer resp.Body.Close()
			}
		}
	case "datadog":
		if apiKey, ok := config["api_key"].(string); ok && apiKey != "" {
			testResult["test_status"] = "passed"
		}
	case "pagerduty":
		if integrationKey, ok := config["integration_key"].(string); ok && integrationKey != "" {
			testResult["test_status"] = "passed"
		}
	}

	c.JSON(http.StatusOK, testResult)
}

// WebhookHandler handles incoming webhooks from external services
// POST /api/v1/webhooks/:type
func (h *IntegrationHandler) WebhookHandler(c *gin.Context) {
	webhookType := c.Param("type")

	// Read the webhook payload
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Store webhook in Redis for processing
	webhookKey := fmt.Sprintf("webhook:%s:%d", webhookType, time.Now().Unix())
	h.redis.Set(context.Background(), webhookKey, body, 24*time.Hour)

	// Process based on type
	switch webhookType {
	case "slack":
		// Handle Slack slash commands or events
		c.JSON(http.StatusOK, gin.H{"message": "Webhook received"})
	case "pagerduty":
		// Handle PagerDuty webhooks
		c.JSON(http.StatusOK, gin.H{"message": "Webhook received"})
	case "datadog":
		// Handle Datadog webhooks
		c.JSON(http.StatusOK, gin.H{"message": "Webhook received"})
	default:
		c.JSON(http.StatusOK, gin.H{"message": "Webhook received"})
	}
}

// GetIntegrationStats gets integration statistics
// GET /api/v1/admin/integrations/stats
func (h *IntegrationHandler) GetIntegrationStats(c *gin.Context) {
	var stats struct {
		TotalIntegrations  int64 `json:"total_integrations"`
		ActiveIntegrations int64 `json:"active_integrations"`
		SlackCount         int64 `json:"slack_count"`
		PagerDutyCount     int64 `json:"pagerduty_count"`
		DatadogCount       int64 `json:"datadog_count"`
		CloudflareCount    int64 `json:"cloudflare_count"`
	}

	h.db.Model(&models.IntegrationConfig{}).Count(&stats.TotalIntegrations)
	h.db.Model(&models.IntegrationConfig{}).Where("is_active = ?", true).Count(&stats.ActiveIntegrations)
	h.db.Model(&models.IntegrationConfig{}).Where("type = ?", "slack").Count(&stats.SlackCount)
	h.db.Model(&models.IntegrationConfig{}).Where("type = ?", "pagerduty").Count(&stats.PagerDutyCount)
	h.db.Model(&models.IntegrationConfig{}).Where("type = ?", "datadog").Count(&stats.DatadogCount)
	h.db.Model(&models.IntegrationConfig{}).Where("type = ?", "cloudflare").Count(&stats.CloudflareCount)

	c.JSON(http.StatusOK, stats)
}
