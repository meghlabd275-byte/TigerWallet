//! TigerWallet AI Agent - Production Ready
//! 
//! AI-powered wallet operations including:
//! - Smart transaction suggestions
//! - Auto portfolio rebalancing
//! - Gas optimization
//! - Price prediction
//! - Risk assessment

#![allow(dead_code)]

pub mod transaction_analyzer;
pub mod portfolio_optimizer;
pub mod gas_optimizer;
pub mod price_predictor;
pub mod risk_assessor;
pub mod agent_service;

pub use transaction_analyzer::*;
pub use portfolio_optimizer::*;
pub use gas_optimizer::*;
pub use price_predictor::*;
pub use risk_assessor::*;
pub use agent_service::*;

// ============================================================================
// Core Types
// ============================================================================

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// AI Agent configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgentConfig {
    /// Enable smart suggestions
    pub enable_suggestions: bool,
    /// Enable auto-rebalancing
    pub enable_auto_rebalance: bool,
    /// Enable gas optimization
    pub enable_gas_optimization: bool,
    /// Enable price prediction
    pub enable_price_prediction: bool,
    /// Enable risk assessment
    pub enable_risk_assessment: bool,
    /// Rebalance threshold (percentage)
    pub rebalance_threshold: f64,
    /// Max slippage tolerance
    pub max_slippage: f64,
    /// API keys for price feeds
    pub price_api_key: Option<String>,
}

impl Default for AgentConfig {
    fn default() -> Self {
        Self {
            enable_suggestions: true,
            enable_auto_rebalance: false,
            enable_gas_optimization: true,
            enable_price_prediction: true,
            enable_risk_assessment: true,
            rebalance_threshold: 5.0,
            max_slippage: 0.5,
            price_api_key: None,
        }
    }
}

/// Token information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenInfo {
    pub address: String,
    pub symbol: String,
    pub name: String,
    pub decimals: u8,
    pub chain_id: u64,
    pub price_usd: f64,
    pub price_change_24h: f64,
    pub volume_24h: f64,
    pub liquidity: f64,
}

/// Portfolio position
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PortfolioPosition {
    pub token: TokenInfo,
    pub balance: f64,
    pub value_usd: f64,
    pub allocation_percent: f64,
    pub pnl_percent: f64,
}

/// Complete portfolio
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Portfolio {
    pub total_value_usd: f64,
    pub positions: Vec<PortfolioPosition>,
    pub last_updated: DateTime<Utc>,
}

/// Transaction suggestion
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionSuggestion {
    pub id: String,
    pub suggestion_type: SuggestionType,
    pub title: String,
    pub description: String,
    pub expected_outcome: String,
    pub confidence: f64,
    pub estimated_gas: u64,
    pub estimated_value_usd: f64,
    pub risk_level: RiskLevel,
    pub action: SuggestionAction,
}

/// Suggestion types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SuggestionType {
    Rebalance,
    Swap,
    Stake,
    Unstake,
    Bridge,
    Claim,
    Invest,
    Diversify,
    OptimizeGas,
    StopLoss,
    TakeProfit,
}

/// Risk levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RiskLevel {
    VeryLow,
    Low,
    Medium,
    High,
    VeryHigh,
}

/// Suggested action
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SuggestionAction {
    pub action_type: ActionType,
    pub from_token: Option<String>,
    pub to_token: Option<String>,
    pub amount: Option<f64>,
    pub slippage: f64,
    pub deadline_seconds: u64,
}

/// Action types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ActionType {
    Swap,
    Stake,
    Unstake,
    Bridge,
    Claim,
    Transfer,
}

/// Gas optimization suggestion
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasOptimization {
    pub current_gas_price_gwei: f64,
    pub suggested_gas_price_gwei: f64,
    pub optimal_time_to_send: DateTime<Utc>,
    pub estimated_savings_percent: f64,
    pub recommended_delay_seconds: i64,
}

/// Price prediction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PricePrediction {
    pub token: String,
    pub current_price: f64,
    pub predictions: Vec<PricePoint>,
    pub trend: PriceTrend,
    pub confidence: f64,
}

/// Price point
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PricePoint {
    pub timestamp: DateTime<Utc>,
    pub predicted_price: f64,
    pub upper_bound: f64,
    pub lower_bound: f64,
}

/// Price trend
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum PriceTrend {
    StrongBullish,
    Bullish,
    Neutral,
    Bearish,
    StrongBearish,
}

/// Risk assessment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAssessment {
    pub overall_risk_score: f64,
    pub risk_factors: Vec<RiskFactor>,
    pub recommendations: Vec<String>,
    pub assessed_at: DateTime<Utc>,
}

/// Risk factor
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskFactor {
    pub factor: String,
    pub severity: RiskLevel,
    pub description: String,
    pub mitigation: String,
}

/// AI Errors
#[derive(Debug, thiserror::Error)]
pub enum AgentError {
    #[error("API error: {0}")]
    ApiError(String),
    
    #[error("Analysis error: {0}")]
    AnalysisError(String),
    
    #[error("Prediction error: {0}")]
    PredictionError(String),
    
    #[error("Configuration error: {0}")]
    ConfigError(String),
    
    #[error("Network error: {0}")]
    NetworkError(String),
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_agent_config_default() {
        let config = AgentConfig::default();
        assert!(config.enable_suggestions);
        assert!(config.enable_gas_optimization);
        assert_eq!(config.rebalance_threshold, 5.0);
    }
    
    #[test]
    fn test_portfolio_creation() {
        let portfolio = Portfolio {
            total_value_usd: 10000.0,
            positions: vec![],
            last_updated: Utc::now(),
        };
        
        assert_eq!(portfolio.total_value_usd, 10000.0);
    }
}
