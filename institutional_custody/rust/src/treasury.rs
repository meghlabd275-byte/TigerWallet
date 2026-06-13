//! Treasury module for Institutional Custody

use crate::CustodyError;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use chrono::{DateTime, Utc};

/// Treasury account
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TreasuryAccount {
    pub id: String,
    pub name: String,
    pub balance: u64,
    pub currency: String,
    pub created_at: DateTime<Utc>,
    pub last_transaction: Option<DateTime<Utc>>,
}

/// Transaction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TreasuryTransaction {
    pub id: String,
    pub account: String,
    pub transaction_type: TransactionType,
    pub amount: u64,
    pub recipient: Option<String>,
    pub reference: String,
    pub status: TransactionStatus,
    pub approved_by: Vec<String>,
    pub created_at: DateTime<Utc>,
    pub executed_at: Option<DateTime<Utc>>,
}

/// Transaction type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum TransactionType {
    Deposit = 1,
    Withdrawal = 2,
    Transfer = 3,
    Payment = 4,
    Fee = 5,
}

/// Transaction status
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum TransactionStatus {
    Pending = 1,
    Approved = 2,
    Executed = 3,
    Rejected = 4,
    Cancelled = 5,
}

/// Treasury service
pub struct TreasuryService {
    /// Accounts
    accounts: HashMap<String, TreasuryAccount>,
    /// Transactions
    transactions: HashMap<String, TreasuryTransaction>,
    /// Balances
    balances: HashMap<String, u64>,
    /// Minimum balance
    minimum_balance: u64,
}

impl TreasuryService {
    pub fn new() -> Self {
        Self {
            accounts: HashMap::new(),
            transactions: HashMap::new(),
            balances: HashMap::new(),
            minimum_balance: 1000,
        }
    }

    /// Deposit
    pub async fn deposit(&self, amount: u64) -> Result<(), CustodyError> {
        if amount == 0 {
            return Err(CustodyError::TreasuryError("Invalid amount".to_string()));
        }
        
        let balance = self.balances.get(&"main".to_string()).unwrap_or(&0);
        self.balances.insert("main".to_string(), balance + amount);
        
        Ok(())
    }

    /// Withdraw
    pub async fn withdraw(
        &self,
        amount: u64,
        recipient: &str,
    ) -> Result<String, CustodyError> {
        let balance = self.balances.get(&"main".to_string()).unwrap_or(&0);
        
        if *balance < amount {
            return Err(CustodyError::TreasuryError("Insufficient balance".to_string()));
        }
        
        if amount < self.minimum_balance {
            return Err(CustodyError::TreasuryError("Below minimum".to_string()));
        }
        
        let tx_id = uuid::Uuid::new_v4().to_string();
        
        let tx = TreasuryTransaction {
            id: tx_id.clone(),
            account: "main".to_string(),
            transaction_type: TransactionType::Withdrawal,
            amount,
            recipient: Some(recipient.to_string()),
            reference: "".to_string(),
            status: TransactionStatus::Pending,
            approved_by: vec![],
            created_at: Utc::now(),
            executed_at: None,
        };
        
        self.transactions.insert(tx_id.clone(), tx);
        
        Ok(tx_id)
    }

    /// Transfer
    pub async fn transfer(
        &self,
        from: &str,
        to: &str,
        amount: u64,
    ) -> Result<String, CustodyError> {
        let balance = self.balances.get(from).unwrap_or(&0);
        
        if *balance < amount {
            return Err(CustodyError::TreasuryError("Insufficient balance".to_string()));
        }
        
        let tx_id = uuid::Uuid::new_v4().to_string();
        
        let tx = TreasuryTransaction {
            id: tx_id.clone(),
            account: from.to_string(),
            transaction_type: TransactionType::Transfer,
            amount,
            recipient: Some(to.to_string()),
            reference: "".to_string(),
            status: TransactionStatus::Pending,
            approved_by: vec![],
            created_at: Utc::now(),
            executed_at: None,
        };
        
        self.transactions.insert(tx_id.clone(), tx);
        
        Ok(tx_id)
    }

    /// Approve transaction
    pub async fn approve_transaction(
        &self,
        tx_id: &str,
        approver: &str,
    ) -> Result<bool, CustodyError> {
        let tx = self.transactions.get_mut(tx_id)
            .ok_or(CustodyError::TreasuryError("Transaction not found".to_string()))?;
        
        tx.approved_by.push(approver.to_string());
        
        // Would check approval threshold
        Ok(tx.approved_by.len() >= 2)
    }

    /// Execute transaction
    pub async fn execute_transaction(
        &self,
        tx_id: &str,
    ) -> Result<(), CustodyError> {
        let tx = self.transactions.get_mut(tx_id)
            .ok_or(CustodyError::TreasuryError("Transaction not found".to_string()))?;
        
        if tx.status != TransactionStatus::Pending {
            return Err(CustodyError::TreasuryError("Invalid status".to_string()));
        }
        
        tx.status = TransactionStatus::Executed;
        tx.executed_at = Some(Utc::now());
        
        // Update balances
        let balance = self.balances.get(&tx.account).unwrap_or(&0);
        self.balances.insert(tx.account.clone(), balance - tx.amount);
        
        Ok(())
    }

    /// Get balance
    pub async fn get_balance(&self) -> Result<u64, CustodyError> {
        Ok(*self.balances.get(&"main".to_string()).unwrap_or(&0))
    }

    /// Get transaction
    pub async fn get_transaction(
        &self,
        tx_id: &str,
    ) -> Option<TreasuryTransaction> {
        self.transactions.get(tx_id).cloned()
    }

    /// Set minimum balance
    pub async fn set_minimum_balance(&mut self, amount: u64) {
        self.minimum_balance = amount;
    }
}

impl Default for TreasuryService {
    fn default() -> Self {
        Self::new()
    }
}