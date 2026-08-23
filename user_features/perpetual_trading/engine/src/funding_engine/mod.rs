//! TigerWallet Funding Engine
//! Handles periodic funding rate payments for perpetual contracts

use rust_decimal::Decimal;
use rust_decimal_macros::dec;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use parking_lot::RwLock;
use chrono::{DateTime, Utc, Duration};

/// Funding rate component
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FundingRate {
    pub symbol: String,
    pub rate: Decimal,
    pub next_funding_time: DateTime<Utc>,
    pub previous_funding_time: DateTime<Utc>,
    pub avg_mark_price: Decimal,
    pub avg_index_price: Decimal,
    pub funding_indicator: i8, // -1, 0, 1
}

impl FundingRate {
    pub fn new(symbol: &str) -> Self {
        let now = Utc::now();
        Self {
            symbol: symbol.to_string(),
            rate: dec!(0),
            next_funding_time: now + Duration::hours(8),
            previous_funding_time: now - Duration::hours(8),
            avg_mark_price: dec!(0),
            avg_index_price: dec!(0),
            funding_indicator: 0,
        }
    }

    pub fn calculate(&mut self, mark_price: Decimal, index_price: Decimal) {
        let price_diff = mark_price - index_price;
        let price_ratio = if index_price > dec!(0) {
            price_diff / index_price
        } else {
            dec!(0)
        };
        
        // Clamp to max 0.75% (hourly)
        let clamped_rate = price_ratio.max(dec!(-0.00075)).min(dec!(0.00075));
        self.rate = clamped_rate;
        
        self.avg_mark_price = mark_price;
        self.avg_index_price = index_price;
        
        self.funding_indicator = if price_diff > dec!(0) {
            1
        } else if price_diff < dec!(0) {
            -1
        } else {
            0
        };
    }
}

/// Funding payment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FundingPayment {
    pub id: String,
    pub user_id: String,
    pub symbol: String,
    pub side: PositionSide,
    pub quantity: Decimal,
    pub funding_rate: Decimal,
    pub payment: Decimal,
    pub funding_time: DateTime<Utc>,
    pub status: PaymentStatus,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum PaymentStatus {
    Pending,
    Paid,
    Failed,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum PositionSide {
    Long,
    Short,
}

/// Funding history
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FundingHistory {
    pub symbol: String,
    pub funding_rate: Decimal,
    pub mark_price: Decimal,
    pub index_price: Decimal,
    pub timestamp: DateTime<Utc>,
}

/// Funding engine
pub struct FundingEngine {
    rates: RwLock<HashMap<String, FundingRate>>,
    payments: RwLock<Vec<FundingPayment>>,
    history: RwLock<Vec<FundingHistory>>,
    max_history: usize,
}

impl FundingEngine {
    pub fn new() -> Self {
        Self {
            rates: RwLock::new(HashMap::new()),
            payments: RwLock::new(Vec::new()),
            history: RwLock::new(Vec::new()),
            max_history: 10000,
        }
    }

    pub fn register_symbol(&self, symbol: &str) {
        let mut rates = self.rates.write();
        rates.insert(symbol.to_string(), FundingRate::new(symbol));
    }

    pub fn update_prices(&self, symbol: &str, mark_price: Decimal, index_price: Decimal) {
        let mut rates = self.rates.write();
        if let Some(rate) = rates.get_mut(symbol) {
            rate.calculate(mark_price, index_price);
        }
    }

    pub fn get_funding_rate(&self, symbol: &str) -> Option<Decimal> {
        self.rates.read().get(symbol).map(|r| r.rate)
    }

    pub fn calculate_funding_payment(
        &self,
        user_id: &str,
        symbol: &str,
        side: PositionSide,
        quantity: Decimal,
    ) -> Option<FundingPayment> {
        let rates = self.rates.read();
        let rate = rates.get(symbol)?;
        
        let payment = if side == PositionSide::Long {
            rate.rate * quantity * rate.avg_mark_price
        } else {
            -rate.rate * quantity * rate.avg_mark_price
        };
        
        Some(FundingPayment {
            id: uuid::Uuid::new_v4().to_string(),
            user_id: user_id.to_string(),
            symbol: symbol.to_string(),
            side,
            quantity,
            funding_rate: rate.rate,
            payment,
            funding_time: rate.next_funding_time,
            status: PaymentStatus::Pending,
        })
    }

    pub fn process_funding(&self, symbol: &str) -> Vec<FundingPayment> {
        let mut result = Vec::new();
        
        let rates = self.rates.read();
        if let Some(rate) = rates.get(symbol) {
            let mut history = self.history.write();
            
            if history.len() >= self.max_history {
                history.remove(0);
            }
            
            history.push(FundingHistory {
                symbol: symbol.to_string(),
                funding_rate: rate.rate,
                mark_price: rate.avg_mark_price,
                index_price: rate.avg_index_price,
                timestamp: Utc::now(),
            });
        }
        
        result
    }

    pub fn get_funding_history(&self, symbol: &str, limit: usize) -> Vec<FundingHistory> {
        let history = self.history.read();
        history.iter()
            .filter(|h| h.symbol == symbol)
            .take(limit)
            .cloned()
            .collect()
    }

    pub fn get_next_funding_time(&self, symbol: &str) -> Option<DateTime<Utc>> {
        self.rates.read().get(symbol).map(|r| r.next_funding_time)
    }
}

impl Default for FundingEngine {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_funding_rate_calculation() {
        let mut rate = FundingRate::new("BTC-USD");
        rate.calculate(dec!(50000), dec!(49900));
        
        assert!(rate.rate < dec!(0));
        assert_eq!(rate.funding_indicator, -1);
    }
}