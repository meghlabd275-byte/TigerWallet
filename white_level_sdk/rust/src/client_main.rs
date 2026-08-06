//! White Level Client Binary

use white_level_sdk::{WhiteLevelClient, Config};
use white_level_sdk::types::*;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize logging
    tracing_subscriber::fmt()
        .with_max_level(tracing::Level::INFO)
        .init();

    println!("White Level Client - TigerWallet SDK");
    println!("====================================\n");

    // Load configuration from environment
    let api_key = std::env::var("WL_API_KEY")
        .expect("WL_API_KEY environment variable required");
    let client_id = std::env::var("WL_CLIENT_ID")
        .expect("WL_CLIENT_ID environment variable required");
    let super_admin_url = std::env::var("WL_SUPER_ADMIN_URL")
        .unwrap_or_else(|_| "http://localhost:8082".to_string());
    let product = std::env::var("WL_PRODUCT")
        .unwrap_or_else(|_| "master_wallet".to_string());

    let product = match product.as_str() {
        "master_wallet" => WhiteLevelProduct::MasterWallet,
        "user_wallet" => WhiteLevelProduct::UserWallet,
        "bots" => WhiteLevelProduct::Bots,
        "bots_clients" => WhiteLevelProduct::BotsClients,
        "project_party" => WhiteLevelProduct::ProjectParty,
        _ => panic!("Invalid product: {}", product),
    };

    // Create configuration
    let config = Config::new(&super_admin_url, &api_key)
        .with_logging(true)
        .with_log_level("debug")
        .with_heartbeat_interval(std::time::Duration::from_secs(30))
        .with_caching(true)
        .with_cache_ttl(std::time::Duration::from_secs(300));

    // Create client
    let mut client = WhiteLevelClient::new(config, product).await?;

    // Connect to Super Admin
    println!("Connecting to Super Admin at {}...", super_admin_url);
    match client.connect(&client_id).await {
        Ok(response) => {
            println!("✅ Connected successfully!");
            println!("   Connection ID: {}", response.connection_id);
            println!("   Session expires: {}", response.expires_at);
            println!("   Features: {:?}", response.config.features);
            println!();
        }
        Err(e) => {
            eprintln!("❌ Connection failed: {}", e);
            std::process::exit(1);
        }
    }

    // Start heartbeat
    client.start_heartbeat();
    println!("💓 Heartbeat started\n");

    // Get enabled fetchers
    let fetchers = client.get_enabled_fetchers();
    println!("📡 Enabled fetchers:");
    for fetcher in &fetchers {
        println!("   - {}", fetcher);
    }
    println!();

    // Example: Fetch prices
    println!("📊 Fetching prices...");
    match client.fetch(
        FetcherType::Prices,
        serde_json::json!({
            "symbols": ["BTC", "ETH", "USDT"]
        })
    ).await {
        Ok(response) => {
            println!("✅ Prices fetched (cached: {})", response.cached);
            println!("   Latency: {}ms", response.latency_ms);
            println!("   Data: {}", response.data);
        }
        Err(e) => {
            eprintln!("❌ Fetch failed: {}", e);
        }
    }

    println!();
    println!("🛑 Press Ctrl+C to disconnect...\n");

    // Wait for interrupt
    tokio::signal::ctrl_c().await?;

    // Cleanup
    client.stop_heartbeat();
    client.disconnect().await?;
    
    println!("\n👋 Disconnected. Goodbye!");

    Ok(())
}
