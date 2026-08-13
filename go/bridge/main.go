package main

// Bridge Aggregator HTTP server — exposes the cross-chain bridge aggregator
// as a REST API. Real bridge quote/execute logic; execution returns
// action_required (client broadcasts via wallet_api /send), never a fake tx hash.

import (
	"context"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	aggregator "github.com/tigerwallet/bridge/aggregator"
)

type Server struct {
	agg *aggregator.BridgeAggregator
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8470"
	}

	agg := aggregator.NewBridgeAggregator(&aggregator.Config{
		MaxBridgeTime:  30 * time.Minute,
		MaxSlippage:    3.0,
		EnableMultiHop: true,
	})
	agg.RegisterBridge("Stargate", aggregator.NewStargateBridge(""))
	agg.RegisterBridge("Across", aggregator.NewAcrossBridge(""))

	srv := &Server{agg: agg}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "bridge"})
	})
	mux.HandleFunc("/api/v1/bridge/routes", srv.handleRoutes)
	mux.HandleFunc("/api/v1/bridge/quote", srv.handleQuote)
	mux.HandleFunc("/api/v1/bridge/transfer", srv.handleTransfer)
	mux.HandleFunc("/api/v1/bridge/history", srv.handleHistory)
	mux.HandleFunc("/api/v1/bridge/status", srv.handleStatus)

	log.Printf("Bridge aggregator service starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func (s *Server) handleRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"routes": []map[string]interface{}{
			{"bridge": "Stargate", "chains": []uint64{1, 56, 137, 42161, 10, 8453}, "slippage": 0.5},
			{"bridge": "Across", "chains": []uint64{1, 10, 42161, 137, 8453}, "slippage": 0.3},
		},
	})
}

func (s *Server) handleQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SrcChain  uint64 `json:"srcChain"`
		DstChain  uint64 `json:"dstChain"`
		SrcToken  string `json:"srcToken"`
		DstToken  string `json:"dstToken"`
		Amount    string `json:"amount"`
		Recipient string `json:"recipient"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		http.Error(w, `{"error":"invalid amount"}`, http.StatusBadRequest)
		return
	}
	quoteReq := aggregator.QuoteRequest{
		SrcChain:  req.SrcChain,
		DstChain:  req.DstChain,
		SrcToken:  req.SrcToken,
		DstToken:  req.DstToken,
		Amount:    amount,
		Recipient: req.Recipient,
	}
	quote, err := s.agg.GetQuote(context.Background(), quoteReq)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quote)
}

func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	// Bridge execution requires an on-chain transaction signed and broadcast
	// by the user via wallet_api /send. We return action_required with the
	// quote details so the client can construct and broadcast the tx.
	var req struct {
		QuoteID   string `json:"quoteId"`
		Bridge    string `json:"bridge"`
		Recipient string `json:"recipient"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "action_required",
		"message":         "Bridge transfer requires client to sign and broadcast via wallet_api /send",
		"bridge":          req.Bridge,
		"recipient":       req.Recipient,
	})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	// Bridge history requires on-chain indexing (query src/dst chain logs).
	// Without a live indexer configured, return an honest empty list.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transfers": []interface{}{},
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	txHash := r.URL.Query().Get("txHash")
	if txHash == "" {
		http.Error(w, `{"error":"txHash parameter required"}`, http.StatusBadRequest)
		return
	}
	// Real status would query the bridge's messaging layer (LayerZero/CCTP).
	// Without a live indexer, return pending.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"txHash": txHash,
		"status": "pending",
	})
}

func init() {
	_ = strconv.Itoa // keep import if unused later
}
