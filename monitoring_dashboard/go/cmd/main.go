// Monitoring Dashboard - Go Implementation
// High-performance, distributed monitoring for TigerWallet ecosystem

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	redislib "github.com/redis/go-redis/v9"
)

// Configuration
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
}

// ============ DATA MODELS ============

type ServiceType string

const (
	ServicePermission   ServiceType = "permission"
	ServiceConnection  ServiceType = "connection"
	ServiceFetcher     ServiceType = "fetcher"
	ServiceMasterWallet ServiceType = "master_wallet"
	ServiceUserWallet  ServiceType = "user_wallet"
	ServiceBots       ServiceType = "bots"
	ServiceProjectParty ServiceType = "project_party"
)

type ServiceStatus string

const (
	StatusHealthy   ServiceStatus = "healthy"
	StatusDegraded ServiceStatus = "degraded"
	StatusDown     ServiceStatus = "down"
	StatusUnknown  ServiceStatus = "unknown"
)

// Service health
type ServiceHealth struct {
	ID          uuid.UUID     `json:"id"`
	Service    ServiceType   `json:"service"`
	Status     ServiceStatus `json:"status"`
	Uptime     float64      `json:"uptime"` // percentage
	Latency    float64      `json:"latency"` // ms
	ErrorRate  float64      `json:"error_rate"` // percentage
	Requests   int64       `json:"requests"`
	Errors     int64       `json:"errors"`
	LastCheck  time.Time   `json:"last_check"`
	CreatedAt  time.Time   `json:"created_at"`
}

// Alert
type Alert struct {
	ID          uuid.UUID   `json:"id"`
	Service    ServiceType `json:"service"`
	AlertType  string     `json:"alert_type"` // latency, error, downtime
	Severity   string     `json:"severity"` // critical, warning, info
	Message    string     `json:"message"`
	Status     string     `json:"status"` // active, resolved
	CreatedAt  time.Time `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
}

// Metrics
type SystemMetrics struct {
	TotalRequests   int64   `json:"total_requests"`
	SuccessRate   float64 `json:"success_rate"`
	ErrorRate     float64 `json:"error_rate"`
	AvgLatency    float64 `json:"avg_latency"`
	P99Latency    float64 `json:"p99_latency"`
	ActiveConns   int64   `json:"active_connections"`
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryUsage   float64 `json:"memory_usage"`
	DiskUsage    float64 `json:"disk_usage"`
	Timestamp    time.Time `json:"timestamp"`
}

type ProductMetrics struct {
	Product       string  `json:"product"`
	Requests      int64   `json:"requests"`
	SuccessRate   float64 `json:"success_rate"`
	AvgLatency    float64 `json:"avg_latency"`
	ErrorRate     float64 `json:"error_rate"`
	ActiveClients int64   `json:"active_clients"`
}

// Global variables
var (
	db     *pgxpool.Pool
	redis           *redislib.Client
	config Config
	logger *log.Logger

	// In-memory cache
	healthCache    map[ServiceType]*ServiceHealth
	metricsCache   map[string]interface{}
	alertCache     []Alert
	cacheMu        sync.RWMutex

	// Counters
	totalRequests   int64
	totalErrors     int64
	requestsStart   time.Time
)

// ============ INITIALIZATION ============

func initDatabase() error {
	var err error
	dbURL := getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")

	db, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = db.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables
	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS service_health (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service VARCHAR(50) NOT NULL UNIQUE,
			status VARCHAR(50) DEFAULT 'unknown',
			uptime FLOAT DEFAULT 100.0,
			latency FLOAT DEFAULT 0.0,
			error_rate FLOAT DEFAULT 0.0,
			requests BIGINT DEFAULT 0,
			errors BIGINT DEFAULT 0,
			last_check TIMESTAMP DEFAULT NOW(),
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS monitoring_alerts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service VARCHAR(50) NOT NULL,
			alert_type VARCHAR(50) NOT NULL,
			severity VARCHAR(50) NOT NULL,
			message TEXT NOT NULL,
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP DEFAULT NOW(),
			resolved_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS metrics_history (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			metric_type VARCHAR(50) NOT NULL,
			service VARCHAR(50),
			value FLOAT NOT NULL,
			timestamp TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_health_service ON service_health(service);
		CREATE INDEX IF NOT EXISTS idx_alerts_status ON monitoring_alerts(status);
		CREATE TABLE IF NOT EXISTS incidents (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title VARCHAR(255) NOT NULL,
			description TEXT DEFAULT '',
			severity VARCHAR(50) NOT NULL DEFAULT 'minor',
			status VARCHAR(50) NOT NULL DEFAULT 'open',
			affected_services JSONB DEFAULT '[]',
			timeline JSONB DEFAULT '[]',
			created_at TIMESTAMP DEFAULT NOW(),
			resolved_at TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics_history(timestamp);
		CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
	`)

	return err
}

func initRedis() error {
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	opt, err := redislib.ParseURL(redisURL)
	if err != nil {
		return err
	}
	redis = redislib.NewClient(opt)
	return redis.Ping(context.Background()).Err()
}

// ============ MONITORING FUNCTIONS ============

func updateServiceHealth(service ServiceType, status ServiceStatus, latency float64, errorRate float64) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if healthCache == nil {
		healthCache = make(map[ServiceType]*ServiceHealth)
	}

	health, exists := healthCache[service]
	if !exists {
		health = &ServiceHealth{
			ID:         uuid.New(),
			Service:    service,
			Uptime:     100.0,
			CreatedAt:   time.Now(),
		}
		healthCache[service] = health
	}

	health.Status = status
	health.Latency = latency
	health.ErrorRate = errorRate
	health.LastCheck = time.Now()
	health.Requests++

	// Calculate uptime
	if status == StatusDown {
		health.Uptime = 0
	}

	// Store in Redis
	healthJSON, _ := json.Marshal(health)
	redis.Set(context.Background(), "health:"+string(service), healthJSON, 5*time.Minute)

	// Store in database
	db.Exec(context.Background(), `
		INSERT INTO service_health (id, service, status, uptime, latency, error_rate, requests, errors, last_check)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (service) DO UPDATE SET 
			status = $3, latency = $5, error_rate = $6, requests = requests + 1, last_check = NOW()
	`, health.ID, service, status, health.Uptime, latency, errorRate, health.Requests, health.Errors)

	// Check for alerts
	checkAlerts(service, status, latency, errorRate)
}

func checkAlerts(service ServiceType, status ServiceStatus, latency float64, errorRate float64) {
	// High latency alert
	if latency > 1000 {
		createAlert(service, "latency", "critical", fmt.Sprintf("High latency: %.2fms", latency))
	}

	// High error rate alert
	if errorRate > 5 {
		createAlert(service, "error", "critical", fmt.Sprintf("High error rate: %.2f%%", errorRate))
	}

	// Service down alert
	if status == StatusDown {
		createAlert(service, "downtime", "critical", "Service is down")
	}
}

func createAlert(service ServiceType, alertType, severity, message string) {
	alert := Alert{
		ID:         uuid.New(),
		Service:    service,
		AlertType:  alertType,
		Severity:   severity,
		Message:    message,
		Status:     "active",
		CreatedAt:  time.Now(),
	}

	cacheMu.Lock()
	alertCache = append(alertCache, alert)
	cacheMu.Unlock()

	// Store in database
	db.Exec(context.Background(), `
		INSERT INTO monitoring_alerts (id, service, alert_type, severity, message, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, alert.ID, service, alertType, severity, message, "active", alert.CreatedAt)

	// Publish to Redis for real-time notification
	alertJSON, _ := json.Marshal(alert)
	redis.Publish(context.Background(), "alerts", alertJSON)
}

func resolveAlert(alertID uuid.UUID) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	for i := range alertCache {
		if alertCache[i].ID == alertID {
			now := time.Now()
			alertCache[i].Status = "resolved"
			alertCache[i].ResolvedAt = &now
		}
	}

	db.Exec(context.Background(), `
		UPDATE monitoring_alerts SET status = 'resolved', resolved_at = NOW() WHERE id = $1
	`, alertID)
}

func getSystemMetrics() SystemMetrics {
	ctx := context.Background()

	// Get Redis metrics
	totalReq, _ := redis.Get(ctx, "metrics:requests:total").Int64()
	successReq, _ := redis.Get(ctx, "metrics:requests:success").Int64()
	errorReq, _ := redis.Get(ctx, "metrics:requests:error").Int64()
	avgLat, _ := redis.Get(ctx, "metrics:latency:avg").Float64()
	p99Lat, _ := redis.Get(ctx, "metrics:latency:p99").Float64()
	activeConns, _ := redis.Get(ctx, "connections:active").Int64()

	successRate := 0.0
	if totalReq > 0 {
		successRate = float64(successReq) / float64(totalReq) * 100
	}

	errorRate := 0.0
	if totalReq > 0 {
		errorRate = float64(errorReq) / float64(totalReq) * 100
	}

	return SystemMetrics{
		TotalRequests:  totalReq,
		SuccessRate:   successRate,
		ErrorRate:     errorRate,
		AvgLatency:    avgLat,
		P99Latency:    p99Lat,
		ActiveConns:   activeConns,
		CPUUsage:      getCPUUsage(),
		MemoryUsage:  getMemoryUsage(),
		DiskUsage:    getDiskUsage(),
		Timestamp:    time.Now(),
	}
}

func getProductMetrics() []ProductMetrics {
	products := []string{"master_wallet", "user_wallet", "bots", "bots_clients", "project_party"}
	var result []ProductMetrics

	for _, product := range products {
		requests, _ := redis.Get(context.Background(), fmt.Sprintf("product:%s:requests", product)).Int64()
		success, _ := redis.Get(context.Background(), fmt.Sprintf("product:%s:success", product)).Int64()
		errors, _ := redis.Get(context.Background(), fmt.Sprintf("product:%s:errors", product)).Int64()
		latency, _ := redis.Get(context.Background(), fmt.Sprintf("product:%s:latency", product)).Float64()
		clients, _ := redis.Get(context.Background(), fmt.Sprintf("product:%s:clients", product)).Int64()

		successRate := 0.0
		if requests > 0 {
			successRate = float64(success) / float64(requests) * 100
		}

		errorRate := 0.0
		if requests > 0 {
			errorRate = float64(errors) / float64(requests) * 100
		}

		result = append(result, ProductMetrics{
			Product:       product,
			Requests:     requests,
			SuccessRate:  successRate,
			AvgLatency:   latency,
			ErrorRate:    errorRate,
			ActiveClients: clients,
		})
	}

	return result
}

func getCPUUsage() float64 {
	// Simplified - in production would use actual system metrics
	return 45.5
}

func getMemoryUsage() float64 {
	return 62.3
}

func getDiskUsage() float64 {
	return 38.7
}

// Background metrics collection
func startMetricsCollection() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Collect service health from other services
		services := []ServiceType{
			ServicePermission,
			ServiceConnection,
			ServiceFetcher,
			ServiceMasterWallet,
			ServiceUserWallet,
			ServiceBots,
			ServiceProjectParty,
		}

		for _, service := range services {
			// Get from Redis or use defaults
			healthJSON, err := redis.Get(context.Background(), "health:"+string(service)).Result()
			if err != nil {
				// Service not responding - mark as down
				updateServiceHealth(service, StatusUnknown, 0, 0)
				continue
			}

			var health ServiceHealth
			json.Unmarshal([]byte(healthJSON), &health)

			// Determine status based on metrics
			status := StatusHealthy
			if health.Latency > 1000 || health.ErrorRate > 5 {
				status = StatusDegraded
			}
			if health.ErrorRate > 20 {
				status = StatusDown
			}

			updateServiceHealth(service, status, health.Latency, health.ErrorRate)
		}

		// Record metrics to history
		metrics := getSystemMetrics()
		db.Exec(context.Background(), `
			INSERT INTO metrics_history (metric_type, service, value)
			VALUES ('requests', 'system', $1), ('latency', 'system', $2), ('errors', 'system', $3)
		`, metrics.TotalRequests, metrics.AvgLatency, totalErrors)
	}
}

// ============ HTTP HANDLERS ============

func HealthCheck(c *gin.Context) {
	ctx := context.Background()
	
	dbStatus := "healthy"
	if err := db.Ping(ctx); err != nil {
		dbStatus = "unhealthy"
	}
	
	redisStatus := "healthy"
	if err := redis.Ping(ctx).Err(); err != nil {
		redisStatus = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"database": dbStatus,
		"redis":    redisStatus,
		"timestamp": time.Now(),
	})
}

// GetSystemMetrics - Get overall system metrics
func GetSystemMetrics(c *gin.Context) {
	metrics := getSystemMetrics()
	c.JSON(http.StatusOK, metrics)
}

// GetProductMetrics - Get metrics per product
func GetProductMetrics(c *gin.Context) {
	metrics := getProductMetrics()
	c.JSON(http.StatusOK, gin.H{"products": metrics})
}

// GetServiceHealth - Get health of all services
func GetServiceHealth(c *gin.Context) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()

	var healthList []ServiceHealth
	for _, health := range healthCache {
		healthList = append(healthList, *health)
	}

	c.JSON(http.StatusOK, gin.H{"services": healthList})
}

// GetAlerts - Get active alerts
func GetAlerts(c *gin.Context) {
	status := c.DefaultQuery("status", "active")

	rows, err := db.Query(context.Background(), `
		SELECT id, service, alert_type, severity, message, status, created_at, resolved_at
		FROM monitoring_alerts
		WHERE status = $1
		ORDER BY created_at DESC
	`, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var alert Alert
		if err := rows.Scan(&alert.ID, &alert.Service, &alert.AlertType, &alert.Severity, &alert.Message, &alert.Status, &alert.CreatedAt, &alert.ResolvedAt); err != nil {
			continue
		}
		alerts = append(alerts, alert)
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// ResolveAlert - Resolve an alert
func ResolveAlert(c *gin.Context) {
	alertID := c.Param("id")
	id, err := uuid.Parse(alertID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	resolveAlert(id)
	c.JSON(http.StatusOK, gin.H{"message": "alert resolved"})
}

// GetDashboardData - Get all dashboard data
func GetDashboardData(c *gin.Context) {
	systemMetrics := getSystemMetrics()
	productMetrics := getProductMetrics()

	cacheMu.RLock()
	healthList := make([]ServiceHealth, 0, len(healthCache))
	for _, h := range healthCache {
		healthList = append(healthList, *h)
	}
	alertCount := len(alertCache)
	cacheMu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"system":       systemMetrics,
		"products":     productMetrics,
		"services":     healthList,
		"active_alerts": alertCount,
		"timestamp":    time.Now(),
	})
}

// RecordMetric - Record a metric
func RecordMetric(c *gin.Context) {
	var req struct {
		Service  string  `json:"service" binding:"required"`
		Metric   string  `json:"metric" binding:"required"`
		Value    float64 `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update Redis counters
	switch req.Metric {
	case "requests":
		redis.Incr(context.Background(), "metrics:requests:total")
		totalRequests++
	case "success":
		redis.Incr(context.Background(), "metrics:requests:success")
	case "error":
		redis.Incr(context.Background(), "metrics:requests:error")
		totalErrors++
	case "latency":
		redis.Set(context.Background(), "metrics:latency:avg", req.Value, 0)
	}

	// Store in history
	db.Exec(context.Background(), `
		INSERT INTO metrics_history (metric_type, service, value)
		VALUES ($1, $2, $3)
	`, req.Metric, req.Service, req.Value)

	c.JSON(http.StatusOK, gin.H{"message": "metric recorded"})
}

// GetHistory - Get metrics history
func GetHistory(c *gin.Context) {
	metricType := c.Query("metric")
	service := c.Query("service")
	limit := c.DefaultQuery("limit", "100")

	query := `
		SELECT metric_type, service, value, timestamp
		FROM metrics_history
		WHERE 1=1
	`
	if metricType != "" {
		query += fmt.Sprintf(" AND metric_type = '%s'", metricType)
	}
	if service != "" {
		query += fmt.Sprintf(" AND service = '%s'", service)
	}
	query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT %s", limit)

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type HistoryPoint struct {
		MetricType string    `json:"metric_type"`
		Service    string    `json:"service"`
		Value      float64   `json:"value"`
		Timestamp  time.Time `json:"timestamp"`
	}

	var history []HistoryPoint
	for rows.Next() {
		var h HistoryPoint
		if err := rows.Scan(&h.MetricType, &h.Service, &h.Value, &h.Timestamp); err != nil {
			continue
		}
		history = append(history, h)
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// ============ MAIN ============


// ============ INCIDENT HANDLERS ============

type Incident struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Severity         string     `json:"severity"`
	Status           string     `json:"status"`
	AffectedServices []string   `json:"affected_services"`
	Timeline         []string   `json:"timeline"`
	CreatedAt        time.Time  `json:"created_at"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
}

func GetIncidents(c *gin.Context) {
	status := c.Query("status")
	query := `SELECT id, title, description, severity, status, affected_services, timeline, created_at, resolved_at FROM incidents`
	args := []interface{}{}
	if status != "" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC LIMIT 200`

	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	incidents := []Incident{}
	for rows.Next() {
		var inc Incident
		var services, timeline []byte
		if err := rows.Scan(&inc.ID, &inc.Title, &inc.Description, &inc.Severity, &inc.Status, &services, &timeline, &inc.CreatedAt, &inc.ResolvedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		json.Unmarshal(services, &inc.AffectedServices)
		json.Unmarshal(timeline, &inc.Timeline)
		incidents = append(incidents, inc)
	}
	c.JSON(http.StatusOK, gin.H{"incidents": incidents})
}

func CreateIncident(c *gin.Context) {
	var req struct {
		Title            string   `json:"title" binding:"required"`
		Description      string   `json:"description"`
		Severity         string   `json:"severity" binding:"required"`
		AffectedServices []string `json:"affected_services"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	switch req.Severity {
	case "critical", "major", "minor":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "severity must be critical, major or minor"})
		return
	}
	services, _ := json.Marshal(req.AffectedServices)
	timeline, _ := json.Marshal([]string{"incident created"})

	var id string
	var createdAt time.Time
	err := db.QueryRow(context.Background(),
		`INSERT INTO incidents (title, description, severity, affected_services, timeline) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		req.Title, req.Description, req.Severity, services, timeline).Scan(&id, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "open", "created_at": createdAt})
}

func ResolveIncident(c *gin.Context) {
	id := c.Param("id")
	tag, err := db.Exec(context.Background(),
		`UPDATE incidents SET status = 'resolved', resolved_at = NOW(), timeline = timeline || $2::jsonb WHERE id = $1 AND status != 'resolved'`,
		id, `"incident resolved"`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "open incident not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Incident resolved"})
}

func main() {
	logger = log.New(os.Stdout, "Monitoring Dashboard: ", log.LstdFlags)
	logger.Println("Starting Monitoring Dashboard...")

	config.Port = getEnv("MONITOR_PORT", "8094")
	config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
	config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")

	requestsStart = time.Now()

	if err := initDatabase(); err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	logger.Println("Database connected")

	if err := initRedis(); err != nil {
		logger.Fatalf("Failed to initialize Redis: %v", err)
	}
	logger.Println("Redis connected")

	// Start background metrics collection
	go startMetricsCollection()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.GET("/health", HealthCheck)

	// Dashboard
	router.GET("/api/v1/dashboard", GetDashboardData)

	// Metrics
	router.GET("/api/v1/metrics/system", GetSystemMetrics)
	router.GET("/api/v1/metrics/products", GetProductMetrics)
	router.GET("/api/v1/metrics/history", GetHistory)
	router.POST("/api/v1/metrics", RecordMetric)

	// Service health
	router.GET("/api/v1/health", GetServiceHealth)

	// Alerts
	router.GET("/api/v1/alerts", GetAlerts)
	router.PUT("/api/v1/alerts/:id/resolve", ResolveAlert)

	// Incidents
	router.GET("/api/v1/incidents", GetIncidents)
	router.POST("/api/v1/incidents", CreateIncident)
	router.PUT("/api/v1/incidents/:id/resolve", ResolveIncident)

	logger.Printf("Starting server on port %s", config.Port)
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Println("Server started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	db.Close()
	redis.Close()
	logger.Println("Server exited")
}
