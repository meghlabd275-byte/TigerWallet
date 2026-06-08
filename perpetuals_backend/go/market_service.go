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

// Ticker represents market ticker data
type Ticker struct {
	Symbol           string `json:"symbol"`
	Price           string `json:"price"`
	PriceChange     string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	High24h        string `json:"high24h"`
	Low24h         string `json:"low24h"`
	Volume24h      string `json:"volume24h"`
	Turnover24h    string `json:"turnover24h"`
	OpenInterest   string `json:"openInterest"`
	FundingRate   string `json:"fundingRate"`
	NextFundingTime int64  `json:"nextFundingTime"`
}

// Depth represents order book depth
type Depth struct {
	Symbol      string     `json:"symbol"`
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids       [][]string `json:"bids"`
	Asks       [][]string `json:"asks"`
}

// Trade represents a trade
type Trade struct {
	ID          string `json:"id"`
	Price      string `json:"price"`
	Quantity  string `json:"quantity"`
	Time       int64  `json:"time"`
	IsBuyerMaker bool  `json:"isBuyerMaker"`
}

// Kline represents kline/candlestick data
type Kline struct {
	OpenTime  int64  `json:"openTime"`
	Open     string `json:"open"`
	High     string `json:"high"`
	Low      string `json:"low"`
	Close    string `json:"close"`
	Volume   string `json:"volume"`
	CloseTime int64  `json:"closeTime"`
}

// FundingInfo represents funding rate information
type FundingInfo struct {
	Symbol           string `json:"symbol"`
	FundingRate     string `json:"fundingRate"`
	FundingRateReal string `json:"fundingRateReal"`
	NextFundingTime int64  `json:"nextFundingTime"`
}

// MarketService handles market data operations
type MarketService struct {
	mu       sync.RWMutex
	tickers  map[string]*Ticker
	depths   map[string]*Depth
	trades   map[string][]*Trade
	klines  map[string]map[string][]*Kline
	prices  map[string]float64
	stopped bool
}

// NewMarketService creates a new market service
func NewMarketService() *MarketService {
	s := &MarketService{
		tickers: make(map[string]*Ticker),
		depths:  make(map[string]*Depth),
		trades:  make(map[string][]*Trade),
		klines:  make(map[string]map[string][]*Kline),
		prices:  make(map[string]float64),
	}

	// Initialize with trading pairs
	symbols := []string{
		"BTC-USD", "ETH-USD", "SOL-USD", "BNB-USD", "XRP-USD",
		"DOGE-USD", "ADA-USD", "AVAX-USD", "DOT-USD", "MATIC-USD",
		"LINK-USD", "UNI-USD", "ATOM-USD", "LTC-USD", "BCH-USD",
		"ETC-USD", "XLM-USD", "ALGO-USD", "VET-USD", "FIL-USD",
		"NEAR-USD", "APT-USD", "ARB-USD", "OP-USD", "AAVE-USD",
		"MKR-USD", "SNX-USD", "LDO-USD", "RPL-USD", "GMX-USD",
		"INJ-USD", "SEI-USD", "TIA-USD", "SUI-USD", "BLUR-USD",
		"JTO-USD", "JUP-USD", "WIF-USD", "BONK-USD", "PEPE-USD",
		"SHIB-USD", "FLOKI-USD", "GALA-USD", "AXS-USD", "MANA-USD",
		"ENJ-USD", "SAND-USD", "CHZ-USD", "THETA-USD", "1INCH-USD",
	}

	for _, symbol := range symbols {
		s.tickers[symbol] = &Ticker{
			Symbol: symbol,
			Price:  "50000",
			High24h: "51000",
			Low24h: "49000",
		}
		s.prices[symbol] = 50000 + rand.Float64()*1000
		s.depths[symbol] = s.generateDepth(symbol)
	}

	return s
}

func (s *MarketService) generateDepth(symbol string) *Depth {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	basePrice := s.prices[symbol]

	bids := make([][]string, 20)
	asks := make([][]string, 20)

	for i := 0; i < 20; i++ {
		bidPrice := basePrice - float64(i)*0.5
		askPrice := basePrice + float64(i)*0.5

		bids[i] = []string{
			fmt.Sprintf("%.2f", bidPrice),
			fmt.Sprintf("%.4f", r.Float64()*10),
		}
		asks[i] = []string{
			fmt.Sprintf("%.2f", askPrice),
			fmt.Sprintf("%.4f", r.Float64()*10),
		}
	}

	return &Depth{
		Symbol:      symbol,
		LastUpdateID: time.Now().Unix(),
		Bids:       bids,
		Asks:       asks,
	}
}

// StartPriceFeed starts the price feed
func (s *MarketService) StartPriceFeed(hub *WSHub) {
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

func (s *MarketService) updatePrices(hub *WSHub) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for symbol, price := range s.prices {
		// Random walk price
		change := (rand.Float64() - 0.5) * 0.1
		s.prices[symbol] = price * (1 + change)

		if ticker, ok := s.tickers[symbol]; ok {
			ticker.Price = fmt.Sprintf("%.2f", s.prices[symbol])
			ticker.PriceChange = fmt.Sprintf("%.2f", change*price)
			ticker.PriceChangePercent = fmt.Sprintf("%.2f", change*100)
			ticker.High24h = fmt.Sprintf("%.2f", s.prices[symbol]*1.02)
			ticker.Low24h = fmt.Sprintf("%.2f", s.prices[symbol]*0.98)
		}

		// Update depth
		s.depths[symbol] = s.generateDepth(symbol)

		// Broadcast to WebSocket clients
		if hub != nil {
			hub.Broadcast(map[string]interface{}{
				"type":  "ticker",
				"data": s.tickers[symbol],
			})
		}
	}
}

// GetTickers gets all tickers
func (s *MarketService) GetTickers(ctx context.Context) ([]*Ticker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tickers []*Ticker
	for _, ticker := range s.tickers {
		tickers = append(tickers, ticker)
	}

	return tickers, nil
}

// GetDepth gets order book depth
func (s *MarketService) GetDepth(ctx context.Context, symbol, limit string) (*Depth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	depth, ok := s.depths[symbol]
	if !ok {
		return nil, fmt.Errorf("symbol not found")
	}

	return depth, nil
}

// GetTrades gets recent trades
func (s *MarketService) GetTrades(ctx context.Context, symbol, limit string) ([]*Trade, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	trades, ok := s.trades[symbol]
	if !ok {
		return nil, nil
	}

	return trades, nil
}

// GetKlines gets kline data
func (s *MarketService) GetKlines(ctx context.Context, symbol, interval, limit string) ([]*Kline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	intervalKlines, ok := s.klines[symbol]
	if !ok {
		// Generate sample klines
		return s.generateKlines(symbol, interval, limit), nil
	}

	return intervalKlines[interval], nil
}

func (s *MarketService) generateKlines(symbol, interval, limit string) []*Kline {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	basePrice := s.prices[symbol]

	limitInt := 100
	if limit != "" {
		fmt.Sscanf(limit, "%d", &limitInt)
	}

	klines := make([]*Kline, limitInt)
	for i := 0; i < limitInt; i++ {
		open := basePrice * (1 + (r.Float64()-0.5)*0.02)
		close := open * (1 + (r.Float64()-0.5)*0.02)
		high := math.Max(open, close) * (1 + r.Float64()*0.01)
		low := math.Min(open, close) * (1 - r.Float64()*0.01)

		klines[i] = &Kline{
			OpenTime:  int64(i) * 3600,
			Open:    fmt.Sprintf("%.2f", open),
			High:    fmt.Sprintf("%.2f", high),
			Low:     fmt.Sprintf("%.2f", low),
			Close:   fmt.Sprintf("%.2f", close),
			Volume:  fmt.Sprintf("%.2f", r.Float64()*1000),
			CloseTime: int64(i+1) * 3600,
		}
	}

	return klines
}

// GetFundingRate gets funding rate
func (s *MarketService) GetFundingRate(ctx context.Context, symbol string) (*FundingInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	fundingRate := (r.Float64() - 0.5) * 0.0001 // -0.01% to +0.01%

	return &FundingInfo{
		Symbol:           symbol,
		FundingRate:     fmt.Sprintf("%.6f", fundingRate),
		FundingRateReal: fmt.Sprintf("%.6f", fundingRate),
		NextFundingTime: time.Now().Add(8*time.Hour).Unix(),
	}, nil
}

// TickerToJSON converts ticker to JSON
func TickerToJSON(ticker *Ticker) string {
	data, _ := json.Marshal(ticker)
	return string(data)
}