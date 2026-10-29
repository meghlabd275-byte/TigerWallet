package main

// Tests for the reward payout primitives. These cover the conversion and
// validation logic that decides how much is paid and to whom, plus the
// fail-closed behaviour when the payout wallet is not configured.

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestDecimalToBaseUnits(t *testing.T) {
	cases := []struct {
		amount   float64
		decimals uint8
		want     string
	}{
		{1, 6, "1000000"},
		{1.5, 6, "1500000"},
		{0.000001, 6, "1"},
		{1234.56, 6, "1234560000"},
		{0.1, 18, "100000000000000000"},
		{1, 18, "1000000000000000000"},
		{100, 0, "100"},
	}
	for _, tc := range cases {
		got, err := decimalToBaseUnits(tc.amount, tc.decimals)
		if err != nil {
			t.Errorf("decimalToBaseUnits(%v, %d): %v", tc.amount, tc.decimals, err)
			continue
		}
		if got.String() != tc.want {
			t.Errorf("decimalToBaseUnits(%v, %d) = %s, want %s", tc.amount, tc.decimals, got, tc.want)
		}
	}
}

func TestDecimalToBaseUnitsRejectsInvalidAmounts(t *testing.T) {
	for _, amount := range []float64{0, -1, -0.5} {
		if _, err := decimalToBaseUnits(amount, 6); err == nil {
			t.Errorf("decimalToBaseUnits(%v) should be rejected", amount)
		}
	}
	// 1e-9 rounds to a sub-unit amount at 6 decimals and must truncate to zero
	// rather than being paid as one base unit.
	got, err := decimalToBaseUnits(0.0000001, 6)
	if err == nil && got.Sign() != 0 {
		t.Errorf("a sub-unit amount produced %s, want rejection or zero", got)
	}
}

func TestDecimalStringToBaseUnits(t *testing.T) {
	cases := []struct {
		raw      string
		decimals uint8
		want     string
	}{
		{"1", 6, "1000000"},
		{"0.5", 6, "500000"},
		{"10.25", 2, "1025"},
		{"0.0000001", 6, ""}, // truncates below one base unit -> rejected
	}
	for _, tc := range cases {
		got, err := decimalStringToBaseUnits(tc.raw, tc.decimals)
		if tc.want == "" {
			if err == nil {
				t.Errorf("decimalStringToBaseUnits(%q, %d) = %s, want rejection", tc.raw, tc.decimals, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("decimalStringToBaseUnits(%q, %d): %v", tc.raw, tc.decimals, err)
			continue
		}
		if got.String() != tc.want {
			t.Errorf("decimalStringToBaseUnits(%q, %d) = %s, want %s", tc.raw, tc.decimals, got, tc.want)
		}
	}
}

func TestDecimalStringToBaseUnitsRejectsGarbage(t *testing.T) {
	for _, raw := range []string{"", "-1", "+1", "abc", "1.2.3", "1.", ".", "1e6", " "} {
		if _, err := decimalStringToBaseUnits(raw, 6); err == nil {
			t.Errorf("decimalStringToBaseUnits(%q) should be rejected", raw)
		}
	}
}

func TestParsePayoutAddress(t *testing.T) {
	valid := "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"
	addr, err := parsePayoutAddress(valid)
	if err != nil {
		t.Fatalf("parsePayoutAddress(%q): %v", valid, err)
	}
	if addr != common.HexToAddress(valid) {
		t.Errorf("parsed address %s, want %s", addr.Hex(), valid)
	}

	invalid := []string{
		"",
		"0x",
		"742d35Cc6634C0532925a3b844Bc454e4438f44e",                      // no 0x prefix
		"0x742d35Cc6634C0532925a3b844Bc454e4438f44",                     // too short
		"0x742d35Cc6634C0532925a3b844Bc454e4438f44eF",                   // too long
		"0xZZZd35Cc6634C0532925a3b844Bc454e4438f44e",                    // not hex
		"0x742d35Cc6634C0532925a3b844Bc454e4438f44e;DROP TABLE rewards", // injection attempt
	}
	for _, raw := range invalid {
		if _, err := parsePayoutAddress(raw); err == nil {
			t.Errorf("parsePayoutAddress(%q) should be rejected", raw)
		}
	}
}

func TestPayoutClientFailsClosedWithoutConfiguration(t *testing.T) {
	// With no RPC endpoint configured the client must refuse to be built, so a
	// caller can never proceed to mark a reward paid without paying it.
	t.Setenv("BUG_BOUNTY_RPC_URL", "")
	os.Unsetenv("BUG_BOUNTY_PRIVATE_KEY")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := NewPayoutClient(ctx); err == nil {
		t.Fatal("NewPayoutClient succeeded without an RPC endpoint")
	}

	t.Setenv("BUG_BOUNTY_RPC_URL", "http://127.0.0.1:1")
	t.Setenv("BUG_BOUNTY_PRIVATE_KEY", "")
	if _, err := NewPayoutClient(ctx); err == nil {
		t.Fatal("NewPayoutClient succeeded without a payout key")
	}
}

func TestABIEncoding(t *testing.T) {
	// transfer(address,uint256) selector is a well-known constant.
	sel := selector("transfer(address,uint256)")
	want := []byte{0xa9, 0x05, 0x9c, 0xbb}
	if len(sel) != 4 {
		t.Fatalf("selector length = %d, want 4", len(sel))
	}
	for i := range want {
		if sel[i] != want[i] {
			t.Fatalf("transfer selector = %x, want %x", sel, want)
		}
	}

	addr := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
	word := padAddress32(addr)
	if len(word) != 32 {
		t.Fatalf("padded address length = %d, want 32", len(word))
	}
	for i := 0; i < 12; i++ {
		if word[i] != 0 {
			t.Fatalf("address padding byte %d is %x, want 0", i, word[i])
		}
	}
	if common.BytesToAddress(word[12:]) != addr {
		t.Error("padded address does not round-trip")
	}

	value := big.NewInt(1234567)
	vword := padLeft32(value)
	if len(vword) != 32 {
		t.Fatalf("padded value length = %d, want 32", len(vword))
	}
	if new(big.Int).SetBytes(vword).Cmp(value) != 0 {
		t.Error("padded value does not round-trip")
	}
}
