//! Tests for the strategy marketplace.

use super::*;

fn author() -> StrategyAuthor {
    StrategyAuthor {
        id: "author-1".to_string(),
        name: "Test Author".to_string(),
        email: "author@example.com".to_string(),
        verified: true,
        rating: 4.5,
        total_sales: 10,
        strategies_published: 3,
    }
}

fn listing(id: &str, price: f64, status: StrategyStatus) -> StrategyListing {
    StrategyListing {
        id: id.to_string(),
        name: format!("Strategy {}", id),
        description: "A test strategy".to_string(),
        author: author(),
        version: "1.0.0".to_string(),
        strategy_type: StrategyType::Arbitrage,
        risk_level: RiskLevel::Medium,
        default_timeframe: Timeframe::Minutes(5),
        pricing: StrategyPricing {
            price_usd: price,
            price_tiger: 0,
            subscription_enabled: false,
            subscription_price_monthly_usd: None,
            subscription_price_monthly_tiger: None,
            royalty_percentage: 0.0,
            refund_period_days: 0,
        },
        performance: PerformanceMetrics {
            total_trades: 0,
            winning_trades: 0,
            losing_trades: 0,
            win_rate: 0.0,
            total_profit: 0.0,
            total_loss: 0.0,
            net_profit: 0.0,
            profit_factor: 0.0,
            average_profit: 0.0,
            average_loss: 0.0,
            largest_profit: 0.0,
            largest_loss: 0.0,
            average_holding_time_seconds: 0,
            sharpe_ratio: 0.0,
            max_drawdown: 0.0,
            average_drawdown: 0.0,
        },
        supported_exchanges: Vec::new(),
        supported_pairs: Vec::new(),
        parameters: Vec::new(),
        requires_hardware_wallet: false,
        requires_api_keys: false,
        min_capital_usd: 0.0,
        max_capital_usd: None,
        is_verified: false,
        status,
        rating: 0.0,
        review_count: 0,
        downloads: 0,
        is_featured: false,
        category_tags: vec!["arbitrage".to_string()],
        created_at: 0,
        updated_at: 0,
    }
}

fn review(strategy_id: &str, user_id: &str, rating: u8) -> StrategyReview {
    StrategyReview {
        id: format!("review-{}-{}", strategy_id, user_id),
        strategy_id: strategy_id.to_string(),
        user_id: user_id.to_string(),
        rating,
        title: "Good".to_string(),
        content: "fine".to_string(),
        pros: Vec::new(),
        cons: Vec::new(),
        created_at: 0,
        helpful_count: 0,
    }
}

// ============================================================================
// Listings
// ============================================================================

#[test]
fn create_listing_validates_its_input() {
    let m = StrategyMarketplace::new();

    let mut l = listing("a", 10.0, StrategyStatus::Active);
    l.id = "  ".to_string();
    assert_eq!(m.create_listing(l).unwrap_err(), MarketplaceError::InvalidOperation);

    let mut l = listing("a", 10.0, StrategyStatus::Active);
    l.pricing.price_usd = -1.0;
    assert_eq!(m.create_listing(l).unwrap_err(), MarketplaceError::InvalidOperation);

    let mut l = listing("a", 10.0, StrategyStatus::Active);
    l.pricing.price_usd = f64::NAN;
    assert_eq!(m.create_listing(l).unwrap_err(), MarketplaceError::InvalidOperation);

    m.create_listing(listing("a", 10.0, StrategyStatus::Active)).unwrap();
    assert_eq!(
        m.create_listing(listing("a", 10.0, StrategyStatus::Active)).unwrap_err(),
        MarketplaceError::AlreadyExists
    );
}

#[test]
fn get_listings_returns_only_active_listings() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("active", 10.0, StrategyStatus::Active)).unwrap();
    m.create_listing(listing("draft", 10.0, StrategyStatus::Draft)).unwrap();

    let found = m.get_listings(StrategyFilters::default()).unwrap();
    assert_eq!(found.len(), 1);
    assert_eq!(found[0].id, "active");
}

#[test]
fn get_listings_applies_filters() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("cheap", 5.0, StrategyStatus::Active)).unwrap();
    m.create_listing(listing("pricey", 500.0, StrategyStatus::Active)).unwrap();

    let mut f = StrategyFilters::default();
    f.min_price = Some(100.0);
    let found = m.get_listings(f).unwrap();
    assert_eq!(found.len(), 1);
    assert_eq!(found[0].id, "pricey");

    let mut f = StrategyFilters::default();
    f.max_price = Some(100.0);
    let found = m.get_listings(f).unwrap();
    assert_eq!(found.len(), 1);
    assert_eq!(found[0].id, "cheap");

    let mut f = StrategyFilters::default();
    f.strategy_type = Some(StrategyType::MarketMaker);
    assert!(m.get_listings(f).unwrap().is_empty());

    let mut f = StrategyFilters::default();
    f.risk_level = Some(RiskLevel::Medium);
    assert_eq!(m.get_listings(f).unwrap().len(), 2);
}

#[test]
fn get_listings_sorts_by_price() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("b", 20.0, StrategyStatus::Active)).unwrap();
    m.create_listing(listing("a", 10.0, StrategyStatus::Active)).unwrap();

    let mut f = StrategyFilters::default();
    f.sort_by = Some("price_low".to_string());
    let found = m.get_listings(f).unwrap();
    assert_eq!(found[0].id, "a");

    let mut f = StrategyFilters::default();
    f.sort_by = Some("price_high".to_string());
    let found = m.get_listings(f).unwrap();
    assert_eq!(found[0].id, "b");
}

#[test]
fn update_listing_changes_only_supplied_fields() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("a", 10.0, StrategyStatus::Active)).unwrap();

    let updated = m
        .update_listing(
            "a",
            StrategyListingUpdate {
                name: Some("Renamed".to_string()),
                description: None,
                price_usd: Some(42.0),
                status: None,
            },
        )
        .unwrap();

    assert_eq!(updated.name, "Renamed");
    assert_eq!(updated.pricing.price_usd, 42.0);
    assert_eq!(updated.description, "A test strategy");
    assert_eq!(updated.status, StrategyStatus::Active);

    assert_eq!(
        m.update_listing("missing", StrategyListingUpdate::default()).unwrap_err(),
        MarketplaceError::NotFound
    );
}

// ============================================================================
// Reviews
// ============================================================================

#[test]
fn add_review_updates_the_listing_rating() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("a", 10.0, StrategyStatus::Active)).unwrap();

    m.add_review(review("a", "u1", 5)).unwrap();
    m.add_review(review("a", "u2", 3)).unwrap();

    let l = m.get_listing("a").unwrap();
    assert_eq!(l.review_count, 2);
    assert!((l.rating - 4.0).abs() < 1e-9);

    assert_eq!(m.get_reviews("a").unwrap().len(), 2);
}

#[test]
fn add_review_rejects_a_missing_strategy() {
    let m = StrategyMarketplace::new();
    // The old implementation stored this and returned Ok, updating nothing.
    assert_eq!(
        m.add_review(review("missing", "u1", 5)).unwrap_err(),
        MarketplaceError::NotFound
    );
}

#[test]
fn add_review_rejects_an_out_of_range_rating() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("a", 10.0, StrategyStatus::Active)).unwrap();

    assert_eq!(
        m.add_review(review("a", "u1", 0)).unwrap_err(),
        MarketplaceError::InvalidOperation
    );
    assert_eq!(
        m.add_review(review("a", "u1", 6)).unwrap_err(),
        MarketplaceError::InvalidOperation
    );
}

#[test]
fn add_review_rejects_a_second_review_from_the_same_user() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("a", 10.0, StrategyStatus::Active)).unwrap();

    m.add_review(review("a", "u1", 5)).unwrap();
    assert_eq!(
        m.add_review(review("a", "u1", 1)).unwrap_err(),
        MarketplaceError::AlreadyExists
    );

    // The rating reflects the single accepted review.
    assert!((m.get_listing("a").unwrap().rating - 5.0).abs() < 1e-9);
}

// ============================================================================
// Orders
// ============================================================================

#[test]
fn free_strategy_is_granted_immediately() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("free", 0.0, StrategyStatus::Active)).unwrap();

    let order = m.create_order("u1", "free", OrderType::Purchase).unwrap();
    assert_eq!(order.status, OrderStatus::Completed);
    assert!(order.completed_at.is_some());

    let owned = m.get_user_strategies("u1").unwrap();
    assert_eq!(owned.len(), 1);
    assert_eq!(owned[0].id, "free");
}

#[test]
fn paid_strategy_stays_pending_until_settled() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("paid", 49.0, StrategyStatus::Active)).unwrap();

    let order = m.create_order("u1", "paid", OrderType::Purchase).unwrap();
    assert_eq!(order.status, OrderStatus::Pending);
    assert_eq!(order.amount, 49.0, "the amount comes from the listing");
    assert!(order.completed_at.is_none());

    // Not granted before settlement. The old implementation had no purchase
    // path at all, so this list was always empty.
    assert!(m.get_user_strategies("u1").unwrap().is_empty());

    let settled = m.complete_order(&order.id, "stripe:pi_123").unwrap();
    assert_eq!(settled.status, OrderStatus::Completed);

    let owned = m.get_user_strategies("u1").unwrap();
    assert_eq!(owned.len(), 1);
    assert_eq!(owned[0].id, "paid");
}

#[test]
fn complete_order_requires_a_settlement_reference() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("paid", 49.0, StrategyStatus::Active)).unwrap();
    let order = m.create_order("u1", "paid", OrderType::Purchase).unwrap();

    assert_eq!(
        m.complete_order(&order.id, "   ").unwrap_err(),
        MarketplaceError::PaymentFailed
    );
    // Nothing was granted.
    assert!(m.get_user_strategies("u1").unwrap().is_empty());
}

#[test]
fn complete_order_cannot_be_applied_twice() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("paid", 49.0, StrategyStatus::Active)).unwrap();
    let order = m.create_order("u1", "paid", OrderType::Purchase).unwrap();

    m.complete_order(&order.id, "stripe:pi_123").unwrap();
    assert_eq!(
        m.complete_order(&order.id, "stripe:pi_123").unwrap_err(),
        MarketplaceError::AlreadyExists
    );
}

#[test]
fn create_order_rejects_a_repeat_request() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("paid", 49.0, StrategyStatus::Active)).unwrap();

    m.create_order("u1", "paid", OrderType::Purchase).unwrap();
    assert_eq!(
        m.create_order("u1", "paid", OrderType::Purchase).unwrap_err(),
        MarketplaceError::AlreadyExists
    );

    // A different buyer is unaffected.
    m.create_order("u2", "paid", OrderType::Purchase).unwrap();
}

#[test]
fn create_order_validates_its_inputs() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("paid", 49.0, StrategyStatus::Active)).unwrap();

    assert_eq!(
        m.create_order("  ", "paid", OrderType::Purchase).unwrap_err(),
        MarketplaceError::Unauthorized
    );
    assert_eq!(
        m.create_order("u1", "missing", OrderType::Purchase).unwrap_err(),
        MarketplaceError::NotFound
    );

    // A draft strategy cannot be bought.
    m.create_listing(listing("draft", 10.0, StrategyStatus::Draft)).unwrap();
    assert_eq!(
        m.create_order("u1", "draft", OrderType::Purchase).unwrap_err(),
        MarketplaceError::InvalidOperation
    );

    // A subscription needs a monthly price.
    assert_eq!(
        m.create_order("u1", "paid", OrderType::Subscription).unwrap_err(),
        MarketplaceError::InvalidOperation
    );
}

#[test]
fn subscription_uses_the_monthly_price() {
    let m = StrategyMarketplace::new();
    let mut l = listing("sub", 100.0, StrategyStatus::Active);
    l.pricing.subscription_enabled = true;
    l.pricing.subscription_price_monthly_usd = Some(9.99);
    m.create_listing(l).unwrap();

    let order = m.create_order("u1", "sub", OrderType::Subscription).unwrap();
    assert_eq!(order.amount, 9.99);
    assert_eq!(order.order_type, OrderType::Subscription);
}

#[test]
fn orders_can_be_retrieved_by_id() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("paid", 49.0, StrategyStatus::Active)).unwrap();
    let order = m.create_order("u1", "paid", OrderType::Purchase).unwrap();

    let fetched = m.get_order(&order.id).unwrap();
    assert_eq!(fetched.id, order.id);
    assert_eq!(fetched.amount, order.amount);

    assert_eq!(m.get_order("missing").unwrap_err(), MarketplaceError::NotFound);
}

// ============================================================================
// Discovery
// ============================================================================

#[test]
fn get_featured_returns_only_active_featured_listings() {
    let m = StrategyMarketplace::new();

    let mut featured = listing("featured", 10.0, StrategyStatus::Active);
    featured.is_featured = true;
    featured.rating = 4.0;
    m.create_listing(featured).unwrap();

    let mut also = listing("also", 10.0, StrategyStatus::Active);
    also.is_featured = true;
    also.rating = 5.0;
    m.create_listing(also).unwrap();

    let mut draft = listing("draft", 10.0, StrategyStatus::Draft);
    draft.is_featured = true;
    m.create_listing(draft).unwrap();

    m.create_listing(listing("plain", 10.0, StrategyStatus::Active)).unwrap();

    let found = m.get_featured(10).unwrap();
    assert_eq!(found.len(), 2);
    assert_eq!(found[0].id, "also", "ordered by rating, highest first");

    assert_eq!(m.get_featured(1).unwrap().len(), 1);
}

#[test]
fn search_matches_name_description_and_tags() {
    let m = StrategyMarketplace::new();
    m.create_listing(listing("a", 10.0, StrategyStatus::Active)).unwrap();

    assert_eq!(m.search("strategy a").unwrap().len(), 1);
    assert_eq!(m.search("A TEST").unwrap().len(), 1);
    assert_eq!(m.search("arbitrage").unwrap().len(), 1);
    assert!(m.search("nonexistent").unwrap().is_empty());
}
