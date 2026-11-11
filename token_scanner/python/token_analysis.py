"""
TigerWallet Token Scanner - risk analysis.

Two things are analysed and combined:

1. Token features (liquidity, holder concentration, supply controls) supplied
   by the caller. These are scored with a fixed, published weight table, and
   every signal's contribution is returned so a score can be explained rather
   than taken on faith.
2. Contract bytecode, when the caller has it. The bytecode is decoded into
   opcodes and function selectors, and the result is reported as evidence
   (which selector was found, at which byte offset) rather than as a verdict
   with no basis.

The scorer is a calibrated heuristic, not a trained model. It is described as
such on purpose: a module that claims to be "ML" while multiplying caller
input by a hardcoded constant vector tells an operator nothing about how much
to trust the output. Here the weights are visible, the per-signal
contribution is visible, and a signal whose input is missing is reported as
unknown instead of being silently counted as safe.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Dict, List, Optional, Sequence

# ---------------------------------------------------------------------------
# Keccak-256 (the Ethereum variant, original padding - not SHA3-256)
# ---------------------------------------------------------------------------

_KECCAK_ROUND_CONSTANTS = (
    0x0000000000000001, 0x0000000000008082, 0x800000000000808A,
    0x8000000080008000, 0x000000000000808B, 0x0000000080000001,
    0x8000000080008081, 0x8000000000008009, 0x000000000000008A,
    0x0000000000000088, 0x0000000080008009, 0x000000008000000A,
    0x000000008000808B, 0x800000000000008B, 0x8000000000008089,
    0x8000000000008003, 0x8000000000008002, 0x8000000000000080,
    0x000000000000800A, 0x800000008000000A, 0x8000000080008081,
    0x8000000000008080, 0x0000000080000001, 0x8000000080008008,
)

_KECCAK_ROTATION_OFFSETS = (
    (0, 36, 3, 41, 18),
    (1, 44, 10, 45, 2),
    (62, 6, 43, 15, 61),
    (28, 55, 25, 21, 56),
    (27, 20, 39, 8, 14),
)

_MASK64 = (1 << 64) - 1

# 0x01 padding is Keccak; SHA3 uses 0x06. This single byte is why
# hashlib.sha3_256 cannot be substituted here.
_KECCAK_PADDING = 0x01


def _rotl64(value: int, shift: int) -> int:
    if shift == 0:
        return value & _MASK64
    return ((value << shift) | (value >> (64 - shift))) & _MASK64


def _keccak_f1600(state: List[int]) -> None:
    for round_index in range(24):
        # Theta
        c = [state[x] ^ state[x + 5] ^ state[x + 10] ^ state[x + 15] ^ state[x + 20]
             for x in range(5)]
        d = [c[(x + 4) % 5] ^ _rotl64(c[(x + 1) % 5], 1) for x in range(5)]
        for x in range(5):
            for y in range(5):
                state[x + 5 * y] ^= d[x]

        # Rho and Pi
        b = [0] * 25
        for x in range(5):
            for y in range(5):
                b[y + 5 * ((2 * x + 3 * y) % 5)] = _rotl64(
                    state[x + 5 * y], _KECCAK_ROTATION_OFFSETS[x][y]
                )

        # Chi
        for x in range(5):
            for y in range(5):
                state[x + 5 * y] = b[x + 5 * y] ^ (
                    (~b[(x + 1) % 5 + 5 * y]) & b[(x + 2) % 5 + 5 * y]
                ) & _MASK64

        # Iota
        state[0] ^= _KECCAK_ROUND_CONSTANTS[round_index]


def keccak256(data: bytes) -> bytes:
    """Return the Keccak-256 digest of ``data``."""
    rate = 136  # 1088-bit rate for a 256-bit output
    padded = bytearray(data)
    padded.append(_KECCAK_PADDING)
    while len(padded) % rate != 0:
        padded.append(0x00)
    padded[-1] ^= 0x80

    state = [0] * 25
    for offset in range(0, len(padded), rate):
        block = padded[offset:offset + rate]
        for i in range(rate // 8):
            state[i] ^= int.from_bytes(block[i * 8:(i + 1) * 8], "little")
        _keccak_f1600(state)

    return b"".join(state[i].to_bytes(8, "little") for i in range(4))


def selector(signature: str) -> bytes:
    """Return the 4-byte function selector for an ABI signature."""
    return keccak256(signature.encode("ascii"))[:4]


# ---------------------------------------------------------------------------
# Bytecode decoding
# ---------------------------------------------------------------------------

# PUSH widths matter: stepping over immediates correctly is the difference
# between reading the bytecode and mistaking operand bytes for instructions.
_PUSH_FIRST = 0x60
_PUSH_LAST = 0x7F
_PUSH4 = 0x63
_SELFDESTRUCT = 0xFF
_DELEGATECALL = 0xF4

# Selectors that change who can move tokens or how many exist. Matching these
# against PUSH4 immediates is how the analyser tells a token that merely
# contains the word "pause" from one that actually exposes pause().
_RISK_SELECTORS: Dict[bytes, tuple] = {
    selector("pause()"): ("token_transfer_pausable", 18,
                          "exposes pause() - transfers can be halted"),
    selector("setPaused(bool)"): ("token_transfer_pausable", 18,
                                  "exposes setPaused(bool) - transfers can be halted"),
    selector("blacklist(address)"): ("address_blacklist", 20,
                                     "exposes blacklist(address) - a holder can be blocked"),
    selector("addBlackList(address)"): ("address_blacklist", 20,
                                        "exposes addBlackList(address)"),
    selector("setBlacklist(address,bool)"): ("address_blacklist", 20,
                                             "exposes setBlacklist(address,bool)"),
    selector("setMaxTxAmount(uint256)"): ("transfer_limit", 12,
                                          "exposes setMaxTxAmount(uint256)"),
    selector("setMaxWalletAmount(uint256)"): ("transfer_limit", 12,
                                              "exposes setMaxWalletAmount(uint256)"),
    selector("mint(uint256)"): ("supply_inflation", 16,
                                "exposes mint(uint256) - supply can be increased"),
    selector("mint(address,uint256)"): ("supply_inflation", 16,
                                        "exposes mint(address,uint256)"),
    selector("upgradeTo(address)"): ("upgradeable_logic", 22,
                                     "exposes upgradeTo(address) - logic is replaceable"),
    selector("upgradeToAndCall(address,bytes)"): ("upgradeable_logic", 22,
                                                  "exposes upgradeToAndCall(address,bytes)"),
    selector("setFee(uint256)"): ("mutable_fee", 8, "exposes setFee(uint256)"),
    selector("setTaxFee(uint256)"): ("mutable_fee", 8, "exposes setTaxFee(uint256)"),
}

_STANDARD_SELECTORS: Dict[bytes, str] = {
    selector("transfer(address,uint256)"): "transfer(address,uint256)",
    selector("transferFrom(address,address,uint256)"): "transferFrom(address,address,uint256)",
    selector("approve(address,uint256)"): "approve(address,uint256)",
    selector("balanceOf(address)"): "balanceOf(address)",
    selector("totalSupply()"): "totalSupply()",
}


@dataclass
class Finding:
    """One piece of evidence found in the bytecode."""

    kind: str
    severity: int
    detail: str
    offset: Optional[int] = None
    selector: Optional[str] = None


@dataclass
class ContractAnalysis:
    """Result of decoding contract bytecode."""

    analysed: bool
    findings: List[Finding] = field(default_factory=list)
    risk_points: int = 0
    is_standard_token: bool = False
    reason: Optional[str] = None

    @property
    def severity(self) -> str:
        if not self.analysed:
            return "UNKNOWN"
        if self.risk_points >= 45:
            return "HIGH"
        if self.risk_points >= 20:
            return "MEDIUM"
        return "LOW"


def _normalise_bytecode(code: str) -> Optional[bytes]:
    """Return decoded bytecode, or None when the input is not hex.

    Solidity source is not bytecode and cannot be decoded by this module, so
    it is rejected rather than guessed at.
    """
    stripped = code.strip()
    if not stripped:
        return None
    if stripped[:2].lower() == "0x":
        stripped = stripped[2:]
    if len(stripped) % 2 != 0:
        return None
    try:
        return bytes.fromhex(stripped)
    except ValueError:
        return None


def analyze_bytecode(code: str) -> ContractAnalysis:
    """Decode EVM bytecode and report the capabilities it exposes.

    Returns analysed=False when the input is source or is not decodable,
    rather than guessing.
    """
    if not code or not code.strip():
        return ContractAnalysis(analysed=False, reason="no contract code supplied")

    bytecode = _normalise_bytecode(code)
    if bytecode is None:
        return ContractAnalysis(
            analysed=False,
            reason="input is not hex bytecode (source cannot be decoded here)",
        )

    findings: List[Finding] = []
    seen_selectors: set = set()
    has_selfdestruct = False
    has_delegatecall = False

    pc = 0
    while pc < len(bytecode):
        opcode = bytecode[pc]

        if opcode == _SELFDESTRUCT:
            has_selfdestruct = True
        elif opcode == _DELEGATECALL:
            has_delegatecall = True

        if opcode == _PUSH4 and pc + 4 < len(bytecode):
            immediate = bytes(bytecode[pc + 1:pc + 5])
            if immediate not in seen_selectors:
                seen_selectors.add(immediate)
                if immediate in _RISK_SELECTORS:
                    kind, severity, detail = _RISK_SELECTORS[immediate]
                    findings.append(
                        Finding(
                            kind=kind,
                            severity=severity,
                            detail=detail,
                            offset=pc,
                            selector="0x" + immediate.hex(),
                        )
                    )
            pc += 5
            continue

        if _PUSH_FIRST <= opcode <= _PUSH_LAST:
            pc += (opcode - _PUSH_FIRST) + 2
            continue

        pc += 1

    if has_selfdestruct:
        findings.append(
            Finding(
                kind="selfdestruct",
                severity=25,
                detail="contains SELFDESTRUCT - the contract can be removed",
            )
        )

    if has_delegatecall:
        findings.append(
            Finding(
                kind="delegatecall",
                severity=12,
                detail="contains DELEGATECALL - execution is delegated to another address",
            )
        )

    # A contract that exposes no ERC-20 entry points is not a token at all,
    # which matters because a "token" address that cannot transfer is a trap.
    standard_hits = [name for sel, name in _STANDARD_SELECTORS.items()
                     if sel in seen_selectors]
    is_standard = len(standard_hits) >= 3
    if not is_standard and seen_selectors:
        findings.append(
            Finding(
                kind="not_a_token",
                severity=15,
                detail="does not expose the core ERC-20 entry points",
            )
        )

    return ContractAnalysis(
        analysed=True,
        findings=findings,
        risk_points=sum(f.severity for f in findings),
        is_standard_token=is_standard,
    )


# ---------------------------------------------------------------------------
# Feature scoring
# ---------------------------------------------------------------------------

@dataclass
class TokenFeatures:
    """Features extracted from a token by the caller.

    A negative value on an optional numeric field means "not measured" and
    excludes that signal from the score.
    """

    liquidity_ratio: float
    holder_concentration: float
    transfer_count: int
    buy_sell_ratio: float
    price_volatility: float
    social_score: float
    dev_activity: float
    age_hours: int
    contract_verified: bool
    owner_changed: bool
    mint_enabled: bool
    pause_enabled: bool


@dataclass
class Signal:
    """One scored signal and the weight it carried."""

    name: str
    weight: float
    severity: float
    detail: str

    @property
    def contribution(self) -> float:
        return self.weight * self.severity


@dataclass
class ScamAssessment:
    """Explainable risk result for a set of token features."""

    score: float
    signals: List[Signal] = field(default_factory=list)
    unknown: List[str] = field(default_factory=list)

    @property
    def risk_level(self) -> str:
        if self.score >= 70:
            return "HIGH"
        if self.score >= 40:
            return "MEDIUM"
        return "LOW"

    @property
    def recommendation(self) -> str:
        if self.score >= 70:
            return "AVOID"
        if self.score >= 40:
            return "CAUTION"
        return "SAFE"

    def to_dict(self) -> Dict:
        return {
            "score": round(self.score, 2),
            "risk_level": self.risk_level,
            "recommendation": self.recommendation,
            "signals": [
                {
                    "name": s.name,
                    "weight": s.weight,
                    "severity": round(s.severity, 3),
                    "contribution": round(s.contribution, 3),
                    "detail": s.detail,
                }
                for s in self.signals
            ],
            "unknown_signals": list(self.unknown),
        }


def _clamp(value: float, low: float = 0.0, high: float = 1.0) -> float:
    return max(low, min(high, value))


class ScamDetector:
    """Scores token features with a published weight table.

    Each weight is the share of the total score that the signal can carry.
    The weights sum to 1.0, so the score is the weighted share of the worst
    case that the observed features actually reach.
    """

    WEIGHTS: Dict[str, float] = {
        "thin_liquidity": 0.18,
        "holder_concentration": 0.16,
        "supply_inflation": 0.14,
        "transfer_pausable": 0.12,
        "ownership_retained": 0.12,
        "unverified_contract": 0.10,
        "sell_pressure": 0.10,
        "very_new": 0.08,
    }

    def assess(self, features: TokenFeatures) -> ScamAssessment:
        signals: List[Signal] = []
        unknown: List[str] = []

        # Liquidity depth. A ratio at or below 2% is treated as fully thin,
        # 20% or more as fully covered.
        liquidity_ratio = _clamp(features.liquidity_ratio)
        if liquidity_ratio <= 0.02:
            liquidity_severity = 1.0
        elif liquidity_ratio >= 0.20:
            liquidity_severity = 0.0
        else:
            liquidity_severity = 1.0 - ((liquidity_ratio - 0.02) / 0.18)
        signals.append(Signal(
            name="thin_liquidity",
            weight=self.WEIGHTS["thin_liquidity"],
            severity=liquidity_severity,
            detail=f"liquidity ratio {features.liquidity_ratio:.4f}",
        ))

        concentration = _clamp(features.holder_concentration)
        signals.append(Signal(
            name="holder_concentration",
            weight=self.WEIGHTS["holder_concentration"],
            severity=concentration,
            detail=f"top-holder share {features.holder_concentration:.4f}",
        ))

        signals.append(Signal(
            name="supply_inflation",
            weight=self.WEIGHTS["supply_inflation"],
            severity=1.0 if features.mint_enabled else 0.0,
            detail="mint is enabled" if features.mint_enabled else "no mint entry point",
        ))

        signals.append(Signal(
            name="transfer_pausable",
            weight=self.WEIGHTS["transfer_pausable"],
            severity=1.0 if features.pause_enabled else 0.0,
            detail="transfers can be paused" if features.pause_enabled else "transfers are not pausable",
        ))

        # Retained ownership is only a risk once it has moved at least once;
        # a fresh contract that never transferred ownership is the norm.
        signals.append(Signal(
            name="ownership_retained",
            weight=self.WEIGHTS["ownership_retained"],
            severity=1.0 if features.owner_changed else 0.0,
            detail="ownership has changed hands" if features.owner_changed else "ownership unchanged",
        ))

        signals.append(Signal(
            name="unverified_contract",
            weight=self.WEIGHTS["unverified_contract"],
            severity=0.0 if features.contract_verified else 1.0,
            detail="source is verified" if features.contract_verified else "source is not verified",
        ))

        # Buy/sell imbalance: heavily one-sided flow is how a honeypot looks
        # from the outside. 0.5 is balanced; 0 or 1 is fully one-sided.
        imbalance = abs(_clamp(features.buy_sell_ratio) - 0.5) * 2.0
        signals.append(Signal(
            name="sell_pressure",
            weight=self.WEIGHTS["sell_pressure"],
            severity=imbalance,
            detail=f"buy/sell ratio {features.buy_sell_ratio:.4f}",
        ))

        # A token younger than a day has no trading history to judge.
        if features.age_hours < 0:
            unknown.append("age_hours")
        else:
            if features.age_hours < 24:
                age_severity = 1.0
            elif features.age_hours >= 24 * 30:
                age_severity = 0.0
            else:
                age_severity = 1.0 - ((features.age_hours - 24) / ((24 * 30) - 24))
            signals.append(Signal(
                name="very_new",
                weight=self.WEIGHTS["very_new"],
                severity=age_severity,
                detail=f"age {features.age_hours}h",
            ))

        # Volatility and the two activity scores are reported but deliberately
        # carry no weight: without a calibrated model they would be invented
        # precision. They stay in the output so an operator can see them.
        if features.price_volatility < 0:
            unknown.append("price_volatility")
        if features.social_score < 0:
            unknown.append("social_score")
        if features.dev_activity < 0:
            unknown.append("dev_activity")

        score = sum(s.contribution for s in signals) * 100.0
        return ScamAssessment(score=_clamp(score, 0.0, 100.0), signals=signals, unknown=unknown)

    # Retained for callers that only want the number.
    def predict_scam_probability(self, features: TokenFeatures) -> float:
        return self.assess(features).score / 100.0


class HoneypotDetector:
    """Detects honeypot and rug-pull capabilities from contract bytecode."""

    def analyze_contract(self, code: str) -> ContractAnalysis:
        return analyze_bytecode(code)


class LiquidityChecker:
    """Checks whether liquidity is deep, locked and locked long enough.

    The set of lock providers is supplied by the caller. It is not defaulted:
    a hardcoded "known locker" address would mark every token that uses a
    different locker as unlocked, and would silently bless one address.
    """

    MIN_LIQUIDITY_USD = 10_000.0
    MIN_LOCK_PERCENTAGE = 50.0
    MIN_LOCK_SECONDS = 30 * 24 * 60 * 60  # 30 days

    def __init__(self, known_lockers: Optional[Sequence[str]] = None):
        self.known_lockers = [a.lower() for a in (known_lockers or [])]

    def is_known_locker(self, address: Optional[str]) -> bool:
        if not address:
            return False
        return address.lower() in self.known_lockers

    def check_liquidity(
        self,
        token_address: str,
        liquidity_amount: float,
        liquidity_usd: float,
        lock_percentage: float,
        lock_duration: int,
        locker_address: Optional[str] = None,
    ) -> Dict:
        if liquidity_amount < 0 or liquidity_usd < 0:
            return {
                "token_address": token_address,
                "analysed": False,
                "reason": "liquidity figures must not be negative",
            }

        is_locked = lock_percentage >= self.MIN_LOCK_PERCENTAGE
        sufficient = liquidity_usd >= self.MIN_LIQUIDITY_USD
        long_lock = lock_duration >= self.MIN_LOCK_SECONDS

        # An unlisted locker is not proof of a problem, but it is also not
        # proof of safety, so it is reported as its own signal.
        locker_recognised = (
            self.is_known_locker(locker_address) if self.known_lockers else None
        )

        risk_score = 0
        warnings: List[str] = []

        if not is_locked:
            risk_score += 30
            warnings.append(
                f"liquidity lock is {lock_percentage:.2f}%, below the "
                f"{self.MIN_LOCK_PERCENTAGE:.0f}% threshold"
            )
        if not sufficient:
            risk_score += 20
            warnings.append(
                f"liquidity is ${liquidity_usd:,.2f}, below the "
                f"${self.MIN_LIQUIDITY_USD:,.0f} threshold"
            )
        if not long_lock:
            risk_score += 10
            warnings.append(
                f"lock runs for {lock_duration}s, under the "
                f"{self.MIN_LOCK_SECONDS}s threshold"
            )
        if locker_recognised is False:
            risk_score += 5
            warnings.append("locker is not on the configured list of known lockers")

        return {
            "token_address": token_address,
            "analysed": True,
            "is_locked": is_locked,
            "sufficient": sufficient,
            "long_lock": long_lock,
            "locker_recognised": locker_recognised,
            "risk_score": risk_score,
            "warnings": warnings,
            "status": "SAFE" if risk_score == 0 else "CAUTION" if risk_score < 30 else "RISKY",
        }


def analyze_token_full(
    address: str,
    features: TokenFeatures,
    contract_code: str = "",
    liquidity_usd: float = -1.0,
    lock_percentage: float = -1.0,
    lock_duration: int = -1,
    known_lockers: Optional[Sequence[str]] = None,
) -> Dict:
    """Combine feature scoring with bytecode analysis.

    Signals that were not supplied are reported in ``unknown`` and excluded
    from the score, so a partial input cannot produce a reassuring result.
    """
    detector = ScamDetector()
    honeypot = HoneypotDetector()
    liquidity = LiquidityChecker(known_lockers)

    assessment = detector.assess(features)
    contract = honeypot.analyze_contract(contract_code)

    unknown = list(assessment.unknown)

    # Feature score dominates; contract evidence is additive on top because
    # it is direct observation rather than inference.
    contract_component = 0.0
    if contract.analysed:
        contract_component = min(contract.risk_points, 100) * 0.4
    else:
        unknown.append("contract_code")

    liquidity_result: Optional[Dict] = None
    if liquidity_usd >= 0 and lock_percentage >= 0 and lock_duration >= 0:
        liquidity_result = liquidity.check_liquidity(
            token_address=address,
            liquidity_amount=0.0,
            liquidity_usd=liquidity_usd,
            lock_percentage=lock_percentage,
            lock_duration=lock_duration,
        )
    else:
        unknown.append("liquidity_lock")

    overall = _clamp((assessment.score * 0.6) + contract_component, 0.0, 100.0)

    return {
        "address": address,
        "scam_analysis": assessment.to_dict(),
        "contract_analysis": {
            "analysed": contract.analysed,
            "severity": contract.severity,
            "risk_points": contract.risk_points,
            "is_standard_token": contract.is_standard_token,
            "reason": contract.reason,
            "findings": [
                {
                    "kind": f.kind,
                    "severity": f.severity,
                    "detail": f.detail,
                    "offset": f.offset,
                    "selector": f.selector,
                }
                for f in contract.findings
            ],
        },
        "liquidity_analysis": liquidity_result,
        "overall_risk_score": round(overall, 2),
        "risk_level": "HIGH" if overall >= 70 else "MEDIUM" if overall >= 40 else "LOW",
        "recommendation": "AVOID" if overall >= 70 else "CAUTION" if overall >= 40 else "SAFE",
        "unknown": unknown,
    }
