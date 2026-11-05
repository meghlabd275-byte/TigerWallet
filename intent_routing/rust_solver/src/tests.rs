//! Tests for the intent solver.
//!
//! The u256 arithmetic is checked against values computed independently, and
//! the EIP-712 verification is exercised with a real secp256k1 key so a
//! signature that does not match the intent's user is rejected.

use super::*;

// ============================================================================
// u256 arithmetic
// ============================================================================

#[test]
fn u256_from_and_display_round_trip() {
    assert_eq!(u256::from(0u64).to_dec_str(), "0");
    assert_eq!(u256::from(1u64).to_dec_str(), "1");
    assert_eq!(u256::from(1000u64).to_dec_str(), "1000");
    assert_eq!(u256::from(u64::MAX).to_dec_str(), "18446744073709551615");
}

#[test]
fn u256_parses_decimals() {
    assert_eq!(u256::from_dec_str("0").unwrap(), u256::zero());
    assert_eq!(u256::from_dec_str("123456789").unwrap().to_dec_str(), "123456789");

    // 2^128, which needs two limbs.
    let big = u256::from_dec_str("340282366920938463463374607431768211456").unwrap();
    assert_eq!(big.to_dec_str(), "340282366920938463463374607431768211456");
}

#[test]
fn u256_rejects_bad_decimal_input() {
    for bad in ["", "12a", "-1", "1.5", " 12"] {
        assert!(u256::from_dec_str(bad).is_err(), "{:?} should be rejected", bad);
    }

    // 2^256 does not fit.
    assert!(u256::from_dec_str(
        "115792089237316195423570985008687907853269984665640564039457584007913129639936"
    )
    .is_err());
}

#[test]
fn u256_addition_carries_across_limbs() {
    let a = u256::from(u64::MAX);
    let b = u256::from(1u64);
    let sum = a + b;
    assert_eq!(sum.to_dec_str(), "18446744073709551616"); // 2^64
}

#[test]
fn u256_addition_overflow_is_detected() {
    let max = u256([u64::MAX; 4]);
    assert!(max.checked_add(u256::one()).is_none());
    assert_eq!((max + u256::one()).to_dec_str(), "0");
}

#[test]
fn u256_subtraction_borrows_correctly() {
    // 2^64 - 1, which must borrow from the second limb.
    let a = u256([0, 1, 0, 0]);
    let b = u256::one();
    let diff = a - b;
    assert_eq!(diff.to_dec_str(), "18446744073709551615");

    // 2^64 - 1 computed the other way round.
    let c = u256([u64::MAX, 0, 0, 0]);
    assert_eq!(c.to_dec_str(), "18446744073709551615");
}

#[test]
fn u256_subtraction_below_zero_saturates() {
    // The old implementation wrapped here and produced a huge number.
    assert_eq!((u256::one() - u256::from(5u64)).to_dec_str(), "0");
    assert!(u256::one().checked_sub(u256::from(5u64)).is_none());
}

#[test]
fn u256_multiplication_does_not_index_out_of_bounds() {
    // i = j = 3 in the old loop wrote to result[7] on a 4-element array.
    let max = u256([u64::MAX; 4]);
    assert!(max.checked_mul(max).is_none());

    // A product that fits exactly: 2^128 * 2^127 would not, so use smaller.
    let a = u256::from(u64::MAX);
    let b = u256::from(2u64);
    assert_eq!(
        a.checked_mul(b).unwrap().to_dec_str(),
        "36893488147419103230"
    );
}

#[test]
fn u256_multiplication_saturates_on_overflow() {
    let max = u256([u64::MAX; 4]);
    assert_eq!((max * max).to_dec_str(), u256([u64::MAX; 4]).to_dec_str());
}

#[test]
fn u256_scaling_rounds_to_nearest() {
    // 1.5 USDC at 6 decimals is 1_500_000 base units.
    assert_eq!(u256::scaled_from_f64(1.5, 6).unwrap().to_dec_str(), "1500000");
    assert_eq!(u256::scaled_from_f64(0.0, 6).unwrap().to_dec_str(), "0");

    // 1 wei of an 18-decimal token.
    assert_eq!(
        u256::scaled_from_f64(1e-18, 18).unwrap().to_dec_str(),
        "1"
    );

    // Rounding: 1.0000005 at 6 decimals rounds to 1_000_001.
    assert_eq!(
        u256::scaled_from_f64(1.0000005, 6).unwrap().to_dec_str(),
        "1000001"
    );
}

#[test]
fn u256_scaling_rejects_invalid_input() {
    assert!(u256::scaled_from_f64(-1.0, 6).is_err());
    assert!(u256::scaled_from_f64(f64::NAN, 6).is_err());
    assert!(u256::scaled_from_f64(f64::INFINITY, 6).is_err());
    assert!(u256::scaled_from_f64(f64::MAX, 18).is_err());
}

#[test]
fn u256_serialises_as_a_decimal_string() {
    // A JSON number is an IEEE-754 double and cannot hold 256 bits exactly.
    let value = u256::from_dec_str("123456789012345678901234567890").unwrap();
    let json = serde_json::to_string(&value).unwrap();
    assert_eq!(json, "\"123456789012345678901234567890\"");

    let back: u256 = serde_json::from_str(&json).unwrap();
    assert_eq!(back, value);
}

#[test]
fn u256_deserialises_json_numbers_for_small_values() {
    let value: u256 = serde_json::from_str("42").unwrap();
    assert_eq!(value.to_dec_str(), "42");
}

#[test]
fn u256_ordering_is_numeric_not_lexicographic() {
    let small = u256::from(2u64);
    let big = u256([0, 1, 0, 0]); // 2^64
    assert!(small < big);
    assert!(big > small);
}

// ============================================================================
// Intent validation
// ============================================================================

fn token(chain: Chain, symbol: &str, decimals: u8) -> Token {
    Token {
        chain,
        address: format!("0x{}", "11".repeat(20)),
        symbol: symbol.to_string(),
        decimals,
    }
}

fn amount(chain: Chain, symbol: &str, decimals: u8, units: f64) -> Amount {
    Amount {
        token: token(chain, symbol, decimals),
        amount: u256::scaled_from_f64(units, decimals).unwrap(),
        amount_usd: 0.0,
    }
}

fn intent(chain: Chain, kind: IntentType) -> Intent {
    let now = now_secs();
    Intent {
        id: "intent-1".to_string(),
        user: "0x1111111111111111111111111111111111111111".to_string(),
        chain,
        intent_type: kind,
        sell: Some(amount(chain, "USDC", 6, 1000.0)),
        buy: Some(amount(chain, "WBTC", 8, 0.015)),
        slippage_bps: 50,
        deadline: now + 3600,
        fill_deadline: now + 1800,
        status: IntentStatus::Pending,
        created_at: now,
        updated_at: now,
        nonce: 1,
        signature: String::new(),
    }
}

#[test]
fn validate_accepts_a_well_formed_intent() {
    assert!(IntentParser::validate(&intent(Chain::Ethereum, IntentType::Swap)).is_ok());
}

#[test]
fn validate_rejects_an_expired_deadline() {
    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.deadline = now_secs() - 1;
    assert!(IntentParser::validate(&i).is_err());
}

#[test]
fn validate_rejects_excessive_slippage() {
    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.slippage_bps = 10_001;
    assert!(IntentParser::validate(&i).is_err());
}

#[test]
fn validate_rejects_zero_amounts() {
    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.sell = Some(Amount {
        token: token(Chain::Ethereum, "USDC", 6),
        amount: u256::zero(),
        amount_usd: 0.0,
    });
    assert!(IntentParser::validate(&i).is_err());

    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.buy = Some(Amount {
        token: token(Chain::Ethereum, "WBTC", 8),
        amount: u256::zero(),
        amount_usd: 0.0,
    });
    assert!(IntentParser::validate(&i).is_err());
}

#[test]
fn validate_requires_both_sides() {
    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.sell = None;
    assert!(IntentParser::validate(&i).is_err());

    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.buy = None;
    assert!(IntentParser::validate(&i).is_err());
}

#[test]
fn validate_requires_cross_chain_type_for_cross_chain_tokens() {
    // Sell on Ethereum, buy on Base, but declared as a same-chain swap.
    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.buy = Some(amount(Chain::Base, "USDC", 6, 1000.0));
    assert!(IntentParser::validate(&i).is_err());

    let mut i = intent(Chain::Ethereum, IntentType::CrossChainSwap);
    i.buy = Some(amount(Chain::Base, "USDC", 6, 1000.0));
    assert!(IntentParser::validate(&i).is_ok());
}

// ============================================================================
// EIP-712 signature verification
// ============================================================================

fn type_hash() -> [u8; 32] {
    // Any fixed 32 bytes stand in for the real type string hash.
    [0x11u8; 32]
}

fn domain_separator() -> [u8; 32] {
    [0x22u8; 32]
}

/// Signs the EIP-712 digest of `intent` with `key` and returns the 65-byte
/// signature in hex.
fn sign_intent(key: &k256::ecdsa::SigningKey, intent: &Intent) -> String {
    use k256::ecdsa::signature::hazmat::PrehashSigner;
    use sha3::{Digest, Keccak256};

    let struct_hash = intent_struct_hash(intent, &type_hash()).unwrap();
    let mut preimage = Vec::new();
    preimage.extend_from_slice(&[0x19, 0x01]);
    preimage.extend_from_slice(&domain_separator());
    preimage.extend_from_slice(&struct_hash);
    let digest = Keccak256::digest(&preimage);

    let (sig, recid): (k256::ecdsa::Signature, k256::ecdsa::RecoveryId) =
        key.sign_prehash(&digest).unwrap();

    let mut out = sig.to_bytes().to_vec();
    out.push(recid.to_byte());
    format!("0x{}", hex::encode(out))
}

fn address_of(key: &k256::ecdsa::SigningKey) -> String {
    use sha3::{Digest, Keccak256};
    let point = key.verifying_key().to_encoded_point(false);
    let hash = Keccak256::digest(&point.as_bytes()[1..]);
    format!("0x{}", hex::encode(&hash[12..]))
}

fn signing_key() -> k256::ecdsa::SigningKey {
    // A fixed key so the test is deterministic.
    k256::ecdsa::SigningKey::from_slice(&[0x42u8; 32]).unwrap()
}

#[test]
fn verify_signature_accepts_a_real_signature() {
    let key = signing_key();
    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.user = address_of(&key);
    i.signature = sign_intent(&key, &i);

    assert!(IntentParser::verify_signature(&i, &i.signature, &domain_separator(), &type_hash()).is_ok());
}

#[test]
fn verify_signature_rejects_another_keys_signature() {
    let key = signing_key();
    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.user = address_of(&key);
    let signed = sign_intent(&key, &i);

    // Claim the intent came from a different address.
    let mut other = i.clone();
    other.user = "0x2222222222222222222222222222222222222222".to_string();

    let err = IntentParser::verify_signature(&other, &signed, &domain_separator(), &type_hash());
    assert!(err.is_err(), "a signature must not verify for another user");
}

#[test]
fn verify_signature_rejects_arbitrary_text() {
    // The old implementation accepted any non-empty string.
    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.user = "0x1111111111111111111111111111111111111111".to_string();

    for bad in ["", "test_sig", "0xdeadbeef", "not-hex"] {
        assert!(
            IntentParser::verify_signature(&i, bad, &domain_separator(), &type_hash()).is_err(),
            "{:?} should not verify",
            bad
        );
    }
}

#[test]
fn verify_signature_rejects_tampered_terms() {
    let key = signing_key();
    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.user = address_of(&key);
    let signed = sign_intent(&key, &i);

    // Change the amount after signing.
    let mut tampered = i.clone();
    tampered.sell = Some(amount(Chain::Ethereum, "USDC", 6, 5000.0));

    assert!(IntentParser::verify_signature(&tampered, &signed, &domain_separator(), &type_hash()).is_err());
}

#[test]
fn verify_signature_rejects_a_wrong_domain() {
    let key = signing_key();
    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.user = address_of(&key);
    i.signature = sign_intent(&key, &i);

    let other_domain = [0x33u8; 32];
    assert!(IntentParser::verify_signature(&i, &i.signature, &other_domain, &type_hash()).is_err());
}

#[test]
fn verify_signature_rejects_a_non_address_user() {
    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.user = "not-an-address".to_string();
    assert!(IntentParser::verify_signature(&i, "0x00", &domain_separator(), &type_hash()).is_err());
}

// ============================================================================
// Hashing
// ============================================================================

#[test]
fn hash_intent_is_deterministic_and_excludes_the_signature() {
    let mut a = intent(Chain::Ethereum, IntentType::Swap);
    a.signature = "0xaaa".to_string();
    let mut b = a.clone();
    b.signature = "0xbbb".to_string();

    assert_eq!(hash_intent(&a), hash_intent(&b), "the signature must not affect the id");

    let h = hash_intent(&a);
    assert!(h.starts_with("0x"));
    assert_eq!(h.len(), 66, "keccak-256 is 32 bytes");
    assert!(is_tx_hash(&h));
}

#[test]
fn hash_intent_changes_with_the_terms() {
    let a = intent(Chain::Ethereum, IntentType::Swap);
    let mut b = a.clone();
    b.nonce = 99;
    assert_ne!(hash_intent(&a), hash_intent(&b));

    let mut c = a.clone();
    c.slippage_bps = 500;
    assert_ne!(hash_intent(&a), hash_intent(&c));
}

// ============================================================================
// Quote engine
// ============================================================================

fn engine_with_prices(sell_price: f64, buy_price: f64) -> QuoteEngine {
    let engine = QuoteEngine::new();
    engine.update_price(Chain::Ethereum, "0x1111111111111111111111111111111111111111".to_string(), sell_price);
    engine.update_price(Chain::Ethereum, "0x1111111111111111111111111111111111111111".to_string(), buy_price);
    engine
}

#[test]
fn quote_rejects_unknown_tokens() {
    let engine = QuoteEngine::new();
    let i = intent(Chain::Ethereum, IntentType::Swap);
    assert!(engine.get_quote(&i).is_err());
}

#[test]
fn quote_rejects_non_positive_prices() {
    let engine = QuoteEngine::new();
    engine.update_price(Chain::Ethereum, "0x1111111111111111111111111111111111111111".to_string(), 0.0);
    let i = intent(Chain::Ethereum, IntentType::Swap);
    assert!(engine.get_quote(&i).is_err());
}

#[test]
fn cross_chain_quote_rejects_an_unknown_bridge() {
    let engine = QuoteEngine::new();
    let i = intent(Chain::Ethereum, IntentType::CrossChainSwap);
    let err = engine.get_cross_chain_quote(&i, Chain::Base, "not-a-bridge");
    assert!(err.is_err(), "an unknown bridge must not be given a made-up time");
}

#[test]
fn cross_chain_quote_reports_zero_cost_without_a_price() {
    let engine = QuoteEngine::new();
    engine.update_price(Chain::Ethereum, "0x1111111111111111111111111111111111111111".to_string(), 1.0);

    let mut i = intent(Chain::Ethereum, IntentType::CrossChainSwap);
    i.buy = Some(amount(Chain::Ethereum, "WBTC", 8, 0.015));

    let quote = engine.get_cross_chain_quote(&i, Chain::Base, "layerzero").unwrap();
    // No price was supplied for Base's native token, so the cost is unknown
    // and reported as zero rather than an assumed $3000.
    assert_eq!(quote.total_gas_usd, 0.0);
    assert_eq!(quote.bridge_time_seconds, 900);
}

// ============================================================================
// Gas estimates
// ============================================================================

#[test]
fn gas_estimate_is_returned_for_every_intent_type() {
    let engine = QuoteEngine::new();
    for kind in [
        IntentType::Swap,
        IntentType::CrossChainSwap,
        IntentType::LimitOrder,
        IntentType::DCA,
        IntentType::TWAP,
    ] {
        let i = intent(Chain::Ethereum, kind);
        assert!(
            engine.estimate_gas(&i).is_some(),
            "{:?} should have an estimate",
            kind
        );
    }
}

// ============================================================================
// Solver network
// ============================================================================

fn solver(id: &str, chains: Vec<Chain>, reputation: u64, active: bool) -> Solver {
    Solver {
        id: id.to_string(),
        address: format!("0x{}", "aa".repeat(20)),
        chains,
        reputation,
        total_fills: 10,
        success_rate: 0.95,
        avg_fill_time_ms: 1200,
        active,
    }
}

fn network() -> SolverNetwork {
    SolverNetwork::new(Arc::new(QuoteEngine::new()))
}

#[test]
fn submit_intent_rejects_a_duplicate_id() {
    let net = network();
    let i = intent(Chain::Ethereum, IntentType::Swap);
    net.submit_intent(i.clone()).unwrap();
    assert!(
        net.submit_intent(i).is_err(),
        "a duplicate id must not overwrite the stored intent"
    );
}

#[test]
fn submit_intent_rejects_an_empty_id() {
    let net = network();
    let mut i = intent(Chain::Ethereum, IntentType::Swap);
    i.id = "  ".to_string();
    assert!(net.submit_intent(i).is_err());
}

#[test]
fn find_best_solver_prefers_reputation() {
    let net = network();
    net.register_solver(solver("low", vec![Chain::Ethereum], 100, true));
    net.register_solver(solver("high", vec![Chain::Ethereum], 900, true));

    let i = intent(Chain::Ethereum, IntentType::Swap);
    let best = net.find_best_solver(&i).unwrap();
    assert_eq!(best.id, "high");
}

#[test]
fn find_best_solver_skips_ineligible_solvers() {
    let net = network();
    net.register_solver(solver("wrong-chain", vec![Chain::Base], 900, true));
    net.register_solver(solver("inactive", vec![Chain::Ethereum], 900, false));
    net.register_solver(solver("unreputed", vec![Chain::Ethereum], 50, true));

    let i = intent(Chain::Ethereum, IntentType::Swap);
    assert!(
        net.find_best_solver(&i).is_none(),
        "no solver here can fill this intent"
    );
}

#[test]
fn record_fill_requires_a_real_transaction_hash() {
    let net = network();
    net.submit_intent(intent(Chain::Ethereum, IntentType::Swap)).unwrap();

    // The old execute_intent stored 32 zero bytes as the hash.
    let zero_hash = format!("0x{}", "00".repeat(32));
    assert!(net.record_fill("intent-1", "solver-1", &zero_hash, 21000).is_err());

    for bad in ["", "0xdeadbeef", "not-hex"] {
        assert!(
            net.record_fill("intent-1", "solver-1", bad, 21000).is_err(),
            "{:?} is not a hash",
            bad
        );
    }

    let real = format!("0x{}", "ab".repeat(32));
    let fill = net.record_fill("intent-1", "solver-1", &real, 21000).unwrap();
    assert_eq!(fill.tx_hash, real);
    assert_eq!(fill.gas_used, 21000);
    assert_eq!(
        net.get_intent("intent-1").unwrap().status,
        IntentStatus::Filled
    );
}

#[test]
fn record_fill_rejects_a_replayed_transaction_hash() {
    let net = network();
    net.submit_intent(intent(Chain::Ethereum, IntentType::Swap)).unwrap();
    let real = format!("0x{}", "cd".repeat(32));

    net.record_fill("intent-1", "solver-1", &real, 21000).unwrap();
    assert!(
        net.record_fill("intent-1", "solver-1", &real, 21000).is_err(),
        "the same transaction must not be recorded twice"
    );
}

#[test]
fn record_fill_rejects_an_unknown_intent() {
    let net = network();
    let real = format!("0x{}", "ee".repeat(32));
    assert!(net.record_fill("nope", "solver-1", &real, 21000).is_err());
}

#[test]
fn record_fill_rejects_a_cancelled_intent() {
    let net = network();
    net.submit_intent(intent(Chain::Ethereum, IntentType::Swap)).unwrap();
    net.cancel_intent("intent-1").unwrap();

    let real = format!("0x{}", "ff".repeat(32));
    assert!(net.record_fill("intent-1", "solver-1", &real, 21000).is_err());
}

#[test]
fn cancel_intent_only_works_while_pending() {
    let net = network();
    net.submit_intent(intent(Chain::Ethereum, IntentType::Swap)).unwrap();
    net.cancel_intent("intent-1").unwrap();
    assert_eq!(
        net.get_intent("intent-1").unwrap().status,
        IntentStatus::Cancelled
    );

    // A second cancel must fail, and an unknown intent must fail.
    assert!(net.cancel_intent("intent-1").is_err());
    assert!(net.cancel_intent("missing").is_err());
}

#[test]
fn get_fills_returns_only_the_matching_intent() {
    let net = network();
    net.submit_intent(intent(Chain::Ethereum, IntentType::Swap)).unwrap();

    let mut second = intent(Chain::Ethereum, IntentType::Swap);
    second.id = "intent-2".to_string();
    net.submit_intent(second).unwrap();

    net.record_fill("intent-1", "solver-1", &format!("0x{}", "11".repeat(32)), 21000)
        .unwrap();
    net.record_fill("intent-2", "solver-2", &format!("0x{}", "22".repeat(32)), 30000)
        .unwrap();

    let fills = net.get_fills("intent-1");
    assert_eq!(fills.len(), 1);
    assert_eq!(fills[0].solver, "solver-1");
}

// ============================================================================
// Disputes
// ============================================================================

#[test]
fn process_dispute_marks_the_intent_and_requires_a_reason() {
    let net = network();
    net.submit_intent(intent(Chain::Ethereum, IntentType::Swap)).unwrap();

    // A dispute must state a reason; the old version accepted anything.
    assert!(net.process_dispute("intent-1", "   ").is_err());
    assert!(net.process_dispute("missing", "reason").is_err());

    net.process_dispute("intent-1", "fill did not match the quote").unwrap();
    assert_eq!(
        net.get_intent("intent-1").unwrap().status,
        IntentStatus::Disputed
    );

    // The reason is retained rather than printed and discarded.
    let disputes = net.get_disputes("intent-1");
    assert_eq!(disputes.len(), 1);
    assert!(disputes[0].reason.contains("did not match"));

    // A disputed intent cannot be filled, and cannot be disputed twice.
    assert!(net.process_dispute("intent-1", "again").is_err());
    let real = format!("0x{}", "33".repeat(32));
    assert!(net.record_fill("intent-1", "solver-1", &real, 21000).is_err());
}

// ============================================================================
// Settlement
// ============================================================================

#[test]
fn settle_cross_chain_validates_hashes() {
    let intents = Arc::new(RwLock::new(HashMap::new()));
    let fills = Arc::new(RwLock::new(Vec::new()));
    let engine = SettlementEngine::new(Arc::clone(&intents), Arc::clone(&fills));

    // No fill recorded yet.
    assert!(engine
        .settle_cross_chain("intent-1", &format!("0x{}", "11".repeat(32)), &format!("0x{}", "22".repeat(32)))
        .is_err());

    intents.write().unwrap().insert(
        "intent-1".to_string(),
        intent(Chain::Ethereum, IntentType::CrossChainSwap),
    );
    fills.write().unwrap().push(Fill {
        intent_id: "intent-1".to_string(),
        solver: "solver-1".to_string(),
        fill_amount: u256::from(1u64),
        fill_price: u256::from(1u64),
        fill_price_usd: 1.0,
        gas_used: 21000,
        tx_hash: format!("0x{}", "11".repeat(32)),
        created_at: now_secs(),
    });

    // Malformed hashes are rejected.
    assert!(engine.settle_cross_chain("intent-1", "", "").is_err());
    assert!(engine.settle_cross_chain("intent-1", "0xabc", "0xdef").is_err());

    // Well-formed hashes settle.
    engine
        .settle_cross_chain(
            "intent-1",
            &format!("0x{}", "11".repeat(32)),
            &format!("0x{}", "22".repeat(32)),
        )
        .unwrap();
    assert_eq!(
        intents.read().unwrap().get("intent-1").unwrap().status,
        IntentStatus::Filled
    );
}

// ============================================================================
// Price improvement engine
// ============================================================================

#[test]
fn optimal_price_walks_the_book() {
    let engine = PriceImprovementEngine::new();
    engine.update_dex_prices(
        Chain::Ethereum,
        "0xtoken".to_string(),
        vec![
            PriceLevel { price: 1.0, liquidity: 100.0 },
            PriceLevel { price: 2.0, liquidity: 100.0 },
        ],
    );

    // Filling 50 units uses only the first level.
    let p = engine
        .calculate_optimal_price(Chain::Ethereum, "0xtoken", u256::from(50u64), Side::Buy)
        .unwrap();
    assert!((p - 1.0).abs() < 1e-9);

    // Filling 150 units uses both: (100*1 + 50*2) / 150.
    let p = engine
        .calculate_optimal_price(Chain::Ethereum, "0xtoken", u256::from(150u64), Side::Buy)
        .unwrap();
    assert!((p - 200.0 / 150.0).abs() < 1e-9);
}

#[test]
fn optimal_price_reports_insufficient_liquidity() {
    let engine = PriceImprovementEngine::new();
    engine.update_dex_prices(
        Chain::Ethereum,
        "0xtoken".to_string(),
        vec![PriceLevel { price: 1.0, liquidity: 10.0 }],
    );

    let err = engine.calculate_optimal_price(Chain::Ethereum, "0xtoken", u256::from(1000u64), Side::Buy);
    assert!(err.is_err());
    assert!(err.unwrap_err().contains("insufficient liquidity"));
}

#[test]
fn optimal_price_rejects_zero_and_unknown_tokens() {
    let engine = PriceImprovementEngine::new();
    assert!(engine
        .calculate_optimal_price(Chain::Ethereum, "0xtoken", u256::zero(), Side::Buy)
        .is_err());
    assert!(engine
        .calculate_optimal_price(Chain::Ethereum, "0xunknown", u256::from(1u64), Side::Buy)
        .is_err());
}

#[test]
fn sell_side_walks_the_book_downwards() {
    let engine = PriceImprovementEngine::new();
    engine.update_dex_prices(
        Chain::Ethereum,
        "0xtoken".to_string(),
        vec![
            PriceLevel { price: 1.0, liquidity: 100.0 },
            PriceLevel { price: 2.0, liquidity: 100.0 },
        ],
    );

    // Selling 50 hits the best (highest) bid first.
    let p = engine
        .calculate_optimal_price(Chain::Ethereum, "0xtoken", u256::from(50u64), Side::Sell)
        .unwrap();
    assert!((p - 2.0).abs() < 1e-9, "sell should start at the highest bid");
}

#[test]
fn improvement_is_measured_in_basis_points() {
    let engine = PriceImprovementEngine::new();
    engine.update_dex_prices(
        Chain::Ethereum,
        "0xtoken".to_string(),
        vec![PriceLevel { price: 100.0, liquidity: 1_000_000.0 }],
    );

    // Filling at 99 against a market of 100 is 100 bps of improvement.
    let bps = engine.find_improvement(Chain::Ethereum, "0xtoken", u256::from(10u64), 99.0);
    assert!((bps - 100.0).abs() < 1e-6, "got {}", bps);

    // Filling at 101 is a 100 bps loss.
    let bps = engine.find_improvement(Chain::Ethereum, "0xtoken", u256::from(10u64), 101.0);
    assert!((bps + 100.0).abs() < 1e-6, "got {}", bps);
}

// ============================================================================
// Helpers
// ============================================================================

#[test]
fn tx_hash_validation() {
    assert!(is_tx_hash(&format!("0x{}", "ab".repeat(32))));
    assert!(is_tx_hash(&format!("{}", "AB".repeat(32))));
    assert!(!is_tx_hash(&format!("0x{}", "ab".repeat(31))));
    assert!(!is_tx_hash("0x"));
    assert!(!is_tx_hash("not-a-hash"));
}

#[test]
fn address_parsing() {
    assert!(parse_address("0x1111111111111111111111111111111111111111").is_some());
    assert!(parse_address("0x1111").is_none());
    assert!(parse_address("1111111111111111111111111111111111111111").is_none());
    assert!(parse_address("0xzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz").is_none());
}

#[test]
fn chain_ids_match_the_networks() {
    assert_eq!(Chain::Ethereum.chain_id(), 1);
    assert_eq!(Chain::Polygon.chain_id(), 137);
    assert_eq!(Chain::Arbitrum.chain_id(), 42161);
    assert_eq!(Chain::Optimism.chain_id(), 10);
    assert_eq!(Chain::Avalanche.chain_id(), 43114);
    assert_eq!(Chain::BSC.chain_id(), 56);
    assert_eq!(Chain::Base.chain_id(), 8453);
}

#[test]
fn struct_hash_is_stable_and_term_sensitive() {
    let a = intent(Chain::Ethereum, IntentType::Swap);
    let h1 = intent_struct_hash(&a, &type_hash()).unwrap();
    let h2 = intent_struct_hash(&a, &type_hash()).unwrap();
    assert_eq!(h1, h2, "the struct hash must be deterministic");

    let mut b = a.clone();
    b.nonce = 2;
    assert_ne!(h1, intent_struct_hash(&b, &type_hash()).unwrap());
}

#[test]
fn struct_hash_requires_a_valid_user_address() {
    let mut a = intent(Chain::Ethereum, IntentType::Swap);
    a.user = "not-an-address".to_string();
    assert!(intent_struct_hash(&a, &type_hash()).is_err());
}
