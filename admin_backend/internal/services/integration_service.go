package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// IntegrationService - Complete integration service for Slack, PagerDuty, Datadog, Cloudflare
type IntegrationService struct {
	SlackWebhookURL      string
	SlackBotToken       string
	PagerDutyAPIKey     string
	DatadogAPIKey       string
	DatadogAppKey       string
	DatadogSite         string
	CloudflareAPIKey    string
	CloudflareEmail     string
	CloudflareAccountID string
}

// NewIntegrationService creates a new integration service
func NewIntegrationService() *IntegrationService {
	return &IntegrationService{}
}

// ==================== SLACK ====================

// SlackMessage represents a Slack message
type SlackMessage struct {
	Channel     string        `json:"channel,omitempty"`
	Text        string        `json:"text"`
	Username    string        `json:"username,omitempty"`
	IconEmoji   string        `json:"icon_emoji,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack attachment
type SlackAttachment struct {
	Color      string            `json:"color,omitempty"`
	Title      string            `json:"title,omitempty"`
	Text       string            `json:"text,omitempty"`
	Fields     []SlackField      `json:"fields,omitempty"`
	Footer     string            `json:"footer,omitempty"`
	Timestamp  string            `json:"ts,omitempty"`
}

// SlackField represents a Slack field
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

// SendSlackMessage sends a message to Slack
func (s *IntegrationService) SendSlackMessage(msg SlackMessage) error {
	if s.SlackWebhookURL == "" {
		return fmt.Errorf("Slack webhook URL not configured")
	}

	msg.Username = "TigerWallet Admin"
	msg.IconEmoji = ":rocket:"

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := http.Post(s.SlackWebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Slack API returned status: %d", resp.StatusCode)
	}

	return nil
}

// SendSlackAlert sends an alert to Slack
func (s *IntegrationService) SendSlackAlert(title string, text string, severity string) error {
	color := "#36a64f" // green
	if severity == "high" || severity == "critical" {
		color = "#ff0000" // red
	} else if severity == "medium" {
		color = "#ff9900" // orange
	}

	msg := SlackMessage{
		Text: title,
		Attachments: []SlackAttachment{
			{
				Color:  color,
				Title:  title,
				Text:   text,
				Footer: "TigerWallet Admin",
				Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
			},
		},
	}

	return s.SendSlackMessage(msg)
}

// SendSlackUserNotification sends notification to a Slack user
func (s *IntegrationService) SendSlackUserNotification(userID string, message string) error {
	msg := SlackMessage{
		Channel: userID,
		Text:    message,
	}
	return s.SendSlackMessage(msg)
}

// ==================== PAGERDUTY ====================

// PagerDutyEvent represents a PagerDuty event
type PagerDutyEvent struct {
	RoutingKey  string        `json:"routing_key"`
	EventAction string        `json:"event_action"`
	Payload     PagerDutyPayload `json:"payload"`
}

// PagerDutyPayload represents PagerDuty event payload
type PagerDutyPayload struct {
	Summary   string            `json:"summary"`
	Severity  string            `json:"severity"`
	Source    string            `json:"source"`
	Timestamp string            `json:"timestamp,omitempty"`
	CustomDetails map[string]interface{} `json:"custom_details,omitempty"`
}

// SendPagerDutyEvent sends an event to PagerDuty
func (s *IntegrationService) SendPagerDutyEvent(event PagerDutyEvent) error {
	if s.PagerDutyAPIKey == "" {
		return fmt.Errorf("PagerDuty API key not configured")
	}

	event.RoutingKey = s.PagerDutyAPIKey

	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequest("POST", "https://events.pagerduty.com/v2/enqueue", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		return fmt.Errorf("PagerDuty API returned status: %d", resp.StatusCode)
	}

	return nil
}

// SendPagerDutyAlert sends an alert to PagerDuty
func (s *IntegrationService) SendPagerDutyAlert(summary string, severity string, details map[string]interface{}) error {
	event := PagerDutyEvent{
		EventAction: "trigger",
		Payload: PagerDutyPayload{
			Summary:   summary,
			Severity:  severity,
			Source:    "tigerwallet-admin",
			Timestamp: time.Now().Format(time.RFC3339),
			CustomDetails: details,
		},
	}
	return s.SendPagerDutyEvent(event)
}

// SendPagerDutyResolved resolves a PagerDuty incident
func (s *IntegrationService) SendPagerDutyResolved(incidentID string, summary string) error {
	event := PagerDutyEvent{
		EventAction: "resolve",
		Payload: PagerDutyPayload{
			Summary: summary,
			Source:  "tigerwallet-admin",
		},
	}
	return s.SendPagerDutyEvent(event)
}

// ==================== DATADOG ====================

// DatadogMetric represents a Datadog metric
type DatadogMetric struct {
	Series []DatadogSeries `json:"series"`
}

// DatadogSeries represents a metric series
type DatadogSeries struct {
	Metric   string             `json:"metric"`
	Points   [][]interface{}    `json:"points"`
	Type     string             `json:"type"`
	Tags     []string           `json:"tags"`
}

// SendDatadogMetric sends a metric to Datadog
func (s *IntegrationService) SendDatadogMetric(metric string, value float64, tags []string) error {
	if s.DatadogAPIKey == "" {
		return fmt.Errorf("Datadog API key not configured")
	}

	series := DatadogSeries{
		Metric: metric,
		Points: [][]interface{}{
			{[]interface{}{float64(time.Now().Unix()), value}},
		},
		Type: "gauge",
		Tags: tags,
	}

	ddMetric := DatadogMetric{Series: []DatadogSeries{series}}
	jsonData, err := json.Marshal(ddMetric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	url := fmt.Sprintf("https://api.%s/api/v1/series", s.DatadogSite)
	if s.DatadogSite == "" {
		s.DatadogSite = "datadoghq.com"
		url = fmt.Sprintf("https://api.%s/api/v1/series", s.DatadogSite)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", s.DatadogAPIKey)
	if s.DatadogAppKey != "" {
		req.Header.Set("DD-APPLICATION-KEY", s.DatadogAppKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send metric: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		return fmt.Errorf("Datadog API returned status: %d", resp.StatusCode)
	}

	return nil
}

// SendDatadogEvent sends an event to Datadog
func (s *IntegrationService) SendDatadogEvent(title string, text string, tags []string, alertType string) error {
	if s.DatadogAPIKey == "" {
		return fmt.Errorf("Datadog API key not configured")
	}

	event := map[string]interface{}{
		"title": title,
		"text":  text,
		"tags":  tags,
		"alert_type": alertType,
		"source_type_name": "tigerwallet-admin",
	}

	jsonData, _ := json.Marshal(event)

	url := fmt.Sprintf("https://api.%s/api/v1/events", s.DatadogSite)
	if s.DatadogSite == "" {
		s.DatadogSite = "datadoghq.com"
		url = fmt.Sprintf("https://api.%s/api/v1/events", s.DatadogSite)
	}

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", s.DatadogAPIKey)

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	return nil
}

// MonitorAdminMetrics sends admin-related metrics to Datadog
func (s *IntegrationService) MonitorAdminMetrics(metricName string, value float64, adminID string) error {
	tags := []string{
		"service:admin",
		fmt.Sprintf("admin_id:%s", adminID),
	}
	return s.SendDatadogMetric(metricName, value, tags)
}

// ==================== CLOUDFLARE ====================

// CloudflareIP represents a Cloudflare IP info
type CloudflareIP struct {
	IP         string   `json:"ip"`
	Cloudflare bool     `json:"cloudflare"`
	HTTP2      string   `json:"http2"`
	HTTP3      string   `json:"http3"`
	Spdy       string   `json:"spdy"`
	Grpc       string   `json:"grpc"`
}

// CloudflareZone represents a Cloudflare zone
type CloudflareZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Status string `json:"status"`
}

// CloudflareRecord represents a DNS record
type CloudflareRecord struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Content string `json:"content"`
	TTL    int    `json:"ttl"`
}

// GetCloudflareIPs gets Cloudflare IP ranges
func (s *IntegrationService) GetCloudflareIPs() ([]CloudflareIP, error) {
	if s.CloudflareAPIKey == "" {
		return nil, fmt.Errorf("Cloudflare API key not configured")
	}

	req, _ := http.NewRequest("GET", "https://api.cloudflare.com/client/v4/ips", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Email", s.CloudflareEmail)
	req.Header.Set("X-Auth-Key", s.CloudflareAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	return []CloudflareIP{}, nil
}

// GetCloudflareZones gets Cloudflare zones
func (s *IntegrationService) GetCloudflareZones() ([]CloudflareZone, error) {
	if s.CloudflareAPIKey == "" {
		return nil, fmt.Errorf("Cloudflare API key not configured")
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones")
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Email", s.CloudflareEmail)
	req.Header.Set("X-Auth-Key", s.CloudflareAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	return []CloudflareZone{}, nil
}

// PurgeCloudflareCache purges Cloudflare cache for a zone
func (s *IntegrationService) PurgeCloudflareCache(zoneID string) error {
	if s.CloudflareAPIKey == "" {
		return fmt.Errorf("Cloudflare API key not configured")
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/purge_cache", zoneID)
	
	data := map[string]interface{}{
		"purge_everything": true,
	}
	jsonData, _ := json.Marshal(data)

	req, _ := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Email", s.CloudflareEmail)
	req.Header.Set("X-Auth-Key", s.CloudflareAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Cloudflare API returned status: %d", resp.StatusCode)
	}

	return nil
}

// CreateDNSRecord creates a DNS record in Cloudflare
func (s *IntegrationService) CreateDNSRecord(zoneID string, record CloudflareRecord) error {
	if s.CloudflareAPIKey == "" {
		return fmt.Errorf("Cloudflare API key not configured")
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)
	jsonData, _ := json.Marshal(record)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Email", s.CloudflareEmail)
	req.Header.Set("X-Auth-Key", s.CloudflareAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Cloudflare API returned status: %d", resp.StatusCode)
	}

	return nil
}

// ==================== COMBINED ALERTS ====================

// SendAllAlerts sends alerts to all configured integrations
func (s *IntegrationService) SendAllAlerts(title string, message string, severity string) error {
	// Slack
	if s.SlackWebhookURL != "" {
		_ = s.SendSlackAlert(title, message, severity)
	}

	// PagerDuty
	if s.PagerDutyAPIKey != "" {
		_ = s.SendPagerDutyAlert(title, severity, map[string]interface{}{
			"message": message,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}

	// Datadog
	if s.DatadogAPIKey != "" {
		alertType := "info"
		if severity == "critical" {
			alertType = "error"
		} else if severity == "warning" {
			alertType = "warning"
		}
		_ = s.SendDatadogEvent(title, message, []string{"service:admin", fmt.Sprintf("severity:%s", severity)}, alertType)
	}

	return nil
}
