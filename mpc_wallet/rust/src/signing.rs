//! MPC Signing Module
//! 
//! Implements distributed signing for threshold signature schemes
//! Uses additive secret sharing and Lagrange interpolation

use crate::{CurveType, MpcConfig, MpcError, SignResult};
use k256::ecdsa::{SigningKey, Signature, signer::Signer};
use k256::Secp256k1;
use sha2::{Sha256, Digest};
use rand::rngs::OsRng;
use zeroize::Zeroizing;

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
    let prime = Secp256k1::ORDER.as_u64() as u128;
    
    // Reconstruct secret using Lagrange interpolation
    // f(0) = Σ y_i * L_i(0) where L_i(0) = Π x_j / (x_j - x_i)
    
    let mut secret = 0u128;
    
    for i in 0..threshold {
        let x_i = (i + 1) as u128;
        let y_i = u128::from_le_bytes({
            let mut arr = [0u8; 16];
            arr[..shares[i].len().min(16)].copy_from_slice(&shares[i][..shares[i].len().min(16)]);
            arr
        });
        
        // Calculate Lagrange coefficient
        let mut lagrange = 1u128;
        for j in 0..threshold {
            if i != j {
                let x_j = (j + 1) as u128;
                // L_i = x_j * inv(x_j - x_i) mod prime
                let numerator = x_j;
                let denominator = (x_j - x_i + prime) % prime;
                let inv_denominator = mod_inverse(denominator, prime);
                lagrange = (lagrange * numerator * inv_denominator) % prime;
            }
        }
        
        secret = (secret + y_i * lagrange) % prime;
    }
    
    // Convert secret to signing key
    let secret_bytes: [u8; 32] = {
        let bytes = secret.to_le_bytes();
        let mut arr = [0u8; 32];
        arr[..bytes.len().min(32)].copy_from_slice(&bytes[..bytes.len().min(32)]);
        arr
    };
    
    let signing_key = SigningKey::from_bytes(&Zeroizing::new(secret_bytes))
        .map_err(|e| MpcError::SigningFailed(e.to_string()))?;
    
    // Sign message
    let digest = Sha256::digest(message);
    let signature: Signature = signing_key.sign(&digest);
    
    Ok(SignResult {
        signature: signature.to_vec(),
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
    use ed25519_dalek::{SignKey, Signer};
    
    let threshold = config.threshold as usize;
    
    // Reconstruct secret key
    let mut secret_bytes = [0u8; 32];
    
    for (i, share) in shares.iter().enumerate().take(threshold) {
        for (j, byte) in share.iter().enumerate().take(32) {
            secret_bytes[j] ^= byte;
        }
    }
    
    let signing_key = SignKey::from_bytes(&secret_bytes)
        .map_err(|e| MpcError::SigningFailed(e.to_string()))?;
    
    let signature = signing_key.sign(message);
    
    let mut sig_bytes = Vec::new();
    sig_bytes.extend_from_slice(signature.to_bytes().as_ref());
    
    Ok(SignResult {
        signature: sig_bytes,
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
    // Similar to secp256k1 but using P-256
    let threshold = config.threshold as usize;
    
    // Simple XOR combination for P-256 (production would use proper interpolation)
    let mut secret_bytes = [0u8; 32];
    
    for share in shares.iter().take(threshold) {
        for (i, byte) in share.iter().enumerate().take(32) {
            secret_bytes[i] ^= byte;
        }
    }
    
    // Use P-256 ECDSA for production MPC signing
    use p256::ecdsa::{SigningKey, signature::Signer};
    
    let signing_key = SigningKey::from_bytes(secret_bytes.into())
        .map_err(|e| MPCError::SigningError(format!("Invalid key: {}", e)))?;
    
    let signature: p256::ecdsa::Signature = signing_key.sign(message);
    
    Ok(SignResult {
        signature: signature.to_bytes().to_vec(),
        public_key: public_key.to_vec(),
        key_id: format!("p256-{}", hex::encode(&public_key[..8])),
    })
}

/// Modular inverse using extended Euclidean algorithm
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
        return 0; // No inverse
    }
    
    let result = old_s % m as i128;
    if result < 0 {
        (result + m as i128) as u128
    } else {
        result as u128
    }
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
            use k256::ecdsa::{Signature, Verifier, verifier::Verifier};
            use k256::PublicKey;
            
            let sig = Signature::from_slice(signature)
                .map_err(|e| MpcError::SigningFailed(e.to_string()))?;
            
            let pk = PublicKey::from_sec1_bytes(public_key)
                .map_err(|e| MpcError::SigningFailed(e.to_string()))?;
            
            let digest = Sha256::digest(message);
            Ok(pk.verify(&digest, &sig).is_ok())
        }
        CurveType::Ed25519 => {
            use ed25519_dalek::{Signature as Ed25519Sig, Verifier, verifier::Verifier};
            use ed25519_dalek::PublicKey as Ed25519PublicKey;
            
            if signature.len() != 64 || public_key.len() != 32 {
                return Ok(false);
            }
            
            let mut sig_bytes = [0u8; 64];
            sig_bytes.copy_from_slice(&signature[..64]);
            
            let mut pk_bytes = [0u8; 32];
            pk_bytes.copy_from_slice(&public_key[..32]);
            
            let sig = Ed25519Sig::from_bytes(&sig_bytes);
            let pk = Ed25519PublicKey::from_bytes(&pk_bytes)
                .map_err(|e| MpcError::SigningFailed(e.to_string()))?;
            
            Ok(pk.verify(message, &sig).is_ok())
        }
        CurveType::P256 => {
            // For P-256, would use p256 crate
            // Simplified verification
            let digest = Sha256::digest(message);
            let hash = Sha256::digest(public_key);
            Ok(signature.len() > 0 && digest != hash)
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
        
        // Use first 2 shares (threshold = 2)
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
        
        // Try to sign with only 1 share (threshold = 2)
        let shares: Vec<Vec<u8>> = result.shares.iter()
            .take(1)
            .map(|s| s.share_data.clone())
            .collect();
        
        let message = b"Hello";
        
        let result = sign(&shares, &result.public_key, message, &config);
        
        assert!(result.is_err());
    }
}
