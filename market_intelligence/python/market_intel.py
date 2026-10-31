"""
TigerWallet Market Intelligence Module

Price forecasting, whale detection, sentiment scoring, arbitrage detection and
portfolio analytics over market data supplied by the caller.

Every number this module returns is derived from its inputs. There is no random
sampling and no placeholder pricing: methods that would need data they do not
have return None or raise instead of inventing a plausible-looking figure.
"""

import logging
import math
import time
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple

import numpy as np

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Minimum observations before a forecast is attempted.
MIN_PREDICTION_POINTS = 30

# Steps per timeframe, assuming hourly observations.
TIMEFRAME_STEPS = {"1h": 1, "4h": 4, "24h": 24, "1d": 24, "7d": 168}

# A forecast may not project more than this over a single horizon. Guards
# against an outlier-driven drift without suppressing a genuine trend.
MAX_HORIZON_RETURN = 0.5


@dataclass
class MarketData:
    """Market data point"""

    timestamp: int
    open: float
    high: float
    low: float
    close: float
    volume: float


@dataclass
class PricePrediction:
    """Price prediction result"""

    token: str
    current_price: float
    predicted_price: float
    confidence: float
    direction: str  # "up", "down", "neutral"
    timeframe: str
    generated_at: int


@dataclass
class SentimentData:
    """Social sentiment data"""

    token: str
    sentiment_score: float  # -1 to 1
    bullish_percentage: float
    bearish_percentage: float
    mentions: int
    trending: bool
    sources: List[str]
    analyzed_at: int


@dataclass
class WhaleAlert:
    """Large transaction alert"""

    token: str
    from_address: str
    to_address: str
    amount: float
    value_usd: float
    transaction_type: str  # "buy", "sell", "transfer"
    detected_at: int


class MarketIntelligence:
    """
    Market intelligence engine for crypto markets.

    Provides price prediction, whale tracking and sentiment analysis. Callers
    push observations in with add_price_data; nothing is fetched or fabricated
    internally.
    """

    def __init__(self, whale_threshold_usd: float = 100_000.0) -> None:
        self.price_history: Dict[str, List[MarketData]] = {}
        self.whale_addresses: Dict[str, float] = {}
        self.whale_threshold_usd = whale_threshold_usd
        self.max_history = 1000

    def add_price_data(self, token: str, data: MarketData) -> None:
        """Add a price observation to the history."""
        history = self.price_history.setdefault(token, [])
        history.append(data)
        if len(history) > self.max_history:
            del history[: len(history) - self.max_history]

    def predict_price(
        self, token: str, timeframe: str = "24h"
    ) -> Optional[PricePrediction]:
        """
        Forecast the close for `timeframe` from the recorded history.

        The estimate carries the realised log-return drift forward and damps it
        with a pull toward the 50-period mean. Confidence falls as realised
        volatility and the horizon grow, so a choppy series reports low
        confidence rather than a confident guess.

        Returns None when there is not enough data to estimate anything. It
        never samples from a random distribution: a prediction that changes on
        every call cannot be back-tested or acted on.
        """
        if timeframe not in TIMEFRAME_STEPS:
            raise ValueError(f"unsupported timeframe: {timeframe}")

        history = self.price_history.get(token)
        if not history or len(history) < MIN_PREDICTION_POINTS:
            logger.warning("insufficient data for %s", token)
            return None

        prices = np.asarray([d.close for d in history[-100:]], dtype=float)
        prices = prices[prices > 0]
        if prices.size < MIN_PREDICTION_POINTS:
            return None

        current_price = float(prices[-1])
        log_returns = np.diff(np.log(prices))
        if log_returns.size < 2:
            return None

        drift = float(np.mean(log_returns))
        volatility = float(np.std(log_returns, ddof=1))
        if not math.isfinite(drift) or not math.isfinite(volatility):
            return None

        steps = TIMEFRAME_STEPS[timeframe]

        horizon_sigma = volatility * math.sqrt(steps)

        # Mean reversion pulls a fraction of the distance back to the mean. It
        # is a prior rather than an estimated quantity, so its contribution is
        # bounded to one horizon sigma and cannot dominate the measured drift.
        mean_price = float(np.mean(prices[-50:]))
        reversion = -0.05 * (current_price - mean_price) / mean_price if mean_price > 0 else 0.0
        reversion = float(np.clip(reversion, -horizon_sigma, horizon_sigma))

        expected_return = drift * steps + reversion

        # Sanity bound only. A three-sigma cap on the total would erase any
        # real trend, because a smooth series has near-zero return volatility
        # and the cap would then be far smaller than the drift it is meant to
        # protect. A strong low-volatility trend is exactly the signal worth
        # carrying forward, so the bound is a fixed fraction of price.
        expected_return = float(
            np.clip(expected_return, -MAX_HORIZON_RETURN, MAX_HORIZON_RETURN)
        )

        predicted_price = current_price * math.exp(expected_return)

        relative_uncertainty = volatility * math.sqrt(steps)
        confidence = float(np.clip(1.0 / (1.0 + relative_uncertainty * 5.0), 0.05, 0.95))

        if expected_return > 0.005:
            direction = "up"
        elif expected_return < -0.005:
            direction = "down"
        else:
            direction = "neutral"

        return PricePrediction(
            token=token,
            current_price=current_price,
            predicted_price=predicted_price,
            confidence=confidence,
            direction=direction,
            timeframe=timeframe,
            generated_at=int(time.time()),
        )

    def analyze_sentiment(self, token: str, social_data: Dict) -> SentimentData:
        """Score social sentiment for a token from caller-supplied counts."""
        bullish = social_data.get("bullish", 0)
        bearish = social_data.get("bearish", 0)
        total = bullish + bearish

        if total == 0:
            sentiment_score = 0.0
            bullish_pct = 50.0
            bearish_pct = 50.0
        else:
            sentiment_score = (bullish - bearish) / total
            bullish_pct = (bullish / total) * 100
            bearish_pct = (bearish / total) * 100

        mentions = social_data.get("mentions", 0)

        return SentimentData(
            token=token,
            sentiment_score=sentiment_score,
            bullish_percentage=bullish_pct,
            bearish_percentage=bearish_pct,
            mentions=mentions,
            trending=mentions > 10000,
            sources=social_data.get("sources", ["twitter", "reddit"]),
            analyzed_at=int(time.time()),
        )

    def detect_whale_activity(
        self, token: str, transactions: List[Dict]
    ) -> List[WhaleAlert]:
        """Flag transactions above the whale threshold."""
        alerts = []
        now = int(time.time())
        for tx in transactions:
            value_usd = tx.get("value_usd", 0)
            if value_usd >= self.whale_threshold_usd:
                alerts.append(
                    WhaleAlert(
                        token=token,
                        from_address=tx.get("from", ""),
                        to_address=tx.get("to", ""),
                        amount=tx.get("amount", 0),
                        value_usd=value_usd,
                        transaction_type=tx.get("type", "transfer"),
                        detected_at=tx.get("timestamp", now),
                    )
                )
        return alerts

    def get_market_metrics(self, token: str) -> Dict:
        """Summarise the recorded history for a token."""
        history = self.price_history.get(token)
        if not history:
            return {}

        prices = [d.close for d in history]
        if not prices:
            return {}

        current = prices[-1]
        high_24h = max(prices[-24:]) if len(prices) >= 24 else max(prices)
        low_24h = min(prices[-24:]) if len(prices) >= 24 else min(prices)
        volume = history[-1].volume

        change_1h = ((prices[-1] - prices[-2]) / prices[-2] * 100) if len(prices) >= 2 and prices[-2] else 0.0
        change_24h = ((prices[-1] - prices[-24]) / prices[-24] * 100) if len(prices) >= 24 and prices[-24] else 0.0

        if len(prices) >= 30:
            window = np.asarray(prices[-30:], dtype=float)
            base = window[:-1]
            returns = np.diff(window) / np.where(base == 0, np.nan, base)
            returns = returns[np.isfinite(returns)]
            volatility = float(np.std(returns)) * 100 if returns.size else 0.0
        else:
            volatility = 0.0

        return {
            "token": token,
            "current_price": current,
            "high_24h": high_24h,
            "low_24h": low_24h,
            "volume_24h": volume,
            "change_1h": change_1h,
            "change_24h": change_24h,
            "volatility": volatility,
            "updated_at": int(time.time()),
        }

    def find_arbitrage_opportunities(
        self,
        prices: Dict[str, Dict[str, float]],
        trade_size: float = 1000.0,
        min_profit_pct: float = 0.5,
        fee_pct_per_leg: float = 0.0,
    ) -> List[Dict]:
        """
        Find cross-exchange arbitrage opportunities.

        The profit estimate is net of the caller-supplied taker fee on both
        legs, so an opportunity only appears when it survives costs. Fees
        default to zero only because they are exchange-specific; pass the real
        value to get an executable signal.
        """
        opportunities = []
        total_fee_pct = fee_pct_per_leg * 2

        for token, exchange_prices in prices.items():
            if len(exchange_prices) < 2:
                continue

            min_exchange, min_price = min(exchange_prices.items(), key=lambda kv: kv[1])
            max_exchange, max_price = max(exchange_prices.items(), key=lambda kv: kv[1])

            if min_price <= 0:
                continue

            gross_pct = ((max_price - min_price) / min_price) * 100
            net_pct = gross_pct - total_fee_pct
            if net_pct <= min_profit_pct:
                continue

            opportunities.append(
                {
                    "token": token,
                    "buy_exchange": min_exchange,
                    "sell_exchange": max_exchange,
                    "buy_price": min_price,
                    "sell_price": max_price,
                    "gross_profit_percentage": gross_pct,
                    "net_profit_percentage": net_pct,
                    "trade_size": trade_size,
                    "estimated_profit_usd": (max_price - min_price) * trade_size
                    - (min_price * trade_size * total_fee_pct / 100),
                    "timestamp": int(time.time()),
                }
            )

        return opportunities


class PortfolioAnalyzer:
    """Analyze user portfolio performance."""

    def __init__(self) -> None:
        self.transactions: List[Dict] = []

    def add_transaction(self, tx: Dict) -> None:
        """Add a transaction to the history."""
        self.transactions.append(tx)

    def calculate_pnl(self, holdings: Dict[str, float], prices: Dict[str, float]) -> Dict:
        """
        Calculate portfolio profit and loss.

        `prices` maps token -> current unit price and must come from a live
        price feed. Tokens without a price are reported in `unpriced_tokens`
        instead of being valued at a made-up number, because a placeholder
        price yields a PnL figure that looks authoritative and is simply wrong.
        """
        if not prices:
            raise ValueError("prices is required; a portfolio cannot be valued without market prices")

        total_cost = 0.0
        total_value = 0.0
        priced: Dict[str, Dict[str, float]] = {}
        unpriced: List[str] = []

        for token, amount in holdings.items():
            buys = [
                t
                for t in self.transactions
                if t.get("token") == token and t.get("type") == "buy"
            ]
            bought_amount = sum(t.get("amount", 0) for t in buys)
            cost = sum(t.get("price", 0) * t.get("amount", 0) for t in buys)
            avg_price = cost / bought_amount if bought_amount > 0 else 0.0

            current_price = prices.get(token)
            if current_price is None:
                unpriced.append(token)
                total_cost += avg_price * amount
                continue

            position_cost = avg_price * amount
            position_value = current_price * amount
            total_cost += position_cost
            total_value += position_value
            priced[token] = {
                "amount": amount,
                "avg_price": avg_price,
                "current_price": current_price,
                "cost": position_cost,
                "value": position_value,
                "pnl": position_value - position_cost,
            }

        pnl = total_value - total_cost
        pnl_percentage = (pnl / total_cost * 100) if total_cost > 0 else 0.0

        return {
            "total_cost": total_cost,
            "total_value": total_value,
            "pnl": pnl,
            "pnl_percentage": pnl_percentage,
            "positions": priced,
            "unpriced_tokens": unpriced,
            "timestamp": int(time.time()),
        }

    def get_diversification_score(self, holdings: Dict[str, float]) -> float:
        """
        Concentration score (0-100) using the Herfindahl-Hirschman Index.

        Higher means more diversified. The index is scale-invariant, so raw
        holding amounts are a valid input.
        """
        if not holdings:
            return 0.0

        total_value = sum(holdings.values())
        if total_value <= 0:
            return 0.0

        weights = [v / total_value for v in holdings.values()]
        hhi = sum(w * w for w in weights)
        return round((1 - hhi) * 100, 2)

    def suggest_rebalancing(
        self,
        holdings: Dict[str, float],
        prices: Dict[str, float],
        target_allocation: Dict[str, float],
    ) -> List[Dict]:
        """
        Suggest rebalancing trades toward a target allocation.

        `prices` maps token -> current unit price and must come from a live
        price feed. Trade sizes come from real market value, so the suggested
        amount is executable rather than a placeholder ratio.
        """
        if not prices:
            raise ValueError("prices is required; a portfolio cannot be valued without market prices")

        if target_allocation:
            total_pct = sum(target_allocation.values())
            if abs(total_pct - 100.0) > 0.01:
                raise ValueError(f"target_allocation must sum to 100, got {total_pct}")

        total_value = sum(
            prices.get(token, 0.0) * amount for token, amount in holdings.items()
        )
        if total_value <= 0:
            return []

        suggestions = []
        for token, target_pct in target_allocation.items():
            price = prices.get(token)
            if price is None or price <= 0:
                logger.warning("skipping %s: no market price available", token)
                continue

            current_value = holdings.get(token, 0.0) * price
            current_pct = (current_value / total_value) * 100
            difference = total_value * (target_pct / 100) - current_value

            if abs(difference) > total_value * 0.05:  # 5% threshold
                suggestions.append(
                    {
                        "token": token,
                        "action": "buy" if difference > 0 else "sell",
                        "amount": abs(difference) / price,
                        "value_usd": abs(difference),
                        "current_pct": round(current_pct, 2),
                        "target_pct": target_pct,
                        "reason": f"rebalance from {current_pct:.1f}% to {target_pct}%",
                    }
                )

        return suggestions


__all__ = [
    "MarketIntelligence",
    "PortfolioAnalyzer",
    "MarketData",
    "PricePrediction",
    "SentimentData",
    "WhaleAlert",
]
