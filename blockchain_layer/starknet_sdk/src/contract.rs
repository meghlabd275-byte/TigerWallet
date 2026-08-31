//! Starknet Contract Utilities
//! 
//! Contract compilation and interaction utilities.

use crate::address::StarknetAddress;
use crate::provider::FunctionCall;

/// Contract ABI type
#[derive(Debug, Clone)]
pub enum AbiType {
    /// Cairo 0 (legacy)
    Legacy,
    /// Cairo 1 (Sierra)
    Sierra,
}

/// Function type in ABI
#[derive(Debug, Clone)]
pub struct FunctionAbi {
    pub name: String,
    pub inputs: Vec<Parameter>,
    pub outputs: Vec<Parameter>,
    pub state_mutability: StateMutability,
}

/// Parameter in function
#[derive(Debug, Clone)]
pub struct Parameter {
    pub name: String,
    pub r#type: String,
}

/// State mutability
#[derive(Debug, Clone, Default)]
pub enum StateMutability {
    #[default]
    View,
    External,
    Payable,
}

/// Contract class (compiled)
#[derive(Debug, Clone)]
pub struct CompiledContract {
    /// ABI
    pub abi: Vec<FunctionAbi>,
    /// Bytecode
    pub bytecode: Vec<u8>,
    /// Compiler version
    pub compiler_version: String,
}

/// Contract factory for deployment
pub struct ContractFactory {
    /// Class hash
    class_hash: [u8; 32],
    /// Deployer address (zero for account deploy)
    deployer: Option<StarknetAddress>,
}

impl ContractFactory {
    /// Create new factory
    pub fn new(class_hash: [u8; 32]) -> Self {
        Self {
            class_hash,
            deployer: None,
        }
    }
    
    /// Set deployer
    pub fn with_deployer(mut self, deployer: StarknetAddress) -> Self {
        self.deployer = Some(deployer);
        self
    }
    
    /// Get deploy transaction data
    pub fn deploy_data(&self, constructor_calldata: Vec<[u8; 32]>, salt: u32) -> ContractDeployData {
        let mut calldata = vec![self.class_hash];
        
        // Add salt
        let mut salt_bytes = [0u8; 32];
        salt_bytes[31] = salt as u8;
        calldata.push(salt_bytes);
        
        // Add constructor calldata length
        let mut len_bytes = [0u8; 32];
        len_bytes[31] = constructor_calldata.len() as u8;
        calldata.push(len_bytes);
        
        // Add constructor calldata
        calldata.extend(constructor_calldata.clone());
        
        // Add deployer (0 for account)
        let deployer_felt = match &self.deployer {
            Some(addr) => addr.to_felt252(),
            None => [0u8; 32],
        };
        calldata.push(deployer_felt);
        
        ContractDeployData {
            class_hash: self.class_hash,
            constructor_calldata,
            salt,
        }
    }
}

/// Contract deploy data
#[derive(Debug, Clone)]
pub struct ContractDeployData {
    pub class_hash: [u8; 32],
    pub constructor_calldata: Vec<[u8; 32]>,
    pub salt: u32,
}

/// Contract interaction helper
pub struct Contract {
    address: StarknetAddress,
    abi: Vec<FunctionAbi>,
}

impl Contract {
    /// Create at address
    pub fn at(address: StarknetAddress) -> Self {
        Self {
            address,
            abi: vec![],
        }
    }
    
    /// Set ABI
    pub fn with_abi(mut self, abi: Vec<FunctionAbi>) -> Self {
        self.abi = abi;
        self
    }
    
    /// Get address
    pub fn address(&self) -> &StarknetAddress {
        &self.address
    }
    
    /// Build function call from ABI
    pub fn function(&self, name: &str) -> Option<FunctionCallBuilder> {
        let func = self.abi.iter().find(|f| f.name == name)?;
        
        Some(FunctionCallBuilder {
            contract: self.address.clone(),
            selector: name.to_string(),
            calldata: vec![],
            inputs: func.inputs.clone(),
        })
    }
}

/// Function call builder
pub struct FunctionCallBuilder {
    contract: StarknetAddress,
    selector: String,
    calldata: Vec<String>,
    inputs: Vec<Parameter>,
}

impl FunctionCallBuilder {
    /// Add argument
    pub fn arg<T: Into<String>>(mut self, value: T) -> Self {
        self.calldata.push(value.into());
        self
    }
    
    /// Build function call
    pub fn build(self) -> FunctionCall {
        // Compute selector hash
        let selector = compute_selector(&self.selector);
        
        FunctionCall {
            contract_address: self.contract.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: self.calldata,
        }
    }
}

/// Compute Starknet function selector (keccak256)
pub fn compute_selector(name: &str) -> [u8; 32] {
    use sha3::{Keccak256, Digest};
    
    let mut hasher = Keccak256::new();
    hasher.update(name.as_bytes());
    let hash = hasher.finalize();
    
    let mut selector = [0u8; 32];
    selector.copy_from_slice(&hash);
    
    selector
}

/// Common contract selectors
pub mod selectors {
    
    
    // ERC-20
    pub const TRANSFER: [u8; 32] = [
        0x84, 0x7e, 0x3d, 0x04, 0x9c, 0x6f, 0x47, 0x7a,
        0xda, 0xf9, 0xc4, 0x5c, 0xd9, 0x20, 0x46, 0x4e,
        0x9e, 0x43, 0x2e, 0x0c, 0xd5, 0x1f, 0x4a, 0xf3,
        0xf2, 0xdc, 0x0a, 0x4e, 0xd1, 0x8e, 0x1f, 0x1e
    ];
    
    pub const TRANSFER_FROM: [u8; 32] = [
        0x36, 0x5c, 0x1e, 0x4c, 0x5e, 0xc4, 0x80, 0x97,
        0x2c, 0x0c, 0x7f, 0x16, 0x3e, 0x4a, 0x77, 0x19,
        0x5a, 0x07, 0x46, 0x82, 0x1f, 0x22, 0x9e, 0x28,
        0xaf, 0x82, 0x94, 0x0f, 0x08, 0x9f, 0x91, 0x52
    ];
    
    pub const APPROVE: [u8; 32] = [
        0x1b, 0x52, 0xf4, 0x3c, 0x4b, 0x1f, 0x58, 0x60,
        0x8f, 0xdd, 0x3a, 0x5e, 0x4c, 0x0f, 0x5f, 0x2c,
        0x1f, 0x1f, 0x5e, 0x51, 0x1d, 0x65, 0x6e, 0xf7,
        0x9b, 0xad, 0x94, 0x46, 0x4a, 0xac, 0xda, 0x0c
    ];
    
    pub const BALANCE_OF: [u8; 32] = [
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x12, 0x31, 0x1e, 0x5c
    ];
    
    pub const TOTAL_SUPPLY: [u8; 32] = [
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x02, 0x9d, 0xa1, 0x8c
    ];
    
    // Account
    pub const EXECUTE: [u8; 32] = [
        0x15, 0xdb, 0x52, 0x9b, 0x92, 0x38, 0x8b, 0x01,
        0x8c, 0xca, 0xaa, 0x44, 0x87, 0x27, 0xce, 0x03,
        0x2e, 0x11, 0x47, 0x95, 0x3b, 0xca, 0x88, 0xf6,
        0x9d, 0xfe, 0x13, 0x85, 0xf9, 0x27, 0x40, 0x1a
    ];
    
    pub const VALIDATE_DEPLOY: [u8; 32] = [
        0x01, 0x2b, 0x20, 0x3e, 0x2e, 0x26, 0x10, 0x47,
        0x69, 0x60, 0x18, 0x1d, 0x78, 0x2c, 0x19, 0x4a,
        0xc6, 0x38, 0x0b, 0x00, 0x6f, 0x3a, 0x9d, 0x81,
        0x9c, 0x78, 0x2e, 0xd3, 0x0f, 0x87, 0x6e, 0x09
    ];
    
    pub const VALIDATE_INVOKE: [u8; 32] = [
        0x00, 0x0f, 0x1f, 0x5e, 0x7c, 0x0a, 0x5d, 0x52,
        0xf8, 0x0b, 0x98, 0x02, 0x1d, 0x09, 0x6a, 0x90,
        0x90, 0x01, 0x05, 0x4f, 0xa1, 0x7c, 0xfa, 0x6b,
        0x53, 0x76, 0x6a, 0x50, 0x0c, 0x1d, 0xfc, 0x5e
    ];
}

/// Known Starknet contract addresses
pub mod addresses {
    use super::*;
    
    /// ETH token (mainnet)
    pub const ETH_MAINNET: &str = "0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc";
    
    /// USDC token (mainnet)
    pub const USDC_MAINNET: &str = "0x053c91253bc968ea4be2b0212f2a2f54630a3b68f45c7b01a3f62a47c0a6e77";
    
    /// USDT token (mainnet)
    pub const USDT_MAINNET: &str = "0x068f5c6a61780768455de69077e5e10625f0e948957cdbb7fc57e79d5a8b1bce";
    
    /// DAI token (mainnet)
    pub const DAI_MAINNET: &str = "0x00da114221cb83fa859dbdb4c44beeaa0bb37c7537ad5ae76f804c34d6391459";
    
    /// Get address from symbol
    pub fn get_token(symbol: &str) -> Option<StarknetAddress> {
        let addr = match symbol.to_uppercase().as_str() {
            "ETH" => ETH_MAINNET,
            "USDC" => USDC_MAINNET,
            "USDT" => USDT_MAINNET,
            "DAI" => DAI_MAINNET,
            _ => return None,
        };
        
        StarknetAddress::from_hex(addr).ok()
    }
}
