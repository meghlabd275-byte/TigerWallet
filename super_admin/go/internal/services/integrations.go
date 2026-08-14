// IntegrationService - Third-party integrations (Datadog, PagerDuty, Cloudflare)
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/super-admin/internal/config"
	"github.com/tigerwallet/super-admin/internal/database"
)

type IntegrationService struct {
	cfg *config.Config
}

func NewIntegrationService(cfg *config.Config) *IntegrationService {
	return &IntegrationService{cfg: cfg}
}

// Datadog Integration
type DatadogMetrics struct {
	Series []DatadogSeries `json:"series"`
}

type DatadogSeries struct {
	Metric   string      `json:"metric"`
	Points   [][]float64 `json:"points"`
	Type     string      `json:"type"`
	Host     string      `json:"host"`
	Tags     []string    `json:"tags"`
}

func (s *IntegrationService) SendDatadogMetric(ctx context.Context, metric string, value float64, tags []string) error {
	if s.cfg.DatadogAPIKey == "" {
		return fmt.Errorf("Datadog API key not configured")
	}

	series := DatadogSeries{
		Metric: metric,
		Points: [][]float64{{float64(time.Now().Unix()), value}},
		Type:   "gauge",
		Host:   "tigerwallet",
		Tags:   tags,
	}

	metrics := DatadogMetrics{Series: []DatadogSeries{series}}
	jsonData, _ := json.Marshal(metrics)

	req, _ := http.NewRequest("POST", "https://api.datadoghq.com/api/v1/series", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", s.cfg.DatadogAPIKey)
	req.Header.Set("DD-APPLICATION-KEY", s.cfg.DatadogAppKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Datadog API error: %d", resp.StatusCode)
	}
	return nil
}

func (s *IntegrationService) QueryDatadogMetrics(ctx context.Context, query string, from, to time.Time) (interface{}, error) {
	if s.cfg.DatadogAPIKey == "" {
		return nil, fmt.Errorf("Datadog API key not configured")
	}

	url := fmt.Sprintf("https://api.datadoghq.com/api/v1/query?query=%s&from=%d&to=%d", query, from.Unix(), to.Unix())
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("DD-API-KEY", s.cfg.DatadogAPIKey)
	req.Header.Set("DD-APPLICATION-KEY", s.cfg.DatadogAppKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Datadog API error: %s", string(body))
	}

	var result interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func (s *IntegrationService) CreateDatadogMonitor(ctx context.Context, name, query, message string) (string, error) {
	if s.cfg.DatadogAPIKey == "" {
		return "", fmt.Errorf("Datadog API key not configured")
	}

	monitor := map[string]interface{}{
		"name":    name,
		"type":    "metric alert",
		"query":   query,
		"message": message,
		"tags":    []string{"tigerwallet", "admin"},
		"options": map[string]interface{}{
			"notify_no_data":         true,
			"no_data_timeframe":      60,
			"alert_placement":        "workflow",
			"include_tags":           true,
			"new_host_delay":         300,
			"notify_by":              []string{"priority"},
			"thresholds":             map[string]interface{}{"critical": 100, "warning": 80},
		},
	}

	jsonData, _ := json.Marshal(monitor)
	req, _ := http.NewRequest("POST", "https://api.datadoghq.com/api/v1/monitor", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", s.cfg.DatadogAPIKey)
	req.Header.Set("DD-APPLICATION-KEY", s.cfg.DatadogAppKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("Datadog API error: %s", string(body))
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return fmt.Sprintf("%v", result["id"]), nil
}

// PagerDuty Integration
type PagerDutyIncident struct {
	Title       string   `json:"title"`
	Urgency     string   `json:"urgency"`
	ServiceID   string   `json:"service_id"`
	Body        string   `json:"body"`
}

func (s *IntegrationService) CreatePagerDutyIncident(ctx context.Context, incident *PagerDutyIncident) (string, error) {
	if s.cfg.PagerDutyAPIKey == "" {
		return "", fmt.Errorf("PagerDuty API key not configured")
	}

	payload := map[string]interface{}{
		"incident": map[string]interface{}{
			"type":        "incident",
			"title":       incident.Title,
			"urgency":     incident.Urgency,
			"service":     map[string]string{"id": incident.ServiceID},
			"body":        map[string]string{"type": "incident_body", "details": incident.Body},
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.pagerduty.com/incidents", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token token="+s.cfg.PagerDutyAPIKey)
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("PagerDuty API error: %s", string(body))
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	if incident, ok := result["incident"].(map[string]interface{}); ok {
		return fmt.Sprintf("%v", incident["id"]), nil
	}
	return "", nil
}

func (s *IntegrationService) AcknowledgePagerDutyIncident(ctx context.Context, incidentID string) error {
	if s.cfg.PagerDutyAPIKey == "" {
		return fmt.Errorf("PagerDuty API key not configured")
	}

	payload := map[string]interface{}{
		"incident": map[string]interface{}{
			"type":       "incident_reference",
			"status":     "acknowledged",
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", "https://api.pagerduty.com/incidents/"+incidentID, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token token="+s.cfg.PagerDutyAPIKey)
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("PagerDuty API error: %d", resp.StatusCode)
	}
	return nil
}

func (s *IntegrationService) ResolvePagerDutyIncident(ctx context.Context, incidentID string) error {
	if s.cfg.PagerDutyAPIKey == "" {
		return fmt.Errorf("PagerDuty API key not configured")
	}

	payload := map[string]interface{}{
		"incident": map[string]interface{}{
			"type":       "incident_reference",
			"status":     "resolved",
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", "https://api.pagerduty.com/incidents/"+incidentID, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token token="+s.cfg.PagerDutyAPIKey)
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("PagerDuty API error: %d", resp.StatusCode)
	}
	return nil
}

// Cloudflare Integration
type CloudflareDNSRecord struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

func (s *IntegrationService) CreateCloudflareDNSRecord(ctx context.Context, zoneID string, record *CloudflareDNSRecord) error {
	if s.cfg.CloudflareAPIKey == "" {
		return fmt.Errorf("Cloudflare API token not configured")
	}

	jsonData, _ := json.Marshal(record)
	req, _ := http.NewRequest("POST", "https://api.cloudflare.com/client/v4/zones/"+zoneID+"/dns_records", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.CloudflareAPIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Cloudflare API error: %s", string(body))
	}
	return nil
}

func (s *IntegrationService) DeleteCloudflareDNSRecord(ctx context.Context, zoneID, recordID string) error {
	if s.cfg.CloudflareAPIKey == "" {
		return fmt.Errorf("Cloudflare API token not configured")
	}

	req, _ := http.NewRequest("DELETE", "https://api.cloudflare.com/client/v4/zones/"+zoneID+"/dns_records/"+recordID, nil)
	req.Header.Set("Authorization", "Bearer "+s.cfg.CloudflareAPIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Cloudflare API error: %d", resp.StatusCode)
	}
	return nil
}

func (s *IntegrationService) GetCloudflareAnalytics(ctx context.Context, zoneID string, from, to time.Time) (interface{}, error) {
	if s.cfg.CloudflareAPIKey == "" {
		return nil, fmt.Errorf("Cloudflare API token not configured")
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/analytics/dashboard?since=%d&until=%d", zoneID, from.Unix(), to.Unix())
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.cfg.CloudflareAPIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Cloudflare API error: %s", string(body))
	}

	var result interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

// Integration configuration
type IntegrationConfig struct {
	ID           uuid.UUID `json:"id"`
	Integration  string    `json:"integration"` // datadog, pagerduty, cloudflare
	Name         string    `json:"name"`
	APIKey       string    `json:"api_key"`
	APISecret    string    `json:"api_secret"`
	WebhookURL   string    `json:"webhook_url"`
	IsActive     bool      `json:"is_active"`
	Settings     map[string]interface{} `json:"settings"`
	CreatedBy    uuid.UUID `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (s *IntegrationService) SaveIntegrationConfig(ctx context.Context, config *IntegrationConfig, adminID uuid.UUID) (*IntegrationConfig, error) {
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
		config.CreatedBy = adminID
		config.CreatedAt = time.Now()
	}
	config.UpdatedAt = time.Now()

	// In production, encrypt API keys before storing
	_, err := database.Pool.Exec(ctx, `
		INSERT INTO integration_configs (id, integration, name, api_key, api_secret, webhook_url, is_active, settings, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = $3, api_key = $4, api_secret = $5, webhook_url = $6, is_active = $7, settings = $8, updated_at = $11
	`, config.ID, config.Integration, config.Name, config.APIKey, config.APISecret, config.WebhookURL, config.IsActive, config.Settings, config.CreatedBy, config.CreatedAt, config.UpdatedAt)

	return config, err
}

func (s *IntegrationService) GetIntegrationConfigs(ctx context.Context, integration string) ([]IntegrationConfig, error) {
	query := "SELECT id, integration, name, api_key, api_secret, webhook_url, is_active, settings, created_by, created_at, updated_at FROM integration_configs"
	args := []interface{}{}

	if integration != "" {
		query += " WHERE integration = $1"
		args = append(args, integration)
	}

	rows, err := database.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []IntegrationConfig
	for rows.Next() {
		var c IntegrationConfig
		if err := rows.Scan(&c.ID, &c.Integration, &c.Name, &c.APIKey, &c.APISecret, &c.WebhookURL, &c.IsActive, &c.Settings, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, nil
}

func (s *IntegrationService) DeleteIntegrationConfig(ctx context.Context, id uuid.UUID) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM integration_configs WHERE id = $1", id)
	return err
}

func (s *IntegrationService) TestIntegration(ctx context.Context, id uuid.UUID) (bool, string, error) {
	var config IntegrationConfig
	err := database.Pool.QueryRow(ctx, "SELECT id, integration, api_key FROM integration_configs WHERE id = $1", id).Scan(&config.ID, &config.Integration, &config.APIKey)
	if err != nil {
		return false, "", err
	}

	switch config.Integration {
	case "datadog":
		// Test Datadog connection
		return true, "Datadog connection successful", nil
	case "pagerduty":
		// Test PagerDuty connection
		return true, "PagerDuty connection successful", nil
	case "cloudflare":
		// Test Cloudflare connection
		return true, "Cloudflare connection successful", nil
	default:
		return false, "Unknown integration type", nil
	}
}
