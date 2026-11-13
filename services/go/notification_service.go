//go:build ignore

// Standalone reference/demo service. Run individually with: go run <file>
// (Tagged "ignore" so the services/go directory is not a broken package —
//  these files are not part of any deployed build; deployed services live
//  under their own modules, e.g. go/*, */go.)
// TigerSwap Notification Service - Production-Ready Implementation
// Email, SMS, Push, Telegram, Discord webhooks
// Multi-channel notification management with templates

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

type NotificationType string

const (
	NotificationEmail    NotificationType = "email"
	NotificationSMS      NotificationType = "sms"
	NotificationPush     NotificationType = "push"
	NotificationTelegram NotificationType = "telegram"
	NotificationDiscord  NotificationType = "discord"
	NotificationWebhook  NotificationType = "webhook"
)

type NotificationPriority string

const (
	PriorityLow    NotificationPriority = "low"
	PriorityNormal NotificationPriority = "normal"
	PriorityHigh   NotificationPriority = "high"
	PriorityUrgent NotificationPriority = "urgent"
)

type Notification struct {
	ID          string                 `json:"id"`
	Type        NotificationType       `json:"type"`
	Priority    NotificationPriority   `json:"priority"`
	Recipient   string                 `json:"recipient"`
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body"`
	HTMLBody    string                 `json:"html_body,omitempty"`
	TemplateID  string                 `json:"template_id,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	ScheduledAt int64                  `json:"scheduled_at,omitempty"`
	SentAt      int64                  `json:"sent_at,omitempty"`
	Status      string                 `json:"status"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   int64                  `json:"created_at"`
}

type EmailConfig struct {
	SMTPHost    string `json:"smtp_host"`
	SMTPPort    int    `json:"smtp_port"`
	SMTPUseTLS  bool   `json:"smtp_use_tls"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	FromAddress string `json:"from_address"`
	FromName    string `json:"from_name"`
}

type SMSConfig struct {
	Provider    string `json:"provider"`
	APIKey      string `json:"api_key"`
	APIEndpoint string `json:"api_endpoint"`
	FromNumber  string `json:"from_number"`
}

type PushConfig struct {
	Provider    string `json:"provider"`
	APIKey      string `json:"api_key"`
	ProjectID   string `json:"project_id"`
	APIEndpoint string `json:"api_endpoint"`
}

type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type DiscordConfig struct {
	WebhookURL string `json:"webhook_url"`
	Username   string `json:"username,omitempty"`
}

type WebhookConfig struct {
	URL     string            `json:"url"`
	Secret  string            `json:"secret,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ============================================================================
// Notification Service
// ============================================================================

type NotificationService struct {
	emailConfig    *EmailConfig
	smsConfig      *SMSConfig
	pushConfig     *PushConfig
	telegramConfig *TelegramConfig
	discordConfig  *DiscordConfig
	webhookConfigs map[string]*WebhookConfig

	queue          chan *Notification
	processedCount int64
	failedCount    int64
	httpClient     *http.Client

	mu        sync.RWMutex
	templates map[string]*Template
}

type Template struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	HTMLBody string `json:"html_body,omitempty"`
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		queue:          make(chan *Notification, 1000),
		templates:      make(map[string]*Template),
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		webhookConfigs: make(map[string]*WebhookConfig),
	}
}

// ============================================================================
// Configuration
// ============================================================================

func (s *NotificationService) ConfigureEmail(config EmailConfig) {
	s.emailConfig = &config
}

func (s *NotificationService) ConfigureSMS(config SMSConfig) {
	s.smsConfig = &config
}

func (s *NotificationService) ConfigurePush(config PushConfig) {
	s.pushConfig = &config
}

func (s *NotificationService) ConfigureTelegram(config TelegramConfig) {
	s.telegramConfig = &config
}

func (s *NotificationService) ConfigureDiscord(config DiscordConfig) {
	s.discordConfig = &config
}

func (s *NotificationService) AddWebhookConfig(name string, config WebhookConfig) {
	s.webhookConfigs[name] = &config
}

// ============================================================================
// Templates
// ============================================================================

func (s *NotificationService) AddTemplate(template *Template) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templates[template.ID] = template
}

func (s *NotificationService) GetTemplate(id string) *Template {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.templates[id]
}

func (s *NotificationService) RenderTemplate(templateID string, data map[string]interface{}) (string, string, error) {
	template := s.GetTemplate(templateID)
	if template == nil {
		return "", "", fmt.Errorf("template not found: %s", templateID)
	}

	subject := s.interpolate(template.Subject, data)
	body := s.interpolate(template.Body, data)
	htmlBody := ""
	if template.HTMLBody != "" {
		htmlBody = s.interpolate(template.HTMLBody, data)
	}

	return subject, body, htmlBody
}

func (s *NotificationService) interpolate(text string, data map[string]interface{}) string {
	result := text
	for key, value := range data {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// ============================================================================
// Send Methods
// ============================================================================

func (s *NotificationService) SendEmail(to, subject, body, htmlBody string) error {
	if s.emailConfig == nil {
		return fmt.Errorf("email not configured")
	}

	notification := &Notification{
		ID:        generateID("email"),
		Type:      NotificationEmail,
		Recipient: to,
		Subject:   subject,
		Body:      body,
		HTMLBody:  htmlBody,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
	}

	return s.sendEmailNotification(notification)
}

// sendEmailNotification delivers a message over SMTP. The notification is only
// marked sent when the server accepts it for delivery.
func (s *NotificationService) sendEmailNotification(n *Notification) error {
	if s.emailConfig == nil {
		return fmt.Errorf("email not configured")
	}

	cfg := s.emailConfig
	if cfg.SMTPHost == "" {
		n.Status = "failed"
		n.Error = "SMTP host is not configured"
		return fmt.Errorf("%s", n.Error)
	}

	from := cfg.FromAddress
	if from == "" {
		from = cfg.Username
	}
	if from == "" {
		n.Status = "failed"
		n.Error = "no sender address configured"
		return fmt.Errorf("%s", n.Error)
	}

	headers := map[string]string{
		"From":         formatMailbox(cfg.FromName, from),
		"To":           n.Recipient,
		"Subject":      n.Subject,
		"MIME-Version": "1.0",
	}

	var body string
	if n.HTMLBody != "" {
		headers["Content-Type"] = "text/html; charset=UTF-8"
		body = n.HTMLBody
	} else {
		headers["Content-Type"] = "text/plain; charset=UTF-8"
		body = n.Body
	}

	message := buildMIME(headers, body)
	addr := net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort))

	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, from, []string{n.Recipient}, []byte(message)); err != nil {
		n.Status = "failed"
		n.Error = err.Error()
		return fmt.Errorf("send email: %w", err)
	}

	n.Status = "sent"
	n.SentAt = time.Now().Unix()
	return nil
}

// formatMailbox renders a From header, quoting the display name when present.
// headerSafe strips CR and LF so a value cannot break out of its header and
// inject additional ones. Every header value passes through here.
func headerSafe(v string) string {
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", "")
	return v
}

func formatMailbox(name, address string) string {
	name = headerSafe(name)
	address = headerSafe(address)
	if strings.TrimSpace(name) == "" {
		return address
	}
	escaped := strings.ReplaceAll(name, `"`, `\"`)
	return fmt.Sprintf("%q <%s>", escaped, address)
}

// buildMIME assembles a message from headers and a body.
func buildMIME(headers map[string]string, body string) string {
	var b strings.Builder
	// A deterministic order keeps the output stable across runs.
	for _, key := range []string{"From", "To", "Subject", "MIME-Version", "Content-Type"} {
		if v, ok := headers[key]; ok {
			b.WriteString(headerSafe(key))
			b.WriteString(": ")
			b.WriteString(headerSafe(v))
			b.WriteString("\r\n")
		}
	}
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}

func (s *NotificationService) SendSMS(to, message string) error {
	if s.smsConfig == nil {
		return fmt.Errorf("SMS not configured")
	}

	notification := &Notification{
		ID:        generateID("sms"),
		Type:      NotificationSMS,
		Recipient: to,
		Body:      message,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
	}

	return s.sendSMSNotification(notification)
}

// sendSMSNotification delivers a message through the configured SMS provider
// endpoint. The notification is only marked sent on a 2xx response.
func (s *NotificationService) sendSMSNotification(n *Notification) error {
	if s.smsConfig == nil {
		return fmt.Errorf("SMS not configured")
	}
	if strings.TrimSpace(s.smsConfig.APIEndpoint) == "" {
		n.Status = "failed"
		n.Error = "SMS API endpoint is not configured"
		return fmt.Errorf("%s", n.Error)
	}
	if !strings.HasPrefix(s.smsConfig.APIEndpoint, "https://") {
		n.Status = "failed"
		n.Error = "SMS API endpoint must use https"
		return fmt.Errorf("%s", n.Error)
	}

	payload := map[string]interface{}{
		"from":    s.smsConfig.FromNumber,
		"to":      n.Recipient,
		"message": n.Body,
	}
	if s.smsConfig.Provider != "" {
		payload["provider"] = s.smsConfig.Provider
	}

	data, err := json.Marshal(payload)
	if err != nil {
		n.Status = "failed"
		n.Error = err.Error()
		return err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, s.smsConfig.APIEndpoint, bytes.NewReader(data))
	if err != nil {
		n.Status = "failed"
		n.Error = err.Error()
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.smsConfig.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.smsConfig.APIKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		n.Status = "failed"
		n.Error = err.Error()
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		n.Status = "failed"
		n.Error = fmt.Sprintf("SMS provider returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
		return fmt.Errorf("%s", n.Error)
	}

	n.Status = "sent"
	n.SentAt = time.Now().Unix()
	return nil
}

func (s *NotificationService) SendPush(token, title, body string, data map[string]interface{}) error {
	if s.pushConfig == nil {
		return fmt.Errorf("push not configured")
	}

	notification := &Notification{
		ID:        generateID("push"),
		Type:      NotificationPush,
		Recipient: token,
		Subject:   title,
		Body:      body,
		Data:      data,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
	}

	return s.sendPushNotification(notification)
}

// sendPushNotification delivers a push message through the configured push
// provider endpoint. The notification is only marked sent on a 2xx response.
func (s *NotificationService) sendPushNotification(n *Notification) error {
	if s.pushConfig == nil {
		return fmt.Errorf("push not configured")
	}
	if strings.TrimSpace(s.pushConfig.APIEndpoint) == "" {
		n.Status = "failed"
		n.Error = "push API endpoint is not configured"
		return fmt.Errorf("%s", n.Error)
	}
	if !strings.HasPrefix(s.pushConfig.APIEndpoint, "https://") {
		n.Status = "failed"
		n.Error = "push API endpoint must use https"
		return fmt.Errorf("%s", n.Error)
	}

	payload := map[string]interface{}{
		"token": n.Recipient,
		"title": n.Subject,
		"body":  n.Body,
	}
	if n.Data != nil {
		payload["data"] = n.Data
	}
	if s.pushConfig.ProjectID != "" {
		payload["project_id"] = s.pushConfig.ProjectID
	}

	data, err := json.Marshal(payload)
	if err != nil {
		n.Status = "failed"
		n.Error = err.Error()
		return err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, s.pushConfig.APIEndpoint, bytes.NewReader(data))
	if err != nil {
		n.Status = "failed"
		n.Error = err.Error()
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.pushConfig.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.pushConfig.APIKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		n.Status = "failed"
		n.Error = err.Error()
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		n.Status = "failed"
		n.Error = fmt.Sprintf("push provider returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
		return fmt.Errorf("%s", n.Error)
	}

	n.Status = "sent"
	n.SentAt = time.Now().Unix()
	return nil
}

func (s *NotificationService) SendTelegram(message string) error {
	if s.telegramConfig == nil {
		return fmt.Errorf("telegram not configured")
	}

	notification := &Notification{
		ID:        generateID("telegram"),
		Type:      NotificationTelegram,
		Recipient: s.telegramConfig.ChatID,
		Body:      message,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
	}

	return s.sendTelegramNotification(notification)
}

func (s *NotificationService) sendTelegramNotification(n *Notification) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.telegramConfig.BotToken)

	payload := map[string]interface{}{
		"chat_id": n.Recipient,
		"text":    n.Body,
	}

	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		n.Status = "failed"
		n.Error = err.Error()
		return err
	}
	defer resp.Body.Close()

	n.Status = "sent"
	n.SentAt = time.Now().Unix()

	return nil
}

func (s *NotificationService) SendDiscord(message string, embeds []DiscordEmbed) error {
	if s.discordConfig == nil {
		return fmt.Errorf("discord not configured")
	}

	notification := &Notification{
		ID:        generateID("discord"),
		Type:      NotificationDiscord,
		Body:      message,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
	}

	return s.sendDiscordNotification(notification, embeds)
}

type DiscordEmbed struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Color       int    `json:"color,omitempty"`
	Footer      string `json:"footer,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

func (s *NotificationService) sendDiscordNotification(n *Notification, embeds []DiscordEmbed) error {
	payload := map[string]interface{}{
		"content": n.Body,
	}

	if len(embeds) > 0 {
		payload["embeds"] = embeds
	}

	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", s.discordConfig.WebhookURL, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		n.Status = "failed"
		n.Error = err.Error()
		return err
	}
	defer resp.Body.Close()

	n.Status = "sent"
	n.SentAt = time.Now().Unix()

	return nil
}

func (s *NotificationService) SendWebhook(name string, payload interface{}) error {
	s.mu.RLock()
	config, ok := s.webhookConfigs[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("webhook config not found: %s", name)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, _ := http.NewRequest("POST", config.URL, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	if config.Secret != "" {
		req.Header.Set("X-Webhook-Secret", config.Secret)
	}

	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook error: %s", string(body))
	}

	return nil
}

// ============================================================================
// Queue Processing
// ============================================================================

func (s *NotificationService) Enqueue(n *Notification) {
	s.queue <- n
}

func (s *NotificationService) StartWorker(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		go s.processQueue(ctx)
	}
}

func (s *NotificationService) processQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case n := <-s.queue:
			s.processNotification(n)
		}
	}
}

func (s *NotificationService) processNotification(n *Notification) {
	var err error

	switch n.Type {
	case NotificationEmail:
		err = s.sendEmailNotification(n)
	case NotificationSMS:
		err = s.sendSMSNotification(n)
	case NotificationPush:
		err = s.sendPushNotification(n)
	case NotificationTelegram:
		err = s.sendTelegramNotification(n)
	case NotificationDiscord:
		err = s.sendDiscordNotification(n, nil)
	}

	if err != nil {
		s.mu.Lock()
		s.failedCount++
		s.mu.Unlock()
	} else {
		s.mu.Lock()
		s.processedCount++
		s.mu.Unlock()
	}
}

// ============================================================================
// Stats
// ============================================================================

func (s *NotificationService) GetStats() NotificationStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return NotificationStats{
		ProcessedCount: s.processedCount,
		FailedCount:    s.failedCount,
		QueueLength:    len(s.queue),
	}
}

type NotificationStats struct {
	ProcessedCount int64 `json:"processed_count"`
	FailedCount    int64 `json:"failed_count"`
	QueueLength    int   `json:"queue_length"`
}

// ============================================================================
// Helpers
// ============================================================================

func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.Println("Starting TigerSwap Notification Service...")

	service := NewNotificationService()

	// Configure Discord webhook (example)
	service.ConfigureDiscord(DiscordConfig{
		WebhookURL: "https://discord.com/api/webhooks/...",
		Username:   "TigerSwap Bot",
	})

	// Add templates
	service.AddTemplate(&Template{
		ID:      "trade_executed",
		Name:    "Trade Executed",
		Subject: "Your {{pair}} trade was executed",
		Body:    "Your {{side}} order for {{amount}} {{pair}} was executed at {{price}}",
	})

	// Send a test notification
	err := service.SendDiscord("TigerSwap Notification Service started", []DiscordEmbed{
		{
			Title:       "Service Status",
			Description: "Notification service is running",
			Color:       3066993, // Green
			Timestamp:   time.Now().Format(time.RFC3339),
		},
	})

	if err != nil {
		log.Printf("Error sending notification: %v", err)
	} else {
		log.Println("Test notification sent successfully")
	}

	// Start worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service.StartWorker(ctx, 3)

	log.Println("Notification Service running...")
	time.Sleep(10 * time.Second)
}
