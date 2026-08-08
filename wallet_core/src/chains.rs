/**
 * Chain-specific address generation and validation
 * 
 * Supports multiple blockchain address formats:
 * - Ethereum: 0x... (20 bytes, Keccak256)
 * - Bitcoin: Legacy, SegWit, Native SegWit
 * - Solana: Base58 encoded
 * - And many more...
 */

use crate::bip32::HDKey;
use crate::bip44::{Chain, coin_types};
use k256::ecdsa::{SigningKey, VerifyingKey};
use k256::elliptic_curve::point::DecompressPoint;
use sha2::{Sha256, Digest};
use sha3::Keccak256;
use ripemd::Ripemd160;
use thiserror::Error;
use std::fmt;

#[derive(Error, Debug)]
pub enum AddressError {
    #[error("Invalid address: {0}")]
    InvalidAddress(String),
    #[error("Invalid public key: {0}")]
    InvalidPublicKey(String),
    #[error("Unsupported chain: {0}")]
    UnsupportedChain(String),
}

pub type Result<T> = std::result::Result<T, AddressError>;

/// Address type
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum AddressType {
    /// Ethereum-style (20 bytes)
    Address20,
    /// Bitcoin legacy (P2PKH)
    P2PKH,
    /// Bitcoin SegWit (P2SH)
    P2SH,
    /// Bitcoin native SegWit (P2WPKH)
    P2WPKH,
    /// Solana (Base58)
    Solana,
    /// Aptos
    Aptos,
    /// Sui
    Sui,
    /// Generic
    Unknown,
}

/// Blockchain address
#[derive(Clone, Debug, Eq, PartialEq, Hash)]
pub struct Address(String);

impl Address {
    /// Create address from HD key and chain
    pub fn from_key(key: &HDKey, chain: Chain) -> Self {
        match chain {
            Chain::Ethereum | Chain::Polygon | Chain::BNBChain | 
                 Chain::Arbitrum | Chain::Optimism | Chain::Avalanche |
                 Chain::Starknet | Chain::ZkSync | Chain::Tron => {
                Self::from_evm_public_key(&key.public_key_bytes())
            },
            Chain::Bitcoin | Chain::Dogecoin | Chain::Litecoin => {
                Self::from_bitcoin_public_key(&key.public_key_bytes(), AddressType::P2WPKH)
            },
            Chain::Solana => {
                Self::from_solana_public_key(&key.public_key())
            },
            Chain::Aptos => {
                Self::from_aptos_public_key(&key.public_key())
            },
            Chain::Sui => {
                Self::from_sui_public_key(&key.public_key())
            },
            Chain::Ton => {
                Self::from_ton_public_key(&key.public_key())
            },
            _ => {
                Self::from_evm_public_key(&key.public_key_bytes())
            },
        }
    }
    
    /// Create Ethereum-style address from public key
    fn from_evm_public_key(pubkey: &[u8]) -> Self {
        // Get 64-byte uncompressed key (skip first byte for uncompressed)
        let mut full_key = vec![0x04];
        full_key.extend_from_slice(pubkey);
        
        // Keccak256 hash of public key
        let mut hasher = Keccak256::new();
        hasher.update(&full_key);
        let hash = hasher.finalize();
        
        // Take last 20 bytes
        let address_bytes = &hash[12..];
        
        // Format as 0x...
        let mut result = "0x".to_string();
        result.push_str(&hex::encode(address_bytes));
        
        Self(result)
    }
    
    /// Create Bitcoin address from public key
    fn from_bitcoin_public_key(pubkey: &[u8], addr_type: AddressType) -> Self {
        // SHA256
        let mut hasher = Sha256::new();
        hasher.update(pubkey);
        let sha_hash = hasher.finalize();
        
        // RIPEMD160
        let mut ripemd = Ripemd160::new();
        ripemd.update(&sha_hash);
        let ripemd_hash = ripemd.finalize();
        
        // Add version byte and create address
        match addr_type {
            AddressType::P2WPKH => {
                // Native SegWit (Bech32)
                let mut data = vec![0x00];  // Version 0
                data.extend_from_slice(&ripemd_hash);
                
                // Simplified - just return hex for now
                Self(format!("bc1q{}", hex::encode(&ripemd_hash[..20])))
            },
            _ => {
                // Legacy/P2SH
                let mut data = vec![0x00];  // Mainnet version
                data.extend_from_slice(&ripemd_hash);
                
                // Double SHA256 for checksum
                let mut cksum_hasher = Sha256::new();
                cksum_hasher.update(&data);
                let cksum1 = cksum_hasher.finalize();
                
                let mut cksum_hasher2 = Sha256::new();
                cksum_hasher2.update(&cksum1);
                let cksum2 = cksum_hasher2.finalize();
                
                data.extend_from_slice(&cksum2[..4]);
                
                Self(bs58::encode(&data).into_string())
            }
        }
    }
    
    /// Create Solana address from public key
    fn from_solana_public_key(pubkey: &[u8]) -> Self {
        // Solana uses Base58 encoding of 32-byte public key
        Self(bs58::encode(pubkey).into_string())
    }
    
    /// Create Aptos address from public key
    fn from_aptos_public_key(pubkey: &[u8]) -> Self {
        // Aptos uses SHA-3 256 of public key, then hex encoding
        let mut hasher = sha3::Sha3_256::new();
        hasher.update(pubkey);
        let hash = hasher.finalize();
        
        // Skip first 8 bytes and convert to hex
        Self(format!("0x{}", hex::encode(&hash[8..])))
    }
    
    /// Create Sui address from public key
    fn from_sui_public_key(pubkey: &[u8]) -> Self {
        // Sui uses SHA-256 of public key
        let mut hasher = Sha256::new();
        hasher.update(pubkey);
        let hash = hasher.finalize();
        
        // Take first 32 bytes and encode as hex with 0x prefix
        Self(format!("0x{}", hex::encode(&hash[..32])))
    }
    
    /// Create TON address from public key
    fn from_ton_public_key(pubkey: &[u8]) -> Self {
        // TON uses workchain + hash
        let mut hasher = Sha256::new();
        hasher.update(pubkey);
        let hash = hasher.finalize();
        
        // TON address format: workchain:hash
        Self(format!("0:{}", hex::encode(&hash[0..32])))
    }
    
    /// Validate address for chain
    pub fn validate(address: &str, chain: Chain) -> Result<()> {
        match chain {
            Chain::Ethereum | Chain::Polygon | Chain::BNBChain |
                 Chain::Arbitrum | Chain::Optimism | Chain::Avalanche => {
                Self::validate_evm(address)
            },
            Chain::Bitcoin => {
                Self::validate_bitcoin(address)
            },
            Chain::Solana => {
                Self::validate_solana(address)
            },
            _ => {
                // For other chains, just check it's not empty
                if address.is_empty() {
                    Err(AddressError::InvalidAddress("Empty address".to_string()))
                } else {
                    Ok(())
                }
            }
        }
    }
    
    /// Validate Ethereum address
    fn validate_evm(address: &str) -> Result<()> {
        if !address.starts_with("0x") {
            return Err(AddressError::InvalidAddress("Must start with 0x".to_string()));
        }
        
        if address.len() != 42 {
            return Err(AddressError::InvalidAddress(
                format!("Address must be 42 chars, got {}", address.len())
            ));
        }
        
        // Check hex validity
        let hex_part = &address[2..];
        if !hex_part.chars().all(|c| c.is_ascii_hexdigit()) {
            return Err(AddressError::InvalidAddress("Invalid hex characters".to_string()));
        }
        
        Ok(())
    }
    
    /// Validate Bitcoin address
    fn validate_bitcoin(address: &str) -> Result<()> {
        // Basic validation - check length and characters
        if address.len() < 26 || address.len() > 62 {
            return Err(AddressError::InvalidAddress("Invalid length".to_string()));
        }
        
        // Check Base58 or Bech32 validity
        let valid_chars = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
        if !address.chars().all(|c| valid_chars.contains(c) || c == 'q' || c == 'p') {
            return Err(AddressError::InvalidAddress("Invalid characters".to_string()));
        }
        
        Ok(())
    }
    
    /// Validate Solana address
    fn validate_solana(address: &str) -> Result<()> {
        // Base58, 32-44 characters
        if address.len() < 32 || address.len() > 44 {
            return Err(AddressError::InvalidAddress(
                format!("Invalid length: {}", address.len())
            ));
        }
        
        let valid_chars = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
        if !address.chars().all(|c| valid_chars.contains(c)) {
            return Err(AddressError::InvalidAddress("Invalid characters".to_string()));
        }
        
        Ok(())
    }
    
    /// Get address string
    pub fn as_str(&self) -> &str {
        &self.0
    }
    
    /// Get as bytes (without 0x prefix for EVM)
    pub fn as_bytes(&self) -> Vec<u8> {
        if self.0.starts_with("0x") {
            hex::decode(&self.0[2..]).unwrap_or_default()
        } else {
            self.0.as_bytes().to_vec()
        }
    }
}

impl fmt::Display for Address {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl std::str::FromStr for Address {
    type Err = AddressError;
    
    fn from_str(s: &str) -> Result<Self> {
        Ok(Self(s.to_string()))
    }
}

impl AsRef<str> for Address {
    fn as_ref(&self) -> &str {
        &self.0
    }
}

/// Chain type
pub type ChainType = crate::bip44::Chain;

/// Get chain from coin type
pub fn chain_from_coin_type(coin_type: u32) -> Option<Chain> {
    match coin_type {
        coin_types::BITCOIN => Some(Chain::Bitcoin),
        coin_types::ETHEREUM => Some(Chain::Ethereum),
        coin_types::POLYGON => Some(Chain::Polygon),
        coin_types::BNB_CHAIN => Some(Chain::BNBChain),
        coin_types::ARBITRUM => Some(Chain::Arbitrum),
        coin_types::OPTIMISM => Some(Chain::Optimism),
        coin_types::AVALANCHE_C => Some(Chain::Avalanche),
        coin_types::SOLANA => Some(Chain::Solana),
        coin_types::APTOS => Some(Chain::Aptos),
        coin_types::SUI => Some(Chain::Sui),
        coin_types::TON => Some(Chain::Ton),
        coin_types::NEAR => Some(Chain::Near),
        coin_types::COSMOS => Some(Chain::Cosmos),
        coin_types::ALGORAND => Some(Chain::Algorand),
        coin_types::HEDERA => Some(Chain::Hedera),
        coin_types::STARKNET => Some(Chain::Starknet),
        coin_types::ZKSYNC => Some(Chain::ZkSync),
        coin_types::TRON => Some(Chain::Tron),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::bip32::HDKey;
    
    #[test]
    fn test_eth_address() {
        let seed = [0u8; 64];
        let key = HDKey::from_seed(&seed).unwrap();
        
        let addr = Address::from_key(&key, Chain::Ethereum);
        
        assert!(addr.to_string().starts_with("0x"));
        assert_eq!(addr.to_string().len(), 42);
    }
    
    #[test]
    fn test_sol_address() {
        let seed = [0u8; 64];
        let key = HDKey::from_seed(&seed).unwrap();
        
        let addr = Address::from_key(&key, Chain::Solana);
        
        assert!(!addr.to_string().starts_with("0x"));
        assert!(addr.to_string().len() >= 32);
    }
    
    #[test]
    fn test_validate_eth() {
        assert!(Address::validate("0x742d35Cc6634C0532925a3b844Bc9e7595f1234", Chain::Ethereum).is_ok());
        assert!(Address::validate("0x742d35Cc6634C0532925a3b844Bc9e7595f123", Chain::Ethereum).is_err());
        assert!(Address::validate("invalid", Chain::Ethereum).is_err());
    }
}
