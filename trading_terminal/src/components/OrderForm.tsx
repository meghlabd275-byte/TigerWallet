import React, { useState, useCallback } from 'react';
import { OrderRequest } from '../services/perpetuals';

interface OrderFormProps {
  currentPrice: number;
  symbol: string;
  account: any;
  onSubmit: (order: OrderRequest) => Promise<any>;
}

export const OrderForm: React.FC<OrderFormProps> = ({ currentPrice, symbol, account, onSubmit }) => {
  const [side, setSide] = useState<'BUY' | 'SELL'>('BUY');
  const [orderType, setOrderType] = useState<'LIMIT' | 'MARKET'>('LIMIT');
  const [price, setPrice] = useState<string>('');
  const [quantity, setQuantity] = useState<string>('');
  const [leverage, setLeverage] = useState<string>('10');
  const [marginType, setMarginType] = useState<'CROSS' | 'ISOLATED'>('CROSS');
  const [reduceOnly, setReduceOnly] = useState(false);
  const [postOnly, setPostOnly] = useState(false);
  const [timeInForce, setTimeInForce] = useState<'GTC' | 'IOC' | 'FOK'>('GTC');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const order: OrderRequest = {
        symbol,
        userId: account?.userId || '',
        side,
        orderType,
        price: orderType === 'LIMIT' ? price : undefined,
        quantity,
        leverage,
        marginType,
        reduceOnly,
        postOnly,
        timeInForce
      };

      await onSubmit(order);
      
      // Reset form
      setPrice('');
      setQuantity('');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [side, orderType, price, quantity, leverage, marginType, reduceOnly, postOnly, timeInForce, symbol, account, onSubmit]);

  const totalValue = parseFloat(quantity || '0') * parseFloat(orderType === 'LIMIT' ? price || currentPrice.toString() : currentPrice.toString());
  const requiredMargin = totalValue / parseFloat(leverage || '10');

  return (
    <form className="order-form" onSubmit={handleSubmit}>
      <div className="order-type-tabs">
        <button
          type="button"
          className={`tab ${side === 'BUY' ? 'active buy' : ''}`}
          onClick={() => setSide('BUY')}
        >
          Buy
        </button>
        <button
          type="button"
          className={`tab ${side === 'SELL' ? 'active sell' : ''}`}
          onClick={() => setSide('SELL')}
        >
          Sell
        </button>
      </div>

      <div className="order-type-selector">
        <select value={orderType} onChange={(e) => setOrderType(e.target.value as any)}>
          <option value="LIMIT">Limit</option>
          <option value="MARKET">Market</option>
        </select>
        <select value={timeInForce} onChange={(e) => setTimeInForce(e.target.value as any)}>
          <option value="GTC">GTC</option>
          <option value="IOC">IOC</option>
          <option value="FOK">FOK</option>
        </select>
      </div>

      {orderType === 'LIMIT' && (
        <div className="form-group">
          <label>Price</label>
          <input
            type="number"
            value={price}
            onChange={(e) => setPrice(e.target.value)}
            placeholder={currentPrice.toString()}
            step="0.5"
          />
        </div>
      )}

      <div className="form-group">
        <label>Quantity</label>
        <input
          type="number"
          value={quantity}
          onChange={(e) => setQuantity(e.target.value)}
          placeholder="0.001"
          step="0.001"
        />
      </div>

      <div className="form-group">
        <label>Leverage</label>
        <div className="leverage-selector">
          {[1, 2, 5, 10, 20, 50, 100].map((lev) => (
            <button
              key={lev}
              type="button"
              className={`leverage-btn ${leverage === lev.toString() ? 'active' : ''}`}
              onClick={() => setLeverage(lev.toString())}
            >
              {lev}x
            </button>
          ))}
        </div>
      </div>

      <div className="form-group">
        <label>Margin Type</label>
        <select value={marginType} onChange={(e) => setMarginType(e.target.value as any)}>
          <option value="CROSS">Cross</option>
          <option value="ISOLATED">Isolated</option>
        </select>
      </div>

      <div className="form-options">
        <label>
          <input
            type="checkbox"
            checked={reduceOnly}
            onChange={(e) => setReduceOnly(e.target.checked)}
          />
          Reduce Only
        </label>
        <label>
          <input
            type="checkbox"
            checked={postOnly}
            onChange={(e) => setPostOnly(e.target.checked)}
          />
          Post Only
        </label>
      </div>

      <div className="order-summary">
        <div className="summary-row">
          <span>Total Value</span>
          <span>${totalValue.toFixed(2)}</span>
        </div>
        <div className="summary-row">
          <span>Required Margin</span>
          <span>${requiredMargin.toFixed(2)}</span>
        </div>
        <div className="summary-row">
          <span>Max Leverage</span>
          <span>{leverage}x</span>
        </div>
      </div>

      {error && <div className="error-message">{error}</div>}

      <button
        type="submit"
        className={`submit-btn ${side.toLowerCase()}`}
        disabled={loading || !quantity}
      >
        {loading ? 'Submitting...' : `${side} ${symbol}`}
      </button>
    </form>
  );
};