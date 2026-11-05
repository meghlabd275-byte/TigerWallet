//! Relayer service for gasless transactions

use crate::{MetaTransaction, GaslessError, RelayerStatus};
use chrono::{DateTime, Utc};
use std::collections::HashMap;

/// Relayer information
#[derive(Debug, Clone)]
pub struct RelayerInfo {
    pub address: String,
    pub active: bool,
    pub fee: u64,
    pub max_gas: u64,
    pub processed: u64,
    pub failed: u64,
}

/// Pending transaction
#[derive(Debug, Clone)]
pub struct PendingTransaction {
    pub tx: MetaTransaction,
    pub submitted_at: u64,
    pub relayer: String,
}

/// A transaction the relayer has mined, successfully or not.
#[derive(Debug, Clone)]
pub struct RelayRecord {
    pub tx_id: String,
    pub tx: MetaTransaction,
    pub success: bool,
    pub at: DateTime<Utc>,
}

/// Relayer service
pub struct RelayerService {
    /// Relayers
    relayers: HashMap<String, RelayerInfo>,
    /// Pending transactions, keyed by id
    pending: HashMap<String, MetaTransaction>,
    /// Nonces, keyed by sender
    nonces: HashMap<String, u64>,
    /// Mined transactions
    history: Vec<RelayRecord>,
    /// Transactions that were mined and reverted
    failed: u32,
}

impl RelayerService {
    pub fn new() -> Self {
        Self {
            relayers: HashMap::new(),
            pending: HashMap::new(),
            nonces: HashMap::new(),
            history: Vec::new(),
            failed: 0,
        }
    }

    /// Returns a queued transaction, if it is still pending.
    pub fn get_pending(&self, tx_id: &str) -> Option<&MetaTransaction> {
        self.pending.get(tx_id)
    }

    /// Register relayer
    pub async fn register(&mut self, address: &str) -> Result<(), GaslessError> {
        self.relayers.insert(address.to_string(), RelayerInfo {
            address: address.to_string(),
            active: true,
            fee: 0,
            max_gas: 500000,
            processed: 0,
            failed: 0,
        });
        Ok(())
    }

    /// Deregister relayer
    pub async fn deregister(&mut self, address: &str) -> Result<(), GaslessError> {
        if let Some(relayer) = self.relayers.get_mut(address) {
            relayer.active = false;
            Ok(())
        } else {
            Err(GaslessError::RelayerError("Not found".to_string()))
        }
    }

    /// Accepts a meta transaction for relay.
    ///
    /// The previous version generated a random id and returned it, recording
    /// nothing: the transaction was neither queued nor counted, and
    /// `get_nonce` was never advanced, so a caller could replay the same
    /// signed transaction indefinitely. The transaction is now validated,
    /// queued under its own id, and the nonce consumed.
    pub async fn submit_transaction(
        &mut self,
        tx: MetaTransaction,
    ) -> Result<String, GaslessError> {
        if tx.from.trim().is_empty() || tx.to.trim().is_empty() {
            return Err(GaslessError::RelayerError(
                "from and to must both be set".to_string(),
            ));
        }
        if tx.signature.is_empty() {
            return Err(GaslessError::InvalidSignature);
        }

        // The nonce must be the one the relayer expects, which is what makes
        // a signed meta transaction single-use.
        let expected = self.get_nonce(&tx.from);
        if tx.nonce != expected {
            return Err(GaslessError::RelayerError(format!(
                "nonce {} does not match the expected {}",
                tx.nonce, expected
            )));
        }

        let tx_id = format!("{}:{}", tx.from, tx.nonce);
        if self.pending.contains_key(&tx_id) {
            return Err(GaslessError::RelayerError(format!(
                "transaction {} is already pending",
                tx_id
            )));
        }

        self.nonces.insert(tx.from.clone(), expected + 1);
        self.pending.insert(tx_id.clone(), tx);

        Ok(tx_id)
    }

    /// Records that a queued transaction was mined.
    pub async fn confirm_transaction(
        &mut self,
        tx_id: &str,
        success: bool,
    ) -> Result<(), GaslessError> {
        let tx = self
            .pending
            .remove(tx_id)
            .ok_or_else(|| GaslessError::RelayerError(format!("{} is not pending", tx_id)))?;

        self.history.push(RelayRecord {
            tx_id: tx_id.to_string(),
            tx,
            success,
            at: Utc::now(),
        });
        if !success {
            self.failed += 1;
        }
        Ok(())
    }

    /// Number of transactions that were mined and reverted.
    pub fn failed_count(&self) -> u32 {
        self.failed
    }

    /// Get nonce
    pub fn get_nonce(&self, user: &str) -> u64 {
        *self.nonces.get(user).unwrap_or(&0)
    }

    /// Get status
    pub async fn get_status(&self) -> RelayerStatus {
        RelayerStatus {
            active_relayers: self.relayers.values().filter(|r| r.active).count() as u32,
            pending_transactions: self.pending.len() as u32,
            // executed counts the ones that were mined successfully, which is
            // the total less the reverts. It was previously the whole history
            // and the failure count was hardcoded to zero.
            executed_transactions: self
                .history
                .iter()
                .filter(|record| record.success)
                .count() as u32,
            failed_transactions: self.failed,
        }
    }
}

impl Default for RelayerService {
    fn default() -> Self {
        Self::new()
    }
}