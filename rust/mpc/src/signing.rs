//! MPC Signing Module
//!
//! Threshold signing that honours each share's real x-coordinate.
//!
//! The previous implementation interpolated with x-coordinates `1..=threshold`
//! regardless of which shares were supplied. Any subset other than the first
//! `threshold` shares therefore combined the wrong y-values with the wrong
//! x-values and produced a secret that was not the key. Here every share
//! contributes its own `index`, so any t-of-n subset reconstructs the same key.
//!
//! Each share is also checked against the Feldman commitments before it is used,
//! so a corrupted or malicious share is rejected instead of silently yielding a
//! broken signature.

use crate::field_ed25519::EdScalar;
use crate::vss;
use crate::{CurveType, MpcConfig, MpcError, SignResult};
use crypto_bigint::U256;
use k256::ecdsa::signature::Signer;
use k256::ecdsa::Signature;
use sha2::{Digest, Sha256};
use zeroize::Zeroizing;

/// A single participant's contribution: its index and share value.
#[derive(Debug, Clone)]
pub struct SigningShare {
    pub index: u32,
    pub share_data: Vec<u8>,
    /// Feldman commitments, when available. When present the share is verified.
    pub commitments: Vec<Vec<u8>>,
}

/// Sign data using a threshold subset of shares.
pub fn sign(
    shares: &[Vec<u8>],
    public_key: &[u8],
    message: &[u8],
    config: &MpcConfig,
) -> Result<SignResult, MpcError> {
    // Legacy call shape: x-coordinates are assumed to be 1..=threshold, which is
    // only correct for the first `threshold` shares of a fresh key.
    let indexed: Vec<SigningShare> = shares
        .iter()
        .enumerate()
        .map(|(i, d)| SigningShare {
            index: (i + 1) as u32,
            share_data: d.clone(),
            commitments: Vec::new(),
        })
        .collect();
    sign_with_shares(&indexed, public_key, message, config)
}

/// Sign using explicit share indices. This is the entry point callers should use.
pub fn sign_with_shares(
    shares: &[SigningShare],
    public_key: &[u8],
    message: &[u8],
    config: &MpcConfig,
) -> Result<SignResult, MpcError> {
    if shares.len() < config.threshold as usize {
        return Err(MpcError::ThresholdNotMet(format!(
            "Need {} shares, got {}",
            config.threshold,
            shares.len()
        )));
    }
    // Duplicate x-coordinates make the Lagrange denominator zero.
    let mut seen = std::collections::HashSet::new();
    for s in shares {
        if s.index == 0 {
            return Err(MpcError::InvalidShare(
                "share index 0 is the secret itself, not a share".to_string(),
            ));
        }
        if !seen.insert(s.index) {
            return Err(MpcError::InvalidShare(format!(
                "duplicate share index {}",
                s.index
            )));
        }
    }

    match config.curve {
        CurveType::Secp256k1 => sign_secp256k1(shares, public_key, message, config),
        CurveType::Ed25519 => sign_ed25519(shares, public_key, message, config),
        CurveType::P256 => sign_p256(shares, public_key, message, config),
    }
}

/// Verify every share that carries commitments before it is used.
fn verify_secp256k1_shares(shares: &[SigningShare]) -> Result<(), MpcError> {
    for s in shares {
        if s.commitments.is_empty() {
            continue;
        }
        if s.share_data.len() != 32 {
            return Err(MpcError::InvalidShare(format!(
                "share {} must be 32 bytes",
                s.index
            )));
        }
        let value = crate::field::bytes_to_scalar(&s.share_data);
        if !vss::verify_share_secp256k1(s.index, &value, &s.commitments)
            .map_err(|e| MpcError::InvalidShare(e.to_string()))?
        {
            return Err(MpcError::InvalidShare(format!(
                "share {} failed Feldman VSS verification",
                s.index
            )));
        }
    }
    Ok(())
}

fn sign_secp256k1(
    shares: &[SigningShare],
    public_key: &[u8],
    message: &[u8],
    config: &MpcConfig,
) -> Result<SignResult, MpcError> {
    use k256::ecdsa::SigningKey;

    verify_secp256k1_shares(shares)?;

    let threshold = config.threshold as usize;
    let selected = &shares[..threshold];

    let xs: Vec<u32> = selected.iter().map(|s| s.index).collect();
    let ys: Vec<U256> = selected
        .iter()
        .map(|s| crate::field::bytes_to_scalar(&s.share_data))
        .collect();

    let secret = crate::field::lagrange_at_zero(&xs, &ys);
    let secret_bytes = Zeroizing::new(crate::field::scalar_to_be_bytes(secret));

    let signing_key = SigningKey::from_slice(&*secret_bytes)
        .map_err(|e| MpcError::SigningFailed(e.to_string()))?;

    // Fail closed if the reconstructed key does not match the published one.
    // A mismatch means the shares were wrong, and signing anyway would produce a
    // signature that no verifier accepts.
    let derived = signing_key
        .verifying_key()
        .to_encoded_point(false)
        .as_bytes()
        .to_vec();
    if derived != public_key {
        return Err(MpcError::SigningFailed(
            "reconstructed key does not match the expected public key".to_string(),
        ));
    }

    // k256's `sign` hashes once internally (RFC 6979), matching `verify`.
    let signature: Signature = signing_key.sign(message);

    Ok(SignResult {
        signature: signature.to_bytes().to_vec(),
        public_key: public_key.to_vec(),
        key_id: String::new(),
    })
}

fn sign_ed25519(
    shares: &[SigningShare],
    public_key: &[u8],
    message: &[u8],
    config: &MpcConfig,
) -> Result<SignResult, MpcError> {
    use ed25519_dalek::{Signer, SigningKey};

    for s in shares {
        if s.commitments.is_empty() {
            continue;
        }
        if s.share_data.len() != 32 {
            return Err(MpcError::InvalidShare(format!(
                "share {} must be 32 bytes",
                s.index
            )));
        }
        let b: [u8; 32] = s.share_data[..].try_into().unwrap();
        let scalar = EdScalar::from_le_bytes(&b);
        if !vss::verify_share_ed25519(s.index, &scalar, &s.commitments)
            .map_err(|e| MpcError::InvalidShare(e.to_string()))?
        {
            return Err(MpcError::InvalidShare(format!(
                "share {} failed Feldman VSS verification",
                s.index
            )));
        }
    }

    let threshold = config.threshold as usize;
    let selected = &shares[..threshold];

    let xs: Vec<u32> = selected.iter().map(|s| s.index).collect();
    let ys: Vec<EdScalar> = selected
        .iter()
        .map(|s| {
            let b: [u8; 32] = s.share_data[..].try_into().unwrap();
            EdScalar::from_le_bytes(&b)
        })
        .collect();

    let seed = Zeroizing::new(
        EdScalar::lagrange_at_zero(&xs, &ys)
            .map_err(|e| MpcError::SigningFailed(e.to_string()))?
            .to_le_bytes(),
    );

    let signing_key = SigningKey::from_bytes(&*seed);

    // Same fail-closed check as secp256k1.
    if signing_key.verifying_key().to_bytes().to_vec() != public_key {
        return Err(MpcError::SigningFailed(
            "reconstructed key does not match the expected public key".to_string(),
        ));
    }

    let signature = signing_key.sign(message);

    Ok(SignResult {
        signature: signature.to_bytes().to_vec(),
        public_key: public_key.to_vec(),
        key_id: String::new(),
    })
}

/// Sign with P-256 (NIST P-256 / secp256r1) using real ECDSA (RFC 6979).
fn sign_p256(
    shares: &[SigningShare],
    public_key: &[u8],
    message: &[u8],
    config: &MpcConfig,
) -> Result<SignResult, MpcError> {
    use p256::ecdsa::signature::Signer;
    use p256::ecdsa::{Signature, SigningKey};

    for s in shares {
        if s.commitments.is_empty() {
            continue;
        }
        if s.share_data.len() != 32 {
            return Err(MpcError::InvalidShare(format!(
                "share {} must be 32 bytes",
                s.index
            )));
        }
        let value = crate::field_p256::bytes_to_scalar(&s.share_data);
        if !vss::verify_share_p256(s.index, &value, &s.commitments)
            .map_err(|e| MpcError::InvalidShare(e.to_string()))?
        {
            return Err(MpcError::InvalidShare(format!(
                "share {} failed Feldman VSS verification",
                s.index
            )));
        }
    }

    let threshold = config.threshold as usize;
    let selected = &shares[..threshold];

    let xs: Vec<u32> = selected.iter().map(|s| s.index).collect();
    let ys: Vec<U256> = selected
        .iter()
        .map(|s| crate::field_p256::bytes_to_scalar(&s.share_data))
        .collect();

    let secret = crate::field_p256::lagrange_at_zero(&xs, &ys);
    let secret_bytes = Zeroizing::new(crate::field_p256::scalar_to_be_bytes(secret));

    let signing_key = SigningKey::from_slice(&*secret_bytes)
        .map_err(|e| MpcError::SigningFailed(format!("invalid P-256 secret: {}", e)))?;

    let derived = signing_key
        .verifying_key()
        .to_encoded_point(false)
        .as_bytes()
        .to_vec();
    if derived != public_key {
        return Err(MpcError::SigningFailed(
            "reconstructed key does not match the expected public key".to_string(),
        ));
    }

    let signature: Signature = signing_key.sign(message);

    Ok(SignResult {
        signature: signature.to_bytes().to_vec(),
        public_key: public_key.to_vec(),
        key_id: format!(
            "p256-{}",
            hex::encode(&public_key[..public_key.len().min(8)])
        ),
    })
}

/// Verify a signature against a public key.
pub fn verify(
    signature: &[u8],
    public_key: &[u8],
    message: &[u8],
    curve: CurveType,
) -> Result<bool, MpcError> {
    match curve {
        CurveType::Secp256k1 => {
            use k256::ecdsa::signature::DigestVerifier;
            use k256::ecdsa::{Signature, VerifyingKey};

            let sig = match Signature::from_slice(signature) {
                Ok(s) => s,
                Err(_) => return Ok(false),
            };
            let verifying_key = match VerifyingKey::from_sec1_bytes(public_key) {
                Ok(k) => k,
                Err(_) => return Ok(false),
            };

            let mut hasher = Sha256::new();
            hasher.update(message);
            Ok(verifying_key.verify_digest(hasher, &sig).is_ok())
        }
        CurveType::Ed25519 => {
            use ed25519_dalek::{Signature as Ed25519Sig, Verifier, VerifyingKey};

            if signature.len() != 64 || public_key.len() != 32 {
                return Ok(false);
            }
            let sig_bytes: [u8; 64] = signature[..64].try_into().unwrap();
            let pk_bytes: [u8; 32] = public_key[..32].try_into().unwrap();

            let sig = Ed25519Sig::from_bytes(&sig_bytes);
            let pk = match VerifyingKey::from_bytes(&pk_bytes) {
                Ok(k) => k,
                Err(_) => return Ok(false),
            };
            Ok(pk.verify(message, &sig).is_ok())
        }
        CurveType::P256 => {
            use p256::ecdsa::signature::DigestVerifier;
            use p256::ecdsa::{Signature, VerifyingKey};

            let sig = match Signature::from_slice(signature) {
                Ok(s) => s,
                Err(_) => return Ok(false),
            };
            let verifying_key = match VerifyingKey::from_sec1_bytes(public_key) {
                Ok(k) => k,
                Err(_) => return Ok(false),
            };

            let mut hasher = Sha256::new();
            hasher.update(message);
            Ok(verifying_key.verify_digest(hasher, &sig).is_ok())
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::key_gen::generate_key_shares;

    const PASS: &str = "test-passphrase-for-mpc";

    fn setup(curve: CurveType, threshold: u32, total: u32) -> crate::KeyGenResult {
        std::env::set_var("MPC_BACKUP_PASSPHRASE", PASS);
        let config = MpcConfig {
            threshold,
            total_shares: total,
            curve,
            key_id: String::new(),
        };
        generate_key_shares(&config, b"test entropy").unwrap()
    }

    fn config_for(curve: CurveType, threshold: u32, total: u32) -> MpcConfig {
        MpcConfig {
            threshold,
            total_shares: total,
            curve,
            key_id: String::new(),
        }
    }

    #[test]
    fn every_subset_of_a_2_of_3_key_signs() {
        for curve in [CurveType::Secp256k1, CurveType::P256, CurveType::Ed25519] {
            let result = setup(curve, 2, 3);
            let config = config_for(curve, 2, 3);
            for (a, b) in [(0, 1), (0, 2), (1, 2)] {
                let shares: Vec<SigningShare> = [a, b]
                    .iter()
                    .map(|&i| SigningShare {
                        index: result.shares[i].index,
                        share_data: result.shares[i].share_data.clone(),
                        commitments: result.shares[i].commitments.clone(),
                    })
                    .collect();
                let msg = b"transfer";
                let sig = sign_with_shares(&shares, &result.public_key, msg, &config)
                    .unwrap_or_else(|e| panic!("{:?} subset ({},{}) failed: {}", curve, a, b, e));
                assert!(
                    verify(&sig.signature, &result.public_key, msg, curve).unwrap(),
                    "{:?} subset ({},{}) produced an invalid signature",
                    curve,
                    a,
                    b
                );
            }
        }
    }

    #[test]
    fn every_subset_of_a_3_of_5_key_signs() {
        let result = setup(CurveType::Secp256k1, 3, 5);
        let config = config_for(CurveType::Secp256k1, 3, 5);
        for subset in [[0, 1, 2], [0, 2, 4], [1, 3, 4], [2, 3, 4], [0, 1, 4]] {
            let shares: Vec<SigningShare> = subset
                .iter()
                .map(|&i| SigningShare {
                    index: result.shares[i].index,
                    share_data: result.shares[i].share_data.clone(),
                    commitments: result.shares[i].commitments.clone(),
                })
                .collect();
            let msg = b"transfer";
            let sig = sign_with_shares(&shares, &result.public_key, msg, &config)
                .unwrap_or_else(|e| panic!("subset {:?} failed: {}", subset, e));
            assert!(verify(&sig.signature, &result.public_key, msg, CurveType::Secp256k1).unwrap());
        }
    }

    #[test]
    fn test_threshold_not_met() {
        let result = setup(CurveType::Secp256k1, 2, 3);
        let config = config_for(CurveType::Secp256k1, 2, 3);
        let shares = vec![SigningShare {
            index: result.shares[0].index,
            share_data: result.shares[0].share_data.clone(),
            commitments: result.shares[0].commitments.clone(),
        }];
        assert!(sign_with_shares(&shares, &result.public_key, b"Hello", &config).is_err());
    }

    #[test]
    fn corrupted_share_is_rejected_before_signing() {
        let result = setup(CurveType::Secp256k1, 2, 3);
        let config = config_for(CurveType::Secp256k1, 2, 3);

        let mut tampered = result.shares[1].share_data.clone();
        tampered[0] ^= 0xff;

        let shares = vec![
            SigningShare {
                index: result.shares[0].index,
                share_data: result.shares[0].share_data.clone(),
                commitments: result.shares[0].commitments.clone(),
            },
            SigningShare {
                index: result.shares[1].index,
                share_data: tampered,
                commitments: result.shares[1].commitments.clone(),
            },
        ];
        let err = sign_with_shares(&shares, &result.public_key, b"msg", &config).unwrap_err();
        assert!(
            format!("{}", err).contains("VSS"),
            "a corrupted share must be rejected by VSS verification, got: {}",
            err
        );
    }

    #[test]
    fn duplicate_indices_are_rejected() {
        let result = setup(CurveType::Secp256k1, 2, 3);
        let config = config_for(CurveType::Secp256k1, 2, 3);
        let s = SigningShare {
            index: result.shares[0].index,
            share_data: result.shares[0].share_data.clone(),
            commitments: result.shares[0].commitments.clone(),
        };
        assert!(sign_with_shares(&[s.clone(), s], &result.public_key, b"m", &config).is_err());
    }

    #[test]
    fn index_zero_is_rejected() {
        let result = setup(CurveType::Secp256k1, 2, 3);
        let config = config_for(CurveType::Secp256k1, 2, 3);
        let shares = vec![
            SigningShare { index: 0, share_data: vec![1u8; 32], commitments: vec![] },
            SigningShare {
                index: result.shares[1].index,
                share_data: result.shares[1].share_data.clone(),
                commitments: vec![],
            },
        ];
        assert!(sign_with_shares(&shares, &result.public_key, b"m", &config).is_err());
    }

    #[test]
    fn mismatched_public_key_is_rejected() {
        let a = setup(CurveType::Secp256k1, 2, 3);
        let b = setup(CurveType::Secp256k1, 2, 3);
        let config = config_for(CurveType::Secp256k1, 2, 3);
        let shares: Vec<SigningShare> = a.shares[..2]
            .iter()
            .map(|s| SigningShare {
                index: s.index,
                share_data: s.share_data.clone(),
                commitments: s.commitments.clone(),
            })
            .collect();
        // Signing a's shares but claiming b's public key must fail.
        assert!(sign_with_shares(&shares, &b.public_key, b"m", &config).is_err());
    }

    #[test]
    fn tampered_message_does_not_verify() {
        let result = setup(CurveType::Secp256k1, 2, 3);
        let config = config_for(CurveType::Secp256k1, 2, 3);
        let shares: Vec<SigningShare> = result.shares[..2]
            .iter()
            .map(|s| SigningShare {
                index: s.index,
                share_data: s.share_data.clone(),
                commitments: s.commitments.clone(),
            })
            .collect();
        let sig = sign_with_shares(&shares, &result.public_key, b"original", &config).unwrap();
        assert!(
            !verify(&sig.signature, &result.public_key, b"tampered", CurveType::Secp256k1).unwrap()
        );
    }

    #[test]
    fn test_sign_and_verify_p256() {
        let result = setup(CurveType::P256, 2, 3);
        let config = config_for(CurveType::P256, 2, 3);
        let shares: Vec<SigningShare> = result.shares[..2]
            .iter()
            .map(|s| SigningShare {
                index: s.index,
                share_data: s.share_data.clone(),
                commitments: s.commitments.clone(),
            })
            .collect();
        let msg = b"Hello, P-256!";
        let sig = sign_with_shares(&shares, &result.public_key, msg, &config).unwrap();
        assert!(verify(&sig.signature, &result.public_key, msg, CurveType::P256).unwrap());
        assert!(!verify(&sig.signature, &result.public_key, b"tampered", CurveType::P256).unwrap());
    }
}
