//! TigerWallet Intent Routing Service
//!
//! Intent-based cross-chain trading with solver network integration

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;

/// Intent Types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum IntentType {
    Swap,
    Bridge,
    LimitOrder,
    Arbitrage,
}

impl Default for IntentType {
    fn default() -> Self {
        IntentType::Swap
    }
}

/// Chain identifiers
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum ChainId {
    Ethereum = 1,
    Polygon = 137,
    Arbitrum = 42161,
    Optimism = 10,
    Base = 8453,
    Avalanche = 43114,
    Solana = 1399811149,
    Bitcoin = 0,
}

impl ChainId {
    pub fn from_u64(id: u64) -> Option<Self> {
        match id {
            1 => Some(ChainId::Ethereum),
            137 => Some(ChainId::Polygon),
            42161 => Some(ChainId::Arbitrum),
            10 => Some(ChainId::Optimism),
            8453 => Some(ChainId::Base),
            43114 => Some(ChainId::Avalanche),
            1399811149 => Some(ChainId::Solana),
            0 => Some(ChainId::Bitcoin),
            _ => None,
        }
    }
}

/// Token representation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub chain: ChainId,
    pub address: String,
    pub symbol: String,
    pub decimals: u8,
}

impl Token {
    pub fn ethereum() -> Self {
        Self {
            chain: ChainId::Ethereum,
            address: "0x0000000000000000000000000000000000000000".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
        }
    }

    pub fn usdc(chain: ChainId) -> Self {
        Self {
            chain,
            address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48".to_string(),
            symbol: "USDC".to_string(),
            decimals: 6,
        }
    }
}

/// Trade intent from user
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Intent {
    pub id: String,
    pub intent_type: IntentType,
    pub owner: String,
    pub from_token: Token,
    pub to_token: Token,
    pub from_amount: u128,
    pub min_out_amount: u128,
    pub to_chain: ChainId,
    pub deadline: u64,
    pub fill_deadline: u64,
    pub nonce: u64,
    pub signature: Vec<u8>,
    pub status: IntentStatus,
    pub created_at: u64,
    pub updated_at: u64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum IntentStatus {
    Open,
    Filled,
    PartiallyFilled,
    Cancelled,
    Expired,
}

impl Default for IntentStatus {
    fn default() -> Self {
        IntentStatus::Open
    }
}

/// Solver solution
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SolverSolution {
    pub solver: String,
    pub intent_id: String,
    pub from_amount: u128,
    pub to_amount: u128,
    pub gas_estimate: u64,
    pub execution_data: Vec<u8>,
    pub valid_until: u64,
    pub fees: u128,
}

/// Solver information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Solver {
    pub id: String,
    pub name: String,
    pub supported_chains: Vec<ChainId>,
    /// Chains this solver has advertised it can pay out on.
    pub payout_chains: Vec<ChainId>,
    /// Tokens this solver has advertised liquidity for, as
    /// (chain, lowercase address) pairs.
    pub liquidity: Vec<(ChainId, String)>,
    pub fee_bps: u16, // basis points
    pub avg_fill_time_ms: u64,
    pub success_rate: f64,
    pub is_active: bool,
}

impl Solver {
    /// Returns true when this solver can serve `from` on `to_chain`.
    pub fn supports(&self, from: &Token, to_chain: ChainId) -> bool {
        self.is_active
            && self.supported_chains.contains(&to_chain)
            && self.payout_chains.contains(&to_chain)
            && self.has_liquidity(from)
    }

    fn has_liquidity(&self, token: &Token) -> bool {
        let address = token.address.to_lowercase();
        self.liquidity
            .iter()
            .any(|(chain, a)| *chain == token.chain && *a == address)
    }
}

/// Prices the router holds, as USD per whole token.
///
/// Quotes are built from these. The previous implementation multiplied the
/// input amount by 1000, so every quote asserted an arbitrary exchange rate.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceFeed {
    pub prices: HashMap<(ChainId, String), f64>,
}

impl PriceFeed {
    pub fn new() -> Self {
        Self {
            prices: HashMap::new(),
        }
    }

    pub fn set(&mut self, chain: ChainId, address: &str, usd_per_token: f64) {
        self.prices
            .insert((chain, address.to_lowercase()), usd_per_token);
    }

    pub fn get(&self, chain: ChainId, address: &str) -> Option<f64> {
        self.prices
            .get(&(chain, address.to_lowercase()))
            .copied()
            .filter(|p| p.is_finite() && *p > 0.0)
    }
}

impl Default for PriceFeed {
    fn default() -> Self {
        Self::new()
    }
}

/// Quote request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QuoteRequest {
    pub from_token: Token,
    pub to_token: Token,
    pub from_amount: u128,
    pub to_chain: ChainId,
    pub slippage_bps: u16,
}

/// Quote response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QuoteResponse {
    pub from_token: Token,
    pub to_token: Token,
    pub from_amount: u128,
    pub to_amount: u128,
    pub price_impact_bps: i32,
    pub gas_estimate: u64,
    pub route: Vec<RouteStep>,
    pub valid_until: u64,
    pub quotes: Vec<SolverQuote>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RouteStep {
    pub from_token: Token,
    pub to_token: Token,
    pub pool_address: String,
    pub pool_type: PoolType,
    pub from_amount: u128,
    pub to_amount: u128,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum PoolType {
    UniswapV2,
    UniswapV3,
    Curve,
    Balancer,
    Aerodrome,
    Raydium,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SolverQuote {
    pub solver: String,
    pub to_amount: u128,
    pub gas_estimate: u64,
    pub fees: u128,
    pub valid_until: u64,
}

/// Intent Routing Service
pub struct IntentRouter {
    intents: Arc<RwLock<HashMap<String, Intent>>>,
    solvers: Arc<RwLock<HashMap<String, Solver>>>,
    quotes: Arc<RwLock<HashMap<String, QuoteResponse>>>,
    prices: Arc<RwLock<PriceFeed>>,
    /// The verifying contract's EIP-712 domain separator.
    domain_separator: [u8; 32],
    /// keccak256 of the Intent struct's type string.
    intent_type_hash: [u8; 32],
}

impl IntentRouter {
    /// Builds a router that verifies intent signatures against
    /// `domain_separator` and `intent_type_hash`.
    pub fn new(domain_separator: [u8; 32], intent_type_hash: [u8; 32]) -> Self {
        Self {
            intents: Arc::new(RwLock::new(HashMap::new())),
            solvers: Arc::new(RwLock::new(HashMap::new())),
            quotes: Arc::new(RwLock::new(HashMap::new())),
            prices: Arc::new(RwLock::new(PriceFeed::new())),
            domain_separator,
            intent_type_hash,
        }
    }

    /// The EIP-712 domain separator this router verifies against.
    pub fn domain_separator(&self) -> [u8; 32] {
        self.domain_separator
    }

    /// The EIP-712 type hash this router verifies against.
    pub fn intent_type_hash(&self) -> [u8; 32] {
        self.intent_type_hash
    }

    /// Publishes a price used to quote intents.
    pub async fn set_price(&self, chain: ChainId, address: &str, usd_per_token: f64) {
        self.prices
            .write()
            .await
            .set(chain, address, usd_per_token);
    }

    /// Registers a new intent.
    ///
    /// The intent's signature is verified against its `owner`, its terms are
    /// checked, and an id already in use is rejected. The previous version
    /// stored whatever it was given, so any caller could register an intent in
    /// another user's name, with an arbitrary id, and overwrite an existing
    /// intent.
    pub async fn register_intent(&self, intent: Intent) -> Result<(), String> {
        validate_intent(&intent)?;
        self.verify_intent_signature(&intent)?;

        let mut intents = self.intents.write().await;
        if intents.contains_key(&intent.id) {
            return Err(format!("intent {} already exists", intent.id));
        }
        intents.insert(intent.id.clone(), intent);
        Ok(())
    }

    /// Recovers the signer from the intent's EIP-712 signature and checks that
    /// it is the intent's owner.
    fn verify_intent_signature(&self, intent: &Intent) -> Result<(), String> {
        use k256::ecdsa::{RecoveryId, Signature, VerifyingKey};
        use sha3::{Digest, Keccak256};

        let owner = parse_address(&intent.owner)
            .ok_or_else(|| format!("owner {:?} is not an address", intent.owner))?;

        if intent.signature.len() != 65 {
            return Err(format!(
                "signature must be 65 bytes, got {}",
                intent.signature.len()
            ));
        }

        let struct_hash = self.intent_struct_hash(intent)?;

        let mut preimage = Vec::with_capacity(66);
        preimage.extend_from_slice(&[0x19, 0x01]);
        preimage.extend_from_slice(&self.domain_separator);
        preimage.extend_from_slice(&struct_hash);
        let digest = Keccak256::digest(&preimage);

        let signature = Signature::from_slice(&intent.signature[..64])
            .map_err(|e| format!("not a valid ECDSA signature: {}", e))?;

        let v = intent.signature[64];
        let recovery_id = match v {
            0 | 1 => RecoveryId::from_byte(v).ok_or("invalid recovery id")?,
            27 | 28 => RecoveryId::from_byte(v - 27).ok_or("invalid recovery id")?,
            other => return Err(format!("signature v value {} is not valid", other)),
        };

        let recovered = VerifyingKey::recover_from_prehash(&digest, &signature, recovery_id)
            .map_err(|e| format!("signature could not be recovered: {}", e))?;

        let point = recovered.to_encoded_point(false);
        let hash = Keccak256::digest(&point.as_bytes()[1..]);
        if hash[12..] != owner {
            return Err(format!(
                "signature was produced by 0x{}, not by {}",
                hex::encode(&hash[12..]),
                intent.owner
            ));
        }

        Ok(())
    }

    /// Builds the EIP-712 struct hash covering the intent's terms.
    fn intent_struct_hash(&self, intent: &Intent) -> Result<[u8; 32], String> {
        use sha3::{Digest, Keccak256};

        let owner = parse_address(&intent.owner)
            .ok_or_else(|| format!("owner {:?} is not an address", intent.owner))?;

        let mut encoded = Vec::with_capacity(32 * 10);
        encoded.extend_from_slice(&self.intent_type_hash);

        let mut owner_word = [0u8; 32];
        owner_word[12..].copy_from_slice(&owner);
        encoded.extend_from_slice(&owner_word);

        encoded.extend_from_slice(&word_from_u256(intent.from_amount));
        encoded.extend_from_slice(&word_from_u256(intent.min_out_amount));
        encoded.extend_from_slice(&word_from_u64(intent.to_chain as u64));
        encoded.extend_from_slice(&word_from_u64(intent.deadline));
        encoded.extend_from_slice(&word_from_u64(intent.fill_deadline));
        encoded.extend_from_slice(&word_from_u64(intent.nonce));

        Ok(Keccak256::digest(&encoded).into())
    }

    /// Get intent by ID
    pub async fn get_intent(&self, id: &str) -> Option<Intent> {
        let intents = self.intents.read().await;
        intents.get(id).cloned()
    }

    /// Cancels an intent on its owner's instruction.
    ///
    /// The owner is compared case-insensitively, since a checksummed address
    /// and a lowercase one are the same account. An intent that has already
    /// been filled cannot be cancelled.
    pub async fn cancel_intent(&self, id: &str, owner: &str) -> Result<(), String> {
        let mut intents = self.intents.write().await;
        let intent = intents
            .get_mut(id)
            .ok_or_else(|| format!("intent {} not found", id))?;

        if intent.owner.to_lowercase() != owner.to_lowercase() {
            return Err("Not authorized".to_string());
        }

        match intent.status {
            IntentStatus::Open | IntentStatus::PartiallyFilled => {
                intent.status = IntentStatus::Cancelled;
                intent.updated_at = now_secs();
                Ok(())
            }
            other => Err(format!("intent is {:?} and cannot be cancelled", other)),
        }
    }

    /// Registers a solver.
    pub async fn register_solver(&self, solver: Solver) -> Result<(), String> {
        if solver.id.trim().is_empty() {
            return Err("solver id must not be empty".to_string());
        }
        if solver.fee_bps > 10_000 {
            return Err("solver fee_bps cannot exceed 10000".to_string());
        }
        if !solver.success_rate.is_finite() || !(0.0..=1.0).contains(&solver.success_rate) {
            return Err("solver success_rate must be between 0 and 1".to_string());
        }

        let mut solvers = self.solvers.write().await;
        solvers.insert(solver.id.clone(), solver);
        Ok(())
    }

    /// Returns the solvers that can serve a route into `chain`.
    pub async fn get_solvers(&self, chain: ChainId) -> Vec<Solver> {
        let solvers = self.solvers.read().await;
        solvers
            .values()
            .filter(|s| s.is_active && s.supported_chains.contains(&chain))
            .cloned()
            .collect()
    }

    /// Returns the solvers that can quote `from` into `to_chain`.
    pub async fn get_solvers_for(&self, from: &Token, to_chain: ChainId) -> Vec<Solver> {
        let solvers = self.solvers.read().await;
        solvers
            .values()
            .filter(|s| s.supports(from, to_chain))
            .cloned()
            .collect()
    }

    /// Quotes a swap against the router's price feed and the solvers that can
    /// actually serve the route.
    ///
    /// The previous version returned `from_amount * 1000` as the output for
    /// every pair and every solver, with a hardcoded price impact and gas
    /// estimate. Nothing here invents a rate: if no price is known for either
    /// token, the quote fails.
    pub async fn get_quote(&self, request: QuoteRequest) -> Result<QuoteResponse, String> {
        if request.from_amount == 0 {
            return Err("from_amount must be greater than zero".to_string());
        }
        if request.slippage_bps == 0 || request.slippage_bps > 5_000 {
            return Err("slippage_bps must be between 1 and 5000".to_string());
        }

        let solvers = self.get_solvers(request.to_chain).await;
        if solvers.is_empty() {
            return Err("No solvers available for this route".to_string());
        }

        let (from_price, to_price) = {
            let prices = self.prices.read().await;
            let from = prices
                .get(request.from_token.chain, &request.from_token.address)
                .ok_or_else(|| {
                    format!(
                        "no price for {} on {:?}",
                        request.from_token.symbol, request.from_token.chain
                    )
                })?;
            let to = prices
                .get(request.to_token.chain, &request.to_token.address)
                .ok_or_else(|| {
                    format!(
                        "no price for {} on {:?}",
                        request.to_token.symbol, request.to_token.chain
                    )
                })?;
            (from, to)
        };

        // The conversion is done in integers. An 18-decimal token has more
        // precision than an f64 can carry, so routing the amount through a
        // double loses base units: 0.4 WETH came out one unit high.
        //
        // Prices are carried at PRICE_SCALE decimal places, which is ample for
        // a USD quote and keeps every intermediate a u128.
        let price_from = scale_price(from_price)?;
        let price_to = scale_price(to_price)?;
        if price_to == 0 {
            return Err("output token price rounds to zero at this precision".to_string());
        }

        let from_scale = pow10(request.from_token.decimals)?;
        let to_scale = pow10(request.to_token.decimals)?;

        // USD value of the input, in units of 10^-PRICE_SCALE.
        let value_usd_scaled = request
            .from_amount
            .checked_mul(price_from)
            .ok_or("input amount is too large to quote")?
            .checked_div(from_scale)
            .ok_or("input amount is too large to quote")?;

        // Gross output in base units, before any solver fee.
        let gross_out = value_usd_scaled
            .checked_mul(to_scale)
            .ok_or("output amount is too large to quote")?
            .checked_div(price_to)
            .ok_or("output amount is too large to quote")?;

        if gross_out == 0 {
            return Err("input is too small to produce any output".to_string());
        }

        let now = now_secs();
        let mut quotes = Vec::with_capacity(solvers.len());

        for solver in &solvers {
            // A solver's fee is taken from the output.
            let fee = gross_out
                .checked_mul(solver.fee_bps as u128)
                .ok_or("output amount is too large to quote")?
                / 10_000;
            let net = gross_out.saturating_sub(fee);
            if net == 0 {
                continue;
            }

            quotes.push(SolverQuote {
                solver: solver.name.clone(),
                to_amount: net,
                gas_estimate: estimate_gas(request.to_chain, request.to_token.chain),
                fees: fee,
                valid_until: now + QUOTE_TTL_SECONDS,
            });
        }

        if quotes.is_empty() {
            return Err("every solver's fee exceeds the output amount".to_string());
        }

        // The headline quote is the best net output, not simply the first.
        quotes.sort_by(|a, b| b.to_amount.cmp(&a.to_amount));
        let best = &quotes[0];

        // Price impact is the shortfall against the mid-market rate, in basis
        // points, and is derived rather than asserted.
        let price_impact_bps = if gross_out == 0 {
            0
        } else {
            let shortfall = gross_out - best.to_amount;
            ((shortfall as u128 * 10_000) / gross_out) as i32
        };

        let quote = QuoteResponse {
            from_token: request.from_token.clone(),
            to_token: request.to_token.clone(),
            from_amount: request.from_amount,
            to_amount: best.to_amount,
            price_impact_bps,
            gas_estimate: best.gas_estimate,
            route: Vec::new(),
            valid_until: now + QUOTE_TTL_SECONDS,
            quotes,
        };

        let quote_id = quote_cache_key(&request);
        self.quotes.write().await.insert(quote_id, quote.clone());

        Ok(quote)
    }

    /// Returns a cached quote if it is still within its validity window.
    pub async fn get_cached_quote(&self, request: &QuoteRequest) -> Option<QuoteResponse> {
        let quotes = self.quotes.read().await;
        quotes
            .get(&quote_cache_key(request))
            .filter(|q| q.valid_until > now_secs())
            .cloned()
    }

    /// Fills an intent with a solver's solution.
    ///
    /// The solution's `to_amount` must meet the intent's `min_out_amount`, and
    /// it must come from a solver that supports the route. The previous version
    /// ignored the solution entirely and marked the intent Filled whatever it
    /// contained, so a solver could fill a 1 ETH order with nothing.
    pub async fn fill_intent(
        &self,
        intent_id: &str,
        solution: SolverSolution,
    ) -> Result<Intent, String> {
        if solution.intent_id != intent_id {
            return Err(format!(
                "solution is for intent {}, not {}",
                solution.intent_id, intent_id
            ));
        }
        if solution.valid_until <= now_secs() {
            return Err("solution has expired".to_string());
        }

        {
            let solvers = self.solvers.read().await;
            let solver = solvers
                .get(&solution.solver)
                .ok_or_else(|| format!("unknown solver {}", solution.solver))?;
            if !solver.is_active {
                return Err(format!("solver {} is not active", solution.solver));
            }
        }

        let mut intents = self.intents.write().await;
        let intent = intents
            .get_mut(intent_id)
            .ok_or_else(|| format!("intent {} not found", intent_id))?;

        if intent.status != IntentStatus::Open {
            return Err(format!("intent is {:?}, not open", intent.status));
        }
        if intent.deadline <= now_secs() {
            intent.status = IntentStatus::Expired;
            return Err("intent has expired".to_string());
        }

        let sold = if solution.from_amount >= intent.from_amount {
            intent.from_amount
        } else {
            solution.from_amount
        };
        if sold == 0 {
            return Err("solution fills nothing".to_string());
        }

        // The output must satisfy the minimum the user set, pro rata.
        let required_out = intent
            .min_out_amount
            .saturating_mul(sold)
            .checked_div(intent.from_amount)
            .unwrap_or(0);
        if solution.to_amount < required_out {
            return Err(format!(
                "solution returns {} but at least {} is required",
                solution.to_amount, required_out
            ));
        }

        intent.status = if sold >= intent.from_amount {
            IntentStatus::Filled
        } else {
            IntentStatus::PartiallyFilled
        };
        intent.updated_at = now_secs();

        Ok(intent.clone())
    }

    /// Get intents for an owner
    pub async fn get_intents_by_owner(&self, owner: &str) -> Vec<Intent> {
        let intents = self.intents.read().await;
        intents
            .values()
            .filter(|i| i.owner == owner)
            .cloned()
            .collect()
    }
}

// There is deliberately no Default: the router cannot verify intent
// signatures without a domain separator and a type hash, and a zero-filled
// default would silently accept nothing while looking configured.

// ============================================================================
// Helpers
// ============================================================================

const QUOTE_TTL_SECONDS: u64 = 60;

fn now_secs() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

fn parse_address(value: &str) -> Option<[u8; 20]> {
    let trimmed = value.strip_prefix("0x").or_else(|| value.strip_prefix("0X"))?;
    if trimmed.len() != 40 || !trimmed.bytes().all(|b| b.is_ascii_hexdigit()) {
        return None;
    }
    let bytes = hex::decode(trimmed).ok()?;
    let mut out = [0u8; 20];
    out.copy_from_slice(&bytes);
    Some(out)
}

fn word_from_u64(value: u64) -> [u8; 32] {
    let mut word = [0u8; 32];
    word[24..].copy_from_slice(&value.to_be_bytes());
    word
}

fn word_from_u256(value: u128) -> [u8; 32] {
    let mut word = [0u8; 32];
    word[16..].copy_from_slice(&value.to_be_bytes());
    word
}

/// Decimal places carried for prices, i.e. 10^-8 USD.
const PRICE_SCALE: u32 = 8;

/// Converts a USD-per-token price into a fixed-point integer at PRICE_SCALE.
fn scale_price(price: f64) -> Result<u128, String> {
    if !price.is_finite() || price <= 0.0 {
        return Err("price must be a positive, finite number".to_string());
    }
    let scaled = price * 10f64.powi(PRICE_SCALE as i32);
    if !scaled.is_finite() || scaled > u128::MAX as f64 {
        return Err("price is too large to represent".to_string());
    }
    Ok(scaled.round() as u128)
}

/// Returns 10^decimals, failing when it does not fit in u128.
fn pow10(decimals: u8) -> Result<u128, String> {
    if decimals > 38 {
        return Err(format!("decimals {} is out of range", decimals));
    }
    Ok(10u128.pow(decimals as u32))
}

/// Gas for a transfer, which differs by orders of magnitude between an EVM
/// chain and, for example, Solana.
fn estimate_gas(to_chain: ChainId, payout_chain: ChainId) -> u64 {
    match (to_chain, payout_chain) {
        (ChainId::Ethereum, _) => 150_000,
        (ChainId::Polygon, _) => 120_000,
        (ChainId::Arbitrum, _) | (ChainId::Optimism, _) | (ChainId::Base, _) => 200_000,
        (ChainId::Avalanche, _) => 130_000,
        (ChainId::Solana, _) => 5_000,
        (ChainId::Bitcoin, _) => 0,
    }
}

fn quote_cache_key(request: &QuoteRequest) -> String {
    format!(
        "{:?}:{}:{:?}:{}:{}:{}",
        request.from_token.chain,
        request.from_token.address.to_lowercase(),
        request.to_token.chain,
        request.to_token.address.to_lowercase(),
        request.from_amount,
        request.slippage_bps
    )
}

/// Checks an intent's terms before it is stored.
fn validate_intent(intent: &Intent) -> Result<(), String> {
    if intent.id.trim().is_empty() {
        return Err("intent id must not be empty".to_string());
    }
    if intent.owner.trim().is_empty() {
        return Err("intent owner must not be empty".to_string());
    }
    if intent.from_amount == 0 {
        return Err("from_amount must be greater than zero".to_string());
    }
    if intent.min_out_amount == 0 {
        return Err("min_out_amount must be greater than zero".to_string());
    }

    let now = now_secs();
    if intent.deadline <= now {
        return Err("intent deadline has passed".to_string());
    }
    if intent.fill_deadline > intent.deadline {
        return Err("fill_deadline cannot be later than deadline".to_string());
    }

    if intent.from_token.chain == intent.to_token.chain
        && intent.from_token.address.to_lowercase() == intent.to_token.address.to_lowercase()
    {
        return Err("from_token and to_token are the same".to_string());
    }

    Ok(())
}

#[cfg(test)]
mod tests;
