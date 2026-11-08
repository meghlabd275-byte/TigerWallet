package main

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newOracleServer stands up a real HTTP server that speaks the price oracle
// protocol, so the tests exercise OraclePriceSource end to end rather than
// stubbing the interface.
func newOracleServer(t *testing.T, prices map[string]float64, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		symbol := strings.TrimPrefix(r.URL.Path, "/api/v1/price/")
		price, ok := prices[strings.ToUpper(symbol)]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"symbol": symbol, "price": price})
	}))
}

func engineWithOracle(t *testing.T, prices map[string]float64) (*PaperTradingEngine, *httptest.Server) {
	t.Helper()
	srv := newOracleServer(t, prices, http.StatusOK)
	e := NewPaperTradingEngine(nil)
	e.SetPriceSource(NewOraclePriceSource(srv.URL))
	return e, srv
}

func TestMarketBuyFillsAtOraclePrice(t *testing.T) {
	e, srv := engineWithOracle(t, map[string]float64{"BTC": 60000})
	defer srv.Close()

	e.CreateAccount("alice")
	order, err := e.CreateOrder("alice", "BTC/USDT", "buy", "market", 0.1, 0, 0)
	if err != nil {
		t.Fatalf("market buy rejected: %v", err)
	}
	if order.Status != "filled" {
		t.Fatalf("status = %q, want filled", order.Status)
	}

	// Fill price must be the oracle price adjusted by the configured slippage,
	// never a value invented by the engine.
	wantFill := 60000 * (1 + e.config.Slippage)
	if math.Abs(order.AvgFillPrice-wantFill) > 1e-6 {
		t.Errorf("fill price = %v, want %v", order.AvgFillPrice, wantFill)
	}
	if order.FilledQty != 0.1 {
		t.Errorf("filled qty = %v, want 0.1", order.FilledQty)
	}

	wantFee := 0.1 * wantFill * e.config.TakerFee
	if math.Abs(order.Fee-wantFee) > 1e-9 {
		t.Errorf("fee = %v, want %v", order.Fee, wantFee)
	}

	account, _ := e.GetAccount("alice")
	wantBalance := e.config.InitialCapital - wantFee
	if math.Abs(account.Balance-wantBalance) > 1e-9 {
		t.Errorf("balance = %v, want %v", account.Balance, wantBalance)
	}
}

// A market order must not be filled from a synthetic price when the oracle is
// unreachable: it stays pending and no fee is charged.
func TestMarketOrderRefusedWhenOracleUnavailable(t *testing.T) {
	e := NewPaperTradingEngine(nil)
	e.SetPriceSource(NewOraclePriceSource("http://127.0.0.1:1"))
	e.CreateAccount("alice")

	order, err := e.CreateOrder("alice", "BTC/USDT", "buy", "market", 0.1, 0, 0)
	if !errors.Is(err, ErrNoPrice) {
		t.Fatalf("err = %v, want ErrNoPrice", err)
	}
	if order != nil {
		t.Errorf("expected no order record, got %+v", order)
	}
	if got := e.GetOpenOrders("alice"); len(got) != 0 {
		t.Errorf("open orders = %d, want 0", len(got))
	}

	account, _ := e.GetAccount("alice")
	if account.Balance != e.config.InitialCapital {
		t.Errorf("balance = %v, want untouched %v", account.Balance, e.config.InitialCapital)
	}
	if len(account.Positions) != 0 {
		t.Errorf("positions = %d, want 0", len(account.Positions))
	}
}

// The oracle is only trusted when it reports a usable number.
func TestOracleRejectsUnusableResponses(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"non-200", http.StatusInternalServerError, ""},
		{"zero price", http.StatusOK, `{"price":0}`},
		{"negative price", http.StatusOK, `{"price":-5}`},
		{"malformed body", http.StatusOK, `not json`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			src := NewOraclePriceSource(srv.URL)
			if _, err := src.Price(context.Background(), "BTC/USDT"); !errors.Is(err, ErrNoPrice) {
				t.Fatalf("err = %v, want ErrNoPrice", err)
			}
		})
	}
}

// A pair symbol must resolve to the oracle's base-asset feed.
func TestOracleMapsPairToBaseAsset(t *testing.T) {
	srv := newOracleServer(t, map[string]float64{"BTC": 60000}, http.StatusOK)
	defer srv.Close()

	src := NewOraclePriceSource(srv.URL)
	for _, symbol := range []string{"BTC/USDT", "btc-usdt", "BTC"} {
		got, err := src.Price(context.Background(), symbol)
		if err != nil {
			t.Fatalf("%s: %v", symbol, err)
		}
		if got != 60000 {
			t.Errorf("%s: price = %v, want 60000", symbol, got)
		}
	}
}

// Positions are namespaced per owner: one user's trade must not appear in
// another user's account.
func TestPositionsAreIsolatedPerAccount(t *testing.T) {
	e, srv := engineWithOracle(t, map[string]float64{"ETH": 2000})
	defer srv.Close()

	e.CreateAccount("alice")
	e.CreateAccount("bob")

	if _, err := e.CreateOrder("alice", "ETH/USDT", "buy", "market", 1, 0, 0); err != nil {
		t.Fatalf("alice order: %v", err)
	}

	alice, _ := e.GetAccount("alice")
	bob, _ := e.GetAccount("bob")

	if len(alice.Positions) != 1 {
		t.Errorf("alice positions = %d, want 1", len(alice.Positions))
	}
	if len(bob.Positions) != 0 {
		t.Errorf("bob positions = %d, want 0", len(bob.Positions))
	}
	for key, pos := range alice.Positions {
		if ownerOf(key) != "alice" {
			t.Errorf("position key %q is not owned by alice", key)
		}
		if pos.Symbol != "ETH/USDT" {
			t.Errorf("position symbol = %q", pos.Symbol)
		}
	}
}

// Equity must track the live mark, so an open long gains as the price rises.
func TestEquityReflectsUnrealizedPnL(t *testing.T) {
	prices := map[string]float64{"ETH": 2000}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := strings.TrimPrefix(r.URL.Path, "/api/v1/price/")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"symbol": symbol, "price": prices["ETH"]})
	}))
	defer srv.Close()

	e := NewPaperTradingEngine(&PaperTradingConfig{
		InitialCapital: 10000,
		Leverage:       1,
		TakerFee:       0,
		Slippage:       0,
		EnableShorting: true,
	})
	e.SetPriceSource(NewOraclePriceSource(srv.URL))
	e.CreateAccount("alice")

	if _, err := e.CreateOrder("alice", "ETH/USDT", "buy", "market", 1, 0, 0); err != nil {
		t.Fatalf("order: %v", err)
	}

	prices["ETH"] = 2500 // mark moves up after the fill
	stats, err := e.GetStats("alice")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if math.Abs(stats.TotalPNL-500) > 1e-6 {
		t.Errorf("unrealized PnL = %v, want 500", stats.TotalPNL)
	}

	account, _ := e.GetAccount("alice")
	if math.Abs(account.Equity-10500) > 1e-6 {
		t.Errorf("equity = %v, want 10500", account.Equity)
	}
}

// Closing a long with an equal opposing sell removes the position.
func TestOpposingOrderClosesPosition(t *testing.T) {
	e, srv := engineWithOracle(t, map[string]float64{"ETH": 2000})
	defer srv.Close()

	e.CreateAccount("alice")
	if _, err := e.CreateOrder("alice", "ETH/USDT", "buy", "market", 1, 0, 0); err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := e.CreateOrder("alice", "ETH/USDT", "sell", "market", 1, 0, 0); err != nil {
		t.Fatalf("close: %v", err)
	}

	account, _ := e.GetAccount("alice")
	if len(account.Positions) != 0 {
		t.Errorf("positions = %d, want 0 after the close", len(account.Positions))
	}
}

// Margin is validated against the reference price, so an order that exceeds
// free margin is rejected instead of being silently accepted.
func TestInsufficientMarginRejected(t *testing.T) {
	e, srv := engineWithOracle(t, map[string]float64{"BTC": 60000})
	defer srv.Close()

	e.CreateAccount("alice")
	// 1 BTC at 60k with leverage 1 needs 60k of the 10k balance.
	_, err := e.CreateOrder("alice", "BTC/USDT", "buy", "market", 1, 0, 0)
	if err == nil {
		t.Fatal("expected an insufficient-margin error")
	}
	if !strings.Contains(err.Error(), "insufficient margin") {
		t.Errorf("err = %v, want insufficient margin", err)
	}
}

func TestOrderValidation(t *testing.T) {
	e, srv := engineWithOracle(t, map[string]float64{"BTC": 60000})
	defer srv.Close()
	e.CreateAccount("alice")

	cases := []struct {
		name            string
		side, orderType string
		quantity, price float64
	}{
		{"zero quantity", "buy", "market", 0, 0},
		{"negative quantity", "buy", "market", -1, 0},
		{"NaN quantity", "buy", "market", math.NaN(), 0},
		{"bad side", "sideways", "market", 1, 0},
		{"bad type", "buy", "stop", 1, 0},
		{"limit without price", "buy", "limit", 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := e.CreateOrder("alice", "BTC/USDT", tc.side, tc.orderType, tc.quantity, tc.price, 0); err == nil {
				t.Fatal("expected a validation error")
			}
		})
	}
}

// A short user id must not panic when it is embedded in an order id.
func TestShortUserIDDoesNotPanic(t *testing.T) {
	e, srv := engineWithOracle(t, map[string]float64{"BTC": 60000})
	defer srv.Close()

	e.CreateAccount("a")
	order, err := e.CreateOrder("a", "BTC/USDT", "buy", "limit", 0.01, 50000, 0)
	if err != nil {
		t.Fatalf("limit order: %v", err)
	}
	if !strings.HasPrefix(order.OrderID, "paper_") {
		t.Errorf("order id = %q, want a paper_ prefix", order.OrderID)
	}
}

// Limit orders stay pending until cancelled and are never filled from a price.
func TestLimitOrderStaysPendingUntilCancelled(t *testing.T) {
	e, srv := engineWithOracle(t, map[string]float64{"BTC": 60000})
	defer srv.Close()

	e.CreateAccount("alice")
	order, err := e.CreateOrder("alice", "BTC/USDT", "buy", "limit", 0.01, 50000, 0)
	if err != nil {
		t.Fatalf("limit order: %v", err)
	}
	if order.Status != "pending" {
		t.Fatalf("status = %q, want pending", order.Status)
	}
	if got := e.GetOpenOrders("alice"); len(got) != 1 {
		t.Fatalf("open orders = %d, want 1", len(got))
	}

	if err := e.CancelOrder(order.OrderID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if got := e.GetOpenOrders("alice"); len(got) != 0 {
		t.Errorf("open orders = %d, want 0 after cancel", len(got))
	}
	if err := e.CancelOrder(order.OrderID); err == nil {
		t.Error("cancelling a cancelled order must fail")
	}
}

func TestResetAccountClearsOrdersAndPositions(t *testing.T) {
	e, srv := engineWithOracle(t, map[string]float64{"ETH": 2000})
	defer srv.Close()

	e.CreateAccount("alice")
	if _, err := e.CreateOrder("alice", "ETH/USDT", "buy", "market", 1, 0, 0); err != nil {
		t.Fatalf("order: %v", err)
	}
	if err := e.ResetAccount("alice"); err != nil {
		t.Fatalf("reset: %v", err)
	}

	account, _ := e.GetAccount("alice")
	if account.Balance != e.config.InitialCapital {
		t.Errorf("balance = %v, want %v", account.Balance, e.config.InitialCapital)
	}
	if len(account.Positions) != 0 {
		t.Errorf("positions = %d, want 0", len(account.Positions))
	}
	if got := e.GetOpenOrders("alice"); len(got) != 0 {
		t.Errorf("open orders = %d, want 0", len(got))
	}
	if err := e.ResetAccount("nobody"); err == nil {
		t.Error("resetting an unknown account must fail")
	}
}

func TestShortingDisabledRejectsSell(t *testing.T) {
	e, srv := engineWithOracle(t, map[string]float64{"ETH": 2000})
	defer srv.Close()
	e.config.EnableShorting = false
	e.CreateAccount("alice")

	if _, err := e.CreateOrder("alice", "ETH/USDT", "sell", "market", 1, 0, 0); err == nil {
		t.Fatal("expected shorting to be rejected")
	}
}

func TestToJSONIsValidJSON(t *testing.T) {
	e, srv := engineWithOracle(t, map[string]float64{"ETH": 2000})
	defer srv.Close()
	e.CreateAccount("alice")
	if _, err := e.CreateOrder("alice", "ETH/USDT", "buy", "market", 1, 0, 0); err != nil {
		t.Fatalf("order: %v", err)
	}

	out, err := e.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	for _, key := range []string{"config", "accounts", "orders", "positions", "startedAt"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}
}
