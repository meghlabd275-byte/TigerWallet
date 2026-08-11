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
}

impl UserWalletClient {
    pub fn new(base_url: impl Into<String>, token: Option<String>) -> Self {
        let http = Client::builder()
            .timeout(Duration::from_secs(30))
            .pool_max_idle_per_host(16)
            .tcp_nodelay(true)
            .build()
            .expect("reqwest client build");
        Self {
            http,
            base_url: base_url.into(),
            token: std::sync::RwLock::new(token),
        }
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
    fn manager_has_21_fetchers() {
        let mgr = UserWalletFetcherManager::default();
        assert_eq!(mgr.count(), 21, "manager should expose 21 fetchers");
        assert!(mgr.get_fetcher("balance").is_some());
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
