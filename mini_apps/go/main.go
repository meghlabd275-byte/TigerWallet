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

// MiniApp represents a mini application
type MiniApp struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Icon        string   `json:"icon"`
	Category    string   `json:"category"`
	Permissions []string `json:"permissions"`
	Developer  string   `json:"developer"`
	Version    string   `json:"version"`
	Verified   bool     `json:"verified"`
	Installs   int      `json:"installs"`
}

// MiniApp runtime service
type MiniAppService struct {
	mu   sync.RWMutex
	apps map[string]*MiniApp
}

func NewMiniAppService() *MiniAppService {
	svc := &MiniAppService{
		apps: make(map[string]*MiniApp),
	}
	svc.initSampleApps()
	return svc
}

func (s *MiniAppService) initSampleApps() {
	apps := []MiniApp{
		{
			ID: "1", Name: "Price Alert", Description: "Get notified of price changes",
			Category: "Tools", Permissions: []string{"notifications"}, Version: "1.0.0",
		},
		{
			ID: "2", Name: "Portfolio Tracker", Description: "Track your portfolio",
			Category: "Finance", Permissions: []string{"wallet:read"}, Version: "1.0.0",
		},
		{
			ID: "3", Name: "Faucet", Description: "Get testnet tokens",
			Category: "Tools", Permissions: []string{"wallet:write"}, Version: "1.0.0",
		},
	}

	for _, app := range apps {
		s.apps[app.ID] = &app
	}
}

func main() {
	router := mux.NewRouter()
	svc := NewMiniAppService()

	router.HandleFunc("/api/v1/apps", svc.listApps).Methods("GET")
	router.HandleFunc("/api/v1/apps/{id}", svc.getApp).Methods("GET")
	router.HandleFunc("/api/v1/apps/{id}/execute", svc.executeApp).Methods("POST")
	router.HandleFunc("/api/v1/permissions", svc.listPermissions).Methods("GET")
	router.HandleFunc("/health", healthCheck).Methods("GET")

	log.Println("Starting Mini Apps on port 8092")
	log.Fatal(http.ListenAndServe(":8092", router))
}

func (s *MiniAppService) listApps(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var apps []*MiniApp
	for _, app := range s.apps {
		apps = append(apps, app)
	}

	json.NewEncoder(w).Encode(apps)
}

func (s *MiniAppService) getApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	s.mu.RLock()
	defer s.mu.RUnlock()

	if app, ok := s.apps[id]; ok {
		json.NewEncoder(w).Encode(app)
	} else {
		http.Error(w, "App not found", http.StatusNotFound)
	}
}

func (s *MiniAppService) executeApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_ = vars["id"]

	var req struct {
		Action string `json:"action"`
		Data   map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{},
	})
}

func (s *MiniAppService) listPermissions(w http.ResponseWriter, r *http.Request) {
	permissions := []string{
		"wallet:read", "wallet:write",
		"notifications", "location",
		"camera", "contacts",
	}

	json.NewEncoder(w).Encode(permissions)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "healthy"})
}