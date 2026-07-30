/**
 * TigerWallet Prediction Markets - Main Component
 * Complete trading interface for prediction markets
 */

import React, { useState, useEffect, useCallback } from 'react';
import { usePrediction, formatPrice, formatVolume, formatTimestamp, getOutcomeColor } from '../services/predictionApi';

// ============================================================================
// Theme Context (for light/dark mode across all pages)
// ============================================================================

interface ThemeContextType {
  theme: 'light' | 'dark';
  setTheme: (theme: 'light' | 'dark') => void;
  toggleTheme: () => void;
}

const ThemeContext = React.createContext<ThemeContextType | null>(null);

export function useTheme() {
  const context = React.useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider');
  }
  return context;
}

interface ThemeProviderProps {
  children: React.ReactNode;
  defaultTheme?: 'light' | 'dark';
}

export function ThemeProvider({ children, defaultTheme = 'dark' }: ThemeProviderProps) {
  const [theme, setTheme] = useState<'light' | 'dark'>(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('tiger_theme');
      if (saved === 'light' || saved === 'dark') return saved;
    }
    return defaultTheme;
  });

  useEffect(() => {
    const root = document.documentElement;
    root.setAttribute('data-theme', theme);
    localStorage.setItem('tiger_theme', theme);
  }, [theme]);

  const toggleTheme = useCallback(() => {
    setTheme(prev => prev === 'light' ? 'dark' : 'light');
  }, []);

  return (
    <ThemeContext.Provider value={{ theme, setTheme, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

// ============================================================================
// Main Prediction Markets Component
// ============================================================================

interface PredictionMarketsProps {
  className?: string;
}

export function PredictionMarkets({ className = '' }: PredictionMarketsProps) {
  const {
    markets,
    featuredMarkets,
    selectedMarket,
    positions,
    trades,
    balance,
    stats,
    loading,
    error,
    theme,
    setTheme,
    selectMarket,
    refreshMarkets,
    placeOrder,
    cancelOrder,
    addFunds,
  } = usePrediction();

  const [activeTab, setActiveTab] = useState<'markets' | 'positions' | 'trades'>('markets');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [showAddFunds, setShowAddFunds] = useState(false);
  const [fundsAmount, setFundsAmount] = useState('');

  const categories = React.useMemo(() => {
    const cats = new Set<string>();
    markets.forEach(m => cats.add(m.category));
    return ['all', ...Array.from(cats)];
  }, [markets]);

  const filteredMarkets = React.useMemo(() => {
    return markets.filter(m => {
      if (selectedCategory !== 'all' && m.category !== selectedCategory) return false;
      if (searchQuery && !m.question.toLowerCase().includes(searchQuery.toLowerCase())) return false;
      return true;
    });
  }, [markets, selectedCategory, searchQuery]);

  const handleAddFunds = async () => {
    const amount = parseFloat(fundsAmount);
    if (amount > 0) {
      await addFunds(Math.floor(amount * 1000000));
      setShowAddFunds(false);
      setFundsAmount('');
    }
  };

  const formatCurrency = (value: number) => {
    if (value >= 1000000) return `$${(value / 1000000).toFixed(2)}M`;
    if (value >= 1000) return `$${(value / 1000).toFixed(2)}K`;
    return `$${(value / 1000000).toFixed(6)}`;
  };

  return (
    <div className={`prediction-markets ${theme} ${className}`} data-theme={theme}>
      {/* Header */}
      <header className="pm-header">
        <div className="pm-header-left">
          <h1 className="pm-title">
            <span className="pm-icon">🎯</span>
            Prediction Markets
          </h1>
          <div className="pm-balance">
            <span className="balance-label">Balance:</span>
            <span className="balance-value">{formatCurrency(balance)}</span>
            <button 
              className="add-funds-btn"
              onClick={() => setShowAddFunds(true)}
            >
              + Add Funds
            </button>
          </div>
        </div>
        <div className="pm-header-right">
          <div className="pm-stats">
            {stats && (
              <>
                <div className="stat-item">
                  <span className="stat-label">Markets</span>
                  <span className="stat-value">{stats.active_markets}</span>
                </div>
                <div className="stat-item">
                  <span className="stat-label">24h Volume</span>
                  <span className="stat-value">{formatVolume(stats.volume_24h)}</span>
                </div>
                <div className="stat-item">
                  <span className="stat-label">Total Trades</span>
                  <span className="stat-value">{stats.total_trades.toLocaleString()}</span>
                </div>
              </>
            )}
          </div>
          <button 
            className="theme-toggle"
            onClick={() => setTheme(theme === 'light' ? 'dark' : 'light')}
          >
            {theme === 'light' ? '🌙' : '☀️'}
          </button>
        </div>
      </header>

      {/* Add Funds Modal */}
      {showAddFunds && (
        <div className="modal-overlay" onClick={() => setShowAddFunds(false)}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <h2>Add Funds</h2>
            <input
              type="number"
              value={fundsAmount}
              onChange={e => setFundsAmount(e.target.value)}
              placeholder="Enter amount"
              className="funds-input"
            />
            <div className="modal-actions">
              <button onClick={() => setShowAddFunds(false)} className="cancel-btn">Cancel</button>
              <button onClick={handleAddFunds} className="confirm-btn">Add</button>
            </div>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="pm-tabs">
        <button
          className={`tab ${activeTab === 'markets' ? 'active' : ''}`}
          onClick={() => setActiveTab('markets')}
        >
          Markets
        </button>
        <button
          className={`tab ${activeTab === 'positions' ? 'active' : ''}`}
          onClick={() => setActiveTab('positions')}
        >
          Positions
        </button>
        <button
          className={`tab ${activeTab === 'trades' ? 'active' : ''}`}
          onClick={() => setActiveTab('trades')}
        >
          Trade History
        </button>
      </div>

      {/* Markets Tab */}
      {activeTab === 'markets' && (
        <div className="pm-markets">
          {/* Filters */}
          <div className="pm-filters">
            <div className="search-box">
              <input
                type="text"
                placeholder="Search markets..."
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                className="search-input"
              />
            </div>
            <div className="category-tabs">
              {categories.map(cat => (
                <button
                  key={cat}
                  className={`category-tab ${selectedCategory === cat ? 'active' : ''}`}
                  onClick={() => setSelectedCategory(cat)}
                >
                  {cat === 'all' ? 'All' : cat}
                </button>
              ))}
            </div>
          </div>

          {/* Featured Markets */}
          {featuredMarkets.length > 0 && !selectedCategory && !searchQuery && (
            <div className="pm-featured">
              <h2 className="section-title">Featured Markets</h2>
              <div className="markets-grid">
                {featuredMarkets.map(market => (
                  <MarketCard 
                    key={market.market_id} 
                    market={market}
                    onSelect={() => selectMarket(market)}
                    selected={selectedMarket?.market_id === market.market_id}
                  />
                ))}
              </div>
            </div>
          )}

          {/* All Markets */}
          <div className="pm-all-markets">
            <h2 className="section-title">
              {selectedCategory === 'all' ? 'All Markets' : selectedCategory}
            </h2>
            {loading ? (
              <div className="loading">Loading markets...</div>
            ) : error ? (
              <div className="error">
                {error}
                <button onClick={refreshMarkets}>Retry</button>
              </div>
            ) : filteredMarkets.length === 0 ? (
              <div className="empty">No markets found</div>
            ) : (
              <div className="markets-grid">
                {filteredMarkets.map(market => (
                  <MarketCard 
                    key={market.market_id} 
                    market={market}
                    onSelect={() => selectMarket(market)}
                    selected={selectedMarket?.market_id === market.market_id}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Positions Tab */}
      {activeTab === 'positions' && (
        <div className="pm-positions">
          <h2 className="section-title">Your Positions</h2>
          {positions.length === 0 ? (
            <div className="empty">No positions yet</div>
          ) : (
            <div className="positions-list">
              {positions.map((pos, idx) => (
                <div key={idx} className="position-card">
                  <div className="position-header">
                    <span className="position-market">Market #{pos.market_id}</span>
                    <span className={`position-pnl ${pos.profit_loss >= 0 ? 'positive' : 'negative'}`}>
                      {pos.profit_loss >= 0 ? '+' : ''}{formatCurrency(pos.profit_loss)}
                    </span>
                  </div>
                  <div className="position-details">
                    <div className="position-stat">
                      <span className="stat-label">Invested</span>
                      <span className="stat-value">{formatCurrency(pos.invested)}</span>
                    </div>
                    <div className="position-stat">
                      <span className="stat-label">Current Value</span>
                      <span className="stat-value">{formatCurrency(pos.current_value)}</span>
                    </div>
                    <div className="position-stat">
                      <span className="stat-label">Quantity</span>
                      <span className="stat-value">{pos.quantity}</span>
                    </div>
                    <div className="position-stat">
                      <span className="stat-label">Avg Price</span>
                      <span className="stat-value">{formatPrice(pos.avg_price)}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Trades Tab */}
      {activeTab === 'trades' && (
        <div className="pm-trades">
          <h2 className="section-title">Trade History</h2>
          {trades.length === 0 ? (
            <div className="empty">No trades yet</div>
          ) : (
            <div className="trades-list">
              {trades.map((trade, idx) => (
                <div key={idx} className="trade-card">
                  <div className="trade-header">
                    <span className={`trade-side ${trade.side}`}>
                      {trade.side.toUpperCase()}
                    </span>
                    <span className="trade-market">Market #{trade.market_id}</span>
                    <span className="trade-time">{formatTimestamp(trade.timestamp)}</span>
                  </div>
                  <div className="trade-details">
                    <div className="trade-stat">
                      <span className="stat-label">Amount</span>
                      <span className="stat-value">{trade.amount}</span>
                    </div>
                    <div className="trade-stat">
                      <span className="stat-label">Price</span>
                      <span className="stat-value">{formatPrice(trade.price)}</span>
                    </div>
                    <div className="trade-stat">
                      <span className="stat-label">Value</span>
                      <span className="stat-value">{formatCurrency(trade.amount * trade.price)}</span>
                    </div>
                    <div className="trade-stat">
                      <span className="stat-label">Fees</span>
                      <span className="stat-value">{formatCurrency(trade.fees)}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Market Detail Modal */}
      {selectedMarket && (
        <MarketDetailModal
          market={selectedMarket}
          onClose={() => selectMarket(null)}
          placeOrder={placeOrder}
          balance={balance}
        />
      )}
    </div>
  );
}

// ============================================================================
// Market Card Component
// ============================================================================

interface MarketCardProps {
  market: any;
  onSelect: () => void;
  selected: boolean;
}

function MarketCard({ market, onSelect, selected }: MarketCardProps) {
  const formatCurrency = (value: number) => {
    if (value >= 1000000) return `$${(value / 1000000).toFixed(2)}M`;
    if (value >= 1000) return `$${(value / 1000).toFixed(2)}K`;
    return `$${(value / 1000000).toFixed(2)}`;
  };

  const formatPrice = (price: number) => (price / 1000000).toFixed(2);

  return (
    <div 
      className={`market-card ${selected ? 'selected' : ''} ${market.status !== 'active' ? 'inactive' : ''}`}
      onClick={onSelect}
    >
      <div className="market-header">
        <span className="market-category">{market.category}</span>
        {market.featured && <span className="featured-badge">⭐ Featured</span>}
      </div>
      <h3 className="market-question">{market.question}</h3>
      <div className="market-outcomes">
        {market.outcomes.map((outcome: any) => (
          <div key={outcome.outcome_id} className="outcome">
            <span className="outcome-name">{outcome.name}</span>
            <span className={`outcome-price ${parseFloat(formatPrice(outcome.price)) >= 0.5 ? 'green' : 'red'}`}>
              {formatPrice(outcome.price)}
            </span>
          </div>
        ))}
      </div>
      <div className="market-footer">
        <span className="market-volume">
          Vol: {formatCurrency(market.total_volume)}
        </span>
        <span className={`market-status ${market.status}`}>
          {market.status}
        </span>
      </div>
    </div>
  );
}

// ============================================================================
// Market Detail Modal
// ============================================================================

interface MarketDetailModalProps {
  market: any;
  onClose: () => void;
  placeOrder: (order: any) => Promise<any>;
  balance: number;
}

function MarketDetailModal({ market, onClose, placeOrder, balance }: MarketDetailModalProps) {
  const [selectedOutcome, setSelectedOutcome] = useState<number>(0);
  const [orderSide, setOrderSide] = useState<'buy' | 'sell'>('buy');
  const [orderAmount, setOrderAmount] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<any>(null);

  const formatCurrency = (value: number) => {
    if (value >= 1000000) return `$${(value / 1000000).toFixed(2)}M`;
    if (value >= 1000) return `$${(value / 1000).toFixed(2)}K`;
    return `$${(value / 1000000).toFixed(6)}`;
  };

  const formatPrice = (price: number) => (price / 1000000).toFixed(2);

  const amount = parseFloat(orderAmount) || 0;
  const totalCost = amount * (market.outcomes[selectedOutcome]?.price || 0) / 1000000;
  const potentialWinnings = orderSide === 'buy' 
    ? (amount / (market.outcomes[selectedOutcome]?.price || 1) * 1000000) - amount
    : amount;

  const handlePlaceOrder = async () => {
    if (amount <= 0) return;
    setLoading(true);
    try {
      const order = await placeOrder({
        market_id: market.market_id,
        outcome_id: selectedOutcome,
        order_type: 'limit',
        side: orderSide,
        price: market.outcomes[selectedOutcome].price,
        amount: Math.floor(amount),
      });
      setResult({ success: true, order });
    } catch (err: any) {
      setResult({ success: false, error: err.message });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="market-detail-modal" onClick={e => e.stopPropagation()}>
        <button className="close-btn" onClick={onClose}>×</button>
        
        <h2 className="modal-title">{market.question}</h2>
        <p className="modal-description">{market.description}</p>
        
        <div className="outcomes-selector">
          <h3>Select Outcome</h3>
          <div className="outcomes-grid">
            {market.outcomes.map((outcome: any, idx: number) => (
              <button
                key={outcome.outcome_id}
                className={`outcome-btn ${selectedOutcome === idx ? 'selected' : ''}`}
                onClick={() => setSelectedOutcome(idx)}
              >
                <span className="outcome-name">{outcome.name}</span>
                <span className="outcome-price">{formatPrice(outcome.price)}</span>
                <span className="outcome-volume">Vol: {formatCurrency(outcome.volume)}</span>
              </button>
            ))}
          </div>
        </div>

        <div className="order-form">
          <h3>Place Order</h3>
          
          <div className="order-side-toggle">
            <button 
              className={`side-btn buy ${orderSide === 'buy' ? 'active' : ''}`}
              onClick={() => setOrderSide('buy')}
            >
              Buy
            </button>
            <button 
              className={`side-btn sell ${orderSide === 'sell' ? 'active' : ''}`}
              onClick={() => setOrderSide('sell')}
            >
              Sell
            </button>
          </div>

          <div className="order-input-group">
            <label>Amount</label>
            <input
              type="number"
              value={orderAmount}
              onChange={e => setOrderAmount(e.target.value)}
              placeholder="Enter amount"
              min="0"
              step="1"
            />
          </div>

          <div className="order-summary">
            <div className="summary-row">
              <span>Total Cost</span>
              <span>{formatCurrency(totalCost)}</span>
            </div>
            <div className="summary-row">
              <span>Potential Winnings</span>
              <span className="winnings">{formatCurrency(potentialWinnings)}</span>
            </div>
            <div className="summary-row">
              <span>Available Balance</span>
              <span>{formatCurrency(balance)}</span>
            </div>
          </div>

          {result && (
            <div className={`order-result ${result.success ? 'success' : 'error'}`}>
              {result.success 
                ? `Order placed: #${result.order.order_id}`
                : `Error: ${result.error}`
              }
            </div>
          )}

          <button 
            className={`place-order-btn ${orderSide}`}
            onClick={handlePlaceOrder}
            disabled={loading || amount <= 0 || totalCost > balance}
          >
            {loading ? 'Processing...' : `${orderSide === 'buy' ? 'Buy' : 'Sell'} ${market.outcomes[selectedOutcome]?.name}`}
          </button>
        </div>
      </div>
    </div>
  );
}

export default PredictionMarkets;
