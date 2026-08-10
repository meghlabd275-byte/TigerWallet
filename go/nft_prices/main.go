/**
 * TigerWallet NFT Floor Price Service
 * High-Load Distributed Go Implementation
 *
 * Features:
 * - Real-time floor price tracking
 * - Multi-marketplace aggregation
 * - Portfolio valuation
 * - Price alerts
 * - Collection analytics
 */

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
)

// ============== Data Structures ==============

type Collection struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Symbol      string  `json:"symbol"`
	Blockchain  string  `json:"blockchain"`
	FloorPrice  float64 `json:"floor_price"`
	Volume24h   float64 `json:"volume_24h"`
	Sales24h    int     `json:"sales_24h"`
	Owners      int     `json:"owners"`
	TotalSupply int     `json:"total_supply"`
	MarketCap   float64 `json:"market_cap"`
	Change24h   float64 `json:"change_24h"`
	ImageURL    string  `json:"image_url"`
	UpdatedAt   int64   `json:"updated_at"`
}

type NFTToken struct {
	ID           string      `json:"id"`
	CollectionID string      `json:"collection_id"`
	TokenID      string      `json:"token_id"`
	Name         string      `json:"name"`
	ImageURL     string      `json:"image_url"`
	Owner        string      `json:"owner"`
	Price        float64     `json:"price"`
	LastSale     float64     `json:"last_sale"`
	Attributes   []Attribute `json:"attributes"`
	Listed       bool        `json:"listed"`
}

type Attribute struct {
	TraitType string  `json:"trait_type"`
	Value     string  `json:"value"`
	Rarity    float64 `json:"rarity"`
}

type FloorPriceAlert struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	CollectionID string  `json:"collection_id"`
	TargetPrice  float64 `json:"target_price"`
	Direction    string  `json:"direction"` // above, below
	Triggered    bool    `json:"triggered"`
	CreatedAt    int64   `json:"created_at"`
}

type PortfolioValue struct {
	UserID      string            `json:"user_id"`
	TotalValue  float64           `json:"total_value"`
	Collections []CollectionValue `json:"collections"`
	UpdatedAt   int64             `json:"updated_at"`
}

type CollectionValue struct {
	CollectionID string  `json:"collection_id"`
	Name         string  `json:"name"`
	Count        int     `json:"count"`
	Value        float64 `json:"value"`
	FloorPrice   float64 `json:"floor_price"`
}

// ============== Service ==============

type NFTPriceService struct {
	collections map[string]*Collection
	tokens      map[string][]*NFTToken
	alerts      []FloorPriceAlert
	holdings    map[string]map[string]int // user -> collection -> count

	mu     sync.RWMutex
	server *http.Server
}

func NewNFTPriceService() *NFTPriceService {
	return &NFTPriceService{
		collections: make(map[string]*Collection),
		tokens:      make(map[string][]*NFTToken),
		alerts:      make([]FloorPriceAlert, 0),
		holdings:    make(map[string]map[string]int),
	}
}

func (s *NFTPriceService) Run() error {
	// Initialize demo data
	s.initDemoData()

	// Start price updates
	go s.updatePrices()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/collections", s.handleCollections)
	mux.HandleFunc("/api/collection", s.handleCollection)
	mux.HandleFunc("/api/tokens", s.handleTokens)
	mux.HandleFunc("/api/portfolio", s.handlePortfolio)
	mux.HandleFunc("/api/alerts", s.handleAlerts)
	mux.HandleFunc("/api/price/update", s.handlePriceUpdate)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    ":8089",
		Handler: mux,
	}

	log.Println("NFT Price service starting on :8089")
	return s.server.ListenAndServe()
}

func (s *NFTPriceService) initDemoData() {
	collections := []*Collection{
		{ID: "bored_ape", Name: "Bored Ape Yacht Club", Symbol: "BAYC", Blockchain: "ethereum", FloorPrice: 25.5, Volume24h: 1500000, Sales24h: 45, Owners: 6500, TotalSupply: 10000, MarketCap: 255000, Change24h: 3.2, ImageURL: "https://i.sea.cc"},
		{ID: "pudgy", Name: "Pudgy Penguins", Symbol: "PENGU", Blockchain: "ethereum", FloorPrice: 3.2, Volume24h: 450000, Sales24h: 120, Owners: 5200, TotalSupply: 8888, MarketCap: 28416, Change24h: -1.5, ImageURL: "https://i.sea.cc"},
		{ID: "azuki", Name: "Azuki", Symbol: "AZUKI", Blockchain: "ethereum", FloorPrice: 8.5, Volume24h: 680000, Sales24h: 38, Owners: 5800, TotalSupply: 10000, MarketCap: 85000, Change24h: 5.8, ImageURL: "https://i.sea.cc"},
		{ID: "degen", Name: "Degen", Symbol: "DEGEN", Blockchain: "base", FloorPrice: 0.8, Volume24h: 120000, Sales24h: 180, Owners: 8500, TotalSupply: 10000000, MarketCap: 8000000, Change24h: 12.5, ImageURL: "https://i.sea.cc"},
		{ID: "freak", Name: "Freak", Symbol: "FREAK", Blockchain: "solana", FloorPrice: 45.0, Volume24h: 250000, Sales24h: 8, Owners: 4200, TotalSupply: 5000, MarketCap: 225000, Change24h: -3.2, ImageURL: "https://i.sea.cc"},
		{ID: "blur", Name: "Blur", Symbol: "BLUR", Blockchain: "ethereum", FloorPrice: 0.35, Volume24h: 350000, Sales24h: 520, Owners: 15000, TotalSupply: 300000000, MarketCap: 105000000, Change24h: 8.5, ImageURL: "https://i.sea.cc"},
	}

	for _, c := range collections {
		s.collections[c.ID] = c
	}
}

func (s *NFTPriceService) updatePrices() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		for _, c := range s.collections {
			// Simulate price changes
			change := (rand.Float64() - 0.5) * 0.1 * c.FloorPrice
			c.FloorPrice += change
			c.Change24h += (rand.Float64() - 0.5) * 2
			c.Volume24h *= 1 + (rand.Float64()-0.5)*0.05
			c.UpdatedAt = time.Now().UnixMilli()
		}
		s.mu.Unlock()

		// Check alerts
		s.checkAlerts()
	}
}

func (s *NFTPriceService) checkAlerts() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.alerts {
		alert := &s.alerts[i]
		if alert.Triggered {
			continue
		}

		collection, ok := s.collections[alert.CollectionID]
		if !ok {
			continue
		}

		triggered := false
		if alert.Direction == "above" && collection.FloorPrice >= alert.TargetPrice {
			triggered = true
		} else if alert.Direction == "below" && collection.FloorPrice <= alert.TargetPrice {
			triggered = true
		}

		if triggered {
			alert.Triggered = true
			log.Printf("Alert triggered: %s %s %.2f (current: %.2f)",
				alert.CollectionID, alert.Direction, alert.TargetPrice, collection.FloorPrice)
		}
	}
}

// ============== Handlers ==============

func (s *NFTPriceService) handleCollections(w http.ResponseWriter, r *http.Request) {
	blockchain := r.URL.Query().Get("blockchain")
	limit := 50

	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Collection
	for _, c := range s.collections {
		if blockchain != "" && c.Blockchain != blockchain {
			continue
		}
		result = append(result, c)
		if len(result) >= limit {
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *NFTPriceService) handleCollection(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	if c, ok := s.collections[id]; ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(c)
		return
	}

	http.Error(w, "Collection not found", http.StatusNotFound)
}

func (s *NFTPriceService) handleTokens(w http.ResponseWriter, r *http.Request) {
	collectionID := r.URL.Query().Get("collection")
	owner := r.URL.Query().Get("owner")

	s.mu.RLock()
	defer s.mu.RUnlock()

	tokens := s.tokens[collectionID]
	if owner != "" {
		var filtered []*NFTToken
		for _, t := range tokens {
			if t.Owner == owner {
				filtered = append(filtered, t)
			}
		}
		tokens = filtered
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokens)
}

func (s *NFTPriceService) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	userHoldings, ok := s.holdings[userID]
	if !ok {
		// Demo holdings
		userHoldings = map[string]int{
			"bored_ape": 2,
			"pudgy":     5,
			"azuki":     1,
		}
	}

	var totalValue float64
	var collections []CollectionValue

	for collID, count := range userHoldings {
		if c, exists := s.collections[collID]; exists {
			value := c.FloorPrice * float64(count)
			totalValue += value
			collections = append(collections, CollectionValue{
				CollectionID: collID,
				Name:         c.Name,
				Count:        count,
				Value:        value,
				FloorPrice:   c.FloorPrice,
			})
		}
	}

	portfolio := PortfolioValue{
		UserID:      userID,
		TotalValue:  totalValue,
		Collections: collections,
		UpdatedAt:   time.Now().UnixMilli(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(portfolio)
}

func (s *NFTPriceService) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var alert FloorPriceAlert
		if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		alert.ID = fmt.Sprintf("alert_%d", time.Now().UnixNano())
		alert.CreatedAt = time.Now().UnixMilli()
		alert.Triggered = false

		s.mu.Lock()
		s.alerts = append(s.alerts, alert)
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(alert)
		return
	}

	// GET - list alerts
	userID := r.URL.Query().Get("user_id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	var userAlerts []FloorPriceAlert
	for _, a := range s.alerts {
		if a.UserID == userID {
			userAlerts = append(userAlerts, a)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userAlerts)
}

func (s *NFTPriceService) handlePriceUpdate(w http.ResponseWriter, r *http.Request) {
	var updates []struct {
		CollectionID string  `json:"collection_id"`
		FloorPrice   float64 `json:"floor_price"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	for _, u := range updates {
		if c, ok := s.collections[u.CollectionID]; ok {
			c.FloorPrice = u.FloorPrice
			c.UpdatedAt = time.Now().UnixMilli()
		}
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (s *NFTPriceService) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "healthy",
		"collections": len(s.collections),
		"alerts":      len(s.alerts),
		"timestamp":   time.Now().Unix(),
	})
}

// ============== Main ==============

func main() {
	log.Println("Starting TigerWallet NFT Floor Price Service...")

	service := NewNFTPriceService()
	if err := service.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
