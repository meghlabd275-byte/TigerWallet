//! Bundle Builder for MEV Protection

use crate::detection::TransactionData;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BundleTransaction {
    pub to: String,
    pub data: String,
    pub value: String,
    pub max_fee_per_gas: String,
    pub max_priority_fee_per_gas: String,
    pub nonce: u64,
    pub chain_id: u64,
}

pub struct BundleBuilder {
    transactions: Vec<TransactionData>,
    enable_randomization: bool,
}

impl BundleBuilder {
    pub fn new() -> Self {
        Self {
            transactions: Vec::new(),
            enable_randomization: true,
        }
    }

    pub fn add_transaction(&mut self, tx: TransactionData) {
        self.transactions.push(tx);
    }

    pub fn randomize_order(&mut self) {
        if self.enable_randomization {
            self.shuffle_with_seed(Self::entropy_seed());
        }
    }

    /// Fisher-Yates over `0..=i`, driven by a caller-supplied seed so the
    /// ordering is reproducible in tests.
    ///
    /// The previous implementation computed `j = i % (i + 1)`, which is
    /// always `i`, so every swap exchanged an element with itself and the
    /// bundle order was never randomised -- the ordering signal MEV searchers
    /// rely on was left intact.
    pub fn shuffle_with_seed(&mut self, mut state: u64) {
        let len = self.transactions.len();
        if len < 2 {
            return;
        }
        for i in (1..len).rev() {
            state = state
                .wrapping_mul(6364136223846793005)
                .wrapping_add(1442695040888963407);
            let j = (state >> 33) as usize % (i + 1);
            self.transactions.swap(i, j);
        }
    }

    fn entropy_seed() -> u64 {
        use std::time::{SystemTime, UNIX_EPOCH};
        let nanos = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_nanos() as u64)
            .unwrap_or(0);
        let stack = &nanos as *const u64 as u64;
        nanos ^ stack.rotate_left(17)
    }

    /// A commitment to the bundle contents, reported to the relay.
    pub fn hash_bundle(txs: &[BundleTransaction]) -> String {
        use sha2::{Digest, Sha256};
        let mut hasher = Sha256::new();
        for tx in txs {
            for field in [
                tx.to.as_str(),
                tx.data.as_str(),
                tx.value.as_str(),
                tx.max_fee_per_gas.as_str(),
                tx.max_priority_fee_per_gas.as_str(),
            ] {
                hasher.update(field.as_bytes());
                hasher.update([0x1f]);
            }
            hasher.update(tx.nonce.to_be_bytes());
            hasher.update([0x1f]);
            hasher.update(tx.chain_id.to_be_bytes());
            hasher.update([0x1e]);
        }
        format!("0x{}", hex_encode(&hasher.finalize()))
    }

    pub fn build(&mut self) -> Vec<BundleTransaction> {
        self.randomize_order();
        
        self.transactions.iter().map(|tx| BundleTransaction {
            to: tx.to.clone(),
            data: tx.data.clone(),
            value: tx.value.clone(),
            max_fee_per_gas: tx.gas_price.clone(),
            max_priority_fee_per_gas: tx.gas_price.clone(),
            nonce: tx.nonce.unwrap_or(0),
            chain_id: tx.chain_id.unwrap_or(1),
        }).collect()
    }

    pub fn clear(&mut self) {
        self.transactions.clear();
    }

    pub fn len(&self) -> usize {
        self.transactions.len()
    }
}

impl Default for BundleBuilder {
    fn default() -> Self {
        Self::new()
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BundleResult {
    pub success: bool,
    pub bundle_hash: Option<String>,
    pub transaction_hash: Option<String>,
    pub error: Option<String>,
    pub gas_used: Option<String>,
    pub effective_gas_price: Option<String>,
    pub block_number: Option<u64>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_bundle_builder() {
        let mut builder = BundleBuilder::new();
        builder.add_transaction(TransactionData {
            from: "0xA".to_string(),
            to: "0xB".to_string(),
            data: "0x".to_string(),
            value: "0".to_string(),
            gas_price: "50".to_string(),
            hash: Some("0x1".to_string()),
        
            ..Default::default()
        });
        
        assert_eq!(builder.len(), 1);
        
        let bundle = builder.build();
        assert_eq!(bundle.len(), 1);
    }

    fn tx(id: &str) -> TransactionData {
        TransactionData {
            from: "0xA".to_string(),
            to: format!("0x{id}"),
            data: "0x".to_string(),
            value: "0".to_string(),
            gas_price: "50".to_string(),
            hash: Some(format!("0x{id}")),
            nonce: Some(1),
            chain_id: Some(1),
        }
    }

    #[test]
    fn shuffle_actually_reorders() {
        // The old shuffle computed j = i % (i + 1), which is always i, so it
        // never moved anything.
        let mut builder = BundleBuilder::new();
        for i in 0..16 {
            builder.add_transaction(tx(&i.to_string()));
        }
        let before: Vec<String> = builder.transactions.iter().map(|t| t.to.clone()).collect();

        // A seed that the LCG will actually spread across the range.
        builder.shuffle_with_seed(0x2545F4914F6CDD1D);

        let after: Vec<String> = builder.transactions.iter().map(|t| t.to.clone()).collect();
        assert_eq!(before.len(), after.len());
        assert_ne!(before, after, "shuffle left the order unchanged");

        let mut sorted_before = before.clone();
        let mut sorted_after = after.clone();
        sorted_before.sort();
        sorted_after.sort();
        assert_eq!(sorted_before, sorted_after, "shuffle changed the contents");
    }

    #[test]
    fn shuffle_is_a_permutation_for_many_seeds() {
        for seed in [1u64, 2, 7, 99, 12345, 0xDEADBEEF, u64::MAX] {
            let mut builder = BundleBuilder::new();
            for i in 0..8 {
                builder.add_transaction(tx(&i.to_string()));
            }
            builder.shuffle_with_seed(seed);
            let mut names: Vec<String> =
                builder.transactions.iter().map(|t| t.to.clone()).collect();
            names.sort();
            let expected: Vec<String> = (0..8).map(|i| format!("0x{i}")).collect();
            assert_eq!(names, expected, "seed {seed} lost or duplicated a tx");
        }
    }

    #[test]
    fn build_carries_the_callers_fee_and_nonce() {
        // The old build() hardcoded max_fee_per_gas "100", priority "2",
        // nonce 0 and chain_id 1 regardless of the transaction.
        let mut builder = BundleBuilder::new();
        let mut t = tx("1");
        t.gas_price = "1234".to_string();
        t.nonce = Some(42);
        t.chain_id = Some(137);
        builder.add_transaction(t);

        let built = builder.build();
        assert_eq!(built.len(), 1);
        assert_eq!(built[0].max_fee_per_gas, "1234");
        assert_eq!(built[0].nonce, 42);
        assert_eq!(built[0].chain_id, 137);
    }

    #[test]
    fn bundle_hash_is_content_addressed() {
        let mut a = BundleBuilder::new();
        a.add_transaction(tx("1"));
        let ha = BundleBuilder::hash_bundle(&a.build());

        let mut b = BundleBuilder::new();
        b.add_transaction(tx("1"));
        let hb = BundleBuilder::hash_bundle(&b.build());
        assert_eq!(ha, hb, "the same bundle must hash the same");

        let mut c = BundleBuilder::new();
        c.add_transaction(tx("2"));
        let hc = BundleBuilder::hash_bundle(&c.build());
        assert_ne!(ha, hc, "different bundles must hash differently");

        assert!(ha.starts_with("0x"));
        assert_eq!(ha.len(), 66, "a sha256 digest is 32 bytes");
    }
}
fn hex_encode(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        out.push(HEX[(byte >> 4) as usize] as char);
        out.push(HEX[(byte & 0x0f) as usize] as char);
    }
    out
}
