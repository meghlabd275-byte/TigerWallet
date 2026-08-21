// auto_approver.go — the pure-Go transaction classifier + auto-sign-rule cache.
//
// This is the Go mirror of the C++ WlAutoApprover (wl_control_plane/cpp) and
// the Rust wl_policy_engine classifier (wl_control_plane/rust). The three
// implementations are kept in lockstep because different product backends use
// different languages:
//   - C++ services FFI into the C++ gate (wait-free, <1µs hot path).
//   - Go services (wl_user_wallet, wl_master_wallet) use THIS file.
//   - Rust/Node services use the Rust crate.
//
// The security boundary is identical in all three:
//   - fee/revenue/treasury withdrawals => ALWAYS Manual (two-party required)
//   - any tx to a treasury/revenue/fee address => ALWAYS Manual
//   - everything else (license-alive + no blocking rule) => Auto-approved
//
// This reconciles the two product invariants:
//   "user txs are auto-signed + auto-approved within a second" (fast path)
//   "no fee/revenue withdrawal without SuperAdmin collaboration" (slow path)
package wlgate

import (
	"strings"
	"sync"
	"sync/atomic"
)

// ApprovalMode is the path a transaction must take.
type ApprovalMode int

const (
	// ModeAuto means the tx is approved in-process (license-alive + no blocking rule).
	ModeAuto ApprovalMode = iota
	// ModeManual means the tx requires SuperAdmin two-party co-sign (control plane).
	ModeManual
)

// ApprovalDecision is the classification result.
type ApprovalDecision struct {
	Mode    ApprovalMode
	Approved bool   // for ModeAuto: true iff license alive + no blocking rule
	Reason  string // human-readable reason (for 403/503 bodies)
	RuleID  string // the blocking rule id, if any
}

// AutoSignRule is a SuperAdmin-defined policy that can block a specific
// auto-approve (e.g. block auto-approve above a daily per-user limit, or
// block a specific token contract). Pushed into the gate on each heartbeat.
type AutoSignRule struct {
	RuleID    string `json:"rule_id"`
	Product   string `json:"product"`
	Fetcher   string `json:"fetcher"`
	TxType    string `json:"tx_type"`     // "*" or a specific tx type name
	Token     string `json:"token"`       // "*" or a token contract address
	MaxAmount string `json:"max_amount"`   // decimal; "0" = unlimited
	Block     bool   `json:"block"`
}

// AutoApprover is the in-process transaction classifier + auto-sign-rule cache.
// One instance per process. Reads are lock-free for the liveness flag; the
// rule set + treasury-address set use an RWMutex (contended only during
// heartbeat refresh).
type AutoApprover struct {
	alive  atomic.Bool
	mu     sync.RWMutex
	reason string
	treasury map[string]struct{} // lowercase addresses
	rules  []AutoSignRule
}

// NewAutoApprover returns an AutoApprover initialized to dead (fail-closed).
func NewAutoApprover() *AutoApprover {
	return &AutoApprover{
		treasury: map[string]struct{}{},
		reason:   "license not yet validated (heartbeat pending)",
	}
}

// SetAlive sets the liveness flag (mirrors the gate heartbeat).
func (a *AutoApprover) SetAlive(alive bool, reason string) {
	a.alive.Store(alive)
	a.mu.Lock()
	defer a.mu.Unlock()
	if alive {
		a.reason = ""
	} else {
		a.reason = reason
	}
}

// IsAlive returns the liveness flag.
func (a *AutoApprover) IsAlive() bool { return a.alive.Load() }

// AddTreasuryAddress marks an address as a fee/revenue/treasury destination.
// Any tx to it is forced to Manual mode.
func (a *AutoApprover) AddTreasuryAddress(addr string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.treasury[strings.ToLower(strings.TrimPrefix(addr, "0x"))] = struct{}{}
}

// SetTreasuryAddresses replaces the treasury-address set.
func (a *AutoApprover) SetTreasuryAddresses(addrs []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.treasury = map[string]struct{}{}
	for _, addr := range addrs {
		a.treasury[strings.ToLower(strings.TrimPrefix(addr, "0x"))] = struct{}{}
	}
}

// SetRules replaces the auto-sign rule set.
func (a *AutoApprover) SetRules(rules []AutoSignRule) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rules = rules
}

// Classify decides the approval mode for an outgoing transaction.
//
// tx_type: "transfer"|"swap"|"stake"|"unstake"|"claim"|"nft_transfer"|
//          "personal_sign"|"typed_data_sign"|"revenue_payout"|
//          "treasury_transfer"|"treasury_sweep"|"fee_withdrawal"
// to:      recipient address (hex, optional for sign-only). "" => no recipient.
// token:   token contract address ("" = native asset).
// amount:  human-readable decimal amount string ("" = no value, e.g. sign-only).
func (a *AutoApprover) Classify(txType, to, token, amount string) ApprovalDecision {
	d := ApprovalDecision{Mode: ModeAuto}
	kind := parseTxKind(txType)

	// 1. Fee / revenue / treasury txs are ALWAYS manual.
	if kind == kindRevenuePayout || kind == kindTreasuryTransfer ||
		kind == kindTreasurySweep || kind == kindFeeWithdrawal {
		d.Mode = ModeManual
		d.Reason = "fee/revenue/treasury withdrawal requires SuperAdmin two-party co-sign"
		return d
	}

	// 2. Recipient on the treasury-address set => Manual.
	if to != "" {
		a.mu.RLock()
		_, hit := a.treasury[strings.ToLower(strings.TrimPrefix(to, "0x"))]
		a.mu.RUnlock()
		if hit {
			d.Mode = ModeManual
			d.Reason = "recipient is a treasury/revenue/fee address (two-party required)"
			return d
		}
	}

	// 3. Manual mode is done (caller must call the control plane).
	if d.Mode == ModeManual {
		return d
	}

	// 4. Auto path: require license alive.
	if !a.IsAlive() {
		d.Approved = false
		d.Reason = "product is not authorized to serve (license suspended/revoked or heartbeat stale)"
		return d
	}

	// 5. Apply auto-sign rules (block rules can deny).
	a.mu.RLock()
	for _, r := range a.rules {
		if !ruleMatches(r, kind, token, amount) {
			continue
		}
		if r.Block {
			a.mu.RUnlock()
			d.Approved = false
			d.Reason = "blocked by auto-sign rule"
			d.RuleID = r.RuleID
			return d
		}
	}
	a.mu.RUnlock()

	// 6. Default: Auto-approve.
	d.Approved = true
	return d
}

type txKind int

const (
	kindUnknown txKind = iota
	kindUserTransfer
	kindSwap
	kindStake
	kindNftTransfer
	kindPersonalSign
	kindTypedDataSign
	kindRevenuePayout
	kindTreasuryTransfer
	kindTreasurySweep
	kindFeeWithdrawal
)

func parseTxKind(s string) txKind {
	switch s {
	case "transfer", "send":
		return kindUserTransfer
	case "swap":
		return kindSwap
	case "stake", "unstake", "claim":
		return kindStake
	case "nft_transfer":
		return kindNftTransfer
	case "personal_sign":
		return kindPersonalSign
	case "typed_data_sign":
		return kindTypedDataSign
	case "revenue_payout":
		return kindRevenuePayout
	case "treasury_transfer":
		return kindTreasuryTransfer
	case "treasury_sweep":
		return kindTreasurySweep
	case "fee_withdrawal":
		return kindFeeWithdrawal
	}
	return kindUnknown
}

func kindName(k txKind) string {
	switch k {
	case kindUserTransfer:
		return "transfer"
	case kindSwap:
		return "swap"
	case kindStake:
		return "stake"
	case kindNftTransfer:
		return "nft_transfer"
	case kindPersonalSign:
		return "personal_sign"
	case kindTypedDataSign:
		return "typed_data_sign"
	case kindRevenuePayout:
		return "revenue_payout"
	case kindTreasuryTransfer:
		return "treasury_transfer"
	case kindTreasurySweep:
		return "treasury_sweep"
	case kindFeeWithdrawal:
		return "fee_withdrawal"
	}
	return "unknown"
}

func ruleMatches(r AutoSignRule, k txKind, token, amount string) bool {
	if r.TxType != "*" && r.TxType != kindName(k) {
		return false
	}
	if r.Token != "*" {
		if token == "" {
			return false
		}
		if strings.ToLower(strings.TrimPrefix(token, "0x")) !=
			strings.ToLower(strings.TrimPrefix(r.Token, "0x")) {
			return false
		}
	}
	if r.MaxAmount != "" && r.MaxAmount != "0" {
		if amount != "" && !amountExceeds(amount, r.MaxAmount) {
			return false
		}
	}
	return true
}

func amountExceeds(amt, cap string) bool {
	a := stripTrailingZeros(amt)
	c := stripTrailingZeros(cap)
	if len(a) != len(c) {
		return len(a) > len(c)
	}
	return a > c
}

func stripTrailingZeros(s string) string {
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}
