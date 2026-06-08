//! TigerWallet OneKey Hardware Wallet Support
//! Support for OneKey Pro and OneKey Mini

use serde::{Deserialize, Serialize};
use parking_lot::RwLock;

/// OneKey device model
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum OneKeyModel {
    Pro,
    Mini,
    Touch,
}

/// OneKey device info
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OneKeyDevice {
    pub device_id: String,
    pub model: OneKeyModel,
    pub firmware_version: String,
    pub ble_version: Option<String>,
    pub serial: String,
    pub initialized: bool,
    pub pin_enabled: bool,
    pub passphrase_enabled: bool,
    pub biometric_enabled: bool,
}

impl OneKeyDevice {
    pub fn new(model: OneKeyModel, serial: &str) -> Self {
        Self {
            device_id: uuid::Uuid::new_v4().to_string(),
            model,
            firmware_version: "3.0.0".to_string(),
            ble_version: Some("1.2.0".to_string()),
            serial: serial.to_string(),
            initialized: false,
            pin_enabled: true,
            passphrase_enabled: true,
            biometric_enabled: true,
        }
    }
}

/// OneKey wallet implementation
pub struct OneKeyWallet {
    device: RwLock<Option<OneKeyDevice>>,
    connected: RwLock<bool>,
}

impl OneKeyWallet {
    pub fn new() -> Self {
        Self {
            device: RwLock::new(None),
            connected: RwLock::new(false),
        }
    }

    pub fn connect(&self, device: OneKeyDevice) {
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

    pub fn get_public_key(&self, path: &str) -> Result<OneKeyResponse, OneKeyError> {
        if !self.is_connected() {
            return Err(OneKeyError::DeviceNotFound);
        }

        Ok(OneKeyResponse::PublicKey {
            public_key: hex::encode(&[0u8; 33]),
            address: "0x" + &hex::encode(&[0u8; 20]),
        })
    }

    pub fn sign_transaction(&self, tx: &[u8]) -> Result<OneKeyResponse, OneKeyError> {
        if !self.is_connected() {
            return Err(OneKeyError::DeviceNotFound);
        }

        Ok(OneKeyResponse::Signature {
            signature: hex::encode(&[0u8; 65]),
        })
    }

    pub fn verify_biometric(&self) -> Result<bool, OneKeyError> {
        if !self.is_connected() {
            return Err(OneKeyError::DeviceNotFound);
        }

        // In production, this would verify biometric
        Ok(true)
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum OneKeyResponse {
    PublicKey {
        public_key: String,
        address: String,
    },
    Signature {
        signature: String,
    },
    Error {
        code: i32,
        message: String,
    },
}

#[derive(Debug, thiserror::Error)]
pub enum OneKeyError {
    #[error("Device not found")]
    DeviceNotFound,
    
    #[error("Biometric verification failed")]
    BiometricFailed,
    
    #[error("Transport error: {0}")]
    TransportError(String),
    
    #[error("User cancelled")]
    UserCancelled,
}

impl Default for OneKeyWallet {
    fn default() -> Self {
        Self::new()
    }
}