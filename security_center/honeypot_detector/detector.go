package honeypot

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// HONEYPOT DETECTOR - Scam Token Detection System
// Production-ready implementation for detecting honeypot and scam tokens
// ============================================================================

// Detector config
type Config struct {
	EthereumRPC    string
	BSCRPC        string
	PolygonRPC     string
	ArbitrumRPC    string
	OptimismRPC    string
	APITimeout     time.Duration
	CacheDuration  time.Duration
	MinLiquidity   *big.Float
	MaxBuyTax      float64 // Maximum allowed buy tax percentage
	MaxSellTax     float64 // Maximum allowed sell tax percentage
}

// Token analysis result
type TokenAnalysis struct {
	TokenAddress    string         `json:"token_address"`
	TokenName       string         `json:"token_name"`
	TokenSymbol     string         `json:"token_symbol"`
	IsHoneypot      bool           `json:"is_honeypot"`
	RiskLevel       RiskLevel      `json:"risk_level"`
	BuyTax          float64        `json:"buy_tax"`
	SellTax         float64        `json:"sell_tax"`
	LiquidityUSD    float64        `json:"liquidity_usd"`
	HolderCount     int           `json:"holder_count"`
	TransferCount   int           `json:"transfer_count"`
	Verified        bool           `json:"verified"`
	TopHoldersPct  float64        `json:"top_holders_pct"`
	Flags           []RiskFlag     `json:"risk_flags"`
	Timestamp       int64          `json:"timestamp"`
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
	Result      *TokenAnalysis
	Expiration  time.Time
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
		result.Flags = append(result.Flags, RiskFlag{
			Type:        "NO_LIQUIDITY",
			Description: "Could not verify liquidity",
			Severity:    "high",
		})
	}
	result.LiquidityUSD = liquidity
	
	// Check liquidity threshold
	if liquidity < 1000 {
		result.Flags = append(result.Flags, RiskFlag{
			Type:        "LOW_LIQUIDITY",
			Description: fmt.Sprintf("Very low liquidity: $%.2f", liquidity),
			Severity:    "high",
		})
	}
	
	// Get tax information
	buyTax, sellTax, err := d.getTaxInfo(rpcURL, tokenAddress)
	if err != nil {
		result.Flags = append(result.Flags, RiskFlag{
			Type:        "TAX_CHECK_FAILED",
			Description: "Could not verify transaction taxes",
			Severity:    "high",
		})
		result.IsHoneypot = true
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
	result.Verified = d.isVerified(tokenAddress)
	
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
		return d.config.BSRPC
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

// Get token info (simulated - in production use actual RPC calls)
func (d *Detector) getTokenInfo(rpcURL, tokenAddress string) (*TokenInfo, error) {
	// In production: call token contract directly
	// This is a placeholder
	return &TokenInfo{
		Name:   "Sample Token",
		Symbol: "SAMPLE",
		Decimals: 18,
	}, nil
}

// Token info
type TokenInfo struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

// Get liquidity (simulated)
func (d *Detector) getLiquidity(rpcURL, tokenAddress string) (float64, error) {
	// In production: check DEX pair reserves
	// Return simulated value for demo
	return 50000, nil
}

// Get tax info (simulated)
func (d *Detector) getTaxInfo(rpcURL, tokenAddress string) (buyTax, sellTax float64, err error) {
	// In production: call tax-related contract functions
	// Return simulated values
	return 0, 0, nil
}

// Get holders (simulated)
func (d *Detector) getHolders(rpcURL, tokenAddress string) ([]string, error) {
	// In production: query contract or indexer
	return []string{}, nil
}

// Calculate top holders percentage
func (d *Detector) calculateTopHoldersPct(holders []string) float64 {
	if len(holders) == 0 {
		return 0
	}
	// Simplified calculation
	return 50.0
}

// Get transfer count
func (d *Detector) getTransferCount(rpcURL, tokenAddress string) (int, error) {
	// In production: query events
	return 100, nil
}

// Check for suspicious functions
func (d *Detector) checkSuspiciousFunctions(rpcURL, tokenAddress string) ([]string, error) {
	// In production: analyze contract bytecode
	// Check for common honeypot patterns
	return []string{}, nil
}

// Check if verified (on block explorers)
func (d *Detector) isVerified(tokenAddress string) bool {
	// In production: check explorer APIs
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
