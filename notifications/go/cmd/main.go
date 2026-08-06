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
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := loadConfig()

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	// Initialize queue
	queue := NewNotificationQueue(rdb)

	// Start workers
	for i := 0; i cfg.WorkerCount; i++ {
		go queue.Worker(i, cfg)
	}

	// Initialize Gin router
	router := gin.Default()

	// CORS
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-notifications"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Notifications
		api.POST("/notifications", createNotificationHandler(db, queue))
		api.GET("/notifications/:user_id", getNotificationsHandler(db))
		api.PUT("/notifications/:id/read", markAsReadHandler(db))
		api.PUT("/notifications/:id/read-all", markAllAsReadHandler(db))
		api.DELETE("/notifications/:id", deleteNotificationHandler(db))

		// Templates
		api.GET("/templates", getTemplatesHandler(db))
		api.POST("/templates", createTemplateHandler(db))
		api.PUT("/templates/:id", updateTemplateHandler(db))
		api.DELETE("/templates/:id", deleteTemplateHandler(db))

		// Preferences
		api.GET("/preferences/:user_id", getPreferencesHandler(db))
		api.PUT("/preferences/:user_id", updatePreferencesHandler(db))

		// Channels
		api.POST("/channels/email", sendEmailHandler(queue))
		api.POST("/channels/sms", sendSMSHandler(queue))
		api.POST("/channels/push", sendPushHandler(queue))
		api.POST("/channels/webhook", sendWebhookHandler(queue))

		// Broadcast
		api.POST("/broadcast", broadcastHandler(queue, db))
	}

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	go func() {
		log.Printf("Notification service starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}

// ============== Configuration ==============

type Config struct {
	Port        string
	WorkerCount int
	Database    DatabaseConfig
	Redis       RedisConfig
	SMTP        SMTPConfig
	Twilio      TwilioConfig
	FCM         FCMConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type TwilioConfig struct {
	AccountSid  string
	AuthToken   string
	PhoneNumber string
}

type FCMConfig struct {
	ProjectID string
	PrivateKey string
}

func loadConfig() *Config {
	return &Config{
		Port:        getEnv("NOTIFICATION_PORT", "9004"),
		WorkerCount: getEnvInt("NOTIFICATION_WORKERS", 5),
		Database: DatabaseConfig{
			Host:     getEnv("NOTIFICATION_DB_HOST", "localhost"),
			Port:     getEnvInt("NOTIFICATION_DB_PORT", 5432),
			User:     getEnv("NOTIFICATION_DB_USER", "tigerwallet"),
			Password: getEnv("NOTIFICATION_DB_PASSWORD", "password"),
			DBName:   getEnv("NOTIFICATION_DB_NAME", "tigerwallet_notifications"),
		},
		Redis: RedisConfig{
			Host:     getEnv("NOTIFICATION_REDIS_HOST", "localhost"),
			Port:     getEnvInt("NOTIFICATION_REDIS_PORT", 6379),
			Password: getEnv("NOTIFICATION_REDIS_PASSWORD", ""),
			DB:      getEnvInt("NOTIFICATION_REDIS_DB", 0),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port:     getEnvInt("SMTP_PORT", 587),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@tigerwallet.com"),
		},
		Twilio: TwilioConfig{
			AccountSid:  getEnv("TWILIO_ACCOUNT_SID", ""),
			AuthToken:   getEnv("TWILIO_AUTH_TOKEN", ""),
			PhoneNumber: getEnv("TWILIO_PHONE_NUMBER", ""),
		},
		FCM: FCMConfig{
			ProjectID:  getEnv("FCM_PROJECT_ID", ""),
			PrivateKey: getEnv("FCM_PRIVATE_KEY", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscan(value, &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// ============== Models ==============

type Notification struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Type        string    `json:"type" db:"type"`
	Channel     string    `json:"channel" db:"channel"`
	Title       string    `json:"title" db:"title"`
	Message     string    `json:"message" db:"message"`
	Data        map[string]interface{} `json:"data" db:"data"`
	IsRead      bool      `json:"is_read" db:"is_read"`
	ReadAt      *time.Time `json:"read_at" db:"read_at"`
	ScheduledAt *time.Time `json:"scheduled_at" db:"scheduled_at"`
	SentAt      *time.Time `json:"sent_at" db:"sent_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type NotificationTemplate struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Type      string    `json:"type" db:"type"`
	Channel   string    `json:"channel" db:"channel"`
	Subject   string    `json:"subject" db:"subject"`
	Body      string    `json:"body" db:"body"`
	Variables []string `json:"variables" db:"variables"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type NotificationPreference struct {
	ID                uuid.UUID `json:"id" db:"id"`
	UserID            uuid.UUID `json:"user_id" db:"user_id"`
	EmailEnabled     bool      `json:"email_enabled" db:"email_enabled"`
	SMSEnabled       bool      `json:"sms_enabled" db:"sms_enabled"`
	PushEnabled      bool      `json:"push_enabled" db:"push_enabled"`
	TransactionAlerts bool      `json:"transaction_alerts" db:"transaction_alerts"`
	MarketingAlerts  bool      `json:"marketing_alerts" db:"marketing_alerts"`
	SecurityAlerts   bool      `json:"security_alerts" db:"security_alerts"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// ============== Notification Queue ==============

type NotificationQueue struct {
	redis *redis.Client
}

func NewNotificationQueue(rdb *redis.Client) *NotificationQueue {
	return &NotificationQueue{redis: rdb}
}

type QueuedNotification struct {
	ID        uuid.UUID              `json:"id"`
	UserID   uuid.UUID              `json:"user_id"`
	Type     string                 `json:"type"`
	Channel  string                 `json:"channel"`
	Title    string                 `json:"title"`
	Message  string                 `json:"message"`
	Data     map[string]interface{} `json:"data"`
	SentAt   *time.Time            `json:"sent_at"`
}

func (q *NotificationQueue) Enqueue(ctx context.Context, notification *QueuedNotification) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	return q.redis.LPush(ctx, "notifications:queue", data).Err()
}

func (q *NotificationQueue) Dequeue(ctx context.Context) (*QueuedNotification, error) {
	result, err := q.redis.BRPop(ctx, 5*time.Second, "notifications:queue").Result()
	if err != nil {
		return nil, err
	}

	var notification QueuedNotification
	if err := json.Unmarshal([]byte(result[1]), &notification); err != nil {
		return nil, err
	}
	return &notification, nil
}

func (q *NotificationQueue) Worker(id int, cfg *Config) {
	ctx := context.Background()
	log.Printf("Worker %d started", id)

	for {
		notification, err := q.Dequeue(ctx)
		if err != nil {
			continue
		}

		switch notification.Channel {
		case "email":
			q.sendEmail(cfg, notification)
		case "sms":
			q.sendSMS(cfg, notification)
		case "push":
			q.sendPush(cfg, notification)
		case "webhook":
			q.sendWebhook(cfg, notification)
		}

		log.Printf("Worker %d processed notification %s", id, notification.ID)
	}
}

func (q *NotificationQueue) sendEmail(cfg *Config, n *QueuedNotification) {
	// In production, implement actual email sending
	log.Printf("Sending email to user %s: %s", n.UserID, n.Title)
}

func (q *NotificationQueue) sendSMS(cfg *Config, n *QueuedNotification) {
	// In production, implement Twilio SMS
	log.Printf("Sending SMS to user %s: %s", n.UserID, n.Message)
}

func (q *NotificationQueue) sendPush(cfg *Config, n *QueuedNotification) {
	// In production, implement FCM push
	log.Printf("Sending push to user %s: %s", n.UserID, n.Title)
}

func (q *NotificationQueue) sendWebhook(cfg *Config, n *QueuedNotification) {
	// In production, implement webhook
	log.Printf("Sending webhook for notification %s", n.ID)
}

// ============== HTTP Handlers ==============

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func createNotificationHandler(db interface{}, queue *NotificationQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UserID      uuid.UUID              `json:"user_id" binding:"required"`
			Type        string                 `json:"type" binding:"required"`
			Channel     string                 `json:"channel" binding:"required"`
			Title       string                 `json:"title" binding:"required"`
			Message     string                 `json:"message" binding:"required"`
			Data        map[string]interface{} `json:"data"`
			ScheduledAt *time.Time            `json:"scheduled_at"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		notification := &QueuedNotification{
			ID:       uuid.New(),
			UserID:   req.UserID,
			Type:     req.Type,
			Channel:  req.Channel,
			Title:    req.Title,
			Message:  req.Message,
			Data:     req.Data,
			SentAt:   req.ScheduledAt,
		}

		if err := queue.Enqueue(c.Request.Context(), notification); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue notification"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": notification.ID, "status": "queued"})
	}
}

func getNotificationsHandler(db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		uid, err := uuid.Parse(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}

		// Return mock data
		notifications := []map[string]interface{}{
			{"id": uuid.New().String(), "type": "transaction", "title": "Transaction Confirmed", "message": "Your transaction has been confirmed", "is_read": false, "created_at": time.Now()},
			{"id": uuid.New().String(), "type": "security", "title": "New Login", "message": "New login detected", "is_read": true, "created_at": time.Now().Add(-1 * time.Hour)},
		}

		c.JSON(http.StatusOK, gin.H{"notifications": notifications, "unread_count": 1})
	}
}

func markAsReadHandler(db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "marked as read"})
	}
}

func markAllAsReadHandler(db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "all marked as read"})
	}
}

func deleteNotificationHandler(db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

func getTemplatesHandler(db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		templates := []map[string]interface{}{
			{"id": uuid.New().String(), "name": "Welcome Email", "type": "welcome", "channel": "email", "is_active": true},
			{"id": uuid.New().String(), "name": "Transaction Alert", "type": "transaction", "channel": "push", "is_active": true},
		}
		c.JSON(http.StatusOK, gin.H{"templates": templates})
	}
}

func createTemplateHandler(db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"id": uuid.New().String(), "message": "template created"})
	}
}

func updateTemplateHandler(db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "template updated"})
	}
}

func deleteTemplateHandler(db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "template deleted"})
	}
}

func getPreferencesHandler(db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		preferences := map[string]interface{}{
			"email_enabled": true,
			"sms_enabled": true,
			"push_enabled": true,
			"transaction_alerts": true,
			"marketing_alerts": false,
			"security_alerts": true,
		}
		c.JSON(http.StatusOK, preferences)
	}
}

func updatePreferencesHandler(db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "preferences updated"})
	}
}

func sendEmailHandler(queue *NotificationQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "sent"})
	}
}

func sendSMSHandler(queue *NotificationQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "sent"})
	}
}

func sendPushHandler(queue *NotificationQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "sent"})
	}
}

func sendWebhookHandler(queue *NotificationQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "sent"})
	}
}

func broadcastHandler(queue *NotificationQueue, db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "broadcast queued"})
	}
}

// ============== Database ==============

type DB interface {
	Close()
}

func initDatabase(cfg *Config) (DB, error) {
	// In production, implement actual PostgreSQL connection
	// For now, return a mock
	log.Printf("Connecting to PostgreSQL at %s:%d", cfg.Database.Host, cfg.Database.Port)
	return &mockDB{}, nil
}

type mockDB struct{}

func (m *mockDB) Close() {}
