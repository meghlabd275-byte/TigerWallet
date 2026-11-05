//! Tests for the intent routing service.
//!
//! The quote path is exercised against a real price feed, and intent
//! registration is exercised with real secp256k1 signatures so that an intent
//! signed by someone other than its stated owner is rejected.

use super::*;

const TOKEN_A: &str = "0x1111111111111111111111111111111111111111";
const TOKEN_B: &str = "0x2222222222222222222222222222222222222222";

fn type_hash() -> [u8; 32] {
    [0x11u8; 32]
}

fn domain() -> [u8; 32] {
    [0x22u8; 32]
}

fn router() -> IntentRouter {
    IntentRouter::new(domain(), type_hash())
}

fn token(chain: ChainId, address: &str, symbol: &str, decimals: u8) -> Token {
    Token {
        chain,
        address: address.to_string(),
        symbol: symbol.to_string(),
        decimals,
    }
}

fn signing_key() -> k256::ecdsa::SigningKey {
    k256::ecdsa::SigningKey::from_slice(&[0x42u8; 32]).unwrap()
}

fn address_of(key: &k256::ecdsa::SigningKey) -> String {
    use sha3::{Digest, Keccak256};
    let point = key.verifying_key().to_encoded_point(false);
    let hash = Keccak256::digest(&point.as_bytes()[1..]);
    format!("0x{}", hex::encode(&hash[12..]))
}

/// Signs `intent` for `key`, replacing any existing signature.
fn sign_for(router: &IntentRouter, key: &k256::ecdsa::SigningKey, intent: &mut Intent) {
    use k256::ecdsa::signature::hazmat::PrehashSigner;
    use sha3::{Digest, Keccak256};

    let struct_hash = router.intent_struct_hash(intent).unwrap();
    let mut preimage = Vec::new();
    preimage.extend_from_slice(&[0x19, 0x01]);
    // The router's own separator, so a test can sign for a different one.
    preimage.extend_from_slice(&router.domain_separator());
    preimage.extend_from_slice(&struct_hash);
    let digest = Keccak256::digest(&preimage);

    let (sig, recid): (k256::ecdsa::Signature, k256::ecdsa::RecoveryId) =
        key.sign_prehash(&digest).unwrap();

    let mut bytes = sig.to_bytes().to_vec();
    bytes.push(recid.to_byte());
    intent.signature = bytes;
}

/// Builds an intent owned by `key`, with a real signature.
fn signed_intent(router: &IntentRouter, key: &k256::ecdsa::SigningKey) -> Intent {
    let now = now_secs();
    let mut intent = Intent {
        id: "intent-1".to_string(),
        intent_type: IntentType::Swap,
        owner: address_of(key),
        from_token: token(ChainId::Ethereum, TOKEN_A, "USDC", 6),
        to_token: token(ChainId::Ethereum, TOKEN_B, "WETH", 18),
        from_amount: 1_000_000_000,
        min_out_amount: 400_000_000_000_000_000,
        to_chain: ChainId::Ethereum,
        deadline: now + 3600,
        fill_deadline: now + 1800,
        nonce: 1,
        signature: Vec::new(),
        status: IntentStatus::Open,
        created_at: now,
        updated_at: now,
    };

    sign_for(router, key, &mut intent);
    intent
}

fn solver(id: &str) -> Solver {
    Solver {
        id: id.to_string(),
        name: id.to_string(),
        supported_chains: vec![ChainId::Ethereum],
        payout_chains: vec![ChainId::Ethereum],
        liquidity: vec![(ChainId::Ethereum, TOKEN_A.to_string())],
        fee_bps: 30,
        avg_fill_time_ms: 5000,
        success_rate: 0.99,
        is_active: true,
    }
}

async fn router_with_prices() -> IntentRouter {
    let r = router();
    r.set_price(ChainId::Ethereum, TOKEN_A, 1.0).await; // USDC
    r.set_price(ChainId::Ethereum, TOKEN_B, 2500.0).await; // WETH
    r
}

// ============================================================================
// Intent registration
// ============================================================================

#[tokio::test]
async fn register_accepts_a_correctly_signed_intent() {
    let r = router();
    let key = signing_key();
    r.register_intent(signed_intent(&r, &key)).await.unwrap();

    let stored = r.get_intent("intent-1").await.unwrap();
    assert_eq!(stored.owner, address_of(&key));
}

#[tokio::test]
async fn register_rejects_an_intent_signed_by_someone_else() {
    let r = router();
    let key = signing_key();
    let mut intent = signed_intent(&r, &key);

    // Claim the intent belongs to a different account. The signature no longer
    // matches the owner.
    intent.owner = "0x9999999999999999999999999999999999999999".to_string();

    assert!(r.register_intent(intent).await.is_err());
}

#[tokio::test]
async fn register_rejects_an_unsigned_intent() {
    let r = router();
    let key = signing_key();
    let mut intent = signed_intent(&r, &key);
    intent.signature = Vec::new();
    assert!(r.register_intent(intent).await.is_err());
}

#[tokio::test]
async fn register_rejects_a_tampered_intent() {
    let r = router();
    let key = signing_key();
    let mut intent = signed_intent(&r, &key);

    // Raise the amount after signing.
    intent.from_amount = 999_999_999_999;
    assert!(r.register_intent(intent).await.is_err());
}

#[tokio::test]
async fn register_rejects_a_signature_from_another_domain() {
    let r = router();
    let other = IntentRouter::new([0x33u8; 32], type_hash());
    let key = signing_key();

    // Signed for a different verifying contract.
    let intent = signed_intent(&other, &key);
    assert!(r.register_intent(intent).await.is_err());
}

#[tokio::test]
async fn register_rejects_a_duplicate_id() {
    let r = router();
    let key = signing_key();
    r.register_intent(signed_intent(&r, &key)).await.unwrap();
    assert!(
        r.register_intent(signed_intent(&r, &key)).await.is_err(),
        "a duplicate id must not overwrite the stored intent"
    );
}

#[tokio::test]
async fn register_rejects_bad_terms() {
    let r = router();
    let key = signing_key();

    let mut i = signed_intent(&r, &key);
    i.id = "  ".to_string();
    assert!(r.register_intent(i).await.is_err());

    let mut i = signed_intent(&r, &key);
    i.from_amount = 0;
    assert!(r.register_intent(i).await.is_err());

    let mut i = signed_intent(&r, &key);
    i.min_out_amount = 0;
    assert!(r.register_intent(i).await.is_err());

    let mut i = signed_intent(&r, &key);
    i.deadline = now_secs() - 1;
    assert!(r.register_intent(i).await.is_err());

    let mut i = signed_intent(&r, &key);
    i.fill_deadline = i.deadline + 1;
    assert!(r.register_intent(i).await.is_err());

    let mut i = signed_intent(&r, &key);
    i.to_token = i.from_token.clone();
    assert!(r.register_intent(i).await.is_err());
}

// ============================================================================
// Cancelling
// ============================================================================

#[tokio::test]
async fn cancel_requires_the_owner() {
    let r = router();
    let key = signing_key();
    r.register_intent(signed_intent(&r, &key)).await.unwrap();

    assert!(r.cancel_intent("intent-1", "0xdeadbeef").await.is_err());
    r.cancel_intent("intent-1", &address_of(&key)).await.unwrap();

    let stored = r.get_intent("intent-1").await.unwrap();
    assert_eq!(stored.status, IntentStatus::Cancelled);
}

#[tokio::test]
async fn cancel_is_case_insensitive_on_the_owner_address() {
    let r = router();
    let key = signing_key();
    r.register_intent(signed_intent(&r, &key)).await.unwrap();

    let checksummed = address_of(&key).to_uppercase().replace("0X", "0x");
    r.cancel_intent("intent-1", &checksummed).await.unwrap();
}

#[tokio::test]
async fn cancel_rejects_a_filled_intent() {
    let r = router();
    let key = signing_key();
    r.register_intent(signed_intent(&r, &key)).await.unwrap();
    r.register_solver(solver("solver-1")).await.unwrap();

    let now = now_secs();
    r.fill_intent(
        "intent-1",
        SolverSolution {
            solver: "solver-1".to_string(),
            intent_id: "intent-1".to_string(),
            from_amount: 1_000_000_000,
            to_amount: 400_000_000_000_000_000,
            gas_estimate: 150_000,
            execution_data: Vec::new(),
            valid_until: now + 60,
            fees: 0,
        },
    )
    .await
    .unwrap();

    assert!(r.cancel_intent("intent-1", &address_of(&key)).await.is_err());
}

#[tokio::test]
async fn cancel_rejects_an_unknown_intent() {
    let r = router();
    assert!(r.cancel_intent("nope", "0x0").await.is_err());
}

// ============================================================================
// Solver registration
// ============================================================================

#[tokio::test]
async fn register_solver_validates_its_input() {
    let r = router();

    let mut s = solver("s1");
    s.id = "  ".to_string();
    assert!(r.register_solver(s).await.is_err());

    let mut s = solver("s1");
    s.fee_bps = 10_001;
    assert!(r.register_solver(s).await.is_err());

    let mut s = solver("s1");
    s.success_rate = 1.5;
    assert!(r.register_solver(s).await.is_err());

    let mut s = solver("s1");
    s.success_rate = f64::NAN;
    assert!(r.register_solver(s).await.is_err());

    r.register_solver(solver("s1")).await.unwrap();
}

// ============================================================================
// Quotes
// ============================================================================

#[tokio::test]
async fn quote_uses_the_price_feed() {
    let r = router_with_prices().await;
    r.register_solver(solver("solver-1")).await.unwrap();

    // 1000 USDC into WETH at $2500, less the solver's 30 bps fee.
    let quote = r
        .get_quote(QuoteRequest {
            from_token: token(ChainId::Ethereum, TOKEN_A, "USDC", 6),
            to_token: token(ChainId::Ethereum, TOKEN_B, "WETH", 18),
            from_amount: 1_000_000_000,
            to_chain: ChainId::Ethereum,
            slippage_bps: 50,
        })
        .await
        .unwrap();

    // 1000 / 2500 = 0.4 WETH, minus 0.3% = 0.3988 WETH. The exact value is
    // 398800000000000000; f64 conversion lands within one base unit of it.
    let expected = 398_800_000_000_000_000u128;
    assert!(
        quote.to_amount.abs_diff(expected) <= 1,
        "got {}",
        quote.to_amount
    );
    assert_eq!(quote.quotes.len(), 1);
    assert!(quote.quotes[0].fees.abs_diff(1_200_000_000_000_000) <= 1);
}

#[tokio::test]
async fn quote_fails_without_a_price() {
    let r = router();
    r.register_solver(solver("solver-1")).await.unwrap();

    // The old implementation returned from_amount * 1000 here.
    let err = r
        .get_quote(QuoteRequest {
            from_token: token(ChainId::Ethereum, TOKEN_A, "USDC", 6),
            to_token: token(ChainId::Ethereum, TOKEN_B, "WETH", 18),
            from_amount: 1_000_000_000,
            to_chain: ChainId::Ethereum,
            slippage_bps: 50,
        })
        .await;
    assert!(err.is_err(), "a quote must not be invented without prices");
}

#[tokio::test]
async fn quote_rejects_invalid_amounts_and_slippage() {
    let r = router_with_prices().await;
    r.register_solver(solver("solver-1")).await.unwrap();

    let base = QuoteRequest {
        from_token: token(ChainId::Ethereum, TOKEN_A, "USDC", 6),
        to_token: token(ChainId::Ethereum, TOKEN_B, "WETH", 18),
        from_amount: 0,
        to_chain: ChainId::Ethereum,
        slippage_bps: 50,
    };
    assert!(r.get_quote(base.clone()).await.is_err());

    let mut over = base.clone();
    over.from_amount = 1_000_000;
    over.slippage_bps = 0;
    assert!(r.get_quote(over).await.is_err());

    let mut over = base.clone();
    over.from_amount = 1_000_000;
    over.slippage_bps = 5_001;
    assert!(r.get_quote(over).await.is_err());
}

#[tokio::test]
async fn quote_fails_without_a_solver() {
    let r = router_with_prices().await;
    assert!(r
        .get_quote(QuoteRequest {
            from_token: token(ChainId::Ethereum, TOKEN_A, "USDC", 6),
            to_token: token(ChainId::Ethereum, TOKEN_B, "WETH", 18),
            from_amount: 1_000_000_000,
            to_chain: ChainId::Ethereum,
            slippage_bps: 50,
        })
        .await
        .is_err());
}

#[tokio::test]
async fn quote_picks_the_best_net_output() {
    let r = router_with_prices().await;

    let mut cheap = solver("cheap");
    cheap.fee_bps = 10; // 0.10%
    let mut pricey = solver("pricey");
    pricey.fee_bps = 300; // 3.00%

    r.register_solver(cheap).await.unwrap();
    r.register_solver(pricey).await.unwrap();

    let quote = r
        .get_quote(QuoteRequest {
            from_token: token(ChainId::Ethereum, TOKEN_A, "USDC", 6),
            to_token: token(ChainId::Ethereum, TOKEN_B, "WETH", 18),
            from_amount: 1_000_000_000,
            to_chain: ChainId::Ethereum,
            slippage_bps: 50,
        })
        .await
        .unwrap();

    assert_eq!(quote.quotes.len(), 2);
    assert_eq!(quote.quotes[0].solver, "cheap");
    assert!(
        quote.to_amount.abs_diff(399_600_000_000_000_000) <= 1,
        "got {}",
        quote.to_amount
    );

    // Price impact reflects the fee actually taken, not an asserted constant.
    assert_eq!(quote.price_impact_bps, 10);
}

#[tokio::test]
async fn quote_gas_estimate_depends_on_the_chain() {
    let r = router();
    r.set_price(ChainId::Solana, TOKEN_A, 1.0).await;
    r.set_price(ChainId::Solana, TOKEN_B, 100.0).await;

    let mut s = solver("solana-solver");
    s.supported_chains = vec![ChainId::Solana];
    s.payout_chains = vec![ChainId::Solana];
    s.liquidity = vec![(ChainId::Solana, TOKEN_A.to_string())];
    r.register_solver(s).await.unwrap();

    let quote = r
        .get_quote(QuoteRequest {
            from_token: token(ChainId::Solana, TOKEN_A, "USDC", 6),
            to_token: token(ChainId::Solana, TOKEN_B, "SOL", 9),
            from_amount: 1_000_000_000,
            to_chain: ChainId::Solana,
            slippage_bps: 50,
        })
        .await
        .unwrap();

    // Solana fees are not the same as Ethereum's.
    assert_eq!(quote.gas_estimate, 5_000);
}

#[tokio::test]
async fn cached_quote_is_returned_only_while_valid() {
    let r = router_with_prices().await;
    r.register_solver(solver("solver-1")).await.unwrap();

    let request = QuoteRequest {
        from_token: token(ChainId::Ethereum, TOKEN_A, "USDC", 6),
        to_token: token(ChainId::Ethereum, TOKEN_B, "WETH", 18),
        from_amount: 1_000_000_000,
        to_chain: ChainId::Ethereum,
        slippage_bps: 50,
    };

    assert!(r.get_cached_quote(&request).await.is_none());
    r.get_quote(request.clone()).await.unwrap();
    assert!(r.get_cached_quote(&request).await.is_some());
}

// ============================================================================
// Filling
// ============================================================================

async fn register_and_fill(
    to_amount: u128,
    from_amount: u128,
    solver_id: &str,
    valid_until: u64,
) -> Result<Intent, String> {
    let r = router();
    let key = signing_key();
    r.register_intent(signed_intent(&r, &key)).await.unwrap();
    r.register_solver(solver("solver-1")).await.unwrap();

    r.fill_intent(
        "intent-1",
        SolverSolution {
            solver: solver_id.to_string(),
            intent_id: "intent-1".to_string(),
            from_amount,
            to_amount,
            gas_estimate: 150_000,
            execution_data: Vec::new(),
            valid_until,
            fees: 0,
        },
    )
    .await
}

#[tokio::test]
async fn fill_accepts_a_solution_that_meets_the_minimum() {
    let now = now_secs();
    let filled = register_and_fill(400_000_000_000_000_000, 1_000_000_000, "solver-1", now + 60)
        .await
        .unwrap();
    assert_eq!(filled.status, IntentStatus::Filled);
}

#[tokio::test]
async fn fill_rejects_a_solution_below_the_minimum() {
    let now = now_secs();
    // The old implementation accepted this and marked the intent Filled.
    let err = register_and_fill(1, 1_000_000_000, "solver-1", now + 60).await;
    assert!(err.is_err(), "an under-delivering solution must be rejected");
}

#[tokio::test]
async fn fill_accepts_a_partial_fill_at_the_pro_rata_minimum() {
    let now = now_secs();
    // Half the input, so half the minimum output is required.
    let filled = register_and_fill(200_000_000_000_000_000, 500_000_000, "solver-1", now + 60)
        .await
        .unwrap();
    assert_eq!(filled.status, IntentStatus::PartiallyFilled);
}

#[tokio::test]
async fn fill_rejects_a_solution_from_an_unknown_solver() {
    let now = now_secs();
    let err = register_and_fill(400_000_000_000_000_000, 1_000_000_000, "not-registered", now + 60).await;
    assert!(err.is_err());
}

#[tokio::test]
async fn fill_rejects_an_expired_solution() {
    let err = register_and_fill(400_000_000_000_000_000, 1_000_000_000, "solver-1", 1).await;
    assert!(err.is_err());
}

#[tokio::test]
async fn fill_rejects_a_solution_for_another_intent() {
    let r = router();
    let key = signing_key();
    r.register_intent(signed_intent(&r, &key)).await.unwrap();
    r.register_solver(solver("solver-1")).await.unwrap();

    let err = r
        .fill_intent(
            "intent-1",
            SolverSolution {
                solver: "solver-1".to_string(),
                intent_id: "intent-2".to_string(),
                from_amount: 1_000_000_000,
                to_amount: 400_000_000_000_000_000,
                gas_estimate: 150_000,
                execution_data: Vec::new(),
                valid_until: now_secs() + 60,
                fees: 0,
            },
        )
        .await;
    assert!(err.is_err());
}

#[tokio::test]
async fn fill_rejects_an_already_filled_intent() {
    let now = now_secs();
    let r = router();
    let key = signing_key();
    r.register_intent(signed_intent(&r, &key)).await.unwrap();
    r.register_solver(solver("solver-1")).await.unwrap();

    let solution = || SolverSolution {
        solver: "solver-1".to_string(),
        intent_id: "intent-1".to_string(),
        from_amount: 1_000_000_000,
        to_amount: 400_000_000_000_000_000,
        gas_estimate: 150_000,
        execution_data: Vec::new(),
        valid_until: now + 60,
        fees: 0,
    };

    r.fill_intent("intent-1", solution()).await.unwrap();
    assert!(r.fill_intent("intent-1", solution()).await.is_err());
}

#[tokio::test]
async fn fill_marks_an_expired_intent_expired() {
    let r = router();
    let key = signing_key();
    let mut intent = signed_intent(&r, &key);
    let soon = now_secs() + 1;
    intent.deadline = soon;
    intent.fill_deadline = soon;
    sign_for(&r, &key, &mut intent);
    r.register_intent(intent).await.unwrap();
    r.register_solver(solver("solver-1")).await.unwrap();

    // Wait for the deadline to pass.
    tokio::time::sleep(std::time::Duration::from_millis(1100)).await;

    let err = r
        .fill_intent(
            "intent-1",
            SolverSolution {
                solver: "solver-1".to_string(),
                intent_id: "intent-1".to_string(),
                from_amount: 1_000_000_000,
                to_amount: 400_000_000_000_000_000,
                gas_estimate: 150_000,
                execution_data: Vec::new(),
                valid_until: now_secs() + 60,
                fees: 0,
            },
        )
        .await;
    assert!(err.is_err());
    assert_eq!(
        r.get_intent("intent-1").await.unwrap().status,
        IntentStatus::Expired
    );
}

// ============================================================================
// Queries
// ============================================================================

#[tokio::test]
async fn intents_are_filtered_by_owner() {
    let r = router();
    let key = signing_key();
    r.register_intent(signed_intent(&r, &key)).await.unwrap();

    let owned = r.get_intents_by_owner(&address_of(&key)).await;
    assert_eq!(owned.len(), 1);
    assert!(r.get_intents_by_owner("0xnobody").await.is_empty());
}

// ============================================================================
// Helpers
// ============================================================================

#[test]
fn price_scaling_is_fixed_point() {
    assert_eq!(scale_price(1.0).unwrap(), 100_000_000);
    assert_eq!(scale_price(2500.0).unwrap(), 250_000_000_000);
    assert!(scale_price(0.0).is_err());
    assert!(scale_price(-1.0).is_err());
    assert!(scale_price(f64::NAN).is_err());
}

#[test]
fn pow10_covers_token_decimals_and_rejects_the_rest() {
    assert_eq!(pow10(0).unwrap(), 1);
    assert_eq!(pow10(6).unwrap(), 1_000_000);
    assert_eq!(pow10(18).unwrap(), 1_000_000_000_000_000_000);
    assert_eq!(pow10(38).unwrap(), 100_000_000_000_000_000_000_000_000_000_000_000_000);
    assert!(pow10(39).is_err());
}

#[test]
fn address_parsing_rejects_malformed_input() {
    assert!(parse_address(TOKEN_A).is_some());
    assert!(parse_address("0x1234").is_none());
    assert!(parse_address("1111111111111111111111111111111111111111").is_none());
    assert!(parse_address("0xzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz").is_none());
}

#[test]
fn solver_support_requires_liquidity_and_payout() {
    let s = solver("s1");
    let from = token(ChainId::Ethereum, TOKEN_A, "USDC", 6);

    assert!(s.supports(&from, ChainId::Ethereum));

    // No liquidity advertised for this token.
    let other = token(ChainId::Ethereum, TOKEN_B, "WETH", 18);
    assert!(!s.supports(&other, ChainId::Ethereum));

    // Inactive solvers cannot serve anything.
    let mut inactive = s.clone();
    inactive.is_active = false;
    assert!(!inactive.supports(&from, ChainId::Ethereum));

    // No payout capability on the destination chain.
    let mut no_payout = s.clone();
    no_payout.payout_chains = Vec::new();
    assert!(!no_payout.supports(&from, ChainId::Ethereum));
}

#[test]
fn price_feed_rejects_non_positive_prices() {
    let mut feed = PriceFeed::new();
    feed.set(ChainId::Ethereum, TOKEN_A, 0.0);
    assert!(feed.get(ChainId::Ethereum, TOKEN_A).is_none());

    feed.set(ChainId::Ethereum, TOKEN_A, -1.0);
    assert!(feed.get(ChainId::Ethereum, TOKEN_A).is_none());

    feed.set(ChainId::Ethereum, TOKEN_A, 1.0);
    assert_eq!(feed.get(ChainId::Ethereum, TOKEN_A), Some(1.0));

    // Lookup is case-insensitive on the address.
    assert_eq!(feed.get(ChainId::Ethereum, &TOKEN_A.to_uppercase()), Some(1.0));
}

#[test]
fn chain_ids_match_the_networks() {
    assert_eq!(ChainId::Ethereum as u64, 1);
    assert_eq!(ChainId::Polygon as u64, 137);
    assert_eq!(ChainId::Arbitrum as u64, 42161);
    assert_eq!(ChainId::Optimism as u64, 10);
    assert_eq!(ChainId::Base as u64, 8453);
    assert_eq!(ChainId::Avalanche as u64, 43114);

    assert_eq!(ChainId::from_u64(1), Some(ChainId::Ethereum));
    assert_eq!(ChainId::from_u64(999), None);
}
