//!
//! TigerWallet Backtesting Engine
//! High-performance backtesting for trading strategies
//!

use serde::{Deserialize, Serialize};
use std::collections::VecDeque;

/// Historical price data point
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceData {
    pub timestamp: i64,
    pub open: f64,
    pub high: f64,
    pub low: f64,
    pub close: f64,
    pub volume: f64,
}

/// Trade execution record
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BacktestTrade {
    pub id: String,
    pub entry_time: i64,
    pub exit_time: Option<i64>,
    pub entry_price: f64,
    pub exit_price: Option<f64>,
    pub quantity: f64,
    pub side: TradeSide,
    pub pnl: Option<f64>,
    pub pnl_percentage: Option<f64>,
    pub status: TradeStatus,
    pub fees: f64,
}

/// Trade side
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum TradeSide {
    Long,
    Short,
}

/// Trade status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum TradeStatus {
    Open,
    Closed,
}

/// Backtest configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BacktestConfig {
    pub symbol: String,
    pub start_time: i64,
    pub end_time: i64,
    pub initial_capital: f64,
    pub fee_percentage: f64,
    pub slippage_percentage: f64,
    pub leverage: f64,
    pub position_size_percentage: f64,
}

/// Backtest result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BacktestResult {
    pub config: BacktestConfig,
    pub trades: Vec<BacktestTrade>,
    pub metrics: BacktestMetrics,
    pub equity_curve: Vec<EquityPoint>,
    pub drawdown_curve: Vec<DrawdownPoint>,
    pub monthly_returns: Vec<MonthlyReturn>,
}

/// Equity curve point
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EquityPoint {
    pub timestamp: i64,
    pub equity: f64,
    pub pnl: f64,
}

/// Drawdown point
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DrawdownPoint {
    pub timestamp: i64,
    pub drawdown: f64,
    pub drawdown_percentage: f64,
}

/// Monthly return
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonthlyReturn {
    pub year: i32,
    pub month: u32,
    pub return_percentage: f64,
    pub trades: u32,
}

/// Backtest metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BacktestMetrics {
    // Basic metrics
    pub initial_capital: f64,
    pub final_capital: f64,
    pub total_return: f64,
    pub total_return_percentage: f64,
    
    // Trade metrics
    pub total_trades: u64,
    pub winning_trades: u64,
    pub losing_trades: u64,
    pub win_rate: f64,
    pub avg_win: f64,
    pub avg_loss: f64,
    pub largest_win: f64,
    pub largest_loss: f64,
    pub avg_trade_duration: u64,
    
    // Risk metrics
    pub sharpe_ratio: f64,
    pub sortino_ratio: f64,
    pub max_drawdown: f64,
    pub max_drawdown_percentage: f64,
    pub avg_drawdown: f64,
    pub volatility: f64,
    
    // Profit metrics
    pub profit_factor: f64,
    pub expectancy: f64,
    pub risk_reward_ratio: f64,
    
    // Time metrics
    pub total_bars: u64,
    pub trading_days: u64,
    pub start_date: i64,
    pub end_date: i64,
}

/// Backtesting engine
pub struct BacktestEngine {
    config: BacktestConfig,
    data: Vec<PriceData>,
    trades: Vec<BacktestTrade>,
    equity_curve: Vec<EquityPoint>,
    current_capital: f64,
    position: Option<Position>,
}

#[derive(Debug, Clone)]
struct Position {
    side: TradeSide,
    entry_price: f64,
    quantity: f64,
    entry_time: i64,
}

impl BacktestEngine {
    /// Create a new backtest engine
    pub fn new(config: BacktestConfig, data: Vec<PriceData>) -> Self {
        Self {
            config,
            data,
            trades: Vec::new(),
            equity_curve: Vec::new(),
            current_capital: config.initial_capital,
            position: None,
        }
    }

    /// Run the backtest
    pub fn run(&mut self) -> BacktestResult {
        // Sort data by timestamp
        self.data.sort_by_key(|d| d.timestamp);
        
        let mut equity = self.config.initial_capital;
        let mut peak_equity = equity;
        let mut max_drawdown = 0.0;
        
        // Process each bar
        for (i, bar) in self.data.iter().enumerate() {
            // Update equity curve
            let pnl = self.calculate_unrealized_pnl(bar.close);
            equity = self.current_capital + pnl;
            
            // Track peak equity and drawdown
            if equity > peak_equity {
                peak_equity = equity;
            }
            
            let drawdown = peak_equity - equity;
            let drawdown_percentage = if peak_equity > 0.0 {
                (drawdown / peak_equity) * 100.0
            } else {
                0.0
            };
            
            if drawdown > max_drawdown {
                max_drawdown = drawdown;
            }
            
            self.equity_curve.push(EquityPoint {
                timestamp: bar.timestamp,
                equity,
                pnl: equity - self.config.initial_capital,
            });
            
            // Process strategy signals (placeholder)
            self.process_bar(bar, i);
        }
        
        // Close any open position at the end
        if let Some(ref position) = self.position {
            let last_bar = self.data.last().unwrap();
            self.close_position(last_bar.close, last_bar.timestamp);
        }
        
        // Calculate final metrics
        let metrics = self.calculate_metrics(max_drawdown);
        
        BacktestResult {
            config: self.config.clone(),
            trades: self.trades.clone(),
            metrics,
            equity_curve: self.equity_curve.clone(),
            drawdown_curve: self.calculate_drawdown_curve(),
            monthly_returns: self.calculate_monthly_returns(),
        }
    }

    /// Process a single bar - implement strategy logic here
    fn process_bar(&mut self, bar: &PriceData, index: usize) {
        // Example: Simple moving average crossover strategy
        if index < 20 {
            return;
        }
        
        // Calculate moving averages
        let short_ma = self.calculate_ma(index, 5);
        let long_ma = self.calculate_ma(index, 20);
        
        // Entry signals
        if short_ma > long_ma && self.position.is_none() {
            let position_size = self.current_capital * self.config.position_size_percentage / 100.0;
            let price_with_slippage = bar.close * (1.0 + self.config.slippage_percentage / 100.0);
            let quantity = position_size / price_with_slippage;
            
            self.open_position(TradeSide::Long, price_with_slippage, quantity, bar.timestamp);
        }
        // Exit signals
        else if short_ma < long_ma && self.position.is_some() {
            let price_with_slippage = bar.close * (1.0 - self.config.slippage_percentage / 100.0);
            self.close_position(price_with_slippage, bar.timestamp);
        }
    }

    /// Calculate simple moving average
    fn calculate_ma(&self, current_index: usize, period: usize) -> f64 {
        if current_index < period {
            return 0.0;
        }
        
        let start = current_index - period + 1;
        let sum: f64 = self.data[start..=current_index]
            .iter()
            .map(|d| d.close)
            .sum();
        
        sum / period as f64
    }

    /// Open a new position
    fn open_position(&mut self, side: TradeSide, price: f64, quantity: f64, timestamp: i64) {
        let fee = price * quantity * self.config.fee_percentage / 100.0;
        self.current_capital -= fee;
        
        self.position = Some(Position {
            side,
            entry_price: price,
            quantity,
            entry_time: timestamp,
        });
    }

    /// Close existing position
    fn close_position(&mut self, price: f64, timestamp: i64) {
        if let Some(position) = self.position.take() {
            let exit_value = price * position.quantity;
            let fee = exit_value * self.config.fee_percentage / 100.0;
            let pnl = match position.side {
                TradeSide::Long => (price - position.entry_price) * position.quantity - fee,
                TradeSide::Short => (position.entry_price - price) * position.quantity - fee,
            };
            
            self.current_capital += exit_value - fee;
            
            let pnl_percentage = (pnl / (position.entry_price * position.quantity)) * 100.0;
            
            self.trades.push(BacktestTrade {
                id: format!("trade_{}", self.trades.len()),
                entry_time: position.entry_time,
                exit_time: Some(timestamp),
                entry_price: position.entry_price,
                exit_price: Some(price),
                quantity: position.quantity,
                side: position.side,
                pnl: Some(pnl),
                pnl_percentage: Some(pnl_percentage),
                status: TradeStatus::Closed,
                fees: fee * 2.0,
            });
        }
    }

    /// Calculate unrealized PnL
    fn calculate_unrealized_pnl(&self, current_price: f64) -> f64 {
        if let Some(ref position) = self.position {
            match position.side {
                TradeSide::Long => (current_price - position.entry_price) * position.quantity,
                TradeSide::Short => (position.entry_price - current_price) * position.quantity,
            }
        } else {
            0.0
        }
    }

    /// Calculate all metrics
    fn calculate_metrics(&self, max_drawdown: f64) -> BacktestMetrics {
        let closed_trades: Vec<&BacktestTrade> = self.trades
            .iter()
            .filter(|t| t.status == TradeStatus::Closed)
            .collect();
        
        let total_trades = closed_trades.len() as u64;
        let winning_trades = closed_trades.iter().filter(|t| t.pnl.unwrap_or(0.0) > 0.0).count() as u64;
        let losing_trades = closed_trades.iter().filter(|t| t.pnl.unwrap_or(0.0) < 0.0).count() as u64;
        
        let total_pnl: f64 = closed_trades.iter().map(|t| t.pnl.unwrap_or(0.0)).sum();
        let total_wins: f64 = closed_trades.iter()
            .filter(|t| t.pnl.unwrap_or(0.0) > 0.0)
            .map(|t| t.pnl.unwrap_or(0.0))
            .sum();
        let total_losses = closed_trades.iter()
            .filter(|t| t.pnl.unwrap_or(0.0) < 0.0)
            .map(|t| t.pnl.unwrap_or(0.0))
            .sum::<f64>().abs();
        
        let avg_win = if winning_trades > 0 { total_wins / winning_trades as f64 } else { 0.0 };
        let avg_loss = if losing_trades > 0 { total_losses / losing_trades as f64 } else { 0.0 };
        
        let largest_win = closed_trades.iter()
            .map(|t| t.pnl.unwrap_or(0.0))
            .fold(0.0, |a, b| a.max(b));
        let largest_loss = closed_trades.iter()
            .map(|t| t.pnl.unwrap_or(0.0))
            .fold(0.0, |a, b| a.min(b));
        
        let win_rate = if total_trades > 0 {
            (winning_trades as f64 / total_trades as f64) * 100.0
        } else {
            0.0
        };
        
        let profit_factor = if total_losses > 0.0 {
            total_wins / total_losses
        } else if total_wins > 0.0 {
            f64::INFINITY
        } else {
            0.0
        };
        
        let expectancy = if total_trades > 0 {
            (win_rate / 100.0 * avg_win) - ((100.0 - win_rate) / 100.0 * avg_loss)
        } else {
            0.0
        };
        
        // Calculate volatility
        let returns: Vec<f64> = self.equity_curve.windows(2)
            .map(|w| {
                if w[0].equity > 0.0 {
                    (w[1].equity - w[0].equity) / w[0].equity
                } else {
                    0.0
                }
            })
            .collect();
        
        let mean_return: f64 = if !returns.is_empty() {
            returns.iter().sum::<f64>() / returns.len() as f64
        } else {
            0.0
        };
        
        let variance: f64 = returns.iter()
            .map(|r| (r - mean_return).powi(2))
            .sum::<f64>() / returns.len() as f64;
        let volatility = variance.sqrt() * (252.0_f64).sqrt();
        
        // Calculate Sharpe ratio (assuming risk-free rate of 2%)
        let risk_free_rate = 0.02;
        let sharpe_ratio = if volatility > 0.0 {
            (mean_return * 252.0 - risk_free_rate) / volatility
        } else {
            0.0
        };
        
        // Sortino ratio
        let negative_returns: Vec<f64> = returns.iter()
            .filter(|r| **r < 0.0)
            .map(|r| r.powi(2))
            .collect();
        let downside_variance = if !negative_returns.is_empty() {
            negative_returns.iter().sum::<f64>() / negative_returns.len() as f64
        } else {
            0.0
        };
        let downside_deviation = downside_variance.sqrt() * (252.0_f64).sqrt();
        let sortino_ratio = if downside_deviation > 0.0 {
            (mean_return * 252.0 - risk_free_rate) / downside_deviation
        } else {
            0.0
        };
        
        let max_drawdown_percentage = if self.config.initial_capital > 0.0 {
            (max_drawdown / self.config.initial_capital) * 100.0
        } else {
            0.0
        };
        
        BacktestMetrics {
            initial_capital: self.config.initial_capital,
            final_capital: self.current_capital,
            total_return: self.current_capital - self.config.initial_capital,
            total_return_percentage: ((self.current_capital - self.config.initial_capital) / self.config.initial_capital) * 100.0,
            total_trades,
            winning_trades,
            losing_trades,
            win_rate,
            avg_win,
            avg_loss,
            largest_win,
            largest_loss,
            avg_trade_duration: 0, // Would need trade duration tracking
            sharpe_ratio,
            sortino_ratio,
            max_drawdown,
            max_drawdown_percentage,
            avg_drawdown: max_drawdown_percentage / 2.0, // Approximation
            volatility: volatility * 100.0,
            profit_factor,
            expectancy,
            risk_reward_ratio: if avg_loss > 0.0 { avg_win / avg_loss } else { 0.0 },
            total_bars: self.data.len() as u64,
            trading_days: (self.data.last().map(|d| d.timestamp).unwrap_or(0) - self.data.first().map(|d| d.timestamp).unwrap_or(0)) / 86400,
            start_date: self.data.first().map(|d| d.timestamp).unwrap_or(0),
            end_date: self.data.last().map(|d| d.timestamp).unwrap_or(0),
        }
    }

    /// Calculate drawdown curve
    fn calculate_drawdown_curve(&self) -> Vec<DrawdownPoint> {
        let mut peak = self.config.initial_capital;
        let mut drawdown_curve = Vec::new();
        
        for point in &self.equity_curve {
            if point.equity > peak {
                peak = point.equity;
            }
            
            let drawdown = peak - point.equity;
            let drawdown_percentage = if peak > 0.0 {
                (drawdown / peak) * 100.0
            } else {
                0.0
            };
            
            drawdown_curve.push(DrawdownPoint {
                timestamp: point.timestamp,
                drawdown,
                drawdown_percentage,
            });
        }
        
        drawdown_curve
    }

    /// Calculate monthly returns
    fn calculate_monthly_returns(&self) -> Vec<MonthlyReturn> {
        let mut monthly_data: std::collections::HashMap<(i32, u32), (f64, u64)> = std::collections::HashMap::new();
        
        for point in &self.equity_curve {
            let datetime = chrono::DateTime::from_timestamp(point.timestamp, 0)
                .map(|dt| dt.naive_utc())
                .unwrap_or_default();
            let year = datetime.format("%Y").to_string().parse().unwrap_or(2024);
            let month = datetime.format("%m").to_string().parse().unwrap_or(1);
            
            let entry = monthly_data.entry((year, month)).or_insert((0.0, 0));
            entry.0 += point.pnl;
            entry.1 += 1;
        }
        
        let mut monthly_returns: Vec<MonthlyReturn> = monthly_data
            .into_iter()
            .map(|((year, month), (pnl, trades))| {
                let return_percentage = if self.config.initial_capital > 0.0 {
                    (pnl / self.config.initial_capital) * 100.0
                } else {
                    0.0
                };
                
                MonthlyReturn {
                    year,
                    month,
                    return_percentage,
                    trades: trades as u32,
                }
            })
            .collect();
        
        monthly_returns.sort_by(|a, b| (a.year, a.month).cmp(&(b.year, b.month)));
        
        monthly_returns
    }
}

// Helper module for datetime (placeholder - would use chrono in production)
mod chrono {
    use std::time::Duration;
    
    pub struct DateTime {
        timestamp: i64,
    }
    
    impl DateTime {
        pub fn from_timestamp(timestamp: i64, _offset: u32) -> Option<Self> {
            Some(Self { timestamp })
        }
    }
    
    impl DateTime {
        pub fn naive_utc(&self) -> NaiveDateTime {
            NaiveDateTime {
                year: 2024,
                month: 1,
                day: 1,
                hour: 0,
                minute: 0,
                second: 0,
            }
        }
    }
    
    pub struct NaiveDateTime {
        pub year: i32,
        pub month: u32,
        pub day: u32,
        pub hour: u32,
        pub minute: u32,
        pub second: u32,
    }
    
    impl NaiveDateTime {
        pub fn format(&self, _format: &str) -> impl std::fmt::Display {
            format!("{:04}-{:02}", self.year, self.month)
        }
    }
}
