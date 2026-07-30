/**
 * TigerWallet Fiat On/Off Ramp - Go API
 * Fiat gateway for buying/selling crypto
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// ============================================================================
// Types
// ============================================================================

type FiatProvider struct {
	ProviderID      string  `json:"provider_id"`
	Name            string  `json:"name"`
	LogoURL         string  `json:"logo_url"`
	SupportedFiat   []string `json:"supported_fiat"`
	SupportedCrypto []string `json:"supported_crypto"`
	BuyEnabled      bool    `json:"buy_enabled"`
	SellEnabled     bool    `json:"sell_enabled"`
	MinBuyAmount    float64 `json:"min_buy_amount"`
	MaxBuyAmount    float64 `json:"max_buy_amount"`
	MinSellAmount   float64 `json:"min_sell_amount"`
	MaxSellAmount   float64 `json:"max_sell_amount"`
	FeePercent      float64 `json:"fee_percent"`
	ProcessingTime  string  `json:"processing_time"`
}

type Order struct {
	OrderID        string    `json:"order_id"`
	UserID         uint64    `json:"user_id"`
	Type           string    `json:"type"` // buy, sell
	ProviderID     string    `json:"provider_id"`
	FiatCurrency   string    `json:"fiat_currency"`
	CryptoCurrency string    `json:"crypto_currency"`
	FiatAmount     float64   `json:"fiat_amount"`
	CryptoAmount   float64   `json:"crypto_amount"`
	ExchangeRate   float64   `json:"exchange_rate"`
	Status         string    `json:"status"` // pending, processing, completed, failed, cancelled
	PaymentMethod  string    `json:"payment_method"`
	WalletAddress  string    `json:"wallet_address"`
	TxHash         *string   `json:"tx_hash,omitempty"`
	CreatedAt      uint64    `json:"created_at"`
	UpdatedAt      uint64    `json:"updated_at"`
	ExpiresAt      uint64    `json:"expires_at"`
}

type ExchangeRate struct {
	FromCurrency string  `json:"from_currency"`
	ToCurrency   string  `json:"to_currency"`
	Rate         float64 `json:"rate"`
	UpdatedAt    uint64  `json:"updated_at"`
}

type FiatService struct {
	providers    map[string]*FiatProvider
	orders       map[string]*Order
	exchangeRates map[string]*ExchangeRate
	nextOrderID  uint64
	mu           sync.RWMutex
}

func NewFiatService() *FiatService {
	svc := &FiatService{
		providers:     make(map[string]*FiatProvider),
		orders:       make(map[string]*Order),
		exchangeRates: make(map[string]*ExchangeRate),
		nextOrderID:  1,
	}

	svc.seedProviders()
	svc.seedRates()
	return svc
}

func (s *FiatService) seedProviders() {
	now := uint64(time.Now().UnixMilli())

	providers := []*FiatProvider{
		{ProviderID: "stripe", Name: "Stripe", LogoURL: "/logos/stripe.png", SupportedFiat: []string{"USD", "EUR", "GBP"}, SupportedCrypto: []string{"ETH", "BTC", "USDT", "USDC"}, BuyEnabled: true, SellEnabled: false, MinBuyAmount: 20, MaxBuyAmount: 25000, FeePercent: 2.5, ProcessingTime: "1-2 days"},
		{ProviderID: "moonpay", Name: "MoonPay", LogoURL: "/logos/moonpay.png", SupportedFiat: []string{"USD", "EUR", "GBP", "AUD"}, SupportedCrypto: []string{"ETH", "BTC", "SOL", "MATIC", "USDC"}, BuyEnabled: true, SellEnabled: true, MinBuyAmount: 30, MaxBuyAmount: 10000, FeePercent: 4.5, ProcessingTime: "30 mins"},
		{ProviderID: "transak", Name: "Transak", LogoURL: "/logos/transak.png", SupportedFiat: []string{"USD", "EUR", "GBP", "INR"}, SupportedCrypto: []string{"ETH", "BTC", "MATIC", "AVAX", "FTM"}, BuyEnabled: true, SellEnabled: false, MinBuyAmount: 50, MaxBuyAmount: 5000, FeePercent: 3.0, ProcessingTime: "1-4 hours"},
		{ProviderID: "banxa", Name: "Banxa", LogoURL: "/logos/banxa.png", SupportedFiat: []string{"USD", "EUR", "GBP", "AUD", "CAD"}, SupportedCrypto: []string{"ETH", "BTC", "SOL", "DOT", "ADA"}, BuyEnabled: true, SellEnabled: true, MinBuyAmount: 100, MaxBuyAmount: 15000, FeePercent: 2.8, ProcessingTime: "2-3 days"},
		{ProviderID: "mercuryo", Name: "Mercuryo", LogoURL: "/logos/mercuryo.png", SupportedFiat: []string{"USD", "EUR", "GBP"}, SupportedCrypto: []string{"ETH", "BTC", "USDT", "USDC", "TRX"}, BuyEnabled: true, SellEnabled: false, MinBuyAmount: 25, MaxBuyAmount: 5000, FeePercent: 3.5, ProcessingTime: "15-30 mins"},
		{ProviderID: "simplex", Name: "Simplex", LogoURL: "/logos/simplex.png", SupportedFiat: []string{"USD", "EUR", "GBP"}, SupportedCrypto: []string{"ETH", "BTC", "USDT", "USDC"}, BuyEnabled: true, SellEnabled: false, MinBuyAmount: 50, MaxBuyAmount: 20000, FeePercent: 3.5, ProcessingTime: "1-2 hours"},
	}

	for _, p := range providers {
		s.providers[p.ProviderID] = p
	}

	log.Printf("Seeded %d fiat providers", len(s.providers))
}

func (s *FiatService) seedRates() {
	now := uint64(time.Now().UnixMilli())

	rates := []*ExchangeRate{
		{FromCurrency: "USD", ToCurrency: "ETH", Rate: 0.00042, UpdatedAt: now},
		{FromCurrency: "USD", ToCurrency: "BTC", Rate: 0.000025, UpdatedAt: now},
		{FromCurrency: "USD", ToCurrency: "USDT", Rate: 1.0, UpdatedAt: now},
		{FromCurrency: "USD", ToCurrency: "USDC", Rate: 1.0, UpdatedAt: now},
		{FromCurrency: "USD", ToCurrency: "SOL", Rate: 0.0085, UpdatedAt: now},
		{FromCurrency: "USD", ToCurrency: "MATIC", Rate: 0.85, UpdatedAt: now},
		{FromCurrency: "EUR", ToCurrency: "ETH", Rate: 0.00045, UpdatedAt: now},
		{FromCurrency: "EUR", ToCurrency: "BTC", Rate: 0.000027, UpdatedAt: now},
		{FromCurrency: "GBP", ToCurrency: "ETH", Rate: 0.00040, UpdatedAt: now},
		{FromCurrency: "GBP", ToCurrency: "BTC", Rate: 0.000024, UpdatedAt: now},
	}

	for _, r := range rates {
		s.exchangeRates[r.FromCurrency+"_"+r.ToCurrency] = r
	}
}

func (s *FiatService) GetProviders() []*FiatProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*FiatProvider, 0, len(s.providers))
	for _, p := range s.providers {
		result = append(result, p)
	}
	return result
}

func (s *FiatService) GetProvider(providerID string) *FiatProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.providers[providerID]
}

func (s *FiatService) GetRate(from, to string) *ExchangeRate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rate, ok := s.exchangeRates[from+"_"+to]
	if !ok {
		// Try reverse
		rate, ok = s.exchangeRates[to+"_"+from]
		if ok && rate.Rate > 0 {
			rev := &ExchangeRate{FromCurrency: from, ToCurrency: to, Rate: 1.0 / rate.Rate, UpdatedAt: rate.UpdatedAt}
			return rev
		}
		return nil
	}
	return rate
}

func (s *FiatService) CreateOrder(userID uint64, orderType, providerID, fiatCurrency, cryptoCurrency, paymentMethod, walletAddress string, fiatAmount float64) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	provider, ok := s.providers[providerID]
	if !ok {
		return nil, fmt.Errorf("provider not found")
	}

	if orderType == "buy" && !provider.BuyEnabled {
		return nil, fmt.Errorf("buy not enabled for this provider")
	}
	if orderType == "sell" && !provider.SellEnabled {
		return nil, fmt.Errorf("sell not enabled for this provider")
	}

	if fiatAmount < provider.MinBuyAmount || fiatAmount > provider.MaxBuyAmount {
		return nil, fmt.Errorf("amount outside allowed range: %v - %v", provider.MinBuyAmount, provider.MaxBuyAmount)
	}

	// Get rate
	rate, ok := s.exchangeRates[fiatCurrency+"_"+cryptoCurrency]
	if !ok {
		return nil, fmt.Errorf("exchange rate not available for %s/%s", fiatCurrency, cryptoCurrency)
	}

	cryptoAmount := fiatAmount / rate.Rate
	fee := fiatAmount * provider.FeePercent / 100

	now := uint64(time.Now().UnixMilli())
	orderID := fmt.Sprintf("FIAT_%s_%d", providerID, s.nextOrderID)
	s.nextOrderID++

	order := &Order{
		OrderID:        orderID,
		UserID:          userID,
		Type:            orderType,
		ProviderID:      providerID,
		FiatCurrency:    fiatCurrency,
		CryptoCurrency:  cryptoCurrency,
		FiatAmount:      fiatAmount,
		CryptoAmount:    cryptoAmount,
		ExchangeRate:    rate.Rate,
		Status:          "pending",
		PaymentMethod:   paymentMethod,
		WalletAddress:   walletAddress,
		CreatedAt:       now,
		UpdatedAt:       now,
		ExpiresAt:       now + 30*60*1000, // 30 minutes
	}

	s.orders[orderID] = order
	return order, nil
}

func (s *FiatService) GetOrder(orderID string) *Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.orders[orderID]
}

func (s *FiatService) GetUserOrders(userID uint64) []*Order {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Order
	for _, o := range s.orders {
		if o.UserID == userID {
			result = append(result, o)
		}
	}
	return result
}

func (s *FiatService) CancelOrder(orderID string, userID uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, ok := s.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found")
	}

	if order.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	if order.Status != "pending" {
		return fmt.Errorf("order cannot be cancelled")
	}

	order.Status = "cancelled"
	order.UpdatedAt = uint64(time.Now().UnixMilli())
	return nil
}

func (s *FiatService) GetSupportedCurrencies() map[string][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string][]string)
	for _, p := range s.providers {
		for _, f := range p.SupportedFiat {
			if result[f] == nil {
				result[f] = []string{}
			}
			for _, c := range p.SupportedCrypto {
				if !contains(result[f], c) {
					result[f] = append(result[f], c)
				}
			}
		}
	}
	return result
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// ============================================================================
// Handlers
// ============================================================================

type Handler struct {
	service *FiatService
}

func NewHandler(svc *FiatService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) GetProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.service.GetProviders()
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": providers})
}

func (h *Handler) GetProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := h.service.GetProvider(vars["id"])
	if provider == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": map[string]string{"code": "NOT_FOUND", "message": "Provider not found"}})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": provider})
}

func (h *Handler) GetRate(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	rate := h.service.GetRate(from, to)
	if rate == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": map[string]string{"code": "RATE_NOT_FOUND", "message": "Exchange rate not available"}})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": rate})
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID := uint64(1)
	if uid := r.Header.Get("X-User-ID"); uid != "" {
		fmt.Sscanf(uid, "%d", &userID)
	}

	var req struct {
		Type           string  `json:"type"`
		ProviderID     string  `json:"provider_id"`
		FiatCurrency   string  `json:"fiat_currency"`
		CryptoCurrency string  `json:"crypto_currency"`
		FiatAmount     float64 `json:"fiat_amount"`
		PaymentMethod  string  `json:"payment_method"`
		WalletAddress  string  `json:"wallet_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": map[string]string{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	order, err := h.service.CreateOrder(userID, req.Type, req.ProviderID, req.FiatCurrency, req.CryptoCurrency, req.PaymentMethod, req.WalletAddress, req.FiatAmount)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": map[string]string{"code": "ORDER_FAILED", "message": err.Error()}})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": order})
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	order := h.service.GetOrder(vars["id"])
	if order == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": map[string]string{"code": "NOT_FOUND", "message": "Order not found"}})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": order})
}

func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID := uint64(1)
	if uid := r.Header.Get("X-User-ID"); uid != "" {
		fmt.Sscanf(uid, "%d", &userID)
	}

	orders := h.service.GetUserOrders(userID)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": orders})
}

func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	userID := uint64(1)
	if uid := r.Header.Get("X-User-ID"); uid != "" {
		fmt.Sscanf(uid, "%d", &userID)
	}

	vars := mux.Vars(r)
	err := h.service.CancelOrder(vars["id"], userID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": map[string]string{"code": "CANCEL_FAILED", "message": err.Error()}})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (h *Handler) GetCurrencies(w http.ResponseWriter, r *http.Request) {
	currencies := h.service.GetSupportedCurrencies()
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": currencies})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting TigerWallet Fiat Ramp API...")

	service := NewFiatService()
	handler := NewHandler(service)

	router := mux.NewRouter()
	router.Use(handlers.ContentTypeHandler(handlers.LoggingHandler(os.Stdout, router), "application/json"))

	router.HandleFunc("/api/v1/providers", handler.GetProviders).Methods("GET")
	router.HandleFunc("/api/v1/providers/{id}", handler.GetProvider).Methods("GET")
	router.HandleFunc("/api/v1/rate", handler.GetRate).Methods("GET")
	router.HandleFunc("/api/v1/currencies", handler.GetCurrencies).Methods("GET")
	router.HandleFunc("/api/v1/orders", handler.CreateOrder).Methods("POST")
	router.HandleFunc("/api/v1/orders", handler.GetOrders).Methods("GET")
	router.HandleFunc("/api/v1/orders/{id}", handler.GetOrder).Methods("GET")
	router.HandleFunc("/api/v1/orders/{id}/cancel", handler.CancelOrder).Methods("POST")

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")

	srv := &http.Server{Addr: ":8446", Handler: router}

	go func() {
		log.Printf("Server listening on :8446")
		srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
