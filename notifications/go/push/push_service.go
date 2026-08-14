// Push Notification Service - FCM & APNS Implementation
// Cross-platform push notifications for iOS, Android, and Web

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PushConfig - Push Notification Configuration
type PushConfig struct {
	// FCM Settings
	FCMProjectID   string `json:"fcm_project_id"`
	FCMCredentials string `json:"fcm_credentials"` // Base64 encoded JSON
	
	// APNS Settings
	APNSKeyPath    string `json:"apns_key_path"`
	APNSKeyID     string `json:"apns_key_id"`
	APNSTeamID    string `json:"apns_team_id"`
	APNSBundleID  string `json:"apns_bundle_id"`
	UseProduction bool   `json:"use_production"`
	
	// Queue Settings
	MaxRetries      int           `json:"max_retries"`
	RetryDelay      time.Duration `json:"retry_delay"`
	WorkerCount     int           `json:"worker_count"`
	QueueBufferSize int           `json:"queue_buffer_size"`
	
	// Rate Limiting
	RateLimitPerMinute int `json:"rate_limit_per_minute"`
	
	// Database Settings
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	
	// Redis Settings
	RedisHost string `json:"redis_host"`
	RedisPort string `json:"redis_port"`
	
	// Server
	ServerPort string `json:"server_port"`
}

// DeviceToken - Device token for push notifications
type DeviceToken struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"index" json:"user_id"`
	Token       string    `gorm:"uniqueIndex" json:"token"`
	Platform    string    `json:"platform"` // ios, android, web
	AppVersion  string    `json:"app_version"`
	Language    string    `json:"language"`
	Timezone    string    `json:"timezone"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PushTemplate - Push notification template
type PushTemplate struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TemplateID  string    `gorm:"uniqueIndex" json:"template_id"`
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Body        string    `gorm:"type:text" json:"body"`
	Data        string    `gorm:"type:jsonb" json:"data"`
	Platform    string    `json:"platform"` // all, ios, android
	Sound       string    `json:"sound"`
	Badge       int       `json:"badge"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PushNotification - Push notification
type PushNotification struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	MessageID   string    `gorm:"uniqueIndex" json:"message_id"`
	UserID      string    `gorm:"index" json:"user_id"`
	Token       string    `gorm:"index" json:"token"`
	Platform    string    `json:"platform"`
	Title       string    `json:"title"`
	Body        string    `gorm:"type:text" json:"body"`
	Data        string    `gorm:"type:jsonb" json:"data"`
	Sound       string    `json:"sound"`
	Badge       int       `json:"badge"`
	Priority    string    `json:"priority"` // high, normal
	Status      string    `gorm:"index" json:"status"` // queued, sending, sent, failed
	RetryCount  int       `gorm:"default:0" json:"retry_count"`
	LastError   string    `gorm:"type:text" json:"last_error"`
	SentAt      *time.Time `json:"sent_at"`
	CreatedAt   time.Time `json:"created_at"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

// PushLog - Push notification log
type PushLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	MessageID   string    `gorm:"uniqueIndex;index" json:"message_id"`
	UserID      string    `gorm:"index" json:"user_id"`
	Token       string    `gorm:"index" json:"token"`
	Platform    string    `json:"platform"`
	Title       string    `json:"title"`
	Status      string    `json:"status"` // sent, delivered, opened, failed
	SentAt      time.Time `json:"sent_at"`
	DeliveredAt *time.Time `json:"delivered_at"`
	OpenedAt    *time.Time `json:"opened_at"`
	Error       string    `gorm:"type:text" json:"error"`
	Provider    string    `json:"provider"`
	CreatedAt   time.Time `json:"created_at"`
}

// FCMToken - Firebase Cloud Messaging token
type FCMToken struct {
	Token         string `json:"token"`
	CollapseKey   string `json:"collapse_key,omitempty"`
	Data          map[string]string `json:"data,omitempty"`
	Notification  *FCMNotification `json:"notification,omitempty"`
	Android       *FCMAndroidConfig `json:"android,omitempty"`
	APNS         *FCMAPNSConfig `json:"apns,omitempty"`
	Webpush      *FCMWebpushConfig `json:"webpush,omitempty"`
	Priority     string `json:"priority,omitempty"`
	TimeToLive   int64  `json:"time_to_live,omitempty"`
}

// FCMNotification - FCM notification payload
type FCMNotification struct {
	Title        string `json:"title"`
	Body         string `json:"body,omitempty"`
	Icon         string `json:"icon,omitempty"`
	Sound        string `json:"sound,omitempty"`
	Badge        string `json:"badge,omitempty"`
	Tag          string `json:"tag,omitempty"`
	ClickAction  string `json:"click_action,omitempty"`
}

// FCMAndroidConfig - FCM Android configuration
type FCMAndroidConfig struct {
	CollapseKey      string            `json:"collapse_key,omitempty"`
	Priority        string            `json:"priority,omitempty"`
	TimeToLive      int64             `json:"time_to_live,omitempty"`
	RestrictedPackageName string       `json:"restricted_package_name,omitempty"`
	Data            map[string]string `json:"data,omitempty"`
	Notification    *FCMAndroidNotification `json:"notification,omitempty"`
}

// FCMAndroidNotification - FCM Android notification
type FCMAndroidNotification struct {
	Title        string `json:"title"`
	Body         string `json:"body,omitempty"`
	Icon         string `json:"icon,omitempty"`
	Sound        string `json:"sound,omitempty"`
	Tag          string `json:"tag,omitempty"`
	ClickAction  string `json:"click_action,omitempty"`
	Color        string `json:"color,omitempty"`
}

// FCMAPNSConfig - FCM APNS configuration
type FCMAPNSConfig struct {
	Headers    map[string]string `json:"headers,omitempty"`
	Payload    *FCMAPNSPayload `json:"payload,omitempty"`
}

// FCMAPNSPayload - FCM APNS payload
type FCMAPNSPayload struct {
	Aps *FCMAPSAps `json:"aps,omitempty"`
}

// FCMAPSAps - APNS payload
type FCMAPSAps struct {
	Alert       interface{} `json:"alert,omitempty"` // string or map
	Badge       int         `json:"badge,omitempty"`
	Sound       string      `json:"sound,omitempty"`
	Category    string      `json:"category,omitempty"`
	ThreadID    string      `json:"thread-id,omitempty"`
	MutableContent int      `json:"mutable-content,omitempty"`
}

// FCMWebpushConfig - FCM WebPush configuration
type FCMWebpushConfig struct {
	Headers map[string]string `json:"headers,omitempty"`
	Data   map[string]string `json:"data,omitempty"`
	Notification *FCMWebpushNotification `json:"notification,omitempty"`
}

// FCMWebpushNotification - WebPush notification
type FCMWebpushNotification struct {
	Title    string `json:"title"`
	Body     string `json:"body,omitempty"`
	Icon     string `json:"icon,omitempty"`
	Badge    string `json:"badge,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	Actions  interface{} `json:"actions,omitempty"`
	Dir      string `json:"dir,omitempty"`
	Lang     string `json:"lang,omitempty"`
	Renotify bool   `json:"renotify,omitempty"`
	RequireInteraction bool `json:"require_interaction,omitempty"`
}

// FCMResponse - FCM response
type FCMResponse struct {
	MessageID      string              `json:"message_id"`
	MulticastID    int64               `json:"multicast_id"`
	Success        int                 `json:"success"`
	Failure        int                 `json:"failure"`
	CanonicalIDs   int                 `json:"canonical_ids"`
	Results       []FCMResult         `json:"results"`
	Error          string              `json:"error,omitempty"`
}

// FCMResult - FCM single message result
type FCMResult struct {
	MessageID       string `json:"message_id,omitempty"`
	Error           string `json:"error,omitempty"`
	CanonicalToken  string `json:"canonical_token_id,omitempty"`
}

// APNSNotification - APNS notification
type APNSNotification struct {
	Aps          *APNSAps          `json:"aps"`
	CustomData   map[string]interface{} `json:"-"`
}

// APNSAps - APNS payload
type APNSAps struct {
	Alert            *APNSAlert       `json:"alert,omitempty"`
	Badge            int              `json:"badge,omitempty"`
	Sound            string           `json:"sound,omitempty"`
	Category         string           `json:"category,omitempty"`
	ThreadID         string           `json:"thread-id,omitempty"`
	MutableContent   int              `json:"mutable-content,omitempty"`
	ContentAvailable int              `json:"content-available,omitempty"`
}

// APNSAlert - APNS alert
type APNSAlert struct {
	Title           string   `json:"title,omitempty"`
	TitleLocKey    string   `json:"title-loc-key,omitempty"`
	TitleLocArgs   []string `json:"title-loc-args,omitempty"`
	Body           string   `json:"body,omitempty"`
	LocKey         string   `json:"loc-key,omitempty"`
	LocArgs        []string `json:"loc-args,omitempty"`
	LaunchImage    string   `json:"launch-image,omitempty"`
	Action         string   `json:"action,omitempty"`
	ActionLocKey   string   `json:"action-loc-key,omitempty"`
}

// PushService - Main push notification service
type PushService struct {
	config     PushConfig
	db         *gorm.DB
	redis      *redis.Client
	fcmClient  *http.Client
	apnsClient *http.Client
	queue      chan *PushNotification
	workers    sync.WaitGroup
	stopCh     chan struct{}
	fcmToken   string
	rateLimiter *RateLimiter
}

// NewPushService - Create new push notification service
func NewPushService(cfg PushConfig) (*PushService, error) {
	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Auto migrate
	err = db.AutoMigrate(&DeviceToken{}, &PushTemplate{}, &PushNotification{}, &PushLog{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	
	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
	})
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	
	// Create FCM client
	fcmClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Create APNS client
	apnsClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Get FCM token
	fcmToken := ""
	if cfg.FCMCredentials != "" {
		fcmToken, err = getFCMAccessToken(cfg.FCMCredentials)
		if err != nil {
			log.Printf("Warning: Failed to get FCM token: %v", err)
		}
	}
	
	// Queue buffer
	queueSize := cfg.QueueBufferSize
	if queueSize == 0 {
		queueSize = 1000
	}
	
	workerCount := cfg.WorkerCount
	if workerCount == 0 {
		workerCount = 5
	}
	
	rateLimit := cfg.RateLimitPerMinute
	if rateLimit == 0 {
		rateLimit = 1000
	}
	
	service := &PushService{
		config:      cfg,
		db:          db,
		redis:       rdb,
		fcmClient:   fcmClient,
		apnsClient:  apnsClient,
		queue:       make(chan *PushNotification, queueSize),
		stopCh:      make(chan struct{}),
		fcmToken:    fcmToken,
		rateLimiter: NewRateLimiter(rateLimit),
	}
	
	// Seed default templates
	service.seedDefaultTemplates()
	
	return service, nil
}

// seedDefaultTemplates - Seed default push templates
func (s *PushService) seedDefaultTemplates() {
	defaultTemplates := []PushTemplate{
		{
			TemplateID: "deposit_received",
			Name:       "Deposit Received",
			Title:      "Deposit Confirmed",
			Body:       "You received {{.Amount}} {{.Currency}}",
			Data:       `{"type": "deposit", "tx_hash": "{{.TxHash}}"}`,
			Platform:   "all",
			Sound:      "default",
			Badge:      1,
			IsActive:   true,
		},
		{
			TemplateID: "withdrawal_initiated",
			Name:       "Withdrawal Initiated",
			Title:      "Withdrawal Sent",
			Body:       "{{.Amount}} {{.Currency}} sent to {{.Address}}",
			Data:       `{"type": "withdrawal", "tx_hash": "{{.TxHash}}"}`,
			Platform:   "all",
			Sound:      "default",
			Badge:      1,
			IsActive:   true,
		},
		{
			TemplateID: "trade_executed",
			Name:       "Trade Executed",
			Title:      "Trade Executed",
			Body:       "Your {{.Side}} order for {{.Amount}} {{.Token}} was executed at {{.Price}}",
			Data:       `{"type": "trade"}`,
			Platform:   "all",
			Sound:      "default",
			Badge:      1,
			IsActive:   true,
		},
		{
			TemplateID: "price_alert",
			Name:       "Price Alert",
			Title:      "Price Alert: {{.Token}}",
			Body:       "{{.Token}} has reached {{.Price}}",
			Data:       `{"type": "price_alert"}`,
			Platform:   "all",
			Sound:      "default",
			Badge:      1,
			IsActive:   true,
		},
		{
			TemplateID: "security_alert",
			Name:       "Security Alert",
			Title:      "Security Alert",
			Body:       "{{.AlertType}} detected on your account",
			Data:       `{"type": "security"}`,
			Platform:   "all",
			Sound:      "default",
			Badge:      1,
			IsActive:   true,
		},
		{
			TemplateID: "kyc_update",
			Name:       "KYC Update",
			Title:      "Identity Verification",
			Body:       "Your KYC status has been updated to {{.Status}}",
			Data:       `{"type": "kyc"}`,
			Platform:   "all",
			Sound:      "default",
			IsActive:   true,
		},
	}
	
	for _, tmpl := range defaultTemplates {
		var existing PushTemplate
		result := s.db.Where("template_id = ?", tmpl.TemplateID).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			s.db.Create(&tmpl)
		}
	}
}

// getFCMAccessToken - Get FCM access token
func getFCMAccessToken(credentialsBase64 string) (string, error) {
	// Decode credentials
	credentialsJSON, err := base64.StdEncoding.DecodeString(credentialsBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode credentials: %w", err)
	}
	
	// Parse credentials
	var creds map[string]interface{}
	if err := json.Unmarshal(credentialsJSON, &creds); err != nil {
		return "", fmt.Errorf("failed to parse credentials: %w", err)
	}
	
	// In production, use Google OAuth2 to get access token
	// For now, return a placeholder
	return "fcm_access_token_placeholder", nil
}

// GenerateMessageID - Generate unique message ID
func (s *PushService) GenerateMessageID() string {
	return fmt.Sprintf("push_%d_%s", time.Now().UnixNano(), randomString(8))
}

// SendPush - Send push notification
func (s *PushService) SendPush(userID, token, platform, title, body, data string, sound string, badge int) (string, error) {
	// Check rate limit
	if !s.rateLimiter.Acquire() {
		return "", fmt.Errorf("rate limit exceeded")
	}
	
	messageID := s.GenerateMessageID()
	
	var err error
	var provider string
	
	switch platform {
	case "ios", "apple":
		err = s.sendAPNS(token, title, body, data, sound, badge)
		provider = "apns"
	case "android":
		err = s.sendFCM(token, title, body, data, sound, badge)
		provider = "fcm"
	case "web":
		err = s.sendWebPush(token, title, body, data, sound, badge)
		provider = "fcm_web"
	default:
		// Try all platforms
		err = s.sendFCM(token, title, body, data, sound, badge)
		provider = "fcm"
	}
	
	// Log
	s.logPush(messageID, userID, token, platform, title, provider, err)
	
	if err != nil {
		return "", err
	}
	
	// Update device last used
	s.db.Model(&DeviceToken{}).Where("token = ?", token).Update("last_used_at", time.Now())
	
	return messageID, nil
}

// sendFCM - Send via FCM
func (s *PushService) sendFCM(token, title, body, data, sound string, badge int) error {
	fcmToken := FCMToken{
		Token:    token,
		Priority: "high",
		Notification: &FCMNotification{
			Title: title,
			Body:  body,
			Sound: sound,
		},
		Android: &FCMAndroidConfig{
			Priority: "high",
			Notification: &FCMAndroidNotification{
				Title: title,
				Body:  body,
				Sound: sound,
			},
		},
	}
	
	if data != "" {
		var dataMap map[string]string
		json.Unmarshal([]byte(data), &dataMap)
		fcmToken.Data = dataMap
		fcmToken.Android.Data = dataMap
	}
	
	payload, _ := json.Marshal(fcmToken)
	
	req, err := http.NewRequest("POST", 
		"https://fcm.googleapis.com/v1/projects/"+s.config.FCMProjectID+"/messages:send",
		bytes.NewReader(payload))
	
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.fcmToken)
	
	resp, err := s.fcmClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("FCM error: %s", string(respBody))
	}
	
	return nil
}

// sendAPNS - Send via APNS
func (s *PushService) sendAPNS(token, title, body, data, sound string, badge int) error {
	alert := &APNSAlert{
		Title: title,
		Body:  body,
	}
	
	notification := APNSNotification{
		Aps: &APNSAps{
			Alert:   alert,
			Sound:   sound,
			Badge:   badge,
		},
	}
	
	if data != "" {
		json.Unmarshal([]byte(data), &notification.CustomData)
	}
	
	payload, _ := json.Marshal(notification)
	
	// APNS endpoint
	hostname := "api.push.apple.com"
	if !s.config.UseProduction {
		hostname = "api.sandbox.push.apple.com"
	}
	
	req, err := http.NewRequest("POST", 
		fmt.Sprintf("https://%s:443/3/device/%s", hostname, token),
		bytes.NewReader(payload))
	
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("apns-topic", s.config.APNSBundleID)
	
	// In production, add APNS auth header
	
	resp, err := s.apnsClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("APNS error: %s", string(respBody))
	}
	
	return nil
}

// sendWebPush - Send Web Push notification
func (s *PushService) sendWebPush(token, title, body, data, sound string, badge int) error {
	// WebPush uses same FCM HTTP v1 API with webpush config
	return s.sendFCM(token, title, body, data, sound, badge)
}

// logPush - Log push notification
func (s *PushService) logPush(messageID, userID, token, platform, title, provider string, err error) {
	status := "sent"
	var errorMsg string
	
	if err != nil {
		status = "failed"
		errorMsg = err.Error()
	}
	
	logEntry := &PushLog{
		MessageID: messageID,
		UserID:    userID,
		Token:     token,
		Platform:  platform,
		Title:     title,
		Status:    status,
		SentAt:    time.Now(),
		Error:     errorMsg,
		Provider:  provider,
		CreatedAt: time.Now(),
	}
	
	if status == "sent" {
		now := time.Now()
		logEntry.SentAt = now
	}
	
	s.db.Create(logEntry)
}

// SendTemplatePush - Send push using template
func (s *PushService) SendTemplatePush(userID, token, platform, templateID string, variables map[string]interface{}) (string, error) {
	var template PushTemplate
	result := s.db.Where("template_id = ? AND is_active = ?", templateID, true).First(&template)
	if result.Error != nil {
		return "", fmt.Errorf("template not found: %s", templateID)
	}
	
	// Parse template
	title := parseTemplateSimple(template.Title, variables)
	body := parseTemplateSimple(template.Body, variables)
	data := template.Data
	if data != "" {
		data = parseTemplateSimple(data, variables)
	}
	
	return s.SendPush(userID, token, platform, title, body, data, template.Sound, template.Badge)
}

// parseTemplateSimple - Simple template parser
func parseTemplateSimple(template string, vars map[string]interface{}) string {
	result := template
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// RegisterDevice - Register device token
func (s *PushService) RegisterDevice(userID, token, platform, appVersion, language, timezone string) error {
	var device DeviceToken
	result := s.db.Where("token = ?", token).First(&device)
	
	if result.Error == gorm.ErrRecordNotFound {
		device = DeviceToken{
			UserID:     userID,
			Token:      token,
			Platform:   platform,
			AppVersion: appVersion,
			Language:   language,
			Timezone:   timezone,
			IsActive:   true,
			CreatedAt:  time.Now(),
		}
		s.db.Create(&device)
	} else {
		s.db.Model(&device).Updates(map[string]interface{}{
			"user_id":     userID,
			"platform":    platform,
			"app_version": appVersion,
			"language":    language,
			"timezone":    timezone,
			"is_active":   true,
			"updated_at":  time.Now(),
		})
	}
	
	return nil
}

// UnregisterDevice - Unregister device token
func (s *PushService) UnregisterDevice(token string) error {
	return s.db.Model(&DeviceToken{}).Where("token = ?", token).Update("is_active", false).Error
}

// GetUserDevices - Get user devices
func (s *PushService) GetUserDevices(userID string) ([]DeviceToken, error) {
	var devices []DeviceToken
	err := s.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&devices).Error
	return devices, err
}

// QueuePush - Queue push notification
func (s *PushService) QueuePush(push *PushNotification) error {
	return s.db.Create(push).Error
}

// ProcessQueue - Process push queue
func (s *PushService) ProcessQueue() {
	for {
		select {
		case <-s.stopCh:
			return
		case push := <-s.queue:
			s.processPush(push)
		}
	}
}

// processPush - Process single push
func (s *PushService) processPush(push *PushNotification) {
	s.db.Model(push).Update("status", "sending")
	
	_, err := s.SendPush(push.UserID, push.Token, push.Platform, push.Title, push.Body, push.Data, push.Sound, push.Badge)
	
	if err != nil {
		push.RetryCount++
		push.LastError = err.Error()
		
		if push.RetryCount >= s.config.MaxRetries {
			s.db.Model(push).Updates(map[string]interface{}{
				"status":     "failed",
				"last_error": err.Error(),
			})
		} else {
			s.db.Model(push).Update("status", "queued")
			time.Sleep(s.config.RetryDelay)
			s.queue <- push
		}
		return
	}
	
	// Success
	now := time.Now()
	s.db.Model(push).Updates(map[string]interface{}{
		"status":  "sent",
		"sent_at": now,
	})
}

// StartWorkers - Start queue workers
func (s *PushService) StartWorkers() {
	for i := 0; i < s.config.WorkerCount; i++ {
		s.workers.Add(1)
		go func() {
			defer s.workers.Done()
			s.ProcessQueue()
		}()
	}
}

// StopWorkers - Stop queue workers
func (s *PushService) StopWorkers() {
	close(s.stopCh)
	s.workers.Wait()
}

// Stats - Get push stats
func (s *PushService) Stats() (map[string]interface{}, error) {
	var total, sent, failed, queued int64
	
	s.db.Model(&PushLog{}).Count(&total)
	s.db.Model(&PushLog{}).Where("status = ?", "sent").Count(&sent)
	s.db.Model(&PushLog{}).Where("status = ?", "failed").Count(&failed)
	s.db.Model(&PushNotification{}).Where("status = ?", "queued").Count(&queued)
	
	return map[string]interface{}{
		"total":  total,
		"sent":   sent,
		"failed": failed,
		"queued": queued,
	}, nil
}

// HTTP Handlers

type SendPushRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Token    string `json:"token" binding:"required"`
	Platform string `json:"platform" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Body     string `json:"body"`
	Data     string `json:"data"`
	Sound    string `json:"sound"`
	Badge    int    `json:"badge"`
}

func (s *PushService) SendPushHandler(c *gin.Context) {
	var req SendPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	sound := req.Sound
	if sound == "" {
		sound = "default"
	}
	
	messageID, err := s.SendPush(req.UserID, req.Token, req.Platform, req.Title, req.Body, req.Data, sound, req.Badge)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "sent", "message_id": messageID})
}

type SendTemplateRequest struct {
	UserID     string                 `json:"user_id" binding:"required"`
	Token      string                 `json:"token" binding:"required"`
	Platform   string                 `json:"platform" binding:"required"`
	TemplateID string                 `json:"template_id" binding:"required"`
	Variables  map[string]interface{} `json:"variables"`
}

func (s *PushService) SendTemplateHandler(c *gin.Context) {
	var req SendTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	messageID, err := s.SendTemplatePush(req.UserID, req.Token, req.Platform, req.TemplateID, req.Variables)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "sent", "message_id": messageID})
}

type RegisterDeviceRequest struct {
	UserID    string `json:"user_id" binding:"required"`
	Token     string `json:"token" binding:"required"`
	Platform  string `json:"platform" binding:"required"`
	AppVersion string `json:"app_version"`
	Language  string `json:"language"`
	Timezone  string `json:"timezone"`
}

func (s *PushService) RegisterDeviceHandler(c *gin.Context) {
	var req RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	err := s.RegisterDevice(req.UserID, req.Token, req.Platform, req.AppVersion, req.Language, req.Timezone)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "registered"})
}

func (s *PushService) UnregisterDeviceHandler(c *gin.Context) {
	token := c.Param("token")
	
	err := s.UnregisterDevice(token)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "unregistered"})
}

func (s *PushService) StatsHandler(c *gin.Context) {
	stats, err := s.Stats()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, stats)
}

// Utility functions

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}

// RateLimiter - Token bucket rate limiter
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter - Create new rate limiter
func NewRateLimiter(tokensPerMinute int) *RateLimiter {
	return &RateLimiter{
		tokens:     tokensPerMinute,
		maxTokens:  tokensPerMinute,
		refillRate: time.Minute,
		lastRefill: time.Now(),
	}
}

// Acquire - Acquire a token
func (r *RateLimiter) Acquire() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	if elapsed >= r.refillRate {
		r.tokens = r.maxTokens
		r.lastRefill = now
	}
	
	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

// Main

func main() {
	cfg := PushConfig{
		FCMProjectID:  getEnv("FCM_PROJECT_ID", ""),
		FCMCredentials: getEnv("FCM_CREDENTIALS", ""),
		APNSKeyPath:   getEnv("APNS_KEY_PATH", ""),
		APNSKeyID:     getEnv("APNS_KEY_ID", ""),
		APNSTeamID:    getEnv("APNS_TEAM_ID", ""),
		APNSBundleID:  getEnv("APNS_BUNDLE_ID", "com.tigerwallet.app"),
		UseProduction:  getEnvBool("APNS_PRODUCTION", false),
		MaxRetries:    getEnvInt("PUSH_MAX_RETRIES", 3),
		RetryDelay:    getEnvDuration("PUSH_RETRY_DELAY", 5*time.Second),
		WorkerCount:   getEnvInt("PUSH_WORKERS", 5),
		QueueBufferSize: getEnvInt("PUSH_QUEUE_SIZE", 1000),
		RateLimitPerMinute: getEnvInt("PUSH_RATE_LIMIT", 1000),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "postgres"),
		DBPassword:   getEnv("DB_PASSWORD", "password"),
		DBName:       getEnv("DB_NAME", "push_db"),
		RedisHost:    getEnv("REDIS_HOST", "localhost"),
		RedisPort:    getEnv("REDIS_PORT", "6379"),
		ServerPort:   getEnv("PUSH_SERVER_PORT", "8089"),
	}
	
	service, err := NewPushService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize push service: %v", err)
	}
	
	// Start workers
	service.StartWorkers()
	
	// Setup HTTP routes
	r := gin.Default()
	
	r.POST("/push", service.SendPushHandler)
	r.POST("/push/template", service.SendTemplateHandler)
	r.POST("/devices", service.RegisterDeviceHandler)
	r.DELETE("/devices/:token", service.UnregisterDeviceHandler)
	r.GET("/stats", service.StatsHandler)
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "push"})
	})
	
	go func() {
		log.Printf("Push Service starting on port %s", cfg.ServerPort)
		if err := r.Run(":" + cfg.ServerPort); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	
	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	<-quit
	
	log.Println("Shutting down push service...")
	service.StopWorkers()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		fmt.Sscanf(value, "%d", &i)
		return i
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		d, err := time.ParseDuration(value)
		if err == nil {
			return d
		}
	}
	return defaultValue
}
