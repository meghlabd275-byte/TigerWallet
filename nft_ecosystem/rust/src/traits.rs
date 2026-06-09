//! NFT Traits Module - Rust Implementation
//! ERC-721 and ERC-1155 trait implementations

use serde::{Deserialize, Serialize};

/// ERC-721 NFT Trait
pub trait NFTTrait {
    fn token_id(&self) -> &str;
    fn owner(&self) -> &str;
    fn uri(&self) -> &str;
    
    fn approve(&mut self, operator: &str, approved: bool);
    fn is_approved(&self, operator: &str) -> bool;
    
    fn transfer(&mut self, from: &str, to: &str);
    fn safe_transfer(&mut self, from: &str, to: &str, data: Option<&[u8]>);
    
    fn balance_of(&self, owner: &str) -> u64;
    fn owner_of(&self) -> &str;
    fn get_approved(&self) -> Option<&str>;
}

/// ERC-721 Enumerable Extension
pub trait NFTEnumerable {
    fn total_supply(&self) -> u64;
    fn token_by_index(&self, index: u64) -> Option<String>;
    fn token_of_owner_by_index(&self, owner: &str, index: u64) -> Option<String>;
}

/// ERC-721 Metadata Extension
pub trait NFTMetadataURI {
    fn name(&self) -> &str;
    fn symbol(&self) -> &str;
    fn token_uri(&self) -> String;
}

/// ERC-1155 Multi-token Standard
pub trait MultiToken {
    fn balance_of(&self, account: &str, id: &str) -> u64;
    fn balance_of_batch(&self, accounts: &[String], ids: &[String]) -> Vec<u64>;
    
    fn setApprovalForAll(&mut self, operator: &str, approved: bool);
    fn is_approved_for_all(&self, account: &str, operator: &str) -> bool;
    
    fn safe_transfer_from(&mut self, from: &str, to: &str, id: &str, amount: u64, data: Option<&[u8]>);
    fn safe_batch_transfer_from(&mut self, from: &str, to: &str, ids: &[String], amounts: &[u64], data: Option<&[u8]>);
    
    fn uri(&self, id: &str) -> String;
}

/// Royalties interface (EIP-2981)
pub trait NFTRoyalties {
    fn royalty_info(&self, token_id: &str, sale_price: u64) -> (String, u64);
}

/// NFT Storage trait for caching
pub trait NFTStorage {
    fn cache_metadata(&self, key: &str, metadata: &NFTMetadata);
    fn get_cached_metadata(&self, key: &str) -> Option<NFTMetadata>;
    fn invalidate_cache(&self, key: &str);
}

/// NFT metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NFTMetadata {
    pub token_id: String,
    pub name: String,
    pub description: String,
    pub image: String,
    pub animation_url: Option<String>,
    pub external_url: Option<String>,
    pub attributes: Vec<Attribute>,
}

/// NFT attribute
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Attribute {
    pub trait_type: String,
    pub value: String,
    pub display_type: Option<String>,
}

/// Chain interaction trait
pub trait NFTChainClient {
    fn get_owner(&self, contract: &str, token_id: &str) -> Option<String>;
    fn get_balance(&self, contract: &str, owner: &str) -> u64;
    fn transfer(&self, contract: &str, from: &str, to: &str, token_id: &str) -> Result<String, String>;
    fn mint(&self, contract: &str, to: &str, uri: &str) -> Result<(String, String), String>;
    fn burn(&self, contract: &str, owner: &str, token_id: &str) -> Result<String, String>;
}

/// Batch operations trait
pub trait NFTBatch {
    fn batch_transfer(&self, transfers: &[Transfer]) -> Result<Vec<String>, String>;
    fn batch_mint(&self, mints: &[Mint]) -> Result<Vec<String>, String>;
}

/// Transfer structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transfer {
    pub contract: String,
    pub from: String,
    pub to: String,
    pub token_id: String,
}

/// Mint structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Mint {
    pub contract: String,
    pub to: String,
    pub uri: String,
}

/// NFT Contract type
#[derive(Debug, Clone, Copy, PartialEq, Serialize, Deserialize)]
pub enum NFTContractType {
    ERC721,
    ERC721A,
    ERC721URIStorage,
    ERC1155,
    ERC1155URIStorage,
}

/// Implementation for ERC-721
pub struct ERC721 {
    token_id: String,
    owner: String,
    uri: String,
    approved: Option<String>,
    balances: std::collections::HashMap<String, u64>,
}

impl ERC721 {
    pub fn new(token_id: String, owner: String, uri: String) -> Self {
        let mut balances = std::collections::HashMap::new();
        balances.insert(owner.clone(), 1);
        
        Self {
            token_id,
            owner,
            uri,
            approved: None,
            balances,
        }
    }
}

impl NFTTrait for ERC721 {
    fn token_id(&self) -> &str {
        &self.token_id
    }
    
    fn owner(&self) -> &str {
        &self.owner
    }
    
    fn uri(&self) -> &str {
        &self.uri
    }
    
    fn approve(&mut self, operator: &str, approved: bool) {
        if approved {
            self.approved = Some(operator.to_string());
        } else {
            self.approved = None;
        }
    }
    
    fn is_approved(&self, operator: &str) -> bool {
        self.approved.as_deref().map(|o| o == operator).unwrap_or(false)
    }
    
    fn transfer(&mut self, from: &str, to: &str) {
        if self.owner == from {
            self.owner = to.to_string();
        }
    }
    
    fn safe_transfer(&mut self, from: &str, to: &str, _data: Option<&[u8]>) {
        self.transfer(from, to);
    }
    
    fn balance_of(&self, owner: &str) -> u64 {
        *self.balances.get(owner).unwrap_or(&0)
    }
    
    fn owner_of(&self) -> &str {
        &self.owner
    }
    
    fn get_approved(&self) -> Option<&str> {
        self.approved.as_deref()
    }
}

impl NFTEnumerable for ERC721 {
    fn total_supply(&self) -> u64 {
        self.balances.values().sum()
    }
    
    fn token_by_index(&self, index: u64) -> Option<String> {
        if index == 0 {
            Some(self.token_id.clone())
        } else {
            None
        }
    }
    
    fn token_of_owner_by_index(&self, owner: &str, index: u64) -> Option<String> {
        if self.balances.get(owner).unwrap_or(&0) > &index && index == 0 {
            Some(self.token_id.clone())
        } else {
            None
        }
    }
}

impl NFTMetadataURI for ERC721 {
    fn name(&self) -> &str {
        "TigerWallet NFT"
    }
    
    fn symbol(&self) -> &str {
        "TW-NFT"
    }
    
    fn token_uri(&self) -> String {
        self.uri.clone()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_erc721() {
        let nft = ERC721::new("1".to_string(), "0xowner".to_string(), "ipfs://metadata".to_string());
        
        assert_eq!(nft.token_id(), "1");
        assert_eq!(nft.owner(), "0xowner");
        assert_eq!(nft.balance_of("0xowner"), 1);
    }
}