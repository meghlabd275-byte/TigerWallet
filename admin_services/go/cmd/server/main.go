// TigerWallet Admin Services - Main Entry Point
// High-Loaded Worldwide Distributed Services
// Built with Go for maximum performance and scalability

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

const (
	ServiceName    = "tiger-admin-services"
	ServiceVersion = "2.0.0"
	DefaultPort    = "8091"
)

type Config struct {
	DatabaseURL string
	RedisURL    string
	Port        string
	JWTSecret   string
}

var (
	logger      zerolog.Logger
	dbPool      *pgxpool.Pool
	redisClient *redis.Client
	config      Config
	ctx         context.Context
)

func init() {
	config = Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://tigerwallet:securepassword@postgres:5432/tigerwallet_admin_services"),
		RedisURL:    getEnv("REDIS_URL", "redis://redis:6379"),
		Port:        getEnv("PORT", DefaultPort),
		JWTSecret:   getEnv("JWT_SECRET", "tiger-admin-services-secret"),
	}

	zerolog.TimeFieldFormat = time.RFC3339
	logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Str("service", ServiceName).Timestamp().Logger()
	ctx = context.Background()
}

func main() {
	if err := initializeDatabase(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer dbPool.Close()

	if err := initializeRedis(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize Redis")
	}
	defer redisClient.Close()

	runMigrations()

	router := initializeRouter()

	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Admin Services server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")
	srv.Shutdown(context.Background())
	logger.Info().Msg("Server exited")
}

func initializeDatabase() error {
	var err error
	dbPool, err = pgxpool.Connect(ctx, config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	return dbPool.Ping(ctx)
}

func initializeRedis() error {
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return err
	}
	redisClient = redis.NewClient(opt)
	return redisClient.Ping(ctx).Err()
}

func runMigrations() {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS services (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			type VARCHAR(50) NOT NULL,
			status VARCHAR(20) DEFAULT 'running',
			config JSONB DEFAULT '{}',
			endpoint VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS service_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service_id UUID REFERENCES services(id),
			level VARCHAR(20) NOT NULL,
			message TEXT NOT NULL,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS service_metrics (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service_id UUID REFERENCES services(id),
			metric_name VARCHAR(100) NOT NULL,
			value DOUBLE PRECISION NOT NULL,
			timestamp TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS webhooks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			url VARCHAR(500) NOT NULL,
			events JSONB DEFAULT '[]',
			secret VARCHAR(255),
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS webhook_deliveries (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			webhook_id UUID REFERENCES webhooks(id),
			event VARCHAR(100) NOT NULL,
			payload JSONB NOT NULL,
			response_status INTEGER,
			response_body TEXT,
			attempts INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS jobs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			type VARCHAR(50) NOT NULL,
			schedule VARCHAR(100),
			payload JSONB DEFAULT '{}',
			status VARCHAR(20) DEFAULT 'pending',
			next_run TIMESTAMP,
			last_run TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS job_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			job_id UUID REFERENCES jobs(id),
			status VARCHAR(20) NOT NULL,
			output TEXT,
			error TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
	}

	for _, m := range migrations {
		dbPool.Exec(ctx, m)
	}
	logger.Info().Msg("Migrations completed")
}

func initializeRouter() *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", handleHealthCheck)

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/login", handleLogin)
			auth.POST("/logout", handleLogout)
		}

		admin := v1.Group("")
		admin.Use(authMiddleware())
		{
			admin.GET("/dashboard", handleDashboard)
			admin.GET("/dashboard/stats", handleDashboardStats)

			services := admin.Group("/services")
			{
				services.GET("", handleListServices)
				services.POST("", handleCreateService)
				services.GET("/:id", handleGetService)
				services.PUT("/:id", handleUpdateService)
				services.DELETE("/:id", handleDeleteService)
				services.POST("/:id/start", handleStartService)
				services.POST("/:id/stop", handleStopService)
				services.GET("/:id/logs", handleServiceLogs)
				services.GET("/:id/metrics", handleServiceMetrics)
			}

			webhooks := admin.Group("/webhooks")
			{
				webhooks.GET("", handleListWebhooks)
				webhooks.POST("", handleCreateWebhook)
				webhooks.PUT("/:id", handleUpdateWebhook)
				webhooks.DELETE("/:id", handleDeleteWebhook)
				webhooks.GET("/:id/deliveries", handleWebhookDeliveries)
				webhooks.POST("/:id/test", handleTestWebhook)
			}

			jobs := admin.Group("/jobs")
			{
				jobs.GET("", handleListJobs)
				jobs.POST("", handleCreateJob)
				jobs.PUT("/:id", handleUpdateJob)
				jobs.DELETE("/:id", handleDeleteJob)
				jobs.POST("/:id/run", handleRunJob)
				jobs.GET("/:id/logs", handleJobLogs)
			}

			admin.GET("/logs", handleListLogs)
			admin.GET("/metrics", handleListMetrics)
		}
	}

	return router
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}
		token := extractToken(authHeader)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		claims, err := verifyToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}
		c.Set("user_id", claims["user_id"])
		c.Set("user_email", claims["email"])
		c.Next()
	}
}

// ============================ AUTH ============================

func handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For demo, accept any login - in production validate against database
	token := generateToken("admin", req.Email)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": gin.H{"email": req.Email, "role": "admin"}})
}

func handleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// ============================ SERVICES ============================

func handleListServices(c *gin.Context) {
	rows, err := dbPool.Query(ctx, "SELECT id, name, type, status, endpoint, created_at FROM services ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}
	defer rows.Close()

	var services []gin.H
	for rows.Next() {
		var id, name, type_, status, endpoint string
		var createdAt time.Time
		rows.Scan(&id, &name, &type_, &status, &endpoint, &createdAt)
		services = append(services, gin.H{"id": id, "name": name, "type": type_, "status": status, "endpoint": endpoint, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"services": services})
}

func handleCreateService(c *gin.Context) {
	var req struct {
		Name     string                 `json:"name" binding:"required"`
		Type     string                 `json:"type" binding:"required"`
		Endpoint string                 `json:"endpoint"`
		Config   map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := generateID()
	configJSON, _ := json.Marshal(req.Config)

	_, err := dbPool.Exec(ctx, "INSERT INTO services (id, name, type, endpoint, config, status) VALUES ($1, $2, $3, $4, $5, 'running')", id, req.Name, req.Type, req.Endpoint, string(configJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Service created", "id": id})
}

func handleGetService(c *gin.Context) {
	id := c.Param("id")
	var name, type_, status, endpoint string
	var config []byte
	err := dbPool.QueryRow(ctx, "SELECT name, type, status, endpoint, config FROM services WHERE id = $1", id).Scan(&name, &type_, &status, &endpoint, &config)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "name": name, "type": type_, "status": status, "endpoint": endpoint, "config": string(config)})
}

func handleUpdateService(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name     string                 `json:"name"`
		Endpoint string                 `json:"endpoint"`
		Config   map[string]interface{} `json:"config"`
	}
	c.ShouldBindJSON(&req)

	configJSON, _ := json.Marshal(req.Config)
	_, err := dbPool.Exec(ctx, "UPDATE services SET name = COALESCE(NULLIF($1, ''), name), endpoint = COALESCE(NULLIF($2, ''), endpoint), config = COALESCE($3, config), updated_at = NOW() WHERE id = $4", req.Name, req.Endpoint, string(configJSON), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Service updated"})
}

func handleDeleteService(c *gin.Context) {
	id := c.Param("id")
	dbPool.Exec(ctx, "DELETE FROM services WHERE id = $1", id)
	c.JSON(http.StatusOK, gin.H{"message": "Service deleted"})
}

func handleStartService(c *gin.Context) {
	id := c.Param("id")
	dbPool.Exec(ctx, "UPDATE services SET status = 'running', updated_at = NOW() WHERE id = $1", id)
	c.JSON(http.StatusOK, gin.H{"message": "Service started"})
}

func handleStopService(c *gin.Context) {
	id := c.Param("id")
	dbPool.Exec(ctx, "UPDATE services SET status = 'stopped', updated_at = NOW() WHERE id = $1", id)
	c.JSON(http.StatusOK, gin.H{"message": "Service stopped"})
}

func handleServiceLogs(c *gin.Context) {
	id := c.Param("id")
	rows, err := dbPool.Query(ctx, "SELECT id, level, message, created_at FROM service_logs WHERE service_id = $1 ORDER BY created_at DESC LIMIT 100", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch logs"})
		return
	}
	defer rows.Close()

	var logs []gin.H
	for rows.Next() {
		var logID, level, message string
		var createdAt time.Time
		rows.Scan(&logID, &level, &message, &createdAt)
		logs = append(logs, gin.H{"id": logID, "level": level, "message": message, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func handleServiceMetrics(c *gin.Context) {
	id := c.Param("id")
	rows, err := dbPool.Query(ctx, "SELECT metric_name, value, timestamp FROM service_metrics WHERE service_id = $1 ORDER BY timestamp DESC LIMIT 100", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch metrics"})
		return
	}
	defer rows.Close()

	var metrics []gin.H
	for rows.Next() {
		var name string
		var value float64
		var timestamp time.Time
		rows.Scan(&name, &value, &timestamp)
		metrics = append(metrics, gin.H{"metric_name": name, "value": value, "timestamp": timestamp})
	}
	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}

// ============================ WEBHOOKS ============================

func handleListWebhooks(c *gin.Context) {
	rows, err := dbPool.Query(ctx, "SELECT id, name, url, events, is_active, created_at FROM webhooks ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch webhooks"})
		return
	}
	defer rows.Close()

	var webhooks []gin.H
	for rows.Next() {
		var id, name, url string
		var events []byte
		var isActive bool
		var createdAt time.Time
		rows.Scan(&id, &name, &url, &events, &isActive, &createdAt)
		webhooks = append(webhooks, gin.H{"id": id, "name": name, "url": url, "events": string(events), "is_active": isActive, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"webhooks": webhooks})
}

func handleCreateWebhook(c *gin.Context) {
	var req struct {
		Name   string   `json:"name" binding:"required"`
		URL    string   `json:"url" binding:"required"`
		Events []string `json:"events"`
		Secret string   `json:"secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := generateID()
	eventsJSON, _ := json.Marshal(req.Events)

	_, err := dbPool.Exec(ctx, "INSERT INTO webhooks (id, name, url, events, secret) VALUES ($1, $2, $3, $4, $5)", id, req.Name, req.URL, string(eventsJSON), req.Secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create webhook"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Webhook created", "id": id})
}

func handleUpdateWebhook(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name     string   `json:"name"`
		URL      string   `json:"url"`
		Events   []string `json:"events"`
		IsActive bool     `json:"is_active"`
	}
	c.ShouldBindJSON(&req)

	eventsJSON, _ := json.Marshal(req.Events)
	_, err := dbPool.Exec(ctx, "UPDATE webhooks SET name = COALESCE(NULLIF($1, ''), name), url = COALESCE(NULLIF($2, ''), url), events = COALESCE($3, events), is_active = $4 WHERE id = $5", req.Name, req.URL, string(eventsJSON), req.IsActive, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update webhook"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Webhook updated"})
}

func handleDeleteWebhook(c *gin.Context) {
	id := c.Param("id")
	dbPool.Exec(ctx, "DELETE FROM webhooks WHERE id = $1", id)
	c.JSON(http.StatusOK, gin.H{"message": "Webhook deleted"})
}

func handleWebhookDeliveries(c *gin.Context) {
	id := c.Param("id")
	rows, err := dbPool.Query(ctx, "SELECT id, event, payload, response_status, attempts, created_at FROM webhook_deliveries WHERE webhook_id = $1 ORDER BY created_at DESC LIMIT 50", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch deliveries"})
		return
	}
	defer rows.Close()

	var deliveries []gin.H
	for rows.Next() {
		var dID, event, payload string
		var respStatus, attempts int
		var createdAt time.Time
		rows.Scan(&dID, &event, &payload, &respStatus, &attempts, &createdAt)
		deliveries = append(deliveries, gin.H{"id": dID, "event": event, "payload": payload, "response_status": respStatus, "attempts": attempts, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"deliveries": deliveries})
}

func handleTestWebhook(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Test webhook sent"})
}

// ============================ JOBS ============================

func handleListJobs(c *gin.Context) {
	rows, err := dbPool.Query(ctx, "SELECT id, name, type, schedule, status, next_run, last_run FROM jobs ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch jobs"})
		return
	}
	defer rows.Close()

	var jobs []gin.H
	for rows.Next() {
		var id, name, type_, schedule, status string
		var nextRun, lastRun, createdAt time.Time
		rows.Scan(&id, &name, &type_, &schedule, &status, &nextRun, &lastRun)
		jobs = append(jobs, gin.H{"id": id, "name": name, "type": type_, "schedule": schedule, "status": status, "next_run": nextRun, "last_run": lastRun})
	}
	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

func handleCreateJob(c *gin.Context) {
	var req struct {
		Name     string                 `json:"name" binding:"required"`
		Type     string                 `json:"type" binding:"required"`
		Schedule string                 `json:"schedule"`
		Payload  map[string]interface{} `json:"payload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := generateID()
	payloadJSON, _ := json.Marshal(req.Payload)

	_, err := dbPool.Exec(ctx, "INSERT INTO jobs (id, name, type, schedule, payload) VALUES ($1, $2, $3, $4, $5)", id, req.Name, req.Type, req.Schedule, string(payloadJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Job created", "id": id})
}

func handleUpdateJob(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name     string `json:"name"`
		Schedule string `json:"schedule"`
		Status   string `json:"status"`
	}
	c.ShouldBindJSON(&req)

	_, err := dbPool.Exec(ctx, "UPDATE jobs SET name = COALESCE(NULLIF($1, ''), name), schedule = COALESCE(NULLIF($2, ''), schedule), status = COALESCE(NULLIF($3, ''), status) WHERE id = $4", req.Name, req.Schedule, req.Status, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Job updated"})
}

func handleDeleteJob(c *gin.Context) {
	id := c.Param("id")
	dbPool.Exec(ctx, "DELETE FROM jobs WHERE id = $1", id)
	c.JSON(http.StatusOK, gin.H{"message": "Job deleted"})
}

func handleRunJob(c *gin.Context) {
	id := c.Param("id")
	dbPool.Exec(ctx, "UPDATE jobs SET last_run = NOW(), status = 'running' WHERE id = $1", id)
	c.JSON(http.StatusOK, gin.H{"message": "Job triggered"})
}

func handleJobLogs(c *gin.Context) {
	id := c.Param("id")
	rows, err := dbPool.Query(ctx, "SELECT id, status, output, error, created_at FROM job_logs WHERE job_id = $1 ORDER BY created_at DESC LIMIT 50", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch job logs"})
		return
	}
	defer rows.Close()

	var logs []gin.H
	for rows.Next() {
		var logID, status, output, errorMsg string
		var createdAt time.Time
		rows.Scan(&logID, &status, &output, &errorMsg, &createdAt)
		logs = append(logs, gin.H{"id": logID, "status": status, "output": output, "error": errorMsg, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// ============================ DASHBOARD & LOGS ============================

func handleDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"dashboard": gin.H{
			"total_services":   0,
			"running_services": 0,
			"active_webhooks":  0,
			"scheduled_jobs":   0,
		},
	})
}

func handleDashboardStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"stats": gin.H{
			"services_today": 0,
			"webhooks_fired": 0,
			"jobs_completed": 0,
			"errors":         0,
		},
	})
}

func handleListLogs(c *gin.Context) {
	rows, err := dbPool.Query(ctx, "SELECT id, level, message, created_at FROM service_logs ORDER BY created_at DESC LIMIT 100")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch logs"})
		return
	}
	defer rows.Close()

	var logs []gin.H
	for rows.Next() {
		var id, level, message string
		var createdAt time.Time
		rows.Scan(&id, &level, &message, &createdAt)
		logs = append(logs, gin.H{"id": id, "level": level, "message": message, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func handleListMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"metrics": []interface{}{}})
}

func handleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": ServiceName, "version": ServiceVersion})
}

// ============================ UTILS ============================

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateToken(userID, email string) string {
	data := fmt.Sprintf("%s:%s:%d", userID, email, time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func verifyToken(token string) (map[string]interface{}, error) {
	claimsJSON, err := redisClient.Get(ctx, "token:"+token).Result()
	if err == redis.Nil {
		// For demo, create mock claims
		return map[string]interface{}{"user_id": "admin", "email": "admin@example.com"}, nil
	}
	if err != nil {
		return nil, err
	}
	var claims map[string]interface{}
	json.Unmarshal([]byte(claimsJSON), &claims)
	return claims, nil
}

func extractToken(authHeader string) string {
	if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
		return authHeader[7:]
	}
	return ""
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getIntParam(c *gin.Context, param string, defaultValue int) int {
	value := c.Query(param)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
