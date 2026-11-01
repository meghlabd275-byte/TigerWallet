package main

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/tigerwallet/account-abstraction-bundler/abi"
)

func testAPI(cfg *Config) *BundlerAPI {
	if cfg == nil {
		cfg = &Config{}
	}
	return &BundlerAPI{
		config:             cfg,
		sponsorshipCounts:  make(map[common.Address]int),
		sponsorshipWindows: make(map[common.Address]time.Time),
	}
}

func opWith(sender common.Address, callGas, verificationGas, preVerificationGas, maxFee string) (*UserOperation, *abi.UserOperation) {
	op := &UserOperation{
		Sender:               sender,
		CallGasLimit:         callGas,
		VerificationGasLimit: verificationGas,
		PreVerificationGas:   preVerificationGas,
		MaxFeePerGas:         maxFee,
	}
	abiOp := &abi.UserOperation{}
	if v, ok := new(big.Int).SetString(maxFee, 10); ok {
		abiOp.MaxFeePerGas = v
	}
	return op, abiOp
}

// TestSponsorshipRejectsZeroSender proves a sponsorship request without a
// sender is refused: the paymaster could not attribute the spend.
func TestSponsorshipRejectsZeroSender(t *testing.T) {
	api := testAPI(nil)
	op, abiOp := opWith(common.Address{}, "1000", "1000", "1000", "1000000000")
	if err := api.checkSponsorshipPolicy(op, abiOp); err == nil {
		t.Fatal("a zero sender was sponsored")
	}
}

// TestSponsorshipEnforcesGasCap proves the sponsor refuses an operation whose
// total gas exceeds the configured cap, which is what bounds the drain.
func TestSponsorshipEnforcesGasCap(t *testing.T) {
	api := testAPI(&Config{MaxSponsorGas: 100_000})
	sender := common.HexToAddress("0x1111111111111111111111111111111111111111")

	// Exactly at the cap is allowed.
	op, abiOp := opWith(sender, "50000", "30000", "20000", "1000000000")
	if err := api.checkSponsorshipPolicy(op, abiOp); err != nil {
		t.Fatalf("an operation at the cap was refused: %v", err)
	}

	// One unit over is refused.
	op, abiOp = opWith(sender, "50001", "30000", "20000", "1000000000")
	if err := api.checkSponsorshipPolicy(op, abiOp); err == nil {
		t.Fatal("an operation above the gas cap was sponsored")
	}
}

// TestSponsorshipRejectsUnparseableGas proves the sponsor will not sign a blob
// whose cost it cannot compute.
func TestSponsorshipRejectsUnparseableGas(t *testing.T) {
	api := testAPI(&Config{MaxSponsorGas: 100_000})
	sender := common.HexToAddress("0x2222222222222222222222222222222222222222")

	for _, tc := range []struct{ call, verif, pre string }{
		{"abc", "1000", "1000"},
		{"1000", "xyz", "1000"},
		{"1000", "1000", ""},
	} {
		op, abiOp := opWith(sender, tc.call, tc.verif, tc.pre, "1000000000")
		if err := api.checkSponsorshipPolicy(op, abiOp); err == nil {
			t.Fatalf("unparseable gas was accepted: %+v", tc)
		}
	}
}

// TestSponsorshipRejectsZeroMaxFee proves a zero fee cap is refused; the
// paymaster must know the maximum it is underwriting.
func TestSponsorshipRejectsZeroMaxFee(t *testing.T) {
	api := testAPI(&Config{MaxSponsorGas: 100_000})
	sender := common.HexToAddress("0x3333333333333333333333333333333333333333")

	op, abiOp := opWith(sender, "1000", "1000", "1000", "0")
	if err := api.checkSponsorshipPolicy(op, abiOp); err == nil {
		t.Fatal("a zero maxFeePerGas was sponsored")
	}
}

// TestSponsorshipAllowlistEnforced proves a non-allowlisted sender is refused
// when an allowlist is configured, and that an empty allowlist permits anyone.
func TestSponsorshipAllowlistEnforced(t *testing.T) {
	allowed := common.HexToAddress("0x4444444444444444444444444444444444444444")
	other := common.HexToAddress("0x5555555555555555555555555555555555555555")

	api := testAPI(&Config{
		MaxSponsorGas:        100_000,
		SponsorshipAllowlist: map[common.Address]bool{allowed: true},
	})

	op, abiOp := opWith(allowed, "1000", "1000", "1000", "1000000000")
	if err := api.checkSponsorshipPolicy(op, abiOp); err != nil {
		t.Fatalf("an allowlisted sender was refused: %v", err)
	}

	op, abiOp = opWith(other, "1000", "1000", "1000", "1000000000")
	if err := api.checkSponsorshipPolicy(op, abiOp); err == nil {
		t.Fatal("a sender outside the allowlist was sponsored")
	}

	// With no allowlist configured every sender is eligible.
	openAPI := testAPI(&Config{MaxSponsorGas: 100_000})
	op, abiOp = opWith(other, "1000", "1000", "1000", "1000000000")
	if err := openAPI.checkSponsorshipPolicy(op, abiOp); err != nil {
		t.Fatalf("a sender was refused when no allowlist is configured: %v", err)
	}
}

// TestSponsorshipDailyQuotaEnforced is the core regression guard: one sender
// must not be able to consume the paymaster deposit without limit.
func TestSponsorshipDailyQuotaEnforced(t *testing.T) {
	api := testAPI(&Config{MaxSponsorGas: 100_000, SponsorshipPerSenderDaily: 2})
	sender := common.HexToAddress("0x6666666666666666666666666666666666666666")

	for i := 0; i < 2; i++ {
		op, abiOp := opWith(sender, "1000", "1000", "1000", "1000000000")
		if err := api.checkSponsorshipPolicy(op, abiOp); err != nil {
			t.Fatalf("sponsorship %d was refused: %v", i+1, err)
		}
		api.recordSponsorship(sender)
	}

	op, abiOp := opWith(sender, "1000", "1000", "1000", "1000000000")
	if err := api.checkSponsorshipPolicy(op, abiOp); err == nil {
		t.Fatal("the daily quota was exceeded")
	}
}

// TestSponsorshipQuotaIsPerSender proves one sender's usage does not consume
// another sender's quota.
func TestSponsorshipQuotaIsPerSender(t *testing.T) {
	api := testAPI(&Config{MaxSponsorGas: 100_000, SponsorshipPerSenderDaily: 1})
	first := common.HexToAddress("0x7777777777777777777777777777777777777777")
	second := common.HexToAddress("0x8888888888888888888888888888888888888888")

	op, abiOp := opWith(first, "1000", "1000", "1000", "1000000000")
	if err := api.checkSponsorshipPolicy(op, abiOp); err != nil {
		t.Fatal(err)
	}
	api.recordSponsorship(first)

	op, abiOp = opWith(second, "1000", "1000", "1000", "1000000000")
	if err := api.checkSponsorshipPolicy(op, abiOp); err != nil {
		t.Fatalf("a second sender was blocked by the first sender's usage: %v", err)
	}
}

// TestSponsorshipQuotaWindowExpires proves the quota is a rolling 24 hour
// window rather than a permanent ban.
func TestSponsorshipQuotaWindowExpires(t *testing.T) {
	api := testAPI(&Config{MaxSponsorGas: 100_000, SponsorshipPerSenderDaily: 1})
	sender := common.HexToAddress("0x9999999999999999999999999999999999999999")

	api.sponsorshipCounts[sender] = 1
	// Age the window past the cutoff.
	api.sponsorshipWindows[sender] = time.Now().Add(-25 * time.Hour)

	op, abiOp := opWith(sender, "1000", "1000", "1000", "1000000000")
	if err := api.checkSponsorshipPolicy(op, abiOp); err != nil {
		t.Fatalf("the quota did not reset after the window elapsed: %v", err)
	}
}
