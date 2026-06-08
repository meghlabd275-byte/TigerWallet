package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Comprehensive Market Data Service with 50+ trading pairs

type MarketDataService struct {
	mu       sync.RWMutex
	tickers  map[string]*MarketTicker
	depths   map[string]*OrderBookDepth
	trades   map[string][]*Trade
	klines   map[string]map[string][]*Kline
	prices   map[string]float64
	stopped  bool
}

// Market Ticker with full data
type MarketTicker struct {
	Symbol            string  `json:"symbol"`
	Price            string  `json:"price"`
	PriceChange      string  `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	High24h          string  `json:"high24h"`
	Low24h           string  `json:"low24h"`
	Volume24h        string  `json:"volume24h"`
	Turnover24h      string  `json:"turnover24h"`
	OpenInterest     string  `json:"openInterest"`
	FundingRate     string  `json:"fundingRate"`
	NextFundingTime int64   `json:"nextFundingTime"`
	MarkPrice        string  `json:"markPrice"`
	IndexPrice       string  `json:"indexPrice"`
	LastUpdateTime   int64   `json:"lastUpdateTime"`
}

// Order Book Depth
type OrderBookDepth struct {
	Symbol      string       `json:"symbol"`
	LastUpdateID int64        `json:"lastUpdateId"`
	Bids        []PriceLevel `json:"bids"`
	Asks        []PriceLevel `json:"asks"`
}

type PriceLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
	Orders   int    `json:"orders"`
}

// Initialize with 50+ trading pairs
func NewMarketDataService() *MarketDataService {
	s := &MarketDataService{
		tickers: make(map[string]*MarketTicker),
		depths:  make(map[string]*OrderBookDepth),
		trades:  make(map[string][]*Trade),
		klines:  make(map[string]map[string][]*Kline),
		prices:  make(map[string]float64),
	}

	// Initialize all trading pairs with realistic prices
	pairs := map[string]float64{
		// Crypto Majors
		"BTC-USD": 67500.0, "ETH-USD": 3450.0, "BNB-USD": 605.0, 
		"XRP-USD": 0.62, "SOL-USD": 145.0, "ADA-USD": 0.45,
		"DOGE-USD": 0.12, "DOT-USD": 7.20,
		
		// DeFi
		"UNI-USD": 11.50, "AAVE-USD": 95.0, "MKR-USD": 1850.0,
		"SNX-USD": 3.20, "LDO-USD": 2.85, "RPL-USD": 35.0,
		"GMX-USD": 42.0, "CRV-USD": 0.35, "COMP-USD": 52.0,
		
		// Layer 1
		"AVAX-USD": 38.0, "MATIC-USD": 0.72, "LINK-USD": 14.50,
		"ATOM-USD": 9.80, "LTC-USD": 85.0, "BCH-USD": 480.0,
		"ALGO-USD": 0.18, "VET-USD": 0.025, "FIL-USD": 5.20,
		"NEAR-USD": 5.80, "APT-USD": 9.50, "ARB-USD": 1.15,
		"OP-USD": 2.10, "SUI-USD": 1.05, "TIA-USD": 16.0,
		"INJ-USD": 25.0, "SEI-USD": 0.55,
		
		// Memecoins
		"PEPE-USD": 0.0000012, "SHIB-USD": 0.000012, "WIF-USD": 2.35,
		"BONK-USD": 0.000025, "FLOKI-USD": 0.00012,
		
		// Gaming/NFT
		"GALA-USD": 0.045, "AXS-USD": 7.20, "MANA-USD": 0.42,
		"ENJ-USD": 0.28, "SAND-USD": 0.38, "CHZ-USD": 0.085,
		"THETA-USD": 1.05, "1INCH-USD": 0.28,
		
		// Stablecoins
		"USDC-USD": 1.00, "USDT-USD": 1.00, "DAI-USD": 1.00,
		
		// Exotics
		"XLM-USD": 0.11, "ETC-USD": 26.0, "XMR-USD": 165.0,
		"ZEC-USD": 45.0, "HBAR-USD": 0.065, "FTM-USD": 0.65,
	}

	for symbol, price := range pairs {
		s.prices[symbol] = price
		
		s.tickers[symbol] = &MarketTicker{
			Symbol:            symbol,
			Price:            fmt.Sprintf("%.8f", price),
			PriceChange:      "0",
			PriceChangePercent: "0",
			High24h:          fmt.Sprintf("%.8f", price*1.02),
			Low24h:           fmt.Sprintf("%.8f", price*0.98),
			Volume24h:        fmt.Sprintf("%.0f", rand.Float64()*100000000),
			Turnover24h:      fmt.Sprintf("%.0f", price*rand.Float64()*10000000),
			OpenInterest:     fmt.Sprintf("%.0f", price*rand.Float64()*50000000),
			FundingRate:      fmt.Sprintf("%.6f", (rand.Float64()-0.5)*0.0001),
			NextFundingTime:  time.Now().Add(8*time.Hour).Unix(),
			MarkPrice:        fmt.Sprintf("%.8f", price),
			IndexPrice:       fmt.Sprintf("%.8f", price),
			LastUpdateTime:   time.Now().Unix(),
		}
		
		s.depths[symbol] = s.generateDepth(symbol, price)
	}
	
	return s
}

func (s *MarketDataService) generateDepth(symbol string, basePrice float64) *OrderBookDepth {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	bids := make([]PriceLevel, 20)
	asks := make([]PriceLevel, 20)
	
	for i := 0; i < 20; i++ {
		bidPrice := basePrice * (1 - float64(i)*0.0005 - r.Float64()*0.0002)
		askPrice := basePrice * (1 + float64(i)*0.0005 + r.Float64()*0.0002)
		
		bids[i] = PriceLevel{
			Price:    fmt.Sprintf("%.8f", bidPrice),
			Quantity: fmt.Sprintf("%.8f", r.Float64()*10+0.1),
			Orders:   rand.Intn(10) + 1,
		}
		
		asks[i] = PriceLevel{
			Price:    fmt.Sprintf("%.8f", askPrice),
			Quantity: fmt.Sprintf("%.8f", r.Float64()*10+0.1),
			Orders:   rand.Intn(10) + 1,
		}
	}
	
	return &OrderBookDepth{
		Symbol:      symbol,
		LastUpdateID: time.Now().UnixNano(),
		Bids:       bids,
		Asks:       asks,
	}
}

// Get all tickers
func (s *MarketDataService) GetAllTickers(ctx context.Context) ([]*MarketTicker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var tickers []*MarketTicker
	for _, ticker := range s.tickers {
		tickers = append(tickers, ticker)
	}
	
	return tickers, nil
}

// Get ticker by symbol
func (s *MarketDataService) GetTicker(ctx context.Context, symbol string) (*MarketTicker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	ticker, ok := s.tickers[symbol]
	if !ok {
		return nil, fmt.Errorf("symbol not found: %s", symbol)
	}
	
	return ticker, nil
}

// Get order book depth
func (s *MarketDataService) GetDepth(ctx context.Context, symbol string, limit int) (*OrderBookDepth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	depth, ok := s.depths[symbol]
	if !ok {
		return nil, fmt.Errorf("symbol not found: %s", symbol)
	}
	
	limitedDepth := *depth
	if limit > 0 && len(limitedDepth.Bids) > limit {
		limitedDepth.Bids = limitedDepth.Bids[:limit]
		limitedDepth.Asks = limitedDepth.Asks[:limit]
	}
	
	return &limitedDepth, nil
}

// Get klines
func (s *MarketDataService) GetKlines(ctx context.Context, symbol, interval, limit string) ([]*Kline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if _, ok := s.klines[symbol]; !ok {
		s.klines[symbol] = make(map[string][]*Kline)
	}
	
	if _, ok := s.klines[symbol][interval]; !ok {
		s.klines[symbol][interval] = s.generateKlines(symbol, interval, 100)
	}
	
	return s.klines[symbol][interval], nil
}

func (s *MarketDataService) generateKlines(symbol, interval string, count int) []*Kline {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	basePrice := s.prices[symbol]
	
	intervals := map[string]int{
		"1m": 60, "5m": 300, "15m": 900, "1h": 3600, 
		"4h": 14400, "1d": 86400,
	}
	
	intervalSec := intervals[interval]
	klines := make([]*Kline, count)
	
	for i := 0; i < count; i++ {
		open := basePrice * (1 + (r.Float64()-0.5)*0.04)
		close := open * (1 + (r.Float64()-0.5)*0.04)
		high := math.Max(open, close) * (1 + r.Float64()*0.02)
		low := math.Min(open, close) * (1 - r.Float64()*0.02)
		
		klines[i] = &Kline{
			OpenTime:  int64(i) * int64(intervalSec),
			Open:     fmt.Sprintf("%.8f", open),
			High:     fmt.Sprintf("%.8f", high),
			Low:      fmt.Sprintf("%.8f", low),
			Close:    fmt.Sprintf("%.8f", close),
			Volume:   fmt.Sprintf("%.0f", r.Float64()*1000000),
			CloseTime: int64(i+1) * int64(intervalSec),
		}
	}
	
	return klines
}

// Start price feed updates
func (s *MarketDataService) StartPriceFeed(hub *WSHub) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if s.stopped {
				return
			}
			s.updatePrices(hub)
		}
	}
}

func (s *MarketDataService) updatePrices(hub *WSHub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for symbol, price := range s.prices {
		change := (rand.Float64() - 0.5) * 0.002
		newPrice := price * (1 + change)
		s.prices[symbol] = newPrice
		
		if ticker, ok := s.tickers[symbol]; ok {
			priceChange := newPrice - price
			priceChangePercent := (priceChange / price) * 100
			
			ticker.Price = fmt.Sprintf("%.8f", newPrice)
			ticker.PriceChange = fmt.Sprintf("%.8f", priceChange)
			ticker.PriceChangePercent = fmt.Sprintf("%.4f", priceChangePercent)
			ticker.MarkPrice = fmt.Sprintf("%.8f", newPrice)
			ticker.LastUpdateTime = time.Now().Unix()
		}
		
		s.depths[symbol] = s.generateDepth(symbol, s.prices[symbol])
		
		if hub != nil {
			hub.Broadcast(map[string]interface{}{
				"type":  "ticker",
				"data": s.tickers[symbol],
			})
		}
	}
}

// Get symbols
func (s *MarketDataService) GetSymbols() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var symbols []string
	for symbol := range s.tickers {
		symbols = append(symbols, symbol)
	}
	return symbols
}