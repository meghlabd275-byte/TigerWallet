package main

// delivery.go — real SMS and email delivery for verification codes.
//
// Both channels fail closed: when no provider is configured, delivery returns
// an error instead of reporting success, so a caller cannot mark a code as sent
// when nothing left the process. The code is never written to a log.

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	errSMSNotConfigured   = errors.New("SMS delivery is not configured (TWILIO_ACCOUNT_SID / TWILIO_AUTH_TOKEN / TWILIO_FROM_NUMBER)")
	errEmailNotConfigured = errors.New("email delivery is not configured (SMTP_USER / SMTP_PASS / FROM_EMAIL)")
)

// deliveryClient is shared so connections are pooled and every request has a
// bounded lifetime.
var deliveryClient = &http.Client{Timeout: 20 * time.Second}

// smsConfig holds the Twilio credentials read from the environment.
type smsConfig struct {
	AccountSID string
	AuthToken  string
	FromNumber string
}

// loadSMSConfig returns the SMS configuration, or an error when any part of it
// is missing. Partial configuration is treated as unconfigured rather than
// attempted, because a half-configured provider always fails at send time.
func loadSMSConfig() (*smsConfig, error) {
	cfg := &smsConfig{
		AccountSID: strings.TrimSpace(os.Getenv("TWILIO_ACCOUNT_SID")),
		AuthToken:  strings.TrimSpace(os.Getenv("TWILIO_AUTH_TOKEN")),
		FromNumber: strings.TrimSpace(os.Getenv("TWILIO_FROM_NUMBER")),
	}
	if cfg.AccountSID == "" || cfg.AuthToken == "" || cfg.FromNumber == "" {
		return nil, errSMSNotConfigured
	}
	return cfg, nil
}

// sendSMS delivers a message through Twilio and returns the provider's message
// id. Any transport, authentication or provider error is returned to the
// caller.
func sendSMS(to, message string) (string, error) {
	cfg, err := loadSMSConfig()
	if err != nil {
		return "", err
	}

	to = normalizePhone(to)
	if to == "" {
		return "", errors.New("invalid destination phone number")
	}

	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json",
		url.PathEscape(cfg.AccountSID))

	form := url.Values{}
	form.Set("To", to)
	form.Set("From", cfg.FromNumber)
	form.Set("Body", message)

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build SMS request: %w", err)
	}
	req.SetBasicAuth(cfg.AccountSID, cfg.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := deliveryClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Twilio returns a JSON body describing the failure. Only a bounded
		// prefix is surfaced so a provider error page cannot flood the log.
		return "", fmt.Errorf("SMS provider returned HTTP %d: %s", resp.StatusCode, readBounded(resp.Body, 512))
	}

	return readBounded(resp.Body, 4096), nil
}

// emailConfig holds the SMTP settings read from the environment.
type emailConfig struct {
	Host      string
	Port      int
	User      string
	Pass      string
	FromEmail string
}

// loadEmailConfig returns the SMTP configuration, or an error when the
// credentials needed to authenticate are missing.
func loadEmailConfig() (*emailConfig, error) {
	cfg := &emailConfig{
		Host:      strings.TrimSpace(getEnv("SMTP_SERVER", "smtp.gmail.com")),
		Port:      587,
		User:      strings.TrimSpace(os.Getenv("SMTP_USER")),
		Pass:      strings.TrimSpace(os.Getenv("SMTP_PASS")),
		FromEmail: strings.TrimSpace(getEnv("FROM_EMAIL", "")),
	}
	if raw := strings.TrimSpace(os.Getenv("SMTP_PORT")); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid SMTP_PORT %q", raw)
		}
		cfg.Port = port
	}
	if cfg.User == "" || cfg.Pass == "" || cfg.FromEmail == "" {
		return nil, errEmailNotConfigured
	}
	return cfg, nil
}

// sendEmail delivers a message over SMTP with STARTTLS, requiring a verified
// server certificate and hostname. Without the hostname check a machine-in-the
// -middle could present any valid certificate and read the code.
func sendEmail(to, subject, body string) error {
	cfg, err := loadEmailConfig()
	if err != nil {
		return err
	}
	if !strings.Contains(to, "@") {
		return errors.New("invalid destination email address")
	}
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		// A newline in a header would allow header injection.
		return errors.New("destination and subject must not contain line breaks")
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))

	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("start SMTP session: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return errors.New("SMTP server does not offer STARTTLS; refusing to send credentials in the clear")
	}
	if err := client.StartTLS(&tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
		return fmt.Errorf("start TLS: %w", err)
	}

	if err := client.Auth(smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}
	if err := client.Mail(cfg.FromEmail); err != nil {
		return fmt.Errorf("set envelope sender: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("set envelope recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open message body: %w", err)
	}

	message := buildMessage(cfg.FromEmail, to, subject, body)
	if _, err := writer.Write([]byte(message)); err != nil {
		writer.Close()
		return fmt.Errorf("write message body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize message body: %w", err)
	}

	return client.Quit()
}

// buildMessage assembles an RFC 5322 message with a quoted-printable-safe body
// and CRLF line endings.
func buildMessage(from, to, subject, body string) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		b.WriteString(line)
		b.WriteString("\r\n")
	}
	return b.String()
}

// normalizePhone strips formatting and returns an E.164 number, or an empty
// string when the input cannot be interpreted.
func normalizePhone(raw string) string {
	var digits strings.Builder
	for i, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == '+' && i == 0:
			digits.WriteRune(r)
		case r == ' ' || r == '-' || r == '(' || r == ')' || r == '.':
		default:
			return ""
		}
	}
	number := digits.String()
	if !strings.HasPrefix(number, "+") {
		number = "+" + number
	}
	// E.164 allows at most 15 digits after the leading plus.
	if len(number) < 8 || len(number) > 16 {
		return ""
	}
	return number
}

// readBounded reads at most limit bytes so a hostile or misconfigured endpoint
// cannot exhaust memory.
func readBounded(r interface{ Read([]byte) (int, error) }, limit int) string {
	buf := make([]byte, 0, limit)
	chunk := make([]byte, 512)
	for len(buf) < limit {
		n, err := r.Read(chunk)
		if n > 0 {
			remaining := limit - len(buf)
			if n > remaining {
				n = remaining
			}
			buf = append(buf, chunk[:n]...)
		}
		if err != nil {
			break
		}
	}
	return strings.TrimSpace(string(buf))
}
