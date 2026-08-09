// TigerWallet Wallet Service — DEPRECATED SHIM
//
// This service historically implemented wallet/key management with NIST P-256
// and a sha512(seed) "BIP-32" derivation. Both are INCORRECT for Ethereum
// (which requires secp256k1 + real HMAC-SHA512 BIP-32) and therefore insecure.
//
// The canonical, audited wallet backend is go/wallet_api (real BIP-39/BIP-32/
// BIP-44, secp256k1, EIP-1559/191/712 signing, AES-256-GCM seed encryption,
// PostgreSQL + Redis). It is the ONLY service that performs key management.
//
// To preserve API compatibility for any legacy caller, this binary now acts
// as a transparent reverse proxy to wallet_api. It performs NO key management,
// NO signing, and NO crypto of its own. Set WALLET_API_URL to the canonical
// backend (default http://localhost:8443).
//
// DO NOT add custom key/signing logic here. New features belong in wallet_api.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	target := strings.TrimRight(os.Getenv("WALLET_API_URL"), "/")
	if target == "" {
		target = "http://localhost:8443"
	}
	backend, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid WALLET_API_URL %q: %v", target, err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(backend)
			r.Out.Host = backend.Host
		},
		ModifyResponse: func(*http.Response) error { return nil },
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error for %s %s: %v", r.Method, r.URL.Path, err)
			http.Error(w, `{"error":"wallet-api backend unavailable"}`, http.StatusBadGateway)
		},
	}

	mux := http.NewServeMux()

	// Health/deprecation info served locally (not proxied).
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, fmt.Sprintf(
			`{"status":"healthy","service":"wallet-service","deprecated":true,`+
				`"replacement":"go/wallet_api","upstream":"%s","timestamp":%d}`,
			target, time.Now().Unix()))
	})

	// Everything else is proxied to the canonical wallet_api.
	mux.Handle("/", proxy)

	log.Printf("wallet-service (DEPRECATED shim) on :%s -> %s", port, target)
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
