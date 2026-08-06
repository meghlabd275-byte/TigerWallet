//! Transaction types and structures

use serde::{Deserialize, Serialize};
use std::fmt;
use std::ops::{Add, Sub, Mul, Div, Rem, BitAnd, BitOr, BitXor, Shl, Shr};
use std::cmp::{PartialEq, PartialOrd, Eq, Ord, Ordering};

/// Transaction type enumeration
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[repr(u8)]
pub enum TransactionType {
    Transfer = 0,
    Swap = 1,
    Stake = 2,
    Unstake = 3,
    Mint = 4,
    Burn = 5,
    Approve = 6,
    TransferFrom = 7,
    Bridge = 8,
    NftTransfer = 9,
    Unknown = 255,
}

impl Default for TransactionType {
    fn default() -> Self {
        TransactionType::Unknown
    }
}

/// Transaction status
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[repr(u8)]
pub enum TransactionStatus {
    Pending = 0,
    Confirmed = 1,
    Failed = 2,
    Flagged = 3,
    Cancelled = 4,
}

impl Default for TransactionStatus {
    fn default() -> Self {
        TransactionStatus::Pending
    }
}

/// Blockchain chain identifiers
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[repr(u8)]
pub enum Chain {
    Ethereum = 0,
    Polygon = 1,
    Bsc = 2,
    Avalanche = 3,
    Solana = 4,
    Arbitrum = 5,
    Optimism = 6,
    Base = 7,
    Bitcoin = 8,
}

impl Default for Chain {
    fn default() -> Self {
        Chain::Ethereum
    }
}

/// 256-bit unsigned integer for token amounts
#[derive(Clone, Serialize, Deserialize)]
pub struct Uint256 {
    words: [u64; 4],
}

impl Uint256 {
    pub fn zero() -> Self {
        Self { words: [0, 0, 0, 0] }
    }
    
    pub fn from_u64(val: u64) -> Self {
        Self { words: [val, 0, 0, 0] }
    }
    
    pub fn from_hex(hex: &str) -> Option<Self> {
        let hex = hex.strip_prefix("0x").unwrap_or(hex);
        if hex.len() > 64 {
            return None;
        }
        
        let mut words = [0u64; 4];
        for (i, chunk) in hex.as_bytes().chunks(16).enumerate() {
            if i >= 4 {
                break;
            }
            let s = std::str::from_utf8(chunk).ok()?;
            words[3 - i] = u64::from_str_radix(s, 16).ok()?;
        }
        
        Some(Self { words })
    }
    
    pub fn low64(&self) -> u64 {
        self.words[0]
    }
    
    pub fn is_zero(&self) -> bool {
        self.words.iter().all(|&w| w == 0)
    }
    
    pub fn to_string(&self) -> String {
        format!("0x{:032x}{:016x}{:016x}{:016x}", 
            self.words[3], self.words[2], self.words[1], self.words[0])
    }
}

impl Default for Uint256 {
    fn default() -> Self {
        Self::zero()
    }
}

impl fmt::Debug for Uint256 {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "Uint256({})", self.to_string())
    }
}

/// Ethereum-style address (20 bytes)
#[derive(Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub struct Address {
    data: [u8; 32],
}

impl Address {
    pub fn zero() -> Self {
        Self { data: [0; 32] }
    }
    
    pub fn from_hex(hex: &str) -> Self {
        let mut addr = Self::zero();
        let hex = hex.strip_prefix("0x").unwrap_or(hex);
        
        for (i, chunk) in hex.as_bytes().chunks(2).take(20).enumerate() {
            let s = std::str::from_utf8(chunk).unwrap_or("00");
            addr.data[31 - i] = u8::from_str_radix(s, 16).unwrap_or(0);
        }
        
        addr
    }
    
    pub fn to_hex(&self) -> String {
        let bytes: Vec<u8> = self.data.iter().rev().take(20).cloned().collect();
        format!("0x{}", bytes.iter().map(|b| format!("{:02x}", b)).collect::<String>())
    }
    
    pub fn is_zero(&self) -> bool {
        self.data.iter().all(|&b| b == 0)
    }
}

impl Default for Address {
    fn default() -> Self {
        Self::zero()
    }
}

impl fmt::Debug for Address {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "Address({})", self.to_hex())
    }
}

/// Transaction hash (32 bytes)
#[derive(Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub struct TxHash {
    data: [u8; 32],
}

impl TxHash {
    pub fn zero() -> Self {
        Self { data: [0; 32] }
    }
    
    pub fn from_hex(hex: &str) -> Self {
        let mut hash = Self::zero();
        let hex = hex.strip_prefix("0x").unwrap_or(hex);
        
        for (i, chunk) in hex.as_bytes().chunks(2).take(32).enumerate() {
            let s = std::str::from_utf8(chunk).unwrap_or("00");
            hash.data[31 - i] = u8::from_str_radix(s, 16).unwrap_or(0);
        }
        
        hash
    }
    
    pub fn to_hex(&self) -> String {
        format!("0x{}", self.data.iter().rev().map(|b| format!("{:02x}", b)).collect::<String>())
    }
}

impl Default for TxHash {
    fn default() -> Self {
        Self::zero()
    }
}

impl fmt::Debug for TxHash {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "TxHash({})", self.to_hex())
    }
}

/// High-precision timestamp (nanoseconds)
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub struct Timestamp {
    pub nanoseconds: u64,
}

impl Timestamp {
    pub fn now() -> Self {
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos() as u64;
        Self { nanoseconds: now }
    }
    
    pub fn from_secs(secs: u64) -> Self {
        Self { nanoseconds: secs * 1_000_000_000 }
    }
    
    pub fn to_micros(&self) -> u64 {
        self.nanoseconds / 1000
    }
}

impl Default for Timestamp {
    fn default() -> Self {
        Self::now()
    }
}

/// Transaction structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub hash: TxHash,
    pub from: Address,
    pub to: Address,
    pub amount: u64,
    pub fee: u64,
    pub chain: Chain,
    #[serde(default)]
    pub txn_type: TransactionType,
    #[serde(default)]
    pub status: TransactionStatus,
    pub nonce: u64,
    pub block_number: u64,
    pub created_at: Timestamp,
    pub processed_at: Timestamp,
    pub confirmed_at: Option<Timestamp>,
    pub gas_limit: u64,
    pub gas_used: u64,
    pub gas_price: u64,
    pub data: Vec<u8>,
    pub memo: String,
    pub verified: bool,
    pub processed: bool,
}

impl Transaction {
    pub fn new(from: Address, to: Address, amount: u64, chain: Chain) -> Self {
        let hash = TxHash::from_hex(&format!("0x{:064x}", rand::random::<u256>()));
        
        Self {
            hash,
            from,
            to,
            amount,
            fee: 0,
            chain,
            txn_type: TransactionType::Transfer,
            status: TransactionStatus::Pending,
            nonce: 0,
            block_number: 0,
            created_at: Timestamp::now(),
            processed_at: Timestamp::now(),
            confirmed_at: None,
            gas_limit: 21000,
            gas_used: 0,
            gas_price: 0,
            data: Vec::new(),
            memo: String::new(),
            verified: false,
            processed: false,
        }
    }
    
    pub fn is_valid(&self) -> bool {
        !self.from.is_zero() && self.gas_price > 0
    }
}

impl Default for Transaction {
    fn default() -> Self {
        Self::new(Address::zero(), Address::zero(), 0, Chain::Ethereum)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_address_from_hex() {
        let addr = Address::from_hex("0x1234567890123456789012345678901234567890");
        assert_eq!(addr.to_hex(), "0x1234567890123456789012345678901234567890");
    }
    
    #[test]
    fn test_uint256_operations() {
        let a = Uint256::from_u64(100);
        let b = Uint256::from_u64(50);
        assert_eq!(a.low64(), 100);
    }
}
