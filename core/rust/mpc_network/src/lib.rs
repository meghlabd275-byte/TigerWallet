//! TigerSwap MPC Network - Production-Ready Distributed Key Management
//! 
//! Complete MPC (Multi-Party Computation) implementation for institutional custody:
//! - MPC Coordinator: Orchestrates key generation and signing ceremonies
//! - MPC Nodes: Distributed signing nodes with threshold key shares
//! - Threshold Signing: t-of-n signature generation
//! - Key Resharing: Dynamic share redistribution
//! - Key Rotation: Periodic key rotation with security

use std::collections::HashMap;

use serde::{Deserialize, Serialize};
use thiserror::Error;

// ============================================================================
// Error Types
// ============================================================================

#[derive(Error, Debug)]
pub enum MPCError {
    #[error("Invalid key share")]
    InvalidKeyShare,
    #[error("Insufficient signatures: needed {0}, got {1}")]
    InsufficientSignatures(usize, usize),
    #[error("Signing ceremony failed: {0}")]
    CeremonyFailed(String),
    #[error("Node communication error: {0}")]
    NodeError(String),
    #[error("Invalid signature")]
    InvalidSignature,
    #[error("Key not found")]
    KeyNotFound,
    #[error("Invalid threshold: need {0}, have {1} nodes")]
    InvalidThreshold(usize, usize),
    #[error("Encryption failed")]
    EncryptionFailed,
    #[error("Decryption failed")]
    DecryptionFailed,
    #[error("Timeout waiting for responses")]
    Timeout,
    #[error("Invalid participant")]
    InvalidParticipant,
}

// ============================================================================
// Signing Session
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SigningSession {
    pub id: String,
    pub session_id: String,
    pub message: Vec<u8>,
    pub message_hash: [u8; 32],
    pub threshold: u32,
    pub total_nodes: u32,
    pub partial_signatures: HashMap<String, PartialSignature>,
    pub started_at: i64,
    pub status: SigningStatus,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum SigningStatus {
    Pending,
    Collecting,
    Ready,
    Completed,
    Failed,
}


// ============================================================================
// Library Exports
// ============================================================================

pub use self::{
    key_share::KeyShare,
    public_key_share::PublicKeyShare,
    threshold_signature::{ThresholdSignature, PartialSignature},
    key_gen::{KeyGenParams, KeyGenState, KeyGenStatus, KeyType},
    mpc_node::MPCNode,
    mpc_coordinator::MPCCoordinator,
    resharing::{ReshareRequest, KeyRotationRequest, RotationReason},
    encrypted_share::EncryptedShare,
    rpc_types::{NodeRequest, NodeResponse},
    mpc_network::MPCNetwork,
};

mod key_share;
mod public_key_share;
mod threshold_signature;
mod key_gen;
mod mpc_node;
mod mpc_coordinator;
mod resharing;
mod encrypted_share;
mod rpc_types;
mod mpc_network;

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_key_share() {
        let share = KeyShare::new();
        let hex = share.to_hex();
        assert_eq!(hex.len(), 64);
    }
    
    #[test]
    fn test_encrypted_share() {
        let share_data = b"test_key_share_data_32_bytes!".to_vec();
        let encrypted = EncryptedShare::encrypt(
            &share_data,
            "node1",
            "session1",
        ).unwrap();
        
        let decrypted = encrypted.decrypt().unwrap();
        assert_eq!(decrypted, share_data);
    }
    
    #[tokio::test]
    async fn test_coordinator_registration() {
        let coordinator = MPCCoordinator::new();
        
        let node = MPCNode::new(
            "node1".to_string(),
            "127.0.0.1:8080".parse().unwrap(),
        );
        
        coordinator.register_node(node).await;
        
        let nodes = coordinator.get_nodes().await;
        assert_eq!(nodes.len(), 1);
    }
    
    #[tokio::test]
    async fn test_key_generation() {
        let coordinator = MPCCoordinator::new();
        
        // Register nodes
        for i in 1..=3 {
            let node = MPCNode::new(
                format!("node{}", i),
                format!("127.0.0.1:{}", 8080 + i).parse().unwrap(),
            );
            coordinator.register_node(node).await;
        }
        
        // Start key generation
        let params = KeyGenParams::new(2, 3);
        let session_id = coordinator.start_key_generation(params).await.unwrap();
        
        assert!(!session_id.is_empty());
    }
    
    #[tokio::test]
    async fn test_signing_ceremony() {
        let coordinator = MPCCoordinator::new();
        
        // Register nodes
        for i in 1..=3 {
            let node = MPCNode::new(
                format!("node{}", i),
                format!("127.0.0.1:{}", 8080 + i).parse().unwrap(),
            );
            coordinator.register_node(node).await;
        }
        
        // Start key generation
        let params = KeyGenParams::new(2, 3);
        coordinator.start_key_generation(params).await.unwrap();
        
        // Start signing
        let signing_id = coordinator
            .start_signing("test-session", b"test message".to_vec())
            .await
            .unwrap();
        
        assert!(!signing_id.is_empty());
    }
}

// ============================================================================
