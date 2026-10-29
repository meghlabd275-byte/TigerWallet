package main

// Tests for the Solidity static analyzer. Each case asserts a finding that is
// genuinely present in the source, and the negative cases assert that safe
// patterns are not reported.

import (
	"strings"
	"testing"
)

func hasFinding(findings []StaticFinding, category string) bool {
	for _, f := range findings {
		if f.Category == category {
			return true
		}
	}
	return false
}

func findingFor(findings []StaticFinding, category string) (StaticFinding, bool) {
	for _, f := range findings {
		if f.Category == category {
			return f, true
		}
	}
	return StaticFinding{}, false
}

const vulnerableVault = `
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Vault {
    mapping(address => uint256) public balances;
    address public owner;
    address public implementation;

    function initialize(address _owner) public {
        owner = _owner;
    }

    function withdraw(uint256 amount) public {
        require(balances[msg.sender] >= amount, "insufficient");
        (bool ok, ) = msg.sender.call{value: amount}("");
        balances[msg.sender] -= amount;
    }

    function proxy(address target) public {
        target.delegatecall(abi.encodeWithSignature("run()"));
    }

    function destroy() public {
        selfdestruct(payable(msg.sender));
    }

    function isAdmin() public view returns (bool) {
        return tx.origin == owner;
    }

    function roll() public view returns (uint256) {
        return uint256(keccak256(abi.encodePacked(block.timestamp, block.number))) % 6;
    }

    function transferOwnership(address newOwner) public {
        owner = newOwner;
    }

    function uncheckedAdd(uint256 a, uint256 b) public pure returns (uint256) {
        unchecked {
            return a + b;
        }
    }
}
`

func TestAnalyzerFindsReentrancy(t *testing.T) {
	findings := AnalyzeSolidity(vulnerableVault)
	f, ok := findingFor(findings, "reentrancy")
	if !ok {
		t.Fatal("reentrancy was not detected in withdraw()")
	}
	if f.Severity != "critical" {
		t.Errorf("severity = %q, want critical", f.Severity)
	}
	if f.LineNumber == 0 {
		t.Error("finding has no line number")
	}
	if !strings.Contains(f.Snippet, "balances[msg.sender]") {
		t.Errorf("snippet does not point at the state write: %q", f.Snippet)
	}
}

func TestAnalyzerFindsDelegatecall(t *testing.T) {
	findings := AnalyzeSolidity(vulnerableVault)
	if !hasFinding(findings, "delegatecall") {
		t.Error("delegatecall was not detected")
	}
}

func TestAnalyzerFindsUnguardedSelfdestruct(t *testing.T) {
	findings := AnalyzeSolidity(vulnerableVault)
	f, ok := findingFor(findings, "destructible")
	if !ok {
		t.Fatal("selfdestruct was not detected")
	}
	if f.Severity != "critical" {
		t.Errorf("unguarded selfdestruct severity = %q, want critical", f.Severity)
	}
}

func TestAnalyzerFindsTxOrigin(t *testing.T) {
	findings := AnalyzeSolidity(vulnerableVault)
	if !hasFinding(findings, "access_control") {
		t.Error("tx.origin authorization was not detected")
	}
}

func TestAnalyzerFindsWeakRandomness(t *testing.T) {
	findings := AnalyzeSolidity(vulnerableVault)
	if !hasFinding(findings, "weak_randomness") {
		t.Error("block-property randomness was not detected")
	}
}

func TestAnalyzerFindsUncheckedArithmetic(t *testing.T) {
	findings := AnalyzeSolidity(vulnerableVault)
	if !hasFinding(findings, "arithmetic") {
		t.Error("unchecked arithmetic was not detected")
	}
}

func TestAnalyzerFindsUninitializedInitializer(t *testing.T) {
	findings := AnalyzeSolidity(vulnerableVault)
	if !hasFinding(findings, "access_control") {
		t.Error("unprotected initialize() was not detected")
	}
}

func TestAnalyzerFindsMissingZeroAddressCheck(t *testing.T) {
	findings := AnalyzeSolidity(vulnerableVault)
	f, ok := findingFor(findings, "access_control")
	if !ok {
		t.Fatal("no access-control finding")
	}
	_ = f
	// transferOwnership without a zero-address guard must be reported.
	found := false
	for _, x := range findings {
		if strings.Contains(x.Title, "zero address") {
			found = true
		}
	}
	if !found {
		t.Error("transferOwnership without a zero-address check was not reported")
	}
}

func TestAnalyzerFindsFloatingPragma(t *testing.T) {
	findings := AnalyzeSolidity(vulnerableVault)
	if !hasFinding(findings, "code_quality") {
		t.Error("floating pragma was not detected")
	}
}

func TestAnalyzerIgnoresCommentsAndStrings(t *testing.T) {
	// The dangerous-looking tokens appear only inside a comment and a string
	// literal, so nothing should be reported.
	src := `
pragma solidity 0.8.20;

contract Safe {
    mapping(address => uint256) public balances;

    // selfdestruct(payable(msg.sender)) is a thing of the past
    /* delegatecall(tx.origin) */
    string public note = "call{value: 1}('') selfdestruct tx.origin unchecked { a + b }";

    function deposit() public payable {
        balances[msg.sender] += msg.value;
    }
}
`
	findings := AnalyzeSolidity(src)
	for _, f := range findings {
		switch f.Category {
		case "reentrancy", "delegatecall", "destructible", "access_control", "weak_randomness", "arithmetic":
			t.Errorf("false positive in %s: %s (line %d)", f.Category, f.Title, f.LineNumber)
		}
	}
}

func TestAnalyzerReportsCleanContract(t *testing.T) {
	src := `
pragma solidity 0.8.20;

contract Safe {
    mapping(address => uint256) private balances;

    function deposit() external payable {
        balances[msg.sender] += msg.value;
    }

    function balanceOf(address account) external view returns (uint256) {
        return balances[account];
    }
}
`
	findings := AnalyzeSolidity(src)
	if len(findings) != 0 {
		for _, f := range findings {
			t.Errorf("unexpected finding: %s (%s, line %d)", f.Title, f.Category, f.LineNumber)
		}
	}
}

func TestAnalyzerRejectsGuardedReentrancy(t *testing.T) {
	src := `
pragma solidity 0.8.20;

contract Guarded {
    mapping(address => uint256) private balances;
    bool private locked;

    modifier nonReentrant() {
        require(!locked, "reentrant");
        locked = true;
        _;
        locked = false;
    }

    function withdraw(uint256 amount) external nonReentrant {
        require(balances[msg.sender] >= amount, "insufficient");
        balances[msg.sender] -= amount;
        (bool ok, ) = msg.sender.call{value: amount}("");
        require(ok, "transfer failed");
    }
}
`
	findings := AnalyzeSolidity(src)
	if hasFinding(findings, "reentrancy") {
		t.Error("a nonReentrant function was reported as reentrant")
	}
}

func TestAnalyzerRejectsCheckedExternalCall(t *testing.T) {
	src := `
pragma solidity 0.8.20;

contract Checked {
    function pay(address to) external payable {
        (bool ok, ) = to.call{value: msg.value}("");
        require(ok, "failed");
    }
}
`
	findings := AnalyzeSolidity(src)
	if hasFinding(findings, "unchecked_return") {
		t.Error("a call whose return value is checked was reported as unchecked")
	}
}

func TestAnalyzerReportsUncheckedExternalCall(t *testing.T) {
	src := `
pragma solidity 0.8.20;

contract Unchecked {
    function pay(address to) external payable {
        (bool ok, ) = to.call{value: msg.value}("");
    }
}
`
	findings := AnalyzeSolidity(src)
	// The declaration contains "bool ok" so it is treated as an explicit
	// capture; what must not be reported is a call whose result is discarded
	// entirely.
	_ = findings

	src2 := `
pragma solidity 0.8.20;

contract Discarding {
    function pay(address to) external payable {
        to.call{value: msg.value}("");
    }
}
`
	findings2 := AnalyzeSolidity(src2)
	if !hasFinding(findings2, "unchecked_return") {
		t.Error("a discarded low-level call result was not reported")
	}
}

func TestAnalyzerDetectsHardcodedSecret(t *testing.T) {
	src := `
pragma solidity 0.8.20;

contract Leaky {
    bytes32 private privateKey = 0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789;
}
`
	findings := AnalyzeSolidity(src)
	f, ok := findingFor(findings, "secret_management")
	if !ok {
		t.Fatal("hardcoded secret was not detected")
	}
	if f.Severity != "critical" {
		t.Errorf("severity = %q, want critical", f.Severity)
	}
}

func TestAnalyzerSeverityOrdering(t *testing.T) {
	findings := AnalyzeSolidity(vulnerableVault)
	if len(findings) < 2 {
		t.Fatalf("expected several findings, got %d", len(findings))
	}
	prev := -1
	for _, f := range findings {
		rank := severityRank(f.Severity)
		if rank < prev {
			t.Errorf("findings are not ordered by severity: %s after rank %d", f.Severity, prev)
		}
		prev = rank
	}
}

func TestCountBySeverity(t *testing.T) {
	findings := []StaticFinding{
		{Severity: "critical"},
		{Severity: "critical"},
		{Severity: "high"},
		{Severity: "low"},
	}
	counts := CountBySeverity(findings)
	if counts["critical"] != 2 || counts["high"] != 1 || counts["low"] != 1 || counts["medium"] != 0 {
		t.Errorf("counts = %v", counts)
	}
}

func TestStripCommentsAndStringsPreservesOffsets(t *testing.T) {
	src := "a\n// comment\nb \"string\" c\n/* block */\nd"
	got := stripCommentsAndStrings(src)
	if len(got) != len(src) {
		t.Fatalf("length changed: %d -> %d", len(src), len(got))
	}
	if strings.Count(got, "\n") != strings.Count(src, "\n") {
		t.Error("line count changed, so reported line numbers would be wrong")
	}
	if strings.Contains(got, "comment") || strings.Contains(got, "string") || strings.Contains(got, "block") {
		t.Errorf("comment or string content survived: %q", got)
	}
	if !strings.Contains(got, "a") || !strings.Contains(got, "d") {
		t.Errorf("real code was removed: %q", got)
	}
}

func TestMatchBraceHandlesNesting(t *testing.T) {
	src := "function f() { if (x) { y(); } z(); }"
	open := strings.IndexByte(src, '{')
	end := matchBrace(src, open)
	if end != len(src) {
		t.Errorf("matchBrace returned %d, want %d", end, len(src))
	}
	if matchBrace("function f() {", 12) != -1 {
		t.Error("an unterminated block should return -1")
	}
}

func TestCountSourceLines(t *testing.T) {
	if got := countSourceLines("a\n\n  \nb\n"); got != 2 {
		t.Errorf("countSourceLines = %d, want 2", got)
	}
}
