// TigerSwap All Bot Types - Complete Trading Bot Platform
// Ultra-low latency Rust implementation for all bot strategies

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH};

// ============================================================================
// ALL BOT TYPES WITH FULL FEATURES
// ============================================================================

// ============================================================================
// ROLE-BASED ACCESS CONTROL
// ============================================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum BotRole {
    Admin,           // Super admin - full platform control
    BotOperator,     // Can manage all bots on platform
    Client,          // Can only manage own bots
}

impl BotRole {
    pub fn as_str(&self) -> &'static str {
        match self {
            BotRole::Admin => "admin",
            BotRole::BotOperator => "bot_operator",
            BotRole::Client => "client",
        }
    }
    
    pub fn can_manage_all_bots(&self) -> bool {
        matches!(self, BotRole::Admin | BotRole::BotOperator)
    }
    
    pub fn can_manage_fees(&self) -> bool {
        matches!(self, BotRole::Admin)
    }
    
    pub fn can_create_bots(&self) -> bool {
        matches!(self, BotRole::Admin | BotRole::BotOperator)
    }
    
    pub fn can_view_all_stats(&self) -> bool {
        matches!(self, BotRole::Admin | BotRole::BotOperator)
    }
    
    pub fn can_suspend_platform(&self) -> bool {
        matches!(self, BotRole::Admin)
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum BotType {
    MarketMaker,      // Standard MM - earn spread
    Arbitrage,        // Cross-exchange/DEX arbitrage
    Sniper,           // Fast trade execution
    Liquidity,        // Provide liquidity
    FrontRun,         // Anticipate large trades (MEV)
    MevBot,           // MEV extraction
    Sandwich,         // Sandwich attacks
    FlashLoan,        // Flash loan strategies
    CrossChain,       // Bridge arbitrage
    PerpHedge,        // Perpetual hedging
    // ========== NEW BOT TYPES ==========
    GridTrading,      // Price grid strategy bot
    DcaBot,          // Dollar-cost averaging bot
    MomentumBot,      // Trend following bot
    MeanReversion,   // Price reversion bot
    ScalpingBot,     // Quick small profits
    AiTradingBot,     // ML-based trading
    SignalBot,        // Trading signals bot
    CustomBot,        // User-defined strategy
}

impl BotType {
    pub fn as_str(&self) -> &'static str {
        match self {
            BotType::MarketMaker => "Market Maker",
            BotType::Arbitrage => "Arbitrage",
            BotType::Sniper => "Sniper",
            BotType::Liquidity => "Liquidity Provider",
            BotType::FrontRun => "Front Run",
            BotType::MevBot => "MEV Bot",
            BotType::Sandwich => "Sandwich",
            BotType::FlashLoan => "Flash Loan",
            BotType::CrossChain => "Cross-Chain",
            BotType::PerpHedge => "Perpetual Hedge",
            BotType::GridTrading => "Grid Trading",
            BotType::DcaBot => "DCA Bot",
            BotType::MomentumBot => "Momentum Bot",
            BotType::MeanReversion => "Mean Reversion",
            BotType::ScalpingBot => "Scalping Bot",
            BotType::AiTradingBot => "AI Trading Bot",
            BotType::SignalBot => "Signal Bot",
            BotType::CustomBot => "Custom Bot",
        }
    }
    
    pub fn description(&self) -> &'static str {
        match self {
            BotType::MarketMaker => "Provide liquidity and earn spread",
            BotType::Arbitrage => "Profit from price differences across exchanges",
            BotType::Sniper => "Execute trades with minimal latency",
            BotType::Liquidity => "Deepen order books and earn fees",
            BotType::FrontRun => "Anticipate and front-run large orders",
            BotType::MevBot => "Extract MEV from mempool",
            BotType::Sandwich => "Wrap trades for profit",
            BotType::FlashLoan => "Use flash loans for risk-free trades",
            BotType::CrossChain => "Bridge assets for cross-chain arbitrage",
            BotType::PerpHedge => "Hedge positions with perps",
            BotType::GridTrading => "Place buy/sell orders at grid levels for steady profits",
            BotType::DcaBot => "Dollar-cost averaging - buy at regular intervals",
            BotType::MomentumBot => "Follow market trends and momentum",
            BotType::MeanReversion => "Trade based on price returning to mean",
            BotType::ScalpingBot => "Quick small profits from small price movements",
            BotType::AiTradingBot => "AI/ML-based trading decisions",
            BotType::SignalBot => "Execute trades based on custom signals",
            BotType::CustomBot => "User-defined custom trading strategy",
        }
    }
    
    pub fn is_enabled_by_default(&self) -> bool {
        match self {
            BotType::MarketMaker | BotType::Arbitrage | BotType::Sniper | BotType::GridTrading | BotType::DcaBot => true,
            _ => false,
        }
    }
    
    pub fn get_default_params(&self) -> HashMap<String, String> {
        let mut params = HashMap::new();
        match self {
            BotType::GridTrading => {
                params.insert("grid_levels".to_string(), "10".to_string());
                params.insert("grid_spacing_pct".to_string(), "1.0".to_string());
                params.insert("order_size_usd".to_string(), "100".to_string());
            },
            BotType::DcaBot => {
                params.insert("buy_interval_hours".to_string(), "24".to_string());
                params.insert("buy_amount_usd".to_string(), "100".to_string());
                params.insert("max_positions".to_string(), "5".to_string());
            },
            BotType::MomentumBot => {
                params.insert("trend_period".to_string(), "20".to_string());
                params.insert("entry_threshold".to_string(), "0.02".to_string());
                params.insert("exit_threshold".to_string(), "-0.01".to_string());
            },
            BotType::MeanReversion => {
                params.insert("lookback_period".to_string(), "50".to_string());
                params.insert("std_dev_threshold".to_string(), "2.0".to_string());
                params.insert("mean_type".to_string(), "sma".to_string());
            },
            BotType::ScalpingBot => {
                params.insert("profit_target_pct".to_string(), "0.1".to_string());
                params.insert("stop_loss_pct".to_string(), "0.05".to_string());
                params.insert("max_spread_pct".to_string(), "0.2".to_string());
            },
            BotType::AiTradingBot => {
                params.insert("model_path".to_string(), "models/default.pt".to_string());
                params.insert("prediction_threshold".to_string(), "0.6".to_string());
                params.insert("training_data_days".to_string(), "90".to_string());
            },
            BotType::SignalBot => {
                params.insert("signal_source".to_string(), "custom".to_string());
                params.insert("signal_endpoint".to_string(), "".to_string());
                params.insert("signal_interval_sec".to_string(), "60".to_string());
            },
            BotType::CustomBot => {
                params.insert("strategy_code".to_string(), "".to_string());
                params.insert("execution_mode".to_string(), "paper".to_string());
            },
            _ => {}
        }
        params
    }
}

// ============================================================================
// Bot Instance with All Features
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BotConfig {
    pub id: String,
    pub name: String,
    pub bot_type: BotType,
    pub description: String,
    pub version: String,
    pub enabled: bool,
    pub is_running: bool,
    pub fee_config: FeeConfig,
    
    // DEX Connections
    pub connected_dexes: Vec<String>,
    pub preferred_dex: String,
    
    // CEX Connections
    pub connected_cexes: Vec<String>,
    
    // Trading Pairs
    pub trading_pairs: Vec<String>,
    
    // Risk Management
    pub max_position_usd: f64,
    pub max_daily_loss_usd: f64,
    pub stop_loss_pct: f64,
    pub take_profit_pct: f64,
    
    // Performance
    pub latency_target_us: u64,
    pub min_profit_bps: u32,
    
    // Strategy Parameters (specific to each type)
    pub strategy_params: HashMap<String, String>,
}

impl BotConfig {
    pub fn new(id: String, name: String, bot_type: BotType) -> Self {
        let fee_config = FeeConfig::new(bot_type);
        
        Self {
            id,
            name,
            bot_type,
            description: bot_type.description().to_string(),
            version: "1.0.0".to_string(),
            enabled: bot_type.is_enabled_by_default(),
            is_running: false,
            fee_config,
            connected_dexes: Vec::new(),
            preferred_dex: String::new(),
            connected_cexes: Vec::new(),
            trading_pairs: Vec::new(),
            max_position_usd: 100000.0,
            max_daily_loss_usd: 5000.0,
            stop_loss_pct: 5.0,
            take_profit_pct: 10.0,
            latency_target_us: 5000,
            min_profit_bps: 10,
            strategy_params: HashMap::new(),
        }
    }
    
    pub fn with_dexes(mut self, dexes: Vec<String>) -> Self {
        self.connected_dexes = dexes.clone();
        if !dexes.is_empty() {
            self.preferred_dex = dexes[0].clone();
        }
        self
    }
    
    pub fn with_cexes(mut self, cexes: Vec<String>) -> Self {
        self.connected_cexes = cexes;
        self
    }
    
    pub fn with_pairs(mut self, pairs: Vec<String>) -> Self {
        self.trading_pairs = pairs;
        self
    }
    
    pub fn with_risk(mut self, max_pos: f64, max_loss: f64, stop_loss: f64, take_profit: f64) -> Self {
        self.max_position_usd = max_pos;
        self.max_daily_loss_usd = max_loss;
        self.stop_loss_pct = stop_loss;
        self.take_profit_pct = take_profit;
        self
    }
    
    pub fn with_latency_target(mut self, target_us: u64) -> Self {
        self.latency_target_us = target_us;
        self
    }
    
    pub fn set_strategy_param(&mut self, key: String, value: String) {
        self.strategy_params.insert(key, value);
    }
}

// ============================================================================
// Fee Configuration
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeeConfig {
    pub monthly_fee_usdt: f64,
    pub per_exchange_fee_usdt: f64,
    pub per_trade_fee_bps: f64,
    pub description: String,
}

impl FeeConfig {
    pub fn new(bot_type: BotType) -> Self {
        match bot_type {
            BotType::MarketMaker => FeeConfig {
                monthly_fee_usdt: 5000.0,
                per_exchange_fee_usdt: 1000.0,
                per_trade_fee_bps: 0.5,
                description: "Market Maker Bot - $5000/month + $1000 per exchange".to_string(),
            },
            BotType::Arbitrage => FeeConfig {
                monthly_fee_usdt: 3000.0,
                per_exchange_fee_usdt: 750.0,
                per_trade_fee_bps: 0.3,
                description: "Arbitrage Bot - $3000/month + $750 per exchange".to_string(),
            },
            BotType::Sniper => FeeConfig {
                monthly_fee_usdt: 2500.0,
                per_exchange_fee_usdt: 500.0,
                per_trade_fee_bps: 0.4,
                description: "Sniper Bot - $2500/month + $500 per exchange".to_string(),
            },
            _ => FeeConfig {
                monthly_fee_usdt: 2500.0,
                per_exchange_fee_usdt: 500.0,
                per_trade_fee_bps: 0.5,
                description: "Standard Bot - $2500/month + $500 per exchange".to_string(),
            }
        }
    }
    
    pub fn total_fee(&self, num_dexes: u32, num_cexes: u32) -> f64 {
        self.monthly_fee_usdt + 
            (self.per_exchange_fee_usdt * (num_dexes + num_cexes) as f64)
    }
}

// ============================================================================
// Bot Stats
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BotStats {
    pub total_pnl: f64,
    pub daily_pnl: f64,
    pub total_volume: f64,
    pub filled_orders: u64,
    pub open_orders: u32,
    pub avg_execution_latency_us: u64,
    pub total_saved_fees: f64,
    pub success_rate: f64,
    pub sharpe_ratio: f64,
    pub max_drawdown: f64,
}

impl BotStats {
    pub fn new() -> Self {
        Self {
            total_pnl: 0.0,
            daily_pnl: 0.0,
            total_volume: 0.0,
            filled_orders: 0,
            open_orders: 0,
            avg_execution_latency_us: 0,
            total_saved_fees: 0.0,
            success_rate: 0.0,
            sharpe_ratio: 0.0,
            max_drawdown: 0.0,
        }
    }
}

// ============================================================================
// Bot Instance - Runtime State
// ============================================================================

#[derive(Debug, Clone)]
pub struct BotInstance {
    pub config: BotConfig,
    pub stats: BotStats,
    pub orders: HashMap<String, Order>,
    pub positions: HashMap<String, Position>,
    pub started_at: i64,
    pub last_trade_at: i64,
}

impl BotInstance {
    pub fn new(config: BotConfig) -> Self {
        Self {
            config,
            stats: BotStats::new(),
            orders: HashMap::new(),
            positions: HashMap::new(),
            started_at: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_millis() as i64,
            last_trade_at: 0,
        }
    }
    
    pub fn start(&mut self) {
        self.config.is_running = true;
        println!("[✓] {} bot started", self.config.name);
    }
    
    pub fn stop(&mut self) {
        self.config.is_running = false;
        println!("[✗] {} bot stopped", self.config.name);
    }
    
    pub fn update_stats(&mut self, pnl_delta: f64) {
        self.stats.total_pnl += pnl_delta;
        self.stats.daily_pnl += pnl_delta;
    }
    
    pub fn get_uptime_seconds(&self) -> i64 {
        let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_millis() as i64;
        (now - self.started_at) / 1000
    }
}

// ============================================================================
// Order & Position
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Order {
    pub id: String,
    pub side: String,
    pub pair: String,
    pub price: f64,
    pub size: f64,
    pub status: String,
    pub exchange: String,
    pub created_at: i64,
    pub execution_latency_us: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Position {
    pub pair: String,
    pub size: f64,
    pub entry_price: f64,
    pub current_price: f64,
    pub unrealized_pnl: f64,
    pub realized_pnl: f64,
    pub opened_at: i64,
}

// ============================================================================
// Bot Manager - Creates and Manages All Bot Types
// ============================================================================

pub struct BotManager {
    pub bots: HashMap<String, Arc<RwLock<BotInstance>>>,
    pub next_id: u32,
}

impl BotManager {
    pub fn new() -> Self {
        Self {
            bots: HashMap::new(),
            next_id: 1,
        }
    }
    
    pub fn create_bot(&mut self, name: String, bot_type: BotType) -> Arc<RwLock<BotInstance>> {
        let id = format!("bot_{:03}", self.next_id);
        self.next_id += 1;
        
        let config = BotConfig::new(id.clone(), name, bot_type);
        let instance = BotInstance::new(config);
        let arc = Arc::new(RwLock::new(instance));
        
        self.bots.insert(id, arc.clone());
        arc
    }
    
    pub fn create_market_maker(&mut self, name: String) -> Arc<RwLock<BotInstance>> {
        self.create_bot(name, BotType::MarketMaker)
    }
    
    pub fn create_arbitrage_bot(&mut self, name: String) -> Arc<RwLock<BotInstance>> {
        self.create_bot(name, BotType::Arbitrage)
    }
    
    pub fn create_sniper_bot(&mut self, name: String) -> Arc<RwLock<BotInstance>> {
        let bot = self.create_bot(name, BotType::Sniper);
        if let Ok(mut b) = bot.write() {
            b.config.latency_target_us = 500; // 500μs target for sniper
        }
        bot
    }
    
    pub fn create_liquidity_bot(&mut self, name: String) -> Arc<RwLock<BotInstance>> {
        self.create_bot(name, BotType::Liquidity)
    }
    
    pub fn create_mev_bot(&mut self, name: String) -> Arc<RwLock<BotInstance>> {
        self.create_bot(name, BotType::MevBot)
    }
    
    pub fn create_sandwich_bot(&mut self, name: String) -> Arc<RwLock<BotInstance>> {
        self.create_bot(name, BotType::Sandwich)
    }
    
    pub fn create_flash_loan_bot(&mut self, name: String) -> Arc<RwLock<BotInstance>> {
        self.create_bot(name, BotType::FlashLoan)
    }
    
    pub fn create_cross_chain_bot(&mut self, name: String) -> Arc<RwLock<BotInstance>> {
        self.create_bot(name, BotType::CrossChain)
    }
    
    pub fn create_perp_hedge_bot(&mut self, name: String) -> Arc<RwLock<BotInstance>> {
        self.create_bot(name, BotType::PerpHedge)
    }
    
    pub fn get_bot(&self, id: &str) -> Option<Arc<RwLock<BotInstance>>> {
        self.bots.get(id).cloned()
    }
    
    pub fn start_all(&mut self) {
        println!("\n[~] Starting all bots...");
        for (_, bot) in &self.bots {
            if let Ok(mut b) = bot.write() {
                b.start();
            }
        }
    }
    
    pub fn stop_all(&mut self) {
        println!("[~] Stopping all bots...");
        for (_, bot) in &self.bots {
            if let Ok(mut b) = bot.write() {
                b.stop();
            }
        }
    }
    
    pub fn list_bots(&self) -> Vec<BotSummary> {
        self.bots.iter().map(|(id, bot)| {
            if let Ok(b) = bot.read() {
                BotSummary {
                    id: id.clone(),
                    name: b.config.name.clone(),
                    bot_type: b.config.bot_type.as_str().to_string(),
                    enabled: b.config.enabled,
                    is_running: b.config.is_running,
                    pnl: b.stats.total_pnl,
                    volume: b.stats.total_volume,
                    orders: b.stats.filled_orders,
                    uptime_seconds: b.get_uptime_seconds(),
                }
            } else {
                BotSummary {
                    id: id.clone(),
                    name: "Unknown".to_string(),
                    bot_type: "Unknown".to_string(),
                    enabled: false,
                    is_running: false,
                    pnl: 0.0,
                    volume: 0.0,
                    orders: 0,
                    uptime_seconds: 0,
                }
            }
        }).collect()
    }
    
    pub fn get_total_stats(&self) -> BotStats {
        let mut total = BotStats::new();
        
        for (_, bot) in &self.bots {
            if let Ok(b) = bot.read() {
                total.total_pnl += b.stats.total_pnl;
                total.daily_pnl += b.stats.daily_pnl;
                total.total_volume += b.stats.total_volume;
                total.filled_orders += b.stats.filled_orders;
                total.open_orders += b.stats.open_orders;
            }
        }
        
        total
    }
}

#[derive(Debug, Clone)]
pub struct BotSummary {
    pub id: String,
    pub name: String,
    pub bot_type: String,
    pub enabled: bool,
    pub is_running: bool,
    pub pnl: f64,
    pub volume: f64,
    pub orders: u64,
    pub uptime_seconds: i64,
}