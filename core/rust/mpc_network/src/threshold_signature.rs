//! Threshold Signature Module - Threshold signature generation and combination

use serde::{Deserialize, Serialize};
use serde_big_array::BigArray;
use thiserror::Error;

use crate::{MPCError, PublicKeyShare};

#[derive(Error, Debug)]
pub enum SignatureError {
    #[error("Invalid partial signature")]
    InvalidPartialSignature,
    #[error("Insufficient partial signatures")]
    InsufficientSignatures,
    #[error("Signature combination failed")]
    CombinationFailed,
}

/// Wrapper for a 65-byte combined signature (serde supports arrays up to 32 natively).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct CombinedSignature(#[serde(with = "BigArray")] [u8; 65]);

impl CombinedSignature {
    pub fn new(arr: [u8; 65]) -> Self {
        Self(arr)
    }
    pub fn as_bytes(&self) -> &[u8; 65] {
        &self.0
    }
}

/// Wrapper for a 33-byte combined public key (serde supports arrays up to 32 natively).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct CombinedPubKey(#[serde(with = "BigArray")] [u8; 33]);

impl CombinedPubKey {
    pub fn new(arr: [u8; 33]) -> Self {
        Self(arr)
    }
    pub fn as_bytes(&self) -> &[u8; 33] {
        &self.0
    }
}

/// Partial signature from a single MPC node
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PartialSignature {
    pub node_id: String,
    #[serde(with = "BigArray")]
    pub signature: [u8; 64],
    #[serde(with = "BigArray")]
    pub public_share: [u8; 33],
    pub message_hash: [u8; 32],
    pub timestamp: i64,
}

impl PartialSignature {
    pub fn new(node_id: String, signature: [u8; 64], public_share: [u8; 33]) -> Self {
        let message_hash = [0u8; 32];
        Self {
            node_id,
            signature,
            public_share,
            message_hash,
            timestamp: chrono::Utc::now().timestamp(),
        }
    }
    
    pub fn verify(&self) -> bool {
        // Simplified verification
        // In production, verify against the public share
        !self.signature.iter().all(|&b| b == 0)
    }
}

/// Combined threshold signature
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ThresholdSignature {
    pub signatures: Vec<PartialSignature>,
    pub message_hash: [u8; 32],
    pub threshold: u32,
    pub total_shares: u32,
    pub combined_signature: Option<CombinedSignature>,
    pub combined_public_key: Option<CombinedPubKey>,
}

impl ThresholdSignature {
    /// Check if we have enough signatures
    pub fn is_complete(&self) -> bool {
        self.signatures.len() >= self.threshold as usize
    }
    
    /// Combine partial signatures into final signature
    pub fn combine(&mut self) -> Result<(), SignatureError> {
        if !self.is_complete() {
            return Err(SignatureError::InsufficientSignatures);
        }
        
        // Simplified combination using XOR (BLS-like in production)
        let mut combined_sig = [0u8; 65];
        let mut combined_pubkey = [0u8; 33];

        for partial in &self.signatures {
            for i in 0..64 {
                combined_sig[i] ^= partial.signature[i];
            }
            for i in 0..33 {
                combined_pubkey[i] ^= partial.public_share[i];
            }
        }

        self.combined_signature = Some(CombinedSignature::new(combined_sig));
        self.combined_public_key = Some(CombinedPubKey::new(combined_pubkey));

        Ok(())
    }

    /// Get combined signature bytes
    pub fn signature_bytes(&self) -> Option<&[u8; 65]> {
        self.combined_signature.as_ref().map(|s| s.as_bytes())
    }

    /// Get combined public key bytes
    pub fn public_key_bytes(&self) -> Option<&[u8; 33]> {
        self.combined_public_key.as_ref().map(|k| k.as_bytes())
    }

    /// Serialize to ethereum signature format
    pub fn to_ethereum_signature(&self) -> Option<String> {
        self.combined_signature.as_ref().map(|sig| hex::encode(sig.as_bytes()))
    }
}

/// Signature commitment for verifiable secret sharing
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignatureCommitment {
    pub commitment: [u8; 32],
    pub round: u32,
    pub node_id: String,
}

impl SignatureCommitment {
    pub fn new(node_id: String, commitment: [u8; 32], round: u32) -> Self {
        Self {
            commitment,
            round,
            node_id,
        }
    }
}

/// Proof of knowledge for zero-knowledge proofs
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KnowledgeProof {
    pub challenge: [u8; 32],
    pub response: [u8; 32],
    #[serde(with = "BigArray")]
    pub public_share: [u8; 33],
}

impl KnowledgeProof {
    pub fn verify(&self) -> bool {
        // Simplified verification
        !self.challenge.iter().all(|&b| b == 0) && !self.response.iter().all(|&b| b == 0)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_partial_signature() {
        let sig = PartialSignature::new(
            "node1".to_string(),
            [1u8; 64],
            [2u8; 33],
        );
        
        assert!(sig.verify());
    }
    
    #[test]
    fn test_threshold_signature_combination() {
        let mut ts = ThresholdSignature {
            signatures: vec![
                PartialSignature::new("node1".to_string(), [1u8; 64], [2u8; 33]),
                PartialSignature::new("node2".to_string(), [3u8; 64], [4u8; 33]),
            ],
            message_hash: [0u8; 32],
            threshold: 2,
            total_shares: 3,
            combined_signature: None,
            combined_public_key: None,
        };
        
        ts.combine().unwrap();
        
        assert!(ts.signature_bytes().is_some());
    }
}