//! Relayer service for gasless transactions

use crate::{MetaTransaction, GaslessError, RelayerStatus};
use std::collections::HashMap;
use tokio::sync::RwLock;

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

/// Relayer service
pub struct RelayerService {
    /// Relayers
    relayers: HashMap<String, RelayerInfo>,
    /// Pending transactions
    pending: HashMap<String, PendingTransaction>,
    /// Nonces
    nonces: HashMap<String, u64>,
    /// History
    history: Vec<String>,
}

impl RelayerService {
    pub fn new() -> Self {
        Self {
            relayers: HashMap::new(),
            pending: HashMap::new(),
            nonces: HashMap::new(),
            history: Vec::new(),
        }
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

    /// Submit transaction
    pub async fn submit_transaction(&self, tx: MetaTransaction) -> Result<String, GaslessError> {
        let tx_id = uuid::Uuid::new_v4().to_string();
        Ok(tx_id)
    }

    /// Get nonce
    pub async fn get_nonce(&self, user: &str) -> u64 {
        *self.nonces.get(user).unwrap_or(&0)
    }

    /// Get status
    pub async fn get_status(&self) -> RelayerStatus {
        RelayerStatus {
            active_relayers: self.relayers.values().filter(|r| r.active).count() as u32,
            pending_transactions: self.pending.len() as u32,
            executed_transactions: self.history.len() as u32,
            failed_transactions: 0,
        }
    }
}

impl Default for RelayerService {
    fn default() -> Self {
        Self::new()
    }
}