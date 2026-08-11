//! TigerWallet UserWallet Fetchers — Rust high-speed, ultra-low-latency client.
//!
//! This crate is the Rust UserWallet data layer. It delegates every request
//! to the canonical TigerWallet Go wallet-api backend (go/wallet_api, port
//! 8443): REAL on-chain RPC (eth_getBalance / eth_call / Etherscan), REAL
//! BIP-39/32/44 HD derivation, REAL secp256k1 signing + broadcast, AES-256-GCM
//! encrypted-seed persistence (PostgreSQL + Redis).
//!
//! Design:
//! - A single pooled async `reqwest::Client` (connection reuse, low latency).
//! - Typed response models (see `types`).
//! - Endpoints the canonical backend exposes are forwarded with the JWT.
//! - Endpoints the backend does NOT expose return `Err` (fail-closed). We
//!   never fabricate balances, prices, transactions, or order books.
//!
//! Fail-closed is the secure default: callers get a real error they can act
//! on, not zeroed/empty data that silently looks like "no activity".

pub mod types;
pub mod fetchers;

pub use fetchers::*;
pub use types::*;

use std::collections::HashMap;
use std::sync::Arc;

/// Sync fetcher trait used by the manager registry. Async fetchers are
/// available directly on `UserWalletClient` (the recommended entry point).
pub trait Fetcher: Send + Sync {
    fn name(&self) -> &str;
    fn fetch(&self, params: HashMap<String, String>) -> Result<serde_json::Value, String>;
    fn initialize(&self) -> Result<(), String> {
        Ok(())
    }
}

/// Registry of all UserWallet fetchers. Each forwards to the canonical
/// wallet-api backend via a shared `UserWalletClient`.
pub struct UserWalletFetcherManager {
    client: Arc<UserWalletClient>,
    fetchers: HashMap<String, Arc<dyn Fetcher>>,
}

impl UserWalletFetcherManager {
    pub fn new(base_url: impl Into<String>, token: Option<String>) -> Self {
        let client = Arc::new(UserWalletClient::new(base_url, token));
        let mut fetchers: HashMap<String, Arc<dyn Fetcher>> = HashMap::new();

        fetchers.insert("balance".into(), Arc::new(BalanceFetcher::new(client.clone())) as Arc<dyn Fetcher>);
        fetchers.insert("transactions".into(), Arc::new(TransactionFetcher::new(client.clone())));
        fetchers.insert("tokens".into(), Arc::new(TokenFetcher::new(client.clone())));
        fetchers.insert("nfts".into(), Arc::new(NftFetcher::new(client.clone())));
        fetchers.insert("gas".into(), Arc::new(GasFetcher::new(client.clone())));
        fetchers.insert("price".into(), Arc::new(PriceFetcher::new(client.clone())));
        fetchers.insert("swap".into(), Arc::new(SwapFetcher::new(client.clone())));
        fetchers.insert("staking".into(), Arc::new(StakingFetcher::new(client.clone())));
        fetchers.insert("dapps".into(), Arc::new(DAppRegistryFetcher::new(client.clone())));

        // DeFi service fetchers — each forwards to a REAL running Go
        // microservice (same service URLs as the Next.js _proxy). See
        // `UserWalletClient::service_get`.
        fetchers.insert("lending".into(), Arc::new(LendingFetcher::new(client.clone())) as Arc<dyn Fetcher>);
        fetchers.insert("copy_trading".into(), Arc::new(CopyTradingFetcher::new(client.clone())));
        fetchers.insert("dao".into(), Arc::new(DaoFetcher::new(client.clone())));
        fetchers.insert("futures".into(), Arc::new(FuturesFetcher::new(client.clone())));
        fetchers.insert("margin".into(), Arc::new(MarginFetcher::new(client.clone())));
        fetchers.insert("prediction".into(), Arc::new(PredictionFetcher::new(client.clone())));
        fetchers.insert("nft_trading".into(), Arc::new(NftTradingFetcher::new(client.clone())));
        fetchers.insert("fiat_ramp".into(), Arc::new(FiatRampFetcher::new(client.clone())));

        // Fetchers for which no standalone HTTP service exists yet. Registered
        // as fail-closed so the manager surface stays complete (21 names)
        // without fabricating data. Wire the corresponding Go service before
        // use.
        for name in [
            "bridge", "options", "p2p", "gift_card", "price_alerts",
        ] {
            fetchers.insert(name.into(), Arc::new(UnavailableFetcher::new(name)) as Arc<dyn Fetcher>);
        }

        Self { client, fetchers }
    }

    pub fn client(&self) -> &UserWalletClient {
        &self.client
    }

    pub fn get_fetcher(&self, name: &str) -> Option<Arc<dyn Fetcher>> {
        self.fetchers.get(name).cloned()
    }

    pub fn list_fetchers(&self) -> Vec<String> {
        self.fetchers.keys().cloned().collect()
    }

    pub fn count(&self) -> usize {
        self.fetchers.len()
    }
}

impl Default for UserWalletFetcherManager {
    fn default() -> Self {
        Self::new(WALLET_API_DEFAULT_URL, None)
    }
}
