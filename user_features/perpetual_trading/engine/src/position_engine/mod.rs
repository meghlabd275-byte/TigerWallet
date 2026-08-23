//! TigerWallet Position Engine
//! Manages user positions and position updates

use rust_decimal::Decimal;
use rust_decimal_macros::dec;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use parking_lot::RwLock;
use chrono::{DateTime, Utc};
use uuid::Uuid;

/// Position side
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum PositionSide {
    Long,
    Short,
}

/// Margin type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum MarginType {
    Cross,
    Isolated,
}

/// Position
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Position {
    pub id: String,
    pub user_id: String,
    pub symbol: String,
    pub side: PositionSide,
    pub quantity: Decimal,
    pub entry_price: Decimal,
    pub mark_price: Decimal,
    pub leverage: Decimal,
    pub margin: Decimal,
    pub unrealized_pnl: Decimal,
    pub realized_pnl: Decimal,
    pub liquidation_price: Decimal,
    pub margin_type: MarginType,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub position_mode: PositionMode,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum PositionMode {
    OneWay,
    Hedge,
}

/// Position response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PositionResponse {
    pub id: String,
    pub symbol: String,
    pub side: PositionSide,
    pub quantity: Decimal,
    pub entry_price: Decimal,
    pub mark_price: Decimal,
    pub leverage: Decimal,
    pub unrealized_pnl: Decimal,
    pub realized_pnl: Decimal,
    pub roe: Decimal,
    pub margin_ratio: Decimal,
    pub liquidation_price: Decimal,
    pub margin_type: MarginType,
    pub updated_at: DateTime<Utc>,
}

impl Position {
    pub fn new(
        user_id: &str,
        symbol: &str,
        side: PositionSide,
        quantity: Decimal,
        entry_price: Decimal,
        margin: Decimal,
        leverage: Decimal,
        margin_type: MarginType,
    ) -> Self {
        let liquidation_price = match side {
            PositionSide::Long => entry_price * (dec!(1) - dec!(1) / leverage),
            PositionSide::Short => entry_price * (dec!(1) + dec!(1) / leverage),
        };
        
        Self {
            id: Uuid::new_v4().to_string(),
            user_id: user_id.to_string(),
            symbol: symbol.to_string(),
            side,
            quantity,
            entry_price,
            mark_price: entry_price,
            leverage,
            margin,
            unrealized_pnl: dec!(0),
            realized_pnl: dec!(0),
            liquidation_price,
            margin_type,
            created_at: Utc::now(),
            updated_at: Utc::now(),
            position_mode: PositionMode::OneWay,
        }
    }

    pub fn update_mark_price(&mut self, new_mark_price: Decimal) {
        let old_pnl = self.unrealized_pnl;
        
        self.unrealized_pnl = match self.side {
            PositionSide::Long => (new_mark_price - self.entry_price) * self.quantity,
            PositionSide::Short => (self.entry_price - new_mark_price) * self.quantity,
        };
        
        self.realized_pnl += old_pnl;
        self.mark_price = new_mark_price;
        
        self.liquidation_price = match self.side {
            PositionSide::Long => self.entry_price * (dec!(1) - dec!(1) / self.leverage),
            PositionSide::Short => self.entry_price * (dec!(1) + dec!(1) / self.leverage),
        };
        
        self.updated_at = Utc::now();
    }

    pub fn add_trade(&mut self, price: Decimal, quantity: Decimal, side: OrderSide) {
        let new_quantity = self.quantity + quantity;
        let new_entry = (self.entry_price * self.quantity + price * quantity) / new_quantity;
        
        self.entry_price = new_entry;
        self.quantity = new_quantity;
        
        self.liquidation_price = match self.side {
            PositionSide::Long => self.entry_price * (dec!(1) - dec!(1) / self.leverage),
            PositionSide::Short => self.entry_price * (dec!(1) + dec!(1) / self.leverage),
        };
        
        self.updated_at = Utc::now();
    }

    pub fn reduce(&mut self, quantity: Decimal) -> Decimal {
        let reduce_qty = quantity.min(self.quantity);
        let pnl = match self.side {
            PositionSide::Long => (self.mark_price - self.entry_price) * reduce_qty,
            PositionSide::Short => (self.entry_price - self.mark_price) * reduce_qty,
        };
        
        self.realized_pnl += pnl;
        self.quantity -= reduce_qty;
        
        if self.quantity <= dec!(0) {
            self.realized_pnl += self.unrealized_pnl;
            self.unrealized_pnl = dec!(0);
        }
        
        self.updated_at = Utc::now();
        
        pnl
    }

    pub fn is_liquidatable(&self) -> bool {
        match self.side {
            PositionSide::Long => self.mark_price <= self.liquidation_price,
            PositionSide::Short => self.mark_price >= self.liquidation_price,
        }
    }

    pub fn to_response(&self) -> PositionResponse {
        let notional = self.quantity * self.mark_price;
        let roe = if self.margin > dec!(0) {
            self.unrealized_pnl / self.margin
        } else {
            dec!(0)
        };
        
        let margin_ratio = if notional > dec!(0) {
            self.margin / notional
        } else {
            dec!(0)
        };
        
        PositionResponse {
            id: self.id.clone(),
            symbol: self.symbol.clone(),
            side: self.side,
            quantity: self.quantity,
            entry_price: self.entry_price,
            mark_price: self.mark_price,
            leverage: self.leverage,
            unrealized_pnl: self.unrealized_pnl,
            realized_pnl: self.realized_pnl,
            roe,
            margin_ratio,
            liquidation_price: self.liquidation_price,
            margin_type: self.margin_type,
            updated_at: self.updated_at,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum OrderSide {
    Buy,
    Sell,
}

/// Position history entry
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PositionHistory {
    pub id: String,
    pub position_id: String,
    pub symbol: String,
    pub side: PositionSide,
    pub quantity: Decimal,
    pub price: Decimal,
    pub realized_pnl: Decimal,
    pub fee: Decimal,
    pub timestamp: DateTime<Utc>,
}

/// Position engine
pub struct PositionEngine {
    positions: RwLock<HashMap<String, Position>>,
    history: RwLock<HashMap<String, Vec<PositionHistory>>>,
}

impl PositionEngine {
    pub fn new() -> Self {
        Self {
            positions: RwLock::new(HashMap::new()),
            history: RwLock::new(HashMap::new()),
        }
    }

    pub fn open_position(
        &self,
        user_id: &str,
        symbol: &str,
        side: PositionSide,
        quantity: Decimal,
        entry_price: Decimal,
        margin: Decimal,
        leverage: Decimal,
        margin_type: MarginType,
    ) -> Result<Position, PositionError> {
        let position = Position::new(
            user_id,
            symbol,
            side,
            quantity,
            entry_price,
            margin,
            leverage,
            margin_type,
        );
        
        let pos_key = format!("{}:{}", user_id, symbol);
        let mut positions = self.positions.write();
        positions.insert(pos_key, position.clone());
        
        Ok(position)
    }

    pub fn get_position(&self, user_id: &str, symbol: &str) -> Option<Position> {
        let pos_key = format!("{}:{}", user_id, symbol);
        self.positions.read().get(&pos_key).cloned()
    }

    pub fn get_all_positions(&self, user_id: &str) -> Vec<Position> {
        let positions = self.positions.read();
        positions.values()
            .filter(|p| p.user_id == user_id && p.quantity > dec!(0))
            .cloned()
            .collect()
    }

    pub fn update_mark_prices(&self, user_id: &str, prices: HashMap<String, Decimal>) {
        let mut positions = self.positions.write();
        
        for (symbol, price) in prices {
            let pos_key = format!("{}:{}", user_id, symbol);
            if let Some(position) = positions.get_mut(&pos_key) {
                position.update_mark_price(price);
            }
        }
    }

    pub fn close_position(&self, user_id: &str, symbol: &str) -> Result<Decimal, PositionError> {
        let pos_key = format!("{}:{}", user_id, symbol);
        let mut positions = self.positions.write();
        
        if let Some(position) = positions.remove(&pos_key) {
            Ok(position.realized_pnl + position.unrealized_pnl)
        } else {
            Err(PositionError::PositionNotFound)
        }
    }

    pub fn get_positions_summary(&self, user_id: &str) -> PositionsSummary {
        let positions = self.get_all_positions(user_id);
        
        let total_unrealized: Decimal = positions.iter().map(|p| p.unrealized_pnl).sum();
        let total_realized: Decimal = positions.iter().map(|p| p.realized_pnl).sum();
        let total_margin: Decimal = positions.iter().map(|p| p.margin).sum();
        
        PositionsSummary {
            user_id: user_id.to_string(),
            total_positions: positions.len() as u32,
            total_unrealized_pnl: total_unrealized,
            total_realized_pnl: total_realized,
            total_margin,
        }
    }
}

impl Default for PositionEngine {
    fn default() -> Self {
        Self::new()
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PositionsSummary {
    pub user_id: String,
    pub total_positions: u32,
    pub total_unrealized_pnl: Decimal,
    pub total_realized_pnl: Decimal,
    pub total_margin: Decimal,
}

#[derive(Debug, thiserror::Error)]
pub enum PositionError {
    #[error("Position not found")]
    PositionNotFound,
    
    #[error("Insufficient position")]
    InsufficientPosition,
    
    #[error("Invalid parameter")]
    InvalidParameter,
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_open_position() {
        let engine = PositionEngine::new();
        
        let position = engine.open_position(
            "user1",
            "BTC-USD",
            PositionSide::Long,
            dec!(1),
            dec!(50000),
            dec!(5000),
            dec!(10),
            MarginType::Cross,
        ).unwrap();
        
        assert_eq!(position.quantity, dec!(1));
    }
    
    #[test]
    fn test_update_mark_price() {
        let mut position = Position::new(
            "user1",
            "BTC-USD",
            PositionSide::Long,
            dec!(1),
            dec!(50000),
            dec!(5000),
            dec!(10),
            MarginType::Cross,
        );
        
        position.update_mark_price(dec!(55000));
        
        assert!(position.unrealized_pnl > dec!(0));
    }
}