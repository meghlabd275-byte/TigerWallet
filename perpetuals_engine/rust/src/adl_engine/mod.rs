//! TigerWallet Auto-Deleveraging (ADL) Engine
//! Handles position deleveraging when liquidation fails

use rust_decimal::Decimal;
use rust_decimal_macros::dec;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use parking_lot::RwLock;

/// ADL event
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ADLEvent {
    pub id: String,
    pub position_id: String,
    pub user_id: String,
    pub symbol: String,
    pub quantity: Decimal,
    pub price: Decimal,
    pub leverage: Decimal,
    pub adl_percent: Decimal,
    pub timestamp: i64,
}

/// ADL queue entry
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ADLQueueEntry {
    pub position_id: String,
    pub user_id: String,
    pub symbol: String,
    pub margin_ratio: Decimal,
    pub unrealized_pnl: Decimal,
    pub priority_score: Decimal,
    pub queue_position: u32,
}

/// ADL engine
pub struct ADLEngine {
    queue: RwLock<Vec<ADLQueueEntry>>,
    history: RwLock<Vec<ADLEvent>>,
    max_queue_size: usize,
}

impl ADLEngine {
    pub fn new() -> Self {
        Self {
            queue: RwLock::new(Vec::new()),
            history: RwLock::new(Vec::new()),
            max_queue_size: 1000,
        }
    }

    /// Add position to ADL queue
    pub fn add_to_queue(&self, entry: ADLQueueEntry) {
        let mut queue = self.queue.write();
        
        // Calculate priority score (lower is more urgent)
        let score = self.calculate_priority(&entry);
        let entry = ADLQueueEntry {
            priority_score: score,
            queue_position: queue.len() as u32,
            ..entry
        };
        
        // Insert sorted by priority
        let pos = queue.iter().position(|e| e.priority_score > score);
        match pos {
            Some(p) => queue.insert(p, entry),
            None => queue.push(entry),
        }
        
        // Trim if needed
        if queue.len() > self.max_queue_size {
            queue.truncate(self.max_queue_size);
        }
    }

    /// Calculate ADL priority (lower = higher priority)
    fn calculate_priority(&self, entry: &ADLQueueEntry) -> Decimal {
        // Priority based on margin ratio and P&L
        let margin_score = dec!(1) - entry.margin_ratio;
        let pnl_score = if entry.unrealized_pnl < dec!(0) {
            entry.unrealized_pnl / dec!(10000) // Negative P&L has higher priority
        } else {
            dec!(0)
        };
        
        margin_score + pnl_score.abs()
    }

    /// Execute ADL for a position
    pub fn execute_adl(&self, position_id: &str, adl_percent: Decimal) -> Option<ADLEvent> {
        let mut queue = self.queue.write();
        
        let pos = queue.iter().position(|e| e.position_id == position_id);
        if pos.is_none() {
            return None;
        }
        
        let entry = queue.remove(pos.unwrap());
        
        let event = ADLEvent {
            id: uuid::Uuid::new_v4().to_string(),
            position_id: entry.position_id.clone(),
            user_id: entry.user_id.clone(),
            symbol: entry.symbol.clone(),
            quantity: dec!(0), // Would be calculated from adl_percent
            price: dec!(0),
            leverage: dec!(10),
            adl_percent,
            timestamp: chrono::Utc::now().timestamp(),
        };
        
        drop(queue);
        self.history.write().push(event.clone());
        
        Some(event)
    }

    /// Get ADL queue for a symbol
    pub fn get_queue(&self, symbol: &str) -> Vec<ADLQueueEntry> {
        self.queue.read()
            .iter()
            .filter(|e| e.symbol == symbol)
            .cloned()
            .collect()
    }

    /// Get ADL history
    pub fn get_history(&self, limit: usize) -> Vec<ADLEvent> {
        self.history.read()
            .iter()
            .rev()
            .take(limit)
            .cloned()
            .collect()
    }

    /// Remove position from queue
    pub fn remove_from_queue(&self, position_id: &str) -> bool {
        let mut queue = self.queue.write();
        let pos = queue.iter().position(|e| e.position_id == position_id);
        match pos {
            Some(p) => {
                queue.remove(p);
                true
            }
            None => false,
        }
    }
}

impl Default for ADLEngine {
    fn default() -> Self {
        Self::new()
    }
}