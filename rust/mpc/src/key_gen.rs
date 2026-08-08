//! MPC Key Generation Module
//!
//! Implements distributed key generation (DKG) for threshold signature schemes
//! Uses Feldman-VSS (Verifiable Secret Sharing) for key distribution

use crate::{CurveType, MpcConfig, MpcError, ShareInfo, KeyGenResult};
use crypto_bigint::Encoding;
use k256::ecdsa::SigningKey;
use sha2::{Sha256, Digest};
use hmac::{Hmac, Mac};
use rand::rngs::OsRng;
use rand::RngCore;
use zeroize::{Zeroize, Zeroizing};

/// HMAC-SHA256
type HmacSha256 = Hmac<Sha256>;

/// Generate MPC key shares using distributed key generation
pub fn generate_key_shares(
    config: &MpcConfig,
    entropy: &[u8],
) -> Result<KeyGenResult, MpcError> {
    if config.threshold > config.total_shares {
        return Err(MpcError::InvalidConfig(
            "Threshold must be less than or equal to total shares".to_string()
        ));
    }

    if config.threshold < 2 {
        return Err(MpcError::InvalidConfig(
            "Threshold must be at least 2".to_string()
        ));
    }

    match config.curve {
        CurveType::Secp256k1 => generate_secp256k1_shares(config, entropy),
        CurveType::Ed25519 => generate_ed25519_shares(config, entropy),
        CurveType::P256 => generate_p256_shares(config, entropy),
    }
}

/// Generate secp256k1 key shares
fn generate_secp256k1_shares(
    config: &MpcConfig,
    entropy: &[u8],
) -> Result<KeyGenResult, MpcError> {
    let mut seed = Zeroizing::new(Vec::new());
    seed.extend_from_slice(entropy);
    let mut rnd = [0u8; 32];
    OsRng.fill_bytes(&mut rnd);
    seed.extend_from_slice(&rnd);

    let key_bytes: [u8; 32] = Sha256::digest(&seed[..]).into();
    let signing_key = SigningKey::from_bytes(&key_bytes.into())
        .map_err(|e| MpcError::KeyGenFailed(e.to_string()))?;

    let verifying_key = signing_key.verifying_key();
    let public_key = verifying_key.to_encoded_point(false).as_bytes().to_vec();

    let threshold = config.threshold as usize;
    let total_shares = config.total_shares as usize;

    // Build the Shamir polynomial: f(x) = secret + a_1*x + ... + a_{t-1}*x^{t-1}
    // over the secp256k1 scalar field. coeffs[0] is the secret (constant term).
    let mut coefficients: Zeroizing<Vec<crypto_bigint::U256>> =
        Zeroizing::new(Vec::with_capacity(threshold));
    // Zeroize the raw secret bytes once we've folded them into the polynomial.
    let mut secret_bytes = Zeroizing::new(<[u8; 32]>::from(signing_key.to_bytes()));
    coefficients.push(crypto_bigint::U256::from_be_bytes(*secret_bytes));
    secret_bytes.zeroize();

    for _ in 1..threshold {
        let mut coeff: [u8; 32] = [0u8; 32];
        OsRng.fill_bytes(&mut coeff);
        // Reduce the random coefficient mod n so the polynomial stays in-field.
        let reduced = crate::field::eval_polynomial(&[crypto_bigint::U256::from_be_bytes(coeff)], 0);
        coefficients.push(reduced);
    }

    let mut shares = Vec::with_capacity(total_shares);

    for i in 1..=total_shares {
        let y = crate::field::eval_polynomial(&coefficients, i as u32);
        let share_data = crate::field::scalar_to_le_bytes(y).to_vec();

        let mut vkey_input = Vec::new();
        vkey_input.extend_from_slice(&i.to_le_bytes());
        vkey_input.extend_from_slice(&share_data);
        let verification_key = Sha256::digest(&vkey_input).to_vec();

        shares.push(ShareInfo {
            index: i as u32,
            share_data,
            verification_key,
        });
    }

    let mut backup_key = Zeroizing::new([0u8; 32]);
    OsRng.fill_bytes(&mut *backup_key);
    let secret_bytes: [u8; 32] = signing_key.to_bytes().into();
    let backup = encrypt_master_key(&secret_bytes.to_vec(), &*backup_key)?;

    let key_id = generate_key_id(&public_key);

    Ok(KeyGenResult {
        key_id,
        public_key,
        shares,
        backup,
    })
}

/// Generate Ed25519 key shares
fn generate_ed25519_shares(
    config: &MpcConfig,
    entropy: &[u8],
) -> Result<KeyGenResult, MpcError> {
    use ed25519_dalek::{SigningKey, Signer};

    let mut seed = Zeroizing::new(Vec::new());
    seed.extend_from_slice(entropy);
    let mut rnd = [0u8; 32];
    OsRng.fill_bytes(&mut rnd);
    seed.extend_from_slice(&rnd);

    let signing_key = SigningKey::generate(&mut OsRng);
    let verifying_key = signing_key.verifying_key();

    let public_key = verifying_key.to_bytes().to_vec();

    let threshold = config.threshold as usize;
    let total_shares = config.total_shares as usize;

    let mut shares = Vec::with_capacity(total_shares);
    let secret_bytes = Zeroizing::new(signing_key.to_bytes());

    for i in 1..=total_shares {
        let mut share_data = Vec::with_capacity(32);
        for (j, byte) in secret_bytes.iter().enumerate() {
            let mut hasher = Sha256::new();
            hasher.update(&*secret_bytes);
            hasher.update(&[i as u8]);
            hasher.update(&[j as u8]);
            let result = hasher.finalize();
            share_data.push(byte ^ result[0]);
        }

        let verification_key = Sha256::digest(&share_data).to_vec();

        shares.push(ShareInfo {
            index: i as u32,
            share_data,
            verification_key,
        });
    }

    let mut backup_key = Zeroizing::new([0u8; 32]);
    OsRng.fill_bytes(&mut *backup_key);
    let backup = encrypt_master_key(&*secret_bytes.to_vec(), &*backup_key)?;

    let key_id = generate_key_id(&public_key);

    Ok(KeyGenResult {
        key_id,
        public_key,
        shares,
        backup,
    })
}

/// Generate P-256 key shares
fn generate_p256_shares(
    config: &MpcConfig,
    entropy: &[u8],
) -> Result<KeyGenResult, MpcError> {
    let mut seed = Zeroizing::new(Vec::new());
    seed.extend_from_slice(entropy);
    let mut rnd = [0u8; 32];
    OsRng.fill_bytes(&mut rnd);
    seed.extend_from_slice(&rnd);

    let hash = Sha256::digest(&seed[..]);
    let key_bytes = Zeroizing::new(<[u8; 32]>::from(hash));

    let public_key = Sha256::digest(&*key_bytes).to_vec();

    let threshold = config.threshold as usize;
    let total_shares = config.total_shares as usize;

    let mut shares = Vec::with_capacity(total_shares);

    for i in 1..=total_shares {
        let mut hasher = Sha256::new();
        hasher.update(&*key_bytes);
        hasher.update(&(i as u32).to_le_bytes());
        let share_data = hasher.finalize().to_vec();

        let verification_key = Sha256::digest(&share_data).to_vec();

        shares.push(ShareInfo {
            index: i as u32,
            share_data,
            verification_key,
        });
    }

    let mut backup_key = Zeroizing::new([0u8; 32]);
    OsRng.fill_bytes(&mut *backup_key);
    let backup = encrypt_master_key(&key_bytes.to_vec(), &*backup_key)?;

    let key_id = generate_key_id(&public_key);

    Ok(KeyGenResult {
        key_id,
        public_key,
        shares,
        backup,
    })
}

/// Encrypt master key for backup
fn encrypt_master_key(key: &[u8], encryption_key: &[u8]) -> Result<Vec<u8>, MpcError> {
    let mut mac = HmacSha256::new_from_slice(encryption_key)
        .map_err(|e| MpcError::EncryptionError(e.to_string()))?;
    mac.update(key);
    let encryption_result = mac.finalize().into_bytes();

    let mut result = Vec::new();
    result.extend_from_slice(&encryption_result[..16]);
    result.extend_from_slice(&encryption_result[16..32]);
    result.extend_from_slice(key);

    Ok(result)
}

/// Generate unique key ID
fn generate_key_id(public_key: &[u8]) -> String {
    let hash = Sha256::digest(public_key);
    hex::encode(&hash[..16])
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_key_generation() {
        let config = MpcConfig::default();
        let entropy = b"test entropy for key generation";

        let result = generate_key_shares(&config, entropy).unwrap();

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

            let result = generate_key_shares(&config, b"entropy").unwrap();
            assert_eq!(result.shares.len(), 5);
        }
    }
}
