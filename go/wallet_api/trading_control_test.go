package main

import (
	"context"
	"math"
	"testing"
)

// Black-Scholes sanity: deep ITM call ≈ S-K, deep OTM call ≈ 0, put-call
// parity at r=0 (C - P = S - K), expiry falls back to intrinsic.
func TestBlackScholes(t *testing.T) {
	S, K := 50000.0, 40000.0
	tY, sigma := 0.25, 0.8

	call := blackScholes("call", S, K, tY, sigma)
	put := blackScholes("put", S, K, tY, sigma)
	if call <= 0 || put <= 0 {
		t.Fatalf("positive prices expected, got call=%v put=%v", call, put)
	}
	// Put-call parity at r=0: C - P = S - K.
	if diff := math.Abs((call - put) - (S - K)); diff > 1e-6 {
		t.Fatalf("put-call parity violated: %v", diff)
	}
	// Deep ITM call approaches intrinsic as vol->0.
	deep := blackScholes("call", S, K, tY, 1e-4)
	if math.Abs(deep-(S-K)) > 1.0 {
		t.Fatalf("deep ITM call should ~= S-K, got %v", deep)
	}
	// Expired option = intrinsic.
	if got := blackScholes("call", S, K, 0, sigma); got != S-K {
		t.Fatalf("expired call should be intrinsic %v, got %v", S-K, got)
	}
	if got := blackScholes("put", S, K, 0, sigma); got != 0 {
		t.Fatalf("expired OTM put should be 0, got %v", got)
	}
	// Degenerate inputs fail closed.
	if got := blackScholes("call", 0, K, tY, sigma); got != 0 {
		t.Fatalf("zero spot should price 0, got %v", got)
	}
	if got := blackScholes("call", S, K, tY, 0); got != 0 {
		t.Fatalf("zero vol should price 0, got %v", got)
	}
}

func TestOptionsIntrinsic(t *testing.T) {
	if got := optionsIntrinsic("call", 100, 90); got != 10 {
		t.Fatalf("call intrinsic: got %v", got)
	}
	if got := optionsIntrinsic("call", 80, 90); got != 0 {
		t.Fatalf("OTM call intrinsic: got %v", got)
	}
	if got := optionsIntrinsic("put", 80, 90); got != 10 {
		t.Fatalf("put intrinsic: got %v", got)
	}
}

func TestValidTradingStatus(t *testing.T) {
	for _, ok := range []string{"active", "stopped", "removed", "suspended"} {
		if !validTradingStatus(ok) {
			t.Fatalf("status %s should be valid", ok)
		}
	}
	for _, bad := range []string{"", "paused", "ACTIVE", "deleted"} {
		if validTradingStatus(bad) {
			t.Fatalf("status %q should be invalid", bad)
		}
	}
}

func TestTradingVerticalsAndKinds(t *testing.T) {
	for _, v := range []string{"spot", "perpetual", "futures", "margin", "options", "copy", "liquidity"} {
		if !tradingVerticals[v] {
			t.Fatalf("vertical %s missing", v)
		}
	}
	for _, k := range []string{"contract", "pool", "pair", "margin_market", "option_series", "copy_trader"} {
		if !tradingEntityKinds[k] {
			t.Fatalf("entity kind %s missing", k)
		}
	}
}

func TestTradingControlKeys(t *testing.T) {
	if got := tradingControlKey("global", "vertical", "margin"); got != "trading:control:global:vertical:margin" {
		t.Fatalf("unexpected key %s", got)
	}
	if got := poolRedisKey(1, "0xaaa", "0xbbb"); got != "1:0XAAA/0XBBB" {
		t.Fatalf("unexpected pool key %s", got)
	}
	if got := optionSeriesKey("BTC", "50000", 1735689600, "call"); got != "BTC-50000-1735689600-CALL" {
		t.Fatalf("unexpected series key %s", got)
	}
}

// normCDF against known values.
func TestNormCDF(t *testing.T) {
	if got := normCDF(0); math.Abs(got-0.5) > 1e-12 {
		t.Fatalf("N(0) should be 0.5, got %v", got)
	}
	if got := normCDF(1.96); math.Abs(got-0.975) > 1e-3 {
		t.Fatalf("N(1.96) should ~= 0.975, got %v", got)
	}
}

// Default-enabled feature flags (owner policy: seamless continuous trading).
// With no Redis reachable (store == nil in unit tests) every builtin feature
// resolves to "enabled"; only an explicit operator stop/pause gates it.
func TestFeatureFlagsDefaultEnabled(t *testing.T) {
	if got := FeatureState("swap_trading"); got != featureStateEnabled {
		t.Fatalf("unset feature must default to enabled, got %s", got)
	}
	if !IsFeatureEnabled("swap_trading") {
		t.Fatal("unset feature must be enabled by default")
	}
	if got := FeatureState(""); got != featureStateDisabled {
		t.Fatalf("empty feature name must stay disabled, got %s", got)
	}
}

// Pair-stop blacklist semantics: unmanaged pairs (or no Redis) never block.
func TestTradingPairStoppedBlacklist(t *testing.T) {
	if tradingPairStopped(context.Background(), "BTC", "USDT") {
		t.Fatal("unmanaged pair must trade freely")
	}
	if tradingPairStopped(context.Background(), "", "USDT") {
		t.Fatal("empty leg must never block")
	}
}
