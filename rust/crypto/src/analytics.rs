//! TigerWallet Analytics Service
//! 
//! Comprehensive analytics and reporting for portfolio, DeFi, and governance

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

// ============================================================================
// Portfolio Analytics
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PortfolioAnalytics {
    pub total_value_usd: f64,
    pub total_cost_basis_usd: f64,
    pub total_pnl_usd: f64,
    pub total_pnl_percent: f64,
    pub assets: Vec<AssetAnalytics>,
    pub chains: Vec<ChainAnalytics>,
    pub DeFi_positions: Vec<DefiPosition>,
    pub nfts: Vec<NftPosition>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AssetAnalytics {
    pub symbol: String,
    pub name: String,
    pub chain: String,
    pub balance: f64,
    pub price_usd: f64,
    pub value_usd: f64,
    pub cost_basis_usd: f64,
    pub pnl_usd: f64,
    pub pnl_percent: f64,
    pub allocation_percent: f64,
    pub change_24h: f64,
    pub change_7d: f64,
    pub change_30d: f64,
    pub volume_24h: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainAnalytics {
    pub chain: String,
    pub total_value_usd: f64,
    pub num_assets: u32,
    pub allocation_percent: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DefiPosition {
    pub protocol: String,
    pub pool: String,
    pub token0: String,
    pub token1: String,
    pub value_usd: f64,
    pub apy: f64,
    pub pnl_7d: f64,
    pub rewards: Vec<Reward>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Reward {
    pub symbol: String,
    pub amount: f64,
    pub value_usd: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NftPosition {
    pub collection: String,
    pub token_id: String,
    pub name: String,
    pub image_url: String,
    pub floor_price_usd: f64,
    pub last_sale_usd: Option<f64>,
    pub attributes: Vec<NftAttribute>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NftAttribute {
    pub trait_type: String,
    pub value: String,
    pub rarity_percent: f64,
}

impl PortfolioAnalytics {
    pub fn new() -> Self {
        PortfolioAnalytics {
            total_value_usd: 0.0,
            total_cost_basis_usd: 0.0,
            total_pnl_usd: 0.0,
            total_pnl_percent: 0.0,
            assets: Vec::new(),
            chains: Vec::new(),
            DeFi_positions: Vec::new(),
            nfts: Vec::new(),
        }
    }
    
    pub fn calculate(&mut self) {
        self.total_value_usd = self.assets.iter().map(|a| a.value_usd).sum();
        self.total_cost_basis_usd = self.assets.iter().map(|a| a.cost_basis_usd).sum();
        self.total_pnl_usd = self.total_value_usd - self.total_cost_basis_usd;
        
        if self.total_cost_basis_usd > 0.0 {
            self.total_pnl_percent = (self.total_pnl_usd / self.total_cost_basis_usd) * 100.0;
        }
        
        // Calculate allocations
        for asset in &mut self.assets {
            if self.total_value_usd > 0.0 {
                asset.allocation_percent = (asset.value_usd / self.total_value_usd) * 100.0;
            }
        }
        
        // Aggregate by chain
        let mut chain_map: HashMap<String, (f64, u32)> = HashMap::new();
        for asset in &self.assets {
            let entry = chain_map.entry(asset.chain.clone()).or_insert((0.0, 0));
            entry.0 += asset.value_usd;
            entry.1 += 1;
        }
        
        self.chains = chain_map.into_iter().map(|(chain, (value, count))| {
            ChainAnalytics {
                chain,
                total_value_usd: value,
                num_assets: count,
                allocation_percent: if self.total_value_usd > 0.0 { 
                    (value / self.total_value_usd) * 100.0 
                } else { 0.0 },
            }
        }).collect();
    }
}

// ============================================================================
// Transaction Analytics
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransactionAnalytics {
    pub total_transactions: u64,
    pub total_volume_usd: f64,
    pub fees_paid_usd: f64,
    pub by_type: HashMap<String, u64>,
    pub by_chain: HashMap<String, u64>,
    pub by_status: HashMap<String, u64>,
    pub daily_volume: Vec<DailyVolume>,
    pub gas_efficiency: GasEfficiency,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DailyVolume {
    pub date: String,
    pub count: u64,
    pub volume_usd: f64,
    pub fees_usd: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GasEfficiency {
    pub average_gas_price_gwei: f64,
    pub total_gas_used: u64,
    pub optimized_transactions: u64,
    pub savings_usd: f64,
}

impl TransactionAnalytics {
    pub fn new() -> Self {
        TransactionAnalytics {
            total_transactions: 0,
            total_volume_usd: 0.0,
            fees_paid_usd: 0.0,
            by_type: HashMap::new(),
            by_chain: HashMap::new(),
            by_status: HashMap::new(),
            daily_volume: Vec::new(),
            gas_efficiency: GasEfficiency {
                average_gas_price_gwei: 0.0,
                total_gas_used: 0,
                optimized_transactions: 0,
                savings_usd: 0.0,
            },
        }
    }
    
    pub fn calculate_gas_savings(&mut self) {
        // Calculate potential savings from gas optimization
        let baseline_cost = self.total_gas_used() as f64 * 50.0; // Assume 50 gwei baseline
        let optimized_cost = self.gas_efficiency.optimized_transactions as f64 * 21000.0 * 30.0; // Optimized 30 gwei
        
        self.gas_efficiency.savings_usd = baseline_cost - optimized_cost;
    }
    
    fn total_gas_used(&self) -> u64 {
        self.daily_volume.iter().map(|d| d.count * 21000).sum()
    }
}

// ============================================================================
// Governance Analytics
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GovernanceAnalytics {
    pub dao_name: String,
    pub proposals: Vec<Proposal>,
    pub delegates: Vec<Delegate>,
    pub voter_turnout_percent: f64,
    pub active_voters: u64,
    pub total_voting_power: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Proposal {
    pub id: String,
    pub title: String,
    pub description: String,
    pub status: ProposalStatus,
    pub votes_for: f64,
    pub votes_against: f64,
    pub votes_abstain: f64,
    pub start_block: u64,
    pub end_block: u64,
    pub execution_data: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum ProposalStatus {
    Pending,
    Active,
    Canceled,
    Defeated,
    Succeeded,
    Queued,
    Executed,
    Expired,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Delegate {
    pub address: String,
    pub votes: f64,
    pub delegators: u64,
    pub proposals_voted: u64,
    pub voting_power_percent: f64,
}

// ============================================================================
// Market Intelligence
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketData {
    pub prices: HashMap<String, TokenPrice>,
    pub trends: HashMap<String, TrendData>,
    pub market_cap: f64,
    pub volume_24h: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenPrice {
    pub symbol: String,
    pub price_usd: f64,
    pub change_1h: f64,
    pub change_24h: f64,
    pub change_7d: f64,
    pub change_30d: f64,
    pub volume_24h: f64,
    pub market_cap: f64,
    pub fully_diluted_market_cap: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrendData {
    pub direction: TrendDirection,
    pub strength: f64,
    pub signals: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum TrendDirection {
    Bullish,
    Bearish,
    Neutral,
}

// ============================================================================
// Risk Analytics
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAnalytics {
    pub portfolio_risk: f64,
    pub concentration_risk: f64,
    pub smart_contract_risk: Vec<ContractRisk>,
    pub market_risk: f64,
    pub recommendations: Vec<RiskRecommendation>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContractRisk {
    pub protocol: String,
    pub risk_score: f64,
    pub audit_status: AuditStatus,
    pub bugs_found: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AuditStatus {
    Audited,
    PartiallyAudited,
    Unaudited,
    InProgress,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskRecommendation {
    pub category: String,
    pub description: String,
    pub impact: String,
    pub action: String,
}

impl RiskAnalytics {
    pub fn analyze(portfolio: &PortfolioAnalytics) -> Self {
        let mut risk = RiskAnalytics {
            portfolio_risk: 0.0,
            concentration_risk: 0.0,
            smart_contract_risk: Vec::new(),
            market_risk: 0.0,
            recommendations: Vec::new(),
        };
        
        // Calculate concentration risk (Herfindahl index)
        let sum_squares: f64 = portfolio.assets.iter()
            .map(|a| (a.allocation_percent / 100.0).powi(2))
            .sum();
        risk.concentration_risk = sum_squares * 100.0;
        
        // Generate recommendations
        if risk.concentration_risk > 25.0 {
            risk.recommendations.push(RiskRecommendation {
                category: "Concentration".to_string(),
                description: "Portfolio is concentrated in few assets".to_string(),
                impact: "High".to_string(),
                action: "Consider diversifying across more assets".to_string(),
            });
        }
        
        risk
    }
}

// ============================================================================
// Tax Report Generator
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TaxReport {
    pub year: u32,
    pub holdings: Vec<Holding>,
    pub transactions: Vec<TaxableTransaction>,
    pub capital_gains: CapitalGains,
    pub income: Vec<IncomeEvent>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Holding {
    pub symbol: String,
    pub chain: String,
    pub quantity: f64,
    pub cost_basis: f64,
    pub current_value: f64,
    pub unrealized_gain: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TaxableTransaction {
    pub date: String,
    pub transaction_type: TransactionType,
    pub symbol: String,
    pub quantity: f64,
    pub proceeds: f64,
    pub cost_basis: f64,
    pub gain_loss: f64,
    pub chain: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum TransactionType {
    Buy,
    Sell,
    Transfer,
    Airdrop,
    StakingReward,
    Interest,
    Mining,
    Income,
    Gift,
    Donation,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CapitalGains {
    pub short_term: f64,
    pub long_term: f64,
    pub total: f64,
    pub tax_estimate: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IncomeEvent {
    pub date: String,
    pub source: String,
    pub symbol: String,
    pub amount: f64,
    pub value_usd: f64,
}

impl TaxReport {
    pub fn generate(portfolio: &PortfolioAnalytics, transactions: &[TaxableTransaction], year: u32) -> Self {
        let mut short_term = 0.0;
        let mut long_term = 0.0;
        
        for tx in transactions {
            if tx.gain_loss != 0.0 {
                // Simplified: consider transactions > 1 year as long term
                short_term += tx.gain_loss;
            }
        }
        
        let total = short_term + long_term;
        
        // Estimate tax (simplified)
        let tax_estimate = if total > 0.0 {
            total * 0.25 // Assume 25% tax rate
        } else {
            0.0
        };
        
        // Build holdings from portfolio
        let holdings = portfolio.assets.iter().map(|a| {
            Holding {
                symbol: a.symbol.clone(),
                chain: a.chain.clone(),
                quantity: a.balance,
                cost_basis: a.cost_basis_usd,
                current_value: a.value_usd,
                unrealized_gain: a.pnl_usd,
            }
        }).collect();
        
        TaxReport {
            year,
            holdings,
            transactions: transactions.to_vec(),
            capital_gains: CapitalGains {
                short_term,
                long_term,
                total,
                tax_estimate,
            },
            income: Vec::new(),
        }
    }
    
    pub fn export_csv(&self) -> String {
        let mut csv = String::from("Date,Type,Symbol,Quantity,Proceeds,Cost Basis,Gain/Loss,Chain\n");
        
        for tx in &self.transactions {
            csv.push_str(&format!(
                "{},{:?},{},{},{},{},{},{}\n",
                tx.date, tx.transaction_type, tx.symbol, tx.quantity,
                tx.proceeds, tx.cost_basis, tx.gain_loss, tx.chain
            ));
        }
        
        csv
    }
    
    pub fn export_json(&self) -> String {
        serde_json::to_string_pretty(self).unwrap_or_default()
    }
}

// ============================================================================
// Performance Metrics
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceMetrics {
    pub time_range: String,
    pub total_return: f64,
    pub total_return_percent: f64,
    pub annualized_return: f64,
    pub sharpe_ratio: f64,
    pub max_drawdown: f64,
    pub volatility: f64,
    pub best_day: f64,
    pub worst_day: f64,
    pub win_rate: f64,
}

impl PerformanceMetrics {
    pub fn calculate(portfolio_values: &[f64]) -> Self {
        let n = portfolio_values.len();
        if n < 2 {
            return PerformanceMetrics::default();
        }
        
        // Calculate returns
        let mut returns: Vec<f64> = Vec::new();
        for i in 1..n {
            let r = (portfolio_values[i] - portfolio_values[i-1]) / portfolio_values[i-1];
            returns.push(r);
        }
        
        // Total return
        let start = portfolio_values.first().unwrap_or(&1.0);
        let end = portfolio_values.last().unwrap_or(&1.0);
        let total_return = end - start;
        let total_return_percent = (total_return / start) * 100.0;
        
        // Annualized return
        let days = n as f64;
        let annualized_return = if days > 365.0 {
            ((end / start).powf(365.0 / days) - 1.0) * 100.0
        } else {
            total_return_percent
        };
        
        // Mean return and std dev
        let mean: f64 = returns.iter().sum::<f64>() / returns.len() as f64;
        let variance: f64 = returns.iter().map(|r| (r - mean).powi(2)).sum::<f64>() / returns.len() as f64;
        let volatility = variance.sqrt() * 100.0;
        
        // Sharpe ratio (assuming 5% risk-free rate)
        let risk_free_rate = 5.0 / 365.0;
        let sharpe_ratio = if volatility > 0.0 {
            (mean - risk_free_rate) / volatility * (252.0_f64).sqrt()
        } else {
            0.0
        };
        
        // Max drawdown
        let mut max_drawdown = 0.0;
        let mut peak = portfolio_values[0];
        for &value in portfolio_values {
            if value > peak {
                peak = value;
            }
            let drawdown = (peak - value) / peak;
            if drawdown > max_drawdown {
                max_drawdown = drawdown;
            }
        }
        
        // Win rate
        let wins = returns.iter().filter(|&&r| r > 0.0).count();
        let win_rate = (wins as f64 / returns.len() as f64) * 100.0;
        
        PerformanceMetrics {
            time_range: format!("{} days", n),
            total_return,
            total_return_percent,
            annualized_return,
            sharpe_ratio,
            max_drawdown: max_drawdown * 100.0,
            volatility,
            best_day: returns.iter().cloned().fold(f64::NEG_INFINITY, f64::max),
            worst_day: returns.iter().cloned().fold(f64::INFINITY, f64::min),
            win_rate,
        }
    }
}

impl Default for PerformanceMetrics {
    fn default() -> Self {
        PerformanceMetrics {
            time_range: "0 days".to_string(),
            total_return: 0.0,
            total_return_percent: 0.0,
            annualized_return: 0.0,
            sharpe_ratio: 0.0,
            max_drawdown: 0.0,
            volatility: 0.0,
            best_day: 0.0,
            worst_day: 0.0,
            win_rate: 0.0,
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
    fn test_portfolio_analytics() {
        let mut portfolio = PortfolioAnalytics::new();
        portfolio.assets.push(AssetAnalytics {
            symbol: "ETH".to_string(),
            name: "Ethereum".to_string(),
            chain: "ethereum".to_string(),
            balance: 10.0,
            price_usd: 2500.0,
            value_usd: 25000.0,
            cost_basis_usd: 20000.0,
            pnl_usd: 5000.0,
            pnl_percent: 25.0,
            allocation_percent: 0.0,
            change_24h: 2.5,
            change_7d: 5.0,
            change_30d: 10.0,
            volume_24h: 1_000_000_000.0,
        });
        
        portfolio.calculate();
        
        assert_eq!(portfolio.total_value_usd, 25000.0);
        assert_eq!(portfolio.total_pnl_usd, 5000.0);
    }
    
    #[test]
    fn test_risk_analytics() {
        let mut portfolio = PortfolioAnalytics::new();
        portfolio.assets.push(AssetAnalytics {
            symbol: "ETH".to_string(),
            name: "Ethereum".to_string(),
            chain: "ethereum".to_string(),
            balance: 10.0,
            price_usd: 2500.0,
            value_usd: 25000.0,
            cost_basis_usd: 20000.0,
            pnl_usd: 5000.0,
            pnl_percent: 25.0,
            allocation_percent: 100.0,
            change_24h: 2.5,
            change_7d: 5.0,
            change_30d: 10.0,
            volume_24h: 1_000_000_000.0,
        });
        
        let risk = RiskAnalytics::analyze(&portfolio);
        
        assert!(risk.concentration_risk > 0.0);
    }
    
    #[test]
    fn test_performance_metrics() {
        let values = vec![10000.0, 10500.0, 10200.0, 11000.0, 10800.0, 11500.0];
        let metrics = PerformanceMetrics::calculate(&values);
        
        assert!(metrics.total_return > 0.0);
    }
}