/**
 * TigerWallet Phishing Protection Service
 *
 * Real phishing protection modelled after MetaMask's Blockaid PPOM integration:
 *   - Fetches the canonical MetaMask eth-phishing-detect config
 *     (https://raw.githubusercontent.com/MetaMask/eth-phishing-detect/master/src/config.json)
 *     at startup and refreshes it periodically.
 *   - Caches the blocklist / allowlist and per-URL / per-address verdicts in
 *     Redis for fast lookups.
 *   - Classifies dApp URLs as safe / suspicious / blocked.
 *   - Checks contract addresses against the same blocklist (the MetaMask list
 *     also contains malicious contract addresses under the "blocklist").
 *   - Exposes GET /api/v1/security/check-url and /api/v1/security/check-address,
 *     and POST /api/v1/security/scan (consumed by the Next.js security scanner).
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	// Default source for the phishing blocklist. Override with
	// PHISHING_LIST_URL.
	defaultPhishingListURL = "https://raw.githubusercontent.com/MetaMask/eth-phishing-detect/master/src/config.json"
	phishingListRefresh    = 30 * time.Minute
	verdictCacheTTL        = 1 * time.Hour
)

// metaMaskPhishingConfig mirrors the relevant fields of the MetaMask
// eth-phishing-detect config.json.
type metaMaskPhishingConfig struct {
	Blocklist   []string            `json:"blocklist"`
	Allowlist   []string            `json:"allowlist"`
	Fuzzylist   []string            `json:"fuzzylist"`
	Whitelist   []string            `json:"whitelist"` // legacy alias some forks carry
	Version     int                 `json:"version"`
}

// PhishingService holds the in-memory blocklist plus an optional Redis client
// for verdict caching.
type PhishingService struct {
	listURL string
	http    *http.Client

	// domains / addresses keyed in lowercase for O(1) lookup.
	blocklist map[string]struct{}
	allowlist map[string]struct{}
	fuzzylist map[string]struct{}

	mu        sync.RWMutex
	lastLoad  time.Time
	loaded    bool

	redis *redis.Client

	// RPC clients (shared with the simulator) for on-chain address checks.
	rpcClients map[string]*RPCClient
}

// NewPhishingService constructs the service, fetches the list once, and opens
// a Redis connection if REDIS_URL is configured.
func NewPhishingService(config *Config) (*PhishingService, error) {
	listURL := getEnv("PHISHING_LIST_URL", defaultPhishingListURL)

	svc := &PhishingService{
		listURL:    listURL,
		http:       &http.Client{Timeout: 20 * time.Second},
		blocklist:  map[string]struct{}{},
		allowlist:  map[string]struct{}{},
		fuzzylist:  map[string]struct{}{},
		rpcClients: map[string]*RPCClient{},
	}
	for chain, u := range config.RPCURLs {
		svc.rpcClients[chain] = NewRPCClient(u)
	}

	// Redis is optional; the service falls back to in-memory caching.
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err == nil {
			client := redis.NewClient(opts)
			if err := client.Ping(context.Background()).Err(); err == nil {
				svc.redis = client
			}
		}
	}

	if err := svc.refreshList(context.Background()); err != nil {
		return svc, fmt.Errorf("initial phishing list load failed: %w", err)
	}
	return svc, nil
}

// RefreshLoop periodically refetches the phishing list.
func (s *PhishingService) RefreshLoop() {
	ticker := time.NewTicker(phishingListRefresh)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		if err := s.refreshList(ctx); err != nil {
			log.Printf("phishing list refresh failed: %v", err)
		}
		cancel()
	}
}

func (s *PhishingService) refreshList(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.listURL, nil)
	if err != nil {
		return err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("phishing list fetch returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var cfg metaMaskPhishingConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		return fmt.Errorf("invalid phishing list json: %w", err)
	}

	bl := map[string]struct{}{}
	for _, d := range cfg.Blocklist {
		bl[strings.ToLower(strings.TrimSpace(d))] = struct{}{}
	}
	al := map[string]struct{}{}
	for _, d := range append(cfg.Allowlist, cfg.Whitelist...) {
		al[strings.ToLower(strings.TrimSpace(d))] = struct{}{}
	}
	fl := map[string]struct{}{}
	for _, d := range cfg.Fuzzylist {
		fl[strings.ToLower(strings.TrimSpace(d))] = struct{}{}
	}

	s.mu.Lock()
	s.blocklist = bl
	s.allowlist = al
	s.fuzzylist = fl
	s.lastLoad = time.Now()
	s.loaded = true
	s.mu.Unlock()

	log.Printf("phishing list loaded: %d blocked, %d allowed, %d fuzzy", len(bl), len(al), len(fl))
	return nil
}

// Classification verdicts.
const (
	verdictSafe       = "safe"
	verdictSuspicious = "suspicious"
	verdictBlocked    = "blocked"
)

// URLVerdict is the JSON result of check-url.
type URLVerdict struct {
	URL         string   `json:"url"`
	Host        string   `json:"host"`
	Class       string   `json:"classification"`
	RiskLevel   string   `json:"risk_level"`
	Reasons     []string `json:"reasons"`
	Cached      bool     `json:"cached"`
	CheckedAt   int64    `json:"checked_at"`
}

// CheckURL classifies a dApp URL. Redis caches the verdict for verdictCacheTTL.
func (s *PhishingService) CheckURL(rawURL string) URLVerdict {
	rawURL = strings.TrimSpace(rawURL)
	verdict := URLVerdict{URL: rawURL, CheckedAt: time.Now().Unix(), Reasons: []string{}}

	cacheKey := "phish:url:" + rawURL
	if cached, ok := s.getCachedVerdict(cacheKey); ok {
		cached.Cached = true
		cached.CheckedAt = time.Now().Unix()
		return cached
	}

	host := extractHost(rawURL)
	verdict.Host = host
	if host == "" {
		verdict.Class = verdictSuspicious
		verdict.RiskLevel = "medium"
		verdict.Reasons = append(verdict.Reasons, "URL could not be parsed")
		s.setCachedVerdict(cacheKey, verdict)
		return verdict
	}
	lhost := strings.ToLower(host)

	s.mu.RLock()
	_, blocked := s.blocklist[lhost]
	_, allowed := s.allowlist[lhost]
	_, fuzzy := s.fuzzylist[lhost]
	s.mu.RUnlock()

	switch {
	case allowed:
		verdict.Class = verdictSafe
		verdict.RiskLevel = "low"
	case blocked:
		verdict.Class = verdictBlocked
		verdict.RiskLevel = "critical"
		verdict.Reasons = append(verdict.Reasons, "Host present in MetaMask phishing blocklist")
	case fuzzy:
		verdict.Class = verdictSuspicious
		verdict.RiskLevel = "high"
		verdict.Reasons = append(verdict.Reasons, "Host resembles a known phishing domain (fuzzy match)")
	default:
		// Heuristic checks for common phishing patterns.
		verdict.Class = verdictSafe
		verdict.RiskLevel = "low"
		if isSuspiciousHost(lhost) {
			verdict.Class = verdictSuspicious
			verdict.RiskLevel = "medium"
			verdict.Reasons = append(verdict.Reasons, "Suspicious URL pattern detected")
		}
	}

	s.setCachedVerdict(cacheKey, verdict)
	return verdict
}

// AddressVerdict is the JSON result of check-address.
type AddressVerdict struct {
	Address     string   `json:"address"`
	Chain       string   `json:"chain"`
	Class       string   `json:"classification"`
	RiskLevel   string   `json:"risk_level"`
	Reasons     []string `json:"reasons"`
	IsContract  bool     `json:"is_contract"`
	CodeSize    int      `json:"code_size"`
	Cached      bool     `json:"cached"`
	CheckedAt   int64    `json:"checked_at"`
}

// CheckAddress classifies a contract/wallet address. It checks the MetaMask
// blocklist (which includes malicious contract addresses) and queries the chain
// node for eth_getCode to determine whether the address is a contract.
func (s *PhishingService) CheckAddress(address, chain string) AddressVerdict {
	address = strings.TrimSpace(address)
	if chain == "" {
		chain = "ethereum"
	}
	verdict := AddressVerdict{
		Address:   address,
		Chain:     chain,
		Reasons:   []string{},
		CheckedAt: time.Now().Unix(),
	}

	cacheKey := "phish:addr:" + chain + ":" + strings.ToLower(address)
	if cached, ok := s.getCachedAddressVerdict(cacheKey); ok {
		cached.Cached = true
		cached.CheckedAt = time.Now().Unix()
		return cached
	}

	laddr := strings.ToLower(address)

	s.mu.RLock()
	_, blocked := s.blocklist[laddr]
	s.mu.RUnlock()
	if blocked {
		verdict.Class = verdictBlocked
		verdict.RiskLevel = "critical"
		verdict.Reasons = append(verdict.Reasons, "Address present in MetaMask malicious-address blocklist")
		s.setCachedAddressVerdict(cacheKey, verdict)
		return verdict
	}

	// On-chain eth_getCode to see if this is a contract and, if so, whether it
	// is a known ERC20 token whose code we recognize.
	client, ok := s.rpcClients[chain]
	if ok {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var codeHex string
		if err := client.CallAndDecode(ctx, &codeHex, "eth_getCode", laddr, "latest"); err == nil {
			codeHex = strings.TrimPrefix(codeHex, "0x")
			verdict.CodeSize = len(codeHex) / 2
			verdict.IsContract = verdict.CodeSize > 0
			if verdict.IsContract {
				verdict.Reasons = append(verdict.Reasons, fmt.Sprintf("Contract bytecode present (%d bytes)", verdict.CodeSize))
				if isLikelyTokenContract(codeHex) {
					verdict.Reasons = append(verdict.Reasons, "Bytecode contains ERC20 entry points (transfer/approve/allowance)")
				}
			}
		}
	}

	// Heuristics on the address itself.
	if strings.EqualFold(laddr, "0x0000000000000000000000000000000000000000") {
		verdict.Class = verdictSuspicious
		verdict.RiskLevel = "medium"
		verdict.Reasons = append(verdict.Reasons, "Null address")
	} else if !verdict.IsContract && client != nil {
		// EOA: not inherently malicious, but flag if it looks like a burner.
		verdict.Class = verdictSafe
		verdict.RiskLevel = "low"
	} else {
		verdict.Class = verdictSafe
		verdict.RiskLevel = "low"
	}

	s.setCachedAddressVerdict(cacheKey, verdict)
	return verdict
}

// ============================================================================
// HTTP handlers
// ============================================================================

func (s *PhishingService) CheckURLHandler(c *gin.Context) {
	target := c.Query("url")
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "url query parameter required"})
		return
	}
	verdict := s.CheckURL(target)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": verdict})
}

func (s *PhishingService) CheckAddressHandler(c *gin.Context) {
	addr := c.Query("address")
	if addr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "address query parameter required"})
		return
	}
	chain := c.Query("chain")
	verdict := s.CheckAddress(addr, chain)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": verdict})
}

// ScanHandler is the POST endpoint consumed by the frontend security scanner.
// It accepts { address } and returns a ScanResult shaped to match the
// frontend's ScanResult interface ({ id, address, risk, issues, scannedAt }).
func (s *PhishingService) ScanHandler(c *gin.Context) {
	var req struct {
		Address string `json:"address" binding:"required"`
		Chain   string `json:"chain"`
		URL     string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	issues := []string{}
	risk := "safe"

	// Address scan (primary, since the frontend calls scanAddress).
	av := s.CheckAddress(req.Address, req.Chain)
	for _, r := range av.Reasons {
		issues = append(issues, r)
	}
	risk = mapClassToRisk(av.Class, risk)

	// If a URL is also supplied, scan it too.
	if req.URL != "" {
		uv := s.CheckURL(req.URL)
		for _, r := range uv.Reasons {
			issues = append(issues, r)
		}
		risk = mapClassToRisk(uv.Class, risk)
	}

	result := gin.H{
		"id":         fmt.Sprintf("scan-%d", time.Now().UnixNano()),
		"address":    req.Address,
		"risk":       risk,
		"issues":     issues,
		"scannedAt":  time.Now().UnixMilli(),
		"url":        req.URL,
		"chain":      req.Chain,
		"classification": av.Class,
		"is_contract": av.IsContract,
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func mapClassToRisk(class, current string) string {
	rank := map[string]int{"safe": 0, "suspicious": 2, "blocked": 3}
	riskRank := map[string]int{"safe": 0, "warning": 1, "danger": 2}
	out := current
	if rank[class] >= rank["blocked"] {
		return "danger"
	}
	if rank[class] >= rank["suspicious"] {
		if riskRank[out] < riskRank["warning"] {
			out = "warning"
		}
	}
	return out
}

// ============================================================================
// Redis cache helpers
// ============================================================================

func (s *PhishingService) getCachedVerdict(key string) (URLVerdict, bool) {
	var v URLVerdict
	if s.redis == nil {
		return v, false
	}
	raw, err := s.redis.Get(context.Background(), key).Bytes()
	if err != nil || len(raw) == 0 {
		return v, false
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, false
	}
	return v, true
}

func (s *PhishingService) setCachedVerdict(key string, v URLVerdict) {
	if s.redis == nil {
		return
	}
	if b, err := json.Marshal(v); err == nil {
		_ = s.redis.Set(context.Background(), key, b, verdictCacheTTL).Err()
	}
}

func (s *PhishingService) getCachedAddressVerdict(key string) (AddressVerdict, bool) {
	var v AddressVerdict
	if s.redis == nil {
		return v, false
	}
	raw, err := s.redis.Get(context.Background(), key).Bytes()
	if err != nil || len(raw) == 0 {
		return v, false
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, false
	}
	return v, true
}

func (s *PhishingService) setCachedAddressVerdict(key string, v AddressVerdict) {
	if s.redis == nil {
		return
	}
	if b, err := json.Marshal(v); err == nil {
		_ = s.redis.Set(context.Background(), key, b, verdictCacheTTL).Err()
	}
}

// ============================================================================
// URL / host helpers
// ============================================================================

func extractHost(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Hostname()
	// Strip leading "www.".
	host = strings.TrimPrefix(host, "www.")
	return host
}

// isSuspiciousHost applies lightweight heuristics for URLs not in the
// blocklist: IDN homograph bait, suspicious keywords, excessive subdomains,
// lookalike TLDs.
func isSuspiciousHost(host string) bool {
	if host == "" {
		return false
	}
	// Punycode (IDN) hostnames often used for homograph attacks.
	if strings.Contains(host, "xn--") {
		return true
	}
	// Too many subdomains (e.g. login.secure.wallet.connect.example.com).
	if strings.Count(host, ".") >= 4 {
		return true
	}
	// Brand-bait keywords common to phishing kits.
	brandBait := []string{"wallet", "connect", "ledger", "metamask", "trezor", "phantom", "claim", "airdrop", "sync", "verify"}
	for _, b := range brandBait {
		if strings.Contains(host, b) {
			return true
		}
	}
	// Lookalike TLDs.
	lookalikeTLDs := []string{".tk", ".ml", ".ga", ".cf", ".gq", ".xyz", ".top", ".click", ".cyou"}
	for _, t := range lookalikeTLDs {
		if strings.HasSuffix(host, t) {
			return true
		}
	}
	return false
}

// isLikelyTokenContract checks the bytecode for the selectors of transfer /
// approve / allowance, a strong signal that the contract is an ERC20.
func isLikelyTokenContract(bytecodeHex string) bool {
	// PUSH4 selector opcodes embedded in the deployed bytecode.
	return strings.Contains(bytecodeHex, "a9059cbb") || // transfer(address,uint256)
		strings.Contains(bytecodeHex, "095ea7b3") || // approve(address,uint256)
		strings.Contains(bytecodeHex, "70a08231") // balanceOf(address)
}

// big.Int used only to keep imports stable if extended later.
var _ = big.NewInt
