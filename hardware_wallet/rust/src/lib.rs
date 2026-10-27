//! TigerWallet hardware-wallet support.
//!
//! Each vendor has a driver in its own module. A driver talks to the device
//! through a transport; when no transport is attached every signing call fails
//! closed, so a caller can never mistake a missing device for a successful
//! signature.
//!
//! [`HardwareWalletManager`] holds one driver per vendor and routes calls to
//! them. It reports a missing vendor as an error rather than a silent success.

pub mod airgap;
pub mod deep;
pub mod ellipal;
pub mod ledger;
pub mod onekey;
pub mod safepal;
pub mod trezor;

pub use airgap::*;
pub use deep::*;
pub use ellipal::*;
pub use ledger::*;
pub use onekey::*;
pub use safepal::*;
pub use trezor::*;

/// The result type shared by the vendor drivers.
pub type HardwareResult<T> = Result<T, HWError>;

/// Supported hardware wallet vendors.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum HardwareVendor {
    Ledger,
    Trezor,
    OneKey,
    AirGap,
    Ellipal,
    SafePal,
}

impl HardwareVendor {
    /// The vendor name used in error messages and diagnostics.
    pub fn name(&self) -> &'static str {
        match self {
            HardwareVendor::Ledger => "ledger",
            HardwareVendor::Trezor => "trezor",
            HardwareVendor::OneKey => "onekey",
            HardwareVendor::AirGap => "airgap",
            HardwareVendor::Ellipal => "ellipal",
            HardwareVendor::SafePal => "safepal",
        }
    }

    /// Whether the vendor exchanges signing requests as QR codes.
    pub fn is_air_gapped(&self) -> bool {
        matches!(self, HardwareVendor::AirGap)
    }
}

/// A unified view over the vendor drivers.
///
/// Every method routes to the driver for the vendor. A vendor that is not
/// attached, or whose driver has no transport, produces an error; nothing here
/// invents a key or a signature.
pub struct HardwareWalletManager {
    ledger: LedgerWallet,
    trezor: TrezorWallet,
    onekey: OneKeyWallet,
    safepal: SafePalWallet,
    ellipal: EllipalWallet,
    airgap: AirGapWallet,
    chain_id: u64,
}

impl HardwareWalletManager {
    /// Create a manager with no device attached.
    pub fn new() -> Self {
        Self {
            ledger: LedgerWallet::new(),
            trezor: TrezorWallet::new(),
            onekey: OneKeyWallet::new(),
            safepal: SafePalWallet::new(),
            ellipal: EllipalWallet::new(),
            airgap: AirGapWallet::new(),
            chain_id: 1,
        }
    }

    /// Create a manager bound to a chain.
    pub fn for_chain(chain_id: u64) -> Self {
        let mut manager = Self::new();
        manager.chain_id = chain_id;
        manager
    }

    /// The chain the devices sign for.
    pub fn chain_id(&self) -> u64 {
        self.chain_id
    }

    /// Attach a Ledger.
    pub fn connect_ledger(&self, device: LedgerDevice) {
        self.ledger.connect(device);
    }

    /// Attach a Trezor.
    pub fn connect_trezor(&self, device: TrezorDevice) {
        self.trezor.connect(device);
    }

    /// Attach a OneKey.
    pub fn connect_onekey(&self, device: OneKeyDevice) {
        self.onekey.connect(device);
    }

    /// Attach a SafePal.
    pub fn connect_safepal(&self, device: SafePalDevice) {
        self.safepal.connect(device);
    }

    /// Attach an Ellipal.
    pub fn connect_ellipal(&self, device: EllipalDevice) {
        self.ellipal.connect(device);
    }

    /// Attach an AirGap.
    pub fn connect_airgap(&self, device: AirGapDevice) {
        self.airgap.connect(device);
    }

    /// Detach a vendor.
    pub fn disconnect(&self, vendor: HardwareVendor) {
        match vendor {
            HardwareVendor::Ledger => self.ledger.disconnect(),
            HardwareVendor::Trezor => self.trezor.disconnect(),
            HardwareVendor::OneKey => self.onekey.disconnect(),
            HardwareVendor::SafePal => self.safepal.disconnect(),
            HardwareVendor::Ellipal => self.ellipal.disconnect(),
            HardwareVendor::AirGap => self.airgap.disconnect(),
        }
    }

    /// Report whether a vendor is attached.
    pub fn is_connected(&self, vendor: HardwareVendor) -> bool {
        match vendor {
            HardwareVendor::Ledger => self.ledger.is_connected(),
            HardwareVendor::Trezor => self.trezor.is_connected(),
            HardwareVendor::OneKey => self.onekey.is_connected(),
            HardwareVendor::SafePal => self.safepal.is_connected(),
            HardwareVendor::Ellipal => self.ellipal.is_connected(),
            HardwareVendor::AirGap => self.airgap.is_connected(),
        }
    }

    /// The vendors that currently have a device attached.
    pub fn connected_vendors(&self) -> Vec<HardwareVendor> {
        [
            HardwareVendor::Ledger,
            HardwareVendor::Trezor,
            HardwareVendor::OneKey,
            HardwareVendor::SafePal,
            HardwareVendor::Ellipal,
            HardwareVendor::AirGap,
        ]
        .into_iter()
        .filter(|vendor| self.is_connected(*vendor))
        .collect()
    }

    /// Read a public key from a vendor.
    pub fn get_public_key(
        &self,
        vendor: HardwareVendor,
        path: &str,
    ) -> HardwareResult<DevicePublicKey> {
        if path.trim().is_empty() {
            return Err(HWError::DeviceRejected(
                "a derivation path is required".to_string(),
            ));
        }

        match vendor {
            HardwareVendor::Ledger => expect_public_key(self.ledger.get_public_key(path)?),
            HardwareVendor::Trezor => expect_public_key(self.trezor.get_public_key(path)?),
            HardwareVendor::OneKey => expect_public_key(self.onekey.get_public_key(path)?),
            HardwareVendor::SafePal => expect_public_key(self.safepal.get_public_key(path)?),
            HardwareVendor::Ellipal => expect_public_key(self.ellipal.get_public_key(path)?),
            HardwareVendor::AirGap => Err(HWError::TransportUnavailable {
                vendor: vendor.name().to_string(),
                reason: "the public key must be read by scanning the device's QR code".to_string(),
            }),
        }
    }

    /// Sign a transaction with a vendor's device.
    pub fn sign_transaction(
        &self,
        vendor: HardwareVendor,
        tx: &[u8],
    ) -> HardwareResult<DeviceSignature> {
        if tx.is_empty() {
            return Err(HWError::DeviceRejected(
                "the transaction payload is empty".to_string(),
            ));
        }

        match vendor {
            HardwareVendor::Ledger => {
                // The Ledger app needs the derivation path to select the key.
                Err(HWError::DeviceRejected(
                    "a derivation path is required for the Ledger device".to_string(),
                ))
            }
            HardwareVendor::Trezor => expect_signature(self.trezor.sign_transaction(tx)?),
            HardwareVendor::OneKey => expect_signature(self.onekey.sign_transaction(tx)?),
            HardwareVendor::SafePal => expect_signature(self.safepal.sign_transaction(tx)?),
            HardwareVendor::Ellipal => expect_signature(self.ellipal.sign_transaction(tx)?),
            HardwareVendor::AirGap => Err(HWError::TransportUnavailable {
                vendor: vendor.name().to_string(),
                reason: "the request must be shown as a QR code and the response scanned back"
                    .to_string(),
            }),
        }
    }

    /// Sign a transaction with a Ledger at a derivation path.
    ///
    /// The Ledger app takes the path as part of the APDU, so it is a separate
    /// entry point rather than an optional argument that could be forgotten.
    pub fn sign_transaction_with_path(
        &self,
        vendor: HardwareVendor,
        tx: &[u8],
        path: &str,
    ) -> HardwareResult<DeviceSignature> {
        if tx.is_empty() {
            return Err(HWError::DeviceRejected(
                "the transaction payload is empty".to_string(),
            ));
        }

        if path.trim().is_empty() {
            return Err(HWError::DeviceRejected(
                "a derivation path is required".to_string(),
            ));
        }

        match vendor {
            HardwareVendor::Ledger => {
                expect_signature(self.ledger.sign_transaction(tx, path)?)
            }
            HardwareVendor::Trezor => expect_signature(self.trezor.sign_transaction(tx)?),
            HardwareVendor::OneKey => expect_signature(self.onekey.sign_transaction(tx)?),
            HardwareVendor::SafePal => expect_signature(self.safepal.sign_transaction(tx)?),
            HardwareVendor::Ellipal => expect_signature(self.ellipal.sign_transaction(tx)?),
            HardwareVendor::AirGap => Err(HWError::TransportUnavailable {
                vendor: vendor.name().to_string(),
                reason: "the request must be shown as a QR code and the response scanned back"
                    .to_string(),
            }),
        }
    }

    /// Sign a message with a vendor's device.
    pub fn sign_message(
        &self,
        vendor: HardwareVendor,
        message: &[u8],
        path: &str,
    ) -> HardwareResult<DeviceSignature> {
        if message.is_empty() {
            return Err(HWError::DeviceRejected("the message is empty".to_string()));
        }

        if path.trim().is_empty() {
            return Err(HWError::DeviceRejected(
                "a derivation path is required".to_string(),
            ));
        }

        match vendor {
            HardwareVendor::Ledger => expect_signature(self.ledger.sign_message(message, path)?),
            HardwareVendor::Trezor => expect_signature(self.trezor.sign_message(message, path)?),
            HardwareVendor::OneKey | HardwareVendor::SafePal | HardwareVendor::Ellipal => {
                Err(HWError::TransportUnavailable {
                    vendor: vendor.name().to_string(),
                    reason: "the driver exposes transaction signing only".to_string(),
                })
            }
            HardwareVendor::AirGap => Err(HWError::TransportUnavailable {
                vendor: vendor.name().to_string(),
                reason: "the request must be shown as a QR code and the response scanned back"
                    .to_string(),
            }),
        }
    }

    /// Build the QR payload for an air-gapped signing request.
    pub fn airgap_sign_request(&self, tx: &[u8]) -> HardwareResult<String> {
        if tx.is_empty() {
            return Err(HWError::DeviceRejected(
                "the transaction payload is empty".to_string(),
            ));
        }

        if !self.airgap.is_connected() {
            return Err(HWError::DeviceNotFound {
                vendor: "airgap".to_string(),
            });
        }

        Ok(self.airgap.generate_sign_request(tx)?)
    }

    /// Read a signature that came back from an air-gapped device.
    pub fn parse_airgap_response(&self, qr_data: &str) -> HardwareResult<DeviceSignature> {
        let response = self.airgap.parse_signature(qr_data)?;

        let signature = decode_signature(&response.signature).ok_or_else(|| {
            HWError::InvalidResponse(
                "the response does not carry a 65-byte signature".to_string(),
            )
        })?;

        Ok(DeviceSignature {
            signature,
            recovery_id: None,
        })
    }
}

impl Default for HardwareWalletManager {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn no_vendor_is_connected_initially() {
        let manager = HardwareWalletManager::new();

        assert!(manager.connected_vendors().is_empty());

        for vendor in [
            HardwareVendor::Ledger,
            HardwareVendor::Trezor,
            HardwareVendor::OneKey,
            HardwareVendor::SafePal,
            HardwareVendor::Ellipal,
            HardwareVendor::AirGap,
        ] {
            assert!(!manager.is_connected(vendor), "{} reported connected", vendor.name());
        }
    }

    #[test]
    fn signing_without_a_device_fails_for_every_vendor() {
        let manager = HardwareWalletManager::new();

        for vendor in [
            HardwareVendor::Ledger,
            HardwareVendor::Trezor,
            HardwareVendor::OneKey,
            HardwareVendor::SafePal,
            HardwareVendor::Ellipal,
            HardwareVendor::AirGap,
        ] {
            let result = manager.sign_transaction_with_path(vendor, &[1, 2, 3], "m/44'/60'/0'/0/0");

            assert!(
                result.is_err(),
                "{} returned a signature without a device",
                vendor.name()
            );
        }
    }

    #[test]
    fn reading_a_key_without_a_device_fails_for_every_vendor() {
        let manager = HardwareWalletManager::new();

        for vendor in [
            HardwareVendor::Ledger,
            HardwareVendor::Trezor,
            HardwareVendor::OneKey,
            HardwareVendor::SafePal,
            HardwareVendor::Ellipal,
            HardwareVendor::AirGap,
        ] {
            assert!(
                manager.get_public_key(vendor, "m/44'/60'/0'/0/0").is_err(),
                "{} returned a key without a device",
                vendor.name()
            );
        }
    }

    #[test]
    fn disconnecting_an_unattached_vendor_is_harmless() {
        let manager = HardwareWalletManager::new();

        manager.disconnect(HardwareVendor::Ellipal);

        assert!(!manager.is_connected(HardwareVendor::Ellipal));
    }

    #[test]
    fn the_chain_id_is_carried_through() {
        assert_eq!(HardwareWalletManager::for_chain(56).chain_id(), 56);
    }

    #[test]
    fn only_airgap_is_air_gapped() {
        assert!(HardwareVendor::AirGap.is_air_gapped());
        assert!(!HardwareVendor::Ledger.is_air_gapped());
    }

    #[test]
    fn an_empty_transaction_is_rejected() {
        let manager = HardwareWalletManager::new();

        let error = manager
            .sign_transaction_with_path(HardwareVendor::Ledger, &[], "m/44'/60'/0'/0/0")
            .unwrap_err();

        assert!(matches!(error, HWError::DeviceRejected(_)));
    }

    #[test]
    fn a_ledger_signature_needs_a_derivation_path() {
        let manager = HardwareWalletManager::new();

        // The path selects the key on the device, so it cannot be omitted.
        let error = manager
            .sign_transaction(HardwareVendor::Ledger, &[1, 2, 3])
            .unwrap_err();

        assert!(matches!(error, HWError::DeviceRejected(_)));
    }
}
