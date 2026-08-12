package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
)

// MetaMask eth-phishing-detect blocklist (raw GitHub). Cached for 1 hour.
const phishingListURL = "https://raw.githubusercontent.com/MetaMask/eth-phishing-detect/master/src/config.json"

type phishingConfig struct {
	Blacklist []string `json:"blacklist"`
	Fuzzy     []string `json:"fuzzylist"`
}

func fetchPhishingList(ctx context.Context) ([]string, []string, error) {
	// Try cache first.
	var cached phishingConfig
	if err := store.GetCache(ctx, "security:phishinglist", &cached); err == nil && len(cached.Blacklist) > 0 {
		return cached.Blacklist, cached.Fuzzy, nil
	}
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", phishingListURL, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	var cfg phishingConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, nil, fmt.Errorf("parse blocklist: %w", err)
	}
	_ = store.SetCache(ctx, "security:phishinglist", cfg, time.Hour)
	return cfg.Blacklist, cfg.Fuzzy, nil
}

func classifyHost(host string, blacklist, fuzzy []string) (string, []string) {
	host = strings.ToLower(strings.TrimSpace(host))
	reasons := []string{}
	for _, b := range blacklist {
		if strings.EqualFold(b, host) {
			reasons = append(reasons, "exact match on MetaMask blocklist")
			return "blocked", reasons
		}
	}
	for _, f := range fuzzy {
		if levenshtein(host, strings.ToLower(f)) <= 2 && len(host) > 3 {
			reasons = append(reasons, "fuzzy match on MetaMask fuzzylist: "+f)
			return "blocked", reasons
		}
	}
	return "benign", reasons
}

// levenshtein returns the edit distance between two strings.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	d := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		d[j] = j
	}
	for i := 1; i <= la; i++ {
		prev := d[0]
		d[0] = i
		for j := 1; j <= lb; j++ {
			tmp := d[j]
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d[j] = min(d[j]+1, d[j-1]+1, prev+cost)
			prev = tmp
		}
	}
	return d[lb]
}

func min(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

// handleCheckURL classifies a dApp URL against the MetaMask phishing blocklist.
func handleCheckURL(c *gin.Context) {
	raw := c.Query("url")
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "url query parameter required"})
		return
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid url"})
		return
	}
	blacklist, fuzzy, err := fetchPhishingList(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "blocklist unavailable"})
		return
	}
	classification, reasons := classifyHost(u.Host, blacklist, fuzzy)
	risk := "low"
	if classification == "blocked" {
		risk = "critical"
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"url": raw, "host": u.Host, "classification": classification,
			"risk_level": risk, "reasons": reasons, "checked_at": time.Now().Unix(),
		},
	})
}

// ethGetCode performs an eth_getCode lookup and returns the raw hex code string.
func ethGetCode(endpoint, addr string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	client, err := rpcClient(endpoint)
	if err != nil {
		return "", err
	}
	defer client.Close()
	var result hexutil.Bytes
	if err := client.CallContext(ctx, &result, "eth_getCode", addr, "latest"); err != nil {
		return "", err
	}
	return string(result), nil
}

// handleCheckAddress checks an address against the blocklist (some entries are
// 0x addresses) and performs an eth_getCode lookup to flag contracts.
func handleCheckAddress(c *gin.Context) {
	addr := strings.ToLower(strings.TrimSpace(c.Query("address")))
	if addr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "address query parameter required"})
		return
	}
	chainID := int64(1)
	if v := c.Query("chain_id"); v != "" {
		fmt.Sscanf(v, "%d", &chainID)
	}
	blacklist, _, err := fetchPhishingList(c.Request.Context())
	classification := "benign"
	reasons := []string{}
	if err == nil {
		for _, b := range blacklist {
			if strings.EqualFold(b, addr) {
				classification = "blocked"
				reasons = append(reasons, "address on MetaMask blocklist")
				break
			}
		}
	}
	// eth_getCode to detect contract accounts.
	isContract := false
	codeSize := 0
	if cfg := evmChainByChainID(chainID); cfg != nil {
		if code, err := ethGetCode(cfg.RPCEndpoint, addr); err == nil && len(code) > 2 {
			isContract = true
			codeSize = (len(code) - 2) / 2 // strip 0x
			if codeSize > 0 {
				reasons = append(reasons, fmt.Sprintf("address is a contract (%d bytes)", codeSize))
			}
		}
	}
	risk := "low"
	if classification == "blocked" {
		risk = "critical"
	} else if isContract {
		risk = "medium"
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"address": addr, "chain_id": chainID, "classification": classification,
			"risk_level": risk, "reasons": reasons, "is_contract": isContract,
			"code_size": codeSize, "checked_at": time.Now().Unix(),
		},
	})
}

// handleSecurityScan runs both checks given an address and/or url in the body.
type scanReq struct {
	Address string `json:"address"`
	URL     string `json:"url"`
	ChainID int64  `json:"chain_id"`
}

func handleSecurityScan(c *gin.Context) {
	var req scanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	result := gin.H{"checked_at": time.Now().Unix()}
	overall := "low"
	if req.Address != "" {
		blacklist, _, _ := fetchPhishingList(c.Request.Context())
		cls, reasons := classifyHost(req.Address, blacklist, nil)
		result["address"] = gin.H{"address": req.Address, "classification": cls, "reasons": reasons}
		if cls == "blocked" {
			overall = "critical"
		}
	}
	if req.URL != "" {
		u, err := url.Parse(req.URL)
		if err == nil && u.Host != "" {
			blacklist, fuzzy, _ := fetchPhishingList(c.Request.Context())
			cls, reasons := classifyHost(u.Host, blacklist, fuzzy)
			result["url"] = gin.H{"url": req.URL, "host": u.Host, "classification": cls, "reasons": reasons}
			if cls == "blocked" {
				overall = "critical"
			}
		}
	}
	result["risk_level"] = overall
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
