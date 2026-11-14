"""Tests for the AI prediction HTTP service.

Focus areas:
  * ingestion is gated and validated (an unauthenticated writer could poison
    the model's inputs, and a bad candle would corrupt the indicators);
  * the provider adapters parse each exchange's REAL response shape;
  * an unreachable provider fails closed — it never falls back to invented
    prices, because a trading bot acts on whatever this service returns.

The provider tests use recorded response shapes (captured from the live
endpoints) rather than mocks of our own code.
"""

import os

import pytest
from fastapi.testclient import TestClient

os.environ.setdefault("AI_INGEST_API_KEY", "test-key")

import ai_service  # noqa: E402
from ai_service import Candle, _parse_provider_rows, _split_symbol  # noqa: E402


@pytest.fixture()
def client():
    return TestClient(ai_service.app)


@pytest.fixture(autouse=True)
def _clean_engine():
    """Each test starts from an empty engine so history cannot leak between
    tests and make an insufficient-data case look like a success."""
    ai_service.engine.price_history.clear()
    ai_service.engine.predictions_cache.clear()
    yield
    ai_service.engine.price_history.clear()
    ai_service.engine.predictions_cache.clear()


def _candles(n=120, start=1700000000, price=50000.0):
    out = []
    px = price
    for i in range(n):
        px *= 1 + (0.002 if i % 3 == 0 else -0.001)
        out.append(
            {
                "timestamp": start + i * 3600,
                "open": px * 0.999,
                "high": px * 1.002,
                "low": px * 0.998,
                "close": px,
                "volume": 1000 + i,
            }
        )
    return out


# ---------------------------------------------------------------------------
# Health and gating
# ---------------------------------------------------------------------------


def test_health_reports_provider_and_gate(client):
    body = client.get("/health").json()
    assert body["status"] == "ok"
    assert body["ingest_enabled"] is True
    assert "candle_provider" in body


def test_ingest_requires_the_key(client):
    resp = client.post("/ingest", json={"symbol": "BTCUSDT", "candles": _candles(1)})
    assert resp.status_code == 401


def test_ingest_rejects_a_wrong_key(client):
    resp = client.post(
        "/ingest",
        headers={"X-API-Key": "wrong"},
        json={"symbol": "BTCUSDT", "candles": _candles(1)},
    )
    assert resp.status_code == 401


def test_ingest_fails_closed_when_no_key_is_configured(client, monkeypatch):
    monkeypatch.setattr(ai_service, "INGEST_API_KEY", "")
    resp = client.post(
        "/ingest",
        headers={"X-API-Key": "anything"},
        json={"symbol": "BTCUSDT", "candles": _candles(1)},
    )
    assert resp.status_code == 503


# ---------------------------------------------------------------------------
# Candle validation
# ---------------------------------------------------------------------------


def test_ingest_rejects_low_greater_than_high(client):
    bad = [{"timestamp": 1, "open": 1.0, "high": 1.0, "low": 2.0, "close": 1.0, "volume": 1.0}]
    resp = client.post(
        "/ingest", headers={"X-API-Key": "test-key"}, json={"symbol": "BTCUSDT", "candles": bad}
    )
    assert resp.status_code == 400
    assert "low > high" in resp.json()["detail"]


def test_ingest_rejects_negative_values(client):
    bad = [{"timestamp": 1, "open": -1.0, "high": 1.0, "low": -2.0, "close": 1.0, "volume": 1.0}]
    resp = client.post(
        "/ingest", headers={"X-API-Key": "test-key"}, json={"symbol": "BTCUSDT", "candles": bad}
    )
    assert resp.status_code == 400


def test_ingest_accepts_and_counts(client):
    resp = client.post(
        "/ingest",
        headers={"X-API-Key": "test-key"},
        json={"symbol": "btcusdt", "candles": _candles(40)},
    )
    assert resp.status_code == 200
    body = resp.json()
    # The symbol is normalised so "btcusdt" and "BTCUSDT" share one history.
    assert body["symbol"] == "BTCUSDT"
    assert body["accepted"] == 40
    assert body["history_length"] == 40


# ---------------------------------------------------------------------------
# Prediction
# ---------------------------------------------------------------------------


def test_predict_without_history_fails_closed(client):
    resp = client.get("/predict?symbol=BTCUSDT")
    assert resp.status_code == 422
    assert "insufficient history" in resp.json()["detail"]


def test_predict_below_the_minimum_history_fails_closed(client):
    # 29 candles is one short of what the indicators need; returning a neutral
    # prediction here would let a bot act on an absence of data.
    client.post(
        "/ingest", headers={"X-API-Key": "test-key"},
        json={"symbol": "BTCUSDT", "candles": _candles(29)},
    )
    resp = client.get("/predict?symbol=BTCUSDT")
    assert resp.status_code == 422


def test_predict_returns_a_real_prediction(client):
    client.post(
        "/ingest", headers={"X-API-Key": "test-key"},
        json={"symbol": "BTCUSDT", "candles": _candles(120)},
    )
    resp = client.get("/predict?symbol=BTCUSDT&timeframe=1h")
    assert resp.status_code == 200
    body = resp.json()
    assert body["symbol"] == "BTCUSDT"
    assert body["current_price"] > 0
    assert body["predicted_price"] > 0
    # predicted_change_pct must be the real derived fraction, not a default.
    expected = (body["predicted_price"] - body["current_price"]) / body["current_price"]
    assert body["predicted_change_pct"] == pytest.approx(expected, rel=1e-9)
    assert 0.0 <= body["confidence"] <= 1.0
    assert body["direction"] in ("up", "down", "neutral")


def test_predict_rejects_an_unsupported_timeframe(client):
    resp = client.get("/predict?symbol=BTCUSDT&timeframe=7h")
    assert resp.status_code == 400
    assert "unsupported timeframe" in resp.json()["detail"]


def test_signal_requires_history(client):
    assert client.get("/signal?symbol=BTCUSDT").status_code == 422


def test_signal_returns_a_shape_the_clients_can_read(client):
    client.post(
        "/ingest", headers={"X-API-Key": "test-key"},
        json={"symbol": "BTCUSDT", "candles": _candles(120)},
    )
    body = client.get("/signal?symbol=BTCUSDT").json()
    assert body["signal_type"] in ("buy", "sell", "hold")
    assert 0.0 <= body["strength"] <= 1.0
    assert isinstance(body["reason"], str)


# ---------------------------------------------------------------------------
# Provider adapters
# ---------------------------------------------------------------------------


def test_symbol_split_handles_the_common_quotes():
    assert _split_symbol("BTCUSDT") == ("BTC", "USDT")
    assert _split_symbol("ETHUSDC") == ("ETH", "USDC")
    assert _split_symbol("SOLUSD") == ("SOL", "USD")
    # A stablecoin base must not be misread as a quote.
    assert _split_symbol("USDCUSDT") == ("USDC", "USDT")


def test_symbol_split_rejects_an_unknown_pair():
    with pytest.raises(Exception):
        _split_symbol("WEIRD")


def test_parses_binance_rows(monkeypatch):
    monkeypatch.setattr(ai_service, "CANDLE_PROVIDER", "binance")
    payload = [
        [1700000000000, "100.0", "110.0", "90.0", "105.0", "12.5", 1700003599999, "0", 0, "0", "0", "0"],
        [1700003600000, "105.0", "115.0", "95.0", "108.0", "13.5", 1700007199999, "0", 0, "0", "0", "0"],
    ]
    rows = _parse_provider_rows(payload, "BTCUSDT")
    assert len(rows) == 2
    assert rows[0].timestamp == 1700000000
    assert rows[0].open == 100.0
    assert rows[0].high == 110.0
    assert rows[0].low == 90.0
    assert rows[0].close == 105.0
    assert rows[0].volume == 12.5
    # Oldest first, so the engine's "last price" is the newest candle.
    assert rows[-1].timestamp == 1700003600


def test_parses_coinbase_rows_and_reorders_them(monkeypatch):
    monkeypatch.setattr(ai_service, "CANDLE_PROVIDER", "coinbase")
    # Coinbase returns [time, low, high, open, close, volume] NEWEST FIRST.
    payload = [
        [1700003600, 95.0, 115.0, 105.0, 108.0, 13.5],
        [1700000000, 90.0, 110.0, 100.0, 105.0, 12.5],
    ]
    rows = _parse_provider_rows(payload, "BTCUSDT")
    assert len(rows) == 2
    assert rows[0].timestamp == 1700000000, "must be reordered oldest-first"
    assert rows[0].open == 100.0
    assert rows[0].high == 110.0
    assert rows[0].low == 90.0
    assert rows[0].close == 105.0
    assert rows[-1].timestamp == 1700003600


def test_parses_kraken_rows(monkeypatch):
    monkeypatch.setattr(ai_service, "CANDLE_PROVIDER", "kraken")
    payload = {
        "error": [],
        "result": {
            "XXBTZUSD": [
                [1700000000, "100.0", "110.0", "90.0", "105.0", "104.0", "12.5", 42],
                [1700003600, "105.0", "115.0", "95.0", "108.0", "107.0", "13.5", 43],
            ],
            "last": 1700003600,
        },
    }
    rows = _parse_provider_rows(payload, "BTCUSDT")
    assert len(rows) == 2
    assert rows[0].timestamp == 1700000000
    assert rows[0].volume == 12.5, "kraken volume is field 6, not field 5"
    assert rows[-1].timestamp == 1700003600


def test_kraken_error_is_surfaced(monkeypatch):
    monkeypatch.setattr(ai_service, "CANDLE_PROVIDER", "kraken")
    with pytest.raises(Exception) as exc:
        _parse_provider_rows({"error": ["EQuery:Unknown asset pair"], "result": {}}, "BTCUSDT")
    assert "Unknown asset pair" in str(exc.value)


def test_empty_provider_response_is_an_error(monkeypatch):
    monkeypatch.setattr(ai_service, "CANDLE_PROVIDER", "binance")
    with pytest.raises(Exception):
        _parse_provider_rows([], "BTCUSDT")


def test_unknown_provider_is_rejected():
    # A misconfigured provider must be a hard error, not a silent fallback.
    original = ai_service.CANDLE_PROVIDER
    try:
        ai_service.CANDLE_PROVIDER = "not_an_exchange"
        with pytest.raises(Exception):
            ai_service._provider_url("BTCUSDT", "1h")
    finally:
        ai_service.CANDLE_PROVIDER = original


def test_fetch_fails_closed_when_the_provider_is_unreachable(client, monkeypatch):
    monkeypatch.setattr(ai_service, "CANDLE_PROVIDER", "binance")
    monkeypatch.setattr(ai_service, "CANDLE_PROVIDER_URL", "http://127.0.0.1:1/klines")
    resp = client.post(
        "/fetch",
        headers={"X-API-Key": "test-key"},
        json={"symbol": "BTCUSDT", "timeframe": "1h", "limit": 50},
    )
    assert resp.status_code == 502
    assert "unreachable" in resp.json()["detail"]
    # Nothing was ingested: a failed fetch must not leave partial state that a
    # later prediction would treat as real history.
    assert not ai_service.engine.price_history.get("BTCUSDT")


def test_fetch_requires_the_key(client):
    resp = client.post("/fetch", json={"symbol": "BTCUSDT", "timeframe": "1h", "limit": 50})
    assert resp.status_code == 401


def test_candle_model_rejects_a_short_series():
    # The schema bound is what stops a caller from asking for a prediction with
    # too little history to compute the indicators.
    from pydantic import ValidationError

    with pytest.raises(ValidationError):
        ai_service.IngestRequest(symbol="BTCUSDT", candles=[])
