// ============================================================================
// TIGERWALLET SECURITY CENTER
// Comprehensive security features including phishing detection, scam detection,
// wallet guardian, and fraud prevention
// ============================================================================

package security

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/sha3"
)

// SecurityAlert represents a security alert
type SecurityAlert struct {
	ID        string          `json:"id"`
	Type      AlertType       `json:"type"`
	Severity  AlertSeverity   `json:"severity"`
	Message   string          `json:"message"`
	Details   json.RawMessage `json:"details,omitempty"`
	Address   string          `json:"address,omitempty"`
	TxHash    string          `json:"tx_hash,omitempty"`
	CreatedAt int64           `json:"created_at"`
	Resolved  bool            `json:"resolved"`
}

// AlertType represents the type of alert
type AlertType string

const (
	AlertPhishing   AlertType = "phishing"
	AlertScam       AlertType = "scam"
	AlertMalware    AlertType = "malware"
	AlertSuspicious AlertType = "suspicious"
	AlertRisk       AlertType = "risk"
	AlertSafe       AlertType = "safe"
)

// AlertSeverity represents alert severity
type AlertSeverity string

const (
	SeverityLow      AlertSeverity = "low"
	SeverityMedium   AlertSeverity = "medium"
	SeverityHigh     AlertSeverity = "high"
	SeverityCritical AlertSeverity = "critical"
)

// RiskScore represents a risk assessment
type RiskScore struct {
	Address        string       `json:"address"`
	Score          int          `json:"score"` // 0-100
	Factors        []RiskFactor `json:"factors"`
	Recommendation string       `json:"recommendation"`
	CheckedAt      int64        `json:"checked_at"`
}

// RiskFactor represents a risk factor
type RiskFactor struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Weight      int    `json:"weight"`
}

// ScamDatabase holds known scam addresses populated at runtime from the
// backend security service / external scam-address feeds. It starts EMPTY
// (no fabricated entries) — an address is only flagged when a real report
// has been registered via RegisterScamAddress or loaded from PostgreSQL.
var ScamDatabase = map[string]ScamInfo{}

// ScamInfo holds information about a scam
type ScamInfo struct {
	Name       string `json:"name"`
	Category   string `json:"category"`
	ReportedAt int64  `json:"reported_at"`
	Website    string `json:"website,omitempty"`
	Socials    string `json:"socials,omitempty"`
}

// RegisterScamAddress adds a real scam report to the in-memory database.
// Used by the backend security service after a verified report is received.
func RegisterScamAddress(address string, info ScamInfo) {
	ScamDatabase[strings.ToLower(address)] = info
}

// WalletGuardian protects users from malicious transactions
type WalletGuardian struct {
	Rules     []ProtectionRule
	Alerts    []SecurityAlert
	Whitelist map[string]bool
	Blacklist map[string]bool
}

// ProtectionRule represents a transaction protection rule
type ProtectionRule struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Type      RuleType `json:"type"`
	Enabled   bool     `json:"enabled"`
	Threshold string   `json:"threshold,omitempty"`
	Addresses []string `json:"addresses,omitempty"`
}

// RuleType represents the type of rule
type RuleType string

const (
	RuleMaxValue     RuleType = "max_value"
	RuleBlacklist    RuleType = "blacklist"
	RuleWhitelist    RuleType = "whitelist"
	RuleUnverified   RuleType = "unverified_token"
	RuleHighSlippage RuleType = "high_slippage"
)

// NewWalletGuardian creates a new wallet guardian
func NewWalletGuardian() *WalletGuardian {
	return &WalletGuardian{
		Rules:     DefaultProtectionRules(),
		Alerts:    make([]SecurityAlert, 0),
		Whitelist: make(map[string]bool),
		Blacklist: make(map[string]bool),
	}
}

// DefaultProtectionRules returns default protection rules
func DefaultProtectionRules() []ProtectionRule {
	return []ProtectionRule{
		{
			ID:        "max_value_eth",
			Name:      "Max ETH per transaction",
			Type:      RuleMaxValue,
			Enabled:   true,
			Threshold: "10",
		},
		{
			ID:      "block_unknown",
			Name:    "Block unknown tokens",
			Type:    RuleUnverified,
			Enabled: true,
		},
		{
			ID:        "high_slippage",
			Name:      "High slippage warning",
			Type:      RuleHighSlippage,
			Enabled:   true,
			Threshold: "5",
		},
	}
}

// AnalyzeTransaction analyzes a transaction for risks.
func (wg *WalletGuardian) AnalyzeTransaction(tx *TransactionAnalysis) *SecurityAlert {
	if tx == nil {
		return nil
	}

	// Check against blacklist
	for _, addr := range tx.To {
		if wg.Blacklist[strings.ToLower(addr)] {
			return &SecurityAlert{
				ID:        generateAlertID(),
				Type:      AlertPhishing,
				Severity:  SeverityCritical,
				Message:   "Transaction to blacklisted address",
				Address:   addr,
				TxHash:    tx.Hash,
				CreatedAt: time.Now().Unix(),
			}
		}
	}

	// Check the value against the configured ceiling. The threshold is only
	// meaningful against a known token price; the native asset is assumed, and
	// a value that cannot be parsed is treated as unsafe rather than ignored.
	for _, rule := range wg.Rules {
		if rule.Type != RuleMaxValue || !rule.Enabled {
			continue
		}
		limit, err := strconv.ParseFloat(rule.Threshold, 64)
		if err != nil {
			continue
		}
		value, err := weiToFloat(tx.Value)
		if err != nil {
			return &SecurityAlert{
				ID:        generateAlertID(),
				Type:      AlertRisk,
				Severity:  SeverityHigh,
				Message:   fmt.Sprintf("Transaction value %q could not be parsed", tx.Value),
				TxHash:    tx.Hash,
				CreatedAt: time.Now().Unix(),
			}
		}
		if value > limit {
			return &SecurityAlert{
				ID:        generateAlertID(),
				Type:      AlertRisk,
				Severity:  SeverityHigh,
				Message:   fmt.Sprintf("Value %.6f exceeds the %.6f limit", value, limit),
				TxHash:    tx.Hash,
				CreatedAt: time.Now().Unix(),
			}
		}
	}

	// An approval for an unverified token grants spend authority to a contract
	// that has not been checked, which is the pattern drainers use.
	if isApprovalCalldata(tx.Data) && tx.Token != "" {
		if !wg.Whitelist[strings.ToLower(tx.Token)] {
			return &SecurityAlert{
				ID:        generateAlertID(),
				Type:      AlertRisk,
				Severity:  SeverityHigh,
				Message:   "Token approval for an unverified token",
				Address:   tx.Token,
				TxHash:    tx.Hash,
				CreatedAt: time.Now().Unix(),
			}
		}
	}

	return nil
}

// approvalSelector is the four-byte selector of approve(address,uint256).
const approvalSelector = "0x095ea7b3"

// isApprovalCalldata reports whether data is a call to approve(address,uint256).
func isApprovalCalldata(data string) bool {
	return len(data) >= 10 && strings.EqualFold(data[:10], approvalSelector)
}

// weiToFloat converts a wei quantity, decimal or 0x-hex, to ether.
func weiToFloat(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}

	var wei *big.Int
	var ok bool
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		wei, ok = new(big.Int).SetString(value[2:], 16)
	} else {
		wei, ok = new(big.Int).SetString(value, 10)
	}
	if !ok {
		return 0, fmt.Errorf("not a numeric value")
	}

	ether := new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(1e18))
	f, _ := ether.Float64()
	return f, nil
}

// TransactionAnalysis represents a transaction to analyze
type TransactionAnalysis struct {
	Hash    string   `json:"hash"`
	From    string   `json:"from"`
	To      []string `json:"to"`
	Value   string   `json:"value"`
	Data    string   `json:"data"`
	ChainID uint64   `json:"chain_id"`
	Token   string   `json:"token,omitempty"`
}

// CheckAddressRisk checks an address for risk factors
// CheckAddressRisk reports the risk of an address.
//
// The only signal available without an external intelligence feed is the local
// scam database. An address that is not in it is reported as unknown rather
// than as new: previously every address received a 30-point 'created recently'
// finding, a claim nothing had verified.
func CheckAddressRisk(address string) *RiskScore {
	score := &RiskScore{
		Address:   address,
		Score:     0,
		Factors:   make([]RiskFactor, 0),
		CheckedAt: time.Now().Unix(),
	}

	if info, ok := ScamDatabase[strings.ToLower(address)]; ok {
		score.Factors = append(score.Factors, RiskFactor{
			Type:        "known_scam",
			Description: fmt.Sprintf("Known scam: %s", info.Name),
			Weight:      100,
		})
		score.Score = 100
		score.Recommendation = "AVOID - This address is associated with a known scam"
		return score
	}

	if !isValidHexAddress(address) {
		score.Factors = append(score.Factors, RiskFactor{
			Type:        "malformed_address",
			Description: "The address is not a 20-byte hexadecimal address",
			Weight:      100,
		})
		score.Score = 100
		score.Recommendation = "AVOID - The address is not a valid account address"
		return score
	}

	score.Recommendation = "UNKNOWN - No report on file for this address; it has not been assessed"
	return score
}

// isValidHexAddress reports whether s is a 0x-prefixed 20-byte hex address.
func isValidHexAddress(s string) bool {
	if len(s) != 42 || !strings.HasPrefix(s, "0x") {
		return false
	}
	for i := 2; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// ValidateAddress validates an address isn't on a blacklist
func ValidateAddress(address string) (bool, string) {
	address = strings.ToLower(address)

	// Check scam database
	if _, ok := ScamDatabase[address]; ok {
		return false, "Address is on scam blacklist"
	}

	// Check common phishing patterns
	badPatterns := []string{
		"0x0000000000000000000000000000000000000000",
	}

	for _, pattern := range badPatterns {
		if address == pattern {
			return false, "Address has invalid pattern"
		}
	}

	return true, ""
}

// AntiPhishing provides phishing detection
type AntiPhishing struct {
	DomainDB map[string]PhishInfo
	SuffixDB map[string]bool
}

// PhishInfo holds phishing site information
type PhishInfo struct {
	Domain     string `json:"domain"`
	Target     string `json:"target"`
	ReportedAt int64  `json:"reported_at"`
	Type       string `json:"type"`
}

// NewAntiPhishing creates a new anti-phishing detector
func NewAntiPhishing() *AntiPhishing {
	return &AntiPhishing{
		DomainDB: make(map[string]PhishInfo),
		SuffixDB: map[string]bool{
			"-wallet.com":  true,
			"-eth.com":     true,
			"-binance.com": true,
			"-uniswap.com": true,
			"-opensea.io":  true,
		},
	}
}

// CheckDomain checks a domain for phishing
func (ap *AntiPhishing) CheckDomain(domain string) *SecurityAlert {
	domain = strings.ToLower(domain)

	// Check if in database
	if info, ok := ap.DomainDB[domain]; ok {
		return &SecurityAlert{
			ID:        generateAlertID(),
			Type:      AlertPhishing,
			Severity:  SeverityHigh,
			Message:   fmt.Sprintf("Known phishing domain: %s", info.Target),
			Details:   json.RawMessage(fmt.Sprintf(`{"domain": "%s", "type": "%s"}`, domain, info.Type)),
			CreatedAt: time.Now().Unix(),
		}
	}

	// Check suspicious suffixes
	for suffix := range ap.SuffixDB {
		if strings.HasSuffix(domain, suffix) {
			// Could be phishing - flag as suspicious
			return &SecurityAlert{
				ID:        generateAlertID(),
				Type:      AlertSuspicious,
				Severity:  SeverityMedium,
				Message:   fmt.Sprintf("Unusual domain suffix: %s", suffix),
				CreatedAt: time.Now().Unix(),
			}
		}
	}

	return nil
}

// SimulateTransaction estimates the effect of a transaction without a node.
//
// This is a local estimate, not an execution: it reports the value transfer and
// flags calldata that grants spend authority. Gas is deliberately left empty,
// because a real figure requires eth_estimateGas. Previously it reported a
// hardcoded 21000 gas and Success true for every input, which is a claim of
// safety that nothing had established.
func SimulateTransaction(tx *TransactionAnalysis) (*SimulationResult, error) {
	if tx == nil {
		return nil, fmt.Errorf("no transaction supplied")
	}

	result := &SimulationResult{
		Success:     false,
		Executed:    false,
		GasEstimate: "",
		TokenChange: make(map[string]string),
		Warnings:    make([]string, 0),
		Errors:      make([]string, 0),
	}

	value, err := weiToFloat(tx.Value)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("value %q is not numeric", tx.Value))
	} else if value > 0 {
		result.TokenChange["native"] = fmt.Sprintf("-%.18g", value)
	}

	if len(tx.Data) > 2 {
		if isApprovalCalldata(tx.Data) {
			result.Warnings = append(result.Warnings,
				"This call grants a contract authority to spend your tokens")
		} else {
			result.Warnings = append(result.Warnings,
				"This transaction calls a contract; its effects were not executed locally")
		}
	}

	result.Warnings = append(result.Warnings,
		"Gas and the resulting balance changes require eth_estimateGas and a node-side simulation")
	return result, nil
}

// SimulationResult represents transaction simulation results
type SimulationResult struct {
	Success     bool              `json:"success"`
	Executed    bool              `json:"executed"`
	GasEstimate string            `json:"gas_estimate"`
	TokenChange map[string]string `json:"token_change"`
	Warnings    []string          `json:"warnings"`
	Errors      []string          `json:"errors"`
}

// ScanContract reports the risk-bearing instructions present in contract code.
//
// Detection walks the bytecode and counts opcodes, skipping PUSH immediates so
// operand bytes are not misread as instructions. The previous implementation
// searched the hex text for ASCII substrings such as "selfdestruct" and
// "delegatecall", which never appear in bytecode, so it could not report
// anything.
func ScanContract(code string) *SecurityAlert {
	bytecode, err := decodeContractCode(code)
	if err != nil {
		return &SecurityAlert{
			ID:        generateAlertID(),
			Type:      AlertRisk,
			Severity:  SeverityMedium,
			Message:   fmt.Sprintf("Contract code could not be parsed: %v", err),
			CreatedAt: time.Now().Unix(),
		}
	}
	if len(bytecode) == 0 {
		return &SecurityAlert{
			ID:        generateAlertID(),
			Type:      AlertRisk,
			Severity:  SeverityMedium,
			Message:   "No contract code was supplied",
			CreatedAt: time.Now().Unix(),
		}
	}

	severity := SeverityMedium
	var findings []string
	for _, instruction := range scanOpcodes(bytecode) {
		findings = append(findings, instruction.description)
		if instruction.severity == SeverityCritical {
			severity = SeverityCritical
		}
	}
	if len(findings) == 0 {
		return nil
	}

	sort.Strings(findings)
	return &SecurityAlert{
		ID:        generateAlertID(),
		Type:      AlertRisk,
		Severity:  severity,
		Message:   strings.Join(findings, "; "),
		CreatedAt: time.Now().Unix(),
	}
}

// riskyInstruction describes what an opcode means for a contract's safety.
type riskyInstruction struct {
	description string
	severity    AlertSeverity
}

// riskyOpcodes maps opcodes to their meaning. A selector-based check is used
// for the upgrade pattern because the capability is expressed by a dispatched
// function rather than a single opcode.
var riskyOpcodes = map[byte]riskyInstruction{
	0xff: {"SELFDESTRUCT can destroy the contract", SeverityCritical},
	0xf4: {"DELEGATECALL runs external code in this contract's storage", SeverityHigh},
	0xf2: {"CALLCODE runs external code in this contract's storage", SeverityHigh},
	0xf5: {"CREATE2 deploys at a predictable address", SeverityMedium},
	0xf0: {"CREATE deploys a new contract", SeverityMedium},
}

// upgradeSelector is the four-byte selector of upgradeTo(address). It is
// derived from the signature so it cannot drift from the ABI.
var upgradeSelector = func() [4]byte {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte("upgradeTo(address)"))
	var out [4]byte
	copy(out[:], h.Sum(nil)[:4])
	return out
}()

// scanOpcodes returns the risky instructions in a contract, skipping PUSH data.
func scanOpcodes(code []byte) []riskyInstruction {
	var found []riskyInstruction
	seen := map[string]bool{}
	upgradeable := false

	for i := 0; i < len(code); {
		op := code[i]

		if instruction, ok := riskyOpcodes[op]; ok && !seen[instruction.description] {
			seen[instruction.description] = true
			found = append(found, instruction)
		}

		// A selector at a PUSH4 site indicates the function is dispatched.
		if op == 0x63 && i+4 < len(code) {
			if code[i+1] == upgradeSelector[0] && code[i+2] == upgradeSelector[1] &&
				code[i+3] == upgradeSelector[2] && code[i+4] == upgradeSelector[3] {
				upgradeable = true
			}
		}

		if op >= 0x60 && op <= 0x7f {
			i += 1 + int(op-0x60) + 1
			continue
		}
		i++
	}

	if upgradeable {
		found = append(found, riskyInstruction{
			description: "upgradeTo(address) allows the contract logic to be replaced",
			severity:    SeverityHigh,
		})
	}
	return found
}

// decodeContractCode converts 0x-prefixed hex to bytes.
func decodeContractCode(code string) ([]byte, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil
	}
	code = strings.TrimPrefix(strings.TrimPrefix(code, "0x"), "0X")
	if len(code)%2 != 0 {
		return nil, fmt.Errorf("odd-length hex string")
	}

	out := make([]byte, len(code)/2)
	for i := range out {
		hi, ok := hexNibble(code[i*2])
		if !ok {
			return nil, fmt.Errorf("invalid hex at offset %d", i*2)
		}
		lo, ok := hexNibble(code[i*2+1])
		if !ok {
			return nil, fmt.Errorf("invalid hex at offset %d", i*2+1)
		}
		out[i] = hi<<4 | lo
	}
	return out, nil
}

// hexNibble converts one hex character to its value.
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

// generateAlertID generates a unique alert ID.
//
// crypto/rand is used rather than a hash of the current time, because two
// alerts raised in the same nanosecond would otherwise share an ID.
func generateAlertID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// A failing entropy source must not silently produce a duplicate id.
		return fmt.Sprintf("alert-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
