//! Paymaster service for gas sponsorship

use crate::{UserOperation, AAError, SponsoredOp};
use chrono::Utc;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Token configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenConfig {
    pub token: String,
    pub exchange_rate: u64,
    pub decimals: u8,
    pub accepted: bool,
}

/// Sponsor policy
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SponsorPolicy {
    pub max_gas_limit: u64,
    pub max_sponsor_amount: u64,
    pub gas_buffer_percent: u32,
    pub cooldown_seconds: i64,
    pub min_stake: u64,
    pub whitelisted_senders: Vec<String>,
}

/// Paymaster service
pub struct PaymasterService {
    /// Entry point
    entry_point: Option<String>,
    /// Paymaster address
    paymaster_address: Option<String>,
    /// Deposits
    deposits: HashMap<String, u64>,
    /// Token configurations
    tokens: HashMap<String, TokenConfig>,
    /// Sponsor policies
    policies: Vec<SponsorPolicy>,
    /// Current policy index
    current_policy: usize,
    /// Total sponsored
    total_sponsored: u64,
    /// Whitelist
    whitelisted: HashMap<String, bool>,
    /// User limits
    user_limits: HashMap<String, u64>,
    /// Owner
    owner: Option<String>,
}

impl PaymasterService {
    pub fn new() -> Self {
        Self {
            entry_point: None,
            paymaster_address: None,
            deposits: HashMap::new(),
            tokens: HashMap::new(),
            policies: vec![SponsorPolicy {
                max_gas_limit: 1_000_000,
                max_sponsor_amount: 10_000_000_000_000_000, // 0.01 ETH
                gas_buffer_percent: 20,
                cooldown_seconds: 3600,
                min_stake: 1e18 as u64,
                whitelisted_senders: vec![],
            }],
            current_policy: 0,
            total_sponsored: 0,
            whitelisted: HashMap::new(),
            user_limits: HashMap::new(),
            owner: None,
        }
    }

    /// Set entry point
    pub fn set_entry_point(&mut self, entry_point: String) {
        self.entry_point = Some(entry_point);
    }

    /// Set paymaster address
    pub fn set_paymaster_address(&mut self, address: String) {
        self.paymaster_address = Some(address);
    }

    /// Set owner
    pub fn set_owner(&mut self, owner: String) {
        self.owner = Some(owner);
    }

    /// Replace the sponsor policy's sender whitelist.
    ///
    /// An empty list means every sender may be sponsored; a non-empty list
    /// restricts sponsorship to the addresses it names.
    pub fn set_whitelisted_senders(&mut self, senders: Vec<String>) {
        if let Some(policy) = self.policies.get_mut(self.current_policy) {
            policy.whitelisted_senders = senders;
        }
    }

    /// The current sponsor policy, if one is configured.
    pub fn current_policy(&self) -> Option<&SponsorPolicy> {
        self.policies.get(self.current_policy)
    }

    /// Deposit
    pub async fn deposit(&mut self, account: &str, amount: u64) -> Result<(), AAError> {
        if amount == 0 {
            return Err(AAError::PaymasterError("Deposit must be greater than zero".to_string()));
        }
        let deposit = self.deposits.entry(account.to_string()).or_insert(0);
        *deposit += amount;
        Ok(())
    }

    /// Withdraw
    ///
    /// Only the paymaster owner may move a deposit out.
    pub async fn withdraw(
        &mut self,
        account: &str,
        amount: u64,
        recipient: &str,
    ) -> Result<(), AAError> {
        match &self.owner {
            Some(owner) if owner == account => {}
            _ => {
                return Err(AAError::PaymasterError(
                    "Only the paymaster owner may withdraw".to_string(),
                ))
            }
        }

        if recipient.is_empty() {
            return Err(AAError::PaymasterError(
                "A withdrawal recipient is required".to_string(),
            ));
        }

        if amount == 0 {
            return Err(AAError::PaymasterError(
                "Withdrawal amount must be greater than zero".to_string(),
            ));
        }

        let deposit = self
            .deposits
            .get_mut(account)
            .ok_or_else(|| AAError::PaymasterError("No deposit".to_string()))?;

        if *deposit < amount {
            return Err(AAError::PaymasterError("Insufficient deposit".to_string()));
        }

        *deposit -= amount;

        Ok(())
    }

    /// Get deposit
    pub async fn get_deposit(&self, account: &str) -> Result<u64, AAError> {
        Ok(*self.deposits.get(account).unwrap_or(&0))
    }

    /// Add token
    pub async fn add_token(
        &mut self,
        token: String,
        exchange_rate: u64,
        decimals: u8,
    ) -> Result<(), AAError> {
        if exchange_rate == 0 {
            return Err(AAError::PaymasterError("Invalid rate".to_string()));
        }
        
        self.tokens.insert(token.clone(), TokenConfig {
            token,
            exchange_rate,
            decimals,
            accepted: true,
        });
        
        Ok(())
    }

    /// Remove token
    pub async fn remove_token(&mut self, token: &str) -> Result<(), AAError> {
        if let Some(config) = self.tokens.get_mut(token) {
            config.accepted = false;
            Ok(())
        } else {
            Err(AAError::PaymasterError("Token not found".to_string()))
        }
    }

    /// Sponsor user operation
    pub async fn sponsor(&mut self, user_op: &UserOperation) -> Result<SponsoredOp, AAError> {
        // Check policy
        let policy = self.policies.get(self.current_policy)
            .ok_or(AAError::PaymasterError("No policy".to_string()))?;
        
        // Calculate gas cost
        let gas_cost = self.calculate_gas_cost(user_op)?;
        
        // Check limits
        if gas_cost > policy.max_sponsor_amount {
            return Err(AAError::PaymasterError("Exceeds max sponsor amount".to_string()));
        }
        
        if user_op.call_gas_limit + user_op.verification_gas_limit > policy.max_gas_limit {
            return Err(AAError::PaymasterError("Exceeds max gas limit".to_string()));
        }
        
        // Check whitelist if sender not whitelisted
        if !policy.whitelisted_senders.is_empty()
            && !policy.whitelisted_senders.contains(&user_op.sender)
        {
            return Err(AAError::PaymasterError(
                "This sender is not on the sponsorship whitelist".to_string(),
            ));
        }

        // A per-sender limit, when set, caps the sponsorship for that sender.
        if let Some(limit) = self.user_limits.get(&user_op.sender) {
            if gas_cost > *limit {
                return Err(AAError::PaymasterError(format!(
                    "Sponsorship of {gas_cost} exceeds the limit of {limit} for this sender"
                )));
            }
        }

        
        self.total_sponsored = self.total_sponsored.saturating_add(gas_cost);
        
        Ok(SponsoredOp {
            user_op: user_op.clone(),
            sponsor_amount: gas_cost,
            expires_at: Utc::now() + chrono::Duration::seconds(policy.cooldown_seconds),
        })
    }

    /// Calculate gas cost
    fn calculate_gas_cost(&self, user_op: &UserOperation) -> Result<u64, AAError> {
        // Base fee
        let base_fee = 30_000_000_000u64; // 30 gwei
        
        let total_gas = user_op.call_gas_limit + 
            user_op.verification_gas_limit + 
            user_op.pre_verification_gas;
        
        let cost = total_gas * base_fee;
        
        // Add buffer
        let policy = self.policies.get(self.current_policy)
            .ok_or(AAError::PaymasterError("No policy".to_string()))?;
        
        let buffered = cost * (100 + policy.gas_buffer_percent as u64) / 100;
        
        Ok(buffered)
    }

    /// Set whitelist
    pub async fn set_whitelist(&mut self, sender: &str, allowed: bool) {
        self.whitelisted.insert(sender.to_string(), allowed);
    }

    /// Set user limit
    pub async fn set_user_limit(&mut self, sender: &str, limit: u64) {
        self.user_limits.insert(sender.to_string(), limit);
    }

    /// Get token count
    pub fn get_token_count(&self) -> usize {
        self.tokens.len()
    }

    /// Get accepted tokens
    pub fn get_accepted_tokens(&self) -> Vec<String> {
        self.tokens.values()
            .filter(|t| t.accepted)
            .map(|t| t.token.clone())
            .collect()
    }
}

impl Default for PaymasterService {
    fn default() -> Self {
        Self::new()
    }
}