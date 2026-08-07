//! Account-abstraction request primitives with strict validation.
use sha3::{Digest, Keccak256};
use thiserror::Error;

#[derive(Debug, Error)]
pub enum AccountAbstractionError {
    #[error("sender must be a 20-byte EVM address")]
    InvalidSender,
    #[error("paymaster and paymaster data must be supplied together")]
    InvalidPaymaster,
    #[error("verification gas limit must be non-zero")]
    InvalidVerificationGas,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct UserOperation {
    pub sender: [u8; 20],
    pub nonce: u64,
    pub init_code: Vec<u8>,
    pub call_data: Vec<u8>,
    pub call_gas_limit: u64,
    pub verification_gas_limit: u64,
    pub pre_verification_gas: u64,
    pub max_fee_per_gas: u128,
    pub max_priority_fee_per_gas: u128,
    pub paymaster_and_data: Vec<u8>,
    pub signature: Vec<u8>,
}

impl UserOperation {
    pub fn validate(&self) -> Result<(), AccountAbstractionError> {
        if self.sender == [0u8; 20] {
            return Err(AccountAbstractionError::InvalidSender);
        }
        if self.verification_gas_limit == 0 {
            return Err(AccountAbstractionError::InvalidVerificationGas);
        }
        if self.paymaster_and_data.is_empty() && !self.signature.is_empty() {
            return Ok(());
        }
        if self.paymaster_and_data.len() == 1 {
            return Err(AccountAbstractionError::InvalidPaymaster);
        }
        Ok(())
    }

    pub fn request_hash(&self, entry_point: [u8; 20], chain_id: u64) -> Result<[u8; 32], AccountAbstractionError> {
        self.validate()?;
        let mut inner = Keccak256::new();
        inner.update(self.sender);
        inner.update(self.nonce.to_be_bytes());
        inner.update(Keccak256::digest(&self.init_code));
        inner.update(Keccak256::digest(&self.call_data));
        inner.update(self.call_gas_limit.to_be_bytes());
        inner.update(self.verification_gas_limit.to_be_bytes());
        inner.update(self.pre_verification_gas.to_be_bytes());
        inner.update(self.max_fee_per_gas.to_be_bytes());
        inner.update(self.max_priority_fee_per_gas.to_be_bytes());
        inner.update(Keccak256::digest(&self.paymaster_and_data));
        let packed = inner.finalize();

        let mut outer = Keccak256::new();
        outer.update(packed);
        outer.update(entry_point);
        outer.update(chain_id.to_be_bytes());
        Ok(outer.finalize().into())
    }
}
