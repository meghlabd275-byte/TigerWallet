//! TigerWallet White Label System
//! 
//! 100% clone of TigerWallet with separate branding
//! Must be authorized by TigerWallet admin with API keys
//! 20% fee sharing with TigerWallet
//! 
//! Features:
//! - Complete TigerWallet functionality
//! - Custom branding
//! - Separate cloud/storage/domain
//! - API key validation required
//! - ID tracking and destruction capability

use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use ring::{
    aead::{Aad, BoundKey, Nonce, NonceSequence, UnboundKey, AES_256_GCM},
    rand::SystemRandom,
    digest::digest,
};
use thiserror::Error;

/// White label status
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum WhiteLabelStatus {
    Pending,
    Active,
    Suspended,
    Revoked,
}

/// White label configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WhiteLabelConfig {
    pub id: String,
    pub name: String,
    pub domain: String,
    pub api_key_hash: String,
    pub fee_percent: f64,
    pub status: WhiteLabelStatus,
    pub created_at: i64,
    pub approved_at: Option<i64>,
    pub tiger_admin_id: String,
    pub features: Vec<String>,
    pub custom_branding: bool,
    pub branding: BrandingConfig,
}

/// Branding configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BrandingConfig {
    pub primary_color: String,
    pub secondary_color: String,
    pub logo_url: String,
    pub favicon_url: String,
    pub app_name: String,
}

/// White label service
pub struct WhiteLabelService {
    /// White labels
    white_labels: HashMap<String, WhiteLabelConfig>,
    /// Active API keys
    active_keys: HashMap<String, String>,
    /// Fee collection
    fee_collection: u64,
    /// Encryption
    encryption_key: [u8; 32],
    rng: SystemRandom,
}

impl WhiteLabelService {
    /// Create new service
    pub fn new() -> Self {
        let mut key = [0u8; 32];
        let rng = SystemRandom::new();
        rng.fill(&mut key).unwrap();
        
        Self {
            white_labels: HashMap::new(),
            active_keys: HashMap::new(),
            fee_collection: 0,
            encryption_key: key,
            rng,
        }
    }
    
    /// Create white label (called by TigerWallet admin)
    pub fn create_white_label(
        &mut self,
        name: &str,
        domain: &str,
        api_key: &str,
        admin_id: &str,
    ) -> Result<WhiteLabelConfig, WLError> {
        // Validate domain not taken
        for wl in self.white_labels.values() {
            if wl.domain == domain {
                return Err(WLError::DomainTaken);
            }
        }
        
        // Hash API key
        let api_key_hash = self.hash_api_key(api_key)?;
        
        let config = WhiteLabelConfig {
            id: self.generate_id(),
            name: name.to_string(),
            domain: domain.to_string(),
            api_key_hash,
            fee_percent: 20.0, // Default 20%
            status: WhiteLabelStatus::Pending,
            created_at: chrono::Utc::now().timestamp(),
            approved_at: None,
            tiger_admin_id: admin_id.to_string(),
            features: vec!["*".to_string()],
            custom_branding: true,
            branding: BrandingConfig {
                primary_color: "#FF6B35".to_string(),
                secondary_color: "#1A1A2E".to_string(),
                logo_url: "".to_string(),
                favicon_url: "".to_string(),
                app_name: name.to_string(),
            },
        };
        
        self.white_labels.insert(config.id.clone(), config.clone());
        
        Ok(config)
    }
    
    /// Approve white label (called by TigerWallet admin)
    pub fn approve_white_label(
        &mut self,
        wl_id: &str,
        admin_id: &str,
    ) -> Result<WhiteLabelConfig, WLError> {
        let wl = self.white_labels.get_mut(wl_id)
            .ok_or(WLError::WhiteLabelNotFound)?;
        
        if wl.status != WhiteLabelStatus::Pending {
            return Err(WLError::AlreadyApproved);
        }
        
        wl.status = WhiteLabelStatus::Active;
        wl.approved_at = Some(chrono::Utc::now().timestamp());
        
        // Add to active keys
        self.active_keys.insert(wl.api_key_hash.clone(), wl_id.to_string());
        
        Ok(wl.clone())
    }
    
    /// Validate API key
    pub fn validate_api_key(&self, api_key: &str) -> Result<WhiteLabelConfig, WLError> {
        let hash = self.hash_api_key(api_key)?;
        
        for wl in self.white_labels.values() {
            if wl.api_key_hash == hash {
                if wl.status == WhiteLabelStatus::Active {
                    return Ok(wl.clone());
                } else if wl.status == WhiteLabelStatus::Pending {
                    return Err(WLError::NotApproved);
                } else {
                    return Err(WLError::Revoked);
                }
            }
        }
        
        Err(WLError::InvalidAPIKey)
    }
    
    /// Suspend white label
    pub fn suspend_white_label(
        &mut self,
        wl_id: &str,
    ) -> Result<(), WLError> {
        let wl = self.white_labels.get_mut(wl_id)
            .ok_or(WLError::WhiteLabelNotFound)?;
        
        wl.status = WhiteLabelStatus::Suspended;
        
        // Remove from active keys
        self.active_keys.remove(&wl.api_key_hash);
        
        Ok(())
    }
    
    /// Revoke white label
    pub fn revoke_white_label(
        &mut self,
        wl_id: &str,
    ) -> Result<(), WLError> {
        let wl = self.white_labels.get_mut(wl_id)
            .ok_or(WLError::WhiteLabelNotFound)?;
        
        wl.status = WhiteLabelStatus::Revoked;
        
        // Remove from active keys
        self.active_keys.remove(&wl.api_key_hash);
        
        Ok(())
    }
    
    /// Destroy white label completely
    pub fn destroy_white_label(
        &mut self,
        wl_id: &str,
    ) -> Result<(), WLError> {
        let wl = self.white_labels.get(wl_id)
            .ok_or(WLError::WhiteLabelNotFound)?;
        
        // Remove from active keys
        self.active_keys.remove(&wl.api_key_hash);
        
        // Remove white label
        self.white_labels.remove(wl_id);
        
        Ok(())
    }
    
    /// Update fee percentage
    pub fn update_fee(
        &mut self,
        wl_id: &str,
        fee_percent: f64,
    ) -> Result<(), WLError> {
        if fee_percent > 20.0 {
            return Err(WLError::FeeTooHigh);
        }
        
        let wl = self.white_labels.get_mut(wl_id)
            .ok_or(WLError::WhiteLabelNotFound)?;
        
        wl.fee_percent = fee_percent;
        
        Ok(())
    }
    
    /// Collect fee from white label
    pub fn collect_fee(&mut self, wl_id: &str, amount: u64) -> Result<u64, WLError> {
        let wl = self.white_labels.get(wl_id)
            .ok_or(WLError::WhiteLabelNotFound)?;
        
        let fee = (amount as f64 * wl.fee_percent / 100.0) as u64;
        
        self.fee_collection += fee;
        
        Ok(fee)
    }
    
    /// Get white label by ID
    pub fn get_white_label(&self, wl_id: &str) -> Option<WhiteLabelConfig> {
        self.white_labels.get(wl_id).cloned()
    }
    
    /// Get all white labels
    pub fn get_all_white_labels(&self) -> Vec<WhiteLabelConfig> {
        self.white_labels.values().cloned().collect()
    }
    
    /// Get total fee collected
    pub fn get_total_fees(&self) -> u64 {
        self.fee_collection
    }
    
    /// Hash API key
    fn hash_api_key(&self, api_key: &str) -> Result<String, WLError> {
        let hash = digest(&ring::digest::SHA256, api_key.as_bytes());
        Ok(hex::encode(hash.as_ref()))
    }
    
    /// Generate ID
    fn generate_id(&self) -> String {
        let timestamp = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        
        let hash = digest(&ring::digest::SHA256, &timestamp.to_be_bytes());
        hex::encode(&hash.as_ref()[..16])
    }
}

/// One-time nonce
struct WLOneNonce {
    nonce: Option<Nonce>,
}

impl WLOneNonce {
    fn new(nonce: Nonce) -> Self {
        Self { nonce: Some(nonce) }
    }
}

impl NonceSequence for WLOneNonce {
    fn advance(&mut self) -> Result<Nonce, ring::error::Unspecified> {
        self.nonce.take().ok_or(ring::error::Unspecified)
    }
}

/// White label errors
#[derive(Debug, Error)]
pub enum WLError {
    #[error("White label not found")]
    WhiteLabelNotFound,
    
    #[error("Domain already taken")]
    DomainTaken,
    
    #[error("Invalid API key")]
    InvalidAPIKey,
    
    #[error("White label not approved")]
    NotApproved,
    
    #[error("White label already approved")]
    AlreadyApproved,
    
    #[error("White label revoked")]
    Revoked,
    
    #[error("Fee too high (max 20%)")]
    FeeTooHigh,
    
    #[error("Hash error")]
    HashError,
}