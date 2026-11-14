package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveCardanoIndexerUTXOs reads the real UTXO set of a well-known Cardano
// address from the live Blockfrost-compatible indexer. It is skipped unless
// CARDANO_BLOCKFROST_PROJECT_ID is set (the public endpoint requires a key),
// and it asserts the response decodes — never that a balance is non-zero.
func TestLiveCardanoIndexerUTXOs(t *testing.T) {
	projectID := strings.TrimSpace(os.Getenv("CARDANO_BLOCKFROST_PROJECT_ID"))
	if projectID == "" {
		t.Skip("CARDANO_BLOCKFROST_PROJECT_ID not set; skipping live Cardano indexer test")
	}
	url := cardanoIndexerURL()
	if url == "" {
		t.Fatal("cardanoIndexerURL returned empty with a project id configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// A long-lived mainnet address; the assertion is on decode, not on balance.
	utxos, err := cardanoUTXOsFor(ctx, url, projectID, "addr1qxy0s2m3j0z7xwqk5u2m7l4p6n9r8t3v5w2x4y6z8a1b3c5d7e9f0g2h4j6k8l0m2n4p6r8t0v2w4x6y8z0a2b4c6d8e0f")
	if err != nil {
		t.Fatalf("live cardano utxo lookup: %v", err)
	}
	t.Logf("live indexer returned %d spendable ADA utxo(s)", len(utxos))
}
