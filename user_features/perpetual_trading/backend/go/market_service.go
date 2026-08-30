package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Comprehensive Market Data Service with 50+ trading pairs

type MarketDataService struct {
	mu          sync.RWMutex
	tickers     map[string]*MarketTicker
	depths      map[string]*OrderBookDepth
	trades      map[string][]*Trade
	klines      map[string]map[string][]*Kline
	pricePoints map[string][]pricePoint
	prices      map[string]float64
	stopped     bool
}

// Market Ticker with full data
type MarketTicker struct {
	Symbol             string `json:"symbol"`
	Price              string `json:"price"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	High24h            string `json:"high24h"`
	Low24h             string `json:"low24h"`
	Volume24h          string `json:"volume24h"`
	Turnover24h        string `json:"turnover24h"`
	OpenInterest       string `json:"openInterest"`
	FundingRate        string `json:"fundingRate"`
	NextFundingTime    int64  `json:"nextFundingTime"`
	MarkPrice          string `json:"markPrice"`
	IndexPrice         string `json:"indexPrice"`
	LastUpdateTime     int64  `json:"lastUpdateTime"`
}

// Order Book Depth
type OrderBookDepth struct {
	Symbol       string       `json:"symbol"`
	LastUpdateID int64        `json:"lastUpdateId"`
	Bids         []PriceLevel `json:"bids"`
	Asks         []PriceLevel `json:"asks"`
}

type PriceLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
	Orders   int    `json:"orders"`
}

// Trade is a single executed trade. Populated by the matching engine.
type Trade struct {
	ID        string `json:"id"`
	Symbol    string `json:"symbol"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	Side      string `json:"side"`
	Timestamp int64  `json:"timestamp"`
}

// Kline is a single OHLC candle.
type Kline struct {
	OpenTime  int64  `json:"openTime"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
	CloseTime int64  `json:"closeTime"`
}

// MarketService is the server-facing name for the market data service.
type MarketService = MarketDataService

// NewMarketService constructs the market data service.
func NewMarketService() *MarketService { return NewMarketDataService() }

// GetTickers returns all tickers (alias of GetAllTickers).
func (s *MarketDataService) GetTickers(ctx context.Context) ([]*MarketTicker, error) {
	return s.GetAllTickers(ctx)
}

// GetTrades returns the real recorded trades for a symbol (empty until the
// matching engine records executions).
func (s *MarketDataService) GetTrades(ctx context.Context, symbol string, limit string) ([]*Trade, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	trades := s.trades[symbol]
	n := atoiDefault(limit, 0)
	if n > 0 && len(trades) > n {
		trades = trades[len(trades)-n:]
	}
	if trades == nil {
		trades = []*Trade{}
	}
	return trades, nil
}

// GetFundingRate returns the current funding rate for a symbol.
func (s *MarketDataService) GetFundingRate(ctx context.Context, symbol string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tickers[symbol]
	if !ok {
		return "", fmt.Errorf("symbol not found: %s", symbol)
	}
	return t.FundingRate, nil
}

// Initialize with 50+ trading pairs
func NewMarketDataService() *MarketDataService {
	s := &MarketDataService{
		tickers:     make(map[string]*MarketTicker),
		depths:      make(map[string]*OrderBookDepth),
		trades:      make(map[string][]*Trade),
		klines:      make(map[string]map[string][]*Kline),
		pricePoints: make(map[string][]pricePoint),
		prices:      make(map[string]float64),
	}

	// Initialize all trading pairs with realistic prices
	// symbols is the real pair catalog (quote USD). Prices are seeded from the
	// CoinGecko oracle below; none are hardcoded.
	symbols := []string{
		"BTC-USD", "ETH-USD", "BNB-USD", "XRP-USD", "SOL-USD", "ADA-USD",
		"DOGE-USD", "DOT-USD", "UNI-USD", "LINK-USD", "LTC-USD", "ATOM-USD",
		"AVAX-USD", "MATIC-USD", "NEAR-USD", "APT-USD", "ARB-USD", "OP-USD",
		"USDC-USD", "USDT-USD",
	}

	// Fetch real USD prices once at startup. Fail-closed: a pair with no live
	// price is omitted (never given a fabricated price).
	base := []string{}
	for _, sym := range symbols {
		base = append(base, strings.SplitN(sym, "-", 2)[0])
	}
	pairs := map[string]float64{}
	live, err := fetchLivePricesUSD(base)
	if err == nil {
		for _, sym := range symbols {
			b := strings.SplitN(sym, "-", 2)[0]
			if p, ok := live[b]; ok && p > 0 {
				pairs[sym] = p
			}
		}
	}

	for symbol, price := range pairs {
		priceStr := fmt.Sprintf("%.8f", price)
		s.prices[symbol] = price
		s.tickers[symbol] = &MarketTicker{
			Symbol:             symbol,
			Price:              priceStr,
			PriceChange:        "0",
			PriceChangePercent: "0",
			High24h:            priceStr,
			Low24h:             priceStr,
			Volume24h:          "0",
			Turnover24h:        "0",
			OpenInterest:       "0",
			FundingRate:        "0",
			NextFundingTime:    0,
			MarkPrice:          priceStr,
			IndexPrice:         priceStr,
			LastUpdateTime:     time.Now().Unix(),
		}
		// depth starts empty (no fabricated book); a real matching engine
		// fills it. GetDepth returns an empty book until then.
		s.depths[symbol] = &OrderBookDepth{Symbol: symbol}
	}

	return s
}

// generateDepth returns an empty order book. A real matching engine maintains
// depth; until one is wired, depth is empty rather than fabricated.
func (s *MarketDataService) generateDepth(symbol string, basePrice float64) *OrderBookDepth {
	return &OrderBookDepth{Symbol: symbol}
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
func (s *MarketDataService) GetDepth(ctx context.Context, symbol string, limit string) (*OrderBookDepth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	depth, ok := s.depths[symbol]
	if !ok {
		return nil, fmt.Errorf("symbol not found: %s", symbol)
	}

	n := atoiDefault(limit, 0)
	limitedDepth := *depth
	if n > 0 && len(limitedDepth.Bids) > n {
		limitedDepth.Bids = limitedDepth.Bids[:n]
		limitedDepth.Asks = limitedDepth.Asks[:n]
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

// generateKlines builds candles from the real recorded price history for the
// symbol. With no history yet it returns an empty slice (never fabricated
// candles).
func (s *MarketDataService) generateKlines(symbol, interval string, count int) []*Kline {
	intervals := map[string]int{"1m": 60, "5m": 300, "15m": 900, "1h": 3600, "4h": 14400, "1d": 86400}
	intervalSec, ok := intervals[interval]
	if !ok {
		intervalSec = 60
	}
	points := s.priceHistory(symbol)
	if len(points) == 0 {
		return []*Kline{}
	}
	return aggregateCandles(points, int64(intervalSec), count)
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

// updatePrices refreshes all tracked prices from the real CoinGecko oracle,
// updates 24h high/low + change, records a history point, and broadcasts the
// updated ticker. No random walk. Fail-closed: keeps last known real price on
// upstream error.
func (s *MarketDataService) updatePrices(hub *WSHub) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bases := make([]string, 0, len(s.prices))
	for symbol := range s.prices {
		bases = append(bases, strings.SplitN(symbol, "-", 2)[0])
	}
	live, err := fetchLivePricesUSD(bases)
	if err != nil || len(live) == 0 {
		return
	}

	for symbol, oldPrice := range s.prices {
		base := strings.SplitN(symbol, "-", 2)[0]
		newPrice, ok := live[base]
		if !ok || newPrice <= 0 {
			continue
		}
		s.prices[symbol] = newPrice
		s.recordPrice(symbol, newPrice)

		if ticker, ok := s.tickers[symbol]; ok {
			priceChange := newPrice - oldPrice
			priceChangePercent := 0.0
			if oldPrice > 0 {
				priceChangePercent = (priceChange / oldPrice) * 100
			}
			ticker.Price = fmt.Sprintf("%.8f", newPrice)
			ticker.MarkPrice = fmt.Sprintf("%.8f", newPrice)
			ticker.IndexPrice = fmt.Sprintf("%.8f", newPrice)
			ticker.PriceChange = fmt.Sprintf("%.8f", priceChange)
			ticker.PriceChangePercent = fmt.Sprintf("%.4f", priceChangePercent)
			ticker.LastUpdateTime = time.Now().Unix()

			// maintain real 24h high/low from recorded prices
			hi, lo := s.highLow24h(symbol, newPrice)
			ticker.High24h = fmt.Sprintf("%.8f", hi)
			ticker.Low24h = fmt.Sprintf("%.8f", lo)
		}

		if hub != nil {
			hub.Broadcast(map[string]interface{}{
				"type": "ticker",
				"data": s.tickers[symbol],
			})
		}
	}
}

// Get symbols
// recordPrice appends a real (timestamp, price) point to the symbol history.
func (s *MarketDataService) recordPrice(symbol string, price float64) {
	if s.klines == nil {
		return
	}
	pts := s.pricePoints[symbol]
	pts = append(pts, pricePoint{ts: time.Now().Unix(), price: price})
	// cap history to 24h worth of 10s points (~8640)
	if len(pts) > 8640 {
		pts = pts[len(pts)-8640:]
	}
	s.pricePoints[symbol] = pts
}

// priceHistory returns recorded price points for a symbol.
func (s *MarketDataService) priceHistory(symbol string) []pricePoint {
	return s.pricePoints[symbol]
}

// highLow24h returns the 24h high/low from recorded history, seeded with the
// current price when history is empty.
func (s *MarketDataService) highLow24h(symbol string, current float64) (float64, float64) {
	hi, lo := current, current
	cutoff := time.Now().Unix() - 86400
	for _, p := range s.pricePoints[symbol] {
		if p.ts < cutoff {
			continue
		}
		if p.price > hi {
			hi = p.price
		}
		if p.price < lo {
			lo = p.price
		}
	}
	return hi, lo
}

// aggregateCandles builds OHLC candles from recorded price points.
func aggregateCandles(points []pricePoint, intervalSec int64, count int) []*Kline {
	if intervalSec <= 0 {
		intervalSec = 60
	}
	type acc struct {
		open, high, low, close float64
		openTime, closeTime    int64
	}
	buckets := map[int64]*acc{}
	var order []int64
	for _, p := range points {
		b := (p.ts / intervalSec) * intervalSec
		a, ok := buckets[b]
		if !ok {
			a = &acc{open: p.price, high: p.price, low: p.price, close: p.price, openTime: b, closeTime: b + intervalSec}
			buckets[b] = a
			order = append(order, b)
			continue
		}
		if p.price > a.high {
			a.high = p.price
		}
		if p.price < a.low {
			a.low = p.price
		}
		a.close = p.price
	}
	// sort ascending
	for i := 0; i < len(order); i++ {
		for j := i + 1; j < len(order); j++ {
			if order[j] < order[i] {
				order[i], order[j] = order[j], order[i]
			}
		}
	}
	if count > 0 && len(order) > count {
		order = order[len(order)-count:]
	}
	out := make([]*Kline, 0, len(order))
	for _, b := range order {
		a := buckets[b]
		out = append(out, &Kline{
			OpenTime:  a.openTime,
			Open:      fmt.Sprintf("%.8f", a.open),
			High:      fmt.Sprintf("%.8f", a.high),
			Low:       fmt.Sprintf("%.8f", a.low),
			Close:     fmt.Sprintf("%.8f", a.close),
			Volume:    "0",
			CloseTime: a.closeTime,
		})
	}
	return out
}

// pricePoint is a recorded (timestamp, price) sample.
type pricePoint struct {
	ts    int64
	price float64
}

func (s *MarketDataService) GetSymbols() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var symbols []string
	for symbol := range s.tickers {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// atoiDefault parses an int string, returning def on error.
func atoiDefault(v string, def int) int {
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
