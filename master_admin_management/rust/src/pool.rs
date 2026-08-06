//! Transaction pool implementation

use crate::transaction::{Transaction, TxHash, Address};
use parking_lot::RwLock;
use std::collections::HashMap;
use std::sync::Arc;

/// Transaction pool for managing pending transactions
pub struct TransactionPool {
    pending: HashMap<TxHash, Transaction>,
    by_sender: HashMap<Address, Vec<TxHash>>,
    by_recipient: HashMap<Address, Vec<TxHash>>,
    max_size: usize,
}

impl TransactionPool {
    /// Create a new transaction pool
    pub fn new(max_size: usize) -> Self {
        Self {
            pending: HashMap::new(),
            by_sender: HashMap::new(),
            by_recipient: HashMap::new(),
            max_size,
        }
    }
    
    /// Add a transaction to the pool
    pub fn add(&mut self, tx: Transaction) -> Result<(), PoolError> {
        if self.pending.len() >= self.max_size {
            return Err(PoolError::PoolFull);
        }
        
        if self.pending.contains_key(&tx.hash) {
            return Err(PoolError::DuplicateTransaction);
        }
        
        // Add to main pool
        let hash = tx.hash.clone();
        self.pending.insert(hash.clone(), tx.clone());
        
        // Index by sender
        self.by_sender
            .entry(tx.from.clone())
            .or_insert_with(Vec::new)
            .push(hash.clone());
        
        // Index by recipient
        self.by_recipient
            .entry(tx.to.clone())
            .or_insert_with(Vec::new)
            .push(hash);
        
        Ok(())
    }
    
    /// Remove a transaction from the pool
    pub fn remove(&mut self, hash: &TxHash) -> Result<Transaction, PoolError> {
        let tx = self.pending.remove(hash)
            .ok_or(PoolError::NotFound)?;
        
        // Remove from sender index
        if let Some(hashes) = self.by_sender.get_mut(&tx.from) {
            hashes.retain(|h| h != hash);
        }
        
        // Remove from recipient index
        if let Some(hashes) = self.by_recipient.get_mut(&tx.to) {
            hashes.retain(|h| h != hash);
        }
        
        Ok(tx)
    }
    
    /// Get a transaction by hash
    pub fn get(&self, hash: &TxHash) -> Option<Transaction> {
        self.pending.get(hash).cloned()
    }
    
    /// Get next transaction for processing (highest gas price)
    pub fn next(&self) -> Option<Transaction> {
        self.pending
            .values()
            .max_by_key(|tx| tx.gas_price)
            .cloned()
    }
    
    /// Get transactions by sender
    pub fn get_by_sender(&self, addr: &Address) -> Vec<Transaction> {
        self.by_sender
            .get(addr)
            .map(|hashes| {
                hashes.iter()
                    .filter_map(|h| self.pending.get(h).cloned())
                    .collect()
            })
            .unwrap_or_default()
    }
    
    /// Get transactions by recipient
    pub fn get_by_recipient(&self, addr: &Address) -> Vec<Transaction> {
        self.by_recipient
            .get(addr)
            .map(|hashes| {
                hashes.iter()
                    .filter_map(|h| self.pending.get(h).cloned())
                    .collect()
            })
            .unwrap_or_default()
    }
    
    /// Get pool size
    pub fn size(&self) -> usize {
        self.pending.len()
    }
    
    /// Get pool capacity
    pub fn capacity(&self) -> usize {
        self.max_size
    }
    
    /// Clear all transactions
    pub fn clear(&mut self) {
        self.pending.clear();
        self.by_sender.clear();
        self.by_recipient.clear();
    }
    
    /// Check if pool contains transaction
    pub fn contains(&self, hash: &TxHash) -> bool {
        self.pending.contains_key(hash)
    }
}

/// Pool error types
#[derive(Debug)]
pub enum PoolError {
    PoolFull,
    DuplicateTransaction,
    NotFound,
}
