//! Ultra-Low Latency Trading Core - Rust Implementation
//! Safety-critical trading components with memory safety

use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{SystemTime, UNIX_EPOCH};

// ============================================================================
// Types
// ============================================================================

pub type Timestamp = u64;
pub type OrderID = u64;
pub type UserID = u64;
pub type Price = f64;
pub type Quantity = f64;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum OrderType {
    Market = 0,
    Limit = 1,
    Stop = 2,
    StopLimit = 3,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum OrderSide {
    Buy = 0,
    Sell = 1,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum OrderStatus {
    Pending = 0,
    Open = 1,
    Filled = 2,
    PartiallyFilled = 3,
    Cancelled = 4,
    Rejected = 5,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum MarginMode {
    Cross = 0,
    Isolated = 1,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum PositionSide {
    Long = 0,
    Short = 1,
}

// ============================================================================
// Trading Pair - Cache-aligned for performance
// ============================================================================

#[repr(align(64))]
pub struct TradingPair {
    pub symbol: String,
    pub base: String,
    pub quote: String,
    pub price: Price,
    /// Price 24 hours ago, supplied by the market data feed. The 24h change is
    /// measured against this, not against the previous tick.
    pub reference_price_24h: Price,
    pub high_24h: Price,
    pub low_24h: Price,
    pub volume_24h: Price,
    pub change_24h: f64,
    pub is_pre_installed: bool,
    pub status: bool,
    pub min_order_size: Quantity,
    pub max_order_size: Quantity,
    pub maker_fee: f64,
    pub taker_fee: f64,
}

/// The parameters an operator supplies when registering a market.
///
/// Pairs are never invented: every field here comes from the exchange's
/// configuration, so the engine cannot advertise a market that does not exist.
#[derive(Debug, Clone)]
pub struct TradingPairConfig {
    pub symbol: String,
    pub base: String,
    pub quote: String,
    /// Last traded price, or `None` when the market has not traded yet.
    pub price: Option<Price>,
    pub min_order_size: Quantity,
    pub max_order_size: Quantity,
    pub maker_fee: f64,
    pub taker_fee: f64,
    pub is_pre_installed: bool,
}

impl TradingPairConfig {
    /// Build a configuration with conservative exchange defaults.
    pub fn new(symbol: &str, base: &str, quote: &str) -> Self {
        Self {
            symbol: symbol.to_string(),
            base: base.to_string(),
            quote: quote.to_string(),
            price: None,
            min_order_size: 0.001,
            max_order_size: 1_000_000.0,
            maker_fee: 0.0002,
            taker_fee: 0.0004,
            is_pre_installed: false,
        }
    }

    pub fn with_price(mut self, price: Price) -> Self {
        self.price = Some(price);
        self
    }

    pub fn with_limits(mut self, min: Quantity, max: Quantity) -> Self {
        self.min_order_size = min;
        self.max_order_size = max;
        self
    }

    pub fn with_fees(mut self, maker: f64, taker: f64) -> Self {
        self.maker_fee = maker;
        self.taker_fee = taker;
        self
    }
}

impl TradingPair {
    /// Build a pair from operator configuration.
    ///
    /// The 24h high, low and change start at the observed price rather than a
    /// derived guess: until the feed reports real statistics there is no
    /// honest value for them.
    pub fn from_config(config: TradingPairConfig) -> Self {
        let price = config.price.unwrap_or(0.0);
        Self {
            symbol: config.symbol,
            base: config.base,
            quote: config.quote,
            price,
            reference_price_24h: price,
            high_24h: price,
            low_24h: price,
            volume_24h: 0.0,
            change_24h: 0.0,
            is_pre_installed: config.is_pre_installed,
            status: true,
            min_order_size: config.min_order_size,
            max_order_size: config.max_order_size,
            maker_fee: config.maker_fee,
            taker_fee: config.taker_fee,
        }
    }
}

// ============================================================================
// Order - Thread-safe
// ============================================================================

#[repr(align(64))]
pub struct Order {
    pub id: OrderID,
    pub user_id: UserID,
    pub symbol: String,
    pub side: OrderSide,
    pub order_type: OrderType,
    pub size: Quantity,
    pub filled: Quantity,
    pub price: Price,
    pub stop_price: Price,
    pub leverage: i32,
    pub margin_mode: MarginMode,
    pub status: OrderStatus,
    pub create_time: Timestamp,
    pub update_time: Timestamp,
}

impl Order {
    pub fn new(
        id: OrderID,
        user_id: UserID,
        symbol: String,
        side: OrderSide,
        order_type: OrderType,
        size: Quantity,
        price: Price,
        leverage: i32,
    ) -> Self {
        let now = current_timestamp();
        Self {
            id,
            user_id,
            symbol,
            side,
            order_type,
            size,
            filled: 0.0,
            price,
            stop_price: 0.0,
            leverage,
            margin_mode: MarginMode::Cross,
            status: OrderStatus::Open,
            create_time: now,
            update_time: now,
        }
    }
    
    pub fn is_fulfilled(&self) -> bool {
        self.filled >= self.size
    }
    
    pub fn remaining(&self) -> Quantity {
        self.size - self.filled
    }
}

// ============================================================================
// Position
// ============================================================================

#[repr(align(64))]
pub struct Position {
    pub id: OrderID,
    pub user_id: UserID,
    pub symbol: String,
    pub side: PositionSide,
    pub size: Quantity,
    pub entry_price: Price,
    pub mark_price: Price,
    pub leverage: i32,
    pub margin: Quantity,
    pub margin_mode: MarginMode,
    pub pnl: f64,
    pub pnl_percent: f64,
    pub liquidation_price: Price,
    pub open_time: Timestamp,
}

impl Position {
    pub fn new(
        id: OrderID,
        user_id: UserID,
        symbol: String,
        side: PositionSide,
        size: Quantity,
        entry_price: Price,
        leverage: i32,
        margin: Quantity,
    ) -> Self {
        let liquidation = if side == PositionSide::Long {
            entry_price * (1.0 - 1.0 / leverage as f64)
        } else {
            entry_price * (1.0 + 1.0 / leverage as f64)
        };
        
        Self {
            id,
            user_id,
            symbol,
            side,
            size,
            entry_price,
            mark_price: entry_price,
            leverage,
            margin,
            margin_mode: MarginMode::Cross,
            pnl: 0.0,
            pnl_percent: 0.0,
            liquidation_price: liquidation,
            open_time: current_timestamp(),
        }
    }
    
    pub fn calculate_pnl(&mut self) {
        let price_diff = if self.side == PositionSide::Long {
            self.mark_price - self.entry_price
        } else {
            self.entry_price - self.mark_price
        };
        
        self.pnl = price_diff * self.size;
        self.pnl_percent = (self.pnl / self.margin) * 100.0;
    }
    
    pub fn is_liquidated(&self) -> bool {
        if self.side == PositionSide::Long {
            self.mark_price <= self.liquidation_price
        } else {
            self.mark_price >= self.liquidation_price
        }
    }
}

// ============================================================================
// Trading Engine - Arc-based for thread safety
// ============================================================================

pub struct TradingEngine {
    pairs: std::collections::HashMap<String, TradingPair>,
    orders: std::collections::HashMap<OrderID, Order>,
    positions: std::collections::HashMap<String, Position>,
    next_order_id: AtomicU64,
}

impl TradingEngine {
    pub fn new() -> Self {
        let engine = Self {
            pairs: std::collections::HashMap::new(),
            orders: std::collections::HashMap::new(),
            positions: std::collections::HashMap::new(),
            next_order_id: AtomicU64::new(1),
        };
        engine
    }
    
    /// Register a market from operator configuration.
    ///
    /// The engine starts empty. A pair exists only once an operator has
    /// listed it, which is what makes the venue's market list truthful.
    pub fn add_pair(&mut self, config: TradingPairConfig) -> &TradingPair {
        let symbol = config.symbol.clone();
        self.pairs.insert(symbol.clone(), TradingPair::from_config(config));
        self.pairs.get(&symbol).expect("pair was just inserted")
    }

    /// Remove a market. Open orders for the symbol are cancelled first so no
    /// order is left referencing a pair that no longer exists.
    pub fn remove_pair(&mut self, symbol: &str) -> bool {
        if self.pairs.remove(symbol).is_none() {
            return false;
        }
        for order in self.orders.values_mut() {
            if order.symbol == symbol && order.status == OrderStatus::Open {
                order.status = OrderStatus::Cancelled;
                order.update_time = current_timestamp();
            }
        }
        true
    }

    /// Halt or resume trading on a market without removing it.
    pub fn set_pair_status(&mut self, symbol: &str, status: bool) -> bool {
        match self.pairs.get_mut(symbol) {
            Some(pair) => {
                pair.status = status;
                true
            }
            None => false,
        }
    }
    
    pub fn get_pair(&self, symbol: &str) -> Option<&TradingPair> {
        self.pairs.get(symbol)
    }
    
    pub fn get_pre_installed_pairs(&self) -> Vec<&TradingPair> {
        self.pairs.values().filter(|p| p.is_pre_installed).collect()
    }
    
    pub fn get_total_pairs(&self) -> usize {
        self.pairs.len()
    }
    
    pub fn create_order(
        &mut self,
        user_id: UserID,
        symbol: &str,
        side: OrderSide,
        order_type: OrderType,
        size: Quantity,
        price: Price,
        leverage: i32,
    ) -> Option<OrderID> {
        let pair = self.pairs.get(symbol)?;
        if !pair.status {
            return None;
        }
        if !size.is_finite() || size < pair.min_order_size || size > pair.max_order_size {
            return None;
        }
        if order_type != OrderType::Market && (!price.is_finite() || price <= 0.0) {
            return None;
        }
        if leverage < 1 {
            return None;
        }
        
        let order_id = self.next_order_id.fetch_add(1, Ordering::Relaxed);
        
        let order = Order::new(
            order_id,
            user_id,
            symbol.to_string(),
            side,
            order_type,
            size,
            price,
            leverage,
        );
        
        self.orders.insert(order_id, order);
        Some(order_id)
    }
    
    pub fn cancel_order(&mut self, order_id: OrderID) -> bool {
        if let Some(order) = self.orders.get_mut(&order_id) {
            if order.status == OrderStatus::Open {
                order.status = OrderStatus::Cancelled;
                order.update_time = current_timestamp();
                return true;
            }
        }
        false
    }
    
    pub fn get_order(&self, order_id: OrderID) -> Option<&Order> {
        self.orders.get(&order_id)
    }
    
    pub fn open_position(
        &mut self,
        user_id: UserID,
        symbol: &str,
        side: PositionSide,
        size: Quantity,
        price: Price,
        leverage: i32,
    ) -> Option<OrderID> {
        let margin = (size * price) / leverage as f64;
        
        let position_id = self.next_order_id.fetch_add(1, Ordering::Relaxed);
        
        let position = Position::new(
            position_id,
            user_id,
            symbol.to_string(),
            side,
            size,
            price,
            leverage,
            margin,
        );
        
        let key = format!("{}_{}", user_id, symbol);
        self.positions.insert(key, position);
        Some(position_id)
    }
    
    pub fn get_position(&self, user_id: UserID, symbol: &str) -> Option<&Position> {
        let key = format!("{}_{}", user_id, symbol);
        self.positions.get(&key)
    }
    
    pub fn get_user_positions(&self, user_id: UserID) -> Vec<&Position> {
        self.positions
            .values()
            .filter(|p| p.user_id == user_id)
            .collect()
    }
    
    /// Apply a mark price from the market data feed.
    ///
    /// The 24h change is measured against the price reported 24 hours ago. It
    /// is not the move since the previous tick: that would report a few
    /// seconds of drift as a daily figure.
    pub fn update_prices(&mut self, symbol: &str, new_price: Price) -> bool {
        if !new_price.is_finite() || new_price <= 0.0 {
            return false;
        }

        let mut found = false;
        if let Some(pair) = self.pairs.get_mut(symbol) {
            found = true;
            pair.price = new_price;

            if new_price > pair.high_24h {
                pair.high_24h = new_price;
            }
            if new_price < pair.low_24h {
                pair.low_24h = new_price;
            }

            pair.change_24h = if pair.reference_price_24h > 0.0 {
                ((new_price - pair.reference_price_24h) / pair.reference_price_24h) * 100.0
            } else {
                0.0
            };
        }

        for position in self.positions.values_mut() {
            if position.symbol == symbol {
                position.mark_price = new_price;
                position.calculate_pnl();
            }
        }

        found
    }

    /// Roll the 24h window: the feed supplies the price observed 24h ago and
    /// the high/low extremes over that window.
    pub fn set_24h_statistics(
        &mut self,
        symbol: &str,
        reference_price: Price,
        high: Price,
        low: Price,
        volume: Quantity,
    ) -> bool {
        match self.pairs.get_mut(symbol) {
            Some(pair) => {
                pair.reference_price_24h = reference_price;
                pair.high_24h = high;
                pair.low_24h = low;
                pair.volume_24h = volume;
                pair.change_24h = if reference_price > 0.0 {
                    ((pair.price - reference_price) / reference_price) * 100.0
                } else {
                    0.0
                };
                true
            }
            None => false,
        }
    }
}

// ============================================================================
// Helper Functions
// ============================================================================

#[inline]
pub fn current_timestamp() -> Timestamp {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_nanos() as Timestamp
}

#[inline]
pub fn calculate_required_margin(order_value: Price, leverage: i32) -> Quantity {
    order_value / leverage as f64
}

#[inline]
pub fn calculate_pnl(
    entry_price: Price,
    current_price: Price,
    size: Quantity,
    side: PositionSide,
) -> f64 {
    match side {
        PositionSide::Long => (current_price - entry_price) * size,
        PositionSide::Short => (entry_price - current_price) * size,
    }
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    fn engine_with_btc() -> TradingEngine {
        let mut engine = TradingEngine::new();
        engine.add_pair(TradingPairConfig::new("BTC/USDT", "BTC", "USDT").with_price(43000.0));
        engine
    }

    #[test]
    fn engine_starts_with_no_invented_markets() {
        let engine = TradingEngine::new();
        assert_eq!(engine.get_total_pairs(), 0);
        assert!(engine.get_pre_installed_pairs().is_empty());
    }

    #[test]
    fn add_pair_registers_the_market() {
        let engine = engine_with_btc();
        assert_eq!(engine.get_total_pairs(), 1);
        let pair = engine.get_pair("BTC/USDT").expect("pair must exist");
        assert_eq!(pair.price, 43000.0);
        // Statistics start at the observed price, not a derived guess.
        assert_eq!(pair.high_24h, 43000.0);
        assert_eq!(pair.low_24h, 43000.0);
        assert_eq!(pair.change_24h, 0.0);
    }

    #[test]
    fn pre_installed_filter_reflects_configuration() {
        let mut engine = TradingEngine::new();
        let mut config = TradingPairConfig::new("BTC/USDT", "BTC", "USDT").with_price(43000.0);
        config.is_pre_installed = true;
        engine.add_pair(config);

        assert_eq!(engine.get_pre_installed_pairs().len(), 1);
    }

    #[test]
    fn create_order_rejects_unknown_market() {
        let mut engine = TradingEngine::new();
        let order_id = engine.create_order(
            1, "NOPE/USDT", OrderSide::Buy, OrderType::Limit, 1.0, 10.0, 10,
        );
        assert!(order_id.is_none());
    }

    #[test]
    fn create_order_accepts_a_valid_limit_order() {
        let mut engine = engine_with_btc();
        let order_id = engine.create_order(
            1, "BTC/USDT", OrderSide::Buy, OrderType::Limit, 1.0, 43000.0, 10,
        );
        assert!(order_id.is_some());
        let order = engine.get_order(order_id.unwrap()).expect("order stored");
        assert_eq!(order.size, 1.0);
        assert_eq!(order.filled, 0.0);
        assert_eq!(order.status, OrderStatus::Open);
        assert_eq!(order.remaining(), 1.0);
    }

    #[test]
    fn create_order_rejects_size_outside_market_limits() {
        let mut engine = TradingEngine::new();
        let mut config = TradingPairConfig::new("BTC/USDT", "BTC", "USDT").with_price(43000.0);
        config.min_order_size = 0.01;
        config.max_order_size = 5.0;
        engine.add_pair(config);

        // Below the minimum.
        assert!(engine
            .create_order(1, "BTC/USDT", OrderSide::Buy, OrderType::Limit, 0.001, 43000.0, 10)
            .is_none());
        // Above the maximum.
        assert!(engine
            .create_order(1, "BTC/USDT", OrderSide::Buy, OrderType::Limit, 10.0, 43000.0, 10)
            .is_none());
        // Inside the band.
        assert!(engine
            .create_order(1, "BTC/USDT", OrderSide::Buy, OrderType::Limit, 1.0, 43000.0, 10)
            .is_some());
    }

    #[test]
    fn create_order_rejects_non_positive_price_on_limit_orders() {
        let mut engine = engine_with_btc();
        assert!(engine
            .create_order(1, "BTC/USDT", OrderSide::Buy, OrderType::Limit, 1.0, 0.0, 10)
            .is_none());
        assert!(engine
            .create_order(1, "BTC/USDT", OrderSide::Buy, OrderType::Limit, 1.0, -5.0, 10)
            .is_none());
    }

    #[test]
    fn halted_market_rejects_new_orders() {
        let mut engine = engine_with_btc();
        assert!(engine.set_pair_status("BTC/USDT", false));

        assert!(engine
            .create_order(1, "BTC/USDT", OrderSide::Buy, OrderType::Limit, 1.0, 43000.0, 10)
            .is_none());

        assert!(engine.set_pair_status("BTC/USDT", true));
        assert!(engine
            .create_order(1, "BTC/USDT", OrderSide::Buy, OrderType::Limit, 1.0, 43000.0, 10)
            .is_some());
    }

    #[test]
    fn remove_pair_cancels_its_open_orders() {
        let mut engine = engine_with_btc();
        let order_id = engine
            .create_order(1, "BTC/USDT", OrderSide::Buy, OrderType::Limit, 1.0, 43000.0, 10)
            .expect("order created");

        assert!(engine.remove_pair("BTC/USDT"));
        assert_eq!(engine.get_total_pairs(), 0);

        let order = engine.get_order(order_id).expect("order retained");
        assert_eq!(order.status, OrderStatus::Cancelled);

        assert!(!engine.remove_pair("BTC/USDT"), "second removal must report false");
    }

    #[test]
    fn change_24h_is_measured_against_the_24h_reference() {
        let mut engine = engine_with_btc();

        // The feed reports that BTC traded at 40000 a day ago, with a 24h band.
        assert!(engine.set_24h_statistics("BTC/USDT", 40000.0, 44000.0, 39000.0, 1234.0));

        // A tick to 42000 is +5% against the 24h reference, not against the
        // previous tick.
        assert!(engine.update_prices("BTC/USDT", 42000.0));

        let pair = engine.get_pair("BTC/USDT").unwrap();
        assert_eq!(pair.price, 42000.0);
        assert!((pair.change_24h - 5.0).abs() < 1e-9, "change was {}", pair.change_24h);
        assert_eq!(pair.high_24h, 44000.0);
        assert_eq!(pair.low_24h, 39000.0);
        assert_eq!(pair.volume_24h, 1234.0);
    }

    #[test]
    fn update_prices_rejects_invalid_ticks() {
        let mut engine = engine_with_btc();
        assert!(!engine.update_prices("BTC/USDT", 0.0));
        assert!(!engine.update_prices("BTC/USDT", -1.0));
        assert!(!engine.update_prices("BTC/USDT", f64::NAN));
        assert!(!engine.update_prices("BTC/USDT", f64::INFINITY));
        assert!(!engine.update_prices("UNKNOWN/USDT", 100.0));
        assert_eq!(engine.get_pair("BTC/USDT").unwrap().price, 43000.0);
    }

    #[test]
    fn update_prices_recomputes_open_position_pnl() {
        let mut engine = engine_with_btc();
        engine.open_position(7, "BTC/USDT", PositionSide::Long, 2.0, 43000.0, 10);

        assert!(engine.update_prices("BTC/USDT", 44000.0));

        let position = engine.get_position(7, "BTC/USDT").expect("position exists");
        assert_eq!(position.mark_price, 44000.0);
        // (44000 - 43000) * 2 = 2000
        assert!((position.pnl - 2000.0).abs() < 1e-9, "pnl was {}", position.pnl);
        assert!(!position.is_liquidated());
    }

    #[test]
    fn position_pnl_long_and_short_have_opposite_signs() {
        let mut long = Position::new(
            1, 1, "BTC/USDT".to_string(), PositionSide::Long, 1.0, 43000.0, 10, 4300.0,
        );
        long.mark_price = 44000.0;
        long.calculate_pnl();
        assert!(long.pnl > 0.0);

        let mut short = Position::new(
            2, 1, "BTC/USDT".to_string(), PositionSide::Short, 1.0, 43000.0, 10, 4300.0,
        );
        short.mark_price = 44000.0;
        short.calculate_pnl();
        assert!(short.pnl < 0.0);
    }

    #[test]
    fn liquidation_price_brackets_the_entry() {
        let long = Position::new(
            1, 1, "BTC/USDT".to_string(), PositionSide::Long, 1.0, 43000.0, 10, 4300.0,
        );
        assert!(long.liquidation_price < long.entry_price);

        let short = Position::new(
            2, 1, "BTC/USDT".to_string(), PositionSide::Short, 1.0, 43000.0, 10, 4300.0,
        );
        assert!(short.liquidation_price > short.entry_price);
    }

    #[test]
    fn liquidation_triggers_when_the_mark_crosses_the_price() {
        let mut long = Position::new(
            1, 1, "BTC/USDT".to_string(), PositionSide::Long, 1.0, 43000.0, 10, 4300.0,
        );
        long.mark_price = long.liquidation_price - 1.0;
        assert!(long.is_liquidated());

        let mut short = Position::new(
            2, 1, "BTC/USDT".to_string(), PositionSide::Short, 1.0, 43000.0, 10, 4300.0,
        );
        short.mark_price = short.liquidation_price + 1.0;
        assert!(short.is_liquidated());
    }

    #[test]
    fn cancel_order_only_affects_open_orders() {
        let mut engine = engine_with_btc();
        let order_id = engine
            .create_order(1, "BTC/USDT", OrderSide::Buy, OrderType::Limit, 1.0, 43000.0, 10)
            .expect("order created");

        assert!(engine.cancel_order(order_id));
        assert_eq!(engine.get_order(order_id).unwrap().status, OrderStatus::Cancelled);
        // A second cancel must report failure, not silently succeed.
        assert!(!engine.cancel_order(order_id));
    }

    #[test]
    fn required_margin_divides_by_leverage() {
        assert_eq!(calculate_required_margin(43000.0, 10), 4300.0);
        assert_eq!(calculate_required_margin(1000.0, 1), 1000.0);
    }

    #[test]
    fn standalone_pnl_helper_matches_position_pnl() {
        let long = calculate_pnl(43000.0, 44000.0, 2.0, PositionSide::Long);
        assert_eq!(long, 2000.0);
        let short = calculate_pnl(43000.0, 44000.0, 2.0, PositionSide::Short);
        assert_eq!(short, -2000.0);
    }
}
