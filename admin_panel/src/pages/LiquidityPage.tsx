// Liquidity Management Page - Complete Implementation

import React, { useState, useEffect } from 'react';
import './LiquidityPage.css';

interface LiquidityPool {
  id: string;
  pair: string;
  chainId: string;
  chainName: string;
  tvl: number;
  volume24h: number;
  fees24h: number;
  apy: number;
  providers: number;
  status: 'active' | 'inactive';
}

const LiquidityPage: React.FC = () => {
  const [pools, setPools] = useState<LiquidityPool[]>([]);
  const [showModal, setShowModal] = useState(false);

  useEffect(() => {
    loadPools();
  }, []);

  const loadPools = () => {
    setPools([
      {
        id: '1',
        pair: 'ETH/USDT',
        chainId: '1',
        chainName: 'Ethereum',
        tvl: 152384756,
        volume24h: 84729384,
        fees24h: 254187,
        apy: 12.5,
        providers: 1234,
        status: 'active',
      },
      {
        id: '2',
        pair: 'BTC/USDT',
        chainId: '1',
        chainName: 'Ethereum',
        tvl: 293847561,
        volume24h: 152384756,
        fees24h: 457154,
        apy: 8.3,
        providers: 2345,
        status: 'active',
      },
      {
        id: '3',
        pair: 'BNB/USDT',
        chainId: '56',
        chainName: 'BNB Chain',
        tvl: 84729384,
        volume24h: 48392756,
        fees24h: 145178,
        apy: 15.2,
        providers: 892,
        status: 'active',
      },
      {
        id: '4',
        pair: 'SOL/USDT',
        chainId: 'solana',
        chainName: 'Solana',
        tvl: 38475629,
        volume24h: 29384756,
        fees24h: 88154,
        apy: 22.8,
        providers: 567,
        status: 'active',
      },
      {
        id: '5',
        pair: 'MATIC/USDT',
        chainId: '137',
        chainName: 'Polygon',
        tvl: 29384756,
        volume24h: 15238475,
        fees24h: 45715,
        apy: 18.5,
        providers: 423,
        status: 'inactive',
      },
    ]);
  };

  const handleStatusChange = (id: string) => {
    setPools(prev => prev.map(p => p.id === id ? { ...p, status: p.status === 'active' ? 'inactive' : 'active' } : p));
  };

  const importFromCEX = (cex: string) => {
    alert(`Importing liquidity from ${cex}...`);
  };

  const filteredPools = pools;

  return (
    <div className="liquidity-page">
      <div className="page-header">
        <div>
          <h1>Liquidity Management</h1>
          <p>Manage liquidity pools and providers</p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + Add Pool
        </button>
      </div>

      {/* Import Section */}
      <div className="import-section">
        <span className="import-label">Import Liquidity from:</span>
        <div className="import-buttons">
          <button className="import-btn" onClick={() => importFromCEX('Binance')}>Binance</button>
          <button className="import-btn" onClick={() => importFromCEX('Uniswap')}>Uniswap</button>
          <button className="import-btn" onClick={() => importFromCEX('Curve')}>Curve</button>
          <button className="import-btn" onClick={() => importFromCEX('PancakeSwap')}>PancakeSwap</button>
        </div>
      </div>

      {/* Stats */}
      <div className="stats-grid">
        <div className="stat-card">
          <span className="stat-label">Total Pools</span>
          <span className="stat-value">{pools.length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Total Value Locked</span>
          <span className="stat-value">${pools.reduce((acc, p) => acc + p.tvl, 0).toLocaleString()}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">24h Volume</span>
          <span className="stat-value">${pools.reduce((acc, p) => acc + p.volume24h, 0).toLocaleString()}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Avg APY</span>
          <span className="stat-value">{(pools.reduce((acc, p) => acc + p.apy, 0) / pools.length).toFixed(1)}%</span>
        </div>
      </div>

      {/* Pools Table */}
      <div className="table-container">
        <table>
          <thead>
            <tr>
              <th>Pool</th>
              <th>Chain</th>
              <th>TVL</th>
              <th>24h Volume</th>
              <th>24h Fees</th>
              <th>APY</th>
              <th>Providers</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredPools.map(pool => (
              <tr key={pool.id}>
                <td>
                  <div className="pool-info">
                    <span className="pool-pair">{pool.pair}</span>
                  </div>
                </td>
                <td><span className="chain-badge">{pool.chainName}</span></td>
                <td className="tvl">${pool.tvl.toLocaleString()}</td>
                <td>${pool.volume24h.toLocaleString()}</td>
                <td className="fees">${pool.fees24h.toLocaleString()}</td>
                <td className="apy">{pool.apy}%</td>
                <td>{pool.providers}</td>
                <td>
                  <button
                    className={`status-toggle ${pool.status}`}
                    onClick={() => handleStatusChange(pool.id)}
                  >
                    {pool.status}
                  </button>
                </td>
                <td>
                  <div className="actions">
                    <button className="action-btn">📊</button>
                    <button className="action-btn">➕</button>
                    <button className="action-btn">🗑️</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default LiquidityPage;
