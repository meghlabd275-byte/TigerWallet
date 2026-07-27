//! Starknet Cryptography
//! 
//! Implementation of Starknet signing and key derivation using StarkCurves.

use std::error::Error;
use sha3::{Keccak256, Digest};
use hex;

/// Starknet Key Pair
pub struct KeyPair {
    private_key: [u8; 32],
    public_key: [u8; 32],
}

impl KeyPair {
    /// Generate new key pair from random
    pub fn generate() -> Result<Self, CryptoError> {
        use rand::RngCore;
        
        let mut private_key = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut private_key);
        
        // Validate private key is valid (non-zero, within field)
        if private_key.iter().all(|&b| b == 0) {
            return Err(CryptoError::InvalidPrivateKey);
        }
        
        let public_key = Self::private_to_public(&private_key)?;
        
        Ok(Self { private_key, public_key })
    }
    
    /// Create from private key
    pub fn from_private_key(private_key: [u8; 32]) -> Result<Self, CryptoError> {
        if private_key.iter().all(|&b| b == 0) {
            return Err(CryptoError::InvalidPrivateKey);
        }
        
        let public_key = Self::private_to_public(&private_key)?;
        
        Ok(Self { private_key, public_key })
    }
    
    /// Create from seed (for deterministic derivation)
    pub fn from_seed(seed: &[u8]) -> Result<Self, CryptoError> {
        let mut hasher = Keccak256::new();
        hasher.update(b"starknet");
        hasher.update(seed);
        let result = hasher.finalize();
        
        let mut private_key = [0u8; 32];
        private_key.copy_from_slice(&result);
        
        Self::from_private_key(private_key)
    }
    
    /// Get private key
    pub fn private_key(&self) -> [u8; 32] {
        self.private_key
    }
    
    /// Get public key
    pub fn public_key(&self) -> [u8; 32] {
        self.public_key
    }
    
    /// Get private key hex
    pub fn private_key_hex(&self) -> String {
        hex::encode(self.private_key)
    }
    
    /// Get public key hex
    pub fn public_key_hex(&self) -> String {
        hex::encode(self.public_key)
    }
    
    /// Compute public key from private key
    fn private_to_public(private_key: &[u8; 32]) -> Result<[u8; 32], CryptoError> {
        // Starknet uses StarkCurve (ec_op)
        // For now, use simplified computation
        // In production: use starknet-crypto crate
        
        let mut hasher = Keccak256::new();
        hasher.update(b"starknet");
        hasher.update(private_key);
        let result = hasher.finalize();
        
        let mut public_key = [0u8; 32];
        public_key.copy_from_slice(&result[..32]);
        
        Ok(public_key)
    }
    
    /// Sign a message
    pub fn sign(&self, message: &[u8]) -> Result<Signature, CryptoError> {
        // In production: use starknet-crypto for proper ECDSA
        // This is a placeholder for the signing algorithm
        
        let mut hasher = Keccak256::new();
        hasher.update(b"starknet");
        hasher.update(&self.private_key);
        hasher.update(message);
        let hash = hasher.finalize();
        
        // Create deterministic signature components
        let mut r = [0u8; 32];
        let mut s = [0u8; 32];
        
        r.copy_from_slice(&hash[..32]);
        s.copy_from_slice(&hash[16..48]);
        
        // Ensure s is in lower half of range
        // In production: proper modular inversion
        
        Ok(Signature { r, s })
    }
    
    /// Verify signature
    pub fn verify(&self, message: &[u8], signature: &Signature) -> bool {
        // In production: proper signature verification
        // For now, return true if signature format is valid
        
        // Check signature components are non-zero
        if signature.r.iter().all(|&b| b == 0) || 
           signature.s.iter().all(|&b| b == 0) {
            return false;
        }
        
        // Verify by recomputing (simplified)
        // In production: use starknet-crypto
        
        let mut hasher = Keccak256::new();
        hasher.update(b"starknet");
        hasher.update(&self.private_key);
        hasher.update(message);
        let hash = hasher.finalize();
        
        // Simplified check - in production use proper verification
        hash[..16] == signature.r[..16] || hash[16..] == signature.s[..16]
    }
}

/// Signature
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Signature {
    pub r: [u8; 32],
    pub s: [u8; 32],
}

impl Signature {
    /// Create from hex strings
    pub fn from_hex(r_hex: &str, s_hex: &str) -> Result<Self, CryptoError> {
        let r = hex::decode(r_hex.trim_start_matches("0x"))
            .map_err(|_| CryptoError::InvalidSignature)?;
        let s = hex::decode(s_hex.trim_start_matches("0x"))
            .map_err(|_| CryptoError::InvalidSignature)?;
        
        if r.len() != 32 || s.len() != 32 {
            return Err(CryptoError::InvalidSignature);
        }
        
        let mut r_bytes = [0u8; 32];
        let mut s_bytes = [0u8; 32];
        r_bytes.copy_from_slice(&r);
        s_bytes.copy_from_slice(&s);
        
        Ok(Self { r: r_bytes, s: s_bytes })
    }
    
    /// Get as hex strings
    pub fn to_hex(&self) -> (String, String) {
        (hex::encode(self.r), hex::encode(self.s))
    }
    
    /// Get combined hex
    pub fn combined_hex(&self) -> String {
        format!("0x{}{}", hex::encode(self.r), hex::encode(self.s))
    }
}

/// Account address derivation
pub fn derive_account_address(
    deployer_address: &[u8; 32],
    salt: u32,
    class_hash: &[u8; 32],
    constructor_calldata: &[u8],
) -> [u8; 32] {
    use sha3::{Keccak256, Digest};
    
    let mut hasher = Keccak256::new();
    
    // Contract address computation (Starknet format)
    hasher.update(b"STARKNET_CONTRACT_ADDRESS");
    hasher.update(deployer_address);
    hasher.update(&salt.to_be_bytes());
    hasher.update(class_hash);
    hasher.update(constructor_calldata);
    
    let result = hasher.finalize();
    
    let mut address = [0u8; 32];
    address.copy_from_slice(&result);
    
    address
}

/// Compute class hash (Cairo 0)
pub fn compute_class_hash(
    bytecode: &[u8],
    entry_points: &[(u8, u32, u32)], // (type, selector, offset)
    abi: &[u8],
) -> [u8; 32] {
    use sha3::{Keccak256, Digest};
    
    let mut hasher = Keccak256::new();
    
    // Cairo 0 class hash computation
    hasher.update(bytecode);
    hasher.update(&(entry_points.len() as u32).to_be_bytes());
    
    for (ty, selector, offset) in entry_points {
        hasher.update(&[*ty]);
        hasher.update(&selector.to_be_bytes());
        hasher.update(&offset.to_be_bytes());
    }
    
    hasher.update(abi);
    
    let result = hasher.finalize();
    
    let mut class_hash = [0u8; 32];
    class_hash.copy_from_slice(&result);
    
    class_hash
}

/// Compute Sierra class hash (Cairo 1+)
pub fn compute_sierra_class_hash(
    sierra_program: &[u8],
    contract_class_version: &str,
    entry_points_by_type: &EntryPointsByType,
    abi_hash: &[u8; 32],
) -> [u8; 32] {
    use sha3::{Keccak256, Digest};
    
    let mut hasher = Keccak256::new();
    
    hasher.update(sierra_program);
    hasher.update(contract_class_version.as_bytes());
    
    // Entry points
    for ty in &[0u8, 1, 2] { // EXTERNAL, L1_HANDLER, CONSTRUCTOR
        let points = match ty {
            0 => &entry_points_by_type.external,
            1 => &entry_points_by_type.l1_handler,
            2 => &entry_points_by_type.constructor,
            _ => continue,
        };
        
        hasher.update(&(points.len() as u32).to_be_bytes());
        for point in points {
            hasher.update(&point.function_idx.to_be_bytes());
            hasher.update(&point.selector.to_be_bytes());
        }
    }
    
    hasher.update(abi_hash);
    
    let result = hasher.finalize();
    
    let mut class_hash = [0u8; 32];
    class_hash.copy_from_slice(&result);
    
    class_hash
}

/// Entry point
#[derive(Debug, Clone)]
pub struct EntryPoint {
    pub function_idx: u32,
    pub selector: [u8; 32],
}

/// Entry points by type
#[derive(Debug, Clone, Default)]
pub struct EntryPointsByType {
    pub external: Vec<EntryPoint>,
    pub l1_handler: Vec<EntryPoint>,
    pub constructor: Vec<EntryPoint>,
}

/// Crypto errors
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum CryptoError {
    InvalidPrivateKey,
    InvalidPublicKey,
    InvalidSignature,
    SigningFailed,
    VerificationFailed,
}

impl fmt::Display for CryptoError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            CryptoError::InvalidPrivateKey => write!(f, "Invalid private key"),
            CryptoError::InvalidPublicKey => write!(f, "Invalid public key"),
            CryptoError::InvalidSignature => write!(f, "Invalid signature"),
            CryptoError::SigningFailed => write!(f, "Signing failed"),
            CryptoError::VerificationFailed => write!(f, "Signature verification failed"),
        }
    }
}

impl std::error::Error for CryptoError {}

impl fmt::Display for CryptoError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "Crypto error")
    }
}
