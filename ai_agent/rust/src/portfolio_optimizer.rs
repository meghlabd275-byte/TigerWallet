//! Portfolio Optimizer Module
//! 
//! Provides portfolio optimization and rebalancing

use crate::{Portfolio, PortfolioPosition, TokenInfo, AgentConfig, AgentError};
use chrono::Utc;

/// Portfolio Optimizer
pub struct PortfolioOptimizer {
    config: AgentConfig,
}

impl PortfolioOptimizer {
    /// Create new optimizer
    pub fn new(config: AgentConfig) -> Self {
        Self { config }
    }
    
    /// Calculate optimal portfolio allocation
    pub fn calculate_optimal_allocation(
        &self,
        portfolio: &Portfolio,
    ) -> Result<PortfolioAllocation, AgentError> {
        let mut target_allocations = Vec::new();
        
        // Simple equal-weight strategy for demonstration
        // In production, would use more sophisticated optimization
        let target_per_token = 100.0 / portfolio.positions.len() as f64;
        
        for position in &portfolio.positions {
            let current_value = position.value_usd;
            let target_value = portfolio.total_value_usd * (target_per_token / 100.0);
            let diff = target_value - current_value;
            
            target_allocations.push(TokenAllocation {
                token: position.token.symbol.clone(),
                current_allocation: position.allocation_percent,
                target_allocation: target_per_token,
                difference_percent: diff / portfolio.total_value_usd * 100.0,
                action: if diff > 0.0 { AllocationAction::Buy } else { AllocationAction::Sell },
                amount_usd: diff.abs(),
            });
        }
        
        Ok(PortfolioAllocation {
            total_value: portfolio.total_value_usd,
            allocations: target_allocations,
            rebalance_needed: self.needs_rebalancing(portfolio),
            estimated_gas: self.estimate_rebalance_gas(portfolio),
        })
    }
    
    /// Check if portfolio needs rebalancing
    fn needs_rebalancing(&self, portfolio: &Portfolio) -> bool {
        let target = 100.0 / portfolio.positions.len() as f64;
        
        for position in &portfolio.positions {
            if (position.allocation_percent - target).abs() > self.config.rebalance_threshold {
                return true;
            }
        }
        
        false
    }
    
    /// Estimate gas for rebalancing
    fn estimate_rebalance_gas(&self, portfolio: &Portfolio) -> u64 {
        // Rough estimate: 65K gas per swap
        let swaps_needed = portfolio.positions.len() / 2;
        65000 * swaps_needed as u64
    }
    
    /// Calculate risk-adjusted returns
    pub fn calculate_risk_adjusted_returns(&self, portfolio: &Portfolio) -> f64 {
        // Simplified Sharpe-like ratio
        let total_return: f64 = portfolio.positions.iter()
            .map(|p| p.pnl_percent)
            .sum();
        
        let avg_allocation: f64 = portfolio.positions.iter()
            .map(|p| p.allocation_percent)
            .sum::<f64>() / portfolio.positions.len() as f64;
        
        // Higher allocation weight * return
        total_return * (avg_allocation / 100.0)
    }
    
    /// Get tax-loss harvesting opportunities
    pub fn find_tax_loss_harvesting(&self, portfolio: &Portfolio) -> Vec<TaxLossHarvestingOpportunity> {
        let mut opportunities = Vec::new();
        
        for position in &portfolio.positions {
            if position.pnl_percent < -10.0 {
                opportunities.push(TaxLossHarvestingOpportunity {
                    token: position.token.symbol.clone(),
                    current_loss_percent: position.pnl_percent.abs(),
                    potential_tax_savings: position.value_usd * 0.1 * (position.pnl_percent.abs() / 100.0),
                    recommendation: format!("Consider selling {} to harvest loss", position.token.symbol),
                });
            }
        }
        
        opportunities
    }
}

#[derive(Debug, Clone)]
pub struct PortfolioAllocation {
    pub total_value: f64,
    pub allocations: Vec<TokenAllocation>,
    pub rebalance_needed: bool,
    pub estimated_gas: u64,
}

#[derive(Debug, Clone)]
pub struct TokenAllocation {
    pub token: String,
    pub current_allocation: f64,
    pub target_allocation: f64,
    pub difference_percent: f64,
    pub action: AllocationAction,
    pub amount_usd: f64,
}

#[derive(Debug, Clone)]
pub enum AllocationAction {
    Buy,
    Sell,
    Hold,
}

#[derive(Debug, Clone)]
pub struct TaxLossHarvestingOpportunity {
    pub token: String,
    pub current_loss_percent: f64,
    pub potential_tax_savings: f64,
    pub recommendation: String,
}

#[cfg(test)]
mod tests {
    use super::*;
    
    fn create_test_portfolio() -> Portfolio {
        Portfolio {
            total_value_usd: 10000.0,
            positions: vec![
                PortfolioPosition {
                    token: TokenInfo {
                        address: "0x123".to_string(),
                        symbol: "ETH".to_string(),
                        name: "Ethereum".to_string(),
                        decimals: 18,
                        chain_id: 1,
                        price_usd: 2000.0,
                        price_change_24h: 2.5,
                        volume_24h: 1000000.0,
                        liquidity: 5000000.0,
                    },
                    balance: 4.0,
                    value_usd: 8000.0,
                    allocation_percent: 80.0,
                    pnl_percent: 10.0,
                },
                PortfolioPosition {
                    token: TokenInfo {
                        address: "0x456".to_string(),
                        symbol: "USDC".to_string(),
                        name: "USD Coin".to_string(),
                        decimals: 6,
                        chain_id: 1,
                        price_usd: 1.0,
                        price_change_24h: 0.01,
                        volume_24h: 5000000.0,
                        liquidity: 10000000.0,
                    },
                    balance: 2000.0,
                    value_usd: 2000.0,
                    allocation_percent: 20.0,
                    pnl_percent: -15.0,
                },
            ],
            last_updated: Utc::now(),
        }
    }
    
    #[test]
    fn test_optimal_allocation() {
        let config = AgentConfig::default();
        let optimizer = PortfolioOptimizer::new(config);
        
        let portfolio = create_test_portfolio();
        let allocation = optimizer.calculate_optimal_allocation(&portfolio).unwrap();
        
        assert!(allocation.rebalance_needed);
    }
    
    #[test]
    fn test_tax_loss_harvesting() {
        let config = AgentConfig::default();
        let optimizer = PortfolioOptimizer::new(config);
        
        let portfolio = create_test_portfolio();
        let opportunities = optimizer.find_tax_loss_harvesting(&portfolio);
        
        // USDC should be a harvesting opportunity (negative returns)
        assert!(!opportunities.is_empty());
    }
}
