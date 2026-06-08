"""
TigerWallet Token Scanner ML Module
Scam detection and rugpull prediction using machine learning
"""

import numpy as np
from typing import Dict, List, Optional
from dataclasses import dataclass


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


class ScamDetector:
    """ML-based scam detection"""
    
    def __init__(self):
        self.model = self._load_model()
    
    def _load_model(self):
        """Load pretrained model"""
        # Simplified model weights
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
            min(features.age_hours / 8760, 1.0),  # 1 year
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


def analyze_contract(code: str) -> Dict:
    """Analyze contract code for security issues"""
    issues = []
    
    # Check for common honeypot patterns
    if "require(false" in code:
        issues.append("Potential honeypot: require(false)")
    if "revert()" in code and "if" not in code:
        issues.append("Unconditional revert")
    if "selfdestruct" in code:
        issues.append("Contains selfdestruct")
    if "delegatecall" in code:
        issues.append("Contains delegatecall - verify destination")
    
    return {
        "issues": issues,
        "severity": "HIGH" if len(issues) > 2 else "MEDIUM" if issues else "LOW",
        "safe": len(issues) == 0,
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
    )
    
    result = detector.analyze_token("0x123...", features)
    print(result)