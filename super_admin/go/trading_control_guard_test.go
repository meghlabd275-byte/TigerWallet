package main

import (
	"os"
	"strings"
	"testing"
)

// The generic trading-management helpers interpolate a table name into their SQL
// text, because an identifier cannot be a bind parameter. master_wallet/backend
// already guards that with an allow-list; the super-admin tier had no guard, so
// the interpolation was one careless caller away from an injection point. These
// tests pin the guard and the set of tables the handlers actually use.

func TestSaValidTradingTableAcceptsOnlyKnownTables(t *testing.T) {
	for table := range saTradingTables {
		if !saValidTradingTable(table) {
			t.Errorf("saValidTradingTable(%q) returned false for a table on the allow-list", table)
		}
	}
	for _, bad := range []string{
		"",
		"trading_contracts; DROP TABLE trading_control_audit",
		"trading_contracts WHERE 1=1 --",
		"pg_shadow",
		"trading_contracts JOIN users ON true",
	} {
		if saValidTradingTable(bad) {
			t.Errorf("saValidTradingTable(%q) accepted a name outside the allow-list", bad)
		}
	}
}

// Every table the overview and the list/status/delete handlers ask for must be
// allowed, otherwise the guard would silently report zeros or 400 the console.
func TestSaTradingTablesCoverTheHandlers(t *testing.T) {
	for _, table := range []string{
		"trading_contracts", "liquidity_pools", "trading_pairs",
		"margin_markets", "options_contracts", "copy_trading_configs",
	} {
		if !saValidTradingTable(table) {
			t.Errorf("a handler uses %q but saValidTradingTable rejects it", table)
		}
	}
}

// Pin that each helper checks before it builds SQL, so a refactor cannot quietly
// drop the guard the way it was missing here.
func TestSaTradingHelpersCheckTheAllowList(t *testing.T) {
	src := readTradingControlSource(t)
	for _, fn := range []string{"saListTrading", "saSetTradingStatus", "saDeleteTradingEntity"} {
		body := functionBody(t, src, fn)
		if !strings.Contains(body, "saValidTradingTable(") {
			t.Errorf("%s does not check saValidTradingTable before building its query", fn)
		}
	}
}

func readTradingControlSource(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("trading_control.go")
	if err != nil {
		t.Fatalf("read trading_control.go: %v", err)
	}
	return string(b)
}

// functionBody returns the text of `func <name>(` up to the next top-level
// function declaration.
func functionBody(t *testing.T, src, name string) string {
	t.Helper()
	marker := "func " + name + "("
	i := strings.Index(src, marker)
	if i < 0 {
		t.Fatalf("%s not found", name)
	}
	rest := src[i:]
	if j := strings.Index(rest[1:], "\nfunc "); j >= 0 {
		return rest[:j+1]
	}
	return rest
}
