/**
 * TigerWallet Monitoring Service
 * 
 * Complete monitoring and alerting with Prometheus metrics,
 * Grafana integration, and health checks.
 * Built with Go for high-load distributed operations.
 */

package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Metrics Types
// ============================================================================

// Metric represents a Prometheus metric
type Metric struct {
	Name      string            `json:"name"`
	Help      string            `json:"help"`
	Type      string            `json:"type"` // counter, gauge, histogram, summary
	Labels    map[string]string `json:"labels"`
	Value     float64           `json:"value"`
	Timestamp int64            `json:"timestamp"`
}

// Counter is a Prometheus counter metric
type Counter struct {
	Name   string            `json:"name"`
	Help   string            `json:"help"`
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
}

// Gauge is a Prometheus gauge metric
type Gauge struct {
	Name   string            `json:"name"`
	Help   string            `json:"help"`
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
}

// Histogram is a Prometheus histogram metric
type Histogram struct {
	Name        string            `json:"name"`
	Help        string            `json:"help"`
	Labels      map[string]string `json:"labels"`
	Count       uint64            `json:"count"`
	Sum         float64           `json:"sum"`
	Buckets     map[float64]uint64 `json:"buckets"`
}

// ServiceHealth represents service health status
type ServiceHealth struct {
	ServiceName  string        `json:"service_name"`
	Status       string        `json:"status"` // healthy, degraded, unhealthy
	Uptime       time.Duration `json:"uptime"`
	Latency      time.Duration `json:"latency"`
	Requests     uint64        `json:"requests"`
	Errors       uint64        `json:"errors"`
	LastCheck   time.Time     `json:"last_check"`
	Dependencies []Dependency  `json:"dependencies"`
}

// Dependency represents a service dependency
type Dependency struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Latency time.Duration `json:"latency"`
}

// Alert represents an alert
type Alert struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Severity    string    `json:"severity"` // critical, warning, info
	Message     string    `json:"message"`
	Service    string    `json:"service"`
	MetricName string    `json:"metric_name"`
	Value      float64   `json:"value"`
	Threshold  float64   `json:"threshold"`
	Status     string    `json:"status"` // firing, resolved
	FiredAt    time.Time `json:"fired_at"`
	ResolvedAt time.Time `json:"resolved_at"`
}

// DashboardData represents dashboard data
type DashboardData struct {
	Metrics    []Metric         `json:"metrics"`
	Services   []ServiceHealth  `json:"services"`
	Alerts     []Alert          `json:"alerts"`
	Uptime     float64         `json:"uptime"`
	TotalRequests uint64        `json:"total_requests"`
	TotalErrors uint64         `json:"total_errors"`
}

// ============================================================================
// Monitoring Service
// ============================================================================

// MonitoringService manages monitoring and metrics
type MonitoringService struct {
	mu           sync.RWMutex
	counters     map[string]*Counter
	gauges       map[string]*Gauge
	histograms   map[string]*Histogram
	services     map[string]*ServiceHealth
	alerts       map[string]*Alert
	startTime   time.Time
	httpServer  *http.Server
}

// Service singleton
var (
	monitoringService     *MonitoringService
	monitoringServiceOnce sync.Once
)

// GetMonitoringService returns the singleton monitoring service
func GetMonitoringService() *MonitoringService {
	monitoringServiceOnce.Do(func() {
		monitoringService = &MonitoringService{
			counters:   make(map[string]*Counter),
			gauges:    make(map[string]*Gauge),
			histograms: make(map[string]*Histogram),
			services:   make(map[string]*ServiceHealth),
			alerts:     make(map[string]*Alert),
			startTime:  time.Now(),
		}
	})
	return monitoringService
}

// ============================================================================
// Counter Methods
// ============================================================================

// IncCounter increments a counter
func (s *MonitoringService) IncCounter(name string, labels map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(name, labels)
	counter, exists := s.counters[key]
	if !exists {
		counter = &Counter{
			Name:   name,
			Labels: labels,
			Value:  0,
		}
		s.counters[key] = counter
	}
	counter.Value++
}

// AddCounter adds value to a counter
func (s *MonitoringService) AddCounter(name string, labels map[string]string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(name, labels)
	counter, exists := s.counters[key]
	if !exists {
		counter = &Counter{
			Name:   name,
			Labels: labels,
			Value:  0,
		}
		s.counters[key] = counter
	}
	counter.Value += value
}

// GetCounter returns counter value
func (s *MonitoringService) GetCounter(name string, labels map[string]string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := makeKey(name, labels)
	if counter, exists := s.counters[key]; exists {
		return counter.Value
	}
	return 0
}

// ============================================================================
// Gauge Methods
// ============================================================================

// SetGauge sets gauge value
func (s *MonitoringService) SetGauge(name string, labels map[string]string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(name, labels)
	gauge, exists := s.gauges[key]
	if !exists {
		gauge = &Gauge{
			Name:   name,
			Labels: labels,
			Value:  0,
		}
		s.gauges[key] = gauge
	}
	gauge.Value = value
}

// IncGauge increments gauge
func (s *MonitoringService) IncGauge(name string, labels map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(name, labels)
	gauge, exists := s.gauges[key]
	if !exists {
		gauge = &Gauge{
			Name:   name,
			Labels: labels,
			Value:  0,
		}
		s.gauges[key] = gauge
	}
	gauge.Value++
}

// DecGauge decrements gauge
func (s *MonitoringService) DecGauge(name string, labels map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(name, labels)
	gauge, exists := s.gauges[key]
	if !exists {
		gauge = &Gauge{
			Name:   name,
			Labels: labels,
			Value:  0,
		}
		s.gauges[key] = gauge
	}
	gauge.Value--
}

// GetGauge returns gauge value
func (s *MonitoringService) GetGauge(name string, labels map[string]string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := makeKey(name, labels)
	if gauge, exists := s.gauges[key]; exists {
		return gauge.Value
	}
	return 0
}

// ============================================================================
// Histogram Methods
// ============================================================================

// ObserveHistogram observes a histogram value
func (s *MonitoringService) ObserveHistogram(name string, labels map[string]string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(name, labels)
	hist, exists := s.histograms[key]
	if !exists {
		hist = &Histogram{
			Name:    name,
			Labels:  labels,
			Count:   0,
			Sum:     0,
			Buckets: make(map[float64]uint64),
		}
		// Initialize default buckets
		hist.Buckets[0.005] = 0
		hist.Buckets[0.01] = 0
		hist.Buckets[0.025] = 0
		hist.Buckets[0.05] = 0
		hist.Buckets[0.1] = 0
		hist.Buckets[0.25] = 0
		hist.Buckets[0.5] = 0
		hist.Buckets[1.0] = 0
		hist.Buckets[2.5] = 0
		hist.Buckets[5.0] = 0
		hist.Buckets[10.0] = 0
		s.histograms[key] = hist
	}

	hist.Count++
	hist.Sum += value

	// Update bucket counts
	for bucket := range hist.Buckets {
		if value <= bucket {
			hist.Buckets[bucket]++
		}
	}
}

// ============================================================================
// Service Health
// ============================================================================

// RegisterService registers a service for monitoring
func (s *MonitoringService) RegisterService(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.services[name] = &ServiceHealth{
		ServiceName: name,
		Status:      "healthy",
		Uptime:      time.Since(s.startTime),
		LastCheck:   time.Now(),
	}
}

// UpdateServiceHealth updates service health status
func (s *MonitoringService) UpdateServiceHealth(name string, status string, latency time.Duration, requests, errors uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if service, exists := s.services[name]; exists {
		service.Status = status
		service.Latency = latency
		service.Requests = requests
		service.Errors = errors
		service.LastCheck = time.Now()
	}
}

// GetServiceHealth returns service health
func (s *MonitoringService) GetServiceHealth(name string) (*ServiceHealth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	service, exists := s.services[name]
	if !exists {
		return nil, fmt.Errorf("service not found")
	}
	return service, nil
}

// GetAllServiceHealth returns all service health
func (s *MonitoringService) GetAllServiceHealth() []ServiceHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ServiceHealth, 0, len(s.services))
	for _, service := range s.services {
		result = append(result, *service)
	}
	return result
}

// ============================================================================
// Alerts
// ============================================================================

// CreateAlert creates a new alert
func (s *MonitoringService) CreateAlert(name, severity, message, service, metricName string, value, threshold float64) *Alert {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert := &Alert{
		ID:          "alert_" + uuid.New().String(),
		Name:        name,
		Severity:    severity,
		Message:     message,
		Service:     service,
		MetricName:  metricName,
		Value:       value,
		Threshold:   threshold,
		Status:      "firing",
		FiredAt:     time.Now(),
	}

	s.alerts[alert.ID] = alert
	return alert
}

// ResolveAlert resolves an alert
func (s *MonitoringService) ResolveAlert(alertID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, exists := s.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found")
	}

	alert.Status = "resolved"
	alert.ResolvedAt = time.Now()
	return nil
}

// GetAlerts returns active alerts
func (s *MonitoringService) GetAlerts(status string) []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Alert, 0)
	for _, alert := range s.alerts {
		if status == "" || alert.Status == status {
			result = append(result, *alert)
		}
	}
	return result
}

// ============================================================================
// Dashboard Data
// ============================================================================

// GetDashboardData returns all dashboard data
func (s *MonitoringService) GetDashboardData() DashboardData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect metrics
	metrics := make([]Metric, 0)
	
	// Add counters
	for _, counter := range s.counters {
		metrics = append(metrics, Metric{
			Name:      counter.Name,
			Type:      "counter",
			Labels:    counter.Labels,
			Value:     counter.Value,
			Timestamp: time.Now().Unix(),
		})
	}
	
	// Add gauges
	for _, gauge := range s.gauges {
		metrics = append(metrics, Metric{
			Name:      gauge.Name,
			Type:      "gauge",
			Labels:    gauge.Labels,
			Value:     gauge.Value,
			Timestamp: time.Now().Unix(),
		})
	}
	
	// Collect services
	services := make([]ServiceHealth, 0)
	for _, service := range s.services {
		services = append(services, *service)
	}
	
	// Collect alerts
	alerts := make([]Alert, 0)
	for _, alert := range s.alerts {
		if alert.Status == "firing" {
			alerts = append(alerts, *alert)
		}
	}
	
	// Calculate totals
	var totalRequests uint64
	var totalErrors uint64
	for _, service := range s.services {
		totalRequests += service.Requests
		totalErrors += service.Errors
	}

	uptime := time.Since(s.startTime).Seconds() / 86400 // days

	return DashboardData{
		Metrics:         metrics,
		Services:       services,
		Alerts:         alerts,
		Uptime:         uptime,
		TotalRequests:  totalRequests,
		TotalErrors:    totalErrors,
	}
}

// ============================================================================
// Prometheus Metrics Export
// ============================================================================

// ExportPrometheus exports metrics in Prometheus format
func (s *MonitoringService) ExportPrometheus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var output string

	// Export counters
	for _, counter := range s.counters {
		labels := formatLabels(counter.Labels)
		output += fmt.Sprintf("# TYPE %s counter\n", counter.Name)
		output += fmt.Sprintf("%s%s %f\n", counter.Name, labels, counter.Value)
	}

	// Export gauges
	for _, gauge := range s.gauges {
		labels := formatLabels(gauge.Labels)
		output += fmt.Sprintf("# TYPE %s gauge\n", gauge.Name)
		output += fmt.Sprintf("%s%s %f\n", gauge.Name, labels, gauge.Value)
	}

	// Export histograms
	for _, hist := range s.histograms {
		labels := formatLabels(hist.Labels)
		output += fmt.Sprintf("# TYPE %s histogram\n", hist.Name)
		output += fmt.Sprintf("%s_count%s %d\n", hist.Name, labels, hist.Count)
		output += fmt.Sprintf("%s_sum%s %f\n", hist.Name, labels, hist.Sum)
		for bucket, count := range hist.Buckets {
			output += fmt.Sprintf("%s_bucket{le=\"%f\"}%s %d\n", hist.Name, bucket, labels, count)
		}
	}

	return output
}

// StartHTTPServer starts the metrics HTTP server
func (s *MonitoringService) StartHTTPServer(addr string) error {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(s.ExportPrometheus()))
	})
	
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.GetDashboardData())
	})
	
	mux.HandleFunc("/alerts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.GetAlerts("")))
	})
	
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return s.httpServer.ListenAndServe()
}

// StopHTTPServer stops the metrics HTTP server
func (s *MonitoringService) StopHTTPServer() error {
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func makeKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += fmt.Sprintf("%s=%s", k, v)
	}
	return key
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	
	result := "{"
	first := true
	for k, v := range labels {
		if !first {
			result += ","
		}
		result += fmt.Sprintf("%s=\"%s\"", k, v)
		first = false
	}
	result += "}"
	return result
}

// ============================================================================
// Default Service Metrics
// ============================================================================

// StartDefaultMetrics starts collecting default system metrics
func (s *MonitoringService) StartDefaultMetrics(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.collectDefaultMetrics()
			}
		}
	}()
}

func (s *MonitoringService) collectDefaultMetrics() {
	// Simulated metrics for demo - in production, collect real system metrics
	
	// HTTP request metrics
	s.SetGauge("http_requests_total", map[string]string{"method": "GET", "status": "200"}, float64(rand.Intn(10000)))
	s.SetGauge("http_requests_total", map[string]string{"method": "POST", "status": "200"}, float64(rand.Intn(5000)))
	
	// Response time
	s.ObserveHistogram("http_request_duration_seconds", map[string]string{"method": "GET"}, rand.Float64()*2)
	s.ObserveHistogram("http_request_duration_seconds", map[string]string{"method": "POST"}, rand.Float64()*3)
	
	// Memory usage (simulated)
	s.SetGauge("process_memory_bytes", map[string]string{}, float64(rand.Intn(1000000000)))
	
	// CPU usage (simulated)
	s.SetGauge("process_cpu_percent", map[string]string{}, rand.Float64()*100)
	
	// Active connections
	s.SetGauge("active_connections", map[string]string{}, float64(rand.Intn(1000)))
	
	// Queue size
	s.SetGauge("queue_size", map[string]string{"service": "wallet"}, float64(rand.Intn(100)))
	s.SetGauge("queue_size", map[string]string{"service": "transaction"}, float64(rand.Intn(50)))
}
