package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics for the tenant-facing permission edge.
//
// permission_bridge is the edge every white-label product calls to learn what
// it is allowed to do. The signals that matter are availability, database
// health, and how many products/permissions are registered.
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
		Name: "permission_bridge_db_up",
		Help: "1 when PostgreSQL answers, 0 otherwise.",
	})

	registeredProducts = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "permission_bridge_products",
		Help: "Number of registered products.",
	})

	permissionRows = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "permission_bridge_permissions",
		Help: "Number of permission records.",
	})
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		dbUp,
		registeredProducts,
		permissionRows,
	)
}

// metricsHandlerState is set from main so a scrape can probe the live database.
var metricsHandlerState *handlers

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

// refreshGauges probes PostgreSQL and counts registered rows for the current
// scrape. The probe is bounded so a hung database cannot stall the endpoint.
func refreshGauges() {
	if metricsHandlerState == nil || metricsHandlerState.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	db := metricsHandlerState.db
	if err := db.Ping(ctx); err != nil {
		dbUp.Set(0)
		return
	}
	dbUp.Set(1)

	var products, perms int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM pb_products`).Scan(&products); err == nil {
		registeredProducts.Set(float64(products))
	}
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM pb_permissions`).Scan(&perms); err == nil {
		permissionRows.Set(float64(perms))
	}
}

func metricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}
}
