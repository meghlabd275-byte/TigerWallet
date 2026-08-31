//! Core zkSync Era types (EVM-compatible)

use serde::{Deserialize, Serialize};
use thiserror::Error;

/// 20-byte EVM address
pub type Address = [u8; 20];

/// 32-byte hash
pub type H256 = [u8; 32];

#[derive(Error, Debug)]
pub enum ZksyncError {
    #[error("Invalid key: {0}")]
    InvalidKey(String),

    #[error("Invalid address: {0}")]
    InvalidAddress(String),

    #[error("Invalid transaction: {0}")]
    InvalidTransaction(String),

    #[error("RPC error: {0}")]
    RpcError(String),

    #[error("Encoding error: {0}")]
    EncodingError(String),
}

/// Transaction status on zkSync Era
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum TransactionStatus {
    Pending,
    Included,
    Verified,
    Failed,
}

/// Transaction receipt
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct TransactionReceipt {
    pub transaction_hash: H256,
    pub status: TransactionStatusWire,
    pub block_number: u64,
    pub gas_used: u64,
    pub effective_gas_price: u64,
    pub contract_address: Option<Address>,
    pub logs: Vec<Log>,
}

/// Wire-friendly status wrapper for serde
#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize)]
pub enum TransactionStatusWire {
    #[default]
    Unknown,
    Success,
    Reverted,
}

/// EVM log entry
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Log {
    pub address: Address,
    pub topics: Vec<H256>,
    pub data: Vec<u8>,
    pub log_index: u64,
}

/// Block summary
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Block {
    pub number: u64,
    pub hash: H256,
    pub parent_hash: H256,
    pub timestamp: u64,
    pub transaction_count: usize,
}
