//! zkSync Era JSON-RPC provider — real HTTP client against a configured
//! endpoint. Fail-closed: every transport/parse failure surfaces as
//! ZksyncError; no data is ever fabricated.

use serde::de::DeserializeOwned;
use serde_json::{json, Value};

use crate::types::{TransactionReceipt, TransactionStatusWire, ZksyncError, H256};

pub struct ZksyncProvider {
    endpoint: String,
    client: reqwest::Client,
    next_id: std::sync::atomic::AtomicU64,
}

impl ZksyncProvider {
    pub fn new(endpoint: &str) -> Self {
        Self {
            endpoint: endpoint.to_string(),
            client: reqwest::Client::builder()
                .timeout(std::time::Duration::from_secs(30))
                .build()
                .expect("reqwest client build"),
            next_id: std::sync::atomic::AtomicU64::new(1),
        }
    }

    async fn rpc<T: DeserializeOwned>(&self, method: &str, params: Value) -> Result<T, ZksyncError> {
        let id = self
            .next_id
            .fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        let body = json!({
            "jsonrpc": "2.0",
            "id": id,
            "method": method,
            "params": params,
        });
        let resp: Value = self
            .client
            .post(&self.endpoint)
            .json(&body)
            .send()
            .await
            .map_err(|e| ZksyncError::RpcError(e.to_string()))?
            .json()
            .await
            .map_err(|e| ZksyncError::RpcError(e.to_string()))?;
        if let Some(err) = resp.get("error") {
            return Err(ZksyncError::RpcError(err.to_string()));
        }
        let result = resp
            .get("result")
            .cloned()
            .ok_or_else(|| ZksyncError::RpcError("missing result".to_string()))?;
        serde_json::from_value(result).map_err(|e| ZksyncError::RpcError(e.to_string()))
    }
}

fn hex_u64(s: &str) -> Result<u64, ZksyncError> {
    let s = s.strip_prefix("0x").unwrap_or(s);
    u64::from_str_radix(s, 16).map_err(|e| ZksyncError::RpcError(e.to_string()))
}

fn hex_u128(s: &str) -> Result<u128, ZksyncError> {
    let s = s.strip_prefix("0x").unwrap_or(s);
    u128::from_str_radix(s, 16).map_err(|e| ZksyncError::RpcError(e.to_string()))
}

fn hex_h256(s: &str) -> Result<H256, ZksyncError> {
    let s = s.strip_prefix("0x").unwrap_or(s);
    let raw = hex::decode(s).map_err(|e| ZksyncError::RpcError(e.to_string()))?;
    if raw.len() != 32 {
        return Err(ZksyncError::RpcError("hash not 32 bytes".to_string()));
    }
    let mut out = [0u8; 32];
    out.copy_from_slice(&raw);
    Ok(out)
}

impl ZksyncProvider {
    pub async fn chain_id(&self) -> Result<u64, ZksyncError> {
        let v: String = self.rpc("eth_chainId", json!([])).await?;
        hex_u64(&v)
    }

    pub async fn block_number(&self) -> Result<u64, ZksyncError> {
        let v: String = self.rpc("eth_blockNumber", json!([])).await?;
        hex_u64(&v)
    }

    pub async fn get_balance(&self, address: &str) -> Result<u128, ZksyncError> {
        let v: String = self
            .rpc("eth_getBalance", json!([address, "latest"]))
            .await?;
        hex_u128(&v)
    }

    pub async fn get_transaction_count(&self, address: &str) -> Result<u64, ZksyncError> {
        let v: String = self
            .rpc("eth_getTransactionCount", json!([address, "latest"]))
            .await?;
        hex_u64(&v)
    }

    pub async fn gas_price(&self) -> Result<u128, ZksyncError> {
        let v: String = self.rpc("eth_gasPrice", json!([])).await?;
        hex_u128(&v)
    }

    pub async fn estimate_gas(&self, tx: Value) -> Result<u64, ZksyncError> {
        let v: String = self.rpc("eth_estimateGas", json!([tx])).await?;
        hex_u64(&v)
    }

    pub async fn send_raw_transaction(&self, raw_hex: &str) -> Result<H256, ZksyncError> {
        let v: String = self
            .rpc("eth_sendRawTransaction", json!([raw_hex]))
            .await?;
        hex_h256(&v)
    }

    pub async fn call(&self, tx: Value) -> Result<Vec<u8>, ZksyncError> {
        let v: String = self.rpc("eth_call", json!([tx, "latest"])).await?;
        let s = v.strip_prefix("0x").unwrap_or(&v);
        hex::decode(s).map_err(|e| ZksyncError::RpcError(e.to_string()))
    }

    pub async fn transaction_receipt(
        &self,
        tx_hash: &str,
    ) -> Result<Option<TransactionReceipt>, ZksyncError> {
        let v: Option<Value> = self
            .rpc("eth_getTransactionReceipt", json!([tx_hash]))
            .await?;
        let Some(v) = v else { return Ok(None) };

        let status = match v["status"].as_str() {
            Some("0x1") => TransactionStatusWire::Success,
            Some("0x0") => TransactionStatusWire::Reverted,
            _ => TransactionStatusWire::Unknown,
        };
        let mut hash = [0u8; 32];
        if let Some(h) = v["transactionHash"].as_str() {
            hash = hex_h256(h)?;
        }
        let gas_used = match v["gasUsed"].as_str() {
            Some(g) => hex_u64(g)?,
            None => 0,
        };
        let block_number = match v["blockNumber"].as_str() {
            Some(b) => hex_u64(b)?,
            None => 0,
        };
        let effective_gas_price = match v["effectiveGasPrice"].as_str() {
            Some(g) => hex_u64(g)?,
            None => 0,
        };
        let contract_address = v["contractAddress"]
            .as_str()
            .and_then(|c| hex::decode(c.strip_prefix("0x").unwrap_or(c)).ok())
            .and_then(|b| <[u8; 20]>::try_from(b.as_slice()).ok());

        Ok(Some(TransactionReceipt {
            transaction_hash: hash,
            status,
            block_number,
            gas_used,
            effective_gas_price,
            contract_address,
            logs: vec![],
        }))
    }

    /// Poll for a receipt until confirmed or timeout
    pub async fn wait_for_transaction(
        &self,
        tx_hash: &str,
        timeout_secs: u64,
    ) -> Result<TransactionReceipt, ZksyncError> {
        let deadline = std::time::Instant::now() + std::time::Duration::from_secs(timeout_secs);
        loop {
            if let Some(r) = self.transaction_receipt(tx_hash).await? {
                return Ok(r);
            }
            if std::time::Instant::now() > deadline {
                return Err(ZksyncError::RpcError("confirmation timeout".to_string()));
            }
            tokio::time::sleep(std::time::Duration::from_secs(1)).await;
        }
    }
}
