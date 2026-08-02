// Future Trading Page - Perpetual Futures Trading
// Supports USDT/any tokens, USDC/any tokens, Cross/Isolated Margin

import React, { useState, useEffect, useCallback } from 'react';
import './FuturesTradingPage.css';

interface TradingPair {
  id: string;
  symbol: string;
  base: string;
  quote: string;
  price: number;
  change24h: number;
  volume24h: number;
  high24h: number;
  low24h: number;
  isPreInstalled: boolean;
}

interface Position {
  id: string;
  symbol: string;
  side: 'long' | 'short';
  size: number;
  entryPrice: number;
  markPrice: number;
  leverage: number;
  margin: number;
  pnl: number;
  pnlPercent: number;
}

interface Order {
  id: string;
  symbol: string;
  side: 'buy' | 'sell';
  type: 'market' | 'limit' | 'stop';
  size: number;
  price: number;
  filled: number;
  status: 'open' | 'filled' | 'cancelled';
  timestamp: number;
}

const TOP_PAIRS: TradingPair[] = [
  { id: '1', symbol: 'BTC/USDT', base: 'BTC', quote: 'USDT', price: 43250.00, change24h: 2.5, volume24h: 1250000000, high24h: 44000, low24h: 42500, isPreInstalled: true },
  { id: '2', symbol: 'ETH/USDT', base: 'ETH', quote: 'USDT', price: 2280.00, change24h: 1.8, volume24h: 850000000, high24h: 2320, low24h: 2240, isPreInstalled: true },
  { id: '3', symbol: 'BNB/USDT', base: 'BNB', quote: 'USDT', price: 312.50, change24h: -0.5, volume24h: 180000000, high24h: 318, low24h: 308, isPreInstalled: true },
  { id: '4', symbol: 'SOL/USDT', base: 'SOL', quote: 'USDT', price: 98.75, change24h: 5.2, volume24h: 320000000, high24h: 102, low24h: 94, isPreInstalled: true },
  { id: '5', symbol: 'XRP/USDT', base: 'XRP', quote: 'USDT', price: 0.62, change24h: -1.2, volume24h: 150000000, high24h: 0.65, low24h: 0.61, isPreInstalled: true },
  { id: '6', symbol: 'DOGE/USDT', base: 'DOGE', quote: 'USDT', price: 0.082, change24h: 3.1, volume24h: 95000000, high24h: 0.085, low24h: 0.079, isPreInstalled: true },
  { id: '7', symbol: 'ADA/USDT', base: 'ADA', quote: 'USDT', price: 0.58, change24h: 0.8, volume24h: 72000000, high24h: 0.60, low24h: 0.57, isPreInstalled: true },
  { id: '8', symbol: 'AVAX/USDT', base: 'AVAX', quote: 'USDT', price: 38.20, change24h: 2.1, volume24h: 89000000, high24h: 39.5, low24h: 37.2, isPreInstalled: true },
  { id: '9', symbol: 'DOT/USDT', base: 'DOT', quote: 'USDT', price: 7.85, change24h: -0.8, volume24h: 45000000, high24h: 8.10, low24h: 7.65, isPreInstalled: true },
  { id: '10', symbol: 'LINK/USDT', base: 'LINK', quote: 'USDT', price: 14.50, change24h: 1.5, volume24h: 68000000, high24h: 14.90, low24h: 14.20, isPreInstalled: true },
  { id: '11', symbol: 'MATIC/USDT', base: 'MATIC', quote: 'USDT', price: 0.92, change24h: 2.8, volume24h: 55000000, high24h: 0.95, low24h: 0.89, isPreInstalled: true },
  { id: '12', symbol: 'LTC/USDT', base: 'LTC', quote: 'USDT', price: 72.30, change24h: 1.2, volume24h: 42000000, high24h: 74, low24h: 71, isPreInstalled: true },
  { id: '13', symbol: 'UNI/USDT', base: 'UNI', quote: 'USDT', price: 6.25, change24h: -0.5, volume24h: 28000000, high24h: 6.45, low24h: 6.10, isPreInstalled: true },
  { id: '14', symbol: 'ATOM/USDT', base: 'ATOM', quote: 'USDT', price: 10.45, change24h: 1.8, volume24h: 35000000, high24h: 10.80, low24h: 10.20, isPreInstalled: true },
  { id: '15', symbol: 'XLM/USDT', base: 'XLM', quote: 'USDT', price: 0.125, change24h: 0.9, volume24h: 32000000, high24h: 0.130, low24h: 0.122, isPreInstalled: true },
  { id: '16', symbol: 'NEAR/USDT', base: 'NEAR', quote: 'USDT', price: 3.25, change24h: 4.2, volume24h: 48000000, high24h: 3.40, low24h: 3.10, isPreInstalled: true },
  { id: '17', symbol: 'APT/USDT', base: 'APT', quote: 'USDT', price: 9.80, change24h: -2.1, volume24h: 52000000, high24h: 10.20, low24h: 9.50, isPreInstalled: true },
  { id: '18', symbol: 'ARB/USDT', base: 'ARB', quote: 'USDT', price: 1.12, change24h: 1.5, volume24h: 38000000, high24h: 1.18, low24h: 1.08, isPreInstalled: true },
  { id: '19', symbol: 'OP/USDT', base: 'OP', quote: 'USDT', price: 2.45, change24h: 2.8, volume24h: 29000000, high24h: 2.55, low24h: 2.35, isPreInstalled: true },
  { id: '20', symbol: 'INJ/USDT', base: 'INJ', quote: 'USDT', price: 35.50, change24h: 5.5, volume24h: 65000000, high24h: 37, low24h: 33.5, isPreInstalled: true },
];

// Backend API URL for perpetual trading
const API_BASE_URL = 'https://api.tigerwallet.com/v1/perpetual';

const FuturesTradingPage: React.FC = () => {
  const [selectedPair, setSelectedPair] = useState<TradingPair | null>(null);
  const [pairs, setPairs] = useState<TradingPair[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [quoteCurrency, setQuoteCurrency] = useState<'USDT' | 'USDC'>('USDT');
  const [leverage, setLeverage] = useState(10);
  const [orderType, setOrderType] = useState<'market' | 'limit' | 'stop'>('market');
  const [orderSide, setOrderSide] = useState<'buy' | 'sell'>('buy');
  const [orderSize, setOrderSize] = useState('');
  const [orderPrice, setOrderPrice] = useState('');
  const [marginMode, setMarginMode] = useState<'cross' | 'isolated'>('cross');
  const [positions, setPositions] = useState<Position[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [activeTab, setActiveTab] = useState<'trade' | 'positions' | 'orders'>('trade');
  const [showAllPairs, setShowAllPairs] = useState(false);
  const [walletBalance, setWalletBalance] = useState({ USDT: 0, USDC: 0 });

  // Load trading pairs from backend
  useEffect(() => {
    const loadData = async () => {
      setLoading(true);
      setError(null);
      try {
        const token = localStorage.getItem('user_token');
        
        // Load pairs
        const pairsRes = await fetch(`${API_BASE_URL}/pairs?quote=${quoteCurrency}`, {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        });
        
        if (pairsRes.ok) {
          const pairsData = await pairsRes.json();
          if (pairsData.pairs) {
            setPairs(pairsData.pairs);
            if (pairsData.pairs.length > 0 && !selectedPair) {
              setSelectedPair(pairsData.pairs[0]);
            }
          }
        }
        
        // Load positions
        const posRes = await fetch(`${API_BASE_URL}/positions`, {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        });
        if (posRes.ok) {
          const posData = await posRes.json();
          setPositions(posData.positions || []);
        }
        
        // Load orders
        const ordersRes = await fetch(`${API_BASE_URL}/orders`, {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        });
        if (ordersRes.ok) {
          const ordersData = await ordersRes.json();
          setOrders(ordersData.orders || []);
        }
        
        // Load balance
        const balRes = await fetch('https://api.tigerwallet.com/v1/wallets/balance', {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        });
        if (balRes.ok) {
          const balData = await balRes.json();
          setWalletBalance({
            USDT: balData.balances?.USDT || 0,
            USDC: balData.balances?.USDC || 0
          });
        }
      } catch (err) {
        console.error('Failed to load data:', err);
        setError('Unable to connect to trading service. Using offline mode.');
        // Fall back to TOP_PAIRS in offline mode
        setPairs(TOP_PAIRS);
        if (!selectedPair) setSelectedPair(TOP_PAIRS[0]);
      } finally {
        setLoading(false);
      }
    };
    
    loadData();
  }, [quoteCurrency]);

  const filteredPairs = pairs.filter(pair => 
    pair.quote === quoteCurrency &&
    (pair.symbol.toLowerCase().includes(searchTerm.toLowerCase()) || 
     pair.base.toLowerCase().includes(searchTerm.toLowerCase()))
  );

  const displayPairs = showAllPairs ? filteredPairs : filteredPairs.filter(p => p.isPreInstalled);

  const handleOrderSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedPair) return;
    
    try {
      const token = localStorage.getItem('user_token');
      const response = await fetch(`${API_BASE_URL}/orders`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {})
        },
        body: JSON.stringify({
          symbol: selectedPair.symbol,
          side: orderSide,
          orderType: orderType,
          size: parseFloat(orderSize),
          price: orderType === 'market' ? selectedPair.price : parseFloat(orderPrice),
          leverage: leverage,
          marginMode: marginMode
        })
      });
      
      if (response.ok) {
        const data = await response.json();
        if (data.order) {
          setOrders([data.order, ...orders]);
        }
        // Reload positions after order
        loadPositions();
        // Reload balance
        loadBalance();
      }
    } catch (err) {
      console.error('Failed to place order:', err);
    }
    setOrderSize('');
    setOrderPrice('');
  };

  const closePosition = (positionId: string) => {
    setPositions(positions.filter(p => p.id !== positionId));
  };

  const calculateOrderValue = () => {
    const size = parseFloat(orderSize) || 0;
    const price = orderType === 'market' ? selectedPair.price : parseFloat(orderPrice) || 0;
    return size * price;
  };

  const calculateRequiredMargin = () => {
    const value = calculateOrderValue();
    return value / leverage;
  };

  return (
    <div className="futures-trading-page">
      <div className="trading-layout">
        {/* Left Panel - Markets */}
        <div className="markets-panel">
          <div className="markets-header">
            <div className="quote-selector">
              <button 
                className={quoteCurrency === 'USDT' ? 'active' : ''}
                onClick={() => setQuoteCurrency('USDT')}
              >
                USDT
              </button>
              <button 
                className={quoteCurrency === 'USDC' ? 'active' : ''}
                onClick={() => setQuoteCurrency('USDC')}
              >
                USDC
              </button>
            </div>
            <input
              type="text"
              placeholder="Search pairs..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
          </div>
          
          <div className="pairs-list-header">
            <span>Pair</span>
            <span>Price</span>
            <span>24h %</span>
          </div>
          
          <div className="pairs-list">
            {displayPairs.map(pair => (
              <div
                key={pair.id}
                className={`pair-item ${selectedPair.id === pair.id ? 'active' : ''} ${pair.isPreInstalled ? 'pre-installed' : ''}`}
                onClick={() => setSelectedPair(pair)}
              >
                <div className="pair-info">
                  <span className="pair-symbol">{pair.symbol}</span>
                  {pair.isPreInstalled && <span className="pre-installed-badge">★</span>}
                </div>
                <div className="pair-price">
                  <span className="price">{pair.price.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: pair.price < 1 ? 6 : 2 })}</span>
                  <span className={`change ${pair.change24h >= 0 ? 'positive' : 'negative'}`}>
                    {pair.change24h >= 0 ? '+' : ''}{pair.change24h.toFixed(2)}%
                  </span>
                </div>
              </div>
            ))}
          </div>
          
          {filteredPairs.length > (showAllPairs ? filteredPairs.length : 20) && (
            <button 
              className="show-more-btn"
              onClick={() => setShowAllPairs(!showAllPairs)}
            >
              {showAllPairs ? 'Show Top Pairs' : `Show All (${filteredPairs.length}+)`}
            </button>
          )}
        </div>

        {/* Center Panel - Chart & Trading */}
        <div className="trading-panel">
          <div className="trading-header">
            <div className="current-pair">
              <h2>{selectedPair.symbol}</h2>
              <span className={`price ${selectedPair.change24h >= 0 ? 'positive' : 'negative'}`}>
                {selectedPair.price.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: selectedPair.price < 1 ? 6 : 2 })}
              </span>
              <span className={`change ${selectedPair.change24h >= 0 ? 'positive' : 'negative'}`}>
                {selectedPair.change24h >= 0 ? '+' : ''}{selectedPair.change24h.toFixed(2)}%
              </span>
            </div>
            <div className="pair-stats">
              <div className="stat">
                <span className="label">24h High</span>
                <span className="value">{selectedPair.high24h.toLocaleString()}</span>
              </div>
              <div className="stat">
                <span className="label">24h Low</span>
                <span className="value">{selectedPair.low24h.toLocaleString()}</span>
              </div>
              <div className="stat">
                <span className="label">24h Vol</span>
                <span className="value">${(selectedPair.volume24h / 1000000).toFixed(2)}M</span>
              </div>
            </div>
          </div>

          {/* Simple Chart Representation */}
          <div className="chart-container">
            <div className="chart-placeholder">
              <div className="chart-title">{selectedPair.symbol} Price Chart</div>
              <div className="chart-info">
                <span>Time: 1H 4H 1D 1W 1M</span>
                <span>Indicators: MA, RSI, MACD</span>
              </div>
              <div className="chart-visual">
                {/* Simulated price line */}
                <svg viewBox="0 0 800 200" className="price-chart">
                  <defs>
                    <linearGradient id="chartGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                      <stop offset="0%" stopColor="#00C853" stopOpacity="0.3" />
                      <stop offset="100%" stopColor="#00C853" stopOpacity="0" />
                    </linearGradient>
                  </defs>
                  <path 
                    d="M0,150 Q100,120 200,140 T400,100 T600,80 T800,60" 
                    fill="none" 
                    stroke="#00C853" 
                    strokeWidth="2"
                  />
                  <path 
                    d="M0,150 Q100,120 200,140 T400,100 T600,80 T800,60 L800,200 L0,200 Z" 
                    fill="url(#chartGradient)"
                  />
                </svg>
              </div>
            </div>
          </div>

          {/* Order Form */}
          <div className="order-form-container">
            <div className="tabs">
              <button 
                className={activeTab === 'trade' ? 'active' : ''} 
                onClick={() => setActiveTab('trade')}
              >
                Trade
              </button>
              <button 
                className={activeTab === 'positions' ? 'active' : ''} 
                onClick={() => setActiveTab('positions')}
              >
                Positions ({positions.length})
              </button>
              <button 
                className={activeTab === 'orders' ? 'active' : ''} 
                onClick={() => setActiveTab('orders')}
              >
                Orders ({orders.length})
              </button>
            </div>

            {activeTab === 'trade' && (
              <form className="order-form" onSubmit={handleOrderSubmit}>
                <div className="margin-mode-selector">
                  <button
                    type="button"
                    className={marginMode === 'cross' ? 'active' : ''}
                    onClick={() => setMarginMode('cross')}
                  >
                    Cross
                  </button>
                  <button
                    type="button"
                    className={marginMode === 'isolated' ? 'active' : ''}
                    onClick={() => setMarginMode('isolated')}
                  >
                    Isolated
                  </button>
                </div>

                <div className="balance-info">
                  <span>Available: {quoteCurrency === 'USDT' ? walletBalance.USDT.toLocaleString() : walletBalance.USDC.toLocaleString()} {quoteCurrency}</span>
                </div>

                <div className="order-type-selector">
                  <button
                    type="button"
                    className={orderType === 'market' ? 'active' : ''}
                    onClick={() => setOrderType('market')}
                  >
                    Market
                  </button>
                  <button
                    type="button"
                    className={orderType === 'limit' ? 'active' : ''}
                    onClick={() => setOrderType('limit')}
                  >
                    Limit
                  </button>
                  <button
                    type="button"
                    className={orderType === 'stop' ? 'active' : ''}
                    onClick={() => setOrderType('stop')}
                  >
                    Stop
                  </button>
                </div>

                <div className="order-side-selector">
                  <button
                    type="button"
                    className={`buy ${orderSide === 'buy' ? 'active' : ''}`}
                    onClick={() => setOrderSide('buy')}
                  >
                    Buy/Long
                  </button>
                  <button
                    type="button"
                    className={`sell ${orderSide === 'sell' ? 'active' : ''}`}
                    onClick={() => setOrderSide('sell')}
                  >
                    Sell/Short
                  </button>
                </div>

                <div className="leverage-slider">
                  <label>Leverage: {leverage}x</label>
                  <input
                    type="range"
                    min="1"
                    max="125"
                    value={leverage}
                    onChange={(e) => setLeverage(parseInt(e.target.value))}
                  />
                  <div className="leverage-presets">
                    {[1, 5, 10, 25, 50, 75, 100, 125].map(l => (
                      <button
                        key={l}
                        type="button"
                        className={leverage === l ? 'active' : ''}
                        onClick={() => setLeverage(l)}
                      >
                        {l}x
                      </button>
                    ))}
                  </div>
                </div>

                {orderType !== 'market' && (
                  <div className="form-group">
                    <label>Price</label>
                    <input
                      type="number"
                      value={orderPrice}
                      onChange={(e) => setOrderPrice(e.target.value)}
                      placeholder="Enter price"
                      step="0.01"
                    />
                  </div>
                )}

                <div className="form-group">
                  <label>Size ({selectedPair.base})</label>
                  <input
                    type="number"
                    value={orderSize}
                    onChange={(e) => setOrderSize(e.target.value)}
                    placeholder="Enter size"
                    step="0.001"
                  />
                </div>

                <div className="order-summary">
                  <div className="summary-row">
                    <span>Order Value:</span>
                    <span>${calculateOrderValue().toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
                  </div>
                  <div className="summary-row">
                    <span>Required Margin:</span>
                    <span>${calculateRequiredMargin().toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
                  </div>
                  <div className="summary-row">
                    <span>Leverage:</span>
                    <span>{leverage}x</span>
                  </div>
                </div>

                <button
                  type="submit"
                  className={`submit-btn ${orderSide}`}
                >
                  {orderSide === 'buy' ? 'Buy' : 'Sell'} {selectedPair.base}
                </button>
              </form>
            )}

            {activeTab === 'positions' && (
              <div className="positions-list">
                {positions.length === 0 ? (
                  <div className="empty-state">No open positions</div>
                ) : (
                  positions.map(position => (
                    <div key={position.id} className="position-item">
                      <div className="position-header">
                        <span className={`side ${position.side}`}>{position.side.toUpperCase()}</span>
                        <span className="symbol">{position.symbol}</span>
                        <button className="close-btn" onClick={() => closePosition(position.id)}>×</button>
                      </div>
                      <div className="position-details">
                        <div className="detail">
                          <span className="label">Size</span>
                          <span className="value">{position.size}</span>
                        </div>
                        <div className="detail">
                          <span className="label">Entry</span>
                          <span className="value">{position.entryPrice.toLocaleString()}</span>
                        </div>
                        <div className="detail">
                          <span className="label">Mark</span>
                          <span className="value">{position.markPrice.toLocaleString()}</span>
                        </div>
                        <div className="detail">
                          <span className="label">Lvg</span>
                          <span className="value">{position.leverage}x</span>
                        </div>
                      </div>
                      <div className={`pnl ${position.pnl >= 0 ? 'positive' : 'negative'}`}>
                        {position.pnl >= 0 ? '+' : ''}{position.pnl.toFixed(2)} ({position.pnlPercent.toFixed(2)}%)
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}

            {activeTab === 'orders' && (
              <div className="orders-list">
                {orders.length === 0 ? (
                  <div className="empty-state">No open orders</div>
                ) : (
                  orders.map(order => (
                    <div key={order.id} className="order-item">
                      <div className="order-header">
                        <span className={`side ${order.side}`}>{order.side.toUpperCase()}</span>
                        <span className="type">{order.type}</span>
                        <span className="symbol">{order.symbol}</span>
                      </div>
                      <div className="order-details">
                        <span>Size: {order.size}</span>
                        <span>Price: {order.price.toLocaleString()}</span>
                        <span className="status">{order.status}</span>
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>
        </div>

        {/* Right Panel - Order Book & Recent Trades */}
        <div className="orderbook-panel">
          <div className="orderbook">
            <h3>Order Book</h3>
            <div className="orderbook-header">
              <span>Price</span>
              <span>Size</span>
              <span>Total</span>
            </div>
            <div className="orderbook-asks">
              {[...Array(10)].map((_, i) => {
                const price = selectedPair.price * (1 + (i + 1) * 0.001);
                const size = Math.random() * 10 + 0.1;
                return (
                  <div key={`ask-${i}`} className="orderbook-row ask">
                    <span className="price">{price.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: selectedPair.price < 1 ? 6 : 2 })}</span>
                    <span className="size">{size.toFixed(4)}</span>
                    <span className="total">{(price * size).toFixed(2)}</span>
                  </div>
                );
              })}
            </div>
            <div className="orderbook-spread">
              <span className="spread-price">{selectedPair.price.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: selectedPair.price < 1 ? 6 : 2 })}</span>
            </div>
            <div className="orderbook-bids">
              {[...Array(10)].map((_, i) => {
                const price = selectedPair.price * (1 - (i + 1) * 0.001);
                const size = Math.random() * 10 + 0.1;
                return (
                  <div key={`bid-${i}`} className="orderbook-row bid">
                    <span className="price">{price.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: selectedPair.price < 1 ? 6 : 2 })}</span>
                    <span className="size">{size.toFixed(4)}</span>
                    <span className="total">{(price * size).toFixed(2)}</span>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default FuturesTradingPage;
