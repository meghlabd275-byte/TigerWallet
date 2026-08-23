//! TigerWallet Risk Engine
//! Manages position risk limits, exposure caps, and risk calculations

use rust_decimal::Decimal;
use rust_decimal_macros::dec;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use parking_lot::RwLock;

/// Risk limit type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum RiskLimitType {
    PositionSize,
    OpenInterest,
    OrderSize,
    Leverage,
    DailyVolume,
    Drawdown,
}

impl std::fmt::Display for RiskLimitType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            RiskLimitType::PositionSize => write!(f, "PositionSize"),
            RiskLimitType::OpenInterest => write!(f, "OpenInterest"),
            RiskLimitType::OrderSize => write!(f, "OrderSize"),
            RiskLimitType::Leverage => write!(f, "Leverage"),
            RiskLimitType::DailyVolume => write!(f, "DailyVolume"),
            RiskLimitType::Drawdown => write!(f, "Drawdown"),
        }
    }
}

/// Risk limit
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskLimit {
    pub limit_type: RiskLimitType,
    pub max_value: Decimal,
    pub current_value: Decimal,
    pub used_percent: Decimal,
}

impl RiskLimit {
    pub fn new(limit_type: RiskLimitType, max_value: Decimal) -> Self {
        Self {
            limit_type,
            max_value,
            current_value: dec!(0),
            used_percent: dec!(0),
        }
    }

    pub fn can_increase(&self, amount: Decimal) -> bool {
        self.current_value + amount <= self.max_value
    }

    pub fn increase(&mut self, amount: Decimal) -> Result<(), RiskError> {
        if !self.can_increase(amount) {
            return Err(RiskError::LimitExceeded(self.limit_type.to_string()));
        }
        self.current_value += amount;
        self.used_percent = self.current_value / self.max_value;
        Ok(())
    }

    pub fn decrease(&mut self, amount: Decimal) {
        self.current_value = (self.current_value - amount).max(dec!(0));
        self.used_percent = self.current_value / self.max_value;
    }
}

/// User risk profile
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserRiskProfile {
    pub user_id: String,
    pub risk_level: RiskLevel,
    pub position_limit: Decimal,
    pub leverage_limit: Decimal,
    pub open_interest: Decimal,
    pub daily_volume_limit: Decimal,
    pub max_positions: u32,
    pub allow_market_orders: bool,
    pub allow_high_leverage: bool,
    pub trading_enabled: bool,
    pub withdrawal_enabled: bool,
}

impl UserRiskProfile {
    pub fn new(user_id: &str) -> Self {
        Self {
            user_id: user_id.to_string(),
            risk_level: RiskLevel::Standard,
            position_limit: dec!(1000000),
            leverage_limit: dec!(10),
            open_interest: dec!(0),
            daily_volume_limit: dec!(10000000),
            max_positions: 20,
            allow_market_orders: true,
            allow_high_leverage: false,
            trading_enabled: true,
            withdrawal_enabled: true,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum RiskLevel {
    Demo,
    Standard,
    Intermediate,
    Advanced,
    Pro,
}

/// Position risk metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PositionRisk {
    pub symbol: String,
    pub side: PositionSide,
    pub quantity: Decimal,
    pub notional_value: Decimal,
    pub leverage: Decimal,
    pub margin: Decimal,
    pub entry_price: Decimal,
    pub mark_price: Decimal,
    pub liquidation_price: Decimal,
    pub unrealized_pnl: Decimal,
    pub margin_ratio: Decimal,
    pub risk_score: Decimal,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum PositionSide {
    Long,
    Short,
}

/// Risk check result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskCheck {
    pub allowed: bool,
    pub reason: Option<String>,
    pub warnings: Vec<String>,
    pub new_leverage: Option<Decimal>,
    pub new_margin_ratio: Option<Decimal>,
}

impl RiskCheck {
    pub fn allowed() -> Self {
        Self {
            allowed: true,
            reason: None,
            warnings: Vec::new(),
            new_leverage: None,
            new_margin_ratio: None,
        }
    }

    pub fn denied(reason: &str) -> Self {
        Self {
            allowed: false,
            reason: Some(reason.to_string()),
            warnings: Vec::new(),
            new_leverage: None,
            new_margin_ratio: None,
        }
    }
}

/// Risk engine
pub struct RiskEngine {
    user_profiles: RwLock<HashMap<String, UserRiskProfile>>,
    global_limits: RwLock<HashMap<RiskLimitType, RiskLimit>>,
    symbol_limits: RwLock<HashMap<String, HashMap<RiskLimitType, RiskLimit>>>,
}

impl RiskEngine {
    pub fn new() -> Self {
        let mut global_limits = HashMap::new();
        global_limits.insert(RiskLimitType::PositionSize, RiskLimit::new(RiskLimitType::PositionSize, dec!(100000000)));
        global_limits.insert(RiskLimitType::OpenInterest, RiskLimit::new(RiskLimitType::OpenInterest, dec!(500000000)));
        global_limits.insert(RiskLimitType::Leverage, RiskLimit::new(RiskLimitType::Leverage, dec!(100)));
        
        Self {
            user_profiles: RwLock::new(HashMap::new()),
            global_limits: RwLock::new(global_limits),
            symbol_limits: RwLock::new(HashMap::new()),
        }
    }

    pub fn create_user_profile(&self, user_id: &str) -> UserRiskProfile {
        let profile = UserRiskProfile::new(user_id);
        let mut profiles = self.user_profiles.write();
        profiles.insert(user_id.to_string(), profile.clone());
        profile
    }

    pub fn get_user_profile(&self, user_id: &str) -> Option<UserRiskProfile> {
        self.user_profiles.read().get(user_id).cloned()
    }

    pub fn check_order(
        &self,
        user_id: &str,
        symbol: &str,
        side: PositionSide,
        quantity: Decimal,
        price: Decimal,
        leverage: Decimal,
    ) -> RiskCheck {
        let profiles = self.user_profiles.read();
        let profile = match profiles.get(user_id) {
            Some(p) => p,
            None => return RiskCheck::denied("User profile not found"),
        };
        
        if !profile.trading_enabled {
            return RiskCheck::denied("Trading disabled");
        }
        
        if leverage > profile.leverage_limit && !profile.allow_high_leverage {
            return RiskCheck::denied("Leverage exceeds limit");
        }
        
        let notional_value = quantity * price;
        if notional_value > profile.position_limit {
            return RiskCheck::denied("Order size exceeds position limit");
        }
        
        if profile.open_interest + notional_value > profile.position_limit {
            return RiskCheck::denied("Would exceed position limit");
        }
        
        let mut warnings = Vec::new();
        
        if leverage >= dec!(50) {
            warnings.push("High leverage warning".to_string());
        }
        
        if quantity * price > profile.daily_volume_limit / dec!(10) {
            warnings.push("Large order warning".to_string());
        }
        
        RiskCheck {
            allowed: true,
            reason: None,
            warnings,
            new_leverage: Some(leverage),
            new_margin_ratio: None,
        }
    }

    pub fn check_withdrawal(&self, user_id: &str, amount: Decimal) -> RiskCheck {
        let profiles = self.user_profiles.read();
        let profile = match profiles.get(user_id) {
            Some(p) => p,
            None => return RiskCheck::denied("User profile not found"),
        };
        
        if !profile.withdrawal_enabled {
            return RiskCheck::denied("Withdrawal disabled");
        }
        
        RiskCheck::allowed()
    }

    pub fn update_position(&self, user_id: &str, notional_value: Decimal) {
        let mut profiles = self.user_profiles.write();
        if let Some(profile) = profiles.get_mut(user_id) {
            profile.open_interest = notional_value;
        }
    }

    pub fn set_risk_level(&self, user_id: &str, level: RiskLevel) {
        let mut profiles = self.user_profiles.write();
        if let Some(profile) = profiles.get_mut(user_id) {
            profile.risk_level = level;
            
            match level {
                RiskLevel::Demo => {
                    profile.leverage_limit = dec!(5);
                    profile.position_limit = dec!(1000);
                    profile.max_positions = 5;
                }
                RiskLevel::Standard => {
                    profile.leverage_limit = dec!(10);
                    profile.position_limit = dec!(100000);
                    profile.max_positions = 10;
                }
                RiskLevel::Intermediate => {
                    profile.leverage_limit = dec!(20);
                    profile.position_limit = dec!(1000000);
                    profile.max_positions = 20;
                }
                RiskLevel::Advanced => {
                    profile.leverage_limit = dec!(50);
                    profile.position_limit = dec!(10000000);
                    profile.max_positions = 30;
                }
                RiskLevel::Pro => {
                    profile.leverage_limit = dec!(100);
                    profile.position_limit = dec!(100000000);
                    profile.max_positions = 50;
                }
            }
        }
    }

    pub fn enable_trading(&self, user_id: &str) {
        let mut profiles = self.user_profiles.write();
        if let Some(profile) = profiles.get_mut(user_id) {
            profile.trading_enabled = true;
        }
    }

    pub fn disable_trading(&self, user_id: &str) {
        let mut profiles = self.user_profiles.write();
        if let Some(profile) = profiles.get_mut(user_id) {
            profile.trading_enabled = false;
        }
    }
}

impl Default for RiskEngine {
    fn default() -> Self {
        Self::new()
    }
}

#[derive(Debug, thiserror::Error)]
pub enum RiskError {
    #[error("Limit exceeded: {0}")]
    LimitExceeded(String),
    
    #[error("Risk check failed: {0}")]
    RiskCheckFailed(String),
    
    #[error("Invalid parameter: {0}")]
    InvalidParameter(String),
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_risk_check_order() {
        let engine = RiskEngine::new();
        engine.create_user_profile("user1");
        
        let check = engine.check_order(
            "user1",
            "BTC-USD",
            PositionSide::Long,
            dec!(1),
            dec!(50000),
            dec!(10),
        );
        
        assert!(check.allowed);
    }
}