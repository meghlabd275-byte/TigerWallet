// Master Wallet Service (services) — deprecation reverse-proxy shim.
//
// This was previously a 2283-line duplicate master-wallet service using broken
// crypto (NIST P-256 instead of secp256k1; SHA-256 instead of Keccak-256 for
// Ethereum address derivation; XOR "encryption"; a JWT path with no real key
// management). The canonical master-wallet backend is go/wallet_api on :8443
// (real BIP-39/BIP-32/BIP-44, real secp256k1, real on-chain broadcast). This
// file is kept only as a transparent reverse proxy so any legacy caller of the
// master-wallet service surface (auth, master/sub-wallets, transactions,
// auto-sign rules, fees, policies) is forwarded to wallet_api instead of hitting
// the removed fake-crypto implementation.
//
// Configure via env: WALLET_API_URL (default http://localhost:8443),
// PORT (default 8081).
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func main() {
	target := os.Getenv("WALLET_API_URL")
	if target == "" {
		target = "http://localhost:8443"
	}
	upstream, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid WALLET_API_URL %q: %v", target, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(upstream)
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = upstream.Host
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"master-wallet-services-shim","upstream":"` + target + `"}`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only the /api/v1 surface is proxied to the canonical wallet_api.
		if !strings.HasPrefix(r.URL.Path, "/api/v1") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found","hint":"this shim proxies /api/v1/* to the canonical wallet_api"}`))
			return
		}
		proxy.ServeHTTP(w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("master-wallet services shim listening on :%s, proxying to %s", port, target)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
