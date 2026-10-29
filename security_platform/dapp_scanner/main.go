package main

// DApp Security Scanner — contract risk assessment, phishing detection and URL
// reputation.
//
// Contract verdicts are derived from the deployed bytecode read over JSON-RPC.
// A contract whose code cannot be read is reported as unscannable rather than
// given a clean score, and an address with no code is reported as an
// externally owned account rather than as a verified contract.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ErrNoRPCConfigured is returned when no chain endpoint is available.
var ErrNoRPCConfigured = errors.New("no chain RPC endpoint is configured (DAPP_SCANNER_RPC_URL or DAPP_SCANNER_RPC_URLS)")

// rpcTimeout bounds every upstream call.
const rpcTimeout = 15 * time.Second

// SecurityScanner holds the operator's blocklists and a cache of analysis
// results.
type SecurityScanner struct {
	mu        sync.RWMutex
	blocklist map[string]bool // hostname -> blocked
	addresses map[string]bool // address -> blocked
	contracts map[string]ContractAnalysis
	client    *http.Client
	rpcURLs   []string
}

// NewSecurityScanner builds a scanner from the environment. RPC endpoints are
// taken from DAPP_SCANNER_RPC_URLS (comma separated) or DAPP_SCANNER_RPC_URL.
func NewSecurityScanner() *SecurityScanner {
	var urls []string
	for _, raw := range strings.Split(os.Getenv("DAPP_SCANNER_RPC_URLS"), ",") {
		if v := strings.TrimSpace(raw); v != "" {
			urls = append(urls, v)
		}
	}
	if v := strings.TrimSpace(os.Getenv("DAPP_SCANNER_RPC_URL")); v != "" {
		urls = append(urls, v)
	}

	return &SecurityScanner{
		blocklist: map[string]bool{},
		addresses: map[string]bool{},
		contracts: map[string]ContractAnalysis{},
		client:    &http.Client{Timeout: rpcTimeout},
		rpcURLs:   urls,
	}
}

// ---------------------------------------------------------------------------
// JSON-RPC
// ---------------------------------------------------------------------------

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// callRPC issues a JSON-RPC request, trying each configured endpoint in turn so
// a single unavailable provider does not make the scanner unusable.
func (s *SecurityScanner) callRPC(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	if len(s.rpcURLs) == 0 {
		return nil, ErrNoRPCConfigured
	}

	payload, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return nil, fmt.Errorf("encode RPC request: %w", err)
	}

	var lastErr error
	for _, endpoint := range s.rpcURLs {
		reqCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("%s: %w", endpoint, err)
			continue
		}

		var decoded rpcResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&decoded)
		resp.Body.Close()
		cancel()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("%s: HTTP %d", endpoint, resp.StatusCode)
			continue
		}
		if decodeErr != nil {
			lastErr = fmt.Errorf("%s: decode response: %w", endpoint, decodeErr)
			continue
		}
		if decoded.Error != nil {
			lastErr = fmt.Errorf("%s: RPC error %d: %s", endpoint, decoded.Error.Code, decoded.Error.Message)
			continue
		}
		return decoded.Result, nil
	}

	if lastErr == nil {
		lastErr = ErrNoRPCConfigured
	}
	return nil, lastErr
}

// getCode fetches the deployed bytecode at an address.
func (s *SecurityScanner) getCode(ctx context.Context, address string) ([]byte, error) {
	raw, err := s.callRPC(ctx, "eth_getCode", address, "latest")
	if err != nil {
		return nil, err
	}
	var hexCode string
	if err := json.Unmarshal(raw, &hexCode); err != nil {
		return nil, fmt.Errorf("decode eth_getCode result: %w", err)
	}
	return decodeHexCode(hexCode)
}

// chainID returns the chain id reported by the endpoint, used to confirm the
// endpoint is reachable.
func (s *SecurityScanner) chainID(ctx context.Context) (string, error) {
	raw, err := s.callRPC(ctx, "eth_chainId")
	if err != nil {
		return "", err
	}
	var hexID string
	if err := json.Unmarshal(raw, &hexID); err != nil {
		return "", fmt.Errorf("decode eth_chainId result: %w", err)
	}
	value, err := hexQuantityToBigInt(hexID)
	if err != nil {
		return "", err
	}
	return value.String(), nil
}

// ---------------------------------------------------------------------------
// Contract scanning
// ---------------------------------------------------------------------------

// ScanContract analyzes the contract at an address. Results are cached only
// when the analysis succeeded, so a transient RPC failure does not pin an
// unscannable verdict for the cache lifetime.
func (s *SecurityScanner) ScanContract(ctx context.Context, address string) (ContractAnalysis, error) {
	if !isHexAddress(address) {
		return ContractAnalysis{}, fmt.Errorf("invalid contract address %q", address)
	}
	key := normalizeAddress(address)

	s.mu.RLock()
	cached, ok := s.contracts[key]
	s.mu.RUnlock()
	if ok {
		return cached, nil
	}

	code, err := s.getCode(ctx, key)
	if err != nil {
		return ContractAnalysis{}, err
	}

	analysis := analyzeBytecode(key, code)

	s.mu.Lock()
	s.contracts[key] = analysis
	s.mu.Unlock()

	return analysis, nil
}

// GetRiskScore returns the 0..100 score for an address, where a higher score
// means fewer risk signals. The score is 0 when the contract cannot be read, so
// an unscannable address is never presented as low risk.
func (s *SecurityScanner) GetRiskScore(ctx context.Context, address string) (int, error) {
	analysis, err := s.ScanContract(ctx, address)
	if err != nil {
		return 0, err
	}
	return analysis.Score, nil
}

// ---------------------------------------------------------------------------
// Blocklists
// ---------------------------------------------------------------------------

// AddToBlocklist blocks a hostname.
func (s *SecurityScanner) AddToBlocklist(host string) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocklist[host] = true
}

// RemoveFromBlocklist unblocks a hostname.
func (s *SecurityScanner) RemoveFromBlocklist(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.blocklist, strings.ToLower(strings.TrimSpace(host)))
}

// AddAddressToBlocklist blocks a contract or wallet address.
func (s *SecurityScanner) AddAddressToBlocklist(address string) error {
	if !isHexAddress(address) {
		return fmt.Errorf("invalid address %q", address)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addresses[normalizeAddress(address)] = true
	return nil
}

// IsAddressBlocked reports whether an address is on the blocklist.
func (s *SecurityScanner) IsAddressBlocked(address string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addresses[normalizeAddress(address)]
}

// blocklistSnapshot copies the host blocklist for use outside the lock.
func (s *SecurityScanner) blocklistSnapshot() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]bool, len(s.blocklist))
	for host := range s.blocklist {
		out[host] = true
	}
	return out
}

// CheckURL assesses a URL against the blocklist and structural heuristics.
func (s *SecurityScanner) CheckURL(rawURL string) URLVerdict {
	return checkURL(rawURL, s.blocklistSnapshot())
}

// ReportPhishing adds a host to the blocklist.
func (s *SecurityScanner) ReportPhishing(host string) {
	s.AddToBlocklist(extractHost(host))
}

func main() {
	scanner := NewSecurityScanner()

	if len(scanner.rpcURLs) == 0 {
		log.Println("dapp-scanner: no chain RPC endpoint configured; contract scanning will report an error until DAPP_SCANNER_RPC_URL is set")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Confirm the endpoint is usable at startup so a misconfiguration is
	// visible immediately rather than on the first scan.
	if len(scanner.rpcURLs) > 0 {
		probeCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
		if id, err := scanner.chainID(probeCtx); err != nil {
			log.Printf("dapp-scanner: chain endpoint probe failed: %v", err)
		} else {
			log.Printf("dapp-scanner: connected to chain id %s", id)
		}
		cancel()
	}

	for _, address := range strings.Split(os.Getenv("DAPP_SCANNER_BLOCKED_ADDRESSES"), ",") {
		if v := strings.TrimSpace(address); v != "" {
			if err := scanner.AddAddressToBlocklist(v); err != nil {
				log.Printf("dapp-scanner: ignoring invalid blocked address %q: %v", v, err)
			}
		}
	}
	for _, host := range strings.Split(os.Getenv("DAPP_SCANNER_BLOCKED_HOSTS"), ",") {
		if v := strings.TrimSpace(host); v != "" {
			scanner.AddToBlocklist(v)
		}
	}

	<-ctx.Done()
	log.Println("dapp-scanner: shutting down")
}
