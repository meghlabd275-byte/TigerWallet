#!/usr/bin/env python3
"""TigerWallet AI prediction HTTP service.

The prediction engine in ``price_prediction.py`` is real (indicator-based
ensemble over ingested candles), but it had no network surface, so nothing in
the platform could use it. This module exposes it to the trading bots.

Design notes
------------
* Candles are INGESTED, never invented. The service fetches OHLCV from a real
  market data provider and stores it; if the provider cannot be reached the
  request fails with a 502 rather than predicting on fabricated prices.
* A prediction is only returned when the engine has enough history to compute
  its indicators. Below that it answers ``insufficient_data`` with a neutral
  direction, so a caller cannot mistake an absence of data for a signal.
* ``/predict`` is what ``bot_core``'s ``ai_trading`` runner polls. The response
  shape matches what the runner decodes.

Run: ``uvicorn ai_service:app --host 0.0.0.0 --port 8091``
"""

from __future__ import annotations

import logging
import os
import time
from typing import Dict, List, Optional

import httpx
from fastapi import FastAPI, Header, HTTPException, Query
from pydantic import BaseModel, Field

from price_prediction import PricePoint, PricePredictionEngine

logger = logging.getLogger("TigerWallet.AI.Service")

# ---------------------------------------------------------------------------
# Configuration (all from the environment; nothing is hardcoded as a secret)
# ---------------------------------------------------------------------------

# OHLCV provider. Exchanges restrict different regions, so the provider is
# selectable: "binance" (global), "binance_us", "coinbase" or "kraken". Each
# adapter parses that exchange's real response shape.
CANDLE_PROVIDER = os.environ.get("AI_CANDLE_PROVIDER", "binance").strip().lower()

# Overrides the endpoint of the selected provider (for a private mirror or a
# regional host). Empty means "use the provider's public endpoint".
CANDLE_PROVIDER_URL = os.environ.get("AI_CANDLE_PROVIDER_URL", "").strip()

# Kraken uses its own asset naming for the public OHLC endpoint.
KRAKEN_ASSETS = {
    "BTC": "XBT",
    "DOGE": "XDG",
}

PROVIDER_ENDPOINTS = {
    "binance": "https://api.binance.com/api/v3/klines",
    "binance_us": "https://api.binance.us/api/v3/klines",
    "coinbase": "https://api.exchange.coinbase.com/products/{product}/candles",
    "kraken": "https://api.kraken.com/0/public/OHLC",
}

# Coinbase accepts only these granularities, in seconds.
COINBASE_GRANULARITY = {
    "1m": 60,
    "5m": 300,
    "15m": 900,
    "1h": 3600,
    "6h": 21600,
    "1d": 86400,
}
# Shared secret required on the mutating endpoints. Ingestion is a write, so it
# is gated; prediction is read-only and open to the platform network.
INGEST_API_KEY = os.environ.get("AI_INGEST_API_KEY", "")

# Interval string -> seconds, used to validate the requested timeframe.
TIMEFRAME_SECONDS: Dict[str, int] = {
    "1m": 60,
    "5m": 300,
    "15m": 900,
    "30m": 1800,
    "1h": 3600,
    "4h": 14400,
    "1d": 86400,
}

# The engine keeps its own history; one process-wide instance is enough because
# the state is per-symbol and the service is not sharded.
engine = PricePredictionEngine()

app = FastAPI(
    title="TigerWallet AI Prediction Service",
    version="1.0.0",
    description=(
        "Real indicator-based price prediction over ingested market data. "
        "Never fabricates prices: an unreachable provider is a 502."
    ),
)


# ---------------------------------------------------------------------------
# Schemas
# ---------------------------------------------------------------------------


class Candle(BaseModel):
    timestamp: int = Field(..., description="Candle open time, unix seconds")
    open: float
    high: float
    low: float
    close: float
    volume: float


class IngestRequest(BaseModel):
    symbol: str = Field(..., min_length=1, max_length=32)
    candles: List[Candle] = Field(..., min_length=1, max_length=5000)


class IngestResponse(BaseModel):
    symbol: str
    accepted: int
    history_length: int


class FetchRequest(BaseModel):
    symbol: str = Field(..., min_length=1, max_length=32)
    timeframe: str = "1h"
    limit: int = Field(200, ge=30, le=1000)


class PredictionResponse(BaseModel):
    symbol: str
    current_price: float
    predicted_price: float
    # Expressed as a fraction (0.01 = +1%), which is what the bot runner reads.
    predicted_change_pct: float
    confidence: float
    direction: str
    timeframe: str
    timestamp: int
    factors: List[str]
    model_version: str


def _require_ingest_key(provided: Optional[str]) -> None:
    """Gate the write endpoints. Fails closed when no key is configured: an
    unauthenticated writer could otherwise poison the model's inputs."""
    if not INGEST_API_KEY:
        raise HTTPException(
            status_code=503,
            detail="AI_INGEST_API_KEY is not configured; ingestion is disabled",
        )
    if provided != INGEST_API_KEY:
        raise HTTPException(status_code=401, detail="invalid ingest key")


def _validate_timeframe(timeframe: str) -> str:
    if timeframe not in TIMEFRAME_SECONDS:
        raise HTTPException(
            status_code=400,
            detail=f"unsupported timeframe {timeframe!r}; supported: {sorted(TIMEFRAME_SECONDS)}",
        )
    return timeframe


@app.get("/health")
def health() -> Dict[str, object]:
    return {
        "status": "ok",
        "service": "ai-prediction",
        "version": app.version,
        "candle_provider": CANDLE_PROVIDER,
        "ingest_enabled": bool(INGEST_API_KEY),
    }


@app.post("/ingest", response_model=IngestResponse)
def ingest(
    req: IngestRequest,
    x_api_key: Optional[str] = Header(default=None, alias="X-API-Key"),
) -> IngestResponse:
    """Ingest caller-supplied candles (the bot platform's own feed)."""
    _require_ingest_key(x_api_key)
    symbol = req.symbol.upper()
    for c in req.candles:
        if not (c.low <= c.high):
            raise HTTPException(
                status_code=400,
                detail=f"candle at {c.timestamp} has low > high",
            )
        if min(c.open, c.high, c.low, c.close, c.volume) < 0:
            raise HTTPException(
                status_code=400,
                detail=f"candle at {c.timestamp} has a negative value",
            )
        engine.add_price_data(
            symbol,
            PricePoint(
                timestamp=c.timestamp,
                open=c.open,
                high=c.high,
                low=c.low,
                close=c.close,
                volume=c.volume,
            ),
        )
    return IngestResponse(
        symbol=symbol,
        accepted=len(req.candles),
        history_length=len(engine.price_history.get(symbol, [])),
    )


def _split_symbol(symbol: str) -> tuple:
    """Split a concatenated market symbol such as BTCUSDT into base and quote.

    Exchanges that name markets by pair (Coinbase, Kraken) need the two halves
    separately. Raises HTTPException when the symbol is not a recognised pair,
    rather than guessing a quote currency and quoting the wrong market.
    """
    upper = symbol.upper()
    for quote in ("USDT", "USDC", "USD", "EUR", "GBP", "BTC", "ETH", "JPY"):
        if upper.endswith(quote) and len(upper) > len(quote):
            return upper[: -len(quote)], quote
    raise HTTPException(
        status_code=400,
        detail=(
            f"cannot split {symbol!r} into base/quote; the selected provider needs "
            f"a pair such as BTCUSDT"
        ),
    )


def _provider_url(symbol: str, timeframe: str) -> tuple:
    """Build the request URL and query params for the selected provider."""
    if CANDLE_PROVIDER in ("binance", "binance_us"):
        url = CANDLE_PROVIDER_URL or PROVIDER_ENDPOINTS[CANDLE_PROVIDER]
        return url, {"symbol": symbol.upper(), "interval": timeframe}

    if CANDLE_PROVIDER == "coinbase":
        if timeframe not in COINBASE_GRANULARITY:
            raise HTTPException(
                status_code=400,
                detail=(
                    f"coinbase supports granularities {sorted(COINBASE_GRANULARITY)}; "
                    f"{timeframe!r} was requested"
                ),
            )
        base, quote = _split_symbol(symbol)
        product = f"{base}-{quote}"
        url = CANDLE_PROVIDER_URL or PROVIDER_ENDPOINTS["coinbase"].format(product=product)
        return url, {"granularity": COINBASE_GRANULARITY[timeframe]}

    if CANDLE_PROVIDER == "kraken":
        if timeframe not in TIMEFRAME_SECONDS:
            raise HTTPException(status_code=400, detail=f"unsupported timeframe {timeframe!r}")
        base, quote = _split_symbol(symbol)
        pair = KRAKEN_ASSETS.get(base, base) + quote
        url = CANDLE_PROVIDER_URL or PROVIDER_ENDPOINTS["kraken"]
        return url, {"pair": pair, "interval": TIMEFRAME_SECONDS[timeframe] // 60}

    raise HTTPException(
        status_code=500,
        detail=(
            f"AI_CANDLE_PROVIDER {CANDLE_PROVIDER!r} is not one of "
            f"{sorted(PROVIDER_ENDPOINTS)}"
        ),
    )


def _parse_provider_rows(payload: object, symbol: str) -> List[Candle]:
    """Parse the selected provider's response into candles, newest last."""
    rows: List[Candle] = []

    if CANDLE_PROVIDER in ("binance", "binance_us"):
        if not isinstance(payload, list):
            raise HTTPException(status_code=502, detail="binance response is not a list")
        # [openTime(ms), open, high, low, close, volume, ...]
        for row in payload:
            rows.append(
                Candle(
                    timestamp=int(row[0]) // 1000,
                    open=float(row[1]),
                    high=float(row[2]),
                    low=float(row[3]),
                    close=float(row[4]),
                    volume=float(row[5]),
                )
            )

    elif CANDLE_PROVIDER == "coinbase":
        if not isinstance(payload, list):
            raise HTTPException(status_code=502, detail="coinbase response is not a list")
        # [time(sec), low, high, open, close, volume], newest first.
        for row in payload:
            rows.append(
                Candle(
                    timestamp=int(row[0]),
                    low=float(row[1]),
                    high=float(row[2]),
                    open=float(row[3]),
                    close=float(row[4]),
                    volume=float(row[5]),
                )
            )
        rows.reverse()

    elif CANDLE_PROVIDER == "kraken":
        if not isinstance(payload, dict):
            raise HTTPException(status_code=502, detail="kraken response is not an object")
        errors = payload.get("error") or []
        if errors:
            raise HTTPException(status_code=502, detail=f"kraken returned {errors}")
        result = payload.get("result") or {}
        series = None
        for key, value in result.items():
            if key == "last":
                continue
            if isinstance(value, list):
                series = value
                break
        if series is None:
            raise HTTPException(
                status_code=502, detail=f"kraken returned no series for {symbol}"
            )
        # [time(sec), open, high, low, close, vwap, volume, count], oldest first.
        for row in series:
            rows.append(
                Candle(
                    timestamp=int(row[0]),
                    open=float(row[1]),
                    high=float(row[2]),
                    low=float(row[3]),
                    close=float(row[4]),
                    volume=float(row[6]),
                )
            )

    if not rows:
        raise HTTPException(
            status_code=502, detail=f"candle provider returned no data for {symbol}"
        )
    # The engine treats the last element as the most recent price, so order the
    # series oldest-first regardless of how the provider returned it.
    rows.sort(key=lambda c: c.timestamp)
    return rows


async def _fetch_candles(symbol: str, timeframe: str, limit: int) -> List[Candle]:
    """Fetch real OHLCV candles from the configured provider.

    Raises HTTPException on any failure. There is deliberately no fallback to
    synthetic data: a prediction built on invented prices would be acted on by
    a live trading bot.
    """
    url, params = _provider_url(symbol, timeframe)
    try:
        async with httpx.AsyncClient(timeout=15.0) as client:
            resp = await client.get(url, params=params)
    except httpx.HTTPError as exc:
        raise HTTPException(
            status_code=502, detail=f"candle provider unreachable: {exc}"
        ) from exc

    if resp.status_code != 200:
        raise HTTPException(
            status_code=502,
            detail=f"candle provider returned {resp.status_code}: {resp.text[:200]}",
        )

    try:
        payload = resp.json()
    except ValueError as exc:
        raise HTTPException(
            status_code=502, detail="candle provider returned a non-JSON body"
        ) from exc

    try:
        candles = _parse_provider_rows(payload, symbol)
    except (IndexError, TypeError, ValueError) as exc:
        raise HTTPException(
            status_code=502,
            detail=f"candle provider returned an unexpected row for {symbol}: {exc}",
        ) from exc

    return candles[-limit:] if len(candles) > limit else candles


@app.post("/fetch", response_model=IngestResponse)
async def fetch(
    req: FetchRequest,
    x_api_key: Optional[str] = Header(default=None, alias="X-API-Key"),
) -> IngestResponse:
    """Pull real candles from the provider into the engine's history."""
    _require_ingest_key(x_api_key)
    timeframe = _validate_timeframe(req.timeframe)
    candles = await _fetch_candles(req.symbol, timeframe, req.limit)
    symbol = req.symbol.upper()
    for c in candles:
        engine.add_price_data(
            symbol,
            PricePoint(
                timestamp=c.timestamp,
                open=c.open,
                high=c.high,
                low=c.low,
                close=c.close,
                volume=c.volume,
            ),
        )
    return IngestResponse(
        symbol=symbol,
        accepted=len(candles),
        history_length=len(engine.price_history.get(symbol, [])),
    )


@app.get("/predict", response_model=PredictionResponse)
async def predict(
    symbol: str = Query(..., min_length=1, max_length=32),
    timeframe: str = Query("1h"),
    refresh: bool = Query(
        False,
        description="Fetch fresh candles from the provider before predicting",
    ),
    x_api_key: Optional[str] = Header(default=None, alias="X-API-Key"),
) -> PredictionResponse:
    """Predict the next move for a symbol.

    With ``refresh=true`` the service pulls new candles first, so a bot polling
    this endpoint always predicts on current data. Without it, the last ingested
    history is used.
    """
    timeframe = _validate_timeframe(timeframe)
    symbol = symbol.upper()

    if refresh:
        _require_ingest_key(x_api_key)
        candles = await _fetch_candles(symbol, timeframe, 200)
        for c in candles:
            engine.add_price_data(
                symbol,
                PricePoint(
                    timestamp=c.timestamp,
                    open=c.open,
                    high=c.high,
                    low=c.low,
                    close=c.close,
                    volume=c.volume,
                ),
            )

    prediction = engine.predict(symbol, timeframe)
    if prediction.current_price <= 0:
        raise HTTPException(
            status_code=422,
            detail=(
                f"insufficient history for {symbol}: {len(engine.price_history.get(symbol, []))} "
                f"candles available, 30 required. Ingest candles or call /fetch first."
            ),
        )

    predicted_change = (
        (prediction.predicted_price - prediction.current_price) / prediction.current_price
    )
    return PredictionResponse(
        symbol=prediction.symbol,
        current_price=prediction.current_price,
        predicted_price=prediction.predicted_price,
        predicted_change_pct=predicted_change,
        confidence=prediction.confidence,
        direction=prediction.direction,
        timeframe=prediction.timeframe,
        timestamp=prediction.timestamp,
        factors=prediction.factors,
        model_version=prediction.model_version,
    )


@app.get("/signal")
def signal(symbol: str = Query(..., min_length=1, max_length=32)) -> Dict[str, object]:
    """Market signal derived from the ingested history (buy/sell/hold)."""
    symbol = symbol.upper()
    if not engine.price_history.get(symbol):
        raise HTTPException(status_code=422, detail=f"no history for {symbol}")
    result = engine.get_market_signal(symbol)
    if hasattr(result, "to_dict"):
        return result.to_dict()
    return result  # type: ignore[return-value]


if __name__ == "__main__":
    import uvicorn

    logging.basicConfig(level=logging.INFO)
    uvicorn.run(
        app,
        host=os.environ.get("AI_HOST", "0.0.0.0"),
        port=int(os.environ.get("AI_PORT", "8091")),
    )
