package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics for the kill-switch control plane.
//
// A halt is the most consequential action in the platform, so the signals that
// matter most are: how many halts are active right now, and whether the halt
// state is still reaching Redis. If the Redis gauge drops, halts may stop
// propagating to white-label products even though PostgreSQL still has them.
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
		Name: "kill_switch_db_up",
		Help: "1 when PostgreSQL answers, 0 otherwise.",
	})

	redisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kill_switch_redis_up",
		Help: "1 when Redis answers, 0 otherwise. Redis is the propagation path for halts.",
	})

	// activeHalts is the number of currently active halt records. It is read
	// from the same query the state endpoint uses, so it cannot drift from what
	// the API reports.
	activeHalts = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kill_switch_active_halts",
		Help: "Number of currently active halts.",
	})

	haltActions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kill_switch_actions_total",
			Help: "Halt and resume actions performed, by action and scope type.",
		},
		[]string{"action", "scope_type"},
	)
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		dbUp,
		redisUp,
		activeHalts,
		haltActions,
	)
}

// metricsServer is set from main so a scrape can probe the live dependencies.
var metricsServer *server

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

// refreshGauges probes PostgreSQL, Redis and the active-halt count for the
// current scrape. Each probe is bounded so a hung dependency cannot stall the
// metrics endpoint.
func refreshGauges() {
	if metricsServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if metricsServer.db != nil {
		if err := metricsServer.db.Ping(ctx); err == nil {
			dbUp.Set(1)
		} else {
			dbUp.Set(0)
		}
	}
	if metricsServer.rdb != nil {
		if err := metricsServer.rdb.Ping(ctx).Err(); err == nil {
			redisUp.Set(1)
		} else {
			redisUp.Set(0)
		}
	}
	if metricsServer.db != nil {
		var n int
		if err := metricsServer.db.QueryRow(ctx, `SELECT COUNT(*) FROM kill_state WHERE active`).Scan(&n); err == nil {
			activeHalts.Set(float64(n))
		}
	}
}

func metricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}
}
