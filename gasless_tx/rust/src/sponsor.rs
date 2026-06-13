//! Sponsor service for gasless transactions

use crate::{MetaTransaction, SponsorshipResult, GasPolicy, GaslessError};
use std::collections::HashMap;
use tokio::sync::RwLock;
use chrono::Utc;

/// Sponsor information
#[derive(Debug, Clone)]
pub struct SponsorInfo {
    pub address: String,
    pub allowance: u64,
    pub spent: u64,
    pub active: bool,
}

/// Sponsor service
pub struct SponsorService {
    /// Sponsors
    sponsors: HashMap<String, SponsorInfo>,
    /// Allowances
    allowances: HashMap<String, HashMap<String, u64>>,
    /// Gas policy
    policy: GasPolicy,
}

impl SponsorService {
    pub fn new() -> Self {
        Self {
            sponsors: HashMap::new(),
            allowances: HashMap::new(),
            policy: GasPolicy {
                gas_price: 30_000_000_000, // 30 gwei
                max_gas: 500000,
                refund_percent: 110,
            },
        }
    }

    /// Add sponsor
    pub async fn add_sponsor(&mut self, address: &str, allowance: u64) -> Result<(), GaslessError> {
        self.sponsors.insert(address.to_string(), SponsorInfo {
            address: address.to_string(),
            allowance,
            spent: 0,
            active: true,
        });
        Ok(())
    }

    /// Remove sponsor
    pub async fn remove_sponsor(&mut self, address: &str) -> Result<(), GaslessError> {
        self.sponsors.remove(address);
        Ok(())
    }

    /// Sponsor transaction
    pub async fn sponsor(&self, tx: &MetaTransaction) -> Result<SponsorshipResult, GaslessError> {
        // Calculate gas cost
        let gas_cost = tx.gas_limit * self.policy.gas_price;
        let refund = gas_cost * self.policy.refund_percent as u64 / 100;
        
        Ok(SponsorshipResult {
            sponsored: true,
            amount: refund,
            expires_at: Utc::now(),
        })
    }

    /// Set policy
    pub async fn set_policy(&mut self, gas_price: u64, max_gas: u64) -> Result<(), GaslessError> {
        self.policy.gas_price = gas_price;
        self.policy.max_gas = max_gas;
        Ok(())
    }
}

impl Default for SponsorService {
    fn default() -> Self {
        Self::new()
    }
}