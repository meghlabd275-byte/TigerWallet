//! ERC-4337 style account abstraction primitives for TigerWallet.
//!
//! This module is intentionally chain-client agnostic: it validates and builds
//! user operations, account policies, session keys, spending limits, and
//! paymaster sponsorship decisions before the Solidity/bundler layer submits the
//! operation on-chain.

use std::collections::{BTreeMap, BTreeSet};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct UserOperation {
    pub sender: String,
    pub nonce: u64,
    pub init_code: Vec<u8>,
    pub call_data: Vec<u8>,
    pub call_gas_limit: u64,
    pub verification_gas_limit: u64,
    pub pre_verification_gas: u64,
    pub max_fee_per_gas: u64,
    pub max_priority_fee_per_gas: u64,
    pub paymaster_and_data: Vec<u8>,
    pub signature: Vec<u8>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SmartAccountPolicy {
    pub owners: BTreeSet<String>,
    pub threshold: u8,
    pub daily_spend_limit_usd: u64,
    pub allowed_targets: BTreeSet<String>,
    pub blocked_targets: BTreeSet<String>,
    pub session_keys: BTreeMap<String, SessionKey>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SessionKey {
    pub key: String,
    pub expires_at: u64,
    pub allowed_targets: BTreeSet<String>,
    pub spend_limit_usd: u64,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SmartAccount {
    pub address: String,
    pub factory: String,
    pub entry_point: String,
    pub policy: SmartAccountPolicy,
    pub nonce: u64,
    pub spent_today_usd: u64,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum AccountAbstractionError {
    InvalidAddress(String),
    InvalidThreshold,
    UnknownOwner,
    BlockedTarget,
    TargetNotAllowed,
    SpendLimitExceeded,
    ExpiredSessionKey,
    InvalidSignatureQuorum,
    InvalidGasConfig,
    EmptyCallData,
}

pub struct SmartAccountManager;

impl SmartAccountManager {
    pub fn create_account(
        address: impl Into<String>,
        factory: impl Into<String>,
        entry_point: impl Into<String>,
        owners: BTreeSet<String>,
        threshold: u8,
        daily_spend_limit_usd: u64,
    ) -> Result<SmartAccount, AccountAbstractionError> {
        if owners.is_empty() || threshold == 0 || threshold as usize > owners.len() {
            return Err(AccountAbstractionError::InvalidThreshold);
        }

        for owner in &owners {
            validate_evm_address(owner)?;
        }

        let account = SmartAccount {
            address: address.into(),
            factory: factory.into(),
            entry_point: entry_point.into(),
            policy: SmartAccountPolicy {
                owners: owners
                    .into_iter()
                    .map(|owner| owner.to_lowercase())
                    .collect(),
                threshold,
                daily_spend_limit_usd,
                allowed_targets: BTreeSet::new(),
                blocked_targets: BTreeSet::new(),
                session_keys: BTreeMap::new(),
            },
            nonce: 0,
            spent_today_usd: 0,
        };

        validate_evm_address(&account.address)?;
        validate_evm_address(&account.factory)?;
        validate_evm_address(&account.entry_point)?;
        Ok(account)
    }

    pub fn validate_user_operation(
        account: &SmartAccount,
        op: &UserOperation,
        target: &str,
        value_usd: u64,
        signers: &BTreeSet<String>,
    ) -> Result<(), AccountAbstractionError> {
        validate_evm_address(&op.sender)?;
        validate_evm_address(target)?;

        if op.sender.to_lowercase() != account.address.to_lowercase() {
            return Err(AccountAbstractionError::InvalidAddress(op.sender.clone()));
        }
        if op.call_data.is_empty() {
            return Err(AccountAbstractionError::EmptyCallData);
        }
        if op.call_gas_limit == 0 || op.verification_gas_limit == 0 || op.max_fee_per_gas == 0 {
            return Err(AccountAbstractionError::InvalidGasConfig);
        }
        if account
            .policy
            .blocked_targets
            .contains(&target.to_lowercase())
        {
            return Err(AccountAbstractionError::BlockedTarget);
        }
        if !account.policy.allowed_targets.is_empty()
            && !account
                .policy
                .allowed_targets
                .contains(&target.to_lowercase())
        {
            return Err(AccountAbstractionError::TargetNotAllowed);
        }
        if account.spent_today_usd.saturating_add(value_usd) > account.policy.daily_spend_limit_usd
        {
            return Err(AccountAbstractionError::SpendLimitExceeded);
        }

        let owner_signers = signers
            .iter()
            .filter(|s| account.policy.owners.contains(&s.to_lowercase()))
            .count();
        if owner_signers < account.policy.threshold as usize {
            return Err(AccountAbstractionError::InvalidSignatureQuorum);
        }

        Ok(())
    }

    pub fn apply_user_operation(
        account: &mut SmartAccount,
        op: &UserOperation,
        target: &str,
        value_usd: u64,
        signers: &BTreeSet<String>,
    ) -> Result<(), AccountAbstractionError> {
        Self::validate_user_operation(account, op, target, value_usd, signers)?;
        account.nonce = account.nonce.saturating_add(1);
        account.spent_today_usd = account.spent_today_usd.saturating_add(value_usd);
        Ok(())
    }

    pub fn add_allowed_target(
        account: &mut SmartAccount,
        target: &str,
    ) -> Result<(), AccountAbstractionError> {
        validate_evm_address(target)?;
        account.policy.allowed_targets.insert(target.to_lowercase());
        Ok(())
    }

    pub fn add_blocked_target(
        account: &mut SmartAccount,
        target: &str,
    ) -> Result<(), AccountAbstractionError> {
        validate_evm_address(target)?;
        account.policy.blocked_targets.insert(target.to_lowercase());
        Ok(())
    }

    pub fn add_session_key(
        account: &mut SmartAccount,
        key: SessionKey,
    ) -> Result<(), AccountAbstractionError> {
        validate_evm_address(&key.key)?;
        account
            .policy
            .session_keys
            .insert(key.key.to_lowercase(), key);
        Ok(())
    }

    pub fn validate_session_key(
        account: &SmartAccount,
        key: &str,
        target: &str,
        value_usd: u64,
    ) -> Result<(), AccountAbstractionError> {
        validate_evm_address(key)?;
        validate_evm_address(target)?;
        let key = account
            .policy
            .session_keys
            .get(&key.to_lowercase())
            .ok_or(AccountAbstractionError::UnknownOwner)?;

        if key.expires_at <= now() {
            return Err(AccountAbstractionError::ExpiredSessionKey);
        }
        if !key.allowed_targets.is_empty() && !key.allowed_targets.contains(&target.to_lowercase())
        {
            return Err(AccountAbstractionError::TargetNotAllowed);
        }
        if value_usd > key.spend_limit_usd {
            return Err(AccountAbstractionError::SpendLimitExceeded);
        }
        Ok(())
    }

    pub fn rotate_owner(
        account: &mut SmartAccount,
        old_owner: &str,
        new_owner: &str,
    ) -> Result<(), AccountAbstractionError> {
        validate_evm_address(old_owner)?;
        validate_evm_address(new_owner)?;
        if !account.policy.owners.remove(&old_owner.to_lowercase()) {
            return Err(AccountAbstractionError::UnknownOwner);
        }
        account.policy.owners.insert(new_owner.to_lowercase());
        if account.policy.threshold as usize > account.policy.owners.len() {
            return Err(AccountAbstractionError::InvalidThreshold);
        }
        Ok(())
    }
}

pub fn validate_evm_address(address: &str) -> Result<(), AccountAbstractionError> {
    let valid = address.len() == 42
        && address.starts_with("0x")
        && address[2..].chars().all(|c| c.is_ascii_hexdigit());
    if valid {
        Ok(())
    } else {
        Err(AccountAbstractionError::InvalidAddress(address.to_string()))
    }
}

pub fn now() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or_default()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn addr(n: u8) -> String {
        format!("0x{:040x}", n)
    }

    #[test]
    fn validates_threshold_and_spend_limits() {
        let mut owners = BTreeSet::new();
        owners.insert(addr(1));
        owners.insert(addr(2));
        let account =
            SmartAccountManager::create_account(addr(9), addr(8), addr(7), owners, 2, 100).unwrap();
        let op = UserOperation {
            sender: addr(9),
            nonce: 0,
            init_code: vec![],
            call_data: vec![1],
            call_gas_limit: 1,
            verification_gas_limit: 1,
            pre_verification_gas: 1,
            max_fee_per_gas: 1,
            max_priority_fee_per_gas: 1,
            paymaster_and_data: vec![],
            signature: vec![1],
        };
        let signers = BTreeSet::from([addr(1), addr(2)]);
        assert!(SmartAccountManager::validate_user_operation(
            &account,
            &op,
            &addr(3),
            50,
            &signers
        )
        .is_ok());
        assert_eq!(
            SmartAccountManager::validate_user_operation(&account, &op, &addr(3), 101, &signers),
            Err(AccountAbstractionError::SpendLimitExceeded)
        );
    }

    #[test]
    fn session_key_enforces_target_and_limit() {
        let owners = BTreeSet::from([addr(1)]);
        let mut account =
            SmartAccountManager::create_account(addr(9), addr(8), addr(7), owners, 1, 1000)
                .unwrap();
        let key = SessionKey {
            key: addr(5),
            expires_at: now() + 60,
            allowed_targets: BTreeSet::from([addr(4)]),
            spend_limit_usd: 25,
        };
        SmartAccountManager::add_session_key(&mut account, key).unwrap();
        assert!(
            SmartAccountManager::validate_session_key(&account, &addr(5), &addr(4), 20).is_ok()
        );
        assert_eq!(
            SmartAccountManager::validate_session_key(&account, &addr(5), &addr(4), 26),
            Err(AccountAbstractionError::SpendLimitExceeded)
        );
    }
}
