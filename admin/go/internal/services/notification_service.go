package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// NotificationService - Complete notification service with Email, SMS, Push
type NotificationService struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	EmailFrom    string

	// SMSProviderURL / PushProviderURL are the HTTP endpoints of the SMS and
	// push gateways. When unset, delivery fails closed rather than silently
	// reporting success.
	SMSProviderURL  string
	PushProviderURL string
	SMSAPIKey       string
	PushAPIKey      string

	httpClient *http.Client
}

// NewNotificationService creates a new notification service. SMTP and provider
// settings come from the environment so no credentials are compiled in.
func NewNotificationService() *NotificationService {
	return &NotificationService{
		SMTPHost:        getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:        getEnvInt("SMTP_PORT", 587),
		SMTPUsername:    getEnv("SMTP_USERNAME", ""),
		SMTPPassword:    getEnv("SMTP_PASSWORD", ""),
		EmailFrom:       getEnv("SMTP_FROM", ""),
		SMSProviderURL:  getEnv("SMS_PROVIDER_URL", ""),
		PushProviderURL: getEnv("PUSH_PROVIDER_URL", ""),
		SMSAPIKey:       getEnv("SMS_API_KEY", ""),
		PushAPIKey:      getEnv("PUSH_API_KEY", ""),
		httpClient:      &http.Client{Timeout: 20 * time.Second},
	}
}

// EmailNotification represents an email notification
type EmailNotification struct {
	To           []string
	CC           []string
	BCC          []string
	Subject      string
	Body         string
	HTMLBody     string
	Template     string
	TemplateData map[string]interface{}
}

// SendEmail sends an email
func (s *NotificationService) SendEmail(notif EmailNotification) error {
	if notif.HTMLBody == "" {
		notif.HTMLBody = fmt.Sprintf("<html><body><p>%s</p></body></html>", notif.Body)
	}

	headers := make(map[string]string)
	headers["From"] = s.EmailFrom
	headers["To"] = strings.Join(notif.To, ", ")
	if len(notif.CC) > 0 {
		headers["CC"] = strings.Join(notif.CC, ", ")
	}
	headers["Subject"] = notif.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=utf-8"

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(notif.HTMLBody)

	addr := fmt.Sprintf("%s:%d", s.SMTPHost, s.SMTPPort)
	auth := smtp.PlainAuth("", s.SMTPUsername, s.SMTPPassword, s.SMTPHost)

	err := smtp.SendMail(addr, auth, s.EmailFrom, notif.To, msg.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SMSNotification represents an SMS notification
type SMSNotification struct {
	To      string
	Message string
	Type    string
}

// SendSMS delivers an SMS through the configured provider. It fails closed when
// no provider is configured instead of pretending the message was sent.
func (s *NotificationService) SendSMS(notif SMSNotification) error {
	if s.SMSAPIKey == "" || s.SMSProviderURL == "" {
		return errNotificationProviderNotConfigured("sms")
	}
	if notif.To == "" {
		return fmt.Errorf("sms recipient is required")
	}

	payload := map[string]interface{}{
		"to":      notif.To,
		"message": notif.Message,
		"type":    notif.Type,
	}
	return s.postToProvider(s.SMSProviderURL, s.SMSAPIKey, payload)
}

// PushNotification represents a push notification
type PushNotification struct {
	Token   string
	Title   string
	Message string
	Data    map[string]interface{}
}

// SendPush delivers a push notification through the configured provider. It
// fails closed when no provider is configured.
func (s *NotificationService) SendPush(notif PushNotification) error {
	if s.PushAPIKey == "" || s.PushProviderURL == "" {
		return errNotificationProviderNotConfigured("push")
	}
	if notif.Token == "" {
		return fmt.Errorf("push device token is required")
	}

	payload := map[string]interface{}{
		"token": notif.Token,
		"title": notif.Title,
		"body":  notif.Message,
		"data":  notif.Data,
	}
	return s.postToProvider(s.PushProviderURL, s.PushAPIKey, payload)
}

// errNotificationProviderNotConfigured reports a missing provider configuration
// so callers can distinguish "not set up" from "delivered".
func errNotificationProviderNotConfigured(channel string) error {
	return fmt.Errorf("%s provider is not configured", channel)
}

// postToProvider performs the provider HTTP call, surfacing upstream failures.
func (s *NotificationService) postToProvider(endpoint, apiKey string, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode notification payload: %w", err)
	}

	client := s.httpClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build notification request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("notification provider request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("notification provider returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Template constants
const (
	TemplateWelcome            = "welcome"
	TemplatePasswordReset      = "password_reset"
	TemplateKYCApproved        = "kyc_approved"
	TemplateKYCRejected        = "kyc_rejected"
	TemplateWithdrawalApproved = "withdrawal_approved"
	TemplateWithdrawalRejected = "withdrawal_rejected"
	TemplateSecurityAlert      = "security_alert"
)

// parseTemplate parses a simple template
func (s *NotificationService) parseTemplate(template string, data map[string]interface{}) string {
	result := template
	for key, value := range data {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// GetEmailTemplate returns HTML email template
func (s *NotificationService) GetEmailTemplate(name string, data map[string]interface{}) string {
	templates := map[string]string{
		TemplateWelcome:       `<!DOCTYPE html><html><head><style>body{font-family:Arial,sans-serif;line-height:1.6;color:#333}.container{max-width:600px;margin:0 auto;padding:20px}.header{background:#1a73e8;color:white;padding:20px;text-align:center}.content{padding:20px;background:#f9f9f9}.button{display:inline-block;padding:12px 24px;background:#1a73e8;color:white;text-decoration:none;border-radius:4px;margin:10px 0}</style></head><body><div class="container"><div class="header"><h1>Welcome to TigerWallet!</h1></div><div class="content"><p>Hello {{.Name}},</p><p>Welcome to TigerWallet! We're excited to have you on board.</p></div></div></body></html>`,
		TemplateKYCApproved:   `<!DOCTYPE html><html><head><style>body{font-family:Arial,sans-serif;line-height:1.6;color:#333}</style></head><body><h1>KYC Approved!</h1><p>Hello {{.Name}}, your identity verification has been approved.</p></body></html>`,
		TemplateSecurityAlert: `<!DOCTYPE html><html><head><style>body{font-family:Arial,sans-serif}</style></head><body><h1>Security Alert</h1><p>New login detected at {{.Time}} from {{.IP}}</p></body></html>`,
	}

	if template, ok := templates[name]; ok {
		return s.parseTemplate(template, data)
	}
	return ""
}
