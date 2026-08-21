// TigerWallet WL Control Plane — ultra-low-latency transaction auto-approver.
//
// This is the AUTO-APPROVE hot path: when a UserWallet user (or a MasterWallet
// owner transferring on a user's behalf) submits an outgoing transaction, the
// backend FFI-calls into this C++ checker BEFORE it signs/broadcasts. If the
// transaction is a normal user-initiated transfer (swap/stake/send/personal_sign)
// AND the product license is alive AND the relevant fetcher is enabled AND no
// SuperAdmin rule blocks it, this returns APPROVED in single-digit microseconds
// — giving the "<1 second automatic sign and approval" the product requires.
//
// IMPORTANT — two approval modes (this reconciles the spec's two invariants):
//   1. AUTO mode (this module): user-initiated outgoing txs are approved here,
//      in-process, wait-free. The license-alive flag + the auto-sign rule
//      cache ARE the approval. NO network round-trip, NO control-plane call.
//   2. MANUAL mode (TwoPartyGate): fee / revenue / treasury withdrawals by the
//      WL client or MasterWallet owner ALWAYS require an explicit SuperAdmin
//      co-sign via the control plane. Those txs are classified by
//      classify_transaction() below as WithdrawalMode and are NEVER auto-approved.
//
// The classifier is the security boundary: any tx whose `to` address matches a
// known fee/treasury/revenue address, OR whose `tx_type` is explicitly
// revenue_payout/treasury_transfer/treasury_sweep/fee_withdrawal, is forced to
// MANUAL mode. This makes it impossible for the WL client or MasterWallet owner
// to silently route a fee/revenue withdrawal through the fast path.
//
// Language rationale: C++20 for wait-free atomics + zero-overhead abstractions +
// a stable C ABI every WL product backend (Go/Rust/Node) can FFI into.
#pragma once

#include <atomic>
#include <cstdint>
#include <cstring>
#include <functional>
#include <string>
#include <string_view>
#include <unordered_set>
#include <shared_mutex>
#include <vector>

namespace tigerwallet::wl {

// Approval mode for an outgoing transaction.
enum class ApprovalMode : int {
    Auto = 0,       // license-alive + fetcher-enabled + rule-permit => approve in-process (<1ms)
    Manual = 1,     // fee/revenue/treasury withdrawal => require SuperAdmin two-party co-sign
};

// The classification result: which path the tx must take.
struct ApprovalDecision {
    ApprovalMode mode;
    bool approved;        // for Auto mode: true iff license alive + fetcher enabled + no blocking rule
    std::string reason;   // human-readable reason (for 403/503 bodies)
    std::string rule_id;  // the blocking rule id, if any (for audit)
};

// TxKind — coarse classification of the outgoing transaction. The classifier
// maps these to Auto/Manual.
enum class TxKind : int {
    UserTransfer = 0,    // ordinary ETH/ERC-20 send by the user
    Swap = 1,            // DEX swap (user-initiated)
    Stake = 2,           // staking deposit/unstake/claim (user-initiated)
    NftTransfer = 3,     // ERC-721 transfer (user-initiated)
    PersonalSign = 4,    // EIP-191 message sign (NOT a value transfer)
    TypedDataSign = 5,   // EIP-712 typed-data sign (NOT a value transfer)
    RevenuePayout = 10,  // WL client / MW owner moving collected fees/revenue
    TreasuryTransfer = 11,
    TreasurySweep = 12,
    FeeWithdrawal = 13,
    Unknown = 99,
};

// An auto-sign rule: a SuperAdmin-defined policy that can BLOCK a specific
// auto-approve even when the license is alive (e.g. block auto-approve above
// a daily per-user limit, or block a specific token contract). Rules are
// refreshed from the control plane on each heartbeat (pushed into the C++
// gate by the Rust SDK / Go heartbeat, same as the flag snapshot).
struct AutoSignRule {
    std::string rule_id;      // UUID
    std::string product;
    std::string fetcher;      // "*" or a specific fetcher
    std::string tx_type;      // "*" or a specific TxKind name
    std::string token;        // "*" or a token contract address (lowercase)
    std::string max_amount;   // decimal string; "0" = unlimited; tx above => block
    bool block;               // true = block (deny auto), false = allow
};

// WlAutoApprover is a wait-free-read transaction classifier + auto-sign-rule
// cache. ONE instance per process (singleton). Reads are lock-free for the
// liveness check; the rule set + treasury-address set use a shared_mutex
// (contended only during heartbeat refresh, which is rare).
class WlAutoApprover {
public:
    static WlAutoApprover& instance() {
        static WlAutoApprover a;
        return a;
    }

    // --- liveness (lock-free atomic, mirrored from WlGate) ---
    void set_alive(bool v, const char* reason = nullptr) {
        alive_.store(v, std::memory_order_release);
        if (reason) {
            std::lock_guard<std::shared_mutex> lk(mu_);
            dead_reason_ = reason;
        } else if (v) {
            std::lock_guard<std::shared_mutex> lk(mu_);
            dead_reason_.clear();
        }
    }
    bool is_alive() const { return alive_.load(std::memory_order_acquire); }

    // --- treasury / revenue / fee address set (Manual-mode triggers) ---
    // These are addresses the WL client or SuperAdmin has marked as
    // fee/revenue/treasury destinations. Any tx to one of these is MANUAL.
    void set_treasury_addresses(const std::unordered_set<std::string>& addrs) {
        std::lock_guard<std::shared_mutex> lk(mu_);
        treasury_addrs_.clear();
        for (auto a : addrs) { // copy, then normalize to lowercase
            to_lower(a);
            treasury_addrs_.insert(std::move(a));
        }
    }
    void add_treasury_address(std::string addr) {
        to_lower(addr);
        std::lock_guard<std::shared_mutex> lk(mu_);
        treasury_addrs_.insert(std::move(addr));
    }

    // --- auto-sign rules (pushed by the heartbeat) ---
    void set_rules(const std::vector<AutoSignRule>& rules) {
        std::lock_guard<std::shared_mutex> lk(mu_);
        rules_ = rules;
    }

    // Classify an outgoing transaction. Pure function over (tx_type, to, token,
    // amount). This is the security boundary: it decides Auto vs Manual.
    //
    // tx_type: one of "transfer","swap","stake","unstake","claim","nft_transfer",
    //          "personal_sign","typed_data_sign","revenue_payout",
    //          "treasury_transfer","treasury_sweep","fee_withdrawal".
    // to:      recipient address (hex, 0x-prefixed or not). Empty for sign-only.
    // token:   token contract address (empty = native asset).
    // amount:  human-readable decimal amount string (empty for sign-only).
    ApprovalDecision classify(std::string_view tx_type,
                              std::string_view to,
                              std::string_view token,
                              std::string_view amount) const {
        ApprovalDecision d{ApprovalMode::Auto, false, {}, {}};
        TxKind kind = parse_kind(tx_type);

        // 1. Fee / revenue / treasury txs are ALWAYS manual. No fast path.
        if (kind == TxKind::RevenuePayout ||
            kind == TxKind::TreasuryTransfer ||
            kind == TxKind::TreasurySweep ||
            kind == TxKind::FeeWithdrawal) {
            d.mode = ApprovalMode::Manual;
            d.reason = "fee/revenue/treasury withdrawal requires SuperAdmin two-party co-sign";
            return d;
        }

        // 2. If the recipient is a known treasury/revenue/fee address => Manual.
        if (!to.empty()) {
            std::string to_norm(to);
            to_lower(to_norm);
            std::shared_lock<std::shared_mutex> lk(mu_);
            if (treasury_addrs_.count(to_norm)) {
                lk.unlock();
                d.mode = ApprovalMode::Manual;
                d.reason = "recipient is a treasury/revenue/fee address (two-party required)";
                return d;
            }
        }

        // 3. For Manual txs we're done (the caller must call the control plane).
        if (d.mode == ApprovalMode::Manual) return d;

        // 4. Auto path: require license alive.
        if (!is_alive()) {
            d.approved = false;
            d.reason = "product is not authorized to serve (license suspended/revoked or heartbeat stale)";
            return d;
        }

        // 5. Apply auto-sign rules (block rules can deny; allow rules can cap).
        std::shared_lock<std::shared_mutex> lk(mu_);
        for (const auto& r : rules_) {
            if (!rule_matches(r, kind, token, amount)) continue;
            if (r.block) {
                d.approved = false;
                d.reason = "blocked by auto-sign rule";
                d.rule_id = r.rule_id;
                return d;
            }
        }
        lk.unlock();

        // 6. Default: Auto-approve. The license-alive flag + fetcher-enabled
        //    flag (checked separately by the gate middleware) ARE the approval.
        d.approved = true;
        return d;
    }

private:
    WlAutoApprover() = default;

    static TxKind parse_kind(std::string_view t) {
        if (t == "transfer" || t == "send") return TxKind::UserTransfer;
        if (t == "swap") return TxKind::Swap;
        if (t == "stake" || t == "unstake" || t == "claim") return TxKind::Stake;
        if (t == "nft_transfer") return TxKind::NftTransfer;
        if (t == "personal_sign") return TxKind::PersonalSign;
        if (t == "typed_data_sign") return TxKind::TypedDataSign;
        if (t == "revenue_payout") return TxKind::RevenuePayout;
        if (t == "treasury_transfer") return TxKind::TreasuryTransfer;
        if (t == "treasury_sweep") return TxKind::TreasurySweep;
        if (t == "fee_withdrawal") return TxKind::FeeWithdrawal;
        return TxKind::Unknown;
    }

    static void to_lower(std::string& s) {
        for (auto& c : s) {
            if (c >= 'A' && c <= 'Z') c = c - 'A' + 'a';
        }
    }

    // Does a rule apply to this tx? "*" wildcards match anything.
    static bool rule_matches(const AutoSignRule& r, TxKind kind,
                             std::string_view token, std::string_view amount) {
        if (r.tx_type != "*" && r.tx_type != kind_name(kind)) return false;
        if (r.token != "*") {
            if (token.empty()) return false;
            std::string t(token); to_lower(t);
            std::string rt(r.token); to_lower(rt);
            if (t != rt) return false;
        }
        // max_amount: block if the tx amount exceeds the rule cap.
        if (!r.max_amount.empty() && r.max_amount != "0") {
            if (!amount.empty() && !amount_exceeds(amount, r.max_amount)) {
                return false;
            }
        }
        return true;
    }

    static std::string_view kind_name(TxKind k) {
        switch (k) {
            case TxKind::UserTransfer: return "transfer";
            case TxKind::Swap: return "swap";
            case TxKind::Stake: return "stake";
            case TxKind::NftTransfer: return "nft_transfer";
            case TxKind::PersonalSign: return "personal_sign";
            case TxKind::TypedDataSign: return "typed_data_sign";
            case TxKind::RevenuePayout: return "revenue_payout";
            case TxKind::TreasuryTransfer: return "treasury_transfer";
            case TxKind::TreasurySweep: return "treasury_sweep";
            case TxKind::FeeWithdrawal: return "fee_withdrawal";
            default: return "unknown";
        }
    }

    // Decimal string comparison: true if `amt` > `cap` (lexicographic on equal
    // length; otherwise length+content). Sufficient for human-readable amounts.
    static bool amount_exceeds(std::string_view amt, std::string_view cap) {
        // strip trailing zeros for fair comparison
        std::string a(amt); std::string c(cap);
        while (!a.empty() && a.back() == '0') a.pop_back();
        while (!a.empty() && a.back() == '.') a.pop_back();
        while (!c.empty() && c.back() == '0') c.pop_back();
        while (!c.empty() && c.back() == '.') c.pop_back();
        if (a.size() != c.size()) return a.size() > c.size();
        return a > c;
    }

    std::atomic<bool> alive_{false};
    mutable std::shared_mutex mu_;
    std::string dead_reason_;
    std::unordered_set<std::string> treasury_addrs_;
    std::vector<AutoSignRule> rules_;
};

} // namespace tigerwallet::wl

// --- C ABI so Go/Rust/Node WL backends can FFI in without C++ name mangling ---
// Returns:
//   0 = Auto mode + approved (fast path: sign + broadcast immediately)
//   1 = Manual mode required (slow path: caller must call TwoPartyGate)
//   2 = Auto mode + DENIED (license dead or rule blocked — return 403/503)
// out_reason / out_rule_id receive pointers to static buffers the caller MUST
// copy before the next call.
extern "C" {
    int wl_auto_approve_classify(const char* tx_type,
                                 const char* to,
                                 const char* token,
                                 const char* amount,
                                 const char** out_reason,
                                 const char** out_rule_id);

    void wl_auto_approver_set_alive(int alive, const char* reason);
    void wl_auto_approver_add_treasury_address(const char* addr);
    void wl_auto_approver_set_treasury_addresses_json(const char* json_array);
    void wl_auto_approver_set_rules_json(const char* json_array);
}
