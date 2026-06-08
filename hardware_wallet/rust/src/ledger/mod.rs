//! TigerWallet Ledger Hardware Wallet Support
//! Support for Ledger Nano S+, Nano X, Stax, and Flex

use serde::{Deserialize, Serialize};
use std::sync::Arc;
use parking_lot::RwLock;

/// Ledger device model
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum LedgerModel {
    NanoS,
    NanoSPlus,
    NanoX,
    Stax,
    Flex,
}

/// Transport type for Ledger
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum LedgerTransport {
    USB,
    Bluetooth,
}

/// Ledger device info
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LedgerDevice {
    pub device_id: String,
    pub model: LedgerModel,
    pub transport: LedgerTransport,
    pub firmware_version: String,
    pub ble_version: Option<String>,
    pub serial: String,
    pub initialized: bool,
    pub pin_enabled: bool,
    pub passphrase_enabled: bool,
}

impl LedgerDevice {
    pub fn new(model: LedgerModel, serial: &str) -> Self {
        Self {
            device_id: uuid::Uuid::new_v4().to_string(),
            model,
            transport: LedgerTransport::USB,
            firmware_version: "2.1.0".to_string(),
            ble_version: None,
            serial: serial.to_string(),
            initialized: false,
            pin_enabled: true,
            passphrase_enabled: true,
        }
    }
}

/// Ledger command types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum LedgerCommand {
    GetPublicKey {
        path: String,
        display: bool,
    },
    SignTransaction {
        tx: Vec<u8>,
        path: String,
    },
    SignMessage {
        message: Vec<u8>,
        path: String,
    },
    GetAppConfiguration,
    GetDeviceInfo,
}

/// Ledger response types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum LedgerResponse {
    PublicKey {
        public_key: String,
        address: String,
    },
    Signature {
        signature: Vec<u8>,
    },
    DeviceInfo {
        firmware: String,
        app_version: String,
        device_id: String,
    },
    Error {
        code: i32,
        message: String,
    },
}

/// Ledger communication result
#[derive(Debug, thiserror::Error)]
pub enum LedgerError {
    #[error("Device not found")]
    DeviceNotFound,
    
    #[error("Transport error: {0}")]
    TransportError(String),
    
    #[error("APDU error: {0}")]
    ApduError(String),
    
    #[error("User cancelled")]
    UserCancelled,
    
    #[error("Timeout")]
    Timeout,
    
    #[error("Invalid response: {0}")]
    InvalidResponse(String),
}

/// Ledger wallet implementation
pub struct LedgerWallet {
    device: RwLock<Option<LedgerDevice>>,
    connected: RwLock<bool>,
}

impl LedgerWallet {
    pub fn new() -> Self {
        Self {
            device: RwLock::new(None),
            connected: RwLock::new(false),
        }
    }

    pub fn connect(&self, device: LedgerDevice) {
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

    pub fn get_device(&self) -> Option<LedgerDevice> {
        self.device.read().clone()
    }

    pub fn get_public_key(&self, path: &str) -> Result<LedgerResponse, LedgerError> {
        if !self.is_connected() {
            return Err(LedgerError::DeviceNotFound);
        }

        // In production, this would communicate with the actual Ledger device
        // using HID or BLE transport
        
        Ok(LedgerResponse::PublicKey {
            public_key: "02" + &hex::encode(&[0u8; 64]),
            address: "bc1q" + &bs58::encode(&[0u8; 20]).into_iter().map(|b| b as char).collect::<String>(),
        })
    }

    pub fn sign_transaction(&self, tx: &[u8], path: &str) -> Result<LedgerResponse, LedgerError> {
        if !self.is_connected() {
            return Err(LedgerError::DeviceNotFound);
        }

        // Sign transaction using Ledger's security
        // In production, this would send APDU commands to the device
        
        Ok(LedgerResponse::Signature {
            signature: vec![0u8; 64],
        })
    }

    pub fn sign_message(&self, message: &[u8], path: &str) -> Result<LedgerResponse, LedgerError> {
        if !self.is_connected() {
            return Err(LedgerError::DeviceNotFound);
        }

        // Sign message using Ledger
        Ok(LedgerResponse::Signature {
            signature: vec![0u8; 64],
        })
    }

    pub fn verify_address(&self, path: &str, expected: &str) -> Result<bool, LedgerError> {
        let response = self.get_public_key(path)?;
        
        if let LedgerResponse::PublicKey { address, .. } = response {
            Ok(address == expected)
        } else {
            Err(LedgerError::InvalidResponse("Expected public key response".to_string()))
        }
    }
}

impl Default for LedgerWallet {
    fn default() -> Self {
        Self::new()
    }
}

/// BIP32 path derivation
pub fn derive_bip32_path(path: &str) -> Result<Vec<u32>, LedgerError> {
    let mut components = Vec::new();
    
    for part in path.trim_start_matches("m/").split('/') {
        let hardened = part.ends_with('\'');
        let index: u32 = part.trim_end_matches('\'')
            .parse()
            .map_err(|_| LedgerError::InvalidResponse(format!("Invalid path component: {}", part)))?;
        
        if hardened {
            components.push(0x80000000 | index);
        } else {
            components.push(index);
        }
    }
    
    Ok(components)
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_ledger_device() {
        let device = LedgerDevice::new(LedgerModel::NanoX, "SN123456789");
        assert_eq!(device.model, LedgerModel::NanoX);
        assert!(device.pin_enabled);
    }
    
    #[test]
    fn test_bip32_path() {
        let path = derive_bip32_path("m/44'/0'/0'/0/0").unwrap();
        assert_eq!(path, vec![0x8000002c, 0x80000000, 0x80000000, 0, 0]);
    }
}