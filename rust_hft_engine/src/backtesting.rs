//!
//! TigerWallet Backtesting Engine
//! High-performance backtesting for trading strategies
//!

use serde::{Deserialize, Serialize};

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
    strategy: Box<dyn Strategy>,
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
    /// Builds a backtest for `config`, `data` and `strategy`.
    ///
    /// Returns an error for a configuration that cannot produce a meaningful
    /// result, rather than running and reporting metrics computed from
    /// nonsense.
    pub fn new(
        config: BacktestConfig,
        data: Vec<PriceData>,
        strategy: Box<dyn Strategy>,
    ) -> Result<Self, BacktestError> {
        if !config.initial_capital.is_finite() || config.initial_capital <= 0.0 {
            return Err(BacktestError::InvalidConfig(
                "initial_capital must be positive".to_string(),
            ));
        }
        if !config.fee_percentage.is_finite() || config.fee_percentage < 0.0 {
            return Err(BacktestError::InvalidConfig(
                "fee_percentage must not be negative".to_string(),
            ));
        }
        if !config.slippage_percentage.is_finite() || config.slippage_percentage < 0.0 {
            return Err(BacktestError::InvalidConfig(
                "slippage_percentage must not be negative".to_string(),
            ));
        }
        if !config.position_size_percentage.is_finite()
            || config.position_size_percentage <= 0.0
            || config.position_size_percentage > 100.0
        {
            return Err(BacktestError::InvalidConfig(
                "position_size_percentage must be between 0 and 100".to_string(),
            ));
        }
        if data.is_empty() {
            return Err(BacktestError::NoData);
        }
        for bar in &data {
            if !bar.close.is_finite() || bar.close <= 0.0 {
                return Err(BacktestError::InvalidConfig(
                    "every bar must have a positive close price".to_string(),
                ));
            }
        }

        let initial_capital = config.initial_capital;
        Ok(Self {
            config,
            data,
            strategy,
            trades: Vec::new(),
            equity_curve: Vec::new(),
            current_capital: initial_capital,
            position: None,
        })
    }

    /// Runs the backtest.
    pub fn run(&mut self) -> BacktestResult {
        // Sort data by timestamp
        self.data.sort_by_key(|d| d.timestamp);

        // Reset any state from a previous run so calling run twice gives the
        // same answer rather than accumulating trades.
        self.trades.clear();
        self.equity_curve.clear();
        self.current_capital = self.config.initial_capital;
        self.position = None;

        let mut equity = self.config.initial_capital;
        let mut peak_equity = equity;
        let mut max_drawdown = 0.0;

        // Process each bar. The bar is copied out so the strategy can be
        // given the full history while `self` is borrowed mutably to record
        // the resulting trade; iterating `self.data` directly held an
        // immutable borrow across `process_bar`.
        let bars = self.data.clone();
        for (i, bar) in bars.iter().enumerate() {
            // Update equity curve
            let pnl = self.calculate_unrealized_pnl(bar.close);
            equity = self.current_capital + pnl;

            // Track peak equity and drawdown
            if equity > peak_equity {
                peak_equity = equity;
            }

            let drawdown = peak_equity - equity;
            if drawdown > max_drawdown {
                max_drawdown = drawdown;
            }

            let initial_capital = self.config.initial_capital;
            self.equity_curve.push(EquityPoint {
                timestamp: bar.timestamp,
                equity,
                pnl: equity - initial_capital,
            });

            self.process_bar(bar, i);
        }

        // Close any open position at the end.
        if self.position.is_some() {
            if let Some(last_bar) = self.data.last() {
                let close = last_bar.close;
                let timestamp = last_bar.timestamp;
                self.close_position(close, timestamp);
            }
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

    /// Applies the configured strategy to one bar.
    ///
    /// The previous version was labelled a placeholder and hardcoded a 5/20
    /// moving-average crossover, so a caller could not backtest anything but
    /// that one strategy. The strategy is now a parameter.
    fn process_bar(&mut self, bar: &PriceData, index: usize) {
        let signal = self.strategy.signal(&self.data, index);
        match signal {
            Signal::Buy if self.position.is_none() => {
                let position_size =
                    self.current_capital * self.config.position_size_percentage / 100.0;
                if position_size <= 0.0 {
                    return;
                }
                let price_with_slippage =
                    bar.close * (1.0 + self.config.slippage_percentage / 100.0);
                if price_with_slippage <= 0.0 {
                    return;
                }
                let quantity = position_size / price_with_slippage;
                self.open_position(TradeSide::Long, price_with_slippage, quantity, bar.timestamp);
            }
            Signal::Sell if self.position.is_some() => {
                let price_with_slippage =
                    bar.close * (1.0 - self.config.slippage_percentage / 100.0);
                self.close_position(price_with_slippage, bar.timestamp);
            }
            _ => {}
        }
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
            // Long and short are both supported by the PnL formula. A short's
            // proceeds are not credited to capital here, because doing so
            // without a margin model would overstate equity.
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
        // Average trade duration, in seconds, over trades that have closed.
        // The previous version reported a constant 0 with a comment saying it
        // would need duration tracking, even though entry_time and exit_time
        // were both recorded.
        let durations: Vec<u64> = self
            .trades
            .iter()
            .filter_map(|t| t.exit_time.map(|exit| exit.saturating_sub(t.entry_time) as u64))
            .collect();
        let avg_trade_duration = if durations.is_empty() {
            0
        } else {
            durations.iter().sum::<u64>() / durations.len() as u64
        };

        // Average drawdown as a percentage of peak equity, from the curve.
        // The previous version reported half the maximum drawdown, which is
        // not the average of anything.
        let curve = self.calculate_drawdown_curve();
        let avg_drawdown_percentage = if curve.is_empty() {
            0.0
        } else {
            curve.iter().map(|d| d.drawdown_percentage).sum::<f64>() / curve.len() as f64
        };

        let closed_trades: Vec<&BacktestTrade> = self.trades
            .iter()
            .filter(|t| t.status == TradeStatus::Closed)
            .collect();

        let total_trades = closed_trades.len() as u64;
        let winning_trades = closed_trades.iter().filter(|t| t.pnl.unwrap_or(0.0) > 0.0).count() as u64;
        let losing_trades = closed_trades.iter().filter(|t| t.pnl.unwrap_or(0.0) < 0.0).count() as u64;

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

        let largest_win = closed_trades
            .iter()
            .map(|t| t.pnl.unwrap_or(0.0))
            .fold(0.0f64, f64::max);
        let largest_loss = closed_trades
            .iter()
            .map(|t| t.pnl.unwrap_or(0.0))
            .fold(0.0f64, f64::min);

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

        // Whole days spanned by the data. The previous version divided the
        // timestamp difference, which is an i64, by 86400 and assigned it to a
        // u64 field, which does not compile.
        let trading_days = match (self.data.first(), self.data.last()) {
            (Some(first), Some(last)) => {
                (last.timestamp.saturating_sub(first.timestamp) / 86_400) as u64
            }
            _ => 0,
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
            avg_trade_duration,
            sharpe_ratio,
            sortino_ratio,
            max_drawdown,
            max_drawdown_percentage,
            avg_drawdown: avg_drawdown_percentage,
            volatility: volatility * 100.0,
            profit_factor,
            expectancy,
            risk_reward_ratio: if avg_loss > 0.0 { avg_win / avg_loss } else { 0.0 },
            total_bars: self.data.len() as u64,
            trading_days: trading_days,
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
            let (year, month) = civil_from_timestamp(point.timestamp);
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

// ============================================================================
// Date handling
// ============================================================================

/// Splits a Unix timestamp into (year, month) using the proleptic Gregorian
/// calendar.
///
/// The previous implementation wrapped a stand-in `chrono` module whose
/// `naive_utc` returned a hardcoded 2024-01-01 and whose `format` ignored the
/// format string and always printed the year and month. Every bar was
/// therefore grouped into January 2024, so `monthly_returns` held exactly one
/// entry no matter how long the backtest ran.
///
/// Days are counted from the civil epoch using Howard Hinnant's algorithm,
/// which is exact for all timestamps in range and needs no dependencies.
fn civil_from_timestamp(timestamp: i64) -> (i32, u32) {
    let days = timestamp.div_euclid(86_400);

    // Shift the epoch to 0000-03-01 so leap days fall at the end of the year.
    let z = days + 719_468;
    let era = if z >= 0 { z } else { z - 146_096 } / 146_097;
    let doe = (z - era * 146_097) as u64; // [0, 146096]
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146_096) / 365; // [0, 399]
    let y = yoe as i64 + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100); // [0, 365]
    let mp = (5 * doy + 2) / 153; // [0, 11]
    let month = if mp < 10 { mp + 3 } else { mp - 9 }; // [1, 12]
    let year = if month <= 2 { y + 1 } else { y };

    (year as i32, month as u32)
}

// ============================================================================
// Strategy
// ============================================================================

/// What a strategy wants to do with the current bar.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Signal {
    /// Open a position if none is open.
    Buy,
    /// Close the open position.
    Sell,
    Hold,
}

/// A trading rule the backtester replays.
///
/// `index` is the position of the current bar in `data`, so a strategy can
/// look back over the history it is given.
pub trait Strategy: Send + Sync {
    fn signal(&self, data: &[PriceData], index: usize) -> Signal;
    fn name(&self) -> &str;
}

/// A moving-average crossover.
///
/// The previous engine hardcoded 5 and 20, so no other strategy could be
/// tested and the two periods could not be varied.
pub struct MovingAverageCrossover {
    pub short_period: usize,
    pub long_period: usize,
}

impl MovingAverageCrossover {
    /// Builds a crossover with `short_period` and `long_period`.
    pub fn new(short_period: usize, long_period: usize) -> Result<Self, BacktestError> {
        if short_period == 0 || long_period == 0 {
            return Err(BacktestError::InvalidConfig(
                "moving-average periods must be greater than zero".to_string(),
            ));
        }
        if short_period >= long_period {
            return Err(BacktestError::InvalidConfig(
                "the short period must be shorter than the long period".to_string(),
            ));
        }
        Ok(Self {
            short_period,
            long_period,
        })
    }

    fn average(data: &[PriceData], index: usize, period: usize) -> Option<f64> {
        if index + 1 < period {
            return None;
        }
        let start = index + 1 - period;
        let sum: f64 = data[start..=index].iter().map(|d| d.close).sum();
        Some(sum / period as f64)
    }
}

impl Strategy for MovingAverageCrossover {
    fn signal(&self, data: &[PriceData], index: usize) -> Signal {
        let short = match Self::average(data, index, self.short_period) {
            Some(v) => v,
            None => return Signal::Hold,
        };
        let long = match Self::average(data, index, self.long_period) {
            Some(v) => v,
            None => return Signal::Hold,
        };

        if short > long {
            Signal::Buy
        } else if short < long {
            Signal::Sell
        } else {
            Signal::Hold
        }
    }

    fn name(&self) -> &str {
        "moving_average_crossover"
    }
}

/// Errors the backtester can report.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum BacktestError {
    InvalidConfig(String),
    NoData,
}

impl std::fmt::Display for BacktestError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            BacktestError::InvalidConfig(msg) => write!(f, "invalid backtest config: {}", msg),
            BacktestError::NoData => write!(f, "no price data was supplied"),
        }
    }
}

impl std::error::Error for BacktestError {}

#[cfg(test)]
mod tests;
