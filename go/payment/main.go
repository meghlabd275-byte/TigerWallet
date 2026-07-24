/**
 * TigerWallet Payment Service
 * High-Load Distributed Go Implementation
 */

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Payment struct {
	ID          string  `json:"id"`
	UserID     string  `json:"user_id"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Method     string  `json:"method"`
	Status     string  `json:"status"`
	Provider   string  `json:"provider"`
	ExternalID string  `json:"external_id"`
	CreatedAt  int64   `json:"created_at"`
}

type FiatPayment struct {
	Payment
	BankCode    string `json:"bank_code"`
	AccountNum  string `json:"account_num"`
	Recipient   string `json:"recipient"`
}

type CryptoPayment struct {
	Payment
	Chain       string `json:"chain"`
	Token       string `json:"token"`
	FromAddr    string `json:"from_address"`
	ToAddr      string `json:"to_address"`
	TxHash      string `json:"tx_hash"`
}

type PaymentService struct {
	fiatPayments  map[string]*FiatPayment
	cryptoPayments map[string]*CryptoPayment
	mu            sync.RWMutex
}

func NewPaymentService() *PaymentService {
	return &PaymentService{
		fiatPayments:  make(map[string]*FiatPayment),
		cryptoPayments: make(map[string]*CryptoPayment),
	}
}

func (s *PaymentService) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/fiat/deposit", s.handleFiatDeposit)
	mux.HandleFunc("/fiat/withdraw", s.handleFiatWithdraw)
	mux.HandleFunc("/crypto/deposit", s.handleCryptoDeposit)
	mux.HandleFunc("/crypto/withdraw", s.handleCryptoWithdraw)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/health", s.handleHealth)

	log.Println("Payment service starting on :8096")
	return http.ListenAndServe(":8096", mux)
}

func (s *PaymentService) handleFiatDeposit(w http.ResponseWriter, r *http.Request) {
	var payment FiatPayment
	json.NewDecoder(r.Body).Decode(&payment)
	
	payment.ID = fmt.Sprintf("fiat_%d", time.Now().UnixNano())
	payment.Status = "pending"
	payment.Method = "bank_transfer"
	payment.CreatedAt = time.Now().UnixMilli()
	
	s.mu.Lock()
	s.fiatPayments[payment.ID] = &payment
	s.mu.Unlock()
	
	// Simulate processing
	go func() {
		time.Sleep(2 * time.Second)
		s.mu.Lock()
		if p, ok := s.fiatPayments[payment.ID]; ok {
			p.Status = "completed"
		}
		s.mu.Unlock()
	}()
	
	json.NewEncoder(w).Encode(payment)
}

func (s *PaymentService) handleFiatWithdraw(w http.ResponseWriter, r *http.Request) {
	var payment FiatPayment
	json.NewDecoder(r.Body).Decode(&payment)
	
	payment.ID = fmt.Sprintf("fiat_%d", time.Now().UnixNano())
	payment.Status = "processing"
	payment.Method = "bank_transfer"
	payment.CreatedAt = time.Now().UnixMilli()
	
	s.mu.Lock()
	s.fiatPayments[payment.ID] = &payment
	s.mu.Unlock()
	
	json.NewEncoder(w).Encode(payment)
}

func (s *PaymentService) handleCryptoDeposit(w http.ResponseWriter, r *http.Request) {
	var payment CryptoPayment
	json.NewDecoder(r.Body).Decode(&payment)
	
	payment.ID = fmt.Sprintf("crypto_%d", time.Now().UnixNano())
	payment.Status = "pending"
	payment.CreatedAt = time.Now().UnixMilli()
	
	s.mu.Lock()
	s.cryptoPayments[payment.ID] = &payment
	s.mu.Unlock()
	
	json.NewEncoder(w).Encode(payment)
}

func (s *PaymentService) handleCryptoWithdraw(w http.ResponseWriter, r *http.Request) {
	var payment CryptoPayment
	json.NewDecoder(r.Body).Decode(&payment)
	
	payment.ID = fmt.Sprintf("crypto_%d", time.Now().UnixNano())
	payment.Status = "broadcasting"
	payment.TxHash = fmt.Sprintf("0x%x", time.Now().UnixNano())
	payment.CreatedAt = time.Now().UnixMilli()
	
	s.mu.Lock()
	s.cryptoPayments[payment.ID] = &payment
	s.mu.Unlock()
	
	json.NewEncoder(w).Encode(payment)
}

func (s *PaymentService) handleStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if p, ok := s.fiatPayments[id]; ok {
		json.NewEncoder(w).Encode(p)
		return
	}
	if p, ok := s.cryptoPayments[id]; ok {
		json.NewEncoder(w).Encode(p)
		return
	}
	
	http.Error(w, "Payment not found", http.StatusNotFound)
}

func (s *PaymentService) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"fiat_payments": len(s.fiatPayments),
		"crypto_payments": len(s.cryptoPayments),
	})
}

func main() {
	log.Println("Starting TigerWallet Payment Service...")
	NewPaymentService().Run()
}
