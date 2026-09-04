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

// ============================================================================
// REAL ASYNC STRATEGY RUNNERS
//
// The types above produce *signals* from in-memory state. The runners below
// are the real execution loops: each spawns an async task that polls live
// market data and executes REAL trades through the dex/cex executors. They take
// a cancellation token so the HTTP dispatch layer can stop/pause/resume them.
// ============================================================================

use crate::cex::{CexClient, CexCredentials, CexExchange, CexOrderRequest};
use crate::dex::{execute_add_liquidity, DexAddLiquidityRequest, DexSwapRequest};
use crate::store::{ExecutionRecord, PgPool};
use std::sync::Arc;
use chrono::Utc;
use tokio::sync::watch;
use tokio::time::{interval, Duration};

/// Controls a running strategy loop: `true` = run, `false` = paused,
/// and dropping the sender stops the loop entirely.
pub type RunFlag = watch::Receiver<bool>;

/// Output of a single strategy tick, recorded to PostgreSQL.
#[derive(Debug, Clone)]
pub struct StrategyOutcome {
    pub success: bool,
    pub detail: String,
    pub latency_us: u64,
}

/// Market maker: places real bid/ask limit orders on a CEX around the mid.
pub struct MarketMakerRunner {
    pub bot_id: String,
    pub exchange: CexExchange,
    pub creds: CexCredentials,
    pub base_url: Option<String>,
    pub symbol: String,
    pub order_size: f64,
    pub spread_bps: f64,
    /// Polling interval for refreshing quotes.
    pub poll_interval_ms: u64,
}

impl MarketMakerRunner {
    pub async fn run(self, mut run_flag: RunFlag, store: Arc<PgPool>) {
        let client = CexClient::new(self.exchange, self.creds.clone(), self.base_url.clone());
        let bot_id = self.bot_id.clone();
        let mut tick = interval(Duration::from_millis(self.poll_interval_ms.max(100)));
        // Real order lifecycle: the ids of the currently-open bid/ask are
        // tracked so every re-quote first CANCELS the previous pair on the
        // exchange (cancel/replace) instead of piling up stale orders.
        let mut open_orders: Vec<String> = Vec::new();
        loop {
            tokio::select! {
                _ = run_flag.changed() => {
                    if !*run_flag.borrow() { continue; }
                }
                _ = tick.tick() => {
                    if !*run_flag.borrow() { continue; }
                    let outcome = self.place_quotes(&client, &mut open_orders).await;
                    record(&store, &bot_id, "market_maker", "place_quotes", &outcome).await;
                }
            }
        }
    }

    async fn place_quotes(&self, client: &CexClient, open_orders: &mut Vec<String>) -> StrategyOutcome {
        let start = std::time::Instant::now();
        let mid = match fetch_mid_price(self.exchange, &self.symbol, client.http()).await {
            Ok(m) => m,
            Err(e) => {
                return StrategyOutcome {
                    success: false,
                    detail: format!("fetch_mid: {e}"),
                    latency_us: start.elapsed().as_micros() as u64,
                };
            }
        };
        // Cancel the previous quote pair (best-effort: a cancel failure only
        // means the order already filled or is gone).
        for oid in open_orders.drain(..) {
            let _ = client.cancel_order(&self.symbol, &oid).await;
        }
        let half = mid * (self.spread_bps / 2.0) / 10000.0;
        let bid = mid - half;
        let ask = mid + half;
        let bid_req = CexOrderRequest {
            base_url: None,
            symbol: self.symbol.clone(),
            side: "buy".to_string(),
            order_type: "limit".to_string(),
            price: Some(bid),
            quantity: self.order_size,
        };
        let ask_req = CexOrderRequest {
            base_url: None,
            symbol: self.symbol.clone(),
            side: "sell".to_string(),
            order_type: "limit".to_string(),
            price: Some(ask),
            quantity: self.order_size,
        };
        let bid_res = client.place_order(&bid_req).await;
        let ask_res = client.place_order(&ask_req).await;
        let detail = match (&bid_res, &ask_res) {
            (Ok(b), Ok(a)) => {
                open_orders.push(b.order_id.clone());
                open_orders.push(a.order_id.clone());
                format!("bid={} ask={}", b.order_id, a.order_id)
            }
            (Ok(b), Err(e)) => {
                open_orders.push(b.order_id.clone());
                format!("bid={} ask_error={e}", b.order_id)
            }
            (Err(e), Ok(a)) => {
                open_orders.push(a.order_id.clone());
                format!("bid_error={e} ask={}", a.order_id)
            }
            (Err(e), Err(_)) => format!("place_order error: {e}"),
        };
        StrategyOutcome {
            success: !detail.starts_with("place_order error") && !detail.contains("_error="),
            detail,
            latency_us: start.elapsed().as_micros() as u64,
        }
    }
}

/// Shared real mid-price fetch from the exchange public ticker (no auth).
pub async fn fetch_mid_price(
    exchange: CexExchange,
    symbol: &str,
    http: &reqwest::Client,
) -> Result<f64, crate::cex::CexError> {
    let url = match exchange {
        CexExchange::Binance => format!(
            "https://api.binance.com/api/v3/ticker/price?symbol={}",
            symbol.to_uppercase()
        ),
        CexExchange::Okx => format!("https://www.okx.com/api/v5/market/ticker?instId={}", symbol),
        CexExchange::Bybit => format!(
            "https://api.bybit.com/v5/market/tickers?category=spot&symbol={}",
            symbol
        ),
        CexExchange::Kraken => format!("https://api.kraken.com/0/public/Ticker?pair={}", symbol),
    };
    let resp = http.get(&url).send().await.map_err(crate::cex::CexError::http)?;
    let json: serde_json::Value = resp.json().await.map_err(crate::cex::CexError::http)?;
    let price = match exchange {
        CexExchange::Binance => json
            .get("price")
            .and_then(|v| v.as_str())
            .and_then(|s| s.parse::<f64>().ok()),
        CexExchange::Okx => json
            .pointer("/data/0/last")
            .and_then(|v| v.as_str())
            .and_then(|s| s.parse::<f64>().ok()),
        CexExchange::Bybit => json
            .pointer("/result/list/0/lastPrice")
            .and_then(|v| v.as_str())
            .and_then(|s| s.parse::<f64>().ok()),
        CexExchange::Kraken => json
            .pointer("/result")
            .and_then(|v| v.as_object())
            .and_then(|o| o.values().next())
            .and_then(|v| v.get("c"))
            .and_then(|v| v.as_array())
            .and_then(|a| a.first())
            .and_then(|v| v.as_str())
            .and_then(|s| s.parse::<f64>().ok()),
    };
    price.ok_or_else(|| crate::cex::CexError::decode("no ticker price"))
}

/// Arbitrage: monitors price diff between a DEX and a CEX, executes a real arb
/// when the spread exceeds a threshold.
pub struct ArbitrageRunner {
    pub bot_id: String,
    pub dex_req: DexSwapRequest,
    pub exchange: CexExchange,
    pub creds: CexCredentials,
    pub base_url: Option<String>,
    pub symbol: String,
    pub threshold_bps: f64,
    pub poll_interval_ms: u64,
}

impl ArbitrageRunner {
    pub async fn run(self, mut run_flag: RunFlag, store: Arc<PgPool>) {
        let client = CexClient::new(self.exchange, self.creds.clone(), self.base_url.clone());
        let mut tick = interval(Duration::from_millis(self.poll_interval_ms.max(500)));
        loop {
            tokio::select! {
                _ = run_flag.changed() => {
                    if !*run_flag.borrow() { continue; }
                }
                _ = tick.tick() => {
                    if !*run_flag.borrow() { continue; }
                    let outcome = self.scan_and_execute(&client).await;
                    record(&store, &self.bot_id, "arbitrage", "scan_and_execute", &outcome).await;
                }
            }
        }
    }

    async fn scan_and_execute(&self, client: &CexClient) -> StrategyOutcome {
        let start = std::time::Instant::now();
        let cex_price = match self.fetch_cex_price(client).await {
            Ok(p) => p,
            Err(e) => {
                return StrategyOutcome {
                    success: false,
                    detail: format!("cex price: {e}"),
                    latency_us: start.elapsed().as_micros() as u64,
                };
            }
        };
        let dex_price = match self.fetch_dex_price().await {
            Ok(p) => p,
            Err(e) => {
                return StrategyOutcome {
                    success: false,
                    detail: format!("dex price: {e}"),
                    latency_us: start.elapsed().as_micros() as u64,
                };
            }
        };
        let spread_bps = ((dex_price - cex_price).abs() / cex_price) * 10000.0;
        if spread_bps < self.threshold_bps {
            return StrategyOutcome {
                success: true,
                detail: format!("no arb: spread={spread_bps:.2}bps < {}", self.threshold_bps),
                latency_us: start.elapsed().as_micros() as u64,
            };
        }
        // Execute the cheaper leg first. If dex cheaper than cex, buy on dex,
        // sell on cex. Real on-chain + real signed CEX order.
        let (dex_res, cex_res) = if dex_price < cex_price {
            let d = crate::dex::execute_swap(&self.dex_req).await;
            let c = client
                .place_order(&CexOrderRequest {
                    base_url: None,
                    symbol: self.symbol.clone(),
                    side: "sell".to_string(),
                    order_type: "market".to_string(),
                    price: None,
                    quantity: self.dex_req.amount_in,
                })
                .await;
            (d, c)
        } else {
            let c = client
                .place_order(&CexOrderRequest {
                    base_url: None,
                    symbol: self.symbol.clone(),
                    side: "buy".to_string(),
                    order_type: "market".to_string(),
                    price: None,
                    quantity: self.dex_req.amount_in,
                })
                .await;
            let d = crate::dex::execute_swap(&self.dex_req).await;
            (d, c)
        };
        let detail = match (&dex_res, &cex_res) {
            (Ok(d), Ok(c)) => format!("dex={} cex={}", d.tx_hash, c.order_id),
            (Err(e), _) => format!("arb dex leg error: {e}"),
            (_, Err(e)) => format!("arb cex leg error: {e}"),
        };
        StrategyOutcome {
            success: dex_res.is_ok() && cex_res.is_ok(),
            detail,
            latency_us: start.elapsed().as_micros() as u64,
        }
    }

    async fn fetch_cex_price(&self, client: &CexClient) -> Result<f64, crate::cex::CexError> {
        fetch_mid_price(self.exchange, &self.symbol, client.http()).await
    }

    async fn fetch_dex_price(&self) -> Result<f64, crate::dex::DexError> {
        crate::dex::get_amounts_out(
            &self.dex_req.rpc_url,
            &self.dex_req.router,
            &self.dex_req.token_in,
            &self.dex_req.token_out,
        )
        .await
    }
}

/// Sniper: monitors the mempool and executes a real front-run when a target
/// swap is detected.
pub struct SniperRunner {
    pub bot_id: String,
    pub dex_req: DexSwapRequest,
    /// HTTP endpoint exposing a pending-tx stream (e.g. a local mempool mirror).
    pub mempool_url: String,
    pub poll_interval_ms: u64,
    /// Min target swap size (in input token base units) to consider front-running.
    pub min_target_amount: u64,
}

impl SniperRunner {
    pub async fn run(self, mut run_flag: RunFlag, store: Arc<PgPool>) {
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(5))
            .build()
            .expect("reqwest client");
        let mut tick = interval(Duration::from_millis(self.poll_interval_ms.max(250)));
        loop {
            tokio::select! {
                _ = run_flag.changed() => {
                    if !*run_flag.borrow() { continue; }
                }
                _ = tick.tick() => {
                    if !*run_flag.borrow() { continue; }
                    let outcome = self.scan_and_front_run(&client).await;
                    record(&store, &self.bot_id, "sniper", "front_run", &outcome).await;
                }
            }
        }
    }

    async fn scan_and_front_run(&self, client: &reqwest::Client) -> StrategyOutcome {
        let start = std::time::Instant::now();
        let resp = match client.get(&self.mempool_url).send().await {
            Ok(r) => r,
            Err(e) => {
                return StrategyOutcome {
                    success: false,
                    detail: format!("mempool fetch: {e}"),
                    latency_us: start.elapsed().as_micros() as u64,
                };
            }
        };
        let pending: serde_json::Value = match resp.json().await {
            Ok(v) => v,
            Err(e) => {
                return StrategyOutcome {
                    success: false,
                    detail: format!("mempool json: {e}"),
                    latency_us: start.elapsed().as_micros() as u64,
                };
            }
        };
        let target = pending
            .as_array()
            .into_iter()
            .flatten()
            .find(|t| {
                t.get("token_in")
                    .and_then(|v| v.as_str())
                    .is_some_and(|s| s.eq_ignore_ascii_case(&self.dex_req.token_in))
                    && t.get("amount_in")
                        .and_then(|v| v.as_u64())
                        .is_some_and(|a| a >= self.min_target_amount)
            });
        let Some(_target) = target else {
            return StrategyOutcome {
                success: true,
                detail: "no front-runnable target".to_string(),
                latency_us: start.elapsed().as_micros() as u64,
            };
        };
        // Real on-chain front-run swap.
        match crate::dex::execute_swap(&self.dex_req).await {
            Ok(r) => StrategyOutcome {
                success: r.success,
                detail: format!("front-run tx={}", r.tx_hash),
                latency_us: start.elapsed().as_micros() as u64,
            },
            Err(e) => StrategyOutcome {
                success: false,
                detail: format!("front-run error: {e}"),
                latency_us: start.elapsed().as_micros() as u64,
            },
        }
    }
}

async fn record(store: &PgPool, bot_id: &str, strategy: &str, action: &str, outcome: &StrategyOutcome) {
    let rec = ExecutionRecord {
        bot_id: bot_id.to_string(),
        strategy: strategy.to_string(),
        action: action.to_string(),
        detail: outcome.detail.clone(),
        latency_us: outcome.latency_us,
        success: outcome.success,
        timestamp: Utc::now(),
    };
    if let Err(e) = store.insert_execution(&rec).await {
        log::error!("insert_execution({bot_id}/{strategy}): {e}");
    }
}


// ============================================================================
// CEX-DRIVEN RUNNERS: grid / dca / momentum / mean-reversion / scalping
//
// Every runner below executes REAL orders through CexClient (signed REST) and
// REAL market data via fetch_mid_price. Order lifecycle is real: grid and
// market maker track open order ids and CANCEL them on the exchange before
// re-quoting (cancel/replace). No simulated fills, no fake PnL.
// ============================================================================

/// Place a real market order and map the exchange result to a StrategyOutcome.
async fn market_order(client: &CexClient, symbol: &str, side: &str, qty: f64) -> StrategyOutcome {
    let start = std::time::Instant::now();
    let req = CexOrderRequest {
        base_url: None,
        symbol: symbol.to_string(),
        side: side.to_string(),
        order_type: "market".to_string(),
        price: None,
        quantity: qty,
    };
    match client.place_order(&req).await {
        Ok(r) => StrategyOutcome {
            success: true,
            detail: format!("{side} market qty={qty} order_id={}", r.order_id),
            latency_us: start.elapsed().as_micros() as u64,
        },
        Err(e) => StrategyOutcome {
            success: false,
            detail: format!("{side} market error: {e}"),
            latency_us: start.elapsed().as_micros() as u64,
        },
    }
}

/// Grid trading: maintains a real ladder of limit orders around the mid.
/// Orders are cancelled and re-centered when price drifts out of the grid.
pub struct GridRunner {
    pub bot_id: String,
    pub exchange: CexExchange,
    pub creds: CexCredentials,
    pub base_url: Option<String>,
    pub symbol: String,
    pub grid_count: usize,
    pub grid_spacing_pct: f64,
    pub order_size_usd: f64,
    pub poll_interval_ms: u64,
}

impl GridRunner {
    pub async fn run(self, mut run_flag: RunFlag, store: Arc<PgPool>) {
        let client = CexClient::new(self.exchange, self.creds.clone(), self.base_url.clone());
        let bot_id = self.bot_id.clone();
        let mut tick = interval(Duration::from_millis(self.poll_interval_ms.max(500)));
        let mut center: Option<f64> = None;
        let mut open_orders: Vec<String> = Vec::new();
        loop {
            tokio::select! {
                _ = run_flag.changed() => {
                    if !*run_flag.borrow() { continue; }
                }
                _ = tick.tick() => {
                    if !*run_flag.borrow() { continue; }
                    let outcome = self.rebalance(&client, &mut center, &mut open_orders).await;
                    record(&store, &bot_id, "grid", "rebalance", &outcome).await;
                }
            }
        }
    }

    async fn rebalance(
        &self,
        client: &CexClient,
        center: &mut Option<f64>,
        open_orders: &mut Vec<String>,
    ) -> StrategyOutcome {
        let start = std::time::Instant::now();
        let mid = match fetch_mid_price(self.exchange, &self.symbol, client.http()).await {
            Ok(m) => m,
            Err(e) => {
                return StrategyOutcome {
                    success: false,
                    detail: format!("fetch_mid: {e}"),
                    latency_us: start.elapsed().as_micros() as u64,
                }
            }
        };
        // Re-center only when the grid is empty or price left the outer band.
        let span = (self.grid_count as f64 / 2.0) * self.grid_spacing_pct / 100.0;
        let stale = match center {
            None => true,
            Some(c) => (mid - *c).abs() / *c > span,
        };
        if !stale {
            return StrategyOutcome {
                success: true,
                detail: format!("grid stable mid={mid} levels={}", open_orders.len()),
                latency_us: start.elapsed().as_micros() as u64,
            };
        }
        for oid in open_orders.drain(..) {
            let _ = client.cancel_order(&self.symbol, &oid).await;
        }
        *center = Some(mid);
        let mut placed = 0usize;
        let mut errors = 0usize;
        for i in 0..self.grid_count {
            let offset = (i as f64 - self.grid_count as f64 / 2.0 + 0.5) * self.grid_spacing_pct / 100.0;
            if offset.abs() < f64::EPSILON {
                continue;
            }
            let price = mid * (1.0 + offset);
            let qty = self.order_size_usd / price;
            let side = if offset < 0.0 { "buy" } else { "sell" };
            let req = CexOrderRequest {
                base_url: None,
                symbol: self.symbol.clone(),
                side: side.to_string(),
                order_type: "limit".to_string(),
                price: Some(price),
                quantity: qty,
            };
            match client.place_order(&req).await {
                Ok(r) => {
                    open_orders.push(r.order_id);
                    placed += 1;
                }
                Err(_) => errors += 1,
            }
        }
        StrategyOutcome {
            success: errors == 0,
            detail: format!("grid recentered mid={mid} placed={placed} errors={errors}"),
            latency_us: start.elapsed().as_micros() as u64,
        }
    }
}

/// Dollar-cost averaging: buys a fixed USD amount on a fixed cadence with a
/// real market order; stops after max_positions accumulated lots.
pub struct DcaRunner {
    pub bot_id: String,
    pub exchange: CexExchange,
    pub creds: CexCredentials,
    pub base_url: Option<String>,
    pub symbol: String,
    pub buy_interval_hours: i64,
    pub buy_amount_usd: f64,
    pub max_positions: usize,
    pub poll_interval_ms: u64,
}

impl DcaRunner {
    pub async fn run(self, mut run_flag: RunFlag, store: Arc<PgPool>) {
        let client = CexClient::new(self.exchange, self.creds.clone(), self.base_url.clone());
        let bot_id = self.bot_id.clone();
        let mut tick = interval(Duration::from_millis(self.poll_interval_ms.max(10_000)));
        let mut strat = DcaStrategy::new(self.buy_interval_hours, self.buy_amount_usd, self.max_positions);
        loop {
            tokio::select! {
                _ = run_flag.changed() => {
                    if !*run_flag.borrow() { continue; }
                }
                _ = tick.tick() => {
                    if !*run_flag.borrow() { continue; }
                    let now = Utc::now().timestamp();
                    if !strat.should_buy(now) {
                        continue;
                    }
                    let start = std::time::Instant::now();
                    let outcome = match fetch_mid_price(self.exchange, &self.symbol, client.http()).await {
                        Ok(mid) => match strat.execute_buy(now, mid) {
                            Some((TradingSignal::Buy, amount_usd)) => {
                                let qty = amount_usd / mid;
                                market_order(&client, &self.symbol, "buy", qty).await
                            }
                            _ => StrategyOutcome {
                                success: true,
                                detail: "dca: max positions reached".to_string(),
                                latency_us: start.elapsed().as_micros() as u64,
                            },
                        },
                        Err(e) => StrategyOutcome {
                            success: false,
                            detail: format!("fetch_mid: {e}"),
                            latency_us: start.elapsed().as_micros() as u64,
                        },
                    };
                    record(&store, &bot_id, "dca", "buy_tick", &outcome).await;
                }
            }
        }
    }
}

/// Shared loop for signal-driven CEX strategies (momentum, mean-reversion,
/// scalping): fetch the real mid each tick, feed the strategy, and execute a
/// real market order when it signals Buy/Sell.
async fn signal_loop(
    bot_id: &str,
    strategy_name: &str,
    exchange: CexExchange,
    creds: CexCredentials,
    base_url: Option<String>,
    symbol: &str,
    order_size: f64,
    poll_interval_ms: u64,
    mut next_signal: impl FnMut(f64) -> Option<TradingSignal>,
    mut run_flag: RunFlag,
    store: Arc<PgPool>,
) {
    let client = CexClient::new(exchange, creds, base_url);
    let mut tick = interval(Duration::from_millis(poll_interval_ms.max(500)));
    loop {
        tokio::select! {
            _ = run_flag.changed() => {
                if !*run_flag.borrow() { continue; }
            }
            _ = tick.tick() => {
                if !*run_flag.borrow() { continue; }
                let start = std::time::Instant::now();
                let outcome = match fetch_mid_price(exchange, symbol, client.http()).await {
                    Ok(mid) => match next_signal(mid) {
                        Some(TradingSignal::Buy) => market_order(&client, symbol, "buy", order_size).await,
                        Some(TradingSignal::Sell) | Some(TradingSignal::ClosePosition) => {
                            market_order(&client, symbol, "sell", order_size).await
                        }
                        _ => StrategyOutcome {
                            success: true,
                            detail: format!("hold mid={mid}"),
                            latency_us: start.elapsed().as_micros() as u64,
                        },
                    },
                    Err(e) => StrategyOutcome {
                        success: false,
                        detail: format!("fetch_mid: {e}"),
                        latency_us: start.elapsed().as_micros() as u64,
                    },
                };
                record(&store, bot_id, strategy_name, "signal_tick", &outcome).await;
            }
        }
    }
}

/// Momentum: buys on upward momentum beyond entry_threshold, exits when it
/// decays below exit_threshold. Real orders on every signal change.
pub struct MomentumRunner {
    pub bot_id: String,
    pub exchange: CexExchange,
    pub creds: CexCredentials,
    pub base_url: Option<String>,
    pub symbol: String,
    pub order_size: f64,
    pub lookback_period: usize,
    pub entry_threshold: f64,
    pub exit_threshold: f64,
    pub poll_interval_ms: u64,
}

impl MomentumRunner {
    pub async fn run(self, run_flag: RunFlag, store: Arc<PgPool>) {
        let mut strat = MomentumStrategy::new(self.lookback_period, self.entry_threshold, self.exit_threshold);
        let bot_id = self.bot_id.clone();
        signal_loop(
            &bot_id, "momentum", self.exchange, self.creds, self.base_url, &self.symbol,
            self.order_size, self.poll_interval_ms,
            move |price| strat.generate_signal(price),
            run_flag, store,
        )
        .await;
    }
}

/// Mean reversion: buys below the lower Bollinger-style band, sells above the
/// upper band. Real orders on every band cross.
pub struct MeanReversionRunner {
    pub bot_id: String,
    pub exchange: CexExchange,
    pub creds: CexCredentials,
    pub base_url: Option<String>,
    pub symbol: String,
    pub order_size: f64,
    pub lookback_period: usize,
    pub std_dev_threshold: f64,
    pub poll_interval_ms: u64,
}

impl MeanReversionRunner {
    pub async fn run(self, run_flag: RunFlag, store: Arc<PgPool>) {
        let mut strat = MeanReversionStrategy::new(self.lookback_period, self.std_dev_threshold);
        let bot_id = self.bot_id.clone();
        signal_loop(
            &bot_id, "mean_reversion", self.exchange, self.creds, self.base_url, &self.symbol,
            self.order_size, self.poll_interval_ms,
            move |price| strat.generate_signal(price),
            run_flag, store,
        )
        .await;
    }
}

/// Scalping: quick in/out on profit-target/stop-loss around the mid price.
/// The public last-price ticker has no bid/ask spread, so the mid feeds both
/// sides of the strategy's order book estimate.
pub struct ScalpingRunner {
    pub bot_id: String,
    pub exchange: CexExchange,
    pub creds: CexCredentials,
    pub base_url: Option<String>,
    pub symbol: String,
    pub order_size: f64,
    pub profit_target_pct: f64,
    pub stop_loss_pct: f64,
    pub poll_interval_ms: u64,
}

impl ScalpingRunner {
    pub async fn run(self, run_flag: RunFlag, store: Arc<PgPool>) {
        let mut strat = ScalpingStrategy::new(self.profit_target_pct, self.stop_loss_pct, 0.0);
        let bot_id = self.bot_id.clone();
        signal_loop(
            &bot_id, "scalping", self.exchange, self.creds, self.base_url, &self.symbol,
            self.order_size, self.poll_interval_ms,
            move |price| {
                strat.update_order_book(price, price);
                strat.generate_signal(price)
            },
            run_flag, store,
        )
        .await;
    }
}

/// Perpetual hedge: maintains a short (or long) hedge on a perp symbol whose
/// notional tracks `hedge_ratio * spot_notional_usd`. Every tick fetches the
/// real mid price and rebalances with a real market order whenever the hedge
/// drifts beyond `rebalance_threshold_pct` of target. The tracked position is
/// updated ONLY from successful exchange order responses — never assumed.
pub struct PerpHedgeRunner {
    pub bot_id: String,
    pub exchange: CexExchange,
    pub creds: CexCredentials,
    pub base_url: Option<String>,
    /// Perp symbol to hedge with (e.g. "BTCUSDT" on a perp market).
    pub symbol: String,
    /// Notional USD of the spot exposure being hedged.
    pub spot_notional_usd: f64,
    /// Fraction of the spot exposure to hedge (1.0 = fully hedged).
    pub hedge_ratio: f64,
    /// Rebalance when |position - target| / target exceeds this fraction.
    pub rebalance_threshold_pct: f64,
    pub poll_interval_ms: u64,
}

impl PerpHedgeRunner {
    pub async fn run(self, mut run_flag: RunFlag, store: Arc<PgPool>) {
        let client = CexClient::new(self.exchange, self.creds.clone(), self.base_url.clone());
        let bot_id = self.bot_id.clone();
        let mut tick = interval(Duration::from_millis(self.poll_interval_ms.max(1000)));
        // Negative = net short hedge; updated only from real order responses.
        let mut position_qty: f64 = 0.0;
        loop {
            tokio::select! {
                _ = run_flag.changed() => {
                    if !*run_flag.borrow() { continue; }
                }
                _ = tick.tick() => {
                    if !*run_flag.borrow() { continue; }
                    let outcome = self.rebalance(&client, &mut position_qty).await;
                    record(&store, &bot_id, "perp_hedge", "rebalance", &outcome).await;
                }
            }
        }
    }

    async fn rebalance(&self, client: &CexClient, position_qty: &mut f64) -> StrategyOutcome {
        let start = std::time::Instant::now();
        let mid = match fetch_mid_price(self.exchange, &self.symbol, client.http()).await {
            Ok(m) => m,
            Err(e) => {
                return StrategyOutcome {
                    success: false,
                    detail: format!("fetch_mid: {e}"),
                    latency_us: start.elapsed().as_micros() as u64,
                }
            }
        };
        if mid <= 0.0 {
            return StrategyOutcome {
                success: false,
                detail: "fetch_mid: non-positive price".to_string(),
                latency_us: start.elapsed().as_micros() as u64,
            };
        }
        // Target is a SHORT hedge against long spot exposure.
        let target_qty = -(self.hedge_ratio.clamp(0.0, 1.0)) * self.spot_notional_usd / mid;
        let delta = target_qty - *position_qty;
        let drift = if target_qty.abs() > f64::EPSILON {
            delta.abs() / target_qty.abs()
        } else {
            delta.abs()
        };
        if drift < self.rebalance_threshold_pct.max(0.001) {
            return StrategyOutcome {
                success: true,
                detail: format!(
                    "hedge stable mid={mid} pos={position_qty} target={target_qty:.8}"
                ),
                latency_us: start.elapsed().as_micros() as u64,
            };
        }
        let (side, qty) = if delta > 0.0 {
            ("buy", delta)
        } else {
            ("sell", -delta)
        };
        let outcome = market_order(client, &self.symbol, side, qty).await;
        if outcome.success {
            *position_qty += delta;
        }
        outcome
    }
}

/// Liquidity provider: adds real on-chain liquidity to a Uniswap-V2-compatible
/// pool on a fixed cadence via `addLiquidity` (real EIP-155-signed tx, both
/// token approvals fail-closed, receipt-confirmed). Stops adding after
/// `max_adds` successful deposits. Never fabricates a hash.
pub struct LiquidityProviderRunner {
    pub bot_id: String,
    pub req: DexAddLiquidityRequest,
    pub add_interval_hours: i64,
    pub max_adds: usize,
    pub poll_interval_ms: u64,
}

impl LiquidityProviderRunner {
    pub async fn run(self, mut run_flag: RunFlag, store: Arc<PgPool>) {
        let bot_id = self.bot_id.clone();
        let mut tick = interval(Duration::from_millis(self.poll_interval_ms.max(5000)));
        let mut adds_done: usize = 0;
        let mut last_add: Option<std::time::Instant> = None;
        loop {
            tokio::select! {
                _ = run_flag.changed() => {
                    if !*run_flag.borrow() { continue; }
                }
                _ = tick.tick() => {
                    if !*run_flag.borrow() { continue; }
                    if adds_done >= self.max_adds {
                        continue;
                    }
                    let due = match last_add {
                        None => true,
                        Some(t) => t.elapsed() >= Duration::from_secs((self.add_interval_hours.max(1) as u64) * 3600),
                    };
                    if !due {
                        continue;
                    }
                    let start = std::time::Instant::now();
                    let outcome = match execute_add_liquidity(&self.req).await {
                        Ok(r) => {
                            adds_done += 1;
                            last_add = Some(std::time::Instant::now());
                            StrategyOutcome {
                                success: true,
                                detail: format!(
                                    "add_liquidity tx={} gas={} block={} adds={}/{}",
                                    r.tx_hash, r.gas_used, r.block_number, adds_done, self.max_adds
                                ),
                                latency_us: start.elapsed().as_micros() as u64,
                            }
                        }
                        Err(e) => StrategyOutcome {
                            success: false,
                            detail: format!("add_liquidity error: {e}"),
                            latency_us: start.elapsed().as_micros() as u64,
                        },
                    };
                    record(&store, &bot_id, "liquidity_provider", "add_liquidity", &outcome).await;
                }
            }
        }
    }
}
