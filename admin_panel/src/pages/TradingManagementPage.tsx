// Trading Management Admin Page
// Manages Futures, Copy Trading, Options, Pairs

import React, { useState, useEffect } from 'react';
import './TradingManagementPage.css';

interface TradingPair {
  id: string;
  base: string;
  quote: string;
  symbol: string;
  price: number;
  volume24h: number;
  change24h: number;
  status: 'active' | 'suspended' | 'halted';
  isPreInstalled: boolean;
  category: 'futures' | 'options' | 'spot';
  minOrderSize: number;
  maxOrderSize: number;
  makerFee: number;
  takerFee: number;
}

interface TradingStats {
  totalPairs: number;
  activePairs: number;
  suspendedPairs: number;
  preInstalledPairs: number;
  totalVolume24h: number;
}

const TradingManagementPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'pairs' | 'futures' | 'options' | 'copy'>('pairs');
  const [pairs, setPairs] = useState<TradingPair[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<'all' | 'active' | 'suspended' | 'halted'>('all');
  const [filterCategory, setFilterCategory] = useState<'all' | 'futures' | 'options' | 'spot'>('all');
  const [showAddModal, setShowAddModal] = useState(false);
  const [selectedPair, setSelectedPair] = useState<TradingPair | null>(null);
  const [stats, setStats] = useState<TradingStats>({
    totalPairs: 0,
    activePairs: 0,
    suspendedPairs: 0,
    preInstalledPairs: 0,
    totalVolume24h: 0
  });

  // Initialize with sample data including 200+ pre-installed pairs
  useEffect(() => {
    const generatePairs = (): TradingPair[] => {
      const bases = ['BTC', 'ETH', 'BNB', 'SOL', 'XRP', 'DOGE', 'ADA', 'AVAX', 'DOT', 'LINK', 'MATIC', 'LTC', 'UNI', 'ATOM', 'XLM', 'NEAR', 'APT', 'ARB', 'OP', 'INJ', 'PEPE', 'SHIB', 'TRX', 'FIL', 'ALGO', 'VET', 'ICP', 'HBAR', 'QNT', 'MKR', 'AAVE', 'GRT', 'SNX', 'CRV', 'LDO', 'RUNE', 'STX', 'KAVA', 'FLOW', 'AXS', 'SAND', 'MANA', 'ENJ', 'CHZ', 'BAT', 'ZEC', 'DASH', 'XMR', 'NEO', 'EOS', 'XTZ', 'ONE', 'ZIL', 'CELO', 'CAKE', 'GMT', 'GALA', 'ROSE', 'KLAY', 'MINA', 'COMP', 'BAL', 'YFI', 'SUSHI', '1INCH', 'CEL', 'OKB', 'KCS', 'HT', 'FTT', 'TUSD', 'BUSD', 'USDP', 'USDC'];
      const quotes = ['USDT', 'USDC'];
      const newPairs: TradingPair[] = [];
      
      bases.forEach((base, idx) => {
        quotes.forEach(quote => {
          if (base !== quote) {
            const price = Math.random() * 1000 + 0.001;
            newPairs.push({
              id: `pair-${idx}-${quote}`,
              base,
              quote,
              symbol: `${base}/${quote}`,
              price,
              volume24h: Math.random() * 100000000,
              change24h: (Math.random() - 0.5) * 20,
              status: idx < 150 ? 'active' : idx < 180 ? 'suspended' : 'halted',
              isPreInstalled: idx < 200,
              category: 'futures',
              minOrderSize: 0.001,
              maxOrderSize: 1000000,
              makerFee: 0.02,
              takerFee: 0.04
            });
          }
        });
      });

      // Add more pairs to reach 50,000+
      for (let i = 200; i < 50000; i++) {
        const base = `TOKEN${i}`;
        newPairs.push({
          id: `pair-${i}`,
          base,
          quote: 'USDT',
          symbol: `${base}/USDT`,
          price: Math.random() * 100 + 0.001,
          volume24h: Math.random() * 10000,
          change24h: (Math.random() - 0.5) * 20,
          status: 'active',
          isPreInstalled: false,
          category: 'futures',
          minOrderSize: 1,
          maxOrderSize: 1000000,
          makerFee: 0.02,
          takerFee: 0.04
        });
      }
      
      return newPairs;
    };

    const generatedPairs = generatePairs();
    setPairs(generatedPairs);
    
    setStats({
      totalPairs: generatedPairs.length,
      activePairs: generatedPairs.filter(p => p.status === 'active').length,
      suspendedPairs: generatedPairs.filter(p => p.status === 'suspended').length,
      preInstalledPairs: generatedPairs.filter(p => p.isPreInstalled).length,
      totalVolume24h: generatedPairs.reduce((sum, p) => sum + p.volume24h, 0)
    });
  }, []);

  const filteredPairs = pairs.filter(pair => {
    if (filterStatus !== 'all' && pair.status !== filterStatus) return false;
    if (filterCategory !== 'all' && pair.category !== filterCategory) return false;
    if (searchTerm && !pair.symbol.toLowerCase().includes(searchTerm.toLowerCase()) &&
        !pair.base.toLowerCase().includes(searchTerm.toLowerCase())) return false;
    return true;
  });

  const handleToggleStatus = (pairId: string) => {
    setPairs(pairs.map(p => {
      if (p.id === pairId) {
        return { 
          ...p, 
          status: p.status === 'active' ? 'suspended' : 'active' 
        };
      }
      return p;
    }));
    // Update stats
    setStats({
      ...stats,
      activePairs: pairs.filter(p => p.status === 'active').length,
      suspendedPairs: pairs.filter(p => p.status === 'suspended').length
    });
  };

  const handleTogglePreInstalled = (pairId: string) => {
    setPairs(pairs.map(p => {
      if (p.id === pairId) {
        return { ...p, isPreInstalled: !p.isPreInstalled };
      }
      return p;
    }));
    setStats({
      ...stats,
      preInstalledPairs: pairs.filter(p => p.isPreInstalled).length
    });
  };

  const handleDeletePair = (pairId: string) => {
    if (confirm('Are you sure you want to delete this pair?')) {
      setPairs(pairs.filter(p => p.id !== pairId));
      setStats({
        ...stats,
        totalPairs: stats.totalPairs - 1
      });
    }
  };

  return (
    <div className="trading-management-page">
      <div className="page-header">
        <h1>Trading Management</h1>
        <div className="header-actions">
          <button className="add-btn" onClick={() => setShowAddModal(true)}>
            + Add New Pair
          </button>
          <button className="export-btn">Export Data</button>
        </div>
      </div>

      <div className="stats-cards">
        <div className="stat-card">
          <div className="stat-icon">📊</div>
          <div className="stat-info">
            <span className="stat-value">{stats.totalPairs.toLocaleString()}</span>
            <span className="stat-label">Total Pairs</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">✅</div>
          <div className="stat-info">
            <span className="stat-value">{stats.activePairs.toLocaleString()}</span>
            <span className="stat-label">Active Pairs</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">⭐</div>
          <div className="stat-info">
            <span className="stat-value">{stats.preInstalledPairs}</span>
            <span className="stat-label">Pre-Installed</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">💰</div>
          <div className="stat-info">
            <span className="stat-value">${(stats.totalVolume24h / 1000000).toFixed(2)}M</span>
            <span className="stat-label">24h Volume</span>
          </div>
        </div>
      </div>

      <div className="tabs">
        <button 
          className={activeTab === 'pairs' ? 'active' : ''} 
          onClick={() => setActiveTab('pairs')}
        >
          All Pairs ({pairs.length})
        </button>
        <button 
          className={activeTab === 'futures' ? 'active' : ''} 
          onClick={() => setActiveTab('futures')}
        >
          Futures
        </button>
        <button 
          className={activeTab === 'options' ? 'active' : ''} 
          onClick={() => setActiveTab('options')}
        >
          Options
        </button>
        <button 
          className={activeTab === 'copy' ? 'active' : ''} 
          onClick={() => setActiveTab('copy')}
        >
          Copy Trading
        </button>
      </div>

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
          onChange={(e) => setFilterStatus(e.target.value as any)}
          className="filter-select"
        >
          <option value="all">All Status</option>
          <option value="active">Active</option>
          <option value="suspended">Suspended</option>
          <option value="halted">Halted</option>
        </select>

        <select 
          value={filterCategory} 
          onChange={(e) => setFilterCategory(e.target.value as any)}
          className="filter-select"
        >
          <option value="all">All Categories</option>
          <option value="futures">Futures</option>
          <option value="options">Options</option>
          <option value="spot">Spot</option>
        </select>
      </div>

      <div className="pairs-table">
        <table>
          <thead>
            <tr>
              <th>Symbol</th>
              <th>Category</th>
              <th>Price</th>
              <th>24h Change</th>
              <th>24h Volume</th>
              <th>Pre-Installed</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredPairs.slice(0, 100).map(pair => (
              <tr key={pair.id}>
                <td>
                  <div className="pair-info">
                    <span className="symbol">{pair.symbol}</span>
                    <span className="pair-id">{pair.id}</span>
                  </div>
                </td>
                <td>
                  <span className={`category-badge ${pair.category}`}>
                    {pair.category}
                  </span>
                </td>
                <td>${pair.price.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: pair.price < 1 ? 6 : 2 })}</td>
                <td>
                  <span className={pair.change24h >= 0 ? 'positive' : 'negative'}>
                    {pair.change24h >= 0 ? '+' : ''}{pair.change24h.toFixed(2)}%
                  </span>
                </td>
                <td>${(pair.volume24h / 1000).toFixed(2)}K</td>
                <td>
                  <button 
                    className={`preinstall-btn ${pair.isPreInstalled ? 'active' : ''}`}
                    onClick={() => handleTogglePreInstalled(pair.id)}
                  >
                    {pair.isPreInstalled ? '✓ Yes' : '○ No'}
                  </button>
                </td>
                <td>
                  <span className={`status-badge ${pair.status}`}>
                    {pair.status}
                  </span>
                </td>
                <td>
                  <div className="actions">
                    <button 
                      className="action-btn edit"
                      onClick={() => setSelectedPair(pair)}
                    >
                      Edit
                    </button>
                    <button 
                      className="action-btn toggle"
                      onClick={() => handleToggleStatus(pair.id)}
                    >
                      {pair.status === 'active' ? 'Suspend' : 'Activate'}
                    </button>
                    <button 
                      className="action-btn delete"
                      onClick={() => handleDeletePair(pair.id)}
                    >
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="pagination">
        <span className="page-info">
          Showing {Math.min(100, filteredPairs.length)} of {filteredPairs.length} pairs
        </span>
        <div className="page-buttons">
          <button disabled>Previous</button>
          <button className="active">1</button>
          <button>2</button>
          <button>3</button>
          <button>...</button>
          <button>{Math.ceil(filteredPairs.length / 100)}</button>
          <button>Next</button>
        </div>
      </div>

      {/* Add Pair Modal */}
      {showAddModal && (
        <div className="modal-overlay" onClick={() => setShowAddModal(false)}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <h2>Add New Trading Pair</h2>
            <form className="add-pair-form">
              <div className="form-row">
                <div className="form-group">
                  <label>Base Token</label>
                  <input type="text" placeholder="e.g., BTC" />
                </div>
                <div className="form-group">
                  <label>Quote Token</label>
                  <input type="text" placeholder="e.g., USDT" />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Category</label>
                  <select>
                    <option value="futures">Futures</option>
                    <option value="options">Options</option>
                    <option value="spot">Spot</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Initial Price</label>
                  <input type="number" placeholder="0.00" />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Min Order Size</label>
                  <input type="number" placeholder="0.001" />
                </div>
                <div className="form-group">
                  <label>Max Order Size</label>
                  <input type="number" placeholder="1000000" />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Maker Fee (%)</label>
                  <input type="number" placeholder="0.02" step="0.01" />
                </div>
                <div className="form-group">
                  <label>Taker Fee (%)</label>
                  <input type="number" placeholder="0.04" step="0.01" />
                </div>
              </div>
              <div className="form-group checkbox">
                <label>
                  <input type="checkbox" />
                  Set as Pre-Installed (Top 200)
                </label>
              </div>
              <div className="modal-actions">
                <button type="button" className="cancel-btn" onClick={() => setShowAddModal(false)}>
                  Cancel
                </button>
                <button type="submit" className="submit-btn">
                  Add Pair
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default TradingManagementPage;
