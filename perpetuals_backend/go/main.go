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
	"github.com/gorilla/websocket"
)

// Config holds server configuration
type Config struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxHeaderBytes  int
	EnableTLS      bool
	CertFile       string
	KeyFile        string
	DatabaseURL    string
	RedisURL      string
	ClickHouseURL  string
}

// Server is the main perpetuals backend server
type Server struct {
	config         Config
	router        *mux.Router
	httpServer    *http.Server
	wsUpgrader    websocket.Upgrader
	orderService  *OrderService
	positionSvc  *PositionService
	marketSvc    *MarketService
	wsHub       *WSHub
	notification *NotificationService
	settlement  *SettlementService
}

// NewServer creates a new perpetuals backend server
func NewServer(cfg Config) *Server {
	s := &Server{
		config:    cfg,
		router:    mux.NewRouter(),
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // In production, implement proper origin checking
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	s.setupRoutes()
	s.setupMiddlewares()

	s.httpServer = &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        s.router,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		MaxHeaderBytes: cfg.MaxHeaderBytes,
	}

	return s
}

func (s *Server) setupRoutes() {
	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Order endpoints
	api.HandleFunc("/orders", s.createOrder).Methods("POST")
	api.HandleFunc("/orders/{orderID}", s.getOrder).Methods("GET")
	api.HandleFunc("/orders/{orderID}", s.cancelOrder).Methods("DELETE")
	api.HandleFunc("/orders", s.getOrders).Methods("GET")

	// Position endpoints
	api.HandleFunc("/positions", s.getPositions).Methods("GET")
	api.HandleFunc("/positions/{symbol}", s.getPosition).Methods("GET")
	api.HandleFunc("/positions/close", s.closePosition).Methods("POST")

	// Market data endpoints
	api.HandleFunc("/market/tickers", s.getTickers).Methods("GET")
	api.HandleFunc("/market/depth/{symbol}", s.getDepth).Methods("GET")
	api.HandleFunc("/market/trades/{symbol}", s.getTrades).Methods("GET")
	api.HandleFunc("/market/kline/{symbol}", s.getKlines).Methods("GET")
	api.HandleFunc("/market/funding/{symbol}", s.getFundingRate).Methods("GET")

	// Account endpoints
	api.HandleFunc("/account/balance", s.getBalance).Methods("GET")
	api.HandleFunc("/account/margin", s.getMarginInfo).Methods("GET")
	api.HandleFunc("/account/risk", s.getRiskInfo).Methods("GET")

	// WebSocket endpoint
	api.HandleFunc("/ws", s.handleWebSocket)

	// Health check
	s.router.HandleFunc("/health", s.healthCheck).Methods("GET")
}

func (s *Server) setupMiddlewares() {
	s.router.Use(handlers.RecoveryHandler())
	s.router.Use(handlers.Logger())
	s.router.Use(handlers.ContentTypeHandler())
}

// Start starts the server
func (s *Server) Start() error {
	log.Printf("Starting TigerWallet Perpetuals Backend on port %s", s.config.Port)

	// Initialize services
	s.orderService = NewOrderService()
	s.positionSvc = NewPositionService()
	s.marketSvc = NewMarketService()
	s.wsHub = NewWSHub()
	s.notification = NewNotificationService()
	s.settlement = NewSettlementService()

	// Start WebSocket hub
	go s.wsHub.Run()

	// Start market data feed
	go s.marketSvc.StartPriceFeed(s.wsHub)

	// Start funding processing
	go s.settlement.StartFundingProcessor(s.wsHub)

	// Start liquidation watcher
	go s.settlement.StartLiquidationWatcher(s.positionSvc, s.wsHub)

	if s.config.EnableTLS {
		return s.httpServer.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
	}

	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// HTTP Handlers

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	order, err := s.orderService.CreateOrder(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusCreated, order)
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["orderID"]

	order, err := s.orderService.GetOrder(r.Context(), orderID)
	if err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, order)
}

func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["orderID"]

	if err := s.orderService.CancelOrder(r.Context(), orderID); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]string{
		"status": "cancelled",
	})
}

func (s *Server) getOrders(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	symbol := r.URL.Query().Get("symbol")

	orders, err := s.orderService.GetOrders(r.Context(), userID, symbol)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, orders)
}

func (s *Server) getPositions(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")

	positions, err := s.positionSvc.GetPositions(r.Context(), userID)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, positions)
}

func (s *Server) getPosition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	userID := r.URL.Query().Get("userId")

	position, err := s.positionSvc.GetPosition(r.Context(), userID, symbol)
	if err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, position)
}

func (s *Server) closePosition(w http.ResponseWriter, r *http.Request) {
	var req ClosePositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.positionSvc.ClosePosition(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, result)
}

func (s *Server) getTickers(w http.ResponseWriter, r *http.Request) {
	tickers, err := s.marketSvc.GetTickers(r.Context())
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, tickers)
}

func (s *Server) getDepth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	limit := r.URL.Query().Get("limit")

	depth, err := s.marketSvc.GetDepth(r.Context(), symbol, limit)
	if err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, depth)
}

func (s *Server) getTrades(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	limit := r.URL.Query().Get("limit")

	trades, err := s.marketSvc.GetTrades(r.Context(), symbol, limit)
	if err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, trades)
}

func (s *Server) getKlines(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	interval := r.URL.Query().Get("interval")
	limit := r.URL.Query().Get("limit")

	klines, err := s.marketSvc.GetKlines(r.Context(), symbol, interval, limit)
	if err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, klines)
}

func (s *Server) getFundingRate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	rate, err := s.marketSvc.GetFundingRate(r.Context(), symbol)
	if err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, rate)
}

func (s *Server) getBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")

	balance, err := s.orderService.GetBalance(r.Context(), userID)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, balance)
}

func (s *Server) getMarginInfo(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")

	info, err := s.positionSvc.GetMarginInfo(r.Context(), userID)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, info)
}

func (s *Server) getRiskInfo(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")

	info, err := s.orderService.GetRiskInfo(r.Context(), userID)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, info)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := NewWSClient(conn, s.wsHub)
	s.wsHub.Register(client)

	go client.WritePump()
	client.ReadPump()
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
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

// Request/Response types

type CreateOrderRequest struct {
	UserID         string  `json:"userId"`
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"` // BUY, SELL
	OrderType     string  `json:"orderType"` // LIMIT, MARKET, STOP, TAKE_PROFIT
	Price        string  `json:"price"`
	Quantity     string  `json:"quantity"`
	ReduceOnly   bool    `json:"reduceOnly"`
	PostOnly     bool    `json:"postOnly"`
	TimeInForce  string  `json:"timeInForce"` // GTC, IOC, FOK
	StopPrice    string  `json:"stopPrice"`
	Leverage     string  `json:"leverage"`
	MarginType   string  `json:"marginType"` // CROSS, ISOLATED
	PositionSide string  `json:"positionSide"` // LONG, SHORT
}

type ClosePositionRequest struct {
	UserID    string `json:"userId"`
	Symbol    string `json:"symbol"`
	Quantity  string `json:"quantity"`
}

func main() {
	cfg := Config{
		Port:           getEnv("PORT", "8080"),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 10,
		EnableTLS:      getEnv("ENABLE_TLS", "false") == "true",
		CertFile:      getEnv("CERT_FILE", ""),
		KeyFile:       getEnv("KEY_FILE", ""),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://localhost:5432/tigerwallet"),
		RedisURL:      getEnv("REDIS_URL", "localhost:6379"),
		ClickHouseURL: getEnv("CLICKHOUSE_URL", "localhost:9000"),
	}

	server := NewServer(cfg)

	// Wait for interrupt signal
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