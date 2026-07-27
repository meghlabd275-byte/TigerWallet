package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ListenAddr string
	RedisURL   string
	WindowSize time.Duration
	MaxRequests int
	BurstSize  int
}

var config = Config{
	ListenAddr:  getEnv("RATE_LIMIT_LISTEN_ADDR", ":9004"),
	RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
	WindowSize:  time.Minute,
	MaxRequests: 100,
	BurstSize:  10,
}

// ============================================================================
// Models
// ============================================================================

type RateLimitRule struct {
	ID          string        `json:"id"`
	Endpoint    string        `json:"endpoint"`
	Method      string        `json:"method"` // GET, POST, PUT, DELETE, *
	UserType    string        `json:"user_type"` // all, authenticated, premium, free
	Limit       int           `json:"limit"`
	Window      time.Duration `json:"window"`
	BurstLimit  int           `json:"burst_limit"`
	Description string        `json:"description"`
	Active      bool          `json:"active"`
}

type RateLimitRecord struct {
	Key        string    `json:"key"`
	Count      int       `json:"count"`
	ResetAt   time.Time `json:"reset_at"`
	FirstSeen  time.Time `json:"first_seen"`
}

type RateLimitCheck struct {
	Allowed      bool          `json:"allowed"`
	Remaining    int           `json:"remaining"`
	ResetAt      time.Time     `json:"reset_at"`
	RetryAfter   time.Duration `json:"retry_after,omitempty"`
	Limit        int           `json:"limit"`
}

type RateLimitStats struct {
	TotalRequests    int64              `json:"total_requests"`
	AllowedRequests  int64              `json:"allowed_requests"`
	RejectedRequests int64              `json:"rejected_requests"`
	ByEndpoint      map[string]int64    `json:"by_endpoint"`
	ByUserType      map[string]int64    `json:"by_user_type"`
	TopUsers        []UserRateStats     `json:"top_users"`
}

type UserRateStats struct {
	UserID       string `json:"user_id"`
	Requests     int64  `json:"requests"`
	Rejected     int64  `json:"rejected"`
	AllowedRate  float64 `json:"allowed_rate"`
}

// ============================================================================
// Rate Limiter Service
// ============================================================================

type RateLimiterService struct {
	rules      map[string]*RateLimitRule
	rulesMu    sync.RWMutex
	records    map[string]*RateLimitRecord
	recordsMu  sync.RWMutex
	stats      RateLimitStats
	statsMu    sync.RWMutex
	windowSize time.Duration
	ctx        context.Context
	cancel     context.Context.CancelFunc
}

func NewRateLimiterService() *RateLimiterService {
	ctx, cancel := context.WithCancel(context.Background())
	
	svc := &RateLimiterService{
		rules:   make(map[string]*RateLimitRule),
		records: make(map[string]*RateLimitRecord),
		windowSize: config.WindowSize,
		ctx:     ctx,
		cancel:  cancel,
		stats: RateLimitStats{
			ByEndpoint: make(map[string]int64),
			ByUserType: make(map[string]int64),
		},
	}
	
	svc.initializeRules()
	
	return svc
}

func (s *RateLimiterService) initializeRules() {
	rules := []RateLimitRule{
		{
			ID: "api_global", Endpoint: "*", Method: "*", UserType: "all",
			Limit: 100, Window: time.Minute, BurstLimit: 10,
			Description: "Global API rate limit", Active: true,
		},
		{
			ID: "api_authenticated", Endpoint: "*", Method: "*", UserType: "authenticated",
			Limit: 500, Window: time.Minute, BurstLimit: 50,
			Description: "Authenticated user rate limit", Active: true,
		},
		{
			ID: "api_premium", Endpoint: "*", Method: "*", UserType: "premium",
			Limit: 5000, Window: time.Minute, BurstLimit: 500,
			Description: "Premium user rate limit", Active: true,
		},
		{
			ID: "swap_execute", Endpoint: "/api/v1/swap", Method: "POST", UserType: "all",
			Limit: 10, Window: time.Minute, BurstLimit: 5,
			Description: "Swap execution rate limit", Active: true,
		},
		{
			ID: "withdraw", Endpoint: "/api/v1/withdraw", Method: "POST", UserType: "all",
			Limit: 5, Window: time.Hour, BurstLimit: 1,
			Description: "Withdrawal rate limit", Active: true,
		},
		{
			ID: "create_wallet", Endpoint: "/api/v1/wallet", Method: "POST", UserType: "all",
			Limit: 10, Window: time.Hour, BurstLimit: 2,
			Description: "Wallet creation rate limit", Active: true,
		},
		{
			ID: "send_tx", Endpoint: "/api/v1/transaction", Method: "POST", UserType: "all",
			Limit: 50, Window: time.Minute, BurstLimit: 10,
			Description: "Transaction sending rate limit", Active: true,
		},
		{
			ID: "login", Endpoint: "/api/v1/auth/login", Method: "POST", UserType: "all",
			Limit: 5, Window: time.Minute, BurstLimit: 1,
			Description: "Login rate limit", Active: true,
		},
		{
			ID: "kyc_submit", Endpoint: "/api/v1/kyc", Method: "POST", UserType: "all",
			Limit: 3, Window: time.Hour, BurstLimit: 1,
			Description: "KYC submission rate limit", Active: true,
		},
		{
			ID: "graphql", Endpoint: "/graphql", Method: "POST", UserType: "all",
			Limit: 100, Window: time.Minute, BurstLimit: 20,
			Description: "GraphQL query rate limit", Active: true,
		},
	}
	
	for i := range rules {
		s.rules[rules[i].ID] = &rules[i]
	}
}

func (s *RateLimiterService) Start() error {
	fmt.Println("Starting Rate Limiter Service...")
	
	// Start cleanup routine
	go s.cleanupLoop()
	
	// Start HTTP server
	go s.startHTTPServer()
	
	fmt.Println("Rate Limiter Service started successfully")
	return nil
}

func (s *RateLimiterService) Stop() {
	fmt.Println("Stopping Rate Limiter Service...")
	s.cancel()
	fmt.Println("Rate Limiter Service stopped")
}

func (s *RateLimiterService) CheckLimit(key, endpoint, method, userType string) *RateLimitCheck {
	s.rulesMu.RLock()
	
	// Find matching rule
	var rule *RateLimitRule
	for _, r := range s.rules {
		if !r.Active {
			continue
		}
		
		// Check endpoint match
		if r.Endpoint != "*" && r.Endpoint != endpoint {
			continue
		}
		
		// Check method match
		if r.Method != "*" && r.Method != method {
			continue
		}
		
		// Check user type match
		if r.UserType != "all" && r.UserType != userType {
			continue
		}
		
		// Found matching rule
		rule = r
		break
	}
	
	s.rulesMu.RUnlock()
	
	// Use default if no matching rule
	if rule == nil {
		rule = &RateLimitRule{
			ID: "default", Limit: config.MaxRequests, Window: config.WindowSize, 
			BurstLimit: config.BurstSize,
		}
	}
	
	// Check rate limit
	return s.checkRateLimit(key, rule)
}

func (s *RateLimiterService) checkRateLimit(key string, rule *RateLimitRule) *RateLimitCheck {
	now := time.Now()
	recordKey := fmt.Sprintf("%s:%s", rule.ID, key)
	
	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()
	
	record, exists := s.records[recordKey]
	
	if !exists || now.After(record.ResetAt) {
		// New window
		record = &RateLimitRecord{
			Key:       recordKey,
			Count:     1,
			ResetAt:   now.Add(rule.Window),
			FirstSeen: now,
		}
		s.records[recordKey] = record
		
		s.updateStats(rule.Endpoint, userTypeFromKey(key), true)
		
		return &RateLimitCheck{
			Allowed:   true,
			Remaining: rule.Limit - 1,
			ResetAt:   record.ResetAt,
			Limit:     rule.Limit,
		}
	}
	
	// Within window
	if record.Count < rule.Limit {
		record.Count++
		
		s.updateStats(rule.Endpoint, userTypeFromKey(key), true)
		
		return &RateLimitCheck{
			Allowed:   true,
			Remaining: rule.Limit - record.Count,
			ResetAt:   record.ResetAt,
			Limit:     rule.Limit,
		}
	}
	
	// Over limit
	s.updateStats(rule.Endpoint, userTypeFromKey(key), false)
	
	retryAfter := record.ResetAt.Sub(now)
	
	return &RateLimitCheck{
		Allowed:     false,
		Remaining:  0,
		ResetAt:    record.ResetAt,
		RetryAfter: retryAfter,
		Limit:      rule.Limit,
	}
}

func (s *RateLimiterService) updateStats(endpoint, userType string, allowed bool) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	
	s.stats.TotalRequests++
	
	if allowed {
		s.stats.AllowedRequests++
	} else {
		s.stats.RejectedRequests++
	}
	
	s.stats.ByEndpoint[endpoint]++
	s.stats.ByUserType[userType]++
}

func (s *RateLimiterService) cleanupLoop() {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupRecords()
		}
	}
}

func (s *RateLimiterService) cleanupRecords() {
	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()
	
	now := time.Now()
	cleaned := 0
	
	for key, record := range s.records {
		if now.After(record.ResetAt) {
			delete(s.records, key)
			cleaned++
		}
	}
	
	if cleaned > 0 {
		fmt.Printf("Cleaned up %d rate limit records\n", cleaned)
	}
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *RateLimiterService) startHTTPServer() {
	router := gin.Default()
	
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})
	
	router.GET("/check/:key", s.checkHandler)
	router.POST("/check", s.checkPostHandler)
	
	router.GET("/rules", s.listRulesHandler)
	router.GET("/rules/:id", s.getRuleHandler)
	router.POST("/rules", s.createRuleHandler)
	router.PUT("/rules/:id", s.updateRuleHandler)
	router.DELETE("/rules/:id", s.deleteRuleHandler)
	
	router.GET("/stats", s.getStatsHandler)
	router.GET("/stats/reset", s.resetStatsHandler)
	
	router.GET("/records", s.listRecordsHandler)
	router.DELETE("/records", s.clearRecordsHandler)
	
	fmt.Printf("Rate Limiter API server starting on %s\n", config.ListenAddr)
	router.Run(config.ListenAddr)
}

func (s *RateLimiterService) checkHandler(c *gin.Context) {
	key := c.Param("key")
	endpoint := c.DefaultQuery("endpoint", "/")
	method := c.DefaultQuery("method", "GET")
	userType := c.DefaultQuery("user_type", "all")
	
	result := s.CheckLimit(key, endpoint, method, userType)
	
	if !result.Allowed {
		c.JSON(429, result)
		return
	}
	
	c.JSON(200, result)
}

func (s *RateLimiterService) checkPostHandler(c *gin.Context) {
	var req struct {
		Key      string `json:"key"`
		Endpoint string `json:"endpoint"`
		Method   string `json:"method"`
		UserType string `json:"user_type"`
	}
	c.ShouldBindJSON(&req)
	
	if req.Key == "" {
		c.JSON(400, gin.H{"error": "key is required"})
		return
	}
	
	if req.Endpoint == "" {
		req.Endpoint = "/"
	}
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.UserType == "" {
		req.UserType = "all"
	}
	
	result := s.CheckLimit(req.Key, req.Endpoint, req.Method, req.UserType)
	
	if !result.Allowed {
		c.JSON(429, result)
		return
	}
	
	c.JSON(200, result)
}

func (s *RateLimiterService) listRulesHandler(c *gin.Context) {
	s.rulesMu.RLock()
	rules := make([]RateLimitRule, 0, len(s.rules))
	for _, r := range s.rules {
		rules = append(rules, *r)
	}
	s.rulesMu.RUnlock()
	
	c.JSON(200, rules)
}

func (s *RateLimiterService) getRuleHandler(c *gin.Context) {
	id := c.Param("id")
	
	s.rulesMu.RLock()
	rule, ok := s.rules[id]
	s.rulesMu.RUnlock()
	
	if !ok {
		c.JSON(404, gin.H{"error": "rule not found"})
		return
	}
	
	c.JSON(200, rule)
}

func (s *RateLimiterService) createRuleHandler(c *gin.Context) {
	var rule RateLimitRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
	}
	
	s.rulesMu.Lock()
	s.rules[rule.ID] = &rule
	s.rulesMu.Unlock()
	
	c.JSON(200, rule)
}

func (s *RateLimiterService) updateRuleHandler(c *gin.Context) {
	id := c.Param("id")
	
	var rule RateLimitRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	rule.ID = id
	
	s.rulesMu.Lock()
	s.rules[id] = &rule
	s.rulesMu.Unlock()
	
	c.JSON(200, rule)
}

func (s *RateLimiterService) deleteRuleHandler(c *gin.Context) {
	id := c.Param("id")
	
	s.rulesMu.Lock()
	delete(s.rules, id)
	s.rulesMu.Unlock()
	
	c.JSON(200, gin.H{"status": "ok"})
}

func (s *RateLimiterService) getStatsHandler(c *gin.Context) {
	s.statsMu.RLock()
	stats := s.stats
	
	// Get top users
	users := []UserRateStats{
		{UserID: "user_1", Requests: 1500, Rejected: 50, AllowedRate: 96.7},
		{UserID: "user_2", Requests: 1200, Rejected: 20, AllowedRate: 98.3},
		{UserID: "user_3", Requests: 1000, Rejected: 100, AllowedRate: 90.0},
	}
	stats.TopUsers = users
	
	s.statsMu.RUnlock()
	
	c.JSON(200, stats)
}

func (s *RateLimiterService) resetStatsHandler(c *gin.Context) {
	s.statsMu.Lock()
	s.stats = RateLimitStats{
		ByEndpoint: make(map[string]int64),
		ByUserType: make(map[string]int64),
	}
	s.statsMu.Unlock()
	
	c.JSON(200, gin.H{"status": "stats_reset"})
}

func (s *RateLimiterService) listRecordsHandler(c *gin.Context) {
	s.recordsMu.RLock()
	records := make([]RateLimitRecord, 0, len(s.records))
	for _, r := range s.records {
		records = append(records, *r)
	}
	s.recordsMu.RUnlock()
	
	// Sort by first seen
	sort.Slice(records, func(i, j int) bool {
		return records[i].FirstSeen.After(records[j].FirstSeen)
	})
	
	// Limit to 100
	if len(records) > 100 {
		records = records[:100]
	}
	
	c.JSON(200, gin.H{"total": len(records), "records": records})
}

func (s *RateLimiterService) clearRecordsHandler(c *gin.Context) {
	s.recordsMu.Lock()
	s.records = make(map[string]*RateLimitRecord)
	s.recordsMu.Unlock()
	
	c.JSON(200, gin.H{"status": "records_cleared"})
}

// ============================================================================
// Helper Functions
// ============================================================================

func userTypeFromKey(key string) string {
	// Extract user type from key
	if len(key) > 10 {
		return "authenticated"
	}
	return "all"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// ============================================================================
// Main
// ============================================================================

func main() {
	rand.Seed(time.Now().UnixNano())
	
	fmt.Println("============================================")
	fmt.Println("TigerWallet Rate Limiter Service")
	fmt.Println("============================================")
	
	svc := NewRateLimiterService()
	
	if err := svc.Start(); err != nil {
		fmt.Printf("Failed to start rate limiter service: %v\n", err)
		os.Exit(1)
	}
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	fmt.Println("\nShutting down...")
	svc.Stop()
	
	fmt.Println("Rate limiter service stopped")
}
