// Blockchain Management Page - Complete Implementation
// Manage blockchain networks, RPC endpoints, and node infrastructure

import React, { useState, useEffect } from 'react';
import './BlockchainPage.css';

interface Blockchain {
  id: string;
  name: string;
  symbol: string;
  chainId: string;
  rpcUrl: string;
  explorerUrl: string;
  type: 'evm' | 'bitcoin' | 'solana' | 'cosmos' | 'other';
  status: 'active' | 'inactive' | 'maintenance';
  gasToken: string;
  avgBlockTime: number;
  transactions: number;
  lastSync: string;
}

const BlockchainPage: React.FC = () => {
  const [blockchains, setBlockchains] = useState<Blockchain[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [editingChain, setEditingChain] = useState<Blockchain | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [filterType, setFilterType] = useState<string>('all');

  useEffect(() => {
    loadBlockchains();
  }, []);

  const loadBlockchains = () => {
    // Mock data - in production, fetch from API
    setBlockchains([
      {
        id: '1',
        name: 'Ethereum',
        symbol: 'ETH',
        chainId: '1',
        rpcUrl: 'https://eth.llamarpc.com',
        explorerUrl: 'https://etherscan.io',
        type: 'evm',
        status: 'active',
        gasToken: 'ETH',
        avgBlockTime: 12,
        transactions: 1523847,
        lastSync: '2 seconds ago',
      },
      {
        id: '2',
        name: 'BNB Smart Chain',
        symbol: 'BNB',
        chainId: '56',
        rpcUrl: 'https://bsc-dataseed.binance.org',
        explorerUrl: 'https://bscscan.com',
        type: 'evm',
        status: 'active',
        gasToken: 'BNB',
        avgBlockTime: 3,
        transactions: 2839471,
        lastSync: '1 seconds ago',
      },
      {
        id: '3',
        name: 'Solana',
        symbol: 'SOL',
        chainId: 'mainnet',
        rpcUrl: 'https://api.mainnet-beta.solana.com',
        explorerUrl: 'https://explorer.solana.com',
        type: 'solana',
        status: 'active',
        gasToken: 'SOL',
        avgBlockTime: 0.4,
        transactions: 894372,
        lastSync: '1 seconds ago',
      },
      {
        id: '4',
        name: 'Polygon',
        symbol: 'MATIC',
        chainId: '137',
        rpcUrl: 'https://polygon-rpc.com',
        explorerUrl: 'https://polygonscan.com',
        type: 'evm',
        status: 'active',
        gasToken: 'MATIC',
        avgBlockTime: 2,
        transactions: 1283746,
        lastSync: '3 seconds ago',
      },
      {
        id: '5',
        name: 'Arbitrum One',
        symbol: 'ETH',
        chainId: '42161',
        rpcUrl: 'https://arb1.arbitrum.io/rpc',
        explorerUrl: 'https://arbiscan.io',
        type: 'evm',
        status: 'active',
        gasToken: 'ETH',
        avgBlockTime: 1,
        transactions: 647293,
        lastSync: '2 seconds ago',
      },
      {
        id: '6',
        name: 'Optimism',
        symbol: 'ETH',
        chainId: '10',
        rpcUrl: 'https://mainnet.optimism.io',
        explorerUrl: 'https://optimistic.etherscan.io',
        type: 'evm',
        status: 'active',
        gasToken: 'ETH',
        avgBlockTime: 2,
        transactions: 483927,
        lastSync: '4 seconds ago',
      },
      {
        id: '7',
        name: 'Avalanche',
        symbol: 'AVAX',
        chainId: '43114',
        rpcUrl: 'https://api.avax.network/ext/bc/C/rpc',
        explorerUrl: 'https://snowtrace.io',
        type: 'evm',
        status: 'active',
        gasToken: 'AVAX',
        avgBlockTime: 2,
        transactions: 392847,
        lastSync: '2 seconds ago',
      },
      {
        id: '8',
        name: 'Base',
        symbol: 'ETH',
        chainId: '8453',
        rpcUrl: 'https://mainnet.base.org',
        explorerUrl: 'https://basescan.org',
        type: 'evm',
        status: 'active',
        gasToken: 'ETH',
        avgBlockTime: 2,
        transactions: 183947,
        lastSync: '1 seconds ago',
      },
    ]);
  };

  const handleStatusChange = async (id: string, newStatus: 'active' | 'inactive' | 'maintenance') => {
    setBlockchains(prev => prev.map(b => b.id === id ? { ...b, status: newStatus } : b));
  };

  const handleDelete = async (id: string) => {
    if (confirm('Are you sure you want to delete this blockchain?')) {
      setBlockchains(prev => prev.filter(b => b.id !== id));
    }
  };

  const handleSave = (chain: Blockchain) => {
    if (editingChain) {
      setBlockchains(prev => prev.map(b => b.id === chain.id ? chain : b));
    } else {
      setBlockchains(prev => [...prev, { ...chain, id: Date.now().toString() }]);
    }
    setShowModal(false);
    setEditingChain(null);
  };

  const filteredChains = blockchains.filter(chain => {
    const matchesSearch = chain.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                       chain.symbol.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesType = filterType === 'all' || chain.type === filterType;
    return matchesSearch && matchesType;
  });

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'status-active';
      case 'inactive': return 'status-inactive';
      case 'maintenance': return 'status-maintenance';
      default: return '';
    }
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'evm': return '⛓️';
      case 'bitcoin': return '₿';
      case 'solana': return '☀️';
      case 'cosmos': return '🌌';
      default: return '🔗';
    }
  };

  return (
    <div className="blockchain-page">
      <div className="page-header">
        <div>
          <h1>Blockchain Management</h1>
          <p>Manage blockchain networks, RPC endpoints, and node infrastructure</p>
        </div>
        <button className="btn btn-primary" onClick={() => { setEditingChain(null); setShowModal(true); }}>
          + Add Blockchain
        </button>
      </div>

      {/* Stats Cards */}
      <div className="stats-grid">
        <div className="stat-card">
          <span className="stat-label">Total Blockchains</span>
          <span className="stat-value">{blockchains.length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Active Networks</span>
          <span className="stat-value">{blockchains.filter(b => b.status === 'active').length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Total Transactions</span>
          <span className="stat-value">{blockchains.reduce((acc, b) => acc + b.transactions, 0).toLocaleString()}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Avg Block Time</span>
          <span className="stat-value">{(blockchains.reduce((acc, b) => acc + b.avgBlockTime, 0) / blockchains.length).toFixed(2)}s</span>
        </div>
      </div>

      {/* Filters */}
      <div className="filters-bar">
        <div className="search-box">
          <span>🔍</span>
          <input
            type="text"
            placeholder="Search blockchains..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
          />
        </div>
        <select value={filterType} onChange={e => setFilterType(e.target.value)}>
          <option value="all">All Types</option>
          <option value="evm">EVM</option>
          <option value="bitcoin">Bitcoin</option>
          <option value="solana">Solana</option>
          <option value="cosmos">Cosmos</option>
        </select>
      </div>

      {/* Blockchain Table */}
      <div className="table-container">
        <table>
          <thead>
            <tr>
              <th>Blockchain</th>
              <th>Chain ID</th>
              <th>Type</th>
              <th>Status</th>
              <th>Gas Token</th>
              <th>Block Time</th>
              <th>Transactions</th>
              <th>Last Sync</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredChains.map(chain => (
              <tr key={chain.id}>
                <td>
                  <div className="chain-info">
                    <span className="chain-icon">{getTypeIcon(chain.type)}</span>
                    <div className="chain-details">
                      <span className="chain-name">{chain.name}</span>
                      <span className="chain-symbol">{chain.symbol}</span>
                    </div>
                  </div>
                </td>
                <td className="mono">{chain.chainId}</td>
                <td><span className="type-badge">{chain.type.toUpperCase()}</span></td>
                <td>
                  <select
                    value={chain.status}
                    onChange={e => handleStatusChange(chain.id, e.target.value as any)}
                    className={`status-select ${getStatusColor(chain.status)}`}
                  >
                    <option value="active">Active</option>
                    <option value="inactive">Inactive</option>
                    <option value="maintenance">Maintenance</option>
                  </select>
                </td>
                <td>{chain.gasToken}</td>
                <td>{chain.avgBlockTime}s</td>
                <td>{chain.transactions.toLocaleString()}</td>
                <td className="muted">{chain.lastSync}</td>
                <td>
                  <div className="actions">
                    <button className="action-btn" onClick={() => { setEditingChain(chain); setShowModal(true); }}>✏️</button>
                    <button className="action-btn" onClick={() => handleDelete(chain.id)}>🗑️</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Add/Edit Modal */}
      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal modal-large" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h2>{editingChain ? 'Edit Blockchain' : 'Add New Blockchain'}</h2>
              <button className="close-btn" onClick={() => setShowModal(false)}>×</button>
            </div>
            <div className="modal-body">
              <div className="form-grid">
                <div className="form-group">
                  <label className="form-label">Blockchain Name</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="Ethereum"
                    defaultValue={editingChain?.name}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Symbol</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="ETH"
                    defaultValue={editingChain?.symbol}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Chain ID</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="1"
                    defaultValue={editingChain?.chainId}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Type</label>
                  <select className="form-input" defaultValue={editingChain?.type || 'evm'}>
                    <option value="evm">EVM</option>
                    <option value="bitcoin">Bitcoin</option>
                    <option value="solana">Solana</option>
                    <option value="cosmos">Cosmos</option>
                    <option value="other">Other</option>
                  </select>
                </div>
                <div className="form-group full-width">
                  <label className="form-label">RPC URL</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="https://rpc.example.com"
                    defaultValue={editingChain?.rpcUrl}
                  />
                </div>
                <div className="form-group full-width">
                  <label className="form-label">Explorer URL</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="https://explorer.example.com"
                    defaultValue={editingChain?.explorerUrl}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Gas Token</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="ETH"
                    defaultValue={editingChain?.gasToken}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Avg Block Time (seconds)</label>
                  <input
                    type="number"
                    className="form-input"
                    placeholder="12"
                    defaultValue={editingChain?.avgBlockTime}
                  />
                </div>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
              <button className="btn btn-primary" onClick={() => {
                const form = document.querySelector('.modal-body');
                const chain: Blockchain = {
                  id: editingChain?.id || Date.now().toString(),
                  name: (form?.querySelectorAll('input')[0] as HTMLInputElement)?.value || '',
                  symbol: (form?.querySelectorAll('input')[1] as HTMLInputElement)?.value || '',
                  chainId: (form?.querySelectorAll('input')[2] as HTMLInputElement)?.value || '',
                  type: (form?.querySelector('select') as HTMLSelectElement)?.value as any || 'evm',
                  rpcUrl: (form?.querySelectorAll('input')[3] as HTMLInputElement)?.value || '',
                  explorerUrl: (form?.querySelectorAll('input')[4] as HTMLInputElement)?.value || '',
                  status: 'active',
                  gasToken: (form?.querySelectorAll('input')[5] as HTMLInputElement)?.value || '',
                  avgBlockTime: parseFloat((form?.querySelectorAll('input')[6] as HTMLInputElement)?.value || '12'),
                  transactions: editingChain?.transactions || 0,
                  lastSync: 'Just now',
                };
                handleSave(chain);
              }}>
                {editingChain ? 'Save Changes' : 'Add Blockchain'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default BlockchainPage;
