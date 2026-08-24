//! Transaction Analyzer Module
//! 
//! Analyzes transactions and provides smart suggestions

use crate::{AgentConfig, TokenInfo, Portfolio, PortfolioPosition, TransactionSuggestion, SuggestionType, RiskLevel, SuggestionAction, ActionType, AgentError};
use chrono::Utc;

/// Transaction Analyzer
pub struct TransactionAnalyzer {
    config: AgentConfig,
}

impl TransactionAnalyzer {
    /// Create new analyzer
    pub fn new(config: AgentConfig) -> Self {
        Self { config }
    }
    
    /// Analyze portfolio and generate suggestions
    pub async fn analyze_portfolio(&self, portfolio: &Portfolio) -> Result<Vec<TransactionSuggestion>, AgentError> {
        let mut suggestions = Vec::new();
        
        // Check for rebalancing opportunities
        if self.config.enable_auto_rebalance {
            suggestions.extend(self.check_rebalancing(portfolio)?);
        }
        
        // Check for gas optimization
        if self.config.enable_gas_optimization {
            suggestions.extend(self.check_gas_optimization(portfolio).await?);
        }
        
        // Check for diversification
        suggestions.extend(self.check_diversification(portfolio)?);
        
        // Check for staking opportunities
        suggestions.extend(self.check_staking_opportunities(portfolio)?);
        
        Ok(suggestions)
    }
    
    /// Check for rebalancing opportunities
    fn check_rebalancing(&self, portfolio: &Portfolio) -> Result<Vec<TransactionSuggestion>, AgentError> {
        let mut suggestions = Vec::new();
        
        for position in &portfolio.positions {
            // If allocation is significantly different from target
            let target_allocation = 100.0 / portfolio.positions.len() as f64;
            let deviation = (position.allocation_percent - target_allocation).abs();
            
            if deviation > self.config.rebalance_threshold {
                let is_over_weight = position.allocation_percent > target_allocation;
                
                suggestions.push(TransactionSuggestion {
                    id: format!("rebalance_{}_{}", position.token.symbol, Utc::now().timestamp()),
                    suggestion_type: SuggestionType::Rebalance,
                    title: format!("Rebalance {}", position.token.symbol),
                    description: if is_over_weight {
                        format!("{} is over-allocated at {:.1}%. Consider reducing exposure.", 
                            position.token.symbol, position.allocation_percent)
                    } else {
                        format!("{} is under-allocated at {:.1}%. Consider increasing exposure.", 
                            position.token.symbol, position.allocation_percent)
                    },
                    expected_outcome: "Portfolio rebalanced to target allocation".to_string(),
                    confidence: 0.85,
                    estimated_gas: 150000,
                    estimated_value_usd: position.value_usd * (deviation / 100.0),
                    risk_level: RiskLevel::Low,
                    action: SuggestionAction {
                        action_type: ActionType::Swap,
                        from_token: Some(position.token.address.clone()),
                        to_token: None,
                        amount: Some(position.value_usd * (deviation / 100.0)),
                        slippage: self.config.max_slippage,
                        deadline_seconds: 600,
                    },
                });
            }
        }
        
        Ok(suggestions)
    }
    
    /// Fetch the current network gas price (gwei) from the configured EVM
    /// RPC endpoint via `eth_gasPrice`. Returns None when no RPC is
    /// configured or the node is unreachable — callers must not fabricate a
    /// value in that case.
    async fn fetch_current_gas_gwei() -> Option<f64> {
        let rpc_url = std::env::var("EVM_RPC_URL").ok()?;
        let client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(5))
            .build()
            .ok()?;
        let body = serde_json::json!({
            "jsonrpc": "2.0",
            "method": "eth_gasPrice",
            "params": [],
            "id": 1
        });
        let resp: serde_json::Value = client.post(&rpc_url).json(&body).send().await.ok()?.json().await.ok()?;
        let hex_price = resp.get("result")?.as_str()?;
        let wei = u128::from_str_radix(hex_price.trim_start_matches("0x"), 16).ok()?;
        Some(wei as f64 / 1e9)
    }

    /// Check for gas optimization opportunities
    async fn check_gas_optimization(&self, _portfolio: &Portfolio) -> Result<Vec<TransactionSuggestion>, AgentError> {
        let mut suggestions = Vec::new();

        // Real network gas price from the configured RPC. If we cannot
        // obtain one we emit no gas suggestion rather than inventing a price.
        let current_gas = match Self::fetch_current_gas_gwei().await {
            Some(g) => g,
            None => return Ok(suggestions),
        };
        let optimal_gas = current_gas * 0.7;
        let savings = (1.0 - optimal_gas / current_gas) * 100.0;
        
        if savings > 10.0 {
            suggestions.push(TransactionSuggestion {
                id: format!("gas_optimize_{}", Utc::now().timestamp()),
                suggestion_type: SuggestionType::OptimizeGas,
                title: "Optimize Gas Costs".to_string(),
                description: format!("Current gas prices are high. Waiting could save approximately {:.1}% on transaction costs.", savings),
                expected_outcome: format!("Save ~{:.1}% on gas fees", savings),
                confidence: 0.9,
                estimated_gas: 21000,
                estimated_value_usd: 0.0,
                risk_level: RiskLevel::VeryLow,
                action: SuggestionAction {
                    action_type: ActionType::Transfer,
                    from_token: None,
                    to_token: None,
                    amount: None,
                    slippage: 0.1,
                    deadline_seconds: 3600, // 1 hour
                },
            });
        }
        
        Ok(suggestions)
    }
    
    /// Check for diversification opportunities
    fn check_diversification(&self, portfolio: &Portfolio) -> Result<Vec<TransactionSuggestion>, AgentError> {
        let mut suggestions = Vec::new();
        
        // If portfolio is too concentrated
        if let Some(largest) = portfolio.positions.iter().max_by(|a, b| a.allocation_percent.partial_cmp(&b.allocation_percent).unwrap()) {
            if largest.allocation_percent > 50.0 {
                suggestions.push(TransactionSuggestion {
                    id: format!("diversify_{}_{}", largest.token.symbol, Utc::now().timestamp()),
                    suggestion_type: SuggestionType::Diversify,
                    title: "Portfolio Too Concentrated".to_string(),
                    description: format!("{} represents {:.1}% of your portfolio. Consider diversifying to reduce risk.", 
                        largest.token.symbol, largest.allocation_percent),
                    expected_outcome: "Reduced concentration risk".to_string(),
                    confidence: 0.8,
                    estimated_gas: 200000,
                    estimated_value_usd: largest.value_usd * 0.3,
                    risk_level: RiskLevel::Medium,
                    action: SuggestionAction {
                        action_type: ActionType::Swap,
                        from_token: Some(largest.token.address.clone()),
                        to_token: None,
                        amount: Some(largest.value_usd * 0.3),
                        slippage: self.config.max_slippage,
                        deadline_seconds: 900,
                    },
                });
            }
        }
        
        // If portfolio has too few assets
        if portfolio.positions.len() < 3 {
            suggestions.push(TransactionSuggestion {
                id: format!("diversify_add_{}", Utc::now().timestamp()),
                suggestion_type: SuggestionType::Diversify,
                title: "Add More Assets".to_string(),
                description: "Your portfolio has limited diversification. Consider adding more assets to spread risk.".to_string(),
                expected_outcome: "Better risk-adjusted returns".to_string(),
                confidence: 0.75,
                estimated_gas: 250000,
                estimated_value_usd: portfolio.total_value_usd * 0.2,
                risk_level: RiskLevel::Low,
                action: SuggestionAction {
                    action_type: ActionType::Swap,
                    from_token: None,
                    to_token: None,
                    amount: Some(portfolio.total_value_usd * 0.2),
                    slippage: self.config.max_slippage,
                    deadline_seconds: 900,
                },
            });
        }
        
        Ok(suggestions)
    }
    
    /// Check for staking opportunities
    fn check_staking_opportunities(&self, portfolio: &Portfolio) -> Result<Vec<TransactionSuggestion>, AgentError> {
        let mut suggestions = Vec::new();
        
        // Check tokens that could be staked for yield
        let stakeable_tokens = ["ETH", "SOL", "ATOM", "DOT", "ADA"];
        
        for position in &portfolio.positions {
            if stakeable_tokens.contains(&position.token.symbol.to_uppercase().as_str()) {
                // If not currently staked (assuming we have this info)
                let staking_yield = match position.token.symbol.to_uppercase().as_str() {
                    "ETH" => 4.5,
                    "SOL" => 6.0,
                    "ATOM" => 5.5,
                    "DOT" => 7.0,
                    "ADA" => 4.0,
                    _ => 5.0,
                };
                
                let annual_reward = position.value_usd * (staking_yield / 100.0);
                
                if annual_reward > 100.0 {
                    suggestions.push(TransactionSuggestion {
                        id: format!("stake_{}_{}", position.token.symbol, Utc::now().timestamp()),
                        suggestion_type: SuggestionType::Stake,
                        title: format!("Stake {} for Yield", position.token.symbol),
                        description: format!("Stake your {} to earn approximately {:.1}% APY (${:.2}/year)", 
                            position.token.symbol, staking_yield, annual_reward),
                        expected_outcome: format!("Earn ${:.2} annually in staking rewards", annual_reward),
                        confidence: 0.95,
                        estimated_gas: 100000,
                        estimated_value_usd: annual_reward,
                        risk_level: RiskLevel::Low,
                        action: SuggestionAction {
                            action_type: ActionType::Stake,
                            from_token: Some(position.token.address.clone()),
                            to_token: None,
                            amount: Some(position.balance),
                            slippage: 0.1,
                            deadline_seconds: 300,
                        },
                    });
                }
            }
        }
        
        Ok(suggestions)
    }
    
    /// Analyze a specific transaction
    pub fn analyze_transaction(
        &self,
        from_token: &str,
        to_token: &str,
        amount: f64,
        expected_price_impact: f64,
    ) -> Result<TransactionSuggestion, AgentError> {
        // Determine risk based on price impact
        let risk = if expected_price_impact > 10.0 {
            RiskLevel::VeryHigh
        } else if expected_price_impact > 5.0 {
            RiskLevel::High
        } else if expected_price_impact > 2.0 {
            RiskLevel::Medium
        } else {
            RiskLevel::Low
        };
        
        let confidence = (1.0 - expected_price_impact / 100.0).max(0.5);
        
        Ok(TransactionSuggestion {
            id: format!("tx_analyze_{}", Utc::now().timestamp()),
            suggestion_type: SuggestionType::Swap,
            title: format!("Swap {} to {}", from_token, to_token),
            description: format!("Swap {:.2} {} to {} with {:.2}% expected price impact", 
                amount, from_token, to_token, expected_price_impact),
            expected_outcome: "Token swap executed".to_string(),
            confidence,
            estimated_gas: 150000, // typical AMM swap gas usage
            estimated_value_usd: amount,
            risk_level: risk,
            action: SuggestionAction {
                action_type: ActionType::Swap,
                from_token: Some(from_token.to_string()),
                to_token: Some(to_token.to_string()),
                amount: Some(amount),
                slippage: self.config.max_slippage,
                deadline_seconds: 600,
            },
        })
    }
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
                    pnl_percent: 0.0,
                },
            ],
            last_updated: Utc::now(),
        }
    }
    
    #[tokio::test]
    async fn test_analyzer_creation() {
        let config = AgentConfig::default();
        let analyzer = TransactionAnalyzer::new(config);
        
        let portfolio = create_test_portfolio();
        let suggestions = analyzer.analyze_portfolio(&portfolio).await.unwrap();
        
        // Should have rebalancing and diversification suggestions
        assert!(!suggestions.is_empty());
    }
    
    #[test]
    fn test_transaction_analysis() {
        let config = AgentConfig::default();
        let analyzer = TransactionAnalyzer::new(config);
        
        let suggestion = analyzer.analyze_transaction("ETH", "USDC", 1.0, 1.5).unwrap();
        
        assert_eq!(suggestion.risk_level, RiskLevel::Low);
    }
}
