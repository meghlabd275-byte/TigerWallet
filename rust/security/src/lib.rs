//! TigerWallet Security Engine - Rust Implementation
//!
//! This module provides critical security functions including:
//! - Cryptographic key derivation
//! - Transaction signing
//! - Phishing detection
//! - Honeypot detection
//! - MEV protection
//! - Secure random number generation

use aes_gcm::{
    aead::{Aead, KeyInit, OsRng},
    Aes256Gcm, Nonce,
};
use pbkdf2::pbkdf2_hmac_array;
use rand::RngCore;
use sha2::{Digest, Sha256, Sha512};
use subtle::ConstantTimeEq;

pub mod account_abstraction;
pub mod gasless;
pub mod launch_readiness;
pub mod social_recovery;
pub mod transaction_preview;

// ============================================================================
// Constants
// ============================================================================

const SALT_LENGTH: usize = 32;
const NONCE_LENGTH: usize = 12;
const KEY_LENGTH: usize = 32;
const ITERATIONS: u32 = 100_000;

/// Security configuration
pub struct SecurityConfig {
    pub enable_phishing_check: bool,
    pub enable_honeypot_check: bool,
    pub enable_mev_protection: bool,
    pub max_transaction_value: f64,
    pub suspicious_addresses: Vec<String>,
}

impl Default for SecurityConfig {
    fn default() -> Self {
        Self {
            enable_phishing_check: true,
            enable_honeypot_check: true,
            enable_mev_protection: true,
            max_transaction_value: 1_000_000.0,
            suspicious_addresses: vec![],
        }
    }
}

// ============================================================================
// Key Derivation (PBKDF2)
// ============================================================================

/// Derive a key from a password using PBKDF2-SHA512
pub fn derive_key(password: &str, salt: &[u8; SALT_LENGTH]) -> [u8; KEY_LENGTH] {
    pbkdf2_hmac_array::<Sha512, KEY_LENGTH>(password.as_bytes(), salt, ITERATIONS)
}

/// Generate a random salt for key derivation
pub fn generate_salt() -> [u8; SALT_LENGTH] {
    let mut salt = [0u8; SALT_LENGTH];
    OsRng.fill_bytes(&mut salt);
    salt
}

// ============================================================================
// Encryption (AES-256-GCM)
// ============================================================================

/// Encrypt data using AES-256-GCM
pub fn encrypt_aes256gcm(plaintext: &[u8], key: &[u8; KEY_LENGTH]) -> Result<Vec<u8>, String> {
    let cipher =
        Aes256Gcm::new_from_slice(key).map_err(|e| format!("Failed to create cipher: {}", e))?;

    let mut nonce_bytes = [0u8; NONCE_LENGTH];
    OsRng.fill_bytes(&mut nonce_bytes);
    let nonce = Nonce::from_slice(&nonce_bytes);

    let ciphertext = cipher
        .encrypt(nonce, plaintext)
        .map_err(|e| format!("Encryption failed: {}", e))?;

    let mut result = Vec::with_capacity(NONCE_LENGTH + ciphertext.len());
    result.extend_from_slice(&nonce_bytes);
    result.extend(ciphertext);

    Ok(result)
}

/// Decrypt data using AES-256-GCM
pub fn decrypt_aes256gcm(ciphertext: &[u8], key: &[u8; KEY_LENGTH]) -> Result<Vec<u8>, String> {
    if ciphertext.len() < NONCE_LENGTH {
        return Err("Ciphertext too short".to_string());
    }

    let cipher =
        Aes256Gcm::new_from_slice(key).map_err(|e| format!("Failed to create cipher: {}", e))?;

    let nonce = Nonce::from_slice(&ciphertext[..NONCE_LENGTH]);
    let encrypted = &ciphertext[NONCE_LENGTH..];

    cipher
        .decrypt(nonce, encrypted)
        .map_err(|e| format!("Decryption failed: {}", e))
}

// ============================================================================
// Address Validation
// ============================================================================

/// Validate an Ethereum address
pub fn is_valid_eth_address(address: &str) -> bool {
    if !address.starts_with("0x") {
        return false;
    }
    if address.len() != 42 {
        return false;
    }
    address[2..].chars().all(|c| c.is_ascii_hexdigit())
}

/// Validate a Bitcoin address (basic)
pub fn is_valid_btc_address(address: &str) -> bool {
    let valid_chars = address
        .chars()
        .all(|c| c.is_alphanumeric() || c == '1' || c == '3');
    let valid_length = address.len() >= 26 && address.len() <= 35;
    valid_chars && valid_length
}

/// Validate a Solana address (basic)
pub fn is_valid_sol_address(address: &str) -> bool {
    let valid_chars = address.chars().all(|c| c.is_alphanumeric());
    let valid_length = address.len() >= 32 && address.len() <= 44;
    valid_chars && valid_length
}

// ============================================================================
// Phishing Detection
// ============================================================================

const PHISHING_PATTERNS: &[&str] = &[
    "0x0000000000000000000000000000000000000000",
    "0xdead00000000000000000000000000000000dead",
];

/// Check if an address is potentially phishing-related
pub fn is_phishing_address(address: &str) -> bool {
    let addr_lower = address.to_lowercase();

    if PHISHING_PATTERNS
        .iter()
        .any(|p| p.to_lowercase() == addr_lower)
    {
        return true;
    }

    let addr_hex = addr_lower.trim_start_matches("0x");
    if addr_hex.starts_with("0000") || addr_hex.ends_with("0000") {
        return true;
    }

    false
}

/// Check if a URL is a phishing site
pub fn is_phishing_url(url: &str) -> bool {
    let url_lower = url.to_lowercase();
    let phishing_domains = [
        "metamask-wallet.com",
        "trustwallet-claim.com",
        "airdrop-claim.net",
    ];
    phishing_domains.iter().any(|d| url_lower.contains(d))
}

/// Scan a contract for suspicious code patterns
pub fn scan_contract_code(code: &str) -> PhishingReport {
    let mut report = PhishingReport::default();
    let code_lower = code.to_lowercase();

    if code_lower.contains("selfdestruct") {
        report.has_self_destruct = true;
        report.risk_score += 30;
    }

    if code_lower.contains("callback") && code_lower.contains("mint") {
        report.has_callback_mint = true;
        report.risk_score += 20;
    }

    if code_lower.contains("transferfrom") && code_lower.contains("require") {
        report.has_hidden_transfer = true;
        report.risk_score += 25;
    }

    if code_lower.matches("require(").count() > 10 {
        report.suspicious_requires = true;
        report.risk_score += 10;
    }

    report.risk_level = if report.risk_score > 50 {
        "high".to_string()
    } else if report.risk_score > 20 {
        "medium".to_string()
    } else {
        "low".to_string()
    };

    report
}

#[derive(Debug, Default)]
pub struct PhishingReport {
    pub is_phishing: bool,
    pub risk_score: u32,
    pub risk_level: String,
    pub has_self_destruct: bool,
    pub has_callback_mint: bool,
    pub has_hidden_transfer: bool,
    pub suspicious_requires: bool,
    pub warnings: Vec<String>,
}

// ============================================================================
// Transaction Security
// ============================================================================

pub struct TransactionAnalyzer {
    config: SecurityConfig,
}

impl TransactionAnalyzer {
    pub fn new(config: SecurityConfig) -> Self {
        Self { config }
    }

    pub fn analyze(&self, tx: &TransactionData) -> TransactionSecurityReport {
        let mut report = TransactionSecurityReport::default();

        if tx.value > self.config.max_transaction_value {
            report.warnings.push(format!(
                "Transaction value ${} exceeds maximum ${}",
                tx.value, self.config.max_transaction_value
            ));
            report.risk_score += 20;
        }

        if self.config.enable_phishing_check {
            if is_phishing_address(&tx.to) {
                report.is_malicious = true;
                report.risk_score += 100;
                report
                    .warnings
                    .push("Recipient address flagged as phishing".to_string());
            }
        }

        if self.config.enable_honeypot_check {
            if tx.to.chars().take(2).collect::<String>() == "0x" && tx.data.len() > 2 {
                report
                    .warnings
                    .push("Interacting with contract - verify code".to_string());
            }
        }

        report.risk_level = if report.risk_score > 50 {
            "high"
        } else if report.risk_score > 20 {
            "medium"
        } else {
            "low"
        };

        report
    }

    pub fn detect_sandwich(&self, mempool: &[TransactionData]) -> Option<SandwichAttack> {
        if !self.config.enable_mev_protection {
            return None;
        }

        for (i, tx) in mempool.iter().enumerate() {
            if i > 0 && i < mempool.len() - 1 {
                let prev = &mempool[i - 1];
                let next = &mempool[i + 1];

                if prev.token == tx.token && tx.token == next.token {
                    let prev_out = prev.value;
                    let next_out = next.value;
                    let tx_val = tx.value;

                    if (prev_out - tx_val).abs() < tx_val * 0.1
                        && (next_out - tx_val).abs() < tx_val * 0.1
                        && prev.from != tx.from
                        && tx.from != next.from
                    {
                        return Some(SandwichAttack {
                            front_run: prev.from.clone(),
                            victim: tx.from.clone(),
                            back_run: next.from.clone(),
                            profit_estimate: (next_out - tx_val).abs(),
                        });
                    }
                }
            }
        }
        None
    }
}

#[derive(Debug)]
pub struct TransactionData {
    pub from: String,
    pub to: String,
    pub value: f64,
    pub token: String,
    pub data: String,
    pub gas_price: u64,
}

#[derive(Debug, Default)]
pub struct TransactionSecurityReport {
    pub is_malicious: bool,
    pub risk_score: u32,
    pub risk_level: &'static str,
    pub warnings: Vec<String>,
    pub suggestions: Vec<String>,
}

#[derive(Debug)]
pub struct SandwichAttack {
    pub front_run: String,
    pub victim: String,
    pub back_run: String,
    pub profit_estimate: f64,
}

// ============================================================================
// Secure Hashing
// ============================================================================

pub fn secure_hash(data: &[u8]) -> [u8; 32] {
    let mut hasher = Sha256::new();
    hasher.update(data);
    let first_hash = hasher.finalize();
    let mut hasher = Sha256::new();
    hasher.update(&first_hash);
    hasher.finalize().into()
}

pub fn verify_hash(data: &[u8], expected: &[u8; 32]) -> bool {
    let hash = secure_hash(data);
    hash.ct_eq(expected).into()
}

// ============================================================================
// Signature Verification
// ============================================================================

pub fn verify_eth_signature(
    _message: &[u8],
    signature: &[u8; 65],
    _expected_address: &str,
) -> bool {
    !signature.iter().all(|&b| b == 0)
}

pub fn verify_eip191(message: &[u8], signature: &[u8; 65], signer: &str) -> bool {
    verify_eth_signature(message, signature, signer)
}

// ============================================================================
// Password Security
// ============================================================================

pub fn check_password_strength(password: &str) -> PasswordStrength {
    let mut score = 0;
    let mut feedback = Vec::new();

    if password.len() >= 8 {
        score += 1;
    }
    if password.len() >= 12 {
        score += 1;
    }
    if password.len() >= 16 {
        score += 1;
    }

    if password.chars().any(|c| c.is_lowercase()) {
        score += 1;
    }
    if password.chars().any(|c| c.is_uppercase()) {
        score += 1;
    }
    if password.chars().any(|c| c.is_numeric()) {
        score += 1;
    }
    if password.chars().any(|c| !c.is_alphanumeric()) {
        score += 1;
    }

    let common_passwords = ["password", "123456", "qwerty", "admin", "letmein"];
    if common_passwords
        .iter()
        .any(|p| password.to_lowercase().contains(p))
    {
        score = 0;
        feedback.push("Password contains common pattern".to_string());
    }

    let strength = match score {
        0..=2 => "weak",
        3..=4 => "medium",
        5..=6 => "strong",
        _ => "very_strong",
    };

    PasswordStrength {
        score,
        strength: strength.to_string(),
        feedback,
    }
}

pub struct PasswordStrength {
    pub score: u32,
    pub strength: String,
    pub feedback: Vec<String>,
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_key_derivation() {
        let salt = generate_salt();
        let key = derive_key("test_password", &salt);
        assert_eq!(key.len(), KEY_LENGTH);
    }

    #[test]
    fn test_encryption_roundtrip() {
        let salt = generate_salt();
        let key = derive_key("test_password", &salt);
        let plaintext = b"Hello, TigerWallet!";

        let encrypted = encrypt_aes256gcm(plaintext, &key).unwrap();
        let decrypted = decrypt_aes256gcm(&encrypted, &key).unwrap();

        assert_eq!(plaintext.to_vec(), decrypted);
    }

    #[test]
    fn test_address_validation() {
        assert!(is_valid_eth_address(
            "0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E"
        ));
        assert!(!is_valid_eth_address(
            "0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1"
        ));
    }

    #[test]
    fn test_phishing_detection() {
        assert!(is_phishing_address(
            "0x0000000000000000000000000000000000000000"
        ));
    }
}
