package main

// analysis.go — real EVM bytecode analysis and contract risk assessment.
//
// The scanner reads the deployed bytecode and reports only what is present in
// it. Selectors are derived by hashing the canonical signatures rather than
// being pasted as literals, and the opcode walk skips PUSH immediates so that
// data bytes are not mistaken for instructions.

import (
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/sha3"
)

// EVM opcodes relevant to the analysis.
const (
	opPush1        = 0x60
	opPush32       = 0x7f
	opCreate       = 0xf0
	opCallCode     = 0xf2
	opDelegateCall = 0xf4
	opCreate2      = 0xf5
	opSelfDestruct = 0xff
)

// opcodeProfile summarizes the instructions in a contract's deployed code.
type opcodeProfile struct {
	Instructions int
	SelfDestruct int
	DelegateCall int
	CallCode     int
	Create       int
	Create2      int
	// DataBytes counts bytes consumed as PUSH immediates. A contract whose
	// code is mostly data is usually a proxy or embeds an opaque payload.
	DataBytes int
}

// walkOpcodes counts instructions, skipping PUSH immediates so that operand
// bytes are never interpreted as opcodes.
func walkOpcodes(code []byte) opcodeProfile {
	var p opcodeProfile
	for i := 0; i < len(code); {
		op := code[i]
		p.Instructions++

		switch op {
		case opSelfDestruct:
			p.SelfDestruct++
		case opDelegateCall:
			p.DelegateCall++
		case opCallCode:
			p.CallCode++
		case opCreate:
			p.Create++
		case opCreate2:
			p.Create2++
		}

		if op >= opPush1 && op <= opPush32 {
			operand := int(op-opPush1) + 1
			p.DataBytes += operand
			i += 1 + operand
			continue
		}
		i++
	}
	return p
}

// selector returns the four-byte function selector for a canonical signature.
func selector(signature string) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(signature))
	return h.Sum(nil)[:4]
}

// wellKnownSelectors maps a human-readable capability to its signature. The
// selectors are computed from the signatures at startup.
var wellKnownSelectors = map[string]string{
	"transfer(address,uint256)":  "erc20_transfer",
	"approve(address,uint256)":   "erc20_approve",
	"balanceOf(address)":         "erc20_balance_of",
	"totalSupply()":              "erc20_total_supply",
	"mint(address,uint256)":      "mint",
	"mint(uint256)":              "mint_single_arg",
	"burn(uint256)":              "burn",
	"owner()":                    "ownable_owner",
	"transferOwnership(address)": "ownable_transfer",
	"supportsInterface(bytes4)":  "erc165",
	"safeTransferFrom(address,address,uint256,bytes)": "erc721_safe_transfer",
	"setBlacklist(address,bool)":                      "blacklist_control",
	// Tether's USDT does not expose setBlacklist; it uses these instead, so the
	// blacklist signal must cover both naming conventions.
	"addBlackList(address)":      "blacklist_control",
	"destroyBlackFunds(address)": "blacklist_control",
	"isBlackListed(address)":     "blacklist_control",
	"setFees(uint256)":           "fee_control",
	"pause()":                    "pausable",
}

// selectorTable maps each computed four-byte selector to the capability it
// indicates. It is keyed by selector rather than by capability because several
// signatures can express the same capability; keying by capability would let
// the last signature silently overwrite the others.
var selectorTable = func() map[[4]byte]string {
	table := make(map[[4]byte]string, len(wellKnownSelectors))
	for signature, capability := range wellKnownSelectors {
		var key [4]byte
		copy(key[:], selector(signature))
		table[key] = capability
	}
	return table
}()

// detectSelectors reports which known capabilities appear in the code. A
// capability counts as present when its selector follows a PUSH4 instruction,
// which is how the Solidity dispatcher loads it.
func detectSelectors(code []byte) map[string]bool {
	found := map[string]bool{}
	for i := 0; i+4 < len(code); i++ {
		if code[i] != opPush1+3 { // PUSH4
			continue
		}
		var key [4]byte
		copy(key[:], code[i+1:i+5])
		if capability, ok := selectorTable[key]; ok {
			found[capability] = true
		}
	}
	return found
}

// ContractAnalysis is the result of analyzing deployed bytecode.
type ContractAnalysis struct {
	Address     string
	IsContract  bool
	CodeSize    int
	OpcodeCount int
	Selectors   map[string]bool
	IsToken     bool
	IsNFT       bool
	IsProxy     bool
	Issues      []Finding
	Score       int
}

// Finding is one risk observation about a contract or URL.
type Finding struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	CWEID       string `json:"cwe_id,omitempty"`
}

// severityWeight is the score deducted for a finding of the given severity.
func severityWeight(severity string) int {
	switch severity {
	case "critical":
		return 50
	case "high":
		return 30
	case "medium":
		return 15
	case "low":
		return 5
	default:
		return 0
	}
}

// analyzeBytecode assesses deployed contract code. An empty code slice means
// the address is an externally owned account, which carries no contract risk.
func analyzeBytecode(address string, code []byte) ContractAnalysis {
	a := ContractAnalysis{Address: address, Score: 100}
	if len(code) == 0 {
		a.IsContract = false
		return a
	}

	a.IsContract = true
	a.CodeSize = len(code)

	profile := walkOpcodes(code)
	a.OpcodeCount = profile.Instructions
	a.Selectors = detectSelectors(code)

	a.IsToken = a.Selectors["erc20_transfer"] && a.Selectors["erc20_balance_of"]
	a.IsNFT = a.Selectors["erc721_safe_transfer"] || a.Selectors["erc165"]

	// A delegatecall to an address held in storage, combined with a small
	// dispatcher, is the standard upgradeable-proxy shape. It is not a
	// vulnerability by itself, but the implementation can be swapped, so the
	// holder of the admin key can change behaviour after a user approves it.
	if profile.DelegateCall > 0 {
		a.IsProxy = true
		a.Issues = append(a.Issues, Finding{
			ID:          "delegatecall_present",
			Severity:    "medium",
			Description: fmt.Sprintf("Deployed code performs %d DELEGATECALL operation(s); execution can be redirected to another contract's code running against this contract's storage.", profile.DelegateCall),
			CWEID:       "CWE-829",
		})
	}

	if profile.CallCode > 0 {
		a.Issues = append(a.Issues, Finding{
			ID:          "callcode_present",
			Severity:    "high",
			Description: "Deployed code uses CALLCODE, a deprecated instruction that executes foreign code in this contract's context. It is no longer emitted by current compilers and its presence indicates unusual construction.",
			CWEID:       "CWE-829",
		})
	}

	if profile.SelfDestruct > 0 {
		a.Issues = append(a.Issues, Finding{
			ID:          "selfdestruct_present",
			Severity:    "high",
			Description: "Deployed code contains SELFDESTRUCT. If reachable by an unauthorized caller the contract's code and balance can be destroyed.",
			CWEID:       "CWE-284",
		})
	}

	if profile.Create2 > 0 {
		a.Issues = append(a.Issues, Finding{
			ID:          "create2_present",
			Severity:    "low",
			Description: "Deployed code uses CREATE2, which allows a contract to be redeployed at a predictable address. Combined with SELFDESTRUCT this enables a metamorphic contract whose code changes between deployments.",
			CWEID:       "CWE-670",
		})
	}

	if profile.Create2 > 0 && profile.SelfDestruct > 0 {
		a.Issues = append(a.Issues, Finding{
			ID:          "metamorphic_contract",
			Severity:    "critical",
			Description: "Deployed code combines CREATE2 with SELFDESTRUCT. The contract can be destroyed and redeployed at the same address with different code, so any approval granted to it can be reused by unrelated logic.",
			CWEID:       "CWE-670",
		})
	}

	if a.Selectors["blacklist_control"] {
		a.Issues = append(a.Issues, Finding{
			ID:          "transfer_restriction",
			Severity:    "high",
			Description: "The contract exposes a blacklist control. The operator can block individual holders from transferring, which means a balance can become unsellable at the operator's discretion.",
			CWEID:       "CWE-284",
		})
	}

	if a.Selectors["fee_control"] && a.IsToken {
		a.Issues = append(a.Issues, Finding{
			ID:          "mutable_fee",
			Severity:    "medium",
			Description: "The token exposes a fee setter. Transfer fees can be changed after purchase, including raised to a level that makes selling uneconomic.",
			CWEID:       "CWE-284",
		})
	}

	if a.Selectors["mint"] || a.Selectors["mint_single_arg"] {
		a.Issues = append(a.Issues, Finding{
			ID:          "mintable_supply",
			Severity:    "medium",
			Description: "The contract exposes a mint function. The supply is not fixed, so holdings can be diluted if the mint authority is retained.",
			CWEID:       "CWE-284",
		})
	}

	// Code that is mostly data is a strong signal of a minimal proxy whose
	// behaviour lives elsewhere.
	if profile.Instructions > 0 {
		ratio := float64(profile.DataBytes) / float64(len(code))
		if ratio > 0.6 && profile.DelegateCall > 0 {
			a.IsProxy = true
			a.Issues = append(a.Issues, Finding{
				ID:          "minimal_proxy",
				Severity:    "low",
				Description: "Most of the deployed code is embedded data rather than instructions, and DELEGATECALL is present, which is the shape of a minimal proxy. The implementation contract must be audited separately.",
				CWEID:       "CWE-829",
			})
		}
	}

	if len(code) < 64 {
		a.Issues = append(a.Issues, Finding{
			ID:          "tiny_contract",
			Severity:    "low",
			Description: fmt.Sprintf("Deployed code is only %d bytes. Contracts this small rarely implement a full token or vault and usually forward to another address.", len(code)),
			CWEID:       "CWE-670",
		})
	}

	for _, f := range a.Issues {
		a.Score -= severityWeight(f.Severity)
	}
	if a.Score < 0 {
		a.Score = 0
	}
	return a
}

// decodeHexCode decodes a JSON-RPC hex quantity such as "0x6080" into bytes.
func decodeHexCode(raw string) ([]byte, error) {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "0x")
	value = strings.TrimPrefix(value, "0X")
	if value == "" {
		return nil, nil
	}
	if len(value)%2 != 0 {
		value = "0" + value
	}
	out := make([]byte, len(value)/2)
	for i := 0; i < len(out); i++ {
		hi, ok := hexNibble(value[i*2])
		if !ok {
			return nil, fmt.Errorf("invalid hex byte at offset %d", i*2)
		}
		lo, ok := hexNibble(value[i*2+1])
		if !ok {
			return nil, fmt.Errorf("invalid hex byte at offset %d", i*2+1)
		}
		out[i] = hi<<4 | lo
	}
	return out, nil
}

// hexNibble converts a single hex character to its value.
func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}

// isHexAddress reports whether s is a 0x-prefixed 20-byte hex address.
func isHexAddress(s string) bool {
	if !strings.HasPrefix(s, "0x") || len(s) != 42 {
		return false
	}
	for i := 2; i < len(s); i++ {
		if _, ok := hexNibble(s[i]); !ok {
			return false
		}
	}
	return true
}

// normalizeAddress lower-cases a validated address so lookups are consistent.
func normalizeAddress(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// hexQuantityToBigInt parses a JSON-RPC hex quantity such as "0x1".
func hexQuantityToBigInt(raw string) (*big.Int, error) {
	value := strings.TrimSpace(raw)
	if !strings.HasPrefix(value, "0x") && !strings.HasPrefix(value, "0X") {
		return nil, fmt.Errorf("not a hex quantity: %q", raw)
	}
	n, ok := new(big.Int).SetString(value[2:], 16)
	if !ok {
		return nil, fmt.Errorf("not a hex quantity: %q", raw)
	}
	return n, nil
}
