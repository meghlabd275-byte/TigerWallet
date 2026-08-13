// main.go — MasterWallet deprecation reverse-proxy shim.
//
// The canonical MasterWallet backend now lives in ./backend/ (module
// github.com/tigerwallet/master-wallet-backend, port 8450). This file is a
// thin stdlib reverse-proxy so any legacy client that still points at this
// module's entry point gets transparently forwarded to the canonical backend.
//
// Build: go build -o master-wallet-shim .   (stdlib only, no external deps)
// Run:   PORT=8450 ./master-wallet-shim     (or MASTER_WALLET_BACKEND_URL)
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func main() {
	backend := os.Getenv("MASTER_WALLET_BACKEND_URL")
	if backend == "" {
		backend = "http://localhost:8450"
	}
	target, err := url.Parse(backend)
	if err != nil {
		log.Fatalf("invalid MASTER_WALLET_BACKEND_URL %q: %v", backend, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	// Preserve the original Host header for the backend.
	origDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		origDirector(r)
		r.Host = target.Host
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8451" // legacy port; canonical backend is 8450
	}

	log.Printf("MasterWallet deprecation shim -> %s on port %s", backend, port)
	if err := http.ListenAndServe(":"+port, proxy); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
