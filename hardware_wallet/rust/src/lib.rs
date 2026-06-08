//! TigerWallet Hardware Wallet Infrastructure
//! Support for Ledger, Trezor, OneKey, AirGap, Ellipal, and SafePal

pub mod ledger;
pub mod trezor;
pub mod onekey;

pub use ledger::*;
pub use trezor::*;
pub use onekey::*;

/// Hardware wallet trait for unified interface
pub trait HardwareWallet {
    fn is_connected(&self) -> bool;
    fn get_public_key(&self, path: &str) -> Result<String, String>;
    fn sign_transaction(&self, tx: &[u8], path: &str) -> Result<Vec<u8>, String>;
    fn sign_message(&self, message: &[u8], path: &str) -> Result<Vec<u8>, String>;
}

/// Supported hardware wallet vendors
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum HardwareVendor {
    Ledger,
    Trezor,
    OneKey,
    AirGap,
    Ellipal,
    SafePal,
}

/// Unified hardware wallet manager
pub struct HardwareWalletManager {
    ledger: LedgerWallet,
    trezor: TrezorWallet,
    onekey: OneKeyWallet,
}

impl HardwareWalletManager {
    pub fn new() -> Self {
        Self {
            ledger: LedgerWallet::new(),
            trezor: TrezorWallet::new(),
            onekey: OneKeyWallet::new(),
        }
    }

    pub fn connect_ledger(&self, device: LedgerDevice) {
        self.ledger.connect(device);
    }

    pub fn connect_trezor(&self, device: TrezorDevice) {
        self.trezor.connect(device);
    }

    pub fn connect_onekey(&self, device: OneKeyDevice) {
        self.onekey.connect(device);
    }

    pub fn disconnect(&self, vendor: HardwareVendor) {
        match vendor {
            HardwareVendor::Ledger => self.ledger.disconnect(),
            HardwareVendor::Trezor => self.trezor.disconnect(),
            HardwareVendor::OneKey => self.onekey.disconnect(),
            _ => {}
        }
    }

    pub fn is_connected(&self, vendor: HardwareVendor) -> bool {
        match vendor {
            HardwareVendor::Ledger => self.ledger.is_connected(),
            HardwareVendor::Trezor => self.trezor.is_connected(),
            HardwareVendor::OneKey => self.onekey.is_connected(),
            _ => false,
        }
    }
}

impl Default for HardwareWalletManager {
    fn default() -> Self {
        Self::new()
    }
}