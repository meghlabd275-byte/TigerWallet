package main

import (
	"os"
	"strings"
	"testing"
)

// The dashboard helpers build their SQL by concatenating a table (and sometimes a
// column or WHERE fragment), because those identifiers cannot be bind
// parameters. These tests pin the allow-lists that keep that concatenation from
// becoming an injection point.
func TestCountQueryAcceptsOnlyKnownTables(t *testing.T) {
	for table := range allowedCountTables {
		if _, err := countQuery(table, ""); err != nil {
			t.Errorf("countQuery(%q, \"\") returned %v; the table is on the allow-list", table, err)
		}
	}
	if _, err := countQuery("wl_clients; DROP TABLE wl_admins", ""); err == nil {
		t.Error("countQuery accepted an unknown table; the table is concatenated into the SQL text")
	}
	if _, err := countQuery("", ""); err == nil {
		t.Error("countQuery accepted an empty table name")
	}
}

func TestCountQueryAcceptsOnlyKnownWhereClauses(t *testing.T) {
	for where := range allowedWhereClauses {
		if _, err := countQuery("wl_clients", where); err != nil {
			t.Errorf("countQuery(\"wl_clients\", %q) returned %v; the clause is on the allow-list", where, err)
		}
	}
	if _, err := countQuery("wl_clients", "1=1 OR 1=1"); err == nil {
		t.Error("countQuery accepted an unknown WHERE clause; the clause is concatenated into the SQL text")
	}
}

func TestSumQueryAcceptsOnlyKnownTableAndColumn(t *testing.T) {
	for table := range allowedCountTables {
		for column := range allowedSumColumns {
			if _, err := sumQuery(table, column); err != nil {
				t.Errorf("sumQuery(%q, %q) returned %v", table, column, err)
			}
		}
	}
	if _, err := sumQuery("wl_liquidity_pools", "value_usd) FROM wl_admins; --"); err == nil {
		t.Error("sumQuery accepted an unknown column; the column is concatenated into the SQL text")
	}
	if _, err := sumQuery("pg_shadow", "value_usd"); err == nil {
		t.Error("sumQuery accepted an unknown table")
	}
}

// Every table the dashboard actually asks for must be on the allow-list,
// otherwise the summary silently reports zero.
func TestDashboardTablesAreAllowed(t *testing.T) {
	for _, table := range []string{
		"wl_clients", "wl_admins", "wl_products", "wl_trading_pairs",
		"wl_liquidity_pools", "wl_token_configs", "wl_market_maker_bots",
	} {
		if _, err := countQuery(table, ""); err != nil {
			t.Errorf("the dashboard queries %q but countQuery rejects it: %v", table, err)
		}
	}
}

// The helpers used to concatenate the caller's arguments with no check at all.
// Pin that each one now routes through an allow-listed builder, so a later
// refactor cannot quietly drop the guard.
func TestDashboardHelpersRouteThroughTheGuards(t *testing.T) {
	src := readMainSource(t)
	for _, fn := range []string{"countWhere", "countRows", "sumColumn"} {
		body := functionBody(t, src, fn)
		if !strings.Contains(body, "countQuery(") && !strings.Contains(body, "sumQuery(") {
			t.Errorf("%s does not build its query through an allow-listed builder", fn)
		}
	}
}

func readMainSource(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
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
		t.Fatalf("%s not found in main.go", name)
	}
	rest := src[i:]
	if j := strings.Index(rest[1:], "\nfunc "); j >= 0 {
		return rest[:j+1]
	}
	return rest
}
