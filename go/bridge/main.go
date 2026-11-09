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
	// Routes are reported from the registered adapters themselves, so a bridge
	// that has no fee schedule configured (and therefore cannot quote) does not
	// appear as available.
	type bridgeRoute struct {
		Bridge string   `json:"bridge"`
		Chains []uint64 `json:"chains"`
	}
	routes := make([]bridgeRoute, 0)
	for _, br := range s.agg.Bridges() {
		if chains := br.SupportedChains(); len(chains) > 0 {
			routes = append(routes, bridgeRoute{Bridge: br.Name(), Chains: chains})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"routes": routes})
}

func (s *Server) handleQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	// Accept both camelCase (srcChain) and snake_case (from_chain) field names
	// so the frontend contract works regardless of convention.
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	pick := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := raw[k]; ok {
				var s string
				if json.Unmarshal(v, &s) == nil && s != "" {
					return s
				}
				var f float64
				if json.Unmarshal(v, &f) == nil {
					return strconv.FormatFloat(f, 'f', -1, 64)
				}
			}
		}
		return ""
	}
	srcChainStr := pick("srcChain", "from_chain")
	dstChainStr := pick("dstChain", "to_chain")
	srcToken := pick("srcToken", "token", "from_token")
	dstToken := pick("dstToken", "to_token")
	if dstToken == "" {
		dstToken = srcToken
	}
	amountStr := pick("amount", "from_amount")
	recipient := pick("recipient")
	srcChain, err := strconv.ParseUint(srcChainStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid or missing srcChain/from_chain"}`, http.StatusBadRequest)
		return
	}
	dstChain, err := strconv.ParseUint(dstChainStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid or missing dstChain/to_chain"}`, http.StatusBadRequest)
		return
	}
	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		// amount may be a decimal human-unit string; accept it as an estimate
		// by converting to smallest unit assuming 18 decimals.
		f, ferr := strconv.ParseFloat(amountStr, 64)
		if ferr != nil || f <= 0 {
			http.Error(w, `{"error":"invalid amount"}`, http.StatusBadRequest)
			return
		}
		amount = big.NewInt(int64(f * 1e18))
	}
	quoteReq := aggregator.QuoteRequest{
		SrcChain:  srcChain,
		DstChain:  dstChain,
		SrcToken:  srcToken,
		DstToken:  dstToken,
		Amount:    amount,
		Recipient: recipient,
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
	// Accept both camelCase and snake_case field names from the frontend.
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	pick := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := raw[k]; ok {
				var s string
				if json.Unmarshal(v, &s) == nil {
					return s
				}
			}
		}
		return ""
	}
	bridge := pick("bridge", "bridge_name")
	if bridge == "" {
		bridge = "Stargate"
	}
	recipient := pick("recipient", "user_id")
	fromChain := pick("from_chain", "srcChain")
	toChain := pick("to_chain", "dstChain")
	token := pick("token", "asset")
	amount := pick("amount", "value")

	if fromChain == "" || toChain == "" || recipient == "" || amount == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "from_chain, to_chain, recipient and amount are required",
		})
		return
	}

	// A bridge transfer is an on-chain action: the source-chain deposit has to
	// be signed and broadcast by the user through wallet_api /send, and the
	// destination leg exists only once a relayer observes that deposit. This
	// endpoint returns the unsigned deposit call the client must sign. No
	// transfer id is issued, because nothing has happened on chain yet and an
	// id here would imply a transfer that does not exist.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  "action_required",
		"action":  "sign_and_broadcast_deposit",
		"message": "Sign the deposit transaction with wallet_api /send. The transfer exists only after that transaction is confirmed on " + fromChain + ".",
		"bridge":  bridge,
		"deposit": map[string]interface{}{
			"from_chain": fromChain,
			"to_chain":   toChain,
			"token":      token,
			"amount":     amount,
			"recipient":  recipient,
		},
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
	empty := []interface{}{}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transfers": empty,
		"history":   empty,
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
