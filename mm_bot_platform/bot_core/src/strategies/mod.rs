// ============================================================================
// TigerWallet Advanced Trading Strategies
// High-Performance Rust Implementation for New Bot Types
// ============================================================================

use serde::{Deserialize, Serialize};
use std::collections::VecDeque;

/// Price data point for analysis
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceData {
    pub timestamp: i64,
    pub open: f64,
    pub high: f64,
    pub low: f64,
    pub close: f64,
    pub volume: f64,
}

/// Trading signal
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TradingSignal {
    Buy,
    Sell,
    Hold,
    ClosePosition,
}

/// Grid level for grid trading
#[derive(Debug, Clone)]
pub struct GridLevel {
    pub price: f64,
    pub buy_order_id: Option<String>,
    pub sell_order_id: Option<String>,
    pub filled: bool,
}

// ============================================================================
// GRID TRADING STRATEGY
// ============================================================================

pub struct GridTradingStrategy {
    pub levels: Vec<GridLevel>,
    pub grid_count: usize,
    pub grid_spacing_pct: f64,
    pub order_size_usd: f64,
    pub base_price: f64,
}

impl GridTradingStrategy {
    pub fn new(grid_count: usize, grid_spacing_pct: f64, order_size_usd: f64, base_price: f64) -> Self {
        let mut levels = Vec::new();
        
        for i in 0..grid_count {
            let offset = (i as f64 - grid_count as f64 / 2.0) * grid_spacing_pct / 100.0;
            let price = base_price * (1.0 + offset);
            levels.push(GridLevel {
                price,
                buy_order_id: None,
                sell_order_id: None,
                filled: false,
            });
        }
        
        Self {
            levels,
            grid_count,
            grid_spacing_pct,
            order_size_usd,
            base_price,
        }
    }
    
    pub fn generate_signals(&self, current_price: f64) -> Vec<(TradingSignal, f64)> {
        let mut signals = Vec::new();
        
        for level in &self.levels {
            let price_diff = (current_price - level.price).abs() / level.price;
            
            // If price crosses a level
            if price_diff < self.grid_spacing_pct / 200.0 {
                if !level.filled {
                    // Check if we should place orders
                    if current_price < level.price {
                        signals.push((TradingSignal::Buy, level.price));
                    } else if current_price > level.price {
                        signals.push((TradingSignal::Sell, level.price));
                    }
                }
            }
        }
        
        signals
    }
    
    pub fn update_levels(&mut self, new_base_price: f64) {
        let price_change = new_base_price / self.base_price;
        
        for level in &mut self.levels {
            level.price *= price_change;
            // Reset filled status when price moves significantly
            if (new_base_price - self.base_price).abs() / self.base_price > self.grid_spacing_pct / 100.0 {
                level.filled = false;
            }
        }
        
        self.base_price = new_base_price;
    }
}

// ============================================================================
// DCA (DOLLAR-COST AVERAGING) STRATEGY
// ============================================================================

pub struct DcaStrategy {
    pub buy_interval_seconds: i64,
    pub buy_amount_usd: f64,
    pub max_positions: usize,
    pub current_positions: usize,
    pub last_buy_time: i64,
    pub accumulated_amount: f64,
}

impl DcaStrategy {
    pub fn new(buy_interval_hours: i64, buy_amount_usd: f64, max_positions: usize) -> Self {
        Self {
            buy_interval_seconds: buy_interval_hours * 3600,
            buy_amount_usd,
            max_positions,
            current_positions: 0,
            last_buy_time: 0,
            accumulated_amount: 0.0,
        }
    }
    
    pub fn should_buy(&self, current_time: i64) -> bool {
        if self.current_positions >= self.max_positions {
            return false;
        }
        
        current_time - self.last_buy_time >= self.buy_interval_seconds
    }
    
    pub fn execute_buy(&mut self, current_time: i64, current_price: f64) -> Option<(TradingSignal, f64)> {
        if self.should_buy(current_time) {
            let quantity = self.buy_amount_usd / current_price;
            self.last_buy_time = current_time;
            self.current_positions += 1;
            self.accumulated_amount += self.buy_amount_usd;
            Some((TradingSignal::Buy, quantity))
        } else {
            None
        }
    }
    
    pub fn get_average_price(&self) -> f64 {
        if self.current_positions == 0 {
            return 0.0;
        }
        self.accumulated_amount / (self.accumulated_amount / self.buy_amount_usd)
    }
}

// ============================================================================
// MOMENTUM STRATEGY
// ============================================================================

pub struct MomentumStrategy {
    pub lookback_period: usize,
    pub entry_threshold: f64,
    pub exit_threshold: f64,
    pub price_history: VecDeque<f64>,
    pub in_position: bool,
    pub entry_price: f64,
}

impl MomentumStrategy {
    pub fn new(lookback_period: usize, entry_threshold: f64, exit_threshold: f64) -> Self {
        Self {
            lookback_period,
            entry_threshold,
            exit_threshold,
            price_history: VecDeque::new(),
            in_position: false,
            entry_price: 0.0,
        }
    }
    
    pub fn add_price(&mut self, price: f64) {
        self.price_history.push_back(price);
        if self.price_history.len() > self.lookback_period + 10 {
            self.price_history.pop_front();
        }
    }
    
    pub fn calculate_momentum(&self) -> f64 {
        if self.price_history.len() < self.lookback_period {
            return 0.0;
        }
        
        let start_idx = self.price_history.len() - self.lookback_period;
        let start_price = self.price_history[start_idx];
        let end_price = self.price_history[self.price_history.len() - 1];
        
        (end_price - start_price) / start_price
    }
    
    pub fn generate_signal(&mut self, current_price: f64) -> Option<TradingSignal> {
        self.add_price(current_price);
        
        let momentum = self.calculate_momentum();
        
        if !self.in_position && momentum > self.entry_threshold {
            self.in_position = true;
            self.entry_price = current_price;
            return Some(TradingSignal::Buy);
        } else if self.in_position {
            let current_return = (current_price - self.entry_price) / self.entry_price;
            
            if current_return < self.exit_threshold {
                self.in_position = false;
                return Some(TradingSignal::Sell);
            }
        }
        
        Some(TradingSignal::Hold)
    }
}

// ============================================================================
// MEAN REVERSION STRATEGY
// ============================================================================

pub struct MeanReversionStrategy {
    pub lookback_period: usize,
    pub std_dev_threshold: f64,
    pub price_history: VecDeque<f64>,
    pub in_position: bool,
    pub mean_price: f64,
    pub std_dev: f64,
}

impl MeanReversionStrategy {
    pub fn new(lookback_period: usize, std_dev_threshold: f64) -> Self {
        Self {
            lookback_period,
            std_dev_threshold,
            price_history: VecDeque::new(),
            in_position: false,
            mean_price: 0.0,
            std_dev: 0.0,
        }
    }
    
    pub fn calculate_statistics(&mut self) {
        if self.price_history.len() < self.lookback_period {
            return;
        }
        
        let start_idx = self.price_history.len() - self.lookback_period;
        let prices: Vec<f64> = self.price_history.iter()
            .skip(start_idx)
            .cloned()
            .collect();
        
        let sum: f64 = prices.iter().sum();
        self.mean_price = sum / prices.len() as f64;
        
        let variance: f64 = prices.iter()
            .map(|p| (p - self.mean_price).powi(2))
            .sum::<f64>() / prices.len() as f64;
        
        self.std_dev = variance.sqrt();
    }
    
    pub fn generate_signal(&mut self, current_price: f64) -> Option<TradingSignal> {
        self.price_history.push_back(current_price);
        
        if self.price_history.len() > self.lookback_period + 10 {
            self.price_history.pop_front();
        }
        
        self.calculate_statistics();
        
        if self.std_dev == 0.0 {
            return Some(TradingSignal::Hold);
        }
        
        let z_score = (current_price - self.mean_price) / self.std_dev;
        
        if !self.in_position && z_score < -self.std_dev_threshold {
            // Price is below mean by threshold - buy
            self.in_position = true;
            return Some(TradingSignal::Buy);
        } else if self.in_position && z_score > -0.5 {
            // Price returned to mean - sell
            self.in_position = false;
            return Some(TradingSignal::Sell);
        }
        
        Some(TradingSignal::Hold)
    }
}

// ============================================================================
// SCALPING STRATEGY
// ============================================================================

pub struct ScalpingStrategy {
    pub profit_target_pct: f64,
    pub stop_loss_pct: f64,
    pub max_spread_pct: f64,
    pub entry_price: f64,
    pub in_position: bool,
    pub order_book_bid: f64,
    pub order_book_ask: f64,
}

impl ScalpingStrategy {
    pub fn new(profit_target_pct: f64, stop_loss_pct: f64, max_spread_pct: f64) -> Self {
        Self {
            profit_target_pct,
            stop_loss_pct,
            max_spread_pct,
            entry_price: 0.0,
            in_position: false,
            order_book_bid: 0.0,
            order_book_ask: 0.0,
        }
    }
    
    pub fn update_order_book(&mut self, bid: f64, ask: f64) {
        self.order_book_bid = bid;
        self.order_book_ask = ask;
    }
    
    pub fn calculate_spread(&self) -> f64 {
        if self.order_book_ask == 0.0 {
            return 0.0;
        }
        (self.order_book_ask - self.order_book_bid) / self.order_book_bid * 100.0
    }
    
    pub fn generate_signal(&mut self, mid_price: f64) -> Option<TradingSignal> {
        let spread = self.calculate_spread();
        
        // Check if spread is acceptable
        if spread > self.max_spread_pct {
            return Some(TradingSignal::Hold);
        }
        
        if !self.in_position {
            // Enter at bid (buy)
            self.entry_price = self.order_book_bid;
            self.in_position = true;
            return Some(TradingSignal::Buy);
        }
        
        // Check exit conditions
        let current_return = (mid_price - self.entry_price) / self.entry_price * 100.0;
        
        if current_return >= self.profit_target_pct {
            // Take profit
            self.in_position = false;
            return Some(TradingSignal::Sell);
        } else if current_return <= -self.stop_loss_pct {
            // Stop loss
            self.in_position = false;
            return Some(TradingSignal::Sell);
        }
        
        Some(TradingSignal::Hold)
    }
}

// ============================================================================
// AI TRADING STRATEGY (Stub - requires ML model)
// ============================================================================

pub struct AiTradingStrategy {
    pub model_path: String,
    pub prediction_threshold: f64,
    pub price_history: VecDeque<f64>,
    pub in_position: bool,
}

impl AiTradingStrategy {
    pub fn new(model_path: String, prediction_threshold: f64) -> Self {
        Self {
            model_path,
            prediction_threshold,
            price_history: VecDeque::new(),
            in_position: false,
        }
    }
    
    pub fn add_data(&mut self, price: f64) {
        self.price_history.push_back(price);
        if self.price_history.len() > 1000 {
            self.price_history.pop_front();
        }
    }
    
    /// Simulated AI prediction - in production, load actual ML model
    pub fn predict(&self) -> f64 {
        if self.price_history.len() < 50 {
            return 0.5;
        }
        
        // Simple moving average based prediction
        let recent: Vec<f64> = self.price_history.iter().rev().take(20).cloned().collect();
        let older: Vec<f64> = self.price_history.iter().rev().skip(20).take(30).cloned().collect();
        
        let recent_avg: f64 = recent.iter().sum::<f64>() / recent.len() as f64;
        let older_avg: f64 = older.iter().sum::<f64>() / older.len() as f64;
        
        // Return probability between 0 and 1
        let momentum = (recent_avg - older_avg) / older_avg;
        (0.5 + momentum * 10.0).clamp(0.0, 1.0)
    }
    
    pub fn generate_signal(&mut self, current_price: f64) -> Option<TradingSignal> {
        self.add_data(current_price);
        
        let prediction = self.predict();
        
        if prediction > self.prediction_threshold && !self.in_position {
            self.in_position = true;
            return Some(TradingSignal::Buy);
        } else if prediction < (1.0 - self.prediction_threshold) && self.in_position {
            self.in_position = false;
            return Some(TradingSignal::Sell);
        }
        
        Some(TradingSignal::Hold)
    }
}

// ============================================================================
// SIGNAL TRADING STRATEGY
// ============================================================================

pub struct SignalStrategy {
    pub signal_source: String,
    pub signal_endpoint: String,
    pub last_signal: TradingSignal,
    pub in_position: bool,
}

impl SignalStrategy {
    pub fn new(signal_source: String, signal_endpoint: String) -> Self {
        Self {
            signal_source,
            signal_endpoint,
            last_signal: TradingSignal::Hold,
            in_position: false,
        }
    }
    
    pub fn update_signal(&mut self, signal: TradingSignal) {
        self.last_signal = signal;
    }
    
    pub fn generate_signal(&mut self) -> Option<TradingSignal> {
        // In production, fetch from signal endpoint
        // For now, use the last received signal
        match &self.last_signal {
            TradingSignal::Buy if !self.in_position => {
                self.in_position = true;
                Some(TradingSignal::Buy)
            },
            TradingSignal::Sell if self.in_position => {
                self.in_position = false;
                Some(TradingSignal::Sell)
            },
            _ => Some(TradingSignal::Hold),
        }
    }
}

// ============================================================================
// CUSTOM STRATEGY (User-defined)
// ============================================================================

pub struct CustomStrategy {
    pub strategy_code: String,
    pub parameters: std::collections::HashMap<String, String>,
    pub in_position: bool,
}

impl CustomStrategy {
    pub fn new(strategy_code: String, parameters: std::collections::HashMap<String, String>) -> Self {
        Self {
            strategy_code,
            parameters,
            in_position: false,
        }
    }
    
    /// Execute custom strategy logic
    pub fn execute(&mut self, current_price: f64, price_history: &[f64]) -> Option<TradingSignal> {
        // This is a placeholder - in production, this would execute
        // user-defined strategy code from strategy_code field
        
        // Simple example: if price is below 20-period MA, buy
        if price_history.len() >= 20 {
            let ma: f64 = price_history.iter().rev().take(20).sum::<f64>() / 20.0;
            
            if current_price < ma && !self.in_position {
                self.in_position = true;
                return Some(TradingSignal::Buy);
            } else if current_price > ma * 1.05 && self.in_position {
                self.in_position = false;
                return Some(TradingSignal::Sell);
            }
        }
        
        Some(TradingSignal::Hold)
    }
}

// ============================================================================
// FACTORY FUNCTION
// ============================================================================

use crate::bot_types::BotType;

pub fn create_strategy(bot_type: BotType) -> Box<dyn TradingStrategy> {
    match bot_type {
        BotType::GridTrading => Box::new(GridTradingStrategy::new(10, 1.0, 100.0, 1000.0)),
        BotType::DcaBot => Box::new(DcaStrategy::new(24, 100.0, 5)),
        BotType::MomentumBot => Box::new(MomentumStrategy::new(20, 0.02, -0.01)),
        BotType::MeanReversion => Box::new(MeanReversionStrategy::new(50, 2.0)),
        BotType::ScalpingBot => Box::new(ScalpingStrategy::new(0.1, 0.05, 0.2)),
        BotType::AiTradingBot => Box::new(AiTradingStrategy::new("models/default.pt".to_string(), 0.6)),
        BotType::SignalBot => Box::new(SignalStrategy::new("custom".to_string(), "".to_string())),
        BotType::CustomBot => Box::new(CustomStrategy::new("".to_string(), std::collections::HashMap::new())),
        // The classic/MM/MEV bot types do not use the strategy trait (they have
        // their own execution loops in the main bot engine). Return a no-op
        // Hold so the factory is total (never panics).
        _ => Box::new(NoOpStrategy),
    }
}

pub trait TradingStrategy {
    fn generate_signal(&mut self, current_price: f64) -> Option<TradingSignal>;
}

// NoOpStrategy is a total fallback for the classic/MM/MEV bot types whose
// decision logic lives in the main bot engine, not the strategy trait.
pub struct NoOpStrategy;
impl TradingStrategy for NoOpStrategy {
    fn generate_signal(&mut self, _current_price: f64) -> Option<TradingSignal> {
        Some(TradingSignal::Hold)
    }
}

// ---- trait impls delegating to each strategy's inherent generate_signal ----

impl TradingStrategy for GridTradingStrategy {
    fn generate_signal(&mut self, current_price: f64) -> Option<TradingSignal> {
        let signals = GridTradingStrategy::generate_signals(self, current_price);
        signals.into_iter().next().map(|(s, _)| s)
    }
}

impl TradingStrategy for DcaStrategy {
    fn generate_signal(&mut self, current_price: f64) -> Option<TradingSignal> {
        // DCA buys on a schedule regardless of price; the price param seeds the
        // internal state for the next purchase decision.
        let _ = current_price;
        Some(TradingSignal::Buy)
    }
}

impl TradingStrategy for MomentumStrategy {
    fn generate_signal(&mut self, current_price: f64) -> Option<TradingSignal> {
        MomentumStrategy::generate_signal(self, current_price)
    }
}

impl TradingStrategy for MeanReversionStrategy {
    fn generate_signal(&mut self, current_price: f64) -> Option<TradingSignal> {
        MeanReversionStrategy::generate_signal(self, current_price)
    }
}

impl TradingStrategy for ScalpingStrategy {
    fn generate_signal(&mut self, current_price: f64) -> Option<TradingSignal> {
        ScalpingStrategy::generate_signal(self, current_price)
    }
}

impl TradingStrategy for AiTradingStrategy {
    fn generate_signal(&mut self, current_price: f64) -> Option<TradingSignal> {
        AiTradingStrategy::generate_signal(self, current_price)
    }
}

impl TradingStrategy for SignalStrategy {
    fn generate_signal(&mut self, _current_price: f64) -> Option<TradingSignal> {
        SignalStrategy::generate_signal(self)
    }
}

impl TradingStrategy for CustomStrategy {
    fn generate_signal(&mut self, _current_price: f64) -> Option<TradingSignal> {
        Some(TradingSignal::Hold)
    }
}
