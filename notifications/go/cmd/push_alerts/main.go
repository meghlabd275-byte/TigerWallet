package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// ============ Configuration ============

type Config struct {
	Port              string
	FCMCredentials   string
	APNSKeyPath      string
	APNSKeyID        string
	APNSTeamID       string
	RedisURL          string
}

// ============ Models ============

type NotificationType string

const (
	TypePriceAlert      NotificationType = "price_alert"
	TypeTransaction    NotificationType = "transaction"
	TypeAirdrop        NotificationType = "airdrop"
	TypeStakingReward  NotificationType = "staking_reward"
	TypeGovernance     NotificationType = "governance"
	TypeSecurity       NotificationType = "security"
	TypeMarketing      NotificationType = "marketing"
)

type NotificationPriority string

const (
	PriorityHigh   NotificationPriority = "high"
	PriorityNormal NotificationPriority = "normal"
	PriorityLow    NotificationPriority = "low"
)

type Notification struct {
	ID          string             `json:"id"`
	UserID      string             `json:"userId"`
	Type        NotificationType  `json:"type"`
	Title       string             `json:"title"`
	Body        string             `json:"body"`
	Data        map[string]string  `json:"data,omitempty"`
	Priority    NotificationPriority `json:"priority"`
	ExpiresAt   *time.Time         `json:"expiresAt,omitempty"`
	CreatedAt   time.Time          `json:"createdAt"`
	ReadAt      *time.Time         `json:"readAt,omitempty"`
}

type Device struct {
	ID           string   `json:"id"`
	UserID       string   `json:"userId"`
	Token        string   `json:"token"`
	Platform     string   `json:"platform"` // ios, android, web
	AppVersion   string   `json:"appVersion"`
	Language     string   `json:"language"`
	Timezone     string   `json:"timezone"`
	LastActiveAt time.Time `json:"lastActiveAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Subscription struct {
	UserID     string             `json:"userId"`
	Types      []NotificationType `json:"types"`
	Devices    []string           `json:"devices"`
	MinAmount  float64            `json:"minAmount,omitempty"` // For price alerts
	Enabled    bool               `json:"enabled"`
}

// ============ Push Service ============

type PushService struct {
	config      *Config
	firebaseApp *firebase.App
	messaging   *messaging.Client
	redis       *redis.Client
}

func NewPushService(config *Config) (*PushService, error) {
	ctx := context.Background()

	// Initialize Firebase
	var firebaseApp *firebase.App
	var messagingClient *messaging.Client

	if config.FCMCredentials != "" {
		opt := option.WithCredentialsJSON([]byte(config.FCMCredentials))
		app, err := firebase.NewApp(ctx, nil, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Firebase: %w", err)
		}
		firebaseApp = app

		client, err := app.Messaging(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get messaging client: %w", err)
		}
		messagingClient = client
	}

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: "",
		DB:       0,
	})

	return &PushService{
		config:    config,
		messaging: messagingClient,
		firebaseApp: firebaseApp,
		redis:     redisClient,
	}, nil
}

// Send Notification
func (s *PushService) SendNotification(ctx context.Context, notification Notification, deviceTokens []string) ([]string, error) {
	if s.messaging == nil {
		return nil, fmt.Errorf("messaging not configured")
	}

	var failedTokens []string

	for _, token := range deviceTokens {
		msg := &messaging.Message{
			Token: token,
			Notification: &messaging.Notification{
				Title: notification.Title,
				Body:  notification.Body,
			},
			Data: notification.Data,
			Android: &messaging.AndroidConfig{
				Priority: s.getAndroidPriority(notification.Priority),
				Notification: &messaging.AndroidNotification{
					Title: notification.Title,
					Body:  notification.Body,
				},
			},
			APNS: &messaging.APNSConfig{
				Headers: map[string]string{
					"apns-priority": s.getAPNSPriority(notification.Priority),
				},
				Payload: &messaging.APNSPayload{
					Aps: &messaging.Aps{
						Alert: &messaging.ApsAlert{
							Title: notification.Title,
							Body:  notification.Body,
						},
					},
				},
			},
		}

		_, err := s.messaging.Send(ctx, msg)
		if err != nil {
			log.Printf("Failed to send to token %s: %v", token, err)
			failedTokens = append(failedTokens, token)
		}
	}

	return failedTokens, nil
}

// Send to User (all devices)
func (s *PushService) SendToUser(ctx context.Context, userID string, notification Notification) error {
	tokens, err := s.GetDeviceTokens(ctx, userID)
	if err != nil {
		return err
	}

	if len(tokens) == 0 {
		return fmt.Errorf("no devices registered for user")
	}

	notification.UserID = userID
	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}

	// Store notification
	if s.redis != nil {
		data, _ := json.Marshal(notification)
		s.redis.Set(ctx, fmt.Sprintf("notification:%s", notification.ID), data, 30*24*time.Hour)

		// Add to user's notification list
		s.redis.LPush(ctx, fmt.Sprintf("user:%s:notifications", userID), notification.ID)
	}

	// Send
	_, err = s.SendNotification(ctx, notification, tokens)
	return err
}

// Register Device
func (s *PushService) RegisterDevice(ctx context.Context, device Device) error {
	device.ID = uuid.New().String()
	device.CreatedAt = time.Now()
	device.LastActiveAt = time.Now()

	if s.redis != nil {
		data, _ := json.Marshal(device)
		s.redis.Set(ctx, fmt.Sprintf("device:%s", device.ID), data, 365*24*time.Hour)
		s.redis.SAdd(ctx, fmt.Sprintf("user:%s:devices", device.UserID), device.ID)
	}

	// Subscribe to FCM topic for this user
	if s.messaging != nil {
		_, err := s.messaging.SubscribeToTopic(ctx, []string{device.Token}, fmt.Sprintf("user_%s", device.UserID))
		return err
	}

	return nil
}

// Get Device Tokens
func (s *PushService) GetDeviceTokens(ctx context.Context, userID string) ([]string, error) {
	if s.redis == nil {
		return []string{}, nil
	}

	deviceIDs, err := s.redis.SMembers(ctx, fmt.Sprintf("user:%s:devices", userID)).Result()
	if err != nil {
		return nil, err
	}

	var tokens []string
	for _, id := range deviceIDs {
		data, err := s.redis.Get(ctx, fmt.Sprintf("device:%s", id)).Bytes()
		if err != nil {
			continue
		}

		var device Device
		json.Unmarshal(data, &device)
		tokens = append(tokens, device.Token)
	}

	return tokens, nil
}

// Create Subscription
func (s *PushService) CreateSubscription(ctx context.Context, sub Subscription) error {
	if s.redis != nil {
		data, _ := json.Marshal(sub)
		s.redis.Set(ctx, fmt.Sprintf("subscription:%s", sub.UserID), data, 365*24*time.Hour)
	}
	return nil
}

// Get Subscription
func (s *PushService) GetSubscription(ctx context.Context, userID string) (*Subscription, error) {
	if s.redis == nil {
		return &Subscription{UserID: userID, Enabled: true}, nil
	}

	data, err := s.redis.Get(ctx, fmt.Sprintf("subscription:%s", userID)).Bytes()
	if err != nil {
		return &Subscription{UserID: userID, Enabled: true}, nil
	}

	var sub Subscription
	json.Unmarshal(data, &sub)
	return &sub, nil
}

// Send Price Alert
func (s *PushService) SendPriceAlert(ctx context.Context, userID, symbol, direction string, price, change float64) error {
	title := fmt.Sprintf("%s Price Alert", strings.ToUpper(symbol))
	body := fmt.Sprintf("%s is now $%.2f (%s%.2f%%)", 
		strings.ToUpper(symbol), 
		price, 
		"+", 
		change,
	)

	notification := Notification{
		Type:     TypePriceAlert,
		Title:    title,
		Body:     body,
		Priority: PriorityHigh,
		Data: map[string]string{
			"symbol":    symbol,
			"price":     fmt.Sprintf("%.2f", price),
			"change":    fmt.Sprintf("%.2f", change),
			"direction": direction,
		},
	}

	return s.SendToUser(ctx, userID, notification)
}

// Send Transaction Notification
func (s *PushService) SendTransactionNotification(
	ctx context.Context,
	userID, txType, symbol string,
	amount, value float64,
	hash string,
) error {
	title := fmt.Sprintf("Transaction %s", txType)
	body := fmt.Sprintf("%s %s (~$%.2f)", txType, fmt.Sprintf("%.4f %s", amount, symbol), value)

	notification := Notification{
		Type:     TypeTransaction,
		Title:    title,
		Body:     body,
		Priority: PriorityHigh,
		Data: map[string]string{
			"type":      txType,
			"symbol":    symbol,
			"amount":    fmt.Sprintf("%.8f", amount),
			"value":     fmt.Sprintf("%.2f", value),
			"hash":      hash,
		},
	}

	return s.SendToUser(ctx, userID, notification)
}

func (s *PushService) getAndroidPriority(priority NotificationPriority) string {
	switch priority {
	case PriorityHigh:
		return "high"
	default:
		return "normal"
	}
}

func (s *PushService) getAPNSPriority(priority NotificationPriority) string {
	switch priority {
	case PriorityHigh:
		return "10"
	case PriorityLow:
		return "5"
	default:
		return "normal"
	}
}

// ============ HTTP Handlers ============

type Handler struct {
	pushService *PushService
}

func NewHandler(pushService *PushService) *Handler {
	return &Handler{pushService: pushService}
}

func (h *Handler) RegisterDevice(c *gin.Context) {
	var device Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.pushService.RegisterDevice(c.Request.Context(), device)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "registered"})
}

func (h *Handler) SendNotification(c *gin.Context) {
	var notification Notification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.Param("userId")

	err := h.pushService.SendToUser(c.Request.Context(), userID, notification)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}

func (h *Handler) Subscribe(c *gin.Context) {
	var sub Subscription
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.pushService.CreateSubscription(c.Request.Context(), sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "subscribed"})
}

func (h *Handler) GetNotifications(c *gin.Context) {
	userID := c.Param("userId")

	// Get notification IDs
	if h.pushService.redis == nil {
		c.JSON(http.StatusOK, gin.H{"notifications": []Notification{}})
		return
	}

	ids, err := h.pushService.redis.LRange(c.Request.Context(), 
		fmt.Sprintf("user:%s:notifications", userID), 0, 50).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var notifications []Notification
	for _, id := range ids {
		data, _ := h.pushService.redis.Get(c.Request.Context(), 
			fmt.Sprintf("notification:%s", id)).Bytes()
		
		var n Notification
		json.Unmarshal(data, &n)
		notifications = append(notifications, n)
	}

	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

func (h *Handler) PriceAlert(c *gin.Context) {
	var req struct {
		UserID    string  `json:"userId" binding:"required"`
		Symbol    string  `json:"symbol" binding:"required"`
		Direction string  `json:"direction" binding:"required"`
		Price     float64 `json:"price" binding:"required"`
		Change    float64 `json:"change" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.pushService.SendPriceAlert(c.Request.Context(), 
		req.UserID, req.Symbol, req.Direction, req.Price, req.Change)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}

// ============ Main ============

func main() {
	config := &Config{
		Port:            getEnv("PORT", "8080"),
		FCMCredentials:  os.Getenv("FCM_CREDENTIALS"),
		APNSKeyPath:     os.Getenv("APNS_KEY_PATH"),
		APNSKeyID:       os.Getenv("APNS_KEY_ID"),
		APNSTeamID:      os.Getenv("APNS_TEAM_ID"),
		RedisURL:        getEnv("REDIS_URL", "localhost:6379"),
	}

	pushService, err := NewPushService(config)
	if err != nil {
		log.Fatalf("Failed to create push service: %v", err)
	}

	handler := NewHandler(pushService)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		api.POST("/devices", handler.RegisterDevice)
		api.POST("/users/:userId/notifications", handler.SendNotification)
		api.POST("/subscriptions", handler.Subscribe)
		api.GET("/users/:userId/notifications", handler.GetNotifications)
		api.POST("/alerts/price", handler.PriceAlert)
	}

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting Push Notification service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
