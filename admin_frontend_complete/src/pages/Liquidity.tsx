/**
 * TigerWallet Admin - Liquidity Management Page
 * Complete implementation with backend connectivity
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { liquidityAPI } from '../services/api';

interface LiquidityPool {
  id: string;
  pair: string;
  tokenA: string;
  tokenB: string;
  reserveA: number;
  reserveB: number;
  totalSupply: number;
  apr: number;
  volume24h: number;
  fees24h: number;
  status: 'active' | 'inactive';
  createdAt: string;
}

export const LiquidityPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [pools, setPools] = useState<LiquidityPool[]>([]);
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<any>(null);
  const [showAddModal, setShowAddModal] = useState(false);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [poolsRes, statsRes] = await Promise.all([
        liquidityAPI.getPools(),
        liquidityAPI.getStats(),
      ]);
      setPools(poolsRes.data);
      setStats(statsRes.data);
    } catch (error) {
      console.error('Failed to load liquidity data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAddLiquidity = async (poolId: string, amountA: number, amountB: number) => {
    try {
      await liquidityAPI.addLiquidity(poolId, { amountA, amountB });
      loadData();
      setShowAddModal(false);
    } catch (error) {
      console.error('Failed to add liquidity:', error);
    }
  };

  const handleRemoveLiquidity = async (poolId: string, amount: number) => {
    try {
      await liquidityAPI.removeLiquidity(poolId, { amount });
      loadData();
    } catch (error) {
      console.error('Failed to remove liquidity:', error);
    }
  };

  return (
    <div className={`page-container ${isDark ? 'dark' : 'light'}`}>
      <div className="page-header">
        <h1>Liquidity Management</h1>
        <button className="theme-btn" onClick={toggleTheme}>
          {isDark ? '☀️ Light' : '🌙 Dark'}
        </button>
      </div>

      {stats && (
        <div className="stats-grid">
          <div className="stat-card">
            <div className="stat-value">{stats.totalPools}</div>
            <div className="stat-label">Total Pools</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">${stats.totalValueLocked.toLocaleString()}</div>
            <div className="stat-label">Total Value Locked</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">${stats.volume24h.toLocaleString()}</div>
            <div className="stat-label">24h Volume</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">${stats.fees24h.toLocaleString()}</div>
            <div className="stat-label">24h Fees</div>
          </div>
        </div>
      )}

      {loading ? (
        <div className="loading">Loading pools...</div>
      ) : (
        <div className="pools-grid">
          {pools.map(pool => (
            <div key={pool.id} className="pool-card">
              <div className="pool-header">
                <h3>{pool.pair}</h3>
                <span className={`status-badge ${pool.status}`}>
                  {pool.status}
                </span>
              </div>
              <div className="pool-details">
                <div className="pool-pair">
                  <span className="token">{pool.tokenA}</span>
                  <span className="arrow">⟷</span>
                  <span className="token">{pool.tokenB}</span>
                </div>
                <div className="pool-stats">
                  <div className="stat-row">
                    <span>Reserve A:</span>
                    <span>{pool.reserveA.toLocaleString()}</span>
                  </div>
                  <div className="stat-row">
                    <span>Reserve B:</span>
                    <span>{pool.reserveB.toLocaleString()}</span>
                  </div>
                  <div className="stat-row">
                    <span>Total Supply:</span>
                    <span>{pool.totalSupply.toLocaleString()}</span>
                  </div>
                  <div className="stat-row highlight">
                    <span>APR:</span>
                    <span>{pool.apr.toFixed(2)}%</span>
                  </div>
                  <div className="stat-row">
                    <span>24h Volume:</span>
                    <span>${pool.volume24h.toLocaleString()}</span>
                  </div>
                  <div className="stat-row">
                    <span>24h Fees:</span>
                    <span>${pool.fees24h.toLocaleString()}</span>
                  </div>
                </div>
              </div>
              <div className="pool-actions">
                <button
                  className="btn-primary"
                  onClick={() => setShowAddModal(true)}
                >
                  Add Liquidity
                </button>
                <button
                  className="btn-secondary"
                  onClick={() => handleRemoveLiquidity(pool.id, pool.totalSupply * 0.1)}
                >
                  Remove 10%
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {showAddModal && (
        <div className="modal-overlay" onClick={() => setShowAddModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Liquidity</h2>
            <button className="close-btn" onClick={() => setShowAddModal(false)}>×</button>
            <form onSubmit={(e) => {
              e.preventDefault();
              const form = e.target as HTMLFormElement;
              const amountA = parseFloat((form.elements.namedItem('amountA') as HTMLInputElement).value);
              const amountB = parseFloat((form.elements.namedItem('amountB') as HTMLInputElement).value);
              handleAddLiquidity(pools[0]?.id || '', amountA, amountB);
            }}>
              <div className="form-group">
                <label>Token A Amount</label>
                <input type="number" name="amountA" required placeholder="0.00" />
              </div>
              <div className="form-group">
                <label>Token B Amount</label>
                <input type="number" name="amountB" required placeholder="0.00" />
              </div>
              <button type="submit" className="btn-primary">Add Liquidity</button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default LiquidityPage;
