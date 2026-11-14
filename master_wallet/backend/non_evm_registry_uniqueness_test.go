package main

import (
	"testing"
)

// TestRegistryRowsAreUnique asserts no chain id is seeded twice. A duplicate
// row means the seeding upsert can flip a chain between two configurations
// depending on iteration order.
func TestRegistryRowsAreUnique(t *testing.T) {
	seen := map[int64]int{}
	for _, c := range defaultNonEVMChains {
		seen[c.ChainID]++
	}
	for id, n := range seen {
		if n > 1 {
			t.Errorf("chain id %d appears %d times in defaultNonEVMChains", id, n)
		}
	}
	t.Logf("defaultNonEVMChains: %d rows, %d distinct ids", len(defaultNonEVMChains), len(seen))
}
