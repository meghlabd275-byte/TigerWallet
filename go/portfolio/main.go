package main

import (
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/tigerwallet/portfolio/portfolio"
)

func main() {
	svc := portfolio.NewService(&portfolio.Config{
		PriceUpdateInterval: 60 * time.Second,
		HistoryRetention:    24 * time.Hour,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler("portfolio"))

	mux.HandleFunc("GET /api/v1/portfolio/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userId")
		p, err := svc.GetPortfolio(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, p)
	})

	mux.HandleFunc("GET /api/v1/portfolio/{userId}/history", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userId")
		p, err := svc.GetPortfolio(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, p.History)
	})

	mux.HandleFunc("GET /api/v1/portfolio/{userId}/allocation", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userId")
		p, err := svc.GetPortfolio(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		allocation := buildAllocation(p)
		writeJSON(w, http.StatusOK, allocation)
	})

	addr := ":" + port("8473")
	log.Printf("portfolio service listening on %s", addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

type allocationEntry struct {
	Symbol   string `json:"symbol"`
	ChainID  uint64 `json:"chainId"`
	ValueUSD string `json:"valueUSD"`
	Percent  string `json:"percent"`
}

// buildAllocation derives a per-asset allocation from the portfolio. The
// underlying Balance.ValueUSD is a *big.Rat; we use its string form so the
// numbers are exact and never require float conversion.
func buildAllocation(p *portfolio.Portfolio) []allocationEntry {
	total := p.TotalValueUSD
	entries := make([]allocationEntry, 0, len(p.Balances))
	for _, b := range p.Balances {
		valStr := "0"
		if b.ValueUSD != nil {
			valStr = b.ValueUSD.RatString()
		}
		pctStr := "0"
		if total != nil && total.Sign() != 0 && b.ValueUSD != nil {
			// percent = value * 100 / total
			scaled := new(big.Rat).Mul(b.ValueUSD, big.NewRat(100, 1))
			scaled.Quo(scaled, total)
			pctStr = scaled.RatString()
		}
		entries = append(entries, allocationEntry{
			Symbol:   b.Symbol,
			ChainID:  b.ChainID,
			ValueUSD: valStr,
			Percent:  pctStr,
		})
	}
	return entries
}

func port(def string) string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return def
}

func healthHandler(service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": service})
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
