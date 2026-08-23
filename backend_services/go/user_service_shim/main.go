// TigerWallet User Service — deprecated reverse-proxy shim.
//
// The canonical TigerWallet wallet backend is go/wallet_api (port 8443):
// REAL on-chain RPC, REAL BIP-39/32/44 HD derivation, REAL secp256k1
// signing + broadcast, AES-256-GCM encrypted-seed persistence (PostgreSQL
// + Redis). It is the ONLY service that performs key management and signing.
//
// The previous implementation of this service used INSECURE DIY crypto:
// `generateMnemonic` derived words via `entropy[i%len]%len(words)` (NOT
// BIP-39), `mnemonicToSeed` was a SHA-256 concat (NOT BIP-32/44),
// `deriveAddress` was SHA-256 (NOT secp256k1/Keccak), and `verifyTOTP` was a
// mere length check. That fake-crypto implementation has been removed and is
// retained only as legacy_main.go.txt for reference of its (non-crypto) data
// models.
//
// This service is now a thin reverse-proxy that forwards every request to the
// canonical wallet_api so legacy clients pointing at :8081 keep working while
// they migrate to :8443. It performs NO key handling and fabricates NO data —
// everything is delegated.
//
// Configure the upstream with WALLET_API_URL (default http://localhost:8443)
// and the listen port with PORT (default :8081).
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

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
			// Prefix /api/v1 if the client called a bare path (legacy clients).
			if !strings.HasPrefix(req.URL.Path, "/api/v1") && req.URL.Path != "/health" {
				if !strings.HasPrefix(req.URL.Path, "/") {
					req.URL.Path = "/" + req.URL.Path
				}
				req.URL.Path = "/api/v1" + req.URL.Path
			}
		},
		Transport: &http.Transport{
			MaxIdleConns:        256,
			MaxIdleConnsPerHost: 64,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	mux := http.NewServeMux()
	mux.Handle("/", proxy)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"user_services","delegated_to":"` + upstream + `"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = ":8081"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	log.Printf("user_services shim listening on %s -> %s (no key handling, no fabricated data)", port, upstream)
	srv := &http.Server{
		Addr:              port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
