package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// captureLog runs fn with the standard logger redirected, and returns what was
// written. The startup check communicates only through the log, so this is the
// only way to assert what an operator would actually see.
func captureLog(fn func()) string {
	var buf bytes.Buffer
	prev := log.Writer()
	flags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(prev)
		log.SetFlags(flags)
	}()
	fn()
	return buf.String()
}

func nodeReporting(chainID string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"` + chainID + `"}`))
	}))
}

func TestPlatformChainCheckConfirmsMatchingChain(t *testing.T) {
	srv := nodeReporting("0x1e1e") // 7710
	defer srv.Close()

	out := captureLog(func() { verifyPlatformChain(context.Background(), srv.URL) })
	if !strings.Contains(out, "confirmed on chain 7710") {
		t.Fatalf("log = %q, want a confirmation naming chain 7710", out)
	}
	if strings.Contains(out, "WARNING") {
		t.Fatalf("log = %q, want no warning for a matching chain", out)
	}
}

func TestPlatformChainCheckWarnsOnMismatch(t *testing.T) {
	srv := nodeReporting("0x1") // a node on Ethereum
	defer srv.Close()

	out := captureLog(func() { verifyPlatformChain(context.Background(), srv.URL) })
	if !strings.Contains(out, "WARNING") {
		t.Fatalf("log = %q, want a warning", out)
	}
	if !strings.Contains(out, "reports chain 1") || !strings.Contains(out, "built for chain 7710") {
		t.Fatalf("log = %q, want it to name both the reported and the expected chain", out)
	}
	if !strings.Contains(out, "TSC_CHAIN_ID=7710") {
		t.Fatalf("log = %q, want the remediation to be stated", out)
	}
}

func TestPlatformChainCheckWarnsWhenEndpointSilent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	out := captureLog(func() { verifyPlatformChain(context.Background(), srv.URL) })
	if !strings.Contains(out, "did not answer eth_chainId") {
		t.Fatalf("log = %q, want a warning that the endpoint did not answer", out)
	}
}

// An unset endpoint is the normal case for a wallet that has not been pointed
// at a chain yet, so it must stay quiet rather than warn on every boot.
func TestPlatformChainCheckStaysQuietWithoutEndpoint(t *testing.T) {
	out := captureLog(func() { verifyPlatformChain(context.Background(), "") })
	if strings.TrimSpace(out) != "" {
		t.Fatalf("log = %q, want no output when no endpoint is configured", out)
	}
}

func TestPlatformChainEndpointPrefersDedicatedVariable(t *testing.T) {
	t.Setenv(tscRPCEnv, "http://dedicated:8604")
	t.Setenv("CHAIN_7710_RPC", "http://generic:8604")
	if got := platformChainEndpoint(); got != "http://dedicated:8604" {
		t.Fatalf("endpoint = %q, want the dedicated variable to win", got)
	}
}

func TestPlatformChainEndpointFallsBackToGeneric(t *testing.T) {
	t.Setenv(tscRPCEnv, "")
	t.Setenv("CHAIN_7710_RPC", "http://generic:8604")
	if got := platformChainEndpoint(); got != "http://generic:8604" {
		t.Fatalf("endpoint = %q, want the generic per-chain variable", got)
	}
}
