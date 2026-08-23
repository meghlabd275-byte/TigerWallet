//! crypto.rs — real cryptography for the Self-Hosted Master Wallet.
//!
//! Port of master_wallet/backend/crypto_core.go + hd_derive.go:
//!   * BIP-39 mnemonic generation (24 words, 256-bit entropy) + validation
//!   * BIP-39 seed derivation (PBKDF2-HMAC-SHA512, 2048 rounds)
//!   * BIP-32 CKD for secp256k1 + BIP-44 path derivation (m/44'/60'/0'/0/n)
//!   * EVM address derivation (keccak256 of uncompressed pubkey, EIP-55)
//!   * Seed encryption at rest: scrypt KDF + AES-256-GCM, constant-time compare

use aes_gcm::aead::{Aead, KeyInit};
use aes_gcm::Aes256Gcm;
use bip39::{Language, Mnemonic};
use hmac::{Hmac, Mac};
use k256::elliptic_curve::sec1::ToEncodedPoint;
use k256::elliptic_curve::PrimeField;
use k256::{PublicKey, Scalar, SecretKey};
use sha2::{Digest, Sha512};
use sha3::Keccak256;

/// Same scrypt parameters as the canonical Go backend (N=2^18, r=8, p=1).
const SCRYPT_LOG_N: u8 = 18;
const SCRYPT_R: u32 = 8;
const SCRYPT_P: u32 = 1;
const SCRYPT_SALT_LEN: usize = 32;
const AES_GCM_NONCE_LEN: usize = 12;
const KEY_LEN: usize = 32;

// ---------------------------------------------------------------------------
// BIP-39
// ---------------------------------------------------------------------------

/// GenerateMnemonic — 24 words (256-bit entropy) from the OS CSPRNG.
pub fn generate_mnemonic() -> String {
    let mut rng = rand::thread_rng();
    let m = Mnemonic::generate_in_with(&mut rng, Language::English, 24)
        .expect("24-word BIP-39 generation is infallible");
    m.to_string()
}

/// ValidateMnemonic — full BIP-39 validation (wordlist membership + checksum).
pub fn validate_mnemonic(mnemonic: &str) -> bool {
    Mnemonic::parse(mnemonic.trim()).is_ok()
}

/// MnemonicToSeed — BIP-39 PBKDF2-HMAC-SHA512 (2048 rounds) → 64-byte seed.
/// Passphrase is always "" (matches the canonical Go backend).
pub fn mnemonic_to_seed(mnemonic: &str) -> [u8; 64] {
    let m = Mnemonic::parse(mnemonic.trim()).expect("mnemonic must be validated first");
    m.to_seed("")
}

// ---------------------------------------------------------------------------
// BIP-32 / BIP-44 HD derivation (secp256k1)
// ---------------------------------------------------------------------------

/// BIP-32 master key from a 64-byte BIP-39 seed: HMAC-SHA512("Bitcoin seed", seed).
pub fn master_key_from_seed(seed: &[u8]) -> ([u8; 32], [u8; 32]) {
    let mut mac = <Hmac<Sha512> as Mac>::new_from_slice(b"Bitcoin seed").unwrap();
    mac.update(seed);
    let i = mac.finalize().into_bytes();
    let mut key = [0u8; 32];
    let mut chain = [0u8; 32];
    key.copy_from_slice(&i[..32]);
    chain.copy_from_slice(&i[32..]);
    (key, chain)
}

fn scalar_from_be(b: &[u8; 32]) -> Option<Scalar> {
    Option::<Scalar>::from(Scalar::from_repr((*b).into()))
}

/// One step of BIP-32 CKD for secp256k1. `index >= 0x80000000` means hardened.
pub fn ckd_priv(
    parent_key: &[u8; 32],
    parent_chain: &[u8; 32],
    index: u32,
) -> Result<([u8; 32], [u8; 32]), String> {
    let mut data = Vec::with_capacity(37);
    if index & 0x8000_0000 != 0 {
        data.push(0u8);
        data.extend_from_slice(parent_key);
    } else {
        let sk = SecretKey::from_slice(parent_key).map_err(|e| e.to_string())?;
        let pk = sk.public_key();
        data.extend_from_slice(pk.to_encoded_point(true).as_bytes());
    }
    data.extend_from_slice(&index.to_be_bytes());

    let mut mac = <Hmac<Sha512> as Mac>::new_from_slice(parent_chain).unwrap();
    mac.update(&data);
    let i = mac.finalize().into_bytes();
    let mut il = [0u8; 32];
    il.copy_from_slice(&i[..32]);
    let mut chain = [0u8; 32];
    chain.copy_from_slice(&i[32..]);

    let il_scalar = scalar_from_be(&il).ok_or("invalid child key (IL >= curve order)")?;
    if il_scalar == Scalar::ZERO {
        return Err("invalid child key (IL == 0)".into());
    }
    let parent_scalar =
        scalar_from_be(parent_key).ok_or("invalid parent key")?;
    let child = il_scalar + parent_scalar; // scalar addition is mod curve order
    if child == Scalar::ZERO {
        return Err("invalid child key (zero)".into());
    }
    let mut child_key = [0u8; 32];
    child_key.copy_from_slice(&child.to_repr());
    Ok((child_key, chain))
}

/// DerivePrivateKeyFromPath — BIP-32 derivation along a path like
/// m/44'/60'/0'/0/n. Segments ending in ' or h/H are hardened.
pub fn derive_private_key_from_path(seed: &[u8], path: &str) -> Result<[u8; 32], String> {
    let path = path.trim();
    if path.is_empty() {
        return Err("empty derivation path".into());
    }
    let segments: Vec<&str> = path.split('/').collect();
    let mut start = 0;
    if segments[0] == "m" || segments[0] == "M" {
        start = 1;
    }
    if start >= segments.len() {
        return Err("invalid derivation path".into());
    }
    let (mut key, mut chain) = master_key_from_seed(seed);
    for seg in &segments[start..] {
        if seg.is_empty() {
            return Err(format!("invalid derivation path segment in {path}"));
        }
        let hardened = seg.ends_with('\'') || seg.ends_with('h') || seg.ends_with('H');
        let num_str = if hardened { &seg[..seg.len() - 1] } else { *seg };
        let mut index: u32 = num_str
            .parse()
            .map_err(|_| format!("invalid derivation index: {seg}"))?;
        if index >= 0x8000_0000 {
            return Err(format!("derivation index out of range: {seg}"));
        }
        if hardened {
            index += 0x8000_0000;
        }
        let (k, c) = ckd_priv(&key, &chain, index)?;
        key = k;
        chain = c;
    }
    Ok(key)
}

/// DeriveEVMPrivateKey — canonical path m/44'/60'/0'/0/{index}.
pub fn derive_evm_private_key(seed: &[u8], index: u32) -> Result<[u8; 32], String> {
    derive_private_key_from_path(seed, &format!("m/44'/60'/0'/0/{index}"))
}

/// Canonical BIP-44 account path m/44'/{coin_type}'/0'/0/{index}
/// (matches the canonical Go backend for secp256k1 chains).
pub fn derive_path_for_account(coin_type: u32, index: u32) -> String {
    format!("m/44'/{coin_type}'/0'/0/{index}")
}

// ---------------------------------------------------------------------------
// EVM address
// ---------------------------------------------------------------------------

/// PrivateKeyToAddress — keccak256(uncompressed pubkey[1..]) last 20 bytes,
/// rendered as an EIP-55 checksummed 0x-prefixed hex string.
pub fn private_key_to_address(priv_key: &[u8; 32]) -> Result<String, String> {
    let sk = SecretKey::from_slice(priv_key).map_err(|e| e.to_string())?;
    let pk = sk.public_key();
    Ok(public_key_to_address(&pk))
}

/// Public key → EIP-55 checksummed address.
pub fn public_key_to_address(pk: &PublicKey) -> String {
    let uncompressed = pk.to_encoded_point(false);
    let hash = Keccak256::digest(&uncompressed.as_bytes()[1..]);
    eip55_checksum(&hash[12..])
}

/// EIP-55 mixed-case checksum encoding of a 20-byte address.
pub fn eip55_checksum(addr: &[u8]) -> String {
    let lower = hex::encode(addr);
    let hash = Keccak256::digest(lower.as_bytes());
    let mut out = String::with_capacity(42);
    out.push_str("0x");
    for (i, c) in lower.chars().enumerate() {
        let nibble = (hash[i / 2] >> (if i % 2 == 0 { 4 } else { 0 })) & 0xf;
        if c.is_ascii_hexdigit() && c.is_ascii_alphabetic() && nibble >= 8 {
            out.push(c.to_ascii_uppercase());
        } else {
            out.push(c);
        }
    }
    out
}

// ---------------------------------------------------------------------------
// Seed encryption at rest: scrypt + AES-256-GCM
// ---------------------------------------------------------------------------

fn derive_scrypt_key(password: &str, salt: &[u8]) -> Result<[u8; KEY_LEN], String> {
    let params = scrypt::Params::new(SCRYPT_LOG_N, SCRYPT_R, SCRYPT_P, KEY_LEN)
        .map_err(|e| e.to_string())?;
    let mut key = [0u8; KEY_LEN];
    scrypt::scrypt(password.as_bytes(), salt, &params, &mut key).map_err(|e| e.to_string())?;
    Ok(key)
}

/// EncryptSeed — scrypt(password, random 32-byte salt) → AES-256-GCM over the
/// 64-byte BIP-39 seed (matches the canonical Go backend, which encrypts the
/// seed bytes, not the mnemonic text). Output: hex(salt || nonce || ct||tag).
pub fn encrypt_seed(seed: &[u8], password: &str) -> Result<String, String> {
    let mut salt = [0u8; SCRYPT_SALT_LEN];
    let mut nonce = [0u8; AES_GCM_NONCE_LEN];
    use rand::RngCore;
    rand::thread_rng().fill_bytes(&mut salt);
    rand::thread_rng().fill_bytes(&mut nonce);

    let key = derive_scrypt_key(password, &salt)?;
    let cipher = Aes256Gcm::new_from_slice(&key).map_err(|e| e.to_string())?;
    let ct = cipher
        .encrypt(aes_gcm::Nonce::from_slice(&nonce), seed)
        .map_err(|e| e.to_string())?;

    let mut blob = Vec::with_capacity(SCRYPT_SALT_LEN + AES_GCM_NONCE_LEN + ct.len());
    blob.extend_from_slice(&salt);
    blob.extend_from_slice(&nonce);
    blob.extend_from_slice(&ct);
    Ok(hex::encode(blob))
}

/// DecryptSeed — inverse of encrypt_seed. AES-GCM tag verification is the
/// constant-time authentication check; wrong passwords fail closed.
pub fn decrypt_seed(encrypted_hex: &str, password: &str) -> Result<Vec<u8>, String> {
    let blob = hex::decode(encrypted_hex).map_err(|e| e.to_string())?;
    if blob.len() < SCRYPT_SALT_LEN + AES_GCM_NONCE_LEN + 16 {
        return Err("encrypted seed too short".into());
    }
    let (salt, rest) = blob.split_at(SCRYPT_SALT_LEN);
    let (nonce, ct) = rest.split_at(AES_GCM_NONCE_LEN);

    let key = derive_scrypt_key(password, salt)?;
    let cipher = Aes256Gcm::new_from_slice(&key).map_err(|e| e.to_string())?;
    cipher
        .decrypt(aes_gcm::Nonce::from_slice(nonce), ct)
        .map_err(|_| "invalid password or corrupted seed".to_string())
}

/// ConstantTimeEq — constant-time byte-slice equality (port of Go helper).
pub fn constant_time_eq(a: &[u8], b: &[u8]) -> bool {
    if a.len() != b.len() {
        return false;
    }
    let mut diff = 0u8;
    for (x, y) in a.iter().zip(b.iter()) {
        diff |= x ^ y;
    }
    diff == 0
}

#[cfg(test)]
mod tests {
    use super::*;

    const TEST_MNEMONIC: &str = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";

    #[test]
    fn bip39_roundtrip() {
        let m = generate_mnemonic();
        assert_eq!(m.split_whitespace().count(), 24);
        assert!(validate_mnemonic(&m));
        let seed = mnemonic_to_seed(&m);
        assert_eq!(seed.len(), 64);
        // Same mnemonic → same seed (deterministic).
        assert_eq!(mnemonic_to_seed(&m), seed);
    }

    #[test]
    fn bip39_rejects_invalid() {
        assert!(!validate_mnemonic("abandon abandon abandon"));
        assert!(!validate_mnemonic("not a real mnemonic at all honestly"));
        assert!(validate_mnemonic(TEST_MNEMONIC));
    }

    #[test]
    fn bip39_seed_vector() {
        // Well-known BIP-39 test vector (empty passphrase).
        let seed = mnemonic_to_seed(TEST_MNEMONIC);
        let expected = "5eb00bbddcf069084889a8ab9155568165f5c453ccb85e70811aaed6f6da5fc19a5ac40b389cd370d086206dec8aa6c43daea6690f20ad3d8d48b2d2ce9e38e4";
        assert_eq!(hex::encode(seed), expected);
    }

    #[test]
    fn evm_address_known_vector() {
        // m/44'/60'/0'/0/0 for the canonical test mnemonic.
        let seed = mnemonic_to_seed(TEST_MNEMONIC);
        let key = derive_evm_private_key(&seed, 0).unwrap();
        assert_eq!(
            hex::encode(key),
            "1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727"
        );
        let addr = private_key_to_address(&key).unwrap();
        assert_eq!(addr, "0x9858EfFD232B4033E47d90003D41EC34EcaEda94");
    }

    #[test]
    fn hd_path_variants() {
        let seed = mnemonic_to_seed(TEST_MNEMONIC);
        // Hardened markers ' and h must agree.
        let a = derive_private_key_from_path(&seed, "m/44'/60'/0'/0/1").unwrap();
        let b = derive_private_key_from_path(&seed, "m/44h/60h/0h/0/1").unwrap();
        assert_eq!(a, b);
        assert!(derive_private_key_from_path(&seed, "m/44'/x/0").is_err());
        // Distinct indices → distinct keys.
        let c = derive_evm_private_key(&seed, 2).unwrap();
        assert_ne!(a, c);
    }

    #[test]
    fn seed_encryption_roundtrip() {
        let enc = encrypt_seed(TEST_MNEMONIC.as_bytes(), "correct horse battery staple").unwrap();
        let dec = decrypt_seed(&enc, "correct horse battery staple").unwrap();
        assert_eq!(dec, TEST_MNEMONIC.as_bytes());
        // Wrong password must fail closed.
        assert!(decrypt_seed(&enc, "wrong password").is_err());
        // Corrupted ciphertext must fail closed.
        let mut blob = hex::decode(&enc).unwrap();
        let n = blob.len();
        blob[n - 1] ^= 0xff;
        assert!(decrypt_seed(&hex::encode(blob), "correct horse battery staple").is_err());
    }

    #[test]
    fn constant_time_compare() {
        assert!(constant_time_eq(b"abc", b"abc"));
        assert!(!constant_time_eq(b"abc", b"abd"));
        assert!(!constant_time_eq(b"abc", b"ab"));
    }
}
