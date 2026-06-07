// ============================================================================
// TIGERWALLET DEX AGGREGATOR
// Multi-DEX routing and swap execution
// ============================================================================

package dex

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// SwapRequest represents a swap request
type SwapRequest struct {
	FromToken  string  `json:"from_token"`
	ToToken   string  `json:"to_token"`
	Amount    string  `json:"amount"`
	FromAddress string `json:"from_address"`
	ChainID   uint64  `json:"chain_id"`
	Slippage  float64 `json:"slippage"`
	GasPrice  string `json:"gas_price,omitempty"`
}

// SwapResult represents a swap result
type SwapResult struct {
	TxHash          string      `json:"tx_hash"`
	FromToken      string      `json:"from_token"`
	ToToken       string      `json:"to_token"`
	FromAmount    string      `json:"from_amount"`
	ToAmount     string      `json:"to_amount"`
	MinReceived  string     `json:"min_received"`
	Route        []RouteHop `json:"route"`
	GasUsed      string     `json:"gas_used"`
	GasFee       string     `json:"gas_fee"`
	PriceImpact float64   `json:"price_impact"`
	ExecutionTime int64   `json:"execution_time"`
}

// RouteHop represents a single hop in a swap route
type RouteHop struct {
	DEX     string `json:"dex"`
	FromToken string `json:"from_token"`
	ToToken  string `json:"to_token"`
	FromAmount string `json:"from_amount"`
	ToAmount  string `json:"to_amount"`
}

// Quote represents a swap quote
type Quote struct {
	FromToken    string     `json:"from_token"`
	ToToken     string     `json:"to_token"`
	FromAmount  string     `json:"from_amount"`
	ToAmount    string     `json:"to_amount"`
	MinReceived  string     `json:"min_received"`
	Route       []RouteHop `json:"route"`
	GasEstimate string    `json:"gas_estimate"`
	PriceImpact float64   `json:"price_impact"`
}

// DEXConfig represents a DEX configuration
type DEXConfig struct {
	Name      string `json:"name"`
	Protocol string `json:"protocol"`
	Router   string `json:"router"`
	Factory  string `json:"factory"`
	Enabled  bool   `json:"enabled"`
}

// SupportedDEXes contains DEX configurations
var SupportedDEXes = map[string]DEXConfig{
	"uniswap_v3": {
		Name:      "Uniswap V3",
		Protocol: "uniswapv3",
		Router:   "0xE592427A0AEce92De3E41F7A1e3f8f1dB4a3F4a5",
		Factory:  "0x1F98431c8aD98523631AEb5F9F1e4e5D5F1e4e5d",
		Enabled:  true,
	},
	"uniswap_v2": {
		Name:      "Uniswap V2",
		Protocol: "uniswapv2",
		Router:   "0x7a250d5630B4cF539739dF2c5dC5bC1c3B3F4a5",
		Factory:  "0x5C69B701f05D5F5dD5F5dC5bC1c3B3F4a5",
		Enabled:  true,
	},
	"sushiswap": {
		Name:      "SushiSwap",
		Protocol: "sushiswap",
		Router:   "0xd9e1CE17F2641f24aE83615d354d4fF3dD4a3F4a5",
		Factory:  "0xC0AEe478e31FE6f0B3d4C5bC1c3B3F4a5",
		Enabled:  true,
	},
	"curve": {
		Name:      "Curve",
		Protocol: "curve",
		Router:   "0x8f3Cf7ad23Cd3CaDbD4C5dC5bC1c3B3F4a5",
		Factory:  "",
		Enabled:  true,
	},
	"balancer": {
		Name:      "Balancer",
		Protocol: "balancer",
		Router:   "0xBA12222222228d801Cd9C3C5bC1c3B3F4a5",
		Factory:  "0x94435A26D25C5bC1c3B3F4a5",
		Enabled:  true,
	},
	"pancakeswap": {
		Name:      "PancakeSwap",
		Protocol: "pancakeswap",
		Router:   "0x10ED43C718714EB63D5aA56041F4d4fF3dD4a3F4a5",
		Factory:  "0x0eD7e4761615f5dD5F5dC5bC1c3B3F4a5",
		Enabled:  true,
	},
	"traderjoe": {
		Name:      "Trader Joe",
		Protocol: "traderjoe",
		Router:   "0xD9e1CE17F2641f24aE83615d354d4fF3dD4a3F4a5",
		Factory:  "0xD9e1CE17F2641f24aE83615d354d4fF3d",
		Enabled:  true,
	},
	"quickswap": {
		Name:      "QuickSwap",
		Protocol: "quickswap",
		Router:   "0xa5E8b4F1D4fF3dD4a3F4a5",
		Factory:  "0x0eD7e4761615f5dD5F5dC5bC1c3B3F4a5",
		Enabled:  true,
	},
}

// Aggregator handles multi-DEX routing
type Aggregator struct {
	DEXes     map[string]DEXConfig
	APIKey    string
}

// NewAggregator creates a new DEX aggregator
func NewAggregator(apiKey string) *Aggregator {
	dexes := make(map[string]DEXConfig)
	for name, config := range SupportedDEXes {
		if config.Enabled {
			dexes[name] = config
		}
	}

	return &Aggregator{
		DEXes:  dexes,
		APIKey: apiKey,
	}
}

// GetQuote gets the best quote from all DEXes
func (agg *Aggregator) GetQuote(req *SwapRequest) (*Quote, error) {
	if req.FromToken == "" || req.ToToken == "" || req.Amount == "" {
		return nil, errors.New("invalid swap request")
	}

	// Get quotes from all DEXes
	var quotes []Quote
	for name, dex := range agg.DEXes {
		quote, err := agg.getDEXQuote(name, dex, req)
		if err != nil {
			continue
		}
		quotes = append(quotes, *quote)
	}

	if len(quotes) == 0 {
		return nil, errors.New("no quotes available")
	}

	// Find best quote (highest output)
	best := quotes[0]
	for i := range quotes {
		if compareAmount(quotes[i].ToAmount, best.ToAmount) > 0 {
			best = quotes[i]
		}
	}

	return &best, nil
}

// getDEXQuote gets a quote from a specific DEX
func (agg *Aggregator) getDEXQuote(name string, dex DEXConfig, req *SwapRequest) (*Quote, error) {
	// Simulate quote calculation (in production, call DEX API)
	fromAmount := new(big.Float)
	fromAmount.SetString(req.Amount)

	// Simulate 0.3% fee + price impact
	toAmount := new(big.Float).Mul(fromAmount, big.NewFloat(0.997))
	toAmountStr := toAmount.Text('f', 0)

	return &Quote{
		FromToken:    req.FromToken,
		ToToken:      req.ToToken,
		FromAmount:   req.Amount,
		ToAmount:    toAmountStr,
		MinReceived:  toAmountStr,
		Route:        []RouteHop{{DEX: name, FromToken: req.FromToken, ToToken: req.ToToken, FromAmount: req.Amount, ToAmount: toAmountStr}},
		GasEstimate:  "150000",
		PriceImpact:  0.1,
	}, nil
}

// compareAmount compares two token amounts
func compareAmount(a, b string) int {
	af := new(big.Float)
	af.SetString(a)
	bf := new(big.Float)
	bf.SetString(b)

	if af.Cmp(bf) > 0 {
		return 1
	} else if af.Cmp(bf) < 0 {
		return -1
	}
	return 0
}

// ExecuteSwap executes a swap
func (agg *Aggregator) ExecuteSwap(req *SwapRequest) (*SwapResult, error) {
	// Get quote first
	quote, err := agg.GetQuote(req)
	if err != nil {
		return nil, err
	}

	// Apply slippage
	minReceived := applySlippage(quote.MinReceived, req.Slippage)

	// Build transaction
	tx := &SwapResult{
		FromToken:     req.FromToken,
		ToToken:       req.ToToken,
		FromAmount:    req.Amount,
		ToAmount:      quote.MinReceived,
		MinReceived:   minReceived,
		Route:         quote.Route,
		GasUsed:       quote.GasEstimate,
		GasFee:        calculateGasFee(quote.GasEstimate, req.GasPrice),
		PriceImpact:   quote.PriceImpact,
		ExecutionTime: 0,
	}

	return tx, nil
}

// applySlippage applies slippage tolerance
func applySlippage(amount string, slippage float64) string {
	amountF := new(big.Float)
	amountF.SetString(amount)

	slippageMultiplier := big.NewFloat(1 - slippage/100)
	result := new(big.Float).Mul(amountF, slippageMultiplier)

	return result.Text('f', 0)
}

// calculateGasFee calculates the gas fee
func calculateGasFee(gasEstimate, gasPrice string) string {
	ge := new(big.Int)
	ge.SetString(gasEstimate, 10)

	gp := new(big.Int)
	if gasPrice == "" {
		gp.SetString("20000000000") // 20 Gwei default
	} else {
		gp.SetString(gasPrice, 10)
	}

	fee := new(big.Int).Mul(ge, gp)
	return fee.String()
}

// GetRoutes gets all available routes for a swap
func (agg *Aggregator) GetRoutes(req *SwapRequest) ([][]RouteHop, error) {
	// Build all possible routes
	var routes [][]RouteHop

	for name := range agg.DEXes {
		dex := agg.DEXes[name]
		route := []RouteHop{
			{DEX: name, FromToken: req.FromToken, ToToken: req.ToToken, FromAmount: req.Amount},
		}
		routes = append(routes, route)
	}

	// Add multi-hop routes for complex swaps
	// (simplified - in production would calculate paths)

	return routes, nil
}

// FindOptimalRoute finds the optimal route considering gas and price
func (agg *Aggregator) FindOptimalRoute(req *SwapRequest) ([]RouteHop, error) {
	routes, err := agg.GetRoutes(req)
	if err != nil {
		return nil, err
	}

	// Sort by total output (simplified)
	sort.Slice(routes, func(i, j int) bool {
		return false // Would compare outputs
	})

	if len(routes) == 0 {
		return nil, errors.New("no routes found")
	}

	return routes[0], nil
}

// BridgeRequest represents a bridge request
type BridgeRequest struct {
	FromChain  uint64 `json:"from_chain"`
	ToChain   uint64 `json:"to_chain"`
	Token     string `json:"token"`
	Amount    string `json:"amount"`
	Recipient string `json:"recipient"`
	Slippage  float64 `json:"slippage"`
}

// BridgeQuote represents a bridge quote
type BridgeQuote struct {
	FromChain  uint64     `json:"from_chain"`
	ToChain   uint64     `json:"to_chain"`
	FromToken string     `json:"from_token"`
	ToToken   string     `json:"to_token"`
	FromAmount string   `json:"from_amount"`
	ToAmount  string    `json:"to_amount"`
	Fee       string    `json:"fee"`
	EstimatedTime string  `json:"estimated_time"`
	Protocol  string    `json:"protocol"`
}

// SupportedBridges contains bridge configurations
var SupportedBridges = map[string]BridgeConfig{
	"stargate": {
		Name:     "Stargate",
		Protocol: "stargate",
	},
	"layerzero": {
		Name:     "LayerZero",
		Protocol: "layerzero",
	},
	"axelar": {
		Name:     "Axelar",
		Protocol: "axelar",
	},
	"wormhole": {
		Name:     "Wormhole",
		Protocol: "wormhole",
	},
}

// BridgeConfig represents a bridge configuration
type BridgeConfig struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
}

// GetBridgeQuote gets a bridge quote
func GetBridgeQuote(req *BridgeRequest) (*BridgeQuote, error) {
	if req.FromChain == req.ToChain {
		return nil, errors.New("same source and destination chain")
	}

	return &BridgeQuote{
		FromChain:     req.FromChain,
		ToChain:     req.ToChain,
		FromToken:   req.Token,
		ToToken:    req.Token,
		FromAmount: req.Amount,
		ToAmount:   req.Amount,
		Fee:       "0",
		EstimatedTime: "10m",
		Protocol:  "stargate",
	}, nil
}