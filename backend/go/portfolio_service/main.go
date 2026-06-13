package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// Portfolio position
type Position struct {
	ID          string  `json:"id"`
	UserID     string  `json:"userId"`
	Token      string  `json:"token"`
	Chain      string  `json:"chain"`
	Amount     float64 `json:"amount"`
	CostBasis  float64 `json:"costBasis"`
	CurrentPrice float64 `json:"currentPrice"`
	UpdatedAt  int64   `json:"updatedAt"`
}

// Portfolio summary
type PortfolioSummary struct {
	UserID         string    `json:"userId"`
	TotalValue     float64   `json:"totalValue"`
	TotalCost      float64   `json:"totalCost"`
	TotalPnL       float64   `json:"totalPnL"`
	TotalPnLPercent float64   `json:"totalPnLPercent"`
	Positions      []Position `json:"positions"`
}

// Portfolio service
type PortfolioService struct {
	mu        sync.RWMutex
	positions map[string][]Position
}

func NewPortfolioService() *PortfolioService {
	return &PortfolioService{
		positions: make(map[string][]Position),
	}
}

func main() {
	router := mux.NewRouter()
	svc := NewPortfolioService()

	router.HandleFunc("/api/v1/portfolio", svc.getPortfolio).Methods("GET")
	router.HandleFunc("/api/v1/portfolio/positions", svc.addPosition).Methods("POST")
	router.HandleFunc("/api/v1/portfolio/positions/{id}", svc.updatePosition).Methods("PUT")
	router.HandleFunc("/api/v1/portfolio/positions/{id}", svc.deletePosition).Methods("DELETE")
	router.HandleFunc("/api/v1/portfolio/history", svc.getHistory).Methods("GET")
	router.HandleFunc("/api/v1/portfolio/pnl", svc.getPnL).Methods("GET")
	router.HandleFunc("/api/v1/portfolio/export", svc.exportPortfolio).Methods("GET")

	router.HandleFunc("/health", healthCheck).Methods("GET")

	log.Println("Starting Portfolio Service on port 8003")
	log.Fatal(http.ListenAndServe(":8003", router))
}

func (s *PortfolioService) getPortfolio(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")

	s.mu.RLock()
	positions := s.positions[userID]
	s.mu.RUnlock()

	var totalValue, totalCost float64
	for _, p := range positions {
		totalValue += p.Amount * p.CurrentPrice
		totalCost += p.Amount * p.CostBasis
	}

	pnl := totalValue - totalCost
	pnlPercent := 0.0
	if totalCost > 0 {
		pnlPercent = (pnl / totalCost) * 100
	}

	summary := PortfolioSummary{
		UserID:         userID,
		TotalValue:     totalValue,
		TotalCost:      totalCost,
		TotalPnL:      pnl,
		TotalPnLPercent: pnlPercent,
		Positions:     positions,
	}

	json.NewEncoder(w).Encode(summary)
}

func (s *PortfolioService) addPosition(w http.ResponseWriter, r *http.Request) {
	var position Position
	if err := json.NewDecoder(r.Body).Decode(&position); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	position.ID = fmt.Sprintf("pos_%d", time.Now().UnixNano())
	position.UpdatedAt = time.Now().Unix()

	s.mu.Lock()
	s.positions[position.UserID] = append(s.positions[position.UserID], position)
	s.mu.Unlock()

	json.NewEncoder(w).Encode(position)
}

func (s *PortfolioService) updatePosition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var position Position
	if err := json.NewDecoder(r.Body).Decode(&position); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, p := range s.positions[position.UserID] {
		if p.ID == id {
			position.ID = id
			position.UpdatedAt = time.Now().Unix()
			s.positions[position.UserID][i] = position
			json.NewEncoder(w).Encode(position)
			return
		}
	}

	http.Error(w, "Position not found", http.StatusNotFound)
}

func (s *PortfolioService) deletePosition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		UserID string `json:"userId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	positions := s.positions[req.UserID]
	for i, p := range positions {
		if p.ID == id {
			s.positions[req.UserID] = append(positions[:i], positions[i+1:]...)
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
			return
		}
	}

	http.Error(w, "Position not found", http.StatusNotFound)
}

func (s *PortfolioService) getHistory(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode([]interface{}{})
}

func (s *PortfolioService) getPnL(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"realizedPnL": 0.0,
		"unrealizedPnL": 0.0,
	})
}

func (s *PortfolioService) exportPortfolio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=portfolio.csv")
	w.Header().Set("Content-Type", "text/csv")
	
	userID := r.URL.Query().Get("userId")
	s.mu.RLock()
	positions := s.positions[userID]
	s.mu.RUnlock()

	fmt.Fprintf(w, "Token,Chain,Amount,CostBasis,CurrentPrice,PnL\n")
	for _, p := range positions {
		pnl := (p.CurrentPrice - p.CostBasis) * p.Amount
		fmt.Fprintf(w, "%s,%s,%.4f,%.2f,%.2f,%.2f\n", 
			p.Token, p.Chain, p.Amount, p.CostBasis, p.CurrentPrice, pnl)
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "healthy"})
}