//! Price Prediction Service

use crate::error::Error;
use crate::models::*;
use chrono::{DateTime, Utc};
use rand::Rng;

/// Price Prediction Service
pub struct PricePredictionService {
    model_weights: Vec<f64>,
}

impl PricePredictionService {
    pub fn new() -> Self {
        Self {
            model_weights: vec![0.3, 0.25, 0.2, 0.15, 0.1],
        }
    }

    pub fn predict(&self, symbol: &str, current_price: f64) -> Result<PricePrediction, Error> {
        let mut rng = rand::thread_rng();
        
        let trend_change: f64 = rng.gen_range(-0.05..0.05);
        let predicted_change = current_price * trend_change;
        let predicted_price = current_price + predicted_change;
        let confidence = rng.gen_range(0.6..0.95);
        
        let trend = if trend_change > 0.01 {
            Trend::Bullish
        } else if trend_change < -0.01 {
            Trend::Bearish
        } else {
            Trend::Neutral
        };
        
        Ok(PricePrediction {
            symbol: symbol.to_string(),
            current_price,
            predicted_price,
            confidence,
            trend,
            timestamp: Utc::now(),
        })
    }

    pub fn batch_predict(&self, symbols: &[(&str, f64)]) -> Vec<PricePrediction> {
        symbols.iter()
            .map(|(symbol, price)| {
                self.predict(symbol, *price).unwrap_or_else(|_| PricePrediction {
                    symbol: symbol.to_string(),
                    current_price: *price,
                    predicted_price: *price,
                    confidence: 0.0,
                    trend: Trend::Neutral,
                    timestamp: Utc::now(),
                })
            })
            .collect()
    }
}

impl Default for PricePredictionService {
    fn default() -> Self {
        Self::new()
    }
}