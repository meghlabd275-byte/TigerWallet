//! TigerWallet Ellipal Hardware Wallet Support
//! Support for Ellipal Titan and Ellipal Mini

use serde::{Deserialize, Serialize};
use parking_lot::RwLock;

/// Ellipal device model
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum EllipalModel {
    Titan,
    TitanPro,
    Mini,
    Mini2,
}

/// Ellipal device info
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EllipalDevice {
    pub device_id: String,
    pub model: EllipalModel,
    pub firmware_version: String,
    pub ble_version: Option<String>,
    pub serial: String,
    pub screen_size: (u32, u32),
    pub battery_level: u8,
    pub initialized: bool,
}

impl EllipalDevice {
    pub fn new(model: EllipalModel, serial: &str) -> Self {
        let screen_size = match model {
            EllipalModel::Titan | EllipalModel::TitanPro => (600, 800),
            EllipalModel::Mini | EllipalModel::Mini2 => (320, 480),
        };
        
        Self {
            device_id: uuid::Uuid::new_v4().to_string(),
            model,
            firmware_version: "2.5.0".to_string(),
            ble_version: Some("1.2.0".to_string()),
            serial: serial.to_string(),
            screen_size,
            battery_level: 100,
            initialized: false,
        }
    }
}

/// Ellipal wallet implementation
pub struct EllipalWallet {
    device: RwLock<Option<EllipalDevice>>,
    connected: RwLock<bool>,
}

impl EllipalWallet {
    pub fn new() -> Self {
        Self {
            device: RwLock::new(None),
            connected: RwLock::new(false),
        }
    }

    pub fn connect(&self, device: EllipalDevice) {
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

    pub fn get_public_key(&self, path: &str) -> Result<EllipalResponse, EllipalError> {
        if !self.is_connected() {
            return Err(EllipalError::DeviceNotFound);
        }

        Ok(EllipalResponse::PublicKey {
            public_key: hex::encode(&[0u8; 33]),
            address: "0x" + &hex::encode(&[0u8; 20]),
        })
    }

    pub fn sign_transaction(&self, tx: &[u8]) -> Result<EllipalResponse, EllipalError> {
        if !self.is_connected() {
            return Err(EllipalError::DeviceNotFound);
        }

        Ok(EllipalResponse::Signature {
            signature: hex::encode(tx),
        })
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EllipalResponse {
    PublicKey {
        public_key: String,
        address: String,
    },
    Signature {
        signature: String,
    },
}

#[derive(Debug, thiserror::Error)]
pub enum EllipalError {
    #[error("Device not found")]
    DeviceNotFound,
    
    #[error("BLE connection error: {0}")]
    BLEError(String),
    
    #[error("Signing error: {0}")]
    SigningError(String),
}

impl Default for EllipalWallet {
    fn default() -> Self {
        Self::new()
    }
}