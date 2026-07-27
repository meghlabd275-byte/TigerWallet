//! Starknet Address Types
//! 
//! Implementation of Starknet address formats and validation.

use std::fmt;
use hex;

/// Starknet Address (Felt252)
#[derive(Clone, PartialEq, Eq, Hash)]
pub struct StarknetAddress {
    /// The 252-bit felt value
    value: [u8; 32],
    /// Whether this is a contract address
    is_contract: bool,
}

impl StarknetAddress {
    /// Create from hex string
    pub fn from_hex(hex_str: &str) -> Result<Self, AddressError> {
        let bytes = hex::decode(hex_str.trim_start_matches("0x"))
            .map_err(|_| AddressError::InvalidHex)?;
        
        if bytes.len() > 32 {
            return Err(AddressError::TooLong);
        }
        
        let mut value = [0u8; 32];
        value[32 - bytes.len()..].copy_from_slice(&bytes);
        
        Ok(Self { value, is_contract: false })
    }
    
    /// Create from felt252 value
    pub fn from_felt252(mut value: [u8; 32]) -> Self {
        // Validate value is within field
        // Starknet field: p = 2^251 + 17 * 2^192 + 1
        let max_felt: [u8; 32] = [
            0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        ];
        
        // Validate within field
        let mut is_contract = false;
        for i in 0..32 {
            if value[i] < max_felt[i] {
                break;
            } else if value[i] > max_felt[i] {
                // Set to zero if out of field
                value = [0u8; 32];
                break;
            }
        }
        
        // Contract addresses are deterministic (hash of contract code)
        if value.iter().all(|&b| b == 0) {
            is_contract = false;
        }
        
        Self { value, is_contract }
    }
    
    /// Create from Starknet public key
    pub fn from_public_key(public_key: &[u8; 32]) -> Self {
        use sha3::{Keccak256, Digest};
        
        let mut hasher = Keccak256::new();
        hasher.update(b"starknet");
        hasher.update(public_key);
        let result = hasher.finalize();
        
        let mut value = [0u8; 32];
        value.copy_from_slice(&result);
        Self { value, is_contract: false }
    }
    
    /// Get as hex string (0x...)
    pub fn to_hex(&self) -> String {
        format!("0x{}", hex::encode(self.value))
    }
    
    /// Get as felt252 bytes
    pub fn to_felt252(&self) -> [u8; 32] {
        self.value
    }
    
    /// Get short address (first 10 chars)
    pub fn to_short(&self) -> String {
        let hex_str = hex::encode(self.value);
        format!("0x{}", &hex_str[..10])
    }
    
    /// Validate address format
    pub fn is_valid(&self) -> bool {
        // Check not zero
        if self.value.iter().all(|&b| b == 0) {
            return false;
        }
        
        // Check within field (simplified)
        // In production: verify < 2^251 + 17*2^192 + 1
        true
    }
    
    /// Set contract flag
    pub fn set_contract(&mut self, is_contract: bool) {
        self.is_contract = is_contract;
    }
    
    /// Check if contract
    pub fn is_contract(&self) -> bool {
        self.is_contract
    }
}

impl fmt::Debug for StarknetAddress {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "StarknetAddress({})", self.to_hex())
    }
}

impl fmt::Display for StarknetAddress {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.to_hex())
    }
}

impl From<[u8; 32]> for StarknetAddress {
    fn from(value: [u8; 32]) -> Self {
        Self::from_felt252(value)
    }
}

impl From<&str> for StarknetAddress {
    fn from(s: &str) -> Self {
        Self::from_hex(s).expect("Invalid Starknet address")
    }
}

impl From<String> for StarknetAddress {
    fn from(s: String) -> Self {
        Self::from_hex(&s).expect("Invalid Starknet address")
    }
}

/// Address errors
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum AddressError {
    InvalidHex,
    TooLong,
    InvalidFormat,
    ZeroAddress,
}

impl fmt::Display for AddressError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AddressError::InvalidHex => write!(f, "Invalid hex string"),
            AddressError::TooLong => write!(f, "Address too long"),
            AddressError::InvalidFormat => write!(f, "Invalid address format"),
            AddressError::ZeroAddress => write!(f, "Zero address not allowed"),
        }
    }
}

impl std::error::Error for AddressError {}

/// Starknet chain IDs
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum StarknetChainId {
    /// Mainnet
    Mainnet,
    /// Goerli testnet
    Goerli,
    /// Goerli 2 testnet
    Goerli2,
    /// Sepolia testnet
    Sepolia,
    /// Local development
    Local,
}

impl StarknetChainId {
    /// Get chain ID as felt
    pub fn to_felt(&self) -> [u8; 32] {
        match self {
            StarknetChainId::Mainnet => {
                let mut id = [0u8; 32];
                id[31] = 0x01; // SN_MAINNET
                id
            }
            StarknetChainId::Goerli => {
                let mut id = [0u8; 32];
                id[31] = 0x03; // SN_GOERLI
                id
            }
            StarknetChainId::Goerli2 => {
                let mut id = [0u8; 32];
                id[31] = 0x05; // SN_GOERLI2
                id
            }
            StarknetChainId::Sepolia => {
                let mut id = [0u8; 32];
                id[31] = 0x06; // SN_SEPOLIA
                id
            }
            StarknetChainId::Local => {
                let mut id = [0u8; 32];
                id[31] = 0x00; // Local
                id
            }
        }
    }
    
    /// Get RPC URL prefix
    pub fn rpc_prefix(&self) -> &'static str {
        match self {
            StarknetChainId::Mainnet => "mainnet",
            StarknetChainId::Goerli => "goerli",
            StarknetChainId::Goerli2 => "goerli2",
            StarknetChainId::Sepolia => "sepolia",
            StarknetChainId::Local => "localhost",
        }
    }
}

impl Default for StarknetChainId {
    fn default() -> Self {
        StarknetChainId::Mainnet
    }
}
