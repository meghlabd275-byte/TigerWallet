/**
 * TigerWallet Admin - Margin Trading Management Page
 * Complete implementation with backend connectivity
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { marginTradingAPI } from '../services/api';

interface MarginPosition {
  id: string;
  userId: string;
  userName: string;
  pair: string;
  side: 'long' | 'short';
  size: number;
  leverage: number;
  entryPrice: number;
  currentPrice: number;
  pnl: number;
  liquidationPrice: number;
  status: 'open' | 'liquidated' | 'closed';
  openedAt: string;
}

export const MarginTradingPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [positions, setPositions] = useState<MarginPosition[]>([]);
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<any>(null);
  const [filter, setFilter] = useState('all');

  useEffect(() => {
    loadData();
  }, [filter]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [positionsRes, statsRes] = await Promise.all([
        marginTradingAPI.getPositions(),
        marginTradingAPI.getLiquidationStats(),
      ]);
      setPositions(positionsRes.data);
      setStats(statsRes.data);
    } catch (error) {
      console.error('Failed to load margin data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleLiquidate = async (positionId: string) => {
    if (!confirm('Are you sure you want to liquidate this position?')) return;
    try {
      await marginTradingAPI.liquidate(positionId);
      loadData();
    } catch (error) {
      console.error('Failed to liquidate position:', error);
    }
  };

  const filteredPositions = positions.filter(pos =>
    filter === 'all' || pos.status === filter
  );

  return (
    <div className={`page-container ${isDark ? 'dark' : 'light'}`}>
      <div className="page-header">
        <h1>Margin Trading Management</h1>
        <button className="theme-btn" onClick={toggleTheme}>
          {isDark ? '☀️ Light' : '🌙 Dark'}
        </button>
      </div>

      {stats && (
        <div className="stats-grid">
          <div className="stat-card">
            <div className="stat-value">{stats.totalPositions}</div>
            <div className="stat-label">Total Positions</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">${stats.totalVolume.toLocaleString()}</div>
            <div className="stat-label">Total Volume</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{stats.liquidationsToday}</div>
            <div className="stat-label">Liquidations Today</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">${stats.liquidatedVolume.toLocaleString()}</div>
            <div className="stat-label">Liquidated Volume</div>
          </div>
        </div>
      )}

      <div className="filters">
        <select value={filter} onChange={(e) => setFilter(e.target.value)}>
          <option value="all">All Positions</option>
          <option value="open">Open</option>
          <option value="liquidated">Liquidated</option>
          <option value="closed">Closed</option>
        </select>
      </div>

      {loading ? (
        <div className="loading">Loading positions...</div>
      ) : (
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>User</th>
                <th>Pair</th>
                <th>Side</th>
                <th>Size</th>
                <th>Leverage</th>
                <th>Entry Price</th>
                <th>Current Price</th>
                <th>PnL</th>
                <th>Liq. Price</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredPositions.map(pos => (
                <tr key={pos.id}>
                  <td>{pos.userName}</td>
                  <td>{pos.pair}</td>
                  <td>
                    <span className={`side-badge ${pos.side}`}>
                      {pos.side.toUpperCase()}
                    </span>
                  </td>
                  <td>{pos.size.toLocaleString()}</td>
                  <td>{pos.leverage}x</td>
                  <td>${pos.entryPrice.toLocaleString()}</td>
                  <td>${pos.currentPrice.toLocaleString()}</td>
                  <td className={pos.pnl >= 0 ? 'profit' : 'loss'}>
                    ${pos.pnl.toLocaleString()}
                  </td>
                  <td>${pos.liquidationPrice.toLocaleString()}</td>
                  <td>
                    <span className={`status-badge ${pos.status}`}>
                      {pos.status}
                    </span>
                  </td>
                  <td>
                    {pos.status === 'open' && (
                      <button
                        className="btn-danger"
                        onClick={() => handleLiquidate(pos.id)}
                      >
                        Liquidate
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

export default MarginTradingPage;
