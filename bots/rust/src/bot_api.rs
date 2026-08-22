//! TigerBots Rust client.
//!
//! Async reqwest client delegating to the standalone Bots backend
//! (`mm_bot_platform/bot_api`, port 8471, path prefix `/api/v1`). JWT bearer
//! auth; the token is held in-process (`Arc<RwLock<Option<String>>>`) — Rust
//! clients are server/CLI and have no keychain dependency by default; callers
//! that need persistence can wrap `set_token`/`token`.
//!
//! Every method issues a real reqwest call against the backend — no stubs,
//! fakes, or mock data. On any non-2xx response the method returns an `Err`
//! carrying the backend's `error` message (fail-closed); it never returns
//! fabricated data.
//!
//! Method set mirrors `bots/web/src/services/api.ts`:
//!   auth: `login`, `register`, `logout`
//!   bots: `list_bots`, `get_bot`, `create_bot`, `delete_bot`, `start_bot`,
//!         `stop_bot`, `pause_bot`, `list_bot_executions`, `list_bot_logs`,
//!         `list_bot_instances`, `current_bot_user`
//!   users: `list_bot_users`, `create_bot_user`, `delete_bot_user`,
//!         `list_bot_transactions`
//!   subscriptions: `get_subscription`, `create_subscription`
//!   fees: `get_fee_configs`, `update_fee_config`
//!   cex: `list_cex`, `add_cex`, `remove_cex`
//!   dex: `list_dex`, `add_dex`, `remove_dex`
//!   keys: `list_api_keys`, `create_api_key`, `delete_api_key`
//!   admin: `admin_list_users`, `admin_user_status`, `admin_stats`,
//!         `admin_get_fee_addresses`, `admin_set_fee_address`,
//!         `admin_delete_fee_address`, `admin_bot_status`
//!   public: `public_tiers`, `health`

use reqwest::{Client, Method};
use serde_json::Value;
use std::sync::{Arc, RwLock};
use std::time::Duration;

pub const BOTS_API_DEFAULT_URL: &str = "http://localhost:8471/api/v1";

/// Error returned by every [`BotsClient`] method. `Http` carries the status
/// code and the backend's `error`/`message` text (or raw body). Fail-closed:
/// the client never fabricates a response on failure.
#[derive(Debug)]
pub enum BotError {
    Network(String),
    Decode(String),
    Http { status: u16, message: String },
    Unauthorized,
}

impl std::fmt::Display for BotError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            BotError::Network(m) => write!(f, "network: {m}"),
            BotError::Decode(m) => write!(f, "decode: {m}"),
            BotError::Http { status, message } => write!(f, "http {status}: {message}"),
            BotError::Unauthorized => write!(f, "not authenticated: no JWT token set"),
        }
    }
}

impl std::error::Error for BotError {}

/// Pooled async HTTP client for the standalone Bots backend.
pub struct BotsClient {
    http: Client,
    base_url: String,
    token: Arc<RwLock<Option<String>>>,
}

impl BotsClient {
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

    /// Convenience constructor using [`BOTS_API_DEFAULT_URL`] and no token.
    pub fn default() -> Self {
        Self::new(BOTS_API_DEFAULT_URL, None)
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

    fn url(&self, path: &str) -> String {
        format!("{}{}", self.base_url, path)
    }

    async fn request(
        &self,
        method: Method,
        path: &str,
        body: Option<Value>,
        authenticated: bool,
    ) -> Result<Value, BotError> {
        let url = self.url(path);
        let mut req = self.http.request(method, &url);
        if authenticated {
            if let Some(t) = self.token() {
                req = req.bearer_auth(t);
            } else {
                return Err(BotError::Unauthorized);
            }
        }
        if let Some(b) = body {
            req = req.json(&b);
        }
        let resp = req.send().await.map_err(|e| BotError::Network(e.to_string()))?;
        let status = resp.status();
        let bytes = resp
            .bytes()
            .await
            .map_err(|e| BotError::Network(e.to_string()))?;
        let text = String::from_utf8_lossy(&bytes);
        let value: Value = if text.trim().is_empty() {
            Value::Null
        } else {
            serde_json::from_str(&text).map_err(|e| BotError::Decode(e.to_string()))?
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
            return Err(BotError::Http {
                status: status.as_u16(),
                message: msg,
            });
        }
        Ok(value)
    }

    async fn get(&self, path: &str, authenticated: bool) -> Result<Value, BotError> {
        self.request(Method::GET, path, None, authenticated).await
    }

    async fn send_body(
        &self,
        method: Method,
        path: &str,
        body: Value,
    ) -> Result<Value, BotError> {
        self.request(method, path, Some(body), true).await
    }

    async fn post(&self, path: &str, body: Value) -> Result<Value, BotError> {
        self.send_body(Method::POST, path, body).await
    }

    async fn put(&self, path: &str, body: Value) -> Result<Value, BotError> {
        self.send_body(Method::PUT, path, body).await
    }

    async fn delete(&self, path: &str) -> Result<Value, BotError> {
        self.request(Method::DELETE, path, None, true).await
    }

    async fn post_no_body(&self, path: &str) -> Result<Value, BotError> {
        self.request(Method::POST, path, None, true).await
    }

    // -- Auth --

    pub async fn register(
        &self,
        username: &str,
        password: &str,
        email: Option<&str>,
        wallet_address: Option<&str>,
        role: Option<&str>,
    ) -> Result<Value, BotError> {
        let mut payload = serde_json::json!({ "username": username, "password": password });
        if let Some(e) = email {
            payload["email"] = Value::String(e.to_string());
        }
        if let Some(w) = wallet_address {
            payload["wallet_address"] = Value::String(w.to_string());
        }
        if let Some(r) = role {
            payload["role"] = Value::String(r.to_string());
        }
        let res = self
            .request(Method::POST, "/auth/register", Some(payload), false)
            .await?;
        if let Some(t) = res.get("token").and_then(|v| v.as_str()) {
            self.set_token(Some(t.to_string()));
        }
        Ok(res)
    }

    pub async fn login(&self, username: &str, password: &str) -> Result<Value, BotError> {
        let payload = serde_json::json!({ "username": username, "password": password });
        let res = self
            .request(Method::POST, "/auth/login", Some(payload), false)
            .await?;
        if let Some(t) = res.get("token").and_then(|v| v.as_str()) {
            self.set_token(Some(t.to_string()));
        }
        Ok(res)
    }

    pub async fn logout(&self) -> Result<(), BotError> {
        let res = self.post_no_body("/auth/logout").await;
        self.set_token(None);
        match res {
            Ok(_) => Ok(()),
            Err(e) => Err(e),
        }
    }

    // -- Health + public tiers --

    pub async fn health(&self) -> Result<Value, BotError> {
        self.get("/health", false).await
    }

    pub async fn public_tiers(&self) -> Result<Value, BotError> {
        self.get("/public/tiers", false).await
    }

    // -- Bots CRUD + lifecycle --

    pub async fn list_bots(&self) -> Result<Value, BotError> {
        self.get("/bots", true).await
    }

    pub async fn get_bot(&self, id: &str) -> Result<Value, BotError> {
        self.get(&format!("/bots/{}", encode(id)), true).await
    }

    pub async fn create_bot(
        &self,
        name: &str,
        bot_type: &str,
        config: Option<Value>,
        exchange: Option<&str>,
        pair: Option<&str>,
    ) -> Result<Value, BotError> {
        let mut payload = serde_json::json!({
            "name": name,
            "bot_type": bot_type,
            "config": config.unwrap_or_else(|| Value::Object(Default::default())),
        });
        if let Some(e) = exchange {
            payload["exchange"] = Value::String(e.to_string());
        }
        if let Some(p) = pair {
            payload["pair"] = Value::String(p.to_string());
        }
        self.post("/bots", payload).await
    }

    pub async fn delete_bot(&self, id: &str) -> Result<Value, BotError> {
        self.delete(&format!("/bots/{}", encode(id))).await
    }

    pub async fn start_bot(&self, id: &str) -> Result<Value, BotError> {
        self.post_no_body(&format!("/bots/{}/start", encode(id))).await
    }

    pub async fn stop_bot(&self, id: &str) -> Result<Value, BotError> {
        self.post_no_body(&format!("/bots/{}/stop", encode(id))).await
    }

    pub async fn pause_bot(&self, id: &str) -> Result<Value, BotError> {
        self.post_no_body(&format!("/bots/{}/pause", encode(id))).await
    }

    pub async fn list_bot_executions(&self, id: &str) -> Result<Value, BotError> {
        self.get(&format!("/bots/{}/executions", encode(id)), true)
            .await
    }

    pub async fn list_bot_logs(&self, id: &str) -> Result<Value, BotError> {
        self.get(&format!("/bots/{}/logs", encode(id)), true).await
    }

    pub async fn list_bot_instances(&self) -> Result<Value, BotError> {
        self.get("/bots/instances", true).await
    }

    pub async fn current_bot_user(&self) -> Result<Value, BotError> {
        self.get("/bots/me", true).await
    }

    // -- Bot users --

    pub async fn list_bot_users(&self) -> Result<Value, BotError> {
        self.get("/bots/users", true).await
    }

    pub async fn create_bot_user(
        &self,
        username: &str,
        password: &str,
        email: Option<&str>,
        wallet_address: Option<&str>,
        role: Option<&str>,
    ) -> Result<Value, BotError> {
        let mut payload = serde_json::json!({ "username": username, "password": password });
        if let Some(e) = email {
            payload["email"] = Value::String(e.to_string());
        }
        if let Some(w) = wallet_address {
            payload["wallet_address"] = Value::String(w.to_string());
        }
        if let Some(r) = role {
            payload["role"] = Value::String(r.to_string());
        }
        self.post("/bots/users", payload).await
    }

    pub async fn delete_bot_user(&self, id: &str) -> Result<Value, BotError> {
        self.delete(&format!("/bots/users/{}", encode(id))).await
    }

    pub async fn list_bot_transactions(&self) -> Result<Value, BotError> {
        self.get("/bots/transactions", true).await
    }

    // -- Subscriptions --

    pub async fn get_subscription(&self) -> Result<Value, BotError> {
        self.get("/subscription", true).await
    }

    pub async fn create_subscription(
        &self,
        tier: &str,
        expires_in: Option<&str>,
    ) -> Result<Value, BotError> {
        let mut payload = serde_json::json!({ "tier": tier });
        if let Some(e) = expires_in {
            payload["expires_in"] = Value::String(e.to_string());
        }
        self.post("/subscription", payload).await
    }

    // -- Fees --

    pub async fn get_fee_configs(&self) -> Result<Value, BotError> {
        self.get("/fees", true).await
    }

    pub async fn update_fee_config(
        &self,
        id: &str,
        name: Option<&str>,
        percentage: Option<&str>,
        enabled: Option<bool>,
    ) -> Result<Value, BotError> {
        let mut payload = serde_json::json!({ "id": id });
        if let Some(n) = name {
            payload["name"] = Value::String(n.to_string());
        }
        if let Some(p) = percentage {
            payload["percentage"] = Value::String(p.to_string());
        }
        if let Some(e) = enabled {
            payload["enabled"] = Value::Bool(e);
        }
        self.put("/fees", payload).await
    }

    // -- CEX connectors --

    pub async fn list_cex(&self) -> Result<Value, BotError> {
        self.get("/cex", true).await
    }

    pub async fn add_cex(&self, name: &str, config: Value) -> Result<Value, BotError> {
        self.post("/cex", serde_json::json!({ "name": name, "config": config }))
            .await
    }

    pub async fn remove_cex(&self, id: &str) -> Result<Value, BotError> {
        self.delete(&format!("/cex/{}", encode(id))).await
    }

    // -- DEX connectors --

    pub async fn list_dex(&self) -> Result<Value, BotError> {
        self.get("/dex", true).await
    }

    pub async fn add_dex(&self, name: &str, config: Value) -> Result<Value, BotError> {
        self.post("/dex", serde_json::json!({ "name": name, "config": config }))
            .await
    }

    pub async fn remove_dex(&self, id: &str) -> Result<Value, BotError> {
        self.delete(&format!("/dex/{}", encode(id))).await
    }

    // -- API keys --

    pub async fn list_api_keys(&self) -> Result<Value, BotError> {
        self.get("/keys", true).await
    }

    pub async fn create_api_key(&self, exchange: &str, api_key: &str) -> Result<Value, BotError> {
        self.post(
            "/keys",
            serde_json::json!({ "exchange": exchange, "api_key": api_key }),
        )
        .await
    }

    pub async fn delete_api_key(&self, id: &str) -> Result<Value, BotError> {
        self.delete(&format!("/keys/{}", encode(id))).await
    }

    // -- Admin (super-admin / finance-admin only) --

    pub async fn admin_list_users(&self) -> Result<Value, BotError> {
        self.get("/admin/users", true).await
    }

    pub async fn admin_user_status(&self, id: &str, active: bool) -> Result<Value, BotError> {
        self.put(
            &format!("/admin/users/{}/status", encode(id)),
            serde_json::json!({ "id": id, "is_active": active }),
        )
        .await
    }

    pub async fn admin_stats(&self) -> Result<Value, BotError> {
        self.get("/admin/stats", true).await
    }

    pub async fn admin_get_fee_addresses(&self) -> Result<Value, BotError> {
        self.get("/admin/fee-addresses", true).await
    }

    pub async fn admin_set_fee_address(
        &self,
        label: &str,
        chain_id: i64,
        address: &str,
    ) -> Result<Value, BotError> {
        self.post(
            "/admin/fee-addresses",
            serde_json::json!({ "label": label, "chain_id": chain_id, "address": address }),
        )
        .await
    }

    pub async fn admin_delete_fee_address(&self, id: &str) -> Result<Value, BotError> {
        self.delete(&format!("/admin/fee-addresses/{}", encode(id))).await
    }

    pub async fn admin_bot_status(&self, id: &str, status: &str) -> Result<Value, BotError> {
        self.post(
            &format!("/admin/bots/{}/status", encode(id)),
            serde_json::json!({ "id": id, "status": status }),
        )
        .await
    }
}

/// Percent-encode a path segment (ids may contain characters unsafe in a URL).
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
