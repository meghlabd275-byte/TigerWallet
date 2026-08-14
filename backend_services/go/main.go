// TigerWallet backend-services (deprecated shim).
//
// The canonical TigerWallet wallet backend is go/wallet_api (port 8443):
// REAL on-chain RPC, REAL BIP-39/32/44 HD derivation, REAL secp256k1 signing
// + broadcast, AES-256-GCM encrypted-seed persistence (PostgreSQL + Redis),
// real CoinGecko prices, real Etherscan history, real gas/chain registry.
// It is the ONLY service that performs key management and signing.
//
// This module previously hosted an in-memory mock backend (hardcoded
// blockchains/tokens, hardcoded admin credentials admin@tigerwallet.com /
// admin123, fake P-256+sha256 crypto, fabricated tx ids, no DB/RPC). That
// implementation has been removed because it fabricated data and used
// insecure DIY crypto. This file is now a thin stdlib reverse-proxy that
// forwards every request to the canonical wallet_api so legacy clients
// pointing at :8080 keep working while they migrate to :8443. It performs NO
// key handling and fabricates NO data — everything is delegated.
//
// Configure the upstream with WALLET_API_URL (default http://localhost:8443)
// and the listen port with PORT (default 8080).
package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
		port = "8080"
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = target.Host
			// Keep the client's Authorization header as-is (JWT for wallet_api).
		},
		FlushInterval: 100 * time.Millisecond,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error for %s: %v", r.URL.Path, err)
			http.Error(w, `{"error":"wallet-api unavailable"}`, http.StatusBadGateway)
		},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"backend_services","upstream":"` + upstream + `","note":"deprecated reverse-proxy to canonical wallet_api (:8443)"}`))
	})

	// Everything under /api/ is proxied to the canonical wallet_api.
	// Legacy /api/v1/blockchains -> /api/v1/chains, /api/v1/tokens ->
	// /api/v1/tokens/registry, matching the canonical route surface.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/blockchains"):
			r.URL.Path = strings.Replace(r.URL.Path, "/api/v1/blockchains", "/api/v1/chains", 1)
			r.URL.RawPath = ""
		case strings.HasPrefix(r.URL.Path, "/api/v1/tokens") && r.URL.Path != "/api/v1/tokens/registry":
			r.URL.Path = strings.Replace(r.URL.Path, "/api/v1/tokens", "/api/v1/tokens/registry", 1)
			r.URL.RawPath = ""
		}
		proxy.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("backend_services reverse-proxy listening on :%s -> %s", port, upstream)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("exited")
}
