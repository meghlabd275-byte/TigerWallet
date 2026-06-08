//! TigerWallet Gas Market
use serde::{Deserialize, Serialize};
use rust_decimal::Decimal;
use rust_decimal_macros::dec;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasPrice {
    pub slow: Decimal,
    pub standard: Decimal,
    pub fast: Decimal,
    pub base_fee: Decimal,
    pub priority_fee: Decimal,
}

impl GasPrice {
    pub fn new() -> Self {
        Self {
            slow: dec!(20),
            standard: dec!(30),
            fast: dec!(50),
            base_fee: dec!(10),
            priority_fee: dec!(20),
        }
    }
    
    pub fn estimate(&self, urgency: &str) -> Decimal {
        match urgency {
            "slow" => self.slow,
            "fast" => self.fast,
            _ => self.standard,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasHistory {
    pub timestamp: i64,
    pub gas_price: Decimal,
    pub block_number: u64,
}

impl Default for GasPrice {
    fn default() -> Self {
        Self::new()
    }
}