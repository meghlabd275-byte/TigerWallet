// Behavioural tests for the prediction market engine.
//
// These assert on observable outcomes: balances before and after settlement,
// position values and input validation. A regression in the payout path fails
// here rather than passing quietly.

use tiger_prediction_backend::{
    MarketStatus, OrderSide, OrderType, OutcomeType, PredictionError,
    PredictionMarketService,
};

const PRICE_60: u64 = 600_000;

fn market_with_two_outcomes(svc: &PredictionMarketService) -> u64 {
    svc.create_market(
        "Will it rain?".to_string(),
        "A test market".to_string(),
        OutcomeType::Binary,
        vec!["Yes".to_string(), "No".to_string()],
        0,
        "General".to_string(),
    )
    .expect("market creation must succeed")
    .market_id
}

/// Place a limit order and fill it, which is what establishes a position.
fn buy(svc: &PredictionMarketService, user: u32, market: u64, outcome: u32, amount: u64) {
    let order = svc
        .place_order(
            user,
            market,
            outcome,
            OrderType::Limit,
            OrderSide::Buy,
            PRICE_60,
            amount,
            None,
        )
        .expect("order placement must succeed");
    svc.fill_order(order.order_id, amount, PRICE_60)
        .expect("fill must succeed");
}

// A losing position settles to zero.
#[test]
fn losing_position_settles_to_zero() {
    let svc = PredictionMarketService::new();
    let market = market_with_two_outcomes(&svc);

    svc.add_funds(1, 1_000_000);
    buy(&svc, 1, market, 0, 100);

    svc.resolve_market(market, 1).expect("resolution must succeed");

    let positions = svc.get_user_positions(1);
    assert_eq!(positions.len(), 1);
    assert_eq!(positions[0].current_value, 0, "a losing share is worth nothing");
    assert!(positions[0].profit_loss < 0, "a losing position must show a loss");
}

// A winning position settles at 1.0 per share and the payout reaches the
// holder's balance. The original code marked the position but never credited
// anything, so the winner received nothing.
#[test]
fn winning_position_pays_out() {
    let svc = PredictionMarketService::new();
    let market = market_with_two_outcomes(&svc);

    svc.add_funds(1, 1_000_000);
    buy(&svc, 1, market, 0, 100);

    let balance_after_buy = svc.get_balance(1);
    svc.resolve_market(market, 0).expect("resolution must succeed");

    assert_eq!(
        svc.get_balance(1),
        balance_after_buy + 100,
        "100 winning shares must pay 100 units"
    );

    let positions = svc.get_user_positions(1);
    assert_eq!(positions[0].current_value, 100, "a winning share settles at 1.0");
}

// The payout is the share count, not the share count times the traded price.
#[test]
fn payout_uses_share_count_not_traded_price() {
    let svc = PredictionMarketService::new();
    let market = market_with_two_outcomes(&svc);

    svc.add_funds(1, 10_000_000);
    buy(&svc, 1, market, 0, 250);

    let before = svc.get_balance(1);
    svc.resolve_market(market, 0).expect("resolution must succeed");

    assert_eq!(svc.get_balance(1), before + 250);
    assert_ne!(
        svc.get_balance(1),
        before + 150,
        "paying quantity * price would underpay the holder"
    );
}

// A partially filled order settles on the filled quantity only.
#[test]
fn partial_fill_settles_on_filled_quantity() {
    let svc = PredictionMarketService::new();
    let market = market_with_two_outcomes(&svc);

    svc.add_funds(1, 1_000_000);

    let order = svc
        .place_order(
            1,
            market,
            0,
            OrderType::Limit,
            OrderSide::Buy,
            PRICE_60,
            100,
            None,
        )
        .expect("order placement must succeed");
    // Only 40 of the 100 ordered shares ever fill.
    svc.fill_order(order.order_id, 40, PRICE_60)
        .expect("fill must succeed");

    let before = svc.get_balance(1);
    svc.resolve_market(market, 0).expect("resolution must succeed");

    assert_eq!(
        svc.get_balance(1),
        before + 40,
        "only the filled 40 shares may settle"
    );
}

// Placing an order without funds must be refused.
#[test]
fn order_without_funds_is_refused() {
    let svc = PredictionMarketService::new();
    let market = market_with_two_outcomes(&svc);

    let result = svc.place_order(
        1,
        market,
        0,
        OrderType::Limit,
        OrderSide::Buy,
        PRICE_60,
        100,
        None,
    );

    assert!(
        matches!(result, Err(PredictionError::InsufficientBalance(_, 0))),
        "an unfunded order must be refused"
    );
}

// An order on a resolved market must be refused.
#[test]
fn order_on_resolved_market_is_refused() {
    let svc = PredictionMarketService::new();
    let market = market_with_two_outcomes(&svc);

    svc.add_funds(1, 1_000_000);
    svc.resolve_market(market, 0).expect("resolution must succeed");

    let result = svc.place_order(
        1,
        market,
        0,
        OrderType::Limit,
        OrderSide::Buy,
        PRICE_60,
        10,
        None,
    );

    assert!(
        matches!(result, Err(PredictionError::MarketNotActive)),
        "a resolved market must not accept orders"
    );
}

// Invalid prices, amounts and outcomes must be refused.
#[test]
fn invalid_order_parameters_are_refused() {
    let svc = PredictionMarketService::new();
    let market = market_with_two_outcomes(&svc);
    svc.add_funds(1, 10_000_000);

    let zero_price = svc.place_order(
        1, market, 0, OrderType::Limit, OrderSide::Buy, 0, 10, None,
    );
    assert!(matches!(zero_price, Err(PredictionError::InvalidPrice(_))));

    let above_one = svc.place_order(
        1, market, 0, OrderType::Limit, OrderSide::Buy, 1_000_001, 10, None,
    );
    assert!(matches!(above_one, Err(PredictionError::InvalidPrice(_))));

    let zero_amount = svc.place_order(
        1, market, 0, OrderType::Limit, OrderSide::Buy, PRICE_60, 0, None,
    );
    assert!(matches!(zero_amount, Err(PredictionError::InvalidAmount(_))));

    let bad_outcome = svc.place_order(
        1, market, 9, OrderType::Limit, OrderSide::Buy, PRICE_60, 10, None,
    );
    assert!(matches!(bad_outcome, Err(PredictionError::InvalidOutcome(9))));
}

// Resolving a market twice must be refused.
#[test]
fn double_resolution_is_refused() {
    let svc = PredictionMarketService::new();
    let market = market_with_two_outcomes(&svc);

    svc.resolve_market(market, 0).expect("first resolution succeeds");
    let second = svc.resolve_market(market, 1);

    assert!(
        matches!(second, Err(PredictionError::MarketNotActive)),
        "a market may only be resolved once"
    );

    let market_state = svc.get_market(market).expect("market must exist");
    assert_eq!(market_state.status, MarketStatus::Resolved);
    assert_eq!(market_state.resolved_outcome, Some(0));
}

// A losing holder must not be credited when the market resolves.
#[test]
fn losing_holder_is_not_credited() {
    let svc = PredictionMarketService::new();
    let market = market_with_two_outcomes(&svc);

    svc.add_funds(1, 1_000_000);
    buy(&svc, 1, market, 0, 100);

    let before = svc.get_balance(1);
    svc.resolve_market(market, 1).expect("resolution must succeed");

    assert_eq!(svc.get_balance(1), before, "a losing position must not be paid");
}

// Identifiers must be unique, otherwise a fill could be recorded against the
// wrong order.
#[test]
fn identifiers_are_unique_and_non_zero() {
    let svc = PredictionMarketService::new();

    let first = market_with_two_outcomes(&svc);
    let second = market_with_two_outcomes(&svc);
    assert_ne!(first, second, "market ids must be unique");
    assert!(first > 0 && second > 0, "ids must be non-zero");

    svc.add_funds(1, 10_000_000);
    let a = svc
        .place_order(1, first, 0, OrderType::Limit, OrderSide::Buy, PRICE_60, 1, None)
        .unwrap();
    let b = svc
        .place_order(1, second, 0, OrderType::Limit, OrderSide::Buy, PRICE_60, 1, None)
        .unwrap();
    assert_ne!(a.order_id, b.order_id, "order ids must be unique");
}

// A cancelled order must not be fillable.
#[test]
fn cancelled_order_cannot_be_filled() {
    let svc = PredictionMarketService::new();
    let market = market_with_two_outcomes(&svc);

    svc.add_funds(1, 1_000_000);
    let order = svc
        .place_order(1, market, 0, OrderType::Limit, OrderSide::Buy, PRICE_60, 10, None)
        .unwrap();

    svc.cancel_order(order.order_id).expect("cancel must succeed");

    let result = svc.fill_order(order.order_id, 10, PRICE_60);
    assert!(
        matches!(result, Err(PredictionError::OrderCancelled(_))),
        "a cancelled order must not fill"
    );
}

// Settlement must not move money between users: a losing holder is not
// credited with the winner's payout.
#[test]
fn settlement_does_not_credit_other_users() {
    let svc = PredictionMarketService::new();
    let market = market_with_two_outcomes(&svc);

    svc.add_funds(1, 1_000_000);
    svc.add_funds(2, 1_000_000);

    buy(&svc, 1, market, 0, 100);
    buy(&svc, 2, market, 1, 50);

    let user2_before = svc.get_balance(2);
    svc.resolve_market(market, 0).expect("resolution must succeed");

    assert_eq!(
        svc.get_balance(2),
        user2_before,
        "a losing holder must not receive the winner's payout"
    );
}

// A market that does not exist cannot be resolved.
#[test]
fn resolving_unknown_market_fails() {
    let svc = PredictionMarketService::new();
    let result = svc.resolve_market(9999, 0);
    assert!(matches!(result, Err(PredictionError::MarketNotFound(9999))));
}
