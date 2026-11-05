//! Sponsor service for gasless transactions

use crate::{MetaTransaction, SponsorshipResult, GasPolicy, GaslessError};
use std::collections::HashMap;
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
    /// Sponsors, keyed by address.
    sponsors: HashMap<String, SponsorInfo>,
    /// Per-sponsor, per-user allowance in wei.
    allowances: HashMap<String, HashMap<String, u64>>,
    /// Gas policy
    policy: GasPolicy,
}

impl SponsorService {
    pub fn new() -> Self {
        Self {
            sponsors: HashMap::new(),
            allowances: HashMap::new(),
            // No default gas price. A hardcoded 30 gwei would silently price
            // every sponsorship, and on a chain whose gas costs a fraction of
            // that the treasury would be drained by refunds it never owed.
            // `set_policy` must be called before anything is sponsored.
            policy: GasPolicy {
                gas_price: 0,
                max_gas: 0,
                refund_percent: 0,
            },
        }
    }

    /// Registers a sponsor with a total allowance in wei.
    pub async fn add_sponsor(&mut self, address: &str, allowance: u64) -> Result<(), GaslessError> {
        if address.trim().is_empty() {
            return Err(GaslessError::SponsorError(
                "sponsor address must not be empty".to_string(),
            ));
        }
        self.sponsors.insert(
            address.to_string(),
            SponsorInfo {
                address: address.to_string(),
                allowance,
                spent: 0,
                active: true,
            },
        );
        Ok(())
    }

    /// Removes a sponsor and every allowance it had granted.
    pub async fn remove_sponsor(&mut self, address: &str) -> Result<(), GaslessError> {
        if self.sponsors.remove(address).is_none() {
            return Err(GaslessError::SponsorError(format!(
                "{} is not a registered sponsor",
                address
            )));
        }
        self.allowances.remove(address);
        Ok(())
    }

    /// Grants `user` a per-sponsor allowance of `amount` wei.
    pub async fn set_allowance(
        &mut self,
        sponsor: &str,
        user: &str,
        amount: u64,
    ) -> Result<(), GaslessError> {
        if !self.sponsors.contains_key(sponsor) {
            return Err(GaslessError::SponsorError(format!(
                "{} is not a registered sponsor",
                sponsor
            )));
        }
        self.allowances
            .entry(sponsor.to_string())
            .or_default()
            .insert(user.to_string(), amount);
        Ok(())
    }

    /// Returns the remaining allowance `sponsor` has granted `user`.
    pub async fn remaining_allowance(&self, sponsor: &str, user: &str) -> u64 {
        self.allowances
            .get(sponsor)
            .and_then(|by_user| by_user.get(user))
            .copied()
            .unwrap_or(0)
    }

    /// Sponsors `tx`, deducting the gas cost from the sponsor's allowance.
    ///
    /// The previous version returned `sponsored: true` unconditionally,
    /// computed a refund from a hardcoded gas price, and never consulted or
    /// decremented the `allowances` map it held. Any transaction was therefore
    /// sponsored at the treasury's expense. The checks below are the ones that
    /// were missing: the policy must be configured, the sponsor must be active
    /// and funded, and the caller must have an allowance.
    pub async fn sponsor(
        &mut self,
        sponsor_address: &str,
        user: &str,
        tx: &MetaTransaction,
    ) -> Result<SponsorshipResult, GaslessError> {
        if self.policy.gas_price == 0 || self.policy.max_gas == 0 {
            return Err(GaslessError::SponsorError(
                "gas policy has not been configured".to_string(),
            ));
        }
        if tx.gas_limit > self.policy.max_gas {
            return Err(GaslessError::GasLimitExceeded);
        }

        let gas_cost = tx
            .gas_limit
            .checked_mul(self.policy.gas_price)
            .ok_or_else(|| GaslessError::SponsorError("gas cost overflowed".to_string()))?;

        // The allowance the sponsor granted this user.
        let user_allowance = self
            .allowances
            .get(sponsor_address)
            .and_then(|by_user| by_user.get(user))
            .copied()
            .unwrap_or(0);
        if user_allowance < gas_cost {
            return Err(GaslessError::InsufficientAllowance);
        }

        let sponsor = self
            .sponsors
            .get_mut(sponsor_address)
            .ok_or_else(|| {
                GaslessError::SponsorError(format!("{} is not a registered sponsor", sponsor_address))
            })?;

        if !sponsor.active {
            return Err(GaslessError::SponsorError(format!(
                "sponsor {} is not active",
                sponsor_address
            )));
        }

        let remaining = sponsor.allowance.saturating_sub(sponsor.spent);
        if remaining < gas_cost {
            return Err(GaslessError::InsufficientAllowance);
        }

        // Deduct from both the sponsor's budget and the user's allowance.
        sponsor.spent += gas_cost;
        if let Some(by_user) = self.allowances.get_mut(sponsor_address) {
            if let Some(entry) = by_user.get_mut(user) {
                *entry -= gas_cost;
            }
        }

        let refund = gas_cost
            .checked_mul(self.policy.refund_percent as u64)
            .ok_or_else(|| GaslessError::SponsorError("refund overflowed".to_string()))?
            / 100;

        Ok(SponsorshipResult {
            sponsored: true,
            amount: refund,
            expires_at: Utc::now(),
        })
    }

    /// Sets the gas policy. A refund below 100% under-refunds the user, and a
    /// gas price of zero would make every sponsorship free.
    pub async fn set_policy(&mut self, gas_price: u64, max_gas: u64) -> Result<(), GaslessError> {
        if gas_price == 0 {
            return Err(GaslessError::SponsorError(
                "gas_price must be greater than zero".to_string(),
            ));
        }
        if max_gas == 0 {
            return Err(GaslessError::SponsorError(
                "max_gas must be greater than zero".to_string(),
            ));
        }
        self.policy.gas_price = gas_price;
        self.policy.max_gas = max_gas;
        Ok(())
    }

    /// The configured gas policy.
    pub fn policy(&self) -> &GasPolicy {
        &self.policy
    }
}

impl Default for SponsorService {
    fn default() -> Self {
        Self::new()
    }
}