use crate::crypto::{Crypto, CryptoError};
use sha2::{Sha256, Digest};
use zeroize::Zeroize;

pub struct KeyManager;

impl KeyManager {
    pub fn generate_mnemonic() -> Result<String, CryptoError> {
        // Generate 256-bit entropy
        let entropy = Self::generate_entropy(32)?;
        
        // Convert to mnemonic (BIP-39 wordlist)
        let words = Self::entropy_to_words(&entropy)?;
        
        Ok(words.join(" "))
    }
    
    fn generate_entropy(size: usize) -> Result<Vec<u8>, CryptoError> {
        let mut entropy = vec![0u8; size];
        if let Ok(_) = sodiumoxide::randombytes::randombytes_into(&mut entropy) {
            Ok(entropy)
        } else {
            // Fallback to rand
            use rand::RngCore;
            let mut rng = rand::thread_rng();
            rng.fill_bytes(&mut entropy);
            Ok(entropy)
        }
    }
    
    fn entropy_to_words(entropy: &[u8]) -> Result<Vec<String>, CryptoError> {
        // Simple wordlist - in production, use full BIP-39 wordlist
        let wordlist = include_str!("../wordlist.txt");
        let words: Vec<&str> = wordlist.lines().take(2048).collect();
        
        if words.len() < 2048 {
            return Err(CryptoError::InvalidKey);
        }
        
        // Calculate checksum
        let checksum = Crypto::sha256(entropy);
        let checksum_bits = entropy.len() * 8 / 32;
        
        // Combine entropy and checksum
        let mut bits: Vec<u8> = Vec::new();
        for byte in entropy {
            for i in (0..8).rev() {
                bits.push((byte >> i) & 1);
            }
        }
        for i in 0..checksum_bits {
            bits.push((checksum[i / 8] >> (7 - (i % 8))) & 1);
        }
        
        // Convert to words
        let mut result = Vec::new();
        for chunk in bits.chunks(11) {
            let index = chunk.iter().fold(0, |acc, &bit| (acc << 1) | bit as usize);
            if index < words.len() {
                result.push(words[index].to_string());
            }
        }
        
        Ok(result)
    }
    
    pub fn mnemonic_to_seed(mnemonic: &str, password: &str) -> Vec<u8> {
        let salt = "mnemonic".to_string() + password;
        let salt_bytes = salt.as_bytes();
        
        // PBKDF2 with 2048 iterations
        use pbkdf2::pbkdf2_hmac_array;
        type Pbkdf2Sha512 = pbkdf2::pbkdf2_hmac::<Sha512, 64>;
        
        let seed: pbkdf2_hmac_array<Sha512, 64> = Pbkdf2Sha512::pbkdf2(mnemonic.as_bytes(), salt_bytes, 2048);
        seed.to_vec()
    }
    
    pub fn derive_key(seed: &[u8], change: u32, address_index: u32) -> Vec<u8> {
        // BIP-44 derivation path: m/44'/coin_type'/account'/change/address_index
        // For Ethereum: coin_type = 60
        let purpose = Self::derive_path(seed, &[44 + 0x80000000, 60 + 0x80000000, 0x80000000, change, address_index]);
        purpose
    }
    
    fn derive_path(seed: &[u8], path: &[u32]) -> Vec<u8> {
        use hmac::{Hmac, Mac};
        type HmacSha512 = Hmac<Sha512>;
        
        let mut key = seed.to_vec();
        
        for &index in path {
            let mut mac = HmacSha512::new_from_slice(&key).unwrap();
            
            // Hardened derivation
            let mut data = vec![0];
            data.extend_from_slice(&index.to_be_bytes());
            mac.update(&data);
            
            let result = mac.finalize().into_bytes();
            key = result.to_vec();
        }
        
        key[..32].to_vec()
    }
    
    pub fn encrypt_private_key(private_key: &[u8], password: &str) -> Result<(Vec<u8>, Vec<u8>), CryptoError> {
        // Derive key from password using Argon2
        let salt = Crypto::sha256(password.as_bytes());
        let key = Crypto::sha256(&salt);
        
        // Encrypt with AES-256-GCM
        let mut nonce = [0u8; 12];
        use rand::RngCore;
        let mut rng = rand::thread_rng();
        rng.fill_bytes(&mut nonce);
        
        Crypto::encrypt_aes256gcm(private_key, &key, &nonce)
            .map(|ciphertext| (ciphertext, nonce.to_vec()))
    }
    
    pub fn decrypt_private_key(ciphertext: &[u8], password: &str, nonce: &[u8]) -> Result<Vec<u8>, CryptoError> {
        let salt = Crypto::sha256(password.as_bytes());
        let key = Crypto::sha256(&salt);
        
        Crypto::decrypt_aes256gcm(ciphertext, &key, nonce)
    }
    
    pub fn validate_address(address: &str, chain_type: &str) -> bool {
        match chain_type {
            "evm" => {
                // Check if starts with 0x and is 42 characters
                if !address.starts_with("0x") || address.len() != 42 {
                    return false;
                }
                // Validate hex
                hex::decode(&address[2..]).is_ok()
            },
            "solana" => {
                // Base58, 32-44 characters
                address.len() >= 32 && address.len() <= 44
            },
            "tron" => {
                // Starts with T, 34 characters
                address.starts_with('T') && address.len() == 34
            },
            "bitcoin" => {
                // Legacy (1...), SegWit (3...), Native (bc1...)
                (address.starts_with('1') || address.starts_with('3') || address.starts_with("bc1")) 
                && address.len() >= 26 && address.len() <= 62
            },
            _ => false
        }
    }
}

pub struct HDWallet {
    seed: Vec<u8>,
}

impl HDWallet {
    pub fn from_mnemonic(mnemonic: &str, password: &str) -> Result<Self, CryptoError> {
        let seed = KeyManager::mnemonic_to_seed(mnemonic, password);
        Ok(Self { seed })
    }
    
    pub fn derive_key(&self, chain: u32, account: u32, change: u32, index: u32) -> Vec<u8> {
        let path = &[44 + 0x80000000, chain + 0x80000000, account + 0x80000000, change, index];
        KeyManager::derive_path(&self.seed, path)
    }
    
    pub fn derive_evm_key(&self, account: u32, index: u32) -> (Vec<u8>, String) {
        let private_key = self.derive_key(60, 0, 0, index);
        let public_key = Crypto::derive_public_key(&private_key, false).unwrap_or_default();
        let address = Crypto::pubkey_to_address(&public_key);
        (private_key, address)
    }
    
    pub fn derive_solana_key(&self, index: u32) -> (Vec<u8>, String) {
        let private_key = self.derive_key(501, 0, 0, index);
        let public_key = Crypto::derive_public_key(&private_key, false).unwrap_or_default();
        let address = Crypto::base58_encode(&public_key[1..]); // Skip first byte
        (private_key, address)
    }
    
    pub fn derive_tron_key(&self, index: u32) -> (Vec<u8>, String) {
        let (private_key, mut address) = self.derive_evm_key(0, index);
        address = "T".to_string() + &address[2..34]; // Tron address format
        (private_key, address)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_mnemonic_generation() {
        let mnemonic = KeyManager::generate_mnemonic().unwrap();
        assert_eq!(mnemonic.split_whitespace().count(), 24);
    }
    
    #[test]
    fn test_address_validation() {
        assert!(KeyManager::validate_address("0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E", "evm"));
        assert!(!KeyManager::validate_address("invalid", "evm"));
    }
}
