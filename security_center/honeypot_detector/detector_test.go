package honeypot

// Tests for the honeypot detector. The important cases are that a detector
// which cannot read a value reports 'unknown' rather than a fabricated figure,
// and that capability detection is driven by the bytecode.

import (
	"strings"
	"testing"
)

func codeOf(parts ...interface{}) []byte {
	var out []byte
	for _, p := range parts {
		switch v := p.(type) {
		case byte:
			out = append(out, v)
		case []byte:
			out = append(out, v...)
		}
	}
	return out
}

func sel(signature string) []byte {
	s := keccakSelector(signature)
	return s[:]
}

func TestKeccakSelectorMatchesKnownValues(t *testing.T) {
	cases := map[string]string{
		"transfer(address,uint256)": "a9059cbb",
		"balanceOf(address)":        "70a08231",
		"pause()":                   "8456cb59",
	}
	for signature, want := range cases {
		if got := hexSelector(signature); got != want {
			t.Errorf("hexSelector(%s) = %s, want %s", signature, got, want)
		}
	}
}

func TestContainsSelectorFindsDispatcherEntry(t *testing.T) {
	code := codeOf(
		byte(0x63), sel("pause()"),
		byte(0x00),
	)
	if !containsSelector(code, "pause()") {
		t.Error("a selector at a PUSH4 site was not detected")
	}
	if containsSelector(code, "mint(address,uint256)") {
		t.Error("a selector that is not in the code was reported as present")
	}
}

func TestContainsSelectorIgnoresBareData(t *testing.T) {
	// A selector appearing without a PUSH4 is data, not a dispatched function.
	code := codeOf(byte(0x00), sel("pause()"), byte(0x00))
	if containsSelector(code, "pause()") {
		t.Error("a selector outside a PUSH4 dispatcher was treated as a capability")
	}
}

func TestCheckSuspiciousFunctionsDetectsRestrictions(t *testing.T) {
	// The detector reads code over RPC, so the detection helper is exercised
	// directly here to prove the matching is driven by the bytecode.
	code := codeOf(
		byte(0x63), sel("pause()"),
		byte(0x63), sel("addBlackList(address)"),
		byte(0x63), sel("mint(address,uint256)"),
	)

	found := map[string]bool{}
	for signature, description := range suspiciousSignatures {
		if containsSelector(code, signature) {
			found[description] = true
		}
	}
	if len(found) != 3 {
		t.Errorf("detected %d capabilities, want 3: %v", len(found), found)
	}
}

func TestCalculateTopHoldersPctWithoutData(t *testing.T) {
	d := &Detector{}
	// With no holder data there is no concentration to report. A non-zero value
	// here would fabricate a centralisation finding.
	if got := d.calculateTopHoldersPct(nil); got != 0 {
		t.Errorf("calculateTopHoldersPct(nil) = %v, want 0", got)
	}
	if got := d.calculateTopHoldersPct([]string{}); got != 0 {
		t.Errorf("calculateTopHoldersPct(empty) = %v, want 0", got)
	}
}

func TestGetHoldersReportsUnavailable(t *testing.T) {
	d := &Detector{}
	holders, err := d.getHolders("http://127.0.0.1:1", "0x0")
	if err == nil {
		t.Fatal("getHolders returned without an indexer")
	}
	if holders != nil {
		t.Errorf("holders = %v, want nil alongside the error", holders)
	}
	if !strings.Contains(err.Error(), "indexer") {
		t.Errorf("error = %v, want it to name the missing indexer", err)
	}
}

func TestGetLiquidityReportsUnavailable(t *testing.T) {
	d := &Detector{}
	liquidity, err := d.getLiquidity("http://127.0.0.1:1", "0x0")
	if err == nil {
		t.Fatal("getLiquidity returned a figure without a configured factory")
	}
	if liquidity != 0 {
		t.Errorf("liquidity = %v, want 0 alongside the error", liquidity)
	}
}

func TestGetTaxInfoFailsClosedWhenUnreachable(t *testing.T) {
	d := &Detector{}
	_, _, err := d.getTaxInfo("http://127.0.0.1:1", "0xdAC17F958D2ee523a2206206994597C13D831ec7")
	if err == nil {
		t.Fatal("getTaxInfo returned a tax with no reachable node")
	}
}

func TestGetTransferCountFailsClosedWhenUnreachable(t *testing.T) {
	d := &Detector{}
	_, err := d.getTransferCount("http://127.0.0.1:1", "0xdAC17F958D2ee523a2206206994597C13D831ec7")
	if err == nil {
		t.Fatal("getTransferCount returned a count with no reachable node")
	}
}

func TestCheckSuspiciousFunctionsFailsClosedWhenUnreachable(t *testing.T) {
	d := &Detector{}
	_, err := d.checkSuspiciousFunctions("http://127.0.0.1:1", "0xdAC17F958D2ee523a2206206994597C13D831ec7")
	if err == nil {
		t.Fatal("checkSuspiciousFunctions returned a verdict with no reachable node")
	}
}

func TestBasisPointsToPercent(t *testing.T) {
	// 500 basis points over a denominator of 100 is 5%.
	word := strings.Repeat("0", 61) + "1f4"
	got, err := basisPointsToPercent("0x"+word, 100)
	if err != nil {
		t.Fatalf("basisPointsToPercent: %v", err)
	}
	if got != 5 {
		t.Errorf("got %v, want 5", got)
	}

	// A value implying a tax above 100% is rejected rather than reported.
	huge := strings.Repeat("0", 56) + "3b9aca00" // 1,000,000,000
	if _, err := basisPointsToPercent("0x"+huge, 100); err == nil {
		t.Error("an implausible tax above 100% was accepted")
	}

	if _, err := basisPointsToPercent("0x01", 100); err == nil {
		t.Error("a truncated uint return value was accepted")
	}
	if _, err := basisPointsToPercent("0x"+word, 0); err == nil {
		t.Error("a zero denominator was accepted")
	}
}

func TestDecodeHex(t *testing.T) {
	code, err := decodeHex("0x60806040")
	if err != nil || len(code) != 4 {
		t.Fatalf("decodeHex = %x, %v; want 4 bytes", code, err)
	}
	if code, err := decodeHex("0x"); err != nil || code != nil {
		t.Errorf("empty hex = %v, %v; want nil, nil", code, err)
	}
	if _, err := decodeHex("0xzz"); err == nil {
		t.Error("invalid hex was accepted")
	}
}

func TestDecodeABIUint(t *testing.T) {
	word := strings.Repeat("0", 56) + "3b9aca00"
	got, err := decodeABIUint("0x" + word)
	if err != nil {
		t.Fatalf("decodeABIUint: %v", err)
	}
	if got.String() != "1000000000" {
		t.Errorf("decoded = %s, want 1000000000", got)
	}
	if _, err := decodeABIUint("0x01"); err == nil {
		t.Error("a truncated uint return value was accepted")
	}
}

func TestGetRPCRejectsUnsupportedChain(t *testing.T) {
	d := &Detector{config: Config{EthereumRPC: "http://eth"}}
	if got := d.getRPC("unknown-chain"); got != "" {
		t.Errorf("getRPC(unknown-chain) = %q, want empty", got)
	}
	if got := d.getRPC("ethereum"); got != "http://eth" {
		t.Errorf("getRPC(ethereum) = %q, want the configured endpoint", got)
	}
}

func TestAnalyzeTokenRejectsUnsupportedChain(t *testing.T) {
	d := &Detector{config: Config{}, cache: map[string]CacheEntry{}}
	if _, err := d.AnalyzeToken("0xdAC17F958D2ee523a2206206994597C13D831ec7", "not-a-chain"); err == nil {
		t.Error("AnalyzeToken accepted an unsupported chain")
	}
}
