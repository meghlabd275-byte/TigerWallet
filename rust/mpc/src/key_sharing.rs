//! MPC Key Sharing Module
//!
//! Key resharing, recovery and encrypted backup. Every path here uses the
//! share's own x-coordinate, so resharing and recovery work for any t-of-n
//! subset rather than only the first `threshold` shares.

use crate::backup_crypto;
use crate::field_ed25519::EdScalar;
use crate::vss;
use crate::{CurveType, MpcConfig, MpcError, ShareInfo};
use crypto_bigint::{Encoding, U256};
use rand::rngs::OsRng;
use rand::RngCore;
use serde::{Deserialize, Serialize};
use zeroize::Zeroizing;

/// Reshare: build a new share set from an existing subset, keeping the same key.
///
/// Used to change the threshold or the participant count without changing the
/// public key. The caller must supply at least `old_threshold` shares.
pub fn reshare(
    old_shares: &[ShareInfo],
    old_threshold: u32,
    new_config: &MpcConfig,
    public_key: &[u8],
) -> Result<Vec<ShareInfo>, MpcError> {
    if old_threshold < 2 {
        return Err(MpcError::InvalidConfig(
            "old threshold must be at least 2".to_string(),
        ));
    }
    if old_shares.len() < old_threshold as usize {
        return Err(MpcError::ThresholdNotMet(
            "Need old threshold shares to reshare".to_string(),
        ));
    }
    if new_config.threshold > new_config.total_shares || new_config.threshold < 2 {
        return Err(MpcError::InvalidConfig(
            "new threshold must be at least 2 and at most the share count".to_string(),
        ));
    }

    let t = old_threshold as usize;
    let selected = &old_shares[..t];

    match new_config.curve {
        CurveType::Secp256k1 => {
            let xs: Vec<u32> = selected.iter().map(|s| s.index).collect();
            let ys: Vec<U256> = selected
                .iter()
                .map(|s| crate::field::bytes_to_scalar(&s.share_data))
                .collect();
            let secret = crate::field::lagrange_at_zero(&xs, &ys);
            let mut coefficients: Zeroizing<Vec<U256>> =
                Zeroizing::new(Vec::with_capacity(new_config.threshold as usize));
            coefficients.push(secret);
            for _ in 1..new_config.threshold {
                coefficients.push(random_u256(secp256k1_in_field));
            }
            build_secp256k1_shares(&coefficients, new_config.total_shares, public_key)
        }
        CurveType::P256 => {
            let xs: Vec<u32> = selected.iter().map(|s| s.index).collect();
            let ys: Vec<U256> = selected
                .iter()
                .map(|s| crate::field_p256::bytes_to_scalar(&s.share_data))
                .collect();
            let secret = crate::field_p256::lagrange_at_zero(&xs, &ys);
            let mut coefficients: Zeroizing<Vec<U256>> =
                Zeroizing::new(Vec::with_capacity(new_config.threshold as usize));
            coefficients.push(secret);
            for _ in 1..new_config.threshold {
                coefficients.push(random_u256(p256_in_field));
            }
            build_p256_shares(&coefficients, new_config.total_shares, public_key)
        }
        CurveType::Ed25519 => {
            let xs: Vec<u32> = selected.iter().map(|s| s.index).collect();
            let ys: Vec<EdScalar> = selected
                .iter()
                .map(|s| EdScalar::from_le_bytes(&fixed32(&s.share_data)))
                .collect();
            let secret = EdScalar::lagrange_at_zero(&xs, &ys)
                .map_err(|e| MpcError::KeySharingFailed(e.to_string()))?;
            let mut coefficients: Zeroizing<Vec<EdScalar>> =
                Zeroizing::new(Vec::with_capacity(new_config.threshold as usize));
            coefficients.push(secret);
            for _ in 1..new_config.threshold {
                coefficients.push(EdScalar::random());
            }
            build_ed25519_shares(&coefficients, new_config.total_shares, public_key)
        }
    }
}

fn fixed32(data: &[u8]) -> [u8; 32] {
    let mut out = [0u8; 32];
    let n = data.len().min(32);
    out[..n].copy_from_slice(&data[..n]);
    out
}

fn secp256k1_in_field(be: &[u8; 32]) -> bool {
    use k256::elliptic_curve::PrimeField;
    Option::<k256::Scalar>::from(k256::Scalar::from_repr((*be).into())).is_some()
}

fn p256_in_field(be: &[u8; 32]) -> bool {
    use p256::elliptic_curve::PrimeField;
    Option::<p256::Scalar>::from(p256::Scalar::from_repr((*be).into())).is_some()
}

fn random_u256(in_field: fn(&[u8; 32]) -> bool) -> U256 {
    loop {
        let mut coeff = [0u8; 32];
        OsRng.fill_bytes(&mut coeff);
        if in_field(&coeff) {
            return U256::from_be_bytes(coeff);
        }
    }
}

/// Confirm the reconstructed secret reproduces the expected public key.
fn assert_secp256k1_key(secret: &U256, public_key: &[u8]) -> Result<(), MpcError> {
    use k256::ecdsa::SigningKey;
    let sk = SigningKey::from_slice(&crate::field::scalar_to_be_bytes(*secret))
        .map_err(|e| MpcError::KeySharingFailed(e.to_string()))?;
    let derived = sk.verifying_key().to_encoded_point(false).as_bytes().to_vec();
    if derived != public_key {
        return Err(MpcError::KeySharingFailed(
            "reshare source shares do not reconstruct the expected public key".to_string(),
        ));
    }
    Ok(())
}

fn assert_p256_key(secret: &U256, public_key: &[u8]) -> Result<(), MpcError> {
    use p256::ecdsa::SigningKey;
    let sk = SigningKey::from_slice(&crate::field_p256::scalar_to_be_bytes(*secret))
        .map_err(|e| MpcError::KeySharingFailed(e.to_string()))?;
    let derived = sk.verifying_key().to_encoded_point(false).as_bytes().to_vec();
    if derived != public_key {
        return Err(MpcError::KeySharingFailed(
            "reshare source shares do not reconstruct the expected public key".to_string(),
        ));
    }
    Ok(())
}

/// Confirm the reconstructed Ed25519 seed reproduces the expected public key.
///
/// The seed is *not* the signing scalar: ed25519 derives the scalar as
/// clamp(SHA512(seed)), so  is not the public key. The seed must go
/// through the real key-expansion path.
fn assert_ed25519_key(seed: &EdScalar, public_key: &[u8]) -> Result<(), MpcError> {
    let derived = ed25519_dalek::SigningKey::from_bytes(&seed.to_le_bytes())
        .verifying_key()
        .to_bytes()
        .to_vec();
    if derived != public_key {
        return Err(MpcError::KeySharingFailed(
            "reconstructed seed does not reproduce the expected public key".to_string(),
        ));
    }
    Ok(())
}

fn build_secp256k1_shares(
    coefficients: &[U256],
    total: u32,
    public_key: &[u8],
) -> Result<Vec<ShareInfo>, MpcError> {
    assert_secp256k1_key(&coefficients[0], public_key)?;
    let commitments =
        vss::commit_secp256k1(coefficients).map_err(|e| MpcError::KeySharingFailed(e.to_string()))?;
    let mut shares = Vec::with_capacity(total as usize);
    for i in 1..=total {
        let y = crate::field::eval_polynomial(coefficients, i);
        if !vss::verify_share_secp256k1(i, &y, &commitments)
            .map_err(|e| MpcError::KeySharingFailed(e.to_string()))?
        {
            return Err(MpcError::KeySharingFailed(
                "reshared share failed its own VSS check".to_string(),
            ));
        }
        shares.push(ShareInfo {
            index: i,
            share_data: crate::field::scalar_to_le_bytes(y).to_vec(),
            commitments: commitments.clone(),
        });
    }
    Ok(shares)
}

fn build_p256_shares(
    coefficients: &[U256],
    total: u32,
    public_key: &[u8],
) -> Result<Vec<ShareInfo>, MpcError> {
    assert_p256_key(&coefficients[0], public_key)?;
    let commitments =
        vss::commit_p256(coefficients).map_err(|e| MpcError::KeySharingFailed(e.to_string()))?;
    let mut shares = Vec::with_capacity(total as usize);
    for i in 1..=total {
        let y = crate::field_p256::eval_polynomial(coefficients, i);
        if !vss::verify_share_p256(i, &y, &commitments)
            .map_err(|e| MpcError::KeySharingFailed(e.to_string()))?
        {
            return Err(MpcError::KeySharingFailed(
                "reshared share failed its own VSS check".to_string(),
            ));
        }
        shares.push(ShareInfo {
            index: i,
            share_data: crate::field_p256::scalar_to_le_bytes(y).to_vec(),
            commitments: commitments.clone(),
        });
    }
    Ok(shares)
}

fn build_ed25519_shares(
    coefficients: &[EdScalar],
    total: u32,
    public_key: &[u8],
) -> Result<Vec<ShareInfo>, MpcError> {
    assert_ed25519_key(&coefficients[0], public_key)?;
    let commitments = vss::commit_ed25519(coefficients);
    let mut shares = Vec::with_capacity(total as usize);
    for i in 1..=total {
        let y = EdScalar::eval_polynomial(coefficients, i);
        if !vss::verify_share_ed25519(i, &y, &commitments)
            .map_err(|e| MpcError::KeySharingFailed(e.to_string()))?
        {
            return Err(MpcError::KeySharingFailed(
                "reshared share failed its own VSS check".to_string(),
            ));
        }
        shares.push(ShareInfo {
            index: i,
            share_data: y.to_le_bytes().to_vec(),
            commitments: commitments.clone(),
        });
    }
    Ok(shares)
}

/// Reconstruct the raw secret from a threshold subset of shares.
///
/// Returns the secret in the curve's canonical encoding: big-endian 32 bytes for
/// secp256k1 / P-256 (private-key form) and the 32-byte seed for Ed25519.
pub fn recover_key(
    shares: &[ShareInfo],
    threshold: u32,
    curve: CurveType,
) -> Result<Vec<u8>, MpcError> {
    if shares.len() < threshold as usize {
        return Err(MpcError::ThresholdNotMet(format!(
            "Need {} shares for recovery, got {}",
            threshold,
            shares.len()
        )));
    }
    if threshold < 2 {
        return Err(MpcError::InvalidConfig(
            "recovery threshold must be at least 2".to_string(),
        ));
    }

    let selected = &shares[..threshold as usize];
    let xs: Vec<u32> = selected.iter().map(|s| s.index).collect();

    match curve {
        CurveType::Secp256k1 => {
            let ys: Vec<U256> = selected
                .iter()
                .map(|s| crate::field::bytes_to_scalar(&s.share_data))
                .collect();
            let secret = crate::field::lagrange_at_zero(&xs, &ys);
            Ok(crate::field::scalar_to_be_bytes(secret).to_vec())
        }
        CurveType::P256 => {
            let ys: Vec<U256> = selected
                .iter()
                .map(|s| crate::field_p256::bytes_to_scalar(&s.share_data))
                .collect();
            let secret = crate::field_p256::lagrange_at_zero(&xs, &ys);
            Ok(crate::field_p256::scalar_to_be_bytes(secret).to_vec())
        }
        CurveType::Ed25519 => {
            let ys: Vec<EdScalar> = selected
                .iter()
                .map(|s| EdScalar::from_le_bytes(&fixed32(&s.share_data)))
                .collect();
            let seed = EdScalar::lagrange_at_zero(&xs, &ys)
                .map_err(|e| MpcError::KeySharingFailed(e.to_string()))?;
            Ok(seed.to_le_bytes().to_vec())
        }
    }
}

/// One encrypted share inside a backup bundle.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EncryptedShare {
    pub index: u32,
    pub blob: Vec<u8>,
    pub commitments: Vec<Vec<u8>>,
}

/// A password-protected backup of a full share set.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ShareBackup {
    pub version: u8,
    pub curve: CurveType,
    pub threshold: u32,
    pub total_shares: u32,
    pub key_id: String,
    pub shares: Vec<EncryptedShare>,
}

impl ShareBackup {
    fn aad(&self) -> Vec<u8> {
        let mut aad = Vec::new();
        aad.extend_from_slice(b"tiger-mpc-share-backup-v1");
        aad.extend_from_slice(self.key_id.as_bytes());
        aad.extend_from_slice(&self.threshold.to_le_bytes());
        aad.extend_from_slice(&self.total_shares.to_le_bytes());
        aad
    }
}

/// Encrypt a share set into a password-protected backup.
///
/// Each share is encrypted separately under a key derived from `passphrase` with
/// PBKDF2, and the bundle header is authenticated as associated data so the
/// threshold and key id cannot be altered.
pub fn backup_key(
    shares: &[ShareInfo],
    key_id: &str,
    threshold: u32,
    total_shares: u32,
    curve: CurveType,
    passphrase: &[u8],
) -> Result<Vec<u8>, MpcError> {
    if passphrase.len() < 12 {
        return Err(MpcError::EncryptionError(
            "backup passphrase must be at least 12 characters".to_string(),
        ));
    }

    let mut backup = ShareBackup {
        version: 1,
        curve,
        threshold,
        total_shares,
        key_id: key_id.to_string(),
        shares: Vec::with_capacity(shares.len()),
    };
    let aad = backup.aad();

    for s in shares {
        let blob = backup_crypto::encrypt(passphrase, &aad, &s.share_data)?;
        backup.shares.push(EncryptedShare {
            index: s.index,
            blob,
            commitments: s.commitments.clone(),
        });
    }

    serde_json::to_vec(&backup).map_err(|e| MpcError::EncryptionError(e.to_string()))
}

/// Restore a share set from a backup produced by [`backup_key`].
pub fn restore_from_backup(backup: &[u8], passphrase: &[u8]) -> Result<Vec<ShareInfo>, MpcError> {
    let parsed: ShareBackup = serde_json::from_slice(backup)
        .map_err(|e| MpcError::KeySharingFailed(format!("malformed backup: {}", e)))?;

    if parsed.version != 1 {
        return Err(MpcError::KeySharingFailed(format!(
            "unsupported backup version {}",
            parsed.version
        )));
    }
    if parsed.shares.is_empty() {
        return Err(MpcError::KeySharingFailed("backup contains no shares".to_string()));
    }

    let aad = parsed.aad();
    let mut out = Vec::with_capacity(parsed.shares.len());
    for es in &parsed.shares {
        let plaintext = backup_crypto::decrypt(passphrase, &aad, &es.blob).map_err(|_| {
            MpcError::KeySharingFailed(
                "backup authentication failed: wrong passphrase or tampered bundle".to_string(),
            )
        })?;
        out.push(ShareInfo {
            index: es.index,
            share_data: plaintext.to_vec(),
            commitments: es.commitments.clone(),
        });
    }

    // Re-verify every restored share against its commitments so a tampered
    // commitment list is caught even if the ciphertext is intact.
    for s in &out {
        if s.commitments.is_empty() {
            continue;
        }
        let ok = match parsed.curve {
            CurveType::Secp256k1 => {
                let v = crate::field::bytes_to_scalar(&s.share_data);
                vss::verify_share_secp256k1(s.index, &v, &s.commitments)
            }
            CurveType::P256 => {
                let v = crate::field_p256::bytes_to_scalar(&s.share_data);
                vss::verify_share_p256(s.index, &v, &s.commitments)
            }
            CurveType::Ed25519 => {
                let v = EdScalar::from_le_bytes(&fixed32(&s.share_data));
                vss::verify_share_ed25519(s.index, &v, &s.commitments)
            }
        }
        .map_err(|e| MpcError::KeySharingFailed(e.to_string()))?;
        if !ok {
            return Err(MpcError::KeySharingFailed(format!(
                "restored share {} failed VSS verification",
                s.index
            )));
        }
    }

    Ok(out)
}

/// Assert that a share set reconstructs the expected public key.
pub fn verify_share_set(
    shares: &[ShareInfo],
    threshold: u32,
    curve: CurveType,
    public_key: &[u8],
) -> Result<bool, MpcError> {
    let secret = recover_key(shares, threshold, curve)?;
    match curve {
        CurveType::Secp256k1 => {
            let mut be = [0u8; 32];
            be.copy_from_slice(&secret[..32]);
            Ok(assert_secp256k1_key(&U256::from_be_bytes(be), public_key).is_ok())
        }
        CurveType::P256 => {
            let mut be = [0u8; 32];
            be.copy_from_slice(&secret[..32]);
            Ok(assert_p256_key(&U256::from_be_bytes(be), public_key).is_ok())
        }
        CurveType::Ed25519 => {
            let s = EdScalar::from_le_bytes(&fixed32(&secret));
            Ok(assert_ed25519_key(&s, public_key).is_ok())
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::key_gen::generate_key_shares;
    use crate::KeyGenResult;

    const PASS: &str = "test-passphrase-for-mpc";

    fn setup(curve: CurveType, threshold: u32, total: u32) -> (KeyGenResult, MpcConfig) {
        std::env::set_var("MPC_BACKUP_PASSPHRASE", PASS);
        let config = MpcConfig {
            threshold,
            total_shares: total,
            curve,
            key_id: String::new(),
        };
        let result = generate_key_shares(&config, b"test entropy").unwrap();
        (result, config)
    }

    #[test]
    fn test_reshare() {
        let (result, _) = setup(CurveType::Secp256k1, 2, 3);
        let new_config = MpcConfig {
            threshold: 3,
            total_shares: 5,
            curve: CurveType::Secp256k1,
            key_id: String::new(),
        };
        let new_shares = reshare(&result.shares, 2, &new_config, &result.public_key).unwrap();
        assert_eq!(new_shares.len(), 5);
        assert!(
            verify_share_set(&new_shares, 3, CurveType::Secp256k1, &result.public_key).unwrap(),
            "reshared set must reconstruct the original public key"
        );
    }

    #[test]
    fn reshare_preserves_the_key_for_every_curve() {
        for curve in [CurveType::Secp256k1, CurveType::P256, CurveType::Ed25519] {
            let (result, _) = setup(curve, 2, 3);
            let new_config = MpcConfig {
                threshold: 3,
                total_shares: 5,
                curve,
                key_id: String::new(),
            };
            let new_shares = reshare(&result.shares, 2, &new_config, &result.public_key).unwrap();
            assert!(
                verify_share_set(&new_shares, 3, curve, &result.public_key).unwrap(),
                "{:?} reshare must preserve the public key",
                curve
            );
        }
    }

    #[test]
    fn reshare_rejects_a_mismatched_public_key() {
        let (a, _) = setup(CurveType::Secp256k1, 2, 3);
        let (b, _) = setup(CurveType::Secp256k1, 2, 3);
        let new_config = MpcConfig {
            threshold: 2,
            total_shares: 3,
            curve: CurveType::Secp256k1,
            key_id: String::new(),
        };
        assert!(
            reshare(&a.shares, 2, &new_config, &b.public_key).is_err(),
            "resharing with the wrong public key must fail"
        );
    }

    #[test]
    fn every_subset_recovers_the_same_secret() {
        let (result, _) = setup(CurveType::Secp256k1, 3, 5);
        let expected = &result.master_seed;
        for subset in [[0, 1, 2], [0, 2, 4], [1, 3, 4], [2, 3, 4]] {
            let shares: Vec<ShareInfo> = subset.iter().map(|&i| result.shares[i].clone()).collect();
            let got = recover_key(&shares, 3, CurveType::Secp256k1).unwrap();
            assert_eq!(&got, expected, "subset {:?}", subset);
        }
    }

    #[test]
    fn ed25519_recovery_returns_the_real_seed() {
        let (result, _) = setup(CurveType::Ed25519, 2, 3);
        let shares: Vec<ShareInfo> = result.shares[1..3].to_vec();
        let got = recover_key(&shares, 2, CurveType::Ed25519).unwrap();
        assert_eq!(got, result.master_seed);
    }

    #[test]
    fn test_backup_restore() {
        let (result, _) = setup(CurveType::Secp256k1, 2, 3);
        let backup = backup_key(
            &result.shares,
            &result.key_id,
            2,
            3,
            CurveType::Secp256k1,
            PASS.as_bytes(),
        )
        .unwrap();
        let restored = restore_from_backup(&backup, PASS.as_bytes()).unwrap();
        assert_eq!(restored.len(), result.shares.len());
        assert!(
            verify_share_set(&restored, 2, CurveType::Secp256k1, &result.public_key).unwrap(),
            "restored shares must reproduce the original key"
        );
    }

    #[test]
    fn backup_round_trips_for_every_curve() {
        for curve in [CurveType::Secp256k1, CurveType::P256, CurveType::Ed25519] {
            let (result, _) = setup(curve, 2, 3);
            let backup =
                backup_key(&result.shares, &result.key_id, 2, 3, curve, PASS.as_bytes()).unwrap();
            let restored = restore_from_backup(&backup, PASS.as_bytes()).unwrap();
            assert!(
                verify_share_set(&restored, 2, curve, &result.public_key).unwrap(),
                "{:?} backup round-trip must preserve the key",
                curve
            );
        }
    }

    #[test]
    fn backup_does_not_leak_share_bytes() {
        let (result, _) = setup(CurveType::Secp256k1, 2, 3);
        let backup = backup_key(
            &result.shares,
            &result.key_id,
            2,
            3,
            CurveType::Secp256k1,
            PASS.as_bytes(),
        )
        .unwrap();
        for s in &result.shares {
            assert!(
                !backup.windows(s.share_data.len()).any(|w| w == &s.share_data[..]),
                "share bytes must not appear in the backup"
            );
        }
    }

    #[test]
    fn backup_rejects_wrong_passphrase_and_tampering() {
        let (result, _) = setup(CurveType::Secp256k1, 2, 3);
        let backup = backup_key(
            &result.shares,
            &result.key_id,
            2,
            3,
            CurveType::Secp256k1,
            PASS.as_bytes(),
        )
        .unwrap();

        assert!(restore_from_backup(&backup, b"a-different-passphrase").is_err());

        let mut tampered: serde_json::Value = serde_json::from_slice(&backup).unwrap();
        tampered["threshold"] = serde_json::json!(1);
        let tampered_bytes = serde_json::to_vec(&tampered).unwrap();
        assert!(
            restore_from_backup(&tampered_bytes, PASS.as_bytes()).is_err(),
            "changing the authenticated header must invalidate the backup"
        );
    }

    #[test]
    fn backup_requires_a_long_passphrase() {
        let (result, _) = setup(CurveType::Secp256k1, 2, 3);
        assert!(backup_key(
            &result.shares,
            &result.key_id,
            2,
            3,
            CurveType::Secp256k1,
            b"short"
        )
        .is_err());
    }

    #[test]
    fn recovery_rejects_too_few_shares() {
        let (result, _) = setup(CurveType::Secp256k1, 3, 5);
        assert!(recover_key(&result.shares[..2], 3, CurveType::Secp256k1).is_err());
    }

    #[test]
    fn verify_share_set_detects_a_wrong_key() {
        let (a, _) = setup(CurveType::Secp256k1, 2, 3);
        let (b, _) = setup(CurveType::Secp256k1, 2, 3);
        assert!(!verify_share_set(&a.shares, 2, CurveType::Secp256k1, &b.public_key).unwrap());
        assert!(verify_share_set(&a.shares, 2, CurveType::Secp256k1, &a.public_key).unwrap());
    }
}
