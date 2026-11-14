package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// Prometheus metrics for the project-party (token listing and launchpad) backend.
//
// Listings are read-heavy and cached, so a Redis outage degrades the discovery
// endpoints without failing them. The gauge makes that degradation visible
// rather than leaving it to be inferred from latency.
var (
	ppHTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by method, route template and status.",
		},
		[]string{"method", "route", "status"},
	)

	ppHTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method and route template.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	ppPGUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "project_party_postgres_up",
		Help: "1 when PostgreSQL answers, 0 otherwise.",
	})

	ppRedisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "project_party_redis_up",
		Help: "1 when Redis answers, 0 otherwise. Discovery responses are cached there.",
	})

	ppLaunchpadOnChain = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "project_party_launchpad_onchain_enabled",
		Help: "1 when the on-chain launchpad is configured, 0 when contributions fail closed.",
	})
)

func init() {
	prometheus.MustRegister(
		ppHTTPRequestsTotal,
		ppHTTPRequestDuration,
		ppPGUp,
		ppRedisUp,
		ppLaunchpadOnChain,
	)
}

// ppMetricsDB and ppMetricsRedis are set from main so a scrape can probe live
// dependencies.
var (
	ppMetricsDB    *pgxpool.Pool
	ppMetricsRedis *redis.Client
)

// registerProjectPartyMetrics mounts /metrics and installs request
// instrumentation. It must run before the routes so the middleware covers them.
func registerProjectPartyMetrics(r *gin.Engine) {
	r.GET("/metrics", func(c *gin.Context) {
		refreshProjectPartyGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	r.Use(projectPartyMetricsMiddleware())
}

func projectPartyMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			// An unmatched request would otherwise create one series per path.
			route = "unmatched"
		}
		ppHTTPRequestsTotal.WithLabelValues(
			c.Request.Method, route, strconv.Itoa(c.Writer.Status()),
		).Inc()
		ppHTTPRequestDuration.WithLabelValues(
			c.Request.Method, route,
		).Observe(time.Since(start).Seconds())
	}
}

// refreshProjectPartyGauges probes PostgreSQL, Redis and the launchpad
// configuration for the current scrape, bounded so a hung dependency cannot
// stall the endpoint.
func refreshProjectPartyGauges() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if ppMetricsDB != nil {
		if err := ppMetricsDB.Ping(ctx); err == nil {
			ppPGUp.Set(1)
		} else {
			ppPGUp.Set(0)
		}
	}
	if ppMetricsRedis != nil {
		if err := ppMetricsRedis.Ping(ctx).Err(); err == nil {
			ppRedisUp.Set(1)
		} else {
			ppRedisUp.Set(0)
		}
	}
	if launchpadOnChainEnabled() {
		ppLaunchpadOnChain.Set(1)
	} else {
		ppLaunchpadOnChain.Set(0)
	}
}
