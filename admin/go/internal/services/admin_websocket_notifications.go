/**
 * TigerWallet Admin WebSocket Notification Service
 * Real-time Notifications for Admin Panel
 * High-Performance, Distributed, Ultra-Low Latency
 *
 * Features:
 * - Real-time WebSocket connections
 * - Push notifications to admins
 * - Event broadcasting
 * - Admin activity monitoring
 * - Alert notifications
 * - KYC/Withdrawal notifications
 */

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"nhooyr.io/websocket"
)

// ============================================================================
// Configuration
// ============================================================================

type NotificationConfig struct {
	Port             string
	RedisURL         string
	JWTSecret        string
	MaxConnections   int
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	PingInterval     time.Duration
	MessageQueueSize int
}

func LoadNotificationConfig() *NotificationConfig {
	return &NotificationConfig{
		Port:             getEnv("NOTIFICATION_PORT", "9096"),
		RedisURL:         getEnv("REDIS_NOTIFICATION_URL", "redis://localhost:6379"),
		JWTSecret:        getEnv("NOTIFICATION_JWT_SECRET", "notification-secret-key"),
		MaxConnections:   getEnvInt("MAX_CONNECTIONS", 10000),
		ReadTimeout:      getEnvDuration("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:     getEnvDuration("WRITE_TIMEOUT", 10*time.Second),
		PingInterval:     getEnvDuration("PING_INTERVAL", 30*time.Second),
		MessageQueueSize: getEnvInt("MESSAGE_QUEUE_SIZE", 1000),
	}
}

// ============================================================================
// Types
// ============================================================================

type NotificationType string

const (
	NotificationTypeKYC         NotificationType = "kyc"
	NotificationTypeWithdrawal  NotificationType = "withdrawal"
	NotificationTypeTransaction NotificationType = "transaction"
	NotificationTypeUser        NotificationType = "user"
	NotificationTypeSystem      NotificationType = "system"
	NotificationTypeSecurity    NotificationType = "security"
	NotificationTypeAlert       NotificationType = "alert"
	NotificationTypeAudit       NotificationType = "audit"
	NotificationTypeGeneral     NotificationType = "general"
)

type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "low"
	PriorityNormal   NotificationPriority = "normal"
	PriorityHigh     NotificationPriority = "high"
	PriorityCritical NotificationPriority = "critical"
)

type Notification struct {
	ID        string                 `json:"id"`
	Type      NotificationType       `json:"type"`
	Priority  NotificationPriority   `json:"priority"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	AdminID   string                 `json:"admin_id,omitempty"`
	Role      string                 `json:"role,omitempty"`
	Read      bool                   `json:"read"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
}

type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type WSClient struct {
	ID        string
	AdminID   string
	Role      string
	Conn      *websocket.Conn
	Send      chan []byte
	Hub       *NotificationHub
	CreatedAt time.Time
	LastPing  time.Time
	mu        sync.Mutex
}

type Subscription struct {
	AdminID   string
	Events    []NotificationType
	CreatedAt time.Time
}

// ============================================================================
// Notification Hub
// ============================================================================

type NotificationHub struct {
	clients       map[string]*WSClient
	subscriptions map[string][]Subscription
	broadcast     chan *Notification
	register      chan *WSClient
	unregister    chan *WSClient
	redisPub      *redis.Client
	redisSub      *redis.PubSub
	mu            sync.RWMutex
	config        *NotificationConfig
	stats         HubStats
}

type HubStats struct {
	TotalConnections  int64 `json:"total_connections"`
	ActiveConnections int64 `json:"active_connections"`
	MessagesSent      int64 `json:"messages_sent"`
	MessagesReceived  int64 `json:"messages_received"`
	Errors            int64 `json:"errors"`
	mu                sync.RWMutex
}

func NewNotificationHub(config *NotificationConfig, redisClient *redis.Client) *NotificationHub {
	hub := &NotificationHub{
		clients:       make(map[string]*WSClient),
		subscriptions: make(map[string][]Subscription),
		broadcast:     make(chan *Notification, config.MessageQueueSize),
		register:      make(chan *WSClient),
		unregister:    make(chan *WSClient),
		redisPub:      redisClient,
		config:        config,
		stats:         HubStats{},
	}

	return hub
}

func (h *NotificationHub) Run() {
	// Start message processor
	go h.processMessages()

	// Start Redis subscriber for distributed notifications
	go h.runRedisSubscriber()

	// Start statistics reporter
	go h.reportStats()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.stats.ActiveConnections++
			h.stats.TotalConnections++
			h.mu.Unlock()
			log.Printf("Client connected: %s (admin: %s, role: %s)", client.ID, client.AdminID, client.Role)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("Client disconnected: %s", client.ID)

		case notification := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				// Check if client should receive this notification
				if h.shouldReceive(client, notification) {
					select {
					case client.Send <- notificationToBytes(notification):
						h.stats.mu.Lock()
						h.stats.MessagesSent++
						h.stats.mu.Unlock()
					default:
						// Client buffer full, close connection
						close(client.Send)
						delete(h.clients, client.ID)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *NotificationHub) shouldReceive(client *WSClient, notification *Notification) bool {
	// Admin receives all notifications for their role
	if notification.Role != "" && notification.Role == client.Role {
		return true
	}

	// Admin receives notifications targeted specifically to them
	if notification.AdminID != "" && notification.AdminID == client.AdminID {
		return true
	}

	// System notifications go to everyone
	if notification.Type == NotificationTypeSystem {
		return true
	}

	// Check subscriptions
	h.mu.RLock()
	defer h.mu.RUnlock()

	if subs, ok := h.subscriptions[client.AdminID]; ok {
		for _, sub := range subs {
			for _, eventType := range sub.Events {
				if string(eventType) == string(notification.Type) {
					return true
				}
			}
		}
	}

	return false
}

func (h *NotificationHub) processMessages() {
	for {
		notification := <-h.broadcast

		// Store in Redis for persistence
		ctx := context.Background()
		notificationJSON, _ := json.Marshal(notification)

		// Store notification
		h.redisPub.Set(ctx, "notification:"+notification.ID, notificationJSON, 24*time.Hour)

		// Add to admin's notification list
		if notification.AdminID != "" {
			h.redisPub.LPush(ctx, "notifications:"+notification.AdminID, notification.ID)
			h.redisPub.LTrim(ctx, "notifications:"+notification.AdminID, 0, 99) // Keep last 100
		}

		// Broadcast to all connected clients
		h.mu.RLock()
		for _, client := range h.clients {
			if h.shouldReceive(client, notification) {
				select {
				case client.Send <- notificationToBytes(notification):
				default:
				}
			}
		}
		h.mu.RUnlock()
	}
}

func (h *NotificationHub) runRedisSubscriber() {
	ctx := context.Background()

	// Subscribe to notification channels
	pubsub := h.redisPub.Subscribe(ctx, "admin:notifications")
	defer pubsub.Close()

	for {
		select {
		case msg := <-pubsub.Channel():
			var notification Notification
			if err := json.Unmarshal([]byte(msg.Payload), &notification); err == nil {
				h.broadcast <- &notification
			}
		case <-time.After(30 * time.Second):
			// Ping Redis to keep connection alive
			h.redisPub.Ping(ctx)
		}
	}
}

func (h *NotificationHub) reportStats() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.stats.mu.RLock()
		log.Printf("Notification Hub Stats - Active: %d, Total: %d, Sent: %d, Received: %d, Errors: %d",
			h.stats.ActiveConnections,
			h.stats.TotalConnections,
			h.stats.MessagesSent,
			h.stats.MessagesReceived,
			h.stats.Errors,
		)
		h.stats.mu.RUnlock()
	}
}

func (h *NotificationHub) Subscribe(adminID string, events []NotificationType) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.subscriptions[adminID] = append(h.subscriptions[adminID], Subscription{
		AdminID:   adminID,
		Events:    events,
		CreatedAt: time.Now(),
	})
}

func (h *NotificationHub) Unsubscribe(adminID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.subscriptions, adminID)
}

func (h *NotificationHub) SendToAdmin(adminID string, notification *Notification) {
	h.broadcast <- notification

	// Also publish to Redis for persistence
	ctx := context.Background()
	notificationJSON, _ := json.Marshal(notification)
	h.redisPub.Publish(ctx, "admin:notifications", notificationJSON)
}

func (h *NotificationHub) BroadcastToRole(role string, notification *Notification) {
	notification.Role = role
	h.broadcast <- notification
}

func (h *NotificationHub) Broadcast(notification *Notification) {
	h.broadcast <- notification
}

func notificationToBytes(notification *Notification) []byte {
	data, _ := json.Marshal(notification)
	return data
}

// ============================================================================
// HTTP Handlers
// ============================================================================

type NotificationHandler struct {
	hub    *NotificationHub
	config *NotificationConfig
	redis  *redis.Client
}

func NewNotificationHandler(hub *NotificationHub, config *NotificationConfig, redisClient *redis.Client) *NotificationHandler {
	return &NotificationHandler{
		hub:    hub,
		config: config,
		redis:  redisClient,
	}
}

func (h *NotificationHandler) HandleWebSocket(c *gin.Context) {
	adminID := c.Query("admin_id")
	role := c.Query("role")

	if adminID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin_id required"})
		return
	}

	// Upgrade to WebSocket
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionContextTakeover,
	})
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &WSClient{
		ID:        uuid.New().String(),
		AdminID:   adminID,
		Role:      role,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		Hub:       h.hub,
		CreatedAt: time.Now(),
		LastPing:  time.Now(),
	}

	h.hub.register <- client

	// Handle connection
	go h.handleConnection(client)
}

func (h *NotificationHandler) handleConnection(client *WSClient) {
	defer func() {
		h.hub.unregister <- client
		client.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := context.Background()

	// Start ping/pong loop
	pingChan := make(chan struct{})
	go func() {
		ticker := time.NewTicker(h.config.PingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := client.Conn.Ping(ctx)
				if err != nil {
					return
				}
				client.LastPing = time.Now()
			case <-pingChan:
				return
			}
		}
	}()

	// Read loop
	for {
		_, message, err := client.Conn.Read(ctx)
		if err != nil {
			break
		}

		h.handleMessage(client, message)
	}

	close(pingChan)
}

func (h *NotificationHandler) handleMessage(client *WSClient, message []byte) {
	h.hub.stats.mu.Lock()
	h.hub.stats.MessagesReceived++
	h.hub.stats.mu.Unlock()

	var wsMsg WSMessage
	if err := json.Unmarshal(message, &wsMsg); err != nil {
		return
	}

	switch wsMsg.Type {
	case "subscribe":
		var payload struct {
			Events []NotificationType `json:"events"`
		}
		if json.Unmarshal(wsMsg.Payload, &payload) == nil {
			h.hub.Subscribe(client.AdminID, payload.Events)
		}

	case "unsubscribe":
		h.hub.Unsubscribe(client.AdminID)

	case "ping":
		pong := map[string]interface{}{
			"type":      "pong",
			"timestamp": time.Now().Unix(),
		}
		pongData, _ := json.Marshal(pong)
		client.Send <- pongData

	case "mark_read":
		var payload struct {
			NotificationID string `json:"notification_id"`
		}
		if json.Unmarshal(wsMsg.Payload, &payload) == nil {
			h.markNotificationRead(client.AdminID, payload.NotificationID)
		}

	case "mark_all_read":
		h.markAllNotificationsRead(client.AdminID)
	}
}

func (h *NotificationHandler) markNotificationRead(adminID, notificationID string) {
	ctx := context.Background()

	// Update in Redis
	notificationKey := "notification:" + notificationID
	notificationJSON, err := h.redis.Get(ctx, notificationKey).Result()
	if err != nil {
		return
	}

	var notification Notification
	if json.Unmarshal([]byte(notificationJSON), &notification) == nil {
		notification.Read = true
		updatedJSON, _ := json.Marshal(notification)
		h.redis.Set(ctx, notificationKey, updatedJSON, 24*time.Hour)
	}
}

func (h *NotificationHandler) markAllNotificationsRead(adminID string) {
	ctx := context.Background()

	// Get all notification IDs
	notificationIDs, err := h.redis.LRange(ctx, "notifications:"+adminID, 0, -1).Result()
	if err != nil {
		return
	}

	for _, id := range notificationIDs {
		h.markNotificationRead(adminID, id)
	}
}

func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	adminID := c.Query("admin_id")
	if adminID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin_id required"})
		return
	}

	ctx := context.Background()

	// Get notification IDs
	notificationIDs, err := h.redis.LRange(ctx, "notifications:"+adminID, 0, 99).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	// Get notifications
	notifications := make([]Notification, 0, len(notificationIDs))
	for _, id := range notificationIDs {
		notificationJSON, err := h.redis.Get(ctx, "notification:"+id).Result()
		if err != nil {
			continue
		}

		var notification Notification
		if json.Unmarshal([]byte(notificationJSON), &notification) == nil {
			notifications = append(notifications, notification)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"total":         len(notifications),
	})
}

func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	var notification Notification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notification.ID = uuid.New().String()
	notification.CreatedAt = time.Now()

	// Send notification
	h.hub.Broadcast(&notification)

	c.JSON(http.StatusCreated, gin.H{
		"id":      notification.ID,
		"message": "Notification sent",
	})
}

func (h *NotificationHandler) GetStats(c *gin.Context) {
	h.hub.stats.mu.RLock()
	defer h.hub.stats.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"total_connections":  h.hub.stats.TotalConnections,
		"active_connections": h.hub.stats.ActiveConnections,
		"messages_sent":      h.hub.stats.MessagesSent,
		"messages_received":  h.hub.stats.MessagesReceived,
		"errors":             h.hub.stats.Errors,
	})
}

// ============================================================================
// Notification Service (for internal use)
// ============================================================================

type WSNotificationService struct {
	hub   *NotificationHub
	redis *redis.Client
}

func NewWSWSNotificationService(hub *NotificationHub, redisClient *redis.Client) *WSNotificationService {
	return &WSNotificationService{
		hub:   hub,
		redis: redisClient,
	}
}

func (s *WSNotificationService) NotifyKYC(userID, status, adminID string) {
	notification := &Notification{
		ID:       uuid.New().String(),
		Type:     NotificationTypeKYC,
		Priority: PriorityHigh,
		Title:    "KYC " + status,
		Message:  fmt.Sprintf("KYC request for user %s has been %s", userID, status),
		Data: map[string]interface{}{
			"user_id": userID,
			"status":  status,
		},
		AdminID:   adminID,
		CreatedAt: time.Now(),
	}

	s.hub.Broadcast(notification)
}

func (s *WSNotificationService) NotifyWithdrawal(withdrawalID, status, adminID string) {
	notification := &Notification{
		ID:       uuid.New().String(),
		Type:     NotificationTypeWithdrawal,
		Priority: PriorityCritical,
		Title:    "Withdrawal " + status,
		Message:  fmt.Sprintf("Withdrawal %s has been %s", withdrawalID, status),
		Data: map[string]interface{}{
			"withdrawal_id": withdrawalID,
			"status":        status,
		},
		AdminID:   adminID,
		CreatedAt: time.Now(),
	}

	s.hub.Broadcast(notification)
}

func (s *WSNotificationService) NotifyTransaction(transactionID, status string) {
	notification := &Notification{
		ID:       uuid.New().String(),
		Type:     NotificationTypeTransaction,
		Priority: PriorityNormal,
		Title:    "Transaction Alert",
		Message:  fmt.Sprintf("Transaction %s: %s", transactionID, status),
		Data: map[string]interface{}{
			"transaction_id": transactionID,
			"status":         status,
		},
		CreatedAt: time.Now(),
	}

	s.hub.Broadcast(notification)
}

func (s *WSNotificationService) NotifySecurity(event, details string) {
	notification := &Notification{
		ID:       uuid.New().String(),
		Type:     NotificationTypeSecurity,
		Priority: PriorityCritical,
		Title:    "Security Alert",
		Message:  details,
		Data: map[string]interface{}{
			"event": event,
		},
		CreatedAt: time.Now(),
	}

	s.hub.Broadcast(notification)
}

func (s *WSNotificationService) NotifySystem(message string) {
	notification := &Notification{
		ID:        uuid.New().String(),
		Type:      NotificationTypeSystem,
		Priority:  PriorityNormal,
		Title:     "System Notification",
		Message:   message,
		CreatedAt: time.Now(),
	}

	s.hub.Broadcast(notification)
}
