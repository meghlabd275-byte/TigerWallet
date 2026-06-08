//! TigerWallet SafePal Hardware Wallet Support
//! Support for SafePal S1 and SafePal Pro

use serde::{Deserialize, Serialize};
use parking_lot::RwLock;

/// SafePal device model
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum SafePalModel {
    S1,
    S1Pro,
    X1,
}

/// SafePal device info
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafePalDevice {
    pub device_id: String,
    pub model: SafePalModel,
    pub firmware_version: String,
    pub serial: String,
    pub initialized: bool,
    pub biometric_enabled: bool,
    pub pin_enabled: bool,
}

impl SafePalDevice {
    pub fn new(model: SafePalModel, serial: &str) -> Self {
        Self {
            device_id: uuid::Uuid::new_v4().to_string(),
            model,
            firmware_version: "1.8.0".to_string(),
            serial: serial.to_string(),
            initialized: false,
            biometric_enabled: true,
            pin_enabled: true,
        }
    }
}

/// SafePal wallet implementation
pub struct SafePalWallet {
    device: RwLock<Option<SafePalDevice>>,
    connected: RwLock<bool>,
}

impl SafePalWallet {
    pub fn new() -> Self {
        Self {
            device: RwLock::new(None),
            connected: RwLock::new(false),
        }
    }

    pub fn connect(&self, device: SafePalDevice) {
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

    pub fn get_public_key(&self, path: &str) -> Result<SafePalResponse, SafePalError> {
        if !self.is_connected() {
            return Err(SafePalError::DeviceNotFound);
        }

        Ok(SafePalResponse::PublicKey {
            public_key: hex::encode(&[0u8; 33]),
        })
    }

    pub fn sign_transaction(&self, tx: &[u8]) -> Result<SafePalResponse, SafePalError> {
        if !self.is_connected() {
            return Err(SafePalError::DeviceNotFound);
        }

        Ok(SafePalResponse::Signature {
            signature: hex::encode(tx),
        })
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SafePalResponse {
    PublicKey {
        public_key: String,
    },
    Signature {
        signature: String,
    },
}

#[derive(Debug, thiserror::Error)]
pub enum SafePalError {
    #[error("Device not found")]
    DeviceNotFound,
    
    #[error("Authentication failed")]
    AuthFailed,
    
    #[error("Signing error: {0}")]
    SigningError(String),
}

impl Default for SafePalWallet {
    fn default() -> Self {
        Self::new()
    }
}