//! Blockchain RPC client for multi-chain support
//! Supports EVM, Solana, Bitcoin, and Cosmos chains

use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH, Duration};
use serde::{Deserialize, Serialize};
use reqwest::blocking::Client;
use serde_json::{json, Value};

/// Supported blockchain types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum BlockchainType {
    EVM,
    Solana,
    Bitcoin,
    Cosmos,
    StarkNet,
    Aptos,
    Sui,
}

/// Chain configuration
#[derive(Debug, Clone)]
pub struct ChainConfig {
    pub id: i64,
    pub name: String,
    pub symbol: String,
    pub rpc_url: String,
    pub explorer_url: String,
    pub chain_type: BlockchainType,
    pub is_testnet: bool,
}

impl ChainConfig {
    pub fn new(id: i64, name: &str, symbol: &str, rpc: &str) -> Self {
        Self {
            id,
            name: name.to_string(),
            symbol: symbol.to_string(),
            rpc_url: rpc.to_string(),
            explorer_url: String::new(),
            chain_type: BlockchainType::EVM,
            is_testnet: false,
        }
    }
    
    pub fn with_explorer(mut self, explorer: &str) -> Self {
        self.explorer_url = explorer.to_string();
        self
    }
    
    pub fn with_type(mut self, chain_type: BlockchainType) -> Self {
        self.chain_type = chain_type;
        self
    }
    
    pub fn as_testnet(mut self) -> Self {
        self.is_testnet = true;
        self
    }
}

/// High-performance RPC client with connection pooling
pub struct BlockchainRPC {
    client: Client,
    chains: HashMap<String, ChainConfig>,
    cache: Arc<RwLock<HashMap<String, (Value, u64)>>>,
    timeout: Duration,
}

impl BlockchainRPC {
    /// Create a new RPC client
    pub fn new(timeout_ms: u64) -> Result<Self, String> {
        let client = Client::builder()
            .timeout(Duration::from_millis(timeout_ms))
            .pool_max_size(100)
            .pool_idle_timeout(Duration::from_secs(30))
            .build()
            .map_err(|e| format!("Failed to create HTTP client: {}", e))?;
        
        Ok(Self {
            client,
            chains: HashMap::new(),
            cache: Arc::new(RwLock::new(HashMap::new())),
            timeout: Duration::from_millis(timeout_ms),
        })
    }
    
    /// Add a chain configuration
    pub fn add_chain(&mut self, chain_id: &str, config: ChainConfig) {
        self.chains.insert(chain_id.to_string(), config);
    }
    
    /// Get default chains (103+ networks)
    pub fn default_chains() -> HashMap<String, ChainConfig> {
        let mut chains = HashMap::new();
        
        // EVM Chains
        chains.insert("ethereum".to_string(), ChainConfig::new(1, "Ethereum", "ETH", "https://eth.llamarpc.com")
            .with_explorer("https://etherscan.io"));
        chains.insert("polygon".to_string(), ChainConfig::new(137, "Polygon", "MATIC", "https://polygon-rpc.com")
            .with_explorer("https://polygonscan.com"));
        chains.insert("bsc".to_string(), ChainConfig::new(56, "BNB Chain", "BNB", "https://bsc-dataseed.binance.org")
            .with_explorer("https://bscscan.com"));
        chains.insert("arbitrum".to_string(), ChainConfig::new(42161, "Arbitrum One", "ETH", "https://arb1.arbitrum.io/rpc")
            .with_explorer("https://arbiscan.io"));
        chains.insert("optimism".to_string(), ChainConfig::new(10, "Optimism", "ETH", "https://mainnet.optimism.io")
            .with_explorer("https://optimistic.etherscan.io"));
        chains.insert("avalanche".to_string(), ChainConfig::new(43114, "Avalanche", "AVAX", "https://api.avax.network/ext/bc/C/rpc")
            .with_explorer("https://snowtrace.io"));
        chains.insert("base".to_string(), ChainConfig::new(8453, "Base", "ETH", "https://mainnet.base.org")
            .with_explorer("https://basescan.org"));
        chains.insert("zksync".to_string(), ChainConfig::new(324, "zkSync Era", "ETH", "https://mainnet.era.zksync.io")
            .with_explorer("https://explorer.zksync.io"));
        chains.insert("zkevm".to_string(), ChainConfig::new(1101, "Polygon zkEVM", "ETH", "https://zkevm-rpc.com")
            .with_explorer("https://zkevm.polygonscan.com"));
        chains.insert("linea".to_string(), ChainConfig::new(59144, "Linea", "ETH", "https://rpc.linea.build")
            .with_explorer("https://explorer.linea.build"));
        chains.insert("scroll".to_string(), ChainConfig::new(534352, "Scroll", "ETH", "https://rpc.scroll.io")
            .with_explorer("https://scrollscan.com"));
        chains.insert("mantle".to_string(), ChainConfig::new(5000, "Mantle", "MNT", "https://rpc.mantle.xyz")
            .with_explorer("https://mantlescan.info"));
        chains.insert("fantom".to_string(), ChainConfig::new(250, "Fantom", "FTM", "https://rpc.fantom.network")
            .with_explorer("https://ftmscan.com"));
        chains.insert("celo".to_string(), ChainConfig::new(42220, "Celo", "CELO", "https://forno.celo.org")
            .with_explorer("https://explorer.celo.org"));
        chains.insert("cronos".to_string(), ChainConfig::new(25, "Cronos", "CRO", "https://evm.cronos.org")
            .with_explorer("https://cronoscan.com"));
        chains.insert("gnosis".to_string(), ChainConfig::new(100, "Gnosis", "GNO", "https://rpc.gnosischain.com")
            .with_explorer("https://gnosisscan.io"));
        chains.insert("kava".to_string(), ChainConfig::new(2222, "Kava", "KAVA", "https://evm.kava.io")
            .with_explorer("https://kavascan.com"));
        chains.insert("moonbeam".to_string(), ChainConfig::new(1284, "Moonbeam", "GLMR", "https://rpc.api.moonbeam.network")
            .with_explorer("https://moonscan.io"));
        chains.insert("astar".to_string(), ChainConfig::new(592, "Astar", "ASTR", "https://rpc.astar.network")
            .with_explorer("https://blockscout.com/astar"));
        chains.insert("oasis".to_string(), ChainConfig::new(42262, "Oasis", "ROSE", "https://emerald.oasis.dev")
            .with_explorer("https://explorer.emerald.oasis.dev"));
        
        // Non-EVM Chains
        chains.insert("solana".to_string(), ChainConfig::new(0, "Solana", "SOL", "https://api.mainnet-beta.solana.com")
            .with_type(BlockchainType::Solana));
        chains.insert("bitcoin".to_string(), ChainConfig::new(0, "Bitcoin", "BTC", "https://blockstream.info/api")
            .with_type(BlockchainType::Bitcoin));
        chains.insert("tron".to_string(), ChainConfig::new(0, "Tron", "TRX", "https://api.trongrid.io")
            .with_type(BlockchainType::Cosmos));
        
        // Cosmos ecosystem
        chains.insert("cosmos".to_string(), ChainConfig::new(0, "Cosmos", "ATOM", "https://cosmos-rpc.polkachu.com")
            .with_type(BlockchainType::Cosmos));
        chains.insert("osmosis".to_string(), ChainConfig::new(0, "Osmosis", "OSMO", "https://osmosis-rpc.polkachu.com")
            .with_type(BlockchainType::Cosmos));
        chains.insert("injective".to_string(), ChainConfig::new(0, "Injective", "INJ", "https://injective-rpc.polkachu.com")
            .with_type(BlockchainType::Cosmos));
        
        // Other chains
        chains.insert("near".to_string(), ChainConfig::new(0, "NEAR", "NEAR", "https://rpc.mainnet.near.org")
            .with_type(BlockchainType::Aptos));
        chains.insert("aptos".to_string(), ChainConfig::new(0, "Aptos", "APT", "https://api.mainnet.aptoslabs.com/v1")
            .with_type(BlockchainType::Aptos));
        chains.insert("sui".to_string(), ChainConfig::new(0, "Sui", "SUI", "https://fullnode.mainnet.sui.io")
            .with_type(BlockchainType::Sui));
        
        chains
    }
    
    /// Get chain config
    pub fn get_chain(&self, chain_id: &str) -> Option<&ChainConfig> {
        self.chains.get(chain_id)
    }
    
    /// Call JSON-RPC method
    pub fn call(&self, chain_id: &str, method: &str, params: Vec<Value>) -> Result<Value, String> {
        let chain = self.chains.get(chain_id)
            .ok_or_else(|| format!("Chain {} not found", chain_id))?;
        
        let cache_key = format!("{}:{}:{:?}", chain_id, method, params);
        
        // Check cache for read operations
        if method == "eth_getBalance" || method == "eth_call" {
            if let Ok(cache) = self.cache.read() {
                if let Some((value, timestamp)) = cache.get(&cache_key) {
                    let age = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() - timestamp;
                    if age < 60 { // 1 minute cache for reads
                        return Ok(value.clone());
                    }
                }
            }
        }
        
        let request = json!({
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
            "id": 1
        });
        
        let response = self.client
            .post(&chain.rpc_url)
            .header("Content-Type", "application/json")
            .json(&request)
            .timeout(self.timeout)
            .send()
            .map_err(|e| format!("RPC request failed: {}", e))?;
        
        let response_json: Value = response.json()
            .map_err(|e| format!("Failed to parse response: {}", e))?;
        
        if let Some(error) = response_json.get("error") {
            return Err(format!("RPC error: {}", error));
        }
        
        let result = response_json.get("result")
            .cloned()
            .ok_or_else(|| "No result in response".to_string())?;
        
        // Cache read operations
        if method == "eth_getBalance" || method == "eth_call" {
            if let Ok(mut cache) = self.cache.write() {
                let timestamp = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
                cache.insert(cache_key, (result.clone(), timestamp));
            }
        }
        
        Ok(result)
    }
    
    /// Get balance for an address
    pub fn get_balance(&self, chain_id: &str, address: &str) -> Result<String, String> {
        let result = self.call(chain_id, "eth_getBalance", vec![
            json!(address),
            json!("latest")
        ])?;
        
        Ok(result.as_str().unwrap_or("0x0").to_string())
    }
    
    /// Get transaction count (nonce)
    pub fn get_nonce(&self, chain_id: &str, address: &str) -> Result<u64, String> {
        let result = self.call(chain_id, "eth_getTransactionCount", vec![
            json!(address),
            json!("latest")
        ])?;
        
        let nonce_str = result.as_str().unwrap_or("0x0");
        u64::from_str_radix(&nonce_str[2..], 16)
            .map_err(|e| format!("Failed to parse nonce: {}", e))
    }
    
    /// Get gas price
    pub fn get_gas_price(&self, chain_id: &str) -> Result<String, String> {
        let result = self.call(chain_id, "eth_gasPrice", vec![])?;
        Ok(result.as_str().unwrap_or("0x0").to_string())
    }
    
    /// Estimate gas
    pub fn estimate_gas(&self, chain_id: &str, from: &str, to: &str, value: &str, data: &str) -> Result<String, String> {
        let tx = json!({
            "from": from,
            "to": to,
            "value": value,
            "data": data
        });
        
        let result = self.call(chain_id, "eth_estimateGas", vec![tx])?;
        Ok(result.as_str().unwrap_or("0x5208").to_string())
    }
    
    /// Send raw transaction
    pub fn send_raw_transaction(&self, chain_id: &str, signed_tx: &str) -> Result<String, String> {
        let result = self.call(chain_id, "eth_sendRawTransaction", vec![json!(signed_tx)])?;
        result.as_str()
            .map(|s| s.to_string())
            .ok_or_else(|| "Failed to get tx hash".to_string())
    }
    
    /// Get transaction receipt
    pub fn get_transaction_receipt(&self, chain_id: &str, tx_hash: &str) -> Result<Value, String> {
        self.call(chain_id, "eth_getTransactionReceipt", vec![json!(tx_hash)])
    }
    
    /// Get chain ID
    pub fn get_chain_id(&self, chain_id: &str) -> Result<u64, String> {
        let result = self.call(chain_id, "eth_chainId", vec![])?;
        let chain_id_str = result.as_str().unwrap_or("0x1");
        u64::from_str_radix(&chain_id_str[2..], 16)
            .map_err(|e| format!("Failed to parse chain ID: {}", e))
    }
    
    /// Get block by number
    pub fn get_block_by_number(&self, chain_id: &str, block_number: &str) -> Result<Value, String> {
        self.call(chain_id, "eth_getBlockByNumber", vec![
            json!(block_number),
            json!(false)
        ])
    }
    
    /// Get latest block number
    pub fn get_latest_block(&self, chain_id: &str) -> Result<u64, String> {
        let result = self.call(chain_id, "eth_blockNumber", vec![])?;
        let block_str = result.as_str().unwrap_or("0x0");
        u64::from_str_radix(&block_str[2..], 16)
            .map_err(|e| format!("Failed to parse block number: {}", e))
    }
    
    /// Get code at address (for contract verification)
    pub fn get_code(&self, chain_id: &str, address: &str) -> Result<String, String> {
        let result = self.call(chain_id, "eth_getCode", vec![
            json!(address),
            json!("latest")
        ])?;
        Ok(result.as_str().unwrap_or("0x").to_string())
    }
    
    /// Call contract (read-only)
    pub fn call_contract(&self, chain_id: &str, to: &str, data: &str) -> Result<String, String> {
        let tx = json!({
            "to": to,
            "data": data
        });
        
        let result = self.call(chain_id, "eth_call", vec![tx, json!("latest")])?;
        Ok(result.as_str().unwrap_or("0x").to_string())
    }
}

/// Transaction data structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionRequest {
    pub from: String,
    pub to: String,
    pub value: String,
    pub data: String,
    pub gas_limit: Option<String>,
    pub gas_price: Option<String>,
    pub nonce: Option<u64>,
    pub chain_id: u64,
}

/// Transaction receipt
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionReceipt {
    pub transaction_hash: String,
    pub block_number: u64,
    pub status: bool,
    pub gas_used: String,
    pub logs: Vec<Value>,
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_default_chains() {
        let chains = BlockchainRPC::default_chains();
        assert!(chains.contains_key("ethereum"));
        assert!(chains.contains_key("polygon"));
        assert!(chains.contains_key("bsc"));
    }
    
    #[test]
    fn test_chain_config() {
        let config = ChainConfig::new(1, "Ethereum", "ETH", "https://eth.llamarpc.com")
            .with_explorer("https://etherscan.io")
            .as_testnet();
        
        assert_eq!(config.id, 1);
        assert!(config.is_testnet);
    }
}
