//! TigerWallet User Wallet — Rust client.
//!
//! A typed, async HTTP client for the canonical TigerWallet backend
//! (`go/wallet_api`, default `http://localhost:8443`). It implements the SAME
//! fetcher set as the web/desktop/android/ios clients — login/register,
//! create/list wallets, balance, transactions, send, sign, tokens, NFTs,
//! gas, price, chains, network status, swap quote, staking quote — plus the
//! auxiliary DeFi features (QR parsing helpers, crypto card, P2P, fiat
//! on/off-ramp, convert via swap, staking actions).
//!
//! All signing is delegated to the backend (`/send`, `/sign`, `/non_evm/*`);
//! this client NEVER fabricates keys, addresses, signatures, or transaction
//! hashes. The 24-word seed is generated server-side (real entropy) and
//! stored only as a scrypt+AES-GCM-encrypted blob — the backend never holds
//! the plaintext seed at rest.

use serde::{de::DeserializeOwned, Deserialize, Serialize};

pub mod wc;
pub use wc::WalletConnectSocket;

const DEFAULT_BASE_URL: &str = "http://localhost:8443";

/// Errors returned by the client. Never leak secrets in messages.
#[derive(Debug, Clone)]
pub enum WalletError {
    Http(String),
    Api(String),
    Json(String),
}

impl std::fmt::Display for WalletError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            WalletError::Http(m) => write!(f, "http error: {m}"),
            WalletError::Api(m) => write!(f, "api error: {m}"),
            WalletError::Json(m) => write!(f, "json error: {m}"),
        }
    }
}

impl std::error::Error for WalletError {}

// ---------------------------------------------------------------------------
// Data models (mirror the backend JSON shapes)
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WalletRecord {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub address: String,
    #[serde(default)]
    pub label: String,
    #[serde(default)]
    pub chain_id: i64,
    #[serde(default, rename = "type")]
    pub wallet_type: String,
    #[serde(default)]
    pub created_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BalanceResult {
    #[serde(default)]
    pub address: String,
    #[serde(default)]
    pub chain_id: i64,
    #[serde(default)]
    pub balance: String,
    #[serde(default)]
    pub symbol: String,
    #[serde(default)]
    pub decimals: i64,
    #[serde(default)]
    pub usd_value: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    #[serde(default)]
    pub hash: String,
    #[serde(default)]
    pub from: String,
    #[serde(default)]
    pub to: String,
    #[serde(default)]
    pub value: String,
    #[serde(default)]
    pub timestamp: i64,
    #[serde(default)]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenBalance {
    #[serde(default)]
    pub contract: String,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub symbol: String,
    #[serde(default)]
    pub decimals: i64,
    #[serde(default)]
    pub balance: String,
    #[serde(default)]
    pub usd_value: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Nft {
    #[serde(default)]
    pub contract: String,
    #[serde(default)]
    pub token_id: String,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub image: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasPrice {
    #[serde(default)]
    pub chain_id: i64,
    #[serde(default)]
    pub gas_price: String,
    #[serde(default)]
    pub max_fee_per_gas: String,
    #[serde(default)]
    pub max_priority_fee_per_gas: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceInfo {
    #[serde(default)]
    pub symbol: String,
    #[serde(default)]
    pub price: f64,
    #[serde(default)]
    pub currency: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainInfo {
    #[serde(default)]
    pub id: i64,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub symbol: String,
    #[serde(default)]
    pub chain_type: String,
    #[serde(default)]
    pub decimals: i64,
    #[serde(default)]
    pub is_testnet: bool,
    #[serde(default)]
    pub explorer_url: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkStatus {
    pub chain_id: i64,
    #[serde(default)]
    pub block_number: String,
    #[serde(default)]
    pub block_number_int: u64,
    #[serde(default)]
    pub syncing: bool,
    #[serde(default)]
    pub rpc_endpoint: String,
    #[serde(default)]
    pub latency_ms: i64,
    #[serde(default)]
    pub timestamp: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapQuote {
    #[serde(default)]
    pub from_token: String,
    #[serde(default)]
    pub to_token: String,
    #[serde(default)]
    pub from_amount: String,
    #[serde(default)]
    pub to_amount: String,
    #[serde(default)]
    pub price_impact: f64,
    #[serde(default)]
    pub route: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakingAsset {
    #[serde(default)]
    pub symbol: String,
    #[serde(default)]
    pub chain_id: i64,
    #[serde(default)]
    pub apy: f64,
    #[serde(default)]
    pub min_stake: f64,
    #[serde(default)]
    pub lock_period: i64,
    #[serde(default)]
    pub verified: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakingQuoteResponse {
    #[serde(default)]
    pub success: bool,
    #[serde(default)]
    pub assets: Vec<StakingAsset>,
    #[serde(default)]
    pub apy: f64,
    #[serde(default)]
    pub min_stake: f64,
    #[serde(default)]
    pub lock_period: i64,
}

// ---------------------------------------------------------------------------
// Passkey / lock / unlock models (mirror the backend JSON shapes)
// ---------------------------------------------------------------------------

/// Parameters for `POST /passkey/wallet` — create a wallet backed by a passkey
/// credential. Required fields are non-optional; the optional ones are only
/// sent when present.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PasskeyWalletParams {
    pub label: String,
    pub chain_id: i64,
    pub account_index: i64,
    pub entropy_bits: i64,
    pub credential_id: String,
    pub public_key: String,
    #[serde(default)]
    pub sign_count: i64,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub attestation: Option<String>,
}

/// Result of `POST /passkey/wallet`. Includes the server-generated mnemonic,
/// unlock key, and unlock token (returned once at creation time).
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PasskeyWalletResult {
    #[serde(default)]
    pub wallet_id: String,
    #[serde(default)]
    pub label: String,
    #[serde(default)]
    pub chain_id: i64,
    #[serde(default)]
    pub address: String,
    #[serde(default)]
    pub derivation_path: String,
    #[serde(default)]
    pub mnemonic: String,
    #[serde(default)]
    pub unlock_key: String,
    #[serde(default)]
    pub unlock_token: String,
}

/// Parameters for `POST /wallets/:id/lock` — set up a passcode and/or passkey
/// lock on a wallet. At least one of the optional fields should be present.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct LockSetupParams {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub passcode: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub passkey_credential_id: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub passkey_public_key: Option<String>,
}

/// Result of `POST /wallets/:id/lock`.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct LockSetupResult {
    #[serde(default)]
    pub status: String,
    #[serde(default)]
    pub has_passcode: bool,
    #[serde(default)]
    pub has_passkey: bool,
}

/// Parameters for `POST /wallets/:id/unlock` — unlock a wallet with a passcode,
/// password, a passkey assertion, or an already-unwrapped unlock key.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct UnlockParams {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub passcode: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub password: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub passkey_assertion: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub passkey_auth_data: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub passkey_client_data: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub unwrapped_unlock_key: Option<String>,
}

/// Result of `POST /wallets/:id/unlock` — a short-lived unlock token used to
/// authorize signing operations without re-sending the passcode/password.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct UnlockResult {
    #[serde(default)]
    pub unlock_token: String,
    #[serde(default)]
    pub expires_in: i64,
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/// UserWalletClient is a thin async wrapper over the canonical wallet_api
/// backend. It performs NO local key material handling — all signing is done
/// server-side from the scrypt-encrypted seed blob (decrypted with the user's
/// password at signing time). This guarantees the client never fabricates a
/// signature or address.
pub struct UserWalletClient {
    base_url: String,
    token: std::sync::Mutex<Option<String>>,
    http: reqwest::Client,
}

impl UserWalletClient {
    pub fn new(base_url: Option<&str>) -> Self {
        let base_url = base_url.unwrap_or(DEFAULT_BASE_URL).trim_end_matches('/').to_string();
        let http = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(30))
            .build()
            .expect("reqwest client build");
        Self { base_url, token: std::sync::Mutex::new(None), http }
    }

    pub fn set_token(&self, token: Option<String>) {
        *self.token.lock().unwrap() = token;
    }

    pub fn token(&self) -> Option<String> {
        self.token.lock().unwrap().clone()
    }

    /// Open a live WalletConnect socket for a pairing topic against the
    /// canonical dapp relay (proxied through this client's base_url).
    pub async fn connect_walletconnect(&self, topic: &str) -> Result<WalletConnectSocket, WalletError> {
        WalletConnectSocket::connect(&self.base_url, topic).await
    }

    fn url(&self, path: &str) -> String {
        // The canonical wallet_api mounts ALL API routes under /api/v1 (only
        // /health is root-level). Normalize here so callers may pass either
        // "/api/v1/x" or "/x" — both resolve to the same backend route.
        if path.starts_with("/api/v1") || path == "/health" {
            format!("{}{}", self.base_url, path)
        } else {
            format!("{}/api/v1{}", self.base_url, path)
        }
    }

    async fn get<T: DeserializeOwned>(&self, path: &str) -> Result<T, WalletError> {
        let mut req = self.http.get(self.url(path));
        if let Some(t) = self.token() {
            req = req.bearer_auth(t);
        }
        let resp = req.send().await.map_err(|e| WalletError::Http(e.to_string()))?;
        let status = resp.status();
        let text = resp.text().await.map_err(|e| WalletError::Http(e.to_string()))?;
        if !status.is_success() {
            return Err(WalletError::Api(text));
        }
        serde_json::from_str(&text).map_err(|e| WalletError::Json(e.to_string()))
    }

    async fn get_query<T: DeserializeOwned>(
        &self,
        path: &str,
        query: &[(&str, String)],
    ) -> Result<T, WalletError> {
        let mut req = self.http.get(self.url(path)).query(query);
        if let Some(t) = self.token() {
            req = req.bearer_auth(t);
        }
        let resp = req.send().await.map_err(|e| WalletError::Http(e.to_string()))?;
        let status = resp.status();
        let text = resp.text().await.map_err(|e| WalletError::Http(e.to_string()))?;
        if !status.is_success() {
            return Err(WalletError::Api(text));
        }
        serde_json::from_str(&text).map_err(|e| WalletError::Json(e.to_string()))
    }

    async fn post<T: DeserializeOwned, B: Serialize>(
        &self,
        path: &str,
        body: &B,
    ) -> Result<T, WalletError> {
        let mut req = self.http.post(self.url(path)).json(body);
        if let Some(t) = self.token() {
            req = req.bearer_auth(t);
        }
        let resp = req.send().await.map_err(|e| WalletError::Http(e.to_string()))?;
        let status = resp.status();
        let text = resp.text().await.map_err(|e| WalletError::Http(e.to_string()))?;
        if !status.is_success() {
            return Err(WalletError::Api(text));
        }
        serde_json::from_str(&text).map_err(|e| WalletError::Json(e.to_string()))
    }

    async fn put<T: DeserializeOwned, B: Serialize>(
        &self,
        path: &str,
        body: &B,
    ) -> Result<T, WalletError> {
        let mut req = self.http.put(self.url(path)).json(body);
        if let Some(t) = self.token() {
            req = req.bearer_auth(t);
        }
        let resp = req.send().await.map_err(|e| WalletError::Http(e.to_string()))?;
        let status = resp.status();
        let text = resp.text().await.map_err(|e| WalletError::Http(e.to_string()))?;
        if !status.is_success() {
            return Err(WalletError::Api(text));
        }
        serde_json::from_str(&text).map_err(|e| WalletError::Json(e.to_string()))
    }

    async fn delete<T: DeserializeOwned>(&self, path: &str) -> Result<T, WalletError> {
        let mut req = self.http.delete(self.url(path));
        if let Some(t) = self.token() {
            req = req.bearer_auth(t);
        }
        let resp = req.send().await.map_err(|e| WalletError::Http(e.to_string()))?;
        let status = resp.status();
        let text = resp.text().await.map_err(|e| WalletError::Http(e.to_string()))?;
        if !status.is_success() {
            return Err(WalletError::Api(text));
        }
        serde_json::from_str(&text).map_err(|e| WalletError::Json(e.to_string()))
    }

    // ---- Auth ----

    pub async fn login(&self, email: &str, password: &str) -> Result<String, WalletError> {
        #[derive(Serialize)]
        struct Req<'a> { email: &'a str, password: &'a str }
        #[derive(Deserialize)]
        #[allow(dead_code)]
        struct Resp { #[serde(default)] token: String, #[serde(default)] user_id: String }
        let r: Resp = self.post("/api/v1/auth/login", &Req { email, password }).await?;
        if r.token.is_empty() {
            return Err(WalletError::Api("no token in response".into()));
        }
        self.set_token(Some(r.token.clone()));
        Ok(r.token)
    }

    pub async fn register(&self, email: &str, password: &str) -> Result<String, WalletError> {
        #[derive(Serialize)]
        struct Req<'a> { email: &'a str, password: &'a str }
        #[derive(Deserialize)]
        #[allow(dead_code)]
        struct Resp { #[serde(default)] token: String, #[serde(default)] user_id: String }
        let r: Resp = self.post("/api/v1/auth/register", &Req { email, password }).await?;
        if r.token.is_empty() {
            return Err(WalletError::Api("no token in response".into()));
        }
        self.set_token(Some(r.token.clone()));
        Ok(r.token)
    }

    // POST /auth/guest { device_id } -> { user_id, token, guest: true }. Public
    // (no auth required). Provisions an anonymous guest account so the user can
    // create/import a wallet without registering. The token is persisted exactly
    // like login (set_token -> in-memory Mutex<Option<String>>).
    pub async fn guest_auth(&self, device_id: &str) -> Result<String, WalletError> {
        #[derive(Serialize)]
        struct Req<'a> { device_id: &'a str }
        #[derive(Deserialize)]
        #[allow(dead_code)]
        struct Resp { #[serde(default)] token: String, #[serde(default)] user_id: String, #[serde(default)] guest: bool }
        let r: Resp = self.post("/api/v1/auth/guest", &Req { device_id }).await?;
        if r.token.is_empty() {
            return Err(WalletError::Api("no token in response".into()));
        }
        self.set_token(Some(r.token.clone()));
        Ok(r.token)
    }

    // ---- Wallets ----

    pub async fn create_wallet(
        &self,
        label: &str,
        password: &str,
        chain_id: i64,
    ) -> Result<WalletRecord, WalletError> {
        #[derive(Serialize)]
        struct Req<'a> { label: &'a str, password: &'a str, chain_id: i64 }
        self.post("/api/v1/wallets", &Req { label, password, chain_id }).await
    }

    pub async fn list_wallets(&self) -> Result<Vec<WalletRecord>, WalletError> {
        #[derive(Deserialize)]
        struct Resp { #[serde(default)] wallets: Vec<WalletRecord> }
        let r: Resp = self.get("/api/v1/wallets").await?;
        Ok(r.wallets)
    }

    // ---- Balance / tokens / NFTs ----

    pub async fn get_balance(&self, address: &str, chain_id: i64) -> Result<BalanceResult, WalletError> {
        self.get_query("/api/v1/balance", &[
            ("address", address.to_string()),
            ("chain_id", chain_id.to_string()),
        ]).await
    }

    /// get_balances aggregates the real on-chain balance of every wallet the
    /// authenticated user owns (GET /wallets then one GET /balance per wallet).
    /// Parity with the web/desktop `getBalances` convenience method.
    pub async fn get_balances(&self) -> Result<Vec<(WalletRecord, BalanceResult)>, WalletError> {
        let wallets = self.list_wallets().await?;
        let mut out = Vec::with_capacity(wallets.len());
        for w in wallets {
            let b = self.get_balance(&w.address, w.chain_id).await?;
            out.push((w, b));
        }
        Ok(out)
    }

    pub async fn get_token_balances(&self, address: &str, chain_id: i64) -> Result<Vec<TokenBalance>, WalletError> {
        #[derive(Deserialize)]
        struct Resp { #[serde(default)] tokens: Vec<TokenBalance> }
        let r: Resp = self.get_query("/api/v1/tokens", &[
            ("address", address.to_string()),
            ("chain_id", chain_id.to_string()),
        ]).await?;
        Ok(r.tokens)
    }

    pub async fn get_nfts(&self, address: &str, chain_id: i64) -> Result<Vec<Nft>, WalletError> {
        #[derive(Deserialize)]
        struct Resp { #[serde(default)] nfts: Vec<Nft> }
        let r: Resp = self.get_query("/api/v1/nfts", &[
            ("address", address.to_string()),
            ("chain_id", chain_id.to_string()),
        ]).await?;
        Ok(r.nfts)
    }

    // ---- Transactions / send / sign ----

    pub async fn get_transactions(&self, address: &str, chain_id: i64) -> Result<Vec<Transaction>, WalletError> {
        #[derive(Deserialize)]
        struct Resp { #[serde(default)] transactions: Vec<Transaction> }
        let r: Resp = self.get_query("/api/v1/transactions", &[
            ("address", address.to_string()),
            ("chain_id", chain_id.to_string()),
        ]).await?;
        Ok(r.transactions)
    }

    pub async fn send_transaction(
        &self,
        wallet_id: &str,
        password: &str,
        to: &str,
        amount: &str,
        chain_id: i64,
        token_address: Option<&str>,
        unlock_token: Option<&str>,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req<'a> {
            wallet_id: &'a str,
            password: &'a str,
            to: &'a str,
            amount: &'a str,
            chain_id: i64,
            #[serde(skip_serializing_if = "Option::is_none")]
            token_address: Option<&'a str>,
            #[serde(skip_serializing_if = "Option::is_none")]
            unlock_token: Option<&'a str>,
        }
        self.post("/api/v1/send", &Req {
            wallet_id, password, to, amount, chain_id, token_address, unlock_token,
        }).await
    }

    // POST /api/v1/auto-send with the SAME body as /send, plus optional
    // ?master_wallet_id=<id> query. Same Bearer JWT auth as /send. Returns the
    // existing send response (raw JSON) PLUS { auto_approved, auto_approval_reason }.
    pub async fn auto_send_transaction(
        &self,
        wallet_id: &str,
        password: &str,
        to: &str,
        amount: &str,
        chain_id: i64,
        token_address: Option<&str>,
        master_wallet_id: Option<&str>,
        unlock_token: Option<&str>,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req<'a> {
            wallet_id: &'a str,
            password: &'a str,
            to: &'a str,
            amount: &'a str,
            chain_id: i64,
            #[serde(skip_serializing_if = "Option::is_none")]
            token_address: Option<&'a str>,
            #[serde(skip_serializing_if = "Option::is_none")]
            unlock_token: Option<&'a str>,
        }
        let path = match master_wallet_id {
            Some(mw) => format!("/api/v1/auto-send?master_wallet_id={}", mw),
            None => "/api/v1/auto-send".to_string(),
        };
        self.post(&path, &Req {
            wallet_id, password, to, amount, chain_id, token_address, unlock_token,
        }).await
    }

    // GET /api/v1/transactions/:txHash?chain_id=N -> { status, block_number?, confirmations? }.
    // Transaction-status proxy (explorer receipt lookup).
    pub async fn get_transaction_status(
        &self,
        tx_hash: &str,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        let path = format!("/api/v1/transactions/{}?chain_id={}", tx_hash, chain_id);
        self.get(&path).await
    }

    pub async fn sign_message(
        &self,
        wallet_id: &str,
        password: &str,
        message: &str,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req<'a> { wallet_id: &'a str, password: &'a str, message: &'a str }
        self.post("/api/v1/sign", &Req { wallet_id, password, message }).await
    }

    // ---- Gas / price / chains / status ----

    pub async fn get_gas_price(&self, chain_id: i64) -> Result<GasPrice, WalletError> {
        self.get_query("/api/v1/gas", &[("chain_id", chain_id.to_string())]).await
    }

    pub async fn get_token_price(&self, symbol: &str) -> Result<PriceInfo, WalletError> {
        self.get_query("/api/v1/price", &[("symbol", symbol.to_string())]).await
    }

    pub async fn get_chains(&self) -> Result<Vec<ChainInfo>, WalletError> {
        #[derive(Deserialize)]
        struct Resp { #[serde(default)] chains: Vec<ChainInfo> }
        let r: Resp = self.get("/api/v1/chains").await?;
        Ok(r.chains)
    }

    /// get_network_status performs a REAL eth_blockNumber RPC call via the
    /// backend's /network-status endpoint (never a fabricated block_number:0).
    pub async fn get_network_status(&self, chain_id: i64) -> Result<NetworkStatus, WalletError> {
        let path = format!("/network-status?chain_id={}", chain_id);
        self.get::<NetworkStatus>(&path).await
    }

    // ---- Swap / Staking / Convert ----

    pub async fn get_swap_quote(
        &self,
        from_token: &str,
        to_token: &str,
        from_amount: &str,
        chain_id: i64,
    ) -> Result<SwapQuote, WalletError> {
        self.get_query("/api/v1/swap/quote", &[
            ("from_token", from_token.to_string()),
            ("to_token", to_token.to_string()),
            ("from_amount", from_amount.to_string()),
            ("chain_id", chain_id.to_string()),
        ]).await
    }

    /// Convert is the same path as swap (cross-token conversion). Provided as
    /// a distinct method for client parity + semantic clarity.
    pub async fn get_convert_quote(
        &self,
        from_token: &str,
        to_token: &str,
        from_amount: &str,
        chain_id: i64,
    ) -> Result<SwapQuote, WalletError> {
        self.get_swap_quote(from_token, to_token, from_amount, chain_id).await
    }

    pub async fn get_staking_quote(&self, _chain_id: i64) -> Result<StakingQuoteResponse, WalletError> {
        // The backend returns all supported staking assets (chain_id is
        // accepted but ignored server-side; the response lists every chain).
        self.get("/api/v1/staking/quote").await
    }

    pub async fn stake(
        &self,
        wallet_id: &str,
        password: &str,
        token: &str,
        amount: &str,
        chain_id: i64,
        staking_contract: Option<&str>,
        call_data: Option<&str>,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req<'a> {
            wallet_id: &'a str,
            password: &'a str,
            token: &'a str,
            amount: &'a str,
            chain_id: i64,
            #[serde(skip_serializing_if = "Option::is_none")]
            staking_contract: Option<&'a str>,
            #[serde(skip_serializing_if = "Option::is_none")]
            call_data: Option<&'a str>,
        }
        self.post("/api/v1/staking/stake", &Req {
            wallet_id, password, token, amount, chain_id, staking_contract, call_data,
        }).await
    }

    pub async fn unstake(
        &self,
        wallet_id: &str,
        password: &str,
        token: &str,
        position_id: &str,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req<'a> {
            wallet_id: &'a str, password: &'a str, token: &'a str,
            position_id: &'a str, chain_id: i64,
        }
        self.post("/api/v1/staking/unstake", &Req {
            wallet_id, password, token, position_id, chain_id,
        }).await
    }

    pub async fn claim_rewards(
        &self,
        wallet_id: &str,
        password: &str,
        token: &str,
        position_id: &str,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req<'a> {
            wallet_id: &'a str, password: &'a str, token: &'a str,
            position_id: &'a str, chain_id: i64,
        }
        self.post("/api/v1/staking/claim", &Req {
            wallet_id, password, token, position_id, chain_id,
        }).await
    }

    // ---- Auxiliary DeFi (fiat ramp, crypto card, P2P) ----
    // These delegate to their dedicated Go microservices via the backend.
    // The client returns the raw JSON value; callers map per the service API.

    pub async fn fiat_ramp_providers(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/api/v1/ramp/providers").await
    }

    pub async fn fiat_ramp_quote(&self, provider_id: &str, amount: &str, fiat: &str, crypto: &str, method: &str) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req<'a> {
            provider_id: &'a str, amount: &'a str, fiat_currency: &'a str,
            crypto_currency: &'a str, payment_method: &'a str,
        }
        self.post("/api/v1/ramp/quote", &Req {
            provider_id, amount, fiat_currency: fiat, crypto_currency: crypto, payment_method: method,
        }).await
    }

    pub async fn crypto_card_rates(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/api/v1/cards/rates").await
    }

    pub async fn p2p_listings(&self) -> Result<serde_json::Value, WalletError> {
        // Alias of get_p2p_adverts — the backend route is /p2p/adverts.
        self.get("/api/v1/p2p/adverts").await
    }

    // -----------------------------------------------------------------------
    // Canonical flat-route fetcher set (:8443, single base_url).
    // Matches the web/desktop/android/ios client contract.
    // -----------------------------------------------------------------------

    // ---- Auth / profile ----

    pub async fn logout(&self) -> Result<(), String> {
        self.set_token(None);
        Ok(())
    }

    /// get_profile decodes the JWT payload locally (no network). Returns the
    /// claims as a JSON object. Errors if no token is set.
    pub async fn get_profile(&self) -> Result<serde_json::Value, String> {
        let token = self.token().ok_or_else(|| "Not authenticated".to_string())?;
        let parts: Vec<&str> = token.split('.').collect();
        if parts.len() < 2 {
            return Err("Malformed token".into());
        }
        let payload_b64 = parts[1];
        // JWT uses base64url without padding.
        let mut decoded = String::new();
        for c in payload_b64.chars() {
            match c {
                '-' => decoded.push('+'),
                '_' => decoded.push('/'),
                _ => decoded.push(c),
            }
        }
        while decoded.len() % 4 != 0 {
            decoded.push('=');
        }
        let bytes = base64_decode(&decoded)?;
        let s = String::from_utf8(bytes).map_err(|e| e.to_string())?;
        serde_json::from_str(&s).map_err(|e| e.to_string())
    }

    pub async fn health(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/health").await
    }

    // ---- Wallets (extended) ----

    pub async fn import_wallet(
        &self,
        label: String,
        password: String,
        mnemonic: String,
        chain_id: Option<i64>,
        passphrase: Option<String>,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            label: String,
            password: String,
            mnemonic: String,
            #[serde(skip_serializing_if = "Option::is_none")]
            chain_id: Option<i64>,
            #[serde(skip_serializing_if = "Option::is_none")]
            passphrase: Option<String>,
        }
        self.post("/wallets", &Req {
            label, password, mnemonic, chain_id, passphrase,
        }).await
    }

    pub async fn export_encrypted_seed(
        &self,
        wallet_id: String,
        password: String,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req { wallet_id: String, password: String }
        let path = format!("/wallets/{}/export-encrypted-seed", url_encode(&wallet_id));
        self.post(&path, &Req { wallet_id, password }).await
    }

    pub async fn import_encrypted_seed(
        &self,
        encrypted_seed: String,
        password: String,
        label: Option<String>,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            encrypted_seed: String,
            password: String,
            #[serde(skip_serializing_if = "Option::is_none")]
            label: Option<String>,
        }
        self.post("/wallets/import-encrypted-seed", &Req {
            encrypted_seed, password, label,
        }).await
    }

    // ---- NFTs ----

    pub async fn transfer_nft(
        &self,
        wallet_id: String,
        password: String,
        to: String,
        token_id: String,
        contract_address: String,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            wallet_id: String,
            password: String,
            to: String,
            token_id: String,
            contract_address: String,
            chain_id: i64,
        }
        self.post("/nft/transfer", &Req {
            wallet_id, password, to, token_id, contract_address, chain_id,
        }).await
    }

    // ---- Transactions / gas ----

    pub async fn get_transaction_receipt(
        &self,
        tx_hash: String,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        let path = format!("/transactions/{}?chain_id={}", url_encode(&tx_hash), chain_id);
        self.get(&path).await
    }

    pub async fn estimate_gas(
        &self,
        from: String,
        to: String,
        value: Option<String>,
        data: Option<String>,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            from: String,
            to: String,
            chain_id: i64,
            #[serde(skip_serializing_if = "Option::is_none")]
            value: Option<String>,
            #[serde(skip_serializing_if = "Option::is_none")]
            data: Option<String>,
        }
        self.post("/gas/estimate", &Req { from, to, value, data, chain_id }).await
    }

    // ---- Swap / AMM ----

    pub async fn execute_swap(
        &self,
        wallet_id: String,
        password: String,
        from_token: String,
        to_token: String,
        from_amount: String,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            wallet_id: String,
            password: String,
            from_token: String,
            to_token: String,
            from_amount: String,
            chain_id: i64,
        }
        self.post("/swap/execute", &Req {
            wallet_id, password, from_token, to_token, from_amount, chain_id,
        }).await
    }

    pub async fn get_amm_quote(
        &self,
        from_token: String,
        to_token: String,
        from_amount: String,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        self.get_query("/amm/quote", &[
            ("from_token", from_token),
            ("to_token", to_token),
            ("from_amount", from_amount),
            ("chain_id", chain_id.to_string()),
        ]).await
    }

    pub async fn amm_swap(
        &self,
        wallet_id: String,
        password: String,
        from_token: String,
        to_token: String,
        from_amount: String,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            wallet_id: String,
            password: String,
            from_token: String,
            to_token: String,
            from_amount: String,
            chain_id: i64,
        }
        self.post("/amm/swap", &Req {
            wallet_id, password, from_token, to_token, from_amount, chain_id,
        }).await
    }

    // ---- Crypto cards ----

    pub async fn get_crypto_card_balance(&self, card_id: String) -> Result<serde_json::Value, WalletError> {
        let path = format!("/cards/{}/balance", url_encode(&card_id));
        self.get(&path).await
    }

    pub async fn get_card_transactions(&self, card_id: String) -> Result<serde_json::Value, WalletError> {
        let path = format!("/cards/{}/transactions", url_encode(&card_id));
        self.get(&path).await
    }

    // ---- Passkey / lock / unlock ----

    /// passkey_create_wallet creates a wallet backed by a passkey credential via
    /// `POST /api/v1/passkey/wallet`. The backend generates the entropy, seed,
    /// address, and a one-time unlock key + unlock token returned in the result.
    pub async fn passkey_create_wallet(
        &self,
        params: PasskeyWalletParams,
    ) -> Result<PasskeyWalletResult, WalletError> {
        self.post("/api/v1/passkey/wallet", &params).await
    }

    /// setup_lock sets up a passcode and/or passkey lock on a wallet via
    /// `POST /api/v1/wallets/:id/lock`. Returns the resulting lock status.
    pub async fn setup_lock(
        &self,
        wallet_id: &str,
        params: LockSetupParams,
    ) -> Result<LockSetupResult, WalletError> {
        let path = format!("/api/v1/wallets/{}/lock", url_encode(wallet_id));
        self.post(&path, &params).await
    }

    /// unlock_wallet unlocks a wallet via `POST /api/v1/wallets/:id/unlock` using
    /// a passcode, password, passkey assertion, or an already-unwrapped unlock
    /// key. Returns a short-lived unlock token.
    pub async fn unlock_wallet(
        &self,
        wallet_id: &str,
        params: UnlockParams,
    ) -> Result<UnlockResult, WalletError> {
        let path = format!("/api/v1/wallets/{}/unlock", url_encode(wallet_id));
        self.post(&path, &params).await
    }

    // ---- KYC ----

    /// get_kyc_status fetches the caller's KYC status via
    /// `GET /api/v1/kyc/status`. If `user_id` is provided it is sent as the
    /// `user_id` query parameter; otherwise the backend infers the user from
    /// the Bearer JWT. The response shape is opaque, so it is returned as a raw
    /// JSON value.
    pub async fn get_kyc_status(
        &self,
        user_id: Option<&str>,
    ) -> Result<serde_json::Value, WalletError> {
        match user_id {
            Some(uid) => self.get_query("/api/v1/kyc/status", &[("user_id", uid.to_string())]).await,
            None => self.get("/api/v1/kyc/status").await,
        }
    }

    /// register_kyc starts a KYC registration via `POST /api/v1/kyc/register`.
    /// The request body is opaque JSON defined by the caller; the response is
    /// returned as a raw JSON value.
    pub async fn register_kyc(
        &self,
        body: serde_json::Value,
    ) -> Result<serde_json::Value, WalletError> {
        self.post("/api/v1/kyc/register", &body).await
    }

    /// submit_kyc submits KYC data via `POST /api/v1/kyc/submit`. The request
    /// body is opaque JSON; the response is returned as a raw JSON value.
    pub async fn submit_kyc(
        &self,
        body: serde_json::Value,
    ) -> Result<serde_json::Value, WalletError> {
        self.post("/api/v1/kyc/submit", &body).await
    }

    /// submit_kyc_document uploads a KYC document via
    /// `POST /api/v1/kyc/document` using multipart/form-data. The caller builds
    /// the `reqwest::multipart::Form` (file part + metadata fields); this method
    /// attaches the Bearer JWT and dispatches the multipart request directly,
    /// since the JSON-only `post` helper cannot carry a multipart body.
    pub async fn submit_kyc_document(
        &self,
        form: reqwest::multipart::Form,
    ) -> Result<serde_json::Value, WalletError> {
        let mut req = self.http.post(self.url("/api/v1/kyc/document")).multipart(form);
        if let Some(t) = self.token() {
            req = req.bearer_auth(t);
        }
        let resp = req.send().await.map_err(|e| WalletError::Http(e.to_string()))?;
        let status = resp.status();
        let text = resp.text().await.map_err(|e| WalletError::Http(e.to_string()))?;
        if !status.is_success() {
            return Err(WalletError::Api(text));
        }
        serde_json::from_str(&text).map_err(|e| WalletError::Json(e.to_string()))
    }

    /// get_kyc_session fetches a KYC session by id via
    /// `GET /api/v1/kyc/session/:id`. The response shape is opaque, so it is
    /// returned as a raw JSON value.
    pub async fn get_kyc_session(
        &self,
        session_id: &str,
    ) -> Result<serde_json::Value, WalletError> {
        let path = format!("/api/v1/kyc/session/{}", url_encode(session_id));
        self.get(&path).await
    }

    // ---- P2P / fiat ramp ----

    /// get_p2p_adverts fetches the P2P advert listings from
    /// `GET /api/v1/p2p/adverts`.
    pub async fn get_p2p_adverts(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/api/v1/p2p/adverts").await
    }

    /// create_p2p_order creates a new P2P trade order via
    /// `POST /api/v1/p2p/orders`. KYC-gated — the backend returns 403 if the
    /// caller has not completed KYC.
    pub async fn create_p2p_order(
        &self,
        body: serde_json::Value,
    ) -> Result<serde_json::Value, WalletError> {
        self.post("/api/v1/p2p/orders", &body).await
    }

    pub async fn get_fiat_offramp_quote(
        &self,
        provider_id: String,
        amount: String,
        fiat: String,
        crypto: String,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            provider_id: String,
            amount: String,
            fiat: String,
            crypto: String,
        }
        self.post("/ramp/offramp-quote", &Req { provider_id, amount, fiat, crypto }).await
    }

    // ---- Non-EVM (Solana / Cosmos / etc.) ----

    pub async fn non_evm_address(
        &self,
        seed: String,
        chain_type: String,
        chain_id: i64,
        path: Option<String>,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            seed: String,
            chain_type: String,
            chain_id: i64,
            #[serde(skip_serializing_if = "Option::is_none")]
            path: Option<String>,
        }
        self.post("/non_evm/address", &Req { seed, chain_type, chain_id, path }).await
    }

    pub async fn non_evm_sign(
        &self,
        seed: String,
        chain_type: String,
        chain_id: i64,
        message_hash: String,
        path: Option<String>,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            seed: String,
            chain_type: String,
            chain_id: i64,
            message_hash: String,
            #[serde(skip_serializing_if = "Option::is_none")]
            path: Option<String>,
        }
        self.post("/non_evm/sign", &Req { seed, chain_type, chain_id, message_hash, path }).await
    }

    pub async fn non_evm_send(
        &self,
        seed: String,
        chain_type: String,
        chain_id: i64,
        to: String,
        value: String,
        path: Option<String>,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            seed: String,
            chain_type: String,
            chain_id: i64,
            to: String,
            value: String,
            #[serde(skip_serializing_if = "Option::is_none")]
            path: Option<String>,
        }
        self.post("/non_evm/send", &Req { seed, chain_type, chain_id, to, value, path }).await
    }

    // ---- Address book ----

    pub async fn get_address_book_contacts(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/address-book/contacts").await
    }

    pub async fn add_contact(
        &self,
        name: String,
        address: String,
        chain_id: Option<i64>,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            name: String,
            address: String,
            #[serde(skip_serializing_if = "Option::is_none")]
            chain_id: Option<i64>,
        }
        self.post("/address-book/contacts", &Req { name, address, chain_id }).await
    }

    pub async fn update_contact(
        &self,
        id: String,
        name: Option<String>,
        address: Option<String>,
        chain_id: Option<i64>,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            #[serde(skip_serializing_if = "Option::is_none")]
            name: Option<String>,
            #[serde(skip_serializing_if = "Option::is_none")]
            address: Option<String>,
            #[serde(skip_serializing_if = "Option::is_none")]
            chain_id: Option<i64>,
        }
        let path = format!("/address-book/contacts/{}", url_encode(&id));
        self.put(&path, &Req { name, address, chain_id }).await
    }

    pub async fn delete_contact(&self, id: String) -> Result<serde_json::Value, WalletError> {
        let path = format!("/address-book/contacts/{}", url_encode(&id));
        self.delete(&path).await
    }

    // ---- Devices ----

    pub async fn get_devices(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/devices").await
    }

    pub async fn register_device(
        &self,
        name: String,
        device_type: String,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req { name: String, device_type: String }
        self.post("/devices", &Req { name, device_type }).await
    }

    pub async fn sync_device(&self, device_id: String) -> Result<serde_json::Value, WalletError> {
        let path = format!("/devices/{}/sync", url_encode(&device_id));
        self.post(&path, &serde_json::json!({})).await
    }

    pub async fn delete_device(&self, device_id: String) -> Result<serde_json::Value, WalletError> {
        let path = format!("/devices/{}", url_encode(&device_id));
        self.delete(&path).await
    }

    // ---- Approvals (token allowances) ----

    pub async fn get_approvals(
        &self,
        address: String,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        self.get_query("/approvals", &[
            ("address", address),
            ("chain_id", chain_id.to_string()),
        ]).await
    }

    pub async fn revoke_approval(&self, approval_id: String) -> Result<serde_json::Value, WalletError> {
        let path = format!("/approvals/{}", url_encode(&approval_id));
        self.delete(&path).await
    }

    // ---- Keystore ----

    pub async fn export_keystore(
        &self,
        wallet_id: String,
        password: String,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req { wallet_id: String, password: String }
        self.post("/keystore/export", &Req { wallet_id, password }).await
    }

    pub async fn import_keystore(
        &self,
        keystore: String,
        password: String,
        label: Option<String>,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            keystore: String,
            password: String,
            #[serde(skip_serializing_if = "Option::is_none")]
            label: Option<String>,
        }
        self.post("/keystore/import", &Req { keystore, password, label }).await
    }

    // ---- Security ----

    pub async fn check_url(&self, url: String) -> Result<serde_json::Value, WalletError> {
        self.get_query("/security/check-url", &[("url", url)]).await
    }

    pub async fn check_address(&self, address: String) -> Result<serde_json::Value, WalletError> {
        self.get_query("/security/check-address", &[("address", address)]).await
    }

    pub async fn security_scan(&self, target: String) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req { target: String }
        self.post("/security/scan", &Req { target }).await
    }

    // ---- Lending ----

    pub async fn get_lending_markets(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/lending/markets").await
    }

    pub async fn get_lending_positions(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/lending/positions").await
    }

    pub async fn lending_supply(
        &self,
        wallet_id: String,
        password: String,
        asset: String,
        amount: String,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            wallet_id: String,
            password: String,
            asset: String,
            amount: String,
            chain_id: i64,
        }
        self.post("/lending/supply", &Req { wallet_id, password, asset, amount, chain_id }).await
    }

    pub async fn lending_borrow(
        &self,
        wallet_id: String,
        password: String,
        asset: String,
        amount: String,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            wallet_id: String,
            password: String,
            asset: String,
            amount: String,
            chain_id: i64,
        }
        self.post("/lending/borrow", &Req { wallet_id, password, asset, amount, chain_id }).await
    }

    pub async fn lending_withdraw(
        &self,
        wallet_id: String,
        password: String,
        asset: String,
        amount: String,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            wallet_id: String,
            password: String,
            asset: String,
            amount: String,
            chain_id: i64,
        }
        self.post("/lending/withdraw", &Req { wallet_id, password, asset, amount, chain_id }).await
    }

    pub async fn lending_repay(
        &self,
        wallet_id: String,
        password: String,
        asset: String,
        amount: String,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            wallet_id: String,
            password: String,
            asset: String,
            amount: String,
            chain_id: i64,
        }
        self.post("/lending/repay", &Req { wallet_id, password, asset, amount, chain_id }).await
    }

    // ---- Copy trading ----

    pub async fn get_copy_traders(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/copytrading/traders").await
    }

    pub async fn follow_trader(
        &self,
        trader_id: String,
        allocation: Option<String>,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            trader_id: String,
            #[serde(skip_serializing_if = "Option::is_none")]
            allocation: Option<String>,
        }
        self.post("/copytrading/follow", &Req { trader_id, allocation }).await
    }

    pub async fn stop_copy_trader(&self, copier_id: String) -> Result<serde_json::Value, WalletError> {
        let path = format!("/copytrading/copiers/{}/stop", url_encode(&copier_id));
        self.post(&path, &serde_json::json!({})).await
    }

    pub async fn get_copy_signals(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/copytrading/signals").await
    }

    // ---- DAO ----

    pub async fn get_dao_proposals(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/dao/proposals").await
    }

    pub async fn create_dao_proposal(
        &self,
        title: String,
        description: String,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req { title: String, description: String }
        self.post("/dao/proposals", &Req { title, description }).await
    }

    pub async fn vote_dao_proposal(
        &self,
        proposal_id: String,
        support: bool,
    ) -> Result<serde_json::Value, WalletError> {
        let path = format!("/dao/proposals/{}/vote", url_encode(&proposal_id));
        self.post(&path, &serde_json::json!({ "support": support })).await
    }

    pub async fn get_dao_delegates(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/dao/delegates").await
    }

    // ---- Perpetuals ----

    pub async fn get_perpetual_positions(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/perpetual/positions").await
    }

    pub async fn create_perpetual_position(
        &self,
        pair: String,
        side: String,
        size: String,
        leverage: i64,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            pair: String,
            side: String,
            size: String,
            leverage: i64,
            chain_id: i64,
        }
        self.post("/perpetual/positions", &Req { pair, side, size, leverage, chain_id }).await
    }

    pub async fn close_perpetual_position(&self, position_id: String) -> Result<serde_json::Value, WalletError> {
        let path = format!("/perpetual/positions/{}/close", url_encode(&position_id));
        self.post(&path, &serde_json::json!({})).await
    }

    // ---- Margin ----

    pub async fn get_margin_positions(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/margin/positions").await
    }

    pub async fn create_margin_position(
        &self,
        pair: String,
        side: String,
        size: String,
        leverage: i64,
        chain_id: i64,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            pair: String,
            side: String,
            size: String,
            leverage: i64,
            chain_id: i64,
        }
        self.post("/margin/positions", &Req { pair, side, size, leverage, chain_id }).await
    }

    pub async fn close_margin_position(&self, position_id: String) -> Result<serde_json::Value, WalletError> {
        let path = format!("/margin/positions/{}/close", url_encode(&position_id));
        self.post(&path, &serde_json::json!({})).await
    }

    // ---- Prediction markets ----

    pub async fn get_prediction_markets(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/prediction/markets").await
    }

    pub async fn place_prediction_bet(
        &self,
        market_id: String,
        side: String,
        amount: String,
    ) -> Result<serde_json::Value, WalletError> {
        let path = format!("/prediction/markets/{}/bet", url_encode(&market_id));
        self.post(&path, &serde_json::json!({ "side": side, "amount": amount })).await
    }

    // ---- Launchpool ----

    pub async fn get_launchpool(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/launchpool").await
    }

    pub async fn get_launchpool_stakes(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/launchpool/stakes").await
    }

    pub async fn launchpool_stake(
        &self,
        wallet_id: String,
        password: String,
        amount: String,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req { wallet_id: String, password: String, amount: String }
        self.post("/launchpool/stake", &Req { wallet_id, password, amount }).await
    }

    pub async fn launchpool_unstake(
        &self,
        wallet_id: String,
        password: String,
        amount: String,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req { wallet_id: String, password: String, amount: String }
        self.post("/launchpool/unstake", &Req { wallet_id, password, amount }).await
    }

    // ---- Token sales ----

    pub async fn get_token_sales(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/token-sales").await
    }

    pub async fn participate_token_sale(
        &self,
        sale_id: String,
        amount: String,
    ) -> Result<serde_json::Value, WalletError> {
        let path = format!("/token-sales/{}/participate", url_encode(&sale_id));
        self.post(&path, &serde_json::json!({ "amount": amount })).await
    }

    // ---- DApps ----

    pub async fn get_dapps(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/dapps").await
    }

    pub async fn get_dapp_categories(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/dapps/categories").await
    }

    // ---- Charts / DeFi / aliases ----

    pub async fn get_chart_history(
        &self,
        token: String,
        days: Option<i64>,
    ) -> Result<serde_json::Value, WalletError> {
        let mut q = vec![("token", token)];
        if let Some(d) = days {
            q.push(("days", d.to_string()));
        }
        self.get_query("/chart/history", &q).await
    }

    pub async fn get_defi_protocols(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/defi/protocols").await
    }

    // ---- Token registry + trading terminal (public) ----

    /// GET /tokens/registry — canonical per-chain token asset registry.
    pub async fn get_token_registry(&self, chain_id: Option<i64>) -> Result<serde_json::Value, WalletError> {
        match chain_id {
            Some(id) => self.get_query("/tokens/registry", &[("chain_id", id.to_string())]).await,
            None => self.get("/tokens/registry").await,
        }
    }

    /// GET /terminal/kline/:symbol — real OHLC candles (CoinGecko-backed).
    pub async fn get_terminal_kline(&self, symbol: &str, days: u32) -> Result<serde_json::Value, WalletError> {
        let days = if days == 0 { 1 } else { days };
        self.get_query(&format!("/terminal/kline/{}", url_encode(symbol)), &[("days", days.to_string())]).await
    }

    /// GET /terminal/ticker/:symbol — real 24h ticker (CoinGecko-backed).
    pub async fn get_terminal_ticker(&self, symbol: &str) -> Result<serde_json::Value, WalletError> {
        self.get(&format!("/terminal/ticker/{}", url_encode(symbol))).await
    }

    pub async fn get_networks(&self) -> Result<Vec<ChainInfo>, WalletError> {
        self.get_chains().await
    }

    pub async fn get_price(&self, coin: String) -> Result<PriceInfo, WalletError> {
        self.get_token_price(&coin).await
    }

    // ---- Bridge (proxied bridge_service :8007) ----

    pub async fn get_bridges(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/api/v1/bridge/routes").await
    }

    pub async fn get_bridge_quote(
        &self,
        from_chain: i64,
        to_chain: i64,
        token: String,
        amount: String,
    ) -> Result<serde_json::Value, WalletError> {
        #[derive(Serialize)]
        struct Req {
            #[serde(rename = "fromChain")]
            from_chain: i64,
            #[serde(rename = "toChain")]
            to_chain: i64,
            token: String,
            amount: String,
        }
        let req = Req { from_chain, to_chain, token, amount };
        self.post("/api/v1/bridge/quote", &req).await
    }

    pub async fn initiate_bridge_transfer(
        &self,
        body: serde_json::Value,
    ) -> Result<serde_json::Value, WalletError> {
        self.post("/api/v1/bridge/transfer", &body).await
    }

    pub async fn get_bridge_tx_status(
        &self,
        tx_id: String,
    ) -> Result<serde_json::Value, WalletError> {
        let path = format!("/api/v1/bridge/tx/{}", url_encode(&tx_id));
        self.get(&path).await
    }

    pub async fn get_bridge_history(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/api/v1/bridge/history").await
    }

    // ---- dApp browser / WalletConnect (proxied dapp_browser :8083) ----

    pub async fn get_dapp_pairings(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/api/v1/dapp/pairings").await
    }

    pub async fn create_dapp_pairing(
        &self,
        body: serde_json::Value,
    ) -> Result<serde_json::Value, WalletError> {
        self.post("/api/v1/dapp/pairings", &body).await
    }

    pub async fn approve_dapp_pairing(
        &self,
        topic: String,
    ) -> Result<serde_json::Value, WalletError> {
        let path = format!("/api/v1/dapp/pairings/{}/approve", url_encode(&topic));
        self.post(&path, &serde_json::json!({})).await
    }

    pub async fn reject_dapp_pairing(
        &self,
        topic: String,
    ) -> Result<serde_json::Value, WalletError> {
        let path = format!("/api/v1/dapp/pairings/{}/reject", url_encode(&topic));
        self.post(&path, &serde_json::json!({})).await
    }

    pub async fn get_dapp_sessions(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/api/v1/dapp/sessions").await
    }

    pub async fn send_dapp_request(
        &self,
        topic: String,
        body: serde_json::Value,
    ) -> Result<serde_json::Value, WalletError> {
        let path = format!("/api/v1/dapp/sessions/{}/request", url_encode(&topic));
        self.post(&path, &body).await
    }

    pub async fn get_dapp_requests(
        &self,
        topic: String,
    ) -> Result<serde_json::Value, WalletError> {
        let path = format!("/api/v1/dapp/sessions/{}/request", url_encode(&topic));
        self.get(&path).await
    }

    pub async fn respond_to_dapp_request(
        &self,
        topic: String,
        request_id: String,
        body: serde_json::Value,
    ) -> Result<serde_json::Value, WalletError> {
        let path = format!(
            "/api/v1/dapp/sessions/{}/request/{}/respond",
            url_encode(&topic),
            url_encode(&request_id)
        );
        self.post(&path, &body).await
    }

    // ------------------------------------------------------------------
    // Price alerts (authenticated): POST/GET/PUT/DELETE /price-alerts
    // ------------------------------------------------------------------

    pub async fn list_price_alerts(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/price-alerts").await
    }

    pub async fn create_price_alert(
        &self,
        symbol: &str,
        target_price: &str,
        direction: &str,
    ) -> Result<serde_json::Value, WalletError> {
        self.post(
            "/price-alerts",
            &serde_json::json!({
                "symbol": symbol,
                "target_price": target_price,
                "direction": direction,
            }),
        )
        .await
    }

    pub async fn update_price_alert(
        &self,
        id: &str,
        body: serde_json::Value,
    ) -> Result<serde_json::Value, WalletError> {
        self.put(&format!("/price-alerts/{}", url_encode(id)), &body).await
    }

    pub async fn delete_price_alert(&self, id: &str) -> Result<serde_json::Value, WalletError> {
        self.delete(&format!("/price-alerts/{}", url_encode(id))).await
    }

    // ------------------------------------------------------------------
    // Transaction simulation (public): POST /simulate
    // ------------------------------------------------------------------

    pub async fn simulate_transaction(
        &self,
        chain_id: i64,
        from: &str,
        to: &str,
        value: Option<&str>,
        data: Option<&str>,
    ) -> Result<serde_json::Value, WalletError> {
        let mut body = serde_json::json!({
            "chain_id": chain_id,
            "from": from,
            "to": to,
        });
        if let Some(v) = value {
            body["value"] = serde_json::json!(v);
        }
        if let Some(d) = data {
            body["data"] = serde_json::json!(d);
        }
        self.post("/simulate", &body).await
    }

    // ------------------------------------------------------------------
    // Watch-only wallets (authenticated): POST /wallets/watch-only
    // ------------------------------------------------------------------

    pub async fn create_watch_only_wallet(
        &self,
        address: &str,
        chain_id: i64,
        label: &str,
    ) -> Result<serde_json::Value, WalletError> {
        self.post(
            "/wallets/watch-only",
            &serde_json::json!({
                "address": address,
                "chain_id": chain_id,
                "label": label,
            }),
        )
        .await
    }

    // ------------------------------------------------------------------
    // Fee transparency: GET /fees (authenticated) + GET /public/fees{,/transactions} (public)
    // ------------------------------------------------------------------

    pub async fn get_fees(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/fees").await
    }

    pub async fn get_public_fees(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/public/fees").await
    }

    pub async fn get_public_fee_transactions(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/public/fees/transactions").await
    }

    // ---------------- Wallet & finance plane ----------------

    /// GET /finance/accounts — multi-chain ledger accounts.
    pub async fn get_finance_accounts(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/finance/accounts").await
    }

    /// GET /finance/history — full double-entry ledger history.
    pub async fn get_finance_history(&self, currency: Option<&str>) -> Result<serde_json::Value, WalletError> {
        let path = match currency {
            Some(c) => format!("/finance/history?currency={c}"),
            None => "/finance/history".to_string(),
        };
        self.get(&path).await
    }

    /// GET /finance/switches — per-token feature switches.
    pub async fn get_finance_switches(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/finance/switches").await
    }

    /// GET /finance/deposit-addresses — deterministic per-user deposit addresses.
    pub async fn get_deposit_addresses(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/finance/deposit-addresses").await
    }

    /// POST /finance/withdrawals — risk-scored, HMAC-signed withdrawal request.
    pub async fn create_withdrawal(
        &self,
        currency: &str,
        amount: &str,
        to_address: &str,
    ) -> Result<serde_json::Value, WalletError> {
        self.post(
            "/finance/withdrawals",
            &serde_json::json!({"currency": currency, "amount": amount, "to_address": to_address}),
        )
        .await
    }

    /// GET /finance/withdrawals — the caller's withdrawal requests.
    pub async fn get_withdrawals(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/finance/withdrawals").await
    }

    /// GET /finance/convert/rates — admin-managed rate book.
    pub async fn get_convert_rates(&self) -> Result<serde_json::Value, WalletError> {
        self.get("/finance/convert/rates").await
    }

    /// POST /finance/convert — instant conversion at the admin rate.
    pub async fn finance_convert(
        &self,
        from_currency: &str,
        to_currency: &str,
        amount: &str,
    ) -> Result<serde_json::Value, WalletError> {
        self.post(
            "/finance/convert",
            &serde_json::json!({"from_currency": from_currency, "to_currency": to_currency, "amount": amount}),
        )
        .await
    }

    /// POST /finance/transfer — atomic KYC-gated internal transfer.
    pub async fn finance_transfer(
        &self,
        to_email: &str,
        currency: &str,
        amount: &str,
    ) -> Result<serde_json::Value, WalletError> {
        self.post(
            "/finance/transfer",
            &serde_json::json!({"to_email": to_email, "currency": currency, "amount": amount}),
        )
        .await
    }

    /// GET /finance/payment-methods — 881-method / 238-country catalog.
    pub async fn get_payment_methods(
        &self,
        country: Option<&str>,
        kind: Option<&str>,
    ) -> Result<serde_json::Value, WalletError> {
        let mut parts: Vec<String> = Vec::new();
        if let Some(c) = country {
            parts.push(format!("country={c}"));
        }
        if let Some(k) = kind {
            parts.push(format!("kind={k}"));
        }
        let path = if parts.is_empty() {
            "/finance/payment-methods".to_string()
        } else {
            format!("/finance/payment-methods?{}", parts.join("&"))
        };
        self.get(&path).await
    }

    /// GET /finance/p2p/escrow — escrow marketplace (or the caller's orders).
    pub async fn get_escrow_orders(&self, mine: bool) -> Result<serde_json::Value, WalletError> {
        self.get(if mine { "/finance/p2p/escrow?mine=true" } else { "/finance/p2p/escrow" }).await
    }

    /// POST /finance/p2p/escrow — open a sell order (funds locked, KYC-gated).
    pub async fn open_escrow(
        &self,
        currency: &str,
        amount: &str,
        fiat_currency: &str,
        fiat_amount: &str,
        payment_method_code: &str,
        country_code: &str,
    ) -> Result<serde_json::Value, WalletError> {
        self.post(
            "/finance/p2p/escrow",
            &serde_json::json!({
                "currency": currency,
                "amount": amount,
                "fiat_currency": fiat_currency,
                "fiat_amount": fiat_amount,
                "payment_method_code": payment_method_code,
                "country_code": country_code,
            }),
        )
        .await
    }

    /// POST /finance/p2p/escrow/:id/:action — accept/paid/release/dispute/cancel.
    pub async fn escrow_action(
        &self,
        id: &str,
        action: &str,
        reason: Option<&str>,
    ) -> Result<serde_json::Value, WalletError> {
        let body = match reason {
            Some(r) => serde_json::json!({"reason": r}),
            None => serde_json::json!({}),
        };
        self.post(&format!("/finance/p2p/escrow/{id}/{action}"), &body).await
    }
}

// ---------------------------------------------------------------------------
// QR parsing helpers (no camera dependency; clients wire their own scanner)
// ---------------------------------------------------------------------------

/// ParsedPayment holds a decoded address + optional amount from a scanned QR
/// payload (bare address, EIP-681 payment URI, or `ethereum:` URI).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ParsedPayment {
    pub address: String,
    pub amount: Option<String>,
    pub chain_id: Option<i64>,
    pub token_address: Option<String>,
}

/// parse_payment_uri decodes a scanned QR string into a ParsedPayment. It
/// handles bare 0x addresses, `ethereum:<addr>` URIs, and EIP-681 payment
/// URIs (`ethereum:0x.../transfer?address=...&uint256=...?value=...`). Returns
/// None if no address can be extracted (fail-closed — never a guessed address).
pub fn parse_payment_uri(input: &str) -> Option<ParsedPayment> {
    let s = input.trim();
    if s.is_empty() {
        return None;
    }
    // Bare 0x address.
    if s.starts_with("0x") && s.len() == 42 {
        return Some(ParsedPayment { address: s.to_string(), amount: None, chain_id: None, token_address: None });
    }
    // ethereum: URI (strip the scheme).
    let body = if let Some(rest) = s.strip_prefix("ethereum:") {
        rest
    } else {
        return None;
    };
    // Split target from query.
    let (target, query) = match body.find('?') {
        Some(i) => (&body[..i], &body[i + 1..]),
        None => (body, ""),
    };
    // target may be "0xaddr" or "0xaddr/transfer" (EIP-681 token transfer).
    let (address, token_address) = if let Some((addr, func)) = target.split_once('/') {
        (addr.to_string(), if func.starts_with("transfer") { Some(String::new()) } else { None })
    } else {
        (target.to_string(), None)
    };
    if !address.starts_with("0x") || address.len() != 42 {
        return None;
    }
    let mut amount = None;
    let mut chain_id = None;
    let mut token_contract = token_address;
    for pair in query.split('&') {
        let (k, v) = match pair.split_once('=') { Some(kv) => kv, None => continue };
        match k {
            "value" => amount = Some(v.to_string()),
            "chainId" => chain_id = v.parse().ok(),
            "address" if token_contract.is_some() => token_contract = Some(v.to_string()),
            "uint256" => {} // token amount; clients map to amount separately
            _ => {}
        }
    }
    Some(ParsedPayment { address, amount, chain_id, token_address: token_contract.filter(|s| !s.is_empty()) })
}

// Minimal base64 (standard alphabet, with padding) decoder and a percent-encoder
// for path/query parameters. These avoid pulling extra crates for the few spots
// that need them (JWT payload decode + path-param encoding).

fn base64_decode(input: &str) -> Result<Vec<u8>, String> {
    const TABLE: [i8; 256] = {
        let mut t = [-1i8; 256];
        let mut i = 0u8;
        while i < 64 {
            let ch = match i {
                0..=25 => b'A' + i,
                26..=51 => b'a' + (i - 26),
                52..=61 => b'0' + (i - 52),
                62 => b'+',
                63 => b'/',
                _ => 0,
            };
            t[ch as usize] = i as i8;
            i += 1;
        }
        t
    };
    let bytes: Vec<u8> = input.bytes().filter(|b| *b != b'=').collect();
    let mut out = Vec::with_capacity(bytes.len() * 3 / 4);
    for chunk in bytes.chunks(4) {
        let vals: Vec<i8> = chunk.iter().map(|b| TABLE[*b as usize]).collect();
        if vals.iter().any(|v| *v < 0) {
            return Err("invalid base64".into());
        }
        let n = vals.len();
        let b0 = vals[0] as u32;
        let b1 = vals[1] as u32;
        out.push(((b0 << 2) | (b1 >> 4)) as u8);
        if n > 2 {
            let b2 = vals[2] as u32;
            out.push((((b1 & 0x0f) << 4) | (b2 >> 2)) as u8);
            if n > 3 {
                let b3 = vals[3] as u32;
                out.push((((b2 & 0x03) << 6) | b3) as u8);
            }
        }
    }
    Ok(out)
}

/// url_encode percent-encodes a path segment per RFC 3986 (unreserved set only).
pub(crate) fn url_encode(input: &str) -> String {
    let mut out = String::with_capacity(input.len());
    for b in input.bytes() {
        if b.is_ascii_alphanumeric() || matches!(b, b'-' | b'_' | b'.' | b'~') {
            out.push(b as char);
        } else {
            out.push_str(&format!("%{:02X}", b));
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_bare_address() {
        let p = parse_payment_uri("0x9858EfFD232B4033E47d90003D41EC34EcaEda94").unwrap();
        assert_eq!(p.address, "0x9858EfFD232B4033E47d90003D41EC34EcaEda94");
        assert!(p.amount.is_none());
    }

    #[test]
    fn test_parse_eip681_payment() {
        let p = parse_payment_uri("ethereum:0x9858EfFD232B4033E47d90003D41EC34EcaEda94?value=1.5&chainId=1").unwrap();
        assert_eq!(p.address, "0x9858EfFD232B4033E47d90003D41EC34EcaEda94");
        assert_eq!(p.amount.as_deref(), Some("1.5"));
        assert_eq!(p.chain_id, Some(1));
    }

    #[test]
    fn test_parse_ethereum_uri_no_value() {
        let p = parse_payment_uri("ethereum:0x9858EfFD232B4033E47d90003D41EC34EcaEda94").unwrap();
        assert_eq!(p.address, "0x9858EfFD232B4033E47d90003D41EC34EcaEda94");
        assert!(p.amount.is_none());
    }

    #[test]
    fn test_parse_invalid_returns_none() {
        assert!(parse_payment_uri("").is_none());
        assert!(parse_payment_uri("not an address").is_none());
        assert!(parse_payment_uri("0xshort").is_none());
    }

    #[test]
    fn test_parse_token_transfer_uri() {
        let p = parse_payment_uri("ethereum:0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48/transfer?address=0x9858EfFD232B4033E47d90003D41EC34EcaEda94&uint256=1000000").unwrap();
        assert_eq!(p.address, "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48");
        assert!(p.token_address.is_some());
    }

    #[test]
    fn test_wallet_error_display() {
        let e = WalletError::Api("bad".into());
        assert!(format!("{e}").contains("bad"));
    }
}
