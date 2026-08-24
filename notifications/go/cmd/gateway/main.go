// TigerWallet Notification Gateway - multi-channel (email/SMS/push) notification service.
// Real implementations only: PostgreSQL persistence, SMTP email, Twilio SMS, FCM push.
// Every channel fails closed with an explicit error when its provider is not configured.
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/api/option"
)

type Config struct {
	Port      string
	JWTSecret string
	Database  DatabaseConfig
	SMTP      SMTPConfig
	Twilio    TwilioConfig
	FCM       FCMConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	URL      string
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
	ProjectID   string
	Credentials string
}

type Server struct {
	cfg *Config
	db  *pgxpool.Pool
	fcm *messaging.Client
}

func main() {
	cfg := loadConfig()

	ctx := context.Background()
	db, err := initDatabase(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	if err := migrate(ctx, db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	srv := &Server{cfg: cfg, db: db}
	if cfg.FCM.Credentials != "" {
		app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.FCM.ProjectID},
			option.WithCredentialsFile(cfg.FCM.Credentials))
		if err != nil {
			log.Fatalf("Failed to initialize FCM: %v", err)
		}
		srv.fcm, err = app.Messaging(ctx)
		if err != nil {
			log.Fatalf("Failed to initialize FCM messaging: %v", err)
		}
	} else {
		log.Printf("FCM_CREDENTIALS not set: push channel will return errors until configured")
	}

	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-notification"})
	})

	api := router.Group("/api/v1/notifications")
	{
		api.POST("/subscribe", srv.subscribeHandler)

		protected := api.Group("")
		protected.Use(srv.jwtAuthMiddleware())
		{
			protected.POST("/send", srv.sendNotificationHandler)
			protected.POST("/send/batch", srv.sendBatchNotificationsHandler)
			protected.POST("/send/email", srv.sendEmailHandler)
			protected.POST("/send/sms", srv.sendSMSHandler)
			protected.POST("/send/push", srv.sendPushHandler)

			protected.GET("/templates", srv.listTemplatesHandler)
			protected.POST("/templates", srv.createTemplateHandler)
			protected.PUT("/templates/:id", srv.updateTemplateHandler)
			protected.DELETE("/templates/:id", srv.deleteTemplateHandler)

			protected.GET("/history", srv.getNotificationHistoryHandler)
			protected.GET("/history/:id", srv.getNotificationByIDHandler)
			protected.GET("/stats", srv.getNotificationStatsHandler)

			protected.GET("/preferences", srv.getPreferencesHandler)
			protected.PUT("/preferences", srv.updatePreferencesHandler)

			protected.GET("/channels", srv.getChannelsHandler)
			protected.PUT("/channels/:channel/status", srv.updateChannelStatusHandler)
		}

		api.POST("/webhooks/email/status", srv.emailStatusWebhookHandler)
		api.POST("/webhooks/sms/status", srv.smsStatusWebhookHandler)
		api.POST("/webhooks/push/status", srv.pushStatusWebhookHandler)

		api.POST("/unsubscribe", srv.unsubscribeHandler)
	}

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	go func() {
		log.Printf("Notification gateway starting on port %s", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpSrv.Shutdown(shutdownCtx)
}

func loadConfig() *Config {
	return &Config{
		Port:      getEnv("NOTIFICATION_PORT", "9002"),
		JWTSecret: getEnv("JWT_SECRET", ""),
		Database: DatabaseConfig{
			Host:     getEnv("NOTIFICATION_DB_HOST", "localhost"),
			Port:     getEnvInt("NOTIFICATION_DB_PORT", 5432),
			User:     getEnv("NOTIFICATION_DB_USER", "tigerwallet"),
			Password: getEnv("NOTIFICATION_DB_PASSWORD", "password"),
			DBName:   getEnv("NOTIFICATION_DB_NAME", "tigerwallet_notification"),
			URL:      os.Getenv("DATABASE_URL"),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
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
			ProjectID:   getEnv("FCM_PROJECT_ID", ""),
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
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultValue
}

// ---------- Database ----------

func initDatabase(ctx context.Context, cfg *Config) (*pgxpool.Pool, error) {
	connStr := cfg.Database.URL
	if connStr == "" {
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			url.QueryEscape(cfg.Database.User), url.QueryEscape(cfg.Database.Password),
			cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	}
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres unreachable: %w", err)
	}
	return pool, nil
}

func migrate(ctx context.Context, db *pgxpool.Pool) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS gateway_notifications (
			id UUID PRIMARY KEY,
			user_id TEXT NOT NULL,
			type TEXT NOT NULL,
			channel TEXT DEFAULT '',
			template_id TEXT DEFAULT '',
			subject TEXT DEFAULT '',
			body TEXT NOT NULL,
			recipients JSONB DEFAULT '[]',
			status TEXT NOT NULL DEFAULT 'pending',
			error_msg TEXT DEFAULT '',
			metadata JSONB DEFAULT '{}',
			sent_at TIMESTAMPTZ,
			delivered_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_gateway_notifications_user ON gateway_notifications(user_id)`,
		`CREATE TABLE IF NOT EXISTS gateway_templates (
			id UUID PRIMARY KEY,
			tenant_id TEXT DEFAULT '',
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			subject TEXT DEFAULT '',
			body TEXT DEFAULT '',
			html_body TEXT DEFAULT '',
			variables JSONB DEFAULT '[]',
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS gateway_preferences (
			user_id TEXT PRIMARY KEY,
			email_enabled BOOLEAN NOT NULL DEFAULT true,
			sms_enabled BOOLEAN NOT NULL DEFAULT true,
			push_enabled BOOLEAN NOT NULL DEFAULT true,
			in_app_enabled BOOLEAN NOT NULL DEFAULT true,
			email_frequency TEXT NOT NULL DEFAULT 'immediate',
			marketing_opt_in BOOLEAN NOT NULL DEFAULT false,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS gateway_subscribers (
			id UUID PRIMARY KEY,
			email TEXT DEFAULT '',
			phone TEXT DEFAULT '',
			device_token TEXT DEFAULT '',
			tenant_id TEXT DEFAULT '',
			is_active BOOLEAN NOT NULL DEFAULT true,
			subscribed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			unsubscribed_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS gateway_channels (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			config JSONB DEFAULT '{}',
			rate_limit INT NOT NULL DEFAULT 100,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`INSERT INTO gateway_channels (id, name, type, status, rate_limit) VALUES
			($1, 'Primary Email', 'email', 'active', 100),
			($2, 'SMS Gateway', 'sms', 'active', 50),
			($3, 'Push Notifications', 'push', 'active', 1000)
		 ON CONFLICT (name) DO NOTHING`,
	}
	for i, stmt := range stmts {
		var err error
		if i == len(stmts)-1 {
			_, err = db.Exec(ctx, stmt, uuid.New(), uuid.New(), uuid.New())
		} else {
			_, err = db.Exec(ctx, stmt)
		}
		if err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}
	return nil
}

// ---------- Auth ----------

// jwtAuthMiddleware verifies HS256 JWTs. Fail-closed: without JWT_SECRET no
// request passes, and tokens are never accepted without a valid signature.
func (s *Server) jwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.cfg.JWTSecret == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "auth not configured"})
			c.Abort()
			return
		}
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}
		claims, err := verifyHS256JWT(strings.TrimPrefix(authHeader, "Bearer "), s.cfg.JWTSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		userID, _ := claims["sub"].(string)
		if userID == "" {
			userID, _ = claims["user_id"].(string)
		}
		c.Set("user_id", userID)
		c.Next()
	}
}

func verifyHS256JWT(token, secret string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed token")
	}
	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errors.New("bad signature encoding")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, errors.New("signature mismatch")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var header map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, err
	}
	if alg, _ := header["alg"].(string); alg != "HS256" {
		return nil, errors.New("unsupported alg")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, err
	}
	if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		return nil, errors.New("token expired")
	}
	return claims, nil
}

// ---------- Channel senders (real provider calls, fail closed) ----------

func (s *Server) sendEmail(to []string, subject, body, htmlBody string) error {
	if s.cfg.SMTP.Host == "" || s.cfg.SMTP.Username == "" {
		return errors.New("SMTP not configured: set SMTP_HOST and SMTP_USERNAME")
	}
	boundary := "tiger-" + uuid.NewString()
	var msg strings.Builder
	from := s.cfg.SMTP.From
	if s.cfg.SMTP.FromName != "" {
		from = fmt.Sprintf("%s <%s>", s.cfg.SMTP.FromName, s.cfg.SMTP.From)
	}
	msg.WriteString("From: " + from + "\r\n")
	msg.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	if htmlBody != "" {
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n\r\n", boundary))
		msg.WriteString(fmt.Sprintf("--%s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n", boundary, body))
		msg.WriteString(fmt.Sprintf("--%s\r\nContent-Type: text/html; charset=utf-8\r\n\r\n%s\r\n", boundary, htmlBody))
		msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n" + body)
	}
	auth := smtp.PlainAuth("", s.cfg.SMTP.Username, s.cfg.SMTP.Password, s.cfg.SMTP.Host)
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTP.Host, s.cfg.SMTP.Port)
	return smtp.SendMail(addr, auth, s.cfg.SMTP.From, to, []byte(msg.String()))
}

func (s *Server) sendSMS(to []string, message string) error {
	if s.cfg.Twilio.AccountSID == "" || s.cfg.Twilio.AuthToken == "" || s.cfg.Twilio.PhoneNumber == "" {
		return errors.New("Twilio not configured: set TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_PHONE_NUMBER")
	}
	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", s.cfg.Twilio.AccountSID)
	client := &http.Client{Timeout: 15 * time.Second}
	for _, recipient := range to {
		form := url.Values{}
		form.Set("From", s.cfg.Twilio.PhoneNumber)
		form.Set("To", recipient)
		form.Set("Body", message)
		req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		if err != nil {
			return err
		}
		req.SetBasicAuth(s.cfg.Twilio.AccountSID, s.cfg.Twilio.AuthToken)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("twilio request failed: %w", err)
		}
		respBody := make([]byte, 4096)
		n, _ := resp.Body.Read(respBody)
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("twilio error %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody[:n])))
		}
	}
	return nil
}

func (s *Server) sendPushNotification(tokens []string, title, body string, data map[string]interface{}) error {
	if s.fcm == nil {
		return errors.New("FCM not configured: set FCM_CREDENTIALS")
	}
	stringData := map[string]string{}
	for k, v := range data {
		stringData[k] = fmt.Sprintf("%v", v)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, token := range tokens {
		_, err := s.fcm.Send(ctx, &messaging.Message{
			Token:        token,
			Notification: &messaging.Notification{Title: title, Body: body},
			Data:         stringData,
		})
		if err != nil {
			return fmt.Errorf("fcm send failed for token %.12s...: %w", token, err)
		}
	}
	return nil
}

// ---------- Handlers ----------

func (s *Server) recordNotification(ctx context.Context, userID, ntype, templateID, subject, body string, recipients []string, metadata map[string]interface{}) (uuid.UUID, error) {
	id := uuid.New()
	recipientsJSON, _ := json.Marshal(recipients)
	metadataJSON, _ := json.Marshal(metadata)
	_, err := s.db.Exec(ctx,
		`INSERT INTO gateway_notifications (id, user_id, type, channel, template_id, subject, body, recipients, status, metadata)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending',$9)`,
		id, userID, ntype, ntype, templateID, subject, body, recipientsJSON, metadataJSON)
	return id, err
}

func (s *Server) markNotification(ctx context.Context, id uuid.UUID, status, errMsg string) {
	if status == "delivered" {
		s.db.Exec(ctx, `UPDATE gateway_notifications SET status=$1, error_msg=$2, delivered_at=now() WHERE id=$3`, status, errMsg, id)
		return
	}
	s.db.Exec(ctx, `UPDATE gateway_notifications SET status=$1, error_msg=$2, sent_at=CASE WHEN $1='sent' THEN now() ELSE sent_at END WHERE id=$3`, status, errMsg, id)
}

func (s *Server) sendNotificationHandler(c *gin.Context) {
	var req struct {
		UserID     string                 `json:"user_id" binding:"required"`
		Type       string                 `json:"type" binding:"required"`
		TemplateID string                 `json:"template_id"`
		Subject    string                 `json:"subject"`
		Body       string                 `json:"body" binding:"required"`
		Recipients []string               `json:"recipients"`
		Metadata   map[string]interface{} `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	id, err := s.recordNotification(ctx, req.UserID, req.Type, req.TemplateID, req.Subject, req.Body, req.Recipients, req.Metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var sendErr error
	switch req.Type {
	case "email":
		sendErr = s.sendEmail(req.Recipients, req.Subject, req.Body, "")
	case "sms":
		sendErr = s.sendSMS(req.Recipients, req.Body)
	case "push":
		sendErr = s.sendPushNotification(req.Recipients, req.Subject, req.Body, req.Metadata)
	case "in_app":
		// persisted above; delivered via /history polling
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported type"})
		return
	}
	if sendErr != nil {
		s.markNotification(ctx, id, "failed", sendErr.Error())
		c.JSON(http.StatusBadGateway, gin.H{"error": sendErr.Error(), "notification_id": id})
		return
	}
	s.markNotification(ctx, id, "sent", "")
	c.JSON(http.StatusOK, gin.H{"notification_id": id, "status": "sent"})
}

func (s *Server) sendBatchNotificationsHandler(c *gin.Context) {
	var req struct {
		Notifications []map[string]interface{} `json:"notifications" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	sent, failed := 0, 0
	results := make([]gin.H, 0, len(req.Notifications))
	for _, notif := range req.Notifications {
		userID, _ := notif["user_id"].(string)
		ntype, _ := notif["type"].(string)
		subject, _ := notif["subject"].(string)
		body, _ := notif["body"].(string)
		recipients := toStringSlice(notif["recipients"])
		if userID == "" || ntype == "" || body == "" {
			failed++
			results = append(results, gin.H{"status": "failed", "error": "user_id, type and body are required"})
			continue
		}
		id, err := s.recordNotification(ctx, userID, ntype, "", subject, body, recipients, nil)
		if err != nil {
			failed++
			results = append(results, gin.H{"status": "failed", "error": err.Error()})
			continue
		}
		var sendErr error
		switch ntype {
		case "email":
			sendErr = s.sendEmail(recipients, subject, body, "")
		case "sms":
			sendErr = s.sendSMS(recipients, body)
		case "push":
			sendErr = s.sendPushNotification(recipients, subject, body, nil)
		case "in_app":
		default:
			sendErr = errors.New("unsupported type")
		}
		if sendErr != nil {
			failed++
			s.markNotification(ctx, id, "failed", sendErr.Error())
			results = append(results, gin.H{"notification_id": id, "status": "failed", "error": sendErr.Error()})
			continue
		}
		sent++
		s.markNotification(ctx, id, "sent", "")
		results = append(results, gin.H{"notification_id": id, "status": "sent"})
	}
	c.JSON(http.StatusOK, gin.H{"sent": sent, "failed": failed, "total": len(req.Notifications), "results": results})
}

func toStringSlice(v interface{}) []string {
	out := []string{}
	if arr, ok := v.([]interface{}); ok {
		for _, item := range arr {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func (s *Server) sendEmailHandler(c *gin.Context) {
	var req struct {
		To         []string               `json:"to" binding:"required"`
		Subject    string                 `json:"subject" binding:"required"`
		Body       string                 `json:"body"`
		HTMLBody   string                 `json:"html_body"`
		TemplateID string                 `json:"template_id"`
		Variables  map[string]interface{} `json:"variables"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	subject, body, htmlBody := req.Subject, req.Body, req.HTMLBody
	if req.TemplateID != "" {
		var tSubject, tBody, tHTML string
		err := s.db.QueryRow(ctx,
			`SELECT subject, body, html_body FROM gateway_templates WHERE id=$1 AND is_active=true`,
			req.TemplateID).Scan(&tSubject, &tBody, &tHTML)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		subject = interpolateTemplate(tSubject, req.Variables)
		body = interpolateTemplate(tBody, req.Variables)
		htmlBody = interpolateTemplate(tHTML, req.Variables)
	}
	if err := s.sendEmail(req.To, subject, body, htmlBody); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Email sent", "recipients": len(req.To)})
}

func (s *Server) sendSMSHandler(c *gin.Context) {
	var req struct {
		To        []string               `json:"to" binding:"required"`
		Message   string                 `json:"message" binding:"required"`
		Template  string                 `json:"template"`
		Variables map[string]interface{} `json:"variables"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	message := req.Message
	if req.Template != "" {
		var tBody string
		err := s.db.QueryRow(c.Request.Context(),
			`SELECT body FROM gateway_templates WHERE id=$1 AND type='sms' AND is_active=true`,
			req.Template).Scan(&tBody)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		message = interpolateTemplate(tBody, req.Variables)
	}
	if err := s.sendSMS(req.To, message); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "SMS sent", "recipients": len(req.To)})
}

func (s *Server) sendPushHandler(c *gin.Context) {
	var req struct {
		DeviceTokens []string               `json:"device_tokens" binding:"required"`
		Title        string                 `json:"title" binding:"required"`
		Body         string                 `json:"body" binding:"required"`
		Data         map[string]interface{} `json:"data"`
		ImageURL     string                 `json:"image_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.sendPushNotification(req.DeviceTokens, req.Title, req.Body, req.Data); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Push notification sent", "recipients": len(req.DeviceTokens)})
}

func (s *Server) subscribeHandler(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		DeviceToken string `json:"device_token"`
		TenantID    string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Email == "" && req.Phone == "" && req.DeviceToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email, phone or device_token required"})
		return
	}
	id := uuid.New()
	_, err := s.db.Exec(c.Request.Context(),
		`INSERT INTO gateway_subscribers (id, email, phone, device_token, tenant_id) VALUES ($1,$2,$3,$4,$5)`,
		id, req.Email, req.Phone, req.DeviceToken, req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"subscriber_id": id, "message": "Subscribed successfully"})
}

func (s *Server) unsubscribeHandler(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		DeviceToken string `json:"device_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := s.db.Exec(c.Request.Context(),
		`UPDATE gateway_subscribers SET is_active=false, unsubscribed_at=now()
		 WHERE (email=$1 AND $1<>'') OR (phone=$2 AND $2<>'') OR (device_token=$3 AND $3<>'')`,
		req.Email, req.Phone, req.DeviceToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscriber not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribed successfully"})
}

func (s *Server) listTemplatesHandler(c *gin.Context) {
	rows, err := s.db.Query(c.Request.Context(),
		`SELECT id, name, type, subject, is_active, created_at FROM gateway_templates ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	templates := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, ttype, subject string
		var active bool
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &ttype, &subject, &active, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		templates = append(templates, gin.H{"id": id, "name": name, "type": ttype, "subject": subject, "is_active": active, "created_at": createdAt.Unix()})
	}
	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

func (s *Server) createTemplateHandler(c *gin.Context) {
	var req struct {
		Name      string   `json:"name" binding:"required"`
		Type      string   `json:"type" binding:"required"`
		Subject   string   `json:"subject"`
		Body      string   `json:"body"`
		HTMLBody  string   `json:"html_body"`
		Variables []string `json:"variables"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := uuid.New()
	varsJSON, _ := json.Marshal(req.Variables)
	_, err := s.db.Exec(c.Request.Context(),
		`INSERT INTO gateway_templates (id, name, type, subject, body, html_body, variables) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, req.Name, req.Type, req.Subject, req.Body, req.HTMLBody, varsJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"template_id": id})
}

func (s *Server) updateTemplateHandler(c *gin.Context) {
	templateID := c.Param("id")
	var req struct {
		Name      string   `json:"name"`
		Subject   string   `json:"subject"`
		Body      string   `json:"body"`
		HTMLBody  string   `json:"html_body"`
		Variables []string `json:"variables"`
		IsActive  *bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	varsJSON, _ := json.Marshal(req.Variables)
	res, err := s.db.Exec(c.Request.Context(),
		`UPDATE gateway_templates SET
			name = COALESCE(NULLIF($2,''), name),
			subject = COALESCE(NULLIF($3,''), subject),
			body = COALESCE(NULLIF($4,''), body),
			html_body = COALESCE(NULLIF($5,''), html_body),
			variables = COALESCE($6, variables),
			is_active = COALESCE($7, is_active),
			updated_at = now()
		 WHERE id=$1`,
		templateID, req.Name, req.Subject, req.Body, req.HTMLBody, varsJSON, req.IsActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Template updated"})
}

func (s *Server) deleteTemplateHandler(c *gin.Context) {
	templateID := c.Param("id")
	res, err := s.db.Exec(c.Request.Context(), `DELETE FROM gateway_templates WHERE id=$1`, templateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Template deleted"})
}

func (s *Server) getNotificationHistoryHandler(c *gin.Context) {
	userID := c.Query("user_id")
	notifType := c.Query("type")
	status := c.Query("status")
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	query := `SELECT id, user_id, type, subject, body, status, sent_at, delivered_at, created_at
	          FROM gateway_notifications WHERE ($1='' OR user_id=$1) AND ($2='' OR type=$2) AND ($3='' OR status=$3)
	          ORDER BY created_at DESC LIMIT $4 OFFSET $5`
	rows, err := s.db.Query(c.Request.Context(), query, userID, notifType, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	notifications := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var uid, ntype, subject, body, nstatus string
		var sentAt, deliveredAt, createdAt *time.Time
		if err := rows.Scan(&id, &uid, &ntype, &subject, &body, &nstatus, &sentAt, &deliveredAt, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		notifications = append(notifications, gin.H{
			"id": id, "user_id": uid, "type": ntype, "subject": subject, "body": body,
			"status": nstatus, "sent_at": sentAt, "delivered_at": deliveredAt, "created_at": createdAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

func (s *Server) getNotificationByIDHandler(c *gin.Context) {
	notifID := c.Param("id")
	var id uuid.UUID
	var uid, ntype, subject, body, nstatus, errMsg string
	var sentAt, deliveredAt, createdAt *time.Time
	err := s.db.QueryRow(c.Request.Context(),
		`SELECT id, user_id, type, subject, body, status, error_msg, sent_at, delivered_at, created_at
		 FROM gateway_notifications WHERE id=$1`, notifID).
		Scan(&id, &uid, &ntype, &subject, &body, &nstatus, &errMsg, &sentAt, &deliveredAt, &createdAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": id, "user_id": uid, "type": ntype, "subject": subject, "body": body,
		"status": nstatus, "error": errMsg, "sent_at": sentAt, "delivered_at": deliveredAt, "created_at": createdAt,
	})
}

func (s *Server) getNotificationStatsHandler(c *gin.Context) {
	var totalSent, totalDelivered, totalFailed int64
	s.db.QueryRow(c.Request.Context(), `SELECT
		COUNT(*) FILTER (WHERE status='sent'),
		COUNT(*) FILTER (WHERE status='delivered'),
		COUNT(*) FILTER (WHERE status='failed')
		FROM gateway_notifications`).Scan(&totalSent, &totalDelivered, &totalFailed)
	byType := map[string]int64{}
	rows, err := s.db.Query(c.Request.Context(), `SELECT type, COUNT(*) FROM gateway_notifications GROUP BY type`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var n int64
			if rows.Scan(&t, &n) == nil {
				byType[t] = n
			}
		}
	}
	deliveryRate := 0.0
	if totalSent+totalDelivered+totalFailed > 0 {
		deliveryRate = float64(totalDelivered) / float64(totalSent+totalDelivered+totalFailed) * 100
	}
	c.JSON(http.StatusOK, gin.H{
		"total_sent": totalSent, "total_delivered": totalDelivered, "total_failed": totalFailed,
		"by_type": byType, "delivery_rate": deliveryRate,
	})
}

func (s *Server) getPreferencesHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	var emailEnabled, smsEnabled, pushEnabled, inAppEnabled, marketingOptIn bool
	var frequency string
	var updatedAt time.Time
	err := s.db.QueryRow(c.Request.Context(),
		`SELECT email_enabled, sms_enabled, push_enabled, in_app_enabled, email_frequency, marketing_opt_in, updated_at
		 FROM gateway_preferences WHERE user_id=$1`, userID).
		Scan(&emailEnabled, &smsEnabled, &pushEnabled, &inAppEnabled, &frequency, &marketingOptIn, &updatedAt)
	if err != nil {
		// No row yet: return defaults without persisting.
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID, "email_enabled": true, "sms_enabled": true, "push_enabled": true,
			"in_app_enabled": true, "email_frequency": "immediate", "marketing_opt_in": false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID, "email_enabled": emailEnabled, "sms_enabled": smsEnabled,
		"push_enabled": pushEnabled, "in_app_enabled": inAppEnabled,
		"email_frequency": frequency, "marketing_opt_in": marketingOptIn, "updated_at": updatedAt,
	})
}

func (s *Server) updatePreferencesHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	var req struct {
		EmailEnabled   *bool  `json:"email_enabled"`
		SMSEnabled     *bool  `json:"sms_enabled"`
		PushEnabled    *bool  `json:"push_enabled"`
		InAppEnabled   *bool  `json:"in_app_enabled"`
		EmailFrequency string `json:"email_frequency"`
		MarketingOptIn *bool  `json:"marketing_opt_in"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := s.db.Exec(c.Request.Context(),
		`INSERT INTO gateway_preferences (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_, err = s.db.Exec(c.Request.Context(),
		`UPDATE gateway_preferences SET
			email_enabled = COALESCE($2, email_enabled),
			sms_enabled = COALESCE($3, sms_enabled),
			push_enabled = COALESCE($4, push_enabled),
			in_app_enabled = COALESCE($5, in_app_enabled),
			email_frequency = COALESCE(NULLIF($6,''), email_frequency),
			marketing_opt_in = COALESCE($7, marketing_opt_in),
			updated_at = now()
		 WHERE user_id=$1`,
		userID, req.EmailEnabled, req.SMSEnabled, req.PushEnabled, req.InAppEnabled, req.EmailFrequency, req.MarketingOptIn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Preferences updated"})
}

func (s *Server) getChannelsHandler(c *gin.Context) {
	rows, err := s.db.Query(c.Request.Context(),
		`SELECT id, name, type, status, rate_limit FROM gateway_channels ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	channels := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, ttype, status string
		var rateLimit int
		if err := rows.Scan(&id, &name, &ttype, &status, &rateLimit); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		channels = append(channels, gin.H{"id": id, "name": name, "type": ttype, "status": status, "rate_limit": rateLimit})
	}
	c.JSON(http.StatusOK, gin.H{"channels": channels})
}

func (s *Server) updateChannelStatusHandler(c *gin.Context) {
	channelID := c.Param("channel")
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status != "active" && req.Status != "inactive" && req.Status != "error" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be active, inactive or error"})
		return
	}
	res, err := s.db.Exec(c.Request.Context(),
		`UPDATE gateway_channels SET status=$2 WHERE id::text=$1 OR name=$1`, channelID, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Channel status updated"})
}

func (s *Server) emailStatusWebhookHandler(c *gin.Context)   { s.statusWebhook(c) }
func (s *Server) smsStatusWebhookHandler(c *gin.Context)     { s.statusWebhook(c) }
func (s *Server) pushStatusWebhookHandler(c *gin.Context)    { s.statusWebhook(c) }

func (s *Server) statusWebhook(c *gin.Context) {
	var req struct {
		MessageID string `json:"message_id" binding:"required"`
		Status    string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	allowed := map[string]bool{"sent": true, "delivered": true, "bounced": true, "failed": true, "queued": true}
	if !allowed[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown status"})
		return
	}
	id, err := uuid.Parse(req.MessageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message_id"})
		return
	}
	s.markNotification(c.Request.Context(), id, req.Status, "")
	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}

func interpolateTemplate(template string, vars map[string]interface{}) string {
	result := template
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", fmt.Sprintf("%v", v))
	}
	return result
}

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
