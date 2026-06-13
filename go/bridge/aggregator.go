package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"
)

// ============================================================================
// Cross-Chain Bridge Aggregator - Go Implementation
// ============================================================================

// BridgeAggregator finds the best cross-chain routes across multiple bridges
type BridgeAggregator struct {
	mu         sync.RWMutex
	bridges    map[string]Bridge
	tokens    map[string]TokenConfig
	gasPrices map[uint64]*GasPrice
	config    *Config
}

// Bridge represents a bridge protocol
type Bridge interface {
	Name() string
	SupportedChains() []uint64
	GetQuote(ctx context.Context, req QuoteRequest) (QuoteResponse, error)
	Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error)
	GetStatus(ctx context.Context, txHash string) (BridgeStatus, error)
}

// TokenConfig represents token configuration on a chain
type TokenConfig struct {
	ChainID      uint64
	Address     string
	Symbol      string
	Decimals    uint8
	WrappedAddr map[uint64]string // chainID -> wrapped token address
}

// GasPrice represents gas price data
type GasPrice struct {
	ChainID       uint64
	BaseFee      *big.Int
	MaxFee       *big.Int
	PriorityFee *big.Int
	UpdatedAt   time.Time
}

// Config for bridge aggregator
type Config struct {
	MaxBridgeTime   time.Duration
	MaxSlippage   float64
	MinLiquidity  *big.Int
	EnableMultiHop bool
}

// QuoteRequest for bridge quote
type QuoteRequest struct {
	SrcChain    uint64
	DstChain   uint64
	SrcToken   string
	DstToken  string
	Amount    *big.Int
	Recipient string
}

// QuoteResponse from bridge
type QuoteResponse struct {
	Bridge        string
	SrcToken      string
	DstToken     string
	AmountIn    *big.Int
	AmountOut   *big.Int
	BridgeFee   *big.Int
	GasEstimate uint64
	Duration   time.Duration
	Slippage    float64
	Route      []Hop
}

// Hop represents a single bridge hop
type Hop struct {
	Bridge   string
	Chain    uint64
	TokenIn  string
	TokenOut string
}

// ExecuteRequest for bridge execution
type ExecuteRequest struct {
	QuoteID    string
	Recipient string
	Deadline  time.Time
}

// ExecuteResponse from bridge
type ExecuteResponse struct {
	TxHash      string
	SrcTxHash   string
	DstTxHash   string
	AmountOut  *big.Int
	Status     BridgeStatus
	ExecutedAt time.Time
}

// BridgeStatus enum
type BridgeStatus int

const (
	StatusPending BridgeStatus = iota
	StatusSrcConfirmed
	StatusDstConfirmed
	StatusCompleted
	StatusFailed
)

// NewBridgeAggregator creates new aggregator
func NewBridgeAggregator(cfg *Config) *BridgeAggregator {
	return &BridgeAggregator{
		bridges:    make(map[string]Bridge),
		tokens:    make(map[string]TokenConfig),
		gasPrices: make(map[uint64]*GasPrice),
		config:    cfg,
	}
}

// RegisterBridge adds a bridge
func (b *BridgeAggregator) RegisterBridge(name string, bridge Bridge) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.bridges[name] = bridge
}

// GetQuote finds best bridge quote
func (b *BridgeAggregator) GetQuote(ctx context.Context, req QuoteRequest) (*QuoteResponse, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Collect quotes from all bridges
	type quoteResult struct {
		bridge string
		quote QuoteResponse
		err   error
	}
	
	ch := make(chan quoteResult, len(b.bridges))
	
	for name, bridge := range b.bridges {
		go func(name string, bridge Bridge) {
			quote, err := bridge.GetQuote(ctx, req)
			ch <- quoteResult{name, quote, err}
		}(name, bridge)
	}
	
	var quotes []quoteResult
	timeout := time.After(5 * time.Second)
	
	for i := 0; i < len(b.bridges); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			break
		case qr := <-ch:
			if qr.err == nil {
				quotes = append(quotes, qr)
			}
		}
	}
	
	if len(quotes) == 0 {
		return nil, fmt.Errorf("no bridges available")
	}
	
	// Sort by amount out (descending)
	sort.Slice(quotes, func(i, j int) bool {
		return quotes[i].quote.AmountOut.Cmp(quotes[j].quote.AmountOut) > 0
	})
	
	resp := &quotes[0].quote
	resp.Bridge = quotes[0].bridge
	
	return resp, nil
}

// Execute executes a bridge transfer
func (b *BridgeAggregator) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResponse, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	bridge, ok := b.bridges[req.QuoteID]
	if !ok {
		return nil, fmt.Errorf("bridge not found: %s", req.QuoteID)
	}
	
	return bridge.Execute(ctx, req)
}

// MultiHopRoute finds multi-hop routes
func (b *BridgeAggregator) MultiHopRoute(ctx context.Context, req QuoteRequest) ([]QuoteResponse, error) {
	if !b.config.EnableMultiHop {
		return nil, fmt.Errorf("multi-hop disabled")
	}
	
	// Find intermediate chains
	intermediateChains := b.findIntermediateChains(req.SrcChain, req.DstChain)
	
	var routes []QuoteResponse
	for _, chain := range intermediateChains {
		// Quote src -> intermediate
		req1 := QuoteRequest{
			SrcChain: req.SrcChain,
			DstChain: chain,
			SrcToken: req.SrcToken,
			DstToken: "", // find best token
			Amount:  req.Amount,
		}
		
		quote1, err := b.GetQuote(ctx, req1)
		if err != nil {
			continue
		}
		
		// Quote intermediate -> dst
		req2 := QuoteRequest{
			SrcChain: chain,
			DstChain: req.DstChain,
			SrcToken: quote1.DstToken,
			DstToken: req.DstToken,
			Amount:  quote1.AmountOut,
		}
		
		quote2, err := b.GetQuote(ctx, req2)
		if err != nil {
			continue
		}
		
		// Combine routes
		combined := QuoteResponse{
			Bridge:   fmt.Sprintf("%s->%s", quote1.Bridge, quote2.Bridge),
			AmountIn: req.Amount,
			AmountOut: quote2.AmountOut,
			BridgeFee: new(big.Int).Add(quote1.BridgeFee, quote2.BridgeFee),
			Duration: quote1.Duration + quote2.Duration,
			Slippage: quote1.Slippage + quote2.Slippage,
			Route:   append(quote1.Route, quote2.Route...),
		}
		
		routes = append(routes, combined)
	}
	
	// Sort by amount out
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].AmountOut.Cmp(routes[j].AmountOut) > 0
	})
	
	return routes, nil
}

func (b *BridgeAggregator) findIntermediateChains(src, dst uint64) []uint64 {
	// Common intermediate chains for cross-chain swaps
	intermediates := []uint64{1, 56, 137, 42161} // ETH, BSC, POLY, ARB
	
	var result []uint64
	for _, chain := range intermediates {
		if chain != src && chain != dst {
			result = append(result, chain)
		}
	}
	
	return result
}

// ============================================================================
// Bridge Implementations
// ============================================================================

// StargateBridge represents Stargate bridge
type StargateBridge struct {
	rpcURL string
}

// NewStargateBridge creates Stargate bridge
func NewStargateBridge(rpcURL string) *StargateBridge {
	return &StargateBridge{rpcURL: rpcURL}
}

func (s *StargateBridge) Name() string        { return "Stargate" }
func (s *StargateBridge) SupportedChains() []uint64 { return []uint64{1, 56, 137, 42161, 10, 8453} }

func (s *StargateBridge) GetQuote(ctx context.Context, req QuoteRequest) (QuoteResponse, error) {
	// Simulate quote - in production would call API
	return QuoteResponse{
		Bridge:     "Stargate",
		SrcToken:    req.SrcToken,
		DstToken:    req.DstToken,
		AmountIn:   req.Amount,
		AmountOut:  new(big.Int).Mul(req.Amount, big.NewInt(99)),
		BridgeFee:  big.NewInt(0),
		Duration:  5 * time.Minute,
		Slippage:  0.5,
		Route:     []Hop{{Bridge: "Stargate", Chain: req.DstChain}},
	}, nil
}

func (s *StargateBridge) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	return ExecuteResponse{
		TxHash:     "0x" + generateHash(64),
		Status:     StatusPending,
		ExecutedAt: time.Now(),
	}, nil
}

func (s *StargateBridge) GetStatus(ctx context.Context, txHash string) (BridgeStatus, error) {
	return StatusCompleted, nil
}

// AcrossBridge represents Across bridge
type AcrossBridge struct {
	rpcURL string
}

// NewAcrossBridge creates Across bridge
func NewAcrossBridge(rpcURL string) *AcrossBridge {
	return &AcrossBridge{rpcURL: rpcURL}
}

func (a *AcrossBridge) Name() string        { return "Across" }
func (a *AcrossBridge) SupportedChains() []uint64 { return []uint64{1, 10, 42161, 137, 8453} }

func (a *AcrossBridge) GetQuote(ctx context.Context, req QuoteRequest) (QuoteResponse, error) {
	return QuoteResponse{
		Bridge:     "Across",
		SrcToken:   req.SrcToken,
		DstToken:   req.DstToken,
		AmountIn:  req.Amount,
		AmountOut: new(big.Int).Mul(req.Amount, big.NewInt(98)),
		BridgeFee:  big.NewInt(0),
		Duration:  3 * time.Minute,
		Slippage:  0.3,
		Route:     []Hop{{Bridge: "Across", Chain: req.DstChain}},
	}, nil
}

func (a *AcrossBridge) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	return ExecuteResponse{
		TxHash:     "0x" + generateHash(64),
		Status:    StatusPending,
		ExecutedAt: time.Now(),
	}, nil
}

func (a *AcrossBridge) GetStatus(ctx context.Context, txHash string) (BridgeStatus, error) {
	return StatusCompleted, nil
}

// ============================================================================
// Utilities
// ============================================================================

func generateHash(length int) string {
	charset := "0123456789abcdef"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================================
// JSON Serialization
// ============================================================================

func (q QuoteRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		SrcChain    uint64 `json:"srcChain"`
		DstChain    uint64 `json:"dstChain"`
		SrcToken    string `json:"srcToken"`
		DstToken    string `json:"dstToken"`
		Amount     string `json:"amount"`
		Recipient  string `json:"recipient"`
	}{
		SrcChain:    q.SrcChain,
		DstChain:   q.DstChain,
		SrcToken:   q.SrcToken,
		DstToken:  q.DstToken,
		Amount:   q.Amount.String(),
		Recipient: q.Recipient,
	})
}

func (q *QuoteRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		SrcChain    uint64 `json:"srcChain"`
		DstChain   uint64 `json:"dstChain"`
		SrcToken   string `json:"srcToken"`
		DstToken  string `json:"dstToken"`
		Amount    string `json:"amount"`
		Recipient string `json:"recipient"`
	}
	
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	
	q.SrcChain = raw.SrcChain
	q.DstChain = raw.DstChain
	q.SrcToken = raw.SrcToken
	q.DstToken = raw.DstToken
	q.Amount = new(big.Int)
	q.Amount.SetString(raw.Amount, 10)
	q.Recipient = raw.Recipient
	
	return nil
}