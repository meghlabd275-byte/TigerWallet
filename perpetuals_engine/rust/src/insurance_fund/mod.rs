//! TigerWallet Insurance Fund
//! Handles insurance fund for liquidation bankruptcies

use rust_decimal::Decimal;
use rust_decimal_macros::dec;
use serde::{Deserialize, Serialize};
use parking_lot::RwLock;

/// Insurance fund event
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InsuranceFundEvent {
    pub id: String,
    pub event_type: InsuranceEventType,
    pub symbol: String,
    pub amount: Decimal,
    pub balance_before: Decimal,
    pub balance_after: Decimal,
    pub timestamp: i64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum InsuranceEventType {
    LiquidationCovered,
    BankruptcyCovered,
    SocializedLoss,
    FundDeposit,
    FundWithdraw,
}

/// Insurance fund state
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InsuranceFund {
    pub total_balance: Decimal,
    pub total_liquidations: u64,
    pub total_bankruptcies: u64,
    pub last_update: i64,
}

/// Insurance fund manager
pub struct InsuranceFundManager {
    balance: RwLock<Decimal>,
    events: RwLock<Vec<InsuranceFundEvent>>,
    stats: RwLock<InsuranceFundStats>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct InsuranceFundStats {
    pub total_covered: Decimal,
    pub liquidations_covered: u64,
    pub bankruptcies_covered: u64,
}

impl InsuranceFundManager {
    pub fn new() -> Self {
        Self {
            balance: RwLock::new(dec!(1000000)), // Start with 1M insurance
            events: RwLock::new(Vec::new()),
            stats: RwLock::new(InsuranceFundStats::default()),
        }
    }

    /// Get current balance
    pub fn get_balance(&self) -> Decimal {
        *self.balance.read()
    }

    /// Cover liquidation loss
    pub fn cover_liquidation(&self, symbol: &str, amount: Decimal) -> Result<InsuranceFundEvent, InsuranceError> {
        let mut balance = self.balance.write();
        
        if *balance < amount {
            return Err(InsuranceError::InsufficientFunds);
        }
        
        let balance_before = *balance;
        *balance -= amount;
        
        let event = InsuranceFundEvent {
            id: uuid::Uuid::new_v4().to_string(),
            event_type: InsuranceEventType::LiquidationCovered,
            symbol: symbol.to_string(),
            amount,
            balance_before,
            balance_after: *balance,
            timestamp: chrono::Utc::now().timestamp(),
        };
        
        // Update stats
        let mut stats = self.stats.write();
        stats.total_covered += amount;
        stats.liquidations_covered += 1;
        
        drop(balance);
        self.events.write().push(event.clone());
        
        Ok(event)
    }

    /// Cover bankruptcy
    pub fn cover_bankruptcy(&self, symbol: &str, amount: Decimal) -> Result<InsuranceFundEvent, InsuranceError> {
        let mut balance = self.balance.write();
        
        if *balance < amount {
            return Err(InsuranceError::InsufficientFunds);
        }
        
        let balance_before = *balance;
        *balance -= amount;
        
        let event = InsuranceFundEvent {
            id: uuid::Uuid::new_v4().to_string(),
            event_type: InsuranceEventType::BankruptcyCovered,
            symbol: symbol.to_string(),
            amount,
            balance_before,
            balance_after: *balance,
            timestamp: chrono::Utc::now().timestamp(),
        };
        
        let mut stats = self.stats.write();
        stats.total_covered += amount;
        stats.bankruptcies_covered += 1;
        
        drop(balance);
        self.events.write().push(event.clone());
        
        Ok(event)
    }

    /// Deposit to insurance fund
    pub fn deposit(&self, amount: Decimal) -> InsuranceFundEvent {
        let mut balance = self.balance.write();
        let balance_before = *balance;
        *balance += amount;
        
        let event = InsuranceFundEvent {
            id: uuid::Uuid::new_v4().to_string(),
            event_type: InsuranceEventType::FundDeposit,
            symbol: "ALL".to_string(),
            amount,
            balance_before,
            balance_after: *balance,
            timestamp: chrono::Utc::now().timestamp(),
        };
        
        drop(balance);
        self.events.write().push(event.clone());
        
        event
    }

    /// Withdraw from insurance fund
    pub fn withdraw(&self, amount: Decimal) -> Result<InsuranceFundEvent, InsuranceError> {
        let mut balance = self.balance.write();
        
        if *balance < amount {
            return Err(InsuranceError::InsufficientFunds);
        }
        
        let balance_before = *balance;
        *balance -= amount;
        
        let event = InsuranceFundEvent {
            id: uuid::Uuid::new_v4().to_string(),
            event_type: InsuranceEventType::FundWithdraw,
            symbol: "ALL".to_string(),
            amount,
            balance_before,
            balance_after: *balance,
            timestamp: chrono::Utc::now().timestamp(),
        };
        
        drop(balance);
        self.events.write().push(event.clone());
        
        Ok(event)
    }

    /// Get events
    pub fn get_events(&self, limit: usize) -> Vec<InsuranceFundEvent> {
        self.events.read()
            .iter()
            .rev()
            .take(limit)
            .cloned()
            .collect()
    }

    /// Get stats
    pub fn get_stats(&self) -> InsuranceFund {
        let balance = *self.balance.read();
        let stats = self.stats.read();
        
        InsuranceFund {
            total_balance: balance,
            total_liquidations: stats.liquidations_covered,
            total_bankruptcies: stats.bankruptcies_covered,
            last_update: chrono::Utc::now().timestamp(),
        }
    }
}

impl Default for InsuranceFundManager {
    fn default() -> Self {
        Self::new()
    }
}

#[derive(Debug, thiserror::Error)]
pub enum InsuranceError {
    #[error("Insufficient insurance funds")]
    InsufficientFunds,
    
    #[error("Invalid amount")]
    InvalidAmount,
}