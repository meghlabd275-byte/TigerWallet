//! Transaction classifier + rule enforcement (Rust mirror of the C++ hot path).
//!
//! This is the authoritative logic the C++ gate snapshots are derived from.
//! Rust services (and Node services via a thin FFI) call this directly.
//!
//! The security boundary: fee/revenue/treasury withdrawals are ALWAYS Manual;
//! everything else is Auto (approved iff license alive + no blocking rule).

use crate::{is_treasury_address, normalize_addr, AutoSignRule};
use std::collections::HashSet;

/// Approval mode (mirrors the C++ ApprovalMode).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ApprovalMode {
    Auto,
    Manual,
}

/// Coarse tx classification (mirrors the C++ TxKind).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TxKind {
    UserTransfer,
    Swap,
    Stake,
    NftTransfer,
    PersonalSign,
    TypedDataSign,
    RevenuePayout,
    TreasuryTransfer,
    TreasurySweep,
    FeeWithdrawal,
    Unknown,
}

/// The classification result.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ApprovalDecision {
    pub mode: ApprovalMode,
    pub approved: bool,
    pub reason: String,
    pub rule_id: String,
}

impl TxKind {
    pub fn from_str(s: &str) -> Self {
        match s {
            "transfer" | "send" => TxKind::UserTransfer,
            "swap" => TxKind::Swap,
            "stake" | "unstake" | "claim" => TxKind::Stake,
            "nft_transfer" => TxKind::NftTransfer,
            "personal_sign" => TxKind::PersonalSign,
            "typed_data_sign" => TxKind::TypedDataSign,
            "revenue_payout" => TxKind::RevenuePayout,
            "treasury_transfer" => TxKind::TreasuryTransfer,
            "treasury_sweep" => TxKind::TreasurySweep,
            "fee_withdrawal" => TxKind::FeeWithdrawal,
            _ => TxKind::Unknown,
        }
    }

    fn name(self) -> &'static str {
        match self {
            TxKind::UserTransfer => "transfer",
            TxKind::Swap => "swap",
            TxKind::Stake => "stake",
            TxKind::NftTransfer => "nft_transfer",
            TxKind::PersonalSign => "personal_sign",
            TxKind::TypedDataSign => "typed_data_sign",
            TxKind::RevenuePayout => "revenue_payout",
            TxKind::TreasuryTransfer => "treasury_transfer",
            TxKind::TreasurySweep => "treasury_sweep",
            TxKind::FeeWithdrawal => "fee_withdrawal",
            TxKind::Unknown => "unknown",
        }
    }
}

/// Classify an outgoing transaction. Pure function over the tx inputs + the
/// policy snapshot (treasury addresses + auto-sign rules) + the liveness flag.
///
/// Returns Manual mode for fee/revenue/treasury withdrawals OR any tx whose
/// recipient is a treasury address. Otherwise Auto mode (approved iff license
/// alive + no blocking rule).
pub fn classify_transaction(
    tx_type: &str,
    to: &str,
    token: &str,
    amount: &str,
    license_alive: bool,
    treasury: &HashSet<String>,
    rules: &[AutoSignRule],
) -> ApprovalDecision {
    let kind = TxKind::from_str(tx_type);

    // 1. Fee / revenue / treasury txs are ALWAYS manual.
    if matches!(
        kind,
        TxKind::RevenuePayout | TxKind::TreasuryTransfer | TxKind::TreasurySweep | TxKind::FeeWithdrawal
    ) {
        return ApprovalDecision {
            mode: ApprovalMode::Manual,
            approved: false,
            reason: "fee/revenue/treasury withdrawal requires SuperAdmin two-party co-sign".into(),
            rule_id: String::new(),
        };
    }

    // 2. Recipient on the treasury-address set => Manual.
    if is_treasury_address(to, treasury) {
        return ApprovalDecision {
            mode: ApprovalMode::Manual,
            approved: false,
            reason: "recipient is a treasury/revenue/fee address (two-party required)".into(),
            rule_id: String::new(),
        };
    }

    // 3. Auto path: require license alive.
    if !license_alive {
        return ApprovalDecision {
            mode: ApprovalMode::Auto,
            approved: false,
            reason: "product is not authorized to serve (license suspended/revoked or heartbeat stale)".into(),
            rule_id: String::new(),
        };
    }

    // 4. Apply auto-sign rules (block rules deny).
    for r in rules {
        if !rule_matches(r, kind, token, amount) {
            continue;
        }
        if r.block {
            return ApprovalDecision {
                mode: ApprovalMode::Auto,
                approved: false,
                reason: "blocked by auto-sign rule".into(),
                rule_id: r.rule_id.clone(),
            };
        }
    }

    // 5. Default: Auto-approve.
    ApprovalDecision {
        mode: ApprovalMode::Auto,
        approved: true,
        reason: String::new(),
        rule_id: String::new(),
    }
}

fn rule_matches(r: &AutoSignRule, kind: TxKind, token: &str, amount: &str) -> bool {
    if r.tx_type != "*" && r.tx_type != kind.name() {
        return false;
    }
    if r.token != "*" {
        if token.is_empty() {
            return false;
        }
        if normalize_addr(token) != normalize_addr(&r.token) {
            return false;
        }
    }
    if !r.max_amount.is_empty() && r.max_amount != "0" {
        if !amount.is_empty() && !amount_exceeds(amount, &r.max_amount) {
            return false;
        }
    }
    true
}

fn amount_exceeds(amt: &str, cap: &str) -> bool {
    let a = strip_trailing_zeros(amt);
    let c = strip_trailing_zeros(cap);
    if a.len() != c.len() {
        return a.len() > c.len();
    }
    a.as_str() > c.as_str()
}

fn strip_trailing_zeros(s: &str) -> String {
    let mut out = s.trim_end_matches('0').to_string();
    if out.ends_with('.') {
        out.pop();
    }
    out
}
