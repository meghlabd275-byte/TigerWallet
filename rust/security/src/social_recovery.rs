//! Guardian-based social recovery with timelocks and anti-scam delay controls.

use std::collections::{BTreeMap, BTreeSet};

use crate::account_abstraction::validate_evm_address;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GuardianConfig {
    pub wallet: String,
    pub guardians: BTreeSet<String>,
    pub threshold: u8,
    pub timelock_seconds: u64,
    pub emergency_contacts: BTreeSet<String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RecoveryRequest {
    pub id: String,
    pub wallet: String,
    pub proposed_owner: String,
    pub requested_at: u64,
    pub executable_at: u64,
    pub approvals: BTreeSet<String>,
    pub cancelled: bool,
    pub executed: bool,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum RecoveryError {
    InvalidAddress,
    InvalidThreshold,
    UnknownGuardian,
    DuplicateRequest,
    RequestNotFound,
    TimelockActive,
    ThresholdNotMet,
    Cancelled,
    AlreadyExecuted,
}

#[derive(Debug, Default)]
pub struct SocialRecoveryEngine {
    configs: BTreeMap<String, GuardianConfig>,
    requests: BTreeMap<String, RecoveryRequest>,
}

impl SocialRecoveryEngine {
    pub fn configure(&mut self, config: GuardianConfig) -> Result<(), RecoveryError> {
        validate_evm_address(&config.wallet).map_err(|_| RecoveryError::InvalidAddress)?;
        if config.guardians.is_empty()
            || config.threshold == 0
            || config.threshold as usize > config.guardians.len()
        {
            return Err(RecoveryError::InvalidThreshold);
        }
        for guardian in &config.guardians {
            validate_evm_address(guardian).map_err(|_| RecoveryError::InvalidAddress)?;
        }
        self.configs
            .insert(config.wallet.to_lowercase(), normalize_config(config));
        Ok(())
    }

    pub fn request_recovery(
        &mut self,
        id: impl Into<String>,
        wallet: &str,
        proposed_owner: &str,
        requested_at: u64,
    ) -> Result<(), RecoveryError> {
        validate_evm_address(wallet).map_err(|_| RecoveryError::InvalidAddress)?;
        validate_evm_address(proposed_owner).map_err(|_| RecoveryError::InvalidAddress)?;
        let id = id.into();
        if self.requests.contains_key(&id) {
            return Err(RecoveryError::DuplicateRequest);
        }
        let cfg = self
            .configs
            .get(&wallet.to_lowercase())
            .ok_or(RecoveryError::RequestNotFound)?;
        self.requests.insert(
            id.clone(),
            RecoveryRequest {
                id,
                wallet: wallet.to_lowercase(),
                proposed_owner: proposed_owner.to_lowercase(),
                requested_at,
                executable_at: requested_at.saturating_add(cfg.timelock_seconds),
                approvals: BTreeSet::new(),
                cancelled: false,
                executed: false,
            },
        );
        Ok(())
    }

    pub fn approve(&mut self, id: &str, guardian: &str) -> Result<(), RecoveryError> {
        validate_evm_address(guardian).map_err(|_| RecoveryError::InvalidAddress)?;
        let req = self
            .requests
            .get_mut(id)
            .ok_or(RecoveryError::RequestNotFound)?;
        let cfg = self
            .configs
            .get(&req.wallet)
            .ok_or(RecoveryError::RequestNotFound)?;
        if !cfg.guardians.contains(&guardian.to_lowercase()) {
            return Err(RecoveryError::UnknownGuardian);
        }
        if req.cancelled {
            return Err(RecoveryError::Cancelled);
        }
        if req.executed {
            return Err(RecoveryError::AlreadyExecuted);
        }
        req.approvals.insert(guardian.to_lowercase());
        Ok(())
    }

    pub fn cancel(&mut self, id: &str) -> Result<(), RecoveryError> {
        let req = self
            .requests
            .get_mut(id)
            .ok_or(RecoveryError::RequestNotFound)?;
        if req.executed {
            return Err(RecoveryError::AlreadyExecuted);
        }
        req.cancelled = true;
        Ok(())
    }

    pub fn execute(&mut self, id: &str, now: u64) -> Result<String, RecoveryError> {
        let req = self
            .requests
            .get_mut(id)
            .ok_or(RecoveryError::RequestNotFound)?;
        let cfg = self
            .configs
            .get(&req.wallet)
            .ok_or(RecoveryError::RequestNotFound)?;
        if req.cancelled {
            return Err(RecoveryError::Cancelled);
        }
        if req.executed {
            return Err(RecoveryError::AlreadyExecuted);
        }
        if now < req.executable_at {
            return Err(RecoveryError::TimelockActive);
        }
        if req.approvals.len() < cfg.threshold as usize {
            return Err(RecoveryError::ThresholdNotMet);
        }
        req.executed = true;
        Ok(req.proposed_owner.clone())
    }

    pub fn request(&self, id: &str) -> Option<&RecoveryRequest> {
        self.requests.get(id)
    }
}

fn normalize_config(mut config: GuardianConfig) -> GuardianConfig {
    config.wallet = config.wallet.to_lowercase();
    config.guardians = config
        .guardians
        .into_iter()
        .map(|g| g.to_lowercase())
        .collect();
    config.emergency_contacts = config
        .emergency_contacts
        .into_iter()
        .map(|g| g.to_lowercase())
        .collect();
    config
}

#[cfg(test)]
mod tests {
    use super::*;
    fn addr(n: u8) -> String {
        format!("0x{:040x}", n)
    }

    #[test]
    fn timelock_and_threshold_are_required() {
        let mut engine = SocialRecoveryEngine::default();
        engine
            .configure(GuardianConfig {
                wallet: addr(1),
                guardians: BTreeSet::from([addr(2), addr(3)]),
                threshold: 2,
                timelock_seconds: 10,
                emergency_contacts: BTreeSet::new(),
            })
            .unwrap();
        engine
            .request_recovery("r1", &addr(1), &addr(9), 100)
            .unwrap();
        engine.approve("r1", &addr(2)).unwrap();
        assert_eq!(
            engine.execute("r1", 111),
            Err(RecoveryError::ThresholdNotMet)
        );
        engine.approve("r1", &addr(3)).unwrap();
        assert_eq!(
            engine.execute("r1", 109),
            Err(RecoveryError::TimelockActive)
        );
        assert_eq!(engine.execute("r1", 110).unwrap(), addr(9));
    }
}
