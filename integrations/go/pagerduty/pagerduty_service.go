// PagerDuty Integration Service
// Incident management, alerting, and on-call scheduling

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PagerDutyConfig - PagerDuty Integration Configuration
type PagerDutyConfig struct {
	// API Settings
	APIKey          string `json:"api_key"`
	IntegrationKey  string `json:"integration_key"` // for events API
	OAuthToken     string `json:"oauth_token"`
	
	// Settings
	AutoResolve    bool   `json:"auto_resolve"`
	SeverityMap   string `json:"severity_map"` // JSON mapping
	
	// Database Settings
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	
	// Redis Settings
	RedisHost string `json:"redis_host"`
	RedisPort string `json:"redis_port"`
	
	// Server
	ServerPort string `json:"server_port"`
}

// PagerDutyService - Main PagerDuty integration service
type PagerDutyService struct {
	config  PagerDutyConfig
	db      *gorm.DB
	redis   *redis.Client
	client  *http.Client
}

// NewPagerDutyService - Create new PagerDuty service
func NewPagerDutyService(cfg PagerDutyConfig) (*PagerDutyService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
	})
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	
	return &PagerDutyService{
		config: cfg,
		db:     db,
		redis:  rdb,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// getHeaders - Get PagerDuty API headers
func (s *PagerDutyService) getHeaders() map[string]string {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Accept":        "application/vnd.pagerduty+json;version=2",
	}
	
	if s.config.OAuthToken != "" {
		headers["Authorization"] = "Bearer " + s.config.OAuthToken
	} else if s.config.APIKey != "" {
		headers["Authorization"] = "Token token=" + s.config.APIKey
	}
	
	return headers
}

// getBaseURL - Get PagerDuty API base URL
func (s *PagerDutyService) getBaseURL() string {
	return "https://api.pagerduty.com"
}

// callAPI - Make API call to PagerDuty
func (s *PagerDutyService) callAPI(method, endpoint string, body []byte) (map[string]interface{}, error) {
	url := s.getBaseURL() + endpoint
	
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	
	for k, v := range s.getHeaders() {
		req.Header.Set(k, v)
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	respBody, _ := io.ReadAll(resp.Body)
	
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)
	
	return result, nil
}

// ==================== Event Integration ====================

// TriggerEvent - Trigger PagerDuty event
func (s *PagerDutyService) TriggerEvent(title, body, severity, source string, customDetails map[string]interface{}) (string, error) {
	// Map severity
	pdSeverity := "warning"
	switch severity {
	case "critical":
		pdSeverity = "critical"
	case "error":
		pdSeverity = "error"
	case "warning":
		pdSeverity = "warning"
	case "info":
		pdSeverity = "info"
	}
	
	event := map[string]interface{}{
		"routing_key": s.config.IntegrationKey,
		"event_action": "trigger",
		"dedup_key":    fmt.Sprintf("%d_%s", time.Now().Unix(), strings.ReplaceAll(title, " ", "_")),
		"payload": map[string]interface{}{
			"summary":   title,
			"severity": pdSeverity,
			"source":    source,
			"custom_details": customDetails,
		},
	}
	
	if body != "" {
		event["payload"].(map[string]interface{})["body"] = body
	}
	
	bodyBytes, _ := json.Marshal(event)
	
	// Use Events API v2
	resp, err := http.Post("https://events.pagerduty.com/v2/enqueue", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	respBody, _ := io.ReadAll(resp.Body)
	
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)
	
	if result["dedup_key"] != nil {
		return result["dedup_key"].(string), nil
	}
	
	return "", nil
}

// ResolveEvent - Resolve PagerDuty event
func (s *PagerDutyService) ResolveEvent(dedupKey, title string) error {
	event := map[string]interface{}{
		"routing_key": s.config.IntegrationKey,
		"event_action": "resolve",
		"dedup_key":    dedupKey,
	}
	
	bodyBytes, _ := json.Marshal(event)
	
	resp, err := http.Post("https://events.pagerduty.com/v2/enqueue", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	return nil
}

// AcknowledgeEvent - Acknowledge PagerDuty event
func (s *PagerDutyService) AcknowledgeEvent(dedupKey, title, user string) error {
	event := map[string]interface{}{
		"routing_key": s.config.IntegrationKey,
		"event_action": "acknowledge",
		"dedup_key":    dedupKey,
		"payload": map[string]interface{}{
			"summary": title,
			"source":  "tigerwallet",
		},
	}
	
	bodyBytes, _ := json.Marshal(event)
	
	resp, err := http.Post("https://events.pagerduty.com/v2/enqueue", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	return nil
}

// ==================== Incident Management ====================

// CreateIncident - Create incident via API
func (s *PagerDutyService) CreateIncident(title, description, urgency, serviceID string, assignments []map[string]interface{}) (string, error) {
	incident := map[string]interface{}{
		"incident": map[string]interface{}{
			"type":        "incident",
			"title":       title,
			"body": map[string]interface{}{
				"type": "markdown",
				"content": description,
			},
			"urgency":     urgency,
			"service":     map[string]interface{}{"id": serviceID, "type": "service_reference"},
			"assignments": assignments,
		},
	}
	
	body, _ := json.Marshal(incident)
	
	result, err := s.callAPI("POST", "/incidents", body)
	if err != nil {
		return "", err
	}
	
	if inc, ok := result["incident"].(map[string]interface{}); ok {
		if id, ok := inc["id"].(string); ok {
			return id, nil
		}
	}
	
	return "", nil
}

// GetIncidents - Get incidents
func (s *PagerDutyService) GetIncidents(status string) ([]map[string]interface{}, error) {
	endpoint := "/incidents"
	if status != "" {
		endpoint += "?statuses[]=" + status
	}
	
	result, err := s.callAPI("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	
	if incidents, ok := result["incidents"].([]interface{}); ok {
		resultIncidents := make([]map[string]interface{}, 0)
		for _, inc := range incidents {
			if m, ok := inc.(map[string]interface{}); ok {
				resultIncidents = append(resultIncidents, m)
			}
		}
		return resultIncidents, nil
	}
	
	return nil, nil
}

// UpdateIncidentStatus - Update incident status
func (s *PagerDutyService) UpdateIncidentStatus(incidentID, status, resolution string) error {
	updates := map[string]interface{}{
		"incident": map[string]interface{}{
			"type": "incident_reference",
			"id":   incidentID,
		},
	}
	
	switch status {
	case "resolved":
		updates["incident"].(map[string]interface{})["resolution"] = resolution
	case "acknowledged":
		// No additional fields
	}
	
	body, _ := json.Marshal(updates)
	
	_, err := s.callAPI("PUT", "/incidents/"+incidentID, body)
	return err
}

// ==================== Services ====================

// GetServices - Get services
func (s *PagerDutyService) GetServices() ([]map[string]interface{}, error) {
	result, err := s.callAPI("GET", "/services", nil)
	if err != nil {
		return nil, err
	}
	
	if services, ok := result["services"].([]interface{}); ok {
		resultServices := make([]map[string]interface{}, 0)
		for _, svc := range services {
			if m, ok := svc.(map[string]interface{}); ok {
				resultServices = append(resultServices, m)
			}
		}
		return resultServices, nil
	}
	
	return nil, nil
}

// ==================== On-Call ====================

// GetOnCall - Get current on-call users
func (s *PagerDutyService) GetOnCall() ([]map[string]interface{}, error) {
	result, err := s.callAPI("GET", "/oncalls?time_zone=UTC", nil)
	if err != nil {
		return nil, err
	}
	
	if oncalls, ok := result["oncalls"].([]interface{}); ok {
		resultOncalls := make([]map[string]interface{}, 0)
		for _, oc := range oncalls {
			if m, ok := oc.(map[string]interface{}); ok {
				resultOncalls = append(resultOncalls, m)
			}
		}
		return resultOncalls, nil
	}
	
	return nil, nil
}

// ==================== Alerts ====================

// AlertTransaction - Send transaction alert
func (s *PagerDutyService) AlertTransaction(txType, txHash, amount, status string) error {
	title := fmt.Sprintf("Transaction Alert: %s - %s", txType, status)
	body := fmt.Sprintf("Transaction: %s\nAmount: %s\nStatus: %s\nHash: %s", 
		txType, amount, status, txHash)
	
	severity := "warning"
	if status == "failed" {
		severity = "error"
	}
	
	_, err := s.TriggerEvent(title, body, severity, "tigerwallet-transactions", map[string]interface{}{
		"type":     txType,
		"hash":     txHash,
		"amount":   amount,
		"status":   status,
	})
	
	return err
}

// AlertWithdrawal - Send withdrawal alert
func (s *PagerDutyService) AlertWithdrawal(userID, amount, currency, status string) error {
	title := fmt.Sprintf("Withdrawal %s: %s %s", status, amount, currency)
	body := fmt.Sprintf("User: %s\nAmount: %s %s\nStatus: %s", userID, amount, currency, status)
	
	severity := "warning"
	if status == "failed" {
		severity = "critical"
	}
	
	_, err := s.TriggerEvent(title, body, severity, "tigerwallet-withdrawals", map[string]interface{}{
		"user_id":  userID,
		"amount":   amount,
		"currency": currency,
		"status":   status,
	})
	
	return err
}

// AlertSecurity - Send security alert
func (s *PagerDutyService) AlertSecurity(alertType, description, ipAddress string) error {
	title := fmt.Sprintf("Security Alert: %s", alertType)
	body := fmt.Sprintf("Alert Type: %s\nDescription: %s\nIP: %s\nTime: %s",
		alertType, description, ipAddress, time.Now().Format("2006-01-02 15:04:05"))
	
	_, err := s.TriggerEvent(title, body, "critical", "tigerwallet-security", map[string]interface{}{
		"alert_type": alertType,
		"description": description,
		"ip_address": ipAddress,
	})
	
	return err
}

// AlertSystem - Send system alert
func (s *PagerDutyService) AlertSystem(component, status, message string) error {
	title := fmt.Sprintf("System Alert: %s - %s", component, status)
	body := fmt.Sprintf("Component: %s\nStatus: %s\nMessage: %s\nTime: %s",
		component, status, message, time.Now().Format("2006-01-02 15:04:05"))
	
	severity := "warning"
	if status == "down" || status == "critical" {
		severity = "critical"
	}
	
	_, err := s.TriggerEvent(title, body, severity, "tigerwallet-system", map[string]interface{}{
		"component": component,
		"status":   status,
		"message":  message,
	})
	
	return err
}

// HTTP Handlers

type TriggerEventRequest struct {
	Title       string                 `json:"title" binding:"required"`
	Body        string                 `json:"body"`
	Severity    string                 `json:"severity"` // critical, error, warning, info
	Source      string                 `json:"source"`
	CustomDetails map[string]interface{} `json:"custom_details"`
}

func (s *PagerDutyService) TriggerEventHandler(c *gin.Context) {
	var req TriggerEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	source := req.Source
	if source == "" {
		source = "tigerwallet"
	}
	
	severity := req.Severity
	if severity == "" {
		severity = "warning"
	}
	
	dedupKey, err := s.TriggerEvent(req.Title, req.Body, severity, source, req.CustomDetails)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "triggered", "dedup_key": dedupKey})
}

type ResolveEventRequest struct {
	DedupKey string `json:"dedup_key" binding:"required"`
	Title    string `json:"title"`
}

func (s *PagerDutyService) ResolveEventHandler(c *gin.Context) {
	var req ResolveEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	err := s.ResolveEvent(req.DedupKey, req.Title)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "resolved"})
}

func (s *PagerDutyService) GetIncidentsHandler(c *gin.Context) {
	status := c.Query("status")
	
	incidents, err := s.GetIncidents(status)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"incidents": incidents})
}

func (s *PagerDutyService) GetServicesHandler(c *gin.Context) {
	services, err := s.GetServices()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"services": services})
}

func (s *PagerDutyService) GetOnCallHandler(c *gin.Context) {
	oncalls, err := s.GetOnCall()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"oncalls": oncalls})
}

type AlertRequest struct {
	AlertType string `json:"alert_type" binding:"required"` // transaction, withdrawal, security, system
	Data      map[string]interface{} `json:"data" binding:"required"`
}

func (s *PagerDutyService) AlertHandler(c *gin.Context) {
	var req AlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	var err error
	
	switch req.AlertType {
	case "transaction":
		txType := fmt.Sprintf("%v", req.Data["type"])
		txHash := fmt.Sprintf("%v", req.Data["hash"])
		amount := fmt.Sprintf("%v", req.Data["amount"])
		status := fmt.Sprintf("%v", req.Data["status"])
		err = s.AlertTransaction(txType, txHash, amount, status)
		
	case "withdrawal":
		userID := fmt.Sprintf("%v", req.Data["user_id"])
		amount := fmt.Sprintf("%v", req.Data["amount"])
		currency := fmt.Sprintf("%v", req.Data["currency"])
		status := fmt.Sprintf("%v", req.Data["status"])
		err = s.AlertWithdrawal(userID, amount, currency, status)
		
	case "security":
		alertType := fmt.Sprintf("%v", req.Data["alert_type"])
		description := fmt.Sprintf("%v", req.Data["description"])
		ipAddress := fmt.Sprintf("%v", req.Data["ip_address"])
		err = s.AlertSecurity(alertType, description, ipAddress)
		
	case "system":
		component := fmt.Sprintf("%v", req.Data["component"])
		status := fmt.Sprintf("%v", req.Data["status"])
		message := fmt.Sprintf("%v", req.Data["message"])
		err = s.AlertSystem(component, status, message)
		
	default:
		c.JSON(400, gin.H{"error": "invalid alert_type"})
		return
	}
	
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "alert_sent"})
}

// Main

func main() {
	cfg := PagerDutyConfig{
		APIKey:         getEnv("PAGERDUTY_API_KEY", ""),
		IntegrationKey: getEnv("PAGERDUTY_INTEGRATION_KEY", ""),
		OAuthToken:    getEnv("PAGERDUTY_OAUTH_TOKEN", ""),
		AutoResolve:   getEnvBool("PAGERDUTY_AUTO_RESOLVE", true),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		DBName:        getEnv("DB_NAME", "pagerduty_db"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		ServerPort:    getEnv("PAGERDUTY_SERVER_PORT", "8096"),
	}
	
	service, err := NewPagerDutyService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize PagerDuty service: %v", err)
	}
	
	r := gin.Default()
	
	r.POST("/events/trigger", service.TriggerEventHandler)
	r.POST("/events/resolve", service.ResolveEventHandler)
	r.POST("/alerts", service.AlertHandler)
	
	r.GET("/incidents", service.GetIncidentsHandler)
	r.GET("/services", service.GetServicesHandler)
	r.GET("/oncall", service.GetOnCallHandler)
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "pagerduty"})
	})
	
	log.Printf("PagerDuty Service starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}
