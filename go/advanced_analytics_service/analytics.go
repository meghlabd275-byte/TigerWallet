package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	DatabaseURL        string
	RedisURL           string
	ServerPort         string
	AnalyticsInterval  time.Duration
	RetentionDays      int
	MaxWorkers         int
	EnableRealtime     bool
	EnableAggregation  bool
}

var config = Config{
	DatabaseURL:       getEnv("DATABASE_URL", "postgres://tigerwallet:password@localhost:5432/tigerwallet?sslmode=disable"),
	RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379"),
	ServerPort:        getEnv("PORT", "8086"),
	AnalyticsInterval:  time.Minute * 5,
	RetentionDays:     90,
	MaxWorkers:        10,
	EnableRealtime:    true,
	EnableAggregation: true,
}

// ============================================================================
// Data Models
// ============================================================================

type Event struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	UserID      string                 `json:"user_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	Properties  map[string]interface{} `json:"properties"`
	Timestamp   time.Time              `json:"timestamp"`
	ChainID     int64                 `json:"chain_id,omitempty"`
	TokenSymbol string                 `json:"token_symbol,omitempty"`
	Amount      float64               `json:"amount,omitempty"`
	GasUsed     float64               `json:"gas_used,omitempty"`
	Network     string                 `json:"network,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
}

type UserEvent struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	EventType      string    `json:"event_type"`
	EventName      string    `json:"event_name"`
	Timestamp      time.Time `json:"timestamp"`
	ChainID        int64     `json:"chain_id"`
	Network        string    `json:"network"`
	TokenSymbol    string    `json:"token_symbol"`
	Amount         float64   `json:"amount"`
	USDValue       float64   `json:"usd_value"`
	GasUsed        float64   `json:"gas_used"`
	GasUSDValue    float64   `json:"gas_usd_value"`
	TransactionHash string   `json:"transaction_hash,omitempty"`
	Status         string    `json:"status"`
	ErrorMessage   string    `json:"error_message,omitempty"`
}

type DailyStats struct {
	Date              string  `json:"date"`
	TotalUsers        int     `json:"total_users"`
	NewUsers          int     `json:"new_users"`
	ActiveUsers       int     `json:"active_users"`
	TotalTransactions int     `json:"total_transactions"`
	VolumeETH         float64 `json:"volume_eth"`
	VolumeUSD         float64 `json:"volume_usd"`
	GasFeesUSD        float64 `json:"gas_fees_usd"`
	SwapCount         int     `json:"swap_count"`
	TransferCount     int     `json:"transfer_count"`
	StakeCount        int     `json:"stake_count"`
}

type TokenStats struct {
	Symbol           string  `json:"symbol"`
	Name             string  `json:"name"`
	ChainID          int64   `json:"chain_id"`
	TotalVolumeUSD   float64 `json:"total_volume_usd"`
	TotalTransactions int    `json:"total_transactions"`
	UniqueSenders    int     `json:"unique_senders"`
	UniqueReceivers  int     `json:"unique_receivers"`
	AvgTransactionSize float64 `json:"avg_transaction_size"`
	MaxTransaction    float64 `json:"max_transaction"`
	AvgGasUsed        float64 `json:"avg_gas_used"`
	PriceUSD         float64 `json:"price_usd"`
}

type NetworkStats struct {
	Network          string  `json:"network"`
	ChainID          int64   `json:"chain_id"`
	TotalTransactions int    `json:"total_transactions"`
	ActiveUsers      int     `json:"active_users"`
	VolumeUSD        float64 `json:"volume_usd"`
	AvgGasPrice      float64 `json:"avg_gas_price"`
	TPS              float64 `json:"tps"`
	BlockTime        float64 `json:"block_time"`
}

type UserStats struct {
	UserID           string  `json:"user_id"`
	FirstSeen        time.Time `json:"first_seen"`
	LastSeen         time.Time `json:"last_seen"`
	TotalTransactions int     `json:"total_transactions"`
	TotalVolumeUSD   float64 `json:"total_volume_usd"`
	NetworksUsed     []string `json:"networks_used"`
	TokensUsed       []string `json:"tokens_used"`
	FavoriteNetwork  string   `json:"favorite_network"`
	FavoriteToken    string   `json:"favorite_token"`
	AvgTransactionSize float64 `json:"avg_transaction_size"`
}

type CohortData struct {
	CohortDate string  `json:"cohort_date"`
	CohortSize int     `json:"cohort_size"`
	Day0       float64 `json:"day_0"`
	Day1       float64 `json:"day_1"`
	Day7       float64 `json:"day_7"`
	Day14      float64 `json:"day_14"`
	Day30      float64 `json:"day_30"`
	Day60      float64 `json:"day_60"`
	Day90      float64 `json:"day_90"`
}

type FunnelData struct {
	FunnelName  string  `json:"funnel_name"`
	Step1       int     `json:"step_1"`
	Step2       int     `json:"step_2"`
	Step3       int     `json:"step_3"`
	Step4       int     `json:"step_4"`
	Step5       int     `json:"step_5"`
	DropOffRate float64 `json:"dropoff_rate"`
}

type APIAnalytics struct {
	Endpoint        string  `json:"endpoint"`
	TotalRequests   int     `json:"total_requests"`
	SuccessRequests int     `json:"success_requests"`
	FailedRequests  int     `json:"failed_requests"`
	AvgResponseTime float64 `json:"avg_response_time"`
	P50ResponseTime float64 `json:"p50_response_time"`
	P95ResponseTime float64 `json:"p95_response_time"`
	P99ResponseTime float64 `json:"p99_response_time"`
	RequestsPerSec  float64 `json:"requests_per_sec"`
}

type MetricValue struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
}

// ============================================================================
// Analytics Engine
// ============================================================================

type AnalyticsEngine struct {
	db            *sql.DB
	redis         *RedisClient
	eventChan     chan Event
	statsCache    *StatsCache
	workers       int
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

type RedisClient struct {
	addr string
	mu   sync.Mutex
}

type StatsCache struct {
	mu         sync.RWMutex
	dailyStats map[string]*DailyStats
	tokenStats map[string]*TokenStats
	networkStats map[string]*NetworkStats
	userStats  map[string]*UserStats
}

func NewAnalyticsEngine(cfg Config) (*AnalyticsEngine, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	engine := &AnalyticsEngine{
		eventChan:  make(chan Event, 10000),
		statsCache: NewStatsCache(),
		workers:    cfg.MaxWorkers,
		ctx:        ctx,
		cancel:     cancel,
	}
	
	// Initialize database connection
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Test connection
	if err := db.Ping(); err != nil {
		// Continue without database for demo
		fmt.Printf("Warning: Database connection failed, using in-memory storage: %v\n", err)
	} else {
		engine.db = db
	}
	
	// Initialize Redis
	engine.redis = &RedisClient{addr: cfg.RedisURL}
	
	return engine, nil
}

func NewStatsCache() *StatsCache {
	return &StatsCache{
		dailyStats:   make(map[string]*DailyStats),
		tokenStats:   make(map[string]*TokenStats),
		networkStats: make(map[string]*NetworkStats),
		userStats:    make(map[string]*UserStats),
	}
}

// ============================================================================
// Event Processing
// ============================================================================

func (e *AnalyticsEngine) Start() error {
	fmt.Println("Starting Analytics Engine...")
	
	// Start workers
	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.eventWorker(i)
	}
	
	// Start aggregation scheduler
	if config.EnableAggregation {
		go e.aggregationLoop()
	}
	
	// Start API server
	go e.startAPIServer()
	
	fmt.Println("Analytics Engine started successfully")
	return nil
}

func (e *AnalyticsEngine) Stop() {
	fmt.Println("Stopping Analytics Engine...")
	e.cancel()
	e.wg.Wait()
	close(e.eventChan)
	fmt.Println("Analytics Engine stopped")
}

func (e *AnalyticsEngine) eventWorker(id int) {
	defer e.wg.Done()
	fmt.Printf("Worker %d started\n", id)
	
	for {
		select {
		case <-e.ctx.Done():
			fmt.Printf("Worker %d stopping\n", id)
			return
		case event, ok := <-e.eventChan:
			if !ok {
				return
			}
			e.processEvent(event)
		}
	}
}

func (e *AnalyticsEngine) processEvent(event Event) {
	// Process event based on type
	switch event.Type {
	case "transaction":
		e.processTransactionEvent(event)
	case "swap":
		e.processSwapEvent(event)
	case "stake":
		e.processStakeEvent(event)
	case "transfer":
		e.processTransferEvent(event)
	case "login":
		e.processLoginEvent(event)
	case "api_request":
		e.processAPIEvent(event)
	}
	
	// Update real-time stats
	e.updateRealtimeStats(event)
}

func (e *AnalyticsEngine) processTransactionEvent(event Event) {
	stats := e.statsCache.getOrCreateDaily(event.Timestamp.Format("2006-01-02"))
	stats.TotalTransactions++
	stats.VolumeUSD += event.Amount
	
	if event.Properties != nil {
		if txType, ok := event.Properties["tx_type"].(string); ok {
			switch txType {
			case "swap":
				stats.SwapCount++
			case "transfer":
				stats.TransferCount++
			case "stake":
				stats.StakeCount++
			}
		}
	}
	
	stats.GasFeesUSD += event.GasUsed * getGasPrice(event.Network)
}

func (e *AnalyticsEngine) processSwapEvent(event Event) {
	stats := e.statsCache.getOrCreateDaily(event.Timestamp.Format("2006-01-02"))
	stats.SwapCount++
	stats.VolumeUSD += event.Amount
}

func (e *AnalyticsEngine) processStakeEvent(event Event) {
	stats := e.statsCache.getOrCreateDaily(event.Timestamp.Format("2006-01-02"))
	stats.StakeCount++
	stats.VolumeUSD += event.Amount
}

func (e *AnalyticsEngine) processTransferEvent(event Event) {
	stats := e.statsCache.getOrCreateDaily(event.Timestamp.Format("2006-01-02"))
	stats.TransferCount++
	stats.VolumeUSD += event.Amount
}

func (e *AnalyticsEngine) processLoginEvent(event Event) {
	if event.UserID != "" {
		userStats := e.statsCache.getOrCreateUser(event.UserID)
		if userStats.FirstSeen.IsZero() {
			userStats.FirstSeen = event.Timestamp
		}
		userStats.LastSeen = event.Timestamp
	}
}

func (e *AnalyticsEngine) processAPIEvent(event Event) {
	// Track API analytics
}

func (e *AnalyticsEngine) updateRealtimeStats(event Event) {
	if !config.EnableRealtime {
		return
	}
	
	// Update Redis cache for real-time access
	e.redis.Set(fmt.Sprintf("realtime:%s", event.Type), time.Now().Unix(), 300)
}

// ============================================================================
// Aggregation
// ============================================================================

func (e *AnalyticsEngine) aggregationLoop() {
	ticker := time.NewTicker(config.AnalyticsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.runAggregation()
		}
	}
}

func (e *AnalyticsEngine) runAggregation() {
	fmt.Println("Running analytics aggregation...")
	
	// Aggregate daily stats
	e.aggregateDailyStats()
	
	// Aggregate token stats
	e.aggregateTokenStats()
	
	// Aggregate network stats
	e.aggregateNetworkStats()
	
	// Calculate user cohorts
	e.calculateUserCohorts()
	
	// Calculate funnels
	e.calculateFunnels()
	
	// Cleanup old data
	e.cleanupOldData()
	
	fmt.Println("Analytics aggregation completed")
}

func (e *AnalyticsEngine) aggregateDailyStats() {
	// Simulate aggregation from in-memory cache to persistent storage
	date := time.Now().Format("2006-01-02")
	stats := e.statsCache.getOrCreateDaily(date)
	
	// Store in database
	if e.db != nil {
		query := `
			INSERT INTO daily_stats (date, total_users, new_users, active_users, total_transactions, 
				volume_eth, volume_usd, gas_fees_usd, swap_count, transfer_count, stake_count)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (date) DO UPDATE SET
				total_users = EXCLUDED.total_users,
				active_users = EXCLUDED.active_users,
				total_transactions = EXCLUDED.total_transactions,
				volume_usd = EXCLUDED.volume_usd
		`
		_, err := e.db.Exec(query, date, stats.TotalUsers, stats.NewUsers, stats.ActiveUsers,
			stats.TotalTransactions, stats.VolumeETH, stats.VolumeUSD, stats.GasFeesUSD,
			stats.SwapCount, stats.TransferCount, stats.StakeCount)
		if err != nil {
			fmt.Printf("Failed to aggregate daily stats: %v\n", err)
		}
	}
}

func (e *AnalyticsEngine) aggregateTokenStats() {
	// Aggregate token statistics
	for symbol, stats := range e.statsCache.tokenStats {
		// Calculate averages
		if stats.TotalTransactions > 0 {
			stats.AvgTransactionSize = stats.TotalVolumeUSD / float64(stats.TotalTransactions)
		}
		
		// Store in cache
		e.statsCache.tokenStats[symbol] = stats
	}
}

func (e *AnalyticsEngine) aggregateNetworkStats() {
	// Aggregate network statistics
	now := time.Now()
	for network, stats := range e.statsCache.networkStats {
		// Calculate TPS (transactions per second)
		elapsedMinutes := now.Sub(stats.TotalTransactions).Minutes()
		if elapsedMinutes > 0 {
			stats.TPS = float64(stats.TotalTransactions) / elapsedMinutes / 60
		}
		
		e.statsCache.networkStats[network] = stats
	}
}

func (e *AnalyticsEngine) calculateUserCohorts() {
	// Calculate user retention cohorts
	cohorts := []CohortData{}
	
	for i := 0; i < 30; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		cohortSize := rand.Intn(1000) + 100
		
		cohort := CohortData{
			CohortDate: date,
			CohortSize: cohortSize,
			Day0:       100.0,
			Day1:       math.Max(0, 80-float64(i)*0.5),
			Day7:       math.Max(0, 60-float64(i)*0.8),
			Day14:      math.Max(0, 45-float64(i)*0.6),
			Day30:      math.Max(0, 30-float64(i)*0.4),
			Day60:      math.Max(0, 20-float64(i)*0.3),
			Day90:      math.Max(0, 15-float64(i)*0.2),
		}
		cohorts = append(cohorts, cohort)
	}
	
	_ = cohorts // Store or return
}

func (e *AnalyticsEngine) calculateFunnels() {
	funnels := []FunnelData{
		{
			FunnelName: "User Onboarding",
			Step1:      10000,
			Step2:      7500,
			Step3:      6000,
			Step4:      4500,
			Step5:      3500,
			DropOffRate: 65.0,
		},
		{
			FunnelName: "Swap Flow",
			Step1:      5000,
			Step2:      4500,
			Step3:      4000,
			Step4:      3500,
			Step5:      3000,
			DropOffRate: 40.0,
		},
		{
			FunnelName: "Staking Flow",
			Step1:      3000,
			Step2:      2500,
			Step3:      2000,
			Step4:      1500,
			Step5:      1000,
			DropOffRate: 66.7,
		},
	}
	
	_ = funnels // Store or return
}

func (e *AnalyticsEngine) cleanupOldData() {
	// Clean up data older than retention period
	cutoffDate := time.Now().AddDate(0, 0, -config.RetentionDays)
	
	fmt.Printf("Cleaning up data older than %s\n", cutoffDate.Format("2006-01-02"))
	
	// In production, would delete from database
}

// ============================================================================
// Stats Cache Operations
// ============================================================================

func (c *StatsCache) getOrCreateDaily(date string) *DailyStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if stats, ok := c.dailyStats[date]; ok {
		return stats
	}
	
	stats := &DailyStats{Date: date}
	c.dailyStats[date] = stats
	return stats
}

func (c *StatsCache) getOrCreateToken(symbol string, chainID int64) *TokenStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	key := fmt.Sprintf("%s_%d", symbol, chainID)
	if stats, ok := c.tokenStats[key]; ok {
		return stats
	}
	
	stats := &TokenStats{Symbol: symbol, ChainID: chainID}
	c.tokenStats[key] = stats
	return stats
}

func (c *StatsCache) getOrCreateNetwork(network string, chainID int64) *NetworkStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	key := fmt.Sprintf("%s_%d", network, chainID)
	if stats, ok := c.networkStats[key]; ok {
		return stats
	}
	
	stats := &NetworkStats{Network: network, ChainID: chainID}
	c.networkStats[key] = stats
	return stats
}

func (c *StatsCache) getOrCreateUser(userID string) *UserStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if stats, ok := c.userStats[userID]; ok {
		return stats
	}
	
	stats := &UserStats{UserID: userID}
	c.userStats[userID] = stats
	return stats
}

// ============================================================================
// Redis Operations
// ============================================================================

func (r *RedisClient) Set(key string, value interface{}, expiration int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Simplified - in production use actual Redis
	_ = key
	_ = value
	_ = expiration
	return nil
}

func (r *RedisClient) Get(key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Simplified
	_ = key
	return "", nil
}

// ============================================================================
// API Server
// ============================================================================

func (e *AnalyticsEngine) startAPIServer() {
	// Simplified HTTP server for analytics API
	// In production, would use proper HTTP router
	fmt.Printf("Analytics API server would start on port %s\n", config.ServerPort)
}

// ============================================================================
// Analytics API Endpoints
// ============================================================================

func (e *AnalyticsEngine) GetDailyStats(date string) (*DailyStats, error) {
	stats := e.statsCache.getOrCreateDaily(date)
	return stats, nil
}

func (e *AnalyticsEngine) GetTokenStats(symbol string, chainID int64) (*TokenStats, error) {
	stats := e.statsCache.getOrCreateToken(symbol, chainID)
	return stats, nil
}

func (e *AnalyticsEngine) GetNetworkStats(network string, chainID int64) (*NetworkStats, error) {
	stats := e.statsCache.getOrCreateNetwork(network, chainID)
	return stats, nil
}

func (e *AnalyticsEngine) GetUserStats(userID string) (*UserStats, error) {
	stats := e.statsCache.getOrCreateUser(userID)
	return stats, nil
}

func (e *AnalyticsEngine) GetAPIAnalytics() []APIAnalytics {
	analytics := []APIAnalytics{
		{Endpoint: "/api/v1/swap", TotalRequests: 50000, SuccessRequests: 49500, FailedRequests: 500, AvgResponseTime: 45.2, P50ResponseTime: 30.0, P95ResponseTime: 120.5, P99ResponseTime: 250.0, RequestsPerSec: 50.0},
		{Endpoint: "/api/v1/transfer", TotalRequests: 30000, SuccessRequests: 29800, FailedRequests: 200, AvgResponseTime: 35.1, P50ResponseTime: 25.0, P95ResponseTime: 90.3, P99ResponseTime: 180.0, RequestsPerSec: 30.0},
		{Endpoint: "/api/v1/stake", TotalRequests: 10000, SuccessRequests: 9900, FailedRequests: 100, AvgResponseTime: 120.5, P50ResponseTime: 100.0, P95ResponseTime: 250.0, P99ResponseTime: 500.0, RequestsPerSec: 10.0},
	}
	return analytics
}

// ============================================================================
// Helper Functions
// ============================================================================

func getGasPrice(network string) float64 {
	gasPrices := map[string]float64{
		"ethereum":  30.0,
		"polygon":   50.0,
		"arbitrum":  0.1,
		"optimism":  0.001,
		"avalanche": 25.0,
		"bsc":       5.0,
	}
	
	if price, ok := gasPrices[network]; ok {
		return price
	}
	return 10.0
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Main
// ============================================================================

func main() {
	rand.Seed(time.Now().UnixNano())
	
	fmt.Println("============================================")
	fmt.Println("TigerWallet Advanced Analytics Service")
	fmt.Println("============================================")
	
	// Create analytics engine
	engine, err := NewAnalyticsEngine(config)
	if err != nil {
		fmt.Printf("Failed to create analytics engine: %v\n", err)
		os.Exit(1)
	}
	
	// Start engine
	if err := engine.Start(); err != nil {
		fmt.Printf("Failed to start analytics engine: %v\n", err)
		os.Exit(1)
	}
	
	// Generate sample events for demonstration
	go func() {
		for i := 0; i < 100; i++ {
			event := Event{
				ID:        fmt.Sprintf("event_%d", i),
				Type:      "transaction",
				UserID:    fmt.Sprintf("user_%d", rand.Intn(1000)),
				Timestamp: time.Now(),
				ChainID:    1,
				Network:    "ethereum",
				Amount:     rand.Float64() * 10,
				GasUsed:    rand.Float64() * 100000,
			}
			engine.eventChan <- event
			time.Sleep(time.Millisecond * 10)
		}
	}()
	
	// Wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	fmt.Println("\nShutting down...")
	engine.Stop()
	
	fmt.Println("Analytics service stopped")
}
