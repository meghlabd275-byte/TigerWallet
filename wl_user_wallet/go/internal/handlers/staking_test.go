package handlers

import (
	"context"
	"encoding/hex"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// The selectors in the staking table must match the canonical signatures.
// A wrong selector produces a call the contract rejects, or worse, one that
// invokes an unrelated function.
func TestStakingSelectorsMatchSignatures(t *testing.T) {
	for _, a := range stakingAssets {
		want := "0x" + hex.EncodeToString(crypto.Keccak256([]byte(a.DepositSignature))[:4])
		if a.DepositMethod != want {
			t.Errorf("%s deposit selector is %s, want %s for %q",
				a.Symbol, a.DepositMethod, want, a.DepositSignature)
		}
		if a.WithdrawalSignature != "" {
			wantW := "0x" + hex.EncodeToString(crypto.Keccak256([]byte(a.WithdrawalSignature))[:4])
			if a.WithdrawalMethod != wantW {
				t.Errorf("%s withdrawal selector is %s, want %s for %q",
					a.Symbol, a.WithdrawalMethod, wantW, a.WithdrawalSignature)
			}
		}
	}
}

// Each deposit contract must be a syntactically valid address, and the
// withdrawal contract, when present, must differ from the deposit contract.
func TestStakingContractsAreDistinctAndWellFormed(t *testing.T) {
	for _, a := range stakingAssets {
		if !common.IsHexAddress(a.DepositContract) {
			t.Errorf("%s deposit contract %q is not a valid address", a.Symbol, a.DepositContract)
			continue
		}
		if common.HexToAddress(a.DepositContract) == (common.Address{}) {
			t.Errorf("%s deposit contract is the zero address", a.Symbol)
		}
		if a.WithdrawalContract == "" {
			continue
		}
		if !common.IsHexAddress(a.WithdrawalContract) {
			t.Errorf("%s withdrawal contract %q is not a valid address", a.Symbol, a.WithdrawalContract)
			continue
		}
		if strings.EqualFold(a.DepositContract, a.WithdrawalContract) {
			t.Errorf("%s reuses the deposit contract as the withdrawal contract", a.Symbol)
		}
	}
}

// The deposit call carries the referral address as its single argument.
func TestSelectorWithArgsEncodesAddress(t *testing.T) {
	referral := common.HexToAddress("0x1111111111111111111111111111111111111111")
	got := selectorWithArgs("0xa1903eab", []common.Address{referral})

	if len(got) != 4+32 {
		t.Fatalf("calldata length is %d, want %d", len(got), 4+32)
	}
	if hex.EncodeToString(got[:4]) != "a1903eab" {
		t.Errorf("selector is %s, want a1903eab", hex.EncodeToString(got[:4]))
	}
	// The address occupies the low 20 bytes of the 32-byte word.
	if hex.EncodeToString(got[4+12:]) != strings.ToLower(referral.Hex()[2:]) {
		t.Errorf("address word is %s, want the referral address right-aligned",
			hex.EncodeToString(got[4:]))
	}
	for i := 4; i < 4+12; i++ {
		if got[i] != 0 {
			t.Fatalf("left padding is not zero at offset %d", i)
		}
	}
}

// A zero referral address must still produce a well-formed call.
func TestSelectorWithArgsZeroAddress(t *testing.T) {
	got := selectorWithArgs("0xa1903eab", []common.Address{{}})
	if len(got) != 36 {
		t.Fatalf("calldata length is %d, want 36", len(got))
	}
	for i := 4; i < 36; i++ {
		if got[i] != 0 {
			t.Fatalf("zero address argument is not zero at offset %d", i)
		}
	}
}

// requestWithdrawals(uint256[],address) encodes a dynamic array followed by an
// address: the head holds the array offset and the owner, and the array data
// follows with its length and elements.
func TestSelectorWithUintArrayLayout(t *testing.T) {
	owner := common.HexToAddress("0x2222222222222222222222222222222222222222")
	amount := new(big.Int).SetUint64(1_000_000_000_000_000_000) // 1e18

	got := selectorWithUintArray("0xd6681042", []*big.Int{amount}, owner)

	// 4 selector + offset + owner + length + one element.
	wantLen := 4 + 32*4
	if len(got) != wantLen {
		t.Fatalf("calldata length is %d, want %d", len(got), wantLen)
	}
	if hex.EncodeToString(got[:4]) != "d6681042" {
		t.Errorf("selector is %s, want d6681042", hex.EncodeToString(got[:4]))
	}

	offset := new(big.Int).SetBytes(got[4:36])
	if offset.Cmp(big.NewInt(64)) != 0 {
		t.Errorf("array offset is %s, want 64", offset)
	}

	ownerWord := got[36:68]
	if hex.EncodeToString(ownerWord[12:]) != strings.ToLower(owner.Hex()[2:]) {
		t.Errorf("owner word is %s, want the owner right-aligned", hex.EncodeToString(ownerWord))
	}

	length := new(big.Int).SetBytes(got[68:100])
	if length.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("array length is %s, want 1", length)
	}

	element := new(big.Int).SetBytes(got[100:132])
	if element.Cmp(amount) != 0 {
		t.Errorf("element is %s, want %s", element, amount)
	}
}

// Several amounts must be laid out in order after the length.
func TestSelectorWithUintArrayMultipleValues(t *testing.T) {
	owner := common.HexToAddress("0x3333333333333333333333333333333333333333")
	values := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}

	got := selectorWithUintArray("0xd6681042", values, owner)
	// selector + offset + owner + length + three elements.
	if len(got) != 4+32*6 {
		t.Fatalf("calldata length is %d, want %d", len(got), 4+32*6)
	}
	if length := new(big.Int).SetBytes(got[68:100]); length.Cmp(big.NewInt(3)) != 0 {
		t.Errorf("array length is %s, want 3", length)
	}
	for i, want := range values {
		start := 100 + i*32
		got := new(big.Int).SetBytes(got[start : start+32])
		if got.Cmp(want) != 0 {
			t.Errorf("element %d is %s, want %s", i, got, want)
		}
	}
}

// A malformed selector must not produce calldata, because silently encoding
// a zero selector would call a random function.
func TestDecodeSelectorRejectsMalformedInput(t *testing.T) {
	for _, bad := range []string{"", "0x", "0xa1", "0xa1903e", "0xzzzzzzzz", "a1903eabffff"} {
		if _, err := decodeSelector(bad); err == nil {
			t.Errorf("decodeSelector(%q) accepted a malformed selector", bad)
		}
	}
	if got := selectorWithArgs("0xzz", nil); got != nil {
		t.Errorf("selectorWithArgs with a bad selector returned %d bytes, want nil", len(got))
	}
	if got := selectorWithUintArray("0xa1903e", nil, common.Address{}); got != nil {
		t.Errorf("selectorWithUintArray with a bad selector returned %d bytes, want nil", len(got))
	}
}

// Lookup must be case-insensitive on the symbol and must honour the chain.
func TestFindStakingAsset(t *testing.T) {
	if a := findStakingAsset("steth", 0); a == nil || a.Symbol != "stETH" {
		t.Errorf("lowercase symbol lookup failed: %+v", a)
	}
	if a := findStakingAsset("rETH", 1); a == nil || a.Symbol != "rETH" {
		t.Errorf("symbol lookup with chain failed: %+v", a)
	}
	if a := findStakingAsset("stETH", 137); a != nil {
		t.Errorf("lookup returned %+v for a chain the asset is not deployed on", a)
	}
	if a := findStakingAsset("not-a-real-asset", 0); a != nil {
		t.Errorf("lookup returned %+v for an unknown asset", a)
	}
}

// The live-rate path must never invent a number. With no provider configured
// the value is unavailable, and the JSON rendering is null.
func TestStakingAprUnavailableIsNull(t *testing.T) {
	t.Setenv("STAKING_APR_PROVIDER_stETH", "")
	if _, ok := resolveStakingApr(context.Background(), "stETH"); ok {
		t.Skip("a live provider answered; the unavailable path cannot be exercised here")
	}
	if got := aprOrNil(0, false); got != nil {
		t.Errorf("aprOrNil for an unavailable rate is %v, want nil", got)
	}
	if got := aprOrNil(2.25, true); got != "2.2500" {
		t.Errorf("aprOrNil for a known rate is %v, want 2.2500", got)
	}
}

// jsonPath must read both numbers and numeric strings, and must not panic on
// a response whose shape differs from what the provider documents.
func TestJSONPath(t *testing.T) {
	obj := map[string]any{
		"data":   map[string]any{"apr": 2.248},
		"yearly": "1.9271765773427391064867126020233124252153715810170811916405527174511841213263788150321052132218719993800",
		"nested": map[string]any{"deep": map[string]any{"v": "5"}},
	}
	if v, ok := jsonPath(obj, "data.apr"); !ok || v != "2.248" {
		t.Errorf("jsonPath(data.apr) = %q, %v", v, ok)
	}
	if v, ok := jsonPath(obj, "yearly"); !ok || !strings.HasPrefix(v, "1.927") {
		t.Errorf("jsonPath(yearly) = %q, %v", v, ok)
	}
	if v, ok := jsonPath(obj, "nested.deep.v"); !ok || v != "5" {
		t.Errorf("jsonPath(nested.deep.v) = %q, %v", v, ok)
	}
	if _, ok := jsonPath(obj, "missing.key"); ok {
		t.Error("jsonPath returned a value for a missing path")
	}
	if _, ok := jsonPath(obj, "data.apr.deeper"); ok {
		t.Error("jsonPath walked into a scalar")
	}
}

// Environment variable suffixes must be stable so operators can rely on them.
func TestSanitizeEnvSuffix(t *testing.T) {
	cases := map[string]string{
		"stETH":       "STETH",
		"rETH":        "RETH",
		"Lido stETH":  "LIDO_STETH",
		"weird-name!": "WEIRD_NAME_",
	}
	for in, want := range cases {
		if got := sanitizeEnvSuffix(in); got != want {
			t.Errorf("sanitizeEnvSuffix(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestStakingAprLive reads the real provider endpoints. It is skipped unless
// STAKING_APR_LIVE=1 so the suite does not depend on network access, but it is
// the check that proves the provider parsing matches the live payloads.
func TestStakingAprLive(t *testing.T) {
	if os.Getenv("STAKING_APR_LIVE") != "1" {
		t.Skip("set STAKING_APR_LIVE=1 to read the live provider endpoints")
	}
	for _, a := range stakingAssets {
		t.Setenv("STAKING_APR_PROVIDER_"+sanitizeEnvSuffix(a.Symbol), a.RateProvider)
		rate, ok := resolveStakingApr(context.Background(), a.Symbol)
		if !ok {
			t.Errorf("%s: no live rate was read from %s", a.Symbol, a.RateProvider)
			continue
		}
		if rate <= 0 || rate > 100 {
			t.Errorf("%s: live rate %v is outside a plausible range", a.Symbol, rate)
			continue
		}
		t.Logf("%s live rate: %.4f%%", a.Symbol, rate)
	}
}
