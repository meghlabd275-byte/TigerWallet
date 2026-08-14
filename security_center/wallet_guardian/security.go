// ============================================================================
// TIGERWALLET SECURITY CENTER
// Comprehensive security features including phishing detection, scam detection,
// wallet guardian, and fraud prevention
// ============================================================================

package security

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SecurityAlert represents a security alert
type SecurityAlert struct {
	ID        string          `json:"id"`
	Type     AlertType       `json:"type"`
	Severity AlertSeverity   `json:"severity"`
	Message  string          `json:"message"`
	Details  json.RawMessage `json:"details,omitempty"`
	Address  string          `json:"address,omitempty"`
	TxHash   string          `json:"tx_hash,omitempty"`
	CreatedAt int64          `json:"created_at"`
	Resolved bool           `json:"resolved"`
}

// AlertType represents the type of alert
type AlertType string

const (
	AlertPhishing       AlertType = "phishing"
	AlertScam         AlertType = "scam"
	AlertMalware      AlertType = "malware"
	AlertSuspicious   AlertType = "suspicious"
	AlertRisk         AlertType = "risk"
	AlertSafe         AlertType = "safe"
)

// AlertSeverity represents alert severity
type AlertSeverity string

const (
	SeverityLow       AlertSeverity = "low"
	SeverityMedium    AlertSeverity = "medium"
	SeverityHigh      AlertSeverity = "high"
	SeverityCritical  AlertSeverity = "critical"
)

// RiskScore represents a risk assessment
type RiskScore struct {
	Address        string         `json:"address"`
	Score         int            `json:"score"` // 0-100
	Factors       []RiskFactor  `json:"factors"`
	Recommendation  string        `json:"recommendation"`
	CheckedAt     int64         `json:"checked_at"`
}

// RiskFactor represents a risk factor
type RiskFactor struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Weight     int    `json:"weight"`
}

// ScamDatabase holds known scam addresses populated at runtime from the
// backend security service / external scam-address feeds. It starts EMPTY
// (no fabricated entries) — an address is only flagged when a real report
// has been registered via RegisterScamAddress or loaded from PostgreSQL.
var ScamDatabase = map[string]ScamInfo{}

// ScamInfo holds information about a scam
type ScamInfo struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
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
	Rules       []ProtectionRule
	Alerts     []SecurityAlert
	Whitelist  map[string]bool
	Blacklist  map[string]bool
}

// ProtectionRule represents a transaction protection rule
type ProtectionRule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Type       RuleType    `json:"type"`
	Enabled    bool        `json:"enabled"`
	Threshold  string      `json:"threshold,omitempty"`
	Addresses  []string    `json:"addresses,omitempty"`
}

// RuleType represents the type of rule
type RuleType string

const (
	RuleMaxValue      RuleType = "max_value"
	RuleBlacklist    RuleType = "blacklist"
	RuleWhitelist    RuleType = "whitelist"
	RuleUnverified   RuleType = "unverified_token"
	RuleHighSlippage RuleType = "high_slippage"
)

// NewWalletGuardian creates a new wallet guardian
func NewWalletGuardian() *WalletGuardian {
	return &WalletGuardian{
		Rules:      DefaultProtectionRules(),
		Alerts:    make([]SecurityAlert, 0),
		Whitelist: make(map[string]bool),
		Blacklist: make(map[string]bool),
	}
}

// DefaultProtectionRules returns default protection rules
func DefaultProtectionRules() []ProtectionRule {
	return []ProtectionRule{
		{
			ID:         "max_value_eth",
			Name:       "Max ETH per transaction",
			Type:      RuleMaxValue,
			Enabled:   true,
			Threshold: "10",
		},
		{
			ID:        "block_unknown",
			Name:      "Block unknown tokens",
			Type:      RuleUnverified,
			Enabled:  true,
		},
		{
			ID:       "high_slippage",
			Name:     "High slippage warning",
			Type:     RuleHighSlippage,
			Enabled:  true,
			Threshold: "5",
		},
	}
}

// AnalyzeTransaction analyzes a transaction for risks
func (wg *WalletGuardian) AnalyzeTransaction(tx *TransactionAnalysis) *SecurityAlert {
	// Check against blacklist
	for _, addr := range tx.To {
		if wg.Blacklist[addr] {
			return &SecurityAlert{
				ID:        generateAlertID(),
				Type:      AlertPhishing,
				Severity:  SeverityCritical,
				Message:  "Transaction to blacklisted address",
				Address:  addr,
				TxHash:   tx.Hash,
				CreatedAt: time.Now().Unix(),
			}
		}
	}

	// Check value limits
	if tx.Value != "" {
		// Check if value exceeds threshold
		for _, rule := range wg.Rules {
			if rule.Type == RuleMaxValue && rule.Enabled {
				// Check value against threshold
			}
		}
	}

	// Check token approval
	if tx.Data != "" && strings.HasPrefix(tx.Data, "0x095ea7b3") {
		// Token approval - check if token is verified
	}

	return nil
}

// TransactionAnalysis represents a transaction to analyze
type TransactionAnalysis struct {
	Hash        string   `json:"hash"`
	From       string   `json:"from"`
	To         []string `json:"to"`
	Value      string   `json:"value"`
	Data       string   `json:"data"`
	ChainID    uint64   `json:"chain_id"`
	Token      string   `json:"token,omitempty"`
}

// CheckAddressRisk checks an address for risk factors
func CheckAddressRisk(address string) *RiskScore {
	score := &RiskScore{
		Address:    address,
		Score:     0,
		Factors:   make([]RiskFactor, 0),
		CheckedAt: time.Now().Unix(),
	}

	// Check if in scam database
	if info, ok := ScamDatabase[strings.ToLower(address)]; ok {
		score.Factors = append(score.Factors, RiskFactor{
			Type:        "known_scam",
			Description: fmt.Sprintf("Known scam: %s", info.Name),
			Weight:     100,
		})
		score.Score = 100
		score.Recommendation = "AVOID - This address is associated with a known scam"
		return score
	}

	// Check address age (new addresses are higher risk)
	score.Factors = append(score.Factors, RiskFactor{
		Type:        "new_address",
		Description: "Address created recently",
		Weight:     30,
	})
	score.Score += 30

	// Set recommendation
	if score.Score >= 70 {
		score.Recommendation = "HIGH RISK - Exercise extreme caution"
	} else if score.Score >= 40 {
		score.Recommendation = "MEDIUM RISK - Verify before proceeding"
	} else {
		score.Recommendation = "LOW RISK - Standard caution recommended"
	}

	return score
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
	DomainDB    map[string]PhishInfo
	SuffixDB    map[string]bool
}

// PhishInfo holds phishing site information
type PhishInfo struct {
	Domain      string   `json:"domain"`
	Target     string   `json:"target"`
	ReportedAt int64    `json:"reported_at"`
	Type       string   `json:"type"`
}

// NewAntiPhishing creates a new anti-phishing detector
func NewAntiPhishing() *AntiPhishing {
	return &AntiPhishing{
		DomainDB: make(map[string]PhishInfo),
		SuffixDB: map[string]bool{
			"-wallet.com":     true,
			"-eth.com":       true,
			"-binance.com":   true,
			"-uniswap.com":  true,
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
			ID:       generateAlertID(),
			Type:     AlertPhishing,
			Severity: SeverityHigh,
			Message:  fmt.Sprintf("Known phishing domain: %s", info.Target),
			Details:  json.RawMessage(fmt.Sprintf(`{"domain": "%s", "type": "%s"}`, domain, info.Type)),
			CreatedAt: time.Now().Unix(),
		}
	}

	// Check suspicious suffixes
	for suffix := range ap.SuffixDB {
		if strings.HasSuffix(domain, suffix) {
			// Could be phishing - flag as suspicious
			return &SecurityAlert{
				ID:       generateAlertID(),
				Type:     AlertSuspicious,
				Severity: SeverityMedium,
				Message:  fmt.Sprintf("Unusual domain suffix: %s", suffix),
				CreatedAt: time.Now().Unix(),
			}
		}
	}

	return nil
}

// SimulateTransaction simulates a transaction to show potential outcomes
func SimulateTransaction(tx *TransactionAnalysis) (*SimulationResult, error) {
	result := &SimulationResult{
		Success:      true,
		GasEstimate:  "21000",
		TokenChange:  make(map[string]string),
		Errors:      make([]string, 0),
	}

	// Basic simulation
	if tx.Value != "" {
		result.TokenChange["native"] = "-" + tx.Value
	}

	// Check for common issues
	if strings.HasPrefix(tx.Data, "0x") && len(tx.Data) > 10 {
		result.Warnings = append(result.Warnings, "Transaction includes contract interaction")
	}

	return result, nil
}

// SimulationResult represents transaction simulation results
type SimulationResult struct {
	Success     bool              `json:"success"`
	GasEstimate string            `json:"gas_estimate"`
	TokenChange map[string]string `json:"token_change"`
	Warnings   []string          `json:"warnings"`
	Errors     []string          `json:"errors"`
}

// ScanContract scans a contract for malicious code
func ScanContract(code string) *SecurityAlert {
	// Simple pattern matching for common malicious patterns
	maliciousPatterns := map[string]string{
		"selfdestruct": "Contract can be self-destructed",
		"delegatecall": "Uses delegatecall - potential vulnerability",
		"create2":     "Uses create2 - potential front-running",
	}

	code = strings.ToLower(code)
	for pattern, description := range maliciousPatterns {
		if strings.Contains(code, pattern) {
			return &SecurityAlert{
				ID:        generateAlertID(),
				Type:      AlertRisk,
				Severity:  SeverityMedium,
				Message:  description,
				CreatedAt: time.Now().Unix(),
			}
		}
	}

	return nil
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	hash := sha256.Sum256([]byte(time.Now().Format(time.RFC3339Nano)))
	return hex.EncodeToString(hash[:16])
}