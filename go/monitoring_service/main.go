package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
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
	ListenAddr      string
	PrometheusPort  string
	CheckInterval   time.Duration
	AlertThreshold  float64
	EnableAlerts   bool
	MaxMetrics     int
}

var config = Config{
	ListenAddr:     getEnv("MONITOR_LISTEN_ADDR", ":9090"),
	PrometheusPort: getEnv("PROMETHEUS_PORT", ":9091"),
	CheckInterval:  time.Second * 15,
	AlertThreshold:  0.8,
	EnableAlerts:   true,
	MaxMetrics:     1000,
}

// ============================================================================
// Metrics Models
// ============================================================================

type Metric struct {
	Name      string                 `json:"name"`
	Type     string                 `json:"type"` // gauge, counter, histogram
	Value    float64                `json:"value"`
	Labels   map[string]string      `json:"labels,omitempty"`
	Help     string                `json:"help,omitempty"`
	Timestamp int64                `json:"timestamp"`
}

type MetricSeries struct {
	Name      string
	Type     string
	Points   []MetricPoint
	Labels   map[string]string
}

type MetricPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

type SystemMetrics struct {
	CPU       CPUStats    `json:"cpu"`
	Memory    MemoryStats `json:"memory"`
	Disk      DiskStats   `json:"disk"`
	Network   NetworkStats `json:"network"`
	Processes []ProcessStats `json:"processes,omitempty"`
}

type CPUStats struct {
	Usage        float64 `json:"usage"`
	Count        int     `json:"count"`
	LoadAvg1Min  float64 `json:"load_avg_1min"`
	LoadAvg5Min  float64 `json:"load_avg_5min"`
	LoadAvg15Min float64 `json:"load_avg_15min"`
}

type MemoryStats struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsagePercent float64 `json:"usage_percent"`
}

type DiskStats struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsagePercent float64 `json:"usage_percent"`
	InodesTotal  uint64  `json:"inodes_total"`
	InodesUsed   uint64  `json:"inodes_used"`
}

type NetworkStats struct {
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	ErrIn       uint64 `json:"err_in"`
	ErrOut      uint64 `json:"err_out"`
}

type ProcessStats struct {
	PID         int     `json:"pid"`
	Name        string  `json:"name"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryMB    float64 `json:"memory_mb"`
	Status      string  `json:"status"`
	NumThreads  int     `json:"num_threads"`
}

type ServiceHealth struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"` // healthy, degraded, down
	Uptime      float64   `json:"uptime"`
	Latency     float64   `json:"latency"` // ms
	Requests    int64     `json:"requests"`
	Errors      int64     `json:"errors"`
	LastCheck   time.Time `json:"last_check"`
	ErrorRate   float64   `json:"error_rate"`
	SuccessRate float64   `json:"success_rate"`
}

type Alert struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Severity    string    `json:"severity"` // critical, warning, info
	Message     string    `json:"message"`
	Service     string    `json:"service,omitempty"`
	MetricName  string    `json:"metric_name,omitempty"`
	Threshold   float64   `json:"threshold"`
	CurrentValue float64 `json:"current_value"`
	Timestamp   time.Time `json:"timestamp"`
	Resolved    bool      `json:"resolved"`
}

type DashboardData struct {
	Timestamp    time.Time       `json:"timestamp"`
	System       SystemMetrics   `json:"system"`
	Services     []ServiceHealth `json:"services"`
	Alerts       []Alert        `json:"alerts"`
	Metrics      []Metric       `json:"metrics"`
}

// ============================================================================
// Monitoring Engine
// ============================================================================

type MonitoringEngine struct {
	metrics   map[string]*MetricSeries
	metricsMu sync.RWMutex
	services  map[string]*ServiceHealth
	servicesMu sync.RWMutex
	alerts    []Alert
	alertsMu  sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	startTime time.Time
	
	// Counters
	totalRequests  int64
	totalErrors   int64
}

func NewMonitoringEngine() *MonitoringEngine {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MonitoringEngine{
		metrics:  make(map[string]*MetricSeries),
		services: make(map[string]*ServiceHealth),
		ctx:      ctx,
		cancel:   cancel,
		startTime: time.Now(),
	}
}

func (e *MonitoringEngine) Start() error {
	fmt.Println("Starting Monitoring Engine...")
	
	// Initialize system metrics collection
	go e.collectSystemMetrics()
	
	// Initialize service health checks
	go e.checkServiceHealth()
	
	// Start HTTP server
	go e.startHTTPServer()
	
	fmt.Println("Monitoring Engine started successfully")
	return nil
}

func (e *MonitoringEngine) Stop() {
	fmt.Println("Stopping Monitoring Engine...")
	e.cancel()
	fmt.Println("Monitoring Engine stopped")
}

func (e *MonitoringEngine) startHTTPServer() {
	router := gin.Default()
	
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})
	
	router.GET("/metrics", e.getMetricsHandler)
	router.GET("/metrics/:name", e.getMetricHandler)
	router.POST("/metrics", e.postMetricHandler)
	
	router.GET("/services", e.getServicesHandler)
	router.GET("/services/:name", e.getServiceHandler)
	router.POST("/services/:name/heartbeat", e.serviceHeartbeatHandler)
	
	router.GET("/alerts", e.getAlertsHandler)
	router.GET("/alerts/:id", e.getAlertHandler)
	router.POST("/alerts/:id/resolve", e.resolveAlertHandler)
	
	router.GET("/dashboard", e.getDashboardHandler)
	
	router.GET("/system", e.getSystemMetricsHandler)
	
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"ready": true})
	})
	
	fmt.Printf("Monitoring API server starting on %s\n", config.ListenAddr)
	router.Run(config.ListenAddr)
}

// ============================================================================
// Metrics Collection
// ============================================================================

func (e *MonitoringEngine) collectSystemMetrics() {
	ticker := time.NewTicker(config.CheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.collectCPU()
			e.collectMemory()
			e.collectDisk()
			e.collectNetwork()
		}
	}
}

func (e *MonitoringEngine) collectCPU() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	
	// Get CPU usage (simplified)
	cpuUsage := getCPUUsage()
	
	e.recordMetric("system_cpu_usage", cpuUsage, nil)
	
	// Get load average
	load1, load5, load15 := getLoadAverage()
	e.recordMetric("system_load_avg_1min", load1, nil)
	e.recordMetric("system_load_avg_5min", load5, nil)
	e.recordMetric("system_load_avg_15min", load15, nil)
	
	// CPU count
	e.recordMetric("system_cpu_count", float64(runtime.NumCPU()), nil)
}

func (e *MonitoringEngine) collectMemory() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	
	// Simplified memory stats
	memTotal := float64(stats.TotalAlloc) * 2 // Estimate
	memUsed := float64(stats.Alloc)
	memUsage := (memUsed / memTotal) * 100
	
	e.recordMetric("system_memory_total", memTotal, nil)
	e.recordMetric("system_memory_used", memUsed, nil)
	e.recordMetric("system_memory_usage_percent", memUsage, nil)
}

func (e *MonitoringEngine) collectDisk() {
	// Simplified disk stats
	diskTotal := float64(500 * 1024 * 1024 * 1024) // 500GB
	diskUsed := float64(250 * 1024 * 1024 * 1024) // 250GB
	diskUsage := (diskUsed / diskTotal) * 100
	
	e.recordMetric("system_disk_total", diskTotal, nil)
	e.recordMetric("system_disk_used", diskUsed, nil)
	e.recordMetric("system_disk_usage_percent", diskUsage, nil)
	
	// Check for high disk usage
	if diskUsage > config.AlertThreshold*100 {
		e.createAlert("HighDiskUsage", "warning", "Disk usage is high",
			"", "system_disk_usage_percent", config.AlertThreshold*100, diskUsage)
	}
}

func (e *MonitoringEngine) collectNetwork() {
	// Simplified network stats
	e.recordMetric("system_network_bytes_sent", float64(rand.Int63n(1000000000)), nil)
	e.recordMetric("system_network_bytes_recv", float64(rand.Int63n(1000000000)), nil)
}

func (e *MonitoringEngine) recordMetric(name string, value float64, labels map[string]string) {
	e.metricsMu.Lock()
	defer e.metricsMu.Unlock()
	
	if series, ok := e.metrics[name]; ok {
		series.Points = append(series.Points, MetricPoint{
			Timestamp: time.Now().Unix(),
			Value:     value,
		})
		
		// Keep only last 1000 points
		if len(series.Points) > config.MaxMetrics {
			series.Points = series.Points[len(series.Points)-config.MaxMetrics:]
		}
	} else {
		e.metrics[name] = &MetricSeries{
			Name:   name,
			Type:   "gauge",
			Points: []MetricPoint{{Timestamp: time.Now().Unix(), Value: value}},
			Labels: labels,
		}
	}
}

// ============================================================================
// Service Health
// ============================================================================

func (e *MonitoringEngine) checkServiceHealth() {
	ticker := time.NewTicker(config.CheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.checkAllServices()
		}
	}
}

func (e *MonitoringEngine) checkAllServices() {
	services := []string{
		"wallet_service",
		"swap_service",
		"staking_service",
		"nft_service",
		"bridge_service",
		"analytics_service",
		"log_service",
		"notification_service",
	}
	
	for _, svc := range services {
		e.checkService(svc)
	}
}

func (e *MonitoringEngine) checkService(name string) {
	e.servicesMu.Lock()
	defer e.servicesMu.Unlock()
	
	health, exists := e.services[name]
	if !exists {
		health = &ServiceHealth{
			Name:      name,
			Status:    "healthy",
			Uptime:    0,
			LastCheck: time.Now(),
		}
		e.services[name] = health
	}
	
	// Simulate health check
	latency := rand.Float64() * 100 // 0-100ms
	errorRate := rand.Float64() * 0.05 // 0-5%
	
	health.Latency = latency
	health.LastCheck = time.Now()
	health.Requests++
	
	if rand.Float64() < errorRate {
		health.Errors++
		health.Status = "degraded"
	} else {
		health.Status = "healthy"
	}
	
	health.ErrorRate = float64(health.Errors) / float64(health.Requests)
	health.SuccessRate = 1.0 - health.ErrorRate
	
	if health.ErrorRate > config.AlertThreshold {
		e.createAlert("HighErrorRate", "warning", fmt.Sprintf("Service %s has high error rate", name),
			name, "service_error_rate", config.AlertThreshold, health.ErrorRate)
	}
	
	health.Uptime = time.Since(e.startTime).Seconds()
}

// ============================================================================
// Alerts
// ============================================================================

func (e *MonitoringEngine) createAlert(name, severity, message, service, metricName string, threshold, currentValue float64) {
	e.alertsMu.Lock()
	defer e.alertsMu.Unlock()
	
	// Check if alert already exists
	for i := range e.alerts {
		if !e.alerts[i].Resolved && e.alerts[i].Name == name {
			e.alerts[i].CurrentValue = currentValue
			e.alerts[i].Timestamp = time.Now()
			return
		}
	}
	
	alert := Alert{
		ID:           fmt.Sprintf("alert_%d", time.Now().Unix()),
		Name:         name,
		Severity:     severity,
		Message:      message,
		Service:      service,
		MetricName:   metricName,
		Threshold:    threshold,
		CurrentValue: currentValue,
		Timestamp:   time.Now(),
		Resolved:    false,
	}
	
	e.alerts = append(e.alerts, alert)
	
	// Keep only last 100 alerts
	if len(e.alerts) > 100 {
		e.alerts = e.alerts[len(e.alerts)-100:]
	}
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (e *MonitoringEngine) getMetricsHandler(c *gin.Context) {
	e.metricsMu.RLock()
	metrics := make([]Metric, 0)
	
	for _, series := range e.metrics {
		if len(series.Points) > 0 {
			latest := series.Points[len(series.Points)-1]
			metrics = append(metrics, Metric{
				Name:      series.Name,
				Type:      series.Type,
				Value:     latest.Value,
				Labels:    series.Labels,
				Timestamp: latest.Timestamp,
			})
		}
	}
	e.metricsMu.RUnlock()
	
	c.JSON(200, metrics)
}

func (e *MonitoringEngine) getMetricHandler(c *gin.Context) {
	name := c.Param("name")
	
	e.metricsMu.RLock()
	series, ok := e.metrics[name]
	e.metricsMu.RUnlock()
	
	if !ok {
		c.JSON(404, gin.H{"error": "metric not found"})
		return
	}
	
	c.JSON(200, series)
}

func (e *MonitoringEngine) postMetricHandler(c *gin.Context) {
	var metric Metric
	if err := c.ShouldBindJSON(&metric); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	if metric.Timestamp == 0 {
		metric.Timestamp = time.Now().Unix()
	}
	
	e.recordMetric(metric.Name, metric.Value, metric.Labels)
	
	c.JSON(200, gin.H{"status": "ok"})
}

func (e *MonitoringEngine) getServicesHandler(c *gin.Context) {
	e.servicesMu.RLock()
	services := make([]ServiceHealth, 0, len(e.services))
	for _, s := range e.services {
		services = append(services, *s)
	}
	e.servicesMu.RUnlock()
	
	c.JSON(200, services)
}

func (e *MonitoringEngine) getServiceHandler(c *gin.Context) {
	name := c.Param("name")
	
	e.servicesMu.RLock()
	service, ok := e.services[name]
	e.servicesMu.RUnlock()
	
	if !ok {
		c.JSON(404, gin.H{"error": "service not found"})
		return
	}
	
	c.JSON(200, service)
}

func (e *MonitoringEngine) serviceHeartbeatHandler(c *gin.Context) {
	name := c.Param("name")
	
	var payload struct {
		Status string  `json:"status"`
		Latency float64 `json:"latency"`
	}
	c.ShouldBindJSON(&payload)
	
	e.servicesMu.Lock()
	health := e.services[name]
	if health == nil {
		health = &ServiceHealth{Name: name}
		e.services[name] = health
	}
	
	health.Status = payload.Status
	health.Latency = payload.Latency
	health.LastCheck = time.Now()
	health.Requests++
	
	e.servicesMu.Unlock()
	
	c.JSON(200, gin.H{"status": "ok"})
}

func (e *MonitoringEngine) getAlertsHandler(c *gin.Context) {
	e.alertsMu.RLock()
	alerts := make([]Alert, len(e.alerts))
	copy(alerts, e.alerts)
	e.alertsMu.RUnlock()
	
	c.JSON(200, alerts)
}

func (e *MonitoringEngine) getAlertHandler(c *gin.Context) {
	id := c.Param("id")
	
	e.alertsMu.RLock()
	defer e.alertsMu.RUnlock()
	
	for _, alert := range e.alerts {
		if alert.ID == id {
			c.JSON(200, alert)
			return
		}
	}
	
	c.JSON(404, gin.H{"error": "alert not found"})
}

func (e *MonitoringEngine) resolveAlertHandler(c *gin.Context) {
	id := c.Param("id")
	
	e.alertsMu.Lock()
	defer e.alertsMu.Unlock()
	
	for i := range e.alerts {
		if e.alerts[i].ID == id {
			e.alerts[i].Resolved = true
			c.JSON(200, gin.H{"status": "ok"})
			return
		}
	}
	
	c.JSON(404, gin.H{"error": "alert not found"})
}

func (e *MonitoringEngine) getDashboardHandler(c *gin.Context) {
	e.metricsMu.RLock()
	metrics := make([]Metric, 0)
	for _, series := range e.metrics {
		if len(series.Points) > 0 {
			latest := series.Points[len(series.Points)-1]
			metrics = append(metrics, Metric{
				Name: series.Name, Type: series.Type, Value: latest.Value,
				Timestamp: latest.Timestamp,
			})
		}
	}
	e.metricsMu.RUnlock()
	
	e.servicesMu.RLock()
	services := make([]ServiceHealth, 0, len(e.services))
	for _, s := range e.services {
		services = append(services, *s)
	}
	e.servicesMu.RUnlock()
	
	e.alertsMu.RLock()
	alerts := make([]Alert, 0)
	for _, a := range e.alerts {
		if !a.Resolved {
			alerts = append(alerts, a)
		}
	}
	e.alertsMu.RUnlock()
	
	dashboard := DashboardData{
		Timestamp: time.Now(),
		Services:   services,
		Alerts:     alerts,
		Metrics:    metrics,
	}
	
	c.JSON(200, dashboard)
}

func (e *MonitoringEngine) getSystemMetricsHandler(c *gin.Context) {
	system := SystemMetrics{
		CPU: CPUStats{
			Usage:       rand.Float64() * 100,
			Count:       runtime.NumCPU(),
			LoadAvg1Min: rand.Float64() * 2,
			LoadAvg5Min: rand.Float64() * 2,
			LoadAvg15Min: rand.Float64() * 2,
		},
		Memory: MemoryStats{
			Total:        16 * 1024 * 1024 * 1024,
			Available:    8 * 1024 * 1024 * 1024,
			Used:         8 * 1024 * 1024 * 1024,
			Free:         8 * 1024 * 1024 * 1024,
			UsagePercent: 50.0,
		},
		Disk: DiskStats{
			Total:        500 * 1024 * 1024 * 1024,
			Used:         250 * 1024 * 1024 * 1024,
			Free:         250 * 1024 * 1024 * 1024,
			UsagePercent: 50.0,
		},
	}
	
	c.JSON(200, system)
}

// ============================================================================
// Helper Functions
// ============================================================================

func getCPUUsage() float64 {
	return rand.Float64() * 100
}

func getLoadAverage() (float64, float64, float64) {
	return rand.Float64() * 2, rand.Float64() * 2, rand.Float64() * 2
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
	fmt.Println("TigerWallet Monitoring Service")
	fmt.Println("============================================")
	
	engine := NewMonitoringEngine()
	
	if err := engine.Start(); err != nil {
		fmt.Printf("Failed to start monitoring engine: %v\n", err)
		os.Exit(1)
	}
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	fmt.Println("\nShutting down...")
	engine.Stop()
	
	fmt.Println("Monitoring service stopped")
}
