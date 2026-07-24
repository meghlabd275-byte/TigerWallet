/**
 * TigerWallet Transaction Validator
 * High-Security Rust Implementation
 */

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub hash: String,
    pub from: String,
    pub to: String,
    pub value: String,
    pub gas: String,
    pub gas_price: String,
    pub nonce: u64,
    pub data: String,
    pub chain_id: u64,
    pub timestamp: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationResult {
    pub valid: bool,
    pub errors: Vec<String>,
    pub warnings: Vec<String>,
    pub risk_score: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionValidator {
    blocklist: RwLock<HashMap<String, String>>,
    nonces: RwLock<HashMap<String, u64>>,
    max_value: f64,
    max_gas_price: f64,
}

impl TransactionValidator {
    pub fn new() -> Self {
        Self {
            blocklist: RwLock::new(HashMap::new()),
            nonces: RwLock::new(HashMap::new()),
            max_value: 1_000_000.0,
            max_gas_price: 500.0,
        }
    }

    pub fn validate(&self, tx: &Transaction) -> ValidationResult {
        let mut errors = Vec::new();
        let mut warnings = Vec::new();
        
        // Check blocklist
        if self.blocklist.read().unwrap().contains_key(&tx.from) {
            errors.push("Address is blocklisted".to_string());
        }
        
        // Check nonce
        let nonces = self.nonces.read().unwrap();
        if let Some(&last) = nonces.get(&tx.from) {
            if tx.nonce < last {
                errors.push("Invalid nonce - possible replay".to_string());
            }
        }
        drop(nonces);
        
        // Check value
        let value: f64 = tx.value.parse().unwrap_or(0.0);
        let value_usd = value / 1e18 * 3500.0;
        if value_usd > self.max_value {
            errors.push("Value exceeds maximum".to_string());
        }
        
        // Check gas price
        let gas_price: f64 = tx.gas_price.parse().unwrap_or(0.0);
        let gas_gwei = gas_price / 1e9;
        if gas_gwei > self.max_gas_price {
            errors.push("Gas price too high".to_string());
        }
        
        // Check timestamp
        let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
        if tx.timestamp > now + 60 || now - tx.timestamp > 300 {
            errors.push("Invalid timestamp".to_string());
        }
        
        // Calculate risk score
        let risk_score = (errors.len() as f64 * 30.0) + (warnings.len() as f64 * 10.0);
        
        // Update nonce on success
        if errors.is_empty() {
            let mut nonces = self.nonces.write().unwrap();
            let current = nonces.get(&tx.from).copied().unwrap_or(0);
            if tx.nonce + 1 > current {
                nonces.insert(tx.from.clone(), tx.nonce + 1);
            }
        }
        
        ValidationResult {
            valid: errors.is_empty(),
            errors,
            warnings,
            risk_score: risk_score.min(100.0),
        }
    }
    
    pub fn block(&self, address: &str) {
        self.blocklist.write().unwrap().insert(address.to_string(), "blocked".to_string());
    }
}

fn main() {
    println!("TigerWallet Transaction Validator");
    println!("================================");
    
    let validator = Arc::new(TransactionValidator::new());
    
    let tx = Transaction {
        hash: "0xabc123".to_string(),
        from: "0x742d35Cc6634C0532925a3b844Bc9e7595f".to_string(),
        to: "0x8Ba1f109551bD432803012645Ac136ddd64DBA72".to_string(),
        value: "1000000000000000000".to_string(),
        gas: "21000".to_string(),
        gas_price: "20000000000".to_string(),
        nonce: 0,
        data: "0x".to_string(),
        chain_id: 1,
        timestamp: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
    };
    
    let result = validator.validate(&tx);
    println!("Valid: {}", result.valid);
    println!("Risk Score: {:.1}", result.risk_score);
    
    println!("\nService running on :8087");
}
