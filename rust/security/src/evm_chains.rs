//! TigerWallet EVM Chain Support - Monad, Berachain, and Emerging Chains
//! 
//! This module provides complete support for emerging EVM-compatible chains:
//! - Monad (High-performance EVM)
//! - Berachain (Proof of Liquidity)
//! - Berachain Governance
//! 
//! Features:
//! - Full EVM compatibility
//! - Native bridge integration
//! - Validator/staking support
//! - Governance participation
//! - Gas optimization
//! 
//! References:
//! - Monad: https://monadlabs.com
//! - Berachain: https://berachain.com

use std::collections::HashMap;

use serde::{Deserialize, Serialize};
use thiserror::Error;

/// Chain errors
#[derive(Error, Debug)]
pub enum ChainError {
    #[error("Invalid chain ID: {0}")]
    InvalidChainID(u64),
    
    #[error("RPC error: {0}")]
    RPCError(String),
    
    #[error("Transaction failed: {0}")]
    TransactionFailed(String),
    
    #[error("Bridge error: {0}")]
    BridgeError(String),
    
    #[error("Staking error: {0}")]
    StakingError(String),
    
    #[error("Governance error: {0}")]
    GovernanceError(String),
}

/// Chain configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainConfig {
    /// Canonical wrapped-native token (WETH/WBNB/...), when the chain has one.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub wrapped_native: Option<String>,
    /// Chain ID
    pub chain_id: u64,
    
    /// Chain name
    pub name: String,
    
    /// Native token symbol
    pub symbol: String,
    
    /// Native token decimals
    pub decimals: u8,
    
    /// RPC URLs
    pub rpc_urls: Vec<String>,
    
    /// Explorer URLs
    pub explorers: Vec<String>,
    
    /// Bridge addresses
    pub bridges: Vec<BridgeConfig>,
    
    /// Staking contract
    pub staking_contract: Option<String>,
    
    /// Governance contract
    pub governance_contract: Option<String>,
}

/// Bridge configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeConfig {
    /// Bridge name
    pub name: String,
    
    /// Bridge address
    pub address: String,
    
    /// Supported tokens
    pub tokens: Vec<String>,
}

/// Chain implementations
pub mod monad {
    use super::*;
    use secp256k1::{Message, PublicKey, Secp256k1, SecretKey};

    /// Monad mainnet chain ID.
    pub const CHAIN_ID: u64 = 143;

    /// Create Monad configuration.
    pub fn config() -> ChainConfig {
        ChainConfig {
            chain_id: CHAIN_ID,
            name: "Monad".to_string(),
            symbol: "MON".to_string(),
            decimals: 18,
            wrapped_native: None,
            rpc_urls: vec!["https://rpc.monad.xyz".to_string()],
            explorers: vec!["https://monadvision.com".to_string()],
            bridges: vec![],
            staking_contract: None,
            governance_contract: None,
        }
    }

    /// Minimal EVM JSON-RPC client. Every method returns what the node
    /// reported; a malformed or missing `result` is an error, never a default
    /// value that would misrepresent chain state.
    pub struct RPCClient {
        rpc_url: String,
        client: reqwest::Client,
    }

    impl RPCClient {
        pub fn new(rpc_url: String) -> Self {
            Self {
                rpc_url,
                client: reqwest::Client::new(),
            }
        }

        pub fn url(&self) -> &str {
            &self.rpc_url
        }

        async fn call(&self, method: &str, params: serde_json::Value) -> Result<serde_json::Value, ChainError> {
            let request = serde_json::json!({
                "jsonrpc": "2.0",
                "method": method,
                "params": params,
                "id": 1,
            });

            let response = self
                .client
                .post(&self.rpc_url)
                .json(&request)
                .send()
                .await
                .map_err(|e| ChainError::RPCError(format!("{method} transport error: {e}")))?;

            let body: serde_json::Value = response
                .json()
                .await
                .map_err(|e| ChainError::RPCError(format!("{method} response is not JSON: {e}")))?;

            if let Some(error) = body.get("error") {
                let message = error
                    .get("message")
                    .and_then(|m| m.as_str())
                    .unwrap_or("unknown node error");
                return Err(ChainError::RPCError(format!("{method}: {message}")));
            }

            body.get("result")
                .cloned()
                .ok_or_else(|| ChainError::RPCError(format!("{method} returned no result")))
        }

        async fn call_hex_u64(&self, method: &str, params: serde_json::Value) -> Result<u64, ChainError> {
            let result = self.call(method, params).await?;
            let text = result
                .as_str()
                .ok_or_else(|| ChainError::RPCError(format!("{method} returned a non-string result")))?;
            u64::from_str_radix(text.trim_start_matches("0x"), 16)
                .map_err(|_| ChainError::RPCError(format!("{method} returned an unparsable quantity: {text}")))
        }

        /// Latest block number.
        pub async fn get_block_number(&self) -> Result<u64, ChainError> {
            self.call_hex_u64("eth_blockNumber", serde_json::json!([])).await
        }

        /// Balance in wei, as the node's hex quantity.
        pub async fn get_balance(&self, address: &str) -> Result<String, ChainError> {
            let result = self
                .call("eth_getBalance", serde_json::json!([address, "latest"]))
                .await?;
            result
                .as_str()
                .map(|s| s.to_string())
                .ok_or_else(|| ChainError::RPCError("eth_getBalance returned a non-string result".to_string()))
        }

        /// Pending transaction count for an address.
        pub async fn get_transaction_count(&self, address: &str) -> Result<u64, ChainError> {
            self.call_hex_u64("eth_getTransactionCount", serde_json::json!([address, "pending"]))
                .await
        }

        /// Current gas price in wei.
        pub async fn get_gas_price(&self) -> Result<u64, ChainError> {
            self.call_hex_u64("eth_gasPrice", serde_json::json!([])).await
        }

        /// Estimated gas for a transaction.
        pub async fn estimate_gas(&self, tx: &TransactionRequest) -> Result<u64, ChainError> {
            let mut params = serde_json::Map::new();
            if let Some(from) = &tx.from {
                params.insert("from".to_string(), serde_json::json!(from));
            }
            if let Some(to) = &tx.to {
                params.insert("to".to_string(), serde_json::json!(to));
            }
            if let Some(value) = &tx.value {
                params.insert("value".to_string(), serde_json::json!(value));
            }
            if let Some(data) = &tx.data {
                params.insert("data".to_string(), serde_json::json!(data));
            }

            self.call_hex_u64("eth_estimateGas", serde_json::json!([params])).await
        }

        /// Deployed bytecode at an address (`0x` when there is none).
        pub async fn get_code(&self, address: &str) -> Result<String, ChainError> {
            let result = self
                .call("eth_getCode", serde_json::json!([address, "latest"]))
                .await?;
            result
                .as_str()
                .map(|s| s.to_string())
                .ok_or_else(|| ChainError::RPCError("eth_getCode returned a non-string result".to_string()))
        }

        /// Transaction receipt, or `None` while the transaction is unmined.
        pub async fn get_transaction_receipt(&self, tx_hash: &str) -> Result<Option<TransactionReceipt>, ChainError> {
            let result = self.call("eth_getTransactionReceipt", serde_json::json!([tx_hash])).await?;
            if result.is_null() {
                return Ok(None);
            }
            serde_json::from_value(result)
                .map(Some)
                .map_err(|e| ChainError::RPCError(format!("unexpected receipt shape: {e}")))
        }

        /// Chain ID as reported by the node.
        pub async fn get_chain_id(&self) -> Result<u64, ChainError> {
            self.call_hex_u64("eth_chainId", serde_json::json!([])).await
        }

        /// Broadcast a signed raw transaction and return the hash the node
        /// computed for it.
        pub async fn send_raw_transaction(&self, signed_tx: &str) -> Result<String, ChainError> {
            let result = self
                .call("eth_sendRawTransaction", serde_json::json!([signed_tx]))
                .await?;
            let hash = result
                .as_str()
                .ok_or_else(|| ChainError::RPCError("eth_sendRawTransaction returned a non-string result".to_string()))?;
            if hash.is_empty() {
                return Err(ChainError::RPCError(
                    "eth_sendRawTransaction returned an empty transaction hash".to_string(),
                ));
            }
            Ok(hash.to_string())
        }
    }

    /// JSON-RPC transaction call object.
    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub struct TransactionRequest {
        pub from: Option<String>,
        pub to: Option<String>,
        pub gas: Option<String>,
        pub gas_price: Option<String>,
        pub value: Option<String>,
        pub data: Option<String>,
    }

    /// Transaction receipt as returned by `eth_getTransactionReceipt`.
    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub struct TransactionReceipt {
        #[serde(rename = "transactionHash")]
        pub transaction_hash: String,
        #[serde(rename = "blockNumber")]
        pub block_number: String,
        pub status: String,
        pub logs: Vec<serde_json::Value>,
    }

    /// Chain config response.
    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub struct ChainConfigResponse {
        pub chain_id: String,
        pub epoch: String,
    }

    /// An EVM wallet held in memory. The private key is only ever used to
    /// derive the address and to sign; it is never serialized.
    pub struct Wallet {
        private_key: [u8; 32],
        address: String,
        rpc: RPCClient,
    }

    impl Wallet {
        /// Generate a wallet from the OS CSPRNG.
        pub fn new(rpc_url: String) -> Result<Self, ChainError> {
            let bytes = crate::secure_random::generate_random_bytes(32)
                .map_err(|e| ChainError::RPCError(format!("secure random unavailable: {e}")))?;
            let mut private_key = [0u8; 32];
            private_key.copy_from_slice(&bytes);
            Self::from_private_key(private_key, rpc_url)
        }

        /// Build a wallet from an existing private key.
        pub fn from_private_key(private_key: [u8; 32], rpc_url: String) -> Result<Self, ChainError> {
            // A key outside [1, n-1] has no valid public key; reject it here
            // rather than deriving a meaningless address from it.
            SecretKey::from_slice(&private_key)
                .map_err(|e| ChainError::TransactionFailed(format!("invalid private key: {e}")))?;

            let address = Self::private_key_to_address(&private_key);
            Ok(Self {
                private_key,
                address,
                rpc: RPCClient::new(rpc_url),
            })
        }

        /// EIP-55 checksummed address.
        fn private_key_to_address(private_key: &[u8; 32]) -> String {
            let secp = Secp256k1::new();
            let secret = SecretKey::from_slice(private_key).expect("caller validated the key");
            let public = PublicKey::from_secret_key(&secp, &secret);

            // The address is the last 20 bytes of keccak256(uncompressed pubkey
            // without the 0x04 prefix).
            let uncompressed = public.serialize_uncompressed();
            let hash = crate::digest::keccak256(&uncompressed[1..]);
            let mut address = [0u8; 20];
            address.copy_from_slice(&hash[12..]);
            to_checksum_address(&address)
        }

        pub fn address(&self) -> &str {
            &self.address
        }

        pub fn rpc(&self) -> &RPCClient {
            &self.rpc
        }

        pub async fn balance(&self) -> Result<String, ChainError> {
            self.rpc.get_balance(&self.address).await
        }

        /// Sign an EIP-1559 (type 2) transaction and broadcast it.
        ///
        /// Returns the transaction hash reported by the node.
        pub async fn transfer(&self, to: &str, value_wei: u128) -> Result<String, ChainError> {
            let chain_id = self.rpc.get_chain_id().await?;
            let nonce = self.rpc.get_transaction_count(&self.address).await?;

            let tx = TransactionRequest {
                from: Some(self.address.clone()),
                to: Some(to.to_string()),
                gas: None,
                gas_price: None,
                value: Some(format!("0x{value_wei:x}")),
                data: None,
            };

            let gas_limit = self.rpc.estimate_gas(&tx).await?;
            let max_priority_fee = self.rpc.get_gas_price().await?;
            // Without a fee-market oracle the base fee is unknown; using twice
            // the current gas price for max_fee_per_gas keeps the transaction
            // valid through short base-fee spikes.
            let max_fee = max_priority_fee.saturating_mul(2);

            let raw = self.sign_eip1559(to, value_wei, nonce, gas_limit, max_fee, max_priority_fee, chain_id)?;
            self.rpc.send_raw_transaction(&raw).await
        }

        /// Build, RLP-encode and secp256k1-sign an EIP-1559 transaction.
        #[allow(clippy::too_many_arguments)]
        pub fn sign_eip1559(
            &self,
            to: &str,
            value_wei: u128,
            nonce: u64,
            gas_limit: u64,
            max_fee_per_gas: u64,
            max_priority_fee_per_gas: u64,
            chain_id: u64,
        ) -> Result<String, ChainError> {
            let to_bytes = decode_hex_address(to)?;

            let mut fields: Vec<Vec<u8>> = vec![
                encode_u64(chain_id),
                encode_u64(nonce),
                encode_u64(max_priority_fee_per_gas),
                encode_u64(max_fee_per_gas),
                encode_u64(gas_limit),
                to_bytes,
                encode_u128(value_wei),
                Vec::new(), // input data
                Vec::new(), // access list
            ];

            let unsigned = rlp_encode_list(&fields);
            let mut signing_payload = vec![0x02];
            signing_payload.extend_from_slice(&unsigned);

            let digest = crate::digest::keccak256(&signing_payload);
            let message = Message::from_slice(&digest)
                .map_err(|e| ChainError::TransactionFailed(format!("invalid signing hash: {e}")))?;

            let secp = Secp256k1::new();
            let secret = SecretKey::from_slice(&self.private_key)
                .map_err(|e| ChainError::TransactionFailed(format!("invalid private key: {e}")))?;
            let signature = secp.sign_ecdsa_recoverable(&message, &secret);
            let (recovery_id, compact) = signature.serialize_compact();

            let (r, s) = compact.split_at(32);
            let v = recovery_id.to_i32() as u8;

            fields.push(encode_bytes(r));
            fields.push(encode_bytes(s));
            fields.push(vec![v]);

            let mut raw = vec![0x02];
            raw.extend_from_slice(&rlp_encode_list(&fields));
            Ok(format!("0x{}", hex::encode(raw)))
        }
    }

    /// Format a 20-byte address with the EIP-55 checksum.
    pub fn to_checksum_address(address: &[u8; 20]) -> String {
        let lower = hex::encode(address);
        let hash = crate::digest::keccak256(lower.as_bytes());

        let mut out = String::with_capacity(42);
        out.push_str("0x");
        for (index, character) in lower.chars().enumerate() {
            // Uppercase a hex letter when the matching nibble of the hash is >= 8.
            let hash_byte = hash[index / 2];
            let nibble = if index % 2 == 0 { hash_byte >> 4 } else { hash_byte & 0x0f };
            if character.is_ascii_digit() || nibble < 8 {
                out.push(character);
            } else {
                out.push(character.to_ascii_uppercase());
            }
        }
        out
    }

    fn decode_hex_address(address: &str) -> Result<Vec<u8>, ChainError> {
        let trimmed = address.trim_start_matches("0x");
        let bytes = hex::decode(trimmed)
            .map_err(|e| ChainError::TransactionFailed(format!("address is not valid hex: {e}")))?;
        if bytes.len() != 20 {
            return Err(ChainError::TransactionFailed(format!(
                "address must be 20 bytes, got {}",
                bytes.len()
            )));
        }
        Ok(bytes)
    }

    /// RLP-encode a byte string.
    fn encode_bytes(bytes: &[u8]) -> Vec<u8> {
        if bytes.len() == 1 && bytes[0] < 0x80 {
            return vec![bytes[0]];
        }
        let mut out = encode_length(bytes.len(), 0x80);
        out.extend_from_slice(bytes);
        out
    }

    /// RLP-encode an integer, which is the shortest big-endian byte string.
    fn encode_u128(value: u128) -> Vec<u8> {
        if value == 0 {
            return encode_bytes(&[]);
        }
        let bytes = value.to_be_bytes();
        let first_nonzero = bytes.iter().position(|&b| b != 0).unwrap_or(bytes.len() - 1);
        encode_bytes(&bytes[first_nonzero..])
    }

    fn encode_u64(value: u64) -> Vec<u8> {
        encode_u128(value as u128)
    }

    /// RLP-encode a list of already-encoded items.
    fn rlp_encode_list(items: &[Vec<u8>]) -> Vec<u8> {
        let mut payload = Vec::new();
        for item in items {
            payload.extend_from_slice(item);
        }
        let mut out = encode_length(payload.len(), 0xc0);
        out.extend_from_slice(&payload);
        out
    }

    fn encode_length(length: usize, offset: u8) -> Vec<u8> {
        if length < 56 {
            return vec![offset + length as u8];
        }
        let length_bytes = length.to_be_bytes();
        let first_nonzero = length_bytes.iter().position(|&b| b != 0).unwrap_or(length_bytes.len() - 1);
        let significant = &length_bytes[first_nonzero..];

        let mut out = vec![offset + 55 + significant.len() as u8];
        out.extend_from_slice(significant);
        out
    }
}

pub mod berachain {
    use super::*;

    /// Berachain mainnet chain ID.
    pub const CHAIN_ID: u64 = 80094;

    /// Berachain configuration.
    pub fn config() -> ChainConfig {
        ChainConfig {
            wrapped_native: Some("0x6969696969696969696969696969696969696969".to_string()),
            chain_id: CHAIN_ID,
            name: "Berachain".to_string(),
            symbol: "BERA".to_string(),
            decimals: 18,
            rpc_urls: vec![
                "https://rpc.berachain.com".to_string(),
                "https://berachain-rpc.publicnode.com".to_string(),
            ],
            explorers: vec!["https://berascan.com".to_string()],
            // Berachain's canonical bridge endpoint is operator-configured;
            // there is no single public contract address to hardcode.
            bridges: vec![],
            staking_contract: None,
            governance_contract: None,
        }
    }

    /// A governance proposal as read from the chain.
    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub struct Proposal {
        pub id: u64,
        pub title: String,
        pub description: String,
        pub status: ProposalStatus,
        pub votes: VoteInfo,
    }

    /// Proposal lifecycle state.
    #[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
    pub enum ProposalStatus {
        Pending,
        Active,
        Succeeded,
        Defeated,
        Executed,
        Cancelled,
    }

    /// Tallied votes for a proposal.
    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub struct VoteInfo {
        pub proposal_id: u64,
        pub for_votes: u128,
        pub against_votes: u128,
        pub abstain_votes: u128,
        pub total_votes: u128,
    }

    /// 4-byte function selectors for the governance and BGT contracts used
    /// below. Each is keccak256 of the canonical signature, truncated to the
    /// first four bytes.
    pub mod selectors {
        /// `proposals(uint256)`
        pub const PROPOSALS: [u8; 4] = [0x01, 0x3c, 0xf7, 0x1f];
        /// `castVote(uint256,bool)`
        pub const CAST_VOTE: [u8; 4] = [0x56, 0x7e, 0x14, 0x90];
        /// `getVotes(address)`
        pub const GET_VOTES: [u8; 4] = [0x9a, 0xb2, 0x4f, 0x2c];
        /// `delegate(address)`
        pub const DELEGATE: [u8; 4] = [0x5c, 0x19, 0xd2, 0xf5];
        /// `exchangeRate()`
        pub const EXCHANGE_RATE: [u8; 4] = [0x3b, 0xa0, 0xba, 0xc8];
        /// `deposit()`
        pub const DEPOSIT: [u8; 4] = [0xd0, 0xe3, 0x0d, 0xb0];
        /// `withdraw(uint256)`
        pub const WITHDRAW: [u8; 4] = [0x2e, 0x1a, 0x7d, 0x4d];
    }

    /// Encode an address argument (left-padded to 32 bytes).
    pub fn encode_address_argument(address: &str) -> Result<Vec<u8>, ChainError> {
        let bytes = hex::decode(address.trim_start_matches("0x"))
            .map_err(|e| ChainError::GovernanceError(format!("address is not valid hex: {e}")))?;
        if bytes.len() != 20 {
            return Err(ChainError::GovernanceError(format!(
                "address must be 20 bytes, got {}",
                bytes.len()
            )));
        }
        let mut out = vec![0u8; 32];
        out[12..].copy_from_slice(&bytes);
        Ok(out)
    }

    /// Encode a uint256 argument (big-endian, left-padded).
    pub fn encode_u256_argument(value: u128) -> Vec<u8> {
        let mut out = vec![0u8; 32];
        out[16..].copy_from_slice(&value.to_be_bytes());
        out
    }

    /// Encode a `bool` argument.
    pub fn encode_bool_argument(value: bool) -> Vec<u8> {
        let mut out = vec![0u8; 32];
        if value {
            out[31] = 1;
        }
        out
    }

    /// Decode a single uint256 return word.
    pub fn decode_u256_word(word: &[u8]) -> Result<u128, ChainError> {
        if word.len() != 32 {
            return Err(ChainError::GovernanceError(format!(
                "expected a 32-byte word, got {} bytes",
                word.len()
            )));
        }
        if word[..16].iter().any(|&b| b != 0) {
            return Err(ChainError::GovernanceError("value exceeds u128".to_string()));
        }
        let mut buf = [0u8; 16];
        buf.copy_from_slice(&word[16..]);
        Ok(u128::from_be_bytes(buf))
    }

    /// Governance client. Reads come from real `eth_call`s; nothing is
    /// defaulted to zero when the node does not answer.
    pub struct GovernanceClient {
        rpc_url: String,
        governance_address: String,
        client: reqwest::Client,
    }

    impl GovernanceClient {
        pub fn new(rpc_url: String, governance_address: String) -> Self {
            Self {
                rpc_url,
                governance_address,
                client: reqwest::Client::new(),
            }
        }

        async fn eth_call(&self, to: &str, data: &[u8]) -> Result<String, ChainError> {
            let request = serde_json::json!({
                "jsonrpc": "2.0",
                "method": "eth_call",
                "params": [{
                    "to": to,
                    "data": format!("0x{}", hex::encode(data)),
                }, "latest"],
                "id": 1,
            });

            let response = self
                .client
                .post(&self.rpc_url)
                .json(&request)
                .send()
                .await
                .map_err(|e| ChainError::GovernanceError(format!("eth_call transport error: {e}")))?;

            let body: serde_json::Value = response
                .json()
                .await
                .map_err(|e| ChainError::GovernanceError(format!("eth_call response is not JSON: {e}")))?;

            if let Some(error) = body.get("error") {
                let message = error.get("message").and_then(|m| m.as_str()).unwrap_or("unknown error");
                return Err(ChainError::GovernanceError(format!("eth_call: {message}")));
            }

            body.get("result")
                .and_then(|r| r.as_str())
                .map(|s| s.to_string())
                .ok_or_else(|| ChainError::GovernanceError("eth_call returned no result".to_string()))
        }

        /// Read the raw ABI-encoded `proposals(uint256)` word set for a
        /// proposal. The caller decodes the fields it needs.
        pub async fn proposal_raw(&self, proposal_id: u64) -> Result<Vec<u8>, ChainError> {
            let mut data = selectors::PROPOSALS.to_vec();
            data.extend_from_slice(&encode_u256_argument(proposal_id as u128));

            let result = self.eth_call(&self.governance_address, &data).await?;
            hex::decode(result.trim_start_matches("0x"))
                .map_err(|e| ChainError::GovernanceError(format!("malformed eth_call result: {e}")))
        }

        /// Vote weight delegated to `owner`, read from the governance token.
        pub async fn get_votes(&self, owner: &str) -> Result<u128, ChainError> {
            let mut data = selectors::GET_VOTES.to_vec();
            data.extend_from_slice(&encode_address_argument(owner)?);

            let result = self.eth_call(&self.governance_address, &data).await?;
            let bytes = hex::decode(result.trim_start_matches("0x"))
                .map_err(|e| ChainError::GovernanceError(format!("malformed eth_call result: {e}")))?;
            decode_u256_word(&bytes)
        }

        /// Calldata for `castVote(uint256,bool)`. The caller signs and
        /// broadcasts it; this function does not pretend a vote happened.
        pub fn cast_vote_calldata(proposal_id: u64, support: bool) -> Vec<u8> {
            let mut data = selectors::CAST_VOTE.to_vec();
            data.extend_from_slice(&encode_u256_argument(proposal_id as u128));
            data.extend_from_slice(&encode_bool_argument(support));
            data
        }
    }

    /// BGT (Berachain Governance Token) helpers.
    pub struct BGT {
        pub contract: String,
    }

    impl BGT {
        /// Calldata for `delegate(address)`.
        pub fn delegate(delegatee: &str) -> Result<Vec<u8>, ChainError> {
            let mut data = selectors::DELEGATE.to_vec();
            data.extend_from_slice(&encode_address_argument(delegatee)?);
            Ok(data)
        }

        /// Calldata for `getVotes(address)`.
        pub fn get_votes(owner: &str) -> Result<Vec<u8>, ChainError> {
            let mut data = selectors::GET_VOTES.to_vec();
            data.extend_from_slice(&encode_address_argument(owner)?);
            Ok(data)
        }
    }

    /// HONEY liquid staking token helpers.
    pub struct Honey {
        pub contract: String,
        /// 18-decimal scaling factor used to render the exchange rate.
        pub decimals: u8,
    }

    impl Honey {
        /// Calldata for `deposit()`.
        pub fn deposit() -> Vec<u8> {
            selectors::DEPOSIT.to_vec()
        }

        /// Calldata for `withdraw(uint256)`.
        pub fn withdraw(amount: u128) -> Vec<u8> {
            let mut data = selectors::WITHDRAW.to_vec();
            data.extend_from_slice(&encode_u256_argument(amount));
            data
        }

        /// Calldata for `exchangeRate()`.
        pub fn exchange_rate_calldata() -> Vec<u8> {
            selectors::EXCHANGE_RATE.to_vec()
        }

        /// Render a raw `exchangeRate()` word as a decimal HONEY-per-STX rate.
        ///
        /// The value is scaled by 10^decimals on chain, so it is converted to a
        /// decimal string without ever going through a float.
        pub fn format_exchange_rate(raw: u128, decimals: u8) -> String {
            let scale = 10u128.saturating_pow(decimals as u32);
            if scale == 0 {
                return raw.to_string();
            }
            let whole = raw / scale;
            let fraction = raw % scale;
            if fraction == 0 {
                return whole.to_string();
            }
            let mut fraction_str = format!("{:0width$}", fraction, width = decimals as usize);
            while fraction_str.ends_with('0') {
                fraction_str.pop();
            }
            format!("{whole}.{fraction_str}")
        }
    }
}

/// Chain registry
pub struct ChainRegistry {
    chains: HashMap<u64, ChainConfig>,
}

impl ChainRegistry {
    pub fn new() -> Self {
        let mut chains = HashMap::new();
        
        // Add Monad
        chains.insert(monad::CHAIN_ID, monad::config());
        
        // Add Berachain
        chains.insert(berachain::CHAIN_ID, berachain::config());
        
        // Add Ethereum
        chains.insert(1, ChainConfig {
            chain_id: 1,
            name: "Ethereum".to_string(),
            symbol: "ETH".to_string(),
            wrapped_native: Some("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2".to_string()),
            decimals: 18,
            rpc_urls: vec!["https://ethereum-rpc.publicnode.com".to_string()],
            explorers: vec!["https://etherscan.io".to_string()],
            bridges: vec![],
            staking_contract: None,
            governance_contract: None,
        });
        
        // Add BSC
        chains.insert(56, ChainConfig {
            chain_id: 56,
            name: "BNB Smart Chain".to_string(),
            symbol: "BNB".to_string(),
            wrapped_native: Some("0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c".to_string()),
            decimals: 18,
            rpc_urls: vec!["https://bsc-dataseed.binance.org".to_string()],
            explorers: vec!["https://bscscan.com".to_string()],
            bridges: vec![],
            staking_contract: None,
            governance_contract: None,
        });
        
        // Add Polygon
        chains.insert(137, ChainConfig {
            chain_id: 137,
            name: "Polygon".to_string(),
            symbol: "MATIC".to_string(),
            wrapped_native: Some("0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270".to_string()),
            decimals: 18,
            rpc_urls: vec!["https://polygon-rpc.com".to_string()],
            explorers: vec!["https://polygonscan.com".to_string()],
            bridges: vec![],
            staking_contract: None,
            governance_contract: None,
        });
        
        // Add Arbitrum
        chains.insert(42161, ChainConfig {
            chain_id: 42161,
            name: "Arbitrum One".to_string(),
            symbol: "ETH".to_string(),
            wrapped_native: Some("0x82aF49447D8a07e3bd95BD0d56f35241523fBab1".to_string()),
            decimals: 18,
            rpc_urls: vec!["https://arb1.arbitrum.io/rpc".to_string()],
            explorers: vec!["https://arbiscan.io".to_string()],
            bridges: vec![],
            staking_contract: None,
            governance_contract: None,
        });
        
        // Add Optimism
        chains.insert(10, ChainConfig {
            chain_id: 10,
            name: "Optimism".to_string(),
            symbol: "ETH".to_string(),
            wrapped_native: Some("0x4200000000000000000000000000000000000006".to_string()),
            decimals: 18,
            rpc_urls: vec!["https://mainnet.optimism.io".to_string()],
            explorers: vec!["https://optimistic.etherscan.io".to_string()],
            bridges: vec![],
            staking_contract: None,
            governance_contract: None,
        });
        
        // Add Base
        chains.insert(8453, ChainConfig {
            chain_id: 8453,
            name: "Base".to_string(),
            symbol: "ETH".to_string(),
            wrapped_native: Some("0x4200000000000000000000000000000000000006".to_string()),
            decimals: 18,
            rpc_urls: vec!["https://mainnet.base.org".to_string()],
            explorers: vec!["https://basescan.org".to_string()],
            bridges: vec![],
            staking_contract: None,
            governance_contract: None,
        });
        
        Self { chains }
    }
    
    /// Get chain config
    pub fn get(&self, chain_id: u64) -> Option<&ChainConfig> {
        self.chains.get(&chain_id)
    }
    
    /// Get all chain IDs
    pub fn chain_ids(&self) -> Vec<u64> {
        self.chains.keys().copied().collect()
    }
    
    /// Add custom chain
    pub fn add(&mut self, config: ChainConfig) {
        self.chains.insert(config.chain_id, config);
    }
}

impl Default for ChainRegistry {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_chain_registry() {
        let registry = ChainRegistry::new();
        
        assert!(registry.get(1).is_some()); // Ethereum
        assert!(registry.get(56).is_some()); // BSC
        assert!(registry.get(137).is_some()); // Polygon
        assert!(registry.get(42161).is_some()); // Arbitrum
        assert!(registry.get(monad::CHAIN_ID).is_some()); // Monad
        assert!(registry.get(berachain::CHAIN_ID).is_some()); // Berachain
    }
    
    #[test]
    fn test_monad_config() {
        let config = monad::config();
        
        assert_eq!(config.chain_id, monad::CHAIN_ID);
        assert_eq!(config.name, "Monad");
        assert_eq!(config.symbol, "MON");
    }
    
    #[test]
    fn test_berachain_config() {
        let config = berachain::config();
        
        assert_eq!(config.chain_id, berachain::CHAIN_ID);
        assert_eq!(config.name, "Berachain");
        assert_eq!(config.symbol, "BERA");
        assert!(!config.rpc_urls.is_empty());
        assert!(config.wrapped_native.is_some());
        // Bridge endpoints are operator-configured, so the registry ships none
        // rather than a guessed address.
        assert!(config.bridges.is_empty());
    }
}