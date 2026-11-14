// Prometheus instrumentation for the Notification Service.
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
	"github.com/redis/go-redis/v9"
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

	// A channel with no provider configured reports "unavailable" rather than
	// claiming delivery, so a non-zero failure count here is the signal that a
	// provider is down or misconfigured.
	notificationsDelivered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notifications_delivered_total",
			Help: "Notification delivery attempts, by channel and outcome.",
		},
		[]string{"channel", "outcome"},
	)

	queueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "notifications_queue_depth",
		Help: "Notifications waiting in the delivery queue.",
	})

	dbUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "notifications_db_up",
		Help: "1 when PostgreSQL answers, 0 otherwise.",
	})

	redisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "notifications_redis_up",
		Help: "1 when Redis answers, 0 otherwise.",
	})
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		notificationsDelivered,
		queueDepth,
		dbUp,
		redisUp,
	)
}

// registerMetrics wires the scrape endpoint and request instrumentation.
func registerMetrics(router *gin.Engine) {
	router.GET("/metrics", func(c *gin.Context) {
		refreshDependencyGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	router.Use(metricsMiddleware)
}

// metricsMiddleware records request count and latency. The route label uses the
// matched pattern rather than the raw path, so a path parameter such as
// /api/v1/notifications/:user_id does not explode label cardinality.
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

// metricsDB and metricsRedis are set from main so a scrape can probe them.
var (
	metricsDB    DB
	metricsRedis *redis.Client
)

// refreshDependencyGauges probes PostgreSQL and Redis. The probe is bounded so
// a hung dependency cannot stall a scrape.
func refreshDependencyGauges() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if metricsDB != nil && metricsDB.Ping(ctx) == nil {
		dbUp.Set(1)
	} else {
		dbUp.Set(0)
	}

	if metricsRedis != nil && metricsRedis.Ping(ctx).Err() == nil {
		redisUp.Set(1)
	} else {
		redisUp.Set(0)
	}
}
