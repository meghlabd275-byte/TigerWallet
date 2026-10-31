"""Tests for the market intelligence module.

Run with:  python3 market_intelligence/python/tests/test_market_intel.py
"""

import math
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from market_intel import (  # noqa: E402
    MarketData,
    MarketIntelligence,
    PortfolioAnalyzer,
)


def series(prices, start_ts=1_700_000_000, volume=1000.0):
    return [
        MarketData(
            timestamp=start_ts + i * 3600,
            open=p,
            high=p * 1.01,
            low=p * 0.99,
            close=p,
            volume=volume,
        )
        for i, p in enumerate(prices)
    ]


def load(engine, token, prices):
    for point in series(prices):
        engine.add_price_data(token, point)


def trending_up(n=120, start=100.0, step=0.5):
    return [start + step * i for i in range(n)]


# --- Refusal to fabricate --------------------------------------------------


def test_predict_returns_none_without_data():
    engine = MarketIntelligence()
    assert engine.predict_price("BTC") is None


def test_predict_returns_none_below_minimum():
    engine = MarketIntelligence()
    load(engine, "BTC", trending_up(n=20))
    assert engine.predict_price("BTC") is None


def test_predict_rejects_unknown_timeframe():
    engine = MarketIntelligence()
    load(engine, "BTC", trending_up(n=120))
    try:
        engine.predict_price("BTC", timeframe="3s")
    except ValueError:
        pass
    else:
        raise AssertionError("unknown timeframe must be rejected")


# --- Determinism -----------------------------------------------------------


def test_prediction_is_deterministic():
    # The old implementation drew the move from np.random.uniform, so identical
    # input produced a different price on every call.
    prices = trending_up(n=120)
    a = MarketIntelligence()
    load(a, "BTC", prices)
    b = MarketIntelligence()
    load(b, "BTC", prices)

    pa = a.predict_price("BTC")
    pb = b.predict_price("BTC")
    assert pa.predicted_price == pb.predicted_price
    assert pa.confidence == pb.confidence
    assert pa.direction == pb.direction


def test_repeated_call_on_one_engine_is_stable():
    engine = MarketIntelligence()
    load(engine, "BTC", trending_up(n=120))
    first = engine.predict_price("BTC")
    second = engine.predict_price("BTC")
    assert first.predicted_price == second.predicted_price


# --- The model reflects the data -------------------------------------------


def test_uptrend_predicts_up():
    engine = MarketIntelligence()
    load(engine, "BTC", trending_up(n=120, start=100.0, step=0.5))
    pred = engine.predict_price("BTC", "24h")
    assert pred.direction == "up", pred.direction
    assert pred.predicted_price > pred.current_price


def test_downtrend_predicts_down():
    engine = MarketIntelligence()
    load(engine, "BTC", trending_up(n=120, start=200.0, step=-0.5))
    pred = engine.predict_price("BTC", "24h")
    assert pred.direction == "down", pred.direction
    assert pred.predicted_price < pred.current_price


def test_flat_market_predicts_neutral():
    engine = MarketIntelligence()
    load(engine, "BTC", [100.0] * 120)
    pred = engine.predict_price("BTC", "24h")
    assert pred.direction == "neutral", pred.direction
    assert math.isclose(pred.predicted_price, 100.0, rel_tol=1e-6)


def test_longer_horizon_moves_further():
    engine = MarketIntelligence()
    load(engine, "BTC", trending_up(n=120, start=100.0, step=0.5))
    one_hour = engine.predict_price("BTC", "1h")
    one_week = engine.predict_price("BTC", "7d")
    assert one_week.predicted_price > one_hour.predicted_price


def test_volatility_lowers_confidence():
    calm = MarketIntelligence()
    load(calm, "BTC", trending_up(n=120, start=100.0, step=0.5))

    choppy_prices = []
    for i in range(120):
        choppy_prices.append(100.0 + (10.0 if i % 2 else -10.0))
    choppy = MarketIntelligence()
    load(choppy, "BTC", choppy_prices)

    assert choppy.predict_price("BTC").confidence < calm.predict_price("BTC").confidence


def test_prediction_is_capped():
    # A single extreme spike must not project an absurd price.
    prices = [100.0] * 119 + [100000.0]
    engine = MarketIntelligence()
    load(engine, "BTC", prices)
    pred = engine.predict_price("BTC", "24h")
    assert pred.predicted_price < pred.current_price * 10, pred.predicted_price


def test_confidence_in_range():
    engine = MarketIntelligence()
    load(engine, "BTC", trending_up(n=120))
    for tf in ("1h", "4h", "24h", "7d"):
        c = engine.predict_price("BTC", tf).confidence
        assert 0.0 < c <= 0.95, (tf, c)


# --- Sentiment and whales --------------------------------------------------


def test_sentiment_from_counts():
    engine = MarketIntelligence()
    s = engine.analyze_sentiment("BTC", {"bullish": 80, "bearish": 20, "mentions": 50})
    assert math.isclose(s.sentiment_score, 0.6)
    assert s.bullish_percentage == 80.0
    assert s.bearish_percentage == 20.0
    assert not s.trending


def test_sentiment_with_no_data_is_neutral():
    engine = MarketIntelligence()
    s = engine.analyze_sentiment("BTC", {})
    assert s.sentiment_score == 0.0
    assert s.bullish_percentage == 50.0
    assert s.bearish_percentage == 50.0


def test_whale_threshold_is_respected():
    engine = MarketIntelligence(whale_threshold_usd=100_000)
    txs = [
        {"from": "a", "to": "b", "amount": 5, "value_usd": 250_000, "type": "buy"},
        {"from": "c", "to": "d", "amount": 1, "value_usd": 5_000, "type": "sell"},
    ]
    alerts = engine.detect_whale_activity("BTC", txs)
    assert len(alerts) == 1
    assert alerts[0].value_usd == 250_000
    assert alerts[0].from_address == "a"


# --- Market metrics --------------------------------------------------------


def test_market_metrics_are_derived():
    engine = MarketIntelligence()
    load(engine, "BTC", trending_up(n=60, start=100.0, step=1.0))
    m = engine.get_market_metrics("BTC")
    assert m["current_price"] == 159.0
    assert m["high_24h"] == 159.0
    assert m["change_1h"] > 0
    assert m["change_24h"] > 0
    assert m["volatility"] >= 0


def test_market_metrics_empty_for_unknown_token():
    assert MarketIntelligence().get_market_metrics("NOPE") == {}


# --- Arbitrage -------------------------------------------------------------


def test_arbitrage_detects_real_spread():
    engine = MarketIntelligence()
    out = engine.find_arbitrage_opportunities(
        {"BTC": {"binance": 100.0, "kraken": 105.0}}, trade_size=1000
    )
    assert len(out) == 1
    assert out[0]["buy_exchange"] == "binance"
    assert out[0]["sell_exchange"] == "kraken"
    assert math.isclose(out[0]["gross_profit_percentage"], 5.0)


def test_arbitrage_net_of_fees():
    engine = MarketIntelligence()
    # 0.6% gross spread disappears once 0.5% is paid on each leg.
    out = engine.find_arbitrage_opportunities(
        {"BTC": {"binance": 100.0, "kraken": 100.6}},
        fee_pct_per_leg=0.5,
    )
    assert out == [], "fees must be able to eliminate an opportunity"


def test_arbitrage_ignores_tiny_spread():
    engine = MarketIntelligence()
    out = engine.find_arbitrage_opportunities({"BTC": {"a": 100.0, "b": 100.1}})
    assert out == []


def test_arbitrage_needs_two_exchanges():
    engine = MarketIntelligence()
    assert engine.find_arbitrage_opportunities({"BTC": {"binance": 100.0}}) == []


# --- Portfolio -------------------------------------------------------------


def test_pnl_requires_prices():
    analyzer = PortfolioAnalyzer()
    analyzer.add_transaction({"token": "BTC", "type": "buy", "amount": 1, "price": 50})
    # Omitting the argument entirely is a TypeError; passing an empty mapping
    # must be rejected explicitly rather than silently valuing at zero.
    for call in (
        lambda: analyzer.calculate_pnl({"BTC": 1}),
        lambda: analyzer.calculate_pnl({"BTC": 1}, {}),
    ):
        try:
            call()
        except (TypeError, ValueError):
            pass
        else:
            raise AssertionError("calculate_pnl must not value a portfolio without prices")


def test_pnl_uses_real_prices():
    analyzer = PortfolioAnalyzer()
    analyzer.add_transaction({"token": "BTC", "type": "buy", "amount": 2, "price": 100})
    out = analyzer.calculate_pnl({"BTC": 2}, {"BTC": 150})
    assert out["total_cost"] == 200
    assert out["total_value"] == 300
    assert out["pnl"] == 100
    assert out["pnl_percentage"] == 50.0
    assert out["unpriced_tokens"] == []


def test_pnl_reports_unpriced_tokens():
    analyzer = PortfolioAnalyzer()
    analyzer.add_transaction({"token": "BTC", "type": "buy", "amount": 1, "price": 100})
    out = analyzer.calculate_pnl({"BTC": 1, "SHIB": 1000}, {"BTC": 200})
    assert "SHIB" in out["unpriced_tokens"]
    assert out["total_value"] == 200, "unpriced holdings must not inflate the total"


def test_diversification_score_bounds():
    analyzer = PortfolioAnalyzer()
    assert analyzer.get_diversification_score({}) == 0.0
    single = analyzer.get_diversification_score({"BTC": 1})
    spread = analyzer.get_diversification_score(
        {"BTC": 1, "ETH": 1, "SOL": 1, "ADA": 1}
    )
    assert single == 0.0, "one asset is fully concentrated"
    assert spread > single
    assert 0.0 <= spread <= 100.0


def test_rebalancing_uses_market_value():
    analyzer = PortfolioAnalyzer()
    # 100% in BTC; target 50/50. Sell half the BTC value.
    out = analyzer.suggest_rebalancing(
        {"BTC": 1.0}, {"BTC": 1000.0, "ETH": 100.0}, {"BTC": 50.0, "ETH": 50.0}
    )
    assert len(out) == 2
    btc = next(s for s in out if s["token"] == "BTC")
    eth = next(s for s in out if s["token"] == "ETH")
    assert btc["action"] == "sell"
    assert math.isclose(btc["amount"], 0.5), btc
    assert eth["action"] == "buy"
    assert math.isclose(eth["amount"], 5.0), eth


def test_rebalancing_rejects_bad_allocation():
    analyzer = PortfolioAnalyzer()
    try:
        analyzer.suggest_rebalancing({"BTC": 1.0}, {"BTC": 100.0}, {"BTC": 50.0})
    except ValueError:
        pass
    else:
        raise AssertionError("an allocation that does not sum to 100 must be rejected")


def test_rebalancing_skips_unpriced_tokens():
    analyzer = PortfolioAnalyzer()
    out = analyzer.suggest_rebalancing(
        {"BTC": 1.0}, {"BTC": 1000.0}, {"BTC": 50.0, "ETH": 50.0}
    )
    assert all(s["token"] != "ETH" for s in out)


if __name__ == "__main__":
    failures = 0
    for name, fn in sorted(list(globals().items())):
        if name.startswith("test_") and callable(fn):
            try:
                fn()
            except Exception as exc:  # noqa: BLE001
                failures += 1
                print(f"FAIL {name}: {exc}")
            else:
                print(f"ok   {name}")
    print()
    print("all tests passed" if failures == 0 else f"{failures} test(s) failed")
    sys.exit(1 if failures else 0)
