// TigerWallet Bots Service — DEPRECATED SHIM
//
// Historically this binary implemented its own bot CRUD + market data with gin,
// pgx, and redis. That implementation was a broken, partial duplicate of the
// canonical bot backend in mm_bot_platform/bot_api, and it fabricated market
// prices ("67500.00" etc.) instead of returning real data.
//
// The canonical, audited bot backend is mm_bot_platform/bot_api (real
// PostgreSQL + Redis bot lifecycle, 18 bot types, subscriptions, fees, CEX/DEX
// connectors, admin management). It runs on port 8471.
//
// To preserve API compatibility for any legacy caller, this binary now acts as
// a transparent reverse proxy to bot_api. It performs NO database access, NO
// bot logic, and NO hardcoded/fake data of its own — every response is the
// real response from bot_api. Set BOT_API_URL to the canonical backend
// (default http://localhost:8471).
//
// DO NOT add bot/CRUD/market logic here. New features belong in
// mm_bot_platform/bot_api.
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
	target := strings.TrimRight(os.Getenv("BOT_API_URL"), "/")
	if target == "" {
		target = "http://localhost:8471"
	}
	backend, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid BOT_API_URL %q: %v", target, err)
	}

	// Unique port — do NOT reuse 8107 (that collides with the
	// project-party-frontend container which maps 8107:80).
	port := os.Getenv("BOTS_PORT")
	if port == "" {
		port = "8108"
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(backend)
			r.Out.Host = backend.Host
		},
		ModifyResponse: func(*http.Response) error { return nil },
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error for %s %s: %v", r.Method, r.URL.Path, err)
			http.Error(w, `{"error":"bot-api backend unavailable"}`, http.StatusBadGateway)
		},
	}

	mux := http.NewServeMux()

	// Health/deprecation info served locally (not proxied).
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, fmt.Sprintf(
			`{"status":"healthy","service":"bots-service","deprecated":true,`+
				`"replacement":"mm_bot_platform/bot_api","upstream":"%s","timestamp":%d}`,
			target, time.Now().Unix()))
	})

	// Everything else (including /api/v1/...) is proxied to the canonical bot_api.
	mux.Handle("/", proxy)

	log.Printf("bots-service (DEPRECATED shim) on :%s -> %s", port, target)
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
