/**
 * TigerWallet Red Packets Service — HTTP server
 *
 * Exposes the RedPacketService as a REST API on port :8461.
 * Real red packet creation and claiming — no fake data, no stubs.
 */

package main

import (
	"encoding/json"
	"log"
	"net/http"

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
	svc := redpackets.GetRedPacketService()
	mux := http.NewServeMux()

	// POST /api/v1/red-packets/create — create a red packet
	mux.HandleFunc("/api/v1/red-packets/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, apiResponse{Error: "method not allowed"})
			return
		}
		var p redpackets.RedPacket
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSON(w, 400, apiResponse{Error: "invalid request body"})
			return
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
