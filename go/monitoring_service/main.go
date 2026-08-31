package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
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
	ListenAddr     string
	PrometheusPort string
	CheckInterval  time.Duration
	AlertThreshold float64
	EnableAlerts   bool
	MaxMetrics     int
}

var config = Config{
	ListenAddr:     getEnv("MONITOR_LISTEN_ADDR", ":9090"),
	PrometheusPort: getEnv("PROMETHEUS_PORT", ":9091"),
	CheckInterval:  time.Second * 15,
	AlertThreshold: 0.8,
	EnableAlerts:   true,
	MaxMetrics:     1000,
}

// ============================================================================
// Metrics Models
// ============================================================================

type Metric struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"` // gauge, counter, histogram
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Help      string            `json:"help,omitempty"`
	Timestamp int64             `json:"timestamp"`
}

type MetricSeries struct {
	Name   string
	Type   string
	Points []MetricPoint
	Labels map[string]string
}

type MetricPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

type SystemMetrics struct {
	CPU       CPUStats       `json:"cpu"`
	Memory    MemoryStats    `json:"memory"`
	Disk      DiskStats      `json:"disk"`
	Network   NetworkStats   `json:"network"`
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
	Total        uint64  `json:"total"`
	Available    uint64  `json:"available"`
	Used         uint64  `json:"used"`
	Free         uint64  `json:"free"`
	UsagePercent float64 `json:"usage_percent"`
}

type DiskStats struct {
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Free         uint64  `json:"free"`
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
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpu_percent"`
	MemoryMB   float64 `json:"memory_mb"`
	Status     string  `json:"status"`
	NumThreads int     `json:"num_threads"`
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
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Severity     string    `json:"severity"` // critical, warning, info
	Message      string    `json:"message"`
	Service      string    `json:"service,omitempty"`
	MetricName   string    `json:"metric_name,omitempty"`
	Threshold    float64   `json:"threshold"`
	CurrentValue float64   `json:"current_value"`
	Timestamp    time.Time `json:"timestamp"`
	Resolved     bool      `json:"resolved"`
}

type DashboardData struct {
	Timestamp time.Time       `json:"timestamp"`
	System    SystemMetrics   `json:"system"`
	Services  []ServiceHealth `json:"services"`
	Alerts    []Alert         `json:"alerts"`
	Metrics   []Metric        `json:"metrics"`
}

// ============================================================================
// Monitoring Engine
// ============================================================================

type MonitoringEngine struct {
	metrics    map[string]*MetricSeries
	metricsMu  sync.RWMutex
	services   map[string]*ServiceHealth
	servicesMu sync.RWMutex
	alerts     []Alert
	alertsMu   sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	startTime  time.Time

	// Counters
	totalRequests int64
	totalErrors   int64
}

func NewMonitoringEngine() *MonitoringEngine {
	ctx, cancel := context.WithCancel(context.Background())

	return &MonitoringEngine{
		metrics:   make(map[string]*MetricSeries),
		services:  make(map[string]*ServiceHealth),
		ctx:       ctx,
		cancel:    cancel,
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

	// memUsed is the real Go heap allocation, not an estimate.
	memUsed := float64(stats.Alloc)
	e.recordMetric("system_memory_used", memUsed, nil)

	// Read the real system MemTotal from /proc/meminfo (Linux). On non-Linux
	// or parse failure we omit total/usage metrics rather than estimating.
	memTotal, ok := readProcMemTotal()
	if !ok {
		return
	}
	e.recordMetric("system_memory_total", memTotal, nil)
	if memTotal > 0 {
		e.recordMetric("system_memory_usage_percent", (memUsed/memTotal)*100, nil)
	}
}

// readProcMemTotal parses the "MemTotal:" line of /proc/meminfo (kB) and
// returns the value in bytes. It returns (0, false) on non-Linux systems or
// any parse failure so callers can omit the metric instead of guessing.
func readProcMemTotal() (float64, bool) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		// ["MemTotal:", "16384000", "kB"]
		if len(fields) < 2 {
			return 0, false
		}
		kb, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return 0, false
		}
		return kb * 1024, true
	}
	return 0, false
}

func (e *MonitoringEngine) collectDisk() {
	// Real disk stats via syscall.Statfs on the root filesystem.
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		return
	}
	diskTotal := float64(st.Blocks * uint64(st.Bsize))
	diskFree := float64(st.Bavail * uint64(st.Bsize))
	diskUsed := diskTotal - diskFree
	diskUsage := 0.0
	if diskTotal > 0 {
		diskUsage = (diskUsed / diskTotal) * 100
	}

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
	// Real network stats from /proc/net/dev (sum of all non-loopback interfaces).
	// Fail-closed: records nothing when the counters cannot be read.
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return
	}
	var sent, recv uint64
	for _, line := range strings.Split(string(data), "\n") {
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		iface := strings.TrimSpace(line[:i])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(line[i+1:])
		if len(fields) < 9 {
			continue
		}
		r, _ := strconv.ParseUint(fields[0], 10, 64)
		t, _ := strconv.ParseUint(fields[8], 10, 64)
		recv += r
		sent += t
	}
	e.recordMetric("system_network_bytes_sent", float64(sent), nil)
	e.recordMetric("system_network_bytes_recv", float64(recv), nil)
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
	// Real health probe: HTTP GET to the service's health endpoint, measuring
	// real latency and recording real success/failure. Fail-closed: a service
	// that does not respond is marked down (never a fabricated healthy).
	e.servicesMu.Lock()
	health, exists := e.services[name]
	if !exists {
		health = &ServiceHealth{
			Name:      name,
			Status:    "unknown",
			Uptime:    0,
			LastCheck: time.Now(),
		}
		e.services[name] = health
	}
	e.servicesMu.Unlock()

	url := serviceHealthURL(name)
	start := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	latency := float64(time.Since(start).Microseconds()) / 1000.0

	e.servicesMu.Lock()
	defer e.servicesMu.Unlock()
	health.Latency = latency
	health.LastCheck = time.Now()
	health.Requests++
	if err != nil || resp == nil {
		health.Errors++
		health.Status = "down"
	} else {
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			health.Status = "healthy"
		} else {
			health.Errors++
			health.Status = "degraded"
		}
	}

	if health.Requests > 0 {
		health.ErrorRate = float64(health.Errors) / float64(health.Requests)
		health.SuccessRate = 1.0 - health.ErrorRate
	}

	if health.ErrorRate > config.AlertThreshold {
		e.createAlert("HighErrorRate", "warning", fmt.Sprintf("Service %s has high error rate", name),
			name, "service_error_rate", config.AlertThreshold, health.ErrorRate)
	}

	health.Uptime = time.Since(e.startTime).Seconds()
}

// serviceHealthURL resolves the health endpoint for a named service. It reads
// SERVICE_<NAME>_HEALTH_URL env var, falling back to a conventional localhost
// URL. Returns empty string (skip probe) when unresolvable.
func serviceHealthURL(name string) string {
	key := "SERVICE_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_HEALTH_URL"
	if v := os.Getenv(key); v != "" {
		return v
	}
	// conventional docker-compose service DNS name
	return "http://" + name + "/health"
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
		Timestamp:    time.Now(),
		Resolved:     false,
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
		Status  string  `json:"status"`
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
		Services:  services,
		Alerts:    alerts,
		Metrics:   metrics,
	}

	c.JSON(200, dashboard)
}

func (e *MonitoringEngine) getSystemMetricsHandler(c *gin.Context) {
	load1, load5, load15 := getLoadAverage()
	cpu := getCPUUsage()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memTotal := 0.0
	if t, ok := readProcMemTotal(); ok {
		memTotal = t
	}
	memUsed := float64(memStats.Alloc)
	memPct := 0.0
	if memTotal > 0 {
		memPct = (memUsed / memTotal) * 100
	}

	var st syscall.Statfs_t
	var diskTotal, diskFree float64
	if err := syscall.Statfs("/", &st); err == nil {
		diskTotal = float64(st.Blocks * uint64(st.Bsize))
		diskFree = float64(st.Bavail * uint64(st.Bsize))
	}
	diskUsed := diskTotal - diskFree
	diskPct := 0.0
	if diskTotal > 0 {
		diskPct = (diskUsed / diskTotal) * 100
	}

	system := SystemMetrics{
		CPU: CPUStats{
			Usage:        cpu,
			Count:        runtime.NumCPU(),
			LoadAvg1Min:  load1,
			LoadAvg5Min:  load5,
			LoadAvg15Min: load15,
		},
		Memory: MemoryStats{
			Total:        uint64(memTotal),
			Available:    uint64(memFree()),
			Used:         uint64(memUsed),
			Free:         uint64(memFree()),
			UsagePercent: memPct,
		},
		Disk: DiskStats{
			Total:        uint64(diskTotal),
			Used:         uint64(diskUsed),
			Free:         uint64(diskFree),
			UsagePercent: diskPct,
		},
	}

	c.JSON(200, system)
}

// memFree returns MemAvailable from /proc/meminfo in bytes (0 when unavailable).
func memFree() float64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "MemAvailable:") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 2 {
			return 0
		}
		kb, err := strconv.ParseFloat(f[1], 64)
		if err != nil {
			return 0
		}
		return kb * 1024
	}
	return 0
}

// ============================================================================
// Helper Functions
// ============================================================================

var lastCPUSample struct {
	total, idle uint64
}

// getCPUUsage returns real CPU usage % from /proc/stat (delta between samples).
// First sample returns 0 (needs two reads for a delta). Fail-closed: 0 on error.
func getCPUUsage() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 5 {
			return 0
		}
		var total, idle uint64
		for i := 1; i < len(f); i++ {
			v, err := strconv.ParseUint(f[i], 10, 64)
			if err != nil {
				return 0
			}
			total += v
			if i == 4 { // idle
				idle = v
			}
		}
		usage := 0.0
		if lastCPUSample.total > 0 && total > lastCPUSample.total {
			dTotal := total - lastCPUSample.total
			dIdle := idle - lastCPUSample.idle
			if dTotal > 0 {
				usage = float64(dTotal-dIdle) / float64(dTotal) * 100
			}
		}
		lastCPUSample.total = total
		lastCPUSample.idle = idle
		return usage
	}
	return 0
}

// getLoadAverage returns the real 1/5/15-min load averages from /proc/loadavg.
// Fail-closed: returns zeros when unavailable.
func getLoadAverage() (float64, float64, float64) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0
	}
	f := strings.Fields(string(data))
	if len(f) < 3 {
		return 0, 0, 0
	}
	l1, _ := strconv.ParseFloat(f[0], 64)
	l5, _ := strconv.ParseFloat(f[1], 64)
	l15, _ := strconv.ParseFloat(f[2], 64)
	return l1, l5, l15
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
