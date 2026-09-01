package main

// non_evm_util.go — shared encoding helpers for the non-EVM SDK families.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// hexOf concatenates hex of the parts.
func hexOf(parts ...[]byte) string {
	var out strings.Builder
	for _, p := range parts {
		out.WriteString(fmt.Sprintf("%x", p))
	}
	return out.String()
}

// base64Decode wraps StdEncoding.
func base64Decode(s string) ([]byte, error) { return base64.StdEncoding.DecodeString(s) }

// utilsBase64 encodes raw bytes with StdEncoding as a string.
func utilsBase64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

// hexStringDecode decodes a hex string (with or without 0x).
func hexStringDecode(s string) ([]byte, error) {
	if len(s) >= 2 && s[:2] == "0x" {
		s = s[2:]
	}
	return hex.DecodeString(s)
}

// u128BE computes the 16-byte big-endian of a U128 byte slice.
func u128BE(b []byte) []byte {
	out := make([]byte, 16)
	if len(b) >= 16 {
		copy(out, b[len(b)-16:])
	} else {
		copy(out[16-len(b):], b)
	}
	return out
}

// nvmHTTP is a bounded client shared by all non-EVM SDK REST/JSON-RPC calls.
var nvmHTTP = &http.Client{Timeout: 25 * time.Second}

// postRaw POSTs a raw body with JSON content-type and returns the body.
func postRaw(ctx context.Context, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := nvmHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw[:minInt(len(raw), 300)])))
	}
	return raw, nil
}

// getRaw GETs a URL and returns the body on 2xx, fail-closed otherwise.
func getRaw(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := nvmHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw[:minInt(len(raw), 300)])))
	}
	return raw, nil
}

// unmarshalCheckError unmarshals raw into dst; nil dst is a pure-check.
func unmarshalCheckError(raw []byte, dst interface{}) error {
	if dst == nil {
		if !json.Valid(raw) {
			return fmt.Errorf("malformed JSON reply")
		}
		return nil
	}
	return json.Unmarshal(raw, dst)
}

// newRequestBytes builds a POST request with a raw body.
func newRequestBytes(ctx context.Context, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// doRead executes the request and returns the body, fail-closed on non-2xx.
func doRead(req *http.Request) ([]byte, error) {
	resp, err := nvmHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw[:minInt(len(raw), 300)])))
	}
	return raw, nil
}

// postFormURLEncoded POSTs an application/x-www-form-urlencoded body.
func postFormURLEncoded(ctx context.Context, url, form string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(form))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return doRead(req)
}
