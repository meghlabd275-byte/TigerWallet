//!
//! TigerWallet Strategy Marketplace
//! High-performance strategy marketplace for trading bots
//!

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::RwLock;

/// Strategy type enumeration
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum StrategyType {
    MarketMaker,
    Arbitrage,
    Sniper,
    Liquidity,
    MevBot,
    Sandwich,
    FlashLoan,
    CrossChain,
    PerpetualHedge,
    FrontRun,
    GridTrading,
    DcaBot,
    MomentumBot,
    MeanReversion,
    ScalpingBot,
    AiTrading,
    SignalBot,
    Custom,
}

/// Risk level enumeration
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum RiskLevel {
    VeryLow,
    Low,
    Medium,
    High,
    VeryHigh,
}

/// Timeframe for strategy execution
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum Timeframe {
    Seconds(u32),
    Minutes(u32),
    Hours(u32),
    Days(u32),
}

/// Strategy author information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StrategyAuthor {
    pub id: String,
    pub name: String,
    pub email: String,
    pub verified: bool,
    pub rating: f64,
    pub total_sales: u64,
    pub strategies_published: u32,
}

/// Strategy pricing
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StrategyPricing {
    pub price_usd: f64,
    pub price_tiger: u64,
    pub subscription_enabled: bool,
    pub subscription_price_monthly_usd: Option<f64>,
    pub subscription_price_monthly_tiger: Option<u64>,
    pub royalty_percentage: f64,
    pub refund_period_days: u32,
}

/// Performance metrics for a strategy
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceMetrics {
    pub total_trades: u64,
    pub winning_trades: u64,
    pub losing_trades: u64,
    pub win_rate: f64,
    pub total_profit: f64,
    pub total_loss: f64,
    pub net_profit: f64,
    pub profit_factor: f64,
    pub average_profit: f64,
    pub average_loss: f64,
    pub largest_profit: f64,
    pub largest_loss: f64,
    pub average_holding_time_seconds: u64,
    pub sharpe_ratio: f64,
    pub max_drawdown: f64,
    pub average_drawdown: f64,
}

/// Supported exchange for strategy
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SupportedExchange {
    pub name: String,
    pub enabled: bool,
    pub testnet_enabled: bool,
}

/// Supported trading pair
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SupportedPair {
    pub base: String,
    pub quote: String,
    pub chain: Option<String>,
}

/// Complete strategy listing
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StrategyListing {
    pub id: String,
    pub name: String,
    pub description: String,
    pub version: String,
    pub author: StrategyAuthor,
    pub strategy_type: StrategyType,
    pub pricing: StrategyPricing,
    pub category_tags: Vec<String>,
    pub risk_level: RiskLevel,
    pub supported_exchanges: Vec<SupportedExchange>,
    pub supported_pairs: Vec<SupportedPair>,
    pub default_timeframe: Timeframe,
    pub performance: PerformanceMetrics,
    pub parameters: Vec<StrategyParameter>,
    pub requires_hardware_wallet: bool,
    pub requires_api_keys: bool,
    pub min_capital_usd: f64,
    pub max_capital_usd: Option<f64>,
    pub created_at: i64,
    pub updated_at: i64,
    pub downloads: u64,
    pub rating: f64,
    pub review_count: u32,
    pub is_featured: bool,
    pub is_verified: bool,
    pub status: StrategyStatus,
}

/// Strategy parameter definition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StrategyParameter {
    pub name: String,
    pub display_name: String,
    pub param_type: ParameterType,
    pub default_value: serde_json::Value,
    pub min_value: Option<f64>,
    pub max_value: Option<f64>,
    pub step: Option<f64>,
    pub description: String,
    pub required: bool,
}

/// Parameter types
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum ParameterType {
    Integer,
    Float,
    Boolean,
    String,
    Enum(Vec<String>),
    Percentage,
    Currency,
}

/// Strategy status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum StrategyStatus {
    Draft,
    PendingReview,
    Active,
    Suspended,
    Deprecated,
}

/// Review for a strategy
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StrategyReview {
    pub id: String,
    pub strategy_id: String,
    pub user_id: String,
    pub rating: u8,
    pub title: String,
    pub content: String,
    pub pros: Vec<String>,
    pub cons: Vec<String>,
    pub created_at: i64,
    pub helpful_count: u32,
}

/// Strategy order
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StrategyOrder {
    pub id: String,
    pub user_id: String,
    pub strategy_id: String,
    pub order_type: OrderType,
    pub amount: f64,
    pub currency: String,
    pub status: OrderStatus,
    pub created_at: i64,
    pub completed_at: Option<i64>,
}

/// Order types
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum OrderType {
    Purchase,
    Subscription,
}

/// Order status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum OrderStatus {
    Pending,
    Processing,
    Completed,
    Failed,
    Refunded,
}

/// Strategy marketplace manager
pub struct StrategyMarketplace {
    listings: RwLock<HashMap<String, StrategyListing>>,
    reviews: RwLock<HashMap<String, Vec<StrategyReview>>>,
    orders: RwLock<HashMap<String, StrategyOrder>>,
    user_purchases: RwLock<HashMap<String, Vec<String>>>,
}

impl StrategyMarketplace {
    pub fn new() -> Self {
        Self {
            listings: RwLock::new(HashMap::new()),
            reviews: RwLock::new(HashMap::new()),
            orders: RwLock::new(HashMap::new()),
            user_purchases: RwLock::new(HashMap::new()),
        }
    }

    /// Create a new strategy listing
    pub fn create_listing(&self, listing: StrategyListing) -> Result<StrategyListing, MarketplaceError> {
        let mut listings = self.listings.write().map_err(|_| MarketplaceError::LockError)?;
        
        if listings.contains_key(&listing.id) {
            return Err(MarketplaceError::AlreadyExists);
        }
        
        listings.insert(listing.id.clone(), listing.clone());
        Ok(listing)
    }

    /// Get a strategy by ID
    pub fn get_listing(&self, id: &str) -> Result<StrategyListing, MarketplaceError> {
        let listings = self.listings.read().map_err(|_| MarketplaceError::LockError)?;
        listings.get(id).cloned().ok_or(MarketplaceError::NotFound)
    }

    /// Get all active listings with filters
    pub fn get_listings(&self, filters: StrategyFilters) -> Result<Vec<StrategyListing>, MarketplaceError> {
        let listings = self.listings.read().map_err(|_| MarketplaceError::LockError)?;
        
        let mut result: Vec<StrategyListing> = listings
            .values()
            .filter(|l| {
                if filters.strategy_type.is_some() && l.strategy_type != filters.strategy_type.as_ref().unwrap() {
                    return false;
                }
                if filters.risk_level.is_some() && l.risk_level != filters.risk_level.as_ref().unwrap() {
                    return false;
                }
                if let Some(min_price) = filters.min_price {
                    if l.pricing.price_usd < min_price {
                        return false;
                    }
                }
                if let Some(max_price) = filters.max_price {
                    if l.pricing.price_usd > max_price {
                        return false;
                    }
                }
                if let Some(ref exchange) = filters.exchange {
                    if !l.supported_exchanges.iter().any(|e| &e.name == exchange && e.enabled) {
                        return false;
                    }
                }
                if let Some(min_rating) = filters.min_rating {
                    if l.rating < min_rating {
                        return false;
                    }
                }
                if l.status != StrategyStatus::Active {
                    return false;
                }
                true
            })
            .cloned()
            .collect();

        // Sort by specified order
        match filters.sort_by.as_deref().unwrap_or("rating") {
            "price_low" => result.sort_by(|a, b| a.pricing.price_usd.partial_cmp(&b.pricing.price_usd).unwrap()),
            "price_high" => result.sort_by(|a, b| b.pricing.price_usd.partial_cmp(&a.pricing.price_usd).unwrap()),
            "rating" => result.sort_by(|a, b| b.rating.partial_cmp(&a.rating).unwrap()),
            "newest" => result.sort_by(|a, b| b.created_at.cmp(&a.created_at)),
            "popular" => result.sort_by(|a, b| b.downloads.cmp(&a.downloads)),
            _ => {}
        }

        Ok(result)
    }

    /// Update a strategy listing
    pub fn update_listing(&self, id: &str, updates: StrategyListingUpdate) -> Result<StrategyListing, MarketplaceError> {
        let mut listings = self.listings.write().map_err(|_| MarketplaceError::LockError)?;
        
        let listing = listings.get_mut(id).ok_or(MarketplaceError::NotFound)?;
        
        if let Some(name) = updates.name {
            listing.name = name;
        }
        if let Some(description) = updates.description {
            listing.description = description;
        }
        if let Some(price_usd) = updates.price_usd {
            listing.pricing.price_usd = price_usd;
        }
        if let Some(status) = updates.status {
            listing.status = status;
        }
        
        listing.updated_at = chrono_timestamp();
        
        Ok(listing.clone())
    }

    /// Add a review to a strategy
    pub fn add_review(&self, review: StrategyReview) -> Result<StrategyReview, MarketplaceError> {
        let mut reviews = self.reviews.write().map_err(|_| MarketplaceError::LockError)?;
        
        let reviews_list = reviews.entry(review.strategy_id.clone()).or_insert_with(Vec::new);
        reviews_list.push(review.clone());
        
        // Update strategy rating
        let mut listings = self.listings.write().map_err(|_| MarketplaceError::LockError)?;
        if let Some(listing) = listings.get_mut(&review.strategy_id) {
            let total_rating: f64 = reviews_list.iter().map(|r| r.rating as f64).sum();
            listing.rating = total_rating / reviews_list.len() as f64;
            listing.review_count = reviews_list.len() as u32;
        }
        
        Ok(review)
    }

    /// Get user's purchased strategies
    pub fn get_user_strategies(&self, user_id: &str) -> Result<Vec<StrategyListing>, MarketplaceError> {
        let purchases = self.user_purchases.read().map_err(|_| MarketplaceError::LockError)?;
        let listings = self.listings.read().map_err(|_| MarketplaceError::LockError)?;
        
        let strategy_ids = purchases.get(user_id).cloned().unwrap_or_default();
        
        let mut user_strategies = Vec::new();
        for id in strategy_ids {
            if let Some(listing) = listings.get(&id) {
                user_strategies.push(listing.clone());
            }
        }
        
        Ok(user_strategies)
    }

    /// Get featured strategies
    pub fn get_featured(&self, limit: usize) -> Result<Vec<StrategyListing>, MarketplaceError> {
        let listings = self.listings.read().map_err(|_| MarketplaceError::LockError)?;
        
        let mut featured: Vec<StrategyListing> = listings
            .values()
            .filter(|l| l.is_featured && l.status == StrategyStatus::Active)
            .cloned()
            .collect();
        
        featured.sort_by(|a, b| b.rating.partial_cmp(&a.rating).unwrap());
        featured.truncate(limit);
        
        Ok(featured)
    }

    /// Search strategies by keyword
    pub fn search(&self, query: &str) -> Result<Vec<StrategyListing>, MarketplaceError> {
        let listings = self.listings.read().map_err(|_| MarketplaceError::LockError)?;
        let query_lower = query.to_lowercase();
        
        Ok(listings
            .values()
            .filter(|l| {
                l.name.to_lowercase().contains(&query_lower) ||
                l.description.to_lowercase().contains(&query_lower) ||
                l.category_tags.iter().any(|t| t.to_lowercase().contains(&query_lower))
            })
            .cloned()
            .collect())
    }
}

/// Filters for searching strategies
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct StrategyFilters {
    pub strategy_type: Option<StrategyType>,
    pub risk_level: Option<RiskLevel>,
    pub exchange: Option<String>,
    pub min_price: Option<f64>,
    pub max_price: Option<f64>,
    pub min_rating: Option<f64>,
    pub sort_by: Option<String>,
}

/// Updates to a strategy listing
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct StrategyListingUpdate {
    pub name: Option<String>,
    pub description: Option<String>,
    pub price_usd: Option<f64>,
    pub status: Option<StrategyStatus>,
}

/// Marketplace errors
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum MarketplaceError {
    NotFound,
    AlreadyExists,
    LockError,
    InvalidOperation,
    PaymentFailed,
    Unauthorized,
}

fn chrono_timestamp() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64
}
