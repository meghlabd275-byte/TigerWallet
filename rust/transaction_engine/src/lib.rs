//! TigerWallet Transaction Engine - High Performance Transaction Processing
//!
//! This module provides ultra-low latency transaction processing for:
//! - Transaction signing
//! - Gas optimization
//! - Transaction simulation
//! - Nonce management
//! - Multi-chain transaction coordination

use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};

// ============================================================================
// Error Types
// ============================================================================

#[derive(Debug, thiserror::Error)]
pub enum TransactionError {
    #[error("Insufficient funds: {0}")]
    InsufficientFunds(String),

    #[error("Nonce too low: expected {0}, got {1}")]
    NonceTooLow(u64, u64),

    #[error("Gas price too low: minimum {0}")]
    GasPriceTooLow(u64),

    #[error("Transaction reverted: {0}")]
    Reverted(String),

    #[error("Chain not supported: {0}")]
    UnsupportedChain(i64),

    #[error("Signing error: {0}")]
    SigningError(String),

    #[error("Network error: {0}")]
    NetworkError(String),

    #[error("No base fee has been supplied for chain {0}")]
    MissingBaseFee(u64),
}

pub type Result<T> = std::result::Result<T, TransactionError>;

// ============================================================================
// Transaction Types
// ============================================================================

/// EVM Transaction
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct EvmTransaction {
    pub chain_id: u64,
    pub nonce: u64,
    pub to: String,
    pub value: String,
    pub data: String,
    pub gas_limit: u64,
    pub gas_price: String,
    pub max_priority_fee_per_gas: Option<String>,
    pub max_fee_per_gas: Option<String>,
}

/// Transaction request
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct TransactionRequest {
    pub from: String,
    pub to: String,
    pub value: Option<String>,
    pub data: Option<String>,
    pub token: Option<String>,
    pub chain_id: i64,
    pub gas_limit: Option<u64>,
    pub gas_price: Option<String>,
}

/// Transaction receipt
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct TransactionReceipt {
    pub transaction_hash: String,
    pub block_number: u64,
    pub block_hash: String,
    pub status: bool,
    pub gas_used: u64,
    pub cumulative_gas_used: u64,
    pub logs: Vec<Log>,
    pub logs_bloom: String,
}

/// Log entry
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct Log {
    pub address: String,
    pub topics: Vec<String>,
    pub data: String,
    pub log_index: u64,
    pub transaction_index: u64,
}

/// Gas estimation
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct GasEstimate {
    pub gas_limit: u64,
    pub gas_price: String,
    pub max_fee_per_gas: String,
    pub max_priority_fee_per_gas: String,
    pub total_cost: String,
}

/// Transaction status
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct TransactionStatus {
    pub hash: String,
    pub status: TxStatus,
    pub block_number: Option<u64>,
    pub confirmed_at: Option<Duration>,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize, PartialEq)]
pub enum TxStatus {
    Pending,
    Confirmed,
    Failed,
    Replaced,
}

// ============================================================================
// Transaction Pool
// ============================================================================

/// Manages pending transactions
pub struct TransactionPool {
    pending: HashMap<String, PendingTx>,
    nonce_manager: HashMap<String, NonceManager>,
}

struct PendingTx {
    tx: EvmTransaction,
    submitted_at: Instant,
    replaced_by: Option<String>,
}

struct NonceManager {
    current_nonce: u64,
    pending_count: u64,
}

impl TransactionPool {
    pub fn new() -> Self {
        Self {
            pending: HashMap::new(),
            nonce_manager: HashMap::new(),
        }
    }

    /// Add transaction to pool
    pub fn add(&mut self, address: &str, tx: EvmTransaction) -> Result<()> {
        let key = format!("{}:{}", address.to_lowercase(), tx.nonce);

        // Check nonce
        if let Some(nm) = self.nonce_manager.get(address) {
            if tx.nonce < nm.current_nonce {
                return Err(TransactionError::NonceTooLow(nm.current_nonce, tx.nonce));
            }
        }

        self.pending.insert(
            key,
            PendingTx {
                tx: tx.clone(),
                submitted_at: Instant::now(),
                replaced_by: None,
            },
        );

        // Update nonce manager
        let nm = self
            .nonce_manager
            .entry(address.to_lowercase())
            .or_insert_with(|| NonceManager {
                current_nonce: tx.nonce,
                pending_count: 0,
            });
        nm.pending_count += 1;

        Ok(())
    }

    /// Get next nonce for address
    pub fn get_next_nonce(&self, address: &str) -> u64 {
        self.nonce_manager
            .get(&address.to_lowercase())
            .map(|nm| nm.current_nonce + nm.pending_count)
            .unwrap_or(0)
    }

    /// Mark transaction as confirmed
    pub fn confirm(&mut self, address: &str, nonce: u64) {
        let key = format!("{}:{}", address.to_lowercase(), nonce);
        self.pending.remove(&key);

        if let Some(nm) = self.nonce_manager.get_mut(&address.to_lowercase()) {
            if nonce >= nm.current_nonce {
                nm.current_nonce = nonce + 1;
                nm.pending_count = nm.pending_count.saturating_sub(1);
            }
        }
    }

    /// Cancel pending transaction
    pub fn cancel(&mut self, address: &str, nonce: u64) -> Result<()> {
        let key = format!("{}:{}", address.to_lowercase(), nonce);

        if let Some(pending) = self.pending.get_mut(&key) {
            pending.replaced_by = Some("cancelled".to_string());
        }

        Ok(())
    }

    /// Replace transaction with new gas price
    pub fn speed_up(
        &mut self,
        address: &str,
        nonce: u64,
        new_gas_price: String,
    ) -> Result<EvmTransaction> {
        let key = format!("{}:{}", address.to_lowercase(), nonce);

        if let Some(pending) = self.pending.get_mut(&key) {
            pending.tx.gas_price = new_gas_price.clone();

            if let Some(ref mut max_fee) = pending.tx.max_fee_per_gas {
                *max_fee = new_gas_price.clone();
            }

            return Ok(pending.tx.clone());
        }

        Err(TransactionError::SigningError(
            "Transaction not found".to_string(),
        ))
    }
}

impl Default for TransactionPool {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================================================
// Gas Optimizer
// ============================================================================

/// Maximum number of gas price samples retained per chain.
const MAX_GAS_HISTORY: usize = 4096;

/// Minimum samples an hour needs before find_optimal_time will consider it.
const MIN_SAMPLES_PER_HOUR: usize = 4;

/// Recommended priority fee per urgency level, in wei.
///
/// This is a floor rather than a measurement: a tip is a policy choice, and
/// there is no canonical on-chain value for it. The base fee is the part that
/// has to come from the chain, and that is supplied by the caller.
fn recommended_priority_fee(urgency: GasUrgency) -> u64 {
    match urgency {
        GasUrgency::Low => 1_000_000_000,     // 1 gwei
        GasUrgency::Medium => 2_000_000_000,  // 2 gwei
        GasUrgency::High => 5_000_000_000,    // 5 gwei
        GasUrgency::Urgent => 10_000_000_000, // 10 gwei
    }
}

/// Gas optimization strategies.
///
/// The optimizer holds no market data of its own. A caller supplies the base
/// fee it has read from the chain, so estimation fails closed with
/// MissingBaseFee rather than substituting an invented default. A guessed base
/// fee is not a harmless placeholder: too low and every transaction is
/// rejected by the node, too high and the user is overcharged.
pub struct GasOptimizer {
    base_fees: HashMap<u64, BaseFeeTracker>,
    historical_prices: HashMap<u64, Vec<GasPricePoint>>,
}

struct BaseFeeTracker {
    current: u64,
    last_update: Instant,
}

struct GasPricePoint {
    price: u64,
    /// Seconds since the Unix epoch at which the sample was observed.
    timestamp: u64,
}

impl GasOptimizer {
    pub fn new() -> Self {
        Self {
            base_fees: HashMap::new(),
            historical_prices: HashMap::new(),
        }
    }

    /// Records the base fee most recently read from `chain_id`.
    pub fn update_base_fee(&mut self, chain_id: u64, base_fee: u64) {
        self.base_fees.insert(
            chain_id,
            BaseFeeTracker {
                current: base_fee,
                last_update: Instant::now(),
            },
        );
    }

    /// Records a gas price sample, used by find_optimal_time. Older samples
    /// are dropped once MAX_GAS_HISTORY is reached.
    pub fn record_gas_price(&mut self, chain_id: u64, price: u64, timestamp: u64) {
        let points = self.historical_prices.entry(chain_id).or_default();
        points.push(GasPricePoint { price, timestamp });
        if points.len() > MAX_GAS_HISTORY {
            let excess = points.len() - MAX_GAS_HISTORY;
            points.drain(..excess);
        }
    }

    /// The base fee last supplied for a chain, if any.
    pub fn base_fee(&self, chain_id: u64) -> Option<u64> {
        self.base_fees.get(&chain_id).map(|tracker| tracker.current)
    }

    /// Estimate a gas price for a chain.
    ///
    /// Fails when no base fee has been supplied for the chain. The total cost
    /// is computed in wei as a u128 because max_fee * gas_limit overflows u64
    /// for realistic values.
    pub fn estimate_gas(&mut self, chain_id: u64, urgency: GasUrgency) -> Result<GasEstimate> {
        let base_fee = self
            .base_fee(chain_id)
            .ok_or(TransactionError::MissingBaseFee(chain_id))?;

        let max_priority = recommended_priority_fee(urgency);
        let max_fee = base_fee.saturating_add(max_priority);

        let gas_limit: u64 = 21_000;

        let total_cost = (max_fee as u128) * (gas_limit as u128);

        Ok(GasEstimate {
            gas_limit,
            gas_price: format!("0x{:x}", max_fee),
            max_fee_per_gas: format!("0x{:x}", max_fee),
            max_priority_fee_per_gas: format!("0x{:x}", max_priority),
            total_cost: format!("0x{:x}", total_cost),
        })
    }

    /// The hour of the day (UTC) whose recorded gas prices have the lowest
    /// mean, expressed as a Duration since midnight.
    ///
    /// Returns None when no hour has at least MIN_SAMPLES_PER_HOUR samples,
    /// because a single observation is not evidence of a cheaper window.
    pub fn find_optimal_time(&self, chain_id: u64) -> Option<Duration> {
        let points = self.historical_prices.get(&chain_id)?;

        let mut total = [0u128; 24];
        let mut count = [0u64; 24];

        for point in points {
            let hour = ((point.timestamp / 3600) % 24) as usize;
            total[hour] += point.price as u128;
            count[hour] += 1;
        }

        let mut best: Option<(u64, u128)> = None;
        for hour in 0..24 {
            if count[hour] < MIN_SAMPLES_PER_HOUR as u64 {
                continue;
            }
            let mean = total[hour] / count[hour] as u128;
            if best.map_or(true, |(_, best_mean)| mean < best_mean) {
                best = Some((hour as u64, mean));
            }
        }

        best.map(|(hour, _)| Duration::from_secs(hour * 3600))
    }
}

impl Default for GasOptimizer {
    fn default() -> Self {
        Self::new()
    }
}

#[derive(Debug, Clone, Copy)]
pub enum GasUrgency {
    Low,
    Medium,
    High,
    Urgent,
}

// ============================================================================
// Transaction Builder
// ============================================================================

/// Build transactions with validation
pub struct TransactionBuilder {
    chain_id: i64,
    from: String,
    to: String,
    value: Option<String>,
    data: Option<String>,
    gas_limit: Option<u64>,
    gas_price: Option<String>,
    nonce: Option<u64>,
    access_list: Option<Vec<AccessListItem>>,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct AccessListItem {
    pub address: String,
    pub storage_keys: Vec<String>,
}

impl TransactionBuilder {
    pub fn new(chain_id: i64, from: String, to: String) -> Self {
        Self {
            chain_id,
            from,
            to,
            value: None,
            data: None,
            gas_limit: None,
            gas_price: None,
            nonce: None,
            access_list: None,
        }
    }

    pub fn value(mut self, value: String) -> Self {
        self.value = Some(value);
        self
    }

    pub fn data(mut self, data: String) -> Self {
        self.data = Some(data);
        self
    }

    pub fn gas_limit(mut self, limit: u64) -> Self {
        self.gas_limit = Some(limit);
        self
    }

    pub fn gas_price(mut self, price: String) -> Self {
        self.gas_price = Some(price);
        self
    }

    pub fn nonce(mut self, nonce: u64) -> Self {
        self.nonce = Some(nonce);
        self
    }

    pub fn build(self) -> EvmTransaction {
        let gas_price = self.gas_price.unwrap_or_else(|| "0x4".to_string()); // 4 gwei default
        let gas_limit = self.gas_limit.unwrap_or(21000);

        EvmTransaction {
            chain_id: self.chain_id as u64,
            nonce: self.nonce.unwrap_or(0),
            to: self.to,
            value: self.value.unwrap_or_else(|| "0x0".to_string()),
            data: self.data.unwrap_or_default(),
            gas_limit,
            gas_price: gas_price.clone(),
            max_priority_fee_per_gas: Some(gas_price.clone()),
            max_fee_per_gas: Some(gas_price),
        }
    }

    /// Validate transaction
    pub fn validate(&self) -> Result<()> {
        // Validate address format
        if !self.from.starts_with("0x") || self.from.len() != 42 {
            return Err(TransactionError::SigningError(
                "Invalid from address".to_string(),
            ));
        }

        if !self.to.starts_with("0x") || self.to.len() != 42 {
            return Err(TransactionError::SigningError(
                "Invalid to address".to_string(),
            ));
        }

        Ok(())
    }
}

// ============================================================================
// Transaction Simulator
// ============================================================================

/// Simulate transactions before execution
pub struct TransactionSimulator {
    vm_state: HashMap<String, AccountState>,
}

struct AccountState {
    balance: u64,
    code: Vec<u8>,
    storage: HashMap<String, String>,
}

impl TransactionSimulator {
    pub fn new() -> Self {
        Self {
            vm_state: HashMap::new(),
        }
    }

    /// Simulate transaction execution
    pub fn simulate(&mut self, tx: &EvmTransaction) -> SimulationResult {
        // In production, would use actual EVM
        // Simplified simulation

        let gas_used = if tx.data.is_empty() {
            21000 // Basic transfer
        } else {
            65000 // Contract interaction
        };

        SimulationResult {
            success: true,
            gas_used,
            gas_refunded: 0,
            logs: vec![],
            return_value: "0x".to_string(),
            revert_reason: None,
        }
    }

    /// Simulate multiple transactions (bundle)
    pub fn simulate_bundle(&mut self, txs: &[EvmTransaction]) -> Vec<SimulationResult> {
        txs.iter().map(|tx| self.simulate(tx)).collect()
    }
}

impl Default for TransactionSimulator {
    fn default() -> Self {
        Self::new()
    }
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct SimulationResult {
    pub success: bool,
    pub gas_used: u64,
    pub gas_refunded: u64,
    pub logs: Vec<Log>,
    pub return_value: String,
    pub revert_reason: Option<String>,
}

// ============================================================================
// Multi-Chain Coordinator
// ============================================================================

/// Coordinate transactions across multiple chains
pub struct MultiChainCoordinator {
    chains: Vec<ChainConfig>,
    pending_txs: HashMap<String, MultiChainTx>,
}

struct ChainConfig {
    id: i64,
    rpc_url: String,
    confirmations: u64,
}

#[derive(Debug, Clone)]
struct MultiChainTx {
    id: String,
    transactions: Vec<EvmTransaction>,
    status: MultiChainStatus,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub enum MultiChainStatus {
    Pending,
    PartiallyConfirmed { confirmed_chains: Vec<i64> },
    Confirmed,
    Failed { failed_chains: Vec<i64> },
}

impl MultiChainCoordinator {
    pub fn new() -> Self {
        Self {
            chains: vec![],
            pending_txs: HashMap::new(),
        }
    }

    /// Add supported chain
    pub fn add_chain(&mut self, id: i64, rpc_url: String, confirmations: u64) {
        self.chains.push(ChainConfig {
            id,
            rpc_url,
            confirmations,
        });
    }

    /// Execute multi-chain transaction
    pub fn execute(
        &mut self,
        id: String,
        transactions: Vec<EvmTransaction>,
    ) -> Result<MultiChainTx> {
        let mc_tx = MultiChainTx {
            id: id.clone(),
            transactions,
            status: MultiChainStatus::Pending,
        };

        self.pending_txs.insert(id, mc_tx.clone());

        Ok(mc_tx)
    }

    /// Check confirmation status
    pub fn check_status(&self, id: &str) -> Option<&MultiChainTx> {
        self.pending_txs.get(id)
    }
}

impl Default for MultiChainCoordinator {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================================================
// Rate Limiter
// ============================================================================

/// Rate limiter for transaction submission
pub struct RateLimiter {
    max_per_second: u32,
    current_count: u32,
    window_start: Instant,
}

impl RateLimiter {
    pub fn new(max_per_second: u32) -> Self {
        Self {
            max_per_second,
            current_count: 0,
            window_start: Instant::now(),
        }
    }

    /// Check if allowed to submit
    pub fn allow(&mut self) -> bool {
        let elapsed = self.window_start.elapsed();

        if elapsed > Duration::from_secs(1) {
            self.current_count = 0;
            self.window_start = Instant::now();
        }

        if self.current_count < self.max_per_second {
            self.current_count += 1;
            return true;
        }

        false
    }

    /// Wait time until next submission
    pub fn wait_time(&self) -> Duration {
        let elapsed = self.window_start.elapsed();
        if elapsed > Duration::from_secs(1) {
            return Duration::ZERO;
        }
        Duration::from_secs(1) - elapsed
    }
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_transaction_pool() {
        let mut pool = TransactionPool::new();

        let tx = EvmTransaction {
            chain_id: 1,
            nonce: 0,
            to: "0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E".to_string(),
            value: "0x0".to_string(),
            data: "0x".to_string(),
            gas_limit: 21000,
            gas_price: "0x4".to_string(),
            max_priority_fee_per_gas: Some("0x4".to_string()),
            max_fee_per_gas: Some("0x4".to_string()),
        };

        let from = "0xabcd1234567890123456789012345678901234567";
        pool.add(from, tx.clone()).unwrap();

        assert_eq!(pool.get_next_nonce(from), 1);
    }

    #[test]
    fn test_gas_estimator_uses_the_supplied_base_fee() {
        let mut optimizer = GasOptimizer::new();

        // Without any observed base fee the estimate must fail closed rather
        // than invent one.
        assert!(matches!(
            optimizer.estimate_gas(1, GasUrgency::Medium),
            Err(TransactionError::MissingBaseFee(1))
        ));

        optimizer.update_base_fee(1, 30_000_000_000);
        let estimate = optimizer
            .estimate_gas(1, GasUrgency::Medium)
            .expect("estimate must succeed once a base fee is known");

        // 30 gwei base + 2 gwei medium tip.
        assert_eq!(estimate.gas_limit, 21_000);
        assert_eq!(
            estimate.max_fee_per_gas,
            format!("0x{:x}", 32_000_000_000u64)
        );
        assert_eq!(
            estimate.max_priority_fee_per_gas,
            format!("0x{:x}", 2_000_000_000u64)
        );
        assert_eq!(
            estimate.total_cost,
            format!("0x{:x}", 32_000_000_000u128 * 21_000u128)
        );
    }

    #[test]
    fn test_gas_estimator_orders_urgency_levels() {
        let mut optimizer = GasOptimizer::new();
        optimizer.update_base_fee(1, 20_000_000_000);

        let low = optimizer.estimate_gas(1, GasUrgency::Low).unwrap();
        let urgent = optimizer.estimate_gas(1, GasUrgency::Urgent).unwrap();

        assert!(low.max_fee_per_gas != urgent.max_fee_per_gas);
        assert_eq!(low.gas_price, format!("0x{:x}", 21_000_000_000u64));
        assert_eq!(urgent.gas_price, format!("0x{:x}", 30_000_000_000u64));
    }

    #[test]
    fn test_find_optimal_time_requires_history() {
        let mut optimizer = GasOptimizer::new();
        assert!(optimizer.find_optimal_time(1).is_none());

        // A single sample is not enough evidence for an hourly window.
        optimizer.record_gas_price(1, 1_000, 0);
        assert!(optimizer.find_optimal_time(1).is_none());
    }

    #[test]
    fn test_find_optimal_time_picks_the_cheapest_hour() {
        let mut optimizer = GasOptimizer::new();

        // Hour 3 of the epoch day is cheap, hour 12 is expensive. Four samples
        // each clears the minimum.
        for _ in 0..4 {
            optimizer.record_gas_price(1, 5_000, 3 * 3600);
            optimizer.record_gas_price(1, 90_000, 12 * 3600);
        }

        assert_eq!(
            optimizer.find_optimal_time(1),
            Some(Duration::from_secs(3 * 3600))
        );
    }

    #[test]
    fn test_record_gas_price_bounds_history() {
        let mut optimizer = GasOptimizer::new();
        for i in 0..(MAX_GAS_HISTORY + 100) {
            optimizer.record_gas_price(1, 1_000, i as u64);
        }

        assert_eq!(
            optimizer.historical_prices.get(&1).map(Vec::len),
            Some(MAX_GAS_HISTORY)
        );
    }

    #[test]
    fn test_rate_limiter() {
        let mut limiter = RateLimiter::new(10);

        assert!(limiter.allow());
        assert!(limiter.allow());

        for _ in 0..8 {
            limiter.allow();
        }

        assert!(!limiter.allow());
    }

    #[test]
    fn test_transaction_builder() {
        let tx = TransactionBuilder::new(
            1,
            "0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E".to_string(),
            "0xabcd1234567890123456789012345678901234567".to_string(),
        )
        .value("0x1000".to_string())
        .gas_limit(21000)
        .build();

        assert_eq!(tx.chain_id, 1);
        assert_eq!(tx.gas_limit, 21000);
    }
}
