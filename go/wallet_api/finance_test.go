package main

// finance_test.go — unit tests for the wallet & finance plane.
// Pure-function coverage: payment catalog invariants, HKDF deposit-address
// derivation (all 5 families), double-entry leg validation, withdrawal
// HMAC sign/verify, and switch defaults. DB-dependent flows are covered by
// the SQL constraints themselves (CHECK balance >= 0, net-zero journals).

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFinancePaymentCatalog(t *testing.T) {
	countries := financeCountries()
	if len(countries) != 238 {
		t.Fatalf("expected 238 countries, got %d", len(countries))
	}
	seenCC := map[string]bool{}
	for _, cc := range countries {
		if len(cc) != 2 {
			t.Fatalf("bad country code %q", cc)
		}
		if seenCC[cc] {
			t.Fatalf("duplicate country code %q", cc)
		}
		seenCC[cc] = true
	}
	methods := financePaymentMethods()
	if len(methods) != 881 {
		t.Fatalf("expected 881 payment methods, got %d", len(methods))
	}
	seenCode := map[string]bool{}
	bank, mobile := 0, 0
	for _, m := range methods {
		if seenCode[m.Code] {
			t.Fatalf("duplicate method code %q", m.Code)
		}
		seenCode[m.Code] = true
		if m.Kind != "bank" && m.Kind != "mobile" {
			t.Fatalf("method %q has invalid kind %q", m.Code, m.Kind)
		}
		if m.Kind == "bank" {
			bank++
		} else {
			mobile++
		}
	}
	if bank == 0 || mobile == 0 {
		t.Fatalf("catalog must contain both bank and mobile methods (bank=%d mobile=%d)", bank, mobile)
	}
	// Availability checks.
	if !paymentMethodAvailableIn("bank_transfer_us", "US") {
		t.Fatal("domestic bank transfer must be available in its country")
	}
	if paymentMethodAvailableIn("bank_transfer_us", "GB") {
		t.Fatal("US domestic bank transfer must NOT be available in GB")
	}
	if !paymentMethodAvailableIn("swift", "ZW") {
		t.Fatal("global rails must be available in every country")
	}
	if !paymentMethodAvailableIn("pix", "BR") || paymentMethodAvailableIn("pix", "US") {
		t.Fatal("Pix must be BR-only")
	}
	if !paymentMethodAvailableIn("sepa", "DE") || paymentMethodAvailableIn("sepa", "US") {
		t.Fatal("SEPA must be EEA-only")
	}
	if _, ok := paymentMethodByCode("nonexistent_rail"); ok {
		t.Fatal("unknown method codes must not resolve")
	}
}

func TestDepositAddressDerivation(t *testing.T) {
	financeCfg.masterSeed = []byte("test-master-seed-not-for-production")
	defer func() { financeCfg.masterSeed = nil }()
	uid := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	var lastBTC string
	for _, spec := range financeDepositChains {
		addr, err := deriveDepositAddress(uid, spec)
		if err != nil {
			t.Fatalf("%s: derivation failed: %v", spec.Asset, err)
		}
		// Determinism.
		again, err := deriveDepositAddress(uid, spec)
		if err != nil || again != addr {
			t.Fatalf("%s: derivation not deterministic", spec.Asset)
		}
		switch spec.Asset {
		case "BTC":
			if !strings.HasPrefix(addr, "bc1q") {
				t.Fatalf("BTC address must be bech32 P2WPKH (bc1q...), got %s", addr)
			}
			if _, err := bech32DecodeToBytes(addr); err != nil {
				t.Fatalf("BTC address failed bech32 decode: %v", err)
			}
			lastBTC = addr
		case "ETH", "BNB", "MATIC":
			if !strings.HasPrefix(addr, "0x") || len(addr) != 42 {
				t.Fatalf("%s address must be 0x + 40 hex, got %s", spec.Asset, addr)
			}
		case "SOL":
			raw, err := base58Decode(addr)
			if err != nil || len(raw) != 32 {
				t.Fatalf("SOL address must be base58(ed25519 pubkey), got %s", addr)
			}
		case "TRX":
			if !strings.HasPrefix(addr, "T") {
				t.Fatalf("TRX address must start with T, got %s", addr)
			}
		case "LTC":
			if !strings.HasPrefix(addr, "L") {
				t.Fatalf("LTC address must start with L, got %s", addr)
			}
		case "DOGE":
			if !strings.HasPrefix(addr, "D") {
				t.Fatalf("DOGE address must start with D, got %s", addr)
			}
		}
	}
	// Different users -> different addresses.
	other := uuid.MustParse("99999999-8888-7777-6666-555555555555")
	spec := financeDepositChains[0]
	otherAddr, err := deriveDepositAddress(other, spec)
	if err != nil || otherAddr == lastBTC {
		t.Fatal("different users must derive different deposit addresses")
	}
	// Fail-closed without a master seed.
	financeCfg.masterSeed = nil
	if _, err := deriveDepositAddress(uid, spec); err == nil {
		t.Fatal("derivation must fail closed when WALLET_MASTER_SEED is unset")
	}
}

func TestValidateLegsDoubleEntry(t *testing.T) {
	uid := uuid.New()
	ok := []ledgerLeg{
		{UserID: uid, Currency: "BTC", Amount: "1.5", Debit: true},
		{UserID: uuid.New(), Currency: "BTC", Amount: "1.5", Debit: false},
	}
	if err := validateLegs(ok); err != nil {
		t.Fatalf("balanced journal rejected: %v", err)
	}
	unbalanced := []ledgerLeg{
		{UserID: uid, Currency: "BTC", Amount: "1.5", Debit: true},
		{UserID: uuid.New(), Currency: "BTC", Amount: "1.4", Debit: false},
	}
	if err := validateLegs(unbalanced); err == nil {
		t.Fatal("unbalanced journal must be rejected")
	}
	if err := validateLegs([]ledgerLeg{{UserID: uid, Currency: "BTC", Amount: "1", Debit: true}}); err == nil {
		t.Fatal("single-leg journal must be rejected")
	}
	bad := []ledgerLeg{
		{UserID: uid, Currency: "BTC", Amount: "-1", Debit: true},
		{UserID: uid, Currency: "BTC", Amount: "1", Debit: false},
	}
	if err := validateLegs(bad); err == nil {
		t.Fatal("negative amounts must be rejected")
	}
}

func TestWithdrawalHMACSignVerify(t *testing.T) {
	financeCfg.withdrawHMAC = []byte("test-hmac-secret")
	defer func() { financeCfg.withdrawHMAC = nil }()
	now := time.Unix(1700000000, 0).UTC()
	sig := recomputeWithdrawalSignature("id-1", "user-1", "BTC", "0.5", "bc1qxyz", now)
	if len(sig) != 64 {
		t.Fatalf("HMAC-SHA256 hex must be 64 chars, got %d", len(sig))
	}
	// Same payload -> same signature (deterministic, verifiable).
	if sig != recomputeWithdrawalSignature("id-1", "user-1", "BTC", "0.5", "bc1qxyz", now) {
		t.Fatal("signature must be deterministic")
	}
	// Any tampering changes the signature.
	if sig == recomputeWithdrawalSignature("id-1", "user-1", "BTC", "0.6", "bc1qxyz", now) {
		t.Fatal("tampered amount must change the signature")
	}
	if sig == recomputeWithdrawalSignature("id-1", "user-1", "BTC", "0.5", "bc1qabc", now) {
		t.Fatal("tampered address must change the signature")
	}
}

func TestBech32ReferenceVectors(t *testing.T) {
	// BIP-173 valid checksum examples (bech32).
	s, err := bech32Encode("a", []byte{})
	if err != nil || s != "a12uel5l" {
		t.Fatalf("BIP-173 vector failed: got %q err=%v, want a12uel5l", s, err)
	}
	// BIP-350 valid checksum examples (bech32m).
	m, err := bech32mEncode("a", []byte{})
	if err != nil || m != "a1lqfn3a" {
		t.Fatalf("BIP-350 vector failed: got %q err=%v, want a1lqfn3a", m, err)
	}
	// Round-trip: encode -> decode must verify the checksum.
	c, err := bech32Encode("cosmos", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
		0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14})
	if err != nil {
		t.Fatal(err)
	}
	dec, err := bech32DecodeToBytes(c)
	if err != nil {
		t.Fatalf("round-trip decode failed for %q: %v", c, err)
	}
	if len(dec) != 20 {
		t.Fatalf("round-trip payload length mismatch: %d", len(dec))
	}
}

func TestStablecoinValuation(t *testing.T) {
	// Stablecoins are pinned to $1 — no upstream call needed.
	if v, ok := usdValueOf(nil, "USDT", 123.45); !ok || v != 123.45 {
		t.Fatalf("USDT must be pinned to $1, got %v ok=%v", v, ok)
	}
	if v, ok := usdValueOf(nil, "usdc", 10); !ok || v != 10 {
		t.Fatalf("USDC must be pinned to $1 (case-insensitive), got %v ok=%v", v, ok)
	}
}
