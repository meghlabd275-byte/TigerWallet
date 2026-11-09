use parking_lot::RwLock;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
pub enum MEVType {
    #[default]
    None,
    FrontRun,
    Sandwich,
    BackRun,
    FlashLoan,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MEVDetection {
    pub mev_type: MEVType,
    pub confidence: u8,
}

impl MEVDetection {
    pub fn clean() -> Self {
        Self { mev_type: MEVType::None, confidence: 100 }
    }
}

/// Flags transactions sent by addresses the operator has identified as
/// searcher/sandwich bots.
///
/// The list is empty by default: shipping a guessed address would mark real
/// users as attackers, so the operator loads the set from its own threat feed
/// (see `load_from_list`).
pub struct MEVProtection {
    known_bots: Arc<RwLock<HashMap<String, bool>>>,
}

impl MEVProtection {
    pub fn new() -> Self {
        Self { known_bots: Arc::new(RwLock::new(HashMap::new())) }
    }

    /// Replaces the known-bot set. Addresses are normalised to lower case so
    /// lookups match regardless of EIP-55 casing.
    pub fn load_from_list<I, S>(&self, addresses: I)
    where
        I: IntoIterator<Item = S>,
        S: AsRef<str>,
    {
        let mut bots = HashMap::new();
        for address in addresses {
            let address = address.as_ref().trim();
            if address.len() == 42 && address.starts_with("0x") {
                bots.insert(address.to_lowercase(), true);
            }
        }
        *self.known_bots.write() = bots;
    }

    pub fn known_bot_count(&self) -> usize {
        self.known_bots.read().len()
    }

    /// Reads `MEV_KNOWN_BOTS` (comma separated addresses) into the detector.
    pub fn load_from_env(&self) {
        let raw = std::env::var("MEV_KNOWN_BOTS").unwrap_or_default();
        self.load_from_list(raw.split(',').filter(|s| !s.trim().is_empty()));
    }

    pub fn detect(&self, _tx_data: &str, from: &str) -> MEVDetection {
        if self.known_bots.read().contains_key(&from.to_lowercase()) {
            MEVDetection { mev_type: MEVType::FrontRun, confidence: 95 }
        } else {
            MEVDetection::clean()
        }
    }
}

impl Default for MEVProtection {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn an_empty_feed_flags_nothing() {
        let protection = MEVProtection::new();
        assert_eq!(protection.known_bot_count(), 0);
        assert_eq!(
            protection.detect("0x", "0x1111111111111111111111111111111111111111").mev_type,
            MEVType::None
        );
    }

    #[test]
    fn a_loaded_address_is_flagged_regardless_of_casing() {
        let protection = MEVProtection::new();
        protection.load_from_list(["0xAbCd000000000000000000000000000000000000"]);
        assert_eq!(protection.known_bot_count(), 1);
        let detection = protection.detect("0x", "0xabcd000000000000000000000000000000000000");
        assert_eq!(detection.mev_type, MEVType::FrontRun);
    }

    #[test]
    fn malformed_entries_are_ignored() {
        let protection = MEVProtection::new();
        protection.load_from_list(["not-an-address", "0x1234", ""]);
        assert_eq!(protection.known_bot_count(), 0);
    }
}
