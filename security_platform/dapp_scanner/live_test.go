package main

// Live verification against a real chain. Skipped unless
// DAPP_SCANNER_LIVE_RPC is set, so the suite does not depend on the network.

import (
	"context"
	"os"
	"testing"
)

func TestLiveContractAnalysis(t *testing.T) {
	rpc := os.Getenv("DAPP_SCANNER_LIVE_RPC")
	if rpc == "" {
		t.Skip("set DAPP_SCANNER_LIVE_RPC to run the live check")
	}

	s := NewSecurityScanner()
	s.rpcURLs = []string{rpc}
	ctx := context.Background()

	id, err := s.chainID(ctx)
	if err != nil {
		t.Fatalf("chainID: %v", err)
	}
	t.Logf("chain id %s", id)

	// USDT on Ethereum: a real ERC-20 with a blacklist control.
	usdt := "0xdAC17F958D2ee523a2206206994597C13D831ec7"
	a, err := s.ScanContract(ctx, usdt)
	if err != nil {
		t.Fatalf("ScanContract(USDT): %v", err)
	}
	t.Logf("USDT: size=%d opcodes=%d token=%v proxy=%v score=%d issues=%d",
		a.CodeSize, a.OpcodeCount, a.IsToken, a.IsProxy, a.Score, len(a.Issues))
	if !a.IsContract || !a.IsToken {
		t.Errorf("USDT was not recognized as an ERC-20 contract")
	}

	// A wallet address has no code.
	eoa := "0x28C6c06298d514Db089934071355E5743bf21d60"
	e, err := s.ScanContract(ctx, eoa)
	if err != nil {
		t.Fatalf("ScanContract(EOA): %v", err)
	}
	if e.IsContract {
		t.Error("an EOA was reported as a contract")
	}
	t.Logf("EOA: isContract=%v score=%d", e.IsContract, e.Score)

	// WETH: a real contract with no blacklist.
	weth := "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
	w, err := s.ScanContract(ctx, weth)
	if err != nil {
		t.Fatalf("ScanContract(WETH): %v", err)
	}
	t.Logf("WETH: size=%d opcodes=%d token=%v proxy=%v score=%d issues=%v",
		w.CodeSize, w.OpcodeCount, w.IsToken, w.IsProxy, w.Score, w.Issues)
}
