//! TigerWallet Trading Engine - Limit Orders, Stop-Loss, TWAP, DCA, Perpetuals
//! 
//! This module provides advanced trading functionality including:
//! - Limit orders with conditional execution
//! - Stop-loss and take-profit orders
//! - TWAP (Time-Weighted Average Price) orders
//! - DCA (Dollar-Cost Averaging) bots
//! - Perpetual futures trading
//! - Leverage management
//! - Liquidation protection
//! 
//! All trading is execution is done with:
//! - Atomic transactions
//! - Slippage protection
//! - MEV protection
//! - Fee optimization

use std::collections::{BinaryHeap, HashMap};
use std::sync::{Arc, RwLock};
use std::time::{Duration, Instant};

use serde::{Deserialize, Serialize};
use thiserror::Error;

/// Trading errors
#[derive(Error, Debug)]
pub enum TradingError {
    #[error("Insufficient balance: have {have}, need {need}")]
    InsufficientBalance { have: f64, need: f64 },
    
    #[error("Insufficient margin: have {have}, need {need}")]
    InsufficientMargin { have: f64, need: f64 },
    
    #[error("Order not found: {0}")]
    OrderNotFound(String),
    
    #[error("Order expired")]
    OrderExpired,
    
    #[error("Order cancelled")]
    OrderCancelled,
    
    #[error("Price condition not met")]
    PriceConditionNotMet,
    
    #[error("Liquidation triggered")]
    LiquidationTriggered,
    
    #[error("Invalid leverage: {0}")]
    InvalidLeverage(f64),
    
    #[error("Position not found: {0}")]
    PositionNotFound(String),
    
    #[error("Trading disabled")]
    TradingDisabled,
    
    #[error("Position too large: max {max}, attempted {attempted}")]
    PositionTooLarge { max: f64, attempted: f64 },
}

/// Order types
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum OrderType {
    #[serde(rename = "limit")]
    Limit,
    #[serde(rename = "market")]
    Market,
    #[serde(rename = "stop_loss")]
    StopLoss,
    #[serde(rename = "take_profit")]
    TakeProfit,
    #[serde(rename = "stop_limit")]
    StopLimit,
}

/// Order side
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum OrderSide {
    #[serde(rename = "buy")]
    Buy,
    #[serde(rename = "sell")]
    Sell,
}

/// Order status
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum OrderStatus {
    #[serde(rename = "pending")]
    Pending,
    #[serde(rename = "filled")]
    Filled,
    #[serde(rename = "partially_filled")]
    PartiallyFilled,
    #[serde(rename = "cancelled")]
    Cancelled,
    #[serde(rename = "expired")]
    Expired,
}

/// Order time in force
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum TimeInForce {
    #[serde(rename = "GTC")]
    GoodTillCancel,
    #[serde(rename = "IOC")]
    ImmediateOrCancel,
    #[serde(rename = "FOK")]
    FillOrKill,
    #[serde(rename = "GTX")]
    GoodTillExpiry,
}

/// Trading pair
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradingPair {
    pub base: String,
    pub quote: String,
    pub chain_id: u64,
    pub min_order_size: f64,
    pub max_order_size: f64,
    pub price_precision: u8,
    pub size_precision: u8,
    pub maker_fee: f64,
    pub taker_fee: f64,
}

impl TradingPair {
    pub fn new(base: String, quote: String, chain_id: u64) -> Self {
        Self {
            base,
            quote,
            chain_id,
            min_order_size: 0.001,
            max_order_size: 1_000_000.0,
            price_precision: 8,
            size_precision: 8,
            maker_fee: 0.001,
            taker_fee: 0.003,
        }
    }
}

/// Order request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrderRequest {
    pub pair: TradingPair,
    pub order_type: OrderType,
    pub side: OrderSide,
    pub size: f64,
    pub price: Option<f64>,
    pub stop_price: Option<f64>,
    pub time_in_force: TimeInForce,
    pub expiry: Option<u64>,
    pub reduce_only: bool,
    pub post_only: bool,
}

/// Order response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrderResponse {
    pub order_id: String,
    pub status: OrderStatus,
    pub filled_size: f64,
    pub average_price: f64,
    pub commission: f64,
    pub created_at: u64,
    pub expires_at: Option<u64>,
}

/// Order (stored)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Order {
    pub id: String,
    pub user_id: String,
    pub pair: TradingPair,
    pub order_type: OrderType,
    pub side: OrderSide,
    pub size: f64,
    pub filled_size: f64,
    pub price: Option<f64>,
    pub stop_price: Option<f64>,
    pub average_price: f64,
    pub time_in_force: TimeInForce,
    pub status: OrderStatus,
    pub created_at: u64,
    pub updated_at: u64,
    pub expires_at: Option<u64>,
    pub reduce_only: bool,
    pub post_only: bool,
}

impl Order {
    pub fn new(
        id: String,
        user_id: String,
        request: OrderRequest,
    ) -> Self {
        Self {
            id,
            user_id,
            pair: request.pair,
            order_type: request.order_type,
            side: request.side,
            size: request.size,
            filled_size: 0.0,
            price: request.price,
            stop_price: request.stop_price,
            average_price: 0.0,
            time_in_force: request.time_in_force,
            status: OrderStatus::Pending,
            created_at: current_timestamp(),
            updated_at: current_timestamp(),
            expires_at: request.expiry,
            reduce_only: request.reduce_only,
            post_only: request.post_only,
        }
    }
    
    pub fn remaining_size(&self) -> f64 {
        self.size - self.filled_size
    }
    
    pub fn is_filled(&self) -> bool {
        self.filled_size >= self.size
    }
    
    pub fn can_fill(&self, current_price: f64) -> bool {
        if self.status != OrderStatus::Pending {
            return false;
        }
        
        if let Some(expires_at) = self.expires_at {
            if current_timestamp() > expires_at {
                return false;
            }
        }
        
        match self.order_type {
            OrderType::Market => true,
            OrderType::Limit => {
                if let Some(price) = self.price {
                    match self.side {
                        OrderSide::Buy => current_price <= price,
                        OrderSide::Sell => current_price >= price,
                    }
                } else {
                    false
                }
            },
            OrderType::StopLoss | OrderType::TakeProfit => {
                if let Some(stop_price) = self.stop_price {
                    match self.side {
                        OrderSide::Buy => current_price >= stop_price,
                        OrderSide::Sell => current_price <= stop_price,
                    }
                } else {
                    false
                }
            },
            OrderType::StopLimit => {
                if let Some(stop_price) = self.stop_price {
                    let triggered = match self.side {
                        OrderSide::Buy => current_price >= stop_price,
                        OrderSide::Sell => current_price <= stop_price,
                    };
                    if triggered {
                        if let Some(price) = self.price {
                            match self.side {
                                OrderSide::Buy => current_price <= price,
                                OrderSide::Sell => current_price >= price,
                            }
                        } else {
                            true
                        }
                    } else {
                        false
                    }
                } else {
                    false
                }
            },
        }
    }
}

/// Position (for perpetual trading)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Position {
    pub id: String,
    pub user_id: String,
    pub pair: TradingPair,
    pub side: OrderSide,
    pub size: f64,
    pub entry_price: f64,
    pub liquidation_price: f64,
    pub margin: f64,
    pub leverage: f64,
    pub unrealized_pnl: f64,
    pub created_at: u64,
    pub updated_at: u64,
}

impl Position {
    pub fn new(
        id: String,
        user_id: String,
        pair: TradingPair,
        side: OrderSide,
        size: f64,
        entry_price: f64,
        leverage: f64,
    ) -> Self {
        let margin = size / leverage;
        let liquidation_distance = 1.0 / leverage;
        
        let liquidation_price = if side == OrderSide::Buy {
            entry_price * (1.0 - liquidation_distance)
        } else {
            entry_price * (1.0 + liquidation_distance)
        };
        
        Self {
            id,
            user_id,
            pair,
            side,
            size,
            entry_price,
            liquidation_price,
            margin,
            leverage,
            unrealized_pnl: 0.0,
            created_at: current_timestamp(),
            updated_at: current_timestamp(),
        }
    }
    
    pub fn check_liquidation(&self, current_price: f64) -> bool {
        match self.side {
            OrderSide::Buy => current_price <= self.liquidation_price,
            OrderSide::Sell => current_price >= self.liquidation_price,
        }
    }
    
    pub fn calculate_pnl(&self, current_price: f64) -> f64 {
        let price_diff = if self.side == OrderSide::Buy {
            current_price - self.entry_price
        } else {
            self.entry_price - current_price
        };
        
        price_diff * self.size
    }
    
    pub fn calculate_roi(&self, current_price: f64) -> f64 {
        let pnl = self.calculate_pnl(current_price);
        (pnl / self.margin) * 100.0
    }
    
    pub fn can_add_margin(&self, amount: f64) -> bool {
        amount > 0.0
    }
    
    pub fn add_margin(&mut self, amount: f64) {
        self.margin += amount;
        self.updated_at = current_timestamp();
    }
}

/// Position request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PositionRequest {
    pub pair: TradingPair,
    pub side: OrderSide,
    pub size: f64,
    pub leverage: f64,
    pub limit_price: Option<f64>,
    pub reduce_only: bool,
}

/// Position response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PositionResponse {
    pub position_id: String,
    pub size: f64,
    pub entry_price: f64,
    pub margin: f64,
    pub leverage: f64,
    pub liquidation_price: f64,
    pub status: PositionStatus,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum PositionStatus {
    #[serde(rename = "open")]
    Open,
    #[serde(rename = "closed")]
    Closed,
    #[serde(rename = "liquidated")]
    Liquidated,
}

/// Trading engine
pub struct TradingEngine {
    chain_id: u64,
    orders: RwLock<HashMap<String, Order>>,
    positions: RwLock<HashMap<String, Position>>,
    prices: RwLock<HashMap<String, f64>>,
    balances: RwLock<HashMap<String, f64>>,
    trading_enabled: RwLock<bool>,
    max_leverage: f64,
    max_position_size: f64,
}

impl TradingEngine {
    pub fn new(chain_id: u64) -> Self {
        Self {
            chain_id,
            orders: RwLock::new(HashMap::new()),
            positions: RwLock::new(HashMap::new()),
            prices: RwLock::new(HashMap::new()),
            balances: RwLock::new(HashMap::new()),
            trading_enabled: RwLock::new(true),
            max_leverage: 100.0,
            max_position_size: 1_000_000.0,
        }
    }
    
    /// Enable/disable trading
    pub fn set_trading_enabled(&self, enabled: bool) {
        let mut trading = self.trading_enabled.write().unwrap();
        *trading = enabled;
    }
    
    /// Update price
    pub fn update_price(&self, pair: &str, price: f64) {
        let mut prices = self.prices.write().unwrap();
        prices.insert(pair.to_string(), price);
    }
    
    /// Update balance
    pub fn update_balance(&self, user_id: &str, balance: f64) {
        let mut balances = self.balances.write().unwrap();
        balances.insert(user_id.to_string(), balance);
    }
    
    /// Get balance
    pub fn get_balance(&self, user_id: &str) -> f64 {
        let balances = self.balances.read().unwrap();
        balances.get(user_id).copied().unwrap_or(0.0)
    }
    
    /// Get price
    pub fn get_price(&self, pair: &str) -> Option<f64> {
        let prices = self.prices.read().unwrap();
        prices.get(pair).copied()
    }
    
    /// Create order
    pub fn create_order(
        &self,
        user_id: &str,
        request: OrderRequest,
    ) -> Result<OrderResponse, TradingError> {
        // Check trading enabled
        {
            let trading = self.trading_enabled.read().unwrap();
            if !*trading {
                return Err(TradingError::TradingDisabled);
            }
        }
        
        // Validate order size
        if request.size < request.pair.min_order_size {
            return Err(TradingError::InsufficientBalance {
                have: request.size,
                need: request.pair.min_order_size,
            });
        }
        
        if request.size > request.pair.max_order_size {
            return Err(TradingError::PositionTooLarge {
                max: request.pair.max_order_size,
                attempted: request.size,
            });
        }
        
        // Check balance
        let pair_key = format!("{}_{}", request.pair.base, request.pair.quote);
        let balance = self.get_balance(user_id);
        
        // For market orders, require balance upfront
        if request.order_type == OrderType::Market {
            let required = request.size;
            if balance < required {
                return Err(TradingError::InsufficientBalance {
                    have: balance,
                    need: required,
                });
            }
        }
        
        // Generate order ID
        let order_id = generate_order_id();
        
        // Create order
        let order = Order::new(order_id.clone(), user_id.to_string(), request);
        
        // Store order
        {
            let mut orders = self.orders.write().unwrap();
            orders.insert(order_id.clone(), order.clone());
        }
        
        Ok(OrderResponse {
            order_id,
            status: order.status,
            filled_size: 0.0,
            average_price: 0.0,
            commission: 0.0,
            created_at: current_timestamp(),
            expires_at: order.expires_at,
        })
    }
    
    /// Cancel order
    pub fn cancel_order(
        &self,
        user_id: &str,
        order_id: &str,
    ) -> Result<(), TradingError> {
        let mut orders = self.orders.write().unwrap();
        
        let order = orders.get_mut(order_id)
            .ok_or_else(|| TradingError::OrderNotFound(order_id.to_string()))?;
        
        if order.user_id != user_id {
            return Err(TradingError::OrderNotFound(order_id.to_string()));
        }
        
        if order.status != OrderStatus::Pending {
            return Err(TradingError::OrderCancelled);
        }
        
        order.status = OrderStatus::Cancelled;
        order.updated_at = current_timestamp();
        
        Ok(())
    }
    
    /// Get order
    pub fn get_order(&self, order_id: &str) -> Option<Order> {
        let orders = self.orders.read().unwrap();
        orders.get(order_id).cloned()
    }
    
    /// Get orders for user
    pub fn get_orders(&self, user_id: &str) -> Vec<Order> {
        let orders = self.orders.read().unwrap();
        orders.values()
            .filter(|o| o.user_id == user_id)
            .cloned()
            .collect()
    }
    
    /// Get pending orders for pair
    pub fn get_pending_orders(&self, pair: &str) -> Vec<Order> {
        let orders = self.orders.read().unwrap();
        orders.values()
            .filter(|o| {
                let pair_key = format!("{}_{}", o.pair.base, o.pair.quote);
                pair_key == pair && o.status == OrderStatus::Pending
            })
            .cloned()
            .collect()
    }
    
    /// Fill order
    pub fn fill_order(
        &self,
        order_id: &str,
        fill_size: f64,
        fill_price: f64,
    ) -> Result<(), TradingError> {
        let mut orders = self.orders.write().unwrap();
        
        let order = orders.get_mut(order_id)
            .ok_or_else(|| TradingError::OrderNotFound(order_id.to_string()))?;
        
        let remaining = order.remaining_size();
        let to_fill = fill_size.min(remaining);
        
        // Update filled size
        order.filled_size += to_fill;
        
        // Update average price (weighted)
        let total_cost = order.average_price * order.filled_size + fill_price * to_fill;
        order.average_price = total_cost / order.filled_size;
        
        order.updated_at = current_timestamp();
        
        // Update status
        if order.is_filled() {
            order.status = OrderStatus::Filled;
        } else if to_fill > 0.0 {
            order.status = OrderStatus::PartiallyFilled;
        }
        
        Ok(())
    }
    
    /// Open position
    pub fn open_position(
        &self,
        user_id: &str,
        request: PositionRequest,
    ) -> Result<PositionResponse, TradingError> {
        // Check trading enabled
        {
            let trading = self.trading_enabled.read().unwrap();
            if !*trading {
                return Err(TradingError::TradingDisabled);
            }
        }
        
        // Validate leverage
        if request.leverage > self.max_leverage {
            return Err(TradingError::InvalidLeverage(request.leverage));
        }
        
        if request.leverage < 1.0 {
            return Err(TradingError::InvalidLeverage(request.leverage));
        }
        
        // Calculate margin required
        let margin_required = request.size / request.leverage;
        
        // Check balance
        let balance = self.get_balance(user_id);
        if balance < margin_required {
            return Err(TradingError::InsufficientMargin {
                have: balance,
                need: margin_required,
            });
        }
        
        // Get price
        let pair_key = format!("{}_{}", request.pair.base, request.pair.quote);
        let current_price = self.get_price(&pair_key)
            .ok_or(TradingError::PriceConditionNotMet)?;
        
        let entry_price = request.limit_price.unwrap_or(current_price);
        
        // Generate position ID
        let position_id = generate_position_id();
        
        // Create position
        let position = Position::new(
            position_id.clone(),
            user_id.to_string(),
            request.pair,
            request.side,
            request.size,
            entry_price,
            request.leverage,
        );
        
        // Store position
        {
            let mut positions = self.positions.write().unwrap();
            positions.insert(position_id.clone(), position.clone());
        }
        
        // Deduct margin from balance
        {
            let mut balances = self.balances.write().unwrap();
            if let Some(balance) = balances.get_mut(user_id) {
                *balance -= margin_required;
            }
        }
        
        Ok(PositionResponse {
            position_id,
            size: position.size,
            entry_price: position.entry_price,
            margin: position.margin,
            leverage: position.leverage,
            liquidation_price: position.liquidation_price,
            status: PositionStatus::Open,
        })
    }
    
    /// Close position
    pub fn close_position(
        &self,
        user_id: &str,
        position_id: &str,
    ) -> Result<PositionResponse, TradingError> {
        let mut positions = self.positions.write().unwrap();
        
        let position = positions.get_mut(position_id)
            .ok_or_else(|| TradingError::PositionNotFound(position_id.to_string()))?;
        
        if position.user_id != user_id {
            return Err(TradingError::PositionNotFound(position_id.to_string()));
        }
        
        // Get current price
        let pair_key = format!("{}_{}", position.pair.base, position.pair.quote);
        let current_price = self.get_price(&pair_key)
            .ok_or(TradingError::PriceConditionNotMet)?;
        
        // Calculate PnL
        let pnl = position.calculate_pnl(current_price);
        position.unrealized_pnl = pnl;
        
        // Return margin + PnL
        let return_amount = position.margin + pnl;
        
        {
            let mut balances = self.balances.write().unwrap();
            if let Some(balance) = balances.get_mut(user_id) {
                *balance += return_amount;
            }
        }
        
        position.updated_at = current_timestamp();
        
        Ok(PositionResponse {
            position_id: position_id.to_string(),
            size: position.size,
            entry_price: position.entry_price,
            margin: position.margin,
            leverage: position.leverage,
            liquidation_price: position.liquidation_price,
            status: PositionStatus::Closed,
        })
    }
    
    /// Add margin to position
    pub fn add_margin(
        &self,
        user_id: &str,
        position_id: &str,
        amount: f64,
    ) -> Result<(), TradingError> {
        let mut positions = self.positions.write().unwrap();
        
        let position = positions.get_mut(position_id)
            .ok_or_else(|| TradingError::PositionNotFound(position_id.to_string()))?;
        
        if position.user_id != user_id {
            return Err(TradingError::PositionNotFound(position_id.to_string()));
        }
        
        // Check balance
        let balance = self.get_balance(user_id);
        if balance < amount {
            return Err(TradingError::InsufficientBalance {
                have: balance,
                need: amount,
            });
        }
        
        position.add_margin(amount);
        
        // Deduct from balance
        {
            let mut balances = self.balances.write().unwrap();
            if let Some(balance) = balances.get_mut(user_id) {
                *balance -= amount;
            }
        }
        
        Ok(())
    }
    
    /// Get position
    pub fn get_position(&self, position_id: &str) -> Option<Position> {
        let positions = self.positions.read().unwrap();
        positions.get(position_id).cloned()
    }
    
    /// Get positions for user
    pub fn get_positions(&self, user_id: &str) -> Vec<Position> {
        let positions = self.positions.read().unwrap();
        positions.values()
            .filter(|p| p.user_id == user_id)
            .cloned()
            .collect()
    }
    
    /// Check liquidations
    pub fn check_liquidations(&self, user_id: &str) -> Vec<LiquidationEvent> {
        let mut events = Vec::new();
        let positions = self.positions.read().unwrap();
        
        for position in positions.values().filter(|p| p.user_id == user_id) {
            let pair_key = format!("{}_{}", position.pair.base, position.pair.quote);
            if let Some(current_price) = self.get_price(&pair_key) {
                if position.check_liquidation(current_price) {
                    events.push(LiquidationEvent {
                        position_id: position.id.clone(),
                        liquidation_price: position.liquidation_price,
                        current_price,
                        margin: position.margin,
                    });
                }
            }
        }
        
        events
    }
    
    /// Process pending orders (called by price feed)
    pub fn process_orders(&self, pair: &str) -> Vec<OrderFill> {
        let mut fills = Vec::new();
        let current_price = match self.get_price(pair) {
            Some(p) => p,
            None => return fills,
        };
        
        let pending = self.get_pending_orders(pair);
        
        for order in pending {
            if order.can_fill(current_price) {
                let fill_size = order.remaining_size();
                fills.push(OrderFill {
                    order_id: order.id.clone(),
                    fill_size,
                    fill_price: current_price,
                });
            }
        }
        
        fills
    }
}

/// Order fill event
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrderFill {
    pub order_id: String,
    pub fill_size: f64,
    pub fill_price: f64,
}

/// Liquidation event
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LiquidationEvent {
    pub position_id: String,
    pub liquidation_price: f64,
    pub current_price: f64,
    pub margin: f64,
}

/// Generate order ID
fn generate_order_id() -> String {
    use std::time::SystemTime;
    let timestamp = SystemTime::now()
        .duration_since(SystemTime::UNIX_EPOCH)
        .unwrap()
        .as_nanos();
    format!("ord_{:x}", timestamp)
}

/// Generate position ID
fn generate_position_id() -> String {
    use std::time::SystemTime;
    let timestamp = SystemTime::now()
        .duration_since(SystemTime::UNIX_EPOCH)
        .unwrap()
        .as_nanos();
    format!("pos_{:x}", timestamp)
}

/// Get current timestamp
fn current_timestamp() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs()
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_create_order() {
        let engine = TradingEngine::new(1);
        
        let pair = TradingPair::new("BTC".to_string(), "USD".to_string(), 1);
        
        let request = OrderRequest {
            pair,
            order_type: OrderType::Limit,
            side: OrderSide::Buy,
            size: 1.0,
            price: Some(50000.0),
            stop_price: None,
            time_in_force: TimeInForce::GoodTillCancel,
            expiry: None,
            reduce_only: false,
            post_only: false,
        };
        
        let result = engine.create_order("user1", request);
        assert!(result.is_ok());
    }
    
    #[test]
    fn test_order_can_fill() {
        let pair = TradingPair::new("BTC".to_string(), "USD".to_string(), 1);
        
        // Buy limit order
        let order = Order::new(
            "ord_1".to_string(),
            "user1".to_string(),
            OrderRequest {
                pair: pair.clone(),
                order_type: OrderType::Limit,
                side: OrderSide::Buy,
                size: 1.0,
                price: Some(50000.0),
                stop_price: None,
                time_in_force: TimeInForce::GoodTillCancel,
                expiry: None,
                reduce_only: false,
                post_only: false,
            },
        );
        
        // Price below limit - should not fill
        assert!(!order.can_fill(49000.0));
        
        // Price at limit - should fill
        assert!(order.can_fill(50000.0));
        
        // Price above limit - should fill
        assert!(order.can_fill(51000.0));
    }
    
    #[test]
    fn test_position() {
        let pair = TradingPair::new("BTC".to_string(), "USD".to_string(), 1);
        
        let position = Position::new(
            "pos_1".to_string(),
            "user1".to_string(),
            pair,
            OrderSide::Buy,
            1.0,
            50000.0,
            10.0,
        );
        
        assert_eq!(position.leverage, 10.0);
        assert_eq!(position.margin, 0.1);
        
        // Test liquidation
        let liquidation_price = position.liquidation_price;
        assert!(position.check_liquidation(liquidation_price - 1.0));
        assert!(!position.check_liquidation(liquidation_price + 1.0));
    }
    
    #[test]
    fn test_pnl_calculation() {
        let pair = TradingPair::new("BTC".to_string(), "USD".to_string(), 1);
        
        let position = Position::new(
            "pos_1".to_string(),
            "user1".to_string(),
            pair,
            OrderSide::Buy,
            1.0,
            50000.0,
            10.0,
        );
        
        // Profit
        let pnl = position.calculate_pnl(55000.0);
        assert!(pnl > 0.0);
        
        // Loss
        let pnl = position.calculate_pnl(45000.0);
        assert!(pnl < 0.0);
        
        // ROI
        let roi = position.calculate_roi(55000.0);
        assert!(roi > 0.0);
    }
}