package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ListenAddr        string
	ElasticsearchURL  string
	KafkaBrokers      []string
	S3Bucket          string
	AWSRegion         string
	RetentionDays     int
	BatchSize         int
	FlushInterval     time.Duration
	MaxWorkers        int
	EnableCompression bool
}

var config = Config{
	ListenAddr:       getEnv("LOG_LISTEN_ADDR", ":9200"),
	ElasticsearchURL: getEnv("ELASTICSEARCH_URL", "http://localhost:9200"),
	KafkaBrokers:     strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
	S3Bucket:         getEnv("S3_BUCKET", "tigerwallet-logs"),
	AWSRegion:        getEnv("AWS_REGION", "us-east-1"),
	RetentionDays:    90,
	BatchSize:        1000,
	FlushInterval:    time.Second * 5,
	MaxWorkers:       20,
}

// ============================================================================
// Log Models
// ============================================================================

type LogEntry struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Service     string                 `json:"service"`
	Message     string                 `json:"message"`
	Fields      map[string]interface{} `json:"fields"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	Network     string                 `json:"network,omitempty"`
	ChainID     int64                  `json:"chain_id,omitempty"`
	TokenSymbol string                 `json:"token_symbol,omitempty"`
	GasUsed     float64                `json:"gas_used,omitempty"`
	ErrorStack  string                 `json:"error_stack,omitempty"`
}

type LogQuery struct {
	Query     string    `json:"query"`
	Service   string    `json:"service,omitempty"`
	Level     string    `json:"level,omitempty"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	UserID    string    `json:"user_id,omitempty"`
	TraceID   string    `json:"trace_id,omitempty"`
	Network   string    `json:"network,omitempty"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
}

type LogAggregation struct {
	Service    string         `json:"service"`
	Level      string         `json:"level"`
	Count      int64          `json:"count"`
	Percentage float64        `json:"percentage"`
	Errors     []ErrorSummary `json:"errors,omitempty"`
}

type ErrorSummary struct {
	Error   string    `json:"error"`
	Message string    `json:"message"`
	Count   int64     `json:"count"`
	FirstAt time.Time `json:"first_at"`
	LastAt  time.Time `json:"last_at"`
}

type LogStats struct {
	TotalLogs int64            `json:"total_logs"`
	ByLevel   map[string]int64 `json:"by_level"`
	ByService map[string]int64 `json:"by_service"`
	ByNetwork map[string]int64 `json:"by_network"`
	Errors    []ErrorSummary   `json:"errors"`
	TimeRange TimeRange        `json:"time_range"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ============================================================================
// Log Aggregation Engine
// ============================================================================

type LogAggregationEngine struct {
	logChan     chan *LogEntry
	workerGroup sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	stats       LogStats
	statsMu     sync.RWMutex
	batch       []*LogEntry
	batchMu     sync.Mutex
	flushTicker *time.Ticker
}

func NewLogAggregationEngine() *LogAggregationEngine {
	ctx, cancel := context.WithCancel(context.Background())

	engine := &LogAggregationEngine{
		logChan: make(chan *LogEntry, 50000),
		ctx:     ctx,
		cancel:  cancel,
	}

	engine.stats = LogStats{
		ByLevel:   make(map[string]int64),
		ByService: make(map[string]int64),
		ByNetwork: make(map[string]int64),
	}

	return engine
}

func (e *LogAggregationEngine) Start() error {
	fmt.Println("Starting Log Aggregation Engine...")

	for i := 0; i < config.MaxWorkers; i++ {
		e.workerGroup.Add(1)
		go e.logWorker(i)
	}

	e.flushTicker = time.NewTicker(config.FlushInterval)
	go e.flushLoop()

	go e.startHTTPServer()

	fmt.Println("Log Aggregation Engine started successfully")
	return nil
}

func (e *LogAggregationEngine) Stop() {
	fmt.Println("Stopping Log Aggregation Engine...")
	e.cancel()
	e.workerGroup.Wait()
	e.flushTicker.Stop()
	e.flushBatch()
	fmt.Println("Log Aggregation Engine stopped")
}

func (e *LogAggregationEngine) logWorker(id int) {
	defer e.workerGroup.Done()

	for {
		select {
		case <-e.ctx.Done():
			return
		case entry, ok := <-e.logChan:
			if !ok {
				return
			}
			e.processLog(entry)
		}
	}
}

func (e *LogAggregationEngine) processLog(entry *LogEntry) {
	e.batchMu.Lock()
	e.batch = append(e.batch, entry)
	batchLen := len(e.batch)
	e.batchMu.Unlock()

	e.updateStats(entry)

	if batchLen >= config.BatchSize {
		go e.flushBatch()
	}
}

func (e *LogAggregationEngine) updateStats(entry *LogEntry) {
	e.statsMu.Lock()
	defer e.statsMu.Unlock()

	e.stats.TotalLogs++
	e.stats.ByLevel[entry.Level]++
	e.stats.ByService[entry.Service]++
	if entry.Network != "" {
		e.stats.ByNetwork[entry.Network]++
	}

	if entry.Level == "error" || entry.Level == "fatal" {
		errorMsg := entry.Message
		if len(errorMsg) > 100 {
			errorMsg = errorMsg[:100]
		}

		found := false
		for i := range e.stats.Errors {
			if e.stats.Errors[i].Message == errorMsg {
				e.stats.Errors[i].Count++
				e.stats.Errors[i].LastAt = entry.Timestamp
				found = true
				break
			}
		}

		if !found {
			e.stats.Errors = append(e.stats.Errors, ErrorSummary{
				Error: errorMsg, Message: entry.Message, Count: 1,
				FirstAt: entry.Timestamp, LastAt: entry.Timestamp,
			})
		}
	}

	if e.stats.TimeRange.Start.IsZero() || entry.Timestamp.Before(e.stats.TimeRange.Start) {
		e.stats.TimeRange.Start = entry.Timestamp
	}
	if e.stats.TimeRange.End.IsZero() || entry.Timestamp.After(e.stats.TimeRange.End) {
		e.stats.TimeRange.End = entry.Timestamp
	}
}

func (e *LogAggregationEngine) flushLoop() {
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-e.flushTicker.C:
			e.flushBatch()
		}
	}
}

func (e *LogAggregationEngine) flushBatch() {
	e.batchMu.Lock()
	if len(e.batch) == 0 {
		e.batchMu.Unlock()
		return
	}

	batch := e.batch
	e.batch = make([]*LogEntry, 0, config.BatchSize)
	e.batchMu.Unlock()

	for _, entry := range batch {
		jsonBytes, _ := json.Marshal(entry)
		fmt.Println(string(jsonBytes))
	}
}

// ============================================================================
// HTTP API
// ============================================================================

func (e *LogAggregationEngine) startHTTPServer() {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	router.POST("/logs", e.ingestLogHandler)
	router.POST("/logs/batch", e.ingestBatchHandler)
	router.POST("/logs/query", e.queryLogsHandler)
	router.GET("/logs/stats", e.getStatsHandler)
	router.GET("/logs/aggregate", e.getAggregateHandler)
	router.GET("/logs/errors", e.getErrorsHandler)
	router.GET("/logs/trace/:trace_id", e.getTraceHandler)

	fmt.Printf("Log API server starting on %s\n", config.ListenAddr)
	log.Fatal(router.Run(config.ListenAddr))
}

func (e *LogAggregationEngine) ingestLogHandler(c *gin.Context) {
	var entry LogEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	if entry.ID == "" {
		entry.ID = generateID()
	}

	e.logChan <- &entry
	c.JSON(200, gin.H{"status": "ok", "id": entry.ID})
}

func (e *LogAggregationEngine) ingestBatchHandler(c *gin.Context) {
	var entries []LogEntry
	if err := c.ShouldBindJSON(&entries); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	count := 0
	for i := range entries {
		if entries[i].Timestamp.IsZero() {
			entries[i].Timestamp = now
		}
		if entries[i].ID == "" {
			entries[i].ID = generateID()
		}
		e.logChan <- &entries[i]
		count++
	}

	c.JSON(200, gin.H{"status": "ok", "count": count})
}

func (e *LogAggregationEngine) queryLogsHandler(c *gin.Context) {
	var query LogQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if query.Limit == 0 {
		query.Limit = 100
	}

	if query.StartTime.IsZero() {
		query.StartTime = time.Now().Add(-time.Hour)
	}
	if query.EndTime.IsZero() {
		query.EndTime = time.Now()
	}

	c.JSON(200, gin.H{"total": 100, "logs": []LogEntry{}})
}

func (e *LogAggregationEngine) getStatsHandler(c *gin.Context) {
	e.statsMu.RLock()
	stats := e.stats
	e.statsMu.RUnlock()

	c.JSON(200, stats)
}

func (e *LogAggregationEngine) getAggregateHandler(c *gin.Context) {
	service := c.Query("service")

	e.statsMu.RLock()
	defer e.statsMu.RUnlock()

	aggregations := []LogAggregation{}

	if service != "" {
		for level, count := range e.stats.ByLevel {
			percentage := 0.0
			if e.stats.TotalLogs > 0 {
				percentage = float64(count) / float64(e.stats.TotalLogs) * 100
			}
			aggregations = append(aggregations, LogAggregation{
				Service: service, Level: level, Count: count,
				Percentage: math.Round(percentage*100) / 100,
			})
		}
	}

	sort.Slice(aggregations, func(i, j int) bool {
		return aggregations[i].Count > aggregations[j].Count
	})

	c.JSON(200, gin.H{"total_logs": e.stats.TotalLogs, "aggregations": aggregations})
}

func (e *LogAggregationEngine) getErrorsHandler(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	e.statsMu.RLock()
	errors := make([]ErrorSummary, len(e.stats.Errors))
	copy(errors, e.stats.Errors)
	e.statsMu.RUnlock()

	sort.Slice(errors, func(i, j int) bool {
		return errors[i].Count > errors[j].Count
	})

	if len(errors) > limit {
		errors = errors[:limit]
	}

	c.JSON(200, gin.H{"errors": errors})
}

func (e *LogAggregationEngine) getTraceHandler(c *gin.Context) {
	traceID := c.Param("trace_id")

	c.JSON(200, gin.H{"trace_id": traceID, "logs": []LogEntry{}})
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateID() string {
	return fmt.Sprintf("%d_%s", time.Now().UnixNano(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
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
	fmt.Println("TigerWallet Log Aggregation Service")
	fmt.Println("============================================")

	engine := NewLogAggregationEngine()

	if err := engine.Start(); err != nil {
		fmt.Printf("Failed to start log engine: %v\n", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	engine.Stop()

	fmt.Println("Log aggregation service stopped")
}
