//! TigerWallet Intent Routing Service
//! 
//! UniswapX/CoW Swap style intent execution with solver network

use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use serde::{Deserialize, Serialize};
use uuid::Uuid;
use chrono::{DateTime, Utc};

mod solver;
mod orderbook;

pub use solver::*;
pub use orderbook::*;

/// Intent routing service
pub struct IntentRoutingService {
    /// Solver service
    solver: Arc<RwLock<SolverService>>,
    /// Order book
    orderbook: Arc<RwLock<OrderBookService>>,
    /// Chain ID
    chain_id: u64,
    /// Contract address
    contract_address: Option<String>,
}

impl IntentRoutingService {
    /// Create new service
    pub fn new(chain_id: u64) -> Self {
        Self {
            solver: Arc::new(RwLock::new(SolverService::new())),
            orderbook: Arc::new(RwLock::new(OrderBookService::new())),
            chain_id,
            contract_address: None,
        }
    }

    /// Initialize
    pub async fn initialize(&mut self, contract_address: String) {
        self.contract_address = Some(contract_address);
    }

    /// Create intent
    pub async fn create_intent(
        &self,
        tokens_in: Vec<String>,
        tokens_out: Vec<String>,
        amounts_in: Vec<u64>,
        amounts_out_min: Vec<u64>,
        prices: Vec<u64>,
        expiry: u64,
    ) -> Result<String, IntentError> {
        if tokens_in.len() != tokens_out.len() {
            return Err(IntentError::InvalidParams);
        }

        let intent = IntentData {
            tokens_in,
            tokens_out,
            amounts_in,
            amounts_out_min,
            prices,
            expiry,
            filled: false,
        };

        let intent_id = Uuid::new_v4().to_string();
        Ok(intent_id)
    }

    /// Fill intent
    pub async fn fill_intent(
        &self,
        intent_id: &str,
        fill_amount: u64,
    ) -> Result<FillResult, IntentError> {
        let solver = self.solver.read().await;
        
        // Find best solver
        let best_solver = solver.find_best_solver(fill_amount).await?;
        
        // Execute fill
        Ok(FillResult {
            intent_id: intent_id.to_string(),
            solver: best_solver,
            amount_out: fill_amount * 1000, // Simplified
        })
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
        let orderbook = self.orderbook.read().await;
        orderbook.create_order(
            sell_token,
            buy_token,
            sell_amount,
            buy_amount,
            deadline,
        ).await
    }

    /// Fill order
    pub async fn fill_order(
        &self,
        order_id: &str,
        fill_amount: u64,
    ) -> Result<FillResult, IntentError> {
        let orderbook = self.orderbook.read().await;
        orderbook.fill_order(order_id, fill_amount).await
    }

    /// Get orders
    pub async fn get_orders(
        &self,
        sell_token: &str,
        buy_token: &str,
    ) -> Result<Vec<OrderInfo>, IntentError> {
        let orderbook = self.orderbook.read().await;
        orderbook.get_orders(sell_token, buy_token).await
    }

    /// Register solver
    pub async fn register_solver(&self, stake: u64) -> Result<(), IntentError> {
        let mut solver = self.solver.write().await;
        solver.register(stake).await
    }

    /// Deregister solver
    pub async fn deregister_solver(&self) -> Result<(), IntentError> {
        let mut solver = self.solver.write().await;
        solver.deregister().await
    }

    /// Get solvers
    pub async fn get_solvers(&self) -> Result<Vec<SolverInfo>, IntentError> {
        let solver = self.solver.read().await;
        solver.get_solvers().await
    }
}

/// Intent data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IntentData {
    pub tokens_in: Vec<String>,
    pub tokens_out: Vec<String>,
    pub amounts_in: Vec<u64>,
    pub amounts_out_min: Vec<u64>,
    pub prices: Vec<u64>,
    pub expiry: u64,
    pub filled: bool,
}

/// Fill result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FillResult {
    pub intent_id: String,
    pub solver: String,
    pub amount_out: u64,
}

/// Intent error
#[derive(Debug, thiserror::Error)]
pub enum IntentError {
    #[error("Invalid params")]
    InvalidParams,
    
    #[error("Intent not found")]
    IntentNotFound,
    
    #[error("Order not found")]
    OrderNotFound,
    
    #[error("Solver error: {0}")]
    SolverError(String),
    
    #[error("Order book error: {0}")]
    OrderBookError(String),
}

use thiserror;