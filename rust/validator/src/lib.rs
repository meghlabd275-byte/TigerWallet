//! Transaction validation: structural checks, replay protection and a risk score.
//!
//! # Fail-closed parsing
//!
//! Every numeric field is parsed into an integer and rejected when malformed.
//! The previous implementation used `parse().unwrap_or(0.0)`, which turned an
//! unparseable value into zero: a transaction carrying `"value": "abc"` or
//! `"value": "-1"` passed the maximum-value check because the check compared
//! zero against the limit. Malformed input now produces an error.
//!
//! # Value limits
//!
//! Amounts are handled as `u128` wei and converted to USD only through a price
//! the caller supplies. The previous version divided by 1e18 and multiplied by
//! a hardcoded 3500, so the limit meant nothing on any chain whose native
//! asset is not priced near ETH.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::RwLock;
use std::time::{SystemTime, UNIX_EPOCH};

/// A transaction submitted for validation.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub hash: String,
    pub from: String,
    pub to: String,
    /// Amount in wei, as a decimal string.
    pub value: String,
    pub gas: String,
    pub gas_price: String,
    pub nonce: u64,
    pub data: String,
    pub chain_id: u64,
    pub timestamp: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ValidationResult {
    pub valid: bool,
    pub errors: Vec<String>,
    pub warnings: Vec<String>,
    pub risk_score: f64,
}

/// Why a transaction was rejected outright.
#[derive(Debug, PartialEq, Eq)]
pub enum ValidationError {
    /// A numeric field was not a valid unsigned integer.
    MalformedField(String),
    /// The address was not a 20-byte hex string.
    InvalidAddress(String),
    /// No price is configured for the chain, so the value limit cannot be
    /// applied. Validation fails closed rather than skipping the check.
    MissingPrice(u64),
}

/// Limits applied to every transaction.
#[derive(Debug, Clone)]
pub struct ValidatorConfig {
    /// Maximum transaction value in USD.
    pub max_value_usd: f64,
    /// Maximum gas price in gwei.
    pub max_gas_price_gwei: f64,
    /// How far in the future a timestamp may be, in seconds.
    pub max_clock_skew_secs: u64,
    /// How old a timestamp may be, in seconds.
    pub max_age_secs: u64,
}

impl Default for ValidatorConfig {
    fn default() -> Self {
        Self {
            max_value_usd: 1_000_000.0,
            max_gas_price_gwei: 500.0,
            max_clock_skew_secs: 60,
            max_age_secs: 300,
        }
    }
}

/// Wei per unit of the native asset.
const WEI_PER_UNIT: f64 = 1e18;
/// Wei per gwei.
const WEI_PER_GWEI: f64 = 1e9;

pub struct TransactionValidator {
    blocklist: RwLock<HashMap<String, String>>,
    allowlist: RwLock<HashMap<String, ()>>,
    /// Highest nonce accepted per sender.
    nonces: RwLock<HashMap<String, u64>>,
    /// Native-asset price in USD per chain id.
    prices: RwLock<HashMap<u64, f64>>,
    config: ValidatorConfig,
}

impl Default for TransactionValidator {
    fn default() -> Self {
        Self::new(ValidatorConfig::default())
    }
}

impl TransactionValidator {
    pub fn new(config: ValidatorConfig) -> Self {
        Self {
            blocklist: RwLock::new(HashMap::new()),
            allowlist: RwLock::new(HashMap::new()),
            nonces: RwLock::new(HashMap::new()),
            prices: RwLock::new(HashMap::new()),
            config,
        }
    }

    /// Record the native-asset price used to value transactions on a chain.
    ///
    /// Without a price the value limit cannot be evaluated, and validation of
    /// any transaction on that chain fails closed.
    pub fn set_price(&self, chain_id: u64, usd_price: f64) {
        if let Ok(mut prices) = self.prices.write() {
            if usd_price.is_finite() && usd_price > 0.0 {
                prices.insert(chain_id, usd_price);
            } else {
                prices.remove(&chain_id);
            }
        }
    }

    pub fn block(&self, address: &str) {
        if let Ok(mut blocklist) = self.blocklist.write() {
            blocklist.insert(normalize_address(address), "blocked".to_string());
        }
    }

    pub fn unblock(&self, address: &str) {
        if let Ok(mut blocklist) = self.blocklist.write() {
            blocklist.remove(&normalize_address(address));
        }
    }

    /// Exempt an address from the blocklist. The allowlist wins over the
    /// blocklist so a treasury address can be restored without unblocking it
    /// for everyone.
    pub fn allow(&self, address: &str) {
        if let Ok(mut allowlist) = self.allowlist.write() {
            allowlist.insert(normalize_address(address), ());
        }
    }

    pub fn is_blocked(&self, address: &str) -> bool {
        let address = normalize_address(address);
        let allowed = self
            .allowlist
            .read()
            .map(|a| a.contains_key(&address))
            .unwrap_or(false);
        if allowed {
            return false;
        }
        self.blocklist
            .read()
            .map(|b| b.contains_key(&address))
            .unwrap_or(false)
    }

    /// Validate a transaction, returning a structured error for anything that
    /// makes the result impossible to compute.
    pub fn validate(&self, tx: &Transaction) -> Result<ValidationResult, ValidationError> {
        let mut errors: Vec<String> = Vec::new();
        let mut warnings: Vec<String> = Vec::new();

        // --- Addresses ---
        let from = normalize_address(&tx.from);
        let to = normalize_address(&tx.to);
        if !is_valid_address(&from) {
            return Err(ValidationError::InvalidAddress(tx.from.clone()));
        }
        if !is_valid_address(&to) {
            return Err(ValidationError::InvalidAddress(tx.to.clone()));
        }
        if !is_valid_hash(&tx.hash) {
            return Err(ValidationError::MalformedField("hash".into()));
        }

        // --- Numeric fields, parsed strictly ---
        let value = parse_u128(&tx.value)
            .ok_or_else(|| ValidationError::MalformedField("value".into()))?;
        let gas = parse_u128(&tx.gas)
            .ok_or_else(|| ValidationError::MalformedField("gas".into()))?;
        let gas_price = parse_u128(&tx.gas_price)
            .ok_or_else(|| ValidationError::MalformedField("gas_price".into()))?;

        // --- Blocklist ---
        if self.is_blocked(&from) {
            errors.push("sender is blocklisted".into());
        }
        if self.is_blocked(&to) {
            errors.push("recipient is blocklisted".into());
        }

        // --- Nonce / replay protection ---
        if let Ok(nonces) = self.nonces.read() {
            if let Some(&last) = nonces.get(&from) {
                if tx.nonce < last {
                    errors.push(format!(
                        "nonce {} is below the last accepted nonce {} (possible replay)",
                        tx.nonce, last
                    ));
                } else if tx.nonce > last + 1 {
                    warnings.push(format!(
                        "nonce {} leaves a gap after {}",
                        tx.nonce, last
                    ));
                }
            }
        }

        // --- Value limit, in USD at the configured price ---
        let price = self
            .prices
            .read()
            .ok()
            .and_then(|p| p.get(&tx.chain_id).copied())
            .ok_or(ValidationError::MissingPrice(tx.chain_id))?;

        let value_units = value as f64 / WEI_PER_UNIT;
        let value_usd = value_units * price;
        if value_usd > self.config.max_value_usd {
            errors.push(format!(
                "value ${:.2} exceeds the ${:.2} limit",
                value_usd, self.config.max_value_usd
            ));
        }

        // --- Gas ---
        let gas_gwei = gas_price as f64 / WEI_PER_GWEI;
        if gas_gwei > self.config.max_gas_price_gwei {
            errors.push(format!(
                "gas price {:.2} gwei exceeds the {:.2} gwei limit",
                gas_gwei, self.config.max_gas_price_gwei
            ));
        }
        if gas == 0 {
            errors.push("gas limit is zero".into());
        }
        if gas_price == 0 {
            warnings.push("gas price is zero".into());
        }

        // --- Timestamp ---
        let now = current_timestamp();
        if tx.timestamp > now + self.config.max_clock_skew_secs {
            errors.push("timestamp is too far in the future".into());
        } else if now > tx.timestamp + self.config.max_age_secs {
            errors.push("timestamp is too old".into());
        }

        // --- Contract interaction ---
        if tx.data != "0x" && !tx.data.is_empty() && !tx.data.starts_with("0x") {
            errors.push("calldata is not hex".into());
        }

        let valid = errors.is_empty();

        // Only a valid transaction advances the nonce, so a rejected
        // transaction cannot consume a nonce the sender has not used.
        if valid {
            if let Ok(mut nonces) = self.nonces.write() {
                let current = nonces.get(&from).copied().unwrap_or(0);
                if tx.nonce + 1 > current {
                    nonces.insert(from.clone(), tx.nonce + 1);
                }
            }
        }

        let risk_score = ((errors.len() as f64 * 30.0) + (warnings.len() as f64 * 10.0)).min(100.0);

        Ok(ValidationResult {
            valid,
            errors,
            warnings,
            risk_score,
        })
    }
}

// ============================================================================
// Helpers
// ============================================================================

fn current_timestamp() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

/// Lowercase an address so `0xABC...` and `0xabc...` are the same entry.
fn normalize_address(address: &str) -> String {
    address.trim().to_ascii_lowercase()
}

/// An address is `0x` followed by exactly 40 hex digits.
fn is_valid_address(address: &str) -> bool {
    match address.strip_prefix("0x") {
        Some(hex) => hex.len() == 40 && hex.chars().all(|c| c.is_ascii_hexdigit()),
        None => false,
    }
}

/// A transaction hash is `0x` followed by exactly 64 hex digits.
fn is_valid_hash(hash: &str) -> bool {
    match hash.strip_prefix("0x") {
        Some(hex) => hex.len() == 64 && hex.chars().all(|c| c.is_ascii_hexdigit()),
        None => false,
    }
}

/// Parse a decimal unsigned integer, rejecting signs, whitespace and overflow.
fn parse_u128(s: &str) -> Option<u128> {
    let s = s.trim();
    if s.is_empty() || !s.chars().all(|c| c.is_ascii_digit()) {
        return None;
    }
    s.parse::<u128>().ok()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn valid_address() -> String {
        format!("0x{}", "a".repeat(40))
    }

    fn valid_hash() -> String {
        format!("0x{}", "b".repeat(64))
    }

    fn validator() -> TransactionValidator {
        let v = TransactionValidator::new(ValidatorConfig::default());
        v.set_price(1, 3500.0);
        v
    }

    fn tx(from: &str, to: &str, value: &str, nonce: u64) -> Transaction {
        Transaction {
            hash: valid_hash(),
            from: from.to_string(),
            to: to.to_string(),
            value: value.to_string(),
            gas: "21000".into(),
            gas_price: "20000000000".into(),
            nonce,
            data: "0x".into(),
            chain_id: 1,
            timestamp: current_timestamp(),
        }
    }

    #[test]
    fn accepts_a_well_formed_transaction() {
        let v = validator();
        let result = v
            .validate(&tx(&valid_address(), &valid_address(), "1000000000000000000", 0))
            .unwrap();

        assert!(result.valid, "errors: {:?}", result.errors);
        assert_eq!(result.risk_score, 0.0);
    }

    #[test]
    fn malformed_value_is_rejected_rather_than_treated_as_zero() {
        let v = validator();
        // The old code used unwrap_or(0.0) here, so these all passed.
        for bad in ["abc", "-1", "1.5", "", " ", "0x10"] {
            let result = v.validate(&tx(&valid_address(), &valid_address(), bad, 0));
            assert_eq!(
                result,
                Err(ValidationError::MalformedField("value".into())),
                "value {:?} should be rejected",
                bad
            );
        }
    }

    #[test]
    fn malformed_gas_fields_are_rejected() {
        let v = validator();
        let mut t = tx(&valid_address(), &valid_address(), "1", 0);
        t.gas = "not-a-number".into();
        assert_eq!(
            v.validate(&t),
            Err(ValidationError::MalformedField("gas".into()))
        );

        let mut t = tx(&valid_address(), &valid_address(), "1", 0);
        t.gas_price = "-5".into();
        assert_eq!(
            v.validate(&t),
            Err(ValidationError::MalformedField("gas_price".into()))
        );
    }

    #[test]
    fn invalid_addresses_and_hashes_are_rejected() {
        let v = validator();
        let mut t = tx(&valid_address(), &valid_address(), "1", 0);
        t.from = "0x1234".into();
        assert!(matches!(
            v.validate(&t),
            Err(ValidationError::InvalidAddress(_))
        ));

        let mut t = tx(&valid_address(), &valid_address(), "1", 0);
        t.to = "not-an-address".into();
        assert!(matches!(
            v.validate(&t),
            Err(ValidationError::InvalidAddress(_))
        ));

        let mut t = tx(&valid_address(), &valid_address(), "1", 0);
        t.hash = "0xdeadbeef".into();
        assert_eq!(
            v.validate(&t),
            Err(ValidationError::MalformedField("hash".into()))
        );
    }

    #[test]
    fn value_limit_uses_the_configured_price() {
        let v = validator();
        // 1 ETH at $3500 is under the $1M limit.
        assert!(v
            .validate(&tx(&valid_address(), &valid_address(), "1000000000000000000", 0))
            .unwrap()
            .valid);

        // 1000 ETH at $3500 is $3.5M, over the limit.
        let result = v
            .validate(&tx(
                &valid_address(),
                &valid_address(),
                "1000000000000000000000",
                0,
            ))
            .unwrap();
        assert!(!result.valid);
        assert!(result.errors.iter().any(|e| e.contains("exceeds")));
    }

    #[test]
    fn validation_fails_closed_without_a_price() {
        let v = TransactionValidator::new(ValidatorConfig::default());
        // No price configured for chain 1.
        let result = v.validate(&tx(&valid_address(), &valid_address(), "1", 0));
        assert_eq!(result, Err(ValidationError::MissingPrice(1)));
    }

    #[test]
    fn a_non_positive_price_is_not_accepted() {
        let v = TransactionValidator::new(ValidatorConfig::default());
        v.set_price(1, 0.0);
        assert_eq!(
            v.validate(&tx(&valid_address(), &valid_address(), "1", 0)),
            Err(ValidationError::MissingPrice(1))
        );

        v.set_price(1, -5.0);
        assert_eq!(
            v.validate(&tx(&valid_address(), &valid_address(), "1", 0)),
            Err(ValidationError::MissingPrice(1))
        );

        v.set_price(1, f64::NAN);
        assert_eq!(
            v.validate(&tx(&valid_address(), &valid_address(), "1", 0)),
            Err(ValidationError::MissingPrice(1))
        );
    }

    #[test]
    fn blocklisted_sender_is_rejected() {
        let v = validator();
        let sender = valid_address();
        v.block(&sender);

        let result = v.validate(&tx(&sender, &valid_address(), "1", 0)).unwrap();
        assert!(!result.valid);
        assert!(result.errors.iter().any(|e| e.contains("blocklisted")));
        assert!(v.is_blocked(&sender));
    }

    #[test]
    fn blocklist_is_case_insensitive() {
        let v = validator();
        v.block(&format!("0x{}", "A".repeat(40)));
        assert!(v.is_blocked(&format!("0x{}", "a".repeat(40))));
    }

    #[test]
    fn allowlist_overrides_the_blocklist() {
        let v = validator();
        let address = valid_address();
        v.block(&address);
        assert!(v.is_blocked(&address));

        v.allow(&address);
        assert!(!v.is_blocked(&address));

        v.unblock(&address);
        assert!(!v.is_blocked(&address));
    }

    #[test]
    fn nonce_regression_is_flagged_as_a_replay() {
        let v = validator();
        let sender = valid_address();

        assert!(v.validate(&tx(&sender, &valid_address(), "1", 5)).unwrap().valid);

        let result = v.validate(&tx(&sender, &valid_address(), "1", 3)).unwrap();
        assert!(!result.valid);
        assert!(result.errors.iter().any(|e| e.contains("replay")));
    }

    #[test]
    fn a_rejected_transaction_does_not_advance_the_nonce() {
        let v = validator();
        let sender = valid_address();
        // Over the gas limit, so it is rejected.
        let mut t = tx(&sender, &valid_address(), "1", 0);
        t.gas_price = "999000000000".into();
        assert!(!v.validate(&t).unwrap().valid);

        // The same nonce must still be usable.
        assert!(v.validate(&tx(&sender, &valid_address(), "1", 0)).unwrap().valid);
    }

    #[test]
    fn nonce_gap_produces_a_warning_not_an_error() {
        let v = validator();
        let sender = valid_address();
        v.validate(&tx(&sender, &valid_address(), "1", 0)).unwrap();

        let result = v.validate(&tx(&sender, &valid_address(), "1", 5)).unwrap();
        assert!(result.valid);
        assert!(result.warnings.iter().any(|w| w.contains("gap")));
        assert!(result.risk_score > 0.0);
    }

    #[test]
    fn excessive_gas_price_is_rejected() {
        let v = validator();
        let mut t = tx(&valid_address(), &valid_address(), "1", 0);
        t.gas_price = "999000000000".into(); // 999 gwei
        let result = v.validate(&t).unwrap();

        assert!(!result.valid);
        assert!(result.errors.iter().any(|e| e.contains("gwei")));
    }

    #[test]
    fn zero_gas_limit_is_rejected() {
        let v = validator();
        let mut t = tx(&valid_address(), &valid_address(), "1", 0);
        t.gas = "0".into();
        assert!(!v.validate(&t).unwrap().valid);
    }

    #[test]
    fn stale_and_future_timestamps_are_rejected() {
        let v = validator();
        let now = current_timestamp();

        let mut old = tx(&valid_address(), &valid_address(), "1", 0);
        old.timestamp = now - 3600;
        assert!(!v.validate(&old).unwrap().valid);

        let mut future = tx(&valid_address(), &valid_address(), "1", 0);
        future.timestamp = now + 3600;
        assert!(!v.validate(&future).unwrap().valid);
    }

    #[test]
    fn non_hex_calldata_is_rejected() {
        let v = validator();
        let mut t = tx(&valid_address(), &valid_address(), "1", 0);
        t.data = "transfer(0x1)".into();
        assert!(!v.validate(&t).unwrap().valid);
    }

    #[test]
    fn risk_score_is_bounded() {
        let v = validator();
        let sender = valid_address();
        v.block(&sender);

        let result = v.validate(&tx(&sender, &valid_address(), "1", 0)).unwrap();
        assert!(result.risk_score <= 100.0);
        assert!(result.risk_score > 0.0);
    }

    #[test]
    fn u128_parser_rejects_non_digits() {
        assert_eq!(parse_u128("0"), Some(0));
        assert_eq!(parse_u128("12345"), Some(12345));
        assert_eq!(parse_u128(" 42 "), Some(42));
        assert_eq!(parse_u128("-1"), None);
        assert_eq!(parse_u128("+1"), None);
        assert_eq!(parse_u128("1e5"), None);
        assert_eq!(parse_u128(""), None);
        // Larger than u128::MAX.
        assert_eq!(parse_u128(&"9".repeat(40)), None);
    }
}
