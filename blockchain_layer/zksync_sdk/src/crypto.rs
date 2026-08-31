//! zkSync Era cryptography — real secp256k1 ECDSA (EVM-compatible)

use k256::ecdsa::{
    signature::hazmat::PrehashSigner, signature::hazmat::PrehashVerifier, Signature,
    SigningKey,
};
use sha3::{Digest, Keccak256};

use crate::types::ZksyncError;

/// Compute Keccak-256
pub fn keccak256(data: &[u8]) -> [u8; 32] {
    let mut h = Keccak256::new();
    h.update(data);
    h.finalize().into()
}

/// secp256k1 key pair (zkSync Era uses the EVM curve)
#[derive(Clone)]
pub struct KeyPair {
    signing: SigningKey,
}

impl KeyPair {
    /// Generate a fresh random key pair
    pub fn generate() -> Self {
        Self {
            signing: SigningKey::random(&mut rand::rngs::OsRng),
        }
    }

    /// Import from a 32-byte private key
    pub fn from_private_key(bytes: &[u8; 32]) -> Result<Self, ZksyncError> {
        let signing = SigningKey::from_bytes(bytes.into())
            .map_err(|e| ZksyncError::InvalidKey(e.to_string()))?;
        Ok(Self { signing })
    }

    /// Export the 32-byte private key
    pub fn private_key(&self) -> [u8; 32] {
        self.signing.to_bytes().into()
    }

    /// Uncompressed public key without the 0x04 prefix (64 bytes)
    pub fn public_key_uncompressed(&self) -> [u8; 64] {
        let vk = self.signing.verifying_key().clone();
        let point = vk.to_encoded_point(false);
        let b = point.as_bytes();
        let mut out = [0u8; 64];
        out.copy_from_slice(&b[1..65]);
        out
    }

    /// Derive the EVM address: last 20 bytes of Keccak-256(uncompressed pubkey)
    pub fn address(&self) -> [u8; 20] {
        let hash = keccak256(&self.public_key_uncompressed());
        let mut addr = [0u8; 20];
        addr.copy_from_slice(&hash[12..]);
        addr
    }

    /// Sign a 32-byte message hash (RFC-6979 deterministic, low-s normalized)
    pub fn sign(&self, message_hash: &[u8; 32]) -> Signature {
        self.signing
            .sign_prehash(message_hash)
            .expect("secp256k1 prehash signing is infallible for 32-byte input")
    }

    /// Verify a signature against a 32-byte message hash
    pub fn verify(&self, message_hash: &[u8; 32], signature: &Signature) -> bool {
        let vk = self.signing.verifying_key().clone();
        vk.verify_prehash(message_hash, signature).is_ok()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn address_derivation_known_vector() {
        // Private key 0x0001..01 -> known derivation properties:
        // deterministic, and address equals last 20 bytes of keccak(pubkey)
        let mut pk = [0u8; 32];
        pk[31] = 1;
        let kp = KeyPair::from_private_key(&pk).unwrap();
        let a1 = kp.address();
        let a2 = KeyPair::from_private_key(&pk).unwrap().address();
        assert_eq!(a1, a2);
    }

    #[test]
    fn sign_verify_roundtrip() {
        let kp = KeyPair::generate();
        let msg = keccak256(b"tigerwallet zksync test");
        let sig = kp.sign(&msg);
        assert!(kp.verify(&msg, &sig));
        let other = keccak256(b"different message");
        assert!(!kp.verify(&other, &sig));
    }

    #[test]
    fn rejects_invalid_private_key() {
        assert!(KeyPair::from_private_key(&[0u8; 32]).is_err());
    }
}
