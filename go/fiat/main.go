package main

import (
"encoding/json"
"fmt"
"log"
"net/http"
"sync"
"time"
)

type FiatOrder struct {
ID        string  `json:"id"`
UserID   string  `json:"user_id"`
Amount   float64 `json:"amount"`
Currency string  `json:"currency"`
Method   string  `json:"method"`
Status   string  `json:"status"`
CreatedAt int64  `json:"created_at"`
}

type FiatService struct {
orders map[string]*FiatOrder
mu     sync.RWMutex
}

func NewFiatService() *FiatService {
return &FiatService{orders: make(map[string]*FiatOrder)}
}

func (s *FiatService) Run() error {
mux := http.NewServeMux()
mux.HandleFunc("/order", s.handleOrder)
mux.HandleFunc("/status", s.handleStatus)
mux.HandleFunc("/health", s.handleHealth)

log.Println("Fiat service on :8100")
return http.ListenAndServe(":8100", mux)
}

func (s *FiatService) handleOrder(w http.ResponseWriter, r *http.Request) {
var order FiatOrder
json.NewDecoder(r.Body).Decode(&order)

order.ID = fmt.Sprintf("fiat_%d", time.Now().UnixNano())
order.Status = "pending"
order.CreatedAt = time.Now().UnixMilli()

s.mu.Lock()
s.orders[order.ID] = &order
s.mu.Unlock()

json.NewEncoder(w).Encode(order)
}

func (s *FiatService) handleStatus(w http.ResponseWriter, r *http.Request) {
id := r.URL.Query().Get("id")

s.mu.RLock()
defer s.mu.RUnlock()

if o, ok := s.orders[id]; ok {
.NewEncoder(w).Encode(o)

}

http.Error(w, "Not found", http.StatusNotFound)
}

func (s *FiatService) handleHealth(w http.ResponseWriter, r *http.Request) {
json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
log.Println("Starting Fiat Service...")
NewFiatService().Run()
}
