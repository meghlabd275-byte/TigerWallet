"""Tests for the gas price forecaster.

Run with:  python3 -m pytest gas_market/python/tests -q
       or:  python3 gas_market/python/tests/test_gas_prediction.py
"""

import math
import os
import sys
from datetime import datetime, timedelta, timezone

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from gas_prediction import (  # noqa: E402
    MIN_OBSERVATIONS,
    GasDataPoint,
    GasPredictor,
    InsufficientHistoryError,
    calculate_savings,
)


def hourly_history(hours=200, base=30.0, daily_amplitude=0.0, slope=0.0, start=None):
    """Build a deterministic observation series."""
    start = start or datetime(2026, 1, 5, 0, 0, tzinfo=timezone.utc)  # a Monday
    points = []
    for h in range(hours):
        ts = start + timedelta(hours=h)
        # Cheap at 02:00 UTC, expensive at 14:00 UTC.
        seasonal = daily_amplitude * math.cos((ts.hour - 2) * 2 * math.pi / 24)
        price = base * (1 + seasonal) + slope * h
        points.append(
            GasDataPoint(
                timestamp=int(ts.timestamp()),
                gas_price=round(price, 4),
                block_number=h,
                base_fee=round(price * 0.8, 4),
            )
        )
    return points


# --- Refusal to fabricate --------------------------------------------------


def test_predict_refuses_without_history():
    p = GasPredictor()
    try:
        p.predict()
    except InsufficientHistoryError as e:
        assert "observations" in str(e)
    else:
        raise AssertionError("predict() must not invent a forecast with no history")


def test_optimize_refuses_without_history():
    p = GasPredictor()
    try:
        p.optimize_gas(budget_gwei=50, urgency="medium")
    except InsufficientHistoryError:
        pass
    else:
        raise AssertionError("optimize_gas() must not invent a recommendation")


def test_predict_refuses_below_minimum():
    p = GasPredictor()
    p.add_observations(hourly_history(hours=MIN_OBSERVATIONS - 1))
    try:
        p.predict()
    except InsufficientHistoryError:
        pass
    else:
        raise AssertionError("predict() must require MIN_OBSERVATIONS")


def test_predict_works_at_minimum():
    p = GasPredictor()
    p.add_observations(hourly_history(hours=MIN_OBSERVATIONS))
    out = p.predict(hours_ahead=3)
    assert len(out["predictions"]) == 3
    assert out["observation_count"] == MIN_OBSERVATIONS


# --- Determinism -----------------------------------------------------------


def test_predictions_are_deterministic():
    # The old implementation drew from np.random, so two calls on identical
    # input returned different numbers.
    points = hourly_history(hours=200, daily_amplitude=0.2, slope=0.05)
    a = GasPredictor()
    a.add_observations(points)
    b = GasPredictor()
    b.add_observations(points)
    assert a.predict(24) == b.predict(24)


def test_repeated_predict_on_one_model_is_stable():
    p = GasPredictor()
    p.add_observations(hourly_history(hours=200, daily_amplitude=0.2))
    assert p.predict(12) == p.predict(12)


# --- The model actually reflects the data ----------------------------------


def test_tracks_the_seasonal_shape():
    # The series peaks at 02:00 UTC and bottoms at 14:00 UTC. The model has to
    # reproduce that ordering, not just produce plausible-looking numbers.
    points = hourly_history(hours=200, daily_amplitude=0.3)
    p = GasPredictor()
    p.add_observations(points)

    day = datetime(2026, 1, 12, tzinfo=timezone.utc)  # a Monday
    by_hour = {
        hour: p._predict_at(int((day + timedelta(hours=hour)).timestamp()))
        for hour in range(24)
    }
    assert min(by_hour, key=by_hour.get) == 14, "14:00 UTC should be cheapest"
    assert max(by_hour, key=by_hour.get) == 2, "02:00 UTC should be dearest"

    average = sum(by_hour.values()) / len(by_hour)
    assert by_hour[14] < average * 0.9, "seasonal trough should be clearly below average"
    assert by_hour[2] > average * 1.1, "seasonal peak should be clearly above average"


def test_trend_is_recovered():
    # A steady 0.1 gwei/hour rise must show up as a positive slope.
    p = GasPredictor()
    p.add_observations(hourly_history(hours=300, slope=0.1))
    assert p._slope_per_hour > 0.05, f"slope not recovered: {p._slope_per_hour}"
    out = p.predict(hours_ahead=1)
    assert out["trend_gwei_per_hour"] > 0.05


def test_higher_base_gives_higher_forecast():
    low = GasPredictor()
    low.add_observations(hourly_history(hours=200, base=20.0))
    high = GasPredictor()
    high.add_observations(hourly_history(hours=200, base=80.0))
    assert high.predict(1)["average_gas"] > low.predict(1)["average_gas"]


# --- Confidence ------------------------------------------------------------


def test_confidence_is_bounded_and_decays():
    p = GasPredictor()
    p.add_observations(hourly_history(hours=300, daily_amplitude=0.2))
    out = p.predict(hours_ahead=48)
    confidences = [x["confidence"] for x in out["predictions"]]
    for c in confidences:
        assert 0.0 < c <= 0.95, f"confidence out of range: {c}"
    # Far-future hours must be less certain than the near ones.
    assert confidences[-1] < confidences[0]


def test_noisy_history_lowers_confidence():
    # Same seasonal shape, but one series is polluted with large residuals.
    clean = GasPredictor()
    clean.add_observations(hourly_history(hours=300, daily_amplitude=0.2))

    noisy_points = hourly_history(hours=300, daily_amplitude=0.2)
    for i, point in enumerate(noisy_points):
        if i % 3 == 0:
            point.gas_price *= 1.0 + (0.5 if i % 2 else -0.5)
    noisy = GasPredictor()
    noisy.add_observations(noisy_points)

    assert noisy._residual_std > clean._residual_std
    assert noisy.predict(1)["predictions"][0]["confidence"] < \
        clean.predict(1)["predictions"][0]["confidence"]


def test_confidence_band_widens_with_horizon():
    p = GasPredictor()
    p.add_observations(hourly_history(hours=300, daily_amplitude=0.2))
    out = p.predict(hours_ahead=24)
    first, last = out["predictions"][0], out["predictions"][-1]
    assert (last["upper_bound"] - last["lower_bound"]) > \
        (first["upper_bound"] - first["lower_bound"])


# --- optimize_gas ----------------------------------------------------------


def test_optimize_clamps_to_budget():
    p = GasPredictor()
    p.add_observations(hourly_history(hours=300, base=100.0))
    rec = p.optimize_gas(budget_gwei=25.0, urgency="low")
    assert rec.max_fee_gwei <= 25.0, "recommendation must respect the budget"
    assert rec.expected_gas <= 25.0


def test_optimize_high_urgency_waits_nothing():
    p = GasPredictor()
    p.add_observations(hourly_history(hours=300, base=100.0, daily_amplitude=0.3))
    rec = p.optimize_gas(budget_gwei=500.0, urgency="high")
    assert rec.wait_seconds == 0, "high urgency cannot wait"


def test_optimize_low_urgency_can_wait_for_the_trough():
    p = GasPredictor()
    p.add_observations(hourly_history(hours=300, base=100.0, daily_amplitude=0.3))
    rec = p.optimize_gas(budget_gwei=500.0, urgency="low")
    assert rec.wait_seconds > 0, "low urgency should wait for the cheap hour"


def test_optimize_rejects_bad_input():
    p = GasPredictor()
    p.add_observations(hourly_history(hours=300))
    for budget in (0, -1):
        try:
            p.optimize_gas(budget_gwei=budget, urgency="low")
        except ValueError:
            pass
        else:
            raise AssertionError(f"budget {budget} must be rejected")
    try:
        p.optimize_gas(budget_gwei=10, urgency="whenever")
    except ValueError:
        pass
    else:
        raise AssertionError("unknown urgency must be rejected")


def test_optimize_priority_fee_within_max_fee():
    p = GasPredictor()
    p.add_observations(hourly_history(hours=300, base=50.0))
    for urgency in ("high", "medium", "low"):
        rec = p.optimize_gas(budget_gwei=200.0, urgency=urgency)
        assert rec.priority_fee_gwei <= rec.max_fee_gwei


# --- calculate_savings -----------------------------------------------------


def test_savings_math():
    out = calculate_savings(50.0, 30.0, 10, gas_per_tx=21000, native_price_usd=3000.0)
    # 20 gwei * 21000 gas = 420000 gwei per tx, x10 = 4200000 gwei
    assert out["savings_per_tx_gwei"] == 420000.0
    assert out["total_savings_gwei"] == 4200000.0
    assert out["total_savings_native"] == 0.0042
    assert out["total_savings_usd"] == 12.6


def test_savings_without_a_price_is_none_not_guessed():
    # The old code hardcoded $1800/ETH and reported a dollar figure anyway.
    out = calculate_savings(50.0, 30.0, 10)
    assert out["total_savings_usd"] is None, "must not invent a native price"


def test_savings_never_negative():
    out = calculate_savings(30.0, 50.0, 10)
    assert out["total_savings_gwei"] == 0.0


def test_savings_rejects_bad_input():
    for args in ((-1, 10, 1), (10, -1, 1), (10, 10, -1), (10, 10, 1, 0)):
        try:
            calculate_savings(*args)
        except ValueError:
            pass
        else:
            raise AssertionError(f"bad input {args} must be rejected")


# --- Retention -------------------------------------------------------------


def test_old_observations_are_trimmed():
    p = GasPredictor(lookback_hours=24)
    p.add_observations(hourly_history(hours=200))
    # 24 hours of window, inclusive of both endpoints.
    assert p.observation_count <= 25, p.observation_count


def test_replay_of_old_data_still_predicts():
    # Anchoring the window to wall-clock time (as the first draft did) would
    # discard a historical series entirely.
    old = datetime(2024, 1, 1, tzinfo=timezone.utc)
    p = GasPredictor()
    p.add_observations(hourly_history(hours=100, start=old))
    assert p.observation_count == 100
    assert len(p.predict(2)["predictions"]) == 2


if __name__ == "__main__":
    failures = 0
    for name, fn in sorted(globals().items()):
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
