package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics for the fiat on/off-ramp service.
//
// Ramp orders move real money between a bank and a wallet, so the signals that
// matter are whether PostgreSQL and Redis are reachable, and whether provider
// webhooks verify. A rise in rejected webhooks while both stores are up points
// at signature handling or a provider sending an unexpected shape, not an
// outage.
var (
	rampHTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by method, route template and status.",
		},
		[]string{"method", "route", "status"},
	)

	rampHTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method and route template.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	rampPGUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "fiat_ramp_postgres_up",
		Help: "1 when PostgreSQL answers, 0 otherwise. Ramp orders are stored there.",
	})

	rampRedisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "fiat_ramp_redis_up",
		Help: "1 when Redis answers, 0 otherwise.",
	})

	rampWebhooksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fiat_ramp_webhooks_total",
			Help: "Provider webhooks by provider and outcome (accepted, rejected, error).",
		},
		[]string{"provider", "outcome"},
	)

	rampOrdersTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fiat_ramp_orders_total",
			Help: "Ramp orders by direction (on, off) and outcome.",
		},
		[]string{"direction", "outcome"},
	)
)

func init() {
	prometheus.MustRegister(
		rampHTTPRequestsTotal,
		rampHTTPRequestDuration,
		rampPGUp,
		rampRedisUp,
		rampWebhooksTotal,
		rampOrdersTotal,
	)
}

// rampMetricsService is set from main so a scrape can probe live dependencies.
var rampMetricsService *FiatRampService

// registerFiatRampMetrics mounts /metrics and installs request instrumentation.
// It must run before the routes so the middleware covers every handler.
func registerFiatRampMetrics(r *gin.Engine) {
	r.GET("/metrics", func(c *gin.Context) {
		refreshRampGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	r.Use(rampMetricsMiddleware())
}

func rampMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			// An unmatched request would otherwise create one series per path.
			route = "unmatched"
		}
		rampHTTPRequestsTotal.WithLabelValues(
			c.Request.Method, route, strconv.Itoa(c.Writer.Status()),
		).Inc()
		rampHTTPRequestDuration.WithLabelValues(
			c.Request.Method, route,
		).Observe(time.Since(start).Seconds())
	}
}

// refreshRampGauges probes PostgreSQL and Redis for the current scrape. Both
// probes are bounded so a hung dependency cannot stall the endpoint.
func refreshRampGauges() {
	if rampMetricsService == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if rampMetricsService.pg != nil {
		if err := rampMetricsService.pg.Ping(ctx); err == nil {
			rampPGUp.Set(1)
		} else {
			rampPGUp.Set(0)
		}
	}
	if rampMetricsService.redis != nil {
		if err := rampMetricsService.redis.Ping(ctx).Err(); err == nil {
			rampRedisUp.Set(1)
		} else {
			rampRedisUp.Set(0)
		}
	}
}
