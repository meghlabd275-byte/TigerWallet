/**
 * TigerWallet Cross-Chain Bridge Service - Complete Implementation
 * 
 * Multi-protocol bridge aggregation with Stargate, LayerZero, Across, Hop
 * High-performance Go service for worldwide distribution
 */

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// TYPES AND STRUCTURES
// ============================================================================

// Chain information
type Chain struct {
	ID           uint64   `json:"id"`
	Name         string   `json:"name"`
	Symbol       string   `json:"symbol"`
	Type         string   `json:"type"` // evm, solana, ton, etc.
	ChainID      string   `json:"chain_id"`
	NativeToken  string   `json:"native_token"`
	ExplorerURL  string   `json:"explorer_url"`
	RPCURLs      []string `json:"rpc_urls"`
	Confirmations uint64   `json:"confirmations"`
}

// Bridge protocol
type BridgeProtocol struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Logo        string   `json:"logo"`
	Chains      []uint64 `json:"chains"`
	Tokens      []string `json:"tokens"`
	FeePercent  float64  `json:"fee_percent"`
	AvgTime     string   `json:"avg_time"`
	IsActive    bool     `json:"is_active"`
}

// Bridge quote request
type BridgeQuoteRequest struct {
	FromChain    uint64   `json:"from_chain"`
	ToChain      uint64   `json:"to_chain"`
	FromToken    string   `json:"from_token"`
	ToToken      string   `json:"to_token"`
	Amount       string   `json:"amount"`
	FromAddress  string   `json:"from_address"`
}

// Bridge quote response
type BridgeQuote struct {
	ID                string         `json:"id"`
	Provider          string         `json:"provider"`
	FromChain        uint64         `json:"from_chain"`
	ToChain          uint64         `json:"to_chain"`
	FromToken        string         `json:"from_token"`
	ToToken          string         `json:"to_token"`
	FromAmount       string         `json:"from_amount"`
	ToAmount         string         `json:"to_amount"`
	ExchangeRate     string         `json:"exchange_rate"`
	GasFee           string         `json:"gas_fee"`
	ProtocolFee      string         `json:"protocol_fee"`
	TotalFee         string         `json:"total_fee"`
	EstimatedTime    string         `json:"estimated_time"`
	MinAmount       string         `json:"min_amount"`
	MaxAmount       string         `json:"max_amount"`
	Slippage        float64        `json:"slippage"`
	ValidUntil      time.Time      `json:"valid_until"`
	Route           []BridgeRoute  `json:"route"`
}

// Bridge route step
type BridgeRoute struct {
	Protocol     string `json:"protocol"`
	FromChain   uint64 `json:"from_chain"`
	ToChain     uint64 `json:"to_chain"`
	FromToken   string `json:"from_token"`
	ToToken     string `json:"to_token"`
	FromAmount  string `json:"from_amount"`
	ToAmount    string `json:"to_amount"`
}

// Bridge transaction
type BridgeTransaction struct {
	ID             string    `json:"id"`
	QuoteID        string    `json:"quote_id"`
	Provider       string    `json:"provider"`
	FromChain      uint64    `json:"from_chain"`
	ToChain        uint64    `json:"to_chain"`
	FromToken      string    `json:"from_token"`
	ToToken        string    `json:"to_token"`
	FromAmount     string    `json:"from_amount"`
	ToAmount       string    `json:"to_amount"`
	FromAddress    string    `json:"from_address"`
	ToAddress      string    `json:"to_address"`
	Status         string    `json:"status"` // pending, processing, completed, failed
	FromTxHash     string    `json:"from_tx_hash"`
	ToTxHash       string    `json:"to_tx_hash"`
	DepositAddress  string    `json:"deposit_address"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// Token support
type TokenSupport struct {
	Symbol     string   `json:"symbol"`
	Name       string   `json:"name"`
	Address    string   `json:"address,omitempty"`
	ChainID    uint64   `json:"chain_id"`
	Decimals   uint8    `json:"decimals"`
	IsNative   bool     `json:"is_native"`
	MinAmount  string   `json:"min_amount"`
	MaxAmount  string   `json:"max_amount"`
}

// ============================================================================
// SERVICE IMPLEMENTATION
// ============================================================================

// BridgeService main service
type BridgeService struct {
	mu         sync.RWMutex
	chains     map[uint64]*Chain
	protocols  map[string]*BridgeProtocol
	tokens     map[uint64]map[string]*TokenSupport
	quotes     map[string]*BridgeQuote
	transactions map[string]*BridgeTransaction
}

// NewBridgeService creates new service
func NewBridgeService() *BridgeService {
	return &BridgeService{
		chains:        make(map[uint64]*Chain),
		protocols:     make(map[string]*BridgeProtocol),
		tokens:        make(map[uint64]map[string]*TokenSupport),
		quotes:        make(map[string]*BridgeQuote),
		transactions:  make(map[string]*BridgeTransaction),
	}
}

// Initialize default chains and protocols
func (s *BridgeService) Initialize() {
	// Add chains
	chains := []*Chain{
		{ID: 1, Name: "Ethereum", Symbol: "ETH", Type: "evm", ChainID: "0x1", NativeToken: "ETH", ExplorerURL: "https://etherscan.io", Confirmations: 12},
		{ID: 56, Name: "BNB Smart Chain", Symbol: "BNB", Type: "evm", ChainID: "0x38", NativeToken: "BNB", ExplorerURL: "https://bscscan.com", Confirmations: 15},
		{ID: 137, Name: "Polygon", Symbol: "MATIC", Type: "evm", ChainID: "0x89", NativeToken: "MATIC", ExplorerURL: "https://polygonscan.com", Confirmations: 100},
		{ID: 42161, Name: "Arbitrum One", Symbol: "ETH", Type: "evm", ChainID: "0xa4b1", NativeToken: "ETH", ExplorerURL: "https://arbiscan.io", Confirmations: 15},
		{ID: 10, Name: "Optimism", Symbol: "ETH", Type: "evm", ChainID: "0xa", NativeToken: "ETH", ExplorerURL: "https://optimistic.etherscan.io", Confirmations: 15},
		{ID: 43114, Name: "Avalanche", Symbol: "AVAX", Type: "evm", ChainID: "0xa86a", NativeToken: "AVAX", ExplorerURL: "https://snowtrace.io", Confirmations: 15},
		{ID: 8453, Name: "Base", Symbol: "ETH", Type: "evm", ChainID: "0x2105", NativeToken: "ETH", ExplorerURL: "https://basescan.org", Confirmations: 15},
		{ID: 501, Name: "Solana", Symbol: "SOL", Type: "solana", ChainID: "501", NativeToken: "SOL", ExplorerURL: "https://solscan.io", Confirmations: 32},
		{ID: 728126428, Name: "TRON", Symbol: "TRX", Type: "tron", ChainID: "728126428", NativeToken: "TRX", ExplorerURL: "https://tronscan.org", Confirmations: 19},
	}

	for _, chain := range chains {
		s.chains[chain.ID] = chain
		s.tokens[chain.ID] = make(map[string]*TokenSupport)
	}

	// Add protocols
	protocols := []*BridgeProtocol{
		{ID: "stargate", Name: "Stargate", Logo: "🌉", Chains: []uint64{1, 56, 137, 42161, 10, 43114}, Tokens: []string{"USDT", "USDC", "ETH", "MATIC", "AVAX"}, FeePercent: 0.06, AvgTime: "5-15 min", IsActive: true},
		{ID: "layerzero", Name: "LayerZero", Logo: "💫", Chains: []uint64{1, 56, 137, 42161, 10, 43114, 8453}, Tokens: []string{"USDT", "USDC", "ETH", "BNB", "MATIC", "AVAX"}, FeePercent: 0.05, AvgTime: "5-20 min", IsActive: true},
		{ID: "across", Name: "Across", Logo: "➡️", Chains: []uint64{1, 42161, 10}, Tokens: []string{"USDC", "ETH", "WETH", "MATIC"}, FeePercent: 0.04, AvgTime: "1-5 min", IsActive: true},
		{ID: "hop", Name: "Hop", Logo: "🐰", Chains: []uint64{1, 42161, 10, 137}, Tokens: []string{"USDC", "USDT", "ETH", "MATIC", "ARB", "OP"}, FeePercent: 0.05, AvgTime: "5-30 min", IsActive: true},
		{ID: "celer", Name: "Celer", Logo: "🔗", Chains: []uint64{1, 56, 137, 43114}, Tokens: []string{"USDT", "USDC", "ETH", "BNB", "AVAX"}, FeePercent: 0.03, AvgTime: "10-30 min", IsActive: true},
	}

	for _, proto := range protocols {
		s.protocols[proto.ID] = proto
	}

	// Add token support per chain
	s.initializeTokenSupport()
}

func (s *BridgeService) initializeTokenSupport() {
	tokenConfigs := []struct {
		chainID uint64
		symbol  string
		name    string
		address string
		decimals uint8
		isNative bool
	}{
		{1, "ETH", "Ethereum", "", 18, true},
		{1, "USDT", "Tether USD", "0xdAC17F958D2ee523a2206206994597C13D831ec7", 6, false},
		{1, "USDC", "USD Coin", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", 6, false},
		{1, "WBTC", "Wrapped Bitcoin", "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", 8, false},
		{1, "MATIC", "Polygon", "0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0", 18, false},
		{56, "BNB", "BNB", "", 18, true},
		{56, "USDT", "Tether USD", "0x55d398326f99059fF775485246999027B3197955", 18, false},
		{56, "USDC", "USD Coin", "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", 18, false},
		{137, "MATIC", "Polygon", "", 18, true},
		{137, "USDT", "Tether USD", "0xc2132D05D31c914a87C6611C10748AEb04B58e8F", 6, false},
		{137, "USDC", "USD Coin", "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", 6, false},
		{42161, "ETH", "Ethereum", "", 18, true},
		{42161, "USDC", "USD Coin", "0xAF88d065d77adC12C02a9A5C0fB19b2a5aF17aC0", 6, false},
		{10, "ETH", "Ethereum", "", 18, true},
		{10, "USDC", "USD Coin", "0x0b2C639c533813f576Aa907AD0E6E4dAbE8fD20B", 6, false},
		{43114, "AVAX", "Avalanche", "", 18, true},
		{43114, "USDC", "USD Coin", "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E", 6, false},
		{8453, "ETH", "Ethereum", "", 18, true},
		{8453, "USDC", "USD Coin", "0x833589fCD6eDb6E08f4c7c32D4f71b54eD17f79", 6, false},
		{501, "SOL", "Solana", "", 9, true},
		{728126428, "TRX", "TRON", "", 6, true},
		{728126428, "USDT", "Tether USD", "TXla8R4itT4BiFkT4DA3R5No4w6MquJrcR", 6, false},
	}

	for _, tc := range tokenConfigs {
		s.tokens[tc.chainID][tc.symbol] = &TokenSupport{
			Symbol:    tc.symbol,
			Name:      tc.name,
			Address:   tc.address,
			ChainID:   tc.chainID,
			Decimals:  tc.decimals,
			IsNative:  tc.isNative,
			MinAmount: "10",
			MaxAmount: "1000000",
		}
	}
}

// ============================================================================
// QUOTE FUNCTIONS
// ============================================================================

// GetQuote returns bridge quote
func (s *BridgeService) GetQuote(ctx context.Context, req BridgeQuoteRequest) (*BridgeQuote, error) {
	// Validate chains
	fromChain, ok := s.chains[req.FromChain]
	if !ok {
		return nil, fmt.Errorf("unsupported source chain: %d", req.FromChain)
	}

	toChain, ok := s.chains[req.ToChain]
	if !ok {
		return nil, fmt.Errorf("unsupported destination chain: %d", req.ToChain)
	}

	// Validate tokens
	fromToken, ok := s.tokens[req.FromChain][req.FromToken]
	if !ok {
		return nil, fmt.Errorf("unsupported token on source chain: %s", req.FromToken)
	}

	toToken, ok := s.tokens[req.ToChain][req.ToToken]
	if !ok {
		return nil, fmt.Errorf("unsupported token on destination chain: %s", req.ToToken)
	}

	// Parse amount
	amount := new(big.Float)
	amount.SetString(req.Amount)
	if amount.Sign() <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}

	// Find best provider
	provider := s.findBestProvider(req.FromChain, req.ToChain, req.FromToken, req.ToToken)
	if provider == nil {
		return nil, fmt.Errorf("no bridge available for this route")
	}

	// Calculate quote (simplified - in production would call provider APIs)
	exchangeRate := s.getExchangeRate(req.FromToken, toToken.Symbol)
	toAmount := new(big.Float).Mul(amount, big.NewFloat(exchangeRate))

	gasFee := "0.001" // Estimated gas fee
	protocolFee := new(big.Float).Quo(amount, big.NewFloat(100*provider.FeePercent))
	totalFee := new(big.Float).Add(new(big.Float).SetString(gasFee), protocolFee)

	quote := &BridgeQuote{
		ID:             generateBridgeID("quote"),
		Provider:       provider.Name,
		FromChain:      req.FromChain,
		ToChain:        req.ToChain,
		FromToken:      req.FromToken,
		ToToken:        toToken.Symbol,
		FromAmount:     req.Amount,
		ToAmount:       toAmount.Text('f', toToken.Decimals),
		ExchangeRate:   fmt.Sprintf("%.8f", exchangeRate),
		GasFee:         gasFee,
		ProtocolFee:    protocolFee.Text('f', 8),
		TotalFee:       totalFee.Text('f', 8),
		EstimatedTime:  provider.AvgTime,
		MinAmount:      fromToken.MinAmount,
		MaxAmount:      fromToken.MaxAmount,
		Slippage:       0.5,
		ValidUntil:     time.Now().Add(5 * time.Minute),
		Route:          []BridgeRoute{{Protocol: provider.Name, FromChain: req.FromChain, ToChain: req.ToChain, FromToken: req.FromToken, ToToken: toToken.Symbol, FromAmount: req.Amount, ToAmount: toAmount.Text('f', toToken.Decimals)}},
	}

	s.mu.Lock()
	s.quotes[quote.ID] = quote
	s.mu.Unlock()

	return quote, nil
}

// GetQuoteByID returns quote by ID
func (s *BridgeService) GetQuoteByID(quoteID string) (*BridgeQuote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	quote, ok := s.quotes[quoteID]
	if !ok {
		return nil, fmt.Errorf("quote not found")
	}

	if time.Now().After(quote.ValidUntil) {
		return nil, fmt.Errorf("quote expired")
	}

	return quote, nil
}

// ============================================================================
// TRANSACTION FUNCTIONS
// ============================================================================

// InitiateBridge initiates a bridge transaction
func (s *BridgeService) InitiateBridge(ctx context.Context, quoteID, fromAddress, toAddress string) (*BridgeTransaction, error) {
	// Get quote
	quote, err := s.GetQuoteByID(quoteID)
	if err != nil {
		return nil, err
	}

	// Validate addresses
	if fromAddress == "" || toAddress == "" {
		return nil, fmt.Errorf("addresses required")
	}

	// Generate deposit address (in production, would call provider API)
	depositAddress := s.generateDepositAddress(quote.Provider, fromAddress)

	tx := &BridgeTransaction{
		ID:            generateBridgeID("tx"),
		QuoteID:       quoteID,
		Provider:       quote.Provider,
		FromChain:     quote.FromChain,
		ToChain:       quote.ToChain,
		FromToken:     quote.FromToken,
		ToToken:       quote.ToToken,
		FromAmount:    quote.FromAmount,
		ToAmount:      quote.ToAmount,
		FromAddress:   fromAddress,
		ToAddress:     toAddress,
		Status:         "pending",
		DepositAddress: depositAddress,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	s.mu.Lock()
	s.transactions[tx.ID] = tx
	s.mu.Unlock()

	// Simulate transaction processing
	go s.processTransaction(tx.ID)

	return tx, nil
}

// GetTransaction returns transaction by ID
func (s *BridgeService) GetTransaction(txID string) (*BridgeTransaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tx, ok := s.transactions[txID]
	if !ok {
		return nil, fmt.Errorf("transaction not found")
	}

	return tx, nil
}

// GetUserTransactions returns all transactions for a user
func (s *BridgeService) GetUserTransactions(address string) []*BridgeTransaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var txs []*BridgeTransaction
	for _, tx := range s.transactions {
		if tx.FromAddress == address || tx.ToAddress == address {
			txs = append(txs, tx)
		}
	}

	return txs
}

// processTransaction simulates transaction processing
func (s *BridgeService) processTransaction(txID string) {
	s.mu.Lock()
	tx, ok := s.transactions[txID]
	if !ok {
		s.mu.Unlock()
		return
	}

	tx.Status = "processing"
	s.mu.Unlock()

	// Simulate source chain confirmation
	time.Sleep(2 * time.Second)

	s.mu.Lock()
	tx.FromTxHash = "0x" + generateBridgeID("tx_hash")
	s.mu.Unlock()

	// Simulate destination chain
	time.Sleep(3 * time.Second)

	s.mu.Lock()
	tx.Status = "completed"
	tx.ToTxHash = "0x" + generateBridgeID("tx_hash")
	now := time.Now()
	tx.CompletedAt = &now
	tx.UpdatedAt = time.Now()
	s.mu.Unlock()
}

// CancelTransaction cancels a pending transaction
func (s *BridgeService) CancelTransaction(txID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, ok := s.transactions[txID]
	if !ok {
		return fmt.Errorf("transaction not found")
	}

	if tx.Status != "pending" && tx.Status != "processing" {
		return fmt.Errorf("cannot cancel transaction in status: %s", tx.Status)
	}

	tx.Status = "failed"
	tx.UpdatedAt = time.Now()

	return nil
}

// ============================================================================
// CHAIN AND PROTOCOL FUNCTIONS
// ============================================================================

// GetChains returns all supported chains
func (s *BridgeService) GetChains() []*Chain {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var chains []*Chain
	for _, chain := range s.chains {
		chains = append(chains, chain)
	}

	return chains
}

// GetChain returns chain by ID
func (s *BridgeService) GetChain(chainID uint64) (*Chain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chain, ok := s.chains[chainID]
	if !ok {
		return nil, fmt.Errorf("chain not found: %d", chainID)
	}

	return chain, nil
}

// GetProtocols returns all protocols
func (s *BridgeService) GetProtocols() []*BridgeProtocol {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var protocols []*BridgeProtocol
	for _, proto := range s.protocols {
		if proto.IsActive {
			protocols = append(protocols, proto)
		}
	}

	return protocols
}

// GetTokensByChain returns supported tokens for a chain
func (s *BridgeService) GetTokensByChain(chainID uint64) []*TokenSupport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tokens []*TokenSupport
	for _, token := range s.tokens[chainID] {
		tokens = append(tokens, token)
	}

	return tokens
}

// GetSupportedRoutes returns supported bridge routes
func (s *BridgeService) GetSupportedRoutes() []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var routes []map[string]interface{}
	for _, proto := range s.protocols {
		if !proto.IsActive {
			continue
		}

		for _, fromChainID := range proto.Chains {
			for _, toChainID := range proto.Chains {
				if fromChainID == toChainID {
					continue
				}

				fromChain, ok := s.chains[fromChainID]
				if !ok {
					continue
				}

				toChain, ok := s.chains[toChainID]
				if !ok {
					continue
				}

				routes = append(routes, map[string]interface{}{
					"provider":   proto.Name,
					"from_chain": fromChain.Name,
					"to_chain":   toChain.Name,
					"tokens":     proto.Tokens,
					"fee":        proto.FeePercent,
					"time":       proto.AvgTime,
				})
			}
		}
	}

	return routes
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (s *BridgeService) findBestProvider(fromChain, toChain uint64, fromToken, toToken string) *BridgeProtocol {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, proto := range s.protocols {
		if !proto.IsActive {
			continue
		}

		// Check if protocol supports both chains
		supportsFrom := false
		supportsTo := false
		for _, c := range proto.Chains {
			if c == fromChain {
				supportsFrom = true
			}
			if c == toChain {
				supportsTo = true
			}
		}

		if !supportsFrom || !supportsTo {
			continue
		}

		// Check if protocol supports the token
		supportsToken := false
		for _, t := range proto.Tokens {
			if t == fromToken || t == toToken {
				supportsToken = true
				break
			}
		}

		if supportsToken {
			return proto
		}
	}

	return nil
}

func (s *BridgeService) getExchangeRate(fromToken, toToken string) float64 {
	// Simplified rates - in production would fetch from price oracles
	rates := map[string]map[string]float64{
		"ETH":  {"ETH": 1.0, "USDT": 3500.0, "USDC": 3500.0, "MATIC": 5384.0, "AVAX": 100.0, "BNB": 5.83, "SOL": 29.17},
		"USDT": {"ETH": 0.000286, "USDT": 1.0, "USDC": 1.0, "MATIC": 1.54, "AVAX": 0.0286, "BNB": 0.00167, "SOL": 0.00833},
		"USDC": {"ETH": 0.000286, "USDT": 1.0, "USDC": 1.0, "MATIC": 1.54, "AVAX": 0.0286, "BNB": 0.00167, "SOL": 0.00833},
		"BNB":  {"ETH": 0.171, "USDT": 600.0, "USDC": 600.0, "MATIC": 923.0, "AVAX": 17.14, "BNB": 1.0, "SOL": 5.0},
		"MATIC":{"ETH": 0.000186, "USDT": 0.65, "USDC": 0.65, "MATIC": 1.0, "AVAX": 0.0186, "BNB": 0.00108, "SOL": 0.00542},
		"AVAX": {"ETH": 0.01, "USDT": 35.0, "USDC": 35.0, "MATIC": 53.85, "AVAX": 1.0, "BNB": 0.0583, "SOL": 0.291},
		"SOL":  {"ETH": 0.0343, "USDT": 120.0, "USDC": 120.0, "MATIC": 184.6, "AVAX": 3.43, "BNB": 0.2, "SOL": 1.0},
	}

	if fromRates, ok := rates[fromToken]; ok {
		if rate, ok := fromRates[toToken]; ok {
			return rate
		}
	}

	return 1.0
}

func (s *BridgeService) generateDepositAddress(provider, userAddress string) string {
	data := fmt.Sprintf("%s:%s", provider, userAddress)
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:20])
}

func generateBridgeID(prefix string) string {
	return fmt.Sprintf("%s_%d_%x", prefix, time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

// ============================================================================
// HTTP HANDLERS
// ============================================================================

func (s *BridgeService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path
	method := r.Method

	switch {
	case path == "/api/v1/chains" && method == http.MethodGet:
		s.handleGetChains(w, r)
	case path == "/api/v1/protocols" && method == http.MethodGet:
		s.handleGetProtocols(w, r)
	case path == "/api/v1/tokens" && method == http.MethodGet:
		s.handleGetTokens(w, r)
	case path == "/api/v1/routes" && method == http.MethodGet:
		s.handleGetRoutes(w, r)
	case path == "/api/v1/quote" && method == http.MethodPost:
		s.handleGetQuote(w, r)
	case strings.HasPrefix(path, "/api/v1/quote/") && method == http.MethodGet:
		s.handleGetQuoteByID(w, r)
	case path == "/api/v1/bridge" && method == http.MethodPost:
		s.handleInitiateBridge(w, r)
	case strings.HasPrefix(path, "/api/v1/transaction/") && method == http.MethodGet:
		s.handleGetTransaction(w, r)
	case path == "/api/v1/transactions" && method == http.MethodGet:
		s.handleGetUserTransactions(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *BridgeService) handleGetChains(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.GetChains())
}

func (s *BridgeService) handleGetProtocols(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.GetProtocols())
}

func (s *BridgeService) handleGetTokens(w http.ResponseWriter, r *http.Request) {
	chainIDStr := r.URL.Query().Get("chain_id")
	if chainIDStr == "" {
		http.Error(w, "chain_id required", http.StatusBadRequest)
		return
	}

	var chainID uint64
	fmt.Sscanf(chainIDStr, "%d", &chainID)

	tokens := s.GetTokensByChain(chainID)
	json.NewEncoder(w).Encode(tokens)
}

func (s *BridgeService) handleGetRoutes(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.GetSupportedRoutes())
}

func (s *BridgeService) handleGetQuote(w http.ResponseWriter, r *http.Request) {
	var req BridgeQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	quote, err := s.GetQuote(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(quote)
}

func (s *BridgeService) handleGetQuoteByID(w http.ResponseWriter, r *http.Request) {
	quoteID := strings.TrimPrefix(path, "/api/v1/quote/")
	quote, err := s.GetQuoteByID(quoteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(quote)
}

func (s *BridgeService) handleInitiateBridge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		QuoteID     string `json:"quote_id"`
		FromAddress string `json:"from_address"`
		ToAddress   string `json:"to_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := s.InitiateBridge(r.Context(), req.QuoteID, req.FromAddress, req.ToAddress)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(tx)
}

func (s *BridgeService) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	txID := strings.TrimPrefix(path, "/api/v1/transaction/")
	tx, err := s.GetTransaction(txID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(tx)
}

func (s *BridgeService) handleGetUserTransactions(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "address required", http.StatusBadRequest)
		return
	}

	txs := s.GetUserTransactions(address)
	json.NewEncoder(w).Encode(txs)
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	service := NewBridgeService()
	service.Initialize()

	fmt.Println("Starting Bridge Service on :8083")
	http.HandleFunc("/", service.ServeHTTP)

	if err := http.ListenAndServe(":8083", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
