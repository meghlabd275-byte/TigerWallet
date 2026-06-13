//! TigerWallet Gasless Transaction Service
//! 
//! EIP-2771 compatible meta-transactions with relayer network and gas sponsorship

use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use serde::{Deserialize, Serialize};
use uuid::Uuid;
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

    /// Sign meta transaction
    pub async fn sign_meta_tx(
        &self,
        tx: &mut MetaTransaction,
        signer: &str,
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
        
        // Sign (would use actual signing in production)
        let signature = sign_message(&hash, signer).await?;
        
        tx.signature = signature.clone();
        
        Ok(signature)
    }

    /// Submit meta transaction to relayer
    pub async fn submit_meta_tx(
        &self,
        tx: MetaTransaction,
    ) -> Result<String, GaslessError> {
        let relayer = self.relayer.read().await;
        relayer.submit_transaction(tx).await
    }

    /// Get nonce
    pub async fn get_nonce(&self, user: &str) -> u64 {
        let relayer = self.relayer.read().await;
        relayer.get_nonce(user).await
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

    /// Sponsor transaction
    pub async fn sponsor_transaction(
        &self,
        tx: &MetaTransaction,
    ) -> Result<SponsorshipResult, GaslessError> {
        let sponsor = self.sponsor.read().await;
        sponsor.sponsor(tx).await
    }

    /// Set gas policy
    pub async fn set_gas_policy(&self, gas_price: u64, max_gas: u64) -> Result<(), GaslessError> {
        let sponsor = self.sponsor.write().await;
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
async fn sign_message(message: &[u8], _signer: &str) -> Result<Vec<u8>, GaslessError> {
    // In production, would use actual cryptographic signing
    // For now, return the message hashed
    use ring::digest::digest;
    
    let hash = digest(&ring::digest::SHA256, message);
    Ok(hash.as_ref().to_vec())
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
}

use thiserror;