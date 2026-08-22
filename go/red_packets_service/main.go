/**
 * TigerWallet Red Packets Service — HTTP server
 *
 * Exposes the RedPacketService as a REST API on port :8468.
 * PostgreSQL-backed — real red packet persistence, no in-memory state.
 */

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	redpackets "github.com/tigerwallet/red-packets-service/redpacket"
)

const port = ":8468"

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

	redpackets.SetRedPacketService(pool)
	svc := redpackets.GetRedPacketService()
	if err := svc.Migrate(ctx); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Red packets service connected to PostgreSQL on", dbURL)

	mux := http.NewServeMux()

	// POST /api/v1/red-packets/create — create a red packet
	mux.HandleFunc("/api/v1/red-packets/create", func(w http.ResponseWriter, r *http.Request) {
		if !enforceFeature(w, GatedFeature) {
			return
		}

		if r.Method != http.MethodPost {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		// Decode the request body including the plaintext password, which is
		// immediately hashed by CreateRedPacket and never persisted in plaintext.
		var body struct {
			SenderID      string `json:"sender_id"`
			SenderAddress string `json:"sender_address"`
			TokenAddress  string `json:"token_address"`
			ChainID       uint64 `json:"chain_id"`
			TotalAmount   string `json:"total_amount"`
			Quantity      int    `json:"quantity"`
			ClaimType     string `json:"claim_type"`
			Password      string `json:"password"`
			Message       string `json:"message"`
			TxHash        string `json:"tx_hash"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, apiResponse{Error: "invalid request body"})
			return
		}
		p := redpackets.RedPacket{
			SenderID:      body.SenderID,
			SenderAddress: body.SenderAddress,
			TokenAddress:  body.TokenAddress,
			ChainID:       body.ChainID,
			TotalAmount:   body.TotalAmount,
			Quantity:      body.Quantity,
			ClaimType:     body.ClaimType,
			PasswordHash:  body.Password,
			Message:       body.Message,
			TxHash:        body.TxHash,
		}
		created, err := svc.CreateRedPacket(r.Context(), &p)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 201, apiResponse{Success: true, Data: created})
	})

	// POST /api/v1/red-packets/claim — claim a red packet
	mux.HandleFunc("/api/v1/red-packets/claim", func(w http.ResponseWriter, r *http.Request) {
		if !enforceFeature(w, GatedFeature) {
			return
		}

		if r.Method != http.MethodPost {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		var body struct {
			PacketID       string `json:"packet_id"`
			ClaimerID      string `json:"claimer_id"`
			ClaimerAddress string `json:"claimer_address"`
			Password       string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, apiResponse{Error: "invalid request body"})
			return
		}
		claim, err := svc.Claim(r.Context(), body.PacketID, body.ClaimerID, body.ClaimerAddress, body.Password)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 201, apiResponse{Success: true, Data: claim})
	})

	// GET /api/v1/red-packets/sent?user_id= — list packets sent by a user
	mux.HandleFunc("/api/v1/red-packets/sent", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		userID := r.URL.Query().Get("user_id")
		packets, err := svc.GetSentPackets(r.Context(), userID)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{Success: true, Data: map[string]any{"packets": packets}})
	})

	// GET /api/v1/red-packets/received?user_id= — list packets a user has claimed
	mux.HandleFunc("/api/v1/red-packets/received", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		userID := r.URL.Query().Get("user_id")
		packets, err := svc.GetReceivedPackets(r.Context(), userID)
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{Success: true, Data: map[string]any{"packets": packets}})
	})

	// GET /api/v1/red-packets/{id} — get red packet by ID
	mux.HandleFunc("/api/v1/red-packets/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		id := r.URL.Path[len("/api/v1/red-packets/"):]
		if id == "" {
			writeJSON(w, 400, apiResponse{Error: "packet id is required"})
			return
		}
		p, err := svc.GetRedPacket(r.Context(), id)
		if err != nil {
			writeJSON(w, 404, apiResponse{Error: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{Success: true, Data: p})
	})

	log.Printf("Red Packets service listening on %s", port)
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
