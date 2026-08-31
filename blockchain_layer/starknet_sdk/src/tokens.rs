//! Starknet Token Standards
//! 
//! ERC-20, ERC-721, ERC-1155 implementations for Starknet.

use crate::address::StarknetAddress;
use crate::contract::*;
use crate::provider::FunctionCall;

/// ERC-20 Token
pub struct Erc20 {
    address: StarknetAddress,
}

impl Erc20 {
    /// Create at address
    pub fn at(address: StarknetAddress) -> Self {
        Self { address }
    }
    
    /// At known token
    pub fn token(symbol: &str) -> Option<Self> {
        let addr = addresses::get_token(symbol)?;
        Some(Self { address: addr })
    }
    
    /// Get name (call contract)
    pub fn name(&self) -> FunctionCall {
        let selector = compute_selector("name");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![],
        }
    }
    
    /// Get symbol
    pub fn symbol(&self) -> FunctionCall {
        let selector = compute_selector("symbol");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![],
        }
    }
    
    /// Get decimals
    pub fn decimals(&self) -> FunctionCall {
        let selector = compute_selector("decimals");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![],
        }
    }
    
    /// Get total supply
    pub fn total_supply(&self) -> FunctionCall {
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selectors::TOTAL_SUPPLY),
            calldata: vec![],
        }
    }
    
    /// Get balance of
    pub fn balance_of(&self, owner: &StarknetAddress) -> FunctionCall {
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selectors::BALANCE_OF),
            calldata: vec![owner.to_hex()],
        }
    }
    
    /// Transfer
    pub fn transfer(&self, to: &StarknetAddress, amount: u128) -> FunctionCall {
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selectors::TRANSFER),
            calldata: vec![to.to_hex(), format!("0x{:x}", amount)],
        }
    }
    
    /// Transfer from
    pub fn transfer_from(&self, from: &StarknetAddress, to: &StarknetAddress, amount: u128) -> FunctionCall {
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selectors::TRANSFER_FROM),
            calldata: vec![from.to_hex(), to.to_hex(), format!("0x{:x}", amount)],
        }
    }
    
    /// Approve
    pub fn approve(&self, spender: &StarknetAddress, amount: u128) -> FunctionCall {
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selectors::APPROVE),
            calldata: vec![spender.to_hex(), format!("0x{:x}", amount)],
        }
    }
    
    /// Get allowance
    pub fn allowance(&self, owner: &StarknetAddress, spender: &StarknetAddress) -> FunctionCall {
        let selector = compute_selector("allowance");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![owner.to_hex(), spender.to_hex()],
        }
    }
}

/// ERC-721 Token (NFT)
pub struct Erc721 {
    address: StarknetAddress,
}

impl Erc721 {
    /// Create at address
    pub fn at(address: StarknetAddress) -> Self {
        Self { address }
    }
    
    /// Get name
    pub fn name(&self) -> FunctionCall {
        let selector = compute_selector("name");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![],
        }
    }
    
    /// Get symbol
    pub fn symbol(&self) -> FunctionCall {
        let selector = compute_selector("symbol");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![],
        }
    }
    
    /// Get owner of token
    pub fn owner_of(&self, token_id: u256) -> FunctionCall {
        let selector = compute_selector("ownerOf");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![format!("0x{}", hex::encode(token_id.0))],
        }
    }
    
    /// Get token URI
    pub fn token_uri(&self, token_id: u256) -> FunctionCall {
        let selector = compute_selector("tokenURI");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![format!("0x{}", hex::encode(token_id.0))],
        }
    }
    
    /// Transfer
    pub fn transfer_from(&self, from: &StarknetAddress, to: &StarknetAddress, token_id: u256) -> FunctionCall {
        let selector = compute_selector("transferFrom");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![from.to_hex(), to.to_hex(), format!("0x{}", hex::encode(token_id.0))],
        }
    }
    
    /// Safe transfer
    pub fn safe_transfer_from(&self, from: &StarknetAddress, to: &StarknetAddress, token_id: u256) -> FunctionCall {
        let selector = compute_selector("safeTransferFrom");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![from.to_hex(), to.to_hex(), format!("0x{}", hex::encode(token_id.0))],
        }
    }
    
    /// Mint (if minter)
    pub fn mint(&self, to: &StarknetAddress, token_id: u256, uri: &str) -> FunctionCall {
        let selector = compute_selector("mint");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![to.to_hex(), format!("0x{}", hex::encode(token_id.0)), uri.to_string()],
        }
    }
}

/// ERC-1155 Token (Multi-token)
pub struct Erc1155 {
    address: StarknetAddress,
}

impl Erc1155 {
    /// Create at address
    pub fn at(address: StarknetAddress) -> Self {
        Self { address }
    }
    
    /// URI
    pub fn uri(&self, id: u256) -> FunctionCall {
        let selector = compute_selector("uri");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![format!("0x{}", hex::encode(id.0))],
        }
    }
    
    /// Balance of
    pub fn balance_of(&self, owner: &StarknetAddress, id: u256) -> FunctionCall {
        let selector = compute_selector("balanceOf");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![owner.to_hex(), format!("0x{}", hex::encode(id.0))],
        }
    }
    
    /// Batch balance of
    pub fn balance_of_batch(&self, owners: &[StarknetAddress], ids: &[u256]) -> FunctionCall {
        let selector = compute_selector("balanceOfBatch");
        let owners_hex: Vec<String> = owners.iter().map(|o| o.to_hex()).collect();
        let ids_hex: Vec<String> = ids.iter().map(|i| format!("0x{}", hex::encode(i.0))).collect();
        
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: [owners_hex, ids_hex].concat(),
        }
    }
    
    /// Safe transfer from
    pub fn safe_transfer_from(
        &self,
        from: &StarknetAddress,
        to: &StarknetAddress,
        id: u256,
        amount: u128,
        data: &[u8],
    ) -> FunctionCall {
        let selector = compute_selector("safeTransferFrom");
        FunctionCall {
            contract_address: self.address.to_hex(),
            entry_point_selector: hex::encode(selector),
            calldata: vec![
                from.to_hex(),
                to.to_hex(),
                format!("0x{}", hex::encode(id.0)),
                format!("0x{:x}", amount),
                hex::encode(data),
            ],
        }
    }
}

/// u256 type (Starknet's 256-bit integer)
#[derive(Debug, Clone, Copy, Default)]
pub struct u256(pub [u8; 32]);

impl u256 {
    pub fn from_u128(v: u128) -> Self {
        let mut bytes = [0u8; 32];
        bytes[16..].copy_from_slice(&v.to_be_bytes());
        Self(bytes)
    }
    
    pub fn as_u128(&self) -> u128 {
        u128::from_be_bytes(self.0[16..].try_into().unwrap())
    }
}

impl From<u128> for u256 {
    fn from(v: u128) -> Self {
        Self::from_u128(v)
    }
}
