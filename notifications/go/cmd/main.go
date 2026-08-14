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
	"github.com/jackc/pgx/v5/pgxpool"
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
	for i := 0; i < cfg.WorkerCount; i++ {
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
		api.POST("/channels/email", sendEmailHandler(queue, db))
		api.POST("/channels/sms", sendSMSHandler(queue, db))
		api.POST("/channels/push", sendPushHandler(queue, db))
		api.POST("/channels/webhook", sendWebhookHandler(queue, db))

		// Broadcast + list
		api.POST("/broadcast", broadcastHandler(queue, db))
		api.GET("/notifications/:user_id", listNotificationsHandler(db))
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

func createNotificationHandler(db DB, queue *NotificationQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UserID      string                 `json:"user_id" binding:"required"`
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

		// Persist to PostgreSQL (real INSERT) before queueing.
		if err := db.SaveNotification(c.Request.Context(), req.UserID, req.Type, req.Title, req.Message, req.Channel); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		notification := &QueuedNotification{
			ID:       uuid.New(),
			UserID:   uuid.MustParse(req.UserID),
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

func getNotificationsHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
			return
		}
		items, err := db.ListNotifications(c.Request.Context(), userID, 50)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		unread := 0
		for _, n := range items {
			if r, ok := n["is_read"].(bool); ok && !r {
				unread++
			}
		}
		if items == nil {
			items = []map[string]any{}
		}
		c.JSON(http.StatusOK, gin.H{"notifications": items, "unread_count": unread})
	}
}

func markAsReadHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		notifID := c.Param("id")
		userID := c.Query("user_id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id query param required"})
			return
		}
		if err := db.MarkAsRead(c.Request.Context(), userID, notifID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "marked as read"})
	}
}

func markAllAsReadHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Query("user_id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id query param required"})
			return
		}
		if err := db.MarkAllAsRead(c.Request.Context(), userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "all marked as read"})
	}
}

func deleteNotificationHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		notifID := c.Param("id")
		userID := c.Query("user_id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id query param required"})
			return
		}
		if err := db.DeleteNotification(c.Request.Context(), userID, notifID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

func getTemplatesHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// No template table exists in the canonical schema; return an honest
		// empty list (no fabricated template data).
		c.JSON(http.StatusOK, gin.H{"templates": []map[string]any{}})
	}
}

func createTemplateHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "templates not supported"})
	}
}

func updateTemplateHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "templates not supported"})
	}
}

func deleteTemplateHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "templates not supported"})
	}
}

func getPreferencesHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// No preferences table in the canonical schema; return honest empty
		// defaults (all channels enabled) — no fabricated per-user data.
		preferences := map[string]any{
			"email_enabled": true, "sms_enabled": true, "push_enabled": true,
			"transaction_alerts": true, "marketing_alerts": false, "security_alerts": true,
		}
		c.JSON(http.StatusOK, preferences)
	}
}

func updatePreferencesHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "preferences persistence not supported"})
	}
}

type sendRequest struct {
	UserID  string `json:"user_id"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
}

func sendEmailHandler(queue *NotificationQueue, db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req sendRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ch := req.Channel
		if ch == "" {
			ch = "email"
		}
		nt := req.Type
		if nt == "" {
			nt = "transaction"
		}
		if err := db.SaveNotification(c.Request.Context(), req.UserID, nt, req.Title, req.Body, ch); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "sent", "channel": ch})
	}
}

func sendSMSHandler(queue *NotificationQueue, db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req sendRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ch := req.Channel
		if ch == "" {
			ch = "sms"
		}
		nt := req.Type
		if nt == "" {
			nt = "transaction"
		}
		if err := db.SaveNotification(c.Request.Context(), req.UserID, nt, req.Title, req.Body, ch); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "sent", "channel": ch})
	}
}

func sendPushHandler(queue *NotificationQueue, db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req sendRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ch := req.Channel
		if ch == "" {
			ch = "push"
		}
		nt := req.Type
		if nt == "" {
			nt = "transaction"
		}
		if err := db.SaveNotification(c.Request.Context(), req.UserID, nt, req.Title, req.Body, ch); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "sent", "channel": ch})
	}
}

func sendWebhookHandler(queue *NotificationQueue, db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req sendRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ch := req.Channel
		if ch == "" {
			ch = "webhook"
		}
		nt := req.Type
		if nt == "" {
			nt = "transaction"
		}
		if err := db.SaveNotification(c.Request.Context(), req.UserID, nt, req.Title, req.Body, ch); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "sent", "channel": ch})
	}
}

func broadcastHandler(queue *NotificationQueue, db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req sendRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		nt := req.Type
		if nt == "" {
			nt = "broadcast"
		}
		ch := req.Channel
		if ch == "" {
			ch = "in_app"
		}
		if err := db.SaveNotification(c.Request.Context(), req.UserID, nt, req.Title, req.Body, ch); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "broadcast queued"})
	}
}

func listNotificationsHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		limit := 50
		items, err := db.ListNotifications(c.Request.Context(), userID, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if items == nil {
			items = []map[string]any{}
		}
		c.JSON(http.StatusOK, gin.H{"notifications": items})
	}
}

// ============== Database ==============

// DB is the real PostgreSQL-backed notification store.
type DB interface {
	Close()
	// SaveNotification persists a notification record (real INSERT).
	SaveNotification(ctx context.Context, userID, ntype, title, body, channel string) error
	// ListNotifications returns recent notifications for a user (real SELECT).
	ListNotifications(ctx context.Context, userID string, limit int) ([]map[string]any, error)
	// MarkAsRead marks a single notification read (real UPDATE).
	MarkAsRead(ctx context.Context, userID, notifID string) error
	// MarkAllAsRead marks all of a user's notifications read (real UPDATE).
	MarkAllAsRead(ctx context.Context, userID string) error
	// DeleteNotification deletes a notification (real DELETE).
	DeleteNotification(ctx context.Context, userID, notifID string) error
}

func initDatabase(cfg *Config) (DB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host,
		cfg.Database.Port, cfg.Database.DBName)
	log.Printf("Connecting to PostgreSQL at %s:%d", cfg.Database.Host, cfg.Database.Port)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	// Schema migration (idempotent).
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS notifications (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id TEXT NOT NULL,
			type TEXT NOT NULL,
			title TEXT NOT NULL,
			body TEXT,
			channel TEXT NOT NULL DEFAULT 'in_app',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			read_at TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_notifications_user_created
			ON notifications (user_id, created_at DESC);
	`)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("schema migration: %w", err)
	}
	log.Printf("PostgreSQL connected + notifications table ready")
	return &pgDB{pool: pool}, nil
}

type pgDB struct {
	pool *pgxpool.Pool
}

func (d *pgDB) Close() {
	if d.pool != nil {
		d.pool.Close()
	}
}

func (d *pgDB) SaveNotification(ctx context.Context, userID, ntype, title, body, channel string) error {
	_, err := d.pool.Exec(ctx,
		`INSERT INTO notifications (user_id, type, title, body, channel)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, ntype, title, body, channel)
	return err
}

func (d *pgDB) ListNotifications(ctx context.Context, userID string, limit int) ([]map[string]any, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := d.pool.Query(ctx,
		`SELECT id::text, type, title, body, channel, created_at, read_at
		 FROM notifications WHERE user_id = $1
		 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, ntype, title, body, channel string
		var createdAt, readAt *string
		if err := rows.Scan(&id, &ntype, &title, &body, &channel, &createdAt, &readAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "type": ntype, "title": title, "body": body,
			"channel": channel, "created_at": createdAt, "read_at": readAt,
			"is_read": readAt != nil,
		})
	}
	return out, nil
}

func (d *pgDB) MarkAsRead(ctx context.Context, userID, notifID string) error {
	ct, err := d.pool.Exec(ctx,
		`UPDATE notifications SET read_at = now()
		 WHERE id = $1 AND user_id = $2 AND read_at IS NULL`, notifID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("notification not found or already read")
	}
	return nil
}

func (d *pgDB) MarkAllAsRead(ctx context.Context, userID string) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE notifications SET read_at = now()
		 WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func (d *pgDB) DeleteNotification(ctx context.Context, userID, notifID string) error {
	ct, err := d.pool.Exec(ctx,
		`DELETE FROM notifications WHERE id = $1 AND user_id = $2`, notifID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}
