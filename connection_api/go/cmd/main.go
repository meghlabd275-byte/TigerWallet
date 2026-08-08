// Connection API Service - Go Implementation
// High-performance, distributed connection management for TigerWallet ecosystem

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Configuration
type Config struct {
	Port             string
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	HeartbeatTimeout int // seconds
	MaxConnections   int
}

// ============ DATA MODELS ============

type WhiteLevelProduct string

const (
	ProductMasterWallet   WhiteLevelProduct = "master_wallet"
	ProductUserWallet    WhiteLevelProduct = "user_wallet"
	ProductBots          WhiteLevelProduct = "bots"
	ProductBotsClients   WhiteLevelProduct = "bots_clients"
	ProductProjectParty  WhiteLevelProduct = "project_party"
)

type ConnectionStatus string

const (
	StatusConnected    ConnectionStatus = "connected"
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusError        ConnectionStatus = "error"
	StatusTimeout      ConnectionStatus = "timeout"
)

// Connection state
type ConnectionState struct {
	ID              uuid.UUID         `json:"id"`
	ClientID        uuid.UUID         `json:"client_id"`
	Product         WhiteLevelProduct `json:"product"`
	ConnectionKey   string            `json:"connection_key"`
	SessionToken    string            `json:"session_token"`
	Status          ConnectionStatus  `json:"status"`
	IPAddress       string            `json:"ip_address"`
	Region          string            `json:"region"`
	Latency         int               `json:"latency"` // ms
	LastHeartbeat   time.Time         `json:"last_heartbeat"`
	ConnectedAt     time.Time         `json:"connected_at"`
	ReconnectCount int               `json:"reconnect_count"`
	Metadata        map[string]string `json:"metadata"`
}

// Connection request/response
type ConnectRequest struct {
	Product     WhiteLevelProduct `json:"product" binding:"required"`
	APIKey      string            `json:"api_key" binding:"required"`
	ClientInfo  map[string]string `json:"client_info"`
	IPAddress   string            `json:"ip_address"`
	Region      string            `json:"region"`
}

type ConnectResponse struct {
	ConnectionKey string            `json:"connection_key"`
	SessionToken string            `json:"session_token"`
	ExpiresAt    time.Time        `json:"expires_at"`
	Config      map[string]string `json:"config"`
}

type HeartbeatRequest struct {
	ConnectionKey string            `json:"connection_key" binding:"required"`
	Latency      int               `json:"latency"`
	Status       ConnectionStatus  `json:"status"`
	Metrics      map[string]int    `json:"metrics"`
}

type DisconnectRequest struct {
	ConnectionKey string `json:"connection_key" binding:"required"`
	Reason       string `json:"reason"`
}

// Global state
var (
	db              *pgxpool.Pool
	redis           *redis.Client
	config          Config
	logger          *log.Logger
	jwtSecret       []byte
	
	// Connection management
	connections     sync.Map // map[string]*ConnectionState
	activeCount    atomic.Int64
	totalConnects  atomic.Int64
	totalDisconnects atomic.Int64
	
	// Rate limiting
	rateLimiters   sync.Map // map[string]*rateLimiter
	
	// Metrics
	metrics struct {
		connectsPerSec  atomic.Int64
		disconnectsPerSec atomic.Int64
		avgLatency      atomic.Int64
		errorsPerSec    atomic.Int64
	}
)

type rateLimiter struct {
	requests int
	window   time.Duration
	lastReset time.Time
	mu       sync.Mutex
}

func newRateLimiter(requests int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		requests:   requests,
		window:     window,
		lastReset:  time.Now(),
	}
}

func (r *rateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if time.Since(r.lastReset) > r.window {
		r.requests = 0
		r.lastReset = time.Now()
	}
	
	if r.requests >= 0 {
		r.requests++
		return true
	}
	return false
}

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
		CREATE TABLE IF NOT EXISTS connection_api_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			client_id UUID NOT NULL,
			product VARCHAR(50) NOT NULL,
			api_key VARCHAR(255) UNIQUE NOT NULL,
			api_key_hash VARCHAR(255) NOT NULL,
			rate_limit INTEGER DEFAULT 1000,
			is_active BOOLEAN DEFAULT true,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			last_used_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS connection_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			client_id UUID NOT NULL,
			product VARCHAR(50) NOT NULL,
			connection_key VARCHAR(255) UNIQUE NOT NULL,
			session_token VARCHAR(512) UNIQUE NOT NULL,
			status VARCHAR(50) DEFAULT 'connected',
			ip_address VARCHAR(45),
			region VARCHAR(50),
			latency INTEGER DEFAULT 0,
			connected_at TIMESTAMP DEFAULT NOW(),
			disconnected_at TIMESTAMP,
			expires_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS connection_metrics (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			connection_key VARCHAR(255) NOT NULL,
			metric_type VARCHAR(50) NOT NULL,
			value INTEGER,
			timestamp TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_api_keys ON connection_api_keys(api_key);
		CREATE INDEX IF NOT EXISTS idx_sessions_key ON connection_sessions(connection_key);
		CREATE INDEX IF NOT EXISTS idx_sessions_token ON connection_sessions(session_token);
		CREATE INDEX IF NOT EXISTS idx_metrics_key ON connection_metrics(connection_key);
	`)

	return err
}

func initRedis() error {
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return err
	}
	redis = redis.NewClient(opt)
	return redis.Ping(context.Background()).Err()
}

// ============ CORE FUNCTIONS ============

func generateConnectionKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "conn_" + base64.URLEncoding.EncodeToString(b)[:32]
}

func generateSessionToken() string {
	b := make([]byte, 64)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

func validateAPIKey(apiKey string, product WhiteLevelProduct) (*uuid.UUID, error) {
	keyHash := hashAPIKey(apiKey)
	
	var clientID uuid.UUID
	err := db.QueryRow(context.Background(), `
		SELECT client_id FROM connection_api_keys 
		WHERE api_key_hash = $1 AND product = $2 AND is_active = true
	`, keyHash, product).Scan(&clientID)

	if err != nil {
		return nil, err
	}

	// Update last used
	db.Exec(context.Background(), "UPDATE connection_api_keys SET last_used_at = NOW() WHERE api_key_hash = $1", keyHash)
	
	return &clientID, nil
}

func createConnection(req ConnectRequest) (*ConnectResponse, error) {
	clientID, err := validateAPIKey(req.APIKey, req.Product)
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	// Check rate limit
	if !checkRateLimit(clientID.String()) {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	// Generate keys
	connKey := generateConnectionKey()
	sessionToken := generateSessionToken()
	expiresAt := time.Now().Add(24 * time.Hour)

	// Create connection state
	conn := &ConnectionState{
		ID:              uuid.New(),
		ClientID:        *clientID,
		Product:         req.Product,
		ConnectionKey:   connKey,
		SessionToken:    sessionToken,
		Status:          StatusConnected,
		IPAddress:       req.IPAddress,
		Region:          req.Region,
		LastHeartbeat:   time.Now(),
		ConnectedAt:     time.Now(),
		ReconnectCount:  0,
		Metadata:        req.ClientInfo,
	}

	// Store in memory
	connections.Store(connKey, conn)
	activeCount.Add(1)
	totalConnects.Add(1)

	// Store in Redis for distributed access
	connJSON, _ := json.Marshal(conn)
	redis.Set(context.Background(), "conn:"+connKey, connJSON, 24*time.Hour)

	// Store session in database
	ipAddr := req.IPAddress
	if ipAddr == "" {
		ipAddr = "unknown"
	}
	region := req.Region
	if region == "" {
		region = "unknown"
	}

	_, err = db.Exec(context.Background(), `
		INSERT INTO connection_sessions 
		(id, client_id, product, connection_key, session_token, status, ip_address, region, connected_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, conn.ID, *clientID, req.Product, connKey, sessionToken, "connected", ipAddr, region, conn.ConnectedAt, expiresAt)

	if err != nil {
		connections.Delete(connKey)
		activeCount.Add(-1)
		return nil, err
	}

	// Get connection config from Redis
	config := getConnectionConfig(req.Product)

	return &ConnectResponse{
		ConnectionKey: connKey,
		SessionToken: sessionToken,
		ExpiresAt:    expiresAt,
		Config:       config,
	}, nil
}

func processHeartbeat(req HeartbeatRequest) error {
	conn, ok := connections.Load(req.ConnectionKey)
	if !ok {
		// Try Redis
		connJSON, err := redis.Get(context.Background(), "conn:"+req.ConnectionKey).Result()
		if err != nil {
			return fmt.Errorf("connection not found")
		}
		
		var newConn ConnectionState
		json.Unmarshal([]byte(connJSON), &newConn)
		conn = &newConn
	}

	state := conn.(*ConnectionState)
	state.LastHeartbeat = time.Now()
	state.Latency = req.Latency
	
	if req.Status != "" {
		state.Status = req.Status
	}

	// Update in memory
	connections.Store(req.ConnectionKey, state)

	// Update in Redis
	connJSON, _ := json.Marshal(state)
	redis.Set(context.Background(), "conn:"+req.ConnectionKey, connJSON, 24*time.Hour)

	// Update metrics
	if req.Metrics != nil {
		for metricType, value := range req.Metrics {
			db.Exec(context.Background(), `
				INSERT INTO connection_metrics (connection_key, metric_type, value)
				VALUES ($1, $2, $3)
			`, req.ConnectionKey, metricType, value)
		}
	}

	// Update database
	db.Exec(context.Background(), `
		UPDATE connection_sessions SET latency = $1, status = $2 WHERE connection_key = $3
	`, req.Latency, state.Status, req.ConnectionKey)

	return nil
}

func disconnect(req DisconnectRequest) error {
	conn, ok := connections.Load(req.ConnectionKey)
	if !ok {
		return fmt.Errorf("connection not found")
	}

	state := conn.(*ConnectionState)
	state.Status = StatusDisconnected

	// Remove from memory
	connections.Delete(req.ConnectionKey)
	activeCount.Add(-1)
	totalDisconnects.Add(1)

	// Remove from Redis
	redis.Del(context.Background(), "conn:"+req.ConnectionKey)

	// Update database
	db.Exec(context.Background(), `
		UPDATE connection_sessions SET status = 'disconnected', disconnected_at = NOW() WHERE connection_key = $1
	`, req.ConnectionKey)

	return nil
}

func checkConnectionHealth() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		timeout := time.Now().Add(-time.Duration(config.HeartbeatTimeout) * time.Second)
		
		connections.Range(func(key, value interface{}) bool {
			conn := value.(*ConnectionState)
			if conn.LastHeartbeat.Before(timeout) {
				conn.Status = StatusTimeout
				disconnect(DisconnectRequest{ConnectionKey: conn.ConnectionKey, Reason: "timeout"})
			}
			return true
		})
	}
}

func getConnectionConfig(product WhiteLevelProduct) map[string]string {
	configKey := "config:" + string(product)
	configJSON, err := redis.Get(context.Background(), configKey).Result()
	if err != nil {
		// Default config
		return map[string]string{
			"heartbeat_interval": "30",
			"reconnect_timeout": "60",
			"max_reconnects": "5",
			"timeout_ms": "5000",
		}
	}

	var config map[string]string
	json.Unmarshal([]byte(configJSON), &config)
	return config
}

func checkRateLimit(clientID string) bool {
	limiter, ok := rateLimiters.Load(clientID)
	if !ok {
		limiter = newRateLimiter(1000, time.Minute)
		rateLimiters.Store(clientID, limiter)
	}
	return limiter.(*rateLimiter).Allow()
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
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
		"status":        "ok",
		"database":      dbStatus,
		"redis":         redisStatus,
		"active_conns":  activeCount.Load(),
		"timestamp":     time.Now(),
	})
}

// Connect - Establish new connection
func Connect(c *gin.Context) {
	var req ConnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ipAddr := c.ClientIP()
	if req.IPAddress != "" {
		ipAddr = req.IPAddress
	}

	resp, err := createConnection(ConnectRequest{
		Product:    req.Product,
		APIKey:     req.APIKey,
		ClientInfo: req.ClientInfo,
		IPAddress:  ipAddr,
		Region:     req.Region,
	})

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Heartbeat - Connection heartbeat
func Heartbeat(c *gin.Context) {
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate session
	sessionToken := c.GetHeader("X-Session-Token")
	if sessionToken != "" {
		conn, ok := connections.Load(req.ConnectionKey)
		if !ok || conn.(*ConnectionState).SessionToken != sessionToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
	}

	err := processHeartbeat(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "heartbeat received"})
}

// Disconnect - Close connection
func Disconnect(c *gin.Context) {
	var req DisconnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := disconnect(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "disconnected"})
}

// GetConnection - Get connection info
func GetConnection(c *gin.Context) {
	connKey := c.Param("connection_key")
	
	conn, ok := connections.Load(connKey)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	c.JSON(http.StatusOK, conn.(*ConnectionState))
}

// GetConnections - Get all connections for client
func GetConnections(c *gin.Context) {
	clientID := c.Query("client_id")
	product := c.Query("product")

	query := `
		SELECT id, client_id, product, connection_key, session_token, status, 
		       ip_address, region, latency, connected_at, expires_at
		FROM connection_sessions
		WHERE status = 'connected'
	`
	if clientID != "" {
		query += fmt.Sprintf(" AND client_id = '%s'", clientID)
	}
	if product != "" {
		query += fmt.Sprintf(" AND product = '%s'", product)
	}
	query += " ORDER BY connected_at DESC LIMIT 100"

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type SessionInfo struct {
		ID            uuid.UUID         `json:"id"`
		ClientID      uuid.UUID         `json:"client_id"`
		Product       WhiteLevelProduct `json:"product"`
		ConnectionKey string            `json:"connection_key"`
		Status        string            `json:"status"`
		IPAddress     string            `json:"ip_address"`
		Region        string            `json:"region"`
		Latency       int               `json:"latency"`
		ConnectedAt   time.Time         `json:"connected_at"`
	}

	var sessions []SessionInfo
	for rows.Next() {
		var s SessionInfo
		if err := rows.Scan(&s.ID, &s.ClientID, &s.Product, &s.ConnectionKey, &s.SessionToken, &s.Status, &s.IPAddress, &s.Region, &s.Latency, &s.ConnectedAt, &s.ExpiresAt); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}

	c.JSON(http.StatusOK, gin.H{"connections": sessions, "total": len(sessions)})
}

// GetMetrics - Get connection metrics
func GetMetrics(c *gin.Context) {
	var totalConns, totalDisconns int64
	db.QueryRow(context.Background(), "SELECT COUNT(*) FROM connection_sessions").Scan(&totalConns)
	db.QueryRow(context.Background(), "SELECT COUNT(*) FROM connection_sessions WHERE status = 'connected'").Scan(&totalDisconns)

	c.JSON(http.StatusOK, gin.H{
		"active_connections":  activeCount.Load(),
		"total_connections":    totalConnects.Load(),
		"total_disconnects":    totalDisconnects.Load(),
		"avg_latency_ms":      metrics.avgLatency.Load(),
	})
}

// ValidateConnection - Validate connection is active
func ValidateConnection(c *gin.Context) {
	var req struct {
		ConnectionKey string `json:"connection_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conn, ok := connections.Load(req.ConnectionKey)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	state := conn.(*ConnectionState)
	
	c.JSON(http.StatusOK, gin.H{
		"valid":       state.Status == StatusConnected,
		"client_id":   state.ClientID,
		"product":     state.Product,
		"last_heartbeat": state.LastHeartbeat,
		"latency":     state.Latency,
	})
}

// ============ MAIN ============

func main() {
	logger = log.New(os.Stdout, "Connection API: ", log.LstdFlags)
	logger.Println("Starting Connection API Service...")

	config.Port = getEnv("CONNECTION_PORT", "8092")
	config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
	config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")
	config.JWTSecret = getEnv("JWT_SECRET", "")
	config.HeartbeatTimeout = 60 // 60 seconds
	config.MaxConnections = 100000

	jwtSecret = []byte(config.JWTSecret)

	if err := initDatabase(); err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	logger.Println("Database connected")

	if err := initRedis(); err != nil {
		logger.Fatalf("Failed to initialize Redis: %v", err)
	}
	logger.Println("Redis connected")

	// Start health check goroutine
	go checkConnectionHealth()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Session-Token")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.GET("/health", HealthCheck)

	// Connection endpoints
	router.POST("/api/v1/connect", Connect)
	router.POST("/api/v1/heartbeat", Heartbeat)
	router.POST("/api/v1/disconnect", Disconnect)
	router.GET("/api/v1/connections/:connection_key", GetConnection)
	router.GET("/api/v1/connections", GetConnections)
	router.POST("/api/v1/validate", ValidateConnection)

	// Metrics
	router.GET("/api/v1/metrics", GetMetrics)

	logger.Printf("Starting server on port %s", config.Port)
	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Get local IP
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			logger.Printf("Server IP: %s", ipnet.IP.String())
		}
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
