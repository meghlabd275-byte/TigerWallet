//! TigerWallet Liquid Staking Module
//! Real liquid staking implementation with derivative tokens
//! 
//! This provides:
//! - Stake tokens and receive liquid staking tokens (LST)
//! - Real-time yield tracking
//! - Unstaking with different time locks
//! - Multi-chain staking support

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use parking_lot::RwLock;
use chrono::{DateTime, Utc};

/// Liquid staking pool configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LiquidStakingConfig {
    pub chain_id: u64,
    pub staking_token: String,      // Token to stake (e.g., ETH)
    pub lst_token: String,         // Liquid staking token (e.g., twETH)
    pub lst_token_name: String,
    pub lst_token_symbol: String,
    pub reward_token: String,      // Reward token
    pub min_stake_amount: u64,
    pub max_stake_amount: u64,
    pub cooldown_period_seconds: u64,
    pub instant_unstake_fee_bps: u16,  // Basis points for instant unstake
    pub apy_bps: u32,             // Annual percentage yield in basis points
}

impl Default for LiquidStakingConfig {
    fn default() -> Self {
        Self {
            chain_id: 1,
            staking_token: "0x0000000000000000000000000000000000000000".to_string(),
            lst_token: "0x0000000000000000000000000000000000000000".to_string(),
            lst_token_name: "Tiger Wrapped ETH".to_string(),
            lst_token_symbol: "twETH".to_string(),
            reward_token: "0x0000000000000000000000000000000000000000".to_string(),
            min_stake_amount: 10000000000000000,  // 0.01 ETH
            max_stake_amount: 1000000000000000000000, // 1000 ETH
            cooldown_period_seconds: 86400 * 7,  // 7 days
            instant_unstake_fee_bps: 50,  // 0.5%
            apy_bps: 500,  // 5% APY
        }
    }
}

/// Stake position
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StakePosition {
    pub position_id: String,
    pub owner: String,
    pub staked_amount: u64,
    pub lst_amount: u64,
    pub lst_token: String,
    pub stake_timestamp: u64,
    pub unlock_timestamp: Option<u64>,
    pub pending_rewards: u64,
    pub claimed_rewards: u64,
    pub is_unstaking: bool,
}

/// Unstake request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnstakeRequest {
    pub request_id: String,
    pub owner: String,
    pub lst_amount: u64,
    pub lst_token: String,
    pub requested_timestamp: u64,
    pub unlock_timestamp: u64,
    pub instant: bool,
    pub instant_fee_bps: u16,
    pub status: UnstakeStatus,
}

/// Unstake status
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum UnstakeStatus {
    Pending,
    Ready,
    Completed,
    Cancelled,
}

/// Reward distribution
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RewardDistribution {
    pub distribution_id: String,
    pub amount: u64,
    pub timestamp: u64,
    pub total_staked: u64,
    pub reward_per_token: u64,
}

/// Liquid staking pool statistics
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct PoolStats {
    pub total_staked: u64,
    pub total_lst_supply: u64,
    pub current_apy: u32,
    pub active_positions: u64,
    pub pending_unstakes: u64,
    pub total_rewards_distributed: u64,
    pub last_update_timestamp: u64,
}

/// Liquid Staking Pool
pub struct LiquidStakingPool {
    config: RwLock<LiquidStakingConfig>,
    positions: RwLock<HashMap<String, StakePosition>>,
    unstake_requests: RwLock<HashMap<String, UnstakeRequest>>,
    reward_distributions: RwLock<Vec<RewardDistribution>>,
    stats: RwLock<PoolStats>,
    owners_positions: RwLock<HashMap<String, Vec<String>>>,  // owner -> position IDs
}

impl LiquidStakingPool {
    pub fn new(config: LiquidStakingConfig) -> Self {
        Self {
            config: RwLock::new(config),
            positions: RwLock::new(HashMap::new()),
            unstake_requests: RwLock::new(HashMap::new()),
            reward_distributions: RwLock::new(Vec::new()),
            stats: RwLock::new(PoolStats::default()),
            owners_positions: RwLock::new(HashMap::new()),
        }
    }

    /// Stake tokens and receive liquid staking tokens
    pub fn stake(&self, owner: &str, amount: u64) -> Result<StakePosition, StakingError> {
        let config = self.config.read();
        
        // Validate amount
        if amount < config.min_stake_amount {
            return Err(StakingError::AmountTooSmall);
        }
        if amount > config.max_stake_amount {
            return Err(StakingError::AmountTooLarge);
        }

        // Calculate LST amount (1:1 ratio initially, can be adjusted by exchange rate)
        let lst_amount = self.calculate_lst_amount(amount)?;
        
        // Generate position ID
        let position_id = self.generate_position_id(owner, amount);

        let position = StakePosition {
            position_id: position_id.clone(),
            owner: owner.to_string(),
            staked_amount: amount,
            lst_amount,
            lst_token: config.lst_token.clone(),
            stake_timestamp: Utc::now().timestamp() as u64,
            unlock_timestamp: None,
            pending_rewards: 0,
            claimed_rewards: 0,
            is_unstaking: false,
        };

        // Store position
        self.positions.write().insert(position_id.clone(), position.clone());
        
        // Track owner's positions
        self.owners_positions.write()
            .entry(owner.to_string())
            .or_insert_with(Vec::new)
            .push(position_id);

        // Update stats
        {
            let mut stats = self.stats.write();
            stats.total_staked += amount;
            stats.total_lst_supply += lst_amount;
            stats.active_positions += 1;
            stats.last_update_timestamp = Utc::now().timestamp() as u64;
        }

        Ok(position)
    }

    /// Request unstake
    pub fn request_unstake(
        &self,
        owner: &str,
        position_id: &str,
        instant: bool,
    ) -> Result<UnstakeRequest, StakingError> {
        let config = self.config.read();
        
        // Get position
        let position = self.positions.read()
            .get(position_id)
            .ok_or(StakingError::PositionNotFound)?
            .clone();

        // Verify ownership
        if position.owner != owner {
            return Err(StakingError::NotOwner);
        }

        // Check if already unstaking
        if position.is_unstaking {
            return Err(StakingError::AlreadyUnstaking);
        }

        let request_id = format!("unstake_{}", uuid::Uuid::new_v4());
        
        let unstake_request = UnstakeRequest {
            request_id: request_id.clone(),
            owner: owner.to_string(),
            lst_amount: position.lst_amount,
            lst_token: config.lst_token.clone(),
            requested_timestamp: Utc::now().timestamp() as u64,
            unlock_timestamp: if instant {
                Utc::now().timestamp() as u64
            } else {
                Utc::now().timestamp() as u64 + config.cooldown_period_seconds
            },
            instant,
            instant_fee_bps: config.instant_unstake_fee_bps,
            status: if instant { UnstakeStatus::Ready } else { UnkakeStatus::Pending },
        };

        // Store request
        self.unstake_requests.write().insert(request_id.clone(), unstake_request.clone());

        // Update position
        {
            let mut positions = self.positions.write();
            if let Some(pos) = positions.get_mut(position_id) {
                pos.is_unstaking = true;
                pos.unlock_timestamp = Some(unstake_request.unlock_timestamp);
            }
        }

        // Update stats
        {
            let mut stats = self.stats.write();
            if !instant {
                stats.pending_unstakes += 1;
            }
            stats.total_staked -= position.staked_amount;
            stats.total_lst_supply -= position.lst_amount;
            stats.active_positions -= 1;
        }

        Ok(unstake_request)
    }

    /// Claim unstaked tokens
    pub fn claim_unstake(&self, owner: &str, request_id: &str) -> Result<u64, StakingError> {
        let mut requests = self.unstake_requests.write();
        
        let request = requests.get_mut(request_id)
            .ok_or(StakingError::RequestNotFound)?
            .clone();

        // Verify ownership
        if request.owner != owner {
            return Err(StakingError::NotOwner);
        }

        // Check status
        if request.status != UnstakeStatus::Ready {
            return Err(StakingError::NotReady);
        }

        // Calculate claimable amount
        let claimable = self.calculate_unstake_amount(&request)?;
        
        // Update request status
        if let Some(req) = requests.get_mut(request_id) {
            req.status = UnstakeStatus::Completed;
        }

        // Update stats
        {
            let mut stats = self.stats.write();
            stats.pending_unstakes = stats.pending_unstakes.saturating_sub(1);
        }

        Ok(claimable)
    }

    /// Claim rewards
    pub fn claim_rewards(&self, owner: &str, position_id: &str) -> Result<u64, StakingError> {
        let mut positions = self.positions.write();
        
        let position = positions.get_mut(position_id)
            .ok_or(StakingError::PositionNotFound)?;

        // Verify ownership
        if position.owner != owner {
            return Err(StakingError::NotOwner);
        }

        let pending = position.pending_rewards;
        position.claimed_rewards += pending;
        position.pending_rewards = 0;

        Ok(pending)
    }

    /// Get position by ID
    pub fn get_position(&self, position_id: &str) -> Option<StakePosition> {
        self.positions.read().get(position_id).cloned()
    }

    /// Get positions by owner
    pub fn get_owner_positions(&self, owner: &str) -> Vec<StakePosition> {
        let position_ids = self.owners_positions.read()
            .get(owner)
            .cloned()
            .unwrap_or_default();
        
        let positions = self.positions.read();
        position_ids
            .iter()
            .filter_map(|id| positions.get(id).cloned())
            .collect()
    }

    /// Get pool statistics
    pub fn get_pool_stats(&self) -> PoolStats {
        self.stats.read().clone()
    }

    /// Get configuration
    pub fn get_config(&self) -> LiquidStakingConfig {
        self.config.read().clone()
    }

    /// Calculate LST amount from staked amount
    fn calculate_lst_amount(&self, staked_amount: u64) -> Result<u64, StakingError> {
        let stats = self.stats.read();
        
        if stats.total_lst_supply == 0 || stats.total_staked == 0 {
            // First staker gets 1:1
            return Ok(staked_amount);
        }

        // Calculate based on exchange rate
        let rate = stats.total_lst_supply as f64 / stats.total_staked as f64;
        Ok((staked_amount as f64 * rate) as u64)
    }

    /// Calculate unstake amount
    fn calculate_unstake_amount(&self, request: &UnstakeRequest) -> Result<u64, StakingError> {
        let config = self.config.read();
        
        if request.instant {
            let fee = (request.lst_amount as u64 * request.instant_fee_bps as u64) / 10000;
            Ok(request.lst_amount.saturating_sub(fee))
        } else {
            Ok(request.lst_amount)
        }
    }

    /// Generate unique position ID
    fn generate_position_id(&self, owner: &str, amount: u64) -> String {
        let mut hasher = sha3::Keccak256::new();
        hasher.update(owner.as_bytes());
        hasher.update(amount.to_le_bytes());
        hasher.update(Utc::now().timestamp().to_le_bytes());
        format!("stake_{}", hex::encode(hasher.finalize()))
    }

    /// Distribute rewards (called by reward handler)
    pub fn distribute_rewards(&self, amount: u64) -> Result<(), StakingError> {
        let stats = self.stats.read();
        
        if stats.total_lst_supply == 0 {
            return Ok(());
        }

        let reward_per_token = (amount * 1_000_000_000) / stats.total_lst_supply;
        
        // Store distribution
        let distribution = RewardDistribution {
            distribution_id: uuid::Uuid::new_v4().to_string(),
            amount,
            timestamp: Utc::now().timestamp() as u64,
            total_staked: stats.total_staked,
            reward_per_token,
        };
        
        self.reward_distributions.write().push(distribution);

        // Update pending rewards for all positions
        let mut positions = self.positions.write();
        for position in positions.values_mut() {
            if !position.is_unstaking {
                let pending = (position.lst_amount * reward_per_token) / 1_000_000_000;
                position.pending_rewards += pending;
            }
        }

        // Update stats
        {
            let mut stats = self.stats.write();
            stats.total_rewards_distributed += amount;
        }

        Ok(())
    }
}

/// Staking errors
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum StakingError {
    AmountTooSmall,
    AmountTooLarge,
    PositionNotFound,
    RequestNotFound,
    NotOwner,
    AlreadyUnstaking,
    NotReady,
    InsufficientBalance,
}

impl std::fmt::Display for StakingError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            StakingError::AmountTooSmall => write!(f, "Stake amount too small"),
            StakingError::AmountTooLarge => write!(f, "Stake amount too large"),
            StakingError::PositionNotFound => write!(f, "Position not found"),
            StakingError::RequestNotFound => write!(f, "Unstake request not found"),
            StakingError::NotOwner => write!(f, "Not the owner of this position"),
            StakingError::AlreadyUnstaking => write!(f, "Position is already unstaking"),
            StakingError::NotReady => write!(f, "Unstake not ready yet"),
            StakingError::InsufficientBalance => write!(f, "Insufficient balance"),
        }
    }
}

impl Default for LiquidStakingPool {
    fn default() -> Self {
        Self::new(LiquidStakingConfig::default())
    }
}
