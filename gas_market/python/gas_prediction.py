"""
TigerWallet Gas Price Prediction

Forecasts gas prices from observed block history using a seasonal
(weekday + hour-of-day) profile plus an exponentially weighted level and a
least-squares trend, and recommends when to submit a transaction for a given
urgency.

The module never invents data: with no history it raises rather than returning
a constant series. Callers feed real observations with ``add_observation``.
"""

import logging
import math
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Dict, List, Tuple

logger = logging.getLogger("TigerWallet.GasMarket")

# A forecast is only meaningful once the model has seen enough blocks to
# separate the daily rhythm from noise.
MIN_OBSERVATIONS = 24

# Blocks are ~12s on Ethereum mainnet; other chains override this.
DEFAULT_BLOCK_SECONDS = 12.0

# Exponential weighting for the level estimate. Small enough that a week of
# history still matters, large enough to react within a few hours.
TREND_ALPHA = 0.05

# A wait has to be worth it before it is recommended.
MIN_SAVING_PCT = 0.05


@dataclass
class GasDataPoint:
    """One observed block."""

    timestamp: int
    gas_price: float
    block_number: int
    base_fee: float = 0.0
    priority_fee: float = 0.0


@dataclass
class GasForecast:
    """A single point forecast."""

    hours_ahead: int
    predicted_gas: float
    confidence: float
    lower_bound: float
    upper_bound: float


@dataclass
class UrgencyRecommendation:
    """When to submit for a given urgency."""

    urgency: str
    wait_seconds: int
    expected_gas: float
    max_fee_gwei: float
    priority_fee_gwei: float


class InsufficientHistoryError(RuntimeError):
    """Raised when a prediction is requested without enough observations."""


class GasPredictor:
    """
    Gas price forecaster.

    The model is deliberately small and explainable: a multiplicative seasonal
    profile indexed by weekday and hour, applied to a level and trend estimated
    from the observations. That answers the practical question ("is gas cheap
    in two hours?") without a training pipeline.
    """

    def __init__(
        self,
        lookback_hours: int = 168,
        block_seconds: float = DEFAULT_BLOCK_SECONDS,
    ) -> None:
        self.lookback_hours = lookback_hours
        self.block_seconds = block_seconds
        self._observations: List[GasDataPoint] = []
        self._seasonal: Dict[Tuple[int, int], float] = {}
        self._level: float = 0.0
        self._slope_per_hour: float = 0.0
        self._residual_std: float = 0.0
        self._fitted = False

    # -- data ingestion ----------------------------------------------------

    def add_observation(self, point: GasDataPoint) -> None:
        """Record one block observation and refit."""
        self._observations.append(point)
        self._observations.sort(key=lambda p: p.timestamp)
        self._trim()
        self._fit()

    def add_observations(self, points: List[GasDataPoint]) -> None:
        """Record many block observations and refit once."""
        self._observations.extend(points)
        self._observations.sort(key=lambda p: p.timestamp)
        self._trim()
        self._fit()

    @property
    def observation_count(self) -> int:
        return len(self._observations)

    def _trim(self) -> None:
        # Anchored to the newest sample, not to wall-clock time, so a backtest
        # or a replay of stored blocks is not thrown away.
        newest = max(p.timestamp for p in self._observations)
        cutoff = newest - self.lookback_hours * 3600
        self._observations = [p for p in self._observations if p.timestamp >= cutoff]

    # -- model fitting -----------------------------------------------------

    def _fit(self) -> None:
        prices = [p.gas_price for p in self._observations if p.gas_price > 0]
        if len(prices) < MIN_OBSERVATIONS:
            self._fitted = False
            return

        # Level: exponential moving average, so recent blocks dominate.
        level = prices[0]
        for price in prices[1:]:
            level = TREND_ALPHA * price + (1.0 - TREND_ALPHA) * level
        self._level = level

        # Trend: least-squares slope of price against elapsed hours.
        base_ts = self._observations[0].timestamp
        xs = [(p.timestamp - base_ts) / 3600.0 for p in self._observations]
        ys = [p.gas_price for p in self._observations]
        n = len(xs)
        mean_x = sum(xs) / n
        mean_y = sum(ys) / n
        denom = sum((x - mean_x) ** 2 for x in xs)
        self._slope_per_hour = (
            sum((xs[i] - mean_x) * (ys[i] - mean_y) for i in range(n)) / denom
            if denom > 0
            else 0.0
        )

        # Seasonal profile: mean price per (weekday, hour) over the overall
        # mean. Slots with no samples fall back to 1.0 (no effect).
        overall = mean_y if mean_y > 0 else 1.0
        buckets: Dict[Tuple[int, int], List[float]] = {}
        for point in self._observations:
            dt = datetime.fromtimestamp(point.timestamp, tz=timezone.utc)
            buckets.setdefault((dt.weekday(), dt.hour), []).append(point.gas_price)
        self._seasonal = {
            key: (sum(vals) / len(vals)) / overall for key, vals in buckets.items()
        }

        # Residual spread drives the confidence band. Measured in-sample, so it
        # reports how well the seasonal shape explains the history at hand.
        self._fitted = True
        residuals = [p.gas_price - self._predict_at(p.timestamp) for p in self._observations]
        if len(residuals) > 1:
            mean_r = sum(residuals) / len(residuals)
            var = sum((r - mean_r) ** 2 for r in residuals) / (len(residuals) - 1)
            self._residual_std = math.sqrt(var)
        else:
            self._residual_std = 0.0

    # -- prediction --------------------------------------------------------

    def _predict_at(self, timestamp: int) -> float:
        if not self._observations:
            return 0.0
        base_ts = self._observations[0].timestamp
        hours_ahead = (timestamp - base_ts) / 3600.0
        dt = datetime.fromtimestamp(timestamp, tz=timezone.utc)
        seasonal = self._seasonal.get((dt.weekday(), dt.hour), 1.0)
        predicted = (self._level + self._slope_per_hour * hours_ahead) * seasonal
        return max(predicted, 0.0)

    def _confidence(self, hours_ahead: int) -> float:
        """
        Confidence from the residual spread and the forecast horizon.

        Uncertainty grows with the horizon, and a noisy history lowers the
        ceiling. Returned in [0.05, 0.95] so the value never claims certainty.
        """
        if self._level <= 0:
            return 0.0
        relative_noise = self._residual_std / self._level
        noise_factor = 1.0 / (1.0 + relative_noise * 4.0)
        horizon_factor = 1.0 / (1.0 + hours_ahead / 48.0)
        return max(0.05, min(0.95, 0.95 * noise_factor * horizon_factor))

    def predict(self, hours_ahead: int = 24) -> Dict:
        """
        Forecast the next ``hours_ahead`` hours.

        Raises InsufficientHistoryError when fewer than MIN_OBSERVATIONS blocks
        have been recorded, instead of returning a fabricated series.
        """
        if hours_ahead < 1:
            raise ValueError("hours_ahead must be >= 1")
        if not self._fitted:
            raise InsufficientHistoryError(
                f"need at least {MIN_OBSERVATIONS} observations, have {len(self._observations)}"
            )

        now = max(p.timestamp for p in self._observations)
        predictions: List[GasForecast] = []
        for h in range(hours_ahead):
            ts = now + h * 3600
            predicted = self._predict_at(ts)
            confidence = self._confidence(h)
            # The band widens with the residual spread and the horizon.
            spread = self._residual_std * (1.0 + h / 12.0)
            predictions.append(
                GasForecast(
                    hours_ahead=h,
                    predicted_gas=round(predicted, 2),
                    confidence=round(confidence, 4),
                    lower_bound=round(max(predicted - spread, 0.0), 2),
                    upper_bound=round(predicted + spread, 2),
                )
            )

        average = sum(p.predicted_gas for p in predictions) / len(predictions)
        return {
            "predictions": [
                {
                    "hours_ahead": p.hours_ahead,
                    "predicted_gas": p.predicted_gas,
                    "confidence": p.confidence,
                    "lower_bound": p.lower_bound,
                    "upper_bound": p.upper_bound,
                }
                for p in predictions
            ],
            "average_gas": round(average, 2),
            "congestion_level": self._congestion_level(predictions[0].predicted_gas),
            "observation_count": len(self._observations),
            "trend_gwei_per_hour": round(self._slope_per_hour, 4),
            "seasonal_applied": bool(self._seasonal),
        }

    @staticmethod
    def _congestion_level(gas_price: float) -> str:
        if gas_price < 20:
            return "LOW"
        if gas_price < 50:
            return "MEDIUM"
        return "HIGH"

    def optimize_gas(self, budget_gwei: float, urgency: str) -> UrgencyRecommendation:
        """
        Recommend when to submit, given a maximum acceptable gas price.

        Compares the forecast for the next hours against the budget and picks
        the cheapest hour within the urgency's patience. It never recommends
        paying more than the budget.
        """
        if budget_gwei <= 0:
            raise ValueError("budget_gwei must be > 0")
        if urgency not in ("high", "medium", "low"):
            raise ValueError("urgency must be one of: high, medium, low")
        if not self._fitted:
            raise InsufficientHistoryError(
                f"need at least {MIN_OBSERVATIONS} observations, have {len(self._observations)}"
            )

        horizon_hours = {"high": 0, "medium": 6, "low": 48}[urgency]

        now = max(p.timestamp for p in self._observations)
        now_price = self._predict_at(now)
        best_hour = 0
        best_price = now_price
        for h in range(1, horizon_hours + 1):
            price = self._predict_at(now + h * 3600)
            if price < best_price:
                best_price = price
                best_hour = h

        saving_pct = (now_price - best_price) / now_price if now_price > 0 else 0.0
        if saving_pct < MIN_SAVING_PCT:
            best_hour = 0
            best_price = now_price

        # A budget is a ceiling: clamp the recommendation down to it.
        if best_price > budget_gwei:
            best_price = budget_gwei

        priority_ratio = {"high": 0.5, "medium": 0.25, "low": 0.1}[urgency]
        priority = min(best_price * priority_ratio, best_price)

        return UrgencyRecommendation(
            urgency=urgency,
            wait_seconds=best_hour * 3600,
            expected_gas=round(best_price, 2),
            max_fee_gwei=round(best_price, 2),
            priority_fee_gwei=round(priority, 2),
        )


def calculate_savings(
    current_gas_gwei: float,
    optimized_gas_gwei: float,
    tx_count: int,
    gas_per_tx: int = 21000,
    native_price_usd: float = 0.0,
) -> Dict:
    """
    Savings from deferring transactions to a cheaper hour.

    ``native_price_usd`` must come from a live price feed. There is no built-in
    default because a guessed price would make the reported dollar figure
    wrong, so the field stays None until the caller supplies one.
    """
    if current_gas_gwei < 0 or optimized_gas_gwei < 0:
        raise ValueError("gas prices must be >= 0")
    if tx_count < 0:
        raise ValueError("tx_count must be >= 0")
    if gas_per_tx <= 0:
        raise ValueError("gas_per_tx must be > 0")

    delta_gwei = max(current_gas_gwei - optimized_gas_gwei, 0.0)
    savings_per_tx_gwei = delta_gwei * gas_per_tx
    total_savings_gwei = savings_per_tx_gwei * tx_count

    result = {
        "savings_per_tx_gwei": round(savings_per_tx_gwei, 2),
        "total_savings_gwei": round(total_savings_gwei, 2),
        "total_savings_native": round(total_savings_gwei / 1e9, 9),
        "total_savings_usd": None,
    }
    if native_price_usd > 0:
        result["total_savings_usd"] = round(total_savings_gwei / 1e9 * native_price_usd, 2)
    return result
