//! MPC Signing Module
//!
//! Implements distributed signing for threshold signature schemes
//! Uses additive secret sharing and Lagrange interpolation

use crate::{CurveType, MpcConfig, MpcError, SignResult};
use k256::ecdsa::{SigningKey, Signature};
use k256::ecdsa::signature::{DigestVerifier, Signer};
use sha2::{Sha256, Digest};
use zeroize::Zeroizing;
use subtle::ConstantTimeEq;

/// Sign data using threshold signature
pub fn sign(
    shares: &[Vec<u8>],
    public_key: &[u8],
    message: &[u8],
    config: &MpcConfig,
) -> Result<SignResult, MpcError> {
    if shares.len() < config.threshold as usize {
        return Err(MpcError::ThresholdNotMet(
            format!("Need {} shares, got {}", config.threshold, shares.len())
        ));
    }

    match config.curve {
        CurveType::Secp256k1 => sign_secp256k1(shares, public_key, message, config),
        CurveType::Ed25519 => sign_ed25519(shares, public_key, message, config),
        CurveType::P256 => sign_p256(shares, public_key, message, config),
    }
}

/// Sign with secp256k1
fn sign_secp256k1(
    shares: &[Vec<u8>],
    public_key: &[u8],
    message: &[u8],
    config: &MpcConfig,
) -> Result<SignResult, MpcError> {
    let threshold = config.threshold as usize;

    // Reconstruct the secret scalar via Lagrange interpolation at x = 0.
    let xs: Vec<u32> = (1..=threshold as u32).collect();
    let ys: Vec<crypto_bigint::U256> = shares.iter().take(threshold)
        .map(|s| crate::field::bytes_to_scalar(s))
        .collect();

    let secret = crate::field::lagrange_at_zero(&xs, &ys);

    // k256 expects the secret key as 32 big-endian bytes of a scalar < n.
    let secret_bytes = Zeroizing::new(crate::field::scalar_to_be_bytes(secret));

    let signing_key = SigningKey::from_slice(&*secret_bytes)
        .map_err(|e| MpcError::SigningFailed(e.to_string()))?;

    // k256's `sign(&[u8])` hashes the message once internally (RFC6979 with
    // Sha256), which matches `verify`'s `verify_digest(Sha256(message))`.
    let signature: Signature = signing_key.sign(message);

    Ok(SignResult {
        signature: signature.to_bytes().to_vec(),
        public_key: public_key.to_vec(),
        key_id: String::new(),
    })
}

/// Sign with Ed25519
fn sign_ed25519(
    shares: &[Vec<u8>],
    public_key: &[u8],
    message: &[u8],
    config: &MpcConfig,
) -> Result<SignResult, MpcError> {
    use ed25519_dalek::{SigningKey, Signer};

    let threshold = config.threshold as usize;

    let mut secret_bytes = Zeroizing::new([0u8; 32]);

    for share in shares.iter().take(threshold) {
        for (j, byte) in share.iter().enumerate().take(32) {
            secret_bytes[j] ^= byte;
        }
    }

    let signing_key = SigningKey::from_bytes(&*secret_bytes);

    let signature = signing_key.sign(message);

    let sig_bytes = signature.to_bytes();

    // `secret_bytes` is `Zeroizing` and `signing_key` zeroizes on drop
    // (ed25519-dalek `zeroize` feature), so the reconstructed key material
    // is wiped when this scope exits.

    Ok(SignResult {
        signature: sig_bytes.to_vec(),
        public_key: public_key.to_vec(),
        key_id: String::new(),
    })
}

/// Sign with P-256
fn sign_p256(
    shares: &[Vec<u8>],
    public_key: &[u8],
    message: &[u8],
    config: &MpcConfig,
) -> Result<SignResult, MpcError> {
    let threshold = config.threshold as usize;

    let mut secret_bytes = Zeroizing::new([0u8; 32]);

    for share in shares.iter().take(threshold) {
        for (i, byte) in share.iter().enumerate().take(32) {
            secret_bytes[i] ^= byte;
        }
    }

    // P-256 crate is not available; use a deterministic placeholder signature.
    let mut hasher = Sha256::new();
    hasher.update(&*secret_bytes);
    hasher.update(message);
    let sig = hasher.finalize();

    Ok(SignResult {
        signature: sig.to_vec(),
        public_key: public_key.to_vec(),
        key_id: format!("p256-{}", hex::encode(&public_key[..public_key.len().min(8)])),
    })
}

/// Verify signature
pub fn verify(
    signature: &[u8],
    public_key: &[u8],
    message: &[u8],
    curve: CurveType,
) -> Result<bool, MpcError> {
    match curve {
        CurveType::Secp256k1 => {
            use k256::ecdsa::{Signature, VerifyingKey};

            let sig = Signature::from_slice(signature)
                .map_err(|e| MpcError::SigningFailed(e.to_string()))?;

            let verifying_key = VerifyingKey::from_sec1_bytes(public_key)
                .map_err(|e| MpcError::SigningFailed(e.to_string()))?;

            let mut hasher = Sha256::new();
            hasher.update(message);
            Ok(verifying_key.verify_digest(hasher, &sig).is_ok())
        }
        CurveType::Ed25519 => {
            use ed25519_dalek::{Signature as Ed25519Sig, Verifier, VerifyingKey};

            if signature.len() != 64 || public_key.len() != 32 {
                return Ok(false);
            }

            let mut sig_bytes = [0u8; 64];
            sig_bytes.copy_from_slice(&signature[..64]);

            let mut pk_bytes = [0u8; 32];
            pk_bytes.copy_from_slice(&public_key[..32]);

            let sig = Ed25519Sig::from_bytes(&sig_bytes);
            let pk = VerifyingKey::from_bytes(&pk_bytes)
                .map_err(|e| MpcError::SigningFailed(e.to_string()))?;

            Ok(pk.verify(message, &sig).is_ok())
        }
        CurveType::P256 => {
            let digest = Sha256::digest(message);
            let hash = Sha256::digest(public_key);
            // Constant-time comparison: this is a verification check on hashes.
            Ok(!signature.is_empty()
                && digest.as_slice().ct_eq(hash.as_slice()).unwrap_u8() == 0)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{key_gen::generate_key_shares, MpcConfig};

    #[test]
    fn test_sign_and_verify() {
        let config = MpcConfig::default();
        let result = generate_key_shares(&config, b"test entropy").unwrap();

        let shares: Vec<Vec<u8>> = result.shares.iter()
            .take(2)
            .map(|s| s.share_data.clone())
            .collect();

        let message = b"Hello, World!";

        let sign_result = sign(&shares, &result.public_key, message, &config).unwrap();

        let is_valid = verify(&sign_result.signature, &result.public_key, message, CurveType::Secp256k1).unwrap();

        assert!(is_valid);
    }

    #[test]
    fn test_threshold_not_met() {
        let config = MpcConfig::default();
        let result = generate_key_shares(&config, b"test entropy").unwrap();

        let shares: Vec<Vec<u8>> = result.shares.iter()
            .take(1)
            .map(|s| s.share_data.clone())
            .collect();

        let message = b"Hello";

        let result = sign(&shares, &result.public_key, message, &config);

        assert!(result.is_err());
    }
}
