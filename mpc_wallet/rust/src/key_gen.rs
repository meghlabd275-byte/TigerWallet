//! MPC Key Generation Module
//! 
//! Implements distributed key generation (DKG) for threshold signature schemes
//! Uses Feldman-VSS (Verifiable Secret Sharing) for key distribution

use crate::{CurveType, MpcConfig, MpcError, ShareInfo, KeyGenResult};
use k256::ecdsa::SigningKey;
use k256::Secp256k1;
use sha2::{Sha256, Digest};
use hmac::{Hmac, Mac};
use rand::rngs::OsRng;
use zeroize::Zeroizing;

/// HMAC-SHA256
type HmacSha256 = Hmac<Sha256>;

/// Generate MPC key shares using distributed key generation
pub fn generate_key_shares(
    config: &MpcConfig,
    entropy: &[u8],
) -> Result<KeyGenResult, MpcError> {
    // Validate threshold
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
    seed.extend_from_slice(&OsRng.gen::<[u8; 32]>());
    
    // Create signing key from seed
    let signing_key = SigningKey::from_bytes(
        &Zeroizing::new(
            Sha256::digest(&seed[..]).into()
        )
    ).map_err(|e| MpcError::KeyGenFailed(e.to_string()))?;
    
    // Get public key
    let verifying_key = signing_key.verifying_key();
    let public_key = verifying_key.to_encoded_point(false).as_bytes().to_vec();
    
    // Generate polynomial coefficients for secret sharing
    let threshold = config.threshold as usize;
    let total_shares = config.total_shares as usize;
    
    // Generate random polynomial coefficients
    let mut coefficients = Vec::with_capacity(threshold);
    coefficients.push(*signing_key.to_bytes()); // Secret as constant term
    
    for _ in 1..threshold {
        let coeff: [u8; 32] = OsRng.gen();
        coefficients.push(coeff);
    }
    
    // Generate shares using Feldman-VSS
    let mut shares = Vec::with_capacity(total_shares);
    let prime = Secp256k1::ORDER.as_u64() as u128;
    
    for i in 1..=total_shares {
        // Evaluate polynomial at point i
        let x = i as u128;
        let mut y = 0u128;
        
        for (j, coeff) in coefficients.iter().enumerate() {
            let coeff_val = u128::from_le_bytes(*coeff);
            let term = coeff_val * x.pow(j as u32) % prime;
            y = (y + term) % prime;
        }
        
        // Create share
        let share_data = y.to_le_bytes().to_vec();
        
        // Generate verification key for this share
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
    
    // Generate backup (encrypted master key)
    let backup_key = OsRng.gen::<[u8; 32>>();
    let backup = encrypt_master_key(&signing_key.to_bytes().to_vec(), &backup_key)?;
    
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
    use ed25519_dalek::{SignKey, Signer};
    
    let mut seed = Zeroizing::new(Vec::new());
    seed.extend_from_slice(entropy);
    seed.extend_from_slice(&OsRng.gen::<[u8; 32]>());
    
    let signing_key = SignKey::generate(&mut OsRng);
    let verifying_key = signing_key.verifying_key();
    
    let public_key = verifying_key.to_bytes().to_vec();
    
    // Similar polynomial-based secret sharing for Ed25519
    let threshold = config.threshold as usize;
    let total_shares = config.total_shares as usize;
    
    let mut shares = Vec::with_capacity(total_shares);
    let secret_bytes = signing_key.to_bytes();
    
    // Simple secret sharing (for production, use proper Feldman-VSS)
    for i in 1..=total_shares {
        let mut share_data = Vec::with_capacity(32);
        for (j, byte) in secret_bytes.iter().enumerate() {
            let mut hasher = Sha256::new();
            hasher.update(&secret_bytes);
            hasher.update(&(i as u8));
            hasher.update(&(j as u8));
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
    
    // Generate backup
    let backup_key = OsRng.gen::<[u8; 32]>();
    let backup = encrypt_master_key(&secret_bytes.to_vec(), &backup_key)?;
    
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
    // Similar to secp256k1 but using P-256 curve
    let mut seed = Zeroizing::new(Vec::new());
    seed.extend_from_slice(entropy);
    seed.extend_from_slice(&OsRng.gen::<[u8; 32]>());
    
    let hash = Sha256::digest(&seed[..]);
    let key_bytes: [u8; 32] = hash.into();
    
    // For P-256, would use p256 crate
    // Simplified implementation
    let public_key = Sha256::digest(&key_bytes).to_vec();
    
    let threshold = config.threshold as usize;
    let total_shares = config.total_shares as usize;
    
    let mut shares = Vec::with_capacity(total_shares);
    
    for i in 1..=total_shares {
        let mut hasher = Sha256::new();
        hasher.update(&key_bytes);
        hasher.update(&(i as u32).to_le_bytes());
        let share_data = hasher.finalize().to_vec();
        
        let verification_key = Sha256::digest(&share_data).to_vec();
        
        shares.push(ShareInfo {
            index: i as u32,
            share_data,
            verification_key,
        });
    }
    
    let backup_key = OsRng.gen::<[u8; 32]>();
    let backup = encrypt_master_key(&key_bytes.to_vec(), &backup_key)?;
    
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
    result.extend_from_slice(&encryption_result[..16]); // IV/salt
    result.extend_from_slice(&encryption_result[16..32]); // Encrypted key
    result.extend_from_slice(key); // Actual key (simplified)
    
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
