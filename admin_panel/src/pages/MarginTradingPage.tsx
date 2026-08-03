// Margin Trading Management Admin Page
// Full control over margin trading pairs, positions, and liquidations

import React, { useState, useEffect } from 'react';
import './MarginTradingPage.css';

interface MarginPair {
  id: string;
  symbol: string;
  base: string;
  quote: string;
  maxLeverage: number;
  defaultLeverage: number;
  maintenanceMargin: number;
  liquidationThreshold: number;
  interestRate: number;
  status: 'active' | 'suspended' | 'halted';
  minOrderSize: number;
  maxOrderSize: number;
  makerFee: number;
  takerFee: number;
}

interface MarginPosition {
  id: string;
  userId: string;
  username: string;
  symbol: string;
  side: 'long' | 'short';
  leverage: number;
  entryPrice: number;
  markPrice: number;
  size: number;
  margin: number;
  unrealizedPnl: number;
  roe: number;
  liquidationPrice: number;
  status: 'open' | 'liquidated' | 'closed';
  openedAt: string;
}

interface LiquidationRecord {
  id: string;
  userId: string;
  username: string;
  symbol: string;
  side: 'long' | 'short';
  leverage: number;
  size: number;
  margin: number;
  liquidationPrice: number;
  markPrice: number;
  remaining: number;
  fee: number;
  timestamp: string;
}

interface MarginSettings {
  enabled: boolean;
  defaultLeverage: number;
  maxLeverage: number;
  liquidationFee: number;
  insuranceFund: number;
  autoDeleverage: boolean;
  bankruptcyThreshold: number;
}

const MarginTradingPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'pairs' | 'positions' | 'liquidations' | 'settings'>('pairs');
  const [pairs, setPairs] = useState<MarginPair[]>([]);
  const [positions, setPositions] = useState<MarginPosition[]>([]);
  const [liquidations, setLiquidations] = useState<LiquidationRecord[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [filterSide, setFilterSide] = useState<string>('all');
  const [selectedPair, setSelectedPair] = useState<MarginPair | null>(null);
  const [showPairModal, setShowPairModal] = useState(false);

  const [marginSettings, setMarginSettings] = useState<MarginSettings>({
    enabled: true,
    defaultLeverage: 10,
    maxLeverage: 125,
    liquidationFee: 0.015,
    insuranceFund: 2500000,
    autoDeleverage: true,
    bankruptcyThreshold: 0.005,
  });

  // Initialize with sample data
  useEffect(() => {
    // Sample margin pairs
    const samplePairs: MarginPair[] = [
      { id: '1', symbol: 'BTC/USDT', base: 'BTC', quote: 'USDT', maxLeverage: 125, defaultLeverage: 10, maintenanceMargin: 0.005, liquidationThreshold: 0.004, interestRate: 0.0001, status: 'active', minOrderSize: 0.001, maxOrderSize: 100, makerFee: 0.01, takerFee: 0.04 },
      { id: '2', symbol: 'ETH/USDT', base: 'ETH', quote: 'USDT', maxLeverage: 100, defaultLeverage: 10, maintenanceMargin: 0.01, liquidationThreshold: 0.008, interestRate: 0.0001, status: 'active', minOrderSize: 0.01, maxOrderSize: 1000, makerFee: 0.01, takerFee: 0.04 },
      { id: '3', symbol: 'BNB/USDT', base: 'BNB', quote: 'USDT', maxLeverage: 75, defaultLeverage: 10, maintenanceMargin: 0.015, liquidationThreshold: 0.012, interestRate: 0.0001, status: 'active', minOrderSize: 0.1, maxOrderSize: 10000, makerFee: 0.01, takerFee: 0.04 },
      { id: '4', symbol: 'SOL/USDT', base: 'SOL', quote: 'USDT', maxLeverage: 50, defaultLeverage: 10, maintenanceMargin: 0.02, liquidationThreshold: 0.015, interestRate: 0.0002, status: 'active', minOrderSize: 0.1, maxOrderSize: 50000, makerFee: 0.02, takerFee: 0.05 },
      { id: '5', symbol: 'XRP/USDT', base: 'XRP', quote: 'USDT', maxLeverage: 50, defaultLeverage: 10, maintenanceMargin: 0.02, liquidationThreshold: 0.015, interestRate: 0.0002, status: 'suspended', minOrderSize: 10, maxOrderSize: 1000000, makerFee: 0.02, takerFee: 0.05 },
      { id: '6', symbol: 'DOGE/USDT', base: 'DOGE', quote: 'USDT', maxLeverage: 50, defaultLeverage: 5, maintenanceMargin: 0.025, liquidationThreshold: 0.02, interestRate: 0.0003, status: 'active', minOrderSize: 100, maxOrderSize: 10000000, makerFee: 0.02, takerFee: 0.05 },
    ];

    // Generate more pairs
    const tokens = ['ADA', 'AVAX', 'DOT', 'LINK', 'MATIC', 'LTC', 'UNI', 'ATOM', 'XLM', 'NEAR', 'APT', 'ARB', 'OP', 'INJ', 'PEPE', 'SHIB', 'TRX', 'FIL', 'ALGO', 'VET'];
    tokens.forEach((token, idx) => {
      samplePairs.push({
        id: String(idx + 7),
        symbol: `${token}/USDT`,
        base: token,
        quote: 'USDT',
        maxLeverage: 50,
        defaultLeverage: 10,
        maintenanceMargin: 0.02,
        liquidationThreshold: 0.015,
        interestRate: 0.0002,
        status: 'active',
        minOrderSize: 1,
        maxOrderSize: 100000,
        makerFee: 0.02,
        takerFee: 0.05,
      });
    });
    setPairs(samplePairs);

    // Sample positions
    const samplePositions: MarginPosition[] = [
      { id: '1', userId: 'u1', username: 'TraderKing', symbol: 'BTC/USDT', side: 'long', leverage: 50, entryPrice: 42000, markPrice: 42500, size: 0.5, margin: 425, unrealizedPnl: 250, roe: 58.82, liquidationPrice: 40800, status: 'open', openedAt: '2024-01-15 08:00:00' },
      { id: '2', userId: 'u2', username: 'LeverageLover', symbol: 'ETH/USDT', side: 'short', leverage: 75, entryPrice: 2350, markPrice: 2280, size: 10, margin: 304, unrealizedPnl: 933.33, roe: 307.28, liquidationPrice: 2450, status: 'open', openedAt: '2024-01-15 06:30:00' },
      { id: '3', userId: 'u3', username: 'WhaleTrader', symbol: 'BTC/USDT', side: 'long', leverage: 100, entryPrice: 41500, markPrice: 42500, size: 2, margin: 850, unrealizedPnl: 2000, roe: 235.29, liquidationPrice: 41750, status: 'open', openedAt: '2024-01-14 22:00:00' },
      { id: '4', userId: 'u4', username: 'DeFiDegen', symbol: 'SOL/USDT', side: 'long', leverage: 30, entryPrice: 95, markPrice: 98, size: 500, margin: 1633.33, unrealizedPnl: 1500, roe: 91.84, liquidationPrice: 88, status: 'open', openedAt: '2024-01-15 10:00:00' },
      { id: '5', userId: 'u5', username: 'MarginMax', symbol: 'BNB/USDT', side: 'short', leverage: 50, entryPrice: 320, markPrice: 312, size: 100, margin: 624, unrealizedPnl: 800, roe: 128.21, liquidationPrice: 335, status: 'open', openedAt: '2024-01-15 09:30:00' },
      { id: '6', userId: 'u6', username: 'LiquidationHunter', symbol: 'BTC/USDT', side: 'long', leverage: 125, entryPrice: 43000, markPrice: 42500, size: 0.1, margin: 34, unrealizedPnl: -50, roe: -147.06, liquidationPrice: 42960, status: 'open', openedAt: '2024-01-15 11:00:00' },
    ];
    setPositions(samplePositions);

    // Sample liquidations
    const sampleLiquidations: LiquidationRecord[] = [
      { id: 'l1', userId: 'u10', username: 'LiquidatedUser1', symbol: 'BTC/USDT', side: 'long', leverage: 100, size: 0.3, margin: 127.50, liquidationPrice: 40850, markPrice: 40750, remaining: 0, fee: 611.25, timestamp: '2024-01-15 08:45:00' },
      { id: 'l2', userId: 'u11', username: 'LiquidatedUser2', symbol: 'ETH/USDT', side: 'short', leverage: 50, size: 5, margin: 228, liquidationPrice: 2420, markPrice: 2430, remaining: 0, fee: 364.50, timestamp: '2024-01-15 07:30:00' },
      { id: 'l3', userId: 'u12', username: 'LiquidatedUser3', symbol: 'SOL/USDT', side: 'long', leverage: 40, size: 200, margin: 490, liquidationPrice: 86, markPrice: 85.50, remaining: 0, fee: 268.50, timestamp: '2024-01-14 23:15:00' },
    ];
    setLiquidations(sampleLiquidations);
  }, []);

  const filteredPairs = pairs.filter(pair => {
    if (filterStatus !== 'all' && pair.status !== filterStatus) return false;
    if (searchTerm && !pair.symbol.toLowerCase().includes(searchTerm.toLowerCase())) return false;
    return true;
  });

  const filteredPositions = positions.filter(pos => {
    if (filterStatus !== 'all' && pos.status !== filterStatus) return false;
    if (filterSide !== 'all' && pos.side !== filterSide) return false;
    if (searchTerm && !pos.symbol.toLowerCase().includes(searchTerm.toLowerCase()) &&
        !pos.username.toLowerCase().includes(searchTerm.toLowerCase())) return false;
    return true;
  });

  const handleTogglePair = (pairId: string) => {
    setPairs(pairs.map(p => 
      p.id === pairId ? { ...p, status: p.status === 'active' ? 'suspended' as const : 'active' as const } : p
    ));
  };

  const handleClosePosition = (positionId: string) => {
    setPositions(positions.map(p => 
      p.id === positionId ? { ...p, status: 'closed' as const } : p
    ));
  };

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      active: '#28a745',
      suspended: '#ffc107',
      halted: '#dc3545',
      open: '#28a745',
      liquidated: '#dc3545',
      closed: '#6c757d',
      long: '#28a745',
      short: '#dc3545',
    };
    return colors[status] || '#6c757d';
  };

  // Stats
  const stats = {
    totalPairs: pairs.length,
    activePairs: pairs.filter(p => p.status === 'active').length,
    totalPositions: positions.filter(p => p.status === 'open').length,
    totalVolume: positions.filter(p => p.status === 'open').reduce((sum, p) => sum + p.size * p.markPrice, 0),
    totalUnrealizedPnl: positions.filter(p => p.status === 'open').reduce((sum, p) => sum + p.unrealizedPnl, 0),
    insuranceFund: marginSettings.insuranceFund,
  };

  return (
    <div className="margin-trading-page">
      <div className="page-header">
        <h1>Margin Trading Management</h1>
        <div className="header-actions">
          <button className="add-btn">+ Add Margin Pair</button>
          <button className="export-btn">Export Data</button>
        </div>
      </div>

      <div className="stats-cards">
        <div className="stat-card">
          <div className="stat-icon">🔄</div>
          <div className="stat-info">
            <span className="stat-value">{stats.totalPairs}</span>
            <span className="stat-label">Margin Pairs</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">✅</div>
          <div className="stat-info">
            <span className="stat-value">{stats.activePairs}</span>
            <span className="stat-label">Active Pairs</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">📊</div>
          <div className="stat-info">
            <span className="stat-value">{stats.totalPositions}</span>
            <span className="stat-label">Open Positions</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">💵</div>
          <div className="stat-info">
            <span className="stat-value">${(stats.totalVolume / 1000000).toFixed(2)}M</span>
            <span className="stat-label">Open Interest</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">📈</div>
          <div className="stat-info">
            <span className={`stat-value ${stats.totalUnrealizedPnl >= 0 ? 'positive' : 'negative'}`}>
              {stats.totalUnrealizedPnl >= 0 ? '+' : ''}${stats.totalUnrealizedPnl.toFixed(2)}
            </span>
            <span className="stat-label">Unrealized PnL</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">🛡️</div>
          <div className="stat-info">
            <span className="stat-value">${(stats.insuranceFund / 1000000).toFixed(2)}M</span>
            <span className="stat-label">Insurance Fund</span>
          </div>
        </div>
      </div>

      <div className="tabs">
        <button 
          className={activeTab === 'pairs' ? 'active' : ''} 
          onClick={() => setActiveTab('pairs')}
        >
          🔄 Margin Pairs ({pairs.length})
        </button>
        <button 
          className={activeTab === 'positions' ? 'active' : ''} 
          onClick={() => setActiveTab('positions')}
        >
          📊 Positions ({positions.filter(p => p.status === 'open').length})
        </button>
        <button 
          className={activeTab === 'liquidations' ? 'active' : ''} 
          onClick={() => setActiveTab('liquidations')}
        >
          ⚠️ Liquidations ({liquidations.length})
        </button>
        <button 
          className={activeTab === 'settings' ? 'active' : ''} 
          onClick={() => setActiveTab('settings')}
        >
          ⚙️ Settings
        </button>
      </div>

      {activeTab === 'pairs' && (
        <div className="pairs-section">
          <div className="filters">
            <input
              type="text"
              placeholder="Search pairs..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
            <select 
              value={filterStatus} 
              onChange={(e) => setFilterStatus(e.target.value)}
              className="filter-select"
            >
              <option value="all">All Status</option>
              <option value="active">Active</option>
              <option value="suspended">Suspended</option>
              <option value="halted">Halted</option>
            </select>
          </div>

          <div className="pairs-table">
            <table>
              <thead>
                <tr>
                  <th>Symbol</th>
                  <th>Max Leverage</th>
                  <th>Default Leverage</th>
                  <th>Maintenance Margin</th>
                  <th>Liquidation Threshold</th>
                  <th>Interest Rate</th>
                  <th>Min Size</th>
                  <th>Max Size</th>
                  <th>Maker Fee</th>
                  <th>Taker Fee</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filteredPairs.map(pair => (
                  <tr key={pair.id}>
                    <td className="symbol">{pair.symbol}</td>
                    <td>{pair.maxLeverage}x</td>
                    <td>{pair.defaultLeverage}x</td>
                    <td>{(pair.maintenanceMargin * 100).toFixed(1)}%</td>
                    <td>{(pair.liquidationThreshold * 100).toFixed(1)}%</td>
                    <td>{(pair.interestRate * 100).toFixed(2)}%</td>
                    <td>{pair.minOrderSize}</td>
                    <td>{pair.maxOrderSize.toLocaleString()}</td>
                    <td>{pair.makerFee}%</td>
                    <td>{pair.takerFee}%</td>
                    <td>
                      <span 
                        className="status-badge"
                        style={{ backgroundColor: getStatusColor(pair.status) }}
                      >
                        {pair.status}
                      </span>
                    </td>
                    <td>
                      <div className="actions">
                        <button 
                          className="action-btn edit"
                          onClick={() => { setSelectedPair(pair); setShowPairModal(true); }}
                        >
                          Edit
                        </button>
                        <button 
                          className="action-btn toggle"
                          onClick={() => handleTogglePair(pair.id)}
                        >
                          {pair.status === 'active' ? 'Suspend' : 'Activate'}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'positions' && (
        <div className="positions-section">
          <div className="filters">
            <input
              type="text"
              placeholder="Search positions..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
            <select 
              value={filterStatus} 
              onChange={(e) => setFilterStatus(e.target.value)}
              className="filter-select"
            >
              <option value="all">All Status</option>
              <option value="open">Open</option>
              <option value="liquidated">Liquidated</option>
              <option value="closed">Closed</option>
            </select>
            <select 
              value={filterSide} 
              onChange={(e) => setFilterSide(e.target.value)}
              className="filter-select"
            >
              <option value="all">All Sides</option>
              <option value="long">Long</option>
              <option value="short">Short</option>
            </select>
          </div>

          <div className="positions-table">
            <table>
              <thead>
                <tr>
                  <th>User</th>
                  <th>Symbol</th>
                  <th>Side</th>
                  <th>Leverage</th>
                  <th>Entry Price</th>
                  <th>Mark Price</th>
                  <th>Size</th>
                  <th>Margin</th>
                  <th>Unrealized PnL</th>
                  <th>ROE</th>
                  <th>Liq. Price</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filteredPositions.map(pos => (
                  <tr key={pos.id}>
                    <td>
                      <div className="user-info">
                        <span className="username">{pos.username}</span>
                        <span className="user-id">{pos.userId}</span>
                      </div>
                    </td>
                    <td className="symbol">{pos.symbol}</td>
                    <td>
                      <span 
                        className="side-badge"
                        style={{ backgroundColor: getStatusColor(pos.side) }}
                      >
                        {pos.side.toUpperCase()}
                      </span>
                    </td>
                    <td>{pos.leverage}x</td>
                    <td>${pos.entryPrice.toLocaleString()}</td>
                    <td>${pos.markPrice.toLocaleString()}</td>
                    <td>{pos.size}</td>
                    <td>${pos.margin.toLocaleString()}</td>
                    <td className={pos.unrealizedPnl >= 0 ? 'pnl-positive' : 'pnl-negative'}>
                      {pos.unrealizedPnl >= 0 ? '+' : ''}${pos.unrealizedPnl.toFixed(2)}
                    </td>
                    <td className={pos.roe >= 0 ? 'pnl-positive' : 'pnl-negative'}>
                      {pos.roe >= 0 ? '+' : ''}{pos.roe.toFixed(2)}%
                    </td>
                    <td className="liq-price">${pos.liquidationPrice.toLocaleString()}</td>
                    <td>
                      <span 
                        className="status-badge"
                        style={{ backgroundColor: getStatusColor(pos.status) }}
                      >
                        {pos.status}
                      </span>
                    </td>
                    <td>
                      <div className="actions">
                        {pos.status === 'open' && (
                          <button 
                            className="action-btn close"
                            onClick={() => handleClosePosition(pos.id)}
                          >
                            Close
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'liquidations' && (
        <div className="liquidations-section">
          <div className="liquidations-table">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>User</th>
                  <th>Symbol</th>
                  <th>Side</th>
                  <th>Leverage</th>
                  <th>Size</th>
                  <th>Margin</th>
                  <th>Liq. Price</th>
                  <th>Mark Price</th>
                  <th>Liq. Fee</th>
                  <th>Timestamp</th>
                </tr>
              </thead>
              <tbody>
                {liquidations.map(liq => (
                  <tr key={liq.id}>
                    <td className="liq-id">{liq.id}</td>
                    <td>
                      <div className="user-info">
                        <span className="username">{liq.username}</span>
                        <span className="user-id">{liq.userId}</span>
                      </div>
                    </td>
                    <td className="symbol">{liq.symbol}</td>
                    <td>
                      <span 
                        className="side-badge"
                        style={{ backgroundColor: getStatusColor(liq.side) }}
                      >
                        {liq.side.toUpperCase()}
                      </span>
                    </td>
                    <td>{liq.leverage}x</td>
                    <td>{liq.size}</td>
                    <td>${liq.margin.toLocaleString()}</td>
                    <td>${liq.liquidationPrice.toLocaleString()}</td>
                    <td>${liq.markPrice.toLocaleString()}</td>
                    <td className="liq-fee">${liq.fee.toFixed(2)}</td>
                    <td>{liq.timestamp}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'settings' && (
        <div className="settings-section">
          <div className="settings-group">
            <h3>General Settings</h3>
            <div className="setting-item">
              <label>
                <input 
                  type="checkbox" 
                  checked={marginSettings.enabled}
                  onChange={(e) => setMarginSettings({ ...marginSettings, enabled: e.target.checked })}
                />
                Enable Margin Trading
              </label>
            </div>
            <div className="setting-item">
              <label>
                <input 
                  type="checkbox" 
                  checked={marginSettings.autoDeleverage}
                  onChange={(e) => setMarginSettings({ ...marginSettings, autoDeleverage: e.target.checked })}
                />
                Enable Auto-Deleverage
              </label>
            </div>
          </div>

          <div className="settings-group">
            <h3>Leverage Settings</h3>
            <div className="setting-item">
              <label>Default Leverage</label>
              <input 
                type="number" 
                value={marginSettings.defaultLeverage}
                onChange={(e) => setMarginSettings({ ...marginSettings, defaultLeverage: parseInt(e.target.value) })}
              />
            </div>
            <div className="setting-item">
              <label>Maximum Leverage</label>
              <input 
                type="number" 
                value={marginSettings.maxLeverage}
                onChange={(e) => setMarginSettings({ ...marginSettings, maxLeverage: parseInt(e.target.value) })}
              />
            </div>
          </div>

          <div className="settings-group">
            <h3>Liquidation Settings</h3>
            <div className="setting-item">
              <label>Liquidation Fee (%)</label>
              <input 
                type="number" 
                step="0.001"
                value={marginSettings.liquidationFee * 100}
                onChange={(e) => setMarginSettings({ ...marginSettings, liquidationFee: parseFloat(e.target.value) / 100 })}
              />
            </div>
            <div className="setting-item">
              <label>Bankruptcy Threshold (%)</label>
              <input 
                type="number" 
                step="0.001"
                value={marginSettings.bankruptcyThreshold * 100}
                onChange={(e) => setMarginSettings({ ...marginSettings, bankruptcyThreshold: parseFloat(e.target.value) / 100 })}
              />
            </div>
          </div>

          <div className="settings-group">
            <h3>Insurance Fund</h3>
            <div className="setting-item">
              <label>Current Balance ($)</label>
              <span className="insurance-value">${marginSettings.insuranceFund.toLocaleString()}</span>
            </div>
          </div>

          <div className="settings-actions">
            <button className="save-btn">Save Settings</button>
          </div>
        </div>
      )}

      {/* Pair Edit Modal */}
      {showPairModal && selectedPair && (
        <div className="modal-overlay" onClick={() => setShowPairModal(false)}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <h2>Edit Margin Pair - {selectedPair.symbol}</h2>
            <div className="pair-form">
              <div className="form-row">
                <div className="form-group">
                  <label>Max Leverage</label>
                  <input type="number" defaultValue={selectedPair.maxLeverage} id="maxLeverageInput" />
                </div>
                <div className="form-group">
                  <label>Default Leverage</label>
                  <input type="number" defaultValue={selectedPair.defaultLeverage} id="defaultLeverageInput" />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Maintenance Margin (%)</label>
                  <input type="number" step="0.001" defaultValue={selectedPair.maintenanceMargin * 100} id="maintenanceMarginInput" />
                </div>
                <div className="form-group">
                  <label>Liquidation Threshold (%)</label>
                  <input type="number" step="0.001" defaultValue={selectedPair.liquidationThreshold * 100} id="liquidationThresholdInput" />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Maker Fee (%)</label>
                  <input type="number" step="0.01" defaultValue={selectedPair.makerFee} id="makerFeeInput" />
                </div>
                <div className="form-group">
                  <label>Taker Fee (%)</label>
                  <input type="number" step="0.01" defaultValue={selectedPair.takerFee} id="takerFeeInput" />
                </div>
              </div>
            </div>
            <div className="modal-actions">
              <button className="cancel-btn" onClick={() => setShowPairModal(false)}>
                Cancel
              </button>
              <button className="submit-btn" onClick={() => setShowPairModal(false)}>
                Save Changes
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default MarginTradingPage;
