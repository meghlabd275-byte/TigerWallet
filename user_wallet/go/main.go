// UserWallet service (deprecated shim).
//
// The canonical TigerWallet wallet backend is go/wallet_api (port 8443):
// REAL on-chain RPC, REAL BIP-39/32/44 HD derivation, REAL secp256k1
// signing + broadcast, AES-256-GCM encrypted-seed persistence (PostgreSQL
// + Redis). It is the ONLY service that performs key management and signing.
//
// This service is a thin reverse-proxy that forwards every request to the
// canonical wallet_api so legacy clients pointing at :8105 keep working
// while they migrate to :8443. It performs NO key handling and fabricates
// NO data — everything is delegated.
//
// Configure the upstream with WALLET_API_URL (default http://localhost:8443)
// and the listen port with PORT (default 8105).
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	upstream := os.Getenv("WALLET_API_URL")
	if upstream == "" {
		upstream = "http://localhost:8443"
	}
	target, err := url.Parse(upstream)
	if err != nil {
		log.Fatalf("invalid WALLET_API_URL %q: %v", upstream, err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8105"
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = target.Host
			// Keep the client's Authorization header as-is (JWT for wallet_api).
			// SetXForwarded already adds X-Forwarded-For/Host.
		},
		FlushInterval: 100 * time.Millisecond,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error for %s: %v", r.URL.Path, err)
			http.Error(w, `{"error":"wallet-api unavailable"}`, http.StatusBadGateway)
		},
	}

	mux := http.NewServeMux()

	// Deprecation notice at root (non-API path).
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"user_wallet","upstream":"` + upstream + `","note":"deprecated reverse-proxy to canonical wallet_api"}`))
	})

	// Everything under /api/ is proxied to the canonical wallet_api.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		// The legacy clients used /api/v1/wallet/* and /api/v1/transactions/*
		// (dead-handler routes). The canonical wallet_api serves /api/v1/*
		// directly. If the legacy path includes an injected "/wallet" segment
		// (e.g. /api/v1/wallet/balances), strip it so the request reaches the
		// real backend route (/api/v1/balances).
		if strings.HasPrefix(r.URL.Path, "/api/v1/wallet/") {
			r.URL.Path = "/api/v1/" + strings.TrimPrefix(r.URL.Path, "/api/v1/wallet/")
			r.URL.RawPath = ""
		}
		proxy.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("UserWallet (deprecated reverse-proxy) listening on :%s -> %s", port, upstream)
	log.Printf("Point clients directly at %s to skip this shim.", upstream)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
