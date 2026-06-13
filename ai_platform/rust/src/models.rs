//! Data models for AI Platform

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// Price Prediction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PricePrediction {
    pub symbol: String,
    pub current_price: f64,
    pub predicted_price: f64,
    pub confidence: f64,
    pub trend: Trend,
    pub timestamp: DateTime<Utc>,
}

/// Trend
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum Trend {
    Bullish,
    Bearish,
    Neutral,
}

/// Prediction Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PredictionRequest {
    pub symbol: String,
    pub timeframe: u64,
}

/// Model Features
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModelFeatures {
    pub price_history: Vec<f64>,
    pub volume_history: Vec<f64>,
    pub volatility: f64,
    pub rsi: f64,
    pub moving_avg: f64,
}

/// Training Data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrainingData {
    pub features: ModelFeatures,
    pub target: f64,
}