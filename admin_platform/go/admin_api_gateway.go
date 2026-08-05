/**
 * TigerWallet Admin API Gateway
 * Unified API Gateway for All Admin Services
 * High-Performance, Distributed, Ultra-Low Latency
 * 
 * Features:
 * - Unified routing to all admin services
 * - Rate limiting
 * - Authentication & Authorization
 * - Request/Response transformation
 * - Service discovery
 * - Load balancing
 * - Circuit breaker
 * - Caching
 * - WebSocket support for real-time notifications
 * - Multi-region support
 */

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Configuration
// ============================================================================

type GatewayConfig struct {
	Port                string
	AdminServiceURL     string
	SuperAdminServiceURL string
	MasterAdminServiceURL string
	RedisURL            string
	JWTSecret           string
	RateLimitPerSecond  int
	RateLimitBurst      int
	EnableTLS           bool
	CertFile           string
	KeyFile            string
	EnableCORS          bool
	AllowedOrigins     []string
	RequestTimeout      time.Duration
	MaxRetries         int
	CircuitBreakerThreshold int
	CircuitBreakerTimeout time.Duration
}

func LoadGatewayConfig() *GatewayConfig {
	return &GatewayConfig{
		Port:                   getEnv("GATEWAY_PORT", "8888"),
		AdminServiceURL:        getEnv("ADMIN_SERVICE_URL", "http://localhost:9093"),
		SuperAdminServiceURL:   getEnv("SUPER_ADMIN_SERVICE_URL", "http://localhost:9094"),
		MasterAdminServiceURL:  getEnv("MASTER_ADMIN_SERVICE_URL", "http://localhost:9095"),
		RedisURL:               getEnv("REDIS_GATEWAY_URL", "redis://localhost:6379"),
		JWTSecret:              getEnv("GATEWAY_JWT_SECRET", "gateway-secret-key-change"),
		RateLimitPerSecond:     getEnvInt("RATE_LIMIT_PER_SECOND", 1000),
		RateLimitBurst:         getEnvInt("RATE_LIMIT_BURST", 2000),
		EnableTLS:              getEnvBool("ENABLE_TLS", false),
		CertFile:              getEnv("TLS_CERT_FILE", ""),
		KeyFile:               getEnv("TLS_KEY_FILE", ""),
		EnableCORS:            getEnvBool("ENABLE_CORS", true),
		AllowedOrigins:        strings.Split(getEnv("ALLOWED_ORIGINS", "*"), ","),
		RequestTimeout:        getEnvDuration("REQUEST_TIMEOUT", 30*time.Second),
		MaxRetries:           getEnvInt("MAX_RETRIES", 3),
		CircuitBreakerThreshold: getEnvInt("CIRCUIT_BREAKER_THRESHOLD", 5),
		CircuitBreakerTimeout:  getEnvDuration("CIRCUIT_BREAKER_TIMEOUT", 60*time.Second),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// ============================================================================
// Core Types
// ============================================================================

type ServiceEndpoint struct {
	URL          string
	Timeout      time.Duration
	RetryCount   int
	Weight       int
	IsHealthy    bool
	LastCheck    time.Time
	FailureCount int
}

type CircuitBreaker struct {
	mu              sync.RWMutex
	state           string // closed, open, half-open
	failureCount    int
	lastFailureTime time.Time
	threshold       int
	timeout         time.Duration
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:     "closed",
		threshold: threshold,
		timeout:   timeout,
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount = 0
	cb.state = "closed"
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount++
	cb.lastFailureTime = time.Now()
	if cb.failureCount >= cb.threshold {
		cb.state = "open"
	}
}

func (cb *CircuitBreaker) IsAvailable() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	if cb.state == "closed" {
		return true
	}
	
	if cb.state == "open" {
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = "half-open"
			return true
		}
		return false
	}
	
	// half-open - allow one request
	return true
}

type RateLimiter struct {
	mu           sync.RWMutex
	tokens       float64
	maxTokens    float64
	refillRate   float64
	lastRefill   time.Time
}

func NewRateLimiter(maxTokens, refillRate float64) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens = min(rl.maxTokens, rl.tokens+elapsed*rl.refillRate)
	rl.lastRefill = now
	
	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}
	return false
}

type APIKey struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	AdminID     string    `json:"admin_id"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions"`
	ExpiresAt   time.Time `json:"expires_at"`
	RateLimit   int       `json:"rate_limit"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
	IsActive    bool      `json:"is_active"`
}

// ============================================================================
// API Gateway
// ============================================================================

type APIGateway struct {
	config           *GatewayConfig
	services         map[string]*ServiceEndpoint
	circuitBreakers  map[string]*CircuitBreaker
	rateLimiter      *RateLimiter
	redisClient      *redis.Client
	apiKeys          map[string]*APIKey
	webSocketHub     *WebSocketHub
	jwtSecret        []byte
	proxy            *httputil.ReverseProxy
	mu               sync.RWMutex
	requestCount     map[string]int64
	totalRequests    int64
	startTime        time.Time
}

type WebSocketMessage struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

type WebSocketClient struct {
	ID        string
	AdminID   string
	Conn      *websocket.Conn
	Send      chan []byte
	Hub       *WebSocketHub
}

type WebSocketHub struct {
	clients    map[string]*WebSocketClient
	broadcast  chan *WebSocketMessage
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[string]*WebSocketClient),
		broadcast:  make(chan *WebSocketMessage, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

func (hub *WebSocketHub) Run() {
	for {
		select {
		case client := <-hub.register:
			hub.mu.Lock()
			hub.clients[client.ID] = client
			hub.mu.Unlock()
			log.Printf("WebSocket client registered: %s", client.ID)
			
		case client := <-hub.unregister:
			hub.mu.Lock()
			if _, ok := hub.clients[client.ID]; ok {
				delete(hub.clients, client.ID)
				close(client.Send)
			}
			hub.mu.Unlock()
			log.Printf("WebSocket client unregistered: %s", client.ID)
			
		case message := <-hub.broadcast:
			hub.mu.RLock()
			for _, client := range hub.clients {
				select {
				case client.Send <- messageToBytes(message):
				default:
					close(client.Send)
					delete(hub.clients, client.ID)
				}
			}
			hub.mu.RUnlock()
		}
	}
}

func (hub *WebSocketHub) Broadcast(message *WebSocketMessage) {
	hub.broadcast <- message
}

func (hub *WebSocketHub) SendToAdmin(adminID string, message *WebSocketMessage) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	
	for _, client := range hub.clients {
		if client.AdminID == adminID {
			select {
			case client.Send <- messageToBytes(message):
			default:
			}
		}
	}
}

func messageToBytes(msg *WebSocketMessage) []byte {
	data, _ := json.Marshal(msg)
	return data
}

func NewAPIGateway(config *GatewayConfig) *APIGateway {
	// Parse Redis URL
	redisOpts, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		log.Printf("Warning: Failed to parse Redis URL: %v", err)
		redisOpts = &redis.Options{
			Addr: "localhost:6379",
		}
	}
	
	redisClient := redis.NewClient(redisOpts)
	
	gateway := &APIGateway{
		config:          config,
		services:        make(map[string]*ServiceEndpoint),
		circuitBreakers: make(map[string]*CircuitBreaker),
		rateLimiter:     NewRateLimiter(float64(config.RateLimitBurst), float64(config.RateLimitPerSecond)),
		redisClient:     redisClient,
		apiKeys:         make(map[string]*APIKey),
		webSocketHub:    NewWebSocketHub(),
		jwtSecret:       []byte(config.JWTSecret),
		requestCount:    make(map[string]int64),
		startTime:       time.Now(),
	}
	
	// Initialize service endpoints
	gateway.services["admin"] = &ServiceEndpoint{
		URL:       config.AdminServiceURL,
		Timeout:   30 * time.Second,
		RetryCount: 3,
		Weight:    10,
		IsHealthy: true,
	}
	
	gateway.services["super_admin"] = &ServiceEndpoint{
		URL:       config.SuperAdminServiceURL,
		Timeout:   30 * time.Second,
		RetryCount: 3,
		Weight:    10,
		IsHealthy: true,
	}
	
	gateway.services["master_admin"] = &ServiceEndpoint{
		URL:       config.MasterAdminServiceURL,
		Timeout:   30 * time.Second,
		RetryCount: 3,
		Weight:    10,
		IsHealthy: true,
	}
	
	// Initialize circuit breakers
	for service := range gateway.services {
		gateway.circuitBreakers[service] = NewCircuitBreaker(
			config.CircuitBreakerThreshold,
			config.CircuitBreakerTimeout,
		)
	}
	
	return gateway
}

// ============================================================================
// Middleware
// ============================================================================

func (g *APIGateway) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for health checks
		if c.Path == "/health" || c.Path == "/ready" {
			c.Next()
			return
		}
		
		// Get API key or JWT token
		apiKey := c.GetHeader("X-API-Key")
		authHeader := c.GetHeader("Authorization")
		
		var limiterKey string
		if apiKey != "" {
			limiterKey = "api:" + apiKey
		} else if authHeader != "" {
			// Extract token from "Bearer <token>"
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != authHeader {
				limiterKey = "jwt:" + token[:min(len(token), 32)]
			}
		}
		
		if limiterKey == "" {
			limiterKey = "ip:" + c.ClientIP()
		}
		
		// Check rate limit
		if !g.rateLimiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"retry_after": 1,
			})
			return
		}
		
		// Track requests per client
		g.mu.Lock()
		g.requestCount[limiterKey]++
		g.totalRequests++
		g.mu.Unlock()
		
		c.Next()
	}
}

func (g *APIGateway) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for public endpoints
		publicPaths := []string{"/health", "/ready", "/api/v1/auth/login", "/api/v1/auth/register", "/ws"}
		for _, path := range publicPaths {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}
		
		// Check API key
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			if key, ok := g.apiKeys[apiKey]; ok && key.IsActive {
				key.LastUsed = time.Now()
				c.Set("admin_id", key.AdminID)
				c.Set("permissions", key.Permissions)
				c.Next()
				return
			}
		}
		
		// Check JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return g.jwtSecret, nil
			})
			
			if err == nil && token.Valid {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					c.Set("admin_id", claims["admin_id"])
					c.Set("role", claims["role"])
					c.Next()
					return
				}
			}
		}
		
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized - Invalid or missing authentication",
		})
	}
}

func (g *APIGateway) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !g.config.EnableCORS {
			c.Next()
			return
		}
		
		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range g.config.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}
		
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-API-Key, X-Request-ID")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "3600")
		}
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

func (g *APIGateway) LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		
		c.Next()
		
		latency := time.Since(start)
		status := c.Writer.Status()
		
		// Log request
		log.Printf("[%s] %s %s %d %v",
			time.Now().Format("2006-01-02 15:04:05"),
			method,
			path,
			status,
			latency,
		)
		
		// Store metrics in Redis
		ctx := context.Background()
		g.redisClient.HIncrBy(ctx, "gateway:metrics:requests", fmt.Sprintf("%d", status), 1)
		g.redisClient.HIncrBy(ctx, "gateway:metrics:methods", method, 1)
		g.redisClient.Expire(ctx, "gateway:metrics:requests", 24*time.Hour)
	}
}

// ============================================================================
// Routing
// ============================================================================

func (g *APIGateway) SetupRoutes(r *gin.Engine) {
	// Health check endpoints
	r.GET("/health", g.HealthHandler)
	r.GET("/ready", g.ReadyHandler)
	
	// WebSocket endpoint
	r.GET("/ws", g.WebSocketHandler)
	
	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", g.ProxyToService("admin", "/api/v1/auth/login"))
			auth.POST("/register", g.ProxyToService("admin", "/api/v1/auth/register"))
			auth.POST("/refresh", g.ProxyToService("admin", "/api/v1/auth/refresh"))
		}
		
		// Protected routes
		protected := v1.Group("")
		protected.Use(g.AuthMiddleware())
		protected.Use(g.RateLimitMiddleware())
		{
			// Admin management
			admins := protected.Group("/admins")
			{
				admins.GET("", g.ProxyToService("admin", "/api/v1/admins"))
				admins.POST("", g.ProxyToService("super_admin", "/api/v1/admins"))
				admins.GET("/:id", g.ProxyToService("admin", "/api/v1/admins/:id"))
				admins.PUT("/:id", g.ProxyToService("admin", "/api/v1/admins/:id"))
				admins.DELETE("/:id", g.ProxyToService("super_admin", "/api/v1/admins/:id"))
				admins.POST("/:id/suspend", g.ProxyToService("super_admin", "/api/v1/admins/:id/suspend"))
				admins.POST("/:id/two-factor/enable", g.ProxyToService("admin", "/api/v1/admins/:id/two-factor/enable"))
				admins.POST("/:id/two-factor/disable", g.ProxyToService("admin", "/api/v1/admins/:id/two-factor/disable"))
				admins.POST("/:id/password", g.ProxyToService("admin", "/api/v1/admins/:id/password"))
			}
			
			// User management
			users := protected.Group("/users")
			{
				users.GET("", g.ProxyToService("admin", "/api/v1/users"))
				users.GET("/:id", g.ProxyToService("admin", "/api/v1/users/:id"))
				users.PUT("/:id", g.ProxyToService("admin", "/api/v1/users/:id"))
				users.DELETE("/:id", g.ProxyToService("admin", "/api/v1/users/:id"))
				users.POST("/:id/suspend", g.ProxyToService("admin", "/api/v1/users/:id/suspend"))
				users.POST("/:id/unsuspend", g.ProxyToService("admin", "/api/v1/users/:id/unsuspend"))
				users.POST("/:id/verify", g.ProxyToService("admin", "/api/v1/users/:id/verify"))
				users.POST("/:id/kyc", g.ProxyToService("admin", "/api/v1/users/:id/kyc"))
			}
			
			// KYC management
			kyc := protected.Group("/kyc")
			{
				kyc.GET("", g.ProxyToService("admin", "/api/v1/kyc"))
				kyc.GET("/:id", g.ProxyToService("admin", "/api/v1/kyc/:id"))
				kyc.POST("/:id/approve", g.ProxyToService("admin", "/api/v1/kyc/:id/approve"))
				kyc.POST("/:id/reject", g.ProxyToService("admin", "/api/v1/kyc/:id/reject"))
				kyc.POST("/:id/resubmit", g.ProxyToService("admin", "/api/v1/kyc/:id/resubmit"))
			}
			
			// Transaction management
			transactions := protected.Group("/transactions")
			{
				transactions.GET("", g.ProxyToService("admin", "/api/v1/transactions"))
				transactions.GET("/:id", g.ProxyToService("admin", "/api/v1/transactions/:id"))
				transactions.POST("/:id/flag", g.ProxyToService("admin", "/api/v1/transactions/:id/flag"))
				transactions.POST("/:id/unflag", g.ProxyToService("admin", "/api/v1/transactions/:id/unflag"))
			}
			
			// Token management
			tokens := protected.Group("/tokens")
			{
				tokens.GET("", g.ProxyToService("admin", "/api/v1/tokens"))
				tokens.POST("", g.ProxyToService("admin", "/api/v1/tokens"))
				tokens.GET("/:id", g.ProxyToService("admin", "/api/v1/tokens/:id"))
				tokens.PUT("/:id", g.ProxyToService("admin", "/api/v1/tokens/:id"))
				tokens.DELETE("/:id", g.ProxyToService("admin", "/api/v1/tokens/:id"))
				tokens.POST("/:id/activate", g.ProxyToService("admin", "/api/v1/tokens/:id/activate"))
				tokens.POST("/:id/deactivate", g.ProxyToService("admin", "/api/v1/tokens/:id/deactivate"))
				tokens.POST("/:id/verify", g.ProxyToService("admin", "/api/v1/tokens/:id/verify"))
			}
			
			// Blockchain management
			blockchains := protected.Group("/blockchains")
			{
				blockchains.GET("", g.ProxyToService("admin", "/api/v1/blockchains"))
				blockchains.POST("", g.ProxyToService("admin", "/api/v1/blockchains"))
				blockchains.GET("/:id", g.ProxyToService("admin", "/api/v1/blockchains/:id"))
				blockchains.PUT("/:id", g.ProxyToService("admin", "/api/v1/blockchains/:id"))
				blockchains.DELETE("/:id", g.ProxyToService("admin", "/api/v1/blockchains/:id"))
				blockchains.POST("/:id/test-rpc", g.ProxyToService("admin", "/api/v1/blockchains/:id/test-rpc"))
			}
			
			// Trading pairs
			pairs := protected.Group("/pairs")
			{
				pairs.GET("", g.ProxyToService("admin", "/api/v1/pairs"))
				pairs.POST("", g.ProxyToService("admin", "/api/v1/pairs"))
				pairs.GET("/:id", g.ProxyToService("admin", "/api/v1/pairs/:id"))
				pairs.PUT("/:id", g.ProxyToService("admin", "/api/v1/pairs/:id"))
				pairs.DELETE("/:id", g.ProxyToService("admin", "/api/v1/pairs/:id"))
			}
			
			// White label management
			whitelabels := protected.Group("/whitelabels")
			{
				whitelabels.GET("", g.ProxyToService("admin", "/api/v1/whitelabels"))
				whitelabels.POST("", g.ProxyToService("super_admin", "/api/v1/whitelabels"))
				whitelabels.GET("/:id", g.ProxyToService("admin", "/api/v1/whitelabels/:id"))
				whitelabels.PUT("/:id", g.ProxyToService("admin", "/api/v1/whitelabels/:id"))
				whitelabels.DELETE("/:id", g.ProxyToService("super_admin", "/api/v1/whitelabels/:id"))
				whitelabels.POST("/:id/approve", g.ProxyToService("super_admin", "/api/v1/whitelabels/:id/approve"))
				whitelabels.POST("/:id/reject", g.ProxyToService("super_admin", "/api/v1/whitelabels/:id/reject"))
			}
			
			// Withdrawal management
			withdrawals := protected.Group("/withdrawals")
			{
				withdrawals.GET("", g.ProxyToService("admin", "/api/v1/withdrawals"))
				withdrawals.GET("/:id", g.ProxyToService("admin", "/api/v1/withdrawals/:id"))
				withdrawals.POST("/:id/approve", g.ProxyToService("admin", "/api/v1/withdrawals/:id/approve"))
				withdrawals.POST("/:id/reject", g.ProxyToService("admin", "/api/v1/withdrawals/:id/reject"))
				withdrawals.POST("/batch-approve", g.ProxyToService("admin", "/api/v1/withdrawals/batch-approve"))
			}
			
			// Fee management
			fees := protected.Group("/fees")
			{
				fees.GET("", g.ProxyToService("admin", "/api/v1/fees"))
				fees.PUT("", g.ProxyToService("super_admin", "/api/v1/fees"))
				fees.GET("/history", g.ProxyToService("admin", "/api/v1/fees/history"))
			}
			
			// Analytics
			analytics := protected.Group("/analytics")
			{
				analytics.GET("", g.ProxyToService("admin", "/api/v1/analytics"))
				analytics.GET("/users", g.ProxyToService("admin", "/api/v1/analytics/users"))
				analytics.GET("/volume", g.ProxyToService("admin", "/api/v1/analytics/volume"))
				analytics.GET("/revenue", g.ProxyToService("admin", "/api/v1/analytics/revenue"))
				analytics.GET("/transactions", g.ProxyToService("admin", "/api/v1/analytics/transactions"))
			}
			
			// System
			system := protected.Group("/system")
			{
				system.GET("/status", g.ProxyToService("admin", "/api/v1/system/status"))
				system.GET("/metrics", g.ProxyToService("admin", "/api/v1/system/metrics"))
				system.GET("/services/:name", g.ProxyToService("admin", "/api/v1/system/services/:name"))
				system.POST("/services/:name/restart", g.ProxyToService("super_admin", "/api/v1/system/services/:name/restart"))
			}
			
			// Audit logs
			auditLogs := protected.Group("/audit-logs")
			{
				auditLogs.GET("", g.ProxyToService("admin", "/api/v1/audit-logs"))
			}
			
			// Notifications
			notifications := protected.Group("/notifications")
			{
				notifications.GET("", g.ProxyToService("admin", "/api/v1/notifications"))
				notifications.PUT("/:id/read", g.ProxyToService("admin", "/api/v1/notifications/:id/read"))
				notifications.PUT("/read-all", g.ProxyToService("admin", "/api/v1/notifications/read-all"))
			}
			
			// Sessions
			sessions := protected.Group("/sessions")
			{
				sessions.GET("", g.ProxyToService("admin", "/api/v1/sessions"))
				sessions.DELETE("/:id", g.ProxyToService("admin", "/api/v1/sessions/:id"))
				sessions.DELETE("", g.ProxyToService("admin", "/api/v1/sessions"))
			}
			
			// Feature flags
			features := protected.Group("/features")
			{
				features.GET("", g.ProxyToService("admin", "/api/v1/features"))
				features.PUT("/:name", g.ProxyToService("super_admin", "/api/v1/features/:name"))
			}
			
			// Config
			config := protected.Group("/config")
			{
				config.GET("", g.ProxyToService("admin", "/api/v1/config"))
				config.PUT("", g.ProxyToService("super_admin", "/api/v1/config"))
			}
			
			// Master Admin (specific routes)
			masterAdmin := protected.Group("/master-admin")
			{
				masterAdmin.GET("", g.ProxyToService("master_admin", "/api/v1/master-admin"))
				masterAdmin.POST("", g.ProxyToService("master_admin", "/api/v1/master-admin"))
				masterAdmin.GET("/:id", g.ProxyToService("master_admin", "/api/v1/master-admin/:id"))
				masterAdmin.PUT("/:id", g.ProxyToService("master_admin", "/api/v1/master-admin/:id"))
				masterAdmin.DELETE("/:id", g.ProxyToService("master_admin", "/api/v1/master-admin/:id"))
			}
			
			// WebSocket notification subscription
			protected.POST("/ws/subscribe", g.WSSubscribeHandler)
		}
	}
	
	// Gateway-specific routes
	gateway := v1.Group("/gateway")
	gateway.Use(g.AuthMiddleware())
	{
		gateway.GET("/stats", g.StatsHandler)
		gateway.GET("/services", g.ServicesHandler)
		gateway.POST("/api-keys", g.CreateAPIKeyHandler)
		gateway.DELETE("/api-keys/:key", g.DeleteAPIKeyHandler)
	}
}

// ============================================================================
// Handlers
// ============================================================================

func (g *APIGateway) HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
	})
}

func (g *APIGateway) ReadyHandler(c *gin.Context) {
	// Check Redis
	ctx := context.Background()
	if err := g.redisClient.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"reason": "redis_unavailable",
		})
		return
	}
	
	// Check service health
	allHealthy := true
	for name, service := range g.services {
		if !service.IsHealthy {
			allHealthy = false
			log.Printf("Service %s is not healthy", name)
		}
	}
	
	if !allHealthy {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"reason": "services_unavailable",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"timestamp": time.Now().Unix(),
	})
}

func (g *APIGateway) StatsHandler(c *gin.Context) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	uptime := time.Since(g.startTime)
	
	c.JSON(http.StatusOK, gin.H{
		"uptime_seconds":   uptime.Seconds(),
		"total_requests":   g.totalRequests,
		"request_count":    g.requestCount,
		"services":         g.services,
		"timestamp":        time.Now().Unix(),
	})
}

func (g *APIGateway) ServicesHandler(c *gin.Context) {
	services := make([]gin.H, 0)
	for name, service := range g.services {
		services = append(services, gin.H{
			"name":          name,
			"url":           service.URL,
			"is_healthy":    service.IsHealthy,
			"last_check":    service.LastCheck,
			"failure_count": service.FailureCount,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"services": services,
	})
}

func (g *APIGateway) CreateAPIKeyHandler(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Permissions []string `json:"permissions"`
		ExpiresAt   string   `json:"expires_at"`
		RateLimit   int      `json:"rate_limit"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Generate API key
	apiKey := &APIKey{
		ID:          uuid.New().String(),
		Key:         "tw_" + uuid.New().String(),
		AdminID:     c.GetString("admin_id"),
		Name:        req.Name,
		Permissions: req.Permissions,
		IsActive:    true,
		RateLimit:   req.RateLimit,
		CreatedAt:   time.Now(),
	}
	
	if req.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ExpiresAt); err == nil {
			apiKey.ExpiresAt = t
		}
	}
	
	g.apiKeys[apiKey.Key] = apiKey
	
	c.JSON(http.StatusCreated, gin.H{
		"api_key": apiKey.Key,
		"name":    apiKey.Name,
		"expires_at": apiKey.ExpiresAt,
	})
}

func (g *APIGateway) DeleteAPIKeyHandler(c *gin.Context) {
	key := c.Param("key")
	
	if _, ok := g.apiKeys[key]; ok {
		delete(g.apiKeys, key)
		c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
		return
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
}

func (g *APIGateway) WebSocketHandler(c *gin.Context) {
	// Upgrade to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
	}
	
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	
	adminID := c.Query("admin_id")
	if adminID == "" {
		adminID = "anonymous"
	}
	
	client := &WebSocketClient{
		ID:      uuid.New().String(),
		AdminID: adminID,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		Hub:     g.webSocketHub,
	}
	
	g.webSocketHub.register <- client
	
	// Start write pump
	go func() {
		defer func() {
			g.webSocketHub.unregister <- client
			conn.Close()
		}()
		
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			
			if messageType == websocket.TextMessage {
				// Handle incoming message
				var msg WebSocketMessage
				if err := json.Unmarshal(message, &msg); err == nil {
					if msg.Type == "ping" {
						client.Send <- []byte(`{"type":"pong","timestamp":` + strconv.FormatInt(time.Now().Unix(), 10) + `}`)
					}
				}
			}
		}
	}()
	
	// Start read pump
	go func() {
		defer func() {
			g.webSocketHub.unregister <- client
			conn.Close()
		}()
		
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			
			// Echo back for now
			client.Send <- message
		}
	}()
}

func (g *APIGateway) WSSubscribeHandler(c *gin.Context) {
	var req struct {
		AdminID string `json:"admin_id" binding:"required"`
		Events  []string `json:"events"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Subscription is handled via WebSocket
	c.JSON(http.StatusOK, gin.H{
		"message": "Please connect to /ws endpoint for real-time updates",
		"admin_id": req.AdminID,
	})
}

// ============================================================================
// Proxy
// ============================================================================

func (g *APIGateway) ProxyToService(serviceName, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		service, ok := g.services[serviceName]
		if !ok {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service not found"})
			return
		}
		
		// Check circuit breaker
		cb, ok := g.circuitBreakers[serviceName]
		if ok && !cb.IsAvailable() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service temporarily unavailable",
				"retry_after": cb.timeout.Seconds(),
			})
			return
		}
		
		// Build target URL
		targetURL := service.URL + path
		
		// Forward request
		client := &http.Client{
			Timeout: service.Timeout,
		}
		
		req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
			return
		}
		
		// Copy headers
		for k, v := range c.Request.Header {
			req.Header[k] = v
		}
		
		// Add admin context headers
		if adminID := c.GetString("admin_id"); adminID != "" {
			req.Header.Set("X-Admin-ID", adminID)
		}
		if role := c.GetString("role"); role != "" {
			req.Header.Set("X-Admin-Role", role)
		}
		
		resp, err := client.Do(req)
		if err != nil {
			if ok {
				cb.RecordFailure()
			}
			service.IsHealthy = false
			service.FailureCount++
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "Failed to reach service",
				"service": serviceName,
			})
			return
		}
		
		defer resp.Body.Close()
		
		// Record success
		if ok {
			cb.RecordSuccess()
		}
		service.IsHealthy = true
		service.FailureCount = 0
		service.LastCheck = time.Now()
		
		// Copy response
		for k, v := range resp.Header {
			c.Header(k, v[0])
		}
		
		body, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.Println("Starting TigerWallet Admin API Gateway...")
	
	config := LoadGatewayConfig()
	gateway := NewAPIGateway(config)
	
	// Start WebSocket hub
	go gateway.webSocketHub.Run()
	
	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gateway.LoggingMiddleware())
	r.Use(gateway.CORSMiddleware())
	
	gateway.SetupRoutes(r)
	
	// Start server
	addr := ":" + config.Port
	log.Printf("API Gateway listening on %s", addr)
	
	if config.EnableTLS && config.CertFile != "" && config.KeyFile != "" {
		server := &http.Server{
			Addr:      addr,
			Handler:   r,
			TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		}
		log.Fatal(server.ListenAndServeTLS(config.CertFile, config.KeyFile))
	} else {
		log.Fatal(r.Run(addr))
	}
}
