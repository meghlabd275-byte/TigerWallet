package main

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics for the token-listing service.
//
// Listing gates whether a token can be traded, so the signals that matter are
// whether the backing store and the KYC engine are reachable, and whether
// payment webhooks are being accepted or rejected. A rejected payment webhook
// means an applicant paid but the listing did not advance, which is a
// customer-visible failure an operator must see.
var (
	listHTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by method, route template and status.",
		},
		[]string{"method", "route", "status"},
	)

	listHTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method and route template.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	listRedisUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "listing_redis_up",
		Help: "1 when Redis answers, 0 otherwise.",
	})

	// listKYCUp reports whether the KYC engine answers. The listing service
	// delegates verification to it, so a down engine blocks every applicant.
	listKYCUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "listing_kyc_engine_up",
		Help: "1 when the KYC engine answers, 0 otherwise.",
	})

	listWebhooksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "listing_payment_webhooks_total",
			Help: "Listing payment webhooks by outcome (accepted, rejected, error).",
		},
		[]string{"outcome"},
	)

	listApplicationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "listing_applications_total",
			Help: "Listing applications submitted, by outcome.",
		},
		[]string{"outcome"},
	)
)

func init() {
	prometheus.MustRegister(
		listHTTPRequestsTotal,
		listHTTPRequestDuration,
		listRedisUp,
		listKYCUp,
		listWebhooksTotal,
		listApplicationsTotal,
	)
}

// listMetricsService is set from main so a scrape can probe live dependencies.
var listMetricsService *ListingService

// registerListingMetrics mounts /metrics and installs request instrumentation.
// It must run before the routes so the middleware covers every handler.
func registerListingMetrics(r *gin.Engine) {
	r.GET("/metrics", func(c *gin.Context) {
		refreshListingGauges()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	r.Use(listingMetricsMiddleware())
}

func listingMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			// An unmatched request would otherwise create one series per path.
			route = "unmatched"
		}
		listHTTPRequestsTotal.WithLabelValues(
			c.Request.Method, route, strconv.Itoa(c.Writer.Status()),
		).Inc()
		listHTTPRequestDuration.WithLabelValues(
			c.Request.Method, route,
		).Observe(time.Since(start).Seconds())
	}
}

// refreshListingGauges probes Redis and the KYC engine for the current scrape.
// Both probes are bounded so a hung dependency cannot stall the endpoint.
func refreshListingGauges() {
	if listMetricsService == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if listMetricsService.redis != nil {
		if err := listMetricsService.redis.Ping(ctx).Err(); err == nil {
			listRedisUp.Set(1)
		} else {
			listRedisUp.Set(0)
		}
	}

	// The listing service delegates identity verification to the KYC engine
	// over HTTP, so reachability is probed the same way the caller would.
	base := kycEngineURL()
	if base == "" {
		listKYCUp.Set(0)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/health", nil)
	if err != nil {
		listKYCUp.Set(0)
		return
	}
	if tok := kycEngineToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		listKYCUp.Set(0)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		listKYCUp.Set(1)
	} else {
		listKYCUp.Set(0)
	}
}
