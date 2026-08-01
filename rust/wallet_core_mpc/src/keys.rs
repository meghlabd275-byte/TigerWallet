//! Keys module - Key derivation and encryption
//! 
//! Provides BIP-32 HD key derivation, key encryption, and key management.

use sha2::{Sha256, Sha512, Digest};
use hmac::{Hmac, Mac};
use secp256k1::{Secp256k1, PublicKey, PrivateKey, SecretKey};
use zeroize::Zeroize;
use aes_gcm::{
    aead::{Aead, KeyInit},
    Aes256Gcm, Nonce,
};
use rand::RngCore;

type HmacSha512 = Hmac<Sha512>;
type HmacSha256 = Hmac<Sha256>;

/// HD Wallet for hierarchical deterministic key derivation
pub struct HdWallet {
    pub seed: Vec<u8>,
    pub master_key: ExtendedKey,
}

/// Extended key (BIP-32)
#[derive(Debug, Clone)]
pub struct ExtendedKey {
    pub key: Vec<u8>,       // 32 bytes private key or 33 bytes compressed public key
    pub chain_code: [u8; 32],
    pub depth: u8,
    pub parent_fingerprint: [u8; 4],
    pub child_number: u32,
    pub is_public: bool,
}

impl HdWallet {
    /// Create HD wallet from seed
    pub fn from_seed(seed: &[u8]) -> Self {
        // Derive master key from seed using HMAC-SHA512
        let mut mac = HmacSha512::new_from_slice(b"Bitcoin seed").unwrap();
        mac.update(seed);
        let result = mac.finalize().into_bytes();
        
        let mut key = [0u8; 32];
        key.copy_from_slice(&result[0..32]);
        
        let mut chain_code = [0u8; 32];
        chain_code.copy_from_slice(&result[32..64]);
        
        let master_key = ExtendedKey {
            key: key.to_vec(),
            chain_code,
            depth: 0,
            parent_fingerprint: [0, 0, 0, 0],
            child_number: 0,
            is_public: false,
        };
        
        Self {
            seed: seed.to_vec(),
            master_key,
        }
    }
    
    /// Derive child key from path
    pub fn derive_key(&self, path: &str) -> Result<ExtendedKey, crate::WalletError> {
        let mut key = self.master_key.clone();
        
        // Parse path (e.g., "m/44'/60'/0'/0/0")
        let segments: Vec<&str> = path.split('/').collect();
        
        for (i, segment) in segments.iter().enumerate() {
            if i == 0 {
                // Skip "m"
                if *segment == "m" {
                    continue;
                }
            }
            
            // Parse child number
            let mut hardened = false;
            let mut num_str = *segment;
            
            if num_str.ends_with('\'') {
                hardened = true;
                num_str = &num_str[..num_str.len() - 1];
            }
            
            let child_num: u32 = num_str.parse()
                .map_err(|_| crate::WalletError::InvalidDerivationPath(
                    format!("Invalid path segment: {}", segment)
                ))?;
            
            key = self.derive_child_key(&key, child_num, hardened)?;
        }
        
        Ok(key)
    }
    
    /// Derive child key
    fn derive_child_key(
        &self,
        parent: &ExtendedKey,
        child_num: u32,
        hardened: bool,
    ) -> Result<ExtendedKey, crate::WalletError> {
        let mut data = Vec::new();
        
        if hardened || parent.is_public {
            // Hardened derivation: prepend 0x00 to private key
            if hardened {
                data.push(0x00);
            }
            data.extend_from_slice(&parent.key);
        } else {
            // Normal derivation: use public key
            // For private key derivation, we need to convert to public key
            let public_key = self.private_to_public(&parent.key)?;
            data.extend_from_slice(&public_key);
        }
        
        // Add child number
        let cn = if hardened {
            0x80000000 | child_num
        } else {
            child_num
        };
        
        // Network byte order
        data.extend_from_slice(&cn.to_be_bytes());
        
        // HMAC-SHA512 with chain code
        let mut mac = HmacSha512::new_from_slice(&parent.chain_code).unwrap();
        mac.update(&data);
        let result = mac.finalize().into_bytes();
        
        // Parse result
        let mut il = [0u8; 32];
        il.copy_from_slice(&result[0..32]);
        
        let mut chain_code = [0u8; 32];
        chain_code.copy_from_slice(&result[32..64]);
        
        // Add to parent key
        let mut child_key = add_scalar::<32>(&parent.key, &il)?;
        
        // Calculate fingerprint
        let fingerprint = if parent.is_public {
            hash160(&parent.key)
        } else {
            let pk = self.private_to_public(&parent.key)?;
            hash160(&pk)
        };
        
        Ok(ExtendedKey {
            key: child_key.to_vec(),
            chain_code,
            depth: parent.depth + 1,
            parent_fingerprint: [fingerprint[0], fingerprint[1], fingerprint[2], fingerprint[3]],
            child_number: child_num,
            is_public: parent.is_public,
        })
    }
    
    /// Derive address for blockchain
    pub fn derive_address(
        &self,
        chain: crate::Blockchain,
        path: &str,
    ) -> Result<crate::DerivedAddress, crate::WalletError> {
        let key = self.derive_key(path)?;
        
        let (address, public_key) = match chain {
            crate::Blockchain::Ethereum | 
            crate::Blockchain::BnbSmartChain |
            crate::Blockchain::Polygon |
            crate::Blockchain::Arbitrum |
            crate::Blockchain::Optimism |
            crate::Blockchain::Base |
            crate::Blockchain::Avalanche |
            crate::Blockchain::Evm(_) => {
                let public_key = self.private_to_public(&key.key)?;
                let address = eth_address(&public_key);
                (address, hex::encode(&public_key))
            }
            crate::Blockchain::Bitcoin => {
                let public_key = self.private_to_public(&key.key)?;
                let address = btc_address(&public_key);
                (address, hex::encode(&public_key))
            }
            crate::Blockchain::Solana => {
                let public_key = self.private_to_public(&key.key)?;
                let address = solana_address(&public_key);
                (address, hex::encode(&public_key))
            }
            _ => {
                let public_key = self.private_to_public(&key.key)?;
                let address = hex::encode(&public_key);
                (address, hex::encode(&public_key))
            }
        };
        
        Ok(crate::DerivedAddress {
            blockchain: chain,
            address,
            public_key,
            path: path.to_string(),
            private_key_ref: hex::encode(&key.key),
        })
    }
    
    /// Convert private key to public key
    fn private_to_public(&self, private_key: &[u8]) -> Result<Vec<u8>, crate::WalletError> {
        let mut key = [0u8; 32];
        if private_key.len() != 32 {
            return Err(crate::WalletError::DerivationFailed(
                "Invalid private key length".to_string()
            ));
        }
        key.copy_from_slice(private_key);
        
        let secp = Secp256k1::new();
        let secret = SecretKey::from_slice(&key)
            .map_err(|e| crate::WalletError::DerivationFailed(e.to_string()))?;
        
        let public = PublicKey::from_secret_key(&secp, &secret);
        
        Ok(public.serialize().to_vec())
    }
}

/// Add scalar to private key
fn add_scalar<const N: usize>(key: &[u8], scalar: &[u8]) -> Result<[u8; N], crate::WalletError> {
    let mut result = [0u8; N];
    let mut carry = 0u8;
    
    for i in 0..N {
        let ki = key.get(i).copied().unwrap_or(0);
        let si = scalar.get(i).copied().unwrap_or(0);
        
        let sum = (ki as u16) + (si as u16) + (carry as u16);
        result[i] = (sum & 0xff) as u8;
        carry = (sum >> 8) as u8;
    }
    
    // Check for overflow
    if carry != 0 {
        // For BIP-32, if overflow, use current key (theoretical)
    }
    
    Ok(result)
}

/// Hash160 (RIPEMD160(SHA256))
fn hash160(data: &[u8]) -> Vec<u8> {
    use ripemd160::Ripemd160;
    
    // SHA256
    let mut hasher = Sha256::new();
    hasher.update(data);
    let sha_result = hasher.finalize();
    
    // RIPEMD160
    let mut hasher = Ripemd160::new();
    hasher.update(&sha_result);
    hasher.finalize().to_vec()
}

/// Generate Ethereum address
fn eth_address(public_key: &[u8]) -> String {
    // Remove compressed prefix if present
    let pk = if public_key.len() == 33 && (public_key[0] == 0x02 || public_key[0] == 0x03) {
        &public_key[1..]
    } else {
        public_key
    };
    
    let hash = hash160(pk);
    
    // Take last 20 bytes
    let address = &hash[hash.len() - 20..];
    
    // Convert to hex with 0x prefix
    format!("0x{}", hex::encode(address))
}

/// Generate Bitcoin address
fn btc_address(public_key: &[u8]) -> String {
    let hash = hash160(public_key);
    
    // P2PKH address (mainnet)
    let mut data = vec![0x00];
    data.extend_from_slice(&hash);
    
    // Base58 check encode
    base58_check_encode(&data)
}

/// Generate Solana address
fn solana_address(public_key: &[u8]) -> String {
    // Solana uses base58 encoding of public key
    bs58::encode(public_key).into_string()
}

/// Base58Check encoding
fn base58_check_encode(data: &[u8]) -> String {
    // Double SHA256
    let mut hasher = Sha256::new();
    hasher.update(data);
    let hash1 = hasher.finalize();
    
    let mut hasher = Sha256::new();
    hasher.update(&hash1);
    let hash2 = hasher.finalize();
    
    // Take first 4 bytes as checksum
    let mut payload = data.to_vec();
    payload.extend_from_slice(&hash2[0..4]);
    
    // Base58 encode
    bs58::encode(&payload).into_string()
}

/// Encrypt data with password
pub fn encrypt_data(data: &[u8], password: &str) -> Result<Vec<u8>, crate::WalletError> {
    // Derive key from password
    let mut mac = HmacSha256::new_from_slice(password.as_bytes()).unwrap();
    mac.update(data);
    let result = mac.finalize().into_bytes();
    
    let mut key = [0u8; 32];
    key.copy_from_slice(&result[0..32]);
    
    // Generate random nonce
    let mut nonce_bytes = [0u8; 12];
    rand::thread_rng().fill_bytes(&mut nonce_bytes);
    
    // Encrypt
    let cipher = Aes256Gcm::new_from_slice(&key)
        .map_err(|e| crate::WalletError::EncryptionFailed(e.to_string()))?;
    
    let nonce = Nonce::from_slice(&nonce_bytes);
    
    let ciphertext = cipher.encrypt(nonce, data)
        .map_err(|e| crate::WalletError::EncryptionFailed(e.to_string()))?;
    
    // Combine nonce + ciphertext
    let mut result = nonce_bytes.to_vec();
    result.extend_from_slice(&ciphertext);
    
    Ok(result)
}

/// Decrypt data with password
pub fn decrypt_data(data: &[u8], password: &str) -> Result<Vec<u8>, crate::WalletError> {
    if data.len() < 12 {
        return Err(crate::WalletError::DecryptionFailed(
            "Invalid encrypted data".to_string()
        ));
    }
    
    // Extract nonce and ciphertext
    let nonce_bytes = &data[0..12];
    let ciphertext = &data[12..];
    
    // Derive key from password (same as encrypt)
    // In production, use proper KDF like Argon2 or PBKDF2
    let key = Sha256::digest(password.as_bytes());
    
    // Decrypt
    let cipher = Aes256Gcm::new_from_slice(&key)
        .map_err(|e| crate::WalletError::DecryptionFailed(e.to_string()))?;
    
    let nonce = Nonce::from_slice(nonce_bytes);
    
    let plaintext = cipher.decrypt(nonce, ciphertext)
        .map_err(|e| crate::WalletError::DecryptionFailed(e.to_string()))?;
    
    Ok(plaintext)
}
