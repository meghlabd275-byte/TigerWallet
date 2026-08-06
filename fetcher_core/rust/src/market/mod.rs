use std::sync::Arc;
use std::time::Instant;

use anyhow::Result;
use async_trait::async_trait;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use tokio::sync::RwLock;
use tracing::{error, info};

use crate::{Fetcher, FetcherConfig, FetcherMetrics, FetcherState, FetcherType, FetchParams, FetchResult};

/// Market data fetcher for prices, order books, and trading data
pub struct MarketFetcher {
    name: String,
    state: RwLock<FetcherState>,
    http_client: Client,
    cache: Arc<crate::cache::FetcherCache>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceData {
    pub symbol: String,
    pub price: f64,
    pub change_24h: f64,
    pub change_percent_24h: f64,
    pub volume_24h: f64,
    pub market_cap: f64,
    pub high_24h: f64,
    pub low_24h: f64,
    pub timestamp: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrderBookEntry {
    pub price: f64,
    pub quantity: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrderBookData {
    pub symbol: String,
    pub bids: Vec<OrderBookEntry>,
    pub asks: Vec<OrderBookEntry>,
    pub spread: f64,
    pub timestamp: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketTicker {
    pub symbol: String,
    pub last_price: f64,
    pub bid_price: f64,
    pub ask_price: f64,
    pub volume_24h: f64,
    pub timestamp: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HistoricalData {
    pub symbol: String,
    pub interval: String,
    pub data: Vec<OHLCV>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OHLCV {
    pub timestamp: i64,
    pub open: f64,
    pub high: f64,
    pub low: f64,
    pub close: f64,
    pub volume: f64,
}

impl MarketFetcher {
    pub fn new(cache: Arc<crate::cache::FetcherCache>) -> Self {
        Self {
            name: "market_fetcher".to_string(),
            state: RwLock::new(FetcherState::default()),
            http_client: Client::builder()
                .timeout(std::time::Duration::from_secs(10))
                .build()
                .expect("Failed to create HTTP client"),
            cache,
        }
    }

    /// Fetch current price for symbols
    pub async fn fetch_prices(&self, symbols: &[String]) -> Result<Vec<PriceData>> {
        let mut prices = Vec::new();
        
        for symbol in symbols {
            let cache_key = format!("price:{}", symbol);
            
            // Try cache first
            if let Ok(cached) = self.cache.get::<PriceData>(&cache_key).await {
                prices.push(cached);
                continue;
            }

            // In production, fetch from real APIs (CoinGecko, Binance, etc.)
            let price_data = PriceData {
                symbol: symbol.clone(),
                price: 0.0,  // Would be fetched from API
                change_24h: 0.0,
                change_percent_24h: 0.0,
                volume_24h: 0.0,
                market_cap: 0.0,
                high_24h: 0.0,
                low_24h: 0.0,
                timestamp: chrono::Utc::now().timestamp(),
            };

            // Cache for 30 seconds
            let _ = self.cache.set(&cache_key, &price_data, 30).await;
            prices.push(price_data);
        }

        Ok(prices)
    }

    /// Fetch order book data
    pub async fn fetch_order_book(&self, symbol: &str) -> Result<OrderBookData> {
        let cache_key = format!("orderbook:{}", symbol);
        
        // Try cache first (5 second TTL)
        if let Ok(cached) = self.cache.get::<OrderBookData>(&cache_key).await {
            return Ok(cached);
        }

        // In production, fetch from exchange APIs
        let order_book = OrderBookData {
            symbol: symbol.to_string(),
            bids: vec![],
            asks: vec![],
            spread: 0.0,
            timestamp: chrono::Utc::now().timestamp(),
        };

        // Cache for 5 seconds
        let _ = self.cache.set(&cache_key, &order_book, 5).await;
        
        Ok(order_book)
    }

    /// Fetch historical OHLCV data
    pub async fn fetch_ohlcv(
        &self, 
        symbol: &str, 
        interval: &str, 
        limit: usize
    ) -> Result<HistoricalData> {
        let cache_key = format!("ohlcv:{}:{}:{}", symbol, interval, limit);
        
        // Try cache first (1 minute TTL)
        if let Ok(cached) = self.cache.get::<HistoricalData>(&cache_key).await {
            return Ok(cached);
        }

        // In production, fetch from exchange APIs
        let data = HistoricalData {
            symbol: symbol.to_string(),
            interval: interval.to_string(),
            data: vec![],
        };

        // Cache for 60 seconds
        let _ = self.cache.set(&cache_key, &data, 60).await;
        
        Ok(data)
    }
}

#[async_trait]
impl Fetcher for MarketFetcher {
    fn fetcher_type(&self) -> FetcherType {
        FetcherType::Market
    }

    fn name(&self) -> &str {
        &self.name
    }

    async fn fetch(&self, params: FetchParams) -> Result<FetchResult> {
        let start = Instant::now();
        
        let symbols = params.symbols.clone().unwrap_or_else(|| vec!["BTC".to_string()]);
        
        let data = match self.fetch_prices(&symbols).await {
            Ok(prices) => {
                let result = serde_json::json!({
                    "type": "prices",
                    "data": prices
                });
                Ok(result)
            }
            Err(e) => {
                error!("Market fetch error: {}", e);
                Err(e)
            }
        };

        let elapsed = start.elapsed().as_millis() as u64;
        
        match data {
            Ok(result_data) => {
                Ok(FetchResult::success(
                    result_data,
                    "market".to_string(),
                    elapsed,
                    false,
                ))
            }
            Err(e) => {
                Ok(FetchResult::error(
                    "market".to_string(),
                    e.to_string(),
                    elapsed,
                ))
            }
        }
    }

    async fn initialize(&mut self, config: &FetcherConfig) -> Result<()> {
        let mut state = self.state.write().await;
        state.config = config.clone();
        state.status = FetcherStatus::Idle;
        info!("Market fetcher initialized with config: {:?}", config);
        Ok(())
    }

    async fn shutdown(&mut self) -> Result<()> {
        let mut state = self.state.write().await;
        state.status = FetcherStatus::Stopped;
        info!("Market fetcher shut down");
        Ok(())
    }
}

/// Price alert configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriceAlert {
    pub id: String,
    pub tenant_id: uuid::Uuid,
    pub symbol: String,
    pub condition: String, // "above" or "below"
    pub target_price: f64,
    pub enabled: bool,
    pub created_at: i64,
    pub triggered_at: Option<i64>,
}

impl PriceAlert {
    pub fn check(&self, current_price: f64) -> bool {
        match self.condition.as_str() {
            "above" => current_price >= self.target_price,
            "below" => current_price <= self.target_price,
            _ => false,
        }
    }
}
