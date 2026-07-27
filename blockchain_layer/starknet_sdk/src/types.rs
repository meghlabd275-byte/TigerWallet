//! Starknet Types
//! 
//! Common types for Starknet.

use serde::{Deserialize, Serialize};

/// Block information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Block {
    #[serde(rename = "block_hash")]
    pub block_hash: Option<String>,
    #[serde(rename = "block_number")]
    pub block_number: Option<u64>,
    #[serde(rename = "parent_hash")]
    pub parent_hash: String,
    #[serde(rename = "timestamp")]
    pub timestamp: u64,
    #[serde(rename = "sequencer_address")]
    pub sequencer_address: String,
    #[serde(rename = "transactions")]
    pub transactions: Vec<String>,
}

/// Transaction information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    #[serde(rename = "transaction_hash")]
    pub transaction_hash: String,
    #[serde(rename = "type")]
    pub transaction_type: String,
    #[serde(rename = "sender_address")]
    pub sender_address: String,
    #[serde(rename = "nonce")]
    pub nonce: String,
    #[serde(rename = "max_fee")]
    pub max_fee: String,
    #[serde(rename = "version")]
    pub version: String,
    #[serde(rename = "signature")]
    pub signature: Option<Vec<String>>,
}

/// Contract class (Sierra or Cairo 0)
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ContractClass {
    #[serde(rename = "program")]
    pub program: Option<String>,
    #[serde(rename = "entry_points_by_type")]
    pub entry_points_by_type: Option<EntryPointsByType>,
    #[serde(rename = "abi")]
    pub abi: Option<String>,
    #[serde(rename = "contract_class_version")]
    pub contract_class_version: Option<String>,
}

/// Entry points by type
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct EntryPointsByType {
    #[serde(rename = "EXTERNAL")]
    pub external: Vec<EntryPoint>,
    #[serde(rename = "L1_HANDLER")]
    pub l1_handler: Vec<EntryPoint>,
    #[serde(rename = "CONSTRUCTOR")]
    pub constructor: Vec<EntryPoint>,
}

/// Entry point
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EntryPoint {
    #[serde(rename = "function_idx")]
    pub function_idx: u32,
    #[serde(rename = "selector")]
    pub selector: String,
    #[serde(rename = "offset")]
    pub offset: String,
}

/// Felt252 (31-byte integer)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Felt252(#[serde(with = "hex")] pub Vec<u8>);

impl Felt252 {
    pub fn from_hex(hex: &str) -> Option<Self> {
        let bytes = hex::decode(hex.trim_start_matches("0x")).ok()?;
        if bytes.len() > 32 {
            return None;
        }
        Some(Self(bytes))
    }
    
    pub fn to_hex(&self) -> String {
        format!("0x{}", hex::encode(&self.0))
    }
}

impl Default for Felt252 {
    fn default() -> Self {
        Self(vec![0])
    }
}

/// RPC error
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RpcError {
    pub code: i32,
    pub message: String,
    pub data: Option<serde_json::Value>,
}
