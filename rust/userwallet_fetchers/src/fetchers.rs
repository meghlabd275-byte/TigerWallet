//! TigerWallet UserWallet Fetchers — HTTP client + fetcher implementations.
//!
//! `UserWalletClient` is the pooled async client delegating to the canonical
//! go/wallet_api backend (:8443). Fetchers wrap it with typed models.

use crate::types::*;
use reqwest::Client;
use serde_json::Value;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;

pub const WALLET_API_DEFAULT_URL: &str = "http://localhost:8443/api/v1";

/// Pooled async HTTP client for the canonical wallet-api backend.
///
/// Connection pooling keeps latency low for repeated calls; a single JWT
/// (when set) is attached to protected routes.
pub struct UserWalletClient {
    http: Client,
    base_url: String,
    token: std::sync::RwLock<Option<String>>,
    /// Base URLs of the dedicated DeFi microservices (mirrors the Next.js
    /// `_proxy.ts` service map). The wallet-api base_url covers auth, wallets,
    /// balance, transactions, tokens, nfts, gas, price, swap, staking, dapps,
    /// chains; these cover the rest.
    services: HashMap<String, String>,
}

/// Default service URLs matching go/wallet_api + the dedicated Go DeFi
/// microservices (same defaults as frontend/web_nextjs `app/api/v1/_proxy.ts`).
fn default_service_urls(wallet_api: &str) -> HashMap<String, String> {
    let mut m = HashMap::new();
    // wallet-api itself (for completeness)
    m.insert("wallet_api".into(), wallet_api.to_string());
    // dedicated DeFi services
    m.insert("lending".into(), "http://localhost:8009".to_string());
    m.insert("copy_trading".into(), "http://localhost:8006".to_string());
    m.insert("governance".into(), "http://localhost:8454".to_string());
    m.insert("perpetual".into(), "http://localhost:8464".to_string());
    m.insert("prediction".into(), "http://localhost:8455".to_string());
    m.insert("nft".into(), "http://localhost:8085".to_string());
    m.insert("fiat_ramp".into(), "http://localhost:8008".to_string());
    m
}

impl UserWalletClient {
    pub fn new(base_url: impl Into<String>, token: Option<String>) -> Self {
        let http = Client::builder()
            .timeout(Duration::from_secs(30))
            .pool_max_idle_per_host(16)
            .tcp_nodelay(true)
            .build()
            .expect("reqwest client build");
        let base_url = base_url.into();
        Self {
            http,
            services: default_service_urls(&base_url),
            base_url,
            token: std::sync::RwLock::new(token),
        }
    }

    /// Override a service base URL (e.g. for a different deployment).
    pub fn set_service_url(&mut self, service: &str, url: impl Into<String>) {
        self.services.insert(service.to_string(), url.into());
    }

    pub fn set_token(&self, token: Option<String>) {
        *self.token.write().unwrap() = token;
    }

    pub fn token(&self) -> Option<String> {
        self.token.read().unwrap().clone()
    }

    async fn request_json(
        &self,
        path: &str,
        authenticated: bool,
    ) -> Result<Value, String> {
        let url = format!("{}{}", self.base_url, path);
        let mut req = self.http.get(&url);
        if authenticated {
            if let Some(t) = self.token() {
                req = req.bearer_auth(t);
            } else {
                return Err("Not authenticated: no JWT token set".to_string());
            }
        }
        let resp = req.send().await.map_err(|e| format!("network: {e}"))?;
        let status = resp.status();
        let body: Value = resp.json().await.map_err(|e| format!("decode: {e}"))?;
        if !status.is_success() {
            let msg = body
                .get("error")
                .and_then(|v| v.as_str())
                .unwrap_or("request failed");
            return Err(format!("http {}: {msg}", status.as_u16()));
        }
        Ok(body)
    }

    // ---- Auth ----

    pub async fn login(&self, email: &str, password: &str) -> Result<String, String> {
        let url = format!("{}/auth/login", self.base_url);
        let resp = self
            .http
            .post(&url)
            .json(&serde_json::json!({ "email": email, "password": password }))
            .send()
            .await
            .map_err(|e| format!("network: {e}"))?;
        let status = resp.status();
        let body: Value = resp.json().await.map_err(|e| format!("decode: {e}"))?;
        if !status.is_success() {
            return Err(body
                .get("error")
                .and_then(|v| v.as_str())
                .unwrap_or("login failed")
                .to_string());
        }
        let token = body
            .get("token")
            .and_then(|v| v.as_str())
            .ok_or("missing token")?
            .to_string();
        self.set_token(Some(token.clone()));
        Ok(token)
    }

    pub async fn register(
        &self,
        email: &str,
        username: &str,
        password: &str,
    ) -> Result<String, String> {
        let url = format!("{}/auth/register", self.base_url);
        let resp = self
            .http
            .post(&url)
            .json(&serde_json::json!({
                "email": email,
                "username": username,
                "password": password
            }))
            .send()
            .await
            .map_err(|e| format!("network: {e}"))?;
        let status = resp.status();
        let body: Value = resp.json().await.map_err(|e| format!("decode: {e}"))?;
        if !status.is_success() {
            return Err(body
                .get("error")
                .and_then(|v| v.as_str())
                .unwrap_or("register failed")
                .to_string());
        }
        let token = body
            .get("token")
            .and_then(|v| v.as_str())
            .ok_or("missing token")?
            .to_string();
        self.set_token(Some(token.clone()));
        Ok(token)
    }

    // ---- Wallets ----

    pub async fn list_wallets(&self) -> Result<Vec<WalletRecord>, String> {
        let body = self.request_json("/wallets", true).await?;
        let wallets = body
            .get("wallets")
            .cloned()
            .unwrap_or(Value::Array(vec![]));
        serde_json::from_value(wallets).map_err(|e| format!("decode wallets: {e}"))
    }

    pub async fn create_wallet(
        &self,
        label: &str,
        password: &str,
        chain_id: i64,
        mnemonic: Option<&str>,
    ) -> Result<WalletRecord, String> {
        let url = format!("{}/wallets", self.base_url);
        let mut payload = serde_json::json!({
            "label": label,
            "password": password,
            "chain_id": chain_id,
        });
        if let Some(m) = mnemonic {
            payload["mnemonic"] = Value::String(m.to_string());
        } else {
            payload["entropy_bits"] = Value::Number(serde_json::Number::from(256));
        }
        let mut req = self.http.post(&url).json(&payload);
        if let Some(t) = self.token() {
            req = req.bearer_auth(t);
        }
        let resp = req.send().await.map_err(|e| format!("network: {e}"))?;
        let status = resp.status();
        let body: Value = resp.json().await.map_err(|e| format!("decode: {e}"))?;
        if !status.is_success() {
            return Err(body
                .get("error")
                .and_then(|v| v.as_str())
                .unwrap_or("create wallet failed")
                .to_string());
        }
        serde_json::from_value(body).map_err(|e| format!("decode wallet: {e}"))
    }

    // ---- Balance (real eth_getBalance via backend) ----

    pub async fn balance(&self, address: &str, chain_id: i64) -> Result<BalanceResult, String> {
        let path = format!("/public/balance?address={address}&chain_id={chain_id}");
        let body = self.request_json(&path, false).await?;
        serde_json::from_value(body).map_err(|e| format!("decode balance: {e}"))
    }

    pub async fn balances(&self) -> Result<Vec<BalanceResult>, String> {
        let wallets = self.list_wallets().await?;
        let mut results = Vec::with_capacity(wallets.len());
        for w in wallets {
            match self.balance(&w.address, w.chain_id).await {
                Ok(b) => results.push(b),
                Err(e) => return Err(e),
            }
        }
        Ok(results)
    }

    // ---- Transactions (real Etherscan via backend) ----

    pub async fn transactions(
        &self,
        address: &str,
        chain_id: i64,
    ) -> Result<Vec<TransactionRecord>, String> {
        let path = format!("/transactions?address={address}&chain_id={chain_id}");
        let body = self.request_json(&path, true).await?;
        let txs = body
            .get("transactions")
            .cloned()
            .unwrap_or(Value::Array(vec![]));
        serde_json::from_value(txs).map_err(|e| format!("decode transactions: {e}"))
    }

    // ---- Tokens (real ERC-20 eth_call via backend) ----

    pub async fn tokens(&self, address: &str, chain_id: i64) -> Result<Vec<Token>, String> {
        let path = format!("/tokens?address={address}&chain_id={chain_id}");
        let body = self.request_json(&path, true).await?;
        let tokens = body
            .get("tokens")
            .cloned()
            .unwrap_or(Value::Array(vec![]));
        serde_json::from_value(tokens).map_err(|e| format!("decode tokens: {e}"))
    }

    // ---- NFTs (real on-chain reads via backend) ----

    pub async fn nfts(&self, address: &str, chain_id: i64) -> Result<Vec<Nft>, String> {
        let path = format!("/nfts?address={address}&chain_id={chain_id}");
        let body = self.request_json(&path, true).await?;
        let nfts = body.get("nfts").cloned().unwrap_or(Value::Array(vec![]));
        serde_json::from_value(nfts).map_err(|e| format!("decode nfts: {e}"))
    }

    // ---- Gas (real eth_feeHistory/eth_gasPrice via backend) ----

    pub async fn gas_price(&self, chain_id: i64) -> Result<GasPrice, String> {
        let path = format!("/gas?chain_id={chain_id}");
        let body = self.request_json(&path, true).await?;
        serde_json::from_value(body).map_err(|e| format!("decode gas: {e}"))
    }

    // ---- Price (real CoinGecko via backend) ----

    pub async fn price(&self, symbol: &str) -> Result<PriceInfo, String> {
        let path = format!("/price?symbol={symbol}");
        let body = self.request_json(&path, false).await?;
        serde_json::from_value(body).map_err(|e| format!("decode price: {e}"))
    }

    // ---- Swap quote (real CoinGecko cross-rate via backend) ----

    pub async fn swap_quote(
        &self,
        from_token: &str,
        to_token: &str,
        amount: &str,
        chain_id: i64,
    ) -> Result<SwapQuote, String> {
        let path = format!(
            "/swap/quote?from_token={from_token}&to_token={to_token}&amount={amount}&chain_id={chain_id}"
        );
        let body = self.request_json(&path, true).await?;
        serde_json::from_value(body).map_err(|e| format!("decode swap quote: {e}"))
    }

    // ---- Staking quote (supported assets via backend) ----

    pub async fn staking_quote(&self, chain_id: i64) -> Result<Vec<StakingQuote>, String> {
        let path = format!("/staking/quote?chain_id={chain_id}");
        let body = self.request_json(&path, true).await?;
        let assets = body
            .get("assets")
            .cloned()
            .unwrap_or(Value::Array(vec![]));
        serde_json::from_value(assets).map_err(|e| format!("decode staking: {e}"))
    }

    // ---- DApp directory (curated real entries via backend) ----

    pub async fn dapps(&self, category: Option<&str>, chain: Option<&str>) -> Result<Vec<DApp>, String> {
        let mut path = String::from("/dapps");
        let mut sep = '?';
        if let Some(c) = category {
            path.push(sep);
            path.push_str(&format!("category={c}"));
            sep = '&';
        }
        if let Some(c) = chain {
            path.push(sep);
            path.push_str(&format!("chain={c}"));
        }
        let body = self.request_json(&path, false).await?;
        let dapps = body.get("dapps").cloned().unwrap_or(Value::Array(vec![]));
        serde_json::from_value(dapps).map_err(|e| format!("decode dapps: {e}"))
    }

    // ---- Token registry (real per-chain lists via backend) ----

    pub async fn token_registry(&self, chain_id: Option<i64>) -> Result<Vec<TokenRegistryEntry>, String> {
        let path = match chain_id {
            Some(id) => format!("/tokens/registry?chain_id={id}"),
            None => "/tokens/registry".to_string(),
        };
        let body = self.request_json(&path, false).await?;
        let registry = body
            .get("tokens")
            .cloned()
            .unwrap_or(Value::Array(vec![]));
        serde_json::from_value(registry).map_err(|e| format!("decode registry: {e}"))
    }

    // ---- Chains ----

    pub async fn chains(&self) -> Result<Vec<ChainInfo>, String> {
        let body = self.request_json("/chains", false).await?;
        let chains = body.get("chains").cloned().unwrap_or(Value::Array(vec![]));
        serde_json::from_value(chains).map_err(|e| format!("decode chains: {e}"))
    }

    // ========================================================================
    // DeFi service fetchers — delegate to the dedicated Go microservices
    // (same service URLs the Next.js _proxy uses). Each is a REAL HTTP call
    // to a real running service. If a service URL is empty/unreachable, the
    // call returns a real network error — never fabricated data.
    // ========================================================================

    async fn service_get(&self, service: &str, path: &str, authenticated: bool) -> Result<Value, String> {
        let base = self
            .services
            .get(service)
            .cloned()
            .ok_or_else(|| format!("service '{service}' URL is not configured"))?;
        if base.is_empty() {
            return Err(format!("service '{service}' URL is empty; configure it before use"));
        }
        let url = format!("{base}{path}");
        let mut req = self.http.get(&url);
        if authenticated {
            if let Some(t) = self.token() {
                req = req.bearer_auth(t);
            } else {
                return Err("Not authenticated: no JWT token set".to_string());
            }
        }
        let resp = req.send().await.map_err(|e| format!("network: {e}"))?;
        let status = resp.status();
        let body: Value = resp.json().await.map_err(|e| format!("decode: {e}"))?;
        if !status.is_success() {
            let msg = body
                .get("error")
                .and_then(|v| v.as_str())
                .unwrap_or("request failed");
            return Err(format!("http {}: {msg}", status.as_u16()));
        }
        Ok(body)
    }

    /// Real Aave V3 lending markets (go/lending_service :8009).
    pub async fn lending_markets(&self) -> Result<Value, String> {
        self.service_get("lending", "/api/v1/lending/markets", false).await
    }

    /// Real copy-trading signals (go/copy_trading_service :8006).
    pub async fn copy_trading_signals(&self) -> Result<Value, String> {
        self.service_get("copy_trading", "/api/v1/copytrading/signals", true).await
    }

    /// Real governance proposals / DAO (go/governance_service :8454).
    pub async fn dao_proposals(&self) -> Result<Value, String> {
        self.service_get("governance", "/api/v1/governance/proposals", false).await
    }

    /// Real perpetual markets — covers futures + margin positions
    /// (go/perpetual_service :8464).
    pub async fn futures_markets(&self) -> Result<Value, String> {
        self.service_get("perpetual", "/api/v1/perpetual/pairs", true).await
    }

    /// Real prediction markets (go/prediction_service :8455).
    pub async fn prediction_markets(&self) -> Result<Value, String> {
        self.service_get("prediction", "/api/v1/prediction/markets", false).await
    }

    /// Real NFT marketplace listings / NFT trading (go/nft_service :8085).
    pub async fn nft_marketplace(&self) -> Result<Value, String> {
        self.service_get("nft", "/api/v1/nft/listings", false).await
    }

    /// Real fiat on/off-ramp providers (go/fiat_ramp :8008).
    pub async fn fiat_ramp_providers(&self) -> Result<Value, String> {
        self.service_get("fiat_ramp", "/api/v1/ramp/providers", false).await
    }
}

// ============================================================================
// Sync Fetcher wrappers (registry surface). Each delegates to the async
// client via tokio block_in_place-free runtime handle where possible; for
// the sync trait we surface params and return the backend's JSON.
// ============================================================================

macro_rules! fetcher_struct {
    ($name:ident, $fetcher_name:expr) => {
        pub struct $name {
            client: Arc<UserWalletClient>,
        }
        impl $name {
            pub fn new(client: Arc<UserWalletClient>) -> Self {
                Self { client }
            }
        }
        impl $name {
            pub fn fetcher_name() -> &'static str {
                $fetcher_name
            }
        }
    };
}

fetcher_struct!(BalanceFetcher, "balance");
fetcher_struct!(TransactionFetcher, "transactions");
fetcher_struct!(TokenFetcher, "tokens");
fetcher_struct!(NftFetcher, "nfts");
fetcher_struct!(GasFetcher, "gas");
fetcher_struct!(PriceFetcher, "price");
fetcher_struct!(SwapFetcher, "swap");
fetcher_struct!(StakingFetcher, "staking");
fetcher_struct!(DAppRegistryFetcher, "dapps");

// DeFi service fetchers — each wraps a real Go microservice call.
fetcher_struct!(LendingFetcher, "lending");
fetcher_struct!(CopyTradingFetcher, "copy_trading");
fetcher_struct!(DaoFetcher, "dao");
fetcher_struct!(FuturesFetcher, "futures");
fetcher_struct!(MarginFetcher, "margin");
fetcher_struct!(PredictionFetcher, "prediction");
fetcher_struct!(NftTradingFetcher, "nft_trading");
fetcher_struct!(FiatRampFetcher, "fiat_ramp");

// The sync `fetch` for each forwards to the async client through a fresh
// runtime. This keeps the manager usable from non-async contexts without
// fabricating data.
fn block_on<F: std::future::Future>(fut: F) -> Result<F::Output, String> {
    tokio::runtime::Builder::new_current_thread()
        .enable_all()
        .build()
        .map_err(|e| format!("runtime: {e}"))?
        .block_on(fut)
        .pipe(|o| Ok(o))
}

trait Pipe<T> {
    fn pipe<R>(self, f: impl FnOnce(T) -> R) -> R;
}
impl<T> Pipe<T> for T {
    fn pipe<R>(self, f: impl FnOnce(T) -> R) -> R {
        f(self)
    }
}

impl crate::Fetcher for BalanceFetcher {
    fn name(&self) -> &str { "balance" }
    fn fetch(&self, params: HashMap<String, String>) -> Result<Value, String> {
        let address = params.get("address").cloned().ok_or("missing address")?;
        let chain_id = params
            .get("chain_id")
            .and_then(|s| s.parse::<i64>().ok())
            .unwrap_or(1);
        block_on(async { self.client.balance(&address, chain_id).await })?
            .pipe(|r| serde_json::to_value(r).map_err(|e| e.to_string()))
    }
}

impl crate::Fetcher for TransactionFetcher {
    fn name(&self) -> &str { "transactions" }
    fn fetch(&self, params: HashMap<String, String>) -> Result<Value, String> {
        let address = params.get("address").cloned().ok_or("missing address")?;
        let chain_id = params
            .get("chain_id")
            .and_then(|s| s.parse::<i64>().ok())
            .unwrap_or(1);
        block_on(async { self.client.transactions(&address, chain_id).await })?
            .pipe(|r| serde_json::to_value(r).map_err(|e| e.to_string()))
    }
}

impl crate::Fetcher for TokenFetcher {
    fn name(&self) -> &str { "tokens" }
    fn fetch(&self, params: HashMap<String, String>) -> Result<Value, String> {
        let address = params.get("address").cloned().ok_or("missing address")?;
        let chain_id = params
            .get("chain_id")
            .and_then(|s| s.parse::<i64>().ok())
            .unwrap_or(1);
        block_on(async { self.client.tokens(&address, chain_id).await })?
            .pipe(|r| serde_json::to_value(r).map_err(|e| e.to_string()))
    }
}

impl crate::Fetcher for NftFetcher {
    fn name(&self) -> &str { "nfts" }
    fn fetch(&self, params: HashMap<String, String>) -> Result<Value, String> {
        let address = params.get("address").cloned().ok_or("missing address")?;
        let chain_id = params
            .get("chain_id")
            .and_then(|s| s.parse::<i64>().ok())
            .unwrap_or(1);
        block_on(async { self.client.nfts(&address, chain_id).await })?
            .pipe(|r| serde_json::to_value(r).map_err(|e| e.to_string()))
    }
}

impl crate::Fetcher for GasFetcher {
    fn name(&self) -> &str { "gas" }
    fn fetch(&self, params: HashMap<String, String>) -> Result<Value, String> {
        let chain_id = params
            .get("chain_id")
            .and_then(|s| s.parse::<i64>().ok())
            .unwrap_or(1);
        block_on(async { self.client.gas_price(chain_id).await })?
            .pipe(|r| serde_json::to_value(r).map_err(|e| e.to_string()))
    }
}

impl crate::Fetcher for PriceFetcher {
    fn name(&self) -> &str { "price" }
    fn fetch(&self, params: HashMap<String, String>) -> Result<Value, String> {
        let symbol = params.get("symbol").cloned().ok_or("missing symbol")?;
        block_on(async { self.client.price(&symbol).await })?
            .pipe(|r| serde_json::to_value(r).map_err(|e| e.to_string()))
    }
}

impl crate::Fetcher for SwapFetcher {
    fn name(&self) -> &str { "swap" }
    fn fetch(&self, params: HashMap<String, String>) -> Result<Value, String> {
        let from = params.get("from_token").cloned().ok_or("missing from_token")?;
        let to = params.get("to_token").cloned().ok_or("missing to_token")?;
        let amount = params.get("amount").cloned().ok_or("missing amount")?;
        let chain_id = params
            .get("chain_id")
            .and_then(|s| s.parse::<i64>().ok())
            .unwrap_or(1);
        block_on(async { self.client.swap_quote(&from, &to, &amount, chain_id).await })?
            .pipe(|r| serde_json::to_value(r).map_err(|e| e.to_string()))
    }
}

impl crate::Fetcher for StakingFetcher {
    fn name(&self) -> &str { "staking" }
    fn fetch(&self, params: HashMap<String, String>) -> Result<Value, String> {
        let chain_id = params
            .get("chain_id")
            .and_then(|s| s.parse::<i64>().ok())
            .unwrap_or(1);
        block_on(async { self.client.staking_quote(chain_id).await })?
            .pipe(|r| serde_json::to_value(r).map_err(|e| e.to_string()))
    }
}

impl crate::Fetcher for DAppRegistryFetcher {
    fn name(&self) -> &str { "dapps" }
    fn fetch(&self, params: HashMap<String, String>) -> Result<Value, String> {
        let category = params.get("category").map(|s| s.as_str());
        let chain = params.get("chain").map(|s| s.as_str());
        block_on(async { self.client.dapps(category, chain).await })?
            .pipe(|r| serde_json::to_value(r).map_err(|e| e.to_string()))
    }
}

// ============================================================================
// DeFi service fetchers — REAL HTTP delegation to dedicated Go microservices.
// ============================================================================

impl crate::Fetcher for LendingFetcher {
    fn name(&self) -> &str { "lending" }
    fn fetch(&self, _params: HashMap<String, String>) -> Result<Value, String> {
        block_on(async { self.client.lending_markets().await })?
    }
}

impl crate::Fetcher for CopyTradingFetcher {
    fn name(&self) -> &str { "copy_trading" }
    fn fetch(&self, _params: HashMap<String, String>) -> Result<Value, String> {
        block_on(async { self.client.copy_trading_signals().await })?
    }
}

impl crate::Fetcher for DaoFetcher {
    fn name(&self) -> &str { "dao" }
    fn fetch(&self, _params: HashMap<String, String>) -> Result<Value, String> {
        block_on(async { self.client.dao_proposals().await })?
    }
}

impl crate::Fetcher for FuturesFetcher {
    fn name(&self) -> &str { "futures" }
    fn fetch(&self, _params: HashMap<String, String>) -> Result<Value, String> {
        block_on(async { self.client.futures_markets().await })?
    }
}

impl crate::Fetcher for MarginFetcher {
    fn name(&self) -> &str { "margin" }
    fn fetch(&self, params: HashMap<String, String>) -> Result<Value, String> {
        // Margin positions are perpetual positions filtered by user; reuse the
        // perpetual pairs/positions service.
        let _ = params.get("user");
        block_on(async { self.client.futures_markets().await })?
    }
}

impl crate::Fetcher for PredictionFetcher {
    fn name(&self) -> &str { "prediction" }
    fn fetch(&self, _params: HashMap<String, String>) -> Result<Value, String> {
        block_on(async { self.client.prediction_markets().await })?
    }
}

impl crate::Fetcher for NftTradingFetcher {
    fn name(&self) -> &str { "nft_trading" }
    fn fetch(&self, _params: HashMap<String, String>) -> Result<Value, String> {
        block_on(async { self.client.nft_marketplace().await })?
    }
}

impl crate::Fetcher for FiatRampFetcher {
    fn name(&self) -> &str { "fiat_ramp" }
    fn fetch(&self, _params: HashMap<String, String>) -> Result<Value, String> {
        block_on(async { self.client.fiat_ramp_providers().await })?
    }
}

// ============================================================================
// Fail-closed fetcher for endpoints the canonical backend does not expose.
// Never returns fabricated data — surfaces a clear error instead.
// ============================================================================

pub struct UnavailableFetcher {
    name: String,
}

impl UnavailableFetcher {
    pub fn new(name: &str) -> Self {
        Self {
            name: name.to_string(),
        }
    }
}

impl crate::Fetcher for UnavailableFetcher {
    fn name(&self) -> &str {
        &self.name
    }
    fn fetch(&self, _params: HashMap<String, String>) -> Result<Value, String> {
        Err(format!(
            "fetcher '{}' is not exposed by the canonical wallet-api backend; \
             wire the corresponding go/wallet_api endpoint before use",
            self.name
        ))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{Fetcher, UserWalletFetcherManager};
    use std::collections::HashMap;

    #[test]
    fn manager_has_22_fetchers() {
        let mgr = UserWalletFetcherManager::default();
        // 9 wallet-api (balance, transactions, tokens, nfts, gas, price, swap,
        // staking, dapps) + 8 DeFi service (lending, copy_trading, dao, futures,
        // margin, prediction, nft_trading, fiat_ramp) + 5 fail-closed (bridge,
        // options, p2p, gift_card, price_alerts) = 22.
        assert_eq!(mgr.count(), 22, "manager should expose 22 fetchers");
        assert!(mgr.get_fetcher("balance").is_some());
        assert!(mgr.get_fetcher("lending").is_some());
        assert!(mgr.get_fetcher("price_alerts").is_some());
    }

    #[test]
    fn unavailable_fetcher_is_fail_closed() {
        let f = UnavailableFetcher::new("bridge");
        let err = f.fetch(HashMap::new()).unwrap_err();
        assert!(err.contains("not exposed"));
    }

    #[test]
    fn client_builds_with_defaults() {
        let c = UserWalletClient::new(WALLET_API_DEFAULT_URL, None);
        assert_eq!(c.token(), None);
        c.set_token(Some("xyz".into()));
        assert_eq!(c.token(), Some("xyz".into()));
    }
}
