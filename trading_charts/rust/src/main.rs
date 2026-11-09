//! TigerWallet Advanced Trading Charts
//! High-performance trading charts with TradingView integration

use actix_web::{web, App, HttpResponse, HttpServer, Responder, middleware};
use chrono::Utc;
use serde::{Deserialize, Serialize};
use sqlx::{postgres::PgPoolOptions, Pool, Postgres, Row};

// ============================================================================
// Data Models
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChartData {
    pub symbol: String,
    pub timeframe: String,
    pub candles: Vec<Candle>,
    pub volume: Vec<VolumeBar>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Candle {
    pub time: i64,
    pub open: f64,
    pub high: f64,
    pub low: f64,
    pub close: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VolumeBar {
    pub time: i64,
    pub volume: f64,
    pub color: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TechnicalIndicator {
    pub name: String,
    pub data: Vec<f64>,
    pub config: serde_json::Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrderBook {
    pub symbol: String,
    pub bids: Vec<PriceLevel>,
    pub asks: Vec<PriceLevel>,
    pub timestamp: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceLevel {
    pub price: f64,
    pub quantity: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketStats {
    pub symbol: String,
    pub price: f64,
    pub change_24h: f64,
    pub change_percent_24h: f64,
    pub high_24h: f64,
    pub low_24h: f64,
    pub volume_24h: f64,
    pub turnover_24h: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChartIndicator {
    pub id: String,
    pub name: String,
    pub indicator_type: String,
    pub settings: serde_json::Value,
    pub is_visible: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradingPair {
    pub id: String,
    pub base_asset: String,
    pub quote_asset: String,
    pub symbol: String,
    pub status: String,
    pub min_price: f64,
    pub max_price: f64,
    pub tick_size: f64,
    pub min_quantity: f64,
    pub max_quantity: f64,
    pub step_size: f64,
}

// ============================================================================
// Application State
// ============================================================================

pub struct AppState {
    pub db_pool: Pool<Postgres>,
}

// ============================================================================
// Database
// ============================================================================

async fn init_database(database_url: &str) -> Result<Pool<Postgres>, sqlx::Error> {
    let pool = PgPoolOptions::new()
        .max_connections(50)
        .connect(database_url)
        .await?;
    Ok(pool)
}

async fn create_schema(pool: &Pool<Postgres>) -> Result<(), sqlx::Error> {
    sqlx::query(
        r#"
        CREATE TABLE IF NOT EXISTS trading_pairs (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            base_asset VARCHAR(20) NOT NULL,
            quote_asset VARCHAR(20) NOT NULL,
            symbol VARCHAR(20) UNIQUE NOT NULL,
            status VARCHAR(20) DEFAULT 'active',
            min_price DECIMAL(20, 8),
            max_price DECIMAL(20, 8),
            tick_size DECIMAL(20, 8),
            min_quantity DECIMAL(20, 8),
            max_quantity DECIMAL(20, 8),
            step_size DECIMAL(20, 8),
            created_at TIMESTAMP DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS candlestick_data (
            id BIGSERIAL PRIMARY KEY,
            symbol VARCHAR(20) NOT NULL,
            timeframe VARCHAR(10) NOT NULL,
            time TIMESTAMP NOT NULL,
            open DECIMAL(20, 8) NOT NULL,
            high DECIMAL(20, 8) NOT NULL,
            low DECIMAL(20, 8) NOT NULL,
            close DECIMAL(20, 8) NOT NULL,
            volume DECIMAL(30, 8) NOT NULL,
            UNIQUE(symbol, timeframe, time)
        );

        CREATE TABLE IF NOT EXISTS chart_indicators (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name VARCHAR(50) NOT NULL,
            indicator_type VARCHAR(50) NOT NULL,
            settings JSONB DEFAULT '{}',
            is_visible BOOLEAN DEFAULT true,
            created_at TIMESTAMP DEFAULT NOW()
        );

        CREATE INDEX IF NOT EXISTS idx_candles_symbol_time ON candlestick_data(symbol, timeframe, time);
        "#
    )
    .execute(pool)
    .await?;

    Ok(())
}

// ============================================================================
// Technical Indicators
// ============================================================================

fn calculate_sma(data: &[f64], period: usize) -> Vec<f64> {
    if data.len() < period {
        return vec![];
    }

    let mut result = vec![0.0; period - 1];
    for i in (period - 1)..data.len() {
        let sum: f64 = data[i + 1 - period..=i].iter().sum();
        result.push(sum / period as f64);
    }
    result
}

fn calculate_ema(data: &[f64], period: usize) -> Vec<f64> {
    if data.len() < period {
        return vec![];
    }

    let multiplier = 2.0 / (period as f64 + 1.0);
    let mut result = vec![0.0; period - 1];
    
    // First EMA is SMA
    let sum: f64 = data[..period].iter().sum();
    result.push(sum / period as f64);
    
    for i in period..data.len() {
        let ema = (data[i] - result.last().unwrap()) * multiplier + result.last().unwrap();
        result.push(ema);
    }
    result
}

fn calculate_rsi(data: &[f64], period: usize) -> Vec<f64> {
    if data.len() <= period {
        return vec![];
    }

    let mut result = vec![0.0; period];
    let mut gains = vec![];
    let mut losses = vec![];

    for i in 1..data.len() {
        let change = data[i] - data[i - 1];
        if change > 0.0 {
            gains.push(change);
            losses.push(0.0);
        } else {
            gains.push(0.0);
            losses.push(change.abs());
        }
    }

    let mut avg_gain: f64 = gains[..period].iter().sum::<f64>() / period as f64;
    let mut avg_loss: f64 = losses[..period].iter().sum::<f64>() / period as f64;

    for i in period..gains.len() {
        avg_gain = (avg_gain * (period - 1) as f64 + gains[i]) / period as f64;
        avg_loss = (avg_loss * (period - 1) as f64 + losses[i]) / period as f64;
        
        let rs = if avg_loss == 0.0 { 100.0 } else { avg_gain / avg_loss };
        let rsi = 100.0 - (100.0 / (1.0 + rs));
        result.push(rsi);
    }

    result
}

fn calculate_bollinger_bands(data: &[f64], period: usize, std_dev: f64) -> (Vec<f64>, Vec<f64>, Vec<f64>) {
    let sma = calculate_sma(data, period);
    let mut upper = vec![];
    let mut lower = vec![];

    for i in (period - 1)..data.len() {
        let slice = &data[i + 1 - period..=i];
        let mean = sma[i - period + 1];
        
        let variance: f64 = slice.iter().map(|x| (x - mean).powi(2)).sum::<f64>() / period as f64;
        let std = variance.sqrt();
        
        upper.push(mean + std_dev * std);
        lower.push(mean - std_dev * std);
    }

    (sma, upper, lower)
}

fn calculate_macd(data: &[f64], fast: usize, slow: usize, signal: usize) -> (Vec<f64>, Vec<f64>, Vec<f64>) {
    let fast_ema = calculate_ema(data, fast);
    let slow_ema = calculate_ema(data, slow);
    
    let mut macd_line = vec![];
    for i in 0..fast_ema.len().min(slow_ema.len()) {
        if i >= slow_ema.len() - fast_ema.len() {
            let idx = slow_ema.len() - fast_ema.len() + i;
            if idx < fast_ema.len() {
                macd_line.push(fast_ema[idx] - slow_ema[i]);
            }
        }
    }
    
    let signal_line = calculate_ema(&macd_line, signal);
    let mut histogram = vec![];
    
    for i in 0..macd_line.len().min(signal_line.len()) {
        histogram.push(macd_line[i + signal_line.len() - macd_line.len()] - signal_line[i]);
    }

    (macd_line, signal_line, histogram)
}

// ============================================================================
// API Handlers
// ============================================================================

async fn get_chart_data(
    state: web::Data<AppState>,
    web::Query(params): web::Query<std::collections::HashMap<String, String>>,
) -> impl Responder {
    let symbol = params.get("symbol").cloned().unwrap_or_else(|| "ETHUSDT".to_string());
    let timeframe = params.get("timeframe").cloned().unwrap_or_else(|| "1h".to_string());
    let limit: i32 = params.get("limit").and_then(|s| s.parse().ok()).unwrap_or(100);

    let rows = match sqlx::query(
        "SELECT time, open, high, low, close, volume FROM candlestick_data
         WHERE replace(replace(upper(symbol), '/', ''), '-', '') = $1
           AND timeframe = $2
         ORDER BY time DESC LIMIT $3",
    )
    .bind(normalize_symbol(&symbol))
    .bind(&timeframe)
    .bind(limit)
    .fetch_all(&state.db_pool)
    .await
    {
        Ok(rows) => rows,
        Err(e) => {
            tracing::error!(symbol = %symbol, error = %e, "failed to load chart data");
            return HttpResponse::ServiceUnavailable()
                .json(serde_json::json!({ "error": "chart data is unavailable" }));
        }
    };

    let candles: Vec<Candle> = rows
        .iter()
        .rev()
        .map(|row| {
            let time: chrono::DateTime<Utc> = row.try_get("time").unwrap_or_else(|_| Utc::now());
            Candle {
                time: time.timestamp(),
                open: row.try_get::<f64, _>("open").unwrap_or(0.0),
                high: row.try_get::<f64, _>("high").unwrap_or(0.0),
                low: row.try_get::<f64, _>("low").unwrap_or(0.0),
                close: row.try_get::<f64, _>("close").unwrap_or(0.0),
            }
        })
        .collect();

    let volume: Vec<VolumeBar> = rows
        .iter()
        .rev()
        .map(|row| {
            let time: chrono::DateTime<Utc> = row.try_get("time").unwrap_or_else(|_| Utc::now());
            let close: f64 = row.try_get("close").unwrap_or(0.0);
            let open: f64 = row.try_get("open").unwrap_or(0.0);
            VolumeBar {
                time: time.timestamp(),
                volume: row.try_get::<f64, _>("volume").unwrap_or(0.0),
                color: if close >= open { "#22C55E".to_string() } else { "#EF4444".to_string() },
            }
        })
        .collect();

    let chart_data = ChartData {
        symbol,
        timeframe,
        candles,
        volume,
    };

    HttpResponse::Ok().json(serde_json::json!({ "data": chart_data }))
}

async fn get_indicators(
    state: web::Data<AppState>,
    web::Query(params): web::Query<std::collections::HashMap<String, String>>,
) -> impl Responder {
    let symbol = params.get("symbol").cloned().unwrap_or_else(|| "ETHUSDT".to_string());
    let indicator_name = params.get("indicator").cloned().unwrap_or_default();
    let period: usize = params.get("period").and_then(|s| s.parse().ok()).unwrap_or(14);

    // Get price data
    let rows = match sqlx::query(
        "SELECT close FROM candlestick_data
          WHERE replace(replace(upper(symbol), '/', ''), '-', '') = $1
          ORDER BY time DESC LIMIT 100",
    )
    .bind(normalize_symbol(&symbol))
    .fetch_all(&state.db_pool)
    .await
    {
        Ok(rows) => rows,
        Err(e) => {
            tracing::error!(symbol = %symbol, error = %e, "failed to load indicator prices");
            return HttpResponse::ServiceUnavailable()
                .json(serde_json::json!({ "error": "indicator data is unavailable" }));
        }
    };

    let prices: Vec<f64> = rows
        .iter()
        .rev()
        .map(|row| row.try_get::<f64, _>("close").unwrap_or(0.0))
        .collect();

    let mut indicators = Vec::new();

    // Calculate requested indicators
    if indicator_name.is_empty() || indicator_name.contains("sma") {
        indicators.push(TechnicalIndicator {
            name: "SMA".to_string(),
            data: calculate_sma(&prices, 20),
            config: serde_json::json!({"period": 20}),
        });
    }

    if indicator_name.is_empty() || indicator_name.contains("ema") {
        indicators.push(TechnicalIndicator {
            name: "EMA".to_string(),
            data: calculate_ema(&prices, period),
            config: serde_json::json!({"period": period}),
        });
    }

    if indicator_name.is_empty() || indicator_name.contains("rsi") {
        indicators.push(TechnicalIndicator {
            name: "RSI".to_string(),
            data: calculate_rsi(&prices, period),
            config: serde_json::json!({"period": period}),
        });
    }

    if indicator_name.is_empty() || indicator_name.contains("bb") {
        let (middle, upper, lower) = calculate_bollinger_bands(&prices, 20, 2.0);
        indicators.push(TechnicalIndicator {
            name: "BB_Middle".to_string(),
            data: middle,
            config: serde_json::json!({"period": 20, "stdDev": 2.0}),
        });
        indicators.push(TechnicalIndicator {
            name: "BB_Upper".to_string(),
            data: upper,
            config: serde_json::json!({"period": 20, "stdDev": 2.0}),
        });
        indicators.push(TechnicalIndicator {
            name: "BB_Lower".to_string(),
            data: lower,
            config: serde_json::json!({"period": 20, "stdDev": 2.0}),
        });
    }

    if indicator_name.is_empty() || indicator_name.contains("macd") {
        let (macd, signal, hist) = calculate_macd(&prices, 12, 26, 9);
        indicators.push(TechnicalIndicator {
            name: "MACD".to_string(),
            data: macd,
            config: serde_json::json!({"fast": 12, "slow": 26, "signal": 9}),
        });
        indicators.push(TechnicalIndicator {
            name: "MACD_Signal".to_string(),
            data: signal,
            config: serde_json::json!({"fast": 12, "slow": 26, "signal": 9}),
        });
        indicators.push(TechnicalIndicator {
            name: "MACD_Histogram".to_string(),
            data: hist,
            config: serde_json::json!({"fast": 12, "slow": 26, "signal": 9}),
        });
    }

    HttpResponse::Ok().json(serde_json::json!({ "indicators": indicators }))
}

/// Strips the separators exchanges disagree on, so ETH/USDT, ETH-USDT and
/// ETHUSDT all resolve to the same pair.
fn normalize_symbol(raw: &str) -> String {
    raw.chars()
        .filter(|ch| *ch != '/' && *ch != '-' && *ch != '_')
        .flat_map(|ch| ch.to_uppercase())
        .collect()
}

/// Loads one side of a pair's aggregated order book.
async fn load_levels(
    pool: &Pool<Postgres>,
    symbol: &str,
    side: &str,
    descending: bool,
    depth: i64,
) -> Result<Vec<PriceLevel>, sqlx::Error> {
    let sql = format!(
        "SELECT l.price, l.total_quantity
           FROM order_book_levels l
           JOIN trading_pairs p ON p.id = l.pair_id
          WHERE replace(replace(upper(p.symbol), '/', ''), '-', '') = $1
            AND l.side = $2
          ORDER BY l.price {} LIMIT $3",
        if descending { "DESC" } else { "ASC" }
    );

    let rows = sqlx::query(&sql)
        .bind(symbol)
        .bind(side)
        .bind(depth)
        .fetch_all(pool)
        .await?;

    Ok(rows
        .iter()
        .map(|row| PriceLevel {
            price: row.try_get::<f64, _>("price").unwrap_or(0.0),
            quantity: row.try_get::<f64, _>("total_quantity").unwrap_or(0.0),
        })
        .collect())
}

async fn get_order_book(
    state: web::Data<AppState>,
    web::Query(params): web::Query<std::collections::HashMap<String, String>>,
) -> impl Responder {
    let symbol = normalize_symbol(
        params.get("symbol").map(String::as_str).unwrap_or("ETHUSDT"),
    );
    // Bounded so a caller cannot ask for an unbounded result set.
    let depth = params
        .get("depth")
        .and_then(|s| s.parse::<i64>().ok())
        .unwrap_or(20)
        .clamp(1, 500);

    // Bids descend from the best price, asks ascend from it.
    let bids = match load_levels(&state.db_pool, &symbol, "buy", true, depth).await {
        Ok(levels) => levels,
        Err(e) => {
            tracing::error!(symbol = %symbol, error = %e, "failed to load bid levels");
            return HttpResponse::ServiceUnavailable()
                .json(serde_json::json!({ "error": "order book is unavailable" }));
        }
    };

    let asks = match load_levels(&state.db_pool, &symbol, "sell", false, depth).await {
        Ok(levels) => levels,
        Err(e) => {
            tracing::error!(symbol = %symbol, error = %e, "failed to load ask levels");
            return HttpResponse::ServiceUnavailable()
                .json(serde_json::json!({ "error": "order book is unavailable" }));
        }
    };

    let order_book = OrderBook {
        symbol,
        bids,
        asks,
        timestamp: Utc::now().timestamp(),
    };

    HttpResponse::Ok().json(serde_json::json!({ "orderBook": order_book }))
}

/// Computes 24h statistics from the aggregated candle window.
///
/// change_percent_24h is relative to the window's opening price; turnover is
/// the quote-denominated sum supplied by the query.
fn derive_market_stats(
    symbol: String,
    open: f64,
    high: f64,
    low: f64,
    close: f64,
    volume: f64,
    turnover: f64,
) -> MarketStats {
    let change = close - open;
    let change_percent = if open == 0.0 {
        0.0
    } else {
        (change / open) * 100.0
    };

    MarketStats {
        symbol,
        price: close,
        change_24h: change,
        change_percent_24h: change_percent,
        high_24h: high,
        low_24h: low,
        volume_24h: volume,
        turnover_24h: turnover,
    }
}

async fn get_market_stats(
    state: web::Data<AppState>,
    web::Query(params): web::Query<std::collections::HashMap<String, String>>,
) -> impl Responder {
    let symbol = normalize_symbol(
        params.get("symbol").map(String::as_str).unwrap_or("ETHUSDT"),
    );
    let timeframe = params.get("timeframe").map(String::as_str).unwrap_or("1h");

    let row = match sqlx::query(
        "SELECT
             (array_agg(open  ORDER BY time ASC))[1]  AS open_24h,
             MAX(high)                                AS high_24h,
             MIN(low)                                 AS low_24h,
             (array_agg(close ORDER BY time DESC))[1] AS close_24h,
             SUM(volume)                              AS volume_24h,
             SUM(volume * close)                      AS turnover_24h
           FROM candlestick_data
          WHERE replace(replace(upper(symbol), '/', ''), '-', '') = $1
            AND timeframe = $2
            AND time >= NOW() - INTERVAL '24 hours'",
    )
    .bind(&symbol)
    .bind(timeframe)
    .fetch_one(&state.db_pool)
    .await
    {
        Ok(row) => row,
        Err(e) => {
            tracing::error!(symbol = %symbol, error = %e, "failed to load market stats");
            return HttpResponse::ServiceUnavailable()
                .json(serde_json::json!({ "error": "market statistics are unavailable" }));
        }
    };

    // The aggregates are NULL when the window holds no candles; reporting a
    // zeroed ticker would be indistinguishable from a real, flat market.
    let open: Option<f64> = row.try_get("open_24h").unwrap_or(None);
    let close: Option<f64> = row.try_get("close_24h").unwrap_or(None);
    let (open, close) = match (open, close) {
        (Some(open), Some(close)) => (open, close),
        _ => {
            return HttpResponse::NotFound().json(serde_json::json!({
                "error": "no candles recorded for this pair in the last 24 hours"
            }));
        }
    };

    let stats = derive_market_stats(
        symbol,
        open,
        row.try_get::<f64, _>("high_24h").unwrap_or(open),
        row.try_get::<f64, _>("low_24h").unwrap_or(open),
        close,
        row.try_get::<f64, _>("volume_24h").unwrap_or(0.0),
        row.try_get::<f64, _>("turnover_24h").unwrap_or(0.0),
    );

    HttpResponse::Ok().json(serde_json::json!({ "stats": stats }))
}

async fn get_trading_pairs(state: web::Data<AppState>) -> impl Responder {
    let rows = match sqlx::query(
        "SELECT id, base_asset, quote_asset, symbol, status, min_price, max_price,
         tick_size, min_quantity, max_quantity, step_size FROM trading_pairs WHERE status = 'active'",
    )
    .fetch_all(&state.db_pool)
    .await
    {
        Ok(rows) => rows,
        Err(e) => {
            tracing::error!(error = %e, "failed to load trading pairs");
            return HttpResponse::ServiceUnavailable()
                .json(serde_json::json!({ "error": "trading pairs are unavailable" }));
        }
    };

    let rows: Vec<TradingPair> = rows
        .iter()
        .map(|row| TradingPair {
            id: row.try_get("id").unwrap_or_default(),
            base_asset: row.try_get("base_asset").unwrap_or_default(),
            quote_asset: row.try_get("quote_asset").unwrap_or_default(),
            symbol: row.try_get("symbol").unwrap_or_default(),
            status: row.try_get("status").unwrap_or_default(),
            min_price: row.try_get("min_price").unwrap_or(0.0),
            max_price: row.try_get("max_price").unwrap_or(0.0),
            tick_size: row.try_get("tick_size").unwrap_or(0.0),
            min_quantity: row.try_get("min_quantity").unwrap_or(0.0),
            max_quantity: row.try_get("max_quantity").unwrap_or(0.0),
            step_size: row.try_get("step_size").unwrap_or(0.0),
        })
        .collect();

    HttpResponse::Ok().json(serde_json::json!({ "pairs": rows }))
}

async fn search_symbol(
    state: web::Data<AppState>,
    web::Query(params): web::Query<std::collections::HashMap<String, String>>,
) -> impl Responder {
    let query = params.get("q").cloned().unwrap_or_default();

    let rows = match sqlx::query(
        "SELECT id, base_asset, quote_asset, symbol, status FROM trading_pairs
         WHERE symbol ILIKE $1 OR base_asset ILIKE $1 OR quote_asset ILIKE $1 LIMIT 10",
    )
    .bind(format!("%{}%", query))
    .fetch_all(&state.db_pool)
    .await
    {
        Ok(rows) => rows,
        Err(e) => {
            tracing::error!(query = %query, error = %e, "failed to search symbols");
            return HttpResponse::ServiceUnavailable()
                .json(serde_json::json!({ "error": "symbol search is unavailable" }));
        }
    };

    let results: Vec<TradingPair> = rows
        .iter()
        .map(|row| TradingPair {
            id: row.try_get("id").unwrap_or_default(),
            base_asset: row.try_get("base_asset").unwrap_or_default(),
            quote_asset: row.try_get("quote_asset").unwrap_or_default(),
            symbol: row.try_get("symbol").unwrap_or_default(),
            status: row.try_get("status").unwrap_or_default(),
            min_price: 0.0,
            max_price: 0.0,
            tick_size: 0.0,
            min_quantity: 0.0,
            max_quantity: 0.0,
            step_size: 0.0,
        })
        .collect();

    HttpResponse::Ok().json(serde_json::json!({ "results": results }))
}

async fn health() -> impl Responder {
    HttpResponse::Ok().json(serde_json::json!({
        "status": "healthy",
        "service": "trading_charts",
        "timestamp": Utc::now().to_rfc3339()
    }))
}

// ============================================================================
// Main
// ============================================================================

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    tracing_subscriber::fmt()
        .with_max_level(tracing::Level::INFO)
        .init();

    tracing::info!("Starting TigerWallet Trading Charts Service");

    let database_url = std::env::var("DATABASE_URL").unwrap_or_else(|_| {
        eprintln!("DATABASE_URL must be set; refusing to start without a database connection string");
        std::process::exit(1);
    });

    let db_pool = init_database(&database_url)
        .await
        .expect("Failed to connect to database");

    create_schema(&db_pool)
        .await
        .expect("Failed to create schema");

    let state = web::Data::new(AppState { db_pool });

    HttpServer::new(move || {
        App::new()
            .app_data(state.clone())
            .wrap(middleware::DefaultHeaders::new().header("X-Version", "1.0.0"))
            .route("/health", web::get().to(health))
            .route("/api/v1/charts/data", web::get().to(get_chart_data))
            .route("/api/v1/charts/indicators", web::get().to(get_indicators))
            .route("/api/v1/charts/orderbook", web::get().to(get_order_book))
            .route("/api/v1/charts/stats", web::get().to(get_market_stats))
            .route("/api/v1/charts/pairs", web::get().to(get_trading_pairs))
            .route("/api/v1/charts/search", web::get().to(search_symbol))
    })
    .bind(("0.0.0.0", 8090))?
    .run()
    .await
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn normalize_symbol_unifies_the_separators_exchanges_disagree_on() {
        for raw in ["ETHUSDT", "ETH/USDT", "ETH-USDT", "eth_usdt", "EthUsdt"] {
            assert_eq!(normalize_symbol(raw), "ETHUSDT", "failed for {raw}");
        }
    }

    #[test]
    fn derive_market_stats_computes_change_from_the_window_open() {
        let stats = derive_market_stats("ETHUSDT".to_string(), 2_000.0, 2_200.0, 1_900.0, 2_100.0, 500.0, 1_000_000.0);

        assert_eq!(stats.price, 2_100.0);
        assert_eq!(stats.change_24h, 100.0);
        assert_eq!(stats.change_percent_24h, 5.0);
        assert_eq!(stats.high_24h, 2_200.0);
        assert_eq!(stats.low_24h, 1_900.0);
    }

    #[test]
    fn derive_market_stats_handles_a_negative_move() {
        let stats = derive_market_stats("ETHUSDT".to_string(), 2_000.0, 2_000.0, 1_800.0, 1_900.0, 10.0, 19_000.0);

        assert_eq!(stats.change_24h, -100.0);
        assert_eq!(stats.change_percent_24h, -5.0);
    }

    /// A zero opening price must not produce a division by zero (NaN or inf),
    /// which would serialise as a non-finite JSON number.
    #[test]
    fn derive_market_stats_does_not_divide_by_zero() {
        let stats = derive_market_stats("NEW".to_string(), 0.0, 0.0, 0.0, 0.0, 0.0, 0.0);

        assert_eq!(stats.change_percent_24h, 0.0);
        assert!(stats.change_percent_24h.is_finite());
    }

    /// Technical indicators are computed over the returned candles, so they
    /// must agree with a hand-computed reference.
    #[test]
    fn calculate_sma_matches_a_reference_window() {
        let data = [1.0, 2.0, 3.0, 4.0, 5.0];
        let sma = calculate_sma(&data, 3);

        // Leading entries are padded, then means of each 3-wide window.
        assert_eq!(sma.len(), 5);
        assert_eq!(sma[2], 2.0); // (1+2+3)/3
        assert_eq!(sma[3], 3.0); // (2+3+4)/3
        assert_eq!(sma[4], 4.0); // (3+4+5)/3
    }

    #[test]
    fn calculate_sma_returns_empty_when_there_is_not_enough_data() {
        assert!(calculate_sma(&[1.0, 2.0], 5).is_empty());
    }
}
