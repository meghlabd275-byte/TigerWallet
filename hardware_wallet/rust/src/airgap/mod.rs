//! TigerWallet AirGap Hardware Wallet Support
//! Support for AirGap Vault and AirGap Wallet

use serde::{Deserialize, Serialize};
use parking_lot::RwLock;

/// AirGap device model
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum AirGapModel {
    Vault,
    Wallet,
    WalletPro,
}

/// AirGap device info
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AirGapDevice {
    pub device_id: String,
    pub model: AirGapModel,
    pub firmware_version: String,
    pub serial: String,
    pub initialized: bool,
    pub security_level: SecurityLevel,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum SecurityLevel {
    Standard,
    Enhanced,
    Maximum,
}

impl AirGapDevice {
    pub fn new(model: AirGapModel, serial: &str) -> Self {
        Self {
            device_id: uuid::Uuid::new_v4().to_string(),
            model,
            firmware_version: "3.0.0".to_string(),
            serial: serial.to_string(),
            initialized: false,
            security_level: SecurityLevel::Maximum,
        }
    }
}

/// AirGap wallet implementation using QR code communication
pub struct AirGapWallet {
    device: RwLock<Option<AirGapDevice>>,
    connected: RwLock<bool>,
}

impl AirGapWallet {
    pub fn new() -> Self {
        Self {
            device: RwLock::new(None),
            connected: RwLock::new(false),
        }
    }

    pub fn connect(&self, device: AirGapDevice) {
        *self.device.write() = Some(device);
        *self.connected.write() = true;
    }

    pub fn disconnect(&self) {
        *self.device.write() = None;
        *self.connected.write() = false;
    }

    pub fn is_connected(&self) -> bool {
        *self.connected.read()
    }

    /// Generate QR code data for signing request
    pub fn generate_sign_request(&self, tx_hash: &[u8]) -> Result<String, AirGapError> {
        if !self.is_connected() {
            return Err(AirGapError::DeviceNotFound);
        }

        // Encode transaction for QR display
        let request = SignRequest {
            tx_hash: hex::encode(tx_hash),
            timestamp: chrono::Utc::now().timestamp(),
            nonce: rand::random::<u32>(),
        };

        Ok(serde_json::to_string(&request).unwrap_or_default())
    }

    /// Parse signature from QR code response
    pub fn parse_signature(&self, qr_data: &str) -> Result<AirGapSignature, AirGapError> {
        serde_json::from_str(qr_data)
            .map_err(|e| AirGapError::InvalidResponse(e.to_string()))
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignRequest {
    pub tx_hash: String,
    pub timestamp: i64,
    pub nonce: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AirGapSignature {
    pub signature: String,
    pub public_key: String,
}

#[derive(Debug, thiserror::Error)]
pub enum AirGapError {
    #[error("Device not found")]
    DeviceNotFound,
    
    #[error("Invalid response: {0}")]
    InvalidResponse(String),
    
    #[error("Security error: {0}")]
    SecurityError(String),
}

impl Default for AirGapWallet {
    fn default() -> Self {
        Self::new()
    }
}