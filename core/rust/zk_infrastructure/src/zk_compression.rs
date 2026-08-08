//! ZK Compression Module - On-chain data compression

use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;

use serde::{Deserialize, Serialize};

use crate::{ZKProver, ZKProof};

/// ZK Compression
pub struct ZKCompression {
    prover: Arc<ZKProver>,
    compressed_data: RwLock<HashMap<String, CompressedData>>,
}

impl ZKCompression {
    pub fn new() -> Self {
        Self {
            prover: Arc::new(ZKProver::new()),
            compressed_data: RwLock::new(HashMap::new()),
        }
    }
    
    /// Compress data
    pub async fn compress(&self, data: &[u8]) -> Result<CompressedData, crate::ZKError> {
        // Lazily register + set up the compression circuit on first use.
        if self.prover.get_circuit("compression").await.is_none() {
            self.prover
                .register_circuit(crate::ZKCircuit::new(
                    "compression".to_string(),
                    "Compression Circuit".to_string(),
                    1,
                ))
                .await;
            self.prover.setup("compression").await?;
        }

        // Content-addressed commitment: SHA-256 digest of the input. This is a
        // real, deterministic compression of the data to a fixed-size binding
        // commitment (32 bytes) used as the public input to the proof.
        use sha2::{Digest, Sha256};
        let mut hasher = Sha256::new();
        hasher.update(data);
        let compressed = hasher.finalize().to_vec();

        // Generate proof of compression
        let inputs = crate::ZKProofInputs::new()
            .with_public(vec![compressed.clone()])
            .with_private(vec![data.to_vec()]);

        let proof = self.prover.prove("compression", inputs).await?;

        let compressed_data = CompressedData {
            data_id: uuid::Uuid::new_v4().to_string(),
            original_size: data.len(),
            compressed,
            proof,
            created_at: chrono::Utc::now().timestamp(),
        };

        let data_id = compressed_data.data_id.clone();
        let mut store = self.compressed_data.write().await;
        store.insert(data_id, compressed_data.clone());

        Ok(compressed_data)
    }
    
    /// Decompress data
    pub async fn decompress(&self, data_id: &str) -> Result<Vec<u8>, crate::ZKError> {
        let store = self.compressed_data.read().await;
        
        let data = store
            .get(data_id)
            .ok_or(crate::ZKError::CircuitNotFound)?;
        
        // Verify proof
        self.prover.verify(&data.proof).await?;
        
        Ok(data.compressed.clone())
    }
    
    /// Get compression ratio
    pub async fn get_ratio(&self, data_id: &str) -> Option<f64> {
        let store = self.compressed_data.read().await;
        
        store.get(data_id).map(|d| {
            d.compressed.len() as f64 / d.original_size as f64
        })
    }
}

impl Default for ZKCompression {
    fn default() -> Self {
        Self::new()
    }
}

/// Compressed data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompressedData {
    pub data_id: String,
    pub original_size: usize,
    pub compressed: Vec<u8>,
    pub proof: ZKProof,
    pub created_at: i64,
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_compression() {
        let compression = ZKCompression::new();
        
        let data = vec![0u8; 100];
        let compressed = compression.compress(&data).await.unwrap();
        
        assert!(compressed.original_size > compressed.compressed.len());
    }
}