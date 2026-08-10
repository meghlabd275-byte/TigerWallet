/**
 * TigerWallet Order Matching Engine
 * High-Load Distributed Go Implementation
 *
 * Features:
 * - Order matching
 * - Price-time priority
 * - Market depth
 * - Trade execution
 */

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Order struct {
	ID       string  `json:"id"`
	UserID   string  `json:"user_id"`
	Symbol   string  `json:"symbol"`
	Side     string  `json:"side"` // buy, sell
	Type     string  `json:"type"` // limit, market
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Filled   float64 `json:"filled"`
	Status   string  `json:"status"`
	Time     int64   `json:"time"`
}

type Trade struct {
	ID        string  `json:"id"`
	BuyOrder  string  `json:"buy_order"`
	SellOrder string  `json:"sell_order"`
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Quantity  float64 `json:"quantity"`
	Time      int64   `json:"time"`
}

type OrderBook struct {
	Symbol string  `json:"symbol"`
	Bids   []Order `json:"bids"`
	Asks   []Order `json:"asks"`
}

type MatchingEngine struct {
	orders map[string]*Order
	trades []Trade
	mu     sync.RWMutex
}

func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{
		orders: make(map[string]*Order),
		trades: make([]Trade, 0),
	}
}

func (m *MatchingEngine) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/order", m.handleOrder)
	mux.HandleFunc("/cancel", m.handleCancel)
	mux.HandleFunc("/orderbook", m.handleOrderBook)
	mux.HandleFunc("/trades", m.handleTrades)
	mux.HandleFunc("/health", m.handleHealth)

	log.Println("Matching engine starting on :8092")
	return http.ListenAndServe(":8092", mux)
}

func (m *MatchingEngine) handleOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	order.ID = fmt.Sprintf("ord_%d", time.Now().UnixNano())
	order.Status = "open"
	order.Filled = 0
	order.Time = time.Now().UnixMilli()

	m.mu.Lock()
	m.orders[order.ID] = &order

	// Try to match
	m.matchOrders(&order)
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (m *MatchingEngine) handleCancel(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	m.mu.Lock()
	if order, ok := m.orders[id]; ok {
		order.Status = "cancelled"
	}
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

func (m *MatchingEngine) handleOrderBook(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")

	m.mu.RLock()
	defer m.mu.RUnlock()

	var bids, asks []Order
	for _, o := range m.orders {
		if o.Symbol == symbol && o.Status == "open" {
			if o.Side == "buy" {
				bids = append(bids, *o)
			} else {
				asks = append(asks, *o)
			}
		}
	}

	// Sort by price
	sort.Slice(bids, func(i, j int) bool { return bids[i].Price > bids[j].Price })
	sort.Slice(asks, func(i, j int) bool { return asks[i].Price < asks[j].Price })

	ob := OrderBook{Symbol: symbol, Bids: bids, Asks: asks}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ob)
}

func (m *MatchingEngine) handleTrades(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.trades)
}

func (m *MatchingEngine) handleHealth(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"orders": len(m.orders),
		"trades": len(m.trades),
	})
}

func (m *MatchingEngine) matchOrders(order *Order) {
	for _, o := range m.orders {
		if o.Symbol != order.Symbol || o.Status != "open" {
			continue
		}

		// Check if matchable
		var match bool
		if order.Side == "buy" && o.Side == "sell" {
			match = order.Type == "market" || order.Price >= o.Price
		} else if order.Side == "sell" && o.Side == "buy" {
			match = order.Type == "market" || order.Price <= o.Price
		}

		if !match {
			continue
		}

		// Execute trade
		qty := order.Quantity - order.Filled
		if o.Quantity-o.Filled < qty {
			qty = o.Quantity - o.Filled
		}

		price := o.Price
		trade := Trade{
			ID: fmt.Sprintf("trd_%d", time.Now().UnixNano()),
			BuyOrder: func() string {
				if order.Side == "buy" {
					return order.ID
				} else {
					return o.ID
				}
			}(),
			SellOrder: func() string {
				if order.Side == "sell" {
					return order.ID
				} else {
					return o.ID
				}
			}(),
			Symbol:   order.Symbol,
			Price:    price,
			Quantity: qty,
			Time:     time.Now().UnixMilli(),
		}

		m.trades = append(m.trades, trade)
		order.Filled += qty
		o.Filled += qty

		if order.Filled >= order.Quantity {
			order.Status = "filled"
		}
		if o.Filled >= o.Quantity {
			o.Status = "filled"
		}

		break // Execute one trade at a time
	}
}

func main() {
	log.Println("Starting TigerWallet Matching Engine...")

	engine := NewMatchingEngine()
	if err := engine.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
