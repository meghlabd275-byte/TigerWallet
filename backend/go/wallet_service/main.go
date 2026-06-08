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

// Wallet represents a user wallet
type Wallet struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	Address     string    `json:"address"`
	Chain       string    `json:"chain"`
	WalletType  string    `json:"walletType"`
	Name        string    `json:"name"`
	CreatedAt   int64     `json:"createdAt"`
	LastUsed   int64     `json:"lastUsed"`
	Balance    float64   `json:"balance"`
}

// Wallet service
type WalletService struct {
	mu      sync.RWMutex
	wallets map[string]*Wallet
}

func NewWalletService() *WalletService {
	return &WalletService{
		wallets: make(map[string]*Wallet),
	}
}

func main() {
	router := mux.NewRouter()
	svc := NewWalletService()

	router.HandleFunc("/api/v1/wallets", svc.createWallet).Methods("POST")
	router.HandleFunc("/api/v1/wallets/{id}", svc.getWallet).Methods("GET")
	router.HandleFunc("/api/v1/wallets/{id}", svc.updateWallet).Methods("PUT")
	router.HandleFunc("/api/v1/wallets/{id}", svc.deleteWallet).Methods("DELETE")
	router.HandleFunc("/api/v1/wallets", svc.listWallets).Methods("GET")
	router.HandleFunc("/api/v1/wallets/{id}/balance", svc.getBalance).Methods("GET")

	router.HandleFunc("/health", healthCheck).Methods("GET")

	log.Println("Starting Wallet Service on port 8001")
	log.Fatal(http.ListenAndServe(":8001", router))
}

func (s *WalletService) createWallet(w http.ResponseWriter, r *http.Request) {
	var wallet Wallet
	if err := json.NewDecoder(r.Body).Decode(&wallet); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	wallet.ID = fmt.Sprintf("wallet_%d", time.Now().UnixNano())
	wallet.CreatedAt = time.Now().Unix()
	wallet.LastUsed = time.Now().Unix()

	s.mu.Lock()
	s.wallets[wallet.ID] = &wallet
	s.mu.Unlock()

	json.NewEncoder(w).Encode(wallet)
}

func (s *WalletService) getWallet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	s.mu.RLock()
	defer s.mu.RUnlock()

	if wallet, ok := s.wallets[id]; ok {
		json.NewEncoder(w).Encode(wallet)
	} else {
		http.Error(w, "Wallet not found", http.StatusNotFound)
	}
}

func (s *WalletService) updateWallet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var wallet Wallet
	if err := json.NewDecoder(r.Body).Decode(&wallet); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if existing, ok := s.wallets[id]; ok {
		wallet.ID = existing.ID
		wallet.CreatedAt = existing.CreatedAt
		wallet.LastUsed = time.Now().Unix()
		s.wallets[id] = &wallet
		json.NewEncoder(w).Encode(wallet)
	} else {
		http.Error(w, "Wallet not found", http.StatusNotFound)
	}
	s.mu.Unlock()
}

func (s *WalletService) deleteWallet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.wallets[id]; ok {
		delete(s.wallets, id)
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	} else {
		http.Error(w, "Wallet not found", http.StatusNotFound)
	}
}

func (s *WalletService) listWallets(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	chain := r.URL.Query().Get("chain")

	s.mu.RLock()
	defer s.mu.RUnlock()

	var wallets []*Wallet
	for _, wallet := range s.wallets {
		if (userID == "" || wallet.UserID == userID) && (chain == "" || wallet.Chain == chain) {
			wallets = append(wallets, wallet)
		}
	}

	json.NewEncoder(w).Encode(wallets)
}

func (s *WalletService) getBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	s.mu.RLock()
	defer s.mu.RUnlock()

	if wallet, ok := s.wallets[id]; ok {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"walletId": id,
			"balance": wallet.Balance,
			"lastUpdate": time.Now().Unix(),
		})
	} else {
		http.Error(w, "Wallet not found", http.StatusNotFound)
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "healthy"})
}