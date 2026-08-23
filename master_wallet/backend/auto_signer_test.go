package main

// auto_signer_test.go — real unit tests for the AutoSigner transaction
// classifier and the user-funds guard. Both are pure functions (the guard
// takes a guardContext resolved from the store), so no mocks of the logic
// under test are used; the DB-dependent paths are nil-guarded.

import (
	"context"
	"math/big"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Classifier
// ---------------------------------------------------------------------------

func TestTxKindFromString(t *testing.T) {
	cases := []struct {
		in   string
		want TxKind
	}{
		{"transfer", TxKindUserTransfer},
		{"send", TxKindUserTransfer},
		{"swap", TxKindSwap},
		{"trade", TxKindSwap},
		{"stake", TxKindStake},
		{"unstake", TxKindStake},
		{"claim", TxKindStake},
		{"nft_transfer", TxKindNftTransfer},
		{"personal_sign", TxKindPersonalSign},
		{"typed_data_sign", TxKindTypedDataSign},
		{"revenue_payout", TxKindRevenuePayout},
		{"treasury_transfer", TxKindTreasuryTransfer},
		{"treasury_sweep", TxKindTreasurySweep},
		{"fee_withdrawal", TxKindFeeWithdrawal},
		{"", TxKindUnknown},
		{"whatever", TxKindUnknown},
		{"TRANSFER", TxKindUserTransfer}, // case-insensitive
		{"  Swap  ", TxKindSwap},         // trimmed
	}
	for _, tc := range cases {
		if got := txKindFromString(tc.in); got != tc.want {
			t.Errorf("txKindFromString(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestClassifyTransaction_AutoApprovable(t *testing.T) {
	auto := []string{"transfer", "send", "swap", "stake", "unstake", "claim", "nft_transfer", "personal_sign", "typed_data_sign"}
	for _, txType := range auto {
		dec := classifyTransaction(txType)
		if dec.Mode != "auto" || !dec.Approved {
			t.Errorf("classifyTransaction(%q) = %+v, want auto/approved", txType, dec)
		}
		if !dec.Kind.autoApprovable() {
			t.Errorf("kind %v should be auto-approvable", dec.Kind)
		}
	}
}

func TestClassifyTransaction_NeverAutoApproved(t *testing.T) {
	// Fee/revenue/treasury withdrawals require the two-party SuperAdmin
	// co-sign path — the daemon must NEVER auto-approve them.
	manual := []string{"revenue_payout", "treasury_transfer", "treasury_sweep", "fee_withdrawal"}
	for _, txType := range manual {
		dec := classifyTransaction(txType)
		if dec.Mode != "manual" || dec.Approved {
			t.Errorf("classifyTransaction(%q) = %+v, want manual/not-approved", txType, dec)
		}
		if dec.Kind.autoApprovable() {
			t.Errorf("kind %v must never be auto-approvable", dec.Kind)
		}
		if dec.Reason == "" {
			t.Errorf("classifyTransaction(%q) should give a reason", txType)
		}
	}
}

func TestClassifyTransaction_UnknownFailClosed(t *testing.T) {
	for _, txType := range []string{"", "garbage", "TRANSFER2", "treasury"} {
		dec := classifyTransaction(txType)
		if dec.Mode != "manual" || dec.Approved || dec.Kind != TxKindUnknown {
			t.Errorf("classifyTransaction(%q) = %+v, want fail-closed manual/unknown", txType, dec)
		}
	}
}

// ---------------------------------------------------------------------------
// User-funds guard
// ---------------------------------------------------------------------------

const (
	testMasterAddr   = "0x1111111111111111111111111111111111111111"
	testSubWallet    = "0x2222222222222222222222222222222222222222"
	testExternalAddr = "0x3333333333333333333333333333333333333333"
	testTreasuryAddr = "0x4444444444444444444444444444444444444444"
)

func validGuardTx() *pendingAutoTx {
	return &pendingAutoTx{
		ID:             "tx-1",
		MasterWalletID: "mw-1",
		SubWalletID:    "sub-1",
		TxType:         "transfer",
		Blockchain:     "ethereum",
		FromAddress:    testSubWallet,
		ToAddress:      testExternalAddr,
		Amount:         "1000000000000000000",
		UserInitiated:  true,
	}
}

func validGuardContext() guardContext {
	return guardContext{
		MasterAddress:     testMasterAddr,
		TreasuryAddresses: map[string]bool{normalizeAddr(testTreasuryAddr): true},
		SubWalletAddress:  testSubWallet,
		SubWalletFound:    true,
	}
}

func TestGuardUserFunds(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(tx *pendingAutoTx, g *guardContext)
		wantErr string // substring; empty = expect pass
	}{
		{
			name:   "valid user transfer to user-chosen external address",
			mutate: func(tx *pendingAutoTx, g *guardContext) {},
		},
		{
			name: "swap kind passes the guard",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.TxType = "swap"
			},
		},
		{
			name: "fee withdrawal is never auto-signed",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.TxType = "fee_withdrawal"
			},
			wantErr: "never auto-signed",
		},
		{
			name: "revenue payout is never auto-signed",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.TxType = "revenue_payout"
			},
			wantErr: "never auto-signed",
		},
		{
			name: "treasury sweep is never auto-signed",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.TxType = "treasury_sweep"
			},
			wantErr: "never auto-signed",
		},
		{
			name: "unknown kind fails closed",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.TxType = "mystery"
			},
			wantErr: "never auto-signed",
		},
		{
			name: "no sub-wallet fails",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.SubWalletID = ""
			},
			wantErr: "user-initiated sub-wallet flow",
		},
		{
			name: "sub-wallet lookup failure fails closed",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				g.SubWalletFound = false
			},
			wantErr: "user-initiated sub-wallet flow",
		},
		{
			name: "from address must match the sub-wallet",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.FromAddress = testExternalAddr
			},
			wantErr: "does not match",
		},
		{
			name: "signing FROM the master wallet is refused",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.FromAddress = testMasterAddr
				g.SubWalletAddress = testMasterAddr
			},
			wantErr: "treasury domain",
		},
		{
			name: "destination must be recorded as user-chosen",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.UserInitiated = false
			},
			wantErr: "user-chosen",
		},
		{
			name: "empty destination fails",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.ToAddress = "  "
			},
			wantErr: "empty destination",
		},
		{
			name: "MasterWallet can never pull user funds (to = master)",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.ToAddress = testMasterAddr
			},
			wantErr: "move user funds to the MasterWallet",
		},
		{
			name: "destination matching master case-insensitively is refused",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.ToAddress = strings.ToUpper(testMasterAddr)
			},
			wantErr: "move user funds to the MasterWallet",
		},
		{
			name: "destination to a treasury/revenue address is refused",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.ToAddress = testTreasuryAddr
			},
			wantErr: "treasury/revenue/fee address",
		},
		{
			name: "malformed amount fails",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.Amount = "not-a-number"
			},
			wantErr: "malformed amount",
		},
		{
			name: "negative amount fails",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.Amount = "-5"
			},
			wantErr: "malformed amount",
		},
		{
			name: "zero amount is allowed (e.g. contract call)",
			mutate: func(tx *pendingAutoTx, g *guardContext) {
				tx.Amount = "0"
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx := validGuardTx()
			g := validGuardContext()
			tc.mutate(tx, &g)
			err := guardUserFunds(tx, g)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("guardUserFunds() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("guardUserFunds() expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("guardUserFunds() error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Value cap
// ---------------------------------------------------------------------------

func TestExceedsCap(t *testing.T) {
	one := big.NewInt(1)
	hundred := big.NewInt(100)
	cases := []struct {
		name string
		amt  *big.Int
		cap  *big.Int
		want bool
	}{
		{"nil cap is unlimited", hundred, nil, false},
		{"zero cap is unlimited", hundred, big.NewInt(0), false},
		{"negative cap is unlimited", hundred, big.NewInt(-1), false},
		{"below cap", one, hundred, false},
		{"equal to cap", hundred, hundred, false},
		{"above cap", big.NewInt(101), hundred, true},
	}
	for _, tc := range cases {
		if got := exceedsCap(tc.amt, tc.cap); got != tc.want {
			t.Errorf("%s: exceedsCap(%v, %v) = %v, want %v", tc.name, tc.amt, tc.cap, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Policy
// ---------------------------------------------------------------------------

func TestAutoSignPolicyAllowsKind(t *testing.T) {
	p := defaultAutoSignPolicy("mw-1")
	if !p.Enabled {
		t.Fatal("default policy should be enabled")
	}
	for _, k := range []TxKind{TxKindUserTransfer, TxKindSwap, TxKindStake, TxKindNftTransfer, TxKindPersonalSign, TxKindTypedDataSign} {
		if !p.allowsKind(k) {
			t.Errorf("default policy should allow %v", k)
		}
	}
	// Revenue/treasury/fee/unknown are never allowed regardless of toggles.
	for _, k := range []TxKind{TxKindRevenuePayout, TxKindTreasuryTransfer, TxKindTreasurySweep, TxKindFeeWithdrawal, TxKindUnknown} {
		if p.allowsKind(k) {
			t.Errorf("policy must never allow %v", k)
		}
	}
	// Per-kind toggle off.
	p.AllowSwap = false
	if p.allowsKind(TxKindSwap) {
		t.Error("swap should be blocked after toggling AllowSwap off")
	}
	// Disabled policy blocks everything.
	p.Enabled = false
	if p.allowsKind(TxKindUserTransfer) {
		t.Error("disabled policy must block all kinds")
	}
	// Nil policy blocks everything (fail-closed).
	var nilP *autoSignPolicy
	if nilP.allowsKind(TxKindUserTransfer) {
		t.Error("nil policy must block all kinds")
	}
}

func TestAutoSignPolicyMaxValueWei(t *testing.T) {
	p := defaultAutoSignPolicy("mw-1")
	if v := p.maxValueWei(); v == nil || v.Sign() != 0 {
		t.Errorf("default cap should parse to 0 (unlimited), got %v", v)
	}
	p.MaxAutoValueWei = "1000000000000000000"
	if v := p.maxValueWei(); v == nil || v.String() != "1000000000000000000" {
		t.Errorf("cap parse failed, got %v", v)
	}
	p.MaxAutoValueWei = "junk"
	if v := p.maxValueWei(); v != nil {
		t.Errorf("malformed cap should parse to nil (unlimited), got %v", v)
	}
	var nilP *autoSignPolicy
	if v := nilP.maxValueWei(); v != nil {
		t.Errorf("nil policy cap should be nil, got %v", v)
	}
}

// TestGetAutoSignPolicyNilStore verifies the DB-dependent loader is
// nil-guarded and falls back to the defaults without a database.
func TestGetAutoSignPolicyNilStore(t *testing.T) {
	var svc *Service
	p := svc.getAutoSignPolicy(context.Background(), "mw-nil")
	if p == nil || !p.Enabled || p.MasterWalletID != "mw-nil" {
		t.Errorf("nil service should yield default policy, got %+v", p)
	}
	svc2 := &Service{}
	p2 := svc2.getAutoSignPolicy(context.Background(), "mw-nil2")
	if p2 == nil || !p2.Enabled {
		t.Errorf("nil store should yield default policy, got %+v", p2)
	}
}
