//! TigerWallet Intent Routing Service
//! 
//! Intent-based cross-chain trading with solver network integration

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;

/// Intent Types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum IntentType {
    Swap,
    Bridge,
    LimitOrder,
    Arbitrage,
}

impl Default for IntentType {
    fn default() -> Self {
        IntentType::Swap
    }
}

/// Chain identifiers
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum ChainId {
    Ethereum = 1,
    Polygon = 137,
    Arbitrum = 42161,
    Optimism = 10,
    Base = 8453,
    Avalanche = 43114,
    Solana = 1399811149,
    Bitcoin = 0,
}

impl ChainId {
    pub fn from_u64(id: u64) -> Option<Self> {
        match id {
            1 => Some(ChainId::Ethereum),
            137 => Some(ChainId::Polygon),
            42161 => Some(ChainId::Arbitrum),
            10 => Some(ChainId::Optimism),
            8453 => Some(ChainId::Base),
            43114 => Some(ChainId::Avalanche),
            1399811149 => Some(ChainId::Solana),
            0 => Some(ChainId::Bitcoin),
            _ => None,
        }
    }
}

/// Token representation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub chain: ChainId,
    pub address: String,
    pub symbol: String,
    pub decimals: u8,
}

impl Token {
    pub fn ethereum() -> Self {
        Self {
            chain: ChainId::Ethereum,
            address: "0x0000000000000000000000000000000000000000".to_string(),
            symbol: "ETH".to_string(),
            decimals: 18,
        }
    }
    
    pub fn usdc(chain: ChainId) -> Self {
        Self {
            chain,
            address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48".to_string(),
            symbol: "USDC".to_string(),
            decimals: 6,
        }
    }
}

/// Trade intent from user
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Intent {
    pub id: String,
    pub intent_type: IntentType,
    pub owner: String,
    pub from_token: Token,
    pub to_token: Token,
    pub from_amount: u128,
    pub min_out_amount: u128,
    pub to_chain: ChainId,
    pub deadline: u64,
    pub fill_deadline: u64,
    pub nonce: u64,
    pub signature: Vec<u8>,
    pub status: IntentStatus,
    pub created_at: u64,
    pub updated_at: u64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum IntentStatus {
    Open,
    Filled,
    PartiallyFilled,
    Cancelled,
    Expired,
}

impl Default for IntentStatus {
    fn default() -> Self {
        IntentStatus::Open
    }
}

/// Solver solution
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SolverSolution {
    pub solver: String,
    pub intent_id: String,
    pub from_amount: u128,
    pub to_amount: u128,
    pub gas_estimate: u64,
    pub execution_data: Vec<u8>,
    pub valid_until: u64,
    pub fees: u128,
}

/// Solver information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Solver {
    pub id: String,
    pub name: String,
    pub supported_chains: Vec<ChainId>,
    pub fee_bps: u16, // basis points
    pub avg_fill_time_ms: u64,
    pub success_rate: f64,
    pub is_active: bool,
}

/// Quote request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QuoteRequest {
    pub from_token: Token,
    pub to_token: Token,
    pub from_amount: u128,
    pub to_chain: ChainId,
    pub slippage_bps: u16,
}

/// Quote response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QuoteResponse {
    pub from_token: Token,
    pub to_token: Token,
    pub from_amount: u128,
    pub to_amount: u128,
    pub price_impact_bps: i32,
    pub gas_estimate: u64,
    pub route: Vec<RouteStep>,
    pub valid_until: u64,
    pub quotes: Vec<SolverQuote>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RouteStep {
    pub from_token: Token,
    pub to_token: Token,
    pub pool_address: String,
    pub pool_type: PoolType,
    pub from_amount: u128,
    pub to_amount: u128,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum PoolType {
    UniswapV2,
    UniswapV3,
    Curve,
    Balancer,
    Aerodrome,
    Raydium,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SolverQuote {
    pub solver: String,
    pub to_amount: u128,
    pub gas_estimate: u64,
    pub fees: u128,
    pub valid_until: u64,
}

/// Intent Routing Service
pub struct IntentRouter {
    intents: Arc<RwLock<HashMap<String, Intent>>>,
    solvers: Arc<RwLock<HashMap<String, Solver>>>,
    quotes: Arc<RwLock<HashMap<String, QuoteResponse>>>,
}

impl IntentRouter {
    pub fn new() -> Self {
        Self {
            intents: Arc::new(RwLock::new(HashMap::new())),
            solvers: Arc::new(RwLock::new(HashMap::new())),
            quotes: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// Register a new intent
    pub async fn register_intent(&self, intent: Intent) -> Result<(), String> {
        let mut intents = self.intents.write().await;
        intents.insert(intent.id.clone(), intent);
        Ok(())
    }

    /// Get intent by ID
    pub async fn get_intent(&self, id: &str) -> Option<Intent> {
        let intents = self.intents.read().await;
        intents.get(id).cloned()
    }

    /// Cancel intent
    pub async fn cancel_intent(&self, id: &str, owner: &str) -> Result<(), String> {
        let mut intents = self.intents.write().await;
        if let Some(intent) = intents.get_mut(id) {
            if intent.owner != owner {
                return Err("Not authorized".to_string());
            }
            intent.status = IntentStatus::Cancelled;
            Ok(())
        } else {
            Err("Intent not found".to_string())
        }
    }

    /// Register solver
    pub async fn register_solver(&self, solver: Solver) -> Result<(), String> {
        let mut solvers = self.solvers.write().await;
        solvers.insert(solver.id.clone(), solver);
        Ok(())
    }

    /// Get available solvers for a chain
    pub async fn get_solvers(&self, chain: ChainId) -> Vec<Solver> {
        let solvers = self.solvers.read().await;
        solvers
            .values()
            .filter(|s| s.is_active && s.supported_chains.contains(&chain))
            .cloned()
            .collect()
    }

    /// Get quote from solvers
    pub async fn get_quote(&self, request: QuoteRequest) -> Result<QuoteResponse, String> {
        let solvers = self.get_solvers(request.to_chain).await;
        
        if solvers.is_empty() {
            return Err("No solvers available for this route".to_string());
        }

        // In production, would query multiple solvers
        // For demo, return mock quote
        let quote = QuoteResponse {
            from_token: request.from_token.clone(),
            to_token: request.to_token.clone(),
            from_amount: request.from_amount,
            to_amount: request.from_amount * 1000, // Mock calculation
            price_impact_bps: -10,
            gas_estimate: 150000,
            route: vec![],
            valid_until: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs() + 60,
            quotes: solvers
                .iter()
                .map(|s| SolverQuote {
                    solver: s.name.clone(),
                    to_amount: request.from_amount * 1000,
                    gas_estimate: 150000,
                    fees: request.from_amount * s.fee_bps as u128 / 10000,
                    valid_until: std::time::SystemTime::now()
                        .duration_since(std::time::UNIX_EPOCH)
                        .unwrap()
                        .as_secs() + 60,
                })
                .collect(),
        };

        // Cache quote
        let quote_id = format!("{}-{}-{}", request.from_token.address, request.to_token.address, request.from_amount);
        let mut quotes = self.quotes.write().await;
        quotes.insert(quote_id, quote.clone());

        Ok(quote)
    }

    /// Fill intent (called by solver)
    pub async fn fill_intent(&self, intent_id: &str, solution: SolverSolution) -> Result<Intent, String> {
        let mut intents = self.intents.write().await;
        
        if let Some(intent) = intents.get_mut(intent_id) {
            if intent.status != IntentStatus::Open {
                return Err("Intent is not open".to_string());
            }
            
            intent.status = IntentStatus::Filled;
            intent.updated_at = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs();
            
            Ok(intent.clone())
        } else {
            Err("Intent not found".to_string())
        }
    }

    /// Get intents for an owner
    pub async fn get_intents_by_owner(&self, owner: &str) -> Vec<Intent> {
        let intents = self.intents.read().await;
        intents
            .values()
            .filter(|i| i.owner == owner)
            .cloned()
            .collect()
    }
}

impl Default for IntentRouter {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_register_intent() {
        let router = IntentRouter::new();
        
        let intent = Intent {
            id: "test-1".to_string(),
            intent_type: IntentType::Swap,
            owner: "0x123".to_string(),
            from_token: Token::ethereum(),
            to_token: Token::usdc(ChainId::Ethereum),
            from_amount: 1_000_000_000_000_000_000u128,
            min_out_amount: 1500_000_000u128,
            to_chain: ChainId::Ethereum,
            deadline: 0,
            fill_deadline: 0,
            nonce: 0,
            signature: vec![],
            status: IntentStatus::Open,
            created_at: 0,
            updated_at: 0,
        };

        router.register_intent(intent.clone()).await.unwrap();
        
        let retrieved = router.get_intent("test-1").await;
        assert!(retrieved.is_some());
    }

    #[tokio::test]
    async fn test_get_quote() {
        let router = IntentRouter::new();
        
        // Register a solver
        let solver = Solver {
            id: "solver-1".to_string(),
            name: "Test Solver".to_string(),
            supported_chains: vec![ChainId::Ethereum],
            fee_bps: 30,
            avg_fill_time_ms: 5000,
            success_rate: 0.99,
            is_active: true,
        };
        
        router.register_solver(solver).await.unwrap();
        
        let request = QuoteRequest {
            from_token: Token::ethereum(),
            to_token: Token::usdc(ChainId::Ethereum),
            from_amount: 1_000_000_000_000_000_000u128,
            to_chain: ChainId::Ethereum,
            slippage_bps: 50,
        };
        
        let quote = router.get_quote(request).await;
        assert!(quote.is_ok());
    }
}
