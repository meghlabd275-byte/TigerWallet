package main

// non_evm_http.go — bounded HTTP helpers shared by the non-EVM state fetchers
// and the UTXO indexers. Every call carries a context deadline and a response
// size cap so a hostile or stalled upstream cannot pin a signing worker.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// nonEVMHTTPClient is shared by the non-EVM REST/RPC helpers.
var nonEVMHTTPClient = &http.Client{Timeout: 25 * time.Second}

// maxNonEVMResponse caps how much of an upstream reply is read.
const maxNonEVMResponse = 1 << 20 // 1 MiB

// httpGetCtx issues a GET with a deadline and returns the body on 2xx.
func httpGetCtx(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	return doNonEVMRequest(req)
}

// postRawJSON issues a JSON POST with a deadline and returns the body on 2xx.
func postRawJSON(ctx context.Context, rawURL string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return doNonEVMRequest(req)
}

// httpGetWithHeaders issues a GET carrying the given headers, skipping empty
// values so an unset credential never produces a blank header.
func httpGetWithHeaders(ctx context.Context, rawURL string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		if strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}
	return doNonEVMRequest(req)
}

// doNonEVMRequest executes a request and normalizes transport/HTTP failures
// into errors that name the endpoint and the upstream message.
func doNonEVMRequest(req *http.Request) ([]byte, error) {
	resp, err := nonEVMHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", req.URL.Host, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxNonEVMResponse))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("%s HTTP %d: %s", req.URL.Host, resp.StatusCode, truncateForError(raw))
	}
	return raw, nil
}

// truncateForError bounds an upstream message included in an error.
func truncateForError(raw []byte) string {
	s := strings.TrimSpace(string(raw))
	if len(s) > 300 {
		return s[:300]
	}
	return s
}
