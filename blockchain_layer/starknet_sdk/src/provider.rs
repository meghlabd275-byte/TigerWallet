//! Starknet RPC Provider
//!
//! Real JSON-RPC client for Starknet nodes. Every method round-trips to the
//! configured endpoint; failures surface as ProviderError. No simulated data.

use std::fmt;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use reqwest::Client;

use crate::address::StarknetAddress;
use crate::types::{Block, ContractClass, Transaction as RpcTransaction};
use crate::transaction::{InvokeTransaction, DeployAccountTransaction, TransactionReceipt};

const DEFAULT_TIMEOUT: u64 = 30;

/// Starknet RPC Provider
pub struct Provider {
    rpc_url: String,
    client: Client,
}

#[derive(Debug, Serialize)]
struct RpcRequest {
    jsonrpc: String,
    method: String,
    params: Value,
    id: u64,
}

#[derive(Debug, Deserialize)]
struct RpcError {
    code: i64,
    message: String,
}

#[derive(Debug, Deserialize)]
struct RpcResponse {
    result: Option<Value>,
    error: Option<RpcError>,
}

/// Block identifier for read calls.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct BlockIdentifier {
    #[serde(rename = "block_number", skip_serializing_if = "Option::is_none")]
    pub block_number: Option<u64>,
    #[serde(rename = "block_hash", skip_serializing_if = "Option::is_none")]
    pub block_hash: Option<String>,
}

/// Function call envelope for starknet_call.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FunctionCall {
    #[serde(rename = "contract_address")]
    pub contract_address: String,
    #[serde(rename = "entry_point_selector")]
    pub entry_point_selector: String,
    #[serde(rename = "calldata", default)]
    pub calldata: Vec<String>,
}

/// Contract code returned by starknet_getCode (legacy).
#[derive(Debug, Clone, Default)]
pub struct ContractCode {
    pub bytecode: Vec<u8>,
    pub abi: Option<Vec<u8>>,
}

/// Fee estimate returned by starknet_estimateFee.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeeEstimate {
    #[serde(rename = "gas_consumed")]
    pub gas_consumed: u64,
    #[serde(rename = "gas_price")]
    pub gas_price: u64,
    #[serde(rename = "overall_fee")]
    pub overall_fee: u64,
    #[serde(rename = "unit")]
    pub unit: FeeUnit,
}

/// Fee unit.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum FeeUnit {
    Wei,
    Fri,
}

/// Result of an invoke transaction submission.
#[derive(Debug, Clone)]
pub struct InvokeResult {
    pub transaction_hash: [u8; 32],
    pub contract_address: [u8; 32],
}

/// Result of an account deployment.
#[derive(Debug, Clone)]
pub struct DeployResult {
    pub transaction_hash: [u8; 32],
    pub contract_address: [u8; 32],
}

/// Provider errors — fail-closed; nothing is fabricated.
#[derive(Debug, Clone)]
pub enum ProviderError {
    NetworkError(String),
    ParseError(String),
    TransactionNotFound,
    BlockNotFound,
    ContractNotFound,
    Timeout,
    RpcError(String),
    InvalidResponse(String),
}

impl fmt::Display for ProviderError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ProviderError::NetworkError(e) => write!(f, "Network error: {}", e),
            ProviderError::ParseError(e) => write!(f, "Parse error: {}", e),
            ProviderError::TransactionNotFound => write!(f, "Transaction not found"),
            ProviderError::BlockNotFound => write!(f, "Block not found"),
            ProviderError::ContractNotFound => write!(f, "Contract not found"),
            ProviderError::Timeout => write!(f, "Request timeout"),
            ProviderError::RpcError(e) => write!(f, "RPC error: {}", e),
            ProviderError::InvalidResponse(e) => write!(f, "Invalid response: {}", e),
        }
    }
}

impl std::error::Error for ProviderError {}

fn hex32_to_bytes(s: &str) -> Result<[u8; 32], ProviderError> {
    let h = s.trim_start_matches("0x");
    let padded = format!("{:0>64}", h);
    let bytes = hex::decode(&padded)
        .map_err(|e| ProviderError::InvalidResponse(format!("bad hex value: {}", e)))?;
    if bytes.len() != 32 {
        return Err(ProviderError::InvalidResponse("expected 32-byte word".into()));
    }
    let mut out = [0u8; 32];
    out.copy_from_slice(&bytes);
    Ok(out)
}

impl Provider {
    /// Create new provider bound to a Starknet JSON-RPC endpoint.
    pub fn new(rpc_url: &str) -> Self {
        let client = Client::builder()
            .timeout(std::time::Duration::from_secs(DEFAULT_TIMEOUT))
            .build()
            .expect("Failed to create HTTP client");
        Self { rpc_url: rpc_url.to_string(), client }
    }

    /// Get chain id via starknet_chainId.
    pub async fn chain_id(&self) -> Result<String, ProviderError> {
        let v = self.rpc_call("starknet_chainId", json!([])).await?;
        v.as_str()
            .map(|s| s.to_string())
            .ok_or_else(|| ProviderError::InvalidResponse("chainId not a string".into()))
    }

    async fn rpc_call(&self, method: &str, params: Value) -> Result<Value, ProviderError> {
        let body = RpcRequest {
            jsonrpc: "2.0".to_string(),
            method: method.to_string(),
            params,
            id: 1,
        };
        let resp = self
            .client
            .post(&self.rpc_url)
            .header("Content-Type", "application/json")
            .json(&body)
            .send()
            .await
            .map_err(|e| ProviderError::NetworkError(e.to_string()))?;
        let parsed: RpcResponse = resp
            .json()
            .await
            .map_err(|e| ProviderError::ParseError(e.to_string()))?;
        if let Some(err) = parsed.error {
            return Err(ProviderError::RpcError(format!("{} (code {})", err.message, err.code)));
        }
        parsed
            .result
            .ok_or_else(|| ProviderError::InvalidResponse("missing result".into()))
    }

    /// Latest block number.
    pub async fn block_number(&self) -> Result<u64, ProviderError> {
        let v = self.rpc_call("starknet_blockNumber", json!([])).await?;
        v.as_u64()
            .ok_or_else(|| ProviderError::InvalidResponse("blockNumber not a u64".into()))
    }

    /// Block by number.
    pub async fn get_block_by_number(&self, block_number: u64) -> Result<Block, ProviderError> {
        let v = self
            .rpc_call(
                "starknet_getBlockWithTxHashes",
                json!([{ "block_number": block_number }]),
            )
            .await?;
        let r = parse_block(&v)?;
        Ok(r)
    }

    /// Block by hash.
    pub async fn get_block_by_hash(&self, block_hash: &[u8; 32]) -> Result<Block, ProviderError> {
        let hex_hash = format!("0x{}", hex::encode(block_hash));
        let v = self
            .rpc_call("starknet_getBlockWithTxHashes", json!([{ "block_hash": hex_hash }]))
            .await?;
        let r = parse_block(&v)?;
        Ok(r)
    }

    /// Nonce for address.
    pub async fn get_nonce(&self, address: &StarknetAddress) -> Result<[u8; 32], ProviderError> {
        let v = self
            .rpc_call(
                "starknet_getNonce",
                json!([
                    BlockIdentifier { block_number: None, block_hash: None },
                    address.to_hex()
                ]),
            )
            .await?;
        let s = v.as_str()
            .ok_or_else(|| ProviderError::InvalidResponse("nonce not a string".into()))?;
        hex32_to_bytes(s)
    }

    /// Balance of address.
    pub async fn get_balance(
        &self,
        address: &StarknetAddress,
        block_id: Option<BlockIdentifier>,
    ) -> Result<[u8; 32], ProviderError> {
        let v = self
            .rpc_call(
                "starknet_getBalance",
                json!([block_id.unwrap_or_default(), address.to_hex()]),
            )
            .await?;
        let s = v.as_str()
            .ok_or_else(|| ProviderError::InvalidResponse("balance not a string".into()))?;
        hex32_to_bytes(s)
    }

    /// Code at address (legacy starknet_getCode).
    pub async fn get_code(&self, address: &StarknetAddress) -> Result<ContractCode, ProviderError> {
        let v = self
            .rpc_call("starknet_getCode", json!([address.to_hex(), Value::Null]))
            .await?;
        let bytecode_hex = v["bytecode"].as_str().unwrap_or("");
        let bytecode = hex::decode(bytecode_hex.trim_start_matches("0x"))
            .map_err(|e| ProviderError::ParseError(e.to_string()))?;
        let abi = v["abi"].as_str().map(|s| s.as_bytes().to_vec());
        Ok(ContractCode { bytecode, abi })
    }

    /// Call contract.
    pub async fn call_contract(
        &self,
        call: FunctionCall,
        block_id: Option<BlockIdentifier>,
    ) -> Result<Vec<[u8; 32]>, ProviderError> {
        let v = self
            .rpc_call("starknet_call", json!([call, block_id.unwrap_or_default()]))
            .await?;
        let arr = v
            .as_array()
            .ok_or_else(|| ProviderError::InvalidResponse("call result not an array".into()))?;
        let mut out = Vec::new();
        for item in arr {
            let s = item
                .as_str()
                .ok_or_else(|| ProviderError::InvalidResponse("array item not a string".into()))?;
            out.push(hex32_to_bytes(s)?);
        }
        Ok(out)
    }

    /// Estimate fee for a set of invoke transactions.
    pub async fn estimate_fee(
        &self,
        requests: &[InvokeTransaction],
        block_id: Option<BlockIdentifier>,
    ) -> Result<Vec<FeeEstimate>, ProviderError> {
        let reqs = serde_json::to_value(requests)
            .map_err(|e| ProviderError::ParseError(e.to_string()))?;
        let v = self
            .rpc_call("starknet_estimateFee", json!([reqs, block_id.unwrap_or_default()]))
            .await?;
        let arr = v
            .as_array()
            .ok_or_else(|| ProviderError::InvalidResponse("estimateFee not an array".into()))?;
        let mut out = Vec::new();
        for item in arr {
            out.push(FeeEstimateWire::from_value(item)?.into());
        }
        Ok(out)
    }

    /// Transaction by hash.
    pub async fn get_transaction(&self, tx_hash: &[u8; 32]) -> Result<RpcTransaction, ProviderError> {
        let hash = format!("0x{}", hex::encode(tx_hash));
        let v = self
            .rpc_call("starknet_getTransactionByHash", json!([hash]))
            .await?;
        if v.is_null() {
            return Err(ProviderError::TransactionNotFound);
        }
        if !v.is_object() {
            return Err(ProviderError::InvalidResponse("transaction not an object".into()));
        }
        serde_json::from_value(v)
            .map_err(|e| ProviderError::ParseError(e.to_string()))
    }

    /// Transaction receipt.
    pub async fn get_transaction_receipt(
        &self,
        tx_hash: &[u8; 32],
    ) -> Result<TransactionReceipt, ProviderError> {
        let hash = format!("0x{}", hex::encode(tx_hash));
        let v = self
            .rpc_call("starknet_getTransactionReceipt", json!([hash]))
            .await?;
        if v.is_null() {
            return Err(ProviderError::TransactionNotFound);
        }
        if !v.is_object() {
            return Err(ProviderError::InvalidResponse("receipt not an object".into()));
        }
        serde_json::from_value(v)
            .map_err(|e| ProviderError::ParseError(e.to_string()))
    }

    /// Class hash at address.
    pub async fn get_class_hash(&self, address: &StarknetAddress) -> Result<[u8; 32], ProviderError> {
        let v = self
            .rpc_call(
                "starknet_getClassHashAt",
                json!([BlockIdentifier { block_number: None, block_hash: None }, address.to_hex()]),
            )
            .await?;
        let s = v.as_str()
            .ok_or_else(|| ProviderError::InvalidResponse("classHash not a string".into()))?;
        hex32_to_bytes(s)
    }

    /// Class at hash.
    pub async fn get_class(&self, class_hash: &[u8; 32]) -> Result<ContractClass, ProviderError> {
        let hash = format!("0x{}", hex::encode(class_hash));
        let v = self
            .rpc_call("starknet_getClass", json!([Value::Null, hash]))
            .await?;
        serde_json::from_value(v)
            .map_err(|e| ProviderError::ParseError(e.to_string()))
    }

    /// Storage at address.
    pub async fn get_storage_at(
        &self,
        address: &StarknetAddress,
        key: &[u8; 32],
        block_id: Option<BlockIdentifier>,
    ) -> Result<[u8; 32], ProviderError> {
        let v = self
            .rpc_call(
                "starknet_getStorageAt",
                json!([
                    address.to_hex(),
                    format!("0x{}", hex::encode(key)),
                    block_id.unwrap_or_default()
                ]),
            )
            .await?;
        let s = v.as_str()
            .ok_or_else(|| ProviderError::InvalidResponse("storage value not a string".into()))?;
        hex32_to_bytes(s)
    }

    /// Submit invoke transaction (broadcast).
    pub async fn add_invoke_transaction(
        &self,
        invoke: InvokeTransaction,
    ) -> Result<InvokeResult, ProviderError> {
        let wire = serde_json::to_value(&invoke)
            .map_err(|e| ProviderError::ParseError(e.to_string()))?;
        let v = self
            .rpc_call("starknet_addInvokeTransaction", json!([wire]))
            .await?;
        let tx_hash = hex32_to_bytes(
            v["transaction_hash"]
                .as_str()
                .ok_or_else(|| ProviderError::InvalidResponse("transaction_hash missing".into()))?,
        )?;
        let addr = hex32_to_bytes(
            v["contract_address"]
                .as_str()
                .ok_or_else(|| ProviderError::InvalidResponse("contract_address missing".into()))?,
        )?;
        Ok(InvokeResult { transaction_hash: tx_hash, contract_address: addr })
    }

    /// Submit account deployment (broadcast).
    pub async fn add_deploy_account_transaction(
        &self,
        deploy: DeployAccountTransaction,
    ) -> Result<DeployResult, ProviderError> {
        fn h(x: &[u8; 32]) -> String {
            format!("0x{}", hex::encode(x))
        }
        let wire = json!({
            "type": "deploy_account",
            "class_hash": h(&deploy.class_hash),
            "salt": h(&deploy.salt),
            "constructor_calldata": deploy
                .constructor_calldata
                .iter()
                .map(|c| h(c))
                .collect::<Vec<_>>(),
            "version": h(&deploy.version),
            "max_fee": h(&deploy.max_fee),
            "nonce": h(&deploy.nonce),
        });
        let v = self
            .rpc_call("starknet_addDeployAccountTransaction", json!([wire]))
            .await?;
        let tx_hash = hex32_to_bytes(
            v["transaction_hash"]
                .as_str()
                .ok_or_else(|| ProviderError::InvalidResponse("transaction_hash missing".into()))?,
        )?;
        let addr = hex32_to_bytes(
            v["contract_address"]
                .as_str()
                .ok_or_else(|| ProviderError::InvalidResponse("contract_address missing".into()))?,
        )?;
        Ok(DeployResult { transaction_hash: tx_hash, contract_address: addr })
    }

    /// Poll until transaction receipt appears or timeout.
    pub async fn wait_for_transaction(
        &self,
        tx_hash: [u8; 32],
        max_wait_seconds: u64,
    ) -> Result<TransactionReceipt, ProviderError> {
        let start = std::time::Instant::now();
        loop {
            if start.elapsed().as_secs() > max_wait_seconds {
                return Err(ProviderError::Timeout);
            }
            match self.get_transaction_receipt(&tx_hash).await {
                Ok(receipt) => return Ok(receipt),
                Err(ProviderError::TransactionNotFound) => {
                    tokio::time::sleep(std::time::Duration::from_secs(5)).await;
                }
                Err(e) => return Err(e),
            }
        }
    }
}

fn parse_block(v: &Value) -> Result<Block, ProviderError> {
    fn hex_or_empty(s: &Value, key: &str) -> Option<String> {
        s[key].as_str().map(|x| x.to_string())
    }
    let txs = v["transactions"]
        .as_array()
        .map(|arr| arr.iter().filter_map(|x| x.as_str().map(|s| s.to_string())).collect())
        .unwrap_or_default();
    Ok(Block {
        block_hash: hex_or_empty(v, "block_hash"),
        block_number: v["block_number"].as_u64(),
        parent_hash: v["parent_hash"].as_str().unwrap_or("").to_string(),
        timestamp: v["timestamp"].as_u64().unwrap_or(0),
        sequencer_address: v["sequencer_address"].as_str().unwrap_or("").to_string(),
        transactions: txs,
    })
}

#[derive(Deserialize)]
struct FeeEstimateWire {
    #[serde(rename = "gas_consumed")]
    gas_consumed: String,
    #[serde(rename = "gas_price")]
    gas_price: String,
    #[serde(rename = "overall_fee")]
    overall_fee: String,
    #[serde(rename = "unit", default)]
    unit: Option<String>,
}

impl FeeEstimateWire {
    fn from_value(v: &Value) -> Result<Self, ProviderError> {
        serde_json::from_value(v.clone()).map_err(|e| ProviderError::ParseError(e.to_string()))
    }
}

impl From<FeeEstimateWire> for FeeEstimate {
    fn from(w: FeeEstimateWire) -> Self {
        fn parse_num(b: &str) -> Option<u64> {
            let trimmed = b.trim_start_matches("0x");
            u64::from_str_radix(trimmed, 16)
                .ok()
                .or_else(|| b.parse::<u64>().ok())
        }
        FeeEstimate {
            gas_consumed: parse_num(&w.gas_consumed).unwrap_or(0),
            gas_price: parse_num(&w.gas_price).unwrap_or(0),
            overall_fee: parse_num(&w.overall_fee).unwrap_or(0),
            unit: match w.unit.as_deref() {
                Some("FRI") => FeeUnit::Fri,
                _ => FeeUnit::Wei,
            },
        }
    }
}
