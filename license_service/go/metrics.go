package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/tigerwallet/license-service/internal/store"
)

// Prometheus metrics for the license control plane.
//
// license_service is the authority every white-label product consults on
// heartbeat. If it stops answering, WL products fail closed, so its
// availability and its dependency probes are the signals that matter most.
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by method, route template and status.",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method and route template.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	dbUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "license_service_db_up",
		Help: "1 when PostgreSQL answers, 0 otherwise.",
	})

	redisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "license_service_redis_up",
		Help: "1 when Redis answers, 0 otherwise. Redis carries the command fan-out to WL products.",
	})

	// wlClients is the number of registered white-label clients. A drop is a
	// provisioning or data-loss signal.
	wlClients = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "license_service_wl_clients",
		Help: "Number of registered white-label clients.",
	})
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		dbUp,
		redisUp,
		wlClients,
	)
}

// metricsStore is set from main so a scrape can probe the live dependencies.
var metricsStore *store.Store

// metricsHub is set from main so a scrape can probe Redis.
var metricsHub *store.Hub

// registerMetrics mounts /metrics and installs the request instrumentation.
// It must be called before the routes so the middleware covers every handler.
func registerMetrics(r *gin.Engine) {
	r.GET("/metrics", metricsHandler())
	r.Use(metricsMiddleware())
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			// An unmatched request would otherwise create one series per path.
			route = "unmatched"
		}
		httpRequestsTotal.WithLabelValues(
			c.Request.Method, route, strconv.Itoa(c.Writer.Status()),
		).Inc()
		httpRequestDuration.WithLabelValues(
			c.Request.Method, route,
		).Observe(time.Since(start).Seconds())
	}
}

// refreshGauges probes PostgreSQL and counts clients for the current scrape.
// The probe is bounded so a hung database cannot stall the metrics endpoint.
func refreshGauges() {
	if metricsStore == nil || metricsStore.DB() == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool := metricsStore.DB()
	if err := pool.Ping(ctx); err == nil {
		dbUp.Set(1)
	} else {
		dbUp.Set(0)
		return
	}

	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM wl_clients`).Scan(&n); err == nil {
		wlClients.Set(float64(n))
	}

	if metricsHub != nil {
		if err := metricsHub.Ping(ctx); err == nil {
			redisUp.Set(1)
		} else {
			redisUp.Set(0)
		}
	}
}

func metricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}
}
