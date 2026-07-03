//! Human-readable transaction preview and DApp risk scoring.
//!
//! This module is designed for the pre-signing path. It does not execute chain
//! calls; instead it converts known calldata selectors and metadata into safety
//! warnings that UI, mobile, extension, and backend policy checks can share.

use std::collections::BTreeSet;

use crate::account_abstraction::validate_evm_address;

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum TransactionIntent {
    NativeTransfer,
    TokenTransfer,
    TokenApproval,
    Permit2Approval,
    ContractInteraction,
    Unknown,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PreviewInput {
    pub from: String,
    pub to: String,
    pub value_wei: u128,
    pub calldata_hex: String,
    pub origin_domain: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TransactionPreview {
    pub intent: TransactionIntent,
    pub human_summary: String,
    pub risk_score: u32,
    pub warnings: Vec<String>,
    pub requires_simulation: bool,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DappRiskProfile {
    pub verified_domains: BTreeSet<String>,
    pub blocked_domains: BTreeSet<String>,
    pub malicious_contracts: BTreeSet<String>,
}

impl DappRiskProfile {
    pub fn assess(&self, input: &PreviewInput) -> TransactionPreview {
        let mut preview = TransactionPreview {
            intent: classify_calldata(&input.calldata_hex),
            human_summary: String::new(),
            risk_score: 0,
            warnings: Vec::new(),
            requires_simulation: !input.calldata_hex.trim().is_empty(),
        };

        if validate_evm_address(&input.from).is_err() || validate_evm_address(&input.to).is_err() {
            preview.risk_score += 100;
            preview
                .warnings
                .push("Invalid EVM address in transaction".to_string());
        }

        let to_lower = input.to.to_lowercase();
        if self.malicious_contracts.contains(&to_lower) {
            preview.risk_score += 100;
            preview
                .warnings
                .push("Recipient contract is on the malicious contract blocklist".to_string());
        }

        if let Some(domain) = input.origin_domain.as_ref().map(|d| normalize_domain(d)) {
            if self.blocked_domains.contains(&domain) {
                preview.risk_score += 100;
                preview
                    .warnings
                    .push("DApp origin domain is blocked".to_string());
            } else if !self.verified_domains.is_empty() && !self.verified_domains.contains(&domain)
            {
                preview.risk_score += 15;
                preview
                    .warnings
                    .push("DApp origin is not in the verified-domain registry".to_string());
            }
        } else {
            preview.risk_score += 10;
            preview
                .warnings
                .push("Missing DApp origin; phishing checks are limited".to_string());
        }

        match preview.intent {
            TransactionIntent::NativeTransfer => {
                preview.human_summary = format!("Send {} wei native token", input.value_wei);
            }
            TransactionIntent::TokenTransfer => {
                preview.human_summary = "Transfer ERC-20 tokens".to_string();
                preview.requires_simulation = true;
            }
            TransactionIntent::TokenApproval => {
                preview.human_summary = "Approve ERC-20 token spending".to_string();
                preview.risk_score += 35;
                preview.warnings.push(
                    "Token approval can allow future token transfers; verify spender and allowance"
                        .to_string(),
                );
                preview.requires_simulation = true;
            }
            TransactionIntent::Permit2Approval => {
                preview.human_summary = "Permit2 approval/signature flow".to_string();
                preview.risk_score += 45;
                preview.warnings.push("Permit2 approval can grant reusable spending rights; verify spender, token, amount, and deadline".to_string());
                preview.requires_simulation = true;
            }
            TransactionIntent::ContractInteraction => {
                preview.human_summary = "Interact with smart contract".to_string();
                preview.risk_score += 20;
                preview
                    .warnings
                    .push("Unknown contract call requires simulation before signing".to_string());
                preview.requires_simulation = true;
            }
            TransactionIntent::Unknown => {
                preview.human_summary = "Unknown transaction type".to_string();
                preview.risk_score += 25;
                preview.warnings.push(
                    "Unable to classify calldata; require simulation and manual review".to_string(),
                );
                preview.requires_simulation = true;
            }
        }

        preview
    }
}

pub fn classify_calldata(calldata_hex: &str) -> TransactionIntent {
    let data = calldata_hex.trim_start_matches("0x").to_lowercase();
    if data.is_empty() {
        return TransactionIntent::NativeTransfer;
    }
    if data.len() < 8 {
        return TransactionIntent::Unknown;
    }
    match &data[..8] {
        "a9059cbb" => TransactionIntent::TokenTransfer,
        "095ea7b3" => TransactionIntent::TokenApproval,
        "3593564c" | "2b67b570" => TransactionIntent::Permit2Approval,
        _ => TransactionIntent::ContractInteraction,
    }
}

pub fn normalize_domain(domain: &str) -> String {
    domain
        .trim()
        .trim_start_matches("https://")
        .trim_start_matches("http://")
        .trim_end_matches('/')
        .to_ascii_lowercase()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn addr(n: u8) -> String {
        format!("0x{:040x}", n)
    }

    #[test]
    fn flags_permit2_and_unverified_domain() {
        let profile = DappRiskProfile {
            verified_domains: BTreeSet::from(["app.tigerwallet.io".to_string()]),
            blocked_domains: BTreeSet::new(),
            malicious_contracts: BTreeSet::new(),
        };
        let preview = profile.assess(&PreviewInput {
            from: addr(1),
            to: addr(2),
            value_wei: 0,
            calldata_hex: "0x3593564c0000".to_string(),
            origin_domain: Some("https://unknown.example/".to_string()),
        });
        assert_eq!(preview.intent, TransactionIntent::Permit2Approval);
        assert!(preview.risk_score >= 60);
        assert!(preview.requires_simulation);
    }

    #[test]
    fn blocks_malicious_contracts() {
        let profile = DappRiskProfile {
            verified_domains: BTreeSet::new(),
            blocked_domains: BTreeSet::new(),
            malicious_contracts: BTreeSet::from([addr(9)]),
        };
        let preview = profile.assess(&PreviewInput {
            from: addr(1),
            to: addr(9),
            value_wei: 1,
            calldata_hex: "".to_string(),
            origin_domain: Some("safe.example".to_string()),
        });
        assert!(preview.risk_score >= 100);
    }
}
