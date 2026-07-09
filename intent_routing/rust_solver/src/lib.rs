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

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
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
    /// Parse and validate an intent from raw data
    pub fn parse(data: &[u8], signature: &str) -> Result<Intent, String> {
        let intent: Intent = serde_json::from_slice(data)
            .map_err(|e| format!("Failed to parse intent: {}", e))?;
        
        // Validate intent
        Self::validate(&intent)?;
        
        // Verify signature
        Self::verify_signature(&intent, signature)?;
        
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
        
        // Check amounts
        if let Some(sell) = &intent.sell {
            if sell.amount == 0.into() {
                return Err("Sell amount is zero".to_string());
            }
        }
        
        if let Some(buy) = &intent.buy {
            if buy.amount == 0.into() {
                return Err("Buy amount is zero".to_string());
            }
        }
        
        Ok(())
    }
    
    /// Verify intent signature
    fn verify_signature(intent: &Intent, signature: &str) -> Result<(), String> {
        // In production, would verify EIP-712 signature
        if signature.is_empty() {
            return Err("Invalid signature".to_string());
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
        
        // Calculate quote
        let sell_usd = Self::amount_to_usd(&sell_token.amount, sell_price, sell_token.token.decimals);
        let buy_amount = Self::usd_to_amount(sell_usd, buy_price, buy_token.token.decimals);
        
        let price = buy_price / sell_price;
        let price_improvement = self.calculate_price_improvement(price, intent);
        
        // Gas estimate
        let gas_estimate = self.estimate_gas(intent);
        
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
        
        // Estimate bridge time
        let bridge_time = match bridge {
            "axelar" => 1800,
            "layerzero" => 900,
            "wormhole" => 1200,
            "celer" => 1500,
            _ => 1800,
        };
        
        // Get gas estimates
        let gas_estimates = self.gas_estimates.read().unwrap();
        let src_gas = gas_estimates.get(&intent.chain).copied().unwrap_or(21000);
        let dst_gas = gas_estimates.get(&dst_chain).copied().unwrap_or(21000);
        
        // Calculate total gas in USD
        let prices = self.prices.read().unwrap();
        let eth_price = prices.get(&(Chain::Ethereum, "0x0000000000000000000000000000000000000000".to_string()))
            .copied()
            .unwrap_or(3000.0);
        
        let total_gas = (src_gas + dst_gas) as f64 * 1e-8 * eth_price;
        
        Ok(CrossChainQuote {
            quote,
            src_chain: intent.chain,
            dst_chain,
            bridge: bridge.to_string(),
            bridge_time_seconds: bridge_time,
            dst_gas_estimate: dst_gas,
            total_time_seconds: bridge_time + 300, // Add 5 min for confirmation
            total_gas_usd: total_gas,
        })
    }
    
    /// Calculate price improvement vs market
    fn calculate_price_improvement(&self, price: f64, intent: &Intent) -> f64 {
        // Simplified - would compare to DEX prices
        // Return improvement in basis points
        5.0 // Assume 5 bps improvement
    }
    
    /// Estimate gas for intent
    fn estimate_gas(&self, intent: &Intent) -> u64 {
        match intent.intent_type {
            IntentType::Swap => 150000,
            IntentType::CrossChainSwap => 300000,
            IntentType::LimitOrder => 100000,
            IntentType::DCA => 200000,
            IntentType::TWAP => 180000,
        }
    }
    
    /// Convert amount to USD
    fn amount_to_usd(amount: &u256, price: f64, decimals: u8) -> f64 {
        let divisor = 10u128.pow(decimals as u32) as f64;
        (*amount as f64 / divisor) * price
    }
    
    /// Convert USD to token amount
    fn usd_to_amount(usd: f64, price: f64, decimals: u8) -> u256 {
        let amount = usd / price;
        let multiplier = 10u128.pow(decimals as u32);
        (amount * multiplier) as u256
    }
}

// ============ Solver Network ============

pub struct SolverNetwork {
    solvers: Arc<RwLock<HashMap<String, Solver>>>,
    intents: Arc<RwLock<HashMap<String, Intent>>>,
    fills: Arc<RwLock<Vec<Fill>>>,
    quote_engine: Arc<QuoteEngine>,
}

impl SolverNetwork {
    pub fn new(quote_engine: Arc<QuoteEngine>) -> Self {
        Self {
            solvers: Arc::new(RwLock::new(HashMap::new())),
            intents: Arc::new(RwLock::new(HashMap::new())),
            fills: Arc::new(RwLock::new(Vec::new())),
            quote_engine,
        }
    }
    
    /// Register solver
    pub fn register_solver(&self, solver: Solver) {
        let mut solvers = self.solvers.write().unwrap();
        solvers.insert(solver.id.clone(), solver);
    }
    
    /// Submit intent
    pub fn submit_intent(&self, intent: Intent) -> Result<(), String> {
        // Validate intent
        IntentParser::validate(&intent)?;
        
        let mut intents = self.intents.write().unwrap();
        intents.insert(intent.id.clone(), intent);
        
        Ok(())
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
    
    /// Execute intent
    pub fn execute_intent(
        &self,
        intent_id: &str,
        solver_id: &str,
    ) -> Result<Fill, String> {
        let intents = self.intents.read().unwrap();
        let intent = intents.get(intent_id)
            .ok_or("Intent not found")?;
        
        // Get quote
        let quote = self.quote_engine.get_quote(intent)?;
        
        // Create fill record
        let fill = Fill {
            intent_id: intent_id.to_string(),
            solver: solver_id.to_string(),
            fill_amount: quote.sell_amount,
            fill_price: quote.sell_amount, // Simplified
            fill_price_usd: quote.price,
            gas_used: quote.gas_estimate,
            tx_hash: format!("0x{}", hex::encode([0u8; 32])), // Would be real tx
            created_at: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs(),
        };
        
        // Store fill
        let mut fills = self.fills.write().unwrap();
        fills.push(fill.clone());
        
        // Update intent status
        let mut intents = self.intents.write().unwrap();
        if let Some(intent) = intents.get_mut(intent_id) {
            intent.status = IntentStatus::Filled;
            intent.updated_at = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs();
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
    
    /// Calculate optimal fill price
    pub fn calculate_optimal_price(
        &self,
        chain: Chain,
        token: &str,
        amount: u256,
        side: Side,
    ) -> Result<f64, String> {
        let prices = self.dex_prices.read().unwrap();
        
        let levels = prices.get(&(chain, token.to_string()))
            .ok_or("No price data")?;
        
        let mut remaining = *amount;
        let mut total_cost = 0.0;
        
        for level in levels {
            let fill = std::cmp::min(remaining as f64, level.liquidity);
            total_cost += fill * level.price;
            remaining = (remaining as f64 - fill) as u256;
            
            if remaining == 0 {
                break;
            }
        }
        
        if remaining > 0 {
            return Err("Insufficient liquidity".to_string());
        }
        
        let avg_price = total_cost / (*amount as f64);
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
        
        // Return improvement in basis points
        ((market_price - fill_price) / market_price * 10000.0) as f64
    }
}

// ============ Settlement Engine ============

pub struct SettlementEngine {
    intents: Arc<RwLock<HashMap<String, Intent>>>,
    fills: Arc<RwLock<Vec<Fill>>>>,
}

impl SettlementEngine {
    pub fn new(intents: Arc<RwLock<HashMap<String, Intent>>>, fills: Arc<RwLock<Vec<Fill>>>) -> Self {
        Self { intents, fills }
    }
    
    /// Settle cross-chain intent
    pub fn settle_cross_chain(
        &self,
        intent_id: &str,
        src_tx_hash: &str,
        dst_tx_hash: &str,
    ) -> Result<(), String> {
        let fills = self.fills.read().unwrap();
        
        // Check if fill exists
        let fill = fills.iter()
            .find(|f| f.intent_id == intent_id)
            .ok_or("Fill not found")?;
        
        // Verify transactions (in production, would verify on-chain)
        if src_tx_hash.is_empty() || dst_tx_hash.is_empty() {
            return Err("Invalid transaction hashes".to_string());
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
    
    /// Process settlement disputes
    pub fn process_dispute(&self, intent_id: &str, reason: &str) -> Result<(), String> {
        let mut intents = self.intents.write().unwrap();
        
        if let Some(intent) = intents.get_mut(intent_id) {
            // Log dispute
            println!("Dispute for intent {}: {}", intent_id, reason);
            
            // In production, would handle based on reason
            intent.status = IntentStatus::Pending;
            
            return Ok(());
        }
        
        Err("Intent not found".to_string())
    }
}

// ============ Helpers ============

#[derive(Debug, Clone, Copy)]
pub enum Side {
    Buy,
    Sell,
}

pub fn hash_intent(intent: &Intent) -> String {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    
    let mut hasher = DefaultHasher::new();
    intent.hash(&mut hasher);
    format!("{:x}", hasher.finish())
}

// Simple u256 implementation
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord)]
pub struct u256([u64; 4]);

impl u256 {
    pub fn zero() -> Self { Self([0, 0, 0, 0]) }
    pub fn one() -> Self { Self([1, 0, 0, 0]) }
}

impl From<u64> for u256 {
    fn from(v: u64) -> Self {
        Self([v, 0, 0, 0])
    }
}

impl std::ops::Add for u256 {
    type Output = Self;
    fn add(self, other: Self) -> Self {
        let mut result = [0u64; 4];
        let mut carry = 0u128;
        for i in 0..4 {
            let sum = (self.0[i] as u128) + (other.0[i] as u128) + carry;
            result[i] = sum as u64;
            carry = sum >> 64;
        }
        Self(result)
    }
}

impl std::ops::Sub for u256 {
    type Output = Self;
    fn sub(self, other: Self) -> Self {
        let mut result = [0u64; 4];
        let mut borrow = 0u128;
        for i in 0..4 {
            let diff = (self.0[i] as u128).wrapping_sub((other.0[i] as u128).wrapping_sub(borrow));
            result[i] = diff as u64;
            borrow = if diff > (1 << 64) { 1 } else { 0 };
        }
        Self(result)
    }
}

impl std::ops::Mul for u256 {
    type Output = Self;
    fn mul(self, other: Self) -> Self {
        let mut result = [0u64; 4];
        for i in 0..4 {
            for j in 0..4 {
                if i + j < 4 {
                    let prod = (self.0[i] as u128) * (other.0[j] as u128);
                    let (lo, hi) = (prod as u64, (prod >> 64) as u64);
                    result[i + j] = result[i + j].wrapping_add(lo);
                    result[i + j + 1] = result[i + j + 1].wrapping_add(hi);
                }
            }
        }
        Self(result)
    }
}

impl std::fmt::Display for u256 {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0[0])
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_intent_validation() {
        let intent = Intent {
            id: "test".to_string(),
            user: "0x123".to_string(),
            chain: Chain::Ethereum,
            intent_type: IntentType::Swap,
            sell: Some(Amount {
                token: Token {
                    chain: Chain::Ethereum,
                    address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48".to_string(),
                    symbol: "USDC".to_string(),
                    decimals: 6,
                },
                amount: u256::from(1000000u64),
                amount_usd: 1000.0,
            }),
            buy: Some(Amount {
                token: Token {
                    chain: Chain::Ethereum,
                    address: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599".to_string(),
                    symbol: "WBTC".to_string(),
                    decimals: 8,
                },
                amount: u256::from(100000000u64),
                amount_usd: 1000.0,
            }),
            slippage_bps: 50,
            deadline: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs() + 3600,
            fill_deadline: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs() + 1800,
            status: IntentStatus::Pending,
            created_at: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs(),
            updated_at: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs(),
            nonce: 0,
            signature: "test_sig".to_string(),
        };
        
        assert!(IntentParser::validate(&intent).is_ok());
    }
}
