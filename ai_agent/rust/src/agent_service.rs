//! Agent Service Module
//! 
//! Main service that coordinates all AI agent components

use crate::{
    AgentConfig, Portfolio, TokenInfo, TransactionSuggestion, 
    GasOptimization, PricePrediction, RiskAssessment,
    TransactionAnalyzer, PortfolioOptimizer, GasOptimizer, 
    PricePredictor, RiskAssessor, AgentError
};
use std::sync::Arc;
use tokio::sync::RwLock;

/// AI Agent Service - Main entry point
pub struct AgentService {
    config: AgentConfig,
    transaction_analyzer: TransactionAnalyzer,
    portfolio_optimizer: PortfolioOptimizer,
    gas_optimizer: Arc<RwLock<GasOptimizer>>,
    price_predictor: Arc<RwLock<PricePredictor>>,
    risk_assessor: RiskAssessor,
}

impl AgentService {
    /// Create new agent service
    pub fn new(config: AgentConfig) -> Self {
        Self {
            config: config.clone(),
            transaction_analyzer: TransactionAnalyzer::new(config.clone()),
            portfolio_optimizer: PortfolioOptimizer::new(config.clone()),
            gas_optimizer: Arc::new(RwLock::new(GasOptimizer::new())),
            price_predictor: Arc::new(RwLock::new(PricePredictor::new())),
            risk_assessor: RiskAssessor,
        }
    }
    
    /// Get transaction suggestions for portfolio
    pub async fn get_suggestions(&self, portfolio: &Portfolio) -> Result<Vec<TransactionSuggestion>, AgentError> {
        self.transaction_analyzer.analyze_portfolio(portfolio).await
    }
    
    /// Analyze specific transaction
    pub async fn analyze_transaction(
        &self,
        from_token: &str,
        to_token: &str,
        amount: f64,
        expected_price_impact: f64,
    ) -> Result<TransactionSuggestion, AgentError> {
        self.transaction_analyzer.analyze_transaction(from_token, to_token, amount, expected_price_impact)
    }
    
    /// Get gas optimization
    pub async fn get_gas_optimization(&self) -> Result<GasOptimization, AgentError> {
        let optimizer = self.gas_optimizer.read().await;
        optimizer.get_optimal_gas()
    }
    
    /// Update gas prices
    pub async fn update_gas_prices(&self, prices: crate::gas_optimizer::GasPrices) {
        let mut optimizer = self.gas_optimizer.write().await;
        optimizer.update_gas_prices(prices);
    }
    
    /// Get price prediction
    pub async fn get_price_prediction(&self, token: &str, hours_ahead: i64) -> Result<PricePrediction, AgentError> {
        let predictor = self.price_predictor.read().await;
        predictor.predict(token, hours_ahead)
    }
    
    /// Update price data
    pub async fn update_price_data(&self, token: &str, price: f64, volume: f64) {
        let mut predictor = self.price_predictor.write().await;
        predictor.add_price_data(price, volume);
    }
    
    /// Get portfolio allocation
    pub async fn get_portfolio_allocation(&self, portfolio: &Portfolio) -> Result<crate::portfolio_optimizer::PortfolioAllocation, AgentError> {
        self.portfolio_optimizer.calculate_optimal_allocation(portfolio)
    }
    
    /// Assess transaction risk
    pub async fn assess_transaction_risk(
        &self,
        from_token: &TokenInfo,
        to_token: &TokenInfo,
        amount: f64,
        portfolio: &Portfolio,
    ) -> Result<RiskAssessment, AgentError> {
        self.risk_assessor.assess_transaction(from_token, to_token, amount, portfolio)
    }
    
    /// Assess portfolio risk
    pub async fn assess_portfolio_risk(&self, portfolio: &Portfolio) -> Result<RiskAssessment, AgentError> {
        self.risk_assessor.assess_portfolio(portfolio)
    }
    
    /// Get complete analysis
    pub async fn get_complete_analysis(&self, portfolio: &Portfolio) -> Result<CompleteAnalysis, AgentError> {
        let suggestions = self.get_suggestions(portfolio).await?;
        let allocation = self.get_portfolio_allocation(portfolio).await?;
        let gas_opt = self.get_gas_optimization().await.ok();
        let risk = self.assess_portfolio_risk(portfolio).await.ok();
        
        Ok(CompleteAnalysis {
            suggestions,
            portfolio_allocation: allocation,
            gas_optimization: gas_opt,
            risk_assessment: risk,
        })
    }
}

/// Complete analysis result
pub struct CompleteAnalysis {
    pub suggestions: Vec<TransactionSuggestion>,
    pub portfolio_allocation: crate::portfolio_optimizer::PortfolioAllocation,
    pub gas_optimization: Option<GasOptimization>,
    pub risk_assessment: Option<RiskAssessment>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::PortfolioPosition;
    use chrono::Utc;
    
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
    async fn test_agent_service() {
        let config = AgentConfig::default();
        let service = AgentService::new(config);
        
        let portfolio = create_test_portfolio();
        
        let suggestions = service.get_suggestions(&portfolio).await.unwrap();
        
        assert!(!suggestions.is_empty());
    }
}
