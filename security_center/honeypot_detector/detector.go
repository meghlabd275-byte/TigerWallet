package honeypot

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/sha3"
)

// ============================================================================
// HONEYPOT DETECTOR - Scam Token Detection System
// Production-ready implementation for detecting honeypot and scam tokens
// ============================================================================

// Detector config
type Config struct {
	EthereumRPC   string
	BSCRPC        string
	PolygonRPC    string
	ArbitrumRPC   string
	OptimismRPC   string
	APITimeout    time.Duration
	CacheDuration time.Duration
	MinLiquidity  *big.Float
	MaxBuyTax     float64 // Maximum allowed buy tax percentage
	MaxSellTax    float64 // Maximum allowed sell tax percentage
}

// Token analysis result
type TokenAnalysis struct {
	TokenAddress  string     `json:"token_address"`
	TokenName     string     `json:"token_name"`
	TokenSymbol   string     `json:"token_symbol"`
	IsHoneypot    bool       `json:"is_honeypot"`
	RiskLevel     RiskLevel  `json:"risk_level"`
	BuyTax        float64    `json:"buy_tax"`
	SellTax       float64    `json:"sell_tax"`
	LiquidityUSD  float64    `json:"liquidity_usd"`
	HolderCount   int        `json:"holder_count"`
	TransferCount int        `json:"transfer_count"`
	Verified      bool       `json:"verified"`
	TopHoldersPct float64    `json:"top_holders_pct"`
	Flags         []RiskFlag `json:"risk_flags"`
	Timestamp     int64      `json:"timestamp"`
}

// Risk levels
type RiskLevel int

const (
	RiskLevelSafe RiskLevel = iota
	RiskLevelLow
	RiskLevelMedium
	RiskLevelHigh
	RiskLevelCritical
)

// Risk flags
type RiskFlag struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// Honeypot detector
type Detector struct {
	config     Config
	httpClient *http.Client
	cache      map[string]CacheEntry
	mu         sync.RWMutex
}

// Cache entry
type CacheEntry struct {
	Result     *TokenAnalysis
	Expiration time.Time
}

// New detector
func NewDetector(config Config) *Detector {
	if config.APITimeout == 0 {
		config.APITimeout = 30 * time.Second
	}
	if config.CacheDuration == 0 {
		config.CacheDuration = 15 * time.Minute
	}
	if config.MinLiquidity == nil {
		config.MinLiquidity = big.NewFloat(1000)
	}

	return &Detector{
		config: config,
		httpClient: &http.Client{
			Timeout: config.APITimeout,
		},
		cache: make(map[string]CacheEntry),
	}
}

// Analyze token for honeypot characteristics
func (d *Detector) AnalyzeToken(tokenAddress, chainID string) (*TokenAnalysis, error) {
	// Check cache
	cacheKey := fmt.Sprintf("%s:%s", chainID, tokenAddress)
	if cached := d.getCached(cacheKey); cached != nil {
		return cached, nil
	}

	// Perform analysis
	result, err := d.analyzeToken(tokenAddress, chainID)
	if err != nil {
		return nil, err
	}

	// Cache result
	d.setCached(cacheKey, result)

	return result, nil
}

// Analyze token
func (d *Detector) analyzeToken(tokenAddress, chainID string) (*TokenAnalysis, error) {
	result := &TokenAnalysis{
		TokenAddress: tokenAddress,
		Timestamp:    time.Now().Unix(),
		Flags:        []RiskFlag{},
	}

	// Get RPC URL
	rpcURL := d.getRPC(chainID)
	if rpcURL == "" {
		return nil, fmt.Errorf("unsupported chain: %s", chainID)
	}

	// Get token info
	tokenInfo, err := d.getTokenInfo(rpcURL, tokenAddress)
	if err != nil {
		result.Flags = append(result.Flags, RiskFlag{
			Type:        "INFO_FETCH_FAILED",
			Description: "Failed to fetch token information",
			Severity:    "high",
		})
		result.IsHoneypot = true
		result.RiskLevel = RiskLevelCritical
		return result, nil
	}

	result.TokenName = tokenInfo.Name
	result.TokenSymbol = tokenInfo.Symbol

	// Get liquidity info
	liquidity, err := d.getLiquidity(rpcURL, tokenAddress)
	if err != nil {
		// Liquidity could not be established. That is an unknown, not a finding
		// of zero liquidity: reporting $0 would wrongly describe a liquid token
		// whose pool this detector simply cannot price.
		result.Flags = append(result.Flags, RiskFlag{
			Type:        "LIQUIDITY_UNKNOWN",
			Description: fmt.Sprintf("liquidity could not be determined: %v", err),
			Severity:    "medium",
		})
	} else {
		result.LiquidityUSD = liquidity
		if liquidity < 1000 {
			result.Flags = append(result.Flags, RiskFlag{
				Type:        "LOW_LIQUIDITY",
				Description: fmt.Sprintf("Very low liquidity: $%.2f", liquidity),
				Severity:    "high",
			})
		}
	}

	// Get tax information
	buyTax, sellTax, err := d.getTaxInfo(rpcURL, tokenAddress)
	if err != nil {
		// An unreadable tax is not evidence of a high tax. It is recorded as
		// unverified so a caller can distinguish 'checked and clean' from
		// 'could not check'.
		result.Flags = append(result.Flags, RiskFlag{
			Type:        "TAX_UNKNOWN",
			Description: fmt.Sprintf("transaction taxes could not be read: %v", err),
			Severity:    "medium",
		})
	} else {
		result.BuyTax = buyTax
		result.SellTax = sellTax

		// Check for honeypot patterns
		if buyTax > d.config.MaxBuyTax || sellTax > d.config.MaxSellTax {
			result.IsHoneypot = true
			result.Flags = append(result.Flags, RiskFlag{
				Type:        "HIGH_TAX",
				Description: fmt.Sprintf("Suspicious taxes - Buy: %.1f%%, Sell: %.1f%%", buyTax, sellTax),
				Severity:    "critical",
			})
		}

		// Honeypot pattern: high sell tax but low buy tax
		if buyTax < 5 && sellTax > 20 {
			result.IsHoneypot = true
			result.Flags = append(result.Flags, RiskFlag{
				Type:        "HONEYPOT_PATTERN",
				Description: "Classic honeypot pattern: low buy tax, high sell tax",
				Severity:    "critical",
			})
		}
	}

	// Get holder information
	holders, err := d.getHolders(rpcURL, tokenAddress)
	if err == nil && len(holders) > 0 {
		result.HolderCount = len(holders)

		// Check holder concentration
		topHoldersPct := d.calculateTopHoldersPct(holders)
		result.TopHoldersPct = topHoldersPct

		if topHoldersPct > 80 {
			result.Flags = append(result.Flags, RiskFlag{
				Type:        "CENTRALIZED",
				Description: fmt.Sprintf("High holder concentration: %.1f%% in top holders", topHoldersPct),
				Severity:    "high",
			})
		}
	}

	// Get transfer count
	transfers, err := d.getTransferCount(rpcURL, tokenAddress)
	if err == nil {
		result.TransferCount = transfers

		if transfers < 10 {
			result.Flags = append(result.Flags, RiskFlag{
				Type:        "LOW_ACTIVITY",
				Description: fmt.Sprintf("Very few transfers: %d", transfers),
				Severity:    "medium",
			})
		}
	}

	// Check for common honeypot functions
	suspiciousFuncs, err := d.checkSuspiciousFunctions(rpcURL, tokenAddress)
	if err == nil && len(suspiciousFuncs) > 0 {
		result.Flags = append(result.Flags, RiskFlag{
			Type:        "SUSPICIOUS_CONTRACT",
			Description: "Contract contains potentially malicious functions",
			Severity:    "critical",
		})
		result.IsHoneypot = true
		for _, fn := range suspiciousFuncs {
			result.Flags = append(result.Flags, RiskFlag{
				Type:        "SUSPICIOUS_FUNC",
				Description: fmt.Sprintf("Suspicious function: %s", fn),
				Severity:    "critical",
			})
		}
	}

	// Check if verified
	// Publishing source requires an explorer API, which is not configured.
	// Reporting false would assert the contract is unverified, which is a claim
	// this detector cannot support; the flag is left false and a warning states
	// that verification was not checked.
	result.Verified = d.isVerified(tokenAddress)
	if !result.Verified {
		result.Flags = append(result.Flags, RiskFlag{
			Type:        "VERIFICATION_UNKNOWN",
			Description: "source verification was not checked; no explorer API is configured",
			Severity:    "low",
		})
	}

	// Determine final risk level
	result.RiskLevel = d.calculateRiskLevel(result)

	return result, nil
}

// Get RPC URL for chain
func (d *Detector) getRPC(chainID string) string {
	switch strings.ToLower(chainID) {
	case "1", "ethereum", "eth":
		return d.config.EthereumRPC
	case "56", "bsc", "binance":
		return d.config.BSCRPC
	case "137", "polygon", "matic":
		return d.config.PolygonRPC
	case "42161", "arbitrum":
		return d.config.ArbitrumRPC
	case "10", "optimism":
		return d.config.OptimismRPC
	default:
		return ""
	}
}

// Get token info from blockchain RPC
func (d *Detector) getTokenInfo(rpcURL, tokenAddress string) (*TokenInfo, error) {
	// Query ERC-20 token contract for metadata
	client := &http.Client{Timeout: 10 * time.Second}

	// Call name()
	nameData, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_call",
		"params": []interface{}{
			map[string]string{
				"to":   tokenAddress,
				"data": "0x06fdde03", // name() selector
			},
			"latest",
		},
		"id": 1,
	})
	if err != nil {
		return nil, err
	}

	nameReq, _ := http.NewRequest("POST", rpcURL, bytes.NewBuffer(nameData))
	nameReq.Header.Set("Content-Type", "application/json")
	nameResp, err := client.Do(nameReq)
	if err != nil {
		return nil, err
	}
	defer nameResp.Body.Close()

	var nameResult struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(nameResp.Body).Decode(&nameResult); err != nil {
		return nil, err
	}

	// Decode name from hex
	name := "Unknown Token"
	if len(nameResult.Result) > 2 {
		nameBytes, _ := hex.DecodeString(nameResult.Result[2:])
		// Remove null bytes
		name = string(bytes.Trim(nameBytes, "\x00"))
	}

	// Call symbol()
	symbolData, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_call",
		"params": []interface{}{
			map[string]string{
				"to":   tokenAddress,
				"data": "0x95d89b41", // symbol() selector
			},
			"latest",
		},
		"id": 1,
	})

	symbolReq, _ := http.NewRequest("POST", rpcURL, bytes.NewBuffer(symbolData))
	symbolReq.Header.Set("Content-Type", "application/json")
	symbolResp, err := client.Do(symbolReq)
	if err != nil {
		return nil, err
	}
	defer symbolResp.Body.Close()

	var symbolResult struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(symbolResp.Body).Decode(&symbolResult); err != nil {
		return nil, err
	}

	symbol := "UNKNOWN"
	if len(symbolResult.Result) > 2 {
		symbolBytes, _ := hex.DecodeString(symbolResult.Result[2:])
		symbol = string(bytes.Trim(symbolBytes, "\x00"))
	}

	// Call decimals()
	decimalsData, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_call",
		"params": []interface{}{
			map[string]string{
				"to":   tokenAddress,
				"data": "0x313ce567", // decimals() selector
			},
			"latest",
		},
		"id": 1,
	})

	decimalsReq, _ := http.NewRequest("POST", rpcURL, bytes.NewBuffer(decimalsData))
	decimalsReq.Header.Set("Content-Type", "application/json")
	decimalsResp, err := client.Do(decimalsReq)
	if err != nil {
		return nil, err
	}
	defer decimalsResp.Body.Close()

	var decimalsResult struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(decimalsResp.Body).Decode(&decimalsResult); err != nil {
		return nil, err
	}

	decimals := 18
	if len(decimalsResult.Result) > 2 {
		decimalsInt, _ := strconv.ParseInt(decimalsResult.Result[2:], 16, 64)
		decimals = int(decimalsInt)
	}

	return &TokenInfo{
		Name:     name,
		Symbol:   symbol,
		Decimals: decimals,
	}, nil
}

// Token info
type TokenInfo struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

// getLiquidity reports the USD value of the token's largest DEX pool.
//
// Reserves are read from the pair contract and priced with the pool's quote
// token. A fabricated liquidity figure is what makes a worthless token look
// tradeable, so a pool that cannot be read is reported as unknown rather than
// defaulted.
func (d *Detector) getLiquidity(rpcURL, tokenAddress string) (float64, error) {
	// Deriving a pair address requires the DEX factory and the pair token, and
	// pricing the reserves requires a quote for the pair token. Neither is
	// configured here, so liquidity is genuinely unknown.
	return 0, fmt.Errorf("liquidity cannot be determined without a configured DEX factory and price source")
}

// getTaxInfo reads the buy and sell tax from the contract.
//
// A tax is only reported when the contract exposes a getter that answers.
// Contracts that charge a fee without exposing it are indistinguishable from
// fee-free ones by static inspection, so an absent getter returns an error
// rather than a zero that would read as 'no tax'.
func (d *Detector) getTaxInfo(rpcURL, tokenAddress string) (buyTax, sellTax float64, err error) {
	client := &http.Client{Timeout: 10 * time.Second}

	getters := []struct {
		name  string
		isBuy bool
	}{
		{"buyFee()", true},
		{"sellFee()", false},
		{"_buyTax()", true},
		{"_sellTax()", false},
		{"buyTax()", true},
		{"sellTax()", false},
	}

	readable := false
	for _, getter := range getters {
		result, err := ethCall(client, rpcURL, tokenAddress, "0x"+hexSelector(getter.name))
		if err != nil || len(result) < 66 {
			continue
		}
		value, err := basisPointsToPercent(result, 100)
		if err != nil {
			continue
		}
		readable = true
		if getter.isBuy {
			buyTax = value
		} else {
			sellTax = value
		}
	}

	if !readable {
		return 0, 0, fmt.Errorf("the contract exposes no readable buy/sell tax getter")
	}
	return buyTax, sellTax, nil
}

// basisPointsToPercent converts a basis-points return value to a percentage,
// where 100 basis points is one percent.
func basisPointsToPercent(raw string, denominator float64) (float64, error) {
	value, err := decodeABIUint(raw)
	if err != nil {
		return 0, err
	}
	if denominator <= 0 {
		return 0, fmt.Errorf("invalid denominator")
	}
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(value), big.NewFloat(denominator)).Float64()
	// A tax above 100% is not a tax. Rejecting implausible values keeps a
	// misread call from being reported as a real tax.
	if f < 0 || f > 100 {
		return 0, fmt.Errorf("tax value out of range: %v", f)
	}
	return f, nil
}

// getHolders returns the holder set for a token.
//
// Enumerating holders requires an indexer; a node cannot list them. The method
// reports that plainly instead of returning an empty slice that the caller
// would treat as 'no holders'.
func (d *Detector) getHolders(rpcURL, tokenAddress string) ([]string, error) {
	return nil, fmt.Errorf("holder enumeration requires an indexer; none is configured")
}

// calculateTopHoldersPct returns the share held by the largest holders.
//
// With no holder data there is no concentration to report. A placeholder here
// previously produced a 50% concentration finding for every token, including
// ones with no holders at all.
func (d *Detector) calculateTopHoldersPct(holders []string) float64 {
	return 0
}

// getTransferCount counts Transfer events emitted by the contract.
//
// The count comes from the logs the node returns. A node that cannot serve the
// range yields an error so the caller reports 'unknown' rather than 'few
// transfers'.
func (d *Detector) getTransferCount(rpcURL, tokenAddress string) (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// keccak256("Transfer(address,address,uint256)")
	const transferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	payload, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_getLogs",
		"params": []map[string]interface{}{{
			"fromBlock": "0x0",
			"toBlock":   "latest",
			"address":   tokenAddress,
			"topics":    []interface{}{transferTopic},
		}},
	})
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var decoded struct {
		Result []json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return 0, err
	}
	if decoded.Error != nil {
		return 0, fmt.Errorf("eth_getLogs: %s", decoded.Error.Message)
	}
	return len(decoded.Result), nil
}

// checkSuspiciousFunctions reports the risk-bearing capabilities present in the
// contract's deployed code.
//
// Detection is by selector: signatures are hashed to their four-byte selectors
// and matched at PUSH4 dispatch sites, so a signature cannot drift from the ABI
// it claims to describe.
func (d *Detector) checkSuspiciousFunctions(rpcURL, tokenAddress string) ([]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	code, err := d.fetchCode(client, rpcURL, tokenAddress)
	if err != nil {
		return nil, err
	}
	if len(code) == 0 {
		return nil, fmt.Errorf("the address holds no contract code")
	}

	var found []string
	for signature, description := range suspiciousSignatures {
		if containsSelector(code, signature) {
			found = append(found, description)
		}
	}
	sort.Strings(found)
	return found, nil
}

// suspiciousSignatures maps a canonical signature to what its presence means.
var suspiciousSignatures = map[string]string{
	"setBlacklist(address,bool)":  "blacklist(address,bool) can block individual holders from transferring",
	"addBlackList(address)":       "addBlackList(address) can block individual holders from transferring",
	"destroyBlackFunds(address)":  "destroyBlackFunds(address) can seize a holder's balance",
	"pause()":                     "pause() can halt all transfers",
	"setMaxTxAmount(uint256)":     "setMaxTxAmount(uint256) caps how much can be moved per transaction",
	"setMaxWalletAmount(uint256)": "setMaxWalletAmount(uint256) caps how much one wallet may hold",
	"enableTrading()":             "enableTrading() implies trading can be disabled by the operator",
	"setFees(uint256)":            "setFees(uint256) allows transfer fees to change after purchase",
	"mint(address,uint256)":       "mint(address,uint256) allows the supply to be increased",
	"upgradeTo(address)":          "upgradeTo(address) allows the contract logic to be replaced",
}

// fetchCode reads deployed bytecode over JSON-RPC.
func (d *Detector) fetchCode(client *http.Client, rpcURL, address string) ([]byte, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_getCode",
		"params":  []interface{}{address, "latest"},
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var decoded struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if decoded.Error != nil {
		return nil, fmt.Errorf("eth_getCode: %s", decoded.Error.Message)
	}
	return decodeHex(decoded.Result)
}

// ethCall performs an eth_call and returns the raw hex result.
func ethCall(client *http.Client, rpcURL, to, data string) (string, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_call",
		"params":  []interface{}{map[string]string{"to": to, "data": data}, "latest"},
	})
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var decoded struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	if decoded.Error != nil {
		return "", fmt.Errorf("eth_call: %s", decoded.Error.Message)
	}
	return decoded.Result, nil
}

// containsSelector reports whether a four-byte selector appears at a PUSH4
// dispatch site in the code.
func containsSelector(code []byte, signature string) bool {
	selector := keccakSelector(signature)
	for i := 0; i+4 < len(code); i++ {
		if code[i] != 0x63 { // PUSH4
			continue
		}
		if code[i+1] == selector[0] && code[i+2] == selector[1] &&
			code[i+3] == selector[2] && code[i+4] == selector[3] {
			return true
		}
	}
	return false
}

// keccakSelector returns the four-byte selector for a canonical signature.
func keccakSelector(signature string) [4]byte {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(signature))
	var out [4]byte
	copy(out[:], h.Sum(nil)[:4])
	return out
}

// hexSelector returns the four-byte selector as hex without the 0x prefix.
func hexSelector(signature string) string {
	selector := keccakSelector(signature)
	return hex.EncodeToString(selector[:])
}

// decodeHex converts a 0x-prefixed hex string to bytes.
func decodeHex(raw string) ([]byte, error) {
	value := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(raw), "0x"), "0X")
	if value == "" {
		return nil, nil
	}
	if len(value)%2 != 0 {
		value = "0" + value
	}
	out := make([]byte, len(value)/2)
	for i := range out {
		hi, ok := hexDigit(value[i*2])
		if !ok {
			return nil, fmt.Errorf("invalid hex at offset %d", i*2)
		}
		lo, ok := hexDigit(value[i*2+1])
		if !ok {
			return nil, fmt.Errorf("invalid hex at offset %d", i*2+1)
		}
		out[i] = hi<<4 | lo
	}
	return out, nil
}

// hexDigit converts one hex character to its value.
func hexDigit(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}

// decodeABIUint decodes a 32-byte uint return value.
func decodeABIUint(raw string) (*big.Int, error) {
	value := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(raw), "0x"), "0X")
	if len(value) < 64 {
		return nil, fmt.Errorf("return value too short for a uint")
	}
	n, ok := new(big.Int).SetString(value[:64], 16)
	if !ok {
		return nil, fmt.Errorf("return value is not a valid uint")
	}
	return n, nil
}

// isVerified reports whether the token's source is published.
//
// Publishing source requires an explorer API, which is not configured here. The
// caller distinguishes 'unknown' from 'unverified' on this signal.
func (d *Detector) isVerified(tokenAddress string) bool {
	return false
}

// Calculate risk level
func (d *Detector) calculateRiskLevel(result *TokenAnalysis) RiskLevel {
	// Count critical flags
	criticalCount := 0
	highCount := 0

	for _, flag := range result.Flags {
		switch flag.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		}
	}

	if criticalCount > 0 {
		return RiskLevelCritical
	}
	if highCount > 2 {
		return RiskLevelHigh
	}
	if highCount > 0 || result.TopHoldersPct > 80 {
		return RiskLevelMedium
	}
	if result.LiquidityUSD < 1000 {
		return RiskLevelLow
	}

	return RiskLevelSafe
}

// Cache methods
func (d *Detector) getCached(key string) *TokenAnalysis {
	d.mu.RLock()
	defer d.mu.RUnlock()

	entry, exists := d.cache[key]
	if !exists || time.Now().After(entry.Expiration) {
		return nil
	}

	return entry.Result
}

func (d *Detector) setCached(key string, result *TokenAnalysis) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache[key] = CacheEntry{
		Result:     result,
		Expiration: time.Now().Add(d.config.CacheDuration),
	}
}

// Batch analyze multiple tokens
func (d *Detector) BatchAnalyze(tokenAddresses []string, chainID string) ([]*TokenAnalysis, error) {
	results := make([]*TokenAnalysis, 0, len(tokenAddresses))

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	for _, addr := range tokenAddresses {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()

			result, err := d.AnalyzeToken(address, chainID)
			mu.Lock()
			if err != nil {
				errors = append(errors, err)
			} else {
				results = append(results, result)
			}
			mu.Unlock()
		}(addr)
	}

	wg.Wait()

	if len(errors) > 0 && len(results) == 0 {
		return nil, errors[0]
	}

	return results, nil
}

// GetRiskLevelString returns risk level as string
func (r RiskLevel) String() string {
	switch r {
	case RiskLevelSafe:
		return "Safe"
	case RiskLevelLow:
		return "Low"
	case RiskLevelMedium:
		return "Medium"
	case RiskLevelHigh:
		return "High"
	case RiskLevelCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// ToJSON converts analysis to JSON
func (a *TokenAnalysis) ToJSON() (string, error) {
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
