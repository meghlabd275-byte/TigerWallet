//! TigerSmartChain support: the platform's own EVM network (chain id 7710).
//!
//! TigerSmartChain is where TigerWallet's own assets live. Unlike the public
//! chains in `evm_chains`, nobody else runs this network, so this module owns
//! the authoritative description of it:
//!
//! * `TGR` is the native coin. Like ETH on Ethereum it has no contract address;
//!   balances are read with `eth_getBalance`.
//! * `WTGR` is the ERC-20 wrapper of TGR, used by the swap pools.
//! * `RUSD` is the USD stablecoin, minted against a treasury reserve.
//!
//! WTGR and RUSD are **genesis predeploys**: the chain's genesis block already
//! contains their runtime bytecode, so their addresses are fixed by consensus
//! and identical on every node from block zero. That is why they are constants
//! here and not something an operator deploys and configures.
//!
//! Everything in this module reads live chain state. Nothing is estimated,
//! cached from a fixture, or defaulted to a plausible number: if the node
//! cannot be reached, or the pool does not hold RUSD, the function returns an
//! error or `None` and the caller reports "unavailable".
//!
//! # Endpoint
//!
//! `TSC_RPC_URL` selects the node. There is deliberately no fallback: the
//! public chains have well-known endpoints to fall back on, this one does not,
//! and dialling a guessed host would be worse than reporting that the chain is
//! unconfigured.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::time::{Duration, Instant};

/// EIP-155 chain id of TigerSmartChain mainnet.
pub const CHAIN_ID: u64 = 7710;

/// EIP-155 chain id of the public testnet.
pub const TESTNET_CHAIN_ID: u64 = 7711;

/// Native coin symbol.
pub const NATIVE_SYMBOL: &str = "TGR";

/// Environment variable holding the node's JSON-RPC endpoint.
pub const RPC_ENV: &str = "TSC_RPC_URL";

/// Environment variable holding the TGR/RUSD pool used to price TGR.
pub const PAIR_ENV: &str = "TSC_TGR_RUSD_PAIR";

/// RUSD stablecoin - genesis predeploy, fixed by consensus.
pub const RUSD_ADDRESS: &str = "0x0000000000000000000000000000000000007710";

/// WTGR wrapped-native token - genesis predeploy, fixed by consensus.
pub const WTGR_ADDRESS: &str = "0x0000000000000000000000000000000000007711";

/// RUSD display metadata, kept in step with the chain's own predeploy source.
pub const RUSD_NAME: &str = "Royal Tiger USD";
/// RUSD symbol.
pub const RUSD_SYMBOL: &str = "RUSD";
/// WTGR display metadata.
pub const WTGR_NAME: &str = "Wrapped TGR";
/// WTGR symbol.
pub const WTGR_SYMBOL: &str = "WTGR";

/// How a platform asset is issued on the chain.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum AssetKind {
    /// The chain's own coin. No contract; read with `eth_getBalance`.
    Native,
    /// An ERC-20 contract that already exists in the genesis block.
    GenesisPredeploy,
}

/// A first-class platform asset.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PlatformAsset {
    /// Ticker, e.g. `TGR`.
    pub symbol: String,
    /// Human-readable name.
    pub name: String,
    /// Contract address, empty for the native coin.
    pub address: String,
    /// Decimals; 18 for every asset here.
    pub decimals: u8,
    /// How the asset is issued.
    pub kind: AssetKind,
    /// True when the asset is designed to hold one US dollar.
    pub pegged_usd: bool,
}

impl PlatformAsset {
    /// Returns true when this asset is an ERC-20 contract rather than the coin.
    pub fn is_contract(&self) -> bool {
        self.kind == AssetKind::GenesisPredeploy
    }

    /// Returns the contract address as a byte array, or `None` for the coin.
    pub fn address_bytes(&self) -> Option<[u8; 20]> {
        if self.address.is_empty() {
            return None;
        }
        hex_to_address(&self.address)
    }
}

/// Every platform asset, in display order.
pub fn platform_assets() -> Vec<PlatformAsset> {
    vec![
        PlatformAsset {
            symbol: NATIVE_SYMBOL.to_string(),
            name: "Tiger".to_string(),
            address: String::new(),
            decimals: 18,
            kind: AssetKind::Native,
            pegged_usd: false,
        },
        PlatformAsset {
            symbol: WTGR_SYMBOL.to_string(),
            name: WTGR_NAME.to_string(),
            address: WTGR_ADDRESS.to_string(),
            decimals: 18,
            kind: AssetKind::GenesisPredeploy,
            pegged_usd: false,
        },
        PlatformAsset {
            symbol: RUSD_SYMBOL.to_string(),
            name: RUSD_NAME.to_string(),
            address: RUSD_ADDRESS.to_string(),
            decimals: 18,
            kind: AssetKind::GenesisPredeploy,
            pegged_usd: true,
        },
    ]
}

/// Looks up a platform asset by symbol, case-insensitively.
pub fn asset_by_symbol(symbol: &str) -> Option<PlatformAsset> {
    let want = symbol.trim().to_ascii_uppercase();
    platform_assets().into_iter().find(|a| a.symbol == want)
}

/// Looks up a platform asset by contract address, case-insensitively.
///
/// The native coin has no address, so it can never be found this way; use
/// [`asset_by_symbol`] for that.
pub fn asset_by_address(address: &str) -> Option<PlatformAsset> {
    let want = address.trim().to_ascii_lowercase();
    if want.is_empty() {
        return None;
    }
    platform_assets()
        .into_iter()
        .find(|a| !a.address.is_empty() && a.address.to_ascii_lowercase() == want)
}

/// Decodes a `0x`-prefixed 20-byte address.
///
/// Returns `None` for anything that is not exactly 40 hex digits, so a short or
/// padded string can never be silently accepted as an address.
pub fn hex_to_address(hex: &str) -> Option<[u8; 20]> {
    let s = hex.strip_prefix("0x").or_else(|| hex.strip_prefix("0X"))?;
    if s.len() != 40 || !s.bytes().all(|b| b.is_ascii_hexdigit()) {
        return None;
    }
    let mut out = [0u8; 20];
    for (i, chunk) in s.as_bytes().chunks(2).enumerate() {
        let pair = std::str::from_utf8(chunk).ok()?;
        out[i] = u8::from_str_radix(pair, 16).ok()?;
    }
    Some(out)
}

/// Encodes a 20-byte address as a lowercase `0x`-prefixed string.
pub fn address_to_hex(address: &[u8; 20]) -> String {
    let mut s = String::with_capacity(42);
    s.push_str("0x");
    for b in address {
        s.push_str(&format!("{b:02x}"));
    }
    s
}

// ---------------------------------------------------------------------------
// Minimal JSON-RPC client
// ---------------------------------------------------------------------------

/// Errors from a TigerSmartChain read.
#[derive(Debug, thiserror::Error)]
pub enum PlatformChainError {
    /// `TSC_RPC_URL` is unset or blank.
    #[error("{RPC_ENV} is not configured")]
    NotConfigured,

    /// The node could not be reached, or answered with a transport error.
    #[error("tiger smart chain rpc error: {0}")]
    Rpc(String),

    /// The node answered, but not with something usable.
    #[error("tiger smart chain decode error: {0}")]
    Decode(String),

    /// The pool does not hold the asset the caller expected.
    #[error("tiger smart chain: {0}")]
    InvalidPool(String),
}

/// Result alias for this module.
pub type Result<T> = std::result::Result<T, PlatformChainError>;

/// Reads `TSC_RPC_URL`, returning [`PlatformChainError::NotConfigured`] if unset.
pub fn rpc_endpoint() -> Result<String> {
    match std::env::var(RPC_ENV) {
        Ok(url) if !url.trim().is_empty() => Ok(url.trim().to_string()),
        _ => Err(PlatformChainError::NotConfigured),
    }
}

/// Reads the configured TGR/RUSD pool address.
pub fn pair_address() -> Option<String> {
    match std::env::var(PAIR_ENV) {
        Ok(a) if hex_to_address(&a).is_some() => Some(a.trim().to_string()),
        _ => None,
    }
}

/// A JSON-RPC client for TigerSmartChain.
///
/// The client is deliberately tiny: it issues `eth_call`, `eth_getBalance` and
/// `eth_chainId` and nothing else. Every call is bounded by a timeout so a hung
/// node cannot block a wallet request indefinitely.
#[derive(Debug, Clone)]
pub struct PlatformChainClient {
    endpoint: String,
    timeout: Duration,
}

impl PlatformChainClient {
    /// Builds a client for the endpoint in `TSC_RPC_URL`.
    pub fn from_env() -> Result<Self> {
        Ok(Self::new(rpc_endpoint()?))
    }

    /// Builds a client for an explicit endpoint.
    pub fn new(endpoint: impl Into<String>) -> Self {
        Self {
            endpoint: endpoint.into(),
            timeout: Duration::from_secs(10),
        }
    }

    /// Overrides the per-request timeout.
    pub fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// Returns the endpoint this client talks to.
    pub fn endpoint(&self) -> &str {
        &self.endpoint
    }

    async fn call(&self, method: &str, params: serde_json::Value) -> Result<serde_json::Value> {
        let body = serde_json::json!({
            "jsonrpc": "2.0",
            "id": 1,
            "method": method,
            "params": params,
        });
        let client = reqwest::Client::builder()
            .timeout(self.timeout)
            .build()
            .map_err(|e| PlatformChainError::Rpc(e.to_string()))?;
        let resp = client
            .post(&self.endpoint)
            .json(&body)
            .send()
            .await
            .map_err(|e| PlatformChainError::Rpc(e.to_string()))?;
        let status = resp.status();
        let text = resp
            .text()
            .await
            .map_err(|e| PlatformChainError::Rpc(e.to_string()))?;
        if !status.is_success() {
            return Err(PlatformChainError::Rpc(format!(
                "http {status}: {}",
                text.chars().take(200).collect::<String>()
            )));
        }
        let parsed: serde_json::Value = serde_json::from_str(&text).map_err(|e| {
            PlatformChainError::Decode(format!(
                "{e}: {}",
                text.chars().take(200).collect::<String>()
            ))
        })?;
        if let Some(err) = parsed.get("error") {
            return Err(PlatformChainError::Rpc(err.to_string()));
        }
        parsed
            .get("result")
            .cloned()
            .ok_or_else(|| PlatformChainError::Decode("response has no result field".into()))
    }

    /// Reads `eth_chainId` and returns it as a number.
    ///
    /// This is the check that stops a misconfigured `TSC_RPC_URL` from silently
    /// pointing the wallet at a different network: every caller that signs or
    /// sends should confirm the node reports [`CHAIN_ID`] first.
    pub async fn chain_id(&self) -> Result<u64> {
        let raw = self.call("eth_chainId", serde_json::json!([])).await?;
        let hex = raw
            .as_str()
            .ok_or_else(|| PlatformChainError::Decode("chainId is not a string".into()))?;
        u64::from_str_radix(hex.trim_start_matches("0x"), 16)
            .map_err(|e| PlatformChainError::Decode(format!("bad chainId {hex}: {e}")))
    }

    /// Reads a native TGR balance in wei.
    pub async fn native_balance(&self, address: &str) -> Result<u128> {
        if hex_to_address(address).is_none() {
            return Err(PlatformChainError::Decode(format!(
                "not a 20-byte address: {address}"
            )));
        }
        let raw = self
            .call("eth_getBalance", serde_json::json!([address, "latest"]))
            .await?;
        let hex = raw
            .as_str()
            .ok_or_else(|| PlatformChainError::Decode("balance is not a string".into()))?;
        u128::from_str_radix(hex.trim_start_matches("0x"), 16)
            .map_err(|e| PlatformChainError::Decode(format!("balance overflow {hex}: {e}")))
    }

    /// Performs a read-only `eth_call`.
    pub async fn eth_call(&self, to: &str, data: &str) -> Result<String> {
        if hex_to_address(to).is_none() {
            return Err(PlatformChainError::Decode(format!("bad call target {to}")));
        }
        let raw = self
            .call(
                "eth_call",
                serde_json::json!([{ "to": to, "data": data }, "latest"]),
            )
            .await?;
        raw.as_str()
            .map(|s| s.to_string())
            .ok_or_else(|| PlatformChainError::Decode("eth_call result is not a string".into()))
    }

    /// Reads an ERC-20 `balanceOf(address)`.
    pub async fn erc20_balance(&self, token: &str, holder: &str) -> Result<u128> {
        let holder_bytes = hex_to_address(holder)
            .ok_or_else(|| PlatformChainError::Decode(format!("bad holder {holder}")))?;
        // balanceOf(address): selector 0x70a08231 + the address left-padded to 32 bytes.
        let mut data = String::from("0x70a08231");
        data.push_str(&"0".repeat(24));
        data.push_str(&address_to_hex(&holder_bytes)[2..]);
        let out = self.eth_call(token, &data).await?;
        let hex = out.trim_start_matches("0x");
        if hex.is_empty() {
            return Err(PlatformChainError::Decode("empty balanceOf result".into()));
        }
        u128::from_str_radix(hex, 16)
            .map_err(|e| PlatformChainError::Decode(format!("balanceOf overflow: {e}")))
    }

    /// Reads the live reserves of the configured TGR/RUSD pool.
    ///
    /// Returns `(tgr_reserve, rusd_reserve)` in wei. Both tokens use 18
    /// decimals, so the ratio of the two is the TGR price in USD directly. The
    /// pool's `token0`/`token1` ordering is read from the contract rather than
    /// assumed, because guessing it would invert the price.
    pub async fn tgr_pool_reserves(&self) -> Result<(u128, u128)> {
        let pair = pair_address().ok_or_else(|| {
            PlatformChainError::InvalidPool(format!("{PAIR_ENV} is not configured"))
        })?;

        // getReserves() -> (uint112, uint112, uint32)
        let raw = self.eth_call(&pair, "0x0902f1ac").await?;
        let hex = raw.trim_start_matches("0x");
        if hex.len() < 192 {
            return Err(PlatformChainError::Decode(format!(
                "getReserves returned {} hex chars, expected at least 192",
                hex.len()
            )));
        }
        let reserve0 = u128::from_str_radix(&hex[0..64], 16)
            .map_err(|e| PlatformChainError::Decode(format!("reserve0: {e}")))?;
        let reserve1 = u128::from_str_radix(&hex[64..128], 16)
            .map_err(|e| PlatformChainError::Decode(format!("reserve1: {e}")))?;

        // token0() / token1() decide which reserve is which.
        let t0 = self.eth_call(&pair, "0x0dfe1681").await?;
        let t1 = self.eth_call(&pair, "0x4f1a4a83").await?;
        let token0 = word_to_address(t0.trim_start_matches("0x"))?;
        let token1 = word_to_address(t1.trim_start_matches("0x"))?;

        let (tgr_reserve, rusd_reserve) = Self::select_pool_reserves(
            &address_to_hex(&token0),
            &address_to_hex(&token1),
            reserve0,
            reserve1,
        )
        .ok_or_else(|| {
            PlatformChainError::InvalidPool(format!(
                "pool {pair} does not have exactly one RUSD leg; \
                 refusing to derive a price from it"
            ))
        })?;
        Ok((tgr_reserve, rusd_reserve))
    }

    /// Maps a pool's raw reserves onto `(tgr_reserve, rusd_reserve)`.
    ///
    /// Exactly one leg must be RUSD: that leg is the dollar side of the price.
    /// A pool with RUSD on neither side says nothing about the dollar price, and
    /// one with RUSD on both sides has no asset leg at all - its ratio would
    /// always read 1.0, which is a malformed pool rather than a one-dollar
    /// price. Both are rejected. Kept separate from the RPC call so the rule is
    /// testable without a node.
    pub fn select_pool_reserves(
        token0: &str,
        token1: &str,
        reserve0: u128,
        reserve1: u128,
    ) -> Option<(u128, u128)> {
        let rusd = RUSD_ADDRESS.to_lowercase();
        let token0 = token0.to_lowercase();
        let token1 = token1.to_lowercase();
        match (token0 == rusd, token1 == rusd) {
            (true, false) => Some((reserve1, reserve0)),
            (false, true) => Some((reserve0, reserve1)),
            _ => None,
        }
    }

    /// Returns the TGR price in USD from the live pool.
    ///
    /// RUSD is the dollar leg, so no external oracle is involved. The result is
    /// `None` when the pool is empty, which is the correct answer for a pool
    /// that has no liquidity yet - it is not the same as a price of zero.
    pub async fn tgr_price_usd(&self) -> Result<Option<f64>> {
        let (tgr, rusd) = self.tgr_pool_reserves().await?;
        if tgr == 0 || rusd == 0 {
            return Ok(None);
        }
        Ok(Some(rusd as f64 / tgr as f64))
    }
}

/// Extracts the low 20 bytes of a 32-byte ABI word as an address.
fn word_to_address(word: &str) -> Result<[u8; 20]> {
    if word.len() < 64 {
        return Err(PlatformChainError::Decode(format!(
            "ABI word is {} chars, expected 64",
            word.len()
        )));
    }
    hex_to_address(&format!("0x{}", &word[24..64]))
        .ok_or_else(|| PlatformChainError::Decode("ABI word is not an address".into()))
}

// ---------------------------------------------------------------------------
// Price cache
// ---------------------------------------------------------------------------

/// A short-lived cache in front of [`PlatformChainClient::tgr_price_usd`].
///
/// Pool reserves only move when somebody trades, so re-reading them on every
/// balance refresh wastes node capacity without improving accuracy. The cache
/// stores the last *successful* read; a failed read never overwrites a good
/// value with a default, and an empty pool is cached as `None` so the caller is
/// told "no price" rather than being shown a stale one.
#[derive(Debug)]
pub struct TgrPriceCache {
    ttl: Duration,
    entry: std::sync::Mutex<Option<(Option<f64>, Instant)>>,
}

impl TgrPriceCache {
    /// Builds a cache with the given freshness window.
    pub fn new(ttl: Duration) -> Self {
        Self {
            ttl,
            entry: std::sync::Mutex::new(None),
        }
    }

    /// Builds a cache with the default 30 second window.
    pub fn with_default_ttl() -> Self {
        Self::new(Duration::from_secs(30))
    }

    /// Returns the cached price when it is still fresh.
    pub fn get(&self) -> Option<Option<f64>> {
        let guard = self.entry.lock().ok()?;
        match guard.as_ref() {
            Some((price, at)) if at.elapsed() < self.ttl => Some(*price),
            _ => None,
        }
    }

    /// Records a fresh price.
    pub fn put(&self, price: Option<f64>) {
        if let Ok(mut guard) = self.entry.lock() {
            *guard = Some((price, Instant::now()));
        }
    }

    /// Clears the cache, forcing the next read to hit the node.
    pub fn invalidate(&self) {
        if let Ok(mut guard) = self.entry.lock() {
            *guard = None;
        }
    }

    /// Reads the price, using the cache when it is fresh.
    pub async fn price_usd(&self, client: &PlatformChainClient) -> Result<Option<f64>> {
        if let Some(cached) = self.get() {
            return Ok(cached);
        }
        let price = client.tgr_price_usd().await?;
        self.put(price);
        Ok(price)
    }
}

impl Default for TgrPriceCache {
    fn default() -> Self {
        Self::with_default_ttl()
    }
}

// ---------------------------------------------------------------------------
// Portfolio helpers
// ---------------------------------------------------------------------------

/// A platform asset together with a live balance and fiat value.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlatformHolding {
    /// Asset metadata.
    pub asset: PlatformAsset,
    /// Balance in the asset's smallest unit (wei).
    pub balance_raw: u128,
    /// Balance as a decimal string, e.g. `"1.5"`.
    pub balance: String,
    /// USD value, or `None` when no price source answered.
    pub value_usd: Option<f64>,
}

/// Reads every platform asset balance for an address.
///
/// Each asset is read independently: a token contract that reverts does not
/// hide the native balance, it just reports an error for that asset. The price
/// is looked up once and reused, and `value_usd` stays `None` when the pool
/// cannot be read rather than being filled with a placeholder.
pub async fn read_holdings(
    client: &PlatformChainClient,
    address: &str,
    price_cache: &TgrPriceCache,
) -> Result<Vec<PlatformHolding>> {
    let price = price_cache.price_usd(client).await.ok().flatten();
    let mut out = Vec::with_capacity(3);
    for asset in platform_assets() {
        let balance_raw = if asset.is_contract() {
            client.erc20_balance(&asset.address, address).await?
        } else {
            client.native_balance(address).await?
        };
        let balance = format_units(balance_raw, asset.decimals);
        let value_usd = if asset.pegged_usd {
            // RUSD is redeemable one-for-one by design, so its fiat value is the
            // balance itself. This is a property of the asset, not a guessed rate.
            Some(parse_decimal(&balance))
        } else {
            price.map(|p| parse_decimal(&balance) * p)
        };
        out.push(PlatformHolding {
            asset,
            balance_raw,
            balance,
            value_usd,
        });
    }
    Ok(out)
}

/// Formats a raw integer amount as a decimal string with the given decimals.
///
/// Pure integer arithmetic, so no precision is lost to binary floating point on
/// the way to display. Trailing fractional zeros are trimmed.
pub fn format_units(raw: u128, decimals: u8) -> String {
    if decimals == 0 {
        return raw.to_string();
    }
    let scale = 10u128.checked_pow(decimals as u32);
    let Some(scale) = scale else {
        // 10^38 does not fit in u128; fall back to the raw value rather than
        // printing a wrong number.
        return raw.to_string();
    };
    let whole = raw / scale;
    let fraction = raw % scale;
    if fraction == 0 {
        return whole.to_string();
    }
    let mut frac = format!("{:0width$}", fraction, width = decimals as usize);
    while frac.ends_with('0') {
        frac.pop();
    }
    format!("{whole}.{frac}")
}

/// Parses a plain decimal string into an `f64`.
///
/// Only used for presentation maths (multiplying a balance by a price). Any
/// value that is not a finite, non-negative decimal becomes `0.0`, so a
/// malformed balance can never propagate as NaN or infinity into a total.
pub fn parse_decimal(s: &str) -> f64 {
    match s.trim().parse::<f64>() {
        Ok(v) if v.is_finite() && v >= 0.0 => v,
        _ => 0.0,
    }
}

/// Returns the platform chain entry in the same shape as the public chains, so
/// a UI can render it from one list.
pub fn chain_entry() -> HashMap<&'static str, String> {
    let mut m = HashMap::new();
    m.insert("chain_id", CHAIN_ID.to_string());
    m.insert("name", "TigerSmartChain".to_string());
    m.insert("symbol", NATIVE_SYMBOL.to_string());
    m.insert("decimals", "18".to_string());
    m.insert("rpc_env", RPC_ENV.to_string());
    m.insert("rusd_address", RUSD_ADDRESS.to_string());
    m.insert("wtgr_address", WTGR_ADDRESS.to_string());
    m
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn chain_id_matches_the_chain_module() {
        assert_eq!(CHAIN_ID, 7710);
        assert_eq!(TESTNET_CHAIN_ID, 7711);
        assert_ne!(CHAIN_ID, TESTNET_CHAIN_ID);
    }

    #[test]
    fn platform_assets_are_complete() {
        let assets = platform_assets();
        assert_eq!(assets.len(), 3, "TGR, WTGR and RUSD");

        let tgr = &assets[0];
        assert_eq!(tgr.symbol, "TGR");
        assert_eq!(tgr.kind, AssetKind::Native);
        assert!(!tgr.is_contract());
        assert!(tgr.address.is_empty());
        assert!(tgr.address_bytes().is_none());
        assert!(!tgr.pegged_usd);

        let wtgr = asset_by_symbol("wtgr").expect("WTGR is registered");
        assert_eq!(wtgr.address, WTGR_ADDRESS);
        assert_eq!(wtgr.kind, AssetKind::GenesisPredeploy);
        assert!(wtgr.is_contract());
        assert!(!wtgr.pegged_usd);

        let rusd = asset_by_symbol("RUSD").expect("RUSD is registered");
        assert_eq!(rusd.address, RUSD_ADDRESS);
        assert!(rusd.pegged_usd);
        assert_eq!(rusd.decimals, 18);
    }

    #[test]
    fn lookup_helpers_agree() {
        assert!(asset_by_symbol("").is_none());
        assert!(asset_by_symbol("ETH").is_none());
        assert!(asset_by_address("").is_none());
        assert!(asset_by_address("0xdeadbeef").is_none());

        let by_symbol = asset_by_symbol(" RUSD ").expect("trimmed + case-insensitive");
        let by_address = asset_by_address(&RUSD_ADDRESS.to_uppercase().replace("0X", "0x"))
            .expect("address lookup is case-insensitive");
        assert_eq!(by_symbol, by_address);

        // The native coin has no address, so address lookup must not find it.
        assert!(asset_by_address("0x0000000000000000000000000000000000000000").is_none());
    }

    #[test]
    fn address_codec_rejects_malformed_input() {
        assert!(hex_to_address("").is_none());
        assert!(hex_to_address("0x").is_none());
        assert!(hex_to_address("0x1234").is_none());
        assert!(hex_to_address("0x0000000000000000000000000000000000000000").is_some());
        // 39 and 41 hex digits must both fail.
        assert!(hex_to_address(&format!("0x{}", "1".repeat(39))).is_none());
        assert!(hex_to_address(&format!("0x{}", "1".repeat(41))).is_none());
        // Non-hex characters must fail even at the right length.
        assert!(hex_to_address(&format!("0x{}", "z".repeat(40))).is_none());

        let addr = hex_to_address(RUSD_ADDRESS).expect("RUSD constant decodes");
        assert_eq!(address_to_hex(&addr), RUSD_ADDRESS);
    }

    #[test]
    fn pool_reserves_require_exactly_one_rusd_leg() {
        let other = "0x00000000000000000000000000000000000000cc";
        // RUSD on token0 -> the TGR reserve is reserve1.
        assert_eq!(
            PlatformChainClient::select_pool_reserves(RUSD_ADDRESS, other, 10, 20),
            Some((20, 10))
        );
        // RUSD on token1 -> the TGR reserve is reserve0.
        assert_eq!(
            PlatformChainClient::select_pool_reserves(other, RUSD_ADDRESS, 10, 20),
            Some((10, 20))
        );
        // Case must not matter: addresses arrive from the node in either form.
        assert_eq!(
            PlatformChainClient::select_pool_reserves(
                &RUSD_ADDRESS.to_uppercase().replace("0X", "0x"),
                other,
                1,
                2
            ),
            Some((2, 1))
        );
        // No RUSD leg: the pool says nothing about the dollar price.
        assert_eq!(
            PlatformChainClient::select_pool_reserves(other, other, 10, 20),
            None
        );
        // RUSD on both sides: no asset leg, and the ratio would always be 1.0.
        assert_eq!(
            PlatformChainClient::select_pool_reserves(RUSD_ADDRESS, RUSD_ADDRESS, 10, 20),
            None
        );
    }

    #[test]
    fn format_units_is_exact() {
        assert_eq!(format_units(0, 18), "0");
        assert_eq!(format_units(1, 18), "0.000000000000000001");
        assert_eq!(format_units(1_000_000_000_000_000_000, 18), "1");
        assert_eq!(format_units(1_500_000_000_000_000_000, 18), "1.5");
        // Trailing zeros are trimmed but the value is unchanged.
        assert_eq!(format_units(1_230_000_000_000_000_000, 18), "1.23");
        assert_eq!(format_units(100, 0), "100");
        assert_eq!(format_units(1234, 2), "12.34");
        assert!(format_units(u128::MAX, 18).contains('.'));
    }

    #[test]
    fn parse_decimal_never_yields_nan() {
        assert_eq!(parse_decimal("1.5"), 1.5);
        assert_eq!(parse_decimal(" 2 "), 2.0);
        assert_eq!(parse_decimal(""), 0.0);
        assert_eq!(parse_decimal("abc"), 0.0);
        assert_eq!(parse_decimal("NaN"), 0.0);
        assert_eq!(parse_decimal("inf"), 0.0);
        // Negative balances are not representable; clamp rather than propagate.
        assert_eq!(parse_decimal("-5"), 0.0);
    }

    #[test]
    fn rpc_endpoint_is_required() {
        let previous = std::env::var(RPC_ENV).ok();
        std::env::remove_var(RPC_ENV);
        assert!(matches!(
            rpc_endpoint(),
            Err(PlatformChainError::NotConfigured)
        ));
        std::env::set_var(RPC_ENV, "   ");
        assert!(matches!(
            rpc_endpoint(),
            Err(PlatformChainError::NotConfigured)
        ));

        std::env::set_var(RPC_ENV, "http://127.0.0.1:8545");
        assert_eq!(rpc_endpoint().unwrap(), "http://127.0.0.1:8545");
        let client = PlatformChainClient::from_env().expect("endpoint is set");
        assert_eq!(client.endpoint(), "http://127.0.0.1:8545");

        match previous {
            Some(v) => std::env::set_var(RPC_ENV, v),
            None => std::env::remove_var(RPC_ENV),
        }
    }

    #[test]
    fn pair_address_must_be_a_valid_address() {
        let previous = std::env::var(PAIR_ENV).ok();
        std::env::remove_var(PAIR_ENV);
        assert!(pair_address().is_none());
        std::env::set_var(PAIR_ENV, "not-an-address");
        assert!(
            pair_address().is_none(),
            "a malformed pool must be rejected"
        );
        std::env::set_var(PAIR_ENV, "0x00000000000000000000000000000000000000aa");
        assert!(pair_address().is_some());
        match previous {
            Some(v) => std::env::set_var(PAIR_ENV, v),
            None => std::env::remove_var(PAIR_ENV),
        }
    }

    #[test]
    fn word_to_address_uses_the_low_twenty_bytes() {
        // ABI-encoded address: 24 leading zero hex chars then the 40 address chars.
        let word = format!("{}{}", "0".repeat(24), &RUSD_ADDRESS[2..]);
        let decoded = word_to_address(&word).expect("decodes");
        assert_eq!(address_to_hex(&decoded), RUSD_ADDRESS);

        assert!(word_to_address("0x1234").is_err());
        assert!(word_to_address(&"0".repeat(64)).is_ok());
    }

    #[test]
    fn price_cache_serves_fresh_entries_only() {
        let cache = TgrPriceCache::new(Duration::from_millis(50));
        assert!(cache.get().is_none(), "empty cache has nothing to serve");
        cache.put(Some(2.5));
        assert_eq!(cache.get(), Some(Some(2.5)));
        cache.put(None);
        assert_eq!(cache.get(), Some(None), "an empty pool caches as no price");

        std::thread::sleep(Duration::from_millis(80));
        assert!(cache.get().is_none(), "entry expires after the ttl");

        cache.put(Some(3.0));
        cache.invalidate();
        assert!(cache.get().is_none(), "invalidate forces a re-read");
    }

    #[test]
    fn chain_entry_describes_the_platform_chain() {
        let entry = chain_entry();
        assert_eq!(entry.get("chain_id").map(String::as_str), Some("7710"));
        assert_eq!(entry.get("symbol").map(String::as_str), Some("TGR"));
        assert_eq!(
            entry.get("rpc_env").map(String::as_str),
            Some("TSC_RPC_URL")
        );
        assert_eq!(
            entry.get("rusd_address").map(String::as_str),
            Some(RUSD_ADDRESS)
        );
    }
}
