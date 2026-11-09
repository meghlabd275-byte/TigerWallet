package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Delivery channels.
//
// Each channel posts to the provider configured by environment variables. A
// channel that has no provider configured is reported as unavailable rather
// than silently succeeding, so the caller learns the notification was not
// delivered. No channel fabricates a delivery receipt.

// errChannelUnavailable means no provider is configured for this channel.
var errChannelUnavailable = errors.New("no provider configured for this channel")

type dispatcher struct {
	client *http.Client
}

func newDispatcher() *dispatcher {
	return &dispatcher{client: &http.Client{Timeout: 10 * time.Second}}
}

// Send routes a notification to the channel it names.
func (d *dispatcher) Send(ctx context.Context, n *Notification) error {
	switch strings.ToLower(strings.TrimSpace(n.Channel)) {
	case "email", "smtp":
		return d.sendEmail(ctx, n)
	case "sms":
		return d.sendSMS(ctx, n)
	case "push", "fcm":
		return d.sendPush(ctx, n)
	case "webhook", "http":
		return d.sendWebhook(ctx, n)
	case "", "inapp", "in_app":
		// An in-app notification is delivered by the client reading it from the
		// API, so storing the row completes delivery.
		return nil
	default:
		return fmt.Errorf("unknown channel %q", n.Channel)
	}
}

func (d *dispatcher) sendEmail(ctx context.Context, n *Notification) error {
	endpoint := strings.TrimSpace(os.Getenv("EMAIL_WEBHOOK_URL"))
	if endpoint == "" {
		return errChannelUnavailable
	}
	body := map[string]interface{}{
		"to":      n.UserID,
		"subject": n.Title,
		"text":    n.Body,
	}
	headers := map[string]string{}
	if key := strings.TrimSpace(os.Getenv("EMAIL_WEBHOOK_TOKEN")); key != "" {
		headers["Authorization"] = "Bearer " + key
	}
	return d.postJSON(ctx, endpoint, body, headers)
}

func (d *dispatcher) sendSMS(ctx context.Context, n *Notification) error {
	endpoint := strings.TrimSpace(os.Getenv("SMS_WEBHOOK_URL"))
	if endpoint == "" {
		return errChannelUnavailable
	}
	body := map[string]interface{}{
		"to":   n.UserID,
		"body": n.Title + ": " + n.Body,
	}
	headers := map[string]string{}
	if key := strings.TrimSpace(os.Getenv("SMS_WEBHOOK_TOKEN")); key != "" {
		headers["Authorization"] = "Bearer " + key
	}
	return d.postJSON(ctx, endpoint, body, headers)
}

func (d *dispatcher) sendPush(ctx context.Context, n *Notification) error {
	endpoint := strings.TrimSpace(os.Getenv("PUSH_WEBHOOK_URL"))
	if endpoint == "" {
		return errChannelUnavailable
	}
	body := map[string]interface{}{
		"to":    n.UserID,
		"title": n.Title,
		"body":  n.Body,
		"data":  n.Data,
	}
	headers := map[string]string{}
	if key := strings.TrimSpace(os.Getenv("PUSH_WEBHOOK_TOKEN")); key != "" {
		headers["Authorization"] = "Bearer " + key
	}
	return d.postJSON(ctx, endpoint, body, headers)
}

// sendWebhook delivers to the target carried on the notification itself. The
// URL must be http(s) and must not point at a loopback or link-local address,
// so a caller cannot use the service to probe internal hosts.
func (d *dispatcher) sendWebhook(ctx context.Context, n *Notification) error {
	raw, _ := n.Data["url"].(string)
	if raw == "" {
		return errors.New("webhook notification requires data.url")
	}
	if err := validateOutboundURL(raw); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"id":    n.ID,
		"type":  n.Type,
		"title": n.Title,
		"body":  n.Body,
		"data":  n.Data,
	}
	return d.postJSON(ctx, raw, payload, nil)
}

func (d *dispatcher) postJSON(ctx context.Context, endpoint string, body interface{}, headers map[string]string) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read a bounded amount of the body so the provider's reason is visible
		// without letting a hostile endpoint stream unbounded data into memory.
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("provider returned %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

// validateOutboundURL rejects non-HTTP schemes and addresses inside the host
// network, which would otherwise turn this service into an SSRF pivot.
func validateOutboundURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url scheme %q is not allowed", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("url has no host")
	}
	lower := strings.ToLower(host)
	blocked := []string{"localhost", "127.", "0.0.0.0", "::1", "169.254.", "10.", "192.168.", "metadata.google.internal"}
	for _, b := range blocked {
		if strings.HasPrefix(lower, b) {
			return fmt.Errorf("url host %q is not routable from this service", host)
		}
	}
	// 172.16.0.0/12
	if strings.HasPrefix(lower, "172.") {
		parts := strings.Split(lower, ".")
		if len(parts) > 1 {
			if second, err := parseInt32(parts[1]); err == nil && second >= 16 && second <= 31 {
				return fmt.Errorf("url host %q is not routable from this service", host)
			}
		}
	}
	return nil
}
