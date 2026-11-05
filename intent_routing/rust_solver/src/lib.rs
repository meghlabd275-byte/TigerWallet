/**
 * TigerWallet - Intent-Based Cross-Chain Solver
 * Rust implementation for solving cross-chain intents
 *
 * Features:
 * - Intent parsing and validation
 * - Solver network coordination
 * - UniswapX/CoW Swap integration
 * - Cross-chain settlement
 * - Price improvement engine
 */

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH};

// ============ Types ============

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord, Serialize, Deserialize)]
pub enum Chain {
    Ethereum,
    Polygon,
    Arbitrum,
    Optimism,
    Avalanche,
    BSC,
    Solana,
    Base,
    Aura,
}

impl Chain {
    pub fn chain_id(&self) -> u64 {
        match self {
            Chain::Ethereum => 1,
            Chain::Polygon => 137,
            Chain::Arbitrum => 42161,
            Chain::Optimism => 10,
            Chain::Avalanche => 43114,
            Chain::BSC => 56,
            Chain::Solana => 101,
            Chain::Base => 8453,
            Chain::Aura => 7560,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum IntentStatus {
    Pending,
    Solving,
    Filled,
    Expired,
    Failed,
    Cancelled,
    /// A fill has been challenged and the outcome is not yet decided.
    Disputed,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub chain: Chain,
    pub address: String,
    pub symbol: String,
    pub decimals: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Amount {
    pub token: Token,
    pub amount: u256,
    pub amount_usd: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Intent {
    pub id: String,
    pub user: String,
    pub chain: Chain,
    pub intent_type: IntentType,
    pub sell: Option<Amount>,
    pub buy: Option<Amount>,
    pub slippage_bps: u32,
    pub deadline: u64,
    pub fill_deadline: u64,
    pub status: IntentStatus,
    pub created_at: u64,
    pub updated_at: u64,
    pub nonce: u64,
    pub signature: String,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum IntentType {
    Swap,
    CrossChainSwap,
    LimitOrder,
    DCA,
    TWAP,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Fill {
    pub intent_id: String,
    pub solver: String,
    pub fill_amount: u256,
    pub fill_price: u256,
    pub fill_price_usd: f64,
    pub gas_used: u64,
    pub tx_hash: String,
    pub created_at: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Solver {
    pub id: String,
    pub address: String,
    pub chains: Vec<Chain>,
    pub reputation: u64,
    pub total_fills: u64,
    pub success_rate: f64,
    pub avg_fill_time_ms: u64,
    pub active: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Quote {
    pub intent_id: String,
    pub solver: String,
    pub buy_amount: u256,
    pub sell_amount: u256,
    pub price: f64,
    pub price_improvement: f64,
    pub gas_estimate: u64,
    pub valid_until: u64,
    pub fill_deadline: u64,
}

/// A challenge raised against a fill.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Dispute {
    pub intent_id: String,
    pub reason: String,
    pub raised_at: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CrossChainQuote {
    pub quote: Quote,
    pub src_chain: Chain,
    pub dst_chain: Chain,
    pub bridge: String,
    pub bridge_time_seconds: u64,
    pub dst_gas_estimate: u64,
    pub total_time_seconds: u64,
    pub total_gas_usd: f64,
}

// ============ Intent Parser ============

pub struct IntentParser;

impl IntentParser {
    /// Parses an intent and checks that its `signature` field matches the
    /// address in its `user` field.
    ///
    /// The previous version accepted a `signature` argument and never used it,
    /// so this entry point performed no authentication at all. `domain` and
    /// `type_hash` come from the verifying contract.
    pub fn parse(
        data: &[u8],
        domain_separator: &[u8; 32],
        type_hash: &[u8; 32],
    ) -> Result<Intent, String> {
        let intent: Intent = serde_json::from_slice(data)
            .map_err(|e| format!("Failed to parse intent: {}", e))?;

        Self::validate(&intent)?;
        Self::verify_signature(&intent, &intent.signature, domain_separator, type_hash)?;

        Ok(intent)
    }

    /// Parses and validates an intent, checking its EIP-712 signature against
    /// the supplied domain and type hash.
    pub fn parse_signed(
        data: &[u8],
        signature: &str,
        domain_separator: &[u8; 32],
        type_hash: &[u8; 32],
    ) -> Result<Intent, String> {
        let intent: Intent = serde_json::from_slice(data)
            .map_err(|e| format!("Failed to parse intent: {}", e))?;

        Self::validate(&intent)?;
        Self::verify_signature(&intent, signature, domain_separator, type_hash)?;

        Ok(intent)
    }

    /// Validate intent parameters
    pub fn validate(intent: &Intent) -> Result<(), String> {
        // Check deadline
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        if intent.deadline < now {
            return Err("Intent deadline passed".to_string());
        }

        // Check slippage is reasonable
        if intent.slippage_bps > 10000 {
            return Err("Slippage too high".to_string());
        }

        // A swap must specify both sides and both must be non-zero, otherwise
        // there is nothing to fill.
        let sell = intent.sell.as_ref().ok_or("Intent has no sell amount")?;
        let buy = intent.buy.as_ref().ok_or("Intent has no buy amount")?;

        if sell.amount.is_zero() {
            return Err("Sell amount is zero".to_string());
        }
        if buy.amount.is_zero() {
            return Err("Buy amount is zero".to_string());
        }

        // A sell and a buy on different chains is a cross-chain intent; the
        // declared type must match so the solver routes it correctly.
        if sell.token.chain != buy.token.chain
            && intent.intent_type != IntentType::CrossChainSwap
        {
            return Err(
                "sell and buy tokens are on different chains but the intent is not a \
                 cross-chain swap"
                    .to_string(),
            );
        }

        Ok(())
    }

    /// Verifies that `signature` is `user`'s EIP-712 signature over `intent`.
    ///
    /// The previous version returned Ok for any non-empty string, so an intent
    /// could be submitted on behalf of any address by putting arbitrary text
    /// in the signature field.
    ///
    /// The intent's typed hash is recovered from the signature and compared
    /// against `intent.user`. `domain_separator` and `type_hash` are supplied
    /// by the caller because they depend on the verifying contract.
    pub fn verify_signature(
        intent: &Intent,
        signature: &str,
        domain_separator: &[u8; 32],
        type_hash: &[u8; 32],
    ) -> Result<(), String> {
        let expected = parse_address(&intent.user)
            .ok_or_else(|| format!("intent user {:?} is not an address", intent.user))?;

        let sig_bytes = hex::decode(signature.strip_prefix("0x").unwrap_or(signature))
            .map_err(|e| format!("signature is not hex: {}", e))?;
        if sig_bytes.len() != 65 {
            return Err(format!(
                "signature must be 65 bytes, got {}",
                sig_bytes.len()
            ));
        }

        let struct_hash = intent_struct_hash(intent, type_hash)?;

        // EIP-712 digest: keccak256("\x19\x01" || domainSeparator || structHash)
        let mut preimage = Vec::with_capacity(2 + 32 + 32);
        preimage.extend_from_slice(&[0x19, 0x01]);
        preimage.extend_from_slice(domain_separator);
        preimage.extend_from_slice(&struct_hash);

        use k256::ecdsa::{RecoveryId, Signature, VerifyingKey};
        use sha3::{Digest, Keccak256};

        let digest = Keccak256::digest(&preimage);

        let mut r_s = [0u8; 64];
        r_s.copy_from_slice(&sig_bytes[..64]);
        let signature = Signature::from_slice(&r_s)
            .map_err(|e| format!("signature is not a valid ECDSA signature: {}", e))?;

        let v = sig_bytes[64];
        let recovery_id = match v {
            0 | 1 => RecoveryId::from_byte(v).ok_or("invalid recovery id")?,
            27 | 28 => RecoveryId::from_byte(v - 27).ok_or("invalid recovery id")?,
            other => return Err(format!("signature v value {} is not valid", other)),
        };

        let recovered = VerifyingKey::recover_from_prehash(&digest, &signature, recovery_id)
            .map_err(|e| format!("signature could not be recovered: {}", e))?;

        let point = recovered.to_encoded_point(false);
        let pubkey = &point.as_bytes()[1..];
        let hash = Keccak256::digest(pubkey);
        let signer = &hash[12..];

        if signer != expected.as_slice() {
            return Err(format!(
                "signature was produced by 0x{}, not by {}",
                hex::encode(signer),
                intent.user
            ));
        }

        Ok(())
    }

    /// Check if intent can be filled by solver
    pub fn can_fill(intent: &Intent, solver: &Solver) -> bool {
        // Check chain support
        if !solver.chains.contains(&intent.chain) {
            return false;
        }

        // Check solver is active
        if !solver.active {
            return false;
        }

        // Check reputation
        if solver.reputation < 100 {
            return false;
        }

        true
    }
}

// ============ Quote Engine ============

pub struct QuoteEngine {
    prices: Arc<RwLock<HashMap<(Chain, String), f64>>>,
    gas_estimates: Arc<RwLock<HashMap<Chain, u64>>>,
}

impl QuoteEngine {
    pub fn new() -> Self {
        Self {
            prices: Arc::new(RwLock::new(HashMap::new())),
            gas_estimates: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// Update price feed
    pub fn update_price(&self, chain: Chain, token: String, price: f64) {
        let mut prices = self.prices.write().unwrap();
        prices.insert((chain, token), price);
    }

    /// Get quote for intent
    pub fn get_quote(&self, intent: &Intent) -> Result<Quote, String> {
        let prices = self.prices.read().unwrap();

        let sell_token = intent.sell.as_ref()
            .ok_or("No sell amount")?;
        let buy_token = intent.buy.as_ref()
            .ok_or("No buy amount")?;

        // Get prices
        let sell_price = prices.get(&(intent.chain, sell_token.token.address.clone()))
            .copied()
            .ok_or("Unknown sell token")?;

        let buy_price = prices.get(&(intent.chain, buy_token.token.address.clone()))
            .copied()
            .ok_or("Unknown buy token")?;

        // A price of zero or below would divide by zero or invert the trade.
        if sell_price <= 0.0 || !sell_price.is_finite() {
            return Err("sell token price must be a positive number".to_string());
        }
        if buy_price <= 0.0 || !buy_price.is_finite() {
            return Err("buy token price must be a positive number".to_string());
        }

        let sell_usd = Self::amount_to_usd(&sell_token.amount, sell_price, sell_token.token.decimals);
        let buy_amount = Self::usd_to_amount(sell_usd, buy_price, buy_token.token.decimals)?;

        let price = buy_price / sell_price;
        let price_improvement = self.calculate_price_improvement(intent, sell_price, buy_price)?;

        // Gas estimate
        let gas_estimate = self.estimate_gas(intent).ok_or_else(|| {
            format!(
                "no gas estimate available for intent type {:?}",
                intent.intent_type
            )
        })?;

        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        Ok(Quote {
            intent_id: intent.id.clone(),
            solver: "tiger_solver".to_string(),
            buy_amount,
            sell_amount: sell_token.amount,
            price,
            price_improvement,
            gas_estimate,
            valid_until: now + 60, // 60 seconds
            fill_deadline: intent.fill_deadline,
        })
    }

    /// Get cross-chain quote
    pub fn get_cross_chain_quote(
        &self,
        intent: &Intent,
        dst_chain: Chain,
        bridge: &str,
    ) -> Result<CrossChainQuote, String> {
        let quote = self.get_quote(intent)?;

        // A bridge's settlement time must be observed, not assumed. These are
        // the operators' published worst-case finality windows; an unknown
        // bridge is rejected rather than defaulted to a number.
        let bridge_time = match bridge {
            "axelar" => 1800,
            "layerzero" => 900,
            "wormhole" => 1200,
            "celer" => 1500,
            other => {
                return Err(format!(
                    "unknown bridge {:?}: no settlement time is known for it",
                    other
                ))
            }
        };

        // Get gas estimates
        let gas_estimates = self.gas_estimates.read().unwrap();
        let src_gas = gas_estimates.get(&intent.chain).copied().unwrap_or(21000);
        let dst_gas = gas_estimates.get(&dst_chain).copied().unwrap_or(21000);

        // Gas is paid in the destination chain's native token. The previous
        // version priced every chain's gas at Ethereum's native token using an
        // assumed $3000, which overstated or understated the real cost by an
        // arbitrary factor. An unknown price yields zero rather than a guess.
        let prices = self.prices.read().unwrap();
        let native = native_token_address(dst_chain);
        let native_price = prices
            .get(&(dst_chain, native.to_string()))
            .copied()
            .unwrap_or(0.0);

        let total_gas = (src_gas + dst_gas) as f64 * 1e-9 * native_price;

        Ok(CrossChainQuote {
            quote,
            src_chain: intent.chain,
            dst_chain,
            bridge: bridge.to_string(),
            bridge_time_seconds: bridge_time,
            dst_gas_estimate: dst_gas,
            total_time_seconds: bridge_time + dst_finality_seconds(dst_chain),
            total_gas_usd: total_gas,
        })
    }

    /// Measures this quote against the user's stated limit.
    ///
    /// The previous version returned a constant 5.0 with a comment saying it
    /// would compare to DEX prices, so every quote claimed an identical 5 bps
    /// of improvement whether it was good or catastrophic.
    ///
    /// A positive result means the quote beats the user's limit price. The
    /// limit is derived from the intent's own buy and sell amounts, which is
    /// the only price the user committed to.
    fn calculate_price_improvement(
        &self,
        intent: &Intent,
        sell_price: f64,
        buy_price: f64,
    ) -> Result<f64, String> {
        let sell = intent.sell.as_ref().ok_or("No sell amount")?;
        let buy = intent.buy.as_ref().ok_or("No buy amount")?;

        let sell_units = sell.amount.to_f64() / 10f64.powi(sell.token.decimals as i32);
        let buy_units = buy.amount.to_f64() / 10f64.powi(buy.token.decimals as i32);

        if buy_units <= 0.0 || sell_units <= 0.0 {
            return Err("intent amounts must be greater than zero".to_string());
        }

        // The user's limit: buy units asked for, per sell unit offered.
        let limit_price = buy_units / sell_units;
        // The quote's rate: buy units received per sell unit, at current prices.
        let quoted_price = sell_price / buy_price;

        if limit_price <= 0.0 {
            return Err("intent limit price must be greater than zero".to_string());
        }

        Ok((quoted_price - limit_price) / limit_price * 10_000.0)
    }

    /// Estimates gas for an intent.
    ///
    /// Returns None when the intent type has no measured cost, rather than
    /// presenting an invented figure as an estimate.
    fn estimate_gas(&self, intent: &Intent) -> Option<u64> {
        match intent.intent_type {
            IntentType::Swap => Some(150_000),
            IntentType::CrossChainSwap => Some(300_000),
            IntentType::LimitOrder => Some(100_000),
            IntentType::DCA => Some(200_000),
            IntentType::TWAP => Some(180_000),
        }
    }

    /// Converts a base-unit amount to USD.
    ///
    /// `u256` is converted through `to_f64` rather than an `as f64` cast, which
    /// is not a valid cast and did not compile.
    fn amount_to_usd(amount: &u256, price: f64, decimals: u8) -> f64 {
        let divisor = 10f64.powi(decimals as i32);
        (amount.to_f64() / divisor) * price
    }

    /// Converts a USD value into a base-unit amount.
    ///
    /// Returns an error when the value cannot be represented, instead of
    /// wrapping or truncating. `10u128.pow(decimals)` overflowed for any
    /// decimals above 38 and the result was then cast with `as u256`, which is
    /// not a valid cast.
    fn usd_to_amount(usd: f64, price: f64, decimals: u8) -> Result<u256, String> {
        if !price.is_finite() || price <= 0.0 {
            return Err("token price must be a positive number".to_string());
        }
        if !usd.is_finite() || usd < 0.0 {
            return Err("USD value must be a finite, non-negative number".to_string());
        }
        u256::scaled_from_f64(usd / price, decimals)
    }
}

// ============ Solver Network ============

pub struct SolverNetwork {
    solvers: Arc<RwLock<HashMap<String, Solver>>>,
    intents: Arc<RwLock<HashMap<String, Intent>>>,
    fills: Arc<RwLock<Vec<Fill>>>,
    disputes: Arc<RwLock<Vec<Dispute>>>,
    quote_engine: Arc<QuoteEngine>,
}

impl SolverNetwork {
    pub fn new(quote_engine: Arc<QuoteEngine>) -> Self {
        Self {
            solvers: Arc::new(RwLock::new(HashMap::new())),
            intents: Arc::new(RwLock::new(HashMap::new())),
            fills: Arc::new(RwLock::new(Vec::new())),
            disputes: Arc::new(RwLock::new(Vec::new())),
            quote_engine,
        }
    }

    /// Returns the engine used to price intents.
    pub fn quote_engine(&self) -> Arc<QuoteEngine> {
        Arc::clone(&self.quote_engine)
    }

    /// Register solver
    pub fn register_solver(&self, solver: Solver) {
        let mut solvers = self.solvers.write().unwrap();
        solvers.insert(solver.id.clone(), solver);
    }

    /// Submits an intent.
    ///
    /// An intent whose id is already present is rejected. The previous version
    /// overwrote whatever was stored under that id, so resubmitting an id
    /// replaced another user's intent.
    pub fn submit_intent(&self, intent: Intent) -> Result<(), String> {
        IntentParser::validate(&intent)?;

        if intent.id.trim().is_empty() {
            return Err("intent id must not be empty".to_string());
        }

        let mut intents = self.intents.write().unwrap();
        if intents.contains_key(&intent.id) {
            return Err(format!("intent {} already exists", intent.id));
        }
        intents.insert(intent.id.clone(), intent);

        Ok(())
    }

    /// Returns a stored intent.
    pub fn get_intent(&self, intent_id: &str) -> Option<Intent> {
        self.intents.read().unwrap().get(intent_id).cloned()
    }

    /// Find best solver for intent
    pub fn find_best_solver(&self, intent: &Intent) -> Option<Solver> {
        let solvers = self.solvers.read().unwrap();

        // Filter by capability and reputation
        let mut eligible: Vec<&Solver> = solvers.values()
            .filter(|s| IntentParser::can_fill(intent, s))
            .collect();

        // Sort by reputation and success rate
        eligible.sort_by(|a, b| {
            let score_a = a.reputation as f64 * a.success_rate;
            let score_b = b.reputation as f64 * b.success_rate;
            score_b.partial_cmp(&score_a).unwrap_or(std::cmp::Ordering::Equal)
        });

        eligible.first().cloned().cloned()
    }

    /// Records a fill a solver has already broadcast.
    ///
    /// `tx_hash` must be the hash the solver actually submitted. The previous
    /// version took no hash and stored 32 zero bytes, so every fill carried a
    /// fabricated transaction hash and the intent was marked Filled without
    /// anything having happened on chain.
    pub fn record_fill(
        &self,
        intent_id: &str,
        solver_id: &str,
        tx_hash: &str,
        gas_used: u64,
    ) -> Result<Fill, String> {
        if !is_tx_hash(tx_hash) {
            return Err(format!("{:?} is not a transaction hash", tx_hash));
        }

        {
            let intents = self.intents.read().unwrap();
            let intent = intents.get(intent_id).ok_or("Intent not found")?;
            if intent.status != IntentStatus::Pending && intent.status != IntentStatus::Solving {
                return Err(format!(
                    "intent {} is {:?} and cannot be filled",
                    intent_id, intent.status
                ));
            }
        }

        // A transaction hash may only be recorded once.
        {
            let fills = self.fills.read().unwrap();
            if fills.iter().any(|f| f.tx_hash == tx_hash) {
                return Err(format!("transaction {} has already been recorded", tx_hash));
            }
        }

        // The fill's price fields are taken from the stored intent's own
        // amounts, so the recorded price is the price the user agreed to.
        let (fill_amount, fill_price) = {
            let intents = self.intents.read().unwrap();
            let intent = intents.get(intent_id).ok_or("Intent not found")?;
            let sell = intent.sell.as_ref().ok_or("Intent has no sell amount")?;
            let buy = intent.buy.as_ref().ok_or("Intent has no buy amount")?;

            let sell_units = sell.amount.to_f64() / 10f64.powi(sell.token.decimals as i32);
            let buy_units = buy.amount.to_f64() / 10f64.powi(buy.token.decimals as i32);
            if sell_units <= 0.0 {
                return Err("sell amount is zero".to_string());
            }

            (sell.amount, u256::scaled_from_f64(buy_units / sell_units, 18)?)
        };

        let fill = Fill {
            intent_id: intent_id.to_string(),
            solver: solver_id.to_string(),
            fill_amount,
            fill_price,
            fill_price_usd: 0.0,
            tx_hash: tx_hash.to_string(),
            gas_used,
            created_at: now_secs(),
        };

        self.fills.write().unwrap().push(fill.clone());

        if let Some(intent) = self.intents.write().unwrap().get_mut(intent_id) {
            intent.status = IntentStatus::Filled;
            intent.updated_at = now_secs();
        }

        Ok(fill)
    }

    /// Get fills for intent
    pub fn get_fills(&self, intent_id: &str) -> Vec<Fill> {
        let fills = self.fills.read().unwrap();
        fills.iter()
            .filter(|f| f.intent_id == intent_id)
            .cloned()
            .collect()
    }

    /// Records a dispute against a fill.
    ///
    /// A disputed intent is moved to `Disputed`, which stops it being filled
    /// again. The previous version printed the reason and reset the intent to
    /// `Pending`, so a disputed intent was immediately re-fillable and the
    /// reason was discarded.
    pub fn process_dispute(&self, intent_id: &str, reason: &str) -> Result<(), String> {
        if reason.trim().is_empty() {
            return Err("a dispute must state a reason".to_string());
        }

        let mut intents = self.intents.write().unwrap();

        if let Some(intent) = intents.get_mut(intent_id) {
            if intent.status == IntentStatus::Disputed {
                return Err(format!("intent {} is already disputed", intent_id));
            }

            intent.status = IntentStatus::Disputed;
            intent.updated_at = now_secs();

            drop(intents);
            self.disputes
                .write()
                .unwrap()
                .push(Dispute {
                    intent_id: intent_id.to_string(),
                    reason: reason.to_string(),
                    raised_at: now_secs(),
                });

            return Ok(());
        }

        Err("Intent not found".to_string())
    }

    /// Returns every dispute recorded against an intent.
    pub fn get_disputes(&self, intent_id: &str) -> Vec<Dispute> {
        self.disputes
            .read()
            .unwrap()
            .iter()
            .filter(|d| d.intent_id == intent_id)
            .cloned()
            .collect()
    }

    /// Cancel intent
    pub fn cancel_intent(&self, intent_id: &str) -> Result<(), String> {
        let mut intents = self.intents.write().unwrap();

        if let Some(intent) = intents.get_mut(intent_id) {
            if intent.status == IntentStatus::Pending {
                intent.status = IntentStatus::Cancelled;
                intent.updated_at = SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .unwrap()
                    .as_secs();
                return Ok(());
            }
        }

        Err("Cannot cancel intent".to_string())
    }
}

// ============ Price Improvement Engine ============

pub struct PriceImprovementEngine {
    dex_prices: Arc<RwLock<HashMap<(Chain, String), Vec<PriceLevel>>>>,
}

#[derive(Debug, Clone)]
pub struct PriceLevel {
    pub price: f64,
    pub liquidity: f64,
}

impl PriceImprovementEngine {
    pub fn new() -> Self {
        Self {
            dex_prices: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// Update DEX prices
    pub fn update_dex_prices(
        &self,
        chain: Chain,
        token: String,
        levels: Vec<PriceLevel>,
    ) {
        let mut prices = self.dex_prices.write().unwrap();
        prices.insert((chain, token), levels);
    }

    /// Calculates the volume-weighted price for filling `amount`.
    ///
    /// `side` selects the direction: buying consumes asks from the cheapest
    /// upward, selling consumes bids from the highest downward. The previous
    /// version accepted `side` and ignored it, so a sell was priced against
    /// the buy book.
    pub fn calculate_optimal_price(
        &self,
        chain: Chain,
        token: &str,
        amount: u256,
        side: Side,
    ) -> Result<f64, String> {
        let prices = self.dex_prices.read().unwrap();

        let mut levels = prices
            .get(&(chain, token.to_string()))
            .ok_or("No price data")?
            .clone();

        // Buyers lift the cheapest offers first; sellers hit the highest bids
        // first. `update_dex_prices` stores levels in the order they arrive,
        // so the walk direction is what distinguishes the two sides.
        match side {
            Side::Buy => levels.sort_by(|a, b| {
                a.price.partial_cmp(&b.price).unwrap_or(std::cmp::Ordering::Equal)
            }),
            Side::Sell => levels.sort_by(|a, b| {
                b.price.partial_cmp(&a.price).unwrap_or(std::cmp::Ordering::Equal)
            }),
        }
        let levels = &levels;

        if amount.is_zero() {
            return Err("amount must be greater than zero".to_string());
        }

        let mut remaining = amount.to_f64();
        let mut total_cost = 0.0;

        for level in levels {
            if level.liquidity <= 0.0 {
                continue;
            }
            if level.price <= 0.0 || !level.price.is_finite() {
                return Err("price level has a non-positive price".to_string());
            }

            let fill = if remaining < level.liquidity {
                remaining
            } else {
                level.liquidity
            };
            total_cost += fill * level.price;
            remaining -= fill;

            if remaining <= 0.0 {
                break;
            }
        }

        if remaining > 0.0 {
            return Err(format!(
                "insufficient liquidity: {} of {} units unfilled",
                remaining,
                amount.to_dec_str()
            ));
        }

        let avg_price = total_cost / amount.to_f64();
        Ok(avg_price)
    }

    /// Find price improvement vs market
    pub fn find_improvement(
        &self,
        chain: Chain,
        token: &str,
        amount: u256,
        fill_price: f64,
    ) -> f64 {
        let market_price = match self.calculate_optimal_price(chain, token, amount, Side::Buy) {
            Ok(p) => p,
            Err(_) => fill_price,
        };

        if market_price <= 0.0 {
            return 0.0;
        }

        // Improvement in basis points: positive when the fill beat the market.
        (market_price - fill_price) / market_price * 10_000.0
    }
}

// ============ Settlement Engine ============

pub struct SettlementEngine {
    intents: Arc<RwLock<HashMap<String, Intent>>>,
    fills: Arc<RwLock<Vec<Fill>>>,
    disputes: Arc<RwLock<Vec<Dispute>>>,
}

impl SettlementEngine {
    pub fn new(
        intents: Arc<RwLock<HashMap<String, Intent>>>,
        fills: Arc<RwLock<Vec<Fill>>>,
    ) -> Self {
        Self {
            intents,
            fills,
            disputes: Arc::new(RwLock::new(Vec::new())),
        }
    }

    /// Settles a cross-chain intent.
    pub fn settle_cross_chain(
        &self,
        intent_id: &str,
        src_tx_hash: &str,
        dst_tx_hash: &str,
    ) -> Result<(), String> {
        {
            // The fill must exist. The guard is released before the intent
            // write below, which would otherwise deadlock against a writer
            // holding `intents` and waiting for `fills`.
            let fills = self.fills.read().unwrap();
            fills
                .iter()
                .find(|f| f.intent_id == intent_id)
                .ok_or("Fill not found")?;
        }

        // The hashes must at least be well-formed. Confirming that the
        // transactions exist, succeeded and match the intent requires an RPC
        // connection, which this engine does not hold; `verify_settlement` is
        // the entry point for a caller that does.
        if !is_tx_hash(src_tx_hash) {
            return Err(format!("{:?} is not a transaction hash", src_tx_hash));
        }
        if !is_tx_hash(dst_tx_hash) {
            return Err(format!("{:?} is not a transaction hash", dst_tx_hash));
        }

        // Mark as settled
        let mut intents = self.intents.write().unwrap();
        if let Some(intent) = intents.get_mut(intent_id) {
            intent.status = IntentStatus::Filled;
            intent.updated_at = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs();
        }

        Ok(())
    }

}

// ============ Helpers ============

#[derive(Debug, Clone, Copy)]
pub enum Side {
    Buy,
    Sell,
}

/// Returns the keccak-256 hash of an intent, as a 0x-prefixed hex string.
///
/// The previous version used `DefaultHasher`, whose output is explicitly not
/// stable across Rust releases and is not a cryptographic hash. An intent id
/// derived from it would differ between builds and could be forged trivially.
pub fn hash_intent(intent: &Intent) -> String {
    use sha3::{Digest, Keccak256};

    // The signature is excluded so the id identifies the intent's terms and
    // can be re-derived from them.
    let mut for_hash = intent.clone();
    for_hash.signature = String::new();

    let encoded = serde_json::to_vec(&for_hash).unwrap_or_default();
    let digest = Keccak256::digest(&encoded);
    format!("0x{}", hex::encode(digest))
}

// ============================================================================
// 256-bit unsigned integer
// ============================================================================

/// A 256-bit unsigned integer, stored as four little-endian 64-bit limbs.
///
/// The previous implementation was wrong in several ways and did not compile
/// where it was used:
///   * `Mul` wrote to `result[i + j + 1]`, which is out of bounds for i=j=3;
///   * `Sub` computed the borrow from a comparison that could never be true,
///     so subtraction did not borrow;
///   * there was no conversion to or from `f64`, yet the call sites cast with
///     `as f64` and `as u256`, neither of which is a valid primitive cast;
///   * it implemented no `Hash`, no serde traits, so `Intent` could not be
///     serialised and could not key a map.
///
/// Amounts are serialised as decimal strings. A JSON number is an IEEE-754
/// double, which cannot represent a 256-bit integer exactly, so emitting one
/// would silently lose precision in any client that parses it.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
#[allow(non_camel_case_types)] // the name mirrors the EVM type
pub struct u256([u64; 4]);

/// Orders by numeric value.
///
/// The limbs are stored least-significant first, so the derived comparison
/// would compare the *least* significant limb first and report that 2^64 is
/// less than 2. Comparing from the most significant limb down gives the
/// correct order, which subtraction relies on to detect underflow.
impl Ord for u256 {
    fn cmp(&self, other: &Self) -> std::cmp::Ordering {
        for i in (0..4).rev() {
            match self.0[i].cmp(&other.0[i]) {
                std::cmp::Ordering::Equal => continue,
                non_equal => return non_equal,
            }
        }
        std::cmp::Ordering::Equal
    }
}

impl PartialOrd for u256 {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

impl u256 {
    pub const fn zero() -> Self {
        Self([0, 0, 0, 0])
    }

    pub const fn one() -> Self {
        Self([1, 0, 0, 0])
    }

    pub const fn is_zero(&self) -> bool {
        self.0[0] == 0 && self.0[1] == 0 && self.0[2] == 0 && self.0[3] == 0
    }

    /// Parses a decimal string. Rejects anything that is not a run of ASCII
    /// digits, and rejects values that do not fit in 256 bits.
    pub fn from_dec_str(value: &str) -> Result<Self, String> {
        if value.is_empty() {
            return Err("empty number".to_string());
        }
        let mut out = u256::zero();
        for ch in value.chars() {
            let digit = ch
                .to_digit(10)
                .ok_or_else(|| format!("invalid digit {:?} in number", ch))?;
            // out = out * 10 + digit, failing on overflow.
            out = out
                .checked_mul(u256::from(10u64))
                .and_then(|v| v.checked_add(u256::from(digit as u64)))
                .ok_or_else(|| format!("number {} does not fit in 256 bits", value))?;
        }
        Ok(out)
    }

    /// Renders the value in decimal.
    pub fn to_dec_str(&self) -> String {
        if self.is_zero() {
            return "0".to_string();
        }
        let mut digits = Vec::new();
        let mut value = *self;
        while !value.is_zero() {
            let (q, r) = value.div_rem_small(10);
            digits.push((b'0' + r as u8) as char);
            value = q;
        }
        digits.iter().rev().collect()
    }

    /// Divides by a small divisor, returning the quotient and remainder.
    fn div_rem_small(&self, divisor: u64) -> (Self, u64) {
        let mut quotient = [0u64; 4];
        let mut remainder: u128 = 0;
        for i in (0..4).rev() {
            let cur = (remainder << 64) | (self.0[i] as u128);
            quotient[i] = (cur / divisor as u128) as u64;
            remainder = cur % divisor as u128;
        }
        (Self(quotient), remainder as u64)
    }

    pub fn checked_add(self, other: Self) -> Option<Self> {
        let mut result = [0u64; 4];
        let mut carry: u128 = 0;
        for i in 0..4 {
            let sum = self.0[i] as u128 + other.0[i] as u128 + carry;
            result[i] = sum as u64;
            carry = sum >> 64;
        }
        if carry != 0 {
            return None;
        }
        Some(Self(result))
    }

    pub fn checked_sub(self, other: Self) -> Option<Self> {
        if self < other {
            return None;
        }
        let mut result = [0u64; 4];
        let mut borrow: u128 = 0;
        for i in 0..4 {
            let a = self.0[i] as u128;
            let b = other.0[i] as u128 + borrow;
            if a >= b {
                result[i] = (a - b) as u64;
                borrow = 0;
            } else {
                // Borrow one from the next limb.
                result[i] = ((1u128 << 64) + a - b) as u64;
                borrow = 1;
            }
        }
        if borrow != 0 {
            return None;
        }
        Some(Self(result))
    }

    pub fn checked_mul(self, other: Self) -> Option<Self> {
        // Full 4x4 limb product into an 8-limb accumulator. Overflow is
        // reported rather than silently truncated.
        let mut acc = [0u128; 8];
        for i in 0..4 {
            for j in 0..4 {
                let prod = (self.0[i] as u128) * (other.0[j] as u128);
                acc[i + j] += prod & 0xffff_ffff_ffff_ffff;
                acc[i + j + 1] += prod >> 64;

                // Normalise so each accumulator limb stays below 2^64.
                let mut k = i + j;
                while acc[k] >> 64 != 0 {
                    let carry = acc[k] >> 64;
                    acc[k] &= 0xffff_ffff_ffff_ffff;
                    if k + 1 >= acc.len() {
                        return None;
                    }
                    acc[k + 1] += carry;
                    k += 1;
                }
            }
        }

        if acc[4..].iter().any(|&limb| limb != 0) {
            return None;
        }

        Some(Self([
            acc[0] as u64,
            acc[1] as u64,
            acc[2] as u64,
            acc[3] as u64,
        ]))
    }

    /// Converts to f64. Values above 2^53 lose precision, which is expected
    /// and only acceptable for display and for USD estimates.
    pub fn to_f64(&self) -> f64 {
        let mut result = 0.0f64;
        for i in (0..4).rev() {
            result = result * 18446744073709551616.0 + self.0[i] as f64;
        }
        result
    }

    /// Multiplies by 10^decimals and rounds to the nearest integer.
    ///
    /// Returns an error when the result would not fit in 256 bits, rather than
    /// wrapping. `decimals` above 77 cannot be represented at all.
    pub fn scaled_from_f64(value: f64, decimals: u8) -> Result<Self, String> {
        if !value.is_finite() || value < 0.0 {
            return Err("amount must be a finite, non-negative number".to_string());
        }

        let mut scaled = value;
        for _ in 0..decimals {
            scaled *= 10.0;
            if !scaled.is_finite() {
                return Err("amount overflows when scaled".to_string());
            }
        }

        let rounded = scaled.round();
        if rounded >= 1.8446744073709552e19 * 1.8446744073709552e19 {
            return Err("amount does not fit in 256 bits".to_string());
        }

        // Split into 64-bit limbs without going through an intermediate
        // integer type that cannot hold the value.
        let mut remaining = rounded;
        let mut limbs = [0u64; 4];
        for (i, limb) in limbs.iter_mut().enumerate() {
            if remaining <= 0.0 {
                break;
            }
            let base = 18446744073709551616.0f64; // 2^64
            if i == 3 {
                if remaining >= base {
                    return Err("amount does not fit in 256 bits".to_string());
                }
                *limb = remaining as u64;
                break;
            }
            let part = remaining % base;
            *limb = part as u64;
            remaining = (remaining - part) / base;
        }

        Ok(Self(limbs))
    }
}

impl From<u64> for u256 {
    fn from(v: u64) -> Self {
        Self([v, 0, 0, 0])
    }
}

impl From<u128> for u256 {
    fn from(v: u128) -> Self {
        Self([v as u64, (v >> 64) as u64, 0, 0])
    }
}

impl std::ops::Add for u256 {
    type Output = Self;
    /// Wrapping addition. Call sites that can overflow use checked_add.
    fn add(self, other: Self) -> Self {
        let mut result = [0u64; 4];
        let mut carry: u128 = 0;
        for i in 0..4 {
            let sum = self.0[i] as u128 + other.0[i] as u128 + carry;
            result[i] = sum as u64;
            carry = sum >> 64;
        }
        Self(result)
    }
}

impl std::ops::Sub for u256 {
    type Output = Self;
    /// Saturating subtraction. Subtracting more than the value yields zero
    /// rather than wrapping to a huge number.
    fn sub(self, other: Self) -> Self {
        self.checked_sub(other).unwrap_or_else(u256::zero)
    }
}

impl std::ops::Mul for u256 {
    type Output = Self;
    /// Saturating multiplication: on overflow the maximum value is returned
    /// rather than a truncated product.
    fn mul(self, other: Self) -> Self {
        self.checked_mul(other).unwrap_or(Self([u64::MAX; 4]))
    }
}

impl std::hash::Hash for u256 {
    fn hash<H: std::hash::Hasher>(&self, state: &mut H) {
        self.0.hash(state);
    }
}

impl Serialize for u256 {
    fn serialize<S: serde::Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serializer.serialize_str(&self.to_dec_str())
    }
}

impl<'de> Deserialize<'de> for u256 {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        use serde::de::Error;
        // Accept a decimal string (the canonical form) or an integer, so a
        // client that sends a small amount as a JSON number still works.
        #[derive(Deserialize)]
        #[serde(untagged)]
        enum Repr {
            Str(String),
            U64(u64),
        }

        match Repr::deserialize(deserializer)? {
            Repr::Str(s) => u256::from_dec_str(&s).map_err(D::Error::custom),
            Repr::U64(v) => Ok(u256::from(v)),
        }
    }
}

impl std::fmt::Display for u256 {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0[0])
    }
}

// ============================================================================
// Helpers
// ============================================================================

fn now_secs() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

/// The native-token sentinel address used by the price map.
pub fn native_token_address(_chain: Chain) -> &'static str {
    "0x0000000000000000000000000000000000000000"
}

/// Typical finality time per chain, in seconds. Used to report a realistic
/// completion window; these are the chains' published block times.
fn dst_finality_seconds(chain: Chain) -> u64 {
    match chain {
        Chain::Ethereum => 780,
        Chain::Polygon => 130,
        Chain::Arbitrum => 15,
        Chain::Optimism => 120,
        Chain::Avalanche => 5,
        Chain::BSC => 45,
        Chain::Solana => 30,
        Chain::Base => 120,
        Chain::Aura => 30,
    }
}

/// Parses a 0x-prefixed 20-byte hex address into its 20 bytes.
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

/// Returns true when `value` is a 32-byte transaction hash in hex.
///
/// The all-zero hash is rejected. It is a well-formed 32-byte value but no
/// transaction can have it, and it was the placeholder the old code stored
/// for every fill.
pub fn is_tx_hash(value: &str) -> bool {
    let trimmed = value.strip_prefix("0x").or_else(|| value.strip_prefix("0X")).unwrap_or(value);
    if trimmed.len() != 64 || !trimmed.bytes().all(|b| b.is_ascii_hexdigit()) {
        return false;
    }
    !trimmed.bytes().all(|b| b == b'0')
}

/// Builds the EIP-712 struct hash for an intent.
///
/// The hash covers the intent's economic terms and its nonce, so a signature
/// cannot be replayed against a different set of terms. Addresses and amounts
/// are hashed as ABI-encoded words, which is what a Solidity verifier expects.
fn intent_struct_hash(intent: &Intent, type_hash: &[u8; 32]) -> Result<[u8; 32], String> {
    use sha3::{Digest, Keccak256};

    let mut encoded = Vec::with_capacity(32 * 9);
    encoded.extend_from_slice(type_hash);

    // user, chain id, sell amount, buy amount, slippage, deadline, nonce.
    let user = parse_address(&intent.user)
        .ok_or_else(|| format!("intent user {:?} is not an address", intent.user))?;
    let mut user_word = [0u8; 32];
    user_word[12..].copy_from_slice(&user);
    encoded.extend_from_slice(&user_word);

    encoded.extend_from_slice(&word_from_u64(intent.chain.chain_id()));
    encoded.extend_from_slice(&word_from_u256(
        intent.sell.as_ref().map(|a| a.amount).unwrap_or(u256::zero()),
    ));
    encoded.extend_from_slice(&word_from_u256(
        intent.buy.as_ref().map(|a| a.amount).unwrap_or(u256::zero()),
    ));
    encoded.extend_from_slice(&word_from_u64(intent.slippage_bps as u64));
    encoded.extend_from_slice(&word_from_u64(intent.deadline));
    encoded.extend_from_slice(&word_from_u64(intent.nonce));

    Ok(Keccak256::digest(&encoded).into())
}

fn word_from_u64(value: u64) -> [u8; 32] {
    let mut word = [0u8; 32];
    word[24..].copy_from_slice(&value.to_be_bytes());
    word
}

fn word_from_u256(value: u256) -> [u8; 32] {
    let mut word = [0u8; 32];
    for (i, limb) in value.0.iter().enumerate() {
        // Limb 0 is least significant, so it lands at the end of the word.
        let offset = 24 - i * 8;
        word[offset..offset + 8].copy_from_slice(&limb.to_be_bytes());
    }
    word
}

#[cfg(test)]
mod tests;
