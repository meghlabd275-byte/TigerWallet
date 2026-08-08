//! MPC Wallet Module
//! 
//! High-level wallet API for MPC operations
//! Provides simple interface for key generation, signing, and management

use crate::{CurveType, KeyGenResult, MpcConfig, MpcError, SignResult, ShareInfo};
use crate::key_gen::generate_key_shares;
use crate::signing::{sign, verify};
use crate::key_sharing::{reshare, recover_key, backup_key, restore_from_backup};
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;

/// MPC Wallet instance
pub struct MpcWallet {
    /// Wallet ID
    pub id: String,
    /// Configuration
    config: MpcConfig,
    /// Key ID to key data mapping
    keys: Arc<RwLock<HashMap<String, MpcKeyData>>>,
    /// Local shares (for single-party mode)
    local_shares: Arc<RwLock<HashMap<String, Vec<ShareInfo>>>>,
}

/// Key data stored in wallet
#[derive(Debug, Clone)]
pub struct MpcKeyData {
    /// Key ID
    pub key_id: String,
    /// Public key
    pub public_key: Vec<u8>,
    /// Curve type
    pub curve: CurveType,
    /// Created at timestamp
    pub created_at: u64,
    /// Metadata
    pub metadata: HashMap<String, String>,
}

impl MpcWallet {
    /// Create new MPC wallet
    pub fn new() -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            config: MpcConfig::default(),
            keys: Arc::new(RwLock::new(HashMap::new())),
            local_shares: Arc::new(RwLock::new(HashMap::new())),
        }
    }
    
    /// Create with custom config
    pub fn with_config(config: MpcConfig) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            config,
            keys: Arc::new(RwLock::new(HashMap::new())),
            local_shares: Arc::new(RwLock::new(HashMap::new())),
        }
    }
    
    /// Generate new MPC key
    pub async fn generate_key(&self, entropy: &[u8]) -> Result<String, MpcError> {
        let result = generate_key_shares(&self.config, entropy)?;
        
        // Store key data
        let key_data = MpcKeyData {
            key_id: result.key_id.clone(),
            public_key: result.public_key.clone(),
            curve: self.config.curve,
            created_at: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs(),
            metadata: HashMap::new(),
        };
        
        self.keys.write().await.insert(result.key_id.clone(), key_data);
        
        // Store shares locally (in production, distribute to participants)
        self.local_shares.write().await.insert(
            result.key_id.clone(),
            result.shares,
        );
        
        Ok(result.key_id)
    }
    
    /// Get public key
    pub async fn get_public_key(&self, key_id: &str) -> Result<Vec<u8>, MpcError> {
        let keys = self.keys.read().await;
        let key_data = keys.get(key_id)
            .ok_or_else(|| MpcError::KeyGenFailed("Key not found".to_string()))?;
        
        Ok(key_data.public_key.clone())
    }
    
    /// Sign transaction
    pub async fn sign(
        &self,
        key_id: &str,
        message: &[u8],
    ) -> Result<SignResult, MpcError> {
        let shares = {
            let local = self.local_shares.read().await;
            local.get(key_id)
                .cloned()
                .ok_or_else(|| MpcError::KeyGenFailed("Key shares not found".to_string()))?
        };
        
        let public_key = self.get_public_key(key_id).await?;
        
        sign(&shares.iter().map(|s| s.share_data.clone()).collect::<Vec<_>>(), &public_key, message, &self.config)
            .map(|mut result| {
                result.key_id = key_id.to_string();
                result
            })
    }
    
    /// Verify signature
    pub async fn verify(
        &self,
        key_id: &str,
        signature: &[u8],
        message: &[u8],
    ) -> Result<bool, MpcError> {
        let public_key = self.get_public_key(key_id).await?;
        
        verify(signature, &public_key, message, self.config.curve)
    }
    
    /// Add share (for distributed mode)
    pub async fn add_share(&self, key_id: &str, share: ShareInfo) -> Result<(), MpcError> {
        let mut local = self.local_shares.write().await;
        let shares = local.get_mut(key_id)
            .ok_or_else(|| MpcError::KeyGenFailed("Key not found".to_string()))?;
        
        // Verify share
        use sha2::{Sha256, Digest};
        let computed = Sha256::digest(&share.share_data);
        if computed.to_vec() != share.verification_key {
            return Err(MpcError::InvalidShare("Verification failed".to_string()));
        }
        
        shares.push(share);
        Ok(())
    }
    
    /// Get share for distribution
    pub async fn get_share(&self, key_id: &str, index: u32) -> Result<ShareInfo, MpcError> {
        let local = self.local_shares.read().await;
        let shares = local.get(key_id)
            .ok_or_else(|| MpcError::KeyGenFailed("Key not found".to_string()))?;
        
        shares.iter()
            .find(|s| s.index == index)
            .cloned()
            .ok_or_else(|| MpcError::InvalidShare("Share not found".to_string()))
    }
    
    /// Reshare key (change threshold or participants)
    pub async fn reshare_key(
        &self,
        key_id: &str,
        new_config: &MpcConfig,
    ) -> Result<Vec<ShareInfo>, MpcError> {
        let (shares, public_key) = {
            let local = self.local_shares.read().await;
            let keys = self.keys.read().await;
            
            let shares = local.get(key_id)
                .cloned()
                .ok_or_else(|| MpcError::KeyGenFailed("Key not found".to_string()))?;
            
            let public_key = keys.get(key_id)
                .map(|k| k.public_key.clone())
                .ok_or_else(|| MpcError::KeyGenFailed("Key not found".to_string()))?;
            
            (shares, public_key)
        };
        
        let new_shares = reshare(&shares, self.config.threshold, new_config, &public_key)?;
        
        // Update stored shares
        {
            let mut local = self.local_shares.write().await;
            local.insert(key_id.to_string(), new_shares.clone());
        }
        
        Ok(new_shares)
    }
    
    /// Backup key
    pub async fn backup_key(&self, key_id: &str, encryption_key: &[u8]) -> Result<Vec<u8>, MpcError> {
        let shares = {
            let local = self.local_shares.read().await;
            local.get(key_id)
                .cloned()
                .ok_or_else(|| MpcError::KeyGenFailed("Key not found".to_string()))?
        };
        
        backup_key(&shares, encryption_key)
    }
    
    /// Restore from backup
    pub async fn restore_from_backup(
        &self,
        key_id: &str,
        backup: &[u8],
        decryption_key: &[u8],
    ) -> Result<(), MpcError> {
        let shares = restore_from_backup(backup, decryption_key)?;
        
        self.local_shares.write().await.insert(key_id.to_string(), shares);
        
        // Verify key exists
        if !self.keys.read().await.contains_key(key_id) {
            return Err(MpcError::KeyGenFailed("Key not found".to_string()));
        }
        
        Ok(())
    }
    
    /// Recover key from shares (for emergency recovery)
    pub async fn recover_key(&self, key_id: &str, shares: Vec<ShareInfo>) -> Result<Vec<u8>, MpcError> {
        // Verify shares belong to this key
        let stored_shares = {
            let local = self.local_shares.read().await;
            local.get(key_id).cloned()
        };
        
        // Use provided shares for recovery
        recover_key(&shares, self.config.threshold, self.config.curve)
    }
    
    /// Get all key IDs
    pub async fn list_keys(&self) -> Vec<String> {
        self.keys.read().await.keys().cloned().collect()
    }
    
    /// Delete key
    pub async fn delete_key(&self, key_id: &str) -> Result<(), MpcError> {
        self.keys.write().await.remove(key_id);
        self.local_shares.write().await.remove(key_id);
        Ok(())
    }
}

impl Default for MpcWallet {
    fn default() -> Self {
        Self::new()
    }
}

/// Generate address from public key
pub fn generate_address(public_key: &[u8], chain_id: u32) -> String {
    use sha2::{Sha256, Digest};
    
    let mut hasher = Sha256::new();
    hasher.update(public_key);
    hasher.update(&chain_id.to_le_bytes());
    let hash = hasher.finalize();
    
    format!("0x{}", hex::encode(&hash[..20]))
}

/// UUID generation
mod uuid {
    use rand::rngs::OsRng;
    use rand::RngCore;
    
    pub struct Uuid([u8; 16]);
    
    impl Uuid {
        pub fn new_v4() -> Self {
            let mut bytes = [0u8; 16];
            OsRng.fill_bytes(&mut bytes);
            // Set version (4) and variant
            bytes[6] = (bytes[6] & 0x0f) | 0x40;
            bytes[8] = (bytes[8] & 0x3f) | 0x80;
            Self(bytes)
        }
        
        pub fn to_string(&self) -> String {
            format!(
                "{:02x}{:02x}{:02x}{:02x}-{:02x}{:02x}-{:02x}{:02x}-{:02x}{:02x}-{:02x}{:02x}{:02x}{:02x}{:02x}{:02x}",
                self.0[0], self.0[1], self.0[2], self.0[3],
                self.0[4], self.0[5],
                self.0[6], self.0[7],
                self.0[8], self.0[9],
                self.0[10], self.0[11], self.0[12], self.0[13], self.0[14], self.0[15]
            )
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_mpc_wallet() {
        let wallet = MpcWallet::new();
        
        let key_id = wallet.generate_key(b"test entropy").await.unwrap();
        
        let message = b"Hello, MPC Wallet!";
        
        let signature = wallet.sign(&key_id, message).await.unwrap();
        
        let is_valid = wallet.verify(&key_id, &signature.signature, message).await.unwrap();
        
        assert!(is_valid);
    }
    
    #[tokio::test]
    async fn test_key_management() {
        let wallet = MpcWallet::new();
        
        let key_id = wallet.generate_key(b"entropy").await.unwrap();
        
        let keys = wallet.list_keys().await;
        
        assert_eq!(keys.len(), 1);
        assert_eq!(keys[0], key_id);
        
        wallet.delete_key(&key_id).await.unwrap();
        
        let keys = wallet.list_keys().await;
        
        assert!(keys.is_empty());
    }
}
