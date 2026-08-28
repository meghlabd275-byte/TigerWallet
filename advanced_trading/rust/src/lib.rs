//! Advanced Trading — Limit orders, TWAP, AMM-DEX aggregation, oracle execution.
//!
//! Real HTTP client logic for the advanced-trading domain. It does NOT
//! fabricate order ids, swap outputs, or gas estimates: every operation calls
//! the configured DEX aggregator / oracle / order-book backend and surfaces
//! the real response, or returns a typed error.

use serde::{Deserialize, Serialize};
use thiserror::Error;

/// TradingService dispatches advanced-trading operations to a backend trading
/// gateway over HTTP. The backend performs the real on-chain AMM routing and
/// settlement; this client never fabricates results.
pub struct TradingService {
    pub chain_id: u64,
    pub gateway_url: String,
    client: reqwest::Client,
}

#[derive(Debug, Error)]
pub enum TradingError {
    #[error("missing configuration: {0}")]
    MissingConfig(String),
    #[error("gateway request failed: {0}")]
    Request(String),
    #[error("gateway rejected operation (HTTP {status}): {body}")]
    Gateway { status: u16, body: String },
    #[error("invalid response: {0}")]
    Parse(String),
}

impl From<reqwest::Error> for TradingError {
    fn from(e: reqwest::Error) -> Self {
        TradingError::Request(e.to_string())
    }
}

impl TradingService {
    pub fn new(chain_id: u64, gateway_url: impl Into<String>) -> Result<Self, TradingError> {
        let gateway_url = gateway_url.into();
        if gateway_url.is_empty() {
            return Err(TradingError::MissingConfig("gateway_url is required".into()));
        }
        Ok(Self {
            chain_id,
            gateway_url,
            client: reqwest::Client::builder()
                .timeout(std::time::Duration::from_secs(15))
                .build()?,
        })
    }

    /// Place a limit order via the trading gateway. Returns the real order id
    /// assigned by the gateway (never fabricated).
    pub async fn place_limit(&self, order: &LimitOrder) -> Result<String, TradingError> {
        let resp = self
            .client
            .post(format!("{}/api/v1/advanced/limit", self.gateway_url))
            .json(order)
            .send()
            .await?;
        let status = resp.status().as_u16();
        let body = resp.text().await.unwrap_or_default();
        if status >= 300 {
            return Err(TradingError::Gateway { status, body });
        }
        let v: GatewayOrderId =
            serde_json::from_str(&body).map_err(|e| TradingError::Parse(e.to_string()))?;
        Ok(v.order_id)
    }

    /// Place a TWAP order via the trading gateway.
    pub async fn place_twap(&self, order: &TWAPOrder) -> Result<String, TradingError> {
        let resp = self
            .client
            .post(format!("{}/api/v1/advanced/twap", self.gateway_url))
            .json(order)
            .send()
            .await?;
        let status = resp.status().as_u16();
        let body = resp.text().await.unwrap_or_default();
        if status >= 300 {
            return Err(TradingError::Gateway { status, body });
        }
        let v: GatewayOrderId =
            serde_json::from_str(&body).map_err(|e| TradingError::Parse(e.to_string()))?;
        Ok(v.order_id)
    }

    /// Aggregate a DEX swap quote via the aggregator gateway. The returned
    /// SwapResult reflects the real router quote (path, expected out, gas).
    pub async fn aggregate_dex(
        &self,
        from: &str,
        to: &str,
        amount: u64,
    ) -> Result<SwapResult, TradingError> {
        let resp = self
            .client
            .get(format!("{}/api/v1/dex/quote", self.gateway_url))
            .query(&[
                ("chain_id", self.chain_id.to_string()),
                ("token_in", from.to_string()),
                ("token_out", to.to_string()),
                ("amount_in", amount.to_string()),
            ])
            .send()
            .await?;
        let status = resp.status().as_u16();
        let body = resp.text().await.unwrap_or_default();
        if status >= 300 {
            return Err(TradingError::Gateway { status, body });
        }
        let q: QuoteResponse =
            serde_json::from_str(&body).map_err(|e| TradingError::Parse(e.to_string()))?;
        Ok(SwapResult {
            from_amount: amount,
            to_amount: q.to_amount,
            path: q.path,
            gas: q.estimated_gas,
        })
    }

    /// Execute (fill) a limit order at the given market price via the gateway.
    pub async fn execute_limit(&self, order_id: &str, price: u64) -> Result<(), TradingError> {
        let resp = self
            .client
            .post(format!(
                "{}/api/v1/advanced/limit/{}/execute",
                self.gateway_url, order_id
            ))
            .json(&serde_json::json!({ "price": price, "chain_id": self.chain_id }))
            .send()
            .await?;
        let status = resp.status().as_u16();
        if status >= 300 {
            let body = resp.text().await.unwrap_or_default();
            return Err(TradingError::Gateway { status, body });
        }
        Ok(())
    }

    /// Execute an order against an oracle price feed via the gateway.
    pub async fn execute_oracle(&self, order_id: &str, oracle: &str) -> Result<(), TradingError> {
        let resp = self
            .client
            .post(format!(
                "{}/api/v1/advanced/limit/{}/oracle",
                self.gateway_url, order_id
            ))
            .json(&serde_json::json!({ "oracle": oracle, "chain_id": self.chain_id }))
            .send()
            .await?;
        let status = resp.status().as_u16();
        if status >= 300 {
            let body = resp.text().await.unwrap_or_default();
            return Err(TradingError::Gateway { status, body });
        }
        Ok(())
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LimitOrder {
    pub id: String,
    pub token_in: String,
    pub token_out: String,
    pub amount_in: u64,
    pub price_limit: u64,
    pub expiry: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TWAPOrder {
    pub id: String,
    pub token_in: String,
    pub token_out: String,
    pub total_amount: u64,
    pub num_orders: u32,
    pub interval: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapResult {
    pub from_amount: u64,
    pub to_amount: u64,
    pub path: Vec<String>,
    pub gas: u64,
}

#[derive(Debug, Deserialize)]
struct GatewayOrderId {
    #[serde(rename = "order_id")]
    order_id: String,
}

#[derive(Debug, Deserialize)]
struct QuoteResponse {
    to_amount: u64,
    path: Vec<String>,
    estimated_gas: u64,
}
