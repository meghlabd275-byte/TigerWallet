// Cloudflare - Cloudflare integration for DNS and security
package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CloudflareClient wraps Cloudflare API interactions
type CloudflareClient struct {
	apiKey  string
	email   string
	zoneID  string
	baseURL string
	client  *http.Client
}

// NewCloudflareClient creates a new Cloudflare client
func NewCloudflareClient(apiKey, email, zoneID string) *CloudflareClient {
	return &CloudflareClient{
		apiKey:  apiKey,
		email:   email,
		zoneID:  zoneID,
		baseURL: "https://api.cloudflare.com/client/v4",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DNSRecord represents a DNS record
type DNSRecord struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Content   string `json:"content"`
	TTL       int    `json:"ttl"`
	Proxied   *bool  `json:"proxied"`
	CreatedAt string `json:"created_on"`
	UpdatedAt string `json:"modified_on"`
}

// CreateDNSRecord creates a new DNS record
func (c *CloudflareClient) CreateDNSRecord(ctx context.Context, recordType, name, content string, proxied bool) (*DNSRecord, error) {
	record := map[string]interface{}{
		"type":    recordType,
		"name":    name,
		"content": content,
		"ttl":     1, // 1 = automatic
		"proxied": proxied,
	}

	var result map[string]interface{}
	if err := c.post(ctx, "/zones/"+c.zoneID+"/dns_records", record, &result); err != nil {
		return nil, err
	}

	resultRecord, ok := result["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response")
	}

	return &DNSRecord{
		ID:      resultRecord["id"].(string),
		Type:    resultRecord["type"].(string),
		Name:    resultRecord["name"].(string),
		Content: resultRecord["content"].(string),
		Proxied: func(b bool) *bool { return &b }(resultRecord["proxied"].(bool)),
	}, nil
}

// GetDNSRecord gets a DNS record by ID
func (c *CloudflareClient) GetDNSRecord(ctx context.Context, recordID string) (*DNSRecord, error) {
	var result map[string]interface{}
	if err := c.get(ctx, "/zones/"+c.zoneID+"/dns_records/"+recordID, &result); err != nil {
		return nil, err
	}

	resultRecord, ok := result["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response")
	}

	return &DNSRecord{
		ID:        resultRecord["id"].(string),
		Type:      resultRecord["type"].(string),
		Name:      resultRecord["name"].(string),
		Content:   resultRecord["content"].(string),
		TTL:       int(resultRecord["ttl"].(float64)),
		CreatedAt: resultRecord["created_on"].(string),
		UpdatedAt: resultRecord["modified_on"].(string),
	}, nil
}

// ListDNSRecords lists all DNS records
func (c *CloudflareClient) ListDNSRecords(ctx context.Context, recordType string) ([]DNSRecord, error) {
	endpoint := "/zones/" + c.zoneID + "/dns_records"
	if recordType != "" {
		endpoint += "?type=" + recordType
	}

	var result map[string]interface{}
	if err := c.get(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	records, ok := result["result"].([]interface{})
	if !ok {
		return []DNSRecord{}, nil
	}

	var dnsRecords []DNSRecord
	for _, r := range records {
		rec := r.(map[string]interface{})
		var proxied *bool
		if p, ok := rec["proxied"].(bool); ok {
			proxied = &p
		}

		dnsRecords = append(dnsRecords, DNSRecord{
			ID:        rec["id"].(string),
			Type:      rec["type"].(string),
			Name:      rec["name"].(string),
			Content:   rec["content"].(string),
			TTL:       int(rec["ttl"].(float64)),
			Proxied:   proxied,
			CreatedAt: rec["created_on"].(string),
			UpdatedAt: rec["modified_on"].(string),
		})
	}

	return dnsRecords, nil
}

// UpdateDNSRecord updates a DNS record
func (c *CloudflareClient) UpdateDNSRecord(ctx context.Context, recordID, recordType, name, content string, proxied bool) error {
	record := map[string]interface{}{
		"type":    recordType,
		"name":    name,
		"content": content,
		"ttl":     1,
		"proxied": proxied,
	}

	return c.put(ctx, "/zones/"+c.zoneID+"/dns_records/"+recordID, record)
}

// DeleteDNSRecord deletes a DNS record
func (c *CloudflareClient) DeleteDNSRecord(ctx context.Context, recordID string) error {
	return c.delete(ctx, "/zones/"+c.zoneID+"/dns_records/"+recordID)
}

// FirewallRule represents a firewall rule
type FirewallRule struct {
	ID          string                 `json:"id"`
	Filter      map[string]interface{} `json:"filter"`
	Action      string                 `json:"action"`
	Priority    int                    `json:"priority"`
	Paused      bool                   `json:"paused"`
	Description string                 `json:"description"`
}

// CreateFirewallRule creates a firewall rule
func (c *CloudflareClient) CreateFirewallRule(ctx context.Context, filter map[string]interface{}, action, description string) (*FirewallRule, error) {
	rule := map[string]interface{}{
		"filter":      filter,
		"action":      action,
		"description": description,
		"paused":      false,
	}

	var result map[string]interface{}
	if err := c.post(ctx, "/zones/"+c.zoneID+"/firewall/rules", map[string]interface{}{"rules": []interface{}{rule}}, &result); err != nil {
		return nil, err
	}

	rules, ok := result["result"].([]interface{})
	if !ok || len(rules) == 0 {
		return nil, fmt.Errorf("invalid response")
	}

	ruleMap := rules[0].(map[string]interface{})
	return &FirewallRule{
		ID:          ruleMap["id"].(string),
		Filter:      ruleMap["filter"].(map[string]interface{}),
		Action:      ruleMap["action"].(string),
		Description: ruleMap["description"].(string),
		Paused:      ruleMap["paused"].(bool),
	}, nil
}

// ListFirewallRules lists all firewall rules
func (c *CloudflareClient) ListFirewallRules(ctx context.Context) ([]FirewallRule, error) {
	var result map[string]interface{}
	if err := c.get(ctx, "/zones/"+c.zoneID+"/firewall/rules", &result); err != nil {
		return nil, err
	}

	rules, ok := result["result"].([]interface{})
	if !ok {
		return []FirewallRule{}, nil
	}

	var firewallRules []FirewallRule
	for _, r := range rules {
		rule := r.(map[string]interface{})
		firewallRules = append(firewallRules, FirewallRule{
			ID:          rule["id"].(string),
			Filter:      rule["filter"].(map[string]interface{}),
			Action:      rule["action"].(string),
			Description: rule["description"].(string),
			Paused:      rule["paused"].(bool),
		})
	}

	return firewallRules, nil
}

// DeleteFirewallRule deletes a firewall rule
func (c *CloudflareClient) DeleteFirewallRule(ctx context.Context, ruleID string) error {
	return c.delete(ctx, "/zones/"+c.zoneID+"/firewall/rules/"+ruleID)
}

// Analytics represents analytics data
type Analytics struct {
	Requests          int64 `json:"requests"`
	RequestsCached    int64 `json:"requests_cached"`
	RequestsUncached  int64 `json:"requests_uncached"`
	Bandwidth         int64 `json:"bandwidth_all"`
	BandwidthCached   int64 `json:"bandwidth_cached"`
	BandwidthUncached int64 `json:"bandwidth_uncached"`
	PageViews         int64 `json:"pageviews"`
	UniqueVisitors    int64 `json:"uniques"`
}

// GetAnalytics gets analytics data
func (c *CloudflareClient) GetAnalytics(ctx context.Context, from, to time.Time) (*Analytics, error) {
	endpoint := fmt.Sprintf("/zones/%s/analytics/dashboard?since=%s&until=%s",
		c.zoneID,
		from.Format(time.RFC3339),
		to.Format(time.RFC3339))

	var result map[string]interface{}
	if err := c.get(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	data, ok := result["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response")
	}

	return &Analytics{
		Requests:          int64(data["requests"].(float64)),
		RequestsCached:    int64(data["requests_cached"].(float64)),
		RequestsUncached:  int64(data["requests_uncached"].(float64)),
		Bandwidth:         int64(data["bandwidth"].(float64)),
		BandwidthCached:   int64(data["bandwidth_cached"].(float64)),
		BandwidthUncached: int64(data["bandwidth_uncached"].(float64)),
		PageViews:         int64(data["pageviews"].(float64)),
		UniqueVisitors:    int64(data["uniques"].(float64)),
	}, nil
}

// PurgeCache purges all cached files
func (c *CloudflareClient) PurgeCache(ctx context.Context) error {
	payload := map[string]interface{}{
		"purge_everything": true,
	}

	return c.post(ctx, "/zones/"+c.zoneID+"/purge_cache", payload, nil)
}

func (c *CloudflareClient) post(ctx context.Context, endpoint string, payload interface{}, result interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("Cloudflare API error: %v", errResp)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

func (c *CloudflareClient) get(ctx context.Context, endpoint string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("Cloudflare API error: %v", errResp)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *CloudflareClient) put(ctx context.Context, endpoint string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", c.baseURL+endpoint, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("Cloudflare API error: %v", errResp)
	}

	return nil
}

func (c *CloudflareClient) delete(ctx context.Context, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("Cloudflare API error: %v", errResp)
	}

	return nil
}

func (c *CloudflareClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Email", c.email)
	req.Header.Set("X-Auth-Key", c.apiKey)
}
