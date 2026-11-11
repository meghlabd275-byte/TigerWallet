"""Tests for the token scanner.

The Keccak vectors are the real function selectors of the ERC-20 and ERC-721
entry points, so a wrong padding constant or a wrong rotation constant fails
the suite rather than silently producing plausible hashes.
"""

import os
import sys

import pytest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from token_analysis import (  # noqa: E402
    LiquidityChecker,
    ScamDetector,
    TokenFeatures,
    analyze_bytecode,
    analyze_token_full,
    keccak256,
    selector,
)

# ---------------------------------------------------------------------------
# Keccak-256
# ---------------------------------------------------------------------------

# (input, expected 4-byte selector) - the canonical selectors used on-chain.
SELECTOR_VECTORS = [
    ("transfer(address,uint256)", "a9059cbb"),
    ("approve(address,uint256)", "095ea7b3"),
    ("balanceOf(address)", "70a08231"),
    ("totalSupply()", "18160ddd"),
    ("transferFrom(address,address,uint256)", "23b872dd"),
    ("pause()", "8456cb59"),
    ("setPaused(bool)", "16c38b3c"),
    ("blacklist(address)", "f9f92be4"),
    ("mint(uint256)", "a0712d68"),
    ("upgradeTo(address)", "3659cfe6"),
    ("renounceOwnership()", "715018a6"),
    ("transferOwnership(address)", "f2fde38b"),
    ("owner()", "8da5cb5b"),
    ("balanceOf(address,uint256)", "00fdd58e"),  # ERC-1155
]


@pytest.mark.parametrize("signature,expected", SELECTOR_VECTORS)
def test_selectors_match_mainnet(signature, expected):
    assert selector(signature).hex() == expected


def test_keccak_empty_input():
    # Keccak-256 of the empty string.
    assert keccak256(b"").hex() == (
        "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
    )


def test_keccak_is_not_sha3():
    # If the padding constant were 0x06 this would equal sha3_256 instead.
    import hashlib

    assert keccak256(b"abc") != hashlib.sha3_256(b"abc").digest()
    assert keccak256(b"abc").hex() == (
        "4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45"
    )


def test_keccak_crosses_the_rate_boundary():
    # 136 bytes is exactly the rate, so this exercises multi-block absorption.
    digest = keccak256(b"a" * 136)
    assert len(digest) == 32
    assert keccak256(b"a" * 136) != keccak256(b"a" * 137)


# ---------------------------------------------------------------------------
# Bytecode decoding
# ---------------------------------------------------------------------------

def _push4(sel_hex: str) -> bytes:
    return bytes([0x63]) + bytes.fromhex(sel_hex)


def test_detects_pausable_token():
    code = _push4("8456cb59")  # pause()
    result = analyze_bytecode("0x" + code.hex())
    assert result.analysed is True
    kinds = [f.kind for f in result.findings]
    assert "token_transfer_pausable" in kinds
    assert result.risk_points > 0
    # The offset points at the PUSH4 instruction itself.
    pause = next(f for f in result.findings if f.kind == "token_transfer_pausable")
    assert pause.offset == 0
    assert pause.selector == "0x8456cb59"


def test_detects_blacklist_and_mint():
    code = _push4("f9f92be4") + _push4("a0712d68")
    result = analyze_bytecode(code.hex())
    kinds = {f.kind for f in result.findings}
    assert "address_blacklist" in kinds
    assert "supply_inflation" in kinds


def test_selfdestruct_and_delegatecall_are_reported():
    result = analyze_bytecode("ff" + "f4")
    kinds = {f.kind for f in result.findings}
    assert "selfdestruct" in kinds
    assert "delegatecall" in kinds


def test_push_operands_are_not_misread_as_opcodes():
    # PUSH32 followed by 32 bytes that happen to look like SELFDESTRUCT and
    # DELEGATECALL. A decoder that does not skip operands reports both.
    code = bytes([0x7F]) + bytes([0xFF, 0xF4]) + bytes(30)
    result = analyze_bytecode(code.hex())
    kinds = {f.kind for f in result.findings}
    assert "selfdestruct" not in kinds
    assert "delegatecall" not in kinds


def test_standard_token_is_recognised():
    code = (
        _push4("a9059cbb")   # transfer
        + _push4("095ea7b3")  # approve
        + _push4("70a08231")  # balanceOf
        + _push4("18160ddd")  # totalSupply
    )
    result = analyze_bytecode(code.hex())
    assert result.is_standard_token is True
    assert not any(f.kind == "not_a_token" for f in result.findings)


def test_contract_without_erc20_entry_points_is_flagged():
    code = _push4("a0712d68")  # only mint(uint256)
    result = analyze_bytecode(code.hex())
    assert result.is_standard_token is False
    assert any(f.kind == "not_a_token" for f in result.findings)


def test_source_code_is_reported_as_unknown_not_scored():
    source = "pragma solidity ^0.8.0; contract T { function pause() public {} }"
    result = analyze_bytecode(source)
    assert result.analysed is False
    assert result.reason is not None
    assert result.risk_points == 0
    assert result.severity == "UNKNOWN"


def test_empty_code_is_unknown():
    result = analyze_bytecode("")
    assert result.analysed is False
    assert "no contract code" in result.reason


def test_odd_length_hex_is_rejected():
    result = analyze_bytecode("0xabc")
    assert result.analysed is False


# ---------------------------------------------------------------------------
# Feature scoring
# ---------------------------------------------------------------------------

def _safe_features(**overrides) -> TokenFeatures:
    base = dict(
        liquidity_ratio=0.5,
        holder_concentration=0.05,
        transfer_count=50_000,
        buy_sell_ratio=0.5,
        price_volatility=0.05,
        social_score=0.8,
        dev_activity=0.9,
        age_hours=24 * 365,
        contract_verified=True,
        owner_changed=False,
        mint_enabled=False,
        pause_enabled=False,
    )
    base.update(overrides)
    return TokenFeatures(**base)


def test_clean_token_scores_low():
    assessment = ScamDetector().assess(_safe_features())
    # Holder concentration is scored linearly, so a healthy-but-nonzero top
    # holder share still contributes a fraction of a point. What matters is
    # that the total stays far inside the LOW band.
    assert assessment.score < 5.0
    assert assessment.risk_level == "LOW"
    assert assessment.recommendation == "SAFE"
    assert assessment.unknown == []


def test_zero_concentration_scores_exactly_zero():
    assessment = ScamDetector().assess(_safe_features(holder_concentration=0.0))
    assert assessment.score == pytest.approx(0.0)


def test_every_risk_signal_scores_high():
    assessment = ScamDetector().assess(_safe_features(
        liquidity_ratio=0.0,
        holder_concentration=1.0,
        buy_sell_ratio=1.0,
        age_hours=1,
        contract_verified=False,
        owner_changed=True,
        mint_enabled=True,
        pause_enabled=True,
    ))
    assert assessment.score == pytest.approx(100.0)
    assert assessment.risk_level == "HIGH"
    assert assessment.recommendation == "AVOID"


def test_weights_sum_to_one():
    assert sum(ScamDetector.WEIGHTS.values()) == pytest.approx(1.0)


def test_each_signal_contribution_is_explained():
    assessment = ScamDetector().assess(_safe_features(pause_enabled=True))
    names = {s.name: s for s in assessment.signals}
    assert "transfer_pausable" in names
    signal = names["transfer_pausable"]
    assert signal.contribution == pytest.approx(signal.weight * signal.severity)
    # The reported score is exactly the sum of the reported contributions.
    assert assessment.score == pytest.approx(
        sum(s.contribution for s in assessment.signals) * 100.0
    )


def test_unmeasured_optional_signal_is_excluded_not_treated_as_safe():
    assessment = ScamDetector().assess(_safe_features(age_hours=-1))
    assert "age_hours" in assessment.unknown
    assert "very_new" not in {s.name for s in assessment.signals}


def test_scores_are_monotonic_in_holder_concentration():
    detector = ScamDetector()
    low = detector.assess(_safe_features(holder_concentration=0.1)).score
    mid = detector.assess(_safe_features(holder_concentration=0.5)).score
    high = detector.assess(_safe_features(holder_concentration=0.9)).score
    assert low < mid < high


def test_liquidity_ramp_is_monotonic():
    detector = ScamDetector()
    scores = [
        detector.assess(_safe_features(liquidity_ratio=r)).score
        for r in (0.0, 0.05, 0.10, 0.15, 0.25)
    ]
    assert scores == sorted(scores, reverse=True)


def test_out_of_range_inputs_are_clamped():
    assessment = ScamDetector().assess(_safe_features(
        liquidity_ratio=-5.0,
        holder_concentration=99.0,
        buy_sell_ratio=-3.0,
    ))
    assert 0.0 <= assessment.score <= 100.0
    for signal in assessment.signals:
        assert 0.0 <= signal.severity <= 1.0


def test_predict_scam_probability_matches_score():
    features = _safe_features(mint_enabled=True)
    detector = ScamDetector()
    assert detector.predict_scam_probability(features) == pytest.approx(
        detector.assess(features).score / 100.0
    )


# ---------------------------------------------------------------------------
# Liquidity
# ---------------------------------------------------------------------------

def test_liquidity_checker_flags_unlocked_liquidity():
    checker = LiquidityChecker()
    result = checker.check_liquidity(
        token_address="0x" + "11" * 20,
        liquidity_amount=1_000.0,
        liquidity_usd=5_000.0,
        lock_percentage=10.0,
        lock_duration=3600,
    )
    assert result["analysed"] is True
    assert result["is_locked"] is False
    assert result["sufficient"] is False
    assert result["status"] == "RISKY"
    assert result["risk_score"] == 60  # 30 + 20 + 10
    assert len(result["warnings"]) == 3


def test_liquidity_checker_accepts_healthy_liquidity():
    result = LiquidityChecker().check_liquidity(
        token_address="0x" + "22" * 20,
        liquidity_amount=1_000_000.0,
        liquidity_usd=500_000.0,
        lock_percentage=100.0,
        lock_duration=365 * 24 * 3600,
    )
    assert result["status"] == "SAFE"
    assert result["risk_score"] == 0


def test_known_locker_list_is_not_defaulted():
    # With no configured lockers the locker is reported as unknown (None),
    # never as recognised, and no points are added.
    result = LiquidityChecker().check_liquidity(
        token_address="0x" + "33" * 20,
        liquidity_amount=1_000.0,
        liquidity_usd=50_000.0,
        lock_percentage=100.0,
        lock_duration=365 * 24 * 3600,
        locker_address="0x" + "44" * 20,
    )
    assert result["locker_recognised"] is None
    assert result["risk_score"] == 0


def test_unlisted_locker_adds_a_warning():
    checker = LiquidityChecker(known_lockers=["0x" + "ab" * 20])
    result = checker.check_liquidity(
        token_address="0x" + "33" * 20,
        liquidity_amount=1_000.0,
        liquidity_usd=50_000.0,
        lock_percentage=100.0,
        lock_duration=365 * 24 * 3600,
        locker_address="0x" + "cd" * 20,
    )
    assert result["locker_recognised"] is False
    assert result["risk_score"] == 5
    assert any("known lockers" in w for w in result["warnings"])


def test_negative_liquidity_is_rejected():
    result = LiquidityChecker().check_liquidity(
        token_address="0x" + "55" * 20,
        liquidity_amount=-1.0,
        liquidity_usd=1.0,
        lock_percentage=1.0,
        lock_duration=1,
    )
    assert result["analysed"] is False


# ---------------------------------------------------------------------------
# Combined analysis
# ---------------------------------------------------------------------------

def test_full_analysis_reports_unknown_inputs():
    result = analyze_token_full("0x" + "66" * 20, _safe_features())
    assert result["contract_analysis"]["analysed"] is False
    assert "contract_code" in result["unknown"]
    assert "liquidity_lock" in result["unknown"]


def test_full_analysis_combines_both_sources():
    code = (_push4("8456cb59") + _push4("f9f92be4") + _push4("a0712d68")).hex()
    result = analyze_token_full(
        "0x" + "77" * 20,
        _safe_features(),
        contract_code=code,
        liquidity_usd=5_000.0,
        lock_percentage=10.0,
        lock_duration=3600,
    )
    assert result["contract_analysis"]["analysed"] is True
    assert result["contract_analysis"]["risk_points"] > 0
    assert result["liquidity_analysis"]["status"] == "RISKY"
    assert result["unknown"] == []
    assert result["overall_risk_score"] > 0
    assert result["risk_level"] in {"LOW", "MEDIUM", "HIGH"}


def test_full_analysis_never_exceeds_one_hundred():
    code = ("ff" + _push4("8456cb59").hex() + _push4("f9f92be4").hex()
            + _push4("a0712d68").hex() + _push4("3659cfe6").hex())
    result = analyze_token_full(
        "0x" + "88" * 20,
        _safe_features(
            liquidity_ratio=0.0,
            holder_concentration=1.0,
            buy_sell_ratio=0.0,
            age_hours=1,
            contract_verified=False,
            owner_changed=True,
            mint_enabled=True,
            pause_enabled=True,
        ),
        contract_code=code,
        liquidity_usd=0.0,
        lock_percentage=0.0,
        lock_duration=0,
    )
    assert 0.0 <= result["overall_risk_score"] <= 100.0
    assert result["recommendation"] == "AVOID"


def test_reported_score_is_reproducible_from_reported_signals():
    """An operator must be able to recompute the score from the response."""
    result = analyze_token_full(
        "0x" + "99" * 20,
        _safe_features(mint_enabled=True, contract_verified=False),
        contract_code="",
    )
    signals = result["scam_analysis"]["signals"]
    recomputed = sum(s["contribution"] for s in signals) * 100.0
    assert recomputed == pytest.approx(result["scam_analysis"]["score"], abs=0.01)
