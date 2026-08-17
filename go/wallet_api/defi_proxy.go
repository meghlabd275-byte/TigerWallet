package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// Reverse-proxy shims for the auxiliary DeFi microservices so that every
// UserWallet client (web/desktop/android/ios/rust/extension/production-react)
// can target a single canonical port (:8443) and reach the full feature set.
// Each DeFi service runs independently with its own PostgreSQL store; this
// layer only forwards authenticated requests transparently — no data is
// fabricated, and upstream failures are surfaced honestly (502/503).
//
// Service URL env vars (all optional, with safe localhost defaults):
//   LENDING_SERVICE_URL      -> go/lending_service      (default :8009)
//   COPYTRADING_SERVICE_URL  -> go/copy_trading_service (default :8006)
//   GOVERNANCE_SERVICE_URL   -> go/governance_service   (default :8454)
//   PREDICTION_SERVICE_URL   -> go/prediction_service   (default :8455)

func serviceURL(envKey, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}
	return fallback
}

// deFiProxy returns a gin handler that reverse-proxies the request to the
// configured upstream DeFi service, preserving the path under /api/v1/<group>
// and forwarding the Authorization header. Upstream is resolved per-request so
// admin/runtime env changes take effect without a restart.
func deFiProxy(upstreamEnv, fallback, groupPrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := serviceURL(upstreamEnv, fallback)
		base, err := url.Parse(raw)
		if err != nil || base.Scheme == "" || base.Host == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "DeFi service URL misconfigured"})
			return
		}
		// Forward the full original path (the services expose /api/v1/<group>/...).
		proxy := httputil.NewSingleHostReverseProxy(base)
		// Preserve the original request path (SingleHostReverseProxy joins
		// base.Path + req.URL.Path; since base.Path is empty this is the raw path).
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			// httputil strips nothing here; keep req.URL.Path as-is.
			req.Host = base.Host
		}
		// Surface upstream errors honestly instead of masking them.
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			c.Error(err) //nolint:errcheck
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "DeFi service unavailable", "detail": err.Error()})
		}
		_ = groupPrefix // kept for readability of the route registration call sites
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
