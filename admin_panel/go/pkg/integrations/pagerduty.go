// PagerDuty - PagerDuty integration for incident management
package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// PagerDutyClient wraps PagerDuty API interactions
type PagerDutyClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewPagerDutyClient creates a new PagerDuty client
func NewPagerDutyClient(apiKey string) *PagerDutyClient {
	return &PagerDutyClient{
		apiKey:  apiKey,
		baseURL: "https://api.pagerduty.com",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Incident represents a PagerDuty incident
type PagerDutyIncident struct {
	ID             string                 `json:"id"`
	IncidentNumber int                    `json:"incident_number"`
	Title          string                 `json:"title"`
	Status         string                 `json:"status"`
	Priority       string                 `json:"priority"`
	Service        map[string]interface{} `json:"service"`
	CreatedAt      time.Time              `json:"created_at"`
}

// CreateIncident creates a new PagerDuty incident
func (p *PagerDutyClient) CreateIncident(ctx context.Context, title, description, urgency, serviceID string) (*PagerDutyIncident, error) {
	incident := map[string]interface{}{
		"incident": map[string]interface{}{
			"type":  "incident",
			"title": title,
			"body": map[string]interface{}{
				"type":    "incident_body",
				"details": description,
			},
			"urgency": urgency,
			"service": map[string]interface{}{
				"id":   serviceID,
				"type": "service_reference",
			},
		},
	}

	var result map[string]interface{}
	if err := p.post(ctx, "/incidents", incident, &result); err != nil {
		return nil, err
	}

	inc, ok := result["incident"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return &PagerDutyIncident{
		ID:        inc["id"].(string),
		Title:     inc["title"].(string),
		Status:    inc["status"].(string),
		CreatedAt: time.Now(),
	}, nil
}

// AcknowledgeIncident acknowledges an incident
func (p *PagerDutyClient) AcknowledgeIncident(ctx context.Context, incidentID, userID string) error {
	update := map[string]interface{}{
		"incident": map[string]interface{}{
			"type":   "incident_reference",
			"status": "acknowledged",
		},
	}

	return p.put(ctx, "/incidents/"+incidentID, update)
}

// ResolveIncident resolves an incident
func (p *PagerDutyClient) ResolveIncident(ctx context.Context, incidentID, userID, resolution string) error {
	update := map[string]interface{}{
		"incident": map[string]interface{}{
			"type":       "incident_reference",
			"status":     "resolved",
			"resolution": resolution,
		},
	}

	return p.put(ctx, "/incidents/"+incidentID, update)
}

// GetIncident gets an incident by ID
func (p *PagerDutyClient) GetIncident(ctx context.Context, incidentID string) (*PagerDutyIncident, error) {
	var result map[string]interface{}
	if err := p.get(ctx, "/incidents/"+incidentID, &result); err != nil {
		return nil, err
	}

	inc, ok := result["incident"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return &PagerDutyIncident{
		ID:        inc["id"].(string),
		Title:     inc["title"].(string),
		Status:    inc["status"].(string),
		Service:   inc["service"].(map[string]interface{}),
		CreatedAt: time.Now(),
	}, nil
}

// ListIncidents lists incidents
func (p *PagerDutyClient) ListIncidents(ctx context.Context, status string) ([]PagerDutyIncident, error) {
	var result map[string]interface{}

	endpoint := "/incidents"
	if status != "" {
		endpoint += "?statuses[]=" + status
	}

	if err := p.get(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	incidents, ok := result["incidents"].([]interface{})
	if !ok {
		return []PagerDutyIncident{}, nil
	}

	var resultIncidents []PagerDutyIncident
	for _, inc := range incidents {
		i := inc.(map[string]interface{})
		resultIncidents = append(resultIncidents, PagerDutyIncident{
			ID:        i["id"].(string),
			Title:     i["title"].(string),
			Status:    i["status"].(string),
			Service:   i["service"].(map[string]interface{}),
			CreatedAt: time.Now(),
		})
	}

	return resultIncidents, nil
}

// AddNote adds a note to an incident
func (p *PagerDutyClient) AddNote(ctx context.Context, incidentID, userID, content string) error {
	note := map[string]interface{}{
		"note": map[string]interface{}{
			"content": content,
		},
	}

	return p.post(ctx, "/incidents/"+incidentID+"/notes", note, nil)
}

// CreateService creates a PagerDuty service
func (p *PagerDutyClient) CreateService(ctx context.Context, name, description string) (string, error) {
	service := map[string]interface{}{
		"service": map[string]interface{}{
			"name":        name,
			"description": description,
			"status":      "active",
			"escalation_policy": map[string]interface{}{
				"id":   "PXXXXXX", // Would be actual policy ID
				"type": "escalation_policy_reference",
			},
		},
	}

	var result map[string]interface{}
	if err := p.post(ctx, "/services", service, &result); err != nil {
		return "", err
	}

	svc, ok := result["service"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response")
	}

	return svc["id"].(string), nil
}

func (p *PagerDutyClient) post(ctx context.Context, endpoint string, payload interface{}, result interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+endpoint, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token token="+p.apiKey)
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("PagerDuty API error: %v", errResp)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

func (p *PagerDutyClient) get(ctx context.Context, endpoint string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Token token="+p.apiKey)
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("PagerDuty API error: %v", errResp)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (p *PagerDutyClient) put(ctx context.Context, endpoint string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", p.baseURL+endpoint, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token token="+p.apiKey)
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("PagerDuty API error: %v", errResp)
	}

	return nil
}
