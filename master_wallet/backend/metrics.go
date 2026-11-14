package main

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics for the MasterWallet backend.
//
// The auto-signer is the component that moves funds without a human in the
// loop, so its outcome is the most important thing to watch here: a rise in
// refusals or broadcast failures is a live incident, not a curiosity.
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
		Name: "master_wallet_db_up",
		Help: "1 when PostgreSQL answers, 0 otherwise.",
	})

	redisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "master_wallet_redis_up",
		Help: "1 when Redis answers, 0 otherwise.",
	})

	// transactionsBroadcast counts transactions accepted by an upstream node,
	// across every chain family. This is the honest "custody moved funds"
	// signal; a signing failure never increments it.
	transactionsBroadcast = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "master_wallet_transactions_broadcast_total",
		Help: "Custody transactions accepted by an upstream node.",
	})

	// autoSignProcessed counts every auto-signer decision. A sustained rise in
	// "failed" or "refused" means funds are not moving as the policy intends.
	autoSignProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "master_wallet_auto_sign_processed_total",
			Help: "Auto-signer decisions by outcome (broadcast, refused, failed, requeued).",
		},
		[]string{"outcome"},
	)

	// autoSignClaimed counts transactions claimed for processing, which is the
	// throughput signal for the cluster-safe claiming loop.
	autoSignClaimed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "master_wallet_auto_sign_claimed_total",
		Help: "Transactions claimed by an auto-signer replica.",
	})

	// autoSignReaped counts stale claims recovered from a crashed replica. A
	// non-zero rate means a replica is dying mid-sign.
	autoSignReaped = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "master_wallet_auto_sign_reaped_total",
		Help: "Stale auto-sign claims recovered from a crashed replica.",
	})

	// twoPartyCoSigns counts SuperAdmin co-sign operations on the
	// revenue/treasury paths, which are the highest-privilege actions.
	twoPartyCoSigns = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "master_wallet_two_party_cosign_total",
		Help: "Two-party co-sign operations completed.",
	})
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		dbUp,
		redisUp,
		transactionsBroadcast,
		autoSignProcessed,
		autoSignClaimed,
		autoSignReaped,
		twoPartyCoSigns,
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

	if metricsStore.db != nil {
		if err := metricsStore.db.Ping(ctx); err == nil {
			dbUp.Set(1)
		} else {
			dbUp.Set(0)
		}
	}
	if metricsStore.redis != nil {
		if err := metricsStore.redis.Ping(ctx).Err(); err == nil {
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
