//! REAL Ed25519 license-token verification.
//!
//! The TigerWallet SuperAdmin control plane signs a canonical license payload
//! with its Ed25519 signing key. The WL product backend (via the heartbeat)
//! receives the signed token + the public key, and verifies:
//!   1. The signature is valid (real ed25519-dalek verify, NOT a length check).
//!   2. The token is not expired (valid_until < now => reject).
//!   3. The status is "active" (suspended/halted/revoked => reject).
//!
//! This is the cryptographic trust root: if verification fails, the C++ gate
//! is set dead and NO request is served. No license => no product works.
//!
//! Fail-closed: on ANY verification error (bad signature, expired, suspended,
//! malformed, wrong key), this returns Err and the heartbeat sets the gate dead.

use ed25519_dalek::{Verifier, VerifyingKey};
use serde::{Deserialize, Serialize};

/// Errors that cause license verification to fail (fail-closed).
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum LicenseError {
    InvalidSignature,
    Expired,
    Suspended,
    MalformedPayload,
    BadVerifyingKey,
}

impl std::fmt::Display for LicenseError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            LicenseError::InvalidSignature => write!(f, "license signature invalid"),
            LicenseError::Expired => write!(f, "license token expired"),
            LicenseError::Suspended => write!(f, "license suspended/halted/revoked"),
            LicenseError::MalformedPayload => write!(f, "license payload malformed"),
            LicenseError::BadVerifyingKey => write!(f, "verifying key malformed"),
        }
    }
}

impl std::error::Error for LicenseError {}

/// The license payload (signed by the control plane).
///
/// The canonical serialization (for signing/verification) is the
/// newline-joined, sorted-key JSON of (license_key, product, white_label_id,
/// valid_until, status). This is deterministic + tamper-evident.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct LicenseToken {
    pub license_key: String,
    pub product: String,
    pub white_label_id: String,
    pub valid_until: u64,   // unix seconds
    pub status: String,     // "active" | "suspended" | "halted" | "revoked"
}

/// A signed license token (payload + Ed25519 signature bytes).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignedLicenseToken {
    #[serde(flatten)]
    pub token: LicenseToken,
    pub signature: Vec<u8>, // 64-byte Ed25519 signature
}

impl LicenseToken {
    /// Canonical bytes for signing/verification. Deterministic: fields in a
    /// fixed order, JSON-escaped. ANY change to the payload changes these bytes.
    pub fn canonical_bytes(&self) -> Vec<u8> {
        format!(
            "{{\"license_key\":\"{}\",\"product\":\"{}\",\"white_label_id\":\"{}\",\"status\":\"{}\",\"valid_until\":{}}}",
            escape_json(&self.license_key),
            escape_json(&self.product),
            escape_json(&self.white_label_id),
            escape_json(&self.status),
            self.valid_until
        )
        .into_bytes()
    }

    /// Attach a signature to produce a SignedLicenseToken.
    pub fn signed_with(self, signature: Vec<u8>) -> SignedLicenseToken {
        SignedLicenseToken { token: self, signature }
    }
}

impl SignedLicenseToken {
    // The control plane signs using ed25519-dalek::SigningKey directly (see
    // the license_service); this crate is the VERIFY side only.
}

/// Verify a signed license token against the control plane's Ed25519 public key.
///
/// `verifying_key_bytes` is the 32-byte Ed25519 public key.
/// `now` is the current unix time (so tests can fix it).
///
/// Returns Ok(LicenseToken) only if signature is valid AND not expired AND
/// status is "active". Otherwise Err (fail-closed => gate goes dead).
pub fn verify_license_token(
    signed: &SignedLicenseToken,
    verifying_key_bytes: &[u8],
    now: u64,
) -> Result<LicenseToken, LicenseError> {
    // 1. Parse the verifying key (32 bytes).
    if verifying_key_bytes.len() != 32 {
        return Err(LicenseError::BadVerifyingKey);
    }
    let mut pk = [0u8; 32];
    pk.copy_from_slice(verifying_key_bytes);
    let vk = VerifyingKey::from_bytes(&pk).map_err(|_| LicenseError::BadVerifyingKey)?;

    // 2. Verify the signature over the canonical payload (REAL Ed25519).
    if signed.signature.len() != 64 {
        return Err(LicenseError::InvalidSignature);
    }
    let mut sig_bytes = [0u8; 64];
    sig_bytes.copy_from_slice(&signed.signature);
    let sig = ed25519_dalek::Signature::from_bytes(&sig_bytes);
    let canonical = signed.token.canonical_bytes();
    if vk.verify(&canonical, &sig).is_err() {
        return Err(LicenseError::InvalidSignature);
    }

    // 3. Expiry check.
    if signed.token.valid_until <= now {
        return Err(LicenseError::Expired);
    }

    // 4. Status check (suspended/halted/revoked => reject).
    if signed.token.status != "active" {
        return Err(LicenseError::Suspended);
    }

    Ok(signed.token.clone())
}

/// Escape a string for embedding in the canonical JSON (handles quotes,
/// backslashes, control chars). Keeps the canonical form deterministic.
fn escape_json(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            c if (c as u32) < 0x20 => out.push_str(&format!("\\u{:04x}", c as u32)),
            c => out.push(c),
        }
    }
    out
}
