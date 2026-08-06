//! Transaction processor

use crate::pool::TransactionPool;
use crate::signature::{SignatureVerifier, EvmSignatureVerifier};
use crate::transaction::{Transaction, TransactionStatus, Chain, Timestamp};
use parking_lot::RwLock;
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::thread;
use std::time::Duration;

/// Processor configuration
#[derive(Debug, Clone)]
pub struct ProcessorConfig {
    pub num_workers: u32,
    pub queue_size: usize,
    pub max_gas_price: u64,
    pub min_gas_price: u64,
    pub block_time_ms: u64,
    pub enable_verification: bool,
    pub enable_deduplication: bool,
    pub supported_chains: Vec<Chain>,
}

impl Default for ProcessorConfig {
    fn default() -> Self {
        Self {
            num_workers: 4,
            queue_size: 100000,
            max_gas_price: 1000,
            min_gas_price: 1,
            block_time_ms: 1000,
            enable_verification: true,
            enable_deduplication: true,
            supported_chains: vec![Chain::Ethereum, Chain::Polygon, Chain::Bsc],
        }
    }
}

/// Processing result
#[derive(Debug)]
pub struct ProcessingResult {
    pub success: bool,
    pub status: TransactionStatus,
    pub error: Option<String>,
    pub processed_at: Timestamp,
    pub gas_used: u64,
}

/// Processor statistics
#[derive(Debug, Default)]
pub struct ProcessorStats {
    pub total_processed: u64,
    pub total_failed: u64,
    pub total_gas_used: u64,
    pub avg_latency_ns: u64,
    pub max_latency_ns: u64,
    pub min_latency_ns: u64,
}

/// Transaction processor
pub struct TransactionProcessor {
    config: ProcessorConfig,
    pool: Arc<RwLock<TransactionPool>>,
    verifier: Arc<EvmSignatureVerifier>,
    running: AtomicBool,
    stats: ProcessorStats,
    stats_lock: RwLock<ProcessorStats>,
}

impl TransactionProcessor {
    pub fn new(config: ProcessorConfig, pool: Arc<RwLock<TransactionPool>>) -> Self {
        Self {
            config,
            pool,
            verifier: Arc::new(EvmSignatureVerifier::new()),
            running: AtomicBool::new(false),
            stats: ProcessorStats::default(),
            stats_lock: RwLock::new(ProcessorStats::default()),
        }
    }
    
    /// Start the processor
    pub fn start(&self) {
        self.running.store(true, Ordering::SeqCst);
        
        for _ in 0..self.config.num_workers {
            let pool = self.pool.clone();
            let running = &self.running;
            let config = self.config.clone();
            let stats_lock = &self.stats_lock;
            
            thread::spawn(move || {
                while running.load(Ordering::SeqCst) {
                    let tx = {
                        let pool = pool.read();
                        pool.next()
                    };
                    
                    if let Some(mut tx) = tx {
                        let start = Timestamp::now();
                        
                        // Process transaction
                        tx.status = TransactionStatus::Confirmed;
                        tx.processed_at = Timestamp::now();
                        
                        let end = Timestamp::now();
                        let latency = end.nanoseconds - start.nanoseconds;
                        
                        // Update stats
                        let mut stats = stats_lock.write();
                        stats.total_processed += 1;
                        stats.total_gas_used += tx.gas_used;
                        
                        if stats.total_processed > 0 {
                            stats.avg_latency_ns = (stats.avg_latency_ns * (stats.total_processed - 1) + latency) 
                                / stats.total_processed;
                        }
                        
                        if latency > stats.max_latency_ns {
                            stats.max_latency_ns = latency;
                        }
                        
                        if latency < stats.min_latency_ns || stats.min_latency_ns == 0 {
                            stats.min_latency_ns = latency;
                        }
                        
                        // Remove from pool
                        let mut pool = pool.write();
                        let _ = pool.remove(&tx.hash);
                    } else {
                        thread::sleep(Duration::from_micros(100));
                    }
                }
            });
        }
    }
    
    /// Stop the processor
    pub fn stop(&self) {
        self.running.store(false, Ordering::SeqCst);
    }
    
    /// Submit a transaction
    pub fn submit(&self, mut tx: Transaction) -> ProcessingResult {
        // Validate transaction
        if tx.from.is_zero() {
            return ProcessingResult {
                success: false,
                status: TransactionStatus::Failed,
                error: Some("Invalid from address".to_string()),
                processed_at: Timestamp::now(),
                gas_used: 0,
            };
        }
        
        if tx.gas_price < self.config.min_gas_price {
            return ProcessingResult {
                success: false,
                status: TransactionStatus::Failed,
                error: Some("Gas price too low".to_string()),
                processed_at: Timestamp::now(),
                gas_used: 0,
            };
        }
        
        if tx.gas_price > self.config.max_gas_price {
            return ProcessingResult {
                success: false,
                status: TransactionStatus::Failed,
                error: Some("Gas price too high".to_string()),
                processed_at: Timestamp::now(),
                gas_used: 0,
            };
        }
        
        // Add to pool
        {
            let mut pool = self.pool.write();
            if let Err(e) = pool.add(tx.clone()) {
                return ProcessingResult {
                    success: false,
                    status: TransactionStatus::Failed,
                    error: Some(format!("{:?}", e)),
                    processed_at: Timestamp::now(),
                    gas_used: 0,
                };
            }
        }
        
        // Process immediately
        tx.status = TransactionStatus::Confirmed;
        tx.processed_at = Timestamp::now();
        
        ProcessingResult {
            success: true,
            status: tx.status,
            error: None,
            processed_at: tx.processed_at,
            gas_used: tx.gas_used,
        }
    }
    
    /// Submit batch of transactions
    pub fn submit_batch(&self, txs: Vec<Transaction>) -> Vec<ProcessingResult> {
        txs.into_iter().map(|tx| self.submit(tx)).collect()
    }
    
    /// Get processor statistics
    pub fn stats(&self) -> ProcessorStats {
        self.stats_lock.read().clone()
    }
    
    /// Check if processor is healthy
    pub fn is_healthy(&self) -> bool {
        self.running.load(Ordering::SeqCst)
    }
}
