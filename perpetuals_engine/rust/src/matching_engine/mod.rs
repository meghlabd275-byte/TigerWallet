//! TigerWallet Perpetuals Matching Engine
//! High-performance order matching system for perpetual futures trading
//! Supports USD-margined and Coin-margined contracts with leverage up to 100x

use rust_decimal::Decimal;
use rust_decimal_macros::dec;
use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BinaryHeap, VecDeque};
use std::sync::Arc;
use parking_lot::RwLock;
use uuid::Uuid;
use chrono::{DateTime, Utc};

/// Order side (buy or sell)
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum OrderSide {
    Buy,
    Sell,
}

/// Order type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum OrderType {
    Limit,
    Market,
    StopMarket,
    StopLimit,
    TakeProfit,
    TrailingStop,
}

/// Order status
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum OrderStatus {
    Pending,
    Open,
    PartiallyFilled,
    Filled,
    Cancelled,
    Rejected,
    Expired,
}

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

/// Price tier for fee structure
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum PriceTier {
    Tier1,  // 0-99,999 USD volume
    Tier2,  // 100,000-999,999 USD volume
    Tier3,  // 1,000,000-9,999,999 USD volume
    Tier4,  // 10,000,000-49,999,999 USD volume
    Tier5,  // 50,000,000+ USD volume
}

/// Order direction for priority queue
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum OrderDirection {
    Asc,  // Buy orders: lowest price first
    Desc, // Sell orders: highest price first
}

/// Maximum price deviation from mark price (5% default)
pub const MAX_PRICE_DEVIATION: Decimal = dec!(0.05);

/// Minimum order size
pub const MIN_ORDER_SIZE: Decimal = dec!(0.001);

/// Tick size (minimum price movement)
pub const TICK_SIZE: Decimal = dec!(0.5);

/// Order identification
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrderId(pub String);

impl OrderId {
    pub fn new() -> Self {
        Self(Uuid::new_v4().to_string())
    }
}

/// Perpetual trading pair
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradingPair {
    pub symbol: String,
    pub base_currency: String,
    pub quote_currency: String,
    pub tick_size: Decimal,
    pub min_order_size: Decimal,
    pub max_order_size: Decimal,
    pub max_leverage: Decimal,
    pub maker_fee: Decimal,
    pub taker_fee: Decimal,
    pub maintenance_margin_rate: Decimal,
    pub initial_margin_rate: Decimal,
    pub price_precision: u32,
    pub quantity_precision: u32,
    pub enabled: bool,
    pub perpetual: bool,
    pub funding_rate_interval_hours: u32,
    pub next_funding_time: DateTime<Utc>,
    pub max_price_impact: Decimal,
}

impl TradingPair {
    pub fn new(symbol: &str, base: &str, quote: &str) -> Self {
        Self {
            symbol: symbol.to_string(),
            base_currency: base.to_string(),
            quote_currency: quote.to_string(),
            tick_size: TICK_SIZE,
            min_order_size: MIN_ORDER_SIZE,
            max_order_size: dec!(1000000),
            max_leverage: dec!(100),
            maker_fee: dec!(0.0001),   // 0.01%
            taker_fee: dec!(0.0002),   // 0.02%
            maintenance_margin_rate: dec!(0.005), // 0.5%
            initial_margin_rate: dec!(0.01),   // 1%
            price_precision: 2,
            quantity_precision: 4,
            enabled: true,
            perpetual: true,
            funding_rate_interval_hours: 8,
            next_funding_time: Utc::now(),
            max_price_impact: MAX_PRICE_DEVIATION,
        }
    }

    pub fn validate_price(&self, price: Decimal) -> Result<(), MatchingError> {
        if price <= dec!(0) {
            return Err(MatchingError::InvalidPrice("Price must be positive".to_string()));
        }
        if price % self.tick_size != dec!(0) {
            return Err(MatchingError::InvalidPrice("Price must be multiple of tick size".to_string()));
        }
        Ok(())
    }

    pub fn validate_quantity(&self, quantity: Decimal) -> Result<(), MatchingError> {
        if quantity < self.min_order_size {
            return Err(MatchingError::InvalidQuantity("Quantity below minimum".to_string()));
        }
        if quantity > self.max_order_size {
            return Err(MatchingError::InvalidQuantity("Quantity exceeds maximum".to_string()));
        }
        Ok(())
    }
}

/// Price level in order book
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceLevel {
    pub price: Decimal,
    pub quantity: Decimal,
    pub orders: Vec<String>,
}

/// Order information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Order {
    pub id: String,
    pub user_id: String,
    pub symbol: String,
    pub side: OrderSide,
    pub order_type: OrderType,
    pub price: Decimal,
    pub quantity: Decimal,
    pub filled_quantity: Decimal,
    pub average_fill_price: Decimal,
    pub remaining_quantity: Decimal,
    pub status: OrderStatus,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub expires_at: Option<DateTime<Utc>>,
    pub reduce_only: bool,
    pub post_only: bool,
    pub time_in_force: TimeInForce,
    pub client_order_id: Option<String>,
    pub stop_price: Option<Decimal>,
    pub trailing_distance: Option<Decimal>,
    pub leverage: Decimal,
    pub margin_type: MarginType,
    pub position_side: Option<PositionSide>,
    pub order_idempotency_key: Option<String>,
}

impl Order {
    pub fn new_limit(
        user_id: &str,
        symbol: &str,
        side: OrderSide,
        price: Decimal,
        quantity: Decimal,
    ) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4().to_string(),
            user_id: user_id.to_string(),
            symbol: symbol.to_string(),
            side,
            order_type: OrderType::Limit,
            price,
            quantity,
            filled_quantity: dec!(0),
            average_fill_price: dec!(0),
            remaining_quantity: quantity,
            status: OrderStatus::Open,
            created_at: now,
            updated_at: now,
            expires_at: None,
            reduce_only: false,
            post_only: false,
            time_in_force: TimeInForce::GoodTillCancel,
            client_order_id: None,
            stop_price: None,
            trailing_distance: None,
            leverage: dec!(1),
            margin_type: MarginType::Cross,
            position_side: None,
            order_idempotency_key: None,
        }
    }

    pub fn new_market(
        user_id: &str,
        symbol: &str,
        side: OrderSide,
        quantity: Decimal,
    ) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4().to_string(),
            user_id: user_id.to_string(),
            symbol: symbol.to_string(),
            side,
            order_type: OrderType::Market,
            price: dec!(0),
            quantity,
            filled_quantity: dec!(0),
            average_fill_price: dec!(0),
            remaining_quantity: quantity,
            status: OrderStatus::Open,
            created_at: now,
            updated_at: now,
            expires_at: None,
            reduce_only: false,
            post_only: false,
            time_in_force: TimeInForce::ImmediateOrCancel,
            client_order_id: None,
            stop_price: None,
            trailing_distance: None,
            leverage: dec!(1),
            margin_type: MarginType::Cross,
            position_side: None,
            order_idempotency_key: None,
        }
    }

    pub fn fill(&mut self, fill_price: Decimal, fill_quantity: Decimal) {
        let total_value = self.average_fill_price * self.filled_quantity + fill_price * fill_quantity;
        self.filled_quantity += fill_quantity;
        self.remaining_quantity = self.quantity - self.filled_quantity;
        
        if self.filled_quantity > dec!(0) {
            self.average_fill_price = total_value / self.filled_quantity;
        } else {
            self.average_fill_price = fill_price;
        }
        
        self.updated_at = Utc::now();
        
        if self.remaining_quantity <= dec!(0) {
            self.status = OrderStatus::Filled;
        } else {
            self.status = OrderStatus::PartiallyFilled;
        }
    }

    pub fn is_filled(&self) -> bool {
        self.status == OrderStatus::Filled || self.remaining_quantity <= dec!(0)
    }
}

/// Time in force
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum TimeInForce {
    GoodTillCancel,
    ImmediateOrCancel,
    FillOrKill,
    GoodTillDate,
}

/// Trade execution
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Trade {
    pub id: String,
    pub symbol: String,
    pub side: OrderSide,
    pub price: Decimal,
    pub quantity: Decimal,
    pub maker_order_id: String,
    pub taker_order_id: String,
    pub maker_fee: Decimal,
    pub taker_fee: Decimal,
    pub executed_at: DateTime<Utc>,
    pub matching_engine_fee: Decimal,
    pub liquidity_flag: LiquidityFlag,
}

impl Trade {
    pub fn new(
        symbol: &str,
        side: OrderSide,
        price: Decimal,
        quantity: Decimal,
        maker_order_id: &str,
        taker_order_id: &str,
        maker_fee: Decimal,
        taker_fee: Decimal,
    ) -> Self {
        Self {
            id: Uuid::new_v4().to_string(),
            symbol: symbol.to_string(),
            side,
            price,
            quantity,
            maker_order_id: maker_order_id.to_string(),
            taker_order_id: taker_order_id.to_string(),
            maker_fee,
            taker_fee,
            executed_at: Utc::now(),
            matching_engine_fee: dec!(0.00001), // 0.001%
            liquidity_flag: LiquidityFlag::Added,
        }
    }
}

/// Liquidity flag for fee calculation
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum LiquidityFlag {
    Added,  // Maker (adds liquidity)
    Removed, // Taker (removes liquidity)
}

/// Order book
pub struct OrderBook {
    symbol: String,
    bids: BTreeMap<Decimal, PriceLevel>,  // Buy orders sorted by price (highest first)
    asks: BTreeMap<Decimal, PriceLevel>, // Sell orders sorted by price (lowest first)
    orders: std::collections::HashMap<String, Order>,
    direction: OrderDirection,
}

impl OrderBook {
    pub fn new(symbol: &str) -> Self {
        Self {
            symbol: symbol.to_string(),
            bids: BTreeMap::new(),
            asks: BTreeMap::new(),
            orders: std::collections::HashMap::new(),
            direction: OrderDirection::Asc,
        }
    }

    pub fn add_order(&mut self, mut order: Order) -> Result<Vec<Trade>, MatchingError> {
        let mut trades = Vec::new();
        
        // Handle market orders immediately
        if order.order_type == OrderType::Market {
            return self.execute_market_order(&mut order);
        }
        
        // Validate order
        if order.price <= dec!(0) {
            return Err(MatchingError::InvalidPrice("Price must be positive".to_string()));
        }
        
        // Check for existing order with same idempotency key
        if let Some(ref key) = order.order_idempotency_key {
            if self.orders.values().any(|o| o.client_order_id.as_ref() == Some(key)) {
                return Err(MatchingError::DuplicateOrder("Duplicate order".to_string()));
            }
        }
        
        // Post-only: don't cross the book
        if order.post_only {
            let (cross_price, _) = self.best_price(order.side.opposite());
            if let Some(cross_price) = cross_price {
                if order.side == OrderSide::Buy && order.price >= cross_price {
                    return Err(MatchingError::WouldCross("Would cross book".to_string()));
                }
                if order.side == OrderSide::Sell && order.price <= cross_price {
                    return Err(MatchingError::WouldCross("Would cross book".to_string()));
                }
            }
        }
        
        // Check reduce-only
        if order.reduce_only {
            let position = self.get_user_position(&order.user_id, &order.symbol);
            if order.side == OrderSide::Sell && position.map(|p| p.quantity).unwrap_or(dec!(0)) < order.quantity {
                return Err(MatchingError::InsufficientPosition("Insufficient position to reduce".to_string()));
            }
        }
        
        // Try to match immediately
        let (matches, remaining) = self.try_match_order(&order)?;
        
        for (matched_order_id, fill_price, fill_quantity) in matches {
            let matched_order = self.orders.get_mut(&matched_order_id).unwrap();
            matched_order.fill(fill_price, fill_quantity);
            
            trades.push(Trade::new(
                &self.symbol,
                order.side,
                fill_price,
                fill_quantity,
                &matched_order_id,
                &order.id,
                dec!(0.0001), // Maker fee
                dec!(0.0002), // Taker fee
            ));
            
            order.fill(fill_price, fill_quantity);
            
            if matched_order.is_filled() {
                self.remove_order(&matched_order_id);
            }
        }
        
        // Add remaining to order book if not fully filled
        if !order.is_filled() {
            order.status = OrderStatus::Open;
            self.orders.insert(order.id.clone(), order.clone());
            
            let price_level = self.orders_by_price
                .entry(order.price)
                .or_insert_with(|| PriceLevel {
                    price: order.price,
                    quantity: dec!(0),
                    orders: Vec::new(),
                });
            
            price_level.orders.push(order.id.clone());
            price_level.quantity += order.remaining_quantity;
        }
        
        Ok(trades)
    }

    fn try_match_order(&self, order: &Order) -> Result<Vec<(String, Decimal, Decimal)>, MatchingError> {
        let mut matches = Vec::new();
        let mut remaining = order.quantity;
        
        let opposite_side = order.side.opposite();
        let book_side = if opposite_side == OrderSide::Buy { &self.bids } else { &self.asks };
        
        let (cross_price, cross_level) = book_side.iter()
            .find(|(_, level)| {
                if order.side == OrderSide::Buy {
                    order.price >= level.price
                } else {
                    order.price <= level.price
                }
            })
            .map(|(price, level)| (*price, level))
            .unwrap_or((dec!(0), PriceLevel::default()));
        
        if cross_price <= dec!(0) {
            return Ok(matches);
        }
        
        let fill_price = if order.time_in_force == TimeInForce::FillOrKill {
            cross_price
        } else {
            order.price
        };
        
        for order_id in &cross_level.orders {
            if remaining <= dec!(0) {
                break;
            }
            
            if let Some(matched) = self.orders.get(order_id) {
                let fill_qty = std::cmp::min(remaining, matched.remaining_quantity);
                matches.push((order_id.clone(), fill_price, fill_qty));
                remaining -= fill_qty;
            }
        }
        
        Ok(matches)
    }

    fn execute_market_order(&mut self, order: &mut Order) -> Result<Vec<Trade>, MatchingError> {
        let mut trades = Vec::new();
        
        let best = self.best_price(order.side.opposite());
        if best.0.is_none() {
            return Err(MatchingError::NoLiquidity("No liquidity available".to_string()));
        }
        
        let fill_price = best.0.unwrap();
        
        for (price_level, level) in self.iter_levels(order.side.opposite()) {
            if remaining <= dec!(0) {
                break;
            }
            
            for order_id in &level.orders {
                if let Some(matched) = self.orders.get_mut(order_id) {
                    let fill_qty = std::cmp::min(remaining, matched.remaining_quantity);
                    matched.fill(fill_price, fill_qty);
                    
                    trades.push(Trade::new(
                        &self.symbol,
                        order.side,
                        fill_price,
                        fill_qty,
                        order_id,
                        &order.id,
                        dec!(0.0001),
                        dec!(0.0002),
                    ));
                    
                    remaining -= fill_qty;
                    
                    if matched.is_filled() {
                        self.orders.remove(order_id);
                    }
                }
            }
        }
        
        if remaining > dec!(0) && order.time_in_force == TimeInForce::ImmediateOrCancel {
            return Err(MatchingError::PartialFill(format!("Partially filled: {} remaining", remaining)));
        }
        
        order.filled_quantity = order.quantity - remaining;
        Ok(trades)
    }

    pub fn cancel_order(&mut self, order_id: &str) -> Result<Order, MatchingError> {
        let order = self.orders.remove(order_id)
            .ok_or_else(|| MatchingError::OrderNotFound(format!("Order {} not found", order_id)))?;
        
        // Remove from price level
        let price_level = if order.side == OrderSide::Buy {
            self.bids.get_mut(&order.price)
        } else {
            self.asks.get_mut(&order.price)
        };
        
        if let Some(level) = price_level {
            level.orders.retain(|id| id != order_id);
            level.quantity -= order.remaining_quantity;
        }
        
        Ok(order)
    }

    pub fn get_order(&self, order_id: &str) -> Option<&Order> {
        self.orders.get(order_id)
    }

    pub fn best_price(&self, side: OrderSide) -> (Option<Decimal>, Option<&PriceLevel>) {
        match side {
            OrderSide::Buy => {
                self.bids.iter()
                    .next()
                    .map(|(price, level)| (*price, level))
                    .map(|(p, l)| (Some(p), Some(l)))
                    .unwrap_or((None, None))
            }
            OrderSide::Sell => {
                self.asks.iter()
                    .next()
                    .map(|(price, level)| (*price, level))
                    .map(|(p, l)| (Some(p), Some(l)))
                    .unwrap_or((None, None))
            }
        }
    }

    pub fn depth(&self, levels: usize) -> Vec<(Decimal, Decimal)> {
        let mut result = Vec::new();
        
        for (price, level) in self.bids.iter().take(levels) {
            result.push((*price, level.quantity));
        }
        
        result
    }

    pub fn spread(&self) -> Option<Decimal> {
        let best_bid = self.bids.keys().next_back();
        let best_ask = self.asks.keys().next();
        
        match (best_bid, best_ask) {
            (Some(bid), Some(ask)) => Some(ask - bid),
            _ => None,
        }
    }
}

impl OrderSide {
    pub fn opposite(&self) -> Self {
        match self {
            OrderSide::Buy => OrderSide::Sell,
            OrderSide::Sell => OrderSide::Buy,
        }
    }
}

/// Matching engine errors
#[derive(Debug, thiserror::Error)]
pub enum MatchingError {
    #[error("Invalid price: {0}")]
    InvalidPrice(String),
    
    #[error("Invalid quantity: {0}")]
    InvalidQuantity(String),
    
    #[error("Order not found: {0}")]
    OrderNotFound(String),
    
    #[error("Duplicate order: {0}")]
    DuplicateOrder(String),
    
    #[error("Would cross book: {0}")]
    WouldCross(String),
    
    #[error("No liquidity: {0}")]
    NoLiquidity(String),
    
    #[error("Partial fill: {0}")]
    PartialFill(String),
    
    #[error("Insufficient position: {0}")]
    InsufficientPosition(String),
    
    #[error("Invalid leverage: {0}")]
    InvalidLeverage(String),
    
    #[error("Insufficient margin: {0}")]
    InsufficientMargin(String),
    
    #[error("Trading pair not found: {0}")]
    TradingPairNotFound(String),
    
    #[error("Trading pair disabled: {0}")]
    TradingPairDisabled(String),
}

/// Main matching engine
pub struct MatchingEngine {
    order_books: RwLock<std::collections::HashMap<String, OrderBook>>,
    trading_pairs: RwLock<std::collections::HashMap<String, TradingPair>>,
    trades: RwLock<VecDeque<Trade>>,
    max_trades_history: usize,
}

impl MatchingEngine {
    pub fn new() -> Self {
        Self {
            order_books: RwLock::new(std::collections::HashMap::new()),
            trading_pairs: RwLock::new(std::collections::HashMap::new()),
            trades: RwLock::new(VecDeque::new()),
            max_trades_history: 100000,
        }
    }

    pub fn register_trading_pair(&self, pair: TradingPair) {
        let mut pairs = self.trading_pairs.write();
        pairs.insert(pair.symbol.clone(), pair);
        
        let mut books = self.order_books.write();
        books.insert(pair.symbol.clone(), OrderBook::new(&pair.symbol));
    }

    pub fn add_order(&self, order: Order) -> Result<Vec<Trade>, MatchingError> {
        let pair = self.trading_pairs.read()
            .get(&order.symbol)
            .ok_or_else(|| MatchingError::TradingPairNotFound(order.symbol.clone()))?
            .clone();
        
        if !pair.enabled {
            return Err(MatchingError::TradingPairDisabled(order.symbol.clone()));
        }
        
        // Validate leverage
        if order.leverage > pair.max_leverage {
            return Err(MatchingError::InvalidLeverage(
                format!("Leverage {} exceeds max {}", order.leverage, pair.max_leverage)
            ));
        }
        
        // Validate order
        if order.order_type != OrderType::Market {
            pair.validate_price(order.price)?;
        }
        pair.validate_quantity(order.quantity)?;
        
        let mut books = self.order_books.write();
        let book = books.get_mut(&order.symbol)
            .ok_or_else(|| MatchingError::TradingPairNotFound(order.symbol.clone()))?;
        
        book.add_order(order)
    }

    pub fn cancel_order(&self, symbol: &str, order_id: &str) -> Result<Order, MatchingError> {
        let mut books = self.order_books.write();
        let book = books.get_mut(symbol)
            .ok_or_else(|| MatchingError::TradingPairNotFound(symbol.to_string()))?;
        
        book.cancel_order(order_id)
    }

    pub fn get_order(&self, symbol: &str, order_id: &str) -> Option<Order> {
        let books = self.order_books.read();
        books.get(symbol)?.get_order(order_id).cloned()
    }

    pub fn get_orders(&self, user_id: &str) -> Vec<Order> {
        let books = self.order_books.read();
        let mut orders = Vec::new();
        
        for book in books.values() {
            for order in book.orders.values() {
                if order.user_id == user_id {
                    orders.push(order.clone());
                }
            }
        }
        
        orders
    }

    pub fn get_trades(&self, symbol: &str, limit: usize) -> Vec<Trade> {
        let trades = self.trades.read();
        trades.iter()
            .filter(|t| t.symbol == symbol)
            .take(limit)
            .cloned()
            .collect()
    }

    pub fn get_orderbook(&self, symbol: &str, depth: usize) -> Option<OrderBookSnapshot> {
        let books = self.order_books.read();
        let book = books.get(symbol)?;
        
        Some(OrderBookSnapshot {
            symbol: symbol.to_string(),
            bids: book.bids.iter()
                .take(depth)
                .map(|(p, l)| (*p, l.quantity))
                .collect(),
            asks: book.asks.iter()
                .take(depth)
                .map(|(p, l)| (*p, l.quantity))
                .collect(),
            last_update_id: 0,
        })
    }

    pub fn get_position(&self, user_id: &str, symbol: &str) -> Option<Position> {
        None // Placeholder - will be implemented in position engine
    }

    pub fn get_positions(&self, user_id: &str) -> Vec<Position> {
        Vec::new() // Placeholder
    }

    pub fn get_mark_price(&self, symbol: &str) -> Option<Decimal> {
        let books = self.order_books.read();
        let book = books.get(symbol)?;
        
        let best_bid = book.bids.keys().next_back();
        let best_ask = book.asks.keys().next();
        
        match (best_bid, best_ask) {
            (Some(bid), Some(ask)) => Some((bid + ask) / dec!(2)),
            (Some(p), None) => Some(*p),
            (None, Some(p)) => Some(*p),
            _ => None,
        }
    }
}

impl Default for OrderBook {
    fn default() -> Self {
        Self::new("")
    }
}

impl Default for PriceLevel {
    fn default() -> Self {
        Self {
            price: dec!(0),
            quantity: dec!(0),
            orders: Vec::new(),
        }
    }
}

impl Default for MatchingEngine {
    fn default() -> Self {
        Self::new()
    }
}

/// Order book snapshot for API responses
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrderBookSnapshot {
    pub symbol: String,
    pub bids: Vec<(Decimal, Decimal)>,
    pub asks: Vec<(Decimal, Decimal)>,
    pub last_update_id: u64,
}

/// Position tracking
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
    pub unrealized_pnl: Decimal,
    pub realized_pnl: Decimal,
    pub margin: Decimal,
    pub maintenance_margin: Decimal,
    pub liquidation_price: Decimal,
    pub margin_type: MarginType,
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_order_creation() {
        let order = Order::new_limit("user1", "BTC-USD", OrderSide::Buy, dec!(50000), dec!(1));
        assert_eq!(order.symbol, "BTC-USD");
        assert_eq!(order.side, OrderSide::Buy);
        assert_eq!(order.quantity, dec!(1));
    }
    
    #[test]
    fn test_order_fill() {
        let mut order = Order::new_limit("user1", "BTC-USD", OrderSide::Buy, dec!(50000), dec!(1));
        order.fill(dec!(50000), dec!(0.5));
        assert_eq!(order.filled_quantity, dec!(0.5));
        assert_eq!(order.average_fill_price, dec!(50000));
        assert_eq!(order.status, OrderStatus::PartiallyFilled);
    }
    
    #[test]
    fn test_order_full_fill() {
        let mut order = Order::new_limit("user1", "BTC-USD", OrderSide::Buy, dec!(50000), dec!(1));
        order.fill(dec!(50000), dec!(1));
        assert!(order.is_filled());
        assert_eq!(order.status, OrderStatus::Filled);
    }
}