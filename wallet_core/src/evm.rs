// ============================================================================
// TIGERWALLET EVM NETWORKS MODULE
// Complete RPC implementations for all EVM chains
// ============================================================================

use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::RwLock;
use serde::{Deserialize, Serialize};
use serde::de::DeserializeOwned;
use reqwest::Client;
use std::collections::BTreeMap;
use ethereum_types::{U64, U256, H256, H160, Address};

/// EVM chain configuration
#[derive(Debug, Clone)]
pub struct EvmChain {
    pub chain_id: u64,
    pub name: String,
    pub symbol: String,
    pub decimals: u8,
    pub rpc_url: String,
    pub explorer_url: String,
    pub explorer_api: String,
    pub native_token: String,
    pub gas_oracle: String,
}

/// Get all supported EVM chains
pub fn get_evm_chains() -> Vec<EvmChain> {
    vec![
        EvmChain {
            chain_id: 1,
            name: "Ethereum".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_url: "https://eth.llamarpc.com".to_string(),
            explorer_url: "https://etherscan.io".to_string(),
            explorer_api: "https://api.etherscan.io/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "https://api.etherscan.io/api?module=gastracker&action=gasoracle".to_string(),
        },
        EvmChain {
            chain_id: 56,
            name: "BNB Smart Chain".to_string(),
            symbol: "BNB".to_string(),
            decimals: 18,
            rpc_url: "https://bsc-dataseed1.binance.org".to_string(),
            explorer_url: "https://bscscan.com".to_string(),
            explorer_api: "https://api.bscscan.com/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "https://api.bscscan.com/api?module=gastracker&action=gasoracle".to_string(),
        },
        EvmChain {
            chain_id: 137,
            name: "Polygon".to_string(),
            symbol: "MATIC".to_string(),
            decimals: 18,
            rpc_url: "https://polygon-rpc.com".to_string(),
            explorer_url: "https://polygonscan.com".to_string(),
            explorer_api: "https://api.polygonscan.com/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "https://api.polygonscan.com/api?module=gastracker&action=gasoracle".to_string(),
        },
        EvmChain {
            chain_id: 42161,
            name: "Arbitrum One".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_url: "https://arb1.arbitrum.io/rpc".to_string(),
            explorer_url: "https://arbiscan.io".to_string(),
            explorer_api: "https://api.arbiscan.io/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
        EvmChain {
            chain_id: 10,
            name: "Optimism".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_url: "https://mainnet.optimism.io".to_string(),
            explorer_url: "https://optimistic.etherscan.io".to_string(),
            explorer_api: "https://api-optimistic.etherscan.io/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
        EvmChain {
            chain_id: 8453,
            name: "Base".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_url: "https://mainnet.base.org".to_string(),
            explorer_url: "https://basescan.org".to_string(),
            explorer_api: "https://api.basescan.org/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
        EvmChain {
            chain_id: 43114,
            name: "Avalanche C-Chain".to_string(),
            symbol: "AVAX".to_string(),
            decimals: 18,
            rpc_url: "https://api.avax.network/ext/bc/C/rpc".to_string(),
            explorer_url: "https://snowtrace.io".to_string(),
            explorer_api: "https://api.snowtrace.io/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
        EvmChain {
            chain_id: 25,
            name: "Cronos".to_string(),
            symbol: "CRO".to_string(),
            decimals: 18,
            rpc_url: "https://evm.cronos.org".to_string(),
            explorer_url: "https://cronoscan.com".to_string(),
            explorer_api: "https://api.cronoscan.com/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
        EvmChain {
            chain_id: 42220,
            name: "Celo".to_string(),
            symbol: "CELO".to_string(),
            decimals: 18,
            rpc_url: "https://forno.celo.org".to_string(),
            explorer_url: "https://explorer.celo.org".to_string(),
            explorer_api: "https://explorer.celo.org/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
        EvmChain {
            chain_id: 8217,
            name: "Klaytn".to_string(),
            symbol: "KLAY".to_string(),
            decimals: 18,
            rpc_url: "https://klaytn.fandom.finance".to_string(),
            explorer_url: "https://scope.klaytn.com".to_string(),
            explorer_api: "https://api.scope.klaytn.com/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
        EvmChain {
            chain_id: 1101,
            name: "Polygon zkEVM".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_url: "https://zkevm-rpc.com".to_string(),
            explorer_url: "https://zkevm.polygonscan.com".to_string(),
            explorer_api: "https://api-zkevm.polygonscan.com/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
        EvmChain {
            chain_id: 324,
            name: "zKSync Era".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_url: "https://main.era.zksync.io".to_string(),
            explorer_url: "https://explorer.zksync.io".to_string(),
            explorer_api: "https://block-explorer-api.mainnet.zksync.io/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
        EvmChain {
            chain_id: 59144,
            name: "Linea".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_url: "https://rpc.linea.build".to_string(),
            explorer_url: "https://lineascan.build".to_string(),
            explorer_api: "https://api.lineascan.build/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
        EvmChain {
            chain_id: 534352,
            name: "Scroll".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_url: "https://rpc.scroll.io".to_string(),
            explorer_url: "https://scrollscan.com".to_string(),
            explorer_api: "https://api.scrollscan.com/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
        EvmChain {
            chain_id: 5000,
            name: "Mantle".to_string(),
            symbol: "MNT".to_string(),
            decimals: 18,
            rpc_url: "https://rpc.mantle.xyz".to_string(),
            explorer_url: "https://explorer.mantle.xyz".to_string(),
            explorer_api: "https://explorer.mantle.xyz/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
        EvmChain {
            chain_id: 168587773,
            name: "Blast".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
            rpc_url: "https://blast-rpc.blastio.io".to_string(),
            explorer_url: "https://blastscan.io".to_string(),
            explorer_api: "https://api.blastscan.io/api".to_string(),
            native_token: "0x0000000000000000000000000000000000000000".to_string(),
            gas_oracle: "".to_string(),
        },
    ]
}

/// Get chain by ID
pub fn get_chain_by_id(chain_id: u64) -> Option<EvmChain> {
    get_evm_chains().into_iter().find(|c| c.chain_id == chain_id)
}

/// EVM RPC client
pub struct EvmRpcClient {
    chain: EvmChain,
    client: Client,
}

impl EvmRpcClient {
    pub fn new(chain: EvmChain) -> Self {
        Self {
            chain,
            client: Client::builder()
                .timeout(Duration::from_secs(30))
                .build()
                .unwrap_or_default(),
        }
    }

    pub fn from_chain_id(chain_id: u64) -> Option<Self> {
        get_chain_by_id(chain_id).map(|c| Self::new(c))
    }

    /// Get chain ID
    pub async fn get_chain_id(&self) -> Result<u64, RpcError> {
        let params = serde_json::json!([]);
        let result: String = self.rpc_call("eth_chainId", params).await?;
        Ok(u64::from_str_radix(&result[2..], 16).unwrap_or(self.chain.chain_id))
    }

    /// Get block number
    pub async fn get_block_number(&self) -> Result<u64, RpcError> {
        let params = serde_json::json!([]);
        let result: String = self.rpc_call("eth_blockNumber", params).await?;
        Ok(u64::from_str_radix(&result[2..], 16).unwrap_or(0))
    }

    /// Get balance
    pub async fn get_balance(&self, address: &str) -> Result<U256, RpcError> {
        let params = serde_json::json!([address, "latest"]);
        let result: String = self.rpc_call("eth_getBalance", params).await?;
        Ok(parse_u256(&result))
    }

    /// Get transaction count (nonce)
    pub async fn get_transaction_count(&self, address: &str) -> Result<u64, RpcError> {
        let params = serde_json::json!([address, "latest"]);
        let result: String = self.rpc_call("eth_getTransactionCount", params).await?;
        Ok(parse_u256(&result).as_u64())
    }

    /// Get transaction by hash
    pub async fn get_transaction(&self, tx_hash: &str) -> Result<Option<Transaction>, RpcError> {
        let params = serde_json::json!([tx_hash]);
        let result: Option<Transaction> = self.rpc_call("eth_getTransactionByHash", params).await?;
        Ok(result)
    }

    /// Get transaction receipt
    pub async fn get_transaction_receipt(&self, tx_hash: &str) -> Result<Option<TransactionReceipt>, RpcError> {
        let params = serde_json::json!([tx_hash]);
        let result: Option<TransactionReceipt> = self.rpc_call("eth_getTransactionReceipt", params).await?;
        Ok(result)
    }

    /// Get block by number
    pub async fn get_block_by_number(&self, block_number: u64) -> Result<Option<Block>, RpcError> {
        let block_hex = format!("0x{:x}", block_number);
        let params = serde_json::json!([block_hex, false]);
        let result: Option<Block> = self.rpc_call("eth_getBlockByNumber", params).await?;
        Ok(result)
    }

    /// Get code at address
    pub async fn get_code(&self, address: &str) -> Result<String, RpcError> {
        let params = serde_json::json!([address, "latest"]);
        let result: String = self.rpc_call("eth_getCode", params).await?;
        Ok(result)
    }

    /// Get storage at address
    pub async fn get_storage_at(&self, address: &str, slot: &str) -> Result<String, RpcError> {
        let params = serde_json::json!([address, slot, "latest"]);
        let result: String = self.rpc_call("eth_getStorageAt", params).await?;
        Ok(result)
    }

    /// Call contract
    pub async fn call(&self, to: &str, data: &str) -> Result<String, RpcError> {
        let params = serde_json::json!([{
            "to": to,
            "data": data
        }, "latest"]);
        let result: String = self.rpc_call("eth_call", params).await?;
        Ok(result)
    }

    /// Estimate gas
    pub async fn estimate_gas(&self, to: &str, from: Option<&str>, value: &str, data: &str) -> Result<u64, RpcError> {
        let mut tx = serde_json::json!({
            "to": to,
            "data": data
        });
        
        if let Some(from) = from {
            tx["from"] = serde_json::json!(from);
        }
        if !value.is_empty() && value != "0x0" {
            tx["value"] = serde_json::json!(value);
        }
        
        let params = serde_json::json!([tx, "latest"]);
        let result: String = self.rpc_call("eth_estimateGas", params).await?;
        Ok(parse_u256(&result).as_u64())
    }

    /// Get gas price
    pub async fn get_gas_price(&self) -> Result<U256, RpcError> {
        let params = serde_json::json!([]);
        let result: String = self.rpc_call("eth_gasPrice", params).await?;
        Ok(parse_u256(&result))
    }

    /// Get max priority fee
    pub async fn get_max_priority_fee(&self) -> Result<U256, RpcError> {
        let params = serde_json::json!([]);
        let result: String = self.rpc_call("eth_maxPriorityFeePerGas", params).await?;
        Ok(parse_u256(&result))
    }

    /// Send raw transaction
    pub async fn send_raw_transaction(&self, signed_tx: &[u8]) -> Result<String, RpcError> {
        let params = serde_json::json!([format!("0x{}", hex::encode(signed_tx))]);
        let result: String = self.rpc_call("eth_sendRawTransaction", params).await?;
        Ok(result)
    }

    /// Get logs
    pub async fn get_logs(&self, from_block: u64, to_block: u64, address: &str, topics: Vec<String>) -> Result<Vec<Log>, RpcError> {
        let params = serde_json::json!([{
            "fromBlock": format!("0x{:x}", from_block),
            "toBlock": format!("0x{:x}", to_block),
            "address": address,
            "topics": topics
        }]);
        let result: Vec<Log> = self.rpc_call("eth_getLogs", params).await?;
        Ok(result)
    }

    /// Get token balance
    pub async fn get_token_balance(&self, token: &str, owner: &str) -> Result<U256, RpcError> {
        let selector = "0x70a08231"; // balanceOf(address)
        let owner_bytes = hex::decode(owner.trim_start_matches("0x")).unwrap_or_default();
        let mut data = vec![0u8; 32 - owner_bytes.len()];
        data.extend_from_slice(&owner_bytes);
        
        let call_data = format!("{}{}", selector, hex::encode(data));
        let result = self.call(token, &call_data).await?;
        
        Ok(parse_u256(&result))
    }

    /// Get token supply
    pub async fn get_token_total_supply(&self, token: &str) -> Result<U256, RpcError> {
        let selector = "0x18160ddd"; // totalSupply()
        let call_data = selector.to_string();
        let result = self.call(token, &call_data).await?;
        
        Ok(parse_u256(&result))
    }

    /// Get token decimals
    pub async fn get_token_decimals(&self, token: &str) -> Result<u8, RpcError> {
        let selector = "0x313ce567"; // decimals()
        let call_data = selector.to_string();
        let result = self.call(token, &call_data).await?;
        
        let decimals = parse_u256(&result).as_u64() as u8;
        Ok(decimals)
    }

    /// Get token name
    pub async fn get_token_name(&self, token: &str) -> Result<String, RpcError> {
        let selector = "0x06fdde03"; // name()
        let call_data = selector.to_string();
        let result: String = self.call(token, &call_data).await?;
        
        // Decode string from result
        let data = hex::decode(result.trim_start_matches("0x")).unwrap_or_default();
        if data.len() > 32 {
            let len = data[32] as usize;
            if data.len() >= 32 + len {
                let name = String::from_utf8_lossy(&data[32..32+len]).to_string();
                return Ok(name);
            }
        }
        Ok("Unknown".to_string())
    }

    /// Get token symbol
    pub async fn get_token_symbol(&self, token: &str) -> Result<String, RpcError> {
        let selector = "0x95d89b41"; // symbol()
        let call_data = selector.to_string();
        let result: String = self.call(token, &call_data).await?;
        
        // Decode string from result
        let data = hex::decode(result.trim_start_matches("0x")).unwrap_or_default();
        if data.len() > 32 {
            let len = data[32] as usize;
            if data.len() >= 32 + len {
                let symbol = String::from_utf8_lossy(&data[32..32+len]).to_string();
                return Ok(symbol);
            }
        }
        Ok("UNKNOWN".to_string())
    }

    /// Internal JSON-RPC call
    async fn rpc_call<T: DeserializeOwned>(&self, method: &str, params: serde_json::Value) -> Result<T, RpcError> {
        let request = serde_json::json!({
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
            "id": 1
        });
        
        let response = self.client
            .post(&self.chain.rpc_url)
            .header("Content-Type", "application/json")
            .json(&request)
            .send()
            .await
            .map_err(|e| RpcError::NetworkError(e.to_string()))?;
        
        let response_text = response.text().await.map_err(|e| RpcError::NetworkError(e.to_string()))?;
        
        let response_json: serde_json::Value = serde_json::from_str(&response_text)
            .map_err(|e| RpcError::ParseError(e.to_string()))?;
        
        if let Some(error) = response_json.get("error") {
            let message = error.get("message").and_then(|m| m.as_str()).unwrap_or("Unknown error");
            return Err(RpcError::RpcError(message.to_string()));
        }
        
        let result = response_json.get("result")
            .ok_or_else(|| RpcError::RpcError("No result in response".to_string()))?;
        
        serde_json::from_value(result.clone())
            .map_err(|e| RpcError::ParseError(e.to_string()))
    }
}

/// Log structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    #[serde(flatten)]
    pub fields: BTreeMap<String, serde_json::Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionReceipt {
    #[serde(flatten)]
    pub fields: BTreeMap<String, serde_json::Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Block {
    #[serde(flatten)]
    pub fields: BTreeMap<String, serde_json::Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Log {
    pub address: String,
    pub topics: Vec<String>,
    pub data: String,
    pub block_number: String,
    pub transaction_hash: String,
    pub log_index: String,
}

/// Parse U256 from hex string
fn parse_u256(hex_str: &str) -> U256 {
    let hex = hex_str.trim_start_matches("0x");
    if hex.is_empty() {
        return U256::zero();
    }
    let bytes = hex::decode(hex).unwrap_or_default();
    let mut padded = [0u8; 32];
    let offset = 32 - bytes.len();
    padded[offset..].copy_from_slice(&bytes);
    U256::from_big_endian(&padded)
}

/// RPC errors
#[derive(Debug, Clone)]
pub enum RpcError {
    NetworkError(String),
    ParseError(String),
    RpcError(String),
}

impl std::fmt::Display for RpcError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            RpcError::NetworkError(e) => write!(f, "Network error: {}", e),
            RpcError::ParseError(e) => write!(f, "Parse error: {}", e),
            RpcError::RpcError(e) => write!(f, "RPC error: {}", e),
        }
    }
}

impl std::error::Error for RpcError {}

/// EVM Network Manager
pub struct EvmNetworkManager {
    chains: HashMap<u64, EvmChain>,
    clients: HashMap<u64, Arc<RwLock<EvmRpcClient>>>,
}

impl EvmNetworkManager {
    pub fn new() -> Self {
        let chains = get_evm_chains()
            .into_iter()
            .map(|c| (c.chain_id, c))
            .collect();
        
        let clients = HashMap::new();
        
        Self { chains, clients }
    }

    pub fn get_client(&mut self, chain_id: u64) -> Option<Arc<RwLock<EvmRpcClient>>> {
        if let Some(client) = self.clients.get(&chain_id) {
            return Some(client.clone());
        }
        
        if let Some(chain) = self.chains.get(&chain_id) {
            let client = Arc::new(RwLock::new(EvmRpcClient::new(chain.clone())));
            self.clients.insert(chain_id, client.clone());
            return Some(client);
        }
        
        None
    }

    pub fn get_chain(&self, chain_id: u64) -> Option<&EvmChain> {
        self.chains.get(&chain_id)
    }

    pub fn list_chains(&self) -> Vec<&EvmChain> {
        self.chains.values().collect()
    }
}

impl Default for EvmNetworkManager {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_get_chains() {
        let chains = get_evm_chains();
        assert!(chains.len() >= 15);
        assert!(chains.iter().any(|c| c.chain_id == 1)); // Ethereum
        assert!(chains.iter().any(|c| c.chain_id == 56)); // BSC
        assert!(chains.iter().any(|c| c.chain_id == 137)); // Polygon
    }

    #[test]
    fn test_get_chain_by_id() {
        let eth = get_chain_by_id(1).unwrap();
        assert_eq!(eth.name, "Ethereum");
        
        let bsc = get_chain_by_id(56).unwrap();
        assert_eq!(bsc.name, "BNB Smart Chain");
    }
}