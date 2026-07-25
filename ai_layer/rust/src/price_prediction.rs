//! TigerWallet AI Price Prediction Engine - Rust Implementation
//! High-performance, ultra-low latency ML-based price prediction

use std::collections::VecDeque;
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH};
use serde::{Deserialize, Serialize};

// ============================================================================
// DATA STRUCTURES
// ============================================================================

/// Price point from market data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PricePoint {
    pub timestamp: u64,
    pub open: f64,
    pub high: f64,
    pub low: f64,
    pub close: f64,
    pub volume: f64,
}

/// Price prediction result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Prediction {
    pub symbol: String,
    pub current_price: f64,
    pub predicted_price: f64,
    pub confidence: f64,
    pub direction: PredictionDirection,
    pub timeframe: String,
    pub timestamp: u64,
    pub factors: Vec<String>,
    pub model_version: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum PredictionDirection {
    Up,
    Down,
    Neutral,
}

/// Market signal
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketSignal {
    pub symbol: String,
    pub signal_type: SignalType,
    pub strength: f64,
    pub reason: String,
    pub timestamp: u64,
    pub indicators: std::collections::HashMap<String, f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum SignalType {
    Buy,
    Sell,
    Hold,
}

/// Technical indicators
#[derive(Debug, Clone)]
pub struct TechnicalIndicators {
    prices: VecDeque<f64>,
    highs: VecDeque<f64>,
    lows: VecDeque<f64>,
    volumes: VecDeque<f64>,
}

impl TechnicalIndicators {
    pub fn new() -> Self {
        Self {
            prices: VecDeque::new(),
            highs: VecDeque::new(),
            lows: VecDeque::new(),
            volumes: VecDeque::new(),
        }
    }

    pub fn add_data_point(&mut self, point: &PricePoint) {
        if self.prices.len() >= 5000 {
            self.prices.pop_front();
            self.highs.pop_front();
            self.lows.pop_front();
            self.volumes.pop_front();
        }
        self.prices.push_back(point.close);
        self.highs.push_back(point.high);
        self.lows.push_back(point.low);
        self.volumes.push_back(point.volume);
    }

    /// Simple Moving Average
    pub fn sma(&self, period: usize) -> f64 {
        if self.prices.is_empty() {
            return 0.0;
        }
        let period = period.min(self.prices.len());
        self.prices.iter().rev().take(period).sum::<f64>() / period as f64
    }

    /// Exponential Moving Average
    pub fn ema(&self, period: usize) -> f64 {
        if self.prices.is_empty() {
            return 0.0;
        }
        let period = period.min(self.prices.len());
        let multiplier = 2.0 / (period as f64 + 1.0);
        
        let mut ema = *self.prices.front().unwrap();
        for price in self.prices.iter().skip(1).take(period) {
            ema = (price * multiplier) + (ema * (1.0 - multiplier));
        }
        ema
    }

    /// Relative Strength Index
    pub fn rsi(&self, period: usize) -> f64 {
        if self.prices.len() < period + 1 {
            return 50.0;
        }
        
        let period = period.min(self.prices.len() - 1);
        let mut gains = 0.0;
        let mut losses = 0.0;
        
        for i in 1..=period {
            let change = self.prices[i] - self.prices[i-1];
            if change > 0.0 {
                gains += change;
            } else {
                losses += change.abs();
            }
        }
        
        let avg_gain = gains / period as f64;
        let avg_loss = losses / period as f64;
        
        if avg_loss == 0.0 {
            return 100.0;
        }
        
        let rs = avg_gain / avg_loss;
        100.0 - (100.0 / (1.0 + rs))
    }

    /// MACD (Moving Average Convergence Divergence)
    pub fn macd(&self, fast: usize, slow: usize) -> (f64, f64, f64) {
        let fast_ema = self.ema(fast);
        let slow_ema = self.ema(slow);
        
        let macd_line = fast_ema - slow_ema;
        let signal_line = macd_line * 0.9; // Simplified
        let histogram = macd_line - signal_line;
        
        (macd_line, signal_line, histogram)
    }

    /// Bollinger Bands
    pub fn bollinger_bands(&self, period: usize, std_dev: f64) -> (f64, f64, f64) {
        if self.prices.is_empty() {
            return (0.0, 0.0, 0.0);
        }
        
        let period = period.min(self.prices.len());
        let sma = self.sma(period);
        
        let variance = self.prices.iter()
            .rev()
            .take(period)
            .map(|x| (x - sma).powi(2))
            .sum::<f64>() / period as f64;
        
        let std = variance.sqrt();
        
        (sma + std_dev * std, sma, sma - std_dev * std)
    }

    /// Average True Range
    pub fn atr(&self, period: usize) -> f64 {
        if self.highs.len() < 2 {
            return 0.0;
        }
        
        let period = period.min(self.highs.len() - 1);
        let mut true_ranges = Vec::new();
        
        for i in 1..=period {
            let high_low = self.highs[i] - self.lows[i];
            let high_close = (self.highs[i] - self.closes[i-1]).abs();
            let low_close = (self.lows[i] - self.closes[i-1]).abs();
            true_ranges.push(high_low.max(high_close).max(low_close));
        }
        
        true_ranges.iter().sum::<f64>() / true_ranges.len() as f64
    }

    /// Stochastic Oscillator
    pub fn stochastic(&self, period: usize) -> (f64, f64) {
        if self.prices.len() < period {
            return (50.0, 50.0);
        }
        
        let period = period.min(self.prices.len());
        let highest_high = self.highs.iter().rev().take(period).fold(f64::MIN, |a, &b| a.max(b));
        let lowest_low = self.lows.iter().rev().take(period).fold(f64::MAX, |a, &b| a.min(b));
        let current_close = *self.prices.back().unwrap();
        
        if highest_high == lowest_low {
            return (50.0, 50.0);
        }
        
        let k = 100.0 * (current_close - lowest_low) / (highest_high - lowest_low);
        (k, k * 0.9)
    }

    /// Price momentum
    pub fn momentum(&self, period: usize) -> f64 {
        if self.prices.len() < period + 1 {
            return 0.0;
        }
        
        let current = self.prices.back().unwrap();
        let past = self.prices.iter().nth(self.prices.len() - period - 1).unwrap();
        
        (current - past) / past
    }

    /// Volume ratio
    pub fn volume_ratio(&self, period: usize) -> f64 {
        if self.volumes.is_empty() || period == 0 {
            return 1.0;
        }
        
        let period = period.min(self.volumes.len());
        let avg_volume: f64 = self.volumes.iter().rev().take(period).sum::<f64>() / period as f64;
        
        self.volumes.back().unwrap() / avg_volume
    }
}

impl Default for TechnicalIndicators {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================================================
// PRICE PREDICTION ENGINE
// ============================================================================

/// High-performance price prediction engine
pub struct PricePredictionEngine {
    indicators: RwLock<TechnicalIndicators>,
    supported_pairs: Vec<String>,
    cache: RwLock<std::collections::HashMap<String, Prediction>>,
    cache_ttl: u64,
}

impl PricePredictionEngine {
    pub fn new() -> Self {
        Self {
            indicators: RwLock::new(TechnicalIndicators::new()),
            supported_pairs: vec![
                "BTCUSDT".to_string(),
                "ETHUSDT".to_string(),
                "BNBUSDT".to_string(),
                "SOLUSDT".to_string(),
                "XRPUSDT".to_string(),
                "ADAUSDT".to_string(),
                "DOGEUSDT".to_string(),
                "AVAXUSDT".to_string(),
                "DOTUSDT".to_string(),
                "MATICUSDT".to_string(),
                "LINKUSDT".to_string(),
                "UNIUSDT".to_string(),
                "ATOMUSDT".to_string(),
                "LTCUSDT".to_string(),
                "ETCUSDT".to_string(),
            ],
            cache: RwLock::new(std::collections::HashMap::new()),
            cache_ttl: 60,
        }
    }

    /// Add price data
    pub fn add_price_data(&self, point: PricePoint) {
        self.indicators.write().add_data_point(&point);
    }

    /// Generate prediction
    pub fn predict(&self, symbol: &str, timeframe: &str) -> Prediction {
        let indicators = self.indicators.read();
        let current_time = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        // Check cache
        let cache_key = format!("{}_{}", symbol, timeframe);
        if let Ok(cache) = self.cache.read() {
            if let Some(pred) = cache.get(&cache_key) {
                if current_time - pred.timestamp < self.cache_ttl {
                    return pred.clone();
                }
            }
        }
        
        // Generate prediction
        let prediction = self.generate_prediction(symbol, timeframe, &indicators, current_time);
        
        // Cache result
        if let Ok(mut cache) = self.cache.write() {
            cache.insert(cache_key, prediction.clone());
        }
        
        prediction
    }

    fn generate_prediction(&self, symbol: &str, timeframe: &str, indicators: &TechnicalIndicators, timestamp: u64) -> Prediction {
        let current_price = *indicators.prices.back().unwrap_or(&0.0);
        
        if current_price == 0.0 {
            return Prediction {
                symbol: symbol.to_string(),
                current_price: 0.0,
                predicted_price: 0.0,
                confidence: 0.0,
                direction: PredictionDirection::Neutral,
                timeframe: timeframe.to_string(),
                timestamp,
                factors: vec!["insufficient_data".to_string()],
                model_version: "2.0.0".to_string(),
            };
        }
        
        // Extract features
        let rsi = indicators.rsi(14);
        let (macd, macd_signal, _) = indicators.macd(12, 26);
        let sma_20 = indicators.sma(20);
        let sma_50 = indicators.sma(50);
        let (bb_upper, bb_middle, bb_lower) = indicators.bollinger_bands(20, 2.0);
        let momentum = indicators.momentum(10);
        let volume_ratio = indicators.volume_ratio(20);
        
        // Calculate confidence
        let mut confidence = 0.5;
        let mut factors = Vec::new();
        
        // RSI alignment
        if rsi < 30.0 {
            confidence += 0.1;
            factors.push("rsi_oversold".to_string());
        } else if rsi > 70.0 {
            confidence += 0.1;
            factors.push("rsi_overbought".to_string());
        }
        
        // MACD alignment
        if macd > macd_signal {
            confidence += 0.1;
            factors.push("macd_bullish".to_string());
        } else {
            confidence -= 0.1;
            factors.push("macd_bearish".to_string());
        }
        
        // Trend alignment
        if sma_20 > sma_50 {
            confidence += 0.15;
            factors.push("trend_bullish".to_string());
        } else {
            confidence -= 0.15;
            factors.push("trend_bearish".to_string());
        }
        
        // Volume confirmation
        if volume_ratio > 1.5 {
            confidence += 0.1;
            factors.push("high_volume".to_string());
        }
        
        // Clamp confidence
        confidence = confidence.max(0.1).min(0.95);
        
        // Generate prediction
        let predicted_change = self.ensemble_prediction(
            rsi, macd, macd_signal, sma_20, sma_50, 
            bb_upper, bb_middle, bb_lower, momentum, volume_ratio,
            timeframe
        );
        
        let predicted_price = current_price * (1.0 + predicted_change);
        
        let direction = if predicted_change > 0.01 {
            PredictionDirection::Up
        } else if predicted_change < -0.01 {
            PredictionDirection::Down
        } else {
            PredictionDirection::Neutral
        };
        
        Prediction {
            symbol: symbol.to_string(),
            current_price,
            predicted_price,
            confidence,
            direction,
            timeframe: timeframe.to_string(),
            timestamp,
            factors,
            model_version: "2.0.0".to_string(),
        }
    }

    fn ensemble_prediction(
        &self,
        rsi: f64,
        macd: f64,
        macd_signal: f64,
        sma_20: f64,
        sma_50: f64,
        bb_upper: f64,
        bb_middle: f64,
        bb_lower: f64,
        momentum: f64,
        volume_ratio: f64,
        timeframe: &str,
    ) -> f64 {
        let mut signals = Vec::new();
        
        // RSI signal
        if rsi < 30.0 {
            signals.push(0.03);
        } else if rsi < 40.0 {
            signals.push(0.01);
        } else if rsi > 70.0 {
            signals.push(-0.03);
        } else if rsi > 60.0 {
            signals.push(-0.01);
        }
        
        // MACD signal
        if macd > macd_signal {
            signals.push(0.02);
        } else {
            signals.push(-0.02);
        }
        
        // Trend signal
        if sma_20 > sma_50 {
            signals.push(0.015);
        } else {
            signals.push(-0.015);
        }
        
        // Bollinger Band signal
        if bb_middle > 0.0 {
            let bb_width = (bb_upper - bb_lower) / bb_middle;
            if bb_width < 0.02 {
                signals.push(0.01);
            }
        }
        
        // Momentum signal
        signals.push(momentum * 0.5);
        
        // Volume confirmation
        if volume_ratio > 1.5 {
            signals.push(0.01);
        }
        
        // Timeframe adjustment
        let multiplier = match timeframe {
            "5m" => 0.1,
            "15m" => 0.2,
            "1h" => 0.5,
            "4h" => 1.0,
            "1d" => 2.0,
            _ => 0.5,
        };
        
        if signals.is_empty() {
            return 0.0;
        }
        
        let sum: f64 = signals.iter().sum();
        (sum / signals.len() as f64) * multiplier
    }

    /// Get market signal
    pub fn get_market_signal(&self, symbol: &str) -> MarketSignal {
        let prediction = self.predict(symbol, "1h");
        
        let (signal_type, strength, reason) = if prediction.confidence < 0.3 {
            (SignalType::Hold, 0.3, "Low confidence prediction".to_string())
        } else if prediction.direction == PredictionDirection::Up && prediction.confidence > 0.6 {
            (SignalType::Buy, prediction.confidence, 
             format!("Strong buy: {}", prediction.factors.join(", ")))
        } else if prediction.direction == PredictionDirection::Down && prediction.confidence > 0.6 {
            (SignalType::Sell, prediction.confidence,
             format!("Strong sell: {}", prediction.factors.join(", ")))
        } else {
            (SignalType::Hold, 0.5,
             format!("Hold: {}", prediction.factors.join(", ")))
        };
        
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        let mut indicators = std::collections::HashMap::new();
        indicators.insert("confidence".to_string(), prediction.confidence);
        
        MarketSignal {
            symbol: symbol.to_string(),
            signal_type,
            strength,
            reason,
            timestamp,
            indicators,
        }
    }

    /// Batch predict
    pub fn batch_predict(&self, symbols: &[String], timeframe: &str) -> Vec<Prediction> {
        symbols.iter().map(|s| self.predict(s, timeframe)).collect()
    }

    /// Get supported pairs
    pub fn get_supported_pairs(&self) -> Vec<String> {
        self.supported_pairs.clone()
    }
}

impl Default for PricePredictionEngine {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================================================
// SCAM DETECTION ENGINE
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenAnalysis {
    pub address: String,
    pub risk_score: u32,
    pub risk_level: String,
    pub flags: Vec<String>,
    pub recommendation: String,
}

/// Scam detection engine
pub struct ScamDetectionEngine {
    suspicious_patterns: Vec<String>,
}

impl ScamDetectionEngine {
    pub fn new() -> Self {
        Self {
            suspicious_patterns: vec![
                "honeypot".to_string(),
                "infinite_mint".to_string(),
                "hidden_owner".to_string(),
                "fake_audit".to_string(),
                "rug_pull".to_string(),
            ],
        }
    }

    /// Analyze token for potential scams
    pub fn analyze_token(&self, address: &str, data: &TokenAnalysisData) -> TokenAnalysis {
        let mut risk_score = 0u32;
        let mut flags = Vec::new();
        
        if data.is_honeypot {
            risk_score += 50;
            flags.push("honeypot_detected".to_string());
        }
        
        if data.can_mint {
            risk_score += 30;
            flags.push("infinite_mint".to_string());
        }
        
        if data.owner_percent > 50 {
            risk_score += 40;
            flags.push("high_owner_supply".to_string());
        }
        
        if !data.is_verified {
            risk_score += 10;
            flags.push("unverified_contract".to_string());
        }
        
        if !data.liquidity_locked {
            risk_score += 30;
            flags.push("unlocked_liquidity".to_string());
        }
        
        if data.trade_tax > 10 {
            risk_score += 20;
            flags.push("high_tax".to_string());
        }
        
        let risk_level = if risk_score < 20 {
            "low".to_string()
        } else if risk_score < 50 {
            "medium".to_string()
        } else {
            "high".to_string()
        };
        
        let recommendation = if risk_score > 50 {
            "avoid".to_string()
        } else if risk_score > 20 {
            "caution".to_string()
        } else {
            "safe".to_string()
        };
        
        TokenAnalysis {
            address: address.to_string(),
            risk_score,
            risk_level,
            flags,
            recommendation,
        }
    }
}

impl Default for ScamDetectionEngine {
    fn default() -> Self {
        Self::new()
    }
}

/// Token data for analysis
#[derive(Debug, Clone)]
pub struct TokenAnalysisData {
    pub is_honeypot: bool,
    pub can_mint: bool,
    pub owner_percent: f64,
    pub is_verified: bool,
    pub liquidity_locked: bool,
    pub trade_tax: f64,
}

// ============================================================================
// UNIT TESTS
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sma() {
        let mut indicators = TechnicalIndicators::new();
        
        for i in 0..30 {
            let point = PricePoint {
                timestamp: i,
                open: 100.0 + i as f64,
                high: 105.0 + i as f64,
                low: 95.0 + i as f64,
                close: 100.0 + i as f64,
                volume: 1000.0,
            };
            indicators.add_data_point(&point);
        }
        
        let sma = indicators.sma(20);
        assert!(sma > 0.0);
    }

    #[test]
    fn test_prediction_engine() {
        let engine = PricePredictionEngine::new();
        
        // Add some price data
        for i in 0..50 {
            let point = PricePoint {
                timestamp: i,
                open: 50000.0 + (i as f64 * 10.0),
                high: 50100.0 + (i as f64 * 10.0),
                low: 49900.0 + (i as f64 * 10.0),
                close: 50000.0 + (i as f64 * 10.0),
                volume: 1000.0,
            };
            engine.add_price_data(point);
        }
        
        let prediction = engine.predict("BTCUSDT", "1h");
        
        assert!(!prediction.symbol.is_empty());
        assert!(prediction.model_version == "2.0.0");
    }

    #[test]
    fn test_scam_detection() {
        let engine = ScamDetectionEngine::new();
        
        let data = TokenAnalysisData {
            is_honeypot: false,
            can_mint: false,
            owner_percent: 10.0,
            is_verified: true,
            liquidity_locked: true,
            trade_tax: 1.0,
        };
        
        let result = engine.analyze_token("0x123", &data);
        
        assert!(result.risk_score < 20);
        assert!(result.recommendation == "safe");
    }
}
