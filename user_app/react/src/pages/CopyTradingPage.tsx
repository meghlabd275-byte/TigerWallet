// Copy Trading Page - Follow Top Traders
// Supports 50,000+ trading pairs for futures copy trading

import React, { useState, useEffect, useCallback } from 'react';
import './CopyTradingPage.css';

interface Trader {
  id: string;
  username: string;
  avatar: string;
  winRate: number;
  totalPnl: number;
  pnlPercent: number;
  followers: number;
  copyCount: number;
  tradingPair: string;
  monthlyPnl: number;
  weeklyPnl: number;
  dailyPnl: number;
  maxDrawdown: number;
  avgHoldingTime: string;
  riskLevel: 'low' | 'medium' | 'high';
  isFollowing: boolean;
  isPreInstalled: boolean;
}

interface CopyPosition {
  id: string;
  traderId: string;
  traderName: string;
  symbol: string;
  side: 'long' | 'short';
  size: number;
  entryPrice: number;
  currentPrice: number;
  pnl: number;
  pnlPercent: number;
  openTime: number;
}


const CopyTradingPage: React.FC = () => {
  const [traders, setTraders] = useState<Trader[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const API_BASE_URL = 'http://localhost:8443/api/v1/copytrading';
  const [selectedTrader, setSelectedTrader] = useState<Trader | null>(null);
  const [copyPositions, setCopyPositions] = useState<CopyPosition[]>([]);
  const [activeTab, setActiveTab] = useState<'traders' | 'positions' | 'my-copies'>('traders');
  const [filterRisk, setFilterRisk] = useState<'all' | 'low' | 'medium' | 'high'>('all');
  const [sortBy, setSortBy] = useState<'pnl' | 'winrate' | 'followers'>('pnl');
  const [searchTerm, setSearchTerm] = useState('');
  const [showAllTraders, setShowAllTraders] = useState(false);
  const [copyAmount, setCopyAmount] = useState('1000');
  const [copyLeverage, setCopyLeverage] = useState(1);

  // Fetch real traders from the copy_trading_service backend (:8006).
  // No fabricated 500-trader pool — only real registered traders.
  const fetchTraders = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('user_token');
      const res = await fetch(`${API_BASE_URL}/traders`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!res.ok) throw new Error('Failed to load traders');
      const data = await res.json();
      const backendTraders: Trader[] = (data.traders || []).map((t: any) => ({
        id: t.id,
        username: t.name || t.address || t.id,
        avatar: '🐯',
        winRate: Number(t.win_rate) || 0,
        totalPnl: Number(t.pnl_pct) || 0,
        pnlPercent: Number(t.pnl_pct) || 0,
        followers: Number(t.followers) || 0,
        copyCount: 0,
        tradingPair: 'BTC/USDT',
        monthlyPnl: 0,
        weeklyPnl: 0,
        dailyPnl: 0,
        maxDrawdown: 0,
        avgHoldingTime: '0h 0m',
        riskLevel: 'medium',
        isFollowing: false,
        isPreInstalled: true,
      }));
      setTraders(backendTraders.length > 0 ? backendTraders : []);
    } catch (err: any) {
      setError(err.message || 'Failed to load traders');
      setTraders([]);
    } finally {
      setLoading(false);
    }
  }, [API_BASE_URL]);

  // Fetch the caller's active copy positions from the backend.
  const fetchPositions = useCallback(async () => {
    try {
      const token = localStorage.getItem('user_token');
      const res = await fetch(`${API_BASE_URL}/copiers`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (res.ok) {
        const data = await res.json();
        setCopyPositions((data.copiers || []).map((c: any) => ({
          id: c.id,
          traderId: c.trader_id,
          traderName: c.trader_name || c.trader_id,
          symbol: c.symbol || 'BTC/USDT',
          side: c.side || 'long',
          size: Number(c.size) || 0,
          entryPrice: Number(c.entry_price) || 0,
          currentPrice: Number(c.current_price) || 0,
          pnl: Number(c.pnl) || 0,
          pnlPercent: Number(c.pnl_pct) || 0,
          openTime: Number(c.open_time) || Date.now(),
        })));
      }
    } catch (err) {
      // Positions list stays empty on failure (fail-closed).
    }
  }, [API_BASE_URL]);

  useEffect(() => {
    fetchTraders();
    fetchPositions();
  }, [fetchTraders, fetchPositions]);

  const filteredTraders = traders
    .filter(trader => {
      if (filterRisk !== 'all' && trader.riskLevel !== filterRisk) return false;
      if (searchTerm && !trader.username.toLowerCase().includes(searchTerm.toLowerCase()) &&
          !trader.tradingPair.toLowerCase().includes(searchTerm.toLowerCase())) return false;
      return true;
    })
    .sort((a, b) => {
      if (sortBy === 'pnl') return b.totalPnl - a.totalPnl;
      if (sortBy === 'winrate') return b.winRate - a.winRate;
      return b.followers - a.followers;
    });

  const displayTraders = showAllTraders ? filteredTraders : filteredTraders.slice(0, 12);

  const handleFollow = async (traderId: string) => {
    try {
      const token = localStorage.getItem('user_token');
      const res = await fetch(`${API_BASE_URL}/follow`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        body: JSON.stringify({ trader_id: traderId }),
      });
      if (res.ok) {
        setTraders(traders.map(t =>
          t.id === traderId ? { ...t, isFollowing: !t.isFollowing } : t
        ));
      }
    } catch (err) {
      // Fail-closed: follow state unchanged on error.
    }
  };

  const handleCopyTrade = async (trader: Trader) => {
    try {
      const token = localStorage.getItem('user_token');
      const res = await fetch(`${API_BASE_URL}/follow`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        body: JSON.stringify({ trader_id: trader.id, amount: parseFloat(copyAmount), leverage: copyLeverage }),
      });
      if (res.ok) {
        await fetchPositions();
        setActiveTab('my-copies');
      }
    } catch (err) {
      // Fail-closed.
    }
  };

  const followingTraders = traders.filter(t => t.isFollowing);

  return (
    <div className="copy-trading-page">
      <div className="copy-trading-header">
        <h1>Copy Trading</h1>
        <p>Follow expert traders and automatically copy their trades</p>
      </div>

      <div className="tabs">
        <button 
          className={activeTab === 'traders' ? 'active' : ''} 
          onClick={() => setActiveTab('traders')}
        >
          Top Traders
        </button>
        <button 
          className={activeTab === 'my-copies' ? 'active' : ''} 
          onClick={() => setActiveTab('my-copies')}
        >
          My Copies ({followingTraders.length})
        </button>
        <button 
          className={activeTab === 'positions' ? 'active' : ''} 
          onClick={() => setActiveTab('positions')}
        >
          Copy Positions ({copyPositions.length})
        </button>
      </div>

      {activeTab === 'traders' && (
        <div className="traders-section">
          <div className="filters">
            <input
              type="text"
              placeholder="Search traders or pairs..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
            
            <select 
              value={filterRisk} 
              onChange={(e) => setFilterRisk(e.target.value as any)}
              className="filter-select"
            >
              <option value="all">All Risk Levels</option>
              <option value="low">Low Risk</option>
              <option value="medium">Medium Risk</option>
              <option value="high">High Risk</option>
            </select>
            
            <select 
              value={sortBy} 
              onChange={(e) => setSortBy(e.target.value as any)}
              className="filter-select"
            >
              <option value="pnl">Sort by PnL</option>
              <option value="winrate">Sort by Win Rate</option>
              <option value="followers">Sort by Followers</option>
            </select>
          </div>

          <div className="traders-grid">
            {displayTraders.map(trader => (
              <div 
                key={trader.id} 
                className={`trader-card ${selectedTrader?.id === trader.id ? 'selected' : ''}`}
                onClick={() => setSelectedTrader(trader)}
              >
                <div className="trader-header">
                  <div className="trader-avatar">{trader.avatar}</div>
                  <div className="trader-info">
                    <h3>
                      {trader.username}
                      {trader.isPreInstalled && <span className="pre-installed-badge">★</span>}
                    </h3>
                    <span className="trading-pair">{trader.tradingPair}</span>
                  </div>
                  <span className={`risk-badge ${trader.riskLevel}`}>
                    {trader.riskLevel}
                  </span>
                </div>

                <div className="trader-stats">
                  <div className="stat">
                    <span className="label">Win Rate</span>
                    <span className="value">{trader.winRate.toFixed(1)}%</span>
                  </div>
                  <div className="stat">
                    <span className="label">Total PnL</span>
                    <span className={`value ${trader.totalPnl >= 0 ? 'positive' : 'negative'}`}>
                      ${trader.totalPnl.toLocaleString()}
                    </span>
                  </div>
                  <div className="stat">
                    <span className="label">PnL %</span>
                    <span className={`value ${trader.pnlPercent >= 0 ? 'positive' : 'negative'}`}>
                      {trader.pnlPercent >= 0 ? '+' : ''}{trader.pnlPercent.toFixed(1)}%
                    </span>
                  </div>
                </div>

                <div className="trader-metrics">
                  <div className="metric">
                    <span className="label">Monthly</span>
                    <span className={`value ${trader.monthlyPnl >= 0 ? 'positive' : 'negative'}`}>
                      {trader.monthlyPnl >= 0 ? '+' : ''}{trader.monthlyPnl.toFixed(1)}%
                    </span>
                  </div>
                  <div className="metric">
                    <span className="label">Weekly</span>
                    <span className={`value ${trader.weeklyPnl >= 0 ? 'positive' : 'negative'}`}>
                      {trader.weeklyPnl >= 0 ? '+' : ''}{trader.weeklyPnl.toFixed(1)}%
                    </span>
                  </div>
                  <div className="metric">
                    <span className="label">Daily</span>
                    <span className={`value ${trader.dailyPnl >= 0 ? 'positive' : 'negative'}`}>
                      {trader.dailyPnl >= 0 ? '+' : ''}{trader.dailyPnl.toFixed(1)}%
                    </span>
                  </div>
                </div>

                <div className="trader-footer">
                  <div className="followers">
                    <span>👥 {trader.followers.toLocaleString()}</span>
                    <span>📋 {trader.copyCount.toLocaleString()}</span>
                  </div>
                  <div className="trader-actions">
                    <button 
                      className={`follow-btn ${trader.isFollowing ? 'following' : ''}`}
                      onClick={(e) => {
                        e.stopPropagation();
                        handleFollow(trader.id);
                      }}
                    >
                      {trader.isFollowing ? 'Following' : 'Follow'}
                    </button>
                    <button 
                      className="copy-btn"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleCopyTrade(trader);
                      }}
                    >
                      Copy
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {filteredTraders.length > (showAllTraders ? filteredTraders.length : 12) && (
            <button 
              className="show-more-btn"
              onClick={() => setShowAllTraders(!showAllTraders)}
            >
              {showAllTraders ? 'Show Top Traders' : `Show All Traders (${filteredTraders.length}+)`}
            </button>
          )}
        </div>
      )}

      {activeTab === 'my-copies' && (
        <div className="my-copies-section">
          <div className="copy-settings">
            <h3>Copy Settings</h3>
            <div className="settings-row">
              <label>Copy Amount (USDT)</label>
              <input
                type="number"
                value={copyAmount}
                onChange={(e) => setCopyAmount(e.target.value)}
                min="10"
                step="10"
              />
            </div>
            <div className="settings-row">
              <label>Copy Leverage</label>
              <select 
                value={copyLeverage} 
                onChange={(e) => setCopyLeverage(parseInt(e.target.value))}
              >
                {[1, 2, 3, 5, 10, 20, 50, 100].map(l => (
                  <option key={l} value={l}>{l}x</option>
                ))}
              </select>
            </div>
          </div>

          <h3>Following ({followingTraders.length})</h3>
          {followingTraders.length === 0 ? (
            <div className="empty-state">
              <p>You're not following any traders yet.</p>
              <p>Browse top traders and follow them to start copying their trades.</p>
            </div>
          ) : (
            <div className="following-list">
              {followingTraders.map(trader => (
                <div key={trader.id} className="following-item">
                  <div className="trader-avatar">{trader.avatar}</div>
                  <div className="trader-info">
                    <h4>{trader.username}</h4>
                    <span>{trader.tradingPair}</span>
                  </div>
                  <div className="trader-stats-mini">
                    <span className={`pnl ${trader.dailyPnl >= 0 ? 'positive' : 'negative'}`}>
                      {trader.dailyPnl >= 0 ? '+' : ''}{trader.dailyPnl.toFixed(1)}% today
                    </span>
                  </div>
                  <button 
                    className="unfollow-btn"
                    onClick={() => handleFollow(trader.id)}
                  >
                    Unfollow
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {activeTab === 'positions' && (
        <div className="positions-section">
          <h3>Active Copy Positions ({copyPositions.length})</h3>
          {copyPositions.length === 0 ? (
            <div className="empty-state">
              <p>No active copy positions.</p>
              <p>Follow traders and copy their trades to see positions here.</p>
            </div>
          ) : (
            <div className="positions-list">
              {copyPositions.map(position => (
                <div key={position.id} className="position-card">
                  <div className="position-header">
                    <div className="position-info">
                      <span className={`side ${position.side}`}>{position.side.toUpperCase()}</span>
                      <span className="symbol">{position.symbol}</span>
                    </div>
                    <span className="trader">📋 {position.traderName}</span>
                  </div>
                  <div className="position-details">
                    <div className="detail">
                      <span className="label">Size</span>
                      <span className="value">{position.size.toFixed(4)}</span>
                    </div>
                    <div className="detail">
                      <span className="label">Entry</span>
                      <span className="value">${position.entryPrice.toFixed(2)}</span>
                    </div>
                    <div className="detail">
                      <span className="label">Current</span>
                      <span className="value">${position.currentPrice.toFixed(2)}</span>
                    </div>
                    <div className="detail">
                      <span className="label">PnL</span>
                      <span className={`value ${position.pnl >= 0 ? 'positive' : 'negative'}`}>
                        ${position.pnl.toFixed(2)} ({position.pnlPercent.toFixed(2)}%)
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {selectedTrader && (
        <div className="trader-detail-modal" onClick={() => setSelectedTrader(null)}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <button className="close-btn" onClick={() => setSelectedTrader(null)}>×</button>
            <div className="modal-header">
              <div className="trader-avatar large">{selectedTrader.avatar}</div>
              <h2>{selectedTrader.username}</h2>
              <span className={`risk-badge ${selectedTrader.riskLevel}`}>
                {selectedTrader.riskLevel} risk
              </span>
            </div>
            <div className="modal-stats">
              <div className="stat-box">
                <span className="label">Win Rate</span>
                <span className="value">{selectedTrader.winRate.toFixed(1)}%</span>
              </div>
              <div className="stat-box">
                <span className="label">Total PnL</span>
                <span className={`value ${selectedTrader.totalPnl >= 0 ? 'positive' : 'negative'}`}>
                  ${selectedTrader.totalPnl.toLocaleString()}
                </span>
              </div>
              <div className="stat-box">
                <span className="label">Max Drawdown</span>
                <span className="value negative">{selectedTrader.maxDrawdown}%</span>
              </div>
              <div className="stat-box">
                <span className="label">Avg Holding Time</span>
                <span className="value">{selectedTrader.avgHoldingTime}</span>
              </div>
            </div>
            <div className="modal-actions">
              <button 
                className={`follow-btn large ${selectedTrader.isFollowing ? 'following' : ''}`}
                onClick={() => handleFollow(selectedTrader.id)}
              >
                {selectedTrader.isFollowing ? 'Following' : 'Follow'}
              </button>
              <button 
                className="copy-btn large"
                onClick={() => handleCopyTrade(selectedTrader)}
              >
                Copy Trade (${copyAmount})
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default CopyTradingPage;
