//! Price Predictor Module
//! 
//! Provides AI-powered price predictions

use crate::{PricePrediction, PricePoint, PriceTrend, AgentError};
use chrono::{DateTime, Utc, Duration};

/// Price Predictor
pub struct PricePredictor {
    /// Historical price data
    price_history: Vec<PriceDataPoint>,
}

#[derive(Debug, Clone)]
struct PriceDataPoint {
    timestamp: DateTime<Utc>,
    price: f64,
    volume: f64,
}

impl Default for PricePredictor {
    fn default() -> Self {
        Self::new()
    }
}

impl PricePredictor {
    /// Create new predictor
    pub fn new() -> Self {
        Self {
            price_history: Vec::new(),
        }
    }
    
    /// Add price data point
    pub fn add_price_data(&mut self, price: f64, volume: f64) {
        self.price_history.push(PriceDataPoint {
            timestamp: Utc::now(),
            price,
            volume,
        });
        
        // Keep last 1000 points
        if self.price_history.len() > 1000 {
            self.price_history.drain(0..self.price_history.len() - 1000);
        }
    }
    
    /// Predict price for token
    pub fn predict(&self, token: &str, hours_ahead: i64) -> Result<PricePrediction, AgentError> {
        if self.price_history.len() < 24 {
            return Err(AgentError::PredictionError("Insufficient data for prediction".to_string()));
        }
        
        // Simple moving average + trend analysis
        let recent_prices: Vec<f64> = self.price_history.iter()
            .rev()
            .take(24)
            .map(|p| p.price)
            .collect();
        
        let current_price = recent_prices[0];
        
        // Calculate trend
        let short_avg: f64 = recent_prices.iter().take(6).sum::<f64>() / 6.0;
        let long_avg: f64 = recent_prices.iter().sum::<f64>() / recent_prices.len() as f64;
        
        let trend = if short_avg > long_avg * 1.05 {
            PriceTrend::Bullish
        } else if short_avg < long_avg * 0.95 {
            PriceTrend::Bearish
        } else {
            PriceTrend::Neutral
        };
        
        // Calculate predictions with confidence intervals
        let volatility = self.calculate_volatility();
        let mut predictions = Vec::new();
        
        for hour in 1..=hours_ahead {
            // Simple linear projection with mean reversion
            let drift = (short_avg - long_avg) / long_avg;
            let projected = current_price * (1.0 + drift * hour as f64 * 0.1);
            
            // Add confidence interval based on volatility and time
            let interval = volatility * projected * (hour as f64).sqrt() * 0.5;
            
            predictions.push(PricePoint {
                timestamp: Utc::now() + Duration::hours(hour),
                predicted_price: projected,
                upper_bound: projected + interval,
                lower_bound: (projected - interval).max(0.01),
            });
        }
        
        let confidence = self.calculate_confidence();
        
        Ok(PricePrediction {
            token: token.to_string(),
            current_price,
            predictions,
            trend,
            confidence,
        })
    }
    
    /// Calculate price volatility
    fn calculate_volatility(&self) -> f64 {
        if self.price_history.len() < 2 {
            return 0.1;
        }
        
        let prices: Vec<f64> = self.price_history.iter().map(|p| p.price).collect();
        let mean = prices.iter().sum::<f64>() / prices.len() as f64;
        
        let variance = prices.iter()
            .map(|p| (p - mean).powi(2))
            .sum::<f64>() / prices.len() as f64;
        
        variance.sqrt() / mean
    }
    
    /// Calculate prediction confidence
    fn calculate_confidence(&self) -> f64 {
        // More data = higher confidence
        let data_factor = (self.price_history.len() as f64 / 100.0).min(1.0);
        
        // Lower volatility = higher confidence
        let volatility = self.calculate_volatility();
        let volatility_factor = (1.0 - volatility).max(0.3);
        
        (data_factor * 0.7 + volatility_factor * 0.3).min(0.95)
    }
    
    /// Find support and resistance levels
    pub fn find_support_resistance(&self) -> Result<(Vec<f64>, Vec<f64>), AgentError> {
        if self.price_history.len() < 50 {
            return Err(AgentError::PredictionError("Insufficient data".to_string()));
        }
        
        let mut prices: Vec<f64> = self.price_history.iter().map(|p| p.price).collect();
        prices.sort_by(|a, b| a.partial_cmp(b).unwrap());
        
        // Find clusters for support/resistance
        let support = vec![prices[0], prices[prices.len() / 4]];
        let resistance = vec![prices[prices.len() * 3 / 4], prices[prices.len() - 1]];
        
        Ok((support, resistance))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_price_prediction() {
        let mut predictor = PricePredictor::new();
        
        // Add some price data
        for i in 0..30 {
            predictor.add_price_data(2000.0 + (i as f64 * 10.0), 1000000.0);
        }
        
        let prediction = predictor.predict("ETH", 24);
        assert!(prediction.is_ok());
    }
}
