// Prometheus instrumentation for the Billing & Subscription Service.
//
// infrastructure/docker/monitoring/prometheus.yml scrapes /metrics on this
// service, so the endpoint must exist or every scrape returns 404 and the
// service is invisible to monitoring.
package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests handled, by method, route and status.",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	// Dependency health, so an alert can fire on a database or cache outage
	// rather than on the absence of traffic.
	dbUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "billing_db_up",
		Help: "1 when the PostgreSQL pool answers, 0 otherwise.",
	})

	redisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "billing_redis_up",
		Help: "1 when Redis answers, 0 otherwise.",
	})

	// Billing-domain counters. These are incremented on the real money paths.
	invoicesCreated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "billing_invoices_created_total",
		Help: "Invoices created.",
	})

	paymentsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "billing_payments_processed_total",
			Help: "Payment webhook results, by outcome.",
		},
		[]string{"outcome"},
	)

	usageRecords = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "billing_usage_records_total",
		Help: "Usage records ingested.",
	})
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		dbUp,
		redisUp,
		invoicesCreated,
		paymentsProcessed,
		usageRecords,
	)
}

// registerMetrics wires the scrape endpoint and the request instrumentation.
func registerMetrics(router *gin.Engine) {
	router.GET("/metrics", func(c *gin.Context) {
		refreshDependencyGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	router.Use(metricsMiddleware)
}

// metricsMiddleware records request count and latency. The route label uses the
// matched pattern rather than the raw path so a path parameter such as
// /api/v1/subscriptions/:client_id does not explode label cardinality.
func metricsMiddleware(c *gin.Context) {
	start := time.Now()
	c.Next()

	route := c.FullPath()
	if route == "" {
		route = "unmatched"
	}
	httpRequestsTotal.WithLabelValues(c.Request.Method, route, strconv.Itoa(c.Writer.Status())).Inc()
	httpRequestDuration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
}

// refreshDependencyGauges probes PostgreSQL and Redis so an alert can fire on a
// dependency outage rather than only on the absence of traffic. The probe is
// bounded so a hung dependency cannot stall a scrape.
func refreshDependencyGauges() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if db != nil && db.Ping(ctx) == nil {
		dbUp.Set(1)
	} else {
		dbUp.Set(0)
	}

	if redisClient != nil && redisClient.Ping(ctx).Err() == nil {
		redisUp.Set(1)
	} else {
		redisUp.Set(0)
	}
}
