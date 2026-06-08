//! TigerWallet Margin Engine
//! Manages margin calculations, requirements, and cross/isolated margin handling

use rust_decimal::Decimal;
use rust_decimal_macros::dec;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use parking_lot::RwLock;

/// Margin type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum MarginType {
    Cross,
    Isolated,
}

/// Margin status
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum MarginStatus {
    Healthy,
    MarginCall,
    PartialLiquidation,
    FullLiquidation,
    Bankrupt,
}

/// Margin account
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarginAccount {
    pub user_id: String,
    pub total_margin: Decimal,
    pub available_margin: Decimal,
    pub used_margin: Decimal,
    pub unrealized_pnl: Decimal,
    pub realized_pnl: Decimal,
    pub total_position_value: Decimal,
    pub margin_ratio: Decimal,
    pub status: MarginStatus,
    pub margin_call_count: u32,
    pub positions: HashMap<String, PositionMargin>,
}

impl MarginAccount {
    pub fn new(user_id: &str) -> Self {
        Self {
            user_id: user_id.to_string(),
            total_margin: dec!(0),
            available_margin: dec!(0),
            used_margin: dec!(0),
            unrealized_pnl: dec!(0),
            realized_pnl: dec!(0),
            total_position_value: dec!(0),
            margin_ratio: dec!(0),
            status: MarginStatus::Healthy,
            margin_call_count: 0,
            positions: HashMap::new(),
        }
    }

    pub fn update(&mut self) {
        self.total_margin = self.available_margin + self.used_margin + self.unrealized_pnl;
        
        if self.total_position_value > dec!(0) {
            self.margin_ratio = self.total_margin / self.total_position_value;
        } else {
            self.margin_ratio = dec!(0);
        }
        
        self.status = if self.margin_ratio <= dec!(0.005) {
            MarginStatus::Bankrupt
        } else if self.margin_ratio <= dec!(0.01) {
            MarginStatus::FullLiquidation
        } else if self.margin_ratio <= dec!(0.025) {
            MarginStatus::PartialLiquidation
        } else if self.margin_ratio <= dec!(0.05) {
            MarginStatus::MarginCall
        } else {
            MarginStatus::Healthy
        };
    }

    pub fn can_open_position(&self, order_value: Decimal, leverage: Decimal) -> bool {
        let required_margin = order_value / leverage;
        self.available_margin >= required_margin
    }
}

/// Position margin details
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PositionMargin {
    pub symbol: String,
    pub position_side: PositionSide,
    pub quantity: Decimal,
    pub entry_price: Decimal,
    pub mark_price: Decimal,
    pub leverage: Decimal,
    pub margin: Decimal,
    pub notional_value: Decimal,
    pub unrealized_pnl: Decimal,
    pub maintenance_margin: Decimal,
    pub margin_ratio: Decimal,
    pub liquidation_price: Decimal,
    pub margin_type: MarginType,
}

impl PositionMargin {
    pub fn new(
        symbol: &str,
        side: PositionSide,
        quantity: Decimal,
        entry_price: Decimal,
        mark_price: Decimal,
        leverage: Decimal,
        margin_type: MarginType,
    ) -> Self {
        let notional_value = quantity * mark_price;
        let maintenance_margin = notional_value * dec!(0.005);
        let margin = notional_value / leverage;
        
        let liquidation_price = match side {
            PositionSide::Long => entry_price * (dec!(1) - dec!(1) / leverage),
            PositionSide::Short => entry_price * (dec!(1) + dec!(1) / leverage),
        };
        
        let unrealized_pnl = match side {
            PositionSide::Long => (mark_price - entry_price) * quantity,
            PositionSide::Short => (entry_price - mark_price) * quantity,
        };
        
        Self {
            symbol: symbol.to_string(),
            position_side: side,
            quantity,
            entry_price,
            mark_price,
            leverage,
            margin,
            notional_value,
            unrealized_pnl,
            maintenance_margin,
            margin_ratio: margin / notional_value,
            liquidation_price,
            margin_type,
        }
    }

    pub fn update_mark_price(&mut self, new_mark_price: Decimal) {
        self.mark_price = new_mark_price;
        self.notional_value = self.quantity * new_mark_price;
        
        self.unrealized_pnl = match self.position_side {
            PositionSide::Long => (new_mark_price - self.entry_price) * self.quantity,
            PositionSide::Short => (self.entry_price - new_mark_price) * self.quantity,
        };
        
        self.margin_ratio = self.margin / self.notional_value;
        
        self.liquidation_price = match self.position_side {
            PositionSide::Long => self.entry_price * (dec!(1) - dec!(1) / self.leverage),
            PositionSide::Short => self.entry_price * (dec!(1) + dec!(1) / self.leverage),
        };
    }
}

/// Position side
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum PositionSide {
    Long,
    Short,
}

/// Margin calculation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarginCalc {
    pub order_value: Decimal,
    pub required_margin: Decimal,
    pub available_after: Decimal,
    pub max_leverage: Decimal,
    pub can_execute: bool,
    pub error: Option<String>,
}

impl MarginCalc {
    pub fn calculate(
        account: &MarginAccount,
        order_value: Decimal,
        desired_leverage: Decimal,
    ) -> Self {
        let required_margin = order_value / desired_leverage;
        let available_after = account.available_margin - required_margin;
        let max_leverage = if account.available_margin > dec!(0) {
            order_value / account.available_margin
        } else {
            dec!(1)
        };
        
        Self {
            order_value,
            required_margin,
            available_after,
            max_leverage,
            can_execute: account.available_margin >= required_margin,
            error: if account.available_margin < required_margin {
                Some("Insufficient margin".to_string())
            } else {
                None
            },
        }
    }
}

/// Isolated margin position
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IsolatedMarginPosition {
    pub id: String,
    pub user_id: String,
    pub symbol: String,
    pub side: PositionSide,
    pub quantity: Decimal,
    pub entry_price: Decimal,
    pub margin: Decimal,
    pub leverage: Decimal,
    pub mark_price: Decimal,
    pub unrealized_pnl: Decimal,
    pub liquidation_price: Decimal,
}

impl IsolatedMarginPosition {
    pub fn new(
        user_id: &str,
        symbol: &str,
        side: PositionSide,
        quantity: Decimal,
        entry_price: Decimal,
        margin: Decimal,
        leverage: Decimal,
    ) -> Self {
        let liquidation_price = match side {
            PositionSide::Long => entry_price * (dec!(1) - dec!(1) / leverage),
            PositionSide::Short => entry_price * (dec!(1) + dec!(1) / leverage),
        };
        
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            user_id: user_id.to_string(),
            symbol: symbol.to_string(),
            side,
            quantity,
            entry_price,
            margin,
            leverage,
            mark_price: entry_price,
            unrealized_pnl: dec!(0),
            liquidation_price,
        }
    }

    pub fn update_mark_price(&mut self, new_mark_price: Decimal) {
        self.mark_price = new_mark_price;
        
        self.unrealized_pnl = match self.side {
            PositionSide::Long => (new_mark_price - self.entry_price) * self.quantity,
            PositionSide::Short => (self.entry_price - new_mark_price) * self.quantity,
        };
        
        self.liquidation_price = match self.side {
            PositionSide::Long => self.entry_price * (dec!(1) - dec!(1) / self.leverage),
            PositionSide::Short => self.entry_price * (dec!(1) + dec!(1) / self.leverage),
        };
    }

    pub fn is_liquidatable(&self) -> bool {
        match self.side {
            PositionSide::Long => self.mark_price <= self.liquidation_price,
            PositionSide::Short => self.mark_price >= self.liquidation_price,
        }
    }
}

/// Cross margin portfolio
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CrossMarginPortfolio {
    pub user_id: String,
    pub total_collateral: Decimal,
    pub total_position_value: Decimal,
    pub total_unrealized_pnl: Decimal,
    pub positions: HashMap<String, PositionMargin>,
}

impl CrossMarginPortfolio {
    pub fn new(user_id: &str) -> Self {
        Self {
            user_id: user_id.to_string(),
            total_collateral: dec!(0),
            total_position_value: dec!(0),
            total_unrealized_pnl: dec!(0),
            positions: HashMap::new(),
        }
    }

    pub fn add_position(&mut self, position: PositionMargin) {
        let symbol = position.symbol.clone();
        self.total_position_value += position.notional_value;
        self.total_unrealized_pnl += position.unrealized_pnl;
        self.positions.insert(symbol, position);
    }

    pub fn remove_position(&mut self, symbol: &str) -> Option<PositionMargin> {
        if let Some(position) = self.positions.remove(symbol) {
            self.total_position_value -= position.notional_value;
            self.total_unrealized_pnl -= position.unrealized_pnl;
            Some(position)
        } else {
            None
        }
    }

    pub fn margin_ratio(&self) -> Decimal {
        if self.total_position_value > dec!(0) {
            (self.total_collateral + self.total_unrealized_pnl) / self.total_position_value
        } else {
            dec!(0)
        }
    }

    pub fn available_margin(&self) -> Decimal {
        let available = self.total_collateral + self.total_unrealized_pnl - self.total_position_value * dec!(0.005);
        if available > dec!(0) {
            available
        } else {
            dec!(0)
        }
    }
}

/// Margin engine
pub struct MarginEngine {
    margin_accounts: RwLock<HashMap<String, MarginAccount>>,
    isolated_positions: RwLock<HashMap<String, IsolatedMarginPosition>>,
    cross_portfolios: RwLock<HashMap<String, CrossMarginPortfolio>>,
}

impl MarginEngine {
    pub fn new() -> Self {
        Self {
            margin_accounts: RwLock::new(HashMap::new()),
            isolated_positions: RwLock::new(HashMap::new()),
            cross_portfolios: RwLock::new(HashMap::new()),
        }
    }

    pub fn create_margin_account(&self, user_id: &str) -> MarginAccount {
        let account = MarginAccount::new(user_id);
        let mut accounts = self.margin_accounts.write();
        accounts.insert(user_id.to_string(), account.clone());
        account
    }

    pub fn get_margin_account(&self, user_id: &str) -> Option<MarginAccount> {
        self.margin_accounts.read().get(user_id).cloned()
    }

    pub fn deposit_margin(&self, user_id: &str, amount: Decimal) -> Result<(), MarginError> {
        let mut accounts = self.margin_accounts.write();
        let account = accounts.get_mut(user_id)
            .ok_or_else(|| MarginError::AccountNotFound(user_id.to_string()))?;
        
        account.total_margin += amount;
        account.available_margin += amount;
        
        Ok(())
    }

    pub fn withdraw_margin(&self, user_id: &str, amount: Decimal) -> Result<Decimal, MarginError> {
        let mut accounts = self.margin_accounts.write();
        let account = accounts.get_mut(user_id)
            .ok_or_else(|| MarginError::AccountNotFound(user_id.to_string()))?;
        
        if account.available_margin < amount {
            return Err(MarginError::InsufficientMargin(
                "Insufficient available margin".to_string()
            ));
        }
        
        account.total_margin -= amount;
        account.available_margin -= amount;
        
        Ok(amount)
    }

    pub fn calculate_order_margin(
        &self,
        user_id: &str,
        order_value: Decimal,
        leverage: Decimal,
        margin_type: MarginType,
    ) -> Result<MarginCalc, MarginError> {
        let accounts = self.margin_accounts.read();
        let account = accounts.get(user_id)
            .ok_or_else(|| MarginError::AccountNotFound(user_id.to_string()))?;
        
        Ok(MarginCalc::calculate(account, order_value, leverage))
    }

    pub fn add_position(
        &self,
        user_id: &str,
        symbol: &str,
        side: PositionSide,
        quantity: Decimal,
        entry_price: Decimal,
        leverage: Decimal,
        margin_type: MarginType,
    ) -> Result<(), MarginError> {
        let notional_value = quantity * entry_price;
        let required_margin = notional_value / leverage;
        
        let mut accounts = self.margin_accounts.write();
        let account = accounts.get_mut(user_id)
            .ok_or_else(|| MarginError::AccountNotFound(user_id.to_string()))?;
        
        if account.available_margin < required_margin {
            return Err(MarginError::InsufficientMargin(
                "Insufficient margin for position".to_string()
            ));
        }
        
        account.available_margin -= required_margin;
        account.used_margin += required_margin;
        
        if margin_type == MarginType::Isolated {
            let mut positions = self.isolated_positions.write();
            let position = IsolatedMarginPosition::new(
                user_id,
                symbol,
                side,
                quantity,
                entry_price,
                required_margin,
                leverage,
            );
            positions.insert(format!("{}:{}", user_id, symbol), position);
        }
        
        let position = PositionMargin::new(
            symbol,
            side,
            quantity,
            entry_price,
            entry_price,
            leverage,
            margin_type,
        );
        
        account.positions.insert(symbol.to_string(), position);
        
        Ok(())
    }

    pub fn update_positions(&self, user_id: &str, prices: HashMap<String, Decimal>) {
        let mut accounts = self.margin_accounts.write();
        if let Some(account) = accounts.get_mut(user_id) {
            for (symbol, mark_price) in prices {
                if let Some(position) = account.positions.get_mut(&symbol) {
                    position.update_mark_price(mark_price);
                }
            }
            
            let mut isolated = self.isolated_positions.write();
            for position in isolated.values_mut() {
                if position.user_id == user_id {
                    if let Some(price) = prices.get(&position.symbol) {
                        position.update_mark_price(*price);
                    }
                }
            }
            
            account.update();
        }
    }

    pub fn liquidate_position(&self, user_id: &str, symbol: &str) -> Result<Decimal, MarginError> {
        let mut accounts = self.margin_accounts.write();
        let account = accounts.get_mut(user_id)
            .ok_or_else(|| MarginError::AccountNotFound(user_id.to_string()))?;
        
        if let Some(position) = account.positions.remove(symbol) {
            let mut isolated = self.isolated_positions.write();
            isolated.remove(&format!("{}:{}", user_id, symbol));
            
            account.available_margin += position.margin;
            account.used_margin -= position.margin;
            account.unrealized_pnl += position.unrealized_pnl;
            
            Ok(position.unrealized_pnl)
        } else {
            Err(MarginError::PositionNotFound(symbol.to_string()))
        }
    }
}

impl Default for MarginEngine {
    fn default() -> Self {
        Self::new()
    }
}

/// Margin errors
#[derive(Debug, thiserror::Error)]
pub enum MarginError {
    #[error("Account not found: {0}")]
    AccountNotFound(String),
    
    #[error("Position not found: {0}")]
    PositionNotFound(String),
    
    #[error("Insufficient margin: {0}")]
    InsufficientMargin(String),
    
    #[error("Invalid leverage: {0}")]
    InvalidLeverage(String),
    
    #[error("Margin type mismatch: {0}")]
    MarginTypeMismatch(String),
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_margin_account() {
        let mut account = MarginAccount::new("user1");
        account.available_margin = dec!(10000);
        account.used_margin = dec!(5000);
        
        account.update();
        
        assert_eq!(account.total_margin, dec!(10000));
        assert_eq!(account.status, MarginStatus::Healthy);
    }
    
    #[test]
    fn test_margin_calculation() {
        let account = MarginAccount::new("user1");
        account.available_margin = dec!(1000);
        
        let calc = MarginCalc::calculate(&account, dec!(5000), dec!(10));
        
        assert!(calc.can_execute);
        assert_eq!(calc.required_margin, dec!(500));
    }
}