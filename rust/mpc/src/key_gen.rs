//! MPC Key Generation Module
//!
//! Distributed key generation (DKG) with Feldman-VSS verifiable secret sharing.
//!
//! The secret is split with a Shamir polynomial over the curve's *scalar field*
//! and every share carries a real curve commitment, so a participant can verify
//! its share before using it. For secp256k1 and P-256 the shared secret is the
//! signing key itself. For Ed25519 the shared secret is the 32-byte seed, which
//! is the value `ed25519-dalek` expands into the signing scalar.

use crate::backup_crypto;
use crate::field_ed25519::EdScalar;
use crate::vss;
use crate::{CurveType, KeyGenResult, MpcConfig, MpcError, ShareInfo};
use crypto_bigint::{Encoding, U256};
use k256::elliptic_curve::PrimeField;
use rand::rngs::OsRng;
use rand::RngCore;
use sha2::{Digest, Sha256};
use zeroize::{Zeroize, Zeroizing};

/// Number of bytes in a seed / private key.
const SEED_LEN: usize = 32;

/// Derive a canonical scalar from caller entropy plus fresh OS randomness.
///
/// `is_canonical` rejects values at or above the curve order, so this
/// rejection-samples until the result is a valid field element. Returning an
/// unreduced value would make the shares interpolate to something that is not
/// the key.
fn derive_canonical_seed<F>(entropy: &[u8], is_canonical: F) -> [u8; SEED_LEN]
where
    F: Fn(&[u8; SEED_LEN]) -> bool,
{
    loop {
        let mut rnd = [0u8; SEED_LEN];
        OsRng.fill_bytes(&mut rnd);
        let mut hasher = Sha256::new();
        hasher.update(entropy);
        hasher.update(rnd);
        let candidate: [u8; SEED_LEN] = hasher.finalize().into();
        if is_canonical(&candidate) {
            return candidate;
        }
    }
}

/// Generate MPC key shares using distributed key generation.
///
/// The backup passphrase is read from `MPC_BACKUP_PASSPHRASE`. Prefer
/// [`generate_key_shares_with_passphrase`] when the caller already manages the
/// secret, so the passphrase never has to live in the process environment.
pub fn generate_key_shares(config: &MpcConfig, entropy: &[u8]) -> Result<KeyGenResult, MpcError> {
    let passphrase = backup_passphrase()?;
    generate_key_shares_with_passphrase(config, entropy, &passphrase)
}

/// Generate MPC key shares, encrypting the master-seed backup with `passphrase`.
pub fn generate_key_shares_with_passphrase(
    config: &MpcConfig,
    entropy: &[u8],
    passphrase: &[u8],
) -> Result<KeyGenResult, MpcError> {
    if passphrase.len() < 12 {
        return Err(MpcError::EncryptionError(
            "backup passphrase must be at least 12 characters".to_string(),
        ));
    }
    if config.threshold > config.total_shares {
        return Err(MpcError::InvalidConfig(
            "Threshold must be less than or equal to total shares".to_string(),
        ));
    }
    if config.threshold < 2 {
        return Err(MpcError::InvalidConfig("Threshold must be at least 2".to_string()));
    }
    if config.total_shares < 2 {
        return Err(MpcError::InvalidConfig(
            "Total shares must be at least 2".to_string(),
        ));
    }

    match config.curve {
        CurveType::Secp256k1 => generate_secp256k1_shares(config, entropy, passphrase),
        CurveType::Ed25519 => generate_ed25519_shares(config, entropy, passphrase),
        CurveType::P256 => generate_p256_shares(config, entropy, passphrase),
    }
}

/// Rejection-sample a random in-field coefficient encoded big-endian.
fn random_coefficient(in_field: fn(&[u8; 32]) -> bool) -> U256 {
    loop {
        let mut coeff = [0u8; SEED_LEN];
        OsRng.fill_bytes(&mut coeff);
        if in_field(&coeff) {
            return U256::from_be_bytes(coeff);
        }
    }
}

/// True when `be` is below the secp256k1 group order.
fn secp256k1_in_field(be: &[u8; 32]) -> bool {
    Option::<k256::Scalar>::from(k256::Scalar::from_repr((*be).into())).is_some()
}

/// True when `be` is below the P-256 group order.
fn p256_in_field(be: &[u8; 32]) -> bool {
    Option::<p256::Scalar>::from(p256::Scalar::from_repr((*be).into())).is_some()
}

fn generate_secp256k1_shares(
    config: &MpcConfig,
    entropy: &[u8],
    passphrase: &[u8],
) -> Result<KeyGenResult, MpcError> {
    use k256::ecdsa::SigningKey;

    let seed = derive_canonical_seed(entropy, secp256k1_in_field);
    let signing_key =
        SigningKey::from_bytes((&seed).into()).map_err(|e| MpcError::KeyGenFailed(e.to_string()))?;
    let public_key = signing_key
        .verifying_key()
        .to_encoded_point(false)
        .as_bytes()
        .to_vec();

    let threshold = config.threshold as usize;
    let total_shares = config.total_shares as usize;

    // coeffs[0] is the secret, encoded exactly as k256 encodes the scalar, so
    // interpolation reproduces the signing key byte-for-byte.
    let mut coefficients: Zeroizing<Vec<U256>> = Zeroizing::new(Vec::with_capacity(threshold));
    coefficients.push(U256::from_be_bytes(signing_key.to_bytes().into()));
    for _ in 1..threshold {
        coefficients.push(random_coefficient(secp256k1_in_field));
    }

    let commitments =
        vss::commit_secp256k1(&coefficients).map_err(|e| MpcError::KeyGenFailed(e.to_string()))?;

    let mut shares = Vec::with_capacity(total_shares);
    for i in 1..=total_shares {
        let y = crate::field::eval_polynomial(&coefficients, i as u32);
        if !vss::verify_share_secp256k1(i as u32, &y, &commitments)
            .map_err(|e| MpcError::KeyGenFailed(e.to_string()))?
        {
            return Err(MpcError::KeyGenFailed(
                "generated share failed its own VSS check".to_string(),
            ));
        }
        shares.push(ShareInfo {
            index: i as u32,
            share_data: crate::field::scalar_to_le_bytes(y).to_vec(),
            commitments: commitments.clone(),
        });
    }

    let key_id = generate_key_id(&public_key);
    let backup = encrypt_master_key(&seed, &key_id, passphrase)?;

    Ok(KeyGenResult {
        key_id,
        public_key,
        shares,
        backup,
        master_seed: seed.to_vec(),
    })
}

/// Generate Ed25519 key shares.
///
/// The seed is shared over the Ed25519 group order. This is exact because the
/// seed is rejection-sampled below the group order, so interpolation returns the
/// same 32 bytes that produced the public key. Sharing the derived signing
/// scalar instead would not work: `ed25519-dalek` clamps it, so it is not
/// uniform over the field and cannot be split with a Shamir polynomial.
fn generate_ed25519_shares(
    config: &MpcConfig,
    entropy: &[u8],
    passphrase: &[u8],
) -> Result<KeyGenResult, MpcError> {
    use ed25519_dalek::SigningKey;

    let seed = derive_canonical_seed(entropy, EdScalar::is_canonical);
    let signing_key = SigningKey::from_bytes(&seed);
    let public_key = signing_key.verifying_key().to_bytes().to_vec();

    let threshold = config.threshold as usize;
    let total_shares = config.total_shares as usize;

    let mut coefficients: Zeroizing<Vec<EdScalar>> = Zeroizing::new(Vec::with_capacity(threshold));
    coefficients.push(EdScalar::from_le_bytes(&seed));
    for _ in 1..threshold {
        coefficients.push(EdScalar::random());
    }

    let commitments = vss::commit_ed25519(&coefficients);

    let mut shares = Vec::with_capacity(total_shares);
    for i in 1..=total_shares {
        let y = EdScalar::eval_polynomial(&coefficients, i as u32);
        if !vss::verify_share_ed25519(i as u32, &y, &commitments)
            .map_err(|e| MpcError::KeyGenFailed(e.to_string()))?
        {
            return Err(MpcError::KeyGenFailed(
                "generated share failed its own VSS check".to_string(),
            ));
        }
        shares.push(ShareInfo {
            index: i as u32,
            share_data: y.to_le_bytes().to_vec(),
            commitments: commitments.clone(),
        });
    }

    let key_id = generate_key_id(&public_key);
    let backup = encrypt_master_key(&seed, &key_id, passphrase)?;

    Ok(KeyGenResult {
        key_id,
        public_key,
        shares,
        backup,
        master_seed: seed.to_vec(),
    })
}

/// Generate P-256 key shares.
fn generate_p256_shares(
    config: &MpcConfig,
    entropy: &[u8],
    passphrase: &[u8],
) -> Result<KeyGenResult, MpcError> {
    use p256::ecdsa::SigningKey;

    let seed = derive_canonical_seed(entropy, p256_in_field);
    let signing_key =
        SigningKey::from_bytes((&seed).into()).map_err(|e| MpcError::KeyGenFailed(e.to_string()))?;
    let public_key = signing_key
        .verifying_key()
        .to_encoded_point(false)
        .as_bytes()
        .to_vec();

    let threshold = config.threshold as usize;
    let total_shares = config.total_shares as usize;

    let mut coefficients: Zeroizing<Vec<U256>> = Zeroizing::new(Vec::with_capacity(threshold));
    coefficients.push(U256::from_be_bytes(signing_key.to_bytes().into()));
    for _ in 1..threshold {
        coefficients.push(random_coefficient(p256_in_field));
    }

    let commitments =
        vss::commit_p256(&coefficients).map_err(|e| MpcError::KeyGenFailed(e.to_string()))?;

    let mut shares = Vec::with_capacity(total_shares);
    for i in 1..=total_shares {
        let y = crate::field_p256::eval_polynomial(&coefficients, i as u32);
        if !vss::verify_share_p256(i as u32, &y, &commitments)
            .map_err(|e| MpcError::KeyGenFailed(e.to_string()))?
        {
            return Err(MpcError::KeyGenFailed(
                "generated share failed its own VSS check".to_string(),
            ));
        }
        shares.push(ShareInfo {
            index: i as u32,
            share_data: crate::field_p256::scalar_to_le_bytes(y).to_vec(),
            commitments: commitments.clone(),
        });
    }

    let key_id = generate_key_id(&public_key);
    let backup = encrypt_master_key(&seed, &key_id, passphrase)?;

    Ok(KeyGenResult {
        key_id,
        public_key,
        shares,
        backup,
        master_seed: seed.to_vec(),
    })
}

/// Load the backup passphrase.
///
/// Fail-closed: without an operator-configured passphrase the seed is never
/// written out, because a default or empty passphrase would make every backup
/// trivially decryptable.
pub fn backup_passphrase() -> Result<Zeroizing<Vec<u8>>, MpcError> {
    match std::env::var("MPC_BACKUP_PASSPHRASE") {
        Ok(v) if v.len() >= 12 => Ok(Zeroizing::new(v.into_bytes())),
        Ok(_) => Err(MpcError::EncryptionError(
            "MPC_BACKUP_PASSPHRASE must be at least 12 characters".to_string(),
        )),
        Err(_) => Err(MpcError::EncryptionError(
            "MPC_BACKUP_PASSPHRASE is not set; refusing to write an unencrypted backup".to_string(),
        )),
    }
}

/// Encrypt the master seed for backup with AES-256-GCM.
///
/// The key id is bound as associated data, so a backup blob cannot be replayed
/// against a different key.
fn encrypt_master_key(seed: &[u8], key_id: &str, passphrase: &[u8]) -> Result<Vec<u8>, MpcError> {
    backup_crypto::encrypt(passphrase, key_id.as_bytes(), seed)
}

/// Decrypt a master-key backup using an explicit passphrase.
pub fn decrypt_master_key_with_passphrase(
    backup: &[u8],
    key_id: &str,
    passphrase: &[u8],
) -> Result<Zeroizing<Vec<u8>>, MpcError> {
    backup_crypto::decrypt(passphrase, key_id.as_bytes(), backup)
}

/// Decrypt a master-key backup using `MPC_BACKUP_PASSPHRASE`.
pub fn decrypt_master_key(backup: &[u8], key_id: &str) -> Result<Zeroizing<Vec<u8>>, MpcError> {
    let passphrase = backup_passphrase()?;
    decrypt_master_key_with_passphrase(backup, key_id, &passphrase)
}

/// Generate unique key ID from the public key.
fn generate_key_id(public_key: &[u8]) -> String {
    let hash = Sha256::digest(public_key);
    hex::encode(&hash[..16])
}

/// Overwrite key material in place.
pub fn wipe(buf: &mut [u8]) {
    buf.zeroize();
}

#[cfg(test)]
mod tests {
    use super::*;

    const PASS: &[u8] = b"test-passphrase-for-mpc";

    fn gen(config: &MpcConfig) -> KeyGenResult {
        generate_key_shares_with_passphrase(config, b"test entropy", PASS).unwrap()
    }

    #[test]
    fn test_key_generation() {
        let config = MpcConfig::default();
        let result = gen(&config);

        assert_eq!(result.shares.len(), 3);
        assert!(!result.public_key.is_empty());
        assert!(!result.key_id.is_empty());
    }

    #[test]
    fn test_different_thresholds() {
        for threshold in 2..=5 {
            let config = MpcConfig {
                threshold,
                total_shares: 5,
                curve: CurveType::Secp256k1,
                key_id: String::new(),
            };
            let result = gen(&config);
            assert_eq!(result.shares.len(), 5);
        }
    }

    #[test]
    fn rejects_invalid_configuration() {
        let bad = MpcConfig {
            threshold: 4,
            total_shares: 3,
            curve: CurveType::Secp256k1,
            key_id: String::new(),
        };
        assert!(generate_key_shares_with_passphrase(&bad, b"e", PASS).is_err());

        let one = MpcConfig {
            threshold: 1,
            total_shares: 3,
            curve: CurveType::Secp256k1,
            key_id: String::new(),
        };
        assert!(
            generate_key_shares_with_passphrase(&one, b"e", PASS).is_err(),
            "threshold 1 is not threshold signing"
        );
    }

    #[test]
    fn short_passphrase_is_rejected() {
        let config = MpcConfig::default();
        assert!(generate_key_shares_with_passphrase(&config, b"entropy", b"short").is_err());
    }

    #[test]
    fn env_passphrase_is_required_by_the_env_entry_point() {
        // The env-based entry point must fail closed when the variable is unset.
        let saved = std::env::var("MPC_BACKUP_PASSPHRASE").ok();
        std::env::remove_var("MPC_BACKUP_PASSPHRASE");
        let config = MpcConfig::default();
        assert!(
            generate_key_shares(&config, b"entropy").is_err(),
            "must not write a backup without a passphrase"
        );
        if let Some(v) = saved {
            std::env::set_var("MPC_BACKUP_PASSPHRASE", v);
        }
    }

    #[test]
    fn backup_round_trips_and_rejects_a_wrong_key_id() {
        let config = MpcConfig::default();
        let result = gen(&config);

        let seed = decrypt_master_key_with_passphrase(&result.backup, &result.key_id, PASS).unwrap();
        assert_eq!(&seed[..], &result.master_seed[..]);
        assert!(
            decrypt_master_key_with_passphrase(&result.backup, "some-other-key-id", PASS).is_err(),
            "a backup must not decrypt under a different key id"
        );
        assert!(
            decrypt_master_key_with_passphrase(&result.backup, &result.key_id, b"wrong-passphrase").is_err()
        );
    }

    #[test]
    fn backup_does_not_contain_the_seed() {
        let config = MpcConfig::default();
        let result = gen(&config);
        assert!(
            !result
                .backup
                .windows(result.master_seed.len())
                .any(|w| w == &result.master_seed[..]),
            "the master seed must never appear in the backup blob"
        );
    }

    #[test]
    fn secp256k1_master_seed_reconstructs_the_public_key() {
        let result = gen(&MpcConfig::default());
        let seed: [u8; 32] = result.master_seed[..].try_into().unwrap();
        let sk = k256::ecdsa::SigningKey::from_bytes((&seed).into()).unwrap();
        assert_eq!(
            sk.verifying_key().to_encoded_point(false).as_bytes().to_vec(),
            result.public_key
        );
    }

    #[test]
    fn ed25519_master_seed_reconstructs_the_public_key() {
        let mut config = MpcConfig::default();
        config.curve = CurveType::Ed25519;
        let result = gen(&config);

        let seed: [u8; 32] = result.master_seed[..].try_into().unwrap();
        let sk = ed25519_dalek::SigningKey::from_bytes(&seed);
        assert_eq!(sk.verifying_key().to_bytes().to_vec(), result.public_key);
    }

    #[test]
    fn p256_master_seed_reconstructs_the_public_key() {
        let mut config = MpcConfig::default();
        config.curve = CurveType::P256;
        let result = gen(&config);

        let seed: [u8; 32] = result.master_seed[..].try_into().unwrap();
        let sk = p256::ecdsa::SigningKey::from_bytes((&seed).into()).unwrap();
        assert_eq!(
            sk.verifying_key().to_encoded_point(false).as_bytes().to_vec(),
            result.public_key
        );
    }

    #[test]
    fn shares_carry_real_commitments() {
        let config = MpcConfig::default();
        let result = gen(&config);

        for s in &result.shares {
            assert_eq!(
                s.commitments.len(),
                config.threshold as usize,
                "one commitment per polynomial coefficient"
            );
            for c in &s.commitments {
                assert_eq!(c.len(), 33, "secp256k1 commitments are compressed points");
            }
        }
    }

    #[test]
    fn ed25519_shares_verify_against_their_commitments() {
        let mut config = MpcConfig::default();
        config.curve = CurveType::Ed25519;
        let result = gen(&config);

        for s in &result.shares {
            let b: [u8; 32] = s.share_data[..].try_into().unwrap();
            let scalar = EdScalar::from_le_bytes(&b);
            assert!(vss::verify_share_ed25519(s.index, &scalar, &s.commitments).unwrap());
        }
    }

    #[test]
    fn seeds_are_unique_across_generations() {
        let a = gen(&MpcConfig::default());
        let b = gen(&MpcConfig::default());
        assert_ne!(a.master_seed, b.master_seed, "fresh OS entropy must be used");
        assert_ne!(a.public_key, b.public_key);
    }
}
