// Datadog Integration Service
// Metrics, logs, events, and monitoring for Datadog

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

// DatadogConfig - Datadog Integration Configuration
type DatadogConfig struct {
	// API Settings
	APIKey    string `json:"api_key"`
	APPKey    string `json:"app_key"`
	Site      string `json:"site"` // datadoghq.com, datadoghq.eu, us3.datadoghq.com
	
	// Custom Metrics
	CustomMetricsPrefix string `json:"custom_metrics_prefix"`
	
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

// DatadogMonitor - Monitor configuration
type DatadogMonitor struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	MonitorID   string    `gorm:"uniqueIndex" json:"monitor_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // metric, log, event, service check
	Query       string    `gorm:"type:text" json:"query"`
	Message     string    `gorm:"type:text" json:"message"`
	Tags        string    `json:"tags"` // JSON array
	Priority    int       `json:"priority"` // P1-P5
	IsEnabled   bool      `gorm:"default:true" json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DatadogMetric - Custom metric definition
type DatadogMetric struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex" json:"name"`
	Type        string    `json:"type"` // count, gauge, rate, histogram
	Description string    `json:"description"`
	Tags        string    `json:"tags"` // JSON array
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// DatadogAlert - Alert history
type DatadogAlert struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AlertID     string    `gorm:"index" json:"alert_id"`
	MonitorID   string    `gorm:"index" json:"monitor_id"`
	MonitorName string    `json:"monitor_name"`
	Severity    string    `json:"severity"` // critical, warning, info
	Status      string    `json:"status"` // triggered, resolved
	Message     string    `gorm:"type:text" json:"message"`
	EventData   string    `gorm:"type:jsonb" json:"event_data"`
	TriggeredAt time.Time `json:"triggered_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// DatadogService - Main Datadog integration service
type DatadogService struct {
	config DatadogConfig
	db     *gorm.DB
	redis  *redis.Client
	client *http.Client
}

// NewDatadogService - Create new Datadog service
func NewDatadogService(cfg DatadogConfig) (*DatadogService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	err = db.AutoMigrate(&DatadogMonitor{}, &DatadogMetric{}, &DatadogAlert{})
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
	
	if cfg.Site == "" {
		cfg.Site = "datadoghq.com"
	}
	
	if cfg.CustomMetricsPrefix == "" {
		cfg.CustomMetricsPrefix = "tigerwallet"
	}
	
	return &DatadogService{
		config: cfg,
		db:     db,
		redis:  rdb,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// getBaseURL - Get Datadog API base URL
func (s *DatadogService) getBaseURL() string {
	return fmt.Sprintf("https://api.%s", s.config.Site)
}

// getHeaders - Get Datadog API headers
func (s *DatadogService) getHeaders() map[string]string {
	return map[string]string{
		"DD-API-KEY":        s.config.APIKey,
		"DD-APPLICATION-KEY": s.config.APPKey,
		"Content-Type":      "application/json",
	}
}

// SendMetric - Send custom metric to Datadog
func (s *DatadogService) SendMetric(name string, value float64, tags map[string]string) error {
	// Add prefix to metric name
	fullName := fmt.Sprintf("%s.%s", s.config.CustomMetricsPrefix, name)
	
	// Build tags array
	var tagsArray []string
	for k, v := range tags {
		tagsArray = append(tagsArray, fmt.Sprintf("%s:%s", k, v))
	}
	
	// Build payload
	series := []map[string]interface{}{
		{
			"metric":   fullName,
			"points":   [][]interface{}{{time.Now().Unix(), value}},
			"type":     "gauge",
			"tags":     tagsArray,
			"hostname": "tigerwallet",
		},
	}
	
	payload, _ := json.Marshal(map[string]interface{}{
		"series": series,
	})
	
	// Send to Datadog
	req, err := http.NewRequest("POST", s.getBaseURL()+"/api/v1/series", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	
	for k, v := range s.getHeaders() {
		req.Header.Set(k, v)
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("datadog error: %s", string(body))
	}
	
	// Cache metric
	s.cacheMetric(name, value, tags)
	
	return nil
}

// SendCount - Send count metric
func (s *DatadogService) SendCount(name string, value float64, tags map[string]string) error {
	fullName := fmt.Sprintf("%s.%s", s.config.CustomMetricsPrefix, name)
	
	var tagsArray []string
	for k, v := range tags {
		tagsArray = append(tagsArray, fmt.Sprintf("%s:%s", k, v))
	}
	
	series := []map[string]interface{}{
		{
			"metric":   fullName,
			"points":   [][]interface{}{{time.Now().Unix(), value}},
			"type":     "count",
			"tags":     tagsArray,
			"hostname": "tigerwallet",
		},
	}
	
	payload, _ := json.Marshal(map[string]interface{}{"series": series})
	
	req, _ := http.NewRequest("POST", s.getBaseURL()+"/api/v1/series", bytes.NewReader(payload))
	for k, v := range s.getHeaders() {
		req.Header.Set(k, v)
	}
	
	resp, _ := s.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	
	return nil
}

// SendEvent - Send event to Datadog
func (s *DatadogService) SendEvent(title, text, alertType, priority string, tags map[string]string) (string, error) {
	event := map[string]interface{}{
		"title":      title,
		"text":       text,
		"alert_type": alertType,
		"priority":   priority,
		"source":     "tigerwallet",
		"timestamp":  time.Now().Unix(),
	}
	
	var tagsArray []string
	for k, v := range tags {
		tagsArray = append(tagsArray, fmt.Sprintf("%s:%s", k, v))
	}
	if len(tagsArray) > 0 {
		event["tags"] = tagsArray
	}
	
	payload, _ := json.Marshal(event)
	
	req, err := http.NewRequest("POST", s.getBaseURL()+"/api/v1/events", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	
	for k, v := range s.getHeaders() {
		req.Header.Set(k, v)
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("datadog error: %s", string(body))
	}
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	eventID := fmt.Sprintf("%v", result["event"]["id"])
	
	// Log alert
	s.logAlert(eventID, title, alertType, priority, text)
	
	return eventID, nil
}

// SendTransactionMetric - Send transaction metric
func (s *DatadogService) SendTransactionMetric(txType, status, chain string, value float64) error {
	tags := map[string]string{
		"type":   txType,
		"status": status,
		"chain":  chain,
	}
	
	return s.SendMetric("transaction", value, tags)
}

// SendUserMetric - Send user activity metric
func (s *DatadogService) SendUserMetric(action, userType string) error {
	tags := map[string]string{
		"action": action,
		"user_type": userType,
	}
	
	return s.SendCount("user.activity", 1, tags)
}

// SendAPIMetric - Send API metric
func (s *DatadogService) SendAPIMetric(endpoint, method, status string, latency float64) error {
	tags := map[string]string{
		"endpoint": endpoint,
		"method":   method,
		"status":   status,
	}
	
	return s.SendMetric("api.latency", latency, tags)
}

// SendSystemMetric - Send system metric
func (s *DatadogService) SendSystemMetric(metricName string, value float64, tags map[string]string) error {
	return s.SendMetric(metricName, value, tags)
}

// QueryMetrics - Query metrics from Datadog
func (s *DatadogService) QueryMetrics(query string, from, to int64) ([]map[string]interface{}, error) {
	params := map[string]interface{}{
		"query": query,
		"from":  from,
		"to":    to,
	}
	
	payload, _ := json.Marshal(params)
	
	req, err := http.NewRequest("GET", s.getBaseURL()+"/api/v1/query", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	
	req.Method = "GET"
	q := req.URL.Query()
	q.Add("query", query)
	q.Add("from", fmt.Sprintf("%d", from))
	q.Add("to", fmt.Sprintf("%d", to))
	req.URL.RawQuery = q.Encode()
	
	for k, v := range s.getHeaders() {
		req.Header.Set(k, v)
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("datadog error: %s", string(body))
	}
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	series, ok := result["series"].([]interface{})
	if !ok {
		return nil, nil
	}
	
	results := make([]map[string]interface{}, 0)
	for _, s := range series {
		results = append(results, s.(map[string]interface{}))
	}
	
	return results, nil
}

// GetMonitors - Get all monitors
func (s *DatadogService) GetMonitors() ([]DatadogMonitor, error) {
	var monitors []DatadogMonitor
	err := s.db.Where("is_enabled = ?", true).Find(&monitors).Error
	return monitors, err
}

// CreateMonitor - Create monitor
func (s *DatadogService) CreateMonitor(name, monitorType, query, message, tags string, priority int) error {
	monitor := &DatadogMonitor{
		Name:     name,
		Type:     monitorType,
		Query:    query,
		Message:  message,
		Tags:     tags,
		Priority: priority,
		IsEnabled: true,
		CreatedAt: time.Now(),
	}
	
	return s.db.Create(monitor).Error
}

// cacheMetric - Cache metric in Redis
func (s *DatadogService) cacheMetric(name string, value float64, tags map[string]string) {
	key := fmt.Sprintf("datadog:metric:%s", name)
	
	data, _ := json.Marshal(map[string]interface{}{
		"value": value,
		"tags":  tags,
		"time":  time.Now().Unix(),
	})
	
	s.redis.Set(context.Background(), key, data, 24*time.Hour)
}

// logAlert - Log alert to database
func (s *DatadogService) logAlert(alertID, title, alertType, priority, message string) {
	alert := &DatadogAlert{
		AlertID:     alertID,
		Severity:    priority,
		Status:      "triggered",
		Message:     message,
		TriggeredAt: time.Now(),
		CreatedAt:   time.Now(),
	}
	
	s.db.Create(alert)
}

// HTTP Handlers

type SendMetricRequest struct {
	Name  string             `json:"name" binding:"required"`
	Value float64            `json:"value" binding:"required"`
	Tags  map[string]string  `json:"tags"`
}

func (s *DatadogService) SendMetricHandler(c *gin.Context) {
	var req SendMetricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	tags := req.Tags
	if tags == nil {
		tags = make(map[string]string)
	}
	
	err := s.SendMetric(req.Name, req.Value, tags)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "sent"})
}

type SendEventRequest struct {
	Title    string             `json:"title" binding:"required"`
	Text     string             `json:"text" binding:"required"`
	AlertType string            `json:"alert_type"` // info, warning, error, success
	Priority string             `json:"priority"` // low, normal, high
	Tags     map[string]string  `json:"tags"`
}

func (s *DatadogService) SendEventHandler(c *gin.Context) {
	var req SendEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	alertType := req.AlertType
	if alertType == "" {
		alertType = "info"
	}
	
	priority := req.Priority
	if priority == "" {
		priority = "normal"
	}
	
	eventID, err := s.SendEvent(req.Title, req.Text, alertType, priority, req.Tags)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "sent", "event_id": eventID})
}

type TransactionMetricRequest struct {
	Type   string  `json:"type" binding:"required"` // deposit, withdrawal, transfer
	Status string  `json:"status" binding:"required"` // success, failed, pending
	Chain  string  `json:"chain" binding:"required"`
	Value  float64 `json:"value" binding:"required"`
}

func (s *DatadogService) TransactionMetricHandler(c *gin.Context) {
	var req TransactionMetricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	err := s.SendTransactionMetric(req.Type, req.Status, req.Chain, req.Value)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "sent"})
}

type QueryRequest struct {
	Query string `json:"query" binding:"required"`
	From  int64  `json:"from"` // Unix timestamp
	To    int64  `json:"to"`   // Unix timestamp
}

func (s *DatadogService) QueryMetricsHandler(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	if req.From == 0 {
		req.From = time.Now().Add(-1 * time.Hour).Unix()
	}
	if req.To == 0 {
		req.To = time.Now().Unix()
	}
	
	results, err := s.QueryMetrics(req.Query, req.From, req.To)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"results": results})
}

type CreateMonitorRequest struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Query    string `json:"query" binding:"required"`
	Message  string `json:"message"`
	Tags     string `json:"tags"`
	Priority int    `json:"priority"`
}

func (s *DatadogService) CreateMonitorHandler(c *gin.Context) {
	var req CreateMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	err := s.CreateMonitor(req.Name, req.Type, req.Query, req.Message, req.Tags, req.Priority)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "created"})
}

func (s *DatadogService) GetMonitorsHandler(c *gin.Context) {
	monitors, err := s.GetMonitors()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"monitors": monitors})
}

// Main

func main() {
	cfg := DatadogConfig{
		APIKey:               getEnv("DD_API_KEY", ""),
		APPKey:               getEnv("DD_APP_KEY", ""),
		Site:                 getEnv("DD_SITE", "datadoghq.com"),
		CustomMetricsPrefix:  getEnv("DD_METRICS_PREFIX", "tigerwallet"),
		DBHost:               getEnv("DB_HOST", "localhost"),
		DBPort:               getEnv("DB_PORT", "5432"),
		DBUser:               getEnv("DB_USER", "postgres"),
		DBPassword:           getEnv("DB_PASSWORD", "password"),
		DBName:               getEnv("DB_NAME", "datadog_db"),
		RedisHost:            getEnv("REDIS_HOST", "localhost"),
		RedisPort:            getEnv("REDIS_PORT", "6379"),
		ServerPort:           getEnv("DATADOG_SERVER_PORT", "8092"),
	}
	
	service, err := NewDatadogService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Datadog service: %v", err)
	}
	
	r := gin.Default()
	
	r.POST("/datadog/metric", service.SendMetricHandler)
	r.POST("/datadog/event", service.SendEventHandler)
	r.POST("/datadog/transaction", service.TransactionMetricHandler)
	r.POST("/datadog/query", service.QueryMetricsHandler)
	r.POST("/datadog/monitors", service.CreateMonitorHandler)
	r.GET("/datadog/monitors", service.GetMonitorsHandler)
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "datadog"})
	})
	
	log.Printf("Datadog Service starting on port %s", cfg.ServerPort)
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
