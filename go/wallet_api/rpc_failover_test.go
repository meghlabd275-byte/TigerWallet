package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
)

func TestEndpointsForIncludesPrimaryFirst(t *testing.T) {
	initRPCFailover()
	// Ethereum mainnet has verified alternates in the registry.
	eps := endpointsFor("https://api.mycryptoapi.com/eth")
	if len(eps) < 2 {
		t.Fatalf("expected primary + alternates, got %v", eps)
	}
	if eps[0] != "https://api.mycryptoapi.com/eth" {
		t.Errorf("primary must be tried first, got %s", eps[0])
	}
}

func TestEndpointsForDeduplicates(t *testing.T) {
	eps := endpointsFor("https://api.mycryptoapi.com/eth")
	seen := map[string]bool{}
	for _, e := range eps {
		if seen[e] {
			t.Errorf("endpoint %s appears twice in %v", e, eps)
		}
		seen[e] = true
	}
}

func TestEndpointsForUnknownEndpointIsItself(t *testing.T) {
	eps := endpointsFor("https://rpc.example.invalid")
	if len(eps) != 1 || eps[0] != "https://rpc.example.invalid" {
		t.Errorf("an endpoint with no alternates must yield only itself, got %v", eps)
	}
}

func TestEndpointsForEmptyIsNil(t *testing.T) {
	if eps := endpointsFor(""); eps != nil {
		t.Errorf("empty endpoint must yield nil, got %v", eps)
	}
}

// A JSON-RPC error means a node answered. Retrying it elsewhere would only
// replace a precise error with a slower, vaguer one.
func TestIsRetryableRPCError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"transport", errors.New("connection refused"), true},
		{"revert", rpc.Error(newRPCError(-32000, "execution reverted")), false},
		{"method not found", rpc.Error(newRPCError(-32601, "method does not exist")), false},
		{"rate limited", rpc.Error(newRPCError(-32005, "limit exceeded")), true},
		{"overloaded", rpc.Error(newRPCError(-32603, "internal error")), true},
	}
	for _, tc := range cases {
		if got := isRetryableRPCError(tc.err); got != tc.want {
			t.Errorf("%s: isRetryableRPCError = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// rpcError is the minimal rpc.Error implementation used by the table above.
type rpcError struct {
	code int
	msg  string
}

func (e rpcError) Error() string  { return e.msg }
func (e rpcError) ErrorCode() int { return e.code }

func newRPCError(code int, msg string) rpcError { return rpcError{code: code, msg: msg} }

// The point of the whole layer: a dead primary must not fail the call.
func TestRPCCallFallsBackToAlternate(t *testing.T) {
	var primaryHits, backupHits atomic.Int64
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits.Add(1)
		// Simulate a dead node: close without a response.
		hj, _ := w.(http.Hijacker)
		if conn, _, err := hj.Hijack(); err == nil {
			conn.Close()
		}
	}))
	defer primary.Close()
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backupHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0x1"}`)
	}))
	defer backup.Close()

	// Register the pair directly; the registry is not involved here.
	rpcFailoverOnce.Do(func() {})
	rpcFailover = map[string][]string{primary.URL: {backup.URL}}

	var out string
	err := rpcCall(context.Background(), primary.URL, func(client *rpc.Client) error {
		return client.CallContext(context.Background(), &out, "eth_chainId")
	})
	if err != nil {
		t.Fatalf("rpcCall should have succeeded on the alternate: %v", err)
	}
	if out != "0x1" {
		t.Errorf("result = %q, want 0x1", out)
	}
	if backupHits.Load() == 0 {
		t.Error("the alternate was never tried")
	}
	if primaryHits.Load() == 0 {
		t.Error("the primary should be tried first")
	}
}

// When every endpoint fails the error must still surface, never a silent zero.
func TestRPCCallReportsFailureWhenAllEndpointsFail(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, _ := w.(http.Hijacker)
		if conn, _, err := hj.Hijack(); err == nil {
			conn.Close()
		}
	}))
	defer dead.Close()

	rpcFailoverOnce.Do(func() {})
	rpcFailover = map[string][]string{dead.URL: {dead.URL + "/alt"}}

	var out string
	err := rpcCall(context.Background(), dead.URL, func(client *rpc.Client) error {
		return client.CallContext(context.Background(), &out, "eth_chainId")
	})
	if err == nil {
		t.Fatal("expected an error when every endpoint is unreachable")
	}
}
