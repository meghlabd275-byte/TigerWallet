package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// WEBHOOK TYPES
// ============================================================================

type Webhook struct {
	ID          string    `json:"id"`
	ClientID    string    `json:"clientId"`
	URL         string    `json:"url"`
	Secret      string    `json:"secret"`
	Events      []string  `json:"events"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	FailedCount int       `json:"failedCount"`
}

type WebhookEvent struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	ClientID string      `json:"clientId"`
	Payload  interface{} `json:"payload"`
	SentAt   time.Time   `json:"sentAt"`
	RetryAt  time.Time   `json:"retryAt"`
}

type WebhookDelivery struct {
	ID          string    `json:"id"`
	WebhookID   string    `json:"webhookId"`
	EventID     string    `json:"eventId"`
	URL         string    `json:"url"`
	StatusCode  int       `json:"statusCode"`
	Response    string    `json:"response"`
	DeliveredAt time.Time `json:"deliveredAt"`
	DurationMs  int64     `json:"durationMs"`
}

// ============================================================================
// WEBHOOK SERVICE
// ============================================================================

type WebhookService struct {
	mu         sync.RWMutex
	webhooks   map[string]*Webhook
	redis      *redis.Client
	httpClient *http.Client
	queue      chan *WebhookEvent
	workers    int
}

func NewWebhookService(redisClient *redis.Client, workers int) *WebhookService {
	svc := &WebhookService{
		webhooks: make(map[string]*Webhook),
		redis:    redisClient,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		queue:   make(chan *WebhookEvent, 1000),
		workers: workers,
	}

	// Start worker pool
	for i := 0; i < workers; i++ {
		go svc.worker()
	}

	return svc
}

func (s *WebhookService) worker() {
	for event := range s.queue {
		s.processEvent(event)
	}
}

func (s *WebhookService) processEvent(event *WebhookEvent) {
	s.mu.RLock()
	var activeWebhooks []*Webhook
	for _, wh := range s.webhooks {
		if wh.IsActive && contains(wh.Events, event.Type) {
			activeWebhooks = append(activeWebhooks, wh)
		}
	}
	s.mu.RUnlock()

	for _, wh := range activeWebhooks {
		s.deliverWebhook(wh, event)
	}
}

func (s *WebhookService) deliverWebhook(wh *Webhook, event *WebhookEvent) {
	start := time.Now()

	// Sign payload
	payload, _ := json.Marshal(event.Payload)
	signature := s.signPayload(payload, wh.Secret)

	req, err := http.NewRequest("POST", wh.URL, bytes.NewBuffer(payload))
	if err != nil {
		s.recordDelivery(wh.ID, event.ID, wh.URL, 0, err.Error(), time.Since(start))
		s.incrementFailedCount(wh.ID)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", event.Type)
	req.Header.Set("X-Webhook-ID", event.ID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.recordDelivery(wh.ID, event.ID, wh.URL, 0, err.Error(), time.Since(start))
		s.incrementFailedCount(wh.ID)
		return
	}
	defer resp.Body.Close()

	responseBody := ""
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	responseBody = buf.String()

	s.recordDelivery(wh.ID, event.ID, wh.URL, resp.StatusCode, responseBody, time.Since(start))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.resetFailedCount(wh.ID)
	} else {
		s.incrementFailedCount(wh.ID)
	}
}

func (s *WebhookService) signPayload(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

func (s *WebhookService) recordDelivery(webhookID, eventID, url string, statusCode int, response string, duration time.Duration) {
	delivery := &WebhookDelivery{
		ID:          uuid.New().String(),
		WebhookID:   webhookID,
		EventID:     eventID,
		URL:         url,
		StatusCode:  statusCode,
		Response:    response,
		DeliveredAt: time.Now(),
		DurationMs:  duration.Milliseconds(),
	}

	// Store in Redis
	key := fmt.Sprintf("webhook:delivery:%s:%s", webhookID, eventID)
	data, _ := json.Marshal(delivery)
	s.redis.Set(context.Background(), key, data, 7*24*time.Hour)
}

func (s *WebhookService) incrementFailedCount(webhookID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if wh, ok := s.webhooks[webhookID]; ok {
		wh.FailedCount++
		if wh.FailedCount >= 5 {
			wh.IsActive = false
		}
	}
}

func (s *WebhookService) resetFailedCount(webhookID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if wh, ok := s.webhooks[webhookID]; ok {
		wh.FailedCount = 0
	}
}

// ============================================================================
// WEBHOOK HTTP HANDLERS
// ============================================================================

func (s *WebhookService) RegisterRoutes(router *gin.RouterGroup) {
	webhooks := router.Group("/webhooks")
	{
		webhooks.POST("", s.createWebhook)
		webhooks.GET("", s.listWebhooks)
		webhooks.GET("/:id", s.getWebhook)
		webhooks.PUT("/:id", s.updateWebhook)
		webhooks.DELETE("/:id", s.deleteWebhook)
		webhooks.POST("/:id/test", s.testWebhook)
		webhooks.GET("/:id/deliveries", s.listDeliveries)
	}
}

func (s *WebhookService) createWebhook(c *gin.Context) {
	var req struct {
		ClientID string   `json:"clientId" binding:"required"`
		URL      string   `json:"url" binding:"required,url"`
		Events   []string `json:"events" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	webhook := &Webhook{
		ID:        uuid.New().String(),
		ClientID:  req.ClientID,
		URL:       req.URL,
		Secret:    generateSecret(),
		Events:    req.Events,
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.webhooks[webhook.ID] = webhook
	s.mu.Unlock()

	c.JSON(http.StatusCreated, webhook)
}

func (s *WebhookService) listWebhooks(c *gin.Context) {
	clientID := c.Query("clientId")

	s.mu.RLock()
	defer s.mu.RUnlock()

	var webhooks []*Webhook
	for _, wh := range s.webhooks {
		if clientID == "" || wh.ClientID == clientID {
			webhooks = append(webhooks, wh)
		}
	}

	c.JSON(http.StatusOK, webhooks)
}

func (s *WebhookService) getWebhook(c *gin.Context) {
	id := c.Param("id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	if wh, ok := s.webhooks[id]; ok {
		c.JSON(http.StatusOK, wh)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
}

func (s *WebhookService) updateWebhook(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)

	s.mu.Lock()
	defer s.mu.Unlock()

	if wh, ok := s.webhooks[id]; ok {
		if url, ok := updates["url"].(string); ok {
			wh.URL = url
		}
		if events, ok := updates["events"].([]string); ok {
			wh.Events = events
		}
		if active, ok := updates["isActive"].(bool); ok {
			wh.IsActive = active
		}
		c.JSON(http.StatusOK, wh)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
}

func (s *WebhookService) deleteWebhook(c *gin.Context) {
	id := c.Param("id")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.webhooks[id]; ok {
		delete(s.webhooks, id)
		c.JSON(http.StatusOK, gin.H{"message": "Webhook deleted"})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
}

func (s *WebhookService) testWebhook(c *gin.Context) {
	id := c.Param("id")

	s.mu.RLock()
	wh, ok := s.webhooks[id]
	s.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		return
	}

	// Send test event
	event := &WebhookEvent{
		ID:       uuid.New().String(),
		Type:     "test",
		ClientID: wh.ClientID,
		Payload: map[string]interface{}{
			"message":   "This is a test webhook",
			"timestamp": time.Now().Unix(),
		},
	}

	s.queue <- event

	c.JSON(http.StatusAccepted, gin.H{"message": "Test event queued"})
}

func (s *WebhookService) listDeliveries(c *gin.Context) {
	webhookID := c.Param("id")

	// Get from Redis
	pattern := fmt.Sprintf("webhook:delivery:%s:*", webhookID)
	keys, _ := s.redis.Keys(context.Background(), pattern).Result()

	var deliveries []WebhookDelivery
	for _, key := range keys {
		data, _ := s.redis.Get(context.Background(), key).Result()
		var delivery WebhookDelivery
		json.Unmarshal([]byte(data), &delivery)
		deliveries = append(deliveries, delivery)
	}

	c.JSON(http.StatusOK, deliveries)
}

// ============================================================================
// EVENT TRIGGERING
// ============================================================================

func (s *WebhookService) TriggerEvent(eventType, clientID string, payload interface{}) {
	event := &WebhookEvent{
		ID:       uuid.New().String(),
		Type:     eventType,
		ClientID: clientID,
		Payload:  payload,
		SentAt:   time.Now(),
		RetryAt:  time.Now().Add(5 * time.Minute),
	}

	select {
	case s.queue <- event:
	default:
		// Queue full, log warning
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func generateSecret() string {
	bytes := make([]byte, 32)
	for i := range bytes {
		bytes[i] = byte(i * 7 % 256)
	}
	return hex.EncodeToString(bytes)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ============================================================================
// WEBHOOK EVENTS
// ============================================================================

// Event types
const (
	EventUserCreated         = "user.created"
	EventUserUpdated         = "user.updated"
	EventUserKycApproved     = "user.kyc.approved"
	EventUserKycRejected     = "user.kyc.rejected"
	EventDeposit             = "transaction.deposit"
	EventWithdrawal          = "transaction.withdrawal"
	EventTrade               = "trade.executed"
	EventWhiteLabelCreated   = "whitelabel.created"
	EventWhiteLabelApproved  = "whitelabel.approved"
	EventWhiteLabelSuspended = "whitelabel.suspended"
	EventAdminLogin          = "admin.login"
	EventAdminAction         = "admin.action"
	EventPlatformAlert       = "platform.alert"
)
