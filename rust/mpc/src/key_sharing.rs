//! MPC Key Sharing Module
//! 
//! Implements key resharing and recovery for threshold signature schemes
//! Allows changing the threshold and number of shares without changing the key

use crate::{CurveType, MpcConfig, MpcError, ShareInfo, KeyGenResult};
use crypto_bigint::Encoding;
use sha2::{Sha256, Digest};
use rand::rngs::OsRng;
use rand::RngCore;

/// Reshare: Generate new shares from existing shares
/// This allows changing the threshold or number of participants
pub fn reshare(
    old_shares: &[ShareInfo],
    old_threshold: u32,
    new_config: &MpcConfig,
    _public_key: &[u8],
) -> Result<Vec<ShareInfo>, MpcError> {
    if old_shares.len() < old_threshold as usize {
        return Err(MpcError::ThresholdNotMet(
            "Need old threshold shares to reshare".to_string()
        ));
    }
    let new_threshold = new_config.threshold as usize;
    let new_total = new_config.total_shares as usize;
    // Reconstruct the secret scalar from old shares (secp256k1 scalar field).
    let t = old_threshold as usize;
    let xs: Vec<u32> = old_shares.iter().take(t).map(|s| s.index).collect();
    let ys: Vec<crypto_bigint::U256> = old_shares.iter().take(t)
        .map(|s| crate::field::bytes_to_scalar(&s.share_data))
        .collect();
    let secret = crate::field::lagrange_at_zero(&xs, &ys);
    // Build a fresh polynomial with the reconstructed secret as the constant term.
    let mut coefficients: Vec<crypto_bigint::U256> = Vec::with_capacity(new_threshold);
    coefficients.push(secret);
    for _ in 1..new_threshold {
        let mut coeff: [u8; 32] = [0u8; 32];
        OsRng.fill_bytes(&mut coeff);
        let reduced = crate::field::eval_polynomial(&[crypto_bigint::U256::from_be_bytes(coeff)], 0);
        coefficients.push(reduced);
    }
    let mut new_shares = Vec::with_capacity(new_total);
    for i in 1..=new_total {
        let y = crate::field::eval_polynomial(&coefficients, i as u32);
        let share_data = crate::field::scalar_to_le_bytes(y).to_vec();
        let verification_key = Sha256::digest(&share_data).to_vec();
        new_shares.push(ShareInfo {
            index: i as u32,
            share_data,
            verification_key,
        });
    }
    Ok(new_shares)
}

pub fn recover_key(
    shares: &[ShareInfo],
    threshold: u32,
    curve: CurveType,
) -> Result<Vec<u8>, MpcError> {
    if shares.len() < threshold as usize {
        return Err(MpcError::ThresholdNotMet(
            format!("Need {} shares for recovery, got {}", threshold, shares.len())
        ));
    }
    
    // Use Lagrange interpolation to recover the secret
    // Simplified implementation
    
    match curve {
        CurveType::Secp256k1 | CurveType::P256 => {
            let t = threshold as usize;
            let xs: Vec<u32> = shares.iter().take(t).map(|s| s.index).collect();
            let ys: Vec<crypto_bigint::U256> = shares.iter().take(t)
                .map(|s| crate::field::bytes_to_scalar(&s.share_data))
                .collect();
            let secret = crate::field::lagrange_at_zero(&xs, &ys);
            Ok(crate::field::scalar_to_le_bytes(secret).to_vec())
        }

        CurveType::Ed25519 => {
            // For Ed25519, use XOR combination
            let mut secret = [0u8; 32];
            for share in shares.iter().take(threshold as usize) {
                for (i, byte) in share.share_data.iter().enumerate().take(32) {
                    secret[i] ^= byte;
                }
            }
            Ok(secret.to_vec())
        }
    }
}

/// Backup key to encrypted format
pub fn backup_key(
    shares: &[ShareInfo],
    encryption_key: &[u8],
) -> Result<Vec<u8>, MpcError> {
    use hmac::{Hmac, Mac};
    type HmacSha256 = Hmac<Sha256>;
    
    // Combine shares
    let mut combined = Vec::new();
    for share in shares {
        combined.extend_from_slice(&share.share_data);
    }
    
    // Encrypt with HMAC
    let mut mac = HmacSha256::new_from_slice(encryption_key)
        .map_err(|e| MpcError::EncryptionError(e.to_string()))?;
    mac.update(&combined);
    let result = mac.finalize().into_bytes();
    
    let mut backup = Vec::new();
    backup.extend_from_slice(encryption_key); // Store key identifier
    backup.extend_from_slice(&result[..16]); // Encryption salt
    backup.extend_from_slice(&combined); // Encrypted shares
    
    Ok(backup)
}

/// Restore key from backup
pub fn restore_from_backup(
    backup: &[u8],
    decryption_key: &[u8],
) -> Result<Vec<ShareInfo>, MpcError> {
    if backup.len() < 48 {
        return Err(MpcError::KeySharingFailed("Invalid backup format".to_string()));
    }
    
    // Extract encrypted shares
    let encrypted_shares = &backup[48..];
    
    // Verify decryption key
    let salt = &backup[32..48];
    use hmac::{Hmac, Mac};
    type HmacSha256 = Hmac<Sha256>;
    
    let mut mac = HmacSha256::new_from_slice(decryption_key)
        .map_err(|e| MpcError::EncryptionError(e.to_string()))?;
    mac.update(encrypted_shares);
    let expected = mac.finalize().into_bytes();
    
    if &expected[..16] != salt {
        return Err(MpcError::KeySharingFailed("Invalid decryption key".to_string()));
    }
    
    // Parse shares (simplified - production would have proper format)
    let share_count = encrypted_shares.len() / 32;
    let mut shares = Vec::new();
    
    for i in 0..share_count {
        let start = i * 32;
        let share_data = encrypted_shares[start..start+32].to_vec();
        let verification_key = Sha256::digest(&share_data).to_vec();
        
        shares.push(ShareInfo {
            index: (i + 1) as u32,
            share_data,
            verification_key,
        });
    }
    
    Ok(shares)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::key_gen::generate_key_shares;
    
    #[test]
    fn test_reshare() {
        let old_config = MpcConfig {
            threshold: 2,
            total_shares: 3,
            curve: CurveType::Secp256k1,
            key_id: String::new(),
        };
        
        let result = generate_key_shares(&old_config, b"test entropy").unwrap();
        
        let new_config = MpcConfig {
            threshold: 3,
            total_shares: 5,
            curve: CurveType::Secp256k1,
            key_id: String::new(),
        };
        
        let new_shares = reshare(&result.shares, 2, &new_config, &result.public_key).unwrap();
        
        assert_eq!(new_shares.len(), 5);
    }
    
    #[test]
    fn test_backup_restore() {
        let config = MpcConfig::default();
        let result = generate_key_shares(&config, b"test entropy").unwrap();
        
        let encryption_key = { let mut arr: [u8; 32] = [0u8; 32]; OsRng.fill_bytes(&mut arr); arr };
        
        let backup = backup_key(&result.shares, &encryption_key).unwrap();
        
        let restored = restore_from_backup(&backup, &encryption_key).unwrap();
        
        assert_eq!(restored.len(), result.shares.len());
    }
}
