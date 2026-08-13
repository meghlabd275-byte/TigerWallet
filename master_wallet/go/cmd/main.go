// Master Wallet Service (port 8080) — deprecation reverse-proxy shim.
//
// The canonical master-wallet backend is go/wallet_api on :8443, which performs
// real BIP-39/BIP-32/BIP-44 key derivation, real secp256k1 transaction signing,
// and real on-chain broadcast. This file is retained only so legacy clients that
// still call :8080 are transparently proxied to the canonical service instead of
// hitting the previous thin-GORM stub that used fake crypto (SHA-256 address
// derivation, XOR "encryption", a fabricated SHA-256 transaction hash, and a
// hasEnoughBalance that always returned true). Those are GONE; this shim
// forwards every request to wallet_api with no key handling of its own.
//
// Configure via env: WALLET_API_URL (default http://localhost:8443),
// PORT (default 8080).
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
		_, _ = w.Write([]byte(`{"status":"ok","service":"master-wallet-shim","upstream":"` + target + `"}`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// This shim only proxies the wallet API surface (/api/v1/*).
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
		port = "8080"
	}
	log.Printf("master-wallet shim listening on :%s, proxying to %s", port, target)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
