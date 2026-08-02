//! TigerWallet Full Fetchers - Rust High-Speed Implementation
//! 
//! This module implements all 20 fetchers in Rust for high-speed operations:
//! - 6 Standard Fetchers
//! - 14 Advanced Fetchers
//! 
//! Built with Rust for performance and memory safety

use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH, Duration};
use std::thread;
use std::sync::atomic::{AtomicU64, AtomicBool, Ordering};
use std::future::Future;
use std::pin::Pin;

// Re-export types
pub mod types;
pub mod fetchers;

pub use types::*;
pub use fetchers::*;

/// Timestamp in milliseconds
pub type Timestamp = u64;
/// Chain ID
pub type ChainId = u64;
/// Gas price in gwei
pub type GasPrice = u64;
/// Token amount as string
pub type TokenAmount = String;
/// Ethereum address
pub type Address = String;

/// Full fetcher manager - orchestrates all fetchers
pub struct FullFetcherManager {
    fetchers: HashMap<String, Arc<dyn Fetcher>>,
    running: Arc<AtomicBool>,
}

impl FullFetcherManager {
    /// Create a new manager
    pub fn new() -> Self {
        let mut fetchers = HashMap::new();
        
        // Standard fetchers
        fetchers.insert("erc20".to_string(), Arc::new(Erc20TokenFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("gas".to_string(), Arc::new(GasEstimatorFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("price".to_string(), Arc::new(PriceFeedFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("dapp".to_string(), Arc::new(DAppConnectionFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("network".to_string(), Arc::new(NetworkFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("swap".to_string(), Arc::new(SwapQuoteFetcher::new()) as Arc<dyn Fetcher>);
        
        // Advanced fetchers
        fetchers.insert("ai_price".to_string(), Arc::new(AIPricePredictorFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("mev".to_string(), Arc::new(MEVOpportunityFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("liquidity".to_string(), Arc::new(LiquidityFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("arbitrage".to_string(), Arc::new(ArbitrageFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("risk".to_string(), Arc::new(TokenRiskFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("contract".to_string(), Arc::new(SmartContractFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("gas_market".to_string(), Arc::new(GasMarketFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("yield".to_string(), Arc::new(DeFiYieldFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("staking".to_string(), Arc::new(StakingOptimizerFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("nft_floor".to_string(), Arc::new(NFTFloorPriceFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("whale".to_string(), Arc::new(WhaleTransactionFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("analytics".to_string(), Arc::new(OnChainAnalyticsFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("simulator".to_string(), Arc::new(TransactionSimulatorFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("cross_chain".to_string(), Arc::new(CrossChainRouteOptimizer::new()) as Arc<dyn Fetcher>);
        
        Self {
            fetchers,
            running: Arc::new(AtomicBool::new(false)),
        }
    }
    
    /// Initialize all fetchers
    pub fn initialize_all(&mut self) -> Result<(), String> {
        for (name, fetcher) in &self.fetchers {
            fetcher.initialize().map_err(|e| format!("Failed to initialize {}: {}", name, e))?;
        }
        Ok(())
    }
    
    /// Start all fetchers
    pub fn start_all(&self) {
        self.running.store(true, Ordering::SeqCst);
        
        for (name, fetcher) in &self.fetchers {
            let running = self.running.clone();
            let fetcher = fetcher.clone_as_fetcher();
            
            thread::spawn(move || {
                while running.load(Ordering::SeqCst) {
                    if let Err(e) = fetcher.fetch() {
                        eprintln!("Fetcher {} error: {}", name, e);
                    }
                    thread::sleep(Duration::from_secs(1));
                }
            });
        }
    }
    
    /// Stop all fetchers
    pub fn stop_all(&self) {
        self.running.store(false, Ordering::SeqCst);
        
        for fetcher in self.fetchers.values() {
            let _ = fetcher.shutdown();
        }
    }
    
    /// Get a specific fetcher
    pub fn get_fetcher(&self, name: &str) -> Option<Arc<dyn Fetcher>> {
        self.fetchers.get(name).cloned()
    }
    
    /// Get statistics for all fetchers
    pub fn get_stats(&self) -> HashMap<String, FetcherStats> {
        let mut stats = HashMap::new();
        
        for (name, fetcher) in &self.fetchers {
            stats.insert(name.clone(), fetcher.get_stats());
        }
        
        stats
    }
    
    /// Print all statistics
    pub fn print_stats(&self) {
        println!("\n=== Fetcher Statistics (Rust) ===");
        
        for (name, stats) in self.get_stats() {
            println!("{}: latency={}ns, requests={}, success_rate={:.2}%",
                name, stats.last_latency_ns, stats.total_requests, stats.success_rate);
        }
    }
}

impl Default for FullFetcherManager {
    fn default() -> Self {
        Self::new()
    }
}

impl Drop for FullFetcherManager {
    fn drop(&mut self) {
        self.stop_all();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_manager_creation() {
        let manager = FullFetcherManager::new();
        assert_eq!(manager.fetchers.len(), 20);
    }
    
    #[test]
    fn test_initialization() {
        let mut manager = FullFetcherManager::new();
        assert!(manager.initialize_all().is_ok());
    }
}
