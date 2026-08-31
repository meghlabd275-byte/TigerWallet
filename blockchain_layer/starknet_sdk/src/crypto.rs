//! Starknet Cryptography
//!
//! Real Stark-curve ECDSA via the audited `starknet-crypto` crate (sign/verify),
//! plus Blake2 address derivation helpers. No fabricated signature math.

use starknet_crypto::{
    get_public_key as sn_get_public_key, sign as sn_sign, verify as sn_verify,
    FieldElement, Signature as SnSignature,
};

/// ECDSA signature on the Stark curve (r, s field elements).
#[derive(Debug, Clone)]
pub struct Signature {
    pub r: FieldElement,
    pub s: FieldElement,
}

impl From<SnSignature> for Signature {
    fn from(sig: SnSignature) -> Self {
        Self { r: sig.r, s: sig.s }
    }
}
use std::fmt;

/// Starknet Key Pair (raw 32-byte scalars wrapping the Stark field element).
pub struct KeyPair {
    private_key: FieldElement,
    public_key: FieldElement,
}

impl KeyPair {
    /// Generate a random key pair.
    pub fn generate() -> Result<Self, CryptoError> {
        use rand::RngCore;
        let mut bytes = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut bytes);
        Self::from_private_key(bytes)
    }

    /// Recover from raw 32-byte private key scalar.
    pub fn from_private_key(private_key: [u8; 32]) -> Result<Self, CryptoError> {
        let fe = FieldElement::from_bytes_be(&private_key)
            .map_err(|_| CryptoError::InvalidPrivateKey)?;
        if fe == FieldElement::ZERO {
            return Err(CryptoError::InvalidPrivateKey);
        }
        let public = sn_get_public_key(&fe);
        Ok(Self { private_key: fe, public_key: public })
    }

    /// Public key as raw 32 bytes.
    pub fn private_key(&self) -> [u8; 32] {
        self.private_key.to_bytes_be()
    }

    /// Public key as raw 32 bytes.
    pub fn public_key(&self) -> [u8; 32] {
        self.public_key.to_bytes_be()
    }

    /// Hex string of the private key (for keystore export only).
    pub fn private_key_hex(&self) -> String {
        hex::encode(self.private_key.to_bytes_be())
    }

    /// Hex string of the public key.
    pub fn public_key_hex(&self) -> String {
        hex::encode(self.public_key.to_bytes_be())
    }

    /// Sign a 32-byte message hash with RFC-6979 deterministic k.
    pub fn sign(&self, message: &[u8]) -> Result<Signature, CryptoError> {
        if message.len() != 32 {
            return Err(CryptoError::InvalidMessageHash);
        }
        let arr: [u8; 32] = message.try_into().map_err(|_| CryptoError::InvalidMessageHash)?;
        let msg = FieldElement::from_bytes_be(&arr)
            .map_err(|_| CryptoError::InvalidMessageHash)?;
        let k = starknet_crypto::rfc6979_generate_k(&msg, &self.private_key, None);
        sn_sign(&self.private_key, &msg, &k)
            .map(|sig| sig.into())
            .map_err(|_| CryptoError::SigningFailed)
    }

    /// Verify a signature over a 32-byte message hash.
    pub fn verify(&self, message: &[u8], signature: &Signature) -> bool {
        if message.len() != 32 {
            return false;
        }
        let arr: [u8; 32] = match message.try_into() {
            Ok(a) => a,
            Err(_) => return false,
        };
        let msg = match FieldElement::from_bytes_be(&arr) {
            Ok(f) => f,
            Err(_) => return false,
        };
        sn_verify(&self.public_key, &msg, &signature.r, &signature.s).unwrap_or(false)
    }

    /// Access the Stark field element scalar (server-side key material only).
    pub(crate) fn private_fe(&self) -> &FieldElement {
        &self.private_key
    }
}

/// Crypto errors — fail-closed, nothing is fabricated.
#[derive(Debug, Clone)]
pub enum CryptoError {
    InvalidPrivateKey,
    InvalidPublicKey,
    InvalidSignature,
    InvalidMessageHash,
    SigningFailed,
    VerificationFailed,
}

impl fmt::Display for CryptoError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            CryptoError::InvalidPrivateKey => write!(f, "Invalid private key"),
            CryptoError::InvalidPublicKey => write!(f, "Invalid public key"),
            CryptoError::InvalidSignature => write!(f, "Invalid signature"),
            CryptoError::InvalidMessageHash => write!(f, "Invalid message hash"),
            CryptoError::SigningFailed => write!(f, "Signing failed"),
            CryptoError::VerificationFailed => write!(f, "Signature verification failed"),
        }
    }
}

impl std::error::Error for CryptoError {}

/// Derive an account address from a public key (Blake2-256).
pub fn derive_account_address(public_key: &[u8; 32]) -> [u8; 32] {
    use blake2::{Blake2s256, Digest};
    let mut hasher = Blake2s256::new();
    hasher.update(public_key);
    let result = hasher.finalize();
    let mut out = [0u8; 32];
    out.copy_from_slice(&result);
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sign_and_verify_roundtrip() {
        let kp = KeyPair::from_private_key([1u8; 32]).unwrap();
        let message = [2u8; 32];
        let sig = kp.sign(&message).unwrap();
        assert!(kp.verify(&message, &sig));
        assert!(!kp.verify(&[3u8; 32], &sig));
    }

    #[test]
    fn test_rejects_wrong_message_length() {
        let kp = KeyPair::from_private_key([1u8; 32]).unwrap();
        assert!(matches!(
            kp.sign(b"short").unwrap_err(),
            CryptoError::InvalidMessageHash
        ));
    }

    #[test]
    fn test_deterministic_signature() {
        let kp = KeyPair::from_private_key([4u8; 32]).unwrap();
        let s1 = kp.sign(&[1u8; 32]).unwrap();
        let s2 = kp.sign(&[1u8; 32]).unwrap();
        assert_eq!(s1.r, s2.r);
        assert_eq!(s1.s, s2.s);
    }

    #[test]
    fn test_address_derivation_is_deterministic() {
        let addr1 = derive_account_address(&[1u8; 32]);
        let addr2 = derive_account_address(&[1u8; 32]);
        assert_eq!(addr1, addr2);
    }
}
