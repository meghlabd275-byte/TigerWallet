package store

import "testing"

// listPositions and ClosePosition concatenate a table name into their SQL text
// (a table identifier cannot be a bind parameter), so they must only ever accept
// a name from the allow-list.
func TestPositionTableAcceptsOnlyKnownTables(t *testing.T) {
	for table := range positionTables {
		got, err := positionTable(table)
		if err != nil {
			t.Errorf("positionTable(%q) returned %v; the table is on the allow-list", table, err)
		}
		if got != table {
			t.Errorf("positionTable(%q) returned %q", table, got)
		}
	}
	for _, bad := range []string{
		"",
		"margin_positions; DROP TABLE perp_positions",
		"perp_positions WHERE 1=1 --",
		"pg_shadow",
		"margin_positions m JOIN wallets w ON true",
	} {
		if _, err := positionTable(bad); err == nil {
			t.Errorf("positionTable(%q) accepted a name outside the allow-list", bad)
		}
	}
}

// The two tables the handlers actually close must both be allowed, otherwise
// closing a position would start failing closed.
func TestPositionTableCoversTheHandlersTables(t *testing.T) {
	for _, table := range []string{"margin_positions", "perp_positions"} {
		if _, err := positionTable(table); err != nil {
			t.Errorf("the handlers pass %q but positionTable rejects it: %v", table, err)
		}
	}
}
