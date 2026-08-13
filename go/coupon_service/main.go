/**
 * TigerWallet Coupon Service — HTTP server
 *
 * Exposes the CouponService as a REST API on port :8460.
 * Real coupon validation and management — no fake data, no stubs.
 */

package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/tigerwallet/coupon-service/coupon"
)

const port = ":8467"

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
	svc := coupon.GetCouponService()
	mux := http.NewServeMux()

	// POST /api/v1/coupon/validate — validate a coupon code
	mux.HandleFunc("/api/v1/coupon/validate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		var body struct {
			Code    string `json:"code"`
			ChainID string `json:"chain_id"`
			Pair    string `json:"pair"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, apiResponse{Error: "invalid request body"})
			return
		}
		c, err := svc.ValidateCoupon(r.Context(), body.Code, body.ChainID, body.Pair)
		if err != nil {
			writeJSON(w, 404, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{Success: true, Data: c})
	})

	// POST /api/v1/coupon/create — create a coupon (admin)
	mux.HandleFunc("/api/v1/coupon/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		var c coupon.Coupon
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeJSON(w, 400, apiResponse{Error: "invalid request body"})
			return
		}
		created, err := svc.CreateCoupon(r.Context(), &c)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 201, apiResponse{Success: true, Data: created})
	})

	// GET /api/v1/coupon/{code} — get coupon by code
	mux.HandleFunc("/api/v1/coupon/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		code := r.URL.Path[len("/api/v1/coupon/"):]
		if code == "" {
			writeJSON(w, 400, apiResponse{Error: "coupon code is required"})
			return
		}
		c, err := svc.GetCoupon(r.Context(), code)
		if err != nil {
			writeJSON(w, 404, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{Success: true, Data: c})
	})

	log.Printf("Coupon service listening on %s", port)
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
