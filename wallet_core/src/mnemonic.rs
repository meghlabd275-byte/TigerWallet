//! Mnemonic module - BIP-39 mnemonic phrase generation and validation

use bip39::{Language, Mnemonic};

/// Generate a new mnemonic phrase
pub fn generate_mnemonic(word_count: usize) -> Result<String, MnemonicError> {
    if !matches!(word_count, 12 | 15 | 18 | 21 | 24) {
        return Err(MnemonicError::InvalidWordCount);
    }
    let mnemonic = Mnemonic::generate_in(Language::English, word_count)
        .map_err(|e| MnemonicError::SeedDerivationFailed(e.to_string()))?;
    Ok(mnemonic.to_string())
}

/// Validate a mnemonic phrase
pub fn validate_mnemonic(mnemonic: &str) -> bool {
    Mnemonic::parse_in(Language::English, mnemonic).is_ok()
}

/// Convert mnemonic to seed
pub fn mnemonic_to_seed(mnemonic: &str, passphrase: &str) -> Result<[u8; 64], MnemonicError> {
    let mnem = Mnemonic::parse_in(Language::English, mnemonic)
        .map_err(|e| MnemonicError::InvalidMnemonic(e.to_string()))?;
    
    let seed = mnem.to_seed(passphrase);
    Ok(seed)
}

/// Get entropy from mnemonic
pub fn mnemonic_to_entropy(mnemonic: &str) -> Result<Vec<u8>, MnemonicError> {
    let mnem = Mnemonic::parse_in(Language::English, mnemonic)
        .map_err(|e| MnemonicError::InvalidMnemonic(e.to_string()))?;
    
    let entropy = mnem.to_entropy();
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