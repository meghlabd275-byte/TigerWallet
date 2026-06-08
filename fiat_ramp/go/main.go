package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// KYC Level
const (
	KYCLevelNone    = 0
	KYCLevelBasic   = 1
	KYCLevelMedium = 2
	KYCLevelHigh   = 3
	KYCLevelMax    = 4
)

// Fiat Ramp Server
type FiatRampServer struct {
	config     Config
	router     *mux.Router
	httpServer *http.Server
	kycSvc     *KYCService
	bankingSvc *BankingService
	paymentSvc *PaymentService
	limitsSvc *LimitsService
	complSvc  *ComplianceService
}

// NewFiatRampServer creates a new fiat ramp server
func NewFiatRampServer(cfg Config) *FiatRampServer {
	s := &FiatRampServer{
		config:     cfg,
		router:     mux.NewRouter(),
		kycSvc:     NewKYCService(),
		bankingSvc: NewBankingService(),
		paymentSvc: NewPaymentService(),
		limitsSvc: NewLimitsService(),
		complSvc:  NewComplianceService(),
	}

	s.setupRoutes()
	s.setupMiddlewares()

	s.httpServer = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return s
}

func (s *FiatRampServer) setupRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// KYC endpoints
	api.HandleFunc("/kyc/start", s.startKYC).Methods("POST")
	api.HandleFunc("/kyc/{userID}", s.getKYCStatus).Methods("GET")
	api.HandleFunc("/kyc/{userID}/verify", s.verifyKYC).Methods("POST")
	api.HandleFunc("/kyc/{userID}/upload", s.uploadDocument).Methods("POST")

	// Buy/Sell endpoints
	api.HandleFunc("/buy/quote", s.getBuyQuote).Methods("POST")
	api.HandleFunc("/buy/execute", s.executeBuy).Methods("POST")
	api.HandleFunc("/sell/quote", s.getSellQuote).Methods("POST")
	api.HandleFunc("/sell/execute", s.executeSell).Methods("POST")

	// Payment endpoints
	api.HandleFunc("/payment/methods", s.getPaymentMethods).Methods("GET")
	api.HandleFunc("/payment/create", s.createPayment).Methods("POST")
	api.HandleFunc("/payment/{orderID}", s.getPaymentStatus).Methods("GET")

	// Limits endpoints
	api.HandleFunc("/limits/{userID}", s.getLimits).Methods("GET")
	api.HandleFunc("/limits/{userID}/update", s.updateLimits).Methods("POST")

	// Banking endpoints
	api.HandleFunc("/banking/accounts", s.getBankAccounts).Methods("GET")
	api.HandleFunc("/banking/accounts", s.addBankAccount).Methods("POST")
	api.HandleFunc("/banking/withdraw", s.initiateWithdrawal).Methods("POST")

	// Compliance endpoints
	api.HandleFunc("/compliance/screen", s.screenTransaction).Methods("POST")
	api.HandleFunc("/compliance/report", s.reportTransaction).Methods("POST")

	// Health check
	s.router.HandleFunc("/health", s.healthCheck).Methods("GET")
}

func (s *FiatRampServer) setupMiddlewares() {
	s.router.Use(handlers.RecoveryHandler())
	s.router.Use(handlers.Logger())
}

func (s *FiatRampServer) Start() error {
	log.Printf("Starting TigerWallet Fiat Ramp on port %s", s.config.Port)
	return s.httpServer.ListenAndServe()
}

func (s *FiatRampServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// KYC Handlers

func (s *FiatRampServer) startKYC(w http.ResponseWriter, r *http.Request) {
	var req StartKYCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.kycSvc.StartKYC(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusCreated, result)
}

func (s *FiatRampServer) getKYCStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	status, err := s.kycSvc.GetStatus(r.Context(), userID)
	if err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, status)
}

func (s *FiatRampServer) verifyKYC(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	var req VerifyKYCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.kycSvc.Verify(r.Context(), userID, req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, result)
}

func (s *FiatRampServer) uploadDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	var req UploadDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.kycSvc.UploadDocument(r.Context(), userID, req)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, result)
}

// Buy/Sell Handlers

func (s *FiatRampServer) getBuyQuote(w http.ResponseWriter, r *http.Request) {
	var req QuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	quote, err := s.paymentSvc.GetBuyQuote(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, quote)
}

func (s *FiatRampServer) executeBuy(w http.ResponseWriter, r *http.Request) {
	var req ExecuteBuyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.paymentSvc.ExecuteBuy(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusCreated, result)
}

func (s *FiatRampServer) getSellQuote(w http.ResponseWriter, r *http.Request) {
	var req QuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	quote, err := s.paymentSvc.GetSellQuote(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, quote)
}

func (s *FiatRampServer) executeSell(w http.ResponseWriter, r *http.Request) {
	var req ExecuteSellRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.paymentSvc.ExecuteSell(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusCreated, result)
}

// Payment Handlers

func (s *FiatRampServer) getPaymentMethods(w http.ResponseWriter, r *http.Request) {
	methods, err := s.paymentSvc.GetMethods(r.Context())
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, methods)
}

func (s *FiatRampServer) createPayment(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	payment, err := s.paymentSvc.Create(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusCreated, payment)
}

func (s *FiatRampServer) getPaymentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["orderID"]

	status, err := s.paymentSvc.GetStatus(r.Context(), orderID)
	if err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, status)
}

// Limits Handlers

func (s *FiatRampServer) getLimits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	limits, err := s.limitsSvc.Get(r.Context(), userID)
	if err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, limits)
}

func (s *FiatRampServer) updateLimits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	var req UpdateLimitsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	limits, err := s.limitsSvc.Update(r.Context(), userID, req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, limits)
}

// Banking Handlers

func (s *FiatRampServer) getBankAccounts(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")

	accounts, err := s.bankingSvc.GetAccounts(r.Context(), userID)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, accounts)
}

func (s *FiatRampServer) addBankAccount(w http.ResponseWriter, r *http.Request) {
	var req AddBankAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	account, err := s.bankingSvc.AddAccount(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusCreated, account)
}

func (s *FiatRampServer) initiateWithdrawal(w http.ResponseWriter, r *http.Request) {
	var req WithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.bankingSvc.Withdraw(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusCreated, result)
}

// Compliance Handlers

func (s *FiatRampServer) screenTransaction(w http.ResponseWriter, r *http.Request) {
	var req ScreenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.complSvc.Screen(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, result)
}

func (s *FiatRampServer) reportTransaction(w http.ResponseWriter, r *http.Request) {
	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.complSvc.Report(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, result)
}

func (s *FiatRampServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, r, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

// Helper functions

func WriteJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, message string) {
	WriteJSON(w, r, status, map[string]string{
		"error": message,
	})
}

// Request types

type StartKYCRequest struct {
	UserID     string `json:"userId"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Country   string `json:"country"`
	Level    int    `json:"level"`
}

type VerifyKYCRequest struct {
	Code string `json:"code"`
}

type UploadDocumentRequest struct {
	DocType string `json:"docType"`
	Data   string `json:"data"`
}

type QuoteRequest struct {
	UserID    string  `json:"userId"`
	FiatAmt  float64 `json:"fiatAmt"`
	CryptoAmt float64 `json:"cryptoAmt"`
	Fiat    string  `json:"fiat"`
	Crypto  string  `json:"crypto"`
	Method  string  `json:"method"`
}

type ExecuteBuyRequest struct {
	UserID    string  `json:"userId"`
	QuoteID  string  `json:"quoteId"`
	FiatAmt  float64 `json:"fiatAmt"`
	Crypto  string  `json:"crypto"`
	Method  string  `json:"method"`
	BankAccount string `json:"bankAccount"`
}

type ExecuteSellRequest struct {
	UserID    string  `json:"userId"`
	QuoteID  string  `json:"quoteId"`
	CryptoAmt float64 `json:"cryptoAmt"`
	Crypto  string  `json:"crypto"`
	Method  string  `json:"method"`
	BankAccount string `json:"bankAccount"`
}

type CreatePaymentRequest struct {
	UserID   string `json:"userId"`
	OrderID  string `json:"orderId"`
	Amount  float64 `json:"amount"`
	Currency string `json:"currency"`
	Method  string `json:"method"`
}

type UpdateLimitsRequest struct {
	Level        int     `json:"level"`
	DailyLimit    float64 `json:"dailyLimit"`
	monthlyLimit float64 `json:"monthlyLimit"`
	YearlyLimit float64 `json:"yearlyLimit"`
}

type AddBankAccountRequest struct {
	UserID     string `json:"userId"`
	BankName   string `json:"bankName"`
	AccountNum string `json:"accountNum"`
	RoutingNum string `json:"routingNum"`
	Country   string `json:"country"`
	Currency  string `json:"currency"`
}

type WithdrawalRequest struct {
	UserID    string  `json:"userId"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	BankAccount string `json:"bankAccount"`
}

type ScreenRequest struct {
	UserID   string `json:"userId"`
	Amount  float64 `json:"amount"`
	Type    string `json:"type"`
	Country string `json:"country"`
}

type ReportRequest struct {
	UserID   string `json:"userId"`
	Type     string `json:"type"`
	Details string `json:"details"`
}

type Config struct {
	Port         string
	DatabaseURL string
	RedisURL    string
}

func main() {
	cfg := Config{
		Port:         getEnv("PORT", "8083"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://localhost:5432/tigerwallet"),
		RedisURL:     getEnv("REDIS_URL", "localhost:6379"),
	}

	server := NewFiatRampServer(cfg)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	if err := server.Stop(); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}