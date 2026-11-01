package main

import (
	"math/big"
	"testing"
)

func TestParseSlippagePercentAcceptsClientFormats(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"0.5", 0.5},
		{"0.5%", 0.5},
		{" 0.5 % ", 0.5},
		{"1", 1},
		{"50", 50},
		{"0.01", 0.01},
	}
	for _, tc := range cases {
		got, err := parseSlippagePercent(tc.in)
		if err != nil {
			t.Errorf("parseSlippagePercent(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseSlippagePercent(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// Zero slippage would mean minOut equals the exact quote, so any pool movement
// reverts the swap. A negative value would invert the limit and accept less
// than nothing. Both must be refused rather than silently applied.
func TestParseSlippagePercentRejectsUnusableValues(t *testing.T) {
	for _, in := range []string{"0", "-1", "-0.5", "51", "100", "abc", "", "  "} {
		if _, err := parseSlippagePercent(in); err == nil {
			t.Errorf("parseSlippagePercent(%q) should fail", in)
		}
	}
}

// The tolerance must move the limit in the protective direction, and the
// ten-thousandths scaling must stay exact for values binary floats cannot hold.
func TestMinOutFromSlippageIsExact(t *testing.T) {
	quoted := big.NewInt(1_000_000_000_000_000_000) // 1e18 = 1 token at 18 dp

	cases := []struct {
		slippage float64
		want     string
	}{
		{0.5, "995000000000000000"},
		{1, "990000000000000000"},
		{0.1, "999000000000000000"},
		{10, "900000000000000000"},
	}
	for _, tc := range cases {
		got := minOutFromSlippage(quoted, tc.slippage)
		if got.String() != tc.want {
			t.Errorf("minOutFromSlippage(1e18, %v%%) = %s, want %s", tc.slippage, got, tc.want)
		}
	}
}

func TestMinOutFromSlippageNeverExceedsQuote(t *testing.T) {
	quoted := big.NewInt(123456789)
	got := minOutFromSlippage(quoted, 0.5)
	if got.Cmp(quoted) > 0 {
		t.Errorf("min out %s must not exceed the quote %s", got, quoted)
	}
}

// An explicit min_output is the user's own limit and must be used verbatim.
func TestResolveMinOutHonoursExplicitLimit(t *testing.T) {
	quoted := big.NewInt(1_000_000_000_000_000_000)
	got, err := resolveMinOut("0.9", "5", 18, quoted)
	if err != nil {
		t.Fatalf("resolveMinOut: %v", err)
	}
	want := big.NewInt(900_000_000_000_000_000)
	if got.Cmp(want) != 0 {
		t.Errorf("min out = %s, want %s (the explicit limit, not the slippage)", got, want)
	}
}

// A limit above the quote can never fill; the caller must be told rather than
// handed a transaction that is guaranteed to revert.
func TestResolveMinOutRejectsLimitAboveQuote(t *testing.T) {
	quoted := big.NewInt(1_000_000_000_000_000_000)
	if _, err := resolveMinOut("2", "", 18, quoted); err == nil {
		t.Fatal("min_output above the quote must be rejected")
	}
}

// With neither field set, the default tolerance still produces a real limit.
func TestResolveMinOutDefaultsToToleranceNotZero(t *testing.T) {
	quoted := big.NewInt(1_000_000_000_000_000_000)
	got, err := resolveMinOut("", "", 18, quoted)
	if err != nil {
		t.Fatalf("resolveMinOut: %v", err)
	}
	if got.Sign() == 0 {
		t.Fatal("default limit must not be zero - a zero limit removes all protection")
	}
	if got.Cmp(quoted) >= 0 {
		t.Errorf("default limit %s should be below the quote %s", got, quoted)
	}
}

func TestResolveMinOutRejectsBadSlippage(t *testing.T) {
	quoted := big.NewInt(1_000_000_000_000_000_000)
	for _, in := range []string{"0", "-3", "75", "junk"} {
		if _, err := resolveMinOut("", in, 18, quoted); err == nil {
			t.Errorf("slippage %q should be rejected", in)
		}
	}
}
