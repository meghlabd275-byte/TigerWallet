//! TigerWallet WL Policy Engine — the security/safety layer for the
//! white-label auto-approve gate.
//!
//! Responsibilities:
//!   1. REAL Ed25519 license-token verification (the cryptographic trust root).
//!      The license token is signed by the TigerWallet SuperAdmin control plane
//!      Ed25519 signing key. If the signature is invalid, the token is
//!      expired, or the product is suspended => the license is REJECTED and
//!      the C++ gate is set dead. No license => no product serves.
//!   2. Transaction classification: maps a tx's (tx_type, to, token, amount) to
//!      Auto or Manual mode. This is the SAME logic as the C++ classifier,
//!      mirrored in Rust for Rust/Node services that don't FFI into C++.
//!      The rule is absolute: fee/revenue/treasury withdrawals are ALWAYS
//!      Manual; everything else (license-alive) is Auto.
//!   3. Auto-sign rule enforcement: a SuperAdmin rule can block a specific
//!      auto-approve (deny) even when the license is alive.
//!
//! Why Rust here: Ed25519 verification is a security primitive that must be
//! memory-safe + side-channel-resistant. ed25519-dalek is the canonical,
//! audited Rust implementation. The classification + rule engine is pure
//! logic with no panics on untrusted input.
//!
//! Language rationale (per project convention):
//!   - C++ (wl_control_plane/cpp): the ultra-low-latency hot path (wait-free
//!     atomics, in-process, <1µs). Go/Rust/Node backends FFI into it.
//!   - Rust (this crate): the security/safety layer (real crypto + the
//!     authoritative rule logic that the C++ gate snapshots are derived from).

use serde::{Deserialize, Serialize};
use std::collections::HashSet;

pub mod license;
pub mod classifier;

pub use classifier::{ApprovalDecision, ApprovalMode, TxKind, classify_transaction};
pub use license::{LicenseToken, verify_license_token, LicenseError};

/// A snapshot of the auto-sign rules + treasury addresses, pushed from the
/// control plane into the C++ gate (and mirrored for Rust services).
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct PolicySnapshot {
    pub rules: Vec<AutoSignRule>,
    pub treasury_addresses: HashSet<String>,
}

/// An auto-sign rule (mirrors the C++ AutoSignRule; serde for the JSON wire format).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutoSignRule {
    pub rule_id: String,
    pub product: String,
    pub fetcher: String,
    pub tx_type: String,
    pub token: String,
    pub max_amount: String,
    pub block: bool,
}

/// Normalize an EVM address to lowercase (for case-insensitive matching).
pub fn normalize_addr(addr: &str) -> String {
    addr.trim_start_matches("0x").to_ascii_lowercase()
}

/// Check whether an address is in the treasury set (case-insensitive).
pub fn is_treasury_address(addr: &str, treasury: &HashSet<String>) -> bool {
    if addr.is_empty() {
        return false;
    }
    let n = normalize_addr(addr);
    treasury.contains(&n) || treasury.contains(addr)
}

#[cfg(test)]
mod tests {
    use super::*;
    use classifier::{ApprovalMode, classify_transaction};
    use ed25519_dalek::{Signature, SigningKey, Signer, VerifyingKey};
    use license::{LicenseToken, verify_license_token};

    #[test]
    fn test_license_token_real_ed25519_roundtrip() {
        // Real Ed25519 keypair (not a fake). The control plane signs the
        // canonical license JSON; the WL backend verifies with the public key.
        let mut csprng = rand::thread_rng();
        let signing = SigningKey::generate(&mut csprng);
        let verifying: VerifyingKey = signing.verifying_key();

        let token = LicenseToken {
            license_key: "wl-key-001".to_string(),
            product: "user_wallet".to_string(),
            white_label_id: "wl-client-uuid".to_string(),
            valid_until: 9999999999,
            status: "active".to_string(),
        };
        let canonical = token.canonical_bytes();
        let sig: Signature = signing.sign(&canonical);
        let signed = token.clone().signed_with(sig.to_bytes().to_vec());

        // Valid signature + active status + not expired => Ok
        let verified = verify_license_token(&signed, &verifying_key_bytes(&verifying), 1000);
        assert!(verified.is_ok(), "valid license should verify: {:?}", verified);

        // Tampered payload => signature invalid
        let mut bad = signed.clone();
        bad.token.product = "master_wallet".to_string();
        let r = verify_license_token(&bad, &verifying_key_bytes(&verifying), 1000);
        assert!(r.is_err());

        // Wrong verifying key => rejected
        let other = SigningKey::generate(&mut csprng);
        let r = verify_license_token(&signed, &verifying_key_bytes(&other.verifying_key()), 1000);
        assert!(r.is_err());

        // Expired => rejected
        let expired = LicenseToken {
            valid_until: 500,
            ..token.clone()
        };
        let exp_sig: Signature = signing.sign(&expired.canonical_bytes());
        let exp_signed = expired.signed_with(exp_sig.to_bytes().to_vec());
        let r = verify_license_token(&exp_signed, &verifying_key_bytes(&verifying), 1000);
        assert!(r.is_err());

        // Suspended => rejected
        let susp = LicenseToken {
            status: "suspended".to_string(),
            ..token.clone()
        };
        let susp_sig: Signature = signing.sign(&susp.canonical_bytes());
        let susp_signed = susp.signed_with(susp_sig.to_bytes().to_vec());
        let r = verify_license_token(&susp_signed, &verifying_key_bytes(&verifying), 1000);
        assert!(r.is_err());
    }

    fn verifying_key_bytes(k: &VerifyingKey) -> Vec<u8> {
        k.to_bytes().to_vec()
    }

    #[test]
    fn test_classifier_user_tx_auto_revenue_manual() {
        let treasury = HashSet::new();
        let rules = vec![];
        let d = classify_transaction("transfer", "0xuser", "", "1.5", true, &treasury, &rules);
        assert_eq!(d.mode, ApprovalMode::Auto);
        assert!(d.approved);

        let d = classify_transaction("revenue_payout", "0xcold", "", "9999", true, &treasury, &rules);
        assert_eq!(d.mode, ApprovalMode::Manual);

        let d = classify_transaction("treasury_sweep", "0xsafe", "", "0", true, &treasury, &rules);
        assert_eq!(d.mode, ApprovalMode::Manual);
    }

    #[test]
    fn test_classifier_treasury_recipient_manual() {
        let mut treasury = HashSet::new();
        treasury.insert(normalize_addr("0xABC123"));
        let rules = vec![];
        // A "transfer" to a treasury address => Manual (security boundary)
        let d = classify_transaction("transfer", "0xabc123", "", "5.0", true, &treasury, &rules);
        assert_eq!(d.mode, ApprovalMode::Manual);
    }

    #[test]
    fn test_classifier_license_dead_denies() {
        let treasury = HashSet::new();
        let rules = vec![];
        let d = classify_transaction("transfer", "0xuser", "", "1.0", false, &treasury, &rules);
        assert_eq!(d.mode, ApprovalMode::Auto);
        assert!(!d.approved);
    }

    #[test]
    fn test_classifier_blocking_rule_denies() {
        let treasury = HashSet::new();
        let rules = vec![AutoSignRule {
            rule_id: "r1".into(),
            product: "user_wallet".into(),
            fetcher: "*".into(),
            tx_type: "swap".into(),
            token: "*".into(),
            max_amount: "0".into(),
            block: true,
        }];
        let d = classify_transaction("swap", "0xrouter", "", "10", true, &treasury, &rules);
        assert_eq!(d.mode, ApprovalMode::Auto);
        assert!(!d.approved);
        assert_eq!(d.rule_id, "r1");
    }

    #[test]
    fn test_normalize_addr() {
        assert_eq!(normalize_addr("0xABCdef"), "abcdef");
        assert_eq!(normalize_addr("ABCDEF"), "abcdef");
        assert_eq!(normalize_addr(""), "");
    }
}
