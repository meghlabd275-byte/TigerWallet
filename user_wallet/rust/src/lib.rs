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
    pub block_number: i64,
    pub connected: bool,
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

    fn url(&self, path: &str) -> String {
        format!("{}{}", self.base_url, path)
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
        }
        self.post("/api/v1/send", &Req {
            wallet_id, password, to, amount, chain_id, token_address,
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
        }
        let path = match master_wallet_id {
            Some(mw) => format!("/api/v1/auto-send?master_wallet_id={}", mw),
            None => "/api/v1/auto-send".to_string(),
        };
        self.post(&path, &Req {
            wallet_id, password, to, amount, chain_id, token_address,
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

    /// get_network_status derives a real connection status from the chains
    /// registry (connected = chain present) and an honest block_number of 0
    /// when no dedicated status endpoint exists — never a fabricated height.
    pub async fn get_network_status(&self, chain_id: i64) -> Result<NetworkStatus, WalletError> {
        let chains = self.get_chains().await?;
        let chain = chains.into_iter().find(|c| c.id == chain_id);
        Ok(NetworkStatus {
            chain_id,
            block_number: 0,
            connected: chain.is_some(),
        })
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
        self.get("/api/v1/p2p/listings").await
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
