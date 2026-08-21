package wlgate

import "testing"

// TestClassifyUserTransferAuto — a plain user-to-user EVM transfer with a live
// license is auto-approved (the <1s fast path).
func TestClassifyUserTransferAuto(t *testing.T) {
	a := NewAutoApprover()
	a.SetAlive(true, "")
	d := a.Classify("transfer", "0x742d35Cc2734C9C53B02628328fAB7E2fC9C2C2C", "", "1.5")
	if d.Mode != ModeAuto {
		t.Fatalf("user transfer should be ModeAuto, got %v", d.Mode)
	}
	if !d.Approved {
		t.Fatalf("live license + user transfer should be approved: %s", d.Reason)
	}
}

// TestClassifyRevenuePayoutManual — revenue/fee/treasury txs are ALWAYS Manual,
// regardless of license liveness.
func TestClassifyRevenuePayoutManual(t *testing.T) {
	a := NewAutoApprover()
	a.SetAlive(true, "")
	for _, tx := range []string{"revenue_payout", "treasury_transfer", "treasury_sweep", "fee_withdrawal"} {
		d := a.Classify(tx, "0x742d35Cc2734C9C53B02628328fAB7E2fC9C2C2C", "", "100")
		if d.Mode != ModeManual {
			t.Fatalf("%s should be ModeManual, got %v", tx, d.Mode)
		}
	}
}

// TestClassifyTreasuryRecipientManual — a tx TO a treasury address is Manual,
// even if the tx type is "transfer".
func TestClassifyTreasuryRecipientManual(t *testing.T) {
	a := NewAutoApprover()
	a.SetAlive(true, "")
	a.SetTreasuryAddresses([]string{"0x5ff137bdcedfce4018a1f3b1e1c4b1c4b1c4b1c4"})
	d := a.Classify("transfer", "0x5FF137BdCedFcE4018a1F3B1E1c4b1C4b1C4b1C4", "", "1.5")
	if d.Mode != ModeManual {
		t.Fatalf("transfer to a treasury address should be ModeManual, got %v (reason: %s)", d.Mode, d.Reason)
	}
}

// TestClassifyLicenseDeadDenies — if the license is dead, a user transfer is
// NOT approved (fail-closed; no auto-signing without SuperAdmin authorization).
func TestClassifyLicenseDeadDenies(t *testing.T) {
	a := NewAutoApprover()
	a.SetAlive(false, "license suspended")
	d := a.Classify("transfer", "0x742d35Cc2734C9C53B02628328fAB7E2fC9C2C2C", "", "1.5")
	if d.Mode != ModeAuto {
		t.Fatalf("user transfer should still be ModeAuto (not forced manual)")
	}
	if d.Approved {
		t.Fatal("dead license must NOT auto-approve")
	}
}

// TestClassifyBlockingRuleDenies — an auto-sign block rule can deny an
// auto-approve even when the license is alive.
func TestClassifyBlockingRuleDenies(t *testing.T) {
	a := NewAutoApprover()
	a.SetAlive(true, "")
	a.SetRules([]AutoSignRule{{
		RuleID: "block-usdc", Product: "user_wallet", Fetcher: "*",
		TxType: "transfer", Token: "0xa0b86991c6218b36c1d19d4a2e9eb0e36009b13e",
		MaxAmount: "0", Block: true,
	}})
	d := a.Classify("transfer", "0x742d35Cc2734C9C53B02628328fAB7E2fC9C2C2C", "0xA0b86991c6218b36c1D19D4a2e9Eb0e36009b13e", "1.5")
	if d.Approved {
		t.Fatal("block rule must deny the auto-approve")
	}
	if d.RuleID != "block-usdc" {
		t.Fatalf("expected rule_id block-usdc, got %q", d.RuleID)
	}
}

// TestClassifyPersonalSignAuto — message signing (no value) is auto-approved.
func TestClassifyPersonalSignAuto(t *testing.T) {
	a := NewAutoApprover()
	a.SetAlive(true, "")
	d := a.Classify("personal_sign", "", "", "")
	if d.Mode != ModeAuto || !d.Approved {
		t.Fatalf("personal_sign should be auto-approved: mode=%v approved=%v reason=%s", d.Mode, d.Approved, d.Reason)
	}
}
