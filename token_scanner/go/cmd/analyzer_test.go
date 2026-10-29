package main

// Tests for the token analyzer. The bytecode fixtures are assembled from real
// EVM opcodes, and the live test (skipped unless TOKEN_SCANNER_LIVE_RPC is set)
// runs the analysis against real mainnet contracts.

import (
	"context"
	"encoding/hex"
	"os"
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
		"approve(address,uint256)":  "095ea7b3",
		"balanceOf(address)":        "70a08231",
		"totalSupply()":             "18160ddd",
		"mint(address,uint256)":     "40c10f19",
	}
	for signature, want := range cases {
		got := keccakSelector(signature)
		if hex.EncodeToString(got[:]) != want {
			t.Errorf("selector(%s) = %s, want %s", signature, hex.EncodeToString(got[:]), want)
		}
	}
}

func TestDetectCapabilitiesFindsTokenSurface(t *testing.T) {
	code := codeOf(
		byte(opPush4), sel("transfer(address,uint256)"),
		byte(opPush4), sel("balanceOf(address)"),
		byte(opPush4), sel("totalSupply()"),
		byte(0x00),
	)
	caps := detectCapabilities(code)
	if !caps["transfer"] || !caps["balance_of"] || !caps["total_supply"] {
		t.Errorf("capabilities = %v, want the ERC-20 surface", caps)
	}
	if caps["mint"] || caps["blacklist"] {
		t.Errorf("capabilities include signals that are not in the code: %v", caps)
	}
}

func TestDetectCapabilitiesFindsRestrictions(t *testing.T) {
	code := codeOf(
		byte(opPush4), sel("transfer(address,uint256)"),
		byte(opPush4), sel("balanceOf(address)"),
		byte(opPush4), sel("mint(address,uint256)"),
		byte(opPush4), sel("pause()"),
		byte(opPush4), sel("addBlackList(address)"),
		byte(opPush4), sel("setMaxTxAmount(uint256)"),
		byte(opPush4), sel("enableTrading()"),
	)
	caps := detectCapabilities(code)
	for _, want := range []string{"mint", "pause", "blacklist", "anti_whale", "trading_switch"} {
		if !caps[want] {
			t.Errorf("capability %q was not detected (got %v)", want, caps)
		}
	}
}

func TestDetectCapabilitiesIgnoresDataOutsideDispatcher(t *testing.T) {
	// A selector appearing at an offset that is not preceded by PUSH4 is data,
	// not a dispatched function, and must not be reported.
	code := codeOf(byte(0x00), sel("mint(address,uint256)"), byte(0x00))
	if detectCapabilities(code)["mint"] {
		t.Error("a selector outside a PUSH4 dispatcher was treated as a capability")
	}
}

func TestScanOpcodesSkipsPushData(t *testing.T) {
	code := codeOf(
		byte(opPush4), sel("transfer(address,uint256)"),
		byte(opPush1), byte(0x00),
		byte(0x00), // STOP
	)
	f := scanOpcodes(code)
	if f.Instructions != 3 {
		t.Errorf("instructions = %d, want 3", f.Instructions)
	}
	if f.DataBytes != 5 {
		t.Errorf("data bytes = %d, want 5", f.DataBytes)
	}
}

func TestScanOpcodesCountsRiskyInstructions(t *testing.T) {
	f := scanOpcodes(codeOf(byte(opDelegateCall), byte(opSelfDestruct), byte(opCreate2), byte(opCallCode)))
	if f.DelegateCall != 1 || f.SelfDestruct != 1 || f.Create2 != 1 || f.CallCode != 1 {
		t.Errorf("facts = %+v, want one each", f)
	}
}

func TestAnalyzeCodeEmptyIsNotAContract(t *testing.T) {
	a := analyzeCode(nil)
	if a.IsContract {
		t.Error("empty code was reported as a contract")
	}
	if a.IsProxy {
		t.Error("empty code was reported as a proxy")
	}
}

func TestAnalyzeCodeDetectsProxy(t *testing.T) {
	code := codeOf(byte(opDelegateCall), byte(opPush4), sel("transfer(address,uint256)"))
	a := analyzeCode(code)
	if !a.IsProxy {
		t.Error("DELEGATECALL was not reported as proxy behaviour")
	}
	if a.Facts.DelegateCall != 1 {
		t.Errorf("delegatecall count = %d, want 1", a.Facts.DelegateCall)
	}
}

func TestAnalyzeCodeDetectsProxyByUpgradeSelector(t *testing.T) {
	code := codeOf(byte(opPush4), sel("upgradeTo(address)"))
	if !analyzeCode(code).IsProxy {
		t.Error("an upgradeTo selector was not reported as proxy behaviour")
	}
}

func TestAnalyzeTokenRejectsBadAddress(t *testing.T) {
	s := &TokenScannerService{config: &Config{EthereumRPC: "http://127.0.0.1:1"}, tokens: map[string]*Token{}}
	if _, err := s.AnalyzeToken("not-an-address"); err == nil {
		t.Error("AnalyzeToken accepted a malformed address")
	}
}

func TestAnalyzeTokenRequiresRPC(t *testing.T) {
	s := &TokenScannerService{config: &Config{}, tokens: map[string]*Token{}}
	_, err := s.AnalyzeToken("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	if err == nil {
		t.Fatal("AnalyzeToken returned a result with no RPC endpoint configured")
	}
	if !strings.Contains(err.Error(), "no RPC endpoint") {
		t.Errorf("error = %v, want a configuration error", err)
	}
}

func TestAnalyzeTokenFailsClosedWhenRPCDown(t *testing.T) {
	// Port 1 is not listening, so the call must fail rather than return a
	// fabricated clean analysis.
	s := &TokenScannerService{
		config: &Config{EthereumRPC: "http://127.0.0.1:1"},
		tokens: map[string]*Token{},
	}
	_, err := s.AnalyzeToken("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	if err == nil {
		t.Fatal("an unreachable RPC produced an analysis instead of an error")
	}
}

func TestCheckHoneypotReportsUnavailableAnalysis(t *testing.T) {
	s := &TokenScannerService{config: &Config{}, tokens: map[string]*Token{}}
	isHoneypot, reason := s.CheckHoneypot("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	if isHoneypot {
		t.Error("a honeypot was reported without any analysis")
	}
	// The reason must disclose that no check was performed rather than assert
	// that the token is clean.
	if !strings.Contains(reason, "could not analyze") {
		t.Errorf("reason = %q, want it to disclose that the check did not run", reason)
	}
}

func TestRiskLevelForScore(t *testing.T) {
	cases := map[int]string{
		0: "low", 19: "low", 20: "medium", 44: "medium",
		45: "high", 69: "high", 70: "critical", 100: "critical",
	}
	for score, want := range cases {
		if got := riskLevelForScore(score); got != want {
			t.Errorf("riskLevelForScore(%d) = %q, want %q", score, got, want)
		}
	}
}

func TestIsValidAddress(t *testing.T) {
	if !isValidAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7") {
		t.Error("a valid address was rejected")
	}
	for _, bad := range []string{
		"", "0x", "dAC17F958D2ee523a2206206994597C13D831ec7",
		"0xdAC17F958D2ee523a2206206994597C13D831ec", "0xZZZ17F958D2ee523a2206206994597C13D831ec7",
	} {
		if isValidAddress(bad) {
			t.Errorf("isValidAddress(%q) = true, want false", bad)
		}
	}
}

func TestDecodeHexBytes(t *testing.T) {
	code, err := decodeHexBytes("0x60806040")
	if err != nil || len(code) != 4 {
		t.Fatalf("decodeHexBytes = %x, %v; want 4 bytes", code, err)
	}
	if code, err := decodeHexBytes("0x"); err != nil || code != nil {
		t.Errorf("empty hex = %v, %v; want nil, nil", code, err)
	}
	if _, err := decodeHexBytes("0xzz"); err == nil {
		t.Error("invalid hex was accepted")
	}
}

func TestEncodeABIArgs(t *testing.T) {
	addr := encodeAddressArg("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	if len(addr) != 64 {
		t.Errorf("address word length = %d, want 64", len(addr))
	}
	if !strings.HasPrefix(addr, strings.Repeat("0", 24)+"dac17f") {
		t.Errorf("address word = %s, want the address lower-cased and left-padded", addr)
	}

	qty := encodeUint256Arg("0xff")
	if len(qty) != 64 || !strings.HasSuffix(qty, "ff") {
		t.Errorf("uint word = %s, want a 64-character left-padded value", qty)
	}
}

func TestDecodeABIString(t *testing.T) {
	// Build the return value as a real ABI encoder would: a 32-byte offset, a
	// 32-byte length, then the UTF-8 payload padded to a 32-byte boundary.
	word := func(value string) string {
		if len(value) > 64 {
			t.Fatalf("word value %q does not fit in 32 bytes", value)
		}
		return strings.Repeat("0", 64-len(value)) + value
	}
	payload := hex.EncodeToString([]byte("USD"))
	raw := "0x" +
		word("20") + // offset to the payload, 32 bytes
		word("3") + // payload length, 3 bytes
		payload + strings.Repeat("0", 64-len(payload))

	got, err := decodeABIString(raw)
	if err != nil {
		t.Fatalf("decodeABIString: %v", err)
	}
	if got != "USD" {
		t.Errorf("decoded = %q, want USD", got)
	}

	// An empty string is a valid return value.
	empty := "0x" + word("20") + word("0")
	if got, err := decodeABIString(empty); err != nil || got != "" {
		t.Errorf("empty string = %q, %v; want \"\", nil", got, err)
	}

	for _, bad := range []string{"0x00", "0x" + word("20") + word("40")} {
		if _, err := decodeABIString(bad); err == nil {
			t.Errorf("truncated return value %q was accepted", bad)
		}
	}
}

func TestDecodeABIUint(t *testing.T) {
	// 1,000,000,000 decimal is 0x3b9aca00, left-padded to a 32-byte word.
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

func TestGetTokenPriceRejectsBadInput(t *testing.T) {
	s := &TokenScannerService{config: &Config{}, tokens: map[string]*Token{}}
	if _, err := s.GetTokenPrice("nonsense"); err == nil {
		t.Error("GetTokenPrice accepted a malformed address")
	}
}

func TestGetTokenPriceFailsClosedWithoutAPI(t *testing.T) {
	s := &TokenScannerService{config: &Config{}, tokens: map[string]*Token{}}
	_, err := s.GetTokenPrice("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	if err == nil {
		t.Fatal("a price was returned with no price API configured")
	}
	if !strings.Contains(err.Error(), "no price API") {
		t.Errorf("error = %v, want a configuration error", err)
	}
}

func TestGetTokenPriceReturnsKnownPrice(t *testing.T) {
	addr := "0xdac17f958d2ee523a2206206994597c13d831ec7"
	s := &TokenScannerService{
		config: &Config{},
		tokens: map[string]*Token{addr: {Address: addr, Price: 1.0}},
	}
	price, err := s.GetTokenPrice(addr)
	if err != nil {
		t.Fatalf("GetTokenPrice: %v", err)
	}
	if price != 1.0 {
		t.Errorf("price = %v, want 1.0", price)
	}
}

// TestLiveTokenAnalysis runs against a real chain. It is skipped unless
// TOKEN_SCANNER_LIVE_RPC is set.
func TestLiveTokenAnalysis(t *testing.T) {
	rpc := os.Getenv("TOKEN_SCANNER_LIVE_RPC")
	if rpc == "" {
		t.Skip("set TOKEN_SCANNER_LIVE_RPC to run the live check")
	}

	s := &TokenScannerService{
		config: &Config{EthereumRPC: rpc},
		tokens: map[string]*Token{},
	}
	ctx := context.Background()

	// USDT: a real ERC-20 with a blacklist control.
	usdt, err := s.AnalyzeTokenOnChain(ctx, "0xdAC17F958D2ee523a2206206994597C13D831ec7", "1")
	if err != nil {
		t.Fatalf("AnalyzeTokenOnChain(USDT): %v", err)
	}
	t.Logf("USDT: score=%d level=%s mint=%v pause=%v blacklist=%v flags=%v",
		usdt.RiskScore, usdt.RiskLevel, usdt.MintFunction, usdt.PauseFunction, usdt.Blacklist, usdt.Flags)

	if usdt.RiskScore == 0 && !usdt.MintFunction && !usdt.PauseFunction && !usdt.Blacklist {
		t.Errorf("USDT analysis found no capabilities at all, which suggests the code was not parsed: %+v", usdt)
	}
	if !usdt.Blacklist {
		t.Errorf("USDT's blacklist control was not detected: %v", usdt.Flags)
	}

	// WETH: an ERC-20 behind a proxy.
	weth, err := s.AnalyzeTokenOnChain(ctx, "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "1")
	if err != nil {
		t.Fatalf("AnalyzeTokenOnChain(WETH): %v", err)
	}
	t.Logf("WETH: score=%d level=%s flags=%v", weth.RiskScore, weth.RiskLevel, weth.Flags)
	if !contains(weth.Flags, "proxy_contract") {
		t.Errorf("WETH was not detected as a proxy: %v", weth.Flags)
	}

	// A wallet address is not a token contract.
	eoa, err := s.AnalyzeTokenOnChain(ctx, "0x28C6c06298d514Db089934071355E5743bf21d60", "1")
	if err != nil {
		t.Fatalf("AnalyzeTokenOnChain(EOA): %v", err)
	}
	if !contains(eoa.Flags, "not_a_contract") {
		t.Errorf("an address with no code was not reported as such: %v", eoa.Flags)
	}
	t.Logf("EOA: level=%s flags=%v", eoa.RiskLevel, eoa.Flags)
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
