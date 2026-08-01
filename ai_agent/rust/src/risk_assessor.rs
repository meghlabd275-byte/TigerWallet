//! Risk Assessor Module
//! 
//! Provides risk assessment for transactions and portfolio

use crate::{RiskAssessment, RiskFactor, RiskLevel, Portfolio, TokenInfo, AgentError};
use chrono::Utc;

/// Risk Assessor
pub struct RiskAssessor;

impl RiskAssessor {
    /// Assess transaction risk
    pub fn assess_transaction(
        &self,
        from_token: &TokenInfo,
        to_token: &TokenInfo,
        amount: f64,
        user_portfolio: &Portfolio,
    ) -> Result<RiskAssessment, AgentError> {
        let mut risk_factors = Vec::new();
        let mut risk_score = 0.0;
        
        // Check if amount is significant compared to portfolio
        let portfolio_value = user_portfolio.total_value_usd;
        let concentration = amount / portfolio_value;
        
        if concentration > 0.5 {
            risk_factors.push(RiskFactor {
                factor: "High Concentration".to_string(),
                severity: RiskLevel::High,
                description: format!("This transaction represents {:.1}% of your portfolio", concentration * 100.0),
                mitigation: "Consider splitting into smaller transactions".to_string(),
            });
            risk_score += 30.0;
        } else if concentration > 0.25 {
            risk_factors.push(RiskFactor {
                factor: "Medium Concentration".to_string(),
                severity: RiskLevel::Medium,
                description: format!("This transaction represents {:.1}% of your portfolio", concentration * 100.0),
                mitigation: "Monitor portfolio balance after transaction".to_string(),
            });
            risk_score += 15.0;
        }
        
        // Check token liquidity
        if to_token.liquidity < 100000.0 {
            risk_factors.push(RiskFactor {
                factor: "Low Liquidity".to_string(),
                severity: RiskLevel::High,
                description: format!("{} has low liquidity (${:.2})", to_token.symbol, to_token.liquidity),
                mitigation: "Use limit order instead of market order".to_string(),
            });
            risk_score += 25.0;
        } else if to_token.liquidity < 1000000.0 {
            risk_factors.push(RiskFactor {
                factor: "Medium Liquidity".to_string(),
                severity: RiskLevel::Medium,
                description: format!("{} has moderate liquidity", to_token.symbol),
                mitigation: "Set appropriate slippage tolerance".to_string(),
            });
            risk_score += 10.0;
        }
        
        // Check price volatility
        if to_token.price_change_24h.abs() > 20.0 {
            risk_factors.push(RiskFactor {
                factor: "High Price Volatility".to_string(),
                severity: RiskLevel::High,
                description: format!("{} moved {:.1}% in 24h", to_token.symbol, to_token.price_change_24h),
                mitigation: "Consider waiting for more stable conditions".to_string(),
            });
            risk_score += 20.0;
        }
        
        // Cap risk score at 100
        risk_score = risk_score.min(100.0);
        
        // Generate recommendations
        let recommendations = self.generate_recommendations(&risk_factors, risk_score);
        
        Ok(RiskAssessment {
            overall_risk_score: risk_score,
            risk_factors,
            recommendations,
            assessed_at: Utc::now(),
        })
    }
    
    /// Assess portfolio risk
    pub fn assess_portfolio(&self, portfolio: &Portfolio) -> Result<RiskAssessment, AgentError> {
        let mut risk_factors = Vec::new();
        let mut risk_score = 0.0;
        
        // Check concentration risk
        if let Some(largest) = portfolio.positions.iter().max_by(|a, b| a.allocation_percent.partial_cmp(&b.allocation_percent).unwrap()) {
            if largest.allocation_percent > 50.0 {
                risk_factors.push(RiskFactor {
                    factor: "Portfolio Concentration".to_string(),
                    severity: RiskLevel::High,
                    description: format!("{} represents {:.1}% of portfolio", largest.token.symbol, largest.allocation_percent),
                    mitigation: "Diversify across more assets".to_string(),
                });
                risk_score += 25.0;
            }
        }
        
        // Check for stablecoins
        let stablecoins = ["USDC", "USDT", "DAI", "FRAX"];
        let stablecoin_value: f64 = portfolio.positions.iter()
            .filter(|p| stablecoins.contains(&p.token.symbol.to_uppercase().as_str()))
            .map(|p| p.value_usd)
            .sum();
        
        let stablecoin_ratio = stablecoin_value / portfolio.total_value_usd;
        
        if stablecoin_ratio > 0.8 {
            risk_factors.push(RiskFactor {
                factor: "Too Much Stablecoin".to_string(),
                severity: RiskLevel::Medium,
                description: format!("{:.1}% in stablecoins - missing yield opportunities", stablecoin_ratio * 100.0),
                mitigation: "Consider deploying stablecoins in yield strategies".to_string(),
            });
            risk_score += 10.0;
        }
        
        // Check for correlated assets (simplified)
        let asset_count = portfolio.positions.len();
        
        if asset_count < 3 {
            risk_factors.push(RiskFactor {
                factor: "Insufficient Diversification".to_string(),
                severity: RiskLevel::High,
                description: "Portfolio has fewer than 3 assets".to_string(),
                mitigation: "Add more diverse assets".to_string(),
            });
            risk_score += 20.0;
        }
        
        // Check for negative positions
        let negative_positions: Vec<_> = portfolio.positions.iter()
            .filter(|p| p.pnl_percent < -20.0)
            .collect();
        
        if !negative_positions.is_empty() {
            risk_factors.push(RiskFactor {
                factor: "Significant Losses".to_string(),
                severity: RiskLevel::High,
                description: format!("{} positions have >20% loss", negative_positions.len()),
                mitigation: "Review loss-making positions for potential tax loss harvesting".to_string(),
            });
            risk_score += 15.0;
        }
        
        // Cap risk score
        risk_score = risk_score.min(100.0);
        
        let recommendations = self.generate_recommendations(&risk_factors, risk_score);
        
        Ok(RiskAssessment {
            overall_risk_score: risk_score,
            risk_factors,
            recommendations,
            assessed_at: Utc::now(),
        })
    }
    
    /// Generate recommendations based on risk factors
    fn generate_recommendations(&self, factors: &[RiskFactor], score: f64) -> Vec<String> {
        let mut recommendations = Vec::new();
        
        if score > 70.0 {
            recommendations.push("Consider postponing large transactions until risk is reduced".to_string());
        }
        
        if factors.iter().any(|f| matches!(f.severity, RiskLevel::High | RiskLevel::VeryHigh)) {
            recommendations.push("Review all high severity risk factors before proceeding".to_string());
        }
        
        if recommendations.is_empty() {
            if score < 30.0 {
                recommendations.push("Risk levels are acceptable for most transactions".to_string());
            } else {
                recommendations.push("Transaction is acceptable with normal precautions".to_string());
            }
        }
        
        recommendations
    }
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
    
    #[test]
    fn test_portfolio_risk() {
        let assessor = RiskAssessor;
        let portfolio = create_test_portfolio();
        
        let assessment = assessor.assess_portfolio(&portfolio).unwrap();
        
        // Should have concentration risk
        assert!(!assessment.risk_factors.is_empty());
    }
}
