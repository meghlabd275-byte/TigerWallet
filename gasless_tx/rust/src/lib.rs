//! TigerWallet Gasless Transaction Service
//!
//! EIP-2771 compatible meta-transactions with relayer network and gas sponsorship

use std::sync::Arc;
use tokio::sync::RwLock;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

mod relayer;
mod sponsor;
mod forwarder;

pub use relayer::*;
pub use sponsor::*;
pub use forwarder::*;

/// Gasless transaction service
pub struct GaslessService {
    /// Relayer service
    relayer: Arc<RwLock<RelayerService>>,
    /// Sponsor service
    sponsor: Arc<RwLock<SponsorService>>,
    /// Forwarder service
    forwarder: Arc<RwLock<ForwarderService>>,
    /// Chain ID
    chain_id: u64,
    /// Contract address
    contract_address: Option<String>,
}

impl GaslessService {
    /// Create new service
    pub fn new(chain_id: u64) -> Self {
        Self {
            relayer: Arc::new(RwLock::new(RelayerService::new())),
            sponsor: Arc::new(RwLock::new(SponsorService::new())),
            forwarder: Arc::new(RwLock::new(ForwarderService::new())),
            chain_id,
            contract_address: None,
        }
    }

    /// Initialize
    pub async fn initialize(&mut self, contract_address: String) {
        self.contract_address = Some(contract_address);
    }

    /// Build meta transaction
    pub async fn build_meta_tx(
        &self,
        from: &str,
        to: &str,
        value: u64,
        data: Vec<u8>,
        gas_limit: u64,
    ) -> Result<MetaTransaction, GaslessError> {
        let nonce = self.get_nonce(from).await;

        Ok(MetaTransaction {
            from: from.to_string(),
            to: to.to_string(),
            value,
            data,
            gas_limit,
            relayer: "".to_string(),
            nonce,
            signature: vec![],
        })
    }

    /// Signs a meta transaction with the user's own key.
    ///
    /// The key is passed in rather than looked up, so this service never holds
    /// user key material.
    pub async fn sign_meta_tx(
        &self,
        tx: &mut MetaTransaction,
        signing_key: &k256::ecdsa::SigningKey,
    ) -> Result<Vec<u8>, GaslessError> {
        // Build domain separator
        let domain = DomainSeparator {
            name: "TigerWallet Gasless".to_string(),
            version: "1.0.0".to_string(),
            chain_id: self.chain_id,
            verifying_contract: self.contract_address.clone().unwrap_or_default(),
        };

        // Encode transaction
        let encoded = encode_meta_tx(tx);

        // Hash
        let hash = self.hash_meta_tx(&domain, &encoded).await?;

        let signature = sign_message(&hash, signing_key)?;

        // The transaction must be signed by the account it claims to be from.
        let expected = parse_address(&tx.from).ok_or_else(|| {
            GaslessError::RelayerError(format!("from {:?} is not an address", tx.from))
        })?;
        let recovered = recover_signer(&hash, &signature)
            .ok_or_else(|| GaslessError::SigningFailed("signature did not recover".to_string()))?;
        if recovered != expected {
            return Err(GaslessError::SigningFailed(format!(
                "signature is from 0x{}, but the transaction is from {}",
                hex::encode(recovered),
                tx.from
            )));
        }

        tx.signature = signature.clone();

        Ok(signature)
    }

    /// Submits a signed meta transaction to the relayer.
    pub async fn submit_meta_tx(&self, tx: MetaTransaction) -> Result<String, GaslessError> {
        let mut relayer = self.relayer.write().await;
        relayer.submit_transaction(tx).await
    }

    /// Records that a queued transaction was mined.
    pub async fn confirm_meta_tx(
        &self,
        tx_id: &str,
        success: bool,
    ) -> Result<(), GaslessError> {
        let mut relayer = self.relayer.write().await;
        relayer.confirm_transaction(tx_id, success).await
    }

    /// Get nonce
    pub async fn get_nonce(&self, user: &str) -> u64 {
        let relayer = self.relayer.read().await;
        relayer.get_nonce(user)
    }

    /// Get relayer status
    pub async fn get_relayer_status(&self) -> RelayerStatus {
        let relayer = self.relayer.read().await;
        relayer.get_status().await
    }

    /// Register relayer
    pub async fn register_relayer(&self, address: &str) -> Result<(), GaslessError> {
        let mut relayer = self.relayer.write().await;
        relayer.register(address).await
    }

    /// Deregister relayer
    pub async fn deregister_relayer(&self, address: &str) -> Result<(), GaslessError> {
        let mut relayer = self.relayer.write().await;
        relayer.deregister(address).await
    }

    /// Add sponsor
    pub async fn add_sponsor(&self, address: &str, allowance: u64) -> Result<(), GaslessError> {
        let mut sponsor = self.sponsor.write().await;
        sponsor.add_sponsor(address, allowance).await
    }

    /// Sponsors a transaction on behalf of `user`, deducting the gas cost from
    /// `sponsor_address`'s allowance.
    pub async fn sponsor_transaction(
        &self,
        sponsor_address: &str,
        user: &str,
        tx: &MetaTransaction,
    ) -> Result<SponsorshipResult, GaslessError> {
        let mut sponsor = self.sponsor.write().await;
        sponsor.sponsor(sponsor_address, user, tx).await
    }

    /// Grants `user` an allowance from `sponsor_address`.
    pub async fn set_sponsor_allowance(
        &self,
        sponsor_address: &str,
        user: &str,
        amount: u64,
    ) -> Result<(), GaslessError> {
        let mut sponsor = self.sponsor.write().await;
        sponsor.set_allowance(sponsor_address, user, amount).await
    }

    /// Remaining allowance granted to `user` by `sponsor_address`.
    pub async fn remaining_allowance(&self, sponsor_address: &str, user: &str) -> u64 {
        let sponsor = self.sponsor.read().await;
        sponsor.remaining_allowance(sponsor_address, user).await
    }

    /// Set gas policy
    pub async fn set_gas_policy(&self, gas_price: u64, max_gas: u64) -> Result<(), GaslessError> {
        let mut sponsor = self.sponsor.write().await;
        sponsor.set_policy(gas_price, max_gas).await
    }

    /// Add trusted forwarder
    pub async fn add_forwarder(&self, address: &str) -> Result<(), GaslessError> {
        let mut forwarder = self.forwarder.write().await;
        forwarder.add(address).await
    }

    /// Verify forwarder
    pub async fn verify_forwarder(&self, address: &str) -> bool {
        let forwarder = self.forwarder.read().await;
        forwarder.is_trusted(address).await
    }

    /// Hash meta transaction
    async fn hash_meta_tx(
        &self,
        domain: &DomainSeparator,
        encoded: &[u8],
    ) -> Result<Vec<u8>, GaslessError> {
        use ring::digest::digest;

        let mut data = vec![];

        // Domain separator
        data.extend_from_slice(domain.name.as_bytes());
        data.extend_from_slice(domain.version.as_bytes());
        data.extend_from_slice(&domain.chain_id.to_be_bytes());
        data.extend_from_slice(domain.verifying_contract.as_bytes());

        // Encoded transaction
        data.extend_from_slice(encoded);

        let hash = digest(&ring::digest::SHA256, &data);

        Ok(hash.as_ref().to_vec())
    }
}

/// Sign message
fn sign_message(
    message: &[u8],
    signing_key: &k256::ecdsa::SigningKey,
) -> Result<Vec<u8>, GaslessError> {
    use k256::ecdsa::signature::hazmat::PrehashSigner;

    // EIP-191 personal-sign: the message is prefixed before hashing so the
    // signature cannot be replayed as a transaction signature.
    let digest = eip191_digest(message);

    let (signature, recovery_id): (k256::ecdsa::Signature, k256::ecdsa::RecoveryId) = signing_key
        .sign_prehash(&digest)
        .map_err(|e| GaslessError::SigningFailed(e.to_string()))?;

    let mut out = signature.to_bytes().to_vec();
    out.push(27 + recovery_id.to_byte());
    Ok(out)
}

/// The EIP-191 personal-sign digest of `message`.
fn eip191_digest(message: &[u8]) -> [u8; 32] {
    use sha3::{Digest, Keccak256};

    let prefix = format!("\x19Ethereum Signed Message:\n{}", message.len());
    let mut hasher = Keccak256::new();
    hasher.update(prefix.as_bytes());
    hasher.update(message);
    hasher.finalize().into()
}

/// Parses a 0x-prefixed 20-byte address.
pub fn parse_address(value: &str) -> Option<[u8; 20]> {
    let trimmed = value.strip_prefix("0x").or_else(|| value.strip_prefix("0X"))?;
    if trimmed.len() != 40 || !trimmed.bytes().all(|b| b.is_ascii_hexdigit()) {
        return None;
    }
    let bytes = hex::decode(trimmed).ok()?;
    let mut out = [0u8; 20];
    out.copy_from_slice(&bytes);
    Some(out)
}

/// Recovers the address that produced `signature` over `message`, or None when
/// the signature is malformed or does not recover.
pub fn recover_signer(message: &[u8], signature: &[u8]) -> Option<[u8; 20]> {
    use k256::ecdsa::{RecoveryId, Signature, VerifyingKey};
    use sha3::{Digest, Keccak256};

    if signature.len() != 65 {
        return None;
    }

    let v = signature[64];
    let signature = Signature::from_slice(&signature[..64]).ok()?;
    let recovery_id = match v {
        0 | 1 => RecoveryId::from_byte(v)?,
        27 | 28 => RecoveryId::from_byte(v - 27)?,
        _ => return None,
    };

    let digest = eip191_digest(message);
    let key = VerifyingKey::recover_from_prehash(&digest, &signature, recovery_id).ok()?;
    let point = key.to_encoded_point(false);

    let hash = Keccak256::digest(&point.as_bytes()[1..]);
    let mut address = [0u8; 20];
    address.copy_from_slice(&hash[12..]);
    Some(address)
}

/// Encode meta transaction
fn encode_meta_tx(tx: &MetaTransaction) -> Vec<u8> {
    let mut data = vec![];

    data.extend_from_slice(tx.from.as_bytes());
    data.extend_from_slice(tx.to.as_bytes());
    data.extend_from_slice(&tx.value.to_be_bytes());
    data.extend_from_slice(&tx.data);
    data.extend_from_slice(&tx.gas_limit.to_be_bytes());
    data.extend_from_slice(tx.relayer.as_bytes());
    data.extend_from_slice(&tx.nonce.to_be_bytes());

    data
}

/// Meta transaction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetaTransaction {
    pub from: String,
    pub to: String,
    pub value: u64,
    pub data: Vec<u8>,
    pub gas_limit: u64,
    pub relayer: String,
    pub nonce: u64,
    pub signature: Vec<u8>,
}

/// Domain separator
#[derive(Debug, Clone)]
pub struct DomainSeparator {
    pub name: String,
    pub version: String,
    pub chain_id: u64,
    pub verifying_contract: String,
}

/// Relayer status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RelayerStatus {
    pub active_relayers: u32,
    pub pending_transactions: u32,
    pub executed_transactions: u32,
    pub failed_transactions: u32,
}

/// Sponsorship result
#[derive(Debug, Clone)]
pub struct SponsorshipResult {
    pub sponsored: bool,
    pub amount: u64,
    pub expires_at: DateTime<Utc>,
}

/// Gas policy
#[derive(Debug, Clone)]
pub struct GasPolicy {
    pub gas_price: u64,
    pub max_gas: u64,
    pub refund_percent: u32,
}

/// Gasless error
#[derive(Debug, thiserror::Error)]
pub enum GaslessError {
    #[error("Invalid signature")]
    InvalidSignature,

    #[error("Relayer error: {0}")]
    RelayerError(String),

    #[error("Sponsor error: {0}")]
    SponsorError(String),

    #[error("Gas limit exceeded")]
    GasLimitExceeded,

    #[error("Insufficient allowance")]
    InsufficientAllowance,

    #[error("Signing failed: {0}")]
    SigningFailed(String),
}

use thiserror;

#[cfg(test)]
mod tests;
