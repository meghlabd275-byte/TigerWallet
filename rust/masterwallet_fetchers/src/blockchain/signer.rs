//! Transaction signer module
//! Provides transaction signing functionality for multiple chains

use std::sync::{Arc, RwLock};
use std::collections::HashMap;

/// Signer interface for different chain types
pub trait Signer: Send + Sync {
    fn chain_type(&self) -> &str;
    fn sign_transaction(&self, tx: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String>;
    fn sign_message(&self, message: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String>;
    fn verify_signature(&self, message: &[u8], signature: &[u8], public_key: &[u8]) -> bool;
}

/// EVM signer using secp256k1
pub struct EVMSigner;

impl EVMSigner {
    pub fn new() -> Self {
        Self
    }
    
    /// Sign an EVM transaction (simplified - in production use proper EIP-155)
    pub fn sign_transaction(&self, tx: &[u8], _private_key: &[u8]) -> Result<Vec<u8>, String> {
        // In production, this would:
        // 1. Parse the transaction
        // 2. Apply EIP-155 signing (chain_id)
        // 3. Sign with secp256k1
        // 4. Encode as RLP
        
        // For now, return the tx as-is (would need proper implementation)
        Ok(tx.to_vec())
    }
    
    /// Sign a message (personal sign)
    pub fn sign_message(&self, message: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String> {
        // In production, this would use proper cryptographic signing
        // Message prefix: "\x19Ethereum Signed Message:\n" + len(message)
        
        if private_key.len() != 32 {
            return Err("Invalid private key length".to_string());
        }
        
        // Simplified - in production use proper secp256k1
        let mut result = Vec::new();
        result.extend_from_slice(private_key);
        result.extend_from_slice(message);
        Ok(result)
    }
    
    /// Verify a signature
    pub fn verify_signature(&self, message: &[u8], signature: &[u8], _public_key: &[u8]) -> bool {
        // In production, use proper signature verification
        signature.len() == 64 || signature.len() == 65
    }
    
    /// Recover address from signature
    pub fn recover_address(&self, message: &[u8], signature: &[u8]) -> Result<String, String> {
        // In production, use proper recovery
        // For now, return a placeholder
        Ok(format!("0x{:x}", message.len()))
    }
}

impl Default for EVMSigner {
    fn default() -> Self {
        Self::new()
    }
}

/// Solana signer
pub struct SolanaSigner;

impl SolanaSigner {
    pub fn new() -> Self {
        Self
    }
    
    pub fn sign_transaction(&self, tx: &[u8], _private_key: &[u8]) -> Result<Vec<u8>, String> {
        // Solana transaction signing would go here
        Ok(tx.to_vec())
    }
}

impl Default for SolanaSigner {
    fn default() -> Self {
        Self::new()
    }
}

impl Signer for SolanaSigner {
    fn chain_type(&self) -> &str { "solana" }
    
    fn sign_transaction(&self, tx: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String> {
        Self::new().sign_transaction(tx, private_key)
    }
    
    fn sign_message(&self, message: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String> {
        if private_key.len() != 64 {
            return Err("Invalid Solana private key length".to_string());
        }
        
        let mut result = Vec::new();
        result.extend_from_slice(private_key);
        result.extend_from_slice(message);
        Ok(result)
    }
    
    fn verify_signature(&self, _message: &[u8], signature: &[u8], _public_key: &[u8]) -> bool {
        signature.len() == 64
    }
}

/// Bitcoin signer
pub struct BitcoinSigner;

impl BitcoinSigner {
    pub fn new() -> Self {
        Self
    }
}

impl Default for BitcoinSigner {
    fn default() -> Self {
        Self::new()
    }
}

impl Signer for BitcoinSigner {
    fn chain_type(&self) -> &str { "bitcoin" }
    
    fn sign_transaction(&self, tx: &[u8], _private_key: &[u8]) -> Result<Vec<u8>, String> {
        // Bitcoin transaction signing
        Ok(tx.to_vec())
    }
    
    fn sign_message(&self, message: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String> {
        if private_key.len() != 32 {
            return Err("Invalid Bitcoin private key length".to_string());
        }
        
        let mut result = Vec::new();
        result.extend_from_slice(private_key);
        result.extend_from_slice(message);
        Ok(result)
    }
    
    fn verify_signature(&self, _message: &[u8], signature: &[u8], _public_key: &[u8]) -> bool {
        signature.len() >= 64
    }
}

/// Signer registry for managing multiple signers
pub struct SignerRegistry {
    signers: Arc<RwLock<HashMap<String, Box<dyn Signer>>>>,
}

impl SignerRegistry {
    pub fn new() -> Self {
        let registry = Self {
            signers: Arc::new(RwLock::new(HashMap::new())),
        };
        
        // Register default signers
        {
            let mut signers = registry.signers.write().unwrap();
            signers.insert("EVM".to_string(), Box::new(EVMSigner::new()));
            signers.insert("Solana".to_string(), Box::new(SolanaSigner::new()));
            signers.insert("Bitcoin".to_string(), Box::new(BitcoinSigner::new()));
        }
        
        registry
    }
    
    pub fn register(&self, chain_type: &str, signer: Box<dyn Signer>) {
        if let Ok(mut signers) = self.signers.write() {
            signers.insert(chain_type.to_string(), signer);
        }
    }
    
    pub fn get(&self, chain_type: &str) -> Option<Box<dyn Signer>> {
        if let Ok(signers) = self.signers.read() {
            signers.get(chain_type).map(|s| s.boxed_clone())
        } else {
            None
        }
    }
    
    pub fn sign_transaction(&self, chain_type: &str, tx: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String> {
        let signer = self.get(chain_type)
            .ok_or_else(|| format!("No signer found for chain type: {}", chain_type))?;
        signer.sign_transaction(tx, private_key)
    }
    
    pub fn sign_message(&self, chain_type: &str, message: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String> {
        let signer = self.get(chain_type)
            .ok_or_else(|| format!("No signer found for chain type: {}", chain_type))?;
        signer.sign_message(message, private_key)
    }
}

impl Default for SignerRegistry {
    fn default() -> Self {
        Self::new()
    }
}

impl Clone for SignerRegistry {
    fn clone(&self) -> Self {
        Self {
            signers: self.signers.clone(),
        }
    }
}

trait SignerBoxedClone {
    fn boxed_clone(&self) -> Box<dyn Signer>;
}

impl<T: Signer + Clone> SignerBoxedClone for T {
    fn boxed_clone(&self) -> Box<dyn Signer> {
        Box::new(self.clone())
    }
}
