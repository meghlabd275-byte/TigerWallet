// Observability Service - PostgreSQL Version
// Comprehensive monitoring, logging, and tracing for TigerWallet ecosystem

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Configuration
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
}

// Observability Models
type LogEntry struct {
	ID          uuid.UUID              `json:"id"`
	Service     string                 `json:"service"`
	Level       string                 `json:"level"` // debug, info, warn, error, fatal
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata"`
	TraceID     string                 `json:"trace_id"`
	SpanID      string                 `json:"span_id"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
}

type MetricEntry struct {
	ID          uuid.UUID              `json:"id"`
	Service     string                 `json:"service"`
	MetricName  string                 `json:"metric_name"`
	Value       float64                `json:"value"`
	Unit        string                 `json:"unit"`
	Tags        map[string]interface{} `json:"tags"`
	Timestamp   time.Time              `json:"timestamp"`
}

type TraceSpan struct {
	ID          uuid.UUID   `json:"id"`
	TraceID     string      `json:"trace_id"`
	SpanID      string      `json:"span_id"`
	ParentID    string      `json:"parent_id"`
	Service     string      `json:"service"`
	Operation   string      `json:"operation"`
	StartTime   time.Time   `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	Duration    int64       `json:"duration"` // in milliseconds
	Status      string      `json:"status"` // ok, error
	Tags        string      `json:"tags"` // JSON object
	Logs        string      `json:"logs"` // JSON array
}

type Alert struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	AlertType   string    `json:"alert_type"` // metric, log, trace
	Condition   string    `json:"condition"` // JSON expression
	Severity    string    `json:"severity"` // critical, high, medium, low
	Status      string    `json:"status"` // active, resolved, silenced
	Service      string    `json:"service"`
	Message      string    `json:"message"`
	Metadata    string    `json:"metadata"` // JSON object
	TriggeredAt *time.Time `json:"triggered_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type ServiceHealth struct {
	ID          uuid.UUID `json:"id"`
	Service     string    `json:"service"`
	Status      string    `json:"status"` // healthy, degraded, down
	Uptime      float64   `json:"uptime"` // percentage
	Latency     float64   `json:"latency"` // ms
	ErrorRate   float64   `json:"error_rate"` // percentage
	RequestsMin int       `json:"requests_min"`
	LastCheckAt time.Time `json:"last_check_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type Dashboard struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Panels      string    `json:"panels"` // JSON array
	Service     string    `json:"service"`
	OwnerID     uuid.UUID `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Global variables
var (
	db     *pgxpool.Pool
	redis  *redis.Client
	config Config
	logger *log.Logger
)

// Initialize database
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
		CREATE TABLE IF NOT EXISTS log_entries (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service VARCHAR(255) NOT NULL,
			level VARCHAR(50) NOT NULL,
			message TEXT NOT NULL,
			metadata JSONB,
			trace_id VARCHAR(255),
			span_id VARCHAR(255),
			source VARCHAR(255),
			timestamp TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS metric_entries (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service VARCHAR(255) NOT NULL,
			metric_name VARCHAR(255) NOT NULL,
			value FLOAT NOT NULL,
			unit VARCHAR(50),
			tags JSONB,
			timestamp TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS trace_spans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			trace_id VARCHAR(255) NOT NULL,
			span_id VARCHAR(255) NOT NULL,
			parent_id VARCHAR(255),
			service VARCHAR(255) NOT NULL,
			operation VARCHAR(255) NOT NULL,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP,
			duration BIGINT,
			status VARCHAR(50) DEFAULT 'ok',
			tags JSONB,
			logs JSONB
		);

		CREATE TABLE IF NOT EXISTS alerts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			alert_type VARCHAR(50) NOT NULL,
			condition TEXT,
			severity VARCHAR(50) NOT NULL,
			status VARCHAR(50) DEFAULT 'active',
			service VARCHAR(255),
			message TEXT,
			metadata JSONB,
			triggered_at TIMESTAMP,
			resolved_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS service_health (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service VARCHAR(255) NOT NULL,
			status VARCHAR(50) DEFAULT 'healthy',
			uptime FLOAT DEFAULT 100.0,
			latency FLOAT DEFAULT 0.0,
			error_rate FLOAT DEFAULT 0.0,
			requests_min INTEGER DEFAULT 0,
			last_check_at TIMESTAMP DEFAULT NOW(),
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS observability_dashboards (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			panels JSONB,
			service VARCHAR(255),
			owner_id UUID NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_logs_service ON log_entries(service);
		CREATE INDEX IF NOT EXISTS idx_logs_level ON log_entries(level);
		CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON log_entries(timestamp);
		CREATE INDEX IF NOT EXISTS idx_metrics_service ON metric_entries(service);
		CREATE INDEX IF NOT EXISTS idx_metrics_name ON metric_entries(metric_name);
		CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metric_entries(timestamp);
		CREATE INDEX IF NOT EXISTS idx_traces_trace ON trace_spans(trace_id);
		CREATE INDEX IF NOT EXISTS idx_traces_service ON trace_spans(service);
		CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
		CREATE INDEX IF NOT EXISTS idx_health_service ON service_health(service);
	`)

	return err
}

// Initialize Redis
func initRedis() error {
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return err
	}

	redis = redis.NewClient(opt)
	return redis.Ping(context.Background()).Err()
}

// Handlers

// IngestLog - Ingest log entry
func IngestLog(c *gin.Context) {
	var log LogEntry
	if err := c.ShouldBindJSON(&log); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.ID = uuid.New()
	log.Timestamp = time.Now()

	metadataJSON, _ := json.Marshal(log.Metadata)

	_, err := db.Exec(context.Background(), `
		INSERT INTO log_entries (id, service, level, message, metadata, trace_id, span_id, source, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, log.ID, log.Service, log.Level, log.Message, metadataJSON, log.TraceID, log.SpanID, log.Source, log.Timestamp)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"log_id": log.ID})
}

// QueryLogs - Query logs
func QueryLogs(c *gin.Context) {
	service := c.Query("service")
	level := c.Query("level")
	limit := c.DefaultQuery("limit", "100")

	query := `
		SELECT id, service, level, message, metadata, trace_id, span_id, source, timestamp
		FROM log_entries
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if service != "" {
		query += fmt.Sprintf(" AND service = $%d", argNum)
		args = append(args, service)
		argNum++
	}
	if level != "" {
		query += fmt.Sprintf(" AND level = $%d", argNum)
		args = append(args, level)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d", argNum)
	args = append(args, limit)

	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var log LogEntry
		var metadata []byte
		if err := rows.Scan(&log.ID, &log.Service, &log.Level, &log.Message, &metadata, &log.TraceID, &log.SpanID, &log.Source, &log.Timestamp); err != nil {
			continue
		}
		json.Unmarshal(metadata, &log.Metadata)
		logs = append(logs, log)
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// RecordMetric - Record metric
func RecordMetric(c *gin.Context) {
	var metric MetricEntry
	if err := c.ShouldBindJSON(&metric); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metric.ID = uuid.New()
	metric.Timestamp = time.Now()

	tagsJSON, _ := json.Marshal(metric.Tags)

	_, err := db.Exec(context.Background(), `
		INSERT INTO metric_entries (id, service, metric_name, value, unit, tags, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, metric.ID, metric.Service, metric.MetricName, metric.Value, metric.Unit, tagsJSON, metric.Timestamp)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, metric)
}

// QueryMetrics - Query metrics
func QueryMetrics(c *gin.Context) {
	service := c.Query("service")
	metricName := c.Query("metric")
	limit := c.DefaultQuery("limit", "100")

	query := `
		SELECT id, service, metric_name, value, unit, tags, timestamp
		FROM metric_entries
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if service != "" {
		query += fmt.Sprintf(" AND service = $%d", argNum)
		args = append(args, service)
		argNum++
	}
	if metricName != "" {
		query += fmt.Sprintf(" AND metric_name = $%d", argNum)
		args = append(args, metricName)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d", argNum)
	args = append(args, limit)

	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var metrics []MetricEntry
	for rows.Next() {
		var metric MetricEntry
		var tags []byte
		if err := rows.Scan(&metric.ID, &metric.Service, &metric.MetricName, &metric.Value, &metric.Unit, &tags, &metric.Timestamp); err != nil {
			continue
		}
		json.Unmarshal(tags, &metric.Tags)
		metrics = append(metrics, metric)
	}

	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}

// StartSpan - Start trace span
func StartSpan(c *gin.Context) {
	var span TraceSpan
	if err := c.ShouldBindJSON(&span); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	span.ID = uuid.New()
	span.StartTime = time.Now()

	_, err := db.Exec(context.Background(), `
		INSERT INTO trace_spans (id, trace_id, span_id, parent_id, service, operation, start_time, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, span.ID, span.TraceID, span.SpanID, span.ParentID, span.Service, span.Operation, span.StartTime, span.Status)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, span)
}

// EndSpan - End trace span
func EndSpan(c *gin.Context) {
	spanID := c.Param("span_id")
	
	endTime := time.Now()
	duration := endTime.UnixMilli()

	_, err := db.Exec(context.Background(), `
		UPDATE trace_spans SET end_time = $1, duration = $2 WHERE span_id = $3
	`, endTime, duration, spanID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "span ended"})
}

// QueryTraces - Query traces
func QueryTraces(c *gin.Context) {
	traceID := c.Query("trace_id")

	rows, err := db.Query(context.Background(), `
		SELECT id, trace_id, span_id, parent_id, service, operation, start_time, end_time, duration, status, tags, logs
		FROM trace_spans
		WHERE trace_id = $1
		ORDER BY start_time ASC
	`, traceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var spans []TraceSpan
	for rows.Next() {
		var span TraceSpan
		var tags []byte
		var logs []byte
		if err := rows.Scan(&span.ID, &span.TraceID, &span.SpanID, &span.ParentID, &span.Service, &span.Operation, &span.StartTime, &span.EndTime, &span.Duration, &span.Status, &tags, &logs); err != nil {
			continue
		}
		span.Tags = string(tags)
		span.Logs = string(logs)
		spans = append(spans, span)
	}

	c.JSON(http.StatusOK, gin.H{"spans": spans})
}

// CreateAlert - Create alert
func CreateAlert(c *gin.Context) {
	var alert Alert
	if err := c.ShouldBindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	alert.ID = uuid.New()
	alert.Status = "active"
	alert.CreatedAt = time.Now()

	metadataJSON, _ := json.Marshal(alert.Metadata)

	_, err := db.Exec(context.Background(), `
		INSERT INTO alerts (id, name, alert_type, condition, severity, status, service, message, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, alert.ID, alert.Name, alert.AlertType, alert.Condition, alert.Severity, alert.Status, alert.Service, alert.Message, metadataJSON, alert.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, alert)
}

// GetAlerts - Get alerts
func GetAlerts(c *gin.Context) {
	status := c.DefaultQuery("status", "active")

	rows, err := db.Query(context.Background(), `
		SELECT id, name, alert_type, condition, severity, status, service, message, metadata, triggered_at, resolved_at, created_at
		FROM alerts
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
		var metadata []byte
		if err := rows.Scan(&alert.ID, &alert.Name, &alert.AlertType, &alert.Condition, &alert.Severity, &alert.Status, &alert.Service, &alert.Message, &metadata, &alert.TriggeredAt, &alert.ResolvedAt, &alert.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(metadata, &alert.Metadata)
		alerts = append(alerts, alert)
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// UpdateServiceHealth - Update service health
func UpdateServiceHealth(c *gin.Context) {
	var health ServiceHealth
	if err := c.ShouldBindJSON(&health); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	health.ID = uuid.New()
	health.LastCheckAt = time.Now()
	health.CreatedAt = time.Now()

	_, err := db.Exec(context.Background(), `
		INSERT INTO service_health (id, service, status, uptime, latency, error_rate, requests_min, last_check_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (service) DO UPDATE SET
			status = $3, uptime = $4, latency = $5, error_rate = $6, requests_min = $7, last_check_at = $8
	`, health.ID, health.Service, health.Status, health.Uptime, health.Latency, health.ErrorRate, health.RequestsMin, health.LastCheckAt, health.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetServiceHealth - Get service health
func GetServiceHealth(c *gin.Context) {
	service := c.Query("service")

	query := `
		SELECT id, service, status, uptime, latency, error_rate, requests_min, last_check_at, created_at
		FROM service_health
	`
	if service != "" {
		query += fmt.Sprintf(" WHERE service = '%s'", service)
	}
	query += " ORDER BY last_check_at DESC"

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var health []ServiceHealth
	for rows.Next() {
		var h ServiceHealth
		if err := rows.Scan(&h.ID, &h.Service, &h.Status, &h.Uptime, &h.Latency, &h.ErrorRate, &h.RequestsMin, &h.LastCheckAt, &h.CreatedAt); err != nil {
			continue
		}
		health = append(health, h)
	}

	c.JSON(http.StatusOK, gin.H{"services": health})
}

// CreateDashboard - Create observability dashboard
func CreateDashboard(c *gin.Context) {
	var dashboard Dashboard
	if err := c.ShouldBindJSON(&dashboard); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dashboard.ID = uuid.New()
	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()

	panelsJSON, _ := json.Marshal(dashboard.Panels)

	_, err := db.Exec(context.Background(), `
		INSERT INTO observability_dashboards (id, name, panels, service, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, dashboard.ID, dashboard.Name, panelsJSON, dashboard.Service, dashboard.OwnerID, dashboard.CreatedAt, dashboard.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dashboard)
}

// GetDashboards - Get dashboards
func GetDashboards(c *gin.Context) {
	ownerID := c.Query("owner_id")

	query := `
		SELECT id, name, panels, service, owner_id, created_at, updated_at
		FROM observability_dashboards
	`
	if ownerID != "" {
		query += fmt.Sprintf(" WHERE owner_id = '%s'", ownerID)
	}
	query += " ORDER BY updated_at DESC"

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var dashboards []Dashboard
	for rows.Next() {
		var dashboard Dashboard
		var panels []byte
		if err := rows.Scan(&dashboard.ID, &dashboard.Name, &panels, &dashboard.Service, &dashboard.OwnerID, &dashboard.CreatedAt, &dashboard.UpdatedAt); err != nil {
			continue
		}
		json.Unmarshal(panels, &dashboard.Panels)
		dashboards = append(dashboards, dashboard)
	}

	c.JSON(http.StatusOK, gin.H{"dashboards": dashboards})
}

// Health check
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
		"status":     "ok",
		"database":   dbStatus,
		"redis":      redisStatus,
		"timestamp":  time.Now(),
	})
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize logger
	logger = log.New(os.Stdout, "Observability: ", log.LstdFlags)
	logger.Println("Starting Observability Service...")

	// Load configuration
	config.Port = getEnv("OBSERVABILITY_PORT", "8096")
	config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
	config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")
	config.JWTSecret = getEnv("JWT_SECRET", "tigerwallet-secret-key")

	// Initialize database
	if err := initDatabase(); err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	logger.Println("Database connected successfully")

	// Initialize Redis
	if err := initRedis(); err != nil {
		logger.Fatalf("Failed to initialize Redis: %v", err)
	}
	logger.Println("Redis connected successfully")

	// Initialize Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
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

	// Health check
	router.GET("/health", HealthCheck)

	// Log routes
	router.POST("/api/v1/logs", IngestLog)
	router.GET("/api/v1/logs", QueryLogs)

	// Metric routes
	router.POST("/api/v1/metrics", RecordMetric)
	router.GET("/api/v1/metrics", QueryMetrics)

	// Trace routes
	router.POST("/api/v1/traces/spans", StartSpan)
	router.PUT("/api/v1/traces/spans/:span_id", EndSpan)
	router.GET("/api/v1/traces", QueryTraces)

	// Alert routes
	router.POST("/api/v1/alerts", CreateAlert)
	router.GET("/api/v1/alerts", GetAlerts)

	// Health routes
	router.POST("/api/v1/health", UpdateServiceHealth)
	router.GET("/api/v1/health", GetServiceHealth)

	// Dashboard routes
	router.POST("/api/v1/dashboards", CreateDashboard)
	router.GET("/api/v1/dashboards", GetDashboards)

	// Start server
	logger.Printf("Starting server on port %s", config.Port)
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Println("Server started successfully")

	// Wait for interrupt signal
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
