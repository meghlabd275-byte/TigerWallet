package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

// API Key for developers
type APIKey struct {
	ID          string `json:"id"`
	DeveloperID string `json:"developerId"`
	Key        string `json:"key"`
	Name       string `json:"name"`
	Permissions []string `json:"permissions"`
	RateLimit  int    `json:"rateLimit"`
	CreatedAt  int64  `json:"createdAt"`
	ExpiresAt int64  `json:"expiresAt"`
}

// Developer profile
type Developer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	APIKeys   []string `json:"apiKeys"`
	Apps      []string `json:"apps"`
	CreatedAt int64   `json:"createdAt"`
}

// SDK package
type SDKPackage struct {
	Name        string `json:"name"`
	Version    string `json:"version"`
	Language   string `json:"language"`
	DownloadURL string `json:"downloadUrl"`
	Checksum  string `json:"checksum"`
}

// Developer platform service
type DeveloperService struct {
	mu         sync.RWMutex
	developers map[string]*Developer
	apiKeys    map[string]*APIKey
	sdks       map[string]*SDKPackage
}

func NewDeveloperService() *DeveloperService {
	svc := &DeveloperService{
		developers: make(map[string]*Developer),
		apiKeys:    make(map[string]*APIKey),
		sdks:      make(map[string]*SDKPackage),
	}
	svc.initSDKs()
	return svc
}

func (s *DeveloperService) initSDKs() {
	sdks := []SDKPackage{
		{Name: "wallet-sdk", Version: "1.0.0", Language: "typescript", DownloadURL: "https://npmjs.com/@tigerwallet/sdk"},
		{Name: "wallet-sdk", Version: "1.0.0", Language: "python", DownloadURL: "https://pypi.org/project/tigerwallet-sdk"},
		{Name: "dapp-sdk", Version: "1.0.0", Language: "typescript", DownloadURL: "https://npmjs.com/@tigerwallet/dapp-sdk"},
	}

	for _, sdk := range sdks {
		key := fmt.Sprintf("%s:%s", sdk.Name, sdk.Language)
		s.sdks[key] = &sdk
	}
}

func main() {
	router := mux.NewRouter()
	svc := NewDeveloperService()

	// Developer routes
	router.HandleFunc("/api/v1/developers", svc.registerDeveloper).Methods("POST")
	router.HandleFunc("/api/v1/developers/{id}", svc.getDeveloper).Methods("GET")
	router.HandleFunc("/api/v1/developers/{id}/keys", svc.createAPIKey).Methods("POST")

	// SDK routes
	router.HandleFunc("/api/v1/sdk", svc.listSDKs).Methods("GET")
	router.HandleFunc("/api/v1/sdk/{name}/{lang}", svc.getSDK).Methods("GET")

	// Docs routes
	router.HandleFunc("/api/v1/docs", svc.getDocs).Methods("GET")

	router.HandleFunc("/health", healthCheck).Methods("GET")

	log.Println("Starting Developer Platform on port 8093")
	log.Fatal(http.ListenAndServe(":8093", router))
}

func (s *DeveloperService) registerDeveloper(w http.ResponseWriter, r *http.Request) {
	var dev Developer
	if err := json.NewDecoder(r.Body).Decode(&dev); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dev.ID = fmt.Sprintf("dev_%d", len(s.developers)+1)

	s.mu.Lock()
	s.developers[dev.ID] = &dev
	s.mu.Unlock()

	json.NewEncoder(w).Encode(dev)
}

func (s *DeveloperService) getDeveloper(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	s.mu.RLock()
	defer s.mu.RUnlock()

	if dev, ok := s.developers[id]; ok {
		json.NewEncoder(w).Encode(dev)
	} else {
		http.Error(w, "Developer not found", http.StatusNotFound)
	}
}

func (s *DeveloperService) createAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_ = vars["id"]

	var req struct {
		Name       string   `json:"name"`
		Permissions []string `json:"permissions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	apiKey := APIKey{
		ID:          fmt.Sprintf("key_%d", len(s.apiKeys)+1),
		Key:         "tw_" + fmt.Sprintf("%d", len(s.apiKeys)),
		Name:        req.Name,
		Permissions: req.Permissions,
		RateLimit:   1000,
	}

	json.NewEncoder(w).Encode(apiKey)
}

func (s *DeveloperService) listSDKs(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sdks []*SDKPackage
	for _, sdk := range s.sdks {
		sdks = append(sdks, sdk)
	}

	json.NewEncoder(w).Encode(sdks)
}

func (s *DeveloperService) getSDK(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	lang := vars["lang"]

	key := fmt.Sprintf("%s:%s", name, lang)

	s.mu.RLock()
	defer s.mu.RUnlock()

	if sdk, ok := s.sdks[key]; ok {
		json.NewEncoder(w).Encode(sdk)
	} else {
		http.Error(w, "SDK not found", http.StatusNotFound)
	}
}

func (s *DeveloperService) getDocs(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"version": "1.0.0",
		"endpoints": []string{
			"/api/v1/wallet/create",
			"/api/v1/wallet/sign",
			"/api/v1/transaction/send",
			"/api/v1/balance/{address}",
		},
	}

	json.NewEncoder(w).Encode(docs)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "healthy"})
}