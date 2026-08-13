package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tigerwallet/token-deployer-service/token"
)

func main() {
	svc := token.GetTokenDeployerService()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler("token_deployer_service"))

	mux.HandleFunc("POST /api/v1/token/deploy", func(w http.ResponseWriter, r *http.Request) {
		var d token.TokenDeployment
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		created, err := svc.CreateDeployment(r.Context(), &d)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		// Validation passed and a pending deployment was recorded. We mark it
		// deploying with an empty tx hash; the real on-chain deploy tx hash is
		// filled in once the deployment is broadcast. No fabricated hash.
		if err := svc.Deploy(r.Context(), created.ID, ""); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, created)
	})

	mux.HandleFunc("GET /api/v1/token/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		d, err := svc.GetDeployment(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, d)
	})

	mux.HandleFunc("GET /api/v1/token/deployments/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userId")
		deployments, err := svc.GetUserDeployments(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, deployments)
	})

	addr := ":" + port("8474")
	log.Printf("token_deployer_service listening on %s", addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
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
