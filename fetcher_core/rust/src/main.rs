use std::sync::Arc;
use std::time::Duration;

use anyhow::Result;
use clap::Parser;
use tokio::sync::RwLock;
use tracing::{error, info, Level};
use tracing_appender::rolling::{RollingFileAppender, Rotation};
use tracing_subscriber::{fmt, layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

mod blockchain;
mod cache;
mod market;

use blockchain::BlockchainFetcher;
use cache::FetcherCache;
use market::MarketFetcher;
use crate::{Fetcher, FetcherConfig, FetcherManager, FetcherState, FetcherType, FetchParams};

/// CLI arguments
#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Redis URL
    #[arg(long, default_value = "redis://localhost:6379")]
    redis_url: String,

    /// Server port
    #[arg(long, default_value = "9002")]
    port: u16,

    /// Log level
    #[arg(long, default_value = "info")]
    log_level: String,

    /// Enable memory cache fallback
    #[arg(long, default_value = "true")]
    memory_cache: bool,
}

/// Main fetcher service
pub struct FetcherService {
    fetchers: RwLock<std::collections::HashMap<FetcherType, Arc<dyn Fetcher>>>,
    cache: Arc<FetcherCache>,
    config: FetcherConfig,
}

impl FetcherService {
    pub fn new(redis_url: &str, enable_memory_fallback: bool) -> Result<Self> {
        let cache = Arc::new(FetcherCache::new(enable_memory_fallback));
        
        // Try to connect to Redis
        if let Err(e) = cache.connect(redis_url).await {
            error!("Failed to connect to Redis: {}", e);
            if !enable_memory_fallback {
                return Err(anyhow::anyhow!("Redis connection required when memory fallback is disabled"));
            }
        }

        Ok(Self {
            fetchers: RwLock::new(std::collections::HashMap::new()),
            cache,
            config: FetcherConfig::default(),
        })
    }

    /// Register all default fetchers
    pub async fn register_default_fetchers(&self) -> Result<()> {
        // Register blockchain fetcher
        let blockchain_fetcher = BlockchainFetcher::new(self.cache.clone());
        self.register_fetcher(Arc::new(blockchain_fetcher)).await?;

        // Register market fetcher
        let market_fetcher = MarketFetcher::new(self.cache.clone());
        self.register_fetcher(Arc::new(market_fetcher)).await?;

        info!("Registered all default fetchers");
        Ok(())
    }

    /// Get cache statistics
    pub async fn get_cache_stats(&self) -> cache::CacheStats {
        self.cache.stats().await
    }
}

#[async_trait]
impl FetcherManager for FetcherService {
    async fn register_fetcher(&self, fetcher: Arc<dyn Fetcher>) -> Result<()> {
        let mut fetchers = self.fetchers.write().await;
        let fetcher_type = fetcher.fetcher_type();
        fetchers.insert(fetcher_type, fetcher);
        info!("Registered fetcher: {:?}", fetcher_type);
        Ok(())
    }

    async fn unregister_fetcher(&self, fetcher_type: FetcherType) -> Result<()> {
        let mut fetchers = self.fetchers.write().await;
        fetchers.remove(&fetcher_type);
        info!("Unregistered fetcher: {:?}", fetcher_type);
        Ok(())
    }

    async fn get_fetcher(&self, fetcher_type: FetcherType) -> Option<Arc<dyn Fetcher>> {
        let fetchers = self.fetchers.read().await;
        fetchers.get(&fetcher_type).cloned()
    }

    async fn fetch(&self, fetcher_type: FetcherType, params: FetchParams) -> Result<FetchResult> {
        let fetcher = self.get_fetcher(fetcher_type).await
            .ok_or_else(|| anyhow::anyhow!("Fetcher not found: {:?}", fetcher_type))?;
        
        fetcher.fetch(params).await
    }

    async fn get_metrics(&self, fetcher_type: FetcherType) -> Option<FetcherMetrics> {
        let fetcher = self.get_fetcher(fetcher_type).await?;
        // In production, implement metric retrieval
        Some(FetcherMetrics::default())
    }

    async fn get_all_metrics(&self) -> Vec<(FetcherType, FetcherMetrics)> {
        vec![]
    }

    async fn start_all(&self) -> Result<()> {
        info!("Starting all fetchers");
        Ok(())
    }

    async fn stop_all(&self) -> Result<()> {
        info!("Stopping all fetchers");
        Ok(())
    }
}

#[tokio::main]
async fn main() -> Result<()> {
    let args = Args::parse();

    // Setup logging
    let file_appender = RollingFileAppender::new(
        Rotation::DAILY,
        "/var/log/tigerwallet",
        "fetcher.log",
    );
    
    let (non_blocking, _guard) = tracing_appender::non_blocking(file_appender);
    
    let filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new(&args.log_level));
    
    tracing_subscriber::registry()
        .with(filter)
        .with(fmt::layer().with_writer(std::io::stdout))
        .with(fmt::layer().with_writer(non_blocking).with_ansi(false))
        .init();

    info!("Starting TigerWallet Fetcher Service");
    info!("Redis URL: {}", args.redis_url);

    // Initialize fetcher service
    let service = FetcherService::new(&args.redis_url, args.memory_cache).await?;
    
    // Register fetchers
    service.register_default_fetchers().await?;

    // Start HTTP server
    let addr = format!("0.0.0.0:{}", args.port);
    info!("Starting HTTP server on {}", addr);
    
    let listener = tokio::net::TcpListener::bind(&addr).await?;
    
    loop {
        let (socket, addr) = listener.accept().await?;
        info!("New connection from: {}", addr);
        
        let service = Arc::new(service.clone());
        
        tokio::spawn(async move {
            if let Err(e) = handle_connection(socket, service).await {
                error!("Connection error: {}", e);
            }
        });
    }
}

async fn handle_connection(
    socket: tokio::net::TcpStream,
    service: Arc<FetcherService>
) -> Result<()> {
    use tokio::io::{AsyncReadExt, AsyncWriteExt};
    
    let mut buffer = [0u8; 4096];
    let mut reader = tokio::io::BufReader::new(socket);
    
    loop {
        let n = reader.read(&mut buffer).await?;
        if n == 0 {
            break;
        }
        
        let request = String::from_utf8_lossy(&buffer[..n]);
        info!("Received request: {}", request.trim());
        
        // Simple HTTP response
        let response = r#"{"status":"ok","service":"tiger-fetcher"}"#;
        let http_response = format!(
            "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\n\r\n{}",
            response.len(),
            response
        );
        
        let mut writer = tokio::io::BufWriter::new(&mut reader);
        writer.write_all(http_response.as_bytes()).await?;
        writer.flush().await?;
    }
    
    Ok(())
}

// Import needed traits and types
use async_trait::async_trait;
