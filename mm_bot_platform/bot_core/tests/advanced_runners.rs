// Tests for the advanced execution runners.
//
// The expression sandbox is the security-critical part of this module: an
// operator supplies the string, so the tests concentrate on what it must
// REFUSE. The prediction helpers are pinned because a wrong value there becomes
// a wrong order on a real exchange.

use std::collections::HashMap;

use tigerswap_bot_core::strategies::advanced::{
    eval_expression, parse_expression, AiAction, AiPrediction,
};

fn env(pairs: &[(&str, f64)]) -> HashMap<String, f64> {
    pairs.iter().map(|(k, v)| (k.to_string(), *v)).collect()
}

#[test]
fn arithmetic_respects_precedence() {
    let e = parse_expression("2 + 3 * 4").unwrap();
    assert_eq!(eval_expression(&e, &env(&[])).unwrap(), 14.0);

    let e = parse_expression("(2 + 3) * 4").unwrap();
    assert_eq!(eval_expression(&e, &env(&[])).unwrap(), 20.0);
}

#[test]
fn exponentiation_is_right_associative() {
    // 2^3^2 must be 2^(3^2) = 512, not (2^3)^2 = 64.
    let e = parse_expression("2 ^ 3 ^ 2").unwrap();
    assert_eq!(eval_expression(&e, &env(&[])).unwrap(), 512.0);
}

#[test]
fn unary_minus_evaluates() {
    let e = parse_expression("-2 + 5").unwrap();
    assert_eq!(eval_expression(&e, &env(&[])).unwrap(), 3.0);
}

#[test]
fn variables_resolve_from_the_environment() {
    let e = parse_expression("price - price_mean").unwrap();
    let v = eval_expression(&e, &env(&[("price", 105.0), ("price_mean", 100.0)])).unwrap();
    assert_eq!(v, 5.0);
}

#[test]
fn comparisons_yield_one_or_zero_for_gating() {
    // An expression must be able to gate an order without a conditional.
    let e = parse_expression("(price > price_mean) * 1").unwrap();
    assert_eq!(
        eval_expression(&e, &env(&[("price", 105.0), ("price_mean", 100.0)])).unwrap(),
        1.0
    );
    assert_eq!(
        eval_expression(&e, &env(&[("price", 95.0), ("price_mean", 100.0)])).unwrap(),
        0.0
    );
}

#[test]
fn allowlisted_functions_evaluate() {
    let e = parse_expression("abs(0 - 7) + min(3, 1) + max(3, 1)").unwrap();
    assert_eq!(eval_expression(&e, &env(&[])).unwrap(), 7.0 + 1.0 + 3.0);

    let e = parse_expression("sqrt(16)").unwrap();
    assert_eq!(eval_expression(&e, &env(&[])).unwrap(), 4.0);

    let e = parse_expression("clamp(15, 0, 10)").unwrap();
    assert_eq!(eval_expression(&e, &env(&[])).unwrap(), 10.0);
}

#[test]
fn unknown_variable_is_rejected() {
    let e = parse_expression("mystery + 1").unwrap();
    let err = eval_expression(&e, &env(&[])).unwrap_err();
    assert!(err.contains("unknown variable"), "got: {err}");
}

#[test]
fn unknown_function_is_rejected() {
    // The allowlist must reject anything not deliberately exposed. This is the
    // property that keeps an operator string from reaching the host.
    let e = parse_expression("system(1)").unwrap();
    let err = eval_expression(&e, &env(&[])).unwrap_err();
    assert!(err.contains("unknown function"), "got: {err}");
}

#[test]
fn division_by_zero_is_an_error_not_an_infinity() {
    let e = parse_expression("1 / 0").unwrap();
    let err = eval_expression(&e, &env(&[])).unwrap_err();
    assert!(err.contains("division by zero"), "got: {err}");

    let e = parse_expression("1 % 0").unwrap();
    let err = eval_expression(&e, &env(&[])).unwrap_err();
    assert!(err.contains("modulo by zero"), "got: {err}");
}

#[test]
fn negative_sqrt_and_nonpositive_ln_are_rejected() {
    let e = parse_expression("sqrt(0 - 1)").unwrap();
    assert!(eval_expression(&e, &env(&[])).unwrap_err().contains("sqrt"));

    let e = parse_expression("ln(0)").unwrap();
    assert!(eval_expression(&e, &env(&[])).unwrap_err().contains("ln"));
}

#[test]
fn wrong_arity_is_rejected() {
    let e = parse_expression("min(1)").unwrap();
    let err = eval_expression(&e, &env(&[])).unwrap_err();
    assert!(err.contains("takes 2 argument"), "got: {err}");
}

#[test]
fn malformed_expressions_are_rejected_at_parse_time() {
    for bad in [
        "1 +",
        "(1 + 2",
        "1 + 2)",
        "1 2",
        "* 3",
        "min(1,)",
        "1 & 2",
        "price +",
        "",
        "1;2",
        "1 + 2; rm -rf /",
    ] {
        assert!(
            parse_expression(bad).is_err(),
            "expression {bad:?} should have been rejected"
        );
    }
}

#[test]
fn expression_cannot_reach_the_host() {
    // Strings that would be dangerous if the input were evaluated as code must
    // fail to parse: only numbers, identifiers, operators and the allowlisted
    // call syntax are accepted.
    for hostile in [
        "std::process::exit(0)",
        "1.to_string()",
        "price.await",
        "loop {}",
        "fn main() {}",
        "|x| x",
        "$(whoami)",
        "`whoami`",
        "1; drop table bots",
        "let x = 1",
        "price[0]",
        "price.field",
    ] {
        assert!(
            parse_expression(hostile).is_err(),
            "hostile expression {hostile:?} should have been rejected at parse time"
        );
    }
}

#[test]
fn host_language_call_syntax_parses_but_cannot_execute() {
    // A name followed by parentheses is valid call syntax, so these parse. They
    // must then fail at evaluation, because only the allowlisted functions are
    // implemented. This is the second half of the sandbox: parsing cannot be
    // the only gate.
    for hostile in [
        "eval(1)",
        "exec(1)",
        "__import__(1)",
        "system(1)",
        "process_exit(1)",
        "include(1)",
        "open(1)",
    ] {
        let parsed = parse_expression(hostile)
            .unwrap_or_else(|e| panic!("{hostile:?} should parse as a call: {e}"));
        let err = eval_expression(&parsed, &env(&[]))
            .expect_err(&format!("hostile call {hostile:?} must not evaluate"));
        assert!(
            err.contains("unknown function"),
            "hostile call {hostile:?} gave an unexpected error: {err}"
        );
    }
}

#[test]
fn non_finite_literals_are_rejected() {
    // A literal that cannot be represented is refused rather than producing an
    // infinity that would become an order size.
    assert!(parse_expression("1e400").is_err());
}

#[test]
fn ai_prediction_requires_magnitude_and_confidence() {
    let confident_bull = AiPrediction {
        predicted_change_pct: 0.02,
        confidence: 0.9,
        direction: None,
    };
    assert_eq!(
        confident_bull.signal(0.005, 0.6),
        Some(AiAction::Buy),
        "a large confident upward move must buy"
    );

    let unsure_bull = AiPrediction {
        predicted_change_pct: 0.02,
        confidence: 0.3,
        direction: None,
    };
    assert_eq!(
        unsure_bull.signal(0.005, 0.6),
        None,
        "a large move the model is unsure about is not a signal"
    );

    let small_move = AiPrediction {
        predicted_change_pct: 0.001,
        confidence: 0.99,
        direction: None,
    };
    assert_eq!(
        small_move.signal(0.005, 0.6),
        None,
        "a confident but tiny move is not worth the fee"
    );

    let bearish = AiPrediction {
        predicted_change_pct: -0.02,
        confidence: 0.9,
        direction: None,
    };
    assert_eq!(bearish.signal(0.005, 0.6), Some(AiAction::Sell));

    let explicit = AiPrediction {
        predicted_change_pct: 0.0,
        confidence: 0.9,
        direction: Some("down".to_string()),
    };
    assert_eq!(
        explicit.signal(0.005, 0.6),
        Some(AiAction::Sell),
        "an explicit direction is honoured"
    );

    let neutral = AiPrediction {
        predicted_change_pct: 0.0,
        confidence: 0.9,
        direction: Some("flat".to_string()),
    };
    assert_eq!(neutral.signal(0.005, 0.6), None);
}

#[test]
fn ai_prediction_decodes_the_service_payload() {
    // The service returns a plain object; the default confidence must be zero
    // (not one) so a payload that omits it cannot cause a trade.
    let p: AiPrediction = serde_json::from_str(r#"{"predicted_change_pct":0.01}"#).unwrap();
    assert_eq!(p.confidence, 0.0);
    assert_eq!(p.signal(0.001, 0.5), None);

    // The alternative field name is accepted.
    let p: AiPrediction = serde_json::from_str(r#"{"change_pct":-0.03,"confidence":0.8}"#).unwrap();
    assert_eq!(p.signal(0.005, 0.6), Some(AiAction::Sell));
}
