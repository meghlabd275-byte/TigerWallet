package main

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// The on-chain integration must stay disabled unless fully configured, so a
// half-configured deployment cannot silently report fabricated success.
func TestOnChainUnconfiguredIsNil(t *testing.T) {
	for _, k := range []string{"STAKING_RPC_URL", "STAKING_OPERATOR_PRIVATE_KEY", "STAKING_CONTRACT_ADDRESS"} {
		os.Unsetenv(k)
	}
	o, err := NewOnChainStakingFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o != nil {
		t.Fatal("expected nil integration when unconfigured")
	}
}

// A partially configured integration must be an error, not a silent no-op.
func TestOnChainPartialConfigFails(t *testing.T) {
	os.Setenv("STAKING_RPC_URL", "http://127.0.0.1:8545")
	os.Setenv("STAKING_OPERATOR_PRIVATE_KEY", "")
	os.Setenv("STAKING_CONTRACT_ADDRESS", "0x0000000000000000000000000000000000000001")
	defer os.Unsetenv("STAKING_RPC_URL")
	defer os.Unsetenv("STAKING_CONTRACT_ADDRESS")

	if _, err := NewOnChainStakingFromEnv(); err == nil {
		t.Fatal("partial configuration accepted")
	}
}

func TestOnChainRejectsBadContractAddress(t *testing.T) {
	os.Setenv("STAKING_RPC_URL", "http://127.0.0.1:8545")
	os.Setenv("STAKING_CONTRACT_ADDRESS", "not-an-address")
	os.Setenv("STAKING_OPERATOR_PRIVATE_KEY", "0x"+strings.Repeat("11", 32))
	defer func() {
		os.Unsetenv("STAKING_RPC_URL")
		os.Unsetenv("STAKING_CONTRACT_ADDRESS")
		os.Unsetenv("STAKING_OPERATOR_PRIVATE_KEY")
	}()

	if _, err := NewOnChainStakingFromEnv(); err == nil {
		t.Fatal("invalid contract address accepted")
	}
}

// A nil integration must fail closed rather than return a hash.
func TestNilIntegrationNeverReturnsHash(t *testing.T) {
	var o *OnChainStaking
	if hash, err := o.Stake(context.Background(), big.NewInt(1)); err == nil || hash != "" {
		t.Fatalf("nil integration returned %q, %v", hash, err)
	}
	if hash, err := o.Withdraw(context.Background(), big.NewInt(1)); err == nil || hash != "" {
		t.Fatalf("nil integration returned %q, %v", hash, err)
	}
	if hash, err := o.ClaimRewards(context.Background()); err == nil || hash != "" {
		t.Fatalf("nil integration returned %q, %v", hash, err)
	}
}

// The staking ABI selectors are fixed by the StakingRewards interface; they
// must not silently drift.
func TestStakingSelectors(t *testing.T) {
	cases := []struct {
		sig  string
		want string
	}{
		{"stake(uint256)", "a694fc3a"},
		{"withdraw(uint256)", "2e1a7d4d"},
		{"getReward()", "3d18b912"},
	}
	for _, c := range cases {
		got := hexStr(crypto.Keccak256([]byte(c.sig))[:4])
		if got != c.want {
			t.Fatalf("selector for %s = %s, want %s", c.sig, got, c.want)
		}
	}

	if hexStr(selStake) != "a694fc3a" || hexStr(selWithdraw) != "2e1a7d4d" || hexStr(selGetReward) != "3d18b912" {
		t.Fatal("package selectors drifted from the interface")
	}
}

func hexStr(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, digits[c>>4], digits[c&0x0f])
	}
	return string(out)
}

// Decimal token amounts must convert to base units exactly, without float
// rounding, and invalid input must be rejected.
func TestParseTokenAmount(t *testing.T) {
	cases := []struct {
		in       string
		decimals int
		want     string
	}{
		{"1", 18, "1000000000000000000"},
		{"1.5", 18, "1500000000000000000"},
		{"0.000000000000000001", 18, "1"},
		{"1.0", 6, "1000000"},
		{"100", 0, "100"},
		{".5", 18, "500000000000000000"},
		{"1.1234567890123456789", 18, "1123456789012345678"}, // extra digits truncated
	}
	for _, c := range cases {
		got, err := parseTokenAmount(c.in, c.decimals)
		if err != nil {
			t.Fatalf("parseTokenAmount(%q, %d): %v", c.in, c.decimals, err)
		}
		if got.String() != c.want {
			t.Fatalf("parseTokenAmount(%q, %d) = %s, want %s", c.in, c.decimals, got, c.want)
		}
	}

	for _, bad := range []string{"", "abc", "1.2.3", "1e5", "-5", "0x10"} {
		if _, err := parseTokenAmount(bad, 18); err == nil {
			t.Fatalf("parseTokenAmount(%q) accepted invalid input", bad)
		}
	}
}

// Large amounts must not lose precision the way float64 would.
func TestParseTokenAmountPrecision(t *testing.T) {
	got, err := parseTokenAmount("123456789012345678901234567890", 18)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "123456789012345678901234567890000000000000000000"
	if got.String() != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestTokenDecimalsDefault(t *testing.T) {
	os.Unsetenv("STAKING_TOKEN_DECIMALS")
	if d := tokenDecimals(); d != 18 {
		t.Fatalf("default decimals = %d, want 18", d)
	}
	os.Setenv("STAKING_TOKEN_DECIMALS", "6")
	defer os.Unsetenv("STAKING_TOKEN_DECIMALS")
	if d := tokenDecimals(); d != 6 {
		t.Fatalf("decimals = %d, want 6", d)
	}
	os.Setenv("STAKING_TOKEN_DECIMALS", "999")
	if d := tokenDecimals(); d != 18 {
		t.Fatalf("out-of-range decimals = %d, want fallback 18", d)
	}
}
