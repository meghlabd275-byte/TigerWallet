//! MEV Protection Service

use crate::detection::{AttackDetector, SuspiciousPattern, TransactionData};
use crate::bundle::{BundleBuilder, BundleResult, BundleTransaction};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MEVConfig {
    pub enabled: bool,
    pub use_flashbots: bool,
    pub use_co_w_protocol: bool,
    pub max_base_fee_gwei: u64,
    pub priority_fee_gwei: u64,
    pub bundle_timeout: u64,
}

impl Default for MEVConfig {
    fn default() -> Self {
        Self {
            enabled: true,
            use_flashbots: true,
            use_co_w_protocol: true,
            max_base_fee_gwei: 100,
            priority_fee_gwei: 2,
            bundle_timeout: 30000,
        }
    }
}

pub struct MEVProtectionService {
    config: MEVConfig,
    detector: AttackDetector,
    bundle_builder: BundleBuilder,
    /// Relay endpoint, e.g. `https://relay.flashbots.net`. `None` means
    /// submission fails closed instead of inventing a hash.
    relay_url: Option<String>,
    relay_signing_key: Option<String>,
    target_block: u64,
}

impl MEVProtectionService {
    pub fn new(config: MEVConfig) -> Self {
        Self {
            config,
            detector: AttackDetector::new(),
            bundle_builder: BundleBuilder::new(),
            relay_url: None,
            relay_signing_key: None,
            target_block: 0,
        }
    }

    pub fn protect_transaction(&mut self, tx: TransactionData) -> ProtectionResult {
        let mut warnings = Vec::new();
        
        if self.detector.is_honeypot(&tx.to) {
            warnings.push("Warning: interacting with known honeypot address".to_string());
        }
        
        if self.config.enabled && self.config.use_flashbots {
            self.bundle_builder.add_transaction(tx);
        }
        
        ProtectionResult {
            protected: warnings.is_empty(),
            warnings: if warnings.is_empty() { None } else { Some(warnings) },
        }
    }

    pub fn protect_batch(&mut self, txs: &[TransactionData]) -> BatchProtectionResult {
        let sandwich_patterns = self.detector.detect_sandwich(txs);
        let arbitrage_patterns = self.detector.detect_arbitrage(txs);
        
        let all_patterns: Vec<SuspiciousPattern> = sandwich_patterns
            .into_iter()
            .chain(arbitrage_patterns.into_iter())
            .collect();
        
        let attack_tx_hashes: std::collections::HashSet<String> = all_patterns
            .iter()
            .flat_map(|p| p.transactions.clone())
            .collect();
        
        let protected_txs: Vec<TransactionData> = txs.iter()
            .filter(|tx| {
                let hash = tx.hash.as_ref().cloned().unwrap_or_default();
                !attack_tx_hashes.contains(&hash)
            })
            .cloned()
            .collect();
        
        BatchProtectionResult {
            protected_txs,
            detected_attacks: all_patterns,
        }
    }

    /// Submit the pending bundle to the configured relay.
    ///
    /// The previous version invented a hash from the clock and returned
    /// `success: true` without contacting anything, so a caller believed a
    /// bundle had reached a relay when no transaction existed.
    ///
    /// With no relay configured this fails closed. With one configured it
    /// performs the real `eth_sendBundle` JSON-RPC call and reports what the
    /// relay returns: `bundleHash` on acceptance, or the relay's own error
    /// message on rejection.
    pub async fn send_protected_bundle(&mut self) -> Result<BundleResult, String> {
        let txs = self.bundle_builder.build();

        if txs.is_empty() {
            return Err("No transactions in bundle".to_string());
        }

        let Some(relay) = self.relay_url.clone() else {
            return Err(
                "no MEV relay configured; call set_relay_url before submitting a bundle"
                    .to_string(),
            );
        };

        let result = self.submit_to_relay(&relay, &txs).await;

        // Only drop the pending bundle once the relay has accepted it, so a
        // transport failure does not silently discard the caller's
        // transactions.
        if matches!(&result, Ok(r) if r.success) {
            self.bundle_builder.clear();
        }
        result
    }

    /// Point the service at a relay, e.g. `https://relay.flashbots.net`.
    pub fn set_relay_url(&mut self, url: String) {
        self.relay_url = Some(url);
    }

    /// The account key used to authenticate to the relay.
    pub fn set_relay_signing_key(&mut self, key: String) {
        self.relay_signing_key = Some(key);
    }

    /// The block the bundle targets. `0` lets the relay pick the next one.
    pub fn set_target_block(&mut self, block: u64) {
        self.target_block = block;
    }

    async fn submit_to_relay(
        &self,
        relay: &str,
        txs: &[BundleTransaction],
    ) -> Result<BundleResult, String> {
        let rpc_txs: Vec<serde_json::Value> = txs
            .iter()
            .map(|tx| {
                serde_json::json!({
                    "to": tx.to,
                    "data": tx.data,
                    "value": tx.value,
                    "maxFeePerGas": tx.max_fee_per_gas,
                    "maxPriorityFeePerGas": tx.max_priority_fee_per_gas,
                    "nonce": tx.nonce,
                    "chainId": tx.chain_id,
                })
            })
            .collect();

        let body = serde_json::json!({
            "jsonrpc": "2.0",
            "id": 1,
            "method": "eth_sendBundle",
            "params": [{ "txs": rpc_txs, "blockNumber": self.target_block }],
        });

        let client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_millis(self.config.bundle_timeout))
            .build()
            .map_err(|e| format!("failed to build relay client: {e}"))?;

        let mut request = client.post(relay).json(&body);

        // A production relay authenticates the request with an EIP-191
        // signature by the calling account.
        if let Some(key) = &self.relay_signing_key {
            request = request.header("X-Flashbots-Signature", sign_relay_request(key, &body)?);
        }

        let response = request
            .send()
            .await
            .map_err(|e| format!("relay request failed: {e}"))?;

        let status = response.status();
        let payload: serde_json::Value = response
            .json()
            .await
            .map_err(|e| format!("relay returned a non-JSON response ({status}): {e}"))?;

        if let Some(error) = payload.get("error") {
            let message = error
                .get("message")
                .and_then(|m| m.as_str())
                .unwrap_or("relay rejected the bundle");
            return Ok(BundleResult {
                success: false,
                bundle_hash: Some(BundleBuilder::hash_bundle(txs)),
                transaction_hash: None,
                error: Some(message.to_string()),
                gas_used: None,
                effective_gas_price: None,
                block_number: None,
            });
        }

        let accepted_hash = payload
            .get("result")
            .and_then(|r| r.get("bundleHash"))
            .and_then(|h| h.as_str())
            .map(|h| h.to_string())
            .unwrap_or_else(|| BundleBuilder::hash_bundle(txs));

        Ok(BundleResult {
            success: true,
            bundle_hash: Some(accepted_hash),
            transaction_hash: None,
            error: None,
            gas_used: None,
            effective_gas_price: None,
            block_number: Some(self.target_block),
        })
    }

    pub fn add_honeypot(&mut self, address: &str) {
        self.detector.add_honeypot(address);
    }

    pub fn clear(&mut self) {
        self.bundle_builder.clear();
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtectionResult {
    pub protected: bool,
    pub warnings: Option<Vec<String>>,
}

#[derive(Debug, Clone)]
pub struct BatchProtectionResult {
    pub protected_txs: Vec<TransactionData>,
    pub detected_attacks: Vec<SuspiciousPattern>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_mev_service_creation() {
        let service = MEVProtectionService::new(MEVConfig::default());
        assert!(service.config.enabled);
    }

    #[test]
    fn test_protect_transaction() {
        let mut service = MEVProtectionService::new(MEVConfig::default());
        
        let tx = TransactionData {
            from: "0xA".to_string(),
            to: "0xPool".to_string(),
            data: "0x".to_string(),
            value: "0".to_string(),
            gas_price: "50".to_string(),
            hash: Some("0x123".to_string()),
        
            ..Default::default()
        };
        
        let result = service.protect_transaction(tx);
        assert!(result.protected);
    }

    #[tokio::test]
    async fn bundle_submission_fails_closed_without_a_relay() {
        // The old implementation returned success:true and a clock-derived
        // hash here, so a caller could not tell that nothing was submitted.
        let mut service = MEVProtectionService::new(MEVConfig::default());
        service.protect_transaction(TransactionData {
            from: "0xA".to_string(),
            to: "0xB".to_string(),
            data: "0x".to_string(),
            value: "1".to_string(),
            gas_price: "50".to_string(),
            hash: Some("0xabc".to_string()),
            nonce: Some(1),
            chain_id: Some(1),
        });

        let err = service.send_protected_bundle().await.unwrap_err();
        assert!(err.contains("no MEV relay configured"), "unexpected: {err}");
    }

    #[tokio::test]
    async fn bundle_submission_fails_closed_on_a_dead_relay() {
        let mut service = MEVProtectionService::new(MEVConfig::default());
        // A port nothing is listening on.
        service.set_relay_url("http://127.0.0.1:1".to_string());
        service.protect_transaction(TransactionData {
            from: "0xA".to_string(),
            to: "0xB".to_string(),
            data: "0x".to_string(),
            value: "1".to_string(),
            gas_price: "50".to_string(),
            hash: Some("0xabc".to_string()),
            nonce: Some(1),
            chain_id: Some(1),
        });

        let result = service.send_protected_bundle().await;
        assert!(result.is_err(), "a dead relay must not report success");
    }

    #[test]
    fn empty_bundle_is_rejected() {
        let mut service = MEVProtectionService::new(MEVConfig::default());
        service.set_relay_url("http://127.0.0.1:1".to_string());
        let rt = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()
            .unwrap();
        let err = rt.block_on(service.send_protected_bundle()).unwrap_err();
        assert!(err.contains("No transactions"));
    }
}
/// Sign a relay request body the way Flashbots expects: an EIP-191 personal
/// message over `keccak256(body)` by the account derived from `private_key`.
/// The header is `"<address>:<signature>"`.
fn sign_relay_request(private_key: &str, body: &serde_json::Value) -> Result<String, String> {
    use k256::ecdsa::SigningKey;
    use sha3::{Digest, Keccak256};

    let encoded =
        serde_json::to_vec(body).map_err(|e| format!("failed to encode relay body: {e}"))?;
    let body_hash = Keccak256::digest(&encoded);

    // EIP-191: keccak256("\x19Ethereum Signed Message:\n32" || body_hash)
    let mut prefixed = Vec::with_capacity(28 + 32);
    prefixed.extend_from_slice(b"\x19Ethereum Signed Message:\n32");
    prefixed.extend_from_slice(&body_hash);
    let message_hash = Keccak256::digest(&prefixed);

    let key_bytes = decode_hex(private_key)
        .ok_or_else(|| "relay signing key is not valid hex".to_string())?;
    let signing_key = SigningKey::from_slice(&key_bytes)
        .map_err(|e| format!("relay signing key is not a valid secp256k1 key: {e}"))?;

    let (signature, recovery_id) = signing_key
        .sign_prehash_recoverable(&message_hash)
        .map_err(|e| format!("failed to sign relay request: {e}"))?;

    let mut sig_bytes = signature.to_bytes().to_vec();
    // v is the recovery id plus 27.
    sig_bytes.push(recovery_id.to_byte() + 27);

    let public_key = signing_key.verifying_key();
    let encoded_point = public_key.to_encoded_point(false);
    let public_key_hash = Keccak256::digest(&encoded_point.as_bytes()[1..]);
    let address = format!("0x{}", hex_encode(&public_key_hash[12..]));

    Ok(format!("{address}:0x{}", hex_encode(&sig_bytes)))
}

/// Decode a hex string, tolerating a `0x` prefix.
fn decode_hex(input: &str) -> Option<Vec<u8>> {
    let trimmed = input.strip_prefix("0x").unwrap_or(input);
    if trimmed.len() % 2 != 0 {
        return None;
    }
    (0..trimmed.len())
        .step_by(2)
        .map(|i| u8::from_str_radix(&trimmed[i..i + 2], 16).ok())
        .collect()
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
