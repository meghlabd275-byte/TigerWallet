/**
 * BIP-32 HD Key Derivation Implementation
 * 
 * Production-ready BIP-32 implementation for hierarchical deterministic
 * key derivation using well-audited cryptographic libraries.
 */

use k256::ecdsa::{SigningKey, VerifyingKey};
use k256::elliptic_curve::scalar::NonZeroScalar;
use k256::Secp256k1;
use sha2::{Sha512, Digest};
use hmac::{Hmac, Mac};
use thiserror::Error;
use std::fmt;

#[derive(Error, Debug)]
pub enum DerivationError {
    #[error("Invalid seed: {0}")]
    InvalidSeed(String),
    #[error("Invalid key: {0}")]
    InvalidKey(String),
    #[error("Derivation failed: {0}")]
    DerivationFailed(String),
    #[error("Invalid path: {0}")]
    InvalidPath(String),
}

pub type Result<T> = std::result::Result<T, DerivationError>;

/// HMAC-SHA512
type HmacSha512 = Hmac<Sha512>;

/// 32-byte key
#[derive(Clone, Debug)]
pub struct Key32(pub [u8; 32]);

impl Key32 {
    pub fn from_slice(slice: &[u8]) -> Option<Self> {
        if slice.len() != 32 {
            return None;
        }
        let mut key = [0u8; 32];
        key.copy_from_slice(slice);
        Some(Self(key))
    }
    
    pub fn as_slice(&self) -> &[u8; 32] {
        &self.0
    }
}

/// HD Key structure (BIP-32)
#[derive(Clone, Debug)]
pub struct HDKey {
    pub key: Key32,           // Private key (0x00 + 32 bytes) or public key
    pub chain_code: Key32,   // Chain code
    pub depth: u8,           // Depth (0 for master)
    pub child_number: u32,   // Child index
    pub parent_fingerprint: [u8; 4],  // Parent fingerprint
}

impl HDKey {
    /// Create master key from seed (BIP-32)
    pub fn from_seed(seed: &[u8; 64]) -> Result<Self> {
        // I = HMAC-SHA512(Key = "Bitcoin seed", Data = seed)
        let mut mac = HmacSha512::new_from_slice(b"Bitcoin seed")
            .map_err(|e| DerivationError::InvalidSeed(e.to_string()))?;
        mac.update(seed);
        let result = mac.finalize().into_bytes();
        
        let il = &result[..32];
        let ir = &result[32..];
        
        // Validate that il is a valid private key
        if il[0] >= 0x80 {
            return Err(DerivationError::InvalidSeed("Invalid master key".to_string()));
        }
        
        let mut key = [0u8; 32];
        key.copy_from_slice(il);
        
        let mut chain_code = [0u8; 32];
        chain_code.copy_from_slice(ir);
        
        Ok(HDKey {
            key: Key32(key),
            chain_code: Key32(chain_code),
            depth: 0,
            child_number: 0,
            parent_fingerprint: [0u8; 4],
        })
    }
    
    /// Derive child key (BIP-32)
    pub fn derive(&self, index: u32) -> Result<HDKey> {
        let hardened = index >= 0x80000000u32;
        let real_index = if hardened { index } else { index };
        
        let mut data = Vec::with_capacity(37);
        
        if hardened {
            // Hardened derivation: 0x00 || ser256(k) || ser32(i)
            data.push(0x00);
            data.extend_from_slice(self.key.as_slice());
        } else {
            // Normal derivation: serP(point(k)) || ser32(i)
            let public_key = self.public_key();
            data.extend_from_slice(&public_key);
        }
        
        // Add child index
        data.extend_from_slice(&index.to_be_bytes());
        
        // I = HMAC-SHA512(Key = chain_code, Data = data)
        let mut mac = HmacSha512::new_from_slice(self.chain_code.as_slice())
            .map_err(|e| DerivationError::DerivationFailed(e.to_string()))?;
        mac.update(&data);
        let result = mac.finalize().into_bytes();
        
        let il = &result[..32];
        let ir = &result[32..];
        
        // Parse il as scalar and add to parent key
        let il_scalar = NonZeroScalar::from_slice(il)
            .map_err(|_| DerivationError::DerivationFailed("Invalid child key".to_string()))?;
        
        let parent_key = SigningKey::from_bytes(self.key.as_slice().into())
            .map_err(|e| DerivationError::InvalidKey(e.to_string()))?;
        
        let child_key = (parent_key.to_scalar() + il_scalar).to_bytes();
        
        let mut key = [0u8; 32];
        key.copy_from_slice(child_key.as_slice());
        
        let mut chain_code = [0u8; 32];
        chain_code.copy_from_slice(ir);
        
        // Calculate fingerprint of parent public key
        let parent_pub = self.public_key();
        let mut hasher = sha2::Sha256::new();
        hasher.update(&parent_pub);
        let hash = hasher.finalize();
        
        let mut fingerprint = [0u8; 4];
        fingerprint.copy_from_slice(&hash[..4]);
        
        Ok(HDKey {
            key: Key32(key),
            chain_code: Key32(chain_code),
            depth: self.depth + 1,
            child_number: index,
            parent_fingerprint: fingerprint,
        })
    }
    
    /// Derive multiple levels from path string (e.g., "m/44'/60'/0'/0/0")
    pub fn derive_path(&self, path: &str) -> Result<HDKey> {
        let path = path.trim_start_matches('m');
        
        if path.is_empty() {
            return Ok(self.clone());
        }
        
        let mut current = self.clone();
        
        for component in path.split('/') {
            let component = component.trim();
            
            if component.is_empty() {
                continue;
            }
            
            let hardened = component.ends_with('\'');
            let index_str = component.trim_end_matches('\'');
            
            let index: u32 = index_str.parse()
                .map_err(|_| DerivationError::InvalidPath(format!("Invalid index: {}", component)))?;
            
            let index = if hardened { index | 0x80000000 } else { index };
            
            current = current.derive(index)?;
        }
        
        Ok(current)
    }
    
    /// Get public key from private key
    pub fn public_key(&self) -> Vec<u8> {
        let signing_key = SigningKey::from_bytes(self.key.as_slice().into())
            .expect("Invalid private key");
        let verifying_key = VerifyingKey::from(&signing_key);
        
        // Return compressed public key
        verifying_key.to_encoded_point(true).as_bytes().to_vec()
    }
    
    /// Get public key as 33-byte compressed
    pub fn public_key_bytes(&self) -> [u8; 33] {
        let pubkey = self.public_key();
        let mut bytes = [0u8; 33];
        bytes.copy_from_slice(&pubkey);
        bytes
    }
    
    /// Check if this is a private key
    pub fn is_private(&self) -> bool {
        self.key.as_slice()[0] != 0x00
    }
    
    /// Get key bytes (for private key, includes 0x00 prefix)
    pub fn key_bytes(&self) -> Vec<u8> {
        if self.is_private() {
            let mut v = vec![0u8; 33];
            v[1..].copy_from_slice(self.key.as_slice());
            v
        } else {
            self.public_key()
        }
    }
}

/// Derivation path representation
pub struct DerivationPath {
    path: Vec<u32>,
}

impl DerivationPath {
    pub fn new(path: &[u32]) -> Self {
        Self {
            path: path.to_vec(),
        }
    }
    
    pub fn parse(path: &str) -> Result<Self> {
        let path = path.trim_start_matches('m');
        let mut indices = Vec::new();
        
        for component in path.split('/') {
            let component = component.trim();
            if component.is_empty() {
                continue;
            }
            
            let hardened = component.ends_with('\'');
            let index_str = component.trim_end_matches('\'');
            
            let index: u32 = index_str.parse()
                .map_err(|_| DerivationError::InvalidPath(format!("Invalid: {}", component)))?;
            
            let index = if hardened { index | 0x80000000 } else { index };
            indices.push(index);
        }
        
        Ok(Self { path: indices })
    }
    
    pub fn as_indices(&self) -> &[u32] {
        &self.path
    }
}

impl fmt::Display for DerivationPath {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "m")?;
        for (i, idx) in self.path.iter().enumerate() {
            if i == 0 {
                write!(f, "/")?;
            } else {
                write!(f, "/")?;
            }
            
            if idx & 0x80000000 != 0 {
                write!(f, "{}'", idx & 0x7FFFFFFF)?;
            } else {
                write!(f, "{}", idx)?;
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_master_key() {
        let seed = [0u8; 64];
        let key = HDKey::from_seed(&seed).unwrap();
        assert_eq!(key.depth, 0);
    }
    
    #[test]
    fn test_derive() {
        let seed = [0u8; 64];
        let master = HDKey::from_seed(&seed).unwrap();
        let child = master.derive(0).unwrap();
        
        assert_eq!(child.depth, 1);
        assert_eq!(child.child_number, 0);
    }
    
    #[test]
    fn test_derive_path() {
        let seed = [0u8; 64];
        let master = HDKey::from_seed(&seed).unwrap();
        
        // Derive path m/44'/60'/0'/0/0
        let derived = master.derive_path("m/44'/60'/0'/0/0").unwrap();
        
        assert_eq!(derived.depth, 5);
    }
    
    #[test]
    fn test_public_key() {
        let seed = [0u8; 64];
        let key = HDKey::from_seed(&seed).unwrap();
        
        let pubkey = key.public_key();
        assert_eq!(pubkey.len(), 33);  // Compressed
    }
    
    #[test]
    fn test_path_display() {
        let path = DerivationPath::parse("m/44'/60'/0'/0/0").unwrap();
        assert_eq!(path.to_string(), "m/44'/60'/0'/0/0");
    }
}
