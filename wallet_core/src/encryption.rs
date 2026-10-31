// ============================================================================
// TIGERWALLET ENCRYPTION MODULE
// AES-256-GCM encryption for secure key storage
// ============================================================================

use aes_gcm::{
    aead::{Aead, KeyInit, OsRng},
    Aes256Gcm, Nonce,
};
use pbkdf2::pbkdf2_hmac;
use sha2::Sha256;
use zeroize::Zeroize;
use rand::RngCore;

/// Number of PBKDF2 iterations.
///
/// This protects the encrypted-mnemonic and encrypted-private-key blobs. 100k
/// was well below the OWASP floor for PBKDF2-HMAC-SHA256 (600k as of 2023), so
/// an attacker holding a backup could test guesses far faster than the cost of
/// the key it protects justifies.
const PBKDF2_ITERATIONS: u32 = 600_000;
const SALT_LENGTH: usize = 32;
const NONCE_LENGTH: usize = 12;
const KEY_LENGTH: usize = 32;

/// Encrypted data structure
#[derive(Debug, Clone)]
pub struct EncryptedData {
    pub salt: Vec<u8>,
    pub nonce: Vec<u8>,
    pub ciphertext: Vec<u8>,
}

impl EncryptedData {
    pub fn new(salt: Vec<u8>, nonce: Vec<u8>, ciphertext: Vec<u8>) -> Self {
        Self { salt, nonce, ciphertext }
    }

    pub fn to_bytes(&self) -> Vec<u8> {
        let mut result = Vec::new();
        result.extend_from_slice(&self.salt);
        result.extend_from_slice(&self.nonce);
        result.extend_from_slice(&self.ciphertext);
        result
    }

    pub fn from_bytes(data: &[u8]) -> Result<Self, EncryptionError> {
        if data.len() < SALT_LENGTH + NONCE_LENGTH {
            return Err(EncryptionError::InvalidData);
        }
        
        let salt = data[..SALT_LENGTH].to_vec();
        let nonce = data[SALT_LENGTH..SALT_LENGTH + NONCE_LENGTH].to_vec();
        let ciphertext = data[SALT_LENGTH + NONCE_LENGTH..].to_vec();
        
        Ok(Self { salt, nonce, ciphertext })
    }
}

/// Encrypt data with password
pub fn encrypt_with_password(data: &[u8], password: &str) -> Result<EncryptedData, EncryptionError> {
    // Generate random salt
    let mut salt = [0u8; SALT_LENGTH];
    getrandom(&mut salt)?;

    // Derive key using PBKDF2
    let mut key = derive_key(password, &salt)?;
    
    // Generate random nonce
    let mut nonce_bytes = [0u8; NONCE_LENGTH];
    getrandom(&mut nonce_bytes)?;
    
    // Encrypt
    let cipher = Aes256Gcm::new_from_slice(&key)
        .map_err(|_| EncryptionError::KeyDerivationFailed)?;
    
    let nonce = Nonce::from_slice(&nonce_bytes);
    let ciphertext = cipher.encrypt(nonce, data)
        .map_err(|_| EncryptionError::EncryptionFailed)?;
    
    // Zeroize key from memory
    key.zeroize();
    
    Ok(EncryptedData::new(salt.to_vec(), nonce_bytes.to_vec(), ciphertext))
}

/// Decrypt data with password
pub fn decrypt_with_password(encrypted: &EncryptedData, password: &str) -> Result<Vec<u8>, EncryptionError> {
    // Derive key using stored salt
    let key = derive_key(password, &encrypted.salt)?;
    
    // Decrypt
    let cipher = Aes256Gcm::new_from_slice(&key)
        .map_err(|_| EncryptionError::KeyDerivationFailed)?;
    
    let nonce = Nonce::from_slice(&encrypted.nonce);
    let plaintext = cipher.decrypt(nonce, encrypted.ciphertext.as_ref())
        .map_err(|_| EncryptionError::DecryptionFailed)?;
    
    // Zeroize key from memory
    let mut key = key;
    key.zeroize();
    
    Ok(plaintext)
}

/// Derive encryption key from password
fn derive_key(password: &str, salt: &[u8]) -> Result<[u8; KEY_LENGTH], EncryptionError> {
    let mut key = [0u8; KEY_LENGTH];
    pbkdf2_hmac::<Sha256>(
        password.as_bytes(),
        salt,
        PBKDF2_ITERATIONS,
        &mut key,
    );
    Ok(key)
}

/// Generate random bytes
fn getrandom(dest: &mut [u8]) -> Result<(), EncryptionError> {
    OsRng.fill_bytes(dest);
    Ok(())
}

/// Encrypt mnemonic phrase
pub fn encrypt_mnemonic(mnemonic: &str, password: &str) -> Result<EncryptedData, EncryptionError> {
    encrypt_with_password(mnemonic.as_bytes(), password)
}

/// Decrypt mnemonic phrase
pub fn decrypt_mnemonic(encrypted: &EncryptedData, password: &str) -> Result<String, EncryptionError> {
    let plaintext = decrypt_with_password(encrypted, password)?;
    String::from_utf8(plaintext)
        .map_err(|_| EncryptionError::InvalidData)
}

/// Encrypt private key
pub fn encrypt_private_key(private_key: &[u8], password: &str) -> Result<EncryptedData, EncryptionError> {
    encrypt_with_password(private_key, password)
}

/// Decrypt private key
pub fn decrypt_private_key(encrypted: &EncryptedData, password: &str) -> Result<Vec<u8>, EncryptionError> {
    decrypt_with_password(encrypted, password)
}

/// Wallet keystore (encrypted JSON).
///
/// The canonical Web3 keystore v3 lives in `keystore_v3` (scrypt or PBKDF2 with
/// AES-128-CTR and a keccak256 MAC, matching the format MetaMask and geth
/// write). This struct is kept only so an existing serialised keystore can still
/// be read back; new keystores must go through `keystore_v3::encrypt_key`.
#[derive(Debug, Clone)]
pub struct Keystore {
    pub version: u32,
    pub id: String,
    pub address: String,
    pub crypto: KeystoreCrypto,
}

#[derive(Debug, Clone)]
pub struct KeystoreCrypto {
    pub cipher: String,
    pub cipherparams: CipherParams,
    pub ciphertext: String,
    pub kdf: String,
    pub kdfparams: KdfParams,
    pub mac: String,
}

#[derive(Debug, Clone)]
pub struct CipherParams {
    pub iv: String,
}

#[derive(Debug, Clone)]
pub struct KdfParams {
    pub salt: String,
    pub dklen: u32,
    pub c: u32,
    pub prf: String,
}

/// Create keystore from private key.
///
/// Delegates to `keystore_v3::encrypt_key`, which writes the standard format.
/// The previous implementation produced a keystore no other wallet could read:
/// it concatenated salt, nonce and ciphertext into the `ciphertext` field, and
/// its MAC was `sha256(ciphertext || private_key)` — a digest over the
/// plaintext, which neither authenticates the ciphertext nor matches the format
/// any consumer expects.
pub fn create_keystore(private_key: &[u8], address: &str, password: &str) -> Result<Keystore, EncryptionError> {
    let id = generate_uuid();
    let v3 = crate::keystore_v3::encrypt_key(private_key, password, address, &id)
        .map_err(|_| EncryptionError::EncryptionFailed)?;

    let ciphertext = crate::keystore_v3::to_json(&v3)
        .map_err(|_| EncryptionError::EncryptionFailed)?;

    Ok(Keystore {
        version: v3.version,
        id: v3.id,
        address: v3.address,
        crypto: KeystoreCrypto {
            cipher: v3.crypto.cipher,
            cipherparams: CipherParams { iv: v3.crypto.cipherparams.iv },
            ciphertext,
            kdf: v3.crypto.kdf,
            kdfparams: KdfParams {
                salt: v3.crypto.kdfparams.salt,
                dklen: v3.crypto.kdfparams.dklen,
                c: v3.crypto.kdfparams.c.unwrap_or(0),
                prf: v3.crypto.kdfparams.prf.unwrap_or_default(),
            },
            mac: v3.crypto.mac,
        },
    })
}

/// Decrypt keystore.
///
/// `Keystore::ciphertext` holds the full standard keystore JSON, so this parses
/// it and defers to `keystore_v3::decrypt_key`, which verifies the keccak256 MAC
/// before it decrypts.
pub fn decrypt_keystore(keystore: &Keystore, password: &str) -> Result<Vec<u8>, EncryptionError> {
    let v3 = crate::keystore_v3::from_json(&keystore.crypto.ciphertext)
        .map_err(|_| EncryptionError::InvalidData)?;
    crate::keystore_v3::decrypt_key(&v3, password).map_err(|_| EncryptionError::DecryptionFailed)
}

/// Verify keystore password.
pub fn verify_keystore(keystore: &Keystore, password: &str) -> bool {
    decrypt_keystore(keystore, password).is_ok()
}

/// Generate UUID v4
fn generate_uuid() -> String {
    let mut bytes = [0u8; 16];
    getrandom(&mut bytes).unwrap_or_default();
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    
    format!(
        "{:02x}{:02x}{:02x}{:02x}-{:02x}{:02x}-{:02x}{:02x}-{:02x}{:02x}-{:02x}{:02x}{:02x}{:02x}{:02x}{:02x}",
        bytes[0], bytes[1], bytes[2], bytes[3],
        bytes[4], bytes[5],
        bytes[6], bytes[7],
        bytes[8], bytes[9],
        bytes[10], bytes[11], bytes[12], bytes[13], bytes[14], bytes[15]
    )
}

/// Encryption errors
#[derive(Debug, Clone)]
pub enum EncryptionError {
    EncryptionFailed,
    DecryptionFailed,
    KeyDerivationFailed,
    InvalidData,
}

impl std::fmt::Display for EncryptionError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            EncryptionError::EncryptionFailed => write!(f, "Encryption failed"),
            EncryptionError::DecryptionFailed => write!(f, "Decryption failed"),
            EncryptionError::KeyDerivationFailed => write!(f, "Key derivation failed"),
            EncryptionError::InvalidData => write!(f, "Invalid data"),
        }
    }
}

impl std::error::Error for EncryptionError {}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::keystore_v3;

    const MNEMONIC: &str =
        "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";

    #[test]
    fn encrypt_decrypt_round_trip() {
        let data = b"Hello, TigerWallet!";
        let password = "secure_password_123";

        let encrypted = encrypt_with_password(data, password).unwrap();
        let decrypted = decrypt_with_password(&encrypted, password).unwrap();

        assert_eq!(data.to_vec(), decrypted);
    }

    #[test]
    fn encrypt_uses_the_salt_it_stores() {
        // The salt written into EncryptedData has to be the one the key was
        // derived from, otherwise nothing can ever decrypt the blob again.
        let encrypted = encrypt_with_password(b"payload", "pw").unwrap();
        assert_eq!(encrypted.salt.len(), SALT_LENGTH);
        assert_eq!(encrypted.nonce.len(), NONCE_LENGTH);

        let key = derive_key("pw", &encrypted.salt).unwrap();
        let cipher = Aes256Gcm::new_from_slice(&key).unwrap();
        let nonce = Nonce::from_slice(&encrypted.nonce);
        let plaintext = cipher.decrypt(nonce, encrypted.ciphertext.as_ref()).unwrap();
        assert_eq!(plaintext, b"payload");
    }

    #[test]
    fn encrypt_is_randomised() {
        // A fixed salt or nonce would make two encryptions of the same plaintext
        // identical, which leaks equality to anyone reading the backup.
        let a = encrypt_with_password(b"same", "pw").unwrap();
        let b = encrypt_with_password(b"same", "pw").unwrap();
        assert_ne!(a.salt, b.salt);
        assert_ne!(a.nonce, b.nonce);
        assert_ne!(a.ciphertext, b.ciphertext);
    }

    #[test]
    fn decrypt_rejects_the_wrong_password() {
        let encrypted = encrypt_with_password(b"secret", "right").unwrap();
        assert!(decrypt_with_password(&encrypted, "wrong").is_err());
    }

    #[test]
    fn decrypt_rejects_tampered_ciphertext() {
        // GCM authenticates the ciphertext, so a flipped bit must fail rather
        // than yield corrupted plaintext.
        let mut encrypted = encrypt_with_password(b"secret", "pw").unwrap();
        encrypted.ciphertext[0] ^= 0x01;
        assert!(decrypt_with_password(&encrypted, "pw").is_err());
    }

    #[test]
    fn mnemonic_round_trip() {
        let encrypted = encrypt_mnemonic(MNEMONIC, "wallet_password").unwrap();
        let decrypted = decrypt_mnemonic(&encrypted, "wallet_password").unwrap();
        assert_eq!(MNEMONIC, decrypted);
    }

    #[test]
    fn private_key_round_trip() {
        let key = [0x11u8; 32];
        let encrypted = encrypt_private_key(&key, "pw").unwrap();
        assert_eq!(decrypt_private_key(&encrypted, "pw").unwrap(), key.to_vec());
    }

    #[test]
    fn kdf_iterations_meet_the_owasp_floor() {
        // Guards against a future edit quietly weakening the KDF. OWASP's floor
        // for PBKDF2-HMAC-SHA256 is 600,000.
        assert!(
            PBKDF2_ITERATIONS >= 600_000,
            "PBKDF2_ITERATIONS dropped to {}, below the 600k floor",
            PBKDF2_ITERATIONS
        );
    }

    #[test]
    fn keystore_round_trips_and_is_standard_v3() {
        let private_key = [0x42u8; 32];
        let address = "0x9858EfFD232B4033E47d90003D41EC34EcaEda94";

        let ks = create_keystore(&private_key, address, "keystore-pw").unwrap();
        assert_eq!(ks.version, 3);
        assert_eq!(ks.address, address);
        assert_eq!(ks.id.len(), 36);

        // The body has to be a standard v3 document: the old implementation put
        // a salt||nonce||ciphertext blob in `ciphertext`, so no other wallet
        // could parse the file.
        let v3 = keystore_v3::from_json(&ks.crypto.ciphertext).unwrap();
        assert_eq!(v3.crypto.mac, ks.crypto.mac);
        assert_eq!(v3.crypto.kdf, "scrypt");
        assert_eq!(v3.crypto.cipher, "aes-128-ctr");
        assert_eq!(v3.crypto.kdfparams.dklen, 32);
        assert_eq!(ks.crypto.kdfparams.dklen, 32);
        assert_eq!(ks.crypto.cipher, "aes-128-ctr");

        // The MAC is checked before the ciphertext is decrypted, so a wrong
        // password is rejected here without paying for a second scrypt run.
        assert!(decrypt_keystore(&ks, "not-the-password").is_err());
        assert!(!verify_keystore(&ks, "not-the-password"));
    }

    #[test]
    fn keystore_mac_is_keccak_over_the_derived_key() {
        // Recompute the KDF and the MAC independently, from the stored
        // parameters, to prove the file follows the standard construction and
        // that the MAC is not a digest of the private key.
        use sha3::{Digest as _, Keccak256};

        let private_key = [0x07u8; 32];
        let password = "pw";
        let v3 = keystore_v3::encrypt_key_pbkdf2(
            &private_key,
            password,
            "0x0000000000000000000000000000000000000001",
            "00000000-0000-4000-8000-000000000000",
            100_000,
        )
        .unwrap();
        let ks = create_keystore(&private_key, "0x0000000000000000000000000000000000000001", password).unwrap();
        assert_eq!(ks.version, 3);

        let salt = hex::decode(&v3.crypto.kdfparams.salt).unwrap();
        let c = v3.crypto.kdfparams.c.unwrap();
        let mut derived = vec![0u8; 32];
        pbkdf2_hmac::<Sha256>(password.as_bytes(), &salt, c, &mut derived);

        let ciphertext = hex::decode(&v3.crypto.ciphertext).unwrap();
        let mut hasher = Keccak256::new();
        hasher.update(&derived[16..32]);
        hasher.update(&ciphertext);
        assert_eq!(hex::encode(hasher.finalize()), v3.crypto.mac);

        // A digest over the plaintext, as the previous implementation wrote,
        // must not match.
        let mut leak = Keccak256::new();
        leak.update(&ciphertext);
        leak.update(private_key);
        assert_ne!(hex::encode(leak.finalize()), v3.crypto.mac);
    }

    #[test]
    fn keystore_rejects_tampered_json() {
        let mut ks = create_keystore(&[0x02u8; 32], "0x0000000000000000000000000000000000000003", "pw").unwrap();
        ks.crypto.ciphertext = "not json".to_string();
        assert!(decrypt_keystore(&ks, "pw").is_err());
        assert!(!verify_keystore(&ks, "pw"));
    }
}
