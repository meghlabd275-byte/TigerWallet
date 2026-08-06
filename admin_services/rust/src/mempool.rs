//! Mempool monitor

use crate::transaction::{Transaction, TxHash, Address};
use parking_lot::RwLock;
use std::collections::HashMap;

/// Mempool monitor for tracking pending transactions
pub struct MempoolMonitor {
    transactions: HashMap<TxHash, Transaction>,
    by_address: HashMap<Address, Vec<TxHash>>,
}

impl MempoolMonitor {
    pub fn new() -> Self {
        Self {
            transactions: HashMap::new(),
            by_address: HashMap::new(),
        }
    }
    
    /// Add a transaction to the mempool
    pub fn add_transaction(&mut self, tx: Transaction) {
        let hash = tx.hash.clone();
        
        // Add to main map
        self.transactions.insert(hash.clone(), tx.clone());
        
        // Index by from address
        self.by_address
            .entry(tx.from.clone())
            .or_insert_with(Vec::new)
            .push(hash.clone());
        
        // Index by to address
        self.by_address
            .entry(tx.to.clone())
            .or_insert_with(Vec::new)
            .push(hash);
    }
    
    /// Remove a transaction from the mempool
    pub fn remove_transaction(&mut self, hash: &TxHash) -> Option<Transaction> {
        if let Some(tx) = self.transactions.remove(hash) {
            // Remove from address indexes
            if let Some(hashes) = self.by_address.get_mut(&tx.from) {
                hashes.retain(|h| h != hash);
            }
            if let Some(hashes) = self.by_address.get_mut(&tx.to) {
                hashes.retain(|h| h != hash);
            }
            Some(tx)
        } else {
            None
        }
    }
    
    /// Get high value transactions
    pub fn get_high_value(&self, threshold: u64) -> Vec<Transaction> {
        self.transactions
            .values()
            .filter(|tx| tx.amount >= threshold)
            .cloned()
            .collect()
    }
    
    /// Get transactions by address (either sender or receiver)
    pub fn get_by_address(&self, addr: &Address) -> Vec<Transaction> {
        self.by_address
            .get(addr)
            .map(|hashes| {
                hashes.iter()
                    .filter_map(|h| self.transactions.get(h).cloned())
                    .collect()
            })
            .unwrap_or_default()
    }
    
    /// Get mempool size
    pub fn size(&self) -> usize {
        self.transactions.len()
    }
    
    /// Check if transaction is in mempool
    pub fn contains(&self, hash: &TxHash) -> bool {
        self.transactions.contains_key(hash)
    }
    
    /// Get all transactions
    pub fn all(&self) -> Vec<Transaction> {
        self.transactions.values().cloned().collect()
    }
    
    /// Clear the mempool
    pub fn clear(&mut self) {
        self.transactions.clear();
        self.by_address.clear();
    }
}

impl Default for MempoolMonitor {
    fn default() -> Self {
        Self::new()
    }
}
