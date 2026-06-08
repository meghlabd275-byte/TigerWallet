import React, { useState, useEffect } from 'react';

interface TradesFeedProps {
  symbol: string;
}

interface Trade {
  id: string;
  price: string;
  quantity: string;
  time: number;
  isBuyerMaker: boolean;
}

export const TradesFeed: React.FC<TradesFeedProps> = ({ symbol }) => {
  const [trades, setTrades] = useState<Trade[]>([]);

  useEffect(() => {
    // Fetch initial trades
    const fetchTrades = async () => {
      try {
        const response = await fetch(`/api/v1/market/trades/${symbol}?limit=50`);
        if (response.ok) {
          const data = await response.json();
          setTrades(data);
        }
      } catch (error) {
        console.error('Failed to fetch trades:', error);
      }
    };

    fetchTrades();
  }, [symbol]);

  return (
    <div className="trades-feed">
      <h3>Recent Trades</h3>
      <div className="trades-list">
        {trades.map((trade) => (
          <div key={trade.id} className={`trade-row ${trade.isBuyerMaker ? 'sell' : 'buy'}`}>
            <span className="price">{parseFloat(trade.price).toFixed(2)}</span>
            <span className="quantity">{parseFloat(trade.quantity).toFixed(4)}</span>
            <span className="time">{new Date(trade.time).toLocaleTimeString()}</span>
          </div>
        ))}
      </div>
    </div>
  );
};