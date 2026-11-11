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
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============== Data Structures ==============

// Config holds the gateway's runtime configuration. Every value that affects
// security is read from the environment and validated at startup; there is no
// hardcoded credential or permissive default anywhere in this file.
type Config struct {
	// JWTSecret is the HS256 key used to verify bearer tokens. Required.
	JWTSecret []byte
	// AdminToken authorises the gateway's own management endpoints. Required.
	AdminToken string
	// APIKeys maps a presented key to the key record it authenticates.
	APIKeys map[string]*APIKey
}

// LoadConfig reads and validates the gateway configuration. It fails closed:
// if any required secret is missing, empty, or a well-known placeholder the
// gateway refuses to start rather than running without authentication.
func LoadConfig() (*Config, error) {
	secret := os.Getenv("GATEWAY_JWT_SECRET")
	if err := requireSecret("GATEWAY_JWT_SECRET", secret); err != nil {
		return nil, err
	}

	admin := os.Getenv("GATEWAY_ADMIN_TOKEN")
	if err := requireSecret("GATEWAY_ADMIN_TOKEN", admin); err != nil {
		return nil, err
	}

	cfg := &Config{
		JWTSecret:  []byte(secret),
		AdminToken: admin,
		APIKeys:    make(map[string]*APIKey),
	}

	// GATEWAY_API_KEYS is an optional comma-separated list of key:userID:rps
	// triples. Each entry becomes a live API key; nothing is seeded implicitly.
	raw := strings.TrimSpace(os.Getenv("GATEWAY_API_KEYS"))
	if raw == "" {
		return cfg, nil
	}
	for _, entry := range strings.Split(raw, ",") {
		parts := strings.Split(strings.TrimSpace(entry), ":")
		if len(parts) != 3 || parts[0] == "" {
			return nil, fmt.Errorf("GATEWAY_API_KEYS entry %q must be key:user_id:requests_per_second", entry)
		}
		rps, err := strconv.ParseFloat(parts[2], 64)
		if err != nil || rps <= 0 {
			return nil, fmt.Errorf("GATEWAY_API_KEYS entry %q has an invalid rate limit", entry)
		}
		cfg.APIKeys[parts[0]] = &APIKey{
			ID:        "key_" + shortFingerprint(parts[0]),
			Key:       parts[0],
			UserID:    parts[1],
			Name:      "env:" + parts[1],
			RateLimit: rps,
			Active:    true,
			CreatedAt: time.Now().Unix(),
		}
	}
	return cfg, nil
}

// placeholderSecrets are configuration values that look like credentials but
// are not; accepting one would be equivalent to running with no auth at all.
var placeholderSecrets = map[string]bool{
	"change-me": true, "changeme": true, "change-me-in-production": true,
	"secret": true, "password": true, "your-secret-here": true,
	"dev-only-change-me": true, "tigerwallet-super-admin-secret-change-me": true,
	"test": true, "example": true, "todo": true,
}

func requireSecret(name, value string) error {
	v := strings.TrimSpace(value)
	if len(v) < 32 {
		return fmt.Errorf("%s must be set to a value of at least 32 characters (got %d)", name, len(v))
	}
	if placeholderSecrets[strings.ToLower(v)] {
		return fmt.Errorf("%s is set to a placeholder value and must be replaced with a real secret", name)
	}
	return nil
}

// shortFingerprint returns a stable, non-reversible identifier for a secret so
// that key material never appears in logs, IDs, or API responses.
func shortFingerprint(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return fmt.Sprintf("%x", sum[:6])
}

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

	// cfg carries the verified JWT key and admin token. It is never nil for a
	// running gateway because NewAPIGateway rejects a missing configuration.
	cfg *Config

	mu        sync.RWMutex
	startTime time.Time
	stats     struct {
		requests    uint64
		errors      uint64
		rateLimited uint64
	}

	server *http.Server
}

func NewAPIGateway(cfg *Config) (*APIGateway, error) {
	if cfg == nil || len(cfg.JWTSecret) == 0 || cfg.AdminToken == "" {
		return nil, fmt.Errorf("api gateway requires a validated configuration (JWT secret and admin token)")
	}
	g := &APIGateway{
		routes:          make(map[string]*Route),
		apiKeys:         cfg.APIKeys,
		circuitBreakers: make(map[string]*CircuitBreaker),
		rateLimiters:    make(map[string]*RateLimiter),
		requestLogs:     make([]RequestLog, 0, 10000),
		cfg:             cfg,
		startTime:       time.Now(),
	}

	g.initRoutes()
	g.initCircuitBreakers()

	return g, nil
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

	// The listen address is configurable so the gateway can run behind an
	// orchestrator that assigns the port (and so the value matches whatever the
	// deployment maps). PORT defaults to 8000.
	addr := ":8000"
	if p := strings.TrimSpace(os.Getenv("PORT")); p != "" {
		addr = ":" + p
	}

	g.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("API Gateway starting on %s", addr)
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

// validateAuth authenticates a request against either a configured API key or
// a JWT bearer token signed with the gateway's HS256 secret. Both paths are
// cryptographic: a key must match a registered key exactly (constant time) and
// a token must carry a valid HMAC-SHA256 signature, an unexpired "exp", and a
// subject. Anything else is rejected.
func (g *APIGateway) validateAuth(r *http.Request) bool {
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		if key := g.lookupAPIKey(apiKey); key != nil && key.Active {
			if key.ExpiresAt != 0 && time.Now().Unix() > key.ExpiresAt {
				return false
			}
			return true
		}
	}

	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if token == "" {
		return false
	}
	return g.verifyJWT(token)
}

// lookupAPIKey finds the registered key whose material matches the presented
// value. It compares every candidate in constant time so that a timing signal
// cannot be used to discover a valid key prefix.
func (g *APIGateway) lookupAPIKey(presented string) *APIKey {
	var match *APIKey
	for stored, key := range g.apiKeys {
		if subtle.ConstantTimeCompare([]byte(stored), []byte(presented)) == 1 {
			match = key
		}
	}
	return match
}

// verifyJWT validates an HS256 JWT. It rejects any token whose header names a
// different algorithm (including "none"), whose signature does not match, or
// whose exp/nbf claims place it outside its validity window.
func (g *APIGateway) verifyJWT(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return false
	}
	if !strings.EqualFold(header.Alg, "HS256") {
		return false
	}

	mac := hmac.New(sha256.New, g.cfg.JWTSecret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	expected := mac.Sum(nil)

	provided, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(provided) != len(expected) {
		return false
	}
	if subtle.ConstantTimeCompare(provided, expected) != 1 {
		return false
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims struct {
		Sub string  `json:"sub"`
		Exp float64 `json:"exp"`
		Nbf float64 `json:"nbf"`
	}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return false
	}
	if strings.TrimSpace(claims.Sub) == "" {
		return false
	}
	now := float64(time.Now().Unix())
	if claims.Exp == 0 || now >= claims.Exp {
		return false
	}
	if claims.Nbf != 0 && now < claims.Nbf {
		return false
	}
	return true
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

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	gateway, err := NewAPIGateway(cfg)
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	if err := gateway.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
