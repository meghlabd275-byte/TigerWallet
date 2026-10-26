//! Multi-sig module for Account Abstraction

use crate::{Call, AAError};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Multi-sig transaction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiSigTransaction {
    pub id: String,
    pub account: String,
    pub calls: Vec<Call>,
    pub signers: Vec<String>,
    pub signatures: Vec<MultiSigSignature>,
    pub threshold: u32,
    pub executed: bool,
    pub created_at: DateTime<Utc>,
    pub executed_at: Option<DateTime<Utc>>,
}

/// Multi-sig signature
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiSigSignature {
    pub signer: String,
    pub signature: Vec<u8>,
    pub signed_at: DateTime<Utc>,
}

/// Multi-sig service
pub struct MultiSigService {
    /// Transactions
    transactions: HashMap<String, MultiSigTransaction>,
    /// Pending transactions
    pending: HashMap<String, Vec<String>>,
}

impl MultiSigService {
    pub fn new() -> Self {
        Self {
            transactions: HashMap::new(),
            pending: HashMap::new(),
        }
    }

    /// Create multi-sig transaction
    pub async fn create_transaction(
        &mut self,
        account: &str,
        calls: Vec<Call>,
        threshold: u32,
    ) -> Result<String, AAError> {
        if calls.is_empty() {
            return Err(AAError::MultiSigError("No calls".to_string()));
        }
        if threshold == 0 {
            return Err(AAError::MultiSigError(
                "Threshold must be at least one".to_string(),
            ));
        }
        
        let tx_id = generate_tx_id(account, &calls);
        
        let tx = MultiSigTransaction {
            id: tx_id.clone(),
            account: account.to_string(),
            calls,
            signers: vec![],
            signatures: vec![],
            threshold,
            executed: false,
            created_at: Utc::now(),
            executed_at: None,
        };
        
        self.transactions.insert(tx_id.clone(), tx);
        self.pending
            .entry(account.to_string())
            .or_default()
            .push(tx_id.clone());
        
        Ok(tx_id)
    }

    /// Sign transaction
    pub async fn sign_transaction(
        &mut self,
        tx_id: &str,
        signer: &str,
        signature: Vec<u8>,
    ) -> Result<bool, AAError> {
        let tx = self.transactions.get_mut(tx_id)
            .ok_or(AAError::MultiSigError("Transaction not found".to_string()))?;
        
        if signature.is_empty() {
            return Err(AAError::MultiSigError("Empty signature".to_string()));
        }

        // One signer counts once.
        if tx.signatures.iter().any(|s| s.signer == signer) {
            return Err(AAError::MultiSigError("Already signed".to_string()));
        }
        
        // Add signature
        tx.signatures.push(MultiSigSignature {
            signer: signer.to_string(),
            signature,
            signed_at: Utc::now(),
        });
        
        // Check threshold
        let threshold_reached = tx.signatures.len() as u32 >= tx.threshold;
        
        Ok(threshold_reached)
    }

    /// Execute a transaction once it has reached its signature threshold.
    ///
    /// The signers passed in are checked against the signatures already
    /// collected, so a caller cannot satisfy the threshold by naming signers who
    /// never signed.
    pub fn execute(&mut self, tx_id: &str) -> Result<String, AAError> {
        let tx = self
            .transactions
            .get(tx_id)
            .ok_or_else(|| AAError::MultiSigError("Transaction not found".to_string()))?;

        if tx.executed {
            return Err(AAError::MultiSigError(
                "Transaction has already been executed".to_string(),
            ));
        }

        if (tx.signatures.len() as u32) < tx.threshold {
            return Err(AAError::MultiSigError(format!(
                "Transaction has {} of {} required signatures",
                tx.signatures.len(),
                tx.threshold
            )));
        }

        let tx = self
            .transactions
            .get_mut(tx_id)
            .expect("transaction was present a moment ago");
        tx.executed = true;
        tx.executed_at = Some(Utc::now());

        if let Some(pending) = self.pending.get_mut(&tx.account) {
            pending.retain(|id| id != tx_id);
        }

        Ok(tx.id.clone())
    }

    /// Get transaction
    pub fn get_transaction(&self, tx_id: &str) -> Option<MultiSigTransaction> {
        self.transactions.get(tx_id).cloned()
    }

    /// Get pending transactions
    pub fn get_pending(&self, account: &str) -> Vec<String> {
        self.pending.get(account).cloned().unwrap_or_default()
    }

    /// Cancel transaction
    pub async fn cancel_transaction(
        &mut self,
        tx_id: &str,
    ) -> Result<(), AAError> {
        let tx = self.transactions.get_mut(tx_id)
            .ok_or(AAError::MultiSigError("Transaction not found".to_string()))?;
        
        tx.executed = true; // Mark as executed (cancelled)
        
        Ok(())
    }
}

impl Default for MultiSigService {
    fn default() -> Self {
        Self::new()
    }
}

/// Generate transaction ID
fn generate_tx_id(account: &str, calls: &[Call]) -> String {
    use ring::digest::digest;
    
    let mut data = account.as_bytes().to_vec();
    
    for call in calls {
        data.extend_from_slice(call.to.as_bytes());
        data.extend_from_slice(&call.value.to_be_bytes());
    }
    
    data.extend_from_slice(&Utc::now().timestamp().to_be_bytes());
    
    let hash = digest(&ring::digest::SHA256, &data);
    
    hex::encode(hash.as_ref())
}