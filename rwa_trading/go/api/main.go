/**
 * TigerWallet RWA Trading - Real World Assets Trading Module
 * Trade tokenized stocks, ETFs, commodities, and real estate
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort string
}

var cfg = Config{ServerPort: ":8445"}

// ============================================================================
// Data Models
// ============================================================================

type RWAAsset struct {
	AssetID          string    `json:"asset_id"`
	Name             string    `json:"name"`
	Symbol           string    `json:"symbol"`
	AssetType        string    `json:"asset_type"` // stock, etf, commodity, real_estate, bond
	Issuer           string    `json:"issuer"`
	TotalSupply      uint64    `json:"total_supply"`
	CirculatingSupply uint64  `json:"circulating_supply"`
	Price            uint64    `json:"price"`       // In cents
	Change24h        int64     `json:"change_24h"`  // In basis points
	Volume24h        uint64    `json:"volume_24h"`
	MarketCap        uint64    `json:"market_cap"`
	UnderlyingPrice  uint64    `json:"underlying_price"` // Real world price
	Blockchain       string    `json:"blockchain"`
	ContractAddress  string    `json:"contract_address"`
	IsActive         bool      `json:"is_active"`
	ImageURL         string    `json:"image_url"`
	CreatedAt        uint64    `json:"created_at"`
}

type RWAPosition struct {
	UserID       uint64    `json:"user_id"`
	AssetID      string    `json:"asset_id"`
	Quantity     uint64    `json:"quantity"`
	AvgPrice     uint64    `json:"avg_price"`
	CurrentValue uint64    `json:"current_value"`
	ProfitLoss   int64     `json:"profit_loss"`
	CreatedAt    uint64    `json:"created_at"`
	UpdatedAt    uint64    `json:"updated_at"`
}

type RWATransaction struct {
	TxID         uint64    `json:"tx_id"`
	UserID       uint64    `json:"user_id"`
	AssetID      string    `json:"asset_id"`
	Type         string    `json:"type"` // buy, sell, transfer
	Status       string    `json:"status"`
	Quantity     uint64    `json:"quantity"`
	Price        uint64    `json:"price"`
	TotalValue   uint64    `json:"total_value"`
	Fees         uint64    `json:"fees"`
	Counterparty *string   `json:"counterparty,omitempty"`
	Timestamp    uint64    `json:"timestamp"`
	CompletedAt  uint64    `json:"completed_at,omitempty"`
}

type RWAMarket struct {
	Asset           RWAAsset   `json:"asset"`
	Bid             uint64     `json:"bid"`
	Ask             uint64     `json:"ask"`
	Spread          uint64     `json:"spread"`
	Depth           uint64     `json:"depth"`
	LastTradePrice  uint64     `json:"last_trade_price"`
	LastTradeTime   uint64     `json:"last_trade_time"`
}

type RWAService struct {
	assets      map[string]*RWAAsset
	positions   map[string]*RWAPosition // userID_assetID -> position
	transactions map[uint64]*RWATransaction
	nextTxID    uint64
	mu          sync.RWMutex
	feeBPS      uint32 // Basis points
}

func NewRWAService() *RWAService {
	svc := &RWAService{
		assets:       make(map[string]*RWAAsset),
		positions:    make(map[string]*RWAPosition),
		transactions: make(map[uint64]*RWATransaction),
		nextTxID:     1,
		feeBPS:       10, // 0.1%
	}

	svc.seedAssets()
	return svc
}

func (s *RWAService) seedAssets() {
	now := uint64(time.Now().UnixMilli())

	// Stocks
	stocks := []*RWAAsset{
		{AssetID: "rwa_aapl", Name: "Apple Inc.", Symbol: "AAPL", AssetType: "stock", Issuer: "Tiger", TotalSupply: 1000000, Price: 17850, Change24h: 150, Volume24h: 5000000, MarketCap: 2780000000000, UnderlyingPrice: 17850, Blockchain: "Ethereum", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
		{AssetID: "rwa_googl", Name: "Alphabet Inc.", Symbol: "GOOGL", AssetType: "stock", Issuer: "Tiger", TotalSupply: 500000, Price: 14100, Change24h: -80, Volume24h: 3000000, MarketCap: 1780000000000, UnderlyingPrice: 14100, Blockchain: "Ethereum", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
		{AssetID: "rwa_msft", Name: "Microsoft Corp.", Symbol: "MSFT", AssetType: "stock", Issuer: "Tiger", TotalSupply: 800000, Price: 37800, Change24h: 250, Volume24h: 8000000, MarketCap: 2820000000000, UnderlyingPrice: 37800, Blockchain: "Polygon", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
		{AssetID: "rwa_tsla", Name: "Tesla Inc.", Symbol: "TSLA", AssetType: "stock", Issuer: "Tiger", TotalSupply: 600000, Price: 24800, Change24h: -320, Volume24h: 12000000, MarketCap: 785000000000, UnderlyingPrice: 24800, Blockchain: "Arbitrum", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
		{AssetID: "rwa_amzn", Name: "Amazon.com Inc.", Symbol: "AMZN", AssetType: "stock", Issuer: "Tiger", TotalSupply: 700000, Price: 17800, Change24h: 180, Volume24h: 6000000, MarketCap: 1850000000000, UnderlyingPrice: 17800, Blockchain: "Ethereum", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
		{AssetID: "rwa_nvda", Name: "NVIDIA Corp.", Symbol: "NVDA", AssetType: "stock", Issuer: "Tiger", TotalSupply: 400000, Price: 87500, Change24h: 1200, Volume24h: 25000000, MarketCap: 2150000000000, UnderlyingPrice: 87500, Blockchain: "Solana", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
		{AssetID: "rwa_meta", Name: "Meta Platforms", Symbol: "META", AssetType: "stock", Issuer: "Tiger", TotalSupply: 450000, Price: 48500, Change24h: -200, Volume24h: 9000000, MarketCap: 1240000000000, UnderlyingPrice: 48500, Blockchain: "Ethereum", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
		{AssetID: "rwa_jpm", Name: "JPMorgan Chase", Symbol: "JPM", AssetType: "stock", Issuer: "Tiger", TotalSupply: 550000, Price: 19500, Change24h: 90, Volume24h: 3500000, MarketCap: 565000000000, UnderlyingPrice: 19500, Blockchain: "Polygon", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
	}

	// ETFs
	etfs := []*RWAAsset{
		{AssetID: "rwa_spy", Name: "SPDR S&P 500 ETF", Symbol: "SPY", AssetType: "etf", Issuer: "Tiger", TotalSupply: 2000000, Price: 47800, Change24h: 80, Volume24h: 15000000, MarketCap: 450000000000, UnderlyingPrice: 47800, Blockchain: "Ethereum", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
		{AssetID: "rwa_qqq", Name: "Invesco QQQ Trust", Symbol: "QQQ", AssetType: "etf", Issuer: "Tiger", TotalSupply: 1500000, Price: 41200, Change24h: 120, Volume24h: 10000000, MarketCap: 185000000000, UnderlyingPrice: 41200, Blockchain: "Ethereum", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
		{AssetID: "rwa_ibt", Name: "iShares Bitcoin Trust", Symbol: "IBIT", AssetType: "etf", Issuer: "Tiger", TotalSupply: 3000000, Price: 4200, Change24h: 250, Volume24h: 20000000, MarketCap: 52000000000, UnderlyingPrice: 4200, Blockchain: "Bitcoin", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
	}

	// Commodities
	commodities := []*RWAAsset{
		{AssetID: "rwa_paxg", Name: "Paxos Gold", Symbol: "PAXG", AssetType: "commodity", Issuer: "Paxos", TotalSupply: 100000, Price: 234500, Change24h: 50, Volume24h: 1500000, MarketCap: 23450000000, UnderlyingPrice: 234500, Blockchain: "Ethereum", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
		{AssetID: "rwa_xau", Name: "Gold Tokenized", Symbol: "XAU", AssetType: "commodity", Issuer: "Tiger", TotalSupply: 50000, Price: 205000, Change24h: 30, Volume24h: 800000, MarketCap: 10250000000, UnderlyingPrice: 205000, Blockchain: "Solana", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
		{AssetID: "rwa_xag", Name: "Silver Tokenized", Symbol: "XAG", AssetType: "commodity", Issuer: "Tiger", TotalSupply: 200000, Price: 2850, Change24h: -20, Volume24h: 500000, MarketCap: 570000000, UnderlyingPrice: 2850, Blockchain: "Polygon", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
	}

	// Bonds
	bonds := []*RWAAsset{
		{AssetID: "rwa_tlt", Name: "20+ Year Treasury Bond", Symbol: "TLT", AssetType: "bond", Issuer: "Tiger", TotalSupply: 1000000, Price: 9500, Change24h: 20, Volume24h: 2000000, MarketCap: 9500000000, UnderlyingPrice: 9500, Blockchain: "Ethereum", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
		{AssetID: "rwa_ighg", Name: "Investment Grade Corp Bond", Symbol: "IGHG", AssetType: "bond", Issuer: "Tiger", TotalSupply: 800000, Price: 10200, Change24h: 10, Volume24h: 1500000, MarketCap: 8160000000, UnderlyingPrice: 10200, Blockchain: "Polygon", ContractAddress: "0x...", IsActive: true, CreatedAt: now},
	}

	for _, a := range stocks { s.assets[a.AssetID] = a }
	for _, a := range etfs { s.assets[a.AssetID] = a }
	for _, a := range commodities { s.assets[a.AssetID] = a }
	for _, a := range bonds { s.assets[a.AssetID] = a }

	log.Printf("Seeded %d RWA assets", len(s.assets))
}

func (s *RWAService) GetAssets(assetType, search string, limit int) []*RWAAsset {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*RWAAsset
	for _, a := range s.assets {
		if !a.IsActive { continue }
		if assetType != "" && a.AssetType != assetType { continue }
		if search != "" && !containsCI(a.Name, search) && !containsCI(a.Symbol, search) { continue }
		result = append(result, a)
		if limit > 0 && len(result) >= limit { break }
	}
	return result
}

func (s *RWAService) GetAsset(assetID string) *RWAAsset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.assets[assetID]
}

func (s *RWAService) GetMarket(assetID string) *RWAMarket {
	s.mu.RLock()
	asset := s.assets[assetID]
	s.mu.RUnlock()

	if asset == nil { return nil }

	spread := asset.Price / 100 // 1% spread
	return &RWAMarket{
		Asset:          *asset,
		Bid:            asset.Price - spread,
		Ask:            asset.Price + spread,
		Spread:         spread,
		Depth:          asset.Volume24h / 100,
		LastTradePrice: asset.Price,
		LastTradeTime:  asset.UpdatedAt,
	}
}

func (s *RWAService) Buy(userID uint64, assetID string, quantity uint64) (*RWATransaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	asset, ok := s.assets[assetID]
	if !ok || !asset.IsActive {
		return nil, fmt.Errorf("asset not found")
	}

	if quantity == 0 {
		return nil, fmt.Errorf("invalid quantity")
	}

	total := asset.Price * quantity
	fees := (total * int64(s.feeBPS)) / 10000

	now := uint64(time.Now().UnixMilli())
	tx := &RWATransaction{
		TxID:        s.nextTxID,
		UserID:       userID,
		AssetID:     assetID,
		Type:        "buy",
		Status:      "completed",
		Quantity:    quantity,
		Price:       asset.Price,
		TotalValue:  total,
		Fees:        uint64(fees),
		Timestamp:   now,
		CompletedAt: now,
	}

	s.transactions[s.nextTxID] = tx
	s.nextTxID++

	// Update position
	posKey := fmt.Sprintf("%d_%s", userID, assetID)
	if pos, exists := s.positions[posKey]; exists {
		newQuantity := pos.Quantity + quantity
		newAvgPrice := (pos.AvgPrice*pos.Quantity + asset.Price*quantity) / newQuantity
		pos.Quantity = newQuantity
		pos.AvgPrice = newAvgPrice
		pos.CurrentValue = newQuantity * asset.Price
		pos.ProfitLoss = int64(pos.CurrentValue) - int64(pos.AvgPrice*newQuantity)
		pos.UpdatedAt = now
	} else {
		s.positions[posKey] = &RWAPosition{
			UserID:       userID,
			AssetID:      assetID,
			Quantity:     quantity,
			AvgPrice:     asset.Price,
			CurrentValue: total,
			ProfitLoss:   0,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}

	return tx, nil
}

func (s *RWAService) Sell(userID uint64, assetID string, quantity uint64) (*RWATransaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	asset, ok := s.assets[assetID]
	if !ok || !asset.IsActive {
		return nil, fmt.Errorf("asset not found")
	}

	posKey := fmt.Sprintf("%d_%s", userID, assetID)
	pos, exists := s.positions[posKey]
	if !exists || pos.Quantity < quantity {
		return nil, fmt.Errorf("insufficient holdings")
	}

	total := asset.Price * quantity
	fees := (total * int64(s.feeBPS)) / 10000

	now := uint64(time.Now().UnixMilli())
	tx := &RWATransaction{
		TxID:        s.nextTxID,
		UserID:       userID,
		AssetID:     assetID,
		Type:        "sell",
		Status:      "completed",
		Quantity:    quantity,
		Price:       asset.Price,
		TotalValue:  total,
		Fees:        uint64(fees),
		Timestamp:   now,
		CompletedAt: now,
	}

	s.transactions[s.nextTxID] = tx
	s.nextTxID++

	// Update position
	pos.Quantity -= quantity
	if pos.Quantity == 0 {
		delete(s.positions, posKey)
	} else {
		pos.CurrentValue = pos.Quantity * asset.Price
		pos.ProfitLoss = int64(pos.CurrentValue) - int64(pos.AvgPrice*pos.Quantity)
		pos.UpdatedAt = now
	}

	return tx, nil
}

func (s *RWAService) GetUserPositions(userID uint64) []*RWAPosition {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*RWAPosition
	for _, pos := range s.positions {
		if pos.UserID == userID {
			result = append(result, pos)
		}
	}
	return result
}

func (s *RWAService) GetUserTransactions(userID uint64) []*RWATransaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*RWATransaction
	for _, tx := range s.transactions {
		if tx.UserID == userID {
			result = append(result, tx)
		}
	}

	// Sort by timestamp descending
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Timestamp < result[j].Timestamp {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// ============================================================================
// HTTP Handlers
// ============================================================================

type Handler struct {
	service *RWAService
}

func NewHandler(svc *RWAService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) GetAssets(w http.ResponseWriter, r *http.Request) {
	assetType := r.URL.Query().Get("type")
	search := r.URL.Query().Get("search")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	assets := h.service.GetAssets(assetType, search, limit)

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": assets})
}

func (h *Handler) GetAsset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	asset := h.service.GetAsset(vars["id"])

	if asset == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": map[string]string{"code": "NOT_FOUND", "message": "Asset not found"}})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": asset})
}

func (h *Handler) GetMarket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	market := h.service.GetMarket(vars["id"])

	if market == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": map[string]string{"code": "NOT_FOUND", "message": "Asset not found"}})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": market})
}

func (h *Handler) BuyAsset(w http.ResponseWriter, r *http.Request) {
	userID := uint64(1)
	if uid := r.Header.Get("X-User-ID"); uid != "" {
		fmt.Sscanf(uid, "%d", &userID)
	}

	var req struct {
		AssetID  string `json:"asset_id"`
		Quantity uint64 `json:"quantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": map[string]string{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	tx, err := h.service.Buy(userID, req.AssetID, req.Quantity)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": map[string]string{"code": "BUY_FAILED", "message": err.Error()}})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": tx})
}

func (h *Handler) SellAsset(w http.ResponseWriter, r *http.Request) {
	userID := uint64(1)
	if uid := r.Header.Get("X-User-ID"); uid != "" {
		fmt.Sscanf(uid, "%d", &userID)
	}

	var req struct {
		AssetID  string `json:"asset_id"`
		Quantity uint64 `json:"quantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": map[string]string{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	tx, err := h.service.Sell(userID, req.AssetID, req.Quantity)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": map[string]string{"code": "SELL_FAILED", "message": err.Error()}})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": tx})
}

func (h *Handler) GetPositions(w http.ResponseWriter, r *http.Request) {
	userID := uint64(1)
	if uid := r.Header.Get("X-User-ID"); uid != "" {
		fmt.Sscanf(uid, "%d", &userID)
	}

	positions := h.service.GetUserPositions(userID)

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": positions})
}

func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	userID := uint64(1)
	if uid := r.Header.Get("X-User-ID"); uid != "" {
		fmt.Sscanf(uid, "%d", &userID)
	}

	txs := h.service.GetUserTransactions(userID)

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": txs})
}

// ============================================================================
// Helpers
// ============================================================================

func containsCI(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

import "strings"

// ============================================================================
// Main
// ============================================================================

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting TigerWallet RWA Trading API...")

	service := NewRWAService()
	handler := NewHandler(service)

	router := mux.NewRouter()
	router.Use(handlers.ContentTypeHandler(handlers.LoggingHandler(os.Stdout, router), "application/json"))

	router.HandleFunc("/api/v1/assets", handler.GetAssets).Methods("GET")
	router.HandleFunc("/api/v1/assets/{id}", handler.GetAsset).Methods("GET")
	router.HandleFunc("/api/v1/assets/{id}/market", handler.GetMarket).Methods("GET")
	router.HandleFunc("/api/v1/buy", handler.BuyAsset).Methods("POST")
	router.HandleFunc("/api/v1/sell", handler.SellAsset).Methods("POST")
	router.HandleFunc("/api/v1/positions", handler.GetPositions).Methods("GET")
	router.HandleFunc("/api/v1/transactions", handler.GetTransactions).Methods("GET")

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")

	srv := &http.Server{Addr: cfg.ServerPort, Handler: router}

	go func() {
		log.Printf("Server listening on %s", cfg.ServerPort)
		srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
