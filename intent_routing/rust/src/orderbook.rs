//! Order book service for intent routing

use crate::{OrderInfo, FillResult, IntentError};
use std::collections::HashMap;
use tokio::sync::RwLock;
use chrono::Utc;

/// Order book service
pub struct OrderBookService {
    /// Orders
    orders: HashMap<String, OrderInfo>,
    /// Order book (sell_token -> buy_token -> order_ids)
    order_book: HashMap<String, HashMap<String, Vec<String>>>,
}

impl OrderBookService {
    pub fn new() -> Self {
        Self {
            orders: HashMap::new(),
            order_book: HashMap::new(),
        }
    }

    /// Create order
    pub async fn create_order(
        &self,
        sell_token: &str,
        buy_token: &str,
        sell_amount: u64,
        buy_amount: u64,
        deadline: u64,
    ) -> Result<String, IntentError> {
        let order_id = uuid::Uuid::new_v4().to_string();
        
        let order = OrderInfo {
            order_id: order_id.clone(),
            sell_token: sell_token.to_string(),
            buy_token: buy_token.to_string(),
            sell_amount,
            buy_amount,
            filled_amount: 0,
            created_at: Utc::now(),
            deadline,
        };
        
        self.orders.insert(order_id.clone(), order);
        
        // Add to order book
        let pair_key = format!("{}_{}", sell_token, buy_token);
        self.order_book
            .entry(pair_key)
            .or_insert_with(Vec::new)
            .push(order_id.clone());
        
        Ok(order_id)
    }

    /// Fill order
    pub async fn fill_order(
        &self,
        order_id: &str,
        fill_amount: u64,
    ) -> Result<FillResult, IntentError> {
        let order = self.orders.get(order_id)
            .ok_or(IntentError::OrderNotFound)?;
        
        let amount_out = (fill_amount * order.buy_amount) / order.sell_amount;
        
        Ok(FillResult {
            intent_id: order_id.to_string(),
            solver: "0x0".to_string(),
            amount_out,
        })
    }

    /// Get orders
    pub async fn get_orders(
        &self,
        sell_token: &str,
        buy_token: &str,
    ) -> Result<Vec<OrderInfo>, IntentError> {
        let pair_key = format!("{}_{}", sell_token, buy_token);
        
        if let Some(order_ids) = self.order_book.get(&pair_key) {
            let mut result = vec![];
            for id in order_ids {
                if let Some(order) = self.orders.get(id) {
                    result.push(order.clone());
                }
            }
            Ok(result)
        } else {
            Ok(vec![])
        }
    }
}

impl Default for OrderBookService {
    fn default() -> Self {
        Self::new()
    }
}