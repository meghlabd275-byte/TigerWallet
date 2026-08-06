//! Block builder

use crate::transaction::Transaction;

/// Block builder for creating new blocks
pub struct BlockBuilder {
    transactions: Vec<Transaction>,
    gas_limit: u64,
    gas_used: u64,
    block_number: u64,
}

impl BlockBuilder {
    pub fn new(block_number: u64, gas_limit: u64) -> Self {
        Self {
            transactions: Vec::new(),
            gas_limit,
            gas_used: 0,
            block_number,
        }
    }
    
    /// Add a transaction to the block
    pub fn add_transaction(&mut self, tx: Transaction) -> Result<(), BlockBuilderError> {
        if self.gas_used + tx.gas_limit > self.gas_limit {
            return Err(BlockBuilderError::GasLimitExceeded);
        }
        
        self.transactions.push(tx);
        self.gas_used += tx.gas_limit;
        
        Ok(())
    }
    
    /// Remove a transaction from the block
    pub fn remove_transaction(&mut self, index: usize) -> Option<Transaction> {
        if index >= self.transactions.len() {
            return None;
        }
        
        let tx = self.transactions.remove(index);
        self.gas_used -= tx.gas_limit;
        
        Some(tx)
    }
    
    /// Build the block (sorts transactions by gas price)
    pub fn build(mut self) -> Vec<Transaction> {
        // Sort by gas price for optimal block composition
        self.transactions.sort_by(|a, b| b.gas_price.cmp(&a.gas_price));
        self.transactions
    }
    
    /// Get gas used
    pub fn gas_used(&self) -> u64 {
        self.gas_used
    }
    
    /// Get transaction count
    pub fn transaction_count(&self) -> usize {
        self.transactions.len()
    }
    
    /// Get block number
    pub fn block_number(&self) -> u64 {
        self.block_number
    }
    
    /// Get remaining gas
    pub fn remaining_gas(&self) -> u64 {
        self.gas_limit - self.gas_used
    }
}

#[derive(Debug)]
pub enum BlockBuilderError {
    GasLimitExceeded,
    InvalidTransaction,
}
