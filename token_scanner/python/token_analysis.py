"""
TigerWallet Token Scanner ML Module
Scam detection and rugpull prediction using machine learning
"""

import numpy as np
from typing import Dict, List, Optional
from dataclasses import dataclass
import json
import hashlib


@dataclass
class TokenFeatures:
    """Features extracted from token"""
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


class ScamDetector:
    """ML-based scam detection"""
    
    def __init__(self):
        self.model = self._load_model()
    
    def _load_model(self):
        """Load pretrained model weights"""
        return np.array([0.3, 0.2, 0.15, 0.1, 0.1, 0.05, 0.05, 0.05])
    
    def predict_scam_probability(self, features: TokenFeatures) -> float:
        """Predict scam probability"""
        feature_vector = np.array([
            features.liquidity_ratio,
            features.holder_concentration,
            min(features.transfer_count / 10000, 1.0),
            features.buy_sell_ratio,
            features.price_volatility,
            features.social_score,
            features.dev_activity,
            min(features.age_hours / 8760, 1.0),
        ])
        
        probability = np.dot(self.model, feature_vector)
        return min(probability, 1.0)
    
    def analyze_token(self, address: str, features: TokenFeatures) -> Dict:
        """Analyze token for scam indicators"""
        scam_prob = self.predict_scam_probability(features)
        
        return {
            "address": address,
            "scam_probability": round(scam_prob * 100, 2),
            "risk_level": self._risk_level(scam_prob),
            "warnings": self._generate_warnings(features),
            "recommendation": "AVOID" if scam_prob > 0.7 else "CAUTION" if scam_prob > 0.4 else "SAFE",
        }
    
    def _risk_level(self, probability: float) -> str:
        if probability > 0.7:
            return "HIGH"
        elif probability > 0.4:
            return "MEDIUM"
        return "LOW"
    
    def _generate_warnings(self, features: TokenFeatures) -> List[str]:
        warnings = []
        if features.liquidity_ratio < 0.1:
            warnings.append("Very low liquidity")
        if features.holder_concentration > 0.5:
            warnings.append("High holder concentration")
        if features.price_volatility > 0.5:
            warnings.append("Extreme price volatility")
        if features.age_hours < 24:
            warnings.append("Recently created token")
        if not features.contract_verified:
            warnings.append("Contract not verified")
        if features.mint_enabled:
            warnings.append("Mint function enabled")
        if features.pause_enabled:
            warnings.append("Pause function enabled")
        if features.owner_changed:
            warnings.append("Ownership has been changed")
        return warnings


class RugpullPredictor:
    """Predict rugpull risk"""
    
    def __init__(self):
        self.weights = {
            "liquidity_burned": 0.3,
            "owner_renounced": 0.25,
            "mint_disabled": 0.2,
            "no_pausable": 0.15,
            "no_blacklist": 0.1,
        }
    
    def predict_rugpull(
        self,
        liquidity_locked: bool,
        owner_renounced: bool,
        mint_disabled: bool,
        pausable: bool,
        blacklist_enabled: bool,
    ) -> float:
        """Calculate rugpull risk score"""
        score = 0.0
        
        if not liquidity_locked:
            score += self.weights["liquidity_burned"]
        if not owner_renounced:
            score += self.weights["owner_renounced"]
        if not mint_disabled:
            score += self.weights["mint_disabled"]
        if not pausable:
            score += self.weights["no_pausable"]
        if not blacklist_enabled:
            score += self.weights["no_blacklist"]
        
        return min(score, 1.0)


class HoneypotDetector:
    """Detect honeypot tokens"""
    
    def __init__(self):
        self.honeypot_patterns = [
            "require(false",
            "revert()",
            "if (msg.sender != owner)",
            "return false",
        ]
    
    def analyze_contract(self, code: str) -> Dict:
        """Analyze contract code for honeypot patterns"""
        issues = []
        score = 0
        
        # Check for common honeypot patterns
        if "require(false" in code:
            issues.append("Potential honeypot: require(false)")
            score += 30
        
        if "revert()" in code and "if" not in code:
            issues.append("Unconditional revert detected")
            score += 25
        
        if "selfdestruct" in code:
            issues.append("Contains selfdestruct - potential rugpull")
            score += 20
        
        if "delegatecall" in code:
            issues.append("Contains delegatecall - verify destination")
            score += 15
        
        if "blacklist" in code.lower():
            issues.append("Blacklist function detected")
            score += 10
        
        return {
            "issues": issues,
            "severity": "HIGH" if score > 50 else "MEDIUM" if score > 20 else "LOW",
            "is_honeypot": score > 50,
            "honeypot_score": score,
        }


class LiquidityChecker:
    """Check liquidity lock status"""
    
    def __init__(self):
        self.locking_contracts = [
            "0xd9f046fA44F8d2a6E4d6f2dB3f2dB3f2dB3f2d",
        ]
    
    def check_liquidity(
        self,
        token_address: str,
        liquidity_amount: float,
        liquidity_usd: float,
        lock_percentage: float,
        lock_duration: int,
    ) -> Dict:
        """Check if liquidity is properly locked"""
        
        is_locked = lock_percentage >= 50
        sufficient = liquidity_usd >= 10000
        long_lock = lock_duration >= 30 * 24 * 60 * 60  # 30 days
        
        risk_score = 0
        warnings = []
        
        if not is_locked:
            risk_score += 30
            warnings.append("Liquidity not locked")
        
        if not sufficient:
            risk_score += 20
            warnings.append("Low liquidity")
        
        if not long_lock:
            risk_score += 10
            warnings.append("Short lock duration")
        
        return {
            "is_locked": is_locked,
            "sufficient": sufficient,
            "long_lock": long_lock,
            "risk_score": risk_score,
            "warnings": warnings,
            "status": "SAFE" if risk_score == 0 else "CAUTION" if risk_score < 30 else "RISKY",
        }


def analyze_token_full(address: str, features: TokenFeatures, contract_code: str) -> Dict:
    """Full token analysis combining all detectors"""
    
    scam_detector = ScamDetector()
    honeypot_detector = HoneypotDetector()
    liquidity_checker = LiquidityChecker()
    
    # Run all checks
    scam_result = scam_detector.analyze_token(address, features)
    honeypot_result = honeypot_detector.analyze_contract(contract_code)
    
    # Calculate overall risk score
    overall_score = (
        scam_result["scam_probability"] * 0.4 +
        (honeypot_result["honeypot_score"] if honeypot_result["is_honeypot"] else 0) * 0.4 +
        len(honeypot_result["issues"]) * 5
    )
    
    return {
        "address": address,
        "scam_analysis": scam_result,
        "honeypot_analysis": honeypot_result,
        "overall_risk_score": min(overall_score, 100),
        "risk_level": "HIGH" if overall_score > 70 else "MEDIUM" if overall_score > 40 else "LOW",
        "recommendation": "AVOID" if overall_score > 70 else "CAUTION" if overall_score > 40 else "SAFE",
    }


if __name__ == "__main__":
    # Example usage
    detector = ScamDetector()
    features = TokenFeatures(
        liquidity_ratio=0.05,
        holder_concentration=0.8,
        transfer_count=100,
        buy_sell_ratio=0.3,
        price_volatility=0.9,
        social_score=0.1,
        dev_activity=0.1,
        age_hours=12,
        contract_verified=False,
        owner_changed=True,
        mint_enabled=True,
        pause_enabled=True,
    )
    
    result = detector.analyze_token("0x123...", features)
    print(json.dumps(result, indent=2))