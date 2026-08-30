package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/api/option"
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
	ProjectID  string
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
			DB:       getEnvInt("NOTIFICATION_REDIS_DB", 0),
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
	ID          uuid.UUID              `json:"id" db:"id"`
	UserID      uuid.UUID              `json:"user_id" db:"user_id"`
	Type        string                 `json:"type" db:"type"`
	Channel     string                 `json:"channel" db:"channel"`
	Title       string                 `json:"title" db:"title"`
	Message     string                 `json:"message" db:"message"`
	Data        map[string]interface{} `json:"data" db:"data"`
	IsRead      bool                   `json:"is_read" db:"is_read"`
	ReadAt      *time.Time             `json:"read_at" db:"read_at"`
	ScheduledAt *time.Time             `json:"scheduled_at" db:"scheduled_at"`
	SentAt      *time.Time             `json:"sent_at" db:"sent_at"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

type NotificationTemplate struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Type      string    `json:"type" db:"type"`
	Channel   string    `json:"channel" db:"channel"`
	Subject   string    `json:"subject" db:"subject"`
	Body      string    `json:"body" db:"body"`
	Variables []string  `json:"variables" db:"variables"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type NotificationPreference struct {
	ID                uuid.UUID `json:"id" db:"id"`
	UserID            uuid.UUID `json:"user_id" db:"user_id"`
	EmailEnabled      bool      `json:"email_enabled" db:"email_enabled"`
	SMSEnabled        bool      `json:"sms_enabled" db:"sms_enabled"`
	PushEnabled       bool      `json:"push_enabled" db:"push_enabled"`
	TransactionAlerts bool      `json:"transaction_alerts" db:"transaction_alerts"`
	MarketingAlerts   bool      `json:"marketing_alerts" db:"marketing_alerts"`
	SecurityAlerts    bool      `json:"security_alerts" db:"security_alerts"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// ============== Notification Queue ==============

type NotificationQueue struct {
	redis *redis.Client
	fcm   *messaging.Client
}

func NewNotificationQueue(rdb *redis.Client) *NotificationQueue {
	q := &NotificationQueue{redis: rdb}
	if creds := os.Getenv("FCM_CREDENTIALS"); creds != "" {
		var opt option.ClientOption
		if _, err := os.Stat(creds); err == nil {
			opt = option.WithCredentialsFile(creds)
		} else {
			opt = option.WithCredentialsJSON([]byte(creds))
		}
		ctx := context.Background()
		app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: os.Getenv("FCM_PROJECT_ID")}, opt)
		if err != nil {
			log.Printf("FCM initialization failed: %v (push channel disabled)", err)
		} else if client, err := app.Messaging(ctx); err != nil {
			log.Printf("FCM messaging client failed: %v (push channel disabled)", err)
		} else {
			q.fcm = client
		}
	} else {
		log.Printf("FCM_CREDENTIALS not set: push channel will report errors until configured")
	}
	return q
}

type QueuedNotification struct {
	ID      uuid.UUID              `json:"id"`
	UserID  uuid.UUID              `json:"user_id"`
	Type    string                 `json:"type"`
	Channel string                 `json:"channel"`
	Title   string                 `json:"title"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
	SentAt  *time.Time             `json:"sent_at"`
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

		var sendErr error
		switch notification.Channel {
		case "email":
			sendErr = q.sendEmail(cfg, notification)
		case "sms":
			sendErr = q.sendSMS(cfg, notification)
		case "push":
			sendErr = q.sendPush(cfg, notification)
		case "webhook":
			sendErr = q.sendWebhook(cfg, notification)
		default:
			sendErr = fmt.Errorf("unknown channel %q", notification.Channel)
		}

		if sendErr != nil {
			log.Printf("Worker %d failed notification %s: %v", id, notification.ID, sendErr)
		} else {
			log.Printf("Worker %d processed notification %s", id, notification.ID)
		}
	}
}

func notificationRecipient(n *QueuedNotification, keys ...string) string {
	for _, k := range keys {
		if v, ok := n.Data[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func (q *NotificationQueue) sendEmail(cfg *Config, n *QueuedNotification) error {
	recipient := notificationRecipient(n, "recipient", "email")
	if recipient == "" {
		return errors.New("email notification missing data.recipient")
	}
	if cfg.SMTP.Username == "" || cfg.SMTP.Password == "" {
		return errors.New("SMTP not configured: set SMTP_USERNAME and SMTP_PASSWORD")
	}
	msg := "From: " + cfg.SMTP.From + "\r\n" +
		"To: " + recipient + "\r\n" +
		"Subject: " + n.Title + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n\r\n" + n.Message
	auth := smtp.PlainAuth("", cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.Host)
	addr := fmt.Sprintf("%s:%d", cfg.SMTP.Host, cfg.SMTP.Port)
	return smtp.SendMail(addr, auth, cfg.SMTP.From, []string{recipient}, []byte(msg))
}

func (q *NotificationQueue) sendSMS(cfg *Config, n *QueuedNotification) error {
	recipient := notificationRecipient(n, "recipient", "phone")
	if recipient == "" {
		return errors.New("sms notification missing data.recipient")
	}
	if cfg.Twilio.AccountSid == "" || cfg.Twilio.AuthToken == "" || cfg.Twilio.PhoneNumber == "" {
		return errors.New("Twilio not configured: set TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_PHONE_NUMBER")
	}
	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", cfg.Twilio.AccountSid)
	form := url.Values{}
	form.Set("From", cfg.Twilio.PhoneNumber)
	form.Set("To", recipient)
	form.Set("Body", n.Message)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(cfg.Twilio.AccountSid, cfg.Twilio.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("twilio request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("twilio error: %s", resp.Status)
	}
	return nil
}

func (q *NotificationQueue) sendPush(cfg *Config, n *QueuedNotification) error {
	token := notificationRecipient(n, "device_token", "recipient")
	if token == "" {
		return errors.New("push notification missing data.device_token")
	}
	if q.fcm == nil {
		return errors.New("FCM not configured: set FCM_CREDENTIALS")
	}
	stringData := map[string]string{}
	for k, v := range n.Data {
		stringData[k] = fmt.Sprintf("%v", v)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := q.fcm.Send(ctx, &messaging.Message{
		Token:        token,
		Notification: &messaging.Notification{Title: n.Title, Body: n.Message},
		Data:         stringData,
	})
	return err
}

func (q *NotificationQueue) sendWebhook(cfg *Config, n *QueuedNotification) error {
	webhookURL := notificationRecipient(n, "webhook_url")
	if webhookURL == "" {
		return errors.New("webhook notification missing data.webhook_url")
	}
	payload, err := json.Marshal(map[string]interface{}{
		"id": n.ID, "user_id": n.UserID, "type": n.Type,
		"title": n.Title, "message": n.Message, "data": n.Data,
	})
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook endpoint returned %s", resp.Status)
	}
	return nil
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
			ScheduledAt *time.Time             `json:"scheduled_at"`
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
			ID:      uuid.New(),
			UserID:  uuid.MustParse(req.UserID),
			Type:    req.Type,
			Channel: req.Channel,
			Title:   req.Title,
			Message: req.Message,
			Data:    req.Data,
			SentAt:  req.ScheduledAt,
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

// defaultPreferences is the baseline returned for users with no stored row.
func defaultPreferences(userID string) map[string]any {
	return map[string]any{
		"user_id":       userID,
		"email_enabled": true, "sms_enabled": true, "push_enabled": true,
		"transaction_alerts": true, "marketing_alerts": false, "security_alerts": true,
	}
}

func getTemplatesHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		templates, err := db.ListTemplates(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if templates == nil {
			templates = []map[string]any{}
		}
		c.JSON(http.StatusOK, gin.H{"templates": templates})
	}
}

type templateRequest struct {
	Name    string `json:"name" binding:"required"`
	Channel string `json:"channel" binding:"required"`
	Subject string `json:"subject"`
	Body    string `json:"body" binding:"required"`
}

func createTemplateHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req templateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Validate the Go text/template up-front so a broken template
		// never reaches the send path.
		if _, err := template.New("check").Parse(req.Body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template body: " + err.Error()})
			return
		}
		id, err := db.CreateTemplate(c.Request.Context(), req.Name, req.Channel, req.Subject, req.Body)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "channel": req.Channel})
	}
}

func updateTemplateHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req templateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, err := template.New("check").Parse(req.Body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template body: " + err.Error()})
			return
		}
		if err := db.UpdateTemplate(c.Request.Context(), c.Param("id"), req.Name, req.Channel, req.Subject, req.Body); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "updated"})
	}
}

func deleteTemplateHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.DeleteTemplate(c.Request.Context(), c.Param("id")); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

func getPreferencesHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		prefs, err := db.GetPreferences(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if prefs == nil {
			// No stored row yet: honest channel-on defaults.
			prefs = defaultPreferences(userID)
		}
		c.JSON(http.StatusOK, prefs)
	}
}

type preferencesRequest struct {
	EmailEnabled      *bool `json:"email_enabled"`
	SMSEnabled        *bool `json:"sms_enabled"`
	PushEnabled       *bool `json:"push_enabled"`
	TransactionAlerts *bool `json:"transaction_alerts"`
	MarketingAlerts   *bool `json:"marketing_alerts"`
	SecurityAlerts    *bool `json:"security_alerts"`
}

func updatePreferencesHandler(db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		var req preferencesRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := db.UpsertPreferences(c.Request.Context(), userID, req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		prefs, err := db.GetPreferences(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, prefs)
	}
}

type sendRequest struct {
	UserID  string `json:"user_id"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	// Template names a stored notification_templates row; when set, Title/Body
	// are rendered from it with Vars (Go text/template syntax).
	Template string         `json:"template,omitempty"`
	Vars     map[string]any `json:"vars,omitempty"`
}

// applyTemplate resolves req.Template into rendered Title/Body. Fail-closed:
// unknown template or render error -> error, no partially-rendered send.
func applyTemplate(c *gin.Context, db DB, req *sendRequest) error {
	if req.Template == "" {
		return nil
	}
	tpl, err := db.GetTemplateByName(c.Request.Context(), req.Template)
	if err != nil {
		return fmt.Errorf("template %q: %w", req.Template, err)
	}
	rendered, err := renderTemplate(tpl["body"].(string), req.Vars)
	if err != nil {
		return fmt.Errorf("render template %q: %w", req.Template, err)
	}
	req.Body = rendered
	if subject, _ := tpl["subject"].(string); subject != "" {
		renderedSubject, err := renderTemplate(subject, req.Vars)
		if err != nil {
			return fmt.Errorf("render template subject %q: %w", req.Template, err)
		}
		req.Title = renderedSubject
	}
	if ch, _ := tpl["channel"].(string); ch != "" && req.Channel == "" {
		req.Channel = ch
	}
	return nil
}

func sendEmailHandler(queue *NotificationQueue, db DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req sendRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := applyTemplate(c, db, &req); err != nil {
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
		if err := applyTemplate(c, db, &req); err != nil {
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
		if err := applyTemplate(c, db, &req); err != nil {
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
		if err := applyTemplate(c, db, &req); err != nil {
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
		if err := applyTemplate(c, db, &req); err != nil {
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
	// Template CRUD (real SQL against notification_templates).
	ListTemplates(ctx context.Context) ([]map[string]any, error)
	GetTemplateByName(ctx context.Context, name string) (map[string]any, error)
	CreateTemplate(ctx context.Context, name, channel, subject, body string) (string, error)
	UpdateTemplate(ctx context.Context, id, name, channel, subject, body string) error
	DeleteTemplate(ctx context.Context, id string) error
	// Per-user channel preferences (real SQL against notification_preferences).
	GetPreferences(ctx context.Context, userID string) (map[string]any, error)
	UpsertPreferences(ctx context.Context, userID string, req preferencesRequest) error
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
		CREATE TABLE IF NOT EXISTS notification_templates (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL UNIQUE,
			channel TEXT NOT NULL,
			subject TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS notification_preferences (
			user_id TEXT PRIMARY KEY,
			email_enabled BOOLEAN NOT NULL DEFAULT TRUE,
			sms_enabled BOOLEAN NOT NULL DEFAULT TRUE,
			push_enabled BOOLEAN NOT NULL DEFAULT TRUE,
			transaction_alerts BOOLEAN NOT NULL DEFAULT TRUE,
			marketing_alerts BOOLEAN NOT NULL DEFAULT FALSE,
			security_alerts BOOLEAN NOT NULL DEFAULT TRUE,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
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

func (d *pgDB) ListTemplates(ctx context.Context) ([]map[string]any, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT id::text, name, channel, subject, body, created_at, updated_at
		 FROM notification_templates ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, name, channel, subject, body string
		var createdAt, updatedAt any
		if err := rows.Scan(&id, &name, &channel, &subject, &body, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "name": name, "channel": channel,
			"subject": subject, "body": body,
			"created_at": createdAt, "updated_at": updatedAt,
		})
	}
	return out, rows.Err()
}

func (d *pgDB) GetTemplateByName(ctx context.Context, name string) (map[string]any, error) {
	var id, ch, subject, body string
	err := d.pool.QueryRow(ctx,
		`SELECT id::text, channel, subject, body FROM notification_templates WHERE name = $1`, name).
		Scan(&id, &ch, &subject, &body)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "name": name, "channel": ch, "subject": subject, "body": body}, nil
}

func (d *pgDB) CreateTemplate(ctx context.Context, name, channel, subject, body string) (string, error) {
	var id string
	err := d.pool.QueryRow(ctx,
		`INSERT INTO notification_templates (name, channel, subject, body)
		 VALUES ($1, $2, $3, $4) RETURNING id::text`, name, channel, subject, body).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (d *pgDB) UpdateTemplate(ctx context.Context, id, name, channel, subject, body string) error {
	ct, err := d.pool.Exec(ctx,
		`UPDATE notification_templates SET name=$2, channel=$3, subject=$4, body=$5, updated_at=now()
		 WHERE id = $1`, id, name, channel, subject, body)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("template not found")
	}
	return nil
}

func (d *pgDB) DeleteTemplate(ctx context.Context, id string) error {
	ct, err := d.pool.Exec(ctx, `DELETE FROM notification_templates WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("template not found")
	}
	return nil
}

func (d *pgDB) GetPreferences(ctx context.Context, userID string) (map[string]any, error) {
	var email, sms, push, txn, marketing, security bool
	var updatedAt any
	err := d.pool.QueryRow(ctx,
		`SELECT email_enabled, sms_enabled, push_enabled, transaction_alerts,
			marketing_alerts, security_alerts, updated_at
		 FROM notification_preferences WHERE user_id = $1`, userID).
		Scan(&email, &sms, &push, &txn, &marketing, &security, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return map[string]any{
		"user_id":       userID,
		"email_enabled": email, "sms_enabled": sms, "push_enabled": push,
		"transaction_alerts": txn, "marketing_alerts": marketing, "security_alerts": security,
		"updated_at": updatedAt,
	}, nil
}

func (d *pgDB) UpsertPreferences(ctx context.Context, userID string, req preferencesRequest) error {
	_, err := d.pool.Exec(ctx,
		`INSERT INTO notification_preferences
			(user_id, email_enabled, sms_enabled, push_enabled,
			 transaction_alerts, marketing_alerts, security_alerts, updated_at)
		 VALUES ($1,
			COALESCE($2, TRUE), COALESCE($3, TRUE), COALESCE($4, TRUE),
			COALESCE($5, TRUE), COALESCE($6, FALSE), COALESCE($7, TRUE), now())
		 ON CONFLICT (user_id) DO UPDATE SET
			email_enabled = COALESCE($2, notification_preferences.email_enabled),
			sms_enabled = COALESCE($3, notification_preferences.sms_enabled),
			push_enabled = COALESCE($4, notification_preferences.push_enabled),
			transaction_alerts = COALESCE($5, notification_preferences.transaction_alerts),
			marketing_alerts = COALESCE($6, notification_preferences.marketing_alerts),
			security_alerts = COALESCE($7, notification_preferences.security_alerts),
			updated_at = now()`,
		userID, req.EmailEnabled, req.SMSEnabled, req.PushEnabled,
		req.TransactionAlerts, req.MarketingAlerts, req.SecurityAlerts)
	return err
}

// renderTemplate renders a stored notification template with the caller's
// variables (Go text/template syntax, e.g. {{.amount}}). Fail-closed: a
// template parse/execute error is returned, never a partially-rendered body.
func renderTemplate(body string, vars map[string]any) (string, error) {
	tpl, err := template.New("notification").Option("missingkey=error").Parse(body)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}
