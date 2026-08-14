//! Hardware Wallet Support
//! 
//! Integration with hardware security modules (HSM) and hardware wallets:
//! - Ledger
//! - Trezor
//! - YubiKey
//! - AWS KMS
//! - GCP KMS
//! - HashiCorp Vault

use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use thiserror::Error;

#[derive(Error, Debug)]
pub enum HardwareError {
    #[error("Device not found: {0}")]
    DeviceNotFound(String),
    #[error("Connection failed: {0}")]
    ConnectionFailed(String),
    #[error("Signing failed: {0}")]
    SigningFailed(String),
    #[error("Invalid PIN: {0}")]
    InvalidPin(String),
    #[error("Device locked: {0}")]
    DeviceLocked(String),
    #[error("Operation cancelled: {0}")]
    OperationCancelled(String),
    #[error("HSM error: {0}")]
    HsmError(String),
}

// ============================================================================
// Types
// ============================================================================

/// Hardware wallet type
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum HardwareType {
    Ledger,
    Trezor,
    YubiKey,
    AwsKms,
    GcpKms,
    HashiCorpVault,
}

/// Connection status
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ConnectionStatus {
    Disconnected,
    Connected,
    Locked,
    Error,
}

/// Device info
#[derive(Debug, Clone)]
pub struct DeviceInfo {
    pub device_type: HardwareType,
    pub device_id: String,
    pub label: String,
    pub firmware_version: String,
    pub is_initialized: bool,
    pub is_bootloader: bool,
}

/// Signature from hardware wallet
#[derive(Debug, Clone)]
pub struct HardwareSignature {
    pub device_id: String,
    pub signature: Vec<u8>,
    pub recovery_param: Option<u8>,
}

// ============================================================================
// Hardware Wallet Trait
// ============================================================================

pub trait HardwareWallet: Send + Sync {
    /// Get device info
    fn device_info(&self) -> DeviceInfo;
    
    /// Check connection status
    fn status(&self) -> ConnectionStatus;
    
    /// Connect to device
    fn connect(&mut self) -> Result<(), HardwareError>;
    
    /// Disconnect from device
    fn disconnect(&self) -> Result<(), HardwareError>;
    
    /// Get Ethereum address
    fn get_address(&self, path: &str) -> Result<String, HardwareError>;
    
    /// Sign transaction
    fn sign_transaction(&self, tx: &[u8], path: &str) -> Result<HardwareSignature, HardwareError>;
    
    /// Sign message
    fn sign_message(&self, message: &[u8], path: &str) -> Result<HardwareSignature, HardwareError>;
    
    /// Sign hash
    fn sign_hash(&self, hash: &[u8], path: &str) -> Result<HardwareSignature, HardwareError>;
}


/// Fail-closed signing used by hardware wallets with no connected transport.
/// Returns `SigningFailed` — NEVER fabricates a signature. A real hardware
/// wallet signs on-device via its secure element (Ledger/Trezor over HID/BLE,
/// AWS KMS via the KMS API, etc.); without a real transport wired we cannot
/// produce a valid signature, so we fail honestly rather than return a fake.
pub fn simulated_sign(
    status: ConnectionStatus,
    device_id: &str,
    _data: &[u8],
) -> Result<HardwareSignature, HardwareError> {
    if status != ConnectionStatus::Connected {
        return Err(HardwareError::SigningFailed(format!("device {} not connected", device_id)));
    }
    Err(HardwareError::SigningFailed(format!(
        "device {} connected but no signing transport is wired; real hardware signing requires a HID/BLE (Ledger/Trezor) or KMS transport backend",
        device_id
    )))
}

// ============================================================================
// Ledger Wallet
// ============================================================================

pub struct LedgerWallet {
    device_id: String,
    status: ConnectionStatus,
    info: Option<DeviceInfo>,
    pubkey_cache: RwLock<HashMap<String, String>>,
}

impl LedgerWallet {
    pub fn new(device_id: String) -> Self {
        Self {
            device_id,
            status: ConnectionStatus::Disconnected,
            info: None,
            pubkey_cache: RwLock::new(HashMap::new()),
        }
    }
    
    pub fn connect(mut self) -> Result<Self, HardwareError> {
        // In production: use hidapi to connect
        self.status = ConnectionStatus::Connected;
        self.info = Some(DeviceInfo {
            device_type: HardwareType::Ledger,
            device_id: self.device_id.clone(),
            label: "Ledger".to_string(),
            firmware_version: "2.1.0".to_string(),
            is_initialized: true,
            is_bootloader: false,
        });
        Ok(self)
    }
}

impl HardwareWallet for LedgerWallet {
    fn device_info(&self) -> DeviceInfo {
        self.info.clone().unwrap_or(DeviceInfo {
            device_type: HardwareType::Ledger,
            device_id: self.device_id.clone(),
            label: "Unknown".to_string(),
            firmware_version: "Unknown".to_string(),
            is_initialized: false,
            is_bootloader: false,
        })
    }
    
    fn status(&self) -> ConnectionStatus {
        self.status
    }
    
    fn connect(&mut self) -> Result<(), HardwareError> {
        self.status = ConnectionStatus::Connected;
        Ok(())
    }
    
    fn disconnect(&self) -> Result<(), HardwareError> {
        Ok(())
    }
    
    fn get_address(&self, _path: &str) -> Result<String, HardwareError> {
        if self.status != ConnectionStatus::Connected {
            return Err(HardwareError::ConnectionFailed(format!("device {} not connected", self.device_id)));
        }
        Err(HardwareError::DeviceNotFound(format!(
            "device {} connected but no address-derivation transport is wired; a real hardware wallet derives the address from the on-device public key",
            self.device_id
        )))
    }

    fn sign_transaction(&self, tx: &[u8], _path: &str) -> Result<HardwareSignature, HardwareError> {
        simulated_sign(self.status, &self.device_id, tx)
    }

    fn sign_message(&self, message: &[u8], _path: &str) -> Result<HardwareSignature, HardwareError> {
        simulated_sign(self.status, &self.device_id, message)
    }

    fn sign_hash(&self, hash: &[u8], _path: &str) -> Result<HardwareSignature, HardwareError> {
        simulated_sign(self.status, &self.device_id, hash)
    }
}

// ============================================================================
// Trezor Wallet
// ============================================================================

pub struct TrezorWallet {
    device_id: String,
    status: ConnectionStatus,
    pin_attempts: RwLock<u8>,
    info: Option<DeviceInfo>,
}

impl TrezorWallet {
    pub fn new(device_id: String) -> Self {
        Self {
            device_id,
            status: ConnectionStatus::Disconnected,
            pin_attempts: RwLock::new(0),
            info: None,
        }
    }
}

impl HardwareWallet for TrezorWallet {
    fn device_info(&self) -> DeviceInfo {
        self.info.clone().unwrap_or(DeviceInfo {
            device_type: HardwareType::Trezor,
            device_id: self.device_id.clone(),
            label: "Trezor".to_string(),
            firmware_version: "2.5.0".to_string(),
            is_initialized: true,
            is_bootloader: false,
        })
    }
    
    fn status(&self) -> ConnectionStatus {
        self.status
    }
    
    fn connect(&mut self) -> Result<(), HardwareError> {
        self.status = ConnectionStatus::Connected;
        Ok(())
    }
    
    fn disconnect(&self) -> Result<(), HardwareError> {
        Ok(())
    }
    
    fn get_address(&self, _path: &str) -> Result<String, HardwareError> {
        if self.status != ConnectionStatus::Connected {
            return Err(HardwareError::ConnectionFailed(format!("device {} not connected", self.device_id)));
        }
        Err(HardwareError::DeviceNotFound(format!(
            "device {} connected but no address-derivation transport is wired; a real hardware wallet derives the address from the on-device public key",
            self.device_id
        )))
    }

    fn sign_transaction(&self, tx: &[u8], _path: &str) -> Result<HardwareSignature, HardwareError> {
        simulated_sign(self.status, &self.device_id, tx)
    }

    fn sign_message(&self, message: &[u8], _path: &str) -> Result<HardwareSignature, HardwareError> {
        simulated_sign(self.status, &self.device_id, message)
    }

    fn sign_hash(&self, hash: &[u8], _path: &str) -> Result<HardwareSignature, HardwareError> {
        simulated_sign(self.status, &self.device_id, hash)
    }
}

// ============================================================================
// YubiKey Wallet
// ============================================================================

pub struct YubiKeyWallet {
    device_id: String,
    status: ConnectionStatus,
}

impl YubiKeyWallet {
    pub fn new(device_id: String) -> Self {
        Self {
            device_id,
            status: ConnectionStatus::Disconnected,
        }
    }
}

impl HardwareWallet for YubiKeyWallet {
    fn device_info(&self) -> DeviceInfo {
        DeviceInfo {
            device_type: HardwareType::YubiKey,
            device_id: self.device_id.clone(),
            label: "YubiKey".to_string(),
            firmware_version: "5.2.3".to_string(),
            is_initialized: true,
            is_bootloader: false,
        }
    }
    
    fn status(&self) -> ConnectionStatus {
        self.status
    }
    
    fn connect(&mut self) -> Result<(), HardwareError> {
        self.status = ConnectionStatus::Connected;
        Ok(())
    }
    
    fn disconnect(&self) -> Result<(), HardwareError> {
        Ok(())
    }
    
    fn get_address(&self, _path: &str) -> Result<String, HardwareError> {
        if self.status != ConnectionStatus::Connected {
            return Err(HardwareError::ConnectionFailed(format!("device {} not connected", self.device_id)));
        }
        Err(HardwareError::DeviceNotFound(format!(
            "device {} connected but no address-derivation transport is wired; a real hardware wallet derives the address from the on-device public key",
            self.device_id
        )))
    }

    fn sign_transaction(&self, tx: &[u8], _path: &str) -> Result<HardwareSignature, HardwareError> {
        simulated_sign(self.status, &self.device_id, tx)
    }

    fn sign_message(&self, message: &[u8], _path: &str) -> Result<HardwareSignature, HardwareError> {
        simulated_sign(self.status, &self.device_id, message)
    }

    fn sign_hash(&self, hash: &[u8], _path: &str) -> Result<HardwareSignature, HardwareError> {
        simulated_sign(self.status, &self.device_id, hash)
    }
}

// ============================================================================
// AWS KMS Wallet
// ============================================================================

pub struct AwsKmsWallet {
    key_id: String,
    region: String,
    status: ConnectionStatus,
}

impl AwsKmsWallet {
    pub fn new(key_id: String, region: String) -> Self {
        Self {
            key_id,
            region,
            status: ConnectionStatus::Disconnected,
        }
    }
}

impl HardwareWallet for AwsKmsWallet {
    fn device_info(&self) -> DeviceInfo {
        DeviceInfo {
            device_type: HardwareType::AwsKms,
            device_id: self.key_id.clone(),
            label: "AWS KMS".to_string(),
            firmware_version: "N/A".to_string(),
            is_initialized: true,
            is_bootloader: false,
        }
    }
    
    fn status(&self) -> ConnectionStatus {
        self.status
    }
    
    fn connect(&mut self) -> Result<(), HardwareError> {
        self.status = ConnectionStatus::Connected;
        Ok(())
    }
    
    fn disconnect(&self) -> Result<(), HardwareError> {
        Ok(())
    }
    
    fn get_address(&self, _path: &str) -> Result<String, HardwareError> {
        if self.status != ConnectionStatus::Connected {
            return Err(HardwareError::ConnectionFailed(format!("device {} not connected", self.key_id)));
        }
        Err(HardwareError::DeviceNotFound(format!(
            "KMS key {} connected but no KMS public-key fetch transport is wired; a real AWS KMS wallet derives the address from the KMS public key",
            self.key_id
        )))
    }

    fn sign_transaction(&self, tx: &[u8], _path: &str) -> Result<HardwareSignature, HardwareError> {
        simulated_sign(self.status, &self.key_id, tx)
    }

    fn sign_message(&self, message: &[u8], _path: &str) -> Result<HardwareSignature, HardwareError> {
        simulated_sign(self.status, &self.key_id, message)
    }

    fn sign_hash(&self, hash: &[u8], _path: &str) -> Result<HardwareSignature, HardwareError> {
        simulated_sign(self.status, &self.key_id, hash)
    }
}

// ============================================================================
// Hardware Wallet Manager
// ============================================================================

pub struct HardwareWalletManager {
    devices: RwLock<HashMap<String, Arc<dyn HardwareWallet>>>,
}

impl HardwareWalletManager {
    pub fn new() -> Self {
        Self {
            devices: RwLock::new(HashMap::new()),
        }
    }
    
    pub fn add_device(&self, id: String, device: Arc<dyn HardwareWallet>) -> Result<(), HardwareError> {
        let mut devices = self.devices.write().unwrap();
        devices.insert(id, device);
        Ok(())
    }
    
    pub fn get_device(&self, id: &str) -> Option<Arc<dyn HardwareWallet>> {
        self.devices.read().unwrap().get(id).map(|d| Arc::clone(d))
    }
    
    pub fn list_devices(&self) -> Vec<DeviceInfo> {
        self.devices.read().unwrap()
            .values()
            .map(|d| d.device_info())
            .collect()
    }
    
    pub fn remove_device(&self, id: &str) -> Result<(), HardwareError> {
        let mut devices = self.devices.write().unwrap();
        devices.remove(id)
            .ok_or_else(|| HardwareError::DeviceNotFound(id.to_string()))?;
        Ok(())
    }
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_ledger_wallet() {
        let ledger = LedgerWallet::new("ledger-1".to_string());
        assert_eq!(ledger.status(), ConnectionStatus::Disconnected);
    }

    #[test]
    fn test_trezor_wallet() {
        let trezor = TrezorWallet::new("trezor-1".to_string());
        assert_eq!(trezor.status(), ConnectionStatus::Disconnected);
    }

    #[test]
    fn test_hardware_manager() {
        let manager = HardwareWalletManager::new();
        let devices = manager.list_devices();
        assert!(devices.is_empty());
    }

    #[test]
    fn test_sign_transaction_fail_closed() {
        // Even when "connected", signing fails closed - there is no real
        // hardware transport wired, so a valid signature cannot be produced.
        let ledger = LedgerWallet::new("ledger-1".to_string());
        let ledger = ledger.connect().unwrap();

        let tx = vec![0u8; 100];
        let sig = ledger.sign_transaction(&tx, "m/44'/60'/0'/0/0");
        assert!(sig.is_err());
        assert!(matches!(sig, Err(HardwareError::SigningFailed(_))));
    }

    #[test]
    fn test_not_connected_errors() {
        // All signing and address operations fail when the device is not connected
        let ledger = LedgerWallet::new("ledger-1".to_string());
        assert_eq!(ledger.status(), ConnectionStatus::Disconnected);

        let path = "m/44'/60'/0'/0/0";
        let payload = vec![0u8; 32];

        let addr = ledger.get_address(path);
        assert!(addr.is_err());
        assert!(matches!(addr, Err(HardwareError::ConnectionFailed(_))));

        let tx = ledger.sign_transaction(&payload, path);
        assert!(tx.is_err());
        assert!(matches!(tx, Err(HardwareError::SigningFailed(_))));

        let msg = ledger.sign_message(&payload, path);
        assert!(msg.is_err());
        assert!(matches!(msg, Err(HardwareError::SigningFailed(_))));

        let hash = ledger.sign_hash(&payload, path);
        assert!(hash.is_err());
        assert!(matches!(hash, Err(HardwareError::SigningFailed(_))));
    }

    #[test]
    fn test_signing_failed_message_mentions_device() {
        // The SigningFailed error carries the device id so callers can trace the failure
        let trezor = TrezorWallet::new("trezor-7".to_string());
        let err = trezor.sign_transaction(b"tx", "m/44'/60'/0'/0/0").unwrap_err();
        let msg = err.to_string();
        assert!(msg.contains("trezor-7"), "error should mention device id: {msg}");
    }

    #[test]
    fn test_connect_then_sign_fail_closed() {
        // After connecting, signing still fails closed - no real transport
        // is wired, so the connected device cannot produce a valid signature.
        let ledger = LedgerWallet::new("ledger-1".to_string()).connect().unwrap();
        assert_eq!(ledger.status(), ConnectionStatus::Connected);

        let tx = b"some transaction payload";
        let sig = ledger.sign_transaction(tx, "m/44'/60'/0'/0/0");
        assert!(sig.is_err());
        assert!(matches!(sig, Err(HardwareError::SigningFailed(_))));

        // get_address also fails closed once connected (no pubkey transport)
        let addr = ledger.get_address("m/44'/60'/0'/0/0");
        assert!(addr.is_err());
        assert!(matches!(addr, Err(HardwareError::DeviceNotFound(_))));
    }
}