//! TigerWallet Liquidation Engine
//! Handles forced liquidation of undercollateralized positions

use rust_decimal::Decimal;
use rust_decimal_macros::dec;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use parking_lot::RwLock;
use chrono::{DateTime, Utc};
use uuid::Uuid;

use crate::position_engine::{PositionSide, MarginType};

/// Liquidation type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum LiquidationType {
    MarginCall,
    Partial,
    Full,
    Bankrupt,
}

/// Liquidation reason
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum LiquidationReason {
    MaintenanceMarginBreach,
    Bankrupt,
    MarginCallExpired,
    Deleveraging,
    MarketMaker,
}

/// Liquidation status
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum LiquidationStatus {
    Pending,
    InProgress,
    Completed,
    Failed,
}

/// Liquidation order
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LiquidationOrder {
    pub id: String,
    pub position_id: String,
    pub user_id: String,
    pub symbol: String,
    pub quantity: Decimal,
    pub price: Decimal,
    pub liquidation_type: LiquidationType,
    pub reason: LiquidationReason,
    pub status: LiquidationStatus,
    pub created_at: DateTime<Utc>,
    pub executed_at: Option<DateTime<Utc>>,
    pub executor_fee: Decimal,
    pub remaining_quantity: Decimal,
    pub order_ids: Vec<String>,
}

impl LiquidationOrder {
    pub fn new(
        position_id: &str,
        user_id: &str,
        symbol: &str,
        quantity: Decimal,
        price: Decimal,
        liquidation_type: LiquidationType,
        reason: LiquidationReason,
    ) -> Self {
        Self {
            id: Uuid::new_v4().to_string(),
            position_id: position_id.to_string(),
            user_id: user_id.to_string(),
            symbol: symbol.to_string(),
            quantity,
            price,
            liquidation_type,
            reason,
            status: LiquidationStatus::Pending,
            created_at: Utc::now(),
            executed_at: None,
            executor_fee: dec!(0.001), // 0.1%
            remaining_quantity: quantity,
            order_ids: Vec::new(),
        }
    }
}

/// Liquidation parameters
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LiquidationParams {
    pub max_liquidation_batch_size: Decimal,
    pub liquidation_timeout_ms: u64,
    pub partial_liquidation_percent: Decimal,
    pub liquidation_fee: Decimal,
    pub bankruptcy_fee: Decimal,
    pub liquidation_buffer: Decimal,
    pub max_liquidations_per_block: u32,
    pub liquidation_priority: LiquidationPriority,
}

impl Default for LiquidationParams {
    fn default() -> Self {
        Self {
            max_liquidation_batch_size: dec!(100), // 100 BTC equivalent
            liquidation_timeout_ms: 5000,
            partial_liquidation_percent: dec!(0.25), // 25% of position
            liquidation_fee: dec!(0.0005),        // 0.05%
            bankruptcy_fee: dec!(0.001),         // 0.1%
            liquidation_buffer: dec!(1.1),      // 10% buffer
            max_liquidations_per_block: 100,
            liquidation_priority: LiquidationPriority::MarginRatio,
        }
    }
}

/// Liquidation priority
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum LiquidationPriority {
    MarginRatio,
    UnrealizedPnL,
    Size,
    Age,
}

/// Liquidation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LiquidationResult {
    pub order_id: String,
    pub filled_quantity: Decimal,
    pub fill_price: Decimal,
    pub executor_fee: Decimal,
    pub remaining_margin: Decimal,
    pub new_liquidation_price: Option<Decimal>,
    pub status: LiquidationStatus,
}

/// Liquidation engine
pub struct LiquidationEngine {
    params: RwLock<LiquidationParams>,
    pending_liquidations: RwLock<Vec<LiquidationOrder>>,
    completed_liquidations: RwLock<Vec<LiquidationOrder>>,
    liquidation_history: RwLock<Vec<LiquidationEvent>>,
}

impl LiquidationEngine {
    pub fn new() -> Self {
        Self {
            params: RwLock::new(LiquidationParams::default()),
            pending_liquidations: RwLock::new(Vec::new()),
            completed_liquidations: RwLock::new(Vec::new()),
            liquidation_history: RwLock::new(Vec::new()),
        }
    }

    pub fn check_liquidation(
        &self,
        position: &Position,
        mark_price: Decimal,
    ) -> LiquidationCheck {
        let params = self.params.read();
        
        // Calculate margin ratio
        let maintenance_margin = position.notional_value(mark_price) * params.liquidation_buffer * dec!(0.005);
        let margin_ratio = position.margin / maintenance_margin;
        
        if margin_ratio <= dec!(1.0) {
            // Bankrupt - full liquidation
            LiquidationCheck {
                should_liquidate: true,
                liquidation_type: LiquidationType::Bankrupt,
                reason: LiquidationReason::Bankrupt,
                liquidation_price: Some(position.liquidation_price(mark_price)),
                partial_liquidate_quantity: None,
                priority_score: dec!(0),
            }
        } else if margin_ratio <= dec!(1.1) {
            // Maintenance margin breach - partial liquidation
            let partial_qty = position.quantity * params.partial_liquidation_percent;
            LiquidationCheck {
                should_liquidate: true,
                liquidation_type: LiquidationType::Partial,
                reason: LiquidationReason::MaintenanceMarginBreach,
                liquidation_price: Some(position.liquidation_price(mark_price)),
                partial_liquidate_quantity: Some(partial_qty),
                priority_score: dec!(1.1) - margin_ratio,
            }
        } else if margin_ratio <= dec!(1.25) {
            // Margin call
            LiquidationCheck {
                should_liquidate: false,
                liquidation_type: LiquidationType::MarginCall,
                reason: LiquidationReason::MarginCallExpired,
                liquidation_price: Some(position.liquidation_price(mark_price)),
                partial_liquidate_quantity: None,
                priority_score: dec!(1.25) - margin_ratio,
            }
        } else {
            LiquidationCheck {
                should_liquidate: false,
                liquidation_type: LiquidationType::MarginCall,
                reason: LiquidationReason::MarginCallExpired,
                liquidation_price: Some(position.liquidation_price(mark_price)),
                partial_liquidate_quantity: None,
                priority_score: dec!(10), // Low priority
            }
        }
    }

    pub fn execute_liquidation(
        &self,
        position: &Position,
        mark_price: Decimal,
        check: &LiquidationCheck,
    ) -> Result<LiquidationResult, LiquidationError> {
        let params = self.params.read();
        
        let quantity = check.partial_liquidate_quantity
            .unwrap_or(position.quantity);
        
        let mut order = LiquidationOrder::new(
            &position.id,
            &position.user_id,
            &position.symbol,
            quantity,
            check.liquidation_price.unwrap_or(mark_price),
            check.liquidation_type,
            check.reason,
        );
        
        // Submit liquidation order to matching engine
        // This would integrate with the matching engine
        
        let executor_fee = params.liquidation_fee * quantity * mark_price;
        let remaining_margin = position.margin - executor_fee;
        
        order.status = LiquidationStatus::Completed;
        order.executed_at = Some(Utc::now());
        
        // Record liquidation
        let mut history = self.liquidation_history.write();
        history.push(LiquidationEvent {
            id: order.id.clone(),
            position_id: position.id.clone(),
            user_id: position.user_id.clone(),
            symbol: position.symbol.clone(),
            quantity,
            price: order.price,
            liquidation_type: check.liquidation_type,
            reason: check.reason,
            timestamp: Utc::now(),
        });
        
        Ok(LiquidationResult {
            order_id: order.id,
            filled_quantity: quantity,
            fill_price: order.price,
            executor_fee,
            remaining_margin,
            new_liquidation_price: check.partial_liquidate_quantity
                .map(|_| position.liquidation_price(mark_price)),
            status: LiquidationStatus::Completed,
        })
    }

    pub fn get_pending_liquidations(&self) -> Vec<LiquidationOrder> {
        self.pending_liquidations.read().clone()
    }

    pub fn get_liquidation_history(&self, symbol: Option<&str>, limit: usize) -> Vec<LiquidationEvent> {
        let history = self.liquidation_history.read();
        history.iter()
            .filter(|e| symbol.map(|s| e.symbol == s).unwrap_or(true))
            .take(limit)
            .cloned()
            .collect()
    }

    pub fn update_params(&self, params: LiquidationParams) {
        *self.params.write() = params;
    }
}

impl Default for LiquidationEngine {
    fn default() -> Self {
        Self::new()
    }
}

/// Liquidation check result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LiquidationCheck {
    pub should_liquidate: bool,
    pub liquidation_type: LiquidationType,
    pub reason: LiquidationReason,
    pub liquidation_price: Option<Decimal>,
    pub partial_liquidate_quantity: Option<Decimal>,
    pub priority_score: Decimal,
}

/// Liquidation event for history
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LiquidationEvent {
    pub id: String,
    pub position_id: String,
    pub user_id: String,
    pub symbol: String,
    pub quantity: Decimal,
    pub price: Decimal,
    pub liquidation_type: LiquidationType,
    pub reason: LiquidationReason,
    pub timestamp: DateTime<Utc>,
}

/// Liquidation errors
#[derive(Debug, thiserror::Error)]
pub enum LiquidationError {
    #[error("Position not found: {0}")]
    PositionNotFound(String),
    
    #[error("Liquidation failed: {0}")]
    LiquidationFailed(String),
    
    #[error("Insufficient liquidity: {0}")]
    InsufficientLiquidity(String),
    
    #[error("Order submission failed: {0}")]
    OrderSubmissionFailed(String),
}

/// Position with liquidation calculations
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Position {
    pub id: String,
    pub user_id: String,
    pub symbol: String,
    pub side: PositionSide,
    pub quantity: Decimal,
    pub entry_price: Decimal,
    pub leverage: Decimal,
    pub margin: Decimal,
    pub margin_type: MarginType,
}

impl Position {
    pub fn notional_value(&self, mark_price: Decimal) -> Decimal {
        self.quantity * mark_price
    }

    pub fn liquidation_price(&self, mark_price: Decimal) -> Decimal {
        let notional = self.notional_value(mark_price);
        let margin_ratio = dec!(1.0) / self.leverage;
        
        match self.side {
            PositionSide::Long => {
                mark_price * (dec!(1) - margin_ratio)
            }
            PositionSide::Short => {
                mark_price * (dec!(1) + margin_ratio)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_liquidation_check_bankrupt() {
        let engine = LiquidationEngine::new();
        
        let position = Position {
            id: "pos1".to_string(),
            user_id: "user1".to_string(),
            symbol: "BTC-USD".to_string(),
            side: PositionSide::Long,
            quantity: dec!(1),
            entry_price: dec!(50000),
            leverage: dec!(20),
            margin: dec!(2500),
            margin_type: MarginType::Cross,
        };
        
        let check = engine.check_liquidation(&position, dec!(42000));
        assert!(check.should_liquidate);
        assert_eq!(check.liquidation_type, LiquidationType::Bankrupt);
    }
    
    #[test]
    fn test_liquidation_check_partial() {
        let engine = LiquidationEngine::new();
        
        let position = Position {
            id: "pos1".to_string(),
            user_id: "user1".to_string(),
            symbol: "BTC-USD".to_string(),
            side: PositionSide::Long,
            quantity: dec!(1),
            entry_price: dec!(50000),
            leverage: dec!(10),
            margin: dec!(5000),
            margin_type: MarginType::Cross,
        };
        
        let check = engine.check_liquidation(&position, dec!(47500));
        assert!(check.should_liquidate);
        assert!(check.partial_liquidate_quantity.is_some());
    }
}