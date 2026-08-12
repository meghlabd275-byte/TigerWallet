//! Transaction signer module — REAL ECDSA (no fake crypto).
//!
//! EVM signing uses real secp256k1 over the Keccak-256 EIP-191 personal-message
//! hash (`\x19Ethereum Signed Message:\n` + len + msg), matching go-ethereum's
//! `crypto.Sign`. The recovery byte is normalized to 27/28. Solana/Bitcoin are
//! fail-closed until a real ed25519/Bitcoin signer is wired (they return
//! `Err` rather than fabricating a signature).

use std::sync::{Arc, RwLock};
use std::collections::HashMap;

use secp256k1::{Message, Secp256k1, SecretKey, ecdsa::{RecoverableSignature, RecoveryId}};
use sha3::{Digest, Keccak256};

/// Signer interface for different chain types
pub trait Signer: Send + Sync {
    fn chain_type(&self) -> &str;
    fn sign_transaction(&self, tx: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String>;
    fn sign_message(&self, message: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String>;
    fn verify_signature(&self, message: &[u8], signature: &[u8], public_key: &[u8]) -> bool;
}

fn eip191_hash(message: &[u8]) -> [u8; 32] {
    let prefix = format!("\x19Ethereum Signed Message:\n{}", message.len());
    let mut hasher = Keccak256::new();
    hasher.update(prefix.as_bytes());
    hasher.update(message);
    let digest = hasher.finalize();
    let mut out = [0u8; 32];
    out.copy_from_slice(&digest);
    out
}

/// EVM signer using real secp256k1 (EIP-191 personal_sign).
#[derive(Clone)]
pub struct EVMSigner;

impl EVMSigner {
    pub fn new() -> Self {
        Self
    }

    /// Sign an EVM transaction hash (the caller supplies the already-hashed
    /// signing payload — typically the RLP-encoded, Keccak-256-hashed tx).
    /// Returns v||r||s with v normalized to 27/28 (EIP-155 recovery-id offset
    /// is applied by the caller if needed).
    pub fn sign_transaction(&self, tx: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String> {
        if private_key.len() != 32 {
            return Err("Invalid private key length: expected 32 bytes".to_string());
        }
        if tx.len() != 32 {
            return Err("Invalid transaction hash length: expected 32 bytes".to_string());
        }
        let secp = Secp256k1::signing_only();
        let secret = SecretKey::from_slice(private_key)
            .map_err(|e| format!("Invalid private key: {}", e))?;
        let mut hash = [0u8; 32];
        hash.copy_from_slice(tx);
        let msg = Message::from_digest(hash);
        let sig = secp.sign_ecdsa_recoverable(&msg, &secret);
        let (recovery_id, serialized) = sig.serialize_compact();
        let mut result = Vec::with_capacity(65);
        result.push(recovery_id.to_i32() as u8 + 27);
        result.extend_from_slice(&serialized);
        Ok(result)
    }

    /// Sign a message with EIP-191 personal_sign prefix.
    /// Returns v||r||s (65 bytes) with v = 27/28.
    pub fn sign_message(&self, message: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String> {
        if private_key.len() != 32 {
            return Err("Invalid private key length: expected 32 bytes".to_string());
        }
        let secp = Secp256k1::signing_only();
        let secret = SecretKey::from_slice(private_key)
            .map_err(|e| format!("Invalid private key: {}", e))?;
        let hash = eip191_hash(message);
        let msg = Message::from_digest(hash);
        let sig = secp.sign_ecdsa_recoverable(&msg, &secret);
        let (recovery_id, serialized) = sig.serialize_compact();
        let mut result = Vec::with_capacity(65);
        result.push(recovery_id.to_i32() as u8 + 27);
        result.extend_from_slice(&serialized);
        Ok(result)
    }

    /// Verify an EIP-191 personal_sign signature (v||r||s) against a public key
    /// by recovering the signer and comparing the recovered pubkey bytes.
    pub fn verify_signature(&self, message: &[u8], signature: &[u8], public_key: &[u8]) -> bool {
        if signature.len() != 65 || public_key.is_empty() {
            return false;
        }
        let v = signature[0];
        if v < 27 {
            return false;
        }
        let Ok(recovery_id) = RecoveryId::from_i32((v - 27) as i32) else { return false };
        let Ok(sig) = RecoverableSignature::from_compact(&signature[1..65], recovery_id) else { return false };
        let secp = Secp256k1::verification_only();
        let hash = eip191_hash(message);
        let msg = Message::from_digest(hash);
        let Ok(recovered) = secp.recover_ecdsa(&msg, &sig) else { return false };
        // Public key may be supplied as 33-byte compressed or 65-byte uncompressed.
        let pk_bytes = recovered.serialize();
        let pk_uncompressed = recovered.serialize_uncompressed();
        pk_bytes.as_slice() == public_key || pk_uncompressed.as_slice() == public_key
    }

    /// Recover the Ethereum address (20-byte Keccak-256 of the uncompressed
    /// pubkey, excluding the 0x04 prefix) from a personal_sign signature.
    pub fn recover_address(&self, message: &[u8], signature: &[u8]) -> Result<String, String> {
        if signature.len() != 65 {
            return Err("Invalid signature length: expected 65 bytes".to_string());
        }
        let v = signature[0];
        if v < 27 {
            return Err("Invalid recovery byte: must be >= 27".to_string());
        }
        let recovery_id = RecoveryId::from_i32((v - 27) as i32)
            .map_err(|e| format!("Invalid recovery id: {}", e))?;
        let sig = RecoverableSignature::from_compact(&signature[1..65], recovery_id)
            .map_err(|e| format!("Invalid signature: {}", e))?;
        let secp = Secp256k1::verification_only();
        let hash = eip191_hash(message);
        let msg = Message::from_digest(hash);
        let pubkey = secp.recover_ecdsa(&msg, &sig)
            .map_err(|e| format!("Recovery failed: {}", e))?;
        let uncompressed = pubkey.serialize_uncompressed();
        // Ethereum address = last 20 bytes of Keccak-256(pubkey[1..65])
        let mut hasher = Keccak256::new();
        hasher.update(&uncompressed[1..65]);
        let digest = hasher.finalize();
        Ok(format!("0x{}", hex::encode(&digest[12..32])))
    }
}

impl Default for EVMSigner {
    fn default() -> Self {
        Self::new()
    }
}

impl Signer for EVMSigner {
    fn chain_type(&self) -> &str { "evm" }

    fn sign_transaction(&self, tx: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String> {
        Self::new().sign_transaction(tx, private_key)
    }

    fn sign_message(&self, message: &[u8], private_key: &[u8]) -> Result<Vec<u8>, String> {
        Self::new().sign_message(message, private_key)
    }

    fn verify_signature(&self, message: &[u8], signature: &[u8], public_key: &[u8]) -> bool {
        Self::new().verify_signature(message, signature, public_key)
    }
}

/// Solana signer — fail-closed (real ed25519 not yet wired here).
#[derive(Clone)]
pub struct SolanaSigner;

impl SolanaSigner {
    pub fn new() -> Self {
        Self
    }
}

impl Default for SolanaSigner {
    fn default() -> Self {
        Self::new()
    }
}

impl Signer for SolanaSigner {
    fn chain_type(&self) -> &str { "solana" }

    fn sign_transaction(&self, _tx: &[u8], _private_key: &[u8]) -> Result<Vec<u8>, String> {
        Err("Solana on-chain signing is not available in masterwallet_fetchers; use the canonical tiger_solana_core crate".to_string())
    }

    fn sign_message(&self, _message: &[u8], _private_key: &[u8]) -> Result<Vec<u8>, String> {
        Err("Solana message signing is not available in masterwallet_fetchers; use the canonical tiger_solana_core crate".to_string())
    }

    fn verify_signature(&self, _message: &[u8], signature: &[u8], _public_key: &[u8]) -> bool {
        // Fail-closed: never accept without real verification.
        let _ = signature;
        false
    }
}

/// Bitcoin signer — fail-closed.
#[derive(Clone)]
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

    fn sign_transaction(&self, _tx: &[u8], _private_key: &[u8]) -> Result<Vec<u8>, String> {
        Err("Bitcoin transaction signing is not available in masterwallet_fetchers".to_string())
    }

    fn sign_message(&self, _message: &[u8], _private_key: &[u8]) -> Result<Vec<u8>, String> {
        Err("Bitcoin message signing is not available in masterwallet_fetchers".to_string())
    }

    fn verify_signature(&self, _message: &[u8], signature: &[u8], _public_key: &[u8]) -> bool {
        let _ = signature;
        false
    }
}

/// Signer registry for managing multiple signers
pub struct SignerRegistry {
    signers: Arc<RwLock<HashMap<String, Arc<dyn Signer>>>>,
}

impl SignerRegistry {
    pub fn new() -> Self {
        let mut signers: HashMap<String, Arc<dyn Signer>> = HashMap::new();
        signers.insert("evm".to_string(), Arc::new(EVMSigner::new()));
        signers.insert("solana".to_string(), Arc::new(SolanaSigner::new()));
        signers.insert("bitcoin".to_string(), Arc::new(BitcoinSigner::new()));
        Self {
            signers: Arc::new(RwLock::new(signers)),
        }
    }

    pub fn register(&self, chain_type: &str, signer: Arc<dyn Signer>) {
        if let Ok(mut signers) = self.signers.write() {
            signers.insert(chain_type.to_string(), signer);
        }
    }

    pub fn get(&self, chain_type: &str) -> Option<Arc<dyn Signer>> {
        if let Ok(signers) = self.signers.read() {
            signers.get(chain_type).cloned()
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

#[cfg(test)]
mod tests {
    use super::*;

    // Deterministic secp256k1 test private key (NOT a real funds key — test only).
    fn test_key() -> [u8; 32] {
        let mut k = [0u8; 32];
        // Use a known-valid scalar in (0, n).
        let hex_key = "4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d0a584f6c5b1";
        for i in 0..32 {
            k[i] = u8::from_str_radix(&hex_key[2 * i..2 * i + 2], 16).unwrap();
        }
        k
    }

    #[test]
    fn test_evm_sign_and_recover_address() {
        let signer = EVMSigner::new();
        let key = test_key();
        let msg = b"hello tigerwallet";
        let sig = signer.sign_message(msg, &key).unwrap();
        assert_eq!(sig.len(), 65);
        let addr = signer.recover_address(msg, &sig).unwrap();
        assert!(addr.starts_with("0x"));
        assert_eq!(addr.len(), 42); // 0x + 40 hex chars
    }

    #[test]
    fn test_evm_verify_signature_roundtrip() {
        let signer = EVMSigner::new();
        let key = test_key();
        let secp = Secp256k1::new();
        let secret = SecretKey::from_slice(&key).unwrap();
        let pubkey = secp256k1::PublicKey::from_secret_key(&secp, &secret);
        let uncompressed = pubkey.serialize_uncompressed();
        let msg = b"verify me";
        let sig = signer.sign_message(msg, &key).unwrap();
        assert!(signer.verify_signature(msg, &sig, &uncompressed));
        // Tampered message must fail
        assert!(!signer.verify_signature(b"tampered", &sig, &uncompressed));
    }

    #[test]
    fn test_evm_rejects_bad_key_length() {
        let signer = EVMSigner::new();
        let res = signer.sign_message(b"msg", &[0u8; 16]);
        assert!(res.is_err());
    }

    #[test]
    fn test_solana_fail_closed() {
        let s = SolanaSigner::new();
        assert!(s.sign_message(b"msg", &[0u8; 64]).is_err());
        assert!(!s.verify_signature(b"msg", &[0u8; 64], &[0u8; 32]));
    }
}
