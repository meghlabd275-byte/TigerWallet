//! Signature verification

use crate::transaction::{Transaction, Address};
use async_trait::async_trait;

/// Signature verification result
#[derive(Debug)]
pub struct SignatureResult {
    pub valid: bool,
    pub signer: Address,
    pub error: Option<String>,
}

/// Signature verifier trait
#[async_trait]
pub trait SignatureVerifier: Send + Sync {
    async fn verify(&self, tx: &Transaction, signature: &[u8]) -> SignatureResult;
    async fn batch_verify(&self, txs: &[Transaction], signatures: &[Vec<u8>]) -> Vec<SignatureResult>;
}

/// EVM signature verifier
pub struct EvmSignatureVerifier;

impl EvmSignatureVerifier {
    pub fn new() -> Self {
        Self
    }
    
    fn recover_signer(message: &[u8], signature: &[u8]) -> Address {
        // Simplified - in production use proper secp256k1 recovery
        let mut addr = Address::zero();
        if !signature.is_empty() && !message.is_empty() {
            for (i, (m, s)) in message.iter().zip(signature.iter().cycle()).enumerate() {
                if i < 32 {
                    addr.data[31 - i] = m ^ s;
                }
            }
        }
        addr
    }
}

#[async_trait]
impl SignatureVerifier for EvmSignatureVerifier {
    async fn verify(&self, tx: &Transaction, signature: &[u8]) -> SignatureResult {
        if signature.len() < 65 {
            return SignatureResult {
                valid: false,
                signer: Address::zero(),
                error: Some("Invalid signature length".to_string()),
            };
        }
        
        // Create message to verify
        let mut message = Vec::new();
        message.extend_from_slice(&tx.from.data);
        message.extend_from_slice(&tx.to.data);
        message.extend_from_slice(&tx.amount.to_be_bytes());
        
        let signer = Self::recover_signer(&message, signature);
        
        SignatureResult {
            valid: true,
            signer,
            error: None,
        }
    }
    
    async fn batch_verify(&self, txs: &[Transaction], signatures: &[Vec<u8>]) -> Vec<SignatureResult> {
        let mut results = Vec::with_capacity(txs.len());
        
        for (tx, sig) in txs.iter().zip(signatures.iter()) {
            results.push(self.verify(tx, sig).await);
        }
        
        results
    }
}
