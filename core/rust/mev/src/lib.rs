//! TigerSwap MEV Protection Engine - Production-Ready
//! 
//! COMPLETELY SELF-CONTAINED with:
//! - Sandwich attack detection
//! - Front-running detection
//! - Back-running detection
//! - Flashbots integration
//! - Private transaction routing
//! - Bundle simulation
//! - Price impact analysis

pub mod bundle;
pub mod detection;
pub mod protection;

pub use bundle::{BundleBuilder, BundleResult, BundleTransaction};
pub use detection::{AttackDetector, AttackType, Severity, SuspiciousPattern, TransactionData};
pub use protection::{BatchProtectionResult, MEVConfig, MEVProtectionService, ProtectionResult};

use std::collections::{HashMap, HashSet};
use std::sync::RwLock;
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};

/// MEV Attack Types
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum MEVAttackType {
    Sandwich,
    FrontRun,
    BackRun,
    Arbitrage,
    Liquidation,
    Unknown,
}

/// Transaction pattern for MEV detection
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionPattern {
    pub sender: String,
    pub target: String,
    pub value: u64,
    pub gas_price: u64,
    pub timestamp: u64,
    pub token_in: Option<String>,
    pub token_out: Option<String>,
    pub amount_in: Option<u64>,
    pub pool_address: Option<String>,
}

impl TransactionPattern {
    pub fn new(sender: String, target: String, value: u64, gas_price: u64) -> Self {
        Self {
            sender,
            target,
            value,
            gas_price,
            timestamp: current_timestamp(),
            token_in: None,
            token_out: None,
            amount_in: None,
            pool_address: None,
        }
    }
}

/// Detected MEV opportunity or attack
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MEVOpportunity {
    pub id: u64,
    pub attack_type: MEVAttackType,
    /// The profit this attack is expected to extract, in the pool's input
    /// token, or `None` when it cannot be derived from the observed fields.
    /// `None` is not zero: it means the inputs needed to price the attack
    /// (pool reserves, the attacker's exit price) were not available, and a
    /// caller must not present it as a concrete number.
    pub profit_estimate: Option<i64>,
    pub affected_addresses: Vec<String>,
    pub pool_addresses: Vec<String>,
    pub timestamp: u64,
    pub confidence: f64,
    pub recommendations: Vec<String>,
}

impl MEVOpportunity {
    pub fn new(
        id: u64,
        attack_type: MEVAttackType,
        profit_estimate: Option<i64>,
        affected_addresses: Vec<String>,
    ) -> Self {
        Self {
            id,
            attack_type,
            profit_estimate,
            affected_addresses,
            pool_addresses: Vec::new(),
            timestamp: current_timestamp(),
            confidence: 0.0,
            recommendations: Vec::new(),
        }
    }
}

/// Flashbots bundle for MEV extraction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MevBundle {
    pub transactions: Vec<Vec<u8>>,
    pub block_number: u64,
    pub min_timestamp: Option<u64>,
    pub max_timestamp: Option<u64>,
    pub reverting_tx_hashes: Vec<String>,
}

impl MevBundle {
    pub fn new(block_number: u64) -> Self {
        Self {
            transactions: Vec::new(),
            block_number,
            min_timestamp: None,
            max_timestamp: None,
            reverting_tx_hashes: Vec::new(),
        }
    }

    pub fn add_transaction(&mut self, tx: Vec<u8>) {
        self.transactions.push(tx);
    }

    pub fn allow_revert(&mut self, tx_hash: String) {
        self.reverting_tx_hashes.push(tx_hash);
    }
}

/// MEV Protection Engine
pub struct MEVEngine {
    detected_attacks: RwLock<Vec<MEVOpportunity>>,
    monitored_pools: RwLock<HashSet<String>>,
    recent_transactions: RwLock<HashMap<String, TransactionPattern>>,
    flashbots_endpoint: RwLock<Option<String>>,
    block_number: RwLock<u64>,
    pool_fee_bps: RwLock<u32>,
}

/// The fee tier used to price a sandwich when the caller has not set one.
/// 30 bps is the 0.30% tier that most AMM pools use.
pub const DEFAULT_POOL_FEE_BPS: u32 = 30;

impl MEVEngine {
    pub fn new() -> Self {
        Self {
            detected_attacks: RwLock::new(Vec::new()),
            monitored_pools: RwLock::new(HashSet::new()),
            recent_transactions: RwLock::new(HashMap::new()),
            flashbots_endpoint: RwLock::new(None),
            block_number: RwLock::new(0),
            pool_fee_bps: RwLock::new(DEFAULT_POOL_FEE_BPS),
        }
    }

    /// Set the fee tier, in basis points, of the pools this engine monitors.
    ///
    /// A sandwich attacker's profit is bounded by the fee the victim pays, so
    /// the estimate is only as good as this number.
    pub fn set_pool_fee_bps(&self, bps: u32) {
        *self.pool_fee_bps.write().unwrap() = bps;
    }

    /// Configure Flashbots endpoint
    pub fn set_flashbots_endpoint(&self, endpoint: String) {
        let mut ep = self.flashbots_endpoint.write().unwrap();
        *ep = Some(endpoint);
    }

    /// Add pool to monitor
    pub fn add_monitored_pool(&self, pool_address: String) {
        let mut pools = self.monitored_pools.write().unwrap();
        pools.insert(pool_address);
    }

    /// Update block number
    pub fn update_block(&self, block_number: u64) {
        let mut bn = self.block_number.write().unwrap();
        *bn = block_number;
    }

    /// Analyze transaction for MEV
    pub fn analyze_transaction(&self, tx: &TransactionPattern) -> Option<MEVOpportunity> {
        // Record transaction
        {
            let mut txs = self.recent_transactions.write().unwrap();
            txs.insert(tx.sender.clone(), tx.clone());
        }

        // Check for sandwich attack
        if let Some(attack) = self.detect_sandwich(tx) {
            self.record_attack(attack.clone());
            return Some(attack);
        }

        // Check for front-running
        if let Some(attack) = self.detect_frontrun(tx) {
            self.record_attack(attack.clone());
            return Some(attack);
        }

        // Check for back-running
        if let Some(attack) = self.detect_backrun(tx) {
            self.record_attack(attack.clone());
            return Some(attack);
        }

        None
    }

    /// Detect a sandwich attack.
    ///
    /// A sandwich is an attacker's trade that brackets a victim's trade on the
    /// same pool, in the same token, with the attacker paying more gas to land
    /// first. So the victim is another sender already in the recent set whose
    /// gas price is below this transaction's.
    ///
    /// The previous implementation looked the victim up under the sender of
    /// the transaction being analysed. `analyze_transaction` records that
    /// transaction under its own sender before calling this, so the lookup
    /// always returned the transaction itself and the gas comparison was
    /// always false — the branch could never be reached.
    fn detect_sandwich(&self, tx: &TransactionPattern) -> Option<MEVOpportunity> {
        let pool = tx.pool_address.as_ref()?;
        let token = tx.token_in.as_ref()?;

        let txs = self.recent_transactions.read().unwrap();

        for (address, victim_tx) in txs.iter() {
            if address == &tx.sender {
                continue;
            }
            // The victim pays less gas, so the attacker lands ahead of them.
            if victim_tx.gas_price >= tx.gas_price {
                continue;
            }
            if victim_tx.pool_address.as_ref() != Some(pool) {
                continue;
            }
            if victim_tx.token_in.as_ref() != Some(token) {
                continue;
            }

            return Some(MEVOpportunity::new(
                0,
                MEVAttackType::Sandwich,
                self.estimate_sandwich_profit(victim_tx),
                vec![tx.sender.clone(), victim_tx.sender.clone()],
            ));
        }

        None
    }

    /// Detect front-running
    fn detect_frontrun(&self, tx: &TransactionPattern) -> Option<MEVOpportunity> {
        if tx.token_in.is_none() || tx.amount_in.is_none() {
            return None;
        }

        let txs = self.recent_transactions.read().unwrap();
        
        // Check if someone is copying victim's trade
        for (address, recent_tx) in txs.iter() {
            if address == &tx.sender {
                continue;
            }

            // Similar trade with higher gas price
            if recent_tx.token_in == tx.token_in &&
               recent_tx.token_out == tx.token_out &&
               recent_tx.gas_price < tx.gas_price {
                
                return Some(MEVOpportunity::new(
                    0,
                    MEVAttackType::FrontRun,
                    self.estimate_frontrun_profit(),
                    vec![tx.sender.clone(), address.clone()],
                ));
            }
        }
        
        None
    }

    /// Detect back-running
    fn detect_backrun(&self, tx: &TransactionPattern) -> Option<MEVOpportunity> {
        // Back-running typically happens after large swaps
        if tx.amount_in.unwrap_or(0) < 100_000_000 {
            return None;
        }

        let txs = self.recent_transactions.read().unwrap();
        
        // Look for arbitrageur following large swap
        for (address, recent_tx) in txs.iter() {
            if address == &tx.sender {
                continue;
            }

            // Similar pool, different direction, lower gas
            if recent_tx.pool_address == tx.pool_address &&
               recent_tx.token_in != tx.token_in &&
               recent_tx.gas_price < tx.gas_price {
                
                return Some(MEVOpportunity::new(
                    0,
                    MEVAttackType::BackRun,
                    self.estimate_backrun_profit(),
                    vec![tx.sender.clone(), address.clone()],
                ));
            }
        }
        
        None
    }

    /// Estimate sandwich profit.
    ///
    /// A sandwich attacker buys ahead of the victim and sells behind them.
    /// What they can capture is bounded by the fee the victim pays on their
    /// own trade, so the estimate is `victim_amount * fee_bps / 10_000`.
    ///
    /// This needs a fee tier. Callers that have not set one get the engine
    /// default, which is a documented assumption rather than a measurement.
    /// Returns `None` when the victim's trade size is unknown, because a
    /// profit cannot be derived from nothing.
    fn estimate_sandwich_profit(&self, victim: &TransactionPattern) -> Option<i64> {
        let victim_amount = victim.amount_in?;
        if victim_amount == 0 {
            return None;
        }

        let fee_bps = u64::from(*self.pool_fee_bps.read().unwrap());
        let profit = (victim_amount as u128 * fee_bps as u128) / 10_000;
        i64::try_from(profit).ok()
    }

    /// Estimate front-run profit.
    ///
    /// Always `None`. Pricing a front-run means knowing what the attacker's
    /// position is worth once the victim's trade moves the pool, and that
    /// requires the pool reserves. The gas-price difference the previous
    /// implementation used is a cost the attacker pays, not a profit, so it
    /// cannot stand in for one. Detection still reports the pattern; only the
    /// number is withheld.
    fn estimate_frontrun_profit(&self) -> Option<i64> {
        None
    }

    /// Estimate back-run profit.
    ///
    /// Always `None`. A back-runner's profit is the arbitrage between the
    /// price the victim's swap leaves behind and the price the pool returns
    /// to. Both come from the reserves, which this detector does not hold, so
    /// the number cannot be derived from the swap size alone. Detection still
    /// reports the pattern; only the number is withheld.
    fn estimate_backrun_profit(&self) -> Option<i64> {
        None
    }

    /// Record detected attack
    fn record_attack(&self, attack: MEVOpportunity) {
        let mut attacks = self.detected_attacks.write().unwrap();
        attacks.push(attack);
        
        // Keep only last 1000 attacks
        if attacks.len() > 1000 {
            attacks.remove(0);
        }
    }

    /// Get all detected attacks
    pub fn get_detected_attacks(&self) -> Vec<MEVOpportunity> {
        self.detected_attacks.read().unwrap().clone()
    }

    /// Create Flashbots bundle
    pub fn create_bundle(&self, txs: Vec<Vec<u8>>, block: u64) -> MevBundle {
        let mut bundle = MevBundle::new(block);
        for tx in txs {
            bundle.add_transaction(tx);
        }
        bundle
    }

    /// Simulate a bundle against real chain state.
    ///
    /// Fails closed. Simulating a bundle means executing it against the state
    /// of a specific block, which requires an archive node. The previous
    /// implementation returned a constant per transaction and called that a
    /// simulation, which would have told a caller that an unprofitable bundle
    /// was worth submitting. There is no honest answer to give without a
    /// node, so this reports that rather than inventing one.
    pub fn simulate_bundle(&self, _bundle: &MevBundle) -> Result<BundleSimulation, String> {
        Err("bundle simulation requires an archive node; none is configured".to_string())
    }

    /// Check if transaction is safe (not vulnerable to MEV)
    pub fn check_transaction_safety(&self, tx: &TransactionPattern) -> SafetyReport {
        let mut attacks = self.analyze_transaction(tx);
        
        if let Some(attack) = attacks.take() {
            SafetyReport {
                is_safe: false,
                risk_level: RiskLevel::High,
                detected_attacks: vec![attack.attack_type],
                recommendations: self.generate_recommendations(&attack),
            }
        } else {
            SafetyReport {
                is_safe: true,
                risk_level: RiskLevel::Low,
                detected_attacks: vec![],
                recommendations: vec![],
            }
        }
    }

    /// Generate protection recommendations
    fn generate_recommendations(&self, attack: &MEVOpportunity) -> Vec<String> {
        match attack.attack_type {
            MEVAttackType::Sandwich => vec![
                "Use private transaction routing".to_string(),
                "Set tight slippage tolerance".to_string(),
                "Consider batch transactions".to_string(),
            ],
            MEVAttackType::FrontRun => vec![
                "Use commit-reveal scheme".to_string(),
                "Randomize transaction timing".to_string(),
                "Use encrypted mempool".to_string(),
            ],
            MEVAttackType::BackRun => vec![
                "Increase transaction size".to_string(),
                "Use flashbots protected endpoint".to_string(),
            ],
            _ => vec![
                "Use MEV protection service".to_string(),
            ],
        }
    }

    /// Get protection statistics
    pub fn get_stats(&self) -> MEVStats {
        let attacks = self.detected_attacks.read().unwrap();
        let pools = self.monitored_pools.read().unwrap();
        
        let mut sandwich_count = 0;
        let mut frontrun_count = 0;
        let mut backrun_count = 0;
        
        for attack in attacks.iter() {
            match attack.attack_type {
                MEVAttackType::Sandwich => sandwich_count += 1,
                MEVAttackType::FrontRun => frontrun_count += 1,
                MEVAttackType::BackRun => backrun_count += 1,
                _ => {}
            }
        }
        
        MEVStats {
            total_attacks: attacks.len(),
            sandwich_attacks: sandwich_count,
            frontrun_attacks: frontrun_count,
            backrun_attacks: backrun_count,
            monitored_pools: pools.len(),
        }
    }
}

impl Default for MEVEngine {
    fn default() -> Self {
        Self::new()
    }
}

/// Bundle simulation result
#[derive(Debug, Clone)]
pub struct BundleSimulation {
    pub total_profit: i64,
    pub gas_used: u64,
    pub state_changes: Vec<StateChange>,
    pub reverts: Vec<String>,
}

#[derive(Debug, Clone)]
pub struct StateChange {
    pub address: String,
    pub slot: String,
    pub before: String,
    pub after: String,
}

/// Safety report
#[derive(Debug, Clone)]
pub struct SafetyReport {
    pub is_safe: bool,
    pub risk_level: RiskLevel,
    pub detected_attacks: Vec<MEVAttackType>,
    pub recommendations: Vec<String>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RiskLevel {
    Low,
    Medium,
    High,
    Critical,
}

/// MEV statistics
#[derive(Debug, Clone)]
pub struct MEVStats {
    pub total_attacks: usize,
    pub sandwich_attacks: usize,
    pub frontrun_attacks: usize,
    pub backrun_attacks: usize,
    pub monitored_pools: usize,
}

fn current_timestamp() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_mev_engine_creation() {
        let engine = MEVEngine::new();
        let stats = engine.get_stats();
        assert_eq!(stats.total_attacks, 0);
    }

    #[test]
    fn test_safety_check() {
        let engine = MEVEngine::new();
        
        let tx = TransactionPattern::new(
            "0xTrader".to_string(),
            "0xPool".to_string(),
            0,
            100,
        );
        
        let report = engine.check_transaction_safety(&tx);
        assert!(report.is_safe);
    }

    #[test]
    fn test_bundle_creation() {
        let engine = MEVEngine::new();
        
        let mut bundle = engine.create_bundle(
            vec![vec![1, 2, 3], vec![4, 5, 6]],
            12345,
        );
        
        assert_eq!(bundle.transactions.len(), 2);
        assert_eq!(bundle.block_number, 12345);
    }

    #[test]
    fn test_pool_monitoring() {
        let engine = MEVEngine::new();
        engine.add_monitored_pool("0xPool1".to_string());
        engine.add_monitored_pool("0xPool2".to_string());
        
        let stats = engine.get_stats();
        assert_eq!(stats.monitored_pools, 2);
    }

    fn pattern(sender: &str, gas_price: u64, amount_in: Option<u64>) -> TransactionPattern {
        let mut tx = TransactionPattern::new(
            sender.to_string(),
            "0xPool".to_string(),
            0,
            gas_price,
        );
        tx.token_in = Some("0xWETH".to_string());
        tx.token_out = Some("0xUSDC".to_string());
        tx.amount_in = amount_in;
        tx.pool_address = Some("0xPool".to_string());
        tx
    }

    #[test]
    fn a_sandwich_profit_is_bounded_by_the_victim_fee() {
        // 100_000_000 in, 30 bps fee => 300_000 capturable.
        let engine = MEVEngine::new();
        let victim = pattern("0xVictim", 100, Some(100_000_000));
        let attacker = pattern("0xAttacker", 200, Some(1_000));

        engine.analyze_transaction(&victim);
        let attack = engine
            .analyze_transaction(&attacker)
            .expect("a matching pool and token must be detected as a sandwich");

        assert_eq!(attack.attack_type, MEVAttackType::Sandwich);
        assert_eq!(
            attack.profit_estimate,
            Some(300_000),
            "profit must follow from the fee tier, not a magic multiplier"
        );
    }

    #[test]
    fn the_sandwich_estimate_follows_a_configured_fee_tier() {
        let engine = MEVEngine::new();
        engine.set_pool_fee_bps(100); // 1.00%

        let victim = pattern("0xVictim", 100, Some(100_000_000));
        let attacker = pattern("0xAttacker", 200, Some(1_000));

        engine.analyze_transaction(&victim);
        let attack = engine.analyze_transaction(&attacker).expect("detected");

        assert_eq!(attack.profit_estimate, Some(1_000_000));
    }

    #[test]
    fn an_unpriceable_attack_reports_unknown_rather_than_zero() {
        // The victim's trade size is unknown, so no profit can be derived.
        // Reporting 0 would claim the attack is worthless.
        let engine = MEVEngine::new();
        let victim = pattern("0xVictim", 100, None);
        let attacker = pattern("0xAttacker", 200, Some(1_000));

        engine.analyze_transaction(&victim);
        let attack = engine.analyze_transaction(&attacker).expect("detected");

        assert_eq!(
            attack.profit_estimate, None,
            "an unknown profit must be None, not a fabricated 0"
        );
    }

    #[test]
    fn a_frontrun_never_invents_a_profit_figure() {
        // The old estimate returned amount * 0.01 - gas_delta, which is
        // unrelated to what the attacker would actually make.
        let engine = MEVEngine::new();
        // A different pool keeps sandwich detection out of the way, so this
        // exercises the front-run branch on its own.
        let mut victim = pattern("0xVictim", 100, Some(50_000));
        victim.pool_address = Some("0xOtherPool".to_string());
        let attacker = pattern("0xAttacker", 500, Some(50_000));

        engine.analyze_transaction(&victim);
        let attack = engine.analyze_transaction(&attacker).expect("detected");

        assert_eq!(attack.attack_type, MEVAttackType::FrontRun);
        assert_eq!(attack.profit_estimate, None);
    }

    #[test]
    fn bundle_simulation_fails_closed_without_a_node() {
        let engine = MEVEngine::new();
        let bundle = engine.create_bundle(vec![vec![1, 2, 3]], 42);

        let result = engine.simulate_bundle(&bundle);

        assert!(
            result.is_err(),
            "simulation without chain state must error, not return fake numbers"
        );
    }

}
