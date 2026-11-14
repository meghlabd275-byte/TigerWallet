package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics for the UserWallet API.
//
// Route labels use the registered route template (c.FullPath()), never the raw
// URL, so a path parameter such as a wallet id does not explode the series
// count.
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
		Name: "wallet_api_db_up",
		Help: "1 when PostgreSQL answers, 0 otherwise.",
	})

	redisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wallet_api_redis_up",
		Help: "1 when Redis answers, 0 otherwise.",
	})

	// TransactionsBroadcast counts signed transactions that were accepted by a
	// node. A send that fails validation is not counted, so this is the honest
	// "money moved" signal rather than "send endpoint called".
	transactionsBroadcast = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "wallet_api_transactions_broadcast_total",
		Help: "Transactions signed and accepted by an upstream node.",
	})

	// AuthFailures counts rejected logins. A sustained rise is the credential
	// stuffing signal.
	authFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "wallet_api_auth_failures_total",
		Help: "Rejected authentication attempts.",
	})
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		dbUp,
		redisUp,
		transactionsBroadcast,
		authFailures,
	)
}

// metricsStore is set from main so a scrape can probe the live dependencies.
var metricsStore *Store

// registerMetrics mounts /metrics and installs the request instrumentation.
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

// refreshDependencyGauges probes PostgreSQL and Redis for the current scrape.
// It is bounded so a hung dependency cannot stall the metrics endpoint.
func refreshDependencyGauges() {
	if metricsStore == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if metricsStore.PG != nil {
		if err := metricsStore.PG.Ping(ctx); err == nil {
			dbUp.Set(1)
		} else {
			dbUp.Set(0)
		}
	}
	if metricsStore.Redis != nil {
		if err := metricsStore.Redis.Ping(ctx).Err(); err == nil {
			redisUp.Set(1)
		} else {
			redisUp.Set(0)
		}
	}
}

// metricsHandler serves the registry after refreshing the dependency gauges.
func metricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshDependencyGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}
}
