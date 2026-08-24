//! Gas Optimizer Module
//! 
//! Provides intelligent gas price optimization

use crate::{GasOptimization, AgentError};
use chrono::{DateTime, Utc, Duration};

/// Gas Optimizer
pub struct GasOptimizer {
    /// Historical gas data
    gas_history: Vec<GasDataPoint>,
    /// Current gas prices
    current_gas: Option<GasPrices>,
}

#[derive(Debug, Clone)]
struct GasDataPoint {
    timestamp: DateTime<Utc>,
    gas_price_gwei: f64,
    block_number: u64,
}

#[derive(Debug, Clone)]
pub struct GasPrices {
    pub low_gwei: f64,
    pub medium_gwei: f64,
    pub high_gwei: f64,
    pub base_fee_gwei: f64,
    pub timestamp: DateTime<Utc>,
}

impl Default for GasOptimizer {
    fn default() -> Self {
        Self::new()
    }
}

impl GasOptimizer {
    /// Create new optimizer
    pub fn new() -> Self {
        Self {
            gas_history: Vec::new(),
            current_gas: None,
        }
    }
    
    /// Update current gas prices
    pub fn update_gas_prices(&mut self, prices: GasPrices) {
        self.current_gas = Some(prices.clone());
        
        // Add to history
        self.gas_history.push(GasDataPoint {
            timestamp: Utc::now(),
            gas_price_gwei: prices.medium_gwei,
            block_number: 0, // Would be from blockchain
        });
        
        // Keep only last 1000 points
        if self.gas_history.len() > 1000 {
            self.gas_history.drain(0..self.gas_history.len() - 1000);
        }
    }
    
    /// Get optimal gas price
    pub fn get_optimal_gas(&self) -> Result<GasOptimization, AgentError> {
        let current = self.current_gas.clone()
            .ok_or_else(|| AgentError::AnalysisError("No gas data available".to_string()))?;
        
        // Analyze historical data to find patterns
        let avg_gas = self.calculate_average_gas();
        let current_deviation = current.medium_gwei - avg_gas;
        
        // Determine optimal strategy
        let (suggested_gas, delay, savings) = if current_deviation > avg_gas * 0.5 {
            // Gas is high, suggest waiting
            (avg_gas * 0.8, 1800, 30.0) // Wait 30 min, save 30%
        } else if current_deviation > avg_gas * 0.2 {
            // Gas is slightly high
            (current.medium_gwei * 0.9, 300, 10.0) // Wait 5 min, save 10%
        } else {
            // Gas is good
            (current.medium_gwei, 0, 0.0) // Send now
        };
        
        let optimal_time = if delay > 0 {
            Utc::now() + Duration::seconds(delay)
        } else {
            Utc::now()
        };
        
        Ok(GasOptimization {
            current_gas_price_gwei: current.medium_gwei,
            suggested_gas_price_gwei: suggested_gas,
            optimal_time_to_send: optimal_time,
            estimated_savings_percent: savings,
            recommended_delay_seconds: delay,
        })
    }
    
    /// Calculate average gas price from history
    fn calculate_average_gas(&self) -> f64 {
        if self.gas_history.is_empty() {
            return 50.0; // Default fallback
        }
        
        // Use last 24 hours of data
        let cutoff = Utc::now() - Duration::hours(24);
        let recent: Vec<_> = self.gas_history.iter()
            .filter(|p| p.timestamp > cutoff)
            .collect();
        
        if recent.is_empty() {
            return self.gas_history.iter()
                .map(|p| p.gas_price_gwei)
                .sum::<f64>() / self.gas_history.len() as f64;
        }
        
        recent.iter()
            .map(|p| p.gas_price_gwei)
            .sum::<f64>() / recent.len() as f64
    }
    
    /// Predict gas prices for next few hours
    pub fn predict_gas(&self, hours_ahead: i64) -> Result<Vec<(DateTime<Utc>, f64)>, AgentError> {
        if self.gas_history.len() < 10 {
            return Err(AgentError::PredictionError("Insufficient data for prediction".to_string()));
        }
        
        // Simple linear regression for prediction
        let n = self.gas_history.len();
        let sum_t: i64 = (0..n).map(|i| i as i64).sum();
        let sum_gas: f64 = self.gas_history.iter().map(|p| p.gas_price_gwei).sum();
        let sum_t2: i64 = (0..n).map(|i| i as i64 * i as i64).sum();
        let sum_tgas: f64 = self.gas_history.iter()
            .enumerate()
            .map(|(i, p)| (i as f64) * p.gas_price_gwei)
            .sum();
        
        let slope = (n as f64 * sum_tgas - sum_t as f64 * sum_gas) / 
                    (n as f64 * sum_t2 as f64 - sum_t as f64 * sum_t as f64);
        let intercept = (sum_gas - slope * sum_t as f64) / n as f64;
        
        let mut predictions = Vec::new();
        let now = Utc::now();
        
        for i in 1..=hours_ahead {
            let prediction = intercept + slope * (n as f64 + i as f64 * 6.0); // Assuming 10 min blocks
            let timestamp = now + Duration::hours(i);
            
            // Gas prices can't be negative
            predictions.push((timestamp, prediction.max(1.0)));
        }
        
        Ok(predictions)
    }
    
    /// Get fee estimates for different urgency levels
    pub fn get_fee_estimates(&self, gas_limit: u64) -> Result<FeeEstimates, AgentError> {
        let current = self.current_gas.clone()
            .ok_or_else(|| AgentError::AnalysisError("No gas data available".to_string()))?;
        
        Ok(FeeEstimates {
            slow: FeeEstimate {
                gas_price_gwei: current.low_gwei,
                total_fee_wei: (current.low_gwei * 1e9) as u64 * gas_limit,
                estimated_time_seconds: 300,
            },
            average: FeeEstimate {
                gas_price_gwei: current.medium_gwei,
                total_fee_wei: (current.medium_gwei * 1e9) as u64 * gas_limit,
                estimated_time_seconds: 60,
            },
            fast: FeeEstimate {
                gas_price_gwei: current.high_gwei,
                total_fee_wei: (current.high_gwei * 1e9) as u64 * gas_limit,
                estimated_time_seconds: 15,
            },
        })
    }
}

#[derive(Debug, Clone)]
pub struct FeeEstimates {
    pub slow: FeeEstimate,
    pub average: FeeEstimate,
    pub fast: FeeEstimate,
}

#[derive(Debug, Clone)]
pub struct FeeEstimate {
    pub gas_price_gwei: f64,
    pub total_fee_wei: u64,
    pub estimated_time_seconds: u64,
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_gas_optimizer() {
        let mut optimizer = GasOptimizer::new();
        
        // Add some gas data
        optimizer.update_gas_prices(GasPrices {
            low_gwei: 20.0,
            medium_gwei: 30.0,
            high_gwei: 50.0,
            base_fee_gwei: 25.0,
            timestamp: Utc::now(),
        });
        
        optimizer.update_gas_prices(GasPrices {
            low_gwei: 25.0,
            medium_gwei: 35.0,
            high_gwei: 55.0,
            base_fee_gwei: 30.0,
            timestamp: Utc::now(),
        });
        
        let result = optimizer.get_optimal_gas();
        assert!(result.is_ok());
    }
    
    #[test]
    fn test_fee_estimates() {
        let mut optimizer = GasOptimizer::new();
        
        optimizer.update_gas_prices(GasPrices {
            low_gwei: 20.0,
            medium_gwei: 30.0,
            high_gwei: 50.0,
            base_fee_gwei: 25.0,
            timestamp: Utc::now(),
        });
        
        let fees = optimizer.get_fee_estimates(21000).unwrap();
        
        assert!(fees.slow.total_fee_wei < fees.average.total_fee_wei);
        assert!(fees.average.total_fee_wei < fees.fast.total_fee_wei);
    }
}
