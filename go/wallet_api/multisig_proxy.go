package main

// multisig_proxy.go — UserWallet multisig proxy.
//
// UserWallet clients (extension/iOS/android/web) must never call the
// MasterWallet backend (:8450) directly — the separation rule keeps :8450
// service-to-service only. This handler reverse-proxies the multisig surface
// through the canonical UserWallet API (:8443):
//
//	/api/v1/wallet/multisig/wallets                    -> /api/v1/master-wallet/<id>/multisig/wallets
//	/api/v1/wallet/multisig/wallets/:wid               -> /api/v1/master-wallet/<id>/multisig/wallets/:wid
//	/api/v1/wallet/multisig/wallets/:wid/transactions  -> /api/v1/master-wallet/<id>/multisig/wallets/:wid/transactions
//	/api/v1/wallet/multisig/transactions/:tid/sign     -> /api/v1/master-wallet/<id>/multisig/transactions/:tid/sign
//	/api/v1/wallet/multisig/transactions/:tid/execute  -> /api/v1/master-wallet/<id>/multisig/transactions/:tid/execute
//
// Config (env):
//	MASTER_WALLET_URL    base URL of the master wallet backend
//	                     (default http://localhost:8450)
//	MASTER_WALLET_ID     default master wallet id (may be overridden per-request
//	                     with the master_wallet_id query param)
//	SERVICE_AUTH_TOKEN   service JWT sent as Bearer to the master wallet backend;
//	                     when unset the client's own Authorization is forwarded.

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// masterWalletMultisigProxy returns a gin handler that reverse-proxies the
// multisig API to the MasterWallet backend, remapping the incoming prefix
// /api/v1/wallet/multisig to /api/v1/master-wallet/<id>/multisig. The master
// wallet id comes from the master_wallet_id query param or MASTER_WALLET_ID.
func masterWalletMultisigProxy() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := serviceURL("MASTER_WALLET_URL", "http://localhost:8450")
		base, err := url.Parse(raw)
		if err != nil || base.Scheme == "" || base.Host == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "master wallet service URL misconfigured"})
			return
		}
		mwID := strings.TrimSpace(c.Query("master_wallet_id"))
		if mwID == "" {
			mwID = strings.TrimSpace(os.Getenv("MASTER_WALLET_ID"))
		}
		if mwID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "master_wallet_id is required (query param or MASTER_WALLET_ID env)"})
			return
		}
		serviceToken := strings.TrimSpace(os.Getenv("SERVICE_AUTH_TOKEN"))

		proxy := httputil.NewSingleHostReverseProxy(base)
		originalDirector := proxy.Director
		incomingPrefix := "/api/v1/wallet/multisig"
		upstreamPrefix := "/api/v1/master-wallet/" + mwID + "/multisig"
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			if strings.HasPrefix(req.URL.Path, incomingPrefix) {
				req.URL.Path = upstreamPrefix + strings.TrimPrefix(req.URL.Path, incomingPrefix)
			}
			req.Host = base.Host
			// Service-to-service auth: the master wallet backend must never see
			// end-user JWTs on :8450; use the service token when configured.
			if serviceToken != "" {
				req.Header.Set("Authorization", "Bearer "+serviceToken)
			}
		}
		// Surface upstream errors honestly instead of masking them.
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			c.Error(err) //nolint:errcheck
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "master wallet service unavailable", "detail": err.Error()})
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
