import React from 'react';

interface OrderBookProps {
  bids: [string, string][];
  asks: [string, string][];
  currentPrice: number;
}

export const OrderBook: React.FC<OrderBookProps> = ({ bids, asks, currentPrice }) => {
  const maxQty = Math.max(
    ...bids.map(b => parseFloat(b[1])),
    ...asks.map(a => parseFloat(a[1])),
    0.0001
  );

  return (
    <div className="orderbook">
      <div className="orderbook-header">
        <span>Price (USD)</span>
        <span>Quantity</span>
        <span>Total</span>
      </div>
      
      <div className="orderbook-asks">
        {asks.slice(0, 10).reverse().map(([price, qty], i) => {
          const total = parseFloat(price) * parseFloat(qty);
          const depth = parseFloat(qty) / maxQty;
          
          return (
            <div key={`ask-${i}`} className="orderbook-row ask">
              <div className="depth-bar" style={{ width: `${depth * 100}%` }} />
              <span className="price">{parseFloat(price).toFixed(2)}</span>
              <span className="quantity">{parseFloat(qty).toFixed(4)}</span>
              <span className="total">{total.toFixed(2)}</span>
            </div>
          );
        })}
      </div>
      
      <div className="orderbook-spread">
        <span className="current-price">
          ${currentPrice.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
        </span>
        <span className="spread">
          Spread: ${bids[0] && asks[0] ? (parseFloat(asks[0][0]) - parseFloat(bids[0][0])).toFixed(2) : '0.00'}
        </span>
      </div>
      
      <div className="orderbook-bids">
        {bids.slice(0, 10).map(([price, qty], i) => {
          const total = parseFloat(price) * parseFloat(qty);
          const depth = parseFloat(qty) / maxQty;
          
          return (
            <div key={`bid-${i}`} className="orderbook-row bid">
              <div className="depth-bar" style={{ width: `${depth * 100}%` }} />
              <span className="price">{parseFloat(price).toFixed(2)}</span>
              <span className="quantity">{parseFloat(qty).toFixed(4)}</span>
              <span className="total">{total.toFixed(2)}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
};