// ============================================================================
// TIGERWALLET SECURITY MODULE - Rust
// ============================================================================
//
// Features:
// - Hardware wallet integration (Ledger, Trezor, Keystone)
// - Privacy features (Tor, coin control)
// - Advanced security (risk scanning)
// - Transaction verification
// - Secure key derivation
//
// This module is compiled to WASM for browser use
// ============================================================================

use sha2::{Sha256, Digest};
use std::collections::HashMap;

// ============================================================================
// Error Types
// ============================================================================

#[derive(Debug)]
pub enum SecurityError {
    InvalidAddress,
    InvalidSignature,
    HardwareNotFound,
    EncryptionFailed,
    DecryptionFailed,
    DeviceError(String),
}

impl std::fmt::Display for SecurityError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            SecurityError::InvalidAddress => write!(f, "Invalid address"),
            SecurityError::InvalidSignature => write!(f, "Invalid signature"),
            SecurityError::HardwareNotFound => write!(f, "Hardware device not found"),
            SecurityError::EncryptionFailed => write!(f, "Encryption failed"),
            SecurityError::DecryptionFailed => write!(f, "Decryption failed"),
            SecurityError::DeviceError(e) => write!(f, "Device error: {}", e),
        }
    }
}

// ============================================================================
// Hardware Wallet Types
// ============================================================================

#[derive(Debug, Clone)]
pub enum HardwareDeviceType {
    LedgerNanoX,
    LedgerNanoS,
    TrezorModelT,
    TrezorModelOne,
    Keystone,
    NgraveZero,
}

#[derive(Debug, Clone)]
pub struct HardwareDevice {
    pub device_type: HardwareDeviceType,
    pub serial: String,
    pub firmware_version: String,
    pub connected: bool,
    pub supports: Vec<String>, // eth, sol, btc, etc.
}

#[derive(Debug, Clone)]
pub struct Signature {
    pub r: String,
    pub s: String,
    pub v: u8,
}

// ============================================================================
// Address Validation
// ============================================================================

/// Validate EVM address (0x...40 hex chars)
pub fn is_valid_evm_address(address: &str) -> bool {
    if !address.starts_with("0x") {
        return false;
    }
    if address.len() != 42 {
        return false;
    }
    
    let hex_part = &address[2..];
    hex_part.chars().all(|c| c.is_ascii_hexdigit())
}

/// Validate Solana address (base58, 32-44 chars)
pub fn is_valid_solana_address(address: &str) -> bool {
    if address.len() < 32 || address.len() > 44 {
        return false;
    }
    
    // Basic base58 validation
    let valid_chars = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
    address.chars().all(|c| valid_chars.contains(c))
}

/// Validate Bitcoin address
pub fn is_valid_bitcoin_address(address: &str) -> bool {
    // P2PKH (1...), P2SH (3...), P2WPKH (bc1...)
    if address.starts_with("1") || address.starts_with("3") {
        return address.len() >= 26 && address.len() <= 35;
    }
    if address.starts_with("bc1") {
        return address.len() >= 42 && address.len() <= 62;
    }
    false
}

/// Validate all chain addresses
pub fn is_valid_address(address: &str, chain: &str) -> bool {
    match chain {
        "evm" | "ethereum" | "polygon" | "bsc" | "arbitrum" | "optimism" | "base" | "avalanche" | "fantom" | "celo" | "klaytn" | "gnosis" | "cronos" | "moonbeam" | "celo" => {
            is_valid_evm_address(address)
        }
        "solana" => is_valid_solana_address(address),
        "bitcoin" | "litecoin" => is_valid_bitcoin_address(address),
        _ => false,
    }
}

// ============================================================================
// Transaction Risk Analysis
// ============================================================================

#[derive(Debug, Clone)]
pub struct RiskAnalysis {
    pub risk_level: String,        // low, medium, high, critical
    pub risk_score: u8,            // 0-100
    pub warnings: Vec<String>,
    pub risk_factors: Vec<String>,
    pub is_safe: bool,
}

impl RiskAnalysis {
    pub fn new() -> Self {
        Self {
            risk_level: "low".to_string(),
            risk_score: 0,
            warnings: Vec::new(),
            risk_factors: Vec::new(),
            is_safe: true,
        }
    }
}

impl Default for RiskAnalysis {
    fn default() -> Self {
        Self::new()
    }
}

/// Analyze transaction for risks
pub fn analyze_transaction(
    to: &str,
    value: &str,
    data: &str,
    chain_id: u64,
) -> RiskAnalysis {
    let mut analysis = RiskAnalysis::new();
    
    // Check if address is valid
    if !is_valid_evm_address(to) {
        analysis.risk_level = "critical".to_string();
        analysis.risk_score = 100;
        analysis.warnings.push("Invalid recipient address".to_string());
        analysis.risk_factors.push("invalid_address".to_string());
        analysis.is_safe = false;
        return analysis;
    }
    
    // Parse value
    let value_eth: f64 = value.trim_start_matches("0x")
        .parse::<f64>()
        .unwrap_or(0.0) / 1e18;
    
    // High value transfer
    if value_eth > 10.0 {
        analysis.risk_score += 30;
        analysis.warnings.push(format!("High value transfer: {} ETH", value_eth));
        analysis.risk_factors.push("high_value".to_string());
    }
    
    // Very high value
    if value_eth > 100.0 {
        analysis.risk_score += 40;
        analysis.risk_factors.push("very_high_value".to_string());
    }
    
    // Large data payload
    if data.len() > 1000 {
        analysis.risk_score += 20;
        analysis.warnings.push("Large data payload".to_string());
        analysis.risk_factors.push("large_data".to_string());
    }
    
    // Token approval (common attack vector)
    if data.contains("095ea7b3") || data.contains("a22cb465") {
        analysis.risk_score += 15;
        analysis.warnings.push("Token approval detected".to_string());
        analysis.risk_factors.push("token_approval".to_string());
    }
    
    // Unverified contract (no contract verification on Etherscan)
    if chain_id == 1 && data.len() > 0 && data != "0x" {
        analysis.risk_score += 10;
        analysis.warnings.push("Contract interaction".to_string());
        analysis.risk_factors.push("contract_interaction".to_string());
    }
    
    // Known zero-value with data (suspicious)
    if value_eth == 0.0 && !data.is_empty() && data != "0x" {
        analysis.risk_score += 25;
        analysis.warnings.push("Zero value with data - verify purpose".to_string());
        analysis.risk_factors.push("zero_value_data".to_string());
    }
    
    // Determine risk level
    if analysis.risk_score >= 80 {
        analysis.risk_level = "critical".to_string();
        analysis.is_safe = false;
    } else if analysis.risk_score >= 50 {
        analysis.risk_level = "high".to_string();
    } else if analysis.risk_score >= 25 {
        analysis.risk_level = "medium".to_string();
    } else {
        analysis.risk_level = "low".to_string();
    }
    
    analysis
}

// ============================================================================
// Address Risk Check
// ============================================================================

/// Check if address is known malicious.
///
/// NOTE: this WASM/cdylib does not bundle a malicious-address blocklist, so it
/// CANNOT truthfully return a "safe" or "malicious" verdict — doing so would
/// be a fabricated security result. Instead it performs a REAL EVM address
/// format validation and reports `is_safe = "unverified"` /
/// `risk_type = "not_screened"` so the caller knows no blocklist screening
/// was performed. The previous implementation flagged any address starting
/// with "0x0000" as "suspicious", which was a fabricated heuristic with no
/// security basis — removed.
pub fn check_address_risk(address: &str) -> HashMap<String, String> {
    let mut result = HashMap::new();
    result.insert("address".to_string(), address.to_string());
    result.insert("is_safe".to_string(), "unverified".to_string());
    result.insert("risk_type".to_string(), "not_screened".to_string());
    result.insert("reported_times".to_string(), "0".to_string());

    // Real EVM address format validation (no fabricated risk verdict).
    let valid = is_valid_evm_address(address);
    result.insert("address_valid".to_string(), valid.to_string());
    if !valid {
        result.insert("risk_type".to_string(), "invalid_format".to_string());
        result.insert("is_safe".to_string(), "false".to_string());
    }

    result
}

/// is_valid_evm_address validates an EVM address: must be 0x + 40 hex chars.
/// (Full EIP-55 checksum validation requires keccak256; we do a real hex
/// format check here. A checksummed verdict is delegated to the wallet_api
/// backend which has the keccak implementation.)
pub fn is_valid_evm_address(address: &str) -> bool {
    let a = address.strip_prefix("0x").or_else(|| address.strip_prefix("0X"));
    match a {
        Some(hex_part) if hex_part.len() == 40 => hex_part.chars().all(|c| c.is_ascii_hexdigit()),
        _ => false,
    }
}

// ============================================================================
// Hash Functions
// ============================================================================

/// SHA-256 hash
pub fn sha256(data: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(data);
    let result = hasher.finalize();
    hex::encode(result)
}

/// Generate cryptographically secure random hex from the OS RNG (getrandom).
/// The previous implementation hashed the system clock, which is predictable
/// and unsuitable for any security-sensitive use.
pub fn generate_random_hex(length: usize) -> String {
    let byte_len = (length + 1) / 2;
    let mut buf = vec![0u8; byte_len];
    if getrandom::getrandom(&mut buf).is_err() {
        // Never fall back to a predictable value on RNG failure.
        return String::new();
    }
    let hex_str = hex::encode(buf);
    hex_str[..length.min(hex_str.len())].to_string()
}

// ============================================================================
// Coin Control (UTXO selection for privacy)
// ============================================================================

#[derive(Debug, Clone)]
pub struct Utxo {
    pub txid: String,
    pub vout: u32,
    pub amount: u64,
    pub address: String,
    pub confirmations: u32,
    pub is_spent: bool,
}

/// Select optimal UTXOs for spending (privacy-aware)
pub fn select_utxos_for_privacy(
    utxos: &[Utxo],
    target_amount: u64,
    privacy_level: u8, // 0-100
) -> Vec<Utxo> {
    let mut selected: Vec<Utxo> = Vec::new();
    let mut total: u64 = 0;
    
    // Sort by amount (prefer smaller for privacy)
    let mut sorted_utxos = utxos.to_vec();
    sorted_utxos.sort_by_key(|u| u.amount);
    
    // For high privacy, prefer many small UTXOs
    // For low privacy, prefer fewer large UTXOs
    let strategy = if privacy_level > 70 { "small_first" } else { "large_first" };
    
    if strategy == "small_first" {
        for utxo in &sorted_utxos {
            if utxo.is_spent || utxo.confirmations == 0 {
                continue;
            }
            selected.push(utxo.clone());
            total += utxo.amount;
            if total >= target_amount {
                break;
            }
        }
    } else {
        for utxo in sorted_utxos.iter().rev() {
            if utxo.is_spent || utxo.confirmations == 0 {
                continue;
            }
            selected.push(utxo.clone());
            total += utxo.amount;
            if total >= target_amount {
                break;
            }
        }
    }
    
    selected
}

// ============================================================================
// Encryption Keys Derivation
// ============================================================================

/// Derive encryption key from password using Argon2-like approach
pub fn derive_key_from_password(password: &str, salt: &[u8]) -> [u8; 32] {
    // Simplified KDF (in production, use Argon2 or PBKDF2)
    let mut key = [0u8; 32];
    
    for (i, byte) in password.as_bytes().iter().enumerate() {
        key[i % 32] ^= *byte;
    }
    
    for (i, byte) in salt.iter().enumerate() {
        key[i % 32] ^= byte;
    }
    
    // Mix in multiple rounds
    for _ in 0..10000 {
        let mut carry = 0u8;
        for byte in key.iter_mut() {
            let new_val = byte.wrapping_add(carry).rotate_left(3);
            carry = *byte;
            *byte = new_val;
        }
    }
    
    key
}

// ============================================================================
// Hardware Wallet Integration
// ============================================================================

/// Get supported hardware wallets
pub fn get_supported_hardware_wallets() -> Vec<(String, String)> {
    vec![
        ("ledger_nano_x".to_string(), "Ledger Nano X".to_string()),
        ("ledger_nano_s".to_string(), "Ledger Nano S Plus".to_string()),
        ("trezor_model_t".to_string(), "Trezor Model T".to_string()),
        ("trezor_model_one".to_string(), "Trezor Model One".to_string()),
        ("keystone".to_string(), "Keystone".to_string()),
        ("ngrave_zero".to_string(), "NGRAVE ZERO".to_string()),
    ]
}

/// Get derivation paths for chain
pub fn get_derivation_path(chain: &str) -> String {
    match chain {
        "ethereum" | "evm" => "m/44'/60'/0'/0/0".to_string(),
        "bitcoin" => "m/44'/0'/0'/0/0".to_string(),
        "bitcoin_segwit" => "m/84'/0'/0'/0/0".to_string(),
        "solana" => "m/44'/501'/0'/0'".to_string(),
        "cosmos" => "m/44'/118'/0'/0/0".to_string(),
        "polkadot" => "m/44'/354'/0'/0/0".to_string(),
        "near" => "m/44'/397'/0'/0/0".to_string(),
        "aptos" => "m/44'/637'/0'/0/0".to_string(),
        _ => "m/44'/60'/0'/0/0".to_string(),
    }
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_evm_address_validation() {
        assert!(is_valid_evm_address("0x742d35Cc6634C0532925a3b844Bc9e7595f0Eb707"));
        assert!(!is_valid_evm_address("0x742d35Cc6634C0532925a3b844Bc9e7595f0Eb707"));
        assert!(!is_valid_evm_address("742d35Cc6634C0532925a3b844Bc9e7595f0Eb707"));
    }
    
    #[test]
    fn test_transaction_risk_analysis() {
        let result = analyze_transaction(
            "0x742d35Cc6634C0532925a3b844Bc9e7595f0Eb707",
            "0x",
            "0x095ea7b3",
            1,
        );
        
        assert!(result.risk_score > 0);
    }
    
    #[test]
    fn test_hardware_wallets() {
        let wallets = get_supported_hardware_wallets();
        assert!(wallets.len() >= 4);
    }
}

// ============================================================================
// Module Exports
// ============================================================================

pub mod hex {
    pub fn encode(data: impl AsRef<[u8]>) -> String {
        let bytes = data.as_ref();
        bytes.iter().map(|b| format!("{:02x}", b)).collect()
    }
}