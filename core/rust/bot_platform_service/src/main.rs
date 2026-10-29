//! Bot Platform Service Binary

use bot_platform_service::{
    BotManager, ArbitrageConfig, GridConfig, MarketMakingConfig, SniperConfig,
};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("TigerSwap Bot Platform Service");
    println!("=====================");

    let manager = BotManager::new();
    println!("Bot Manager initialized");

    println!("\nCreating Grid Trading Bot...");
    let grid_id = manager
        .create_grid_bot("grid-1".to_string(), GridConfig::new(10, 0.01, 100.0))
        .await;
    println!("Created Grid Bot {}", grid_id);

    println!("\nCreating Market Making Bot...");
    let mm_id = manager
        .create_mm_bot("mm-1".to_string(), MarketMakingConfig::new(0.001, 1.0))
        .await;
    println!("Created MM Bot {}", mm_id);

    println!("\nCreating Arbitrage Bot...");
    let arb_id = manager
        .create_arbitrage_bot("arb-1".to_string(), ArbitrageConfig::new(1.0))
        .await;
    println!("Created Arbitrage Bot {}", arb_id);

    println!("\nCreating Sniper Bot...");
    let sniper_id = manager
        .create_sniper_bot(
            "sniper-1".to_string(),
            SniperConfig::new(vec!["0xnew_token".to_string()]),
        )
        .await;
    println!("Created Sniper Bot {}", sniper_id);

    println!("\nListing all bots...");
    for bot in manager.list_bots().await {
        println!("- {} ({:?}) - {:?}", bot.bot_id, bot.bot_type, bot.status);
    }

    println!("\nBot Platform Service ready!");

    Ok(())
}
