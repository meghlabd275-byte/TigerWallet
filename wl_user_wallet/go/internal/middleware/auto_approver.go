// auto_approver.go — the pure-Go transaction classifier + auto-sign-rule cache
// for the standalone WL-UserWallet backend.
//
// This mirrors wl_shared/go/wlgate/auto_approver.go and the C++ WlGate
// classifier. The security boundary is identical:
//   - fee/revenue/treasury withdrawals => ALWAYS Manual (two-party required)
//   - any tx to a treasury/revenue/fee address => ALWAYS Manual
//   - everything else (license-alive + no blocking rule) => Auto-approved
//
// This reconciles the two product invariants:
//   "user txs are auto-signed + auto-approved within a second" (fast path)
//   "no fee/revenue withdrawal without SuperAdmin collaboration" (slow path)
package middleware

import (
	"strings"
	"sync"
	"sync/atomic"
)

// ApprovalMode is the path a transaction must take.
type ApprovalMode int

const (
	ModeAuto ApprovalMode = iota
	ModeManual
)

// ApprovalDecision is the classification result.
type ApprovalDecision struct {
	Mode     ApprovalMode
	Approved bool
	Reason   string
	RuleID   string
}

// AutoSignRule is a SuperAdmin-defined policy that can block a specific
// auto-approve. Pushed into the gate on each heartbeat.
type AutoSignRule struct {
	RuleID    string `json:"rule_id"`
	Product   string `json:"product"`
	Fetcher   string `json:"fetcher"`
	TxType    string `json:"tx_type"`
	Token     string `json:"token"`
	MaxAmount string `json:"max_amount"`
	Block     bool   `json:"block"`
}

// autoApprover is the in-process transaction classifier. One per process.
type autoApprover struct {
	alive    atomic.Bool
	mu       sync.RWMutex
	reason   string
	treasury map[string]struct{}
	rules    []AutoSignRule
}

var approver = &autoApprover{treasury: map[string]struct{}{}, reason: "license not yet validated"}

// SetApproverAlive mirrors the gate liveness for the auto-approver.
func SetApproverAlive(alive bool, reason string) {
	approver.alive.Store(alive)
	approver.mu.Lock()
	if alive {
		approver.reason = ""
	} else {
		approver.reason = reason
	}
	approver.mu.Unlock()
}

// AddTreasuryAddress marks an address as a fee/revenue/treasury destination.
func AddTreasuryAddress(addr string) {
	approver.mu.Lock()
	defer approver.mu.Unlock()
	approver.treasury[strings.ToLower(strings.TrimPrefix(addr, "0x"))] = struct{}{}
}

// SetTreasuryAddresses replaces the treasury-address set.
func SetTreasuryAddresses(addrs []string) {
	approver.mu.Lock()
	defer approver.mu.Unlock()
	approver.treasury = map[string]struct{}{}
	for _, a := range addrs {
		approver.treasury[strings.ToLower(strings.TrimPrefix(a, "0x"))] = struct{}{}
	}
}

// SetAutoSignRules replaces the auto-sign rule set.
func SetAutoSignRules(rules []AutoSignRule) {
	approver.mu.Lock()
	defer approver.mu.Unlock()
	approver.rules = rules
}

// ApproverIsAlive returns the auto-approver liveness flag.
func ApproverIsAlive() bool { return approver.alive.Load() }

// ClassifyTx decides the approval mode for an outgoing transaction.
//
// txType: "transfer"|"swap"|"stake"|"unstake"|"claim"|"nft_transfer"|
//         "personal_sign"|"typed_data_sign"|"revenue_payout"|
//         "treasury_transfer"|"treasury_sweep"|"fee_withdrawal"
// to:     recipient address (optional for sign-only).
// token:  token contract address ("" = native).
// amount: human-readable decimal amount ("" = no value).
//
// Returns ModeManual for fee/revenue/treasury OR any tx to a treasury address.
// Otherwise ModeAuto (approved iff license alive + no blocking rule).
func ClassifyTx(txType, to, token, amount string) ApprovalDecision {
	d := ApprovalDecision{Mode: ModeAuto}
	kind := parseKind(txType)

	// 1. Fee / revenue / treasury txs are ALWAYS manual.
	if kind == kRevenuePayout || kind == kTreasuryTransfer ||
		kind == kTreasurySweep || kind == kFeeWithdrawal {
		d.Mode = ModeManual
		d.Reason = "fee/revenue/treasury withdrawal requires SuperAdmin two-party co-sign"
		return d
	}

	// 2. Recipient on the treasury-address set => Manual.
	if to != "" {
		approver.mu.RLock()
		_, hit := approver.treasury[strings.ToLower(strings.TrimPrefix(to, "0x"))]
		approver.mu.RUnlock()
		if hit {
			d.Mode = ModeManual
			d.Reason = "recipient is a treasury/revenue/fee address (two-party required)"
			return d
		}
	}

	if d.Mode == ModeManual {
		return d
	}

	// 3. Auto path: require license alive.
	if !ApproverIsAlive() {
		d.Approved = false
		d.Reason = "product is not authorized to serve (license suspended/revoked or heartbeat stale)"
		return d
	}

	// 4. Apply auto-sign rules.
	approver.mu.RLock()
	for _, r := range approver.rules {
		if !ruleApplies(r, kind, token, amount) {
			continue
		}
		if r.Block {
			approver.mu.RUnlock()
			d.Approved = false
			d.Reason = "blocked by auto-sign rule"
			d.RuleID = r.RuleID
			return d
		}
	}
	approver.mu.RUnlock()

	// 5. Default: Auto-approve.
	d.Approved = true
	return d
}

type txKind int

const (
	kUnknown txKind = iota
	kUserTransfer
	kSwap
	kStake
	kNftTransfer
	kPersonalSign
	kTypedDataSign
	kRevenuePayout
	kTreasuryTransfer
	kTreasurySweep
	kFeeWithdrawal
)

func parseKind(s string) txKind {
	switch s {
	case "transfer", "send":
		return kUserTransfer
	case "swap":
		return kSwap
	case "stake", "unstake", "claim":
		return kStake
	case "nft_transfer":
		return kNftTransfer
	case "personal_sign":
		return kPersonalSign
	case "typed_data_sign":
		return kTypedDataSign
	case "revenue_payout":
		return kRevenuePayout
	case "treasury_transfer":
		return kTreasuryTransfer
	case "treasury_sweep":
		return kTreasurySweep
	case "fee_withdrawal":
		return kFeeWithdrawal
	}
	return kUnknown
}

func kindStr(k txKind) string {
	switch k {
	case kUserTransfer:
		return "transfer"
	case kSwap:
		return "swap"
	case kStake:
		return "stake"
	case kNftTransfer:
		return "nft_transfer"
	case kPersonalSign:
		return "personal_sign"
	case kTypedDataSign:
		return "typed_data_sign"
	case kRevenuePayout:
		return "revenue_payout"
	case kTreasuryTransfer:
		return "treasury_transfer"
	case kTreasurySweep:
		return "treasury_sweep"
	case kFeeWithdrawal:
		return "fee_withdrawal"
	}
	return "unknown"
}

func ruleApplies(r AutoSignRule, k txKind, token, amount string) bool {
	if r.TxType != "*" && r.TxType != kindStr(k) {
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
		if amount != "" && !exceeds(amount, r.MaxAmount) {
			return false
		}
	}
	return true
}

func exceeds(amt, cap string) bool {
	a := strip(amt)
	c := strip(cap)
	if len(a) != len(c) {
		return len(a) > len(c)
	}
	return a > c
}

func strip(s string) string {
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}
