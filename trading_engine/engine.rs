//! TigerWallet Trading Engine - Rust Implementation
//! High-frequency trading with MEV protection

use serde::{Deserialize, Serialize};
use std::collections::{HashMap, VecDeque};
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH};

// ============================================================================
// Order Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum OrderType {
    Market,
    Limit,
    StopLoss,
    StopLossLimit,
    TakeProfit,
    TakeProfitLimit,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum OrderSide {
    Buy,
    Sell,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum OrderStatus {
    Pending,
    Open,
    PartiallyFilled,
    Filled,
    Cancelled,
    Rejected,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum TimeInForce {
    GTC,
    IOC,
    FOK,
    GTD,
}

// ============================================================================
// Order
// ============================================================================

#[derive(Debug, Clone)]
pub struct Order {
    pub id: String,
    pub user_id: String,
    pub market: String,
    pub side: OrderSide,
    pub order_type: OrderType,
    pub quantity: f64,
    pub price: Option<f64>,
    pub filled_quantity: f64,
    pub avg_fill_price: f64,
    pub status: OrderStatus,
    pub time_in_force: TimeInForce,
    pub created_at: u64,
    pub updated_at: u64,
    pub expires_at: Option<u64>,
}

impl Order {
    pub fn new(
        user_id: &str,
        market: &str,
        side: OrderSide,
        order_type: OrderType,
        quantity: f64,
        price: Option<f64>,
    ) -> Self {
        let now = current_timestamp();
        
        Order {
            id: generate_order_id(),
            user_id: user_id.to_string(),
            market: market.to_string(),
            side,
            order_type,
            quantity,
            price,
            filled_quantity: 0.0,
            avg_fill_price: 0.0,
            status: OrderStatus::Pending,
            time_in_force: TimeInForce::GTC,
            created_at: now,
            updated_at: now,
            expires_at: None,
        }
    }

    pub fn fill(&mut self, quantity: f64, price: f64) {
        let total_value = self.avg_fill_price * self.filled_quantity + price * quantity;
        self.filled_quantity += quantity;
        self.avg_fill_price = if self.filled_quantity > 0.0 {
            total_value / self.filled_quantity
        } else {
            price
        };
        
        if self.filled_quantity >= self.quantity {
            self.status = OrderStatus::Filled;
        } else {
            self.status = OrderStatus::PartiallyFilled;
        }
        
        self.updated_at = current_timestamp();
    }

    pub fn cancel(&mut self) {
        self.status = OrderStatus::Cancelled;
        self.updated_at = current_timestamp();
    }
}

// ============================================================================
// Order Book
// ============================================================================

#[derive(Debug)]
pub struct OrderBook {
    pub market: String,
    bids: VecDeque<Order>,
    asks: VecDeque<Order>,
}

impl OrderBook {
    pub fn new(market: &str) -> Self {
        OrderBook {
            market: market.to_string(),
            bids: VecDeque::new(),
            asks: VecDeque::new(),
        }
    }

    pub fn add_order(&mut self, order: Order) -> Result<(), TradeError> {
        match order.side {
            OrderSide::Buy => self.bids.push_back(order),
            OrderSide::Sell => self.asks.push_back(order),
        }
        Ok(())
    }

    pub fn match_orders(&mut self) -> Vec<Trade> {
        let mut trades = Vec::new();
        
        let mut bids: Vec<_> = self.bids.iter().cloned().collect();
        let mut asks: Vec<_> = self.asks.iter().cloned().collect();
        
        bids.sort_by(|a, b| b.price.unwrap_or(0.0).partial_cmp(&a.price.unwrap_or(0.0)).unwrap_or(std::cmp::Ordering::Equal));
        asks.sort_by(|a, b| a.price.unwrap_or(0.0).partial_cmp(&b.price.unwrap_or(0.0)).unwrap_or(std::cmp::Ordering::Equal));
        
        let mut bid_idx = 0;
        let mut ask_idx = 0;
        
        while bid_idx < bids.len() && ask_idx < asks.len() {
            let bid = &bids[bid_idx];
            let ask = &asks[ask_idx];
            
            let bid_price = bid.price.unwrap_or(f64::MAX);
            let ask_price = ask.price.unwrap_or(0.0);
            
            if bid_price >= ask_price {
                let trade_price = ask_price;
                let trade_quantity = bid.quantity.min(ask.quantity);
                
                trades.push(Trade {
                    id: generate_trade_id(),
                    market: self.market.clone(),
                    price: trade_price,
                    quantity: trade_quantity,
                    timestamp: current_timestamp(),
                });
                
                bid_idx += 1;
                ask_idx += 1;
            } else {
                break;
            }
        }
        
        trades
    }
}

// ============================================================================
// Trade
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Trade {
    pub id: String,
    pub market: String,
    pub price: f64,
    pub quantity: f64,
    pub timestamp: u64,
}

// ============================================================================
// Trading Engine
// ============================================================================

pub struct TradingEngine {
    order_books: Arc<RwLock<HashMap<String, OrderBook>>>,
    orders: Arc<RwLock<HashMap<String, Order>>>,
}

impl TradingEngine {
    pub fn new() -> Self {
        TradingEngine {
            order_books: Arc::new(RwLock::new(HashMap::new())),
            orders: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    pub fn create_order(&self, order: Order) -> Result<Order, TradeError> {
        {
            let mut books = self.order_books.write().unwrap();
            let book = books.entry(order.market.clone()).or_insert_with(|| OrderBook::new(&order.market));
            book.add_order(order.clone())?;
        }
        
        {
            let mut orders = self.orders.write().unwrap();
            orders.insert(order.id.clone(), order.clone());
        }
        
        self.try_match(&order.market);
        
        Ok(order)
    }

    fn try_match(&self, market: &str) {
        let mut books = self.order_books.write().unwrap();
        
        if let Some(book) = books.get_mut(market) {
            let trades = book.match_orders();
            
            for trade in trades {
                let mut orders = self.orders.write().unwrap();
                
                // Simplified - would update actual orders
                println!("Trade executed: {} @ {}", trade.quantity, trade.price);
            }
        }
    }
}

// ============================================================================
// Errors
// ============================================================================

#[derive(Debug, thiserror::Error)]
pub enum TradeError {
    #[error("Order not found")]
    OrderNotFound,
    
    #[error("Insufficient balance")]
    InsufficientBalance,
    
    #[error("Invalid order")]
    InvalidOrder,
}

// ============================================================================
// Helpers
// ============================================================================

fn current_timestamp() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
}

fn generate_order_id() -> String {
    format!("ord_{}_{}", current_timestamp(), rand_id(8))
}

fn generate_trade_id() -> String {
    format!("trd_{}_{}", current_timestamp(), rand_id(8))
}

fn rand_id(len: usize) -> String {
    use rand::rngs::OsRng;
    use rand::RngCore;
    
    let mut bytes = vec![0u8; len];
    OsRng.fill_bytes(&mut bytes);
    hex::encode(bytes)
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_order_creation() {
        let order = Order::new(
            "user1",
            "ETH-USDC",
            OrderSide::Buy,
            OrderType::Limit,
            1.0,
            Some(2000.0),
        );
        
        assert_eq!(order.status, OrderStatus::Pending);
    }
}