package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := loadConfig()

	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Prometheus metrics
	initMetrics()

	router := gin.Default()
	router.Use(corsMiddleware())

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-monitoring"})
	})

	api := router.Group("/api/v1/monitoring")
	{
		// Health checks
		api.GET("/health", getHealthChecksHandler)
		api.POST("/health", createHealthCheckHandler)
		api.PUT("/health/:id", updateHealthCheckHandler)
		api.DELETE("/health/:id", deleteHealthCheckHandler)

		// Metrics
		api.GET("/metrics", getMetricsHandler)
		api.GET("/metrics/:tenant_id", getTenantMetricsHandler)

		// Alerts
		api.GET("/alerts", getAlertsHandler)
		api.POST("/alerts", createAlertHandler)
		api.PUT("/alerts/:id", updateAlertHandler)
		api.DELETE("/alerts/:id", deleteAlertHandler)

		// Dashboards
		api.GET("/dashboards", getDashboardsHandler)
		api.POST("/dashboards", createDashboardHandler)
		api.GET("/dashboards/:id", getDashboardHandler)

		// Logs
		api.GET("/logs", getLogsHandler)
		api.POST("/logs", ingestLogHandler)

		// Status
		api.GET("/status", getSystemStatusHandler)
		api.GET("/status/tenants", getTenantsStatusHandler)

		// Incidents
		api.GET("/incidents", getIncidentsHandler)
		api.POST("/incidents", createIncidentHandler)
		api.PUT("/incidents/:id/resolve", resolveIncidentHandler)

		// Uptime
		api.GET("/uptime", getUptimeHandler)
		api.GET("/uptime/:tenant_id", getTenantUptimeHandler)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	// Start background monitoring
	go startMonitoring(cfg)

	go func() {
		log.Printf("Monitoring service starting on port %s", cfg.Port)
		srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

type Config struct {
	Port     string
	Database DatabaseConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

func loadConfig() *Config {
	return &Config{
		Port: getEnv("MONITORING_PORT", "9003"),
		Database: DatabaseConfig{
			Host:     getEnv("MONITORING_DB_HOST", "localhost"),
			Port:     getEnvInt("MONITORING_DB_PORT", 5432),
			User:     getEnv("MONITORING_DB_USER", "tigerwallet"),
			Password: getEnv("MONITORING_DB_PASSWORD", "password"),
			DBName:   getEnv("MONITORING_DB_NAME", "tigerwallet_monitoring"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	var v int
	_, err := fmt.Sscan(os.Getenv(key), &v)
	if err != nil {
		return defaultValue
	}
	return v
}

// Metrics
var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tigerwallet_requests_total",
			Help: "Total number of requests",
		},
		[]string{"service", "endpoint", "method", "status"},
	)

	requestsDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tigerwallet_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"service", "endpoint", "method"},
	)

	activeConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tigerwallet_active_connections",
			Help: "Number of active connections",
		},
		[]string{"service"},
	)

	errorsTotal = prometheus.NewCounterVec(
		petheus.CounterOpts{
			Name: "tigerwallet_errors_total",
			Help: "Total number of errors",
		},
		[]string{"service", "error_type"},
	)

	apiCallsPerTenant = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tigerwallet_api_calls_per_tenant",
			Help: "API calls per tenant",
		},
		[]string{"tenant_id", "endpoint"},
	)

	tenantUptime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tigerwallet_tenant_uptime",
			Help: "Tenant uptime percentage",
		},
		[]string{"tenant_id"},
	)

	queueSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tigerwallet_queue_size",
			Help: "Queue size",
		},
		[]string{"queue_name"},
	)

	cpuUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tigerwallet_cpu_usage_percent",
			Help: "CPU usage percentage",
		},
		[]string{"service", "host"},
	)

	memoryUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tigerwallet_memory_usage_bytes",
			Help: "Memory usage in bytes",
		},
		[]string{"service", "host"},
	)
)

func initMetrics() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(requestsDuration)
	prometheus.MustRegister(activeConnections)
	prometheus.MustRegister(errorsTotal)
	prometheus.MustRegister(apiCallsPerTenant)
	prometheus.MustRegister(tenantUptime)
	prometheus.MustRegister(queueSize)
	prometheus.MustRegister(cpuUsage)
	prometheus.MustRegister(memoryUsage)
}

// Models
type HealthCheck struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	URL         string    `json:"url" db:"url"`
	Type        string    `json:"type" db:"type"` // http, tcp, ping
	Interval    int       `json:"interval" db:"interval"` // seconds
	Timeout     int       `json:"timeout" db:"timeout"` // seconds
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	Status      string    `json:"status" db:"status"` // up, down, pending
	LastCheckAt *time.Time `json:"last_check_at" db:"last_check_at"`
	LastResponseTime *int `json:"last_response_time" db:"last_response_time"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Metric struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID string    `json:"tenant_id" db:"tenant_id"`
	Service  string    `json:"service" db:"service"`
	Metric   string    `json:"metric" db:"metric"`
	Value    float64   `json:"value" db:"value"`
	Labels   string    `json:"labels" db:"labels"` // JSON
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

type Alert struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	Name        string    `json:"name" db:"name"`
	Condition   string    `json:"condition" db:"condition"` // JSON
	Severity    string    `json:"severity" db:"severity"` // critical, warning, info
	Status      string    `json:"status" db:"status"` // firing, resolved
	TriggeredAt *time.Time `json:"triggered_at" db:"triggered_at"`
	ResolvedAt  *time.Time `json:"resolved_at" db:"resolved_at"`
	Message     string    `json:"message" db:"message"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Dashboard struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	Name      string    `json:"name" db:"name"`
	Layout    string    `json:"layout" db:"layout"` // JSON
	Widgets   string    `json:"widgets" db:"widgets"` // JSON
	CreatedBy string    `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type LogEntry struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	Service   string    `json:"service" db:"service"`
	Level     string    `json:"level" db:"level"` // debug, info, warn, error
	Message   string    `json:"message" db:"message"`
	Metadata  string    `json:"metadata" db:"metadata"` // JSON
	Source    string    `json:"source" db:"source"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

type Incident struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	Severity    string    `json:"severity" db:"severity"` // critical, major, minor
	Status      string    `json:"status" db:"status"` // open, investigating, resolved
	AffectedServices []string `json:"affected_services" db:"affected_services"` // JSON
	Timeline   string    `json:"timeline" db:"timeline"` // JSON
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at" db:"resolved_at"`
}

// Handlers
func getHealthChecksHandler(c *gin.Context) {
	tenantID := c.Query("tenant_id")

	checks := []map[string]interface{}{
		{
			"id":     uuid.New().String(),
			"name":   "API Health",
			"url":    "https://api.tigerwallet.com/health",
			"type":   "http",
			"status": "up",
		},
		{
			"id":     uuid.New().String(),
			"name":   "Database Health",
			"url":    "tcp://db.tigerwallet.com:5432",
			"type":   "tcp",
			"status": "up",
		},
	}

	c.JSON(http.StatusOK, gin.H{"health_checks": checks})
}

func createHealthCheckHandler(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		URL      string `json:"url" binding:"required"`
		Type     string `json:"type" binding:"required"`
		Interval int    `json:"interval"`
		TenantID string `json:"tenant_id"`
	}
	c.ShouldBindJSON(&req)

	check := map[string]interface{}{
		"id":       uuid.New().String(),
		"name":     req.Name,
		"url":      req.URL,
		"type":     req.Type,
		"interval": req.Interval,
		"status":   "pending",
	}

	c.JSON(http.StatusCreated, gin.H{"health_check": check})
}

func updateHealthCheckHandler(c *gin.Context) {
	checkID := c.Param("id")

	var req struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Interval int    `json:"interval"`
		Status   string `json:"status"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{"message": "Health check updated"})
}

func deleteHealthCheckHandler(c *gin.Context) {
	checkID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"message": "Health check deleted"})
}

func getMetricsHandler(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	service := c.Query("service")
	from := c.Query("from")
	to := c.Query("to")

	metrics := map[string]interface{}{
		"requests_per_second": 1500.0,
		"avg_response_time_ms": 45.0,
		"error_rate": 0.01,
		"active_connections": 500,
	}

	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}

func getTenantMetricsHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	metrics := map[string]interface{}{
		"tenant_id":        tenantID,
		"api_calls":        100000,
		"api_calls_limit":  1000000,
		"bandwidth_mb":     500,
		"storage_mb":      1000,
		"uptime_percent":   99.99,
	}

	c.JSON(http.StatusOK, metrics)
}

func getAlertsHandler(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	status := c.Query("status")

	alerts := []map[string]interface{}{
		{
			"id":          uuid.New().String(),
			"name":        "High Error Rate",
			"severity":    "warning",
			"status":      "firing",
			"triggered_at": time.Now().Add(-1 * time.Hour).Unix(),
		},
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func createAlertHandler(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		Condition string `json:"condition" binding:"required"`
		Severity  string `json:"severity" binding:"required"`
		TenantID  string `json:"tenant_id"`
	}
	c.ShouldBindJSON(&req)

	alert := map[string]interface{}{
		"id":       uuid.New().String(),
		"name":     req.Name,
		"condition": req.Condition,
		"severity": req.Severity,
		"status":   "firing",
	}

	c.JSON(http.StatusCreated, gin.H{"alert": alert})
}

func updateAlertHandler(c *gin.Context) {
	alertID := c.Param("id")

	var req struct {
		Status string `json:"status"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{"message": "Alert updated"})
}

func deleteAlertHandler(c *gin.Context) {
	alertID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"message": "Alert deleted"})
}

func getDashboardsHandler(c *gin.Context) {
	tenantID := c.Query("tenant_id")

	dashboards := []map[string]interface{}{
		{
			"id":        uuid.New().String(),
			"name":      "Main Dashboard",
			"widgets":   6,
			"created_at": time.Now().Unix(),
		},
	}

	c.JSON(http.StatusOK, gin.H{"dashboards": dashboards})
}

func createDashboardHandler(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Layout   string `json:"layout"`
		Widgets  string `json:"widgets"`
		TenantID string `json:"tenant_id"`
	}
	c.ShouldBindJSON(&req)

	dashboard := map[string]interface{}{
		"id":         uuid.New().String(),
		"name":       req.Name,
		"layout":     req.Layout,
		"widgets":    req.Widgets,
		"created_at": time.Now().Unix(),
	}

	c.JSON(http.StatusCreated, gin.H{"dashboard": dashboard})
}

func getDashboardHandler(c *gin.Context) {
	dashboardID := c.Param("id")

	dashboard := map[string]interface{}{
		"id":       dashboardID,
		"name":     "Main Dashboard",
		"layout":   "{\"rows\": 3}",
		"widgets":  "[{\"type\": \"chart\", \"title\": \"Requests\"}]",
	}

	c.JSON(http.StatusOK, dashboard)
}

func getLogsHandler(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	service := c.Query("service")
	level := c.Query("level")
	from := c.Query("from")
	to := c.Query("to")
	limit := c.DefaultQuery("limit", "100")

	logs := []map[string]interface{}{
		{
			"id":       uuid.New().String(),
			"service":  "api",
			"level":    "info",
			"message":  "Request processed",
			"timestamp": time.Now().Unix(),
		},
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": 1000})
}

func ingestLogHandler(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id"`
		Service  string `json:"service" binding:"required"`
		Level    string `json:"level" binding:"required"`
		Message  string `json:"message" binding:"required"`
		Metadata string `json:"metadata"`
	}
	c.ShouldBindJSON(&req)

	logEntry := map[string]interface{}{
		"id":        uuid.New().String(),
		"tenant_id": req.TenantID,
		"service":   req.Service,
		"level":     req.Level,
		"message":   req.Message,
		"timestamp": time.Now().Unix(),
	}

	c.JSON(http.StatusAccepted, gin.H{"log": logEntry})
}

func getSystemStatusHandler(c *gin.Context) {
	status := map[string]interface{}{
		"status":         "healthy",
		"version":        "1.0.0",
		"uptime_seconds": 86400,
		"services": map[string]interface{}{
			"api":        "healthy",
			"database":   "healthy",
			"cache":      "healthy",
			"queue":      "healthy",
		},
		"resources": map[string]interface{}{
			"cpu_percent":    45.0,
			"memory_percent": 60.0,
			"disk_percent":   30.0,
		},
	}

	c.JSON(http.StatusOK, status)
}

func getTenantsStatusHandler(c *gin.Context) {
	tenants := []map[string]interface{}{
		{
			"tenant_id":   "t1",
			"name":       "Company A",
			"status":     "active",
			"uptime":     99.99,
			"api_calls":  50000,
			"last_active": time.Now().Unix(),
		},
		{
			"tenant_id":   "t2",
			"name":       "Company B",
			"status":     "active",
			"uptime":     99.95,
			"api_calls":  30000,
			"last_active": time.Now().Unix(),
		},
	}

	c.JSON(http.StatusOK, gin.H{"tenants": tenants})
}

func getIncidentsHandler(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	status := c.Query("status")

	incidents := []map[string]interface{}{
		{
			"id":          uuid.New().String(),
			"title":       "API Latency Spike",
			"severity":    "minor",
			"status":      "resolved",
			"created_at":  time.Now().Add(-24 * time.Hour).Unix(),
			"resolved_at": time.Now().Add(-23 * time.Hour).Unix(),
		},
	}

	c.JSON(http.StatusOK, gin.H{"incidents": incidents})
}

func createIncidentHandler(c *gin.Context) {
	var req struct {
		Title       string   `json:"title" binding:"required"`
		Description string   `json:"description"`
		Severity    string   `json:"severity" binding:"required"`
		TenantID    string   `json:"tenant_id"`
	}
	c.ShouldBindJSON(&req)

	incident := map[string]interface{}{
		"id":          uuid.New().String(),
		"title":       req.Title,
		"description": req.Description,
		"severity":    req.Severity,
		"status":      "open",
		"created_at":  time.Now().Unix(),
	}

	c.JSON(http.StatusCreated, gin.H{"incident": incident})
}

func resolveIncidentHandler(c *gin.Context) {
	incidentID := c.Param("id")

	var req struct {
		Resolution string `json:"resolution"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{
		"id":         incidentID,
		"status":     "resolved",
		"resolved_at": time.Now().Unix(),
	})
}

func getUptimeHandler(c *gin.Context) {
	uptime := map[string]interface{}{
		"overall":   99.99,
		"last_24h":  99.95,
		"last_7d":   99.98,
		"last_30d":  99.99,
		"incidents": 2,
	}

	c.JSON(http.StatusOK, uptime)
}

func getTenantUptimeHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	uptime := map[string]interface{}{
		"tenant_id": tenantID,
		"uptime":    99.99,
		"last_24h":  99.95,
		"last_7d":   99.98,
		"last_30d":  99.99,
		"api_calls": 50000,
		"errors":     5,
	}

	c.JSON(http.StatusOK, uptime)
}

// Background monitoring
func startMonitoring(cfg *Config) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		// Update metrics
		cpuUsage.WithLabelValues("api", "server1").Set(45.0)
		memoryUsage.WithLabelValues("api", "server1").Set(1024 * 1024 * 512)
		queueSize.WithLabelValues("default").Set(100)
	}
}

// Middleware
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// Database
type DB struct{}

func initDatabase(cfg *Config) (*DB, error) {
	log.Printf("Connecting to PostgreSQL at %s:%d", cfg.Database.Host, cfg.Database.Port)
	return &DB{}, nil
}

func (d *DB) Close() {}
