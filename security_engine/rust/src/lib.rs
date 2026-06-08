//! TigerWallet Security Engine
//! Signature verification, risk assessment, phishing detection

use serde::{Deserialize, Serialize};
use parking_lot::RwLock;
use std::collections::HashMap;

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
        let mut domains = HashMap::new();
        domains.insert("fake-metamask.com".to_string(), true);
        domains.insert("airdrop-claim.net".to_string(), true);
        domains.insert("free-nft-mint.io".to_string(), true);
        
        Self {
            known_phishing_domains: RwLock::new(domains),
            risk_thresholds: RwLock::new(RiskThresholds {
                low: 25.0,
                medium: 50.0,
                high: 75.0,
            }),
        }
    }

    /// Verify signature
    pub fn verify_signature(&self, signature: &[u8], message: &[u8], public_key: &[u8]) -> SignatureVerifyResult {
        // Mock signature verification
        SignatureVerifyResult {
            valid: !signature.is_empty() && !public_key.is_empty(),
            signer: hex::encode(public_key),
            algorithm: "ECDSA".to_string(),
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
        
        let risk_level = if risk_score >= 75.0 {
            RiskLevel::Critical
        } else if risk_score >= 50.0 {
            RiskLevel::High
        } else if risk_score >= 25.0 {
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

    /// Detect phishing
    pub fn detect_phishing(&self, domain: &str) -> PhishingResult {
        let is_known = self.known_phishing_domains.read().contains_key(domain);
        
        let mut matched_patterns = Vec::new();
        
        // Check suspicious patterns
        if domain.contains("free") || domain.contains("airdrop") {
            matched_patterns.push("Suspicious keyword".to_string());
        }
        if domain.contains("claim") || domain.contains("gift") {
            matched_patterns.push("Airdrop claim pattern".to_string());
        }
        if domain.ends_with(".xyz") || domain.ends_with(".io") {
            matched_patterns.push("Suspicious TLD".to_string());
        }
        
        let confidence = if is_known {
            95.0
        } else if !matched_patterns.is_empty() {
            60.0
        } else {
            0.0
        };
        
        let recommendations = if confidence > 50.0 {
            vec![
                "Verify domain carefully".to_string(),
                "Check official sources".to_string(),
                "Never share seed phrase".to_string(),
            ]
        } else {
            vec![]
        };
        
        PhishingResult {
            is_phishing: is_known || confidence > 50.0,
            confidence,
            domain: domain.to_string(),
            matched_patterns,
            recommendations,
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