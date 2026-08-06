//! TigerWallet High-Speed Transaction Processing Library
//! 
//! This module provides high-performance transaction processing capabilities
//! with ultra-low latency for the TigerWallet platform.

pub mod transaction;
pub mod pool;
pub mod signature;
pub mod processor;
pub mod block;
pub mod mempool;

pub use transaction::{Transaction, TransactionType, TransactionStatus, Chain, Address, TxHash};
pub use pool::TransactionPool;
pub use signature::{SignatureVerifier, SignatureResult};
pub use processor::{TransactionProcessor, ProcessorConfig, ProcessingResult, ProcessorStats};
pub use block::BlockBuilder;
pub use mempool::MempoolMonitor;

use std::sync::Arc;
use parking_lot::RwLock;

/// High-level API for TigerWallet transaction processing
pub struct TigerWalletEngine {
    pool: Arc<RwLock<TransactionPool>>,
    processor: Arc<TransactionProcessor>,
    mempool: Arc<RwLock<MempoolMonitor>>,
}

impl TigerWalletEngine {
    /// Create a new engine with the given configuration
    pub fn new(config: ProcessorConfig) -> Self {
        let pool = Arc::new(RwLock::new(TransactionPool::new(100000)));
        let processor = Arc::new(TransactionProcessor::new(config, pool.clone()));
        let mempool = Arc::new(RwLock::new(MempoolMonitor::new()));
        
        Self { pool, processor, mempool }
    }
    
    /// Start the processing engine
    pub fn start(&self) {
        self.processor.start();
    }
    
    /// Stop the processing engine
    pub fn stop(&self) {
        self.processor.stop();
    }
    
    /// Submit a transaction for processing
    pub fn submit_transaction(&self, tx: Transaction) -> ProcessingResult {
        self.processor.submit(tx)
    }
    
    /// Submit multiple transactions in batch
    pub fn submit_batch(&self, txs: Vec<Transaction>) -> Vec<ProcessingResult> {
        self.processor.submit_batch(txs)
    }
    
    /// Get current statistics
    pub fn stats(&self) -> ProcessorStats {
        self.processor.stats()
    }
    
    /// Check if the engine is healthy
    pub fn is_healthy(&self) -> bool {
        self.processor.is_healthy()
    }
    
    /// Get pending transaction count
    pub fn pending_count(&self) -> usize {
        self.pool.read().size()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_transaction_creation() {
        let tx = Transaction::new(
            Address::from_hex("0x1234567890123456789012345678901234567890"),
            Address::from_hex("0x0987654321098765432109876543210987654321"),
            1000,
            Chain::Ethereum,
        );
        
        assert_eq!(tx.amount, 1000);
        assert_eq!(tx.chain, Chain::Ethereum);
        assert_eq!(tx.status, TransactionStatus::Pending);
    }
    
    #[test]
    fn test_pool_operations() {
        let pool = TransactionPool::new(1000);
        
        let tx = Transaction::new(
            Address::from_hex("0x1234567890123456789012345678901234567890"),
            Address::from_hex("0x0987654321098765432109876543210987654321"),
            1000,
            Chain::Ethereum,
        );
        
        assert!(pool.add(tx.clone()).is_ok());
        assert_eq!(pool.size(), 1);
        
        assert!(pool.remove(&tx.hash).is_ok());
        assert_eq!(pool.size(), 0);
    }
}
