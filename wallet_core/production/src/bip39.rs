/**
 * BIP-39 Mnemonic Phrase Implementation
 * 
 * Production-ready BIP-39 implementation using well-audited libraries.
 * Supports mnemonic generation, validation, and seed derivation.
 */

use bip39::{Mnemonic as Bip39Mnemonic, MnemonicType, MnemonicError, Seed};
use bip39::wordlist::ENGLISH_WORDLIST;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum MnemonicError {
    #[error("Invalid mnemonic: {0}")]
    Invalid(String),
    #[error("Wordlist error: {0}")]
    WordlistError(String),
    #[error("Seed derivation error: {0}")]
    SeedError(String),
}

pub type Result<T> = std::result::Result<T, MnemonicError>;

/// Mnemonic word count options
pub const WORD_COUNTS: &[usize] = &[12, 15, 18, 21, 24];

/// Mnemonic type
pub enum Wordlist {
    English,
    // Other wordlists can be added
}

impl Wordlist {
    pub fn as_bip39(&self) -> &'static str {
        match self {
            Wordlist::English => "English",
        }
    }
}

/// BIP-39 Mnemonic
pub struct Mnemonic {
    mnemonic: Bip39Mnemonic,
}

impl Mnemonic {
    /// Generate new mnemonic with specified word count
    pub fn new(wordlist: Wordlist, words: usize) -> Result<Self> {
        let word_count = match words {
            12 => MnemonicType::Words12,
            15 => MnemonicType::Words15,
            18 => MnemonicType::Words18,
            21 => MnemonicType::Words21,
            24 => MnemonicType::Words24,
            _ => return Err(MnemonicError::Invalid(format!("Invalid word count: {}", words))),
        };
        
        let mnemonic = Bip39Mnemonic::new(word_count, ENGLISH_WORDLIST)
            .map_err(|e| MnemonicError::Invalid(e.to_string()))?;
        
        Ok(Self { mnemonic })
    }
    
    /// Create from existing phrase
    pub fn from_phrase(phrase: &str) -> Result<Self> {
        let mnemonic = Bip39Mnemonic::from_phrase(phrase, ENGLISH_WORDLIST)
            .map_err(|e| MnemonicError::Invalid(e.to_string()))?;
        
        Ok(Self { mnemonic })
    }
    
    /// Validate mnemonic phrase
    pub fn validate(phrase: &str, wordlist: Wordlist) -> Result<Self> {
        Self::from_phrase(phrase)
    }
    
    /// Get mnemonic words
    pub fn words(&self) -> String {
        self.mnemonic.phrase().to_string()
    }
    
    /// Get entropy bytes
    pub fn entropy(&self) -> Vec<u8> {
        self.mnemonic.entropy().to_vec()
    }
}

/// Convert mnemonic to 64-byte seed
pub fn mnemonic_to_seed(mnemonic: &str, password: &str) -> Result<[u8; 64]> {
    let mnem = Bip39Mnemonic::from_phrase(mnemonic, ENGLISH_WORDLIST)
        .map_err(|e| MnemonicError::Invalid(e.to_string()))?;
    
    let seed = mnem.to_seed(password);
    
    let mut result = [0u8; 64];
    result.copy_from_slice(&seed);
    
    Ok(result)
}

/// Get entropy from mnemonic
pub fn mnemonic_to_entropy(mnemonic: &str) -> Result<Vec<u8>> {
    let mnem = Bip39Mnemonic::from_phrase(mnemonic, ENGLISH_WORDLIST)
        .map_err(|e| MnemonicError::Invalid(e.to_string()))?;
    
    Ok(mnem.entropy().to_vec())
}

/// Create mnemonic from entropy
pub fn entropy_to_mnemonic(entropy: &[u8]) -> Result<String> {
    let mnem = Bip39Mnemonic::from_entropy(entropy, ENGLISH_WORDLIST)
        .map_err(|e| MnemonicError::Invalid(e.to_string()))?;
    
    Ok(mnem.phrase().to_string())
}

/// Validate mnemonic without creating object
pub fn is_valid_mnemonic(phrase: &str) -> bool {
    Bip39Mnemonic::from_phrase(phrase, ENGLISH_WORDLIST).is_ok()
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_generate_12_words() {
        let mnem = Mnemonic::new(Wordlist::English, 12).unwrap();
        assert_eq!(mnem.words().split_whitespace().count(), 12);
    }
    
    #[test]
    fn test_generate_24_words() {
        let mnem = Mnemonic::new(Wordlist::English, 24).unwrap();
        assert_eq!(mnem.words().split_whitespace().count(), 24);
    }
    
    #[test]
    fn test_roundtrip() {
        let mnem = Mnemonic::new(Wordlist::English, 12).unwrap();
        let phrase = mnem.words();
        
        let validated = Mnemonic::validate(&phrase, Wordlist::English);
        assert!(validated.is_ok());
    }
    
    #[test]
    fn test_seed_derivation() {
        let mnem = Mnemonic::new(Wordlist::English, 12).unwrap();
        let seed = mnemonic_to_seed(mnem.words(), "password").unwrap();
        assert_eq!(seed.len(), 64);
    }
    
    #[test]
    fn test_validation() {
        let valid = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";
        assert!(is_valid_mnemonic(valid));
        
        let invalid = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon invalidword";
        assert!(!is_valid_mnemonic(invalid));
    }
}
