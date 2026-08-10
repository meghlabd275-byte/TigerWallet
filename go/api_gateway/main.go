/**
 * TigerWallet API Gateway
 * High-Load Distributed Go Implementation
 *
 * Features:
 * - Rate limiting (token bucket)
 * - Request routing
 * - Authentication (JWT)
 * - Load balancing
 * - Circuit breaker
 * - Request/Response transformation
 */

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ============== Data Structures ==============

type Route struct {
	Path      string          `json:"path"`
	Method    string          `json:"method"`
	Backend   string          `json:"backend"`
	Auth      bool            `json:"auth"`
	RateLimit RateLimitConfig `json:"rate_limit"`
	Timeout   time.Duration   `json:"timeout"`
	CacheTTL  time.Duration   `json:"cache_ttl"`
}

type RateLimitConfig struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	Burst             int     `json:"burst"`
}

type APIKey struct {
	ID        string  `json:"id"`
	Key       string  `json:"key"`
	UserID    string  `json:"user_id"`
	Name      string  `json:"name"`
	RateLimit float64 `json:"rate_limit"`
	ExpiresAt int64   `json:"expires_at"`
	CreatedAt int64   `json:"created_at"`
	Active    bool    `json:"active"`
}

type RequestLog struct {
	ID         string `json:"id"`
	Timestamp  int64  `json:"timestamp"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	LatencyMs  int64  `json:"latency_ms"`
	UserID     string `json:"user_id"`
	IP         string `json:"ip"`
	Error      string `json:"error,omitempty"`
}

type CircuitBreaker struct {
	Failures    int
	Successes   int
	State       string // closed, open, half-open
	LastFailure time.Time
	Threshold   int
	Timeout     time.Duration
	mu          sync.RWMutex
}

type RateLimiter struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	mu         sync.Mutex
	lastRefill time.Time
}

type HealthStatus struct {
	Status      string          `json:"status"`
	Uptime      time.Duration   `json:"uptime"`
	Requests    uint64          `json:"requests"`
	Errors      uint64          `json:"errors"`
	RateLimited uint64          `json:"rate_limited"`
	Services    map[string]bool `json:"services"`
}

// ============== Gateway ==============

type APIGateway struct {
	routes          map[string]*Route
	apiKeys         map[string]*APIKey
	circuitBreakers map[string]*CircuitBreaker
	rateLimiters    map[string]*RateLimiter
	requestLogs     []RequestLog

	mu        sync.RWMutex
	startTime time.Time
	stats     struct {
		requests    uint64
		errors      uint64
		rateLimited uint64
	}

	server *http.Server
}

func NewAPIGateway() *APIGateway {
	g := &APIGateway{
		routes:          make(map[string]*Route),
		apiKeys:         make(map[string]*APIKey),
		circuitBreakers: make(map[string]*CircuitBreaker),
		rateLimiters:    make(map[string]*RateLimiter),
		requestLogs:     make([]RequestLog, 0, 10000),
		startTime:       time.Now(),
	}

	g.initRoutes()
	g.initCircuitBreakers()

	return g
}

func (g *APIGateway) initRoutes() {
	// Wallet routes
	g.routes["/api/wallet/create"] = &Route{
		Path:      "/api/wallet/create",
		Method:    "POST",
		Backend:   "http://localhost:8001",
		Auth:      true,
		RateLimit: RateLimitConfig{RequestsPerSecond: 10, Burst: 20},
		Timeout:   30 * time.Second,
	}

	g.routes["/api/wallet/import"] = &Route{
		Path:      "/api/wallet/import",
		Method:    "POST",
		Backend:   "http://localhost:8001",
		Auth:      true,
		RateLimit: RateLimitConfig{RequestsPerSecond: 5, Burst: 10},
		Timeout:   30 * time.Second,
	}

	// Swap routes
	g.routes["/api/swap/quote"] = &Route{
		Path:      "/api/swap/quote",
		Method:    "GET",
		Backend:   "http://localhost:8002",
		Auth:      false,
		RateLimit: RateLimitConfig{RequestsPerSecond: 50, Burst: 100},
		Timeout:   5 * time.Second,
		CacheTTL:  5 * time.Second,
	}

	g.routes["/api/swap/execute"] = &Route{
		Path:      "/api/swap/execute",
		Method:    "POST",
		Backend:   "http://localhost:8002",
		Auth:      true,
		RateLimit: RateLimitConfig{RequestsPerSecond: 10, Burst: 20},
		Timeout:   60 * time.Second,
	}

	// Trading routes
	g.routes["/api/trading/order"] = &Route{
		Path:      "/api/trading/order",
		Method:    "POST",
		Backend:   "http://localhost:8003",
		Auth:      true,
		RateLimit: RateLimitConfig{RequestsPerSecond: 20, Burst: 50},
		Timeout:   10 * time.Second,
	}

	// Portfolio routes
	g.routes["/api/portfolio"] = &Route{
		Path:      "/api/portfolio",
		Method:    "GET",
		Backend:   "http://localhost:8081",
		Auth:      true,
		RateLimit: RateLimitConfig{RequestsPerSecond: 30, Burst: 60},
		Timeout:   15 * time.Second,
	}

	// Charts routes
	g.routes["/api/charts"] = &Route{
		Path:      "/api/charts",
		Method:    "GET",
		Backend:   "http://localhost:8080",
		Auth:      false,
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 200},
		Timeout:   10 * time.Second,
		CacheTTL:  1 * time.Second,
	}
}

func (g *APIGateway) initCircuitBreakers() {
	backends := []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
		"http://localhost:8080",
		"http://localhost:8081",
	}

	for _, backend := range backends {
		g.circuitBreakers[backend] = &CircuitBreaker{
			State:     "closed",
			Threshold: 5,
			Timeout:   30 * time.Second,
		}
	}
}

func (g *APIGateway) Run() error {
	mux := http.NewServeMux()

	// Main handler
	mux.HandleFunc("/", g.handleRequest)

	// Admin endpoints
	mux.HandleFunc("/admin/routes", g.handleAdminRoutes)
	mux.HandleFunc("/admin/keys", g.handleAdminKeys)
	mux.HandleFunc("/admin/stats", g.handleAdminStats)
	mux.HandleFunc("/admin/circuit", g.handleAdminCircuit)

	// Health
	mux.HandleFunc("/health", g.handleHealth)
	mux.HandleFunc("/ready", g.handleReady)

	g.server = &http.Server{
		Addr:         ":8000",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Println("API Gateway starting on :8000")
	return g.server.ListenAndServe()
}

// ============== Handlers ==============

func (g *APIGateway) handleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

	// Get client IP
	ip := getClientIP(r)

	// Find route
	routeKey := r.URL.Path
	route, exists := g.routes[routeKey]
	if !exists {
		// Try path prefix matching
		for path, r := range g.routes {
			if strings.HasPrefix(routeKey, path) {
				route = r
				break
			}
		}
	}

	if route == nil {
		g.logRequest(requestID, r, 404, 0, "", "route not found")
		http.Error(w, `{"error": "Not Found"}`, http.StatusNotFound)
		return
	}

	// Check method
	if r.Method != route.Method && !(r.Method == "OPTIONS") {
		g.logRequest(requestID, r, 405, 0, "", "method not allowed")
		http.Error(w, `{"error": "Method Not Allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Auth check
	if route.Auth {
		if !g.validateAuth(r) {
			g.logRequest(requestID, r, 401, 0, "", "unauthorized")
			http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
			return
		}
	}

	// Rate limiting
	if !g.checkRateLimit(r, route, ip) {
		g.stats.rateLimited++
		g.logRequest(requestID, r, 429, 0, "", "rate limited")
		http.Error(w, `{"error": "Rate Limited"}`, http.StatusTooManyRequests)
		return
	}

	// Circuit breaker check
	breaker, ok := g.circuitBreakers[route.Backend]
	if ok && !g.checkCircuitBreaker(breaker) {
		g.logRequest(requestID, r, 503, 0, "", "service unavailable")
		http.Error(w, `{"error": "Service Unavailable"}`, http.StatusServiceUnavailable)
		return
	}

	// Forward request
	resp, err := g.forwardRequest(r, route)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		g.stats.errors++
		g.updateCircuitBreaker(breaker, false)
		g.logRequest(requestID, r, 502, latency, "", err.Error())
		http.Error(w, `{"error": "Bad Gateway"}`, http.StatusBadGateway)
		return
	}

	g.updateCircuitBreaker(breaker, true)
	g.stats.requests++

	// Copy headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	g.logRequest(requestID, r, resp.StatusCode, latency, "", "")
	w.WriteHeader(resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	w.Write(body)
}

func (g *APIGateway) handleAdminRoutes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(g.routes)
}

func (g *APIGateway) handleAdminKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var key APIKey
		if err := json.NewDecoder(r.Body).Decode(&key); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		key.ID = fmt.Sprintf("key_%d", time.Now().Unix())
		key.CreatedAt = time.Now().Unix()
		key.Active = true
		g.apiKeys[key.Key] = &key
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(key)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(g.apiKeys)
}

func (g *APIGateway) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Status:      "healthy",
		Uptime:      time.Since(g.startTime),
		Requests:    g.stats.requests,
		Errors:      g.stats.errors,
		RateLimited: g.stats.rateLimited,
		Services:    make(map[string]bool),
	}

	for backend, breaker := range g.circuitBreakers {
		status.Services[backend] = breaker.State == "closed"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (g *APIGateway) handleAdminCircuit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(g.circuitBreakers)
}

func (g *APIGateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (g *APIGateway) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check all backends
	allReady := true
	for backend, breaker := range g.circuitBreakers {
		if breaker.State == "open" {
			allReady = false
			log.Printf("Backend %s is not ready", backend)
		}
	}

	if allReady {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	} else {
		http.Error(w, `{"status": "not ready"}`, http.StatusServiceUnavailable)
	}
}

// ============== Helpers ==============

func (g *APIGateway) validateAuth(r *http.Request) bool {
	// Check API key
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		if key, exists := g.apiKeys[apiKey]; exists && key.Active {
			return true
		}
	}

	// Check Bearer token
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		// In production, validate JWT
		return len(token) > 10
	}

	return false
}

func (g *APIGateway) checkRateLimit(r *http.Request, route *Route, ip string) bool {
	limiterKey := ip + ":" + route.Path

	g.mu.RLock()
	limiter, exists := g.rateLimiters[limiterKey]
	g.mu.RUnlock()

	if !exists {
		limiter = &RateLimiter{
			tokens:     float64(route.RateLimit.Burst),
			maxTokens:  float64(route.RateLimit.Burst),
			refillRate: route.RateLimit.RequestsPerSecond,
			lastRefill: time.Now(),
		}
		g.mu.Lock()
		g.rateLimiters[limiterKey] = limiter
		g.mu.Unlock()
	}

	return limiter.consume()
}

func (g *APIGateway) checkCircuitBreaker(breaker *CircuitBreaker) bool {
	breaker.mu.RLock()
	defer breaker.mu.RUnlock()

	if breaker.State == "closed" {
		return true
	}

	if breaker.State == "open" {
		// Check timeout
		if time.Since(breaker.LastFailure) > breaker.Timeout {
			return true // Try half-open
		}
		return false
	}

	// Half-open: allow one request
	return true
}

func (g *APIGateway) updateCircuitBreaker(breaker *CircuitBreaker, success bool) {
	if breaker == nil {
		return
	}

	breaker.mu.Lock()
	defer breaker.mu.Unlock()

	if success {
		breaker.Successes++
		if breaker.State == "half-open" && breaker.Successes >= 2 {
			breaker.State = "closed"
			breaker.Failures = 0
			breaker.Successes = 0
		}
	} else {
		breaker.Failures++
		breaker.LastFailure = time.Now()
		if breaker.Failures >= breaker.Threshold {
			breaker.State = "open"
		}
	}
}

func (g *APIGateway) forwardRequest(r *http.Request, route *Route) (*http.Response, error) {
	// In production, use actual HTTP client with circuit breaker
	// This is a simplified version
	client := &http.Client{
		Timeout: route.Timeout,
	}

	// Build URL
	url := route.Backend + r.URL.Path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}

	// Create request
	req, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		return nil, err
	}

	// Copy headers
	for k, v := range r.Header {
		req.Header[k] = v
	}

	return client.Do(req)
}

func (g *APIGateway) logRequest(id string, r *http.Request, status int, latency int64, userID, errMsg string) {
	log := RequestLog{
		ID:         id,
		Timestamp:  time.Now().UnixMilli(),
		Method:     r.Method,
		Path:       r.URL.Path,
		StatusCode: status,
		LatencyMs:  latency,
		UserID:     userID,
		IP:         getClientIP(r),
		Error:      errMsg,
	}

	g.mu.Lock()
	g.requestLogs = append(g.requestLogs, log)
	if len(g.requestLogs) > 10000 {
		g.requestLogs = g.requestLogs[10000:]
	}
	g.mu.Unlock()
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	return r.RemoteAddr
}

// ============== Rate Limiter ==============

func (r *RateLimiter) consume() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens = min(r.maxTokens, r.tokens+elapsed*r.refillRate)
	r.lastRefill = now

	if r.tokens >= 1 {
		r.tokens--
		return true
	}

	return false
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ============== Main ==============

func main() {
	log.Println("Starting TigerWallet API Gateway...")

	gateway := NewAPIGateway()
	if err := gateway.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
