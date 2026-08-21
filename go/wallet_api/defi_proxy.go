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
	return deFiProxyRewrite(upstreamEnv, fallback, groupPrefix, "/api/v1/"+groupPrefix)
}

// deFiProxyRewrite behaves like deFiProxy but remaps the path when the
// upstream service mounts its routes under a different prefix than the one
// the UserWallet clients use: the incoming prefix "/api/v1/<groupPrefix>" is
// replaced by upstreamPrefix verbatim (query string preserved). Pass
// upstreamPrefix="" for an upstream that mounts its routes at the root.
func deFiProxyRewrite(upstreamEnv, fallback, groupPrefix, upstreamPrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := serviceURL(upstreamEnv, fallback)
		base, err := url.Parse(raw)
		if err != nil || base.Scheme == "" || base.Host == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "DeFi service URL misconfigured"})
			return
		}
		// Forward the original path (the services expose /api/v1/<group>/...).
		proxy := httputil.NewSingleHostReverseProxy(base)
		originalDirector := proxy.Director
		incomingPrefix := "/api/v1/" + groupPrefix
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			// Remap "/api/v1/<group>" -> upstreamPrefix verbatim (identity
			// when the prefixes match; root-mount when upstreamPrefix=="").
			if strings.HasPrefix(req.URL.Path, incomingPrefix) {
				req.URL.Path = upstreamPrefix + strings.TrimPrefix(req.URL.Path, incomingPrefix)
			}
			req.Host = base.Host
		}
		// Surface upstream errors honestly instead of masking them.
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			c.Error(err) //nolint:errcheck
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "DeFi service unavailable", "detail": err.Error()})
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// cardsProxy reverse-proxies the UserWallet /api/v1/cards/* surface to the
// canonical card_service (:8457), which keys one card account per user and
// mounts its routes at /api/v1/card/{balance,transactions,rates} (no :id).
// Clients address the per-card style (/cards/<id>/balance); the :id segment
// is the user's card-account alias and is dropped here (the account is
// derived from the authenticated JWT, so no account data can be spoofed).
func cardsProxy(upstreamEnv, fallback string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := serviceURL(upstreamEnv, fallback)
		base, err := url.Parse(raw)
		if err != nil || base.Scheme == "" || base.Host == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "card service URL misconfigured"})
			return
		}
		rest := c.Param("path") // e.g. "/rates", "/<id>/balance"
		op := rest
		if parts := strings.SplitN(strings.TrimPrefix(rest, "/"), "/", 2); len(parts) == 2 {
			op = "/" + parts[1] // drop the <id> segment
		}
		if op != "/balance" && op != "/transactions" && op != "/rates" {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown card operation"})
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(base)
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.URL.Path = "/api/v1/card" + op
			req.Host = base.Host
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			c.Error(err) //nolint:errcheck
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "card service unavailable", "detail": err.Error()})
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
