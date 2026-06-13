//! Bundler service for ERC-4337 User Operations

use crate::{UserOperation, AAError};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::VecDeque;

/// Bundler status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BundlerStatus {
    pub pending_ops: u64,
    pub included_ops: u64,
    pub failed_ops: u64,
    pub avg_gas_price: u64,
}

/// Pending user operation
#[derive(Debug, Clone)]
pub struct PendingUserOp {
    pub user_op: UserOperation,
    pub submitted_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    pub retries: u32,
    pub status: OpStatus,
}

/// Operation status
#[derive(Debug, Clone, PartialEq)]
pub enum OpStatus {
    Pending,
    Bundled,
    Included,
    Failed,
    Expired,
}

/// Bundler service
pub struct BundlerService {
    /// Entry point address
    entry_point: Option<String>,
    /// Bundler address
    bundler_address: Option<String>,
    /// Pending operations
    pending_ops: VecDeque<PendingUserOp>,
    /// Operation history
    history: Vec<OpHistory>,
    /// Gas settings
    max_gas_price: u64,
    /// Max retries
    max_retries: u32,
    /// TTL for operations
    op_ttl_seconds: i64,
}

/// Operation history
#[derive(Debug, Clone)]
pub struct OpHistory {
    pub op_hash: String,
    pub status: OpStatus,
    pub included_at: Option<DateTime<Utc>>,
    pub gas_price: u64,
    pub block_number: u64,
}

impl BundlerService {
    pub fn new() -> Self {
        Self {
            entry_point: None,
            bundler_address: None,
            pending_ops: VecDeque::new(),
            history: Vec::new(),
            max_gas_price: 100_000_000_000, // 100 gwei
            max_retries: 3,
            op_ttl_seconds: 300, // 5 minutes
        }
    }

    /// Set entry point
    pub fn set_entry_point(&mut self, entry_point: String) {
        self.entry_point = Some(entry_point);
    }

    /// Set bundler address
    pub fn set_bundler_address(&mut self, address: String) {
        self.bundler_address = Some(address);
    }

    /// Send user operation
    pub async fn send_user_op(&self, user_op: UserOperation) -> Result<String, AAError> {
        // Generate operation hash
        let op_hash = self.compute_op_hash(&user_op);
        
        // Validate operation
        self.validate_operation(&user_op)?;
        
        // Estimate gas
        let estimated_gas = self.estimate_gas(&user_op).await?;
        
        // Build bundle transaction
        let tx = self.build_bundle_tx(&user_op, estimated_gas).await?;
        
        // Send to entry point
        self.send_to_entry_point(&tx).await?;
        
        // Record history
        let history = OpHistory {
            op_hash: op_hash.clone(),
            status: OpStatus::Included,
            included_at: Some(Utc::now()),
            gas_price: estimated_gas,
            block_number: 0, // Would be fetched from chain
        };
        
        Ok(op_hash)
    }

    /// Compute operation hash
    fn compute_op_hash(&self, user_op: &UserOperation) -> String {
        let mut data = vec![];
        
        data.extend_from_slice(user_op.sender.as_bytes());
        data.extend_from_slice(&user_op.nonce.to_be_bytes());
        data.extend_from_slice(&user_op.call_gas_limit.to_be_bytes());
        data.extend_from_slice(&user_op.verification_gas_limit.to_be_bytes());
        data.extend_from_slice(&user_op.pre_verification_gas.to_be_bytes());
        
        if let Some(ref init) = user_op.init_code {
            data.extend_from_slice(init.factory.as_bytes());
        }
        
        if let Some(ref call) = user_op.call_data {
            data.extend_from_slice(call);
        }
        
        use ring::digest::digest;
        let hash = digest(&ring::digest::SHA256, &data);
        
        hex::encode(hash.as_ref())
    }

    /// Validate operation
    fn validate_operation(&self, user_op: &UserOperation) -> Result<(), AAError> {
        if user_op.sender.is_empty() {
            return Err(AAError::InvalidSender);
        }
        
        if user_op.call_gas_limit < 21000 {
            return Err(AAError::InsufficientGas);
        }
        
        if user_op.verification_gas_limit < 50000 {
            return Err(AAError::InsufficientVerificationGas);
        }
        
        Ok(())
    }

    /// Estimate gas for operation
    async fn estimate_gas(&self, user_op: &UserOperation) -> Result<u64, AAError> {
        // Base gas for execution
        let base_gas: u64 = 21000;
        
        // Verification gas
        let verification_gas = user_op.verification_gas_limit;
        
        // Call gas
        let call_gas = user_op.call_gas_limit;
        
        // Pre-verification gas
        let pre_verification_gas = user_op.pre_verification_gas;
        
        let total = base_gas + verification_gas + call_gas + pre_verification_gas;
        
        Ok(total)
    }

    /// Build bundle transaction
    async fn build_bundle_tx(
        &self,
        user_op: &UserOperation,
        _estimated_gas: u64,
    ) -> Result<BundleTransaction, AAError> {
        let entry_point = self.entry_point.clone()
            .ok_or(AAError::NotInitialized)?;
        
        Ok(BundleTransaction {
            to: entry_point,
            data: encode_user_op(user_op),
            gas_limit: user_op.verification_gas_limit + user_op.call_gas_limit,
        })
    }

    /// Send to entry point
    async fn send_to_entry_point(
        &self,
        tx: &BundleTransaction,
    ) -> Result<(), AAError> {
        // In production, this would send to blockchain
        // For now, just validate
        Ok(())
    }

    /// Get status
    pub async fn get_status(&self) -> BundlerStatus {
        BundlerStatus {
            pending_ops: self.pending_ops.len() as u64,
            included_ops: self.history.iter()
                .filter(|h| h.status == OpStatus::Included)
                .count() as u64,
            failed_ops: self.history.iter()
                .filter(|h| h.status == OpStatus::Failed)
                .count() as u64,
            avg_gas_price: self.calculate_avg_gas_price(),
        }
    }

    /// Calculate average gas price
    fn calculate_avg_gas_price(&self) -> u64 {
        if self.history.is_empty() {
            return self.max_gas_price;
        }
        
        let total: u64 = self.history.iter()
            .map(|h| h.gas_price)
            .sum();
        
        total / self.history.len() as u64
    }

    /// Cancel pending operation
    pub async fn cancel_operation(&self, op_hash: &str) -> Result<(), AAError> {
        // Find and remove operation
        let mut found = false;
        
        self.pending_ops.retain(|op| {
            let hash = self.compute_op_hash(&op.user_op);
            if hash == op_hash {
                found = true;
                false
            } else {
                true
            }
        });
        
        if found {
            Ok(())
        } else {
            Err(AAError::BundlerError("Operation not found".to_string()))
        }
    }

    /// Replace operation with higher gas
    pub async fn replace_operation(
        &self,
        old_op_hash: &str,
        new_user_op: UserOperation,
    ) -> Result<String, AAError> {
        // Cancel old operation
        self.cancel_operation(old_op_hash).await?;
        
        // Send new operation
        self.send_user_op(new_user_op).await
    }
}

impl Default for BundlerService {
    fn default() -> Self {
        Self::new()
    }
}

/// Bundle transaction
#[derive(Debug, Clone)]
pub struct BundleTransaction {
    pub to: String,
    pub data: Vec<u8>,
    pub gas_limit: u64,
}

/// Encode user operation
fn encode_user_op(user_op: &UserOperation) -> Vec<u8> {
    let mut data = vec![];
    
    // Function selector for handleOps
    data.extend_from_slice(&[0xc4, 0x53, 0x19, 0x5f]); // handleOps selector
    
    // ABI encode
    data.extend_from_slice(&[0x40, 0x00, 0x00, 0x00]); // offset to ops array
    data.extend_from_slice(&[0x01, 0x00, 0x00, 0x00]); // ops length = 1
    
    // Would encode full UserOperation struct
    // Simplified for demonstration
    
    data
}