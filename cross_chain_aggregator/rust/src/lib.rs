//! TigerWallet Cross-Chain Aggregator
//!
//! Produces cross-chain route quotes from real on-chain state.
//!
//! Every number returned here is derived from an `eth_call` against an
//! operator-configured RPC endpoint: pool reserves and token decimals are read
//! from the contracts, not assumed. When an input needed for a quote is
//! unavailable the call fails closed with a specific error rather than
//! returning an estimated figure.
//!
//! Amounts are `U256` in the token's smallest unit throughout and are never
//! converted to floating point, so a quote for an 18-decimal token is exact.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::str::FromStr;

use thiserror::Error;

/// Minimal unsigned 256-bit integer.
///
/// Token amounts are integers. Routing them through `f64` loses precision above
/// 2^53 and silently rounds, which is unacceptable for a value that becomes a
/// signed transaction.
#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize)]
pub struct U256(pub [u64; 4]);

// Ordering must compare the most significant limb first. Deriving it on the
// [u64; 4] array would compare limb 0 first, which is the least significant,
// and would make every comparison in the long division wrong.
impl PartialEq for U256 {
    fn eq(&self, other: &Self) -> bool {
        self.0 == other.0
    }
}

impl Eq for U256 {}

impl PartialOrd for U256 {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

impl Ord for U256 {
    fn cmp(&self, other: &Self) -> std::cmp::Ordering {
        for i in (0..4).rev() {
            match self.0[i].cmp(&other.0[i]) {
                std::cmp::Ordering::Equal => continue,
                ordering => return ordering,
            }
        }
        std::cmp::Ordering::Equal
    }
}

impl U256 {
    pub const ZERO: U256 = U256([0, 0, 0, 0]);

    pub fn from_u64(value: u64) -> Self {
        U256([value, 0, 0, 0])
    }

    pub fn is_zero(&self) -> bool {
        self.0 == [0, 0, 0, 0]
    }

    /// Multiply two 256-bit values, returning `None` on overflow.
    pub fn checked_mul(self, other: Self) -> Option<Self> {
        let mut out = [0u64; 4];
        for i in 0..4 {
            let mut carry: u128 = 0;
            for j in 0..4 {
                if i + j > 3 {
                    // Anything landing beyond the top limb is an overflow.
                    if self.0[i] != 0 && other.0[j] != 0 {
                        return None;
                    }
                    continue;
                }
                let cur = out[i + j] as u128 + (self.0[i] as u128) * (other.0[j] as u128) + carry;
                out[i + j] = cur as u64;
                carry = cur >> 64;
            }
            if carry != 0 {
                return None;
            }
        }
        Some(U256(out))
    }

    fn div_floor(self, divisor: Self) -> Self {
        if divisor.is_zero() {
            return U256::ZERO;
        }

        let mut quotient = [0u64; 4];
        let mut remainder = U256::ZERO;

        for limb in (0..4).rev() {
            for bit in (0..64).rev() {
                remainder = remainder.shl1();
                if (self.0[limb] >> bit) & 1 == 1 {
                    remainder.0[0] |= 1;
                }
                if remainder >= divisor {
                    remainder = remainder.sub(divisor);
                    quotient[limb] |= 1u64 << bit;
                }
            }
        }
        U256(quotient)
    }

    /// Add two values, returning `None` on overflow.
    pub fn checked_add(self, other: Self) -> Option<Self> {
        let mut out = [0u64; 4];
        let mut carry = 0u128;
        for (slot, (a, b)) in out.iter_mut().zip(self.0.iter().zip(other.0.iter())) {
            let cur = *a as u128 + *b as u128 + carry;
            *slot = cur as u64;
            carry = cur >> 64;
        }
        if carry != 0 {
            return None;
        }
        Some(U256(out))
    }

    fn shl1(self) -> Self {
        let mut out = [0u64; 4];
        let mut carry = 0u64;
        for (slot, limb) in out.iter_mut().zip(self.0.iter()) {
            *slot = (limb << 1) | carry;
            carry = limb >> 63;
        }
        U256(out)
    }

    fn sub(self, other: Self) -> Self {
        let mut out = [0u64; 4];
        let mut borrow = 0u64;
        for (slot, (a, b)) in out.iter_mut().zip(self.0.iter().zip(other.0.iter())) {
            let (value, b1) = a.overflowing_sub(*b);
            let (value, b2) = value.overflowing_sub(borrow);
            *slot = value;
            borrow = (b1 as u64) + (b2 as u64);
        }
        U256(out)
    }

    /// `self * 10_000 + addend`, used by the AMM denominator.
    fn scaled_by_10k_plus(self, addend: U256) -> Option<Self> {
        let scaled = self.checked_mul(U256::from_u64(10_000))?;
        let mut out = [0u64; 4];
        let mut carry: u128 = 0;
        for (slot, (a, b)) in out.iter_mut().zip(scaled.0.iter().zip(addend.0.iter())) {
            let cur = *a as u128 + *b as u128 + carry;
            *slot = cur as u64;
            carry = cur >> 64;
        }
        if carry != 0 {
            return None;
        }
        Some(U256(out))
    }

    /// Parse a decimal amount into the smallest unit of `decimals`.
    ///
    /// The digits are used as written rather than being converted through a
    /// float, so "0.1" with 18 decimals is exactly 100000000000000000.
    pub fn parse_units(value: &str, decimals: u8) -> Result<Self, AggregatorError> {
        let value = value.trim();
        if value.is_empty() {
            return Err(AggregatorError::InvalidAmount);
        }

        let (whole, fraction) = match value.split_once('.') {
            Some((whole, fraction)) => (whole, fraction),
            None => (value, ""),
        };
        if !whole.chars().all(|c| c.is_ascii_digit())
            || !fraction.chars().all(|c| c.is_ascii_digit())
        {
            return Err(AggregatorError::InvalidAmount);
        }

        let decimals = decimals as usize;
        if fraction.len() > decimals {
            // More precision than the token can represent.
            return Err(AggregatorError::InvalidAmount);
        }

        let mut digits = String::with_capacity(whole.len() + decimals);
        digits.push_str(if whole.is_empty() { "0" } else { whole });
        digits.push_str(fraction);
        digits.push_str(&"0".repeat(decimals - fraction.len()));

        Self::from_dec_str(&digits)
    }

    /// Render the value as a decimal string with `decimals` places.
    pub fn format_units(self, decimals: u8) -> String {
        let digits = self.to_dec_string();
        let decimals = decimals as usize;
        if decimals == 0 {
            return digits;
        }

        let padded = if digits.len() <= decimals {
            format!("{}{}", "0".repeat(decimals - digits.len() + 1), digits)
        } else {
            digits
        };

        let split = padded.len() - decimals;
        let (whole, fraction) = padded.split_at(split);
        let fraction = fraction.trim_end_matches('0');
        if fraction.is_empty() {
            whole.to_string()
        } else {
            format!("{}.{}", whole, fraction)
        }
    }

    fn from_dec_str(value: &str) -> Result<Self, AggregatorError> {
        let mut out = [0u64; 4];
        for digit in value.chars() {
            let d = digit.to_digit(10).ok_or(AggregatorError::InvalidAmount)? as u64;
            let mut carry = d as u128;
            for limb in out.iter_mut() {
                let cur = (*limb as u128) * 10 + carry;
                *limb = cur as u64;
                carry = cur >> 64;
            }
            if carry != 0 {
                return Err(AggregatorError::AmountTooLarge);
            }
        }
        Ok(U256(out))
    }

    fn to_dec_string(self) -> String {
        if self.is_zero() {
            return "0".to_string();
        }
        let mut limbs = self.0;
        let mut digits = Vec::new();
        while limbs != [0u64; 4] {
            let mut remainder: u128 = 0;
            for limb in (0..4).rev() {
                let cur = (remainder << 64) | limbs[limb] as u128;
                limbs[limb] = (cur / 10) as u64;
                remainder = cur % 10;
            }
            digits.push(char::from_digit(remainder as u32, 10).unwrap_or('0'));
        }
        digits.iter().rev().collect()
    }

    /// Encode as a 32-byte big-endian word for ABI encoding.
    pub fn to_abi_word(self) -> [u8; 32] {
        let mut out = [0u8; 32];
        for (i, limb) in self.0.iter().enumerate() {
            out[24 - i * 8..32 - i * 8].copy_from_slice(&limb.to_be_bytes());
        }
        out
    }

    /// Decode from a 32-byte big-endian word.
    pub fn from_abi_word(word: &[u8]) -> Result<Self, AggregatorError> {
        if word.len() != 32 {
            return Err(AggregatorError::Rpc(format!(
                "expected a 32-byte word, got {} bytes",
                word.len()
            )));
        }
        let mut limbs = [0u64; 4];
        for (i, chunk) in word.chunks_exact(8).enumerate() {
            let mut bytes = [0u8; 8];
            bytes.copy_from_slice(chunk);
            limbs[3 - i] = u64::from_be_bytes(bytes);
        }
        Ok(U256(limbs))
    }
}

/// Division that yields zero for a zero divisor instead of panicking.
///
/// A zero divisor here means a pool reported no reserve, which the callers
/// detect separately and reject; panicking inside a quote would take the
/// service down over malformed chain data.
impl std::ops::Div for U256 {
    type Output = U256;

    fn div(self, divisor: U256) -> U256 {
        self.div_floor(divisor)
    }
}

impl std::fmt::Display for U256 {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.to_dec_string())
    }
}

impl FromStr for U256 {
    type Err = AggregatorError;

    fn from_str(value: &str) -> Result<Self, Self::Err> {
        let value = value.trim();
        if value.is_empty() {
            return Err(AggregatorError::InvalidAmount);
        }
        Self::from_dec_str(value)
    }
}

/// Errors returned by the aggregator.
#[derive(Debug, Error)]
pub enum AggregatorError {
    #[error("chain {0} is not configured; set AGGREGATOR_RPC_URL_{0}")]
    ChainNotConfigured(u64),
    #[error("token {token} on chain {chain_id} could not be read: {reason}")]
    TokenUnreadable {
        token: String,
        chain_id: u64,
        reason: String,
    },
    #[error("no liquidity pool is configured for {from} -> {to} on chain {chain_id}")]
    NoPoolConfigured {
        from: String,
        to: String,
        chain_id: u64,
    },
    #[error("pool {pool} returned unusable reserves")]
    InvalidReserves { pool: String },
    #[error("bridge {0} has no configured endpoint")]
    BridgeNotConfigured(String),
    #[error("bridge {0} does not support chain {1}")]
    BridgeChainUnsupported(String, u64),
    #[error("bridge {0} returned no usable fee for this route")]
    BridgeFeeUnavailable(String),
    #[error("invalid amount")]
    InvalidAmount,
    #[error("amount is too large to represent")]
    AmountTooLarge,
    #[error("the quote would return zero tokens")]
    OutputTooSmall,
    #[error("slippage tolerance {0} is outside the accepted range of 1 to 5000 basis points")]
    InvalidSlippage(u16),
    #[error("rpc error: {0}")]
    Rpc(String),
}

/// Networks the aggregator knows how to address.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum CrossChainNetwork {
    Ethereum,
    BnbChain,
    Polygon,
    Optimism,
    Arbitrum,
    Avalanche,
    Base,
}

impl CrossChainNetwork {
    pub fn chain_id(&self) -> u64 {
        match self {
            Self::Ethereum => 1,
            Self::BnbChain => 56,
            Self::Polygon => 137,
            Self::Optimism => 10,
            Self::Arbitrum => 42161,
            Self::Avalanche => 43114,
            Self::Base => 8453,
        }
    }

    pub fn from_chain_id(id: u64) -> Option<Self> {
        match id {
            1 => Some(Self::Ethereum),
            56 => Some(Self::BnbChain),
            137 => Some(Self::Polygon),
            10 => Some(Self::Optimism),
            42161 => Some(Self::Arbitrum),
            43114 => Some(Self::Avalanche),
            8453 => Some(Self::Base),
            _ => None,
        }
    }

    /// Native coin symbol for the chain.
    fn native_symbol(&self) -> &'static str {
        match self {
            Self::Ethereum | Self::Optimism | Self::Arbitrum | Self::Base => "ETH",
            Self::BnbChain => "BNB",
            Self::Polygon => "MATIC",
            Self::Avalanche => "AVAX",
        }
    }
}

/// Token metadata read from the chain.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub address: String,
    pub symbol: String,
    pub decimals: u8,
    pub chain_id: u64,
}

/// A Uniswap-V2 style pool the operator has registered.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PoolConfig {
    pub chain_id: u64,
    pub address: String,
    pub token0: String,
    pub token1: String,
    /// Fee in basis points, e.g. 30 for 0.30%.
    pub fee_bps: u16,
}

/// A bridge the operator has registered.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeConfig {
    pub name: String,
    /// Chains the bridge can move value between.
    pub supported_chains: Vec<u64>,
    pub enabled: bool,
    /// Endpoint returning a fee for a route.
    pub fee_endpoint: Option<String>,
}

/// Result of a same-chain swap quote.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapQuote {
    pub chain_id: u64,
    pub pool: String,
    pub from_token: Token,
    pub to_token: Token,
    pub from_amount: String,
    pub to_amount: String,
    pub price_impact_bps: u32,
    pub fee_bps: u16,
}

/// Result of a cross-chain quote.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CrossChainQuote {
    pub from_chain_id: u64,
    pub to_chain_id: u64,
    pub bridge: String,
    pub from_token: Token,
    pub to_token: Token,
    pub from_amount: String,
    pub bridge_fee: String,
    pub received_amount: String,
    pub min_received_amount: String,
    pub slippage_tolerance_bps: u16,
}

/// The cross-chain aggregator.
pub struct CrossChainAggregator {
    rpc_urls: HashMap<u64, String>,
    pools: Vec<PoolConfig>,
    bridges: HashMap<String, BridgeConfig>,
    http: reqwest::Client,
}

impl CrossChainAggregator {
    /// Build an aggregator from operator configuration.
    ///
    /// `rpc_urls` maps a chain id to an RPC endpoint. Nothing is read from the
    /// chain until a quote is requested, so construction cannot fail on network
    /// state.
    pub fn new(rpc_urls: HashMap<u64, String>) -> Self {
        Self {
            rpc_urls,
            pools: Vec::new(),
            bridges: HashMap::new(),
            http: reqwest::Client::builder()
                .timeout(std::time::Duration::from_secs(15))
                .build()
                .unwrap_or_default(),
        }
    }

    /// Register a liquidity pool.
    pub fn add_pool(&mut self, pool: PoolConfig) {
        self.pools.push(pool);
    }

    /// Register a bridge.
    pub fn add_bridge(&mut self, config: BridgeConfig) {
        self.bridges.insert(config.name.clone(), config);
    }

    /// Chains with a configured RPC endpoint.
    pub fn supported_chains(&self) -> Vec<u64> {
        let mut chains: Vec<u64> = self.rpc_urls.keys().copied().collect();
        chains.sort_unstable();
        chains
    }

    /// Read ERC-20 metadata from the chain.
    pub async fn token_metadata(
        &self,
        token: &str,
        chain_id: u64,
    ) -> Result<Token, AggregatorError> {
        // The zero address denotes the native coin.
        if token == "0x0000000000000000000000000000000000000000"
            || token.eq_ignore_ascii_case("native")
        {
            let network = CrossChainNetwork::from_chain_id(chain_id)
                .ok_or(AggregatorError::ChainNotConfigured(chain_id))?;
            return Ok(Token {
                address: token.to_string(),
                symbol: network.native_symbol().to_string(),
                decimals: 18,
                chain_id,
            });
        }

        let unreadable = |reason: String| AggregatorError::TokenUnreadable {
            token: token.to_string(),
            chain_id,
            reason,
        };

        let decimals_word = self
            .eth_call(chain_id, token, "0x313ce567", &[])
            .await
            .map_err(|err| unreadable(err.to_string()))?;

        if decimals_word.len() < 32 {
            return Err(unreadable(
                "decimals() returned no data; the address may not be a contract".to_string(),
            ));
        }

        let decimals_value = U256::from_abi_word(&decimals_word[0..32])
            .map_err(|err| unreadable(err.to_string()))?
            .0[0];

        let decimals = u8::try_from(decimals_value)
            .map_err(|_| unreadable(format!("decimals() returned {decimals_value}")))?;

        let symbol = self
            .erc20_symbol(chain_id, token)
            .await
            .unwrap_or_else(|| "UNKNOWN".to_string());

        Ok(Token {
            address: token.to_string(),
            symbol,
            decimals,
            chain_id,
        })
    }

    /// Best-effort read of an ERC-20 symbol.
    ///
    /// `symbol()` is not part of the original ERC-20 standard and a few tokens
    /// implement it as `bytes32` instead of `string`, so a failure here only
    /// costs the display name and must not fail the quote.
    async fn erc20_symbol(&self, chain_id: u64, token: &str) -> Option<String> {
        let word = self.eth_call(chain_id, token, "0x95d89b41", &[]).await.ok()?;

        // string: offset (32 bytes) | length (32 bytes) | data
        if word.len() >= 96 {
            let length = U256::from_abi_word(&word[32..64]).ok()?.0[0] as usize;
            if length > 0 && length <= 32 && word.len() >= 64 + length {
                if let Ok(text) = std::str::from_utf8(&word[64..64 + length]) {
                    let trimmed = text.trim_matches('\0').trim();
                    if !trimmed.is_empty() {
                        return Some(trimmed.to_string());
                    }
                }
            }
        }

        // bytes32: the value occupies the whole word.
        if word.len() >= 32 {
            let bytes: Vec<u8> = word[0..32].iter().copied().take_while(|b| *b != 0).collect();
            if let Ok(text) = std::str::from_utf8(&bytes) {
                let trimmed = text.trim();
                if !trimmed.is_empty()
                    && trimmed
                        .chars()
                        .all(|c| c.is_ascii_graphic() || c == ' ')
                {
                    return Some(trimmed.to_string());
                }
            }
        }
        None
    }

    /// Quote a swap through a registered Uniswap-V2 style pool.
    pub async fn quote_swap(
        &self,
        chain_id: u64,
        from_token: &str,
        to_token: &str,
        from_amount: &str,
    ) -> Result<SwapQuote, AggregatorError> {
        // The pool is a configuration check, so it is resolved before any RPC
        // call: reporting a missing pool is more accurate than surfacing a
        // transport error first, and it avoids two wasted round trips.
        let pool = self
            .find_pool(chain_id, from_token, to_token)
            .ok_or_else(|| AggregatorError::NoPoolConfigured {
                from: from_token.to_string(),
                to: to_token.to_string(),
                chain_id,
            })?;

        let from_meta = self.token_metadata(from_token, chain_id).await?;
        let to_meta = self.token_metadata(to_token, chain_id).await?;

        let amount_in = U256::parse_units(from_amount, from_meta.decimals)?;
        if amount_in.is_zero() {
            return Err(AggregatorError::InvalidAmount);
        }

        let (reserve_in, reserve_out) = self.pool_reserves(&pool, from_token).await?;

        let amount_out = uniswap_v2_out(amount_in, reserve_in, reserve_out, pool.fee_bps)?;
        if amount_out.is_zero() {
            return Err(AggregatorError::OutputTooSmall);
        }

        Ok(SwapQuote {
            chain_id,
            pool: pool.address.clone(),
            from_token: from_meta,
            to_token: to_meta.clone(),
            from_amount: from_amount.to_string(),
            to_amount: amount_out.format_units(to_meta.decimals),
            price_impact_bps: price_impact_bps(amount_in, reserve_in),
            fee_bps: pool.fee_bps,
        })
    }

    /// Quote a cross-chain transfer.
    ///
    /// The bridge fee is read from the bridge's configured endpoint. A bridge
    /// that cannot report its fee fails the quote rather than having a fee
    /// assumed on its behalf.
    pub async fn quote_cross_chain(
        &self,
        from_chain_id: u64,
        to_chain_id: u64,
        bridge_name: &str,
        token: &str,
        amount: &str,
        slippage_tolerance_bps: u16,
    ) -> Result<CrossChainQuote, AggregatorError> {
        if !(1..=5000).contains(&slippage_tolerance_bps) {
            return Err(AggregatorError::InvalidSlippage(slippage_tolerance_bps));
        }

        let bridge = self
            .bridges
            .get(bridge_name)
            .ok_or_else(|| AggregatorError::BridgeNotConfigured(bridge_name.to_string()))?;
        if !bridge.enabled {
            return Err(AggregatorError::BridgeNotConfigured(bridge_name.to_string()));
        }
        for chain in [from_chain_id, to_chain_id] {
            if !bridge.supported_chains.contains(&chain) {
                return Err(AggregatorError::BridgeChainUnsupported(
                    bridge_name.to_string(),
                    chain,
                ));
            }
        }

        let from_token = self.token_metadata(token, from_chain_id).await?;
        let to_token = self.token_metadata(token, to_chain_id).await?;

        let amount_in = U256::parse_units(amount, from_token.decimals)?;
        if amount_in.is_zero() {
            return Err(AggregatorError::InvalidAmount);
        }

        let fee = self
            .bridge_fee(bridge, from_chain_id, to_chain_id, amount_in)
            .await?;
        if fee >= amount_in {
            return Err(AggregatorError::OutputTooSmall);
        }
        let received = amount_in.sub(fee);

        // Minimum received applies the caller's slippage tolerance to the
        // post-fee amount.
        let min_received = received
            .checked_mul(U256::from_u64(10_000 - slippage_tolerance_bps as u64))
            .map(|product| product / U256::from_u64(10_000))
            .ok_or(AggregatorError::AmountTooLarge)?;

        Ok(CrossChainQuote {
            from_chain_id,
            to_chain_id,
            bridge: bridge.name.clone(),
            from_token: from_token.clone(),
            to_token,
            from_amount: amount.to_string(),
            bridge_fee: fee.format_units(from_token.decimals),
            received_amount: received.format_units(from_token.decimals),
            min_received_amount: min_received.format_units(from_token.decimals),
            slippage_tolerance_bps,
        })
    }

    fn find_pool(&self, chain_id: u64, from_token: &str, to_token: &str) -> Option<PoolConfig> {
        let from = from_token.to_lowercase();
        let to = to_token.to_lowercase();
        self.pools
            .iter()
            .find(|pool| {
                pool.chain_id == chain_id
                    && ((pool.token0.to_lowercase() == from && pool.token1.to_lowercase() == to)
                        || (pool.token0.to_lowercase() == to
                            && pool.token1.to_lowercase() == from))
            })
            .cloned()
    }

    /// Read both reserves from a pool, ordered to match the swap direction.
    async fn pool_reserves(
        &self,
        pool: &PoolConfig,
        from_token: &str,
    ) -> Result<(U256, U256), AggregatorError> {
        // getReserves() returns (uint112, uint112, uint32).
        let reserves = self
            .eth_call(pool.chain_id, &pool.address, "0x0902f1ac", &[])
            .await?;
        if reserves.len() < 64 {
            return Err(AggregatorError::InvalidReserves {
                pool: pool.address.clone(),
            });
        }

        let reserve0 = U256::from_abi_word(&reserves[0..32])?;
        let reserve1 = U256::from_abi_word(&reserves[32..64])?;
        if reserve0.is_zero() || reserve1.is_zero() {
            return Err(AggregatorError::InvalidReserves {
                pool: pool.address.clone(),
            });
        }

        // Reserve ordering follows token0/token1. Reading token0 from the pool
        // is authoritative; the configured value is used only if the call
        // cannot answer.
        let token0 = match self
            .eth_call(pool.chain_id, &pool.address, "0x0dfe1681", &[])
            .await
        {
            Ok(word) if word.len() >= 32 => format!("0x{}", hex::encode(&word[12..32])),
            _ => pool.token0.clone(),
        };

        if token0.eq_ignore_ascii_case(from_token) {
            Ok((reserve0, reserve1))
        } else {
            Ok((reserve1, reserve0))
        }
    }

    /// Ask a bridge for its fee, in the token's smallest unit.
    async fn bridge_fee(
        &self,
        bridge: &BridgeConfig,
        from_chain_id: u64,
        to_chain_id: u64,
        amount: U256,
    ) -> Result<U256, AggregatorError> {
        let endpoint = bridge
            .fee_endpoint
            .as_ref()
            .ok_or_else(|| AggregatorError::BridgeFeeUnavailable(bridge.name.clone()))?;

        let url = format!(
            "{}/fee?fromChainId={}&toChainId={}&amount={}",
            endpoint.trim_end_matches('/'),
            from_chain_id,
            to_chain_id,
            amount
        );

        let response = self
            .http
            .get(&url)
            .send()
            .await
            .map_err(|err| AggregatorError::Rpc(format!("bridge {}: {}", bridge.name, err)))?;

        if !response.status().is_success() {
            return Err(AggregatorError::BridgeFeeUnavailable(bridge.name.clone()));
        }

        let body: serde_json::Value = response
            .json()
            .await
            .map_err(|err| AggregatorError::Rpc(format!("bridge {}: {}", bridge.name, err)))?;

        let fee = body
            .get("fee")
            .and_then(|value| value.as_str())
            .ok_or_else(|| AggregatorError::BridgeFeeUnavailable(bridge.name.clone()))?;

        // The fee is reported in the source token's smallest unit.
        U256::from_str(fee).map_err(|_| AggregatorError::BridgeFeeUnavailable(bridge.name.clone()))
    }

    /// Perform an `eth_call` and return the decoded return data.
    async fn eth_call(
        &self,
        chain_id: u64,
        to: &str,
        selector: &str,
        args: &[[u8; 32]],
    ) -> Result<Vec<u8>, AggregatorError> {
        let rpc_url = self
            .rpc_urls
            .get(&chain_id)
            .ok_or(AggregatorError::ChainNotConfigured(chain_id))?;

        let mut data = selector.to_string();
        for arg in args {
            data.push_str(&hex::encode(arg));
        }

        let payload = serde_json::json!({
            "jsonrpc": "2.0",
            "id": 1,
            "method": "eth_call",
            "params": [
                { "to": to, "data": data },
                "latest"
            ]
        });

        let response = self
            .http
            .post(rpc_url)
            .json(&payload)
            .send()
            .await
            .map_err(|err| AggregatorError::Rpc(err.to_string()))?;

        let body: serde_json::Value = response
            .json()
            .await
            .map_err(|err| AggregatorError::Rpc(err.to_string()))?;

        if let Some(error) = body.get("error") {
            return Err(AggregatorError::Rpc(error.to_string()));
        }

        let result = body
            .get("result")
            .and_then(|value| value.as_str())
            .ok_or_else(|| AggregatorError::Rpc("response contained no result".to_string()))?;

        let trimmed = result.trim_start_matches("0x");
        if trimmed.is_empty() {
            // A call to an address with no code returns "0x".
            return Ok(Vec::new());
        }
        hex::decode(trimmed).map_err(|err| AggregatorError::Rpc(err.to_string()))
    }
}

/// Uniswap-V2 output amount for an exact input swap.
///
/// `out = (in * (10000 - fee) * reserve_out) / (reserve_in * 10000 + in * (10000 - fee))`
pub fn uniswap_v2_out(
    amount_in: U256,
    reserve_in: U256,
    reserve_out: U256,
    fee_bps: u16,
) -> Result<U256, AggregatorError> {
    if reserve_in.is_zero() || reserve_out.is_zero() {
        return Err(AggregatorError::InvalidReserves {
            pool: "reserve is zero".to_string(),
        });
    }
    if fee_bps >= 10_000 {
        return Err(AggregatorError::InvalidReserves {
            pool: "fee is not less than 100%".to_string(),
        });
    }

    let fee_factor = U256::from_u64(10_000 - fee_bps as u64);
    let amount_in_with_fee = amount_in
        .checked_mul(fee_factor)
        .ok_or(AggregatorError::AmountTooLarge)?;

    let numerator = amount_in_with_fee
        .checked_mul(reserve_out)
        .ok_or(AggregatorError::AmountTooLarge)?;
    let denominator = reserve_in
        .scaled_by_10k_plus(amount_in_with_fee)
        .ok_or(AggregatorError::AmountTooLarge)?;

    Ok(numerator.div_floor(denominator))
}

/// Price impact of a trade, in basis points.
///
/// `amount_in / (reserve_in + amount_in)` is the share of the pool the trade
/// consumes, which for a constant-product pool is the change in marginal price.
pub fn price_impact_bps(amount_in: U256, reserve_in: U256) -> u32 {
    let total = match reserve_in.checked_add(amount_in) {
        Some(total) => total,
        None => return 10_000,
    };
    if total.is_zero() {
        return 10_000;
    }

    let scaled = match amount_in.checked_mul(U256::from_u64(10_000)) {
        Some(value) => value,
        None => return 10_000,
    };
    let impact = scaled.div_floor(total).0[0];
    u32::try_from(impact.min(10_000)).unwrap_or(10_000)
}

#[cfg(test)]
mod tests {
    use super::*;

    const NATIVE: &str = "0x0000000000000000000000000000000000000000";

    fn u(value: u64) -> U256 {
        U256::from_u64(value)
    }

    #[test]
    fn parse_units_is_exact() {
        // The float path would give 99999999999999999 for this input.
        assert_eq!(
            U256::parse_units("0.1", 18).unwrap().to_string(),
            "100000000000000000"
        );
        assert_eq!(U256::parse_units("1", 6).unwrap().to_string(), "1000000");
        assert_eq!(
            U256::parse_units("123.456", 6).unwrap().to_string(),
            "123456000"
        );
        assert_eq!(U256::parse_units(".5", 2).unwrap().to_string(), "50");
    }

    #[test]
    fn parse_units_rejects_malformed_input() {
        assert!(U256::parse_units("", 18).is_err());
        assert!(U256::parse_units("abc", 18).is_err());
        assert!(U256::parse_units("1.2.3", 18).is_err());
        assert!(U256::parse_units("-1", 18).is_err());
        // More precision than the token can hold.
        assert!(U256::parse_units("0.1234567", 6).is_err());
    }

    #[test]
    fn format_units_round_trips() {
        assert_eq!(
            U256::parse_units("123.456", 6).unwrap().format_units(6),
            "123.456"
        );
        assert_eq!(u(1).format_units(18), "0.000000000000000001");
        assert_eq!(u(0).format_units(6), "0");
        assert_eq!(u(1_000_000).format_units(6), "1");
    }

    #[test]
    fn u256_arithmetic_does_not_overflow_silently() {
        let max = U256([u64::MAX; 4]);
        assert!(max.checked_mul(u(2)).is_none());
        assert_eq!(u(6).checked_mul(u(7)).unwrap(), u(42));
        assert_eq!(u(100) / u(3), u(33));
        assert_eq!(u(7) / u(0), U256::ZERO);
    }

    #[test]
    fn u256_division_is_exact_for_large_values() {
        // 2^128 / 2^64 == 2^64
        let big = U256([0, 0, 1, 0]);
        let divisor = U256([0, 1, 0, 0]);
        assert_eq!(big / divisor, U256([0, 1, 0, 0]));
    }

    #[test]
    fn abi_word_round_trips() {
        let value = U256::parse_units("1", 18).unwrap();
        assert_eq!(U256::from_abi_word(&value.to_abi_word()).unwrap(), value);
        assert!(U256::from_abi_word(&[0u8; 31]).is_err());
    }

    #[test]
    fn amm_output_matches_constant_product() {
        // A 1000/1000 pool with a 0.3% fee.
        let out = uniswap_v2_out(u(10), u(1000), u(1000), 30).unwrap();
        // (10 * 9970 * 1000) / (1000 * 10000 + 10 * 9970) = 99700000 / 10099700
        assert_eq!(out, u(9));
    }

    #[test]
    fn amm_output_rejects_bad_pools() {
        assert!(uniswap_v2_out(u(10), u(0), u(1000), 30).is_err());
        assert!(uniswap_v2_out(u(10), u(1000), u(0), 30).is_err());
        assert!(uniswap_v2_out(u(10), u(1000), u(1000), 10_000).is_err());
    }

    #[test]
    fn amm_output_handles_large_reserves() {
        // A pool holding 2^80 units per side: the intermediate product exceeds
        // 128 bits, which is where a narrower type would overflow.
        let reserve = U256([0, 1 << 16, 0, 0]);
        let out = uniswap_v2_out(u(1_000_000), reserve, reserve, 30).unwrap();
        assert!(!out.is_zero());
        assert!(out < reserve);
    }

    #[test]
    fn amm_output_moves_with_reserves() {
        // A deeper output reserve pays out more, and a deeper input reserve
        // pays out less. The comparison is monotonic rather than an exact
        // ratio because the AMM result is truncated to whole units.
        let base = uniswap_v2_out(u(10), u(1000), u(1000), 30).unwrap();
        let deeper_out = uniswap_v2_out(u(10), u(1000), u(2000), 30).unwrap();
        let deeper_in = uniswap_v2_out(u(10), u(2000), u(1000), 30).unwrap();

        assert!(deeper_out > base);
        assert!(deeper_in < base);
        // The payout can never exceed the reserve that funds it.
        assert!(deeper_out < u(2000));
    }

    #[test]
    fn price_impact_is_bounded() {
        assert_eq!(price_impact_bps(u(1), u(99)), 100);
        assert_eq!(price_impact_bps(u(0), u(100)), 0);
        // Never reports more than 100%.
        assert!(price_impact_bps(u(1000), u(1)) <= 10_000);
    }

    #[test]
    fn aggregator_reports_unconfigured_chains() {
        assert!(CrossChainAggregator::new(HashMap::new())
            .supported_chains()
            .is_empty());
    }

    #[tokio::test]
    async fn token_metadata_fails_closed_without_rpc() {
        let aggregator = CrossChainAggregator::new(HashMap::new());
        let err = aggregator
            .token_metadata("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", 1)
            .await
            .unwrap_err();
        assert!(matches!(err, AggregatorError::TokenUnreadable { .. }));
    }

    #[tokio::test]
    async fn native_token_metadata_is_known() {
        let aggregator = CrossChainAggregator::new(HashMap::new());
        let token = aggregator
            .token_metadata("0x0000000000000000000000000000000000000000", 1)
            .await
            .unwrap();
        assert_eq!(token.symbol, "ETH");
        assert_eq!(token.decimals, 18);
    }

    #[tokio::test]
    async fn swap_quote_requires_a_registered_pool() {
        let aggregator = CrossChainAggregator::new(HashMap::new());
        let err = aggregator
            .quote_swap(1, NATIVE, "0x1", "1")
            .await
            .unwrap_err();
        assert!(matches!(err, AggregatorError::NoPoolConfigured { .. }));
    }

    #[tokio::test]
    async fn cross_chain_quote_requires_a_configured_bridge() {
        let aggregator = CrossChainAggregator::new(HashMap::new());
        let err = aggregator
            .quote_cross_chain(1, 137, "Stargate", NATIVE, "1", 50)
            .await
            .unwrap_err();
        assert!(matches!(err, AggregatorError::BridgeNotConfigured(_)));
    }

    #[tokio::test]
    async fn cross_chain_quote_rejects_bad_slippage() {
        let aggregator = CrossChainAggregator::new(HashMap::new());
        assert!(matches!(
            aggregator
                .quote_cross_chain(1, 137, "Stargate", NATIVE, "1", 0)
                .await
                .unwrap_err(),
            AggregatorError::InvalidSlippage(0)
        ));
        assert!(matches!(
            aggregator
                .quote_cross_chain(1, 137, "Stargate", NATIVE, "1", 6000)
                .await
                .unwrap_err(),
            AggregatorError::InvalidSlippage(6000)
        ));
    }

    #[tokio::test]
    async fn bridge_without_a_fee_endpoint_fails_closed() {
        let mut aggregator = CrossChainAggregator::new(HashMap::new());
        aggregator.add_bridge(BridgeConfig {
            name: "Stargate".to_string(),
            supported_chains: vec![1, 137],
            enabled: true,
            fee_endpoint: None,
        });

        let err = aggregator
            .quote_cross_chain(1, 137, "Stargate", NATIVE, "1", 50)
            .await
            .unwrap_err();
        assert!(matches!(err, AggregatorError::BridgeFeeUnavailable(_)));
    }

    #[tokio::test]
    async fn bridge_rejects_unsupported_chain() {
        let mut aggregator = CrossChainAggregator::new(HashMap::new());
        aggregator.add_bridge(BridgeConfig {
            name: "Stargate".to_string(),
            supported_chains: vec![1],
            enabled: true,
            fee_endpoint: Some("http://127.0.0.1:1".to_string()),
        });

        let err = aggregator
            .quote_cross_chain(1, 137, "Stargate", NATIVE, "1", 50)
            .await
            .unwrap_err();
        assert!(matches!(
            err,
            AggregatorError::BridgeChainUnsupported(_, 137)
        ));
    }

    #[tokio::test]
    async fn disabled_bridge_is_rejected() {
        let mut aggregator = CrossChainAggregator::new(HashMap::new());
        aggregator.add_bridge(BridgeConfig {
            name: "Stargate".to_string(),
            supported_chains: vec![1, 137],
            enabled: false,
            fee_endpoint: Some("http://127.0.0.1:1".to_string()),
        });

        let err = aggregator
            .quote_cross_chain(1, 137, "Stargate", NATIVE, "1", 50)
            .await
            .unwrap_err();
        assert!(matches!(err, AggregatorError::BridgeNotConfigured(_)));
    }

    #[test]
    fn network_chain_ids_are_correct() {
        assert_eq!(CrossChainNetwork::Ethereum.chain_id(), 1);
        assert_eq!(CrossChainNetwork::BnbChain.chain_id(), 56);
        assert_eq!(CrossChainNetwork::Polygon.chain_id(), 137);
        assert_eq!(CrossChainNetwork::Arbitrum.chain_id(), 42161);
        assert_eq!(CrossChainNetwork::Optimism.chain_id(), 10);
        assert_eq!(CrossChainNetwork::Avalanche.chain_id(), 43114);
        assert_eq!(CrossChainNetwork::Base.chain_id(), 8453);

        for network in [
            CrossChainNetwork::Ethereum,
            CrossChainNetwork::BnbChain,
            CrossChainNetwork::Polygon,
            CrossChainNetwork::Arbitrum,
            CrossChainNetwork::Optimism,
            CrossChainNetwork::Avalanche,
            CrossChainNetwork::Base,
        ] {
            assert_eq!(
                CrossChainNetwork::from_chain_id(network.chain_id()),
                Some(network)
            );
        }
    }
}
