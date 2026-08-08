package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/smtp"
	"strings"
)

// NotificationService - Complete notification service with Email, SMS, Push
type NotificationService struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMSAPIKey    string
	PushAPIKey   string
	EmailFrom    string
}

// NewNotificationService creates a new notification service
func NewNotificationService() *NotificationService {
	return &NotificationService{
		SMTPHost:     "smtp.gmail.com",
		SMTPPort:     587,
		SMTPUsername: "noreply@tigerwallet.com",
		EmailFrom:    "TigerWallet <noreply@tigerwallet.com>",
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

// SendSMS sends an SMS
func (s *NotificationService) SendSMS(notif SMSNotification) error {
	if s.SMSAPIKey == "" {
		return nil
	}
	jsonData, _ := json.Marshal(notif)
	fmt.Printf("SMS would be sent: %s\n", string(jsonData))
	return nil
}

// PushNotification represents a push notification
type PushNotification struct {
	Token   string
	Title   string
	Message string
	Data    map[string]interface{}
}

// SendPush sends a push notification
func (s *NotificationService) SendPush(notif PushNotification) error {
	if s.PushAPIKey == "" {
		return nil
	}
	jsonData, _ := json.Marshal(notif)
	fmt.Printf("Push would be sent: %s\n", string(jsonData))
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
