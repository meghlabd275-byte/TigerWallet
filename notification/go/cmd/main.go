package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	cfg := loadConfig()

	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-notification"})
	})

	api := router.Group("/api/v1/notifications")
	{
		// Public
		api.POST("/subscribe", subscribeHandler)

		// Protected
		protected := api.Group("")
		protected.Use(jwtAuthMiddleware())
		{
			// Send notifications
			protected.POST("/send", sendNotificationHandler)
			protected.POST("/send/batch", sendBatchNotificationsHandler)
			protected.POST("/send/email", sendEmailHandler)
			protected.POST("/send/sms", sendSMSHandler)
			protected.POST("/send/push", sendPushHandler)

			// Templates
			protected.GET("/templates", listTemplatesHandler)
			protected.POST("/templates", createTemplateHandler)
			protected.PUT("/templates/:id", updateTemplateHandler)
			protected.DELETE("/templates/:id", deleteTemplateHandler)

			// Notification history
			protected.GET("/history", getNotificationHistoryHandler)
			protected.GET("/history/:id", getNotificationByIDHandler)
			protected.GET("/stats", getNotificationStatsHandler)

			// Preferences
			protected.GET("/preferences", getPreferencesHandler)
			protected.PUT("/preferences", updatePreferencesHandler)

			// Channels
			protected.GET("/channels", getChannelsHandler)
			protected.PUT("/channels/:channel/status", updateChannelStatusHandler)
		}

		// Webhooks
		api.POST("/webhooks/email/status", emailStatusWebhookHandler)
		api.POST("/webhooks/sms/status", smsStatusWebhookHandler)
		api.POST("/webhooks/push/status", pushStatusWebhookHandler)

		// Unsubscribe
		api.POST("/unsubscribe", unsubscribeHandler)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	go func() {
		log.Printf("Notification service starting on port %s", cfg.Port)
		srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

type Config struct {
	Port     string
	Database DatabaseConfig
	SMTP     SMTPConfig
	Twilio   TwilioConfig
	FCM      FCMConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
}

type TwilioConfig struct {
	AccountSID  string
	AuthToken   string
	PhoneNumber string
}

type FCMConfig struct {
	ProjectID string
	Credentials string
}

func loadConfig() *Config {
	return &Config{
		Port: getEnv("NOTIFICATION_PORT", "9002"),
		Database: DatabaseConfig{
			Host:     getEnv("NOTIFICATION_DB_HOST", "localhost"),
			Port:     getEnvInt("NOTIFICATION_DB_PORT", 5432),
			User:     getEnv("NOTIFICATION_DB_USER", "tigerwallet"),
			Password: getEnv("NOTIFICATION_DB_PASSWORD", "password"),
			DBName:   getEnv("NOTIFICATION_DB_NAME", "tigerwallet_notification"),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "smtp.mailgun.org"),
			Port:     getEnvInt("SMTP_PORT", 587),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@tigerwallet.com"),
			FromName: getEnv("SMTP_FROM_NAME", "TigerWallet"),
		},
		Twilio: TwilioConfig{
			AccountSID:  getEnv("TWILIO_ACCOUNT_SID", ""),
			AuthToken:   getEnv("TWILIO_AUTH_TOKEN", ""),
			PhoneNumber: getEnv("TWILIO_PHONE_NUMBER", ""),
		},
		FCM: FCMConfig{
			ProjectID:  getEnv("FCM_PROJECT_ID", ""),
			Credentials: getEnv("FCM_CREDENTIALS", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	var v int
	_, err := fmt.Sscan(os.Getenv(key), &v)
	if err != nil {
		return defaultValue
	}
	return v
}

// Models
type Notification struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	TenantID   *uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Type       string    `json:"type" db:"type"` // email, sms, push, in_app
	Channel    string    `json:"channel" db:"channel"`
	TemplateID *uuid.UUID `json:"template_id" db:"template_id"`
	Subject    string    `json:"subject" db:"subject"`
	Body       string    `json:"body" db:"body"`
	Recipients []string  `json:"recipients" db:"recipients"`
	Status     string    `json:"status" db:"status"` // pending, sent, delivered, failed
	SentAt     *time.Time `json:"sent_at" db:"sent_at"`
	DeliveredAt *time.Time `json:"delivered_at" db:"delivered_at"`
	FailedAt   *time.Time `json:"failed_at" db:"failed_at"`
	ErrorMsg   string    `json:"error_msg" db:"error_msg"`
	Metadata   string    `json:"metadata" db:"metadata"` // JSON
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type Template struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID   *uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name       string    `json:"name" db:"name"`
	Type       string    `json:"type" db:"type"` // email, sms, push
	Subject    string    `json:"subject" db:"subject"`
	Body       string    `json:"body" db:"body"`
	HTMLBody   string    `json:"html_body" db:"html_body"`
	Variables  []string  `json:"variables" db:"variables"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type UserPreference struct {
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	EmailEnabled   bool      `json:"email_enabled" db:"email_enabled"`
	SMSEnabled    bool      `json:"sms_enabled" db:"sms_enabled"`
	PushEnabled   bool      `json:"push_enabled" db:"push_enabled"`
	InAppEnabled  bool      `json:"in_app_enabled" db:"in_app_enabled"`
	EmailFrequency string   `json:"email_frequency" db:"email_frequency"` // immediate, daily, weekly
	MarketingOptIn bool     `json:"marketing_opt_in" db:"marketing_opt_in"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type Subscriber struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Email          string    `json:"email" db:"email"`
	Phone          string    `json:"phone" db:"phone"`
	DeviceToken    string    `json:"device_token" db:"device_token"`
	TenantID      *uuid.UUID `json:"tenant_id" db:"tenant_id"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	SubscribedAt  time.Time `json:"subscribed_at" db:"subscribed_at"`
	UnsubscribedAt *time.Time `json:"unsubscribed_at" db:"unsubscribed_at"`
}

type Channel struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Type      string    `json:"type" db:"type"` // email, sms, push
	Status    string    `json:"status" db:"status"` // active, inactive, error
	Config    string    `json:"config" db:"config"` // JSON
	RateLimit int       `json:"rate_limit" db:"rate_limit"` // per second
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Handlers
func sendNotificationHandler(c *gin.Context) {
	var req struct {
		UserID     string   `json:"user_id" binding:"required"`
		Type       string   `json:"type" binding:"required"` // email, sms, push, in_app
		TemplateID string   `json:"template_id"`
		Subject    string   `json:"subject"`
		Body       string   `json:"body" binding:"required"`
		Recipients []string `json:"recipients"`
		Metadata   map[string]interface{} `json:"metadata"`
	}
	c.ShouldBindJSON(&req)

	notification := map[string]interface{}{
		"id":         uuid.New().String(),
		"user_id":    req.UserID,
		"type":       req.Type,
		"template_id": req.TemplateID,
		"subject":    req.Subject,
		"body":       req.Body,
		"status":     "sent",
		"sent_at":    time.Now().Unix(),
		"created_at": time.Now().Unix(),
	}

	// Send based on type
	switch req.Type {
	case "email":
		go sendEmail(req.Recipients, req.Subject, req.Body)
	case "sms":
		go sendSMS(req.Recipients, req.Body)
	case "push":
		go sendPushNotification(req.Recipients, req.Subject, req.Body)
	}

	c.JSON(http.StatusOK, gin.H{"notification": notification, "message": "Notification sent"})
}

func sendBatchNotificationsHandler(c *gin.Context) {
	var req struct {
		Notifications []map[string]interface{} `json:"notifications" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	sent := 0
	failed := 0

	for _, notif := range req.Notifications {
		// Send each notification
		sent++
	}

	c.JSON(http.StatusOK, gin.H{
		"sent":   sent,
		"failed": failed,
		"total":  len(req.Notifications),
	})
}

func sendEmailHandler(c *gin.Context) {
	var req struct {
		To          []string               `json:"to" binding:"required"`
		Subject     string                 `json:"subject" binding:"required"`
		Body        string                 `json:"body"`
		HTMLBody    string                 `json:"html_body"`
		TemplateID  string                 `json:"template_id"`
		Variables   map[string]interface{} `json:"variables"`
	}
	c.ShouldBindJSON(&req)

	// Get template if provided
	var subject = req.Subject
	var body = req.Body
	var htmlBody = req.HTMLBody

	if req.TemplateID != "" {
		template := getEmailTemplate(req.TemplateID)
		if template != nil {
			subject = interpolateTemplate(template["subject"].(string), req.Variables)
			body = interpolateTemplate(template["body"].(string), req.Variables)
			htmlBody = interpolateTemplate(template["html_body"].(string), req.Variables)
		}
	}

	err := sendEmail(req.To, subject, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email sent", "recipients": len(req.To)})
}

func sendSMSHandler(c *gin.Context) {
	var req struct {
		To       []string               `json:"to" binding:"required"`
		Message  string                 `json:"message" binding:"required"`
		Template string                 `json:"template"`
		Variables map[string]interface{} `json:"variables"`
	}
	c.ShouldBindJSON(&req)

	message := req.Message
	if req.Template != "" {
		template := getSMSTemplate(req.Template)
		if template != nil {
			message = interpolateTemplate(template["body"].(string), req.Variables)
		}
	}

	err := sendSMS(req.To, message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "SMS sent", "recipients": len(req.To)})
}

func sendPushHandler(c *gin.Context) {
	var req struct {
		DeviceTokens []string               `json:"device_tokens" binding:"required"`
		Title       string                 `json:"title" binding:"required"`
		Body        string                 `json:"body" binding:"required"`
		Data        map[string]interface{} `json:"data"`
		ImageURL    string                 `json:"image_url"`
	}
	c.ShouldBindJSON(&req)

	err := sendPushNotification(req.DeviceTokens, req.Title, req.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Push notification sent", "recipients": len(req.DeviceTokens)})
}

func subscribeHandler(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		DeviceToken string `json:"device_token"`
		TenantID    string `json:"tenant_id"`
	}
	c.ShouldBindJSON(&req)

	subscriber := map[string]interface{}{
		"id":            uuid.New().String(),
		"email":        req.Email,
		"phone":        req.Phone,
		"device_token": req.DeviceToken,
		"tenant_id":    req.TenantID,
		"is_active":    true,
		"subscribed_at": time.Now().Unix(),
	}

	c.JSON(http.StatusCreated, gin.H{"subscriber": subscriber, "message": "Subscribed successfully"})
}

func unsubscribeHandler(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		DeviceToken string `json:"device_token"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribed successfully"})
}

func listTemplatesHandler(c *gin.Context) {
	templates := []map[string]interface{}{
		{
			"id":       uuid.New().String(),
			"name":     "Welcome Email",
			"type":     "email",
			"subject":  "Welcome to TigerWallet",
			"is_active": true,
		},
		{
			"id":       uuid.New().String(),
			"name":     "Password Reset",
			"type":     "email",
			"subject":  "Reset Your Password",
			"is_active": true,
		},
		{
			"id":       uuid.New().String(),
			"name":     "Transaction Alert",
			"type":     "push",
			"is_active": true,
		},
	}

	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

func createTemplateHandler(c *gin.Context) {
	var req struct {
		Name      string   `json:"name" binding:"required"`
		Type      string   `json:"type" binding:"required"`
		Subject   string   `json:"subject"`
		Body      string   `json:"body" binding:"required"`
		HTMLBody  string   `json:"html_body"`
		Variables []string `json:"variables"`
	}
	c.ShouldBindJSON(&req)

	template := map[string]interface{}{
		"id":          uuid.New().String(),
		"name":        req.Name,
		"type":        req.Type,
		"subject":     req.Subject,
		"body":        req.Body,
		"html_body":   req.HTMLBody,
		"variables":   req.Variables,
		"is_active":   true,
		"created_at":  time.Now().Unix(),
	}

	c.JSON(http.StatusCreated, gin.H{"template": template})
}

func updateTemplateHandler(c *gin.Context) {
	templateID := c.Param("id")

	var req struct {
		Name      string   `json:"name"`
		Subject   string   `json:"subject"`
		Body      string   `json:"body"`
		HTMLBody  string   `json:"html_body"`
		Variables []string `json:"variables"`
		IsActive  *bool   `json:"is_active"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{"message": "Template updated"})
}

func deleteTemplateHandler(c *gin.Context) {
	templateID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"message": "Template deleted"})
}

func getNotificationHistoryHandler(c *gin.Context) {
	userID := c.Query("user_id")
	notifType := c.Query("type")
	status := c.Query("status")
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	notifications := []map[string]interface{}{
		{
			"id":       uuid.New().String(),
			"type":    "email",
			"subject": "Welcome to TigerWallet",
			"status":  "delivered",
			"sent_at": time.Now().Add(-24 * time.Hour).Unix(),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"total":        100,
	})
}

func getNotificationByIDHandler(c *gin.Context) {
	notifID := c.Param("id")

	notification := map[string]interface{}{
		"id":         notifID,
		"type":       "email",
		"subject":    "Welcome to TigerWallet",
		"body":       "Welcome to TigerWallet...",
		"status":     "delivered",
		"sent_at":    time.Now().Add(-24 * time.Hour).Unix(),
		"delivered_at": time.Now().Add(-23 * time.Hour).Unix(),
	}

	c.JSON(http.StatusOK, notification)
}

func getNotificationStatsHandler(c *gin.Context) {
	stats := map[string]interface{}{
		"total_sent":      10000,
		"total_delivered": 9500,
		"total_failed":    500,
		"by_type": map[string]interface{}{
			"email": 6000,
			"sms":   3000,
			"push":  1000,
		},
		"delivery_rate":   95.0,
		"open_rate":       45.0,
		"click_rate":      20.0,
	}

	c.JSON(http.StatusOK, stats)
}

func getPreferencesHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	preferences := map[string]interface{}{
		"user_id":           userID,
		"email_enabled":    true,
		"sms_enabled":     true,
		"push_enabled":    true,
		"in_app_enabled":  true,
		"email_frequency": "immediate",
		"marketing_opt_in": true,
	}

	c.JSON(http.StatusOK, preferences)
}

func updatePreferencesHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		EmailEnabled   *bool  `json:"email_enabled"`
		SMSEnabled    *bool  `json:"sms_enabled"`
		PushEnabled   *bool  `json:"push_enabled"`
		InAppEnabled  *bool  `json:"in_app_enabled"`
		EmailFrequency string `json:"email_frequency"`
		MarketingOptIn *bool  `json:"marketing_opt_in"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{"message": "Preferences updated"})
}

func getChannelsHandler(c *gin.Context) {
	channels := []map[string]interface{}{
		{
			"id":         uuid.New().String(),
			"name":       "Primary Email",
			"type":       "email",
			"status":     "active",
			"rate_limit": 100,
		},
		{
			"id":         uuid.New().String(),
			"name":       "SMS Gateway",
			"type":       "sms",
			"status":     "active",
			"rate_limit": 50,
		},
		{
			"id":         uuid.New().String(),
			"name":       "Push Notifications",
			"type":       "push",
			"status":     "active",
			"rate_limit": 1000,
		},
	}

	c.JSON(http.StatusOK, gin.H{"channels": channels})
}

func updateChannelStatusHandler(c *gin.Context) {
	channelID := c.Param("channel")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{"message": "Channel status updated"})
}

func emailStatusWebhookHandler(c *gin.Context) {
	var req struct {
		MessageID string `json:"message_id"`
		Status    string `json:"status"` // sent, delivered, bounced, failed
	}
	c.ShouldBindJSON(&req)

	updateNotificationStatus(req.MessageID, req.Status)

	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}

func smsStatusWebhookHandler(c *gin.Context) {
	var req struct {
		MessageID string `json:"message_id"`
		Status    string `json:"status"` // queued, sent, delivered, failed
	}
	c.ShouldBindJSON(&req)

	updateNotificationStatus(req.MessageID, req.Status)

	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}

func pushStatusWebhookHandler(c *gin.Context) {
	var req struct {
		MessageID string `json:"message_id"`
		Status    string `json:"status"` // sent, delivered, failed
	}
	c.ShouldBindJSON(&req)

	updateNotificationStatus(req.MessageID, req.Status)

	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}

// Services
func sendEmail(to []string, subject, body string) error {
	log.Printf("Sending email to %v: %s", to, subject)
	return nil
}

func sendSMS(to []string, message string) error {
	log.Printf("Sending SMS to %v: %s", to, message)
	return nil
}

func sendPushNotification(tokens []string, title, body string) error {
	log.Printf("Sending push to %v: %s - %s", tokens, title, body)
	return nil
}

func getEmailTemplate(id string) map[string]interface{} {
	return map[string]interface{}{
		"subject":   "Welcome to TigerWallet",
		"body":       "Hello {{name}}, welcome to TigerWallet!",
		"html_body":  "<h1>Welcome {{name}}</h1><p>Welcome to TigerWallet!</p>",
	}
}

func getSMSTemplate(id string) map[string]interface{} {
	return map[string]interface{}{
		"body": "TigerWallet: {{message}}",
	}
}

func interpolateTemplate(template string, vars map[string]interface{}) string {
	result := template
	for k, v := range vars {
		result = strings.Replace(result, "{{"+k+"}}", fmt.Sprintf("%v", v), -1)
	}
	return result
}

func updateNotificationStatus(messageID, status string) {
	log.Printf("Updating notification %s status to %s", messageID, status)
}

// Middleware
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func jwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}
		c.Set("user_id", "user-123")
		c.Next()
	}
}

// Database
type DB struct{}

func initDatabase(cfg *Config) (*DB, error) {
	log.Printf("Connecting to PostgreSQL at %s:%d", cfg.Database.Host, cfg.Database.Port)
	return &DB{}, nil
}

func (d *DB) Close() {}

var strings = struct {
	Replace func(s, old, new string, n int) string
}{
	Replace: func(s, old, new string, n int) string {
		result := s
		for i := 0; n < 0 || i < n; i++ {
			if idx := find(result, old); idx >= 0 {
				result = result[:idx] + new + result[idx+len(old):]
			} else {
				break
			}
		}
		return result
	},
}

func find(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
