/**
 * TigerWallet Airdrop Service — HTTP server
 *
 * Exposes the AirdropService as a REST API on port :8465.
 * Uses Go stdlib net/http for high-load distributed operation.
 * PostgreSQL-backed — real campaign/claim persistence, no in-memory state.
 */

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tigerwallet/airdrop-service/airdrop"
)

const port = ":8465"

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
	ctx := context.Background()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	airdrop.SetAirdropService(pool)
	svc := airdrop.GetAirdropService()
	if err := svc.Migrate(ctx); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Airdrop service connected to PostgreSQL on", dbURL)

	mux := http.NewServeMux()

	// GET /api/v1/airdrop/campaigns — list all campaigns
	mux.HandleFunc("/api/v1/airdrop/campaigns", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		campaigns, err := svc.GetAllCampaigns(r.Context())
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{Success: true, Data: campaigns})
	})

	// POST /api/v1/airdrop/campaigns — create campaign
	mux.HandleFunc("/api/v1/airdrop/campaigns/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		var c airdrop.AirdropCampaign
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeJSON(w, 400, apiResponse{Error: "invalid request body"})
			return
		}
		created, err := svc.CreateCampaign(r.Context(), &c)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 201, apiResponse{Success: true, Data: created})
	})

	// POST /api/v1/airdrop/claim — create a claim
	mux.HandleFunc("/api/v1/airdrop/claim", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		var claim airdrop.AirdropClaim
		if err := json.NewDecoder(r.Body).Decode(&claim); err != nil {
			writeJSON(w, 400, apiResponse{Error: "invalid request body"})
			return
		}
		created, err := svc.CreateClaim(r.Context(), &claim)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 201, apiResponse{Success: true, Data: created})
	})

	// GET /api/v1/airdrop/campaigns/{id} — get campaign by ID
	mux.HandleFunc("/api/v1/airdrop/campaigns/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		id := r.URL.Path[len("/api/v1/airdrop/campaigns/"):]
		if id == "" {
			campaigns, err := svc.GetAllCampaigns(r.Context())
			if err != nil {
				writeJSON(w, 500, apiResponse{Error: err.Error()})
				return
			}
			writeJSON(w, 200, apiResponse{Success: true, Data: campaigns})
			return
		}
		c, err := svc.GetCampaign(r.Context(), id)
		if err != nil {
			writeJSON(w, 404, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{Success: true, Data: c})
	})

	// POST /api/v1/airdrop/claim/{id}/confirm — confirm claim with tx hash
	mux.HandleFunc("/api/v1/airdrop/claim/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		parts := splitPath(r.URL.Path, "/api/v1/airdrop/claim/")
		if len(parts) < 2 || parts[1] != "confirm" {
			writeJSON(w, 400, apiResponse{Error: "invalid path; expected /api/v1/airdrop/claim/{id}/confirm"})
			return
		}
		var body struct {
			TxHash string `json:"tx_hash"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, apiResponse{Error: "invalid request body"})
			return
		}
		if err := svc.ClaimTokens(r.Context(), parts[0], body.TxHash); err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{Success: true})
	})

	log.Printf("Airdrop service listening on %s", port)
	srv := &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  15 * 1e9,
		WriteTimeout: 15 * 1e9,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
	_ = strconv.Itoa // keep import if unused
}

func splitPath(path, prefix string) []string {
	rest := path[len(prefix):]
	var parts []string
	start := 0
	for i := 0; i < len(rest); i++ {
		if rest[i] == '/' {
			parts = append(parts, rest[start:i])
			start = i + 1
		}
	}
	if start < len(rest) {
		parts = append(parts, rest[start:])
	}
	return parts
}
