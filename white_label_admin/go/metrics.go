package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// Prometheus metrics for the white-label admin backend.
//
// This tier is tenant-scoped, so the useful signal is whether the shared
// PostgreSQL pool and the feature-flag Redis are reachable. Feature flags are
// published to Redis; if that link is down an operator's enable/disable action
// silently stops reaching the tenants, which is exactly what this gauge shows.
var (
	wlaHTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by method, route template and status.",
		},
		[]string{"method", "route", "status"},
	)

	wlaHTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method and route template.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	wlaPGUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wl_admin_postgres_up",
		Help: "1 when PostgreSQL answers, 0 otherwise.",
	})

	wlaRedisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wl_admin_redis_up",
		Help: "1 when Redis answers, 0 otherwise. Feature flags propagate through it.",
	})
)

func init() {
	prometheus.MustRegister(
		wlaHTTPRequestsTotal,
		wlaHTTPRequestDuration,
		wlaPGUp,
		wlaRedisUp,
	)
}

// wlaPG and wlaRedis are set from main so a scrape can probe live dependencies.
// wlaRedis is the raw client, not the wrapper, so this package does not depend
// on the internal redis package's type.
var (
	wlaPG    *pgxpool.Pool
	wlaRedis *redis.Client
)

// registerWLMetrics mounts /metrics and installs request instrumentation. It
// must run before the routes so the middleware covers every handler.
func registerWLMetrics(r *gin.Engine) {
	r.GET("/metrics", func(c *gin.Context) {
		refreshWLGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	r.Use(wlMetricsMiddleware())
}

func wlMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			// An unmatched request would otherwise create one series per path.
			route = "unmatched"
		}
		wlaHTTPRequestsTotal.WithLabelValues(
			c.Request.Method, route, strconv.Itoa(c.Writer.Status()),
		).Inc()
		wlaHTTPRequestDuration.WithLabelValues(
			c.Request.Method, route,
		).Observe(time.Since(start).Seconds())
	}
}

// refreshWLGauges probes PostgreSQL and Redis for the current scrape, bounded
// so a hung dependency cannot stall the endpoint.
func refreshWLGauges() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if wlaPG != nil {
		if err := wlaPG.Ping(ctx); err == nil {
			wlaPGUp.Set(1)
		} else {
			wlaPGUp.Set(0)
		}
	}
	if wlaRedis != nil {
		if err := wlaRedis.Ping(ctx).Err(); err == nil {
			wlaRedisUp.Set(1)
		} else {
			wlaRedisUp.Set(0)
		}
	}
}
