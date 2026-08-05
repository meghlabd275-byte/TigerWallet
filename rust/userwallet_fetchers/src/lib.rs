//! TigerWallet UserWallet Fetchers - Rust High-Speed Implementation
//! 
//! This module implements fetchers specifically for UserWallet apps
//! - User wallet operations only
//! - No admin functionality
//! - No master wallet operations
//! 
//! Features:
//! - Ultra-low latency with connection pooling
//! - Redis caching integration
//! - Real blockchain RPC integration (EVM, Solana, etc.)
//! - 21 production-ready fetchers

use std::collections::HashMap;
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};

pub mod types;
pub mod fetchers;

pub use types::*;
pub use fetchers::*;

/// UserWallet fetcher manager - only includes user-facing operations
/// All 21 fetchers for complete UserWallet functionality
pub struct UserWalletFetcherManager {
    fetchers: HashMap<String, Arc<dyn Fetcher>>,
}

impl UserWalletFetcherManager {
    pub fn new() -> Self {
        let mut fetchers = HashMap::new();
        
        // Core wallet operations (8 fetchers)
        fetchers.insert("balance".to_string(), Arc::new(BalanceFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("transactions".to_string(), Arc::new(TransactionFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("tokens".to_string(), Arc::new(TokenFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("nfts".to_string(), Arc::new(NftFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("swap".to_string(), Arc::new(SwapFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("staking".to_string(), Arc::new(StakingFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("gas".to_string(), Arc::new(GasFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("price".to_string(), Arc::new(PriceFetcher::new()) as Arc<dyn Fetcher>);
        
        // DeFi operations (13 additional fetchers)
        fetchers.insert("bridge".to_string(), Arc::new(BridgeFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("lending".to_string(), Arc::new(LendingFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("nft_trading".to_string(), Arc::new(NftTradingFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("options".to_string(), Arc::new(OptionsFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("futures".to_string(), Arc::new(FuturesFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("margin".to_string(), Arc::new(MarginFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("p2p".to_string(), Arc::new(P2PFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("copy_trading".to_string(), Arc::new(CopyTradingFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("dao".to_string(), Arc::new(DAOFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("gift_card".to_string(), Arc::new(GiftCardFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("fiat_ramp".to_string(), Arc::new(FiatRampFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("dapp_registry".to_string(), Arc::new(DAppRegistryFetcher::new()) as Arc<dyn Fetcher>);
        fetchers.insert("price_alerts".to_string(), Arc::new(PriceAlertFetcher::new()) as Arc<dyn Fetcher>);
        
        Self { fetchers }
    }
    
    pub fn get_fetcher(&self, name: &str) -> Option<Arc<dyn Fetcher>> {
        self.fetchers.get(name).cloned()
    }
    
    pub fn list_fetchers(&self) -> Vec<String> {
        self.fetchers.keys().cloned().collect()
    }
    
    /// Get total count of all fetchers
    pub fn count(&self) -> usize {
        self.fetchers.len()
    }
}

// Default implementation for UserWalletFetcherManager
impl Default for UserWalletFetcherManager {
    fn default() -> Self {
        Self::new()
    }
}
