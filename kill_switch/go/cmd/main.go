// Kill Switch API - Remote Control for White Level Products
// High-performance, distributed remote command system

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

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
}

type CommandType string
type CommandStatus string

const (
	CommandDisable      CommandType = "disable"
	CommandEnable       CommandType = "enable"
	CommandUpdateConfig CommandType = "update_config"
	CommandRestart      CommandType = "restart"
	CommandShutdown     CommandType = "shutdown"
	CommandClearCache  CommandType = "clear_cache"
	CommandForceSync   CommandType = "force_sync"
)

const (
	StatusPending    CommandStatus = "pending"
	StatusExecuting CommandStatus = "executing"
	StatusCompleted CommandStatus = "completed"
	StatusFailed    CommandStatus = "failed"
	StatusTimeout   CommandStatus = "timeout"
)

var db *pgxpool.Pool
var redis *redis.Client
var config Config
var logger *log.Logger

func main() {
	logger = log.New(os.Stdout, "Kill Switch: ", log.LstdFlags)
	logger.Println("Starting Kill Switch API...")

	config.Port = getEnv("KILL_SWITCH_PORT", "8098")
	config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
	config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")

	var err error
	db, err = pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	logger.Println("Database connected")

	opt, _ := redis.ParseURL(config.RedisURL)
	redis = redis.NewClient(opt)
	redis.Ping(context.Background())
	logger.Println("Redis connected")

	initDatabase()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.GET("/health", healthCheck)
	router.POST("/api/v1/commands", sendCommand)
	router.GET("/api/v1/commands/:id", getCommand)
	router.GET("/api/v1/clients", getAllClients)
	router.PUT("/api/v1/clients/:client_id/status", updateClientStatus)
	router.GET("/api/v1/clients/:client_id/status", getClientStatus)
	router.POST("/api/v1/flags", setFeatureFlag)
	router.GET("/api/v1/clients/:client_id/flags", getFeatureFlags)
	router.POST("/api/v1/heartbeat", clientHeartbeat)
	router.GET("/api/v1/clients/:client_id/commands", getClientCommands)

	logger.Printf("Starting server on port %s", config.Port)
	srv := &http.Server{Addr: ":" + config.Port, Handler: router}

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
	srv.Shutdown(ctx)
	db.Close()
	redis.Close()
	logger.Println("Server exited")
}

func initDatabase() {
	db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS remote_commands (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			client_id UUID NOT NULL,
			product VARCHAR(50) NOT NULL,
			command VARCHAR(50) NOT NULL,
			params JSONB,
			status VARCHAR(50) DEFAULT 'pending',
			result TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			executed_at TIMESTAMP,
			completed_at TIMESTAMP,
			expires_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS client_status (
			client_id UUID PRIMARY KEY,
			product VARCHAR(50) NOT NULL,
			status VARCHAR(50) DEFAULT 'active',
			is_connected BOOLEAN DEFAULT false,
			last_heartbeat TIMESTAMP,
			commands_pending INTEGER DEFAULT 0,
			metadata JSONB,
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS feature_flags (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			client_id UUID NOT NULL,
			feature VARCHAR(100) NOT NULL,
			is_enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(client_id, feature)
		);

		CREATE INDEX IF NOT EXISTS idx_commands_client ON remote_commands(client_id);
		CREATE INDEX IF NOT EXISTS idx_commands_status ON remote_commands(status);
		CREATE INDEX IF NOT EXISTS idx_client_status ON client_status(client_id);
		CREATE INDEX IF NOT EXISTS idx_feature_flags_client ON feature_flags(client_id);
	`)
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "kill-switch"})
}

func sendCommand(c *gin.Context) {
	var req struct {
		ClientID uuid.UUID              `json:"client_id" binding:"required"`
		Product  string                `json:"product" binding:"required"`
		Command  CommandType           `json:"command" binding:"required"`
		Params   map[string]interface{} `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := map[string]interface{}{
		"id":        uuid.New(),
		"client_id": req.ClientID,
		"product":   req.Product,
		"command":   req.Command,
		"params":    req.Params,
		"status":    StatusExecuting,
		"created_at": time.Now(),
	}

	cmdJSON, _ := json.Marshal(cmd)
	redis.Publish(context.Background(), fmt.Sprintf("commands:%s", req.ClientID), cmdJSON)

	db.Exec(context.Background(), `
		INSERT INTO remote_commands (id, client_id, product, command, params, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, cmd["id"], req.ClientID, req.Product, req.Command, req.Params, StatusExecuting, time.Now())

	c.JSON(http.StatusCreated, cmd)
}

func getCommand(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var result map[string]interface{}
	err = db.QueryRow(context.Background(), `
		SELECT id, client_id, product, command, params, status, result, created_at, executed_at, completed_at
		FROM remote_commands WHERE id = $1
	`, uid).Scan(&result["id"], &result["client_id"], &result["product"], &result["command"], &result["params"], &result["status"], &result["result"], &result["created_at"], &result["executed_at"], &result["completed_at"])

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func getClientCommands(c *gin.Context) {
	clientID := c.Param("client_id")
	uid, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
		return
	}

	rows, _ := db.Query(context.Background(), `
		SELECT id, command, status, created_at FROM remote_commands 
		WHERE client_id = $1 ORDER BY created_at DESC LIMIT 50
	`, uid)
	defer rows.Close()

	var commands []map[string]interface{}
	for rows.Next() {
		var cmd map[string]interface{}
		rows.Scan(&cmd["id"], &cmd["command"], &cmd["status"], &cmd["created_at"])
		commands = append(commands, cmd)
	}

	c.JSON(http.StatusOK, gin.H{"commands": commands})
}

func getAllClients(c *gin.Context) {
	rows, err := db.Query(context.Background(), `
		SELECT client_id, product, status, is_connected, last_heartbeat, updated_at
		FROM client_status ORDER BY updated_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var clients []map[string]interface{}
	for rows.Next() {
		var client map[string]interface{}
		rows.Scan(&client["client_id"], &client["product"], &client["status"], &client["is_connected"], &client["last_heartbeat"], &client["updated_at"])
		clients = append(clients, client)
	}

	c.JSON(http.StatusOK, gin.H{"clients": clients})
}

func updateClientStatus(c *gin.Context) {
	clientID := c.Param("client_id")
	uid, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
		return
	}

	var req struct {
		Status    string `json:"status" binding:"required"`
		Connected bool   `json:"connected"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Exec(context.Background(), `
		INSERT INTO client_status (client_id, status, is_connected, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (client_id) DO UPDATE SET status = $2, is_connected = $3, updated_at = NOW()
	`, uid, req.Status, req.Connected)

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

func getClientStatus(c *gin.Context) {
	clientID := c.Param("client_id")
	uid, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
		return
	}

	var status map[string]interface{}
	err = db.QueryRow(context.Background(), `
		SELECT client_id, product, status, is_connected, last_heartbeat, commands_pending, updated_at
		FROM client_status WHERE client_id = $1
	`, uid).Scan(&status["client_id"], &status["product"], &status["status"], &status["is_connected"], &status["last_heartbeat"], &status["commands_pending"], &status["updated_at"])

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}

	c.JSON(http.StatusOK, status)
}

func setFeatureFlag(c *gin.Context) {
	var req struct {
		ClientID  uuid.UUID `json:"client_id" binding:"required"`
		Feature   string    `json:"feature" binding:"required"`
		IsEnabled bool      `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Exec(context.Background(), `
		INSERT INTO feature_flags (id, client_id, feature, is_enabled, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (client_id, feature) DO UPDATE SET is_enabled = $4, updated_at = NOW()
	`, uuid.New(), req.ClientID, req.Feature, req.IsEnabled)

	flagJSON, _ := json.Marshal(req)
	redis.Publish(context.Background(), fmt.Sprintf("features:%s", req.ClientID), flagJSON)

	c.JSON(http.StatusOK, gin.H{"message": "feature flag updated"})
}

func getFeatureFlags(c *gin.Context) {
	clientID := c.Param("client_id")
	uid, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
		return
	}

	rows, _ := db.Query(context.Background(), `
		SELECT id, feature, is_enabled, created_at, updated_at
		FROM feature_flags WHERE client_id = $1
	`, uid)
	defer rows.Close()

	var flags []map[string]interface{}
	for rows.Next() {
		var flag map[string]interface{}
		rows.Scan(&flag["id"], &flag["feature"], &flag["is_enabled"], &flag["created_at"], &flag["updated_at"])
		flags = append(flags, flag)
	}

	c.JSON(http.StatusOK, gin.H{"flags": flags})
}

func clientHeartbeat(c *gin.Context) {
	var req struct {
		ClientID     uuid.UUID `json:"client_id" binding:"required"`
		IsConnected  bool      `json:"is_connected"`
		PendingCount int       `json:"pending_count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var product string
	db.QueryRow(context.Background(), "SELECT COALESCE(product, '') FROM client_status WHERE client_id = $1", req.ClientID).Scan(&product)

	db.Exec(context.Background(), `
		INSERT INTO client_status (client_id, product, is_connected, last_heartbeat, commands_pending, updated_at)
		VALUES ($1, $2, $3, NOW(), $4, NOW())
		ON CONFLICT (client_id) DO UPDATE SET is_connected = $3, last_heartbeat = NOW(), commands_pending = $4, updated_at = NOW()
	`, req.ClientID, product, req.IsConnected, req.PendingCount)

	rows, _ := db.Query(context.Background(), `
		SELECT id, command, params FROM remote_commands 
		WHERE client_id = $1 AND status = 'pending' AND (expires_at IS NULL OR expires_at > NOW())
	`, req.ClientID)
	defer rows.Close()

	var commands []map[string]interface{}
	for rows.Next() {
		var cmd map[string]interface{}
		rows.Scan(&cmd["id"], &cmd["command"], &cmd["params"])
		commands = append(commands, cmd)
	}

	c.JSON(http.StatusOK, gin.H{"message": "heartbeat received", "commands": commands})
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
