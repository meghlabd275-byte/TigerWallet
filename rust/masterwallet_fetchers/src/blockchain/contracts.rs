//! Smart contract interfaces for MasterWallet operations
//! Provides ABI encoding and contract interaction

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Contract interface
#[derive(Debug, Clone)]
pub struct Contract {
    pub address: String,
    pub abi: Vec<Function>,
}

/// Function in a contract ABI
#[derive(Debug, Clone)]
pub struct Function {
    pub name: String,
    pub inputs: Vec<Parameter>,
    pub outputs: Vec<Parameter>,
    pub state_mutability: StateMutability,
}

/// Parameter in function signature
#[derive(Debug, Clone)]
pub struct Parameter {
    pub name: String,
    pub param_type: String,
    pub indexed: bool,
}

/// State mutability
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum StateMutability {
    Pure,
    View,
    Nonpayable,
    Payable,
}

/// ERC-20 Token contract interface
pub fn erc20_abi() -> Vec<Function> {
    vec![
        Function {
            name: "name".to_string(),
            inputs: vec![],
            outputs: vec![Parameter { name: "".to_string(), param_type: "string".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "symbol".to_string(),
            inputs: vec![],
            outputs: vec![Parameter { name: "".to_string(), param_type: "string".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "decimals".to_string(),
            inputs: vec![],
            outputs: vec![Parameter { name: "".to_string(), param_type: "uint8".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "totalSupply".to_string(),
            inputs: vec![],
            outputs: vec![Parameter { name: "".to_string(), param_type: "uint256".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "balanceOf".to_string(),
            inputs: vec![Parameter { name: "owner".to_string(), param_type: "address".to_string(), indexed: false }],
            outputs: vec![Parameter { name: "".to_string(), param_type: "uint256".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "transfer".to_string(),
            inputs: vec![
                Parameter { name: "to".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "amount".to_string(), param_type: "uint256".to_string(), indexed: false },
            ],
            outputs: vec![Parameter { name: "".to_string(), param_type: "bool".to_string(), indexed: false }],
            state_mutability: StateMutability::Nonpayable,
        },
        Function {
            name: "approve".to_string(),
            inputs: vec![
                Parameter { name: "spender".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "amount".to_string(), param_type: "uint256".to_string(), indexed: false },
            ],
            outputs: vec![Parameter { name: "".to_string(), param_type: "bool".to_string(), indexed: false }],
            state_mutability: StateMutability::Nonpayable,
        },
        Function {
            name: "transferFrom".to_string(),
            inputs: vec![
                Parameter { name: "from".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "to".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "amount".to_string(), param_type: "uint256".to_string(), indexed: false },
            ],
            outputs: vec![Parameter { name: "".to_string(), param_type: "bool".to_string(), indexed: false }],
            state_mutability: StateMutability::Nonpayable,
        },
        Function {
            name: "allowance".to_string(),
            inputs: vec![
                Parameter { name: "owner".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "spender".to_string(), param_type: "address".to_string(), indexed: false },
            ],
            outputs: vec![Parameter { name: "".to_string(), param_type: "uint256".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
    ]
}

/// ERC-721 NFT contract interface
pub fn erc721_abi() -> Vec<Function> {
    vec![
        Function {
            name: "name".to_string(),
            inputs: vec![],
            outputs: vec![Parameter { name: "".to_string(), param_type: "string".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "symbol".to_string(),
            inputs: vec![],
            outputs: vec![Parameter { name: "".to_string(), param_type: "string".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "ownerOf".to_string(),
            inputs: vec![Parameter { name: "tokenId".to_string(), param_type: "uint256".to_string(), indexed: false }],
            outputs: vec![Parameter { name: "".to_string(), param_type: "address".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "balanceOf".to_string(),
            inputs: vec![Parameter { name: "owner".to_string(), param_type: "address".to_string(), indexed: false }],
            outputs: vec![Parameter { name: "".to_string(), param_type: "uint256".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "safeTransferFrom".to_string(),
            inputs: vec![
                Parameter { name: "from".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "to".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "tokenId".to_string(), param_type: "uint256".to_string(), indexed: false },
            ],
            outputs: vec![],
            state_mutability: StateMutability::Nonpayable,
        },
    ]
}

/// ERC-1155 Multi-token contract interface
pub fn erc1155_abi() -> Vec<Function> {
    vec![
        Function {
            name: "balanceOf".to_string(),
            inputs: vec![
                Parameter { name: "account".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "id".to_string(), param_type: "uint256".to_string(), indexed: false },
            ],
            outputs: vec![Parameter { name: "".to_string(), param_type: "uint256".to_string(), indexed: false }],
            state_mutability: StateMutability::View,
        },
        Function {
            name: "safeTransferFrom".to_string(),
            inputs: vec![
                Parameter { name: "from".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "to".to_string(), param_type: "address".to_string(), indexed: false },
                Parameter { name: "id".to_string(), param_type: "uint256".to_string(), indexed: false },
                Parameter { name: "amount".to_string(), param_type: "uint256".to_string(), indexed: false },
                Parameter { name: "data".to_string(), param_type: "bytes".to_string(), indexed: false },
            ],
            outputs: vec![],
            state_mutability: StateMutability::Nonpayable,
        },
    ]
}

/// Contract registry for known contracts
pub struct ContractRegistry {
    contracts: HashMap<String, Contract>,
}

impl ContractRegistry {
    pub fn new() -> Self {
        let mut contracts = HashMap::new();
        
        // ERC-20
        contracts.insert("ERC20".to_string(), Contract {
            address: "".to_string(),
            abi: erc20_abi(),
        });
        
        // ERC-721
        contracts.insert("ERC721".to_string(), Contract {
            address: "".to_string(),
            abi: erc721_abi(),
        });
        
        // ERC-1155
        contracts.insert("ERC1155".to_string(), Contract {
            address: "".to_string(),
            abi: erc1155_abi(),
        });
        
        Self { contracts }
    }
    
    pub fn get(&self, name: &str) -> Option<&Contract> {
        self.contracts.get(name)
    }
    
    pub fn register(&mut self, name: &str, address: &str, abi: Vec<Function>) {
        self.contracts.insert(name.to_string(), Contract {
            address: address.to_string(),
            abi,
        });
    }
}

impl Default for ContractRegistry {
    fn default() -> Self {
        Self::new()
    }
}

/// ABI encoder for contract calls
pub struct ABIEncoder;

impl ABIEncoder {
    /// Encode function call data
    pub fn encode(function_name: &str, params: &[&dyn ParamEncoder]) -> Result<Vec<u8>, String> {
        // Simplified ABI encoding
        // In production, use proper Solidity ABI encoding
        
        let mut result = Vec::new();
        
        // Function selector (first 4 bytes of keccak256 of function signature)
        let selector = Self::function_selector(function_name);
        result.extend_from_slice(&selector);
        
        // Encode parameters
        for param in params {
            result.extend_from_slice(&param.encode());
        }
        
        Ok(result)
    }
    
    /// Generate function selector
    pub fn function_selector(function_name: &str) -> [u8; 4] {
        // Simplified - in production use proper keccak256
        let mut hash = [0u8; 4];
        let bytes = function_name.as_bytes();
        for (i, &b) in bytes.iter().take(4).enumerate() {
            hash[i] = b;
        }
        hash
    }
}

/// Interface for parameter encoding
pub trait ParamEncoder {
    fn encode(&self) -> Vec<u8>;
}

/// Address parameter encoder
pub struct AddressEncoder(pub String);

impl ParamEncoder for AddressEncoder {
    fn encode(&self) -> Vec<u8> {
        let mut result = vec![0u8; 32];
        let addr = &self.0[2..]; // Remove 0x prefix
        let bytes = hex::decode(addr).unwrap_or_default();
        result[32 - bytes.len()..].copy_from_slice(&bytes);
        result
    }
}

/// Uint256 parameter encoder
pub struct Uint256Encoder(pub String);

impl ParamEncoder for Uint256Encoder {
    fn encode(&self) -> Vec<u8> {
        let mut result = vec![0u8; 32];
        let bytes = hex::decode(&self.0[2..]).unwrap_or_default();
        result[32 - bytes.len()..].copy_from_slice(&bytes);
        result
    }
}
