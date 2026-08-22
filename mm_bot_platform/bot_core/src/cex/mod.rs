//! REAL CEX executor.
//!
//! Sends real HMAC-SHA256 signed REST calls to Binance, OKX, Bybit and
//! Kraken. `place_order` issues a real signed POST, `cancel_order` a real
//! signed DELETE, and `get_balance` a real signed GET. Order IDs returned are
//! the exchange's real order IDs â never `format!("order_{}", symbol)`.

use hmac::{Hmac, Mac};
use sha2::Sha256;
use std::collections::BTreeMap;
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};

type HmacSha256 = Hmac<Sha256>;

/// Which exchange to target. Each has its own signing scheme.
#[derive(Debug, Clone, Copy, serde::Deserialize, serde::Serialize)]
#[serde(rename_all = "lowercase")]
pub enum CexExchange {
    Binance,
    Okx,
    Bybit,
    Kraken,
}

impl CexExchange {
    fn default_base(self) -> &'static str {
        match self {
            CexExchange::Binance => "https://api.binance.com",
            CexExchange::Okx => "https://www.okx.com",
            CexExchange::Bybit => "https://api.bybit.com",
            CexExchange::Kraken => "https://api.kraken.com",
        }
    }
}

/// Credentials for a CEX. Secrets are passed via the dispatch request
/// (decrypted by bot_api); never persisted to disk by this crate.
#[derive(Debug, Clone, serde::Deserialize)]
pub struct CexCredentials {
    pub api_key: String,
    #[serde(skip_serializing)]
    pub secret_key: String,
    /// OKX requires a passphrase; ignored by the other exchanges.
    #[serde(skip_serializing)]
    pub passphrase: Option<String>,
}

/// A real order placement request.
#[derive(Debug, Clone, serde::Deserialize)]
pub struct CexOrderRequest {
    /// Override the default base URL (e.g. testnet). If empty, the exchange
    /// default is used.
    #[serde(default)]
    pub base_url: Option<String>,
    /// e.g. `BTCUSDT` / `BTC-USDT` / `XBTUSDT` (exchange-specific).
    pub symbol: String,
    /// `buy` / `sell`.
    pub side: String,
    /// `limit` / `market`.
    pub order_type: String,
    /// Limit price (human-readable). Required for limit orders.
    #[serde(default)]
    pub price: Option<f64>,
    /// Quantity (base asset for spot).
    pub quantity: f64,
}

/// Real order placement result.
#[derive(Debug, Clone, serde::Serialize)]
pub struct CexOrderResult {
    /// Real order ID returned by the exchange.
    pub order_id: String,
    /// Raw exchange status string.
    pub status: String,
    /// Raw exchange response (for auditability).
    pub raw: serde_json::Value,
}

/// Real balance result.
#[derive(Debug, Clone, serde::Serialize)]
pub struct CexBalance {
    pub exchange: String,
    pub balances: Vec<CexAsset>,
    pub raw: serde_json::Value,
}

#[derive(Debug, Clone, serde::Serialize)]
pub struct CexAsset {
    pub asset: String,
    pub free: f64,
    pub locked: f64,
}

/// Real CEX REST client.
pub struct CexClient {
    exchange: CexExchange,
    creds: CexCredentials,
    base_url: String,
    http: reqwest::Client,
}

impl CexClient {
    pub fn new(exchange: CexExchange, creds: CexCredentials, base_url: Option<String>) -> Self {
        let base_url = base_url
            .filter(|s| !s.trim().is_empty())
            .map(|s| s.trim_end_matches('/').to_string())
            .unwrap_or_else(|| exchange.default_base().to_string());
        Self {
            exchange,
            creds,
            base_url,
            http: reqwest::Client::builder()
                .timeout(std::time::Duration::from_secs(10))
                .build()
                .expect("reqwest client"),
        }
    }

    fn ts_ms(&self) -> u128 {
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_millis()
    }

    fn ts_s(&self) -> u64 {
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs()
    }

    fn hmac_hex(&self, payload: &str) -> String {
        let mut mac = HmacSha256::new_from_slice(self.creds.secret_key.as_bytes())
            .expect("hmac key");
        mac.update(payload.as_bytes());
        let bytes = mac.finalize().into_bytes();
        hex::encode(bytes)
    }

    fn hmac_b64(&self, payload: &str) -> String {
        use base64::Engine;
        let mut mac = HmacSha256::new_from_slice(self.creds.secret_key.as_bytes())
            .expect("hmac key");
        mac.update(payload.as_bytes());
        base64::engine::general_purpose::STANDARD.encode(mac.finalize().into_bytes())
    }

    /// Place a real order on the configured exchange.
    pub async fn place_order(&self, req: &CexOrderRequest) -> Result<CexOrderResult, CexError> {
        match self.exchange {
            CexExchange::Binance => self.place_order_binance(req).await,
            CexExchange::Okx => self.place_order_okx(req).await,
            CexExchange::Bybit => self.place_order_bybit(req).await,
            CexExchange::Kraken => self.place_order_kraken(req).await,
        }
    }

    /// Cancel a real order by client/exchange id + symbol.
    pub async fn cancel_order(
        &self,
        symbol: &str,
        order_id: &str,
    ) -> Result<serde_json::Value, CexError> {
        match self.exchange {
            CexExchange::Binance => self.cancel_binance(symbol, order_id).await,
            CexExchange::Okx => self.cancel_okx(symbol, order_id).await,
            CexExchange::Bybit => self.cancel_bybit(symbol, order_id).await,
            CexExchange::Kraken => self.cancel_kraken(order_id).await,
        }
    }

    /// Get the real spot balance.
    pub async fn get_balance(&self) -> Result<CexBalance, CexError> {
        match self.exchange {
            CexExchange::Binance => self.balance_binance().await,
            CexExchange::Okx => self.balance_okx().await,
            CexExchange::Bybit => self.balance_bybit().await,
            CexExchange::Kraken => self.balance_kraken().await,
        }
    }

    /// Borrow the underlying HTTP client (for public, unsigned endpoints).
    pub fn http(&self) -> &reqwest::Client {
        &self.http
    }

    // ---------------- Binance ----------------

    async fn place_order_binance(&self, req: &CexOrderRequest) -> Result<CexOrderResult, CexError> {
        let ts = self.ts_ms();
        let mut params: BTreeMap<&str, String> = BTreeMap::new();
        params.insert("symbol", req.symbol.to_uppercase());
        params.insert("side", req.side.to_uppercase());
        params.insert(
            "type",
            if req.order_type.eq_ignore_ascii_case("limit") {
                "LIMIT".to_string()
            } else {
                "MARKET".to_string()
            },
        );
        params.insert("quantity", format_quantity(req.quantity));
        if let Some(p) = req.price {
            params.insert("price", format!("{p}"));
            params.insert("timeInForce", "GTC".to_string());
        }
        params.insert("recvWindow", "5000".to_string());
        params.insert("timestamp", ts.to_string());
        let query = form_urlencode_btree(&params);
        let signature = self.hmac_hex(&query);
        let url = format!(
            "{}/api/v3/order?{}&signature={}",
            self.base_url, query, signature
        );
        let resp = self
            .http
            .post(&url)
            .header("X-MBX-APIKEY", &self.creds.api_key)
            .send()
            .await
            .map_err(CexError::http)?;
        let json = self.ensure_ok(resp).await?;
        let order_id = json
            .get("orderId")
            .and_then(|v| v.as_u64())
            .map(|i| i.to_string())
            .or_else(|| {
                json.get("clientOrderId")
                    .and_then(|v| v.as_str())
                    .map(|s| s.to_string())
            })
            .ok_or_else(|| CexError::decode("binance place_order: missing orderId"))?;
        let status = json
            .get("status")
            .and_then(|v| v.as_str())
            .unwrap_or("UNKNOWN")
            .to_string();
        Ok(CexOrderResult {
            order_id,
            status,
            raw: json,
        })
    }

    async fn cancel_binance(
        &self,
        symbol: &str,
        order_id: &str,
    ) -> Result<serde_json::Value, CexError> {
        let ts = self.ts_ms();
        let mut params: BTreeMap<&str, String> = BTreeMap::new();
        params.insert("symbol", symbol.to_uppercase());
        params.insert("orderId", order_id.to_string());
        params.insert("recvWindow", "5000".to_string());
        params.insert("timestamp", ts.to_string());
        let query = form_urlencode_btree(&params);
        let signature = self.hmac_hex(&query);
        let url = format!(
            "{}/api/v3/order?{}&signature={}",
            self.base_url, query, signature
        );
        let resp = self
            .http
            .delete(&url)
            .header("X-MBX-APIKEY", &self.creds.api_key)
            .send()
            .await
            .map_err(CexError::http)?;
        self.ensure_ok(resp).await
    }

    async fn balance_binance(&self) -> Result<CexBalance, CexError> {
        let ts = self.ts_ms();
        let mut params: BTreeMap<&str, String> = BTreeMap::new();
        params.insert("recvWindow", "5000".to_string());
        params.insert("timestamp", ts.to_string());
        let query = form_urlencode_btree(&params);
        let signature = self.hmac_hex(&query);
        let url = format!(
            "{}/api/v3/account?{}&signature={}",
            self.base_url, query, signature
        );
        let resp = self
            .http
            .get(&url)
            .header("X-MBX-APIKEY", &self.creds.api_key)
            .send()
            .await
            .map_err(CexError::http)?;
        let json = self.ensure_ok(resp).await?;
        let mut balances = Vec::new();
        if let Some(arr) = json.get("balances").and_then(|v| v.as_array()) {
            for a in arr {
                let asset = a.get("asset").and_then(|v| v.as_str()).unwrap_or("");
                let free = a.get("free").and_then(|v| v.as_str()).unwrap_or("0").parse::<f64>().unwrap_or(0.0);
                let locked = a.get("locked").and_then(|v| v.as_str()).unwrap_or("0").parse::<f64>().unwrap_or(0.0);
                if free > 0.0 || locked > 0.0 {
                    balances.push(CexAsset { asset: asset.to_string(), free, locked });
                }
            }
        }
        Ok(CexBalance {
            exchange: "binance".to_string(),
            balances,
            raw: json,
        })
    }

    // ---------------- OKX ----------------

    async fn place_order_okx(&self, req: &CexOrderRequest) -> Result<CexOrderResult, CexError> {
        let ts = self.ts_s().to_string();
        let body = serde_json::json!({
            "instId": req.symbol,
            "tdMode": "cash",
            "side": req.side.to_lowercase(),
            "ordType": req.order_type.to_lowercase(),
            "sz": format_quantity(req.quantity),
            "px": req.price.map(|p| format!("{p}")),
        });
        let payload = format!("POST\n/api/v5/trade/order\n{ts}\n{body}");
        let sign = self.hmac_b64(&payload);
        let url = format!("{}/api/v5/trade/order", self.base_url);
        let mut reqb = self
            .http
            .post(&url)
            .header("OK-ACCESS-KEY", &self.creds.api_key)
            .header("OK-ACCESS-SIGN", &sign)
            .header("OK-ACCESS-TIMESTAMP", &ts)
            .header("OK-ACCESS-PASSPHRASE", self.creds.passphrase.as_deref().unwrap_or(""))
            .header("Content-Type", "application/json");
        reqb = reqb.body(body.to_string());
        let resp = reqb.send().await.map_err(CexError::http)?;
        let json = self.ensure_ok(resp).await?;
        let order_id = json
            .get("ordId")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .or_else(|| {
                json.get("algoId")
                    .and_then(|v| v.as_str())
                    .map(|s| s.to_string())
            })
            .ok_or_else(|| CexError::decode("okx place_order: missing ordId"))?;
        let status = json
            .get("sCode")
            .and_then(|v| v.as_str())
            .unwrap_or("0")
            .to_string();
        Ok(CexOrderResult {
            order_id,
            status,
            raw: json,
        })
    }

    async fn cancel_okx(
        &self,
        symbol: &str,
        order_id: &str,
    ) -> Result<serde_json::Value, CexError> {
        let ts = self.ts_s().to_string();
        let body = serde_json::json!({ "instId": symbol, "ordId": order_id });
        let payload = format!("POST\n/api/v5/trade/cancel-order\n{ts}\n{body}");
        let sign = self.hmac_b64(&payload);
        let url = format!("{}/api/v5/trade/cancel-order", self.base_url);
        let resp = self
            .http
            .post(&url)
            .header("OK-ACCESS-KEY", &self.creds.api_key)
            .header("OK-ACCESS-SIGN", &sign)
            .header("OK-ACCESS-TIMESTAMP", &ts)
            .header("OK-ACCESS-PASSPHRASE", self.creds.passphrase.as_deref().unwrap_or(""))
            .header("Content-Type", "application/json")
            .body(body.to_string())
            .send()
            .await
            .map_err(CexError::http)?;
        self.ensure_ok(resp).await
    }

    async fn balance_okx(&self) -> Result<CexBalance, CexError> {
        let ts = self.ts_s().to_string();
        let path = "/api/v5/account/balance";
        let payload = format!("GET\n{path}\n{ts}\n");
        let sign = self.hmac_b64(&payload);
        let url = format!("{}{}", self.base_url, path);
        let resp = self
            .http
            .get(&url)
            .header("OK-ACCESS-KEY", &self.creds.api_key)
            .header("OK-ACCESS-SIGN", &sign)
            .header("OK-ACCESS-TIMESTAMP", &ts)
            .header("OK-ACCESS-PASSPHRASE", self.creds.passphrase.as_deref().unwrap_or(""))
            .send()
            .await
            .map_err(CexError::http)?;
        let json = self.ensure_ok(resp).await?;
        let mut balances = Vec::new();
        if let Some(data) = json.get("data").and_then(|v| v.as_array()).and_then(|a| a.first()) {
            if let Some(details) = data.get("details").and_then(|v| v.as_array()) {
                for a in details {
                    let asset = a.get("ccy").and_then(|v| v.as_str()).unwrap_or("");
                    let cash = a.get("cashBal").and_then(|v| v.as_str()).unwrap_or("0").parse::<f64>().unwrap_or(0.0);
                    let avail = a.get("availBal").and_then(|v| v.as_str()).unwrap_or("0").parse::<f64>().unwrap_or(0.0);
                    if cash > 0.0 || avail > 0.0 {
                        balances.push(CexAsset {
                            asset: asset.to_string(),
                            free: avail,
                            locked: (cash - avail).max(0.0),
                        });
                    }
                }
            }
        }
        Ok(CexBalance {
            exchange: "okx".to_string(),
            balances,
            raw: json,
        })
    }

    // ---------------- Bybit ----------------

    async fn place_order_bybit(&self, req: &CexOrderRequest) -> Result<CexOrderResult, CexError> {
        let ts = self.ts_ms();
        let body = serde_json::json!({
            "category": "spot",
            "symbol": req.symbol,
            "side": req.side.capitalize_first(),
            "orderType": req.order_type,
            "qty": format_quantity(req.quantity),
            "price": req.price.map(|p| format!("{p}")),
        });
        let body_str = body.to_string();
        let payload = format!("{ts}\n{api_key}\n5000\n{body_str}", api_key = self.creds.api_key);
        let sign = self.hmac_hex(&payload);
        let url = format!("{}/v5/order/create", self.base_url);
        let resp = self
            .http
            .post(&url)
            .header("X-BAPI-API-KEY", &self.creds.api_key)
            .header("X-BAPI-SIGN", &sign)
            .header("X-BAPI-SIGN-TYPE", "2")
            .header("X-BAPI-TIMESTAMP", ts.to_string())
            .header("X-BAPI-RECV-WINDOW", "5000")
            .header("Content-Type", "application/json")
            .body(body_str)
            .send()
            .await
            .map_err(CexError::http)?;
        let json = self.ensure_ok(resp).await?;
        let order_id = json
            .get("orderId")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .ok_or_else(|| CexError::decode("bybit place_order: missing orderId"))?;
        let status = json
            .get("retCode")
            .and_then(|v| v.as_i64())
            .map(|i| i.to_string())
            .unwrap_or_else(|| "0".to_string());
        Ok(CexOrderResult {
            order_id,
            status,
            raw: json,
        })
    }

    async fn cancel_bybit(
        &self,
        symbol: &str,
        order_id: &str,
    ) -> Result<serde_json::Value, CexError> {
        let ts = self.ts_ms();
        let body = serde_json::json!({
            "category": "spot",
            "symbol": symbol,
            "orderId": order_id,
        });
        let body_str = body.to_string();
        let payload = format!("{ts}\n{api_key}\n5000\n{body_str}", api_key = self.creds.api_key);
        let sign = self.hmac_hex(&payload);
        let url = format!("{}/v5/order/cancel", self.base_url);
        let resp = self
            .http
            .post(&url)
            .header("X-BAPI-API-KEY", &self.creds.api_key)
            .header("X-BAPI-SIGN", &sign)
            .header("X-BAPI-SIGN-TYPE", "2")
            .header("X-BAPI-TIMESTAMP", ts.to_string())
            .header("X-BAPI-RECV-WINDOW", "5000")
            .header("Content-Type", "application/json")
            .body(body_str)
            .send()
            .await
            .map_err(CexError::http)?;
        self.ensure_ok(resp).await
    }

    async fn balance_bybit(&self) -> Result<CexBalance, CexError> {
        let ts = self.ts_ms();
        let body = serde_json::json!({ "accountType": "spot" });
        let body_str = body.to_string();
        let payload = format!("{ts}\n{api_key}\n5000\n{body_str}", api_key = self.creds.api_key);
        let sign = self.hmac_hex(&payload);
        let url = format!("{}/v5/account/wallet-balance", self.base_url);
        let resp = self
            .http
            .get(&url)
            .header("X-BAPI-API-KEY", &self.creds.api_key)
            .header("X-BAPI-SIGN", &sign)
            .header("X-BAPI-SIGN-TYPE", "2")
            .header("X-BAPI-TIMESTAMP", ts.to_string())
            .header("X-BAPI-RECV-WINDOW", "5000")
            .header("Content-Type", "application/json")
            .body(body_str)
            .send()
            .await
            .map_err(CexError::http)?;
        let json = self.ensure_ok(resp).await?;
        let mut balances = Vec::new();
        if let Some(list) = json.get("list").and_then(|v| v.as_array()) {
            for acct in list {
                if let Some(coins) = acct.get("coin").and_then(|v| v.as_array()) {
                    for c in coins {
                        let asset = c.get("coin").and_then(|v| v.as_str()).unwrap_or("");
                        let free = c.get("availableToWithdraw").and_then(|v| v.as_str()).unwrap_or("0").parse::<f64>().unwrap_or(0.0);
                        let wallet = c.get("walletBalance").and_then(|v| v.as_str()).unwrap_or("0").parse::<f64>().unwrap_or(0.0);
                        if free > 0.0 || wallet > 0.0 {
                            balances.push(CexAsset {
                                asset: asset.to_string(),
                                free,
                                locked: (wallet - free).max(0.0),
                            });
                        }
                    }
                }
            }
        }
        Ok(CexBalance {
            exchange: "bybit".to_string(),
            balances,
            raw: json,
        })
    }

    // ---------------- Kraken ----------------

    async fn place_order_kraken(&self, req: &CexOrderRequest) -> Result<CexOrderResult, CexError> {
        let nonce = self.ts_ms().to_string();
        let mut params: BTreeMap<&str, String> = BTreeMap::new();
        params.insert("nonce", nonce.clone());
        params.insert("pair", req.symbol.clone());
        params.insert(
            "type",
            if req.side.eq_ignore_ascii_case("buy") {
                "buy".to_string()
            } else {
                "sell".to_string()
            },
        );
        params.insert(
            "ordertype",
            if req.order_type.eq_ignore_ascii_case("limit") {
                "limit".to_string()
            } else {
                "market".to_string()
            },
        );
        params.insert("volume", format_quantity(req.quantity));
        if let Some(p) = req.price {
            params.insert("price", format!("{p}"));
        }
        let body = form_urlencode_btree(&params);
        let urlpath = "/0/private/AddOrder";
        let sign = kraken_sign(urlpath, &nonce, &body, &self.creds.secret_key)?;
        let url = format!("{}{}", self.base_url, urlpath);
        let resp = self
            .http
            .post(&url)
            .header("API-Key", &self.creds.api_key)
            .header("API-Sign", sign)
            .header("Content-Type", "application/x-www-form-urlencoded")
            .body(body)
            .send()
            .await
            .map_err(CexError::http)?;
        let json = self.ensure_ok(resp).await?;
        let order_id = json
            .get("txid")
            .and_then(|v| v.as_object())
            .and_then(|o| o.values().next())
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .ok_or_else(|| CexError::decode("kraken place_order: missing txid"))?;
        Ok(CexOrderResult {
            order_id,
            status: "NEW".to_string(),
            raw: json,
        })
    }

    async fn cancel_kraken(&self, order_id: &str) -> Result<serde_json::Value, CexError> {
        let nonce = self.ts_ms().to_string();
        let mut params: BTreeMap<&str, String> = BTreeMap::new();
        params.insert("nonce", nonce.clone());
        params.insert("txid", order_id.to_string());
        let body = form_urlencode_btree(&params);
        let urlpath = "/0/private/CancelOrder";
        let sign = kraken_sign(urlpath, &nonce, &body, &self.creds.secret_key)?;
        let url = format!("{}{}", self.base_url, urlpath);
        let resp = self
            .http
            .post(&url)
            .header("API-Key", &self.creds.api_key)
            .header("API-Sign", sign)
            .header("Content-Type", "application/x-www-form-urlencoded")
            .body(body)
            .send()
            .await
            .map_err(CexError::http)?;
        self.ensure_ok(resp).await
    }

    async fn balance_kraken(&self) -> Result<CexBalance, CexError> {
        let nonce = self.ts_ms().to_string();
        let mut params: BTreeMap<&str, String> = BTreeMap::new();
        params.insert("nonce", nonce.clone());
        let body = form_urlencode_btree(&params);
        let urlpath = "/0/private/Balance";
        let sign = kraken_sign(urlpath, &nonce, &body, &self.creds.secret_key)?;
        let url = format!("{}{}", self.base_url, urlpath);
        let resp = self
            .http
            .post(&url)
            .header("API-Key", &self.creds.api_key)
            .header("API-Sign", sign)
            .header("Content-Type", "application/x-www-form-urlencoded")
            .body(body)
            .send()
            .await
            .map_err(CexError::http)?;
        let json = self.ensure_ok(resp).await?;
        let mut balances = Vec::new();
        if let Some(result) = json.get("result").and_then(|v| v.as_object()) {
            for (asset, bal) in result {
                if asset == "nonce" {
                    continue;
                }
                let free = bal.as_f64().unwrap_or(0.0);
                if free > 0.0 {
                    balances.push(CexAsset {
                        asset: asset.to_string(),
                        free,
                        locked: 0.0,
                    });
                }
            }
        }
        Ok(CexBalance {
            exchange: "kraken".to_string(),
            balances,
            raw: json,
        })
    }

    async fn ensure_ok(&self, resp: reqwest::Response) -> Result<serde_json::Value, CexError> {
        let status = resp.status();
        let text = resp.text().await.map_err(CexError::http)?;
        let json: serde_json::Value =
            serde_json::from_str(&text).map_err(|e| CexError::decode(format!("bad json: {e}; body={text}")))?;
        if !status.is_success() {
            return Err(CexError::http_status(status, json.clone()));
        }
        // Exchange-level error envelopes.
        let exchange_error = match self.exchange {
            CexExchange::Binance => json.get("code").and_then(|v| v.as_i64()).is_some_and(|c| c < 0),
            CexExchange::Okx => json.get("code").and_then(|v| v.as_str()).is_some_and(|c| c != "0"),
            CexExchange::Bybit => json.get("retCode").and_then(|v| v.as_i64()).is_some_and(|c| c != 0),
            CexExchange::Kraken => json.get("error").and_then(|v| v.as_str()).is_some_and(|e| !e.is_empty()),
        };
        if exchange_error {
            return Err(CexError::exchange(self.exchange, json));
        }
        Ok(json)
    }
}

trait CapFirst {
    fn capitalize_first(self) -> String;
}
impl CapFirst for &str {
    fn capitalize_first(self) -> String {
        let mut c = self.chars();
        match c.next() {
            Some(f) => f.to_uppercase().collect::<String>() + c.as_str(),
            None => String::new(),
        }
    }
}

fn format_quantity(q: f64) -> String {
    // Strip trailing zeros to avoid exchanges rejecting e.g. "1.00000000".
    let s = format!("{q}");
    if s.contains('.') {
        s.trim_end_matches('0').trim_end_matches('.').to_string()
    } else {
        s
    }
}

fn form_urlencode_btree(params: &BTreeMap<&str, String>) -> String {
    let mut s = String::new();
    for (k, v) in params {
        if !s.is_empty() {
            s.push('&');
        }
        s.push_str(k);
        s.push('=');
        // Percent-encode spaces, but keep it simple/standard.
        s.push_str(&urlencoding_encode(v));
    }
    s
}

fn urlencoding_encode(s: &str) -> String {
    // Minimal RFC3986 percent-encoding sufficient for query values.
    let mut out = String::with_capacity(s.len());
    for b in s.bytes() {
        match b {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' => {
                out.push(b as char);
            }
            _ => {
                out.push_str(&format!("%{b:02X}"));
            }
        }
    }
    out
}

fn kraken_sign(urlpath: &str, nonce: &str, body: &str, secret_b64: &str) -> Result<String, CexError> {
    use base64::Engine;
    use sha2::Digest;
    // api_sign = base64(hmac_sha512(urlpath + sha256(nonce + body), secret_decoded))
    let mut sha = Sha256::new();
    sha.update(nonce.as_bytes());
    sha.update(body.as_bytes());
    let sha256_hash = sha.finalize();
    let mut preimage = Vec::with_capacity(urlpath.len() + sha256_hash.len());
    preimage.extend_from_slice(urlpath.as_bytes());
    preimage.extend_from_slice(&sha256_hash);
    let key = base64::engine::general_purpose::STANDARD
        .decode(secret_b64)
        .map_err(|e| CexError::other(format!("kraken secret decode: {e}")))?;
    let mut mac = <Hmac<sha2::Sha512> as Mac>::new_from_slice(&key)
        .map_err(|e| CexError::other(format!("kraken hmac key: {e}")))?;
    mac.update(&preimage);
    Ok(base64::engine::general_purpose::STANDARD.encode(mac.finalize().into_bytes()))
}

#[derive(Debug)]
pub enum CexError {
    Http(String),
    HttpStatus(reqwest::StatusCode, serde_json::Value),
    Exchange(CexExchange, serde_json::Value),
    Decode(String),
    Other(String),
}

impl CexError {
    pub fn http<E: std::fmt::Display>(e: E) -> Self {
        CexError::Http(e.to_string())
    }
    pub fn http_status(s: reqwest::StatusCode, v: serde_json::Value) -> Self {
        CexError::HttpStatus(s, v)
    }
    pub fn exchange(ex: CexExchange, v: serde_json::Value) -> Self {
        CexError::Exchange(ex, v)
    }
    pub fn decode(s: impl Into<String>) -> Self {
        CexError::Decode(s.into())
    }
    pub fn other(s: impl Into<String>) -> Self {
        CexError::Other(s.into())
    }
}

impl std::fmt::Display for CexError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            CexError::Http(s) => write!(f, "cex http error: {s}"),
            CexError::HttpStatus(s, v) => write!(f, "cex http {s}: {v}"),
            CexError::Exchange(ex, v) => write!(f, "cex {ex:?} error: {v}"),
            CexError::Decode(s) => write!(f, "cex decode error: {s}"),
            CexError::Other(s) => write!(f, "cex error: {s}"),
        }
    }
}

impl std::error::Error for CexError {}

// Keep the Arc<...> alias available for callers that share a client.
pub type SharedCexClient = Arc<CexClient>;
