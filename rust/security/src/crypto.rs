//! Cryptography facade for the security modules.
//!
//! The modules in this crate were written against a `crypto` sub-namespace that
//! was never committed. This module provides it with real primitives only.

pub mod digest {
    pub use crate::digest::{keccak256, ripemd160, sha256, Digest, Sha256};
}

pub mod secure_random {
    pub use crate::secure_random::{
        fill_random, generate_random_bytes, generate_random_hex, RandomError,
    };
}

pub mod hmac {
    use hmac::{Hmac, Mac};
    use sha2::Sha256;

    /// HMAC-SHA256 of `data` under `key`.
    ///
    /// This is a MAC, not a signature: it can only be verified by a party that
    /// holds the same key.
    pub fn hmac_sha256(key: &[u8], data: &[u8]) -> Result<Vec<u8>, String> {
        let mut mac =
            Hmac::<Sha256>::new_from_slice(key).map_err(|e| format!("invalid HMAC key: {e}"))?;
        mac.update(data);
        Ok(mac.finalize().into_bytes().to_vec())
    }
}

pub mod aead {
    //! AES-256-GCM used to protect share material at rest.

    use aes_gcm::aead::{Aead, KeyInit, Payload};
    use aes_gcm::{Aes256Gcm, Nonce};

    const NONCE_LEN: usize = 12;

    /// Encrypt `plaintext`, returning `nonce || ciphertext || tag`.
    pub fn encrypt(key: &[u8; 32], plaintext: &[u8], aad: &[u8]) -> Result<Vec<u8>, String> {
        let cipher = Aes256Gcm::new_from_slice(key).map_err(|e| format!("invalid key: {e}"))?;
        let mut nonce_bytes = [0u8; NONCE_LEN];
        crate::secure_random::fill_random(&mut nonce_bytes)
            .map_err(|e| format!("secure random unavailable: {e}"))?;

        let ciphertext = cipher
            .encrypt(
                Nonce::from_slice(&nonce_bytes),
                Payload {
                    msg: plaintext,
                    aad,
                },
            )
            .map_err(|e| format!("encryption failed: {e}"))?;

        let mut out = Vec::with_capacity(NONCE_LEN + ciphertext.len());
        out.extend_from_slice(&nonce_bytes);
        out.extend_from_slice(&ciphertext);
        Ok(out)
    }

    /// Decrypt output produced by [`encrypt`].
    pub fn decrypt(key: &[u8; 32], sealed: &[u8], aad: &[u8]) -> Result<Vec<u8>, String> {
        if sealed.len() <= NONCE_LEN {
            return Err("ciphertext is too short to contain a nonce".to_string());
        }

        let (nonce_bytes, ciphertext) = sealed.split_at(NONCE_LEN);
        let cipher = Aes256Gcm::new_from_slice(key).map_err(|e| format!("invalid key: {e}"))?;

        cipher
            .decrypt(
                Nonce::from_slice(nonce_bytes),
                Payload {
                    msg: ciphertext,
                    aad,
                },
            )
            .map_err(|_| {
                "decryption failed: wrong key, wrong associated data, or tampered ciphertext"
                    .to_string()
            })
    }

    #[cfg(test)]
    mod tests {
        use super::*;

        #[test]
        fn round_trip() {
            let key = [7u8; 32];
            let sealed = encrypt(&key, b"share material", b"key-1").unwrap();
            assert_ne!(&sealed[12..], b"share material");
            assert_eq!(decrypt(&key, &sealed, b"key-1").unwrap(), b"share material");
        }

        #[test]
        fn nonce_is_unique_per_call() {
            let key = [7u8; 32];
            let a = encrypt(&key, b"same", b"").unwrap();
            let b = encrypt(&key, b"same", b"").unwrap();
            assert_ne!(a, b);
        }

        #[test]
        fn tampering_is_rejected() {
            let key = [7u8; 32];
            let mut sealed = encrypt(&key, b"share material", b"key-1").unwrap();
            let last = sealed.len() - 1;
            sealed[last] ^= 0x01;
            assert!(decrypt(&key, &sealed, b"key-1").is_err());
        }

        #[test]
        fn wrong_aad_is_rejected() {
            let key = [7u8; 32];
            let sealed = encrypt(&key, b"share material", b"key-1").unwrap();
            assert!(decrypt(&key, &sealed, b"key-2").is_err());
        }
    }
}

pub use digest::Digest as CryptoDigest;
