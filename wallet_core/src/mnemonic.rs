//! Mnemonic module - BIP-39 mnemonic phrase generation and validation

use bip39::{Mnemonic, MnemonicType, Language};
use rand::thread_rng;
use sha2::{Sha256, Digest};

/// Generate a new mnemonic phrase
pub fn generate_mnemonic(word_count: usize) -> Result<String, MnemonicError> {
    let mnemonic_type = match word_count {
        12 => MnemonicType::Words12,
        15 => MnemonicType::Words15,
        18 => MnemonicType::Words18,
        21 => MnemonicType::Words21,
        24 => MnemonicType::Words24,
        _ => return Err(MnemonicError::InvalidWordCount),
    };
    
    let mut rng = thread_rng();
    let mnemonic = Mnemonic::new(mnemonic_type, Language::English, &mut rng)?;
    Ok(mnemonic.phrase().to_string())
}

/// Validate a mnemonic phrase
pub fn validate_mnemonic(mnemonic: &str) -> bool {
    match Mnemonic::from_phrase(mnemonic, Language::English) {
        Ok(_) => true,
        Err(_) => false,
    }
}

/// Convert mnemonic to seed
pub fn mnemonic_to_seed(mnemonic: &str, passphrase: &str) -> Result<[u8; 64], MnemonicError> {
    let mnem = Mnemonic::from_phrase(mnemonic, Language::English)
        .map_err(|e| MnemonicError::InvalidMnemonic(e.to_string()))?;
    
    let seed = mnem.to_seed(passphrase);
    Ok(seed)
}

/// Get entropy from mnemonic
pub fn mnemonic_to_entropy(mnemonic: &str) -> Result<Vec<u8>, MnemonicError> {
    let mnem = Mnemonic::from_phrase(mnemonic, Language::English)
        .map_err(|e| MnemonicError::InvalidMnemonic(e.to_string()))?;
    
    let entropy = mnem.entropy();
    Ok(entropy.to_vec())
}

/// Mnemonic errors
#[derive(Debug, thiserror::Error)]
pub enum MnemonicError {
    #[error("Invalid word count")]
    InvalidWordCount,
    
    #[error("Invalid mnemonic: {0}")]
    InvalidMnemonic(String),
    
    #[error("Seed derivation failed: {0}")]
    SeedDerivationFailed(String),
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_generate_mnemonic() {
        let mnemonic = generate_mnemonic(12).unwrap();
        assert_eq!(mnemonic.split_whitespace().count(), 12);
    }
    
    #[test]
    fn test_validate_mnemonic() {
        let mnemonic = generate_mnemonic(12).unwrap();
        assert!(validate_mnemonic(&mnemonic));
    }
    
    #[test]
    fn test_mnemonic_to_seed() {
        let mnemonic = generate_mnemonic(12).unwrap();
        let seed = mnemonic_to_seed(&mnemonic, "").unwrap();
        assert_eq!(seed.len(), 64);
    }
}