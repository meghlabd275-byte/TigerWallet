// Data Platform Service - PostgreSQL Version
// Comprehensive data analytics and reporting platform for TigerWallet ecosystem

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

// Data Models
type AnalyticsEvent struct {
	ID          uuid.UUID              `json:"id"`
	EventType   string                 `json:"event_type"`
	UserID      *uuid.UUID            `json:"user_id"`
	SessionID   string                 `json:"session_id"`
	Properties  map[string]interface{} `json:"properties"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	Timestamp   time.Time              `json:"timestamp"`
}

type UserActivity struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	ResourceID  string    `json:"resource_id"`
	Metadata    string    `json:"metadata"`
	IPAddress   string    `json:"ip_address"`
	Timestamp   time.Time `json:"timestamp"`
}

type Report struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	ReportType  string    `json:"report_type"` // daily, weekly, monthly, custom
	DateFrom    time.Time `json:"date_from"`
	DateTo      time.Time `json:"date_to"`
	Status      string    `json:"status"` // pending, processing, completed, failed
	ResultURL   string    `json:"result_url"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

type Dashboard struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Widgets     string    `json:"widgets"` // JSON array of widgets
	OwnerID     uuid.UUID `json:"owner_id"`
	IsPublic    bool      `json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Metric struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	MetricType  string    `json:"metric_type"` // counter, gauge, histogram
	Unit        string    `json:"unit"`
	Value       float64   `json:"value"`
	Tags        string    `json:"tags"` // JSON object
	Timestamp   time.Time `json:"timestamp"`
}

type DataSource struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	SourceType   string    `json:"source_type"` // database, api, file
	Connection   string    `json:"connection"` // connection string or API endpoint
	Query        string    `json:"query"` // SQL query or API parameters
	LastSyncAt   *time.Time `json:"last_sync_at"`
	Status       string    `json:"status"` // active, inactive, error
	CreatedAt    time.Time `json:"created_at"`
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
		CREATE TABLE IF NOT EXISTS analytics_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			event_type VARCHAR(255) NOT NULL,
			user_id UUID,
			session_id VARCHAR(255),
			properties JSONB,
			ip_address VARCHAR(45),
			user_agent TEXT,
			timestamp TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS user_activities (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			action VARCHAR(255) NOT NULL,
			resource VARCHAR(255),
			resource_id VARCHAR(255),
			metadata JSONB,
			ip_address VARCHAR(45),
			timestamp TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS reports (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			report_type VARCHAR(50) NOT NULL,
			date_from TIMESTAMP NOT NULL,
			date_to TIMESTAMP NOT NULL,
			status VARCHAR(50) DEFAULT 'pending',
			result_url TEXT,
			created_by UUID NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			completed_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS dashboards (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			description TEXT,
			widgets JSONB,
			owner_id UUID NOT NULL,
			is_public BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS metrics (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			metric_type VARCHAR(50) NOT NULL,
			unit VARCHAR(50),
			value FLOAT NOT NULL,
			tags JSONB,
			timestamp TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS data_sources (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			source_type VARCHAR(50) NOT NULL,
			connection TEXT NOT NULL,
			query TEXT,
			last_sync_at TIMESTAMP,
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_events_type ON analytics_events(event_type);
		CREATE INDEX IF NOT EXISTS idx_events_user ON analytics_events(user_id);
		CREATE INDEX IF NOT EXISTS idx_events_timestamp ON analytics_events(timestamp);
		CREATE INDEX IF NOT EXISTS idx_activities_user ON user_activities(user_id);
		CREATE INDEX IF NOT EXISTS idx_activities_timestamp ON user_activities(timestamp);
		CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics(name);
		CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);
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

// TrackEvent - Track analytics event
func TrackEvent(c *gin.Context) {
	var event AnalyticsEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event.ID = uuid.New()
	event.Timestamp = time.Now()

	propertiesJSON, _ := json.Marshal(event.Properties)

	_, err := db.Exec(context.Background(), `
		INSERT INTO analytics_events (id, event_type, user_id, session_id, properties, ip_address, user_agent, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.ID, event.EventType, event.UserID, event.SessionID, propertiesJSON, event.IPAddress, event.UserAgent, event.Timestamp)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"event_id": event.ID})
}

// GetEvents - Get analytics events
func GetEvents(c *gin.Context) {
	eventType := c.Query("type")
	userID := c.Query("user_id")
	limit := c.DefaultQuery("limit", "100")

	query := `
		SELECT id, event_type, user_id, session_id, properties, ip_address, user_agent, timestamp
		FROM analytics_events
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if eventType != "" {
		query += fmt.Sprintf(" AND event_type = $%d", argNum)
		args = append(args, eventType)
		argNum++
	}
	if userID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argNum)
		args = append(args, userID)
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

	var events []AnalyticsEvent
	for rows.Next() {
		var event AnalyticsEvent
		var properties []byte
		if err := rows.Scan(&event.ID, &event.EventType, &event.UserID, &event.SessionID, &properties, &event.IPAddress, &event.UserAgent, &event.Timestamp); err != nil {
			continue
		}
		json.Unmarshal(properties, &event.Properties)
		events = append(events, event)
	}

	c.JSON(http.StatusOK, gin.H{"events": events, "total": len(events)})
}

// RecordActivity - Record user activity
func RecordActivity(c *gin.Context) {
	var activity UserActivity
	if err := c.ShouldBindJSON(&activity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	activity.ID = uuid.New()
	activity.Timestamp = time.Now()

	metadataJSON, _ := json.Marshal(activity.Properties)

	_, err := db.Exec(context.Background(), `
		INSERT INTO user_activities (id, user_id, action, resource, resource_id, metadata, ip_address, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, activity.ID, activity.UserID, activity.Action, activity.Resource, activity.ResourceID, metadataJSON, activity.IPAddress, activity.Timestamp)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, activity)
}

// GetUserActivities - Get user activities
func GetUserActivities(c *gin.Context) {
	userID := c.Param("user_id")
	limit := c.DefaultQuery("limit", "50")

	rows, err := db.Query(context.Background(), `
		SELECT id, user_id, action, resource, resource_id, metadata, ip_address, timestamp
		FROM user_activities
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var activities []UserActivity
	for rows.Next() {
		var activity UserActivity
		var metadata []byte
		if err := rows.Scan(&activity.ID, &activity.UserID, &activity.Action, &activity.Resource, &activity.ResourceID, &metadata, &activity.IPAddress, &activity.Timestamp); err != nil {
			continue
		}
		json.Unmarshal(metadata, &activity.Properties)
		activities = append(activities, activity)
	}

	c.JSON(http.StatusOK, gin.H{"activities": activities})
}

// CreateReport - Create a new report
func CreateReport(c *gin.Context) {
	var report Report
	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report.ID = uuid.New()
	report.CreatedAt = time.Now()
	report.Status = "pending"

	_, err := db.Exec(context.Background(), `
		INSERT INTO reports (id, name, report_type, date_from, date_to, status, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, report.ID, report.Name, report.ReportType, report.DateFrom, report.DateTo, report.Status, report.CreatedBy, report.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, report)
}

// GetReports - Get reports
func GetReports(c *gin.Context) {
	rows, err := db.Query(context.Background(), `
		SELECT id, name, report_type, date_from, date_to, status, result_url, created_by, created_at, completed_at
		FROM reports
		ORDER BY created_at DESC
		LIMIT 50
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		var report Report
		if err := rows.Scan(&report.ID, &report.Name, &report.ReportType, &report.DateFrom, &report.DateTo, &report.Status, &report.ResultURL, &report.CreatedBy, &report.CreatedAt, &report.CompletedAt); err != nil {
			continue
		}
		reports = append(reports, report)
	}

	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

// CreateDashboard - Create a new dashboard
func CreateDashboard(c *gin.Context) {
	var dashboard Dashboard
	if err := c.ShouldBindJSON(&dashboard); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dashboard.ID = uuid.New()
	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()

	widgetsJSON, _ := json.Marshal(dashboard.Widgets)

	_, err := db.Exec(context.Background(), `
		INSERT INTO dashboards (id, name, description, widgets, owner_id, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, dashboard.ID, dashboard.Name, dashboard.Description, widgetsJSON, dashboard.OwnerID, dashboard.IsPublic, dashboard.CreatedAt, dashboard.UpdatedAt)

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
		SELECT id, name, description, widgets, owner_id, is_public, created_at, updated_at
		FROM dashboards
		WHERE 1=1
	`
	if ownerID != "" {
		query += fmt.Sprintf(" AND owner_id = '%s'", ownerID)
	}
	query += " ORDER BY updated_at DESC LIMIT 50"

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var dashboards []Dashboard
	for rows.Next() {
		var dashboard Dashboard
		var widgets []byte
		if err := rows.Scan(&dashboard.ID, &dashboard.Name, &dashboard.Description, &widgets, &dashboard.OwnerID, &dashboard.IsPublic, &dashboard.CreatedAt, &dashboard.UpdatedAt); err != nil {
			continue
		}
		json.Unmarshal(widgets, &dashboard.Widgets)
		dashboards = append(dashboards, dashboard)
	}

	c.JSON(http.StatusOK, gin.H{"dashboards": dashboards})
}

// RecordMetric - Record a metric
func RecordMetric(c *gin.Context) {
	var metric Metric
	if err := c.ShouldBindJSON(&metric); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metric.ID = uuid.New()
	metric.Timestamp = time.Now()

	tagsJSON, _ := json.Marshal(metric.Tags)

	_, err := db.Exec(context.Background(), `
		INSERT INTO metrics (id, name, metric_type, unit, value, tags, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, metric.ID, metric.Name, metric.MetricType, metric.Unit, metric.Value, tagsJSON, metric.Timestamp)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, metric)
}

// GetMetrics - Get metrics
func GetMetrics(c *gin.Context) {
	name := c.Query("name")
	limit := c.DefaultQuery("limit", "100")

	query := `
		SELECT id, name, metric_type, unit, value, tags, timestamp
		FROM metrics
		WHERE 1=1
	`
	if name != "" {
		query += fmt.Sprintf(" AND name = '%s'", name)
	}
	query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT %s", limit)

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var metrics []Metric
	for rows.Next() {
		var metric Metric
		var tags []byte
		if err := rows.Scan(&metric.ID, &metric.Name, &metric.MetricType, &metric.Unit, &metric.Value, &tags, &metric.Timestamp); err != nil {
			continue
		}
		json.Unmarshal(tags, &metric.Tags)
		metrics = append(metrics, metric)
	}

	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}

// GetAnalyticsSummary - Get analytics summary
func GetAnalyticsSummary(c *gin.Context) {
	var result struct {
		TotalEvents      int64   `json:"total_events"`
		TotalUsers       int64   `json:"total_users"`
		TotalActivities  int64   `json:"total_activities"`
		ActiveUsers24h   int64   `json:"active_users_24h"`
	}

	db.QueryRow(context.Background(), `SELECT COUNT(*) FROM analytics_events`).Scan(&result.TotalEvents)
	db.QueryRow(context.Background(), `SELECT COUNT(DISTINCT user_id) FROM analytics_events WHERE user_id IS NOT NULL`).Scan(&result.TotalUsers)
	db.QueryRow(context.Background(), `SELECT COUNT(*) FROM user_activities`).Scan(&result.TotalActivities)
	db.QueryRow(context.Background(), `SELECT COUNT(DISTINCT user_id) FROM analytics_events WHERE timestamp > NOW() - INTERVAL '24 hours'`).Scan(&result.ActiveUsers24h)

	c.JSON(http.StatusOK, result)
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
	logger = log.New(os.Stdout, "Data Platform: ", log.LstdFlags)
	logger.Println("Starting Data Platform Service...")

	// Load configuration
	config.Port = getEnv("DATA_PORT", "8090")
	config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
	config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")
	config.JWTSecret = getEnv("JWT_SECRET", "")

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

	// Analytics routes
	router.POST("/api/v1/analytics/events", TrackEvent)
	router.GET("/api/v1/analytics/events", GetEvents)
	router.GET("/api/v1/analytics/summary", GetAnalyticsSummary)

	// Activity routes
	router.POST("/api/v1/activities", RecordActivity)
	router.GET("/api/v1/activities/:user_id", GetUserActivities)

	// Report routes
	router.POST("/api/v1/reports", CreateReport)
	router.GET("/api/v1/reports", GetReports)

	// Dashboard routes
	router.POST("/api/v1/dashboards", CreateDashboard)
	router.GET("/api/v1/dashboards", GetDashboards)

	// Metric routes
	router.POST("/api/v1/metrics", RecordMetric)
	router.GET("/api/v1/metrics", GetMetrics)

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
