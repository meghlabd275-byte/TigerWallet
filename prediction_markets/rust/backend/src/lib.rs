/**
 * TigerWallet Prediction Markets - Rust Backend
 * High-performance wallet integration for prediction markets
 */

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use parking_lot::RwLock;
use thiserror::Error;

// ============================================================================
// Error Types
// ============================================================================

#[derive(Error, Debug)]
pub enum PredictionError {
    #[error("Market not found: {0}")]
    MarketNotFound(u64),
    
    #[error("Order not found: {0}")]
    OrderNotFound(u64),
    
    #[error("Insufficient balance: required {0}, available {1}")]
    InsufficientBalance(u64, u64),
    
    #[error("Market is not active")]
    MarketNotActive,
    
    #[error("Market is resolved")]
    MarketResolved,
    
    #[error("Invalid outcome: {0}")]
    InvalidOutcome(u32),
    
    #[error("Order cancelled: {0}")]
    OrderCancelled(u64),
    
    #[error("Order expired")]
    OrderExpired,
    
    #[error("Trading halted for market")]
    TradingHalted,
    
    #[error("Trading paused for outcome")]
    TradingPaused,
    
    #[error("Invalid price: {0}")]
    InvalidPrice(String),
    
    #[error("Invalid amount: {0}")]
    InvalidAmount(String),
    
    #[error("RPC error: {0}")]
    RpcError(String),
    
    #[error("Database error: {0}")]
    DatabaseError(String),
}

impl Serialize for PredictionError {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(&self.to_string())
    }
}

// ============================================================================
// Data Types
// ============================================================================

/// Outcome in a prediction market
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Outcome {
    pub outcome_id: u32,
    pub name: String,
    pub price: u64,
    pub volume: u64,
    pub probability: f64,
}

/// Prediction market
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Market {
    pub market_id: u64,
    pub question: String,
    pub description: String,
    pub category: String,
    pub outcome_type: OutcomeType,
    pub outcomes: Vec<Outcome>,
    pub status: MarketStatus,
    pub resolution_time: u64,
    pub resolved_outcome: Option<u32>,
    pub volume_24h: u64,
    pub total_volume: u64,
    pub featured: bool,
    pub image_url: Option<String>,
    pub created_at: u64,
    pub updated_at: u64,
}

/// User position
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Position {
    pub market_id: u64,
    pub outcome_id: u32,
    pub user_id: u32,
    pub quantity: u64,
    pub avg_price: u64,
    pub invested: u64,
    pub current_value: u64,
    pub profit_loss: i64,
}

/// Order
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Order {
    pub order_id: u64,
    pub market_id: u64,
    pub outcome_id: u32,
    pub user_id: u32,
    pub order_type: OrderType,
    pub side: OrderSide,
    pub price: u64,
    pub amount: u64,
    pub filled_amount: u64,
    pub status: OrderStatus,
    pub timestamp: u64,
    pub expires_at: u64,
}

/// Trade
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Trade {
    pub trade_id: u64,
    pub order_id: u64,
    pub market_id: u64,
    pub outcome_id: u32,
    pub side: OrderSide,
    pub price: u64,
    pub amount: u64,
    pub fees: u64,
    pub timestamp: u64,
    pub user_id: u32,
    pub tx_hash: Option<String>,
}

// ============================================================================
// Enums
// ============================================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum OutcomeType {
    Binary,
    Categorical,
    Scalar,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum MarketStatus {
    Active,
    Paused,
    Resolving,
    Resolved,
    Cancelled,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum OrderType {
    Market,
    Limit,
    StopLoss,
    TakeProfit,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_balance")]
pub enum OrderSide {
    Buy,
    Sell,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum OrderStatus {
    Pending,
    PartiallyFilled,
    Filled,
    Cancelled,
    Expired,
}

// ============================================================================
// Price Oracle
// ============================================================================

pub struct PriceOracle {
    feeds: RwLock<HashMap<String, u64>>,
}

impl PriceOracle {
    pub fn new() -> Self {
        Self {
            feeds: RwLock::new(HashMap::new()),
        }
    }
    
    pub fn update_price(&self, market_id: u64, outcome_id: u32, price: u64) {
        let key = format!("{}_{}", market_id, outcome_id);
        self.feeds.write().insert(key, price);
    }
    
    pub fn get_price(&self, market_id: u64, outcome_id: u32) -> Option<u64> {
        let key = format!("{}_{}", market_id, outcome_id);
        self.feeds.read().get(&key).copied()
    }
    
    pub fn resolve_market(&self, market_id: u64, num_outcomes: u32) -> Option<u32> {
        let feeds = self.feeds.read();
        let mut best_outcome = 0;
        let mut best_price = 0u64;
        
        for outcome_id in 0..num_outcomes {
            let key = format!("{}_{}", market_id, outcome_id);
            if let Some(price) = feeds.get(&key) {
                if *price > best_price {
                    best_price = *price;
                    best_outcome = outcome_id;
                }
            }
        }
        
        if best_price > 0 { Some(best_outcome) } else { None }
    }
}

impl Default for PriceOracle {
    fn default() -> Self { Self::new() }
}

// ============================================================================
// Prediction Market Service
// ============================================================================

pub struct PredictionMarketService {
    markets: RwLock<HashMap<u64, Market>>,
    positions: RwLock<HashMap<(u32, u64, u32), Position>>,
    orders: RwLock<HashMap<u64, Order>>,
    trades: RwLock<HashMap<u32, Vec<Trade>>>,
    balances: RwLock<HashMap<u32, u64>>,
    next_market_id: RwLock<u64>,
    next_order_id: RwLock<u64>,
    next_trade_id: RwLock<u64>,
    price_oracle: Arc<PriceOracle>,
    trading_fee_bps: RwLock<u32>,
}

impl PredictionMarketService {
    pub fn new() -> Self {
        Self {
            markets: RwLock::new(HashMap::new()),
            positions: RwLock::new(HashMap::new()),
            orders: RwLock::new(HashMap::new()),
            trades: RwLock::new(HashMap::new()),
            balances: RwLock::new(HashMap::new()),
            next_market_id: RwLock::new(1),
            next_order_id: RwLock::new(1),
            next_trade_id: RwLock::new(1),
            price_oracle: Arc::new(PriceOracle::new()),
            trading_fee_bps: RwLock::new(30),
        }
    }
    
    pub fn create_market(
        &self,
        question: String,
        description: String,
        outcome_type: OutcomeType,
        outcome_names: Vec<String>,
        resolution_time: u64,
        category: String,
    ) -> Result<Market, PredictionError> {
        let market_id = { *self.next_market_id.write() += 1; market_id };
        
        let outcomes: Vec<Outcome> = outcome_names
            .iter()
            .enumerate()
            .map(|(i, name)| Outcome {
                outcome_id: i as u32,
                name: name.clone(),
                price: 500_000,
                volume: 0,
                probability: 1.0 / outcome_names.len() as f64,
            })
            .collect();
        
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH).unwrap()
            .as_millis() as u64;
        
        let market = Market {
            market_id,
            question,
            description,
            category,
            outcome_type,
            outcomes,
            status: MarketStatus::Active,
            resolution_time,
            resolved_outcome: None,
            volume_24h: 0,
            total_volume: 0,
            featured: false,
            image_url: None,
            created_at: now,
            updated_at: now,
        };
        
        self.markets.write().insert(market_id, market.clone());
        Ok(market)
    }
    
    pub fn get_market(&self, market_id: u64) -> Option<Market> {
        self.markets.read().get(&market_id).cloned()
    }
    
    pub fn get_markets(
        &self,
        status: Option<MarketStatus>,
        category: Option<String>,
        offset: u32,
        limit: u32,
    ) -> Vec<Market> {
        self.markets.read()
            .values()
            .filter(|m| {
                if let Some(s) = status { if m.status != s { return false; } }
                if let Some(ref c) = category { if &m.category != c { return false; } }
                true
            })
            .skip(offset as usize)
            .take(limit as usize)
            .cloned()
            .collect()
    }
    
    pub fn place_order(
        &self,
        user_id: u32,
        market_id: u64,
        outcome_id: u32,
        order_type: OrderType,
        side: OrderSide,
        price: u64,
        amount: u64,
        expires_at: Option<u64>,
    ) -> Result<Order, PredictionError> {
        let market = self.markets.read()
            .get(&market_id)
            .ok_or(PredictionError::MarketNotFound(market_id))?
            .clone();
        
        if market.status != MarketStatus::Active {
            return Err(PredictionError::MarketNotActive);
        }
        
        if outcome_id >= market.outcomes.len() as u32 {
            return Err(PredictionError::InvalidOutcome(outcome_id));
        }
        
        if price == 0 || price > 1_000_000 {
            return Err(PredictionError::InvalidPrice(format!("Price must be 1-1000000, got {}", price)));
        }
        
        if amount == 0 {
            return Err(PredictionError::InvalidAmount("Amount must be > 0".to_string()));
        }
        
        let total_cost = (amount * price) / 1_000_000;
        let fee = (total_cost * *self.trading_fee_bps.read()) / 10000;
        let total_required = total_cost + fee;
        
        let balance = self.balances.read().get(&user_id).copied().unwrap_or(0);
        if balance < total_required {
            return Err(PredictionError::InsufficientBalance(total_required, balance));
        }
        
        self.balances.write().insert(user_id, balance - total_required);
        
        let order_id = { *self.next_order_id.write() += 1; order_id };
        
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH).unwrap()
            .as_millis() as u64;
        
        let order = Order {
            order_id,
            market_id,
            outcome_id,
            user_id,
            order_type,
            side,
            price,
            amount,
            filled_amount: 0,
            status: OrderStatus::Pending,
            timestamp: now,
            expires_at: expires_at.unwrap_or(now + 86400000),
        };
        
        self.orders.write().insert(order_id, order.clone());
        
        {
            let mut markets = self.markets.write();
            if let Some(m) = markets.get_mut(&market_id) {
                m.total_volume += total_cost;
                m.volume_24h += total_cost;
                m.outcomes[outcome_id as usize].volume += amount;
                m.updated_at = now;
            }
        }
        
        Ok(order)
    }
    
    pub fn fill_order(
        &self,
        order_id: u64,
        fill_amount: u64,
        fill_price: u64,
    ) -> Result<Trade, PredictionError> {
        let order = self.orders.write()
            .get_mut(&order_id)
            .ok_or(PredictionError::OrderNotFound(order_id))?
            .clone();
        
        if order.status == OrderStatus::Cancelled || order.status == OrderStatus::Filled {
            return Err(PredictionError::OrderCancelled(order_id));
        }
        
        let fill_value = (fill_amount * fill_price) / 1_000_000;
        let fee = (fill_value * *self.trading_fee_bps.read()) / 10000;
        
        let trade_id = { *self.next_trade_id.write() += 1; trade_id };
        
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH).unwrap()
            .as_millis() as u64;
        
        let trade = Trade {
            trade_id,
            order_id,
            market_id: order.market_id,
            outcome_id: order.outcome_id,
            side: order.side,
            price: fill_price,
            amount: fill_amount,
            fees: fee,
            timestamp: now,
            user_id: order.user_id,
            tx_hash: None,
        };
        
        {
            let mut orders = self.orders.write();
            if let Some(o) = orders.get_mut(&order_id) {
                o.filled_amount += fill_amount;
                o.status = if o.filled_amount >= o.amount {
                    OrderStatus::Filled
                } else {
                    OrderStatus::PartiallyFilled
                };
            }
        }
        
        self.update_position(order.user_id, order.market_id, order.outcome_id, fill_amount, fill_price);
        
        self.trades.write()
            .entry(order.user_id)
            .or_insert_with(Vec::new)
            .push(trade.clone());
        
        Ok(trade)
    }
    
    fn update_position(&self, user_id: u32, market_id: u64, outcome_id: u32, amount: u64, price: u64) {
        let key = (user_id, market_id, outcome_id);
        let cost = (amount * price) / 1_000_000;
        
        if let Some(pos) = self.positions.write().get_mut(&key) {
            let total_cost = pos.invested + cost;
            let total_quantity = pos.quantity + amount;
            pos.avg_price = if total_quantity > 0 { (total_cost * 1_000_000) / total_quantity } else { 0 };
            pos.quantity = total_quantity;
            pos.invested = total_cost;
        } else {
            self.positions.write().insert(key, Position {
                market_id,
                outcome_id,
                user_id,
                quantity: amount,
                avg_price: price,
                invested: cost,
                current_value: cost,
                profit_loss: 0,
            });
        }
    }
    
    pub fn get_user_positions(&self, user_id: u32) -> Vec<Position> {
        self.positions.read()
            .values()
            .filter(|p| p.user_id == user_id)
            .cloned()
            .collect()
    }
    
    pub fn resolve_market(&self, market_id: u64, outcome_id: u32) -> Result<(), PredictionError> {
        let mut markets = self.markets.write();
        let market = markets.get_mut(&market_id)
            .ok_or(PredictionError::MarketNotFound(market_id))?;
        
        if market.status != MarketStatus::Active {
            return Err(PredictionError::MarketNotActive);
        }
        
        if outcome_id >= market.outcomes.len() as u32 {
            return Err(PredictionError::InvalidOutcome(outcome_id));
        }
        
        market.status = MarketStatus::Resolved;
        market.resolved_outcome = Some(outcome_id);
        
        let resolution_price = market.outcomes[outcome_id as usize].price;
        
        for (_, pos) in self.positions.write().iter_mut() {
            if pos.market_id == market_id {
                pos.current_value = (pos.quantity * resolution_price) / 1_000_000;
                pos.profit_loss = pos.current_value as i64 - pos.invested as i64;
            }
        }
        
        Ok(())
    }
    
    pub fn add_funds(&self, user_id: u32, amount: u64) {
        let current = *self.balances.read().get(&user_id).get_or_insert(&0);
        self.balances.write().insert(user_id, current + amount);
    }
    
    pub fn get_balance(&self, user_id: u32) -> u64 {
        *self.balances.read().get(&user_id).get_or_insert(&0)
    }
    
    pub fn price_oracle(&self) -> Arc<PriceOracle> {
        Arc::clone(&self.price_oracle)
    }
}

impl Default for PredictionMarketService {
    fn default() -> Self { Self::new() }
}
