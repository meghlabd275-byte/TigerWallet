//! TigerWallet Token Scanner
//! Honeypot detection, rugpull detection, contract analysis

use rust_decimal::Decimal;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenAnalysis {
    pub address: String,
    pub risk_score: u8,
    pub is_honeypot: bool,
    pub is_rugpull: bool,
    pub liquidity_locked: bool,
    pub owner_renounced: bool,
    pub mint_authority: String,
    pub liquidity_pairs: Vec<String>,
    pub warnings: Vec<String>,
}

impl TokenAnalysis {
    pub fn new(address: &str) -> Self {
        Self {
            address: address.to_string(),
            risk_score: 0,
            is_honeypot: false,
            is_rugpull: false,
            liquidity_locked: false,
            owner_renounced: false,
            mint_authority: "".to_string(),
            liquidity_pairs: Vec::new(),
            warnings: Vec::new(),
        }
    }

    pub fn analyze(&mut self, contract_code: &str) {
        // Check for honeypot patterns
        if contract_code.contains("require(false") || contract_code.contains("revert()") {
            self.risk_score += 30;
            self.warnings.push("Potential honeypot pattern detected".to_string());
        }

        // Check liquidity
        if !self.liquidity_locked {
            self.risk_score += 20;
            self.warnings.push("Liquidity not locked".to_string());
        }

        // Check owner
        if !self.owner_renounced {
            self.risk_score += 10;
            self.warnings.push("Owner not renounced".to_string());
        }

        self.is_honeypot = self.risk_score > 50;
        self.is_rugpull = self.risk_score > 70;
    }
}