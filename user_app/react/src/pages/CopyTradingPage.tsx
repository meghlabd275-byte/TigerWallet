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

const TOP_TRADERS: Trader[] = [
  {
    id: '1',
    username: 'CryptoWhale',
    avatar: '🐋',
    winRate: 78.5,
    totalPnl: 125000,
    pnlPercent: 156.2,
    followers: 15234,
    copyCount: 4521,
    tradingPair: 'BTC/USDT',
    monthlyPnl: 12.5,
    weeklyPnl: 3.2,
    dailyPnl: 0.8,
    maxDrawdown: -8.5,
    avgHoldingTime: '2h 30m',
    riskLevel: 'medium',
    isFollowing: false,
    isPreInstalled: true
  },
  {
    id: '2',
    username: 'DeFiMaster',
    avatar: '🎯',
    winRate: 82.3,
    totalPnl: 98500,
    pnlPercent: 142.8,
    followers: 12456,
    copyCount: 3890,
    tradingPair: 'ETH/USDT',
    monthlyPnl: 15.2,
    weeklyPnl: 4.1,
    dailyPnl: 1.2,
    maxDrawdown: -6.2,
    avgHoldingTime: '4h 15m',
    riskLevel: 'low',
    isFollowing: false,
    isPreInstalled: true
  },
  {
    id: '3',
    username: 'AltSeason',
    avatar: '🚀',
    winRate: 71.2,
    totalPnl: 87000,
    pnlPercent: 198.5,
    followers: 8923,
    copyCount: 2156,
    tradingPair: 'SOL/USDT',
    monthlyPnl: 22.5,
    weeklyPnl: 8.3,
    dailyPnl: 2.1,
    maxDrawdown: -12.8,
    avgHoldingTime: '1h 45m',
    riskLevel: 'high',
    isFollowing: false,
    isPreInstalled: true
  },
  {
    id: '4',
    username: 'GridTrader',
    avatar: '📊',
    winRate: 85.1,
    totalPnl: 67800,
    pnlPercent: 98.3,
    followers: 6543,
    copyCount: 1890,
    tradingPair: 'BNB/USDT',
    monthlyPnl: 8.2,
    weeklyPnl: 2.1,
    dailyPnl: 0.5,
    maxDrawdown: -4.2,
    avgHoldingTime: '6h 20m',
    riskLevel: 'low',
    isFollowing: false,
    isPreInstalled: true
  },
  {
    id: '5',
    username: 'MomentumKing',
    avatar: '👑',
    winRate: 75.8,
    totalPnl: 54200,
    pnlPercent: 125.6,
    followers: 9876,
    copyCount: 2567,
    tradingPair: 'DOGE/USDT',
    monthlyPnl: 18.5,
    weeklyPnl: 5.2,
    dailyPnl: 1.5,
    maxDrawdown: -15.2,
    avgHoldingTime: '0h 45m',
    riskLevel: 'high',
    isFollowing: false,
    isPreInstalled: true
  },
  {
    id: '6',
    username: 'SwingTrader',
    avatar: '🌊',
    winRate: 68.5,
    totalPnl: 42500,
    pnlPercent: 88.2,
    followers: 5432,
    copyCount: 1234,
    tradingPair: 'XRP/USDT',
    monthlyPnl: 10.2,
    weeklyPnl: 2.8,
    dailyPnl: 0.3,
    maxDrawdown: -9.5,
    avgHoldingTime: '12h 30m',
    riskLevel: 'medium',
    isFollowing: false,
    isPreInstalled: true
  },
  {
    id: '7',
    username: 'BotMaster',
    avatar: '🤖',
    winRate: 88.2,
    totalPnl: 38900,
    pnlPercent: 72.5,
    followers: 4321,
    copyCount: 987,
    tradingPair: 'AVAX/USDT',
    monthlyPnl: 6.8,
    weeklyPnl: 1.5,
    dailyPnl: 0.2,
    maxDrawdown: -3.2,
    avgHoldingTime: '8h 00m',
    riskLevel: 'low',
    isFollowing: false,
    isPreInstalled: true
  },
  {
    id: '8',
    username: 'NanoGainer',
    avatar: '💎',
    winRate: 73.2,
    totalPnl: 31500,
    pnlPercent: 145.8,
    followers: 7654,
    copyCount: 1876,
    tradingPair: 'PEPE/USDT',
    monthlyPnl: 25.2,
    weeklyPnl: 9.5,
    dailyPnl: 3.2,
    maxDrawdown: -18.5,
    avgHoldingTime: '0h 30m',
    riskLevel: 'high',
    isFollowing: false,
    isPreInstalled: true
  },
  {
    id: '9',
    username: 'StableTrader',
    avatar: '🛡️',
    winRate: 91.2,
    totalPnl: 28900,
    pnlPercent: 52.3,
    followers: 3210,
    copyCount: 654,
    tradingPair: 'LINK/USDT',
    monthlyPnl: 4.2,
    weeklyPnl: 1.0,
    dailyPnl: 0.1,
    maxDrawdown: -2.1,
    avgHoldingTime: '24h 00m',
    riskLevel: 'low',
    isFollowing: false,
    isPreInstalled: true
  },
  {
    id: '10',
    username: 'FlashBoys',
    avatar: '⚡',
    winRate: 76.8,
    totalPnl: 24500,
    pnlPercent: 168.5,
    followers: 5678,
    copyCount: 1432,
    tradingPair: 'MATIC/USDT',
    monthlyPnl: 14.5,
    weeklyPnl: 4.8,
    dailyPnl: 1.8,
    maxDrawdown: -11.2,
    avgHoldingTime: '1h 15m',
    riskLevel: 'high',
    isFollowing: false,
    isPreInstalled: true
  },
  {
    id: '11',
    username: 'TrendFollower',
    avatar: '📈',
    winRate: 69.5,
    totalPnl: 21200,
    pnlPercent: 95.2,
    followers: 4321,
    copyCount: 1098,
    tradingPair: 'DOT/USDT',
    monthlyPnl: 8.8,
    weeklyPnl: 2.5,
    dailyPnl: 0.6,
    maxDrawdown: -7.8,
    avgHoldingTime: '5h 45m',
    riskLevel: 'medium',
    isFollowing: false,
    isPreInstalled: true
  },
  {
    id: '12',
    username: 'OptionsGuru',
    avatar: '🎰',
    winRate: 65.2,
    totalPnl: 18900,
    pnlPercent: 82.5,
    followers: 3456,
    copyCount: 876,
    tradingPair: 'NEAR/USDT',
    monthlyPnl: 11.2,
    weeklyPnl: 3.2,
    dailyPnl: 0.9,
    maxDrawdown: -10.5,
    avgHoldingTime: '3h 30m',
    riskLevel: 'medium',
    isFollowing: false,
    isPreInstalled: true
  }
];

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

  // Generate additional traders to simulate large pool
  useEffect(() => {
    const additionalTraders: Trader[] = [];
    const avatars = ['🐵', '🦊', '🦁', '🐯', '🐲', '🐍', '🐴', '🦄', '🐝', '🦋', '🌸', '🌺', '🌻', '🌹', '🍀'];
    const pairs = ['BTC/USDT', 'ETH/USDT', 'BNB/USDT', 'SOL/USDT', 'XRP/USDT', 'DOGE/USDT', 'ADA/USDT', 'AVAX/USDT', 'DOT/USDT', 'LINK/USDT', 'MATIC/USDT', 'LTC/USDT', 'UNI/USDT', 'ATOM/USDT', 'XLM/USDT', 'NEAR/USDT', 'APT/USDT', 'ARB/USDT', 'OP/USDT', 'INJ/USDT', 'PEPE/USDT', 'SHIB/USDT', 'TRX/USDT', 'FIL/USDT', 'ALGO/USDT', 'VET/USDT', 'ICP/USDT', 'HBAR/USDT', 'QNT/USDT', 'MKR/USDT', 'AAVE/USDT', 'GRT/USDT', 'SNX/USDT', 'CRV/USDT', 'LDO/USDT', 'RUNE/USDT', 'STX/USDT', 'KAVA/USDT', 'FLOW/USDT', 'AXS/USDT', 'SAND/USDT', 'MANA/USDT', 'ENJ/USDT', 'CHZ/USDT', 'BAT/USDT', 'ZEC/USDT', 'DASH/USDT', 'XMR/USDT', 'NEO/USDT', 'EOS/USDT', 'XTZ/USDT', 'ONE/USDT', 'ZIL/USDT', 'CELO/USDT', 'CAKE/USDT', 'GMT/USDT', 'GALA/USDT', 'ROSE/USDT', 'KLAY/USDT', 'MINA/USDT', 'COMP/USDT', 'BAL/USDT', 'YFI/USDT', 'SUSHI/USDT', '1INCH/USDT', 'CEL/USDT', 'OKB/USDT', 'KCS/USDT', 'HT/USDT', 'FTT/USDT', 'TUSD/USDT', 'BUSD/USDT', 'USDP/USDT', 'USDC/USDT'];
    const riskLevels: ('low' | 'medium' | 'high')[] = ['low', 'medium', 'high'];
    
    for (let i = 0; i < 500; i++) {
      const risk = riskLevels[Math.floor(Math.random() * 3)];
      additionalTraders.push({
        id: `trader-${i + 100}`,
        username: `Trader${i + 100}`,
        avatar: avatars[i % avatars.length],
        winRate: Math.random() * 30 + 60,
        totalPnl: Math.random() * 100000 + 1000,
        pnlPercent: Math.random() * 200 + 20,
        followers: Math.floor(Math.random() * 10000) + 100,
        copyCount: Math.floor(Math.random() * 5000) + 50,
        tradingPair: pairs[i % pairs.length],
        monthlyPnl: Math.random() * 30 - 5,
        weeklyPnl: Math.random() * 10 - 2,
        dailyPnl: Math.random() * 3 - 1,
        maxDrawdown: -Math.random() * 20 - 2,
        avgHoldingTime: `${Math.floor(Math.random() * 24)}h ${Math.floor(Math.random() * 60)}m`,
        riskLevel: risk,
        isFollowing: false,
        isPreInstalled: false
      });
    }
    setTraders([...TOP_TRADERS, ...additionalTraders]);
  }, []);

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

  const displayTraders = showAllTraders ? filteredTraders : filteredTraders.filter(t => t.isPreInstalled);

  const handleFollow = (traderId: string) => {
    setTraders(traders.map(t => 
      t.id === traderId ? { ...t, isFollowing: !t.isFollowing } : t
    ));
  };

  const handleCopyTrade = (trader: Trader) => {
    const newPosition: CopyPosition = {
      id: Date.now().toString(),
      traderId: trader.id,
      traderName: trader.username,
      symbol: trader.tradingPair,
      side: trader.dailyPnl >= 0 ? 'long' : 'short',
      size: parseFloat(copyAmount) / 1000,
      entryPrice: Math.random() * 1000 + 10,
      currentPrice: Math.random() * 1000 + 10,
      pnl: 0,
      pnlPercent: 0,
      openTime: Date.now()
    };
    setCopyPositions([...copyPositions, newPosition]);
    setActiveTab('my-copies');
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
