//! Authenticated encryption for key-share backups.
//!
//! Key material is encrypted with AES-256-GCM under a key derived from the
//! operator passphrase with PBKDF2-HMAC-SHA256. GCM is an AEAD: decryption
//! fails if the ciphertext, nonce or associated data was altered, so a tampered
//! backup is rejected rather than silently producing garbage key material.
//!
//! The previous implementation used HMAC as if it were encryption: it appended
//! the plaintext shares verbatim and "authenticated" them with an HMAC keyed by
//! the same bytes it stored in the header. Anyone holding a backup could read
//! the shares, and could also forge a new tag, so it provided neither
//! confidentiality nor integrity.

use aes_gcm::aead::{Aead, KeyInit, Payload};
use aes_gcm::{Aes256Gcm, Nonce};
use pbkdf2::pbkdf2_hmac;
use rand::rngs::OsRng;
use rand::RngCore;
use sha2::Sha256;
use zeroize::Zeroizing;

use crate::MpcError;

/// Iteration count for PBKDF2-HMAC-SHA256. Matches the strength used elsewhere
/// in the repository for passphrase-derived key material.
#[cfg(not(test))]
const PBKDF2_ROUNDS: u32 = 600_000;
/// Tests exercise the KDF's correctness, not its cost, and the suite would
/// otherwise spend minutes in PBKDF2. Production keeps the full count above.
#[cfg(test)]
const PBKDF2_ROUNDS: u32 = 1_000;
const SALT_LEN: usize = 32;
const NONCE_LEN: usize = 12;
const TAG_LEN: usize = 16;
/// magic || version || salt || nonce || ciphertext+tag
const MAGIC: &[u8; 4] = b"TMPC";
const VERSION: u8 = 1;
const HEADER_LEN: usize = 4 + 1 + SALT_LEN + NONCE_LEN;
const MIN_BLOB_LEN: usize = HEADER_LEN + TAG_LEN;

/// Derive a 256-bit key from a passphrase and salt.
fn derive_key(passphrase: &[u8], salt: &[u8]) -> Zeroizing<[u8; 32]> {
    let mut key = Zeroizing::new([0u8; 32]);
    pbkdf2_hmac::<Sha256>(passphrase, salt, PBKDF2_ROUNDS, &mut *key);
    key
}

/// Encrypt `plaintext` under `passphrase`, binding `aad` as associated data.
///
/// `aad` is authenticated but not encrypted; callers pass the key id and share
/// index so a blob cannot be replayed against a different key or participant.
pub fn encrypt(passphrase: &[u8], aad: &[u8], plaintext: &[u8]) -> Result<Vec<u8>, MpcError> {
    let mut salt = [0u8; SALT_LEN];
    let mut nonce_bytes = [0u8; NONCE_LEN];
    OsRng.fill_bytes(&mut salt);
    OsRng.fill_bytes(&mut nonce_bytes);

    let key = derive_key(passphrase, &salt);
    let cipher = Aes256Gcm::new_from_slice(&*key)
        .map_err(|e| MpcError::EncryptionError(e.to_string()))?;

    let ciphertext = cipher
        .encrypt(
            &Nonce::from(nonce_bytes),
            Payload { msg: plaintext, aad },
        )
        .map_err(|_| MpcError::EncryptionError("share encryption failed".to_string()))?;

    let mut out = Vec::with_capacity(HEADER_LEN + ciphertext.len());
    out.extend_from_slice(MAGIC);
    out.push(VERSION);
    out.extend_from_slice(&salt);
    out.extend_from_slice(&nonce_bytes);
    out.extend_from_slice(&ciphertext);
    Ok(out)
}

/// Decrypt a blob produced by [`encrypt`]. Fails closed on any tampering.
pub fn decrypt(passphrase: &[u8], aad: &[u8], blob: &[u8]) -> Result<Zeroizing<Vec<u8>>, MpcError> {
    if blob.len() < MIN_BLOB_LEN {
        return Err(MpcError::EncryptionError("backup blob is truncated".to_string()));
    }
    if &blob[..4] != MAGIC {
        return Err(MpcError::EncryptionError("unrecognised backup format".to_string()));
    }
    if blob[4] != VERSION {
        return Err(MpcError::EncryptionError(format!(
            "unsupported backup version {}",
            blob[4]
        )));
    }

    let salt = &blob[5..5 + SALT_LEN];
    let mut nonce_bytes = [0u8; NONCE_LEN];
    nonce_bytes.copy_from_slice(&blob[5 + SALT_LEN..HEADER_LEN]);
    let ciphertext = &blob[HEADER_LEN..];

    let key = derive_key(passphrase, salt);
    let cipher = Aes256Gcm::new_from_slice(&*key)
        .map_err(|e| MpcError::EncryptionError(e.to_string()))?;

    let plaintext = cipher
        .decrypt(
            &Nonce::from(nonce_bytes),
            Payload { msg: ciphertext, aad },
        )
        .map_err(|_| {
            MpcError::EncryptionError(
                "backup authentication failed: wrong passphrase or tampered data".to_string(),
            )
        })?;

    Ok(Zeroizing::new(plaintext))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn round_trip() {
        let pass = b"correct horse battery staple";
        let aad = b"keyid:1";
        let secret = b"threshold share material";

        let blob = encrypt(pass, aad, secret).unwrap();
        let out = decrypt(pass, aad, &blob).unwrap();
        assert_eq!(&out[..], secret);
    }

    #[test]
    fn ciphertext_does_not_contain_plaintext() {
        let secret = b"UNIQUEMARKERSHAREBYTES";
        let blob = encrypt(b"pw", b"aad", secret).unwrap();
        assert!(
            !blob.windows(secret.len()).any(|w| w == secret),
            "the plaintext must not appear anywhere in the backup blob"
        );
    }

    #[test]
    fn wrong_passphrase_fails_closed() {
        let blob = encrypt(b"right", b"aad", b"secret").unwrap();
        assert!(decrypt(b"wrong", b"aad", &blob).is_err());
    }

    #[test]
    fn tampered_ciphertext_fails_closed() {
        let mut blob = encrypt(b"pw", b"aad", b"secret material here").unwrap();
        let last = blob.len() - 1;
        blob[last] ^= 0x01;
        assert!(decrypt(b"pw", b"aad", &blob).is_err());
    }

    #[test]
    fn tampered_header_fails_closed() {
        let mut blob = encrypt(b"pw", b"aad", b"secret").unwrap();
        blob[10] ^= 0x01; // flip a salt bit
        assert!(decrypt(b"pw", b"aad", &blob).is_err());
    }

    #[test]
    fn mismatched_aad_fails_closed() {
        let blob = encrypt(b"pw", b"keyid:1", b"secret").unwrap();
        assert!(
            decrypt(b"pw", b"keyid:2", &blob).is_err(),
            "a blob must not decrypt under different associated data"
        );
    }

    #[test]
    fn each_encryption_uses_a_fresh_nonce_and_salt() {
        let a = encrypt(b"pw", b"aad", b"same plaintext").unwrap();
        let b = encrypt(b"pw", b"aad", b"same plaintext").unwrap();
        assert_ne!(a, b, "nonce/salt reuse would break GCM security");
    }

    #[test]
    fn truncated_blob_fails_closed() {
        let blob = encrypt(b"pw", b"aad", b"secret").unwrap();
        assert!(decrypt(b"pw", b"aad", &blob[..blob.len() / 2]).is_err());
    }
}
