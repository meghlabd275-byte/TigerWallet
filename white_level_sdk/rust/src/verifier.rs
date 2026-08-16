//! Real Ed25519 license-token verification.
//!
//! A `SignedLicenseToken` is produced by the TigerWallet SuperAdmin control
//! plane (`license_service/go/internal/crypto`). The token payload is
//! canonical-JSON serialized and signed with the control plane's Ed25519
//! private key. This module verifies the signature with the corresponding
//! public key (distributed to WL products out-of-band) and enforces the
//! status/expiry/validity-window constraints. Any failure => the token is
//! rejected (fail-closed). No path returns "valid" without a real, matching
//! Ed25519 signature over the exact token bytes.

use ed25519_dalek::{Signature, Verifier, VerifyingKey};
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};

/// The cryptographically-signed payload proving a WL product is authorized.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct LicenseToken {
    pub license_id: String,
    pub wl_client_id: String,
    pub product: String,
    pub plan: String,
    pub status: String, // active | suspended | revoked | expired | halted
    pub valid_from: i64,
    pub valid_until: i64,
    pub max_users: i64,
    pub max_wallets: i64,
    pub max_bots: i64,
    pub features: Vec<String>,
    pub issued_at: i64,
    pub not_before: i64,
    pub expires_at: i64, // short-lived token expiry
    pub nonce: String,
}

/// A LicenseToken plus its Ed25519 signature + the verifier public key (hex).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignedLicenseToken {
    pub token: LicenseToken,
    pub signature: String, // hex
    #[serde(default)]
    pub public_key: String, // hex (optional — usually known out-of-band)
}

/// LicenseVerifier holds the control plane's public key and verifies tokens.
pub struct LicenseVerifier {
    pubkey: VerifyingKey,
}

#[derive(Debug, thiserror::Error)]
pub enum VerifyError {
    #[error("invalid hex encoding")]
    BadHex,
    #[error("invalid public key")]
    BadKey,
    #[error("malformed signature")]
    BadSignature,
    #[error("signature invalid (tampered or forged)")]
    InvalidSignature,
    #[error("token expired at {0}")]
    TokenExpired(i64),
    #[error("token not yet valid (not_before={0})")]
    NotYetValid(i64),
    #[error("license status is {0}, not active")]
    NotActive(String),
    #[error("license validity window ended at {0}")]
    LicenseExpired(i64),
    #[error("serialization error: {0}")]
    Serde(#[from] serde_json::Error),
}

impl LicenseVerifier {
    /// Create a verifier from the control plane's hex-encoded Ed25519 public
    /// key (32 bytes / 64 hex chars). Distributed out-of-band to WL products.
    pub fn from_hex(pub_hex: &str) -> Result<Self, VerifyError> {
        let bytes = hex::decode(pub_hex).map_err(|_| VerifyError::BadHex)?;
        if bytes.len() != 32 {
            return Err(VerifyError::BadKey);
        }
        let mut arr = [0u8; 32];
        arr.copy_from_slice(&bytes);
        let pubkey = VerifyingKey::from_bytes(&arr).map_err(|_| VerifyError::BadKey)?;
        Ok(Self { pubkey })
    }

    /// Verify a signed license token: real Ed25519 signature check + status +
    /// expiry + validity-window constraints. Returns Ok(()) only when ALL pass.
    pub fn verify(&self, slt: &SignedLicenseToken) -> Result<(), VerifyError> {
        // 1. Real Ed25519 signature verification over the canonical token bytes.
        let payload = canonical_json(&slt.token)?;
        let sig_bytes = hex::decode(&slt.signature).map_err(|_| VerifyError::BadHex)?;
        if sig_bytes.len() != 64 {
            return Err(VerifyError::BadSignature);
        }
        let mut sig_arr = [0u8; 64];
        sig_arr.copy_from_slice(&sig_bytes);
        let sig = Signature::from_bytes(&sig_arr);
        self.pubkey
            .verify(&payload, &sig)
            .map_err(|_| VerifyError::InvalidSignature)?;

        // 2. Temporal + status constraints (defense in depth — even if a valid
        //    signature is replayed, an expired/suspended token is rejected).
        let now = now_secs();
        if slt.token.expires_at > 0 && now > slt.token.expires_at {
            return Err(VerifyError::TokenExpired(slt.token.expires_at));
        }
        if slt.token.not_before > 0 && now < slt.token.not_before {
            return Err(VerifyError::NotYetValid(slt.token.not_before));
        }
        if slt.token.status != "active" {
            return Err(VerifyError::NotActive(slt.token.status.clone()));
        }
        if slt.token.valid_until > 0 && now > slt.token.valid_until {
            return Err(VerifyError::LicenseExpired(slt.token.valid_until));
        }
        Ok(())
    }
}

/// canonical_json produces deterministic JSON (sorted object keys, no
/// whitespace) so the signature is stable across encoders (matches the Go
/// control plane's `canonicalJSON` which round-trips through map[string]any).
fn canonical_json(v: &LicenseToken) -> Result<Vec<u8>, VerifyError> {
    // Serialize to a serde_json::Value then re-serialize. serde_json sorts map
    // keys by default when emitting Value::Object, giving canonical ordering.
    let val: serde_json::Value = serde_json::to_value(v)?;
    let mut buf = Vec::new();
    let formatter = serde_json::ser::CompactFormatter;
    let mut ser = serde_json::Serializer::with_formatter(&mut buf, formatter);
    serde::Serialize::serialize(&val, &mut ser)?;
    Ok(buf)
}

fn now_secs() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_tampered_token() {
        // A verifier with an arbitrary key rejects anything not signed by it.
        let pub_hex = "0000000000000000000000000000000000000000000000000000000000000000";
        // 0x00..00 is not a valid Ed25519 point, so from_hex fails — use a real
        // test vector instead: generate a keypair, sign, tamper, verify fails.
        use ed25519_dalek::{SigningKey, Signer};
        let mut rng = rand::rngs::OsRng;
        let signing = SigningKey::generate(&mut rng);
        let verifying: VerifyingKey = (&signing).into();
        let pub_hex = hex::encode(verifying.to_bytes());
        let v = LicenseVerifier::from_hex(&pub_hex).unwrap();
        let mut token = sample_token();
        let payload = canonical_json(&token).unwrap();
        let sig = signing.sign(&payload);
        let slt = SignedLicenseToken {
            token: token.clone(),
            signature: hex::encode(sig.to_bytes()),
            public_key: pub_hex.clone(),
        };
        assert!(v.verify(&slt).is_ok());
        // Tamper: change the product. Signature no longer matches.
        token.product = "tampered".to_string();
        let slt_bad = SignedLicenseToken {
            token,
            signature: slt.signature.clone(),
            public_key: pub_hex,
        };
        assert!(matches!(v.verify(&slt_bad), Err(VerifyError::InvalidSignature)));
    }

    #[test]
    fn rejects_expired_token() {
        use ed25519_dalek::{SigningKey, Signer};
        let mut rng = rand::rngs::OsRng;
        let signing = SigningKey::generate(&mut rng);
        let verifying: VerifyingKey = (&signing).into();
        let v = LicenseVerifier::from_hex(&hex::encode(verifying.to_bytes())).unwrap();
        let mut token = sample_token();
        token.expires_at = 1; // expired in 1970
        let payload = canonical_json(&token).unwrap();
        let sig = signing.sign(&payload);
        let slt = SignedLicenseToken { token, signature: hex::encode(sig.to_bytes()), public_key: hex::encode(verifying.to_bytes()) };
        assert!(matches!(v.verify(&slt), Err(VerifyError::TokenExpired(_))));
    }

    #[test]
    fn rejects_suspended_status() {
        use ed25519_dalek::{SigningKey, Signer};
        let mut rng = rand::rngs::OsRng;
        let signing = SigningKey::generate(&mut rng);
        let verifying: VerifyingKey = (&signing).into();
        let v = LicenseVerifier::from_hex(&hex::encode(verifying.to_bytes())).unwrap();
        let mut token = sample_token();
        token.status = "suspended".to_string();
        let payload = canonical_json(&token).unwrap();
        let sig = signing.sign(&payload);
        let slt = SignedLicenseToken { token, signature: hex::encode(sig.to_bytes()), public_key: hex::encode(verifying.to_bytes()) };
        assert!(matches!(v.verify(&slt), Err(VerifyError::NotActive(_))));
    }

    fn sample_token() -> LicenseToken {
        let now = now_secs();
        LicenseToken {
            license_id: "lic-1".into(),
            wl_client_id: "wl-1".into(),
            product: "master_wallet".into(),
            plan: "pro".into(),
            status: "active".into(),
            valid_from: now - 60,
            valid_until: now + 86400,
            max_users: 100,
            max_wallets: 500,
            max_bots: 50,
            features: vec!["wallet.create".into()],
            issued_at: now,
            not_before: now - 60,
            expires_at: now + 300,
            nonce: "n-1".into(),
        }
    }
}
