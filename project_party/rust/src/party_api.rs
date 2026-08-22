//! ProjectParty Rust client.
//!
//! Async reqwest client delegating to the standalone ProjectParty backend
//! (`project_party/go/cmd/main.go`, port 8106, path prefix `/api/v1`, JWT auth
//! + RBAC). The token is held in-process (`Arc<RwLock<Option<String>>>`) — Rust
//! clients are server/CLI and have no keychain dependency by default; callers
//! that need persistence can wrap `set_token`/`token`.
//!
//! Every method issues a real reqwest call against the backend — no stubs,
//! fakes, or mock data. On any non-2xx response the method returns an `Err`
//! carrying the backend's `error` message (fail-closed); it never returns
//! fabricated data.
//!
//! Method set matches `project_party/web/src/services/api.ts` + the discovery,
//! pricing, analytics, compliance routes the task requires.

use reqwest::{Client, Method};
use serde_json::Value;
use std::sync::{Arc, RwLock};
use std::time::Duration;

pub const PARTY_API_DEFAULT_URL: &str = "http://localhost:8106/api/v1";

#[derive(Debug)]
pub enum PartyError {
    Network(String),
    Decode(String),
    Http { status: u16, message: String },
    Unauthorized,
}

impl std::fmt::Display for PartyError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            PartyError::Network(m) => write!(f, "network: {m}"),
            PartyError::Decode(m) => write!(f, "decode: {m}"),
            PartyError::Http { status, message } => write!(f, "http {status}: {message}"),
            PartyError::Unauthorized => write!(f, "not authenticated: no JWT token set"),
        }
    }
}

impl std::error::Error for PartyError {}

pub struct PartyClient {
    http: Client,
    base_url: String,
    token: Arc<RwLock<Option<String>>>,
}

impl PartyClient {
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
            token: Arc::new(RwLock::new(token)),
        }
    }

    pub fn default() -> Self {
        Self::new(PARTY_API_DEFAULT_URL, None)
    }

    pub fn set_token(&self, token: Option<String>) {
        *self.token.write().unwrap() = token;
    }

    pub fn token(&self) -> Option<String> {
        self.token.read().unwrap().clone()
    }

    pub fn base_url(&self) -> &str {
        &self.base_url
    }

    // -- internal request helpers --

    fn url(&self, path: &str, absolute: bool) -> String {
        if absolute {
            let base = self.base_url.trim_end_matches("/api/v1");
            format!("{}{}", base, path)
        } else {
            format!("{}{}", self.base_url, path)
        }
    }

    async fn request(
        &self,
        method: Method,
        path: &str,
        body: Option<Value>,
        authenticated: bool,
        absolute: bool,
    ) -> Result<Value, PartyError> {
        let url = self.url(path, absolute);
        let mut req = self.http.request(method, &url);
        if authenticated {
            if let Some(t) = self.token() {
                req = req.bearer_auth(t);
            } else {
                return Err(PartyError::Unauthorized);
            }
        }
        if let Some(b) = body {
            req = req.json(&b);
        }
        let resp = req.send().await.map_err(|e| PartyError::Network(e.to_string()))?;
        let status = resp.status();
        let bytes = resp
            .bytes()
            .await
            .map_err(|e| PartyError::Network(e.to_string()))?;
        let text = String::from_utf8_lossy(&bytes);
        let value: Value = if text.trim().is_empty() {
            Value::Null
        } else {
            serde_json::from_str(&text).map_err(|e| PartyError::Decode(e.to_string()))?
        };
        if !status.is_success() {
            let msg = value
                .get("error")
                .or_else(|| value.get("message"))
                .and_then(|v| v.as_str())
                .map(|s| s.to_string())
                .unwrap_or_else(|| {
                    if text.trim().is_empty() {
                        format!("HTTP {}", status.as_u16())
                    } else {
                        text.to_string()
                    }
                });
            return Err(PartyError::Http {
                status: status.as_u16(),
                message: msg,
            });
        }
        Ok(value)
    }

    async fn get(&self, path: &str, authenticated: bool) -> Result<Value, PartyError> {
        self.request(Method::GET, path, None, authenticated, false).await
    }

    async fn get_absolute(&self, path: &str) -> Result<Value, PartyError> {
        self.request(Method::GET, path, None, false, true).await
    }

    async fn send_body(
        &self,
        method: Method,
        path: &str,
        body: Value,
    ) -> Result<Value, PartyError> {
        self.request(method, path, Some(body), true, false).await
    }

    async fn post(&self, path: &str, body: Value) -> Result<Value, PartyError> {
        self.send_body(Method::POST, path, body).await
    }

    async fn put(&self, path: &str, body: Value) -> Result<Value, PartyError> {
        self.send_body(Method::PUT, path, body).await
    }

    async fn delete(&self, path: &str) -> Result<Value, PartyError> {
        self.request(Method::DELETE, path, None, true, false).await
    }

    async fn post_no_body(&self, path: &str) -> Result<Value, PartyError> {
        self.request(Method::POST, path, None, true, false).await
    }

    // -- Health --

    pub async fn get_health(&self) -> Result<Value, PartyError> {
        self.get_absolute("/health").await
    }

    // -- Auth --

    pub async fn register(&self, username: &str, password: &str) -> Result<Value, PartyError> {
        let payload = serde_json::json!({ "username": username, "password": password });
        let res = self
            .request(Method::POST, "/auth/register", Some(payload), false, false)
            .await?;
        if let Some(t) = res.get("token").and_then(|v| v.as_str()) {
            self.set_token(Some(t.to_string()));
        }
        Ok(res)
    }

    pub async fn login(&self, username: &str, password: &str) -> Result<Value, PartyError> {
        let payload = serde_json::json!({ "username": username, "password": password });
        let res = self
            .request(Method::POST, "/auth/login", Some(payload), false, false)
            .await?;
        if let Some(t) = res.get("token").and_then(|v| v.as_str()) {
            self.set_token(Some(t.to_string()));
        }
        Ok(res)
    }

    // -- Discovery (public) --

    pub async fn get_coins(&self) -> Result<Value, PartyError> {
        self.get("/coins", false).await
    }

    pub async fn search_tokens(&self, q: &str) -> Result<Value, PartyError> {
        self.get(&format!("/search?q={}", encode(q)), false).await
    }

    pub async fn get_featured(&self) -> Result<Value, PartyError> {
        self.get("/featured", false).await
    }

    pub async fn get_trending(&self) -> Result<Value, PartyError> {
        self.get("/trending", false).await
    }

    pub async fn get_market(&self) -> Result<Value, PartyError> {
        self.get("/market", false).await
    }

    // -- Tokens --

    pub async fn list_tokens(&self, status: Option<&str>) -> Result<Value, PartyError> {
        let path = match status {
            Some(s) => format!("/tokens?status={}", encode(s)),
            None => "/tokens".to_string(),
        };
        self.get(&path, true).await
    }

    pub async fn get_token(&self, id: &str) -> Result<Value, PartyError> {
        self.get(&format!("/tokens/{}", encode(id)), true).await
    }

    pub async fn create_token(&self, data: Value) -> Result<Value, PartyError> {
        self.post("/tokens", data).await
    }

    pub async fn update_token(&self, id: &str, data: Value) -> Result<Value, PartyError> {
        self.put(&format!("/tokens/{}", encode(id)), data).await
    }

    pub async fn delete_token(&self, id: &str) -> Result<Value, PartyError> {
        self.delete(&format!("/tokens/{}", encode(id))).await
    }

    pub async fn submit_token(&self, id: &str) -> Result<Value, PartyError> {
        self.post_no_body(&format!("/tokens/{}/submit", encode(id))).await
    }

    pub async fn approve_token(&self, id: &str) -> Result<Value, PartyError> {
        self.post_no_body(&format!("/tokens/{}/approve", encode(id))).await
    }

    pub async fn reject_token(&self, id: &str) -> Result<Value, PartyError> {
        self.post_no_body(&format!("/tokens/{}/reject", encode(id))).await
    }

    // -- Listings --

    pub async fn list_listings(&self, status: Option<&str>) -> Result<Value, PartyError> {
        let path = match status {
            Some(s) => format!("/listings?status={}", encode(s)),
            None => "/listings".to_string(),
        };
        self.get(&path, true).await
    }

    pub async fn get_listing(&self, id: &str) -> Result<Value, PartyError> {
        self.get(&format!("/listings/{}", encode(id)), true).await
    }

    pub async fn create_listing(&self, data: Value) -> Result<Value, PartyError> {
        self.post("/listings", data).await
    }

    pub async fn update_listing_status(&self, id: &str, status: &str) -> Result<Value, PartyError> {
        self.put(
            &format!("/listings/{}/status", encode(id)),
            serde_json::json!({ "status": status }),
        )
        .await
    }

    pub async fn feature_listing(&self, id: &str) -> Result<Value, PartyError> {
        self.post_no_body(&format!("/listings/{}/featured", encode(id))).await
    }

    // -- Launchpad --

    pub async fn list_launchpads(&self, status: Option<&str>) -> Result<Value, PartyError> {
        let path = match status {
            Some(s) => format!("/launchpad?status={}", encode(s)),
            None => "/launchpad".to_string(),
        };
        self.get(&path, true).await
    }

    pub async fn get_launchpad(&self, id: &str) -> Result<Value, PartyError> {
        self.get(&format!("/launchpad/{}", encode(id)), true).await
    }

    pub async fn create_launchpad(&self, data: Value) -> Result<Value, PartyError> {
        self.post("/launchpad/create", data).await
    }

    pub async fn contribute(&self, id: &str, amount: &str) -> Result<Value, PartyError> {
        self.post(
            &format!("/launchpad/{}/contribute", encode(id)),
            serde_json::json!({ "amount": amount }),
        )
        .await
    }

    pub async fn claim_tokens(&self, id: &str) -> Result<Value, PartyError> {
        self.post_no_body(&format!("/launchpad/{}/claim", encode(id))).await
    }

    pub async fn cancel_launchpad(&self, id: &str) -> Result<Value, PartyError> {
        self.post_no_body(&format!("/launchpad/{}/cancel", encode(id))).await
    }

    // -- Market-making --

    pub async fn get_maker_orders(&self, token_id: Option<&str>) -> Result<Value, PartyError> {
        let path = match token_id {
            Some(t) => format!("/market-making/orders?token_id={}", encode(t)),
            None => "/market-making/orders".to_string(),
        };
        self.get(&path, true).await
    }

    pub async fn get_market_maker_status(&self, token_id: &str) -> Result<Value, PartyError> {
        self.get(&format!("/market-making/status/{}", encode(token_id)), true)
            .await
    }

    pub async fn create_maker_orders(&self, data: Value) -> Result<Value, PartyError> {
        self.post("/market-making/orders", data).await
    }

    pub async fn update_order_status(&self, id: &str, status: &str) -> Result<Value, PartyError> {
        self.put(
            &format!("/market-making/orders/{}/status", encode(id)),
            serde_json::json!({ "status": status }),
        )
        .await
    }

    pub async fn add_liquidity(&self, data: Value) -> Result<Value, PartyError> {
        self.post("/market-making/liquidity/add", data).await
    }

    pub async fn remove_liquidity(&self, data: Value) -> Result<Value, PartyError> {
        self.post("/market-making/liquidity/remove", data).await
    }

    // -- Pricing --

    pub async fn get_token_price(&self, token_id: &str) -> Result<Value, PartyError> {
        self.get(&format!("/pricing/{}", encode(token_id)), true).await
    }

    pub async fn get_price_history(&self, token_id: &str) -> Result<Value, PartyError> {
        self.get(&format!("/pricing/history/{}", encode(token_id)), true)
            .await
    }

    pub async fn set_token_price(&self, token_id: &str, price: &str) -> Result<Value, PartyError> {
        self.post(
            "/pricing/set",
            serde_json::json!({ "token_id": token_id, "price": price }),
        )
        .await
    }

    pub async fn update_price(&self, token_id: &str, price: &str) -> Result<Value, PartyError> {
        self.post(
            "/pricing/update",
            serde_json::json!({ "token_id": token_id, "price": price }),
        )
        .await
    }

    // -- Analytics (public) --

    pub async fn get_trading_volume(&self) -> Result<Value, PartyError> {
        self.get("/analytics/volume", false).await
    }

    pub async fn get_liquidity(&self) -> Result<Value, PartyError> {
        self.get("/analytics/liquidity", false).await
    }

    pub async fn get_holder_count(&self) -> Result<Value, PartyError> {
        self.get("/analytics/holders", false).await
    }

    pub async fn get_transaction_count(&self) -> Result<Value, PartyError> {
        self.get("/analytics/transactions", false).await
    }

    // -- Compliance --

    pub async fn get_audit_status(&self, token_id: &str) -> Result<Value, PartyError> {
        self.get(&format!("/compliance/audit/{}", encode(token_id)), true)
            .await
    }

    pub async fn get_kyc_status(&self, token_id: &str) -> Result<Value, PartyError> {
        self.get(&format!("/compliance/kyc/{}", encode(token_id)), true)
            .await
    }

    pub async fn request_audit(&self, data: Value) -> Result<Value, PartyError> {
        self.post("/compliance/audit", data).await
    }

    pub async fn submit_kyc(&self, data: Value) -> Result<Value, PartyError> {
        self.post("/compliance/kyc/submit", data).await
    }

    // -- Fees --

    pub async fn get_listing_fees(&self) -> Result<Value, PartyError> {
        self.get("/fees", false).await
    }

    pub async fn calculate_fees(&self, data: Value) -> Result<Value, PartyError> {
        self.post("/fees/calculate", data).await
    }

    pub async fn pay_fees(&self, data: Value) -> Result<Value, PartyError> {
        self.post("/fees/pay", data).await
    }

    // -- Favorites (auth) --

    pub async fn list_favorites(&self) -> Result<Value, PartyError> {
        self.get("/favorites", true).await
    }

    pub async fn add_favorite(&self, data: Value) -> Result<Value, PartyError> {
        self.post("/favorites", data).await
    }

    pub async fn remove_favorite(&self, id: &str) -> Result<Value, PartyError> {
        self.delete(&format!("/favorites/{}", encode(id))).await
    }
}

fn encode(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    for b in s.as_bytes() {
        let c = *b as char;
        if c.is_ascii_alphanumeric() || c == '-' || c == '_' || c == '.' || c == '~' {
            out.push(c);
        } else {
            out.push_str(&format!("%{:02X}", b));
        }
    }
    out
}
