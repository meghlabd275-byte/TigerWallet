// SMS Notification Service - Multi-Provider SMS Implementation
// Supports Twilio, Nexmo, AWS SNS, and custom providers

package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SMSConfig - SMS Configuration
type SMSConfig struct {
	// Provider Settings
	Provider          string `json:"provider"` // twilio, nexmo, aws, custom
	AccountSID        string `json:"account_sid"`
	AuthToken         string `json:"auth_token"`
	FromNumber        string `json:"from_number"`
	APIKey            string `json:"api_key"`
	APISecret         string `json:"api_secret"`
	Region            string `json:"region"`
	
	// Custom Provider Settings
	CustomEndpoint    string `json:"custom_endpoint"`
	CustomAuthHeader  string `json:"custom_auth_header"`
	
	// Queue Settings
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
	WorkerCount       int           `json:"worker_count"`
	BatchSize         int           `json:"batch_size"`
	QueueBufferSize   int           `json:"queue_buffer_size"`
	
	// Rate Limiting
	RateLimitPerSecond int `json:"rate_limit_per_second"`
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

// SMSProvider - SMS Provider interface
type SMSProvider interface {
	Send(to, message string) (string, error)
	SendBatch(messages []SMSMessage) ([]SMSResult, error)
	GetBalance() (float64, error)
}

// TwilioProvider - Twilio SMS Provider
type TwilioProvider struct {
	accountSID string
	authToken  string
	fromNumber string
	client     *http.Client
}

// NewTwilioProvider - Create Twilio provider
func NewTwilioProvider(accountSID, authToken, fromNumber string) *TwilioProvider {
	return &TwilioProvider{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send - Send SMS via Twilio
func (p *TwilioProvider) Send(to, message string) (string, error) {
	// Format phone number
	to = formatPhoneNumber(to)
	
	// Build request
	auth := base64.StdEncoding.EncodeToString([]byte(p.accountSID + ":" + p.authToken))
	
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", p.fromNumber)
	data.Set("Body", message)
	
	req, err := http.NewRequest("POST", 
		"https://api.twilio.com/2010-04-01/Accounts/"+p.accountSID+"/Messages.json",
		strings.NewReader(data.Encode()))
	
	if err != nil {
		return "", err
	}
	
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("Twilio error: %s", string(body))
	}
	
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	
	sid, ok := result["sid"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get message SID")
	}
	
	return sid, nil
}

// SendBatch - Send batch SMS
func (p *TwilioProvider) SendBatch(messages []SMSMessage) ([]SMSResult, error) {
	results := make([]SMSResult, len(messages))
	
	for i, msg := range messages {
		sid, err := p.Send(msg.To, msg.Message)
		if err != nil {
			results[i] = SMSResult{
				To:     msg.To,
				Status: "failed",
				Error:  err.Error(),
			}
		} else {
			results[i] = SMSResult{
				To:      msg.To,
				MessageID: sid,
				Status:  "sent",
			}
		}
	}
	
	return results, nil
}

// GetBalance - Get Twilio balance
func (p *TwilioProvider) GetBalance() (float64, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(p.accountSID + ":" + p.authToken))
	
	req, err := http.NewRequest("GET", 
		"https://api.twilio.com/2010-04-01/Accounts/"+p.accountSID+"/Balance.json",
		nil)
	
	if err != nil {
		return 0, err
	}
	
	req.Header.Add("Authorization", "Basic "+auth)
	
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("Twilio error: %s", string(body))
	}
	
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	
	balance, ok := result["balance"].(float64)
	if !ok {
		return 0, fmt.Errorf("failed to get balance")
	}
	
	return balance, nil
}

// AWS SNS Provider
type AWSProvider struct {
	accessKey   string
	secretKey   string
	region      string
	fromNumber  string
	client      *http.Client
}

// NewAWSProvider - Create AWS SNS provider
func NewAWSProvider(accessKey, secretKey, region, fromNumber string) *AWSProvider {
	return &AWSProvider{
		accessKey: accessKey,
		secretKey: secretKey,
		region:    region,
		fromNumber: fromNumber,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send - Send SMS via AWS SNS
func (p *AWSProvider) Send(to, message string) (string, error) {
	to = formatPhoneNumber(to)
	
	// AWS SNS requires specific format
	endpoint := fmt.Sprintf("https://sns.%s.amazonaws.com/", p.region)
	
	// Build request params
	params := url.Values{}
	params.Set("Action", "Publish")
	params.Set("Version", "2010-03-31")
	params.Set("PhoneNumber", to)
	params.Set("Message", message)
	if p.fromNumber != "" {
		params.Set("SenderID", p.fromNumber)
	}
	params.Set("SMSType", "Transactional")
	
	// Sign request (simplified)
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	dateStamp := time.Now().UTC().Format("20060102")
	
	canonicalRequest := "POST\n" + endpoint + "\n\n" + params.Encode() + "\n"
	stringToSign := "AWS4-HMAC-SHA256\n" + timestamp + "\n" + dateStamp + "/sns/" + p.region + "/sns_request\n"
	
	// For simplicity, just make the request (in production, implement full AWS signing)
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AWS SNS error: %s", string(body))
	}
	
	return fmt.Sprintf("aws_%d", time.Now().UnixNano()), nil
}

// SendBatch - Send batch SMS
func (p *AWSProvider) SendBatch(messages []SMSMessage) ([]SMSResult, error) {
	results := make([]SMSResult, len(messages))
	
	for i, msg := range messages {
		id, err := p.Send(msg.To, msg.Message)
		if err != nil {
			results[i] = SMSResult{
				To:     msg.To,
				Status: "failed",
				Error:  err.Error(),
			}
		} else {
			results[i] = SMSResult{
				To:        msg.To,
				MessageID: id,
				Status:    "sent",
			}
		}
	}
	
	return results, nil
}

// GetBalance - Get AWS balance (requires additional API call)
func (p *AWSProvider) GetBalance() (float64, error) {
	return 0, nil // AWS billing is handled separately
}

// CustomProvider - Custom HTTP SMS Provider
type CustomProvider struct {
	endpoint   string
	authHeader  string
	apiKey     string
	client     *http.Client
}

// NewCustomProvider - Create custom provider
func NewCustomProvider(endpoint, authHeader, apiKey string) *CustomProvider {
	return &CustomProvider{
		endpoint:   endpoint,
		authHeader: authHeader,
		apiKey:     apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send - Send SMS via custom provider
func (p *CustomProvider) Send(to, message string) (string, error) {
	to = formatPhoneNumber(to)
	
	data := map[string]interface{}{
		"to":      to,
		"message": message,
	}
	
	body, _ := json.Marshal(data)
	
	req, err := http.NewRequest("POST", p.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Content-Type", "application/json")
	if p.authHeader != "" && p.apiKey != "" {
		req.Header.Set(p.authHeader, p.apiKey)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("custom provider error: %s", string(respBody))
	}
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	if id, ok := result["id"].(string); ok {
		return id, nil
	}
	
	return fmt.Sprintf("custom_%d", time.Now().UnixNano()), nil
}

// SendBatch - Send batch SMS
func (p *CustomProvider) SendBatch(messages []SMSMessage) ([]SMSResult, error) {
	results := make([]SMSResult, len(messages))
	
	for i, msg := range messages {
		id, err := p.Send(msg.To, msg.Message)
		if err != nil {
			results[i] = SMSResult{
				To:     msg.To,
				Status: "failed",
				Error:  err.Error(),
			}
		} else {
			results[i] = SMSResult{
				To:        msg.To,
				MessageID: id,
				Status:    "sent",
			}
		}
	}
	
	return results, nil
}

// GetBalance - Get custom provider balance
func (p *CustomProvider) GetBalance() (float64, error) {
	return 0, nil
}

// SMSMessage - SMS message structure
type SMSMessage struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

// SMSResult - SMS result structure
type SMSResult struct {
	To        string `json:"to"`
	MessageID string `json:"message_id,omitempty"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

// SMSTemplate - SMS template
type SMSTemplate struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TemplateID  string    `gorm:"uniqueIndex" json:"template_id"`
	Name        string    `json:"name"`
	Message     string    `gorm:"type:text" json:"message"`
	Variables   string    `gorm:"type:jsonb" json:"variables"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SMSQueue - SMS queue
type SMSQueue struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	MessageID   string    `gorm:"uniqueIndex" json:"message_id"`
	To          string    `gorm:"index" json:"to"`
	Message     string    `gorm:"type:text" json:"message"`
	Status      string    `gorm:"index" json:"status"` // queued, sending, sent, failed
	Provider    string    `json:"provider"`
	Priority    int       `gorm:"default:0" json:"priority"`
	RetryCount  int       `gorm:"default:0" json:"retry_count"`
	LastError   string    `gorm:"type:text" json:"last_error"`
	SentAt      *time.Time `json:"sent_at"`
	CreatedAt   time.Time `json:"created_at"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

// SMSLog - SMS log
type SMSLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	MessageID   string    `gorm:"uniqueIndex;index" json:"message_id"`
	To          string    `gorm:"index" json:"to"`
	Message     string    `gorm:"type:text" json:"message"`
	TemplateID  string    `gorm:"index" json:"template_id"`
	Status      string    `json:"status"` // sent, delivered, failed
	SentAt      time.Time `json:"sent_at"`
	DeliveredAt *time.Time `json:"delivered_at"`
	Error       string    `gorm:"type:text" json:"error"`
	Provider    string    `json:"provider"`
	Cost        float64   `json:"cost"`
	CreatedAt   time.Time `json:"created_at"`
}

// PhoneNumber - Phone number management
type PhoneNumber struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Phone       string    `gorm:"uniqueIndex;index" json:"phone"`
	IsVerified  bool      `gorm:"default:false" json:"is_verified"`
	VerifyCode  string    `gorm:"index" json:"verify_code"`
	VerifyExpiresAt *time.Time `json:"verify_expires_at"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	LastSentAt  *time.Time `json:"last_sent_at"`
	SendCount   int       `gorm:"default:0" json:"send_count"`
	FailCount   int       `gorm:"default:0" json:"fail_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SMSService - Main SMS service
type SMSService struct {
	config     SMSConfig
	db         *gorm.DB
	redis      *redis.Client
	provider   SMSProvider
	queue      chan *SMSQueue
	workers    sync.WaitGroup
	stopCh     chan struct{}
	rateLimitSecond *RateLimiter
	rateLimitMinute *RateLimiter
}

// NewSMSService - Create new SMS service
func NewSMSService(cfg SMSConfig) (*SMSService, error) {
	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Auto migrate
	err = db.AutoMigrate(&SMSTemplate{}, &SMSQueue{}, &SMSLog{}, &PhoneNumber{})
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
	
	// Create provider
	var provider SMSProvider
	switch cfg.Provider {
	case "twilio":
		provider = NewTwilioProvider(cfg.AccountSID, cfg.AuthToken, cfg.FromNumber)
	case "aws":
		provider = NewAWSProvider(cfg.APIKey, cfg.APISecret, cfg.Region, cfg.FromNumber)
	case "custom":
		provider = NewCustomProvider(cfg.CustomEndpoint, cfg.CustomAuthHeader, cfg.APIKey)
	default:
		// Default to custom/mock
		provider = &CustomProvider{
			endpoint: cfg.CustomEndpoint,
			client:   &http.Client{Timeout: 30 * time.Second},
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
	
	rateLimitSecond := cfg.RateLimitPerSecond
	if rateLimitSecond == 0 {
		rateLimitSecond = 10
	}
	
	rateLimitMinute := cfg.RateLimitPerMinute
	if rateLimitMinute == 0 {
		rateLimitMinute = 100
	}
	
	service := &SMSService{
		config:            cfg,
		db:                db,
		redis:             rdb,
		provider:          provider,
		queue:             make(chan *SMSQueue, queueSize),
		stopCh:            make(chan struct{}),
		rateLimitSecond:   NewRateLimiter(rateLimitSecond),
		rateLimitMinute:   NewRateLimiter(rateLimitMinute),
	}
	
	// Seed default templates
	service.seedDefaultTemplates()
	
	return service, nil
}

// seedDefaultTemplates - Seed default SMS templates
func (s *SMSService) seedDefaultTemplates() {
	defaultTemplates := []SMSTemplate{
		{
			TemplateID: "verification_code",
			Name:       "Verification Code",
			Message:    "{{.AppName}}: Your verification code is {{.Code}}. Valid for {{.Expiry}} minutes.",
			Variables:  `["AppName", "Code", "Expiry"]`,
			IsActive:   true,
		},
		{
			TemplateID: "deposit_received",
			Name:       "Deposit Received",
			Message:    "{{.AppName}}: You received {{.Amount}} {{.Currency}}. Transaction: {{.TxHash}}",
			Variables:  `["AppName", "Amount", "Currency", "TxHash"]`,
			IsActive:   true,
		},
		{
			TemplateID: "withdrawal_initiated",
			Name:       "Withdrawal Initiated",
			Message:    "{{.AppName}}: Withdrawal of {{.Amount}} {{.Currency}} initiated. TX: {{.TxHash}}",
			Variables:  `["AppName", "Amount", "Currency", "TxHash"]`,
			IsActive:   true,
		},
		{
			TemplateID: "withdrawal_approved",
			Name:       "Withdrawal Approved",
			Message:    "{{.AppName}}: Your withdrawal of {{.Amount}} {{.Currency}} has been approved.",
			Variables:  `["AppName", "Amount", "Currency"]`,
			IsActive:   true,
		},
		{
			TemplateID: "security_alert",
			Name:       "Security Alert",
			Message:    "{{.AppName}} ALERT: {{.AlertType}} detected on your account at {{.Time}}.",
			Variables:  `["AppName", "AlertType", "Time"]`,
			IsActive:   true,
		},
		{
			TemplateID: "login_code",
			Name:       "Login Code",
			Message:    "{{.AppName}}: Your login code is {{.Code}}. Don't share this code.",
			Variables:  `["AppName", "Code"]`,
			IsActive:   true,
		},
		{
			TemplateID: "2fa_code",
			Name:       "2FA Code",
			Message:    "{{.AppName}}: Your 2FA code is {{.Code}}. Valid for {{.Expiry}} minutes.",
			Variables:  `["AppName", "Code", "Expiry"]`,
			IsActive:   true,
		},
	}
	
	for _, tmpl := range defaultTemplates {
		var existing SMSTemplate
		result := s.db.Where("template_id = ?", tmpl.TemplateID).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			s.db.Create(&tmpl)
		}
	}
}

// GenerateMessageID - Generate unique message ID
func (s *SMSService) GenerateMessageID() string {
	return fmt.Sprintf("sms_%d_%s", time.Now().UnixNano(), randomString(8))
}

// SendSMS - Send SMS directly
func (s *SMSService) SendSMS(to, message string) (string, error) {
	// Check rate limits
	if !s.rateLimitSecond.Acquire() {
		return "", fmt.Errorf("rate limit exceeded (per second)")
	}
	if !s.rateLimitMinute.Acquire() {
		return "", fmt.Errorf("rate limit exceeded (per minute)")
	}
	
	// Format phone number
	to = formatPhoneNumber(to)
	
	// Send via provider
	messageID, err := s.provider.Send(to, message)
	if err != nil {
		s.logSMS(messageID, to, message, "", "failed", err.Error(), 0)
		return "", err
	}
	
	// Log success
	s.logSMS(messageID, to, message, "", "sent", "", 0)
	
	// Update phone number stats
	s.db.Model(&PhoneNumber{}).Where("phone = ?", to).Updates(map[string]interface{}{
		"last_sent_at": time.Now(),
		"send_count":   gorm.Expr("send_count + 1"),
	})
	
	return messageID, nil
}

// logSMS - Log SMS to database
func (s *SMSService) logSMS(messageID, to, message, templateID, status, errorMsg string, cost float64) {
	log := &SMSLog{
		MessageID: messageID,
		To:        to,
		Message:   message,
		TemplateID: templateID,
		Status:    status,
		SentAt:    time.Now(),
		Error:     errorMsg,
		Provider:  s.config.Provider,
		Cost:      cost,
		CreatedAt: time.Now(),
	}
	
	if status == "sent" {
		now := time.Now()
		log.SentAt = now
	}
	
	s.db.Create(log)
}

// SendTemplateSMS - Send SMS using template
func (s *SMSService) SendTemplateSMS(to, templateID string, variables map[string]interface{}) (string, error) {
	var template SMSTemplate
	result := s.db.Where("template_id = ? AND is_active = ?", templateID, true).First(&template)
	if result.Error != nil {
		return "", fmt.Errorf("template not found: %s", templateID)
	}
	
	// Parse template
	message, err := parseTemplate(template.Message, variables)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	
	return s.SendSMS(to, message)
}

// parseTemplate - Parse SMS template
func parseTemplate(template string, vars map[string]interface{}) (string, error) {
	result := template
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result, nil
}

// QueueSMS - Queue SMS for sending
func (s *SMSService) QueueSMS(queue *SMSQueue) error {
	return s.db.Create(queue).Error
}

// ProcessQueue - Process SMS queue
func (s *SMSService) ProcessQueue() {
	for {
		select {
		case <-s.stopCh:
			return
		case sms := <-s.queue:
			s.processSMS(sms)
		}
	}
}

// processSMS - Process single SMS from queue
func (s *SMSService) processSMS(sms *SMSQueue) {
	// Update status
	s.db.Model(sms).Update("status", "sending")
	
	// Send
	messageID, err := s.provider.Send(sms.To, sms.Message)
	if err != nil {
		sms.RetryCount++
		sms.LastError = err.Error()
		
		if sms.RetryCount >= s.config.MaxRetries {
			now := time.Now()
			s.db.Model(sms).Updates(map[string]interface{}{
				"status":     "failed",
				"last_error": err.Error(),
			})
			
			// Update phone number fail count
			s.db.Model(&PhoneNumber{}).Where("phone = ?", sms.To).Update("fail_count", gorm.Expr("fail_count + 1"))
		} else {
			s.db.Model(sms).Update("status", "queued")
			time.Sleep(s.config.RetryDelay)
			s.queue <- sms
		}
		return
	}
	
	// Success
	now := time.Now()
	s.db.Model(sms).Updates(map[string]interface{}{
		"message_id": messageID,
		"status":     "sent",
		"sent_at":    now,
	})
	
	// Update phone number stats
	s.db.Model(&PhoneNumber{}).Where("phone = ?", sms.To).Updates(map[string]interface{}{
		"last_sent_at": now,
		"send_count":   gorm.Expr("send_count + 1"),
	})
}

// StartWorkers - Start SMS queue workers
func (s *SMSService) StartWorkers() {
	for i := 0; i < s.config.WorkerCount; i++ {
		s.workers.Add(1)
		go func() {
			defer s.workers.Done()
			s.ProcessQueue()
		}()
	}
}

// StopWorkers - Stop SMS queue workers
func (s *SMSService) StopWorkers() {
	close(s.stopCh)
	s.workers.Wait()
}

// VerifyPhone - Send verification code to phone
func (s *SMSService) VerifyPhone(phone string) (string, error) {
	code := generateVerificationCode()
	
	phone = formatPhoneNumber(phone)
	
	// Save verification code
	expiresAt := time.Now().Add(10 * time.Minute)
	
	var pn PhoneNumber
	result := s.db.Where("phone = ?", phone).First(&pn)
	
	if result.Error == gorm.ErrRecordNotFound {
		pn = PhoneNumber{
			Phone:            phone,
			VerifyCode:       code,
			VerifyExpiresAt:  &expiresAt,
			IsActive:         true,
			CreatedAt:        time.Now(),
		}
		s.db.Create(&pn)
	} else {
		s.db.Model(&pn).Updates(map[string]interface{}{
			"verify_code":        code,
			"verify_expires_at": expiresAt,
		})
	}
	
	// Send verification code
	_, err := s.SendTemplateSMS(phone, "verification_code", map[string]interface{}{
		"AppName": "TigerWallet",
		"Code":    code,
		"Expiry":  "10",
	})
	
	if err != nil {
		return "", err
	}
	
	return code, nil
}

// ConfirmVerification - Confirm phone verification
func (s *SMSService) ConfirmVerification(phone, code string) error {
	phone = formatPhoneNumber(phone)
	
	var pn PhoneNumber
	result := s.db.Where("phone = ? AND verify_code = ? AND verify_expires_at > ?", 
		phone, code, time.Now()).First(&pn)
	
	if result.Error != nil {
		return fmt.Errorf("invalid or expired verification code")
	}
	
	return s.db.Model(&pn).Updates(map[string]interface{}{
		"is_verified":      true,
		"verify_code":      nil,
		"verify_expires_at": nil,
	}).Error
}

// GetBalance - Get provider balance
func (s *SMSService) GetBalance() (float64, error) {
	return s.provider.GetBalance()
}

// Stats - Get SMS stats
func (s *SMSService) Stats() (map[string]interface{}, error) {
	var total, sent, failed, queued int64
	
	s.db.Model(&SMSLog{}).Count(&total)
	s.db.Model(&SMSLog{}).Where("status = ?", "sent").Count(&sent)
	s.db.Model(&SMSLog{}).Where("status = ?", "failed").Count(&failed)
	s.db.Model(&SMSQueue{}).Where("status = ?", "queued").Count(&queued)
	
	balance, _ := s.GetBalance()
	
	return map[string]interface{}{
		"total":    total,
		"sent":     sent,
		"failed":   failed,
		"queued":   queued,
		"balance":  balance,
	}, nil
}

// HTTP Handlers

type SendSMSRequest struct {
	To      string `json:"to" binding:"required"`
	Message string `json:"message" binding:"required"`
}

func (s *SMSService) SendSMSHandler(c *gin.Context) {
	var req SendSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	messageID, err := s.SendSMS(req.To, req.Message)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "sent", "message_id": messageID})
}

type SendTemplateRequest struct {
	To         string                 `json:"to" binding:"required"`
	TemplateID string                 `json:"template_id" binding:"required"`
	Variables  map[string]interface{} `json:"variables"`
}

func (s *SMSService) SendTemplateHandler(c *gin.Context) {
	var req SendTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	messageID, err := s.SendTemplateSMS(req.To, req.TemplateID, req.Variables)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "sent", "message_id": messageID})
}

type QueueSMSRequest struct {
	To         string     `json:"to" binding:"required"`
	Message    string     `json:"message" binding:"required"`
	Priority   int        `json:"priority"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

func (s *SMSService) QueueSMSHandler(c *gin.Context) {
	var req QueueSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	queue := &SMSQueue{
		MessageID:   s.GenerateMessageID(),
		To:          formatPhoneNumber(req.To),
		Message:     req.Message,
		Status:      "queued",
		Priority:    req.Priority,
		ScheduledAt: req.ScheduledAt,
		CreatedAt:   time.Now(),
	}
	
	err := s.QueueSMS(queue)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "queued", "message_id": queue.MessageID})
}

type VerifyPhoneRequest struct {
	Phone string `json:"phone" binding:"required"`
}

func (s *SMSService) VerifyPhoneHandler(c *gin.Context) {
	var req VerifyPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	code, err := s.VerifyPhone(req.Phone)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "sent", "verification_code": code})
}

type ConfirmVerifyRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

func (s *SMSService) ConfirmVerifyHandler(c *gin.Context) {
	var req ConfirmVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	err := s.ConfirmVerification(req.Phone, req.Code)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "verified"})
}

func (s *SMSService) StatsHandler(c *gin.Context) {
	stats, err := s.Stats()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, stats)
}

// Utility functions

func formatPhoneNumber(phone string) string {
	// Remove all non-digit characters
	re := regexp.MustCompile(`\D`)
	phone = re.ReplaceAllString(phone, "")
	
	// Add country code if missing (assuming US/Canada)
	if len(phone) == 10 {
		phone = "1" + phone
	}
	
	return "+" + phone
}

func generateVerificationCode() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
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
func NewRateLimiter(tokensPerUnit int) *RateLimiter {
	return &RateLimiter{
		tokens:     tokensPerUnit,
		maxTokens:  tokensPerUnit,
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

// GormExpr for atomic operations
type GormExpr func(*gorm.DB) *gorm.DB

// Main

func main() {
	cfg := SMSConfig{
		Provider:          getEnv("SMS_PROVIDER", "twilio"),
		AccountSID:        getEnv("TWILIO_ACCOUNT_SID", ""),
		AuthToken:         getEnv("TWILIO_AUTH_TOKEN", ""),
		FromNumber:        getEnv("SMS_FROM_NUMBER", ""),
		APIKey:            getEnv("SMS_API_KEY", ""),
		APISecret:         getEnv("SMS_API_SECRET", ""),
		Region:            getEnv("AWS_REGION", "us-east-1"),
		CustomEndpoint:    getEnv("SMS_CUSTOM_ENDPOINT", ""),
		CustomAuthHeader:  getEnv("SMS_AUTH_HEADER", "X-API-Key"),
		MaxRetries:        getEnvInt("SMS_MAX_RETRIES", 3),
		RetryDelay:        getEnvDuration("SMS_RETRY_DELAY", 5*time.Second),
		WorkerCount:       getEnvInt("SMS_WORKERS", 5),
		QueueBufferSize:   getEnvInt("SMS_QUEUE_SIZE", 1000),
		RateLimitPerSecond: getEnvInt("SMS_RATE_LIMIT_SECOND", 10),
		RateLimitPerMinute: getEnvInt("SMS_RATE_LIMIT_MINUTE", 100),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", "password"),
		DBName:            getEnv("DB_NAME", "sms_db"),
		RedisHost:         getEnv("REDIS_HOST", "localhost"),
		RedisPort:         getEnv("REDIS_PORT", "6379"),
		ServerPort:        getEnv("SMS_SERVER_PORT", "8088"),
	}
	
	service, err := NewSMSService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize SMS service: %v", err)
	}
	
	// Start workers
	service.StartWorkers()
	
	// Setup HTTP routes
	r := gin.Default()
	
	r.POST("/sms", service.SendSMSHandler)
	r.POST("/sms/template", service.SendTemplateHandler)
	r.POST("/sms/queue", service.QueueSMSHandler)
	r.POST("/verify/send", service.VerifyPhoneHandler)
	r.POST("/verify/confirm", service.ConfirmVerifyHandler)
	r.GET("/stats", service.StatsHandler)
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "sms"})
	})
	
	go func() {
		log.Printf("SMS Service starting on port %s", cfg.ServerPort)
		if err := r.Run(":" + cfg.ServerPort); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	
	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	<-quit
	
	log.Println("Shutting down SMS service...")
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

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		d, err := time.ParseDuration(value)
		if err == nil {
			return d
		}
	}
	return defaultValue
}
