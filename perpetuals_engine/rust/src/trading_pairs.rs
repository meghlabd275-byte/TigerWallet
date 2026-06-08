//! TigerWallet Perpetuals Trading Pairs
//! Support for 50+ trading pairs

use rust_decimal::Decimal;
use rust_decimal_macros::dec;
use serde::{Deserialize, Serialize};

/// All supported perpetual trading pairs
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradingPair {
    pub symbol: String,
    pub base_currency: String,
    pub quote_currency: String,
    pub tick_size: Decimal,
    pub min_order_size: Decimal,
    pub max_order_size: Decimal,
    pub max_leverage: Decimal,
    pub maker_fee: Decimal,
    pub taker_fee: Decimal,
    pub maintenance_margin_rate: Decimal,
    pub initial_margin_rate: Decimal,
    pub price_precision: u32,
    pub quantity_precision: u32,
    pub enabled: bool,
    pub category: String,
}

impl TradingPair {
    pub fn new(symbol: &str, base: &str, quote: &str, category: &str) -> Self {
        Self {
            symbol: symbol.to_string(),
            base_currency: base.to_string(),
            quote_currency: quote.to_string(),
            tick_size: dec!(0.5),
            min_order_size: dec!(0.001),
            max_order_size: dec!(1000000),
            max_leverage: dec!(100),
            maker_fee: dec!(0.0001),
            taker_fee: dec!(0.0002),
            maintenance_margin_rate: dec!(0.005),
            initial_margin_rate: dec!(0.01),
            price_precision: 2,
            quantity_precision: 4,
            enabled: true,
            category: category.to_string(),
        }
    }
}

/// Get all trading pairs
pub fn get_all_trading_pairs() -> Vec<TradingPair> {
    vec![
        // Crypto Majors
        TradingPair::new("BTC-USD", "BTC", "USD", "Crypto Majors"),
        TradingPair::new("ETH-USD", "ETH", "USD", "Crypto Majors"),
        TradingPair::new("BNB-USD", "BNB", "USD", "Crypto Majors"),
        TradingPair::new("XRP-USD", "XRP", "USD", "Crypto Majors"),
        TradingPair::new("SOL-USD", "SOL", "USD", "Crypto Majors"),
        TradingPair::new("ADA-USD", "ADA", "USD", "Crypto Majors"),
        TradingPair::new("DOGE-USD", "DOGE", "USD", "Crypto Majors"),
        TradingPair::new("DOT-USD", "DOT", "USD", "Crypto Majors"),
        
        // DeFi
        TradingPair::new("UNI-USD", "UNI", "USD", "DeFi"),
        TradingPair::new("AAVE-USD", "AAVE", "USD", "DeFi"),
        TradingPair::new("MKR-USD", "MKR", "USD", "DeFi"),
        TradingPair::new("SNX-USD", "SNX", "USD", "DeFi"),
        TradingPair::new("LDO-USD", "LDO", "USD", "DeFi"),
        TradingPair::new("RPL-USD", "RPL", "USD", "DeFi"),
        TradingPair::new("GMX-USD", "GMX", "USD", "DeFi"),
        TradingPair::new("CRV-USD", "CRV", "USD", "DeFi"),
        TradingPair::new("COMP-USD", "COMP", "USD", "DeFi"),
        
        // Layer 1
        TradingPair::new("AVAX-USD", "AVAX", "USD", "Layer 1"),
        TradingPair::new("MATIC-USD", "MATIC", "USD", "Layer 1"),
        TradingPair::new("LINK-USD", "LINK", "USD", "Layer 1"),
        TradingPair::new("ATOM-USD", "ATOM", "USD", "Layer 1"),
        TradingPair::new("LTC-USD", "LTC", "USD", "Layer 1"),
        TradingPair::new("BCH-USD", "BCH", "USD", "Layer 1"),
        TradingPair::new("ALGO-USD", "ALGO", "USD", "Layer 1"),
        TradingPair::new("VET-USD", "VET", "USD", "Layer 1"),
        TradingPair::new("FIL-USD", "FIL", "USD", "Layer 1"),
        TradingPair::new("NEAR-USD", "NEAR", "USD", "Layer 1"),
        TradingPair::new("APT-USD", "APT", "USD", "Layer 1"),
        TradingPair::new("ARB-USD", "ARB", "USD", "Layer 1"),
        TradingPair::new("OP-USD", "OP", "USD", "Layer 1"),
        TradingPair::new("SUI-USD", "SUI", "USD", "Layer 1"),
        TradingPair::new("TIA-USD", "TIA", "USD", "Layer 1"),
        TradingPair::new("INJ-USD", "INJ", "USD", "Layer 1"),
        TradingPair::new("SEI-USD", "SEI", "USD", "Layer 1"),
        
        // Memecoins
        TradingPair::new("PEPE-USD", "PEPE", "USD", "Memecoins"),
        TradingPair::new("SHIB-USD", "SHIB", "USD", "Memecoins"),
        TradingPair::new("WIF-USD", "WIF", "USD", "Memecoins"),
        TradingPair::new("BONK-USD", "BONK", "USD", "Memecoins"),
        TradingPair::new("FLOKI-USD", "FLOKI", "USD", "Memecoins"),
        
        // Gaming/NFT
        TradingPair::new("GALA-USD", "GALA", "USD", "Gaming"),
        TradingPair::new("AXS-USD", "AXS", "USD", "Gaming"),
        TradingPair::new("MANA-USD", "MANA", "USD", "Gaming"),
        TradingPair::new("ENJ-USD", "ENJ", "USD", "Gaming"),
        TradingPair::new("SAND-USD", "SAND", "USD", "Gaming"),
        TradingPair::new("CHZ-USD", "CHZ", "USD", "Gaming"),
        TradingPair::new("THETA-USD", "THETA", "USD", "Gaming"),
        TradingPair::new("1INCH-USD", "1INCH", "USD", "Gaming"),
        
        // Stablecoins (for testing)
        TradingPair::new("USDC-USD", "USDC", "USD", "Stablecoins"),
        TradingPair::new("USDT-USD", "USDT", "USD", "Stablecoins"),
        TradingPair::new("DAI-USD", "DAI", "USD", "Stablecoins"),
        
        // Exotics
        TradingPair::new("XLM-USD", "XLM", "USD", "Exotics"),
        TradingPair::new("ETC-USD", "ETC", "USD", "Exotics"),
        TradingPair::new("XMR-USD", "XMR", "USD", "Exotics"),
        TradingPair::new("ZEC-USD", "ZEC", "USD", "Exotics"),
        TradingPair::new("HBAR-USD", "HBAR", "USD", "Exotics"),
        TradingPair::new("FTM-USD", "FTM", "USD", "Exotics"),
        TradingPair::new("SAND-USD", "SAND", "USD", "Exotics"),
    ]
}