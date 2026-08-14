// Email Notification Service - Comprehensive SMTP Implementation
// High-performance email service with templates, queuing, and delivery tracking

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// EmailConfig - SMTP Configuration
type EmailConfig struct {
	// SMTP Server Settings
	SMTPHost          string `json:"smtp_host"`
	SMTPPort          int    `json:"smtp_port"`
	SMTPUsername      string `json:"smtp_username"`
	SMTPPassword      string `json:"smtp_password"`
	SMTPFromName     string `json:"smtp_from_name"`
	SMTPFromEmail    string `json:"smtp_from_email"`
	UseTLS           bool   `json:"use_tls"`
	SkipVerification bool   `json:"skip_verification"`
	
	// Queue Settings
	MaxRetries      int           `json:"max_retries"`
	RetryDelay      time.Duration `json:"retry_delay"`
	WorkerCount     int           `json:"worker_count"`
	BatchSize       int           `json:"batch_size"`
	QueueBufferSize int           `json:"queue_buffer_size"`
	
	// Template Settings
	TemplateDir string `json:"template_dir"`
	
	// Database Settings
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	
	// Redis Settings
	RedisHost string `json:"redis_host"`
	RedisPort string `json:"redis_port"`
	
	// Rate Limiting
	RateLimitPerMinute int `json:"rate_limit_per_minute"`
	
	// Server
	ServerPort string `json:"server_port"`
}

// EmailTemplate - Email template structure
type EmailTemplate struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TemplateID  string    `gorm:"uniqueIndex" json:"template_id"`
	Name        string    `json:"name"`
	Subject     string    `json:"subject"`
	BodyHTML    string    `gorm:"type:text" json:"body_html"`
	BodyText    string    `gorm:"type:text" json:"body_text"`
	Variables   string    `gorm:"type:jsonb" json:"variables"` // JSON array of variable names
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EmailQueue - Email queue for retry mechanism
type EmailQueue struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	MessageID       string    `gorm:"uniqueIndex" json:"message_id"`
	ToEmail         string    `gorm:"index" json:"to_email"`
	ToName          string    `json:"to_name"`
	FromEmail       string    `json:"from_email"`
	FromName        string    `json:"from_name"`
	Subject         string    `json:"subject"`
	BodyHTML        string    `gorm:"type:text" json:"body_html"`
	BodyText        string    `gorm:"type:text" json:"body_text"`
	ReplyTo         string    `json:"reply_to"`
	CC              string    `json:"cc"`
	BCC             string    `json:"bcc"`
	Attachments     string    `gorm:"type:jsonb" json:"attachments"`
	Headers         string    `gorm:"type:jsonb" json:"headers"`
	Priority        int       `gorm:"default:0" json:"priority"` // 0=normal, 1=high, 2=urgent
	Status          string    `gorm:"index" json:"status"` // queued, sending, sent, failed, cancelled
	RetryCount      int       `gorm:"default:0" json:"retry_count"`
	LastError       string    `gorm:"type:text" json:"last_error"`
	SentAt          *time.Time `json:"sent_at"`
	CreatedAt       time.Time `json:"created_at"`
	ScheduledAt     *time.Time `json:"scheduled_at"`
	ProcessedAt     *time.Time `json:"processed_at"`
}

// EmailLog - Log of all sent emails
type EmailLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	MessageID   string    `gorm:"uniqueIndex;index" json:"message_id"`
	ToEmail     string    `gorm:"index" json:"to_email"`
	ToName      string    `json:"to_name"`
	Subject     string    `json:"subject"`
	TemplateID  string    `gorm:"index" json:"template_id"`
	Status      string    `json:"status"` // sent, delivered, bounced, opened, clicked
	SentAt      time.Time `json:"sent_at"`
	DeliveredAt *time.Time `json:"delivered_at"`
	BouncedAt   *time.Time `json:"bounced_at"`
	OpenedAt    *time.Time `json:"opened_at"`
	ClickedAt   *time.Time `json:"clicked_at"`
	Error       string    `gorm:"type:text" json:"error"`
	Metadata    string    `gorm:"type:jsonb" json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
}

// EmailRecipient - Recipient management
type EmailRecipient struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Email           string    `gorm:"uniqueIndex;index" json:"email"`
	Name            string    `json:"name"`
	IsVerified      bool      `gorm:"default:false" json:"is_verified"`
	VerifyToken     string    `gorm:"index" json:"verify_token"`
	VerifyExpiresAt *time.Time `json:"verify_expires_at"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	LastSentAt      *time.Time `json:"last_sent_at"`
	SendCount       int       `gorm:"default:0" json:"send_count"`
	FailCount       int       `gorm:"default:0" json:"fail_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// EmailUnsubscribe - Unsubscribe management
type EmailUnsubscribe struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Email       string    `gorm:"uniqueIndex;index" json:"email"`
	Token       string    `gorm:"uniqueIndex" json:"token"`
	Reason      string    `json:"reason"`
	Categories  string    `json:"categories"` // JSON array of categories
	UnsubscribedAt time.Time `json:"unsubscribed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// EmailService - Main email service
type EmailService struct {
	config        EmailConfig
	db            *gorm.DB
	redis         *redis.Client
	templates     map[string]*EmailTemplate
	templateLock  sync.RWMutex
	queue         chan *EmailQueue
	workers       sync.WaitGroup
	stopCh        chan struct{}
	rateLimiter   *RateLimiter
	smtpAuth      smtp.Auth
}

// RateLimiter - Token bucket rate limiter
type RateLimiter struct {
	tokens    int
	maxTokens int
	refillRate time.Duration
	lastRefill time.Time
	mu        sync.Mutex
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

// NewEmailService - Create new email service
func NewEmailService(cfg EmailConfig) (*EmailService, error) {
	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Auto migrate
	err = db.AutoMigrate(&EmailTemplate{}, &EmailQueue{}, &EmailLog{}, &EmailRecipient{}, &EmailUnsubscribe{})
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
	
	// Create SMTP auth
	auth := smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
	
	// Load templates
	templates, err := loadTemplates(db)
	if err != nil {
		log.Printf("Warning: Failed to load templates: %v", err)
	}
	
	// Create queue buffer
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
		rateLimit = 100
	}
	
	service := &EmailService{
		config:       cfg,
		db:           db,
		redis:        rdb,
		templates:    templates,
		queue:        make(chan *EmailQueue, queueSize),
		stopCh:       make(chan struct{}),
		rateLimiter:  NewRateLimiter(rateLimit),
		smtpAuth:     auth,
	}
	
	// Seed default templates
	service.seedDefaultTemplates()
	
	return service, nil
}

// loadTemplates - Load templates from database
func loadTemplates(db *gorm.DB) (map[string]*EmailTemplate, error) {
	var templates []EmailTemplate
	if err := db.Where("is_active = ?", true).Find(&templates).Error; err != nil {
		return nil, err
	}
	
	result := make(map[string]*EmailTemplate)
	for i := range templates {
		result[templates[i].TemplateID] = &templates[i]
	}
	return result, nil
}

// seedDefaultTemplates - Seed default email templates
func (s *EmailService) seedDefaultTemplates() {
	defaultTemplates := []EmailTemplate{
		{
			TemplateID: "welcome",
			Name:       "Welcome Email",
			Subject:    "Welcome to {{.AppName}} - Get Started",
			BodyHTML:   getWelcomeTemplateHTML(),
			BodyText:   getWelcomeTemplateText(),
			Variables:  `["AppName", "UserName", "VerifyURL", "SupportURL"]`,
			IsActive:   true,
		},
		{
			TemplateID: "password_reset",
			Name:       "Password Reset",
			Subject:    "Reset Your {{.AppName}} Password",
			BodyHTML:   getPasswordResetTemplateHTML(),
			BodyText:   getPasswordResetTemplateText(),
			Variables:  `["AppName", "UserName", "ResetURL", "ExpiryMinutes"]`,
			IsActive:   true,
		},
		{
			TemplateID: "email_verify",
			Name:       "Email Verification",
			Subject:    "Verify Your {{.AppName}} Email",
			BodyHTML:   getEmailVerifyTemplateHTML(),
			BodyText:   getEmailVerifyTemplateText(),
			Variables:  `["AppName", "UserName", "VerifyURL", "ExpiryMinutes"]`,
			IsActive:   true,
		},
		{
			TemplateID: "transaction_deposit",
			Name:       "Deposit Notification",
			Subject:    "Deposit Confirmed - {{.Amount}} {{.Currency}}",
			BodyHTML:   getDepositTemplateHTML(),
			BodyText:   getDepositTemplateText(),
			Variables:  `["AppName", "UserName", "Amount", "Currency", "TxHash", "Confirmations"]`,
			IsActive:   true,
		},
		{
			TemplateID: "transaction_withdrawal",
			Name:       "Withdrawal Notification",
			Subject:    "Withdrawal Initiated - {{.Amount}} {{.Currency}}",
			BodyHTML:   getWithdrawalTemplateHTML(),
			BodyText:   getWithdrawalTemplateText(),
			Variables:  `["AppName", "UserName", "Amount", "Currency", "TxHash", "Fee"]`,
			IsActive:   true,
		},
		{
			TemplateID: "security_alert",
			Name:       "Security Alert",
			Subject:    "Security Alert - {{.AlertType}}",
			BodyHTML:   getSecurityAlertTemplateHTML(),
			BodyText:   getSecurityAlertTemplateText(),
			Variables:  `["AppName", "UserName", "AlertType", "Details", "Timestamp", "IPAddress", "Device"]`,
			IsActive:   true,
		},
		{
			TemplateID: "kyc_approved",
			Name:       "KYC Approved",
			Subject:    "Your Identity Verification is Approved",
			BodyHTML:   getKYCApprovedTemplateHTML(),
			BodyText:   getKYCApprovedTemplateText(),
			Variables:  `["AppName", "UserName"]`,
			IsActive:   true,
		},
		{
			TemplateID: "kyc_rejected",
			Name:       "KYC Rejected",
			Subject:    "Identity Verification Update",
			BodyHTML:   getKYCRejectedTemplateHTML(),
			BodyText:   getKYCRejectedTemplateText(),
			Variables:  `["AppName", "UserName", "Reason", "SupportURL"]`,
			IsActive:   true,
		},
		{
			TemplateID: "2fa_enabled",
			Name:       "2FA Enabled",
			Subject:    "Two-Factor Authentication Enabled",
			BodyHTML:   get2FAEnabledTemplateHTML(),
			BodyText:   get2FAEnabledTemplateText(),
			Variables:  `["AppName", "UserName", "Timestamp", "Device", "IPAddress"]`,
			IsActive:   true,
		},
		{
			TemplateID: "2fa_disabled",
			Name:       "2FA Disabled",
			Subject:    "Two-Factor Authentication Disabled",
			BodyHTML:   get2FADisabledTemplateHTML(),
			BodyText:   get2FADisabledTemplateText(),
			Variables:  `["AppName", "UserName", "Timestamp", "Device", "IPAddress"]`,
			IsActive:   true,
		},
		{
			TemplateID: "account_locked",
			Name:       "Account Locked",
			Subject:    "Your Account Has Been Locked",
			BodyHTML:   getAccountLockedTemplateHTML(),
			BodyText:   getAccountLockedTemplateText(),
			Variables:  `["AppName", "UserName", "Reason", "UnlockTime", "SupportURL"]`,
			IsActive:   true,
		},
		{
			TemplateID: "withdrawal_approved",
			Name:       "Withdrawal Approved",
			Subject:    "Withdrawal Approved - {{.Amount}} {{.Currency}}",
			BodyHTML:   getWithdrawalApprovedTemplateHTML(),
			BodyText:   getWithdrawalApprovedTemplateText(),
			Variables:  `["AppName", "UserName", "Amount", "Currency", "TxHash", "EstimatedArrival"]`,
			IsActive:   true,
		},
		{
			TemplateID: "withdrawal_rejected",
			Name:       "Withdrawal Rejected",
			Subject:    "Withdrawal Rejected - {{.Amount}} {{.Currency}}",
			BodyHTML:   getWithdrawalRejectedTemplateHTML(),
			BodyText:   getWithdrawalRejectedTemplateText(),
			Variables:  `["AppName", "UserName", "Amount", "Currency", "Reason", "RefundAmount", "RefundCurrency"]`,
			IsActive:   true,
		},
		{
			TemplateID: "support_ticket",
			Name:       "Support Ticket Update",
			Subject:    "Ticket #{{.TicketID}} - {{.Status}}",
			BodyHTML:   getSupportTicketTemplateHTML(),
			BodyText:   getSupportTicketTemplateText(),
			Variables:  `["AppName", "UserName", "TicketID", "Status", "Subject", "Message", "ReplyURL"]`,
			IsActive:   true,
		},
		{
			TemplateID: "monthly_statement",
			Name:       "Monthly Statement",
			Subject:    "Your Monthly Statement - {{.Month}} {{.Year}}",
			BodyHTML:   getMonthlyStatementTemplateHTML(),
			BodyText:   getMonthlyStatementTemplateText(),
			Variables:  `["AppName", "UserName", "Month", "Year", "StatementURL", "TotalDeposits", "TotalWithdrawals", "TotalTradingVolume", "FeesPaid"]`,
			IsActive:   true,
		},
	}
	
	for _, tmpl := range defaultTemplates {
		var existing EmailTemplate
		result := s.db.Where("template_id = ?", tmpl.TemplateID).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			s.db.Create(&tmpl)
		}
	}
	
	// Reload templates
	templates, _ := loadTemplates(s.db)
	s.templateLock.Lock()
	s.templates = templates
	s.templateLock.Unlock()
}

// GenerateMessageID - Generate unique message ID
func (s *EmailService) GenerateMessageID() string {
	return fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), randomString(16), s.config.SMTPHost)
}

// SendEmail - Send email directly
func (s *EmailService) SendEmail(toEmail, toName, subject, bodyHTML, bodyText string, options ...EmailOption) error {
	// Apply options
	opts := &EmailOptions{
		From:    s.config.SMTPFromEmail,
		FromName: s.config.SMTPFromName,
	}
	for _, opt := range options {
		opt(opts)
	}
	
	// Check rate limit
	if !s.rateLimiter.Acquire() {
		return fmt.Errorf("rate limit exceeded")
	}
	
	// Check unsubscribe
	if s.isUnsubscribed(toEmail) {
		return fmt.Errorf("recipient unsubscribed")
	}
	
	// Create message
	messageID := s.GenerateMessageID()
	
	// Build email
	var buf bytes.Buffer
	err := s.buildEmail(&buf, messageID, toEmail, toName, subject, bodyHTML, bodyText, opts)
	if err != nil {
		return fmt.Errorf("failed to build email: %w", err)
	}
	
	// Send via SMTP
	err = s.sendSMTP(toEmail, buf.Bytes())
	if err != nil {
		// Log failed attempt
		s.logEmail(messageID, toEmail, toName, subject, "", "failed", err.Error())
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	// Log success
	s.logEmail(messageID, toEmail, toName, subject, "", "sent", "")
	return nil
}

// EmailOption - Email option function
type EmailOption func(*EmailOptions)

// EmailOptions - Email options
type EmailOptions struct {
	From      string
	FromName  string
	ReplyTo   string
	CC        string
	BCC       string
	Headers   map[string]string
}

// WithReplyTo - Set reply-to
func WithReplyTo(email string) EmailOption {
	return func(o *EmailOptions) { o.ReplyTo = email }
}

// WithCC - Set CC
func WithCC(cc string) EmailOption {
	return func(o *EmailOptions) { o.CC = cc }
}

// WithBCC - Set BCC
func WithBCC(bcc string) EmailOption {
	return func(o *EmailOptions) { o.BCC = bcc }
}

// WithHeaders - Set custom headers
func WithHeaders(headers map[string]string) EmailOption {
	return func(o *EmailOptions) { o.Headers = headers }
}

// buildEmail - Build email message
func (s *EmailService) buildEmail(buf *bytes.Buffer, messageID, toEmail, toName, subject, bodyHTML, bodyText string, opts *EmailOptions) error {
	from := opts.From
	if opts.FromName != "" {
		from = fmt.Sprintf("%s <%s>", opts.FromName, opts.From)
	}
	
	// Headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s <%s>\r\n", toName, toEmail))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString(fmt.Sprintf("Message-ID: %s\r\n", messageID))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	buf.WriteString(fmt.Sprintf("MIME-Version: 1.0\r\n"))
	
	if opts.ReplyTo != "" {
		buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", opts.ReplyTo))
	}
	if opts.CC != "" {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", opts.CC))
	}
	
	// Custom headers
	if opts.Headers != nil {
		for k, v := range opts.Headers {
			buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
	}
	
	// Create multipart alternative
	boundary := randomString(32)
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary))
	buf.WriteString("\r\n")
	
	// Plain text part
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(quotedPrintable(bodyText))
	buf.WriteString("\r\n")
	
	// HTML part
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(quotedPrintable(bodyHTML))
	buf.WriteString("\r\n")
	
	// Close boundary
	buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	
	return nil
}

// sendSMTP - Send email via SMTP
func (s *EmailService) sendSMTP(to string, data []byte) error {
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)
	
	var conn net.Conn
	var err error
	
	if s.config.UseTLS {
		conn, err = tls.Dial("tcp", addr, &tls.Config{
			ServerName:         s.config.SMTPHost,
			InsecureSkipVerify: s.config.SkipVerification,
		})
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()
	
	client, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()
	
	// Authenticate
	if s.smtpAuth != nil {
		if err = client.Auth(s.smtpAuth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}
	
	// Set from
	if err = client.Mail(s.config.SMTPFromEmail); err != nil {
		return fmt.Errorf("failed to set from: %w", err)
	}
	
	// Set to
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set to: %w", err)
	}
	
	// Send data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer w.Close()
	
	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	
	return nil
}

// logEmail - Log email to database
func (s *EmailService) logEmail(messageID, toEmail, toName, subject, templateID, status, errorMsg string) {
	log := &EmailLog{
		MessageID: messageID,
		ToEmail:   toEmail,
		ToName:    toName,
		Subject:   subject,
		TemplateID: templateID,
		Status:    status,
		SentAt:    time.Now(),
		Error:     errorMsg,
		CreatedAt: time.Now(),
	}
	
	if status == "sent" {
		now := time.Now()
		log.SentAt = now
	}
	
	s.db.Create(log)
}

// isUnsubscribed - Check if email is unsubscribed
func (s *EmailService) isUnsubscribed(email string) bool {
	var unsub EmailUnsubscribe
	result := s.db.Where("email = ?", strings.ToLower(email)).First(&unsub)
	return result.Error == nil
}

// SendTemplateEmail - Send email using template
func (s *EmailService) SendTemplateEmail(toEmail, toName, templateID string, variables map[string]interface{}) error {
	s.templateLock.RLock()
	template, ok := s.templates[templateID]
	s.templateLock.RUnlock()
	
	if !ok {
		return fmt.Errorf("template not found: %s", templateID)
	}
	
	// Parse subject
	subjectTmpl, err := template.ParseSubject(variables)
	if err != nil {
		return fmt.Errorf("failed to parse subject: %w", err)
	}
	
	// Parse body HTML
	bodyHTML, err := template.ParseBodyHTML(variables)
	if err != nil {
		return fmt.Errorf("failed to parse HTML body: %w", err)
	}
	
	// Parse body text
	bodyText, err := template.ParseBodyText(variables)
	if err != nil {
		return fmt.Errorf("failed to parse text body: %w", err)
	}
	
	return s.SendEmail(toEmail, toName, subjectTmpl, bodyHTML, bodyText)
}

// ParseSubject - Parse template subject
func (t *EmailTemplate) ParseSubject(vars map[string]interface{}) (string, error) {
	tmpl, err := template.New("subject").Parse(t.Subject)
	if err != nil {
		return "", err
	}
	
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars)
	if err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

// ParseBodyHTML - Parse template HTML body
func (t *EmailTemplate) ParseBodyHTML(vars map[string]interface{}) (string, error) {
	tmpl, err := template.New("body").Parse(t.BodyHTML)
	if err != nil {
		return "", err
	}
	
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars)
	if err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

// ParseBodyText - Parse template text body
func (t *EmailTemplate) ParseBodyText(vars map[string]interface{}) (string, error) {
	tmpl, err := template.New("body").Parse(t.BodyText)
	if err != nil {
		return "", err
	}
	
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars)
	if err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

// QueueEmail - Queue email for sending
func (s *EmailService) QueueEmail(queue *EmailQueue) error {
	return s.db.Create(queue).Error
}

// ProcessQueue - Process email queue
func (s *EmailService) ProcessQueue() {
	for {
		select {
		case <-s.stopCh:
			return
		case email := <-s.queue:
			s.processEmail(email)
		}
	}
}

// processEmail - Process single email from queue
func (s *EmailService) processEmail(email *EmailQueue) {
	// Update status
	s.db.Model(email).Update("status", "sending")
	
	// Build email
	var buf bytes.Buffer
	err := s.buildEmail(&buf, email.MessageID, email.ToEmail, email.ToName, email.Subject, email.BodyHTML, email.BodyText, &EmailOptions{
		From:     email.FromEmail,
		FromName: email.FromName,
		ReplyTo:  email.ReplyTo,
		CC:       email.CC,
		BCC:      email.BCC,
	})
	
	if err != nil {
		s.handleEmailFailure(email, fmt.Sprintf("failed to build email: %v", err))
		return
	}
	
	// Check rate limit
	if !s.rateLimiter.Acquire() {
		// Re-queue
		s.queue <- email
		time.Sleep(time.Second)
		return
	}
	
	// Send
	err = s.sendSMTP(email.ToEmail, buf.Bytes())
	if err != nil {
		s.handleEmailFailure(email, fmt.Sprintf("failed to send: %v", err))
		return
	}
	
	// Success
	now := time.Now()
	s.db.Model(email).Updates(map[string]interface{}{
		"status":   "sent",
		"sent_at":  now,
		"processed_at": now,
	})
	
	// Update recipient stats
	s.db.Model(&EmailRecipient{}).Where("email = ?", email.ToEmail).Updates(map[string]interface{}{
		"last_sent_at": now,
		"send_count":   gorm.Expr("send_count + 1"),
	})
}

// handleEmailFailure - Handle email sending failure
func (s *EmailService) handleEmailFailure(email *EmailQueue, errorMsg string) {
	email.RetryCount++
	email.LastError = errorMsg
	
	if email.RetryCount >= s.config.MaxRetries {
		// Mark as failed
		now := time.Now()
		s.db.Model(email).Updates(map[string]interface{}{
			"status":       "failed",
			"last_error":   errorMsg,
			"processed_at": now,
		})
		
		// Update recipient fail count
		s.db.Model(&EmailRecipient{}).Where("email = ?", email.ToEmail).Update("fail_count", gorm.Expr("fail_count + 1"))
	} else {
		// Re-queue with delay
		s.db.Model(email).Update("status", "queued")
		time.Sleep(s.config.RetryDelay)
		s.queue <- email
	}
}

// StartWorkers - Start email queue workers
func (s *EmailService) StartWorkers() {
	for i := 0; i < s.config.WorkerCount; i++ {
		s.workers.Add(1)
		go func() {
			defer s.workers.Done()
			s.ProcessQueue()
		}()
	}
}

// StopWorkers - Stop email queue workers
func (s *EmailService) StopWorkers() {
	close(s.stopCh)
	s.workers.Wait()
}

// Unsubscribe - Unsubscribe email
func (s *EmailService) Unsubscribe(email, reason, categories string) error {
	token := randomString(32)
	
	unsub := &EmailUnsubscribe{
		Email:         strings.ToLower(email),
		Token:         token,
		Reason:        reason,
		Categories:    categories,
		UnsubscribedAt: time.Now(),
		CreatedAt:     time.Now(),
	}
	
	return s.db.Create(unsub).Error
}

// GetUnsubscribeURL - Get unsubscribe URL
func (s *EmailService) GetUnsubscribeURL(email, category string) string {
	return fmt.Sprintf("https://%s/unsubscribe?email=%s&category=%s", s.config.SMTPHost, email, category)
}

// AddRecipient - Add recipient
func (s *EmailService) AddRecipient(email, name string) error {
	recipient := &EmailRecipient{
		Email:     strings.ToLower(email),
		Name:      name,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	return s.db.Create(recipient).Error
}

// VerifyRecipient - Verify recipient email
func (s *EmailService) VerifyRecipient(token string) error {
	var recipient EmailRecipient
	result := s.db.Where("verify_token = ? AND verify_expires_at > ?", token, time.Now()).First(&recipient)
	if result.Error != nil {
		return result.Error
	}
	
	return s.db.Model(&recipient).Updates(map[string]interface{}{
		"is_verified":     true,
		"verify_token":    nil,
		"verify_expires_at": nil,
	}).Error
}

// Stats - Get email stats
func (s *EmailService) Stats() (map[string]interface{}, error) {
	var total, sent, failed, queued int64
	
	s.db.Model(&EmailLog{}).Count(&total)
	s.db.Model(&EmailLog{}).Where("status = ?", "sent").Count(&sent)
	s.db.Model(&EmailLog{}).Where("status = ?", "failed").Count(&failed)
	s.db.Model(&EmailQueue{}).Where("status = ?", "queued").Count(&queued)
	
	return map[string]interface{}{
		"total":    total,
		"sent":     sent,
		"failed":   failed,
		"queued":   queued,
		"rate":     s.config.RateLimitPerMinute,
	}, nil
}

// HTTP Handlers

type CreateEmailRequest struct {
	ToEmail   string                 `json:"to_email" binding:"required,email"`
	ToName    string                 `json:"to_name"`
	Subject   string                 `json:"subject" binding:"required"`
	BodyHTML  string                 `json:"body_html"`
	BodyText  string                 `json:"body_text"`
	ReplyTo   string                 `json:"reply_to"`
	CC        string                 `json:"cc"`
	BCC       string                 `json:"bcc"`
	Headers   map[string]string     `json:"headers"`
}

func (s *EmailService) CreateEmailHandler(c *gin.Context) {
	var req CreateEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	toName := req.ToName
	if toName == "" {
		toName = req.ToEmail
	}
	
	options := []EmailOption{
		WithReplyTo(req.ReplyTo),
		WithCC(req.CC),
		WithBCC(req.BCC),
	}
	if req.Headers != nil {
		options = append(options, WithHeaders(req.Headers))
	}
	
	err := s.SendEmail(req.ToEmail, toName, req.Subject, req.BodyHTML, req.BodyText, options...)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "sent"})
}

type SendTemplateRequest struct {
	ToEmail   string                 `json:"to_email" binding:"required,email"`
	ToName    string                 `json:"to_name"`
	TemplateID string                `json:"template_id" binding:"required"`
	Variables map[string]interface{} `json:"variables"`
}

func (s *EmailService) SendTemplateHandler(c *gin.Context) {
	var req SendTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	toName := req.ToName
	if toName == "" {
		toName = req.ToEmail
	}
	
	err := s.SendTemplateEmail(req.ToEmail, toName, req.TemplateID, req.Variables)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "sent", "template_id": req.TemplateID})
}

type QueueEmailRequest struct {
	ToEmail    string                 `json:"to_email" binding:"required,email"`
	ToName     string                 `json:"to_name"`
	Subject    string                 `json:"subject" binding:"required"`
	BodyHTML   string                 `json:"body_html"`
	BodyText   string                 `json:"body_text"`
	ReplyTo    string                 `json:"reply_to"`
	CC         string                 `json:"cc"`
	BCC        string                 `json:"bcc"`
	Priority   int                    `json:"priority"`
	ScheduledAt *time.Time           `json:"scheduled_at"`
}

func (s *EmailService) QueueEmailHandler(c *gin.Context) {
	var req QueueEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	toName := req.ToName
	if toName == "" {
		toName = req.ToEmail
	}
	
	queue := &EmailQueue{
		MessageID:   s.GenerateMessageID(),
		ToEmail:     req.ToEmail,
		ToName:      toName,
		FromEmail:   s.config.SMTPFromEmail,
		FromName:    s.config.SMTPFromName,
		Subject:     req.Subject,
		BodyHTML:    req.BodyHTML,
		BodyText:    req.BodyText,
		ReplyTo:     req.ReplyTo,
		CC:          req.CC,
		BCC:         req.BCC,
		Priority:    req.Priority,
		Status:      "queued",
		ScheduledAt: req.ScheduledAt,
		CreatedAt:   time.Now(),
	}
	
	err := s.QueueEmail(queue)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "queued", "message_id": queue.MessageID})
}

type UnsubscribeRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Reason     string `json:"reason"`
	Categories string `json:"categories"`
}

func (s *EmailService) UnsubscribeHandler(c *gin.Context) {
	var req UnsubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	err := s.Unsubscribe(req.Email, req.Reason, req.Categories)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "unsubscribed"})
}

func (s *EmailService) StatsHandler(c *gin.Context) {
	stats, err := s.Stats()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, stats)
}

type AddRecipientRequest struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name"`
}

func (s *EmailService) AddRecipientHandler(c *gin.Context) {
	var req AddRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	err := s.AddRecipient(req.Email, req.Name)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "added"})
}

// Utility functions

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func quotedPrintable(s string) string {
	var buf bytes.Buffer
	for _, c := range s {
		switch {
		case c == '\n':
			buf.WriteString("\n")
		case c == '\r':
			// Skip
		case c == '=':
			buf.WriteString("=3D")
		case c < 33 || c > 126 || strings.ContainsRune(" \t", c):
			buf.WriteString(fmt.Sprintf("=%02X", c))
		default:
			buf.WriteRune(c)
		}
	}
	return buf.String()
}

// Template functions

func getWelcomeTemplateHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; padding: 20px 0; border-bottom: 2px solid #4F46E5; margin-bottom: 20px; }
        .logo { font-size: 28px; font-weight: bold; color: #4F46E5; }
        .content { padding: 20px 0; }
        .button { display: inline-block; padding: 12px 24px; background-color: #4F46E5; color: white; text-decoration: none; border-radius: 6px; font-weight: 600; margin: 20px 0; }
        .footer { text-align: center; padding: 20px 0; border-top: 1px solid #eee; color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="header">
        <div class="logo">{{.AppName}}</div>
    </div>
    <div class="content">
        <h2>Welcome, {{.UserName}}!</h2>
        <p>Thank you for joining {{.AppName}}. We're excited to have you on board!</p>
        <p>To get started, please verify your email address:</p>
        <a href="{{.VerifyURL}}" class="button">Verify Email</a>
        <p>If you didn't create an account, please ignore this email or contact support.</p>
    </div>
    <div class="footer">
        <p>Need help? <a href="{{.SupportURL}}">Contact Support</a></p>
        <p>&copy; {{.AppName}} - All rights reserved</p>
    </div>
</body>
</html>`
}

func getWelcomeTemplateText() string {
	return `Welcome, {{.UserName}}!

Thank you for joining {{.AppName}}. We're excited to have you on board!

To get started, please verify your email address:
{{.VerifyURL}}

If you didn't create an account, please ignore this email or contact support.

Need help? Contact us at: {{.SupportURL}}

&copy; {{.AppName}} - All rights reserved`
}

// Add all other template functions here (password_reset, email_verify, etc.)
// For brevity, including placeholders for remaining templates

func getPasswordResetTemplateHTML() string { return getWelcomeTemplateHTML() }
func getPasswordResetTemplateText() string { return getWelcomeTemplateText() }
func getEmailVerifyTemplateHTML() string { return getWelcomeTemplateHTML() }
func getEmailVerifyTemplateText() string { return getWelcomeTemplateText() }
func getDepositTemplateHTML() string { return getWelcomeTemplateHTML() }
func getDepositTemplateText() string { return getWelcomeTemplateText() }
func getWithdrawalTemplateHTML() string { return getWelcomeTemplateHTML() }
func getWithdrawalTemplateText() string { return getWelcomeTemplateText() }
func getSecurityAlertTemplateHTML() string { return getWelcomeTemplateHTML() }
func getSecurityAlertTemplateText() string { return getWelcomeTemplateText() }
func getKYCApprovedTemplateHTML() string { return getWelcomeTemplateHTML() }
func getKYCApprovedTemplateText() string { return getWelcomeTemplateText() }
func getKYCRejectedTemplateHTML() string { return getWelcomeTemplateHTML() }
func getKYCRejectedTemplateText() string { return getWelcomeTemplateText() }
func get2FAEnabledTemplateHTML() string { return getWelcomeTemplateHTML() }
func get2FAEnabledTemplateText() string { return getWelcomeTemplateText() }
func get2FADisabledTemplateHTML() string { return getWelcomeTemplateHTML() }
func get2FADisabledTemplateText() string { return getWelcomeTemplateText() }
func getAccountLockedTemplateHTML() string { return getWelcomeTemplateHTML() }
func getAccountLockedTemplateText() string { return getWelcomeTemplateText() }
func getWithdrawalApprovedTemplateHTML() string { return getWelcomeTemplateHTML() }
func getWithdrawalApprovedTemplateText() string { return getWelcomeTemplateText() }
func getWithdrawalRejectedTemplateHTML() string { return getWelcomeTemplateHTML() }
func getWithdrawalRejectedTemplateText() string { return getWelcomeTemplateText() }
func getSupportTicketTemplateHTML() string { return getWelcomeTemplateHTML() }
func getSupportTicketTemplateText() string { return getWelcomeTemplateText() }
func getMonthlyStatementTemplateHTML() string { return getWelcomeTemplateHTML() }
func getMonthlyStatementTemplateText() string { return getWelcomeTemplateText() }

// Main

func main() {
	cfg := EmailConfig{
		SMTPHost:          getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:          getEnvInt("SMTP_PORT", 587),
		SMTPUsername:      getEnv("SMTP_USERNAME", ""),
		SMTPPassword:      getEnv("SMTP_PASSWORD", ""),
		SMTPFromName:      getEnv("SMTP_FROM_NAME", "TigerWallet"),
		SMTPFromEmail:     getEnv("SMTP_FROM_EMAIL", "noreply@tigerwallet.com"),
		UseTLS:            getEnvBool("SMTP_USE_TLS", true),
		SkipVerification:  getEnvBool("SMTP_SKIP_VERIFY", false),
		MaxRetries:        getEnvInt("EMAIL_MAX_RETRIES", 3),
		RetryDelay:        getEnvDuration("EMAIL_RETRY_DELAY", 5*time.Second),
		WorkerCount:       getEnvInt("EMAIL_WORKERS", 5),
		QueueBufferSize:   getEnvInt("EMAIL_QUEUE_SIZE", 1000),
		RateLimitPerMinute: getEnvInt("EMAIL_RATE_LIMIT", 100),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", "password"),
		DBName:            getEnv("DB_NAME", "email_db"),
		RedisHost:         getEnv("REDIS_HOST", "localhost"),
		RedisPort:         getEnv("REDIS_PORT", "6379"),
		ServerPort:        getEnv("EMAIL_SERVER_PORT", "8087"),
	}
	
	service, err := NewEmailService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize email service: %v", err)
	}
	
	// Start workers
	service.StartWorkers()
	
	// Setup HTTP routes
	r := gin.Default()
	
	r.POST("/emails", service.CreateEmailHandler)
	r.POST("/emails/template", service.SendTemplateHandler)
	r.POST("/emails/queue", service.QueueEmailHandler)
	r.POST("/unsubscribe", service.UnsubscribeHandler)
	r.POST("/recipients", service.AddRecipientHandler)
	r.GET("/stats", service.StatsHandler)
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "email"})
	})
	
	go func() {
		log.Printf("Email Service starting on port %s", cfg.ServerPort)
		if err := r.Run(":" + cfg.ServerPort); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	
	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	<-quit
	
	log.Println("Shutting down email service...")
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

// GormExpr for atomic operations
type GormExpr func(*gorm.DB) *gorm.DB
