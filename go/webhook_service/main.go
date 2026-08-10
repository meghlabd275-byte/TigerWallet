package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ListenAddr  string
	DatabaseURL string
	SecretKey   string
	MaxRetries  int
	Timeout     time.Duration
	MaxWorkers  int
}

var config = Config{
	ListenAddr:  getEnv("WEBHOOK_LISTEN_ADDR", ":9002"),
	DatabaseURL: getEnv("DATABASE_URL", "postgres://tigerwallet:password@localhost:5432/tigerwallet?sslmode=disable"),
	SecretKey:   getEnv("WEBHOOK_SECRET", "your-secret-key"),
	MaxRetries:  3,
	Timeout:     time.Second * 30,
	MaxWorkers:  20,
}

// ============================================================================
// Models
// ============================================================================

type Webhook struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	URL         string            `json:"url"`
	Events      []string          `json:"events"`
	Secret      string            `json:"secret"`
	Active      bool              `json:"active"`
	RetryPolicy RetryPolicy       `json:"retry_policy"`
	Headers     map[string]string `json:"headers,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type RetryPolicy struct {
	MaxRetries        int           `json:"max_retries"`
	RetryInterval     time.Duration `json:"retry_interval"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
}

type WebhookEvent struct {
	ID           string                 `json:"id"`
	WebhookID    string                 `json:"webhook_id"`
	Event        string                 `json:"event"`
	Payload      map[string]interface{} `json:"payload"`
	Status       string                 `json:"status"` // pending, delivered, failed
	Attempts     int                    `json:"attempts"`
	ResponseCode int                    `json:"response_code"`
	ResponseBody string                 `json:"response_body,omitempty"`
	DeliveredAt  *time.Time             `json:"delivered_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

type WebhookDelivery struct {
	EventID   string                 `json:"event_id"`
	WebhookID string                 `json:"webhook_id"`
	URL       string                 `json:"url"`
	Event     string                 `json:"event"`
	Payload   map[string]interface{} `json:"payload"`
	Headers   map[string]string      `json:"headers"`
	Signature string                 `json:"signature"`
	Attempts  int                    `json:"attempts"`
	Status    string                 `json:"status"`
}

type WebhookStats struct {
	TotalEvents    int64            `json:"total_events"`
	Delivered      int64            `json:"delivered"`
	Failed         int64            `json:"failed"`
	Pending        int64            `json:"pending"`
	DeliveryRate   float64          `json:"delivery_rate"`
	ByEvent        map[string]int64 `json:"by_event"`
	AverageLatency float64          `json:"average_latency_ms"`
}

// ============================================================================
// Webhook Service
// ============================================================================

type WebhookService struct {
	db         *sql.DB
	webhooks   map[string]*Webhook
	webhooksMu sync.RWMutex
	queue      chan *WebhookDelivery
	workers    int
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	stats      WebhookStats
	statsMu    sync.RWMutex
}

func NewWebhookService() (*WebhookService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	svc := &WebhookService{
		webhooks: make(map[string]*Webhook),
		queue:    make(chan *WebhookDelivery, 10000),
		workers:  config.MaxWorkers,
		ctx:      ctx,
		cancel:   cancel,
		stats: WebhookStats{
			ByEvent: make(map[string]int64),
		},
	}

	// Initialize database
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		fmt.Printf("Warning: Database connection failed: %v\n", err)
	} else {
		svc.db = db
	}

	// Initialize default webhooks
	svc.initializeWebhooks()

	return svc, nil
}

func (s *WebhookService) initializeWebhooks() {
	// Default webhook events
	defaultEvents := []string{
		"wallet.created",
		"wallet.deleted",
		"transaction.pending",
		"transaction.confirmed",
		"transaction.failed",
		"swap.executed",
		"stake.started",
		"stake.completed",
		"withdraw.requested",
		"withdraw.completed",
		"deposit.received",
		"kyc.submitted",
		"kyc.approved",
		"kyc.rejected",
	}

	_ = defaultEvents
}

func (s *WebhookService) Start() error {
	fmt.Println("Starting Webhook Service...")

	// Start workers
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	// Start HTTP server
	go s.startHTTPServer()

	fmt.Println("Webhook Service started successfully")
	return nil
}

func (s *WebhookService) Stop() {
	fmt.Println("Stopping Webhook Service...")
	s.cancel()
	s.wg.Wait()
	close(s.queue)

	if s.db != nil {
		s.db.Close()
	}

	fmt.Println("Webhook Service stopped")
}

func (s *WebhookService) worker(id int) {
	defer s.wg.Done()
	fmt.Printf("Webhook worker %d started\n", id)

	client := &http.Client{
		Timeout: config.Timeout,
	}

	for {
		select {
		case <-s.ctx.Done():
			fmt.Printf("Webhook worker %d stopping\n", id)
			return
		case delivery, ok := <-s.queue:
			if !ok {
				return
			}
			s.processDelivery(client, delivery)
		}
	}
}

func (s *WebhookService) processDelivery(client *http.Client, delivery *WebhookDelivery) {
	fmt.Printf("Processing webhook delivery: %s to %s\n", delivery.Event, delivery.URL)

	startTime := time.Now()

	// Prepare request
	payload, _ := json.Marshal(delivery.Payload)
	signature := s.generateSignature(payload, config.SecretKey)

	req, err := http.NewRequest("POST", delivery.URL, bytes.NewBuffer(payload))
	if err != nil {
		s.handleFailure(delivery, 0, err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", delivery.Event)
	req.Header.Set("X-Webhook-ID", delivery.EventID)

	for key, value := range delivery.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := client.Do(req)
	latency := time.Since(startTime).Milliseconds()

	if err != nil {
		s.handleFailure(delivery, 0, err.Error())
		return
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.handleSuccess(delivery, resp.StatusCode, string(body), latency)
	} else {
		s.handleFailure(delivery, resp.StatusCode, string(body))
	}

	// Update stats
	s.updateStats(delivery.Event, resp.StatusCode, latency)
}

func (s *WebhookService) handleSuccess(delivery *WebhookDelivery, statusCode int, body string, latency int64) {
	now := time.Now()

	fmt.Printf("Webhook delivered successfully: %s to %s (latency: %dms)\n",
		delivery.Event, delivery.URL, latency)

	s.statsMu.Lock()
	s.stats.Delivered++
	s.statsMu.Unlock()

	// Would update database
	_ = statusCode
	_ = body
	_ = now
}

func (s *WebhookService) handleFailure(delivery *WebhookDelivery, statusCode int, error string) {
	delivery.Attempts++

	fmt.Printf("Webhook delivery failed: %s to %s (attempt %d): %s\n",
		delivery.Event, delivery.URL, delivery.Attempts, error)

	s.statsMu.Lock()
	s.stats.Failed++
	s.statsMu.Unlock()

	// Retry if attempts < max
	if delivery.Attempts < config.MaxRetries {
		time.Sleep(time.Second * time.Duration(delivery.Attempts))
		s.queue <- delivery
	}
}

func (s *WebhookService) updateStats(event string, statusCode int, latency int64) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	s.stats.TotalEvents++
	s.stats.ByEvent[event]++

	// Update average latency
	oldAvg := s.stats.AverageLatency
	count := float64(s.stats.TotalEvents)
	s.stats.AverageLatency = ((oldAvg * (count - 1)) + float64(latency)) / count

	// Update delivery rate
	if s.stats.TotalEvents > 0 {
		s.stats.DeliveryRate = float64(s.stats.Delivered) / float64(s.stats.TotalEvents) * 100
	}
}

func (s *WebhookService) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

func (s *WebhookService) TriggerEvent(event string, payload map[string]interface{}) {
	s.webhooksMu.RLock()
	defer s.webhooksMu.RUnlock()

	for _, webhook := range s.webhooks {
		if !webhook.Active {
			continue
		}

		// Check if webhook listens to this event
		listenEvent := false
		for _, e := range webhook.Events {
			if e == event || e == "*" {
				listenEvent = true
				break
			}
		}

		if !listenEvent {
			continue
		}

		delivery := &WebhookDelivery{
			EventID:   generateID(),
			WebhookID: webhook.ID,
			URL:       webhook.URL,
			Event:     event,
			Payload:   payload,
			Headers:   webhook.Headers,
			Signature: "",
			Attempts:  0,
			Status:    "pending",
		}

		s.queue <- delivery
	}
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *WebhookService) startHTTPServer() {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	router.GET("/webhooks", s.listWebhooksHandler)
	router.GET("/webhooks/:id", s.getWebhookHandler)
	router.POST("/webhooks", s.createWebhookHandler)
	router.PUT("/webhooks/:id", s.updateWebhookHandler)
	router.DELETE("/webhooks/:id", s.deleteWebhookHandler)
	router.POST("/webhooks/:id/test", s.testWebhookHandler)

	router.GET("/events", s.listEventsHandler)
	router.GET("/events/:id", s.getEventHandler)

	router.GET("/stats", s.getStatsHandler)

	router.POST("/trigger/:event", s.triggerEventHandler)

	fmt.Printf("Webhook API server starting on %s\n", config.ListenAddr)
	router.Run(config.ListenAddr)
}

func (s *WebhookService) listWebhooksHandler(c *gin.Context) {
	userID := c.Query("user_id")

	s.webhooksMu.RLock()
	webhooks := make([]Webhook, 0)
	for _, w := range s.webhooks {
		if userID == "" || w.UserID == userID {
			webhooks = append(webhooks, *w)
		}
	}
	s.webhooksMu.RUnlock()

	c.JSON(200, webhooks)
}

func (s *WebhookService) getWebhookHandler(c *gin.Context) {
	id := c.Param("id")

	s.webhooksMu.RLock()
	webhook, ok := s.webhooks[id]
	s.webhooksMu.RUnlock()

	if !ok {
		c.JSON(404, gin.H{"error": "webhook not found"})
		return
	}

	c.JSON(200, webhook)
}

func (s *WebhookService) createWebhookHandler(c *gin.Context) {
	var webhook Webhook
	if err := c.ShouldBindJSON(&webhook); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if webhook.ID == "" {
		webhook.ID = generateID()
	}

	if webhook.Secret == "" {
		webhook.Secret = generateSecret()
	}

	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = time.Now()

	s.webhooksMu.Lock()
	s.webhooks[webhook.ID] = &webhook
	s.webhooksMu.Unlock()

	c.JSON(200, webhook)
}

func (s *WebhookService) updateWebhookHandler(c *gin.Context) {
	id := c.Param("id")

	var webhook Webhook
	if err := c.ShouldBindJSON(&webhook); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	webhook.ID = id
	webhook.UpdatedAt = time.Now()

	s.webhooksMu.Lock()
	s.webhooks[id] = &webhook
	s.webhooksMu.Unlock()

	c.JSON(200, webhook)
}

func (s *WebhookService) deleteWebhookHandler(c *gin.Context) {
	id := c.Param("id")

	s.webhooksMu.Lock()
	delete(s.webhooks, id)
	s.webhooksMu.Unlock()

	c.JSON(200, gin.H{"status": "ok"})
}

func (s *WebhookService) testWebhookHandler(c *gin.Context) {
	id := c.Param("id")

	s.webhooksMu.RLock()
	webhook, ok := s.webhooks[id]
	s.webhooksMu.RUnlock()

	if !ok {
		c.JSON(404, gin.H{"error": "webhook not found"})
		return
	}

	// Send test event
	testPayload := map[string]interface{}{
		"event":     "test",
		"message":   "This is a test webhook",
		"timestamp": time.Now().Unix(),
	}

	delivery := &WebhookDelivery{
		EventID:   generateID(),
		WebhookID: id,
		URL:       webhook.URL,
		Event:     "test",
		Payload:   testPayload,
		Headers:   webhook.Headers,
		Attempts:  0,
		Status:    "pending",
	}

	s.queue <- delivery

	c.JSON(200, gin.H{"status": "test_sent", "delivery_id": delivery.EventID})
}

func (s *WebhookService) listEventsHandler(c *gin.Context) {
	webhookID := c.Query("webhook_id")
	limit := c.DefaultQuery("limit", "50")

	// Would query database in production
	events := []WebhookEvent{}

	_ = webhookID
	_ = limit

	c.JSON(200, events)
}

func (s *WebhookService) getEventHandler(c *gin.Context) {
	id := c.Param("id")

	event := WebhookEvent{
		ID:        id,
		WebhookID: "webhook_123",
		Event:     "transaction.confirmed",
		Payload:   map[string]interface{}{},
		Status:    "delivered",
		Attempts:  1,
		CreatedAt: time.Now(),
	}

	c.JSON(200, event)
}

func (s *WebhookService) getStatsHandler(c *gin.Context) {
	s.statsMu.RLock()
	stats := s.stats
	s.statsMu.RUnlock()

	c.JSON(200, stats)
}

func (s *WebhookService) triggerEventHandler(c *gin.Context) {
	event := c.Param("event")

	var payload map[string]interface{}
	c.ShouldBindJSON(&payload)

	if payload == nil {
		payload = map[string]interface{}{
			"triggered_at": time.Now().Unix(),
		}
	}

	s.TriggerEvent(event, payload)

	c.JSON(200, gin.H{"status": "triggered", "event": event})
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateID() string {
	return fmt.Sprintf("wh_%d_%s", time.Now().UnixNano(), randomString(8))
}

func generateSecret() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Main
// ============================================================================

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println("============================================")
	fmt.Println("TigerWallet Webhook Service")
	fmt.Println("============================================")

	svc, err := NewWebhookService()
	if err != nil {
		fmt.Printf("Failed to create webhook service: %v\n", err)
		os.Exit(1)
	}

	if err := svc.Start(); err != nil {
		fmt.Printf("Failed to start webhook service: %v\n", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	svc.Stop()

	fmt.Println("Webhook service stopped")
}
