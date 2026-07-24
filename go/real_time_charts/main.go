/**
 * TigerWallet Real-Time Charts Service
 * 
 * High-performance WebSocket service for live cryptocurrency charts
 * Uses Go for high load handling and worldwide distribution
 * 
 * Features:
 * - Sub-second chart updates
 * - Multiple timeframes (1m, 5m, 15m, 1h, 4h, 1d)
 * - Technical indicators (SMA, EMA, RSI, MACD, Bollinger)
 * - Real-time order book visualization
 * - Volume analysis
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ============== Data Structures ==============

type Candle struct {
	Timestamp  int64   `json:"timestamp"`
	Open       float64 `json:"open"`
	High       float64 `json:"high"`
	Low        float64 `json:"low"`
	Close      float64 `json:"close"`
	Volume     float64 `json:"volume"`
	Trades     int     `json:"trades"`
}

type Ticker struct {
	Symbol        string  `json:"symbol"`
	Price         float64 `json:"price"`
	Change24h     float64 `json:"change_24h"`
	ChangePercent float64 `json:"change_percent"`
	High24h       float64 `json:"high_24h"`
	Low24h        float64 `json:"low_24h"`
	Volume24h     float64 `json:"volume_24h"`
	Timestamp     int64   `json:"timestamp"`
}

type OrderBookLevel struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
	Total  float64 `json:"total"`
}

type OrderBook struct {
	Symbol    string           `json:"symbol"`
	Bids      []OrderBookLevel `json:"bids"`
	Asks      []OrderBookLevel `json:"asks"`
	Timestamp int64            `json:"timestamp"`
}

type ChartRequest struct {
	Symbol    string `json:"symbol"`
	Timeframe string `json:"timeframe"` // 1m, 5m, 15m, 1h, 4h, 1d
	Limit     int    `json:"limit"`
}

type ChartResponse struct {
	Symbol     string    `json:"symbol"`
	Timeframe  string    `json:"timeframe"`
	Candles    []Candle  `json:"candles"`
	Indicators Indicators `json:"indicators"`
}

type Indicators struct {
	SMA20       []float64 `json:"sma_20"`
	SMA50       []float64 `json:"sma_50"`
	EMA20       []float64 `json:"ema_20"`
	RSI         []float64 `json:"rsi"`
	MACD        MACD      `json:"macd"`
	Bollinger   Bollinger `json:"bollinger"`
}

type MACD struct {
	MACDLine  []float64 `json:"macd_line"`
	Signal    []float64 `json:"signal"`
	Histogram []float64 `json:"histogram"`
}

type Bollinger struct {
	Upper []float64 `json:"upper"`
	Middle []float64 `json:"middle"`
	Lower []float64 `json:"lower"`
}

type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// ============== Service ==============

type ChartService struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan WSMessage
	register  chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex

	// Market data storage
	tickers   map[string]*Ticker
	candles   map[string]map[string][]Candle // symbol -> timeframe -> candles
	orderBooks map[string]*OrderBook

	// Price aggregation (multiple sources)
	priceSources []string

	// WebSocket upgrader
	upgrader websocket.Upgrader
}

var (
	supportedTimeframes = []string{"1m", "5m", "15m", "1h", "4h", "1d"}
	supportedSymbols   = []string{
		"ETH/USDT", "BTC/USDT", "BNB/USDT", "SOL/USDT", "XRP/USDT",
		"ADA/USDT", "DOGE/USDT", "MATIC/USDT", "DOT/USDT", "LTC/USDT",
		"AVAX/USDT", "LINK/USDT", "ATOM/USDT", "UNI/USDT", "XLM/USDT",
	}
)

func NewChartService() *ChartService {
	return &ChartService{
		clients:      make(map[*websocket.Conn]bool),
		broadcast:    make(chan WSMessage, 256),
		register:     make(chan *websocket.Conn),
		unregister:   make(chan *websocket.Conn),
		tickers:      make(map[string]*Ticker),
		candles:      make(map[string]map[string][]Candle),
		orderBooks:   make(map[string]*OrderBook),
		priceSources: []string{"coinbase", "binance", "kraken"},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for demo
			},
		},
	}
}

func (s *ChartService) Run() {
	// Start data collection
	go s.collectMarketData()

	// Start WebSocket hub
	go s.runHub()

	// Start HTTP server
	http.HandleFunc("/ws", s.handleWebSocket)
	http.HandleFunc("/api/charts", s.handleChartsAPI)
	http.HandleFunc("/api/ticker", s.handleTickerAPI)
	http.HandleFunc("/api/orderbook", s.handleOrderBookAPI)
	http.HandleFunc("/api/indicators", s.handleIndicatorsAPI)
	http.HandleFunc("/health", s.handleHealth)

	log.Println("Chart service starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *ChartService) runHub() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(s.clients))

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				client.Close()
			}
			s.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(s.clients))

		case message := <-s.broadcast:
			s.mu.RLock()
			for client := range s.clients {
				err := client.WriteJSON(message)
				if err != nil {
					client.Close()
					delete(s.clients, client)
				}
			}
			s.mu.RUnlock()
		}
	}
}

func (s *ChartService) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	s.register <- conn

	// Handle incoming messages
	go func() {
		defer func() {
			s.unregister <- conn
			conn.Close()
		}()

		for {
			var msg WSMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				break
			}
			s.handleWSMessage(conn, msg)
		}
	}()

	// Send initial data
	go s.sendInitialData(conn)
}

func (s *ChartService) handleWSMessage(conn *websocket.Conn, msg WSMessage) {
	switch msg.Type {
	case "subscribe":
		if symbols, ok := msg.Payload.(map[string]interface{})["symbols"]; ok {
			log.Printf("Client subscribed to: %v", symbols)
		}
	case "unsubscribe":
		log.Printf("Client unsubscribed")
	case "ping":
		conn.WriteJSON(WSMessage{Type: "pong"})
	}
}

func (s *ChartService) sendInitialData(conn *websocket.Conn) {
	// Send all tickers
	for _, symbol := range supportedSymbols {
		if ticker, ok := s.tickers[symbol]; ok {
			conn.WriteJSON(WSMessage{
				Type:    "ticker",
				Payload: ticker,
			})
		}
	}

	// Send order books
	for symbol, ob := range s.orderBooks {
		conn.WriteJSON(WSMessage{
			Type:    "orderbook",
			Payload: ob,
		})
	}
}

func (s *ChartService) collectMarketData() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		s.updateTickers()
		s.updateCandles()
		s.updateOrderBooks()
		s.broadcastUpdates()
	}
}

func (s *ChartService) updateTickers() {
	for _, symbol := range supportedSymbols {
		// Simulate price updates (replace with real API calls)
		price := getSimulatedPrice(symbol)
		change := getSimulatedChange()

		ticker := &Ticker{
			Symbol:        symbol,
			Price:         price,
			Change24h:     price * change,
			ChangePercent: change * 100,
			High24h:       price * 1.05,
			Low24h:        price * 0.95,
			Volume24h:     getSimulatedVolume(symbol),
			Timestamp:     time.Now().UnixMilli(),
		}
		s.tickers[symbol] = ticker
	}
}

func (s *ChartService) updateCandles() {
	for _, symbol := range supportedSymbols {
		if s.candles[symbol] == nil {
			s.candles[symbol] = make(map[string][]Candle)
		}

		for _, tf := range supportedTimeframes {
			candle := generateCandle(symbol, tf)
			s.candles[symbol][tf] = append(s.candles[symbol][tf], candle)
			if len(s.candles[symbol][tf]) > 1000 {
				s.candles[symbol][tf] = s.candles[symbol][tf][1:]
			}
		}
	}
}

func (s *ChartService) updateOrderBooks() {
	for _, symbol := range supportedSymbols {
		price := getSimulatedPrice(symbol)
		bids := generateOrderBookLevels(price, "bid", 15)
		asks := generateOrderBookLevels(price, "ask", 15)

		s.orderBooks[symbol] = &OrderBook{
			Symbol:    symbol,
			Bids:      bids,
			Asks:      asks,
			Timestamp: time.Now().UnixMilli(),
		}
	}
}

func (s *ChartService) broadcastUpdates() {
	for _, symbol := range supportedSymbols {
		if ticker, ok := s.tickers[symbol]; ok {
			s.broadcast <- WSMessage{
				Type:    "ticker",
				Payload: ticker,
			}
		}

		if ob, ok := s.orderBooks[symbol]; ok {
			s.broadcast <- WSMessage{
				Type:    "orderbook",
				Payload: ob,
			}
		}
	}
}

// ============== HTTP Handlers ==============

func (s *ChartService) handleChartsAPI(w http.ResponseWriter, r *http.Request) {
	var req ChartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	candles := s.candles[req.Symbol][req.Timeframe]
	if len(candles) > req.Limit {
		candles = candles[len(candles)-req.Limit:]
	}

	indicators := s.calculateIndicators(candles)

	resp := ChartResponse{
		Symbol:     req.Symbol,
		Timeframe:  req.Timeframe,
		Candles:    candles,
		Indicators: indicators,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *ChartService) handleTickerAPI(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	
	if symbol != "" {
		if ticker, ok := s.tickers[symbol]; ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ticker)
			return
		}
	}

	// Return all tickers
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.tickers)
}

func (s *ChartService) handleOrderBookAPI(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	
	if ob, ok := s.orderBooks[symbol]; ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ob)
		return
	}
	
	http.Error(w, "Symbol not found", http.StatusNotFound)
}

func (s *ChartService) handleIndicatorsAPI(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "1h"
	}

	candles := s.candles[symbol][timeframe]
	indicators := s.calculateIndicators(candles)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(indicators)
}

func (s *ChartService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"clients":    len(s.clients),
		"symbols":    len(s.tickers),
		"timestamp":  time.Now().Unix(),
	})
}

// ============== Technical Indicators ==============

func (s *ChartService) calculateIndicators(candles []Candle) Indicators {
	if len(candles) < 50 {
		return Indicators{}
	}

	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}

	return Indicators{
		SMA20:     calculateSMA(closes, 20),
		SMA50:     calculateSMA(closes, 50),
		EMA20:     calculateEMA(closes, 20),
		RSI:       calculateRSI(closes, 14),
		MACD:      calculateMACD(closes),
		Bollinger: calculateBollinger(closes, 20, 2),
	}
}

func calculateSMA(data []float64, period int) []float64 {
	if len(data) < period {
		return nil
	}

	sma := make([]float64, len(data)-period+1)
	for i := period - 1; i < len(data); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += data[j]
		}
		sma[i-period+1] = sum / float64(period)
	}
	return sma
}

func calculateEMA(data []float64, period int) []float64 {
	if len(data) < period {
		return nil
	}

	ema := make([]float64, len(data))
	multiplier := 2.0 / float64(period+1)

	// First EMA is SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += data[i]
	}
	ema[period-1] = sum / float64(period)

	// Calculate rest
	for i := period; i < len(data); i++ {
		ema[i] = (data[i] - ema[i-1])*multiplier + ema[i-1]
	}
	return ema
}

func calculateRSI(data []float64, period int) []float64 {
	if len(data) < period+1 {
		return nil
	}

	rsi := make([]float64, len(data)-period)
	var gains, losses float64

	for i := 1; i <= period; i++ {
		change := data[i] - data[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	if avgLoss == 0 {
		rsi[0] = 100
	} else {
		rs := avgGain / avgLoss
		rsi[0] = 100 - (100 / (1 + rs))
	}

	for i := period + 1; i < len(data); i++ {
		change := data[i] - data[i-1]
		gain, loss := change, 0.0
		if change < 0 {
			loss = -change
			gain = 0
		}

		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)

		if avgLoss == 0 {
			rsi[i-period] = 100
		} else {
			rs := avgGain / avgLoss
			rsi[i-period] = 100 - (100 / (1 + rs))
		}
	}

	return rsi
}

func calculateMACD(data []float64) MACD {
	ema12 := calculateEMA(data, 12)
	ema26 := calculateEMA(data, 26)

	if ema12 == nil || ema26 == nil {
		return MACD{}
	}

	macdLine := make([]float64, len(ema12))
	for i := range ema12 {
		macdLine[i] = ema12[i] - ema26[i]
	}

	signal := calculateEMA(macdLine, 9)
	histogram := make([]float64, len(macdLine))
	for i := range macdLine {
		if i < len(signal) {
			histogram[i] = macdLine[i] - signal[i]
		}
	}

	return MACD{
		MACDLine:  macdLine,
		Signal:    signal,
		Histogram: histogram,
	}
}

func calculateBollinger(data []float64, period int, stdDev float64) Bollinger {
	sma := calculateSMA(data, period)
	if sma == nil {
		return Bollinger{}
	}

	upper := make([]float64, len(sma))
	lower := make([]float64, len(sma))

	for i := 0; i < len(sma); i++ {
		start := i
		end := i + period
		if end > len(data) {
			break
		}

		sum := 0.0
		for j := start; j < end; j++ {
			sum += (data[j] - sma[i]) * (data[j] - sma[i])
		}
		std := stdDev * sqrt(sum/float64(period))

		upper[i] = sma[i] + std
		lower[i] = sma[i] - std
	}

	return Bollinger{
		Upper:  upper,
		Middle: sma,
		Lower:  lower,
	}
}

// ============== Helpers ==============

func getSimulatedPrice(symbol string) float64 {
	prices := map[string]float64{
		"ETH/USDT":  3500.0,
		"BTC/USDT":  65000.0,
		"BNB/USDT":  600.0,
		"SOL/USDT":  145.0,
		"XRP/USDT":  0.55,
	}
	if price, ok := prices[symbol]; ok {
		return price + (randFloat64() - 0.5) * price * 0.01
	}
	return 100.0
}

func getSimulatedChange() float64 {
	return (randFloat64() - 0.5) * 0.1
}

func getSimulatedVolume(symbol string) float64 {
	volumes := map[string]float64{
		"ETH/USDT":  1500000000,
		"BTC/USDT":  35000000000,
		"BNB/USDT":  1200000000,
	}
	return volumes[symbol]
}

func generateCandle(symbol, timeframe string) Candle {
	price := getSimulatedPrice(symbol)
	volatility := 0.02

	open := price
	change := (randFloat64() - 0.5) * 2 * volatility * price
	close := open + change
	high := max(open, close) + randFloat64() * volatility * price
	low := min(open, close) - randFloat64() * volatility * price

	duration, _ := time.ParseDuration(timeframe + "m")
	if timeframe == "1h" || timeframe == "4h" || timeframe == "1d" {
		duration, _ = time.ParseDuration(timeframe)
	}

	return Candle{
		Timestamp: time.Now().Unix() - int64(duration.Seconds()),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    getSimulatedVolume(symbol) * randFloat64(),
		Trades:    int(randFloat64() * 10000),
	}
}

func generateOrderBookLevels(price float64, side string, count int) []OrderBookLevel {
	levels := make([]OrderBookLevel, count)
	spread := price * 0.001

	for i := 0; i < count; i++ {
		var lvlPrice float64
		if side == "bid" {
			lvlPrice = price - spread - (float64(i) * price * 0.0005)
		} else {
			lvlPrice = price + spread + (float64(i) * price * 0.0005)
		}

		amount := randFloat64() * 10
		levels[i] = OrderBookLevel{
			Price:  lvlPrice,
			Amount: amount,
			Total:  lvlPrice * amount,
		}
	}
	return levels
}

func sqrt(x float64) float64 {
	return x / 2
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

var (
	lastRand float64
)

func randFloat64() float64 {
	// Simple pseudo-random for demo
	lastRand = (lastRand*1103515245 + 12345) / 2147483648
	if lastRand < 0 {
		lastRand = -lastRand
	}
	if lastRand > 1 {
		lastRand = lastRand - float64(int(lastRand))
	}
	return lastRand
}

// ============== Main ==============

func main() {
	ctx := context.Background()
	log.Println("Starting TigerWallet Real-Time Charts Service...")
	
	service := NewChartService()
	service.Run()
	
	<-ctx.Done()
}
