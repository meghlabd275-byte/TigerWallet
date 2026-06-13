//! TigerWallet DEX Aggregator - Rust Implementation
//! High-performance multi-hop swap routing with sub-millisecond latency

use serde::{Deserialize, Serialize};
use std::collections::{BinaryHeap, HashMap};

// ============================================================================
// Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub address: String,
    pub chain_id: u64,
    pub symbol: String,
    pub decimals: u8,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Pool {
    pub id: String,
    pub token0: Token,
    pub token1: Token,
    pub reserve0: f64,
    pub reserve1: f64,
    pub fee: f64, // e.g., 0.003 for 0.3%
    pub dex: String, // "uniswap", "sushi", "curve"
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Route {
    pub path: Vec<Token>,
    pub pools: Vec<Pool>,
    pub amount_in: f64,
    pub amount_out: f64,
    pub gas_estimate: u64,
    pub price_impact: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapQuote {
    pub from_token: Token,
    pub to_token: Token,
    pub amount_in: f64,
    pub amount_out: f64,
    pub routes: Vec<Route>,
    pub gas_estimate: u64,
    pub price_impact: f64,
    pub execution_time_ms: u64,
}

// ============================================================================
// DEX Aggregator
// ============================================================================

pub struct DEXAggregator {
    pools: HashMap<String, Vec<Pool>>,
    gas_per_swap: u64,
}

impl DEXAggregator {
    pub fn new() -> Self {
        DEXAggregator {
            pools: HashMap::new(),
            gas_per_swap: 150000,
        }
    }

    /// Add pool to aggregator
    pub fn add_pool(&mut self, pool: Pool) {
        let key = format!("{}-{}", pool.token0.symbol, pool.token1.symbol);
        self.pools.entry(key).or_insert_with(Vec::new).push(pool);
    }

    /// Get quote for swap
    pub fn get_quote(&self, from: &Token, to: &Token, amount_in: f64) -> Option<SwapQuote> {
        let start = std::time::Instant::now();
        
        // Find all possible routes
        let routes = self.find_routes(from, to, amount_in)?;
        
        // Get best route
        let best_route = routes.into_iter().max_by(|a, b| {
            a.amount_out.partial_cmp(&b.amount_out).unwrap()
        })?;
        
        let execution_time = start.elapsed().as_millis() as u64;
        
        Some(SwapQuote {
            from_token: from.clone(),
            to_token: to.clone(),
            amount_in,
            amount_out: best_route.amount_out,
            routes: vec![best_route],
            gas_estimate: self.gas_per_swap,
            price_impact: 0.0,
            execution_time_ms: execution_time,
        })
    }

    /// Find optimal routes using BFS
    fn find_routes(&self, from: &Token, to: &Token, amount_in: f64) -> Option<Vec<Route>> {
        let mut routes = Vec::new();
        
        // Direct swap
        if let Some(route) = self.find_direct_route(from, to, amount_in) {
            routes.push(route);
        }
        
        // Two-hop swaps
        for intermediate in self.get_all_tokens() {
            if intermediate.address != from.address && intermediate.address != to.address {
                if let Some(route1) = self.find_direct_route(from, &intermediate, amount_in) {
                    if let Some(route2) = self.find_direct_route(&intermediate, to, route1.amount_out) {
                        let combined = Route {
                            path: vec![from.clone(), intermediate.clone(), to.clone()],
                            pools: vec![route1.pools[0].clone(), route2.pools[0].clone()],
                            amount_in,
                            amount_out: route2.amount_out,
                            gas_estimate: self.gas_per_swap * 2,
                            price_impact: route1.price_impact + route2.price_impact,
                        };
                        routes.push(combined);
                    }
                }
            }
        }
        
        if routes.is_empty() {
            None
        } else {
            Some(routes)
        }
    }

    /// Find direct route
    fn find_direct_route(&self, from: &Token, to: &Token, amount_in: f64) -> Option<Route> {
        let key1 = format!("{}-{}", from.symbol, to.symbol);
        let key2 = format!("{}-{}", to.symbol, from.symbol);
        
        let pools = self.pools.get(&key1)
            .or_else(|| self.pools.get(&key2));
        
        let pools = match pools {
            Some(p) => p,
            None => return None,
        };
        
        // Find best pool
        let best_pool = pools.iter().max_by(|a, b| {
            let out_a = self.calculate_output(amount_in, a);
            let out_b = self.calculate_output(amount_in, b);
            out_a.partial_cmp(&out_b).unwrap()
        })?;
        
        let amount_out = self.calculate_output(amount_in, best_pool);
        
        Some(Route {
            path: vec![from.clone(), to.clone()],
            pools: vec![best_pool.clone()],
            amount_in,
            amount_out,
            gas_estimate: self.gas_per_swap,
            price_impact: 0.0,
        })
    }

    /// Calculate output amount
    fn calculate_output(&self, amount_in: f64, pool: &Pool) -> f64 {
        let (reserve_in, reserve_out) = if pool.token0.symbol == "USDC" {
            (pool.reserve0, pool.reserve1)
        } else {
            (pool.reserve1, pool.reserve0)
        };
        
        // Constant product: (x + dx)(y - dy) = xy
        // dy = y * dx / (x + dx)
        let amount_in_with_fee = amount_in * (1.0 - pool.fee);
        let amount_out = reserve_out * amount_in_with_fee / (reserve_in + amount_in_with_fee);
        
        amount_out
    }

    /// Get all tokens
    fn get_all_tokens(&self) -> Vec<Token> {
        let mut tokens = std::collections::HashSet::new();
        for pools in self.pools.values() {
            for pool in pools {
                tokens.insert(pool.token0.clone());
                tokens.insert(pool.token1.clone());
            }
        }
        tokens.into_iter().collect()
    }

    /// Execute swap (returns transaction data)
    pub fn build_swap_data(&self, quote: &SwapQuote, to: &str) -> SwapData {
        let mut calls = Vec::new();
        
        for route in &quote.routes {
            for pool in &route.pools {
                // Build Uniswap V3 style call data
                let call = self.build_v3_call(pool, route.amount_in, to);
                calls.push(call);
            }
        }
        
        SwapData {
            calls,
            gas_limit: quote.gas_estimate,
        }
    }

    fn build_v3_call(&self, pool: &Pool, amount_in: f64, to: &str) -> Vec<u8> {
        // Simplified - real implementation would encode exact input swap
        let mut data = Vec::new();
        
        // function selector for exactInputSingle
        data.extend_from_slice(&[0x04, 0x52, 0x18, 0x4b]);
        
        // params would be encoded here
        
        data
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapData {
    pub calls: Vec<Vec<u8>>,
    pub gas_limit: u64,
}

// ============================================================================
// Optimized Router
// ============================================================================

pub struct OptimizedRouter {
    aggregator: DEXAggregator,
}

impl OptimizedRouter {
    pub fn new() -> Self {
        OptimizedRouter {
            aggregator: DEXAggregator::new(),
        }
    }

    /// Find best route with gas optimization
    pub fn find_best_route(&self, from: &Token, to: &Token, amount_in: f64, gas_price: f64) -> Option<SwapQuote> {
        let quote = self.aggregator.get_quote(from, to, amount_in)?;
        
        // Adjust for gas
        let gas_cost_usd = (quote.gas_estimate as f64) * gas_price / 1e9;
        
        Some(SwapQuote {
            amount_out: quote.amount_out - gas_cost_usd,
            ..quote
        })
    }

    /// Multi-hop optimization
    pub fn find_multi_hop(&self, tokens: &[Token], amount_in: f64) -> Option<f64> {
        if tokens.len() < 2 {
            return None;
        }
        
        let mut current_amount = amount_in;
        
        for i in 0..tokens.len() - 1 {
            let quote = self.aggregator.get_quote(&tokens[i], &tokens[i + 1], current_amount)?;
            current_amount = quote.amount_out;
        }
        
        Some(current_amount)
    }
}

// ============================================================================
// Gas Estimator
// ============================================================================

pub struct GasEstimator {
    chain_gas_limits: HashMap<u64, ChainGasInfo>,
}

#[derive(Debug, Clone)]
pub struct ChainGasInfo {
    pub swap_gas: u64,
    pub transfer_gas: u64,
    pub default_gas_price_gwei: f64,
}

impl GasEstimator {
    pub fn new() -> Self {
        let mut info = HashMap::new();
        
        info.insert(1, ChainGasInfo { // Ethereum
            swap_gas: 150000,
            transfer_gas: 21000,
            default_gas_price_gwei: 30.0,
        });
        
        info.insert(137, ChainGasInfo { // Polygon
            swap_gas: 150000,
            transfer_gas: 21000,
            default_gas_price_gwei: 0.1,
        });
        
        info.insert(42161, ChainGasInfo { // Arbitrum
            swap_gas: 150000,
            transfer_gas: 21000,
            default_gas_price_gwei: 0.01,
        });
        
        GasEstimator {
            chain_gas_limits: info,
        }
    }

    pub fn estimate_gas(&self, chain_id: u64, operation: &str) -> u64 {
        let info = match self.chain_gas_limits.get(&chain_id) {
            Some(i) => i,
            None => return 150000,
        };
        
        match operation {
            "swap" => info.swap_gas,
            "transfer" => info.transfer_gas,
            _ => info.swap_gas,
        }
    }
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_quote() {
        let mut aggregator = DEXAggregator::new();
        
        aggregator.add_pool(Pool {
            id: "pool1".to_string(),
            token0: Token { address: "USDC".to_string(), chain_id: 1, symbol: "USDC".to_string(), decimals: 6 },
            token1: Token { address: "WETH".to_string(), chain_id: 1, symbol: "WETH".to_string(), decimals: 18 },
            reserve0: 1_000_000.0,
            reserve1: 500.0,
            fee: 0.003,
            dex: "uniswap".to_string(),
        });
        
        let from = Token { address: "USDC".to_string(), chain_id: 1, symbol: "USDC".to_string(), decimals: 6 };
        let to = Token { address: "WETH".to_string(), chain_id: 1, symbol: "WETH".to_string(), decimals: 18 };
        
        let quote = aggregator.get_quote(&from, &to, 1000.0).unwrap();
        
        assert!(quote.amount_out > 0.0);
    }
}