package main

import (
	"context"
	"os"
	"testing"
)

// Live check: without ETHERSCAN_API_KEY, Ethereum mainnet history must come
// from the keyless Blockscout backend and return real transactions.
func TestLiveKeylessHistory(t *testing.T) {
	if os.Getenv("LIVE_HISTORY") == "" {
		t.Skip("set LIVE_HISTORY=1 to run")
	}
	os.Unsetenv("ETHERSCAN_API_KEY")
	addr := "0x28C6c06298d514Db089934071355E5743bf21d60"
	txs, err := FetchTransactionHistoryForChain(context.Background(), 1, addr, 5)
	if err != nil {
		t.Fatalf("keyless history failed: %v", err)
	}
	if len(txs) == 0 {
		t.Fatal("expected real transactions from the keyless backend")
	}
	t.Logf("chain 1 keyless: %d txs, first=%s from=%s to=%s value=%s time=%d",
		len(txs), txs[0].Hash, txs[0].From, txs[0].To, txs[0].Value, txs[0].Time)
	if txs[0].Hash == "" || txs[0].Time == 0 {
		t.Fatalf("incomplete transaction: %+v", txs[0])
	}
}
