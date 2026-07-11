package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// Configuration
type Config struct {
	Port            string
	RedisURL        string
	FCMCredentials  string
	APNSKeyPath     string
	FCMProjectID    string
}

// Notification Types
type NotificationType string

const (
	TypePriceAlert       NotificationType = "price_alert"
	TypeTransaction      NotificationType = "transaction"
	TypeAirdrop          NotificationType = "airdrop"
	TypeGasPrice         NotificationType = "gas_price"
	TypePortfolioChange  NotificationType = "portfolio_change"
	TypeStakingReward    NotificationType = "staking_reward"
	TypeDAppNotification NotificationType = "dapp_notification"
)

// Notification Priority
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

// Device Type
type DeviceType string

const (
	DeviceiOS     DeviceType = "ios"
	DeviceAndroid DeviceType = "android"
	DeviceWeb     DeviceType = "web"
)

// Notification
type Notification struct {
	ID            string          `json:"id"`
	Type          NotificationType `json:"type"`
	Title         string          `json:"title"`
	Body          string          `json:"body"`
	Data          map[string]string `json:"data,omitempty"`
	Priority      Priority        `json:"priority"`
	DeviceToken   string          `json:"device_token"`
	DeviceType    DeviceType      `json:"device_type"`
	ScheduledAt   *time.Time      `json:"scheduled_at,omitempty"`
	ExpiresAt     *time.Time      `json:"expires_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// Device Registration
type DeviceRegistration struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	DeviceToken  string     `json:"device_token"`
	DeviceType   DeviceType `json:"device_type"`
	AppVersion   string     `json:"app_version"`
	LastActiveAt time.Time  `json:"last_active_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Price Alert
type PriceAlert struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	TokenAddress  string  `json:"token_address"`
	TokenSymbol  string  `json:"token_symbol"`
	TargetPrice   float64 `json:"target_price"`
	Condition     string  `json:"condition"` // "above" or "below"
	IsActive      bool    `json:"is_active"`
	TriggeredAt   *time.Time `json:"triggered_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// AlertEvent
type AlertEvent struct {
	Type    string  `json:"type"`
	Symbol  string  `json:"symbol"`
	Price   float64 `json:"price"`
	Change  float64 `json:"change"`
	Time    int64   `json:"time"`
}

// Service
type NotificationService struct {
	config       *Config
	redis        *redis.Client
	fcmClient    *messaging.Client
	httpServer   *http.Server
}

// New creates a new notification service
func New(config *Config) (*NotificationService, error) {
	// Initialize Redis
	redisOpts, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	
	redisClient := redis.NewClient(redisOpts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	// Initialize FCM
	var fcmClient *messaging.Client
	if config.FCMProjectID != "" {
		opt := option.WithCredentialsFile(config.FCMCredentials)
		app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: config.FCMProjectID}, opt)
		if err != nil {
			log.Printf("Warning: FCM initialization failed: %v", err)
		} else {
			fcmClient, err = app.Messaging(ctx)
			if err != nil {
				log.Printf("Warning: FCM client creation failed: %v", err)
			}
		}
	}

	return &NotificationService{
		config:    config,
		redis:     redisClient,
		fcmClient: fcmClient,
	}, nil
}

// Start starts the notification service
func (s *NotificationService) Start() error {
	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)
	
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	
	// Health check
	router.GET("/health", s.healthCheck)
	
	// API routes
	api := router.Group("/api/v1")
	{
		// Device registration
		api.POST("/devices", s.registerDevice)
		api.DELETE("/devices/:id", s.unregisterDevice)
		api.PUT("/devices/:id", s.updateDevice)
		
		// Notifications
		api.POST("/notifications", s.sendNotification)
		api.POST("/notifications/batch", s.sendBatchNotifications)
		api.GET("/notifications/:id", s.getNotification)
		api.GET("/notifications", s.listNotifications)
		
		// Price alerts
		api.POST("/alerts/price", s.createPriceAlert)
		api.GET("/alerts/price", s.listPriceAlerts)
		api.PUT("/alerts/price/:id", s.updatePriceAlert)
		api.DELETE("/alerts/price/:id", s.deletePriceAlert)
		
		// Subscriptions
		api.POST("/subscribe", s.subscribe)
		api.POST("/unsubscribe", s.unsubscribe)
	}
	
	// Start server
	s.httpServer = &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	
	go func() {
		log.Printf("Starting notification service on port %s", s.config.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()
	
	// Start background workers
	go s.startPriceAlertWorker()
	go s.startCleanupWorker()
	
	return nil
}

// Stop stops the notification service
func (s *NotificationService) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	
	if s.redis != nil {
		return s.redis.Close()
	}
	
	return nil
}

// Register device
func (s *NotificationService) registerDevice(c *gin.Context) {
	var reg DeviceRegistration
	if err := c.ShouldBindJSON(&reg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	reg.ID = uuid.New().String()
	reg.CreatedAt = time.Now()
	reg.LastActiveAt = time.Now()
	
	// Store in Redis
	key := fmt.Sprintf("device:%s", reg.ID)
	data, _ := json.Marshal(reg)
	if err := s.redis.Set(c.Request.Context(), key, data, 0).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device"})
		return
	}
	
	// Add to user's device list
	userDevicesKey := fmt.Sprintf("user:%s:devices", reg.UserID)
	s.redis.SAdd(c.Request.Context(), userDevicesKey, reg.ID)
	
	c.JSON(http.StatusCreated, reg)
}

// Unregister device
func (s *NotificationService) unregisterDevice(c *gin.Context) {
	id := c.Param("id")
	
	// Get device
	key := fmt.Sprintf("device:%s", id)
	data, err := s.redis.Get(c.Request.Context(), key).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}
	
	var reg DeviceRegistration
	json.Unmarshal(data, &reg)
	
	// Remove from user's device list
	userDevicesKey := fmt.Sprintf("user:%s:devices", reg.UserID)
	s.redis.SRem(c.Request.Context(), userDevicesKey, id)
	
	// Delete device
	s.redis.Del(c.Request.Context(), key)
	
	c.JSON(http.StatusOK, gin.H{"message": "Device unregistered"})
}

// Update device
func (s *NotificationService) updateDevice(c *gin.Context) {
	id := c.Param("id")
	
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get existing device
	key := fmt.Sprintf("device:%s", id)
	data, err := s.redis.Get(c.Request.Context(), key).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}
	
	var reg DeviceRegistration
	json.Unmarshal(data, &reg)
	
	// Update fields
	if token, ok := updates["device_token"].(string); ok {
		reg.DeviceToken = token
	}
	if version, ok := updates["app_version"].(string); ok {
		reg.AppVersion = version
	}
	reg.LastActiveAt = time.Now()
	
	// Save
	data, _ = json.Marshal(reg)
	s.redis.Set(c.Request.Context(), key, data, 0)
	
	c.JSON(http.StatusOK, reg)
}

// Send notification
func (s *NotificationService) sendNotification(c *gin.Context) {
	var notif Notification
	if err := c.ShouldBindJSON(&notif); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	notif.ID = uuid.New().String()
	notif.CreatedAt = time.Now()
	
	// Validate device type and send
	if notif.DeviceType == DeviceAndroid && s.fcmClient != nil {
		s.sendFCM(c.Request.Context(), &notif)
	} else if notif.DeviceType == DeviceiOS {
		s.sendAPNS(c.Request.Context(), &notif)
	}
	
	// Store notification
	notifKey := fmt.Sprintf("notification:%s", notif.ID)
	data, _ := json.Marshal(notif)
	s.redis.Set(c.Request.Context(), notifKey, data, 7*24*time.Hour)
	
	c.JSON(http.StatusCreated, notif)
}

// Send FCM
func (s *NotificationService) sendFCM(ctx context.Context, notif *Notification) error {
	if s.fcmClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}
	
	message := &messaging.Message{
		Token: notif.DeviceToken,
		Notification: &messaging.Notification{
			Title: notif.Title,
			Body:  notif.Body,
		},
		Data: notif.Data,
		Android: &messaging.AndroidConfig{
			Priority: string(notif.Priority),
		},
	}
	
	_, err := s.fcmClient.Send(ctx, message)
	return err
}

// Send APNS (placeholder - would need APNS implementation)
func (s *NotificationService) sendAPNS(ctx context.Context, notif *Notification) error {
	// APNS implementation would go here
	log.Printf("Would send APNS notification to: %s", notif.DeviceToken)
	return nil
}

// Send batch
func (s *NotificationService) sendBatchNotifications(c *gin.Context) {
	var req struct {
		Notifications []Notification `json:"notifications"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	results := make([]string, len(req.Notifications))
	for i, notif := range req.Notifications {
		notif.ID = uuid.New().String()
		notif.CreatedAt = time.Now()
		
		if notif.DeviceType == DeviceAndroid && s.fcmClient != nil {
			s.sendFCM(c.Request.Context(), &notif)
		}
		
		results[i] = notif.ID
	}
	
	c.JSON(http.StatusCreated, gin.H{"ids": results})
}

// Get notification
func (s *NotificationService) getNotification(c *gin.Context) {
	id := c.Param("id")
	
	key := fmt.Sprintf("notification:%s", id)
	data, err := s.redis.Get(c.Request.Context(), key).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	
	var notif Notification
	json.Unmarshal(data, &notif)
	
	c.JSON(http.StatusOK, notif)
}

// List notifications
func (s *NotificationService) listNotifications(c *gin.Context) {
	userID := c.Query("user_id")
	limit := c.DefaultQuery("limit", "50")
	
	userNotifsKey := fmt.Sprintf("user:%s:notifications", userID)
	notifIDs, err := s.redis.SMembers(c.Request.Context(), userNotifsKey).Result()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"notifications": []})
		return
	}
	
	notifications := make([]Notification, 0, len(notifIDs))
	for _, id := range notifIDs {
		key := fmt.Sprintf("notification:%s", id)
		data, _ := s.redis.Get(c.Request.Context(), key).Bytes()
		var notif Notification
		json.Unmarshal(data, &notif)
		notifications = append(notifications, notif)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"total":         len(notifications),
	})
}

// Price alerts
func (s *NotificationService) createPriceAlert(c *gin.Context) {
	var alert PriceAlert
	if err := c.ShouldBindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	alert.ID = uuid.New().String()
	alert.CreatedAt = time.Now()
	alert.IsActive = true
	
	// Store
	key := fmt.Sprintf("price_alert:%s", alert.ID)
	data, _ := json.Marshal(alert)
	s.redis.Set(c.Request.Context(), key, data, 0)
	
	// Add to user's alerts
	userAlertsKey := fmt.Sprintf("user:%s:price_alerts", alert.UserID)
	s.redis.SAdd(c.Request.Context(), userAlertsKey, alert.ID)
	
	c.JSON(http.StatusCreated, alert)
}

func (s *NotificationService) listPriceAlerts(c *gin.Context) {
	userID := c.Query("user_id")
	
	userAlertsKey := fmt.Sprintf("user:%s:price_alerts", userID)
	alertIDs, err := s.redis.SMembers(c.Request.Context(), userAlertsKey).Result()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"alerts": []})
		return
	}
	
	alerts := make([]PriceAlert, 0, len(alertIDs))
	for _, id := range alertIDs {
		key := fmt.Sprintf("price_alert:%s", id)
		data, _ := s.redis.Get(c.Request.Context(), key).Bytes()
		var alert PriceAlert
		json.Unmarshal(data, &alert)
		alerts = append(alerts, alert)
	}
	
	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func (s *NotificationService) updatePriceAlert(c *gin.Context) {
	id := c.Param("id")
	
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	key := fmt.Sprintf("price_alert:%s", id)
	data, err := s.redis.Get(c.Request.Context(), key).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}
	
	var alert PriceAlert
	json.Unmarshal(data, &alert)
	
	if active, ok := updates["is_active"].(bool); ok {
		alert.IsActive = active
	}
	if price, ok := updates["target_price"].(float64); ok {
		alert.TargetPrice = price
	}
	
	data, _ = json.Marshal(alert)
	s.redis.Set(c.Request.Context(), key, data, 0)
	
	c.JSON(http.StatusOK, alert)
}

func (s *NotificationService) deletePriceAlert(c *gin.Context) {
	id := c.Param("id")
	
	key := fmt.Sprintf("price_alert:%s", id)
	s.redis.Del(c.Request.Context(), key)
	
	c.JSON(http.StatusOK, gin.H{"message": "Alert deleted"})
}

// Subscribe
func (s *NotificationService) subscribe(c *gin.Context) {
	var req struct {
		UserID    string   `json:"user_id"`
		Topics    []string `json:"topics"`
		Channel   string   `json:"channel"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	for _, topic := range req.Topics {
		topicKey := fmt.Sprintf("topic:%s:subscribers", topic)
		s.redis.SAdd(c.Request.Context(), topicKey, req.UserID)
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Subscribed successfully"})
}

// Unsubscribe
func (s *NotificationService) unsubscribe(c *gin.Context) {
	var req struct {
		UserID  string   `json:"user_id"`
		Topics  []string `json:"topics"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	for _, topic := range req.Topics {
		topicKey := fmt.Sprintf("topic:%s:subscribers", topic)
		s.redis.SRem(c.Request.Context(), topicKey, req.UserID)
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribed successfully"})
}

// Price alert worker
func (s *NotificationService) startPriceAlertWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	ctx := context.Background()
	
	for range ticker.C {
		// Get all active alerts
		keys, err := s.redis.Keys(ctx, "price_alert:*").Result()
		if err != nil {
			continue
		}
		
		for _, key := range keys {
			data, err := s.redis.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}
			
			var alert PriceAlert
			if json.Unmarshal(data, &alert); err != nil || !alert.IsActive {
				continue
			}
			
			// Check price (would fetch from price service in production)
			currentPrice := getMockPrice(alert.TokenSymbol)
			
			shouldTrigger := (alert.Condition == "above" && currentPrice >= alert.TargetPrice) ||
				(alert.Condition == "below" && currentPrice <= alert.TargetPrice)
			
			if shouldTrigger {
				now := time.Now()
				alert.TriggeredAt = &now
				alert.IsActive = false
				
				// Update alert
				data, _ = json.Marshal(alert)
				s.redis.Set(ctx, key, data, 0)
				
				// Send notification
				notif := &Notification{
					ID:          uuid.New().String(),
					Type:        TypePriceAlert,
					Title:       fmt.Sprintf("%s Price Alert", alert.TokenSymbol),
					Body:        fmt.Sprintf("%s is now $%.2f (%s $%.2f)", alert.TokenSymbol, currentPrice, alert.Condition, alert.TargetPrice),
					Priority:    PriorityHigh,
					DeviceToken: alert.UserID, // Would lookup device token
					DeviceType:  DeviceAndroid,
					CreatedAt:   time.Now(),
				}
				
				if s.fcmClient != nil {
					s.sendFCM(ctx, notif)
				}
			}
		}
	}
}

// Cleanup worker
func (s *NotificationService) startCleanupWorker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	ctx := context.Background()
	
	for range ticker.C {
		// Clean up old notifications
		keys, _ := s.redis.Keys(ctx, "notification:*").Result()
		for _, key := range keys {
			ttl, _ := s.redis.TTL(ctx, key).Result()
			if ttl < 0 {
				s.redis.Del(ctx, key)
			}
		}
	}
}

// Health check
func (s *NotificationService) healthCheck(c *gin.Context) {
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	}
	
	// Check Redis
	if err := s.redis.Ping(c.Request.Context()).Err(); err != nil {
		health["redis"] = "unhealthy"
	} else {
		health["redis"] = "healthy"
	}
	
	// Check FCM
	if s.fcmClient != nil {
		health["fcm"] = "healthy"
	} else {
		health["fcm"] = "not_configured"
	}
	
	c.JSON(http.StatusOK, health)
}

// Mock price function
func getMockPrice(symbol string) float64 {
	prices := map[string]float64{
		"BTC":  65000.0,
		"ETH":  3500.0,
		"SOL":  145.0,
		"BNB":  580.0,
		"XRP":  0.55,
		"ADA":  0.45,
		"DOGE": 0.12,
		"AVAX": 35.0,
	}
	
	if price, ok := prices[symbol]; ok {
		return price
	}
	return 0.0
}

func main() {
	config := &Config{
		Port:           os.Getenv("PORT"),
		RedisURL:       os.Getenv("REDIS_URL"),
		FCMCredentials: os.Getenv("FCM_CREDENTIALS"),
		FCMProjectID:   os.Getenv("FCM_PROJECT_ID"),
	}
	
	if config.Port == "" {
		config.Port = "8080"
	}
	if config.RedisURL == "" {
		config.RedisURL = "redis://localhost:6379"
	}
	
	service, err := New(config)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}
	
	if err := service.Start(); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}
	
	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down notification service...")
	
	if err := service.Stop(); err != nil {
		log.Printf("Error stopping service: %v", err)
	}
}
