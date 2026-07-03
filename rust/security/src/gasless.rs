//! Gasless transaction policy engine for EIP-2771/ERC-4337 relayers.

use std::collections::{BTreeMap, BTreeSet};

use crate::account_abstraction::validate_evm_address;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ForwardRequest {
    pub from: String,
    pub to: String,
    pub value: u128,
    pub gas: u64,
    pub nonce: u64,
    pub deadline: u64,
    pub data: Vec<u8>,
    pub signature: Vec<u8>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SponsorPolicy {
    pub sponsor_id: String,
    pub active: bool,
    pub allowed_callers: BTreeSet<String>,
    pub allowed_targets: BTreeSet<String>,
    pub max_gas_per_tx: u64,
    pub daily_gas_budget: u64,
    pub used_gas_today: u64,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RelayerQuote {
    pub relayer: String,
    pub sponsored: bool,
    pub estimated_gas: u64,
    pub sponsor_id: Option<String>,
    pub user_fee_bps: u16,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum GaslessError {
    InvalidAddress,
    MissingSignature,
    Expired,
    SponsorInactive,
    CallerNotAllowed,
    TargetNotAllowed,
    GasLimitExceeded,
    BudgetExceeded,
    NonceTooLow,
}

#[derive(Debug, Default)]
pub struct GaslessRelayer {
    pub trusted_forwarders: BTreeSet<String>,
    pub sponsor_policies: BTreeMap<String, SponsorPolicy>,
    pub nonces: BTreeMap<String, u64>,
}

impl GaslessRelayer {
    pub fn register_forwarder(&mut self, address: impl Into<String>) -> Result<(), GaslessError> {
        let address = address.into().to_lowercase();
        validate_evm_address(&address).map_err(|_| GaslessError::InvalidAddress)?;
        self.trusted_forwarders.insert(address);
        Ok(())
    }

    pub fn upsert_policy(&mut self, mut policy: SponsorPolicy) {
        policy.allowed_callers = policy
            .allowed_callers
            .into_iter()
            .map(|a| a.to_lowercase())
            .collect();
        policy.allowed_targets = policy
            .allowed_targets
            .into_iter()
            .map(|a| a.to_lowercase())
            .collect();
        self.sponsor_policies
            .insert(policy.sponsor_id.clone(), policy);
    }

    pub fn validate_request(
        &self,
        req: &ForwardRequest,
        sponsor_id: &str,
        now: u64,
    ) -> Result<RelayerQuote, GaslessError> {
        validate_evm_address(&req.from).map_err(|_| GaslessError::InvalidAddress)?;
        validate_evm_address(&req.to).map_err(|_| GaslessError::InvalidAddress)?;
        if req.signature.is_empty() {
            return Err(GaslessError::MissingSignature);
        }
        if req.deadline <= now {
            return Err(GaslessError::Expired);
        }
        let expected_nonce = self
            .nonces
            .get(&req.from.to_lowercase())
            .copied()
            .unwrap_or_default();
        if req.nonce < expected_nonce {
            return Err(GaslessError::NonceTooLow);
        }
        let policy = self
            .sponsor_policies
            .get(sponsor_id)
            .ok_or(GaslessError::SponsorInactive)?;
        if !policy.active {
            return Err(GaslessError::SponsorInactive);
        }
        if !policy.allowed_callers.is_empty()
            && !policy.allowed_callers.contains(&req.from.to_lowercase())
        {
            return Err(GaslessError::CallerNotAllowed);
        }
        if !policy.allowed_targets.is_empty()
            && !policy.allowed_targets.contains(&req.to.to_lowercase())
        {
            return Err(GaslessError::TargetNotAllowed);
        }
        if req.gas > policy.max_gas_per_tx {
            return Err(GaslessError::GasLimitExceeded);
        }
        if policy.used_gas_today.saturating_add(req.gas) > policy.daily_gas_budget {
            return Err(GaslessError::BudgetExceeded);
        }
        Ok(RelayerQuote {
            relayer: "policy-selected".to_string(),
            sponsored: true,
            estimated_gas: req.gas,
            sponsor_id: Some(sponsor_id.to_string()),
            user_fee_bps: 0,
        })
    }

    pub fn reserve_sponsorship(
        &mut self,
        req: &ForwardRequest,
        sponsor_id: &str,
        now: u64,
    ) -> Result<RelayerQuote, GaslessError> {
        let quote = self.validate_request(req, sponsor_id, now)?;
        let policy = self
            .sponsor_policies
            .get_mut(sponsor_id)
            .ok_or(GaslessError::SponsorInactive)?;
        policy.used_gas_today = policy.used_gas_today.saturating_add(req.gas);
        self.nonces
            .insert(req.from.to_lowercase(), req.nonce.saturating_add(1));
        Ok(quote)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    fn addr(n: u8) -> String {
        format!("0x{:040x}", n)
    }

    #[test]
    fn sponsor_policy_enforces_budget() {
        let mut relayer = GaslessRelayer::default();
        relayer.upsert_policy(SponsorPolicy {
            sponsor_id: "consumer-onboarding".into(),
            active: true,
            allowed_callers: BTreeSet::new(),
            allowed_targets: BTreeSet::from([addr(2)]),
            max_gas_per_tx: 100,
            daily_gas_budget: 200,
            used_gas_today: 60,
        });
        let req = ForwardRequest {
            from: addr(1),
            to: addr(2),
            value: 0,
            gas: 100,
            nonce: 0,
            deadline: 100,
            data: vec![1],
            signature: vec![1],
        };
        assert_eq!(
            relayer
                .validate_request(&req, "consumer-onboarding", 1)
                .unwrap()
                .sponsored,
            true
        );
        let mut high = req.clone();
        high.gas = 101;
        assert_eq!(
            relayer.validate_request(&high, "consumer-onboarding", 1),
            Err(GaslessError::GasLimitExceeded)
        );
    }
}
