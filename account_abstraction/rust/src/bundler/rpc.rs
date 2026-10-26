//! Minimal JSON-RPC client for the methods the bundler needs.
//!
//! Only the handful of Ethereum JSON-RPC methods required to build and submit a
//! bundle are implemented, so the service does not pull in a full node client.

use crate::AAError;
use serde::Deserialize;
use serde_json::{json, Value};

/// A JSON-RPC client bound to one endpoint.
pub struct JsonRpcClient {
    url: String,
    http: reqwest::Client,
}

#[derive(Deserialize)]
struct RpcResponse {
    result: Option<Value>,
    error: Option<RpcError>,
}

#[derive(Deserialize)]
struct RpcError {
    code: i64,
    message: String,
}

impl JsonRpcClient {
    pub fn new(url: impl Into<String>) -> Self {
        Self {
            url: url.into(),
            http: reqwest::Client::new(),
        }
    }

    /// Send one JSON-RPC request and return its result.
    async fn call(&self, method: &str, params: Value) -> Result<Value, AAError> {
        let body = json!({
            "jsonrpc": "2.0",
            "id": 1,
            "method": method,
            "params": params,
        });

        let response = self
            .http
            .post(&self.url)
            .json(&body)
            .send()
            .await
            .map_err(|e| AAError::BundlerError(format!("{method} request failed: {e}")))?;

        let status = response.status();
        let text = response
            .text()
            .await
            .map_err(|e| AAError::BundlerError(format!("{method} response unreadable: {e}")))?;

        let parsed: RpcResponse = serde_json::from_str(&text).map_err(|_| {
            AAError::BundlerError(format!("{method} returned HTTP {status} with a non-JSON body"))
        })?;

        if let Some(error) = parsed.error {
            return Err(AAError::BundlerError(format!(
                "{method} failed: {} (code {})",
                error.message, error.code
            )));
        }

        parsed
            .result
            .ok_or_else(|| AAError::BundlerError(format!("{method} returned no result")))
    }

    /// Parse a `0x`-prefixed hex quantity into a `u128`.
    fn parse_quantity(value: &Value, method: &str) -> Result<u128, AAError> {
        let text = value
            .as_str()
            .ok_or_else(|| AAError::BundlerError(format!("{method} returned a non-string result")))?;

        let stripped = text.strip_prefix("0x").unwrap_or(text);
        if stripped.is_empty() {
            return Err(AAError::BundlerError(format!(
                "{method} returned an empty quantity"
            )));
        }

        u128::from_str_radix(stripped, 16)
            .map_err(|e| AAError::BundlerError(format!("{method} returned {text}: {e}")))
    }

    /// `eth_chainId`.
    pub async fn chain_id(&self) -> Result<u64, AAError> {
        let result = self.call("eth_chainId", json!([])).await?;
        Ok(Self::parse_quantity(&result, "eth_chainId")? as u64)
    }

    /// `eth_gasPrice`.
    pub async fn gas_price(&self) -> Result<u128, AAError> {
        let result = self.call("eth_gasPrice", json!([])).await?;
        Self::parse_quantity(&result, "eth_gasPrice")
    }

    /// `eth_getTransactionCount` for the pending block.
    pub async fn transaction_count(&self, address: &str) -> Result<u64, AAError> {
        let result = self
            .call("eth_getTransactionCount", json!([address, "pending"]))
            .await?;
        Ok(Self::parse_quantity(&result, "eth_getTransactionCount")? as u64)
    }

    /// `eth_estimateGas`.
    pub async fn estimate_gas(&self, from: &str, to: &str, data: &str) -> Result<u128, AAError> {
        let result = self
            .call(
                "eth_estimateGas",
                json!([{ "from": from, "to": to, "data": data }]),
            )
            .await?;
        Self::parse_quantity(&result, "eth_estimateGas")
    }

    /// `eth_sendRawTransaction`, returning the transaction hash.
    pub async fn send_raw_transaction(&self, raw_hex: &str) -> Result<String, AAError> {
        let result = self
            .call("eth_sendRawTransaction", json!([raw_hex]))
            .await?;

        result
            .as_str()
            .map(|value| value.to_string())
            .ok_or_else(|| {
                AAError::BundlerError("eth_sendRawTransaction returned no transaction hash".to_string())
            })
    }

    /// `eth_call`, returning the raw return data.
    pub async fn call_contract(&self, to: &str, data: &str) -> Result<String, AAError> {
        let result = self
            .call("eth_call", json!([{ "to": to, "data": data }, "latest"]))
            .await?;

        result
            .as_str()
            .map(|value| value.to_string())
            .ok_or_else(|| AAError::BundlerError("eth_call returned a non-string result".to_string()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn quantities_are_parsed_from_hex() {
        assert_eq!(
            JsonRpcClient::parse_quantity(&json!("0x1"), "test").unwrap(),
            1
        );
        assert_eq!(
            JsonRpcClient::parse_quantity(&json!("0x10"), "test").unwrap(),
            16
        );
        assert_eq!(
            JsonRpcClient::parse_quantity(&json!("0x0"), "test").unwrap(),
            0
        );
    }

    #[test]
    fn malformed_quantities_are_rejected() {
        assert!(JsonRpcClient::parse_quantity(&json!("0x"), "test").is_err());
        assert!(JsonRpcClient::parse_quantity(&json!("0xzz"), "test").is_err());
        assert!(JsonRpcClient::parse_quantity(&json!(1), "test").is_err());
    }
}
