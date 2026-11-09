package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestAPI points an ExchangeAPI at a stub server that replays a fixed body,
// so the parsing logic is exercised without touching a real exchange.
func newTestAPI(t *testing.T, name, body string, wantPath string) *ExchangeAPI {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wantPath != "" && r.URL.Path != wantPath {
			t.Errorf("unexpected path: got %s, want %s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return &ExchangeAPI{
		Name:       name,
		BaseURL:    srv.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func TestKrakenGetTicker(t *testing.T) {
	body := `{"error":[],"result":{"XXBTZUSD":{"a":["50001.0","1","1"],"b":["49999.0","1","1"],"c":["50000.0","0.1"],"v":["100.0","200.0"],"h":["51000.0","52000.0"],"l":["49000.0","48000.0"],"o":"49500.0"}}}`
	api := newTestAPI(t, "Kraken", body, "/0/public/Ticker")

	ticker, err := api.krakenGetTicker("BTCUSD")
	if err != nil {
		t.Fatalf("krakenGetTicker: %v", err)
	}
	if ticker.LastPrice != 50000.0 {
		t.Errorf("LastPrice = %v, want 50000", ticker.LastPrice)
	}
	if ticker.BidPrice != 49999.0 || ticker.AskPrice != 50001.0 {
		t.Errorf("bid/ask = %v/%v, want 49999/50001", ticker.BidPrice, ticker.AskPrice)
	}
	// Kraken's v/h/l arrays are [today, last24h]; the 24h figure is index 1.
	if ticker.Volume24h != 200.0 {
		t.Errorf("Volume24h = %v, want 200 (index 1)", ticker.Volume24h)
	}
	if ticker.High24h != 52000.0 || ticker.Low24h != 48000.0 {
		t.Errorf("high/low = %v/%v, want 52000/48000", ticker.High24h, ticker.Low24h)
	}
}

func TestKrakenGetTickerReportsAPIError(t *testing.T) {
	api := newTestAPI(t, "Kraken", `{"error":["EQuery:Unknown asset pair"],"result":{}}`, "")

	if _, err := api.krakenGetTicker("NOPE"); err == nil {
		t.Fatal("expected an error for an unknown pair, got nil")
	}
}

func TestKuCoinGetTicker(t *testing.T) {
	body := `{"code":"200000","data":{"symbol":"BTC-USDT","last":"50000","buy":"49999","sell":"50001","vol":"100","volValue":"5000000","changeRate":"0.02","high":"51000","low":"49000"}}`
	api := newTestAPI(t, "KuCoin", body, "/api/v1/market/stats")

	ticker, err := api.kucoinGetTicker("BTC-USDT")
	if err != nil {
		t.Fatalf("kucoinGetTicker: %v", err)
	}
	if ticker.LastPrice != 50000.0 {
		t.Errorf("LastPrice = %v, want 50000", ticker.LastPrice)
	}
	// KuCoin reports changeRate as a fraction; the ticker exposes a percentage.
	if ticker.PriceChangePct != 2.0 {
		t.Errorf("PriceChangePct = %v, want 2", ticker.PriceChangePct)
	}
}

func TestKuCoinGetTickerRejectsBadCode(t *testing.T) {
	api := newTestAPI(t, "KuCoin", `{"code":"400100","msg":"symbol does not exist"}`, "")

	if _, err := api.kucoinGetTicker("NOPE-USDT"); err == nil {
		t.Fatal("expected an error for a bad response code, got nil")
	}
}

func TestOKXGetTicker(t *testing.T) {
	body := `{"code":"0","msg":"","data":[{"instId":"BTC-USDT","last":"50000","open24h":"49000","bidPx":"49999","askPx":"50001","vol24h":"100","volCcy24h":"5000000","high24h":"51000","low24h":"48000"}]}`
	api := newTestAPI(t, "OKX", body, "/api/v5/market/ticker")

	ticker, err := api.okxGetTicker("BTCUSDT")
	if err != nil {
		t.Fatalf("okxGetTicker: %v", err)
	}
	if ticker.LastPrice != 50000.0 {
		t.Errorf("LastPrice = %v, want 50000", ticker.LastPrice)
	}
	// Change is derived from open24h, not supplied by OKX as a percentage.
	if ticker.PriceChange != 1000.0 {
		t.Errorf("PriceChange = %v, want 1000", ticker.PriceChange)
	}
	if ticker.PriceChangePct < 2.04 || ticker.PriceChangePct > 2.05 {
		t.Errorf("PriceChangePct = %v, want ~2.0408", ticker.PriceChangePct)
	}
}

func TestGateGetTicker(t *testing.T) {
	// Gate returns a bare array, which needs different decoding from the others.
	body := `[{"currency_pair":"BTC_USDT","last":"50000","highest_bid":"49999","lowest_ask":"50001","base_volume":"100","quote_volume":"5000000","change_percentage":"2.04","high_24h":"51000","low_24h":"48000"}]`
	api := newTestAPI(t, "Gate", body, "/api/v4/spot/tickers")

	ticker, err := api.gateGetTicker("BTCUSDT")
	if err != nil {
		t.Fatalf("gateGetTicker: %v", err)
	}
	if ticker.LastPrice != 50000.0 {
		t.Errorf("LastPrice = %v, want 50000", ticker.LastPrice)
	}
	if ticker.QuoteVolume24h != 5000000.0 {
		t.Errorf("QuoteVolume24h = %v, want 5000000", ticker.QuoteVolume24h)
	}
}

func TestGateGetTickerRejectsEmptyArray(t *testing.T) {
	api := newTestAPI(t, "Gate", `[]`, "")

	if _, err := api.gateGetTicker("NOPE_USDT"); err == nil {
		t.Fatal("expected an error for an empty array, got nil")
	}
}

func TestBitgetGetTicker(t *testing.T) {
	body := `{"code":"00000","msg":"success","data":[{"symbol":"BTCUSDT","lastPr":"50000","bidPr":"49999","askPr":"50001","baseVolume":"100","quoteVolume":"5000000","change24h":"0.02","high24h":"51000","low24h":"48000"}]}`
	api := newTestAPI(t, "Bitget", body, "/api/v2/spot/market/tickers")

	ticker, err := api.bitgetGetTicker("BTCUSDT")
	if err != nil {
		t.Fatalf("bitgetGetTicker: %v", err)
	}
	if ticker.LastPrice != 50000.0 {
		t.Errorf("LastPrice = %v, want 50000", ticker.LastPrice)
	}
	if ticker.PriceChangePct != 2.0 {
		t.Errorf("PriceChangePct = %v, want 2", ticker.PriceChangePct)
	}
}

func TestHuobiGetTicker(t *testing.T) {
	body := `{"status":"ok","tick":{"close":50000,"bid":[49999,1],"ask":[50001,1],"amount":100,"vol":5000000,"high":51000,"low":48000}}`
	api := newTestAPI(t, "Huobi", body, "/market/detail/merged")

	ticker, err := api.huobiGetTicker("BTCUSDT")
	if err != nil {
		t.Fatalf("huobiGetTicker: %v", err)
	}
	if ticker.LastPrice != 50000.0 {
		t.Errorf("LastPrice = %v, want 50000", ticker.LastPrice)
	}
	if ticker.BidPrice != 49999.0 || ticker.AskPrice != 50001.0 {
		t.Errorf("bid/ask = %v/%v, want 49999/50001", ticker.BidPrice, ticker.AskPrice)
	}
}

func TestHuobiGetTickerRejectsErrorStatus(t *testing.T) {
	api := newTestAPI(t, "Huobi", `{"status":"error","err-msg":"invalid symbol"}`, "")

	if _, err := api.huobiGetTicker("NOPE"); err == nil {
		t.Fatal("expected an error for an error status, got nil")
	}
}

func TestMEXCGetTicker(t *testing.T) {
	body := `{"symbol":"BTCUSDT","lastPrice":"50000","bidPrice":"49999","askPrice":"50001","volume":"100","quoteVolume":"5000000","priceChange":"1000","priceChangePercent":"2.04","highPrice":"51000","lowPrice":"48000"}`
	api := newTestAPI(t, "MEXC", body, "/api/v3/ticker/24hr")

	ticker, err := api.mexcGetTicker("BTCUSDT")
	if err != nil {
		t.Fatalf("mexcGetTicker: %v", err)
	}
	if ticker.LastPrice != 50000.0 {
		t.Errorf("LastPrice = %v, want 50000", ticker.LastPrice)
	}
	if ticker.PriceChangePct != 2.04 {
		t.Errorf("PriceChangePct = %v, want 2.04", ticker.PriceChangePct)
	}
}

func TestMEXCGetTickerRejectsErrorObject(t *testing.T) {
	api := newTestAPI(t, "MEXC", `{"code":400,"msg":"invalid symbol"}`, "")

	if _, err := api.mexcGetTicker("NOPE"); err == nil {
		t.Fatal("expected an error for an error object, got nil")
	}
}

func TestSymbolNormalisation(t *testing.T) {
	cases := []struct {
		fn   func(string) string
		in   string
		want string
	}{
		{krakenSymbol, "BTCUSDT", "BTC/USDT"},
		{krakenSymbol, "ETHBTC", "ETH/BTC"},
		{krakenSymbol, "SOLUSD", "SOL/USD"},
		{okxSymbol, "BTCUSDT", "BTC-USDT"},
		{okxSymbol, "ETHBTC", "ETH-BTC"},
		{gateSymbol, "BTCUSDT", "BTC_USDT"},
		{huobiSymbol, "BTCUSDT", "btcusdt"},
		// A symbol with no recognised quote currency is passed through unchanged.
		{krakenSymbol, "WEIRD", "WEIRD"},
		{okxSymbol, "WEIRD", "WEIRD"},
	}
	for _, tc := range cases {
		if got := tc.fn(tc.in); got != tc.want {
			t.Errorf("normalisation of %q = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestGetTickerRejectsUnknownExchange(t *testing.T) {
	api := &ExchangeAPI{Name: "NotAnExchange", httpClient: &http.Client{Timeout: time.Second}}

	if _, err := api.GetTicker("BTCUSDT"); err == nil {
		t.Fatal("expected an error for an unsupported exchange, got nil")
	}
}
