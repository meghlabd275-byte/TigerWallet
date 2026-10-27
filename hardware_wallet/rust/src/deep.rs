//! Deep hardware-wallet integration.
//!
//! This module routes a signing request to the transport for a vendor and
//! returns what comes back. It does not sign anything itself: the point of a
//! hardware wallet is that the key never leaves the device, so a signature
//! produced here would be worthless.
//!
//! A device that cannot be reached produces an error. No path in this module
//! returns an empty or invented signature, because a caller that received one
//! would broadcast a transaction no chain accepts.

use crate::airgap::{AirGapDevice, AirGapError, AirGapWallet};
use crate::ellipal::{EllipalDevice, EllipalError, EllipalResponse, EllipalWallet};
use crate::ledger::{LedgerDevice, LedgerError, LedgerResponse, LedgerWallet};
use crate::onekey::{OneKeyDevice, OneKeyError, OneKeyResponse, OneKeyWallet};
use crate::safepal::{SafePalDevice, SafePalError, SafePalResponse, SafePalWallet};
use crate::trezor::{TrezorDevice, TrezorError, TrezorResponse, TrezorWallet};

/// The vendors this layer can reach.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DeepVendor {
    Ledger,
    LedgerStax,
    Trezor,
    TrezorSafe3,
    OneKey,
    SafePal,
    Ellipal,
    AirGap,
    Keystone,
}

impl DeepVendor {
    /// The vendor name used in error messages and diagnostics.
    pub fn name(&self) -> &'static str {
        match self {
            DeepVendor::Ledger => "ledger",
            DeepVendor::LedgerStax => "ledger-stax",
            DeepVendor::Trezor => "trezor",
            DeepVendor::TrezorSafe3 => "trezor-safe-3",
            DeepVendor::OneKey => "onekey",
            DeepVendor::SafePal => "safepal",
            DeepVendor::Ellipal => "ellipal",
            DeepVendor::AirGap => "airgap",
            DeepVendor::Keystone => "keystone",
        }
    }

    /// Whether the vendor signs away from this machine and exchanges results as
    /// QR codes.
    pub fn is_air_gapped(&self) -> bool {
        matches!(self, DeepVendor::AirGap | DeepVendor::Keystone)
    }
}

/// A failure in the deep-integration layer.
#[derive(Debug, thiserror::Error)]
pub enum HWError {
    #[error("the {vendor} transport is unavailable: {reason}")]
    TransportUnavailable { vendor: String, reason: String },

    #[error("no {vendor} device is connected")]
    DeviceNotFound { vendor: String },

    #[error("the device rejected the request: {0}")]
    DeviceRejected(String),

    #[error("the device response could not be read: {0}")]
    InvalidResponse(String),

    #[error("the {vendor} driver reported: {message}")]
    Vendor { vendor: String, message: String },
}

impl From<LedgerError> for HWError {
    fn from(error: LedgerError) -> Self {
        HWError::Vendor {
            vendor: "ledger".to_string(),
            message: error.to_string(),
        }
    }
}

impl From<TrezorError> for HWError {
    fn from(error: TrezorError) -> Self {
        HWError::Vendor {
            vendor: "trezor".to_string(),
            message: error.to_string(),
        }
    }
}

impl From<OneKeyError> for HWError {
    fn from(error: OneKeyError) -> Self {
        HWError::Vendor {
            vendor: "onekey".to_string(),
            message: error.to_string(),
        }
    }
}

impl From<SafePalError> for HWError {
    fn from(error: SafePalError) -> Self {
        HWError::Vendor {
            vendor: "safepal".to_string(),
            message: error.to_string(),
        }
    }
}

impl From<EllipalError> for HWError {
    fn from(error: EllipalError) -> Self {
        HWError::Vendor {
            vendor: "ellipal".to_string(),
            message: error.to_string(),
        }
    }
}

impl From<AirGapError> for HWError {
    fn from(error: AirGapError) -> Self {
        HWError::Vendor {
            vendor: "airgap".to_string(),
            message: error.to_string(),
        }
    }
}

/// A public key read from a device, with the address it controls.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DevicePublicKey {
    pub public_key: String,
    /// Present when the vendor reports it. A device that does not report an
    /// address leaves this `None` rather than having one invented for it.
    pub address: Option<String>,
}

/// A signature produced by a device.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DeviceSignature {
    /// The 65-byte `r || s || v` value.
    pub signature: Vec<u8>,
    /// The recovery id, when the vendor reports it separately.
    pub recovery_id: Option<u8>,
}

/// A hardware wallet the deep layer holds open.
pub struct HardwareWalletDeep {
    /// The chain id the device should sign for.
    pub chain_id: u64,
    ledger: LedgerWallet,
    trezor: TrezorWallet,
    onekey: OneKeyWallet,
    safepal: SafePalWallet,
    ellipal: EllipalWallet,
    airgap: AirGapWallet,
}

impl HardwareWalletDeep {
    /// Create the integration for a chain.
    pub fn new(chain_id: u64) -> Self {
        Self {
            chain_id,
            ledger: LedgerWallet::new(),
            trezor: TrezorWallet::new(),
            onekey: OneKeyWallet::new(),
            safepal: SafePalWallet::new(),
            ellipal: EllipalWallet::new(),
            airgap: AirGapWallet::new(),
        }
    }

    /// The chain the device should sign for.
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
    pub fn disconnect(&self, vendor: DeepVendor) {
        match vendor {
            DeepVendor::Ledger | DeepVendor::LedgerStax => self.ledger.disconnect(),
            DeepVendor::Trezor | DeepVendor::TrezorSafe3 => self.trezor.disconnect(),
            DeepVendor::OneKey => self.onekey.disconnect(),
            DeepVendor::SafePal => self.safepal.disconnect(),
            DeepVendor::Ellipal => self.ellipal.disconnect(),
            DeepVendor::AirGap | DeepVendor::Keystone => self.airgap.disconnect(),
        }
    }

    /// Report whether a vendor is attached.
    pub fn is_connected(&self, vendor: DeepVendor) -> bool {
        match vendor {
            DeepVendor::Ledger | DeepVendor::LedgerStax => self.ledger.is_connected(),
            DeepVendor::Trezor | DeepVendor::TrezorSafe3 => self.trezor.is_connected(),
            DeepVendor::OneKey => self.onekey.is_connected(),
            DeepVendor::SafePal => self.safepal.is_connected(),
            DeepVendor::Ellipal => self.ellipal.is_connected(),
            DeepVendor::AirGap | DeepVendor::Keystone => self.airgap.is_connected(),
        }
    }

    /// Read a public key from a vendor at a derivation path.
    ///
    /// The call fails unless the vendor is attached and its transport works; no
    /// key is ever invented.
    pub fn get_public_key(&self, vendor: DeepVendor, path: &str) -> Result<DevicePublicKey, HWError> {
        if path.trim().is_empty() {
            return Err(HWError::DeviceRejected(
                "a derivation path is required".to_string(),
            ));
        }

        match vendor {
            DeepVendor::Ledger | DeepVendor::LedgerStax => {
                expect_public_key(self.ledger.get_public_key(path)?)
            }
            DeepVendor::Trezor | DeepVendor::TrezorSafe3 => {
                expect_public_key(self.trezor.get_public_key(path)?)
            }
            DeepVendor::OneKey => expect_public_key(self.onekey.get_public_key(path)?),
            DeepVendor::SafePal => expect_public_key(self.safepal.get_public_key(path)?),
            DeepVendor::Ellipal => expect_public_key(self.ellipal.get_public_key(path)?),
            DeepVendor::AirGap | DeepVendor::Keystone => Err(HWError::TransportUnavailable {
                vendor: vendor.name().to_string(),
                reason: "the public key must be read by scanning the device's QR code".to_string(),
            }),
        }
    }

    /// Ask a vendor to sign a transaction.
    ///
    /// The signature is the device's own. If the device cannot be reached the
    /// call fails; it never returns an empty or invented value.
    pub fn sign_transaction(
        &self,
        vendor: DeepVendor,
        tx: &[u8],
        path: &str,
    ) -> Result<DeviceSignature, HWError> {
        if tx.is_empty() {
            return Err(HWError::DeviceRejected(
                "the transaction payload is empty".to_string(),
            ));
        }

        match vendor {
            DeepVendor::Ledger | DeepVendor::LedgerStax => {
                expect_signature(self.ledger.sign_transaction(tx, path)?)
            }
            DeepVendor::Trezor | DeepVendor::TrezorSafe3 => {
                expect_signature(self.trezor.sign_transaction(tx)?)
            }
            DeepVendor::OneKey => expect_signature(self.onekey.sign_transaction(tx)?),
            DeepVendor::SafePal => expect_signature(self.safepal.sign_transaction(tx)?),
            DeepVendor::Ellipal => expect_signature(self.ellipal.sign_transaction(tx)?),
            DeepVendor::AirGap | DeepVendor::Keystone => Err(HWError::TransportUnavailable {
                vendor: vendor.name().to_string(),
                reason: "the request must be shown as a QR code and the response scanned back"
                    .to_string(),
            }),
        }
    }

    /// Ask a vendor to sign a message.
    pub fn sign_message(
        &self,
        vendor: DeepVendor,
        message: &[u8],
        path: &str,
    ) -> Result<DeviceSignature, HWError> {
        if message.is_empty() {
            return Err(HWError::DeviceRejected("the message is empty".to_string()));
        }

        match vendor {
            DeepVendor::Ledger | DeepVendor::LedgerStax => {
                expect_signature(self.ledger.sign_message(message, path)?)
            }
            DeepVendor::Trezor | DeepVendor::TrezorSafe3 => {
                expect_signature(self.trezor.sign_message(message, path)?)
            }
            DeepVendor::OneKey | DeepVendor::SafePal => Err(HWError::TransportUnavailable {
                vendor: vendor.name().to_string(),
                reason: "the driver exposes transaction signing only".to_string(),
            }),
            DeepVendor::Ellipal => Err(HWError::TransportUnavailable {
                vendor: vendor.name().to_string(),
                reason: "message signing is not part of the Ellipal QR protocol".to_string(),
            }),
            DeepVendor::AirGap | DeepVendor::Keystone => Err(HWError::TransportUnavailable {
                vendor: vendor.name().to_string(),
                reason: "the request must be shown as a QR code and the response scanned back"
                    .to_string(),
            }),
        }
    }

    /// Build the QR payload for an air-gapped signing request.
    ///
    /// The payload carries the transaction and a nonce so the device can show
    /// the user what they are approving and reject a replay.
    pub fn airgap_sign_request(&self, tx: &[u8]) -> Result<String, HWError> {
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
    ///
    /// The response is accepted only when it carries a well-formed signature
    /// over a 65-byte `r || s || v` layout. A truncated or malformed value is
    /// rejected rather than passed on to be broadcast.
    pub fn parse_airgap_response(&self, qr_data: &str) -> Result<DeviceSignature, HWError> {
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

impl Default for HardwareWalletDeep {
    fn default() -> Self {
        Self::new(1)
    }
}

/// Decode a `0x`-prefixed signature, returning `None` unless it is exactly 65
/// bytes.
///
/// The length check matters: a device that reports a different length has
/// either truncated the value or sent something that is not a signature, and
/// neither can be broadcast.
pub fn decode_signature(signature: &str) -> Option<Vec<u8>> {
    let trimmed = signature.trim();
    let digits = trimmed.strip_prefix("0x").unwrap_or(trimmed);

    if digits.len() != 130 {
        return None;
    }

    let decoded = hex::decode(digits).ok()?;

    if decoded.len() == 65 {
        Some(decoded)
    } else {
        None
    }
}

/// Turn a vendor response into a public key, or report why it is not one.
pub fn expect_public_key<T: VendorResponse>(response: T) -> Result<DevicePublicKey, HWError> {
    response.into_public_key().ok_or_else(|| {
        HWError::InvalidResponse("the device did not return a public key".to_string())
    })
}

/// Turn a vendor response into a signature, or report why it is not one.
pub fn expect_signature<T: VendorResponse>(response: T) -> Result<DeviceSignature, HWError> {
    response.into_signature().ok_or_else(|| {
        HWError::InvalidResponse("the device did not return a signature".to_string())
    })
}

/// A vendor response that carries either a public key or a signature.
pub trait VendorResponse {
    fn into_public_key(self) -> Option<DevicePublicKey>;
    fn into_signature(self) -> Option<DeviceSignature>;
}

impl VendorResponse for LedgerResponse {
    fn into_public_key(self) -> Option<DevicePublicKey> {
        match self {
            LedgerResponse::PublicKey {
                public_key,
                address,
            } => Some(DevicePublicKey {
                public_key,
                address: Some(address),
            }),
            _ => None,
        }
    }

    fn into_signature(self) -> Option<DeviceSignature> {
        match self {
            LedgerResponse::Signature { signature } => Some(DeviceSignature {
                signature,
                recovery_id: None,
            }),
            _ => None,
        }
    }
}

impl VendorResponse for TrezorResponse {
    fn into_public_key(self) -> Option<DevicePublicKey> {
        match self {
            TrezorResponse::PublicKey { public_key, .. } => Some(DevicePublicKey {
                public_key,
                address: None,
            }),
            _ => None,
        }
    }

    fn into_signature(self) -> Option<DeviceSignature> {
        match self {
            TrezorResponse::Signature { signature, recid } => Some(DeviceSignature {
                signature,
                recovery_id: Some(recid),
            }),
            _ => None,
        }
    }
}

impl VendorResponse for OneKeyResponse {
    fn into_public_key(self) -> Option<DevicePublicKey> {
        match self {
            OneKeyResponse::PublicKey {
                public_key,
                address,
            } => Some(DevicePublicKey {
                public_key,
                address: Some(address),
            }),
            _ => None,
        }
    }

    fn into_signature(self) -> Option<DeviceSignature> {
        match self {
            OneKeyResponse::Signature { signature } => decode_signature(&signature).map(|bytes| {
                DeviceSignature {
                    signature: bytes,
                    recovery_id: None,
                }
            }),
            _ => None,
        }
    }
}

impl VendorResponse for SafePalResponse {
    fn into_public_key(self) -> Option<DevicePublicKey> {
        match self {
            SafePalResponse::PublicKey { public_key } => Some(DevicePublicKey {
                public_key,
                address: None,
            }),
            _ => None,
        }
    }

    fn into_signature(self) -> Option<DeviceSignature> {
        match self {
            SafePalResponse::Signature { signature } => decode_signature(&signature).map(|bytes| {
                DeviceSignature {
                    signature: bytes,
                    recovery_id: None,
                }
            }),
            _ => None,
        }
    }
}

impl VendorResponse for EllipalResponse {
    fn into_public_key(self) -> Option<DevicePublicKey> {
        match self {
            EllipalResponse::PublicKey {
                public_key,
                address,
            } => Some(DevicePublicKey {
                public_key,
                address: Some(address),
            }),
            _ => None,
        }
    }

    fn into_signature(self) -> Option<DeviceSignature> {
        match self {
            EllipalResponse::Signature { signature } => decode_signature(&signature).map(|bytes| {
                DeviceSignature {
                    signature: bytes,
                    recovery_id: None,
                }
            }),
            _ => None,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_transaction_cannot_be_signed_without_a_device() {
        let deep = HardwareWalletDeep::new(1);

        let error = deep
            .sign_transaction(DeepVendor::Ledger, &[1, 2, 3], "m/44'/60'/0'/0/0")
            .unwrap_err();

        // The failure is reported; an empty signature is never returned.
        assert!(matches!(error, HWError::Vendor { .. } | HWError::DeviceNotFound { .. }));
    }

    #[test]
    fn a_public_key_cannot_be_read_without_a_device() {
        let deep = HardwareWalletDeep::new(1);

        let error = deep
            .get_public_key(DeepVendor::Trezor, "m/44'/60'/0'/0/0")
            .unwrap_err();

        assert!(matches!(error, HWError::Vendor { .. }));
    }

    #[test]
    fn an_empty_transaction_is_rejected() {
        let deep = HardwareWalletDeep::new(1);

        let error = deep
            .sign_transaction(DeepVendor::Ledger, &[], "m/44'/60'/0'/0/0")
            .unwrap_err();

        assert!(matches!(error, HWError::DeviceRejected(_)));
    }

    #[test]
    fn an_empty_message_is_rejected() {
        let deep = HardwareWalletDeep::new(1);

        let error = deep
            .sign_message(DeepVendor::Trezor, &[], "m/44'/60'/0'/0/0")
            .unwrap_err();

        assert!(matches!(error, HWError::DeviceRejected(_)));
    }

    #[test]
    fn an_empty_derivation_path_is_rejected() {
        let deep = HardwareWalletDeep::new(1);

        let error = deep.get_public_key(DeepVendor::Ledger, "  ").unwrap_err();

        assert!(matches!(error, HWError::DeviceRejected(_)));
    }

    #[test]
    fn an_air_gapped_request_needs_a_device() {
        let deep = HardwareWalletDeep::new(1);

        let error = deep.airgap_sign_request(&[1, 2, 3]).unwrap_err();

        assert!(matches!(error, HWError::DeviceNotFound { .. }));
    }

    #[test]
    fn an_air_gapped_signature_must_be_sixty_five_bytes() {
        let deep = HardwareWalletDeep::new(1);

        // A response carrying a short signature is refused rather than
        // forwarded to the chain.
        let short = format!(
            r#"{{"signature":"0x{}","public_key":"0x02"}}"#,
            "ab".repeat(64)
        );

        let error = deep.parse_airgap_response(&short).unwrap_err();

        assert!(matches!(error, HWError::InvalidResponse(_)));
    }

    #[test]
    fn an_air_gapped_signature_is_accepted_when_well_formed() {
        let deep = HardwareWalletDeep::new(1);

        let good = format!(
            r#"{{"signature":"0x{}","public_key":"0x02"}}"#,
            "ab".repeat(65)
        );

        let signature = deep.parse_airgap_response(&good).unwrap();

        assert_eq!(signature.signature.len(), 65);
    }

    #[test]
    fn a_signature_must_be_sixty_five_bytes() {
        assert!(decode_signature(&("0x".to_string() + &"ab".repeat(65))).is_some());
        assert!(decode_signature(&("ab".repeat(65))).is_some());

        // Too short, too long, not hex, and empty must all be refused.
        assert!(decode_signature(&("0x".to_string() + &"ab".repeat(64))).is_none());
        assert!(decode_signature(&("0x".to_string() + &"ab".repeat(66))).is_none());
        assert!(decode_signature("0xzz").is_none());
        assert!(decode_signature("").is_none());
    }

    #[test]
    fn a_vendor_is_reported_by_name() {
        assert_eq!(DeepVendor::LedgerStax.name(), "ledger-stax");
        assert_eq!(DeepVendor::TrezorSafe3.name(), "trezor-safe-3");
    }

    #[test]
    fn only_qr_vendors_are_air_gapped() {
        assert!(DeepVendor::AirGap.is_air_gapped());
        assert!(DeepVendor::Keystone.is_air_gapped());
        assert!(!DeepVendor::Ledger.is_air_gapped());
    }

    #[test]
    fn the_chain_id_is_carried_through() {
        assert_eq!(HardwareWalletDeep::new(137).chain_id(), 137);
    }

    #[test]
    fn a_response_that_is_not_a_key_is_reported() {
        let response = LedgerResponse::Error {
            code: 1,
            message: "no".to_string(),
        };

        assert!(expect_public_key(response).is_err());
    }

    #[test]
    fn a_public_key_response_carries_its_address() {
        let response = LedgerResponse::PublicKey {
            public_key: "0x02".to_string(),
            address: "0x9d8A62f656a8d1615C1294fd71e9CFb3E4855A4F".to_string(),
        };

        let key = expect_public_key(response).unwrap();

        assert_eq!(key.address.as_deref(), Some("0x9d8A62f656a8d1615C1294fd71e9CFb3E4855A4F"));
    }

    #[test]
    fn a_trezor_key_response_leaves_the_address_unset() {
        // Trezor reports a chain code, not an address; nothing is invented.
        let response = TrezorResponse::PublicKey {
            public_key: "0x02".to_string(),
            chain_code: "0x00".to_string(),
        };

        let key = expect_public_key(response).unwrap();

        assert!(key.address.is_none());
    }

    #[test]
    fn a_trezor_signature_carries_its_recovery_id() {
        let response = TrezorResponse::Signature {
            signature: vec![0u8; 65],
            recid: 1,
        };

        let signature = expect_signature(response).unwrap();

        assert_eq!(signature.recovery_id, Some(1));
        assert_eq!(signature.signature.len(), 65);
    }
}
