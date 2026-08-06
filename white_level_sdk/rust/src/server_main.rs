//! White Level Server Binary

use white_level_sdk::{Config, WhiteLevelProduct};
use white_level_sdk::types::*;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("White Level Server - TigerWallet SDK");
    println!("====================================\n");
    
    println!("This is the server-side component for hosting a White Level product.");
    println!("Use the client binary for connecting to Super Admin.");
    
    Ok(())
}
