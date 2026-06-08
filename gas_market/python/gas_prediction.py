"""
TigerWallet Gas Price Prediction
ML-based gas price forecasting
"""

import numpy as np
from typing import List, Dict
from dataclasses import dataclass


@dataclass
class GasDataPoint:
    timestamp: int
    gas_price: float
    block_number: int
    base_fee: float


class GasPredictor:
    """Gas price prediction using historical data"""
    
    def __init__(self, lookback_hours: int = 168):
        self.lookback = lookback_hours
        self.model = self._train_model()
    
    def _train_model(self):
        """Train prediction model"""
        # Simplified linear model
        return {
            "slope": 0.8,
            "intercept": 15.0,
            "volatility": 0.2,
        }
    
    def predict(self, hours_ahead: int = 24) -> Dict:
        """Predict gas price"""
        base = self.model["intercept"]
        slope = self.model["slope"]
        
        predictions = []
        for h in range(hours_ahead):
            predicted = base + (slope * h / 24)
            predictions.append({
                "hours_ahead": h,
                "predicted_gas": round(predicted, 2),
                "confidence": self._confidence(h),
            })
        
        return {
            "predictions": predictions,
            "average_gas": round(np.mean([p["predicted_gas"] for p in predictions]), 2),
            "congestion_level": self._congestion_level(predictions[0]["predicted_gas"]),
        }
    
    def _confidence(self, hours_ahead: int) -> float:
        """Calculate prediction confidence"""
        confidence = 1.0 - (hours_ahead / 168) * 0.5
        return max(confidence, 0.5)
    
    def _congestion_level(self, gas_price: float) -> str:
        if gas_price < 20:
            return "LOW"
        elif gas_price < 50:
            return "MEDIUM"
        return "HIGH"
    
    def optimize_gas(&self, budget_gwei: float, urgency: str) -> Dict:
        """Optimize gas spending"""
        if urgency == "high":
            return {"gas_price": budget_gwei, "wait_time": 0}
        elif urgency == "medium":
            return {"gas_price": budget_gwei * 0.8, "wait_time": 60}
        else:
            return {"gas_price": budget_gwei * 0.5, "wait_time": 300}


def calculate_savings(current_gas: float, optimized_gas: float, tx_count: int) -> Dict:
    """Calculate gas savings"""
    savings_per_tx = (current_gas - optimized_gas) * 21000
    total_savings = savings_per_tx * tx_count
    
    return {
        "savings_per_tx_gwei": savings_per_tx,
        "total_savings_gwei": total_savings,
        "total_savings_usd": round(total_savings * 1800 / 1e9, 2),
    }


if __name__ == "__main__":
    predictor = GasPredictor()
    result = predictor.predict(hours_ahead=24)
    print(result)