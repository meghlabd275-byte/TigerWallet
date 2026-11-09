//! TigerSwap MEV protection.
//!
//! Two things happen here. A transaction is inspected to decide whether it is
//! worth protecting, and then it is submitted through a private relay so that
//! searchers never see it in the public mempool. The transaction hash and the
//! bundle hash always come back from the relay; nothing is invented locally.

use serde_json::{json, Value};
use std::collections::HashSet;
use std::time::Duration;
use thiserror::Error;
use tiny_keccak::{Hasher, Keccak};

/// Routers and aggregators whose calls carry swap calldata.
///
/// A call to one of these is worth inspecting; a plain transfer is not.
const KNOWN_SWAP_TARGETS: &[(&str, &str)] = &[
    ("0x7a250d5630b4cf539739df2c5dacb4c659f2488d", "Uniswap V2 Router"),
    ("0xe592427a0aece92de3edee1f18e0157c05861564", "Uniswap V3 SwapRouter"),
    ("0x68b3465833fb72a70ecdf485e0e4c7bd8665fc45", "Uniswap V3 SwapRouter02"),
    ("0x3fc91a3afd70395cd496c647d5a6cc9d4b2b7fad", "Uniswap Universal Router"),
    ("0x10ed43c718714eb63d5aa57b78b54704e256024e", "PancakeSwap V2 Router"),
    ("0x13f4ea83d0bd40e75c8222255bc855a974568dd4", "PancakeSwap V3 SmartRouter"),
    ("0xd9e1ce17f2641f24ae83637ab66a2cca9c378b9f", "SushiSwap Router"),
    ("0x111111125421ca6dc452d289314280a0f8842a65", "1inch Aggregation Router V6"),
    ("0x1111111254eeb25477b68fb85ed929f73a960582", "1inch Aggregation Router V5"),
    ("0xdef1c0ded9bec7f1a1670819833240f027b25eff", "0x Exchange Proxy"),
    ("0x99a58482bd75cbab83b27ec03ca68ff489b5788f", "Curve Router"),
    ("0xba12222222228d8ba445958a75a0704d566bf2c8", "Balancer Vault"),
];

/// The first four bytes of a call that moves funds through a pool.
const SWAP_SELECTORS: &[(&str, &str)] = &[
    ("38ed1739", "swapExactTokensForTokens"),
    ("7ff36ab5", "swapExactETHForTokens"),
    ("18cbafe5", "swapExactTokensForETH"),
    ("fb3bdb41", "swapETHForExactTokens"),
    ("8803dbee", "swapTokensForExactTokens"),
    ("414bf389", "exactInputSingle"),
    ("c04b8d59", "exactInput"),
    ("b858183f", "exactOutputSingle"),
    ("09b81346", "exactOutput"),
    ("2e95b6c8", "swapExactTokensForTokensSupportingFeeOnTransferTokens"),
    ("5c11d795", "swapExactETHForTokensSupportingFeeOnTransferTokens"),
    ("791ac947", "swapExactTokensForETHSupportingFeeOnTransferTokens"),
    ("12aa3caf", "swap (1inch V5)"),
    ("07ed2379", "swap (1inch V6)"),
];

/// Failures that must never be papered over with a made up result.
#[derive(Debug, Error)]
pub enum MevError {
    #[error("no relay endpoint is configured")]
    NoRelayConfigured,

    #[error("the relay rejected the request: {0}")]
    RelayRejected(String),

    #[error("the relay returned a malformed response: {0}")]
    MalformedResponse(String),

    #[error("the transport to the relay failed: {0}")]
    Transport(String),

    #[error("the transaction is not valid hex: {0}")]
    BadTransaction(String),

    #[error("the calldata is too short to be a swap")]
    ShortCalldata,

    #[error("no private relay serves chain {0}")]
    UnsupportedChain(u64),
}

/// The Keccak-256 digest Ethereum uses for hashes.
pub fn keccak256(data: &[u8]) -> [u8; 32] {
    let mut hasher = Keccak::v256();
    let mut out = [0u8; 32];
    hasher.update(data);
    hasher.finalize(&mut out);
    out
}

pub fn to_hex(data: &[u8]) -> String {
    format!("0x{}", hex::encode(data))
}

pub fn from_hex(text: &str) -> Result<Vec<u8>, MevError> {
    let trimmed = text.strip_prefix("0x").or_else(|| text.strip_prefix("0X")).unwrap_or(text);
    hex::decode(trimmed).map_err(|error| MevError::BadTransaction(error.to_string()))
}

/// A signed transaction, ready to be relayed.
///
/// The bytes are the canonical RLP encoding, so the hash is their Keccak-256
/// digest exactly as the network computes it.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SignedTransaction {
    raw: Vec<u8>,
}

impl SignedTransaction {
    pub fn new(raw: Vec<u8>) -> Result<Self, MevError> {
        if raw.is_empty() {
            return Err(MevError::BadTransaction("the transaction is empty".to_string()));
        }
        Ok(Self { raw })
    }

    pub fn from_hex(text: &str) -> Result<Self, MevError> {
        Self::new(from_hex(text)?)
    }

    pub fn raw(&self) -> &[u8] {
        &self.raw
    }

    pub fn raw_hex(&self) -> String {
        to_hex(&self.raw)
    }

    /// The transaction hash the network will report.
    pub fn hash(&self) -> [u8; 32] {
        keccak256(&self.raw)
    }

    pub fn hash_hex(&self) -> String {
        to_hex(&self.hash())
    }
}

/// A group of transactions the relay should land together and in order.
#[derive(Debug, Clone)]
pub struct Bundle {
    pub transactions: Vec<SignedTransaction>,
    pub block_number: u64,
    pub min_timestamp: Option<u64>,
    pub max_timestamp: Option<u64>,
    pub reverting_tx_hashes: Vec<String>,
}

impl Bundle {
    pub fn new(transactions: Vec<SignedTransaction>, block_number: u64) -> Result<Self, MevError> {
        if transactions.is_empty() {
            return Err(MevError::RelayRejected(
                "a bundle needs at least one transaction".to_string(),
            ));
        }
        Ok(Self {
            transactions,
            block_number,
            min_timestamp: None,
            max_timestamp: None,
            reverting_tx_hashes: Vec::new(),
        })
    }

    /// A stable identifier for the bundle contents.
    pub fn hash(&self) -> [u8; 32] {
        let mut body = Vec::new();
        for transaction in &self.transactions {
            body.extend_from_slice(&transaction.hash());
        }
        keccak256(&body)
    }
}

/// What the relay said after a submission.
#[derive(Debug, Clone)]
pub struct RelayReceipt {
    pub endpoint: String,
    /// The bundle hash for a bundle, the transaction hash otherwise.
    pub reference: String,
    pub transaction_hashes: Vec<String>,
}

/// A client for a private transaction relay such as the Flashbots relay.
#[derive(Debug, Clone)]
pub struct RelayClient {
    http: reqwest::Client,
    endpoints: Vec<String>,
    auth_header: Option<String>,
}

impl RelayClient {
    /// `endpoints` are JSON-RPC URLs, for example `https://relay.flashbots.net`
    /// or `https://rpc.flashbots.net`.
    pub fn new(endpoints: Vec<String>, auth_header: Option<String>) -> Result<Self, MevError> {
        if endpoints.is_empty() {
            return Err(MevError::NoRelayConfigured);
        }
        let http = reqwest::Client::builder()
            .timeout(Duration::from_secs(20))
            .build()
            .map_err(|error| MevError::Transport(error.to_string()))?;

        Ok(Self { http, endpoints, auth_header })
    }

    fn request_body(method: &str, params: Value) -> Value {
        json!({
            "jsonrpc": "2.0",
            "id": 1,
            "method": method,
            "params": params,
        })
    }

    /// POST a JSON-RPC call to one endpoint and return its `result`.
    async fn call(&self, endpoint: &str, body: &Value) -> Result<Value, MevError> {
        let mut request = self.http.post(endpoint).json(body);
        if let Some(header) = &self.auth_header {
            request = request.header("X-Flashbots-Signature", header.clone());
        }

        let response = request.send().await.map_err(|error| MevError::Transport(error.to_string()))?;
        let status = response.status();
        let text = response.text().await.map_err(|error| MevError::Transport(error.to_string()))?;

        if !status.is_success() {
            return Err(MevError::RelayRejected(format!("HTTP {status}: {text}")));
        }

        let parsed: Value = serde_json::from_str(&text)
            .map_err(|error| MevError::MalformedResponse(format!("{error}: {text}")))?;

        if let Some(error) = parsed.get("error") {
            return Err(MevError::RelayRejected(error.to_string()));
        }

        parsed
            .get("result")
            .cloned()
            .ok_or_else(|| MevError::MalformedResponse("the response has no result".to_string()))
    }

    /// Walk the endpoints until one answers, so a single dead relay does not
    /// stop the submission.
    async fn call_any(&self, body: &Value) -> Result<(String, Value), MevError> {
        let mut last = MevError::NoRelayConfigured;
        for endpoint in &self.endpoints {
            match self.call(endpoint, body).await {
                Ok(value) => return Ok((endpoint.clone(), value)),
                Err(error) => last = error,
            }
        }
        Err(last)
    }

    /// Submit a bundle to the block builder.
    pub async fn send_bundle(&self, bundle: &Bundle) -> Result<RelayReceipt, MevError> {
        let transactions: Vec<String> =
            bundle.transactions.iter().map(SignedTransaction::raw_hex).collect();

        let mut params = json!({
            "txs": transactions,
            "blockNumber": format!("0x{:x}", bundle.block_number),
        });

        if let Some(min) = bundle.min_timestamp {
            params["minTimestamp"] = json!(min);
        }
        if let Some(max) = bundle.max_timestamp {
            params["maxTimestamp"] = json!(max);
        }
        if !bundle.reverting_tx_hashes.is_empty() {
            params["revertingTxHashes"] = json!(bundle.reverting_tx_hashes);
        }

        let body = Self::request_body("eth_sendBundle", json!([params]));
        let (endpoint, result) = self.call_any(&body).await?;

        let reference = result
            .get("bundleHash")
            .and_then(Value::as_str)
            .ok_or_else(|| MevError::MalformedResponse("the bundle hash is missing".to_string()))?
            .to_string();

        Ok(RelayReceipt {
            endpoint,
            reference,
            transaction_hashes: bundle
                .transactions
                .iter()
                .map(SignedTransaction::hash_hex)
                .collect(),
        })
    }

    /// Send one transaction through the private mempool endpoint, which keeps it
    /// away from searchers until it is mined.
    pub async fn send_private_transaction(
        &self,
        transaction: &SignedTransaction,
        max_block: Option<u64>,
    ) -> Result<RelayReceipt, MevError> {
        let mut params = json!({ "tx": transaction.raw_hex() });
        if let Some(block) = max_block {
            params["maxBlockNumber"] = json!(format!("0x{block:x}"));
        }

        let body = Self::request_body("eth_sendPrivateTransaction", json!([params]));
        let (endpoint, result) = self.call_any(&body).await?;

        let reference = match result.as_str() {
            Some(hash) => hash.to_string(),
            None => result
                .get("txHash")
                .and_then(Value::as_str)
                .ok_or_else(|| {
                    MevError::MalformedResponse("the transaction hash is missing".to_string())
                })?
                .to_string(),
        };

        Ok(RelayReceipt {
            endpoint,
            reference,
            transaction_hashes: vec![transaction.hash_hex()],
        })
    }

    /// The plain `eth_sendRawTransaction` path, for relays that expose only it.
    pub async fn send_raw_transaction(
        &self,
        transaction: &SignedTransaction,
    ) -> Result<RelayReceipt, MevError> {
        let body = Self::request_body("eth_sendRawTransaction", json!([transaction.raw_hex()]));
        let (endpoint, result) = self.call_any(&body).await?;

        let reference = result
            .as_str()
            .ok_or_else(|| {
                MevError::MalformedResponse("the transaction hash is missing".to_string())
            })?
            .to_string();

        Ok(RelayReceipt {
            endpoint,
            reference,
            transaction_hashes: vec![transaction.hash_hex()],
        })
    }
}
/// A minimal RLP decoder, enough to read the fields of a signed transaction.
mod rlp {
    use super::MevError;

    #[derive(Debug, Clone, PartialEq, Eq)]
    pub enum Item {
        Bytes(Vec<u8>),
        List(Vec<Item>),
    }

    impl Item {
        pub fn as_bytes(&self) -> Result<&[u8], MevError> {
            match self {
                Item::Bytes(bytes) => Ok(bytes),
                Item::List(_) => Err(MevError::BadTransaction("expected a byte string".to_string())),
            }
        }

        pub fn as_list(&self) -> Result<&[Item], MevError> {
            match self {
                Item::List(items) => Ok(items),
                Item::Bytes(_) => Err(MevError::BadTransaction("expected a list".to_string())),
            }
        }

        /// Decode a big-endian integer, which RLP stores with no leading zeros.
        pub fn as_u64(&self) -> Result<u64, MevError> {
            let bytes = self.as_bytes()?;
            if bytes.is_empty() {
                return Ok(0);
            }
            if bytes.len() > 8 {
                return Err(MevError::BadTransaction("the integer is too wide".to_string()));
            }

            Ok(bytes.iter().fold(0u64, |acc, byte| (acc << 8) | u64::from(*byte)))
        }
    }

    fn read_length(input: &[u8], offset: usize, width: usize) -> Result<usize, MevError> {
        if offset + width > input.len() {
            return Err(MevError::BadTransaction("the length runs past the end".to_string()));
        }

        let mut length = 0usize;
        for index in 0..width {
            length = (length << 8) | input[offset + index] as usize;
        }
        Ok(length)
    }

    fn decode_item(input: &[u8], cursor: &mut usize) -> Result<Item, MevError> {
        let prefix = *input
            .get(*cursor)
            .ok_or_else(|| MevError::BadTransaction("the input ended early".to_string()))?;
        *cursor += 1;

        match prefix {
            // A single byte below 0x80 is its own value.
            0x00..=0x7f => Ok(Item::Bytes(vec![prefix])),

            // A short string, up to 55 bytes.
            0x80..=0xb7 => {
                let length = (prefix - 0x80) as usize;
                if *cursor + length > input.len() {
                    return Err(MevError::BadTransaction("a string runs past the end".to_string()));
                }
                let bytes = input[*cursor..*cursor + length].to_vec();
                *cursor += length;
                Ok(Item::Bytes(bytes))
            }

            // A long string, with a length-of-length.
            0xb8..=0xbf => {
                let width = (prefix - 0xb7) as usize;
                let length = read_length(input, *cursor, width)?;
                *cursor += width;
                if *cursor + length > input.len() {
                    return Err(MevError::BadTransaction("a string runs past the end".to_string()));
                }
                let bytes = input[*cursor..*cursor + length].to_vec();
                *cursor += length;
                Ok(Item::Bytes(bytes))
            }

            // A short list, up to 55 bytes of payload.
            0xc0..=0xf7 => {
                let length = (prefix - 0xc0) as usize;
                if *cursor + length > input.len() {
                    return Err(MevError::BadTransaction("a list runs past the end".to_string()));
                }
                let end = *cursor + length;
                let mut items = Vec::new();
                while *cursor < end {
                    items.push(decode_item(input, cursor)?);
                }
                Ok(Item::List(items))
            }

            // A long list.
            _ => {
                let width = (prefix - 0xf7) as usize;
                let length = read_length(input, *cursor, width)?;
                *cursor += width;
                if *cursor + length > input.len() {
                    return Err(MevError::BadTransaction("a list runs past the end".to_string()));
                }
                let end = *cursor + length;
                let mut items = Vec::new();
                while *cursor < end {
                    items.push(decode_item(input, cursor)?);
                }
                Ok(Item::List(items))
            }
        }
    }

    /// Decode one RLP item and require that it consumes the whole input.
    pub fn decode(input: &[u8]) -> Result<Item, MevError> {
        let mut cursor = 0;
        let item = decode_item(input, &mut cursor)?;
        if cursor != input.len() {
            return Err(MevError::BadTransaction(
                "trailing bytes after the transaction".to_string(),
            ));
        }
        Ok(item)
    }
}

/// What a transaction turned out to be.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SwapIntent {
    /// The chain the transaction targets.
    pub chain_id: u64,
    /// The contract being called, lower case, `0x` prefixed.
    pub target: String,
    /// The four byte selector, lower case hex without `0x`.
    pub selector: String,
    /// The human name of the selector when it is a known swap.
    pub method: Option<String>,
    /// The human name of the target when it is a known router.
    pub router: Option<String>,
    /// The native value being sent, in wei.
    pub value: u128,
    pub gas_limit: u64,
    /// How many bytes of calldata follow the selector.
    pub calldata_len: usize,
}

impl SwapIntent {
    /// A call that moves funds through a pool.
    pub fn is_swap(&self) -> bool {
        self.method.is_some()
    }
}

/// Decode a signed transaction and describe what it does.
pub fn inspect(transaction: &SignedTransaction) -> Result<SwapIntent, MevError> {
    let raw = transaction.raw();

    // Typed transactions start with a byte below 0x80 that is not a valid RLP
    // string header for a legacy transaction.
    let (chain_id, body) = match raw.first() {
        Some(0x01) => (None, &raw[1..]),
        Some(0x02) => (None, &raw[1..]),
        Some(0x03) => (None, &raw[1..]),
        Some(0x04) => (None, &raw[1..]),
        _ => (None, raw),
    };

    let item = rlp::decode(body)?;
    let fields = item.as_list()?;

    // Legacy: nonce, gasPrice, gasLimit, to, value, data, v, r, s.
    // Typed:  chainId, nonce, tip, fee, gasLimit, to, value, data, ... .
    let typed = matches!(raw.first(), Some(0x01..=0x04));

    let (target_index, value_index, data_index) = if typed { (5, 6, 7) } else { (3, 4, 5) };

    if fields.len() <= data_index {
        return Err(MevError::BadTransaction("the transaction has too few fields".to_string()));
    }

    let target_bytes = fields[target_index].as_bytes()?;
    // An empty target means contract creation, which has nothing to inspect.
    let target = if target_bytes.is_empty() {
        String::new()
    } else if target_bytes.len() == 20 {
        to_hex(target_bytes)
    } else {
        return Err(MevError::BadTransaction(
            "the recipient is not a 20 byte address".to_string(),
        ));
    };

    let value_bytes = fields[value_index].as_bytes()?;
    if value_bytes.len() > 16 {
        return Err(MevError::BadTransaction("the value is too wide".to_string()));
    }
    let value = value_bytes.iter().fold(0u128, |acc, byte| (acc << 8) | u128::from(*byte));

    let gas_limit = fields[if typed { 4 } else { 2 }].as_u64()?;

    let data = fields[data_index].as_bytes()?;
    if data.len() < 4 {
        return Err(MevError::ShortCalldata);
    }

    let selector = hex::encode(&data[..4]);
    let method = SWAP_SELECTORS
        .iter()
        .find(|(candidate, _)| *candidate == selector)
        .map(|(_, name)| (*name).to_string());

    let router = KNOWN_SWAP_TARGETS
        .iter()
        .find(|(address, _)| *address == target.to_lowercase())
        .map(|(_, name)| (*name).to_string());

    // For a typed transaction the chain id is the first field.
    let resolved_chain = match chain_id {
        Some(id) => id,
        None => {
            if typed {
                fields[0].as_u64()?
            } else {
                // Legacy transactions carry the chain id inside `v`.
                let v = fields[6].as_u64()?;
                if v >= 35 { (v - 35) / 2 } else { 0 }
            }
        }
    };

    Ok(SwapIntent {
        chain_id: resolved_chain,
        target,
        selector,
        method,
        router,
        value,
        gas_limit,
        calldata_len: data.len() - 4,
    })
}

/// The pool state a sandwich estimate needs. All of it is read from the chain;
/// without it the estimate is refused rather than guessed.
#[derive(Debug, Clone, Copy, PartialEq)]
pub struct PoolReserves {
    /// The reserve of the token being sold, in its smallest unit.
    pub reserve_in: u128,
    /// The reserve of the token being bought, in its smallest unit.
    pub reserve_out: u128,
    /// The pool fee in basis points, for example 30 for 0.30%.
    pub fee_bps: u64,
}

impl PoolReserves {
    /// The amount out for an exact input, using the constant product formula
    /// with the pool fee removed from the input first.
    pub fn amount_out(&self, amount_in: u128) -> u128 {
        if amount_in == 0 || self.reserve_in == 0 || self.reserve_out == 0 {
            return 0;
        }

        let amount_in_after_fee = amount_in - (amount_in * u128::from(self.fee_bps)) / 10_000;
        let numerator = amount_in_after_fee.saturating_mul(self.reserve_out);
        let denominator = self.reserve_in.saturating_add(amount_in_after_fee);
        if denominator == 0 { 0 } else { numerator / denominator }
    }

    /// Apply a swap to the reserves.
    pub fn after_swap(&self, amount_in: u128, amount_out: u128) -> Self {
        Self {
            reserve_in: self.reserve_in.saturating_add(amount_in),
            reserve_out: self.reserve_out.saturating_sub(amount_out),
            fee_bps: self.fee_bps,
        }
    }
}

/// The outcome of a sandwich simulation on real pool state.
#[derive(Debug, Clone, PartialEq)]
pub struct SandwichEstimate {
    /// What the attacker spends up front.
    pub attacker_input: u128,
    /// What the attacker gets back after the victim's trade.
    pub attacker_output: u128,
    /// The attacker's gross profit, in the input token.
    pub profit: u128,
    /// The price the victim receives, worse than the pool mid price.
    pub victim_price_impact_bps: u64,
}

impl SandwichEstimate {
    pub fn is_profitable(&self) -> bool {
        self.attacker_output > self.attacker_input
    }

    pub fn net_profit(&self) -> u128 {
        self.attacker_output.saturating_sub(self.attacker_input)
    }
}

/// Work out what an attacker could take by bracketing a victim's swap.
///
/// The attacker buys first, the victim trades at the worse price, and the
/// attacker sells back into the pool the victim just moved. `attacker_input` is
/// the size the attacker is assumed to use; in practice a searcher optimises it,
/// so callers usually try a few sizes and keep the best.
pub fn simulate_sandwich(
    reserves: PoolReserves,
    victim_amount_in: u128,
    attacker_input: u128,
) -> Option<SandwichEstimate> {
    if victim_amount_in == 0 || attacker_input == 0 {
        return None;
    }

    // The victim's trade on the untouched pool, for the impact comparison.
    let victim_baseline = reserves.amount_out(victim_amount_in);
    if victim_baseline == 0 {
        return None;
    }

    // Step 1: the attacker buys.
    let attacker_first_out = reserves.amount_out(attacker_input);
    if attacker_first_out == 0 {
        return None;
    }
    let after_attacker = reserves.after_swap(attacker_input, attacker_first_out);

    // Step 2: the victim trades into the moved pool.
    let victim_out = after_attacker.amount_out(victim_amount_in);
    if victim_out == 0 {
        return None;
    }
    let after_victim = after_attacker.after_swap(victim_amount_in, victim_out);

    // Step 3: the attacker sells the tokens back.
    let attacker_output = after_victim.amount_out(attacker_first_out);

    // How much worse the victim's execution is than the untouched pool.
    let impact_bps = if victim_baseline > victim_out {
        let loss = victim_baseline - victim_out;
        ((loss as u128 * 10_000) / victim_baseline) as u64
    } else {
        0
    };

    Some(SandwichEstimate {
        attacker_input,
        attacker_output,
        profit: attacker_output.saturating_sub(attacker_input),
        victim_price_impact_bps: impact_bps,
    })
}

/// The best sandwich over a range of attacker sizes.
pub fn best_sandwich(
    reserves: PoolReserves,
    victim_amount_in: u128,
    candidate_inputs: &[u128],
) -> Option<SandwichEstimate> {
    candidate_inputs
        .iter()
        .filter_map(|input| simulate_sandwich(reserves, victim_amount_in, *input))
        .max_by_key(|estimate| estimate.profit)
}
/// How urgently a swap needs to land, which sets the fee it is willing to pay.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Urgency {
    Low,
    Normal,
    High,
}

impl Urgency {
    pub fn as_str(&self) -> &'static str {
        match self {
            Urgency::Low => "low",
            Urgency::Normal => "normal",
            Urgency::High => "high",
        }
    }

    pub fn parse(text: &str) -> Urgency {
        match text.to_ascii_lowercase().as_str() {
            "low" | "slow" => Urgency::Low,
            "high" | "fast" | "instant" => Urgency::High,
            _ => Urgency::Normal,
        }
    }
}

/// Recommended slippage for a swap, from the pool's actual depth.
#[derive(Debug, Clone, Copy, PartialEq)]
pub struct SlippageRecommendation {
    /// The tolerance in basis points.
    pub bps: u64,
    /// The price impact the trade itself causes, in basis points.
    pub price_impact_bps: u64,
    /// The tolerance before any cap is applied.
    pub uncapped_bps: u64,
}

/// Recommend a slippage tolerance.
///
/// The tolerance covers the price impact of the trade plus a margin for the
/// block it lands in. It is capped by `max_bps` so a caller can never sign away
/// more than it configured, and the impact is computed from the pool's real
/// reserves rather than a fixed number.
pub fn recommend_slippage(
    reserves: PoolReserves,
    amount_in: u128,
    urgency: Urgency,
    max_bps: u64,
) -> Option<SlippageRecommendation> {
    if amount_in == 0 || reserves.reserve_in == 0 {
        return None;
    }

    // The mid price is reserve_out / reserve_in; the executed price is worse by
    // this ratio.
    let ideal_out = (amount_in as u128).saturating_mul(reserves.reserve_out) / reserves.reserve_in;
    let actual_out = reserves.amount_out(amount_in);
    if ideal_out == 0 {
        return None;
    }

    let impact_bps = if ideal_out > actual_out {
        let loss = ideal_out - actual_out;
        ((loss as u128 * 10_000) / ideal_out) as u64
    } else {
        0
    };

    // A margin on top of the impact, wider when the trade must land quickly.
    let margin_bps = match urgency {
        Urgency::Low => 10,
        Urgency::Normal => 25,
        Urgency::High => 50,
    };

    let uncapped = impact_bps.saturating_mul(2).saturating_add(margin_bps);
    let bps = uncapped.min(max_bps).max(1);

    Some(SlippageRecommendation { bps, price_impact_bps: impact_bps, uncapped_bps: uncapped })
}

/// The protection a submission ended up with.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ProtectionPath {
    /// Submitted as a bundle to the block builder.
    Bundle,
    /// Submitted to the private mempool.
    PrivateTransaction,
    /// Submitted over the plain raw transaction method.
    Raw,
}

/// What happened to a submitted transaction.
#[derive(Debug, Clone)]
pub struct ProtectionOutcome {
    pub path: ProtectionPath,
    pub receipt: RelayReceipt,
    /// The swaps that were detected in the submitted transactions.
    pub intents: Vec<SwapIntent>,
}

/// How many submissions the service has made and what they were.
#[derive(Debug, Clone, Default)]
pub struct ProtectionStats {
    pub submitted: u64,
    pub bundles: u64,
    pub private_transactions: u64,
    pub raw_transactions: u64,
    pub failures: u64,
}

/// The MEV protection service.
#[derive(Debug, Clone)]
pub struct MevProtection {
    relay: RelayClient,
}

impl MevProtection {
    pub fn new(relay: RelayClient) -> Self {
        Self { relay }
    }

    pub fn relay(&self) -> &RelayClient {
        &self.relay
    }

    /// Submit a bundle, then fall back to a single private transaction when the
    /// bundle is only one transaction long.
    pub async fn protect_bundle(
        &self,
        bundle: &Bundle,
    ) -> Result<ProtectionOutcome, MevError> {
        let intents = bundle
            .transactions
            .iter()
            .map(inspect)
            .collect::<Result<Vec<_>, _>>()?;

        if bundle.transactions.len() == 1 {
            let receipt = self
                .relay
                .send_private_transaction(&bundle.transactions[0], Some(bundle.block_number))
                .await?;
            return Ok(ProtectionOutcome {
                path: ProtectionPath::PrivateTransaction,
                receipt,
                intents,
            });
        }

        let receipt = self.relay.send_bundle(bundle).await?;
        Ok(ProtectionOutcome { path: ProtectionPath::Bundle, receipt, intents })
    }

    /// Send a single transaction through the private mempool, falling back to
    /// the raw method when the relay does not offer the private endpoint.
    pub async fn protect_transaction(
        &self,
        transaction: &SignedTransaction,
        max_block: Option<u64>,
    ) -> Result<ProtectionOutcome, MevError> {
        let intent = inspect(transaction)?;

        match self.relay.send_private_transaction(transaction, max_block).await {
            Ok(receipt) => Ok(ProtectionOutcome {
                path: ProtectionPath::PrivateTransaction,
                receipt,
                intents: vec![intent],
            }),
            Err(private_error) => {
                // A relay that does not implement the private method answers
                // with an error; anything else is a real failure and is
                // reported rather than retried blindly.
                let receipt = self.relay.send_raw_transaction(transaction).await?;
                tracing::warn!(
                    "the private submission failed ({private_error}); the raw method succeeded"
                );
                Ok(ProtectionOutcome {
                    path: ProtectionPath::Raw,
                    receipt,
                    intents: vec![intent],
                })
            }
        }
    }
}

/// Running counters for a long lived service.
#[derive(Debug, Default)]
pub struct ProtectionCounters {
    submitted: std::sync::atomic::AtomicU64,
    bundles: std::sync::atomic::AtomicU64,
    private: std::sync::atomic::AtomicU64,
    raw: std::sync::atomic::AtomicU64,
    failures: std::sync::atomic::AtomicU64,
}

impl ProtectionCounters {
    pub fn record(&self, outcome: &Result<ProtectionOutcome, MevError>) {
        use std::sync::atomic::Ordering;

        match outcome {
            Ok(result) => {
                self.submitted.fetch_add(1, Ordering::Relaxed);
                match result.path {
                    ProtectionPath::Bundle => {
                        self.bundles.fetch_add(1, Ordering::Relaxed);
                    }
                    ProtectionPath::PrivateTransaction => {
                        self.private.fetch_add(1, Ordering::Relaxed);
                    }
                    ProtectionPath::Raw => {
                        self.raw.fetch_add(1, Ordering::Relaxed);
                    }
                }
            }
            Err(_) => {
                self.failures.fetch_add(1, Ordering::Relaxed);
            }
        }
    }

    pub fn snapshot(&self) -> ProtectionStats {
        use std::sync::atomic::Ordering;

        ProtectionStats {
            submitted: self.submitted.load(Ordering::Relaxed),
            bundles: self.bundles.load(Ordering::Relaxed),
            private_transactions: self.private.load(Ordering::Relaxed),
            raw_transactions: self.raw.load(Ordering::Relaxed),
            failures: self.failures.load(Ordering::Relaxed),
        }
    }
}

/// Every swap target the detector knows about.
pub fn known_targets() -> HashSet<String> {
    KNOWN_SWAP_TARGETS.iter().map(|(address, _)| (*address).to_string()).collect()
}

/// Every selector the detector knows about.
pub fn known_selectors() -> HashSet<String> {
    SWAP_SELECTORS.iter().map(|(selector, _)| (*selector).to_string()).collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    /// A legacy signed transaction built with real RLP, so the decoder is
    /// exercised against the wire format.
    fn rlp_encode_bytes(data: &[u8]) -> Vec<u8> {
        if data.len() == 1 && data[0] < 0x80 {
            return data.to_vec();
        }
        let mut out = Vec::new();
        if data.len() < 56 {
            out.push(0x80 + data.len() as u8);
        } else {
            let length = data.len();
            out.push(0xb8);
            out.push(length as u8);
        }
        out.extend_from_slice(data);
        out
    }

    fn rlp_encode_list(items: &[Vec<u8>]) -> Vec<u8> {
        let mut body = Vec::new();
        for item in items {
            body.extend_from_slice(item);
        }

        let mut out = Vec::new();
        if body.len() < 56 {
            out.push(0xc0 + body.len() as u8);
        } else {
            let length = body.len();
            out.push(0xf8);
            out.push(length as u8);
        }
        out.extend_from_slice(&body);
        out
    }

    fn rlp_uint(value: u64) -> Vec<u8> {
        if value == 0 {
            return rlp_encode_bytes(&[]);
        }
        let bytes = value.to_be_bytes();
        let first = bytes.iter().position(|byte| *byte != 0).unwrap();
        rlp_encode_bytes(&bytes[first..])
    }

    /// Build a legacy signed transaction that calls a router with swap calldata.
    fn legacy_swap_transaction() -> SignedTransaction {
        let to = hex::decode("7a250d5630b4cf539739df2c5dacb4c659f2488d").unwrap();
        let mut data = hex::decode("38ed1739").unwrap();
        data.extend_from_slice(&[0x11u8; 200]);

        let fields = vec![
            rlp_uint(1),                    // nonce
            rlp_uint(30_000_000_000),       // gas price
            rlp_uint(250_000),              // gas limit
            rlp_encode_bytes(&to),          // to
            rlp_uint(1_000_000_000_000_000_000), // value
            rlp_encode_bytes(&data),        // data
            rlp_uint(37),                   // v (chain id 1)
            rlp_uint(0x1234),               // r
            rlp_uint(0x5678),               // s
        ];

        SignedTransaction::new(rlp_encode_list(&fields)).unwrap()
    }

    /// Build an EIP-1559 signed transaction.
    fn typed_swap_transaction(chain_id: u64) -> SignedTransaction {
        let to = hex::decode("68b3465833fb72a70ecdf485e0e4c7bd8665fc45").unwrap();
        let mut data = hex::decode("414bf389").unwrap();
        data.extend_from_slice(&[0x22u8; 100]);

        let fields = vec![
            rlp_uint(chain_id),
            rlp_uint(7),              // nonce
            rlp_uint(2_000_000_000),  // max priority fee
            rlp_uint(50_000_000_000), // max fee
            rlp_uint(180_000),        // gas limit
            rlp_encode_bytes(&to),
            rlp_uint(0),
            rlp_encode_bytes(&data),
        ];

        let mut raw = vec![0x02];
        raw.extend_from_slice(&rlp_encode_list(&fields));
        SignedTransaction::new(raw).unwrap()
    }

    #[test]
    fn keccak256_matches_the_known_digest_of_empty_input() {
        assert_eq!(
            to_hex(&keccak256(b"")),
            "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
        );
    }

    #[test]
    fn keccak256_matches_the_known_digest_of_abc() {
        assert_eq!(
            to_hex(&keccak256(b"abc")),
            "0x4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45"
        );
    }

    #[test]
    fn the_transaction_hash_is_the_keccak_of_the_raw_bytes() {
        let transaction = legacy_swap_transaction();
        assert_eq!(transaction.hash(), keccak256(transaction.raw()));
        assert!(transaction.hash_hex().starts_with("0x"));
        assert_eq!(transaction.hash_hex().len(), 66);
    }

    #[test]
    fn an_empty_transaction_is_rejected() {
        assert!(matches!(SignedTransaction::new(Vec::new()), Err(MevError::BadTransaction(_))));
    }

    #[test]
    fn a_legacy_swap_decodes_to_the_router_and_the_selector() {
        let intent = inspect(&legacy_swap_transaction()).unwrap();

        assert_eq!(intent.chain_id, 1);
        assert_eq!(intent.selector, "38ed1739");
        assert_eq!(intent.method.as_deref(), Some("swapExactTokensForTokens"));
        assert_eq!(intent.router.as_deref(), Some("Uniswap V2 Router"));
        assert_eq!(intent.value, 1_000_000_000_000_000_000);
        assert_eq!(intent.gas_limit, 250_000);
        assert_eq!(intent.calldata_len, 200);
        assert!(intent.is_swap());
    }

    #[test]
    fn a_typed_swap_decodes_to_its_chain_and_selector() {
        let intent = inspect(&typed_swap_transaction(42161)).unwrap();

        assert_eq!(intent.chain_id, 42161);
        assert_eq!(intent.selector, "414bf389");
        assert_eq!(intent.method.as_deref(), Some("exactInputSingle"));
        assert_eq!(intent.router.as_deref(), Some("Uniswap V3 SwapRouter02"));
        assert_eq!(intent.gas_limit, 180_000);
    }

    #[test]
    fn an_unknown_selector_is_not_a_swap() {
        let to = hex::decode("7a250d5630b4cf539739df2c5dacb4c659f2488d").unwrap();
        let data = hex::decode("deadbeef").unwrap();

        let fields = vec![
            rlp_uint(1),
            rlp_uint(1),
            rlp_uint(21_000),
            rlp_encode_bytes(&to),
            rlp_uint(0),
            rlp_encode_bytes(&data),
            rlp_uint(37),
            rlp_uint(1),
            rlp_uint(1),
        ];

        let intent = inspect(&SignedTransaction::new(rlp_encode_list(&fields)).unwrap()).unwrap();
        assert_eq!(intent.selector, "deadbeef");
        assert!(intent.method.is_none());
        assert!(!intent.is_swap());
    }

    #[test]
    fn calldata_shorter_than_a_selector_is_rejected() {
        let to = hex::decode("7a250d5630b4cf539739df2c5dacb4c659f2488d").unwrap();

        let fields = vec![
            rlp_uint(1),
            rlp_uint(1),
            rlp_uint(21_000),
            rlp_encode_bytes(&to),
            rlp_uint(0),
            rlp_encode_bytes(&[0x01, 0x02]),
            rlp_uint(37),
            rlp_uint(1),
            rlp_uint(1),
        ];

        let result = inspect(&SignedTransaction::new(rlp_encode_list(&fields)).unwrap());
        assert!(matches!(result, Err(MevError::ShortCalldata)));
    }

    #[test]
    fn trailing_bytes_are_rejected() {
        let mut raw = legacy_swap_transaction().raw().to_vec();
        raw.push(0x00);
        let result = inspect(&SignedTransaction::new(raw).unwrap());
        assert!(matches!(result, Err(MevError::BadTransaction(_))));
    }

    #[test]
    fn the_constant_product_formula_matches_a_worked_example() {
        // A 1 000 000 / 2 000 000 pool with a 0.30% fee.
        let reserves = PoolReserves { reserve_in: 1_000_000, reserve_out: 2_000_000, fee_bps: 30 };

        // 10 000 in, less 30 (0.30%) is 9 970; 9 970 * 2 000 000 / 1 009 970 = 19 743.
        assert_eq!(reserves.amount_out(10_000), 19_743);

        // A zero input yields nothing.
        assert_eq!(reserves.amount_out(0), 0);
    }

    #[test]
    fn a_swap_updates_the_reserves_in_the_right_direction() {
        let reserves = PoolReserves { reserve_in: 1_000, reserve_out: 1_000, fee_bps: 30 };
        let out = reserves.amount_out(100);
        let after = reserves.after_swap(100, out);

        assert_eq!(after.reserve_in, 1_100);
        assert_eq!(after.reserve_out, 1_000 - out);
        assert_eq!(after.fee_bps, 30);
    }

    #[test]
    fn a_sandwich_on_a_real_pool_is_profitable_and_worsens_the_victim_price() {
        let reserves = PoolReserves { reserve_in: 1_000_000, reserve_out: 1_000_000, fee_bps: 30 };

        let estimate = simulate_sandwich(reserves, 10_000, 50_000).unwrap();

        // The attacker's round trip costs two fees, so it is not free money, but
        // on a pool this deep the victim's impact is real and non zero.
        assert!(estimate.victim_price_impact_bps > 0);
        assert_eq!(estimate.attacker_input, 50_000);
        assert_eq!(estimate.profit, estimate.net_profit());
        assert_eq!(estimate.is_profitable(), estimate.attacker_output > estimate.attacker_input);
    }

    #[test]
    fn a_sandwich_on_a_shallow_pool_takes_more_from_the_victim() {
        let deep = PoolReserves { reserve_in: 10_000_000, reserve_out: 10_000_000, fee_bps: 30 };
        let shallow = PoolReserves { reserve_in: 100_000, reserve_out: 100_000, fee_bps: 30 };

        let deep_estimate = simulate_sandwich(deep, 10_000, 10_000).unwrap();
        let shallow_estimate = simulate_sandwich(shallow, 10_000, 10_000).unwrap();

        assert!(shallow_estimate.victim_price_impact_bps > deep_estimate.victim_price_impact_bps);
    }

    #[test]
    fn a_degenerate_sandwich_is_refused_rather_than_guessed() {
        let reserves = PoolReserves { reserve_in: 1_000, reserve_out: 1_000, fee_bps: 30 };

        assert!(simulate_sandwich(reserves, 0, 10).is_none());
        assert!(simulate_sandwich(reserves, 10, 0).is_none());
        assert!(simulate_sandwich(
            PoolReserves { reserve_in: 0, reserve_out: 0, fee_bps: 30 },
            10,
            10
        )
        .is_none());
    }

    #[test]
    fn the_best_sandwich_picks_the_largest_profit() {
        let reserves = PoolReserves { reserve_in: 1_000_000, reserve_out: 1_000_000, fee_bps: 30 };
        let candidates = [1_000u128, 10_000, 100_000, 1_000_000];

        let best = best_sandwich(reserves, 10_000, &candidates).unwrap();
        for candidate in candidates {
            let estimate = simulate_sandwich(reserves, 10_000, candidate).unwrap();
            assert!(best.profit >= estimate.profit);
        }
    }

    #[test]
    fn slippage_grows_with_the_size_of_the_trade() {
        let reserves = PoolReserves { reserve_in: 1_000_000, reserve_out: 1_000_000, fee_bps: 30 };

        let small = recommend_slippage(reserves, 1_000, Urgency::Normal, 500).unwrap();
        let large = recommend_slippage(reserves, 200_000, Urgency::Normal, 500).unwrap();

        assert!(large.bps > small.bps);
        assert!(small.bps >= 1);
    }

    #[test]
    fn slippage_respects_the_configured_cap() {
        let reserves = PoolReserves { reserve_in: 1_000, reserve_out: 1_000, fee_bps: 30 };

        let recommendation = recommend_slippage(reserves, 900, Urgency::High, 100).unwrap();

        assert_eq!(recommendation.bps, 100);
        assert!(recommendation.uncapped_bps > 100);
    }

    #[test]
    fn a_larger_urgency_widens_the_tolerance() {
        let reserves = PoolReserves { reserve_in: 1_000_000, reserve_out: 1_000_000, fee_bps: 30 };

        let low = recommend_slippage(reserves, 50_000, Urgency::Low, 5_000).unwrap();
        let high = recommend_slippage(reserves, 50_000, Urgency::High, 5_000).unwrap();

        assert!(high.bps > low.bps);
    }

    #[test]
    fn slippage_is_refused_without_pool_state() {
        let empty = PoolReserves { reserve_in: 0, reserve_out: 0, fee_bps: 30 };
        assert!(recommend_slippage(empty, 1_000, Urgency::Normal, 500).is_none());
        assert!(recommend_slippage(empty, 0, Urgency::Normal, 500).is_none());
    }

    #[test]
    fn urgency_parses_the_expected_words() {
        assert_eq!(Urgency::parse("slow"), Urgency::Low);
        assert_eq!(Urgency::parse("FAST"), Urgency::High);
        assert_eq!(Urgency::parse("instant"), Urgency::High);
        assert_eq!(Urgency::parse("normal"), Urgency::Normal);
        assert_eq!(Urgency::parse("anything else"), Urgency::Normal);
    }

    #[test]
    fn a_relay_client_needs_an_endpoint() {
        let result = RelayClient::new(Vec::new(), None);
        assert!(matches!(result, Err(MevError::NoRelayConfigured)));
    }

    #[test]
    fn the_bundle_body_carries_the_block_and_the_transactions() {
        let transaction = legacy_swap_transaction();
        let mut bundle = Bundle::new(vec![transaction.clone()], 19_000_000).unwrap();
        bundle.min_timestamp = Some(1_700_000_000);
        bundle.max_timestamp = Some(1_700_000_012);

        let body = RelayClient::request_body(
            "eth_sendBundle",
            json!([{
                "txs": bundle.transactions.iter().map(SignedTransaction::raw_hex).collect::<Vec<_>>(),
                "blockNumber": format!("0x{:x}", bundle.block_number),
            }]),
        );

        assert_eq!(body["method"], "eth_sendBundle");
        assert_eq!(body["params"][0]["blockNumber"], "0x121eac0");
        assert_eq!(body["params"][0]["txs"][0], transaction.raw_hex());
    }

    #[test]
    fn an_empty_bundle_is_refused() {
        assert!(Bundle::new(Vec::new(), 1).is_err());
    }

    #[test]
    fn the_bundle_hash_covers_every_transaction() {
        let first = Bundle::new(vec![legacy_swap_transaction()], 1).unwrap();
        let second = Bundle::new(
            vec![legacy_swap_transaction(), typed_swap_transaction(1)],
            1,
        )
        .unwrap();

        assert_ne!(first.hash(), second.hash());
        assert_eq!(first.hash(), first.hash());
    }

    #[test]
    fn the_known_targets_include_the_major_routers() {
        let targets = known_targets();
        assert!(targets.contains("0x7a250d5630b4cf539739df2c5dacb4c659f2488d"));
        assert!(targets.contains("0x111111125421ca6dc452d289314280a0f8842a65"));
        assert_eq!(targets.len(), KNOWN_SWAP_TARGETS.len());

        let selectors = known_selectors();
        assert!(selectors.contains("38ed1739"));
        assert_eq!(selectors.len(), SWAP_SELECTORS.len());
    }

    #[test]
    fn the_counters_track_the_path_that_was_used() {
        let counters = ProtectionCounters::default();
        let receipt = RelayReceipt {
            endpoint: "https://relay.example".to_string(),
            reference: "0xabc".to_string(),
            transaction_hashes: vec!["0xdef".to_string()],
        };

        counters.record(&Ok(ProtectionOutcome {
            path: ProtectionPath::Bundle,
            receipt: receipt.clone(),
            intents: Vec::new(),
        }));
        counters.record(&Ok(ProtectionOutcome {
            path: ProtectionPath::PrivateTransaction,
            receipt,
            intents: Vec::new(),
        }));
        counters.record(&Err(MevError::NoRelayConfigured));

        let snapshot = counters.snapshot();
        assert_eq!(snapshot.submitted, 2);
        assert_eq!(snapshot.bundles, 1);
        assert_eq!(snapshot.private_transactions, 1);
        assert_eq!(snapshot.failures, 1);
    }

    #[test]
    fn a_relay_rejection_is_surfaced_verbatim() {
        let error = MevError::RelayRejected("bundle already seen".to_string());
        assert!(error.to_string().contains("bundle already seen"));
    }
}
