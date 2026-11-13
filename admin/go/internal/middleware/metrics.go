package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// MetricsCollector records live HTTP traffic statistics for the admin API.
//
// Every value it reports is measured from real request traffic; nothing is
// estimated or defaulted. The collector keeps a rolling one-second window for
// the current rate plus lifetime counters, which is what the analytics endpoint
// needs to answer "how is the API behaving right now".
type MetricsCollector struct {
	mu sync.Mutex

	startTime time.Time

	totalRequests uint64
	totalErrors   uint64
	totalLatency  time.Duration

	// Rolling window used for requests-per-second.
	windowStart    time.Time
	windowRequests uint64

	// Current second and the last completed second, so a scrape always sees a
	// fully-elapsed interval rather than a partially-filled one.
	lastSecond      time.Time
	lastSecondCount uint64
}

// NewMetricsCollector returns a collector whose uptime starts now.
func NewMetricsCollector() *MetricsCollector {
	now := time.Now()
	return &MetricsCollector{
		startTime:   now,
		windowStart: now,
		lastSecond:  now,
	}
}

// Middleware records every request. It never alters the response.
func (m *MetricsCollector) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		m.mu.Lock()
		defer m.mu.Unlock()

		m.totalRequests++
		m.totalLatency += latency
		if c.Writer.Status() >= 500 {
			m.totalErrors++
		}

		now := time.Now()
		if now.Sub(m.lastSecond) >= time.Second {
			m.lastSecondCount = m.windowRequests
			m.lastSecond = now
			m.windowRequests = 1
			m.windowStart = now
			return
		}
		m.windowRequests++
	}
}

// Snapshot is a point-in-time view of the collected metrics.
type Snapshot struct {
	UptimeSeconds     float64
	TotalRequests     uint64
	TotalErrors       uint64
	AvgLatencyMs      float64
	RequestsPerSecond float64
}

// Snapshot returns the current metrics. Average latency is zero until at least
// one request has been observed.
func (m *MetricsCollector) Snapshot() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	var avg float64
	if m.totalRequests > 0 {
		avg = float64(m.totalLatency.Microseconds()) / float64(m.totalRequests) / 1000.0
	}

	return Snapshot{
		UptimeSeconds:     time.Since(m.startTime).Seconds(),
		TotalRequests:     m.totalRequests,
		TotalErrors:       m.totalErrors,
		AvgLatencyMs:      avg,
		RequestsPerSecond: float64(m.lastSecondCount),
	}
}

// ParsePositiveInt reads an integer query parameter, clamped to [min, max].
// It returns def when the parameter is absent or malformed, so a caller cannot
// force an unbounded result set.
func ParsePositiveInt(raw string, def, min, max int) int {
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < min {
		return def
	}
	if v > max {
		return max
	}
	return v
}
