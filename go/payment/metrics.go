package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics for the payment service.
//
// Payment is a money path, so the signals that matter most are whether the
// dependencies that back it are still reachable, and whether webhooks are
// arriving and being accepted. A drop in accepted webhooks while Redis stays up
// points at signature or payload handling rather than an outage.
var (
	payHTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by method, route template and status.",
		},
		[]string{"method", "route", "status"},
	)

	payHTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method and route template.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	payRedisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "payment_redis_up",
		Help: "1 when Redis answers, 0 otherwise. Payment state is stored in Redis.",
	})

	// payWebhooksTotal counts webhook deliveries by outcome. "rejected" means
	// the signature or the on-chain re-read did not verify, which is the
	// signal an operator needs to see rather than a silent 200.
	payWebhooksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_webhooks_total",
			Help: "Payment webhook deliveries by outcome (accepted, rejected, error).",
		},
		[]string{"outcome"},
	)

	payDepositsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_deposits_total",
			Help: "Deposit intents created, by chain and outcome.",
		},
		[]string{"chain", "outcome"},
	)
)

func init() {
	prometheus.MustRegister(
		payHTTPRequestsTotal,
		payHTTPRequestDuration,
		payRedisUp,
		payWebhooksTotal,
		payDepositsTotal,
	)
}

// payMetricsService is set from main so a scrape can probe live dependencies.
var payMetricsService *PaymentService

// registerPaymentMetrics mounts /metrics and installs request instrumentation.
// It must run before the routes so the middleware covers every handler.
func registerPaymentMetrics(r *gin.Engine) {
	r.GET("/metrics", func(c *gin.Context) {
		refreshPaymentGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	r.Use(paymentMetricsMiddleware())
}

func paymentMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			// An unmatched request would otherwise create one series per path.
			route = "unmatched"
		}
		payHTTPRequestsTotal.WithLabelValues(
			c.Request.Method, route, strconv.Itoa(c.Writer.Status()),
		).Inc()
		payHTTPRequestDuration.WithLabelValues(
			c.Request.Method, route,
		).Observe(time.Since(start).Seconds())
	}
}

// refreshPaymentGauges probes Redis for the current scrape, bounded so a hung
// dependency cannot stall the endpoint.
func refreshPaymentGauges() {
	if payMetricsService == nil || payMetricsService.redis == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := payMetricsService.redis.Ping(ctx).Err(); err == nil {
		payRedisUp.Set(1)
	} else {
		payRedisUp.Set(0)
	}
}
