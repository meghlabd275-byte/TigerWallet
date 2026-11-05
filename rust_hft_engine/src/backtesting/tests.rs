//! Tests for the backtesting engine.

use super::*;

fn bar(timestamp: i64, close: f64) -> PriceData {
    PriceData {
        timestamp,
        open: close,
        high: close,
        low: close,
        close,
        volume: 1000.0,
    }
}

fn config(initial_capital: f64) -> BacktestConfig {
    BacktestConfig {
        symbol: "ETHUSDT".to_string(),
        start_time: 0,
        end_time: 2_000_000_000,
        initial_capital,
        fee_percentage: 0.1,
        slippage_percentage: 0.05,
        leverage: 1.0,
        position_size_percentage: 100.0,
    }
}

/// Builds a price series that rises then falls, so a crossover strategy
/// produces both an entry and an exit.
fn rising_then_falling() -> Vec<PriceData> {
    let mut data = Vec::new();
    let mut ts = 1_700_000_000i64;
    for i in 0..25 {
        data.push(bar(ts, 100.0 + i as f64));
        ts += 86_400;
    }
    for i in 0..25 {
        data.push(bar(ts, 124.0 - i as f64));
        ts += 86_400;
    }
    data
}

fn engine(config: BacktestConfig, data: Vec<PriceData>) -> BacktestEngine {
    let strategy = Box::new(MovingAverageCrossover::new(5, 20).unwrap());
    BacktestEngine::new(config, data, strategy).unwrap()
}

// ============================================================================
// Configuration
// ============================================================================

#[test]
fn new_rejects_an_invalid_configuration() {
    let data = rising_then_falling();
    let strategy = || Box::new(MovingAverageCrossover::new(5, 20).unwrap()) as Box<dyn Strategy>;

    let mut c = config(10_000.0);
    c.initial_capital = 0.0;
    assert!(BacktestEngine::new(c, data.clone(), strategy()).is_err());

    let mut c = config(10_000.0);
    c.initial_capital = -1.0;
    assert!(BacktestEngine::new(c, data.clone(), strategy()).is_err());

    let mut c = config(10_000.0);
    c.fee_percentage = -0.1;
    assert!(BacktestEngine::new(c, data.clone(), strategy()).is_err());

    let mut c = config(10_000.0);
    c.slippage_percentage = -0.1;
    assert!(BacktestEngine::new(c, data.clone(), strategy()).is_err());

    let mut c = config(10_000.0);
    c.position_size_percentage = 0.0;
    assert!(BacktestEngine::new(c, data.clone(), strategy()).is_err());

    let mut c = config(10_000.0);
    c.position_size_percentage = 101.0;
    assert!(BacktestEngine::new(c, data.clone(), strategy()).is_err());
}

#[test]
fn new_rejects_empty_or_invalid_data() {
    let strategy = || Box::new(MovingAverageCrossover::new(5, 20).unwrap()) as Box<dyn Strategy>;

    match BacktestEngine::new(config(10_000.0), Vec::new(), strategy()) {
        Err(BacktestError::NoData) => {}
        Err(other) => panic!("expected NoData, got {:?}", other),
        Ok(_) => panic!("an empty series must be rejected"),
    }

    // A zero or negative close price would divide by zero downstream.
    let bad = vec![bar(1, 100.0), bar(2, 0.0)];
    assert!(BacktestEngine::new(config(10_000.0), bad, strategy()).is_err());

    let bad = vec![bar(1, 100.0), bar(2, f64::NAN)];
    assert!(BacktestEngine::new(config(10_000.0), bad, strategy()).is_err());
}

#[test]
fn moving_average_crossover_validates_its_periods() {
    assert!(MovingAverageCrossover::new(0, 20).is_err());
    assert!(MovingAverageCrossover::new(5, 0).is_err());
    assert!(MovingAverageCrossover::new(20, 5).is_err());
    assert!(MovingAverageCrossover::new(20, 20).is_err());
    assert!(MovingAverageCrossover::new(5, 20).is_ok());
}

// ============================================================================
// Strategy signals
// ============================================================================

#[test]
fn crossover_signals_hold_until_both_averages_exist() {
    let s = MovingAverageCrossover::new(5, 20).unwrap();
    let data = rising_then_falling();

    for i in 0..19 {
        assert_eq!(s.signal(&data, i), Signal::Hold, "index {}", i);
    }
    // At index 19 the long average first exists, and price has been rising.
    assert_eq!(s.signal(&data, 19), Signal::Buy);
}

#[test]
fn crossover_signals_sell_when_the_short_average_falls_below() {
    let s = MovingAverageCrossover::new(5, 20).unwrap();
    let data = rising_then_falling();

    let last = data.len() - 1;
    assert_eq!(s.signal(&data, last), Signal::Sell);
}

// ============================================================================
// Running a backtest
// ============================================================================

#[test]
fn run_produces_trades_and_a_consistent_equity_curve() {
    let mut e = engine(config(10_000.0), rising_then_falling());
    let result = e.run();

    assert_eq!(result.equity_curve.len(), 50);
    assert!(
        !result.trades.is_empty(),
        "a crossover over this series should trade"
    );

    for trade in &result.trades {
        assert!(trade.exit_time.is_some());
        assert!(trade.exit_price.is_some());
        assert!(trade.pnl.is_some());
        assert!(trade.pnl_percentage.is_some());
        assert_eq!(trade.status, TradeStatus::Closed);
    }

    // Final capital must equal the initial capital plus the reported return.
    let initial = result.config.initial_capital;
    let expected_final = initial + result.metrics.total_return;
    assert!(
        (result.metrics.final_capital - expected_final).abs() < 1e-6,
        "final capital {} does not match initial {} + return {}",
        result.metrics.final_capital,
        initial,
        result.metrics.total_return
    );
}

#[test]
fn run_is_idempotent() {
    let mut e = engine(config(10_000.0), rising_then_falling());
    let first = e.run();
    let second = e.run();

    assert_eq!(
        first.trades.len(),
        second.trades.len(),
        "a second run must not accumulate trades"
    );
    assert_eq!(first.metrics.final_capital, second.metrics.final_capital);
}

#[test]
fn run_with_a_single_bar_produces_no_trades() {
    let mut e = engine(config(10_000.0), vec![bar(1_700_000_000, 100.0)]);
    let result = e.run();

    assert!(result.trades.is_empty());
    assert_eq!(result.metrics.total_trades, 0);
    assert_eq!(result.metrics.win_rate, 0.0);
    assert_eq!(result.metrics.final_capital, 10_000.0);
}

// ============================================================================
// Metrics that were previously invented
// ============================================================================

#[test]
fn average_trade_duration_is_computed_from_the_trades() {
    let mut e = engine(config(10_000.0), rising_then_falling());
    let result = e.run();

    assert!(!result.trades.is_empty());
    // Every bar is a day apart, so a trade held for N bars lasts N days.
    assert!(
        result.metrics.avg_trade_duration > 0,
        "the previous implementation always reported 0"
    );
    assert_eq!(result.metrics.avg_trade_duration % 86_400, 0);
}

#[test]
fn average_drawdown_is_an_average_not_a_derived_constant() {
    let mut e = engine(config(10_000.0), rising_then_falling());
    let result = e.run();

    // The old implementation returned exactly half the maximum.
    let half_max = result.metrics.max_drawdown_percentage / 2.0;
    if result.metrics.max_drawdown_percentage > 0.0 {
        assert!(
            (result.metrics.avg_drawdown - half_max).abs() > f64::EPSILON,
            "average drawdown should not be exactly half the maximum"
        );
    }

    // It is the mean of the drawdown curve.
    let curve = &result.drawdown_curve;
    let mean = curve.iter().map(|d| d.drawdown_percentage).sum::<f64>() / curve.len() as f64;
    assert!((result.metrics.avg_drawdown - mean).abs() < 1e-9);
}

#[test]
fn drawdown_curve_is_non_negative() {
    let mut e = engine(config(10_000.0), rising_then_falling());
    let result = e.run();

    assert_eq!(result.drawdown_curve.len(), result.equity_curve.len());
    for point in &result.drawdown_curve {
        assert!(point.drawdown >= 0.0);
        assert!(point.drawdown_percentage >= 0.0);
    }
}

#[test]
fn trading_days_spans_the_data() {
    let mut e = engine(config(10_000.0), rising_then_falling());
    let result = e.run();

    // 50 bars, one day apart, so the span is 49 days.
    assert_eq!(result.metrics.trading_days, 49);
    assert_eq!(result.metrics.total_bars, 50);
}

#[test]
fn metrics_are_internally_consistent() {
    let mut e = engine(config(10_000.0), rising_then_falling());
    let result = e.run();
    let m = &result.metrics;

    assert_eq!(m.total_trades, m.winning_trades + m.losing_trades);
    assert_eq!(m.total_trades, result.trades.len() as u64);
    assert!((0.0..=100.0).contains(&m.win_rate));
    assert!(m.largest_win >= 0.0);
    assert!(m.largest_loss <= 0.0);
    assert!(m.volatility >= 0.0);
    assert!(m.max_drawdown >= 0.0);

    if m.total_trades > 0 {
        let expected_win_rate = m.winning_trades as f64 / m.total_trades as f64 * 100.0;
        assert!((m.win_rate - expected_win_rate).abs() < 1e-9);
    }
}

// ============================================================================
// Monthly grouping
// ============================================================================

#[test]
fn civil_from_timestamp_matches_known_dates() {
    assert_eq!(civil_from_timestamp(0), (1970, 1));
    assert_eq!(civil_from_timestamp(1_700_000_000), (2023, 11));
    assert_eq!(civil_from_timestamp(1_709_164_800), (2024, 2));
    assert_eq!(civil_from_timestamp(1_704_067_199), (2023, 12));
    assert_eq!(civil_from_timestamp(1_704_067_200), (2024, 1));
}

#[test]
fn monthly_returns_span_more_than_one_month() {
    // The previous implementation hardcoded 2024-01 for every timestamp, so
    // this always held exactly one entry.
    let mut e = engine(config(10_000.0), rising_then_falling());
    let result = e.run();

    assert!(
        result.monthly_returns.len() >= 2,
        "50 days of data spans at least two months, got {}",
        result.monthly_returns.len()
    );

    for pair in result.monthly_returns.windows(2) {
        assert!((pair[0].year, pair[0].month) < (pair[1].year, pair[1].month));
    }

    let total: u32 = result.monthly_returns.iter().map(|m| m.trades).sum();
    assert_eq!(total as usize, result.equity_curve.len());
}

// ============================================================================
// Position accounting
// ============================================================================

#[test]
fn fees_are_charged_on_both_sides() {
    let data = rising_then_falling();
    let mut with_fees = engine(config(10_000.0), data.clone());
    let result_with = with_fees.run();

    let mut free = config(10_000.0);
    free.fee_percentage = 0.0;
    free.slippage_percentage = 0.0;
    let mut without_fees = engine(free, data);
    let result_without = without_fees.run();

    assert!(!result_with.trades.is_empty());
    assert_eq!(result_with.trades.len(), result_without.trades.len());
    assert!(
        result_with.metrics.final_capital < result_without.metrics.final_capital,
        "fees must reduce the final capital"
    );
}

#[test]
fn trade_ids_are_unique() {
    let mut e = engine(config(10_000.0), rising_then_falling());
    let result = e.run();

    let mut ids: Vec<&str> = result.trades.iter().map(|t| t.id.as_str()).collect();
    ids.sort_unstable();
    let before = ids.len();
    ids.dedup();
    assert_eq!(before, ids.len(), "trade ids must be unique");
}
