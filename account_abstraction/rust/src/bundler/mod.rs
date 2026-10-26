//! Bundler service for ERC-4337 User Operations.
//!
//! The bundler builds the `handleOps` calldata for the canonical EntryPoint and
//! submits it to a node over `eth_sendRawTransaction`. It is fail-closed: with
//! no RPC endpoint or no signing key configured, submission returns an error
//! rather than reporting an operation as included.

use crate::{AAError, UserOperation};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::VecDeque;
use std::sync::Arc;
use tokio::sync::Mutex;

pub mod abi;
mod rpc;

pub use rpc::JsonRpcClient;

/// Bundler status.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BundlerStatus {
    pub pending_ops: u64,
    pub included_ops: u64,
    pub failed_ops: u64,
    pub avg_gas_price: u64,
}

/// Pending user operation.
#[derive(Debug, Clone)]
pub struct PendingUserOp {
    pub user_op: UserOperation,
    pub submitted_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    pub retries: u32,
    pub status: OpStatus,
}

/// Operation status.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum OpStatus {
    Pending,
    Bundled,
    Included,
    Failed,
    Expired,
}

/// Operation history.
#[derive(Debug, Clone)]
pub struct OpHistory {
    pub op_hash: String,
    pub status: OpStatus,
    pub included_at: Option<DateTime<Utc>>,
    pub gas_price: u64,
    pub block_number: u64,
}

/// A signed transaction ready to broadcast.
#[derive(Debug, Clone)]
pub struct BundleTransaction {
    /// Destination: the EntryPoint.
    pub to: String,
    /// `handleOps` calldata.
    pub data: Vec<u8>,
    /// Gas limit for the transaction.
    pub gas_limit: u64,
    /// Max fee per gas, in wei.
    pub max_fee_per_gas: u128,
    /// Max priority fee per gas, in wei.
    pub max_priority_fee_per_gas: u128,
    /// Value sent with the transaction, in wei.
    pub value: u128,
}

/// Bundler service.
pub struct BundlerService {
    entry_point: Option<String>,
    bundler_address: Option<String>,
    pending_ops: VecDeque<PendingUserOp>,
    history: Vec<OpHistory>,
    max_gas_price: u64,
    /// Maximum attempts before an operation is abandoned.
    max_retries: u32,
    /// How long an operation may sit before it expires.
    op_ttl_seconds: i64,
    /// Chain id used when signing transactions.
    chain_id: u64,
    /// JSON-RPC client. Absent until an endpoint is configured.
    rpc: Option<Arc<JsonRpcClient>>,
    /// Signer for the bundle transaction. Absent until a key is configured.
    signer: Option<Arc<crate::tx::TxSigner>>,
}

impl BundlerService {
    pub fn new(chain_id: u64) -> Self {
        Self {
            entry_point: None,
            bundler_address: None,
            pending_ops: VecDeque::new(),
            history: Vec::new(),
            max_gas_price: 100_000_000_000, // 100 gwei
            max_retries: 3,
            op_ttl_seconds: 300, // 5 minutes
            chain_id,
            rpc: None,
            signer: None,
        }
    }

    /// Configure the JSON-RPC endpoint bundles are submitted to.
    pub fn set_rpc_endpoint(&mut self, url: impl Into<String>) {
        self.rpc = Some(Arc::new(JsonRpcClient::new(url)));
    }

    /// Configure the signer used for bundle transactions.
    pub fn set_signer(&mut self, signer: crate::tx::TxSigner) {
        self.bundler_address = Some(signer.address());
        self.signer = Some(Arc::new(signer));
    }

    /// Set the EntryPoint address.
    pub fn set_entry_point(&mut self, entry_point: String) {
        self.entry_point = Some(entry_point);
    }

    /// Set the bundler address.
    pub fn set_bundler_address(&mut self, address: String) {
        self.bundler_address = Some(address);
    }

    /// The configured EntryPoint, if any.
    pub fn entry_point(&self) -> Option<&str> {
        self.entry_point.as_deref()
    }

    /// The configured bundler address, if any.
    pub fn bundler_address(&self) -> Option<&str> {
        self.bundler_address.as_deref()
    }

    /// True when a submission can actually be attempted.
    pub fn can_submit(&self) -> bool {
        self.entry_point.is_some() && self.rpc.is_some() && self.signer.is_some()
    }

    /// Submit a user operation, returning the transaction hash.
    ///
    /// The hash returned is the hash the node accepted. Nothing is recorded as
    /// included until the node responds.
    pub async fn send_user_op(&mut self, user_op: UserOperation) -> Result<String, AAError> {
        let op_hash = self.compute_op_hash(&user_op)?;
        self.validate_operation(&user_op)?;

        let gas_price = self.suggest_gas_price().await?;
        let tx = self.build_bundle_tx(&user_op, gas_price).await?;
        let tx_hash = self.send_to_entry_point(&tx).await?;

        self.history.push(OpHistory {
            op_hash,
            status: OpStatus::Included,
            included_at: Some(Utc::now()),
            gas_price,
            block_number: 0,
        });

        Ok(tx_hash)
    }

    /// Compute the ERC-4337 user operation hash.
    ///
    /// The hash is `keccak256(abi.encode(hash of packed fields, entryPoint, chainId))`,
    /// matching the EntryPoint's `getUserOpHash`.
    pub fn compute_op_hash(&self, user_op: &UserOperation) -> Result<String, AAError> {
        let entry_point = self
            .entry_point
            .as_deref()
            .ok_or(AAError::NotInitialized)?;

        let hash = abi::user_op_hash(user_op, entry_point, self.chain_id)?;
        Ok(hex::encode(hash))
    }

    /// Validate a user operation before building a transaction.
    fn validate_operation(&self, user_op: &UserOperation) -> Result<(), AAError> {
        if user_op.sender.is_empty() {
            return Err(AAError::InvalidSender);
        }
        if user_op.call_gas_limit == 0 {
            return Err(AAError::InsufficientGas);
        }
        if user_op.verification_gas_limit == 0 {
            return Err(AAError::InsufficientVerificationGas);
        }
        if user_op.signature.is_empty() {
            return Err(AAError::InvalidSignature);
        }
        Ok(())
    }

    /// Ask the node for the current gas price.
    async fn suggest_gas_price(&self) -> Result<u64, AAError> {
        let rpc = self.rpc.as_ref().ok_or_else(|| {
            AAError::BundlerError("no JSON-RPC endpoint configured".to_string())
        })?;

        let gas_price = rpc.gas_price().await?;

        if gas_price > self.max_gas_price as u128 {
            return Err(AAError::BundlerError(format!(
                "network gas price {gas_price} exceeds the configured maximum {}",
                self.max_gas_price
            )));
        }

        Ok(gas_price as u64)
    }

    /// Build the `handleOps` transaction.
    async fn build_bundle_tx(
        &self,
        user_op: &UserOperation,
        gas_price: u64,
    ) -> Result<BundleTransaction, AAError> {
        let entry_point = self
            .entry_point
            .clone()
            .ok_or(AAError::NotInitialized)?;

        let data = abi::encode_handle_ops(std::slice::from_ref(user_op))?;

        let gas_limit = user_op
            .call_gas_limit
            .saturating_add(user_op.verification_gas_limit)
            .saturating_add(user_op.pre_verification_gas)
            .saturating_add(100_000);

        Ok(BundleTransaction {
            to: entry_point,
            data,
            gas_limit,
            max_fee_per_gas: gas_price as u128,
            max_priority_fee_per_gas: (gas_price / 10) as u128,
            value: 0,
        })
    }

    /// Sign and broadcast the bundle transaction.
    async fn send_to_entry_point(&self, tx: &BundleTransaction) -> Result<String, AAError> {
        let rpc = self.rpc.as_ref().ok_or_else(|| {
            AAError::BundlerError("no JSON-RPC endpoint configured".to_string())
        })?;
        let signer = self.signer.as_ref().ok_or_else(|| {
            AAError::BundlerError("no signing key configured".to_string())
        })?;

        let nonce = rpc.transaction_count(&signer.address()).await?;
        let chain_id = rpc.chain_id().await?;

        if chain_id != self.chain_id {
            return Err(AAError::BundlerError(format!(
                "node reports chain id {chain_id}, but the bundler is configured for {}",
                self.chain_id
            )));
        }

        let raw = signer.sign_eip1559(tx, chain_id, nonce)?;

        rpc.send_raw_transaction(&raw).await
    }

    /// The configured retry ceiling.
    pub fn max_retries(&self) -> u32 {
        self.max_retries
    }

    /// The configured operation time to live, in seconds.
    pub fn op_ttl_seconds(&self) -> i64 {
        self.op_ttl_seconds
    }

    /// Current bundler status.
    pub fn get_status(&self) -> BundlerStatus {
        BundlerStatus {
            pending_ops: self.pending_ops.len() as u64,
            included_ops: self
                .history
                .iter()
                .filter(|h| h.status == OpStatus::Included)
                .count() as u64,
            failed_ops: self
                .history
                .iter()
                .filter(|h| h.status == OpStatus::Failed)
                .count() as u64,
            avg_gas_price: self.calculate_avg_gas_price(),
        }
    }

    fn calculate_avg_gas_price(&self) -> u64 {
        if self.history.is_empty() {
            return self.max_gas_price;
        }

        let total: u128 = self.history.iter().map(|h| h.gas_price as u128).sum();
        (total / self.history.len() as u128) as u64
    }

    /// Remove a pending operation.
    pub fn cancel_operation(&mut self, op_hash: &str) -> Result<(), AAError> {
        // Pop from the front so the queue is not borrowed while its elements are
        // hashed, which would need a second, immutable borrow of `self`.
        let mut kept = VecDeque::with_capacity(self.pending_ops.len());
        let mut found = false;

        while let Some(op) = self.pending_ops.pop_front() {
            if self.compute_op_hash(&op.user_op)? == op_hash {
                found = true;
            } else {
                kept.push_back(op);
            }
        }
        self.pending_ops = kept;

        if !found {
            return Err(AAError::BundlerError("Operation not found".to_string()));
        }
        Ok(())
    }

    /// Replace a pending operation with one that carries a higher fee.
    pub async fn replace_operation(
        &mut self,
        old_op_hash: &str,
        new_user_op: UserOperation,
    ) -> Result<String, AAError> {
        self.cancel_operation(old_op_hash)?;
        self.send_user_op(new_user_op).await
    }
}

/// Thread-safe handle to a bundler service.
pub type SharedBundler = Arc<Mutex<BundlerService>>;

/// Encode a user operation for the `eth_getUserOperationReceipt` style lookup.
pub fn encode_user_op(user_op: &UserOperation) -> Result<Vec<u8>, AAError> {
    abi::encode_user_op(user_op)
}
