package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics for the trading-bot API.
//
// The bot API is the control plane in front of bot_core, so the signals that
// matter are whether PostgreSQL and Redis are reachable and whether dispatch to
// bot_core is succeeding. A bot whose start request is accepted but whose
// dispatch fails is a silent failure an operator must be able to see.
var (
	botHTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by method, route template and status.",
		},
		[]string{"method", "route", "status"},
	)

	botHTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method and route template.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	botPGUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bot_api_postgres_up",
		Help: "1 when PostgreSQL answers, 0 otherwise.",
	})

	botRedisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bot_api_redis_up",
		Help: "1 when Redis answers, 0 otherwise.",
	})

	// botDispatchesTotal counts dispatch attempts to bot_core by outcome.
	// "error" means bot_core could not be reached or refused the request.
	botDispatchesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bot_api_dispatches_total",
			Help: "Dispatch attempts to bot_core by bot type and outcome.",
		},
		[]string{"bot_type", "outcome"},
	)
)

func init() {
	prometheus.MustRegister(
		botHTTPRequestsTotal,
		botHTTPRequestDuration,
		botPGUp,
		botRedisUp,
		botDispatchesTotal,
	)
}

// botMetricsSvc is set from main so a scrape can probe live dependencies.
var botMetricsSvc *service

// registerBotMetrics mounts /metrics and installs request instrumentation. It
// must run before the routes so the middleware covers every handler.
func registerBotMetrics(r *gin.Engine) {
	r.GET("/metrics", func(c *gin.Context) {
		refreshBotGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	r.Use(botMetricsMiddleware())
}

func botMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			// An unmatched request would otherwise create one series per path.
			route = "unmatched"
		}
		botHTTPRequestsTotal.WithLabelValues(
			c.Request.Method, route, strconv.Itoa(c.Writer.Status()),
		).Inc()
		botHTTPRequestDuration.WithLabelValues(
			c.Request.Method, route,
		).Observe(time.Since(start).Seconds())
	}
}

// refreshBotGauges probes PostgreSQL and Redis for the current scrape, bounded
// so a hung dependency cannot stall the endpoint.
func refreshBotGauges() {
	if botMetricsSvc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if botMetricsSvc.pg != nil {
		if err := botMetricsSvc.pg.Ping(ctx); err == nil {
			botPGUp.Set(1)
		} else {
			botPGUp.Set(0)
		}
	}
	if botMetricsSvc.redis != nil {
		if err := botMetricsSvc.redis.Ping(ctx).Err(); err == nil {
			botRedisUp.Set(1)
		} else {
			botRedisUp.Set(0)
		}
	}
}
