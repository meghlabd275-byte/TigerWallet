//! TigerWallet High-Performance Calculator
//! 
//! Ultra-low latency calculations for trading, pricing, and risk management

use std::sync::Arc;
use parking_lot::RwLock;
use serde::{Deserialize, Serialize};

/// Calculation precision level
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Precision {
    Low,      // 6 decimal places
    Medium,   // 8 decimal places  
    High,     // 12 decimal places
    Ultra,    // 18 decimal places (for DeFi)
}

impl Precision {
    fn scale(&self) -> u64 {
        match self {
            Precision::Low => 1_000_000,
            Precision::Medium => 100_000_000,
            Precision::High => 1_000_000_000_000,
            Precision::Ultra => 1_000_000_000_000_000_000,
        }
    }
}

/// Price quote with metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceQuote {
    pub symbol: String,
    pub bid_price: f64,
    pub ask_price: f64,
    pub mid_price: f64,
    pub spread: f64,
    pub spread_bps: f64,
    pub timestamp: i64,
    pub confidence: f64,
    pub source: String,
}

impl PriceQuote {
    pub fn calculate_mid(&self) -> f64 {
        (self.bid_price + self.ask_price) / 2.0
    }

    pub fn calculate_spread(&self) -> f64 {
        self.ask_price - self.bid_price
    }

    pub fn calculate_spread_bps(&self) -> f64 {
        if self.mid_price == 0.0 { return 0.0; }
        (self.spread / self.mid_price) * 10000.0
    }

    pub fn calculate_slippage(&self, trade_size: f64) -> f64 {
        let impact = trade_size.sqrt() * 0.001;
        self.mid_price * impact
    }
}

/// Swap calculation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapCalculation {
    pub from_token: String,
    pub to_token: String,
    pub from_amount: f64,
    pub to_amount: f64,
    pub price_impact: f64,
    pub price_impact_bps: f64,
    pub gas_estimate: f64,
    pub gas_cost_usd: f64,
    pub total_cost_usd: f64,
    pub route: Vec<String>,
    pub expected_output: f64,
    pub minimum_output: f64,
}

impl SwapCalculation {
    pub fn with_slippage(&self, tolerance: f64) -> f64 {
        self.expected_output * (1.0 - tolerance)
    }

    pub fn net_output(&self) -> f64 {
        self.to_amount - (self.gas_cost_usd / self.price_impact)
    }
}

/// Position calculation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Position {
    pub symbol: String,
    pub size: f64,
    pub entry_price: f64,
    pub current_price: f64,
    pub unrealized_pnl: f64,
    pub unrealized_pnl_pct: f64,
    pub leverage: f64,
    pub margin: f64,
    pub liquidation_price: f64,
    pub risk_level: RiskLevel,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum RiskLevel {
    Safe,
    Moderate,
    High,
    Critical,
    Liquidated,
}

impl Position {
    pub fn calculate_pnl(&mut self) {
        self.unrealized_pnl = self.size * (self.current_price - self.entry_price);
        
        if self.margin > 0.0 {
            self.unrealized_pnl_pct = (self.unrealized_pnl / self.margin) * 100.0;
        }

        let pnl_pct = self.unrealized_pnl_pct;
        self.risk_level = if pnl_pct > 50.0 {
            RiskLevel::Safe
        } else if pnl_pct > 20.0 {
            RiskLevel::Moderate
        } else if pnl_pct > 0.0 {
            RiskLevel::High
        } else if pnl_pct > -20.0 {
            RiskLevel::Critical
        } else {
            RiskLevel::Liquidated
        };
    }

    pub fn calculate_liquidation(&mut self) {
        if self.leverage > 1.0 {
            let margin_ratio = 1.0 / self.leverage;
            self.liquidation_price = self.entry_price * (1.0 - margin_ratio);
        }
    }
}

/// Portfolio calculation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PortfolioValue {
    pub total_value_usd: f64,
    pub positions: Vec<Position>,
    pub cash_usd: f64,
    pub daily_pnl: f64,
    pub daily_pnl_pct: f64,
    pub total_pnl: f64,
    pub total_pnl_pct: f64,
}

impl PortfolioValue {
    pub fn calculate_total(&mut self) {
        let positions_value: f64 = self.positions.iter()
            .map(|p| p.size * p.current_price)
            .sum();
        
        self.total_value_usd = positions_value + self.cash_usd;
    }
}

/// High-performance calculator engine
pub struct Calculator {
    precision: Precision,
    cache: Arc<RwLock<CalculationCache>>,
}

struct CalculationCache {
    prices: std::collections::HashMap<String, f64>,
    calculations: std::collections::HashMap<String, f64>,
}

impl Calculator {
    pub fn new(precision: Precision) -> Self {
        Self {
            precision,
            cache: Arc::new(RwLock::new(CalculationCache {
                prices: std::collections::HashMap::new(),
                calculations: std::collections::HashMap::new(),
            })),
        }
    }

    pub fn calculate_swap(
        &self,
        from_token: &str,
        to_token: &str,
        from_amount: f64,
        price_quote: &PriceQuote,
        gas_price: f64,
    ) -> SwapCalculation {
        let scale = self.precision.scale() as f64;
        
        let to_amount = from_amount * price_quote.mid_price;
        let rounded_output = (to_amount * scale).round() / scale;
        
        let price_impact = if from_amount > 1000000.0 {
            from_amount.sqrt() * 0.0001
        } else {
            0.0
        };
        
        let price_impact_bps = price_impact * 10000.0;
        
        let gas_estimate = 150000.0;
        let gas_cost = gas_estimate * gas_price;
        
        let total_cost_usd = gas_cost * price_quote.mid_price;
        
        SwapCalculation {
            from_token: from_token.to_string(),
            to_token: to_token.to_string(),
            from_amount,
            to_amount: rounded_output,
            price_impact,
            price_impact_bps,
            gas_estimate,
            gas_cost_usd: total_cost_usd,
            total_cost_usd,
            route: vec![from_token.to_string(), to_token.to_string()],
            expected_output: rounded_output,
            minimum_output: rounded_output * 0.995,
        }
    }

    pub fn calculate_position(
        &self,
        symbol: &str,
        size: f64,
        entry_price: f64,
        current_price: f64,
        leverage: f64,
    ) -> Position {
        let margin = size * entry_price / leverage;
        
        let mut position = Position {
            symbol: symbol.to_string(),
            size,
            entry_price,
            current_price,
            unrealized_pnl: 0.0,
            unrealized_pnl_pct: 0.0,
            leverage,
            margin,
            liquidation_price: 0.0,
            risk_level: RiskLevel::Safe,
        };
        
        position.calculate_pnl();
        position.calculate_liquidation();
        
        position
    }

    pub fn calculate_portfolio(
        &self,
        positions: Vec<Position>,
        cash_usd: f64,
    ) -> PortfolioValue {
        let mut portfolio = PortfolioValue {
            total_value_usd: 0.0,
            positions,
            cash_usd,
            daily_pnl: 0.0,
            daily_pnl_pct: 0.0,
            total_pnl: 0.0,
            total_pnl_pct: 0.0,
        };
        
        portfolio.calculate_total();
        
        portfolio.total_pnl = portfolio.positions.iter()
            .map(|p| p.unrealized_pnl)
            .sum();
        
        if portfolio.total_value_usd > 0.0 {
            portfolio.total_pnl_pct = (portfolio.total_pnl / portfolio.total_value_usd) * 100.0;
        }
        
        portfolio
    }

    pub fn calculate_apy(apr: f64, compounding_frequency: u32) -> f64 {
        let n = compounding_frequency as f64;
        let base = 1.0 + apr / n;
        base.powf(n) - 1.0
    }

    pub fn calculate_compound(
        principal: f64,
        rate: f64,
        periods: u32,
        compounds_per_period: u32,
    ) -> f64 {
        let r = rate / compounds_per_period as f64;
        let n = (compounds_per_period * periods) as f64;
        principal * (1.0 + r).powf(n)
    }

    pub fn calculate_present_value(
        future_value: f64,
        rate: f64,
        periods: u32,
    ) -> f64 {
        future_value / (1.0 + rate).powf(periods as f64)
    }

    pub fn calculate_npv(
        cash_flows: &[f64],
        discount_rate: f64,
    ) -> f64 {
        cash_flows.iter()
            .enumerate()
            .map(|(i, &cf)| cf / (1.0 + discount_rate).powf(i as f64))
            .sum()
    }

    pub fn calculate_var(
        returns: &[f64],
        confidence: f64,
    ) -> f64 {
        if returns.is_empty() { return 0.0; }
        
        let mut sorted = returns.to_vec();
        sorted.sort_by(|a, b| a.partial_cmp(b).unwrap());
        
        let index = ((1.0 - confidence) * returns.len() as f64) as usize;
        sorted[index]
    }

    pub fn calculate_sharpe_ratio(
        returns: &[f64],
        risk_free_rate: f64,
    ) -> f64 {
        if returns.is_empty() { return 0.0; }
        
        let avg_return: f64 = returns.iter().sum::<f64>() / returns.len() as f64;
        
        let variance: f64 = returns.iter()
            .map(|r| (r - avg_return).powi(2))
            .sum::<f64>() / returns.len() as f64;
        
        let std_dev = variance.sqrt();
        
        if std_dev == 0.0 { return 0.0; }
        
        (avg_return - risk_free_rate) / std_dev
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_swap_calculation() {
        let calc = Calculator::new(Precision::High);
        
        let quote = PriceQuote {
            symbol: "ETH/USDT".to_string(),
            bid_price: 3500.0,
            ask_price: 3501.0,
            mid_price: 3500.5,
            spread: 1.0,
            spread_bps: 0.286,
            timestamp: 0,
            confidence: 0.99,
            source: "aggregator".to_string(),
        };
        
        let swap = calc.calculate_swap("ETH", "USDT", 1.0, &quote, 30.0);
        
        assert!(swap.to_amount > 0.0);
        assert!(swap.gas_cost_usd > 0.0);
    }

    #[test]
    fn test_position_calculation() {
        let calc = Calculator::new(Precision::High);
        
        let position = calc.calculate_position("ETH", 1.0, 3500.0, 3600.0, 10.0);
        
        assert!(position.unrealized_pnl > 0.0);
        assert!(position.liquidation_price > 0.0);
    }

    #[test]
    fn test_apy_calculation() {
        let apy = Calculator::calculate_apy(0.10, 365);
        assert!(apy > 0.10);
    }
}
