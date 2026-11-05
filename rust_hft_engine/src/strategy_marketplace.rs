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

    /// Creates a strategy listing.
    pub fn create_listing(&self, listing: StrategyListing) -> Result<StrategyListing, MarketplaceError> {
        if listing.id.trim().is_empty() {
            return Err(MarketplaceError::InvalidOperation);
        }
        if !listing.pricing.price_usd.is_finite() || listing.pricing.price_usd < 0.0 {
            return Err(MarketplaceError::InvalidOperation);
        }
        if listing
            .pricing
            .subscription_price_monthly_usd
            .is_some_and(|p| !p.is_finite() || p < 0.0)
        {
            return Err(MarketplaceError::InvalidOperation);
        }

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
                if let Some(want) = &filters.strategy_type {
                    if &l.strategy_type != want {
                        return false;
                    }
                }
                if let Some(want) = &filters.risk_level {
                    if &l.risk_level != want {
                        return false;
                    }
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
            // total_cmp orders NaN deterministically instead of panicking.
            "price_low" => result.sort_by(|a, b| a.pricing.price_usd.total_cmp(&b.pricing.price_usd)),
            "price_high" => result.sort_by(|a, b| b.pricing.price_usd.total_cmp(&a.pricing.price_usd)),
            "rating" => result.sort_by(|a, b| b.rating.total_cmp(&a.rating)),
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

    /// Adds a review to a strategy.
    ///
    /// The review is rejected when the strategy does not exist or the rating
    /// is outside 1..=5. The previous version stored the review under any
    /// strategy id, updated nothing when the listing was missing, and still
    /// returned Ok, so a caller could not tell that the review had gone
    /// nowhere.
    pub fn add_review(&self, review: StrategyReview) -> Result<StrategyReview, MarketplaceError> {
        if !(1..=5).contains(&review.rating) {
            return Err(MarketplaceError::InvalidOperation);
        }

        let mut listings = self.listings.write().map_err(|_| MarketplaceError::LockError)?;
        if !listings.contains_key(&review.strategy_id) {
            return Err(MarketplaceError::NotFound);
        }

        let mut reviews = self.reviews.write().map_err(|_| MarketplaceError::LockError)?;
        let reviews_list = reviews
            .entry(review.strategy_id.clone())
            .or_insert_with(Vec::new);

        // One review per user per strategy.
        if reviews_list.iter().any(|r| r.user_id == review.user_id) {
            return Err(MarketplaceError::AlreadyExists);
        }
        reviews_list.push(review.clone());

        // Recompute the listing's rating from the reviews actually held, and
        // write it back so the two cannot drift apart.
        let total_rating: f64 = reviews_list.iter().map(|r| r.rating as f64).sum();
        let count = reviews_list.len() as u32;
        if let Some(listing) = listings.get_mut(&review.strategy_id) {
            listing.rating = total_rating / count as f64;
            listing.review_count = count;
        }

        Ok(review)
    }

    /// Returns the reviews for a strategy.
    pub fn get_reviews(&self, strategy_id: &str) -> Result<Vec<StrategyReview>, MarketplaceError> {
        let reviews = self.reviews.read().map_err(|_| MarketplaceError::LockError)?;
        Ok(reviews.get(strategy_id).cloned().unwrap_or_default())
    }

    /// Places a purchase or subscription order for a strategy.
    ///
    /// The amount is taken from the listing's own price, not from the caller.
    /// The previous version had no purchase path at all: `orders` and
    /// `user_purchases` were declared and never written, so
    /// `get_user_strategies` always returned an empty list and a strategy
    /// could never be bought.
    pub fn create_order(
        &self,
        user_id: &str,
        strategy_id: &str,
        order_type: OrderType,
    ) -> Result<StrategyOrder, MarketplaceError> {
        if user_id.trim().is_empty() {
            return Err(MarketplaceError::Unauthorized);
        }

        let (price, is_free) = {
            let listings = self.listings.read().map_err(|_| MarketplaceError::LockError)?;
            let listing = listings.get(strategy_id).ok_or(MarketplaceError::NotFound)?;

            if listing.status != StrategyStatus::Active {
                return Err(MarketplaceError::InvalidOperation);
            }

            let is_free = listing.pricing.price_usd <= 0.0
                && listing.pricing.subscription_price_monthly_usd.unwrap_or(0.0) <= 0.0;

            let price = match order_type {
                OrderType::Purchase => listing.pricing.price_usd,
                OrderType::Subscription => listing
                    .pricing
                    .subscription_price_monthly_usd
                    .ok_or(MarketplaceError::InvalidOperation)?,
            };

            if !price.is_finite() || price < 0.0 {
                return Err(MarketplaceError::InvalidOperation);
            }

            (price, is_free)
        };

        // The order id is derived from the listing and the buyer, so a
        // repeated request cannot be recorded twice.
        let order_id = format!("{}-{}", strategy_id, user_id);
        {
            let orders = self.orders.read().map_err(|_| MarketplaceError::LockError)?;
            if orders.contains_key(&order_id) {
                return Err(MarketplaceError::AlreadyExists);
            }
        }

        let now = chrono_timestamp();
        let order = StrategyOrder {
            id: order_id.clone(),
            user_id: user_id.to_string(),
            strategy_id: strategy_id.to_string(),
            order_type,
            amount: price,
            currency: "USD".to_string(),
            // A paid order stays Pending until a payment processor confirms
            // it. A free strategy is completed immediately.
            status: if is_free {
                OrderStatus::Completed
            } else {
                OrderStatus::Pending
            },
            created_at: now,
            completed_at: if is_free { Some(now) } else { None },
        };

        self.orders
            .write()
            .map_err(|_| MarketplaceError::LockError)?
            .insert(order_id, order.clone());

        if order.status == OrderStatus::Completed {
            self.grant_strategy(user_id, strategy_id)?;
        }

        Ok(order)
    }

    /// Marks a pending order as paid and grants the strategy.
    ///
    /// This is the only path that grants access to a paid strategy, and it
    /// takes a settlement reference so the grant can be traced.
    pub fn complete_order(
        &self,
        order_id: &str,
        settlement_reference: &str,
    ) -> Result<StrategyOrder, MarketplaceError> {
        if settlement_reference.trim().is_empty() {
            return Err(MarketplaceError::PaymentFailed);
        }

        let granted = {
            let mut orders = self.orders.write().map_err(|_| MarketplaceError::LockError)?;
            let order = orders.get_mut(order_id).ok_or(MarketplaceError::NotFound)?;

            match order.status {
                OrderStatus::Pending | OrderStatus::Processing => {}
                OrderStatus::Completed => return Err(MarketplaceError::AlreadyExists),
                _ => return Err(MarketplaceError::InvalidOperation),
            }

            order.status = OrderStatus::Completed;
            order.completed_at = Some(chrono_timestamp());
            order.clone()
        };

        self.grant_strategy(&granted.user_id, &granted.strategy_id)?;
        Ok(granted)
    }

    /// Returns an order by id.
    pub fn get_order(&self, order_id: &str) -> Result<StrategyOrder, MarketplaceError> {
        let orders = self.orders.read().map_err(|_| MarketplaceError::LockError)?;
        orders
            .get(order_id)
            .cloned()
            .ok_or(MarketplaceError::NotFound)
    }

    /// Records that `user_id` owns `strategy_id`.
    fn grant_strategy(&self, user_id: &str, strategy_id: &str) -> Result<(), MarketplaceError> {
        let mut purchases = self
            .user_purchases
            .write()
            .map_err(|_| MarketplaceError::LockError)?;
        let owned = purchases.entry(user_id.to_string()).or_default();
        if !owned.iter().any(|id| id == strategy_id) {
            owned.push(strategy_id.to_string());
        }
        Ok(())
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

        featured.sort_by(|a, b| b.rating.total_cmp(&a.rating));
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

/// Seconds since the Unix epoch, or 0 if the clock is before it.
fn chrono_timestamp() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}

#[cfg(test)]
mod tests;
