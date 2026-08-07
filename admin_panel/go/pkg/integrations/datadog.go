// Datadog - Datadog integration for monitoring and metrics
package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DatadogClient wraps Datadog API interactions
type DatadogClient struct {
	apiKey string
	appKey string
	site   string
	client *http.Client
}

// NewDatadogClient creates a new Datadog client
func NewDatadogClient(apiKey, appKey, site string) *DatadogClient {
	return &DatadogClient{
		apiKey: apiKey,
		appKey: appKey,
		site:   site,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// MetricPoint represents a single metric data point
type MetricPoint struct {
	Metric string
	Type   string // gauge, count, rate
	Value  float64
	Tags   []string
	Host   string
	Time   time.Time
}

// SendMetrics sends metrics to Datadog
func (d *DatadogClient) SendMetrics(ctx context.Context, metrics []MetricPoint) error {
	series := make([]map[string]interface{}, len(metrics))

	for i, m := range metrics {
		series[i] = map[string]interface{}{
			"metric": m.Metric,
			"type":   m.Type,
			"points": [][]interface{}{
				{int(m.Time.Unix()), m.Value},
			},
			"tags": m.Tags,
		}
		if m.Host != "" {
			series[i]["host"] = m.Host
		}
	}

	payload := map[string]interface{}{
		"series": series,
	}

	return d.post(ctx, "/api/v1/series", payload)
}

// SendEvent sends an event to Datadog
func (d *DatadogClient) SendEvent(ctx context.Context, title, text, priority, alertType string, tags []string) error {
	event := map[string]interface{}{
		"title":      title,
		"text":       text,
		"priority":   priority,
		"alert_type": alertType,
		"tags":       tags,
		"source":     "tigerwallet",
		"timestamp":  time.Now().Unix(),
	}

	return d.post(ctx, "/api/v1/events", event)
}

// SendLog sends a log to Datadog
func (d *DatadogClient) SendLog(ctx context.Context, message, level, service string, tags map[string]string) error {
	logEntry := map[string]interface{}{
		"message":   message,
		"timestamp": time.Now().UnixMilli(),
		"level":     level,
		"service":   service,
		"source":    "tigerwallet",
	}

	if tags != nil {
		logEntry["tags"] = tags
	}

	return d.post(ctx, "/api/v2/logs", logEntry)
}

// QueryMetrics queries metrics from Datadog
func (d *DatadogClient) QueryMetrics(ctx context.Context, query string, from, to time.Time) ([]map[string]interface{}, error) {
	params := map[string]interface{}{
		"query": query,
		"from":  from.Unix(),
		"to":    to.Unix(),
	}

	var result map[string]interface{}
	if err := d.get(ctx, "/api/v1/query", params, &result); err != nil {
		return nil, err
	}

	series, ok := result["series"].([]map[string]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	return series, nil
}

// CreateDashboard creates a Datadog dashboard
func (d *DatadogClient) CreateDashboard(ctx context.Context, title, description string, widgets []map[string]interface{}) error {
	dashboard := map[string]interface{}{
		"title":       title,
		"widgets":     widgets,
		"layout_type": "ordered",
		"description": description,
	}

	return d.post(ctx, "/api/v1/dashboard", dashboard)
}

// CreateMonitor creates a Datadog monitor
func (d *DatadogClient) CreateMonitor(ctx context.Context, name, query, message, alertType string) error {
	monitor := map[string]interface{}{
		"name":    name,
		"type":    "query alert",
		"query":   query,
		"message": message,
		"tags":    []string{"tigerwallet", "admin"},
		"options": map[string]interface{}{
			"thresholds": map[string]interface{}{
				"critical": alertType,
			},
			"notify_no_data":    true,
			"no_data_timeframe": 20,
		},
	}

	return d.post(ctx, "/api/v1/monitor", monitor)
}

func (d *DatadogClient) post(ctx context.Context, endpoint string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://api.%s%s", d.site, endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", d.apiKey)
	req.Header.Set("DD-APPLICATION-KEY", d.appKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("Datadog API error: %v", errResp)
	}

	return nil
}

func (d *DatadogClient) get(ctx context.Context, endpoint string, params map[string]interface{}, result interface{}) error {
	url := fmt.Sprintf("https://api.%s%s", d.site, endpoint)

	// Add query parameters
	q := ""
	for k, v := range params {
		q += fmt.Sprintf("%s=%v&", k, v)
	}
	if q != "" {
		url += "?" + q[:len(q)-1]
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("DD-API-KEY", d.apiKey)
	req.Header.Set("DD-APPLICATION-KEY", d.appKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("Datadog API error: %v", errResp)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
