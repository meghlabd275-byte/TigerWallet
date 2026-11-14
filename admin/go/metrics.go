package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tigerwallet/admin/pkg/database"
	"github.com/tigerwallet/admin/pkg/redis"
)

// Prometheus metrics for the canonical admin backend.
//
// The admin API already keeps an in-process traffic collector for its
// analytics endpoint, but Prometheus cannot scrape that: it needs a /metrics
// route. Without one the scrape target reports 404 and shows as permanently
// down, which hides real admin outages behind constant alert noise.
var (
	adminHTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by method, route template and status.",
		},
		[]string{"method", "route", "status"},
	)

	adminHTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method and route template.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	adminDBUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "admin_db_up",
		Help: "1 when PostgreSQL answers, 0 otherwise.",
	})

	adminRedisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "admin_redis_up",
		Help: "1 when Redis answers, 0 otherwise.",
	})

	adminAuthFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_auth_failures_total",
			Help: "Rejected admin authentications by reason.",
		},
		[]string{"reason"},
	)
)

func init() {
	prometheus.MustRegister(
		adminHTTPRequestsTotal,
		adminHTTPRequestDuration,
		adminDBUp,
		adminRedisUp,
		adminAuthFailures,
	)
}

// adminMetricsState is set from main so a scrape can probe the live dependencies.
var adminMetricsState struct {
	db    *database.PostgresDB
	redis *redis.RedisClient
}

// registerAdminMetrics mounts /metrics and installs the request instrumentation.
// It must be called before the routes so the middleware covers every handler.
func registerAdminMetrics(router *gin.Engine) {
	router.GET("/metrics", func(c *gin.Context) {
		refreshAdminGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	router.Use(adminMetricsMiddleware())
}

func adminMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			// An unmatched request would otherwise create one series per path.
			route = "unmatched"
		}
		adminHTTPRequestsTotal.WithLabelValues(
			c.Request.Method, route, strconv.Itoa(c.Writer.Status()),
		).Inc()
		adminHTTPRequestDuration.WithLabelValues(
			c.Request.Method, route,
		).Observe(time.Since(start).Seconds())
	}
}

// refreshAdminGauges probes PostgreSQL and Redis for the current scrape. Both
// probes are bounded so a hung dependency cannot stall the endpoint.
func refreshAdminGauges() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if adminMetricsState.db != nil && adminMetricsState.db.DB != nil {
		if sqlDB, err := adminMetricsState.db.DB.DB(); err == nil {
			if err := sqlDB.PingContext(ctx); err == nil {
				adminDBUp.Set(1)
			} else {
				adminDBUp.Set(0)
			}
		}
	}

	if adminMetricsState.redis != nil && adminMetricsState.redis.Client != nil {
		if err := adminMetricsState.redis.Client.Ping(ctx).Err(); err == nil {
			adminRedisUp.Set(1)
		} else {
			adminRedisUp.Set(0)
		}
	}
}
