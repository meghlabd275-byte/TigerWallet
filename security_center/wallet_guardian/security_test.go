package security

// Tests for the security centre. The important cases are that a check which
// cannot establish a fact reports 'unknown' rather than a fabricated verdict,
// and that contract scanning reads bytecode rather than searching for ASCII.

import (
	"encoding/hex"
	"strings"
	"testing"

	"golang.org/x/crypto/sha3"
)

// hexSelectorForTest returns the four-byte selector of a signature as hex.
func hexSelectorForTest(signature string) string {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(signature))
	return hex.EncodeToString(h.Sum(nil)[:4])
}

func TestIsApprovalCalldata(t *testing.T) {
	// approve(address,uint256) with a spender and an amount.
	approve := "0x095ea7b3" + strings.Repeat("0", 24) + "dac17f958d2ee523a2206206994597c13d831ec7" +
		strings.Repeat("f", 64)
	if !isApprovalCalldata(approve) {
		t.Error("an approve() call was not recognised")
	}
	if !isApprovalCalldata("0x095EA7B3" + strings.Repeat("0", 128)) {
		t.Error("an approve() call with upper-case hex was not recognised")
	}
	for _, notApproval := range []string{"", "0x", "0x095ea7b", "0xa9059cbb" + strings.Repeat("0", 128)} {
		if isApprovalCalldata(notApproval) {
			t.Errorf("isApprovalCalldata(%q) = true, want false", notApproval)
		}
	}
}

func TestWeiToFloat(t *testing.T) {
	cases := map[string]float64{
		"":                    0,
		"0":                   0,
		"1000000000000000000": 1,
		"0xde0b6b3a7640000":   1,
		"500000000000000000":  0.5,
	}
	for input, want := range cases {
		got, err := weiToFloat(input)
		if err != nil {
			t.Errorf("weiToFloat(%q): %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("weiToFloat(%q) = %v, want %v", input, got, want)
		}
	}
	for _, bad := range []string{"abc", "0xzz", "1.5"} {
		if _, err := weiToFloat(bad); err == nil {
			t.Errorf("weiToFloat(%q) accepted a non-integer value", bad)
		}
	}
}

func TestAnalyzeTransactionEnforcesValueLimit(t *testing.T) {
	wg := NewWalletGuardian()
	// The default ceiling is 10 ether.
	over := &TransactionAnalysis{Value: "11000000000000000000"}
	if alert := wg.AnalyzeTransaction(over); alert == nil {
		t.Fatal("a transfer above the value limit was not flagged")
	}

	under := &TransactionAnalysis{Value: "1000000000000000000"}
	if alert := wg.AnalyzeTransaction(under); alert != nil {
		t.Errorf("a transfer below the value limit was flagged: %+v", alert)
	}
}

func TestAnalyzeTransactionRejectsUnparseableValue(t *testing.T) {
	wg := NewWalletGuardian()
	// An unparseable value must not silently pass the limit check.
	alert := wg.AnalyzeTransaction(&TransactionAnalysis{Value: "not-a-number"})
	if alert == nil {
		t.Fatal("an unparseable value was not flagged")
	}
	if !strings.Contains(alert.Message, "could not be parsed") {
		t.Errorf("message = %q, want it to explain the parse failure", alert.Message)
	}
}

func TestAnalyzeTransactionFlagsUnverifiedApproval(t *testing.T) {
	wg := NewWalletGuardian()
	tx := &TransactionAnalysis{
		Data:  "0x095ea7b3" + strings.Repeat("0", 128),
		Token: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
	}
	if alert := wg.AnalyzeTransaction(tx); alert == nil {
		t.Fatal("an approval for an unverified token was not flagged")
	}

	// Once the token is whitelisted the approval is expected.
	wg.Whitelist["0xdac17f958d2ee523a2206206994597c13d831ec7"] = true
	if alert := wg.AnalyzeTransaction(tx); alert != nil {
		t.Errorf("an approval for a whitelisted token was flagged: %+v", alert)
	}
}

func TestAnalyzeTransactionFlagsBlacklistedRecipient(t *testing.T) {
	wg := NewWalletGuardian()
	wg.Blacklist["0xbad"] = true
	alert := wg.AnalyzeTransaction(&TransactionAnalysis{To: []string{"0xBAD"}})
	if alert == nil {
		t.Fatal("a transfer to a blacklisted address was not flagged")
	}
	if alert.Severity != SeverityCritical {
		t.Errorf("severity = %v, want critical", alert.Severity)
	}
}

func TestCheckAddressRiskDoesNotInventFindings(t *testing.T) {
	// An address with no report must be reported as unknown, not as new. The
	// previous implementation returned 30 for every address.
	score := CheckAddressRisk("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	if score.Score != 0 {
		t.Errorf("score = %d, want 0 for an unreported address", score.Score)
	}
	if len(score.Factors) != 0 {
		t.Errorf("factors = %+v, want none for an unreported address", score.Factors)
	}
	if !strings.HasPrefix(score.Recommendation, "UNKNOWN") {
		t.Errorf("recommendation = %q, want it to state the address is unassessed", score.Recommendation)
	}
}

func TestCheckAddressRiskFlagsKnownScam(t *testing.T) {
	const scam = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	RegisterScamAddress(scam, ScamInfo{Name: "Test Scam"})
	defer delete(ScamDatabase, scam)

	score := CheckAddressRisk(scam)
	if score.Score != 100 {
		t.Errorf("score = %d, want 100 for a known scam", score.Score)
	}
}

func TestCheckAddressRiskFlagsMalformedAddress(t *testing.T) {
	score := CheckAddressRisk("not-an-address")
	if score.Score != 100 {
		t.Errorf("score = %d, want 100 for a malformed address", score.Score)
	}
}

func TestIsValidHexAddress(t *testing.T) {
	if !isValidHexAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7") {
		t.Error("a valid address was rejected")
	}
	for _, bad := range []string{"", "0x", "0xdAC17F958D2ee523a2206206994597C13D831ec", "0xZZZ17F958D2ee523a2206206994597C13D831ec7"} {
		if isValidHexAddress(bad) {
			t.Errorf("isValidHexAddress(%q) = true, want false", bad)
		}
	}
}

func TestSimulateTransactionDoesNotClaimSuccess(t *testing.T) {
	// A local estimate has not executed anything, so it must not report
	// Success true or a fabricated gas figure.
	result, err := SimulateTransaction(&TransactionAnalysis{
		Value: "1000000000000000000",
		Data:  "0xa9059cbb" + strings.Repeat("0", 128),
	})
	if err != nil {
		t.Fatalf("SimulateTransaction: %v", err)
	}
	if result.Success {
		t.Error("Success was reported true without executing the transaction")
	}
	if result.Executed {
		t.Error("Executed was reported true without a node-side simulation")
	}
	if result.GasEstimate != "" {
		t.Errorf("GasEstimate = %q, want empty without eth_estimateGas", result.GasEstimate)
	}
	if result.TokenChange["native"] != "-1" {
		t.Errorf("native change = %q, want -1", result.TokenChange["native"])
	}
	if len(result.Warnings) == 0 {
		t.Error("no warning was issued about the limits of the estimate")
	}
}

func TestSimulateTransactionFlagsApproval(t *testing.T) {
	result, err := SimulateTransaction(&TransactionAnalysis{
		Data: "0x095ea7b3" + strings.Repeat("0", 128),
	})
	if err != nil {
		t.Fatalf("SimulateTransaction: %v", err)
	}
	joined := strings.Join(result.Warnings, " ")
	if !strings.Contains(joined, "authority to spend") {
		t.Errorf("warnings = %v, want the approval risk stated", result.Warnings)
	}
}

func TestSimulateTransactionRejectsNil(t *testing.T) {
	if _, err := SimulateTransaction(nil); err == nil {
		t.Error("SimulateTransaction accepted a nil transaction")
	}
}

func TestSimulateTransactionReportsBadValue(t *testing.T) {
	result, err := SimulateTransaction(&TransactionAnalysis{Value: "nonsense"})
	if err != nil {
		t.Fatalf("SimulateTransaction: %v", err)
	}
	if len(result.Errors) == 0 {
		t.Error("a non-numeric value was not reported as an error")
	}
}

func TestScanContractDetectsOpcodes(t *testing.T) {
	// PUSH1 0x00, SELFDESTRUCT, DELEGATECALL.
	alert := ScanContract("0x6000fff4")
	if alert == nil {
		t.Fatal("SELFDESTRUCT and DELEGATECALL were not detected")
	}
	if alert.Severity != SeverityCritical {
		t.Errorf("severity = %v, want critical because of SELFDESTRUCT", alert.Severity)
	}
	if !strings.Contains(alert.Message, "SELFDESTRUCT") {
		t.Errorf("message = %q, want SELFDESTRUCT named", alert.Message)
	}
	if !strings.Contains(alert.Message, "DELEGATECALL") {
		t.Errorf("message = %q, want DELEGATECALL named", alert.Message)
	}
}

func TestScanContractIgnoresOpcodesInPushData(t *testing.T) {
	// PUSH2 0xfff4 embeds the bytes of SELFDESTRUCT and DELEGATECALL as
	// immediate data. They are not executed and must not be reported.
	if alert := ScanContract("0x61fff4"); alert != nil {
		t.Errorf("opcodes inside PUSH data were reported as instructions: %+v", alert)
	}
}

func TestScanContractDetectsUpgradeSelector(t *testing.T) {
	// PUSH4 <upgradeTo(address) selector>. The selector is derived from the
	// signature so it matches what a compiler emits.
	code := "0x63" + hexSelectorForTest("upgradeTo(address)")
	alert := ScanContract(code)
	if alert == nil {
		t.Fatal("an upgradeable contract was not detected")
	}
	if !strings.Contains(alert.Message, "upgradeTo") {
		t.Errorf("message = %q, want the upgrade capability named", alert.Message)
	}
}

func TestScanContractReportsCleanCode(t *testing.T) {
	// PUSH1 0x01, PUSH1 0x02, ADD, STOP - no risky instruction.
	if alert := ScanContract("0x600160020100"); alert != nil {
		t.Errorf("clean code produced an alert: %+v", alert)
	}
}

func TestScanContractRejectsBadInput(t *testing.T) {
	if alert := ScanContract("0xzz"); alert == nil {
		t.Error("invalid hex was not reported")
	}
	if alert := ScanContract(""); alert == nil {
		t.Error("empty code was not reported")
	}
}

func TestDecodeContractCode(t *testing.T) {
	code, err := decodeContractCode("0x6000")
	if err != nil || len(code) != 2 {
		t.Fatalf("decodeContractCode = %x, %v; want 2 bytes", code, err)
	}
	if code, err := decodeContractCode(""); err != nil || code != nil {
		t.Errorf("empty = %v, %v; want nil, nil", code, err)
	}
	if _, err := decodeContractCode("0x123"); err == nil {
		t.Error("an odd-length hex string was accepted")
	}
}

func TestGenerateAlertIDIsUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := generateAlertID()
		if seen[id] {
			t.Fatalf("generateAlertID produced a duplicate: %s", id)
		}
		seen[id] = true
	}
}

func TestCheckDomainDetectsSuspiciousSuffix(t *testing.T) {
	ap := NewAntiPhishing()
	if alert := ap.CheckDomain("trust-wallet.com"); alert == nil {
		t.Error("a suspicious suffix was not flagged")
	}
	if alert := ap.CheckDomain("trustwallet.com"); alert != nil {
		t.Errorf("a normal domain was flagged: %+v", alert)
	}
}
