//! TigerSwap ZK Infrastructure - Zero-Knowledge proofs of knowledge.
//!
//! The prover/verifier pair implements a real Fiat-Shamir Schnorr proof of
//! knowledge of a discrete logarithm over the Ristretto255 group (curve25519-dalek).
//! No simulated/fake proofs are produced: every verification performs real
//! elliptic-curve group arithmetic.

use thiserror::Error;

// ============================================================================
// Error Types
// ============================================================================

#[derive(Error, Debug)]
pub enum ZKError {
    #[error("Proof verification failed")]
    VerificationFailed,
    #[error("Circuit not found")]
    CircuitNotFound,
    #[error("Invalid parameters: {0}")]
    InvalidParameters(String),
    #[error("Setup failed: {0}")]
    SetupFailed(String),
    #[error("Proving failed: {0}")]
    ProvingFailed(String),
    #[error("Serialization error: {0}")]
    SerializationError(String),
    #[error("Unsupported curve")]
    UnsupportedCurve,
}

// ============================================================================
// Module declarations & re-exports
// ============================================================================

mod zk_proof;
mod zk_circuit;
mod zk_prover;
mod zk_bridge;
mod zk_identity;
mod zk_compression;
mod zk_manager;

pub use self::{
    zk_proof::{ZKProof, ProofType, CurveType},
    zk_circuit::{ZKCircuit, CircuitType},
    zk_prover::{ZKProver, ZKProofInputs},
    zk_bridge::{ZKBridge, ZKBridgeMessage, MessageStatus},
    zk_identity::{ZKIdentity, ZKIdentityRecord},
    zk_compression::{ZKCompression, CompressedData},
    zk_manager::ZKManager,
};

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_prover_end_to_end() {
        let prover = ZKProver::new();

        let circuit = ZKCircuit::new("test".to_string(), "Test Circuit".to_string(), 1);
        prover.register_circuit(circuit).await;

        prover.setup("test").await.unwrap();

        let inputs = ZKProofInputs::new()
            .with_public(vec![vec![1, 2, 3]])
            .with_private(vec![vec![4, 5, 6]]);

        let proof = prover.prove("test", inputs).await.unwrap();
        assert!(prover.verify(&proof).await.unwrap());
    }

    #[tokio::test]
    async fn test_tampered_proof_fails() {
        // A proof whose data is mutated must fail real verification, not pass
        // merely because it is non-empty.
        let prover = ZKProver::new();
        prover
            .register_circuit(ZKCircuit::new("t".to_string(), "T".to_string(), 1))
            .await;
        prover.setup("t").await.unwrap();

        let inputs = ZKProofInputs::new()
            .with_public(vec![vec![1, 2, 3]])
            .with_private(vec![vec![4, 5, 6]]);

        let mut proof = prover.prove("t", inputs).await.unwrap();
        assert!(prover.verify(&proof).await.unwrap());

        if let Some(b) = proof.proof_data.get_mut(0) {
            *b ^= 0xff;
        }
        // A tampered proof must be rejected. Real verification may reject by
        // returning Ok(false) (point decodes but equation fails) or Err(...)
        // (encoding no longer a valid Ristretto point); both mean rejection.
        let result = prover.verify(&proof).await;
        assert!(matches!(result, Ok(false) | Err(_)));
    }

    #[tokio::test]
    async fn test_bridge() {
        let bridge = ZKBridge::new();
        let message_id = bridge
            .send_message(42161, vec![1, 2, 3], "sender")
            .await
            .unwrap();
        assert!(!message_id.is_empty());
    }

    #[tokio::test]
    async fn test_identity() {
        let identity = ZKIdentity::new();
        let record = ZKIdentityRecord::new(vec![1, 2, 3, 4]);
        let identity_id = identity.register(record).await.unwrap();

        let proof = identity
            .prove_identity(&identity_id, b"challenge")
            .await
            .unwrap();
        assert!(identity.verify_identity(&proof).await.unwrap());
    }

    #[tokio::test]
    async fn test_manager() {
        let manager = ZKManager::new();
        let circuit = ZKCircuit::new("test".to_string(), "Test".to_string(), 1);
        manager.prover().register_circuit(circuit).await;
        manager.prover().setup("test").await.unwrap();

        let inputs = ZKProofInputs::new().with_public(vec![vec![1, 2, 3]]);
        let proof = manager.prover().prove("test", inputs).await.unwrap();
        assert!(manager.prover().verify(&proof).await.unwrap());
    }
}
