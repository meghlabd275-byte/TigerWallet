package main

// Tests for the DApp scanner. The bytecode cases are assembled from real EVM
// opcodes so the walk and selector detection are exercised on realistic input,
// and the URL cases assert that an unknown URL is never reported as safe.

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

// buildCode assembles opcodes, skipping any argument bytes supplied after a
// PUSH so the fixture stays readable.
func buildCode(parts ...interface{}) []byte {
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

func TestSelectorMatchesCanonicalSignature(t *testing.T) {
	// The ERC-20 transfer selector is a well-known constant.
	if got := selector("transfer(address,uint256)"); hex.EncodeToString(got) != "a9059cbb" {
		t.Errorf("transfer selector = %s, want a9059cbb", hex.EncodeToString(got))
	}
	if got := selector("approve(address,uint256)"); hex.EncodeToString(got) != "095ea7b3" {
		t.Errorf("approve selector = %s, want 095ea7b3", hex.EncodeToString(got))
	}
	if got := selector("balanceOf(address)"); hex.EncodeToString(got) != "70a08231" {
		t.Errorf("balanceOf selector = %s, want 70a08231", hex.EncodeToString(got))
	}
}

func TestWalkOpcodesSkipsPushData(t *testing.T) {
	// PUSH4 <selector> PUSH1 0x00 STOP. The four selector bytes and the one
	// immediate byte must not be counted as instructions.
	code := buildCode(
		byte(opPush1+3), selector("transfer(address,uint256)"),
		byte(opPush1), byte(0x00),
		byte(0x00), // STOP
	)
	p := walkOpcodes(code)
	if p.Instructions != 3 {
		t.Errorf("instructions = %d, want 3 (PUSH4, PUSH1, STOP)", p.Instructions)
	}
	if p.DataBytes != 5 {
		t.Errorf("data bytes = %d, want 5", p.DataBytes)
	}
}

func TestWalkOpcodesCountsRiskyInstructions(t *testing.T) {
	code := buildCode(byte(opDelegateCall), byte(opSelfDestruct), byte(opCreate2))
	p := walkOpcodes(code)
	if p.DelegateCall != 1 || p.SelfDestruct != 1 || p.Create2 != 1 {
		t.Errorf("profile = %+v, want one each of DELEGATECALL/SELFDESTRUCT/CREATE2", p)
	}
}

func TestDetectSelectorsFindsToken(t *testing.T) {
	code := buildCode(
		byte(opPush1+3), selector("transfer(address,uint256)"),
		byte(opPush1+3), selector("balanceOf(address)"),
		byte(opPush1+3), selector("approve(address,uint256)"),
	)
	found := detectSelectors(code)
	if !found["erc20_transfer"] || !found["erc20_balance_of"] || !found["erc20_approve"] {
		t.Errorf("detected selectors = %v, want the ERC-20 set", found)
	}
}

func TestAnalyzeBytecodeEmptyCodeIsEOA(t *testing.T) {
	a := analyzeBytecode("0x0000000000000000000000000000000000000000", nil)
	if a.IsContract {
		t.Error("an address with no code was reported as a contract")
	}
	if len(a.Issues) != 0 {
		t.Errorf("an EOA produced findings: %+v", a.Issues)
	}
	// An EOA carries no contract risk, so it keeps the full score.
	if a.Score != 100 {
		t.Errorf("EOA score = %d, want 100", a.Score)
	}
}

func TestAnalyzeBytecodePlainTokenScoresWell(t *testing.T) {
	// A straightforward ERC-20: dispatcher selectors, no risky instructions,
	// and enough code to avoid the tiny-contract heuristic.
	code := buildCode(
		byte(opPush1+3), selector("transfer(address,uint256)"),
		byte(opPush1+3), selector("balanceOf(address)"),
		byte(opPush1+3), selector("totalSupply()"),
	)
	for i := 0; i < 80; i++ {
		code = append(code, 0x00) // STOP padding, a plain no-op
	}
	a := analyzeBytecode("0x1111111111111111111111111111111111111111", code)
	if !a.IsContract || !a.IsToken {
		t.Errorf("token was not recognized: %+v", a)
	}
	if a.Score != 100 {
		t.Errorf("score = %d, want 100 for a token with no risk signals (issues: %+v)", a.Score, a.Issues)
	}
	if a.IsProxy {
		t.Error("a token with no DELEGATECALL was flagged as a proxy")
	}
}

func TestAnalyzeBytecodeFlagsDelegatecallProxy(t *testing.T) {
	code := buildCode(byte(opDelegateCall), byte(opPush1+3), selector("transfer(address,uint256)"))
	a := analyzeBytecode("0x2222222222222222222222222222222222222222", code)
	if !a.IsProxy {
		t.Error("DELEGATECALL was not flagged as proxy behaviour")
	}
	if !hasIssue(a, "delegatecall_present") {
		t.Errorf("delegatecall finding missing: %+v", a.Issues)
	}
	if a.Score >= 100 {
		t.Errorf("score = %d, want a deduction for DELEGATECALL", a.Score)
	}
}

func TestAnalyzeBytecodeFlagsMetamorphicContract(t *testing.T) {
	code := buildCode(byte(opCreate2), byte(opSelfDestruct))
	a := analyzeBytecode("0x3333333333333333333333333333333333333333", code)
	if !hasIssue(a, "metamorphic_contract") {
		t.Errorf("CREATE2 + SELFDESTRUCT was not flagged as metamorphic: %+v", a.Issues)
	}
	if severityOfFinding(a, "metamorphic_contract") != "critical" {
		t.Error("metamorphic contract was not rated critical")
	}
}

func TestAnalyzeBytecodeFlagsBlacklistControl(t *testing.T) {
	code := buildCode(
		byte(opPush1+3), selector("transfer(address,uint256)"),
		byte(opPush1+3), selector("balanceOf(address)"),
		byte(opPush1+3), selector("setBlacklist(address,bool)"),
	)
	a := analyzeBytecode("0x4444444444444444444444444444444444444444", code)
	if !hasIssue(a, "transfer_restriction") {
		t.Errorf("blacklist control was not flagged: %+v", a.Issues)
	}
}

func TestAnalyzeBytecodeFlagsMintableAndMutableFee(t *testing.T) {
	code := buildCode(
		byte(opPush1+3), selector("transfer(address,uint256)"),
		byte(opPush1+3), selector("balanceOf(address)"),
		byte(opPush1+3), selector("mint(address,uint256)"),
		byte(opPush1+3), selector("setFees(uint256)"),
	)
	a := analyzeBytecode("0x5555555555555555555555555555555555555555", code)
	if !hasIssue(a, "mintable_supply") {
		t.Error("mintable supply was not flagged")
	}
	if !hasIssue(a, "mutable_fee") {
		t.Error("mutable fee was not flagged")
	}
}

func TestAnalyzeBytecodeFlagsTinyContract(t *testing.T) {
	a := analyzeBytecode("0x6666666666666666666666666666666666666666", []byte{0x00, 0x01})
	if !hasIssue(a, "tiny_contract") {
		t.Errorf("a 2-byte contract was not flagged: %+v", a.Issues)
	}
}

func TestAnalyzeBytecodeNeverExceedsBounds(t *testing.T) {
	// A contract made of nothing but SELFDESTRUCT must clamp at zero, not go
	// negative and wrap when converted.
	code := make([]byte, 20)
	for i := range code {
		code[i] = opSelfDestruct
	}
	a := analyzeBytecode("0x7777777777777777777777777777777777777777", code)
	if a.Score < 0 {
		t.Errorf("score = %d, want a value in 0..100", a.Score)
	}
}

func TestDecodeHexCode(t *testing.T) {
	code, err := decodeHexCode("0x60806040")
	if err != nil {
		t.Fatalf("decodeHexCode: %v", err)
	}
	if len(code) != 4 || code[0] != 0x60 || code[3] != 0x40 {
		t.Errorf("decoded = %x, want 60806040", code)
	}

	if code, err := decodeHexCode("0x"); err != nil || code != nil {
		t.Errorf("empty code = %v, %v; want nil, nil", code, err)
	}
	// Odd-length input is left-padded rather than rejected.
	if code, err := decodeHexCode("0x123"); err != nil || len(code) != 2 {
		t.Errorf("odd-length code = %v, %v; want 2 bytes", code, err)
	}
	for _, bad := range []string{"0xzz", "0x60zz"} {
		if _, err := decodeHexCode(bad); err == nil {
			t.Errorf("decodeHexCode(%q) accepted invalid hex", bad)
		}
	}
}

func TestIsHexAddress(t *testing.T) {
	valid := "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"
	if !isHexAddress(valid) {
		t.Error("a valid address was rejected")
	}
	for _, bad := range []string{
		"",
		"0x",
		"742d35Cc6634C0532925a3b844Bc454e4438f44e",
		"0x742d35Cc6634C0532925a3b844Bc454e4438f44",
		"0x742d35Cc6634C0532925a3b844Bc454e4438f44eFF",
		"0xZZZd35Cc6634C0532925a3b844Bc454e4438f44e",
		"0x742d35Cc6634C0532925a3b844Bc454e4438f44e;DROP",
	} {
		if isHexAddress(bad) {
			t.Errorf("isHexAddress(%q) = true, want false", bad)
		}
	}
}

func TestScanContractRejectsBadAddress(t *testing.T) {
	s := &SecurityScanner{contracts: map[string]ContractAnalysis{}}
	if _, err := s.ScanContract(context.Background(), "not-an-address"); err == nil {
		t.Error("ScanContract accepted a malformed address")
	}
}

func TestScanContractFailsClosedWithoutRPC(t *testing.T) {
	s := &SecurityScanner{contracts: map[string]ContractAnalysis{}}
	_, err := s.ScanContract(context.Background(), "0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
	if !errors.Is(err, ErrNoRPCConfigured) {
		t.Fatalf("error = %v, want ErrNoRPCConfigured", err)
	}

	// A failed scan must not leave a cached verdict behind, otherwise the next
	// caller would receive a fabricated clean result.
	if _, ok := s.contracts["0x742d35cc6634c0532925a3b844bc454e4438f44e"]; ok {
		t.Error("a failed scan populated the cache")
	}

	if _, err := s.GetRiskScore(context.Background(), "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"); err == nil {
		t.Error("GetRiskScore returned a score for an unscannable contract")
	}
}

func TestGetRiskScoreDoesNotDeadlock(t *testing.T) {
	// The previous implementation took a read lock, unlocked it on the miss
	// path, and then called the scanning method which took the same read lock
	// again. With the cache populated the call must simply return.
	s := &SecurityScanner{contracts: map[string]ContractAnalysis{}}
	addr := "0x8888888888888888888888888888888888888888"
	s.contracts[strings.ToLower(addr)] = ContractAnalysis{Address: addr, Score: 42}

	score, err := s.GetRiskScore(context.Background(), addr)
	if err != nil {
		t.Fatalf("GetRiskScore: %v", err)
	}
	if score != 42 {
		t.Errorf("score = %d, want 42", score)
	}
}

func TestCheckURLCleanURLIsUnknownNotSafe(t *testing.T) {
	s := NewSecurityScanner()
	v := s.CheckURL("https://app.uniswap.org")
	if v.Verdict != "unknown" {
		t.Errorf("verdict = %q, want unknown (the scanner has no evidence either way)", v.Verdict)
	}
	if v.RiskScore != 0 {
		t.Errorf("risk score = %d, want 0", v.RiskScore)
	}
}

func TestCheckURLBlocklistedHost(t *testing.T) {
	s := NewSecurityScanner()
	s.ReportPhishing("fakeuniswap.com")
	v := s.CheckURL("https://fakeuniswap.com/connect")
	if v.Verdict != "blocked" {
		t.Errorf("verdict = %q, want blocked", v.Verdict)
	}
	if v.RiskScore != 100 {
		t.Errorf("risk score = %d, want 100", v.RiskScore)
	}
}

func TestCheckURLBrandImpersonation(t *testing.T) {
	s := NewSecurityScanner()
	v := s.CheckURL("https://trustwallet-verify.example.com")
	if !hasReason(v, "brand_impersonation") {
		t.Errorf("brand impersonation was not detected: %+v", v.Reasons)
	}
	if v.Verdict != "blocked" {
		t.Errorf("verdict = %q, want blocked", v.Verdict)
	}
}

func TestCheckURLOfficialBrandDomainIsClean(t *testing.T) {
	s := NewSecurityScanner()
	for _, u := range []string{
		"https://trustwallet.com",
		"https://www.trustwallet.com/download",
		"https://metamask.io",
		"https://app.uniswap.org",
	} {
		v := s.CheckURL(u)
		if hasReason(v, "brand_impersonation") {
			t.Errorf("%s was reported as brand impersonation", u)
		}
	}
}

func TestCheckURLPunycodeHomograph(t *testing.T) {
	s := NewSecurityScanner()
	v := s.CheckURL("https://xn--trstwallet-3db.com")
	if !hasReason(v, "punycode_host") {
		t.Errorf("punycode host was not flagged: %+v", v.Reasons)
	}
}

func TestCheckURLCredentialHarvestKeyword(t *testing.T) {
	s := NewSecurityScanner()
	v := s.CheckURL("https://example.com/import-seed-phrase")
	if !hasReason(v, "credential_harvest_keyword") {
		t.Errorf("seed-phrase path was not flagged: %+v", v.Reasons)
	}
	if v.Verdict != "blocked" {
		t.Errorf("verdict = %q, want blocked", v.Verdict)
	}
}

func TestCheckURLIPLiteral(t *testing.T) {
	s := NewSecurityScanner()
	v := s.CheckURL("http://192.168.1.10/login")
	if !hasReason(v, "ip_literal_host") {
		t.Errorf("IP literal host was not flagged: %+v", v.Reasons)
	}
	// A private address is more serious than a public one.
	found := findingByID(v, "ip_literal_host")
	if found == nil || found.Severity != "high" {
		t.Errorf("private IP severity = %v, want high", found)
	}
}

func TestCheckURLUnparsable(t *testing.T) {
	s := NewSecurityScanner()
	v := s.CheckURL("://")
	if v.Verdict != "suspicious" {
		t.Errorf("verdict = %q, want suspicious for an unparsable URL", v.Verdict)
	}
}

func TestExtractHost(t *testing.T) {
	cases := map[string]string{
		"https://app.uniswap.org/swap": "app.uniswap.org",
		"http://www.example.com":       "example.com",
		"example.com":                  "example.com",
		"https://Example.COM/Path":     "example.com",
		"":                             "",
	}
	for in, want := range cases {
		if got := extractHost(in); got != want {
			t.Errorf("extractHost(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAddressBlocklist(t *testing.T) {
	s := NewSecurityScanner()
	addr := "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"
	if s.IsAddressBlocked(addr) {
		t.Error("an address was blocked before being added")
	}
	if err := s.AddAddressToBlocklist(addr); err != nil {
		t.Fatalf("AddAddressToBlocklist: %v", err)
	}
	if !s.IsAddressBlocked(addr) {
		t.Error("a blocked address was not reported as blocked")
	}
	// Matching must be case-insensitive: the same address written in different
	// case is the same account, so it must resolve to the same blocklist entry.
	if !s.IsAddressBlocked(strings.ToUpper(addr)) {
		t.Error("address matching is case-sensitive")
	}
	if err := s.AddAddressToBlocklist("nonsense"); err == nil {
		t.Error("AddAddressToBlocklist accepted a malformed address")
	}
}

func TestRemoveFromBlocklist(t *testing.T) {
	s := NewSecurityScanner()
	s.ReportPhishing("evil.example")
	if s.CheckURL("https://evil.example").Verdict != "blocked" {
		t.Fatal("host was not blocked")
	}
	s.RemoveFromBlocklist("evil.example")
	if s.CheckURL("https://evil.example").Verdict == "blocked" {
		t.Error("host remained blocked after removal")
	}
}

// --- helpers ---

func hasIssue(a ContractAnalysis, id string) bool {
	for _, f := range a.Issues {
		if f.ID == id {
			return true
		}
	}
	return false
}

func severityOfFinding(a ContractAnalysis, id string) string {
	for _, f := range a.Issues {
		if f.ID == id {
			return f.Severity
		}
	}
	return ""
}

func hasReason(v URLVerdict, id string) bool {
	return findingByID(v, id) != nil
}

func findingByID(v URLVerdict, id string) *Finding {
	for i := range v.Reasons {
		if v.Reasons[i].ID == id {
			return &v.Reasons[i]
		}
	}
	return nil
}
