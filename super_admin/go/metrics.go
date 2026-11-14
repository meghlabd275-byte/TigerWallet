package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/tigerwallet/super-admin/internal/database"
)

// Prometheus metrics for the SuperAdmin control plane.
//
// SuperAdmin is the highest-privilege surface in the platform, so the signals
// that matter are availability, dependency health, and a count of the accounts
// that can reach it.
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
		Name: "super_admin_db_up",
		Help: "1 when PostgreSQL answers, 0 otherwise.",
	})

	redisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "super_admin_redis_up",
		Help: "1 when Redis answers, 0 otherwise. Redis carries the feature-flag fan-out.",
	})

	// adminAccounts counts privileged accounts. An unexpected rise is a
	// privilege-escalation signal.
	adminAccounts = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "super_admin_accounts",
		Help: "Number of admin accounts.",
	})
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		dbUp,
		redisUp,
		adminAccounts,
	)
}

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

// refreshGauges probes PostgreSQL, Redis and counts accounts for the current
// scrape. Each probe is bounded so a hung dependency cannot stall the endpoint.
func refreshGauges() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if database.Pool != nil {
		if err := database.Pool.Ping(ctx); err == nil {
			dbUp.Set(1)
			var n int
			if err := database.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_users`).Scan(&n); err == nil {
				adminAccounts.Set(float64(n))
			}
		} else {
			dbUp.Set(0)
		}
	}
	if redisClient != nil && redisClient.Client != nil {
		if err := redisClient.Client.Ping(ctx).Err(); err == nil {
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
