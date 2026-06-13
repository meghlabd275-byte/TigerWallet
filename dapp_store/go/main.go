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

// DApp represents a decentralized application
type DApp struct {
	ID          string   `json:"id"`
	Name       string   `json:"name"`
	Description string  `json:"description"`
	Category   string   `json:"category"`
	URL        string   `json:"url"`
	Icon       string   `json:"icon"`
	Chain      string   `json:"chain"`
	Verified   bool     `json:"verified"`
	Rating     float64  `json:"rating"`
	Reviews    int      `json:"reviews"`
	Installs   int      `json:"installs"`
	Developers []string `json:"developers"`
	Tags      []string `json:"tags"`
}

// Review represents a user review
type Review struct {
	ID        string `json:"id"`
	DAppID   string `json:"dappId"`
	UserID   string `json:"userId"`
	Rating   int    `json:"rating"`
	Review   string `json:"review"`
	Helpful  int    `json:"helpful"`
	Date    int64  `json:"date"`
}

// DAppStore is the main store service
type DAppStore struct {
	mu    sync.RWMutex
	dapps map[string]*DApp
}

func NewDAppStore() *DAppStore {
	store := &DAppStore{
		dapps: make(map[string]*DApp),
	}
	store.initSampleApps()
	return store
}

func (s *DAppStore) initSampleApps() {
	apps := []DApp{
		{
			ID: "1", Name: "Uniswap", Description: "DEX Protocol", Category: "DeFi",
			URL: "uniswap.org", Chain: "ethereum", Verified: true, Rating: 4.8,
			Reviews: 1000, Installs: 500000, Tags: []string{"DEX", "Swap"},
		},
		{
			ID: "2", Name: "OpenSea", Description: "NFT Marketplace", Category: "NFT",
			URL: "opensea.io", Chain: "ethereum", Verified: true, Rating: 4.5,
			Reviews: 800, Installs: 300000, Tags: []string{"NFT", "Marketplace"},
		},
		{
			ID: "3", Name: "Aave", Description: "Lending Protocol", Category: "DeFi",
			URL: "aave.com", Chain: "ethereum", Verified: true, Rating: 4.7,
			Reviews: 500, Installs: 200000, Tags: []string{"Lending", "Borrow"},
		},
	}
	
	for _, app := range apps {
		s.dapps[app.ID] = &app
	}
}

func main() {
	router := mux.NewRouter()
	store := NewDAppStore()

	// Routes
	router.HandleFunc("/api/v1/dapps", store.listDApps).Methods("GET")
	router.HandleFunc("/api/v1/dapps/{id}", store.getDApp).Methods("GET")
	router.HandleFunc("/api/v1/dapps", store.createDApp).Methods("POST")
	router.HandleFunc("/api/v1/dapps/{id}/reviews", store.addReview).Methods("POST")
	router.HandleFunc("/api/v1/dapps/search", store.searchDApps).Methods("GET")
	router.HandleFunc("/api/v1/categories", store.listCategories).Methods("GET")
	router.HandleFunc("/health", healthCheck).Methods("GET")

	log.Println("Starting DApp Store on port 8090")
	log.Fatal(http.ListenAndServe(":8090", router))
}

func (s *DAppStore) listDApps(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	chain := r.URL.Query().Get("chain")

	s.mu.RLock()
	var dapps []*DApp
	for _, app := range s.dapps {
		if (category == "" || app.Category == category) && (chain == "" || app.Chain == chain) {
			dapps = append(dapps, app)
		}
	}
	s.mu.RUnlock()

	json.NewEncoder(w).Encode(dapps)
}

func (s *DAppStore) getDApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	s.mu.RLock()
	defer s.mu.RUnlock()
	
	app, ok := s.dapps[id]
	if !ok {
		http.Error(w, "DApp not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(app)
}

func (s *DAppStore) createDApp(w http.ResponseWriter, r *http.Request) {
	var app DApp
	if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	app.ID = fmt.Sprintf("dapp_%d", time.Now().UnixNano())

	s.mu.Lock()
	s.dapps[app.ID] = &app
	s.mu.Unlock()

	json.NewEncoder(w).Encode(app)
}

func (s *DAppStore) addReview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dappID := vars["id"]

	var review Review
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	review.ID = fmt.Sprintf("rev_%d", time.Now().UnixNano())
	review.DAppID = dappID
	review.Date = time.Now().Unix()

	json.NewEncoder(w).Encode(review)
}

func (s *DAppStore) searchDApps(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	s.mu.RLock()
	var results []*DApp
	for _, app := range s.dapps {
		if contains(app.Name, query) || contains(app.Description, query) {
			results = append(results, app)
		}
	}
	s.mu.RUnlock()

	json.NewEncoder(w).Encode(results)
}

func (s *DAppStore) listCategories(w http.ResponseWriter, r *http.Request) {
	categories := []string{"DeFi", "NFT", "Gaming", "Social", "Tools", "Bridge", "Analytics"}

	json.NewEncoder(w).Encode(categories)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "healthy"})
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0
}