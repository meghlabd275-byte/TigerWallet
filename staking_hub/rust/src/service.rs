//! Staking Service Implementation

use crate::error::Error;
use crate::models::*;
use chrono::{DateTime, Utc};
use std::sync::{Arc, RwLock};
use std::collections::HashMap;
use uuid::Uuid;

/// Staking Service
pub struct StakingService {
    positions: RwLock<HashMap<String, StakingPosition>>,
    pools: RwLock<HashMap<String, StakingPool>>,
}

impl StakingService {
    pub fn new() -> Self {
        let service = Self {
            positions: RwLock::new(HashMap::new()),
            pools: RwLock::new(HashMap::new()),
        };
        service.initialize_pools();
        service
    }

    fn initialize_pools(&self) {
        let mut pools = self.pools.write().unwrap();
        
        pools.insert("eth2".to_string(), StakingPool {
            id: "eth2".to_string(),
            name: "Ethereum 2.0".to_string(),
            token: "ETH".to_string(),
            apy: 4.5,
            min_stake: 0.01,
            max_stake: 1000.0,
            lock_period: 365 * 24 * 60 * 60,
            validators: vec![],
        });
        
        pools.insert("sol".to_string(), StakingPool {
            id: "sol".to_string(),
            name: "Solana".to_string(),
            token: "SOL".to_string(),
            apy: 6.2,
            min_stake: 0.1,
            max_stake: 10000.0,
            lock_period: 180 * 24 * 60 * 60,
            validators: vec![],
        });
    }

    pub fn stake(&self, req: StakeRequest) -> Result<StakingPosition, Error> {
        let pools = self.pools.read().unwrap();
        
        let pool = pools.get(&req.pool_id)
            .ok_or_else(|| Error::NotFound("Pool not found".to_string()))?;
        
        if req.amount < pool.min_stake {
            return Err(Error::InvalidRequest("Amount below minimum".to_string()));
        }
        
        if req.amount > pool.max_stake {
            return Err(Error::InvalidRequest("Amount above maximum".to_string()));
        }
        
        let position = StakingPosition {
            id: Uuid::new_v4().to_string(),
            user_id: req.user_id,
            token: pool.token.clone(),
            amount: req.amount,
            rewards: 0.0,
            lock_period: pool.lock_period,
            started_at: Utc::now(),
            unlocked_at: None,
        };
        
        let mut positions = self.positions.write().unwrap();
        positions.insert(position.id.clone(), position.clone());
        
        Ok(position)
    }

    pub fn unstake(&self, req: UnstakeRequest) -> Result<StakingPosition, Error> {
        let mut positions = self.positions.write().unwrap();
        
        let position = positions.get_mut(&req.position_id)
            .ok_or_else(|| Error::NotFound("Position not found".to_string()))?;
        
        if position.unlocked_at.is_some() {
            return Err(Error::UnstakingError("Already unlocked".to_string()));
        }
        
        position.unlocked_at = Some(Utc::now());
        
        Ok(position.clone())
    }

    pub fn claim_rewards(&self, req: ClaimRewardsRequest) -> Result<f64, Error> {
        let mut positions = self.positions.write().unwrap();
        
        let position = positions.get_mut(&req.position_id)
            .ok_or_else(|| Error::NotFound("Position not found".to_string()))?;
        
        let rewards = position.rewards;
        position.rewards = 0.0;
        
        Ok(rewards)
    }

    pub fn get_position(&self, position_id: &str) -> Result<StakingPosition, Error> {
        let positions = self.positions.read().unwrap();
        
        positions.get(position_id)
            .cloned()
            .ok_or_else(|| Error::NotFound("Position not found".to_string()))
    }

    pub fn get_pools(&self) -> Vec<StakingPool> {
        let pools = self.pools.read().unwrap();
        pools.values().cloned().collect()
    }
}

impl Default for StakingService {
    fn default() -> Self {
        Self::new()
    }
}