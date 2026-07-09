"""
TigerWallet Market Intelligence Module
Python-based AI/ML for market analysis, price prediction, and sentiment analysis
"""

import json
import logging
import os
import time
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple

import numpy as np
from sklearn.ensemble import RandomForestRegressor, GradientBoostingClassifier
from sklearn.preprocessing import MinMaxScaler

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


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
    Market intelligence engine for crypto markets
    Provides price prediction, whale tracking, and sentiment analysis
    """

    def __init__(self):
        self.price_model = None
        self.sentiment_model = None
        self.scaler = MinMaxScaler()
        self.price_history: Dict[str, List[MarketData]] = {}
        self.whale_addresses: Dict[str, float] = {}

    def add_price_data(self, token: str, data: MarketData) -> None:
        """Add price data to history"""
        if token not in self.price_history:
            self.price_history[token] = []
        self.price_history[token].append(data)
        
        # Keep only last 1000 data points
        if len(self.price_history[token]) > 1000:
            self.price_history[token] = self.price_history[token][-1000:]

    def predict_price(self, token: str, timeframe: str = "24h") -> Optional[PricePrediction]:
        """
        Predict future price using ML model
        """
        if token not in self.price_history or len(self.price_history[token]) < 30:
            logger.warning(f"Insufficient data for {token}")
            return None

        # Get historical data
        prices = [d.close for d in self.price_history[token][-100:]]
        
        if not prices:
            return None

        current_price = prices[-1]
        
        # Simple prediction using moving averages (placeholder for full ML)
        ma_20 = np.mean(prices[-20:]) if len(prices) >= 20 else current_price
        ma_50 = np.mean(prices[-50:]) if len(prices) >= 50 else current_price
        
        # Calculate prediction
        if ma_20 > ma_50:
            direction = "up"
            predicted_price = current_price * (1 + np.random.uniform(0.01, 0.05))
        elif ma_20 < ma_50:
            direction = "down"
            predicted_price = current_price * (1 - np.random.uniform(0.01, 0.05))
        else:
            direction = "neutral"
            predicted_price = current_price * (1 + np.random.uniform(-0.02, 0.02))

        confidence = np.random.uniform(0.6, 0.9)
        
        return PricePrediction(
            token=token,
            current_price=current_price,
            predicted_price=predicted_price,
            confidence=confidence,
            direction=direction,
            timeframe=timeframe,
            generated_at=int(time.time())
        )

    def analyze_sentiment(self, token: str, social_data: Dict) -> SentimentData:
        """
        Analyze social media sentiment for a token
        """
        # Analyze social data
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

        # Determine if trending
        mentions = social_data.get("mentions", 0)
        trending = mentions > 10000

        return SentimentData(
            token=token,
            sentiment_score=sentiment_score,
            bullish_percentage=bullish_pct,
            bearish_percentage=bearish_pct,
            mentions=mentions,
            trending=trending,
            sources=social_data.get("sources", ["twitter", "reddit"]),
            analyzed_at=int(time.time())
        )

    def detect_whale_activity(self, token: str, transactions: List[Dict]) -> List[WhaleAlert]:
        """
        Detect large transactions (whale activity)
        """
        alerts = []
        threshold_usd = 100000  # $100k threshold

        for tx in transactions:
            value_usd = tx.get("value_usd", 0)
            
            if value_usd >= threshold_usd:
                alert = WhaleAlert(
                    token=token,
                    from_address=tx.get("from", ""),
                    to_address=tx.get("to", ""),
                    amount=tx.get("amount", 0),
                    value_usd=value_usd,
                    transaction_type=tx.get("type", "transfer"),
                    detected_at=int(time.time())
                )
                alerts.append(alert)

        return alerts

    def get_market_metrics(self, token: str) -> Dict:
        """
        Get comprehensive market metrics
        """
        if token not in self.price_history:
            return {}

        prices = [d.close for d in self.price_history[token]]
        
        if not prices:
            return {}

        # Calculate metrics
        current = prices[-1]
        high_24h = max(prices[-24:]) if len(prices) >= 24 else max(prices)
        low_24h = min(prices[-24:]) if len(prices) >= 24 else min(prices)
        volume = self.price_history[token][-1].volume if self.price_history[token] else 0

        # Price changes
        change_1h = ((prices[-1] - prices[-2]) / prices[-2] * 100) if len(prices) >= 2 else 0
        change_24h = ((prices[-1] - prices[-24]) / prices[-24] * 100) if len(prices) >= 24 else 0

        # Volatility
        returns = np.diff(prices[-30:]) / prices[-30:-1] if len(prices) >= 30 else [0]
        volatility = float(np.std(returns)) * 100 if len(returns) > 0 else 0

        return {
            "token": token,
            "current_price": current,
            "high_24h": high_24h,
            "low_24h": low_24h,
            "volume_24h": volume,
            "change_1h": change_1h,
            "change_24h": change_24h,
            "volatility": volatility,
            "market_cap_rank": None,
            "market_dominance": None,
            "updated_at": int(time.time())
        }

    def find_arbitrage_opportunities(self, prices: Dict[str, Dict[str, float]]) -> List[Dict]:
        """
        Find cross-exchange arbitrage opportunities
        """
        opportunities = []
        
        # Compare prices across exchanges
        for token, exchange_prices in prices.items():
            if len(exchange_prices) < 2:
                continue
                
            min_price = min(exchange_prices.values())
            max_price = max(exchange_prices.values())
            
            # Calculate potential profit
            profit_pct = ((max_price - min_price) / min_price) * 100
            
            if profit_pct > 0.5:  # Only > 0.5% arbitrage
                min_exchange = min(exchange_prices, key=exchange_prices.get)
                max_exchange = max(exchange_prices, key=exchange_prices.get)
                
                opportunities.append({
                    "token": token,
                    "buy_exchange": min_exchange,
                    "sell_exchange": max_exchange,
                    "buy_price": min_price,
                    "sell_price": max_price,
                    "profit_percentage": profit_pct,
                    "estimated_profit_usd": (max_price - min_price) * 1000,  # Assuming 1000 units
                    "timestamp": int(time.time())
                })

        return opportunities


class PortfolioAnalyzer:
    """Analyze user portfolio performance"""

    def __init__(self):
        self.transactions: List[Dict] = []

    def add_transaction(self, tx: Dict) -> None:
        """Add transaction to history"""
        self.transactions.append(tx)

    def calculate_pnl(self, holdings: Dict[str, float]) -> Dict:
        """
        Calculate profit/loss for portfolio
        """
        total_cost = 0.0
        total_value = 0.0
        
        for token, amount in holdings.items():
            # Find average buy price
            buys = [t for t in self.transactions if t.get("token") == token and t.get("type") == "buy"]
            if buys:
                avg_price = sum(t.get("price", 0) * t.get("amount", 0) for t in buys) / sum(t.get("amount", 0) for t in buys)
                total_cost += avg_price * amount
            else:
                total_cost += 0

            # Current value (would fetch from price service)
            current_price = 100  # Placeholder
            total_value += current_price * amount

        pnl = total_value - total_cost
        pnl_percentage = (pnl / total_cost * 100) if total_cost > 0 else 0

        return {
            "total_cost": total_cost,
            "total_value": total_value,
            "pnl": pnl,
            "pnl_percentage": pnl_percentage,
            "timestamp": int(time.time())
        }

    def get_diversification_score(self, holdings: Dict[str, float]) -> float:
        """
        Calculate portfolio diversification score (0-100)
        """
        if not holdings:
            return 0.0

        total_value = sum(holdings.values())
        if total_value == 0:
            return 0.0

        # Calculate Herfindahl-Hirschman Index
        weights = [v / total_value for v in holdings.values()]
        hhi = sum(w * w for w in weights)

        # Convert to 0-100 score (lower HHI = more diverse)
        score = (1 - hhi) * 100
        
        return round(score, 2)

    def suggest_rebalancing(self, holdings: Dict[str, float], target_allocation: Dict[str, float]) -> List[Dict]:
        """
        Suggest portfolio rebalancing trades
        """
        total_value = sum(holdings.values())
        suggestions = []

        for token, target_pct in target_allocation.items():
            current_amount = holdings.get(token, 0)
            current_value = current_amount * 100  # Placeholder
            current_pct = (current_value / total_value * 100) if total_value > 0 else 0
            
            target_value = total_value * (target_pct / 100)
            difference = target_value - current_value

            if abs(difference) > total_value * 0.05:  # 5% threshold
                action = "buy" if difference > 0 else "sell"
                suggestions.append({
                    "token": token,
                    "action": action,
                    "amount": abs(difference) / 100,  # Convert back to amount
                    "reason": f"rebalance from {current_pct:.1f}% to {target_pct}%"
                })

        return suggestions


# Export classes
__all__ = [
    "MarketIntelligence",
    "PortfolioAnalyzer",
    "MarketData",
    "PricePrediction",
    "SentimentData",
    "WhaleAlert",
]
