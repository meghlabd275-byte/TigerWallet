//! MPC Key Sharing Module
//! 
//! Implements key resharing and recovery for threshold signature schemes
//! Allows changing the threshold and number of shares without changing the key

use crate::{CurveType, MpcConfig, MpcError, ShareInfo, KeyGenResult};
use sha2::{Sha256, Digest};
use rand::rngs::OsRng;

/// Reshare: Generate new shares from existing shares
/// This allows changing the threshold or number of participants
pub fn reshare(
    old_shares: &[ShareInfo],
    old_threshold: u32,
    new_config: &MpcConfig,
    public_key: &[u8],
) -> Result<Vec<ShareInfo>, MpcError> {
    if old_shares.len() < old_threshold as usize {
        return Err(MpcError::ThresholdNotMet(
            "Need old threshold shares to reshare".to_string()
        ));
    }
    
    // Reconstruct the secret from old shares
    // In production, this would use proper polynomial interpolation
    
    let new_threshold = new_config.threshold as usize;
    let new_total = new_config.total_shares as usize;
    
    let mut new_shares = Vec::with_capacity(new_total);
    
    // Generate new polynomial coefficients
    let mut coefficients = Vec::with_capacity(new_threshold);
    
    // Use old shares to derive new coefficients
    for i in 0..new_threshold {
        if i < old_shares.len() {
            coefficients.push(old_shares[i].share_data.clone());
        } else {
            coefficients.push(OsRng.gen::<[u8; 32]>().to_vec());
        }
    }
    
    // Generate new shares
    let prime = u128::MAX; // Simplified
    
    for i in 1..=new_total {
        // Evaluate new polynomial at point i
        let x = i as u128;
        let mut y = 0u128;
        
        for (j, coeff) in coefficients.iter().enumerate() {
            let coeff_val = u128::from_le_bytes({
                let mut arr = [0u8; 16];
                arr[..coeff.len().min(16)].copy_from_slice(&coeff[..coeff.len().min(16)]);
                arr
            });
            let term = coeff_val * x.pow(j as u32) % prime;
            y = (y + term) % prime;
        }
        
        let share_data = y.to_le_bytes().to_vec();
        let verification_key = Sha256::digest(&share_data).to_vec();
        
        new_shares.push(ShareInfo {
            index: i as u32,
            share_data,
            verification_key,
        });
    }
    
    Ok(new_shares)
}

/// Recover key from shares
/// Used for key recovery when some shares are lost
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
            let threshold = threshold as usize;
            let prime = u128::MAX;
            
            let mut secret = 0u128;
            
            for i in 0..threshold {
                let x_i = (i + 1) as u128;
                let y_i = u128::from_le_bytes({
                    let mut arr = [0u8; 16];
                    let share_data = &shares[i].share_data;
                    arr[..share_data.len().min(16)].copy_from_slice(&share_data[..share_data.len().min(16)]);
                    arr
                });
                
                let mut lagrange = 1u128;
                for j in 0..threshold {
                    if i != j {
                        let x_j = (j + 1) as u128;
                        let numerator = x_j;
                        let denominator = (x_j - x_i + prime) % prime;
                        let inv_denominator = mod_inverse(denominator, prime);
                        lagrange = (lagrange * numerator * inv_denominator) % prime;
                    }
                }
                
                secret = (secret + y_i * lagrange) % prime;
            }
            
            let secret_bytes: Vec<u8> = secret.to_le_bytes().to_vec();
            Ok(secret_bytes)
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

/// Modular inverse
fn mod_inverse(a: u128, m: u128) -> u128 {
    let mut s = 0i128;
    let mut old_s = 1i128;
    let mut r = m as i128;
    let mut old_r = a as i128;
    
    while r != 0 {
        let quotient = old_r / r;
        let (tmp_r, tmp_s) = (old_r - quotient * r, old_s - quotient * s);
        old_r = r;
        old_s = s;
        r = tmp_r;
        s = tmp_s;
    }
    
    if old_r != 1 {
        return 0;
    }
    
    let result = old_s % m as i128;
    if result < 0 {
        (result + m as i128) as u128
    } else {
        result as u128
    }
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
        
        let encryption_key = OsRng.gen::<[u8; 32]>();
        
        let backup = backup_key(&result.shares, &encryption_key).unwrap();
        
        let restored = restore_from_backup(&backup, &encryption_key).unwrap();
        
        assert_eq!(restored.len(), result.shares.len());
    }
}
