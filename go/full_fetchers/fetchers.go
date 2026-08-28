/**
 * TigerWallet Full Fetchers - Go Distributed/High-Load Implementation
 *
 * This package implements all 20 fetchers in Go for distributed systems:
 * - 6 Standard Fetchers
 * - 14 Advanced Fetchers
 *
 * Built with Go for high load and distributed systems
 *
 * @author TigerWallet Team
 * @version 1.0.0
 */

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// TYPE DEFINITIONS
// =============================================================================

type Timestamp int64
type ChainID uint64
type GasPrice uint64
type TokenAmount string
type Address string

// Token metadata
type TokenMetadata struct {
	Address     Address   `json:"address"`
	Name        string    `json:"name"`
	Symbol      string    `json:"symbol"`
	Decimals    uint8     `json:"decimals"`
	LogoURL     string    `json:"logoUrl"`
	TotalSupply string    `json:"totalSupply"`
	IsVerified  bool      `json:"isVerified"`
	LastUpdated Timestamp `json:"lastUpdated"`
}

// Price data
type PriceData struct {
	TokenAddress Address   `json:"tokenAddress"`
	PriceUSD     float64   `json:"priceUsd"`
	PriceETH     float64   `json:"priceEth"`
	Change24h    float64   `json:"change24h"`
	Volume24h    float64   `json:"volume24h"`
	MarketCap    float64   `json:"marketCap"`
	Timestamp    Timestamp `json:"timestamp"`
	Confidence   uint8     `json:"confidence"`
}

// Gas data
type GasData struct {
	ChainID              ChainID   `json:"chainId"`
	GasPriceGwei         GasPrice  `json:"gasPriceGwei"`
	GasLimit             uint64    `json:"gasLimit"`
	EstimatedGas         uint64    `json:"estimatedGas"`
	MaxFeePerGas         uint64    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas uint64    `json:"maxPriorityFeePerGas"`
	NetworkCongestion    string    `json:"networkCongestion"`
	Timestamp            Timestamp `json:"timestamp"`
}

// Network data
type NetworkData struct {
	ChainID       ChainID   `json:"chainId"`
	Name          string    `json:"name"`
	Symbol        string    `json:"symbol"`
	RPCURL        string    `json:"rpcUrl"`
	BlockNumber   uint64    `json:"blockNumber"`
	BlockTimeMs   uint64    `json:"blockTimeMs"`
	GasLimit      uint64    `json:"gasLimit"`
	NetworkStatus string    `json:"networkStatus"`
	LastSynced    Timestamp `json:"lastSynced"`
}

// Swap quote
type SwapQuote struct {
	FromToken    Address     `json:"fromToken"`
	ToToken      Address     `json:"toToken"`
	FromAmount   TokenAmount `json:"fromAmount"`
	ToAmount     TokenAmount `json:"toAmount"`
	PriceImpact  float64     `json:"priceImpact"`
	GasLimit     uint64      `json:"gasLimit"`
	EstimatedGas uint64      `json:"estimatedGas"`
	Route        []SwapRoute `json:"route"`
	ExpiresAt    Timestamp   `json:"expiresAt"`
}

// Swap route
type SwapRoute struct {
	Protocol      string      `json:"protocol"`
	FromToken     Address     `json:"fromToken"`
	ToToken       Address     `json:"toToken"`
	FromAmount    TokenAmount `json:"fromAmount"`
	ToAmount      TokenAmount `json:"toAmount"`
	FeePercentage float64     `json:"feePercentage"`
}

// MEV opportunity
type MEVOpportunity struct {
	Type               string    `json:"type"`
	FrontRunTx         string    `json:"frontRunTx"`
	BackRunTx          string    `json:"backRunTx"`
	EstimatedProfitETH float64   `json:"estimatedProfitEth"`
	EstimatedProfitUSD float64   `json:"estimatedProfitUsd"`
	AffectedAddresses  []Address `json:"affectedAddresses"`
	BlockNumber        uint64    `json:"blockNumber"`
	DetectedAt         Timestamp `json:"detectedAt"`
}

// Liquidity data
type LiquidityData struct {
	PairAddress  Address   `json:"pairAddress"`
	TokenA       Address   `json:"tokenA"`
	TokenB       Address   `json:"tokenB"`
	ReserveA     float64   `json:"reserveA"`
	ReserveB     float64   `json:"reserveB"`
	LiquidityUSD float64   `json:"liquidityUsd"`
	Volume24h    float64   `json:"volume24h"`
	Fees24h      float64   `json:"fees24h"`
	LastUpdated  Timestamp `json:"lastUpdated"`
}

// Arbitrage opportunity
type ArbitrageOpportunity struct {
	DEXA                string  `json:"dexA"`
	DEXB                string  `json:"dexB"`
	TokenA              Address `json:"tokenA"`
	TokenB              Address `json:"tokenB"`
	PriceDiffPercentage float64 `json:"priceDiffPercentage"`
	MaxTradeAmount      float64 `json:"maxTradeAmount"`
	EstimatedProfit     float64 `json:"estimatedProfit"`
	ProfitableBlock     uint64  `json:"profitableBlock"`
}

// Token risk data
type TokenRiskData struct {
	TokenAddress     Address   `json:"tokenAddress"`
	RiskScore        uint8     `json:"riskScore"`
	RiskLevel        string    `json:"riskLevel"`
	IsVerified       bool      `json:"isVerified"`
	IsHoneypot       bool      `json:"isHoneypot"`
	IsPausable       bool      `json:"isPausable"`
	IsMintable       bool      `json:"isMintable"`
	HasBlacklist     bool      `json:"hasBlacklist"`
	HolderCount      float64   `json:"holderCount"`
	TransferCount24h float64   `json:"transferCount24h"`
	Flags            []string  `json:"flags"`
	AnalyzedAt       Timestamp `json:"analyzedAt"`
}

// Smart contract info
type ContractInfo struct {
	ContractAddress Address           `json:"contractAddress"`
	ContractType    string            `json:"contractType"`
	SourceCode      string            `json:"sourceCode"`
	IsVerified      bool              `json:"isVerified"`
	CompilerVersion string            `json:"compilerVersion"`
	Functions       []string          `json:"functions"`
	ABI             map[string]string `json:"abi"`
	LastVerified    Timestamp         `json:"lastVerified"`
}

// DeFi yield data
type YieldData struct {
	Protocol    string    `json:"protocol"`
	PoolAddress Address   `json:"poolAddress"`
	RewardToken Address   `json:"rewardToken"`
	APY         float64   `json:"apy"`
	TVL         float64   `json:"tvl"`
	RewardRate  float64   `json:"rewardRate"`
	LockPeriod  uint64    `json:"lockPeriod"`
	RiskLevel   string    `json:"riskLevel"`
	LastUpdated Timestamp `json:"lastUpdated"`
}

// Staking data
type StakingData struct {
	Validator        Address `json:"validator"`
	Network          string  `json:"network"`
	TotalStaked      float64 `json:"totalStaked"`
	RewardsEarned    float64 `json:"rewardsEarned"`
	Commission       float64 `json:"commission"`
	UptimePercentage float64 `json:"uptimePercentage"`
	LastRewardBlock  uint64  `json:"lastRewardBlock"`
}

// NFT floor price
type NFTFloorPrice struct {
	CollectionAddress Address   `json:"collectionAddress"`
	CollectionName    string    `json:"collectionName"`
	FloorPriceETH     float64   `json:"floorPriceEth"`
	FloorPriceUSD     float64   `json:"floorPriceUsd"`
	Volume24h         float64   `json:"volume24h"`
	Sales24h          uint64    `json:"sales24h"`
	AveragePrice      float64   `json:"averagePrice"`
	LastSale          Timestamp `json:"lastSale"`
}

// Whale transaction
type WhaleTransaction struct {
	TxHash      string      `json:"txHash"`
	From        Address     `json:"from"`
	To          Address     `json:"to"`
	Amount      TokenAmount `json:"amount"`
	AmountUSD   float64     `json:"amountUsd"`
	TokenSymbol string      `json:"tokenSymbol"`
	Timestamp   Timestamp   `json:"timestamp"`
	BlockNumber uint64      `json:"blockNumber"`
}

// On-chain analytics
type OnChainAnalytics struct {
	ChainID              ChainID   `json:"chainId"`
	TotalValueLocked     float64   `json:"totalValueLocked"`
	TotalVolume24h       float64   `json:"totalVolume24h"`
	TotalTransactions24h float64   `json:"totalTransactions24h"`
	AverageGasPrice      float64   `json:"averageGasPrice"`
	ActiveAddresses      uint64    `json:"activeAddresses"`
	DeFiTVL              float64   `json:"defiTvl"`
	NFTVolume            float64   `json:"nftVolume"`
	Timestamp            Timestamp `json:"timestamp"`
}

// Transaction simulation result
type SimulationResult struct {
	TxHash         string     `json:"txHash"`
	Success        bool       `json:"success"`
	RevertReason   string     `json:"revertReason"`
	GasUsed        uint64     `json:"gasUsed"`
	StateChanges   string     `json:"stateChanges"`
	EstimatedValue float64    `json:"estimatedValue"`
	Logs           []LogEvent `json:"logs"`
	SimulatedAt    Timestamp  `json:"simulatedAt"`
}

// Log event
type LogEvent struct {
	Address  Address  `json:"address"`
	Topics   []string `json:"topics"`
	Data     string   `json:"data"`
	LogIndex uint64   `json:"logIndex"`
}

// Cross-chain route
type CrossChainRoute struct {
	FromChain            string       `json:"fromChain"`
	ToChain              string       `json:"toChain"`
	FromToken            Address      `json:"fromToken"`
	ToToken              Address      `json:"toToken"`
	FromAmount           TokenAmount  `json:"fromAmount"`
	ToAmount             TokenAmount  `json:"toAmount"`
	PriceImpact          float64      `json:"priceImpact"`
	EstimatedTimeMinutes uint64       `json:"estimatedTimeMinutes"`
	TotalFeeUSD          float64      `json:"totalFeeUsd"`
	Steps                []BridgeStep `json:"steps"`
}

// Bridge step
type BridgeStep struct {
	Protocol  string  `json:"protocol"`
	FromChain string  `json:"fromChain"`
	ToChain   string  `json:"toChain"`
	FromToken Address `json:"fromToken"`
	ToToken   Address `json:"toToken"`
}

// Fetcher statistics
type FetcherStats struct {
	Name               string  `json:"name"`
	LastLatencyNs      uint64  `json:"lastLatencyNs"`
	TotalRequests      uint64  `json:"totalRequests"`
	SuccessfulRequests uint64  `json:"successfulRequests"`
	SuccessRate        float64 `json:"successRate"`
}

// =============================================================================
// BASE FETCHER
// =============================================================================

type BaseFetcher struct {
	Name               string
	Running            atomic.Bool
	LastLatencyNs      atomic.Uint64
	TotalRequests      atomic.Uint64
	SuccessfulRequests atomic.Uint64
}

func NewBaseFetcher(name string) *BaseFetcher {
	return &BaseFetcher{
		Name: name,
	}
}

func (f *BaseFetcher) SetRunning(running bool) {
	f.Running.Store(running)
}

func (f *BaseFetcher) IsRunning() bool {
	return f.Running.Load()
}

func (f *BaseFetcher) UpdateLatency(latencyNs uint64) {
	f.LastLatencyNs.Store(latencyNs)
}

func (f *BaseFetcher) RecordRequest(success bool) {
	f.TotalRequests.Add(1)
	if success {
		f.SuccessfulRequests.Add(1)
	}
}

func (f *BaseFetcher) GetStats() FetcherStats {
	total := f.TotalRequests.Load()
	success := f.SuccessfulRequests.Load()

	var rate float64
	if total > 0 {
		rate = float64(success) / float64(total) * 100.0
	}

	return FetcherStats{
		Name:               f.Name,
		LastLatencyNs:      f.LastLatencyNs.Load(),
		TotalRequests:      total,
		SuccessfulRequests: success,
		SuccessRate:        rate,
	}
}

// =============================================================================
// STANDARD FETCHERS
// =============================================================================

// ERC-20 Token Fetcher
type ERC20TokenFetcher struct {
	*BaseFetcher
	Tokens sync.Map // map[Address]TokenMetadata
}

func NewERC20TokenFetcher() *ERC20TokenFetcher {
	return &ERC20TokenFetcher{
		BaseFetcher: NewBaseFetcher("ERC20TokenFetcher"),
	}
}

func (f *ERC20TokenFetcher) Initialize() error {
	fmt.Println("Initializing ERC20 Token Fetcher...")

	// Add default tokens
	defaultTokens := []TokenMetadata{
		{
			Address:     "0x0000000000000000000000000000000000000000",
			Name:        "Ethereum",
			Symbol:      "ETH",
			Decimals:    18,
			IsVerified:  true,
			LastUpdated: currentTimestamp(),
		},
		{
			Address:     "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			Name:        "Tether USD",
			Symbol:      "USDT",
			Decimals:    6,
			IsVerified:  true,
			LastUpdated: currentTimestamp(),
		},
		{
			Address:     "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
			Name:        "USD Coin",
			Symbol:      "USDC",
			Decimals:    6,
			IsVerified:  true,
			LastUpdated: currentTimestamp(),
		},
		{
			Address:     "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599",
			Name:        "Wrapped Bitcoin",
			Symbol:      "WBTC",
			Decimals:    8,
			IsVerified:  true,
			LastUpdated: currentTimestamp(),
		},
	}

	for _, token := range defaultTokens {
		f.Tokens.Store(token.Address, token)
	}

	f.SetRunning(true)
	return nil
}

func (f *ERC20TokenFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	endpoint := chainRPCEndpoint(1)
	ok := true
	var firstErr error
	f.Tokens.Range(func(key, value interface{}) bool {
		token := value.(TokenMetadata)
		if token.Address == "0x0000000000000000000000000000000000000000" {
			return true // native asset has no contract metadata
		}
		name, symbol, decimals, totalSupply, err := fetchERC20Metadata(ctx, endpoint, string(token.Address))
		if err != nil {
			ok = false
			firstErr = err
			return true // a single failing token must not freeze the rest
		}
		if name != "" {
			token.Name = name
		}
		if symbol != "" {
			token.Symbol = symbol
		}
		token.Decimals = decimals
		if totalSupply != nil {
			token.TotalSupply = totalSupply.String()
		}
		token.LastUpdated = currentTimestamp()
		f.Tokens.Store(key, token)
		return true
	})

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(ok)
	return firstErr
}

func (f *ERC20TokenFetcher) GetToken(address Address) (TokenMetadata, bool) {
	val, ok := f.Tokens.Load(address)
	if ok {
		return val.(TokenMetadata), true
	}
	return TokenMetadata{}, false
}

func (f *ERC20TokenFetcher) GetAllTokens() []TokenMetadata {
	var tokens []TokenMetadata
	f.Tokens.Range(func(key, value interface{}) bool {
		tokens = append(tokens, value.(TokenMetadata))
		return true
	})
	return tokens
}

func (f *ERC20TokenFetcher) Shutdown() error {
	f.SetRunning(false)
	f.Tokens = sync.Map{}
	return nil
}

// Gas Estimator Fetcher
type GasEstimatorFetcher struct {
	*BaseFetcher
	GasData sync.Map // map[ChainID]GasData
}

func NewGasEstimatorFetcher() *GasEstimatorFetcher {
	return &GasEstimatorFetcher{
		BaseFetcher: NewBaseFetcher("GasEstimatorFetcher"),
	}
}

func (f *GasEstimatorFetcher) Initialize() error {
	fmt.Println("Initializing Gas Estimator Fetcher...")

	// Chains to track. Gas values are zero here on purpose: they are
	// populated from eth_gasPrice / eth_maxPriorityFeePerGas by Fetch.
	// Hardcoded prices would be fabricated data.
	for _, id := range []ChainID{1, 56, 137, 42161, 10, 8453} {
		f.GasData.Store(id, GasData{ChainID: id, GasLimit: 30000000, EstimatedGas: 21000})
	}

	f.SetRunning(true)
	return nil
}

func (f *GasEstimatorFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	ok := true
	var firstErr error
	f.GasData.Range(func(key, value interface{}) bool {
		chainID := key.(ChainID)
		data := value.(GasData)
		endpoint := chainRPCEndpoint(chainID)
		if endpoint == "" {
			return true
		}
		gasPrice, err := ethGasPrice(ctx, endpoint)
		if err != nil {
			ok = false
			firstErr = err
			return true
		}
		gwei := weiToGwei(gasPrice)
		data.GasPriceGwei = GasPrice(uint64(gwei))
		data.MaxFeePerGas = uint64(gwei)
		if tip, err := ethMaxPriorityFee(ctx, endpoint); err == nil {
			tipGwei := uint64(weiToGwei(tip))
			data.MaxPriorityFeePerGas = tipGwei
			data.MaxFeePerGas += tipGwei
		}
		// Congestion from the real base-fee trend over recent blocks.
		if fees, err := ethFeeHistory(ctx, endpoint, 5); err == nil && len(fees) >= 2 {
			first := weiToGwei(fees[0])
			last := weiToGwei(fees[len(fees)-1])
			switch {
			case first > 0 && last > first*1.5:
				data.NetworkCongestion = "high"
			case first > 0 && last < first*0.7:
				data.NetworkCongestion = "low"
			default:
				data.NetworkCongestion = "normal"
			}
		}
		data.Timestamp = currentTimestamp()
		f.GasData.Store(chainID, data)
		return true
	})

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(ok)
	return firstErr
}

func (f *GasEstimatorFetcher) GetGas(chainID ChainID) (GasData, bool) {
	val, ok := f.GasData.Load(chainID)
	if ok {
		return val.(GasData), true
	}
	return GasData{}, false
}

func (f *GasEstimatorFetcher) EstimateGas(from, to Address, data string, chainID ChainID) uint64 {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()
	if endpoint := chainRPCEndpoint(chainID); endpoint != "" {
		msg := callMsg{From: string(from), To: string(to)}
		if data != "" {
			msg.Data = data
		}
		if est, err := ethEstimateGas(ctx, endpoint, msg); err == nil && est > 0 {
			// Node-authoritative estimate with a 20% safety buffer.
			return uint64(float64(est) * 1.2)
		}
	}
	// Offline fallback: intrinsic gas per the Yellow Paper / EIP-2028
	// (21000 base + 4 per zero byte + 16 per non-zero byte).
	gas := uint64(21000)
	d := strings.TrimPrefix(strings.TrimPrefix(data, "0x"), "0X")
	for i := 0; i+1 < len(d); i += 2 {
		if d[i] == '0' && d[i+1] == '0' {
			gas += 4
		} else {
			gas += 16
		}
	}
	return uint64(float64(gas) * 1.2)
}

func (f *GasEstimatorFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// Price Feed Fetcher
type PriceFeedFetcher struct {
	*BaseFetcher
	Prices sync.Map // map[string]PriceData
}

func NewPriceFeedFetcher() *PriceFeedFetcher {
	return &PriceFeedFetcher{
		BaseFetcher: NewBaseFetcher("PriceFeedFetcher"),
	}
}

func (f *PriceFeedFetcher) Initialize() error {
	fmt.Println("Initializing Price Feed Fetcher...")

	// No seed prices: hardcoded values would be fabricated data. Real prices
	// are populated from CoinGecko in Fetch.
	f.SetRunning(true)
	return nil
}

func (f *PriceFeedFetcher) fetchAssets() map[string]struct {
	pair string
	addr Address
} {
	return map[string]struct {
		pair string
		addr Address
	}{
		"ethereum":        {"ETH/USD", "0x0000000000000000000000000000000000000000"},
		"bitcoin":         {"BTC/USD", "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599"},
		"tether":          {"USDT/USD", "0xdAC17F958D2ee523a2206206994597C13D831ec7"},
		"usd-coin":        {"USDC/USD", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"},
		"wrapped-bitcoin": {"WBTC/USD", "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599"},
	}
}

func (f *PriceFeedFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	assets := f.fetchAssets()
	ids := make([]string, 0, len(assets))
	for id := range assets {
		ids = append(ids, id)
	}
	url := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=usd,eth&include_24hr_change=true&include_24hr_vol=true&include_market_cap=true",
		coinGeckoBase(), strings.Join(ids, ","))

	var resp map[string]map[string]float64
	if err := httpGetJSON(ctx, url, coinGeckoHeaders(), &resp); err != nil {
		f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
		f.RecordRequest(false)
		return err
	}
	for id, meta := range assets {
		row, ok := resp[id]
		if !ok {
			continue
		}
		f.Prices.Store(meta.pair, PriceData{
			TokenAddress: meta.addr,
			PriceUSD:     row["usd"],
			PriceETH:     row["eth"],
			Change24h:    row["usd_24h_change"],
			Volume24h:    row["usd_24h_vol"],
			MarketCap:    row["usd_market_cap"],
			Timestamp:    currentTimestamp(),
			Confidence:   95,
		})
	}
	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(true)
	return nil
}

func (f *PriceFeedFetcher) GetPrice(pair string) (PriceData, bool) {
	val, ok := f.Prices.Load(pair)
	if ok {
		return val.(PriceData), true
	}
	return PriceData{}, false
}

func (f *PriceFeedFetcher) GetPriceUSD(token Address) float64 {
	for _, meta := range f.fetchAssets() {
		if meta.addr == token {
			if val, ok := f.Prices.Load(meta.pair); ok {
				return val.(PriceData).PriceUSD
			}
			return 0.0
		}
	}
	return 0.0
}

func (f *PriceFeedFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// DApp Connection Fetcher (WalletConnect)
type DAppConnectionFetcher struct {
	*BaseFetcher
	Sessions sync.Map // map[string]WCSession
}

type WCSession struct {
	Topic         string    `json:"topic"`
	WalletAddress Address   `json:"walletAddress"`
	PeerMetadata  string    `json:"peerMetadata"`
	ChainID       string    `json:"chainId"`
	CreatedAt     Timestamp `json:"createdAt"`
	UpdatedAt     Timestamp `json:"updatedAt"`
	ExpiresAt     Timestamp `json:"expiresAt"`
}

func NewDAppConnectionFetcher() *DAppConnectionFetcher {
	return &DAppConnectionFetcher{
		BaseFetcher: NewBaseFetcher("DAppConnectionFetcher"),
	}
}

func (f *DAppConnectionFetcher) Initialize() error {
	fmt.Println("Initializing DApp Connection Fetcher...")
	f.SetRunning(true)
	return nil
}

func (f *DAppConnectionFetcher) Fetch() error {
	start := time.Now()

	// Clean up expired sessions
	now := currentTimestamp()
	f.Sessions.Range(func(key, value interface{}) bool {
		session := value.(WCSession)
		if session.ExpiresAt < now {
			f.Sessions.Delete(key)
		}
		return true
	})

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(true)

	return nil
}

func (f *DAppConnectionFetcher) CreateSession(walletAddress Address, peerMetadata string) string {
	topic := "0x" + generateHex(32)

	session := WCSession{
		Topic:         topic,
		WalletAddress: walletAddress,
		PeerMetadata:  peerMetadata,
		ChainID:       "1",
		CreatedAt:     currentTimestamp(),
		UpdatedAt:     currentTimestamp(),
		ExpiresAt:     currentTimestamp() + 600000, // 10 minutes
	}

	f.Sessions.Store(topic, session)

	return topic
}

func (f *DAppConnectionFetcher) Disconnect(topic string) bool {
	if _, ok := f.Sessions.Load(topic); ok {
		f.Sessions.Delete(topic)
		return true
	}
	return false
}

func (f *DAppConnectionFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// Network Fetcher
type NetworkFetcher struct {
	*BaseFetcher
	Networks sync.Map // map[ChainID]NetworkData
}

func NewNetworkFetcher() *NetworkFetcher {
	return &NetworkFetcher{
		BaseFetcher: NewBaseFetcher("NetworkFetcher"),
	}
}

func (f *NetworkFetcher) Initialize() error {
	fmt.Println("Initializing Network Fetcher...")

	networks := map[ChainID]NetworkData{
		1: {
			ChainID:       1,
			Name:          "Ethereum",
			Symbol:        "ETH",
			RPCURL:        "https://eth.llamarpc.com",
			BlockTimeMs:   0,
			GasLimit:      30000000,
			NetworkStatus: "synced",
			LastSynced:    currentTimestamp(),
		},
		56: {
			ChainID:       56,
			Name:          "BNB Smart Chain",
			Symbol:        "BNB",
			RPCURL:        "https://bsc-dataseed.binance.org",
			BlockTimeMs:   0,
			GasLimit:      30000000,
			NetworkStatus: "synced",
			LastSynced:    currentTimestamp(),
		},
		137: {
			ChainID:       137,
			Name:          "Polygon",
			Symbol:        "MATIC",
			RPCURL:        "https://polygon-rpc.com",
			BlockTimeMs:   0,
			GasLimit:      30000000,
			NetworkStatus: "synced",
			LastSynced:    currentTimestamp(),
		},
	}

	for chainID, data := range networks {
		f.Networks.Store(chainID, data)
	}

	f.SetRunning(true)
	return nil
}

func (f *NetworkFetcher) Fetch() error {
	start := time.Now()

	ok := true
	var firstErr error
	f.Networks.Range(func(key, value interface{}) bool {
		chainID := key.(ChainID)
		network := value.(NetworkData)
		endpoint := network.RPCURL
		if override := chainRPCEndpoint(chainID); override != "" {
			endpoint = override
		}
		if endpoint == "" {
			return true
		}
		ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
		probeStart := time.Now()
		block, err := ethBlockNumber(ctx, endpoint)
		cancel()
		if err != nil {
			network.NetworkStatus = "unreachable"
			ok = false
			firstErr = err
			f.Networks.Store(chainID, network)
			return true
		}
		network.BlockNumber = block
		network.BlockTimeMs = uint64(time.Since(probeStart).Milliseconds())
		network.NetworkStatus = "synced"
		network.LastSynced = currentTimestamp()
		f.Networks.Store(chainID, network)
		return true
	})

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(ok)
	return firstErr
}

func (f *NetworkFetcher) GetNetwork(chainID ChainID) (NetworkData, bool) {
	val, ok := f.Networks.Load(chainID)
	if ok {
		return val.(NetworkData), true
	}
	return NetworkData{}, false
}

func (f *NetworkFetcher) SwitchNetwork(chainID ChainID) bool {
	_, ok := f.Networks.Load(chainID)
	return ok
}

func (f *NetworkFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// Swap Quote Fetcher
type SwapQuoteFetcher struct {
	*BaseFetcher
}

func NewSwapQuoteFetcher() *SwapQuoteFetcher {
	return &SwapQuoteFetcher{
		BaseFetcher: NewBaseFetcher("SwapQuoteFetcher"),
	}
}

func (f *SwapQuoteFetcher) Initialize() error {
	fmt.Println("Initializing Swap Quote Fetcher...")
	f.SetRunning(true)
	return nil
}

func (f *SwapQuoteFetcher) Fetch() error {
	start := time.Now()

	// Quotes are fetched on-demand

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(true)

	return nil
}

// GetQuoteWithError returns a real on-chain Uniswap V3 Quoter quote.
// fromAmount must be an integer amount in the token's base units.
func (f *SwapQuoteFetcher) GetQuoteWithError(fromToken, toToken, fromAmount TokenAmount, chainID ChainID) (SwapQuote, error) {
	endpoint := chainRPCEndpoint(chainID)
	if endpoint == "" {
		return SwapQuote{}, fmt.Errorf("no RPC endpoint for chain %d", uint64(chainID))
	}
	amountIn, err := parseTokenAmount(fromAmount)
	if err != nil {
		return SwapQuote{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	// Try fee tiers from cheapest to deepest liquidity.
	var out *big.Int
	var lastErr error
	usedFee := uint64(0)
	for _, fee := range []uint64{500, 3000, 10000} {
		out, lastErr = quoteExactInputSingle(ctx, endpoint, string(fromToken), string(toToken), fee, amountIn)
		if lastErr == nil {
			usedFee = fee
			break
		}
	}
	if out == nil {
		return SwapQuote{}, fmt.Errorf("no UniV3 pool quotes the pair: %v", lastErr)
	}
	const estimatedGas = uint64(180000)
	return SwapQuote{
		FromToken:    Address(fromToken),
		ToToken:      Address(toToken),
		FromAmount:   fromAmount,
		ToAmount:     TokenAmount(out.String()),
		GasLimit:     estimatedGas,
		EstimatedGas: estimatedGas,
		Route: []SwapRoute{{
			Protocol:      "uniswap_v3",
			FromToken:     Address(fromToken),
			ToToken:       Address(toToken),
			FromAmount:    fromAmount,
			ToAmount:      TokenAmount(out.String()),
			FeePercentage: float64(usedFee) / 10000.0,
		}},
		ExpiresAt: currentTimestamp() + 30000,
	}, nil
}

// GetQuote keeps the legacy signature; failures return an empty quote.
// Callers that need diagnostics use GetQuoteWithError.
func (f *SwapQuoteFetcher) GetQuote(fromToken, toToken, fromAmount TokenAmount, chainID ChainID) SwapQuote {
	quote, err := f.GetQuoteWithError(fromToken, toToken, fromAmount, chainID)
	if err != nil {
		return SwapQuote{
			FromToken:  Address(fromToken),
			ToToken:    Address(toToken),
			FromAmount: fromAmount,
			ExpiresAt:  currentTimestamp(),
		}
	}
	return quote
}

func (f *SwapQuoteFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// =============================================================================
// ADVANCED FETCHERS
// =============================================================================

// AI Price Predictor Fetcher
type AIPricePredictorFetcher struct {
	*BaseFetcher
	Predictions sync.Map // map[Address]PricePrediction
}

type PricePrediction struct {
	Token        Address            `json:"token"`
	CurrentPrice float64            `json:"currentPrice"`
	Predictions  map[uint64]float64 `json:"predictions"`
	Confidence   float64            `json:"confidence"`
	PredictedAt  Timestamp          `json:"predictedAt"`
}

func NewAIPricePredictorFetcher() *AIPricePredictorFetcher {
	return &AIPricePredictorFetcher{
		BaseFetcher: NewBaseFetcher("AIPricePredictorFetcher"),
	}
}

func (f *AIPricePredictorFetcher) Initialize() error {
	fmt.Println("Initializing AI Price Predictor Fetcher...")
	f.SetRunning(true)
	return nil
}

func (f *AIPricePredictorFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	assets := map[string]Address{
		"ethereum": "0x0000000000000000000000000000000000000000",
		"bitcoin":  "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599",
	}
	ok := true
	var firstErr error
	for id, token := range assets {
		url := fmt.Sprintf("%s/coins/%s/market_chart?vs_currency=usd&days=30&interval=daily", coinGeckoBase(), id)
		var chart struct {
			Prices [][2]float64 `json:"prices"`
		}
		if err := httpGetJSON(ctx, url, coinGeckoHeaders(), &chart); err != nil {
			ok = false
			firstErr = err
			continue
		}
		if len(chart.Prices) < 10 {
			continue
		}
		slope, _, r2 := linearRegression(chart.Prices)
		current := chart.Prices[len(chart.Prices)-1][1]
		predictions := map[uint64]float64{}
		for _, horizon := range []uint64{3600, 21600, 43200, 86400, 604800} {
			p := current + slope*float64(horizon)*1000
			if p < 0 {
				p = 0
			}
			predictions[horizon] = p
		}
		confidence := r2 * 100
		if confidence > 95 {
			confidence = 95
		}
		f.Predictions.Store(token, PricePrediction{
			Token:        token,
			CurrentPrice: current,
			Predictions:  predictions,
			Confidence:   confidence,
			PredictedAt:  currentTimestamp(),
		})
	}

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(ok)
	return firstErr
}

func (f *AIPricePredictorFetcher) GetPrediction(token Address, horizonSecs uint64) (PricePrediction, bool) {
	val, ok := f.Predictions.Load(token)
	if ok {
		return val.(PricePrediction), true
	}
	return PricePrediction{}, false
}

func (f *AIPricePredictorFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// MEV Opportunity Fetcher
type MEVOpportunityFetcher struct {
	*BaseFetcher
	Opportunities []MEVOpportunity
	mu            sync.Mutex
}

func NewMEVOpportunityFetcher() *MEVOpportunityFetcher {
	return &MEVOpportunityFetcher{
		BaseFetcher: NewBaseFetcher("MEVOpportunityFetcher"),
	}
}

func (f *MEVOpportunityFetcher) Initialize() error {
	fmt.Println("Initializing MEV Opportunity Fetcher...")
	f.SetRunning(true)
	return nil
}

func (f *MEVOpportunityFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	endpoint := chainRPCEndpoint(1)
	block, err := ethGetBlock(ctx, endpoint, "latest", true)
	if err != nil {
		f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
		f.RecordRequest(false)
		return err
	}
	blockNum, _ := strconv.ParseUint(strings.TrimPrefix(block.Number, "0x"), 16, 64)

	// Known DEX routers whose flow is screened for same-block repeated-swap
	// patterns (a conservative, honest sandwich heuristic — profit is not
	// estimable without trace access, so it is reported as unknown/zero).
	routers := map[string]bool{
		"0x7a250d5630b4cf539739df2c5dacb4c659f2488d": true, // Uniswap V2 Router
		"0xe592427a0aece92de3edee1f18e0157c05861564": true, // Uniswap V3 Router
		"0x68b3465833fb72a70ecdf485e0e4c7bd8665fc45": true, // Uniswap V3 Router02
	}
	byRouterSender := map[string]map[string]int{}
	for _, tx := range block.FullTransactions() {
		to := strings.ToLower(tx.To)
		if routers[to] {
			if byRouterSender[to] == nil {
				byRouterSender[to] = map[string]int{}
			}
			byRouterSender[to][strings.ToLower(tx.From)]++
		}
	}
	var found []MEVOpportunity
	for router, senders := range byRouterSender {
		var involved []Address
		dupes := 0
		for sender, count := range senders {
			if count > 1 {
				dupes += count - 1
			}
			involved = append(involved, Address(sender))
		}
		if dupes > 0 {
			found = append(found, MEVOpportunity{
				Type:               "repeated_swap_pattern",
				FrontRunTx:         router,
				EstimatedProfitETH: 0,
				EstimatedProfitUSD: 0,
				AffectedAddresses:  involved,
				BlockNumber:        blockNum,
				DetectedAt:         currentTimestamp(),
			})
		}
	}
	f.mu.Lock()
	f.Opportunities = append(found, f.Opportunities...)
	if len(f.Opportunities) > 100 {
		f.Opportunities = f.Opportunities[:100]
	}
	f.mu.Unlock()

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(true)
	return nil
}

func (f *MEVOpportunityFetcher) GetOpportunities() []MEVOpportunity {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Opportunities
}

func (f *MEVOpportunityFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// Liquidity Fetcher
type LiquidityFetcher struct {
	*BaseFetcher
	Liquidity sync.Map // map[string]LiquidityData
}

func NewLiquidityFetcher() *LiquidityFetcher {
	return &LiquidityFetcher{
		BaseFetcher: NewBaseFetcher("LiquidityFetcher"),
	}
}

func (f *LiquidityFetcher) Initialize() error {
	fmt.Println("Initializing Liquidity Fetcher...")
	f.SetRunning(true)
	return nil
}

// defaultPairs are the tracked mainnet UniV2 pairs (tokenA, tokenB).
var defaultPairs = [][2]Address{
	{"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"}, // WETH/USDC
	{"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "0xdAC17F958D2ee523a2206206994597C13D831ec7"}, // WETH/USDT
}

func (f *LiquidityFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	endpoint := chainRPCEndpoint(1)
	factory := os.Getenv("FULL_FETCHERS_V2_FACTORY")
	if factory == "" {
		factory = "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f" // Uniswap V2 factory
	}
	ok := true
	var firstErr error
	for _, p := range defaultPairs {
		pairAddr, err := resolveV2Pair(ctx, endpoint, factory, string(p[0]), string(p[1]))
		if err != nil {
			ok = false
			firstErr = err
			continue
		}
		r0, r1, err := fetchV2Reserves(ctx, endpoint, pairAddr)
		if err != nil {
			ok = false
			firstErr = err
			continue
		}
		// Reserves follow the pair's token0/token1 ordering, resolved on-chain.
		token0, err := fetchPairToken0(ctx, endpoint, pairAddr)
		if err != nil {
			ok = false
			firstErr = err
			continue
		}
		decimalsA := uint8(18)
		if strings.EqualFold(string(p[0]), "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48") ||
			strings.EqualFold(string(p[0]), "0xdAC17F958D2ee523a2206206994597C13D831ec7") {
			decimalsA = 6
		}
		var reserveA, reserveB float64
		if strings.EqualFold(token0, string(p[0])) {
			reserveA = bigToFloat(r0, int(decimalsA))
			reserveB = bigToFloat(r1, 18)
		} else {
			reserveA = bigToFloat(r1, int(decimalsA))
			reserveB = bigToFloat(r0, 18)
		}
		f.Liquidity.Store(string(p[0])+"_"+string(p[1]), LiquidityData{
			PairAddress:  Address(pairAddr),
			TokenA:       p[0],
			TokenB:       p[1],
			ReserveA:     reserveA,
			ReserveB:     reserveB,
			LiquidityUSD: 0, // USD valuation is filled in once a PriceFeedFetcher is attached
			LastUpdated:  currentTimestamp(),
		})
	}

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(ok)
	return firstErr
}

func (f *LiquidityFetcher) GetLiquidity(tokenA, tokenB Address) (LiquidityData, bool) {
	key := string(tokenA) + "_" + string(tokenB)
	val, ok := f.Liquidity.Load(key)
	if ok {
		return val.(LiquidityData), true
	}
	return LiquidityData{}, false
}

func (f *LiquidityFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// Arbitrage Fetcher
type ArbitrageFetcher struct {
	*BaseFetcher
	Opportunities []ArbitrageOpportunity
	mu            sync.Mutex
}

func NewArbitrageFetcher() *ArbitrageFetcher {
	return &ArbitrageFetcher{
		BaseFetcher: NewBaseFetcher("ArbitrageFetcher"),
	}
}

func (f *ArbitrageFetcher) Initialize() error {
	fmt.Println("Initializing Arbitrage Fetcher...")
	f.SetRunning(true)
	return nil
}

func (f *ArbitrageFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	endpoint := chainRPCEndpoint(1)
	factory := os.Getenv("FULL_FETCHERS_V2_FACTORY")
	if factory == "" {
		factory = "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f"
	}
	const (
		weth   = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
		usdc   = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
		v3pool = "0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640" // USDC/WETH 0.05%
	)

	pairAddr, err := resolveV2Pair(ctx, endpoint, factory, weth, usdc)
	if err != nil {
		f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
		f.RecordRequest(false)
		return err
	}
	r0, r1, err := fetchV2Reserves(ctx, endpoint, pairAddr)
	if err != nil {
		f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
		f.RecordRequest(false)
		return err
	}
	// UniV2 USDC/WETH: token0 = USDC (6dp), token1 = WETH (18dp).
	usdcR := bigToFloat(r0, 6)
	wethR := bigToFloat(r1, 18)
	if wethR == 0 {
		return fmt.Errorf("empty WETH reserve")
	}
	v2Price := usdcR / wethR // USD per WETH

	sqrtP, err := fetchV3SqrtPriceX96(ctx, endpoint, v3pool)
	if err != nil {
		f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
		f.RecordRequest(false)
		return err
	}
	// price(token1/token0) = (sqrtP/2^96)^2 ; scale by 10^(18-6).
	ratio := new(big.Float).SetInt(sqrtP)
	ratio.Quo(ratio, new(big.Float).SetInt(new(big.Int).Lsh(big.NewInt(1), 96)))
	ratio.Mul(ratio, ratio)
	v3Price, _ := new(big.Float).Mul(ratio, big.NewFloat(1e12)).Float64()

	min := v2Price
	if v3Price < min {
		min = v3Price
	}
	if min <= 0 {
		return fmt.Errorf("zero on-chain price")
	}
	diffPct := (v2Price - v3Price)
	if diffPct < 0 {
		diffPct = -diffPct
	}
	diffPct = diffPct / min * 100

	// Net profit for a $50k cycle after two 0.3% DEX fees; recorded only when
	// genuinely positive. Block tag ties the record to a real chain state.
	blockNum, _ := ethBlockNumber(ctx, endpoint)
	if diffPct >= 0.6 {
		profit := 50000.0 * (diffPct/100.0 - 0.006)
		if profit > 0 {
			f.mu.Lock()
			f.Opportunities = append([]ArbitrageOpportunity{{
				DEXA:                "uniswap_v2",
				DEXB:                "uniswap_v3",
				TokenA:              Address(weth),
				TokenB:              Address(usdc),
				PriceDiffPercentage: diffPct,
				MaxTradeAmount:      50000.0,
				EstimatedProfit:     profit,
				ProfitableBlock:     blockNum,
			}}, f.Opportunities...)
			if len(f.Opportunities) > 100 {
				f.Opportunities = f.Opportunities[:100]
			}
			f.mu.Unlock()
		}
	}

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(true)
	return nil
}

func (f *ArbitrageFetcher) GetProfitable() []ArbitrageOpportunity {
	f.mu.Lock()
	defer f.mu.Unlock()

	var result []ArbitrageOpportunity
	for _, opp := range f.Opportunities {
		if opp.EstimatedProfit >= 50.0 {
			result = append(result, opp)
		}
	}
	return result
}

func (f *ArbitrageFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// Token Risk Fetcher
type TokenRiskFetcher struct {
	*BaseFetcher
	Risks sync.Map // map[Address]TokenRiskData
}

func NewTokenRiskFetcher() *TokenRiskFetcher {
	return &TokenRiskFetcher{
		BaseFetcher: NewBaseFetcher("TokenRiskFetcher"),
	}
}

func (f *TokenRiskFetcher) Initialize() error {
	fmt.Println("Initializing Token Risk Fetcher...")
	f.SetRunning(true)
	return nil
}

// riskSelectors found in bytecode disclose admin functions on the token.
var riskSelectors = map[string]struct {
	flag     string
	scoreAdd uint8
}{
	"40c10f19": {"mint(address,uint256)", 25}, // mint
	"8456cb59": {"pause()", 20},               // pause
	"f9f92be4": {"addBlackList(address)", 15}, // USDT-style blacklist
	"0d8f23e0": {"setBlackList(address)", 15}, // blacklist setter
}

func (f *TokenRiskFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	endpoint := chainRPCEndpoint(1)
	watchlist := []Address{
		"0xdAC17F958D2ee523a2206206994597C13D831ec7", // USDT
		"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", // USDC
		"0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", // WBTC
	}
	if extra := os.Getenv("FULL_FETCHERS_RISK_TOKENS"); extra != "" {
		for _, a := range strings.Split(extra, ",") {
			watchlist = append(watchlist, Address(strings.TrimSpace(a)))
		}
	}

	ok := true
	var firstErr error
	for _, token := range watchlist {
		code, err := ethGetCode(ctx, endpoint, string(token))
		if err != nil {
			ok = false
			firstErr = err
			continue
		}
		data := TokenRiskData{TokenAddress: token, AnalyzedAt: currentTimestamp()}
		body := strings.ToLower(strings.TrimPrefix(code, "0x"))
		if len(body) < 2 {
			data.RiskScore = 100
			data.RiskLevel = "critical"
			data.Flags = []string{"no_contract_code"}
			f.Risks.Store(token, data)
			continue
		}
		score := uint8(10)
		// EIP-1967 proxy implementation slot.
		impl, err := ethGetStorageAt(ctx, endpoint, string(token),
			"0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
		if err == nil {
			if n, perr := parseHexBig(impl); perr == nil && n.Sign() != 0 {
				data.Flags = append(data.Flags, "upgradeable_proxy")
			}
		}
		for sel, meta := range riskSelectors {
			if strings.Contains(body, sel) {
				data.Flags = append(data.Flags, meta.flag)
				score += meta.scoreAdd
				switch meta.flag {
				case "mint(address,uint256)":
					data.IsMintable = true
				case "pause()":
					data.IsPausable = true
				default:
					data.HasBlacklist = true
				}
			}
		}
		if score > 100 {
			score = 100
		}
		data.RiskScore = score
		switch {
		case score < 30:
			data.RiskLevel = "low"
		case score < 60:
			data.RiskLevel = "medium"
		default:
			data.RiskLevel = "high"
		}
		f.Risks.Store(token, data)
	}

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(ok)
	return firstErr
}

func (f *TokenRiskFetcher) GetRisk(token Address) (TokenRiskData, bool) {
	val, ok := f.Risks.Load(token)
	if ok {
		return val.(TokenRiskData), true
	}
	return TokenRiskData{}, false
}

func (f *TokenRiskFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// Smart Contract Fetcher
type SmartContractFetcher struct {
	*BaseFetcher
	Contracts sync.Map // map[Address]ContractInfo
}

func NewSmartContractFetcher() *SmartContractFetcher {
	return &SmartContractFetcher{
		BaseFetcher: NewBaseFetcher("SmartContractFetcher"),
	}
}

func (f *SmartContractFetcher) Initialize() error {
	fmt.Println("Initializing Smart Contract Fetcher...")
	f.SetRunning(true)
	return nil
}

func (f *SmartContractFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	endpoint := chainRPCEndpoint(1)
	watchlist := []Address{
		"0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D", // Uniswap V2 Router
		"0xE592427A0AEce92De3Edee1F18E0157C05861564", // Uniswap V3 Router
		"0xdAC17F958D2ee523a2206206994597C13D831ec7", // USDT
	}
	if extra := os.Getenv("FULL_FETCHERS_CONTRACTS"); extra != "" {
		for _, a := range strings.Split(extra, ",") {
			watchlist = append(watchlist, Address(strings.TrimSpace(a)))
		}
	}

	ok := true
	var firstErr error
	for _, addr := range watchlist {
		code, err := ethGetCode(ctx, endpoint, string(addr))
		if err != nil {
			ok = false
			firstErr = err
			continue
		}
		info := ContractInfo{ContractAddress: addr, ContractType: "contract", LastVerified: currentTimestamp()}
		if len(code) <= 2 {
			info.ContractType = "eoa"
			f.Contracts.Store(addr, info)
			continue
		}
		impl, err := ethGetStorageAt(ctx, endpoint, string(addr),
			"0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
		if err == nil {
			if n, perr := parseHexBig(impl); perr == nil && n.Sign() != 0 {
				info.ContractType = "proxy"
			}
		}
		// Verification status only from a real explorer API.
		if key := os.Getenv("ETHERSCAN_API_KEY"); key != "" {
			var resp struct {
				Result []struct {
					ABI             string `json:"ABI"`
					ContractName    string `json:"ContractName"`
					CompilerVersion string `json:"CompilerVersion"`
				} `json:"result"`
			}
			url := fmt.Sprintf("https://api.etherscan.io/api?module=contract&action=getsourcecode&address=%s&apikey=%s", addr, key)
			if err := httpGetJSON(ctx, url, nil, &resp); err == nil && len(resp.Result) > 0 && resp.Result[0].ABI != "Contract source code not verified" {
				info.IsVerified = true
				info.CompilerVersion = resp.Result[0].CompilerVersion
				info.ABI = map[string]string{"contract": resp.Result[0].ABI}
			}
		}
		f.Contracts.Store(addr, info)
	}

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(ok)
	return firstErr
}

func (f *SmartContractFetcher) GetContract(address Address) (ContractInfo, bool) {
	val, ok := f.Contracts.Load(address)
	if ok {
		return val.(ContractInfo), true
	}
	return ContractInfo{}, false
}

func (f *SmartContractFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// Gas Market Fetcher
type GasMarketFetcher struct {
	*BaseFetcher
	BaseFees sync.Map // map[ChainID][]float64 (gwei)
}

func NewGasMarketFetcher() *GasMarketFetcher {
	return &GasMarketFetcher{
		BaseFetcher: NewBaseFetcher("GasMarketFetcher"),
	}
}

func (f *GasMarketFetcher) Initialize() error {
	fmt.Println("Initializing Gas Market Fetcher...")
	f.SetRunning(true)
	return nil
}

func (f *GasMarketFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	chainID := ChainID(1)
	endpoint := chainRPCEndpoint(chainID)
	fees, err := ethFeeHistory(ctx, endpoint, 20)
	if err != nil {
		f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
		f.RecordRequest(false)
		return err
	}
	gwei := make([]float64, 0, len(fees))
	for _, fee := range fees {
		gwei = append(gwei, weiToGwei(fee))
	}
	f.BaseFees.Store(chainID, gwei)

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(true)
	return nil
}

func (f *GasMarketFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// DeFi Yield Fetcher
type DeFiYieldFetcher struct {
	*BaseFetcher
	Yields sync.Map // map[string]YieldData
}

func NewDeFiYieldFetcher() *DeFiYieldFetcher {
	return &DeFiYieldFetcher{
		BaseFetcher: NewBaseFetcher("DeFiYieldFetcher"),
	}
}

func (f *DeFiYieldFetcher) Initialize() error {
	fmt.Println("Initializing DeFi Yield Fetcher...")

	// No seed yields — fabricated APY/TVL is exactly the kind of fake data
	// this fetcher must not serve. Fetch populates from DefiLlama.
	f.SetRunning(true)
	return nil
}

type llamaPool struct {
	Pool    string   `json:"pool"`
	Project string   `json:"project"`
	Chain   string   `json:"chain"`
	Apy     float64  `json:"apy"`
	TvlUsd  float64  `json:"tvlUsd"`
	Tokens  []string `json:"underlyingTokens"`
}

func (f *DeFiYieldFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	var resp struct {
		Data []llamaPool `json:"data"`
	}
	if err := httpGetJSON(ctx, "https://yields.llama.fi/pools", nil, &resp); err != nil {
		f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
		f.RecordRequest(false)
		return err
	}
	// Keep the top pool per protocol on Ethereum by TVL.
	best := map[string]llamaPool{}
	for _, pool := range resp.Data {
		if pool.Chain != "Ethereum" {
			continue
		}
		if cur, ok := best[pool.Project]; !ok || pool.TvlUsd > cur.TvlUsd {
			best[pool.Project] = pool
		}
	}
	for project, pool := range best {
		risk := "high"
		switch {
		case pool.TvlUsd > 1e9:
			risk = "low"
		case pool.TvlUsd > 1e8:
			risk = "medium"
		}
		var token Address
		if len(pool.Tokens) > 0 {
			token = Address(pool.Tokens[0])
		}
		f.Yields.Store(project, YieldData{
			Protocol:    project,
			PoolAddress: Address(pool.Pool),
			RewardToken: token,
			APY:         pool.Apy,
			TVL:         pool.TvlUsd,
			RiskLevel:   risk,
			LastUpdated: currentTimestamp(),
		})
	}

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(true)
	return nil
}

func (f *DeFiYieldFetcher) GetBestYields(minTVL float64) []YieldData {
	var result []YieldData

	f.Yields.Range(func(key, value interface{}) bool {
		yield := value.(YieldData)
		if yield.TVL >= minTVL {
			result = append(result, yield)
		}
		return true
	})

	return result
}

func (f *DeFiYieldFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// Staking Optimizer Fetcher
type StakingOptimizerFetcher struct {
	*BaseFetcher
	Staking sync.Map // map[string]StakingData
}

func NewStakingOptimizerFetcher() *StakingOptimizerFetcher {
	return &StakingOptimizerFetcher{
		BaseFetcher: NewBaseFetcher("StakingOptimizerFetcher"),
	}
}

func (f *StakingOptimizerFetcher) Initialize() error {
	fmt.Println("Initializing Staking Optimizer Fetcher...")
	f.SetRunning(true)
	return nil
}

func (f *StakingOptimizerFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	var lido struct {
		Data struct {
			Apr float64 `json:"apr"`
		} `json:"data"`
	}
	if err := httpGetJSON(ctx, "https://eth-api.lido.fi/v1/protocol/steth/apr/last", nil, &lido); err != nil {
		f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
		f.RecordRequest(false)
		return err
	}
	f.Staking.Store("ethereum-lido", StakingData{
		Validator:     "lido",
		Network:       "ethereum",
		Commission:    10, // Lido charges a 10% fee on staking rewards
		RewardsEarned: lido.Data.Apr,
	})

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(true)
	return nil
}

func (f *StakingOptimizerFetcher) GetBestValidator(network string) (StakingData, bool) {
	val, ok := f.Staking.Load(network)
	if ok {
		return val.(StakingData), true
	}
	return StakingData{}, false
}

func (f *StakingOptimizerFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// NFT Floor Price Fetcher
type NFTFloorPriceFetcher struct {
	*BaseFetcher
	FloorPrices sync.Map // map[string]NFTFloorPrice
}

func NewNFTFloorPriceFetcher() *NFTFloorPriceFetcher {
	return &NFTFloorPriceFetcher{
		BaseFetcher: NewBaseFetcher("NFTFloorPriceFetcher"),
	}
}

func (f *NFTFloorPriceFetcher) Initialize() error {
	fmt.Println("Initializing NFT Floor Price Fetcher...")
	f.SetRunning(true)
	return nil
}

func (f *NFTFloorPriceFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	key := os.Getenv("OPENSEA_API_KEY")
	if key == "" {
		f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
		f.RecordRequest(false)
		return fmt.Errorf("OPENSEA_API_KEY required for floor prices")
	}
	collections := []string{"boredapeyachtclub", "cryptopunks", "pudgypenguins"}
	if v := os.Getenv("FULL_FETCHERS_NFT_COLLECTIONS"); v != "" {
		collections = strings.Split(v, ",")
	}

	for _, slug := range collections {
		var stats struct {
			Total struct {
				Floor float64 `json:"floor_price"`
			} `json:"total"`
			Intervals []struct {
				Interval string  `json:"interval"`
				Volume   float64 `json:"volume"`
				Sales    float64 `json:"sales"`
			} `json:"intervals"`
		}
		url := "https://api.opensea.io/api/v2/collections/" + strings.TrimSpace(slug) + "/stats"
		err := httpGetJSON(ctx, url, map[string]string{"X-API-KEY": key}, &stats)
		if err != nil {
			f.RecordRequest(false)
			return err
		}
		fp := NFTFloorPrice{
			CollectionAddress: Address(slug),
			CollectionName:    slug,
			FloorPriceETH:     stats.Total.Floor,
			LastSale:          currentTimestamp(),
		}
		for _, iv := range stats.Intervals {
			if iv.Interval == "one_day" {
				fp.Volume24h = iv.Volume
				fp.Sales24h = uint64(iv.Sales)
			}
		}
		f.FloorPrices.Store(slug, fp)
	}

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(true)
	return nil
}

func (f *NFTFloorPriceFetcher) GetFloorPrice(collection string) (NFTFloorPrice, bool) {
	val, ok := f.FloorPrices.Load(collection)
	if ok {
		return val.(NFTFloorPrice), true
	}
	return NFTFloorPrice{}, false
}

func (f *NFTFloorPriceFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// Whale Transaction Fetcher
type WhaleTransactionFetcher struct {
	*BaseFetcher
	Transactions []WhaleTransaction
	mu           sync.Mutex
}

func NewWhaleTransactionFetcher() *WhaleTransactionFetcher {
	return &WhaleTransactionFetcher{
		BaseFetcher: NewBaseFetcher("WhaleTransactionFetcher"),
	}
}

func (f *WhaleTransactionFetcher) Initialize() error {
	fmt.Println("Initializing Whale Transaction Fetcher...")
	f.SetRunning(true)
	return nil
}

// transferTopic is the ERC-20 Transfer event signature hash.
const transferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

func (f *WhaleTransactionFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	endpoint := chainRPCEndpoint(1)
	latest, err := ethBlockNumber(ctx, endpoint)
	if err != nil {
		f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
		f.RecordRequest(false)
		return err
	}
	from := latest
	if latest > 60 {
		from = latest - 60
	}

	tokens := []struct {
		addr     string
		symbol   string
		decimals int
		minUSD   float64
	}{
		{"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", "USDC", 6, 250000},
		{"0xdAC17F958D2ee523a2206206994597C13D831ec7", "USDT", 6, 250000},
	}

	for _, tkn := range tokens {
		logs, err := ethGetLogs(ctx, endpoint, logFilter{
			FromBlock: hexQuantity(from),
			ToBlock:   hexQuantity(latest),
			Address:   tkn.addr,
			Topics:    []string{transferTopic},
		})
		if err != nil {
			f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
			f.RecordRequest(false)
			return err
		}
		for _, log := range logs {
			if len(log.Topics) < 3 {
				continue
			}
			amount, err := parseHexBig(log.Data)
			if err != nil {
				continue
			}
			usd := bigToFloat(amount, tkn.decimals)
			if usd < tkn.minUSD {
				continue
			}
			fromAddr := "0x" + strings.TrimPrefix(log.Topics[1], "0x")
			toAddr := "0x" + strings.TrimPrefix(log.Topics[2], "0x")
			blockNum, _ := strconv.ParseUint(strings.TrimPrefix(log.BlockNum, "0x"), 16, 64)
			tx := WhaleTransaction{
				TxHash:      log.TxHash,
				From:        Address(fromAddr),
				To:          Address(toAddr),
				Amount:      TokenAmount(amount.String()),
				AmountUSD:   usd,
				TokenSymbol: tkn.symbol,
				Timestamp:   currentTimestamp(),
				BlockNumber: blockNum,
			}
			f.mu.Lock()
			f.Transactions = append([]WhaleTransaction{tx}, f.Transactions...)
			if len(f.Transactions) > 200 {
				f.Transactions = f.Transactions[:200]
			}
			f.mu.Unlock()
		}
	}

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(true)
	return nil
}

func (f *WhaleTransactionFetcher) GetRecent(limit int) []WhaleTransaction {
	f.mu.Lock()
	defer f.mu.Unlock()

	if limit > len(f.Transactions) {
		limit = len(f.Transactions)
	}
	return f.Transactions[:limit]
}

func (f *WhaleTransactionFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// On-Chain Analytics Fetcher
type OnChainAnalyticsFetcher struct {
	*BaseFetcher
	Analytics sync.Map // map[ChainID]OnChainAnalytics
}

func NewOnChainAnalyticsFetcher() *OnChainAnalyticsFetcher {
	return &OnChainAnalyticsFetcher{
		BaseFetcher: NewBaseFetcher("OnChainAnalyticsFetcher"),
	}
}

func (f *OnChainAnalyticsFetcher) Initialize() error {
	fmt.Println("Initializing On-Chain Analytics Fetcher...")
	f.SetRunning(true)
	return nil
}

type llamaChain struct {
	Name string  `json:"name"`
	Tvl  float64 `json:"tvl"`
}

func (f *OnChainAnalyticsFetcher) Fetch() error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	var ok = true
	var firstErr error
	for _, chainID := range []ChainID{1, 56, 137} {
		endpoint := chainRPCEndpoint(chainID)
		if endpoint == "" {
			continue
		}
		block, err := ethGetBlock(ctx, endpoint, "latest", false)
		if err != nil {
			ok = false
			firstErr = err
			continue
		}
		gasUsed, _ := strconv.ParseUint(strings.TrimPrefix(block.GasUsed, "0x"), 16, 64)
		gp, _ := ethGasPrice(ctx, endpoint)

		data := OnChainAnalytics{
			ChainID:              chainID,
			TotalTransactions24h: float64(len(block.Transactions)),
			AverageGasPrice:      0,
			Timestamp:            currentTimestamp(),
		}
		if gp != nil {
			data.AverageGasPrice = weiToGwei(gp)
		}
		_ = gasUsed
		f.Analytics.Store(chainID, data)
	}
	// Chain TVL from DefiLlama.
	var chains []llamaChain
	if err := httpGetJSON(ctx, "https://api.llama.fi/v2/chains", nil, &chains); err == nil {
		for _, ch := range chains {
			if ch.Name == "Ethereum" {
				if val, ok2 := f.Analytics.Load(ChainID(1)); ok2 {
					d := val.(OnChainAnalytics)
					d.TotalValueLocked = ch.Tvl
					d.DeFiTVL = ch.Tvl
					f.Analytics.Store(ChainID(1), d)
				}
			}
		}
	}

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(ok)
	return firstErr
}

func (f *OnChainAnalyticsFetcher) GetAnalytics(chainID ChainID) (OnChainAnalytics, bool) {
	val, ok := f.Analytics.Load(chainID)
	if ok {
		return val.(OnChainAnalytics), true
	}
	return OnChainAnalytics{}, false
}

func (f *OnChainAnalyticsFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// Transaction Simulator Fetcher
type TransactionSimulatorFetcher struct {
	*BaseFetcher
}

func NewTransactionSimulatorFetcher() *TransactionSimulatorFetcher {
	return &TransactionSimulatorFetcher{
		BaseFetcher: NewBaseFetcher("TransactionSimulatorFetcher"),
	}
}

func (f *TransactionSimulatorFetcher) Initialize() error {
	fmt.Println("Initializing Transaction Simulator Fetcher...")
	f.SetRunning(true)
	return nil
}

func (f *TransactionSimulatorFetcher) Fetch() error {
	start := time.Now()

	// Simulations are done on-demand

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(true)

	return nil
}

// SimulateWithError executes a real stateless simulation on the node:
// eth_estimateGas for gas accounting and eth_call for the revert verdict.
// No transaction is broadcast; TxHash therefore stays empty.
func (f *TransactionSimulatorFetcher) SimulateWithError(from, to Address, value TokenAmount, data string, chainID ChainID) (SimulationResult, error) {
	endpoint := chainRPCEndpoint(chainID)
	if endpoint == "" {
		return SimulationResult{}, fmt.Errorf("no RPC endpoint for chain %d", uint64(chainID))
	}
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	msg := callMsg{From: string(from), To: string(to)}
	if data != "" && data != "0x" {
		msg.Data = data
	}
	if value != "" && value != "0" {
		if v, err := parseTokenAmount(value); err == nil {
			msg.Value = hexQuantity(v.Uint64())
		}
	}

	result := SimulationResult{StateChanges: "{}", SimulatedAt: currentTimestamp()}
	gas, gasErr := ethEstimateGas(ctx, endpoint, msg)
	if gasErr == nil {
		result.GasUsed = gas
	}
	_, callErr := ethCall(ctx, endpoint, msg, "latest")
	if callErr == nil {
		result.Success = true
		return result, nil
	}
	result.Success = false
	if rerr, ok := callErr.(*rpcError); ok {
		result.RevertReason = decodeRevertReason(rerr.Data)
		if result.RevertReason == "" {
			result.RevertReason = rerr.Message
		}
	}
	return result, nil
}

// Simulate keeps the legacy signature; diagnostics require SimulateWithError.
func (f *TransactionSimulatorFetcher) Simulate(from, to Address, value TokenAmount, data string, chainID ChainID) SimulationResult {
	result, err := f.SimulateWithError(from, to, value, data, chainID)
	if err != nil {
		return SimulationResult{Success: false, StateChanges: "{}", SimulatedAt: currentTimestamp()}
	}
	return result
}

func (f *TransactionSimulatorFetcher) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// Cross-Chain Route Optimizer
type CrossChainRouteOptimizer struct {
	*BaseFetcher
}

func NewCrossChainRouteOptimizer() *CrossChainRouteOptimizer {
	return &CrossChainRouteOptimizer{
		BaseFetcher: NewBaseFetcher("CrossChainRouteOptimizer"),
	}
}

func (f *CrossChainRouteOptimizer) Initialize() error {
	fmt.Println("Initializing Cross-Chain Route Optimizer...")
	f.SetRunning(true)
	return nil
}

func (f *CrossChainRouteOptimizer) Fetch() error {
	start := time.Now()

	// Fetch routes

	f.UpdateLatency(uint64(time.Since(start).Nanoseconds()))
	f.RecordRequest(true)

	return nil
}

// FindBestRouteWithError quotes a real cross-chain route through LI.FI
// (https://li.quest/v1). Chain identifiers are LI.FI chain keys (eth, bsc,
// pol, arb, ...). Amount is an integer in the token's base units.
func (f *CrossChainRouteOptimizer) FindBestRouteWithError(fromChain, toChain string, fromToken, toToken Address, amount TokenAmount) (CrossChainRoute, error) {
	base := os.Getenv("LIFI_API_URL")
	if base == "" {
		base = "https://li.quest/v1"
	}
	if _, err := parseTokenAmount(amount); err != nil {
		return CrossChainRoute{}, err
	}
	url := fmt.Sprintf("%s/quote?fromChain=%s&toChain=%s&fromToken=%s&toToken=%s&fromAmount=%s&fromAddress=0x000000000000000000000000000000000000dEaD",
		strings.TrimRight(base, "/"),
		strings.TrimSpace(fromChain), strings.TrimSpace(toChain),
		string(fromToken), string(toToken), string(amount))

	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()
	var resp struct {
		Tool          string `json:"tool"`
		IncludedSteps []struct {
			Tool string `json:"tool"`
		} `json:"includedSteps"`
		Estimate struct {
			ToAmount          string  `json:"toAmount"`
			ExecutionDuration float64 `json:"executionDuration"`
			FeeCosts          []struct {
				AmountUSD string `json:"amountUSD"`
			} `json:"feeCosts"`
			GasCosts []struct {
				AmountUSD string `json:"amountUSD"`
			} `json:"gasCosts"`
		} `json:"estimate"`
	}
	if err := httpGetJSON(ctx, url, nil, &resp); err != nil {
		return CrossChainRoute{}, err
	}

	feeUSD := 0.0
	for _, c := range resp.Estimate.FeeCosts {
		if v, err := strconv.ParseFloat(c.AmountUSD, 64); err == nil {
			feeUSD += v
		}
	}
	for _, c := range resp.Estimate.GasCosts {
		if v, err := strconv.ParseFloat(c.AmountUSD, 64); err == nil {
			feeUSD += v
		}
	}
	steps := []BridgeStep{}
	for _, step := range resp.IncludedSteps {
		steps = append(steps, BridgeStep{
			Protocol:  step.Tool,
			FromChain: fromChain,
			ToChain:   toChain,
			FromToken: fromToken,
			ToToken:   toToken,
		})
	}
	if len(steps) == 0 {
		steps = []BridgeStep{{Protocol: resp.Tool, FromChain: fromChain, ToChain: toChain, FromToken: fromToken, ToToken: toToken}}
	}

	return CrossChainRoute{
		FromChain:            fromChain,
		ToChain:              toChain,
		FromToken:            fromToken,
		ToToken:              toToken,
		FromAmount:           amount,
		ToAmount:             TokenAmount(resp.Estimate.ToAmount),
		EstimatedTimeMinutes: uint64(resp.Estimate.ExecutionDuration / 60),
		TotalFeeUSD:          feeUSD,
		Steps:                steps,
	}, nil
}

// FindBestRoute keeps the legacy signature; failures return an empty route.
func (f *CrossChainRouteOptimizer) FindBestRoute(fromChain, toChain string, fromToken, toToken Address, amount TokenAmount) CrossChainRoute {
	route, err := f.FindBestRouteWithError(fromChain, toChain, fromToken, toToken, amount)
	if err != nil {
		return CrossChainRoute{
			FromChain:  fromChain,
			ToChain:    toChain,
			FromToken:  fromToken,
			ToToken:    toToken,
			FromAmount: amount,
		}
	}
	return route
}

func (f *CrossChainRouteOptimizer) Shutdown() error {
	f.SetRunning(false)
	return nil
}

// =============================================================================
// FETCHER MANAGER
// =============================================================================

type FullFetcherManager struct {
	fetchers map[string]interface {
		Initialize() error
		Fetch() error
		Shutdown() error
		GetStats() FetcherStats
	}
	running atomic.Bool
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewFullFetcherManager() *FullFetcherManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &FullFetcherManager{
		fetchers: make(map[string]interface {
			Initialize() error
			Fetch() error
			Shutdown() error
			GetStats() FetcherStats
		}),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (m *FullFetcherManager) AddFetcher(name string, fetcher interface {
	Initialize() error
	Fetch() error
	Shutdown() error
	GetStats() FetcherStats
}) {
	m.fetchers[name] = fetcher
}

func (m *FullFetcherManager) InitializeAll() error {
	fmt.Println("Initializing all fetchers...")

	for name, fetcher := range m.fetchers {
		if err := fetcher.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize %s: %w", name, err)
		}
	}

	fmt.Println("All fetchers initialized successfully!")
	return nil
}

func (m *FullFetcherManager) StartAll() {
	m.running.Store(true)

	for name, fetcher := range m.fetchers {
		m.wg.Add(1)
		go func(name string, fetcher interface {
			Fetch() error
			Shutdown() error
			GetStats() FetcherStats
		}) {
			defer m.wg.Done()

			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for m.running.Load() {
				select {
				case <-m.ctx.Done():
					return
				case <-ticker.C:
					if err := fetcher.Fetch(); err != nil {
						fmt.Printf("Fetcher %s error: %v\n", name, err)
					}
				}
			}
		}(name, fetcher)
	}
}

func (m *FullFetcherManager) StopAll() {
	fmt.Println("Stopping all fetchers...")
	m.running.Store(false)
	m.cancel()
	m.wg.Wait()

	for _, fetcher := range m.fetchers {
		fetcher.Shutdown()
	}
}

func (m *FullFetcherManager) GetStats() map[string]FetcherStats {
	stats := make(map[string]FetcherStats)

	for name, fetcher := range m.fetchers {
		stats[name] = fetcher.GetStats()
	}

	return stats
}

func (m *FullFetcherManager) PrintStats() {
	fmt.Println("\n=== Fetcher Statistics (Go) ===")

	for name, stats := range m.GetStats() {
		fmt.Printf("%s: latency=%dns, requests=%d, success_rate=%.2f%%\n",
			name, stats.LastLatencyNs, stats.TotalRequests, stats.SuccessRate)
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func currentTimestamp() Timestamp {
	return Timestamp(time.Now().UnixMilli())
}

func generateHex(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

// =============================================================================
// REAL DATA HELPERS
// =============================================================================

// bigToFloat converts a raw integer amount to a human float by decimals.
func bigToFloat(n *big.Int, decimals int) float64 {
	f := new(big.Float).SetInt(n)
	d := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	v, _ := new(big.Float).Quo(f, d).Float64()
	return v
}

// parseTokenAmount parses a positive integer amount expressed in base units.
func parseTokenAmount(a TokenAmount) (*big.Int, error) {
	str := strings.TrimSpace(string(a))
	if str == "" {
		return nil, fmt.Errorf("empty amount")
	}
	n, ok := new(big.Int).SetString(str, 10)
	if !ok || n.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be a positive integer in base units")
	}
	return n, nil
}

// linearRegression least-squares fit over (timestamp_ms, price) points.
// Returns slope per millisecond, intercept, and R-squared goodness of fit.
func linearRegression(points [][2]float64) (slopePerMs, intercept, r2 float64) {
	n := float64(len(points))
	if n < 2 {
		return 0, 0, 0
	}
	t0 := points[0][0]
	var sx, sy, sxx, sxy float64
	for _, pt := range points {
		x := pt[0] - t0
		y := pt[1]
		sx += x
		sy += y
		sxx += x * x
		sxy += x * y
	}
	denom := n*sxx - sx*sx
	if denom == 0 {
		return 0, sy / n, 0
	}
	slope := (n*sxy - sx*sy) / denom
	intercept = (sy - slope*sx) / n
	mean := sy / n
	var ssRes, ssTot float64
	for _, pt := range points {
		x := pt[0] - t0
		d := pt[1] - (slope*x + intercept)
		ssRes += d * d
		dt := pt[1] - mean
		ssTot += dt * dt
	}
	if ssTot == 0 {
		return slope, intercept, 0
	}
	return slope, intercept, 1 - ssRes/ssTot
}

// decodeRevertReason decodes a standard Error(string) revert payload (0x08c379a0).
func decodeRevertReason(data string) string {
	data = strings.TrimPrefix(data, "0x")
	if !strings.HasPrefix(data, "08c379a0") {
		return ""
	}
	return decodeABIString("0x" + data[8:])
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("TigerWallet Full Fetchers - Go Distributed System")
	fmt.Println("==================================================")

	// Create manager
	manager := NewFullFetcherManager()

	// Add standard fetchers
	manager.AddFetcher("erc20", NewERC20TokenFetcher())
	manager.AddFetcher("gas", NewGasEstimatorFetcher())
	manager.AddFetcher("price", NewPriceFeedFetcher())
	manager.AddFetcher("dapp", NewDAppConnectionFetcher())
	manager.AddFetcher("network", NewNetworkFetcher())
	manager.AddFetcher("swap", NewSwapQuoteFetcher())

	// Add advanced fetchers
	manager.AddFetcher("ai_price", NewAIPricePredictorFetcher())
	manager.AddFetcher("mev", NewMEVOpportunityFetcher())
	manager.AddFetcher("liquidity", NewLiquidityFetcher())
	manager.AddFetcher("arbitrage", NewArbitrageFetcher())
	manager.AddFetcher("risk", NewTokenRiskFetcher())
	manager.AddFetcher("contract", NewSmartContractFetcher())
	manager.AddFetcher("gas_market", NewGasMarketFetcher())
	manager.AddFetcher("yield", NewDeFiYieldFetcher())
	manager.AddFetcher("staking", NewStakingOptimizerFetcher())
	manager.AddFetcher("nft_floor", NewNFTFloorPriceFetcher())
	manager.AddFetcher("whale", NewWhaleTransactionFetcher())
	manager.AddFetcher("analytics", NewOnChainAnalyticsFetcher())
	manager.AddFetcher("simulator", NewTransactionSimulatorFetcher())
	manager.AddFetcher("cross_chain", NewCrossChainRouteOptimizer())

	// Initialize
	if err := manager.InitializeAll(); err != nil {
		fmt.Printf("Failed to initialize: %v\n", err)
		return
	}

	// Start
	manager.StartAll()

	// Run for a bit then print stats
	for i := 0; i < 5; i++ {
		time.Sleep(2 * time.Second)
		manager.PrintStats()
	}

	// Stop
	manager.StopAll()

	fmt.Println("All fetchers stopped.")
}
