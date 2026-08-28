//! TigerWallet Advanced Orders Execution Engine

use rust_decimal::Decimal;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum AdvancedOrderType {
    StopLoss,
    TakeProfit,
    TrailingStop,
    OCO,
    TWAP,
    VWAP,
    Iceberg,
    Trigger,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum OrderSide {
    Buy,
    Sell,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdvancedOrder {
    pub id: String,
    pub user_id: String,
    pub symbol: String,
    pub side: OrderSide,
    pub order_type: AdvancedOrderType,
    pub quantity: Decimal,
    pub stop_price: Option<Decimal>,
    pub trigger_price: Option<Decimal>,
    pub status: AdvancedOrderStatus,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum AdvancedOrderStatus {
    Active,
    Triggered,
    Filled,
    Cancelled,
}

impl AdvancedOrder {
    pub fn new_stop_loss(user_id: &str, symbol: &str, side: OrderSide, quantity: Decimal, stop_price: Decimal) -> Self {
        Self {
            id: Uuid::new_v4().to_string(),
            user_id: user_id.to_string(),
            symbol: symbol.to_string(),
            side,
            order_type: AdvancedOrderType::StopLoss,
            quantity,
            stop_price: Some(stop_price),
            trigger_price: None,
            status: AdvancedOrderStatus::Active,
        }
    }

    pub fn should_trigger(&self, current_price: Decimal) -> bool {
        if let Some(stop_price) = self.stop_price {
            match self.side {
                OrderSide::Buy => current_price >= stop_price,
                OrderSide::Sell => current_price <= stop_price,
            }
        } else {
            false
        }
    }
}