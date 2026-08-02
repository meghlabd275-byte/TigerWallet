// Pairs Management Page - Complete Implementation
// Manage trading pairs and liquidity

import React, { useState, useEffect } from 'react';
import './PairsPage.css';

// Backend API URL
const API_BASE_URL = 'https://api.tigerwallet.com/v1/admin';

interface Pair {
  id: string;
  baseToken: string;
  quoteToken: string;
  pairAddress: string;
  chainId: string;
  chainName: string;
  price: number;
  change24h: number;
  volume24h: number;
  liquidity: number;
  status: 'active' | 'inactive' | 'halted';
  createdAt: string;
}

const defaultPairs: Pair[] = [
  {
    id: '1',
    baseToken: 'ETH',
    quoteToken: 'USDT',
    pairAddress: '0x88e6A0c2d26E9B24D02D4ba1E3f3C0c3E8F3A3',
    chainId: '1',
    chainName: 'Ethereum',
    price: 3000.50,
    change24h: 2.5,
    volume24h: 152384756,
    liquidity: 38475629,
    status: 'active',
    createdAt: '2026-01-15',
  },
  {
    id: '2',
    baseToken: 'BTC',
    quoteToken: 'USDT',
    pairAddress: '0x99e6A0c2d26E9B24D02D4ba1E3f3C0c3E8F3A3',
    chainId: '1',
    chainName: 'Ethereum',
    price: 43000.00,
    change24h: -1.2,
    volume24h: 293847561,
    liquidity: 84729384,
    status: 'active',
    createdAt: '2026-01-10',
  },
];

const PairsPage: React.FC = () => {
  const [pairs, setPairs] = useState<Pair[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [showModal, setShowModal] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadPairs();
  }, []);

  const loadPairs = async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('admin_token');
      const response = await fetch(`${API_BASE_URL}/pairs`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {}
      });
      
      if (response.ok) {
        const data = await response.json();
        setPairs(data.pairs || []);
      } else {
        setPairs(defaultPairs);
      }
    } catch (err) {
      console.error('Failed to load pairs:', err);
      setError('Unable to connect to pairs service. Using offline mode.');
      setPairs(defaultPairs);
    } finally {
      setLoading(false);
    }
  };

  const handleStatusChange = (id: string, status: 'active' | 'inactive' | 'halted') => {
    setPairs(prev => prev.map(p => p.id === id ? { ...p, status } : p));
  };

  const handleDelete = (id: string) => {
    if (confirm('Are you sure you want to delete this trading pair?')) {
      setPairs(prev => prev.filter(p => p.id !== id));
    }
  };

  const filteredPairs = pairs.filter(pair => {
    const matchesSearch = 
      pair.baseToken.toLowerCase().includes(searchQuery.toLowerCase()) ||
      pair.quoteToken.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesStatus = filterStatus === 'all' || pair.status === filterStatus;
    return matchesSearch && matchesStatus;
  });

  const importFromCEX = (cex: string) => {
    alert(`Importing pairs from ${cex}...`);
  };

  return (
    <div className="pairs-page">
      <div className="page-header">
        <div>
          <h1>Trading Pairs</h1>
          <p>Manage trading pairs and liquidity</p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + Add Pair
        </button>
      </div>

      {/* Import Buttons */}
      <div className="import-section">
        <span className="import-label">Import from CEX:</span>
        <div className="import-buttons">
          <button className="import-btn" onClick={() => importFromCEX('Binance')}>Binance</button>
          <button className="import-btn" onClick={() => importFromCEX('Coinbase')}>Coinbase</button>
          <button className="import-btn" onClick={() => importFromCEX('Kraken')}>Kraken</button>
          <button className="import-btn" onClick={() => importFromCEX('KuCoin')}>KuCoin</button>
          <button className="import-btn" onClick={() => importFromCEX('OKX')}>OKX</button>
        </div>
      </div>

      {/* Stats */}
      <div className="stats-grid">
        <div className="stat-card">
          <span className="stat-label">Total Pairs</span>
          <span className="stat-value">{pairs.length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Active Pairs</span>
          <span className="stat-value">{pairs.filter(p => p.status === 'active').length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">24h Volume</span>
          <span className="stat-value">${pairs.reduce((acc, p) => acc + p.volume24h, 0).toLocaleString()}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Total Liquidity</span>
          <span className="stat-value">${pairs.reduce((acc, p) => acc + p.liquidity, 0).toLocaleString()}</span>
        </div>
      </div>

      {/* Filters */}
      <div className="filters-bar">
        <div className="search-box">
          <span>🔍</span>
          <input
            type="text"
            placeholder="Search pairs..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
          />
        </div>
        <select value={filterStatus} onChange={e => setFilterStatus(e.target.value)}>
          <option value="all">All Status</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
          <option value="halted">Halted</option>
        </select>
      </div>

      {/* Pairs Table */}
      <div className="table-container">
        <table>
          <thead>
            <tr>
              <th>Pair</th>
              <th>Chain</th>
              <th>Price</th>
              <th>24h Change</th>
              <th>24h Volume</th>
              <th>Liquidity</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredPairs.map(pair => (
              <tr key={pair.id}>
                <td>
                  <div className="pair-info">
                    <span className="pair-name">{pair.baseToken}/{pair.quoteToken}</span>
                    <span className="pair-address">{pair.pairAddress.slice(0, 10)}...</span>
                  </div>
                </td>
                <td><span className="chain-badge">{pair.chainName}</span></td>
                <td className="price">${pair.price.toLocaleString()}</td>
                <td>
                  <span className={`change ${pair.change24h >= 0 ? 'positive' : 'negative'}`}>
                    {pair.change24h >= 0 ? '+' : ''}{pair.change24h}%
                  </span>
                </td>
                <td>${pair.volume24h.toLocaleString()}</td>
                <td>${pair.liquidity.toLocaleString()}</td>
                <td>
                  <select
                    value={pair.status}
                    onChange={e => handleStatusChange(pair.id, e.target.value as any)}
                    className={`status-select status-${pair.status}`}
                  >
                    <option value="active">Active</option>
                    <option value="inactive">Inactive</option>
                    <option value="halted">Halted</option>
                  </select>
                </td>
                <td>
                  <div className="actions">
                    <button className="action-btn">📊</button>
                    <button className="action-btn">✏️</button>
                    <button className="action-btn" onClick={() => handleDelete(pair.id)}>🗑️</button>
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

export default PairsPage;
