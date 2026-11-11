//! TigerWallet Security Engine
//! Signature verification, risk assessment, phishing detection

use serde::{Deserialize, Serialize};
use parking_lot::RwLock;
use std::collections::HashMap;

pub mod security;
pub mod phishing;
pub mod signature;

#[cfg(test)]
mod kdf_tests;
#[cfg(test)]
mod phishing_tests;
#[cfg(test)]
mod signature_tests;

pub use phishing::{heuristic_score, normalize_host, PhishingVerdict};
pub use signature::{
    decode_hex, evm_address_from_public_key, keccak256, verify_auto, verify_eip191_message,
    verify_ed25519, verify_secp256k1_digest, SignatureAlgorithm, VerifiedSignature,
};

/// Signature verification result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignatureVerifyResult {
    pub valid: bool,
    pub signer: String,
    pub algorithm: String,
}

/// Transaction risk assessment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionRisk {
    pub risk_score: f64,
    pub risk_level: RiskLevel,
    pub warnings: Vec<String>,
    pub factors: Vec<RiskFactor>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum RiskLevel {
    Low,
    Medium,
    High,
    Critical,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RiskFactor {
    HighValue,
    NewRecipient,
    UnusualToken,
    SmartContractInteraction,
    FlashLoan,
    CrossChain,
    UnverifiedContract,
    Honeypot,
    PhishingAttempt,
}

/// Phishing detection result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PhishingResult {
    pub is_phishing: bool,
    pub confidence: f64,
    pub domain: String,
    pub matched_patterns: Vec<String>,
    pub recommendations: Vec<String>,
}

/// Security engine
pub struct SecurityEngine {
    known_phishing_domains: RwLock<HashMap<String, bool>>,
    risk_thresholds: RwLock<RiskThresholds>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskThresholds {
    pub low: f64,
    pub medium: f64,
    pub high: f64,
}

impl SecurityEngine {
    pub fn new() -> Self {
        Self {
            known_phishing_domains: RwLock::new(HashMap::new()),
            risk_thresholds: RwLock::new(RiskThresholds {
                low: 25.0,
                medium: 50.0,
                high: 75.0,
            }),
        }
    }

    /// Verify a signature against a public key.
    ///
    /// The algorithm is selected from the public key length and the signature
    /// is checked cryptographically. Malformed input verifies as invalid.
    pub fn verify_signature(&self, signature: &[u8], message: &[u8], public_key: &[u8]) -> SignatureVerifyResult {
        let verified = verify_auto(message, signature, public_key);
        SignatureVerifyResult {
            valid: verified.valid,
            signer: verified.signer,
            algorithm: verified.algorithm.as_str().to_string(),
        }
    }

    /// Verify an EIP-191 personal-sign message (EVM chains).
    pub fn verify_evm_personal_sign(&self, signature: &[u8], message: &[u8], public_key: &[u8]) -> SignatureVerifyResult {
        let verified = verify_eip191_message(message, signature, public_key);
        SignatureVerifyResult {
            valid: verified.valid,
            signer: verified.signer,
            algorithm: verified.algorithm.as_str().to_string(),
        }
    }

    /// Verify an Ed25519 signature (Solana, Cosmos, NEAR, Sui).
    pub fn verify_ed25519_signature(&self, signature: &[u8], message: &[u8], public_key: &[u8]) -> SignatureVerifyResult {
        let verified = verify_ed25519(message, signature, public_key);
        SignatureVerifyResult {
            valid: verified.valid,
            signer: verified.signer,
            algorithm: verified.algorithm.as_str().to_string(),
        }
    }

    /// Assess transaction risk
    pub fn assess_transaction_risk(&self, tx: &Transaction) -> TransactionRisk {
        let mut risk_score = 0.0;
        let mut warnings = Vec::new();
        let mut factors = Vec::new();
        
        // Check value
        if tx.value_usd > 100000.0 {
            risk_score += 30.0;
            warnings.push("High value transaction".to_string());
            factors.push(RiskFactor::HighValue);
        }
        
        // Check recipient age
        if tx.recipient_age_hours < 24 {
            risk_score += 20.0;
            warnings.push("New recipient address".to_string());
            factors.push(RiskFactor::NewRecipient);
        }
        
        // Check contract interaction
        if tx.interacts_with_contract {
            risk_score += 15.0;
            warnings.push("Smart contract interaction".to_string());
            factors.push(RiskFactor::SmartContractInteraction);
        }
        
        // Check unverified contract
        if tx.contract_verified == Some(false) {
            risk_score += 25.0;
            warnings.push("Unverified contract".to_string());
            factors.push(RiskFactor::UnverifiedContract);
        }
        
        // Use the operator-configured thresholds rather than fixed values, so
        // tightening the engine takes effect everywhere at once.
        let thresholds = self.risk_thresholds.read();
        let risk_level = if risk_score >= thresholds.high {
            RiskLevel::Critical
        } else if risk_score >= thresholds.medium {
            RiskLevel::High
        } else if risk_score >= thresholds.low {
            RiskLevel::Medium
        } else {
            RiskLevel::Low
        };
        
        TransactionRisk {
            risk_score: min(risk_score, 100.0),
            risk_level,
            warnings,
            factors,
        }
    }

    /// Detect phishing.
    ///
    /// Combines the operator-managed blocklist with the heuristic scorer in
    /// `phishing`. A blocklist hit is conclusive; otherwise the domain must
    /// accumulate enough independent signals to be flagged.
    pub fn detect_phishing(&self, domain: &str) -> PhishingResult {
        let blocklist: Vec<String> = self
            .known_phishing_domains
            .read()
            .keys()
            .cloned()
            .collect();

        let verdict = phishing::inspect(domain, &blocklist, &[]);

        PhishingResult {
            is_phishing: verdict.is_phishing,
            confidence: verdict.confidence,
            domain: verdict.domain,
            matched_patterns: verdict.signals,
            recommendations: verdict.recommendations,
        }
    }

    /// Add phishing domain to blocklist
    pub fn block_domain(&self, domain: &str) {
        self.known_phishing_domains.write()
            .insert(domain.to_string(), true);
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub from: String,
    pub to: String,
    pub value_usd: f64,
    pub token: Option<String>,
    pub chain: String,
    pub recipient_age_hours: i64,
    pub interacts_with_contract: bool,
    pub contract_verified: Option<bool>,
    pub data: Option<String>,
}

fn min(a: f64, b: f64) -> f64 {
    if a < b { a } else { b }
}

impl Default for SecurityEngine {
    fn default() -> Self {
        Self::new()
    }
}