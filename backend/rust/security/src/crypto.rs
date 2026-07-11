use sha2::{Sha256, Sha512, Digest};
use sha3::{Keccak256, Keccak512};
use secp256k1::{Secp256k1, SecretKey, PublicKey, Message, Signature};
use ed25519_dalek::{Signer, SigningKey, Verifier, VerificationKey};
use aes_gcm::{Aes256Gcm, Key, Nonce};
use aes_gcm::aead::{Aead, KeyInit};
use rand::RngCore;
use zeroize::Zeroize;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum CryptoError {
    #[error("Encryption failed")]
    EncryptionFailed,
    #[error("Decryption failed")]
    DecryptionFailed,
    #[error("Invalid key")]
    InvalidKey,
    #[error("Signing failed")]
    SigningFailed,
    #[error("Verification failed")]
    VerificationFailed,
    #[error("Invalid signature")]
    InvalidSignature,
    #[error("Invalid address")]
    InvalidAddress,
}

pub struct Crypto;

impl Crypto {
    pub fn sha256(data: &[u8]) -> Vec<u8> {
        let mut hasher = Sha256::new();
        hasher.update(data);
        hasher.finalize().to_vec()
    }

    pub fn sha512(data: &[u8]) -> Vec<u8> {
        let mut hasher = Sha512::new();
        hasher.update(data);
        hasher.finalize().to_vec()
    }

    pub fn keccak256(data: &[u8]) -> Vec<u8> {
        let mut hasher = Keccak256::new();
        hasher.update(data);
        hasher.finalize().to_vec()
    }

    pub fn keccak512(data: &[u8]) -> Vec<u8> {
        let mut hasher = Keccak512::new();
        hasher.update(data);
        hasher.finalize().to_vec()
    }

    pub fn encrypt_aes256gcm(plaintext: &[u8], key: &[u8], nonce: &[u8]) -> Result<Vec<u8>, CryptoError> {
        if key.len() != 32 || nonce.len() != 12 {
            return Err(CryptoError::InvalidKey);
        }

        let key = Key::<Aes256Gcm>::from_slice(key);
        let cipher = Aes256Gcm::new(key);
        let nonce = Nonce::from_slice(nonce);

        cipher.encrypt(nonce, plaintext)
            .map_err(|_| CryptoError::EncryptionFailed)
    }

    pub fn decrypt_aes256gcm(ciphertext: &[u8], key: &[u8], nonce: &[u8]) -> Result<Vec<u8>, CryptoError> {
        if key.len() != 32 || nonce.len() != 12 {
            return Err(CryptoError::InvalidKey);
        }

        let key = Key::<Aes256Gcm>::from_slice(key);
        let cipher = Aes256Gcm::new(key);
        let nonce = Nonce::from_slice(nonce);

        cipher.decrypt(nonce, ciphertext)
            .map_err(|_| CryptoError::DecryptionFailed)
    }

    pub fn generate_secp256k1_key() -> Vec<u8> {
        let secp = Secp256k1::new();
        let mut rng = rand::thread_rng();
        let mut key = [0u8; 32];
        rng.fill_bytes(&mut key);
        
        let secret_key = SecretKey::from_slice(&key).expect("32 bytes, within curve order");
        secret_key.as_ref().to_vec()
    }

    pub fn derive_public_key(private_key: &[u8], compressed: bool) -> Result<Vec<u8>, CryptoError> {
        let secp = Secp256k1::new();
        let secret_key = SecretKey::from_slice(private_key)
            .map_err(|_| CryptoError::InvalidKey)?;
        
        let public_key = PublicKey::from_secret_key(&secp, &secret_key);
        
        if compressed {
            Ok(public_key.serialize_compressed().to_vec())
        } else {
            Ok(public_key.serialize_uncompressed().to_vec())
        }
    }

    pub fn sign_message(message: &[u8], private_key: &[u8]) -> Result<Vec<u8>, CryptoError> {
        let secp = Secp256k1::new();
        let secret_key = SecretKey::from_slice(private_key)
            .map_err(|_| CryptoError::InvalidKey)?;
        
        let message = Message::from_slice(message)
            .map_err(|_| CryptoError::SigningFailed)?;
        
        let signature = secp.sign_message(&message, &secret_key);
        
        Ok(signature.to_vec())
    }

    pub fn verify_signature(message: &[u8], signature: &[u8], public_key: &[u8]) -> Result<bool, CryptoError> {
        let secp = Secp256k1::new();
        
        let message = Message::from_slice(message)
            .map_err(|_| CryptoError::InvalidSignature)?;
        
        let signature = Signature::from_compact(signature)
            .map_err(|_| CryptoError::InvalidSignature)?;
        
        let public_key = PublicKey::from_slice(public_key)
            .map_err(|_| CryptoError::InvalidSignature)?;

        Ok(secp.verify_message(&message, signature, &public_key).is_ok())
    }

    pub fn generate_ed25519_key() -> (Vec<u8>, Vec<u8>) {
        let mut rng = rand::thread_rng();
        let signing_key = SigningKey::generate(&mut rng);
        let verification_key = signing_key.verifying_key();
        
        (signing_key.to_bytes().to_vec(), verification_key.to_bytes().to_vec())
    }

    pub fn sign_ed25519(message: &[u8], private_key: &[u8]) -> Result<Vec<u8>, CryptoError> {
        let signing_key = SigningKey::from_bytes(private_key.try_into()
            .map_err(|_| CryptoError::InvalidKey)?);
        
        let signature = signing_key.sign(message);
        Ok(signature.to_bytes().to_vec())
    }

    pub fn verify_ed25519(message: &[u8], signature: &[u8], public_key: &[u8]) -> Result<bool, CryptoError> {
        let verification_key = VerificationKey::from_bytes(public_key.try_into()
            .map_err(|_| CryptoError::InvalidKey)?);
        
        let signature = ed25519_dalek::Signature::from_bytes(signature.try_into()
            .map_err(|_| CryptoError::InvalidSignature)?);
        
        Ok(verification_key.verify(message, &signature).is_ok())
    }

    pub fn pubkey_to_address(public_key: &[u8]) -> String {
        let hash = Self::keccak256(public_key);
        let address = &hash[12..];
        format!("0x{}", hex::encode(address))
    }

    pub fn is_valid_address(address: &str) -> bool {
        if !address.starts_with("0x") || address.len() != 42 {
            return false;
        }
        
        hex::decode(&address[2..]).is_ok()
    }
}

pub fn encrypt_mnemonic(mnemonic: &[u8], password: &str) -> Result<(Vec<u8>, Vec<u8>), CryptoError> {
    let salt = Crypto::sha256(password.as_bytes());
    let key = Crypto::sha256(&salt);
    
    let mut nonce = [0u8; 12];
    rand::thread_rng().fill_bytes(&mut nonce);
    
    let ciphertext = Crypto::encrypt_aes256gcm(mnemonic, &key, &nonce)?;
    
    Ok((ciphertext, nonce.to_vec()))
}

pub fn decrypt_mnemonic(ciphertext: &[u8], password: &str, nonce: &[u8]) -> Result<Vec<u8>, CryptoError> {
    let salt = Crypto::sha256(password.as_bytes());
    let key = Crypto::sha256(&salt);
    
    Crypto::decrypt_aes256gcm(ciphertext, &key, nonce)
}
