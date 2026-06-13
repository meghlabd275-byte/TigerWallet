//! TigerWallet Address Generation - Rust Implementation
//! Multi-chain address derivation (BTC, ETH, SOL, etc.)

use serde::{Deserialize, Serialize};

// ============================================================================
// Address Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum ChainType {
    Bitcoin,
    Ethereum,
    Solana,
    Tron,
    Cosmos,
    Aptos,
    Sui,
    Ton,
    Near,
    Algorand,
    Polkadot,
}

// ============================================================================
// Chain Configuration
// ============================================================================

#[derive(Debug, Clone)]
pub struct ChainConfig {
    pub chain_type: ChainType,
    pub coin_type: u32,
    pub symbol: String,
    pub name: String,
    pub decimals: u8,
}

impl ChainConfig {
    pub fn ethereum() -> Self {
        ChainConfig {
            chain_type: ChainType::Ethereum,
            coin_type: 60, // BIP-44
            symbol: "ETH".to_string(),
            name: "Ethereum".to_string(),
            decimals: 18,
        }
    }

    pub fn bitcoin() -> Self {
        ChainConfig {
            chain_type: ChainType::Bitcoin,
            coin_type: 0,
            symbol: "BTC".to_string(),
            name: "Bitcoin".to_string(),
            decimals: 8,
        }
    }

    pub fn solana() -> Self {
        ChainConfig {
            chain_type: ChainType::Solana,
            coin_type: 501,
            symbol: "SOL".to_string(),
            name: "Solana".to_string(),
            decimals: 9,
        }
    }
}

// ============================================================================
// Address Generator
// ============================================================================

pub struct AddressGenerator {
    seed: [u8; 64],
}

impl AddressGenerator {
    /// Create from seed
    pub fn from_seed(seed: &[u8; 64]) -> Self {
        AddressGenerator { seed: *seed }
    }

    /// Derive Ethereum address
    pub fn derive_ethereum(&self, path: &str) -> Result<String, AddressError> {
        let private_key = self.derive_private_key(path)?;
        let public_key = derive_public_key(&private_key);
        let address = public_key_to_ethereum_address(&public_key);
        Ok(format!("0x{}", address))
    }

    /// Derive Bitcoin address (P2WPKH)
    pub fn derive_bitcoin(&self, path: &str) -> Result<String, AddressError> {
        let private_key = self.derive_private_key(path)?;
        let public_key = derive_public_key(&private_key);
        let address = public_key_to_bitcoin_address(&public_key);
        Ok(address)
    }

    /// Derive Solana address
    pub fn derive_solana(&self, path: &str) -> Result<String, AddressError> {
        let private_key = self.derive_private_key(path)?;
        let public_key = derive_ed25519_public_key(&private_key);
        let address = base58_encode(&public_key);
        Ok(address)
    }

    /// Derive Tron address
    pub fn derive_tron(&self, path: &str) -> Result<String, AddressError> {
        let eth_address = self.derive_ethereum(path)?;
        let tron_address = eth_to_tron_address(&eth_address);
        Ok(tron_address)
    }

    /// Derive Cosmos address
    pub fn derive_cosmos(&self, path: &str) -> Result<String, AddressError> {
        let private_key = self.derive_private_key(path)?;
        let public_key = derive_public_key(&private_key);
        let address = public_key_to_cosmos_address(&public_key);
        Ok(address)
    }

    /// Derive private key for path
    fn derive_private_key(&self, path: &str) -> Result<[u8; 32], AddressError> {
        // Simplified BIP-32 derivation
        let path_components: Vec<&str> = path.trim_start_matches("m/").split('/').collect();
        
        let mut key = self.seed[..32].to_vec();
        let mut chain_code = self.seed[32..].to_vec();
        
        for component in path_components {
            let hardened = component.ends_with('\'');
            let idx = component.trim_end_matches('\'').parse::<u32>()
                .map_err(|_| AddressError::InvalidPath)?;
            
            let idx = if hardened { idx | 0x80000000 } else { idx };
            
            // HMAC-SHA512
            let mut data = vec![0u8]; // Hardened marker
            data.extend_from_slice(&key);
            data.extend_from_slice(&idx.to_be_bytes());
            
            let hmac_result = hmac_sha512(&chain_code, &data);
            key = hmac_result[..32].to_vec();
            chain_code = hmac_result[32..].to_vec();
        }
        
        let mut result = [0u8; 32];
        result.copy_from_slice(&key);
        Ok(result)
    }
}

// ============================================================================
// Helper Functions
// ============================================================================

fn derive_public_key(private_key: &[u8; 32]) -> [u8; 64] {
    use secp256k1::{PublicKey, SecretKey};
    
    let secret = SecretKey::from_slice(private_key)
        .expect("32 bytes");
    let public = PublicKey::from_secret_key(&secret);
    
    let serialized = public.serialize_uncompressed();
    let mut result = [0u8; 64];
    result.copy_from_slice(&serialized[1..65]);
    result
}

fn derive_ed25519_public_key(private_key: &[u8; 32]) -> [u8; 32] {
    use ed25519_dalek::{PublicKey, SecretKey};
    
    let secret = SecretKey::from_bytes(private_key)
        .expect("32 bytes");
    let public: PublicKey = (&secret).into();
    
    public.to_bytes()
}

fn public_key_to_ethereum_address(public_key: &[u8; 64]) -> String {
    let hash = keccak256(public_key);
    let address_bytes = &hash[12..];
    hex::encode(address_bytes)
}

fn public_key_to_bitcoin_address(public_key: &[u8; 64]) -> String {
    // P2WPKH (Bech32)
    use base58::ToBase58;
    
    let hash = ripemd160_hash(&sha256_hash(public_key));
    
    // Witness program version 0
    let mut program = vec![0x00];
    program.extend_from_slice(&hash);
    
    // Convert to bech32
    bech32_encode("bc", &program)
}

fn public_key_to_cosmos_address(public_key: &[u8; 64]) -> String {
    let hash = ripemd160_hash(public_key);
    
    // Cosmos bech32 encoding
    let mut data = vec![0x1C]; // cosmos prefix
    data.extend_from_slice(&hash[..20]);
    
    bech32_encode("cosmos", &data)
}

fn eth_to_tron_address(eth_address: &str) -> String {
    let addr_bytes = hex::decode(&eth_address[2..])
        .expect("valid hex");
    
    // Add 0x41 prefix
    let mut tron_addr = vec![0x41];
    tron_addr.extend_from_slice(&addr_bytes);
    
    // Add checksum
    let hash = sha256_hash(&tron_addr);
    tron_addr.extend_from_slice(&hash[..4]);
    
    base58::ToBase58::to_base58(&tron_addr)
}

fn hmac_sha512(key: &[u8], data: &[u8]) -> [u8; 64] {
    use hmac::{Hmac, Mac};
    use sha2::Sha512;
    
    type HmacSha512 = Hmac<Sha512>;
    
    let mut mac = HmacSha512::new_from_slice(key).expect("HMAC can take key of any size");
    mac.update(data);
    
    let result = mac.finalize();
    let bytes: [u8; 64] = result.into_bytes().into();
    bytes
}

fn sha256_hash(data: &[u8]) -> Vec<u8> {
    use sha2::{Sha256, Digest};
    let mut hasher = Sha256::new();
    hasher.update(data);
    hasher.finalize().to_vec()
}

fn keccak256(data: &[u8]) -> Vec<u8> {
    use keccak::{Keccak, digest::Digest};
    let mut hasher = Keccak::new_keccak_256();
    hasher.update(data);
    hasher.finalize().to_vec()
}

fn ripemd160_hash(data: &[u8]) -> Vec<u8> {
    use ripemd160::Ripemd160;
    let mut hasher = Ripemd160::new();
    hasher.update(data);
    hasher.finalize().to_vec()
}

fn base58_encode(data: &[u8]) -> String {
    use base58::ToBase58;
    data.to_base58()
}

fn bech32_encode(hrp: &str, data: &[u8]) -> String {
    const CHARSET: &[u8] = b"qpzry9x8gf2tvdw0s3jn54khce6mua7l";
    
    let mut result = hrp.to_string();
    result.push('1');
    
    for byte in data {
        result.push(CHARSET[(*byte & 0x1F) as usize] as char);
    }
    
    result
}

// ============================================================================
// Errors
// ============================================================================

#[derive(Debug, thiserror::Error)]
pub enum AddressError {
    #[error("Invalid derivation path: {0}")]
    InvalidPath(String),
    
    #[error("Invalid public key: {0}")]
    InvalidPublicKey(String),
    
    #[error("Encoding error: {0}")]
    EncodingError(String),
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_ethereum_address() {
        let seed = [0u8; 64];
        let generator = AddressGenerator::from_seed(&seed);
        
        let address = generator.derive_ethereum("m/44'/60'/0'/0/0").unwrap();
        
        assert!(address.starts_with("0x"));
        assert_eq!(address.len(), 42);
    }

    #[test]
    fn test_bitcoin_address() {
        let seed = [0u8; 64];
        let generator = AddressGenerator::from_seed(&seed);
        
        let address = generator.derive_bitcoin("m/84'/0'/0'/0/0").unwrap();
        
        assert!(address.starts_with("bc1"));
    }

    #[test]
    fn test_solana_address() {
        let seed = [0u8; 64];
        let generator = AddressGenerator::from_seed(&seed);
        
        let address = generator.derive_solana("m/44'/501'/0'/0'").unwrap();
        
        assert!(address.len() > 30);
    }
}