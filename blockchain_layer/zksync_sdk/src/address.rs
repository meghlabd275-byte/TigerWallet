//! zkSync Era addresses — standard EVM 20-byte addresses with EIP-55 checksum

use crate::crypto::keccak256;
use crate::types::{Address, ZksyncError};
use std::fmt;

/// EIP-55 checksummed EVM address
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct ZksyncAddress(pub Address);

use serde::{Deserialize, Serialize};

impl ZksyncAddress {
    /// Parse a hex address string (with or without 0x, EIP-55 or lowercase)
    pub fn from_hex(s: &str) -> Result<Self, ZksyncError> {
        let s = s.strip_prefix("0x").unwrap_or(s);
        if s.len() != 40 {
            return Err(ZksyncError::InvalidAddress(format!(
                "expected 40 hex chars, got {}",
                s.len()
            )));
        }
        let raw = hex::decode(s).map_err(|e| ZksyncError::InvalidAddress(e.to_string()))?;
        let mut bytes = [0u8; 20];
        bytes.copy_from_slice(&raw);

        // If mixed-case, enforce EIP-55 checksum; all-lower/all-upper accepted
        if s.chars().any(|c| c.is_ascii_uppercase()) && s.chars().any(|c| c.is_ascii_lowercase())
        {
            let expected = Self(bytes).to_checksum_hex();
            if expected[2..] != *s {
                return Err(ZksyncError::InvalidAddress(
                    "EIP-55 checksum mismatch".to_string(),
                ));
            }
        }
        Ok(Self(bytes))
    }

    /// EIP-55 checksummed hex string
    pub fn to_checksum_hex(&self) -> String {
        let lower = hex::encode(self.0);
        let hash = keccak256(lower.as_bytes());
        let mut out = String::with_capacity(42);
        out.push_str("0x");
        for (i, c) in lower.chars().enumerate() {
            let nibble = (hash[i / 2] >> (if i % 2 == 0 { 4 } else { 0 })) & 0xf;
            if c.is_ascii_hexdigit() && c.is_ascii_alphabetic() && nibble >= 8 {
                out.push(c.to_ascii_uppercase());
            } else {
                out.push(c);
            }
        }
        out
    }

    /// Raw 20 bytes
    pub fn as_bytes(&self) -> &Address {
        &self.0
    }

    pub fn zero() -> Self {
        Self([0u8; 20])
    }
}

impl fmt::Display for ZksyncAddress {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.to_checksum_hex())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn eip55_known_vectors() {
        // Official EIP-55 test vectors
        let cases = [
            "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
            "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359",
            "0xdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB",
            "0xD1220A0cf47c7B9Be7A2E6BA89F429762e7b9aDb",
        ];
        for c in cases {
            let a = ZksyncAddress::from_hex(c).unwrap();
            assert_eq!(a.to_checksum_hex(), c);
        }
    }

    #[test]
    fn rejects_bad_checksum() {
        assert!(ZksyncAddress::from_hex("0x5aAeb6053f3E94C9b9A09f33669435E7Ef1BeAed").is_err());
    }

    #[test]
    fn accepts_lowercase() {
        let a = ZksyncAddress::from_hex("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed").unwrap();
        assert_eq!(
            a.to_checksum_hex(),
            "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"
        );
    }
}
