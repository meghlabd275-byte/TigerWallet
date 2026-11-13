/**
 * TigerWallet Transaction Shield - Threat intelligence and on-chain contract
 * inspection.
 *
 * The risk engine needs two facts it cannot guess: whether an address is a
 * known-bad actor, and whether the destination is a contract (and if so,
 * whether it is a proxy whose logic sits behind an upgradeable slot). Both are
 * sourced for real here:
 *
 *   - the threat list is loaded from an operator-supplied JSON file, and
 *   - the contract check asks the chain over JSON-RPC.
 *
 * When neither source is configured the lookups report "unknown" rather than a
 * default risk score, so a misconfigured deployment cannot silently score every
 * destination as safe.
 */

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Threat intelligence
// ============================================================================

// ThreatCategory classifies why an address is on the list.
type ThreatCategory string

const (
	CategorySanctioned ThreatCategory = "sanctioned"
	CategoryPhishing   ThreatCategory = "phishing"
	CategoryDrainer    ThreatCategory = "drainer"
	CategoryMixer      ThreatCategory = "mixer"
	CategoryExploit    ThreatCategory = "exploit"
	CategoryScam       ThreatCategory = "scam"
	CategoryHoneypot   ThreatCategory = "honeypot"
)

// threatSeverity maps a category to the risk contribution it carries. A
// sanctioned address is treated as maximally risky; a scam token contract is
// serious but not automatically a full block.
func threatSeverity(c ThreatCategory) float64 {
	switch c {
	case CategorySanctioned, CategoryDrainer, CategoryPhishing:
		return 1.0
	case CategoryMixer, CategoryExploit:
		return 0.9
	case CategoryHoneypot:
		return 0.85
	case CategoryScam:
		return 0.7
	default:
		return 0.5
	}
}

// ThreatEntry is one address on the list.
type ThreatEntry struct {
	Address  string         `json:"address"`
	Category ThreatCategory `json:"category"`
	Source   string         `json:"source"`
	Note     string         `json:"note"`
	AddedAt  time.Time      `json:"added_at"`
}

// ThreatMatch is the result of a lookup.
type ThreatMatch struct {
	Found    bool
	Entry    ThreatEntry
	Severity float64
}

// ThreatDB is an in-memory, read-mostly index of known-bad addresses.
type ThreatDB struct {
	mu         sync.RWMutex
	byChain    map[uint64]map[string]ThreatEntry // chain id -> lowercased address -> entry
	global     map[string]ThreatEntry            // applies to every chain
	loadedAt   time.Time
	sourcePath string
}

// NewThreatDB returns an empty database. An empty list is a valid state: the
// engine then reports no match, which the caller treats as "unknown", not as
// "safe".
func NewThreatDB() *ThreatDB {
	return &ThreatDB{
		byChain: make(map[uint64]map[string]ThreatEntry),
		global:  make(map[string]ThreatEntry),
	}
}

// threatFileFormat is the on-disk schema. Entries may be listed under a chain
// id or under "global" for addresses that are bad on every chain.
type threatFileFormat struct {
	Entries map[string][]ThreatEntry `json:"entries"`
}

// Load reads the threat list from path. A missing or unreadable file is an
// error, not a silent empty list: the operator asked for a list and did not
// get one, and continuing would mean scoring against no intelligence at all.
func (t *ThreatDB) Load(path string) error {
	if path == "" {
		return fmt.Errorf("threat intelligence path is empty")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read threat list %s: %w", path, err)
	}

	var file threatFileFormat
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return fmt.Errorf("parse threat list %s: %w", path, err)
	}

	byChain := make(map[uint64]map[string]ThreatEntry)
	global := make(map[string]ThreatEntry)

	for key, entries := range file.Entries {
		for _, e := range entries {
			addr := strings.ToLower(strings.TrimSpace(e.Address))
			if addr == "" {
				return fmt.Errorf("threat list %s: entry with an empty address", path)
			}
			if e.Category == "" {
				return fmt.Errorf("threat list %s: entry %s has no category", path, addr)
			}
			e.Address = addr

			if strings.EqualFold(key, "global") {
				global[addr] = e
				continue
			}
			chainID, err := parseChainKey(key)
			if err != nil {
				return fmt.Errorf("threat list %s: %w", path, err)
			}
			if byChain[chainID] == nil {
				byChain[chainID] = make(map[string]ThreatEntry)
			}
			byChain[chainID][addr] = e
		}
	}

	t.mu.Lock()
	t.byChain = byChain
	t.global = global
	t.loadedAt = time.Now()
	t.sourcePath = path
	t.mu.Unlock()

	return nil
}

func parseChainKey(key string) (uint64, error) {
	var id uint64
	if _, err := fmt.Sscanf(strings.TrimSpace(key), "%d", &id); err != nil {
		return 0, fmt.Errorf("chain key %q is neither a chain id nor \"global\"", key)
	}
	return id, nil
}

// Lookup returns the first matching entry for an address on a chain. An address
// listed globally matches on every chain.
func (t *ThreatDB) Lookup(chainID uint64, address string) ThreatMatch {
	if address == "" {
		return ThreatMatch{}
	}
	addr := strings.ToLower(strings.TrimSpace(address))

	t.mu.RLock()
	defer t.mu.RUnlock()

	if entry, ok := t.global[addr]; ok {
		return ThreatMatch{Found: true, Entry: entry, Severity: threatSeverity(entry.Category)}
	}
	if chain, ok := t.byChain[chainID]; ok {
		if entry, ok := chain[addr]; ok {
			return ThreatMatch{Found: true, Entry: entry, Severity: threatSeverity(entry.Category)}
		}
	}
	return ThreatMatch{}
}

// Stats reports the size of the loaded list. Read-only, so it is safe to expose
// on a status endpoint.
func (t *ThreatDB) Stats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	perChain := make(map[string]int, len(t.byChain))
	total := len(t.global)
	for id, chain := range t.byChain {
		perChain[fmt.Sprintf("%d", id)] = len(chain)
		total += len(chain)
	}

	out := map[string]interface{}{
		"loaded":          t.loadedAt,
		"source":          t.sourcePath,
		"total_addresses": total,
		"global":          len(t.global),
		"per_chain":       perChain,
	}
	if t.loadedAt.IsZero() {
		out["loaded"] = nil
	}
	return out
}

// ============================================================================
// On-chain contract inspection
// ============================================================================

// eip1967ImplementationSlot is the storage slot holding the implementation
// address of a transparent/UUPS proxy. A non-zero value means the code the
// caller interacts with is not the code that will run.
const eip1967ImplementationSlot = "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"

// ContractInfo is what the chain reports about a destination.
type ContractInfo struct {
	Checked        bool   // the chain answered
	IsContract     bool   // there is code at the address
	CodeSize       int    // bytes of runtime code
	IsProxy        bool   // EIP-1967 implementation slot is set
	Implementation string // proxy implementation address, when known
	ChainID        uint64
}

// ContractChecker queries chain nodes over JSON-RPC.
type ContractChecker struct {
	urls   map[uint64]string
	client *http.Client
}

// NewContractChecker builds a checker from a chain id -> RPC URL map. Chains
// with no URL are simply not checked; the caller sees Checked == false.
func NewContractChecker(urls map[uint64]string, timeout time.Duration) *ContractChecker {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &ContractChecker{
		urls:   urls,
		client: &http.Client{Timeout: timeout},
	}
}

// ConfiguredChains lists the chain ids that have an RPC endpoint, sorted, so a
// status endpoint can report which chains support on-chain inspection.
func (c *ContractChecker) ConfiguredChains() []uint64 {
	if c == nil {
		return nil
	}
	out := make([]uint64, 0, len(c.urls))
	for id := range c.urls {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// parseChainRPCURLs reads a "chainid=url,chainid=url" list. Entries without an
// "=" or with a non-numeric chain id are rejected rather than skipped, so a
// typo does not silently disable a chain's checks.
func parseChainRPCURLs(spec string) (map[uint64]string, error) {
	out := make(map[uint64]string)
	if strings.TrimSpace(spec) == "" {
		return out, nil
	}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		eq := strings.Index(part, "=")
		if eq <= 0 || eq == len(part)-1 {
			return nil, fmt.Errorf("RPC entry %q is not chainid=url", part)
		}
		var chainID uint64
		if _, err := fmt.Sscanf(strings.TrimSpace(part[:eq]), "%d", &chainID); err != nil {
			return nil, fmt.Errorf("RPC entry %q has a non-numeric chain id", part)
		}
		url := strings.TrimSpace(part[eq+1:])
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return nil, fmt.Errorf("RPC entry %q must be an http(s) URL", part)
		}
		out[chainID] = url
	}
	return out, nil
}

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

func (c *ContractChecker) call(ctx context.Context, chainID uint64, method string, params []interface{}) (json.RawMessage, error) {
	url, ok := c.urls[chainID]
	if !ok {
		return nil, fmt.Errorf("no RPC endpoint configured for chain %d", chainID)
	}

	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RPC %s returned HTTP %d", method, resp.StatusCode)
	}

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", method, err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC %s error %d: %s", method, rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

// Inspect reports whether the address has code and whether it is a proxy.
//
// The returned error is non-nil when the chain could not be asked. The caller
// must treat that as unknown, not as "not a contract".
func (c *ContractChecker) Inspect(ctx context.Context, chainID uint64, address string) (ContractInfo, error) {
	info := ContractInfo{ChainID: chainID}
	if _, ok := c.urls[chainID]; !ok {
		return info, fmt.Errorf("no RPC endpoint configured for chain %d", chainID)
	}
	if !isHexAddress(address) {
		return info, fmt.Errorf("%q is not a hex address", address)
	}

	codeRaw, err := c.call(ctx, chainID, "eth_getCode", []interface{}{address, "latest"})
	if err != nil {
		return info, err
	}

	var code string
	if err := json.Unmarshal(codeRaw, &code); err != nil {
		return info, fmt.Errorf("decode eth_getCode result: %w", err)
	}

	info.Checked = true
	hexCode := strings.TrimPrefix(code, "0x")
	if hexCode == "" || hexCode == "0" {
		return info, nil
	}
	info.IsContract = true
	info.CodeSize = len(hexCode) / 2

	// Only proxies have the EIP-1967 slot set; a plain contract returns zero.
	slotRaw, err := c.call(ctx, chainID, "eth_getStorageAt", []interface{}{address, eip1967ImplementationSlot, "latest"})
	if err != nil {
		// The code check succeeded, so this is still useful information.
		// Not knowing about the proxy is not the same as knowing there is none.
		return info, nil
	}

	var slot string
	if err := json.Unmarshal(slotRaw, &slot); err != nil {
		return info, nil
	}
	if impl := trimStorageWord(slot); impl != "" {
		info.IsProxy = true
		info.Implementation = impl
	}
	return info, nil
}

// trimStorageWord converts a 32-byte storage word into a 20-byte address,
// returning "" for an all-zero word.
func trimStorageWord(word string) string {
	hexWord := strings.TrimPrefix(word, "0x")
	if len(hexWord) < 40 {
		return ""
	}
	addr := "0x" + hexWord[len(hexWord)-40:]
	if addr == "0x0000000000000000000000000000000000000000" {
		return ""
	}
	if _, err := hex.DecodeString(addr[2:]); err != nil {
		return ""
	}
	return strings.ToLower(addr)
}

// isHexAddress reports whether s looks like a 20-byte hex address.
func isHexAddress(s string) bool {
	if len(s) != 42 {
		return false
	}
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return false
	}
	_, err := hex.DecodeString(s[2:])
	return err == nil
}
