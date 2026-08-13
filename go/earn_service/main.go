/**
 * TigerWallet Earn Service — HTTP server
 *
 * Exposes the EarnService as a REST API on port :8458.
 * Real earn product management and deposits — no fake data, no stubs.
 */

package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/tigerwallet/earn-service/earn"
)

const port = ":8466"

type apiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	svc := earn.GetEarnService()
	mux := http.NewServeMux()

	// GET /api/v1/earn/products — list all products (optional ?type=&status=)
	mux.HandleFunc("/api/v1/earn/products", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		productType := r.URL.Query().Get("type")
		status := r.URL.Query().Get("status")
		products, err := svc.GetAllProducts(r.Context(), productType, status)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{Success: true, Data: products})
	})

	// POST /api/v1/earn/products — create product (admin)
	mux.HandleFunc("/api/v1/earn/products/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		var p earn.EarnProduct
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSON(w, 400, apiResponse{Error: "invalid request body"})
			return
		}
		created, err := svc.CreateProduct(r.Context(), &p)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 201, apiResponse{Success: true, Data: created})
	})

	// POST /api/v1/earn/deposit — create deposit
	mux.HandleFunc("/api/v1/earn/deposit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		var d earn.UserDeposit
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			writeJSON(w, 400, apiResponse{Error: "invalid request body"})
			return
		}
		created, err := svc.Deposit(r.Context(), &d)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 201, apiResponse{Success: true, Data: created})
	})

	// POST /api/v1/earn/withdraw — withdraw from deposit
	mux.HandleFunc("/api/v1/earn/withdraw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		var body struct {
			DepositID string `json:"deposit_id"`
			Amount    string `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, apiResponse{Error: "invalid request body"})
			return
		}
		result, err := svc.Withdraw(r.Context(), body.DepositID, body.Amount)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{Success: true, Data: map[string]string{"result": result}})
	})

	// POST /api/v1/earn/claim — claim rewards
	mux.HandleFunc("/api/v1/earn/claim", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		var body struct {
			DepositID string `json:"deposit_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, apiResponse{Error: "invalid request body"})
			return
		}
		result, err := svc.Claim(r.Context(), body.DepositID)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{Success: true, Data: map[string]string{"result": result}})
	})

	// GET /api/v1/earn/deposits?user_id= — list user deposits
	mux.HandleFunc("/api/v1/earn/deposits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			writeJSON(w, 400, apiResponse{Error: "user_id is required"})
			return
		}
		deposits, err := svc.GetUserDeposits(r.Context(), userID)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{Success: true, Data: deposits})
	})

	log.Printf("Earn service listening on %s", port)
	srv := &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  15 * 1e9,
		WriteTimeout: 15 * 1e9,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
