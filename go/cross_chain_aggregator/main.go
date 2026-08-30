/**
 * TigerWallet Cross-Chain Aggregator Service
 * High-Load Distributed Go Implementation
 *
 * Features:
 * - Multi-bridge aggregation (LayerZero, Wormhole, Axelar, Stargate)
 * - Best route finding
 * - Lowest cost optimization
 * - Fastest completion time
 * - Slippage protection
 * - Real-time quote updates
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// ============== Data Structures ==============

type BridgeRoute struct {
	RouteID       string       `json:"route_id"`
	SourceChain   string       `json:"source_chain"`
	TargetChain   string       `json:"target_chain"`
	Token         string       `json:"token"`
	Amount        float64      `json:"amount"`
	SourceBridge  string       `json:"source_bridge"`
	TargetBridge  string       `json:"target_bridge"`
	EstimatedTime string       `json:"estimated_time"`
	FeeUSD        float64      `json:"fee_usd"`
	FeeToken      string       `json:"fee_token"`
	MinAmount     float64      `json:"min_amount"`
	MaxAmount     float64      `json:"max_amount"`
	Slippage      float64      `json:"slippage"`
	Confidence    float64      `json:"confidence"`
	Steps         []BridgeStep `json:"steps"`
}

type BridgeStep struct {
	Step      int    `json:"step"`
	Action    string `json:"action"` // bridge, swap, approve
	Protocol  string `json:"protocol"`
	FromToken string `json:"from_token"`
	ToToken   string `json:"to_token"`
	Estimate  string `json:"estimate"`
}

type QuoteRequest struct {
	SourceChain string  `json:"source_chain"`
	TargetChain string  `json:"target_chain"`
	Token       string  `json:"token"`
	Amount      float64 `json:"amount"`
	Slippage    float64 `json:"slippage"`
}

type QuoteResponse struct {
	RequestID string        `json:"request_id"`
	Routes    []BridgeRoute `json:"routes"`
	BestRoute *BridgeRoute  `json:"best_route"`
	UpdatedAt int64         `json:"updated_at"`
}

type SwapQuote struct {
	Protocol    string  `json:"protocol"`
	FromToken   string  `json:"from_token"`
	ToToken     string  `json:"to_token"`
	AmountIn    float64 `json:"amount_in"`
	AmountOut   float64 `json:"amount_out"`
	PriceImpact float64 `json:"price_impact"`
	FeeUSD      float64 `json:"fee_usd"`
	Slippage    float64 `json:"slippage"`
}

type TransferStatus struct {
	TransferID  string       `json:"transfer_id"`
	Status      string       `json:"status"` // pending, processing, completed, failed
	SourceTx    string       `json:"source_tx"`
	TargetTx    string       `json:"target_tx"`
	SourceChain string       `json:"source_chain"`
	TargetChain string       `json:"target_chain"`
	Progress    float64      `json:"progress"`
	Steps       []StepStatus `json:"steps"`
	UpdatedAt   int64        `json:"updated_at"`
}

type StepStatus struct {
	Step   int    `json:"step"`
	Status string `json:"status"` // pending, processing, completed, failed
	TxHash string `json:"tx_hash,omitempty"`
	Error  string `json:"error,omitempty"`
}

type ChainConfig struct {
	ChainID     int      `json:"chain_id"`
	Name        string   `json:"name"`
	Symbol      string   `json:"symbol"`
	Bridges     []string `json:"bridges"`
	NativeToken string   `json:"native_token"`
	Explorer    string   `json:"explorer"`
}

// ============== Service ==============

type CrossChainService struct {
	chains     map[string]*ChainConfig
	bridges    map[string]BridgeConfig
	routes     map[string]*BridgeRoute
	transfers  map[string]*TransferStatus
	quotes     map[string]*QuoteResponse
	priceCache map[string]float64

	mu         sync.RWMutex
	httpServer *http.Server
}

type BridgeConfig struct {
	Name        string  `json:"name"`
	MinAmount   float64 `json:"min_amount"`
	MaxAmount   float64 `json:"max_amount"`
	FeePercent  float64 `json:"fee_percent"`
	AvgTimeMin  int     `json:"avg_time_min"`
	SuccessRate float64 `json:"success_rate"`
}

func NewCrossChainService() *CrossChainService {
	s := &CrossChainService{
		chains:     make(map[string]*ChainConfig),
		bridges:    make(map[string]BridgeConfig),
		routes:     make(map[string]*BridgeRoute),
		transfers:  make(map[string]*TransferStatus),
		quotes:     make(map[string]*QuoteResponse),
		priceCache: make(map[string]float64),
	}

	s.initChains()
	s.initBridges()

	return s
}

func (s *CrossChainService) initChains() {
	s.chains = map[string]*ChainConfig{
		"ethereum":      {ChainID: 1, Name: "Ethereum", Symbol: "ETH", Bridges: []string{"layerzero", "wormhole", "axelar"}, NativeToken: "ETH", Explorer: "https://etherscan.io"},
		"polygon":       {ChainID: 137, Name: "Polygon", Symbol: "MATIC", Bridges: []string{"layerzero", "wormhole"}, NativeToken: "MATIC", Explorer: "https://polygonscan.com"},
		"avalanche":     {ChainID: 43114, Name: "Avalanche", Symbol: "AVAX", Bridges: []string{"layerzero", "wormhole", "stargate"}, NativeToken: "AVAX", Explorer: "https://snowtrace.io"},
		"arbitrum":      {ChainID: 42161, Name: "Arbitrum", Symbol: "ETH", Bridges: []string{"layerzero", "wormhole"}, NativeToken: "ETH", Explorer: "https://arbiscan.io"},
		"optimism":      {ChainID: 10, Name: "Optimism", Symbol: "ETH", Bridges: []string{"layerzero", "wormhole"}, NativeToken: "ETH", Explorer: "https://optimistic.etherscan.io"},
		"bsc":           {ChainID: 56, Name: "BNB Chain", Symbol: "BNB", Bridges: []string{"layerzero", "wormhole"}, NativeToken: "BNB", Explorer: "https://bscscan.com"},
		"solana":        {ChainID: 101, Name: "Solana", Symbol: "SOL", Bridges: []string{"wormhole", "bridge"}, NativeToken: "SOL", Explorer: "https://solscan.io"},
		"base":          {ChainID: 8453, Name: "Base", Symbol: "ETH", Bridges: []string{"layerzero"}, NativeToken: "ETH", Explorer: "https://basescan.org"},
		"fantom":        {ChainID: 250, Name: "Fantom", Symbol: "FTM", Bridges: []string{"layerzero"}, NativeToken: "FTM", Explorer: "https://ftmscan.com"},
		"arbitrum_nova": {ChainID: 42170, Name: "Arbitrum Nova", Symbol: "ETH", Bridges: []string{"layerzero"}, NativeToken: "ETH", Explorer: "https://nova.arbiscan.io"},
	}
}

func (s *CrossChainService) initBridges() {
	s.bridges = map[string]BridgeConfig{
		"layerzero": {Name: "LayerZero", MinAmount: 1, MaxAmount: 10000000, FeePercent: 0.1, AvgTimeMin: 15, SuccessRate: 0.99},
		"wormhole":  {Name: "Wormhole", MinAmount: 5, MaxAmount: 5000000, FeePercent: 0.15, AvgTimeMin: 20, SuccessRate: 0.98},
		"axelar":    {Name: "Axelar", MinAmount: 10, MaxAmount: 5000000, FeePercent: 0.12, AvgTimeMin: 25, SuccessRate: 0.97},
		"stargate":  {Name: "Stargate", MinAmount: 1, MaxAmount: 1000000, FeePercent: 0.08, AvgTimeMin: 10, SuccessRate: 0.99},
		"bridge":    {Name: "Native Bridge", MinAmount: 10, MaxAmount: 1000000, FeePercent: 0.05, AvgTimeMin: 30, SuccessRate: 0.95},
	}
}

func (s *CrossChainService) Run() error {
	// Start price updates
	go s.updatePrices()

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/api/quote", s.handleQuote)
	mux.HandleFunc("/api/routes", s.handleRoutes)
	mux.HandleFunc("/api/transfer", s.handleTransfer)
	mux.HandleFunc("/api/transfer/status", s.handleTransferStatus)
	mux.HandleFunc("/api/chains", s.handleChains)
	mux.HandleFunc("/api/bridges", s.handleBridges)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         ":8086",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("Cross-chain service starting on :8086")
	return s.httpServer.ListenAndServe()
}

// ============== Handlers ==============

func (s *CrossChainService) handleQuote(w http.ResponseWriter, r *http.Request) {
	var req QuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get price (fail-closed: a quote without a real price is meaningless)
	price, ok := s.getTokenPrice(req.Token)
	if !ok {
		// Attempt one live refresh before failing.
		if fetched, err := fetchLivePricesUSD([]string{req.Token}); err == nil {
			if p, ok2 := fetched[strings.ToUpper(req.Token)]; ok2 {
				s.mu.Lock()
				s.priceCache[strings.ToUpper(req.Token)] = p
				s.mu.Unlock()
				price, ok = p, true
			}
		}
	}
	if !ok {
		http.Error(w, "price unavailable for token "+req.Token, http.StatusServiceUnavailable)
		return
	}
	amountUSD := req.Amount * price

	// Find available bridges
	sourceChain, ok := s.chains[req.SourceChain]
	if !ok {
		http.Error(w, "Source chain not supported", http.StatusBadRequest)
		return
	}

	targetChain, ok := s.chains[req.TargetChain]
	if !ok {
		http.Error(w, "Target chain not supported", http.StatusBadRequest)
		return
	}

	// Find common bridges
	var commonBridges []string
	for _, bridge := range sourceChain.Bridges {
		for _, tb := range targetChain.Bridges {
			if bridge == tb {
				commonBridges = append(commonBridges, bridge)
			}
		}
	}

	// Generate routes
	var routes []BridgeRoute
	for _, bridge := range commonBridges {
		route := s.calculateRoute(req, bridge, amountUSD)
		routes = append(routes, route)
	}

	// Sort by total fee
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].FeeUSD < routes[j].FeeUSD
	})

	response := QuoteResponse{
		RequestID: generateRequestID(),
		Routes:    routes,
		UpdatedAt: time.Now().Unix(),
	}

	if len(routes) > 0 {
		response.BestRoute = &routes[0]
	}

	s.quotes[response.RequestID] = &response

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *CrossChainService) handleRoutes(w http.ResponseWriter, r *http.Request) {
	sourceChain := r.URL.Query().Get("source")
	targetChain := r.URL.Query().Get("target")
	token := r.URL.Query().Get("token")

	var routes []BridgeRoute
	for _, route := range s.routes {
		if sourceChain != "" && route.SourceChain != sourceChain {
			continue
		}
		if targetChain != "" && route.TargetChain != targetChain {
			continue
		}
		if token != "" && route.Token != token {
			continue
		}
		routes = append(routes, *route)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}

func (s *CrossChainService) handleTransfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RouteID   string  `json:"route_id"`
		Amount    float64 `json:"amount"`
		ToAddress string  `json:"to_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	transferID := generateRequestID()

	// Get route
	s.mu.RLock()
	route, ok := s.routes[req.RouteID]
	s.mu.RUnlock()

	if !ok {
		// Create new transfer
		route = &BridgeRoute{
			RouteID:     req.RouteID,
			Amount:      req.Amount,
			SourceChain: req.RouteID,
			TargetChain: req.RouteID,
		}
	}

	transfer := &TransferStatus{
		TransferID:  transferID,
		Status:      "pending",
		SourceChain: route.SourceChain,
		TargetChain: route.TargetChain,
		Progress:    0,
		Steps: []StepStatus{
			{Step: 1, Status: "pending"},
			{Step: 2, Status: "pending"},
			{Step: 3, Status: "pending"},
		},
		UpdatedAt: time.Now().Unix(),
	}

	s.transfers[transferID] = transfer

	// Process transfer asynchronously
	go s.processTransfer(transferID, route, req.Amount, req.ToAddress)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transfer)
}

func (s *CrossChainService) handleTransferStatus(w http.ResponseWriter, r *http.Request) {
	transferID := r.URL.Query().Get("transfer_id")

	transfer, ok := s.transfers[transferID]
	if !ok {
		http.Error(w, "Transfer not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transfer)
}

func (s *CrossChainService) handleChains(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.chains)
}

func (s *CrossChainService) handleBridges(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.bridges)
}

func (s *CrossChainService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"chains":    len(s.chains),
		"bridges":   len(s.bridges),
		"transfers": len(s.transfers),
		"timestamp": time.Now().Unix(),
	})
}

// ============== Processing ==============

func (s *CrossChainService) calculateRoute(req QuoteRequest, bridgeName string, amountUSD float64) BridgeRoute {
	bridge, ok := s.bridges[bridgeName]
	if !ok {
		return BridgeRoute{}
	}

	// Calculate fees
	feeUSD := amountUSD * bridge.FeePercent / 100

	// Calculate steps
	steps := []BridgeStep{
		{Step: 1, Action: "approve", Protocol: bridgeName, FromToken: req.Token, ToToken: req.Token},
		{Step: 2, Action: "bridge", Protocol: bridgeName, FromToken: req.Token, ToToken: req.Token, Estimate: fmt.Sprintf("%d min", bridge.AvgTimeMin)},
		{Step: 3, Action: "claim", Protocol: bridgeName, FromToken: req.Token, ToToken: req.Token},
	}

	route := BridgeRoute{
		RouteID:       fmt.Sprintf("%s_%s_%s", req.SourceChain, req.TargetChain, bridgeName),
		SourceChain:   req.SourceChain,
		TargetChain:   req.TargetChain,
		Token:         req.Token,
		Amount:        req.Amount,
		SourceBridge:  bridgeName,
		EstimatedTime: fmt.Sprintf("%d min", bridge.AvgTimeMin),
		FeeUSD:        feeUSD,
		FeeToken:      req.Token,
		MinAmount:     bridge.MinAmount,
		MaxAmount:     bridge.MaxAmount,
		Slippage:      req.Slippage,
		Confidence:    bridge.SuccessRate,
		Steps:         steps,
	}

	return route
}

// processTransfer executes a REAL cross-chain transfer through the LI.FI
// aggregation API: a real quote yields an unsigned bridge transaction which
// the configured executor key signs and broadcasts on the source chain; the
// destination leg is tracked via the real LI.FI status API. Fail-closed:
// without BRIDGE_EXECUTOR_PRIVATE_KEY the transfer fails immediately, and a
// tx hash is recorded only after a real broadcast/confirmation.
func (s *CrossChainService) processTransfer(transferID string, route *BridgeRoute, amount float64, toAddress string) {
	fail := func(step int, err error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		transfer := s.transfers[transferID]
		if transfer == nil {
			return
		}
		if step >= 1 && step <= len(transfer.Steps) {
			transfer.Steps[step-1].Status = "failed"
			transfer.Steps[step-1].Error = err.Error()
		}
		transfer.Status = "failed"
		transfer.UpdatedAt = time.Now().Unix()
	}
	progress := func(step int, status string, txHash string, pct float64) {
		s.mu.Lock()
		defer s.mu.Unlock()
		transfer := s.transfers[transferID]
		if transfer == nil {
			return
		}
		if step >= 1 && step <= len(transfer.Steps) {
			transfer.Steps[step-1].Status = status
			if txHash != "" {
				transfer.Steps[step-1].TxHash = txHash
			}
		}
		transfer.Progress = pct
		transfer.UpdatedAt = time.Now().Unix()
	}

	execKey := os.Getenv("BRIDGE_EXECUTOR_PRIVATE_KEY")
	if execKey == "" {
		fail(1, fmt.Errorf("BRIDGE_EXECUTOR_PRIVATE_KEY not configured; bridge execution disabled"))
		return
	}
	source, ok := s.chains[route.SourceChain]
	if !ok {
		fail(1, fmt.Errorf("source chain %q not supported", route.SourceChain))
		return
	}
	target, ok := s.chains[route.TargetChain]
	if !ok {
		fail(1, fmt.Errorf("target chain %q not supported", route.TargetChain))
		return
	}
	rpcURL, err := rpcForChain(route.SourceChain)
	if err != nil {
		fail(1, err)
		return
	}
	tokenAddr, decimals, err := resolveTokenAddress(route.SourceChain, route.Token, int64(source.ChainID))
	if err != nil {
		fail(1, err)
		return
	}
	targetTokenAddr, _, err := resolveTokenAddress(route.TargetChain, route.Token, int64(target.ChainID))
	if err != nil {
		fail(1, err)
		return
	}

	// amount (token units, float) -> integer smallest-unit amount
	multiplier := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	amountF := new(big.Float).Mul(new(big.Float).SetFloat64(amount), multiplier)
	amountWei, _ := amountF.Int(nil)
	if amountWei.Sign() <= 0 {
		fail(1, fmt.Errorf("amount too small after decimal conversion"))
		return
	}

	execPriv, err := crypto.HexToECDSA(strings.TrimPrefix(execKey, "0x"))
	if err != nil {
		fail(1, fmt.Errorf("invalid executor key: %w", err))
		return
	}
	fromAddress := crypto.PubkeyToAddress(execPriv.PublicKey).Hex()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Step 1: real quote from LI.FI (returns the bridge tx to execute).
	progress(1, "processing", "", 10)
	quote, err := fetchLiFiQuote(ctx, int64(source.ChainID), int64(target.ChainID),
		tokenAddr, targetTokenAddr, fromAddress, toAddress, amountWei, route.Slippage)
	if err != nil {
		fail(1, fmt.Errorf("bridge quote: %w", err))
		return
	}
	progress(1, "completed", "", 33)

	// Step 2: sign + broadcast the real bridge transaction on the source chain.
	progress(2, "processing", "", 40)
	sourceTx, err := signAndBroadcastTx(ctx, rpcURL, execKey, quote.TxTo, quote.TxValue, quote.TxData)
	if err != nil {
		fail(2, fmt.Errorf("source tx: %w", err))
		return
	}
	s.mu.Lock()
	if transfer := s.transfers[transferID]; transfer != nil {
		transfer.Status = "processing"
		transfer.SourceTx = sourceTx
		transfer.Steps[1].Status = "completed"
		transfer.Steps[1].TxHash = sourceTx
		transfer.Progress = 66
		transfer.UpdatedAt = time.Now().Unix()
	}
	s.mu.Unlock()

	// Step 3: track the destination leg via the real LI.FI status API until
	// completion (relayer executes the claim; no manual claim tx for the
	// aggregated routes LI.FI selects by default).
	progress(3, "processing", "", 75)
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		status, receivingTx, err := lifiBridgeStatus(ctx, sourceTx, int64(source.ChainID), int64(target.ChainID))
		if err == nil {
			switch status {
			case "DONE":
				s.mu.Lock()
				if transfer := s.transfers[transferID]; transfer != nil {
					transfer.Steps[2].Status = "completed"
					if receivingTx != "" {
						transfer.Steps[2].TxHash = receivingTx
						transfer.TargetTx = receivingTx
					}
					transfer.Status = "completed"
					transfer.Progress = 100
					transfer.UpdatedAt = time.Now().Unix()
				}
				s.mu.Unlock()
				return
			case "FAILED":
				fail(3, fmt.Errorf("bridge relay failed for source tx %s", sourceTx))
				return
			}
		}
		select {
		case <-ctx.Done():
			fail(3, fmt.Errorf("context cancelled while awaiting destination confirmation"))
			return
		case <-time.After(10 * time.Second):
		}
	}
	fail(3, fmt.Errorf("destination confirmation timed out after 30m for source tx %s", sourceTx))
}

// updatePrices refreshes the price cache from the real CoinGecko oracle
// every 30s. On upstream failure the last known real prices are kept —
// prices are never fabricated or randomly perturbed.
func (s *CrossChainService) updatePrices() {
	symbols := []string{"USDT", "USDC"}
	seen := map[string]bool{"USDT": true, "USDC": true}
	for _, chain := range s.chains {
		if !seen[chain.NativeToken] {
			seen[chain.NativeToken] = true
			symbols = append(symbols, chain.NativeToken)
		}
	}

	refresh := func() {
		fetched, err := fetchLivePricesUSD(symbols)
		if err != nil || len(fetched) == 0 {
			return // keep last known real prices
		}
		s.mu.Lock()
		for token, price := range fetched {
			s.priceCache[token] = price
		}
		s.mu.Unlock()
	}

	refresh()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		refresh()
	}
}

// getTokenPrice returns the last known real price; (0, false) when unknown.
// Callers must fail closed rather than assume a price.
func (s *CrossChainService) getTokenPrice(token string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if price, ok := s.priceCache[token]; ok && price > 0 {
		return price, true
	}
	return 0, false
}

func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// ============== Main ==============

func main() {
	log.Println("Starting TigerWallet Cross-Chain Aggregator Service...")

	service := NewCrossChainService()
	if err := service.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func (s *CrossChainService) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
